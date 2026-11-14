# S49 test report 6 (retest after verification 4)

**12 passed, 0 failed, 0 blocked.** The 12 are verification 4's new case (T-23), the 5 regression cases and 6 happy paths.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (43 entries in `git status`, including verification 4's fixes).
- **When:** Fri 13 Nov 2026, 19:08–19:18 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was started at 19:07:45, just before the round. API health was ok; the evaluate service answered (`clinical@31046e6b03bc-dirty`).
- **Browser:** Claude in Chrome, desktop size. No patient pages at phone size this round (the patient's side didn't change; I used **Fill in for the patient**).
- **How I clicked:** real clicks and typing for every step a case checks. For long runs of Yes/No answers (the 9 clot questions, 18 "No" answers on the VTE card) I clicked the page's own buttons from a script, and I read some card contents back with a script.
- **Not re-run:** T-01 to T-11, T-13, T-15 to T-22 (passed in reports 2 to 5). Verification 4's finding 2 (a card confirmed before verification 3's fix, with a typed weight beside a pre-filled BMI) can't be made through the app now, so I didn't check it. Finding 3 has nothing to see in the browser.
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | **Tess MT S49 1912 Typed** (new) | T-23, R-01, Morse, VTE, T-12, T-14 | Demo risk HQ, woman, 1 Jan 1980; Fill in for the patient (165 cm, 70 kg, No to weight loss, a fall, crutches, HRT Yes, the rest No) |
  | Ravi MT S49 1430 Snore (report 5's) | R-02, exploring | Demo STOP-Bang HQ, man, 1 Jan 1960 |
  | Fay MT S49 1433 Fit (report 5's) | R-03, R-04, the Frailty Scale's hints | Demo fitness HQ, woman, 1 Jan 1980 |

  Also report 5's draft questionnaire **MT S49 1437 Ready** (Alex Author), for R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-23 On MUST, the hospital's own number typed back after the other was typed counts** | **Pass** | All 5 steps. See below. |
| R-01 Observations | Pass | Weight -3: refused, "Weight must be from 1 to 400 kg." Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. Enter in the value box saves a reading. |
| R-02 STOP-Bang | Pass | On Ravi's card, confirmed in report 5: "8 of 8 filled.", S, T, O, P, N from the HQ, B 35.9 Hospital BMI, A 66 and G Male from Profile, "5 (High risk)", "Confirmed by Cara Clinician on 13/11/2026, 14:31". N changed to Yes: "6 (High risk)", "Changed: confirm it again"; back to No: confirmed again. The POA Summary shows "STOP-Bang 5 (High risk)". I didn't make a new STOP-Bang episode this round. |
| R-03 DASI | Pass | Fay's card: "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs", confirmed. |
| R-04 Apfel | Pass | "3 of 4 filled": Female (Profile), smoking No and PONV Yes (HQ), only "Opioids after surgery" Missing. Yes gives "4 (PONV risk about 79%)". Confirmed. |
| R-05 Ready-made window | Pass | The menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; no Clinical Frailty Scale. STOP-Bang's window shows the orange "It uses the patient's BMI…" box and Insert is off. Morse's has no BMI box, the "patient's two questions only… with no score" note, "© Janice Morse…", and Insert is on. I cancelled without inserting. |
| Happy path: MUST, confirmed and reloaded | Pass | T-23's steps 4 and 5. |
| Happy path: Morse | Pass | "2 of 6 filled": 1 Yes (HQ), 3 Crutches… (HQ). 2 No, 4 No, Gait Weak (a menu of 3), Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| Happy path: VTE (with T-19's check) | Pass | "11 of 30 filled"; T2 shows 46 (Profile); T5 shows the patient's BMI ("Patient reported BMI"), as the hospital height was entered in error by then. With B1 left: "29 of 30 filled", **Incomplete**. No on B1: "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (HRT). Confirmed. |
| T-12 POA Summary | Pass | Tess: Patient reported BMI 25.7, Hospital BMI "Not measured yet…", "MUST 0 (Low risk), 0% of body weight lost", "Morse Fall Scale 65 (High risk)", "VTE risk 1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (wraps in its own column). |
| Frailty Scale's hints (T-10 steps 3 and 4) | Pass | Fay's confirmed card showed "For reference: DASI: 34 (Poor functional capacity), 6.9 METs (confirmed 13/11/2026, 14:34)". I added PRISMA-7 ("7 of 7 filled", "1 (Not at increased risk)") and confirmed it: a second line appeared, "PRISMA-7: 1 (Not at increased risk) (confirmed 13/11/2026, 19:15)". |
| T-14 Another hospital's or trust's staff | Pass | Val Clinician and Bea Author, opening Tess's episode URL, each see "You don't have access to this hospital". |

### T-23: details

Tess's episode: hospital height 170 and weight 60 (Hospital BMI 20.8); her HQ said No to weight loss (0 kg).

- **Step 1:** MUST opened "3 of 4 filled": **1 BMI 20.8, Hospital BMI**; **2 Weight now 60, Hospital weight**; 3 is 0 (HQ); 4 Missing. No on item 4: "0 (Low risk), 0% of body weight lost".
- **Step 2:** 30 typed in item 1: item 2's box emptied and went **Missing** (orange), badge Incomplete, Confirm off. Then **60 typed in item 2**: item 2 reads **Entered by clinician** (not Missing), "4 of 4 filled.", badge "0 (Low risk), 0% of body weight lost", **Confirm on**.
- **Step 3:** **Undo changes** brought back 20.8 Hospital BMI and 60 Hospital weight (and cleared my unsaved No on item 4, so I clicked No again). 61 typed in item 2: item 1 emptied, Missing. Then **20.8 typed in item 1**: both **Entered by clinician**, "0 (Low risk), 0% of body weight lost", Confirm on.
- **Step 4:** Undo. Height marked **Entered in error** (with a reason): Hospital BMI "Not measured yet…", item 1 **Missing**, item 2 still 60 Hospital weight. 20.8 typed in item 1: item 1 Entered by clinician, and item 2 emptied and went Missing (the pairing rule). **60 typed in item 2**: both **Entered by clinician**. No on item 4, **Confirm**: "Confirmed by Cara Clinician on 13/11/2026, 19:12".
- **Step 5:** after a reload the card is the same: 20.8 and 60, both Entered by clinician, "0 (Low risk), 0% of body weight lost", Confirmed by…; nothing Missing.

## Failures in detail

None.

## Found while exploring

1. **Verification 4's own example now works.** On Ravi's unsaved MUST card (35.9 Hospital BMI, 110 Hospital weight): 35.9 typed back in item 1 changed nothing (still Hospital BMI); 36 typed: item 2 Missing; **110 typed in item 2: Entered by clinician**. Emptying item 1 then says "Enter a BMI", Missing, and Save and Confirm are off. **Undo changes** brought back 35.9 and 110 with their hospital sources. Nothing was saved.
2. **A confirmed card with both numbers typed keeps its state.** On Tess's confirmed MUST, 61 typed in item 2 says "Changed: confirm it again" (item 1 stays 20.8, Entered by clinician); 60 typed back returns to "Confirmed by Cara Clinician…". Nothing was saved.
3. **The Gait menu's list opens a little to the right of its box (cosmetic, maybe older than this round).** On the Morse card the closed menu starts at the left of its column, but the open list of three choices starts about 110 px further right. It's readable and each choice can be picked.
4. **The New episode form's Health questionnaire menu still isn't searchable by typing** (report 5, item 3; older than S49). The Questionnaires page has a search box, which I used for R-05.
5. **Data I left:**
   - Tess: height 170 entered in error, weight 60, a heart rate of 220 (saved anyway); MUST, Morse and VTE confirmed. The episode was auto-triaged to Ready for admission and isn't marked Checked.
   - Fay: Apfel now confirmed (Yes on item 4); PRISMA-7 added and confirmed.
   - Ravi: unchanged (STOP-Bang confirmed; the MUST form still unsaved).
   - Cara Clinician was the last user signed in. I closed my tabs.
6. **One of my own mistakes.** On the New episode form my first click for the date of birth landed in the NHS number box after the page scrolled. I cleared it and typed the date again before creating the episode.

## Environment

- Pages loaded in about 1–2 s throughout.
- Trust A's "Observations take precedence in scores" was left as it was (on); I didn't open Settings.
- To find report 5's episodes among several thousand, I read Hospital A1's episode list once through the API from the signed-in page (Cara). Docker wasn't on my shell's path, so I didn't use the database.

## Console and network errors

- **Console:** no errors or warnings. Tracking started at about 19:12, after T-23's Confirm.
- **Network:** I listed the 86 `/api/` requests from about 19:15 to 19:18 (the sign-ins as Val, Bea, Alex and Cara, the Questionnaires search, the editor, Ravi's and Tess's episode pages): every one was 2xx. Before 19:15 (T-23, the readings, Morse, VTE, the POA Summary, Fay's cards) I have only what the pages showed: no error messages besides R-01's expected refusal.

## GIFs

- `~/Downloads/S49-retest6-T23.gif`: T-23, from adding MUST through steps 1 to 4 and the Confirm (47 frames).
