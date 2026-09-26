.PHONY: dev-api dev-web test lint

dev-api:
	go run ./apps/api

dev-web:
	npm --prefix apps/web run dev

test:
	go test ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web test -- --run

lint:
	go vet ./apps/api ./internal/... ./workers/...
	npm --prefix apps/web run lint
