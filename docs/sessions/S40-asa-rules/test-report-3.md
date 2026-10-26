# S40 test report 3 (retest after verification 1)

**Summary: 13 passed, 1 failed, 0 blocked.**
- **F1 is fixed.** A Rules page or Test the whole HQ page left open on an old version now gets "This version has changed since you opened it. Reload and try again.", and the newer draft isn't touched.
- **F2 is fixed as described.** While Save rule is saving, ↑ ↓, the pencils, the bins, the no-grade tick and the confirm Delete are all disabled.
- **The failure (Minor) is the other direction of the same race.** While some other change is saving, an open rule form's **Save rule** stays enabled. Clicking it sends a second save with the old revision, which gets the false 409 "Someone else changed these rules…" and loses the rule.

**Environment**
- Commit: `d602717`. The working tree is not clean (43 paths): S40's uncommitted build plus the fixes from verification 1, as expected before verification.
- Run: 26 Oct 2026, 22:23–22:35 GMT, at http://localhost:3000 (`make dev`, run by the conductor).
- Browser: Chrome through Claude in Chrome, with a 1500×707 viewport. I was signed in as Alex Author throughout.
- Test HQs:
  - **MT S40 2225** (`c725ed2f-60a1-46f2-80c0-e1f1dc01a61b`), new in this run, with one empty patient Question Set. I published v1 and v2 during the run, and it now has **v3** as a draft.
  - **MT S40 2112** (`59e893d3-…`) v3, from the earlier reports.
- To slow the network for F2, I replaced `window.fetch` in the page with a wrapper that holds `PUT …/rules` for 6–25 s (JavaScript in the page, no code changed). A reload removed it.

## Results

| Case | Result | Note |
|---|---|---|
| **F1** Stale Rules page after publish and new version | Pass | Tab A opened MT S40 2225's Rules page on "V1 · DRAFT". In tab B I published v1 and made v2. In tab A, editing Low risk's target to Ready for POA and pressing Save rule gave "This version has changed since you opened it. Reload and try again." (PUT 409). The stale ↓ gave the same message, and the list stayed in its old order. After a reload: "V2 · DRAFT", the default rules unchanged, rules revision 0. GIF `S40-F1-stale-version.gif`. |
| **F1** Stale Test the whole HQ page | Pass | The test page opened on v1. Saving "MT stale case" with Enter after v2 existed gave the same message, and no case was added to v2. |
| **F1** Stale page on a version published since it was opened | Pass | Tab A stayed open on v2 (draft) while I published v2. ↓ then gave "Only draft versions can be changed.". After v3 was made, the same click gave "This version has changed since you opened it…". v3 kept its order, its setting and its one case. |
| **F1** API checks (confirming) | Pass | A PUT with no `versionId` got **400** "The rules could not be read.". A PUT with another version's id got **409** with the plain words. The revision was unchanged after both. |
| **F2** Save rule disables the other controls | Pass | Saving High risk was held for 8 s. During the save, the setting's checkbox, Move up/down, Edit and Delete on the other rule were `disabled`, and Save rule showed a spinner. Everything was enabled again afterwards, and the save went through. |
| **F2** The confirm Delete while saving | Pass | During a held delete of High risk, **Delete** and every row control were disabled. "Keep it" stayed enabled, which is harmless. |
| **F2** Save rule while another change is saving | **Fail (Minor)** | See F-1 below. |
| T-01 New HQ defaults | Pass | MT S40 2225 → Rules shows "V1 · DRAFT", the setting ticked, and **1. Low risk** / **2. High risk** in the plan's words. It says "No test cases yet." |
| T-02 / T-11 Edit a rule, a case fails, set it back | Pass | On MT S40 2112 v3, Low risk set to "at most I" reads "is I and there are no red flags" and gives "2 of 3 failed". Setting it back to II gives "All 3 passed" after a reload. |
| T-03 (quick) New rule with an emoji name | Pass | "MT Second 😀" was saved: "When the suggested ASA is III or more: add the task “Phone the patient” for the nurse team". |
| T-05 Reorder | Pass | ↑ on rule 2 made MT Second rule 1, and the order held after a reload. The first rule's ↑ and the last rule's ↓ are disabled. |
| T-06 The no-grade setting | Pass | Unticked, it shows "Off: they're never auto-triaged; a clinician reviews them.", and that held after a reload. Ticked again, the line goes. |
| T-18 Two tabs | Pass | Tab 1 unticked the setting. Tab 2, not reloaded, saved a Low risk edit and got "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved." After reloading tab 2: tab 1's setting, and Low risk unchanged. |
| T-09 / T-16 / T-19 / R-02 Save a case, publish, new version, deep link | Pass | • "MT fit" saved with Enter, and the address got `?case=t_hcwqzo`. Rules showed "All 1 passed".<br>• Published v2: "V2 · PUBLISHED", READ ONLY, the read-only words, and only a disabled checkbox.<br>• Create new version gave "V3 · DRAFT" with both rules in order, the setting and the passing case.<br>• `…/rules/test?case=t_nope99` opens with no case loaded and no console error. T-17's Demo HQ is at v5 on this database (see report 1's note); it shows "published before HQs had rules". |

## Failures in detail

### F-1 (Minor): an open rule form's Save rule isn't disabled while another change is saving, so it can still cause a false "Someone else changed these rules"

The F2 fix makes Save rule set `busy`, so the row controls lock while a rule saves. The reverse isn't covered: the rule form's **Save rule** (`RuleForm.tsx`, `loading={saving}`, its own state) ignores the page's `busy`. The same goes for **Add rule**, which stays enabled during a save and opens a form whose Save rule is enabled. A second PUT then goes out with the revision from before the first save, and gets the 409 that verification 1 described for F2.

- **Signed in as:** Alex Author.
- **URL:** `/o/…fa/h/…0a/questionnaires/c725ed2f-60a1-46f2-80c0-e1f1dc01a61b/rules` (MT S40 2225 v2 draft, and again on v3).
- **Steps:**
  1. **Add rule**, name "MT race", **+ Add action** → Add a task "MT race task" (don't save yet).
  2. With a slow network (I held `PUT …/rules` for 6 s in the page), untick the no-grade setting.
  3. While that save is in flight, press the form's **Save rule**. It is enabled and shows a spinner.
- **Expected:** Save rule is disabled (or waits) while the other change saves. Once it lands, the rule saves on the new revision.
- **Actual:** after the delay, the form shows "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved." After a reload, the setting is off (the first save landed) and "MT race" doesn't exist (lost). Nobody else changed anything.
- **The same in the other order:** during a held delete of High risk, Add rule opened a new form whose Save rule was enabled.
- **How likely:** it needs a click on Save rule within one save's round trip. Locally that's tens of ms. On AWS (a Lambda cold start, a phone network) it's a second or more.
- **Suggested fix:** pass `busy` to `RuleForm` and disable its Save rule (and Add rule) while `busy`. Or chain saves so each uses the latest revision.
- **Console/network:** none apart from the expected 409 on the second PUT.
- **GIF:** `S40-F2-open-form-race.gif`.

## Found while exploring

- **The test page for an empty Question Set shows SurveyJS's own text (cosmetic, staff only).** On Test the whole HQ, a Question Set with no questions shows "The survey doesn't contain any visible elements.". That's the library's default wording, not ours. The Preview probably does the same. It's only seen by authors.
- **Keep it stays enabled during a held delete.** Pressing it would hide the confirmation while the delete still goes through. This is harmless, and the list updates when the save lands.
- **Checked and fine:**
  - A stale tab's refused save leaves its list as it was. It doesn't show the change as if it had saved.
  - An emoji in a rule name saves, shows, and appears in the buttons' aria-labels ("Move MT Second 😀 down").
  - Both F1 messages are in plain words.

## Console and network errors

- No console errors in any of the three tabs.
- The only failed requests were the expected ones: the 409s (stale version, stale revision, the F-1 race), the 400 for a missing `versionId`, and the published version's refusal. All of them were caused on purpose.

## Hand checks for Rahul

None.

## GIFs

- `~/Downloads/S40-F1-stale-version.gif`: the F1 retest, the main path. A Rules page open on V1 gets "This version has changed since you opened it. Reload and try again."; after a reload it shows V2 with the defaults unchanged.
- `~/Downloads/S40-F2-open-form-race.gif`: F-1. A filled rule form, the setting unticked on a slow network, then Save rule, then the false "Someone else changed these rules…".

Left behind (made up, local): the HQ **MT S40 2225** (v1 and v2 published, v3 a draft holding "MT Second 😀", Low risk and the case "MT fit"). MT S40 2112 v3 is as I found it.
