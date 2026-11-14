# S49 test report 3 (retest after verification 1)

**12 passed, 0 failed, 0 blocked.** The 12 are T-17, T-18, 5 regression cases and 5 happy paths.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (38 entries in `git status`, including verification 1's fixes).
- **When:**
  - Sun 8 Nov 2026, 18:39–19:05 IST.
  - Then Fri 13 Nov 2026, 11:17–11:22 IST. Between the two, the round stopped because the Mac ran out of memory (see "Environment").
- **URL:** http://localhost:3000.
  - Rahul ran `make dev`, restarted at 18:42 after the fixes and again on 13 Nov.
  - API and evaluate health were ok both times.
- **Browser:** Claude in Chrome.
  - As in reports 1 and 2, `resize_window` had no effect.
  - The patient pages were tested in a **390 px iframe** on a localhost page, with `scrollWidth` checked.
- **What I ran:**
  - verification 1's new cases, T-17 and T-18;
  - every regression case, R-01 to R-05;
  - happy paths through MUST (with the new "There's no BMI now" note), Morse, VTE, the Frailty Scale's hints (T-10) and the POA Summary (T-12, T-15 Back).
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | Wendy MT S49 1845 NoHeight | T-17, MUST happy path, T-12, T-15 | Demo risk HQ, woman, 1 Jan 1980; Fill in for the patient (165 cm, 70 kg, no weight loss) |
  | Mark MT S49 1900 Man | T-18 (man), T-09-style VTE, R-01, Morse | Demo risk HQ, man, 1 Jan 1966; the patient through the invite at 390 px |
  | Olly MT S49 1905 Other | T-18 (Other) | Demo risk HQ, sex Other, 1 Jan 1966; through the invite at 390 px (HRT Yes) |
  | Rex MT S49 1902 Snore | R-02 | Demo STOP-Bang HQ, man, 1 Jan 1960; Fill in for the patient |
  | Fay MT S49 1905 Fit | R-03, R-04, T-10 | Demo fitness HQ, woman, 1 Jan 1980; Fill in for the patient |

  Also a draft questionnaire, **MT S49 1120 Ready parts retest 3** (Alex Author), for R-05. Nothing was inserted into it.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-17 MUST: a hospital weight without a height** | **Pass** | See below. |
| **T-18 A man isn't asked about pregnancy, contraception or HRT** | **Pass** | See below. |
| R-01 Observations | Pass | Weight -3: Go's 400, "Weight must be from 1 to 400 kg." Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. |
| R-02 STOP-Bang | Pass | 8 of 8. S, T, O, P and N from the HQ; B 34.9 Hospital BMI; A 66 Profile; G Male Profile. "4 (High risk)", confirmed. The POA Summary shows "STOP-Bang 4 (High risk)" under Hospital BMI. |
| R-03 DASI | Pass | "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs". |
| R-04 Apfel | Pass | "3 of 4 filled": Female (Profile), smoking No and PONV Yes (HQ), only "Opioids after surgery" Missing. Yes gives "4 (PONV risk about 79%)". |
| R-05 Ready-made window | Pass | See below. |
| Happy path: MUST confirmed, then the height entered in error | Pass | See below. |
| Happy path: Morse | Pass | "2 of 6 filled: answer the 4 items highlighted." 1 No (HQ), 3 Nothing (HQ). 2 Yes, 4 No, Gait Weak, Knows their own ability gives "25 (Moderate risk)", confirmed. |
| Happy path: VTE, 60 and nothing ticked (T-09's check) | Pass | Mark (60): T2 "60" Profile. No to every missing item gives "0 (No thrombosis risk factors ticked), Bleeding risk factors: 0". |
| T-10 Frailty Scale and hints | Pass | See below. |
| T-12 / T-15 POA Summary, Back | Pass | Wendy's POA Summary shows "MUST 0 (Low risk), 0% of body weight lost" under Hospital BMI. Back returns to the episode with MUST as confirmed. |

### T-17: details

- **Step 2 (weight 60 only):**
  - "2 of 4 filled: answer the 2 items highlighted."
  - **1 BMI** is Missing: orange, with an empty box.
  - **2 Weight now** is 60, Hospital weight.
  - 3 is 0 from the HQ (the "No" to weight loss).
  - The patient's 25.7 isn't used.
- **Step 3:**
  - Adding Height 170 changed item 1 to **20.8, Hospital BMI** without a reload.
  - Marking that height **Entered in error** (reason "Made-up: wrong patient") made item 1 Missing again, with item 2 still 60 Hospital weight.
  - This stayed the same after a reload.
- **Step 4:** Ada set Trust A's "Observations take precedence in scores" to off and saved. After a reload as Cara, items 1 and 2 showed **25.7 Patient reported BMI** and **70 Patient reported weight**.
- **Afterwards:** I set the trust back to "Use Preopsis's default" and saved ("Trust A's settings were saved.").

### T-18: details

- **Step 1 (a man, 390 px):** page 3 has **6** questions. None of them asks about pregnancy, HRT or an oestrogen contraceptive. The frame width is 388 px and `scrollWidth` is 388, so there's no sideways scroll. The send showed "Thank you, Mark".
- **Step 2:**
  - VTE says "8 of 30 filled".
  - **T8 HRT, T9 Oestrogen contraceptive and T11 Pregnancy are Missing**, not No.
  - T1, T4, T7, T10, B3 and B7 come from the HQ; T2 is 60 (Profile).
  - With No to every missing item, then **Yes on T8**, the result is "**1** (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0". It confirmed, and stayed after a reload.
- **Step 3:**
  - **Other** (through the invite at 390 px) sees all **9** questions (388 of 388 px). Their HRT Yes pre-fills T8 **Yes (HQ)**; T9 and T11 are No (HQ); 11 of 30 are filled.
  - The **woman** (Wendy, by Fill in for the patient) sees all 9.

### R-05: details

- The score menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk. The Clinical Frailty Scale isn't in it.
- STOP-Bang's window shows the orange "It uses the patient's BMI…" box.
- MUST's window has no BMI box. It shows "In an HQ it's the patient's weight-loss questions only, with no score…" and BAPEN's "reproduced here with the kind permission of BAPEN…" line.

### Happy path: MUST confirmed, then the height entered in error

- On Wendy's episode, with a new height of 170, MUST says "4 of 4 filled".
- No on item 4 gives "0 (Low risk), 0% of body weight lost", and it confirmed.
- Then I marked that height **Entered in error**. The card keeps the confirmed score and says:
  - **"There's no BMI now (20.8 when confirmed)."**
  - "The score stays as confirmed. To use a new value, change the item and confirm again."
- This is the new `sinceConfirmed` wording.

### T-10: details

1. After the Frailty Scale was added: "0 of 1 filled", Level Missing, and **no** blue box.
2. After DASI was confirmed: "For reference: DASI: 34 (Poor functional capacity), 6.9 METs (confirmed 13/11/2026, 11:18)".
3. After PRISMA-7 ("7 of 7 filled", "1 (Not at increased risk)") was confirmed, a second line appears in the **same** box. There's 1 box on the page.
4. Picking levels:
   - Level 5 gives "5 (Living with Mild Frailty)".
   - Level 8 gives "8 (Living with Very Severe Frailty)". The closed menu shows it whole: `scrollWidth` 210 = `clientWidth` 210.
   - Level 4 gives "4 (Living with Very Mild Frailty)", source Clinician, confirmed.
5. DASI item 1 → No, then Save: DASI is "31.25 … 6.6 METs, Not confirmed". The DASI line is gone from the box and the PRISMA-7 line stays.

## Failures in detail

None.

## Found while exploring

1. **A slow page load once.** On 8 Nov one episode page took about 8 s to show. Soon after, the tabs froze; that was the out-of-memory Mac (see "Environment"), not the app. On 13 Nov every page loaded in about 1–2 s.
2. **Enter saves an observation** (height, weight), as report 2 said.
3. **Data I left:**
   - Mark has a heart rate of 220 (saved anyway).
   - Wendy has two heights marked entered in error.
   - Olly's VTE card was added and left unsaved.
   - Fay's Apfel was added, with Yes not saved, and isn't confirmed. Her DASI is left **not confirmed** (T-10 step 6).
   - Alex Author is left signed in on the tab.

## Environment

- **The servers were stale at first.** On 8 Nov, `make dev` had been running since 15:25, before verification 1's fixes (about 17:58). Rahul restarted it at 18:42, before any case ran.
- **The Mac ran out of memory during R-03.**
  - Swap was 23.1 of 24 GB used and the load average was about 80–140.
  - Chrome's localhost tabs froze: a screenshot or script timed out after 30–45 s, on pages that had worked before.
  - From the terminal the servers answered in about 20 ms, and no Chrome process was busy, so I don't count this as an app fault.
  - I stopped there. On 13 Nov (load about 4), with `make dev` running again, I finished R-03, R-04, R-05 and T-10. Fay's answers had been saved but not sent, so I sent them first.

## Console and network errors

- **Console:** no errors or warnings in any tab while tracking was on.
- **Network:** the only 4xx was R-01's deliberate 400 (`POST …/observations`). Up to that point every other `/api/` request was 2xx: 94 on 8 Nov. Network tracking wasn't on in the tab used on 13 Nov.

## GIFs

- `~/Downloads/S49-retest3-T17.gif`: T-17, from the weight-only MUST card through the height and the height entered in error, to the precedence switch and the patient's own pair.
