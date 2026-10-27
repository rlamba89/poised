# S44 build notes: server checks on send and on Complete review

Builder, automatic run, 27 Oct 2026. The brief had nothing to ask Rahul.

## What was built

### Go

- **`internal/evaluate`:** a third call, **`Shown(ctx, ShownInput) ([]uuid.UUID, error)`** on the `Evaluator` interface: the sets shown, in order, for the answers that count now. `ShownInput` is the form's id, every set (whole, with its audience), the answers by set and the patient (age, sex). `HTTP.Shown` posts to **`/shown`** (5 s, as Evaluate; a reply without a `shown` list is an error, and no error holds the reply). The fake `Recorder` gives every set by default; **`Hide(ids…)`** makes it leave sets out (as a condition would), `ShownInputs()` lists the calls, and `Fail` fails it too. Shared fixture **`testdata/shown.json`** (the same four sets as `evaluate.json`; the clinician's copy of About you says No to smoking and 1 night's snoring, so Travel is shown and Sleep isn't). Tests `TestHTTPShown`, the new `TestHTTPFailures` case, `TestRecorder`.
- **Send** (`submitForm` in `patient.go`, both routes): after the freeze and the status line, in the same transaction, the one evaluator call S41 already makes now also decides the send. If the result's `patientSets` has a set not complete, Go answers **409 "Please finish “Your health” before sending your answers."** (several: "“About you” and “Your health”"), and the transaction rolls back: nothing frozen, no status change or event, no `form_outputs` row. Complete → as S41–S43 (stored, rule actions, auto-triage). The call failing → as S41 (accepted, Ready for review, the note, the stored error, the log line). `unfinishedSets` and `quotedList` are the words. The chapters are now read once per send and passed to `uncheckedTicks` and `evaluateSent`.
- **Complete review** (`completeReview` in `validation.go`): after the move to Ready for POA, in its transaction (the episode row is locked), `unvalidatedSets` asks `Shown` with every set, **the answers that count now** (the clinician's copy where there is one, else the patient's, as the episode page's `allCurrentAnswers`) and the patient **as on the day they sent** (age at `submitted_at` in the hospital's time zone, as `hqPatient`). A shown set with no validated copy → **409 "Validate every Question Set first: “About you” isn't validated."** (several: "“Your health” and “Nurse checks” aren't validated."), rolled back. The call failing → allowed as before, with one log line `EVALUATE_FAILED complete review, form <id>: …` (no answers); nothing is stored and no General notes line is written (the brief: "allow it as today and log the line"). `inZone` (in `patient.go`) is `today`'s time-zone step, shared.
- **Integration tests** (`server_checks_integration_test.go`, fake evaluator):
  - `TestSendRefusedWhileASetIsIncomplete`: an incomplete set → 409 naming it, nothing frozen (status, submitted, events, status history, `form_outputs` all unchanged) and the patient can still save; two sets named; staff on behalf the same; complete → sent with exactly one evaluator call and one stored result; a set the patient's form doesn't show isn't asked for.
  - `TestCompleteReviewChecksEverySetShown`: none validated → 409 naming all three; the evaluator's input (every set whole, the clinician's copy over the patient's, age 60 on the 2020 send date, sex); one or two not validated → named; **a set the fake says is hidden doesn't count** → 204, Ready for POA, the usual event; all validated → as today.
  - `TestCompleteReviewWhenTheEvaluatorFails` (not parallel: reads the log): allowed with nothing validated; one `EVALUATE_FAILED complete review, form …` line, no answers in the log; no `form_outputs` row, no note.
  - The send's evaluator failure is S41's `TestSubmitWhenTheEvaluatorFails` (unchanged, passes).
  - **Older tests** that completed a review without validating now validate first (`validateAll`, which stamps a validated clinician row for every set): `TestCompleteReview`'s happy path, `TestReopenReview`, `TestUndoStatus`, `TestEpisodeStatusView`, `TestUpdateEpisodeKeepsAStatusThatJustChanged`, `TestAuditRowsWritten`. `TestReopenReview`'s check of the kept clinician row now names its set (there are three rows now). The five cases of both submit routes and of Complete review are unchanged and pass.

### TypeScript (for Rahul to review)

- **`packages/clinical/src/evaluate.ts`:** `shownSetsFor(input: ShownInput): { shown: string[] }` is three lines: check the input (the shared part of `evaluate`'s check, now `checkAnswered`), then `shownSets(sets, answers, "clinician", patient)` — the same function the episode page calls before enabling **Complete HQ review** — and return the ids. Every set's answers count, a clinician set's too (unlike `evaluate`, which looks only at what the patient sent). Tests in `evaluate.test.ts`: the shared fixture; the clinician's copy deciding; a clinician set's condition on another clinician set's answer; the patient's age; bad input refused.
- **`apps/evaluate/src/handler.ts`:** `POST /shown` → `shownSetsFor`; its error log names only the form, as `/evaluate`'s. Tests in `server.test.ts` (the fixture over HTTP, 400 on bad input, the log holding no answer).

### e2e

No new journey (the brief). Journeys that sent an HQ through the API with required answers missing, to set up something else, now give every required answer first:
- `test_j16b_prefill_cross_trust.py`: `sendable()` completes Trust B's Demo pre-op HQ with `DEMO_PRE_OP_SENDABLE` under the journey's own answers (used by `trust_b_then_trust_a`, which J5 and the audit journey share, and by the "one detail different" journey);
- `test_j5_poa_summary.py` (the started-fresh journey): Cara gives About you's and Your health's required answers before sending the follow-up HQ;
- `test_j8_reminders.py`: the patient answers every required question before sending.

## Decided by the builder (automatic mode)

1. **A third service call, `/shown`, for Complete review** rather than reusing `/evaluate`: `/evaluate` counts only the patient's answers (the rules look at what the patient sent), while Complete review needs the sets shown for the clinician's answers, as the episode page works them out. One small pure function on the same `shownSets` the page uses, so they can't disagree. S45's Lambda needs to route this path too (`handle()` already does).
2. **The send's check runs after the freeze, in the same transaction**, and a refusal rolls everything back. The brief said "before freezing"; the effect is the same (nothing frozen, status unchanged), and this way the answers checked are exactly the ones that would be frozen, with still one call per send.
3. **Complete review's check runs after the status update, in its transaction**, for the same reason (the episode row is locked, so the stamps read are the ones the review completes on).
4. **Every incomplete or unvalidated set is named**, not just the first ("“About you” and “Your health”").
5. **Complete review's evaluator failure writes no General notes line** (only the log line), as the brief says; the browser's own check still stands then.
6. **The patient's age for Complete review is on the day they sent** (as the page's `hqPatient`), at the hospital's time zone (as S41's send).

## Not done (out of scope)

Storing a result or re-running rules on Complete review; the Lambda (S45).

## Test results (27 Oct)

- `make lint test`: pass (Go unit; Vitest 285 + 12 + 416; CDK 124).
- `make test-integration`: pass (all packages; `internal/httpapi` ok).
- `make e2e` against a private stack (API :8180, web :3100, evaluate :8190, all on this code; the conductor's `make dev` was started before these changes; the private stack is stopped): first full run 88 passed, 10 failed, all fixed above except the known one; last full run **95 passed, 3 failed, 5 skipped**. None of the 3 is S44's:
  - `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`: the shared database's huge audit log ("Cannot create a string longer than 0x1fffffe8"), known;
  - `test_j7c_returning_patient.py::test_j7c_twins_on_one_phone_are_asked_their_name` and `test_j9b_cancel.py::test_j9b_cancel_with_a_reason_ends_the_patients_side`: both passed in the first full run, and fail the same on the conductor's stack (old code) after it: the sign-in link's rate limit (10 per IP address per hour, PAT-11) is used up by the repeated runs. They don't send or review.
- **The conductor must restart `make dev` before testing:** the running evaluate service has no `/shown`, so until then Complete review would fall back (allowed, with an `EVALUATE_FAILED` line).
- A one-off check against the real evaluate service (workflow HQ, not kept): a send with About you and Your health unfinished → 409 naming both; Complete review with nothing validated → 409 naming all three; with Nurse checks left → 409 naming it; all validated → 204.

## Suggested commits

1. `S44: the evaluate service's /shown (packages/clinical shownSetsFor, apps/evaluate, Go's Shown and the fake), with the shared fixture` — `packages/clinical/src/evaluate.ts`, `evaluate.test.ts`, `apps/evaluate/src/handler.ts`, `server.test.ts`, `apps/api/internal/evaluate/*`, `testdata/shown.json`.
2. `S44: the server refuses an incomplete send and an unvalidated Complete review` — `apps/api/internal/httpapi/patient.go`, `validation.go`, `server_checks_integration_test.go`, and the older integration tests' `validateAll`.
3. `S44: e2e journeys send complete HQs` — `e2e/journeys/test_j16b_prefill_cross_trust.py`, `test_j5_poa_summary.py`, `test_j8_reminders.py`.
4. `S44: docs` — this folder, `docs/pending.md`, `docs/plans/*`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/plan-workflow.md`, `CLAUDE.md`.

## Test report 1 (27 Oct): all passed

15 passed, 0 failed, 2 optional regressions skipped (R-01, R-04). No code change (the conductor's instruction). Its exploring notes are in pending.md, "From S44's test report 1": the older "…so the answers can't be changed" wording on a repeated Complete review, a closed window's text left in the page. **Hand check for Rahul:** the patient's form at 390×844 (the tester's window couldn't be resized; S44 adds nothing visible). The tester checked T-11's `EVALUATE_FAILED` lines in `make dev`'s output.
