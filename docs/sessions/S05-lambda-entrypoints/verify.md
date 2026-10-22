# S05 verification: per-country config and the staff/patient router split

# Verification 1 (12 Oct 2026): **Verified**

Automatic run (`/run-sessions`). I reviewed the whole uncommitted working tree (HEAD `6563286`), looking only at S05's changes, not the 12 Oct planning edits. I ran every test, ran `/code-review high` and weighed its findings myself, and spot-checked the router split in the browser and with curl.

## Scope against the brief

Everything in the brief is built, and nothing big was added beyond it:
- **F3 Step 1:** `NewStaffRouter`, `NewPatientRouter` and `NewRouter`, all built from `Deps{Pool, Config}`.
  - The dev login (`/api/dev/users`, `/api/dev/login`) is mounted only when `DEV_LOGIN=true` ([router.go:63](../../../apps/api/internal/httpapi/router.go#L63)).
  - `make dev` sets `DEV_LOGIN=true`.
- **`cmd/staff-api` and `cmd/patient-api`:** plain servers over `internal/serve`.
  - They hold no state in memory and don't touch the disk.
  - `make staff-api` and `make patient-api` start them.
- **`internal/config`:** reads environment variables only.
  - It falls back to UK defaults locally.
  - With `CONFIG_STRICT=true`, one error names every missing value and the server doesn't start.
  - The unit tests were written first, as the build notes record.
- **`GET /api/config`:** returns `{locale, timeZone}` and nothing else.
- **The web app:** dates use the locale from that route (Rahul's answer, 12 Oct).
- **"Done when":** `make lint test test-integration e2e` pass. Run alone, `cmd/patient-api` serves `/api/p/…` and returns 404 for the staff routes and the dev login (checked myself, below).

## Checklist

- **Tests first:** `GET /api/config` has its five cases (`TestConfig`).
  - Wrong role and another trust don't apply, because the route holds no trust data. The test proves a signed-in user gets the same answer.
  - The router split has three integration tests: `TestPatientRouter`, `TestStaffRouter` and `TestDevLoginOffUnlessEnabled`. They assert that no rows are written through the wrong router.
  - `internal/config` has unit tests, `config.ts` and `format.ts` have Vitest tests, and J3 now checks the date-of-birth format.
- **Tenant isolation:** no query changed.
  - The patient router serves only `/api/p/…`, plus health and config.
  - `chapter.ForPatient` and its stripping are untouched.
- **Nothing hardcoded for the UK (A-20):** the grep finds only comments in `apps/web/src/lib/format.ts`, plus `internal/config/config.go`, which is allowed.
  - The new guard test (`literals_test.go`) enforces this from now on.
  - One older gap remains, which the guard can't see (finding 1).
- **Security:**
  - No secrets or connection strings are in the code. `.env.example` has only the existing local placeholders.
  - `/api/config` exposes only the locale and time zone.
  - The patient API can't reach staff routes (tests, plus my curl checks).
  - The start-up log line holds no patient data.
- **Docs:** these are updated:
  - the F3 Status section;
  - `docs/plans/README.md` and `docs/sessions/README.md`;
  - `docs/decisions.md` (S05 section);
  - `CLAUDE.md`, `README.md` and `apps/web/AGENTS.md`.

  The incumbent-name grep prints nothing.

## Findings (all minor; none blocks; logged in [pending.md](../../pending.md) §3)

These came from `/code-review high`, and I weighed each one:

1. **Answer dates in notes, outputs and the POA Summary are still day/month/year.** [packages/clinical/src/outputs.ts:62](../../../packages/clinical/src/outputs.ts#L62)
   - `formatDate` turns `yyyy-MM-dd` into `dd/MM/yyyy`. This is a UK assumption with no locale literal, so the guard test can't see it.
   - It's older than S05 and outside the brief, which named only `format.ts` and `OptionLists.tsx`.
   - It's harmless while only the UK runs.
   - **Question for Rahul, before a second country:** should `packages/clinical` take the locale? **Recommended: yes,** passed in as `format.ts` does.
2. **The `DEFAULT_LOCALE` shape check accepts tags that the browser rejects.** [config.go:60](../../../apps/api/internal/config/config.go#L60)
   - `en-12` and `en-gb-gb` pass, but `new Intl.DateTimeFormat` throws on them, so every page that shows a date would crash.
   - Report 1's `zz-ZZ` was harmless (the browser falls back), but these aren't.
   - The risk is low, because the value is set once per country. Still, tighten the region part to `[A-Z]{2}` or three digits, or check the tag in `infra/countries.py` (S06).
3. **The first render waits for `GET /api/config` with no timeout.** [main.tsx:25](../../../apps/web/src/main.tsx#L25)
   - A request that hangs (rather than fails) leaves a blank page.
   - Add `AbortSignal.timeout(…)` before S06, because Lambda cold starts make this more likely.
4. **Small:**
   - `APP_URL=/` is trimmed to empty after the "missing" check, so it passes strict mode ([config.go:102](../../../apps/api/internal/config/config.go#L102)).
   - The literal guard skips any line that starts with `*` (so a Go `*x = "Europe/London"` line isn't checked), and it reads only `.go`, `.ts` and `.tsx` files.

**Considered and dismissed:**
- **"Strict mode should refuse `DEV_LOGIN=true`":** no. QA on AWS runs in strict mode *and* needs the dev login (F4). The guard against the dev login in Training or Production is F4's CDK synth check. Recorded in decisions.md.
- **"`timeZone` is loaded but unused":** this is the builder's decision 4. Times stay in the browser's zone until S08's hospital time zones.
- **"A new `Intl.DateTimeFormat` per call":** negligible at these list sizes.

## Test results (12 Oct, my runs)

- `make lint`: go vet and tsc clean.
- `make test`: every Go package ok. Vitest: web 4 files, 51 tests passed; clinical 14 files, 111 tests passed.
- `make test-integration`: every package ok (`httpapi` 4.8 s).
- `make e2e` (against the conductor's `make dev`): **5 passed** in 22 s.

## Spot checks (mine)

1. **`patient-api` alone** (`make patient-api ADDR=:8093`, with curl):
   - `/api/health`, `/api/config` and a real `/api/p/{token}` returned 200;
   - `/api/p/not-a-token` returned 404;
   - `/api/me`, `GET /api/dev/users`, `POST /api/dev/login`, `POST /api/logout`, `/api/codes` and `/api/h/…/episodes` all returned 404;
   - the log line said `dev login off, locale en-GB, time zone Europe/London`.
2. **Patient page through `patient-api` alone:** I ran a second Vite (`API_URL=http://localhost:8093`, `--port 3093 --strictPort`) and opened a patient link in Claude in Chrome. The HQ (About you / Your health) rendered.
   - From that page, `fetch` got 404 for `/api/dev/users`, `/api/me` and a staff episodes URL, and 200 for `/api/config`.
   - The window wouldn't resize to 390×844, the same tool limit report 1 hit. S05 changed no patient UI.
3. **Another hospital:** signed in as Bea Author (Hospital B), the Hospital A episodes URL showed "You don't have access to this hospital."
4. **Dates:** as Cara Clinician, the episodes list shows en-GB date-times (`12/10/2026, 22:23`) with the config loaded from the main server.

I stopped both extra servers afterwards. The conductor's `make dev` is untouched.

## Hand checks for Rahul

- None required.
- Optional: open a patient link in a real 390×844 window. Both the tester and I hit the window-resize limit, but S05 changed no patient UI.

## Suggested commits

1. **F3 Step 1 + local Steps 2–3: `internal/config`, the staff and patient routers, `GET /api/config`, `cmd/staff-api` and `cmd/patient-api`.**
   - `apps/api/internal/config/*`
   - `apps/api/internal/httpapi/router.go` and `routers_integration_test.go`
   - `apps/api/internal/serve/serve.go`
   - `apps/api/cmd/{api,staff-api,patient-api}/main.go`
   - `apps/api/internal/apitest/apitest.go`
   - `Makefile` and `.env.example`
2. **Dates in the deployment's locale (A-20).**
   - `apps/web/src/lib/{config,config.test,format,format.test}.ts`
   - `apps/web/src/main.tsx`
   - `apps/web/src/features/editor/OptionLists.tsx`
   - `e2e/journeys/test_j3_patient.py`
3. **Docs: S05's build notes, test plan, report, verification and status, plus the 12 Oct planning edits.** This commit holds the mixed files whole:
   - `docs/sessions/S05-lambda-entrypoints/*`
   - `docs/plans/f3-lambda-entrypoints.md` and `docs/plans/README.md`
   - `docs/sessions/README.md`
   - `docs/decisions.md` and `docs/pending.md`
   - `CLAUDE.md`, `README.md` and `apps/web/AGENTS.md`

Next: `/build-session S08`.
