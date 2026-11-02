# S46 test report 5: observations (retest after verification 2)

**Summary: 16 passed, 0 failed, 0 blocked** of the 16 cases run. All three of verification 2's findings are fixed: T-19, T-20 and T-21 pass. **One Major problem came up while exploring and isn't covered by any case:** a stale Settings page still undoes a change to a *different row* of the ranges table. See "Found while exploring".

- **Environment:** `e3a7d86` with the S46 work uncommitted (55 changed or new files). Tested 31 Oct 2026, 16:20–16:37, at http://localhost:3000.
- **Servers:** `make dev` from 15:15. This round's changes are web code only (edited 16:07), so no restart was needed; Vite served them.
- **Browser:** Chrome via Claude in Chrome, 1512 and 1280 px wide, for every case. The tab group disappeared once, at about 16:22, just after the episode was made; I opened a new tab and carried on, and nothing was lost.
- **Headless Chromium** (Playwright from `e2e/.venv`) only for T-20 at 390×844 and the POA Summary's print layout, because Chrome's window didn't shrink: `resize_window` to 390×844 left it at 1280 px.
- **How T-21 was forced:** in my tab, I replaced the page's `fetch` so `GET …/observations` answered 500, then a rejected request. A reload took that away. Nothing in the app was changed.
- **Other API and database use, to confirm what I saw:** `GET …/settings` for the stored settings; a logged copy of each PATCH body in the T-19 tab; read-only SQL on `observations` and `audit_log`; `GET` of the observations routes for the 403 and 404.
- **Test data:** Rae MT S46 1623 Retest5 (Hospital A1), episode `53c42393-…`, 6 readings, left open. The patient saved 170 cm and 82 kg, and didn't send.
- **Settings put back as found (checked with `GET …/settings`):**
  - Trust A: `trust: {}`, so Observation ranges and "Mark not responding on day" are on the default.
  - Hospital A1: stored values are only its phone and email, as before. Observations is on Follow trust.
  - Hospital A2: `{}`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-19 A Settings page left open** | **Pass** | Save with no edits: "Nothing has changed.", no PATCH. Second tab set Usual to 105. The stale tab, still showing 100, changed only "Mark not responding on day" to 9: its PATCH held only `reminders.notRespondingDay`. After reload: 105 and 9. Both put back. See exploring for the case this doesn't cover. |
| **T-20 A five-digit year** | **Pass** | The box held `20266-10-31T16:22`. Red "When was it measured? Enter the date and time.", no reading, no console error, no request (the audit count below has no row for it). The same at 390×844 in headless, with no POST and no page error. |
| **T-21 POA Summary when readings don't load** | **Pass** | Forced 500 and a dropped connection: the row says "Couldn't be loaded", on both tabs and in the print layout. Reload: back to 25.5. Observations off: no row on either tab, no "Couldn't be loaded", Patient reported BMI stays. |
| T-10 The trust's own ranges | Pass | 50 to 100 and 30 to 150 saved. Cara: 110 warns "…usual range (50 to 100)", 160 refused "Heart rate must be from 30 to 150 beats/min." All three of Ada's refusal messages exact, nothing stored from them. Hospital A1 shows "Set for the whole trust". Back on the default, 110 saves with no warning. |
| T-13 A correction outside the range | Pass | 380: the exact yellow box, BMI still 26.1, nothing saved. Change it keeps what's typed. 500: red "Weight must be from 1 to 400 kg." with no box. 210 → Save anyway: BMI 68.6. 78: no box, BMI 25.5. |
| T-15 Before the date of birth | Pass | 1926 and 0026 refused with the exact text. 12/03/1980 (the date of birth) saves, listed last, and the BMI stays on today's weight. |
| T-18 One changed range | Pass | `observations.ranges` held only `heartRate`; the other rows showed Preopsis's numbers. A row set back to its default (Weight) dropped out of the stored value. Put back. |
| T-01 Record each kind (happy path) | Pass, part | Height, weight and heart rate only, this round. Text, units and "entered by" right; the kind stays chosen and the box empties; Enter submits. BMI 26.1 for 175 cm and 80 kg. |
| T-05 Entered in error (happy path) | Pass | Empty reason refused with the exact text. Struck through, badge, who, when and why; no buttons left; Enter submits. BMI unchanged, as it was an older weight. |
| T-06 POA Summary (happy path) | Pass | Patient reported BMI 28.4, Hospital BMI "25.5 (measured 31/10/2026, 16:22)", on both tabs and in the print layout. |
| T-09 Observations off | Pass | Run for T-21. Off for Hospital A1: no card, no Hospital BMI row, API 403 "Observations aren't turned on at this hospital.", other cards there. Follow trust: card and BMI back. |
| R-01 Anaesthetist's ASA grade | Pass | ASA II added, marked entered in error with a reason; both in the General notes. |
| R-02 Tasks | Pass | Added, completed, re-opened, above Observations. |
| R-03 Patient reported BMI | Pass | 28.4 for 170 cm and 82 kg. |
| R-04 Another setting | Pass | Seen in T-19: 9 saved, and the range stayed exactly as it was. This time with a competing edit, which report 4 didn't have. |
| R-05 Audit | Pass | For this episode, as I did them: Recorded 4 + 3 refused (400), Corrected 2 + 1 refused (400), Marked entered in error 1. Each row names the patient; no values. |

**Not re-run this round** (passed in report 4): T-02, T-03, T-04's own steps, T-08, T-11's Back, T-12, T-14, T-16, T-17, and Val and Bea in T-07. The patient's side of T-07 was seen again: nothing about observations, and `/api/p/episodes/<id>/observations` gives 404.

## Failures in detail

None against the plan.

## Found while exploring

- **A stale Settings page still undoes a change to another row of the ranges table (Major: wrong data, silently, on a clinical limit).** Not counted as a failure, because T-19 only covers a different setting. It has the same effect as verification 2's F-1, but needs a narrower set of events: both admins must be editing the ranges table.
  - **Signed in as:** Ada Admin, Settings, the whole trust, two tabs at http://localhost:3000/o/…/h/…/settings. Trust A already has its own heart rate range.
  - **Steps:**
    1. Tabs 1 and 2 both show Heart rate Usual to **105**.
    2. Tab 2: change it to **108**, Save. `GET …/settings` shows `softMax: 108`.
    3. Tab 1, no reload (still showing 105): change only **Weight** Usual to, to 190, Save.
  - **Expected:** heart rate stays 108, or the second save is told someone else changed the table.
  - **Actual:** "Trust A's settings were saved." Tab 1's PATCH holds the whole table, `heartRate.softMax: 105` included, and the stored value goes **back to 105**. Neither admin is told.
  - **Why, as far as I can see from the PATCH:** the ranges are one setting, so any edit to one row sends every row as that page last read them.
  - No console or network errors.
- **Scheduled Jobs, where the time helpers moved from:** presets and the box fill correctly. Year 20266 and Run jobs gives "Enter a date and time.", and nothing runs. I didn't run the jobs for real, as that would change other test data.
- **A red message stays while you type a new value (Cosmetic):** after 500 is refused on the Correct form, or 160 on Add a reading, the red line stays until the next Save. The yellow box does close when the value changes, so the two behave differently.
- **Settings' Save moves after a save (Cosmetic):** the green or blue message appears above the button and pushes it off the screen, so a second click on the same spot lands on the message and does nothing. It cost me one retry. Same kind as report 4's point about Save reading moving.
- **Already known (pending.md):** with observations off, each page open writes refused audit rows. This round: 5 `observation.list` 403 rows, from two page opens (doubled by dev's effects) plus my own `GET`.
- **Good:** a stale tab shows the new value once its own save reloads the page data. A row set back to the default leaves the stored value, so it follows Preopsis's numbers again.

## Console and network errors not tied to a case

- **Chrome console:** no errors or warnings, read from just after T-01 onwards. Tracking starts when first read, so the first page load wasn't captured.
- **Headless Chromium:** no page errors or console errors in the T-20 run.
- **False alarm, tooling only:** the extension shows 503 for `POST /api/logout`, `POST /api/dev/login` and a task PATCH. Each worked on screen. It's report 4's false alarm for empty answers. The audit log has no 5xx from 16:20 to the end of this round.
- **Five 500s from before this round, cause unknown:** the audit log has 500s on `episode.list` at 16:10:19, 16:14:07, 16:17:23 and 16:17:58, and on `episode.view` at 16:13:54, by `staff`. None is an observations route, and none is on my episode. Episodes from E2E journeys were being created at about the same time. The audit row doesn't say why, I didn't cause them, and I couldn't reproduce them; the Go log is in Rahul's terminal.

## Hand checks for Rahul

- T-12 and the patient's view at 390×844 in **real Chrome**: the window wouldn't shrink. Headless Chromium at that size passed (T-20 this round; the rest in report 4).

## GIFs and screenshots

- `~/Downloads/S46-retest5-happy-path.gif`: the main happy path (T-01, T-20, T-15, T-13).
- In the scratchpad `/private/tmp/claude-501/-Users-rahul-poised/01ba4f3d-8e60-4980-86ab-566e7dafbb3f/scratchpad/s46r5/`:
  - `t20-390.png`: the T-20 message at 390×844;
  - `print-ok.png`, `print-fail.png`: the POA Summary's print layout, with the BMI and with "Couldn't be loaded";
  - `poa.pdf`, `poa-fail.pdf`: the same as PDFs, rendered but not opened, as there's no PDF reader on this machine. The print results above come from the page's text and screenshots under print styles.
