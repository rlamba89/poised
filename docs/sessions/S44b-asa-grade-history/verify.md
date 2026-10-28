# S44b verification

## Verification 2 (28 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1's four findings are fixed, and each fix holds in the code, the tests and the live app. Every suite passes.

### Verification 1's findings, re-checked

1. **The reason rule: fixed.**
   - **Go:** `asa_grades.go:96` now asks `HasAnaesthetistGrade`, so any grade counts, even one entered in error, and an add without a reason gives 422.
   - **Web:** `needsReason` (`asaGrades.ts:47`) is `grades.length > 0`, and `AnaesthetistGrades.tsx` shows the reason box from it.
   - **The test:** the integration case at `asa_grades_integration_test.go:192` now expects 422, then 201 with a reason.
   - **Docs:** decisions.md and CLAUDE.md say "every grade after the episode's first".
   - **Live:** H3 (`5da19b29-…`) has ASA II entered in error, then ASA III "The correct grade". POSTs with no reason and with `"   "` both gave 422 "Say why the grade is changed.".
2. **The migration's comment: fixed** (`00035…sql:9–10`).
3. **The delete guard: added.** A `BEFORE DELETE` trigger checks whether the episode still exists, so the cascade from deleting an episode still goes through. An integration test covers both the refusal and the cascade, and the tester confirmed the refusal in the database. `Down` drops it with `IF EXISTS`.
4. **Auto-triage reads the grade only when a rule requires it: fixed** (`patient.go:630`, guarded like `statuses.go:279`).

### Findings

None blocking. `/code-review high` raised these, and I set them aside:
- **Two separate reads in `getEpisode`.** The counting grade comes from `GetEpisode`'s subquery and the history from `ListAnaesthetistGrades`, so they could disagree for one load if another tab adds a grade between them. That is cosmetic until a reload, and any write from that page gets a stale 409.
- **"The latest not entered in error" is written several times:** in SQL in `GetEpisode` and `CountingAnaesthetistGrade`, in the migration's Down, and in the browser's `countingGrade` (used for the Counts badge). It's a small duplication, and the shared tests hold the copies in step. Worth a SQL function if the rule ever changes.
- **The reason limit counts bytes, not characters:** this is the shared `optionalReason`/`requiredReason` helper from earlier sessions (verification 1 noted it too).
- **Quality nits:**
  - Marking an error reads the grade twice, and adding one runs two queries under the lock.
  - The ASA choices list is defined in both `PoaSummary.tsx` and `AnaesthetistGrades.tsx`.
  - `GetEpisode`'s subquery runs on every episode route.
  - None of these is a defect.

### Checks

- **Scope:**
  - Everything in the brief is done, as verification 1 found.
  - The fixes added nothing outside it.
- **Tests first:**
  - The fixed rule has its failing-first integration case, and `needsReason` has its unit test.
  - Both routes keep their five cases plus stale, cancelled/archived and invalid input.
- **A-20 grep:** nothing new (only the earlier comments in `format.ts` and `patient/patient.go`, and the infra tests).
- **Security:**
  - The audit rows hold IDs only.
  - A reason goes only into the staff General notes.
  - No patient route touches grades.
- **Docs:**
  - decisions.md, CLAUDE.md, pending.md and AGENTS.md are updated.
  - The C6 plan, the plans README and the sessions README now say Verified.
  - Repocheck passes.

### Browser spot-checks (Claude in Chrome)

All as Cara Clinician (Hospital A1) on H3, which is at Ready for admission with the rule "High risk":
- **The reason rule (finding 1):** no reason gave 422, and a blank reason gave 422.
- **Marking the only counting grade entered in error:** 409 "Add the correct grade first, or move the episode back from Ready for admission.".
- **The PATCH:** `anaesthetistAsa` gave 400 with the anaesthetist's message.
- **DELETE on a grade:** 404, because there's no such route.

Other users on H3:
- **Bea (Trust B):** GET and add gave 404.
- **Val (Hospital A2):** GET and mark gave 404.
- **Signed out:** 401.

Afterwards the revision (11), the counting grade (ASA III) and the history (2 grades) were unchanged.

**The POA Summary at about 500 px wide:**
- "Anaesthetist ASA grade: ASA III";
- ASA III with the Counts badge, its line "Given by Cara Clinician … : The correct grade", and the Entered in error button not clipped;
- ASA II struck through, with "Entered in error: marked by … : Wrong patient";
- the Add box with the required "Why is the grade changed?".

### Test results (28 Oct)

- **`make lint test`:** exit 0. Vitest 299 + 13 + 444 passed, CDK 124 passed, and every Go package ok.
- **`make test-integration`:** exit 0 (httpapi 28.0 s).
- **`make e2e`** against the conductor's `make dev`: exit 0, **98 passed, 5 skipped** (the AWS-only journeys) in 7 min 27 s.

### Commits

These are verification 1's suggested commits, made locally in automatic mode and not pushed:
1. `S44b: the anaesthetist's ASA grade as a history: migration 00035, the grade routes, the PATCH refuses it` (apps/api, with its integration tests).
2. `S44b: the POA Summary's grade history, Add grade and Entered in error` (apps/web, with `asaGrades.test.ts`).
3. `S44b: e2e: J11b re-grades and marks an error; the two-tab journey adds the grade; the audit journey waits for its filtered list`.
4. `S44b: docs` (not `docs/market/`, and not `run-state.md`).

### Hand checks for Rahul

- **The patient's steps at 390×844:** the testers' window wouldn't resize. S44b doesn't change the patient's side.
- **Pending question:** should the app refuse a re-grade equal to the grade that counts? This is in pending.md, and the recommendation is to keep accepting it as a confirmation.

## Verification 1 (28 Oct 2026, automatic run)

**Verdict: Needs fixes.** One finding: the reason rule doesn't follow Rahul's answer. Everything else is sound. Tenancy, roles, the lock and revision, audit, the PATCH refusal, the trigger, S43's requirement on every path, and the UI all check out, and every test suite passes.

### Findings (worst first)

1. **A grade added after an entered-in-error one needs no reason, but Rahul's answer says it must give one.**
   - **What Rahul said:** [answers.md](answers.md) item 2: "**required for a re-grade** (every grade after the first); the first grade needs none".
   - **What was built:** a reason is required only *while a grade counts*. Where it's decided:
     - `apps/api/internal/httpapi/asa_grades.go:92` (`was != nil && reason == ""`);
     - `apps/web/src/features/episodes/asaGrades.ts:49` (`counts && !reason.trim()`);
     - `AnaesthetistGrades.tsx:31,68` (the reason box shows only while a grade counts);
     - the test that pins it: `asa_grades_integration_test.go:192` ("once the only grade is entered in error, the next needs no reason").
   - **The builder's reasoning:** "a correction after a mistake isn't a re-grade". The brief's route line supports it, and the builder noted that the brief's migration line ("required unless it's the episode's first") says the other thing.
   - **Not faithful:** Rahul's parenthesis defines a re-grade as *every grade after the first*, and the brief's migration line agrees. So "first" means the episode's first grade, not the first one that counts now.
   - **Seen live:** H3 (`0162fa12-…`) has ASA II entered in error ("Wrong patient"), then ASA III by the same clinician with `reason: null`. The history shows the second grade with no why.
   - **Why it matters:** the history's purpose is that each grade after the first explains itself. The entered-in-error reason explains why II stopped counting, not why III is the right grade.
   - **Fix (test-first):**
     - Require a reason whenever the episode already has **any** grade, entered in error or not (Go: count every grade, not the counting one; 422 as now).
     - Show the reason box once any grade exists (`grades.length > 0`), and make `addGradeRequest` take that.
     - Flip the integration case at :192 to expect 422 without a reason and 201 with one. Update `asaGrades.test.ts`, J11b/T-08's expectation, decisions.md's 27 Oct S44b bullet and the CLAUDE.md line ("a reason required for every grade after the episode's first").
   - **The General notes after an error:** keep "set to N" if nothing counts, with the reason after the colon, as `withReason` already does.
2. **Low: `00035_anaesthetist_asa_grades.sql:9` is stale.** It says GetEpisode *and LockEpisodeStatus* work the grade out from the table, but LockEpisodeStatus no longer reads it (`countingGrade` does, after the lock). Fix the comment while you're there.
3. **Low, optional: the database refuses an UPDATE of a grade but not a DELETE.**
   - The trigger (`00035…sql:41`) is `BEFORE UPDATE` only, yet the build notes say the trigger makes "never changed" hold "whatever a later query does", and the brief says "no edit and no delete … anywhere".
   - The API has no delete, and no query deletes, so this isn't a live gap.
   - If you add a `BEFORE DELETE` guard, it must still let `ON DELETE CASCADE` from `episodes` through. If that's awkward, record "API-only for delete" in decisions.md instead.
4. **Trivial: `patient.go:630`** reads `countingGrade` on every auto-triage to Ready for admission, even when no rule requires the grade. `statuses.go` guards with `AnaesthetistAsaRequiredBy != nil` first; do the same for consistency.

**Not findings** (I weighed these from `/code-review high` and set them aside):
- **The reason limit counts bytes, not characters:** this is the shared `optionalReason` from earlier sessions.
- **`given_at` uses the transaction's start time:** the revision check under the lock serialises adds, so order can't invert.
- **"Entered in error" is offered on the grade the server will refuse:** a documented choice, and the refusal says why.
- **"No ASA grades have been added." next to "No grade yet":** tester-reviewed in T-02.

### Checks

- **Scope:**
  - The brief is covered: migration (column dropped, carried over, Down tested by the builder), both routes, no edit or delete route, PATCH 400 (even for `null`), S43's requirement reading the grade that counts (status moves, blocked moves and Undo via `GetEpisode`, auto-triage), the 409 at Ready for admission/Admitted/Discharged, the POA Summary history, J11b, and the docs.
  - Nothing unasked-for of size. The `test_audit.py` wait is a test-only flake fix, explained in the build notes.
- **Tests first:** both routes have the five cases plus stale, cancelled/archived and invalid input (`asa_grades_integration_test.go`); `TestTheRequiredGradeCantGoMissing` covers the requirement paths; `asaGrades.test.ts` covers the web logic.
- **Tenancy:** both routes go through `hosp(...)` and `loadEpisode`, and the grade lookup is scoped by `episode_id`. Live, as Bea (Trust B), Val (A2) and Max (A2 clinician, Trust B admin) on an A1 episode: GET, add and mark all gave 404. Another episode's grade id on H3's path gave 404 "Grade not found."; no sign-in gave 401.
- **A-20 grep:** nothing new (only earlier comments in `format.ts` and `patient.go`).
- **Security:** the audit rows are IDs only (`asa.add`, `asa.error`). Reasons go only into staff General notes, never into logs or messages. No patient route touches grades.
- **Docs:** decisions.md, pending.md, CLAUDE.md, AGENTS.md, the C6 plan's Status and both READMEs are updated. Repocheck passes (in `make test`).

### Browser spot-checks (Claude in Chrome, Cara Clinician unless said)

- **Tenancy probes:** the 404s and the 401 above.
- **H3 at Ready for admission, rule "High risk", one counting grade:** marking it entered in error gave 409 "Add the correct grade first, or move the episode back from Ready for admission.". The grade still counts.
- **PATCH `{anaesthetistAsa: null}`:** 400 with the anaesthetist message. A grade of 7 gave 400; a stale revision gave 409.
- **H3's POA Summary:** "Anaesthetist ASA grade: ASA III", ASA III badged Counts, ASA II struck through with its entered-in-error line, and the Add box with the required reason box. Finding 1 is visible here: ASA III has no reason.

### Test results (28 Oct)

- **`make lint test`:** exit 0. Vitest 298 + 13 + 444 passed; CDK 124 passed; Go all ok.
- **`make test-integration`:** exit 0 (httpapi 27.6 s).
- **`make e2e`** against the conductor's `make dev`: exit 0, **98 passed, 5 skipped** (the AWS-only journeys) in 7 min 20 s.

### Suggested commits (after the fix and a retest)

1. `S44b: the anaesthetist's ASA grade as a history: migration 00035, the grade routes, the PATCH refuses it` (apps/api, with its integration tests)
2. `S44b: the POA Summary's grade history, Add grade and Entered in error` (apps/web, with `asaGrades.test.ts`)
3. `S44b: e2e: J11b re-grades and marks an error; the two-tab journey adds the grade; the audit journey waits for its filtered list` (e2e)
4. `S44b: docs` (CLAUDE.md, decisions.md, pending.md, plans, sessions README, this folder; not `docs/market/`)

### Hand checks for Rahul

- The patient's steps at 390×844 (the testers' window wouldn't resize; S44b doesn't change the patient's side).
