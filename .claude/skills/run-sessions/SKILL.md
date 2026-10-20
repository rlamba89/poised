---
name: run-sessions
description: The conductor. It runs work sessions automatically, one after another. For each session it runs a builder agent, a new manual-tester agent for every test round with the reports going back to the builder, then a verifier agent and a local commit, and moves on. It stops only for Rahul's questions or a limit. Use when Rahul types /run-sessions [from] [to], or asks to run, resume or continue the sessions automatically.
---

# Run sessions automatically (the conductor)

You are the **conductor**. You never build, test, review or fix anything yourself. Your job:
- start one agent per role;
- pass messages between them through the session folders;
- keep the app running on the latest code;
- ask Rahul only when you must.

**Keep your own context small.** Agents end with a status line and a few lines of summary, and the details stay in each session's folder (`docs/sessions/<id>-<slug>/`). Don't read whole reports or diffs.

**Rahul's standing decisions (12 Oct, in [docs/decisions.md](../../../docs/decisions.md)):**
- **Commits:** the verifier commits locally to `master` after a **Verified** verdict. **Nobody pushes.**
- **Limits per session:** at most **4 test reports** in total, and at most **2 verifications**. Past either, stop and ask.
- **Local only:**
  - no AWS;
  - no real email or SMS;
  - no AWS-phase session unless Rahul names it in the arguments.

## 0. Pre-flight (once per run)

1. **Load the deferred tools** in one ToolSearch call: `select:SendMessage,PushNotification,TaskStop`. If `PushNotification` isn't there, tell Rahul at the end of the pre-flight instead.
2. **Read the run order and history:**
   - [docs/sessions/README.md](../../../docs/sessions/README.md): the run order is the "Local phase" table, then the "AWS phase" table;
   - `docs/decisions.md` §1, the newest entries in §2, and §5;
   - `docs/sessions/run-state.md`, if it exists: you're resuming.
3. **Work out the range** from the arguments (`/run-sessions [from] [to]`).
   - **Default:** from the first session in the run order that isn't **Verified**, to the last local-phase session.
   - **Never include an AWS-phase session** unless Rahul named it.
4. **Check the environment.** Don't fix anything yourself; if a check fails, ask Rahul.
   - `docker compose up -d --wait db` works.
   - `git rev-parse --abbrev-ref HEAD` is `master`.
   - `git status --short` shows only the first session's own work. If there's anything else, ask Rahul.
   - Ports 3000 and 8080 are free: `lsof -nP -iTCP:3000 -iTCP:8080 -sTCP:LISTEN`. If Rahul's own `make dev` holds them, ask him to stop it, because the conductor runs the dev server from now on.
   - Chrome: Claude in Chrome must be connected. (An agent can call `mcp__claude-in-chrome__tabs_context_mcp` with `createIfEmpty: true`.)
5. **Ask every question up front.** For each session in the range, read the "Ask Rahul first" part of its `brief.md`.
   - Skip anything already answered in that session's `answers.md` or `build-notes.md`, or in `docs/decisions.md`.
   - Ask the rest with **AskUserQuestion**:
     - up to 4 questions per call, in run order;
     - each with 2–4 options, the recommended one first and marked "(Recommended)";
     - the session ID in the header.
   - Write each answer to that session's **`answers.md`**: the question, the answer, any notes, the date, and "asked by the conductor".
   - Then run `/update-wiki`, so the answers are also in `docs/decisions.md`.
6. **Write `docs/sessions/run-state.md`** with:
   - the range;
   - the current session and stage;
   - the test round and verification count;
   - the builder agent's ID;
   - the dev server's task ID;
   - "hand checks for Rahul" (cases testers couldn't do);
   - a short log line per event, with the time.

   Update it after **every** step, because a stopped run resumes from it. The file is gitignored.
7. **Tell Rahul:** the sessions in the range, that it may take many hours, and that he'll get a notification for each question and each finished session. Then start.

## 1. One session

**Find the stage** from the session folder, so a resumed run picks up where it was:

| What's there | Stage |
| --- | --- |
| `verify.md`'s newest verdict is Verified and `git status` is clean | done: go to the next session |
| `verify.md`'s newest verdict is Needs fixes | **FIX (verification)** |
| The newest `test-report-<n>.md` has no failures, and there's no newer verdict | **VERIFY** |
| The newest test report has failures that haven't been fixed since | **FIX (report)** |
| `test-plan.md` exists, but there's no test report since the last fix | **TEST** |
| Otherwise | **BUILD** |

**BUILD**
- Start a **builder** agent (`general-purpose`, in the background) with this prompt:
  > You are the builder for session `<id>` in an automatic run. Invoke the `build-session` skill with the arguments `<id> auto`, and follow it, including its "Automatic mode" section. End your final message with one line, `STATUS: READY_FOR_TEST`, `STATUS: NEEDS_ANSWER` or `STATUS: BLOCKED <reason>`, then at most 5 lines of summary.
- Save its agent ID in `run-state.md`, and **wait for its completion notification** (never poll).
- Then act on its status:
  - **READY_FOR_TEST** → TEST.
  - **NEEDS_ANSWER** → ASK, then continue the **same** builder.
  - **BLOCKED** → STOP.

**TEST**
1. **RESTART** the app first.
2. Start a **new tester agent every round** (`general-purpose`, in the background). Never reuse a tester, so it doesn't carry assumptions. Prompt:
   > You are the manual tester for session `<id>` in an automatic run. Invoke the `manual-test` skill with the arguments `<id> auto` (or `<id> retest auto` after the first round), and follow it, including its "Automatic mode" section. End with one line, `STATUS: ALL_PASSED`, `STATUS: FAILURES <n>` or `STATUS: BLOCKED <reason>`, then the report's path.
3. Wait. Copy any "hand check for Rahul" cases from the report into `run-state.md` (read only that part).
4. Act on the status:
   - **ALL_PASSED:** continue the same builder: "Test report <n> is ready, all passed." It does its Finish steps and returns `READY_FOR_TEST`. Then → VERIFY.
   - **FAILURES:** if this was test report 4 → STOP. Otherwise → FIX (report).
   - **BLOCKED:** RESTART the app once and test again. If it's blocked again → STOP.

**FIX (report)**
- Continue the **same** builder with SendMessage: "Test report <n> is ready."
- If that agent can't be reached (e.g. after a resume), start a new builder as in BUILD. It resumes from `build-notes.md`.
- Wait, then act on its status as in BUILD. READY_FOR_TEST → TEST (a retest).

**VERIFY**
- Start a **new verifier agent** (`general-purpose`, in the background):
  > You are the verifier for session `<id>` in an automatic run. Invoke the `verify-session` skill with the arguments `<id> auto`, and follow it, including its "Automatic mode" section. End with one line, `STATUS: VERIFIED <commit hashes>`, `STATUS: NEEDS_FIXES` or `STATUS: BLOCKED <reason>`, then at most 5 lines of summary.
- Wait, then act on its status:
  - **VERIFIED:**
    1. Check that `git status --short` is clean, and `git log --oneline -5` shows the commits.
    2. Log it in `run-state.md`.
    3. Notify Rahul: "<id> verified and committed (<hashes>)."
    4. Go to the next session.
  - **NEEDS_FIXES:** if this was verification 2 → STOP. Otherwise → FIX (verification).
  - **BLOCKED** → STOP.

**FIX (verification)**
- Continue the same builder (or start a new one): "Verification found problems: read `verify.md`, fix them test-first."
- Wait. READY_FOR_TEST → TEST (a retest), then VERIFY again.

## 2. Helpers

**RESTART the app** (before every test round, so the tester sees the latest Go code; `make dev` doesn't reload Go):
1. Stop the previous dev server with TaskStop, using the task ID in `run-state.md`.
2. Check that ports 3000 and 8080 are free. If something else holds them → STOP.
3. Run `make migrate seed` in the foreground.
4. Start `make dev` with Bash, `run_in_background: true`, and save its task ID.
5. Wait for both to answer, with no `sleep`. Use `curl -sf --retry 60 --retry-delay 2 --retry-connrefused http://localhost:8080/api/health` and the same for `http://localhost:3000`.
6. If it doesn't come up → STOP, and show the last 30 lines of the dev server's output.

**ASK:**
1. Read `questions.md` in the session folder.
2. Notify Rahul: "<id> needs an answer."
3. Ask with AskUserQuestion (recommendation first).
4. Append the answer to `answers.md`.
5. Continue the same builder: "Answers are in `answers.md`."

**STOP** (when a limit is hit, something is BLOCKED, the environment breaks, git has unexpected changes, or anything would need a push, AWS, real messages or a paid service):
1. Write the reason in `run-state.md`.
2. Notify Rahul.
3. Ask with AskUserQuestion, with the options that fit, e.g.:
   - "One more round for the builder";
   - "Verify anyway";
   - "Skip to the next session" (it stays unverified and uncommitted, so recommend against it);
   - "Stop the run".
4. Do what he chooses.

**If Rahul types while the run is going** (e.g. "pause after the current agent" or "pause after this session"): finish the agent that's running, then do what he asked, and note it in `run-state.md`.

**Whenever the run pauses or stops** (Rahul asked, STOP, or the end of the range):
1. Stop the dev server.
2. Put a **"Next by hand:"** line at the top of `run-state.md`, with the exact command Rahul would type to carry on himself, e.g. `/manual-test S09 retest`, `/build-session S09` or `/verify-session S09`. Add one line on what that step is for.

Rahul may take over by hand and later hand back with `/run-sessions`. The stage table in section 1 works out where things are from the folders, so it never relies on your memory.

## 3. Rules

- **One agent at a time.** There's one Chrome and one dev server. Agents run in the background, and you wait for their completion notification.
- **You never edit code, tests, briefs or plans,** and you never commit. You may:
  - run `make`, `docker`, `curl`, `lsof` and read-only `git` commands;
  - start and stop the dev server;
  - write `run-state.md`, `answers.md` and the decisions recorded by `/update-wiki`.
- **Never push. Never deploy. Never send a real message.**
- **Don't widen a session.** A builder asking to change its brief is a question for Rahul.
- **Before starting a session,** check that the session before it in the run order is Verified.

## 4. End of the run

When the range is done, or Rahul stops it:
1. Stop the dev server, and write the "Next by hand:" line (section 2).
2. Notify Rahul.
3. Give a short summary:
   - each session's result, with its commit hashes;
   - the questions that were asked;
   - the **hand checks for Rahul**, collected from the reports;
   - anything logged in `docs/pending.md`;
   - what comes next. After the local phase, that's the AWS phase, which needs Rahul there.
