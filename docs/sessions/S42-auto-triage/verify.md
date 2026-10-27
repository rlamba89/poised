# S42 verification

## Verification 1 (27 Oct 2026, automatic run)

**Verdict: Verified.**

I reviewed the whole change against the brief, Rahul's answers, the C6 plan, the build notes and both test reports. That covered 42 modified files plus migration 00033, `internal/triage` and `triage_integration_test.go`. I also ran `/code-review high`, every automated suite and the e2e journeys, and spot-checked the riskiest parts myself. Test report 2 says everything passed.

### The guards (safety-relevant), checked closely

- **A failed evaluate means no move.** `evaluateSent` returns nil when the call fails (`apps/api/internal/httpapi/patient.go`), and `autoTriage` runs only on a result. The episode stays Ready for review, with "The automatic checks didn't run". The integration test `TestAutoTriageOnSend` covers the failing fake, and both testers stopped the real service (T-13, Pass twice).
- **A red or amber flag, an Incomplete score, or an anaesthetist's-ASA rule means no move.** Both sides check these.
  - `packages/clinical` `firedRules` takes the auto-triage action out of the rule and gives a `heldReason`. Go's `triage.Decide` then re-checks the result itself: `len(res.Flags) > 0`, any `Sets[].Scores[].Incomplete`, or a fired `requireAnaesthetistAsa` with the target Ready for admission.
  - A rule that was held never carries an `autoTriage` action, so Go can't pick it up (that includes `noGrade`).
  - Unknown targets are ignored (`slices.Contains(asarules.Targets, …)`). `CheckAutomatic` is a separate table, so no role can make `auto_triage` by hand (`TestCheckEveryPair`, `TestCheckAutomaticEveryPair`).
- **An independent check of the invariant on the shared local database,** against the real service's stored results (`form_outputs`): every one of the 54 episodes with an `auto_triage` event has a result (none had an error), and none had a flag, an Incomplete score, or an anaesthetist's-ASA rule with a Ready for admission target. All 5 flagged sends today stayed un-triaged.
- **A re-open clears the auto-triage.** `SetEpisodeStatus` clears it with `reopen`. Undo can't step back an auto-triage (`TestUndoTarget`).

### Scope

Everything in the brief is built, with Rahul's answer 2 in place of the brief's question 2: the message goes at once by default, and the per-HQ `messageAfterCheck` setting holds it until Checked. The builder's differences are recorded in the build notes and the C6 plan's Status:
- the move goes from Ready for review, after the Submit event;
- a guard writes a note;
- a re-open clears the auto-triage.

There's nothing big that the brief didn't ask for.

### Tests, tenancy, security, docs

- `triage-checked` has its five integration cases, plus twice, never auto-triaged, not sent, cancelled, archived and stale. It's audited (`triage.checked`), and it's a staff route only.
- The re-open, the messages and the worklist have their own tests (`TestReopenAutoTriaged`, `TestAutoTriageMessages`, `TestWorklists`). J11's patient half and the second-tab journey are in `e2e/`.
- Tenancy: every query is by episode id after `requireHospital` has resolved it within the hospital.
- The A-20 grep finds only test mobiles in `triage_integration_test.go`. `repocheck` passes.
- Nothing in the triage notes or the messages holds an answer.
- The docs are updated: CLAUDE.md, AGENTS.md, decisions.md, pending.md (including the hazard log), and the plan and sessions READMEs.

### Findings (none blocking; added to pending.md under "From S42's verification")

1. **Minor: a held Ready for admission message can be sent twice** (`apps/api/internal/httpapi/statuses.go`, `triageChecked`; `db/queries/episodes.sql` `SetEpisodeStatus`).
   - This needs `messageAfterCheck` on, which isn't the default.
   - The sequence: the episode is auto-triaged with the message held. A clinician sets On hold, then Ready for admission again, and that Set sends the message. A clinician then clicks Checked, and `triage_message_pending` is still true, so the message goes a second time.
   - The fix is to clear `triage_message_pending` on every status change made by hand, not just a re-open. That matches builder's decision 6: a clinician who moves it has made their own decision.
2. **Product question for Rahul: the Incomplete guard counts clinician sets' scores** (`apps/api/internal/triage/triage.go` `incomplete`; `rules.ts` `reasonFor`).
   - `evaluate()` puts answers only into the patient's sets. A Calculation in a clinician set that needs a clinician-entered input (for example a measured neck size, or a BMI from the nurse's measurements) is therefore always Incomplete at send. That blocks auto-triage for every patient on that HQ.
   - This fails safe and matches "no score is Incomplete" as written. None of today's HQs is affected: the seeded and test HQs auto-triage.
   - Options:
     - keep it (recommended until C7's observations exist);
     - count only scores whose inputs are all patient questions.
3. **Minor: `messageAfterCheck` is read with its decode error ignored** (`patient.go`, `autoTriage`). If the stored rules couldn't be read, the message would go at once. `asarules.Check` already refuses such a value when it's saved, and "at once" is the default, so this can't happen today. Decoding through `asarules.RuleSet` would remove the third definition of the field.
4. **Tidy-up:**
   - `automaticStatusMessage` nearly copies `statusMessage`.
   - `Decide`'s Blocked strings repeat the `guards` map.
   - `in_worklist`'s `auto_triaged` branch looks up `episodes` again for each row, where it could take the two columns as arguments.
   - `AutoTriageEpisode` returning 0 rows becomes a 500 for the whole send. That can't happen in the same transaction, but it isn't fail-safe in shape.
   - `EpisodePage.tsx` has a second import from `./StatusControl`.

### Spot checks (Claude in Chrome, as Cara Clinician at Hospital A1)

- **Cross-hospital and cross-trust Checked:**
  - Hospital A2's auto-triaged episode by its own path: 404 "Hospital not found". Through A1's path: 404 "Episode not found".
  - A Trust B episode: 404 both ways.
- **Refusals:** no revision 400; a stale revision 409 `stale`; a cancelled one 409; an already-checked one 409. A clinician re-opening gets 403.
- **The episode page** of an auto-triaged episode (Sam MT42r T13b): the teal card shows the rule, "no ASA grade, no red or amber flags", "Not yet checked by a clinician." and Checked, and the sets are stamped. Clicking Checked showed "Checked by Cara Clinician on 27/10/2026, 02:53.", and the button went. Its `audit_log` row is `triage.checked` 204, and the episode is no longer `in_worklist('auto_triaged')`.
- **The invariant query** over every auto-triaged episode (above).
- I didn't stop the evaluate service: both test rounds did (T-13), and the integration test covers it. The conductor's evaluate (node PID 23488) is still the only listener on :8090.

### Test results

- `make lint`: clean.
- `make test`: every Go package ok (incl. `internal/triage`, `internal/episode`, `internal/repocheck`). Vitest: web 279 passed, apps/evaluate 10, packages/clinical 411. Infra: 124 passed.
- `make test-integration`: every package ok (29).
- `make e2e` (against the conductor's make dev): **94 passed, 5 skipped, 1 failed**. The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known one: Trust A's huge local audit log, "Cannot create a string longer than 0x1fffffe8 characters". It isn't S42's.

### Hand checks for Rahul

- None.

### Commits (local, not pushed)

1. `S42: the auto_triage move (system only), the Auto-triaged worklist, the message on auto-triage`
2. `S42: triage.Decide and its guards; evaluate.Result's scores; asarules messageAfterCheck`
3. `S42: auto-triage on send, Checked, re-open clears it`
4. `S42: the guards in packages/clinical; the episode page's card and stamps, the POA Summary, the Rules page's setting`
5. `S42: J11's patient half; published_hq without auto-triage`
6. `S42: docs`
