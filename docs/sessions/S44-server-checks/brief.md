# S44: Server checks on send and on complete review

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 7 of 8.
**Requirements:** D-1 and §10 ([saas-requirements.md](../../saas-requirements.md): evaluate runs "on submit and on complete review"); two gaps in [pending.md](../../pending.md) that waited for C6: "The API accepts Send with no answers" (S14) and "Complete review isn't checked on the server" (§1a, S02).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S43 verified.

**What exists today:**
- **Send:** the web app keeps "Send my answers" disabled until every shown set is complete, but `submitForm` (`internal/httpapi/patient.go`) accepts a send with required answers missing; Go's only check is the required video tick (`uncheckedTicks`, S34). After S41 the evaluator's result already says whether each shown patient set is complete.
- **Complete review:** `completeReview` (`internal/httpapi/validation.go`) succeeds while the episode is Ready for review, even if no set is validated; only the browser checks that every shown set is validated (decisions.md, 8 Oct: Go can't know which sets are shown).

**Ask Rahul first:** nothing. Both checks were agreed for C6 when the gaps were found; if the evaluator can't be reached, each falls back to today's behaviour (the browser's check), so nothing stops working.

## Build

- **Send, test-first:** before freezing the answers, Go asks the evaluator. If a shown patient set is incomplete, refuse with 409 and a plain message naming it ("Please finish “Your health” before sending your answers."). Both routes (the patient's and staff's on-behalf). If the evaluator fails, accept the send as today (S41's fail-safe applies: Ready for review, the note, the log line). Reuse one evaluator call for the check and the stored result, so a send costs one call.
- **Complete review, test-first:** Go asks the evaluator which sets are shown for the clinician's answers, and refuses (409, "Validate every Question Set first: “About you” isn't validated.") while a shown set has no validated copy. If the evaluator fails, allow it as today and log the `EVALUATE_FAILED` line.
- **Integration tests** (fake evaluator): an incomplete result → 409, nothing frozen, status unchanged; complete → as today; evaluator failure → accepted. Complete review: an unvalidated shown set → 409; a set the fake says is hidden doesn't count; evaluator failure → allowed. The existing five cases still pass.
- **pending.md:** strike both items, pointing to this session.
- **E2E:** the existing journeys pass (J3, J4, J7, J11, J16 send and review real HQs). No new journey: the web app already stops these in the browser.

## Not in this session

- Storing a result on complete review, or re-running rules then.
- The Lambda on AWS (S45).

## Manual test focus

1. Send an incomplete HQ by calling the API directly: refused, naming the set; the episode is unchanged. The normal send in the app works.
2. Complete review by calling the API with one shown set not validated: refused, naming it. A set hidden by its condition doesn't need validating.
3. With the evaluator stopped, both work as before (and the log has the line).
