# S05: build notes

Builder: automatic run (`/run-sessions`), 12 Oct 2026.

## Answers

- **Dates use the locale from `GET /api/config`** instead of the hardcoded `en-GB` (Rahul, 12 Oct, [answers.md](answers.md)). Built.

## What was built

### Go

- **`internal/config`** (new): `config.Load(getenv)` reads every setting from the environment, once, at start-up.
  - `DATABASE_URL` and `TOKEN_SECRET` are always required.
  - `API_ADDR` (default `:8080`) and `DEV_LOGIN` (default off; `true`/`false` only).
  - **Per-country values (A-20):**

    | Env var | Local default | Checked |
    | --- | --- | --- |
    | `APP_URL` | `http://localhost:3000` | an http(s) address; a trailing `/` is dropped |
    | `DEFAULT_TIME_ZONE` | `Europe/London` | a real IANA zone (`Local` refused) |
    | `DEFAULT_LOCALE` | `en-GB` | a language tag such as `en-CA` |
    | `PHONE_COUNTRY` | `GB` | two capital letters |

  - **`CONFIG_STRICT=true`** (for AWS later): no local defaults. One error names every missing variable, and the server doesn't start.
  - The time-zone database is built into the binary (`time/tzdata`), so a zone never depends on the host (Lambda's base image).
  - `Config.Public()` is what the browser may see: the locale and time zone only.
  - **A guard test** (`literals_test.go`) scans `apps/api`, `apps/web/src` and `packages/clinical/src` (not tests or comments) and fails on a locale (`"en-GB"`), IANA zone (`"Europe/London"`) or AWS region (`"eu-west-2"`) literal outside `internal/config`. Checked by planting `"en-GB"` in `format.ts`: it failed, naming the file and line.
- **`internal/httpapi/router.go`:** `Deps{Pool, Config}`, and three routers:
  - `NewStaffRouter`: `/api/h/…`, `/api/me`, `/api/codes`, `/api/categories`, `/api/logout`, plus `/api/dev/users` and `/api/dev/login` **only when `DEV_LOGIN=true`**;
  - `NewPatientRouter`: `/api/p/…` only;
  - both serve `GET /api/health` and the new `GET /api/config`;
  - `NewRouter` mounts both (the local all-in-one server and the tests).
- **`GET /api/config`** → `{"locale": "en-GB", "timeZone": "Europe/London"}`. No sign-in (the sign-in page and the patient's link need it first). Nothing else is in it: not the web address, phone country, database or secret.
- **`internal/serve`** (new): `serve.Main(name, router)` loads the config, opens the pool and listens. It logs whether the dev login is on, plus the locale and zone. S06 adds the Lambda branch here.
- **`cmd/api`** (the local all-in-one), **`cmd/staff-api`** and **`cmd/patient-api`** (new) are one line each over `serve.Main`. They keep no state in memory and never touch the disk (Lambda-friendly, 12 Oct).
- **`internal/apitest`:** `NewWith(t, Options{Router, Env})` starts a test server with another router or environment. The config goes through the real `config.Load`. `New(t)` is unchanged for every existing test (full router, `DEV_LOGIN=true`).

### Make and env

- `make dev` sets `DEV_LOGIN=true` for the API. Nothing else changes for Rahul: his `.env` needs no edit.
- **`make patient-api`** and **`make staff-api`** (new) run one function alone, as on AWS. `ADDR` is the port, default `:8081`, so it can run beside `make dev`. `make staff-api` turns the dev login on.
- `.env.example` documents `DEV_LOGIN`, the four per-country variables and `CONFIG_STRICT` (commented out: local defaults apply).

### Web (TypeScript for Rahul to review)

- **`apps/web/src/lib/config.ts`** (new, ~20 lines): `loadConfig()` calls `GET /api/config` once and keeps the result in a module variable. `locale()` returns its locale, or `undefined` before it's loaded (then `Intl` uses the browser's own locale).
- **`apps/web/src/main.tsx`:** creates the router first, so the first page's code starts downloading, then waits for `loadConfig()` before rendering. If the config can't be fetched, it logs a console error and renders anyway: the pages show their own "can't reach the server" errors, and dates fall back to the browser's locale.
- **`apps/web/src/lib/format.ts`:** `formatDateTime`, `formatDate` and the new `formatShortDate` pass `locale()` to `Intl.DateTimeFormat` / `toLocaleDateString`, instead of `"en-GB"`. A formatter is now made per call, which is cheap at these list sizes.
- **`OptionLists.tsx`:** uses `formatShortDate` instead of `toLocaleDateString("en-GB")`.

## Decided by the builder (automatic mode)

1. **`Deps` holds the pool and the whole `config.Config`.** The token secret comes from `Config.TokenSecret`, not a separate field, so there's one source for every setting.
2. **Env var names:** `APP_URL`, `DEFAULT_TIME_ZONE`, `DEFAULT_LOCALE`, `PHONE_COUNTRY`, `CONFIG_STRICT`, `DEV_LOGIN`. The plan named only `APP_URL` and `DEV_LOGIN`.
3. **`/api/dev/users` is switched off with `/api/dev/login`.** It only serves the stub login page, and it lists every user.
4. **Times stay in the browser's time zone; only the locale comes from config.** That's today's behaviour. `GET /api/config` returns the time zone for later. Hospital time zones come in S08 (agreed 12 Oct), which can decide which zone staff screens use.
5. **A route a router doesn't serve gets Go's plain-text `404 page not found`,** not the JSON error shape. A JSON catch-all would also turn every wrong-method 405 into a 404. No client calls these paths.
6. **Lambda pool settings (`MaxConns=2`, short idle time) wait for S06**, with the adapter. Locally, pgx's defaults are unchanged.
7. **`cmd/seed` and `cmd/import-hq` still read `DATABASE_URL` directly.** They're local tools, not functions, so moving them onto `internal/config` would only add the `TOKEN_SECRET` requirement for nothing.
8. **The guard test lives in Go** (`internal/config/literals_test.go`), though it also scans the TypeScript. **Reason:** it runs in `make test` with no new tooling, next to the only package allowed the values.

## Differences from the plan

- F3 Step 3's SSM and the fake SSM client moved to S06 by the brief (A-22). The strict-mode "missing value stops start-up" is built and tested now, from environment variables.
- `make build-lambdas`, `algnhsa` and `apps/api/dist/` in `.gitignore`: S06.

## Steps

- [x] `internal/config` unit tests, then the package
- [x] Router split integration tests (`routers_integration_test.go`), then `Deps`, the three routers, `GET /api/config` and the `DEV_LOGIN` gate
- [x] `cmd/staff-api`, `cmd/patient-api`, `internal/serve`; `make dev` sets `DEV_LOGIN=true`; `make patient-api` / `make staff-api`
- [x] Web: `config.ts` and `format.ts` tests, then the code; `OptionLists.tsx`; `main.tsx`
- [x] J3 checks the date of birth is shown in the deployment's format ("17 May 1980"). Checked it fails when the API runs with `DEFAULT_LOCALE=en-US` (the page then showed "May 17, 1980 (age 46)").
- [x] Guard test against per-country literals
- [x] Docs: CLAUDE.md, README layout, `apps/web/AGENTS.md`, `.env.example`
- [x] `make lint test test-integration e2e`
- [x] test-plan.md

## Tests added

- **Go unit tests** (`internal/config/config_test.go`): local defaults; every value read; one override keeps the other defaults; the trailing slash; the database and secret are required even locally; strict mode (a complete made-up country starts; every missing value is named; one missing value; the database and secret are still needed); 11 bad values refused in both modes; `Public()`.
- **Go guard test:** `internal/config/literals_test.go`.
- **Integration tests** (`internal/httpapi/routers_integration_test.go`):
  - `TestPatientRouter`: it serves the patient's link, health and config, and 404s every staff route, the dev login included (with `DEV_LOGIN=true`).
  - `TestStaffRouter`: it serves the staff routes (signed in through its own dev login), health and config, and 404s the patient's link (GET, PUT answers, submit), signed in or not, writing no rows.
  - `TestDevLoginOffUnlessEnabled`: with `DEV_LOGIN` unset, `NewRouter` and `NewStaffRouter` 404 both dev routes; `/api/me` still answers 401.
  - `TestConfig`, the five cases for `GET /api/config`:
    - **happy path:** a made-up country in strict mode (`fr-CA`, `America/Toronto`) comes back exactly, with no other keys;
    - **no sign-in:** the same answer;
    - **wrong role / another trust:** a signed-in user gets the same answer, since there's no trust data;
    - **invalid input:** POST is 405, and query parameters change nothing;
    - **plus:** the local defaults when nothing is set.
- **Vitest:** `apps/web/src/lib/config.test.ts` (loads once; `undefined` before loading; a failed fetch rejects and keeps no locale) and `format.test.ts` (en-GB and en-US for all three formatters).
- **e2e:** J3 checks the episode page's date of birth.

## Test results (12 Oct, before testing)

- `make lint`: go vet and tsc clean.
- `make test`: Go ok (100 tests and subtests pass); Vitest: web 4 files, 51 tests; clinical 14 files, 111 tests.
- `make test-integration`: every package ok (355 tests and subtests pass).
- `make e2e` (against `make dev`): **5 passed** in 25 s.
- Smoke: `make patient-api ADDR=:8091` answered `/api/health` and `/api/config` 200, and `/api/me`, `/api/dev/users`, `POST /api/dev/login`, `/api/h/…/episodes` 404. A second Vite (`API_URL=http://localhost:8091 … --port 3091`) served the patient page and its `/api/p/{token}` 200, with `/api/dev/users` 404.

## Manual testing

- **Report 1 (12 Oct): 22 of 22 passed** (T-01 to T-17, R-01 to R-05). No fixes were needed.
- **The tester's deviations:**
  - Port 3001 was taken by an unrelated server, so T-12's web app ran on :3003. Future plans should use `--strictPort` and a free port.
  - The window wouldn't resize, so the phone-size cases ran in a 390×844 iframe.
- **Exploration notes, considered:**
  - **A well-formed but unknown locale (`zz-ZZ`) starts the server.** Accepted and recorded in decisions.md (12 Oct, S05): the value is set once per country in infra config.
  - **No `Cache-Control` on `/api/config`.** Accepted: it's fetched once per full page load.
  - Every other note confirmed the expected behaviour: an empty value falls back locally; a dead API still renders an error, not a blank page.

## Hand checks for Rahul

- None required. Optional: open a patient link in a real 390×844 window, because the tester used an iframe.

## Suggested commits

1. **F3 Step 1 + local Steps 2–3: `internal/config`, the staff and patient routers, `GET /api/config`, `cmd/staff-api` and `cmd/patient-api`.** Files: `apps/api/internal/config/*`, `apps/api/internal/httpapi/router.go`, `apps/api/internal/httpapi/routers_integration_test.go`, `apps/api/internal/serve/serve.go`, `apps/api/cmd/{api,staff-api,patient-api}/main.go`, `apps/api/internal/apitest/apitest.go`, `Makefile`, `.env.example`.
2. **Dates in the deployment's locale (A-20).** Files: `apps/web/src/lib/{config,config.test,format,format.test}.ts`, `apps/web/src/main.tsx`, `apps/web/src/features/editor/OptionLists.tsx`, `e2e/journeys/test_j3_patient.py`.
3. **Docs: S05 build notes, test plan, reports, status, decisions.** Files: `docs/sessions/S05-lambda-entrypoints/*`, `docs/plans/f3-lambda-entrypoints.md`, `docs/plans/README.md` (with the 12 Oct planning changes, per decisions.md §2), `docs/sessions/README.md`, `docs/decisions.md`, `CLAUDE.md`, `README.md`, `apps/web/AGENTS.md`.
