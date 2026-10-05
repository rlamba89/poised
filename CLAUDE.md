# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

**Poised** is a pre-operative assessment SaaS for hospitals (NHS trusts) and their patients. It was called **sj-demo** until 2 Oct 2026, and older docs may still use that name.

Today it's a working local prototype:
- a clinical questionnaire **authoring tool** built on SurveyJS;
- one thin pass of the **episode workflow**: publish an HQ → the patient fills it in via a link → a clinician validates each Question Set → POA Summary.

The next work is turning it into a multi-trust SaaS on AWS.

**Read before starting any work:**
1. [docs/plans/README.md](docs/plans/README.md): the roadmap. Foundation plans F1–F4, then core plans C1–C10, with a status table. **Keep the table current.** Work in plan order unless Rahul says otherwise.
2. [docs/decisions.md](docs/decisions.md): every decision so far, with dates and reasons, plus Rahul's working preferences and the environment gotchas.
3. [docs/saas-requirements.md](docs/saas-requirements.md): the SaaS requirements and architecture. Its §0 is the decision log. Items marked *Proposed* aren't agreed, so ask before building them.
4. Background, as needed:
   - [docs/requirements.md](docs/requirements.md) (authoring requirement IDs such as PNL-07);
   - [docs/plan-workflow.md](docs/plan-workflow.md) (how the episode workflow is built);
   - [docs/pending.md](docs/pending.md) (known gaps).

## How to work here

- **Keep it simple.** Look for the simpler way before adding a service, library or abstraction.
- **Ask, with a recommendation:** for any stack or product choice, give options with a recommended one first. Don't assume. Check official docs (SurveyJS, AWS) before planning around a feature.
- **Backend is Go; keep TypeScript to the frontend and `packages/clinical`.** Rahul reads Go, not TypeScript, so explain any TypeScript you add that he must review.
- **Lifebox (the incumbent product) is product evidence only, never the template for technical architecture.** Give every tech choice its own reason.
- **Testing** (full rules in [docs/plans/README.md](docs/plans/README.md#testing-rules-apply-to-every-plan)):
  - **TDD:** write the failing test first.
  - **Integration tests are preferred, and mean only API → database:** a real HTTP request goes through the real Go router into a real migrated Postgres, and the test asserts the response *and* the rows. Every route gets 5 cases: happy path, no sign-in, wrong role, another tenant's data, invalid input.
  - External services (SMS, email, S3, Cognito, the clock, the `evaluate` Lambda) are recording fakes behind Go interfaces. There are no other integration tests.
  - **End-to-end:** Playwright + Python (pytest) in `e2e/`, written as user journeys. Patient journeys run at phone size.
- **Commits:** one commit per plan step, each with its tests. Update the plan's Status section, and record any differences from the plan.
- **Record decisions in the repo, not in Claude memory.** Claude's memory and chat history stay on one machine. Add every new decision, preference or gotcha to [docs/decisions.md](docs/decisions.md) in the same session. The project skill **`/update-wiki`** (`.claude/skills/update-wiki/`) does this: run it whenever a decision is made or a plan step finishes, and before ending a session.
- **`spikes/ui-*.ts` and `spikes/mt/` are gitignored local scripts**, so don't rely on them existing. Committed browser tests go in `e2e/` (plan F1).
- **No real patient data** goes into this system before go-live (plan C10). Seed and test data are made up.
- **The remote is Rahul's personal GitHub** (`github.com/rlamba89/poised`, private). Never push to a Lifebox organisation, and never put the Lifebox company's GitHub organisation name anywhere in the repo, including module paths, docs and commit messages. It was scrubbed from the history on 2 Oct.

## Commands

Requires Docker, Go 1.26+ and Node 20.9+.

```sh
docker compose up -d db     # Postgres 17 on :5432 (user/password/db: sj)
make migrate seed           # goose migrations, then Lifebox codes + demo hospitals/users (idempotent)
make dev                    # Go API on :8080 + web on :3000 (web proxies /api/* to the API); Ctrl-C stops both
make test                   # go test ./... + Vitest in every workspace
make lint                   # go vet + tsc --noEmit
make sqlc                   # regenerate apps/api/internal/db after editing apps/api/db/queries/*.sql
make migrate-down           # roll back the last migration
```

**Running one test:**
- **Go:** `cd apps/api && go test ./internal/httpapi -run TestSaveContentRevision -v`
- **TypeScript (`packages/clinical`):** `cd packages/clinical && npx vitest run src/outputs.test.ts -t "computeOutputs"`

**Signing in locally:** use the stub login page at http://localhost:3000. The seeded users are:
- Alex Author (author + publisher, Hospital A)
- Val Viewer (Hospital A)
- Cara Clinician (clinician, Hospital A)
- Bea Author (Hospital B)

**Planned targets that don't exist yet:** `make test-integration` (API→database, `-tags integration`), `make e2e-setup` and `make e2e` (Playwright). They arrive with plan F1.

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
- **`apps/web`** is today Next.js 16 + Mantine, but every page is a client component.
  - Pages live in `src/app/**/page.tsx`, and the real UI is in `src/features/{editor,preview,episodes,patient,outputs,questionnaires}`.
  - `src/lib/api.ts` is a thin fetch wrapper.
  - **Plan F2 replaces Next.js with React + Vite + React Router**, because Amplify Hosting supports only Next.js ≤ 15. `apps/web/AGENTS.md` (the Next.js agent rules) goes away then.
- **`apps/api`** is Go with stdlib `net/http` (Go 1.22+ method/wildcard patterns), pgx v5, **sqlc** (`internal/db` is generated, so don't edit it) and **goose** migrations (`db/migrations`). sqlc and goose run as Go tools.
  - `internal/httpapi/router.go` lists every route. Middleware: `requireUser` (a dev HS256 JWT in an httpOnly cookie) and `requireHospital` (role check per hospital; routes are `/api/h/{hid}/…`).
  - `internal/chapter` holds the content rules. `chapter.ForPatient` strips clinician-only elements, `clinicalOutputs` and `testCases` before patient JSON leaves the server, and `FilterAnswers` drops non-patient answer keys.
  - `internal/episode` handles episode statuses. `internal/lifebox` + `cmd/import-lifebox` import a Lifebox HQ export.
- **Content model:**
  - A questionnaire has versions (draft → published → retired), and a version has **chapters ("Question Sets")**. Each chapter is one SurveyJS JSON document (`chapters.content`), with a `revision` that is bumped on every save; a stale revision → 409.
  - **Published versions are read-only.** "New version" copies chapters, keeping stable IDs.
  - **Episodes** are tied to one published version.
  - **Answers are stored per actor** (`episode_answers`: the patient's answers, frozen at submit, plus the clinician's validated copy), so the POA Summary can show both.
- **Key rule (with one planned exception):** Go never evaluates SurveyJS expressions, because no non-JS evaluator exists. Outputs are computed in the browser from `packages/clinical`, and Go only stores and enforces.
  - **Exception (agreed 2 Oct, plan C6):** a small Node `evaluate` Lambda will run `packages/clinical` on the server when a patient submits, so auto-triage can be trusted.
  - Go may walk the JSON (for example, to strip clinician-only content), but it never evaluates.

## Target architecture (plans F3–F4, C1+)

AWS **eu-west-2**, serverless, near-£0 when idle before the first customer. The details are in [docs/saas-requirements.md §13](docs/saas-requirements.md).
- **Compute:** four Lambdas from one Go codebase (`staff-api`, `patient-api`, `jobs`) plus the Node `evaluate`, behind API Gateway HTTP API.
- **Frontend:** a React SPA on **Amplify Hosting**. Rewrites send `/api/*` to API Gateway and deep links to the SPA.
- **Database:** **Postgres on Neon** until the first customer, then **RDS** (plain Postgres throughout).
- **Sign-in:** Cognito for staff (managed login + TOTP MFA → our own session cookie). Patients sign in by **magic link + date of birth**, built in Go.
- **Messaging:** SES for email and AWS End User Messaging for SMS.
- **Infrastructure:** **CDK in Python** (`infra/`) and **CodePipeline** (CDK Pipelines). No SAM.
- **Tenancy:** a **trust** (`org_id`) owns many hospitals, and every patient-data query is scoped to one trust. All patient-data access goes through `store.ForOrg(orgID)`, so a trust's data can later live in another AWS region (code stays in London).
