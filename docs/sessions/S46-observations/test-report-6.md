# S46 test report 6: observations (retest after test report 5)

**Summary: 18 passed, 0 failed, 0 blocked** of the 18 cases run. The four new cases (T-22 to T-25) pass, and so do the listed regressions, all five R cases and four happy paths. Report 5's Major problem, a stale tab undoing another row of the ranges table, is fixed: tried in five variants, plus five saves at the same moment. Exploring found no new failures, only two small points.

- **Environment:** `e3a7d86` with the S46 work uncommitted (63 changed or new files). Tested 31 Oct 2026, 17:03–17:16, at http://localhost:3000. No code changed during the round.
- **Servers:** at the first check `make dev` was still the one from 15:15, before the fixes. Rahul restarted it at 17:03:37, and the new process was confirmed before any testing. No migration was needed (the database is at 00037).
- **Browser:** Chrome via Claude in Chrome, 1280 px wide, for every case. The tab group disappeared once, at about 17:10, between steps; I opened a new tab, and nothing was lost.
- **Scripted inside the real page:** to save round trips, I focused boxes and clicked Save and tick boxes with page JavaScript. All values were typed as real keystrokes, and the long notes as real pastes (Cmd+V). I logged PATCH bodies by wrapping the page's `fetch`.
- **Headless Chromium** (Playwright from `e2e/.venv`) for 390×844, the page left open (it needs two users at once) and the print layout. The window size was tried in Chrome in report 5 and didn't shrink.
- **API and database, to confirm what I saw:** `GET …/settings`; read-only SQL on `observations` and `audit_log`; `GET` of the patient route (404). **Two uses of the API that aren't through the page's own buttons,** both on purpose: aborted `fetch` calls for T-24, and five `PATCH …/settings` at once while exploring.
- **Test data:** Rae MT S46 1704 Retest6 (Hospital A1), episode `01f28ab2-…`. The patient saved 170 cm and 82 kg, and didn't send.
- **Settings put back as found (checked with `GET …/settings`):**
  - Trust A: `trust: {}`.
  - Hospital A1: stored values are only its phone and email, as before. Observations is on Follow trust, and on.
  - Hospital A2: `{}`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-22 Two admins editing the ranges** | **Pass** | Tab 2 set Heart rate Usual to 105; stale tab 1 changed only Weight: refused with the exact text, nothing stored, and it keeps what was typed. Reload: 105 and 200; making the change again saves. Same row (108 then 110): the second refused, 108 stored. Ticking the default in a stale tab: refused. A stale tab changing only another setting: saves. |
| **T-23 A note too long** | **Pass** | Real paste of 2,200 characters into an ECG: "The note is too long (2000 characters at most).", no request, all 2,200 still in the box. Exactly 2,000 saves whole (database: 2000, ending `END2000`). Correct has the same limit: 2,200 refused, 2,000 saved. Also refused at 390×844 with the note kept. |
| **T-24 Leaving a page mid-request** | **Pass** | 10 aborted `fetch` calls to the episode list, 8 stopped in time: 8 rows with status 499, shown as a grey **Cancelled**, none as 500. Filtered to Ada, her older row of 16:14 still says Refused (500), in red. |
| **T-25 Observations off, no refused rows** | **Pass** | Off for Hospital A1. As Cara: no card, no Hospital BMI row on either POA tab. The network log has no `…/observations` API request on the episode or POA Summary (only Vite's source files of that name). No `observation.list` 403 rows since the restart. Turned back on. |
| T-09 Observations off | Pass | As T-25, with other cards there and Patient reported BMI staying. Follow trust: card and BMI back. The episode's answer carries `observationsOn`. |
| T-10 The trust's own ranges | Pass | 50 to 100 and 30 to 150 saved. Cara: 110 warns "…usual range (50 to 100)", 160 refused "Heart rate must be from 30 to 150 beats/min." Ada's three refusal messages exact, nothing stored from them. Hospital A1 shows "Set for the whole trust". Back on the default, 110 saves with no warning. |
| T-16 A long unbroken note | Pass | 400 "x": wraps inside the card, all of it readable, at 1280 px and 390 px, with no sideways scroll. At 390 px it's still 650 px tall, in a 107 px column (pending.md). |
| T-18 One changed range | Pass | `observations.ranges` held only `heartRate`; a row set back to its default dropped out. The other rows showed Preopsis's numbers. Put back. |
| T-19 A Settings page left open | Pass | Save with no edits: "Nothing has changed.", no second PATCH. Stale tab changed only "Mark not responding on day" to 9: saved, with Heart rate staying at tab 2's 109. Put back. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added, marked entered in error with a reason, in the General notes; "No grade yet" afterwards. |
| R-02 Tasks | Pass | Added, completed, re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm and 82 kg, both tabs. |
| R-04 Another setting | Pass | Seen in T-19 and T-22 step 6. |
| R-05 Audit | Pass | For this episode, as I did them: Recorded 7 + 1 refused (400), Corrected 2, Marked entered in error 1, Opened 20. Each row names the patient; no values. |
| T-01 Record each kind (happy path) | Pass, part | Height, weight, heart rate and ECG this round. Enter submits. BMI 26.1 for 175 cm and 80 kg. |
| T-04 Correct a reading (happy path) | Pass | 80 to 78, reason given: BMI 25.5, old value struck through with "Corrected to 78.0 kg". Empty reason not re-tried. |
| T-05 Entered in error (happy path) | Pass | The only height marked: struck through, who, when and why; the BMI says "Not measured yet…". A new height counts again. |
| T-06 POA Summary (happy path) | Pass | Patient reported BMI 28.4, Hospital BMI "25.5 (measured 31/10/2026, 17:12)", on both tabs and under print styles. |

**Not re-run this round** (passed in earlier reports): T-02, T-03, T-08, T-11, T-12, T-13, T-14, T-15, T-17, T-20, T-21's forced failure, and Val and Bea in T-07. The patient's side of T-07 was seen again: nothing about observations, and `/api/p/episodes/<id>/observations` gives 404.

## Failures in detail

None.

## Found while exploring

- **Report 5's Major, tried harder (all fine):**
  - Tab 2 ticks the default, then a stale tab edits a row: refused.
  - Both tabs start on the default, tab 2 sets a range, the other sets its own: refused.
  - A stale tab changes a range **and** another setting together: refused whole, so neither is stored (day stayed 9, not the typed 8). The typed values stay on the page, and the message says to reload.
  - Five `PATCH …/settings` at the same moment with the same starting value: one 200, four 409, and the one that won is what's stored.
- **A corrected long note shows in full twice over (Cosmetic):** the old row keeps its note and adds "Corrected to" followed by the whole new note, above the new row with that same note. For a 2,000-character ECG, the new text appears twice, about 2,000 characters each. Short values read fine ("Corrected to 78.0 kg").
- **A long note fills the Add a reading form (Cosmetic):** the note box grows with its text, so after a 2,000-character paste Save reading and Measured at sit at the bottom of a tall form. Everything can be reached by scrolling.
- **Good:**
  - The limit counts characters: 2,000 Japanese characters (6,000 bytes) save whole, and so do 1,200 emoji. Report 4's point about a silent cut is gone: `maxLength` is off and a real paste keeps everything.
  - **A page left open when Observations is turned off:** Save reading shows "Observations aren't turned on at this hospital." in the card, nothing is stored, and the page doesn't break. Tried in headless, with Sam as a second user. It's one refused `observation.add` row in the audit log, which is correct: someone tried to write.
  - Changing the kind of reading clears a red message and the typed value.
- **Already known:** the red message stays while you edit the same value (report 5), and a reading can be dated the day before the date of birth (report 4).

## Console and network errors not tied to a case

- **Audit log since the restart at 17:03:37:** no 5xx. Statuses: 200, 201, 204, one 400 (the refused 160), one 403 (the page left open, on purpose) and eight 499s (T-24).
- **Headless Chromium:** no page errors. The only console error is the browser's failed-resource line for that 403.
- **Chrome console:** thin this round. The first read, early on, was empty; the tab group then disappeared, and the last tab's tracking began only at the final read, which was also empty. So nothing was seen, but most of the round wasn't watched in Chrome. Headless and the audit log are the better evidence.
- **Before the restart, not this code:** 40 `episode.list` 500s at 16:41 and four 499s from 16:52 to 17:00. No test of mine caused them, and no pytest was running during this round. The last audit row before I began T-24 was from my own testing.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**, as in report 5. Headless Chromium at that size passed T-16 and T-23 this round.

## GIFs and screenshots

- `~/Downloads/S46-retest6-T22.gif`: the main happy path (the two tabs, the refusal, the reload and the save).
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/01ba4f3d-8e60-4980-86ab-566e7dafbb3f/scratchpad/s46r6/`:
  - `t16-390.png`: the long note at 390×844;
  - `left-open.png`: the page left open, at 390×844;
  - `print.png`: the POA Summary under print styles;
  - `poa.pdf`: the same as a PDF, rendered but not opened, as there's no PDF reader on this machine.
