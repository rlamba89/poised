# Development plans: roadmap

Agreed 2 Oct 2026. These plans turn [../saas-requirements.md](../saas-requirements.md) into buildable steps. Requirement IDs (ORG-01, PAT-04…) and decision IDs (A-4, D-2…) refer to that document.

**How the work is ordered:**
1. **The foundation comes first (F1–F4).** It's the scaffolding everything else stands on: tests, the frontend move, Lambda entry points, and AWS with the pipeline. It adds no product features.
2. **Then the core features (C1–C10),** one plan at a time.
3. Every plan ends with something running that you can test before the next one starts.

## Order and status

| # | Plan | Delivers | Status |
| --- | --- | --- | --- |
| F1 | [Test foundation](f1-test-foundation.md) | API→database integration tests, a Playwright (Python) end-to-end suite covering today's workflow, and make targets | In progress: S01 verified |
| F2 | [Frontend: React + Vite + React Router](f2-react-vite.md) | Next.js replaced. F1's end-to-end suite proves nothing broke. | Not started |
| F3 | [Go API: Lambda entry points](f3-lambda-entrypoints.md) | staff-api, patient-api and jobs entry points over one shared router; local dev unchanged | Not started |
| F4 | [AWS and pipeline](f4-aws-pipeline.md) | CDK (Python), Neon, Amplify, API Gateway, Lambdas, CodePipeline with test gates, the QA environment, the per-country config and "add a country" runbook (A-20), and the idle cost measured | Not started |
| C1 | [Trusts and staff](c1-trusts-staff.md) | Trusts, hospitals, opt-ins, Cognito staff sign-in with MFA, roles, invites, audit log, the hand-made superadmin, and nothing hardcoded for the UK (A-20) | Not started |
| C2 | [Patients and sign-in](c2-patients-signin.md) | Patient records by NHS number, magic-link sign-in, invites by SMS/email, patient home, jobs and reminders | Not started |
| C3 | [Episode lifecycle](c3-episode-lifecycle.md) | Procedures, `episode_forms`, every status, worklists, tasks, cancel, archive | Not started |
| C4 | [Question keys and pre-fill](c4-question-keys-prefill.md) | Keys, question library, "Same as…", pre-fill and confirmation screens | Not started |
| C5 | [Authoring additions](c5-authoring-additions.md) | Capture blocks, Summary element, Video and Information elements, content library | Not started |
| C6 | [Scores, ASA and auto-triage](c6-scores-asa-triage.md) | Score bands, ASA contributions and rules, the `evaluate` Lambda, auto-triage | Not started |
| C7 | [Observations and assessments](c7-observations-assessments.md) | Observations; STOP-Bang first, then the other assessments | Not started |
| C8 | [Files](c8-files.md) | Files both ways, on the episode and the POA Summary | Not started |
| C9 | [Documents and PROMs](c9-documents-proms.md) | Document templates and PDF; PROMs plans, sends and results; licensed instruments as opt-ins | Not started |
| C10 | [Go-live](c10-go-live.md) | Move to RDS, the Training and Production stages, compliance, pen test | Not started |

**Work sessions:** F1–F4 and C1–C2 are split into sessions S01–S16, run as build → manual test → verify. See [../sessions/README.md](../sessions/README.md).

**Keep this table current.** When a plan's work is committed, mark it Done with the date, and note any differences from the plan in that plan's Status section.

**How detailed each plan is:**
- F1–F4 are written step by step, because they come next.
- C1–C10 set the scope, data, API, screens and tests. A session refines one into steps when it starts it, and asks Rahul about anything that changed.

## Testing rules (apply to every plan)

Healthcare software must not break silently. Each test type has a fixed job:

| Type | What it covers | Tool | Runs |
| --- | --- | --- | --- |
| **Unit (TDD)** | Pure logic: Go packages, `packages/clinical`, small React helpers | `go test`, Vitest | On every save, and in the pipeline build |
| **Integration (preferred)** | **Only the path from an API request to the database.** A real HTTP request goes into the real Go router, which uses a real migrated Postgres. The test checks the HTTP response **and** the rows in the database. | `go test -tags integration` | In the pipeline build, before any deploy |
| **End-to-end** | A user's journey through the real app in a browser | Playwright + Python (pytest), in `e2e/` | Locally against `make dev`; in the pipeline against QA after each deploy |

1. **Write the test first (TDD).** For every change, write the failing test (unit or integration), make it pass, then tidy up. A commit that adds behaviour without a test that would have failed is incomplete.
2. **Integration tests get preference.**
   - **Every API route** has integration tests for:
     - the happy path;
     - refused without a sign-in;
     - refused for the wrong role;
     - refused for **another trust's** data (tenant isolation);
     - invalid input.
   - Use unit tests for logic that has no HTTP or database: scoring, rules, parsers, status transitions.
3. **Integration tests cover API → database and nothing else.**
   - No frontend integration tests, and no tests against real AWS services.
   - Anything the API calls outside the database is behind a Go interface and replaced by a recording fake in tests: SMS, email, S3, Cognito, the `evaluate` Lambda and the clock. The fake lets the test assert what *would* have been sent.
   - **Scheduled jobs:** each job's *decision* (what's due, given the data and the time) is a pure function with unit tests. Its database writes reuse the store functions the API integration tests already cover.
4. **No fake database.** Today's handler tests use a fake querier (`fakeQ` in `internal/httpapi`). New tests use the real database. Old fake-querier tests are replaced by integration tests when their area is next touched.
5. **End-to-end tests follow journeys, not screens.**
   - Each plan adds or updates the journeys it affects.
   - Patient journeys run at **phone size** (Playwright device emulation).
   - Tests create their own data through the API, never depend on each other's order, and clean up or use unique names.
   - On failure, the Playwright trace is saved (and kept as a pipeline artifact).
6. **Pipeline gates (F4):**
   - unit and integration tests must pass before anything deploys;
   - end-to-end tests must pass on QA before Training or Production can be promoted.

## Definition of done (every plan)

- [ ] Tests were written first. `make test`, `make test-integration` and `make e2e` all pass locally.
- [ ] New API routes have the five integration cases from rule 2.
- [ ] The plan's end-to-end journeys exist and pass.
- [ ] It's deployed to QA by the pipeline, with the end-to-end tests green (from F4 on).
- [ ] The plan's Status section is updated, and this table is too.
- [ ] Each step is committed on its own, as in the earlier plans.
