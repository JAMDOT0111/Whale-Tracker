package service

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"eth-sweeper/model"
	"fmt"
	"math/big"
	"strconv"
	"strings"
	"sync"
)

const (
	approvalEventTopic   = "0x8c5be1e5ebec7d5bd14f71427d1e84f3dd0314c0f7b2291e5b200ac8c7c3b925"
	defaultApprovalLimit = 20
	maxApprovalLimit     = 50
)

var approvalLookbackBlocks = []int64{250_000, 1_000_000, 5_000_000, 0}

var unlimitedApprovalValue = func() *big.Int {
	value := new(big.Int)
	value.Exp(big.NewInt(2), big.NewInt(256), nil)
	value.Sub(value, big.NewInt(1))
	return value
}()

type tokenMetadata struct {
	Name     string
	Symbol   string
	Decimals int
}

type ApprovalService struct {
	etherscan *EtherscanClient
	metaCache sync.Map
}

func NewApprovalService(etherscan *EtherscanClient) *ApprovalService {
	return &ApprovalService{etherscan: etherscan}
}

func (s *ApprovalService) GetTokenApprovals(_ context.Context, address string, limit int) (model.TokenApprovalResponse, error) {
	if s.etherscan == nil || s.etherscan.apiKey == "" {
		return model.TokenApprovalResponse{}, fmt.Errorf("ETHERSCAN_API_KEY is required for token approval scans")
	}
	if limit <= 0 {
		limit = defaultApprovalLimit
	}
	if limit > maxApprovalLimit {
		limit = maxApprovalLimit
	}

	logs, err := s.fetchApprovalLogs(address, limit)
	if err != nil {
		return model.TokenApprovalResponse{}, err
	}

	items := make([]model.TokenApprovalItem, 0, len(logs))
	for _, entry := range logs {
		item, ok := s.logToApprovalItem(entry)
		if !ok {
			continue
		}
		items = append(items, item)
	}

	return model.TokenApprovalResponse{
		Address:        address,
		Items:          items,
		ScannedAt:      nowISO(),
		Source:         "etherscan_approval_logs",
		CandidateOnly:  true,
		LimitApplied:   limit,
		ReviewRequired: true,
	}, nil
}

func (s *ApprovalService) fetchApprovalLogs(address string, limit int) ([]model.EtherscanLog, error) {
	latestBlock, err := s.latestBlockNumber()
	if err != nil {
		return s.fetchApprovalLogsPage(address, limit, "0")
	}

	var lastErr error
	for _, lookback := range approvalLookbackBlocks {
		fromBlock := "0"
		if lookback > 0 && latestBlock > lookback {
			fromBlock = strconv.FormatInt(latestBlock-lookback, 10)
		}

		logs, fetchErr := s.fetchApprovalLogsPage(address, limit, fromBlock)
		if fetchErr == nil {
			if len(logs) > 0 || lookback == 0 {
				return logs, nil
			}
			continue
		}
		lastErr = fetchErr
	}

	if lastErr != nil {
		return nil, lastErr
	}
	return []model.EtherscanLog{}, nil
}

func (s *ApprovalService) fetchApprovalLogsPage(address string, limit int, fromBlock string) ([]model.EtherscanLog, error) {
	resp, err := s.etherscan.callAPI(map[string]string{
		"module":       "logs",
		"action":       "getLogs",
		"fromBlock":    fromBlock,
		"toBlock":      "latest",
		"page":         "1",
		"offset":       strconv.Itoa(limit),
		"sort":         "desc",
		"topic0":       approvalEventTopic,
		"topic0_1_opr": "and",
		"topic1":       topicAddress(address),
	})
	if err != nil {
		return nil, err
	}

	var logs []model.EtherscanLog
	if err := json.Unmarshal(resp.Result, &logs); err != nil {
		var noRecords string
		if json.Unmarshal(resp.Result, &noRecords) == nil && strings.Contains(strings.ToLower(noRecords), "no records") {
			return []model.EtherscanLog{}, nil
		}
		return nil, fmt.Errorf("decode approval logs: %w", err)
	}
	return logs, nil
}

func (s *ApprovalService) latestBlockNumber() (int64, error) {
	resp, err := s.etherscan.callAPI(map[string]string{
		"module": "proxy",
		"action": "eth_blockNumber",
	})
	if err != nil {
		return 0, err
	}

	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return 0, err
	}
	value := hexToBigInt(result)
	if value == nil {
		return 0, fmt.Errorf("invalid latest block response")
	}
	return value.Int64(), nil
}

func (s *ApprovalService) logToApprovalItem(entry model.EtherscanLog) (model.TokenApprovalItem, bool) {
	if len(entry.Topics) < 3 {
		return model.TokenApprovalItem{}, false
	}

	tokenAddress := strings.ToLower(strings.TrimSpace(entry.Address))
	spender := topicHexToAddress(entry.Topics[2])
	if !IsValidEthAddress(tokenAddress) || !IsValidEthAddress(spender) {
		return model.TokenApprovalItem{}, false
	}

	value := hexToBigInt(entry.Data)
	if value == nil {
		value = big.NewInt(0)
	}

	meta := s.tokenMetadata(tokenAddress)
	display := value.String()
	if meta.Decimals > 0 {
		display = formatTokenAmount(value, meta.Decimals)
	}

	spenderLabel := ""
	riskLevel := "low"
	if label := LookupAddress(spender); label != nil {
		spenderLabel = label.Name
		switch normalizeLabelCategory(label.Tag) {
		case "bridge", "defi_protocol":
			riskLevel = "medium"
		case "exchange":
			riskLevel = "low"
		}
	} else if s.etherscan.IsContract(spender) {
		riskLevel = "medium"
	}
	if value.Cmp(unlimitedApprovalValue) == 0 {
		riskLevel = "high"
	}

	approvalType := "limited"
	if value.Cmp(unlimitedApprovalValue) == 0 {
		approvalType = "unlimited"
	}

	return model.TokenApprovalItem{
		TokenAddress:    tokenAddress,
		TokenName:       meta.Name,
		TokenSymbol:     meta.Symbol,
		TokenDecimals:   meta.Decimals,
		Spender:         spender,
		SpenderLabel:    spenderLabel,
		ApprovalValue:   value.String(),
		ApprovalDisplay: display,
		ApprovalType:    approvalType,
		RiskLevel:       riskLevel,
		TxHash:          strings.ToLower(strings.TrimSpace(entry.TransactionHash)),
		Timestamp:       unixToISOFromHex(entry.TimeStamp),
	}, true
}

func (s *ApprovalService) tokenMetadata(tokenAddress string) tokenMetadata {
	cacheKey := strings.ToLower(tokenAddress)
	if cached, ok := s.metaCache.Load(cacheKey); ok {
		return cached.(tokenMetadata)
	}

	meta := tokenMetadata{
		Name:     "",
		Symbol:   "",
		Decimals: 0,
	}

	if name, err := s.ethCallString(tokenAddress, "0x06fdde03"); err == nil {
		meta.Name = name
	}
	if symbol, err := s.ethCallString(tokenAddress, "0x95d89b41"); err == nil {
		meta.Symbol = symbol
	}
	if decimals, err := s.ethCallUint8(tokenAddress, "0x313ce567"); err == nil {
		meta.Decimals = decimals
	}

	s.metaCache.Store(cacheKey, meta)
	return meta
}

func (s *ApprovalService) ethCallString(contractAddress, selector string) (string, error) {
	raw, err := s.ethCall(contractAddress, selector)
	if err != nil {
		return "", err
	}
	if raw == "" || raw == "0x" {
		return "", fmt.Errorf("empty response")
	}

	trimmed := strings.TrimPrefix(raw, "0x")
	if len(trimmed) == 64 {
		decoded, err := hex.DecodeString(trimmed)
		if err != nil {
			return "", err
		}
		return strings.TrimRight(string(decoded), "\x00"), nil
	}
	if len(trimmed) < 128 {
		return "", fmt.Errorf("unexpected string response")
	}

	lengthWord := trimmed[64:128]
	lengthValue, ok := new(big.Int).SetString(lengthWord, 16)
	if !ok {
		return "", fmt.Errorf("invalid string length")
	}
	length := int(lengthValue.Int64())
	if length <= 0 {
		return "", fmt.Errorf("empty string length")
	}
	start := 128
	end := start + length*2
	if len(trimmed) < end {
		return "", fmt.Errorf("string response shorter than expected")
	}
	decoded, err := hex.DecodeString(trimmed[start:end])
	if err != nil {
		return "", err
	}
	return string(decoded), nil
}

func (s *ApprovalService) ethCallUint8(contractAddress, selector string) (int, error) {
	raw, err := s.ethCall(contractAddress, selector)
	if err != nil {
		return 0, err
	}
	value := hexToBigInt(raw)
	if value == nil {
		return 0, fmt.Errorf("invalid uint8 response")
	}
	return int(value.Int64()), nil
}

func (s *ApprovalService) ethCall(contractAddress, data string) (string, error) {
	resp, err := s.etherscan.callAPI(map[string]string{
		"module": "proxy",
		"action": "eth_call",
		"to":     contractAddress,
		"data":   data,
		"tag":    "latest",
	})
	if err != nil {
		return "", err
	}

	var result string
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return "", err
	}
	return result, nil
}

func topicAddress(address string) string {
	return "0x000000000000000000000000" + strings.TrimPrefix(strings.ToLower(address), "0x")
}

func topicHexToAddress(topic string) string {
	cleaned := strings.TrimPrefix(strings.ToLower(topic), "0x")
	if len(cleaned) < 40 {
		return ""
	}
	return "0x" + cleaned[len(cleaned)-40:]
}

func hexToBigInt(value string) *big.Int {
	cleaned := strings.TrimPrefix(strings.TrimSpace(value), "0x")
	if cleaned == "" {
		return nil
	}
	result, ok := new(big.Int).SetString(cleaned, 16)
	if !ok {
		return nil
	}
	return result
}

func unixToISOFromHex(value string) string {
	if parsed := hexToBigInt(value); parsed != nil {
		return unixToISO(parsed.String())
	}
	return value
}

func formatTokenAmount(value *big.Int, decimals int) string {
	if value == nil {
		return "0"
	}
	if decimals <= 0 {
		return value.String()
	}
	numerator := new(big.Float).SetInt(value)
	denominator := new(big.Float).SetInt(new(big.Int).Exp(big.NewInt(10), big.NewInt(int64(decimals)), nil))
	result := new(big.Float).Quo(numerator, denominator)
	return strings.TrimRight(strings.TrimRight(result.Text('f', min(decimals, 8)), "0"), ".")
}
