# Local dev: `docker compose up -d db`, `make migrate seed`, `make dev`.
-include .env
export

API := apps/api
GOOSE := cd $(API) && go tool goose -dir db/migrations postgres "$(DATABASE_URL)"

.PHONY: dev test test-integration lint db-up migrate migrate-down seed sqlc

# Local development only: the stub sign-in on, and cookies allowed over plain HTTP. Never in a deployed stage.
DEV_LOGIN ?= true
COOKIE_SECURE ?= false
APP_ORIGIN ?= http://localhost:3000
# Seals patient links (32 bytes, base64). This one is for local development only; a deployed
# stage gets its own from SSM, and changing it means existing links can't be shown again.
LINK_KEY ?= +oYrNtlcJ/N1kubqki7SvHBpR/azB49mlJM7UjMe9ho=

# A maintenance database on the compose Postgres; the integration tests create their own databases from it.
TEST_DATABASE_URL ?= postgres://sj:sj@localhost:5432/postgres?sslmode=disable

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

# API → database tests (build tag "integration"), each in its own database copied from a migrated template.
test-integration:
	docker compose up -d --wait db
	cd $(API) && TEST_DATABASE_URL="$(TEST_DATABASE_URL)" go test -tags integration ./...

lint: node_modules
	cd $(API) && go vet ./...
	npm run lint --workspaces --if-present
