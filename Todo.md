# ETH Sweeper — 實作進度

## 已完成功能

### 後端 (Go + Gin)
- [x] Etherscan API V2 封裝（4 種交易類型、分頁、快取）
- [x] 滑動視窗 Rate Limiter（3 req/sec）+ 自動 Retry
- [x] 交易掃描 API（POST /api/scan）
- [x] 單層圖建構 API（POST /api/graph）
- [x] 餘額查詢 API（POST /api/balance）
- [x] 合約地址檢測（eth_getCode）
- [x] 已知地址標籤（60+ 交易所/跨鏈橋）
- [x] .env 環境變數管理（godotenv）

### 前端 (React + Vite + TypeScript)
- [x] 地址輸入與驗證
- [x] Cytoscape.js 互動式關係圖（力導向佈局）
- [x] 節點視覺區分（EOA/合約/交易所/跨鏈橋/已標記）
- [x] 右鍵選單（標記/命名/篩選交易/Etherscan 連結）
- [x] 右側滑出交易面板（類型篩選、對手地址篩選）
- [x] ETH 餘額 + 代幣持倉卡片
- [x] Recharts 交易時間軸
- [x] D3-Sankey 資金流向圖
- [x] 搜尋紀錄（localStorage）
- [x] 標記 + 命名持久化（localStorage）
- [x] URL 歷史（瀏覽器上一頁/下一頁）
- [x] 圖的局部更新（標記/命名不重建圖）

### 基建
- [x] Docker 打包（docker-compose）
- [x] 技術文件（DOCS.md）

---

## 待實作功能（Stub 已建立）

以下功能的 API 接口已定義在 `handler/scan.go`，前端 client 已定義在 `api/client.ts`，
目前回傳 501 Not Implemented。

### 🟡 中等難度

#### ENS 名稱解析
- **API**: `POST /api/resolve-ens`
- **後端**: 呼叫 Etherscan `proxy/eth_call` 或第三方 ENS API，將 `vitalik.eth` 解析為 `0xd8dA...`
- **前端**: 輸入框支援 ENS 名稱；圖上節點顯示 ENS 名稱
- **參考**: [ENS Docs](https://docs.ens.domains/)

#### CSV 匯出
- **API**: `POST /api/export`
- **後端**: 取得交易後轉為 CSV，回傳 `Content-Type: text/csv`
- **前端**: 交易面板加「匯出 CSV」按鈕，觸發下載
- **欄位**: hash, from, to, value, asset, category, timestamp

#### Gas 使用分析
- **API**: `POST /api/gas-analytics`
- **後端**: 在 `EtherscanNormalTx` 加入 `gasUsed`, `gasPrice` 欄位，統計：
  - 總 gas 花費（ETH）
  - 平均每筆 gas
  - 最高 gas 的交易
  - 按月分組統計
- **前端**: 新增 `GasAnalytics.tsx` 元件，用 Recharts 畫 gas 趨勢圖

### 🟠 較高難度

#### 代幣授權檢查
- **API**: `POST /api/token-approvals`
- **後端**: 查詢 ERC-20 的 `Approve` 事件 log，列出：
  - 被授權的合約地址
  - 授權的代幣
  - 授權額度（unlimited 或具體金額）
- **前端**: 新增 `TokenApprovals.tsx`，列表顯示，標記風險高的無限授權
- **參考**: Etherscan Event Log API

#### 合約互動解碼
- **API**: `POST /api/contract-decode`
- **後端**: 取得交易的 input data，用 ABI 或 4byte 資料庫解碼函式名稱和參數
- **前端**: 交易面板中顯示解碼後的函式名稱（如 `swap(uint256, address[])`）
- **參考**: [4byte.directory](https://www.4byte.directory/)

### 🔴 高難度

#### 地址風險評分
- **API**: `POST /api/risk-score`
- **後端**: 根據多個因素計算 0-100 風險分數：
  - 是否與混幣器互動（Tornado Cash 地址清單）
  - 交易頻率異常度
  - 是否與已知詐騙地址互動
  - 帳戶年齡和活躍度
- **前端**: 新增 `RiskScore.tsx`，顯示分數儀表板和風險因素細項
- **注意**: 需要維護混幣器/詐騙地址清單

#### 多鏈支援
- **後端**: `chainid` 參數化，支援 Polygon (137)、Arbitrum (42161)、Optimism (10) 等
- **前端**: 加鏈切換下拉選單
- **注意**: labels.go 的地址清單需要按鏈區分

#### 即時交易監控（WebSocket）
- **後端**: 新增 WebSocket endpoint，用 Etherscan 或 Alchemy 的 pending tx stream
- **前端**: 即時更新圖和交易列表
- **注意**: 架構改動較大，需要 goroutine 管理

### 🟡 基礎建設

#### MongoDB 整合
- **後端**: 安裝 `go.mongodb.org/mongo-driver`，建立 `db/mongo.go` 連線模組
- **集合設計**:
  - `users` — 使用者帳號資料
  - `marks` — 使用者標記的地址（by user_id）
  - `names` — 使用者自訂地址名稱（by user_id）
  - `history` — 搜尋紀錄（by user_id）
  - `labels` — 自訂標籤（可擴充 labels.go 的硬編碼清單）
  - `tx_cache` — 交易快取（避免重複 API 呼叫，設 TTL）
- **Docker**: `docker-compose.yml` 加入 MongoDB 服務
- **.env**: 加入 `MONGO_URI=mongodb://localhost:27017/eth-sweeper`

#### JWT 使用者登入/註冊
- **API**:
  - `POST /api/auth/register` — 帳號密碼註冊（密碼用 bcrypt 雜湊）
  - `POST /api/auth/login` — 登入，回傳 JWT token
  - `GET /api/auth/me` — 用 token 取得當前使用者資訊
- **後端**: 新增 `handler/auth.go`、`service/auth.go`、`middleware/jwt.go`
- **前端**: 新增登入/註冊頁面、token 存 localStorage、API 請求帶 Authorization header
- **套件**: `golang-jwt/jwt/v5`、`golang.org/x/crypto/bcrypt`

#### 使用者資料持久化
- **後端**: 需要登入的 API 加上 JWT middleware
  - `GET /api/user/marks` — 取得該使用者的標記
  - `POST /api/user/marks` — 新增/移除標記
  - `GET /api/user/names` — 取得自訂名稱
  - `POST /api/user/names` — 設定名稱
  - `GET /api/user/history` — 取得搜尋紀錄
- **前端**: 登入後從後端載入資料，取代 localStorage；未登入時 fallback 到 localStorage

---

## 教學建議實作順序

1. **MongoDB 整合** — 學資料庫連線、集合設計
2. **JWT 登入/註冊** — 學認證機制、middleware
3. **使用者資料持久化** — 學 CRUD + 權限控制
4. **ENS 名稱解析** — 學 API 整合
5. **CSV 匯出** — 學 HTTP response 格式、前端檔案下載
6. **Gas 分析** — 學資料聚合 + 圖表
7. **代幣授權** — 學 Event Log 解析
8. **合約解碼** — 學 ABI 解碼
9. **風險評分** — 學評分模型設計
10. **多鏈支援** — 學架構重構
11. **即時監控** — 學 WebSocket
12. **雲端部署** — 學 CI/CD + 雲端服務（Cloudflare Pages + Railway + MongoDB Atlas）

---

## 本次任務：改為單一地址輸入搜尋分析

- [x] 在詳情區上方加入地址輸入框（沿用既有元件）
- [x] 輸入有效地址後呼叫單地址查詢分析流程
- [x] 保留原本清單點選地址功能
- [x] 調整空狀態提示文案為輸入導向
- [ ] 前端 lint 檢查（本機環境找不到 npm 指令，待你本機重跑）

---

## 本次任務：重要人物相關新聞

- [x] 新增後端 Google News RSS 服務，抓取 Vitalik / Trump 加密貨幣相關新聞
- [x] 新增 `/api/news/crypto-figures` endpoint，來源失敗或無結果時回傳真實空狀態
- [x] 前端在「ETH 相關報導」下方新增「重要人物相關新聞」區塊
- [x] 更新 README / IMPLEMENTATION 說明與資料來源
- [x] 執行後端測試與前端 build/lint
