# S44c build notes

Builder, automatic run, 28 Oct 2026. Every item was answered on 28 Oct ([answers.md](answers.md)); nothing was open, and nothing new needed Rahul.

## What was built

1. **The patient's form clears a hidden answer at once** (answer 1).
   - `packages/clinical/src/respondent.ts` **`loadForPatient(model, data)`** (named `clearHiddenAnswers` until verification 1, F1): the answers, then SurveyJS's documented `clearInvisibleValues: "onHiddenContainer"` (survey-core 3.1.2's typings: "Clears a question value when the question or its container (page or panel) becomes invisible. If the question is invisible initially, its value is removed on survey completion."). Each clearing fires `onValueChanged`, so the form's autosave sends it like any other change. `PatientHQ`'s `SetForm` calls it, so the patient's form and staff's **Fill in for the patient** both clear.
   - **A pre-filled answer in a question hidden when the set opens** stays until the patient finishes the set ("Finish this section" clears it, and saves), and a send drops it anyway (S44a's `keepSent`, unchanged, the backstop).
   - **Whole Question Sets** (and questions in other sets): **`clearedAnswers(sets, before, after, patient, changed)`** compares `shownAnswers` (S44a) before and after the change and returns each other set's answers without the ones this change hid. `PatientHQ` runs it after every save the server accepts (`onSaved`), saves each such set through its own saver and puts the cleared answers on screen. Only answers this change hid go (an answer already hidden is left for the send). It builds the sets' surveys only when another set mentions a changed key (a cheap check over the other sets' JSON), since the form saves on every pause in typing.
2. **A held auto-triage message stays held after Undo or un-archive** (answer 2). `SetEpisodeStatus` now keeps `triage_message_pending` unless the change sends its own message (`end_wait`: `episode.StatusMessage(move, to) != ""`, i.e. Ready for admission by hand, Cancel) or is a re-open. Checked still sends it only while the episode is at the status it tells the patient about. `GET …/episodes/{eid}` shows `triageMessagePending` true only there, so the card doesn't say "the message goes when you click Checked" while On hold or archived.
3. **A High STOP-Bang holds back the default Low risk auto-triage** (answer 3). A new rule test, **`when.notBand: {score, bands}`**: the score is in none of these bands; an HQ without the score (or an Incomplete score, which holds auto-triage anyway) passes. In `rules.ts` (`matches`, `whenWords`: "STOP-Bang score isn't High risk", plus " (not in this HQ)" when the HQ hasn't it; the Publish window's short line leaves it out then; `ruleProblems`: only a band the score lacks), Go's `asarules` (`When.NotBand`, the same shape check as `band`), the Rules page's form ("and a score isn't in a band", offering a missing ready-made score as "STOP-Bang score (not in this HQ)", `notBandChoices`). **`defaults.json`**'s Low risk rule has `"notBand": {"score": "stopbang_score", "bands": ["b_high"]}`. Only new HQs (and new versions of HQs without rules) get it; existing HQs keep their rules.
4. **STOP-Bang is Incomplete for sex "Other"** (answer 4): `allAnswered` passes `otherSexUnscored` to `inputAnswered` when it's working out the ready-made STOP-Bang's score. Its score settings say "A patient whose sex is Not known or Other makes STOP-Bang Incomplete." An author's own score is unchanged ("Other" is an answer, with no male point).
5. **No "Patient answered" line on a question the patient never saw** (answer 5): **`shownToPatient(sets, sentAnswers, patient)`** gives, per patient set shown, the questions the patient's form showed for the answers they sent (a send keeps only those, so it's what they saw). Validate's `showCorrections` and `poaSummary`'s changes (and its "Changed by clinician" badge) use it.
6. **Small fixes** (S44a verifications 2–3):
   - `TouchVersion` updates only a draft (`AND status = 'draft'`).
   - A clinician's save with no row re-reads the episode's status, so one that waited for a Complete review says "The HQ review is already complete, so the answers can't be changed.".
   - Rename or delete of a Question Set deleted meanwhile: 409 "This Question Set has been deleted." (the version is re-read: published → "Only draft versions can be changed." as before). **Also once it's gone** (a stale tab's rename, delete or content save): `loadChapterMeta`'s 404 says the same, instead of "Chapter not found.".
   - Complete review: `reviewNotOpen(status, reviewed)`: "The HQ review is already complete." only when `review_completed_at` is set, else "This episode isn't waiting for review any more." (On hold, Not ready, auto-triaged past review); a review that waited and found the episode moved on re-reads it.
7. **E2E:** J16 and J11 updated (above, in Tests). New **J3b** (`e2e/journeys/test_j3b_hidden_answers.py`, `e2e/data/hidden-answers-hq.json`): the chain and a follow-up set at phone size, the reload, Yes again empty, the stored rows, the send; then Validate and the POA Summary with no "Patient answered" line under questions the patient never saw.
8. **Docs:** pending.md (the S44c items struck through, two new notes), decisions.md (28 Oct, S44c), CLAUDE.md (two lines), `apps/web/AGENTS.md`, `e2e/README.md` (J3b), the C6 plan's Status and the plans and sessions tables.

## Decided by the builder (automatic mode)

- **"onHiddenContainer", not "onHidden":** the brief says the question, its page or its whole Question Set; "onHidden" wouldn't clear a question whose page or group is hidden.
- **Other sets are cleared after each accepted save, not on each keystroke**, and only for what this change hid (the before/after comparison), so an answer pre-filled into a hidden question isn't cleared just because the patient changed something unrelated.
- **The held message is kept through every change without its own message** (On hold, Not ready, Admit, archive), rather than restored only on Undo or un-archive: simpler (no history to read) and equivalent, since Checked sends only at the auto-triaged status and every change that sends a message ends it. So nothing goes twice.
- **`notBand` as a separate test** (not a "not" flag on `band`), so a rule can have both; an absent score passes, and isn't a publish problem, or every new HQ without STOP-Bang would be refused.
- **"Other" Incomplete only for STOP-Bang** (the brief's scope), not for every score with a sex line.
- **A question the patient never saw isn't a "change"** on the POA Summary either: no line in "Changed by the clinician on …" and no "Changed by clinician" badge on its note.
- **"This Question Set has been deleted." for the stale-tab 404 too**, as a tester would see it that way; another trust's set id gets the same 404 (no difference to learn from).

## TypeScript for Rahul to review

- `packages/clinical/src/respondent.ts`: `loadForPatient` (the answers, then one SurveyJS setting), `clearedAnswers` (before/after `shownAnswers`, with the "is it mentioned elsewhere" check first), `shownToPatient` (the patient view's visible questions per shown set).
- `apps/web/src/features/patient/PatientHQ.tsx`: `lastSaved` (what the server had at the last check), `clearHidden` (runs `clearedAnswers` after a save and saves each cleared set), `SetForm` calls `onSaved` after each accepted save and loads its model with `loadForPatient`.
- `packages/clinical/src/rules.ts`: `notBand` in `RuleWhen`, `matches`, `whenWords`, `shortWhen`, `ruleProblems`; `notBandChoices` (STOP-Bang's title and bands from `ready.ts`, for an HQ without it). `apps/web/src/features/rules/RuleForm.tsx`: the new checkbox and its two pickers.
- `packages/clinical/src/scores.ts` / `questionTypes.ts`: `inputAnswered(…, otherSexUnscored)`, set for STOP-Bang's score key.
- `packages/clinical/src/poa.ts`, `apps/web/src/features/episodes/corrections.ts`, `ValidateSet.tsx`: the `shownToPatient` check.
- `apps/web/src/features/editor/TypeSettings.tsx`: the STOP-Bang help line.

## Tests

- **TDD:** each part's tests were written first and seen failing (the rules and STOP-Bang tests, the Go shape test that didn't compile, the held-message and wording integration tests), except the shared helpers' own unit tests, written alongside.
- **Unit (Vitest, `packages/clinical`):** `respondent.test.ts` (clearing in the patient's form: the chain and onValueChanged, a hidden-at-open answer kept until Finish; `clearedAnswers`: a hidden set and the set it showed, nothing hidden, an answer already hidden left; `shownToPatient`), `rules.test.ts` (High → no Low risk; Low → auto-triage; an HQ without STOP-Bang; the Preview of the set with the earlier BMI; words; a missing band; an author's own `notBand`), `ready.test.ts` (sex Not known and Other Incomplete), `scores.test.ts` (an author's score still scores Other), `poa.test.ts` (no change line for a hidden set's or a hidden question's answer; a shown-but-skipped one keeps it).
- **Go unit:** `asarules_test.go` (the defaults' `notBand`; refused without a score or bands; kept with every other kind).
- **Integration:** `TestAutoTriageMessages` (On hold → Undo → Checked: once; archive → un-archive → Checked: once; On hold → Ready for admission → Checked: once, now checking what the page shows instead of the column), `TestCompleteReview` (past review without one, reviewed then On hold, moved on while waiting), `TestClinicianSaveWaitsForCompleteReview` (the message), `TestPublishAsksTheEvaluator` (rename/delete while another author deletes the set: 409; then 404 for rename, delete and content save; `TouchVersion` on published vs draft).
- **E2E:** J3b (new).
- Results (28 Oct):
  - `make lint test`: exit 0. Vitest 299 + 13 + 461 passed; CDK 124 passed; Go all ok.
  - `make test-integration`: exit 0 (every package ok).
  - `make e2e` against `make dev`: **99 passed, 5 skipped** (7 min 28 s; the 5 skipped are the AWS-only journeys). The first full run had 2 failures, both expected from this session's answers, and the journeys were updated: **J16** checked S25's old behaviour that a pre-filled answer hidden by its condition comes back when shown again (now it's cleared at once, comes back empty and is typed again); **J11** checked the defaults' Low risk words (now "…and STOP-Bang score isn't High risk (not in this HQ)…").

## Hand checks for Rahul

- None new. (Publish's "last changed by" race can't be timed by hand; the integration test covers the query.)

## Suggested commits

1. `S44c: the patient's form clears an answer as soon as its question is hidden, and other sets' answers a change hides` (respondent.ts + tests, PatientHQ.tsx, J3b and its data, J16's hidden pre-filled answer)
2. `S44c: a held auto-triage message stays held after Undo or un-archive` (episodes.sql, statuses.go, episodes.go, sqlc output, triage_integration_test.go)
3. `S44c: a High STOP-Bang holds back the default Low risk auto-triage (notBand)` (rules.ts + tests, asarules + tests, defaults.json, RuleForm.tsx, J11's words)
4. `S44c: STOP-Bang is Incomplete for sex Other` (scores.ts, questionTypes.ts, ready.test.ts, scores.test.ts, TypeSettings.tsx)
5. `S44c: no "Patient answered" line on a question the patient never saw` (shownToPatient, poa.ts + test, corrections.ts, ValidateSet.tsx)
6. `S44c: small fixes: TouchVersion on drafts only, clearer refusals` (questionnaires.sql, chapters.go, validation.go, their integration tests)
7. `S44c: a clearing save made by the form doesn't stop Start fresh (Rahul's answer 8)` (migration 00036, episodes.sql, sqlc output, validation.go, patient.go, answerSaver.ts + test, PatientHQ.tsx, TestStartFresh)
8. `S44c: docs`

## Test report 1 (5 passed, 1 failed, 15 blocked by the browser)

- **F1, the sample patient's Sex offered only Female and Male** (Preview and Test the whole HQ), so "Other makes STOP-Bang Incomplete" couldn't be tried before publishing: fixed. Both now use `SAMPLE_SEXES` (`features/preview/samplePatient.ts`), the patient record's own list (Female, Male, Other, Not known). Test first: `samplePatient.test.ts` failed (no module), then passed. Go already accepts any sex in a saved test case.
- **Cosmetic: "STOP-Bang score (not in t…" cut off** in the Low risk edit form's score box: the box is wider (300 px). JSX only, no automated test; T-10 step 1 checks it.
- **Exploring notes** (a BMI Calculator page is locked; titles commit on blur; a required follow-up shown again must be answered before Finish): not S44c's and expected; not changed.
- T-08 and T-11 now say where the score's value shows (the Preview's Clinician view; Test the whole HQ shows only the rules).
- Results after the fixes (28 Oct): `make lint test` exit 0 (Vitest 300 + 13 + 461, CDK 124, Go all ok); `make e2e` **98 passed, 1 failed, 5 skipped**. The failure was J7: the captured-messages page found 16 messages to its random made-up mobile instead of 1, because the local database already had messages to that number from earlier runs (the made-up range has only 1,000 numbers). That isn't S44c's. J7 passed when re-run alone, with J10 and J11 (19 passed).

## Test report 2 (28 Oct): all passed (16 of 16)

- **T-14 steps 2–3 (plan note):** the UI never lets an unreviewed episode at Ready for review go On hold, so "This episode isn't waiting for review any more." is reachable only through a race or an auto-triaged episode's API call. The tester confirmed it through the API. The test plan now marks it as covered by the integration tests (`TestCompleteReview`), and step 3 keeps the UI part ("already complete").
- **Exploring notes** (not changed; added to pending.md): the stale editor tab's "This Question Set has been deleted." banner stays until the next error (cosmetic); the POA Summary shows no STOP-Bang answers or score (older than S44c); General notes say "ASA grade set to 3" where the page says "ASA III" (S44b's wording); future-dated reminder notes on manual-test episodes after an e2e run moved the jobs clock (test data).

## Verification 1 (needs fixes: F1, F2; notes N1–N5)

- **F1, the patient's form cleared answers when a set opened** (the clearing was switched on before the answers loaded, so an answer an "isn't" condition hides once the answers are there was cleared on open, and saved by Sign out or the on-behalf Episode link): fixed. `clearHiddenAnswers` became **`loadForPatient(model, data)`**, which loads the answers first and then switches the clearing on; `SetForm` uses it, so the patient's form and staff's Fill in for the patient (the same component) both load first. Test first: `respondent.test.ts` "loading answers clears nothing, even under an "isn't" condition" (`{q_b} <> 'n'`, loaded with `{q_b: 'n', q_x: …}`: kept, no change fired, gone on Finish) failed with the old order, then passed; the chain test still passes with the new order.
- **F2, "STOP-Bang score (not in this HQ)" dropped out of the Rules page's "Score not in a band" list once another score was chosen:** fixed. New **`notBandChoices(scores, current)`** (`rules.ts`): the HQ's scores, then each ready-made score the HQ hasn't, always, then a chosen score that's neither. `RuleForm` uses it, and the checkbox is no longer disabled in an HQ with no banded score (STOP-Bang is always offered). Test first: `rules.test.ts` "notBandChoices" (3 cases) failed, then passed.
- **N1–N5:** added to pending.md ("From S44c's verification 1"), not changed.
- Results after the fixes (28 Oct): `make lint test` exit 0 (Vitest 300 + 13 + 465, CDK 124, Go all ok); `make test-integration` exit 0; `make e2e` **99 passed, 5 skipped** (8 min 03 s).

## Test report 3 (28 Oct): all passed (6 of 6)

- F1, F2 and the short regression (T-01 at phone size, T-07, T-10 steps 1–2) passed. Exploring notes: the greyed BMI and Ready-made score buttons on a page that already holds a question have no tooltip ("BMI goes on its own page" would help); added to pending.md. Not changed.

## Verification 2 (needs fixes: F3; note N6)

- **F3, editing a typed number (or text) wiped a follow-up's answer:** with `textUpdateMode: "onTyping"`, every key sets the value, so correcting 20 to 25 (Backspace, then 5) hid the follow-up for an instant and SurveyJS's immediate clearing wiped it. Fixed: **`loadForPatient(model, data)` now returns `{ atSave }`** and does the clearing itself (SurveyJS's `clearInvisibleValues` is no longer set; its default still clears hidden answers on Finish). It keeps the questions shown at the last check; after a **choice** it clears at once the answers to questions shown then and hidden now (repeated for a chain), so a chain never shows a follow-up through a hidden answer; after a **typed** change (text, number, comment) only at the autosave (`persist`, and the save when leaving the set), after the pause in typing. Loading still clears nothing (F1).
  - **Found while fixing:** the first version set a ref inside `useMemo`; React 19's development mode builds the memo twice, so the ref belonged to the discarded model and the save cleared nothing (J3b's new case caught it). The model and its clearing are now returned together from the memo.
  - Tests first: `respondent.test.ts` "a typed number edited 20 → 2 → 25 before the save keeps the follow-up; left at 2, the save clears it" (and the chain test now also checks a required follow-up isn't left showing after B No), red before the change. **J3b** has a second test, `test_editing_a_typed_number_keeps_the_follow_up_it_still_shows` (`e2e/data/hidden-number-hq.json`): at phone size, 20 → Backspace → 5 keeps "Tell us more" on screen and in the stored row; left at 2 it goes at the save.
- **N6** (other sets' clearing updates `lastSaved` before the saves return) and the note that the new defaults reach only new HQs: added to pending.md.
- Results after the fix (28 Oct): `make lint test` exit 0 (Vitest 300 + 13 + 466, CDK 124, Go all ok); `make test-integration` exit 0; J3b (both tests), J16 and J3 passed against `make dev` (5 passed). **The full `make e2e` couldn't finish: the Mac's disk filled up during the run** (`ENOSPC` creating Playwright's trace folders; 481 MB free of 460 GB, with 51 journeys erroring at setup and one failing at the same point). That's the environment, not S44c (decisions.md §4 has the Docker and Defender disk gotchas). Re-run `make e2e` once there's space.
- After the disk was freed (28 Oct): full `make e2e` against `make dev`: **99 passed, 1 failed, 5 skipped** (8 min 37 s). The failure is **J7** again, not S44c's. The captured-messages page found 2 messages for its random made-up mobile instead of 1, a number collision with earlier runs, as after test report 1. J7 then passed alone (6 passed). pending.md: `sj_before_s44a` is struck through (dropped by the conductor at Rahul's request).

## Test report 4 (28 Oct): all passed (7 of 7)

- F3 at phone size and through Fill in for the patient, the chain's immediate clearing, and the regression all passed. The exploring checks (leaving at once after a number change, emptying the box and retyping, a pre-filled follow-up, Validate's commit on blur, another patient's episode) all behaved correctly. A brief gap while the hidden questions close up is cosmetic; not changed.

## Verification 3 (needs fixes: F4, F5; Rahul's answer 8; note N7)

- **F4, a BMI's weight, or a number feeding a Calculation, still wiped a follow-up while being edited** (the change event named the `bmi` composite or the `expression`, neither "typed"): fixed by turning the test round. `loadForPatient` clears at once only after a **choice** (radio, checkbox, dropdown, tag box, image picker, boolean, rating, ranking); every other change waits for the autosave. It also remembers a question a deferred change shows (verification 3's N7, second way), so typing 20 (follow-up shows, is answered) then 2 before the pause still clears it at the save. Tests first (red, then green): `respondent.test.ts` "a BMI weight edited 120 → 12 → 125…", "a number feeding a Calculation, edited 42 → 4 → 45…", "a typed change that shows a question, answered and hidden again with no pause…". **J3b** third test, `test_editing_a_bmi_weight_keeps_the_follow_up_it_still_shows` (`e2e/data/hidden-bmi-hq.json`: BMI on page 1, its follow-up on page 2), at phone size.
- **F5, a `notBand` on a deleted author's score passed silently:** `ruleProblems` now reports "its score isn't in this HQ any more" unless the score is a ready-made one (STOP-Bang). Test first: `rules.test.ts` "a missing author's score is a publish problem; a missing STOP-Bang isn't".
- **Rahul's answer 8 (verification 3's N8): a clear made by the form mustn't stop Start fresh.** `clearHidden`'s save of another set is marked (`answerSaver.save(body, true)` → PUT `{…, cleared: true}`); Go adds 1 to the new **`episode_answers.cleared_saves`** (migration **00036**) when the save is so marked **and** its answers are contained in the old ones (`EXCLUDED.data <@ episode_answers.data`: only removals), and `StartFreshPrefill` allows `revision = 1 + cleared_saves`. A save marked so that changes an answer still counts as a change. Tests first: `TestStartFresh` "a clearing save by the form leaves Start fresh open; one that changes an answer doesn't" (red, then green); `answerSaver.test.ts` "passes on whether a save is the form's own clearing". **Needs `make migrate`.**
- **N7** (the leaving save doesn't clear other sets): pending.md.
- Results (28 Oct): `make lint test` exit 0 (Vitest 300 + 13 + 471, CDK 124, Go ok); `make test-integration` exit 0; `make e2e` **100 passed, 1 failed, 5 skipped** (9 min 06 s). The failure was J7's made-up mobile collision again ("count 1, got 3"; pending.md), which passed alone (6 passed). Disk: 13 GB free after the run.

## Test report 5 (28 Oct): all passed (8 of 8)

- F4 (a BMI's weight; a number feeding a score), the chain, F5, answer 8's set-up as the plan allowed, and the regression passed. Exploring: the "show then hide within one pause" case cleared at the save as intended; a pre-filled follow-up survived a number edit. One wording note: with an author's score deleted, the Publish window's auto-triage line leaves out the rule's missing-score test, but Publish is refused anyway; added to pending.md. Not changed.
