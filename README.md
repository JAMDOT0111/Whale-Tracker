# ETH Sweeper

Ethereum 鏈上地址掃描與視覺化工具。

掃描任意 ETH 錢包地址，列出所有交易紀錄（ETH / Internal / ERC-20 / ERC-721），並以互動式圖形展示上下游地址關係。

## 功能

- **關係圖** — Cytoscape.js 互動式圖，節點顏色/形狀區分 EOA、合約、交易所、跨鏈橋
- **交易面板** — 右側滑出面板，可篩選類型和對手地址
- **餘額卡片** — 顯示 ETH 餘額和代幣持倉
- **時間軸** — 按日期統計轉入/轉出交易數量
- **資金流向** — Sankey 圖顯示 ETH 流向
- **右鍵選單** — 標記、命名節點，查看相關交易
- **搜尋紀錄** — 自動記錄最近掃描的地址
- **URL 歷史** — 支援瀏覽器上一頁/下一頁

## 技術棧

| | 技術 |
|--|------|
| 後端 | Go, Gin, Etherscan API V2 |
| 前端 | React, TypeScript, Vite, Tailwind CSS |
| 圖表 | Cytoscape.js, Recharts, D3-Sankey |
| 部署 | Docker, docker-compose |

## 快速開始

### 本地開發

```bash
# 1. 設定 API Key
cp backend/.env.sample backend/.env
# 編輯 backend/.env，填入 Etherscan API Key

# 2. 啟動後端
cd backend
go run main.go

# 3. 啟動前端（另開終端）
cd frontend
npm install
npm run dev

# 4. 開啟 http://localhost:5173
```

### Docker

```bash
# 1. 設定 API Key
cp backend/.env.sample backend/.env
# 編輯 backend/.env

# 2. 一鍵啟動
docker compose up --build

# 3. 開啟 http://localhost:3000
```

## Linter & Formatter

### 前端（ESLint + Prettier）

```bash
cd frontend

# 檢查 lint 錯誤
npm run lint

# 自動修復 lint 錯誤
npm run lint:fix

# 檢查格式
npm run format:check

# 自動格式化
npm run format
```

### 後端（go vet + gofmt）

```bash
cd backend

# 靜態分析（檢查常見錯誤）
go vet ./...

# 檢查格式（列出未格式化的檔案）
gofmt -l .

# 自動格式化
gofmt -w .
```

## 文件

- [DOCS.md](DOCS.md) — 完整技術文件（架構、API 規格、模組說明）
- [Todo.md](Todo.md) — 實作進度與待辦功能

## 取得 API Key

1. 前往 https://etherscan.io/register 註冊
2. 登入後到 https://etherscan.io/myapikey
3. 點 Add 建立 API Key
4. 貼到 `backend/.env`

免費版：100,000 calls/day，3 calls/sec。
