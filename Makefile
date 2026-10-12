# Local dev: `docker compose up -d db`, `make migrate seed`, `make dev`.
-include .env
export

API := apps/api
GOOSE := cd $(API) && go tool goose -dir db/migrations postgres "$(DATABASE_URL)"

.PHONY: dev test test-integration e2e-setup e2e lint db-up migrate migrate-down seed sqlc

node_modules: package-lock.json
	npm install
	@touch node_modules

.env:
	cp .env.example .env

# Runs the Go API and the web app (Vite) together; Ctrl-C stops both.
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

# End-to-end journeys (e2e/, Playwright + pytest). e2e-setup makes e2e/.venv with Python 3.12+
# (PYTHON names it), installs the pinned requirements, then Chromium and WebKit.
PYTHON ?= python3.12
e2e-setup:
	@$(PYTHON) -c 'import sys; sys.exit(0 if sys.version_info >= (3, 12) else "e2e needs Python 3.12+; run: make e2e-setup PYTHON=<path to python3.12+>")'
	$(PYTHON) -m venv e2e/.venv
	e2e/.venv/bin/pip install -q -r e2e/requirements.txt
	e2e/.venv/bin/playwright install chromium webkit

# Runs the journeys headless against E2E_BASE_URL (default http://localhost:3000), so `make dev`
# must be running. Extra pytest options go in ARGS, e.g. make e2e ARGS="--headed --slowmo 300".
e2e:
	cd e2e && .venv/bin/pytest $(ARGS)

# The integration tag adds the integration test files to the vet.
lint: node_modules
	cd $(API) && go vet -tags integration ./...
	npm run lint --workspaces --if-present
