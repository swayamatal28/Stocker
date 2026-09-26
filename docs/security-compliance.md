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

## Required before internet production

- [ ] Replace local passwords with secret-manager values and enable TLS/Secure cookies/HSTS.
- [ ] Add synchronizer/double-submit CSRF tokens if SameSite boundaries or cross-site clients change; Phase 1 validates browser Origin for refresh/logout.
- [ ] Add route-level RBAC enforcement for admin/analyst endpoints and record privileged audit events.
- [ ] Configure OpenTelemetry exporter, Prometheus scraping, SLO alerts, trace sampling and PII redaction tests.
- [ ] Add email verification, reset flow, credential-stuffing protection and session/device management.
- [ ] Enable managed MongoDB continuous backup, encrypted MongoDB/Redis transport and key rotation.
- [ ] Run SAST, `govulncheck`, `npm audit`, SBOM generation, secret scanning and signed container scanning in CI.
- [ ] Complete provider terms/DPA/data-residency reviews and document user consent/unsubscribe flows.
- [ ] Penetration-test auth, IDOR, SSE authorization, admin actions, parser sandboxing, SSRF and prompt injection.
- [ ] Add retention/deletion automation for raw documents, accounts, notification details and audit data.
- [ ] Establish incident response, correction/retraction handling and financial-content review procedures.
