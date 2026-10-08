# Work sessions

Plans F1–F4 and C1–C2 split into **16 sessions**. Together they deliver the six platform features:
- superadmin;
- hospital onboarding;
- staff and patient invites;
- permissions;
- staff in several hospitals;
- patients in several hospitals.

Each session is small enough for one Claude session, and ends with something a tester can try in a browser. Plans C3–C10 get split the same way when we reach them.

## How a session runs

There are three roles, each in its own Claude session:

| Role | Skill | Does |
| --- | --- | --- |
| **Builder** | `/build-session <id>` | Asks the brief's questions, builds test-first, writes `test-plan.md`, then fixes whatever the tester reports. |
| **Tester** | `/manual-test <id>` | A **new** session each time. It drives the real app in Chrome like a manual tester, never touches code, and writes `test-report-<n>.md`. |
| **Verifier** | `/verify-session <id>` | Reviews the diff, runs every test, spot-checks in the browser, writes `verify.md`, and offers the commits. |

```
you: /build-session S08 ─► builder: questions → build (TDD) → test-plan.md
                           "start make dev, then /manual-test S08 in a new session"
you: /manual-test S08 ───► tester: browser → test-report-1.md → "tell the builder"
you (in builder): "Test report 1 is ready." ─► builder: fixes test-first
                           "/manual-test S08 retest in a new session"
        … repeat until a report says everything passed …
you: /verify-session S08 ─► verifier: review + tests + spot-check → verify.md → "commit?"
```

## What you type

| When | Where | Type |
| --- | --- | --- |
| Start a session | a new Claude session | `/build-session S01` |
| The builder says the test plan is ready | start `make dev` in **your own terminal**, then open a new Claude session | `/manual-test S01` |
| The tester has finished | the **builder** session | `Test report 1 is ready.` |
| The builder has fixed things | a new Claude session | `/manual-test S01 retest` |
| The builder says it's tested | the planning session, or a new one | `/verify-session S01` |
| The verifier asks to commit | the verifier session | yes or no |

## Rules

- **One session at a time, in order.** A session starts only when the one before it is **Verified and committed**, so unreviewed changes never pile up.
- **A fresh Claude session for every builder and every test round.** Testers must not carry the builder's assumptions.
- **During testing, `make dev` runs in your own terminal.** Servers that Claude starts are stopped after a time limit.
- **Nothing is committed until it's verified,** and only with your yes.
- **A brief changes only with your yes.** Builders ask rather than widen the scope.
- **Each session's files live in its folder:**
  - `brief.md` (the scope);
  - `build-notes.md` (the builder's notes);
  - `test-plan.md`;
  - `test-report-<n>.md`;
  - `verify.md`.

## Before S01

1. **Commit the planning docs:** this folder, `.claude/skills/`, and the 8 Oct decisions. Then S01's diff holds only S01's work.
2. **Set up Chrome** with the Claude in Chrome extension connected. The tester and verifier use it.
3. **Start the paperwork early,** because it takes days:
   - your one-off AWS setup, before S06 (F4 Step 0, about 30 minutes);
   - SES production access and UK SMS sender registration, before S16 (C2).

## Sessions

**Status values:** Not started → Building → Testing → Tested → **Verified**.

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S01](S01-integration-harness/brief.md) | F1 | Integration test harness, plus tests for the authoring routes | Docker running | Not started |
| [S02](S02-workflow-route-tests/brief.md) | F1 | Integration tests for the episode, patient and clinician routes | — | Not started |
| [S03](S03-e2e-journeys/brief.md) | F1 | Playwright journeys J1–J5 and `make e2e` | Python 3.12+ | Not started |
| [S04](S04-react-vite/brief.md) | F2 | Next.js → React + Vite + React Router | Answer: rename the UI to Poised now? | Not started |
| [S05](S05-lambda-entrypoints/brief.md) | F3 | Lambda entry points, `internal/config`, `GET /api/config` | — | Not started |
| [S06](S06-aws-qa-stack/brief.md) | F4 | CDK app, `infra/countries.py`, the QA environment on AWS | **AWS Step 0 done** | Not started |
| [S07](S07-pipeline/brief.md) | F4 | CodePipeline with test gates, the "add a country" runbook | Push access for test pushes | Not started |
| [S08](S08-trusts-permissions/brief.md) | C1 | Trusts, hospitals, memberships, role ladder, hospital picker | Answer the brief's questions | Not started |
| [S09](S09-staff-invites/brief.md) | C1 | Staff list, invites, change role, remove access | — | Not started |
| [S10](S10-superadmin-console/brief.md) | C1 | Superadmin console: trusts, hospitals, suspend, opt-ins, settings | Answer the brief's questions | Not started |
| [S11](S11-audit-log/brief.md) | C1 | Audit log and CSV export | Answer the brief's questions | Not started |
| [S12](S12-cognito-signin/brief.md) | C1 | Real staff sign-in with Cognito + MFA, on QA | Cognito plan choice | Not started |
| [S13](S13-patient-records/brief.md) | C2 | Patient records, NHS number check, search, episodes per hospital | Answer the brief's questions | Not started |
| [S14](S14-patient-invites/brief.md) | C2 | Invites by SMS and email, magic link + date of birth, submit | — | Not started |
| [S15](S15-patient-home/brief.md) | C2 | Patient accounts across trusts, sign-in link, patient home | Answer the brief's questions | Not started |
| [S16](S16-jobs-reminders/brief.md) | C2 | Jobs, reminders and "not responding", messaging on AWS | SES/SMS paperwork started | Not started |

**Order:** this is the roadmap order: AWS (S05–S07) before the features. To get the features working locally first, run S01–S04, S08–S11, S13–S15, then S05–S07, S12 and S16. Tell the builder if you choose that.
