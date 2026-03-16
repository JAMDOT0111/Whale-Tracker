package service

import (
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"log"
	"math"
	"math/big"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

type EtherscanClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
	cache      sync.Map
	rateMu     sync.Mutex
	callTimes  []time.Time
}

func NewEtherscanClient() *EtherscanClient {
	apiKey := os.Getenv("ETHERSCAN_API_KEY")
	if apiKey == "" {
		panic("ETHERSCAN_API_KEY environment variable is required")
	}

	return &EtherscanClient{
		apiKey:  apiKey,
		baseURL: "https://api.etherscan.io/v2/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (c *EtherscanClient) GetTransactions(address string, pageKey string, pageSize int) (*model.ScanResponse, error) {
	if pageSize <= 0 || pageSize > 10000 {
		pageSize = 100
	}

	page := 1
	if pageKey != "" {
		if p, err := strconv.Atoi(pageKey); err == nil && p > 0 {
			page = p
		}
	}

	var allTx []model.Transaction

	normalTx, err := c.fetchNormalTx(address, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("normal tx: %w", err)
	}
	allTx = append(allTx, normalTx...)

	internalTx, err := c.fetchInternalTx(address, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("internal tx: %w", err)
	}
	allTx = append(allTx, internalTx...)

	tokenTx, err := c.fetchTokenTx(address, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("erc20 tx: %w", err)
	}
	allTx = append(allTx, tokenTx...)

	nftTx, err := c.fetchNFTTx(address, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("erc721 tx: %w", err)
	}
	allTx = append(allTx, nftTx...)

	nextPage := ""
	if len(normalTx) == pageSize || len(internalTx) == pageSize || len(tokenTx) == pageSize || len(nftTx) == pageSize {
		nextPage = strconv.Itoa(page + 1)
	}

	return &model.ScanResponse{
		Transactions: allTx,
		PageKey:      nextPage,
		Total:        len(allTx),
	}, nil
}

func (c *EtherscanClient) GetAllTransactionsForGraph(address string) ([]model.Transaction, error) {
	if cached, ok := c.cache.Load(strings.ToLower(address)); ok {
		return cached.([]model.Transaction), nil
	}

	var allTx []model.Transaction
	graphPageSize := 200

	for _, fetcher := range []func(string, int, int) ([]model.Transaction, error){
		c.fetchNormalTx,
		c.fetchInternalTx,
		c.fetchTokenTx,
		c.fetchNFTTx,
	} {
		txs, err := fetcher(address, 1, graphPageSize)
		if err != nil {
			continue
		}
		allTx = append(allTx, txs...)
	}

	c.cache.Store(strings.ToLower(address), allTx)
	return allTx, nil
}

func (c *EtherscanClient) GetBalance(address string) (*model.BalanceResponse, error) {
	resp, err := c.callAPI(map[string]string{
		"module":  "account",
		"action":  "balance",
		"address": address,
		"tag":     "latest",
	})
	if err != nil {
		return nil, fmt.Errorf("eth balance: %w", err)
	}

	var balWei string
	json.Unmarshal(resp.Result, &balWei)
	ethBalance := weiToEth(balWei)

	tokenResp, err := c.callAPI(map[string]string{
		"module":  "account",
		"action":  "tokenlist",
		"address": address,
	})

	var tokens []model.TokenBalance
	if err == nil {
		var tokenInfos []model.EtherscanTokenInfo
		if json.Unmarshal(tokenResp.Result, &tokenInfos) == nil {
			for _, t := range tokenInfos {
				if t.Balance == "" || t.Balance == "0" {
					continue
				}
				tokens = append(tokens, model.TokenBalance{
					Symbol:  t.TokenSymbol,
					Name:    t.TokenName,
					Balance: tokenValueToDecimal(t.Balance, t.TokenDecimal),
				})
			}
		}
	}

	return &model.BalanceResponse{
		EthBalance: ethBalance,
		Tokens:     tokens,
	}, nil
}

func (c *EtherscanClient) IsContract(address string) bool {
	cacheKey := "contract:" + strings.ToLower(address)
	if cached, ok := c.cache.Load(cacheKey); ok {
		return cached.(bool)
	}

	resp, err := c.callAPI(map[string]string{
		"module":  "proxy",
		"action":  "eth_getCode",
		"address": address,
	})
	if err != nil {
		return false
	}

	var code string
	json.Unmarshal(resp.Result, &code)
	isContract := code != "" && code != "0x"

	c.cache.Store(cacheKey, isContract)
	return isContract
}

const maxCallsPerWindow = 3
const windowDuration = 1100 * time.Millisecond // 1.1s window for safety margin
const maxRetries = 3

func (c *EtherscanClient) rateLimit() {
	c.rateMu.Lock()
	defer c.rateMu.Unlock()

	now := time.Now()

	// Remove timestamps older than the window
	valid := c.callTimes[:0]
	for _, t := range c.callTimes {
		if now.Sub(t) < windowDuration {
			valid = append(valid, t)
		}
	}
	c.callTimes = valid

	// If we've hit the limit, wait until the oldest call exits the window
	if len(c.callTimes) >= maxCallsPerWindow {
		waitUntil := c.callTimes[0].Add(windowDuration)
		if sleepDur := time.Until(waitUntil); sleepDur > 0 {
			time.Sleep(sleepDur)
		}
		c.callTimes = c.callTimes[1:]
	}

	c.callTimes = append(c.callTimes, time.Now())
}

func (c *EtherscanClient) callAPI(params map[string]string) (*model.EtherscanResponse, error) {
	var lastErr error

	for attempt := range maxRetries {
		c.rateLimit()

		req, err := http.NewRequest("GET", c.baseURL, nil)
		if err != nil {
			return nil, fmt.Errorf("create request: %w", err)
		}

		q := req.URL.Query()
		q.Set("chainid", "1")
		q.Set("apikey", c.apiKey)
		for k, v := range params {
			q.Set(k, v)
		}
		req.URL.RawQuery = q.Encode()

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("call etherscan: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, fmt.Errorf("etherscan returned status %d", resp.StatusCode)
		}

		var result model.EtherscanResponse
		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			return nil, fmt.Errorf("decode response: %w", err)
		}

		if result.Status == "0" && result.Message != "No transactions found" {
			var errMsg string
			json.Unmarshal(result.Result, &errMsg)

			if strings.Contains(errMsg, "rate limit") {
				lastErr = fmt.Errorf("rate limited (attempt %d/%d)", attempt+1, maxRetries)
				log.Printf("[etherscan] %v, retrying in 1.5s...", lastErr)
				time.Sleep(1500 * time.Millisecond)
				continue
			}

			if errMsg != "" {
				return nil, fmt.Errorf("etherscan error: %s", errMsg)
			}
		}

		return &result, nil
	}

	return nil, fmt.Errorf("etherscan: max retries exceeded: %w", lastErr)
}

func (c *EtherscanClient) fetchNormalTx(address string, page, offset int) ([]model.Transaction, error) {
	resp, err := c.callAPI(map[string]string{
		"module":     "account",
		"action":     "txlist",
		"address":    address,
		"startblock": "0",
		"endblock":   "99999999",
		"page":       strconv.Itoa(page),
		"offset":     strconv.Itoa(offset),
		"sort":       "desc",
	})
	if err != nil {
		return nil, err
	}

	var txs []model.EtherscanNormalTx
	if err := json.Unmarshal(resp.Result, &txs); err != nil {
		return nil, nil
	}

	result := make([]model.Transaction, 0, len(txs))
	for _, tx := range txs {
		if tx.IsError == "1" {
			continue
		}
		result = append(result, model.Transaction{
			Hash:        tx.Hash,
			From:        strings.ToLower(tx.From),
			To:          strings.ToLower(tx.To),
			Value:       weiToEth(tx.Value),
			Asset:       "ETH",
			Category:    "external",
			BlockNumber: tx.BlockNumber,
			Timestamp:   unixToISO(tx.TimeStamp),
		})
	}
	return result, nil
}

func (c *EtherscanClient) fetchInternalTx(address string, page, offset int) ([]model.Transaction, error) {
	resp, err := c.callAPI(map[string]string{
		"module":     "account",
		"action":     "txlistinternal",
		"address":    address,
		"startblock": "0",
		"endblock":   "99999999",
		"page":       strconv.Itoa(page),
		"offset":     strconv.Itoa(offset),
		"sort":       "desc",
	})
	if err != nil {
		return nil, err
	}

	var txs []model.EtherscanInternalTx
	if err := json.Unmarshal(resp.Result, &txs); err != nil {
		return nil, nil
	}

	result := make([]model.Transaction, 0, len(txs))
	for _, tx := range txs {
		if tx.IsError == "1" {
			continue
		}
		result = append(result, model.Transaction{
			Hash:        tx.Hash,
			From:        strings.ToLower(tx.From),
			To:          strings.ToLower(tx.To),
			Value:       weiToEth(tx.Value),
			Asset:       "ETH",
			Category:    "internal",
			BlockNumber: tx.BlockNumber,
			Timestamp:   unixToISO(tx.TimeStamp),
		})
	}
	return result, nil
}

func (c *EtherscanClient) fetchTokenTx(address string, page, offset int) ([]model.Transaction, error) {
	resp, err := c.callAPI(map[string]string{
		"module":     "account",
		"action":     "tokentx",
		"address":    address,
		"startblock": "0",
		"endblock":   "99999999",
		"page":       strconv.Itoa(page),
		"offset":     strconv.Itoa(offset),
		"sort":       "desc",
	})
	if err != nil {
		return nil, err
	}

	var txs []model.EtherscanTokenTx
	if err := json.Unmarshal(resp.Result, &txs); err != nil {
		return nil, nil
	}

	result := make([]model.Transaction, 0, len(txs))
	for _, tx := range txs {
		result = append(result, model.Transaction{
			Hash:        tx.Hash,
			From:        strings.ToLower(tx.From),
			To:          strings.ToLower(tx.To),
			Value:       tokenValueToDecimal(tx.Value, tx.TokenDecimal),
			Asset:       tx.TokenSymbol,
			Category:    "erc20",
			BlockNumber: tx.BlockNumber,
			Timestamp:   unixToISO(tx.TimeStamp),
		})
	}
	return result, nil
}

func (c *EtherscanClient) fetchNFTTx(address string, page, offset int) ([]model.Transaction, error) {
	resp, err := c.callAPI(map[string]string{
		"module":     "account",
		"action":     "tokennfttx",
		"address":    address,
		"startblock": "0",
		"endblock":   "99999999",
		"page":       strconv.Itoa(page),
		"offset":     strconv.Itoa(offset),
		"sort":       "desc",
	})
	if err != nil {
		return nil, err
	}

	var txs []model.EtherscanNFTTx
	if err := json.Unmarshal(resp.Result, &txs); err != nil {
		return nil, nil
	}

	result := make([]model.Transaction, 0, len(txs))
	for _, tx := range txs {
		result = append(result, model.Transaction{
			Hash:        tx.Hash,
			From:        strings.ToLower(tx.From),
			To:          strings.ToLower(tx.To),
			Value:       "TokenID:" + tx.TokenID,
			Asset:       tx.TokenSymbol,
			Category:    "erc721",
			BlockNumber: tx.BlockNumber,
			Timestamp:   unixToISO(tx.TimeStamp),
		})
	}
	return result, nil
}

func weiToEth(weiStr string) string {
	if weiStr == "" || weiStr == "0" {
		return "0"
	}
	wei := new(big.Float)
	wei.SetString(weiStr)
	divisor := new(big.Float).SetFloat64(1e18)
	eth := new(big.Float).Quo(wei, divisor)
	return strings.TrimRight(strings.TrimRight(eth.Text('f', 18), "0"), ".")
}

func tokenValueToDecimal(valueStr, decimalStr string) string {
	if valueStr == "" || valueStr == "0" {
		return "0"
	}
	decimals, err := strconv.Atoi(decimalStr)
	if err != nil || decimals == 0 {
		return valueStr
	}
	val := new(big.Float)
	val.SetString(valueStr)
	divisor := new(big.Float).SetFloat64(math.Pow(10, float64(decimals)))
	result := new(big.Float).Quo(val, divisor)
	return strings.TrimRight(strings.TrimRight(result.Text('f', decimals), "0"), ".")
}

func unixToISO(timestamp string) string {
	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return timestamp
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}
