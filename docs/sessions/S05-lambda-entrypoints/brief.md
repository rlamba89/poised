# S05: Per-country config and the staff/patient router split

*Local phase (12 Oct): runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Plan:** F3 ([f3-lambda-entrypoints.md](../../plans/f3-lambda-entrypoints.md)), Step 1, the local half of Steps 2–3, plus A-20 rules 1–3 ([saas-requirements §13 "Countries"](../../saas-requirements.md)). The Lambda adapter, SSM and `make build-lambdas` move to S06 (AWS phase).
**Depends on:** S04 verified.

**Ask Rahul first:**
- Format dates using the locale from `GET /api/config`, instead of the hardcoded `en-GB` in `apps/web/src/lib/format.ts` and `apps/web/src/features/editor/OptionLists.tsx`?
  - This is a *Proposed* item under A-20.
  - **Recommended: yes.** It's small, and it removes the UK assumption from the web app.

## Build

- **F3 Step 1:** the staff and patient routers, with `NewRouter` mounting both for local use, and `DEV_LOGIN` off by default (on in `make dev`).
- **`cmd/staff-api` and `cmd/patient-api`** as plain local servers (`http.ListenAndServe`). There's no Lambda adapter yet; S06 adds it.
- **The code stays Lambda-friendly** (12 Oct): no state in the server's memory, and no reliance on local disk.
- **`internal/config` holds the per-country values** (A-20): `APP_URL`, the default time zone, locale and phone country. It reads environment variables only; SSM comes in S06. There are UK defaults for local dev, plus a strict mode (`CONFIG_STRICT=true`, used on AWS later) in which a missing value stops start-up. Write the unit tests first.
- **`GET /api/config`** returns the public per-country values: locale and time zone (and later the sign-in URL). The web app reads it once.

## Not in this session

- the Lambda adapter, SSM and AWS resources (S06);
- jobs (S16) and the evaluate function (C6).

## Done when

- `make lint test test-integration e2e` pass.
- `cmd/patient-api` run alone serves `/api/p/…`, and returns 404 for staff routes and the dev login.

## Manual test focus

- a regression smoke of J1–J5 by hand;
- **the routers split correctly.** The test plan may ask the tester to run `cmd/patient-api` alone on another port for a few minutes, and check (in the browser or with curl) that the patient link works and staff routes and the dev login return 404;
- dates still look right.
