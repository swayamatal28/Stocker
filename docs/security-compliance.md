# Security and compliance checklist

## Implemented in the foundation

- [x] 12-character minimum and bcrypt cost 12 password hashing.
- [x] 15-minute signed access tokens; 30-day opaque refresh tokens stored only as SHA-256 hashes and rotated on use.
- [x] HttpOnly, Strict SameSite refresh cookie; Secure required by production configuration.
- [x] Access tokens are held only in browser memory; a single-flight refresh restores sessions and retries expired authenticated requests.
- [x] Refresh/logout reject mismatched browser origins, and logout revokes the server-side refresh session.
- [x] Structured MongoDB filters, escaped user search patterns, and Gin input validation.
- [x] Redis-backed IP rate limiter; atomic database watchlist constraint.
- [x] Authenticated SSE endpoint, Redis Streams bridge, bounded subscriber buffers, dropped-event metrics and reconnecting web client.
- [x] Restrictive API CSP, frame denial, MIME sniffing and referrer protections. The production static host must add a web-app CSP.
- [x] JSON structured logs without request bodies, credentials, tokens, passwords or notification destinations.
- [x] Liveness/readiness, bounded server timeouts, circuit-ready source design and graceful shutdown.
- [x] Secrets are environment-only; `.env` is ignored.
- [x] Immutable evidence/signal snapshots and audit-log schema.
- [x] Source text is isolated as untrusted data; provider output must pass embedded JSON Schema, code validation, linked-symbol checks, exact-excerpt checks and numeric grounding.
- [x] AI provider calls are bounded by timeout, input/output size and an atomic daily cost budget; the default provider is offline and zero-cost.
- [x] No CAPTCHA, authentication, paywall, robots, access-control or rate-limit bypass.
- [x] Alert rules and histories are owner-filtered; targeted SSE delivery is isolated by authenticated user ID, and quiet-hour delivery rechecks pause/revocation state.
- [x] Only in-app notifications are active; external channel preferences never collect or transmit a destination without a configured consent workflow.
- [x] Production startup fails closed unless MongoDB and Redis transports are encrypted and refresh cookies are Secure.
- [x] Aggregate-only Prometheus metrics, bounded-cardinality latency histograms, W3C trace correlation, SLO alert rules and a Grafana dashboard are included.
- [x] Retention automation, checksum-verified backup/isolated restore scripts, a native deployment runbook, load tests and CycloneDX SBOM generation are included.
- [x] Event-time evaluation rejects and persists look-ahead leakage before computing accuracy or precision.

## Required before internet production

- [ ] Replace local passwords with secret-manager values and enable TLS/Secure cookies/HSTS.
- [ ] Add synchronizer/double-submit CSRF tokens if SameSite boundaries or cross-site clients change; Phase 1 validates browser Origin for refresh/logout.
- [ ] Add route-level RBAC enforcement for admin/analyst endpoints and record privileged audit events.
- [ ] Connect the implemented OTLP exporter and Prometheus/SLO assets to an authenticated production backend; choose production trace sampling and complete PII redaction tests.
- [ ] Add email verification, reset flow, credential-stuffing protection and session/device management.
- [ ] Enable managed MongoDB continuous backup, vault encryption/access logs and the documented 90-day key rotation process.
- [ ] Run the supplied SAST, `govulncheck`, `npm audit`, SBOM and secret-scan gate in CI and retain signed release evidence (container scanning is not applicable to the native deployment).
- [ ] Complete provider terms/DPA/data-residency reviews and document user consent/unsubscribe flows.
- [ ] Penetration-test auth, IDOR, SSE authorization, admin actions, parser sandboxing, SSRF and prompt injection.
- [ ] Automate the documented operator-reviewed account-erasure workflow; scheduled retention for raw documents, sessions, notification details and evaluation audit summaries is implemented.
- [ ] Establish incident response, correction/retraction handling and financial-content review procedures.
