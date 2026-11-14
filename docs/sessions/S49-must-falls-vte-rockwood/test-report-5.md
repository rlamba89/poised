# S49 test report 5 (retest after verification 3)

**12 passed, 0 failed, 0 blocked.** The 12 are verification 3's new case (T-22), the 5 regression cases and 6 happy paths.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (41 entries in `git status`, including verification 3's fixes).
- **When:** Fri 13 Nov 2026, 14:17–14:37 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was started at 14:17:18, after the fixes (`packages/clinical/src/assessments.ts` and `apps/web/src/features/episodes/assessments.ts` are from 13:07–13:08).
  - API health was ok.
- **Browser:** Claude in Chrome. The patient's pages were tested in a **390 px iframe** on a localhost page, with `scrollWidth` checked (as in reports 1 to 4).
- **How I clicked:** real clicks and typing for every step that a case checks. For long runs of Yes/No answers (the 9 clot questions, 18 "No" answers on the VTE card, the fitness HQ) I clicked the page's own buttons from a script, and I read card contents back with a script.
- **Not re-run:** T-17 and T-21 (the pair rule with a reading entered in error or precedence off; passed in reports 3 and 4), T-20, and T-01 to T-09, T-11 and T-16 (passed in report 2). Verification 3's finding 4 (the "There's no BMI now" wording) has no test case and I didn't check it.
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | Mona MT S49 1420 Pair | T-22, R-01, MUST, VTE, Morse, T-12, T-13, T-14 | Demo risk HQ, woman, 1 Jan 1980; Fill in for the patient (165 cm, 70 kg, 2 kg lost, a fall, crutches, HRT Yes) |
  | Ravi MT S49 1430 Snore | R-02 | Demo STOP-Bang HQ, man, 1 Jan 1960; Fill in for the patient |
  | Fay MT S49 1433 Fit | R-03, R-04, the Frailty Scale | Demo fitness HQ, woman, 1 Jan 1980; Fill in for the patient |

  Also a draft questionnaire, **MT S49 1437 Ready** (Alex Author), with the ready-made MUST inserted. Used for R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-22 A weight or BMI typed on MUST leaves the other for the clinician** | **Pass** | All 5 steps. See below. |
| R-01 Observations | Pass | Weight -3: refused, "Weight must be from 1 to 400 kg." Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. |
| R-02 STOP-Bang | Pass | "8 of 8 filled." S, T, O, P and N from the HQ; B 35.9 Hospital BMI; A 66 Profile; G Profile. "5 (High risk)", confirmed. The POA Summary shows "STOP-Bang 5 (High risk)" under Hospital BMI 35.9. |
| R-03 DASI | Pass | "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs". Confirmed. |
| R-04 Apfel | Pass | "3 of 4 filled": Female (Profile), smoking No and PONV Yes (HQ), only "Opioids after surgery" Missing. Yes gives "4 (PONV risk about 79%)". |
| R-05 Ready-made window | Pass | The menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; no Clinical Frailty Scale. STOP-Bang's window shows the orange "It uses the patient's BMI…" box and Insert is off. MUST's has no BMI box, the "weight-loss questions only, with no score" note and BAPEN's line. Inserted: a Group "Weight loss" with 2 questions, both **Always ask fresh**, no Calculation. |
| Happy path: MUST, saved and confirmed | Pass | See below. |
| Happy path: VTE (with T-19's check) | Pass | "11 of 30 filled". With B1 left: "29 of 30 filled", **Incomplete**, Confirm greyed out. No on B1: "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (HRT). 30.1 typed in T5: "2 (…)", T5 **Entered by clinician**, and no other item was cleared. Confirmed. |
| Happy path: Morse | Pass | 1 Yes (HQ), 3 Crutches… (HQ). 2 No, 4 No, Gait Weak (a menu of 3), Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| T-12 POA Summary | Pass | Mona: Patient reported BMI 25.7, Hospital BMI 20.1, "MUST 2 (High risk), 3.8% of body weight lost", "VTE risk 2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0" (wraps in its own column), "Morse Fall Scale 65 (High risk)". Browser Back returns to the episode page with the cards as saved. |
| Frailty Scale and its hint (T-10, in part) | Pass | "0 of 1 filled", Level Missing, no blue box. After DASI was confirmed: "For reference: DASI: 34 (Poor functional capacity), 6.9 METs (confirmed 13/11/2026, 14:34)". The menu lists the levels; 5 gives "5 (Living with Mild Frailty)", source Clinician, confirmed. I didn't add PRISMA-7 this round. |
| T-13 and T-14 | Pass | Mona's "Thank you" page and her home (388 of 388 px) say nothing about MUST, Morse, VTE, frailty, scores, BMI or readings. Val Clinician opening Mona's episode URL sees "You don't have access to this hospital". I didn't try Bea Author this round. |

### T-22: details

Mona's episode: hospital height 170 and weight 60 (Hospital BMI 20.8); her HQ gave 2 kg lost.

- **Step 1:** MUST opened "3 of 4 filled": **1 BMI 20.8, Hospital BMI**; **2 Weight now 60, Hospital weight**; 3 is 2 (HQ); 4 Missing. No on item 4: "0 (Low risk), 3.2% of body weight lost".
- **Step 2 (50 typed in item 2):** item 1's box emptied and went **Missing** (orange); item 2 **Entered by clinician**. "3 of 4 filled: answer the item highlighted." Badge **Incomplete**, Confirm greyed out, "Confirm once every item has an answer."
- **Step 3 (17.3 typed in item 1):** both **Entered by clinician**; "**2 (High risk), 3.8% of body weight lost**"; Confirm on.
  - The plan's example says "4 (High risk)". With 2 kg lost, 2 is right by T-03's cut-offs (BMI 17.3 scores 2; 2 of 52 kg is 3.8%, which scores 0). The plan's example is off, not the app.
- **Step 4:** **Undo changes** brought back 20.8 and 60 (and cleared my unsaved No on item 4). Then 18 typed in item 1 only: item 2's box emptied, **Missing**; item 1 is 18, Entered by clinician.
- **Step 5:** Undo, No on item 4, **Confirm**: "Confirmed by Cara Clinician on 13/11/2026, 14:21". Then 50 typed in item 2: item 1 emptied and Missing, badge Incomplete, "**Changed: confirm it again**", Confirm greyed out. **Undo changes** brought back the confirmed card (20.8 Hospital BMI, 60 Hospital weight, "0 (Low risk), 3.2%…", Confirmed by…).

### Happy path: MUST, saved and confirmed

- I typed 50 in item 2 on the confirmed card and clicked **Save** (the BMI Missing). After a reload: "Not confirmed", Incomplete, **1 BMI still Missing**, 2 Weight now 50 Entered by clinician. The saved typed weight didn't bring the hospital BMI back.
- 17.3 typed in item 1, **Confirm**: "2 (High risk), 3.8% of body weight lost", "Confirmed by Cara Clinician on 13/11/2026, 14:27". The POA Summary shows the same.

## Failures in detail

None.

## Found while exploring

1. **No note on a MUST card confirmed on typed values (a question, not a failure).** Mona's MUST was confirmed on a typed BMI and weight (17.3, 50). I then corrected the hospital weight from 60 to 58 (Hospital BMI 20.1). The card stays "2 (High risk), 3.8%…" with no yellow "The hospital BMI is now…" box. That seems right, since neither value came from the readings, but T-04's note doesn't cover this case either way.
2. **An incomplete MUST card can be saved.** With item 1 Missing, Save is on (Confirm is off). The save worked and a reload showed the same card.
3. **The New episode form's Health questionnaire menu can't be searched by typing.** Typing "Demo risk" into it did nothing. Locally the list now holds several thousand E2E questionnaires, so I picked the HQ by scrolling. Older than S49.
4. **Data I left:**
   - Mona: a heart rate of 220 (saved anyway); the weight corrected to 58; MUST confirmed on typed values; VTE confirmed with 30.1 typed in T5; Morse confirmed.
   - Fay: Apfel's Yes on item 4 wasn't saved; DASI and the Frailty Scale are confirmed.
   - Ravi: STOP-Bang confirmed.
   - The draft questionnaire MT S49 1437 Ready.
   - Alex Author was the last staff user signed in; the patient (Mona) is signed out. I closed my tab.
5. **Two of my own clicks missed.** After a card changed height, one click landed beside a box and one on plain text at the top of the page. Nothing was changed by either, and I redid both steps.

## Environment

- Pages loaded in about 1–2 s throughout.
- Trust A's "Observations take precedence in scores" was left as it was (on); I didn't open Settings.

## Console and network errors

- **Console:** no errors or warnings. Tracking started at about 14:28, after T-22.
- **Network:** I listed the 112 `/api/` requests from 14:28 to 14:34 (Mona's later steps, Ravi's and Fay's episodes, their sends, readings, assessments and Confirms): every one was 2xx.
  - Before 14:28 (T-22 and R-01's refused weight) and after 14:34 (R-05, T-13, T-14) I have only what the pages showed.

## GIFs

- `~/Downloads/S49-retest5-T22.gif`: T-22, from adding MUST through steps 1 to 5 and the final Undo.
