# S38 test report 5 (retest after verification 2)

**Summary:** 8 cases run: V-04, V-05 and V-06 (verification 2's fix F4), then the plan's quick pass of T-10, T-12, T-13 and R-03. **8 passed, 0 failed, 0 blocked.**
- F4 is fixed. A Question Set shown by a score now follows a changed earlier answer, for the patient, for staff filling in for them, and on the clinician's side.
- Deleting STOP-Bang now warns about a score in a later set that uses one of its questions.
- Exploring found nothing that fails. One behaviour is worth a look (see "Found while exploring").

**Environment:**
- Commit `a44ac2c` on master. The working tree is **not clean**: it holds S38's uncommitted build and fixes (56 changed or new files).
- Tested on 26 Oct 2026, 16:02–16:22 GMT, at http://localhost:3000 (`make dev`, API :8080). The database was freshly migrated and seeded.
- Chrome through Claude in Chrome, desktop size (1549 × 783).
- Users:
  - Alex Author, for authoring, the Preview and publishing.
  - Cara Clinician, for episodes, Fill in for the patient, Validate and the POA Summary.
  - One patient, through the invite link from /qa/messages and their date of birth.

**Test data (all made up, built through the editor):**
- **"MT S38 1605 Follow-up HQ"** v1, published. It has three Patient sets:
  - **Snoring**: "Do you snore loudly?" (Yes 1, No 0).
  - **Tiredness**: "Are you tired in the day?" and the Calculation "Snoring score", which adds up the snoring question, with Clinical **off**.
  - **Sleep follow-up**: "Has anyone seen you stop breathing?", shown only when Snoring score ≥ 1.
- **"MT S38 1605 Scores HQ"**, built as the plan's setup describes:
  - **Sleep** › Page 1 has snore and tired, both scored 1/0. Sleep › Page 2 has the BMI.
  - **Neck** › Page 1 has collar (scored 1/0), a Clinical Summary and "Sleep risk".
  - "Sleep risk" adds up the 3 questions, with the lines Age > 50, Sex is Male and BMI > 35, the bands Low ≤ 2 and High above, and "High risk of sleep apnoea" (Cardiovascular) on High.
  - v1 is published. v2 is published with Sleep shown only when "Patient's sex is Female".
- **"MT S38 1605 STOP-Bang HQ"** v1 draft, with three sets:
  - Measures (BMI);
  - Sleep check (STOP-Bang);
  - Extra, with the Calculation "MT snoring again", which adds up STOP-Bang's snoring question.
- Patients:
  - **Sam MTvfour S38**: female, 01/01/1980, Follow-up HQ. Used for V-04 and V-05 through the invite link.
  - **Sam MTfive S38a**: male, 01/05/1975 (51), Scores HQ v1. Used for T-10.
  - **Sam MTfive S38c**: female, 01/03/1986 (40), Scores HQ v1. Used for T-12.
  - **Sam MTfive S38d**: male, 01/02/1966 (60), Scores HQ v2. Used for T-13.
  - **Sam MTvfour S38x**: female, 02/02/1981, Follow-up HQ. Used for exploring, through Fill in for the patient.

## Results

| Case | Result | Note |
|---|---|---|
| **V-04** | **Pass** | The patient used the invite link, Continue and their date of birth. At first the list had Snoring and Tiredness only. After Snoring **Yes** and Finish, **Sleep follow-up** appeared at once and the list said "1 of 3 sections complete". After Tiredness **Yes**, it said "2 of 3" and **Send my answers** stayed disabled. I reviewed Snoring, changed it to **No** and finished, without opening Tiredness again. Sleep follow-up **went** from the list, it said "2 of 2 sections complete", and Send was enabled. This held after a reload. I changed Snoring back to **Yes**: Sleep follow-up came back, it said "2 of 3", and Send was disabled again. Opening Tiredness after the change showed the Snoring score as 0, which is right. |
| **V-05** | **Pass** | I set Snoring back to No and sent. As Cara, the episode's Health questionnaire card lists **Snoring** and **Tiredness** only. The POA Summary has no Sleep follow-up section on either tab (Validated summary or Patient answers). It says "No answers produce anything for the summary yet." |
| **V-06** | **Pass** | In the STOP-Bang draft, I deleted the STOP-Bang Group from the Sleep check card. The "Delete this?" window says: "Other items depend on this. Their conditions will no longer work: • Extra · "MT snoring again" (calculation)". I pressed Cancel, and the Group is still there. There were no console errors. |
| T-10 | Pass | Sam MTfive S38a is male and 51. I filled in for the patient: snore Yes, tired No, 170 cm and 70 kg (BMI 24.2), collar Yes. I sent the answers; the patient form never showed "Sleep risk". Validate Neck shows **"Sleep risk" 4 (High)**, with "High risk of sleep apnoea" in the Page 1 summary box. After Validate, the POA Summary's Neck › "Page 1 summary" lists "High risk of sleep apnoea". |
| T-12 | Pass | Sam MTfive S38c is female and 40. Answers: snore Yes, tired Yes, 160 cm and 100 kg (BMI 39.1), collar No. Validate Neck first showed **3 (High)** with the note. In Validate Sleep I changed tired to **No**; the "Patient answered: Yes" note appeared, and I pressed Validate. When I opened Neck's Validate again, it showed **2 (Low)**, and the summary box said "No disclosures on this page yet." No console errors, and every API request returned 200. |
| T-13 | Pass | I published v2 (Sleep shown only for Female). Sam MTfive S38d is male and 60, on v2. Fill in for the patient listed **Neck only**, and so did the Health questionnaire card. Validate Neck (collar Yes) shows **"Sleep risk" Incomplete**, and the summary box says "No disclosures on this page yet." The POA Summary has no "High risk" note: "No answers produce anything for the summary yet." |
| R-03 | Pass | In the v2 draft, Sleep's Preview with the sample patient Female says "This Question Set is shown only when Patient's sex is Female. The sample patient meets it, so they would see this Question Set." With Male it turns red: "The sample patient doesn't meet it, so they wouldn't see this Question Set. The questions below are shown anyway." This is unchanged. As in T-13, the set is hidden for a male patient. |

## Failures

None.

## Found while exploring

- **A clinician's correction can bring back a set that was hidden when the answers were sent. It then shows the patient's old answers to it.** This is not a failure; it's for Rahul to decide whether it's the intended behaviour.
  - Steps: in Fill in for the patient for Sam MTvfour S38x on the Follow-up HQ:
    1. Snoring Yes, Tiredness Yes, Sleep follow-up Yes.
    2. Change Snoring to **No**. Sleep follow-up disappears, as expected. Send.
  - The episode lists Snoring and Tiredness only, as expected.
  - Then in Validate Snoring I corrected the answer to **Yes** and pressed Validate. The Health questionnaire card now lists **Sleep follow-up** too, as "Not started" with Validate.
  - Its Validate page shows "Has anyone seen you stop breathing?" answered **Yes**. That is the answer given before the set was hidden; it wasn't in the set the patient sent.
  - This follows the rule that a clinician's correction counts (as in T-12), so it may well be right. But the clinician is shown, as the patient's answer, an answer from a set that was hidden when the patient sent.
- Staff filling in for the patient get the same live behaviour as the patient. The follow-up set appears after Snoring Yes and goes after Snoring No.
- Cosmetic, not new: the delete warning puts a score under "Their conditions will no longer work". The score's sum is what breaks, not a condition. A score card with one line says "plus 1 points line", but the plan's V-01 expects this wording.
- On the questionnaire list, the first click on **New Questionnaire** after loading the page sometimes did nothing. That was in a batch of clicks; a single click worked every time. I couldn't make it happen again, so this may be my tooling.

## Console and network errors

- No console errors or warnings on the editor (STOP-Bang delete), Validate or the episode page.
- No failed API requests (all 200) on the reloads I checked.

## GIFs

- `~/Downloads/S38-V04.gif`: V-04's main path, from the patient's side. It shows Snoring Yes and Tiredness Yes with the follow-up listed, then Snoring changed to No (follow-up gone, Send enabled), then back to Yes and No again, and Send.

## Hand checks for Rahul

- None from this round. T-14 at a true 390 × 844 is still listed by the verifier.
