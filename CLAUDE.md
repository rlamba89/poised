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
make dev                    # Go API on :8080 + web (Vite) on :3000 (web proxies /api/* to the API); Ctrl-C stops both
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

npm workspaces monorepo (`apps/web`, `packages/*`), plus a separate Go module in `apps/api`.

- **`packages/clinical`** is plain TypeScript with no React. It holds the domain logic on top of SurveyJS JSON, used by the editor, the preview and the clinician and patient views:
  - the editor model (`doc.ts`);
  - conditions (`logic.ts`) and per-type settings (`fields.ts`);
  - custom properties and stable IDs (`properties.ts`, `ids.ts`);
  - viewer rules (`viewer.ts`);
  - **disclosures and outputs** (`outputs.ts`: `computeOutputs`);
  - clinical summaries, test cases, translations, publish checks (`publish.ts`), and the POA Summary (`poa.ts`).
  - Most behaviour lives here and is unit-tested with Vitest.
- **`apps/web`** is a React single-page app: React + Vite + React Router + Mantine (plan F2 replaced Next.js, because Amplify Hosting supports only Next.js ≤ 15).
  - `src/routes.tsx` lists every URL and its page (`src/pages/…`); the real UI is in `src/features/{editor,preview,episodes,patient,outputs,questionnaires}`. See `apps/web/AGENTS.md`.
  - `src/lib/api.ts` is a thin fetch wrapper. Vite proxies `/api/*` to the Go API (`API_URL`).
  - `make dev` runs `vite`; `npm run build -w apps/web` then `npm run preview -w apps/web` serves the production build on :3000.
- **`apps/api`** is Go with stdlib `net/http` (Go 1.22+ method/wildcard patterns), pgx v5, **sqlc** (`internal/db` is generated, so don't edit it) and **goose** migrations (`db/migrations`). sqlc and goose run as Go tools.
  - `internal/httpapi/router.go` lists every route, in two routers: `NewStaffRouter` (`/api/o/…`, `/api/me`, the dev login only when `DEV_LOGIN=true` (local only), and `POST /api/auth/callback` when there's a Cognito staff pool (AWS, S12)) and `NewPatientRouter` (`/api/p/…` only); both serve `/api/health` and `GET /api/config`. `NewRouter` mounts both for `make dev` and the tests (`cmd/api`; `cmd/staff-api` and `cmd/patient-api` run one each).
  - **Every route that reads or changes patient data is wrapped in `audited`** (`internal/httpapi/audit.go`, S11): one append-only `audit_log` row per request, IDs only. Wrap any new patient-data route too.
  - `internal/config` reads every setting from the environment. It's the only place a per-country value may live (A-20; a test fails on a locale, time zone or region literal anywhere else). Local dev defaults to the UK; `CONFIG_STRICT=true` refuses to start with one missing. The web app gets the locale from `GET /api/config` (`apps/web/src/lib/config.ts`).
  - Middleware: `requireUser` (a dev HS256 JWT in an httpOnly cookie), then `requireHospital` for `/api/o/{oid}/h/{hid}/…` or `requireOrg` for `/api/o/{oid}/…`. They re-read the memberships on every request and give non-members a 404. The role ladder is `internal/role` (`clinician < super_clinician < admin`, A-5).
  - `internal/chapter` holds the content rules. `chapter.ForPatient` strips clinician-only elements, `clinicalOutputs` and `testCases` before patient JSON leaves the server, and `FilterAnswers` drops non-patient answer keys.
  - `internal/patient` holds the patient-record rules (matching typed details to the trust's record, the search term, E.164 mobiles); `internal/nationalid` the identifier validators (the NHS number's Modulus 11).
  - `internal/episode` handles episode statuses. `internal/hqimport` + `cmd/import-hq` import an incumbent HQ export.
- **Content model:**
  - **Tenancy (S08):** questionnaires and option lists belong to a trust (`org_id`; `hospital_id` NULL = every hospital of the trust). Patient records belong to the trust; an episode belongs to one hospital, and a hospital-scoped user sees only their hospital's episodes.
  - A questionnaire has versions (draft → published → retired), and a version has **chapters ("Question Sets")**. Each chapter is one SurveyJS JSON document (`chapters.content`), with a `revision` that is bumped on every save; a stale revision → 409.
  - **Published versions are read-only.** "New version" copies chapters, keeping stable IDs.
  - **Episodes** hold **forms** (`episode_forms`, S18): today only one, the HQ, on one published version, created with the episode. C7 adds assessments and C9 PROMs. The routes still name only the episode and resolve its HQ form on the server.
  - **Procedures (S19):** the trust's list (`procedures`: name, code, specialty, default HQ by questionnaire, retired; `hospital_id` NULL = every hospital). Whole-trust super clinicians and admins manage it (`/api/o/{oid}/procedures`); an episode names one (`procedure_id`, optional) with its `planned_date` and `poa_date`. A retired procedure stays on its episodes but isn't offered. Picking one on a new episode chooses its default HQ, else the hospital's `defaultHq` setting (Go resolves both).
  - **Answers are stored per form, per actor** (`episode_answers`: the patient's answers, frozen when the form is submitted, plus the clinician's validated copy), so the POA Summary can show both.
  - **Pre-fill (S25):** the first open of an HQ form fills each patient set with no answers yet from the patient's earlier submitted HQs in the same trust (`internal/prefill`: by key, then by exact wording). It's stored as the patient's row at revision 1 (S26: 0 means "no row yet" to a save), with `prefill` saying where each answer came from. "Started" is the generated column `episode_answers.started`: saved past that revision, or a pre-filled set's card answered.
  - **Other trusts (S26):** another trust's answers are used only when **both** trusts have `crossTrustPrefill` on **and** the patient said Yes to "Use the answers you gave …?" (`patient_prefill_consents`, once per account and source trust), never on the patient's behalf. They fill only what the form's own trust's forms didn't ask, and match by key only for a **platform library key**, else by exact wording (S23's lock holds within one trust only); choices are always mapped by option text, and trust A's map keeps no trust B id. They're read one trust per query (`ListPrefillSources` with that trust's `org_id`), and each read writes a `prefill.read` audit row under the source trust. The clinician's view and the POA Summary mark every pre-filled answer shown (source, link or wording, "Changed by patient").
- **Key rule (with one planned exception):** Go never evaluates SurveyJS expressions, because no non-JS evaluator exists. Outputs are computed in the browser from `packages/clinical`, and Go only stores and enforces.
  - **Exception (agreed 2 Oct, plan C6):** a small Node `evaluate` Lambda will run `packages/clinical` on the server when a patient submits, so auto-triage can be trusted.
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
