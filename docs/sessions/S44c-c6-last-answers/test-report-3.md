# S44c test report 3 (retest after verification 1)

**Summary: 6 passed, 0 failed, 0 blocked.** This round covered everything listed at the top of the test plan under "Retest after verification 1": F1, F2, and the short regression (T-01 at phone size, T-07, and T-10 steps 1–2).

- **Environment:**
  - Code: `f143b17` with the S44c changes uncommitted (working tree dirty).
  - Time: 28 Oct 2026, 15:07 to about 15:20 GMT.
  - App: http://localhost:3000 under the conductor's `make dev`: API :8080, web :3000, and the evaluate service on :8090, which reports `clinical@f143b177d60e-dirty`.
- **Browser:** Claude in Chrome, Browser 1 (bce1f965…).
  - The extension dropped once, at about 15:08. I re-selected Browser 1 and nothing was lost.
  - `resize_window` doesn't change the viewport, which stays 2560 px wide. So for the phone-size case, I loaded the patient's invite link in a 390×844 same-origin iframe on the same tab. Inside the frame the page lays out at 390 px, and I worked it with real clicks.
- **Test data:** made in the UI unless noted.
  - **HQs:**
    - **MT S44c band 1515** is new. One patient set has a scored Yes/No question (Yes 1, No 0) and a clinician-only Calculation, "Snore score", with bands Low and High. It has no STOP-Bang and uses the default rules. v1 is published and v2 is a draft.
    - **MT S44c sb 1520** is a new draft. One set holds a Number question, a BMI page and a ready-made STOP-Bang page.
    - **MT S44c chain 1335** v1 from round 1 was reused.
  - **Episodes:**
    - "Sam MT S44c R3T01 1510" was made on the New episode page as Cara.
    - "Sam MT S44c R3OB 1512" was made with the same `POST …/episodes` call the page makes, run from the browser as Cara.
- **API, database and terminal use:**
  - I read the invite link from `GET /api/qa/messages`, the data behind /qa/messages.
  - I read the stored `episode_answers` rows (revision, data, updated_at) to confirm that opening a set saves nothing and that a change clears the hidden answers.
  - I signed in as Alex with the dev login's `POST /api/dev/login`, the same call the login page makes.
  - To watch the "Saving…/Saved" status between screenshots, I polled it with a small in-page script.

## Results

| Case | Result | Note |
|---|---|---|
| **F1**, patient (on T-01's episode) | **Pass** | Details below the table. |
| **F1**, Fill in for the patient (T-03 steps) | **Pass** | Details below the table. |
| **F2**, an HQ with a banded score and no STOP-Bang | **Pass** | Details below the table. |
| **F2**, an HQ with no banded score | **Pass** | Details below the table. |
| **T-01**, the chain at phone size (390 px frame) | **Pass** | Details below the table. |
| **T-07**, a new HQ with STOP-Bang | **Pass** | The rule reads "…and there are no red flags and STOP-Bang score isn't High risk: auto-triage to Ready for admission". Edit shows "and a score isn't in a band" ticked, with **STOP-Bang score** and **High risk**; the list offers only "STOP-Bang score", with no "(not in this HQ)" copy. The Publish window (cancelled, not published) reads "ASA I–II, no red flags, STOP-Bang score not High risk → Ready for admission". |
| **T-10** steps 1–2 | **Pass** | Details below the table. |

**F1, patient (on T-01's episode):**
- I opened **Review** on a finished Breathing set. The status went straight to **Saved**, with no "Saving…" while the set opened.
- The stored row was unchanged: revision 3, `updated_at` from before the Review click.
- **B Yes → No:**
  - A and C went at once, and the status went "Saving…" then "Saved".
  - The rows became Breathing `{B: No}` at revision 4 and Pillows `{}`.

**F1, Fill in for the patient (T-03 steps):**
- As Cara, on a new episode I answered B Yes, A Yes, C "Coughing" and D Yes, and finished both sets.
- **Opening Review on Breathing:** the status stayed Saved and nothing was saved; the row was still revision 3.
- **B → No:** A and C went at once ("Saving…" then "Saved"). Back on the list, Pillows was no longer listed and the count read "1 of 1".
- **B Yes again:** A came back empty, and after A Yes, C came back empty.
- **← Episode:** back on the episode page with nothing refused. The stored rows were Breathing `{B: Yes, A: Yes}` and Pillows `{}`.
- No console errors.

**F2, an HQ with a banded score and no STOP-Bang** (MT S44c band 1515 v1, draft):
- **Edit Low risk:** the "Score not in a band" list offered **Snore score** and **STOP-Bang score (not in this HQ)**.
- **Choosing Snore score:** after that, the list still offered STOP-Bang score (not in this HQ).
- **Choosing STOP-Bang back:** the bands offered were Low, Intermediate and High risk. I picked High risk and pressed Save rule.
- **The saved words:** "…and there are no red flags and STOP-Bang score isn't High risk (not in this HQ): auto-triage to Ready for admission".
- GIF: S44c-r3-F2.gif.

**F2, an HQ with no banded score** (the same HQ before its score was added):
- "and a score isn't in a band" could be unticked and ticked again (it's no longer greyed out).
- Ticked, it offered "STOP-Bang score (not in this HQ)", and the bands Low, Intermediate and High risk.
- I re-picked High risk and saved; the words were unchanged.

**T-01, the chain at phone size (390 px frame):**
- **Steps 1–2:** only Breathing was listed, with "0 of 1 sections complete".
- **Step 3:** after Breathing was finished, Pillows appeared. D Yes gave "2 of 2".
- **Step 4:** B → No hid A and C at once, with "Saving…" then "Saved".
- **Step 5:** "1 of 1", and Pillows was gone.
- **Step 6:** after a reload, B showed No and A and C were hidden.
- **Step 7:**
  - B Yes showed A empty, and A Yes showed C empty.
  - Finish with C empty gave "Response required.", as expected for a required question. I typed "Snoring" and finished.
  - Pillows was listed again, and its Start opened D empty.
- **Step 8:**
  - B No, Finish, then **Send my answers** (in-page confirm) gave "Thank you, Sam".
  - The page was never wider than the frame: `scrollWidth` 390 = `clientWidth` 390.
  - The episode is at Ready for review, and the frozen Breathing row is `{B: No}`.
- GIF: S44c-r3-T01.gif.

**T-10, steps 1–2** (MT S44c band 1515, which has no STOP-Bang):
- **Step 1:**
  - Low risk reads "…and STOP-Bang score isn't High risk (not in this HQ): auto-triage to Ready for admission".
  - Edit shows the score as "STOP-Bang score (not in this HQ)", in full.
- **Step 2:**
  - The Publish window read "This version auto-triages: ASA I–II, no red flags → Ready for admission…".
  - It published with no problem.

## Found while exploring

- **F2's original path, after a save:**
  - On a v2 draft of MT S44c band 1515, I saved Low risk with **Snore score** and band **High**. It reads "…and Snore score isn't High: auto-triage to Ready for admission".
  - When I edited it again, the list still offered STOP-Bang score (not in this HQ).
  - Switching to it cleared the band (Snore's "High" didn't carry over). Save rule with no band was refused in the form with "Rule “Low risk”: choose the score and at least one of its bands." I cancelled; v2 keeps the Snore score rule.
- **Not a defect, just how the editor works:**
  - BMI must be alone on its page. On a page that already holds a question, the BMI button and Ready-made score are greyed out with no tooltip, so I needed a new page.
  - When no Question Set is selected, the editor header shows only Rules and Publish. The buttons move left when a set is open, which made me click Rules instead of Publish once.
  - Cosmetic at most: a short tooltip on the greyed BMI button ("BMI goes on its own page") would help an author.
- **Required check:** pressing Finish with an empty required question that has just reappeared gives "Response required." This is SurveyJS's normal check and fine.

## Console and network errors

- None in the staff tabs, checked with `read_console_messages` for errors.
- The patient frame's console isn't captured by the extension.
- No failed requests were seen. Every save behaved as expected, confirmed by the stored rows.

## GIFs

- `~/Downloads/S44c-r3-T01.gif`: T-01 at phone size, up to the send confirmation.
- `~/Downloads/S44c-r3-F2.gif`: F2. Banded score, STOP-Bang still offered, re-picked with High risk, saved.

## Hand checks for Rahul

- None.
