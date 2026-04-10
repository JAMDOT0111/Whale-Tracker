package main

import (
	"encoding/csv"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

// ==========================================
// 1. 定義 API 回傳與整合資料結構
// ==========================================

type EtherscanAPIResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type Transaction struct {
	TimeStamp       string `json:"timeStamp"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	IsError         string `json:"isError"`
	ContractAddress string `json:"contractAddress"`
	TokenSymbol     string `json:"tokenSymbol"`
}

type AddressRawData struct {
	Address  string               `json:"address"`
	Label    string               `json:"label"` // 即時推論時預設為 "0"
	NormalTx EtherscanAPIResponse `json:"normal_tx"`
	Internal EtherscanAPIResponse `json:"internal_tx"`
	TokenTx  EtherscanAPIResponse `json:"token_tx"`
	Balance  EtherscanAPIResponse `json:"balance"`
}

type FeatureRow struct {
	Address  string
	Label    string
	Features []string
}

// 將 Wei (字串) 轉換為 Ether (float64)
func parseEther(weiStr string) float64 {
	if weiStr == "" {
		return 0.0
	}
	wei, err := strconv.ParseFloat(weiStr, 64)
	if err != nil {
		return 0.0
	}
	return wei / 1e18
}

// ==========================================
// 2. Etherscan 爬蟲模組
// ==========================================

func fetchAPI(url string) (EtherscanAPIResponse, error) {
	var apiResp EtherscanAPIResponse
	resp, err := http.Get(url)
	if err != nil {
		return apiResp, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return apiResp, err
	}

	err = json.Unmarshal(body, &apiResp)
	return apiResp, err
}

// ==========================================
// 3. 特徵轉換模組 (與你最終版 transformer 邏輯 100% 一致)
// ==========================================

func calculateFeatures(data AddressRawData) FeatureRow {
	address := strings.ToLower(data.Address)

	// 解析 JSON
	var normalTxs, internalTxs, tokenTxs []Transaction
	json.Unmarshal(data.NormalTx.Result, &normalTxs)
	json.Unmarshal(data.Internal.Result, &internalTxs)
	json.Unmarshal(data.TokenTx.Result, &tokenTxs)

	// 解析餘額
	var balanceStr string
	json.Unmarshal(data.Balance.Result, &balanceStr)
	totalEtherBalance := parseEther(balanceStr)

	// --- 變數宣告 ---
	var (
		sentTnx, receivedTnx, createdContracts int
		totalEtherSent, totalEtherReceived     float64

		minValReceived = math.MaxFloat64
		maxValReceived float64
		minValSent     = math.MaxFloat64
		maxValSent     float64

		totalEtherSentContracts float64
		minValSentContract      = math.MaxFloat64
		maxValSentContract      float64

		erc20TotalTnx                               int
		erc20TotalEtherReceived, erc20TotalEtherSent float64
		erc20MinValRec                              = math.MaxFloat64
		erc20MaxValRec                              float64
		erc20MinValSent                             = math.MaxFloat64
		erc20MaxValSent                             float64

		sentTimes, receivedTimes             []int64
		erc20SentTimes, erc20ReceivedTimes   []int64
		firstTxTime, lastTxTime              int64

		uniqueReceivedFrom = make(map[string]bool)
		uniqueSentTo       = make(map[string]bool)
		uniqueErc20RecFrom = make(map[string]bool)
		uniqueErc20SentTo  = make(map[string]bool)

		erc20SentTokenMap = make(map[string]int)
		erc20RecTokenMap  = make(map[string]int)

		zeroValueTxCount, dustValueTxCount, failedTxCount, fakeTokenInteractionCount int
	)

	// --- 處理一般交易 ---
	for _, tx := range normalTxs {
		if tx.IsError == "1" {
			failedTxCount++
			continue
		}

		valEther := parseEther(tx.Value)
		txTime, _ := strconv.ParseInt(tx.TimeStamp, 10, 64)

		if firstTxTime == 0 || txTime < firstTxTime {
			firstTxTime = txTime
		}
		if txTime > lastTxTime {
			lastTxTime = txTime
		}

		if strings.ToLower(tx.From) == address {
			sentTnx++
			totalEtherSent += valEther
			sentTimes = append(sentTimes, txTime)

			if valEther < minValSent { minValSent = valEther }
			if valEther > maxValSent { maxValSent = valEther }

			if tx.To != "" {
				uniqueSentTo[strings.ToLower(tx.To)] = true
			} else {
				createdContracts++
			}

			if valEther == 0 {
				zeroValueTxCount++
			} else if valEther < 1.0 {
				dustValueTxCount++
			}

		} else if strings.ToLower(tx.To) == address {
			receivedTnx++
			totalEtherReceived += valEther
			receivedTimes = append(receivedTimes, txTime)

			if valEther < minValReceived { minValReceived = valEther }
			if valEther > maxValReceived { maxValReceived = valEther }

			uniqueReceivedFrom[strings.ToLower(tx.From)] = true
		}
	}

	// --- 處理內部合約交易 ---
	for _, tx := range internalTxs {
		if tx.IsError == "1" { continue }
		valEther := parseEther(tx.Value)
		if strings.ToLower(tx.From) == address {
			totalEtherSentContracts += valEther
			if valEther < minValSentContract { minValSentContract = valEther }
			if valEther > maxValSentContract { maxValSentContract = valEther }
		}
	}

	// --- 處理 ERC-20 交易 ---
	for _, tx := range tokenTxs {
		erc20TotalTnx++
		valEther := parseEther(tx.Value)
		txTime, _ := strconv.ParseInt(tx.TimeStamp, 10, 64)
		symbol := tx.TokenSymbol

		if strings.ToLower(tx.From) == address {
			erc20TotalEtherSent += valEther
			erc20SentTimes = append(erc20SentTimes, txTime)
			erc20SentTokenMap[symbol]++
			uniqueErc20SentTo[strings.ToLower(tx.To)] = true

			if valEther < erc20MinValSent { erc20MinValSent = valEther }
			if valEther > erc20MaxValSent { erc20MaxValSent = valEther }

			if valEther == 0 {
				zeroValueTxCount++
				fakeTokenInteractionCount++
			}
		} else if strings.ToLower(tx.To) == address {
			erc20TotalEtherReceived += valEther
			erc20ReceivedTimes = append(erc20ReceivedTimes, txTime)
			erc20RecTokenMap[symbol]++
			uniqueErc20RecFrom[strings.ToLower(tx.From)] = true

			if valEther < erc20MinValRec { erc20MinValRec = valEther }
			if valEther > erc20MaxValRec { erc20MaxValRec = valEther }
		}
	}

	// --- 輔助函式與極值校正 ---
	calcAvgMinBetween := func(times []int64) float64 {
		if len(times) < 2 {
			return 0.0
		}
		sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
		var totalDiff int64
		for i := 1; i < len(times); i++ {
			totalDiff += (times[i] - times[i-1])
		}
		return (float64(totalDiff) / float64(len(times)-1)) / 60.0
	}

	findMostFrequentToken := func(m map[string]int) string {
		maxCount, mostFreq := 0, "None"
		for token, count := range m {
			if count > maxCount {
				maxCount, mostFreq = count, token
			}
		}
		return mostFreq
	}

	if minValSent == math.MaxFloat64 { minValSent = 0 }
	if minValReceived == math.MaxFloat64 { minValReceived = 0 }
	if minValSentContract == math.MaxFloat64 { minValSentContract = 0 }
	if erc20MinValSent == math.MaxFloat64 { erc20MinValSent = 0 }
	if erc20MinValRec == math.MaxFloat64 { erc20MinValRec = 0 }

	// --- 填入最終的 51 個 Features 陣列 ---
	features := make([]string, 51)

	features[0] = fmt.Sprintf("%.2f", calcAvgMinBetween(sentTimes))
	features[1] = fmt.Sprintf("%.2f", calcAvgMinBetween(receivedTimes))

	if firstTxTime > 0 && lastTxTime > 0 {
		features[2] = fmt.Sprintf("%.2f", float64(lastTxTime-firstTxTime)/60.0)
	} else {
		features[2] = "0.00"
	}

	features[3] = fmt.Sprintf("%d", sentTnx)
	features[4] = fmt.Sprintf("%d", receivedTnx)
	features[5] = fmt.Sprintf("%d", createdContracts)
	features[6] = fmt.Sprintf("%d", len(uniqueReceivedFrom))
	features[7] = fmt.Sprintf("%d", len(uniqueSentTo))

	features[8] = fmt.Sprintf("%.6f", minValReceived)
	features[9] = fmt.Sprintf("%.6f", maxValReceived)
	if receivedTnx > 0 {
		features[10] = fmt.Sprintf("%.6f", totalEtherReceived/float64(receivedTnx))
	} else {
		features[10] = "0.000000"
	}
	features[11] = fmt.Sprintf("%.6f", minValSent)
	features[12] = fmt.Sprintf("%.6f", maxValSent)
	if sentTnx > 0 {
		features[13] = fmt.Sprintf("%.6f", totalEtherSent/float64(sentTnx))
	} else {
		features[13] = "0.000000"
	}

	features[14] = fmt.Sprintf("%.6f", minValSentContract)
	features[15] = fmt.Sprintf("%.6f", maxValSentContract)
	if len(internalTxs) > 0 {
		features[16] = fmt.Sprintf("%.6f", totalEtherSentContracts/float64(len(internalTxs)))
	} else {
		features[16] = "0.000000"
	}

	features[17] = fmt.Sprintf("%d", len(normalTxs)+len(internalTxs))
	features[18] = fmt.Sprintf("%.6f", totalEtherSent)
	features[19] = fmt.Sprintf("%.6f", totalEtherReceived)
	features[20] = fmt.Sprintf("%.6f", totalEtherSentContracts)
	features[21] = fmt.Sprintf("%.6f", totalEtherBalance)

	features[22] = fmt.Sprintf("%d", erc20TotalTnx)
	features[23] = fmt.Sprintf("%.6f", erc20TotalEtherReceived)
	features[24] = fmt.Sprintf("%.6f", erc20TotalEtherSent)
	features[25] = "0"
	features[26] = fmt.Sprintf("%d", len(uniqueErc20SentTo))
	features[27] = fmt.Sprintf("%d", len(uniqueErc20RecFrom))
	features[28] = "0"
	features[29] = "0"

	features[30] = fmt.Sprintf("%.2f", calcAvgMinBetween(erc20SentTimes))
	features[31] = fmt.Sprintf("%.2f", calcAvgMinBetween(erc20ReceivedTimes))
	features[32] = "0"
	features[33] = "0"

	features[34] = fmt.Sprintf("%.6f", erc20MinValRec)
	features[35] = fmt.Sprintf("%.6f", erc20MaxValRec)
	if len(erc20ReceivedTimes) > 0 {
		features[36] = fmt.Sprintf("%.6f", erc20TotalEtherReceived/float64(len(erc20ReceivedTimes)))
	} else {
		features[36] = "0.000000"
	}
	features[37] = fmt.Sprintf("%.6f", erc20MinValSent)
	features[38] = fmt.Sprintf("%.6f", erc20MaxValSent)
	if len(erc20SentTimes) > 0 {
		features[39] = fmt.Sprintf("%.6f", erc20TotalEtherSent/float64(len(erc20SentTimes)))
	} else {
		features[39] = "0.000000"
	}
	features[40] = "0"
	features[41] = "0"
	features[42] = "0"
	features[43] = fmt.Sprintf("%d", len(erc20SentTokenMap))
	features[44] = fmt.Sprintf("%d", len(erc20RecTokenMap))

	features[45] = findMostFrequentToken(erc20SentTokenMap)
	features[46] = findMostFrequentToken(erc20RecTokenMap)

	totalAllTxs := float64(len(normalTxs) + erc20TotalTnx)
	if totalAllTxs > 0 {
		features[47] = fmt.Sprintf("%.4f", float64(zeroValueTxCount)/totalAllTxs)
		features[48] = fmt.Sprintf("%.4f", float64(dustValueTxCount)/totalAllTxs)
	} else {
		features[47], features[48] = "0.0", "0.0"
	}
	features[49] = fmt.Sprintf("%d", fakeTokenInteractionCount)
	if len(normalTxs) > 0 {
		features[50] = fmt.Sprintf("%.4f", float64(failedTxCount)/float64(len(normalTxs)))
	} else {
		features[50] = "0.0"
	}

	for i := range features {
		if features[i] == "" || features[i] == "NaN" || features[i] == "+Inf" {
			features[i] = "0"
		}
	}

	return FeatureRow{Address: data.Address, Label: data.Label, Features: features}
}

// ==========================================
// 4. 主程式：串接下載與轉換
// ==========================================

func main() {
	address := flag.String("address", "", "請輸入要檢測的以太坊地址")
	apiKey := flag.String("apikey", "", "Etherscan API Key")
	flag.Parse()

	if *address == "" || *apiKey == "" {
		log.Fatal("❌ 請提供參數，範例: go run pipeline.go -address 0x... -apikey YOUR_KEY")
	}

	fmt.Printf("🔍 開始爬取並分析地址: %s\n", *address)

	// API 端點設定 (使用 Etherscan V2 格式)
	urlNormal := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=txlist&address=%s&startblock=0&endblock=99999999&sort=desc&apikey=%s", *address, *apiKey)
	urlInternal := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=txlistinternal&address=%s&startblock=0&endblock=99999999&sort=desc&apikey=%s", *address, *apiKey)
	urlToken := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=tokentx&address=%s&startblock=0&endblock=99999999&sort=desc&apikey=%s", *address, *apiKey)
	urlBalance := fmt.Sprintf("https://api.etherscan.io/v2/api?chainid=1&module=account&action=balance&address=%s&tag=latest&apikey=%s", *address, *apiKey)

	// 抓取 4 種資料 (每次呼叫間隔 200ms 以符合免費版 API 限制)
	fmt.Println("⏳ 正在獲取 Normal Transactions...")
	normalData, err1 := fetchAPI(urlNormal)
	time.Sleep(200 * time.Millisecond)

	fmt.Println("⏳ 正在獲取 Internal Transactions...")
	internalData, err2 := fetchAPI(urlInternal)
	time.Sleep(200 * time.Millisecond)

	fmt.Println("⏳ 正在獲取 ERC20 Transactions...")
	tokenData, err3 := fetchAPI(urlToken)
	time.Sleep(200 * time.Millisecond)

	fmt.Println("⏳ 正在獲取 Balance...")
	balanceData, err4 := fetchAPI(urlBalance)

	if err1 != nil || err2 != nil || err3 != nil || err4 != nil {
		log.Fatalf("❌ 下載資料時發生網路錯誤，請檢查 API Key 或網路連線。")
	}

	// 打包資料
	rawData := AddressRawData{
		Address:  *address,
		Label:    "0", // 推論時預設給 0，模型不管這個
		NormalTx: normalData,
		Internal: internalData,
		TokenTx:  tokenData,
		Balance:  balanceData,
	}

	fmt.Println("⚙️ 正在計算特徵矩陣...")
	featureRow := calculateFeatures(rawData)

	// 匯出 CSV 檔案
	file, err := os.Create("target.csv")
	if err != nil {
		log.Fatalf("❌ 無法建立目標檔案: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	headers := []string{"Address", "Flag",
		"Avg min between sent tnx", "Avg min between received tnx", "Time Diff between first and last (Mins)",
		"Sent tnx", "Received Tnx", "Number of Created Contracts", "Unique Received From Addresses", "Unique Sent To Addresses",
		"min value received", "max value received", "avg val received", "min val sent", "max val sent", "avg val sent",
		"min value sent to contract", "max val sent to contract", "avg value sent to contract",
		"total transactions", "total Ether sent", "total ether received", "total ether sent contracts", "total ether balance",
		"Total ERC20 tnxs", "ERC20 total Ether received", "ERC20 total ether sent", "ERC20 total Ether sent contract",
		"ERC20 uniq sent addr", "ERC20 uniq rec addr", "ERC20 uniq sent addr.1", "ERC20 uniq rec contract addr",
		"ERC20 avg time between sent tnx", "ERC20 avg time between rec tnx", "ERC20 avg time between rec 2 tnx", "ERC20 avg time between contract tnx",
		"ERC20 min val rec", "ERC20 max val rec", "ERC20 avg val rec", "ERC20 min val sent", "ERC20 max val sent", "ERC20 avg val sent",
		"ERC20 min val sent contract", "ERC20 max val sent contract", "ERC20 avg val sent contract",
		"ERC20 uniq sent token name", "ERC20 uniq rec token name", "ERC20 most sent token type", "ERC20_most_rec_token_type",
		"Zero_Value_Tx_Ratio", "Dust_Value_Tx_Ratio", "Fake_Token_Interaction_Count", "Failed_Tx_Ratio",
	}

	writer.Write(headers)
	
	// 組合 CSV 行：Address + Label + Features
	csvRow := append([]string{featureRow.Address, featureRow.Label}, featureRow.Features...)
	writer.Write(csvRow)

	fmt.Println("🎉 特徵萃取成功！已匯出至 target.csv，請執行 python predict.py 進行判讀。")
}