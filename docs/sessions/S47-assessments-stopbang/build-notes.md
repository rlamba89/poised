# S47 build notes: assessments, starting with STOP-Bang

[Brief](brief.md) · [Answers](answers.md) · [Test plan](test-plan.md). Plan C7, part 2 of 4. Built 5 Nov 2026.

## Answers (Rahul, 5 Nov)

All four recommended options: (1) a clinician adds an assessment from the episode page; (2) the neck item is a Yes/No the clinician answers; (3) a trust setting, on by default, puts the hospital BMI first; (4) an HQ holding STOP-Bang fills every item, and the clinician still confirms.

## What was built

### Go

- **Migration 00038 `assessments`:**
  - An assessment is a **platform questionnaire**: `questionnaires.assessment` names it (`stopbang`), with no trust and no author. `org_id` and `created_by` may now be NULL, but only together with `assessment` (a check). `questionnaire_versions.updated_by` and `chapters.updated_by` may be NULL too (the platform's rows).
  - `episode_forms.assessment` says which assessment a form is (required exactly when `kind = 'assessment'`), with a unique index: one of each per episode.
  - `episode_answers.sources` holds where each of the clinician's answers came from.
  - Confirm is the clinician row's `validated_at`, by `updated_by`. A save without confirming clears it, as Validate's stamp works.
- **sqlc:** making `org_id`, `created_by` and `updated_by` nullable would have changed about 35 call sites. Instead the trust's queries cast them (`sqlc.arg(org_id)::uuid`, `q.created_by::uuid AS created_by`), so the Go code is unchanged. Every query on a trust's questionnaires already filters by `org_id`, so the platform's never show in a trust's lists. Every form query already filters `kind = 'hq'`, so the patient's routes, the reminders and the patient's home never see an assessment.
- **`internal/assessment` (new):** the assessments' locked JSON (`stopbang.json`, embedded), `Lookup`, and `Check`, which checks what a clinician saves:
  - every answer is one of the assessment's questions;
  - each answer is an option the question offers, or a number within its limits;
  - each answer has exactly one source: hq, observation or clinician. The profile isn't stored.
- **Routes** (`internal/httpapi/assessments.go`), each `audited`, with Audit page labels:
  - `GET …/episodes/{eid}/assessments` (`assessment.list`): the episode's assessments (content, answers, sources, revision, Confirm), the ones still to add, and `obsPrecedence`;
  - `POST …/assessments {assessment}` (`assessment.add`): 400 for an unknown assessment, 409 if the episode already has it;
  - `PUT …/assessments/{fid} {data, sources, revision}` (`assessment.save`): clears the Confirm;
  - `POST …/assessments/{fid}/confirm` (same body, `assessment.confirm`).

  A stale revision is a 409 with the usual "changed somewhere else" message. A cancelled or archived episode is a 409, and `feature.assessments` off is a 403 on every route. The episode GET sends `assessmentsOn`. `lockedObservations` became the shared `lockedEpisode`.
- **Publishing the platform version:** the first add publishes STOP-Bang's version 1 (`publishedAssessment`, under an advisory lock). If Go's JSON later differs (new wording), the next add publishes a new version and retires the old one. Episodes keep the version they were given, so each shows what its clinician saw. No seed or deploy step is needed.
- **Setting:** `observations.precedence`, "Observations take precedence in scores": trust-only, a Bool, in the Observations group, on by default. This is the brief's `obs_precedence`, named like the other settings. The Settings page draws it with no new code.
- **Seed:** two made-up Trust A HQs for testing, **Demo sleep HQ** (S, T, O and P by STOP-Bang's keys, and a BMI) and **Demo STOP-Bang HQ** (the ready-made group and a BMI). Both are added to `TestSeedAndE2EQuestionnairesAgreeOnKeys`.

### TypeScript (for Rahul to review)

- **`packages/clinical/src/assessments.ts` (new):**
  - `assessmentContent("stopbang")` is the assessment's JSON: a BMI number question (`stopbang_bmi`, 5–150) plus `stopBangGroup("stopbang_bmi")` from `ready.ts`. The editor, the HQ and the assessment therefore share one wording, scoring and set of keys.
  - **`assessmentItems(content, from, saved)`** gives the 8 items in the acronym's order, each with a value, a source and whether it scores. It's a pure function with table tests, in Go terms. The rules:
    - **hq** items come by key from the HQ's shown sets and questions only: a question a condition hides counts as unanswered, and a value that isn't one of the assessment's options counts as missing.
    - **The BMI** is the hospital BMI first when the setting is on, the patient's reported BMI first when it's off, each falling back to the other.
    - **Age and sex** come from the profile. Sex Other or Not known is missing (as in the HQ, S44c).
    - **Saved items:** while the assessment is unconfirmed, the clinician's own items are kept and the rest are filled afresh. Once it's confirmed, the confirmed answers are shown as they are.
  - `withDraft` puts the screen's changes in place, as the clinician's. `assessmentAnswers` is what's stored. `assessmentResult` works out the score and band with the score's own expression ("5 (High risk)", or "Incomplete"). `countingHqAnswers` gives the clinician's validated copy, else the patient's answers.
  - **The JSON Go stores is written from this file** by `npm run assessments -w packages/clinical` (`scripts/assessments.mjs`, using esbuild). `assessments.test.ts` fails if `apps/api/internal/assessment/stopbang.json` differs.
- **`apps/web/src/features/episodes/assessments.ts`:** the types, `useAssessments` (doesn't call the routes when assessments are off), `prefillFrom` (the HQ's answers count only once the patient has sent them), `shownItems`, `saveRequest`, the words (`sourceText`, `valueText`, `confirmText`) and `poaLines`, which works out a confirmed score with the patient's age on the day of Confirm.
- **`AssessmentsPanel.tsx`:** the episode page's Assessments card, below Observations:
  - **Add assessment** (a menu of the ones still to add);
  - each assessment's items: letter, label and wording, a Yes/No pair or a BMI box, and the source badge, with a missing item in orange;
  - the result badge, "N of 8 filled", Confirm (only when nothing is missing), Save and Undo changes, and "Confirmed by … on …" or "Changed: confirm it again".
- **The episode page** loads the HQ, the observations and the assessments once and passes them to the cards (every read is an audit row). The POA Summary adds one line per assessment under Hospital BMI.

### E2E

**J12** (`e2e/journeys/test_j12_observations.py`) has two new cases:
- **The HQ with S, T, O and P:** a sent HQ, then a hospital BMI. STOP-Bang opens with 7 of 8 filled, each with its source, the neck missing. The clinician answers the neck and confirms, and the POA Summary shows "6 (High risk)" beside the hospital BMI. A change, then Save, gives "Not confirmed"; confirming again gives "7 (High risk)".
- **The HQ with the ready-made STOP-Bang:** 8 of 8 filled, waiting for Confirm. A hospital BMI recorded with the card open takes B's place at once. This found a bug, fixed: the BMI box kept the value it opened with.

New data: `e2e/data/stopbang-items-hq.json` and `stopbang-hq.json`.

## Decided by the builder

1. **An assessment is a platform questionnaire with real versions,** not a definition held only in code. An episode's form points at the exact version (and so the exact wording) its clinician saw, and the answers sit in `episode_answers` as the brief says.
2. **The score isn't stored.** Go doesn't evaluate (the key rule). The browser works out a confirmed assessment's score from the stored answers and the patient's age and sex on the day of Confirm, as the HQ uses the day of sending (S38). If the patient record's sex is changed later, a confirmed score follows it, as the HQ's scores do.
3. **The patient's HQ answers count only once they've sent them.** Before that they may still change them. The clinician's validated copy counts once a set is validated (the brief).
4. **While an assessment is unconfirmed, the HQ and observation items are filled afresh on each open.** The HQ or a reading may have changed, and the trust's setting may have too. Items the clinician entered are kept. Once confirmed, it shows exactly what was confirmed.
5. **With the setting off, the patient's BMI comes first but the hospital BMI still fills a missing one,** and the other way round with it on. "Off" chooses which comes first; it doesn't leave the item empty when a value exists.
6. **Sex Other or Not known leaves G missing,** so the score stays Incomplete and can't be confirmed, as in the HQ (S44c, Rahul 28 Oct). The card says why. A clinician can't change the profile's age or sex in the assessment; they come from the patient's record.
7. **Confirm needs every item answered** (the browser's check). Go can't tell, as it doesn't evaluate; it checks only the shape.
8. **An assessment may be added at any status except cancelled or archived,** including before the patient sends the HQ. Adding one doesn't change the episode's revision and writes no General notes line, as readings don't.
9. **No removing an assessment** (the brief has none).
10. **The BMI box takes a plain decimal with at most one decimal place** (a comma allowed), like S46's reading boxes. Go refuses a value outside 5–150.

## Differences from the brief

- The setting is called `observations.precedence`, not `obs_precedence`, to match the other settings' keys (`observations.ranges`). It's the same setting.
- The brief's "a Go file (or seed) defines each assessment" is a Go embedded JSON file, written from `packages/clinical`. The platform's questionnaire row is made by the first add, not by the seed, so QA and new databases need no step.

## Test results (5 Nov)

- `make lint test`: pass. packages/clinical 511 tests, web 348, evaluate 25, CDK 132 (Python), and every Go package.
- `make test-integration`: pass. That includes 31 new assessment cases (the five per route, plus a duplicate add, a new version when the wording changes, a stale revision, cancelled or archived, assessments off, the patient's routes, the audit rows) and `TestAssessmentsMigration` (the checks, and rolling back and forward).
- `make e2e` (against `make dev`): **111 passed, 5 skipped** (the Cognito journeys, AWS only) in 10 min 58 s.

## Fixes after test report 1 (5 Nov)

Report 1: 20 of 21 passed, 1 failed (F-1), and 8 notes from exploring.

- **F-1 (T-10): "STOP-Bang is already on this episode." vanished.** The message was kept in the Add assessment menu, and the reload after the 409 left nothing to add, so the menu (and its message) went. The card now keeps the add's message itself, with a close button. J12: `test_a_second_tab_adding_stop_bang_is_told_it_is_already_there` (two tabs).
- **Note 1: the source badges were cut off** ("PATIENT REPORTE…"). They're in normal case now, and the column fits its words. J12 checks that the badge's text fits its box at 1456 wide.
- **Note 2: an out-of-range BMI wasn't flagged in the browser.**
  - The BMI item carries its question's limits (`min`, `max` on `AssessmentItem`).
  - The box says "Between 5 and 150", and Save and Confirm wait while a box holds anything that isn't a value. This includes "abc" after a valid number: before the fix, Save would quietly have sent the earlier number.
  - Go's refusal names the limits: "BMI: a number from 5 to 150." (`assessment.Check`).
  - `bmiInput` in `apps/web/.../assessments.ts` has table tests, and J12 has `test_a_bmi_outside_its_limits_is_flagged_and_not_saved`.
  - Emptying a box that showed a value says "Enter a BMI"; Undo changes brings the value back.
- **Note 3: a read-only card still asked for answers.** On a cancelled or archived episode it now says only "6 of 8 filled." (`filledText`, tested).
- **Note 5: a change back to the HQ's answer stayed "Clinician".** A change back to the value shown keeps that value's source (`withDraft`, tested), and the card no longer counts it as a change: no Save, no "Changed: confirm it again". This covers changes on the same screen. An item the clinician saved earlier stays the clinician's.
- **Note 7:** the test plan now expects "Hospital BMI 37 (measured …)", as S46's card shows it.
- **Not changed:**
  - **Note 4** (a confirmed STOP-Bang beside a newer hospital BMI) is a question for Rahul, below.
  - **Note 6** (Enter in the BMI box): the card has two actions, Save and Confirm, so Enter has no obvious one. The buttons are reached by Tab.
- **Tests after the fixes:**
  - `make lint test`: pass. clinical 513, web 355.
  - `make test-integration`: pass.
  - J12: 12 passed.

**Open question for Rahul (note 4):** when a confirmed assessment's pre-filled value has changed since Confirm (a corrected weight, say), should the card say so? Recommended: a line on the card, "The hospital BMI is now 24.7 (37 when confirmed)", with nothing changed until a clinician re-confirms. The alternative is to leave it as it is (the confirmed record shows what was confirmed).

## Test report 2 (5 Nov): all passed, and two fixes after it

Retest: 22 of 22 passed. Two of the notes from exploring were fixed afterwards, each with a J12 check that fails without it:

- **Note 1: the score went on using a BMI the box no longer held.** For example, "2" typed after 36.5 still scored B's point, and an emptied box still counted as filled. Now B shows **Missing** while its box holds a mistake or is emptied, so the count says 7 of 8 and the score is Incomplete (`withoutInvalid`, tested). Save and Confirm were already greyed out, so nothing wrong could be saved.
- **Note 2: the Yes/No buttons and the BMI box moved sideways** when a wider source badge appeared. The source column is now a fixed 170 px, wide enough for "Patient reported BMI".
- **Note 3** is still Rahul's open question (a confirmed assessment beside a newer reading, above).
- **Tests after the fixes:**
  - `make lint test`: pass. web 357, clinical 513.
  - J12: 12 passed.
  - These two fixes weren't retested by hand; the verifier should spot-check B's box.

## Fixes after verification 1 (5 Nov)

The verdict was Needs fixes ([verify.md](verify.md)). Findings 1 and 2 are fixed, and so are minors 3, 4 and 5. Each was test-first.

- **Finding 1: a re-confirm showed a score with the old Confirm day's age.** While a confirmed assessment was changed, the card still worked A out with the patient's age on the earlier Confirm day. Confirm stamps today, so the POA Summary could then show a score the clinician never saw (e.g. 50 then, 51 now).
  - Now the Confirm day's age is used only while the assessment is shown as confirmed and unchanged. Once it's changed, the age is today's, the day Confirm will record. The note under the card follows.
  - `scoreDay` in `apps/web/.../assessments.ts` is tested with a patient who was 50 at the first Confirm.
  - No J12 check: it needs a Confirm stamped on an earlier day.
- **Finding 2: the card could be confirmed before the observations had loaded, or after their load failed.** B would then quietly have been the patient's BMI, against the trust's setting.
  - `inputsState` (tested) says whether the HQ and the observations are in (the observations only when they're on).
  - Until they are, Save and Confirm are off. If either failed to load, the card says which, e.g. "The observations didn't load, so the hospital BMI can't be checked. Reload the page to save or confirm."
  - J12: `test_stop_bang_waits_for_the_observations` (the observations' route fails); it fails without the fix.
- **Minor 3: Go now refuses a Confirm with an item unanswered.** It returns 400 "Answer every item before confirming: Neck has no answer." `assessment.Unanswered` only walks the JSON, which the key rule allows. A Save without every item still works. Covered by `TestUnanswered` and a `TestSaveAndConfirmAssessment` case.
- **Minor 4: a pre-filled BMI outside the question's limits (5–150) isn't used.** The other BMI is used instead, else B is missing (`assessmentItems`, tested).
- **Minor 5:** `packages/clinical` lists `esbuild` 0.28.2 (the version `apps/evaluate` uses) in its devDependencies. `package-lock.json` changed only by esbuild's platform packages losing their `peer` flag.
- **Minor 6** (Go's list of assessments kept by hand) is added to the S48 brief, where three more assessments arrive.
- **Tests:**
  - `make lint test`: pass. web 364, clinical 514.
  - `make test-integration`: pass.
  - J12: 13 passed.

**Test report 3 (5 Nov): 17 of 17 passed,** including T-18 and report 2's two fixes. The tester made the observations fail by replacing the page's `fetch`, because Claude in Chrome can't block requests in DevTools; J12 covers it with a failing route. There are no new findings. Rahul's open question (a confirmed assessment beside a newer reading) is still open.

## Fixes after verification 2 (6 Nov)

The verdict was Needs fixes, with one blocking finding ([verify.md](verify.md), verification 2).

- **Finding 1: the POA Summary silently left out a confirmed assessment when the assessments failed to load.** It now shows one line, "Assessments: Couldn't be loaded", as the Hospital BMI line does. A missing High-risk STOP-Bang would have read as "no assessment done".
  - `poaLines` takes the load's error. It's unit-tested for an error, loading, off, confirmed and not confirmed.
  - J12: `test_the_poa_summary_says_when_the_assessments_did_not_load`. It fails without the fix.
- **Minor 2: a BMI box that started empty couldn't be emptied again after a value was typed.** It showed "Enter a BMI" and blocked Save, and Undo changes threw away the other items' changes too.
  - Emptying such a box now leaves B missing, as it started, and isn't a change.
  - A box that started with a value still asks for one ("Enter a BMI").
  - J12's BMI check covers both, and it failed before the fix.
- **Minor 3 is Rahul's question:** which HQ answers the pre-fill uses mid-validation (see the open question below).
- **Filed elsewhere:**
  - minors 4 and 6 are in [pending.md](../../pending.md) (S47's verification 2);
  - minors 5 and 7 are in the S48 brief.
- **Tests:**
  - `make lint test`: pass.
  - `make test-integration`: pass.
  - J12: 14 passed.

**Open question for Rahul (verification 2, minor 3), answered 6 Nov (answer 5, below):**
- **The question:** while a clinician is validating a Question Set but hasn't pressed Validate, should STOP-Bang's pre-fill use their copy?
- **Today:** it uses the clinician's copy only once the set is validated, as the brief says. The rest of the episode page (the Question Sets card, the suggested ASA, Complete review) uses the clinician's copy as soon as they've started.
- **Recommended:** use it once started, like the rest of the page. The same clinician confirms the assessment.

## After verification 3: Rahul's answers 5–7 (6 Nov)

Verification 3 found the build sound and sent it back for Rahul's three answers ([verify.md](verify.md), [answers.md](answers.md) 5–7). Verification 2's open question (minor 3) is answer 5.

- **Answer 5: the pre-fill uses the clinician's copy of a set as soon as they've started it.**
  - `countingHqAnswers` (clinical `assessments.ts`) now matches `currentAnswers` in `hq.ts`: the clinician's row if there is one, else the patient's.
  - Validate starts from the patient's answers, so a started copy holds every answer, not only the changed ones; the web tests' fixture does too.
  - Tests:
    - clinical: not started, started, validated;
    - web: the same through `prefillFrom`;
    - J12 `test_a_confirmed_stop_bang_says_what_has_changed_since`: T reads No after a part-done Validate.
- **Answer 7: age and sex are frozen at Confirm.**
  - `assessmentAnswers` now stores `patientAge` and `patientSex` with source `profile`, on every save. Only a Confirm's are ever read back.
  - `assessmentItems` shows a confirmed row's stored values.
  - `assessmentResult` uses an age and sex stored in the answers over the patient it's given. The card and `poaLines` both go through it, so neither moves.
  - A row confirmed before this has none stored, so it falls back to the record on the day of Confirm, as before.
  - Go's `assessment.Check` accepts them:
    - only with source `profile`, and only where the content names `{patientAge}` / `{patientSex}`;
    - the age must be a whole number from 0 to 150, and the sex `male` or `female` (Other and Not known leave G missing, as before);
    - `profile` on any other item is refused, as is another source on these two.
  - Tests:
    - `TestCheck` (8 new cases);
    - integration: a Confirm storing them, with `profile` on the neck and `clinician` on the age refused (400) for both routes;
    - clinical: a confirmed score unmoved by a changed date of birth or sex, an old row's fallback, and an unconfirmed row following the record;
    - web: the POA Summary's line unmoved by a corrected record.
  - The card's notes now read the items: "The age is the patient's on the day it was confirmed (66)" uses the stored age, and the "doesn't say male or female" note shows only when G is missing.
- **Answer 6: a confirmed assessment says when a value it used has changed since.**
  - `sinceConfirmed(confirmed, fresh)` in web `assessments.ts` gives one sentence per change. `fresh` is a pre-fill without the saved row, with the record as of the day of Confirm, so a birthday since isn't a change but a corrected date of birth is.
  - The sentences:
    - "The hospital BMI is now 24.7 (36.4 when confirmed).";
    - "The patient reported BMI is now 30 (Hospital BMI 36.4 when confirmed).";
    - "There's no BMI now (…)";
    - "The HQ now says No to Snoring (Yes when confirmed).";
    - "The HQ no longer answers Snoring (…)";
    - "The patient's record now gives an age of 36 that day (66 when confirmed).";
    - "The patient's record now says Female (Male when confirmed)." and "The patient's record no longer says male or female (…)".
  - The clinician's own answers aren't compared.
  - The card shows them in a yellow box (`assessment-since`), only while the assessment is shown as confirmed and unchanged, with "The score stays as confirmed. To use a new value, change the item and confirm again."
  - Tests:
    - web: 7 cases;
    - J12: confirm at a hospital BMI of 37, correct the weight, see "The hospital BMI is now 27.8 (37 when confirmed).", and the score and the POA Summary stay 5 (High risk).
- **Shown red first:**
  - every new unit and Go case;
  - the integration case (with the profile check turned off);
  - the J12 journey, twice: with the old `countingHqAnswers` it stopped at T, and without the since box at the line.
- **Decided by the builder:**
  - **The age compared is the record's on the day of Confirm, not today's,** so a birthday since doesn't raise a line.
  - **No "Use the new values" button.** Rahul's answer was one line on the card. To use a new hospital BMI the clinician types it, so B says "Entered by clinician". A button that takes the new values with their sources is a small follow-up if clinicians want it; it's in pending.md.
  - **Age and sex go on every save, not only Confirm.** It's one path, and an unconfirmed row's are never read.
- **Tests (6 Nov):**
  - `make lint test`: pass (web 373, clinical 518, evaluate 25, CDK 132, Go);
  - `make test-integration`: pass;
  - J12: 15 passed. It ran on a second local stack (API :8082, web :3002), as Rahul's `make dev` API predates the Go change: **restart `make dev` before the manual round.**

## Fix after test report 4 (6 Nov)

Report 4: 6 of 7 passed. **F-1 (Major):** after a confirmed STOP-Bang was changed, A and G kept the frozen age and sex (66, Male) instead of the record's (36, Female). A Confirm from there stored the old values again, and the yellow box came straight back.

- **Cause:** once changed, the card still built its items from the confirmed row, which reads the stored age and sex. The unit case "an unconfirmed row follows the record" covered a saved unconfirmed row, not a confirmed one with unsaved changes.
- **Fix:** `changedItems` (`apps/web/src/features/episodes/assessments.ts`): once the card is changed, the age and sex come from the record as of today (the day a Confirm records). The other items keep their confirmed values until they're changed (answer 6). The card uses it whenever it's changed, so a Save or Confirm stores the record's age and sex.
- **The hint:** when the yellow box has an age or sex line, it adds "Once any item is changed, the age and sex are the record's.", because A and G can't be changed on the card.
- **Test:** web: "once changed, the age and sex follow the record, so a new Confirm stores them". It covers 36/female, the neck change as the clinician's, B still 37, the score 5 (High risk), and the save body. Shown red first (66, male). No journey: there's no screen that changes a patient's record (T-21 uses `psql`).
- **Test plan:** T-19 now says the Demo STOP-Bang HQ, as the tester ran it. The Demo sleep HQ's episodes are auto-triaged by the default Low risk rule, which completes the review.
- **Answered (Rahul, 6 Nov: leave it; answers.md 8):** if the *only* change since Confirm is the record's age or sex, the box can't be cleared without changing some other item, because a change back to the confirmed value isn't a change. The recommendation is to leave it (a corrected date of birth or sex after a Confirm is rare) and fold it into pending.md's "Use the new values" button if clinicians ask.
- **Tests (6 Nov):**
  - `make lint test`: pass (web 374, clinical 518, evaluate 25, CDK 132, Go).
  - J12 against `make dev`: 15 passed on two runs. One earlier run, started straight after the edits, failed `test_an_hq_with_stop_bang_opens_it_complete_waiting_for_confirm` once. That journey passed alone and on both full runs since. Its trace was gone, and the most likely cause is Vite reloading the edited files.

## Test report 5 (6 Nov): all passed

6 of 6: T-21 in full with the extended step 4 (36 / Female and 4 (High risk) after the change; stored `patientAge 36`, `patientSex "female"` at Confirm; no yellow box after the reload), T-03, T-07, and T-01, T-02 and T-05 as setup. Notes:
- The tester's browser tool logged the submit's 204 as a 503. The app sent the HQ and the audit row says 204, so it's a tool fault, not the app's.
- The 4xx audit rows at 14:09–14:12 IST are the builder's J12 runs (the journeys test refusals), not the round's.

Next: re-verification (`/verify-session S47`, verification 4).

## Suggested commits (Rahul commits after verification)

1. **S47: assessments as platform questionnaires; the add, save and Confirm routes; observations.precedence.** Migration 00038; the queries (`assessments.sql` and the casts in `questionnaires.sql`, `chapters.sql`, `episodes.sql`, `patients.sql`, `accounts.sql` and `admin.sql`); `internal/assessment`; `httpapi/assessments.go`, router, audit labels, `assessmentsOn`; settings; the integration tests and the migration test. Tests: `go test ./internal/assessment ./internal/settings`, `TestAddAssessment`, `TestListAssessments`, `TestSaveAndConfirmAssessment`, `TestPatientNeverSeesAssessments`, `TestEpisodeSaysWhetherAssessmentsAreOn`, `TestAssessmentsAudited`, `TestAssessmentsMigration`.
2. **S47: STOP-Bang's pre-fill and score in packages/clinical; the JSON for Go.** `assessments.ts`, its tests, `scripts/assessments.mjs`, `stopbang.json`. Tests: `assessments.test.ts`.
3. **S47: the Assessments card, the POA Summary's line, the demo HQs; J12.** The web files, the seed's HQs and keys test, the e2e data and journeys. Tests: `apps/web` `assessments.test.ts`, `cmd/seed` tests, J12.
4. **S47: docs.**
