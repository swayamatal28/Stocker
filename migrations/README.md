# MongoDB data migrations

STOCKER uses MongoDB, so it does not apply SQL migrations. Collection indexes and idempotent development seed documents are created by `internal/store.Mongo.EnsureIndexes` and `internal/store.Mongo.Seed` when the API starts.

Future data-shape migrations belong here as versioned Go commands. Each migration must:

- record its version and completion time in the `schema_migrations` collection;
- be idempotent and safe to resume after interruption;
- update documents in bounded batches;
- preserve historical evidence and signal snapshots;
- provide a tested rollback or forward-repair procedure.
