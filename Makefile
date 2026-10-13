# Local dev: `docker compose up -d db`, `make migrate seed`, `make dev`.
-include .env
export

API := apps/api
GOOSE := cd $(API) && go tool goose -dir db/migrations postgres "$(DATABASE_URL)"

.PHONY: dev staff-api patient-api test test-integration e2e-setup e2e lint db-up migrate migrate-down seed sqlc platform-admin

node_modules: package-lock.json
	npm install
	@touch node_modules

.env:
	cp .env.example .env

# Runs the Go API (every route, with the dev login) and the web app (Vite) together; Ctrl-C
# stops both.
dev: node_modules .env
	@trap 'kill 0' INT TERM EXIT; \
	(cd $(API) && DEV_LOGIN=true go run ./cmd/api) & \
	npm run dev -w apps/web & \
	wait

# One API function alone, as it runs on AWS (plan F3, D-4): the patient's link only, or the
# staff routes only (with the dev login). ADDR is where it listens (default :8081), so it can
# run beside make dev, e.g. `make patient-api` then `curl localhost:8081/api/health`.
ADDR ?= :8081
patient-api: .env
	cd $(API) && API_ADDR=$(ADDR) go run ./cmd/patient-api

staff-api: .env
	cd $(API) && API_ADDR=$(ADDR) DEV_LOGIN=true go run ./cmd/staff-api

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

# Adds a superadmin by hand (A-21), e.g. make platform-admin EMAIL=you@example.org NAME="Your Name".
# Run it once per environment; running it again changes nothing.
platform-admin: .env
	@test -n "$(EMAIL)" || (echo 'Usage: make platform-admin EMAIL=you@example.org [NAME="Your Name"]' && exit 2)
	cd $(API) && go run ./cmd/platform-admin add "$(EMAIL)" "$(NAME)"

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
