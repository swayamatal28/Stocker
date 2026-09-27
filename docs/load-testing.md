# Load and capacity evidence

Run `scripts/run-load.ps1 -BenchTime 5s` on a release-like host. It executes a 250-subscriber targeted SSE isolation test, a 1,000-subscriber fan-out benchmark and a 1,000-item ingestion batch benchmark. Store raw output with CPU/RAM, Go version and commit SHA.

Release acceptance: no cross-user SSE delivery, no test failure/drop under the isolation test, stable memory across repeated runs, and no material regression versus the previous approved release. Benchmark throughput is hardware-specific; establish the first approved run as baseline and investigate >20% degradation. For production sizing, repeat with real MongoDB/Redis staging instances and the expected source/item distribution rather than treating in-memory benchmark numbers as an SLO.
