# S38: Scores across Question Sets

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 1 of 8.
**Requirements:** SCR-01, SCR-02 ([saas-requirements.md](../../saas-requirements.md) §8); CAL-02, CAL-03 ([requirements.md](../../requirements.md)). SCR-04 is question 2.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S37 verified (plan C5 done).

**Why first:** scores and bands feed the suggested ASA (S39) and the rules (S40).

**What exists today (much of SCR-01 is built, from plan-features step 9):**
- Options carry a `score` (the option editor; the card shows "2 pts", `ElementCard.tsx`).
- A **Calculation** element (SurveyJS `expression`) with a form in `packages/clinical/src/fields.ts` (`calcFormOf`, `formatCalc`): a score total `score('q_a', 'q_b')`, years since a date `age({q})`, BMI `bmi({h}, {w})`, or a custom formula.
- The functions `score()`, `band()` and `bmi()` are registered in `packages/clinical/src/questionTypes.ts` (`registerClinicalFunctions`).
- **Bands** on a Calculation (`bands: [{id, label, max, clinicalOutputs}]`, `bandsOf`/`bandFor` in `logic.ts`): each band can carry disclosures (`DisclosureModal.tsx`), `computeOutputs` (`outputs.ts`) produces them, and `band('<calc>')` works in conditions (the condition builder offers "<calc> band"). `showBands` shows "5 (High)".
- The patient's age and sex are survey variables (`{patientAge}`, `{patientSex}`; `respondent.ts` `patientVariables`, `testcases.ts` `modelFor`).
- **A calculation can only use its own Question Set:** `logicProblems` (`logic.ts`, `unknownReferences`) flags anything else. Only a Question Set's own condition may test earlier sets (`questionSets.ts`, `chapterConditionProblems`, `combineChapters`; `respondent.ts` `shownSets` passes earlier answers).

**Ask Rahul first:**
1. **Points for age, sex and BMI.** STOP-Bang gives 1 point for "over 50", 1 for "male" and 1 for "BMI over 35". How does an author add these to a score?
   - (a) **The score form gets "Add points when…" lines**, e.g. "Add 1 point when Patient's age is over 50", "Add 1 point when Sex is Male", "Add 1 point when BMI (the calculation) is over 35".
   - (b) The author types a formula by hand, e.g. `score('q_snore') + iif({patientAge} > 50, 1, 0)`.

   **Recommended: (a).** Authors never type formulas; (b) still works as a custom formula.
2. **Ready-made scores** (STOP-Bang, Apfel and the like, with fixed wording and scoring, inserted into an HQ in one go).
   - (a) **Not now: build them with the assessments in plan C7**, which needs the same ready-made scores. In C6 authors build scores themselves.
   - (b) Now, in an extra session before S39.

   **Recommended: (a).** C7 is where the validated scores are checked against their published sources.

## Build

- **`packages/clinical`, test-first (Vitest):**
  - **a score may use questions from earlier Question Sets**, by their key (the question's `name`): the same rule as a Question Set's condition (earlier sets only, never later ones). A question in a set that isn't shown counts as unanswered. `logicProblems` allows these references and still flags missing or later ones;
  - each set's survey gets the earlier shown sets' answers and their options' scores, so `score()` and `band()` work in the patient view, the clinician view (Validate), the preview, test cases and the POA Summary alike. One helper builds this, used everywhere (it's also what S41's evaluate service will call);
  - the age, sex and BMI points as answered in question 1 (unit tests: each kind of line, the edge values, a missing answer adds nothing);
  - BMI from the HQ: a BMI Calculation (in this set or an earlier one) can be used in a points line or a band. Observations are plan C7.
- **Editor:** the score form's question picker also lists earlier sets' choice questions, grouped "Set › Page" (as the Question Set condition builder does). Lines for question 1's answer.
- **Preview** (`features/preview/PreviewPage.tsx`, one Question Set at a time): it can't see earlier sets' answers, so such a score shows a short note ("Uses answers from earlier Question Sets, which count as unanswered here"). Test cases can give those answers (keep it simple; the builder proposes the exact way in the build notes).
- **No Go change:** Go still never evaluates. If a key check needs to know about cross-set references (`internal/keys`), add a unit test.
- **E2E:** extend J10 (`e2e/journeys/test_j10_authoring.py`): an author adds a score in set 2 using a question from set 1 and the patient's age, with two bands, one carrying a disclosure; the patient (phone) answers; Validate shows the band's disclosure.

## Not in this session

- ASA from codes and the suggested ASA with reasons (S39); rules (S40); server-side evaluation (S41).
- Ready-made validated scores (question 2); observations as a BMI source (C7).
- Using a score from another questionnaire or episode.

## Manual test focus

1. In set 2, make a score from two questions in set 1 and one in set 2. Publish, send it as a patient, and open Validate: the total and band are right. Set 2's preview shows the note about earlier sets.
2. A score that names a question in a later set, or a deleted one, is flagged and blocks Publish.
3. Age, sex and BMI points (as answered): a patient aged 51 gets the point, aged 50 doesn't.
4. Hide set 1 by its condition: its answers no longer count in set 2's score.
5. A band's disclosure appears on Validate and the POA Summary.
6. Old HQs with single-set scores behave exactly as before.
