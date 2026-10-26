# S40 test report 4 (retest after report 3's fix)

**Summary: 14 passed, 0 failed, 0 blocked.**
- **Report 3's F-1 is fixed.** Only one rules change saves at a time. While a change is saving, every control that saves is disabled:
  - an open rule form's **Save rule**;
  - **Add rule**, ↑ ↓, the pencils and the bins;
  - the no-grade tick;
  - the confirm **Delete** and **Keep it**;
  - **Save as test case**, and its Enter.

  Once the first save lands, the next one saves on the new revision with no false "Someone else changed these rules".
- **F1 still holds.** A page left open on an old version gets the right message, and the newer draft isn't touched:
  - "Only draft versions can be changed." once the version is published;
  - "This version has changed since you opened it. Reload and try again." once a newer version exists.
- **A quick pass of the Rules page is clean:** T-04, T-07, T-15, T-16 and T-18.

**Environment**
- Commit: `d602717`. The working tree is not clean (44 paths): S40's uncommitted build plus the fixes, as expected before verification.
- Run: 26 Oct 2026, 22:40–22:48 GMT, at http://localhost:3000 (`make dev`, run by the conductor).
- Browser: Chrome through Claude in Chrome, with a 1500×707 viewport.
- Users: Alex Author, and Sue Super for T-07. I signed back in as Alex at the end.
- Test HQ: **MT S40 2225** (`c725ed2f-60a1-46f2-80c0-e1f1dc01a61b`), from report 3. During this run I published v3, and it now has **v4** as a draft.
- To slow the network, I replaced `window.fetch` in the page with a wrapper that holds `PUT …/rules` for 3–8 s. This was JavaScript in the page only, no code was changed, and a reload removed it.

## Results

| Case | Result | Note |
|---|---|---|
| **F-1** A filled new-rule form while the setting saves | Pass | I filled **Add rule** as "MT race 4" with the task "MT race task", then unticked the setting with its PUT held for 8 s. During the hold, the form's **Save rule** was `disabled`, as were the tick, every ↑ ↓, pencil and bin, and the case's bin. Add rule was hidden (the form was open). Once the PUT landed (200), everything was enabled again. Save rule then saved (200). After a reload both were kept: the setting off, and "3. MT race 4 … add the task “MT race task” for the nurse team". GIF `S40-R4-one-save-at-a-time.gif`. |
| **F-1** Save rule locks the page | Pass | With Save rule held for 3 s, the tick, all row controls, the case bin and Save rule itself were disabled. They were enabled again afterwards. |
| **F-1** A held delete | Pass | I binned MT race 4 and pressed **Delete** with the PUT held for 8 s. **Add rule**, the tick, every row control, the confirm **Delete** and **Keep it**, and the case bin were all disabled. (Report 3 had Add rule and Keep it enabled here.) The delete landed (200) and the rule was gone. |
| **F-1** An open Edit form while a reorder saves | Pass | I opened Edit on Low risk and renamed it "Low risk MT4". With the PUT held for 6 s, I pressed MT Second's ↓. During the hold, the form's Save rule, Add rule, the tick and the row controls were disabled. After it landed, Save rule worked (200). After a reload: "1. Low risk MT4", "2. MT Second 😀". Nobody else changed anything, and no false 409 appeared. |
| **F-1** Save as test case while saving | Pass | On Test the whole HQ I saved "MT busy 4" with Enter, with the PUT held for 5 s. During the hold the button was `disabled`, and a second submit sent no PUT (one PUT in total). The case then gave "“MT busy 4” passes", and the address got `?case=…`. |
| **F-1** A click on a force-enabled button | Pass (note) | During a held save I removed `disabled` from ↓ in the DOM and clicked it. No second PUT went out. React ignores clicks on a button whose prop is disabled, so this doesn't reach `oneAtATime`'s own refusal. That refusal is covered by `oneAtATime.test.ts`. |
| **F1** A stale Rules page on a version published since | Pass | The Rules tab was open on V3 (draft). In another tab I published v3. The stale ↓ gave "Only draft versions can be changed.", and the order was unchanged. |
| **F1** A stale Rules page after a new version | Pass | I created v4 in the other tab. In the stale V3 tab, unticking the setting gave "This version has changed since you opened it. Reload and try again.". The tick went back to on, and the controls were free again afterwards. After a reload: "V4 · DRAFT", the setting on, both rules in order, and both cases passing. |
| **F1** A stale Test the whole HQ page | Pass | The test page was opened on v3. Saving "MT stale 4" with Enter after v4 existed gave the same message (PUT 409). v4 has no "MT stale 4". |
| T-02 / T-04 Edit, then a refused rule | Pass | Saving "Bad" with no actions gave "Rule “Bad”: add at least one action.". The form stayed open, the list was unchanged, and the controls were still usable. Cancel left no Bad. Renaming "Low risk MT4" back to "Low risk" saved, and the two cases went from "2 of 2 failed" back to "All 2 passed". |
| T-05 Reorder | Pass | ↓ and ↑ held after a reload. The first rule's ↑ and the last rule's ↓ are disabled. |
| T-06 / T-18 The setting, and two tabs | Pass | Tab 1 unticked the setting. Tab 2, not reloaded, pressed ↓ and got "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved.". After reloading tab 2: the setting off ("Off: they're never auto-triaged; a clinician reviews them."), and the order unchanged. I ticked it again in tab 1. |
| T-07 Sue Super is read-only | Pass | Sue sees "V4 · DRAFT", **READ ONLY**, and "Read only: only super clinicians and admins of the whole trust can change the trust's rules.". The only control is the disabled tick: no Add rule, arrows, pencils or bins, and no test case bins. Test the whole HQ opens and shows the read-only line instead of Save as test case. |
| T-15 / T-16 The Publish window, publish, new version | Pass | The window for v3 reads "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I.". After publishing: "V3 · PUBLISHED", READ ONLY. Create new version gave V4 with the same rules, setting and cases. |

## Failures in detail

None.

## Found while exploring

- **Add rule while an Edit form is open drops the unsaved edit without asking (minor UX, not a regression).** I opened Edit on "Low risk MT4", typed a new name, then clicked **Add rule**, which is enabled while an Edit form is open. The edit form closed and a blank new-rule form opened. The typed rename was lost silently. Only one form shows at a time, so this may be intended, but a user could lose an edit. Suggestion: hide or disable Add rule while a form is open, or ask first.
- **Enter in the rule form's Name box doesn't save.** The plan promises Enter only for the test case name, so this isn't a failure. I'm noting it because Enter does save on the test page.
- **Cancel stays enabled while Save rule is saving.** Pressing it would close the form while the save still lands, which is harmless. It's the same as report 3's note on Keep it, which is now disabled.
- **Still there from report 3 (cosmetic, staff only):** an empty Question Set on Test the whole HQ shows SurveyJS's own "The survey doesn't contain any visible elements.".

## Console and network errors

- No console errors or warnings in any tab.
- The only failed requests were the ones I caused on purpose:
  - the stale-version 409s on the Rules and test pages;
  - the published version's refusal;
  - the two-tab 409 in T-18.

## Hand checks for Rahul

None.

## GIFs

- `~/Downloads/S40-R4-one-save-at-a-time.gif`, the main path:
  - a filled rule form while the setting saves, with Save rule disabled;
  - the rule saving once the first save lands;
  - the held delete, with everything locked;
  - Save as test case, and the stale-version messages.

Left behind (made up, local): **MT S40 2225**.
- v1–v3 are published.
- v4 is a draft with the default-like rules "Low risk" and "MT Second 😀", the setting on, and the cases "MT fit" and "MT busy 4" (both passing).
