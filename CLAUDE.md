# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**Preopsis** is a pre-operative assessment SaaS for hospitals (NHS trusts) and their patients. It was called **Poised** until 20 Oct 2026 and **sj-demo** until 2 Oct, and older docs may still use those names. Code and AWS names still say `poised` on purpose (the repo, the Go module, `@poised/*`, `poised-<env>-…`; [decisions.md](docs/decisions.md), 20 Oct). The logo and colours are in [docs/brand/](docs/brand/README.md).

Today it's a working local prototype:
- a clinical questionnaire **authoring tool** built on SurveyJS;
- one thin pass of the **episode workflow**: publish an HQ → the patient fills it in via a link → a clinician validates each Question Set → POA Summary.

The next work is turning it into a multi-trust SaaS on AWS.

**Read before starting any work:**
1. [docs/plans/README.md](docs/plans/README.md): the roadmap. Foundation plans F1–F4, then core plans C1–C10, with a status table. **Keep the table current.** Work in plan order unless Rahul says otherwise.
2. [docs/decisions.md](docs/decisions.md): every decision so far, with dates and reasons, plus Rahul's working preferences and the environment gotchas.
3. [docs/saas-requirements.md](docs/saas-requirements.md): the SaaS requirements and architecture. Its §0 is the decision log. Items marked *Proposed* aren't agreed, so ask before building them.
4. [docs/sessions/README.md](docs/sessions/README.md): the plans split into **work sessions** (S01…), each with a brief. The three project skills run them:
   - **`/build-session <id>`:** builds the session test-first and writes a manual test plan.
   - **`/manual-test <id>`:** a separate session that tests in Chrome like a manual tester. It never edits code.
   - **`/verify-session <id>`:** reviews and verifies before Rahul commits.
   - **`/run-sessions [from] [to]`:** the conductor. It runs all three automatically, session after session, and commits locally after each Verified verdict (never pushes). It stops only for Rahul's questions or a limit.
   - **Local first (12 Oct):** no AWS until the first demo or customer. Email and SMS are captured, and the dev login stands in for Cognito.
5. Background, as needed:
   - [docs/requirements.md](docs/requirements.md) (authoring requirement IDs such as PNL-07);
   - [docs/plan-workflow.md](docs/plan-workflow.md) (how the episode workflow is built);
   - [docs/pending.md](docs/pending.md) (known gaps).

## How to work here

- **Keep it simple.** Look for the simpler way before adding a service, library or abstraction.
- **Ask, with a recommendation:** for any stack or product choice, give options with a recommended one first. Don't assume. Check official docs (SurveyJS, AWS) before planning around a feature.
- **Backend is Go; keep TypeScript to the frontend and `packages/clinical`.** Rahul reads Go, not TypeScript, so explain any TypeScript you add that he must review.
- **The incumbent product is product evidence only, never the template for technical architecture.** Give every tech choice its own reason.
- **Testing** (full rules in [docs/plans/README.md](docs/plans/README.md#testing-rules-apply-to-every-plan)):
  - **TDD:** write the failing test first.
  - **Integration tests are preferred, and mean only API → database:** a real HTTP request goes through the real Go router into a real migrated Postgres, and the test asserts the response *and* the rows. Every route gets 5 cases: happy path, no sign-in, wrong role, another tenant's data, invalid input.
  - External services (SMS, email, S3, Cognito, the clock, the `evaluate` Lambda) are recording fakes behind Go interfaces. There are no other integration tests.
  - **End-to-end:** Playwright + Python (pytest) in `e2e/`, written as user journeys. Patient journeys run at phone size.
- **Commits:** one commit per plan step, each with its tests. Update the plan's Status section, and record any differences from the plan.
- **Record decisions in the repo, not in Claude memory.** Claude's memory and chat history stay on one machine. Add every new decision, preference or gotcha to [docs/decisions.md](docs/decisions.md) in the same session. The project skill **`/update-wiki`** (`.claude/skills/update-wiki/`) does this: run it whenever a decision is made or a plan step finishes, and before ending a session.
- **`spikes/ui-*.ts` and `spikes/mt/` are gitignored local scripts**, so don't rely on them existing. Committed browser tests are the journeys in `e2e/`.
- **No real patient data** goes into this system before go-live (plan C10). Seed and test data are made up.
- **The remote is Rahul's personal GitHub** (`github.com/rlamba89/poised`, private). Never push to the incumbent's GitHub organisation, and never name the incumbent product or its company anywhere in the repo, including file names, code, module paths, docs and commit messages; say "the incumbent" (22 Oct). `TestIncumbentNameAppearsNowhere` (`apps/api/internal/repocheck`, run by `make test`) fails if the name appears. It was scrubbed from the history on 2 Oct.

## Commands

Requires Docker, Go 1.26+ and Node 22.22+.

```sh
docker compose up -d db     # Postgres 17 on :5432 (user/password/db: sj)
make migrate seed           # goose migrations, then incumbent codes + demo hospitals/users (idempotent)
make dev                    # Go API on :8080 + web (Vite) on :3000 (web proxies /api/* to the API) + the evaluate service on :8090 (S41); Ctrl-C stops all three
make evaluate               # the evaluate service alone (npm run evaluate; EVALUATE_URL's port)
make patient-api            # one API function alone, as on AWS (ADDR=:8081 default); also make staff-api
make test                   # go test ./... + Vitest in every workspace + the CDK tests (no database needed)
make test-integration       # API → database tests (-tags integration): a fresh database per test
make lint                   # go vet (with the integration tag) + tsc --noEmit
make e2e-setup              # once: e2e/.venv (Python 3.12+), pinned Playwright + pytest, Chromium and WebKit
make e2e                    # journeys J1–J5 in real browsers against make dev (ARGS=--headed to watch)
make build-lambdas          # apps/api/dist/<fn>/bootstrap (arm64) for the Lambdas
make deploy-qa PROFILE=<aws profile>   # AWS QA by hand: stacks, migrate + seed Neon, web upload (infra/README.md; once per account: make aws-setup)
make deploy-pipeline PROFILE=<aws profile>  # once per account: the pipeline; then a push to master tests, deploys QA and runs e2e on it
make platform-admin EMAIL=… # add a superadmin by hand (A-21); idempotent
make sqlc                   # regenerate apps/api/internal/db after editing apps/api/db/queries/*.sql
make migrate-down           # roll back the last migration
```

**Running one test:**
- **Go:** `cd apps/api && go test ./internal/httpapi -run TestSaveContentRevision -v`
- **Go integration:** `cd apps/api && TEST_DATABASE_URL='postgres://sj:sj@localhost:5432/postgres?sslmode=disable' go test -tags integration -count=1 ./internal/httpapi -run TestSaveChapterContent -v`. The harness is `internal/apitest` (see its package comment); route tests are `internal/httpapi/*_integration_test.go`.
- **TypeScript (`packages/clinical`):** `cd packages/clinical && npx vitest run src/outputs.test.ts -t "computeOutputs"` (the web app's route table: `cd apps/web && npx vitest run`)
- **One journey:** `cd e2e && .venv/bin/pytest journeys/test_j3_patient.py --headed`. A failure keeps its trace in `e2e/test-results/` ([e2e/README.md](e2e/README.md)).

**Signing in locally:** use the stub login page at http://localhost:3000. The seeded users (S08) are:
- Ada Admin (admin, the whole of Trust A)
- Alex Author (super clinician, the whole of Trust A)
- Cara Clinician (clinician, Hospital A1 only)
- Sue Super (super clinician, Hospital A1 only: views but can't change the trust's questionnaires)
- Val Clinician (clinician, Hospital A2 only)
- Bea Author (super clinician, the whole of Trust B)
- Max Multi (clinician at Hospital A2; admin of the whole of Trust B)
- Sam Superadmin (a platform admin, in no trust)


## Architecture

npm workspaces monorepo (`apps/web`, `apps/evaluate`, `packages/*`), plus a separate Go module in `apps/api`.

- **`packages/clinical`** is plain TypeScript with no React. It holds the domain logic on top of SurveyJS JSON, used by the editor, the preview and the clinician and patient views:
  - the editor model (`doc.ts`);
  - conditions (`logic.ts`) and per-type settings (`fields.ts`);
  - custom properties and stable IDs (`properties.ts`, `ids.ts`);
  - viewer rules (`viewer.ts`);
  - **disclosures and outputs** (`outputs.ts`: `computeOutputs`);
  - clinical summaries, test cases, translations, publish checks (`publish.ts`), and the POA Summary (`poa.ts`);
  - the suggested ASA with its reasons (`asa.ts`; the whole HQ's is `hqSuggestedAsa` in `respondent.ts`, S39).
  - Most behaviour lives here and is unit-tested with Vitest.
- **`apps/web`** is a React single-page app: React + Vite + React Router + Mantine (plan F2 replaced Next.js, because Amplify Hosting supports only Next.js ≤ 15).
  - `src/routes.tsx` lists every URL and its page (`src/pages/…`); the real UI is in `src/features/{editor,preview,episodes,patient,outputs,questionnaires}`. See `apps/web/AGENTS.md`.
  - `src/lib/api.ts` is a thin fetch wrapper. Vite proxies `/api/*` to the Go API (`API_URL`).
  - `make dev` runs `vite`; `npm run build -w apps/web` then `npm run preview -w apps/web` serves the production build on :3000.
- **`apps/api`** is Go with stdlib `net/http` (Go 1.22+ method/wildcard patterns), pgx v5, **sqlc** (`internal/db` is generated, so don't edit it) and **goose** migrations (`db/migrations`). sqlc and goose run as Go tools.
  - `internal/httpapi/router.go` lists every route, in two routers: `NewStaffRouter` (`/api/o/…`, `/api/me`, the dev login only when `DEV_LOGIN=true` (local only), and `POST /api/auth/callback` when there's a Cognito staff pool (AWS, S12)) and `NewPatientRouter` (`/api/p/…` only); both serve `/api/health` and `GET /api/config`. `NewRouter` mounts both for `make dev` and the tests (`cmd/api`; `cmd/staff-api` and `cmd/patient-api` run one each).
  - **Every route that reads or changes patient data is wrapped in `audited`** (`internal/httpapi/audit.go`, S11): one append-only `audit_log` row per request, IDs only. A caller who left mid-request is status 499, shown as Cancelled. Wrap any new patient-data route too.
  - `internal/config` reads every setting from the environment. It's the only place a per-country value may live (A-20; a test fails on a locale, time zone or region literal anywhere else). Local dev defaults to the UK; `CONFIG_STRICT=true` refuses to start with one missing. The web app gets the locale from `GET /api/config` (`apps/web/src/lib/config.ts`).
  - Middleware: `requireUser` (a dev HS256 JWT in an httpOnly cookie), then `requireHospital` for `/api/o/{oid}/h/{hid}/…` or `requireOrg` for `/api/o/{oid}/…`. They re-read the memberships on every request and give non-members a 404. The role ladder is `internal/role` (`clinician < super_clinician < admin`, A-5).
  - `internal/chapter` holds the content rules. `chapter.ForPatient` strips clinician-only elements, `clinicalOutputs` and `testCases` before patient JSON leaves the server, and `FilterAnswers` drops non-patient answer keys.
  - `internal/patient` holds the patient-record rules (matching typed details to the trust's record, the search term, E.164 mobiles); `internal/nationalid` the identifier validators (the NHS number's Modulus 11).
  - `internal/asaguide` is the ASA guide (S39): the ASA's published examples per code (`guide.json`, embedded), sent with each code by `GET /api/codes` as `asaChoices`; made up for testing until a clinician checks it.
  - `internal/episode` handles episode statuses. `internal/hqimport` + `cmd/import-hq` import an incumbent HQ export.
- **Content model:**
  - **Tenancy (S08):** questionnaires and option lists belong to a trust (`org_id`; `hospital_id` NULL = every hospital of the trust). Patient records belong to the trust; an episode belongs to one hospital, and a hospital-scoped user sees only their hospital's episodes.
  - A questionnaire has versions (draft → published → retired), and a version has **chapters ("Question Sets")**. Each chapter is one SurveyJS JSON document (`chapters.content`), with a `revision` that is bumped on every save; a stale revision → 409.
  - **Published versions are read-only.** "New version" copies chapters, keeping stable IDs.
  - **Episodes** hold **forms** (`episode_forms`, S18): today only one, the HQ, on one published version, created with the episode. C7 adds assessments and C9 PROMs. The routes still name only the episode and resolve its HQ form on the server.
  - **Procedures (S19):** the trust's list (`procedures`: name, code, specialty, default HQ by questionnaire, retired; `hospital_id` NULL = every hospital). Whole-trust super clinicians and admins manage it (`/api/o/{oid}/procedures`); an episode names one (`procedure_id`, optional) with its `planned_date` and `poa_date`. A retired procedure stays on its episodes but isn't offered. Picking one on a new episode chooses its default HQ, else the hospital's `defaultHq` setting (Go resolves both).
  - **Information (S36, INF-04):** a procedure's videos and leaflets (`procedures.info_content`, content library ids) are **copied onto an episode when it's made** (`episode_info_items`; Rahul, 25 Oct: a booked patient keeps their list); changing the episode's procedure swaps the copied items, and staff add or remove items for one episode. The patient sees the active ones, with the hospital's phone and email, under Information on their home. The estimated time per Question Set is `packages/clinical`'s `estimatedMinutes`, with the seconds per question from `SECONDS_PER_QUESTION` (Go config, default 15, sent by `GET /api/config`).
  - **Answers are stored per form, per actor** (`episode_answers`: the patient's answers, frozen when the form is submitted, plus the clinician's validated copy), so the POA Summary can show both.
  - **Pre-fill (S25):** the first open of an HQ form fills each patient set with no answers yet from the patient's earlier submitted HQs in the same trust (`internal/prefill`: by key, then by exact wording). It runs **once per form** (S28: `episode_forms.prefilled_at`, set under a row lock), so later opens never read earlier HQs again. It's stored as the patient's row at revision 1 (S26: 0 means "no row yet" to a save), with `prefill` saying where each answer came from. "Started" is the generated column `episode_answers.started`: saved past that revision, or a pre-filled set's card answered.
  - **Other trusts (S26, S27):** another trust's answers are **always copied, without asking** (Rahul, 24 Oct; S27 removed S26's `crossTrustPrefill` gate, the patient's consent question and `patient_prefill_consents`, migration 00027), for the patient and for staff on their behalf; a suspended trust's aren't. A set holding them opens on a card naming the hospital, with **Start fresh** (`prefill_choice` `start_fresh`: Go clears that set's pre-filled answers, only while the set is unchanged; the form's own saves that only clear hidden answers don't count, `cleared_saves`, S44c). They fill only what the form's own trust's forms didn't ask, and match by key only for a **platform library key**, else by exact wording (S23's lock holds within one trust only); choices are always mapped by option text, and trust A's map keeps no trust B id. They're read one trust per query (`ListPrefillSources` with that trust's `org_id`), and each read writes a `prefill.read` audit row under the source trust; caused by staff, that row names only their trust ("Staff at Trust A": `audit_log.actor_org_id`, no `user_id`, S28). **Fill in for the patient** (S28) first joins a record with no account to the existing account matching all four fields (date of birth, mobile or email, first name, surname): that record only, no new account, no contacts copied, an `account.join` row by the staff member (`joinOnBehalf`). The clinician's view and the POA Summary mark every pre-filled answer shown (source, link or wording, "Changed by patient").
- **ASA rules (S40):** each questionnaire version holds its rules (`questionnaire_versions.rules`, one JSON object: the rules, `noGradeAutoTriage`, whole-HQ test cases; `rules_revision` for stale saves). A new HQ and a new version of one without rules get the platform defaults (`internal/asarules/defaults.json`); Go checks only the shape. Which rules fire is `packages/clinical`'s `rules.ts` (preview, test cases, and from S41 the evaluate service). A rule's "when" may test that a score is *not* in some bands (`notBand`, S44c): the defaults' Low risk isn't for a High STOP-Bang (an HQ without STOP-Bang passes).
- **Auto-triage (S42):** on send, `internal/triage.Decide` reads the stored result: the first fired rule with an auto-triage action moves the episode from Ready for review to its target (the system-only move `auto_triage`, `episode.CheckAutomatic`; the history names the rule), unless a guard stops it (evaluator failed, a red or amber flag, a score Incomplete, the anaesthetist's ASA required). `packages/clinical`'s `firedRules` applies the same guards (`heldReason`), so the Preview and test cases say what a send does. The episode's `triage_*` columns hold it (a re-open clears them); it's on the **Auto-triaged** worklist until a clinician POSTs `…/triage-checked`. The patient's status message goes at once, or on Checked if the HQ's rules say `messageAfterCheck`; a held message survives a change that sends no message of its own (On hold, archive), so an Undo or un-archive back lets Checked send it (S44c). The other actions are S43's:
- **The rules' other actions (S43):** on the same send, before auto-triage, `internal/triage.ActionsOf` turns the fired rules into writes (`ruleActions` in `httpapi/patient.go`): a task for the rule's team (`tasks.created_by_rule`; not again while that rule's task is open), a flag (`episode_flags`; not again while it stands), `episodes.anaesthetist_asa_required_by` (while no anaesthetist's grade is saved, a move to Ready for admission is a 409 "Add the anaesthetist's ASA grade first." and the move comes in `moves` with `blocked`; un-archive isn't held back), and `episodes.rule_suggestions` (shown, never applied); each with an automatic General notes line. Any clinician at the hospital clears a flag with a reason (`POST …/flags/{fid}/clear`, audited `flag.clear`); the flags come with the episode and the list rows.
- **The anaesthetist's ASA grade (S44b)** is a history, never changed or cleared: `anaesthetist_asa_grades` (migration 00035; `episodes.anaesthetist_asa` is gone, triggers refuse any edit but one entered-in-error mark, and any delete but the episode's own). Any clinician at the hospital adds one (`POST …/anaesthetist-asa`; a reason required for every grade after the episode's first, 422) or marks one entered in error with a reason (`POST …/anaesthetist-asa/{gid}/entered-in-error`), audited `asa.add` / `asa.error`; the grade that **counts** is the latest not entered in error (`episode.anaesthetistAsa`, read after the lock by `countingGrade`; the history is `anaesthetistAsaGrades`). The episode PATCH refuses the grade (400). While a rule requires it at Ready for admission or later, the only counting grade can't be marked (409).
- **Observations (S46, OBS-01–04):** a clinician's readings on an episode (`observations`, migration 00037; `internal/observation`: the kinds, stored units, the platform's ranges and the hard limits). Never edited: **Correct** (a new row; the old one superseded) or **Entered in error**, each with a reason (`…/observations/{obsid}/correct|entered-in-error`). Outside the usual range the panel warns, on a correction too; outside the hard limits, or before the patient's date of birth, Go refuses (400). A correction marked entered in error doesn't bring back what it replaced, unlike the ASA grades (decisions.md, 30 Oct). A trust's admins may set their own ranges (the trust setting `observations.ranges`, within fixed rules); the display units for glucose and temperature are country config (`GLUCOSE_UNIT`, `TEMPERATURE_UNIT`). The **hospital BMI** (latest height and weight that count) is `packages/clinical`'s `hospitalBmi`, beside Patient reported BMI on the POA Summary. A save of the ranges sends what the page loaded, and a stale one is a 409. `feature.observations` off: the routes are 403, and `GET …/episodes/{eid}`'s `observationsOn` keeps the web app from calling them.
- **Assessments (S47, §9; S48):** validated instruments a clinician adds to an episode (`POST …/episodes/{eid}/assessments`, "Add assessment → STOP-Bang", DASI, PRISMA-7 or Apfel PONV), completes and confirms. Each is a **platform questionnaire** (`questionnaires.assessment`, no trust or author, migration 00038) whose locked JSON is `internal/assessment/<id>.json`, with the list in `list.json` and each item's message name in the content's `labels`, all written from `packages/clinical`'s `assessments.ts` (`npm run assessments -w packages/clinical`; its test fails if they differ); the first add publishes it, a change of wording publishes a new version (episodes keep theirs), and any version with the same content is reused (S48). DASI adds its METs to the result; the HQ's Apfel is the patient's two questions only, the opioids item the clinician's (S48 answers 4). The form is `episode_forms` kind `assessment` (one of each per episode); the clinician's answers are its clinician row in `episode_answers`, with `sources` (hq, observation, clinician; and profile for the patient's age and sex, frozen at Confirm, S47 answers 7) and the revision check; **Confirm** is that row's `validated_at` (`…/assessments/{fid}/confirm`), and a save without it clears it. The browser pre-fills each item (`assessmentItems`: by key from the HQ's answers that count, once sent (a set's clinician copy once started); the hospital BMI or the patient's as the trust's `observations.precedence` says; age and sex from the record) and works out the score with the patient's age on the day of Confirm; Go only checks the shape (`assessment.Check`). The trust's queries cast `org_id`/`created_by` (`sqlc.arg(org_id)::uuid`) so sqlc's types didn't change. `feature.assessments` off: 403, and the episode GET's `assessmentsOn`.
- **Content library (S33):** `content_items` (`org_id` NULL = Preopsis's, seen by every trust): videos (YouTube or Vimeo links only, A-6, checked by `internal/content`; length typed by hand) and leaflets (an https link until C8), with free tags and PX-13's checklist topics. Platform admins manage Preopsis's (`/api/admin/content`), a trust's whole-trust admins its own (`/api/o/{oid}/content`); authors search the active ones (`…/content/search`). Items are retired, never deleted. A platform admin can also retire a starter library question (`library_questions.retired`): it's no longer offered, but questions already inserted keep working.
- **Scores (S38):** a Calculation's score adds up option points from its own and **earlier** Question Sets (by key) and "Add N points when…" lines (age, sex, BMI, earlier answers, another score's band), built from menus, never typed; a missing input makes it **Incomplete** (no band). One helper, `packages/clinical`'s `earlierContexts`, gives each set's survey the earlier shown sets' answers, in every view. **STOP-Bang, DASI, PRISMA-7 and Apfel** are ready-made and locked (`ready.ts`, each a `ReadyScore` with its source, licence and `build`); their published scoring is `ready.test.ts`. The licences of DASI (Greenlight Health) and PRISMA-7 (CESS) are go-live items, beside STOP-Bang's (S48).
- **Key rule (with one exception):** Go never evaluates SurveyJS expressions, because no non-JS evaluator exists. Outputs are computed in the browser from `packages/clinical`, and Go only stores and enforces.
  - **Exception (agreed 2 Oct, plan C6; built in S41):** the **evaluate service** runs `packages/clinical` on the server. `packages/clinical/src/evaluate.ts` holds its pure functions (`evaluate`: what a sent HQ's answers give, as the clinician's view shows them before validation; `checkPublish`: the browser's publish checks; `shownSetsFor` (S44): the sets shown for the answers that count now, for Complete review); `apps/evaluate` serves them over HTTP on 127.0.0.1 (`npm run evaluate`, started by `make dev`; built with esbuild, its version the git commit). Go calls it through `internal/evaluate` (`Evaluator`; `HTTP` at `EVALUATE_URL`; `Recorder`, the fake in `internal/apitest`). On send (both submit routes) Go stores the reply in `form_outputs` (S42 auto-triages from it); if the call fails the send still works, the error is stored, an `EVALUATE_FAILED` line is logged (no answers) and the General notes say "The automatic checks didn't run. Review this HQ by hand." Publish asks it for the publish checks too (422 with the problems; 503 if it can't be reached). **S44:** a send with a set the patient's form shows incomplete is refused (409 "Please finish “…” before sending your answers.", nothing frozen), and Complete review is refused while a set shown has no validated copy (409 "Validate every Question Set first: …"); if the service can't be reached, both go through as before (the browser's checks stand). **S44a:** `/evaluate` also returns `sent` (the patient's answers as their form showed them) and Go's `keepSent` drops the rest from the frozen answers; a score input a condition hides adds 0 (`scores.ts` `inputHidden`). The shared fixtures `apps/api/internal/evaluate/testdata/*.json` keep both sides in step. S45 puts it in a Lambda.
  - Go may walk the JSON (for example, to strip clinician-only content), but it never evaluates.

## Target architecture (plans F3–F4, C1+)

AWS, serverless, near-£0 when idle before the first customer. The details are in [docs/saas-requirements.md §13](docs/saas-requirements.md).
- **Compute:** four Lambdas from one Go codebase (`staff-api`, `patient-api`, `jobs`) plus the Node `evaluate`, behind API Gateway HTTP API.
- **Frontend:** a React SPA on **Amplify Hosting**. Rewrites send `/api/*` to API Gateway and deep links to the SPA.
- **Database:** **Postgres on Neon** until the first customer, then **RDS** (plain Postgres throughout).
- **Sign-in:** Cognito for staff (managed login + TOTP MFA → our own session cookie). Patients sign in by **magic link + date of birth**, built in Go.
- **Messaging:** SES for email and AWS End User Messaging for SMS.
- **Infrastructure:** **CDK in Python** (`infra/`) and **CodePipeline** (CDK Pipelines). No SAM.
- **Tenancy:** a **trust** (`org_id`) owns many hospitals, and every patient-data query is scoped to one trust.
- **Countries (A-20, agreed 8 Oct):** **one deployment per country**, each at its own web address (`uk.<domain>`) in its own AWS region. Only the UK (eu-west-2) runs now.
  - **Never hardcode the UK or London** (region, domain, time zone, locale, phone country, NHS-only identifiers).
  - Per-country values live only in `infra/countries.py` and Go's `internal/config`, and the web app reads them from `GET /api/config`.
  - A new country must be config plus a runbook, never a code change.
