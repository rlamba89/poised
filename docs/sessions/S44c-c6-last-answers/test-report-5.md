# S44c test report 5 (retest after verification 3)

**Summary: 8 passed, 0 failed, 0 blocked.** This round covered everything listed at the top of the test plan under "Retest after verification 3":
- F4: a BMI's weight, and a number that feeds a score;
- the chain still clearing at once (T-01 steps 4–7 at phone size);
- F5;
- Answer 8 (Start fresh after the form's own clearing save);
- the short regression: T-03 and R-05.

- **Environment:**
  - Code: `f143b17` with the S44c changes uncommitted (working tree dirty, 64 paths).
  - Time: 28 Oct 2026, 19:32 to about 20:00 GMT.
  - App: http://localhost:3000 under the conductor's `make dev`: API :8080, web :3000, and the evaluate service on :8090, which reports `clinical@f143b177d60e-dirty`. The database was migrated to 00036.
- **Browser:** Claude in Chrome, Browser 1 (bce1f965…).
  - `resize_window` again had no effect on the viewport, which stayed 2560 px wide.
  - So, as in reports 3 and 4, the patient's pages ran in a 390×844 same-origin iframe, worked with real clicks and key presses. "Reload" means reloading the frame.
- **Test data (all made up):**
  - **HQs built in the editor:**
    - As Alex: **MT S44c weight 1935**, **MT S44c unitscore 1940** (v1 and v2), **MT S44c frailty 1950** (left as a draft) and **MT S44c plain 1955**.
    - As Bea, in Trust B: **MT S44c B pillows 2000**.
  - **HQ reused:** MT S44c chain 1335 v1.
  - **Episodes**, all for Sam, DOB 17/05/1980:
    - **R5W 1940** was made on the New episode page.
    - The rest were made with the same `POST …/episodes` call the page makes, run from the signed-in page: R5S, R5C, R5OBC, R5S2, a second episode for R5S2 on MT S44c units 1615, and R5X (one in Trust B and one in Trust A).
- **API, database and terminal use:**
  - I read invite links from `GET /api/qa/messages`.
  - I read the stored `episode_answers` rows (revision, `cleared_saves`, data, prefill) after each step.
  - I watched "Saving…/Saved" with a small in-page poller.
  - I made one direct publish call and one direct Start fresh call. Both are described below.

## Results

| Case | Result | Note |
|---|---|---|
| **F4, a BMI's weight** (patient, phone size) | **Pass** | 120 → Backspace → 5 kept "Diet plan" on screen, in the row and after a reload. Weight 60 cleared it. Back to 120, the follow-up came back empty. |
| **F4, a number that feeds a score** (patient, phone size) | **Pass** | Tested twice: with the score clinician-only, and with the score shown to the patient, which really exercises the expression route. 20 → 2 → 25 kept "Wine with dinner" both times. |
| **The chain still clears at once** (T-01 steps 2–7, phone size) | **Pass** | B No hid A and C at once ("Saved → Saving… → Saved"). Pillows was cleared in the same save. A, C and D came back empty. |
| **F5**, a rule on a deleted score | **Pass** | Publish showed the problem and its button was disabled. The server also refused with 422. |
| **F5**, an HQ without STOP-Bang keeps the defaults | **Pass** | The HQ published with no problem. |
| **Answer 8**, Start fresh after the form's own clearing save | **Pass** | Set up across trusts by hand. The server accepted Start fresh after the clearing save. Details below. |
| **T-03**, Fill in for the patient on the chain | **Pass** | |
| **R-05**, pre-fill | **Pass** | |

**F4, a BMI's weight** (MT S44c weight 1935, episode R5W):
- **The HQ:**
  - Set **Weight**.
  - Page 1 has the BMI.
  - Page 2 has an optional long text **Tell us about your weight**, shown when "BMI Calculator (BMI) > 35" (`{q_lbeewd.bmi} > 35`).
  - Low risk was deleted. Publish said "This version doesn't auto-triage anyone."
- **Entering the answers:**
  - Height 170 and weight 120 gave a BMI of 41.5.
  - **Next**, then I typed "Diet plan".
  - The row was `{bmi: 41.5, height 170, weight 120, q_dpdmum: "Diet plan"}` at revision 2.
- **Correcting 120 to 125:** **Previous**, a click in the weight box, End, Backspace, then 5.
  - At "12" the BMI read 4.2, and the button changed to Finish this section, because page 2 was hidden for that instant.
  - At "125" the BMI read 43.3.
  - After the pause, the row was `{bmi 43.3, weight 125, q_dpdmum: "Diet plan"}` at revision 3: kept.
  - **Next** showed "Diet plan".
- **Reload, Review, Next:** "Diet plan" was still there.
- **Weight 60** (BMI 20.8):
  - After the save, the row was `{bmi 20.8, weight 60}` at revision 4, with the follow-up gone.
  - After a reload, the set showed only **Finish this section**: the follow-up page was hidden.
  - Changing the weight back to 120, then **Next**, showed the follow-up **empty**.
- GIF: `~/Downloads/S44c-r5-F4-bmi.gif`.

**F4, a number that feeds a score** (MT S44c unitscore 1940):
- **The HQ:**
  - Set **Alcohol**.
  - Number **How many units of alcohol a week?** (required).
  - Optional long text **Tell us more about your drinking**, shown when units > 14.
  - A Calculation **Alcohol score**, "Add 1 point when units > 14". Its expression is `iif(allAnswered('q_geirll'), iif({q_geirll} > 14, 1, 0), '')`.
  - Low risk was deleted.
- **v1, as the plan has it** (episode R5S): the Calculation is **Clinical**, as a new Calculation is by default. STOP-Bang's score is clinician-only as well.
  - 20, "Wine with dinner", then 20 → Backspace (2) → 5.
  - The follow-up disappeared at "2" and came back with its text at "25".
  - The row was `{units: 25, why: "Wine with dinner"}`.
- **Why I added v2:** a clinician-only element is stripped from the patient's form. So with a clinical score, the patient's survey has no expression, and v1 doesn't reach F4's second route.
- **v2: Clinical switched off**, so the patient sees **Alcohol score** (episode R5S2):
  - **Entering the answers:** 20 showed score 1 and the follow-up, and I typed "Wine with dinner".
  - **Correcting 20 to 25:** Backspace showed score 0 with the follow-up gone. Then 5 showed score 1 and "Wine with dinner" back.
    - The row was `{units: 25, why: "Wine with dinner", score: 1}` at revision 3.
  - **Reload, then Review:** units 25, "Wine with dinner" and score 1 were all still there.
  - **Units 2:**
    - The follow-up went, and the score read 0.
    - The row was `{units: 2, score: 0}` at revision 4.
  - **Reload, Review, then 20:** the follow-up showed **empty**, with score 1.

**The chain still clears at once** (T-01 steps 2–7, phone size; episode R5C on MT S44c chain 1335):
- **Steps 2–3:**
  - The list showed only Breathing ("0 of 1").
  - I answered B Yes, A Yes and C "Coughing", then pressed Finish. Pillows appeared.
  - Pillows: D Yes, then Finish. The count read "2 of 2 sections complete".
- **Step 4:** Review Breathing, then B No.
  - A and C went at once, and the poller recorded "Saved → Saving… → Saved".
  - The rows were Breathing `{B: No}` at revision 4, and Pillows `{}` at revision 2 with `cleared_saves` 1. Pillows' save is marked as the form's own clearing (answer 8).
- **Step 5:** Pillows was gone, and the count read "1 of 1".
- **Step 6:** after a reload and Review, B showed No, with A and C hidden.
- **Step 7:**
  - B Yes showed A **empty**, and A Yes showed C **empty**.
  - I typed "Snoring" and pressed Finish. Pillows was listed again, and its **Start** opened D **empty**.
- **Width:** `scrollWidth` 388 = `clientWidth` 388 (the frame's inner width), so nothing was wider than the screen.

**F5** (as Alex, MT S44c frailty 1950, a draft):
- **The HQ:**
  - A Number **How many falls this year (MT F5)?**
  - A Calculation **Frailty**, "1 point when falls > 1", with bands **Low** (≤ 2) and **High** (above).
- **Editing Low risk:** "and a score isn't in a band" was already ticked, with STOP-Bang (not in this HQ).
  - The score list offered **Frailty** and **STOP-Bang score (not in this HQ)**.
  - I chose Frailty / **High** and pressed **Save rule**.
  - The rule read "…no red flags and Frailty isn't High: auto-triage to Ready for admission".
- **Deleting Frailty, then Publish:**
  - Publish showed "1 problem to fix first: **Rules** · Rule “Low risk”: its score isn't in this HQ any more", and **Publish** was disabled.
  - I also posted the publish request myself, as Alex. The server answered **422** with that problem ("This version can't be published until these problems are fixed.").
- **An HQ without STOP-Bang and with the default rules** (MT S44c plain 1955: one Yes/No question):
  - Publish showed no problem. Its line read "This version auto-triages: ASA I–II, no red flags → Ready for admission…".
  - It published as v1.

**Answer 8, Start fresh after the form's own clearing save** (set up across two trusts by hand):
- **Trust B side:**
  - As Bea, Trust B HQ **MT S44c B pillows 2000**: one Yes/No **Do you use more than two pillows?**, with the same wording as the chain's D.
  - A Trust B episode for Sam "MT S44c R5X 2000". The patient answered **Yes** and sent it.
- **Trust A side:** as Cara, a chain episode for the same person (same name, date of birth and email).
  - On the patient's first open, the chain's **Pillows** set was pre-filled from Hospital B1, at revision 1 (`matchedBy: wording`, `otherTrust: true`). It was still hidden.
  - B Yes, A Yes, C, then Finish: Pillows showed **PLEASE CHECK** with a **Check** button.
- **Hiding Pillows again:** Review Breathing, then B **No**.
  - The form's clearing save emptied Pillows: revision 2, `cleared_saves` 1, data `{}`.
  - B Yes, A Yes, then Finish: Pillows came back as **Start**, and it opened D empty.
  - There was no card, because no pre-filled answer was left, which is correct.
- **Start fresh itself:** with no card in the UI, I posted Start fresh on that set from the patient's signed-in page, the call the card's button makes (`POST /api/p/episodes/{eid}/prefill/{cid}`, `start_fresh`).
  - The server answered **204**, not refused.
  - The row became revision 3, `prefill_choice` = start_fresh.
  - Before S44c, revision 2 would have refused it.
- **Not set up:** a set that keeps other pre-filled answers after the clear, so the card stays up. That needs a question-level condition across sets with two matching Trust B questions. The integration test `TestStartFresh` covers it.

**T-03, Fill in for the patient on the chain** (Cara, desktop width; episode R5OBC):
- **Steps 3–7:** the same results as T-01.
  - B No hid A and C at once.
  - The rows were Breathing `{B: No}` at revision 4, and Pillows `{}` at revision 2 with `cleared_saves` 1.
  - After a reload, B Yes showed A empty, A Yes showed C empty, and Pillows' D was empty.
- **← Episode:** back on the episode page, with nothing lost or refused.
- No console errors.

**R-05, pre-fill:**
- **Setup:** the R5S2 patient sent unitscore v2 with 20 and "Beer at weekends". Then a new episode for the same patient on **MT S44c units 1615**, which uses the same wording.
- **The list:** Alcohol showed **PLEASE CHECK** and a **Check** button.
- **The card:** "We've filled in 2 answers from your questionnaire on 28 Oct 2026", with **Check each answer** and **Everything is still correct**.
- **Check each answer:** 20 and "Beer at weekends" each showed "From your answers on 28 Oct 2026".

## Found while exploring

Nothing failed.

- **A typed change that shows a question, then hides it again within one pause** (verification 3's N7, second way):
  - On the weight set: weight 120 showed the follow-up, and I typed "Snack".
  - Straight away: Previous, select the weight, then 60.
  - The stored row ended as `{bmi 20.8, weight 60}`, with the follow-up cleared at the save.
- **A pre-filled follow-up kept through a number edit:** on R-05's set, 20 → Backspace → 5 kept the pre-filled "Beer at weekends". The row was `{units: 25, why: "Beer at weekends"}`.
- **Not a defect: a BMI answer never pre-fills.**
  - A second weight-HQ episode for R5W opened with **Start** and no card.
  - That's by design: `keys.TakesPart` excludes the BMI block, as the prefill package says.
  - So I used the units HQ for R-05.
- **Not a defect: "BMI" greyed out on the Add content bar of a page that already holds a question.**
  - This is by design: BMI must be alone on its page (`AddContentBar.tsx`).
- **Not a defect: the clinician-only score.**
  - A new Calculation is Clinical by default, and STOP-Bang's score is clinician-only, so neither reaches the patient's form.
  - F4's expression route only matters for an author's score with Clinical switched off. v2 above covers it.
- **Cosmetic, not a defect:** the follow-up's space closes over a fraction of a second after a hide, as in report 4.
- **Possible small gap in wording (not a defect):** with Frailty deleted, the Publish window's auto-triage line read "ASA I–II, no red flags → Ready for admission", without mentioning the rule's missing score. Publish is refused anyway, so nothing goes out with it.

## Console and network errors

- None in the staff tab, checked with `read_console_messages` for errors and warnings.
- The patient frame's console isn't captured by the extension.
- Every save was confirmed in the stored rows. None failed or conflicted, and there was no "Not saved".

## GIFs

- `~/Downloads/S44c-r5-F4-bmi.gif`: F4 BMI at phone size.
  - 170/120 and "Diet plan", then Previous, then 120 → 12 → 125: the follow-up keeps "Diet plan".
  - Reload: still there. Weight 60 clears it. Back to 120, the follow-up is empty.

## Hand checks for Rahul

- None.
