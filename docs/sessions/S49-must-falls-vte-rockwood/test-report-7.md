# S49 test report 7 (retest after verification 5)

**11 passed, 1 failed, 0 blocked.** The 12 are verification 5's new case (T-24), the changed T-23, the 5 regression cases and 5 happy paths.

The one failure is Minor: every result T-24 writes down was met, but a box the pairing rule marks **Missing** keeps showing its old number when that box was typed in earlier.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (44 entries in `git status`, including verification 5's fixes).
- **When:** Fri 13 Nov 2026, 20:29–20:45 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was started at 20:29:09, just before the round. API health was ok; the evaluate service answered (`clinical@31046e6b03bc-dirty`).
- **Browser:** Claude in Chrome, desktop size. No patient pages at phone size this round (the patient's side didn't change; I used **Fill in for the patient**).
- **How I clicked:** real clicks and typing for every step a case checks. For long runs of Yes/No answers (the 9 clot questions, 18 "No" answers on the VTE card) I clicked the page's own buttons from a script, and I read some card contents back with a script.
- **Not re-run:** T-01 to T-11, T-13, T-16 to T-22 (passed in reports 2 to 5).
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | **Tia MT S49 2032 Back** (new) | T-24, R-01 (the refusal) | Demo risk HQ, woman, 1 Jan 1980; Fill in for the patient (165 cm, 70 kg, Yes to weight loss, 2 kg, a fall, crutches, HRT Yes, the rest No); readings 170 cm, 60 kg |
  | **Una MT S49 2040 Kept** (new) | T-23, R-01 (the warning), Morse, VTE, T-12, T-14 | The same, but No to weight loss; readings 170 cm, 60 kg |
  | Ravi MT S49 1430 Snore (report 5's) | R-02, exploring | Demo STOP-Bang HQ, man, 1 Jan 1960 |
  | Fay MT S49 1433 Fit (report 5's) | R-03, R-04, the Frailty Scale's hints | Demo fitness HQ, woman, 1 Jan 1980 |

  Also report 5's draft questionnaire **MT S49 1437 Ready** (Alex Author), for R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-24 On MUST, a box typed in and put back counts as untouched** | **Fail (Minor)** | Sources, badges, Save and the yellow line are all as expected. But in steps 2 and 3 the box marked **Missing** still shows the old number (20.8, then 60). See F-1. |
| **T-23 On MUST, the hospital's own number typed back after the other was typed counts** | **Pass** | All 5 steps, with the changed step 4: the hospital's weight is kept. See below. |
| R-01 Observations | Pass | Weight -3 (Enter): refused, "Weight must be from 1 to 400 kg." Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. Enter in the value box saves a reading. |
| R-02 STOP-Bang | Pass | Ravi's card, confirmed in report 5, still reads "8 of 8 filled.", S, T, O, P, N from the HQ, B 35.9 Hospital BMI, A 66 and G Male from Profile, "5 (High risk)", "Confirmed by Cara Clinician on 13/11/2026, 14:31". I didn't make a new STOP-Bang episode. |
| R-03 DASI | Pass | Fay's card: "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs", confirmed. |
| R-04 Apfel | Pass | Fay's card, confirmed in report 6: "4 of 4 filled.", Female (Profile), smoking and PONV from the HQ, opioids Clinician, "4 (PONV risk about 79%)". I didn't see it open at 3 of 4 again this round. |
| R-05 Ready-made window | Pass | The menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; no Clinical Frailty Scale. STOP-Bang's window shows the orange "It uses the patient's BMI…" box and Insert is off. MUST's has no BMI box, the "weight-loss questions only, with no score" note, BAPEN's permission line and "© BAPEN…", and Insert is on. I cancelled without inserting. |
| Happy path: Morse | Pass | Una: "2 of 6 filled": 1 Yes (HQ), 3 Crutches… (HQ). 2 No, 4 No, Gait Weak (a menu of 3), Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| Happy path: VTE (with T-19's check) | Pass | Una: "11 of 30 filled"; T5 shows 25.7, the patient's BMI, as the hospital height was entered in error by then. With B1 left: "29 of 30 filled", **Incomplete**, B1 highlighted. No on B1: "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (HRT). Confirmed. |
| T-12 POA Summary | Pass | Una, both tabs: Patient reported BMI 25.7, Hospital BMI "Not measured yet…", "MUST 0 (Low risk), 0% of body weight lost", "Morse Fall Scale 65 (High risk)", "VTE risk 1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (wraps in its own column). The browser's Back returns to the episode page (T-15 step 2). |
| Frailty Scale's hints | Pass | Fay's confirmed card still shows "For reference", "DASI: 34 (Poor functional capacity), 6.9 METs (confirmed 13/11/2026, 14:34)" and "PRISMA-7: 1 (Not at increased risk) (confirmed 13/11/2026, 19:15)". |
| T-14 Another hospital's or trust's staff | Pass | Val Clinician and Bea Author, opening Una's episode URL, each see "You don't have access to this hospital". |

### T-24: what was seen at each step

Tia's episode: hospital height 170 and weight 60 (Hospital BMI 20.8); her HQ said Yes to weight loss, 2 kg.

- **Step 1:** MUST opened "3 of 4 filled": 20.8 Hospital BMI, 60 Hospital weight, 2 (HQ), item 4 Missing. No on item 4: "0 (Low risk), 3.2% of body weight lost".
- **Step 2:** 36 typed in item 1: Entered by clinician, item 2 emptied and Missing. **20.8 typed back: item 1 reads Hospital BMI**, item 2 is 60 Hospital weight again, the score is back. Then 50 typed in item 2: **item 1 is Missing** (orange), item 2 Entered by clinician, badge **Incomplete**, Confirm off. As expected, except that item 1's box still shows 20.8 (F-1).
- **Step 3:** Undo. 61 typed in item 2 (item 1 emptied, Missing), then 60 back: both hospital sources again. Then 17.3 typed in item 1: **item 2 is Missing**, badge Incomplete. As expected, except that item 2's box still shows 60 (F-1).
- **Step 4:** Undo. 50 typed in item 2: item 1's box emptied, Missing. I clicked in item 1's empty box and typed 1: "Between 5 and 150", Save off. Deleted it: **no "Enter a BMI" message, and Save is on.**
- **Step 5:** Undo, **Confirm**: "Confirmed by Cara Clinician on 13/11/2026, 20:34". The episode had been auto-triaged, so its HQ review was closed and a clinician can't re-open it. I signed in as **Sue Super**, clicked **Re-open HQ review** (with a reason), opened the set with **Validate**, changed the weight lost from 2 to 3 (it saved) and went back. The MUST card's yellow box reads: **"The HQ now gives 3 kg for Unplanned weight loss in the past 3–6 months (2 kg when confirmed)."** and "The score stays as confirmed. To use a new value, change the item and confirm again."

### T-23: what was seen at each step

Una's episode: hospital height 170 and weight 60; her HQ said No to weight loss (0 kg). I saved the card after answering No on item 4, so that Undo kept that answer.

- **Step 1:** 20.8 Hospital BMI, 60 Hospital weight, 0 (HQ). No on item 4: "0 (Low risk), 0% of body weight lost".
- **Step 2:** 30 typed in item 1: item 2 emptied, Missing, Incomplete, Confirm off. 60 typed in item 2: **Entered by clinician**, "0 (Low risk), 0% of body weight lost", Confirm on.
- **Step 3:** Undo. 61 typed in item 2, then 20.8 typed in item 1: both **Entered by clinician**, a score, Confirm on.
- **Step 4 (changed):** Undo. Height marked **Entered in error**: Hospital BMI "Not measured yet…", item 1 Missing, item 2 still 60 Hospital weight. 20.8 typed in item 1: item 1 **Entered by clinician** and **item 2 stays 60 Hospital weight**, "4 of 4 filled.", Confirm on. **Confirm**: "Confirmed by Cara Clinician on 13/11/2026, 20:39".
- **Step 5:** after a reload the card is the same: 20.8 Entered by clinician, 60 Hospital weight, the same result, confirmed; nothing Missing.

## Failures in detail

### F-1 (Minor): a box marked Missing keeps showing its old number, if that box was typed in before

- **Where:** the MUST card on an episode page, signed in as Cara Clinician (also seen as Sue Super). Tia MT S49 2032 Back: `/o/…00fa/h/…000a/episodes/728ea442-68ca-4807-acd1-a3628121333a`.
- **Steps:**
  1. On a MUST card showing 20.8 Hospital BMI and 60 Hospital weight, type **36** in item 1, then type **20.8** back (item 1 reads Hospital BMI again).
  2. Type **50** in item 2.
- **Expected:** item 1 is Missing and its box is empty (showing the grey "BMI" hint), as it is when item 1 was never typed in (T-22 step 2, and T-24 step 4 this round).
- **Actual:** item 1 is orange with the **Missing** badge, the card says "3 of 4 filled" and Incomplete, but the box still shows **20.8** as a real value. Clicking elsewhere doesn't clear it.
- **The mirror** (T-24 step 3): 61 then 60 in item 2, then 17.3 in item 1: item 2 is Missing but its box still shows **60**.
- **Also on a confirmed card** (Tia's, after step 5) and on Ravi's older unsaved card (35.9 and 110).
- **What's stored is right.** **Save** in that state stores item 1 as missing: after Save the box is empty, and after a reload too. Typing the same number over the one shown turns the item into Entered by clinician with a score. **Undo changes** clears it.
- **Why it matters:** a clinician sees a number in a row that says Missing, and could read the 20.8 as still in use beside the new weight. The score isn't wrong, as the card is Incomplete and can't be confirmed.
- **Console and network:** nothing.
- **GIF:** `S49-retest7-T24.gif` (it shows step 2's state).

## Found while exploring

1. **A clinician can't do T-24 step 5 alone on an auto-triaged episode.** A Demo risk HQ with these answers is auto-triaged to Ready for admission on send, so the set offers only **View** ("The HQ review is complete, so these answers can't be changed."). **Re-open HQ review** is offered to a super clinician (Sue Super), not to Cara. That's the existing rule, not S49's; the test plan could say so.
2. **After the re-open, saving the half-typed MUST card picked up the HQ's new 3 kg.** On Tia's card (confirmed with 2 kg, yellow line showing 3 kg), I typed a new weight and clicked Save: item 3 then showed 3 (HQ) and the card was "Not confirmed". That matches "change the item and confirm again".
3. **The Gait menu's list still opens to the right of its box** (report 6, item 3; cosmetic). The closed menu starts at the left of its column and the open list about 110 px further right.
4. **The New episode form's Health questionnaire menu still isn't searchable by typing** (report 5, item 3; older than S49).
5. **Data I left:**
   - Tia: marked Checked by Cara, then re-opened by Sue Super, so it's at **Ready for review**; the clinician's copy of the set has 3 kg and isn't validated. MUST is saved, not confirmed: weight 50 (Entered by clinician), BMI Missing.
   - Una: height 170 entered in error, weight 60, a heart rate of 220 (saved anyway); MUST, Morse and VTE confirmed. Auto-triaged to Ready for admission, not marked Checked.
   - Ravi and Fay: unchanged (I undid my typing on Ravi's MUST card).
   - Cara Clinician was the last user signed in. I closed my tab.

## Environment

- Pages loaded in about 1–2 s throughout.
- Trust A's "Observations take precedence in scores" was left as it was (on); I didn't open Settings.
- The episode list shows e2e episodes created at 20:27–20:28, just before `make dev` restarted; nothing new from e2e appeared during the round.
- To find report 5's episodes among about 4,000, I read Hospital A1's episode list once through the API from the signed-in page (Cara). I didn't use the database.

## Console and network errors

- **Console:** no errors or warnings. Tracking started at about 20:39, after T-23's Confirm.
- **Network:** I listed the 144 `/api/` requests from about 20:41 to 20:44 (the episode pages, the sign-ins as Val, Bea, Alex and Cara, the Questionnaires search, the editor): every one was 2xx. Before that (T-24, T-23, the readings, Morse, VTE) I have only what the pages showed: no error messages besides R-01's expected refusal.

## GIFs

- `~/Downloads/S49-retest7-T24.gif`: T-24, from adding MUST through steps 1 to 4 and the Confirm (50 frames).
