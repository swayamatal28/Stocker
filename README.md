# STOCKER

STOCKER is a personal market-research application for NSE and BSE-listed companies. It combines a manually maintained portfolio with delayed market prices, company fundamentals, recent news, stock-impact summaries, alerts, and explainable research.

> STOCKER is for informational research only. It does not connect to a broker, place trades, or provide guaranteed outcomes.

## Features

- Portfolio tracking for up to 50 holdings with quantity and average buy price
- Current value, invested value, daily change, and overall profit or loss
- Manual refresh of all portfolio prices and fundamentals
- Recent company and market news from configured publisher feeds
- Separate news-impact lists for portfolio holdings and other affected stocks
- Financial checks based on P/E, ROE, debt-to-equity, current ratio, operating margin, revenue growth, free cash flow, and 52-week price position
- Optional portfolio advice through a configured model provider
- Price and event alerts with cooldowns, quiet hours, and in-app delivery
- Market overview, sectors, movers, related companies, and risk flags
- Responsive interface with dark and light themes
- Source timestamps, delay labels, and links to original material

## Technology

| Area | Technology |
|---|---|
| API and workers | Go with Gin |
| Web application | React, Vite, Tailwind CSS, TanStack Query, Zustand |
| Primary database | MongoDB |
| Queues and live events | Redis Streams |
| Market information | Yahoo Finance delayed endpoints |
| News collection | RSS and Atom feeds |
| Authentication | Short-lived access tokens and rotating HttpOnly refresh sessions |

The backend includes structured logging, rate limiting, health endpoints, metrics, graceful shutdown, retry handling, deduplication, and retention jobs.

## Requirements

- Go 1.27 or newer
- Node.js and npm
- MongoDB with `mongod` available in `PATH`
- Redis available on `localhost:6379`

MongoDB and Redis must both be available for the complete application. The API can start without Redis only when `REDIS_REQUIRED=false`, but background news ingestion, analysis queues, distributed rate limiting, and live updates will be unavailable.

## Local setup

Create `.env` from `.env.example` and replace placeholder values where required. Do not commit `.env` or any real credentials.

Install the frontend packages once:

```powershell
cd C:\Users\Swayam\Desktop\Stocker
npm install --prefix apps/web
```

Start the backend from the project root:

```powershell
cd C:\Users\Swayam\Desktop\Stocker
go run main.go
```

The backend launcher loads `.env`, starts MongoDB when needed, checks Redis, builds the API and workers, and supervises the processes. Runtime files and logs are stored under `data/` and `.cache/runtime/`.

Start the frontend in a second terminal, also from the project root:

```powershell
cd C:\Users\Swayam\Desktop\Stocker
npm run dev
```

Open <http://localhost:5173>.

## Using the application

1. Register with a password containing at least 12 characters.
2. Search by company name, NSE symbol, BSE code, or ISIN.
3. Add a holding with its quantity and average buy price.
4. Use **Refresh** in the header to update every portfolio holding and reload the visible website data.
5. Select a holding to review its price, fundamentals, financial checks, risks, related companies, news, and optional portfolio advice.

The refresh result reports how many holdings were updated. If a provider request fails, the previous value keeps its original timestamp and is not presented as newly refreshed. Outside market hours, the latest available value will normally be the most recent closing or traded price.

## Data behavior

### Market information

The default provider supplies delayed NSE quotes and published company fundamentals without an API key. Only the public stock symbol or search term is sent to the provider. Portfolio quantities, average prices, account details, cookies, application secrets, article text, and environment values are not included.

Candlestick history and price charts are outside the application scope. Every displayed price includes its provider timestamp and delay status.

### News

News feeds are configured in [configs/news-sources.live.json](configs/news-sources.live.json). Each source has its own attribution, access policy, polling interval, rate limit, timeout, retention setting, and policy review date.

Only feed-provided headlines, summaries, timestamps, and canonical links are collected. Article pages, paywalls, login barriers, and access controls are not bypassed.

News is limited to a 24-hour window. Expired stories are excluded from queries, rejected during ingestion, and removed with their related analysis records by the maintenance worker. Portfolio holdings and essential application records are preserved.

Runtime mock providers and demo seeding are disabled. Synthetic legacy market and news records are removed during live startup.

### Portfolio advice

Portfolio advice is disabled until an approved endpoint is configured. With the default placeholder configuration, the interface displays `AI advice not available`.

To enable it, set:

```dotenv
AI_PROVIDER=http-json
AI_ENDPOINT=https://your-approved-provider.example/v1/advice
AI_API_KEY=insert your provider api key
AI_MODEL=insert your model name
```

The advice request contains only public company information: symbol, company name, quote, fundamentals, rule-based research results, and recent public-news metadata. It excludes portfolio quantity, average buy price, profit or loss, user identity, sessions, cookies, and application secrets.

## Main endpoints

- `GET /health/live` - process health
- `GET /health/ready` - database and dependency readiness
- `GET /api/v1/portfolio` - portfolio holdings and totals
- `POST /api/v1/refresh` - refresh every portfolio holding
- `GET /api/v1/news` - recent news
- `GET /api/v1/news/{id}/impact` - affected market and portfolio stocks
- `GET /api/v1/stocks/{symbol}` - stock details and research
- `GET /api/v1/portfolio/{symbol}/ai-advice` - optional portfolio advice

The complete contract is available in [docs/openapi.yaml](docs/openapi.yaml).

## Verification

```powershell
go test -buildvcs=false ./apps/api ./internal/... ./workers/...
go vet -buildvcs=false ./apps/api ./internal/... ./workers/...
npm --prefix apps/web test -- --run
npm --prefix apps/web run build
```

MongoDB-backed integration checks use temporary databases that are removed after each run:

```powershell
$env:STOCKER_INTEGRATION_TEST='1'
go test -buildvcs=false ./internal/httpapi ./internal/intelligence -count=1
```

Additional checks:

```powershell
./scripts/run-load.ps1 -BenchTime 5s
./scripts/security-check.ps1 -RequireExternalTools
```

## Documentation

- [Architecture](docs/architecture.md)
- [Data sources and assumptions](docs/assumptions-and-sources.md)
- [Database model](docs/database.md)
- [Ingestion and background jobs](docs/ingestion-and-jobs.md)
- [Analysis output schema](docs/ai-output.schema.json)
- [Signal scoring](docs/signal-scoring.md)
- [Security and compliance](docs/security-compliance.md)
- [API contract](docs/openapi.yaml)
- [Operations, backup, and recovery](docs/operations.md)
- [Evaluation](docs/evaluation.md)
- [Service objectives](docs/slo.md)
- [Load testing](docs/load-testing.md)
