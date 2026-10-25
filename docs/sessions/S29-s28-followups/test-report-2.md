# S29 test report 2 (automatic run)

**Summary:** 17 passed, 0 failed, 0 blocked. That's T-01 to T-11 and R-01 to R-06, plus four extra checks while exploring.

**Environment:**
- HEAD `91c2409`. The working tree isn't clean: it holds S29's uncommitted changes, plus `docs/market/`, which I left alone.
- Sunday 25 Oct 2026, 14:23 to 14:50 GMT.
- URL http://localhost:3000. The conductor's `make dev` was running (API :8080 `{"status":"ok"}`, web 200).
- Browser: Claude in Chrome, profile "trk" (`bac77f71-…`). This time it reached localhost.

## How the test was run (differences from the plan)
- **Phone size:**
  - This Chrome window wouldn't resize to 390 × 844; the viewport stayed at 1126 wide.
  - So every patient page ran inside a same-origin **390 px-wide iframe** on the app's own page. The app lays out by the iframe's viewport, so it's a true 390 px layout.
  - The overflow check was `document.documentElement.scrollWidth` inside the frame: 390 every time.
  - T-04's "two tabs" were two 390 px iframes side by side, sharing the patient's session.
- **Network failures** (T-05, T-06): in the browser only, by overriding `window.fetch` exactly as the plan says. For patient pages I overrode the iframe's `fetch`. A reload put the network back each time. The Go API was never touched.
- **Setup made through the API, not the UI.** These are the same calls the e2e helpers make (`e2e/support.py`): Bea's episode, the patient's answers and Send, then Cara's follow-up episode. I used them for:
  - the six people with a Trust B history: S29 T02a, T02b, T03a, T03b, T04 and T06;
  - T-08/T-09's five episodes (E1–E5);
  - E2 and E5's answers and Send;
  - E5's validation and Complete review;
  - the T-10 episode.

  Everything the cases test went through the UI:
  - every pre-fill card button (patient and staff);
  - Fill in for the patient;
  - Add task, Cancel, Archive and Un-archive;
  - the Audit filters and Export CSV;
  - the T-07 episode, created with New episode;
  - authoring and publishing "S29 BMI test MT";
  - the patient answering and sending T-10;
  - Validate;
  - Same as….

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Audit page loads | Pass | Who / From / To / Patient filters, a What column, and Show with no filters lists rows. |
| T-02 Patient's three buttons | Pass | Exactly one row each: **Patient chose to check each pre-filled answer** (T02a), **Patient confirmed pre-filled answers** (T02a) and **Patient started a Question Set fresh** (T02b). No "Patient checked pre-filled answers". After Start fresh, T02b's About you opened empty, with no card and no red banner. The CSV (`~/Downloads/audit-2026-10-25-143419.csv`, 14 rows) has the same three Action wordings. |
| T-03 Staff's three buttons | Pass | Exactly one row each: **Checked each pre-filled answer with the patient** and **Confirmed pre-filled answers with the patient** (T03a), and **Started a Question Set fresh for the patient** (T03b). All by Cara Clinician at Hospital A1. No old shared wording. |
| T-04 Second button in another tab | Pass | Tab 1 pressed Everything is still correct; tab 2, still on the card, pressed Check each answer. Neither showed an error. The Audit has one **Patient confirmed pre-filled answers** row and one **Patient chose to check each pre-filled answer** row. |
| T-05 Staff page network error | Pass | POA Summary → Save comment, and the episode page → Add task (Nurse): both show **"We couldn't reach the server. Check your connection and try again."** No "Failed to fetch" anywhere. After a reload, Save comment saved. |
| T-06 Patient page, and Start fresh whose reload fails | Pass | With every request failing, changing snoring showed the plain red message and "Not saved", with no "Failed to fetch". With only reads failing, Yes, start fresh showed only the **"The answers in this section were cleared"** card ("…couldn't be loaded again just now. Please try again." and **Reload**), with no red banner. Width 390. After a reload, About you shows **Start** (no card) and opens empty. |
| T-07 No open-task count once archived | Pass | Before: **Tasks (1 open)**, and the list says "1 open task". Archived: the title has no count, the task is still listed, and the page says "This episode is archived, so its tasks can't be changed." With Show archived, the row reads ARCHIVED with no count. After Un-archive, "(1 open)" and "1 open task" are back. A cancelled episode with an open task (T08 E1) still shows "1 open task". |
| T-08 Archived from Cancelled: the patient sees the cancelled message | Pass | E1 (opened, never sent) and E2 (sent, then cancelled), both archived, show "This questionnaire was cancelled by the hospital, so there's nothing more to fill in. Please contact the hospital if you have any questions." The home shows both as **Cancelled by the hospital**. |
| T-09 Archived from another status | Pass | E4 (HQ not complete → archived): "This questionnaire is closed, so there's nothing more to fill in…". E5 (validated, Ready for POA → archived): "Thank you, Sam. Your answers have been sent to the hospital.", with no review promise. The home shows E5 **Sent** with **View**, and E4 **Closed by the hospital**. |
| T-10 Hidden height doesn't feed the BMI | Pass | Patient: Yes, 170, 82, then No, then sent. The POA's BMI is "—" on both tabs. Validated with Yes and 170: **28.4** on both tabs. Validated again with No (the clinician's row still holds `height: 170`): "—" on both tabs. Regression: the Demo pre-op HQ (T08 E2: 170/82) shows **28.4** on both tabs. |
| T-11 "Same as…" warns before replacing a link | Pass | On "Do you smoke?" from the library, the orange line reads: This question is already linked to “Do you smoke?”. Using this replaces that link. On a new Select One: no line. On "How tall are you?" (T-10's new version): the warning names "How tall are you?". After Use this → pregnancy, reopening warns: …linked to “Is there any chance you could be pregnant?”. Another new question shows no leftover warning. |
| R-01 Fill in for the patient | Pass | Cara on T03a/T03b (never opened by the patient): Trust B's answers are filled in, and the card names Hospital B1. |
| R-02 Start fresh on the POA Summary, and BMI before validation | Pass | T03b after Cara sent it: both tabs say "…Cara Clinician chose “Start fresh” with the patient on 25 Oct 2026, so the pre-filled answers were cleared…". The BMI before validation is 25.7 (165 cm / 70 kg). |
| R-03 Answer click → Validate at that question, and back | Pass | Clicking "Do you snore loudly?" on the Patient answers tab opened the set at `?q=snoring` with the question highlighted. Back returned to `?tab=patient` with Patient answers selected. |
| R-04 Archived episode is read-only | Pass | The status is disabled. The only buttons are Sign out and "Un-archive: back to HQ not complete": no Add comment, no Add task. Un-archive restores them all. |
| R-05 Same as… gives the ID and options and keeps the text | Pass | See the note below. Use this → pregnancy gave `ID pregnant`, Yes / No / I'm not sure, and kept "MT S29 new question". Use this → "Do you snore loudly?" on another new question gave `ID snoring`, Yes / No, the label **Pre-fills from 12 other HQs**, and kept its text. |
| R-06 Audit filters and Export CSV | Pass | Who (Patients; Cara), Patient and Export CSV were used in T-02 and T-03. Date filter: patient "S29 T02" from 24 Oct to 24 Oct → 0 rows ("Nothing matches these filters."); 25 Oct to 25 Oct → 38 rows. |

**Note on R-05 / T-11 step 6:**
- The plan's example candidate, "Is there any chance you could be pregnant?", is an **Always ask fresh** question. The window says "ALWAYS ASK FRESH, LIKE THE ORIGINAL".
- So after Use this the label reads **Always ask fresh**, not "Pre-fills from N other HQs". That is consistent with the design, not an app fault.
- I checked the "Pre-fills from N" label with "Do you snore loudly?" instead, and it showed.
- The plan's wording for R-05 could name a non-fresh candidate next time.

## Failures in detail
None.

## Found while exploring
1. **A hidden height held in the patient's own answers (stronger T-10 check):**
   - Through the patient API, I sent a "S29 BMI test MT" episode (S29 T10b) whose stored patient answers are `{q: No, height: 170, weight: 82}`. Through the UI, SurveyJS clears the hidden height on Send, so T-10 step 3 never stores it.
   - The POA Summary shows BMI "—" on both tabs. The new rule holds before validation as well as after.
2. **The plain message on page loads too:**
   - Staff: with every request failing, opening Episodes from the top bar shows the same red "We couldn't reach the server…" message.
   - Patient: the same failure, then View on their home, shows the same message.
   - No "Failed to fetch" anywhere.
3. **Another patient's episode id:** a signed-in patient (T10b) opening `/home/episodes/<T08 E3's id>` gets "Episode not found." (API 404). No data shows.
4. **Emoji in a note:** "MT S29 double click 😀" saved and displays correctly.
5. **Known, not new:**
   - **Double-clicking Save comment saves the note twice** (two `events` in the episode). Already in `docs/pending.md` ("A double-click on 'Save comment' saves the note twice"). I left two notes on Sam S29 T03a.
   - **Same as… squeezes the Use this button:** it becomes an unlabeled teal square, or "Us", when a candidate's "In …" list of HQ names is long. The many E2E HQs make this happen for "Do you have diabetes?" and "Do you snore loudly?". Already in `docs/pending.md` (the Use this button squeezed to a sliver).
6. **Small observations, no action asked:**
   - The patient's cancelled / closed / not-found pages show no hospital name at the top right, unlike the form pages. Probably as before S29.
   - "Everything is still correct" on Cara's fill-in for T03b's Your health showed the whole first page, not a jump to the first unanswered question. The question (pregnancy) is on the same page, so this may be intended.

## Console and network errors
- **Console:** no errors or warnings in the tab while tracking was on.
- **Network:**
  - No unexpected 4xx/5xx from the app.
  - The extension's request log lists **503** for some POSTs that answer **204** (`…/prefill/{cid}`, `…/submit`, `patient-form/submit`). Each one succeeded:
    - the page went on as normal;
    - my own `fetch` of the same call returned 204;
    - the episodes are `ready_for_review`;
    - the audit CSV's Result column shows 204.
  - So this is the recording tool mislabelling empty responses, not an app error.
- The only intended failures were the overridden `fetch` calls in T-05 and T-06, which never left the browser.

## Hand checks for Rahul
None needed for this plan. Not run, on the conductor's instruction: the gateway-502 case. The Go API was never stopped.

## GIFs
Saved to `~/Downloads`:
- `S29-T02.gif`: the main happy path. Patient T02a presses Check each answer, then Everything is still correct; patient T02b presses Start fresh → Yes, start fresh. At 390 px.
- `S29-T06.gif`:
  - the network failure on a patient page;
  - Start fresh with reads failing (only the "cleared" card);
  - the reload, after which About you is empty.

## Test data left behind (all made up)
- **Patients:** Sam S29 T02a, T02b, T03a, T03b, T04, T06, T07, T08 (E1–E5), T10 and T10b.
- **Questionnaire:** "S29 BMI test MT" (v1 published; v2 draft with the extra Same as… questions).
- **Two notes on Sam S29 T03a:** "MT S29 T05 comment after reload", and the duplicated "MT S29 double click 😀".
- **Episodes:**
  - T07 is un-archived, with one open Nurse task.
  - T08 E1 and E2 are cancelled then archived; E4 and E5 are archived.
