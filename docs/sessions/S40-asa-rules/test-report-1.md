# S40 test report 1

**Summary: 23 passed, 1 failed, 0 blocked.** The failure is minor. There is 1 hand check for Rahul: phone size for R-05.

**Environment**
- Commit: `d602717`. The working tree is not clean: S40's uncommitted build is in it, as expected before verification.
- Run: 26 Oct 2026, 21:10–21:45 GMT, at http://localhost:3000 (`make dev` run by the conductor).
- Browser: Chrome through Claude in Chrome, with a 1500×717 viewport. That window wouldn't resize, so the narrow-width checks used a second window at 500 px, Chrome's minimum.
- Test HQ: **MT S40 2112** (`59e893d3-fd20-449e-8962-a3cb4dd0424a`), built through the UI as in T-00.
  - v1 is published.
  - v2 is a draft. It now has the Snorer rule again and a High band on Snore score, which I added to reproduce the T-08 failure.
- Patient episode for R-05: **Mia MTSforty Rules**, Hospital A1.

## Results

| Case | Result | Note |
|---|---|---|
| T-00 Build the HQ | Pass | Built as written. The seed has no "General" category, so I used Anaesthetic for "Frail". A new Calculation is **Clinical** (clinician-only) by default, and I left it that way (see F-1). |
| T-01 Default rules in plain words | Pass | URL, title, "V1 · DRAFT", the setting ticked, both default rules worded as in the plan, and "No test cases yet". Reload, the ← link and browser Back all work. |
| T-02 Edit a rule's target | Pass | Ready for POA was kept after a reload. Setting it back restored the default wording, and Cancel changed nothing. |
| T-03 Each kind of "when" | Pass | Snorer, Heart code, Hip and Red were all worded as expected and kept after a reload. The code list offered only Smoker and Heart failure. Hip reads "When the suggested ASA is **at most VI** and the procedure is …" (see E-1). |
| T-04 Refusals in plain words | Pass | All six messages were exact ("add at least one action", "Enter the task.", "name the flag …", "Give each rule a name.", "Two rules are called “low risk” …", "auto-triage only once."). Cancel added nothing. |
| T-05 Reorder and delete | Pass | ↑ moved Snorer to 2 and the order survived a reload. The first ↑ and the last ↓ are disabled. Delete “Hip”? → Keep it changed nothing, then Delete removed it, and it stayed gone after a reload. |
| T-06 No-grade setting | Pass | The "Off: they're never auto-triaged; a clinician reviews them." line appears and survives a reload. |
| T-07 Who can change rules | Pass | Sue and Cara both get Read only: the badge and the line, no buttons, and the tick disabled. The test page says "Read only…" in place of Save. Bea gets "You don't have access to this hospital", and with her own trust/hospital in the URL, "Questionnaire not found." on both `/rules` and `/rules/test`. I also checked the API: Cara's PUT `/rules` gets 403, and Bea's PUT gets 404 through either URL. |
| T-08 Preview: rules that fire | **Fail** (Minor) | Steps 2–4 and 6 pass. In step 5, in the **Patient** view the plan asks for, the Preview shows no Snore score and Snorer doesn't fire. In the Clinician view and in Test the whole HQ, Snorer does fire. See F-1. |
| T-09 Test the whole HQ, save a case | Pass | Only the About you and Your heart tabs, plus Sam Sample, 54, Female, and Procedure None. Answers are kept across tabs. "“Smoker” passes", `?case=…` added, "All 1 passed", lines listed. Heart: ASA III with High risk and Heart code. Both cases still pass after a reload. |
| T-10 Open a saved case | Pass | Answers and "“Smoker” passes" came back. Clear answers removed `?case=` and gave "No suggested ASA". |
| T-11 Changed rule fails a case, blocks Publish | Pass | "1 of 2 failed". Clicking the name shows "− expected, not produced: Rule fires: Low risk …". Publish lists "Rules · Test case “Smoker” fails" and is disabled. The case passes again after reverting, fails when the target is Ready for POA, and passes when reverted. The bin removed it. |
| T-12 Rule that can't fire | Pass | Removing the High band gave "These rules can't fire, and block Publish · Rule “Snorer”: a band it tests isn't in “Snore score” any more". The words show the band ID `b_kwij`. Publish lists the problem and is disabled. Deleting Snorer clears the warning. |
| T-13 No-grade setting in the test page | Pass | Off: "No suggested ASA" with "Low risk: no auto-triage: no ASA grade" and "Red: add the task …; no auto-triage: no ASA grade". On: back to auto-triage to Ready for admission. |
| T-14 Clinician sets don't count | Pass | There is no Nurse checks tab on the test page. |
| T-15 Publish window line | Pass | On: "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I." Off: "… Patients with no ASA grade are never auto-triaged." With no auto-triage action: "This version doesn't auto-triage anyone." With Red present, the "; ASA I–VI, no red flags → POA complete" part was added. |
| T-16 Published is read-only; new version copies | Pass | "V1 · PUBLISHED", Read only with the published-version line, no buttons, and cases listed. The test page works but shows Read only in place of Save. The API PUT gets 409 "Only draft versions can be changed.". Create new version gives "V2 · DRAFT" with the same 3 rules in order, the setting, and the passing case, all editable. |
| T-17 Seeded HQ has default rules | Pass (see note) | Trust A's Demo pre-op HQ on this local database is at **v5**, made by hand on 23 Oct, before S40. It correctly says "This version was published before HQs had rules, so it has none. A new version gets the default rules." Its seeded v1 (retired) and Trust B's seeded Demo HQ v1 (published) both have Low risk + High risk with the setting on (checked in the database). On a fresh seed this case would show the defaults. |
| T-18 Two tabs: stale save refused | Pass | Tab 2 showed "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved." After a reload it shows tab 1's setting change and Low risk unchanged. |
| T-19 Deep links | Pass | `/rules/test` opens directly. `?case=t_nope123` opens with no case loaded and no error (no console errors either). |
| R-01 Per-set test cases | Pass | Saved "MT S40 set case" in v2's About you Preview, and Run all gives "All 1 passed". |
| R-02 Publish / Create new version | Pass | Covered by T-16: v1 published and v2 created. |
| R-03 Suggested ASA | Pass | Smoker plus heart failure gives "Suggested ASA III" with its reason. A clinician set's display condition offers "Suggested ASA" (I set it back to Always afterwards). |
| R-04 Editor basics | Pass | Added a Text question, renamed it, and undid both; it says "All changes saved". |
| R-05 Patient sends an HQ | Pass | Cara created the episode on MT S40 2112 v1. The invite was captured at /qa/messages. The patient confirmed her date of birth, finished both sets and sent. The episode shows **Ready for review**, with nothing auto-triaged. The patient pages ran at desktop width (the window couldn't go to 390), and the "Thank you" page looked fine at 500 px. **Hand check for Rahul:** the patient journey at 390×844. |

## Failures in detail

### F-1 (Minor): the Preview's "Rules that fire" in the Patient view ignores clinician-only scores, so it disagrees with the whole-HQ test

**Steps:**
1. Sign in as Alex. Open MT S40 2112 (v2 draft) → About you → **Preview**, which opens in the **Patient** view.
2. Answer How often do you snore? **Every night**. Snore score is a Calculation whose **Clinical** switch is on, the editor's default for a new Calculation, with bands Low ≤ 2 and High above.
3. Look at Clinical outputs → Rules that fire.

**Expected:** this is T-08 step 5, in the Patient view the plan asks for: Snore score "3 (High)" and "Snorer: flag “Sleep apnoea”".

**Actual:**
- In the Patient view there is no Scores line, and Rules that fire lists only Low risk; Snorer is missing.
- In the **Clinician** view of the same Preview, "Snore score: 3 (High)" appears and Snorer fires.
- Test the whole HQ with the same answers also fires Snorer.

The build notes say the rules use the clinician's view, so clinician-only scores count. The whole-HQ test does that, but the per-set Preview works out "Rules that fire" from whichever Patient/Clinician view is selected. An author checking in the default Patient view is told a rule won't fire when it will.

**Details:**
- URL: `/o/…fa/h/…0a/chapters/c8f52f4d-1fc3-4278-849d-2efcc76eca78/preview`, signed in as Alex Author.
- No console errors and no failed requests.
- GIF: `~/Downloads/S40-T08.gif`.

## Found while exploring

- **E-1 (cosmetic): uneven ASA wording.** "is at most II" reads "is I or II", but "is at most VI" reads "is at most VI" (T-03 Hip). Matching the other forms ("I to VI", or no ASA clause at all) would read more evenly.
- **E-2 (minor): a failing whole-HQ case can't be updated in place.**
  - Opening a failing case and pressing Save as test case with the same name adds a **second** case called "Heart" (new id `t_qtymof`), and the failing one stays.
  - Rule names must be unique, but case names aren't.
  - The only fix is to delete the old one by hand.
- **E-3 (cosmetic): only one bin confirms.** The bin on a whole-HQ test case deletes at once, with no "Delete …? / Keep it" step, while the rule bin asks first.
- **E-4 (cosmetic): Enter doesn't save a test case.** Pressing Enter in the test-case name box doesn't save it; you have to click Save as test case.
- **E-5:** after a failed save, the red error stays in the rule form while you go on editing, until the next Save. It's harmless, just stale.
- **E-6 (probably not S40):** in the editor, a Select One question added to the **Clinician** set "Nurse checks" shows a **PATIENT** badge on its card.
- **Checked and fine:**
  - A rule name with leading/trailing spaces and an emoji is trimmed in the messages ("Rule “🙂 Spacey”: …").
  - A 101-character flag name is refused with the plain-words message.
  - The rules and test pages at 500 px have no horizontal scroll, and the read-only state reads well.
- **API checks:**
  - PUT `/rules`: a clinician gets 403, another trust's user 404, and a published version 409.
  - A stale revision is refused (seen in the T-18 UI).

## Console and network errors

None outside the cases. There were no console errors on the Rules page, the test page, the Preview or the patient pages, and every PUT `/rules` I made as Alex returned 200.

## Hand checks for Rahul

- R-05 at true phone size, 390×844: the Chrome window here couldn't go below 500 px wide.

## GIFs

- `~/Downloads/S40-T09.gif`: the main happy path. Rules page → Test the whole HQ → answer → save "Smoker" → back to Rules, "All 1 passed".
- `~/Downloads/S40-T08.gif`: F-1. In the Patient view, Every night gives no Snorer; in the Clinician view, Snore score 3 (High) and Snorer fires.
