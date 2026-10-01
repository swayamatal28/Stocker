# Event-time signal evaluation

`GET /api/v1/evaluation/report` reconstructs each historical signal at its original `generated_at`. Entry quotes must have both `as_of` and `retrieved_at` no later than that decision. Exit quotes must be at/after the configured horizon and must have been retrieved by the report's `asOf`. Signals without mature prices remain pending.

Every feature timestamp is compared with decision time. A later timestamp is persisted to `evaluation_leakage` and the signal is excluded, not silently scored. Valid outcomes are idempotently persisted to `signal_outcomes`; `evaluation_runs` records each report summary. Version `event-time-v1` uses a ±25 signal-strength direction and a ±1% realised-return neutral band.

The report includes accuracy and positive precision separately by confidence band, event category, sector and horizon. A zero precision with zero predicted positives is “not estimable,” not evidence of poor predictions. Small samples, transaction costs, survivorship bias and provider-history gaps limit conclusions; the UI therefore shows counts and never describes the result as a trading guarantee.

Run the engine unit tests and `TestPhase6EventTimeEvaluationRejectsLeakedSignal` with the Mongo integration flag to produce repeatable evaluation/leakage evidence.
