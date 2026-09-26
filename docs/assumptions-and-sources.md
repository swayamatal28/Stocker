# Assumptions and source/licensing register

## Product assumptions

1. The product name is **STOCKER** everywhere in code, copy, deployment, and documentation.
2. UTC is stored in every service and database timestamp. The UI defaults to `Asia/Kolkata` (IST) for presentation.
3. “Live” means near-real-time only within the upstream provider's expressly permitted polling interval. The UI labels delayed and mock values; it never implies tick-level delivery.
4. The first release is research-only: no brokerage connection, order routing, execution, or guaranteed language.
5. Each user has one default watchlist capped at 10 securities. A unique MongoDB index and atomic conditional update are the final enforcement layers.
6. No production AI call occurs without a configured provider, schema validation, evidence persistence, and budget limit. Local development uses deterministic fixtures.
7. MongoDB text indexes are the initial search engine. An Atlas Search/vector integration is optional and must not become the only way evidence is found.

## Source register and permissions needed

Publicly viewable does not automatically mean permitted for automated collection or redistribution. Before enabling any adapter, the operator must record terms, licence, attribution, retention, redistribution, rate limits, and legal review in `source_policies`.

| Source class | Phase-1 state | Requirement before production use |
|---|---|---|
| NSE corporate announcements/filings | Adapter ready; production feed disabled | NSE publishes an RSS directory, but its general Terms of Use prohibit systematic automated collection and storage/redistribution without permission. Use only under a specific feed/data agreement that expressly covers STOCKER. Confirm attachment retention and redistribution rights. |
| BSE corporate announcements/filings | Disabled | Review current BSE website/API terms; obtain an authorised feed/API or explicit permission. Confirm rate and redistribution terms. |
| Licensed quotes/candles | Mock only | Contract with a SEBI/exchange-authorised vendor as applicable. API key, display entitlement, delay label, cache/retention rights, and audit requirements are provider-specific. |
| Fundamentals and shareholding | Seed metadata only | Licensed provider or permitted company/exchange filings. Preserve reporting period, standalone/consolidated basis, units, source, and `as_of`. |
| News APIs | Disabled | Subscription/API credentials plus rights for headline/full-text storage, summarisation, display, retention, and derived analytics. |
| RSS/Atom feeds | Adapter boundary only | Feed must explicitly permit automated consumption for this use. Store attribution and link to the publisher; do not assume full-text reuse rights. |
| Public HTML pages | No site adapter | Written/contractual permission or terms that explicitly allow automation; fresh robots review; conservative domain rate; no login, CAPTCHA, paywall, or access-control bypass. |
| Hosted AI model | Disabled; generic HTTP JSON adapter implemented | API credential, DPA/data residency review, prompt/content retention settings, cost ceiling, model/version allowlist. |
| Local deterministic intelligence | Enabled for development and tests | Project-owned rules, offline execution, zero external data transfer, schema/grounding/adversarial regression tests. |
| Email/Telegram/browser push | Interface/data model only | Provider credentials, consent, unsubscribe/revocation flow, destination minimisation and redaction. |

## Production onboarding gate for a source

- Record owner, legal basis, terms URL/version, licence, permitted fields, attribution text, polling rate, retention, and redistribution rules.
- Use the most official/licensed source available. Official does **not** remove the need to comply with access and data-use terms.
- Obtain credentials through a secrets manager; never commit or log them.
- Test against saved fixtures. Live contract tests run only in an isolated, rate-limited environment.
- Configure timeout, exponential backoff with jitter, circuit breaker, per-domain limiter, parser version, health SLO, and deletion job.
- Disable automatically after repeated parser failures or a policy expiry. Do not silently fall back to a less-authoritative scraper.

## Current Phase 2 source decision (2026-09-24)

The project-owned `mock-exchange` fixture remains the only enabled source by default. A generic RSS/Atom adapter is implemented and tested against a saved fixture, but `MOCK_PROVIDERS=false` will still fail closed unless `INGEST_AUTOMATED_ACCESS_ALLOWED=true` and all source-policy environment fields are supplied.

NSE's official RSS directory describes feeds intended for feed-reader subscription: <https://www.nseindia.com/static/rss-feed>. However, NSE's current general Terms of Use prohibit systematic or automated collection and restrict copying, storage, aggregation, and redistribution without prior written permission: <https://www.nseindia.com/static/nse-terms-of-use>. NSE's data policy also requires a relevant agreement governing access, handling, display, and redistribution: <https://www.nseindia.com/static/market-data/nse-data-policy>. For that reason, STOCKER does not ship an enabled NSE URL or claim permission based only on the existence of RSS links.

## Phase 1 security-master decision

Phase 1 deliberately supports only the six project-owned fixture records (`RELIANCE`, `HDFCBANK`, `INFY`, `TCS`, `ITC`, and `LT`). They exist to verify NSE/BSE symbol and ISIN search plus watchlist workflows; they are not represented as a licensed or complete exchange security master.

Expanding beyond those fixtures is a production data-source decision, not a code default. The operator must select a licensed or expressly permitted security-master provider, record its display/retention/redistribution rights in `source_policies`, and add dated contract tests before importing its universe. Until that approval exists, STOCKER fails closed at the six fixture records.

## Phase 3 model-provider decision

Development defaults to `local-deterministic`, a project-owned offline rules provider. It performs conservative event/sentiment/materiality classification, incurs no model cost, and never transmits source text. This is the enabled Phase 3 provider and the reproducible test oracle; it is not represented as a general-purpose language model.

An `http-json` adapter is available for an approved hosted or local model endpoint. It remains opt-in and requires explicit endpoint/model/key configuration, bounded time/output, a per-request cost estimate, and the daily budget ceiling. Before production use, record the provider/model allowlist, DPA, residency/retention settings, model licence, pricing, evaluation results and incident owner. Non-English translation fails closed unless the selected provider implements the translation contract.
