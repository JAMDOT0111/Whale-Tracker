package service

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"eth-sweeper/model"
)
type FraudPrediction struct {
IsFraud      bool    `json:"is_fraud"`
Confidence   float64 `json:"confidence"`
ProbPhishing float64 `json:"prob_phishing"`
ProbNormal   float64 `json:"prob_normal"`
}

func PredictAddressFraud(address string, normalTxs, internalTxs, tokenTxs []model.Transaction, balanceStr string) (*FraudPrediction, error) {
address = strings.ToLower(address)

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

parseEther := func(valStr string) float64 {
		if valStr == "" {
			return 0
		}
		// model.Transaction Value is already converted to Ether in etherscan.go
		if strings.HasPrefix(valStr, "TokenID:") {
			return 0 // handle erc721 if mixed in
		}
		val, _ := strconv.ParseFloat(valStr, 64)
		return val
	}

	parseUnix := func(isoTime string) int64 {
		t, err := time.Parse(time.RFC3339, isoTime)
		if err != nil {
			return 0
		}
		return t.Unix()
	}

	totalEtherBalance := parseEther(balanceStr)

	for _, tx := range normalTxs {
		if tx.Category == "failed" || tx.Category == "error" || tx.Asset == "" {
			failedTxCount++
			// Note: The Etherscan API mapping in etherscan.go currently drops isError="1" entirely.
			// So this slice will likely be clean and failedTxCount may remain 0. 
			// We keep logic in case etherscan.go includes failed txs in the future.
		}

		valEther := parseEther(tx.Value)
		txTime := parseUnix(tx.Timestamp)
if firstTxTime == 0 || txTime < firstTxTime { firstTxTime = txTime }
if txTime > lastTxTime { lastTxTime = txTime }

if strings.ToLower(tx.From) == address {
sentTnx++
totalEtherSent += valEther
sentTimes = append(sentTimes, txTime)

if valEther < minValSent { minValSent = valEther }
if valEther > maxValSent { maxValSent = valEther }
if tx.To != "" { uniqueSentTo[strings.ToLower(tx.To)] = true } else { createdContracts++ }
if valEther == 0 { zeroValueTxCount++ } else if valEther < 1.0 { dustValueTxCount++ }
} else if strings.ToLower(tx.To) == address {
receivedTnx++
totalEtherReceived += valEther
receivedTimes = append(receivedTimes, txTime)
if valEther < minValReceived { minValReceived = valEther }
if valEther > maxValReceived { maxValReceived = valEther }
uniqueReceivedFrom[strings.ToLower(tx.From)] = true
}
}

for _, tx := range internalTxs {
if tx.Category == "failed" { continue }
valEther := parseEther(tx.Value)
if strings.ToLower(tx.From) == address {
totalEtherSentContracts += valEther
if valEther < minValSentContract { minValSentContract = valEther }
if valEther > maxValSentContract { maxValSentContract = valEther }
}
}

for _, tx := range tokenTxs {
		erc20TotalTnx++
		valEther := parseEther(tx.Value)
		txTime := parseUnix(tx.Timestamp)
		symbol := tx.Asset

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

calcAvgMinBetween := func(times []int64) float64 {
if len(times) < 2 { return 0.0 }
sort.Slice(times, func(i, j int) bool { return times[i] < times[j] })
var totalDiff int64
for i := 1; i < len(times); i++ { totalDiff += (times[i] - times[i-1]) }
return (float64(totalDiff) / float64(len(times)-1)) / 60.0
}

findMostFreq := func(m map[string]int) string {
maxCount, mostFreq := 0, "None"
for t, c := range m {
if c > maxCount { maxCount, mostFreq = c, t }
}
return mostFreq
}

if minValSent == math.MaxFloat64 { minValSent = 0 }
if minValReceived == math.MaxFloat64 { minValReceived = 0 }
if minValSentContract == math.MaxFloat64 { minValSentContract = 0 }
if erc20MinValSent == math.MaxFloat64 { erc20MinValSent = 0 }
if erc20MinValRec == math.MaxFloat64 { erc20MinValRec = 0 }

features := make([]string, 51)
features[0] = fmt.Sprintf("%.2f", calcAvgMinBetween(sentTimes))
features[1] = fmt.Sprintf("%.2f", calcAvgMinBetween(receivedTimes))
if firstTxTime > 0 && lastTxTime > 0 { features[2] = fmt.Sprintf("%.2f", float64(lastTxTime-firstTxTime)/60.0) } else { features[2] = "0.00" }
features[3] = fmt.Sprintf("%d", sentTnx)
features[4] = fmt.Sprintf("%d", receivedTnx)
features[5] = fmt.Sprintf("%d", createdContracts)
features[6] = fmt.Sprintf("%d", len(uniqueReceivedFrom))
features[7] = fmt.Sprintf("%d", len(uniqueSentTo))
features[8] = fmt.Sprintf("%.6f", minValReceived)
features[9] = fmt.Sprintf("%.6f", maxValReceived)
if receivedTnx > 0 { features[10] = fmt.Sprintf("%.6f", totalEtherReceived/float64(receivedTnx)) } else { features[10] = "0.000000" }
features[11] = fmt.Sprintf("%.6f", minValSent)
features[12] = fmt.Sprintf("%.6f", maxValSent)
if sentTnx > 0 { features[13] = fmt.Sprintf("%.6f", totalEtherSent/float64(sentTnx)) } else { features[13] = "0.000000" }
features[14] = fmt.Sprintf("%.6f", minValSentContract)
features[15] = fmt.Sprintf("%.6f", maxValSentContract)
if len(internalTxs) > 0 { features[16] = fmt.Sprintf("%.6f", totalEtherSentContracts/float64(len(internalTxs))) } else { features[16] = "0.000000" }

features[17] = fmt.Sprintf("%d", len(normalTxs)+len(internalTxs))
features[18] = fmt.Sprintf("%.6f", totalEtherSent)
features[19] = fmt.Sprintf("%.6f", totalEtherReceived)
features[20] = fmt.Sprintf("%.6f", totalEtherSentContracts) 
features[21] = fmt.Sprintf("%.6f", totalEtherBalance)

features[22] = fmt.Sprintf("%d", erc20TotalTnx)
features[23] = fmt.Sprintf("%.6f", erc20TotalEtherReceived)
features[24] = fmt.Sprintf("%.6f", erc20TotalEtherSent)
features[25] = "0.000000" // ERC20 total Ether sent contract
features[26] = fmt.Sprintf("%d", len(uniqueErc20SentTo))
features[27] = fmt.Sprintf("%d", len(uniqueErc20RecFrom))
features[28] = "0" // uniq sent addr.1
features[29] = "0" // rec contract addr
features[30] = fmt.Sprintf("%.2f", calcAvgMinBetween(erc20SentTimes))
features[31] = fmt.Sprintf("%.2f", calcAvgMinBetween(erc20ReceivedTimes))
features[32] = "0.00" // rec 2 tnx
features[33] = "0.00" // contract tnx
features[34] = fmt.Sprintf("%.6f", erc20MinValRec)
features[35] = fmt.Sprintf("%.6f", erc20MaxValRec)
if len(erc20ReceivedTimes) > 0 { features[36] = fmt.Sprintf("%.6f", erc20TotalEtherReceived/float64(len(erc20ReceivedTimes))) } else { features[36] = "0.000000" }
features[37] = fmt.Sprintf("%.6f", erc20MinValSent)
features[38] = fmt.Sprintf("%.6f", erc20MaxValSent)
if len(erc20SentTimes) > 0 { features[39] = fmt.Sprintf("%.6f", erc20TotalEtherSent/float64(len(erc20SentTimes))) } else { features[39] = "0.000000" }
features[40] = "0.000000" // min val contract
features[41] = "0.000000" // max val contract
features[42] = "0.000000" // avg val contract
features[43] = fmt.Sprintf("%d", len(erc20SentTokenMap))
features[44] = fmt.Sprintf("%d", len(erc20RecTokenMap))
features[45] = findMostFreq(erc20SentTokenMap)
features[46] = findMostFreq(erc20RecTokenMap)

totalAllTxs := float64(len(normalTxs) + len(tokenTxs))
if totalAllTxs > 0 {
	features[47] = fmt.Sprintf("%.6f", float64(zeroValueTxCount)/totalAllTxs)
	features[48] = fmt.Sprintf("%.6f", float64(dustValueTxCount)/totalAllTxs)
} else {
	features[47], features[48] = "0.0", "0.0"
}
features[49] = fmt.Sprintf("%d", fakeTokenInteractionCount)
if len(normalTxs) > 0 {
	features[50] = fmt.Sprintf("%.6f", float64(failedTxCount)/float64(len(normalTxs)))
} else {
	features[50] = "0.0"
}

header := []string{
"Address", "Flag", "Avg min between sent tnx", "Avg min between received tnx",
"Time Diff between first and last (Mins)", "Sent tnx", "Received Tnx", "Number of Created Contracts",
"Unique Received From Addresses", "Unique Sent To Addresses", "min value received",
"max value received", "avg val received", "min val sent", "max val sent", "avg val sent",
"min value sent to contract", "max val sent to contract", "avg value sent to contract",
"total transactions (including tnx to create contract)", "total Ether sent", "total ether received",
"total ether sent contracts", "total ether balance", " Total ERC20 tnxs", " ERC20 total Ether received",
" ERC20 total ether sent", " ERC20 total Ether sent contract", " ERC20 uniq sent addr",
" ERC20 uniq rec addr", " ERC20 uniq sent addr.1", " ERC20 uniq rec contract addr",
" ERC20 avg time between sent tnx", " ERC20 avg time between rec tnx", " ERC20 avg time between rec 2 tnx",
" ERC20 avg time between contract tnx", " ERC20 min val rec", " ERC20 max val rec",
" ERC20 avg val rec", " ERC20 min val sent", " ERC20 max val sent", " ERC20 avg val sent",
" ERC20 min val sent contract", " ERC20 max val sent contract", " ERC20 avg val sent contract",
" ERC20 uniq sent token name", " ERC20 uniq rec token name", " ERC20 most sent token type",
" ERC20_most_rec_token_type", "Zero_Value_Tx_Ratio", "Dust_Value_Tx_Ratio", "Fake_Token_Interaction_Count", "Failed_Tx_Ratio",
}

if len(features) < len(header)-2 {
for len(features) < len(header)-2 { features = append(features, "0") }
}

for i := range features {
	if features[i] == "" || features[i] == "NaN" || features[i] == "+Inf" {
		features[i] = "0"
	}
}

row := append([]string{address, "0"}, features...)

tmpDir := os.TempDir()
csvPath := filepath.Join(tmpDir, fmt.Sprintf("target_%s.csv", address))
f, err := os.Create(csvPath)
if err != nil { return nil, err }
w := csv.NewWriter(f)
w.Write(header)
w.Write(row[:len(header)])
w.Flush()
f.Close()
defer os.Remove(csvPath)

	cwd, _ := os.Getwd()
	mlDir := filepath.Join(cwd, "ml") // assuming we are in backend dir
	predictPy := filepath.Join(mlDir, "predict.py")
	modelJson := filepath.Join(mlDir, "eth_fraud_detection_model.json")

	var cmd *exec.Cmd

	// 檢查是否有編譯好的執行檔 (PyInstaller executable)
	predictExe := filepath.Join(mlDir, "predict.exe")
	predictBin := filepath.Join(mlDir, "predict")
	if stat, err := os.Stat(predictExe); err == nil && !stat.IsDir() {
		// Windows executable
		cmd = exec.Command(predictExe, "--csv", csvPath, "--model", modelJson)
	} else if stat, err := os.Stat(predictBin); err == nil && !stat.IsDir() {
		// Linux/Mac executable
		cmd = exec.Command(predictBin, "--csv", csvPath, "--model", modelJson)
	} else {
		// 檢查是否有專案專屬的 Python 虛擬環境 (venv)
		pythonExe := "python"
		if stat, err := os.Stat(filepath.Join(cwd, "venv", "Scripts", "python.exe")); err == nil && !stat.IsDir() {
			// Windows 的 venv
			pythonExe = filepath.Join(cwd, "venv", "Scripts", "python.exe")
		} else if stat, err := os.Stat(filepath.Join(cwd, "venv", "bin", "python")); err == nil && !stat.IsDir() {
			// Mac/Linux 的 venv
			pythonExe = filepath.Join(cwd, "venv", "bin", "python")
		}
		cmd = exec.Command(pythonExe, predictPy, "--csv", csvPath, "--model", modelJson)
	}

	output, err := cmd.CombinedOutput()
if err != nil {
return nil, fmt.Errorf("predict error: %w, out: %s", err, string(output))
}

lines := strings.Split(strings.TrimSpace(string(output)), "\n")
var result FraudPrediction
for _, line := range lines {
if strings.HasPrefix(line, "{") && strings.HasSuffix(line, "}") {
if err := json.Unmarshal([]byte(line), &result); err == nil {
return &result, nil
}
}
}

return nil, fmt.Errorf("parse error")
}
