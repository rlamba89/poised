# S44a build notes: C6 follow-ups

Builder, automatic run, 27 Oct 2026. Every question was answered on 27 Oct ([answers.md](answers.md)); nothing new to ask.

## What was built

### 0. The local database

- **Reset** to migrated + seeded, keeping HJE Full HQ (its questionnaire, version and 4 chapters copied out with `\copy` and back after `make migrate seed`). 351 MB → 10 MB.
  - *Differs:* the classifier refused `DROP DATABASE`, so the old database was **renamed `sj_before_s44a`** (still in the Postgres container). Rahul can drop it when he's happy: `docker compose exec -T db psql -U sj -d postgres -c "DROP DATABASE sj_before_s44a"`.
- **S39's "E" drafts:** after the reset no version holds any test case (chapter or whole-HQ), so there was nothing to update and no script was written.
- **`test_audit.py` reads the audit CSV as a stream:** `support.stream_find` GETs the CSV with the request context's cookies (`urllib`) and checks it line by line; the export test reads its download with `csv.DictReader` over the open file. "Listed episodes" rows aren't capped.

### 1. A hidden score input adds 0 (`packages/clinical`, one place)

- `scores.ts`: **`inputHidden(survey, name)`**: a question in this set its condition, group or page hides (a Calculation is never hidden), or an earlier set's key listed in `Earlier.hidden`. `inputAnswered` counts a hidden input as answered; `score()` skips a hidden question's points; **`scoreAsShown`** leaves out an "Add N points when…" line that tests a hidden input.
- `Earlier.hidden` (new, optional): the earlier sets' keys a condition hid, from `hqSets` (every key of a hidden set, and each hidden question of a shown set) and `givenEarlier` (the Preview's and test cases': a hidden set's keys).
- `bandIn`: an "also when" runs with a hidden input's value removed.
- The editor's score help text says so.

**For Rahul (TypeScript):** the line rule is applied by wrapping SurveyJS's `SurveyModel.prototype.beforeExpressionRunning` once (in `registerClinicalFunctions`): SurveyJS passes every expression through it just before running it (it's what its `onExpressionRunning` event uses). For a score our form wrote, `scoreAsShown` rewrites the expression without the lines whose subject is hidden at that moment. Doing it on the prototype means every survey any view builds (patient, Validate, POA Summary, Preview, test cases, the evaluate service) gets it, without each place remembering to.

### 2. On send, only what the patient's form showed is kept

- `respondent.ts` **`shownAnswers(sets, answers, viewer, patient)`**: each set's answers without those to questions a condition hides there (their "Other" text too), `{}` for a hidden set. One pass (conditions worked out once on the answers given).
- `evaluate()` works its result out **from what was sent** and returns it as **`sent`** (by patient set). Shared fixture `evaluate.json` extended: Travel (hidden) has an answer, Sleep has a hidden `q_naps`; `sent` leaves both out.
- Go: `evaluate.Result.Sent`; **`keepSent`** (`patient.go`, both submit routes, after S44's incomplete check, in the send's transaction) only **removes** the keys the service left out of each patient row (never adds or changes an answer), with their pre-fill entries (`KeepSentAnswers`: `prefill` filtered to the kept keys, `{}` stays `{}` so `started` doesn't move). The service failing, or a reply naming no set, keeps everything (S41's fail-safe).
- The fake: `Recorder.HideAnswers(set, keys…)` / `ShowAnswers()`; by default `Sent` is the patient's answers.
- `form_outputs` stores the reply whole, so it now also holds the `sent` answers (*builder's choice*: "stored as it came", as S41).

### 3. Publish refuses a patient condition on clinician-only content

- `questionSets.ts` **`patientConditionProblems`** (in `publishProblems`, so the browser's window and the server's check through the evaluate service both refuse it): a patient set's own condition, or a page, question, option or skip rule in it (not a clinician-only element's own condition), testing a clinician-only element of this or an earlier set, or anything in an earlier clinician set. One message each, with how to fix it. Fixture case added to `publish_problems.json`. No e2e or seeded HQ trips it (checked over `e2e/data` and `apitest/testdata`).

### 4. The thank-you page

"Your answers have been sent to the hospital. We'll contact you about what happens next." (an archived episode's View: only the first sentence, as before). The page has no translations. J3 checks it at phone size.

### 5. The auto-triage warning

- `warnings.ts` **`autoTriageWarnings(sets, rules)`**: when a rule auto-triages, each score that needs a clinician-entered answer (a clinician-only question, a clinician set's question, or a score that needs one; its "also when" inputs too) is named with its inputs. Shown on the **Rules page** (yellow, "you can still publish") and in the **Publish window** ("Before you publish (these don't stop it)"). Not blocking.

### 6. The rules' suggested status

`status.ts` **`reachedOrPassed`** and `suggestionLines(suggestions, episode, history)`: a suggestion shows until the episode reaches or passes it on the main path; for On hold / Not ready, the history decides (ever at the status, or past it on the main path; an undone change doesn't count); none once cancelled, archived, admitted or discharged. J11b checks it goes after Complete review.

### 7. The verifiers' four defects

- **S42, a held message sent twice:** `SetEpisodeStatus` sets `triage_message_pending = false` on every change by hand. Test `TestAutoTriageMessages/On hold, then Ready for admission…`.
- **S44, a save slipping past Complete review:** `SaveClinicianAnswers` takes the episode row **`FOR NO KEY UPDATE OF e`**. *Differs from the brief (FOR SHARE):* the touch trigger updates the episode's row in the same statement, so two saves each holding SHARE would deadlock upgrading it; NO KEY UPDATE serialises saves on one episode (they're quick) and still conflicts with Complete review's FOR UPDATE. Test `TestClinicianSaveWaitsForCompleteReview` (both orders).
- **S41, a rules save racing Publish:** Publish re-reads the version under **`LockVersionForPublish`** (`FOR UPDATE`) after the trust's lock, checks it's still a draft, and sends *those* rules; the chapters are read after it. A rules save then waits and is refused (it saves only a draft). *Added:* `SaveChapterContent` takes the version row `FOR SHARE` (and saves only into a draft), so a Question Set save racing Publish waits and is refused too ("what's checked is what's published" needs the chapters as well). Tests in `TestPublishAsksTheEvaluator`.
- **S38, the score picker offering repeating-group questions:** `scoredQuestions` leaves out questions inside a repeating group (this set's and earlier ones'), as `pointsSubjects` does (`repeatingParents` now exported).

### 8. Small improvements

- **Messages:** a second Complete review → "The HQ review is already complete." (`reviewNotOpen`); a refused Checked on an episode already checked → "Someone checked this while you had it open." (code `stale`, so the page reloads to "Checked by…"). J11's two-tab journey checks the words.
- **ASA guide nudges:** `ClinicalOutput.asaNone` (`true` when the author chose No ASA grade; the editor starts from it); `ungradedAsaWarnings(sets, guide)` warns in the Publish window, which fetches each HQ code's guide examples as it opens (`asaChoicesOf`, one code search each).
- **The suggested ASA:** the episode page shows it only once the patient has sent (`patientSubmittedAt`); **Validate shows it too**, worked out again on every change (`hqSuggestedAsa` over the answers that count now with this set's as on screen). J5's ASA journey checks it changes as Cara corrects heart failure.
- **The age in the hospital's time zone:** `GET …/episodes/{eid}` now gives `hospitalTimeZone` (null: the deployment's); `hqPatient` passes it (or `deploymentTimeZone()`) to `patientVariables` → `ageFrom(dob, at, timeZone)`, which reads the date in that zone (`dateIn`, `Intl.DateTimeFormat`, no locale literal). The patient's own form already got its age from Go.

### 9. Authors' bands get "also when"

- `scores.ts`: `bandWhenSubjects` (the score itself, then the points lines' menus), `formatBandWhen` (complete rules joined with "and"), `parseBandWhen` (back to rules; undefined for one it can't show, e.g. STOP-Bang's with "or"), `bandWhenInputs`. **`allAnswered` also requires a band's "also when" inputs** (except the score itself), so a missing one makes the score Incomplete: the band can't be decided (*builder's reading* of "Incomplete inputs"). `bandIn` already honoured `when` for any band.
- Editor (`TypeSettings.tsx` `BandWhen`): under each band, **Also <band> when…** adds lines from the menus (`RuleRow`), saved as soon as complete; a ready-made score's rule shows as words, read-only. Publish names a line testing something deleted (`logicProblems` already checks `band` expressions).

### Tests

- Vitest (`packages/clinical`): scores (hidden question, hidden set, lines, a hidden page, the Calculation before the question it tests, the picker, "also when"), evaluate (`sent`, the result from what was sent), publish (`patientConditionProblems`), warnings, respondent (age in a zone). Web: `status.test.ts`.
- Go: `evaluate_test.go` (`Sent` from the fixture, `HideAnswers`); integration `TestSendKeepsOnlyWhatTheFormShowed` (hidden question and set dropped with their pre-fill, staff on behalf, all shown, evaluator down, a reply naming no set), `TestClinicianSaveWaitsForCompleteReview`, the Publish race cases, `TestAutoTriageMessages`, `TestTriageChecked`, `TestCompleteReview`'s "only once" wording, `TestGetEpisode`'s time zone.
- e2e: J3 (thank-you words), J5 (suggested ASA on Validate), J11 (Checked wording), J11b (suggestion gone after Complete review), J10 (a hidden set now adds 0 in the Preview: "2 (Low)"), J16 (re-opens the seeded demo HQ's auto-triaged send before validating: after the reset its v1 has the default rules), `test_audit.py` (stream).

## Decided by the builder (automatic mode)

1. The old database renamed, not dropped (the drop was refused); Rahul drops `sj_before_s44a` when he wants.
2. Hidden line rule through SurveyJS's `beforeExpressionRunning`, once on the prototype (above).
3. `shownAnswers` is one pass; a stale answer on a hidden question that made another question visible keeps that other question's answer.
4. `SaveClinicianAnswers` takes `FOR NO KEY UPDATE`, not `FOR SHARE` (deadlock with the touch trigger).
5. Question Set content saves also wait for Publish (`FOR SHARE` on the version).
6. A band's "also when" inputs are score inputs (missing → Incomplete).
7. The Checked refusal on an already-checked episode keeps code `stale` (the page reloads), with its own words.
8. The thank-you page for an archived episode still says only the first sentence.
9. Validate's suggested ASA is worked out on every change (fine at today's sizes; debounce if a big HQ feels slow).

## Hand checks for Rahul

- The patient's pages at 390×844 if the tester's window can't be resized (T-04, T-12).
- Drop `sj_before_s44a` once happy with the reset.

## Test results (27 Oct)

- `make lint test`: pass (Go unit; Vitest web 288, evaluate 13, clinical 442; CDK 124).
- `make test-integration`: pass (all packages).
- `make e2e` against `make dev` (started by the builder on the fresh database, stopped after): first full run 95 passed, 3 failed, 5 skipped; the 3 (J10 Preview expectation, J16 on the reset database's auto-triaging demo HQ, J5's new step leaving the page on Validate) fixed and re-run: pass.

## Suggested commits

1. `S44a: reset the local database; test_audit reads the audit CSV as a stream` — `e2e/support.py`, `e2e/journeys/test_audit.py`, `e2e/journeys/test_j16_prefill.py`.
2. `S44a: a score input a condition hides adds 0` — `packages/clinical/src/scores.ts`, `questionTypes.ts`, `respondent.ts` (hqSets), `questionSets.ts` (givenEarlier), `scores.test.ts`, `apps/web/src/features/editor/TypeSettings.tsx` (help text), `e2e/journeys/test_j10_authoring.py`.
3. `S44a: on send, keep only what the patient's form showed` — `respondent.ts` (shownAnswers), `evaluate.ts`, `evaluate.test.ts`, `apps/api/internal/evaluate/*` (with `testdata/evaluate.json`), `db/queries/episodes.sql` (KeepSentAnswers), `internal/db/*`, `internal/httpapi/patient.go`, `server_checks_integration_test.go`.
4. `S44a: Publish refuses a patient condition on clinician-only content` — `questionSets.ts`, `publish.ts`, `publish.test.ts`, `testdata/publish_problems.json`.
5. `S44a: authors are warned when auto-triage can't work, and about ungraded ASA codes; No ASA grade remembered` — `warnings.ts`, `warnings.test.ts`, `index.ts`, `types.ts`, `RulesPage.tsx`, `PublishModal.tsx`, `OutputEditor.tsx`.
6. `S44a: the thank-you page's neutral wording; the suggested status hides once reached` — `PatientHQ.tsx`, `status.ts`, `status.test.ts`, `StatusControl.tsx`, `e2e/journeys/test_j3_patient.py`, `test_j11b_high_risk.py`.
7. `S44a: the verifiers' four defects` — `db/queries/episodes.sql` (SetEpisodeStatus, SaveClinicianAnswers), `questionnaires.sql` (LockVersionForPublish), `chapters.sql` (SaveChapterContent), `internal/db/*`, `questionnaires.go`, `scores.ts` (scoredQuestions), `logic.ts`, the integration tests (`triage_`, `server_checks_`, `evaluate_integration_test.go`), `scores.test.ts`.
8. `S44a: clearer messages, the suggested ASA on Validate, the age in the hospital's time zone` — `validation.go`, `statuses.go`, `validation_integration_test.go`, `triage_integration_test.go`, `EpisodePage.tsx`, `ValidateSet.tsx`, `hq.ts`, `types.ts` (web), `respondent.ts` (ageFrom), `respondent.test.ts`, `db/queries/episodes.sql` (GetEpisode), `episodes_integration_test.go`, `e2e/journeys/test_j5_poa_summary.py`, `test_j11_triage.py`.
9. `S44a: "also when" on authors' bands` — `scores.ts`, `questionTypes.ts` (allAnswered), `warnings.ts`, `TypeSettings.tsx`, `scores.test.ts`.
10. `S44a: docs` — this folder, `docs/pending.md`, `docs/decisions.md`, `docs/sessions/README.md`, `docs/plans/*`.

(Some files carry several steps; splitting them by hunk is fine, or fold 2–9 into fewer commits.)

## Test report 1 (27 Oct): 26 of 27 passed

- **F1 (T-06), fixed:** a set staff filled in for the patient whose answers were all dropped on send still said "Patient's answers entered by Cara Clinician" (the row's `updated_by`). `KeepSentAnswers` now clears `updated_by` when no answer is left (a set that keeps answers still names who entered them). Test: `TestSendKeepsOnlyWhatTheFormShowed/filled in by staff…`.
- **Exploring findings fixed (small, in scope):**
  - E1: the score card says "plus 1 line adding points" (not "1 points line"); J10 updated.
  - E3: the new publish message no longer ends a question with "?." ("…can't see Neck over 40 cm? Move…").
  - E4: a refused second Complete HQ review (409) reloads the episode and its sets, keeping the message, as a refused Checked does.
- **Not changed, noted in pending.md:**
  - E2: the card counting a deleted question as "in earlier Question Sets" (S38's, already listed).
  - E5: a suggested status writes no General notes line. That's S43's design: only tasks, flags and a new requirement do. CLAUDE.md's wording is loose.
  - E6: the POA Summary shows a score only through its band's disclosure. Older, not S44a's.
  - E7: once, not reproduced.
  - E8: a hidden set's Validate page opens by URL, empty. Expected.
- **Results:** `make lint test`: pass. `make test-integration`: pass. `make e2e` on the builder's own `make dev` (stopped after): 97 passed, 1 failed, 5 skipped. The failure, `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`, is a tooltip on hover that didn't appear; it passed twice when run alone, and the audit page is untouched (flaky, noted in pending.md).

## Test report 2 (27 Oct): all passed

9 of 9 (the retest of T-06, T-16, T-09, T-17 step 2, R-02, R-05, and T-01, T-04, T-07, a T-20 spot check). No code change. Exploring notes are informational only (Enter submits the date of birth; "Patient answered: no answer" under a clinician's answer on a set the patient never saw is accurate). Phone size: the tester's window couldn't resize this round; round 1 checked the thank-you page at 390×844 with Playwright, and nothing patient-facing changed since.

**Status:** Tested; verification 1 needed fixes (below).

## Verification 1 (27 Oct): needs fixes, done test-first

1. **The age under another calendar (th-TH, fa-IR):** `dateIn` now sets `calendar: "gregory"` (with Latin digits) whatever the browser's locale; `dateIn` takes an optional locale for the test (th-TH, fa-IR, ar-SA, ja-JP-u-ca-japanese all give 2026-12-11).
2. **A chained follow-up kept but hidden everywhere after:** `shownAnswers` repeats its pass until nothing changes (each pass only drops, so it ends; capped at 50), so what's frozen is exactly what every later view shows, and running it again changes nothing. Tests: the B→A→C chain gives `{qB: "no"}` and is stable; a set shown only through a hidden answer goes too; the shared fixture's Sleep gains `q_nap_hours`, shown only through the hidden `q_naps`, and `sent` leaves it out. Rahul's product question (the patient's screen can still show C) is in pending.md.
3. **Other Question Set changes racing Publish:** `CreateChapter`, `UpdateChapterMeta` and `DeleteChapter` write only into a draft, holding the version's row `FOR SHARE` (as `SaveChapterContent`); the reorder takes `LockDraftVersion` (`FOR SHARE`) in its transaction. A refused change answers 409 "Only draft versions can be changed.". `cmd/import-hq` adjusted to the new return values. Test: `TestPublishAsksTheEvaluator`'s audience, delete, add and reorder cases (each waits for the version, then is refused, and the version's sets are unchanged).
4. **A content save refused by Publish:** `saveContent` re-reads the version's status after no row and answers `notDraft` when it's been published (else the stale message as before). Asserted in the existing race case.
6. **`form_outputs` held a copy of the answers sent:** stored now without `sent` (`evaluate.Result.Stored`, unit test `TestStoredLeavesOutSent`; the send's integration test checks the stored row).
5, 7 (and 2's question for Rahul): recorded in pending.md ("From S44a's build").

**Results:** `make lint test`: pass (clinical 444, web 288, evaluate 13, CDK 124). `make test-integration`: pass. `make e2e` on the builder's own `make dev` (stopped after): **98 passed, 5 skipped**.

**Suggested commits:** as above; finding 1 goes with commit 8, 2 with 3, 3 and 4 with 7, 6 with 3.

## Test report 3 (27 Oct): all passed

17 of 17 (the retest after verification 1: T-04, T-05, the new chain case, T-08 with a refused change after Publish, T-18, T-02, T-19–T-21, R-01–R-06, T-12, a full Complete HQ review). No code change. Exploring notes: the patient's screen still shows a chained follow-up that isn't kept (Rahul's question, already in pending.md); a high STOP-Bang alone doesn't stop the default Low risk rule (not S44a's; added to pending.md for Rahul).

**Status:** Tested; verification 2 needed one fix (below).

## Verification 2 (27 Oct): one fix, done test-first

- **A reorder deadlocked with a content save (or a second reorder):** `LockDraftVersion` took the version `FOR SHARE`, then the same transaction's `TouchVersion` updated that row, upgrading the lock while a content save held SHARE and waited for a set the reorder had moved. It now takes **`FOR NO KEY UPDATE`** (as `SaveClinicianAnswers` does, builder's decision 4): the reorder waits for a save in progress instead, and it still conflicts with Publish's FOR UPDATE. Test: `TestPublishAsksTheEvaluator/a reorder while a Question Set save holds the version` (reproduced the 500 "deadlock detected" first; now 204, the order applied).
- **Recorded in pending.md:** the Complete review wording for an episode past review, S45 measuring Publish's evaluate call under the lock, and `scoreAsShown`'s unbounded cache. The duplicate "Built in S44a" on pending.md's S43 row is removed.
- **Results:** `make lint test`: pass. `make test-integration`: pass. `make e2e` on the builder's own `make dev` (stopped after): **98 passed, 5 skipped**.
- **Suggested commits:** the fix goes with commit 7.

## Test report 4 (27 Oct): all passed

11 of 11: the reorder with an autosave straight after it, two quick reorders, a reorder from a stale tab after Publish ("Only draft versions can be changed."), T-04, T-08, T-17 step 2, R-02. No code change. Exploring notes: the reorder route's refusals are as expected (403 for a clinician and a hospital-only super clinician, 404 for another trust, 409 for a wrong list, 400 for bad JSON); the "Patient answered: no answer" wording on a set the patient never saw is in pending.md for Rahul.

**Status:** Verified (27 Oct, verification 3; see [verify.md](verify.md)).
