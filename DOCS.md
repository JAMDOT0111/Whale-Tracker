# ETH Sweeper — 技術文件

## 專案簡介

ETH Sweeper 是一個 Ethereum 鏈上地址掃描與視覺化工具，可以掃描任意 ETH 錢包地址，列出所有交易紀錄，並將上下游地址關係畫成互動式圖形。

適合用於：
- 鏈上資金追蹤與分析
- 錢包活動審計
- 學習區塊鏈瀏覽器的實作方式

---

## 架構總覽

```
┌──────────────────────────────────────────────────────────┐
│                    Frontend (React)                       │
│  Vite + TypeScript + Tailwind CSS                        │
│                                                          │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌────────────┐ │
│  │ GraphView│ │ Sankey   │ │ Timeline  │ │ Balance    │ │
│  │Cytoscape │ │ D3-Sankey│ │ Recharts  │ │ Card       │ │
│  └──────────┘ └──────────┘ └───────────┘ └────────────┘ │
│  ┌──────────┐ ┌──────────┐ ┌───────────┐ ┌────────────┐ │
│  │ TxPanel  │ │ Context  │ │ Marked    │ │ Search     │ │
│  │          │ │ Menu     │ │ Addresses │ │ History    │ │
│  └──────────┘ └──────────┘ └───────────┘ └────────────┘ │
│                      │                                   │
│                Vite Proxy /api → :8080                   │
└──────────────────────┬───────────────────────────────────┘
                       │ HTTP POST (JSON)
┌──────────────────────┴───────────────────────────────────┐
│                   Backend (Go + Gin)                      │
│                                                          │
│  ┌─────────────────────────────────────────────────────┐ │
│  │ handler/scan.go                                     │ │
│  │  POST /api/scan     → 掃描交易                      │ │
│  │  POST /api/graph    → 建構關係圖                    │ │
│  │  POST /api/balance  → 查詢餘額                      │ │
│  └─────────────────────────────────────────────────────┘ │
│  ┌───────────────┐ ┌──────────────┐ ┌────────────────┐  │
│  │ etherscan.go  │ │ graph.go     │ │ labels.go      │  │
│  │ API 呼叫封裝  │ │ 圖建構邏輯   │ │ 已知地址標籤   │  │
│  │ Rate Limiter  │ │ 合約檢測     │ │ 交易所/跨鏈橋  │  │
│  │ 快取          │ │              │ │                │  │
│  └───────┬───────┘ └──────────────┘ └────────────────┘  │
│          │                                               │
└──────────┼───────────────────────────────────────────────┘
           │ HTTPS (Rate Limited: 3 req/sec)
┌──────────┴───────────────────────────────────────────────┐
│              Etherscan API V2 (chainid=1)                 │
│  account/txlist, txlistinternal, tokentx, tokennfttx     │
│  account/balance, account/tokenlist                      │
│  proxy/eth_getCode                                       │
└──────────────────────────────────────────────────────────┘
```

---

## 技術選型

| 層級 | 技術 | 選用原因 |
|------|------|----------|
| 後端語言 | Go | 高效能、強型別、適合 API 服務 |
| HTTP 框架 | Gin | 輕量、高效能、社群活躍 |
| 前端框架 | React + TypeScript | 元件化、型別安全 |
| 建置工具 | Vite | 快速 HMR、原生 ESM |
| CSS | Tailwind CSS v4 | Utility-first、快速開發 |
| 圖視覺化 | Cytoscape.js | 專業級圖分析、多種佈局、互動豐富 |
| 長條圖 | Recharts | React 原生圖表庫 |
| 流向圖 | D3-Sankey | 專業 Sankey 圖實作 |
| 資料來源 | Etherscan API V2 | 免費 100K calls/day、涵蓋所有交易類型 |

---

## 後端 API 規格

### POST /api/scan

掃描地址的交易紀錄。

**Request:**
```json
{
  "address": "0x...",
  "page_size": 100,
  "page_key": ""
}
```

**Response:**
```json
{
  "transactions": [
    {
      "hash": "0x...",
      "from": "0x...",
      "to": "0x...",
      "value": "1.5",
      "asset": "ETH",
      "category": "external",
      "block_number": "12345678",
      "timestamp": "2024-01-01T00:00:00Z"
    }
  ],
  "page_key": "",
  "total": 42
}
```

### POST /api/graph

取得地址的一層關係圖資料。

**Request:**
```json
{ "address": "0x..." }
```

**Response:**
```json
{
  "nodes": [
    {
      "id": "0x...",
      "label": "Binance",
      "is_center": false,
      "is_contract": false,
      "tag": "exchange",
      "tag_name": "Binance",
      "tx_count": 42
    }
  ],
  "edges": [
    { "source": "0x...", "target": "0x...", "value": "10.5", "tx_count": 3 }
  ]
}
```

### POST /api/balance

查詢地址的 ETH 餘額與代幣持倉。

**Request:**
```json
{ "address": "0x..." }
```

**Response:**
```json
{
  "eth_balance": "1.234",
  "tokens": [
    { "symbol": "USDT", "name": "Tether USD", "balance": "1000" }
  ]
}
```

---

## 後端核心模組

### service/etherscan.go
- `EtherscanClient` — 封裝所有 Etherscan API 呼叫
- **滑動視窗 Rate Limiter**：1.1 秒視窗內最多 3 次呼叫
- **自動 Retry**：收到 rate limit 錯誤時等 1.5 秒重試，最多 3 次
- **記憶體快取**：`sync.Map` 快取交易資料和合約檢查結果
- 支援 4 種交易類型：external、internal、erc20、erc721
- 合約檢測：`eth_getCode` 判斷 EOA vs 合約

### service/graph.go
- `GraphService.BuildGraph()` — 單層圖建構
- 取得中心地址的交易，聚合為節點和邊
- 邊的 value 和 tx_count 會聚合同方向的交易
- 對每個節點檢查合約狀態和已知標籤

### service/labels.go
- 內建 60+ 個已知地址（交易所、跨鏈橋）
- 涵蓋：Binance、Coinbase、Kraken、OKX、Bybit、Bitfinex、Gemini、KuCoin、Gate.io、HTX、MEXC
- 跨鏈橋：Wormhole、Stargate、Across、Hop、Celer、Synapse、Arbitrum/Optimism/Polygon/Base/zkSync Bridge

---

## 前端核心元件

| 元件 | 功能 |
|------|------|
| `App.tsx` | 主頁面、狀態管理、URL 歷史 |
| `AddressInput` | 地址輸入框 |
| `GraphView` | Cytoscape.js 互動式關係圖 |
| `TransactionPanel` | 右側滑出交易列表面板 |
| `TransactionTimeline` | Recharts 交易時間軸長條圖 |
| `SankeyChart` | D3-Sankey 資金流向圖 |
| `BalanceCard` | ETH 餘額 + 代幣持倉卡片 |
| `NodeContextMenu` | 右鍵選單（標記/命名/篩選交易/Etherscan） |
| `MarkedAddresses` | 已標記地址面板 |
| `SearchHistory` | 最近搜尋紀錄 |

### 前端狀態持久化（localStorage）
| Key | 內容 |
|-----|------|
| `eth-sweeper-history` | 搜尋紀錄 |
| `eth-sweeper-marks` | 標記的地址 |
| `eth-sweeper-names` | 自訂地址名稱 |

### 圖上節點視覺區分
| 形狀 | 顏色 | 意義 |
|------|------|------|
| 圓形 | 金色 | 中心地址 |
| 圓形 | 紫色 | 一般地址 (EOA) |
| 方塊 | 綠色 | 合約地址 |
| 菱形 | 紅色 | 交易所 |
| 六角形 | 青色 | 跨鏈橋 |
| 黃色粗邊框 | — | 使用者標記 |

---

## 環境設定

1. 註冊 [Etherscan](https://etherscan.io/register) 取得免費 API Key
2. 複製 `backend/.env.sample` 為 `backend/.env`，填入 API Key
3. 啟動後端：`cd backend && go run main.go`
4. 啟動前端：`cd frontend && npm install && npm run dev`
5. 開啟 http://localhost:5173

---

## 目錄結構

```
Eth-Sweeper/
├── backend/
│   ├── main.go                    # 入口點，Gin server 設定
│   ├── handler/scan.go            # API handler
│   ├── service/etherscan.go       # Etherscan API 封裝 + Rate Limiter
│   ├── service/graph.go           # 圖建構邏輯
│   ├── service/labels.go          # 已知地址標籤清單
│   ├── model/types.go             # 資料結構定義
│   ├── .env.sample                # 環境變數範例
│   ├── go.mod / go.sum
├── frontend/
│   ├── src/App.tsx                # 主頁面
│   ├── src/components/            # 所有 UI 元件
│   ├── src/api/client.ts          # 後端 API 呼叫
│   ├── src/types/index.ts         # TypeScript 型別定義
│   ├── vite.config.ts             # Vite + Tailwind + Proxy 設定
│   └── package.json
├── DOCS.md                        # 本文件
├── Todo.md                        # 實作進度與待辦事項
├── docker-compose.yml             # Docker 一鍵啟動
├── .gitignore
└── README.md
```
