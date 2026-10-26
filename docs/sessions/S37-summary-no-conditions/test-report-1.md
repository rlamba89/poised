# S37 test report 1: no conditions on summary boxes

**17 passed, 0 failed, 0 blocked** (13 cases + 4 regression).

- Environment: HEAD `389cbe5`, working tree not clean (S37's uncommitted changes plus `docs/market/`), 26 Oct 2026 10:31–10:50 GMT, http://localhost:3000 (the conductor's `make dev`), Chrome ("rahul chrome" profile). Automatic mode.
- Test data: questionnaire **"S37 Summary HQ 1031"** (Trust A, Hospital A1): v1 retired, v2 published, v3 draft (left as a draft, with a summary that has a condition).
- **How I gave a summary a condition:** Setup 3's Console snippet, run through the extension's JavaScript tool. It's the same code, except it **returns** its message instead of calling `alert()`, because automatic mode mustn't open browser dialogs. Same API call (`PUT …/chapters/{id}/content`), same `{patientAge} >= 18`. I reloaded the page after each run, as the plan says.
- **Other checks I made outside the UI:**
  - I read the chapter JSON through the API to confirm that a removed condition is gone from the saved draft (T-09), and to find a question ID for R-03.
  - One `psql` query confirmed the published versions were never rewritten: v1 and v2 have no condition; only the v3 draft does.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 A new summary has no Logic tab | Pass | **Settings** is the only tab. The panel shows the Title and the "Only clinicians see this box…" help, with no warning. The card says "Covers the page “Heart”" and has no "Displays when" box. |
| T-02 A summary inside a Group | Pass | "Breathing summary": same as T-01. The card says "Covers the Group “Breathing”". |
| T-03 Other elements still offer Logic | Pass | The question shows Settings \| Disclosures \| Logic (Builder: Display Always / Conditionally). The Group shows Settings \| Logic. Clicking the summary while on a Logic tab opens it on Settings; going back to the question shows Logic again. No console errors. |
| T-04 Give a question a condition | Pass | "Displays when Do you get chest pain? is Yes" appears on the asthma card. The summaries are unaffected. |
| T-05 A draft with no summary condition publishes | Pass | After deleting the Breathing summary, the window says "Patients can then be given this version…", with no problems listed. |
| T-06 Publish names the summary | Pass | The card shows "Displays when Patient's age ≥ 18". The window says "1 problem to fix first": "**About you** · The clinical summary “Heart summary” has a condition: remove it (a summary is always shown)". Publish is disabled. |
| T-07 The summary's Settings say why | Pass | The orange warning text matches the plan word for word, and a **Logic** tab is offered. |
| T-08 The name follows a typed title | Pass | The problem now names “Heart checks”. |
| T-09 Remove it in the Code tab | Pass | "Visible if" held `{patientAge} >= 18`. After emptying it and pressing Save logic, the Logic tab, the warning and the "Displays when" box are all gone, and Publish lists no problems. The API confirms the saved summary has no `visibleIf`. |
| T-10 Remove it with Display: Always | Pass | Clicking the "Displays when" box opens the summary's Logic tab. After choosing Always, the result is the same as T-09. |
| T-11 Or delete the summary | Pass | The problem is listed. After deleting the summary, none are listed, and v1 publishes and opens read-only. |
| T-12 Reload and back button | Pass | v2 had a summary with a condition. The problem showed after a reload, after going to the list and Back/Forward, and after reopening the questionnaire from the list. It also showed when I pressed Publish before opening any Question Set. Once I removed the condition in Code, v2 published. |
| T-13 A read-only viewer (Sue Super) | Pass | On v3, Sue sees a "READ ONLY" editor with no Publish button. The card shows "Displays when". The Logic tab is offered, with the Builder and Code fields disabled. There's **no** orange warning. |
| R-01 Two summaries in one Group | Pass | The window lists "Two clinical summaries cover the same Group “Breathing”: keep one" next to the condition problem. |
| R-02 Page Logic | Pass | Heart's menu → Logic opens "Page logic · Heart", with Display and Skip rules. |
| R-03 Logic problems button | Pass | I gave "Do you get chest pain?" the condition `{q_zuofqo} notempty` in the Code tab, since the Builder offers only earlier questions. The "1 logic problem" button appeared. Clicking the problem opened that question's Logic tab. I then removed the condition (Always). |
| R-04 Video and Information offer Logic | Pass | Both show Settings \| Logic, with Display Always / Conditionally. A Section does too (checked while exploring). |

## Failures

None.

## Found while exploring

All of these are outside S37's cases. None is a failure.

1. **Required-if and enable-if conditions from the API (it works).**
   - I gave the Breathing summary `requiredIf` and `enableIf` through the API, and gave the page summary a `visibleIf` of three spaces.
   - Publish listed the Breathing summary's problem. The card showed "Required when…" and "Read-only when…".
   - The whitespace-only `visibleIf` was ignored: no problem listed and no Logic tab, as the build notes say.
   - The Code tab offered Visible if / Required if / Enable if, and each one could be cleared. Once all three were cleared, the Logic tab and the warning went away.
2. **(Minor, existing behaviour, not S37's change.) The Code tab drops an unsaved edit when you switch sub-tab.**
   - Steps: empty "Required if", switch to "Enable if" without pressing Save logic, empty it there too, then press Save logic.
   - Result: only Enable if was saved. "Required if" still held its condition.
   - Save logic saves one field at a time; S37 didn't change that. With two conditions to remove, the author has to save each one separately. The warning's wording ("Remove the condition in the Logic tab…") doesn't mention this.
3. **(Minor) Display: Always does nothing for a summary whose only conditions are required-if or enable-if.**
   - The warning points to "Display: Always, or the Code tab". But in that case the Builder shows only Display, and it's already Always.
   - Only the Code tab can remove these conditions. This can only happen with JSON made outside the editor.
4. **(Cosmetic) Two summaries with the same title and a condition give one problem line.**
   - Copying a summary that has a condition copies the condition too.
   - The Publish window then shows a single "The clinical summary “Breathing summary” has a condition" line, because identical messages are de-duplicated, plus the two-summaries problem.
   - Once one is fixed, the other is still listed, so nothing is missed.
5. **(Cosmetic, existing behaviour) A new summary's delete window doesn't use its title.**
   - It says "“Clinical summary” will be deleted, with its options and disclosures." For a summary whose title was typed, it says "“Heart checks”…".
   - Summaries have no options or disclosures, so the second half of the sentence doesn't fit either.
6. **Undo works with this change.**
   - Undo after removing a condition brings back the condition, the "Displays when" box, the warning and the Logic tab.
   - Undo after copying a summary removes the copy.
7. **Console and network were clean.** The patient page heading reads "Clinical Page" on a Patient Question Set. That's existing wording, noted only.

## Console and network errors

None. I read the console with the pattern `error|Error|warn` and found nothing. The `/api/` requests on a reload of the editor and while selecting elements all returned 200.

## GIFs

- `/Users/rahul/Downloads/S37-happy-path.gif`: the main happy path, from T-01 to T-09. There were no failures, so there are no failure GIFs.

## Hand checks for Rahul

None.

## Left behind

- "S37 Summary HQ 1031" v3 is still a draft. It has a Breathing summary with a condition, and an empty Section, Video and Information block.
- My Console helper was removed from the browser's localStorage.
