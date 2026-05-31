# MongoDB data migrations

STOCKER uses MongoDB, so it does not apply SQL migrations. Collection indexes and compatibility migrations are applied by the Go services when they start. Runtime demo seeding is disabled; live discovery and ingestion populate the database. The portfolio intentionally reuses the physical `watchlists` collection so existing user selections are preserved; older rows without quantity or average price appear as incomplete holdings until the user supplies those values.

Future data-shape migrations belong here as versioned Go commands. Each migration must:

- record its version and completion time in the `schema_migrations` collection;
- be idempotent and safe to resume after interruption;
- update documents in bounded batches;
- preserve historical evidence and signal snapshots;
- provide a tested rollback or forward-repair procedure.
