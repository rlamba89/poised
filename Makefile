# Local dev: `docker compose up -d db`, `make migrate seed`, `make dev`.
-include .env
export

API := apps/api
GOOSE := cd $(API) && go tool goose -dir db/migrations postgres "$(DATABASE_URL)"

.PHONY: dev test test-integration lint db-up migrate migrate-down seed sqlc

node_modules: package-lock.json
	npm install
	@touch node_modules

.env:
	cp .env.example .env

# Runs the Go API and Next.js together; Ctrl-C stops both.
dev: node_modules .env
	@trap 'kill 0' INT TERM EXIT; \
	(cd $(API) && go run ./cmd/api) & \
	npm run dev -w apps/web & \
	wait

db-up:
	docker compose up -d --wait db

# Waits up to 30s for the compose database, which starts slower than `up -d` returns.
migrate: .env
	@for i in $$(seq 30); do docker compose exec -T db pg_isready -U sj >/dev/null 2>&1 && break; sleep 1; done
	$(GOOSE) up

migrate-down: .env
	$(GOOSE) down

seed: .env
	cd $(API) && go run ./cmd/seed

sqlc:
	cd $(API) && go tool sqlc generate -f db/sqlc.yaml

test: node_modules
	cd $(API) && go test ./...
	npm test --workspaces --if-present

# API → database tests (-tags integration, apps/api/internal/apitest). Each test gets its own
# database, copied from a template that every migration is run into first.
TEST_DATABASE_URL ?= postgres://sj:sj@localhost:5432/postgres?sslmode=disable
test-integration: .env db-up
	cd $(API) && go test -tags integration -count=1 ./...

# The integration tag adds the integration test files to the vet.
lint: node_modules
	cd $(API) && go vet -tags integration ./...
	npm run lint --workspaces --if-present
