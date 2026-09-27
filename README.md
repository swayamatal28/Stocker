# STOCKER

STOCKER is an evidence-first research application for NSE/BSE-listed companies. It connects attributed news and exchange announcements to securities, combines them with market context, and presents explainable—not prescriptive—signals.

> Informational research only. STOCKER does not place trades and its signals are not financial advice.

## What is included

- Go 1.27 + Gin API with MongoDB, Redis, structured JSON logs, readiness/liveness, metrics endpoint, rate limits, and graceful shutdown.
- Rotating refresh sessions, memory-only short-lived access tokens, HttpOnly-cookie session restoration, bcrypt password hashing, trusted-origin checks, and role-ready authorization.
- React 19 with plain JavaScript/JSX, Vite, Tailwind, TanStack Query, Zustand, responsive navigation, and dark/light themes.
- Database-enforced 10-stock watchlist with add/remove/pause/resume controls, provider-backed NSE/BSE discovery, delayed fixture fallback, persisted source health, authenticated SSE updates, and an explainable fixture signal.
- Phase 2 RSS/Atom adapter, policy gate, durable cursors, retries, rate limits, circuit breaking, exact/near deduplication, stock links, Mongo-backed news APIs, Redis Streams outbox, dead-letter replay, saved parser fixtures, and a working News Explorer.
- Phase 3 deterministic local intelligence provider plus an opt-in approved HTTP JSON provider, language detection/translation boundary, entity linking, prompt/model/cost metadata, dual schema validation, numeric grounding, append-only analyses/evidence/signals, Redis consumer worker, and explainability UI.
- Phase 4 provider-neutral market service with a credential-free public-API adapter, current/day/52-week quote fields, period/unit/basis-aware fundamentals, OLA Electric identity and alias linking, sectors, breadth, movers, evidence-derived events, peers, risk flags, and a stock-detail UI with up to 10 deduplicated articles. Candles and charts are intentionally excluded from the revised scope.
- Phase 5 watchlist-scoped event and market-threshold rules, confidence/severity gates, cluster/rule/cooldown deduplication, quiet-hour deferral, user-isolated in-app/SSE delivery, inspectable evidence, pause/revoke/read controls, and morning/closing/daily briefings.
- Phase 6 event-time backtesting with explicit leakage rejection and sliced accuracy/precision, aggregate Prometheus metrics and trace correlation, SLO/dashboard assets, ingestion/SSE load tests, retention automation, CycloneDX SBOM/security checks, and native backup/restore/deployment runbooks.
- Versioned `signal-v2` scoring engine with evidence gates, contradiction handling, source citations, freshness, and immutable input snapshots.

The mock provider is intentional: no third-party site is scraped and no unlicensed market values are presented as live. See [assumptions and source licensing](docs/assumptions-and-sources.md).

## Run locally

On Windows/VS Code, the complete local stack now starts with only two terminals. The backend launcher imports `.env` once, starts `mongod`, starts Redis when `redis-server` is installed, builds the API/workers, and supervises them:

```powershell
.\run-backend.ps1
```

In the second terminal, start the web application:

```powershell
.\run-frontend.ps1
```

Open <http://localhost:5173>. If PowerShell blocks local scripts for the current process, first run `Set-ExecutionPolicy -Scope Process Bypass` in each terminal. Runtime data and logs stay in ignored `data/` and `.cache/runtime/` directories.

MongoDB must be installed with `mongod` available in `PATH`; the launcher creates `data/db` and starts it automatically when port 27017 is free. Redis on port 6379 is optional for the basic development UI. When `redis-server` is installed, the same backend command starts it and enables ingestion, analysis queues, distributed rate limiting and live events. The default `local-deterministic` AI provider is offline, costs nothing and sends no content externally.

Phase 4 also defaults to the offline fixture provider. To opt into the supplied experimental public endpoint, set `MARKET_PROVIDER=indian-stock-api` and explicitly set `MARKET_ALLOW_INSECURE_HTTP=true`. No API key is used or forwarded; only public symbols/search terms leave STOCKER. If the endpoint is unavailable, persisted snapshots remain labelled with their original age rather than being presented as fresh.

Phase 5 evaluates active alert rules every `ALERT_EVALUATION_INTERVAL` (one minute by default) and immediately after rule creation or resumption. In-app delivery is enabled; browser, email and Telegram preferences remain unavailable until the user explicitly consents to a destination and an operator configures a provider. No notification destination is collected by the current build.

For multiple approved news feeds, set `INGEST_SOURCES_JSON` to a JSON array following [the disabled candidate register](configs/news-sources.example.json). Every enabled source must first have `automatedAccessAllowed`, `robotsChecked`, attribution, licence, terms, rate, timeout, retention, and policy-expiry values reviewed. The worker gives every feed an independent schedule and cursor.

Open <http://localhost:5173>, register with a password of at least 12 characters, search for `OLAELEC`, `RELIANCE`, `HDFCBANK`, `INFY`, `TCS`, `ITC`, or `LT`, and add up to 10 securities. API docs are in [OpenAPI](docs/openapi.yaml).

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
$env:STOCKER_INTEGRATION_TEST='1'; go test ./internal/intelligence -run TestCollectedItemProducesPersistedAnalysisAndSignal -count=1 -v
$env:STOCKER_INTEGRATION_TEST='1'; go test ./internal/httpapi -run TestPhase4MarketAPIs -count=1 -v
$env:STOCKER_INTEGRATION_TEST='1'; go test ./internal/httpapi -run TestPhase5RuleTriggersOneInspectableUserIsolatedAlert -count=1 -v
$env:STOCKER_INTEGRATION_TEST='1'; go test ./internal/httpapi -run TestPhase6EventTimeEvaluationRejectsLeakedSignal -count=1 -v
./scripts/run-load.ps1 -BenchTime 5s
./scripts/security-check.ps1 -RequireExternalTools
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
- [Event-time model evaluation](docs/evaluation.md)
- [SLOs and alert policy](docs/slo.md)
- [Load and capacity testing](docs/load-testing.md)
- [Phase 6 security review](docs/security-review.md)

## Delivery phases

Phases 1–6 are implemented as working vertical slices under the revised no-candles/no-charts scope. Phase 3 can turn a collected, linked item into a schema-valid, source-cited analysis and append-only signal using the offline provider. Phase 4 defaults to transparent synthetic fixtures; the experimental public market adapter is opt-in because its upstream data rights and availability are not guaranteed. Phase 5 delivers evidence-backed in-app monitoring without collecting external notification destinations. Phase 6 adds repeatable evaluation, observability, load, security, lifecycle, recovery and native deployment evidence; external penetration testing and managed production services remain operator launch gates.
