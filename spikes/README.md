# Step 0 spikes (throwaway)

SurveyJS 3.1.2 (exact pin), Next.js 16.3.7, React 19.3. Run on 30 Sep 2026.

## Results

| # | Spike | Result |
| --- | --- | --- |
| 1 | Creator renders in Next.js via `dynamic(..., { ssr: false })` | **Pass** |
| 2 | `clinicalOutputs` set from our React modal: modified, autosave, undo, redo | **Fail as planned**, **pass with a fix** (below) |
| 3 | Generated `name` / `value` keep the default label, including on copy | **Pass** |
| 4 | `getAllQuestions(true)` excludes questions in hidden panels | **Pass** |

### Spike 2 failure
- **Redo loses question-level outputs.** When a property already holds an array, `Base.setPropertyValue` in survey-core overwrites the existing array's contents instead of storing the new array (questions have `arraysInfo`, so they take this path). Undo therefore empties the array that the redo step recorded. Option items don't take this path, which is why their redo works.
- **Rapid edits merge into one undo step.** A custom property with no type defaults to `string`, and the Creator merges edits to string properties made within 1 second.
- **Fix that passes (`/spike2?fix=1`):** give the property a non-string type and hold the outputs internally in an immutable `{ items }` box. The saved JSON stays a plain array because of the public `onSerializeValue` and `onSetValue` hooks. All reads and writes go through two helpers, `outputsOf(obj)` and `setOutputs(obj, items)`.

### Other findings
- `showTitlesInExpressions` is **deprecated and hidden in v3**. Use `useElementTitles: true`. The Logic tab then shows "Do you smoke?" instead of `q_smoke1` (checked).
- **The built-in "None" item can't carry outputs.** Only `showNoneItem` and `noneText` are saved, so custom properties on `noneItem` are lost. Use a normal choice with `isExclusive: true` (v3 `checkboxitem`) instead: it's saved in `choices`, gets a stable ID and keeps its outputs (checked).
- **React Creator class.** `SurveyCreatorComponent` needs `new SurveyCreator(...)` from `survey-creator-react`, not `SurveyCreatorModel`.
- **Adorner badge text.** It needs `disableShrink: true`, otherwise the question toolbar shrinks it to an icon, and it has none. A `ComputedUpdater` title updates live on add, undo and redo.
- **Default choices on a new question** don't raise `onItemValueAdded`, so give them IDs in `onQuestionAdded`. The Creator also pre-fills the next choice value by incrementing the last one; `onItemValueAdded` overrides it.
- The trial-licence banner shows, as expected.

## Re-run
```sh
npm install
npx tsx spike4-visible-questions.ts
npx tsx spike2-exclusive.ts
npx next build && npx next start -p 3100 &
# CHROME_PATH is optional: point it at a local Chromium if Playwright's own isn't installed
npx tsx spike2.test.ts          # as planned: 3 fails
FIX=1 npx tsx spike2.test.ts    # with the fix: all pass
npx tsx spike3.test.ts
```

## Headless-browser checks (`ui-*.ts`)

Each script drives the running app (`make dev`) and saves screenshots in `shots/`. They aren't part of `make test`.

```sh
CHROME_PATH=<chrome-headless-shell> npx tsx spikes/ui-features.ts
```

- `ui-editor.ts`, `ui-preview.ts`: the redesign demo.
- `ui-hq.ts`: opens HJE Full HQ.
- `ui-features.ts`, `ui-features2.ts`: the plan-features.md features.
- `cleanup.ts`: deletes the test questionnaires the scripts create.
