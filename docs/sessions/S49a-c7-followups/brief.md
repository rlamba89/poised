# S49a: C7 follow-ups (Rahul's answers of 14 Nov)

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), a follow-up session added on 14 Nov, after S49 and before C8.
**Requirements:** the open items S49 left in [pending.md](../../pending.md) ("From S49's build"), answered in chat on 14 Nov (decisions.md, 14 Nov, "C7's open items answered").
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S49 verified (14 Nov).

**Ask Rahul first:** nothing. Every item below was answered on 14 Nov. If an item turns out bigger or different from what's written here, ask (questions.md); don't widen it.

## Build (each part test-first: the failing test, then the code)

### 1. MUST: a saved BMI typed where the hospital has no height keeps the hospital weight when changed (S49 verification 7, finding 1)

- Rahul's 13 Nov rule: a BMI the clinician types where the pre-fill has none (a hospital weight, no height) keeps the hospital's weight beside it. Today that holds on the first typing and after a reload, but **changing a saved typed BMI** empties the hospital weight on screen (`withDraft` takes "typed into a gap" to mean "had no source", and a saved BMI has one).
- Fix in `packages/clinical`'s `assessments.ts`: work out "this measure fills a gap in the pre-fill" once, in `assessmentItems` (a flag on the item), and use it in both `assessmentItems` and `withDraft`, which also removes the pairing rule's duplicated gap test.
- Unit cases: a saved typed BMI beside a hospital weight with no height, changed: the weight stays Hospital weight; the screen and a reload agree. Every S49 pairing case (verifications 3–6) passes unchanged. One J12 step.

### 2. Number boxes: no mouse wheel, no spinner (S49 test report 1, exploring 1)

- Rahul: "we should not have [the wheel] at all; keep it a simple input box." Every number answer is a plain text box: scrolling over it never changes its value, and it has no up/down arrows. This covers the patient's form, the clinician's Validate copy, Fill in for the patient, the Preview, and the assessment cards (already text boxes; check).
- In SurveyJS, a `text` question with `inputType: "number"` renders `<input type="number">`. Prefer one place in the web app (the survey model setup shared by every view, e.g. an `onAfterRenderQuestion` or a CSS and `inputmode="decimal"` approach) over changing every questionnaire's JSON. Check SurveyJS's docs first; validation (min, max, a number) must stay as it is.
- Tests: a unit or component test where possible; a journey step in a patient journey (phone size) that scrolls over a focused number box and checks the value didn't change.

### 3. The audit CSV: long "Listed episodes" id lists fit a spreadsheet (S49 verification 3)

- Rahul (14 Nov, as on 27 Oct): **keep every id** in the audit log. Only the CSV export changes: a "Listed episodes" row whose ids pass a spreadsheet cell's 32,767 characters is split across rows (the same action, time and user, "part 1 of N"), so the file opens whole in Excel and LibreOffice.
- Integration test: a list of more than 860 episodes exports as several rows holding every id once; the audit table itself is unchanged.

### 4. The record's sex is Other or Not known (S48 test report 1)

- On PRISMA-7, Apfel and STOP-Bang, when the item highlighted is the record's sex and the record has neither male nor female, the card's headline says so instead of "answer the item highlighted": e.g. "The patient's record has no male or female sex, so this score can't be worked out." (web `assessments.ts`, `filledText`). Web unit test.

### 5. J7's flaky mobile (S49 verification 7)

- `made_up_mobile` picks one of 1,000 numbers, so on a large local database J7 can find two captured texts for its "new" mobile. Filter the captured messages also by the time the journey started, or by the patient's surname. Run J7 a few times on Rahul's local database.

### 6. Cosmetic: an assessment card's menu list opens under its box (S49 test report 6)

- The Morse card's Gait list opens about 110 px right of its closed box (`comboboxProps` width 300, position `bottom-end` in `AssessmentsPanel.tsx`). Open it lined up with the box (still wide enough for the Frailty Scale's longest level, S49 report 1's exploring 2).

## Not in this session (Rahul, 14 Nov)

- A retired starter question's key stays offered by "Same as…" through a trust's own copies, as built.
- The clinician checks (wording, bands, MUST's rounding edges), the licences and the other go-live items stay in pending.md for C10.

## Tests

- `make lint`, `make test`, `make test-integration`, and the full `make e2e` (J7, J12, the patient journeys and the audit journey especially).
- Update pending.md (each item done), the build notes, and the status tables.
