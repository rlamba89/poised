# S37 build notes: no conditions on summary boxes

Builder, automatic run, 26 Oct 2026. No plan of its own: Rahul's answer of 26 Oct to S32's open question ([decisions.md](../../decisions.md) §2; [pending.md](../../pending.md), "From S32's verification 1"). Nothing to ask (no answers.md).

## What was built

### `packages/clinical` (TypeScript, for Rahul to review)

`src/summary.ts`:
- **`hasCondition(el)`**: true when an element has a non-empty `visibleIf`, `requiredIf` or `enableIf` (shown, required or read-only when…).
- **`SummaryInfo.conditional`**: each summary found by the walk records `hasCondition`.
- **`summaryProblems`** adds one problem per summary with a condition: `The clinical summary “<title>” has a condition: remove it (a summary is always shown)`. `publishProblems` (`publish.ts`) already adds every summary problem with the Question Set's name, so the Publish window shows "About you · The clinical summary “Heart checks” has a condition: …". Nothing else changed in `publish.ts`.
- **`logicOffered(el)`**: the editor's rule, `!isSummary(el) || hasCondition(el)`. No Logic on a summary, unless it already has a condition (made by hand or by the API), so the author can remove it.

### `apps/web` (TypeScript)

- `features/editor/SettingsPanel.tsx`: the **Logic** tab (and its panel) shows only when `logicOffered(el)`; if the editor asks for Logic on an element that doesn't offer it, Settings shows instead. A summary's Settings show an orange warning (`data-testid="summary-condition"`) when it has a condition and the user can change it.
- `features/editor/LogicEditor.tsx`: the Code tab also lists "Required if" and "Enable if" on an element that isn't an answer but has one (a summary from the API), so every condition can be removed there.
- The card's "Displays when" box is unchanged: it shows only when there's a condition, so a summary normally has none; one that has a condition shows it, and clicking it opens the Logic tab to remove it.

### Go

**Nothing.** Go's publish (`publishQuestionnaire`) checks only the question keys (S23); summaries, logic problems and test cases are the browser's checks (plan 2.1), so there is **no integration test** for this (the brief: "only if Go checks summaries at publish"). The API alone can still save and publish a summary with a condition, like a logic problem; the editor's Publish window is the check.

### Published versions

Never rewritten. `withClinicalSummaries` still copies a summary's `visibleIf` onto its clinician panel, so a published summary with a condition behaves as before.

### End-to-end

`e2e/journeys/test_j10_authoring.py`:
- the summaries journey also checks that a summary has no Logic tab and no "Displays when", and a question still has Logic;
- new `test_a_summary_with_a_condition_cannot_be_published`: a draft from `e2e/data/summary-condition-hq.json` (a page summary with a `visibleIf`, as the API can save one): Publish names the summary and is disabled; the summary's Settings warn; the condition is removed in Logic → Code; the Logic tab, warning and "Displays when" go, and the HQ publishes.

## Decided by the builder (automatic mode)

1. **The Logic tab stays offered on a summary that already has a condition**, only so it can be removed. The brief says both "no Logic tab for a summary" and "the author can remove the condition from the Code tab", and the Code tab is inside Logic. A new summary never has one; one with a condition (only possible from outside the editor) shows Logic until it's removed.
2. **A warning in the summary's Settings** says why it can't be published and how to fix it (hidden from read-only viewers, who can't).
3. **"Linking to the item, as other publish problems do":** no publish problem links to its item today (the window lists plain text: Question Set · message). The new problem follows the same form, naming the Question Set and the summary's title. Making publish problems clickable would be a change for every problem, so it's left out.
4. **An empty condition (`""` or spaces) is no condition**, so it neither blocks Publish nor offers Logic.
5. **The tester gives a summary a condition with a Console snippet** in the test plan (the API's chapter save), since the editor no longer can.

## Tests (each written before its code)

- `packages/clinical/src/summary.test.ts`: `summaryProblems` with a shown/required/read-only condition on three summaries, an empty condition, and `logicOffered` (summary, summary with a condition, other elements).
- `packages/clinical/src/publish.test.ts`: the publish problem naming the set and the summary; a summary without a condition passes.
- J10 as above.

### Results (26 Oct)

- `make lint`: passes.
- `make test`: Go ok; Vitest 268 (web) and 309 (clinical) passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (all journeys, against a private stack on :3100/:8090): 84 passed, 5 skipped (the Cognito-only ones), 15 min.

## Hand checks for Rahul

None expected: everything in the brief can be done in the browser (the condition is added with the test plan's Console snippet).

## Suggested commits

1. `S37: no conditions on summary boxes` — `packages/clinical/src/{summary,summary.test,publish.test}.ts`; `apps/web/src/features/editor/{SettingsPanel,LogicEditor}.tsx`; `e2e/journeys/test_j10_authoring.py`, `e2e/data/summary-condition-hq.json`; `apps/web/AGENTS.md`.
2. `S37: docs` — this folder, `docs/{decisions,pending}.md`, session status.

## Test rounds

- Test plan written 26 Oct ([test-plan.md](test-plan.md)).
- **Test report 1** (26 Oct): **all passed, 17 of 17** (13 cases, 4 regression checks). **S37 tested.**
  - Exploring 1 and 6: checks that passed (required-if, enable-if and a spaces-only condition from the API; Undo).
  - Exploring 2–5 and 7, no code change (pending.md, "From S37's test report 1"): the Code tab saves one field at a time; Display: Always can't remove a required-if or enable-if; two same-titled summaries give one problem line; the delete window's wording for a summary; the "Clinical Page" heading.
  - No hand checks.
