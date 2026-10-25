# Work sessions

Plans F1–F4 and C1–C2 split into **17 sessions** (plus S02a, a bug fix). Together they deliver the six platform features:
- superadmin;
- hospital onboarding;
- staff and patient invites;
- permissions;
- staff in several hospitals;
- patients in several hospitals.

Each session is small enough for one Claude session, and ends with something a tester can try in a browser.

Plan C3 is split into **S18–S22** (20 Oct), and plan C4 into **S23–S26** (21 Oct). Plans C5–C10 get split the same way when we reach them.

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

## Automatic runs: `/run-sessions`

**One Claude session (the conductor) runs the whole loop for you.** It takes each session in the run order below through four stages:
1. a **builder** agent;
2. a **new tester agent for every test round**, with the report going back to the same builder until everything passes;
3. a **verifier** agent;
4. a local commit to `master`.

Then it moves on to the next session. It stops only for your questions or a limit, and sends a phone notification when it does.

**Start it:**
1. Stop your own `make dev`, because the conductor runs and restarts it itself.
2. Make sure Docker is running and Chrome is open, with Claude in Chrome connected.
3. Run `caffeinate -dis` in a terminal, so the Mac doesn't sleep.
4. Open a new Claude session in the repo, in a permission mode that lets it edit files and run `make`, `go`, `npm`, `docker` and `git`. Otherwise permission prompts stall the run.
5. Type `/run-sessions`. That runs from the first session that isn't Verified to the end of the local phase. Or give a range, e.g. `/run-sessions S08 S11`.
6. Answer the questions it asks at the start. Every brief's "Ask Rahul first" items are asked then, in one sitting.

**What it may and may not do** (agreed 12 Oct):
- It **commits locally to `master`** after each Verified verdict. **It never pushes.**
- **Limits:** at most **4 test rounds** and **2 verifications** per session. Past either, it stops and asks you.
- It never touches AWS, and never sends real email or SMS.
- It never runs an AWS-phase session unless you name it.

**Resume:** if a run stops (a closed laptop, a crash, or after a question), type `/run-sessions` again in a new session. It reads `run-state.md` and each session's folder, and carries on from where it was.

**The manual way still works.** The table below is for running a session by hand.

## What you type (by hand)

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
- **During testing by hand, `make dev` runs in your own terminal.** Servers that Claude starts are stopped after a time limit. (An automatic run restarts it before each test round instead.)
- **Nothing is committed until it's verified.** By hand, only with your yes. In an automatic run, with your standing yes (local commits only, never a push).
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
   - SES production access and UK SMS sender registration, before S17. Both are in the AWS phase.

## Sessions

**Status values:** Not started → Building → Testing → Tested → **Verified**.

### Done

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S01](S01-integration-harness/brief.md) | F1 | Integration test harness, plus tests for the authoring routes | Docker running | Verified |
| [S02](S02-workflow-route-tests/brief.md) | F1 | Integration tests for the episode, patient and clinician routes | — | Verified |
| [S02a](S02a-answer-revisions/brief.md) | F1 | Bug fix: a stale save of clinician or patient answers gets a 409, not a silent overwrite (S02 report 2, X-1); an episode PATCH can't put back a status that just changed | Answered 9 Oct | Verified (11 Oct) |
| [S03](S03-e2e-journeys/brief.md) | F1 | Playwright journeys J1–J5 and `make e2e` | Python 3.12+ | Verified (12 Oct) |

### Local phase: now, run in this order

Everything runs on your Mac. Email and SMS are captured, not sent; the dev login stands in for Cognito; MinIO will stand in for S3 (C8). No AWS (decided 12 Oct).

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S04](S04-react-vite/brief.md) | F2 | Next.js → React + Vite + React Router | Answered 12 Oct: rename to Poised | Verified (12 Oct) |
| [S05](S05-lambda-entrypoints/brief.md) | F3 | `internal/config` (per-country values), the staff and patient routers split, `GET /api/config`; local only | — | Verified (12 Oct) |
| [S08](S08-trusts-permissions/brief.md) | C1 | Trusts, hospitals, memberships, role ladder, hospital picker | Answer the brief's questions (incl. the `dev` branch) | Verified (13 Oct) |
| [S09](S09-staff-invites/brief.md) | C1 | Staff list, invites, change role, remove access | — | Verified (13 Oct) |
| [S10](S10-superadmin-console/brief.md) | C1 | Superadmin console: trusts, hospitals, suspend, opt-ins, settings | Answer the brief's questions | Verified (13 Oct) |
| [S11](S11-audit-log/brief.md) | C1 | Audit log and CSV export | Answered 12 Oct | Verified (13 Oct) |
| [S13](S13-patient-records/brief.md) | C2 | Patient records, NHS number check, search, episodes per hospital | Answered 12 Oct | Verified (13 Oct) |
| [S14](S14-patient-invites/brief.md) | C2 | Invites by SMS and email, magic link + date of birth, submit | Answered 12 Oct | Verified (13 Oct, verification 3; X-1 skipped by Rahul, in pending.md) |
| [S15](S15-patient-home/brief.md) | C2 | Patient accounts across trusts, sign-in link, patient home | Answered 12 Oct (PAT-10 built now) | Verified (13 Oct, verification 2; the unjoined-twin sign-in is an accepted risk; R-6 a hand check) |
| [S16](S16-jobs-reminders/brief.md) | C2 | Jobs, reminders and "not responding", run by a local timer; messages captured | Answered 12 Oct (any time; starting stops reminders); 13 Oct answers 3–5 | Verified (13 Oct, verification 1; minor follow-ups in pending.md) |

### AWS phase: later, before the first demo or customer, with you there

These need your AWS setup, GitHub pushes and real messages, so they don't run unattended.

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S06](S06-aws-qa-stack/brief.md) | F3 + F4 | The Lambda adapter, the CDK app, `infra/countries.py`, the QA environment on AWS | **AWS Step 0 done** | **Verified** (14 Oct, [verify.md](S06-aws-qa-stack/verify.md), verification 1; test report 1: 19 of 19 passed; QA: https://main.d33kh0gi8vk83v.amplifyapp.com) |
| [S07](S07-pipeline/brief.md) | F4 | CodePipeline with test gates, the "add a country" runbook | Push access for test pushes | **Verified** (14 Oct, [verify.md](S07-pipeline/verify.md), verification 1; test report 1: 17 of 17 passed; pipeline live, runs R1–R4 as planned, proof done) |
| [S12](S12-cognito-signin/brief.md) | C1 | Real staff sign-in with Cognito + MFA, on QA | Answered 14 Oct (Essentials: checked in the docs) | **Verified** (14 Oct, [verify.md](S12-cognito-signin/verify.md), verification 1; test report 2 all passed; hand checks T-08, T-09, R-05, H-1–H-3; deployed to QA by hand, the pipeline takes over at the next push) |
| [S17](S17-aws-messaging-jobs/brief.md) | C2 | Jobs Lambda + EventBridge schedule, SES and SMS on AWS, real sending switched on | **SES/SMS approvals** | **Verified** (15 Oct, [verify.md](S17-aws-messaging-jobs/verify.md), verification 1; test report 2 all passed, run locally as Rahul chose; report 1 on QA blocked at Cognito's sign-in, its AWS checks passed; a real email and a real text from POISED reached Rahul; hand checks in verify.md; deployed to QA by hand, the pipeline takes over at the next push) |

### Plan C3: episode lifecycle (local, run in this order)

Split on 20 Oct. These run locally like the local phase above, with messages captured. A push afterwards deploys them to QA through the pipeline.

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S18](S18-episode-forms/brief.md) | C3 | `episode_forms`: answers move from the episode to its HQ form. No visible change; a full regression | Answered 20 Oct ([answers.md](S18-episode-forms/answers.md)) | **Verified** (20 Oct, [verify.md](S18-episode-forms/verify.md)) |
| [S19](S19-procedures/brief.md) | C3 | The trust's procedure list (PRC-01); procedure, planned date and POA date on the episode; the procedure picks the default HQ | Answered 20 Oct ([answers.md](S19-procedures/answers.md)) | **Verified** (21 Oct, [verify.md](S19-procedures/verify.md)) |
| [S20](S20-statuses/brief.md) | C3 | The transition table, every status (admitted, discharged, cancelled), status events, cancel, re-open, the episode revision; J9b | Answered 20 Oct ([answers.md](S20-statuses/answers.md)) | **Verified** (21 Oct, [verify.md](S20-statuses/verify.md)) |
| [S21](S21-tasks-worklists/brief.md) | C3 | Episode tasks (TSK-01), worklists on the clinician home (EPI-02), status messages to the patient (EPI-08) | Answered 20 Oct ([answers.md](S21-tasks-worklists/answers.md)) | **Verified** (21 Oct, [verify.md](S21-tasks-worklists/verify.md)) |
| [S22](S22-archive/brief.md) | C3 | Archive and un-archive, the daily auto-archive job (EPI-05), a patient closing their account (EPI-09); J9 | Answered 20 Oct ([answers.md](S22-archive/answers.md)) | **Verified** (21 Oct, [verify.md](S22-archive/verify.md)) |

### Plan C4: question keys and pre-fill (local, run in this order)

Split on 21 Oct. Local like C3, with messages captured.

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S23](S23-question-keys/brief.md) | C4 | Question keys in the editor: unique across an HQ, option values locked at publish (Go), "Always ask fresh", the "Pre-fills from N other HQs" label (PRE-05) | Answered 22 Oct ([answers.md](S23-question-keys/answers.md)) | **Verified** (22 Oct, [verification 3](S23-question-keys/verify.md); test report 5 all passed) |
| [S24](S24-question-library/brief.md) | C4 | The question library (platform starter set + the trust's own), Insert from library keeping the key, "Same as…" with `keys.Compatible`; PNL-06 changed | Answered 22 Oct ([answers.md](S24-question-library/answers.md)) | **Verified** (23 Oct, [verification 3](S24-question-library/verify.md); test report 4 all passed) |
| [S25](S25-prefill/brief.md) | C4 | Pre-fill from the same trust's earlier HQs (PRE-01, PRE-06), the opening card with its two buttons (PRE-02), the markers (PRE-03); J16's patient half | Answered 22 Oct ([answers.md](S25-prefill/answers.md)) | **Verified** (23 Oct, [verification 1](S25-prefill/verify.md); test report 2 all passed) |
| [S26](S26-prefill-clinician-cross-trust/brief.md) | C4 | The clinician's pre-fill markers (PRE-04), pre-fill from another trust with both settings on and the patient's agreement; J16 completed, J16b | Answered 22 Oct ([answers.md](S26-prefill-clinician-cross-trust/answers.md)) | **Verified** (23 Oct, [verification 2](S26-prefill-clinician-cross-trust/verify.md); [test report 3](S26-prefill-clinician-cross-trust/test-report-3.md) all passed after verification 1's fixes and Rahul's library-key rule) |

### After plan C4 (local)

Rahul's 24 Oct decisions after plan C4, before plan C5 (S27; then S28 for S27's open items).

| ID | Plan | Builds | Needs from you first | Status |
| --- | --- | --- | --- | --- |
| [S27](S27-c4-followups/brief.md) | — | Pre-fill from other trusts without asking (no trust setting; the card names the hospital, with Start fresh), the archived read-only View and badge, neutral Cancelled wording, format names in the publish refusal, clickable answers on the POA Summary | Answered 24 Oct ([answers.md](S27-c4-followups/answers.md)) | **Verified** (24 Oct, [verification 3](S27-c4-followups/verify.md)) |
| [S28](S28-s27-followups/brief.md) | — | S27's open items as Rahul answered them (24 Oct): the four-field join on the patient's behalf, another trust's audit names only the trust, other trusts read on the first open only, Start fresh on the POA Summary, the archived View's wording, the cancelled-after-sent home card, NOT RESPONDING hidden once cancelled, the reported BMI before validation; plus S27's small fixes | Answered 25 Oct ([answers.md](S28-s27-followups/answers.md)) | **Verified** (25 Oct, [verification 1](S28-s27-followups/verify.md)) |

**Why this order:** nothing needs AWS until someone outside your laptop has to use the app. Building locally first lets the automatic run go without AWS stops. The cost is that the AWS hosting setup and its near-£0 idle cost are proven later (decided 12 Oct; [decisions.md](../decisions.md)).
