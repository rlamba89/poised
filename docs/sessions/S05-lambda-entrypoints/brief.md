# S05: Lambda entry points and per-country config

**Plan:** F3 ([f3-lambda-entrypoints.md](../../plans/f3-lambda-entrypoints.md)), all of it, plus A-20 rules 1–3 ([saas-requirements §13 "Countries"](../../saas-requirements.md)).
**Depends on:** S04 verified.

**Ask Rahul first:**
- Format dates using the locale from `GET /api/config`, instead of the hardcoded `en-GB` in `apps/web/src/lib/format.ts` and `apps/web/src/features/editor/OptionLists.tsx`?
  - This is a *Proposed* item under A-20.
  - **Recommended: yes.** It's small, and it removes the UK assumption from the web app.

## Build

- **F3 Steps 1–4:**
  - the staff and patient routers;
  - `cmd/staff-api` and `cmd/patient-api`, using `algnhsa` on Lambda;
  - `internal/config`, reading SSM;
  - `make build-lambdas`;
  - `DEV_LOGIN` off by default.
- **`internal/config` holds the per-country values** (A-20): `APP_URL`, the default time zone, locale and phone country. There are UK defaults locally. On Lambda, a missing value stops the function at start-up. Write the unit tests first.
- **`GET /api/config`** returns the public per-country values: locale and time zone (and later the sign-in URL). The web app reads it once.

## Not in this session

- AWS resources (S06);
- the jobs and evaluate functions (S16 and later).

## Done when

- `make lint test test-integration e2e` pass.
- `make build-lambdas` builds both functions.
- `cmd/patient-api` run alone serves `/api/p/…`, and returns 404 for staff routes and the dev login.

## Manual test focus

- a regression smoke of J1–J5 by hand;
- **the routers split correctly.** The test plan may ask the tester to run `cmd/patient-api` alone on another port for a few minutes, and check (in the browser or with curl) that the patient link works and staff routes and the dev login return 404;
- dates still look right.
