# S08 build notes: trusts, hospitals and permissions

Builder: automatic run (`/run-sessions`), 12 Oct 2026. Rahul's answers: [answers.md](answers.md) (time zone per hospital: yes; roles mapped as proposed with a reseed; the `dev` branch ignored entirely, and it wasn't read; a hospital-scoped user sees the trust's patient record but only their hospital's episodes).

## What was built

### Data (migration `00006_trusts.sql`)
- `orgs` (trusts): name, unique code, status (`active`/`suspended`), `settings jsonb`.
- `hospitals`: + `org_id`, status, `settings`, `timezone` (NULL = the deployment's `DEFAULT_TIME_ZONE`). `UNIQUE (org_id, id)` lets other tables check a hospital is in their trust.
- `users.cognito_sub` (unique, empty until S12); `platform_admins (user_id)`.
- `memberships (user_id, org_id, hospital_id NULL = whole trust, role)`, role ∈ `clinician`, `super_clinician`, `admin`, one per user per trust or hospital (`UNIQUE NULLS NOT DISTINCT`). A composite foreign key stops a hospital membership naming another trust's hospital.
- Questionnaires and option lists: + `org_id`, `hospital_id` now nullable (NULL = the trust's). Name uniqueness is per trust.
- Patients: `org_id` instead of `hospital_id` (the trust's record). Episodes: + `org_id`, with composite keys to the hospital and the patient of the same trust.
- **Existing rows are carried over:** each old hospital becomes the only hospital of a trust of its own, roles map as agreed (highest kept), and content becomes the trust's. Down migration included (best effort: trust-wide memberships are lost).

### API
- **Every staff route moved from `/api/h/{hid}/…` to `/api/o/{oid}/h/{hid}/…`.** `requireHospital` reads the caller's roles at that hospital of that trust (trust-wide and/or hospital membership) on every request, keeps the highest, and answers 404 to non-members, to a hospital that isn't in the trust, and to a malformed id.
- **New: `GET /api/o/{oid}/episodes`** ("All hospitals", STF-03), behind `requireOrg`: every hospital's episodes for a trust-wide member; only their own hospitals' for anyone else. Each row now carries `hospitalId` and `hospitalName` (on the hospital list too).
- **`GET /api/me`** returns `{user, platformAdmin, trusts: [{id, name, code, role, hospitals: [{id, name, timeZone, role}]}]}`.
- **The ladder** (`internal/role`): authoring, chapters, option lists, publish, new version → super clinician; deleting someone else's draft → admin; patients and episodes → any member (everyone is at least a clinician, A-5). Messages say "Only super clinicians and admins can …".
- Questionnaire and option-list queries show the trust's content plus that hospital's own; creating one makes it the trust's. Patients list the trust's records. Creating an episode needs a patient of the trust and a published HQ this hospital uses.
- `cmd/import-hq -hospital <id>` imports into that hospital's trust.

### Seed (`make seed`)
Trust A (Hospital A1, Hospital A2), Trust B (Hospital B1); Ada Admin, Alex Author, Cara Clinician, Val Clinician (was Val Viewer), Bea Author, Max Multi (two trusts), Sam Superadmin. On a database from before S08, the seed **adopts** the migrated trust holding the old Hospital A (now A1) as Trust A, so HJE Full HQ and other local data stay. Demo users' memberships are reset on every run.

### Web app
- URLs: `/o/:orgId/h/:hospitalId/…`; `/` is the **hospital picker** (one card per trust, each hospital with your role there). With one hospital, sign-in goes straight in.
- Header: **Poised · <trust> · hospital box** (grouped by trust, switches hospital and trust) · Questionnaires · Episodes · name · role badge · Sign out. Questionnaires and Episodes are shown to everyone.
- Episodes: **Hospital | All hospitals** switch for trust-wide members, kept in the URL as `?hospitals=all`, with a Hospital column; a row opens at its own hospital.
- Permissions come from `atLeast(role, …)`; the editor is read-only below super clinician.

**TypeScript for Rahul to review** (all small):
- `apps/web/src/lib/auth.tsx`: the `Me`/`Trust`/`Hospital` types from `/api/me`; `atLeast(role, need)` is the ladder (a rank lookup, like `role.AtLeast` in Go); `homePath`, `startPath` (one hospital → straight in, else `/`), `findHospital`; `HospitalContext`/`useHospital()` hand each page its trust, hospital, role and URL `base`.
- `apps/web/src/pages/hospital/HospitalLayout.tsx`: works out where the page is from the URL ids and `/api/me`, shows "You don't have access to this hospital" when it isn't one of yours, and renders the header.
- `apps/web/src/pages/Home.tsx`: the picker. `EpisodeList.tsx`: the All hospitals switch.
- Every feature component that took `hospitalId` now takes `base` (`/o/<trust>/h/<hospital>`), which starts both its page links and its API calls.

## Decided by the builder (automatic mode)
Recorded in [decisions.md](../../decisions.md) §2, 12 Oct "S08".
1. **Hospital routes nest under the trust** (`/api/o/{oid}/h/{hid}/…`) rather than `/api/o/{oid}/episodes/{eid}` with a hospital check per row. Smallest change, one middleware query.
2. **404, not 403, for non-members**, including just after a membership is removed (C1 says 403; NFR-04 says don't let trusts and hospitals be probed).
3. **All content is the trust's** (`hospital_id` NULL), created and existing. Hospital-owned content is supported by the queries (tested) but nothing creates it.
4. ~~A hospital-scoped super clinician can edit the trust's shared questionnaires.~~ *Superseded 12 Oct by Rahul's answer 5: whole-trust roles only (see Round 1 below).*
5. **`hospitals.timezone` NULL = the deployment's default**, returned (resolved) by `/api/me`; screens still use the browser's zone.
6. **"All hospitals" for a hospital-scoped user** returns only their hospitals' episodes (the UI doesn't offer the switch to them).
7. **Seed:** Val Viewer became Val Clinician at A2 (useful for hospital-isolation tests); Max Multi is clinician at A2 and Trust B's admin.
8. **Suspended status is stored, not enforced** (S10).
9. **The fake-querier route tests were deleted**; their cases already had API → database tests (testing rule 4). `TestPatientInputValidate` and `TestEpisodeChangeValidate` stay as unit tests.

## Differences from the plan
- C1's "removed membership → 403" is a 404 (decision 2).
- Patient records moved to the trust in S08 (Rahul's answer), ahead of C2/S13; `hospital_number` stays on the patient until S13.
- Added a journey beyond J1–J5: `e2e/journeys/test_trusts.py` (picker, switching, All hospitals, a URL leak).

## Tests
- **Unit:** `internal/role` (the ladder, parsing, highest of two roles); `TestPatientInputValidate`; web `src/lib/auth.test.ts` (ladder, start page, picker, finding a hospital only in its own trust); `src/routes.test.ts` (new URLs; old `/h/…` URLs are not found).
- **Integration (`trusts_integration_test.go`, new):** trust A's admin gets 404 for everything of trust B, including mixed trust/hospital ids; a hospital-scoped clinician gets 404 for the other hospital's URLs and for its episode at their own hospital's URL, sees only their episodes, and can open an episode for a patient first seen at the other hospital; hospital-owned content stays at its hospital; the role ladder (clinician can't publish, super clinician can; super clinician can't delete another's draft, admin can; trust-wide clinician + hospital super clinician publishes only there); a removed membership is refused on the next request; `GET /api/o/{oid}/episodes` with the five cases.
- **`TestMe`** rewritten for trusts (highest role per hospital, trust-wide role, effective time zone, platform admin, no membership, another trust, deleted user).
- **Every existing route test** moved to the trust model. Episode routes' "wrong role" case became "a clinician at the trust's other hospital → 404", since no role is below clinician.
- **End-to-end:** J1–J5 at the new URLs (J1 now goes through the picker); new `test_trusts.py`.

Results (12 Oct, 22:4x):
- `make lint`: go vet and tsc clean.
- `make test`: Go ok (role, config guard, …); Vitest web 56 passed, clinical 111 passed.
- `make test-integration`: all packages ok.
- `make e2e` against `make dev`: **7 passed** (J1–J5 + 2 trust journeys).
- Migration checked on a copy of the local database (up, down, up) and on an empty one; `make seed` run twice on each.

## Suggested commits (Rahul commits after verification)
1. **Trusts, hospitals, memberships and the role ladder (C1 / S08: data and API)**: `apps/api/db/migrations/00006_trusts.sql`, `apps/api/db/queries/*`, `apps/api/internal/db/*` (generated), `apps/api/internal/role/`, `apps/api/internal/httpapi/*` (handlers, middleware, router, tests, deleted fake tests), `apps/api/internal/apitest/*`, `apps/api/cmd/seed/main.go`, `apps/api/cmd/import-hq/main.go`.
2. **Hospital picker, trust header and All hospitals (S08: web and journeys)**: `apps/web/src/**`, `apps/web/AGENTS.md`, `e2e/**`.
   Commits 1 and 2 only work together (the API paths change in 1, the web app follows in 2); one commit for both is fine.
3. **Docs: S08 decisions, build notes, test plan and status**: `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S08-trusts-permissions/*`, `README.md`, `CLAUDE.md`, plus the 12 Oct planning-doc changes per decisions.md §2.

## Round 1 (test report 1: 25/25 passed) and Rahul's answer 5
- **Answer 5 (Rahul, 12 Oct): only whole-trust super clinicians and admins change the trust's questionnaires and option lists.** Built test-first:
  - `TestTrustContentNeedsAWholeTrustRole` (new; red first): every change (create, reorder, add chapter, rename, save content, new version, publish, option list create/update/delete, delete draft) is 403 for a super clinician *and* an admin of one hospital only, with nothing written; 401 signed out; 404 for another trust; the role is checked before the body is read; they can still view; they can change their hospital's own content; a whole-trust super clinician can do all of it; deleting another's draft needs a whole-trust admin.
  - `TestRoleLadder`'s mixed-role case now publishes only its hospital's own HQ. The usual cast's Alex and Bea are now whole-trust super clinicians.
  - Go: `place.TrustRole` (from `RolesAtHospital`, which now returns each membership's scope) and `canChange(r, owner, need)` in `middleware.go`; chapters, questionnaires and option lists use it with the content's `hospital_id` (`GetChapterMeta.owner_hospital_id`, new `GetOptionListOwner`).
  - Web: `canChange(here, ownerHospitalId, need)` in `src/lib/auth.tsx` (unit-tested), used by the list (New, Edit/View, Delete; rows now carry `hospitalId`) and the editor (read-only, Publish, Create new version).
  - Seed: **Sue Super**, super clinician of Hospital A1 only. e2e: `test_only_whole_trust_staff_change_the_trusts_questionnaires`.
- **Report item 1 (switching hospital lands on Questionnaires):** fixed; switching keeps Episodes or Questionnaires (e2e: Max switches from B1 Episodes to A2 Episodes).
- **Report items 2–3 (cosmetic):** the **View** item has an eye icon; the delete dialog keeps the name while fading out.
- Results: `make lint`, `make test` (web 58, clinical 111), `make test-integration` all pass; `make e2e` **8 passed**.

## Round 2 (test report 2: 20 passed, 1 failed)
- **F-1 (Preview let a view-only user try to save test cases):** fixed test-first.
  - API: `GET …/chapters/{cid}` now returns `ownerHospitalId` (null for the trust's), so the browser can tell whose the questionnaire is. `TestGetChapter` checks it (red first).
  - Web: the Preview works out `readOnly` with `canChange` (and draft-only), and the Test cases panel then hides the name box, **Save answers** and the delete icons, and says why ("Read only: only super clinicians and admins of the whole trust can change test cases.", or "…only in a draft version."). Loading and running test cases still work. The header's **Edit** reads **Open in editor** for them.
  - e2e: Sue's journey now opens the Preview and checks it (red first).
- **Found while exploring, fixed (small, safe):**
  - **The staff header below about 768px:** it now scrolls sideways inside itself, the hospital box is narrower on small screens, and the user's name hides below `md`. e2e `test_the_staff_header_fits_a_narrow_window` (390px; red first). The Episodes table is still wider than a phone; logged in [pending.md](../../pending.md) §3.
  - **The Publish dialog flashed "ready" on Cancel:** it keeps the last problems while it fades out (`PublishModal.tsx`).
  - **The Preview crashed on outputs without `codes`** (the e2e fixture's notes): `OutputsPanel` treats missing codes as none and a missing note category as "", which also removed a React key warning. Checked by hand in a script: choosing Yes on the fixture shows "Smoker" with no crash.
- Results: `make lint`, `make test` (web 58, clinical 111), `make test-integration` all pass; `make e2e` **9 passed**.

## Round 3 (test report 3: 13 passed, 1 failed)
- **F-1 (Save answers did nothing on outputs without `codes` or a note category):** fixed at the root, test-first.
  - `packages/clinical`: a new `normalizeOutput` (in `properties.ts`) gives every output a `codes` list and every note a `text` and `category` ("" if missing). Every reader goes through it: `outputsOf` (options and questions), `computeOutputs`' score bands and grid cells, and `validateOutput`. So the Outputs panel, test cases (`outputLines`, Run all), the publish check's test-case run and the POA Summary all get whole outputs.
  - `outputLines` writes a note without a category as `Note: …` (with one: `Note (Lifestyle): …`, unchanged).
  - Web: the editor's Disclosures dialog reads outputs through `normalizeOutput` too (`outputsAt`), so `OutputSummary` and `OutputEditor` never see a missing list. The ad-hoc guards added in round 2 to the Preview were replaced by this.
  - Tests: `packages/clinical/src/partialOutputs.test.ts` (5 tests, all red first) on outputs shaped like the e2e fixture (`{id, note: {text}}`, and `{id}`), on options, bands and grid cells: normalising, computing, test-case lines, Run all and the publish check, and validation. e2e: `test_a_whole_trust_super_clinician_saves_a_test_case_on_a_note_only_output` (save and Run all on the fixture).
- **Cosmetic (editor's "Delete this?" dialog lost its body while fading out):** it keeps the last question shown while it closes.
- Results: `make lint`, `make test` (web 58, clinical 116), `make test-integration` all pass; `make e2e` **10 passed**.

## Round 4 (test report 4: 13 passed, 0 failed)
- Everything passed. No code was changed after report 3's fixes (no test rounds left).
- The report's three "found while exploring" items (the Disclosures dialog's fade-out text, squeezed test-case badges, Enter not saving a test case) are logged in [pending.md](../../pending.md) §3.
- Status: **Tested**, awaiting `/verify-session S08`.

## Hand checks for Rahul
- None yet.
