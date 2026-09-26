# STOCKER

STOCKER is an evidence-first research application for NSE/BSE-listed companies. It connects attributed news and exchange announcements to securities, combines them with market context, and presents explainable—not prescriptive—signals.

> Informational research only. STOCKER does not place trades and its signals are not financial advice.

## What is included

- Go 1.27 + Gin API with MongoDB, Redis, structured JSON logs, readiness/liveness, metrics endpoint, rate limits, and graceful shutdown.
- Rotating refresh sessions, memory-only short-lived access tokens, HttpOnly-cookie session restoration, bcrypt password hashing, trusted-origin checks, and role-ready authorization.
- React 19 with plain JavaScript/JSX, Vite, Tailwind, TanStack Query, Zustand, Recharts, responsive navigation, and dark/light themes.
- Database-enforced 10-stock watchlist with add/remove/pause/resume controls, NSE/BSE/ISIN search across the six-stock Phase 1 fixture universe, delayed mock quotes, persisted source health, authenticated SSE updates, and an explainable fixture signal.
- Phase 2 RSS/Atom adapter, policy gate, durable cursors, retries, rate limits, circuit breaking, exact/near deduplication, stock links, Mongo-backed news APIs, Redis Streams outbox, dead-letter replay, saved parser fixtures, and a working News Explorer.
- Versioned scoring engine with evidence gates and unit tests.

The mock provider is intentional: no third-party site is scraped and no unlicensed market values are presented as live. See [assumptions and source licensing](docs/assumptions-and-sources.md).

## Run locally

Install and start MongoDB as a native service, then set the values from `.env.example` in your shell. The default expects MongoDB on `localhost:27017`. Redis on `localhost:6379` is optional for the basic development UI and required for ingestion, distributed rate limiting, queues, and production (`REDIS_REQUIRED=true`).

```bash
go run ./apps/api
npm --prefix apps/web install
npm --prefix apps/web run dev
```

The API seeds one project-owned evidence fixture in mock mode. In a third terminal, start the ingestion worker to exercise polling, Redis Streams delivery, source cursors, and ongoing collection:

```bash
go run ./workers/ingestion
```

Open <http://localhost:5173>, register with a password of at least 12 characters, search for `RELIANCE`, `HDFCBANK`, `INFY`, `TCS`, `ITC`, or `LT`, and add up to 10 securities. API docs are in [OpenAPI](docs/openapi.yaml).

## Verification

```bash
go test ./apps/api ./internal/... ./workers/...
go vet ./apps/api ./internal/... ./workers/...
npm --prefix apps/web test -- --run
npm --prefix apps/web run build
```

With local MongoDB running, execute the isolated API integration flow (it creates and drops only a uniquely named test database):

```powershell
$env:STOCKER_INTEGRATION_TEST='1'; go test ./internal/httpapi -run TestPhase1AuthSearchAndWatchlistFlow -count=1 -v
```

## Documentation index

- [Architecture and component diagram](docs/architecture.md)
- [Assumptions and provider/licensing register](docs/assumptions-and-sources.md)
- [Database ER model](docs/database.md)
- [Source adapters and background jobs](docs/ingestion-and-jobs.md)
- [AI schema and grounding contract](docs/ai-output.schema.json)
- [Signal scoring specification](docs/signal-scoring.md)
- [Security/compliance checklist](docs/security-compliance.md)
- [API contract](docs/openapi.yaml)
- [Native operations, backup, and recovery](docs/operations.md)

## Delivery phases

Phase 1 is complete as a working vertical slice, including session recovery, full watchlist management, authenticated live-update plumbing, real navigation for available modules, and Mongo-backed integration coverage. Phase 2 provides an end-to-end, database-backed ingestion and exploration path using a project-owned fixture by default. The reusable RSS/Atom adapter remains fail-closed until its source-policy environment fields explicitly approve automated access; exchange data must not be enabled without the required agreement. Phase 3 AI/provider work and Phases 4–6 remain future implementation work.
