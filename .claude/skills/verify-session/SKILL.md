---
name: verify-session
description: Independently verify a work session (S01, S02…) that has passed manual testing. Review the diff against the brief and plan, run every automated test, spot-check the riskiest functions in the browser, give a Verified or Needs-fixes verdict, and offer the commits. Use when Rahul types /verify-session <id>, or asks to verify or check a finished session.
---

# Verify a session

You are the **reviewer.** Rahul relies on your verdict before he commits a session's work. The session ID (e.g. `S08`) is in the arguments. Be independent: check what was built, and don't take the build notes or the test reports on trust.

## 1. Read

The session's folder `docs/sessions/<id>-<slug>/`:
- `brief.md`;
- `build-notes.md`;
- `test-plan.md`;
- every `test-report-<n>.md` (the last one must say all passed).

Also read the plan sections the brief points to.

## 2. Review the change

1. **Look at the whole change:** `git status` and `git diff`, plus any untracked files.
2. **Run `/code-review high`** on the working tree, and weigh its findings yourself.
3. **Check each of these, and note the file and line for any problem:**
   - **Scope:** everything in the brief is done. Nothing big is there that the brief didn't ask for.
   - **Tests first:** every new route has the five integration cases: happy path, no sign-in, wrong role, another trust's data, invalid input. Logic has unit tests. Journeys are in `e2e/`.
   - **Tenant isolation:** every patient-data query is scoped by `org_id`, and hospital-scoped users only see their hospital.
   - **Nothing hardcoded for the UK** (A-20). This should print nothing outside config, seed and test files:
     `grep -rnE "eu-west-2|Europe/London|en-GB|\+44" apps packages infra --include='*.go' --include='*.ts' --include='*.tsx' --include='*.py'`
   - **Security:**
     - no secrets or connection strings in the repo;
     - no patient data in logs or SMS/email text;
     - the patient API can't reach staff routes;
     - clinician-only content is still stripped for patients.
   - **Docs:**
     - the plan's Status section, `docs/plans/README.md` and `docs/sessions/README.md` are updated;
     - decisions are recorded in `docs/decisions.md`;
     - this check prints nothing: `grep -rIlE "lifebox-health(c)are" . --exclude-dir=node_modules --exclude-dir=.git`

## 3. Run it

1. Run `make lint test test-integration`. Once `make e2e` exists, also run it, with `make dev` running in Rahul's terminal (ask him to start it if it isn't).
2. **Spot-check 3–5 of the riskiest functions yourself** in the browser through Claude in Chrome. Pick from: permissions, another trust's data, patient-facing screens at 390×844, anything a test report found and the builder fixed. Use the same browser rules as the `manual-test` skill.

## 4. Verdict

Write `verify.md` in the session folder:
- **Verdict:** **Verified** or **Needs fixes**.
- **Findings,** worst first, with `file:line`, and why each matters.
- **Test results:** the summary lines.
- **Suggested commits:** one per plan step, each with its tests. Docs go in their own commit.

**If Verified:**
1. Set the session's row in `docs/sessions/README.md` to **Verified**. If it was the plan's last session, mark the plan **Done** in `docs/plans/README.md` with today's date.
2. Ask Rahul whether to commit, using the suggested messages. **Commit only with his yes. Never push unless he asks.**
3. Tell him the next session's prompt: `/build-session <next id>`.

**If Needs fixes:** give Rahul this line to paste into the builder session:
"Verification found problems: read `docs/sessions/<id>-<slug>/verify.md`, fix them test-first, then ask me for a retest."
