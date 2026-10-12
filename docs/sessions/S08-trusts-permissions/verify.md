# S08 verification

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Verified.**

The verifier reviewed the working tree on top of `603d3d5`, covering S08's code and S08's own docs. The 12 Oct planning-doc changes that predate S08 weren't reviewed.

### Scope
Everything in the brief is built:
- **Data:** migration `00006_trusts.sql` adds:
  - `orgs`;
  - on hospitals: `org_id`, status, settings and a time zone;
  - `users.cognito_sub`;
  - memberships with an optional hospital;
  - `platform_admins`;
  - content owned by the trust.
- **Routes:** every staff route is now under `/api/o/{oid}/…`, and membership is re-read on every request. There's also `GET /api/me` and "All hospitals".
- **Web app:** `/o/:oid/h/:hid/…`, with the picker, the trust and hospital header, and the switch.
- **Seed:** every user the brief asked for, plus Sue Super (for answer 5).
- **Journeys:** J1–J5 at the new URLs, plus `test_trusts.py`.

Rahul's answers 1–5 are all followed, including answer 5 (only whole-trust staff may change the trust's content).

Nothing big was added beyond the brief. The extra fixes are small and recorded:
- the header at narrow widths;
- the dialog fade-outs;
- `normalizeOutput`.

### Findings (all minor; none blocks)
These come from `/code-review high`, weighed by the verifier. All are logged in [pending.md](../../pending.md) §3.

1. **`GET /api/me` leaves out a trust with no hospitals** (`apps/api/db/queries/users.sql:19`, an inner join to `hospitals`).
   - It can't happen today: every seeded trust has hospitals, and nothing creates trusts yet.
   - It matters for S10: a trust admin invited before the first hospital exists would see the "ask an admin" page.
2. **Hospital-owned content: three things the UI and API don't line up on.** None of these can happen today, because nothing creates hospital-owned content (decision 3).
   - `apps/web/src/features/editor/OptionLists.tsx:118` offers "Save as a list" and delete without `canChange`. A hospital-only super clinician editing their hospital's own HQ would get a 403.
   - `DraftNameExists` (`apps/api/db/queries/questionnaires.sql:34`) checks the whole trust, so it can refuse a name that the caller can't see in their list.
   - `contentRole` (`apps/api/internal/httpapi/questionnaires.go:70`) always says "of the whole trust", even for hospital-owned content.
3. **`createVersion` checks for a draft with the same name without taking `LockOrgQuestionnaires`** (`questionnaires.go:242`).
   - This is older than S08: it never took the hospital lock either.
   - S08 widens the check to the whole trust, so two people at different hospitals could now race on it.
4. **`memberships` has no primary key** after `00006_trusts.sql:49`: it was dropped and replaced only by `UNIQUE NULLS NOT DISTINCT`.
   - That's fine for Postgres.
   - Logical replication (Neon → RDS in C10) needs a replica identity. Add a surrogate `id` before then.
5. **Migration 00006's Down step fails if a trust with no hospitals owns a questionnaire** (`00006_trusts.sql:153`).
   - It can't happen today.
   - The build notes already call the Down step "best effort".
6. **Cleanups:**
   - `Home.tsx:16` repeats `startPath`.
   - Signing in fetches `/api/me` two or three times.
   - Each reader calls `normalizeOutput` itself, rather than once when content loads (a deliberate choice, recorded in decisions.md).

### Checked by hand
- **Tenant isolation:**
  - Every query that names a hospital also checks the trust.
  - `requireHospital` joins the hospital to the trust in `RolesAtHospital`.
  - `GetEpisode` is scoped by the hospital, which the middleware has already checked belongs to the trust.
  - `CreateEpisode` checks that the patient and the HQ belong to the same trust.
  - Composite foreign keys stop rows from crossing trusts.
- **A-20 grep:** it finds only comments in `apps/web/src/lib/format.ts`, which are older than S08. The "lifebox-health(c)are" grep finds nothing.
- **Security:**
  - The diff adds no secrets and no new log lines.
  - `TestPatientRouter` still covers the patient router never serving staff routes.
  - `chapter.ForPatient` is unchanged.

### Test results
- `make lint`: go vet and tsc are clean.
- `make test`:
  - Go: all packages ok.
  - Vitest web: **58 passed**.
  - Vitest clinical: **116 passed**.
- `make test-integration`: all packages ok (httpapi 8.6 s).
- `make e2e` against the conductor's `make dev`: **10 passed** in 33 s (J1–J5 and the 5 trust journeys).

### Spot checks (API with curl, and the browser through Claude in Chrome)
1. **Trust A's admin (Ada) and Trust B URLs:**
   - 404 for Trust B's episodes, its "All hospitals", its questionnaires and its patients.
   - 404 for the mixed URL `/o/TA/h/B1`.
   - 200 for her own Hospital A2.
2. **Hospital-scoped clinician (Cara, A1):**
   - 404 for A2's episodes.
   - 404 for an A2 episode through the A1 URL, through the A2 URL, through `/hq`, and through PATCH.
   - Her "All hospitals" list holds only Hospital A1 (Ada's holds A1 and A2).
   - In the browser:
     - the A2 episode through the A1 URL shows "Episode not found.";
     - the A2 URL shows "You don't have access to this hospital".
3. **The role ladder and answer 5:**
   - Sue Super (super clinician of A1 only) gets 403 with the plain message for creating a questionnaire, publishing, adding a chapter and creating an option list. She can still view (200), and the editor shows **READ ONLY** with no Publish button.
   - Cara gets 403 for publishing.
   - Ada gets 400 for an empty chapter name, so the role check passes and the input is then validated.
   - Signed out: 401.
4. **The two-trust user (Max):**
   - The picker shows Trust A with Hospital A2 as Clinician, and Trust B as "Admin, whole trust".
   - Trust B shows none of Trust A's questionnaires.
   - The header's switch from B1 Episodes to A2 Episodes stays on Episodes, and he gets no "All hospitals" switch in Trust A.
   - No console errors.

### Hand checks for Rahul
None.

### Suggested commits
1. **Trusts, hospitals, memberships and the role ladder, with the hospital picker (C1 / S08):** the API and the web app in one commit, because the API paths and the web app's URLs change together.
   - **API:** `apps/api/**` (migration, queries, generated `internal/db`, `internal/role`, httpapi handlers, middleware, router and tests, the deleted fake-querier tests, apitest, seed, import-lifebox).
   - **Clinical logic:** `packages/clinical/**` (`normalizeOutput` and its tests).
   - **Web app:** `apps/web/**`.
   - **Journeys:** `e2e/**`.
2. **Docs: S08 build notes, test plan, test reports, verification and status:**
   - S08's own folder: `docs/sessions/S08-trusts-permissions/*` (brief, answers, build notes, test plan, reports 1–4, verify).
   - `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md` and `docs/sessions/README.md`.
   - `README.md` and `CLAUDE.md`.
   - Some of these files mix S08's changes with the 12 Oct planning changes, and are committed whole.
