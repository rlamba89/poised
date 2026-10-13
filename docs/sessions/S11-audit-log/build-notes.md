# S11 build notes: the audit log

Builder: automatic run (`/run-sessions`), 13 Oct 2026. Answers: [answers.md](answers.md): **one row per request** that reads or changes patient data, list screens included (who, when, trust, hospital, route and the record IDs it touched); **keep everything, delete nothing for now**.

## What was built

### Data (migration `00009_audit_log.sql`; up, down, up checked on the local database)
- **`audit_log`**: `id` (bigint identity), `at`, `org_id` (not null), `hospital_id` (NULL = a trust route), `actor` (`staff` or `patient`), `user_id` (the staff member; NULL exactly when the actor is the patient, by a CHECK), `action` (e.g. `episode.view`), `route` (the route **pattern**, e.g. `GET /api/o/{oid}/h/{hid}/episodes/{eid}`: never the path, because the patient's path holds their link's token), `status` (the HTTP status answered), `patient_ids uuid[]`, `episode_ids uuid[]`. **No patient values**: names are looked up when the log is read.
- Foreign keys to `orgs`, `hospitals` and `users` **don't cascade**, so nothing can delete a row by deleting its trust or person.
- **Append-only:** a trigger refuses `UPDATE` and `DELETE` (per row) and `TRUNCATE` (per statement) with "audit_log is append-only: … is not allowed".
- Indexes: `(org_id, at DESC)` for the list; GIN on `patient_ids` for the patient filter.

### Go (`internal/httpapi/audit.go`, `db/queries/audit.sql`)
- **`audited(action, handler)`** wraps every patient-data route (`router.go`). It runs inside `requireHospital`/`requireOrg`, so only requests that passed the sign-in and membership checks are written; the trust and hospital come from there. Handlers note what they touched with `auditOf(r).touched(patient, episode)`: `loadEpisode` (every per-episode route), `loadByToken` (the patient's link, which also gives the trust and hospital), and the list and create handlers.
- **No row, no data:** the handler's response is held (`heldResponse`) until its row is written; if the insert fails, the caller gets a 500 and nothing else. A change the handler already saved stays saved (the handlers don't run in one transaction with the audit insert).
- **Which routes** (16 actions, labels in `auditActions`): Listed patients, Added a patient, Listed episodes, Listed episodes (all hospitals), Created an episode, Opened an episode, Changed an episode, Added a note, Opened the answers (`…/hq`, used by the episode, validate and POA Summary pages), Saved the clinician's answers, Completed the HQ review; the patient's link: opened their questionnaire, saved answers, sent their answers; and **Viewed / Exported the audit log**.
- **What's written for refusals:** a request refused by the sign-in or membership check (401, 404 for non-members and other trusts) writes nothing; a refusal after it (an episode id from another hospital → 404, a bad ASA grade → 400, a stale save → 409) is written with its status. The patient's link writes a row once its token names an episode (a suspended trust's 403 included); an unknown token writes nothing.
- **`GET /api/o/{oid}/audit`** (whole-trust admins only, like Staff and Settings; anyone else in the trust 403, others 404):
  - filters: `user` (a staff id, or `patient` for the patient's link), `from`/`to` (RFC 3339 times, from inclusive, to not), `patient` (part of the name in either order, the hospital number, or the record's id; matched among the trust's own patients only, up to 100 characters);
  - JSON: `{rows: [{id, at, actor, userId, userName, userEmail, hospitalId, hospitalName, action, actionLabel, route, status, patients: [{id, name, hospitalNumber}], episodeIds}], more, limit: 500, users: [{id, name, email}]}`, newest first; `users` is everyone in this trust's log (removed staff included, STF-04);
  - **`?format=csv`**: every matching row (no limit), `Content-Disposition: attachment; filename="audit-<date>-<time>.csv"`; columns Time, Who, Email, Action, Hospital, Patients, Patient record IDs, Episode IDs, Route, Result. Times are RFC 3339 in `tz` (the browser's IANA zone) or the deployment's `DEFAULT_TIME_ZONE`. Cells starting `= + - @`, tab or CR get a leading `'` (CSV injection: names come from people).
  - Invalid input → 400: a `user` that isn't an id, an unreadable time, `from` not before `to`, a `format` other than json/csv, an unknown `tz` (or `Local`), a patient search over 100 characters.
- The audit row's time is `Deps.Clock` (tests move it), not the database's `now()`. The row is written even if the caller has disconnected (`context.WithoutCancel`), since a change may already be saved; there's no automated test for a mid-request disconnect.
- `GetEpisodeByToken` now also returns the episode's trust, hospital and patient ids (for the log only; the patient's JSON is built by hand and doesn't include them). `ListEpisodes` returns `patientId`.

### Web app
- **`/o/:orgId/h/:hospitalId/audit`** (`pages/hospital/Audit.tsx` → `features/audit/AuditPage.tsx`), linked as **Audit** in the header for whole-trust admins (`canSeeAudit` in `lib/auth.tsx`, = `canManageStaff`). Switching hospital keeps Audit where the user manages the trust (`switchPath`).

**TypeScript for Rahul to review:**
- `src/features/audit/filters.ts`: the filters live in the page's URL as days and text (`?user=&from=YYYY-MM-DD&to=YYYY-MM-DD&patient=`), so reload, back and deep links keep them. `auditQuery` turns them into the API's query: **the days become the browser's own midnights** (from = start of `from`, to = start of the day after `to`), sent as exact times, so no time zone is named in the code (A-20); an impossible day (30 Feb) is dropped. The export adds `format=csv` and the browser's zone (`Intl…resolvedOptions().timeZone`), so the CSV's times match the screen.
- `src/features/audit/AuditPage.tsx`: a form (Who, From, To, Patient, **Show**, **Clear**) that writes the filters to the URL; the list loads whenever the URL's filters change (or Show is pressed again). It only fetches on Show, not on each keystroke, because **every load is itself a row**. `ownKey` remembers the filters the page wrote itself, so only back/forward/deep links refill the form (otherwise a slow navigation could undo what's being typed: found by the journey). **Export CSV** is a plain download link to the API with the applied filters. A row with several patients shows "N patients" with the first ten names on hover.

## Decided by the builder (automatic mode)
1. **Refusals:** only requests that passed the sign-in and membership checks are logged (they belong to that trust), each with its HTTP status, so a probe for another hospital's episode shows as "Refused (404)". Requests refused before that write nothing (there's no trust to put them in that the caller may see), and neither does an unknown patient link.
2. **The patient's own link is logged**, with "Patient (their link)" as the actor: it reads and changes patient data (STF-06), and a trust needs to see when answers came in.
3. **No row, no data** (fail closed): if the row can't be written the request answers 500 and sends nothing. For changes, the change may already be saved; it's logged as a server error.
4. **Reading and exporting the log are logged too** (AUD-01: exports are recorded), without patient IDs. They're trust routes (Hospital "Whole trust").
5. **Who reads it:** whole-trust admins only, as for Staff and Settings (S09/S10). A hospital-only admin gets 403. The platform admin has no access (OPS-06).
6. **List screens record every patient and episode they showed** (Rahul's "the record IDs it touched"), so filtering by a patient also finds the lists that showed them.
7. **The screen shows the newest 500 rows**; the CSV has them all. **The CSV is built in memory** before it's sent (needed for "no row, no data"); fine at today's sizes, see pending.md.
8. **The time filters are days in the browser's zone**, converted by the browser; the CSV's times use the browser's zone too (`tz`), falling back to the deployment's.
9. **Retention:** nothing is deleted (Rahul's answer 2). Deleting rows past the trust's retention period will need a deliberate, recorded change to the append-only trigger.

## Differences from the plan
- None in scope. Not built: the questionnaire change history (AUD-01/AUD-02, authoring); IP address or browser in rows; paging past 500 rows on screen.
- **In `make dev`, most views are logged twice:** React's StrictMode runs each page's first data load twice in development. The production build loads once. The test plan says so.

## Tests
- **Integration (written first, failing: `internal/httpapi/audit_integration_test.go`):**
  - `TestAuditRowsWritten`: one row per request for every patient-data route, with the right trust, hospital (NULL for All hospitals), actor/user, route pattern, status and record IDs (lists record every patient listed; creates record the new ids); the patient's link with the patient as actor and no token in the route; a refusal after the membership check written with its status (404, 400); nothing for no sign-in, another hospital's clinician, another trust, another trust's admin, an unknown link; nothing for `/api/me`, questionnaires, published HQs, staff, settings.
  - `TestAuditFailsClosed`: with inserts refused (a test trigger), an episode, the patient list and the patient's link each answer 500 and send no patient data.
  - `TestAuditLogIsAppendOnly`: UPDATE, DELETE and TRUNCATE all fail with the append-only message; the row is still there.
  - `TestListAudit`: happy path (newest first, names read now, users list); reading the log is logged; filter by user and by the patient's link; by time (from inclusive, to not, using the fake clock 48 h back); by patient name (either order, any case), hospital number, record id; trust B's patient not found from A; **no sign-in** 401; **wrong role** (clinician, whole-trust super clinician, hospital-only admin) 403 for list and CSV, writing no row; **another trust** (B's admin 404 for A; B's log has none of A's rows or people); **invalid input** (8 cases, 400) and a bad trust id (404).
  - `TestAuditLimitAndCSV`: 510 rows → the list stops at 500 with `more`; the CSV has all of them plus the earlier list view, header, labels, ids, status, times in the requested zone, `'=HYPERLINK…` defused; the export writes `audit.export`; the CSV takes the filters; no `tz` uses the deployment's.
  - `routers_integration_test.go`: the patient router doesn't serve `/audit`; the staff router does.
- **Unit (Vitest):** `features/audit/filters.test.ts` (5: reading the URL, days → midnights, empty filters, CSV + zone, bad days dropped); `auth.test.ts` (who sees Audit; switching hospital keeps Audit); `routes.test.ts` (`…/audit` is the audit page inside the hospital layout).
- **End-to-end (`e2e/journeys/test_audit.py`, 3 journeys):** Cara opens a patient's episode and changes an answer in her own browser; Ada filters by Cara and the patient (URL keeps them across a reload), sees the actions with the patient's name, filters by the patient's link and by days (today / tomorrow), goes back, and the exported CSV has the same rows as the list; a clinician has no Audit link and is refused at the URL; Trust B's admin finds nothing of Trust A's patient and doesn't list Cara.

Results (13 Oct, 09:1x):
- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web 85 passed, clinical 116 passed.
- `make test-integration`: all packages ok.
- `make e2e` against `make dev` (after `make seed`): **26 passed** (J1–J6, trusts, staff, audit).

## Round 1 (test report 1: 19 passed, 0 failed)

No code changed. T-12 step 3's wording was corrected in the test plan: switching to a hospital where the user doesn't manage the trust goes to Episodes, as intended (`switchPath`); the refusal is checked at the typed `/audit` address. Two cosmetic items (the CSV's Patients column lacks the hospital number; the narrow Who dropdown) are logged in [pending.md](../../pending.md) §3.

## Verification 1 (Needs fixes): fixes, each test-first

1. **The Audit screen's JSON grew with the trust** (3.97 MB at 337 patients: every list row named every patient). Each row now gives the screen `patientCount` and `episodeCount` plus at most the **first ten** patients (named) and episode ids (`auditRowPatients`); the CSV still names them all. The de-duplication of IDs in `touched` and `auditListRows` is now a set (`idSet`, a map beside the slice), not a linear scan. Measured on the local database: **0.40 MB** for 500 rows, bounded per row whatever the trust's size. Test: `TestAuditListCapsPatientsPerRow` (41 patients and 30 episodes listed → 10 of each with the counts; the page stays under 12 kB for two rows; the CSV has all 41 names and ids and all 30 episode ids), failing first. Web: `namesOf` (Vitest) shows "A, B and 330 more" from the count.
2. **A patient search of the log wasn't recorded against the patient.** With the `patient` filter, the "Viewed / Exported the audit log" row now records the matched patients' IDs, so a later search for that patient shows the lookups. Unfiltered views still name nobody. Test: `TestListAudit`, "searching the log for a patient is a read of their data…" (view, export, and a no-match search; then Ada's lookups appear under the patient), failing first.
3. **A slow response could overwrite a newer one, and an error left the old rows on screen.** The list's effect ignores a response that's no longer current, and an error clears the table. E2E: `test_bad_filters_say_so_and_never_leave_old_rows_on_screen` (a 101-character search shows the message and no table).
4. **An impossible day in the URL was dropped silently.** `badDays` (Vitest) checks From and To; the page then says "Enter valid dates.", shows no table and greys out Export CSV, and Show refuses one too. E2E: the same journey opens `?from=2026-02-30&to=2026-03-05`.

Findings 5–8 (the CSV's size against Lambda's 6 MB limit, the plain download link, the audit insert after the handler's write, `heldResponse` hiding the writer) are logged in [pending.md](../../pending.md) §3. The test plan has new cases T-15 to T-18.

Results (13 Oct, after the fixes): `make lint` clean; `make test` Go ok, Vitest web 87 and clinical 116 passed; `make test-integration` all ok; `make seed && make e2e` **28 passed** (audit ×5).

## Round 3 (test report 2: all 8 re-run cases passed): its "found while exploring" items 1–4, each test-first

1. **A refused export was logged as "Viewed the audit log".** The handler now takes the action from `format=csv` before checking the filters, so a refused export is logged as "Exported the audit log" (status 400). Test: `TestListAudit`, "a refused export is logged as an export, a refused view as a view" (failed first, twice). **Export CSV is greyed out (no link) whenever the applied filters were refused** (a too-long search, a bad user id, an impossible day), not only for impossible days.
2. **After a refused search, Who showed "Someone else".** The Who list is now kept from the last list that loaded (`users` state), so a refused load doesn't empty it.
3. **An impossible day in a link was invisible in the form**, so Show kept failing until Clear. The form now starts with that day emptied (`withoutBadDays`, Vitest), and the message names it: "Enter valid dates. There's no 2026-02-30." Show then works with the other filters.
4. **No loading sign:** a small spinner shows until the first list loads (trivial, so done rather than logged).

E2E: `test_bad_filters_say_so_and_never_leave_old_rows_on_screen` now also checks Who keeps "Ada Admin (…)", Export CSV has no link after the refused search and the bad-day link, the bad day is named and its box emptied, and Show works afterwards without Clear. Test plan: new T-19 to T-21; T-18's message updated.

Results (13 Oct, after round 3): `make lint` clean; `make test` Go ok, Vitest web 88 and clinical 116 passed; `make test-integration` all ok; `make seed && make e2e` **28 passed**.

## Round 4 (test report 3: all 6 re-run cases passed)

No code changed. Its two cosmetic items (no spinner while a changed filter loads; Who lists no people when the page opens straight onto a refused search) are logged in [pending.md](../../pending.md) §3.

## Hand checks for Rahul
- **T-10:** click **Export CSV** once; check `audit-<date>-<time>.csv` lands in Downloads and opens in a spreadsheet (the tester fetched the link's content instead of saving a file; the e2e journey does download it).
- **R-1:** the patient journey in a real 390×844 window or device mode (the tester's window couldn't be resized, so it ran in a 390×844 iframe; the e2e journeys run it on an emulated phone).

## Suggested commits (Rahul commits after verification)
1. **The audit log: table, middleware and API (C1 / S11: data and API)**: `apps/api/db/migrations/00009_audit_log.sql`, `apps/api/db/queries/{audit.sql,episodes.sql}`, `apps/api/internal/db/*` (generated), `apps/api/internal/httpapi/{audit.go,router.go,episodes.go,patient.go,audit_integration_test.go,routers_integration_test.go}`.
2. **The Audit screen and journeys (S11: web and journeys)**: `apps/web/src/{routes.tsx,routes.test.ts,lib/auth.tsx,lib/auth.test.ts,pages/hospital/{Audit.tsx,HospitalLayout.tsx},features/audit/}`, `apps/web/AGENTS.md`, `e2e/journeys/test_audit.py`.
3. **Docs: S11 decisions, build notes, test plan and status**: `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S11-audit-log/*`, `CLAUDE.md`, plus S11's share of the 12 Oct planning-doc changes (its `brief.md`, `answers.md`).
