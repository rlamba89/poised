# S49 test report 2 (retest)

**15 passed, 0 failed, 0 blocked** (10 test cases + 5 regression cases)

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (36 entries in `git status`: the session's build and report 1, not committed yet).
- **When:** Sun 8 Nov 2026, 16:38–17:00 IST.
- **URL:** http://localhost:3000 (`make dev` run by Rahul). API health was ok.
- **Browser:** Claude in Chrome. As in report 1, `resize_window` had no effect (viewport stayed 1512 px wide). For the phone-size pages I used a **390×780 iframe** on a localhost page and checked `scrollWidth` (388 of 388) on every patient page.
- **What I re-ran:** T-12 (report 1's only failure), the cases the plan changed after report 1 (T-01, T-03, T-10), the main happy path through MUST, Morse and VTE (T-02, T-04, T-07, T-08), T-13, and every regression case (R-01 to R-05).
- **Test data (all made up):**

  | Episode | Used for | HQ, patient |
  | --- | --- | --- |
  | Rosa MT S49 1640 Retest | T-01/02/03/04/07/08/12/13, R-01 | Demo risk HQ, woman, 1 Jan 1965; answered by the patient through the invite |
  | Iris MT S49 1650 Fit | T-10, R-03, R-04 | Demo fitness HQ, woman, 1 Jan 1980; Fill in for the patient |
  | Ray MT S49 1655 Snore | R-02 | Demo STOP-Bang HQ, man, 1 Jan 1960; Fill in for the patient |

  Also a draft questionnaire **MT S49 1656 Ready parts retest** (Alex Author), for R-05. Nothing was inserted into it.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-12 POA Summary shows each result** (report 1, F-1) | **Pass** | **Fixed.** MUST "2 (High risk), 9.8% of body weight lost", Morse "65 (High risk)", VTE "3 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 2", Frailty "Not confirmed". On **both tabs** every label is 190 px wide and every value starts at x = 961. The VTE value wraps onto three lines in its own column. |
| T-01 Patient answers Demo risk HQ on a phone | Pass | 390 px frame, no sideways scroll. "How much" shows only after Yes. An empty amount blocks Next ("Response required."). Confirm window, then "Thank you, Rosa". The home page has none of MUST, Morse, VTE, Frailty, score, %, High/Medium risk or assessment (the HQ's name isn't counted now). |
| T-02 MUST opens 3 of 4 | Pass | "3 of 4 filled: answer the item highlighted."; 20.8 Hospital BMI, 60 Hospital weight, 6 HQ; item 4 orange with the full wording, Missing; Incomplete, Confirm greyed. No → "1 (Medium risk), 9.1% of body weight lost", Clinician. Not confirmed after Save. |
| T-03 MUST cut-offs (spot check) | Pass | 18.4 / 89.9 / 10.1 / No → "4 (High risk), 10.1%…", each typed box "Entered by clinician". **Typing 20.8 back into box 1 restored "Hospital BMI"**, as the updated plan says. 101 in box 3 → "Between 0 and 100", Missing, Save and Confirm greyed. Undo changes brought back T-02's values. I didn't re-run all 12 rows: they passed in report 1 and the scoring hasn't changed. |
| T-04 Corrected weight before Confirm | Pass | After correcting 60 → 55 ("Made-up: outdoor clothes"), without a reload: 19 Hospital BMI, 55 Hospital weight, item 4 No (Clinician), "2 (High risk), 9.8% of body weight lost". Confirmed by Cara Clinician. I didn't re-run the second correction to 58 (passed in report 1). |
| T-07 Morse | Pass | "2 of 6 filled: answer the 4 items highlighted."; 1 Yes (HQ), 3 "Crutches, a stick or a walking frame" (HQ). Gait is a menu of 3; mental status has 2 buttons. 2 No, 4 No, Weak, Overestimates → "65 (High risk)". Confirmed. |
| T-08 VTE pre-filled | Pass | "11 of 30 filled: answer the 19 items highlighted."; T2 61 Profile; T5 box 19 (Hospital BMI after the 55 kg correction). No to every missing item → "2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1". T5 30 → 2, 30.1 → 3. B1 Yes → "Bleeding risk factors: 2"; confirmed. |
| T-10 Frailty Scale and hints | Pass | 0 of 1, Missing, no blue box. After DASI is confirmed: "DASI: 34 (Poor functional capacity), 6.9 METs (confirmed …)". After PRISMA-7: a second line, "PRISMA-7: 1 (Not at increased risk) (confirmed …)". Badges 5 / 9 / 4, Clinician, confirmed. **The closed menu shows each level whole**: "8: Living with Very Severe Frailty" and "4: Living with Very Mild Frailty" fit (box `scrollWidth` 210 = `clientWidth` 210; report 1's exploring #2 is fixed). DASI item 1 No + Save: the DASI line goes, the PRISMA-7 line stays, and there's only one "For reference" box on the page. |
| T-13 Patient never sees an assessment | Pass | After MUST, Morse and VTE were confirmed, I signed in again through the invite at 390 px. The sent page and home show nothing about assessments. `GET /api/p/episodes/{id}` (read through the API) has only answers and question keys (`morse_falls`, `vte_hrt`…): no scores. |
| R-01 Observations | Pass | Weight -3: Go's 400, "Weight must be from 1 to 400 kg." Heart rate 220: "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saves it. |
| R-02 STOP-Bang | Pass | 8 of 8: S T O P N from the HQ, B Hospital BMI 32.7, A 66 Profile, G Profile; "4 (High risk)"; confirmed; "STOP-Bang 4 (High risk)" on the POA Summary. |
| R-03 DASI | Pass | "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs". |
| R-04 Apfel | Pass | "3 of 4 filled: answer the item highlighted.": Female (Profile), smoking No and PONV Yes (HQ), only "Opioids after surgery" missing. Yes → "4 (PONV risk about 79%)". |
| R-05 Ready-made window BMI note | Pass | The menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk, with no Frailty Scale. STOP-Bang's window shows the orange "It uses the patient's BMI…" box. MUST's has none, and shows BAPEN's permission line and "In an HQ it's the patient's weight-loss questions only, with no score…". |

## Failures in detail

None.

## Found while exploring

1. **The POA Summary at narrower widths.** I loaded it in a 390 px and a 768 px frame: no sideways scroll at either. Every value starts in the same column (x = 237 in the frame), and the label column stays 190 px. At 390 px the VTE value wraps onto five lines; Hospital BMI and MUST wrap onto three. It's readable, but narrow. Staff pages aren't phone-first, so this is only a note.
2. **Enter saves an observation.** Typing a height and pressing Enter saved the reading. That's good for keyboard use.
3. **Not checked:** the POA Summary's **print** layout for the long VTE value. The fix is a flex rule in `Detail`, and print may lay it out differently. Worth one look in Print preview.
4. **Data I changed:** on Rosa's episode I opened Correct on the **height** by mistake and closed it with Back. Nothing was saved. On Iris's episode, DASI is left **not confirmed** (T-10 step 6). Apfel was added there and left unsaved, with Yes not saved. Alex Author is left signed in on the tab.

## Console and network errors

- **Console:** no errors or warnings in either tab while tracking was on.
- **Network:** 132 `/api/` requests. The only 4xx was R-01's deliberate 400 (`POST …/observations`).

## GIFs and screenshots

- `~/Downloads/S49-retest-MUST-happy.gif`: the main happy path (readings, MUST 3 of 4, No, Save, T-03 typing, Undo, the weight correction, Confirm).
- `~/Downloads/S49-retest-T12.gif`: the POA Summary, both tabs, with the values lined up.
- /var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-4N0RuJ/screenshot-1794136604498-1.png: the POA Summary's values, zoomed in.
