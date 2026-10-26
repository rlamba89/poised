# S38 build notes: scores across Question Sets

Builder, automatic run, 26 Oct 2026. Rahul's answers ([answers.md](answers.md)) replace the brief's questions: **menu lines** ("Add N points when…", never a typed formula), **Incomplete, never low risk** for a missing input, and **STOP-Bang now**, ready-made and locked.

## What was built

### `packages/clinical` (Vitest, test-first)

- **`scores.ts` (new): the score form and the earlier sets' answers.**
  - A score is written as `iif(allAnswered('q_a', 'q_b', 'patientAge'), score('q_a', 'q_b') + iif({patientAge} > 50, 1, 0), '')`. `formatScore` writes it from the form (questions + **points lines**: `{points, rule}`), `parseScore` reads it back only on an exact round trip (anything else stays a custom formula).
  - **Incomplete:** `allAnswered()` (registered in `questionTypes.ts`) is false while any input has no answer: a question here that isn't shown (its own, its group's or its page's condition) counts as unanswered; an earlier set's answer counts only if that set was shown; the patient's sex "unknown" counts as missing. An Incomplete score has no value, so **no band, no band disclosure, and `band()` is empty**. `showBands` shows "Incomplete" (the survey's on-screen value).
  - **Earlier sets:** `Earlier = {doc, values}`; `applyEarlier(model, earlier)` sets each earlier answer as a survey variable (so `{q_a}` works), keeps the earlier JSON for `score()` and `band()` (options' points, bands), and re-runs the expressions (SurveyJS can't know `score('q_a')` depends on `q_a`).
  - `bandIn`: a band's optional **"also when"** condition (`when`, with `whenText` in words) picks it whatever the total (STOP-Bang's high-risk rule); otherwise by maximum as before.
  - `earlierKeysUsed(doc)`: the keys a set's Calculations use that it doesn't have.
- **`respondent.ts` `earlierContexts(sets, answers, viewer, patient)`: the one helper** every view uses. For each shown set, in order: the earlier shown sets' answers (only questions shown, plus every Calculation's value worked out with the sets before it) and their JSON. An HQ none of whose scores use another set gets nothing, so older HQs behave exactly as before. `isSetComplete` / `isSetReadyToSend` take it (a condition on a cross-set score's band).
- `modelFor(json, viewer, patient, earlier?)`, `runTestCase(json, tc, earlierDoc?)` (a case's `earlier` answers), `poaSummary` (builds the contexts itself).
- **`logicProblems(doc, {earlier, later})`:** a Calculation (and a band's condition) may use earlier sets' keys; a later set's is "it uses a question in a later Question Set (…)", a missing one "it tests something that no longer exists (…)" as before; an earlier answer that no longer exists is named. A question's own condition still may not test another set ("it tests a question in another Question Set (…)"). `publishProblems` passes both, and the earlier sets to test cases.
- **`pointsSubjects(doc, earlier, calc)`** (`logic.ts`): the menus: **Patient** (Age, Sex), **Measurements** (a BMI question's BMI, a BMI Calculation), **Other scores** (another Calculation's band), then choice and number questions grouped "Set › Page" (earlier sets first, then "This Question Set › Page").
- `computeOutputs` also returns `scores` (each score's value, band, or `incomplete`), shown in the Preview.
- References: `referencesIn` / `rewriteReferences` know `allAnswered()` and `'q.bmi'`; renames (Same as…) reach bands' conditions and test cases' `earlier` answers.
- **`ready.ts` (new): STOP-Bang, ready-made.** `stopBangGroup(bmi)`: a Group `stopbang` (`readyScore: "stopbang"`, `readyScoreBmi`) with the five yes/no questions (`stopbang_snoring` … `stopbang_neck`, 1 point for Yes), and the clinician-only score `stopbang_score`: questions + 1 point each for BMI over 35, age over 50, male; bands Low 0–2, Intermediate 3–4, High 5–8 **or** 2+ of the 4 STOP questions with male, BMI over 35 or a large neck. `readyScoreProblems` blocks publishing any change except the Group's condition and the disclosures. `ready.test.ts` holds its published scoring as worked examples (14 cases, edges included).

### `apps/web`

- **Editor:** the score form lists scored questions of this set and the earlier ones, grouped ("This Question Set", "Sleep › Sleep habits"), and has **Add points when…** lines (`PointsLines`, reusing the condition builder's `RuleRow`, without "is answered"). Unfinished lines aren't saved. The card says e.g. "Total of the scores of 3 questions (2 in earlier Question Sets), plus 1 points line". A custom formula may name earlier keys. The header's problem list checks against the earlier and later sets; deleting a question warns about other sets' scores that use it.
- **Ready-made score** button on the page's (or a Section's) Add bar → "Insert a ready-made score" (source, licence, **BMI from**: the BMI questions/Calculations of this and earlier sets; refused if the HQ has one). Its cards show **Ready-made**; the questions have no Copy/Move/Delete and can't be dragged; their Settings are read-only with a note (source, licence), no Logic tab, Disclosures allowed; the Group can be moved, deleted and given a condition. Nothing can be added into it. The score's Settings show "Also High risk when: …".
- **Every view gets the earlier answers:** `PatientHQ` (per set, and for "ready to send"), `ValidateSet` (`clinicianModel(…, earlier)`), the POA Summary (`poaSummary`).
- **Preview:** when a score uses earlier sets: "Uses answers from earlier Question Sets, which count as unanswered here. To try them, give them below." with a box per earlier question used (choice → its options with points; number, a Calculation or a BMI → a number). The Clinical outputs panel lists **Scores** ("Sleep risk: 3 (High)" or Incomplete). A saved test case keeps the earlier answers (`earlier`), and Load gives them back. `SetCondition` now gets the earlier sets from the same fetch (`useEarlierSets`).
- **The age the HQ's rules use** on the episode page, Validate and the POA Summary is the patient's age **on the day they sent their answers** (`hqPatient`), else today's.

### Go

No change: Go never evaluates. Calculations aren't keyed (`keys.notAnswered`), and Go doesn't check references, so no key check needed cross-set knowledge (the brief's "if"); STOP-Bang's fixed keys and options are compatible across a trust's HQs.

### E2E

`e2e/journeys/test_j10_authoring.py` (data `e2e/data/scores-hq.json`):
- `test_a_score_across_question_sets`: in set 2 Alex builds "Sleep risk" from set 1's two questions (listed "Sleep › Sleep habits") and set 2's one, plus "1 point when Age > 40" from the Patient menu; Low/High bands, High with a disclosure (Respiratory). The set's Preview (clinician): the note, "Incomplete", then the earlier answers given → "3 (High)" and the disclosure. Published; the patient (46, iPhone/WebKit) answers both sets; Cara's Validate shows "3 (High)" and the note in the summary box; validated; the POA Summary lists it.
- `test_a_ready_made_stop_bang`: inserted with set 1's BMI (chosen already), locked cards (no Delete, read-only text, no Logic, Disclosures there), the high-risk rule in words, a second refused, publishes.

## Decided by the builder (automatic mode)

1. **"Missing" includes a question or set a condition hides** (safest reading of "never low risk", and the brief's "counts as unanswered"): a new score is Incomplete then. So an author shouldn't put a conditional question in a score. *Open for Rahul* (pending.md): should a question hidden on purpose add 0 instead?
2. **Older scores (`score('q_a', 'q_b')`) are unchanged:** they total what's answered, as before; saved again from the form, they become Incomplete-aware. A custom formula is never rewritten.
3. **Age on the day the patient sent their answers** (research §2.2), so "over 50" never changes later. Before sending, today's.
4. **STOP-Bang:** the updated version (neck 17 in / 43 cm men, 16 in / 41 cm women); BMI from the HQ (chosen at insert), age and sex from the patient's record; "unknown" sex → Incomplete, "other" → no point. Fixed keys, so once per HQ. Its published examples are **Vitest cases** (`ready.test.ts`, run by `make test`) rather than test cases in the HQ: a chapter test case compares disclosure lines, and STOP-Bang's bands have none until a trust adds them. Wording is ours, close to the published form: **a clinician should check it against the licensed form** when the UHN licence is in place (C10).
5. **What may change in a ready-made score:** the Group's condition and every disclosure (they're the trust's response); nothing else (Publish says "STOP-Bang has been changed… Delete it and insert it again").
6. **"Also when" bands only for ready-made scores:** no editor control for authors yet (the research suggested one; not in Rahul's answer).
7. **Points lines' menus** as in the research: Patient, Measurements (BMI only), Other scores (bands), Answers. No "is answered" comparison. Points may be any number.
8. **Only Calculations use earlier sets.** A question's own condition testing another set is still a publish problem (now named as such).
9. **Ready-made scores go on a page or in a Section**, not inside another Group.

## TypeScript for Rahul to review

- `packages/clinical/src/scores.ts`: the form ↔ expression, `allAnswered`'s rule (`inputAnswered`), `applyEarlier`, `bandIn`.
- `packages/clinical/src/respondent.ts` `earlierContexts`: the one helper; S41's evaluate service should call it.
- `packages/clinical/src/ready.ts`: STOP-Bang's definition and the lock. `ready.test.ts`: the published scoring.
- `packages/clinical/src/logic.ts` `logicProblems`, `pointsSubjects`.

## Tests

- `packages/clinical`: 343 passed (new: `scores.test.ts` 16, `ready.test.ts` 18; `fields.test.ts` updated for the new score format).
- `apps/web`: 270 passed (new `episodes/hq.test.ts`).
- `make lint`: clean. `make test`: Go ok, Vitest as above, CDK 124 passed. `make test-integration`: every package ok.
- `make e2e` (every journey, on a private stack on :3093/:8093 on the shared `sj` database, stopped afterwards): **85 passed, 5 skipped (Cognito-only), 1 failed**: J7's text count found 5 texts to its made-up mobile (one of 1,000 fictional numbers, reused by earlier runs on this long-lived database: the same flake J8 had before S30), not S38. Re-run, J7 and J10: **11 passed**.

## Hand checks for Rahul

- None expected beyond the test plan.

## Suggested commits

1. `S38: scores across Question Sets, points lines, Incomplete` — `packages/clinical/src/{scores,scores.test,fields,fields.test,logic,doc,library,questionTypes,outputs,testcases,respondent,poa,publish,index}.ts`; `apps/web/src/features/editor/{TypeSettings,ConditionBuilder,ElementCard,QuestionnaireEditor,SettingsPanel}.tsx`; `apps/web/src/features/{patient/PatientHQ,episodes/ValidateSet,episodes/PoaSummary,episodes/EpisodePage}.tsx`, `episodes/hq.ts`, `episodes/hq.test.ts`, `preview/{clinicianView.ts,PreviewPage.tsx,TestCases.tsx,SetCondition.tsx,EarlierAnswers.tsx}`.
2. `S38: ready-made STOP-Bang` — `packages/clinical/src/{ready,ready.test}.ts`; `apps/web/src/features/editor/{ReadyScoreWindow,AddContentBar}.tsx`, `context.ts`.
3. `S38: J10 extended` — `e2e/journeys/test_j10_authoring.py`, `e2e/data/scores-hq.json`, `e2e/README.md`.
4. `S38: docs` — this folder, `CLAUDE.md`, `apps/web/AGENTS.md`, `docs/decisions.md`, `docs/pending.md`, plan and sessions status.

## Test report 1 (25 of 25 passed; 1 failure found while exploring)

- **F-1 (Minor), fixed test-first:** after **Clear answers** in the Preview, the earlier answers in the blue box stopped counting and the summary box kept a stale disclosure. There were two causes:
  - SurveyJS's `clear()` also drops the survey variables (the sample patient and the earlier answers) and doesn't refresh the summary boxes. Now **Clear answers builds the survey again** (`PreviewPage.restart`), so the effects set everything once more.
  - **A deeper bug, found while writing the test:** SurveyJS skips re-running a Calculation when none of its `{…}` variables changed. A score names its questions as text (`score('q_a')`), so a score with a line such as `{patientAge} > 50` wasn't worked out again when one of its questions changed after the survey had loaded. That hit the Preview after Clear, a clinician's correction on Validate, and a patient answering. `registerClinicalFunctions` now sets `settings.expressionQuestionTrackDependencies = false`, so Calculations always run again (SurveyJS's older behaviour). Conditions are unaffected.
  - Tests: `scores.test.ts` "works out when this set is answered after the earlier answers are given" (also checks a change made afterwards); J10 `test_a_score_across_question_sets` now presses Clear answers in the Preview, checks that the score is Incomplete and the summary box is empty, re-answers and gets 3 (High) and the note back.
- **Console warning (`<div>` inside `<p>`)** from the Incomplete badge in the Preview's Scores list: fixed (the row is a `div`).
- **O-1 "1 points line":** kept on purpose. "Points line" is the control's name (**Add points line**).
- **O-2 header problems are the open Question Set's:** as before S38 (the header lists the open set's problems; the Publish window lists every set's). No change.
- **O-3 a failed Create episode:** not reproduced, and not S38.
- **T-14 at a true 390 px:** a hand check for Rahul (Chrome's window stops at 500 px). The J10 journey's patient runs at iPhone size (WebKit).
- After the fixes: `make lint` clean; `make test`: clinical 344 and web 270 passed, CDK 124 passed, Go ok; e2e J10 (5 passed), J3, J4, J5, J16 and Information (10 passed).

## Test report 2 (retest; 1 failure found while exploring)

- **F-1 (Minor), fixed test-first: the Preview and test cases counted an earlier set's given answers even when its condition hid it from the sample patient** (the real flow was right: Incomplete). New `givenEarlier(sets, given, patient, viewer)` (`questionSets.ts`) gives each answer to the set holding its key, skips a set its condition hides for that patient, and names the hidden sets. `runTestCase` now takes the earlier sets (not their combined JSON) and uses it, so Publish's test-case check matches the real HQ. The Preview's blue box says "The sample patient wouldn't see “Sleep”, so its answers don't count." *Simplification:* the Preview doesn't build each earlier set's survey, so a question hidden *inside* an earlier set still counts there (the real flow skips it).
  - Tests: `scores.test.ts` "leaves out the answers of an earlier set the case's patient wouldn't see"; J10 (`scores-hq.json`'s Sleep is now shown only to women): the Preview with a male sample patient shows the note, Incomplete and no disclosure, then 3 (High) again for a woman.
- **Console error (`key` spread into props)** in `EarlierAnswers.tsx`: fixed (`key` is passed directly).
- After the fixes: `make lint` clean; `make test`: clinical 345, web 270, CDK 124 passed, Go ok; e2e J10 5 passed.

## Test report 3 (all passed)

- Every case passed. One note, not changed: the Preview's earlier-answers box accepts a negative BMI (Preview only; in pending.md).
- Finished: the session row is **Tested**, plan C6's status and the plans table are updated, decisions.md has the S38 entry, pending.md has S38's notes.

## Verification 1 (needs fixes), fixed test-first

- **F1: a removed or changed points line wasn't saved while another line was unfinished.** New `linesToSave(draft, saved)` (`scores.ts`) saves the complete lines whenever they differ from what's saved; `PointsLines` uses it. Tests: `scores.test.ts` "saving points lines"; J10 adds a Sex line, an unfinished line, removes the Sex line, and finds "plus 1 points line" kept after a reload.
- **F2: a key repeated in two Question Sets crashed the score editor** (Mantine refuses repeated option values). `pointsSubjects` lists each key once (the last one added, so this set's, as `applyEarlier` does), and the picker's options come from the new `scoredQuestions(doc, earlier, calc)` (each key once, this set's winning, else the latest earlier set's). Tests: `scores.test.ts` "a key repeated in two Question Sets".
- **F3: inserting STOP-Bang now also refuses when any of its fixed keys is already in the HQ** (`alreadyHere(stopBangGroup(bmi))`, e.g. a question that adopted `stopbang_snoring` with Same as…).
- Minor: `PatientHQ` works out the earlier sets' answers only when the saved answers change (not on each keystroke); `isEmpty` is shared (`scores.ts`). The rest are in pending.md (the Preview's earlier Calculations, `hqPatient`'s time zone, the card's "not found", the global SurveyJS setting).
- Also new: J10 `test_a_repeated_key_does_not_break_the_score_editor` (a draft repeating a key in two sets opens the score's Settings; the key is listed once in both menus).
- After the fixes: `make lint` clean; `make test`: clinical 349, web 270, CDK 124 passed, Go ok; `make e2e` against `make dev`: **86 passed, 5 skipped** (the full run, started before the repeated-key journey was added; that one passed on its own run).

## Test report 4 (all passed, after verification 1's fixes)

- Finished again: the session row says **Tested** (test report 4), plan C6's status and the plans table say it awaits verification 2, and decisions.md's S38 entry records verification 1 and report 4.

## Verification 2 (needs fixes), fixed test-first

- **F4 (blocking): a Question Set condition on a cross-set score read the score's saved value**, which goes stale when the patient changes an earlier set (set 2's score is saved only when set 2 is saved). New **`hqSets(sets, answers, viewer, patient)`** (`respondent.ts`) does one pass: for an HQ whose scores use only their own set, the saved answers decide, as before (no survey built). Otherwise each shown set's survey is built with the sets before it, and the next set's condition sees the saved answers with each Calculation's **worked-out** value in place of the saved one (an Incomplete score has none). `shownSets` and `earlierContexts` are both built on it; the POA Summary and `PatientHQ` (cached on the saved answers) use it once.
  - Tests: `scores.test.ts` "Question Set conditions on a score that uses earlier sets": a set wrongly shown, one wrongly hidden (and one never saved), Send held back until the shown set is answered, and older HQs unchanged. J10 `test_a_set_shown_by_a_score_follows_a_changed_earlier_answer` (`data/score-condition-hq.json`, a patient-visible score): the patient's list drops the follow-up set after Snoring changes to No though Tiredness still holds 1, and shows it (Send disabled) the other way.
- **Minor, fixed:**
  - `awaitsCheck` (the pre-fill card, `features/patient/prefill.ts`) builds the set with its earlier answers, as the form does.
  - A rename (Same as…) also renames STOP-Bang's `readyScoreBmi` (`library.ts`); `ready.test.ts` checks Publish then has no problem.
  - `applyEarlier` counts only this set's own questions (not those inside a BMI question or a repeating group), so an earlier set's `height` still reaches a line; `scores.test.ts` checks it.
  - Deleting a Group warns about other sets' scores that use its questions (e.g. STOP-Bang's).
  - The `structuredClone` before `findElement` stays: `findElement` gives a page without elements an empty list, and the store's documents mustn't change.
- **To pending.md:** a Question Set condition (or any condition) on a **clinician-only** score can't work in the patient's form, since Go strips clinician-only content (as before S38 for a band in the same set); `earlierContexts`/`hqSets` build a survey per shown set even where one is needed.
- After the fixes: `make lint` clean; `make test`: clinical 355, web 270, CDK 124 passed, Go ok; `make e2e` against `make dev`: **88 passed, 5 skipped** (all journeys, J10's new ones included).

## Test report 5 (all passed, after verification 2's fix)

- Every case passed. Its exploring finding (a clinician's correction can bring back a set hidden when the patient sent, showing the patient's old unsent answers) is a question for Rahul in pending.md, with options; no code change, as the conductor asked. The delete warning's wording for a score is noted there too.
- Finished: the session row says **Tested** (test report 5); plan C6's status and the plans table say it awaits verification 3; decisions.md and pending.md are current.
