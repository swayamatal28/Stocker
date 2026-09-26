# Operations, backup and recovery

For local backups, stop write-producing workers and run `mongodump --uri="$MONGODB_URI" --db=stocker --archive=stocker.archive --gzip`. A recovery drill restores into a separate database with `mongorestore --archive=stocker.archive --gzip --nsFrom='stocker.*' --nsTo='stocker_restore.*'`, starts the API against it, verifies `/health/ready`, document counts, indexes, the newest quote/evidence watermarks, and sampled signal-evidence links.

Production should use a managed encrypted MongoDB replica set with continuous backup, multi-zone replicas, and quarterly restore drills. Target RPO is 15 minutes and target RTO is 2 hours for the initial release. Redis is reconstructed from MongoDB/outbox state; its AOF improves restart behaviour but is not the system of record.

Alerts: API 5xx >2%, p95 >750 ms, queue age >2 polling intervals, parser failure ratio >10%, circuit open, source stale beyond policy, signal without evidence (must be zero), raw-retention deletion lag, notification failure >5%, AI schema rejection/cost anomalies, database disk/PITR failure.

Run versioned document migrations before rolling API/worker instances. Migrations are forward-compatible and idempotent; destructive field cleanup requires a separate, rehearsed release. Roll back application releases first. Never rewrite historical signals or evidence during rollback.
