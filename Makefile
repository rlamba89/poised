# Local dev: `docker compose up -d db`, `make migrate seed`, `make dev`.
-include .env
export

API := apps/api
GOOSE := cd $(API) && go tool goose -dir db/migrations postgres "$(DATABASE_URL)"

.PHONY: dev staff-api patient-api test test-integration e2e-setup e2e lint db-up migrate migrate-down seed sqlc platform-admin \
	build-lambdas infra-test synth aws-setup deploy-stacks deploy-web aws-db deploy-qa destroy-stacks \
	deploy-pipeline destroy-pipeline qa-users qa-code qa-password e2e-users aws-platform-admin

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

# The pipeline starts its own Postgres and runs `make test-integration DB_UP=` (infra/pipeline_stack.py).
DB_UP ?= docker compose up -d --wait db
db-up:
	$(DB_UP)

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

test: node_modules infra/.venv
	cd $(API) && go test ./...
	npm test --workspaces --if-present
	cd infra && .venv/bin/pytest -q

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

# --- AWS (plans F3/F4, S06): see infra/README.md -------------------------------------------
# ENV names an entry in infra/countries.py. PROFILE names the AWS CLI profile to use, e.g.
#   make aws-setup PROFILE=AdministratorAccess-026090536258
# and wins over AWS_PROFILE from .env or the shell. Every script prints the profile and account,
# and refuses to run if they aren't the entry's account.
ENV ?= uk-qa
PROFILE ?=
AWS_ENV = $(if $(PROFILE),AWS_PROFILE=$(PROFILE))
AWS_CDK_VERSION := 2.1145.0
AWS_CDK := cd infra && $(AWS_ENV) npx --yes aws-cdk@$(AWS_CDK_VERSION)

# The Lambdas (provided.al2023, arm64): apps/api/dist/<fn>/bootstrap.
build-lambdas:
	@for fn in staff-api patient-api; do \
		echo "build-lambdas: $$fn"; \
		(cd $(API) && GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc -trimpath \
			-ldflags='-s -w' -o dist/$$fn/bootstrap ./cmd/$$fn) || exit 1; \
	done

# The CDK app's Python (pinned in infra/requirements.txt), with PYTHON as for e2e-setup.
infra/.venv: infra/requirements.txt
	$(PYTHON) -m venv infra/.venv
	infra/.venv/bin/pip install -q -r infra/requirements.txt
	@touch infra/.venv

# The CDK tests alone (make test runs them too): no AWS account needed.
infra-test: infra/.venv
	cd infra && .venv/bin/pytest -q

synth: infra/.venv build-lambdas
	$(AWS_CDK) synth --quiet

# Once per account and region (and safe to repeat): CDK bootstrap, then the database URL and the
# budget alarm's address into SSM, e.g.
#   make aws-setup PROFILE=<profile> NEON_DATABASE_URL='postgres://…' ALERT_EMAIL=you@example.org
aws-setup: infra/.venv
	$(AWS_ENV) infra/scripts/aws-setup.sh $(ENV)

# The environment's ApiStack and WebStack, by hand (the pipeline does this on a push).
deploy-stacks: infra/.venv build-lambdas
	$(AWS_ENV) infra/scripts/check-account.sh $(ENV)
	$(AWS_CDK) deploy 'pipeline/$(ENV)/*' --require-approval never

# The migrations and the (idempotent) seed, against the environment's database.
aws-db: infra/.venv
	$(AWS_ENV) infra/scripts/db.sh $(ENV)

# The web app: a production build, uploaded to Amplify.
deploy-web: node_modules infra/.venv
	npm run build -w apps/web
	$(AWS_ENV) infra/scripts/deploy-web.sh $(ENV)

# QA's test users (S12, cmd/qa-users): the seed's staff get sign-ins with known passwords and
# authenticator secrets, kept in SSM (the pipeline does this after each seed). Then, for a tester:
#   make qa-code WHO='Cara Clinician'       the authenticator's current 6-digit code
#   make qa-password WHO='Cara Clinician'   copies the password to the clipboard (never printed)
qa-users: infra/.venv
	$(AWS_ENV) infra/scripts/qa-users.sh $(ENV) setup

qa-code: infra/.venv
	@test -n "$(WHO)" || (echo "Usage: make qa-code WHO='Cara Clinician' (a name or an email)" && exit 2)
	$(AWS_ENV) infra/scripts/qa-users.sh $(ENV) code "$(WHO)"

qa-password: infra/.venv
	@test -n "$(WHO)" || (echo "Usage: make qa-password WHO='Cara Clinician' (a name or an email)" && exit 2)
	$(AWS_ENV) infra/scripts/qa-users.sh $(ENV) password "$(WHO)"

# The test users' sign-ins into e2e/.qa-users.json (gitignored), to run the journeys against QA:
#   make e2e-users PROFILE=… && make e2e E2E_BASE_URL=<QA's address>
e2e-users: infra/.venv
	$(AWS_ENV) infra/scripts/e2e-users.sh $(ENV)

# A superadmin in an AWS environment (A-21): the account, the platform_admins row and the staff
# pool's user, who gets Cognito's invitation email, e.g.
#   make aws-platform-admin PROFILE=… EMAIL=you@example.org NAME="Your Name"
aws-platform-admin: infra/.venv
	@test -n "$(EMAIL)" || (echo 'Usage: make aws-platform-admin EMAIL=you@example.org [NAME="Your Name"]' && exit 2)
	$(AWS_ENV) infra/scripts/platform-admin.sh $(ENV) "$(EMAIL)" "$(NAME)"

# Everything, in order (after make aws-setup).
deploy-qa: deploy-stacks aws-db qa-users deploy-web

# Deletes the environment's stacks (the database, in Neon, and SSM's values are kept).
destroy-stacks: infra/.venv
	$(AWS_ENV) infra/scripts/check-account.sh $(ENV)
	$(AWS_CDK) destroy 'pipeline/$(ENV)/*' --force

# The pipeline (plan F4 Step 2), once: after this it updates itself on every push. Its GitHub
# connection starts Pending until it's approved in the console (infra/README.md).
deploy-pipeline: infra/.venv build-lambdas
	$(AWS_ENV) infra/scripts/check-account.sh pipeline
	$(AWS_CDK) deploy pipeline --require-approval never

# Deletes the pipeline, its connection and its buckets (not the environments it deployed).
destroy-pipeline: infra/.venv
	$(AWS_ENV) infra/scripts/check-account.sh pipeline
	$(AWS_CDK) destroy pipeline --force
