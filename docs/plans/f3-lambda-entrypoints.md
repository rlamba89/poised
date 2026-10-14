# F3: Go API: Lambda entry points

**Goal:** the Go API can run as separate AWS Lambda functions (D-4) and still runs as one local server for development.

F3 adds **staff-api** and **patient-api**. The other two functions come with the features that need them:
- **jobs**, in C2, when the first scheduled work (invites and reminders) appears;
- **evaluate**, in C6.

**Split on 12 Oct (local first, A-22):**
- **Local half, S05:** Step 1, local `cmd/staff-api` and `cmd/patient-api`, and env-only `internal/config`.
- **Lambda half, S06 (AWS phase):** `algnhsa`, SSM and `make build-lambdas`.

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

**Done (14 Oct 2026): S06, the Lambda half, verified 14 Oct** ([verify.md](../sessions/S06-aws-qa-stack/verify.md)). **12 Oct: S05, the local half, verified** (report 1: 22 of 22 passed; [verify.md](../sessions/S05-lambda-entrypoints/verify.md)). The Lambda half is S06. Done:
- **Step 1:** `NewStaffRouter`, `NewPatientRouter` and `NewRouter`, with `Deps{Pool, Config}`. The dev login (`/api/dev/users` and `/api/dev/login`) exists only with `DEV_LOGIN=true`; `make dev` sets it.
- **Step 2, local:** `cmd/staff-api` and `cmd/patient-api` as plain servers over `internal/serve`; `make staff-api` / `make patient-api` run one alone.
- **Step 3, local:** `internal/config` reads env vars only, with UK defaults locally and `CONFIG_STRICT=true` refusing to start with a missing value. Plus `GET /api/config` (locale and time zone), and a guard test against per-country literals anywhere else.

**S06 (14 Oct, building):** the Lambda half is built and unit-tested: `algnhsa` v1.1.0 (payload v2) in `internal/serve` (`serve.Lambda`), `config.Resolve` (SSM and Secrets Manager at cold start, with a fake in the tests), `CONFIG_STRICT` required on Lambda, the Lambda pool settings, `FORWARDED_HOPS` for the patient's address, and `make build-lambdas`. Running on QA since 14 Oct (journeys pass against it). **S06 tested 14 Oct** (test report 1: 19 of 19 passed); **S06 verified 14 Oct: F3 is done.**

**Differences:** SSM, the Lambda adapter, the Lambda pool settings and `make build-lambdas` are S06 (A-22). The web app's dates use the config's locale (answered 12 Oct). Details: [S05 build notes](../sessions/S05-lambda-entrypoints/build-notes.md).
