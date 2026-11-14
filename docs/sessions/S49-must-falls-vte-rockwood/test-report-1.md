# S49 test report 1

**20 passed, 1 failed (cosmetic), 0 blocked** (16 test cases + 5 regression cases)

- **Commit:** `31046e6` (master) plus S49's uncommitted changes (34 changed or new files in `git status`: the session's build, not committed yet).
- **When:** Sun 8 Nov 2026, 15:25–16:10 IST. The plan says "today is 7 Nov". The ages don't change (every birthday is 1 Jan).
- **URL:** http://localhost:3000 (`make dev` run by Rahul). API health was ok.
- **Browser:** Claude in Chrome. The window was full screen and **couldn't be resized** (`resize_window` had no effect; the viewport stayed 1512×861). For the phone-size cases (T-01, T-13), I loaded the patient's pages in a **390×844 iframe** on a localhost page. Media queries follow the frame's width, so the layout is the phone layout. I checked `scrollWidth` = 390 on every patient page, so there's no sideways scroll. A real phone or a resized window is still worth one look.
- **Test data (all made up):**

  | Episode | Used for | HQ, patient |
  | --- | --- | --- |
  | Mia MT S49 1530 Risk | T-01/02/03/04/07/08/12/13/14/15 | Demo risk HQ, woman, 1 Jan 1965; answered by the patient through the invite |
  | Nora MT S49 1550 NoLoss | T-05/06 | Demo risk HQ, woman, 1970; Fill in for the patient |
  | Leo MT S49 1555 Sixty | T-09/16 | Demo risk HQ, man, 1 Jan 1966; Fill in |
  | Ivy MT S49 1600 Frail | T-10, R-01, R-03, R-04 | Demo fitness HQ, woman, 1 Jan 1980; Fill in |
  | Rex MT S49 1605 Snore | R-02 | Demo STOP-Bang HQ, man, 1960; Fill in |

  Also a draft questionnaire **MT S49 1545 Ready parts** (Alex Author) for T-11.
- **Settings:** for T-06 I turned "Observations take precedence in scores" off and saved. I then put it back to **Use Preopsis's default** (on) and saved again.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Patient answers Demo risk HQ on a phone | Pass | 390 px frame, no sideways scroll on any page. "How much" shows only after Yes. Required questions block Next ("Response required."). In-app confirm window, then "Thank you, Mia". Home shows nothing about assessments (see exploring #4 on the HQ's name). |
| T-02 MUST opens 3 of 4 | Pass | 20.8 Hospital BMI, 60 Hospital weight, 6 HQ, item 4 orange Missing with the full wording, Incomplete, Confirm greyed. No → "1 (Medium risk), 9.1% of body weight lost". After Save: "Not confirmed". GIF recorded. |
| T-03 MUST cut-offs typed by hand | Pass | All 12 rows match. Messages: "Between 0 and 100", "A number, one decimal place" (-1, abc), "Enter a number" (empty), "Between 1 and 400" (0). Save and Confirm greyed while a box is wrong. Undo brings back T-02's values. |
| T-04 Corrected weight before / after Confirm | Pass | Without a reload: 19 / 55, sources kept, No kept, "2 (High risk), 9.8%…". Confirmed by Cara, kept after reload. After the 58 correction the badge stays, and the yellow box reads exactly as expected. |
| T-05 "No" = 0 kg; no readings | Pass | 25 Patient reported BMI, 64 Patient reported weight, 0 HQ, item 4 missing. |
| T-06 Trust precedence decides MUST's weight | Pass | With readings: 27.3 Hospital BMI / 70 Hospital weight. Setting off: 25 Patient reported BMI / 64 Patient reported weight. Setting restored. |
| T-07 Morse | Pass | "2 of 6 filled: answer the 4 items highlighted."; 1 Yes HQ, 3 menu HQ. Gait is a 3-choice menu, mental status 2 buttons. 65 / 0 / 20 / 25 / 45 / 50 with the right bands. Confirmed. |
| T-08 VTE pre-filled | Pass | "11 of 30 filled: answer the 19 items highlighted."; T2 61 Profile; T5 box 20.1 Hospital BMI; 9 from the HQ (T8 and B3 Yes). Form wording shown; "The age is the patient's today." Badges: 2 / 3 (31) / 2 (30) / 3 (30.1); B1 Yes → "Bleeding risk factors: 2"; confirmed. |
| T-09 VTE: 60 isn't over 60 | Pass | T2 60; "0 (No thrombosis risk factors ticked), Bleeding risk factors: 0". |
| T-10 Frailty Scale + hints | Pass | 0 of 1, menu of 9 levels, no blue box. DASI line, then the PRISMA-7 line, as expected. Badges 5 / 9 / 4, source Clinician, confirmed. After DASI is changed and saved, its line goes and PRISMA-7's stays. Only one "For reference" box on the page. |
| T-11 Ready-made parts in an HQ | Pass | Menu: STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale, VTE risk (no Frailty Scale). Each window has the source, licence and "patient's questions only, no score". Groups: Weight loss (2), Falls (2), Blood clots (9); no Calculation. Fields locked. Preview shows only the questions. |
| **T-12 POA Summary shows each result** | **Fail (Cosmetic)** | All four values are right, but **the VTE risk value doesn't line up** with the other values, on both tabs. See F-1. |
| T-13 Patient never sees an assessment | Pass | Home and sent HQ at 390 px show nothing. `GET /api/p/episodes/{id}` returns only the patient's answers and question keys (`vte_hrt`, `must_wtloss`…), no scores (checked through the API). |
| T-14 Other hospital / trust | Pass | Val and Bea: "You don't have access to this hospital". Through the API, every hospital ID of theirs gives 404 for the episode and its assessments. |
| T-15 Reload, deep link, back | Pass | Unsaved Yes ("4 (High risk)…", "Changed: confirm it again") is gone after a reload; the card is as confirmed. Back from the POA Summary returns to the episode with the assessments as saved. |
| T-16 Cancelled episode read-only | Pass | VTE card shows text only: 0 buttons, 0 inputs. No Save, Confirm or Add assessment. |
| R-01 Observations | Pass | Weight -3: Go's 400, "Weight must be from 1 to 400 kg." Heart rate 220 warns ("…outside the usual range (50 to 120). Save it anyway?") and saves with **Save anyway**. |
| R-02 STOP-Bang | Pass | S T O P N from the HQ, B Hospital BMI (31.6), A 66 Profile, G Male Profile; "4 (High risk)"; confirmed; on the POA Summary. |
| R-03 DASI | Pass | "12 of 12 filled.", "34 (Poor functional capacity), 6.9 METs". |
| R-04 Apfel | Pass | 3 of 4, waiting only for "Opioids after surgery"; Yes → "4 (PONV risk about 79%)". |
| R-05 Ready-made window BMI note | Pass | The BMI warning shows for STOP-Bang; MUST's window has none. |

## Failures in detail

### F-1 (T-12) The VTE risk value is out of line on the POA Summary (Cosmetic)

- **Signed in as:** Cara Clinician, Hospital A1.
- **URL:** `/o/…00fa/h/…000a/episodes/fcb9da7a-4d02-4f88-a8a5-d68892ce5782/poa` (both the Validated summary and Patient answers tabs).
- **Steps:** confirm a VTE risk assessment on an episode (any result: the text is always long), then open the POA Summary.
- **Expected:** every value starts in the same column ("Each value lines up with the others").
- **Actual:**
  - "3 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 2" wraps onto two lines.
  - It starts about 64 px left of the other values.
  - Its label "VTE risk" column shrinks from 190 px to 126 px.
  - The row is a flex row, and the label shrinks when the value is long. A `flex-shrink: 0` (or a fixed width) on the label would probably fix it. The MUST value is close to the limit too.
- **Console / network:** none.
- **Evidence:**
  - `~/Downloads/S49-T12.gif`
  - /private/tmp/claude-501/-Users-rahul-poised/9f194696-afc6-437f-8bac-d20439341a5b/scratchpad/S49-T12-vte-alignment.png

## Found while exploring

1. **Mouse wheel changes a patient's number answer (Minor, patient-facing, desktop only).**
   - **Where:** T-01, "About how much weight have you lost without trying (kg)?".
   - **What happened:** I typed 6, then scrolled the page with the mouse over the still-focused box. The answer silently became **5**.
   - **Why it matters:** that number feeds MUST's weight-loss %, so a patient on a desktop could send a wrong value without noticing. Phones have no wheel.
   - **Likely cause:** the browser's default behaviour on a focused `type=number` box. It probably affects every number question, not just S49's, so it may be older than this session.
2. **Frailty Level box cuts off the label (Cosmetic).** After choosing level 4, the closed menu reads "4: Living with Very Mild F". Levels 6–8 are longer still. The badge shows the full name.
3. **"5,5" is accepted as 5.5** in a MUST number box (9.1%). Leading spaces are accepted. "1e1", "5.55" and 😀 are refused with "A number, one decimal place". Probably fine; noting the comma in case it isn't meant.
4. **The seeded HQ's name contains "risk".** The patient's home shows "Demo risk HQ". It's seed data, not an assessment, but T-01 says no "risk" anywhere for the patient. A real trust would name its own HQs.
5. **Typing back the pre-filled value restores its source** (MUST item 3 back to 6 → HQ; Morse 1 Yes and 3 Crutches → HQ) rather than "Entered by clinician". That seems right; noting it because T-03 says "a typed item's source is Entered by clinician".
6. **Settings: the precedence switch looks off while "Use Preopsis's default" is ticked.** It's greyed with the knob on the left, but it's actually checked (on), which is confusing to read. This setting isn't new in S49.
7. **Things that worked:**
   - **Two tabs:** a stale Save on Morse got a 409 with "…changed somewhere else since you opened them. Reload to see the latest; your last change was not saved." plus a Reload button.
   - **Double-click on Confirm:** one POST.
   - **Another episode's assessment ID in the URL:** 404 "Assessment not found.".
   - **Adding an assessment to the cancelled episode** through the API: 409 "This episode is cancelled, so it can't be changed."
   - **Keyboard:** Tab from a box reaches "4: Yes" (with `aria-pressed`), and Enter sets it.
8. **Data I changed while exploring:** on Mia's episode, Morse was saved at 80, then put back to 65 and **re-confirmed** (16:0x). The other cards are as T-04/T-08 left them. On Ivy's episode, DASI is left **not confirmed** (T-10 step 6). Apfel and the heart-rate reading were added there.

## Console and network errors

- **Console:** no errors or warnings, in either tab, from the time tracking started (mid-session).
- **Network:** the only 4xx were the ones I triggered on purpose (R-01's 400, the 409 stale save, the 404 and 409 API probes, T-14's 404s).
- **Dev mode:** `GET /api/me` and the episode GETs run twice on each load (React dev mode), with no errors.

## GIFs and screenshots

- `~/Downloads/S49-T02-happy.gif`: the main happy path (MUST: item 4 No, then Save).
- `~/Downloads/S49-T12.gif`: F-1.
- /private/tmp/claude-501/-Users-rahul-poised/9f194696-afc6-437f-8bac-d20439341a5b/scratchpad/S49-T12-vte-alignment.png: F-1, zoomed in.
