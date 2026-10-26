# S39 test report 1

**Summary: 22 cases passed, 0 failed, 0 blocked. Exploring found 2 failures to fix (both Minor) and 3 cosmetic notes.**

- **Code:** `c1f6c69`, with S39's uncommitted changes in the working tree (41 changed or new files).
- **When:** 26 Oct 2026, 17:50–18:25.
- **Where:** http://localhost:3000 (`make dev`, run by the conductor), in Chrome through Claude in Chrome.
- **Window:** the window couldn't be resized (it stayed 2560 px wide). For R-05 I checked the patient pages at 390 px inside a 390×844 iframe on the same origin.
- **Data:** the database still holds data from earlier runs. Everything below was made for this round:
  - HQs (as Alex Author):
    - **MT S39 1750 ASA** (draft): About you, then Anaesthetic plan (clinician).
    - **MT S39 POA 1810** v1 and v2: About you (smoke, surgery today), Your heart, Anaesthetic plan.
    - **MT S39 1900 TrustB** (draft, as Bea Author).
  - Episodes (Cara Clinician, Hospital A1). Each has a made-up `@example.test` email, and the patient answered through the invite link:

    | Episode | Version | Used for |
    | --- | --- | --- |
    | MT S39 1820 Heartt | v1 | T-10, T-11, T-12, T-16, R-03 |
    | MT S39 1830 Nograde | v1 | T-14 |
    | MT S39 1840 Emerg | v1 | T-15, R-02 |
    | MT S39 1850 Hidden | v2 | T-13 step 2 |
    | MT S39 1855 Corrected | v2 | T-13 step 3 |

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | The row for 13645005 says "· ASA III". Clicking it fills in III with the box "ASA III: COPD (Adults)" (selected) and "No ASA grade", and the reason line reads "COPD" (from the ASA guide). The list shows ASA III (COPD), and it's still there after a reload. |
| T-02 | Pass | For E11.9 the grade stays empty, and a pale yellow box lists the 3 examples (2 Adults, 1 Children) and "No ASA grade". Save is refused with "Choose the ASA grade from the ASA guide, or No ASA grade." Choosing III fills it in with the reason, and it saves as ASA III (Poorly controlled diabetes). |
| T-03 | Pass | E11.9 with "No ASA grade" saves with no ASA badge. |
| T-04 | Pass | COPD fills in III. Changing it to IV removes the reason, and back to III brings "COPD" back. Clearing it with ✕ leaves no grade and no reason, and it saves with no badge. For I50.9, picking IV by hand gives no reason and saves as ASA IV. |
| T-05 | Pass | Cough (49727002) has no "· ASA" hint, no guide box and no grade. Editing T-01's disclosure and adding Smoker keeps III; the box offers only "ASA II: Current smoker (Adults)" and "No ASA grade". Choosing Smoker changes the grade to II with the "Current smoker" reason. Cancel kept ASA III (COPD). |
| T-06 | Pass | The window's text is as expected. The COPD option has ASA III (COPD); the diabetes option has no grade. *See F2.* |
| T-07 | Pass | Sue Super sees ASA III (COPD) and the other disclosure, with only Done and ✕ (no Add, Edit or Remove). |
| T-08 | Pass | The Clinician preview with Yes shows "Suggested ASA / ASA III / Chronic obstructive lung disease (13645005) – COPD". A test case saved as "MT COPD yes" passes (All 1 passed). |
| T-09 | Pass | The clinician set offers Suggested ASA under "Patient and viewer", with "is at least / is at most / is" and I–VI. The note under the condition is shown. The condition is kept after a reload. Patient sets (both the first one and the second, "Your heart" in v2) don't offer it. Switching to Patient shows "1 logic problem" in the header ("This Question Set: only a clinician's Question Set can test the suggested ASA"), and the Publish window shows "Anaesthetic plan · Question Set condition: only a clinician's…" with Publish disabled. Switching back to Clinician clears it. |
| T-10 | Pass | The episode page says "**Suggested ASA III**: Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction (the anaesthetist confirms)", and Anaesthetic plan is listed. The POA Summary's Validated tab shows Suggested ASA III, one reason (not smoking), the "Suggested from the answers…" line, then the nurse's and anaesthetist's boxes. The Patient answers tab shows the same without the boxes. |
| T-11 | Pass | The reason link opens Validate for "Your heart" with the question highlighted in yellow and a "POA Summary" link back. That link returns to `?tab=patient`. |
| T-12 | Pass | After changing heart failure to No: the validated tab says "Suggested ASA II · Smoker (77176002) – Current smoker", the patient tab still says III, the episode page line says II, and Anaesthetic plan is gone. Changing it back to Yes brings back III and the plan set. |
| T-13 | Pass | v2 has Your heart shown only when smoke is Yes. With Smoke No, the episode page and both POA tabs say "No suggested ASA: no answer gives an ASA grade". On the second episode, changing smoke to No on Validate hides Your heart; the validated tab says No suggested ASA and the patient tab still says III. |
| T-14 | Pass | "No suggested ASA…" shows on the episode page and both POA tabs. The anaesthetist grade II saved and was kept after a reload; the General notes say "Anaesthetist ASA grade set to 2". |
| T-15 | Pass | The POA Summary says "Suggested ASA IIE" with "Smoker (77176002) – Current smoker" and "Needs surgery today (emergency)". The episode page line matches. The About you preview with both answers shows ASA IIE and both lines. |
| T-16 | Pass | `…/poa?tab=patient` reloads on that tab with its suggestion. The episode address opened in another tab shows the line. |
| T-17 | Pass | Bea (Trust B) gets the same guide choice for 13645005 (ASA III: COPD (Adults), filled in). Trust A's episode address shows "You don't have access to this hospital", and the API returns `404 {"error":"Hospital not found."}`. |
| R-01 | Pass | The S38 episode "Sam MTfive S38a" (Scores HQ v1): Validate Neck shows "Sleep risk 4 (High)" with "High risk of sleep apnoea", and the POA Summary lists it under Neck. This matches S38 report 5. |
| R-02 | Pass | The "Needs surgery today" note on the validated tab opens Validate About you with that question highlighted. Back returns to the POA Summary. |
| R-03 | Pass | Nurse ASA III saved and was kept after a reload; the General notes say "Nurse ASA grade set to 3". |
| R-04 | Pass | "MT COPD yes" (Suggested ASA III) passes. With a temporary extra ASA IV disclosure, the Publish window listed "Test case 'MT COPD yes' fails" and Publish was disabled. I then cleared that grade. I didn't run a test case saved before S39. |
| R-05 | Pass (with a caveat) | The patient's set list (desktop width) shows only About you and Your heart, never Anaesthetic plan, and nothing about ASA. At 390 px (iframe), `/home` and the sent page have no horizontal scroll (scrollWidth 388 = clientWidth) and no ASA text. I couldn't resize the window, so the unsent set list wasn't seen at 390 px. |

## Failures to fix (found while exploring)

### F1: Removing the code that set the grade leaves its grade and reason (Minor: wrong reason shown to clinicians)

- **Signed in as:** Alex Author.
- **URL:** `/questionnaires/87b900a7-f42f-4eff-99f8-482fa932bc77`, draft MT S39 1750 ASA, Select Many question → Statement 1.
- **Steps:**
  1. Add a disclosure.
  2. Add SNOMED 13645005 (it fills in III with "COPD").
  3. Add SNOMED 77176002 (the grade stays III and the box adds "ASA II: Current smoker").
  4. Remove the COPD code with its ✕.
  5. Save output.
- **Expected:** the COPD grade and its reason go with the COPD code. At least the "COPD" reason shouldn't stay on a disclosure that no longer has the COPD code. The author should then be asked again (Smoker's example, or No ASA grade).
- **Actual:**
  - After removing COPD, the grade stays **III**, no radio is selected, and the line still says "Shown with the suggested ASA as: 'COPD' (from the ASA guide)".
  - It saves as **SMOKER · ASA III (COPD)**.
  - The Clinician preview then shows "Suggested ASA / ASA III / **Smoker (77176002) – COPD**". The POA Summary and the episode page would show the same wrong reason.
- **Console and network:** no errors.
- **GIF:** `~/Downloads/S39-F1-stale-reason.gif`.
- **Data note:** I left this wrong disclosure on Statement 1 of the draft, so the builder can see it.

### F2: "From the code library" says to choose a severity-dependent grade in the option's disclosure, but that disclosure offers no ASA guide choice (Minor)

- **Signed in as:** Alex Author. Same draft, Select Many question.
- **Steps:**
  1. Settings → **From the code library**.
  2. Add 13645005 and E11.9, then **Add 2 options**.
  3. Disclosures → the "Type 2 diabetes mellitus…" option → **Edit** its disclosure.
- **Expected:** the window said "where the grade depends on how severe it is, choose it in the option's disclosure". So the option's disclosure should offer the guide's diabetes examples (II Well-controlled / III Poorly controlled / II Children / No ASA grade), and picking one should give its reason.
- **Actual:**
  - The Edit window has no "ASA from the ASA guide" box, because the code was added before this edit (T-05's rule).
  - Picking III by hand gives no reason line, even though III has exactly one example for E11.9 ("Poorly controlled diabetes").
  - The only way to get the guide's choice and words is to remove the code and add it again.
- **Console and network:** no errors.

## Cosmetic (not counted)

- **C1 (T-02):** after "Choose the ASA grade from the ASA guide, or No ASA grade." appears, choosing a grade doesn't clear the red message. It stays until Save.
- **C2:** the episode page says "**No suggested ASA**: no answer gives an ASA grade **(the anaesthetist confirms)**". The anaesthetist has nothing to confirm when there's no suggestion. The POA Summary leaves that phrase out.
- **C3 (T-09 step 5):** when a set with a Suggested ASA condition is switched to Patient, the condition row shows an empty "Select a question…". The author can't see what the problem condition tests until they switch back.

## Other observations (likely as before S39, not counted)

- When an answer's disclosures have codes but no note, the POA Summary body says "No answers produce anything for the summary yet.", while the ASA section above it lists that answer's code as the reason. This is consistent with earlier sessions (the summary lists notes), but it reads oddly next to the new reasons.
- The editor shows the "Patients see" setting on a clinician set, and the question card says "PATIENT" inside a clinician set.

## Console and network errors

None. No console errors or warnings were recorded on the editor, preview, episode or POA Summary pages. Every `/api/` request I captured returned 200, except the expected 404 for Bea on Trust A's episode.

## GIFs

- `~/Downloads/S39-T01.gif`: the happy path, the ASA guide filling in ASA III (COPD).
- `~/Downloads/S39-T10.gif`: the POA Summary's suggested ASA on both tabs, and the reason link to Validate.
- `~/Downloads/S39-F1-stale-reason.gif`: F1.

## Hand checks for Rahul

- R-05 at a real 390×844 window: the unsent patient set list (my window couldn't be resized; I checked the sent pages in a 390 px iframe).
