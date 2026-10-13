# S11 test report 2: the audit log (retest after verification 1)

**Summary:** 8 passed, 0 failed, 0 blocked. I found two small things while exploring. Both are in S11's area and come from the new fixes, but neither is a failure of a planned case (see "Found while exploring").

**Environment:**
- Commit `5897b3b`. The working tree is not clean: the S11 code and the planning docs are uncommitted, as expected in the automatic run.
- 13 Oct 2026, 09:50–10:00 BST.
- `make dev`, run by the conductor: http://localhost:3000 for Cara Clinician and http://tester.localhost:3000 for Ada Admin.
- Chrome, via Claude in Chrome. The viewport is 2560 px wide.

**Scope:** the four new cases T-15 to T-18, then quick runs of T-01, T-07, T-08 (steps 1, 4 and 5) and T-10. I also re-checked the Audit list and the CSV, because their data changed. Test report 1 had no failures or blocked cases to re-run.

**My test data:** **Tess AuditMT0955** (Hospital A1, procedure "Hernia repair", note "MT S11 retest note"), created by Cara for T-01.

## Results

| Case | Result | Note |
|---|---|---|
| T-15 List row shows a count and at most ten names | Pass | Cara's "Listed patients" shows "346 patients", and hovering shows ten names then "and 336 more". "Listed episodes" shows "313 patients". In the API response no row has more than 10 patients or 10 episode IDs, and each row also carries `patientCount` and `episodeCount`. The CSV filtered to Cara has every name: 346 names and 346 record IDs on Listed patients; 313 names, 313 IDs and 321 episode IDs on Listed episodes. The unfiltered list response is **408 KB** and took **69 ms** (before the fix it was about 4 MB). |
| T-16 Looking up a patient is logged against them | Pass | Patient: Bloggs → Show, then the export. Filtering Who: Ada Admin plus Patient: Bloggs shows "Exported the audit log" and "Viewed the audit log", both with **Jo Bloggs (TEST0001)**. Who: Ada only: the unfiltered and person-only views show "—". Extra check: `TEST000` matches three patients, and the database row for that view lists Jo Bloggs, Pat Example and Robin Sample. |
| T-17 Refused search shows no old rows | Pass | 105 x's → red "The patient search is too long." and no table (`document.querySelectorAll('table')` is 0). See the notes under "Found while exploring". |
| T-18 Impossible day in the address | Pass | `?from=2026-02-30&to=2026-03-05` → red "Enter valid dates.", no table, **Export CSV** greyed out, and no API request is sent. **Clear** brings the rows back. |
| T-01 Clinician's views and changes are logged (quick) | Pass | Cara created Tess AuditMT0955 and added a note. Ada's Audit shows, all Hospital A1 and Done: Listed patients (346), Added a patient, Created an episode, Opened an episode / Opened the answers, Added a note, and Listed episodes (314). Only name and hospital number are shown. I didn't repeat the status change: it's locked until the HQ is complete, and test report 1 already covered "Changed an episode". |
| T-07 Filter by person (quick) | Pass | Who lists "Patients (their own link)" first, then each person as "Name (email)". Choosing Cara writes `?user=…`, and only Cara's rows show. Opening the address with `?user=…` fresh restores "Cara Clinician (cara.clinician@…)" in the box. **Clear** empties the form and brings back everyone ("Showing the newest 500 rows"). |
| T-08 Filter by days (steps 1, 4, 5) | Pass | Today/today: today's rows. From 14/10 with To 13/10 → red "The start date must be before the end date.", and the list doesn't change. Back restores the earlier filters (empty) with their rows. |
| T-10 CSV matches the list (quick) | Pass | Who: Cara plus Patient: AuditMT0955: the list has 10 rows and the CSV 10, matching row for row on time, action, hospital, patient and result (200 or 201). The header is as specified, and times look like `2026-10-13T09:56:01+01:00`. I fetched the button's own link instead of saving a file, as in report 1. |

**Extra check (stale responses, part of the fix):** I delayed the `patient=Bloggs` request by 3 s with a fetch wrapper in the page. Then I applied Bloggs and immediately applied TEST0002. When the slow Bloggs response landed, the page kept showing TEST0002's rows (249 rows; Bloggs has 242) and the TEST0002 filter. **Pass:** the older response is ignored.

## Failures

None.

## Hand checks for Rahul

None new. The two from report 1 still apply if not done yet: clicking **Export CSV** for a real file download, and the patient journey in a real 390×844 window.

## Found while exploring

1. **After a refused search, Who shows "Someone else" instead of the chosen person.** Cosmetic, S11. Steps: as Ada, choose Who: Ada Admin and press Show. Then type 101+ characters in Patient and press Show. The Who box changes to "Someone else" while the address still has `user=<Ada's id>`. The people list comes from the last successful response, and the T-17 fix now clears that response on an error. Opening Who at that point lists only "Patients (their own link)" and "Someone else". It recovers on the next good search.
2. **Export CSV stays clickable after a refused search.** Minor, S11. Steps: as Ada, type 101+ characters in Patient and press Show. **Export CSV** is still active (it's greyed out only for impossible days, T-18). Its link answers **400** with JSON `{"error":"The patient search is too long."}`, so clicking it would download an error instead of a CSV. I didn't click it, to avoid a download; I fetched the link instead. The same applies to `?user=not-a-uuid`. The server also logs that refused export as **"Viewed the audit log"** (400), not "Exported the audit log". It parses `format=csv` only after the filters validate, so a refused export looks like a refused view in the log.
3. **An impossible day stays hidden in the form.** Cosmetic. After opening `?from=2026-02-30…`, the From box looks empty, because a date field can't show 30 February. Pressing Show still says "Enter valid dates." until you press Clear or pick a new From date. That's reasonable, but the user can't see which date is wrong.
4. **No loading indicator on the Audit screen.** Cosmetic. On first load the page shows the filters and nothing else for about a second before the table appears.
5. **Tool artefact, not a bug:** clicks sent to the Ada tab right after working in the Cara tab sometimes didn't reach the page. A capture listener saw no events at all. They worked after a screenshot brought the tab forward, so this isn't an app problem.

## Console and network errors

- **Console:** no errors or warnings on the Audit page or Cara's pages.
- **Network:** the only 4xx responses were the two 400s I triggered on purpose (the too-long search and its CSV link).

## GIFs

None saved: nothing failed, and exporting a GIF downloads a file, which I avoid in automatic mode.
