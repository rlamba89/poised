# F3: Go API: Lambda entry points

**Goal:** the Go API can run as separate AWS Lambda functions (D-4) and still runs as one local server for development.

F3 adds **staff-api** and **patient-api**. The other two functions come with the features that need them:
- **jobs**, in C2, when the first scheduled work (invites and reminders) appears;
- **evaluate**, in C6.

**Depends on:** F1 (integration harness).

**Not in F3:** AWS resources (that's F4), and new routes or behaviour.

## Step 1: split the router by audience

- `httpapi.NewStaffRouter(deps)` serves:
  - `/api/h/…`
  - `/api/me`, `/api/codes`, `/api/categories`
  - `/api/logout`
  - the dev login (only when `DEV_LOGIN=true`)
- `httpapi.NewPatientRouter(deps)` serves only `/api/p/…`.
- Both serve `GET /api/health`.
- `httpapi.NewRouter(deps)` (local only) mounts both on one mux, so `make dev` and F1's tests keep working.
- `deps` is a struct holding the pool and the token secret. Later plans add the clock, SMS, email, S3, Cognito and evaluate to it. Tests pass fakes.
- **The dev login is off unless `DEV_LOGIN=true`.** It's on locally, and on in QA until C1 replaces it with Cognito, but **never in Training or Production** (F4 enforces this).

**Integration tests (write first):**
- The patient router returns 404 for a staff route (e.g. `/api/h/{hid}/episodes`) and for the dev login.
- The staff router returns 404 for `/api/p/{token}`.
- With `DEV_LOGIN` unset, `POST /api/dev/login` → 404.

## Step 2: entry points

- `cmd/staff-api/main.go` and `cmd/patient-api/main.go`. Each loads its config, builds its router, then:
  - **on Lambda** (when `AWS_LAMBDA_FUNCTION_NAME` is set), hands the router to an adapter for API Gateway **HTTP API (payload v2)**. Use `github.com/akrylysov/algnhsa` (a single `algnhsa.ListenAndServe(handler, opts)` call). Check that its current release supports payload v2 before pinning it.
  - **otherwise**, calls `http.ListenAndServe` (useful for running one function alone).
- `cmd/api` stays as the local all-in-one server.

## Step 3: config and database on Lambda

- **`internal/config`:**
  - reads environment variables;
  - if `DATABASE_URL_PARAM` / `TOKEN_SECRET_PARAM` are set, it fetches those **SSM Parameter Store** SecureStrings once, at cold start (AWS SDK for Go v2). The standard tier is free.
  - **Per-country values come only from here (A-20):** the web address (`APP_URL`), default time zone and locale, phone country, and later the Cognito, SES, SMS and bucket names.
    - Local dev has UK defaults.
    - On Lambda, a missing value stops the function at start-up, so there's no silent fallback to the UK.
    - No other package may contain a region, domain, time zone or locale literal.
  - Unit-test it with a fake SSM client, including the missing-value case.
- **Pool settings on Lambda:** `MaxConns=2` (each Lambda instance serves one request at a time), and a short `MaxConnIdleTime`.
- **Neon (F4) connection:** use Neon's **direct** endpoint, not the pooled one. At our low concurrency it's simpler, and the pooled endpoint needs pgx changes for prepared statements.

## Step 4: build

- `make build-lambdas` produces `apps/api/dist/<fn>/bootstrap`, using `GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags lambda.norpc` for `provided.al2023`.
- Add `apps/api/dist/` to `.gitignore`.

## Checks

- `make test`, `make test-integration` and `make e2e` pass (J1–J5 against `make dev`).
- `make build-lambdas` builds both functions.
- Running `cmd/patient-api` locally serves `/api/p/{token}` and returns 404 for staff routes.

## Status

Not started.
