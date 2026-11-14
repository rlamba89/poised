# S49 test report 8 (retest after test report 7)

**14 passed, 0 failed, 0 blocked.** The 14 are report 7's failed case (T-24, with its changed steps 2 and 3), the 5 regression cases, and 8 happy paths and earlier cases.

Report 7's F-1 is fixed. A box the pairing rule marks Missing is now empty, showing its grey hint ("BMI" or "kg"). I checked this on two episodes, after clicking away, and on a confirmed card.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (45 entries in `git status`).
- **When:** Sat 14 Nov 2026, 11:15–11:31 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was started at 11:14:43, just before the round. API health was ok.
- **Browser:** Claude in Chrome. Desktop cases ran at 1280×900.
  - The patient's pages ran at **500×701**, because Chrome won't make the window narrower than 500 px (asked for 390×844). There was no sideways scrolling: `scrollWidth` was 500.
- **How I clicked:** real clicks and typing for every step a case checks.
  - A script answered the 9 clot questions (and Xan's whole page 2), and the 18 missing VTE items with "No". B1, the item T-19 checks, I clicked by hand.
  - I read some card text back with a script.
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | **Wyn MT S49 1120 Empty** (new) | T-01, T-24 step 2, R-01, Morse, VTE, T-12, T-14, T-15 step 2 | Demo risk HQ, woman, 1 Jan 1980. **The patient answered through the invite link**: 165 cm, 70 kg, No then Yes to weight loss, 2 kg, a fall, crutches, HRT Yes, blood thinner Yes, the other 7 No. Readings 170 cm, 60 kg. |
  | **Xan MT S49 1125 Mirror** (new) | T-24 steps 1–5, exploring | Demo risk HQ, woman, 1 Jan 1980; **Fill in for the patient** (165 cm, 70 kg, Yes, 2 kg, a fall, crutches, all 9 clot questions No). Readings 170 cm, 60 kg. |
  | Ravi MT S49 1430 Snore (report 5's) | R-02 | Demo STOP-Bang HQ |
  | Fay MT S49 1433 Fit (report 5's) | R-03, R-04, the Frailty Scale's hints | Demo fitness HQ |

  Also report 5's draft questionnaire **MT S49 1437 Ready** (Alex Author), for R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-24 On MUST, a box typed in and put back counts as untouched** | **Pass** | Every step as expected, including the changed steps 2 and 3: the Missing box is **empty**. See below. |
| T-01 The patient answers the Demo risk HQ (at 500 px, see above) | Pass | No first: no "how much" question. Yes: it appears. Next with nothing answered on page 2: "Response required." on both. The send says "Thank you, Wyn". The home page shows only the HQ (Sent, View), Information and contact details: nothing about assessments, scores or bands. |
| T-02 MUST opens 3 of 4 | Pass | Wyn and Xan: "3 of 4 filled: answer the item highlighted.", 20.8 Hospital BMI, 60 Hospital weight, 2 (HQ), item 4 Missing (orange) with the full wording. Confirm and Save off. No: "0 (Low risk), 3.2% of body weight lost". |
| R-01 Observations | Pass | Weight -3 (Enter): "Weight must be from 1 to 400 kg.", nothing saved. Heart rate 220 (Enter): "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. Enter in the value box saves a reading. |
| R-02 STOP-Bang | Pass | Ravi's card: "8 of 8 filled.", S, T, O, P, N from the HQ, B Hospital BMI (35.9), A 66 and G Male from Profile, "5 (High risk)", confirmed 13/11/2026, 14:31. Ravi's POA Summary shows "STOP-Bang 5 (High risk)". I didn't make a new STOP-Bang episode. |
| R-03 DASI | Pass | Fay's card: "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs", confirmed. |
| R-04 Apfel | Pass | Fay's card: "4 of 4 filled.", Female (Profile), smoking from the HQ, "4 (PONV risk about 79%)", confirmed 13/11/2026, 19:15. I didn't see it open at 3 of 4 again this round. |
| R-05 Ready-made window | Pass | The Score menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; there's no Clinical Frailty Scale. STOP-Bang shows the orange "It uses the patient's BMI…" box, and Insert is off. MUST has no BMI box, has the "weight-loss questions only, with no score" note and BAPEN's lines, and Insert is on. I cancelled without inserting. |
| Morse (T-07) | Pass | Wyn: "2 of 6 filled": 1 Yes (HQ), 3 Crutches… (HQ). Then 2 No, 4 No, Gait Weak, Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| VTE (T-08, with T-19's check) | Pass | Wyn: "11 of 30 filled"; T2 46 (Profile); T5 20.8 (Hospital BMI). With B1 left: "29 of 30 filled", Incomplete, B1 highlighted. No on B1: "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1" (HRT; the blood thinner). Confirmed. |
| Clinical Frailty Scale | Pass | Wyn: added, "0 of 1 filled", left unconfirmed; no "For reference" box (no DASI or PRISMA-7 on this episode). Fay's confirmed card still shows "For reference" with DASI's 34 and PRISMA-7's 1. |
| T-12 POA Summary | Pass | Wyn, both tabs: Patient reported BMI 25.7, Hospital BMI 20.8, "MUST 0 (Low risk), 3.8% of body weight lost", "Morse Fall Scale 65 (High risk)", "VTE risk 1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1" (wraps within its own column), "Clinical Frailty Scale Not confirmed". |
| T-15 step 2 Back | Pass | Back from the POA Summary returns to the episode page; a reload shows the four cards as saved and confirmed. |
| T-14 Another hospital's or trust's staff | Pass | Val Clinician and Bea Author, opening Wyn's episode URL, each see "You don't have access to this hospital". The episode's API call from their pages was a 404. |

### T-24: what was seen at each step

Xan's episode: hospital height 170 and weight 60 (Hospital BMI 20.8). The HQ said Yes to weight loss, 2 kg. I saved the card after No on item 4, so that Undo kept that answer.

- **Step 1:** "3 of 4 filled", 20.8 Hospital BMI, 60 Hospital weight, 2 (HQ), item 4 Missing. No: "0 (Low risk), 3.2% of body weight lost".
- **Step 2:** 36 typed in item 1: Entered by clinician; item 2 empty ("kg") and Missing.
  - 20.8 typed back: item 1 **Hospital BMI**, item 2 **60 Hospital weight**, the score is back.
  - 50 typed in item 2: **item 1 Missing and its box empty, showing the grey "BMI" hint**. Item 2 Entered by clinician, badge **Incomplete**, Confirm off.
  - After clicking elsewhere, the box's value is still `""`.
  - The same on Wyn's episode.
- **Step 3:** Undo. 61 typed in item 2 (item 1 emptied and Missing), then 60 back: both hospital sources again, score back.
  - 17.3 typed in item 1: **item 2 Missing and its box empty ("kg" hint)**, badge Incomplete. The box's value read back as `""`.
- **Step 4:** Undo. 50 typed in item 2: item 1 empty and Missing.
  - Clicked into item 1, typed 1: "Between 5 and 150", Save off.
  - Deleted it: **no "Enter a BMI" message, Save on**.
- **Step 5:** Undo, **Confirm**: "Confirmed by Cara Clinician on 14/11/2026, 11:23".
  - The episode had been auto-triaged, so I signed in as **Sue Super**, clicked **Re-open HQ review** (with a reason) and **Validate** on the set.
  - I changed the weight lost from 2 to **3** ("Patient answered: 2" shows under it; "Saved") and went back.
  - The MUST card's yellow box reads **"The HQ now gives 3 kg for Unplanned weight loss in the past 3–6 months (2 kg when confirmed)."** with "The score stays as confirmed. To use a new value, change the item and confirm again."
- **The fix on a confirmed card too:** on Xan's confirmed card I typed 36 then 20.8 back, then 50 in item 2.
  - Item 1 was Missing with an **empty** box, and the card said "Changed: confirm it again".
  - **Undo changes** brought back the confirmed 20.8 / 60 / 2 / No, and the yellow line.

## Failures in detail

None.

## Found while exploring

1. **Both boxes typed keeps both.** On Xan's confirmed card:
   - 17.3 in item 1 (item 2 emptied), then 52 typed into the empty item 2.
   - Result: both Entered by clinician, "2 (High risk), 3.7% of body weight lost" (BMI 17.3 scores 2; 2 of 54 kg scores 0).
   - Undo brought the confirmed card back.
2. **A saved weight can't be emptied.** On Wyn's card, I clicked Save by mistake while item 1 was Missing and item 2 was 50.
   - Save stored it: item 1 stayed Missing with an empty box after a reload. This matches report 7's note that what's stored is right.
   - Then I emptied item 2's box: "Enter a number" in red, Save off. **Undo changes** put 50 back.
   - That's the "a box that had a value asks for one" rule. A clinician who wants the hospital's numbers back on a saved card has to type them in: Undo goes back only to the saved state.
   - (Because of this mis-click, I ran T-24 steps 3–5 on a second episode, Xan.)
3. **While a confirmed card has unsaved changes, the yellow "The HQ now gives 3 kg…" line goes away.** Item 3 still shows 2 (HQ). The line comes back after **Undo changes**.
   - Report 7 (item 2) saw that Save then takes the HQ's 3 kg. So a clinician editing the card doesn't see the new HQ value until they save.
   - This may be as designed. I'm noting it because the line's advice is "change the item and confirm again".
4. **A clinician can't do T-24 step 5 alone on an auto-triaged episode** (as in report 7, item 1). Re-open HQ review is a super clinician's (Sue Super).
5. **The Gait menu's open list still starts right of its box** (cosmetic, in pending.md).
6. **Data I left:**
   - **Wyn:** readings 170 cm, 60 kg and a heart rate of 220 (saved anyway). MUST, Morse and VTE confirmed. The Clinical Frailty Scale added, unconfirmed. Auto-triaged to Ready for admission, not marked Checked.
   - **Xan:** MUST confirmed (20.8 / 60 / 2 / No). HQ review re-opened by Sue Super, so it's at **Ready for review**. The clinician's copy of the set has 3 kg and isn't validated.
   - Ravi, Fay and the draft MT S49 1437 Ready: unchanged.
   - Cara Clinician was the last user signed in. I closed my tab.

## Console and network errors

- **Console:** no errors or warnings. Tracking started at about 11:29, after R-05.
- **Network:** I listed the 37 `/api/` requests from about 11:29 to 11:30: the sign-in, the episode list, both episode pages with their HQ, tasks, information, observations and assessments. Every one was 2xx.
  - The two 404s were my own `fetch` of Wyn's episode while signed in as Val and as Bea (T-14).
  - Before that I have only what the pages showed: no error messages besides R-01's expected refusal.
- To find report 5's episodes among about 4,000, I read Hospital A1's episode list once through the API from the signed-in page (Cara). I didn't use the database.

## GIFs

- `~/Downloads/S49-retest8-T24.gif`: from adding MUST on Wyn's episode, through step 2 there and the mis-clicked Save, to Xan's T-24 steps 1–4 (50 frames, the recorder's limit).
