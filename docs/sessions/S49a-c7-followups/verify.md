# S49a verification

## Verification 2 (14 Nov 2026)

**Reviewer's note:** a fresh session, separate from the builder's. I re-read the brief, the build notes, the test plan and both test reports, then reviewed the whole working-tree diff myself. I also ran `/code-review high` and weighed each of its findings against the code.

**Verdict: Verified.** Verification 1's four findings are fixed. Everything left is minor or cosmetic and goes to pending.md, plus one question for Rahul (finding 2).

### Findings (worst first; none blocking)

1. **Cosmetic: with the record's sex Other or Not known, the card still says "Confirm once every item has an answer."** ([AssessmentsPanel.tsx:209](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L209)). Seen in the browser: on PRISMA-7 with sex Other and every other item answered, the headline correctly reads "6 of 7 filled. The patient's record has no male or female sex, so this score can't be worked out." The dimmed hint under Confirm still asks for every item to be answered, which the clinician can't do. Before S49a the note under the card explained it. Fix: hide the hint when the only missing item is the sex (or say "This score can't be confirmed."). This is one line, so it's a pending item, not a reason to hold the session.
2. **For Rahul: the Up/Down keys are blocked, but the box is still a spin button to a screen reader** ([numberBoxes.ts:26](../../../apps/web/src/lib/numberBoxes.ts#L26)). `<input type="number">` keeps the ARIA role `spinbutton` (J3b finds it by that role), so a screen-reader user hears that Up/Down change it, and then nothing happens. Blocking the arrow keys was the builder's own addition beyond the brief, which asked for "no wheel, no spinner". It's recorded in decisions.md (14 Nov) as "easy to drop". *Recommended:* keep it (a plain box doesn't step), and add "number boxes: spinbutton role with inert arrow keys" to C10's accessibility check. Dropping it is a one-line change if Rahul prefers.
3. **Cosmetic, Preview only: a stale "A number" message** ([EarlierAnswers.tsx:96](../../../apps/web/src/features/preview/EarlierAnswers.tsx#L96)). `PlainNumber` keeps what was typed in local state. If the author types `abc` and then loads a saved test case that gives that question a number, the box shows the number with "A number" still under it. Clear answers doesn't reach these boxes (it never did), so a saved test case loading over a mistyped box is the only way to trigger this.
4. **Trivial: the integration test measures cells in runes, not UTF-16 units** ([audit_integration_test.go:832](../../../apps/api/internal/httpapi/audit_integration_test.go#L832)). Its ids are UUIDs, so the two counts are the same there; `TestCellParts`' emoji case is what guards the UTF-16 rule.

### Checked and not findings

- **Verification 1's fixes:**
  - `test_audit.py`'s `fold_parts` folds a split request back into one row, checking that its parts run 1 to M in order and share the time, who, email, hospital, route and result.
  - The Preview's earlier-answers boxes are `TextInput`s with `inputMode="decimal"`. In Chrome, as Alex in the tester's "MT S49a 1830 earlier numbers" draft (Clinician view), I typed 14, pressed Up×2 and Down×3, and scrolled 3 notches over the box: it still held 14. Both boxes are `type="text"`.
  - `cellLen` counts UTF-16 units, and the unit test has an emoji case.
  - The "115 px" wording is fixed.
- **`fillsGap`** is worked out once, from the pre-fill alone, in `assessmentItems`, and `unpaired` reads the flag. `withDraft`'s old test ("had no source") is gone, so a saved typed BMI changed later keeps the hospital weight. The unit cases cover the screen, `draftAfter` and a second reload. A BMI typed over the hospital's own BMI still empties the weight.
- **The review's other points, dismissed:**
  - `typedNumber` refuses negatives and ".5". So does the assessment cards' `numberInput`, and no pre-op number answer is negative.
  - A missing age can't happen: `date_of_birth` is NOT NULL.
  - The wheel taking the focus off a box was accepted in verification 1.
  - `cellParts` doesn't check a run's first name, but names are at most 100 bytes.
  - `typedNumber` and `numberInput` follow different rules on purpose: the cards' boxes take one decimal place and have limits.
- **The CSV's Action cell says "(part N of M)".** A spreadsheet filter on "begins with Listed episodes" still finds every part. This is the builder's recorded choice (decisions.md, 14 Nov).
- **Scope:** all six parts of the brief are built, and nothing large beyond them.
- **Tests:** there's no new route, so no new five-case set; the export route keeps its cases. Unit tests cover `fillsGap`, `filledText`, `plainNumberBoxes`, `typedNumber` and `cellParts`. Journeys cover J3b (the wheel, Chromium at 390×844), J10 (the earlier answers), J12 (MUST and the menu), J7 and Audit.
- **Tenant isolation:** no query changed, and the CSV writer works on rows already scoped to the trust.
- **A-20:** the grep finds only older comments and the infra tests' own UK lists, nothing in the changed files.
- **Security:**
  - No secrets in the diff.
  - Nothing new is logged.
  - `csvSafe` still runs on every part.
  - The patient router is untouched.
- **Docs:** C7's Status, `docs/plans/README.md`, `docs/sessions/README.md`, pending.md, decisions.md (including the `make dev` gotcha in §4) and `apps/web/AGENTS.md` are all updated, and `repocheck` passes.

### Browser spot-checks (Claude in Chrome, 14 Nov; `make dev`'s API started 18:22, after the last Go edit at 17:46)

1. **The Preview's earlier answers (Alex):** 14 held through the arrow keys and the wheel; both boxes are `type="text"` with `inputMode="decimal"`. `abc` shows "A number". Clear answers leaves these boxes alone (it did before S49a too).
2. **Sex Other on PRISMA-7 (Val, Hospital A2, made-up patient "Ola MT S49a verify 56296", email only):** the headline is the new sentence, both while items were missing ("1 of 7 filled: answer the 5 items highlighted. The patient's…") and with only the sex missing. The old note under the card is gone, leaving finding 1's hint.
3. **Earlier checks I didn't repeat:** test report 2 re-ran the CSV split (900-episode row: 2 parts, every id once, longest cell 32,754 UTF-16 units) and the other trust's 403, after `make dev` was restarted. The integration test covers both.

### Test results

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 404 passed (36 files), evaluate 25 passed, clinical 653 passed (30 files); CDK 132 passed.
- `make test-integration`: all packages ok (httpapi 27.1 s).
- `make e2e` (against `make dev`, its API started after the last Go change; Chromium and WebKit): **124 passed, 5 skipped** (Cognito-only and QA-only) in 875.85s (0:14:35).

### Suggested commits

1. `S49a: MUST keeps the hospital weight when a saved typed BMI changes`: `packages/clinical/src/assessments.ts`, `assessments.test.ts`, `e2e/journeys/test_j12_observations.py` (the no-height test's hunk).
2. `S49a: number boxes have no wheel, arrow keys or spinner`: `apps/web/src/lib/numberBoxes.ts` and `numberBoxes.test.ts`, `main.tsx`, `globals.css`, `features/preview/EarlierAnswers.tsx`, `e2e/journeys/test_j3b_hidden_answers.py`, `e2e/journeys/test_j10_authoring.py`.
3. `S49a: the audit CSV splits a list too long for a spreadsheet cell`: `internal/httpapi/audit.go`, `audit_test.go`, `audit_integration_test.go`, `e2e/journeys/test_audit.py`.
4. `S49a: an assessment says when the record has no male or female sex`: `apps/web/src/features/episodes/assessments.ts` and `assessments.test.ts`.
5. `S49a: J7 counts only the texts captured during its run`: `e2e/journeys/test_j7_patient_invites.py`.
6. `S49a: an assessment card's menu opens under its box`: `AssessmentsPanel.tsx`, `e2e/journeys/test_j12_observations.py` (the menu test's hunk).
7. `S49a: docs`: the session folder, `docs/pending.md`, `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/c7-observations-assessments.md`, `docs/sessions/README.md`, `apps/web/AGENTS.md`.

J12 holds hunks for commits 1 and 6, so I'll split it by hunk (`git apply --cached` of each hunk). If you'd rather not split it, it can go whole in commit 6.

## Verification 1 (14 Nov 2026)

**Reviewer's note:** this verification ran in the same Claude session that built S49a, so it isn't fully independent. To make up for that, the code review ran in a separate reviewer that hadn't seen the builder's reasoning, and each of its findings was re-checked against the code; the risky parts were re-checked in the browser rather than taken from the build notes or test report. A fresh session's `/verify-session S49a` would be more independent if Rahul prefers.

**Verdict: Needs fixes** (a failing journey, and two small fixes, test-first).

### Findings (worst first)

1. **`make e2e` fails: the Audit journey expects the CSV to have as many rows as the screen** ([test_audit.py:121](../../../e2e/journeys/test_audit.py#L121): "29 CSV rows, 19 listed"). Cara's rows filtered to one patient include "Listed episodes" rows, and on a local database with about 4,390 episodes in Hospital A1 each now exports as 6 "part" rows. That is the new behaviour working as briefed, but the journey wasn't updated. **The build notes' e2e result was wrong for this journey:** the builder's `make e2e` ran against a Go API started before the CSV change (`make dev` doesn't rebuild Go on a change), so it still exported one row per request; the tester restarted `make dev` afterwards. Fix: the journey folds a request's "(part N of M)" rows back into one before comparing (and checks the parts share the time and route), and the build notes' e2e line is corrected. **Gotcha for decisions.md §4:** a journey run after a Go change needs `make dev` restarted first.
2. **The Preview's earlier-answers number box still has its up/down arrows, and the arrow keys step both of that panel's number boxes** ([EarlierAnswers.tsx:73](../../../apps/web/src/features/preview/EarlierAnswers.tsx#L73)). The brief says every number answer, "the Preview" included, is a plain box. Only the BMI box (line 70) got `hideControls`; the plain number one beside it didn't. Both are Mantine `NumberInput`s (they render `type="text"`), so `plainNumberBoxes`, which only handles `type="number"`, doesn't stop Mantine's own Up/Down stepping, and `hideControls` only hides the arrows. The build notes say "The Preview's earlier-answers BMI gets `hideControls`", which is true but leaves the other box out. Why it matters: these are earlier answers that feed the Preview's scores, the same risk the session set out to remove (a value moving by accident). Fix: a plain box for both (a `TextInput` with `inputMode="decimal"`, as the assessment cards do, or stop Mantine's stepping), with a unit or journey test.
3. **The audit CSV's cell limit is counted in code points, while a spreadsheet's 32,767-character limit counts UTF-16 units** ([audit.go:613](../../../apps/api/internal/httpapi/audit.go#L613) `cellParts`, `utf8.RuneCountInString`). A name with characters outside the Basic Multilingual Plane (an emoji, some CJK characters) counts once here and twice in Excel, so a names cell close to the limit could still be cut short. Names are typed by people, so this is reachable, though unlikely. Fix: count UTF-16 units (`len(utf16.Encode([]rune(s)))`, or `utf16.RuneLen` per rune), and add a case to `TestCellParts`.
4. **Trivial:** the comment at [AssessmentsPanel.tsx:257](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L257) and J12's docstring say the menu opened "110 px" to the right; the measurement was 115 px (the brief's "about 110" came from the test report). Say "about 115 px", or "off to the right".

### Checked and not findings

- **`fillsGap` is worked out from the pre-fill now, not from when the BMI was typed** (`packages/clinical/src/assessments.ts`). If the hospital later records a height, a saved typed BMI is then "typed over a pre-filled BMI", and the weight goes Missing on the next load. That follows Rahul's 13 Nov rule (a BMI typed over a pre-filled one empties the other), it's how S49 behaved before this session, and it never under-scores: the card goes Incomplete until the weight is entered. The brief asked for exactly this ("work out … once, in `assessmentItems`").
- **The age, like the sex, can't be answered on the card,** but every patient record has a date of birth (`date_of_birth NOT NULL`), so an assessment's age is never missing.
- **One over-long name can't make a cell over the limit on its own:** names are at most 100 bytes each ([patients.go:118](../../../apps/api/internal/httpapi/patients.go#L118)).
- **"Part N" puts runs of patients and runs of episodes side by side:** as in the single row before, a row's patients and episodes are two separate lists, never paired by position.
- **A wheel over a focused number box takes the focus off it**, even a small trackpad movement. That's the design the build notes give (the page still scrolls; it triggers SurveyJS's blur-time save). Accepted.
- **Scope:** all six parts of the brief are built; nothing large beyond it. The builder's choices (the arrow keys too; any over-long CSV row; the duplicate sex note removed) are recorded in decisions.md (14 Nov, "S49a built and tested").
- **Tests:** no new route, so no new five-case set. The export route keeps its cases (`TestListAudit`, `TestAuditLimitAndCSV`); the new `TestAuditCSVSplitsALongList` and `TestCellParts` cover the split. Unit tests cover the MUST fix, the headline and `plainNumberBoxes`; journeys cover the MUST fix (J12), the wheel (J3b, Chromium at 390×844), the menu (J12) and J7.
- **Tenant isolation:** unchanged queries; the CSV writer works on rows already scoped to the trust.
- **A-20:** no region, time zone, locale or phone literal in the changed files (the `+44…` in J7 is its existing made-up test mobile).
- **Security:** no secrets; nothing new is logged; the CSV still defuses formulas (`csvSafe` runs on every part).
- **Docs:** the C7 Status section, `docs/plans/README.md`, `docs/sessions/README.md`, pending.md and decisions.md are updated; `go test ./internal/repocheck` passes.

### Browser spot-checks (Claude in Chrome, 14 Nov)

1. **The Preview's number answer:** in the Demo risk HQ's Preview, the weight box is `type="number"` with `appearance: textfield` (no arrows). Typed 70, a real mouse wheel (3 notches down) over the focused box left 70, moved the focus off it and scrolled the page. (The Sample patient Age box beside it still has its arrows: pending.md, from test report 1.)
2. **The audit CSV split (as Ada, Cara's last 20 minutes):** one list request split as "Listed episodes (part 1 of 6)" to "(part 6 of 6)", with the same time, who, email, hospital, route and result on every part; 4,391 episode ids, all different; the longest cell 32,754 characters. The screen shows one row per request with the whole count.
3. **Another trust's admin:** Max Multi (admin of Trust B, a clinician at Trust A's Hospital A2) gets **403** "Only the trust's admins can see the audit log." for Trust A's CSV.

### Test results

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest web 402 passed, evaluate 25 passed, clinical 653 passed; CDK 132 passed.
- `make test-integration`: all packages ok.
- `make e2e` (against `make dev`, restarted by the tester with the S49a Go code): **1 failed** (`test_audit.py::test_an_admin_finds_a_clinicians_actions_filters_them_and_exports_the_csv`, finding 1), 122 passed, 5 skipped.

### Suggested commits (after the fixes)

1. `S49a: MUST keeps the hospital weight when a saved typed BMI changes`: `packages/clinical/src/assessments.ts`, `assessments.test.ts`, J12's no-height test.
2. `S49a: number boxes have no wheel, arrow keys or spinner`: `apps/web/src/lib/numberBoxes.ts` and its test, `main.tsx`, `globals.css`, `EarlierAnswers.tsx`, J3b's new test.
3. `S49a: the audit CSV splits a list too long for a spreadsheet cell`: `internal/httpapi/audit.go`, `audit_test.go`, `audit_integration_test.go`.
4. `S49a: an assessment says when the record has no male or female sex`: web `assessments.ts` and its test.
5. `S49a: J7 counts only the texts captured during its run`: `e2e/journeys/test_j7_patient_invites.py`.
6. `S49a: an assessment card's menu opens under its box`: `AssessmentsPanel.tsx`, J12's menu test.
7. `S49a: docs`: the session folder, pending.md, the status tables, decisions.md, `apps/web/AGENTS.md`.
