# S11 test report 1: the audit log

**Summary:** 19 passed, 0 failed, 0 blocked. Two hand checks for Rahul are listed below. They are parts of cases that passed, which my tools couldn't do the way a person would.

**Environment:**
- Commit `5897b3b`. The working tree is not clean: the S11 code and the planning docs are uncommitted, as expected in the automatic run.
- 13 Oct 2026, 09:18–09:37 BST.
- `make dev`, run by the conductor: http://localhost:3000, with http://tester.localhost:3000 as the second person.
- Chrome, via Claude in Chrome. The window couldn't be resized: `resize_window` had no effect and the viewport stayed at 2560×1296. So the phone-size check (R-1) ran inside a 390×844 iframe on a localhost page.

**My test data:**
- **Tess AuditMT0919** (Hospital A1): T-01's patient. I used this surname instead of the plan's `Audit01`, so it wouldn't clash with earlier runs.
- **Zed 🙂 =MTInject0935** (MTS110935): created while exploring.
- **Zoe S11 Tester** (zoe.s11@test.example): invited in R-3.
- Trust A's patient phone number is now set to `01632 960111` (R-4).
- Sam J5 1310 091731 7581 is now **On hold** (T-01's status change).

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Clinician's views and changes are logged | Pass | Cara's rows are all there, Hospital A1, Done: Listed patients (328 patients), Added a patient, Created an episode, Opened an episode / Opened the answers (Tess AuditMT0919), Added a note, Listed episodes ("301 patients", and hovering shows names + "and 291 more"). Only the name and hospital number are shown. The new episode's status and procedure can't be edited while the HQ isn't complete, so I changed the status on another episode (Sam J5, POA complete → On hold): "Changed an episode", Done. |
| T-02 Changing an answer while validating | Pass | "Saved the clinician's answers" and "Opened the answers" for Sam J3 1310 091721 2ada, Hospital A1, Done. |
| T-03 Patient link logged as "Patient (their link)" | Pass | "Patient opened their questionnaire" and "Patient saved answers" (later also "Patient sent their answers") for Tess, Hospital A1. A wrong link shows "This link doesn't work. Please check it, or contact the hospital." and writes no row. |
| T-04 Refusal after the access check is logged | Pass | Cara using an A2 episode id under A1's URL sees "Episode not found." The log shows "Opened an episode", Patient "—", Hospital A1, **Refused (404)**. |
| T-05 Screens without patient data write nothing | Pass | Questionnaires, Staff and Settings wrote no rows. Coming back to Audit wrote only "Viewed the audit log". |
| T-06 All hospitals is logged as the whole trust | Pass | Ada: "Listed episodes (all hospitals)", 321 patients, **Whole trust**. |
| T-07 Filter by person | Pass | The Who list is "Patients (their own link)" first, then "Name (email)" for Ada, Alex, Cara, Max, Val and two E2E Zoes. Choosing Cara gives 413 rows, all Cara's. The URL gets `?user=…`, and reloading keeps it. Clear brings back everyone (500, "Showing the newest 500 rows") and empties the form. |
| T-08 Filter by days | Pass | Today/today: 500 rows, all 13/10. Tomorrow/tomorrow: "Nothing matches these filters." Yesterday/yesterday: nothing (the database confirms every audit row is from today). From after To: red "The start date must be before the end date." and the list doesn't change (checked with both an empty and a full list). Back restores the earlier filters and their rows. |
| T-09 Filter by patient | Pass | `AuditMT0919`: 22 rows (Tess's own rows, plus the list rows that listed her, including Ada's All hospitals). `tess auditmt0919` and `AuditMT0919 Tess`: the same 22. `TEST0001`: 134 list rows. `Nobody-Like-This`: "Nothing matches these filters." Cara + AuditMT0919: 16 rows, all Cara's. Enter in the Patient box works like Show. |
| T-10 CSV matches the list | Pass (download is a hand check) | I fetched the Export CSV link's own URL inside the page instead of saving a file (see Hand checks). Filtered by Cara + AuditMT0919, the CSV has 16 rows and the list 16, and they match row for row: time, who, action, hospital, patient and result. Header as specified. Results are 200/201, and the full export also has 204, 404 and 409. Times look like `2026-10-13T09:21:49+01:00`. The filename from `Content-Disposition` is `audit-2026-10-13-092647.csv`. Unfiltered: CSV 744 rows against 500 in the list, and the newest row is "Viewed the audit log". After Show, "Exported the audit log" appears. |
| T-11 Clinician can't see Audit | Pass | Cara, Alex Author and Sue Super: no Audit link. Typing `/audit` shows "Only the trust's admins can see the audit log" with no table. The API answers 403 for the list and the CSV. |
| T-12 Trust B's admin sees nothing of Trust A | Pass (with a note) | Max at B1: 45 rows, all Max's, B1 or Whole trust. Patient `AuditMT0919` → nothing. Who has only "Patients (their own link)" and Max. Max with A1's URL → "You don't have access to this hospital". **Note on step 3:** switching to Hospital A2 in the header drops the Audit link but goes to **Episodes**, not to an Audit page with the message. This is by design (build notes: `switchPath` keeps Audit only where the user manages the trust). Typing A2's `/audit` URL does show "Only the trust's admins can see the audit log". The plan's wording should say this. |
| T-13 Switching hospital keeps Audit | Pass | Ada stays on `/h/…2a/audit`, with the same trust's log. |
| T-14 No edit or delete controls | Pass | The page's only controls are Export CSV, Who, From, To, Patient, Show and Clear. |
| R-1 Patient journey at phone size | Pass (see Hand checks) | Run in a 390×844 iframe: both sets answered, "Send your answers?" → Send → "Thank you, Tess". The episode moved to Ready for review, and the audit shows "Patient sent their answers". The layout fits at 390 px. |
| R-2 Episodes and All hospitals | Pass | Cara's list has A1 episodes only, and an episode opens. Ada's All hospitals shows A1 and A2 with a Hospital column. |
| R-3 Staff invite | Pass | Invited zoe.s11@test.example (Clinician, A1): green "Invitation sent…" banner. They're in the staff list and on the dev sign-in list. |
| R-4 Settings | Pass | Unticked "Use Poised's default" for the phone, entered a number and saved: green "Trust A's settings were saved." (It shows at the bottom of the form.) |
| R-5 Validating | Pass | Cara validated all three sets of Tess's episode (Nurse checks: blood pressure 120/80), then "Complete HQ review" → Ready for POA. The audit has "Completed the HQ review" and "Saved the clinician's answers". |

## Failures

None.

## Hand checks for Rahul

1. **T-10: the actual file download.** I didn't click Export CSV, because that saves a file to disk and in automatic mode I avoid downloads. Instead I fetched the button's exact link in the page and compared the content and the `Content-Disposition` filename. Please click **Export CSV** once, check the file `audit-<date>-<time>.csv` lands in Downloads, and open it in a spreadsheet.
2. **R-1 in a real phone-size window.** The browser window couldn't be resized, so the patient journey ran in a 390×844 iframe. That responds to the same media queries, but a quick look at a real 390×844 window or device mode is worth doing.

## Found while exploring

None of these is a failure:
- **Double submits:** double-clicking **Show** loads the list twice and writes two "Viewed the audit log" rows. That's expected, since each load is logged. Double-clicking **Create episode** created only one patient and one episode.
- **Odd filters are handled cleanly:**
  - Patient filter `%` or `_` → "Nothing matches" (no wildcard leak).
  - `?user=not-a-uuid` → "Choose someone from the list." (logged as a 400 "Viewed the audit log").
  - `?from=garbage&to=2026-02-30` → the bad days are ignored and every row shows.
- **Emoji and a leading `=` in a name:** Zed 🙂 =MTInject0935 shows correctly in the list and the CSV, which is UTF-8. The CSV's Patients cell starts with the first name, so the `=` wasn't at the start of a cell, and this didn't exercise the `'` defusing (the integration tests cover it).
- **Hospital number:** the screen shows the patient as "Name (hospital number)", but the CSV's Patients column has the name only. Its record IDs are in their own column. This is just an observation, since the CSV's columns are out of scope.
- **Who dropdown:** it is as narrow as the field, so "Name (email)" options wrap onto two lines. Cosmetic.
- **Hospital column:** Hospital A1 rows show no hospital number for patients without one (e.g. "Sam J3 … 2ada"), which is correct.
- **Opening an A2 episode via its own A2 URL** as Cara gives "You don't have access to this hospital" and writes no row. This matches the brief: only refusals after the membership check are logged.
- **Validate page:** the existing-patient search in New episode filters on the client. Opening the drawer writes one "Listed patients" row, and typing writes nothing more.
- **Database check (read-only):** `audit_log` has the append-only triggers. Every row is from today, and the only 400s are my `user=not-a-uuid` test.

## Console and network errors

- **Console:** no errors or warnings on the Audit page or the clinician pages.
- **Network:** the only unexpected-looking statuses are the `503`s for `/api/logout` and `/api/dev/login`. These are the extension showing `204` as `503`, as the plan says to ignore. The 403s were the ones I triggered on purpose (T-11, T-12).

## GIFs

None saved. Nothing failed, and exporting a GIF downloads a file, which I avoid in automatic mode. I recorded the T-01 happy path but discarded it.
