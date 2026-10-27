# S41 build notes: the evaluate service, run locally

Builder, automatic run, 26 Oct 2026. Rahul's answer ([answers.md](answers.md)): **push only after S45** (nothing is pushed from this session; the conductor commits locally only).

**What it does:** when an HQ is sent (by the patient, or by staff for them), Go asks a small Node service what the answers give and stores the reply in `form_outputs`. **Nothing acts on it yet** (S42). Publishing asks the same service for the browser's publish checks.

## What was built

### `packages/clinical` (TypeScript for Rahul to review)

**`src/evaluate.ts` (new)**: the service's two operations, as pure functions (no database, no state, no logging):
- **`evaluate(input, version)`**. `input` is what Go sends: `formId`, `sets` (every Question Set of the published version, in order, each `{id, name, audience, content}`, the content whole), `rules` (the version's stored ASA rules, or null), `answers` (the patient's, by set id), `patient` (`{age, sex}`: no name or other details) and `procedureId`. It:
  1. keeps only the answers of the **patient's** sets (a clinician set's answers never count, as S40's rules say);
  2. runs `hqResult` on every set **in the clinician's view** (the same call S40's `hqRuleResult` and the Rules page use, built on S38's `hqSets`/`earlierContexts`), so a clinician-only score worked out from the patient's answers (STOP-Bang) counts;
  3. runs `firedRules` on that result with the procedure;
  4. lists every red and amber flag in words ("Do you get chest pain?: Yes", as test cases write flags);
  5. in the **patient's own view** (`hqSets(patientSets, …, "patient")` as the patient's form does), lists the sets the patient's form shows and whether each is complete (`isSetComplete`), for S44.
  It returns `{evaluator, shown, sets: [{id, outputs, scores}], suggestedAsa (or null), flags, rules: [{id, name, actions, held, line}], patientSets: [{id, complete}]}`. An input of the wrong shape throws `EvaluateInputError` (the service answers 400); its message never quotes an answer.
- **`checkPublish({sets, rules})`**: `publishProblems(sets, ruleSetOf(rules))` with **each set's audience passed** (pending.md, S39/S40: a missing audience counts as a patient set, which would refuse a valid clinician set testing the suggested ASA). Returns `{problems: [{set, message}]}`.

**Changed:** `respondent.ts` `hqResult` also returns `sets` (each shown set's own outputs and scores), so the result can be stored set by set. Nothing else changes for its callers.

**Tests:** `evaluate.test.ts` (16): the whole result equals the **shared fixture**'s expected output; the shown sets (a hidden set isn't there); each set's disclosures and scores with bands; a clinician set's answer (ASA IV) never counts; the suggested ASA with its reason; red and amber flags; the rules that fire, in order, with actions and lines; the patient's sets' completeness (one incomplete); no rules → none fire; a procedure rule; age and sex points from the patient context; bad inputs refused; `checkPublish` against every case of the publish fixture.

### `apps/evaluate` (new; TypeScript for Rahul to review)

A tiny Node program, ~100 lines:
- `src/handler.ts`: `handle(method, path, body, version)`: `POST /evaluate` → `evaluate()`, `POST /publish-problems` → `checkPublish()`, `GET /health` → `{ok, evaluator}`; 400 for a body that isn't JSON or isn't the right shape, 404/405 otherwise. When the work throws, it logs **one line naming only the form's id, the error's kind and code locations** (not the error's message, which could quote a value) and answers 500. S45's Lambda will call `handle()` too.
- `src/server.ts`: `node:http` on **127.0.0.1 only**, body limit 10 MB (413).
- `src/main.ts`: the port is **EVALUATE_URL's port** (default 8090), so one setting serves both Go and the service.
- `build.mjs`: **esbuild** bundles it into `dist/evaluate.cjs` (one file, ~2.6 MB with SurveyJS); the evaluator's version is `clinical@<git commit>` plus `-dirty` when `packages/clinical` or `apps/evaluate` has uncommitted changes.
- **`npm run evaluate`** (root `package.json`) builds and runs it. **`make dev` starts it as a third process**, so the conductor's restart picks it up; `make evaluate` runs it alone.
- Tests (`src/server.test.ts`, Vitest, 10): over real HTTP on a random port: `/evaluate` gives the shared fixture's output; `/publish-problems` each fixture case; `/health`; 400s; 404/405; 413; the error log names the form and no answer; the port from EVALUATE_URL.
- New dev dependency: **esbuild 0.28.2** (Aug 2026); `@types/node` as the web app's.

### The shared fixtures

`apps/api/internal/evaluate/testdata/evaluate.json` (an input and its expected output: two patient sets shown, one hidden, a clinician set; ASA II + III; a red and an amber flag; a score band; the default rules plus a band rule; one incomplete set) and `publish_problems.json` (a failing whole-HQ test case; a passing one; a pre-S40 version with no rules). Go's tests, `packages/clinical`'s and `apps/evaluate`'s all read them, as `video_links.json` does for S33/S34.

### Go

- **`internal/evaluate` (new):** the `Evaluator` interface (`Evaluate`, `PublishProblems`); the types (`Input`, `Result` with the fields Go will read in S42–S44 plus `Raw`, the whole reply, which is what's stored; `PublishInput`, `Problem`); **`HTTP`**, the client (timeouts **5 s** for Evaluate, which a patient waits on, and **20 s** for the publish checks, which run every test case; 20 MB reply limit; a reply with no `evaluator` is refused; **no error ever holds the reply or an answer**); **`Recorder`**, the fake (records calls; `Fail`, `Reply`, `SetProblems`; by default every set shown, nothing found). `FailedMarker = "EVALUATE_FAILED"`.
- **`internal/config`:** `EVALUATE_URL` (default `http://127.0.0.1:8090`; **required with CONFIG_STRICT**; an http(s) address with no path).
- **`Deps.Evaluator`** (a missing one panics, as SMS does); `serve.Setup` gives `evaluate.HTTP{URL: cfg.EvaluateURL}`; `internal/apitest` gives a `Recorder` (`Client.Evaluator`).
- **Migration 00032 `form_outputs`**: `id`, `episode_form_id`, `result` jsonb, `evaluator_version`, `error`, `created_at`; result xor error. One row per send (a re-opened form sent again gets another).
- **On send** (`submitForm`, both routes): after the answers are frozen and the status recorded, in the same transaction, `evaluateSent` sends every set (whole, with its audience), the version's rules, the patient's answers, their **age today at the hospital** (`ageOn`, the hospital's time zone) and sex, and the procedure; stores the reply. **On failure:** the send still succeeds (204, Ready for review); the error row is stored; one log line `EVALUATE_FAILED form <id>: <error>`; and a General notes line "The automatic checks didn't run. Review this HQ by hand." (an automatic event, no user). `GetEpisodeForm` now also gives the episode's procedure and the version's rules.
- **On publish:** inside the publish transaction, after the key checks are worked out, Go asks for `PublishProblems` (every set with its audience, and the version's rules as stored now). Its problems come first, then the key problems, in the **same 422 refusal as today** (`{error, problems}`), which the Publish window already lists. If the service can't be reached: **503 "The checks couldn't run. Try again in a minute."** and a log line `EVALUATE_FAILED publish checks, version <id>: …`.
- **Tests:** `internal/evaluate` (unit): the fixture's input round-trips Go's `Input` and `PublishInput` (so every field the TS side reads survives Go's types); the HTTP client against a server answering the fixture (request, decoding, `Raw` kept whole); failures (500, not JSON, no evaluator, nothing listening, a timeout, publish 502) with no reply text in the error; the Recorder. `internal/config`: the default, set, strict-required, bad values. **Integration** (`evaluate_integration_test.go`): `TestSubmitStoresWhatTheAnswersGive` (the input: every set with audience and clinician-only content, only the patient's answers, the rules, age and sex, the procedure; the row stored with its version; the status and General notes unchanged; a refused second send doesn't call it; staff on behalf the same); `TestSubmitWhenTheEvaluatorFails` (204, Ready for review, the error row, the General notes line, exactly one `EVALUATE_FAILED` line naming the form with no answer in the log; on behalf the same); `TestPublishAsksTheEvaluator` (sets with audiences, the rules, null without); `TestPublishRefusedByTheEvaluator` (a failing test case → 422 with it named, still a draft; listed with key problems; unreachable → 503, still a draft, then publishes). The existing five cases on both submit routes and publish pass unchanged (no new routes).

## Test report 1 (26–27 Oct): 15 of 17 passed; F-1 fixed

- **F-1 (Minor, T-09/T-10):** the "The automatic checks didn't run. Review this HQ by hand." line was stored as an ordinary event, so the General notes showed it as the patient's ("| Patient"), even when Cara sent it. **Fixed test-first:** `TestSubmitWhenTheEvaluatorFails` now checks the line is `automatic` with no user, on both routes (it failed, 0 of 1); `evaluateSent` writes it with a new query `AddAutomaticEpisodeEventNow` (automatic, at the transaction's time, so it always sits right after the status event the same send wrote; the jobs' `AddAutomaticEpisodeEvent` takes the run's clock time, which in tests is earlier and would put it first). It now shows "Automatic", like a reminder's line.
- Not changed (exploring): the editor's Save output closing with the code list open, and the PATIENT badge on a clinician set's question, are not S41's (the badge is already in pending.md from S40).
- After the fix: `make lint`, `make test` and `make test-integration` pass.

## Test report 2 (27 Oct): all passed

T-09 and T-10 (F-1's retest), the regression cases and the main happy paths passed. Its exploring notes, both as designed (no change, nothing for pending.md):
- **Nothing of the stored result reaches the patient** (`GET /api/p/episodes/{eid}` has no result, flags or error): expected; only staff will see it, from S42.
- **"HQ completed by the patient" and "The automatic checks didn't run…" share a `created_at`** (one transaction): their order comes from `episode_events.seq`, which `ListEpisodeEvents` orders by after the time ("notes written at the same time, in the order written", S22's migration 00023). That's the designed tie-break, not luck.
- (Not S41) The on-behalf form's clinician-only text box looks greyed out before it's clicked; typing works.

## Decided by the builder (automatic mode)

1. **Publish's refusal is 422, not the brief's 400**: "the problems listed as today" is today's 422 `{error, problems}` from the key checks, which the Publish window already shows; one refusal lists both kinds.
2. **The evaluator is called inside the send's transaction**, so the frozen answers, the status, the result row and the failure note are written together (S42 will auto-triage in the same place). The 5 s timeout bounds how long the form's row is held.
3. **Flags include amber as well as red** (the brief says red flags; S42's safety check needs "no amber/red flags", Rahul's C6 answer).
4. **Each fired rule carries its test-case line** (`line`), so S42's event and a person reading `form_outputs` see it in words.
5. **The patient's name is not sent** (the brief's context is age, sex and procedure); a title using `{patientName}` is worked out with an empty name in the stored result.
6. **The age is worked out by Go** at the hospital's time zone on the day of sending (the browser's `hqPatient` uses the browser's time zone; pending.md S38 note).
7. **The service listens on 127.0.0.1 and reads its port from `EVALUATE_URL`**, one setting for both sides.
8. **The service never logs an error's message** (only its kind and code locations), as a SurveyJS error could quote a value.
9. **Publish's evaluator call runs after the org lock**, inside the transaction, so it checks the rules and chapters as stored at that moment (this also closes S40 verification 1's "Publish checks the rules loaded when the editor opened": the server checks the stored rules).

## Differences from the brief and plan

- 422 instead of 400 (decision 1).
- `form_outputs` keeps every send (a row per send) rather than one per form.

## For S45 (must do)

- **QA won't start without `EVALUATE_URL`**: it's required under CONFIG_STRICT, and `infra/` doesn't set it yet, so every Lambda (including jobs) would refuse to start. Fine while nothing is pushed (answer 1); S45 must set it (or swap in the Lambda client) before the push.
- Copy the `EVALUATE_FAILED` marker into the alarm (as `delivery_test.go` checks `DELIVERY_FAILED_MARKER`).
- The Lambda can call `handle()` from `apps/evaluate/src/handler.ts`.

## Hand checks for Rahul

- T-09's log line, if the tester can't see the dev server's output (an integration test checks it).

## Test results (26 Oct)

- `make lint`: clean (go vet with the integration tag; tsc in packages/clinical, apps/web, apps/evaluate).
- `make test`: every Go package ok (incl. `internal/evaluate`, `internal/config`, `internal/repocheck`); Vitest web 276 passed, apps/evaluate 10 passed, packages/clinical 408 passed (27 files); infra 124 passed.
- `make test-integration`: every package ok (incl. the four new S41 tests).
- **e2e** on a private stack (web :3100, API :8180, evaluate :8190, the shared local database; the conductor's make dev untouched): **91 passed, 5 skipped, 2 failed**: `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a` (the known huge local audit CSV, pending.md) and `test_content_library.py` (a 5 s wait on the huge local episode list under the full run's load; it passed alone, 5 of 5). J3, J7 and J16 (they send HQs) passed, and every send stored a `form_outputs` row with no error (e.g. Low risk firing on a no-grade patient).

## Suggested commits

1. `S41: evaluate() and checkPublish() in packages/clinical, with the shared fixtures` — `packages/clinical/src/evaluate.ts`, `evaluate.test.ts`, `respondent.ts`, `index.ts`, `apps/api/internal/evaluate/testdata/*.json`.
2. `S41: the evaluate service (apps/evaluate), npm run evaluate, make dev starts it` — `apps/evaluate/**`, root `package.json`, `package-lock.json`, `Makefile`.
3. `S41: Go's evaluate client and fake, EVALUATE_URL, Deps.Evaluator` — `internal/evaluate/evaluate.go` + test, `internal/config`, `internal/httpapi/router.go`, `internal/serve`, `internal/apitest`, `cmd/jobs` test, `.env.example`.
4. `S41: on send store form_outputs (fails safe); publish re-checks through the service` — migration 00032, `db/queries/episodes.sql`, `internal/db`, `patient.go`, `questionnaires.go`, `evaluate_integration_test.go`, `routers_integration_test.go`.
5. `S41: docs` — CLAUDE.md, this folder, decisions.md, pending.md, plans and sessions READMEs, the C6 plan's status.
