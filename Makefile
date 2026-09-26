.PHONY: dev-api dev-web dev-ingestion dev-analysis test lint

dev-api:
	go run ./apps/api

dev-web:
	npm --prefix apps/web run dev

dev-ingestion:
	go run ./workers/ingestion

dev-analysis:
	go run ./workers/analysis

test:
	go test ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web test -- --run

lint:
	go vet ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web run lint
