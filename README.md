# Whale-Tracker

ETH whale and smart-money tracking prototype for public on-chain analysis and Gmail notifications.

掃描巨鯨地址、分析鏈上互動、追蹤地址異動並寄送 Gmail 通知。本專案只讀取公開鏈上資料，不處理私鑰、不簽交易、不自動轉帳。

## Features

- Whale list from Etherscan Top Accounts live data
- Watchlist tracking with configurable ETH notification threshold
- Gmail OAuth / SMTP notification delivery
- Address balance, transaction, relationship graph, timeline, and flow visualization
- ETH price chart, public news feed, labels, risk and heuristic alert evidence
- Vitalik / Trump crypto figure news feed from Google News RSS
- One-user-experience design: no paid membership tiers and no Stripe integration

## Tech Stack

| Area | Technology |
| --- | --- |
| Backend | Go, Gin, Etherscan API V2 |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
| Charts | Cytoscape.js, Recharts, D3-Sankey |
| Deployment | Docker, docker-compose |

## Quick Start

```bash
# 1. Configure environment
cp backend/.env.sample backend/.env
# Edit backend/.env and fill in local API/OAuth credentials

# 2. Start backend
cd backend
go run main.go

# 3. Start frontend in another terminal
cd frontend
npm install
npm run dev

# 4. Open the app
# http://localhost:5173
```

## Useful Environment Variables

```env
ETHERSCAN_API_KEY=
ETHERSCAN_TOP_ACCOUNTS_PAGES=400
AUTO_IMPORT_WHALES_ON_START=true
ENABLE_DEMO_DATA=false
COINGECKO_API_KEY=
GOOGLE_OAUTH_CLIENT_ID=
GOOGLE_OAUTH_CLIENT_SECRET=
GOOGLE_OAUTH_REDIRECT_URL=http://127.0.0.1:8080/api/auth/google/callback
FRONTEND_URL=http://127.0.0.1:5173
ENABLE_JOBS=false
WATCHLIST_SCAN_INTERVAL=1m
```

Do not commit real API keys, OAuth client secrets, Gmail app passwords, or production database credentials.

## Linter And Formatter

```bash
cd frontend
npm run lint
npm run format:check
```

```bash
cd backend
go vet ./...
gofmt -l .
```

## Documentation

- [DOCS.md](DOCS.md): technical documentation
- [Todo.md](Todo.md): implementation progress and upcoming work
- [IMPLEMENTATION.md](IMPLEMENTATION.md): no-membership whale scanner implementation notes
