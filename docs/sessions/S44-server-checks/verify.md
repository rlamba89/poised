# S44 verification

## Verification 1 (27 Oct 2026, automatic run)

**Verdict: Verified.**

Reviewed against the brief and C6's plan: the whole working-tree diff, `/code-review high`, the tests run again, and four spot checks in Chrome. Everything in the brief is built and nothing beyond it. Both pending.md gaps are struck. The fail-safe matches S41's.

### Findings (none blocking, worst first)

1. **A save that clears a validation can slip past Complete review's check** (`validation.go:150`, `unvalidatedSets`; `db/queries/episodes.sql:398`, `SaveClinicianAnswers`). Complete review locks the episode row and then reads the stamps. `SaveClinicianAnswers` checks `e.status = 'ready_for_review'` without taking that lock, so a second clinician re-saving a set *without* Validate in the same few milliseconds can commit after the read. The review then completes with that set unvalidated. This needs two people acting on one episode at the same instant, and the set was validated a moment earlier. A `FOR SHARE OF e` in the save would close it. Added to pending.md.
2. **Complete review's evaluator call runs inside the transaction, after the row lock** (`validation.go:150`). A slow or hanging service holds the episode row for up to 5 s (`EvaluateTimeout`). The send has had the same shape since S41. This is acceptable locally, but S45's Lambda cold starts make it worth watching. Added to pending.md.
3. **The patient's age can differ by a day between the browser and the server** (`validation.go`, `unvalidatedSets`, versus `hq.ts` `hqPatient`). Go works out the age at sending in the hospital's time zone, and the browser uses the viewer's. They disagree only when a patient sends on their birthday around midnight and a set's condition is on age, so Go could name a set the page doesn't show. Added to pending.md.
4. **Small cleanups (no behaviour change):**
   - `submitForm` still reads `ListEpisodeAnswers` twice (`uncheckedTicks` and `evaluateSent`).
   - `unvalidatedSets` makes an extra `GetEpisodeForm` query only for the time zone.
   - The loop that builds `[]evaluate.Set` is copied in `evaluateSent` and `unvalidatedSets`.
   - `Recorder.Hide` keeps the caller's slice without cloning it.
5. **The builder's choices are sound.**
   - A third call (`/shown`) uses the same `shownSets` as the episode page, so the two can't disagree. `/evaluate` counts only the patient's answers.
   - The send's check runs after the freeze, and a refusal rolls back the transaction. The tests confirm nothing is frozen: status, `submitted_at`, events, status history and `form_outputs` are all unchanged.
   - Checked: `evaluateSent` writes only inside the transaction, so a refusal leaves nothing behind. The audit row is written outside it, as for every route.

### Checks

- **Scope:** both server checks, on both send routes; the fail-safes; the integration tests the brief lists (incomplete → 409 with nothing frozen; complete → as before, with one evaluator call; evaluator failure → accepted; unvalidated shown set → 409; hidden set doesn't count; evaluator failure → allowed with the log line). The five existing cases for each route still pass.
- **Tenant isolation:** no new query. `unvalidatedSets` reads the episode `loadEpisode` already scoped to the hospital. Spot check: Bea (Trust B) gets 404 on a Trust A episode's `complete-review`.
- **A-20:** the grep finds nothing new outside the e2e test files (which existed before).
- **Security:** the log lines name only the form id (tested on both sides). `/shown` is only on the local evaluate service, and patient JSON is unchanged. `repocheck` passes.
- **Docs:** the plan's Status, `docs/plans/README.md`, `docs/sessions/README.md`, `decisions.md` (27 Oct entry), `pending.md`, `plan-workflow.md` and `CLAUDE.md` are all updated.

### Spot checks in Chrome (Cara Clinician; the conductor's `make dev`, evaluate service with `/shown`)

1. **Complete review with nothing validated** (Demo pre-op HQ, Ready for review) → `409 "Validate every Question Set first: “About you”, “Your health” and “Nurse checks” aren't validated."`
2. **Another trust:** Bea Author on the same episode → `404`.
3. **Staff on-behalf send, conditional set** (the tester's "S44 test 0625" HQ): with snore = Yes and Sleep unanswered → `409 "Please finish “Sleep” before sending your answers."`, and the form stays unsent. With snore = No → `204`.
4. **Hidden sets don't count:** `complete-review` → `409 … “About you” isn't validated.` (Sleep and Nurse checks, both hidden, aren't named). The page lists only About you. After validating it in the page, **Complete HQ review** moved the episode to Ready for POA with "HQ review completed by Cara Clinician".

I didn't stop the evaluate service while `make e2e` was running against it. The fail-safe is covered by `TestCompleteReviewWhenTheEvaluatorFails`, by S41's `TestSubmitWhenTheEvaluatorFails`, and by the tester's T-11/T-12.

### Test results

- `make lint test test-integration`: pass (exit 0). Vitest: 285 + 12 + 416 passed. CDK: 124 passed. Go integration: every package ok, `internal/httpapi` ok.
- `make e2e` (against the conductor's `make dev`, on this code): **96 passed, 2 failed, 5 skipped**. Neither failure comes from S44:
  - `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a` is the shared database's huge audit log, already known.
  - `test_j7_patient_invites.py::test_j7_invite_by_text_and_email_then_date_of_birth_and_send` found 2 captured messages for its mobile where it expected 1. That's a reused mobile on the shared database, already known for J7. It fails before any send.
  - J3, J4, J11, J11b, J16 and the three changed journeys (J5, J8, J16b) all passed.

### Hand checks for Rahul

- The patient's form at 390×844 (T-01, T-02, T-05). The tester's window couldn't be resized. S44 adds nothing visible.

### Commits

1. `S44: the evaluate service's /shown (packages/clinical shownSetsFor, apps/evaluate, Go's Shown and the fake), with the shared fixture`
2. `S44: the server refuses an incomplete send and an unvalidated Complete review`
3. `S44: e2e journeys send complete HQs`
4. `S44: docs`
