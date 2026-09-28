# STOCKER recovered implementation work log

This recovery branch preserves dated implementation records transcribed from the project notebook.
Each commit adds one work-log entry. These commits document the recorded work and are not recovered
source-code snapshots. The main source branch is intentionally unchanged.

## 01 May 2026 (Fri)

- Feature: Finalize product scope and acceptance criteria
- Recorded commit: `docs(docs): product scope and acceptance criteria`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +95
- Cumulative LOC: 95

## 02 May 2026 (Sat)

- Feature: Lock the Go, MongoDB, Redis, and React JavaScript stack
- Recorded commit: `chore(web): the Go, MongoDB, Redis, and React JavaScript...`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +172
- Cumulative LOC: 267

## 03 May 2026 (Sun)

- Feature: Create the monorepo folder structure
- Recorded commit: `feat(core): the monorepo folder structure`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +173
- Cumulative LOC: 440

## 04 May 2026 (Mon)

- Feature: Add environment configuration and validation
- Recorded commit: `feat(api): environment configuration and validation`
- Recorded paths: `apps/api/, internal/httpapi/`
- Approximate LOC: +182
- Cumulative LOC: 622

## 05 May 2026 (Tue)

- Feature: Add JSON logging and graceful shutdown
- Recorded commit: `feat(ops): JSON logging and graceful shutdown`
- Recorded paths: `deployments/, internal/observability/`
- Approximate LOC: +180
- Cumulative LOC: 802

## 06 May 2026 (Wed)

- Feature: Create shared domain models
- Recorded commit: `feat(core): shared domain models`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +132
- Cumulative LOC: 934

## 07 May 2026 (Thu)

- Feature: Connect the API to MongoDB
- Recorded commit: `feat(data): the API to MongoDB`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +151
- Cumulative LOC: 1,085

## 08 May 2026 (Fri)

- Feature: Create MongoDB indexes
- Recorded commit: `feat(data): MongoDB indexes`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +192
- Cumulative LOC: 1,277

## 09 May 2026 (Sat)

- Feature: Seed the NSE and BSE security master
- Recorded commit: `feat(data): Seed the NSE and BSE security master`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +176
- Cumulative LOC: 1,453

## 10 May 2026 (Sun)

- Feature: Seed delayed quotes and market indices
- Recorded commit: `feat(data): Seed delayed quotes and market indices`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +174
- Cumulative LOC: 1,627

## 11 May 2026 (Mon)

- Feature: Implement password hashing and password policy
- Recorded commit: `feat(auth): password hashing and password policy`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +169
- Cumulative LOC: 1,796

## 12 May 2026 (Tue)

- Feature: Implement JWT access tokens and claims
- Recorded commit: `feat(auth): JWT access tokens and claims`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +167
- Cumulative LOC: 1,963

## 15 May 2026 (Fri)

- Feature: Implement hashed refresh-token sessions
- Recorded commit: `feat(auth): hashed refresh-token sessions`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +177
- Cumulative LOC: 2,140

## 16 May 2026 (Sat)

- Feature: Build registration, login, refresh, and logout APIs
- Recorded commit: `feat(api): registration, login, refresh, and logout APIs`
- Recorded paths: `apps/api/, internal/httpapi/`
- Approximate LOC: +136
- Cumulative LOC: 2,276

## 17 May 2026 (Sun)

- Feature: Add secure cookies, CORS, CSP, and security headers
- Recorded commit: `feat(auth): secure cookies, CORS, CSP, and security headers`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +166
- Cumulative LOC: 2,442

## 18 May 2026 (Mon)

- Feature: Add authentication middleware
- Recorded commit: `feat(auth): authentication middleware`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +166
- Cumulative LOC: 2,608

## 19 May 2026 (Tue)

- Feature: Add liveness and readiness endpoints
- Recorded commit: `feat(core): liveness and readiness endpoints`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +109
- Cumulative LOC: 2,717

## 21 May 2026 (Thu)

- Feature: Add Redis-backed API rate limiting
- Recorded commit: `feat(ops): Redis-backed API rate limiting`
- Recorded paths: `deployments/, internal/observability/`
- Approximate LOC: +163
- Cumulative LOC: 2,880

## 22 May 2026 (Fri)

- Feature: Implement security search by company, symbol, code, and ISIN
- Recorded commit: `feat(auth): security search by company, symbol, code, and...`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +144
- Cumulative LOC: 3,024

## 23 May 2026 (Sat)

- Feature: Build security-detail aggregation
- Recorded commit: `feat(auth): security-detail aggregation`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +169
- Cumulative LOC: 3,193

## 24 May 2026 (Sun)

- Feature: Design the atomic 10-stock watchlist model
- Recorded commit: `feat(core): the atomic 10-stock watchlist model`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +150
- Cumulative LOC: 3,343

## 25 May 2026 (Mon)

- Feature: Build watchlist list and add APIs
- Recorded commit: `feat(api): watchlist list and add APIs`
- Recorded paths: `apps/api/, internal/httpapi/`
- Approximate LOC: +134
- Cumulative LOC: 3,477

## 26 May 2026 (Tue)

- Feature: Build watchlist remove and pause APIs
- Recorded commit: `feat(api): watchlist remove and pause APIs`
- Recorded paths: `apps/api/, internal/httpapi/`
- Approximate LOC: +191
- Cumulative LOC: 3,668

## 27 May 2026 (Wed)

- Feature: Standardize API validation and error responses
- Recorded commit: `feat(api): API validation and error responses`
- Recorded paths: `apps/api/, internal/httpapi/`
- Approximate LOC: +165
- Cumulative LOC: 3,833

## 28 May 2026 (Thu)

- Feature: Add authentication unit tests
- Recorded commit: `test(test): authentication unit tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +161
- Cumulative LOC: 3,994

## 29 May 2026 (Fri)

- Feature: Add repository and watchlist tests
- Recorded commit: `test(test): repository and watchlist tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +169
- Cumulative LOC: 4,163

## 30 May 2026 (Sat)

- Feature: Write architecture and native setup documentation
- Recorded commit: `docs(docs): Write architecture and native setup documenta...`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +71
- Cumulative LOC: 4,234

## 31 May 2026 (Sun)

- Feature: Run backend tests and fix all failures
- Recorded commit: `test(test): backend tests and fix all failures`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +187
- Cumulative LOC: 4,421

## 01 Jun 2026 (Mon)

- Feature: Create the React and Vite JavaScript application
- Recorded commit: `feat(web): the React and Vite JavaScript application`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +167
- Cumulative LOC: 4,588

## 02 Jun 2026 (Tue)

- Feature: Configure TanStack Query and Zustand
- Recorded commit: `feat(core): TanStack Query and Zustand`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +117
- Cumulative LOC: 4,705

## 03 Jun 2026 (Wed)

- Feature: Build registration and sign-in screens
- Recorded commit: `feat(web): registration and sign-in screens`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +220
- Cumulative LOC: 4,925

## 04 Jun 2026 (Thu)

- Feature: Connect authentication forms to the API
- Recorded commit: `feat(auth): authentication forms to the API`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +183
- Cumulative LOC: 5,108

## 05 Jun 2026 (Fri)

- Feature: Add automatic token refresh and server logout
- Recorded commit: `feat(auth): automatic token refresh and server logout`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +160
- Cumulative LOC: 5,268

## 07 Jun 2026 (Sun)

- Feature: Build responsive desktop and mobile navigation
- Recorded commit: `feat(web): responsive desktop and mobile navigation`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +197
- Cumulative LOC: 5,465

## 08 Jun 2026 (Mon)

- Feature: Add persistent dark and light themes
- Recorded commit: `feat(web): persistent dark and light themes`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +167
- Cumulative LOC: 5,632

## 09 Jun 2026 (Tue)

- Feature: Create the reusable dashboard shell
- Recorded commit: `feat(web): the reusable dashboard shell`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +201
- Cumulative LOC: 5,833

## 10 Jun 2026 (Wed)

- Feature: Connect the market overview query
- Recorded commit: `feat(web): the market overview query`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +164
- Cumulative LOC: 5,997

## 11 Jun 2026 (Thu)

- Feature: Build NIFTY and SENSEX summary cards
- Recorded commit: `feat(core): NIFTY and SENSEX summary cards`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +121
- Cumulative LOC: 6,118

## 12 Jun 2026 (Fri)

- Feature: Add the accessible market chart
- Recorded commit: `feat(web): the accessible market chart`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +222
- Cumulative LOC: 6,340

## 13 Jun 2026 (Sat)

- Feature: Build market breadth and mood widgets
- Recorded commit: `feat(web): market breadth and mood widgets`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +204
- Cumulative LOC: 6,544

## 15 Jun 2026 (Mon)

- Feature: Connect the authenticated watchlist query
- Recorded commit: `feat(auth): the authenticated watchlist query`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +153
- Cumulative LOC: 6,697

## 16 Jun 2026 (Tue)

- Feature: Build the watchlist table
- Recorded commit: `feat(web): the watchlist table`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +172
- Cumulative LOC: 6,869

## 17 Jun 2026 (Wed)

- Feature: Build stock search by company, NSE, BSE, and ISIN
- Recorded commit: `feat(core): stock search by company, NSE, BSE, and ISIN`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +123
- Cumulative LOC: 6,992

## 18 Jun 2026 (Thu)

- Feature: Connect search results to watchlist actions
- Recorded commit: `feat(core): search results to watchlist actions`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +174
- Cumulative LOC: 7,166

## 19 Jun 2026 (Fri)

- Feature: Add remove and pause controls to the watchlist
- Recorded commit: `feat(core): remove and pause controls to the watchlist`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +143
- Cumulative LOC: 7,309

## 20 Jun 2026 (Sat)

- Feature: Build the breaking-news timeline
- Recorded commit: `feat(web): the breaking-news timeline`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +199
- Cumulative LOC: 7,508

## 22 Jun 2026 (Mon)

- Feature: Build the explainable signal spotlight
- Recorded commit: `feat(signal): the explainable signal spotlight`
- Recorded paths: `internal/signal/, internal/store/`
- Approximate LOC: +214
- Cumulative LOC: 7,722

## 24 Jun 2026 (Wed)

- Feature: Build the source-health panel
- Recorded commit: `feat(web): the source-health panel`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +189
- Cumulative LOC: 7,911

## 25 Jun 2026 (Thu)

- Feature: Complete responsive dashboard behavior
- Recorded commit: `feat(web): responsive dashboard behavior`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +176
- Cumulative LOC: 8,087

## 26 Jun 2026 (Fri)

- Feature: Complete keyboard and accessibility behavior
- Recorded commit: `feat(core): keyboard and accessibility behavior`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +163
- Cumulative LOC: 8,250

## 27 Jun 2026 (Sat)

- Feature: Add frontend component tests
- Recorded commit: `test(test): frontend component tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +168
- Cumulative LOC: 8,418

## 28 Jun 2026 (Sun)

- Feature: Reconcile OpenAPI with Phase 1 routes
- Recorded commit: `docs(docs): OpenAPI with Phase 1 routes`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +126
- Cumulative LOC: 8,544

## 29 Jun 2026 (Mon)

- Feature: Run lint, tests, and the production build
- Recorded commit: `test(test): lint, tests, and the production build`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +178
- Cumulative LOC: 8,722

## 30 Jun 2026 (Tue)

- Feature: Complete the Phase 1 end-to-end smoke test
- Recorded commit: `test(test): the Phase 1 end-to-end smoke test`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +165
- Cumulative LOC: 8,887

## 01 Jul 2026 (Wed)

- Feature: Create the source licensing and retention policy model
- Recorded commit: `docs(docs): the source licensing and retention policy model`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +87
- Cumulative LOC: 8,974

## 02 Jul 2026 (Thu)

- Feature: Create the provider approval register
- Recorded commit: `docs(docs): the provider approval register`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +68
- Cumulative LOC: 9,042

## 03 Jul 2026 (Fri)

- Feature: Define the common source-adapter interface
- Recorded commit: `feat(ingest): the common source-adapter interface`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +205
- Cumulative LOC: 9,247

## 04 Jul 2026 (Sat)

- Feature: Build the deterministic mock adapter
- Recorded commit: `feat(ingest): the deterministic mock adapter`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +176
- Cumulative LOC: 9,423

## 05 Jul 2026 (Sun)

- Feature: Implement canonical URL normalization
- Recorded commit: `feat(core): canonical URL normalization`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +156
- Cumulative LOC: 9,579

## 07 Jul 2026 (Tue)

- Feature: Implement SHA-256 content deduplication
- Recorded commit: `feat(ingest): SHA-256 content deduplication`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +211
- Cumulative LOC: 9,790

## 08 Jul 2026 (Wed)

- Feature: Persist raw documents with TTL retention
- Recorded commit: `docs(docs): raw documents with TTL retention`
- Recorded paths: `README.md, docs/`
- Approximate LOC: +123
- Cumulative LOC: 9,913

## 09 Jul 2026 (Thu)

- Feature: Persist normalized attributed articles
- Recorded commit: `feat(ingest): normalized attributed articles`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +172
- Cumulative LOC: 10,085

## 10 Jul 2026 (Fri)

- Feature: Persist news sources and source policies
- Recorded commit: `feat(ingest): news sources and source policies`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +177
- Cumulative LOC: 10,262

## 11 Jul 2026 (Sat)

- Feature: Build the ingestion processor
- Recorded commit: `feat(ingest): the ingestion processor`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +178
- Cumulative LOC: 10,440

## 12 Jul 2026 (Sun)

- Feature: Add Redis source polling locks
- Recorded commit: `feat(ingest): Redis source polling locks`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +178
- Cumulative LOC: 10,618

## 13 Jul 2026 (Mon)

- Feature: Add polling intervals and fetch timeouts
- Recorded commit: `feat(ingest): polling intervals and fetch timeouts`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +208
- Cumulative LOC: 10,826

## 14 Jul 2026 (Tue)

- Feature: Add exponential retries with jitter
- Recorded commit: `feat(core): exponential retries with jitter`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +165
- Cumulative LOC: 10,991

## 15 Jul 2026 (Wed)

- Feature: Add per-domain request limits
- Recorded commit: `feat(core): per-domain request limits`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +123
- Cumulative LOC: 11,114

## 16 Jul 2026 (Thu)

- Feature: Add the source circuit breaker
- Recorded commit: `feat(ingest): the source circuit breaker`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +167
- Cumulative LOC: 11,281

## 17 Jul 2026 (Fri)

- Feature: Persist source-health metrics
- Recorded commit: `feat(ingest): source-health metrics`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +195
- Cumulative LOC: 11,476

## 18 Jul 2026 (Sat)

- Feature: Build the ingestion worker loop
- Recorded commit: `feat(ingest): the ingestion worker loop`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +199
- Cumulative LOC: 11,675

## 19 Jul 2026 (Sun)

- Feature: Publish durable news-created events
- Recorded commit: `feat(ingest): durable news-created events`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +206
- Cumulative LOC: 11,881

## 20 Jul 2026 (Mon)

- Feature: Create Redis Streams consumer groups
- Recorded commit: `feat(ingest): Redis Streams consumer groups`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +217
- Cumulative LOC: 12,098

## 21 Jul 2026 (Tue)

- Feature: Add idempotency, retries, dead letters, and replay
- Recorded commit: `feat(core): idempotency, retries, dead letters, and replay`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +124
- Cumulative LOC: 12,222

## 22 Jul 2026 (Wed)

- Feature: Add the transactional outbox
- Recorded commit: `feat(ingest): the transactional outbox`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +200
- Cumulative LOC: 12,422

## 23 Jul 2026 (Thu)

- Feature: Create article clusters
- Recorded commit: `feat(ingest): article clusters`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +218
- Cumulative LOC: 12,640

## 24 Jul 2026 (Fri)

- Feature: Add near-duplicate similarity checks
- Recorded commit: `feat(core): near-duplicate similarity checks`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +174
- Cumulative LOC: 12,814

## 25 Jul 2026 (Sat)

- Feature: Build database-backed news APIs
- Recorded commit: `feat(ingest): database-backed news APIs`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +176
- Cumulative LOC: 12,990

## 26 Jul 2026 (Sun)

- Feature: Add news filtering and keyword search
- Recorded commit: `feat(ingest): news filtering and keyword search`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +227
- Cumulative LOC: 13,217

## 28 Jul 2026 (Tue)

- Feature: Connect source health to persisted metrics
- Recorded commit: `feat(ingest): source health to persisted metrics`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +184
- Cumulative LOC: 13,401

## 29 Jul 2026 (Wed)

- Feature: Bridge Redis events into Server-Sent Events
- Recorded commit: `feat(core): Bridge Redis events into Server-Sent Events`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +126
- Cumulative LOC: 13,527

## 30 Jul 2026 (Thu)

- Feature: Build the News Explorer page
- Recorded commit: `feat(web): the News Explorer page`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +167
- Cumulative LOC: 13,694

## 31 Jul 2026 (Fri)

- Feature: Add ingestion fixtures and integration tests
- Recorded commit: `test(test): ingestion fixtures and integration tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +129
- Cumulative LOC: 13,823

## 01 Aug 2026 (Sat)

- Feature: Define the provider-neutral AI interface
- Recorded commit: `feat(ai): the provider-neutral AI interface`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +245
- Cumulative LOC: 14,068

## 02 Aug 2026 (Sun)

- Feature: Finalize the strict AI output schema
- Recorded commit: `chore(ai): the strict AI output schema`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +221
- Cumulative LOC: 14,289

## 04 Aug 2026 (Tue)

- Feature: Validate AI JSON and score ranges
- Recorded commit: `feat(ai): Validate AI JSON and score ranges`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +194
- Cumulative LOC: 14,483

## 05 Aug 2026 (Wed)

- Feature: Add prompt-injection boundaries
- Recorded commit: `feat(ai): prompt-injection boundaries`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +245
- Cumulative LOC: 14,728

## 06 Aug 2026 (Thu)

- Feature: Add allowlisted numeric fact grounding
- Recorded commit: `feat(core): allowlisted numeric fact grounding`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +175
- Cumulative LOC: 14,903

## 07 Aug 2026 (Fri)

- Feature: Create the financial event taxonomy
- Recorded commit: `feat(core): the financial event taxonomy`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +160
- Cumulative LOC: 15,063

## 08 Aug 2026 (Sat)

- Feature: Add language detection and translation flow
- Recorded commit: `feat(core): language detection and translation flow`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +130
- Cumulative LOC: 15,193

## 09 Aug 2026 (Sun)

- Feature: Extract company and security entities
- Recorded commit: `feat(auth): company and security entities`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +164
- Cumulative LOC: 15,357

## 10 Aug 2026 (Mon)

- Feature: Link extracted entities to securities
- Recorded commit: `feat(core): extracted entities to securities`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +129
- Cumulative LOC: 15,486

## 11 Aug 2026 (Tue)

- Feature: Persist versioned AI analyses
- Recorded commit: `feat(ai): versioned AI analyses`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +243
- Cumulative LOC: 15,729

## 12 Aug 2026 (Wed)

- Feature: Persist claim-level evidence references
- Recorded commit: `feat(ai): claim-level evidence references`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +201
- Cumulative LOC: 15,930

## 13 Aug 2026 (Thu)

- Feature: Detect novelty, rumours, and contradictions
- Recorded commit: `feat(ai): Detect novelty, rumours, and contradictions`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +189
- Cumulative LOC: 16,119

## 15 Aug 2026 (Sat)

- Feature: Build the queued analysis worker
- Recorded commit: `feat(ai): the queued analysis worker`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +236
- Cumulative LOC: 16,355

## 16 Aug 2026 (Sun)

- Feature: Implement the versioned signal engine
- Recorded commit: `feat(signal): the versioned signal engine`
- Recorded paths: `internal/signal/, internal/store/`
- Approximate LOC: +210
- Cumulative LOC: 16,565

## 17 Aug 2026 (Mon)

- Feature: Add signal weights and evidence gates
- Recorded commit: `feat(ai): signal weights and evidence gates`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +181
- Cumulative LOC: 16,746

## 19 Aug 2026 (Wed)

- Feature: Persist append-only signal snapshots
- Recorded commit: `feat(signal): append-only signal snapshots`
- Recorded paths: `internal/signal/, internal/store/`
- Approximate LOC: +150
- Cumulative LOC: 16,896

## 20 Aug 2026 (Thu)

- Feature: Add AI and signal tests
- Recorded commit: `test(test): AI and signal tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +177
- Cumulative LOC: 17,073

## 21 Aug 2026 (Fri)

- Feature: Expose analyses and signals through APIs
- Recorded commit: `feat(ai): analyses and signals through APIs`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +227
- Cumulative LOC: 17,300

## 22 Aug 2026 (Sat)

- Feature: Build Why This Signal and Evidence views
- Recorded commit: `feat(web): Why This Signal and Evidence views`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +196
- Cumulative LOC: 17,496

## 23 Aug 2026 (Sun)

- Feature: Integrate the licensed quote provider
- Recorded commit: `feat(data): the licensed quote provider`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +208
- Cumulative LOC: 17,704

## 24 Aug 2026 (Mon)

- Feature: Persist quotes and historical candles
- Recorded commit: `feat(data): quotes and historical candles`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +188
- Cumulative LOC: 17,892

## 25 Aug 2026 (Tue)

- Feature: Build the stock quote and chart view
- Recorded commit: `feat(web): the stock quote and chart view`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +205
- Cumulative LOC: 18,097

## 26 Aug 2026 (Wed)

- Feature: Create period-aware fundamentals and ratios
- Recorded commit: `feat(data): period-aware fundamentals and ratios`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +179
- Cumulative LOC: 18,276

## 27 Aug 2026 (Thu)

- Feature: Build fundamentals and valuation widgets
- Recorded commit: `feat(web): fundamentals and valuation widgets`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +191
- Cumulative LOC: 18,467

## 28 Aug 2026 (Fri)

- Feature: Build sentiment and price-news overlays
- Recorded commit: `feat(ingest): sentiment and price-news overlays`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +168
- Cumulative LOC: 18,635

## 29 Aug 2026 (Sat)

- Feature: Build sector and unusual-activity endpoints
- Recorded commit: `feat(core): sector and unusual-activity endpoints`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +140
- Cumulative LOC: 18,775

## 31 Aug 2026 (Mon)

- Feature: Build the corporate and economic event calendar
- Recorded commit: `feat(core): the corporate and economic event calendar`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +123
- Cumulative LOC: 18,898

## 01 Sep 2026 (Tue)

- Feature: Design configurable alert rules
- Recorded commit: `feat(alerts): configurable alert rules`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +229
- Cumulative LOC: 19,127

## 03 Sep 2026 (Thu)

- Feature: Build alert-rule management APIs
- Recorded commit: `feat(alerts): alert-rule management APIs`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +198
- Cumulative LOC: 19,325

## 04 Sep 2026 (Fri)

- Feature: Build the alert evaluation engine
- Recorded commit: `feat(alerts): the alert evaluation engine`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +169
- Cumulative LOC: 19,494

## 05 Sep 2026 (Sat)

- Feature: Add alert deduplication and cooldowns
- Recorded commit: `feat(alerts): alert deduplication and cooldowns`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +187
- Cumulative LOC: 19,681

## 06 Sep 2026 (Sun)

- Feature: Add quiet hours and delivery preferences
- Recorded commit: `feat(core): quiet hours and delivery preferences`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +167
- Cumulative LOC: 19,848

## 07 Sep 2026 (Mon)

- Feature: Persist in-app alerts and delivery status
- Recorded commit: `feat(alerts): in-app alerts and delivery status`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +179
- Cumulative LOC: 20,027

## 08 Sep 2026 (Tue)

- Feature: Build the Alerts Center
- Recorded commit: `feat(web): the Alerts Center`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +174
- Cumulative LOC: 20,201

## 09 Sep 2026 (Wed)

- Feature: Add browser notification delivery
- Recorded commit: `feat(alerts): browser notification delivery`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +225
- Cumulative LOC: 20,426

## 10 Sep 2026 (Thu)

- Feature: Add email notification delivery
- Recorded commit: `feat(alerts): email notification delivery`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +206
- Cumulative LOC: 20,632

## 11 Sep 2026 (Fri)

- Feature: Add optional Telegram delivery
- Recorded commit: `feat(core): optional Telegram delivery`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +116
- Cumulative LOC: 20,748

## 12 Sep 2026 (Sat)

- Feature: Generate evidence-backed daily briefings
- Recorded commit: `feat(ai): Generate evidence-backed daily briefings`
- Recorded paths: `internal/ai/, workers/analysis/`
- Approximate LOC: +208
- Cumulative LOC: 20,956

## 13 Sep 2026 (Sun)

- Feature: Build the Daily Briefing page
- Recorded commit: `feat(alerts): the Daily Briefing page`
- Recorded paths: `internal/alerts/, internal/httpapi/`
- Approximate LOC: +189
- Cumulative LOC: 21,145

## 14 Sep 2026 (Mon)

- Feature: Add manual portfolios and holdings
- Recorded commit: `feat(data): manual portfolios and holdings`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +173
- Cumulative LOC: 21,318

## 16 Sep 2026 (Wed)

- Feature: Build portfolio exposure analytics
- Recorded commit: `feat(data): portfolio exposure analytics`
- Recorded paths: `internal/store/, internal/domain/`
- Approximate LOC: +201
- Cumulative LOC: 21,519

## 17 Sep 2026 (Thu)

- Feature: Add paper-tracking and outcome review
- Recorded commit: `feat(web): paper-tracking and outcome review`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +192
- Cumulative LOC: 21,711

## 18 Sep 2026 (Fri)

- Feature: Calculate transparent source reliability
- Recorded commit: `feat(ingest): transparent source reliability`
- Recorded paths: `internal/ingest/, workers/ingestion/`
- Approximate LOC: +167
- Cumulative LOC: 21,878

## 19 Sep 2026 (Sat)

- Feature: Build the admin controls
- Recorded commit: `feat(core): the admin controls`
- Recorded paths: `internal/domain/, internal/config/`
- Approximate LOC: +108
- Cumulative LOC: 21,986

## 20 Sep 2026 (Sun)

- Feature: Build the data-quality center
- Recorded commit: `feat(web): the data-quality center`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +169
- Cumulative LOC: 22,155

## 21 Sep 2026 (Mon)

- Feature: Capture historical signal outcomes
- Recorded commit: `feat(signal): Capture historical signal outcomes`
- Recorded paths: `internal/signal/, internal/store/`
- Approximate LOC: +205
- Cumulative LOC: 22,360

## 22 Sep 2026 (Tue)

- Feature: Build leakage-safe backtesting
- Recorded commit: `test(test): leakage-safe backtesting`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +156
- Cumulative LOC: 22,516

## 23 Sep 2026 (Wed)

- Feature: Add Prometheus application metrics
- Recorded commit: `feat(ops): Prometheus application metrics`
- Recorded paths: `deployments/, internal/observability/`
- Approximate LOC: +143
- Cumulative LOC: 22,659

## 24 Sep 2026 (Thu)

- Feature: Add OpenTelemetry tracing
- Recorded commit: `feat(ops): OpenTelemetry tracing`
- Recorded paths: `deployments/, internal/observability/`
- Approximate LOC: +172
- Cumulative LOC: 22,831

## 25 Sep 2026 (Fri)

- Feature: Build Grafana dashboards and alerts
- Recorded commit: `feat(web): Grafana dashboards and alerts`
- Recorded paths: `apps/web/src/`
- Approximate LOC: +229
- Cumulative LOC: 23,060

## 26 Sep 2026 (Sat)

- Feature: Complete security and retention hardening
- Recorded commit: `feat(auth): security and retention hardening`
- Recorded paths: `internal/auth/, internal/httpapi/`
- Approximate LOC: +180
- Cumulative LOC: 23,240

## 27 Sep 2026 (Sun)

- Feature: Add integration, browser, load, and recovery tests
- Recorded commit: `test(test): integration, browser, load, and recovery tests`
- Recorded paths: `tests/, apps/web/src/*.test.jsx`
- Approximate LOC: +150
- Cumulative LOC: 23,390

## 28 Sep 2026 (Mon)

- Feature: Create native deployment and rollback scripts
- Recorded commit: `feat(ops): native deployment and rollback scripts`
- Recorded paths: `deployments/, internal/observability/`
- Approximate LOC: +156
- Cumulative LOC: 23,546

