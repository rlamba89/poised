---
name: build-session
description: Build one planned work session (S01, S02…) from docs/sessions/, prove it with automated tests, write a manual test plan for a separate tester session, then fix whatever the tester reports until everything passes. Use when Rahul types /build-session <id>, or asks to build, start or continue a session such as "S08". Also use when he says a test report is ready.
---

# Build a session

You are the **builder** for one session. The session ID (e.g. `S08`) is in the arguments. If it's missing, ask.

**Your job:**
1. Build exactly what the session's brief says.
2. Prove it with automated tests.
3. Write a manual test plan for a separate **tester** session, which drives the app in a real browser.
4. Fix what the tester reports, until a report says everything passes.

Later, a **verifier** session reviews your work before Rahul commits it.

**Where things go:** each session has a folder, `docs/sessions/<id>-<slug>/`.
- `brief.md`: the scope. Rahul and the planning session wrote it. Don't change it without Rahul's yes.
- `build-notes.md`: yours. Record the answers to questions, the steps done, any differences from the plan, test results and suggested commits.
- `test-plan.md`: yours. Written for the tester (section 3).
- `test-report-<n>.md`: the tester's.
- `verify.md`: the verifier's.

## 1. Start

1. **Read:**
   - [docs/sessions/README.md](../../../docs/sessions/README.md);
   - the session's `brief.md`;
   - every plan section the brief points to;
   - [docs/decisions.md](../../../docs/decisions.md) §1, the newest entries in §2, and §5.
2. **Check the previous session's row** in `docs/sessions/README.md`. If it isn't **Verified**, stop and tell Rahul.
3. **Run `git status`.** If there are uncommitted changes that aren't from this session, ask Rahul whether to commit them first. Don't build on top of someone else's unreviewed work without saying so.
4. **Ask the brief's questions.** If the brief has an "Ask Rahul first" list, ask everything in **one** AskUserQuestion call, with your recommended option first. Record the answers with `/update-wiki` before building anything.
5. **Make a step list** from the brief with your todo tool, and keep it current.

**Resuming:** if `build-notes.md` already exists, read it and the latest test report, then carry on from where it stopped.

## 2. Build

- **Follow the testing rules** in [docs/plans/README.md](../../../docs/plans/README.md#testing-rules-apply-to-every-plan):
  - TDD: write the failing test first.
  - Every new API route gets the five integration cases: happy path, no sign-in, wrong role, another trust's data, invalid input.
  - External services are recording fakes behind Go interfaces.
  - End-to-end journeys go in `e2e/`, with patient journeys at phone size.
- **Stay in scope.** If the brief is wrong or something is missing, ask Rahul. Don't widen the scope yourself.
- **Never hardcode the UK or London** (A-20 in [docs/saas-requirements.md](../../../docs/saas-requirements.md) §13). That means no region, domain, time zone, locale or phone-country literals outside the config.
- **Keep it simple** (CLAUDE.md). Rahul reads Go, not TypeScript, so explain any TypeScript he must review in `build-notes.md`.
- **Don't commit or push.** Rahul commits after verification. In `build-notes.md`, keep a list of suggested commits, one per plan step, each with its tests.
- **When the build is done,** run `make lint test test-integration`. Once `make e2e` exists, also run it against `make dev`. All must pass. Put the summary lines in `build-notes.md`.

## 3. Write the manual test plan

Write `test-plan.md` for a tester who has **never seen the code** and will use only the browser. Include:

1. **Setup:**
   - the exact commands Rahul runs in **his own terminal** before testing. Usually that's `docker compose up -d db`, `make migrate seed`, then `make dev`, leaving `make dev` running. Use the QA URL instead for AWS sessions.
   - the URL to open;
   - the seeded users and how to sign in as each;
   - any dev-only helpers the tester needs, such as the captured-messages page or "run jobs now".
2. **Test cases.** Give each an ID (`T-01`…), the functionality it covers, numbered steps a person can follow, and the expected result. Cover **every functionality in the brief**:
   - happy paths;
   - refused paths: wrong role, another trust's or hospital's data, invalid input;
   - edge cases;
   - reload and deep links, and the back button;
   - **phone size (390×844)** for anything a patient sees.
3. **Regression:** 3–6 short cases re-checking what earlier sessions built (list them in the brief's order).
4. **Out of scope:** what the tester should not report.

Then give Rahul this, ready to paste:

> **Start the app in your own terminal first:** `docker compose up -d db && make migrate seed && make dev` (or name the QA URL for AWS sessions).
> **Then open a NEW Claude session and type:** `/manual-test <id>`

## 4. Fix loop

When Rahul says a test report is ready (`test-report-<n>.md`):

1. **Read the whole report.**
2. **For each Fail:**
   - reproduce it;
   - write a test that fails because of it (unit, integration or e2e, whichever is closest);
   - fix it;
   - re-run all the automated tests.
3. **If a reported failure is expected behaviour or out of scope,** don't change code. Explain why in `build-notes.md`, and in the next test plan if it would confuse the tester again.
4. **If a test case was itself wrong,** correct `test-plan.md`, and note the change at the top.
5. **Give Rahul the re-test prompt:** "Open a NEW Claude session and type `/manual-test <id> retest`."

Repeat until a report says every case passed.

## 5. Finish (only after a clean report)

1. Update the plan's **Status** section, and the row in `docs/plans/README.md`, e.g. "In progress: S08 tested". Mark a plan **Done** only when its last session is verified.
2. Set the session's row in `docs/sessions/README.md` to **Tested**.
3. Run `/update-wiki` for any decisions, differences from the plan, or gotchas.
4. Tell Rahul: "<id> passed manual testing (report <n>). Next: in the planning session (or a new one) type `/verify-session <id>`."
