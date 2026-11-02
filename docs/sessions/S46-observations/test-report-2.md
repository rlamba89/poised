# S46 test report 2: observations (retest)

**Summary: 13 passed, 2 failed, 0 blocked** (10 cases + 5 regressions re-run). Both failures are the same new minor bug (F-4). F-1, F-2 and F-3 from report 1 are fixed. T-12 was run in headless Chromium again (see the note).

- **Environment:** `e3a7d86` with the S46 work uncommitted (50 changed or new files). Tested 29 Oct 2026, 21:28–21:46, at http://localhost:3000.
- **Servers:** `make dev` in Rahul's terminal, restarted at 21:27 at my request. The one running before was from 17:09, older than the Go fix for F-1 (17:41).
- **Browser:** Chrome via Claude in Chrome, at desktop size (1512×805).
- **Phone size:** the Chrome window again wouldn't resize to 390×844 (the page stayed 1512×805). So T-12 and the print PDF were done in headless Chromium (Playwright from `e2e/.venv`), signed in through the dev login. The patient steps were done in Chrome at desktop size.
- **Test data:** Nia MT S46 2128 Retest (Hospital A1), episode `e5852f9e-…`.
- **Settings put back as found** (checked with `GET /api/o/<trust>/settings`):
  - Trust A's Observation ranges: Preopsis's default.
  - "Mark not responding on day": the default.
- **Not re-run:** T-08 and T-09, which passed in report 1 and weren't touched by the fixes.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Record each kind | Pass | **F-1 fixed.** All 11 kinds save, ECG and Urinalysis included, with no console errors. The labels show units, the newest is first, "· entered by Cara Clinician", the kind stays chosen and the boxes empty. Hospital BMI 26.3. Enter submits a number box. |
| T-02 An earlier time | Pass | Tomorrow 08:00: "The time it was measured can't be in the future." Yesterday 08:00: listed last, with 28/10/2026 08:00; BMI unchanged. After the save, Measured at went back to now. |
| T-03 Soft warnings and hard limits | Pass | **F-2 fixed:** 80/120 gives "The diastolic must be lower than the systolic." at once, with no yellow box. 100/100 gives the same. 220 warns with the exact text, and Change it closes it. Empty ECG note: "Enter the ECG note." "1e2": "Heart rate: enter a number." (new, as the builder noted). |
| T-04 Correct a reading | **Fail** | Everything in the plan works: the empty reason is refused, 80.4 → 78.0 → 78.2, the struck-through history, "corrected by Cara Clinician: …", the same time, BMI 25.5. **But the red "Say why the reading is corrected." stays on screen under the old reading after the correction is saved** (F-4). |
| T-05 Entered in error | **Fail** | The empty reason is refused, and the BP is struck through with a badge, "Entered in error: marked by Cara Clinician on 29/10/2026, 21:32: Made-up: another patient's" and no buttons. **The red "Say why the reading was entered in error." stays after the mark succeeds** (F-4). |
| T-06 POA Summary | Pass | Both tabs: "Patient reported BMI 28.4" (170 cm / 82 kg, saved by the patient, not sent), then "Hospital BMI 25.5 (measured 29/10/2026, 21:30)". The print PDF keeps both. |
| T-07 step 4: the patient | Pass | The patient's home shows nothing about observations. `/api/p/episodes/<id>` has no observation, BMI or reading fields, and `/api/p/episodes/<id>/observations` is a 404. Checked from the patient's own session. |
| T-10 The trust's own ranges | Pass | **F-3 fixed:** "-1" stays in the box, and Save is refused: "Heart rate: a limit can't be below 0." Nothing was stored. "Heart rate: the usual range must be inside its hard limits, lowest first." and "SpO₂: a limit can't be above 100%." are exact. 50/100/30/150 saves as typed. Cara: 110 warns "(50 to 100)"; 160: "Heart rate must be from 30 to 150 beats/min." A1 shows "Set for the whole trust" / "THE TRUST'S OWN". Back on the default, 110 saves with no warning. |
| T-11 Reload, deep link, Back | Pass | After a reload: the same list and the same BMI. POA Summary → Back: the episode page with the card. |
| T-12 Phone size (390×844) | Pass (headless) | BP 136/88 added and corrected to 130/82 in headless Chromium at 390×844. The boxes stack, the buttons are reachable, scrollWidth is 390 throughout, and there are no page errors. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added, then marked entered in error with a reason; both are in the General notes. |
| R-02 Tasks | Pass | A task is added, completed and re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm / 82 kg. |
| R-04 Another setting | Pass | "Mark not responding on day" 9 saved, with the Observation ranges still on the default (`null`). Put back to the default. |
| R-05 Audit | Pass | "Recorded an observation", "Corrected an observation", "Marked an observation entered in error" and "Opened an episode's observations", each naming the patient, with no values. Refusals show as REFUSED (400 for the future time, 409 for the stale correction). |

### The builder's other changes, checked

- **Plain text boxes for the ranges:** yes. "-1", "abc" and an empty box are kept as typed and refused on Save.
- **"Measured at" is the time of Save:** yes. The form showed 21:33; I saved at 21:35 without changing it, and the reading says 21:35. See the note below.
- **"1e2" refused:** yes, "Heart rate: enter a number."
- **A stale tab's Correct shows the message and reloads the list:** yes. Tab B marked SpO₂ 97 entered in error. In tab A, Correct → Save correction gave "This reading has already been corrected or marked entered in error." The list then showed the reading as Entered in error, with no buttons.

## Failures in detail

### F-4 (Minor): the "Say why…" error stays after a correction or mark succeeds

**Where:** Cara Clinician, `/o/…fa/h/…0a/episodes/e5852f9e-ea8e-4b5e-a135-d68c0676de70`.

**Steps:**
1. On any active reading, click **Correct**, type a new value, leave the reason empty, and click **Save correction**. "Say why the reading is corrected." shows (correct).
2. Type a reason and click **Save correction**.

**Expected:** the correction saves and the red message goes away.

**Actual:**
- The correction saves: the new value is at the top, and the old one is struck through with **Corrected**.
- But the red "Say why the reading is corrected." stays under the old, struck-through reading (an `alert` in the page).
- The same happens with **Entered in error**: "Say why the reading was entered in error." stays under the reading after it's marked.
- The messages pile up, one for each reading done this way (three on screen at once in my run). They go only on a page reload.

A clinician reading the card sees an error next to a change that worked, and may think it didn't save. Seen for weight, heart rate and blood pressure. There were no console errors.

**GIF:** `~/Downloads/S46-retest-F4.gif` (heart rate 72: Correct, empty reason, then a reason, Save; the error stays).

I didn't see this in report 1. It may have come with the stale-tab change, which keeps a per-reading message after the form closes.

## Found while exploring

- **Measured at shows an old time:** the box isn't updated while the form is open, though Save uses the time of Save (as the builder intended). My form showed 21:33, and the reading was saved as 21:35. A clinician who reads the box before saving sees a different time from the one stored. A small hint ("now, unless you change it") or a refresh at Save would avoid it. Minor.
- **The ranges' error doesn't say which row:** "abc" or an empty box in one range gives "Observation ranges: give each measurement's four limits as numbers." The other range errors name the measurement ("Heart rate: …"). Minor.
- **Double-click on Save reading** saved one reading. Good.
- **Enter** submits a number box. In the ECG and Urinalysis note boxes it adds a new line, which seems right for a note.
- **The respiratory rate label wraps** ("Respiratory rate / (breaths/min)") at desktop width and pushes its row down a little. Cosmetic.
- **The kind dropdown can't be typed into** to jump to an item; with 11 kinds you scroll it. Cosmetic.
- **Print (cosmetic, from report 1, not fixed):** in the POA Summary's print layout, the Hospital BMI value still wraps ("25.5 (measured 29/10/2026, / 21:30)") and sits a little left of the other values in its column.
- **Phone size (cosmetic, from S21, not S46):** the Tasks card's **Complete** button is still cut off ("Comple") at 390 px wide (`t12-1-bp-form.png`).

## Console and network errors not tied to a case

- There were no console errors in Chrome all session, and no page errors in headless Chromium.
- The only 4xx responses were expected (the audit log shows them): 400 for the future time, and 409 for the stale correction.
- The audit log also shows a `Listed episodes … REFUSED (500)` for Cara at 17:53. That was before this retest, during an earlier e2e run, not from S46's routes. Worth a look by whoever owns the list.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**: the window here wouldn't resize. Headless Chromium at that size passed.

## GIFs and screenshots

- `~/Downloads/S46-retest-T01.gif`: the main happy path, all 11 kinds (F-1's ECG and urinalysis first), F-2's BP check, 1e2, the 220 warning.
- `~/Downloads/S46-retest-F4.gif`: F-4.
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/5924e668-881d-4703-ae3d-61a2740bd075/scratchpad/s46/out/`:
  - `t12-1-bp-form.png`, `t12-2-saved.png`, `t12-3-correct-form.png`, `t12-4-corrected.png`: T-12 at 390×844;
  - `poa.pdf` and `poa.pdf.png`: the POA Summary's print layout.
