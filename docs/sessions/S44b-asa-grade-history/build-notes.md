# S44b build notes

Builder, automatic run, 27 Oct 2026. The brief's questions were answered on 27 Oct ([answers.md](answers.md)); none were open.

## What was built

1. **Migration 00035** (`apps/api/db/migrations/00035_anaesthetist_asa_grades.sql`): the table `anaesthetist_asa_grades` (grade 1–6, `given_by`, `given_at`, `reason`, and `error_by`/`error_at`/`error_reason`, all three or none). Every existing `episodes.anaesthetist_asa` is carried over as the episode's first grade with no user ("Recorded before the history"); its time is that of the General notes line "Anaesthetist ASA grade set to …", else the migration's. **The column is dropped**: nothing needs to read it, because the grade that counts is worked out from the table. A trigger refuses any change to a row except marking it entered in error once. Down puts the column back with the grade that counts. Checked on the local database: 12 graded episodes carried over exactly, and Down then Up gave the same grades.
2. **Queries** (`db/queries/asa_grades.sql`, `episodes.sql`): `GetEpisode` gives the grade that counts as `anaesthetist_asa` (a join on the latest grade not entered in error), so the episode's JSON is unchanged. `LockEpisodeStatus` no longer reads the grade; `CountingAnaesthetistGrade` reads it after the lock, in its own statement (see below). `UpdateEpisode` no longer writes it.
3. **Routes** (`internal/httpapi/asa_grades.go`, `router.go`), both audited (`asa.add`, `asa.error`, IDs only), for any clinician at the hospital (who could set the grade before):
   - `POST …/episodes/{eid}/anaesthetist-asa` `{grade, reason, revision}` → 201 `{id}`. 400 for a grade outside 1–6, no revision or a reason over 1,000 characters; **422** "Say why the grade is changed." with no reason while a grade counts; 409 stale; 409 for a cancelled or archived episode.
   - `POST …/episodes/{eid}/anaesthetist-asa/{gid}/entered-in-error` `{reason, revision}` → 204. 400 with no reason; 404 for a grade not this episode's; 409 "This grade is already marked entered in error." (it can't be unmarked); **409 "Add the correct grade first, or move the episode back from Ready for admission."** when it's the only grade that counts, a rule requires the grade, and the episode is at Ready for admission, Admitted or Discharged.
   - **No edit or delete route.** The episode PATCH refuses `anaesthetistAsa` (400, even `null`); the nurse's grade stays on the PATCH.
   - `GET …/episodes/{eid}` adds `anaesthetistAsaGrades` (oldest first).
4. **S43's requirement reads the grade that counts:** the status moves (`applyStatus`), the status control's blocked moves and Undo (`statusView`, from `GetEpisode`) and auto-triage on send (`patient.go`).
5. **The POA Summary** (`apps/web/src/features/episodes/AnaesthetistGrades.tsx`, `asaGrades.ts`, `PoaSummary.tsx`): the grade that counts, the history newest first, **Add grade**, and **Entered in error** on each standing grade. The top details' "Anaesthetist ASA grade" and the requirement lines (episode page and POA Summary) use the grade that counts, as before.
6. **E2E:** J11b now adds ASA III, re-grades to ASA IV with a reason, marks IV entered in error, is refused marking the only grade at Ready for admission, and admits. `test_statuses.py`'s two-tab journey adds the anaesthetist's grade through the new form.
7. **Docs:** pending.md (the S43 item struck through), decisions.md (27 Oct, S44b), CLAUDE.md (one content-model line), `apps/web/AGENTS.md`, the C6 plan's Status and the plans and sessions tables.

## Decided by the builder (automatic mode)

- **The column is dropped**, not kept: `GetEpisode` works the grade out, so nothing reads the column.
- **The locked paths read the grade after the lock, in a statement of their own** (`countingGrade`). If the locking statement joined the grades, a request that waited for the lock could see the grades as they were before the other request added or marked one (PostgreSQL re-reads only the locked row). Every grade write takes the same lock and bumps the revision, so status moves and grade writes are applied one after the other.
- **A database trigger refuses edits** (any change but marking entered in error once), in addition to the API having no edit route. "Never changed" is the point of the session, so it holds whatever a later query does.
- ~~**A reason is required while a grade counts.**~~ *Reversed in verification 1:* a reason is required for every grade after the episode's first (Rahul's answer 2).
- **Ready for admission or later** = Ready for admission, Admitted, Discharged. On hold and Not ready aren't "later": there the only grade can be marked, and the move back to Ready for admission is refused until a grade counts (S43's check).
- **A grade with the same number as the one that counts is accepted** (with a reason), e.g. confirmed after a new test.
- **General notes:** "Anaesthetist ASA grade set to 3" (S20's wording, for the first; a reason, if given, is added after a colon), "Anaesthetist ASA grade changed from 3 to 4: New echo result", "Anaesthetist ASA grade 4 marked entered in error: Wrong patient"; the same grade again reads "confirmed as 3" (test report 1).
- **The history is newest first**, so the latest sits under the grade that counts. The grade that counts carries a **Counts** badge.
- **Words on the page:** "Anaesthetist ASA grade: ASA III" (or "No grade yet"); each entry "Given by Cara Clinician on …[: reason]" or "Recorded before the history (…)"; a grade entered in error is struck through, badged "Entered in error", with "Entered in error: marked by … on …: why". The reason box is "Why is the grade changed?"; the error form's is "Why was it entered in error?".
- **The "Entered in error" button shows on every standing grade**, including the one the server will refuse; the refusal says why, in the form (manual test focus 3).
- **Go helper:** `changeEpisode` (tasks and flags) now wraps `changeLockedEpisode`, whose change sees the episode as locked and words its own General notes line.
- **Audit labels** "Added the anaesthetist's ASA grade" and "Marked an anaesthetist's ASA grade entered in error".

## TypeScript for Rahul to review

- `apps/web/src/features/episodes/asaGrades.ts`: the grade type as Go sends it, `countingGrade` (the latest not entered in error; the page uses Go's `anaesthetistAsa` for the top line and this for the **Counts** badge), the history's words, and `addGradeRequest` (what the Add form sends, or what's missing). Unit-tested in `asaGrades.test.ts`.
- `AnaesthetistGrades.tsx`: the section itself: two small inline forms, each sending the revision the page loaded; a stale refusal keeps what was typed and offers Reload. After a Reload, a grade someone else marked closes its open form.
- `PoaSummary.tsx`: `AsaGrades` keeps only the nurse's Select and Save, then renders `AnaesthetistGrades`.

## Tests

- **TDD:** the integration tests (`asa_grades_integration_test.go`) and `asaGrades.test.ts` were written before the code. The web test was run red first; the Go tests didn't compile until the routes existed and then passed.
- New integration tests: `TestAddAnaesthetistGrade`, `TestMarkAnaesthetistGradeInError` (each route's five cases plus stale, cancelled/archived, the PATCH refusal, no edit or delete routes, and the trigger), `TestTheRequiredGradeCantGoMissing` (the requirement at Ready for admission, Admitted and Discharged; an earlier grade to fall back on; Admit then Undo; cancelled; before Ready for admission; no rule).
- Changed: `TestUpdateEpisode*` (the PATCH takes only the nurse's grade), `TestTwoTabsDontOverwriteEachOther`, `TestReadyForAdmissionNeedsTheAnaesthetistsASA` (the grade added and marked through the new routes).
- Results (27 Oct):
  - `make lint test`: exit 0. Vitest 298 + 13 + 444 passed; CDK 124 passed; Go all ok.
  - `make test-integration`: exit 0 (every package ok; httpapi 31.4 s).
  - `make e2e` against `make dev`: **98 passed, 5 skipped** in 7 min 10 s (the 5 skipped are the AWS-only journeys, as before).

## Test report 1 (18 of 19 passed; F1 cosmetic)

- **F1, the "Entered in error" button cut to "Entered in err" at about 500 px** beside a wrapped "Given by …" line: fixed. The button no longer shrinks (`flexShrink: 0`) and the text beside it may (`minWidth: 0`). Test first: J11b now checks at a 500 px viewport that no history button's label is clipped. It failed, then passed after the fix.
- **Exploring note 1, "changed from 3 to 3":** a re-grade with the same grade as the one that counts is still accepted (with a reason; my earlier choice, e.g. a second anaesthetist agreeing). The note now reads "Anaesthetist ASA grade **confirmed as** 3: <reason>". Test first: `TestAddAnaesthetistGrade` "the same grade again…". Whether to refuse it instead is a small product question, added to pending.md for Rahul.
- **Exploring note 2, "No ASA grades have been added." above a struck-through history:** it now shows only when there's no nurse grade **and** no anaesthetist's grade in the history. The anaesthetist's own line already says "No grade yet". This is cosmetic, with no automated test (JSX only); T-08 and T-13 of the test plan now check it.
- **Exploring note 7 (print shows the empty nurse dropdown):** older than S44b, not changed.
- **Not S44b's, but it made `make e2e` red once:** `test_audit.py`'s "a list row names ten patients" hovered a row of the *unfiltered* list just before the filtered one replaced it, so no tooltip showed. It failed 2 of 3 runs alone once the local audit log had grown. The journey now waits for the filtered list's response before hovering, and passed 4 of 4 runs. Test-only change.
- Results after the fixes (28 Oct): `make lint test` exit 0; `make test-integration` exit 0; `make e2e` **98 passed, 5 skipped** (7 min 01 s).

## Test report 2 (28 Oct): all passed (14 of 14)

F1 fixed; T-08, T-13, T-14 and the new T-15 passed, with R-01 to R-05. Its exploring notes need no change: a long unbroken reason wraps inside the entry; a PATCH with no `revision` gives the general "The request could not be read." before the anaesthetist's message (hand-written calls only); the extension shows 204s as 503 (a known tool artefact, decisions.md §4).

**Hand check for Rahul:** the patient's steps at 390×844 (the testers' window wouldn't resize; S44b doesn't change the patient's side).

## Verification 1 (28 Oct): needs fixes, done

1. **The reason rule (finding 1):** a reason is now required for **every grade after the episode's first**, as Rahul's answer 2 says, including the correct grade after one entered in error. Go asks `HasAnaesthetistGrade` (any grade, entered in error or not) instead of the grade that counts; the web app shows the required reason box once the history has any grade (`needsReason` in `asaGrades.ts`). Test first: the integration case "every grade after the episode's first needs a reason, even after the only one was entered in error" (422, then 201 with "set to 2: <reason>") and `asaGrades.test.ts`'s `needsReason` failed, then passed. Docs: decisions.md, CLAUDE.md, AGENTS.md, the test plan (T-08, T-13).
2. **The migration's comment (finding 2):** it now says the locked paths read the grade after `LockEpisodeStatus`.
3. **A delete guard (finding 3):** migration 00035 also has a `BEFORE DELETE` trigger that refuses deleting a grade while its episode exists, so the episode's own deletion (the cascade) still takes them. Test first: "no edit and no delete" now deletes a grade (refused) and deletes an episode (its grades go). *I edited 00035 rather than adding 00036, since it isn't committed; the local database was migrated down and up again, which keeps each episode's grade that counts but not the history made while testing.*
4. **Auto-triage (finding 4):** `patient.go` reads the grade only when a rule requires it, as `statuses.go` does.

The Down step now drops the delete function `IF EXISTS` (a database migrated with the first 00035 has none). Down then Up checked locally twice; both triggers are there.

Results (28 Oct): `make lint test` exit 0; `make test-integration` exit 0; `make e2e` against a fresh `make dev` **98 passed, 5 skipped** (7 min 05 s).

## Test report 3 (28 Oct): all passed (15 of 15)

Verification 1's fixes passed in the UI and the API (T-08 step 4, T-13 step 3, the delete refusal), with the happy path and R-01 to R-05. Its exploring notes need no change. A tester can't run the episode-delete cascade by hand (another foreign key stops it first); the integration test covers it. The nurse grade's "set to" wording is older and out of scope.

**Hand check for Rahul:** the patient's steps at 390×844 (the testers' window wouldn't resize; S44b doesn't change the patient's side).

## Suggested commits

One plan step (S44b):
1. `S44b: the anaesthetist's ASA grade as a history: migration 00035, the grade routes, the PATCH refuses it` (apps/api: migration, queries, generated `internal/db`, `asa_grades.go`, `episodes.go`, `statuses.go`, `patient.go`, `tasks.go`, `router.go`, `audit.go`; tests: `asa_grades_integration_test.go`, the changed integration tests)
2. `S44b: the POA Summary's grade history, Add grade and Entered in error` (apps/web: `asaGrades.ts` + test, `AnaesthetistGrades.tsx`, `PoaSummary.tsx`, `EpisodePage.tsx`, `types.ts`, `status.ts`, `AGENTS.md`)
3. `S44b: e2e: J11b re-grades and marks an error (and checks the buttons at 500 px); the two-tab journey adds the grade; the audit journey waits for its filtered list` (e2e)
4. `S44b: docs` (CLAUDE.md, decisions.md, pending.md, plans, sessions README, this folder)
