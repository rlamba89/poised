# S46 test report 3: observations (retest)

**Summary: 13 passed, 0 failed, 0 blocked** (8 cases + 5 regressions re-run). F-4 from report 2 is fixed, and the two small changes after it work. One new minor point came up while exploring (a long unbroken note is cut off); it isn't in the plan, so it's not counted as a failure. T-12 was run in headless Chromium again (see the note).

- **Environment:** `e3a7d86` with the S46 work uncommitted (50 changed or new files). Tested 30 Oct 2026, 18:58–19:15, at http://localhost:3000.
- **Servers:** `make dev` in Rahul's terminal, started at 18:58, after the builder's last fixes.
- **Browser:** Chrome via Claude in Chrome, at 1280×757.
- **Phone size:** Chrome did resize this time, but it stops at 500 px wide, so 390×844 still can't be reached. T-12, the odd inputs, Val and Bea's checks and the print PDF were done in headless Chromium (Playwright from `e2e/.venv`), signed in through the dev login. Patient steps were done in Chrome at desktop size.
- **Test data:** Ola MT S46 1900 Retest (Hospital A1), episode `6d05c72e-…`.
- **Settings put back as found** (checked with `GET /api/o/<trust>/settings`: the trust's own settings are empty):
  - Trust A's Observation ranges: Preopsis's default.
  - "Mark not responding on day": the default.
- **Not re-run:** T-08, T-09 and T-11, which passed before and weren't touched by the fixes. T-01 wasn't repeated in full (weight, height, heart rate and blood pressure only; all 11 kinds passed in report 2).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 An earlier time | Pass | Yesterday 08:00 is listed last with its own time. The hint goes when the time is changed and comes back after the save. |
| T-03 Soft warnings and hard limits | Pass | Steps 1 and 4 only. 220 warns with the exact text and Save anyway saves it. 80/120 gives "The diastolic must be lower than the systolic." at once, with no yellow box. |
| T-04 Correct a reading | Pass | **F-4 fixed.** Empty reason: "Say why the reading is corrected." After a reason and Save, the message is gone. The struck-through 80.4 kg, the Corrected badge, "Corrected to 78.0 kg" and the kept time are right. Back after a refused save clears it too. |
| T-05 Entered in error | Pass | **F-4 fixed.** Empty reason: "Say why the reading was entered in error." After a reason and Mark, the message is gone and no `alert` is left on the page. With the only height marked: "Not measured yet (needs a height and a weight)". A new height gave BMI 25.2. |
| T-06 POA Summary | Pass | Both tabs: "Patient reported BMI 28.4" (170 cm / 82 kg, saved by the patient, not sent), then "Hospital BMI 25.2 (measured 30/10/2026, 19:02)". The print PDF keeps both. |
| T-07 Who can see and record | Pass | Step 4: the patient's home shows nothing about observations, the episode JSON has no reading fields, and `/api/p/episodes/<id>/observations` is a 404. Steps 2 and 3, by API only: `GET …/observations` as Val and as Bea gives 404. |
| T-10 The trust's own ranges | Pass | "-1": "Heart rate: a limit can't be below 0." with nothing stored. HR 50/100/30/150 saves as typed. Cara: 110 warns "(50 to 100)"; 160: "Heart rate must be from 30 to 150 beats/min." (Cara's two checks in headless Chromium.) Back on the default, the effective range is 50–120 / 20–300; I didn't re-enter 110 after that. |
| T-12 Phone size (390×844) | Pass (headless) | BP 136/88 added and corrected to 130/82. The boxes stack, the new hint fits, the buttons are reachable, and there are no page errors. scrollWidth was 390 at the end of the run. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added, then marked entered in error with a reason; both are in the General notes. |
| R-02 Tasks | Pass | A task is added, completed and re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm / 82 kg. |
| R-04 Another setting | Pass | "Mark not responding on day" 9 saved; the ranges stayed unset. Put back to the default. |
| R-05 Audit | Pass | For this episode: Recorded 5, Corrected 1, Corrected REFUSED (409) 1, Marked entered in error 2, Opened an episode's observations 19. The counts match what I did at that point. Each row names the patient, and no values appear. |

### The builder's other changes, checked

- **The ranges' message names the row:** "abc" in Weight's Usual from: "Weight: give all four limits as numbers." An empty Peak flow Usual from: "Peak flow: give all four limits as numbers."
- **The Measured at hint:** "The time you save it, unless you change it." shows until the time is changed, and fits at 390 px.
- **A stale tab's message still stays, as intended:** tab B marked heart rate 72 entered in error. In tab A, Correct gave "This reading has already been corrected or marked entered in error." and the list reloaded. That message stays under the reading through later saves, until the page is reloaded.

## Failures in detail

None.

## Found while exploring

- **A long unbroken note is cut off (Minor, new):** an ECG note of 400 "x" characters with no spaces saves, but it doesn't wrap. It runs off the card on one line, and the card hides the rest, at 1280 px and at 390 px (the text is about 3,087 px wide in a 958 px or 356 px card). The page itself doesn't scroll sideways, and the Correct and Entered in error buttons stay on screen. Scrolling to the reading also moved the card's contents 16 px to the left. So a clinician can't read the rest of the note. It would only matter for a long pasted link or code; I didn't try a note of long text with spaces, though long reasons wrap. Screenshots: `x-long-note-1280.png`, `x-long-note-390.png`.
- **Narrow reading text at phone size (cosmetic):** beside the two buttons, a reading's text gets about 107 px, so "Measured 30/10/2026, 19:06 · corrected by Cara Clinician: …" takes five or six short lines (`t12-4-corrected.png`).
- **Emoji, `<b>` tags and leading spaces** in a correction reason: shown as plain text, and the spaces are trimmed. Good.
- **Reloading with a Correct form open:** the form closes and nothing is saved. Good.
- **Print (cosmetic, from report 1, in pending.md):** the Hospital BMI value still wraps and sits a little left of its column.
- **Phone size (cosmetic, from S21):** the Tasks card's **Complete** button is still cut off ("Comple") at 390 px (`t12-1-bp-form.png`).

## Console and network errors not tied to a case

- **Chrome console:** nothing from when I started reading it (partway through T-05) to the end. The earlier steps weren't captured.
- **Headless Chromium:** no page or console errors in the runs that finished. The first run stopped on a mistake in my own script's selector, after the BP had been saved and corrected; it printed no errors before stopping, but its error list was never reached.
- **Network:** I didn't capture requests in Chrome this round. The audit log shows only one refusal for this episode's observations, the expected 409 from the stale tab.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**: the window stops at 500 px wide. Headless Chromium at that size passed.

## GIFs and screenshots

- `~/Downloads/S46-retest3-F4-fixed.gif`: the happy path and F-4's fix (Correct, Back, Entered in error).
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/5924e668-881d-4703-ae3d-61a2740bd075/scratchpad/s46/out3/`:
  - `t12-1-bp-form.png`, `t12-3-correct-form.png`, `t12-4-corrected.png`: T-12 at 390×844;
  - `x-long-note.png`, `x-long-note-1280.png`, `x-long-note-390.png`: the long note;
  - `poa.pdf` and `poa.pdf.png`: the POA Summary's print layout.
