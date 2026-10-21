# S19 verification

## Verification 1 (21 Oct 2026, verifier, automatic run)

**Verdict: Verified.**

What I checked:
- the whole working tree against the brief, the answers and plan C3;
- `/code-review high`, with each finding weighed below;
- every suite, including `make e2e` on my own API (:8092) and Vite (:3012);
- the riskiest routes over real HTTP, and three screens in Chrome (trk profile) against the conductor's `make dev`.

No finding blocks the session. They're all minor, and the ones worth keeping are in pending.md ("From the S19 verification").

### Scope

Everything in the brief, as answered, is built:
- **`procedures`** (migration 00020): org_id, `hospital_id` (a composite FK to the trust's own hospitals; NULL = every hospital, answers 2), name unique per trust in any case, code, specialty, default HQ, `info_content` and `proms_plan` (answers 3: the columns now, as empty JSON lists), retired (answers 5).
- **Routes** `GET/POST /api/o/{oid}/procedures` and `PATCH …/{prid}`: whole-trust super clinicians and admins change it (answers 1). POST and PATCH are wrapped in `audited`.
- **Episodes:** `procedure_id` replaces the free text (answers 4a, carried across by the migration), plus `planned_date` and `poa_date`. Both can be edited until the HQ review is complete (then 409). A retired procedure stays on its episodes.
- **The default HQ** (answers 6): the procedure's own, then the hospital's `defaultHq`, resolved in Go (`?hospital=`).
- **Shown** in the episode list, the episode page, the POA Summary and the patient's home. **Seed:** a demo HQ and procedures per trust.

The deviations are recorded in the plan's Status and in decisions.md:
- the default HQ is stored as a questionnaire, not a version;
- specialty is optional;
- INF-04 and PRM-01 are columns only, with nothing to write or show.

All three are reasonable.

### Findings (worst first, none blocking)

1. **The Procedures page's Default HQ list and the API's check use different sets** (`apps/web/src/features/procedures/ProceduresPage.tsx:54-55` vs `apps/api/internal/httpapi/procedures.go:212`).
   - The page lists `…/h/{hid}/published-hqs`, which includes the current hospital's own questionnaires.
   - `checkProcedure` accepts only the trust's (`PublishedQuestionnairesInOrg`, `hospital_id IS NULL`).
   - **Not reachable today:** questionnaires can only be created for the whole trust (`createQuestionnaire`), and the dev database has no hospital-only published questionnaire. Once hospital-only HQs exist, the page would offer a choice the API refuses. The fix is either to list the trust-wide set (as Settings' default HQ does), or to let a hospital-only procedure take its own hospital's HQ.
   - Logged in pending.md.
2. **New episode: picking a procedure before the HQ list has loaded loses its default HQ** (`apps/web/src/features/episodes/NewEpisodePage.tsx:57-60`).
   - `hqForProcedure` checks against `hqs`, which is still `[]` until `published-hqs` returns, and the hospital's default HQ then fills in.
   - Both requests start together and the procedure list appears only when its own response arrives, so in practice the window is a few milliseconds. The clinician also sees the HQ before Create.
   - Logged in pending.md.
3. **`episodes.procedure_id` and `procedures.default_questionnaire_id` are single-column FKs** (`apps/api/db/migrations/00020_procedures.sql:23,36`). The repo's tenancy pattern is a composite `(org_id, …)` FK, as for `episodes_patient_fkey`, `episodes_hospital_fkey` and `procedures_hospital_fkey`.
   - Every write path checks the trust in Go (`OfferedProcedureName`, `checkProcedure`). I checked over HTTP: Trust B's create with a Trust A procedure gets 400.
   - So the database alone doesn't stop a cross-trust link from a future code path. It's defence in depth: a cheap follow-up migration (a unique `(org_id, id)` on procedures, then the composite FK). Logged in pending.md.
4. **Pushing S19 has the same deploy-then-migrate window on QA as S18** (verify.md S18, finding 1).
   - 00020 drops `episodes.procedure`, and the new Lambdas read `procedure_id`.
   - So episode routes return 500 on QA between the Lambda deploy and the migrate step. QA has only made-up data, so this is acceptable. Know it before the push.
5. **General notes show the dates as ISO** ("Planned procedure date set to 2026-11-03"), while the details show 03/11/2026. This is a recorded builder's choice (decisions.md, S19): notes are stored text. It's locale-neutral and fine for now.
6. **Small code-review notes, not acted on:**
   - `updateProcedure` reads the request body after taking the row lock (`procedures.go:278`), and checks the role twice.
   - `hospitalHQs` repeats the effective-settings decode found in `invite.go` and `jobs.go`.
   - `ListPatientEpisodesHere` selects `planned_date`, which nothing shows yet.
   - `EpisodePage.tsx`/`PoaSummary.tsx` wrap `formatOptionalDate` in a redundant ternary.

### Checks from the skill

- **Tests first / five cases:** `TestListProcedures`, `TestCreateProcedure` and `TestUpdateProcedure` each cover the happy path, no sign-in, the wrong role (a clinician, and a super clinician of one hospital), another trust and invalid input. The episode create/update/get/list tests cover the procedure and the dates. The migration has its own test (Up, the carry-across, Down, Up). The journeys are `test_procedures.py`, plus J3 and J7c.
- **Tenant isolation:** every procedure query has `org_id`. `ListProcedures` limits a hospital-only member to the trust's procedures and their own hospitals'. The offered list and check also limit by hospital. Over HTTP:
  - Bea (Trust B) gets 404 on Trust A's list and on a PATCH, and 404 on a Trust A procedure by Trust B's path;
  - Val (A2) gets 404 on `?hospital=A1`, and her list has no Cataract surgery;
  - Sue and Cara get 403 on a POST or PATCH;
  - no sign-in gets 401;
  - `/api/p/…` gets 404 for the staff route.
- **Nothing hardcoded for the UK:** the grep shows only comments (`format.ts` examples, `patient.go` docs).
- **Security:**
  - no secrets;
  - the audit rows carry no patient data;
  - the General notes hold only the procedure's name and the dates;
  - the patient home gets only the procedure's name.
- **Docs:** the plan's Status, plans/README, sessions/README, decisions.md, CLAUDE.md and AGENTS.md are all updated. The `lifebox-health(c)are` grep prints nothing.

### Test results (21 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 177 passed (18 files) + 116 passed (15 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 12.1 s, seed, apitest with the migration test).
- `make e2e` (own API :8092, Vite :3012, `E2E_BASE_URL=http://localhost:3012`): **48 passed, 5 skipped, 0 failed** (2 min 48 s).

### Spot checks (Chrome, trk profile, http://localhost:3000)

1. **Val Clinician, New episode at Hospital A2:** the Procedure list has no Cataract surgery (A1 only). Picking Total hip replacement (HIP-01) set Health questionnaire to "Demo pre-op HQ v1".
2. **Sue Super, deep link to `…/procedures`:** "Only super clinicians and admins of the whole trust can manage its procedures", with no list.
3. **A retired procedure on an existing episode.** Over HTTP, I made procedure "VERIFY S19 1120 bunion" and an A1 episode with it, with dates 2026-11-03 and 2026-10-28, then retired the procedure.
   - A later PATCH of the consultant kept the procedure; a new episode with it gets 400.
   - In Chrome as Cara, the details show 03/11/2026 and 28/10/2026, and Edit procedure and dates shows "VERIFY S19 1120 bunion (no longer offered)" selected. I cancelled without saving.
4. Tenancy and roles over HTTP: see "Tenant isolation" above.

### Hand checks for Rahul

- **T-17 at 390×844:** the patient's home in device emulation at 390 px: the procedure line, and nothing scrolling sideways. The trk window can't go below 500 px. J7c checks the same home at phone size automatically, and passed.

### Test data I created (local database)

- Procedure "VERIFY S19 1120 bunion" (VR-1, Trust A, retired).
- Patient Vera Verifys (born 9 March 1971, vera.verify.s19@example.test), with one Hospital A1 episode `6fbfd30c…` on that procedure.

### Commits

1. `S19: procedures and episode dates — the trust's procedure list picks the default HQ` (the code and its tests, including the journeys).
2. `docs: S19 verified — build notes, test plan and reports, answers, verification, decisions, plan and session status`.

Next: `/build-session S20`.
