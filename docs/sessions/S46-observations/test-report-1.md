# S46 test report 1: observations

**Summary: 14 passed, 3 failed, 0 blocked** (12 cases + 5 regressions). T-12 was run in a headless browser (see the note), not in Chrome.

- **Environment:** `e3a7d86` with the S46 work uncommitted (49 changed or new files). Tested 29 Oct 2026, 17:05–17:38, at http://localhost:3000 (`make dev`, Rahul's terminal).
- **Browser:** Chrome via Claude in Chrome, at desktop size (2560×1352).
- **Phone size:** the Chrome window wouldn't resize to 390×844 (it stayed 2560×1296, probably macOS full screen). So T-12 and the phone-size look at T-06/T-07.4 were done in headless Chromium (Playwright from `e2e/.venv`) at 390×844, signed in through the dev login. The patient steps in T-06/T-07.4 were done in Chrome at desktop size.
- **Test data:**
  - Mia MT S46 1105 Smith: the main episode, Hospital A1.
  - Leo MT S46 1125 Jones: cancelled in T-08.
- **Settings put back as found:**
  - Trust A's Observation ranges: Preopsis's default again.
  - "Mark not responding on day": default (7) again.
  - Hospital A1 Observations: "Follow trust (On)" again.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Record each kind | **Fail** | All 9 number kinds pass, with correct labels, units, the newest first and BMI 26.3. **ECG and Urinalysis can't be saved:** Save reading does nothing, and the console shows a TypeError (F-1). |
| T-02 An earlier time | Pass | Yesterday 08:00 is listed below today's readings; BMI keeps today's weight. Tomorrow: "The time it was measured can't be in the future." |
| T-03 Soft warnings and hard limits | **Fail** | 220 warns with the exact text; Change it and Save anyway both work. -3, 400, 72.5, empty, "eighty", "82,5" and the empty ECG note all pass. **BP 80/120 first shows "Systolic 80 mmHg is outside the usual range… Save it anyway?"**; only after Save anyway does it say "The diastolic must be lower than the systolic." (F-2) |
| T-04 Correct a reading | Pass | The empty reason is refused. The old value is struck through with a Corrected badge and "Corrected to 78.0 kg"; the new one has "corrected by Cara Clinician: …", keeps the time, BMI 25.5. A correction of the correction (78.2) works, and both earlier values stay. |
| T-05 Entered in error | Pass | The empty reason is refused. The reading is struck through with a badge, "Entered in error: marked by Cara Clinician on 29/10/2026, 17:18: Made-up: another patient's", and no buttons. Latest height in error: the BMI used the earlier 170 cm (27.1). Both heights in error: "Not measured yet (needs a height and a weight)". |
| T-06 POA Summary | Pass | Both tabs: "Patient reported BMI 28.4" (170 cm / 82 kg), then "Hospital BMI 25.5 (measured 29/10/2026, 17:19)". The print layout (PDF) keeps both. An episode with no readings shows "Not measured yet (needs a height and a weight)". |
| T-07 Who can see and record | Pass | Sue records (SpO₂ 98, "entered by Sue Super"). Val and Bea see "You don't have access to this hospital". The API gives them 404 for GET and POST, and "Episode not found." through Bea's own trust path. The patient's pages and `/api/p/…` JSON hold nothing about observations, and `/api/p/episodes/<id>/observations` is a 404. |
| T-08 Cancelled episode | Pass | The readings and BMI show, with no Add a reading, Correct or Entered in error. Checked by API too: POST and entered-in-error both give 409 "This episode is cancelled, so it can't be changed." |
| T-09 Observations switched off | Pass | Off for A1: no Observations card, no Hospital BMI line on the POA Summary, the other cards unchanged, the API 403. Back on: the card and every reading are back. |
| T-10 The trust's own ranges | **Fail** | Steps 1, 2, 3, 5 and 6 pass: 110 warns "(50 to 100)", 160 gives "Heart rate must be from 30 to 150 beats/min.", A1 shows "Set for the whole trust" / "THE TRUST'S OWN", and 110 no longer warns once back on the default. Step 4: "usual from 20" and "SpO₂ 101" give the exact messages. **"-1" can't be typed: the box drops the minus, and Save stored a hard limit of 1 with no error** (F-3). |
| T-11 Reload, deep link, Back | Pass | After a reload the same list (struck-through readings included) and the same BMI show. POA Summary → Back lands on the episode page with the card. |
| T-12 Phone size (390×844) | Pass (headless) | Blood pressure added and corrected at 390×844 in headless Chromium. The boxes stack, nothing is cut off, there's no sideways scroll (scrollWidth 390) and no page errors. Screenshots are listed below. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added on the POA Summary, then marked entered in error with a reason; the General notes have both. |
| R-02 Tasks | Pass | A task is added, completed and re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm / 82 kg, the same as before. |
| R-04 Another setting | Pass | "Mark not responding on day" 9 saved; the Observation ranges row (trust's own HR 50/100/30/150) was unchanged. Put back to the default. |
| R-05 Audit | Pass | "Recorded an observation", "Corrected an observation", "Marked an observation entered in error" and "Opened an episode's observations" each name the patient. No values appear anywhere in the rows, and refusals show as REFUSED (400/404/409). |

## Failures in detail

### F-1 (Major): ECG and Urinalysis readings can't be recorded

**Where:** Cara Clinician, `/o/…fa/h/…0a/episodes/382e88f2-…` (any episode).

**Steps:**
1. On the Observations card, choose **ECG** (or **Urinalysis**) in Add a reading.
2. Type a note, e.g. "Sinus rhythm (made-up)".
3. Click **Save reading**.

**Expected:** "ECG: Sinus rhythm (made-up)" at the top of the list.

**Actual:** nothing happens. No message is shown and no request is sent. The box keeps the text, and every click logs:

```
TypeError: Cannot read properties of null (reading 'some')
    at outsideUsual (packages/clinical/src/observations.ts:64)
    at warningText (apps/web/src/features/episodes/observations.ts:55)
    at save (apps/web/src/features/episodes/ObservationsPanel.tsx)
```

**What I saw when I checked:**
- `GET …/observations` sends `"measures": null` for the note kinds (`{"key":"ecg","label":"ECG","measures":null,"note":true}`, the same for urinalysis). `warningText` passes `k.measures` to `outsideUsual`, which calls `.some` on it.
- The API itself accepts an ECG: a POST through the browser's fetch gave 201. That reading then showed, and **Correct** on it worked from the UI. So only *adding* a note kind fails, in the browser.
- An empty ECG note still gets "Enter the ECG note.", because that check runs first.

**Severity:** Major. Two of the eleven kinds (OBS-01) can't be recorded at all. Clinicians only, nothing patient-facing.

**GIF:** `~/Downloads/S46-T01.gif` (the main happy path for nine kinds, then the ECG save failing).

### F-2 (Minor): a diastolic above the systolic asks "Save it anyway?" first

**Where:** Cara, the Mia episode.

**Steps:** Blood pressure, Systolic **80**, Diastolic **120**, then Save reading.

**Expected (T-03.4):** "The diastolic must be lower than the systolic."

**Actual:**
- The yellow box appears first: "Systolic 80 mmHg is outside the usual range (90 to 180). Save it anyway?"
- Only after **Save anyway** does the red error come (from the server, 400).

**Note:** 100/105 (both inside the usual ranges) gives the red error at once. The warning shouldn't be offered for a reading that will be refused. For weight -3 the plan says "no warning box", and the same should hold here.

### F-3 (Minor): a trust range of "-1" is quietly saved as 1

**Where:** Ada Admin, Settings → Whole trust → Observation ranges (default unticked).

**Steps:** Heart rate **Refused below**: select it, type `-1`, then **Save the trust's settings**.

**Expected (T-10.4):** refused: "Heart rate: a limit can't be below 0."

**Actual:**
- The box shows `1`: the minus sign is dropped as it's typed.
- Save says "Trust A's settings were saved."
- `GET /api/o/<trust>/settings` then held `"heartRate":{"hardMax":150,"hardMin":1,…}`. The admin typed one value and another was stored, without a word.

I put it back to 30, and later to the default.

## Found while exploring

- **Double-click on Save reading** saved one reading only. Good.
- **Leading and trailing spaces** ("  75 ") are trimmed and saved as 75. Good.
- **"1e2"** is accepted as a heart rate of **100** (scientific notation). Harmless, but odd for a clinical number box.
- **Two tabs:** in a stale tab, correcting a reading the other tab had already corrected gives "This reading has already been corrected or marked entered in error." But the stale list isn't refreshed and there's no Reload button, so the old reading still shows Correct. That's unlike the episode's other stale-write errors (SaveError with Reload). Minor.
- **Emoji, « » quotes, `&` and `<b>tags</b>`** in a correction reason show as plain text (no HTML). Good.
- **Empty "Measured at":** "When was it measured? Enter the date and time." Good.
- **"Measured at" can lag the save:** the default time is set when the form resets, not when Save is clicked. For example, the height saved at 17:20 shows "Measured 17:19". A nurse who leaves the form open for a while gets an earlier time unless they change it. Worth a thought; it may be as designed.
- **A reading with an earlier time** sorts below newer ones, often off screen, with no confirmation. Saving yesterday's weight gives no visible sign that anything happened unless you scroll. Minor UX.
- **The "Measured at" time is kept** after a refused future time and a change of kind. The next kind (heart rate) still carried tomorrow 08:00 and would have been refused again. Minor.
- **Print (cosmetic):** in the POA Summary's print layout, the Hospital BMI value wraps ("25.5 (measured 29/10/2026, / 17:19)") and sits slightly left of the other values in its column.
- **Phone size (cosmetic, from S21, not S46):** the Tasks card's **Complete** button is cut off ("Comple") at 390 px wide (`t12-1-bp-form.png`).
- **The plan's wording:** "Add a reading → Height" reads like a button, but Add a reading is the label of the "What was measured" dropdown. Fine once seen.
- **Audit volume:** each episode page load writes two "Opened an episode's observations" rows (the dev build loads it twice), and every save writes another. That made 76 rows in this session. It may be just dev React's double effects; worth checking on QA.

## Console and network errors not tied to a case

- None apart from F-1's TypeError.
- The 4xx responses seen were all expected:
  - 400 for the refused readings;
  - 403 with observations off;
  - 404 for Val and Bea;
  - 409 on the cancelled episode and the stale correction.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**: the window here wouldn't resize. Headless Chromium at that size passed.

## GIFs and screenshots

- `~/Downloads/S46-T01.gif`: T-01's happy path and the F-1 failure.
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/32da2806-20c3-4756-8c1c-fb65150b52a5/scratchpad/s46/`:
  - `poa.pdf`: the POA Summary's print layout;
  - `t12-1-bp-form.png`, `t12-2-saved.png`, `t12-3-correct-form.png`, `t12-4-corrected.png`: T-12 at 390×844.
