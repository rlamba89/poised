# S44c test report 4 (retest after verification 2)

**Summary: 7 passed, 0 failed, 0 blocked.** This round covered everything listed at the top of the test plan under "Retest after verification 2":
- F3 (a typed number's follow-up), for the patient at phone size and for Fill in for the patient;
- the chain still clearing at once (T-01 steps 4–7 at phone size, including "B No before typing in C");
- the short regression: T-03, T-12 step 2, R-05.

- **Environment:**
  - Code: `f143b17` with the S44c changes uncommitted (working tree dirty).
  - Time: 28 Oct 2026, 16:14 to about 16:30 GMT.
  - App: http://localhost:3000 under the conductor's `make dev`: API :8080, web :3000, and the evaluate service on :8090, which reports `clinical@f143b177d60e-dirty`.
  - E2E journeys were running against the same app at the same time (E2E HQs and episodes stamped 16:11–16:12). They didn't touch my data.
- **Browser:** Claude in Chrome, Browser 1 (bce1f965…).
  - `resize_window` again had no effect on the viewport, which stayed 2560 px wide.
  - So for the phone-size cases I loaded the patient's invite link in a 390×844 same-origin iframe, as in report 3. Inside the frame the page lays out at 390 px, and I worked it with real clicks and key presses.
  - "Reload" for the patient means reloading the frame (`contentWindow.location.reload()`).
- **Test data:**
  - **HQ:** **MT S44c units 1615** is new, built in the editor as Alex.
    - It has one patient set, **Alcohol**.
    - Number **How many units of alcohol a week?** is required.
    - Long text **Tell us more about your drinking** is shown when units > 14. I made it **optional**, the riskier case from verification 2.
    - I deleted **Low risk** on the Rules page. Publish said "This version doesn't auto-triage anyone." v1 is published.
  - **MT S44c chain 1335** v1 from round 1 was reused.
  - **Episodes**, all for Sam, DOB 17/05/1980:
    - "Sam MT S44c R4U 1625" (units, patient) was made on the New episode page.
    - Four were made with the same `POST …/episodes` call the page makes, run from the page as Cara:
      - "R4C 1627": chain, patient;
      - "R4OBU 1627": units, on behalf;
      - "R4OBC 1627": chain, on behalf;
      - "R4V 1627": chain, sent with B No.
    - A second R4U episode was made the same way for R-05.
- **API, database and terminal use:**
  - I read invite links from `GET /api/qa/messages`, the data behind /qa/messages.
  - I read the stored `episode_answers` rows (revision, data, time) after each step to confirm what was saved.
  - I watched "Saving…/Saved" with a small in-page poller.
  - I read the field values on Validate with one `querySelectorAll`.

## Results

| Case | Result | Note |
|---|---|---|
| **F3, patient, phone size** (MT S44c units 1615) | **Pass** | 20 → Backspace → 5 keeps "Wine with dinner" on screen and in the row. Units 2 clears it. Back to 20, the follow-up is empty. Details below. |
| **F3, Fill in for the patient** (as Cara) | **Pass** | The same steps give the same result on screen and in the stored rows. |
| **The chain still clears at once** (T-01 steps 4–7, phone size) | **Pass** | B No hid A and C at once ("Saved → Saving… → Saved"), and the row became `{B: No}` with Pillows `{}`. B Yes, A Yes, then B No before typing in C: **Finish this section** worked, with no "Response required". |
| **T-03**, Fill in for the patient on the chain | **Pass** | |
| **T-12 step 2** (and step 3) | **Pass** | |
| **R-05**, pre-fill | **Pass** | |
| No regression on opening a set (F1) | **Pass** | Opening Review on a finished set left the status at **Saved** with no "Saving…", on both the patient and on-behalf forms. |

**F3, patient, phone size** (MT S44c units 1615):
- **Entering the answers:**
  - Units 20 showed "Tell us more", and I typed "Wine with dinner".
  - The status read Saved, and the row was `{units: 20, why: "Wine with dinner"}` at revision 2.
- **Correcting 20 to 25:** I clicked in the units box, pressed End, Backspace, then 5.
  - At "2" the follow-up disappeared for that instant.
  - At "25" it came back with "Wine with dinner".
  - After the pause, the row was `{units: 25, why: "Wine with dinner"}` at revision 3.
- **Reload, then Review:** units 25 and "Wine with dinner" were both still there.
- **Units 2:**
  - The follow-up went.
  - After the save, the row was `{units: 2}` at revision 4.
- **Reload, Review, then 20** (End, type 0): "Tell us more" showed **empty**.
- GIF: `~/Downloads/S44c-r4-F3.gif`.

**F3, Fill in for the patient** (as Cara, desktop width; episode R4OBU):
- **Correcting 20 to 25** (Backspace, then 5): the follow-up kept "Wine with dinner", and the row was `{units: 25, why: "Wine with dinner"}` at revision 3.
- **Reload, then Review:** units 25 and the text were both still there.
- **Units 2:** the row became `{units: 2}`.
- **Reload, Review, then 20:** "Tell us more" was empty.

**T-01 steps 4–7** (phone size, episode R4C):
- **Before step 4:** Breathing B Yes, A Yes, C "Coughing"; Pillows D Yes. The count read "2 of 2".
- **Step 4:** B No hid A and C at once, and the status went "Saved → Saving… → Saved". The rows were Breathing `{B: No}` at revision 4 and Pillows `{}`.
- **Step 5:** Pillows was gone, and the count read "1 of 1".
- **Step 6:** after a reload, B showed No and A and C were hidden.
- **Step 7, first round:**
  - B Yes showed A empty, and A Yes showed C empty.
  - Then **B No before typing in C**, and Finish: back on the list with "1 of 1", and no "Response required".
- **Step 7, second round:**
  - Review, then B Yes: A was empty.
  - A Yes: C was empty.
  - I typed "Snoring" and pressed Finish. Pillows was listed again, and its Start opened D empty.
- **Width:** `scrollWidth` 390 = `clientWidth` 390.

**T-03, Fill in for the patient on the chain** (Cara, episode R4OBC):
- **Steps 3–7:** the same results as T-01. B No hid A and C at once ("Saved → Saving… → Saved"), and Pillows went from the list. After a reload, B Yes showed A empty, A Yes showed C empty, and Pillows' D was empty.
- **← Episode:** back on the episode page, with nothing refused.
- **Rows:** Breathing `{B: Yes, A: Yes, C: "Snoring"}`, Pillows `{}`.
- No console errors.

**T-12 step 2 (and step 3)** (episode R4V, which the patient sent with B No):
- **Changes on Validate Breathing:**
  - B No → Yes showed "Patient answered: No".
  - A Yes showed no line.
  - C "Coughing" showed no line.
  - The page text had exactly one "Patient answered" line.
- **After Validate:** **Pillows** was listed as Not started, and **Complete HQ review** was disabled ("Validate every Question Set…").

**R-05, pre-fill** (a second episode on MT S44c units 1615 for the R4U patient, who had sent 30 and "Wine with dinner"):
- **The list:** Alcohol showed "PLEASE CHECK" and a **Check** button.
- **The card:** "We've filled in 2 answers from your questionnaire on 28 Oct 2026", with **Check each answer** and **Everything is still correct**.
- **Check each answer:** 30 and "Wine with dinner" each showed "From your answers on 28 Oct 2026".

## Found while exploring

Nothing failed. Each check below behaved correctly.

- **Leaving straight after a number change:**
  - Units 20 with "Beer at weekends", then units 3, then **All sections** at once, without waiting for Saved.
  - The row became `{units: 3}`, cleared on the save made when leaving the set.
- **Emptying the box and retyping:**
  - On behalf: units 20 with "Cider 🍺 on Fridays", then Cmd+A, Backspace (the box was empty), then 18.
  - The follow-up kept its text, emoji included, and the row was `{units: 18, why: "Cider 🍺 on Fridays"}`.
- **A pre-filled follow-up:**
  - On R-05's set, 30 → Backspace → 5 kept the pre-filled "Wine with dinner".
  - The row was `{units: 35, why: "Wine with dinner"}`.
- **The clinician's Validate with the same edit:**
  - On the sent R4U episode, 30 → Backspace → 5: the follow-up stayed throughout. Validate commits a typed value when you leave the box, not on each key.
  - After Tab: "Patient answered: 30" under units.
  - The clinician's row was `{units: 35, why: "Wine with dinner"}`; nothing was lost.
- **Another patient's episode:** opening R4V's `/home/episodes/{id}` while signed in as the R4U patient gave "Episode not found."
- **Cosmetic, not a defect:** right after B No on the chain, the space where A and C were closes up over a fraction of a second, so a screenshot taken at once shows a gap above Finish. It's gone a moment later.
- **Not a defect:** the new number question showed "PRE-FILLS FROM 6 OTHER HQS" in the editor straight away, matched by exact wording against the E2E journeys' HQs (J3b's hidden-number HQ). That's the expected pre-fill-by-wording badge.

## Console and network errors

- None in the staff tabs, checked with `read_console_messages` for errors and warnings.
- The patient frame's console isn't captured by the extension.
- Every save was confirmed in the stored rows, and none failed or conflicted.

## GIFs

- `~/Downloads/S44c-r4-F3.gif`: F3 at phone size. 20 and "Wine with dinner", then 20 → 2 → 25 keeps the follow-up; reload; units 2 clears it; back to 20, the follow-up is empty.

## Hand checks for Rahul

- None.
