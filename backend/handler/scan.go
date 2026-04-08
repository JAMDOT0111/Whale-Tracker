package handler

import (
	"eth-sweeper/model"
	"eth-sweeper/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	etherscan *service.EtherscanClient
	graph     *service.GraphService
	store     *service.AppStore
	prices    *service.PriceService
	news      *service.NewsService
	figures   *service.FigureNewsService
	alerts    *service.AlertService
}

func NewHandler(etherscan *service.EtherscanClient, graph *service.GraphService, store *service.AppStore, prices *service.PriceService, news *service.NewsService, figures *service.FigureNewsService, alerts *service.AlertService) *Handler {
	return &Handler{
		etherscan: etherscan,
		graph:     graph,
		store:     store,
		prices:    prices,
		news:      news,
		figures:   figures,
		alerts:    alerts,
	}
}

func (h *Handler) ScanAddress(c *gin.Context) {
	var req model.ScanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	req.Address = strings.ToLower(strings.TrimSpace(req.Address))
	if !isValidEthAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}

	resp, err := h.etherscan.GetTransactions(req.Address, req.PageKey, req.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch transactions: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetGraph(c *gin.Context) {
	var req model.GraphRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	req.Address = strings.ToLower(strings.TrimSpace(req.Address))
	if !isValidEthAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}

	resp, err := h.graph.BuildGraph(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to build graph: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *Handler) GetBalance(c *gin.Context) {
	var req model.BalanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request: " + err.Error()})
		return
	}

	req.Address = strings.ToLower(strings.TrimSpace(req.Address))
	if !isValidEthAddress(req.Address) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Ethereum address"})
		return
	}

	resp, err := h.etherscan.GetBalance(req.Address)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch balance: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, resp)
}

// POST /api/resolve-ens — ENS 名稱解析
func (h *Handler) ResolveENS(c *gin.Context) {
	// TODO: 接收 ENS 名稱（如 vitalik.eth），回傳對應的 0x 地址
	// 使用 Etherscan proxy/eth_call 或第三方 ENS API
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// POST /api/export — 匯出交易紀錄為 CSV
func (h *Handler) ExportCSV(c *gin.Context) {
	// TODO: 接收 address，查詢交易後轉為 CSV 格式回傳
	// Content-Type: text/csv, Content-Disposition: attachment
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// POST /api/gas-analytics — Gas 使用分析
func (h *Handler) GetGasAnalytics(c *gin.Context) {
	// TODO: 接收 address，統計 gas 花費（總量、平均、最高、按月分組）
	// 需要在 EtherscanNormalTx 中加入 gasUsed, gasPrice 欄位
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// POST /api/token-approvals — 代幣授權檢查
func (h *Handler) GetTokenApprovals(c *gin.Context) {
	// TODO: 接收 address，查詢 ERC-20 approve 事件
	// 列出哪些合約被授權可以花費多少代幣
	// 使用 Etherscan event log API
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// POST /api/risk-score — 地址風險評分
func (h *Handler) GetRiskScore(c *gin.Context) {
	// TODO: 接收 address，根據以下因素計算風險分數：
	// - 是否與已知混幣器互動（Tornado Cash 等）
	// - 交易頻率異常
	// - 是否與被標記的詐騙地址互動
	// - 帳戶年齡
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

// POST /api/contract-decode — 合約互動解碼
func (h *Handler) DecodeContract(c *gin.Context) {
	// TODO: 接收 tx hash，取得 input data 並解碼函式簽名
	// 使用 Etherscan getabi + 4byte 解碼
	c.JSON(http.StatusNotImplemented, gin.H{"error": "Not implemented yet"})
}

func isValidEthAddress(addr string) bool {
	if len(addr) != 42 {
		return false
	}
	if !strings.HasPrefix(addr, "0x") {
		return false
	}
	for _, c := range addr[2:] {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}
