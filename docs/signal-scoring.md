# Signal scoring specification

Version `signal-v1` is implemented in `internal/signal/engine.go` and persisted in `signal_versions`.

All normalized inputs are on `[-100, 100]`, except evidence metadata and confidence/reliability/materiality/recency on `[0, 100]`. Materiality, source reliability, confidence, recency, and confirmations take the sign of news sentiment. Contradiction is always subtractive.

| Component | Weight |
|---|---:|
| News sentiment | 0.16 |
| Event materiality | 0.13 |
| Source reliability | 0.08 |
| AI confidence | 0.09 |
| Recency | 0.08 |
| Independent confirmations | 0.07 |
| Price movement | 0.11 |
| Volume anomaly | 0.08 |
| Sector movement | 0.05 |
| Index movement | 0.04 |
| Valuation | 0.05 |
| Financial health | 0.08 |
| Contradictory evidence | −0.08 |

Evidence gates run before scoring: at least one evidence reference, AI confidence ≥25, and source reliability ≥20. Otherwise the output is `Insufficient evidence`.

Thresholds: `≥60` Strong positive setup; `25..59.9` Positive setup; `-24.9..24.9` Neutral/watch; `-59.9..-25` Negative setup; `≤-60` Strong negative setup.

Every stored result includes score, label, confidence, horizon, supporting reasons, risks, invalidation conditions, source links, freshness watermark, full input snapshot, and scoring version. Historical rows are append-only. Evaluation uses event-time joins and data-availability timestamps—not revised values published later—to prevent look-ahead leakage. A backtest fails if any feature's `available_at` exceeds the simulated decision time.

This model is an explainable ranking heuristic, not a forecast of return or a buy/sell recommendation.
