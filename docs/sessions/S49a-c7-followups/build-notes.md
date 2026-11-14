# S49a build notes: C7 follow-ups

[Brief](brief.md) · [Test plan](test-plan.md). Plan C7, a follow-up after S49. Built 14 Nov 2026.

No questions: every item was answered by Rahul on 14 Nov (decisions.md, "C7's open items answered").

## What was built

### 1. MUST: a saved typed BMI keeps the hospital weight when changed (S49 verification 7, finding 1)

**TypeScript, for Rahul to review** (`packages/clinical/src/assessments.ts`):
- An item gains **`fillsGap`**: set by `assessmentItems` on a BMI or weight the **pre-fill** has no value for (MUST's BMI where the hospital has a weight but no height). It's worked out from the pre-fill alone, never from the saved row, so a saved typed BMI still counts as filling the gap.
- **`unpaired`** (the rule "a typed BMI or weight clears the other") now reads that flag, instead of taking a test function. `assessmentItems` and `withDraft` both call it with no second argument, so the gap test exists once. Before, `withDraft`'s test was "the item had no source", which a saved BMI fails, so changing it emptied the hospital weight.
- A BMI typed **over the hospital's own BMI**, saved and changed, still empties the weight (the gap is the pre-fill's, so `fillsGap` isn't set): tested.

**Tests:** two unit cases in `assessments.test.ts` (the saved BMI changed, on the screen, through `draftAfter`, and after a second save and reload; and the over-the-hospital's-BMI case). Every S49 pairing case (verifications 3–6) passes unchanged. **J12:** `test_must_with_a_hospital_weight_but_no_height_keeps_the_weight_when_the_bmi_is_typed` now saves, reloads, changes the BMI to 22 and checks the weight stays "Hospital weight" 60.

### 2. Number boxes: no mouse wheel, no spinner (S49 test report 1, exploring 1)

**Checked first (SurveyJS v3.1.2's source, React 19's):**
- SurveyJS draws a Number question as `<input type="number">` (`type: this.question.inputType` in survey-react-ui), turns the typed text into a number itself (`correctValueType`), and blocks `e`, `E` and `+` on keydown (`shouldPreventNumberInput`). There's no SurveyJS setting for the wheel or the arrows; its numeric `inputMask` would change every HQ's JSON and the value checks.
- **React 19 sets an input's `type` again on every update** (`updateInput` in react-dom: `element.type = type`), so changing the box to `type="text"` from `onAfterRenderQuestion` wouldn't hold past the first keystroke. So the box stays `type="number"`, and SurveyJS's checks (a number, min, max) are untouched.
- Measured in Playwright: on Chromium (desktop and phone emulation) a wheel over a focused number box stepped 6 to 5; on WebKit it didn't.

**Built (TypeScript, for Rahul to review):**
- **`apps/web/src/lib/numberBoxes.ts`** `plainNumberBoxes(document)`, installed once in `main.tsx` (like `reloadWhenRestored`), so it covers every view: the patient's form, Fill in for the patient, Validate, the Preview and Test the whole HQ. Two document listeners:
  - a **wheel** over the focused number box takes the focus off it (`blur`), so the browser doesn't step the value, and the page still scrolls (not `preventDefault`, which would stop the page scrolling under the pointer);
  - the **Up and Down arrow keys** on a number box do nothing (`preventDefault`).
- **`globals.css`:** `appearance: textfield` and no `::-webkit-inner/outer-spin-button` on `input[type="number"]`: no up/down arrows.
- The Preview's earlier-answers BMI (a Mantine `NumberInput`) gets `hideControls`.
- The assessment cards' boxes were already text boxes (`TextInput`); checked, unchanged.

**Tests:** `numberBoxes.test.ts` (3 cases: the wheel blurs only the focused number box and doesn't stop the scroll; other boxes untouched; the arrow keys). **J3b:** `test_a_scroll_over_a_focused_number_box_never_changes_it`, on **Chromium at 390×844** (the `phone` fixture is WebKit, which never stepped, so a step there couldn't fail): the wheel down and up and the arrow keys leave 6, `appearance` is `textfield`, and the stored answer is 6. Run without `plainNumberBoxes` it fails (6 became 5); with it, it passes.

### 3. The audit CSV splits a long list (S49 verification 3)

**Go** (`internal/httpapi/audit.go`):
- `writeAuditCSV`: a row whose Patients, Patient record IDs or Episode IDs cell would pass **32,767 characters** (Excel's cell limit; one less, for the apostrophe `csvSafe` may add) is written as several rows, the same time, who, email, hospital, route and result on each, the Action "Listed episodes (part 1 of 2)". Each part holds a run of the patients (names and ids kept side by side) and of the episodes; every id once.
- `cellParts(columns ...[]string) [][2]int` does the splitting by characters (not bytes). It applies to any action, though only list rows can get that long. 862 ids fit one cell; 863 need two.
- The audit table and the screen's list are unchanged.

**Tests:** integration `TestAuditCSVSplitsALongList` (an `episode.list` row of 900 patients and 900 episodes: 2 CSV rows, "part 1 of 2" / "part 2 of 2", same request, no cell over 32,767, names beside ids, every id once; the table row still holds 900 of each; the screen shows one row with counts of 900). Unit `TestCellParts` (0, 862 and 863 ids; two columns where long names end a run first). No new route, so no new five cases; the export route's are TestListAudit's and TestAuditLimitAndCSV's.

### 4. The record's sex is Other or Not known (S48 test report 1)

**TypeScript** (`apps/web/src/features/episodes/assessments.ts`): `filledText` no longer asks for the sex item (it can't be answered on the card); when it's missing it adds *"The patient's record has no male or female sex, so this score can't be worked out."*, also on a read-only card. E.g. "7 of 8 filled. The patient's…" or "6 of 8 filled: answer the item highlighted. The patient's…".

**Tests:** `assessments.test.ts`: Other and Not known on PRISMA-7, Apfel and STOP-Bang (editable and read-only); the sex the only item missing; other items still asked for, the sex not counted; male or female as before.

### 5. J7's flaky mobile (S49 verification 7)

**J7** (`test_j7_invite_by_text_and_email_then_date_of_birth_and_send`): notes the newest captured message id for its made-up mobile before creating the episode, then checks exactly one new text since (by id, via `captured`), and that the /qa/messages page's newest card is it. Before, it expected the page to show exactly one text for a mobile that's one of 1,000. The email step after it now waits for the email's own card (its address on the newest card, then exactly one): the first full `make e2e` hit a real collision there, the page still showing the mobile's two cards when the step read the list (a strict-mode error). Then run with the mobile pinned to one with 4 earlier texts (a forced collision, the pin reverted after): passed; and 5 times with random mobiles: passed.

### 6. An assessment card's menu opens under its box (S49 test report 6, exploring 3)

**TypeScript** (`AssessmentsPanel.tsx`): the menu's `comboboxProps` were `{ width: 300, position: "bottom-end" }`: a 300 px list pinned to the right edge of a box the table had stretched to 410 px, so it began 115 px right of the box. Now `{ width: "target", position: "bottom-start" }` with `styles={{ dropdown: { minWidth: 300 } }}`: as wide as its box, starting at its left edge, never narrower than the Frailty Scale's longest level needs.

**Tests:** J12 `test_an_assessment_cards_menu_opens_under_its_box` (Morse's Gait and the Frailty Scale's Level: the list starts within the dropdown's own 5 px padding of the box's left edge and is at least as wide; "8: Living with Very Severe Frailty" shows whole). It failed before the fix (115 px).

## Decided by the builder

1. **The arrow keys don't step a number box either** (Rahul asked for no wheel and no spinner: "keep it a simple input box"; a plain box doesn't step on Up/Down, and the risk is the same). Easy to drop if he'd rather keep them.
2. **The fix is app-wide, for every `<input type="number">`** (one listener and one CSS rule), rather than per survey model: simpler, and it also covers any number box a later view adds. The settings and jobs pages use Mantine's `NumberInput` (a text box with its own arrows), left as they are: they're settings, not answers.
3. **The CSV splits any row too long for a cell**, not only "Listed episodes", with "(part N of M)" added to the Action.
4. **The card's second line about the sex is gone** ("The patient's record doesn't say male or female, so the score stays Incomplete, as in the HQ."): the headline now says it, and saying it twice was clutter. `cardNotes` keeps the age line.

## Test results (14 Nov)

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 402 passed (36 files), evaluate 25 passed, clinical 653 passed (30 files); CDK 132 passed.
- `make test-integration`: all packages ok (httpapi 28.8 s).
- `make e2e` (against `make dev`, Chromium and WebKit): 122 passed, 5 skipped (Cognito-only and QA-only), **1 failed: J7's email step**, a collision on the made-up mobile (above), fixed. After the fix, J7's file (6 journeys) passed 3 times, once with a forced collision. **Corrected after verification 1:** this run's Go API had been started before the CSV change (`make dev` doesn't rebuild Go), so the Audit journey still saw one CSV row per request; with the change it fails (finding 1, fixed below).

## Suggested commits (Rahul commits after verification)

1. `S49a: MUST keeps the hospital weight when a saved typed BMI changes` — `packages/clinical/src/assessments.ts`, `assessments.test.ts`, J12's no-height test.
2. `S49a: number boxes have no wheel, arrow keys or spinner` — `apps/web/src/lib/numberBoxes.ts` and its test, `main.tsx`, `globals.css`, `EarlierAnswers.tsx`, J3b's new test, J10's earlier-answers journey (verification 1).
3. `S49a: the audit CSV splits a list too long for a spreadsheet cell` — `internal/httpapi/audit.go`, `audit_test.go`, `audit_integration_test.go`, `e2e/journeys/test_audit.py` (verification 1).
4. `S49a: an assessment says when the record has no male or female sex` — web `assessments.ts` and its test.
5. `S49a: J7 counts only the texts captured during its run` — `e2e/journeys/test_j7_patient_invites.py`.
6. `S49a: an assessment card's menu opens under its box` — `AssessmentsPanel.tsx`, J12's menu test.
7. `S49a: docs` — this folder, pending.md, the status tables, decisions.md.

## Test report 1 (14 Nov): all passed, 17 of 17

- T-05 and T-12 step 3 were wrong in the plan (a sent Demo risk HQ is auto-triaged, so read-only; Mental status is two buttons): the tester ran equivalents, and the plan is corrected.
- The tester ran T-03 at 500×701 (Chrome won't go narrower); the journey `test_a_scroll_over_a_focused_number_box_never_changes_it` covers 390×844.
- T-08 ran on real local list rows (4,383 episodes: 6 parts, longest cell 32,754), as the Setup's insert hadn't been run.
- Its exploring notes are in pending.md ("From S49a's test report 1"); none is new in S49a. The most notable, for Rahul: Validate saves a number only when its box loses focus while the header says "Saved".

## Verification 1 (14 Nov): needs fixes; fixed 14 Nov

([verify.md](verify.md).)
1. **The Audit journey counted CSV rows against screen rows** (29 CSV rows, 19 listed: each "Listed episodes" row of about 4,390 episodes is now 6 "part" rows). `test_audit.py`'s `fold_parts` folds a request's "(part N of M)" rows back into one, checking they are parts 1 to M in order with the same time, who, email, hospital, route and result, before comparing. Checked to fail without the fold ("29 CSV rows, 19 listed") on a second stack whose Go was started after the CSV change.
2. **The Preview's earlier-answers number boxes still stepped** (Mantine `NumberInput`s, `type="text"`, so `plainNumberBoxes` didn't reach them; only the BMI had `hideControls`). Both are now `PlainNumber`, a `TextInput` with `inputMode="decimal"` (`EarlierAnswers.tsx`), read by `typedNumber` (`lib/numberBoxes.ts`: digits with one decimal point or comma; "20." counts while typing 20.8). TypeScript for Rahul: `PlainNumber` keeps what was typed while it is the value, like the assessment cards' boxes. Tests (red first): `typedNumber`'s unit cases; a J10 journey (an HQ whose Neck score uses an earlier number and the earlier BMI: arrow keys and the wheel leave 14 and 20.8), checked to fail with the old boxes ("unexpected value 15").
3. **The CSV cell limit is counted in UTF-16 units** (`cellLen` in `audit.go`, as a spreadsheet counts: an emoji is two). `TestCellParts` has 100 names of 1,000 emoji (red first: a part of 64,062 units).
4. "about 115 px", not 110, in `AssessmentsPanel.tsx`'s comment, J12's docstring and the test plan.
- Test plan: T-06 step 3 (the earlier answers' boxes).
- Tests after the fixes: `make lint`, `make test` (clinical 653, web 404, evaluate 25, CDK 132, Go), `make test-integration` pass. Full `make e2e`: 124 passed, 5 skipped in 816.57s (0:13:36) (second stack, its Go started after every Go change; one new journey).

## Test report 2 (14 Nov): all passed, 13 of 13

- The retest after verification 1's fixes, with `make dev` restarted after the last Go change: T-06 step 3 (new), T-05, T-12 steps 2–3, the audit CSV (T-08 with the plan's 900-episode row: 2 parts, 900 ids, none repeated, longest cell 32,754 UTF-16 units; T-09), the happy paths and R-04, R-05.
- Its exploring notes are in pending.md ("From S49a's test report 2"): two cosmetic notes, two seen again from report 1. None is new in S49a. (Its note 2 quotes verify.md: the assessment cards' boxes have no `inputMode`; only the Preview's new boxes and Observations have it.)
