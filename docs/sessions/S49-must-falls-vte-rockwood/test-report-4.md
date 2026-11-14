# S49 test report 4 (retest after verification 2)

**13 passed, 0 failed, 0 blocked.** The 13 are verification 2's new cases (T-19, T-20, T-21), the 5 regression cases and 5 happy paths.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (39 entries in `git status`, including verification 2's fixes).
- **When:** Fri 13 Nov 2026, 12:12–12:34 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was started at 12:12:33, after the fixes (the changed files are from 11:56–11:57).
  - API health was ok.
- **Browser:** Claude in Chrome.
  - As in reports 1 to 3, `resize_window` had no effect.
  - The patient pages were tested in a **390 px iframe** on a localhost page, with `scrollWidth` checked.
- **How I clicked:** real clicks and typing for every step that a case checks. For long runs of Yes/No answers (the 9 clot questions, the 18 "No" answers on a VTE card, the fitness HQ) I clicked the page's own buttons from a script, and I read card contents back with a script.
- **What I ran:**
  - T-19, T-20 and T-21, each step as written;
  - R-01 to R-05, on new episodes;
  - happy paths: MUST, Morse, the POA Summary (T-12), the Frailty Scale and its hints (T-10), and the patient's pages (T-13).
- **Not re-run by hand:** T-17 and T-18 (passed in report 3). T-01 to T-09, T-11, T-14 and T-16 (passed in report 2).
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | Tess MT S49 1215 Fresh (1st) | T-19, T-20 step 1, MUST, Morse, T-12, T-13 | Demo risk HQ, woman, 1 Jan 1980; the patient through the invite at 390 px |
  | Tess MT S49 1215 Fresh (2nd) | T-20 step 2 | Demo risk HQ, the same patient record; opened through the new invite at 390 px, not sent |
  | Pia MT S49 1224 Pair | T-21, R-01 | MT S49 1222 Weight MUST, woman, 1 Jan 1975; Fill in for the patient (68 kg, no weight loss) |
  | Fern MT S49 1230 Fit | R-03, R-04, T-10 | Demo fitness HQ, woman, 1 Jan 1980; Fill in for the patient |
  | Ray MT S49 1233 Snore | R-02 | Demo STOP-Bang HQ, man, 1 Jan 1960; Fill in for the patient |

  Also a published questionnaire, **MT S49 1222 Weight MUST** (Alex Author): the library's "What do you weigh now?" and the ready-made MUST, no height. Used for T-21 and R-05.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-19 VTE stays Incomplete until every item is answered** | **Pass** | See below. |
| **T-20 Recent-time questions aren't copied into the next HQ** | **Pass** | See below. |
| **T-21 MUST, precedence off, the hospital's pair** | **Pass** | See below. |
| R-01 Observations | Pass | Weight -3: refused, "Weight must be from 1 to 400 kg." (`POST …/observations` 400). Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. |
| R-02 STOP-Bang | Pass | "8 of 8 filled." S, T, O, P and N from the HQ; B 34.9 Hospital BMI; A 66 Profile; G Male Profile. "4 (High risk)", confirmed. The POA Summary shows "STOP-Bang 4 (High risk)" under Hospital BMI 34.9. |
| R-03 DASI | Pass | "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs". |
| R-04 Apfel | Pass | "3 of 4 filled": Female (Profile), smoking No and PONV Yes (HQ), only "Opioids after surgery" Missing. Yes gives "4 (PONV risk about 79%)". |
| R-05 Ready-made window | Pass | The menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; no Clinical Frailty Scale. STOP-Bang's window shows the orange "It uses the patient's BMI…" box and Insert is off; MUST's has no BMI box, the "weight-loss questions only, with no score" note and BAPEN's line. |
| Happy path: MUST | Pass | See below. |
| Happy path: Morse | Pass | "2 of 6 filled: answer the 4 items highlighted." 1 Yes (HQ), 3 Crutches… (HQ). 2 No, 4 No, Gait Weak (a menu of 3), Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| T-12 POA Summary | Pass | Tess: under Hospital BMI 20.8, "VTE risk 2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0", "MUST 1 (Medium risk), 7.7% of body weight lost", "Morse Fall Scale 65 (High risk)". The VTE value wraps in its own column. Browser Back returns to the episode page. |
| T-10 Frailty Scale and hints | Pass | See below. |
| T-13 The patient never sees an assessment | Pass | Tess's "Thank you" page and her home (both HQs listed, 388 of 388 px) say nothing about MUST, Morse, VTE, frailty, scores, BMI or readings. |

### T-19: details

- **Step 1 (a B item left):** the card opened "11 of 30 filled: answer the 19 items highlighted." I answered No on the 18 missing items other than B1.
  - "29 of 30 filled: answer the item highlighted."
  - The badge says **Incomplete**, with no thrombosis result.
  - "Confirm once every item has an answer." shows, and **Confirm is greyed out**.
  - B1 Active bleeding is highlighted orange, Missing.
- **Step 2:** No on B1 gives "30 of 30 filled." and "**2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0**" (T7 and T11 are Yes from the HQ). Confirm is enabled.
- **The same with a T item left:** after **Undo changes**, I answered every missing item but T3 Dehydration. The badge is Incomplete, Confirm is greyed out and T3 is highlighted. No on T3 gives the result again.
- **Confirm:** "Confirmed by Cara Clinician on 13/11/2026, 12:18" (`POST …/confirm` 200), kept after a reload.

### T-20: details

- **Step 1:** Tess answered 165 cm, 70 kg, weight loss **Yes, 5 kg**, a fall **Yes**, Crutches…, family history of a clot Yes, pregnant **Yes**, No to the other 7. "Thank you, Tess".
- **Step 2:** a second episode for the same record (found by surname, **Choose**) on Demo risk HQ. The new invite opens the set as **Please check**: "We've filled in 10 answers from your questionnaire on 13 Nov 2026."
  - **Page 1:** height 165 is filled ("From your answers on 13 Nov 2026"). "What do you weigh now?" is empty. "Have you lost weight without trying…?" has **neither Yes nor No**. After I clicked Yes, "About how much…(kg)?" was **empty**.
  - **Page 2:** "Have you had a fall in the last 3 months?" is **empty**. The walking aid is filled (Crutches…).
  - **Page 3:** "Are you pregnant…?" is **empty**. The other 8 are filled as sent (family history Yes, 7 No), each marked "From your answers…".
- 15 answers were sent and 10 copied. The 5 left out are today's weight (as before S49) and the four the case names.

### T-21: details

- **Step 1:** Alex Author's new HQ has the library's weight question and the ready-made MUST (a Group "Weight loss", 2 questions, both marked **Always ask fresh**). Published. Pia's answers: 68 kg, No to weight loss; sent.
- **Step 2:** Ada turned Trust A's "Observations take precedence in scores" **off** and saved.
- **Step 3:** with Height 170 and Weight 60 (hospital BMI 20.8), MUST opened "3 of 4 filled": **1 BMI 20.8, Hospital BMI**; **2 Weight now 60, Hospital weight**; 3 is 0 (HQ); 4 Missing.
- **The height entered in error** (reason "Made-up: wrong patient"): without a reload, "2 of 4 filled: answer the 2 items highlighted." **1 BMI is Missing** (orange, empty box) and **2 Weight now is 68, Patient reported weight**. The same after a reload.
- **Afterwards:** I set the trust back to "Use Preopsis's default" and saved ("Trust A's settings were saved."). After a reload the box is still ticked.

### Happy path: MUST

- On Tess's first episode with no readings, MUST opened "3 of 4 filled": 25.7 Patient reported BMI, 70 Patient reported weight, 5 HQ.
- With a hospital height only (170), the card kept the patient's pair. With the weight too (60), it changed without a reload to **20.8 Hospital BMI** and **60 Hospital weight**.
- No on item 4 gives "**1 (Medium risk), 7.7% of body weight lost**" (5 of 65 kg). Confirmed.

### T-10: details

1. The Frailty Scale opened "0 of 1 filled", Level Missing, with **no** blue box.
2. After DASI was confirmed: "For reference: DASI: 34 (Poor functional capacity), 6.9 METs (confirmed 13/11/2026, 12:30)".
3. After PRISMA-7 ("7 of 7 filled", "1 (Not at increased risk)") was confirmed, a second line appeared in the same box. There's 1 box on the page.
4. Levels: 5 gives "5 (Living with Mild Frailty)"; 8 gives "8 (Living with Very Severe Frailty)", shown whole in the closed menu (`scrollWidth` 210 = `clientWidth` 210); 4 gives "4 (Living with Very Mild Frailty)", source Clinician, confirmed.
5. DASI item 1 → No, then Save: DASI is "31.25 … 6.6 METs, Not confirmed". The DASI line is gone from the box and the PRISMA-7 line stays.

## Failures in detail

None.

## Found while exploring

1. **A confirmed VTE card notes the new hospital BMI.** Tess's VTE was confirmed with her own BMI (25.7). After the hospital readings were added, the card says "The hospital BMI is now 20.8 (Patient reported BMI 25.7 when confirmed)." and "The score stays as confirmed…". This reads well.
2. **An emptied number box makes a confirmed VTE card Incomplete.** I emptied T5's BMI box on the confirmed card: "Enter a BMI", T5 Missing, the badge Incomplete, "Changed: confirm it again", and Save and Confirm greyed out. A reload brought the confirmed card back.
3. **"0 pts" beside Yes and No in the editor (a question, not a failure).** In the editor, the inserted MUST question "Have you lost weight without trying…?" shows a "0 pts" chip beside both Yes and No, though the window says it has no score in an HQ. I can't say if this is new; the patient never sees it.
4. **Data I left:**
   - Tess's second episode: the HQ is part answered (weight 69, weight loss No, a fall No) and not sent.
   - Pia: a heart rate of 220 (saved anyway), a height marked entered in error, and MUST added, not saved.
   - Fern: Apfel was added and its Yes on item 4 wasn't saved; DASI is left **not confirmed** (T-10 step 5).
   - The published questionnaire MT S49 1222 Weight MUST.
   - Cara Clinician is left signed in on the tab.

## Environment

- The episode list had e2e data from 12:11 (a `make e2e` run just before this round). It didn't affect any case.
- Pages loaded in about 1–2 s throughout.

## Console and network errors

- **Console:** no errors or warnings in either tab.
- **Network:** the only 4xx I saw was R-01's deliberate 400 (`POST …/observations`).
  - I listed the `/api/` requests up to T-21's send (about 100) and every one was 2xx, including the publish.
  - After that I listed only the observation and Confirm calls: all 2xx but that 400.
  - Tracking started after T-19's Confirm, and I didn't list the two settings saves. For those three I have only what the page showed ("Confirmed by…", "Trust A's settings were saved.").

## GIFs

- `~/Downloads/S49-retest4-T19.gif`: T-19, from the new VTE card through B1 left empty, Undo, T3 left empty, to Confirm.
- `~/Downloads/S49-retest4-T21.gif`: T-21 step 3, the readings, MUST with the hospital's pair, then the height entered in error.
