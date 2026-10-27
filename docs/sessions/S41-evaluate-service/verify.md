# S41 verification

## Verification 1 (27 Oct 2026, automatic run)

**Verdict: Verified.**

Reviewed the whole change (27 modified files plus the new `apps/evaluate`, `internal/evaluate`, migration 00032, `evaluate.ts` and the integration tests) against the brief, the C6 plan and both test reports, ran `/code-review high`, every automated suite and the e2e journeys, and spot-checked in Chrome.

### Scope

Everything in the brief is built: `evaluate()` and `checkPublish()` as pure functions in `packages/clinical`; the Node service in `apps/evaluate` (127.0.0.1 only, port from `EVALUATE_URL`, esbuild, versioned by commit); `make dev` starts it as a third process; Go's `Evaluator` interface, HTTP client with short timeouts and the `Recorder` fake; `EVALUATE_URL` in `internal/config` (required under CONFIG_STRICT); `Deps.Evaluator` panics when missing; the shared fixtures read by Go and both TypeScript suites; `form_outputs` written on both submit routes; fail-safe (204, error row, one `EVALUATE_FAILED` line with the form id only, an automatic General notes line); publish re-checked through the service (422 with the problems, 503 when unreachable). No new routes. Nothing beyond the brief.

Differences, both recorded in the build notes and the plan: publish refuses with the existing 422 (not 400), and `form_outputs` keeps a row per send.

### Findings (none blocking; recorded in `docs/pending.md` under S41)

1. **Minor: a rules (or chapter) save during the publish check can be published unchecked.** `apps/api/internal/httpapi/questionnaires.go:249` checks `row.Rules`, read by `loadQuestionnaire` before the transaction, and the check can take up to 20 s. A second tab's rules save that commits in that window is then published, because `PublishVersion` doesn't look at `rules_revision`. This needs two tabs and a save within seconds. Before S41 the API didn't check rules at all. The fix is to re-read the version row `FOR UPDATE`, with its chapters, inside the transaction. I corrected pending.md's claim that S41 fully closes S40's "rules loaded when the editor opened" item.
2. **Minor (for S45): the evaluate calls run while locks are held.** Publish holds the trust's advisory lock (`questionnaires.go:220`) for up to 20 s, and a send holds the form's and episode's rows (`patient.go:560`) for up to 5 s. This was the builder's deliberate choice (decision 2) and is fine locally. It's worth revisiting with Lambda cold starts and Neon's small pool.
3. **Tidy-up:**
   - The service's 10 MB body limit (`apps/evaluate/src/handler.ts:13`) shows as a permanent 503 "Try again in a minute" for a very large version.
   - `EVALUATE_URL` accepts a query, fragment or user info (`config.go:370`), and the service uses only its port (`handler.ts:25` `portOf`).
   - `evaluate.Problem` duplicates `keys.Problem`.
   - `cmpDuration` duplicates `cmp.Or`.

I weighed these code-review findings and set them aside:
- **The `Recorder` treating an empty audience differently:** `chapters.audience` is `NOT NULL DEFAULT 'patient'`, so it can't happen.
- **The age's time zone vs the browser's:** already in pending.md (S38).
- **QA failing without `EVALUATE_URL`:** already recorded as S45's must-do. Nothing is pushed before S45 (Rahul's answer 1).

### Checks

- **Tests first:** there are no new routes. The submit routes and publish keep their five cases, and four new integration tests cover the stored row, the fail-safe on both routes, the publish input and the publish refusal (422 and 503). Unit tests cover the Go client, fake and config; `evaluate.test.ts` has 16 tests; `apps/evaluate`'s `server.test.ts` has 10.
- **Tenant isolation:** `form_outputs` is written only from the already scoped form (`f.FormID`, the version's chapters). Nothing reads it yet. Publish runs after `loadQuestionnaire` and the role check, so another trust's request gets a 404 before the evaluator is called.
- **A-20 grep:** only the existing comments in `apps/web/src/lib/format.ts` and `internal/patient` match. Nothing new.
- **Security:**
  - Go sends no name or contact details. The service logs only the form id, the error's kind and the code locations. Go's `EVALUATE_FAILED` line holds the form or version id and the transport error. I read the error rows in the database: no answers.
  - The service listens on 127.0.0.1 only (`lsof`).
  - No patient route exposes the result (test report 2).
  - Clinician-only content still goes only to the server-side evaluator, never to the patient.
- **Docs:** CLAUDE.md, decisions.md, pending.md, the C6 plan's status and both READMEs are updated. `repocheck` passes.

### Spot checks in Chrome (Claude in Chrome, the conductor's make dev)

1. **Another trust:** Bea Author (Trust B) POSTing publish on Trust A's "MT S41 2350" got **404** "Hospital not found."
2. **Wrong role:** Cara Clinician got **403** "Only super clinicians and admins of the whole trust…".
3. **Signed out:** **401** "Please sign in."
4. **F-1's fix:** the T-09 episode's General notes show "The automatic checks didn't run. Review this HQ by hand. | **Automatic**" after "HQ completed by the patient | Patient". The status is Ready for review.
5. **`form_outputs` rows from the e2e and tester sends:**
   - every row has a result and the version `clinical@1ec4c87a3274-dirty`;
   - Low risk and High risk fire as expected, and ASA III and the red flag appear on the High risk row;
   - an API-sent incomplete HQ (row 54) correctly records both patient sets `complete: false` (S44 will refuse such sends);
   - the error rows hold only the transport error.

I didn't stop the evaluate service: the tester covered the fail-safe twice (T-09 to T-11), and the integration tests cover it.

### Test results

- `make lint`: clean.
- `make test`: every Go package ok (including `internal/evaluate`, `internal/config` and `internal/repocheck`). Vitest:
  - web: 276 passed (29 files);
  - apps/evaluate: 10 passed;
  - packages/clinical: 408 passed (27 files).

  infra: 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against the conductor's make dev): **92 passed, 5 skipped, 1 failed**. The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known huge local audit CSV ("Cannot create a string longer than 0x1fffffe8 characters"), and isn't from S41. J3, J7 and J16, which send HQs, passed.

### Hand checks for Rahul

- None needed. The tester saw the `EVALUATE_FAILED` log line, and test report 2 covered phone size with a headless 390×844 screenshot.

### Suggested commits

1. `S41: evaluate() and checkPublish() in packages/clinical, with the shared fixtures`
2. `S41: the evaluate service (apps/evaluate), npm run evaluate, make dev starts it`
3. `S41: Go's evaluate client and fake, EVALUATE_URL, Deps.Evaluator`
4. `S41: on send store form_outputs (fails safe); publish re-checks through the service`
5. `S41: docs`

### Next

`/build-session S42`
