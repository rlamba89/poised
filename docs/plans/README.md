# Development plans: roadmap

Agreed 2 Oct 2026. These plans turn [../saas-requirements.md](../saas-requirements.md) into buildable steps. Requirement IDs (ORG-01, PAT-04…) and decision IDs (A-4, D-2…) refer to that document.

**How the work is ordered:**
1. **The foundation comes first (F1–F4).** It's the scaffolding everything else stands on: tests, the frontend move, Lambda entry points, and AWS with the pipeline. It adds no product features.
2. **Then the core features (C1–C10),** one plan at a time.
3. Every plan ends with something running that you can test before the next one starts.

## Order and status

| # | Plan | Delivers | Status |
| --- | --- | --- | --- |
| F1 | [Test foundation](f1-test-foundation.md) | API→database integration tests, a Playwright (Python) end-to-end suite covering today's workflow, and make targets | **Done** (12 Oct 2026): S01, S02, S02a and S03 verified |
| F2 | [Frontend: React + Vite + React Router](f2-react-vite.md) | Next.js replaced. F1's end-to-end suite proves nothing broke. | **Done** (12 Oct 2026): S04 verified |
| F3 | [Go API: Lambda entry points](f3-lambda-entrypoints.md) | staff-api, patient-api and jobs entry points over one shared router; local dev unchanged | **Done** (14 Oct 2026): S05 (local half) verified 12 Oct; S06 (the Lambda half) verified 14 Oct |
| F4 | [AWS and pipeline](f4-aws-pipeline.md) | CDK (Python), Neon, Amplify, API Gateway, Lambdas, CodePipeline with test gates, the QA environment, the per-country config and "add a country" runbook (A-20), and the idle cost measured | **Done** (14 Oct 2026): S06 and S07 verified; the pipeline is live and proven, and the second-region proof is done. Only Step 4's idle-cost figure is still to come (Rahul reads Cost Explorer on 21 Oct) |
| C1 | [Trusts and staff](c1-trusts-staff.md) | Trusts, hospitals, opt-ins, Cognito staff sign-in with MFA, roles, invites, audit log, the hand-made superadmin, and nothing hardcoded for the UK (A-20) | **Done** (14 Oct 2026): S08–S11 verified 13 Oct; S12 (Cognito staff sign-in with MFA) verified 14 Oct. Rahul's hand checks: S12's H-1 to H-3 |
| C2 | [Patients and sign-in](c2-patients-signin.md) | Patient records by NHS number, magic-link sign-in, invites by SMS/email, patient home, jobs and reminders | **Done** (15 Oct 2026): S13–S16 verified 13 Oct; S17 (jobs and real messaging on AWS) verified 15 Oct. Rahul's hand checks: S17's verify.md |
| C3 | [Episode lifecycle](c3-episode-lifecycle.md) | Procedures, `episode_forms`, every status, worklists, tasks, cancel, archive | **Done** (21 Oct 2026): S18 verified 20 Oct; S19–S22 verified 21 Oct. EPI-08 and EPI-09 included. Rahul's hand checks: S22's verify.md |
| C4 | [Question keys and pre-fill](c4-question-keys-prefill.md) | Keys, question library, "Same as…", pre-fill and confirmation screens | **Done** (23 Oct 2026): S23 verified (22 Oct, verification 3); S24 verified (23 Oct, verification 3); S25 verified (23 Oct, verification 1); S26 verified (23 Oct, verification 2) |
| C5 | [Authoring additions](c5-authoring-additions.md) | Capture blocks, Summary element, Video and Information elements, content library | **Done** (26 Oct 2026): split into S31–S36 (25 Oct); S31 (capture blocks) dropped by Rahul, capture blocks deferred; S32–S35 verified (25–26 Oct); S36 verified (26 Oct) |
| C6 | [Scores, ASA and auto-triage](c6-scores-asa-triage.md) | Score bands, ASA contributions and rules, the `evaluate` Lambda, auto-triage | **Done** (29 Oct 2026): split (26 Oct 2026) into S38–S45 (S45 in the AWS phase); Rahul answered the questions 26 Oct; S38 verified (26 Oct, verification 3); S39 verified (26 Oct, verification 2); S40 verified (26 Oct, verification 2); S41 verified (27 Oct, verification 1); S42 verified (27 Oct, verification 1); S43 verified (27 Oct, verification 2, after verification 1's fixes); S44 verified (27 Oct, verification 1); S44a verified (27 Oct, verification 3, after verifications 1–2's fixes); S44b verified (28 Oct, verification 2, after verification 1's fixes); S44c added 28 Oct (C6's last open items; tested 28 Oct; verification 1 needed fixes (F1, F2), done; test report 3 all passed; verification 2 needed one fix (F3), done; test report 4 all passed; verification 3 needed fixes (F4, F5, Rahul's answer 8), done; test report 5 all passed; verified 28 Oct, verification 4); S45 (AWS phase, with Rahul): code built 28 Oct ([build notes](../sessions/S45-evaluate-lambda/build-notes.md)); deployed to QA by hand and measured 29 Oct; tested 29 Oct (test report 1 all passed); verified 29 Oct (verification 1); H-2, a green pipeline after the push, still to do |
| C7 | [Observations and assessments](c7-observations-assessments.md) | Observations; STOP-Bang first, then the other assessments | **Done 14 Nov 2026:** split (29 Oct 2026) into S46–S49 ([sessions](../sessions/README.md)); S46 verified 2 Nov (verification 3); S47 verified as built 6 Nov (verification 3); Rahul's three answers (6 Nov) built 6 Nov; test report 5 all passed 6 Nov; **S47 verified 6 Nov (verification 4)**; **S48 verified 6 Nov** (verification 1); **S49 tested 8 Nov** (test report 2 all passed); verification 1 needed fixes (MUST's BMI and weight from one source; VTE's three questions not asked of men), done; test report 3 all passed (13 Nov); verification 2 needed fixes (VTE's Confirm with a bleeding item empty; Rahul's two answers), done 13 Nov; test report 4 all passed; verification 3 needed one fix (a typed MUST weight kept the pre-filled BMI), done 13 Nov; test report 5 all passed (13 Nov); verification 4 needed fixes (on MUST, the hospital's own number typed back after the other was cleared again; a confirmed card's pair), done 13 Nov; test report 6 all passed (13 Nov); verification 5 needed one fix (on MUST, a BMI typed and typed back, then a weight typed, keeps the old BMI) and Rahul's answer, done 13 Nov; test report 7: 11 of 12 (F-1 minor: a MUST box marked Missing still showed its old number), fixed 13 Nov; test report 8 all passed (14 Nov); verification 6 needed one fix (on MUST, a box typed in earlier kept counting once the other was put back), done 14 Nov; test report 9 all passed (14 Nov); **S49 verified 14 Nov (verification 7)**; follow-ups (Rahul's answers, 14 Nov) in [S49a](../sessions/S49a-c7-followups/brief.md), before C8: **S49a tested 14 Nov** (test report 1 all passed); verification 1 needed fixes, done 14 Nov; test report 2 all passed (14 Nov); **S49a verified 14 Nov (verification 2)** |
| C8 | [Files](c8-files.md) | Files both ways, on the episode and the POA Summary | **Split 15 Nov 2026** into S50–S53 ([sessions](../sessions/README.md)); S50 next |
| C9 | [Documents and PROMs](c9-documents-proms.md) | Document templates and PDF; PROMs plans, sends and results; licensed instruments as opt-ins | Not started |
| C10 | [Go-live](c10-go-live.md) | Move to RDS, the Training and Production stages, compliance, pen test | Not started |

**Work sessions:** F1–F4 and C1–C2 are split into sessions S01–S17, run as build → manual test → verify, by hand or automatically with `/run-sessions`. See [../sessions/README.md](../sessions/README.md). C3 is split into S18–S22 (20 Oct), C4 into S23–S26 (21 Oct), C5 into S31–S36 (25 Oct), and C6 into S38–S45 (26 Oct; S44a–S44c added 27–28 Oct).

**Local first (12 Oct, A-22):** no AWS until the first demo or customer.
- **Now:** F2, the local half of F3 (S05), C1 without Cognito (S08–S11), and C2 without AWS (S13–S16).
- **Later, in the AWS phase:** the Lambda half of F3 and F4 (S06, S07), Cognito (S12), and jobs and messaging on AWS (S17).

The table keeps the plan order. The sessions index has the run order.

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
