# Service objectives and alerts

Initial monthly objectives exclude planned maintenance: API availability 99.9%; p95 non-stream request latency under 750 ms; successful approved-source polls 99%; analyzed-event queue age below two polling intervals; signal-without-evidence count zero; and in-app notification delivery 99% within five minutes outside quiet hours.

Prometheus scrapes `/metrics` over the private operations network. Import `deployments/grafana/stocker-dashboard.json` and load `deployments/prometheus-alerts.yaml`. Page on API absence, sustained 5xx ratio, dead letters, source health failure or new leakage; ticket latency/SSE drops and AI-spend anomalies. Logs carry W3C-compatible trace IDs and request/span IDs for correlation. Setting `OTEL_EXPORTER_OTLP_ENDPOINT` enables batched OTLP/HTTP server spans with parent propagation; the bundled collector is a local receiver template, so production still needs an approved authenticated backend/exporter.
