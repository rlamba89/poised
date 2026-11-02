# S46 verification

## Verification 3 (2 Nov 2026)

**Verdict: Verified.** Verification 2's three findings and test report 5's Major are fixed, and test report 6 passed all 18 cases. The automated tests pass. My browser checks of tenancy, the patient API, the hard limits, the soft warning, a correction and the triggers all held. The code review found nothing that blocks a commit: six small points are below, for pending.md.

Reviewed: `e3a7d86` plus the uncommitted S46 work (41 changed files and 22 new ones, with `docs/market/` and the S47–S49 briefs). No code has changed since test report 6 (newest code file 31 Oct 22:20 local). The running `make dev` started at 22:33 local, after that change.

### Findings

None that block. `/code-review high` ran as a separate agent. I checked each of its points against the code and weighed them:

- **The audit's Cancelled is broad** (`apps/api/internal/httpapi/audit.go:222`). Any 5xx becomes 499 once the caller has gone, so a real database failure that happens just before a browser leaves is shown grey, not red. It's narrow, `serverError` still logs the cause, and Rahul chose Cancelled in report 5. *For pending.md:* record 499 only for a `context.Canceled` error, if it's ever confusing.
- **The episode's GET now fails if the hospital's settings can't be read** (`episodes.go:263`). It's one more query per page load, for `observationsOn`. Stored settings are validated when they're saved, so this is a theoretical failure. Not blocking.
- **`useObservations` has no guard against loads finishing out of order** (`ObservationsPanel.tsx:28`). Two quick saves could leave the older list on screen until the next change. It's unlikely locally, and more likely on a cold Lambda. *For pending.md.*
- **A time in the spring-forward gap is moved by an hour** (`apps/web/src/lib/localTime.ts:16`). If a clinician types 01:30 on the day the clocks go forward, it's stored as 02:30. A reading can't be edited, only corrected, so the wrong time stays. It's rare. *For pending.md.*
- **`features/settings/obsRanges.ts:5` imports `parseNumber` from `features/episodes/observations`.** It belongs in `src/lib`, beside `localTime.ts`. Tidying only.
- **Not taken up:** the review would merge per measure on the server rather than refuse a stale save (409). Rahul chose the refusal in report 5 (decisions.md, 31 Oct). The review also called the country display units more plumbing than a label needs. That was answer 3, agreed on 29 Oct.

### Checked and fine

- **Verification 2's fixes,** read in the code: `sameRanges` compares both sides as Save sends them, so key order no longer matters. `fromLocalInput` refuses a five-digit year, and `jobsTime` now uses the same helper. The POA Summary shows "Couldn't be loaded" instead of hiding the row. Report 5's stale-ranges 409 is `staleRanges`, run under the settings row lock, and it compares values, not text.
- **Tenancy, in real Chrome against `make dev`:** 14 attempts on Hospital A1's episode `01f28ab2…` and one of its readings all got 404, and nothing was written:
  - Val with A1's path and with her own A2 path: GET, add, correct and entered in error;
  - Bea with Trust A's path and with Trust B's own path;
  - the patient API's `/api/p/episodes/…/observations` (GET and POST).
- **As Cara, in the panel:**
  - Weight −5: refused with "Weight must be from 1 to 400 kg."
  - 380: "outside the usual range (35 to 200). Save it anyway?", then Save anyway. The Hospital BMI went to 124.1 with height 175.
  - Correct to 70 with a reason: the BMI went to 22.9. The 380 is struck through with "Corrected to 70.0 kg", and the reason, who and when are shown.
  - The same correction or an entered-in-error mark on the old reading, sent again: 409. A time an hour ahead: 400, "can't be in the future."
- **Triggers:** `UPDATE … value` and `DELETE` on that row are both refused by the database (each run in a transaction that was rolled back).
- **Routes:** every query in `observations.go` is keyed by an episode that `loadEpisode` checked under the hospital and trust. A correction reads the reading again under the episode lock before it inserts.
- **A-20:** the grep finds only the comments in `apps/web/src/lib/format.ts` and `apps/api/internal/patient/patient.go`, as before; S46 changed neither.
- **Security:** no secrets; audit rows hold IDs only; the routes are only on the staff router.
- **Docs:** the plan Status, `docs/plans/README.md`, `docs/sessions/README.md`, decisions.md (29–31 Oct), CLAUDE.md, AGENTS.md and pending.md are updated. `repocheck` passes.

### Test results

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 340, `apps/evaluate` 25, `packages/clinical` 485; CDK 132.
  - My first run used `make … DB_UP=` on one line, and the variable leaked into the CDK's Makefile test (`test_the_pipeline_can_skip_docker_compose`). Run on its own, the CDK suite passes 132 of 132. That was my mistake, not the code's.
- `make test-integration DB_UP=`: all 30 packages ok (`internal/httpapi` 25 s). `docker` isn't on this shell's PATH, so `DB_UP=` is needed; Postgres was already up.
- `make e2e` against `make dev`: **109 passed, 5 skipped** (the QA-only Cognito journeys) in 9 min 56 s, J12's 8 tests included.

### Test data

Episode `01f28ab2…` ("Rae MT S46 1704 Retest6", Hospital A1) now also has a 380 kg weight corrected to 70 kg ("Verifier V3: …"). The refused attempts are in the audit log. No settings were changed.

### Hand checks for Rahul

T-12 and the patient's view at 390×844 in real Chrome. The extension's window stops at 500 px; headless Chromium at that size passed.

### Suggested commits

The builder's four in [build-notes.md](build-notes.md#suggested-commits), with these additions, as nothing is committed yet:

- **Commit 1:** also `apps/api/internal/httpapi/{audit_test,audit_integration_test,episodes}.go` and `apps/api/internal/apitest/apitest.go`.
- **Commit 2:** also `apps/api/internal/httpapi/trust_settings.go` (the ranges and the stale-save 409).
- **Commit 3:** also `apps/web/src/lib/localTime{,.test}.ts`, `apps/web/src/features/qa/*`, `apps/web/src/features/audit/*` and `apps/web/AGENTS.md`.
- **Commit 4 (docs):** also `docs/pending.md` and this file.
- **Leave out** `docs/market/` (not S46).
- **Your choice:** the S47–S49 briefs could go in the docs commit, as the C7 split, or wait for their own sessions.

## Verification 2 (31 Oct 2026)

**Verdict: Needs fixes.** Verification 1's five findings are all fixed, and the automated tests pass. But the code review found a bug that silently undoes a trust's clinical limit, and a second where Save does nothing with no message. I reproduced both against the running app. All three findings below are small, and none needs a decision from Rahul.

**How independent this is:** the same Claude session ran test report 4 and this verification, so this isn't the separate reviewer the skill intends. To make up for that, I went back to the code and raw output rather than the report, ran `/code-review high` as a separate agent, and tested each of its findings. Report 4 passed T-18 and R-04 with F-1 present, because neither case has a competing edit; those results stand, but they prove less than they appear to.

Reviewed: `e3a7d86` plus the uncommitted S46 work (50 changed or new files). `make dev` was restarted at 15:15 today, and no code has changed since.

### Findings, worst first

**F-1 (Major): a stale Settings page silently undoes another admin's observation range.**
`apps/web/src/features/settings/SettingsPage.tsx:119` (`same`, at `:82`, compares JSON text).

Go stores the ranges as `jsonb`, which returns the keys as `hardMax, hardMin, softMax, softMin`; `rangesForSave` builds `softMin, softMax, hardMin, hardMax`. So the two never compare equal, and the ranges go in every PATCH once a trust has its own. Other settings aren't affected: an unchanged one isn't sent. `build-notes.md` says an unchanged table doesn't save again; it does.

Seen against `make dev`, as Ada:
- Save with no edits: a PATCH carrying the ranges, and "Trust A's settings were saved." instead of "Nothing has changed."
- A page stays open. A second tab changes heart rate Usual to from 110 to 105. The open page changes only "Mark not responding on day" and saves. Its PATCH holds `observations.ranges` as well, and heart rate goes **back to 110**, with no message.

So an admin can undo a clinical limit while changing something unrelated, and neither admin finds out. The unit test compares the draft's order with itself, so it can't catch this.

*Fix, test first:* compare both sides in one form, e.g. `same(value, rangesForSave(rangesDraft(own[d.key], d.default), d.default))` for `obsRanges`. A unit test with the keys in `jsonb` order, and a J12 step: Save with no edits says "Nothing has changed."

**F-2 (Minor): a five-digit year makes Save do nothing, with no message, and the reading isn't saved.**
`apps/web/src/features/episodes/observations.ts:118` (`fromLocalInput`); called before any `try` from `ObservationsPanel.tsx:124`.

Seen in real Chrome as Cara: weight 80, Measured at year typed as 20266 (the box keeps `20266-10-31T15:52`), Save reading. An uncaught `RangeError: Invalid time value`, no request, no message, no warning, and the form sits there as it was. A clinician may walk away thinking it saved. It's the one path where a mistyped time gets no answer: T-15's 1926 and 0026 are refused properly.

*Fix, test first:* `readingRequest` returns an error for a time that isn't valid ("When was it measured? Enter the date and time." is already Go's wording). `features/qa/jobsTime.ts` has the same pair of helpers and already checks this: move them to `src/lib` and use them, rather than keeping two copies.

**F-3 (Minor): when the readings fail to load, the POA Summary hides the Hospital BMI with no message.**
`apps/web/src/features/episodes/PoaSummary.tsx:44` keeps only `.view`; `:157` then shows no row. Read from the code, not reproduced: my tools can't block a request.

A 500 or a dropped connection looks exactly like observations being turned off, so a clinician reviewing the summary can't tell that a measured BMI exists but didn't load. S47 will feed this BMI into scores.

*Fix, test first:* keep `.error`, and show it in the row (for example "Couldn't be loaded") instead of hiding the row.

### Not blocking (for pending.md)

- **With observations off, each page open writes a refused audit row.** Seen in the audit table: 4 `observation.list` 403 rows from T-09's two page opens, doubled by dev's effects, so 1 per open in production. `useObservations` always calls the route. The audit log would fill with failures for a feature the hospital doesn't have. Sending the switch with `GET …/episodes/:id` would stop it.
- **Reasons are limited in bytes, not characters** (`statuses.go:195`, `:107`; not S46's code, but S46's routes use it). The box and the database count characters, so about 340 characters of a three-byte script is refused as "too long (1,000 characters at most)". It only matters once a country uses a non-Latin script (A-20).
- **Tidying, if the builder is in the file anyway:** `lockedObservations` (`observations.go:273`) repeats `s.inTx`; the `n == 0` branch at `:199` can't run under the lock; the "Save it anyway?" block is written twice in `ObservationsPanel.tsx`; `ObsRanges` in `settings.go:265` parses the JSON twice.
- Test report 4's three small points are already in pending.md.

### Checked and fine

- **Verification 1's findings:** F-1, F-2 and F-4 by test report 4 and J12. F-3 by its integration test plus 6 corrections at once in report 4 (one 201, five 409s). F-5 by its unit tests only (mixed offsets; not tried by hand).
- **Scope:** everything in the brief is built, and the answers are followed. Beyond the brief: a trust's ranges (Rahul's answer 1) and the date of birth limit (Rahul's request).
- **Tests first:** the four routes have the five cases, plus same-moment corrections, the triggers and audit rows.
- **Tenant isolation:** every query is keyed by an `episode_id` that `GetEpisode` has checked. Tried in real Chrome as Val, with her **own** hospital's path and Hospital A1's episode and reading IDs: GET, add, correct and entered in error all 404, nothing written. With A1's path: the page says "You don't have access to this hospital."
- **Patient side:** the routes are only in `staffRoutes`; nothing in the patient handlers or `internal/chapter` refers to observations.
- **Deleting an episode:** with a chain of corrections, the rows go with it (3 to 0, in a transaction rolled back), so the self-references and the no-delete trigger don't block retention. `login_links` and `patient_sessions` have no cascade, which isn't S46's.
- **The logic:** `Check`, `CheckRanges`, `counting`, `latest`, `hospitalBmi`, `outsideUsual` and the migration's triggers, read line by line. `reportedBmi` and the HQ's `bmi()` now share `bmiOf`, with the same rounding.
- **A-20:** the grep finds only `apps/web/src/lib/format.ts` and `apps/api/internal/patient/patient.go` outside config, seed and test files; S46 changed neither.
- **Security:** no secrets; no values in audit rows.
- **Docs:** plan Status, `docs/plans/README.md`, `docs/sessions/README.md`, decisions.md, CLAUDE.md, AGENTS.md and pending.md are updated. `repocheck` passes. `build-notes.md` needs the correction in F-1.

### Test results

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 328, `apps/evaluate` 25, `packages/clinical` 485; CDK 132.
- `make test-integration DB_UP=`: all 30 packages ok (`internal/httpapi` 31 s, not cached). `DB_UP=` because `docker` isn't on this shell's PATH; Postgres was already up.
- `make e2e`: **104 passed, 1 failed, 5 skipped** in 10 min, J12's 4 tests included. The failure is J7, the known flake (pending.md): 3 captured texts to a reused made-up mobile instead of 1. It passed 3 of 3 alone.

Browser spot-checks: F-2 and tenancy in real Chrome, 1280 px. F-1 in headless Chromium, because it needs two pages open with a saved order of events.

### Test data and settings

Made-up episode "Rae MT S46 V2 1600" at Hospital A1, with 3 weight readings and the refused tenancy attempts in the audit log. Trust A's settings are back to `{}`, checked afterwards.

### Hand checks for Rahul

T-12 and the patient's view at 390×844 in real Chrome (the extension's window stops at 500 px; headless Chromium at that size passed).

### Suggested commits (after the fixes)

The builder's four in [build-notes.md](build-notes.md) stand; these fixes are all web code, so they go into commit 3, since none of it is committed. `docs/market/` isn't part of S46: leave it out.

### What to retest

F-1 and F-2 with new test cases; F-3 if it can be forced; T-10, T-13, T-15, T-18 and R-04 as regressions.

## Verification 1 (30 Oct 2026)

**Verdict: Needs fixes.** Two findings need work before commit: one is a gap in a Must requirement (OBS-03), the other needs Rahul's decision. Three more are small and sit in code S47 builds on. Tenancy, audit, the patient's side, the triggers and A-20 all hold, and the automated tests pass.

Reviewed: `e3a7d86` plus the uncommitted S46 work (50 changed or new files). Only a pytest cache has changed since test report 3, so the servers running matched the code reviewed.

### Findings, worst first

**F-1 (Major): a correction never gets the soft warning (OBS-03).**
`apps/web/src/features/episodes/ObservationsPanel.tsx:210-214` posts straight away; only `AddReading` calls `warningText` (`:129-134`).

Seen in Chrome as Cara: weight 80 kg, **Correct**, 380 (a mistyped 80), a reason, Enter. It saved with no "Save it anyway?", and the Hospital BMI became **117.3**. The same value as a new reading asks first. A correction is a value typed by hand, as likely to be mistyped as a new one, and S47 will feed this BMI into scores. T-04 and J12 test only the warning on a new reading.

*Fix:* the same warning and **Save anyway** on the Correct form, with a J12 step. Hard limits stay Go's.

**F-2 (Major, Rahul's decision): marking a correction entered in error leaves nothing counting, unlike the ASA grades.**
`packages/clinical/src/observations.ts:38` (`counting`); migration 00037 `:52` (a superseded row can't change).

Seen in Chrome, carrying on from F-1: mark the 380 kg correction entered in error. Both rows are now struck through, the 80 kg row has no buttons, and the Hospital BMI says **"Not measured yet"**, though 80 kg was the right value.

The S44b ASA grades work differently: when one is entered in error, "the one before it counts again" (decisions.md, S44's answers). S46's answer 2 says "as S44b does", so this is either a difference nobody decided or an unrecorded choice. It's in neither the build notes nor the test plan.

**Question for Rahul.** When a correction is marked entered in error, should the value it replaced count again?
- **(b) Recommended: no, and the form says so.** The Entered in error form on a correction adds: "The value it corrected won't count again. To put a value back, use Correct." Nothing changes in the table. The reason to differ from the ASA grades: an older grade was a valid judgement, but a corrected reading is one a clinician has already said was wrong, so it shouldn't come back without anyone choosing it. Record it in decisions.md.
- **(a) Yes, as the ASA grades.** The old reading counts again and gets its buttons back. This changes `counting`, `loadObservation`, both update queries, the trigger and the unique index on `corrects` (a reinstated reading must be correctable again), and a value someone called wrong silently feeds the BMI again.

**F-3 (Minor): two corrections at once answer 500, not 409.**
`apps/api/internal/httpapi/observations.go:156` checks "still counts" before the lock; `:186` inserts before `:190` supersedes, so the loser hits `observations_corrects_idx`.

Measured against `make dev`: 30 rounds of 6 simultaneous corrections of one reading gave **30 × 201, 147 × 500, 3 × 409**. The data was right every time (one winner per round), but the 409 branch at `:191` almost never runs, and the panel reloads only on a 409 (`ObservationsPanel.tsx:228`), so the second clinician gets "Something went wrong" over a stale list. The sequential test at `observations_integration_test.go:369` can't reach this.

*Fix:* re-read the reading under the lock, as `markAnaesthetistGradeInError` does (`asa_grades.go:159`), before the insert. A concurrent integration test first.

**F-4 (Minor): each POA tab switch reloads the readings and writes audit rows.**
`apps/web/src/features/episodes/PoaSummary.tsx:124`: `useObservations` sits in `Summary`, which is keyed by tab (`:88`). Seen in Chrome: two tab switches gave 4 GETs (2 per switch in dev's double effects, so 1 in production), each an "Opened an episode's observations" row, and the Hospital BMI line is missing until the reply comes. This is why test report 3 counted 19 such rows. The audit log is a compliance record, so it shouldn't fill up with tab clicks.

*Fix:* call the hook in the parent that renders the Tabs, and pass the result down.

**F-5 (Minor): readings' times are compared as strings.**
`packages/clinical/src/observations.ts:43,56`; `apps/web/src/features/episodes/observations.ts:49`.

The API's strings aren't uniform: seen locally, `2026-10-20T11:00:00+01:00` beside `…Z`, and Go trims a fraction's zeros. So string order isn't time order across a clock change on a non-UTC host, or with fractions of a second. On Lambda (UTC), with the panel's whole minutes, it won't show. It costs two lines to fix now, and S47's pre-fill uses `latest`.

*Fix:* compare `Date.parse`, with a unit test holding mixed offsets.

### Not blocking (added to pending.md if Rahul agrees; no code change needed now)

- **The ranges editor always saves all ten measures** (builder's choice 8). So a trust that changes one limit keeps today's made-up defaults for the other nine, including when the clinician-checked values ship. Worth deciding at C10: `obsRanges.ts:30` could send only the rows that differ.
- **"Measured at" has no earlier limit:** 0026-10-30 and 1926-10-30 both save (201), and the panel's Correct can't change a time, so the way out is Entered in error and a new reading. The reading sits at the bottom of the list, and the BMI goes on using any newer-dated weight. A floor such as the patient's date of birth would stop it.
- **Tidying, while in that file:** `setKind(kind)` at `ObservationsPanel.tsx:140` does nothing, and `gone` (`:190`) is `!counts`.

### Checked and fine

- **Scope:** everything in the brief is built, and the answers are followed (a trust's ranges; Correct and Entered in error; country units).
- **Tests first:** all four routes have the five cases, plus hard limits, the trust's own limits, cancelled and archived, observations off, 409s, the triggers and the audit rows. Logic has unit tests. J12 is in `e2e/`.
- **Tenant isolation:** every observation query takes an `episode_id` that `GetEpisode` has already checked against the hospital, and `requireHospital` checks the membership. As Val and as Bea, in Chrome: GET and POST 404, and the page says "You don't have access to this hospital."
- **Patient side:** the routes are only in `staffRoutes`.
- **A-20:** the grep finds only `apps/web/src/lib/format.ts` and `apps/api/internal/patient/patient.go` outside config, seed and test files, and S46 changed neither. Units come from config. The two new variables are set everywhere strict config needs them (`infra/api_stack.py`, `countries.py`).
- **Security:** no secrets; no values in audit rows; logged database errors don't carry row contents.
- **Docs:** plan Status, `docs/plans/README.md`, `docs/sessions/README.md`, decisions.md, CLAUDE.md, AGENTS.md and pending.md are updated. `repocheck` passes.
- **F-1 to F-4 of the test reports:** re-checked by J12. No console errors in my browser run.

### Test results

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 325, `apps/evaluate` 25, `packages/clinical` 482; CDK 132.
- `make test-integration`: all packages ok (`internal/httpapi` 23 s, not cached).
- `make e2e`: **104 passed, 5 skipped** (the QA-only Cognito journeys) in 9 min 44 s, J12's 3 tests included.

Browser spot-checks (Claude in Chrome, 1456×829, as Cara, Val and Bea): F-1, F-2 and F-4 above, and tenancy. API checks (Python against `make dev`): F-3 and the old dates. Test data: made-up "Verify S46 …" episodes at Hospital A1. No settings were changed.

### Hand checks for Rahul

T-12 and the patient's view at 390×844 in real Chrome (from test report 3: the extension's window stops at 500 px; headless Chromium at that size passed).

### Suggested commits (after the fixes)

The builder's four in [build-notes.md](build-notes.md) stand. The fixes go into those commits, since none of it is committed yet. `docs/market/` isn't part of S46: leave it out.
