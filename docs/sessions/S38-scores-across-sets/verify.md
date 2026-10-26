# S38 verification

## Verification 3 (26 Oct 2026, automatic run; the extra round Rahul allowed)

**Verdict: Verified.**

F4 is fixed. `hqSets` (`packages/clinical/src/respondent.ts`) does one pass. For an HQ whose scores use only their own set, the saved answers decide, as before (no survey built). Otherwise each shown set's survey is built with the sets before it, and the next set's condition sees each Calculation's worked-out value in place of the saved one. `shownSets`, `earlierContexts`, the POA Summary and `PatientHQ` all use it. The tests cover a set wrongly shown, one wrongly hidden, a set never saved, Send held back, and older HQs unchanged (`scores.test.ts`), plus J10's new journey. Verification 2's minor items are fixed or in pending.md.

The three open questions in pending.md's S38 section are Rahul's to decide; they don't block this verdict. These are a hidden score input counting as missing, the clinician's correction bringing back a hidden set, and a condition on a clinician-only score.

### Findings (none blocking; all added to pending.md under "Verification 3")

1. **Validate's "Patient answered: …" under a patient-visible cross-set score shows the stale saved value.** `apps/web/src/features/episodes/corrections.ts:12` compares the Calculation's value with the patient's saved one.
   - Reproduced in Chrome, Follow-up HQ:
     - Snoring No; Tiredness saved (its score 0);
     - Snoring changed to Yes, then sent.
   - Validate Tiredness shows "Snoring score" **1**, which is right, with "Patient answered: 0".
   - It's misleading, not wrong: the score shown is right, but the note suggests a correction that nobody made.
2. **A Question Set condition on a clinician-only cross-set score.** This is the open question already in pending.md. A probe showed the clinician side differs from the patient's:
   - `shownSets` with viewer `patient` gives sets 1 and 2;
   - with viewer `clinician` it gives 1, 2 and 3.
   - So the episode page lists a set the patient never saw, as Not started.
   - Builder's `scores.test.ts:356` uses a clinician-only score with the clinician viewer, so it doesn't show this.
   - A Publish warning would close it.
3. **`scoredQuestions` (`packages/clinical/src/scores.ts:48`) lists scored questions inside a repeating group,** as the picker did before S38. A score using one is now Incomplete for every patient; before S38 it silently added 0.
4. **Deleting a page or a Question Set doesn't warn** about a later set's score that uses its questions (`QuestionnaireEditor.tsx:356`). Deleting one element does warn, and Publish catches it.
5. **Smaller items:**
   - STOP-Bang gives sex "Other" a definite band (decision 4 in the build notes; Rahul may prefer Incomplete).
   - The patient's form uses today's age, while staff views use the age on the day the answers were sent.
   - `EpisodePage.tsx:260` builds the surveys on every render.
   - `PatientHQ`'s cache key stringifies all the answers on each render.
   - `ready.ts` has its own `canonical`, and `earlierPoints` repeats `score()`'s loop.
   - The code review also repeated items already in pending.md: copying a page holding STOP-Bang, the Preview's `givenEarlier`, and the global SurveyJS setting.

### Checks

- **Scope:**
  - The F4 fix and verification 2's minor fixes add nothing beyond them, apart from J10's new journey and `e2e/data/score-condition-hq.json`.
  - Go is unchanged.
- **Tests first:** F4 has Vitest cases (both directions, Send, older HQs) and a journey. There are no new routes.
- **Tenant isolation and security:** no Go change, and `chapter.ForPatient` is unchanged.
- **A-20:** the grep finds only comments (`format.ts`, `patient.go`), as before.
- **Docs:**
  - plan C6's status, plans/README, sessions/README (now Verified), decisions.md and pending.md are current;
  - `repocheck` passes.

### Test results

- `make lint`: clean.
- `make test`: Go ok; `packages/clinical` **355 passed**; `apps/web` **270 passed**; CDK **124 passed**.
- `make test-integration`: every package ok.
- `make e2e` (against `make dev`): **88 passed, 5 skipped** (Cognito-only), with J10's S38 journeys included.
- **Chrome (Cara Clinician, Hospital A1), using made-up patient "Sam VThree S38v" on "MT S38 1605 Follow-up HQ v1", through Fill in for the patient:**
  - Snoring No, then Tiredness Yes (its saved score is 0).
  - I then changed Snoring to Yes without opening Tiredness. **Sleep follow-up appeared** ("2 of 3"), and Send stayed disabled until it was answered.
  - After sending, the episode lists all three sets. Validate Tiredness shows the score **1**, with finding 1's note.
  - No console errors.
- **Probe:** a throwaway Vitest file, deleted, confirmed finding 2.
- **Code review (high):** 10 findings, weighed above. None blocks.

### Hand checks for Rahul

- T-14 at a true 390 × 844 (unchanged).

### Commits

The scores and STOP-Bang code share files: `publish.ts`, `library.ts`, `index.ts`, `QuestionnaireEditor.tsx`, `ElementCard.tsx` and `SettingsPanel.tsx`. A separate STOP-Bang commit would leave the first one unable to build, so the code goes in one commit:
1. `S38: scores across Question Sets, points lines, Incomplete, ready-made STOP-Bang` (`packages/clinical`, `apps/web`, with their tests).
2. `S38: J10 extended` (`e2e/journeys/test_j10_authoring.py`, `e2e/data/{scores-hq,score-condition-hq}.json`, `e2e/README.md`).
3. `S38: docs`: this folder, `CLAUDE.md`, `apps/web/AGENTS.md`, `docs/decisions.md` (with §5's "cloud" → "Claude"), `docs/pending.md`, and the plan and sessions status. It leaves out `docs/market/`.

## Verification 2 (26 Oct 2026, automatic run)

**Verdict: Needs fixes.**

Verification 1's three fixes hold:
- **F1:** I checked it myself in Chrome. With an empty line open, I removed "Sex is Male" from "Sleep risk". The server's expression lost the line (revision 1 → 2), and the card said "plus 2 points lines".
- **F2:** the score with a key repeated in two sets opens without a crash.
- **F3:** `alreadyHere(stopBangGroup(bmi))` is in place.

Every automated test passes.

One new problem stops it (F4), and it's in what this session adds. The saved value of a score that uses earlier sets goes out of date when an earlier set is changed. Question Set conditions read that saved value, so a later set can be shown or hidden wrongly.

### Findings (worst first)

**F4. A Question Set condition on a cross-set score reads the score's saved value, which goes out of date when an earlier set is changed.** The main problem is `packages/clinical/src/respondent.ts:59` (`shownSets`). The same happens in `earlierContexts` (`respondent.ts:86`), `poa.ts:66` and `:144`, `PatientHQ.tsx:232` and `EpisodePage.tsx:260`.
- **How it happens:**
  - SurveyJS saves a Calculation's value in its set's answers. Before S38, a score used only its own set, so the saved value always matched that set's answers.
  - Now set 2's score can use set 1. If the patient answers set 2 and then goes back to change set 1, set 2's saved score is never worked out again: no view saves set 2 unless it's opened and changed.
  - `shownSets` decides which later sets are shown from the raw saved answers (`values = { ...values, ...answers[s.id] }`). The condition builder offers an earlier Calculation's value and band, so an author can write a condition on it.
- **Reproduced (Vitest probe, since deleted):** the sets were:
  - set 1: `q_a`, Yes 1 / No 0;
  - set 2: score `s` from `q_a`;
  - set 3: condition `{s} >= 1`.

  Saved answers: set 1 `{q_a: "no"}` (changed after set 2 was saved), set 2 `{s: 1}`. `shownSets` gives **1,2,3**, though `earlierContexts` works the score out as `s: 0`. Every view uses `shownSets`: the patient's form and their "Send my answers" check, the episode page, and the POA Summary.
- **Why it matters:**
  - Example: "show the sleep follow-up set when Sleep risk is High". A patient who changes an earlier answer that raises the score never gets the follow-up set, and can send without it.
  - The other way round, the patient is asked a set that no longer applies.
  - Either way, the clinician's screens follow the stale value too, while Validate shows the correct score. Before S38 this couldn't happen.
- **Fix (builder to choose the exact shape):** decide which sets are shown from worked-out values, not saved ones.
  - Have `shownSets` walk the sets the way `earlierContexts` does: build each shown set's model with its earlier context, and carry its worked-out Calculation values forward. Or merge the two into one pass that returns both.
  - Keep the cheap path when no score uses another set, so older HQs behave exactly as before.
  - **Tests:** the probe above, in both directions (a set wrongly shown, and one wrongly hidden), and `isSetReadyToSend` / "Send my answers" for the hidden case.

**Minor (not blocking; add to pending.md, or fix along with F4 if cheap):**
- `apps/web/src/features/patient/prefill.ts:69` and its helpers (`awaitsCheck`, `shownPrefilled`, `firstUnfilled`) build the set's model with no `earlier`, unlike `SetForm` and `isSetReadyToSend`.
  - Suppose a question in set 2 shows only when a cross-set score's band is High. These helpers then treat it as hidden, so "Please check" and "Everything is still correct" can skip a pre-filled question the patient sees.
  - Passing the same `earlierContexts` entry fixes it.
- `packages/clinical/src/ready.ts:165`: `readyScoreProblems` compares against `stopBangGroup(readyScoreBmi)`, but `renameIn` (`library.ts:122`) rewrites the BMI key in the expression and the band's `when`, not `readyScoreBmi`. A probe confirmed it: after a rename, Publish says "STOP-Bang has been changed…".
  - The editor can't reach it today: Same as… isn't offered for a BMI question or a Calculation (`canAskFresh`, `keys.ts:13`).
  - It would matter if a rename path ever reaches them. Rewriting `readyScoreBmi` in `renameIn` is a one-line fix.
- `packages/clinical/src/scores.ts:136` (`applyEarlier`): `getAllQuestions(false, false, true)` includes a composite's inner questions.
  - So a BMI question in this set hides an earlier set's library `height` or `weight` answer of the same name. A points line on that answer then stays Incomplete.
  - This is an edge case (a line on the raw height or weight).
- `apps/web/src/features/editor/QuestionnaireEditor.tsx:273`: deleting a Group (for example STOP-Bang) warns only about other sets that use the Group's own name, not its questions' keys. Publish catches it.
- `earlierContexts` builds a model for every shown set, even when Validate needs one, and `poaSummary` builds each set twice. That's fine at today's sizes.
- Verification 1's minor items not yet done are still in pending.md (the card's "not found", `hqPatient`'s time zone, the Preview's earlier Calculations, the global SurveyJS setting). `QuestionnaireEditor.tsx:253` still has a needless `structuredClone`.

### Checks

- **Scope:** as in verification 1. The fixes add nothing beyond F1–F3, apart from J10's repeated-key journey.
- **Tests first:**
  - F1: `scores.test.ts` "saving points lines", plus J10.
  - F2: `scores.test.ts` "a key repeated in two Question Sets", plus J10 `test_a_repeated_key_does_not_break_the_score_editor`.
  - There are no new routes.
- **Tenant isolation and security:** no Go change, and `chapter.ForPatient` is unchanged.
- **A-20:** the grep finds only comments (`format.ts`, `patient.go`), as before.
- **Docs:** the session row, plan C6's status, decisions.md and pending.md are current. `repocheck` passes.

### Test results

- `make lint`: clean.
- `make test`: Go ok; `packages/clinical` **349 passed**; `apps/web` **270 passed**; CDK **124 passed**.
- `make test-integration`: every package ok.
- `make e2e`: not re-run by me. The builder's run after the fixes gave 86 passed and 5 skipped, and the repeated-key journey passed on its own run.
- **Chrome (Alex Author, "MT S38 1500 Scores HQ" v2 draft):** F1 and F2 checked as above, with no console errors. The "Sex is Male" line I removed stays removed in that draft (test data).
- **Probes:** two throwaway Vitest files, deleted. They confirmed F4 and the `readyScoreBmi` rename.

### Hand checks for Rahul

- T-14 at a true 390 × 844 (unchanged from verification 1).

### Suggested commits (once fixed)

As in verification 1, with F4's fix and tests in commit 1.

Verification found problems: read `docs/sessions/S38-scores-across-sets/verify.md`, fix them test-first, then ask me for a retest.


## Verification 1 (26 Oct 2026, automatic run)

**Verdict: Needs fixes.**

The session does what the brief and Rahul's answers ask: scores across earlier Question Sets, "Add N points when…" lines from menus, Incomplete for a missing input, one helper (`earlierContexts`) used by the patient's form, Validate and the POA Summary, the Preview's earlier-answers box, and a locked STOP-Bang (Rahul chose it for now in answers.md). Go is unchanged and every automated test passes. Two editor bugs stop it, both reproduced in Chrome. F1 is the important one: an author can remove a points line, see it gone and "All changes saved", and still publish a score that adds it.

### Findings (worst first)

**F1. A removed or changed points line isn't saved while another line is unfinished, and the editor says "All changes saved".** `apps/web/src/features/editor/TypeSettings.tsx:354` (`PointsLines.update`).
- `update` calls `onChange` only when *every* draft line is complete (`next.every(isCompleteLine)`). If any line is unfinished, a change to the saved lines is kept only in the component's state.
- **Reproduced (Chrome, Alex Author):** a score with one saved line "1 point when Age > 50". I pressed **Add points line** (left empty) and deleted the Age line. The form showed only the empty line, and the header said "All changes saved". The card still said "plus 1 points line". The chapter on the server still had `iif({patientAge} > 50, 1, 0)` (revision unchanged).
- **Why it matters:** this is a clinical score. Publishing now gives patients over 50 a point the author removed, and the band and its disclosure can change, with nothing on screen to say so. Editing a saved line's points or value has the same problem while another line is unfinished.
- **Fix:** save the complete lines whenever they differ from what's saved, e.g. `const complete = next.filter(isCompleteLine); if (JSON.stringify(complete) !== key) onChange(complete);`. `formatScore` already drops unfinished lines, so the guard isn't needed. Add a test: a saved line, an unfinished one added, the saved one removed → saved without it.

**F2. Selecting a score crashes the editor when a draft repeats a key in two Question Sets.** `apps/web/src/features/editor/TypeSettings.tsx:258` (`scored`), and `pointsSubjects` (`packages/clinical/src/logic.ts:354`) for the lines' menu.
- The options merge earlier sets' questions with this set's without removing repeated keys. Mantine throws on a repeated option value ("Duplicate options are not supported").
- **Reproduced (Chrome):** set 1 and set 2 each with a scored `q_vfcollar`, and a score in set 2. Clicking the score's card replaced the editor with the "Something went wrong" page (console: `Option with value "q_vfcollar" was provided more than once`). Before S38 this click worked.
- Go lets a draft hold a repeated key (`keys.Duplicates` is a problem list at publish, not a save refusal), and `applyEarlier`'s own comment (`scores.ts:101`) expects older HQs to have them. So an author can hit this.
- **Fix:** list each key once in both menus (this set's wins, matching `applyEarlier`), with a unit test on `pointsSubjects` and the score picker's options.

**F3 (minor). Inserting STOP-Bang checks only for an element named `stopbang`.** `apps/web/src/features/editor/QuestionnaireEditor.tsx:253`.
- It doesn't check the Group's fixed child keys (`stopbang_snoring` … `stopbang_score`). A question that adopted one of them with **Same as…** from another HQ would end up repeated, which only shows at publish.
- **Fix:** use the existing `alreadyHere(stopBangGroup(bmi))` (`takenNames` over the whole Group), which also removes the needless `structuredClone`.

**Minor, for pending.md (not blocking):**
- `apps/web/src/features/patient/PatientHQ.tsx:231`: `earlierContexts` runs on every render, building and disposing a SurveyJS model per shown set, including on each keystroke while a set is open. Memoise it on the HQ and the saved answers.
- `apps/web/src/features/episodes/hq.ts:45` (`hqPatient`): the age on the day of sending is worked out in the browser's time zone, not the trust's. Near midnight on a birthday, a clinician in another time zone could see a different age, and so an "over 50" point the patient's screen didn't have. Before S38 every age was already the browser's "today", so this isn't new.
- `packages/clinical/src/questionSets.ts` `givenEarlier`: the Preview and test cases don't work out an earlier set's Calculations from the given answers. A later set's `band('<earlier calc>')` is empty unless the author types the Calculation's value, while the real HQ computes it. This is documented as a simplification for questions hidden inside an earlier set. Add this case to the same note.
- `packages/clinical/src/scores.ts:127` repeats `respondent.ts:103`'s `isEmpty`. Share one.
- `apps/web/src/features/editor/ElementCard.tsx:268`: the card counts a deleted question as "in earlier Question Sets". Publish catches it, but the card could say "not found".
- `packages/clinical/src/questionTypes.ts`: `settings.expressionQuestionTrackDependencies = false` is global, so every Calculation re-runs on every change in every survey. It's recorded in decisions.md as a gotcha, and is fine at today's sizes. Revisit if large HQs feel slow.
- Open for Rahul (already in pending.md): a question a condition hides on purpose makes its score Incomplete.

### Checks

- **Scope:** everything in the brief, with Rahul's answers (menu lines, Incomplete, STOP-Bang now). STOP-Bang is extra to the brief but was asked for in answers.md. The UHN licence is recorded for C10 in decisions.md §5.
- **Tests first:** new logic has Vitest (`scores.test.ts`, `ready.test.ts`, `hq.test.ts`, `fields.test.ts`), and J10 has two new journeys. No new routes, so no new integration cases are needed.
- **Tenant isolation:** no Go change. The Preview fetches earlier sets through the existing hospital-scoped chapter routes.
- **A-20:** the grep finds only comments in `format.ts`/`patient.go` and the infra tests' UK lists, as before.
- **Security:** no secrets, and no patient data in logs or messages. `chapter.ForPatient` is unchanged. STOP-Bang's score is clinician-only.
- **Docs:** decisions.md (S38 entry, and §5's "cloud" → "Claude"), pending.md, plan C6's status, plans/README and sessions/README are updated. `repocheck` passes.

### Test results

- `make lint`: clean.
- `make test`: Go ok; `packages/clinical` **345 passed**; `apps/web` **270 passed**; CDK **124 passed**.
- `make test-integration`: every package ok.
- `make e2e`: not re-run by me. The builder's run gave J10 5 passed after the last fix, and test report 3 all passed.
- **Chrome spot checks:** F1 and F2 reproduced as above. The probe questionnaire was deleted afterwards.

### Hand checks for Rahul

- T-14 at a true 390 × 844 (the patient's sets at phone size). The tester's Chrome can't go below 500 px, and J10's patient runs at iPhone size in WebKit.

### Suggested commits (once fixed)

1. `S38: scores across Question Sets, points lines, Incomplete`: the files in build notes' commit 1, plus the F1/F2 fixes and tests.
2. `S38: ready-made STOP-Bang`: `packages/clinical/src/{ready,ready.test}.ts`, `apps/web/src/features/editor/{ReadyScoreWindow,AddContentBar}.tsx`, `context.ts`, `QuestionnaireEditor.tsx`'s insert (F3).
3. `S38: J10 extended`: `e2e/journeys/test_j10_authoring.py`, `e2e/data/scores-hq.json`, `e2e/README.md`.
4. `S38: docs`: this folder, `CLAUDE.md`, `apps/web/AGENTS.md`, `docs/decisions.md`, `docs/pending.md`, plan and sessions status. Leave out `docs/market/`.

Verification found problems: read `docs/sessions/S38-scores-across-sets/verify.md`, fix them test-first, then ask me for a retest.
