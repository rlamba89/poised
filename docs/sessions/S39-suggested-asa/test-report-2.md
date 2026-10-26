# S39 test report 2 (retest)

**Summary: 22 cases re-run, all passed. 0 failed, 0 blocked. Report 1's F1, F2 and cosmetic notes C1–C3 are all fixed.**

- **Code:** `c1f6c69`, plus S39's uncommitted changes and the fixes for report 1, all in the working tree (not committed).
- **When:** 26 Oct 2026, 18:28–18:47.
- **Where:** http://localhost:3000. The conductor ran `make dev` on freshly migrated and seeded data. I tested in Chrome through Claude in Chrome.
- **Window:** the resize to 390×844 was accepted but had no effect (the window stayed 1549 px wide). As in report 1, I checked phone width in a 390×844 same-origin iframe. This time that included the patient's **unsent** set list.
- **Data made for this round:**
  - HQ **MT S39 R2 1835 Retest** v1, published, made as Alex Author. It has:
    - About you: "Do you have lung disease?" (Yes → COPD, ASA III; No → Smoker, ASA II). The Smoker disclosure was made by adding COPD and Smoker, then removing COPD.
    - A Select Many question with three options added through "From the code library": COPD (III), Type 2 diabetes (III, Poorly controlled diabetes) and Heart failure (no grade).
    - Anaesthetic plan: a clinician set shown when the suggested ASA is at least III.
  - Test case **MT R2 COPD diabetes** (Clinician).
  - Episode **Rita MT S39 R2 1900 Retest**: Cara Clinician, Hospital A1, made-up email `mt-s39-r2-1900@example.test`. The patient answered through the invite link with lung Yes, diabetes and heart failure, and sent. Cara then validated About you with lung No and diabetes unticked, and set Nurse ASA III.
  - A Smoker disclosure (ASA II) on Statement 2 of report 1's draft **MT S39 1750 ASA**, added while exploring (the double-click check).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | The row says "· ASA III". Clicking it fills in III, selects "ASA III: COPD (Adults)" and gives the reason line "COPD" (from the ASA guide). It saves as ASA III (COPD) and is the same after a reload. |
| T-02 (part) | Pass | Covered within T-05b step 4: Save is refused with "Choose the ASA grade from the ASA guide, or No ASA grade.", and the pale yellow box lists the three diabetes examples. |
| T-04 step 5 | Pass | For I50.9 (opened later from an option's disclosure), picking IV by hand selects no radio and shows no reason. |
| T-05 (changed step 2) | Pass | Editing the saved COPD disclosure shows the guide box with "ASA III: COPD (Adults)" selected. Adding Smoker keeps **III**, and the box adds "ASA II: Current smoker (Adults)". Choosing it gives II with "Current smoker". Cancel keeps ASA III (COPD). |
| T-05b step 1 | Pass | COPD then Smoker: the grade stays III. |
| T-05b step 2 (**F1**) | Pass | Removing COPD gives **II** with Smoker's example selected and the reason "Current smoker". It saves as "SNOMED 77176002 · SMOKER · ASA II (Current smoker)". In the Clinician preview, answer No shows "ASA II / Smoker (77176002) – Current smoker". |
| T-05b step 3 | Pass | COPD chosen (III) with Smoker, then Smoker removed: III (COPD) stays, and the box lists only COPD. |
| T-05b step 4 | Pass | COPD + E11.9, then COPD removed: the grade is empty and the yellow box lists the three diabetes examples. Save is refused with the red message. |
| T-05b step 5 | Pass | V typed by hand, COPD added, then E11.9 removed: the grade stays V. |
| C1 (report 1) | Pass | Choosing "ASA III: Poorly controlled diabetes" after the red message clears the message straight away. |
| T-06 (**F2**) | Pass | The window text is as expected. COPD's option is ASA III (COPD), and diabetes' option has no grade. Editing the diabetes disclosure **now shows the guide box** (no radio selected). Picking **III from the grade list** selects "Poorly controlled diabetes" and gives its reason; it saves as ASA III (Poorly controlled diabetes). A heart-failure option added the same way saves with no choice (not refused), as the plan says. |
| T-07 | Pass | Sue Super, on the published HQ: the diabetes option shows the ASA III badge and "(Poorly controlled diabetes)", with only Done and ✕ (no Add, Edit or Remove). |
| T-08 | Pass | Clinician preview: Yes gives "ASA III / Chronic obstructive lung disease (13645005) – COPD". Yes + diabetes gives ASA III with both reasons. |
| T-09 | Pass | Clinician set "Anaesthetic plan": Suggested ASA is under "Patient and viewer", "is at least" / III. The note under the condition is shown. |
| C3 (report 1) | Pass | Switching the set to Patient shows "1 logic problem", and the condition row **still shows "Suggested ASA / is at least / III"**. Switching back to Clinician clears the problem. |
| T-10 | Pass | Episode page: "Suggested ASA III: Chronic obstructive lung disease (13645005) – COPD; Type 2 diabetes mellitus without complications (E11.9) – Poorly controlled diabetes (the anaesthetist confirms)", and Anaesthetic plan is listed. The POA Summary's Validated tab shows ASA III, both reasons, the "Suggested from the answers…" line and the nurse's and anaesthetist's boxes. The Patient answers tab shows the same without the boxes. |
| T-11 | Pass | The diabetes reason opens Validate for "About you" at `?q=q_oymema`, with the Select Many card highlighted in yellow and a "← POA Summary" link. |
| T-12 | Pass | Validated with lung No and diabetes unticked: the episode page says "Suggested ASA II: Smoker (77176002) – Current smoker", and Anaesthetic plan has gone from the list. The validated tab says II (Smoker); the Patient answers tab still says III with both reasons. (A first validation with only lung changed correctly showed III with just the diabetes reason.) |
| T-14 / C2 (report 1) | Pass | Episode "MT S39 1830 Nograde": the episode page says "No suggested ASA: no answer gives an ASA grade" **without** "(the anaesthetist confirms)". Both POA tabs say "No suggested ASA: no answer gives an ASA grade." |
| T-16 | Pass | `…/poa?tab=patient` loads directly on that tab with its suggestion (III). |
| R-01 | Pass | The S38 episode "Sam MTfive S38a": the POA Summary still lists "High risk of sleep apnoea" under Neck. |
| R-02 | Pass | "MT S39 1840 Emerg": the "Needs surgery today" note opens Validate About you with that question highlighted. The POA Summary button returns to the summary at the same scroll position. |
| R-03 | Pass | Nurse ASA III was saved and was kept after a reload ("Nurse ASA grade: ASA III"). The General notes say "Nurse ASA grade set to 3". |
| R-04 | Pass (part) | Test case "MT R2 COPD diabetes" (Suggested ASA III): Run all says "All 1 passed", and Publish went through. I didn't recheck the "failing test case blocks Publish" half (it passed in report 1, and nothing there changed). |
| R-05 | Pass (iframe) | Patient home, desktop: only "About you" (no Anaesthetic plan), and no "ASA" or "Anaesthetic" text. At 390 px in an iframe, the **unsent** set list has no sideways scroll (scrollWidth 386 = clientWidth) and no ASA text. |

Not re-run: T-03, T-13, T-15 and T-17. They passed in report 1, and nothing they depend on changed.

## Failures

None.

## Found while exploring

- **A disclosure saved before the F1 fix keeps its wrong reason.** On report 1's draft "MT S39 1750 ASA", Statement 1 still says "SMOKER · ASA III (COPD)". Opening Edit shows grade III, the guide box offers only "ASA II: Current smoker" / "No ASA grade" (nothing selected), and the line still reads "COPD" (from the ASA guide). This is leftover test data made by the old bug, and a new disclosure can no longer get into this state. Not counted. The builder may want to decide whether a reason that no remaining code offers should be dropped when such a disclosure is opened or saved.
- **Double-clicking Save output** on a new disclosure saved it once (a 🔗 1 badge, one PUT). The second click closed the Disclosures window.
- Adding COPD to a disclosure that already has II from Smoker keeps II, and the box adds "ASA III: COPD". This matches T-05's rule.
- Opening a long disclosure (two codes plus four guide examples) puts Save output close to the bottom of a 783 px window, but it's still reachable.

## Console and network errors

- No console errors or warnings on the editor, episode, POA Summary or patient pages, from the point tracking started.
- Every `/api/` call returned 200, except two shown as **503** by the extension: `POST /api/logout` and `POST /api/dev/login` during a sign-out and sign-in. Both worked: the user changed as expected. Called directly, `curl -X POST localhost:3000/api/logout` returns **204**. So this looks like the extension's report of a request cut short by the page navigating away, not a server error.

## GIFs

- `~/Downloads/S39-R2-T01-T05b.gif`: T-01 (the guide fills in ASA III COPD), T-05 step 2, and the F1 fix (removing COPD gives II, Current smoker).
- `~/Downloads/S39-R2-T10-T12.gif`: the episode page and the POA Summary on both tabs, the reason link to Validate, and the validated tab dropping to II.

## Hand checks for Rahul

- R-05 in a real 390×844 window. The extension's resize had no effect, so I checked phone width in a 390 px iframe instead, which this time included the unsent set list.
