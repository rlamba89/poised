# F1: Test foundation

**Goal:** put the safety net in place before anything changes. After F1:
- every existing API route has integration tests (API → real Postgres);
- today's whole workflow is covered by Playwright end-to-end journeys.

F2 (the frontend rewrite) then relies on these to prove nothing broke.

**Testing rules:** [README](README.md#testing-rules-apply-to-every-plan). Unit tests are TDD. Integration tests go from the API to the database only. End-to-end tests use Playwright + Python in `e2e/`.

**Not in F1:** new features, and changes to app behaviour. If a test finds a bug, record it under Status, and fix it in its own commit with the test that shows it.

## Step 1: integration test harness (Go)

- **Package:** `apps/api/internal/apitest`, with every file behind the build tag `//go:build integration`. That keeps `go test ./...` fast and database-free.
- **Database per test, from a template:**
  - `TestMain` connects to `TEST_DATABASE_URL`, an admin connection to the compose Postgres, e.g. `postgres://sj:sj@localhost:5432/postgres`.
  - It (re)creates `sj_test_template` and runs every goose migration into it, using goose as a library (`goose.Up`).
  - `apitest.New(t)` runs `CREATE DATABASE t_<random> TEMPLATE sj_test_template`, opens a pool, builds the **real** router (`httpapi.NewRouter`), and drops the database in `t.Cleanup`.
  - Each test is isolated, and tests can run in parallel (`t.Parallel()`).
- **Helpers** (keep them small):
  - `c.LoginAs(user)`: signs in through the real `POST /api/dev/login` and keeps the cookie.
  - `c.Do(method, path, body) *Resp`: sends a request and decodes JSON. Asserting the status code is the test's job.
  - `c.DB`: the pool, so tests can assert rows with plain SQL.
  - **Fixtures:** `fx.Hospital(t)`, `fx.User(t, hospital, roles...)`, `fx.PublishedHQ(t, hospital, contentJSON)` and `fx.Episode(t, …)`. They insert rows directly with SQL (or the sqlc queries), so tests stay short. Fixture HQ content lives in `apitest/testdata/*.json`.
- **External calls:** there are none today. Plans C1 onward add a `Deps` struct to the router (clock, SMS, email, S3, Cognito, evaluate), and `apitest` passes recording fakes.
- **Makefile:**
  - `make test-integration` starts the compose database (`docker compose up -d --wait db`) and runs `go test -tags integration ./...` with `TEST_DATABASE_URL` set.
  - Add `TEST_DATABASE_URL` to `.env.example`.

**Check:** a first test, `GET /api/health` → 200, passes. A deliberately wrong migration makes `TestMain` fail with a clear message.

## Step 2: integration tests for every existing route

For each route group in `internal/httpapi/router.go`, write the five cases from the README's rule 2: happy path, no sign-in, wrong role, **other hospital's data**, and invalid input. Assert both the response and the database rows.

| Group | Must also cover |
| --- | --- |
| Questionnaires, versions, publish | Publishing retires the previous version. Content writes to a published version are refused. "New version" copies the Question Sets with the same stable IDs. |
| Chapters (Question Sets) | Saving with a stale `revision` → 409 and nothing written. Revision +1 on save. |
| Option lists, codes, categories | Code search finds by code prefix and by words in the description. |
| Patients and episodes | Creating an episode needs a published version. A "created" event is written. Status changes write events. |
| Patient link (`/api/p/{token}`) | **The patient JSON has no `clinicianOnly` elements, `clinicalOutputs` or `testCases`.** Answer keys that aren't patient questions are dropped. Answers are frozen after submit. A bad token → 404. |
| Clinician answers, validate, complete review | The first write copies the patient's answers. Validated sets are stamped. Complete review is refused until every set is validated. Read-only afterwards. |
| Notes | Append-only. User and time are recorded. |

Leave the existing fake-querier tests (`fakeQ`) alone. Per rule 4, they're replaced only when their area is next changed.

**Check:** `make test-integration` passes. Removing the clinician-content strip in `chapter/patient.go` makes a test fail (try it, then revert).

## Step 3: end-to-end suite (Playwright + Python)

- **Folder `e2e/`:**
  - `requirements.txt` with pinned `pytest`, `pytest-playwright` and `playwright`;
  - `conftest.py`;
  - `journeys/`;
  - `README.md`, explaining how to run it and how to read a trace.
  - Python 3.12+, in a virtualenv at `e2e/.venv` (gitignored).
- **Config:**
  - `E2E_BASE_URL` (default `http://localhost:3000`);
  - headless by default, with `--headed` for watching;
  - `--tracing retain-on-failure`, saved to `e2e/test-results/` (gitignored).
- **Data setup:**
  - Through the API, with Playwright's `APIRequestContext` (no extra library): sign in with the dev stub, create a questionnaire from a fixture JSON, publish it, and create a patient and episode.
  - Every name has a unique suffix, so runs never collide.
  - The UI is used only for the journey being tested.
- **Selectors:**
  - Use roles and labels (`get_by_role`, `get_by_label`). This also checks accessibility.
  - Add `data-testid` only where an element has no accessible name, and list those in `e2e/README.md`.
- **Journeys** (port the steps from `spikes/ui-workflow.ts`):

| ID | Journey | Viewport |
| --- | --- | --- |
| J1 | **Author:** create a questionnaire, add a Question Set, page and question, save, reload, and see it's kept. The preview shows the question. | Desktop |
| J2 | **Publish:** publish blocked by a logic problem, fix it, publish, "Create new version". | Desktop |
| J3 | **Patient:** the clinician creates an episode and copies the link. The patient opens it, fills every set (leaving part-way and coming back keeps the answers), and submits. The status is Ready for review. | **Phone (iPhone 13)** for the patient |
| J4 | **Clinician validates:** opens each set, corrects one answer (a "Patient answered…" note appears), validates every set, completes the review. The status is Ready for POA. | Desktop |
| J5 | **POA Summary:** both tabs. The corrected answer is marked on the validated tab. Change the episode status; a comment is added to General notes. The Print button exists. | Desktop |

- **Makefile:**
  - `make e2e-setup` creates the venv, installs the requirements and runs `playwright install chromium webkit`.
  - `make e2e` runs `pytest` in `e2e/` against `E2E_BASE_URL`. It expects `make dev` to be running.
- **Retire the old spikes:** delete `spikes/ui-workflow.ts` once J3–J5 cover it. Leave the other `spikes/ui-*.ts` files, which are local exploration scripts.

**Check:** `make e2e` passes headless three runs in a row (to catch flakiness). Breaking the patient submit button's label makes J3 fail with a readable trace.

## Step 4: docs

- `README.md` (root): a Tests section listing the three `make` targets and what each covers.
- [manual-test-plan.md](../manual-test-plan.md): for each manual test now automated by J1–J5, note "automated: J<n>".

## Status

- **Step 1 done 6 Oct 2026** as Step 0 of [auth first](auth-first.md) (not committed yet): `apps/api/internal/apitest`, `make test-integration`, `TEST_DATABASE_URL` in `.env.example`. As planned, except that the client is an `httptest.Server` with a cookie jar, and fixtures for trusts were added (`Org`, `HospitalIn`, `TrustUser`, `Member`).
- Steps 2–4: not started. The auth work added integration tests for its own routes and for sign-in and trust checks on every hospital route; the other five-case tests per route are still to do.
