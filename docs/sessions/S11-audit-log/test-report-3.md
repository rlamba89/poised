# S11 test report 3: the audit log (retest after test report 2)

**Summary:** 6 passed, 0 failed, 0 blocked. I found two small cosmetic things while exploring (see below). Neither is a failure of a planned case.

**Environment:**
- Commit `5897b3b`. The working tree is not clean: the S11 code and the planning docs are uncommitted, as expected in the automatic run.
- 13 Oct 2026, 10:04–10:12 BST.
- `make dev`, run by the conductor and freshly migrated and seeded. Ada Admin was signed in at http://tester.localhost:3000 (Trust A, Hospital A1 → Audit).
- Chrome, via Claude in Chrome. The viewport is 1549 px wide.

**Scope:** as listed at the top of test-plan.md, T-17 to T-21, plus a quick T-10. Test report 2 had no failures or blocked cases to re-run. I also re-checked the four "found while exploring" items from report 2, because the fixes target them.

## Results

| Case | Result | Note |
|---|---|---|
| T-17 Refused search shows no old rows | Pass | 105 x's → red "The patient search is too long." and no table. Export CSV is greyed out: it's a link with no `href` and a not-allowed cursor, so clicking it does nothing. |
| T-18 Impossible day in the address | Pass | `?from=2026-02-30&to=2026-03-05` → red "Enter valid dates. There's no 2026-02-30.", no table, Export CSV greyed out. No audit API request is sent (only `/api/config` and `/api/me`). **Clear** → the address loses its query, both dates are empty, and 500 rows come back with Export active. |
| T-19 Refused search keeps the person, no export, refused export logged as export | Pass | Who: Ada Admin → Show, then 105 x's → Enter. The red message shows with no table. Who still shows "Ada Admin (ada.admin@trust-a.example)", and its list still has every person. Export CSV is greyed out. **Clear** brings back 500 rows, and Export is active again. Opening `/api/o/<Trust A>/audit?format=csv&user=nope` shows `{"error":"Choose someone from the list."}` in the browser (no download). Back on Audit, Show: the newest row is Ada's **"Exported the audit log"**, Whole trust, **Refused (400)**. The refused *views* from steps 2 and T-17 are still "Viewed the audit log", Refused (400), as they should be. |
| T-20 Bad-day link names it; Show works after fixing it | Pass | From is empty and To shows 05/03/2026. I emptied To with the keyboard and pressed **Show** without Clear: 500 rows, Export active, and the address drops the dates. Variant: I reopened the link and typed From 01/03/2026 (To still 05/03/2026), then pressed Enter. The address became `from=2026-03-01&to=2026-03-05`, with "0 rows / Nothing matches these filters." and Export active. |
| T-21 Loading sign | Pass | On reload, a small spinner shows under the filters before the table appears. I caught it by slowing the page's audit fetch by 4 s with a fetch wrapper. It goes away when the rows arrive. |
| T-10 CSV matches the list (quick) | Pass | Who: Cara Clinician plus Patient: `Sam A1 1310 100317` (an E2E patient Cara created today). The list has 3 rows (Opened an episode, Created an episode, Added a patient) and the CSV has 3, in the same order with the same times, actions, hospital and patient. Results are 200, 201 and 201, and times look like `2026-10-13T10:03:17+01:00`. The header is as specified. The good export is then logged as "Exported the audit log", Done, against that patient. I fetched the button's own link rather than saving a file, as in the earlier reports. |

**Report 2's exploring items, re-checked:**
1. Who showing "Someone else" after a refused search: **fixed** (T-19).
2. Export clickable after a refused search, and a refused export logged as a view: **fixed** (T-17, T-19).
3. An impossible day hidden in the form: **fixed**. The message names the day and the box is emptied, so Show works (T-20).
4. No loading indicator: **fixed** for the first load (T-21). See exploring item 1 for a remaining gap on later loads.

## Failures

None.

## Hand checks for Rahul

None new. The two from report 1 still apply if not done yet: clicking **Export CSV** for a real file download, and the patient journey in a real 390×844 window.

## Found while exploring

1. **No spinner when a changed filter loads; the old rows stay until the new ones arrive.** Cosmetic, S11. With the audit fetch slowed by 4 s, I typed Patient: Bloggs and pressed Show. For those 4 s, the address already said `patient=Bloggs`, but the page still showed the previous, unfiltered 500 rows with "Showing the newest 500 rows" and no spinner. Then it switched to Bloggs's 298 rows. The spinner (T-21) only appears when there's no table yet. On a normal local load this lasts well under a second, so it's barely visible.
2. **Opened straight onto a refused search, Who lists no people.** Cosmetic, S11. Opening an Audit address with a 101+ character `patient=` (no successful load yet in that page) shows the correct red message. But the Who list then has only "Patients (their own link)", because the people come from a successful response and there hasn't been one. It recovers after Clear. Similarly, a `?user=` with a well-formed ID that isn't anyone in the trust shows "Someone else" and 0 rows, which seems reasonable.
3. **Tool artefact, not a bug:** clicks sent right after a page load sometimes didn't reach the page. A capture listener confirmed no pointer or click events arrived at all. They worked after a screenshot brought the tab forward (the same as in report 2).

## Console and network errors

- **Console:** no errors or warnings on the Audit page.
- **Network:** the only 4xx responses were the 400s I triggered on purpose (the too-long search, loaded twice by the development double-load, and the `user=nope` export).

## GIFs

None saved: nothing failed, and exporting a GIF downloads a file, which I avoid in automatic mode.
