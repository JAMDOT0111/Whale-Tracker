# ETH Whale & Smart Money Scanner

This repo now implements the first-stage, no-membership scanner plan.

## What Is Included

- No feature tiers and no billing integration.
- Etherscan Top Accounts sync for the whale seed list. Configure `ETHERSCAN_TOP_ACCOUNTS_CSV_URL` for a CSV source, or let the backend parse the public Etherscan accounts pages as a fallback.
- Whale list API with threshold filtering, sorting, pagination, labels, confidence, source, and evidence references.
- Single-address detail APIs that reuse the existing Etherscan scanner and graph code.
- Watchlists, alert events, notification preferences, Gmail dry-run/send support, and a 30-minute optional scheduler.
- ETH price series from CoinGecko with demo fallback data.
- ETH news links from GDELT with demo fallback data.
- Vitalik / Trump crypto-related news from Google News RSS with explicit empty states when no matching article is found.
- PostgreSQL migration schema for the planned durable store and Docker Compose services for PostgreSQL/Redis.
- A new dark dashboard UI with whale filters, `TEH -> ETH` correction, watchlist controls, price chart, news, alerts, and address detail panels.

## Runtime Notes

- The current Go runtime uses an in-memory app store so the project can start without a database during local development.
- PostgreSQL tables are provided in `backend/migrations/001_eth_scanner.sql`; wiring the repository implementation to those tables is the next persistence step.
- Set `ETHERSCAN_API_KEY` to enable live balance, transaction, and graph lookups. Without it, the whale dashboard still opens with demo seed data and clear API errors for live scans.
- Set `ETHERSCAN_TOP_ACCOUNTS_CSV_URL` to an authorized CSV URL for the top accounts seed data. If it is empty, `POST /api/admin/whales/import-etherscan-url` fetches Etherscan accounts pages directly. Use `ETHERSCAN_TOP_ACCOUNTS_PAGES=400` to fetch up to 10,000 accounts, and `AUTO_IMPORT_WHALES_ON_START=true` to sync when the server starts.
- Important-figure crypto news is fetched from Google News RSS. The backend filters article titles so the panel only shows items that mention both the person and crypto-related keywords.
- Set `GMAIL_DRY_RUN=true` for local notification testing. Set `GMAIL_ACCESS_TOKEN` and `GMAIL_FROM` only when you are ready to send through Gmail API.
- Set `ENABLE_JOBS=true` to run the background watchlist scanner.

## Useful Endpoints

- `GET /api/whales?min_balance_eth=1000&sort=balance_desc&page=1`
- `POST /api/admin/whales/import-etherscan-csv`
- `POST /api/admin/whales/import-etherscan-url`
- `GET /api/addresses/:address`
- `GET /api/addresses/:address/transactions`
- `GET /api/addresses/:address/graph`
- `GET /api/addresses/:address/ai-summary`
- `GET /api/prices/eth/ohlc?interval=5m`
- `GET /api/news/eth`
- `GET /api/news/crypto-figures`
- `POST /api/watchlists`
- `GET /api/alerts`
- `POST /api/notification-preferences`

## Safety Boundary

This application only reads public chain data and sends notifications. It does not handle private keys, sign transactions, sweep assets, or automatically transfer funds.
