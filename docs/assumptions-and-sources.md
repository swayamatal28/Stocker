# Assumptions and source/licensing register

## Product assumptions

1. The product name is **STOCKER** everywhere in code, copy, deployment, and documentation.
2. UTC is stored in every service and database timestamp. The UI defaults to `Asia/Kolkata` (IST) for presentation.
3. “Live” means near-real-time only within the upstream provider's polling interval. Quote timestamps and delay labels remain visible; the product never implies tick-level delivery.
4. The first release is research-only: no brokerage connection, order routing, execution, or guaranteed language.
5. Each user has one manually entered portfolio capped at 50 holdings. Quantity and average buy price are required for new holdings; a unique MongoDB index and atomic conditional update are the final enforcement layers.
6. No hosted AI call occurs without a configured provider, schema validation, evidence persistence, and budget limit. The default deterministic provider analyzes real collected source text locally.
7. MongoDB text indexes are the initial search engine. An Atlas Search/vector integration is optional and must not become the only way evidence is found.

## Source register and permissions needed

Publicly viewable does not automatically mean permitted for automated collection or redistribution. Before enabling any adapter, the operator must record terms, licence, attribution, retention, redistribution, rate limits, and legal review in `source_policies`.

| Source class | Phase-1 state | Requirement before production use |
|---|---|---|
| NSE corporate announcements/filings | Adapter ready; production feed disabled | NSE publishes an RSS directory, but its general Terms of Use prohibit systematic automated collection and storage/redistribution without permission. Use only under a specific feed/data agreement that expressly covers STOCKER. Confirm attachment retention and redistribution rights. |
| BSE corporate announcements/filings | Disabled | Review current BSE website/API terms; obtain an authorised feed/API or explicit permission. Confirm rate and redistribution terms. |
| Current quote/range snapshots | Credential-free Yahoo Finance adapter enabled | The free endpoint does not establish redistribution rights or an uptime SLA. For production, contract with an authorised vendor as applicable and record display entitlement, delay label, cache/retention rights, and audit requirements. Candles/charts are outside scope. |
| Fundamentals and shareholding | Yahoo Finance fundamental time series enabled; shareholding unavailable | Treat the credential-free endpoint as best-effort. For production, use a licensed provider or permitted company/exchange filings. Preserve reporting period, basis, units, source, and `as_of`. |
| News APIs | Disabled | Subscription/API credentials plus rights for headline/full-text storage, summarisation, display, retention, and derived analytics. |
| RSS/Atom feeds | Six publisher feeds configured | Store only feed-provided headlines/summaries, preserve attribution and canonical links, and re-review publisher terms/robots policy before the recorded policy expiry. |
| Public HTML pages | No site adapter | Written/contractual permission or terms that explicitly allow automation; fresh robots review; conservative domain rate; no login, CAPTCHA, paywall, or access-control bypass. |
| Hosted AI model | Disabled; generic HTTP JSON adapter implemented | API credential, DPA/data residency review, prompt/content retention settings, cost ceiling, model/version allowlist. |
| Local deterministic intelligence | Enabled for live local analysis | Project-owned rules, offline execution, zero external data transfer, schema/grounding/adversarial regression tests. |
| Email/Telegram/browser push | Interface/data model only | Provider credentials, consent, unsubscribe/revocation flow, destination minimisation and redaction. |

## Production onboarding gate for a source

- Record owner, legal basis, terms URL/version, licence, permitted fields, attribution text, polling rate, retention, and redistribution rules.
- Use the most official/licensed source available. Official does **not** remove the need to comply with access and data-use terms.
- Obtain credentials through a secrets manager; never commit or log them.
- Use deterministic unit tests plus the explicit opt-in live contract test; never make routine tests depend on publisher availability.
- Configure timeout, exponential backoff with jitter, circuit breaker, per-domain limiter, parser version, health SLO, and deletion job.
- Disable automatically after repeated parser failures or a policy expiry. Do not silently fall back to a less-authoritative scraper.

## Current Phase 2 source decision (2026-09-27)

Runtime mock ingestion is disabled. `configs/news-sources.live.json` enables six rate-limited publisher RSS feeds whose endpoints and robots files were checked on 2026-09-27. STOCKER reads only the feed payload, stores its attribution and canonical link, and does not crawl article pages, paywalls, logins, or access controls. Operators remain responsible for rechecking terms and permissions before the recorded policy expiry.

NSE's official RSS directory describes feeds intended for feed-reader subscription: <https://www.nseindia.com/static/rss-feed>. However, NSE's current general Terms of Use prohibit systematic or automated collection and restrict copying, storage, aggregation, and redistribution without prior written permission: <https://www.nseindia.com/static/nse-terms-of-use>. NSE's data policy also requires a relevant agreement governing access, handling, display, and redistribution: <https://www.nseindia.com/static/market-data/nse-data-policy>. For that reason, STOCKER does not ship an enabled NSE URL or claim permission based only on the existence of RSS links.

## Phase 1 security-master decision

Runtime security seeding is disabled. Search queries Yahoo Finance's public search endpoint for NSE equities and persists the returned provider attribution. Existing project-seeded identity records may remain only as accurate reference identities; their synthetic quotes/signals are purged, and a live search or quote refresh replaces their source metadata.

## Phase 3 model-provider decision

Development defaults to `local-deterministic`, a project-owned offline rules provider. It performs conservative event/sentiment/materiality classification, incurs no model cost, and never transmits source text. This is the enabled Phase 3 provider and the reproducible test oracle; it is not represented as a general-purpose language model.

An `http-json` adapter is available for an approved hosted or local model endpoint. It remains opt-in and requires explicit endpoint/model/key configuration, bounded time/output, a per-request cost estimate, and the daily budget ceiling. Before production use, record the provider/model allowlist, DPA, residency/retention settings, model licence, pricing, evaluation results and incident owner. Non-English translation fails closed unless the selected provider implements the translation contract.

Portfolio AI advice uses the same explicit opt-in boundary. The request contains only the selected public symbol/company name, current public quote, public fundamentals, deterministic research checks, and recent public-news titles/source/date. It excludes quantity, average buy price, profit/loss, user/session identifiers, cookies, environment values and all other secrets. A missing, placeholder or malformed key/endpoint fails closed and the UI states that AI advice is unavailable.

News and its derived impact data are transient. Items at least 24 hours old are rejected during ingestion, excluded from reads and removed with their linked analyses/signals by maintenance. Portfolio holdings and essential identity/configuration records are not subject to this transient deletion.

## Phase 4 market and news-source decision (2026-09-27)

The revised Phase 4 scope excludes candles and charts. STOCKER now stores current price, change, open, day/52-week ranges, volume and point-in-time ratios with separate market `as_of` and retrieval timestamps plus source, delay, unit, period and basis metadata.

`MARKET_PROVIDER=yahoo-finance` enables credential-free NSE search, delayed quote snapshots and company fundamental time series. The adapter has no mechanism to forward API keys, cookies, authorization headers, user IDs, portfolios, article text, prompts or environment values; only a public symbol or search term is transmitted. The previously supplied plaintext service at `65.0.104.9` timed out during the 2026-09-27 check and is not the active provider.

Multi-source RSS/Atom ingestion is loaded from `INGEST_SOURCES_FILE=configs/news-sources.live.json`, with independent attribution, terms URL, rate, timeout, retention period and expiry for every source. Screener is not included because its published service licence limits material to personal, non-commercial transitory viewing. Zee Business is not enabled because its feed returned HTTP 403 to the worker user agent during the current validation.

## Phase 5 delivery decision

The first active notification channel is in-app delivery over the authenticated, user-targeted SSE stream. Browser push, email and Telegram flags are modelled as per-rule preferences but remain `not_configured`; the application does not collect an address, device subscription or Telegram identifier until an operator adds a consented provider and deletion/revocation workflow.

The watchlist has been replaced by a manual portfolio. STOCKER stores only the stock, quantity, average buy price and alert preference; it does not connect to a broker, import transactions or place orders. Current value, total profit/loss and daily profit/loss are calculated from the latest timestamped quote.

## Portfolio research decision (2026-09-27)

Portfolio stock research is deterministic and does not use machine learning. It evaluates only available published figures using visible formulas for P/E, return on equity, debt/equity, current ratio, operating margin, revenue growth, free cash flow and position in the 52-week range. Missing figures are omitted rather than estimated, and the result is explicitly informational rather than a buy/sell recommendation.

Screener is not scraped or mirrored. Its published terms limit use and copying, so STOCKER uses the enabled market provider's fundamental time series and keeps provider attribution and reporting dates. A future operator may add Screener only under a licence that expressly permits automated access, storage and display.
