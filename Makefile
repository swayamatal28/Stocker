.PHONY: dev-api dev-web dev-ingestion dev-analysis maintenance test lint load security

dev-api:
	go run ./apps/api

dev-web:
	npm --prefix apps/web run dev

dev-ingestion:
	go run ./workers/ingestion

dev-analysis:
	go run ./workers/analysis

maintenance:
	go run ./workers/maintenance

test:
	go test ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web test -- --run

lint:
	go vet ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web run lint

load:
	powershell -ExecutionPolicy Bypass -File scripts/run-load.ps1

security:
	powershell -ExecutionPolicy Bypass -File scripts/security-check.ps1
