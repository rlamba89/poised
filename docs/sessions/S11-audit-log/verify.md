# S11 verification

## Verification 2 (13 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1's four fixes and test report 2's four exploring items are in, each with a test, and they work against `make dev`:
- the Audit screen's page is bounded;
- a patient search is recorded against that patient;
- stale or refused loads never leave the wrong rows on screen;
- an impossible day is named, and refused exports are logged as exports.

Every automated suite passes, and test reports 2 and 3 passed. Nothing found needs fixing before commit. The remaining items are follow-ups, now in [pending.md](../../pending.md) §3 ("Found by S11's verification 2").

### Verification 1's findings, re-checked

1. **The page grew with the trust: fixed.**
   - `auditListRows` (`apps/api/internal/httpapi/audit.go:354`) now gives the screen at most ten patients and ten episode ids per row, plus `patientCount` and `episodeCount`. The CSV keeps them all (`perRow = 0`).
   - De-duplication is a map-backed `idSet` (`audit.go:68`).
   - Test: `TestAuditListCapsPatientsPerRow`.
   - **Measured now:** the unfiltered page is **390 KB** for 500 rows (was 3.97 MB) and took 59 ms. No row has more than 10 patients or episode ids; the largest count is 358.
2. **A patient search wasn't recorded: fixed** (`audit.go:305–318`).
   - Test: `TestListAudit`, "searching the log for a patient…".
   - **Checked live:** after `patient=TEST0001`, Ada's lookups show under Jo Bloggs ("Viewed / Exported the audit log"), and an unfiltered view still names nobody.
3. **A stale response or an error left the wrong rows: fixed** (`AuditPage.tsx:76–99`, a `current` flag and `setPage(null)` on error).
   - **Checked live:** a 105-character search shows "The patient search is too long.", no table, and Export CSV greyed out.
4. **An impossible day was dropped silently: fixed** (`badDays` and `withoutBadDays` in `filters.ts`).
   - **Checked live:** `?from=2026-02-30&to=2026-03-05` shows "Enter valid dates. There's no 2026-02-30.", no table, and Export greyed out. Clear brings the rows back.
5. **Findings 5–8 are in pending.md, as asked.**

Test report 2's exploring items:
- **A refused export is logged as "Exported the audit log"** (`audit.go:296`). Test: `TestListAudit`, "a refused export…". Checked live: `format=csv&user=nope` → 400, then the newest row is `audit.export` with status 400.
- Who keeps the chosen person after a refused load; Export has no link while the filters are refused; there's a first-load spinner. All checked by test report 3.

### Findings (none blocks; all logged in pending.md §3)

1. **Filtered by a patient, a list row's ten names usually don't include that patient** (`audit.go:355–360`).
   - Measured: filtering by "Bloggs" gives 333 list rows ("352 patients" and similar), and none names Jo Bloggs among its ten. The row did touch her, and the CSV names her.
   - This is a side effect of fix 1. It's the most useful follow-up: put the matched patients first in each row when the filter is set (a few lines plus a test).
2. **Every list load stores every listed patient's id in its row** (`episodes.go:62`, `:124`). That's about 800 KB of `uuid[]` and GIN entries per list load at 50k patients.
   - This follows Rahul's answer ("the record IDs it touched").
   - Revisit before a large trust goes live, for example by storing the list's filters instead.
3. **A short patient search ("a") records every patient it matches** as looked up, and sends them all to `ListAudit`'s `&&` filter (`audit.go:315`). A minimum length or a cap would help.
4. **`audited` answers without a row when the trust is unknown** (`audit.go:119`).
   - Today only an unknown patient link takes this path, which is correct.
   - A future route wrapped outside `requireHospital`/`requireOrg` would skip the log silently. It's cheap hardening to answer 500 there unless the response is a 404.
5. **The spinner can stay on** after a bad-day link loads while an earlier load is still running (`AuditPage.tsx:69–75` doesn't clear `loading`). Reached only by Back or a link. Cosmetic.
6. **The export sends the browser's IANA zone,** which Go's zone database may not know. That export then answers 400, and the plain link saves the error as the file (verification 1's finding 6). `parseAuditFilter` also repeats `validTimeZone` (`admin.go:479`).

From `/code-review high`, these were weighed and already logged:
- the insert after the handler's write (retry duplicates);
- the CSV held in memory twice;
- the `DISTINCT` behind the Who list.

### Checked and fine

- **Scope:** the whole brief is done, with nothing big outside it (as in verification 1). The fixes since then are confined to `audit.go`, `AuditPage.tsx`, `filters.ts` and their tests.
- **Tests:** `GET /api/o/{oid}/audit` still has the five cases (happy path, 401, 403 ×3, another trust 404, invalid input 400 ×8). New cases cover the per-row cap, a patient search recorded against the patient, and a refused export logged as an export. Vitest covers `badDays`, `withoutBadDays` and `namesOf`. The e2e journey `test_bad_filters_say_so_and_never_leave_old_rows_on_screen` was added.
- **Tenant isolation:**
  - `MatchAuditPatients`, `AuditPatientNames` and `ListAudit` are scoped by `org_id`.
  - A patient search records only the trust's own patients.
  - Cara Clinician gets 403 for the list and the CSV (checked live).
- **A-20:** the grep finds only the pre-S11 comments in `apps/web/src/lib/format.ts`.
- **Security:**
  - no secrets;
  - route patterns only, never link tokens;
  - IDs only in rows;
  - `/audit` isn't on the patient router;
  - `chapter.ForPatient` is untouched.
- **The `lifebox-health(c)are` grep:** prints nothing.
- **Docs:** decisions.md, pending.md, the C1 plan's Status, plans/README and sessions/README are updated (by this verification, to Verified).

### Spot checks (Claude in Chrome, against `make dev`)

1. **Ada Admin** (http://tester.localhost:3000):
   - unfiltered page 390 KB, ≤10 patients per row;
   - a patient search is recorded against Jo Bloggs;
   - the refused export is logged as `audit.export` 400;
   - for TEST0001, the CSV has 350 rows and the JSON 351 (the extra row is the CSV export itself, recorded against the patient).
2. **Ada, screen:**
   - the bad-day link shows the named day, no table, and Export greyed out;
   - Clear, then Patient "Bloggs", gives the right rows, with the lookups under "Jo Bloggs (TEST0001)";
   - a 105-character search shows the error, no table, and Export greyed out.
3. **Cara Clinician** (http://localhost:3000):
   - `GET …/audit` → 403, and the CSV → 403;
   - her episode-list and episode views appear in Ada's log at Hospital A1, Done.

### Test results

- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web **88 passed**, clinical **116 passed**.
- `make test-integration`: every package ok (`internal/httpapi` 6.1 s).
- `make seed && make e2e` against `make dev`: **28 passed** in 86 s.

### Hand checks for Rahul (they don't block)

- **T-10:** click **Export CSV** once, and check that `audit-<date>-<time>.csv` lands in Downloads and opens in a spreadsheet.
- **R-1:** the patient journey in a real 390×844 window or device mode.

### Commits (made by this verification, local only, not pushed)

1. **The audit log: table, middleware and API (C1 / S11: data and API):** the migration, `audit.sql`, `episodes.sql`, the generated `internal/db`, and `httpapi/{audit.go,router.go,episodes.go,patient.go,audit_integration_test.go,routers_integration_test.go}`.
2. **The Audit screen and journeys (C1 / S11: web and journeys):** `apps/web/src/{routes.tsx,routes.test.ts,lib/auth.tsx,lib/auth.test.ts,pages/hospital/{Audit.tsx,HospitalLayout.tsx},features/audit/}`, `apps/web/AGENTS.md`, `e2e/journeys/test_audit.py`.
3. **Docs:** `CLAUDE.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S11-audit-log/*`.
   - `brief.md` holds a one-line 12 Oct planning change and is committed whole.
   - Other planning-only files are left uncommitted.

**Next:** `/build-session S13`.

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Needs fixes.** The audit log does what the brief and Rahul's answers ask, and it's well tested:
- one append-only row per patient-data request, IDs only;
- "no row, no data";
- a whole-trust-admin-only list with filters and a CSV export;
- every route has its five integration cases.

Every automated suite passes, and test report 1 passed 19/19. Two things need fixing before it's committed:
1. **The Audit screen's JSON already weighs 4 MB** with 337 patients. It will break at the size of a real trust.
2. **Searching the log for a patient isn't itself recorded.**

Both are small fixes.

### Findings, worst first

**Must fix**

1. **The Audit screen sends every listed patient's name for every list row, so the page grows with the trust's size.**
   - **Where:** `apps/api/internal/httpapi/audit.go:306` (`auditListRows`, called for the JSON page). "Listed patients" (`listPatients`, the New episode drawer) touches **every patient in the trust**, and "Listed episodes" touches every episode's patient. `auditListRows` turns each ID into `{id, name, hospitalNumber}`.
   - **Measured against `make dev` now:** with 337 patients and 1,069 log rows, `GET /api/o/{Trust A}/audit` (no filters) is **3.97 MB**, holding 27,306 patient objects (up to 332 on one row). The screen shows "332 patients" and at most ten names on hover.
   - **Why it matters:** the response passes Lambda's 6 MB limit at about 600 patients (target architecture, F3/F4). A real NHS trust has tens of thousands of patients, so the page would then fail with an error, and it's slow and heavy even locally.
   - **Also:** the IDs are de-duplicated with a linear scan (`containsID`, `audit.go:85`), called per row from `touched` (`episodes.go:62`, `:124`) and in `auditListRows` (`audit.go:328`). That's quadratic in the trust's size: about 10^10 comparisons for one page at 10k patients.
   - **Fix:**
     - The JSON row sends `patientCount` plus at most the first 10 patients (named). The tooltip already shows ten and "and N more".
     - The CSV keeps every name and ID.
     - Use a `map[uuid.UUID]struct{}` (or a set alongside the slice) for the de-duplication in `touched` and `auditListRows`.
     - **Test first:** a `TestListAudit` case where a list row touched more than 10 patients gives 10 patients and the right `patientCount`, and the CSV row still has them all.
2. **A patient search on the audit log isn't recorded against that patient.**
   - **Where:** `audit.go:283–292`. With `?patient=…`, the log returns that patient's name and hospital number, plus every access to their record. But the "Viewed / Exported the audit log" row has `patient_ids = '{}'`.
   - **Why it matters:** reading the log is a read of patient data (STF-06). Rahul's answer 1 asks for "the record IDs it touched". If an admin looks up a patient's history, a later search for that patient doesn't show it.
   - **Fix:**
     - When the `patient` filter is set, note the matched patient IDs on the `audit.view` / `audit.export` row: `auditOf(r).touched(id, uuid.Nil)` for each `MatchAuditPatients` result.
     - Unfiltered views stay as they are: their names are capped by finding 1, and logging every name shown would bring finding 1's size problem into the log.
     - **Test first:** in `TestListAudit`, after a patient-filtered view, a second view filtered by that patient lists the first view's `audit.view` row.

**Should fix in the same round (cheap, and on the screen this session built)**

3. **A slow response can overwrite a newer one, and an error leaves the old table on screen.**
   - **Where:** `apps/web/src/features/audit/AuditPage.tsx:53–66`.
   - **The effect:** the effect never ignores a stale response, so an unfiltered load that finishes after a filtered one shows unfiltered rows under the filtered form. A 400 (e.g. a patient search over 100 characters) shows the error above the previous filter's rows, which look like results.
   - **Fix:** add a `cancelled` flag in the effect's cleanup, and `setPage(null)` on error.
4. **An impossible day in the URL is dropped silently, while the form still shows it.**
   - **Where:** `apps/web/src/features/audit/filters.ts:45`, plus the form fill at `AuditPage.tsx:56`.
   - **The effect:** `?from=2026-02-30&to=2026-03-05` lists everything up to 5 March, while From still shows the bad day. The CSV uses the same query.
   - **Fix:** clear the bad day from the form, or show "Enter valid dates." The test report noted this case and found it handled, but the screen doesn't say the filter was ignored.

**Minor: log in [pending.md](../../pending.md) §3; no fix needed now**

5. **The CSV export has no size limit and is buffered twice** (`writeAuditCSV`, then `heldResponse`). The full export is already **5.57 MB** (1,009 lines). In the AWS phase it will pass Lambda's 6 MB response limit. **Recommendation:** export to S3 and return a pre-signed link, or stream it (S06/S17). pending.md mentions the in-memory buffering but not the Lambda limit.
6. **Export CSV is a plain `<a download>`** (`AuditPage.tsx:105`). A 400, 403 or 500 is saved as a file holding `{"error": …}`, and the page doesn't say anything.
7. **The audit insert runs after the handler's own write** (`audit.go:106`), in a separate statement. This is the documented "no row, no data" design.
   - If the insert fails after a POST has saved, the user sees a 500 for a change that happened, and a retry creates a duplicate patient, episode or note.
   - If a handler panics after saving, `recoverPanics` answers 500 and no row is written (`audit.go:101`). A deferred write with status 500 in `audited` would close this.
   - Both are rare. Writing the row in the handler's own transaction is the root-cause fix, to consider when handlers move to transactions.
8. **`heldResponse` hides the real writer** (`audit.go:134`). `http.MaxBytesReader` can't reach its `requestTooLarge` hook, so the connection isn't marked to close after an oversized body. Harmless today.
9. `ListAuditUsers`'s `DISTINCT` over the whole log is already in pending.md.

### Checked and fine

- **Scope:** everything in the brief is done:
  - `audit_log` with a trigger refusing UPDATE, DELETE and TRUNCATE, and foreign keys that don't cascade;
  - the middleware, writing IDs only;
  - `GET /api/o/{oid}/audit` with user, date and patient filters, and the CSV;
  - the Audit screen;
  - the integration tests.

  Nothing big outside the brief.
- **Every patient-data route is wrapped** in `audited` (`router.go:131–151`, plus the three patient-link routes). The routes left unwrapped carry no patient data: questionnaires, chapters, option lists, published HQs, staff, settings, admin and `/api/me`.
- **Tenant isolation:**
  - `ListAudit`, `MatchAuditPatients` and `AuditPatientNames` are all scoped by `org_id`.
  - The list is for whole-trust admins only (`auditAdmin`).
  - `loadEpisode` notes the episode only after the hospital check, so another hospital's patient ID never reaches the log.
- **Tests:** `TestListAudit` has all five cases (happy, no sign-in 401, wrong role 403 ×3, another trust 404 with none of A's rows or people in B's log, invalid input 400 ×8). Also tested:
  - rows written for every route (`TestAuditRowsWritten`);
  - fail-closed (`TestAuditFailsClosed`);
  - append-only (`TestAuditLogIsAppendOnly`);
  - the 500-row limit and the CSV, including formula defusing (`TestAuditLimitAndCSV`);
  - the patient router not serving `/audit`.

  Vitest covers the filters, `canSeeAudit`/`switchPath` and the route. There are three journeys in `e2e/journeys/test_audit.py`.
- **A-20:** the grep finds only comments in `apps/web/src/lib/format.ts` (from before S11). The CSV's zone is the browser's `tz` or the deployment's config.
- **Security:**
  - no secrets;
  - the route **pattern** is stored, never the path, so no link tokens are stored;
  - no patient values in rows or logs;
  - `/audit` isn't on the patient router;
  - `chapter.ForPatient` is untouched;
  - `GetEpisodeByToken`'s new org, hospital and patient IDs aren't sent to the patient (`patientHQ` builds its JSON by hand).
- **Docs:** decisions.md (13 Oct "S11"), pending.md, the C1 plan's Status and differences, plans/README and sessions/README are all updated. The `lifebox-health(c)are` grep prints nothing.

### Spot checks (Claude in Chrome, against `make dev`, http://tester.localhost:3000)

1. **Cara Clinician:**
   - `GET /api/o/{Trust A}/audit` → 403 with "Only the trust's admins can see the audit log.", and the CSV → 403.
   - Typing `…/h/A1/audit` shows the red "Only the trust's admins can see the audit log".
   - The header has no Audit link.
2. **Max Multi** (Trust B admin, clinician at A2):
   - Trust A's CSV → 403.
   - Trust B's log for `Bloggs` and for `TEST0001` → 0 rows.
   - Trust B's Who list is only Max; the rows are Hospital B1 or Whole trust.
3. **Ada Admin:** Audit filtered by patient `TEST0002` renders the right rows ("Listed episodes (all hospitals)", "Listed episodes", "Whole trust" / "Hospital A1" / "Hospital A2", Done). An unfiltered page is 3.97 MB and took 88 ms on localhost (finding 1).
4. **CSV via curl as Ada:** 200, 5.57 MB, 1,009 lines, header as specified (finding 5).

### Test results

- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web **85 passed**, clinical **116 passed**.
- `make test-integration`: every package ok (`internal/httpapi` 8.5 s).
- `make seed && make e2e` against `make dev`: **26 passed** in 81 s (audit ×3, J1–J5, J6 ×6, staff ×7, trusts ×5).

### Hand checks for Rahul (from test report 1; they don't block)

- **T-10:** click **Export CSV** once. Check that `audit-<date>-<time>.csv` lands in Downloads and opens in a spreadsheet.
- **R-1:** the patient journey in a real 390×844 window or device mode.

### Suggested commits (after the fixes are verified)

1. **The audit log: table, middleware and API (C1 / S11: data and API):**
   - `apps/api/db/migrations/00009_audit_log.sql`;
   - `apps/api/db/queries/{audit.sql,episodes.sql}`;
   - `apps/api/internal/db/*` (generated);
   - `apps/api/internal/httpapi/{audit.go,router.go,episodes.go,patient.go,audit_integration_test.go,routers_integration_test.go}`.
2. **The Audit screen and journeys (C1 / S11: web and journeys):**
   - `apps/web/src/{routes.tsx,routes.test.ts,lib/auth.tsx,lib/auth.test.ts,pages/hospital/{Audit.tsx,HospitalLayout.tsx},features/audit/}`;
   - `apps/web/AGENTS.md`;
   - `e2e/journeys/test_audit.py`.
3. **Docs: S11 brief, answers, build notes, test plan, test reports, verification and status:**
   - `CLAUDE.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S11-audit-log/*`.
   - `brief.md` holds a one-line 12 Oct planning change, committed whole.
   - Leave the other planning-only files uncommitted.

**For the builder:** "Verification found problems: read `docs/sessions/S11-audit-log/verify.md`, fix them test-first, then ask me for a retest."
