# S44c verification

## Verification 4 (28 Oct 2026, automatic run; the extra round Rahul allowed after verification 3)

**Verdict: Verified.** Verification 3's F4 and F5 are fixed, and Rahul's answer 8 is built (migration 00036) as he asked. Their tests are there and pass. I re-checked F4 myself in the browser through Fill in for the patient. The code review raised nothing new that's a defect against the brief or answers.md; its new notes (N9, N10) are minor and in pending.md.

### Verification 3 and answer 8: checked

- **F4 holds.** `loadForPatient` (`packages/clinical/src/respondent.ts`, `CHOICES` / `isChoice`) now clears at once only after a choice (radio, checkbox, dropdown, tag box, image picker, boolean, rating, ranking). Every other change, including the BMI composite and an `expression`, waits for the save (`atSave`), and a question such a change shows is added to `shown`. So N7's second way is fixed too.
  - The unit tests are in `respondent.test.ts`: BMI 120 → 12 → 125, a number feeding a Calculation 42 → 4 → 45, and "shown then hidden with no pause". The chain test still passes.
  - J3b's third test edits a BMI weight at phone size.
- **F5 holds.** `ruleProblems` (`rules.ts`) reports a `notBand` on a missing score unless it's a ready-made one (`READY_NAMES`), and `matches` is unchanged. It's tested in `rules.test.ts`: "a missing author's score is a publish problem; a missing STOP-Bang isn't". Test report 5 saw Publish refused, with a 422.
- **Answer 8 holds.** `clearHidden` marks its saves (`answerSaver.save(body, true)` → `cleared: true`). `SavePatientAnswers` adds to `cleared_saves` only when the new data is contained in the old (`<@`), and `StartFreshPrefill` allows `revision = 1 + cleared_saves`.
  - `TestStartFresh` has both cases: removing an answer keeps Start fresh (204), and changing one stops it (409).
  - Migration 00036 has an Up and a Down.
  - On `started` (raised by the code review): a clearing save still moves the revision, so the row's `started` turns true. But `started` is read per form (any set started), and a clearing save always follows the patient's own save in another set, so nothing changes. decisions.md says so.

### Findings

**None blocking.**

The code review's other points are either known or minor:
- **Already in pending.md:** N2 (a content save racing a delete), N5 (the visibility walk written three times, `clearedAnswers`' cost, `samplePatient.ts`), N6 (`lastSaved` set before the clearing saves return, and other sets' drafts) and N7 (the unmount save doesn't call `onSaved`).
- **The list rows' `triageMessagePending`** isn't masked as the episode page's is, but only `EpisodePage.tsx:295` reads it, from `getEpisode`, so nothing shows wrongly.
- **Added to pending.md ("From S44c's verification 4"):**
  - **N9:** `writeChapterGone` calls a database error "deleted" (409).
  - **N10:** a hand-made `cleared` save that unticks one checkbox box passes `<@`. It only lets the patient start their own set fresh.
  - The `started` note above.

### Test results (28 Oct)

- `make lint test test-integration`: **exit 0**.
  - Vitest: 300 + 13 + 471 passed.
  - CDK: 124 passed.
  - Go unit and integration: every package ok.
- `make e2e` against the conductor's `make dev`: **101 passed, 5 skipped** in 8 min 39 s.
  - J3b's three tests passed, and J7 passed this time.
  - The 5 skipped are the AWS-only journeys.
- A-20 grep over the changed and new files: only made-up `+44` mobiles in `triage_integration_test.go`, and docs. `repocheck` passes (in `make test`).

### Spot-checks

1. **F4 through Fill in for the patient** (Claude in Chrome, Browser 1, as Cara; a new episode for a made-up patient on "MT S44c weight 1935"). Test report 5 tried this only as the patient.
   - Height 170 and weight 120 gave a BMI of 41.5. On the next page I typed "Verifier diet note" in the follow-up, and it was stored at revision 2.
   - Previous, then the weight selected and typed over as 130. While I typed, it read 1 and then 13, which hide the follow-up. The stored row became `{bmi 45, weight 130, follow-up "Verifier diet note"}` at revision 3: **kept**.
   - Weight 70 (BMI 24.2): after the pause, the stored row was `{bmi 24.2, weight 70}` at revision 4. The follow-up was **cleared**, and the button read "Finish this section". **Pass.**
2. **Permissions on the on-behalf save with `cleared: true`:**
   - No sign-in: 401.
   - Bea (Trust B) on Trust A's path: 404 "Hospital not found."
   - Val (Hospital A2) on Hospital A1's path: 404.
   - The row was unchanged (revision 4). **Pass.**
3. **The patient router can't reach staff routes:** `/api/p/o/…` is 404. **Pass.**
4. **F5 and answer 8 in the code and tests,** as above. Test report 5 covered both in Chrome, including the server's 422 and the Start fresh 204 after a clearing save.

### Hand checks for Rahul

- None.

### Commits (made locally; not pushed)

The changes were committed by file, one commit per brief part, with that part's tests. A file holding two parts goes with the earlier commit, so each commit builds:
1. Go: the held message, the four small fixes, and answer 8's clearing-save count. This is the SQL, the sqlc output, migration 00036, the handlers and the integration tests.
2. `notBand` and the defaults.
3. STOP-Bang Incomplete for sex Other, with the sample patient's Sex options.
4. Clearing on hide in the patient's form, with answer 8's marked save, J3b and J16.
5. No "Patient answered" line.
6. The docs.

`docs/market/` and `docs/sessions/run-state.md` are left out.

## Verification 3 (28 Oct 2026, automatic run; the extra round Rahul allowed)

**Verdict: Needs fixes.** Verification 2's F3 is fixed for a plain typed number or text box, and test report 4 and J3b show it. But the fix decides "typed or not" by the changed question's type (`text` or `comment` only), so two common ways a typed number reaches a condition still wipe a follow-up while the patient is editing (F4). I confirmed it in a real browser at phone size. There's also one small safety gap in the new `notBand` test (F5). Both fixes are small. Everything else from verifications 1 and 2 holds.

### Verifications 1 and 2: checked

- **F1 holds.** `loadForPatient` (`packages/clinical/src/respondent.ts:258`) loads the answers first and only clears answers to questions shown at the last check. The "isn't" test (`respondent.test.ts:170`) passes, and test report 4 saw no save when a finished set was opened.
- **F2 holds.** `notBandChoices` (`rules.ts:193`) is unchanged since verification 2.
- **F3 holds for a plain number or text box.** A change to a `text` or `comment` question now clears hidden answers only at the save (`persist` → `atSave`, `PatientHQ.tsx:594`; the unmount save, `:665`). The unit test (`respondent.test.ts:154`), J3b's second test and test report 4 (patient at phone size, and Fill in for the patient) cover it. F4 below is the same defect by two other routes.
- **N1–N6 are in pending.md,** as agreed.

### Findings, worst first

**Must fix**

- **F4. A typed number still wipes a follow-up while it's being edited, when it's a BMI height or weight, or when it feeds a Calculation** (`packages/clinical/src/respondent.ts:242`, `isTyped`, used at `:275`).
  - **What happens.** `isTyped` treats only a change to a `text` or `comment` question as typed. Every other change runs `clear()` at once. Two common cases get past it:
    1. **BMI.** The BMI Calculator is a composite question (`type: "bmi"`, `questionTypes.ts:52`). Its height and weight boxes are inner text boxes, and under `textUpdateMode: "onTyping"` each key changes the composite's value. The change event names the `bmi` question, so `clear()` runs on every key. The condition builder offers "BMI" as a subject (`logic.ts:294`), so `{q_bmi.bmi} > 35` is an ordinary follow-up condition.
    2. **A number that feeds a Calculation** (a score, such as STOP-Bang's neck size, or an author's own score). Typing re-computes the expression question, and SurveyJS fires `onValueChanged` for it with type `expression`, which isn't "typed". So `clear()` runs mid-edit.
  - **Confirmed:**
    - **In a real browser at phone size:** a temporary Playwright journey against `make dev`, deleted afterwards. BMI on page 1, and "Tell us about your weight" on page 2 shown when BMI > 35.
      - The answers were height 170, weight 120 and "Diet plan".
      - Then Previous, and weight 120 → Backspace → 5.
      - The stored row went from `{q_v3_bmi: {height: 170, weight: 120, bmi: 41.5}, q_v3_why: "Diet plan"}` to `{q_v3_bmi: {height: 170, weight: 125, bmi: 43.3}}`, and the follow-up box was empty.
    - **In survey-core** (a temporary Vitest file, deleted afterwards):
      - BMI weight 120 → 12 clears the follow-up at once.
      - A number feeding `iif({q_neck} > 40, 1, 0)`, with a follow-up shown when `{q_neck} > 40`: 42 → 4 → 45 loses the follow-up's answer. At 45 the expression's change event arrives before the follow-up is shown again.
  - **Why it matters.** It's the same harm as F3: an ordinary correction loses a disclosure, and nothing stops the send if the follow-up is optional. Weight and BMI are among the most common clinical thresholds.
  - **Fix (builder's choice; recommended first):**
    - **Recommended: turn the test round.** Clear at once only after a change to a choice question: SurveyJS's `selectbase` family (radio, checkbox, dropdown, tag box, image picker) plus `boolean` and `rating`. Every other change waits for the save: text, comment, number, the BMI composite, an expression, matrices and dynamic panels. A choice is what chains need at once, and a deferred change still clears at the save, chains included.
    - **Or:** keep the list of typed types, and also treat as typed a composite whose inner question is a text box, and every `expression` change. This is easier to get wrong.
  - **Tests first:**
    - Unit tests in `respondent.test.ts`, next to the 20 → 2 → 25 one:
      - BMI weight 120 → 12 → 125 keeps a `{q_bmi.bmi} > 35` follow-up, and left at 12, the save clears it;
      - a number feeding an expression, 42 → 4 → 45, keeps its follow-up.
    - J3b (or its number test's data) with a BMI follow-up, edited with Backspace then a digit at phone size.
    - The chain test must still pass.

- **F5. A `notBand` test on an author's own score that's no longer in the HQ always passes, and Publish says nothing** (`packages/clinical/src/rules.ts:387-390`, with `matches` at `:117-120`).
  - **What happens.** "Passes when the HQ hasn't the score" is right for the ready-made STOP-Bang (the defaults must work in an HQ without it). But it applies to every score.
    - An author writes "ASA ≤ 2, no red flags, and Frailty isn't High → Ready for admission". Later they delete or re-key the Frailty Calculation.
    - The rule then auto-triages every patient it would have held back. `ruleProblems` reports nothing, so Publish goes through.
    - The Rules page only adds "(not in this HQ)" after the score's key.
    - The same rule written with `band` is refused ("its score isn't in this HQ any more").
  - **Why it matters.** The rule silently drops a safety exclusion an author added on purpose.
  - **Fix:** in `ruleProblems`, report a `notBand` score that's neither in the HQ nor a ready-made score (`READY_NAMES`) with the same wording as `band`'s: "its score isn't in this HQ any more". Leave `matches` as it is. Go's shape check needn't change.
  - **Tests first:** in `rules.test.ts`, a `notBand` on a missing author's score is a problem, and on a missing STOP-Bang it isn't (the defaults' case, already tested).

**Notes (no change needed now; add to pending.md if the builder agrees)**

- **N7. Two ways a hidden answer is left saved until Finish or the send.** In each, the screen agrees with the model, and the S44a backstop drops the answer at send.
  - **The unmount save doesn't call `onSaved`** (`PatientHQ.tsx:665`). After a change in one set, leaving by an in-app link within the 0.6 s pause doesn't clear the answers that change hid in other sets. The next `clearHidden` then moves `lastSaved` past it.
  - **A typed change that shows a question doesn't add it to the "shown at the last check" list** (`loadForPatient`'s `shown` is updated only in `clear()`). Say the patient types 20 (the follow-up shows), types its answer, then sets units back to 2, with no 0.6 s pause anywhere. The save then doesn't see the follow-up as hidden by this change. The fix is to add newly shown questions to `shown` on every change. It would fit with F4's fix.
- **N8. `clearHidden` can save another set that was pre-filled and never opened** (`PatientHQ.tsx:254`). This happens when a choice in one set hides a pre-filled, shown answer in another. That set's revision then goes past 1, so it counts as started, and its Start fresh is refused (`episodes.sql:384`, `revision = 1`).
  - This is the cleared answer as Rahul's answer 1 asks. But a set the patient never opened becomes "started", and Start fresh only works on another trust's answers. Worth a line in pending.md, with the question for Rahul: should a clearing save keep the set's revision for Start fresh?
- **The code review raised these again:** N1 (Checked while On hold drops a held message), N2 (a content save racing a delete), N3 (`notBand` passes when the score is Incomplete) and N5 (`clearedAnswers`' cost on each save, and the visibility walk written three times). All are already in pending.md.

### Test results (28 Oct)

- `make lint test test-integration`: **exit 0**.
  - Vitest: 300 + 13 + 466 passed.
  - CDK: 124 passed.
  - Go unit: all ok.
  - Integration: every package ok.
- `make e2e` against the conductor's `make dev`: **100 passed, 5 skipped** in 8 min 13 s. J3b passes both tests, but neither has a BMI or a Calculation, so they don't cover F4. The 5 skipped are the AWS-only journeys.
- A-20 grep over the changed files: as in verification 2 (test files only). `repocheck` passes (in `make test`).

### Spot-checks

1. **F4 in a real browser** (Playwright, Chromium at phone size, against `make dev`): reproduced, as above.
2. **F3's plain-number case** is covered by J3b's second test (in the e2e run) and test report 4's Chrome checks. I re-read the code path: `persist` → `atSave`, and the unmount save.
3. **The patient router can't reach staff routes:** `/api/p/o/…` is 404.
4. **F5** was confirmed from the code. Both `matches` and `ruleProblems` skip a missing score, and `whenWords` (`rules.ts:229-232`) shows the key with "(not in this HQ)".

### Suggested commits (after F4 and F5): verification 1's list stands.

### For the builder

"Verification found problems: read `docs/sessions/S44c-c6-last-answers/verify.md`, fix them test-first, then ask me for a retest."

## Verification 2 (28 Oct 2026, automatic run)

**Verdict: Needs fixes.** Verification 1's F1 and F2 are fixed and hold: the code, their unit tests and test report 3 agree. But the clearing that F1's fix kept has one more case that loses answers (F3), found by `/code-review high` and confirmed in a real browser. Everything else stands as in verification 1.

### Verification 1's findings: checked

- **F1 is fixed.** `loadForPatient` (`packages/clinical/src/respondent.ts:245`) loads the answers first, then sets `clearInvisibleValues = "onHiddenContainer"`. `SetForm` uses it (`apps/web/src/features/patient/PatientHQ.tsx:533`), so staff's Fill in for the patient does too. The new test "loading answers clears nothing, even under an "isn't" condition" (`respondent.test.ts:155`) covers the case I gave. Test report 3 confirmed it in the browser.
- **F2 is fixed.** `notBandChoices` (`packages/clinical/src/rules.ts:193`) always offers the ready-made STOP-Bang when the HQ hasn't it. `RuleForm.tsx:68` uses it, and three unit tests cover it.
- **N1–N5 are in pending.md** ("From S44c's verification 1"), as agreed.

### Findings, worst first

**Must fix**

- **F3. Editing a typed number (or text) can wipe a follow-up's answer, on screen and on the server** (`packages/clinical/src/respondent.ts:247` with `PatientHQ.tsx`'s `m.textUpdateMode = "onTyping"`).
  - **What happens.** Under `onTyping`, survey-core sets a text or number box's value on every key-up (`QuestionTextModel.onKeyUp` → `updateValueOnEvent`; `isTextValue` covers number). So the half-typed value counts, and "onHiddenContainer" clears a follow-up its condition hides for that instant. The follow-up comes back empty with the next key.
  - **Example.**
    1. The patient answers "Units a week" 20, and "Tell us more" (shown when `{units} > 14`) "Wine with dinner".
    2. They correct 20 to 25: Backspace (the value is 2, so the follow-up hides and is cleared), then 5.
    3. The follow-up shows again, empty, and the autosave stores `{units: 25}`.
  - **Confirmed twice:**
    - In survey-core directly: a script, with `onValueChanged` firing `why → null`.
    - In a real browser at phone size: a temporary Playwright journey against `make dev`, deleted afterwards.
      - The stored row went from `{q_zz_why: "Wine with dinner", q_zz_units: 20}` to `{q_zz_units: 25}`.
      - The box was empty.
  - **Why it matters.**
    - A patient's disclosure is lost by an ordinary correction.
    - If the follow-up is optional, nothing stops the send. A required one is caught only at Finish, and the patient must type it again.
    - Number thresholds on a typed answer are a normal clinical pattern: units of alcohol, cigarettes a day, weight.
    - This came in with S44c: before it, the answer stayed and only the send dropped hidden answers.
  - **Fix (builder's choice; Rahul's "as soon as it's hidden" still holds if it's done at the autosave's pause).**
    - **Recommended:** clear the open set's hidden answers when the autosave runs (after the pause in typing), not on each key.
      - Use the same before/after comparison `clearedAnswers` uses for other sets: an answer whose question showed at the last save and doesn't now.
      - Put it in the model, then save.
      - Keep "an answer hidden when the set opens stays until Finish".
    - **Or:** keep SurveyJS's immediate clearing only for choice answers.
    - Either way, Finish and the send still drop what's hidden (S44a's backstop).
  - **Tests first:**
    - A unit test in `respondent.test.ts`: a number edit 20 → 2 → 25 keeps the follow-up's answer, while 20 → 2 left there (then the save) clears it.
    - J3b (or a new case in it): a number condition, edited with Backspace then a digit at phone size, keeps the follow-up's answer, on screen and in the stored row.
    - The chain tests must still pass.

**Notes (no change needed now; add to pending.md if the builder agrees)**

- **N6. `clearHidden` updates `lastSaved` before the clearing saves come back** (`PatientHQ.tsx:250`).
  - If one fails (not a conflict), later comparisons no longer see those answers as shown, so the server keeps them until the send drops them.
  - It also replaces the other set's draft with the server's copy minus the hidden answers, so an earlier failed, unsaved draft in that set is lost.
  - Both are rare (a failed save is already shown as "Not saved"), and the send's backstop covers the first.
- **The new defaults reach only new HQs** (code review raised it). That's as the brief and Rahul's answer say ("Existing HQs keep their rules"). It isn't a defect, but it's worth knowing: the seeded and QA HQs keep the old Low risk rule.
- **N2, N3 and N5 were raised again by the code review.** They're already in pending.md.

### Test results (28 Oct)

- `make lint test test-integration`: **exit 0**.
  - Vitest: 300 + 13 + 465 passed.
  - CDK: 124 passed.
  - Go unit: all ok.
  - Integration (`-count=1`): every package ok.
- `make e2e` against the conductor's `make dev`: **99 passed, 5 skipped** in 8 min 37 s (the 5 skipped are the AWS-only journeys). J3b passes, but it has no typed-number condition, so it doesn't cover F3.
- A-20 grep over the changed files: only test files (made-up `+44` mobiles in `triage_integration_test.go`, and J16's comment).

### Spot-checks

1. **F3 in a real browser** (Playwright, Chromium at phone size, against `make dev`): reproduced, as above.
2. **F1 and F2 in the code and tests:** they hold. Test report 3 covered them in Chrome, at phone size.
3. **The changed files have no UK literal outside tests.** `repocheck` passes (in `make test`).

### Suggested commits (after F3's fix): verification 1's list stands.

### For the builder

"Verification found problems: read `docs/sessions/S44c-c6-last-answers/verify.md`, fix them test-first, then ask me for a retest."

## Verification 1 (28 Oct 2026, automatic run)

**Verdict: Needs fixes.** One real defect (F1, a one-line fix and a unit test), plus one small one worth doing in the same round (F2). Everything else in the brief is built and matches Rahul's answers of 28 Oct. The tests are all green, and the browser spot-checks passed.

### Findings, worst first

**Must fix**

- **F1. The patient's form clears answers when a set opens, not only when a question is hidden later** (`apps/web/src/features/patient/PatientHQ.tsx:532`, `clearHiddenAnswers(m)` before `m.data = data`; the same order is in `packages/clinical/src/respondent.test.ts:126`).
  - **What happens.** SurveyJS's `clearInvisibleValues` is set *before* the saved answers are loaded. Take a question that is shown while the set is empty but hidden once the answers load: a condition such as `{a} <> 'no'`, "is empty", or any "isn't" test. Its answer is cleared the moment the set opens.
  - **Checked against survey-core directly.** With this setting before `data`, loading `{a:'no', b:'x'}` where `b` shows when `{a} <> 'no'` leaves `{a:'no'}` and fires `onValueChanged('b')`. With the setting after `data`, `b` is kept until a later hide or Finish, as SurveyJS's docs describe.
  - **Why it matters.**
    - The build notes, `decisions.md` (28 Oct, S44c) and `apps/web/AGENTS.md` all say an answer hidden when the set opens "stays until the patient finishes the set". That's only true for `=`-style conditions; the unit test uses only those.
    - The screen then differs from what the server has, while the status still says "Saved".
    - Sign out or the on-behalf Episode link (`flushRef` → `persist`) saves the cleared answers, though the patient changed nothing. For a pre-filled set that's a save past revision 1, so the set counts as "started" without the patient doing anything, which can affect Start fresh ("only while the set is unchanged").
  - **Fix.** Call `clearHiddenAnswers(m)` after `m.data = data` in `SetForm`. Write the failing unit test first: a question whose condition is `{q_b} <> 'n'`, loaded with `{q_b: 'n', q_x: '…'}`, must keep `q_x` until `doComplete()`, with no `onValueChanged` on load. Make the test helper `form()` use the same order. Keep the chain test passing (survey-core still clears on a later hide when the option is set after the data; checked).

**Should fix (small, same round)**

- **F2. On the Rules page, "STOP-Bang score (not in this HQ)" can't be picked again once the author changes the score** (`apps/web/src/features/rules/RuleForm.tsx:68-76`).
  - `notScores` adds the ready-made STOP-Bang only while it is the selected `notBand` score.
  - In an HQ that has another score with bands, an author who changes the "Score not in a band" select loses the option. The defaults' test can then be restored only by rebuilding the rule by hand.
  - **Fix.** Always offer the ready-made STOP-Bang (`readyScoreNames(STOP_BANG_KEYS.score)`) in that select when the HQ hasn't it. Test it in the form's helper, if it has one, or note it for the tester.

**Notes (no change needed now; add to pending.md if the builder agrees)**

- **N1. Checked clicked while On hold or archived drops a held message for good** (`statuses.go:477`, `CheckEpisodeTriage` sets `triage_message_pending = false`). An Undo back to Ready for admission then sends nothing. The brief scopes this to "not yet Checked", and decisions.md already records it ("Checked clicked away from that status ends it (pending.md)"), so it's as decided. Worth confirming with Rahul before go-live.
- **N2. A content save racing another author's delete still says "Someone else changed this chapter…"** (`chapters.go`, `saveContent`'s no-row branch). Rename and delete now say "This Question Set has been deleted." This wasn't in the brief; it's a small wording gap.
- **N3. `notBand` passes when the score is Incomplete.** Only auto-triage is held for an Incomplete score, so an author's own `notBand` rule with a task, flag or suggestion still fires when the band couldn't be decided. The defaults use `notBand` only with auto-triage, which is held, so they're safe. It's documented as the builder's choice. Worth a line in pending.md for authors' own rules.
- **N4. Another trust's Question Set id now gets 404 "This Question Set has been deleted."** (`chapters.go:53`). Nothing leaks, because a missing id gets the same answer; checked in the browser session below. The wording is odd for an id that was never deleted, but it's the builder's recorded choice.
- **N5. Tidy-ups.**
  - `clearedAnswers` turns every other set's JSON into text on each autosave (`respondent.ts`).
  - `shownToPatient` repeats `shownAnswersOnce`'s visibility walk, and Validate builds every patient set's survey to read one.
  - "Is the held message still for this status" is written in two places (`episodes.go:304`, `statuses.go:477`).
  - `samplePatient.ts` only re-exports `SEXES`.
  - None of these is wrong; they're for later.

### Scope and rules checked

- **The six parts of the brief are built:** clear on hide (apart from F1); the held message across Undo and un-archive; `notBand` and the defaults; STOP-Bang Incomplete for sex Other; no "Patient answered" line on a question the patient never saw; the four small fixes.
- **Nothing large is outside the brief.** The sample patient's Sex options are test report 1's F1.
- **No new routes,** so the five-case rule doesn't apply. The changed routes have integration tests: `TestAutoTriageMessages`, `TestCompleteReview`, `TestClinicianSaveWaitsForCompleteReview` and `TestPublishAsksTheEvaluator`.
- **Tenant isolation is unchanged.** The new `GetEpisode` re-reads pass `HospitalID`, and `VersionStatus` reads a version already loaded through the trust-scoped `loadChapterMeta`.
- **A-20:** the grep over the changed files finds only an e2e comment ("en-GB, make dev's default locale"), which is test code.
- **Security:** no secrets, and no answers in logs or messages. Patient JSON is still stripped (`chapter.ForPatient` is untouched). The patient router has no staff routes: `/api/p/o/…` is 404.
- **Docs:** pending.md, decisions.md, CLAUDE.md, AGENTS.md, e2e/README.md, the C6 plan's Status and both tables are updated. `repocheck` passes (in `make test`).

### Browser spot-checks (Claude in Chrome, Browser 1)

1. **The chain on the patient's screen** (new episode on "MT S44c chain 1335"):
   - I answered B Yes → A Yes → C "Coughing fits", then B No. A and C went from the screen, and the stored row was `{"q_gfdmxo": "o_pqzh"}` at revision 4.
   - After a reload, B Yes showed A empty and C hidden. **Pass.**
   - The window couldn't be resized to 390×844 (the extension's resize had no effect), so this ran at desktop width. J3b covers phone size, and passed.
2. **Permissions on a Question Set rename** (API, with dev-login cookies):
   - No sign-in: 401.
   - Cara (clinician): 403.
   - Bea (Trust B) on Trust A's path: 404 "Hospital not found."
   - Bea on her own trust's path with Trust A's set id: 404 for both rename and delete, and the set is unchanged. **Pass.**
3. **The patient router can't reach staff routes:** 404. **Pass.**
4. **The finding behind F1** was confirmed in survey-core with a script (above), not in the browser. No seeded HQ has an "isn't"-style condition with an answer hidden at open.

### Test results (28 Oct)

- `make lint test test-integration`: **exit 0**.
  - Vitest: 300 + 13 + 461 passed.
  - CDK: 124 passed.
  - Go unit: all packages ok.
  - Integration (`-count=1`): every package ok.
- `make e2e` against the conductor's `make dev`: **99 passed, 5 skipped** in 8 min 27 s. The 5 skipped are the AWS-only journeys.

### Suggested commits (after the fixes; the builder's list stands)

1. `S44c: the patient's form clears an answer as soon as its question is hidden, and other sets' answers a change hides`
2. `S44c: a held auto-triage message stays held after Undo or un-archive`
3. `S44c: a High STOP-Bang holds back the default Low risk auto-triage (notBand)`
4. `S44c: STOP-Bang is Incomplete for sex Other`
5. `S44c: no "Patient answered" line on a question the patient never saw`
6. `S44c: small fixes: TouchVersion on drafts only, clearer refusals`
7. `S44c: docs` (leave out `docs/market/` and `docs/sessions/run-state.md`)

### For the builder

"Verification found problems: read `docs/sessions/S44c-c6-last-answers/verify.md`, fix them test-first, then ask me for a retest."
