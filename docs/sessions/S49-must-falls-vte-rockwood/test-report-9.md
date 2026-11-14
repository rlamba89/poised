# S49 test report 9 (retest after verification 6)

**17 passed, 0 failed, 0 blocked.** Report 8 had no failures. So this round ran:
- the new case, T-25 (from verification 6);
- the 5 regression cases;
- T-24 and T-23 step 2, re-run on the changed MUST logic;
- 9 happy paths and earlier cases.

T-25 passes on an unconfirmed card and on a confirmed one.

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (`git status` not clean).
- **When:** Sat 14 Nov 2026, 12:58–13:13 IST.
- **URL:** http://localhost:3000.
  - Rahul's `make dev` was restarted at 12:58, just before the round. API health was ok, and the evaluate service was `clinical@31046e6b03bc-dirty`.
- **Browser:** Claude in Chrome. Desktop cases ran at 1280×900.
  - The patient's pages ran at **606 px wide**, because Chrome wouldn't make the window narrower (asked for 390×844). There was no sideways scrolling: `scrollWidth` was 606.
- **How I clicked:** real clicks and typing for every step a case checks.
  - A script answered the 9 clot questions on the patient's page 3 (HRT and blood thinner Yes, the rest No).
  - A script also answered VTE's 18 missing items other than B1 with No. B1, the item T-19 checks, I clicked by hand.
  - I read some card text and box values back with a script.
- **Test data (all made up):**

  | Episode | Used for | HQ, patient, how answered |
  | --- | --- | --- |
  | **Zia MT S49 1305 Pair** (new) | T-01, T-02, R-01, T-25, T-24, T-23 step 2, Morse, VTE + T-19, T-12, T-13, T-14, T-15, exploring | Demo risk HQ, woman, 1 Jan 1980. **The patient answered through the invite link**: 165 cm, 70 kg, No then Yes to weight loss, 2 kg, a fall, crutches, HRT Yes, blood thinner Yes, the other 7 No. Readings 170 cm, 60 kg, heart rate 220. |
  | Ravi MT S49 1430 Snore (report 5's) | R-02 | Demo STOP-Bang HQ |
  | Fay MT S49 1433 Fit (report 5's) | R-03, R-04, the Frailty Scale's hints | Demo fitness HQ |

  Also report 5's draft questionnaire **MT S49 1437 Ready** (Alex Author), for R-05. I found it with the Questionnaires search.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-25 On MUST, putting the other box back makes the earlier box untouched again** | **Pass** | Every step as expected, on an unconfirmed card and again after Confirm. See below. |
| T-24 (re-run) | Pass | Steps 2–4 on the confirmed card. 36 → 20.8 back → 50: item 1 Missing with an empty box. 61 → 60 back → 17.3: item 2 Missing with an empty box. 50, then 1 typed and deleted: "Between 5 and 150" for the 1; once deleted, no "Enter a BMI" and Save on. Step 5 not re-run: it passed in report 8, and verification 6 didn't change it. |
| T-23 step 2 (re-run) | Pass | 30 in item 1, then the hospital's 60 typed into the emptied weight: both Entered by clinician, "0 (Low risk), 3.2%…", Confirm on. Undo brought back the confirmed card. |
| T-01 The patient answers the Demo risk HQ (at 606 px, see above) | Pass | No first: no "how much" question. Yes: it appears. Next with nothing answered on page 2: "Response required." on both. A man's three questions show for this woman (9 on page 3). The send says "Thank you, Zia". The home page shows only the HQ (Sent, View), Information and contact details. The only "risk" there is the HQ's own name. |
| T-02 MUST opens 3 of 4 | Pass | "3 of 4 filled: answer the item highlighted.", 20.8 Hospital BMI, 60 Hospital weight, 2 (HQ), item 4 Missing (orange) with the full wording; Confirm and Save off. No: "0 (Low risk), 3.2% of body weight lost". |
| R-01 Observations | Pass | Weight -3 (Enter): "Weight must be from 1 to 400 kg.", nothing saved. Heart rate 220 (Enter): "Heart rate 220 beats/min is outside the usual range (50 to 120). Save it anyway?"; **Save anyway** saved it. Hospital BMI 20.8 after height and weight. |
| R-02 STOP-Bang | Pass | Ravi's card: "8 of 8 filled.", S, T, O, P from the HQ, B 35.9 Hospital BMI, A 66 Profile, "5 (High risk)", confirmed 13/11/2026, 14:31. The POA Summary shows "STOP-Bang 5 (High risk)". |
| R-03 DASI | Pass | Fay's card: "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs", confirmed. |
| R-04 Apfel | Pass | Fay's card: "4 of 4 filled.", Female (Profile), smoking No (HQ), PONV Yes (HQ), opioids Yes (Clinician), "4 (PONV risk about 79%)", confirmed. |
| Frailty Scale hints (T-10) | Pass | Fay's Frailty Scale card: "5 (Living with Mild Frailty)", and the For reference box with DASI's and PRISMA-7's confirmed lines. |
| R-05 Ready-made window | Pass | The Score menu lists STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale and VTE risk; there's no Clinical Frailty Scale. STOP-Bang shows the orange "It uses the patient's BMI…" box, and Insert is off. MUST has the "weight-loss questions only, with no score" note and BAPEN's lines, and Insert is on. I cancelled without inserting. |
| Morse (T-07) | Pass | "2 of 6 filled: answer the 4 items highlighted.": 1 Yes (HQ), 3 Crutches… (HQ). Then 2 No, 4 No, Gait Weak, Overestimates or forgets their limits: "65 (High risk)", confirmed. |
| VTE (T-08, with T-19) | Pass | "11 of 30 filled"; T2 46 (Profile); T5 Hospital BMI. With B1 left: "29 of 30 filled: answer the item highlighted.", Incomplete, Confirm off, "Confirm once every item has an answer.", B1 highlighted. No on B1: "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1" (HRT; the blood thinner). Confirmed. After Confirm the note reads "The age is the patient's on the day it was confirmed (46)." |
| T-12 POA Summary | Pass | Under Hospital BMI (20.8): MUST "0 (Low risk), 3.2% of body weight lost", Morse "65 (High risk)", VTE "1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1", Clinical Frailty Scale "Not confirmed". The VTE value wraps within its own column. Both tabs (Validated summary, Patient answers) show the same lines. |
| T-13 The patient never sees an assessment | Pass | After everything was confirmed, the patient's home and their sent HQ (re-opened at 606 px): no MUST, Morse, VTE, Frailty, score, band, BMI or reading text. |
| T-14 Another hospital's or trust's staff | Pass | Val (Hospital A2) and Bea (Trust B), opening the episode's URL: "You don't have access to this hospital". I also called the episode's assessments list from the signed-in page (`fetch`) through every org and hospital id in each user's `/api/me`: all **404**. |
| T-15 Reload, deep link, back | Pass | MUST item 4 changed to Yes ("2 (High risk)…", "Changed: confirm it again"), then reload: the card is as confirmed (0 Low risk, No). POA Summary, then browser Back: the episode page, with the three confirmed cards. |

### T-25 in detail

On Zia's MUST card, with 170 cm and 60 kg recorded, item 4 No.

- **Step 2:**
  - 61 in item 2: item 1 Missing, its box empty (grey "BMI").
  - 20.8 typed into that empty box: both **Entered by clinician**, "0 (Low risk), 3.2%…", Confirm on.
  - 60 typed back in item 2: **Hospital BMI** and **Hospital weight** again.
  - 50 in item 2: **item 1 Missing**, box value read back as `""` after clicking away. Item 2 Entered by clinician, badge **Incomplete**, Confirm off.
- **Step 3:** Undo, No again.
  - 36 in item 1: item 2 Missing, its box empty.
  - 60 typed into item 2: both Entered by clinician.
  - 20.8 back in item 1: both hospital sources.
  - 25 in item 1: **item 2 Missing**, box value `""` ("kg" hint), Incomplete, Confirm off.
- **Step 4:** Undo. 61 in item 2, then 20.8 in item 1: both **Entered by clinician** (20.8 and 61), "0 (Low risk), 3.2% of body weight lost", Confirm on.
- **On a confirmed card:** Undo, No, **Confirm** ("Confirmed by Cara Clinician on 14/11/2026, 13:04"). Then step 2 again:
  - after 60 back, the card read "Confirmed" again with Confirm and Save off;
  - after 50, item 1 was Missing with an empty box, "Changed: confirm it again", Incomplete.
  - **Undo changes** brought back the confirmed card.

## Failures in detail

None.

## Found while exploring

1. **Keyboard:** Tab and Shift+Tab move between MUST's boxes, and Cmd+A selects a box's text.
   - I used the keyboard for a longer chain: 61 in item 2, Shift+Tab, 20.8, Tab, 60 back, Shift+Tab, 36, then 20.8 back.
   - Each step showed the expected sources: after 36, item 2 Missing and empty.
   - At the end the card read "Confirmed" again, with Confirm and Save off.
2. **Emptying a pre-filled box** (BMI 20.8, Hospital BMI) shows "Enter a BMI" in red, the item goes Missing and Save turns off. This is the existing "a box that had a value asks for one" rule (report 8, item 2). Undo put it back.
3. **Two tabs:** the same episode in two tabs, on the unconfirmed Clinical Frailty Scale card.
   - Tab B saved level 5. Then tab A (opened earlier) picked level 4 and clicked Save.
   - Tab A got **409** with "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved." and a **Reload** button. Reload showed level 5.
   - Good behaviour.
4. **Double-clicking Confirm** (Frailty Scale) sent one `POST …/confirm` (200), with no error.
5. **Odd inputs in MUST's BMI box:**
   - " 21" (a leading space) counts as 21, Entered by clinician.
   - "2e1" and an emoji: "A number, one decimal place", Save off.
   - **"20,8" (a comma) counts as 20.8**: the item kept "Hospital BMI" and the card stayed "Confirmed" with nothing to save, while the box still showed "20,8".
   - Reasonable for a UK user. Noted only because the box shows what was typed, not "20.8".
6. **The Gait menu's open list still starts right of its box** (cosmetic; already in pending.md).
7. **The episode was auto-triaged to Ready for admission** by the rule "Low risk" as soon as the patient sent. This is as in reports 7 and 8. It isn't a problem; I didn't need the HQ review this round.
8. **Data I left:**
   - **Zia MT S49 1305 Pair:** readings 170 cm, 60 kg, heart rate 220 (saved anyway).
     - MUST, Morse, VTE and the Clinical Frailty Scale confirmed (the Frailty Scale at level 5, from the two-tab check).
     - Auto-triaged to Ready for admission, not marked Checked.
   - Ravi, Fay and the draft MT S49 1437 Ready: unchanged (Ready's window opened and cancelled).
   - Cara Clinician was the last user signed in. I closed my tabs.

## Console and network errors

- **Console:** no errors or warnings on either tab. Tracking started at about 13:08, for the exploring.
- **Network:** I listed the 75 `/api/` requests from 13:07 to 13:09 (the T-14 checks, the sign-ins and outs, R-05's editor, the episode page). Every one was 2xx except:
  - the **404s** from my own T-14 `fetch` calls as Val and Bea;
  - the **409** from the two-tab check (expected).
- After that, the only assessment requests were that 409 and one Confirm (200).
- To find report 5's episodes among about 4,150, I read Hospital A1's episode list once through the API from the signed-in page (Cara). I didn't use the database.

## GIFs

- `~/Downloads/S49-retest9-T25.gif`: from adding MUST on Zia's episode, through T-25 steps 2–4, Confirm, step 2 on the confirmed card, and T-24 steps 2–4 (50 frames, the recorder's limit).
