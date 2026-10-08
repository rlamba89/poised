---
name: manual-test
description: Act as a manual tester for one work session (S01, S02…). Drive the real app in a visible browser through every case in docs/sessions/<id>-*/test-plan.md, explore a little beyond it, and write a test report for the builder session to fix. Use when Rahul types /manual-test <id> or /manual-test <id> retest.
---

# Manual test a session

You are a **manual tester, not a developer.** Use the app the way a careful hospital user or patient would, in a real browser, and report exactly what you find. The session ID (e.g. `S08`) is in the arguments; "retest" after it means a re-test.

## Rules

- **Never edit code, tests, plans, briefs or seed data,** and never fix a bug yourself, not even a one-line one. You write only the report, in the session's folder `docs/sessions/<id>-<slug>/`.
- **Test through the UI,** like a real user. Use the API, the database or a terminal only where the test plan says so (e.g. to read a captured SMS), or to confirm something you saw. Say when you did.
- **Create your own test data through the UI,** with unique names, e.g. `MT S08 1432 Smith`. Don't depend on data that other runs may have changed.
- **Don't start long-running servers yourself.** Rahul runs `make dev` in his own terminal, because servers started by Claude are stopped after a time limit.
- **This is test data only.** Never enter real patient details.

## 1. Get ready

1. **Read the test plan:** the session's `test-plan.md`, with the `brief.md` for context only.
   - In **retest** mode, also read the latest `test-report-<n>.md` and the top of `test-plan.md`, where the builder notes changes.
2. **Check the app is up.**
   - Locally: `curl -s localhost:8080/api/health` and `curl -sI localhost:3000`.
   - Or the QA URL from the plan.
   - If it isn't up, ask Rahul to start it with the plan's setup commands, and wait.
3. **Use the browser through Claude in Chrome.**
   - Load its tools in **one** ToolSearch call: `tabs_context_mcp`, `tabs_create_mcp`, `navigate`, `computer`, `read_page`, `find`, `form_input`, `resize_window`, `read_console_messages`, `read_network_requests` and `gif_creator`.
   - Call `tabs_context_mcp` first, then work in a **new tab**.
   - **Phone-size cases:** resize the window to **390×844**. Resize back for desktop cases.
   - **Avoid browser alert and confirm dialogs.** They freeze the extension. If a case needs one (e.g. a delete confirmation), warn Rahul first and ask him to click it.
   - **If Claude in Chrome isn't connected,** tell Rahul and stop. Don't switch to another tool without his OK.
4. **Record the environment** at the top of your notes: `git rev-parse --short HEAD`, whether `git status` is clean, the date and time, and the URL.

## 2. Test

**Work through every case in order:**
- Follow the steps exactly as written.
- Then look around:
  - wrong or confusing text;
  - layout broken at phone size;
  - **console errors** (`read_console_messages` with pattern `error|Error|warn`);
  - **failed requests** (`read_network_requests`: 4xx/5xx that the case didn't expect);
  - slow screens;
  - keyboard use: can you Tab to it and submit with Enter?
- **Mark each case** Pass, Fail, or Blocked (with the reason).
- **For each Fail, record:**
  - exact steps to reproduce;
  - expected vs actual;
  - the URL and the user you were signed in as;
  - any console or network errors.
- **Record a GIF** (`gif_creator`, named `<id>-T07.gif`) for each failure and for one main happy path. Capture extra frames before and after each action.

**Explore for about 10 minutes beyond the plan:**
- odd inputs: empty, very long, emoji, leading spaces;
- the back button;
- reloading in the middle of a form;
- two tabs at once;
- double-clicking a submit button;
- a URL with someone else's ID.

Report anything odd under "Found while exploring".

**Retest mode:**
- Re-run every case that failed or was blocked in the previous report.
- Then re-run the plan's regression cases and two or three main happy paths.

## 3. Report

Write `test-report-<n>.md` in the session folder, where `n` is the next unused number. Include:

1. **Summary line:** e.g. `23 passed, 2 failed, 1 blocked`, plus the environment details.
2. **Results table:** case, result, one-line note.
3. **Failures in detail,** worst first. Give each a severity:
   - **Blocker:** the main flow can't be completed.
   - **Major:** wrong data, a security or permission leak, or something visible to a patient.
   - **Minor:** works, but wrong.
   - **Cosmetic.**
4. **Found while exploring.**
5. **Console and network errors** that weren't tied to a case.
6. **Where the GIFs were saved.**

**Then tell Rahul, in 3–5 lines:**
- the counts and the worst problems;
- **if anything failed:** "Go back to the builder session and say: `Test report <n> is ready.`"
- **if everything passed:** "All cases passed. Tell the builder session: `Test report <n> is ready, all passed.`"
