# S11 manual test plan: the audit log

For a tester who hasn't seen the code. Use only the browser.

**Changed after test report 2 (for the retest):** new cases **T-19** (a refused search keeps the person chosen and can't be exported, and a refused export is logged as an export), **T-20** (a link with an impossible day names it and empties that box, so Show works again) and **T-21** (a loading sign). **T-18**'s expected message is now "Enter valid dates. There's no 2026-02-30." Run T-17 to T-21, plus a quick T-10.

**Changed after verification 1:** new cases **T-15** (a list row shows a count and at most ten names), **T-16** (looking up a patient in the log is itself logged against them), **T-17** (a refused search shows no old rows) and **T-18** (an impossible day in the address is reported). Run these four, plus a quick T-01, T-07 and T-10.

**Changed after test report 1 (all passed):** T-12 step 3 now says that switching to a hospital where you don't manage the trust goes to Episodes; the refusal message is checked by typing the `/audit` address.

**What S11 built:** every request that reads or changes patient data (opening a patient list, the episode list, an episode, its answers, saving answers, notes, status changes, the patient's own link) now writes **one row** to the trust's audit log: who, when, which hospital, what, and which patients. The trust's whole-trust admins read it on a new **Audit** screen, filter it by person, days and patient, and **export it as a CSV**. Nobody can change or delete a row.

## Setup

**Rahul, in your own terminal** (the automatic run does this itself):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

**Open:** http://localhost:3000. It goes to the development sign-in page: click a person to sign in as them. **Sign out** is in the page header.

**A second person at the same time:** open **http://tester.localhost:3000** in another tab. It's the same app but keeps its own sign-in, so you can be two people at once (e.g. Cara in one tab, Ada in the other).

**Seeded people:**

| Person | What they are |
| --- | --- |
| Ada Admin | Admin of the whole of Trust A (Hospitals A1 and A2): **sees the Audit screen** |
| Alex Author | Super clinician of the whole of Trust A |
| Sue Super | Super clinician of Hospital A1 only |
| Cara Clinician | Clinician of Hospital A1 only |
| Val Clinician | Clinician of Hospital A2 only |
| Bea Author | Super clinician of the whole of Trust B (Hospital B1) |
| Max Multi | Clinician at Hospital A2, and **admin of the whole of Trust B** |
| Sam Superadmin | Poised's platform admin; no trust |

**Seeded patients (Trust A):** Jo Bloggs (TEST0001), Pat Example (TEST0002), Robin Sample (TEST0003). Earlier automated runs leave many more ("Sam J4 …", "Sam Audit …"); that's expected.

**Good to know:**
- **In `make dev`, most views appear twice in the log.** The development build of React loads each page's data twice on purpose (StrictMode), so opening an episode writes two "Opened an episode" rows a fraction of a second apart. The production build loads once. Don't report the pairs as a bug; do report a view or change with **no** row.
- **Reading the log is itself logged:** every time the Audit list loads (opening it, or pressing **Show**), a row "Viewed the audit log" is added, and every export adds "Exported the audit log". So the newest row is often Ada's own.
- The list shows the newest **500** rows; the CSV has every row that matches.
- Times on the screen and in the CSV are in your browser's time zone.
- The Claude in Chrome network log shows a `204` response as `503`; ignore that.
- You may need a published HQ to create an episode: **Alex Author** → Questionnaires shows the published ones. If none is published, Alex can publish any draft (Questionnaires → open it → Publish).

## Test cases

### Rows are written

**T-01: A clinician's views and changes appear in the audit log** (manual test focus 1)
1. In one tab, sign in as **Cara Clinician**. Go to **Episodes**.
2. Click **New episode**, choose **New patient**, and create a patient with a surname you'll recognise, e.g. first name `Tess`, surname `Audit01`, any date of birth and sex, and any published HQ. Save. (This writes "Added a patient" and "Created an episode".)
3. You're on the new episode. Change its **status** or the **procedure**, then add a **comment** in General notes ("Add comment" → type → "Save comment").
4. Go back to **Episodes** (the list).
5. In a second tab (http://tester.localhost:3000), sign in as **Ada Admin**. Click **Audit** in the header.
6. Expected: the Audit log page, newest first, with columns When, Who, What, Patient, Hospital, Result. Near the top you see Cara Clinician's rows, Hospital A1, Result **Done**:
   - "Listed episodes" (Patient shows e.g. "12 patients"; hover it to see some names);
   - "Added a note", "Changed an episode", "Opened an episode" and "Opened the answers" with **Tess Audit01** as the patient;
   - "Created an episode" and "Added a patient" for Tess Audit01;
   - "Listed patients" (from opening the New episode form, which lists the trust's patients).
7. Expected: no patient details other than the name (and hospital number) are shown: no date of birth, phone or answers.

**T-02: Changing an answer while validating is logged**
1. As **Cara** (first tab), open an episode whose status is **Ready for review** (if there's none: as the patient, open an episode's link from its page (the "Patient link"), answer and send it; or ask Rahul to run `make e2e`, which leaves several).
2. Click **Validate** on a Question Set, change one answer, and wait for "Saved".
3. As **Ada** (second tab), press **Show** on the Audit screen.
4. Expected: Cara's rows "Saved the clinician's answers" and "Opened the answers" for that patient, Hospital A1, Done.

**T-03: The patient's own link is logged with "Patient (their link)" as who**
1. As **Cara**, open any episode still at **HQ not complete** and copy its **Patient link**. Open the link in a new tab (no sign-in needed). Answer a question in the first Question Set.
2. As **Ada**, on Audit, choose **Who: Patients (their own link)** and press **Show**.
3. Expected: rows "Patient opened their questionnaire" and "Patient saved answers" (and "Patient sent their answers" if you sent it), Who **Patient (their link)**, the patient's name, the episode's hospital.
4. Edit the link's last few characters so it's wrong and open it. Expected: "This link doesn't work…", and **no** new row (press Show again to check).

**T-04: A refusal after access is checked is logged with its result**
1. As **Cara**, open an episode, then change the address bar: replace the episode id at the end with the id of an episode at **Hospital A2** (as Ada: Episodes → All hospitals → open an A2 episode, copy the id from its URL). Press Enter.
2. Expected: Cara sees "Episode not found." (or similar).
3. As **Ada**, filter **Who: Cara Clinician** and press Show. Expected: a row "Opened an episode", Patient "—", Result **Refused (404)**.

**T-05: Screens without patient data write nothing**
1. As **Ada**, note the newest row on Audit (it will be "Viewed the audit log").
2. Open **Questionnaires**, **Staff** and **Settings**, then come back to **Audit**.
3. Expected: the only new rows are "Viewed the audit log" (from coming back); nothing for Questionnaires, Staff or Settings.

**T-06: All hospitals is logged as the whole trust**
1. As **Ada**, go to **Episodes** and choose **All hospitals**.
2. Back on **Audit**, expected: a row by Ada Admin "Listed episodes (all hospitals)", Hospital **Whole trust**.

### Filters

**T-07: Filter by person** (manual test focus 2)
1. As **Ada** on Audit, open **Who**. Expected: "Patients (their own link)" first, then each person who appears in the log, as "Name (email)", e.g. "Cara Clinician (cara.clinician@hospital-a.example)".
2. Choose **Cara Clinician** and press **Show** (or press Enter in a box). Expected: only Cara's rows; the count above the table ("N rows") changes.
3. Expected: the address bar now ends with `?user=…`. **Reload** the page: the same filter and rows come back.
4. Press **Clear**. Expected: everyone's rows again and the filters are empty.

**T-08: Filter by days** (manual test focus 2)
1. Set **From** and **To** to today and press Show. Expected: today's rows.
2. Set **From** and **To** to tomorrow and press Show. Expected: "Nothing matches these filters."
3. Set **From** to yesterday and **To** to yesterday. Expected: nothing, unless the log has rows from yesterday (earlier runs).
4. Set **From** after **To** (e.g. From tomorrow, To today) and press Show. Expected: red message "The start date must be before the end date."; the list doesn't change.
5. Press the browser's **Back** button. Expected: the previous filters come back in the form, with their rows.

**T-09: Filter by patient**
1. Type `Audit01` (T-01's patient's surname) in **Patient** and press Show. Expected: only rows touching Tess Audit01: the views and changes of their episode, plus each "Listed episodes" / "Listed patients" that listed them.
2. Try `tess audit01` (lower case), `Audit01 Tess` (surname first), and a hospital number such as `TEST0001` (Jo Bloggs). Expected: each finds that patient's rows.
3. Try `Nobody-Like-This`. Expected: "Nothing matches these filters."
4. Combine **Who: Cara Clinician** with **Patient: Audit01**. Expected: only Cara's rows for that patient.

### Export

**T-10: The CSV matches the list** (manual test focus 3)
1. As **Ada**, filter **Who: Cara Clinician** and **Patient: Audit01**, press Show, and note the rows.
2. Click **Export CSV**. A file `audit-<date>-<time>.csv` downloads. Open it (a spreadsheet, or a text editor).
3. Expected: a header row `Time, Who, Email, Action, Hospital, Patients, Patient record IDs, Episode IDs, Route, Result`, then **the same rows as the list, in the same order**: same actions, hospital and patient names; Result is a number (200, 201 or 204 for "Done"; 404 etc. for refusals); Time is in your time zone, e.g. `2026-10-13T09:41:07+01:00`, and matches the list's time.
4. Clear the filters and export again. Expected: every row of the trust's log (more than the list if it says "Showing the newest 500 rows"). The newest row is "Viewed the audit log" (the list load before the export); the export's own row appears the next time you press Show.

### Who can see it

**T-11: A clinician can't see the Audit screen** (manual test focus 4)
1. Sign in as **Cara Clinician**. Expected: the header has no **Audit** link.
2. Type the Audit URL yourself: take Cara's address, e.g. `…/o/<trust>/h/<hospital>/episodes`, and change the end to `/audit`. Expected: "Only the trust's admins can see the audit log", and no table.
3. Repeat as **Alex Author** (whole-trust super clinician) and **Sue Super** (super clinician of A1). Expected: the same.

**T-12: Trust B's admin sees nothing of Trust A** (manual test focus 5)
1. Sign in as **Max Multi**, choose **Hospital B1** in the header, and click **Audit**.
2. Expected: Trust B's log only. Filter **Patient: Audit01** → "Nothing matches these filters." The **Who** list doesn't include Cara or Ada.
3. Switch to **Hospital A2** in the header (Max is only a clinician there). Expected: the Audit link disappears and you land on **Episodes** (switching keeps Audit only in a trust you manage). Then type A2's Audit address yourself (change the end of the URL to `/audit`). Expected: "Only the trust's admins can see the audit log", and no table.
4. As Max (at Hospital B1), put Trust A's id in the address (copy it from Ada's Audit URL: `/o/<Trust A id>/h/<Hospital A1 id>/audit`). Expected: "You don't have access to this hospital" (Max isn't at Hospital A1).

**T-13: Switching hospital keeps Audit**
1. As **Ada** on Audit, switch to **Hospital A2** in the header. Expected: you stay on Audit (same trust's log).

**T-14: Rows can't be changed or deleted**
- There's no edit or delete control anywhere on the Audit screen. (The database refuses changes too; that's covered by automated tests, so nothing to do here.)

### Fixes after verification 1

**T-15: A list row shows how many patients it touched, and at most ten names**
1. As **Cara**, open **Episodes** (it lists every episode of Hospital A1), then open **New episode** and close it (it lists the trust's patients).
2. As **Ada**, on Audit, choose **Who: Cara Clinician** and press Show.
3. Expected: "Listed episodes" and "Listed patients" rows show e.g. "301 patients". Hovering shows ten names, then "and 291 more" (the number is the count minus ten).
4. Export the CSV with the same filter. Expected: those rows' **Patients** cell lists **every** name (separated by "; "), and **Patient record IDs** every id.
5. The page should load quickly even with no filters (a few hundred kilobytes; before the fix it was about 4 MB).

**T-16: Looking up a patient in the log is itself logged against that patient**
1. As **Ada**, filter **Patient: Bloggs** (Jo Bloggs) and press Show. Then press **Export CSV**.
2. Clear the filters, then filter **Who: Ada Admin** and **Patient: Bloggs**, and press Show.
3. Expected: rows "Exported the audit log" and "Viewed the audit log" by Ada, with **Jo Bloggs (TEST0001)** as the patient: the lookups of Jo's history now show in Jo's history.
4. Filter **Who: Ada Admin** only. Expected: "Viewed the audit log" rows from **unfiltered** views show Patient "—" (an unfiltered view names nobody).

**T-17: A refused search shows the message and no old rows**
1. As **Ada** on Audit (rows showing), type 101 or more characters in **Patient** (e.g. paste a long line of x's) and press Show.
2. Expected: red "The patient search is too long." and **no table** (before the fix the previous rows stayed under the message).

**T-18: An impossible day in the address is reported, not ignored**
1. As **Ada**, open the Audit address with `?from=2026-02-30&to=2026-03-05` added to the end (30 February doesn't exist).
2. Expected: red "Enter valid dates. There's no 2026-02-30.", no table, and **Export CSV** greyed out. Press **Clear**: the rows come back.

### Fixes after test report 2

**T-19: A refused search keeps the person chosen, can't be exported, and a refused export is logged as an export**
1. As **Ada** on Audit, choose **Who: Ada Admin** and press Show.
2. Type 101 or more characters in **Patient** and press Show.
3. Expected: red "The patient search is too long.", no table, **Who still shows "Ada Admin (ada.admin@trust-a.example)"** (not "Someone else"), and **Export CSV is greyed out** (it can't be clicked).
4. Press **Clear**. Expected: rows back, and Export CSV active again.
5. Open the export address with a refused filter yourself: in the address bar, replace the page's address with `/api/o/<Trust A id>/audit?format=csv&user=nope` (the Trust A id is the part after `/o/` in the Audit address). Expected: the browser shows `{"error":"Choose someone from the list."}` (or downloads it).
6. Back on Audit, press Show with no filters. Expected: the newest rows include Ada's **"Exported the audit log"** with Result **Refused (400)** (before the fix it said "Viewed the audit log").

**T-20: A link with an impossible day names it, and Show works after fixing it**
1. As **Ada**, open the Audit address with `?from=2026-02-30&to=2026-03-05` on the end.
2. Expected: "Enter valid dates. There's no 2026-02-30.", no table, Export CSV greyed out. **From is empty**; **To** shows 05/03/2026 (or your browser's format).
3. Empty **To** too (or choose a From and To of your own) and press **Show**, without pressing Clear. Expected: the rows appear, and Export CSV is active.

**T-21: A loading sign while the list loads**
1. As **Ada**, reload the Audit page. Expected: a small spinner shows under the filters until the table appears (it may be very brief on a fast machine).

## Regression (earlier sessions)

**R-1 (S03/S04): The patient's journey still works at phone size.** Open a patient link at **390×844**, answer every Question Set and **Send**. Expected: the thank-you screen; the episode moves to Ready for review. (Then T-03's rows show it.)

**R-2 (S08): Episodes and All hospitals.** As **Cara**, Episodes lists Hospital A1's episodes only, and an episode opens. As **Ada**, All hospitals lists both A1 and A2.

**R-3 (S09): Staff.** As **Ada**, Staff lists the trust's people; invite `zoe.s11@test.example` as a Clinician of Hospital A1; they appear in the sign-in list.

**R-4 (S10): Settings.** As **Ada**, Settings opens; change the trust's phone and Save; the green "saved" message shows beside Save.

**R-5 (S02a): Validating.** As **Cara**, validate a Question Set of a Ready-for-review episode; it saves, and "Complete HQ review" works once every set is validated.

## Out of scope (don't report)

- Rows appearing **twice** in `make dev` (React's development double-load; see "Good to know").
- "Viewed the audit log" rows from your own use of the Audit screen.
- No IP address or browser in the rows; no paging beyond 500 rows on screen (export the CSV); the CSV's columns and wording.
- Removing old rows after the retention period (not built: nothing is deleted for now, Rahul's answer).
- Authoring and questionnaire changes aren't in this log (it's patient data only; AUD-01's questionnaire history is later work).
- Real sign-in (Cognito, S12), real email and SMS.
