# S40 test report 2 (retest)

**Summary: 14 passed, 0 failed, 0 blocked.** F-1 from report 1 is fixed. So are the exploring findings E-1 to E-5. Two cosmetic notes are listed below. No hand checks are left: R-05 at phone size was covered by J3 on an emulated iPhone 13 (390×844).

**Environment**
- Commit: `d602717`. The working tree is not clean (41 paths): S40's uncommitted build plus the fixes from report 1, as expected before verification.
- Run: 26 Oct 2026, 21:49–22:05 GMT, at http://localhost:3000 (`make dev`, run by the conductor).
- Browser: Chrome through Claude in Chrome, with a 1500×717 viewport. The window still won't go below 500 px wide.
- Test HQs:
  - **MT S40 2112** (`59e893d3-fd20-449e-8962-a3cb4dd0424a`), from report 1. Its v2 draft had the clinician-only Snore score with Low/High bands, and the Low risk, High risk, Heart code and Snorer rules. In this run I published v2 and created **v3** (a draft).
  - **MT S40 2200** (`1c23065a-d5b6-4d83-af8f-30c6264b08e1`), new and empty, for the defaults.
- Patient episode for R-05: **Noa MTSforty Retest**, Hospital A1, on MT S40 2112 v2.

## Results

| Case | Result | Note |
|---|---|---|
| T-08 Preview: rules that fire (report 1's F-1) | Pass | About you Preview, in the **Patient** view, as Alex:<br>• No answers: Low risk fires.<br>• Smoke Yes: ASA II, and Low risk still fires.<br>• Chest pain Yes (red): "None".<br>• Snore Every night: **Snorer: flag “Sleep apnoea”** fires, although Snore score is Clinical and isn't shown in the Patient view.<br>The Clinician view shows "Snore score: 3 (High)" and Snorer. Your heart, heart failure Yes: ASA III, High risk (all three actions) and Heart code: suggest On hold. GIF `S40-T08-retest.gif`. |
| T-03 wording (E-1) | Pass | Snorer ("at least I") reads "is **I to VI**". A new rule "Hip", "at most VI", reads "When the suggested ASA is I to VI and the procedure is Total hip replacement or Cataract surgery: require the anaesthetist's ASA before Ready for admission". Changed to at most III, it reads "is **I to III**". "At most II" still reads "I or II", and "at least III" still reads "III or more". |
| T-04 / E-5 stale refusal | Pass | Saving Hip with no actions shows "Rule “Hip”: add at least one action.". Pressing **+ Add action** clears the message at once. |
| T-09b Replace a case; Enter saves; the bin asks | Pass | • **Open** "Heart" shows the name box "Heart", the **Replace test case** button and "A test case has this name: saving replaces it with these answers and lines.".<br>• Heart failure No, then Enter in the name box: "“Heart” passes". Back on Rules there is still **one** Heart, with the new lines (No suggested ASA; Low risk).<br>• A new name + Enter added "MT Enter case" (`?case=t_zfdqtw`).<br>• Its bin shows "Delete “MT Enter case”? Delete / Keep it". Keep it changed nothing; Delete removed it, and it stayed gone after a reload.<br>GIF `S40-T09b.gif`. |
| E-2 names are unique (Go) | Pass | Typing " heart " on the test page is recognised as the existing "Heart": the button reads Replace test case. **API check:** a PUT `/rules` with a second case named "heart" gets 400 "Two test cases are called “heart”: give each its own name." The stored set is unchanged. |
| T-01 New HQ defaults | Pass | MT S40 2200 → Rules shows "V1 · DRAFT", the setting ticked, **1. Low risk** and **2. High risk** worded exactly as in the plan, and "No test cases yet." |
| T-09 Save a case (happy path) | Pass | On v3, smoke Yes, chest pain No, snore Never, then "Smoker" + Enter: Suggested ASA II (Smoker 77176002 – Current smoker), Low risk fires, and "“Smoker” passes". The Rules page shows "All 2 passed". |
| T-11 Changed rule fails a case, blocks Publish | Pass | Setting Low risk's target to Ready for POA gives "1 of 1 failed". The Publish window shows "1 problem to fix first · **Rules** · Test case “Heart” fails", with Publish disabled. Setting the target back makes the case pass again. |
| T-15 Publish line | Pass | "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I." With the target at Ready for POA, it read "… → Ready for POA." |
| T-16 Publish; read-only; new version copies | Pass | • Published v2.<br>• Rules: "V2 · PUBLISHED", READ ONLY, "Read only: this version is published. To change its rules, create a new version.", and no buttons.<br>• **Create new version** gives "V3 · DRAFT" with the same 5 rules in order, the setting, and the passing case. |
| T-07 Clinician read-only (spot check) | Pass | As Cara, v3's Rules shows a READ ONLY badge and "Read only: only super clinicians and admins of the whole trust can change the trust's rules.". There are no buttons, and the setting's checkbox is `disabled`. |
| R-01 Per-set test cases | Pass | In v3's About you Preview, **Run all** gives "All 1 passed" (MT S40 set case). |
| R-03 Suggested ASA | Pass | Test the whole HQ with smoke Yes + heart failure Yes gives "Suggested ASA III" (Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction). High risk and Heart code fire. |
| R-04 Editor basics | Pass | Added a Text question to About you and renamed it "MT R04 renamed"; the card updated. Two Cmd+Z undid the rename, then the add. It says "All changes saved". |
| R-05 A patient sends an HQ | Pass | • Cara created the episode on MT S40 2112 v2. The invite was captured at /qa/messages.<br>• The patient confirmed DOB 5/5/1970, filled About you (smoker, no chest pain, snores every night) and Your heart (No), then Send my answers → Send my answers: "Thank you, Noa".<br>• The episode shows **READY FOR REVIEW**. Nothing was auto-triaged, although Low risk matches (rules don't act yet, as expected).<br>• The Chrome run was at desktop width. **Phone size:** I ran the committed `e2e/journeys/test_j3_patient.py`, which runs on an emulated iPhone 13 (390×844, WebKit) and checks there's no sideways scroll, plus `test_j11_triage.py`, against `make dev`: **3 passed**. |

## Failures in detail

None.

## Found while exploring

- **E-7 (cosmetic): a test case's lines put the ASA line first or last depending on the alphabet.**
  - The lines are sorted alphabetically, so "No suggested ASA" comes first ("Heart") and "Suggested ASA II" comes last ("Smoker", after "Rule fires: …").
  - The build notes say the ASA line comes first, then the rules.
  - It's harmless, but the Rules page lists cases inconsistently.
- **E-8 (cosmetic): the Procedure boxes don't take typing.** The sample patient's Procedure box on Test the whole HQ is a plain select, and so is New episode's Health questionnaire. On this local database both lists hold dozens of "E2E …" entries, so finding "Cataract surgery" or "MT S40 2112 v2" means scrolling. The rule form's procedure picker does filter as you type. This is more of a nuisance on long-lived local data than on a real trust's list.
- **Checked and fine:**
  - Test the whole HQ on an HQ with no Question Sets says "This HQ has no Question Set for the patient." and still shows Low risk (no grade counts as ASA I), with no console errors.
  - Choosing Cataract surgery as the sample patient's procedure makes "Hip: require the anaesthetist's ASA…" fire, and the opened case then shows its expected lines in red.
  - A case name typed with spaces and in other letter case (" heart ") is matched to the existing "Heart".
- **Not a finding, just noting it:** once, the first click on **Rules** right after closing the Publish window didn't navigate; the second click did. It's probably the window's closing animation catching the click. It didn't happen again.

## Console and network errors

None. There were no console errors on the Rules page, the test page, the Preview or the patient pages. Every PUT `/rules` made through the UI succeeded. The only 400 was my deliberate duplicate-name API check.

## Hand checks for Rahul

None. R-05 at 390×844 was covered by J3 on an emulated iPhone 13 (see R-05).

## GIFs

- `~/Downloads/S40-T08-retest.gif`: the F-1 retest. In the Patient view, Every night fires Snorer; switching to Clinician shows Snore score 3 (High).
- `~/Downloads/S40-T09b.gif`: the main happy path. Open "Heart" (Replace test case), change an answer, Enter to save, save a new case with Enter, then the bin's Delete / Keep it.
