# S41: The evaluate service, run locally

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 4 of 8.
**Requirements:** D-1 ([saas-requirements.md](../../saas-requirements.md) §0, §10), D-4 (§13), OPS-07; A-22 (local first, 12 Oct).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.* The Lambda itself is S45 (AWS phase).

**Depends on:** S40 verified.

**What this session does:** the server works out the results itself when an HQ is sent, and stores them. **It acts on nothing yet** (auto-triage is S42).

**What exists today:**
- **Go never evaluates SurveyJS** (CLAUDE.md's key rule). The one agreed exception (D-1, 2 Oct) is a small Node `evaluate` function running `packages/clinical`, called by Go on submit.
- **Submit:** `submitForm` in `internal/httpapi/patient.go` (the patient's `POST /api/p/episodes/{eid}/submit` and staff's on-behalf `…/patient-form/submit`, both audited) freezes the patient's answers and moves the episode to Ready for review, through `episode.Check` and `recordStatus`. Go's only own check is `uncheckedTicks` (S34).
- **External services are Go interfaces with recording fakes** in tests (`Deps` in `internal/httpapi/router.go`: Cognito, SMS, Email, Clock; its comment says "Later plans add S3 and evaluate"). Settings come only from `internal/config`.
- **Publish:** the browser runs `publishProblems` (`packages/clinical/src/publish.ts`: logic, summaries, videos, test cases); Go's `publishQuestionnaire` (`questionnaires.go`) checks only question keys, so the API alone can publish a version whose test cases fail (S37's notes).
- `make dev` starts the Go API and Vite. `packages/clinical` has no Node entry point; its Vitest tests already run SurveyJS in Node.
- AWS sends an alarm on log lines starting with a marker (`DELIVERY_FAILED`, OPS-05, `infra/api_stack.py`); S45 copies that for this service.

**Ask Rahul first:**
1. **Pushing to GitHub while the checker runs only on your Mac.** After this session the server needs the checker when a patient sends an HQ. QA on AWS won't have it until S45 (which runs with you).
   - (a) **Don't push from S41 until S45 is done.** Run S38–S44 automatically, then S45 with you, then push.
   - (b) Run S45 straight after S41, with you, then carry on.
   - (c) Push anyway: on QA every send goes to "Ready for review" with an error alert, and J11 is skipped on QA.

   **Recommended: (a).** The automatic run isn't interrupted, and QA never runs half-built.

## Build

- **The handler, in TypeScript** (explain it for Rahul): one pure function `evaluate(input)` in `packages/clinical` (unit-tested with Vitest), and a tiny Node program (e.g. `apps/evaluate/`, built with esbuild) that serves it.
  - **Input:** the published version (every Question Set's JSON with its audience, and the rules), the patient's answers per set, and the patient context (age, sex, the procedure). Go sends everything; the service has no database and no state.
  - **Output:** the shown Question Sets; each one's disclosures, scores and bands; the suggested ASA with reasons (S39); red flags; the rules that fire and their actions (S40); whether each shown patient set is complete (for S44); and the evaluator's version. The same result the clinician's view shows for the patient's answers before any validation, built with S38's helper.
  - **A second operation, `publishProblems`**, runs `publish.ts` on a version's sets and rules.
  - **It never logs answers.** An error log names only the episode form's ID.
- **Run locally:** `npm run evaluate` serves both operations over HTTP on localhost (the port from config). `make dev` starts it as a third process.
- **Go, test-first:**
  - `internal/evaluate`: an `Evaluator` interface (`Evaluate`, `PublishProblems`), an HTTP client with a short timeout (`EVALUATE_URL` in `internal/config`; required under `CONFIG_STRICT`), and a **recording fake** for tests. `Deps.Evaluator` (a missing one panics, as SMS does). The integration harness (`internal/apitest`) passes the fake.
  - **A shared fixture file** (e.g. `testdata/evaluate/*.json`: an input and its expected output) that both the Go client's tests and the TypeScript handler's tests read, so the two sides stay in step (as `video_links.json` does for S33/S34).
  - **On submit** (both routes): after the answers are frozen, call the evaluator and store the result in **`form_outputs`** (next migration: the form, the result as JSON, the evaluator's version, an error if any, the time). The episode still goes to Ready for review, as today.
  - **If the call fails** (down, timeout, bad reply): the submit still succeeds for the patient; store the error row, log one line starting `EVALUATE_FAILED` (no answers in it; S45's alarm counts these), and add a General notes line "The automatic checks didn't run. Review this HQ by hand."
  - **On publish:** Go asks the evaluator for `publishProblems` and refuses (400, the problems listed as today) while any are found, so a failing test case blocks publishing through the API too (the plan's integration test). If the evaluator can't be reached, publish is refused (503, "The checks couldn't run. Try again in a minute."); nothing is published unchecked.
- **Integration tests** (fake evaluator): submit stores the result row; submit with the evaluator failing still returns 204, stores the error, writes the note and leaves Ready for review; publish refused when the fake reports a failing test case, and on evaluator failure; the existing five cases on both submit routes and publish still pass. **No new routes.**
- **E2E:** the existing journeys pass with `make dev` starting the evaluator (J3, J7, J16 send HQs).

## Not in this session

- Acting on the result: auto-triage (S42), tasks, flags and requirements (S43).
- Refusing an incomplete send, and checking Complete review (S44).
- The Lambda, its CDK and the alarm on AWS (S45).
- Running the evaluator on Complete review or on clinicians' saves.

## Manual test focus

1. `make dev` starts three processes; sending an HQ stores a result row for it (check `form_outputs` in the database): the right disclosures, suggested ASA and rules.
2. Stop the evaluator and send an HQ: the patient sees the normal "sent" screen; the episode is Ready for review with the "automatic checks didn't run" note; the log has one `EVALUATE_FAILED` line and no answers.
3. Publish a version whose test case fails, by calling the API directly: refused, with the problem named. With the evaluator stopped: refused with "try again".
4. Normal publishing, sending and validating work as before.
