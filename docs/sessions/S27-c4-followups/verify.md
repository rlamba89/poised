# S27 verification

## Verification 3 (24 Oct 2026, automatic run; a round Rahul allowed past the usual two)

**Verdict: Verified.**

Verification 2's Must fix is fixed and holds on the conductor's shared database. Its wording finding is fixed too, with a unit test first. The session as a whole matches the brief and answers 1–5. `make lint`, `make test` and `make test-integration` pass. Two whole e2e runs against the conductor's `make dev` each had one failure, a different one each time. Both are older timing flakes in journeys S27 doesn't touch, and each passes when run again (details below). No finding below blocks; the new ones are in pending.md ("From S27's verification 3").

### Verification 2's fixes, checked
1. **`test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`** (`e2e/journeys/test_audit.py:131-157`):
   - it now checks only its own data: the surname filter's "Nothing matches these filters.", then Trust B's audit CSV, which must not name this episode's id or its record's id;
   - it also checks that Trust A's CSV does name the episode, so the check isn't vacuous;
   - it passed in both whole runs on the conductor's database, which holds Cara's four `prefill.read` rows in Trust B's log, the case verification 2 caught.
2. **"…including ones at Hospital B1 and Hospital C1"** when every answer came from two other hospitals (`apps/web/src/features/patient/prefill.ts`, `cardText`):
   - it now says "…from your earlier questionnaires at Hospital B1 and Hospital C1, the latest on …";
   - "including" is kept only for a mix with this trust's own answers;
   - a Vitest case covers it.

   I traced the branches: one other hospital on one day gives "…your questionnaire at Hospital B1 on …"; a mix gives "…including one at …"; this trust only names no hospital.

### Findings (none blocking; worst first)
`/code-review high` raised nine points again. Six are already in pending.md from verifications 1 and 2:
- every open re-reads other trusts;
- Start fresh reloads the whole HQ;
- the POA Summary says nothing for a set started fresh;
- the way back's cmd-click;
- `highlightQuestion` and `focusQuestion` are near-copies;
- the A-20 exception for the "NHS number" and "UK postcode" names, recorded in decisions.md on 24 Oct.

Three are new. Each is minor and never silent, so they're in pending.md, not fixes:
1. **If the reload after Start fresh fails,** the card stays on "Yes, start fresh" with the old answers (`PatientHQ.tsx:226-236`: `choose` awaits `load()`, which catches its own error, so `choose` returns true).
   - The page shows the error.
   - Any button then gets Go's 409 "…Please reload the page.", so nothing is saved wrongly.
2. **`restoreSpot` can't be cancelled** (`jump.ts`; the `PoaSummary.tsx` effect has no clean-up). It polls for up to about a second, so leaving the POA Summary within that second could scroll the next page once. Cosmetic.
3. **The read-only View of an archived episode still says "A member of the team will review them."** (`PatientHQ.tsx:279`). The hospital has closed the episode. Rahul to judge.

### Spot checks
- **POA Summary → Validate → Back** (Cara's hospital, "Sam S27 T01r4 1414", Patient answers tab):
  - the Your health set lists four answers pre-filled from Hospital B1 (another trust), with "matched by same wording" or "…by link", as links;
  - "Do you snore loudly?" opened `…/sets/…312?q=snoring`, with the question in view and its marker "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust)…";
  - the browser's Back returned to `?tab=patient`, with the link at the same height (490.5 px before and after).
- **No Trust A ids in Trust B's audit** (signed in as Max Multi, Trust B's admin):
  - Trust B's audit CSV (200) holds 98 distinct ids, and none is a Trust A episode or record;
  - it names Cara four times, which is agreed;
  - in the database, none of the 11,030 `audit_log` rows names a patient or episode outside its own row's trust.
- **Settings** (Trust B, as Max): there's no `crossTrustPrefill`, and the Cancelled default reads "…has cancelled your pre-operative assessment…".
- I left Chrome signed in as Ada Admin, as I found it.

### Checked and fine
- **The A-20 grep:** only doc comments (`lib/format.ts`, `patient.go`).
- **The removed names:** `crossTrustPrefill`, `prefillConsent` and `patient_prefill_consents` remain only in tests, migrations and one comment about the removal (`settings.go:11`).
- **The database:** the conductor's is at migration 00027.
- **Docs:**
  - decisions.md records verification 2's gotcha (run the whole e2e suite on the shared database);
  - pending.md had verification 2's section twice, the first copy with the item that's now fixed, so I removed that copy;
  - `repocheck` passes in `make test`.

### Test results
- `make lint`: passed.
- `make test`:
  - Go: all ok;
  - Vitest: 227 passed (web) and 152 passed (clinical);
  - CDK: 124 passed.
- `make test-integration`: every package ok (`internal/httpapi` 15.8 s, `internal/apitest`).
- **`make e2e`, the whole suite, against the conductor's `make dev`** (its `sj` database at 00027, with the testers' cross-trust data):
  - **Run 1:** 1 failed, 64 passed, 5 skipped. The failure was `test_statuses.py::test_a_stale_tab_is_told_and_keeps_its_input` (S20's): the second tab's save went through with no conflict. It passed 3 of 3 times on its own (4 of 4 tests each time). S22's and S24's builders saw the same flake. It's now in pending.md.
  - **Run 2:** 1 failed, 64 passed, 5 skipped. The failure was `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all` (S11's hover tooltip). That flake is already in pending.md from S25. Run on its own seven times, the file passed five times and had one failure twice: once this tooltip test, once a run whose output I didn't keep.
  - Every journey passed in at least one of the two runs, the S27 ones (J16b, J9, J5 and the changed audit test) in both.

### Hand checks for Rahul
None.

### Commits
1. `S27: pre-fill from other trusts without asking …`, with every code and test change, the e2e journeys and their README rows.
2. `docs: S27 verified …`, with the session folder, decisions, pending, requirements, CLAUDE.md, AGENTS.md and the session status.

`docs/market/` (Rahul's) and `docs/sessions/run-state.md` are left out.

## Verification 2 (24 Oct 2026, automatic run)

**Verdict: Needs fixes.**

Verification 1's four findings are fixed, test first, and they hold in the browser. The session as a whole matches the brief and answers 1–5. `make lint`, `make test` and `make test-integration` pass.

But the whole e2e suite, run as verification 1 asked, has **one failure caused by S27**. It's the one Must fix below. The fix is in the test, not the product. The other findings don't block, and they're recorded in pending.md.

### Must fix
1. **`test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a` fails on any database where a Trust A clinician has filled in for a patient with a Trust B history.** (`e2e/journeys/test_audit.py:140`)
   - **What happens:** the test signs in as Max Multi (Trust B's admin), opens the **Who** filter, and expects no "Cara Clinician (cara.clinician@hospital-a.example)" option. Since S27, Cara's on-behalf reads of Trust B's answers write `prefill.read` rows **in Trust B's log, as Cara**. That's as agreed (decisions.md §2, 24 Oct: "on behalf its actor is the staff member"), and Trust B's Who filter therefore lists her.
   - **On the conductor's database:** Trust B's log holds four `prefill.read` rows by Cara (11:22–12:30, from the manual tests). The result was **1 failed, 64 passed, 5 skipped**.
   - The builder's run passed only because it used a fresh database (`sj_s27`).
   - **Why it matters:**
     - `make e2e` must pass on the shared dev database.
     - The pipeline runs e2e on the **persistent** QA database after every push. Once anyone fills in for a cross-trust patient there as Cara, every later pipeline run fails.
   - **Fix (test-first is the test itself):** make the test independent of other tests' and testers' data. **Recommended:** keep the patient-filter check ("Nothing matches these filters." for the new surname). Replace the Who check with a check that Trust B's log has no row for this test's new episode or record, for example the episode's id in the CSV or the Episode filter. *Or:* create the episode as a staff member who can never read Trust B. That's not robust, because any Trust A clinician can fill in on behalf.
   - Then re-run the whole suite **on the conductor's database**, not only a fresh one.
   - *Not a product change:* Trust B's admin seeing which Trust A staff member read its patients' answers, with their email, is the agreed audit behaviour. Noted for Rahul under "Notes" below.

### Verification 1's fixes, checked
1. **A stale card after Start fresh** (`prefill.go`, the `RecordPrefillChoice` fallback): once the recorded choice is `start_fresh`, the other two buttons get a 409 with `startedFreshMsg`.
   - `TestStartFresh` covers both buttons, for the patient and for staff on the patient's behalf.
   - **In Chrome:** I opened Cara's Fill in for "Sam S27 V2 1324" (`932caac0-…`). About you showed the card "…from Sam's questionnaire at Hospital B1 on 24 Oct 2026…" with Start fresh. I then posted `start_fresh` from a second "tab" (204). On the card still showing, **Everything is still correct** and **Check each answer** each showed the red "…cleared with “Start fresh” somewhere else… Please reload the page.", and the card stayed. In the database the row was `{}`, at revision 2, `start_fresh`, by Cara.
2. **"…at Hospital B1" with no "including"** when every answer is one other hospital's (`cardText`, `allThere`): a Vitest case covers it.
3. **Start fresh decided from the shown answers** (`offersStartFresh(prefill, shown)`): a Vitest case covers it.
4. **The 400 names all three buttons:** `TestStartFresh` checks it.

### Other findings (none blocking; worst first)
1. **The card still says "including ones at Hospital B1 and Hospital C1" when every answer came from those two other hospitals.** (`apps/web/src/features/patient/prefill.ts`, `cardText`)
   - This is the same wording problem as verification 1's finding 2, for two or more other hospitals. `allThere` handles only one.
   - It's rare: a person with sent HQs at two other trusts and none at this one. Fix: say "…at Hospital B1 and Hospital C1…" when every entry is another trust's. Recorded in pending.md.
2. **Start fresh reloads the whole HQ** (`PatientHQ.tsx` `choose` → `load()`), which replaces every set's saver.
   - A save of another set that is still in flight when the reload returns (left within about 0.6 s of pressing Start fresh) would leave that set's new saver one revision behind. Its next save would then get the "changed on another device" 409, which asks for a reload. Nothing is lost silently.
   - The same `load()` is used by the 409 Reload, and the confirm step makes this timing unlikely.
3. **The POA Summary says nothing for a set started fresh.** (`PoaSummary.tsx` `SetSection`) The choice line shows only with pre-filled marks, and a set started fresh has none.
   - Validate says it ("…chose “Start fresh” … so the pre-filled answers were cleared…").
   - This fits PRE-04 (the POA Summary lists the pre-filled answers shown, and there are none). Rahul may want the line on the POA Summary too.
4. **Go accepts Start fresh when the only other-trust entry is on a hidden question** (`StartFreshPrefill`'s `jsonb_path_exists`), while the card doesn't offer it there. Only a stale tab or a hand-made request gets there. Verification 1 kept Go's check as it is, deliberately.
5. **Small things on the POA Summary's way back** (`jump.ts`, `PoaSummary.tsx`):
   - a cmd/ctrl-click on an answer still saves the spot, which a later unrelated Back to the POA Summary would use;
   - which accordion sets were closed isn't kept, so Back after closing some lands near, not on, the answer.

   Cosmetic.
6. **Already in pending.md from verification 1:**
   - every open re-reads other trusts;
   - the badge on a cancelled episode;
   - "Custom pattern" against Go's name for it.

   `/code-review high` raised them again. The "NHS number"/"UK postcode" names in `keys.go` are the recorded A-20 exception (decisions.md §2, 24 Oct).

### Spot checks in Chrome
- **Fix 1, on behalf:** passed, as above.
- **POA Summary → Validate → Back** (Cara, "Sam S27 T01x 1325", Patient answers tab):
  - clicking "Have you ever been sick…" opened `…/sets/…312?q=ponv_history`;
  - the question was centred and highlighted yellow, with its marker "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust)…";
  - the browser's Back returned to `?tab=patient` with the link at the same height (716 px before and after).
- **Archived after sent** ("Sam S27 T11 1230"):
  - signed in by the email sign-in link and date of birth, the home shows **Sent** with **View**;
  - View shows "Thank you, Sam";
  - the form's GET is a 200 with `submitted: true`, and a save and a send are each a 403 `archived`.
- **Settings** (Ada): the trust's settings have no `crossTrustPrefill`, and the Cancelled default reads "…has cancelled your pre-operative assessment…".
- *The window wouldn't resize to 390 × 844 in this browser, so these patient screens were seen at desktop width. Testers and the e2e journeys (J9, J16b) cover phone size.*

### Checked and fine
- **Scope:**
  - all Build items and answers 1–5 are present (see verification 1);
  - the fixes added nothing else.
- **Tenant isolation:**
  - other trusts are found only through the account (`ListPrefillTrusts`, the person's own records, active trusts);
  - each is read with its own `org_id`;
  - `StartFreshPrefill` is keyed by a form already resolved within the session's or hospital's scope.
- **The archived GET:** only the patient's `GET /api/p/episodes/{eid}` is let through, and only for a form sent before archiving. `prefillForm` returns at once for a sent form, so nothing is written. Staff's on-behalf access stays a 409.
- **A-20 grep:** only doc comments (`lib/format.ts`, `patient.go`), as before.
- **Removed names:** `crossTrustPrefill`, `prefillConsent` and `patient_prefill_consents` remain only in tests, migrations and comments about their removal.
- **Docs:**
  - decisions, pending, requirements, CLAUDE.md and AGENTS.md are updated;
  - `repocheck` passes in `make test`.

### Test results
- `make lint`: passed.
- `make test`:
  - Go: all ok;
  - Vitest: 227 passed (web) and 152 passed (clinical);
  - CDK: 124 passed.
- `make test-integration`: every package ok. Verbose: `TestStartFresh` (8 subtests, the stale-card case included), `TestPrefillCrossTrust` (8), `TestArchivedAfterSentKeepsItsView` and `TestStatusMessages` pass.
- `make e2e`, the whole suite, against the conductor's `make dev`: **1 failed, 64 passed, 5 skipped**. The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a` (Must fix 1). J16b, J9 and J5 passed.

### Notes for Rahul (no change needed)
- **Trust B's audit log names Trust A's staff member, with their email, when they fill in for a patient with a Trust B history.** It follows from answer 3 and the recorded `prefill.read` rule (the actor is whoever caused the read). It's right for accountability: Trust B, as controller, sees who read its patients' answers. Worth a line in the IG/DPO review (C10), with the cross-trust copying already listed there.

### Hand checks for Rahul
None.

### For the builder
Fix Must fix 1 in `e2e/journeys/test_audit.py` (and, if convenient, other finding 1's wording, with a Vitest case first), then run the **whole** e2e suite against the conductor's `make dev` (database at 00027, with the testers' data). No manual retest is needed for a test-only change, unless the conductor wants one.

### Suggested commits (once verified)
1. `S27: pre-fill from other trusts without asking …`, as in build-notes.md, with every code and test change.
   The e2e journeys and their README rows go in this commit too.
2. `docs: S27 verified …`, with the session folder, decisions, pending, requirements, CLAUDE.md, AGENTS.md and the session status.

## Verification 1 (24 Oct 2026, automatic run)

**Verdict: Needs fixes.**

The session is complete as briefed and answered: cross-trust pre-fill without asking (for the patient and on their behalf), migration 00027, the card naming the hospital with Start fresh, the archived read-only View, the badge, neutral Cancelled text and subject, format names in the refusal, and clickable POA Summary answers. Every test passes. But Start fresh has one real hole that can send an empty Question Set without anyone being told, plus two small mistakes in the card. The fixes are small, and each needs a failing test first.

### Findings, worst first

1. **Must fix: a stale tab's other button is accepted after Start fresh, and can send an empty set without warning.** (`apps/api/internal/httpapi/prefill.go:217-221`)
   - **What happens:** after Start fresh in one place (the patient's phone, or another tab), a card still open elsewhere can press **Everything is still correct** or **Check each answer**. `RecordPrefillChoice` matches no row, and the fallback returns **204** ("the first button stays"), because *a* choice is recorded. It doesn't check that the choice is `start_fresh`, which emptied the set.
   - **Reproduced in Chrome:**
     - I opened Cara's on-behalf form for a new Trust A episode of "Sam S27 T01r 1300" (`9cf3de9e-…`). The Your health card showed "…including one at Hospital B1…" with Start fresh.
     - I posted `start_fresh` for that set from the page as a second tab (204).
     - I then pressed **Everything is still correct** on the card still on screen. It was accepted: the survey showed the copied answers, with their "From Sam's answers…" markers. The server held `data: {}` with choice `start_fresh`.
   - **When nobody is told:** in my run a question was left to answer, so the next save hit the stale-revision 409 ("Your answers weren't saved"). But in a set where **every** visible question was pre-filled, Everything is still correct saves nothing. The set counts as done from the tab's old copy, and **Send** freezes the server's **empty** set. The hospital gets an empty Question Set that the sender saw as fully answered.
   - **Why it matters:**
     - The build notes promise "a tab still showing the answers … can't save them back", and this path gets round it.
     - decisions.md §1 says "Nothing may break silently".
   - **Fix:**
     - In the fallback, when the recorded choice is `start_fresh` and the request isn't, return 409 with the "changed since you opened it … Please reload the page." message. The card already shows Go's refusal.
     - Test first: a `TestStartFresh` case for each of the two buttons, and for staff on the patient's behalf.

2. **Should fix: the card says "including one at Hospital B1" when every answer came from Hospital B1.** (`apps/web/src/features/patient/prefill.ts:83-90`, `cardText`)
   - When all the answers are another trust's, from one hospital, but from two dates, `oneSource` is false. The card reads "…from your earlier questionnaires, including one at Hospital B1, the latest on …". "Including" tells the patient some answers came from somewhere else, and none did.
   - **Fix:** in that case say "…from your earlier questionnaires at Hospital B1, the latest on …". Add a Vitest case next to the existing "several hospitals" cases.

3. **Should fix: Start fresh can show on a card that names no other hospital.** (`apps/web/src/features/patient/prefill.ts:40`, `offersStartFresh`)
   - It looks at **every** entry in the set's map, but the card describes only the **shown** answers (`shownPrefilled`). Suppose another trust's answer is on a question a condition hides, and the shown answers are all this trust's. Then the card reads "…from your questionnaire on …" (no hospital), yet it offers Start fresh, which also clears this trust's own answers.
   - That goes against answers 1–2: Start fresh belongs to the card that names another hospital.
   - **Fix:** decide from the shown entries, the same list `cardText` gets. Go's check can stay as it is.
   - **Test:** a Vitest case with an `otherTrust` entry on a hidden question.

4. **Trivial: the 400 for an unknown choice lists only two buttons.** (`prefill.go:194`) It says "Choose “Check each answer” or “Everything is still correct”." Add Start fresh, or make it neutral.

### Notes (no change needed in S27; recorded for Rahul)
- **Every open reads every other trust again.**
  - An open form with a set that nothing matched (it never gets a row) re-runs `ListPrefillTrusts` and one `ListPrefillSources` per other trust **on every open** (`prefill.go:66`). Each run writes another `prefill.read` row in that trust's audit log.
  - S26 already did this, but only after the patient's Yes. Since S27 it applies to everyone with another trust's history, and to staff on their behalf.
  - The rows are accurate (the data really is read), but Trust B's log will be noisy.
  - **Possible later change:** read other trusts only on the first open of the form.
  - Worth a pending.md line; it's not a correctness bug.
- **The Not responding badge on a cancelled episode.** `notResponding` (`status.ts:35`) hides the badge for archived episodes only, as briefed. A cancelled episode flagged earlier still shows it, although the worklist (`in_worklist`) lists only `hq_not_complete`. Outside the brief; a candidate for pending.md.
- **Go and the editor can name the same pattern differently.** `formatOf` (`keys.go:256`) names a pattern from `textFormatNames` even when the editor marked the question `textFormat: "custom"` (a custom pattern that started as a copy of phone). Go would say "phone number" where the editor says "Custom pattern". Harmless: the lock compares the same names on both sides.
- **The "NHS number" and "UK postcode" names** are the recorded A-20 exception (decisions.md, 24 Oct); the comment says so. The A-20 grep shows nothing new outside config, seed and tests (only doc comments in `lib/format.ts` and `patient.go`).
- **F1 (on-behalf pre-fill for a record not yet joined to the patient's account)** is recorded in pending.md for Rahul. The builder kept the S15 behaviour, which fits answer 3 as written ("too", that is, the same answers the patient gets). It isn't a finding here.

### Checked and fine
- **Scope:**
  - every Build item in the brief and answers 1–5 is built;
  - the brief's "email address" format was already named in words before S27;
  - nothing big is there that the brief didn't ask for.
- **Migration 00027:**
  - drops the consent table and the stored setting, and widens the check;
  - Down puts the table back and maps `start_fresh` to `check_each`;
  - `TestPrefillWithoutAskingMigration` covers it.
- **Tenant isolation:**
  - other trusts are found only through the person's account (`ListPrefillTrusts`, the person's own records, active trusts only);
  - their answers are read one trust per query by that trust's `org_id`;
  - no trust B ids reach trust A's rows (S26's tests, rewritten, still assert it);
  - `StartFreshPrefill` is keyed by the form, which `loadPatientForm`/`loadStaffForm` resolve with the session's or hospital's scope;
  - `TestStartFresh` has the five cases.
- **Archived View:**
  - only a patient **GET** of a form **sent before** archiving is let through (`patient.go:331`). `prefillForm` returns at once for a sent form, so the GET writes nothing.
  - Every write is still a 403, and staff on-behalf access is still a 409.
- **The consent route is gone:** `POST /api/p/…/prefill-consent` returns 404 in the browser, and the form's JSON has no `prefillConsent`.
- **In the browser:** Cara's on-behalf card for another trust's answers names Hospital B1 and offers Start fresh. After Start fresh the set is `{}` at revision 2 with choice `start_fresh`. A stale save then gets the 409 "Your answers weren't saved" screen.
- **Docs:**
  - the decisions, pending, requirements, CLAUDE.md and AGENTS.md changes are present;
  - the sessions table says Tested;
  - `repocheck` passes inside `make test`.

### Test results
- `make lint`: passed.
- `make test`:
  - Go: all ok;
  - Vitest: 226 passed (web) and 152 passed (clinical);
  - CDK: 124 passed.
- `make test-integration`: every package ok, including `internal/httpapi` and `internal/apitest`.
- E2E against the conductor's `make dev`: J16b, J9 and J5 (the three journeys S27 changed) gave **5 passed**. The builder's full run was 65 passed, 5 skipped. The whole suite should be re-run at the next verification.

### For the builder
Fix findings 1–3 test-first, and finding 4 if convenient. Then ask for a retest: findings 1–3, plus T-01/T-02 and T-07 as regression.

### Suggested commits (once verified)
1. `S27: pre-fill from other trusts without asking …` with its tests, as in build-notes.md's final suggestion, plus the fixes above.
2. `docs: S27 verified — build notes, test plan and reports, verification, decisions, pending, requirements, CLAUDE.md, AGENTS.md, session status`.
