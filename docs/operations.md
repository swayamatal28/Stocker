# Native operations, backup and recovery

STOCKER deploys as native Go binaries plus a static Vite bundle; Docker is not required. MongoDB is the system of record. Redis carries locks, limits and streams and can be reconstructed from MongoDB/outbox state.

## Build and release

1. Run the verification, load and security commands from `README.md`. Archive their output, the commit SHA and `artifacts/stocker-sbom.cdx.json` with the release.
2. Build `go build -trimpath -o artifacts/stocker-api ./apps/api`, and repeat for `workers/ingestion`, `workers/analysis`, and `workers/maintenance`. Run `npm --prefix apps/web run build` and publish `apps/web/dist` behind an HTTPS static host/reverse proxy.
3. Store production values in the host secret manager, not a file in the repository. `APP_ENV=production` refuses plaintext MongoDB/Redis connections, insecure cookies and short JWT secrets.
4. Run the API once against the production database before adding traffic; startup applies idempotent index/data migrations. Start workers only after `/health/ready` succeeds.
5. On Linux install four systemd units with an unprivileged `stocker` account, `EnvironmentFile` pointing to a root-readable secret file, `Restart=on-failure`, and `NoNewPrivileges=true`. On Windows use four NSSM/Windows Service entries under a non-admin service account. Run maintenance as a single instance or scheduled task with `MAINTENANCE_RUN_ONCE=true`.
6. Terminate TLS at the reverse proxy, expose only `/api/v1`, `/health/*` and a network-restricted `/metrics`, and configure HSTS on the web/static host. Do not expose MongoDB or Redis publicly.

Rollback application binaries and the static bundle together, then verify ready/metrics and a sampled login/search flow. Schema changes are forward-compatible and idempotent; never rewrite historical evidence, signals or outcomes during rollback.

## Backup and restore drill

Managed production must use encrypted multi-zone MongoDB with continuous point-in-time backup. Target RPO is 15 minutes and RTO is 2 hours. Run an additional native archive and checksum with `scripts/backup-mongodb.ps1`; store it in an encrypted, access-logged vault separate from the database account.

Quarterly, restore with `scripts/restore-drill.ps1`. The script refuses a target that does not end with `_restore_<drill-id>`, validates the checksum when a manifest is present, and restores only into the isolated namespace. Point a temporary API at that database and verify `/health/ready`, collection counts, indexes, newest quote/evidence watermarks, evaluation runs, and sampled signal-to-evidence links. Record timestamps, RPO/RTO achieved, operator, backup ID and discrepancies. Drop the isolated database only after evidence is approved.

## Retention and deletion

`workers/maintenance` removes expired refresh sessions/raw payloads, old delivered/read alerts and their deliveries, revoked rules, briefings and evaluation-run summaries. Windows are configured with `ALERT_RETENTION_DAYS`, `BRIEFING_RETENTION_DAYS`, and `AUDIT_RETENTION_DAYS`. Immutable normalized evidence, analyses, signals and signal outcomes are preserved. Account erasure remains an operator-reviewed request: export the affected object IDs, delete user/watchlist/rule/alert/briefing/session records in a transaction, retain only legally required pseudonymized audit evidence, then record completion without the user's content.

## Keys and incident response

Rotate JWT, database, Redis and external-provider credentials at least every 90 days and immediately after suspected exposure. Use overlap where supported: introduce the new credential, restart/verify, revoke the old one, and inspect authentication failures. JWT rotation invalidates access tokens; revoke refresh sessions for a full session reset.

For an incident: declare severity and incident lead; preserve logs/audit evidence; contain credentials/routes/workers; assess user/data/financial-content impact; correct or retract affected articles/signals without rewriting historical records; notify stakeholders under applicable timelines; restore and validate; then publish a blameless review with detection and prevention actions. Never paste secrets or user data into tickets or AI providers.

See `docs/slo.md` for alerts, `docs/load-testing.md` for capacity evidence, `docs/evaluation.md` for model evidence and `docs/security-review.md` for the release gate.
