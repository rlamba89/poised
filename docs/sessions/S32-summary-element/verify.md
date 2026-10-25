# S32 verification

## Verification 1 (25 Oct 2026, automatic run)

**Verdict: Verified.**

Everything in the brief is built, each part with tests written first, and nothing big outside it. Rahul's answer (a), each note once under the nearest summary, is what `summaryOwners` does. There's no new route, no migration and no new query.

### What was checked
- **`packages/clinical/src/summary.ts`:**
  - `walk` gives every element the nearest summary around it: its own Group's or Section's, else an enclosing one's, else the page's.
  - `upgradeSummaries` reads the old switch as `cs_<page>`, so comments keep `cs_comments_<page>`. It never changes its input.
  - `summaryProblems` blocks two summaries with the same scope, and one inside a repeating group.
  - `withClinicalSummaries` expands each summary where it was placed. `refreshClinicalSummaries` is given the raw chapter content by both callers (`ValidateSet.tsx:131`, `PreviewPage.tsx:125`), and it upgrades that content itself.
- **`poa.ts`:** the page's own block comes first, for notes no summary covers, then one block per summary. Empty blocks are dropped, and none of a page's notes are lost.
- **Patients:**
  - Go's `chapter.ForPatient` (`patient.go:44`) strips a summary by its type as well as its flag. So does TypeScript's `stripClinicianOnly` (`viewer.ts:23`).
  - The integration test `TestPatientHQ` asserts that no `clinicalsummary` or `cs_secret` reaches the patient.
- **hqimport:** the incumbent's summary becomes an element where it sat, on the page or in its Group. Unit-tested.
- **Tenancy:** no new query. Spot-checked below.
- **A-20 grep:** none of S32's added lines has a country literal.
- **repocheck:** passes (in `make test`).
- **Security:** no secrets, and no patient data in logs or messages.
- **Docs:**
  - SGN-04 and QT-09 are changed in requirements.md.
  - decisions.md has the builder's choices, and its §5 is updated.
  - pending.md has the tester's two cosmetic notes.
  - The plan's Status, plans/README, sessions/README, AGENTS.md and the e2e README are updated.

### Findings (none blocking, worst first)
1. **A summary with a condition still takes its notes when the condition hides it** (`summary.ts:75`; `panelFor` copies `visibleIf` at `summary.ts:150`).
   - Example: a Group summary shown only "if age > 70", with a page summary too. For a younger patient, Validate shows neither the Group's notes nor its comments box. The POA Summary still prints the notes under that summary's title.
   - Nothing is lost from the POA Summary, but the clinician doesn't see those notes while validating.
   - **For Rahul:** either the notes fall back to the next visible summary, or the editor stops offering Logic on a summary (simpler).
2. **The incumbent import can bring in two summaries in one scope** (`hqimport/convert.go:195`). Before, any number on a page collapsed into one switch. An export with two at the same level would now import, but fail to publish until one is deleted.
   - The incumbent's own SGN-04 rule (one per page) makes this unlikely. If it happens, the publish problem names it, so it's safe.
3. **A summary can still end up inside a repeating group by an indirect route.** Examples: moving a Group that holds one into a repeating group, a Group nested inside one, or making its Group repeatable.
   - The Add bar (`AddContentBar.tsx:47`) and "Move into" (`doc.ts:403`) check only the direct case.
   - Publish blocks it (`summaryProblems`), so it only affects a draft's preview, where its comments box sits inside the rows.
4. **Two capitalisations on one POA Summary** (`poa.ts:73` vs `summary.ts:86`):
   - The page's own block is "Heart Summary".
   - An untitled summary (and an old switch, now an element) is "Heart summary".
   - Cosmetic.
5. **Cost:** `refreshClinicalSummaries` walks the chapter three times on each answer change. For old-switch content, each walk also copies it (`upgradeSummaries`, `structuredClone`). `SummarySettings` (`SettingsPanel.tsx:201`) clones the chapter on each render to work out the default title. This is fine at today's sizes. One walk returning both the summaries and the owners, plus a `defaultTitle` on `SummaryInfo`, would tidy it up.
6. **Small ones:**
   - Setting a summary's title back to the default saves `title: undefined`, which would drop that title's translations (`SettingsPanel.tsx:211`).
   - `"clinicalsummary"` is written out again in `keys.go:84` and `convert.go:197` rather than using `chapter.SummaryType`.
   - The delete confirmation's wording is already in pending.md.
7. **`TestStatusMessages/AWS_reports_a_status_message_undelivered` (the builder's one-off failure) is a flake, not S32's.**
   - S32 changed no messaging code, and the test passed in this verification's full integration run.
   - Cause: `lastNote` takes the last General notes row. The bounce note is recorded with the test's `time.Now()`, while the "sent by email" note uses the database's clock. A small clock difference between Docker and the Mac can put them in the wrong order.
   - Fix: pass a time after the send, or have `lastNote` find the note by its text.

### Spot checks (Claude in Chrome, trk profile, against `make dev`)
- **The patient never gets a summary, and other users can't reach the questionnaire:**
  - Cara's on-behalf patient form (`GET …/episodes/<id>/patient-form`, the patient's JSON) has no `clinicalsummary`, `cs_`, `clinicianOnly` or `clinicalOutputs`. The staff HQ for the same episode has all three summaries.
  - Val (Hospital A2) gets a 404 on that A1 episode. Bea (Trust B) gets a 404 on the episode and on the questionnaire.
- **POA Summary (S32 test 1650, POA complete):** the blocks Breathing summary (Inhaler and its comment), Blood summary (Anaemia and its comment), Heart overview (Angina and its comment) and Lungs Summary (Cough). No note shows twice, and the "Changed by the clinician" line is kept.
- **Old switch, published version:** the POA Summary shows "Old heart summary" with "Old angina note" and "Clinical comments: Old comment kept", the comment saved under the old key.
- **Validate:** the Breathing and Blood boxes sit inside their Group and Section, in the left column ("No disclosures in this group/section yet"). Heart overview is in the clinician column ("on this page"). No console errors.

### Test results
- `make lint`: clean.
- `make test`:
  - Go: all ok.
  - Vitest: web 244 passed (23 files), clinical 182 passed (19 files).
  - CDK: 124 passed.
- `make test-integration`: all ok, `TestStatusMessages` included.
- `make e2e`: 72 passed, 1 failed, 5 skipped (the Cognito journeys, QA only).
  - The failure was `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`. Its hover tooltip ("N patients") didn't appear, and S32 doesn't touch the Audit page.
  - It passed when run alone, and again with all of `test_audit.py` and J10 (7 passed).
  - It's a hover flake, possibly from my spot checks running as Cara alongside it. It's noted in pending.md.

### Hand checks for Rahul
- None. The tester ran all 28 cases in the browser.

### Suggested commits
1. `S32: the Clinical Summary element in packages/clinical`: `packages/clinical/src/{summary,doc,publish,poa,keys,viewer}.ts` and their tests.
2. `S32: the editor, clinician view and POA Summary use summary elements`: `apps/web/src/features/editor/{AddContentBar,Canvas,ElementCard,SettingsPanel}.tsx`, `useChapters.ts`, `features/preview/{clinicianView.ts,PreviewPage.tsx}`, `features/episodes/{ValidateSet,PoaSummary}.tsx`, `globals.css`, `apps/web/AGENTS.md`.
3. `S32: Go imports and strips summary elements`: `internal/hqimport`, `internal/chapter`, `internal/keys`, `internal/apitest/testdata/workflow.json`, `internal/httpapi/patient_integration_test.go`.
4. `S32: J10 starts (Clinical Summaries)`: `e2e/journeys/test_j10_authoring.py`, `e2e/data/summaries-hq.json`, `e2e/README.md`.
5. `docs: S32 verified`: the session folder, `docs/requirements.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plans/`, `docs/sessions/README.md`.
