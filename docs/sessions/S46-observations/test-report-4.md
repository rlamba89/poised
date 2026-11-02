# S46 test report 4: observations (retest after verification 1)

**Summary: 23 passed, 0 failed, 0 blocked** (all 18 cases + 5 regressions). The six cases added after verification 1 (T-13 to T-18) pass, and so does verification's F-3 (corrections at the same moment), tried against the running app. Three small points came up while exploring; none is a failure against the plan.

- **Environment:** `e3a7d86` with the S46 work uncommitted (50 changed or new files). Tested 31 Oct 2026, 15:14–15:37, at http://localhost:3000.
- **Servers:** the first check found `make dev` still running from 30 Oct 18:58, before the fixes of 20:15. Rahul restarted it at 15:15 and the new process was confirmed before any testing.
- **Browser:** Chrome via Claude in Chrome, for T-13 to T-18, R-01, R-02, R-04, R-05 and T-08, and the Sam steps of T-09.
- **Headless Chromium** (Playwright from `e2e/.venv`, signed in through the dev login) for everything else, including all of 390×844, because Chrome's window still stops at 500 px wide. That covers T-07 (Sue, Val, Bea, the patient), T-06's print PDF, Cara's side of T-09, T-10, and the T-03, T-04 and T-05 messages.
- **Database and API, read-only checks:** audit rows, stored readings and stored settings, each to confirm something seen on screen. `GET …/settings` for T-18. Direct POSTs only for the same-moment corrections and the tenancy checks.
- **Test data:** Rae MT S46 1516 Retest (Hospital A1), episode `559f6595-…`: 29 readings, now cancelled (T-08). The patient saved 170 cm and 82 kg, and didn't send.
- **Settings put back as found:**
  - Trust A: `GET …/settings` gives `trust: {}`, so Observation ranges and "Mark not responding on day" are on the default.
  - Hospital A1: Observations follows the trust; its stored settings are the same as before I started.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Record each kind | Pass | All 11 kinds entered this round, across Chrome and headless (SpO₂ by Sue). Text and unit labels are right, newest first, and the box empties after Save. Used weight 80 (BMI 26.1), not the plan's 80.4. |
| T-02 An earlier time | Pass | Yesterday 08:00 is listed below today's readings; the BMI doesn't change; the hint comes back. Tomorrow-style future date (1 Jan 2027): "The time it was measured can't be in the future." |
| T-03 Soft warnings and hard limits | Pass | Exact texts for 400, 72.5, empty, "eighty", -3, 80/120 (no yellow box), an empty ECG. "82,5" saved as 82.5. **Not re-entered:** heart rate 220 itself. The new-reading warning was seen with weight 380 and heart rate 110, and Save anyway on the Correct form (T-13). |
| T-04 Correct a reading | Pass | Empty reason refused; after a reason and Save, no message stays. Struck through, badge, "Corrected to …", kept time. Corrections of corrections work (81 → 82 → 83). |
| T-05 Entered in error | Pass | Empty reason refused; Back clears it. Struck through, badge, who, when and why; no buttons left. **Step 4 (the height) not repeated:** "Not measured yet" after marking was seen in T-14 instead. |
| T-06 POA Summary | Pass | Both tabs: Patient reported BMI 28.4, Hospital BMI "27.1 (measured 31/10/2026, 15:18)". The print PDF keeps both. The empty-readings wording was seen on the episode page, not re-checked on the POA Summary. |
| T-07 Who can see and record | Pass | Sue added SpO₂ 97. Val and Bea: "You don't have access to this hospital", no card, no patient name; `…/observations` GET and POST 404. Patient at 390×844: nothing on their pages, no reading fields in their JSON, `/api/p/episodes/<id>/observations` 404. |
| T-08 Cancelled episodes | Pass | Cancelled: all 29 readings and the BMI stay; no Add a reading, no buttons in the card. |
| T-09 Observations off | Pass | Off for Hospital A1: no card; no Hospital BMI line on either POA tab, Patient reported BMI stays; the other cards are unchanged; API 403; no page errors. Follow trust: all back. Re-run because F-4 moved this hook. |
| T-10 The trust's own ranges | Pass | See T-18. All three refusal messages exact, and "abc" names its row; nothing was stored from a refused save. Hospital A1 shows "Set for the whole trust". Back on the default, 110 saves with no warning. |
| T-11 Reload, deep link, Back | Pass | Reload keeps the list, the struck-through rows and the BMI. POA Summary then Back shows the card. |
| T-12 Phone size (390×844) | Pass (headless) | BP 136/88 added, corrected to 130/82. The boxes wrap, nothing is cut off, scrollWidth is 390. |
| **T-13 A correction outside the range** | **Pass** | 380: the exact yellow box, nothing saved, BMI still 26.1, no request sent. Change it keeps what's typed. 500: red "Weight must be from 1 to 400 kg." with no box. 210 → Save anyway: saved, BMI 68.6. 78: saved with no box, BMI 25.5. The box fits at 390 px. |
| **T-14 A correction marked entered in error** | **Pass** | The form says "The value it corrected won't count again. To put a value back, use Correct."; a plain height's form doesn't. Marked: all struck through, BMI "Not measured yet…". A new reading counts again, and correcting a correction counts (83 kg, BMI 27.1). |
| **T-15 Before the date of birth** | **Pass** | 1926, 0026 and 10/03/1980 refused with the exact text, nothing saved. 12/03/1980 (the date of birth) saves, listed last, and the BMI stays on today's weight. See exploring for 11/03/1980. |
| **T-16 A long unbroken note** | **Pass** | 395 "x" (a few short of 400): wraps inside the card, all readable, buttons on screen, at 1512 px (Chrome) and 390 px (headless). No sideways scroll. See exploring for the height at 390 px. |
| **T-17 POA tabs and the audit log** | **Pass** | 5 tab switches: 1,767 animation frames, none without the Hospital BMI; no requests; no audit rows after the page's first load. |
| **T-18 One changed range** | **Pass** | `observations.ranges` holds only `heartRate`; the other rows still show Preopsis's numbers. It also works end to end: Cara gets 50 to 100 and 30 to 150 on heart rate, and Preopsis's 35 to 200 and 1 to 400 on weight. Put back. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added, marked entered in error with a reason; both in the General notes. |
| R-02 Tasks | Pass | Added, completed, re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm and 82 kg. |
| R-04 Another setting | Pass | "Mark not responding on day" 9 saved while a heart rate range was set; the range stayed exactly as it was. |
| R-05 Audit | Pass | For this episode at that point: Recorded 6 + 3 refused (400), Corrected 4 + 1 refused (400), Marked entered in error 2. These match what I did; the 380 and 210 warnings correctly sent nothing. Each row names the patient, and no values appear. |

### Verification's other findings, checked

- **F-3, corrections at the same moment:** 6 at once on one reading gave `201, 409, 409, 409, 409, 409`, with one row written (verification 1 measured mostly 500s). Correct and Entered in error at once gave `409` and `204`.
- **F-5, times compared as text:** not tried by hand; it needs mixed offsets. Unit tests only.

## Failures in detail

None.

## Found while exploring

- **A reading can be dated the day before the date of birth (Minor, by design):** 11/03/1980 15:19 saves for a patient born 12/03/1980, though the message says it "can't be before the patient's date of birth". 10/03/1980 is refused. This matches the build notes (the limit is 14 hours before that midnight in UTC, for any time zone). I've left it for Rahul to weigh; it isn't counted as a failure.
- **A pasted note is cut at 2,000 characters with no message (Minor):** a note of 2,200 and one of 6,000 characters were both stored at exactly 2,000 (checked in the database). The box's `maxLength` does this. The clinician isn't told the end was dropped, which could matter for a long pasted ECG report. Go's "The note is too long" can't be reached from the page.
- **A long note is very tall at phone size (Cosmetic):** at 390 px the text column is 107 px wide beside the two buttons, so the 395-character note is about 650 px tall, with empty space under the buttons (`t16-390.png`). All of it is readable. This is report 3's narrow-text point; it shows far more with a long note.
- **Save reading moves when the time is changed (Cosmetic):** the "The time you save it…" hint goes away, so the form gets about 34 px shorter and the button moves up. My first click, aimed from the earlier screen, landed below it and did nothing.
- **Good:**
  - Double-clicking Save reading or Save correction saves once.
  - Editing the value closes the yellow box, so Save anyway can't save a number it didn't ask about.
  - `<b>` and `<script>` in a note show as plain text; no dialog ran.
  - "1e2" and Arabic digits are refused as not a number; spaces around a number are trimmed; 80.123456 gets "use at most 1 decimal place".
  - An emptied Measured at gets "When was it measured? Enter the date and time."
  - Reloading with a Correct form open closes it and saves nothing.
  - A 300-character unbroken reason wraps at 390 px.
- **Already known (print, in pending.md):** the Hospital BMI value still wraps and sits a little left of its column.

## Console and network errors not tied to a case

- **Chrome console:** nothing, from when I started reading it (late in T-16). Earlier steps weren't captured.
- **Headless Chromium:** no page errors. The only console errors were failed-resource lines for the 400s and 409s the tests caused on purpose.
- **5xx:** none in the audit log for the whole round (384 rows), and none seen in the headless runs.
- **False alarm, tooling only:** the extension's network log shows **503** for every `entered-in-error` POST. It's wrong: the page's own `fetch` saw 204 for the same request, and the audit row says 204. It's likely the extension mishandling an empty 204 answer. Future testers: don't trust that tool's status on these requests.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**: the window stops at 500 px wide. Headless Chromium at that size passed.

## GIFs and screenshots

- `~/Downloads/S46-retest4-T13.gif`: the main happy path (a correction: warning, Change it, the hard limit, Save anyway, a value inside the range).
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/15aa3266-3850-42c9-83f4-9625b7ae961e/scratchpad/s46/out/`:
  - `t12-correct-form.png`, `t13-390.png`, `t16-390.png`: at 390×844;
  - `poa.pdf` and `poa-1.png`: the POA Summary's print layout;
  - `patient-1-hq.png`, `patient-height-weight.png`: the patient at 390×844;
  - `t09-off-episode.png`, `t09-on-episode.png`: T-09.
