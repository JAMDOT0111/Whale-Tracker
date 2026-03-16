# ETH Sweeper

On-chain Ethereum address scanner and visualization tool.

Scan any ETH wallet address, list all transactions (ETH / Internal / ERC-20 / ERC-721), and visualize upstream/downstream address relationships with an interactive graph.

## Features

- **Relationship Graph** — Interactive Cytoscape.js graph with node shapes/colors distinguishing EOA, contracts, exchanges, and cross-chain bridges
- **Transaction Panel** — Slide-out side panel with category and counterparty filtering
- **Balance Card** — Displays ETH balance and token holdings
- **Timeline** — Bar chart showing daily in/out transaction counts
- **Fund Flow** — Sankey diagram visualizing ETH flow direction
- **Context Menu** — Right-click to mark, rename nodes, or view related transactions
- **Search History** — Automatically records recently scanned addresses
- **URL History** — Browser back/forward navigation support

## Tech Stack

| | Technology |
|--|------|
| Backend | Go, Gin, Etherscan API V2 |
| Frontend | React, TypeScript, Vite, Tailwind CSS |
| Charts | Cytoscape.js, Recharts, D3-Sankey |
| Deployment | Docker, docker-compose |

## Quick Start

### Local Development

```bash
# 1. Set up API Key
cp backend/.env.sample backend/.env
# Edit backend/.env and fill in your Etherscan API Key

# 2. Start backend
cd backend
go run main.go

# 3. Start frontend (open another terminal)
cd frontend
npm install
npm run dev

# 4. Open http://localhost:5173
```

### Docker

```bash
# 1. Set up API Key
cp backend/.env.sample backend/.env
# Edit backend/.env

# 2. Launch
docker compose up --build

# 3. Open http://localhost:3000
```

## Linter & Formatter

### Frontend (ESLint + Prettier)

```bash
cd frontend

# Check for lint errors
npm run lint

# Auto-fix lint errors
npm run lint:fix

# Check formatting
npm run format:check

# Auto-format
npm run format
```

### Backend (go vet + gofmt)

```bash
cd backend

# Static analysis (check for common errors)
go vet ./...

# Check formatting (list unformatted files)
gofmt -l .

# Auto-format
gofmt -w .
```

## Documentation

- [DOCS.md](DOCS.md) — Full technical documentation (architecture, API spec, module details)
- [Todo.md](Todo.md) — Implementation progress and upcoming features

## Getting an API Key

1. Register at https://etherscan.io/register
2. After login, go to https://etherscan.io/myapikey
3. Click "Add" to create a new API Key
4. Paste it into `backend/.env`

Free tier: 100,000 calls/day, 3 calls/sec.
