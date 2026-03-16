package model

import "encoding/json"

type Transaction struct {
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	Asset       string `json:"asset"`
	Category    string `json:"category"`
	BlockNumber string `json:"block_number"`
	Timestamp   string `json:"timestamp"`
}

type ScanRequest struct {
	Address  string `json:"address" binding:"required"`
	PageSize int    `json:"page_size"`
	PageKey  string `json:"page_key"`
}

type ScanResponse struct {
	Transactions []Transaction `json:"transactions"`
	PageKey      string        `json:"page_key"`
	Total        int           `json:"total"`
}

type GraphRequest struct {
	Address string `json:"address" binding:"required"`
}

type GraphNode struct {
	ID         string `json:"id"`
	Label      string `json:"label"`
	IsCenter   bool   `json:"is_center"`
	IsContract bool   `json:"is_contract"`
	Tag        string `json:"tag,omitempty"`
	TagName    string `json:"tag_name,omitempty"`
	TxCount    int    `json:"tx_count"`
}

type GraphEdge struct {
	Source  string `json:"source"`
	Target  string `json:"target"`
	Value   string `json:"value"`
	TxCount int    `json:"tx_count"`
}

type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
}

type BalanceRequest struct {
	Address string `json:"address" binding:"required"`
}

type TokenBalance struct {
	Symbol  string `json:"symbol"`
	Name    string `json:"name"`
	Balance string `json:"balance"`
}

type BalanceResponse struct {
	EthBalance string         `json:"eth_balance"`
	Tokens     []TokenBalance `json:"tokens"`
}

type EtherscanTokenInfo struct {
	TokenName    string `json:"TokenName"`
	TokenSymbol  string `json:"TokenSymbol"`
	TokenDecimal string `json:"TokenDecimal"`
	Balance      string `json:"balance"`
}

// Etherscan API types

type EtherscanResponse struct {
	Status  string          `json:"status"`
	Message string          `json:"message"`
	Result  json.RawMessage `json:"result"`
}

type EtherscanNormalTx struct {
	BlockNumber string `json:"blockNumber"`
	TimeStamp   string `json:"timeStamp"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	IsError     string `json:"isError"`
}

type EtherscanInternalTx struct {
	BlockNumber string `json:"blockNumber"`
	TimeStamp   string `json:"timeStamp"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	Value       string `json:"value"`
	IsError     string `json:"isError"`
}

type EtherscanTokenTx struct {
	BlockNumber  string `json:"blockNumber"`
	TimeStamp    string `json:"timeStamp"`
	Hash         string `json:"hash"`
	From         string `json:"from"`
	To           string `json:"to"`
	Value        string `json:"value"`
	TokenName    string `json:"tokenName"`
	TokenSymbol  string `json:"tokenSymbol"`
	TokenDecimal string `json:"tokenDecimal"`
}

type EtherscanNFTTx struct {
	BlockNumber string `json:"blockNumber"`
	TimeStamp   string `json:"timeStamp"`
	Hash        string `json:"hash"`
	From        string `json:"from"`
	To          string `json:"to"`
	TokenID     string `json:"tokenID"`
	TokenName   string `json:"tokenName"`
	TokenSymbol string `json:"tokenSymbol"`
}
