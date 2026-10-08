# S01 build notes

**Builder session:** 8 Oct 2026. **Status:** verified (8 Oct, [verify.md](verify.md)). Report 1 passed all 19 cases.

## Before building

- **Questions:** the brief has none.
- **Uncommitted planning docs:** Rahul chose to commit them first, without `reports/` and `research_notes/` (still untracked; an open item in decisions §5). Commit `223e69b`.
- **Docker:** running (`poised-db-1`, Postgres 17). The `docker` CLI isn't on the PATH of Claude's shell (it's in `/Applications/Docker.app/Contents/Resources/bin`), so Claude prefixes that when it runs `make test-integration`. Rahul's own terminal is fine.
- **The local `sj` database was on the `dev` branch's schema** (migrations 5–8 from `557be2c` "Auth first…", applied 7 Oct), so master's `make seed` and `make dev` couldn't work. Rahul said to break and rebuild it. Done on 8 Oct:
  - full backup: `pg_dump -Fc` in Claude's scratchpad (`sj-dev-branch-2026-10-08.dump`; temporary, lost on reboot);
  - `sj` dropped, recreated, `make migrate seed`;
  - **HJE Full HQ** copied back (questionnaire, version, 4 Question Sets; same ids). It was the only questionnaire. There were no episodes.
- **Stale Next.js types:** `apps/web/.next/dev/types/` (generated, gitignored) still listed the `dev` branch's routes (`/o/[orgId]/…`, `/auth/callback`), so `tsc` in `make lint` failed. Claude deleted that folder; `next dev` regenerates it.

## Steps

1. [x] **F1 Step 1, harness:** `internal/apitest`, the first test `GET /api/health`, `make test-integration`, `TEST_DATABASE_URL`.
2. [x] **Questionnaires, versions, publish:** five cases for each of the 6 routes.
3. [x] **Chapters (Question Sets):** five cases for each of the 6 routes.
4. [x] **Option lists, codes, categories:** five cases for each of the 6 routes.
5. [x] **Two bugs found, each fixed test-first** (below).
6. [x] `make lint test test-integration` pass; the "Done when" checks; test plan written.

## How the harness works (for Rahul, in Go terms)

All of it is behind `//go:build integration`, so `make test` (no tag) never compiles it and needs no database.

- **`apitest.Main(m)`**, called from a package's `TestMain`:
  - connects to `TEST_DATABASE_URL` (the compose Postgres's `postgres` database);
  - takes a Postgres **advisory lock**, held until the package's tests finish;
  - finds `apps/api` by looking for `go.mod` above the package folder (this works under `-trimpath`);
  - drops test databases left behind by killed runs (`sj_test_` + 16 hex characters);
  - drops and recreates `sj_test_template`, then runs every goose migration into it with goose as a library (`goose.NewProvider(...).Up`).
  - If anything fails, no test runs and it prints why. A broken migration prints its file name and the Postgres error.
- **`apitest.New(t)`** returns `(c *Client, fx *Fixtures)`:
  - `CREATE DATABASE sj_test_<random> TEMPLATE sj_test_template`, a pool to it (max 4 connections), and the real `httpapi.NewRouter` behind an `httptest.Server`;
  - `t.Cleanup` closes the server and pool and drops the database.
- **Client:**
  - `c` isn't signed in;
  - `c.LoginAs(t, user)` signs in through the real `POST /api/dev/login` and returns a **new** client that keeps the cookie, so a test can hold several users at once;
  - `c.Do(method, path, body)` returns a `*Resp` (`Code`, `Body`). A string body is sent as is, for broken JSON. If the server drops the connection, `Code` is 0 and `Body` holds the error;
  - `r.Want(t, code)`, `r.Decode(t, &v)` and `r.Message()` (the `{"error"}` text);
  - `c.DB` is the test database's pool, for asserting rows with plain SQL.
- **Fixtures** (`fx`) insert rows directly:
  - `Hospital(t)`;
  - `User(t, hospital, roles...)`;
  - `DraftHQ(t, hospital, by, file)` and `PublishedHQ(t, hospital, by, file)`, made from `apitest/testdata/<file>`;
  - `Episode(t, hq, by)`, which also adds a made-up patient.
  - `testdata/basic.json` has two Question Sets (patient and clinician) with stable IDs (`q_smokes`, `o_nevr`…).
- **Route tests** live next to the routes, in `internal/httpapi/*_integration_test.go` (package `httpapi_test`).
  - `newCast(t)` sets up hospital A with Alex (author + publisher) and a viewer, and hospital B with Bea (author).
  - `refused(t, r, code)` also checks there's a plain-language message.

## Differences from the plan (F1 Step 1)

- **`LoginAs` takes `t`:** `c.LoginAs(t, user)`, not `c.LoginAs(user)`. A failure in a subtest must fail the subtest, not its parent.
- **Test databases are named `sj_test_<random>`,** not `t_<random>`, so the leftover sweep only matches names the harness makes.
- **Fixture signatures:** `DraftHQ` and `PublishedHQ` take the creating user and a testdata file name. The plan had `fx.PublishedHQ(t, hospital, contentJSON)`, but `created_by` needs a user, and one file holds several Question Sets. `DraftHQ` is new, because most chapter tests need a draft. `Episode(t, hq, by)` isn't used by a route test until S02; `TestFixtures` (in `apitest`) checks the API can read every fixture's rows.
- **The advisory lock and the leftover sweep** aren't in the plan. Without the lock, `go test ./...` runs `apitest` and `httpapi` in parallel, and each would drop the other's template. The sweep stops killed runs from filling the disk.
- **`make test-integration` runs `go test -tags integration -count=1 ./...`.** `-count=1` stops Go reusing cached results, which don't know the database changed.
- **`make lint` now vets with `-tags integration`,** so the integration test files are vetted too. It still needs no database.
- **`TEST_DATABASE_URL` has a default in the Makefile.** Rahul's existing `.env` predates it, and `.env` is only copied from `.env.example` when it's missing.

## How the five cases apply

Every route in the three groups has all five, except where a case can't exist:

| Route kind | "Wrong role" case | "Another hospital" case |
| --- | --- | --- |
| Writes | viewer (and author-without-publisher for publish) → 403 | Bea at A's URL → 404, and Bea at B's URL with A's id → 404 |
| Reads (`GET`) | asserts a **viewer can read** (no role is needed) | same as writes |
| Codes, categories | any signed-in user reads; Bea gets the same results | shared reference data: no hospital copy to reach |

- **Categories take no input,** so there's no invalid-input case.
- **Refusals also assert nothing was written:** row counts, statuses and revisions are unchanged.
- **Auth routes** (`/api/me`, `/api/dev/users`, `/api/dev/login`, `/api/logout`) aren't in S01's groups. S02's "Done when" covers every route in `router.go`, so they belong there. `/api/me` and `/api/dev/login` are already exercised by `TestFixtures` and `LoginAs`.

## Bugs found (recorded in F1 Status too)

1. **`GET /api/h/{hid}/questionnaires?page=99999999999` → 500.**
   - **Cause:** `(page-1)*20` overflowed `int32` and Postgres refused a negative OFFSET.
   - **Fix:** `pageNumber()` clamps the page to the last one whose offset fits. It's an empty page, not an error.
   - **Unit test:** `TestPageNumber`. Integration test: the `TestListQuestionnaires/invalid_input` case.
   - **Not reachable from the UI,** which keeps the page in React state. T-12 checks it in the browser address bar.
2. **`PATCH /api/h/{hid}/chapters/{cid}` with a field set to `null` crashed the handler.**
   - **Cause:** a nil pointer dereference. Go's server recovered, but it dropped the connection with no response.
   - **Fix:** the details became a `chapterDetails` type with `validate()`, like `questionnaireInput`. A null field → 400 "The request could not be read."
   - **Unit test:** `TestChapterDetailsValidate`. Integration test: the `TestUpdateChapter/invalid_input` case.
   - **Not reachable from the UI,** which never sends null. T-13 checks it from the DevTools console.

## Checks done

- **A broken migration** (a temporary `00005_broken.sql` with a syntax error) makes `TestMain` stop every test with:
  ```
  apitest: a migration failed, so no integration test can run.
  db/migrations/00005_broken.sql: ERROR: syntax error at or near ";" (SQLSTATE 42601)
  FAIL	github.com/rlamba89/poised/apps/api/internal/httpapi
  ```
  Reverted.
- **Other setup failures are clear too:**
  - no `TEST_DATABASE_URL` → "TEST_DATABASE_URL is not set. Run `make test-integration`, which sets it";
  - Postgres down → "can't connect to Postgres at localhost:5999 … Is it running? Try `docker compose up -d db`".
- **The tests catch real regressions.** Each break was temporary, made in generated code, and reverted:
  - no revision check on save → `TestSaveChapterContent` fails;
  - publishing no longer retires the old version → `TestPublishQuestionnaire` fails;
  - `GetQuestionnaire` not scoped to the hospital → the "another hospital" cases of 5 tests fail.
- **`make test` runs without a database:** `go test -count=1 ./...` passes with `DATABASE_URL` and `TEST_DATABASE_URL` pointing at a closed port.
- **Both fixes, live:** checked against the real API on a spare port with the dev database. `?page=99999999999` → `{"items":[],"page":107374182,…}`; a null `name` → 400, and the chapter keeps its name.

## Test results (8 Oct)

`make lint test test-integration` (exit 0):

```
go vet -tags integration ./...                       (no findings)
tsc --noEmit                                         (no errors, after removing the stale .next/dev/types)
make test:  ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 13 passed (13), Tests 103 passed (103)
make test-integration:
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	0.675s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	0.784s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	0.478s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	0.165s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	2.276s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	0.616s
```

- **20 integration tests:** `TestFixtures`, `TestHealth`, and 18 route tests, each with subtests for the five cases.
- **About 3 s** for the whole integration run.
- **Two names repeat** between the old fake-querier tests (package `httpapi`) and the new ones (package `httpapi_test`): `TestCreateOptionList` and `TestCreateVersion`. Go allows it, since they're different packages, and `-run` matches both. The old ones go when their area is next changed (testing rule 4).

## Code review (8 Oct, before manual testing)

Rahul asked for a review before the test plan ran. `/code-review` gave 15 findings.

**Fixed** (test and harness code only; no app behaviour changed):
- **`-trimpath` broke the harness:** `runtime.Caller` gives a module path, not a folder, so no migrations were found and every test failed. `moduleDir()` now looks for `go.mod` above the working directory. Checked with `go test -trimpath -tags integration`: it failed before and passes now.
- **`LoginAs` and four helper closures used the parent test's `t` inside subtests,** so a failure would have been reported against the wrong test. They now take `t`.
- **`listRow` ignored query errors,** so a broken query would have made the "refusals changed nothing" check pass without checking anything. It now fails the test.
- **The test-database prefix is now `sj_test_`.**
- **Smaller fixes:**
  - the `pageNumber` comment was off by one;
  - the chapter PATCH builds `chapterDetails` with field names, not positions;
  - `TestCreateVersion` reuses the `firstQuestion` helper;
  - `make test-integration` depends on `db-up` instead of repeating its command.

**Rahul decided:**
- **`PATCH /episodes/{id}` with `{"status": null}` (or `procedure`, `anaesthetic`, `consultant`) crashes in the same way as bug 2.** S02 fixes it test-first with that route's integration tests, plus a small recover middleware, so any handler panic becomes a logged, plain-language 500. It's recorded in the S02 brief and F1 Status.
- **The five-case convention for reference data** (codes, categories) is agreed (decisions.md, 8 Oct).

**Not changed, with reasons:**
- **Cast the OFFSET to `bigint` instead of clamping.** It would still need a clamp: `?page=` at the int64 maximum overflows `(page-1)*20` too.
- **Plain `string` fields in `chapterDetails`, so `null` keeps the old value.** That would accept a `null` silently (a 204 that changes nothing). Refusing it with a 400 is explicit, in line with "nothing may break silently".
- **The lock runs integration packages one after another,** and each builds its own template. With 2 packages, the run takes about 3 s. Running them in parallel needs per-package templates and sweeps. Revisit if it gets slow.
- **A `.env` value of `TEST_DATABASE_URL` beats a shell variable.** `DATABASE_URL` already behaves the same way. To point elsewhere, use `make test-integration TEST_DATABASE_URL=…`.
- **`-tags integration ./...` also re-runs the unit tests,** and with the tag the `httpapi` unit tests need Postgres. It costs about 1 s, and `./...` picks up new integration packages without anyone editing the Makefile. `make test` stays the database-free run.
- **The sweep has no "is this a local server?" check.** The F4 pipeline may use a database service that isn't on localhost. The `sj_test_` prefix limits what it can match.

## Test report 1 (8 Oct): all passed

[test-report-1.md](test-report-1.md): 15 test cases and 4 regression cases passed, none failed or blocked. No code changed after it.

- **What the tester found while exploring** is all older than S01, which changed no web code and no list query. It's out of S01's scope ("no change to app behaviour, apart from bug fixes"), so it's recorded in [pending.md](../../pending.md) §3 instead of being fixed here: the stale delete-dialog error, `total` 0 past the last page, dead-end error pages, "chapter" in user-facing text, and small items.
- **Tester gotchas, recorded in decisions.md §4:**
  - the tester's shell also lacks `docker` on its PATH;
  - the Claude in Chrome network log shows 204 responses as 503.
- **Not tested:** reloading in the middle of an unsaved edit. The native "leave page?" prompt freezes the browser tool. It isn't affected by S01.

## Suggested commits

In this order. Each one passes `make test` and `make test-integration` on its own. The code-review fixes are already folded into commits 1, 3 and 4.

1. **Add the integration test harness and make test-integration (F1 Step 1)**
   - `apps/api/internal/apitest/` (`apitest.go`, `fixtures.go`, `apitest_test.go`, `testdata/basic.json`);
   - `apps/api/internal/httpapi/main_integration_test.go` (`TestMain` + `TestHealth`);
   - `Makefile`, `.env.example`, `apps/api/go.mod`.
2. **Fix a 500 on a huge ?page= in the questionnaire list**
   - `apps/api/internal/httpapi/questionnaires.go` and `questionnaires_test.go`.
3. **Fix a crash when a chapter PATCH sends a null field**
   - `apps/api/internal/httpapi/chapters.go` and `chapters_test.go`.
4. **Integration tests for the authoring routes (F1 Step 2, part 1 of 2)**
   - `apps/api/internal/httpapi/` `cast_`, `questionnaires_`, `chapters_`, `optionlists_` and `codes_integration_test.go`.
5. **Docs: S01 build notes, test plan and status**
   - `docs/sessions/S01-integration-harness/` (including `test-report-1.md`), `docs/sessions/README.md`, `docs/pending.md`, `docs/sessions/S02-workflow-route-tests/brief.md` (the episode crash);
   - `docs/plans/README.md`, `docs/plans/f1-test-foundation.md`;
   - `docs/decisions.md`, `CLAUDE.md`.

## TypeScript

None added or changed.
