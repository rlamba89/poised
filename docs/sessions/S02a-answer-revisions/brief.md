# S02a: A stale save of clinician answers is refused, not silently applied

**Plan:** F1 ([f1-test-foundation.md](../../plans/f1-test-foundation.md)), a bug fix found by S02's manual testing ([test-report-2.md](../S02-workflow-route-tests/test-report-2.md), X-1). F1 fixes bugs in their own commits, with the test that shows them.
**Depends on:** S02 verified and committed.

**Ask Rahul first:**
- **Patient answers too?** The patient's autosave (`PUT /api/p/{token}/answers/{cid}`) can lose answers in the same way: the patient has the link open on a phone and a laptop, and the older one saves last. Recommended: **yes, the same check in this session.** It's the same pattern on the same table, and a patient's lost answer is as silent as a clinician's.

## The problem (X-1)

Two tabs, or two clinicians, open the same Question Set of an episode.
1. Tab A changes an answer, and it autosaves.
2. Tab B, still showing the old answer, clicks **Validate**.
3. Tab B's save is accepted and stamped "Validated". Tab A's change is lost, and nobody is told. The POA Summary shows tab B's answer.

**Cause:** `PUT /api/h/{hid}/episodes/{eid}/answers/{cid}` has no stale-revision check. Question Set saves in the editor already have one (409).

## The fix: a revision check, like Question Set saves (Rahul, 8 Oct)

Rahul chose this over "one editor at a time" (a lock). A per-person lock wouldn't stop one person's two tabs. And a lock needs expiry, a "still here" signal and a take-over button, and still needs this check after a lock expires.

- **Database:** a migration adds `revision int NOT NULL DEFAULT 0` to `episode_answers`.
- **Reading:** `GET …/episodes/{eid}/hq` returns each answers row's `revision`. (If the patient's answers are in scope, `GET /api/p/{token}` returns theirs too.)
- **Saving:** the body carries the revision the browser started from: `{data, validated, revision}`. With no row yet, it sends `0`.
  - The save happens only if the row's revision still matches (or there's no row and the revision is `0`). The revision goes up by 1, and the response returns the new one: `200 {"revision": n}`.
  - **Otherwise it's a 409 and nothing is written.** The message, in plain language: "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved."
  - **Two first saves at once** (both tabs send `0`): one wins, the other gets the 409. Use the insert's `ON CONFLICT … WHERE revision = …` so it's one statement.
  - The existing rules stay: saves only while Ready for review, and the stamp is set by `validated: true` and cleared by a later save without it.
- **Browser** (`ValidateSet.tsx`, and `PatientHQ.tsx` if in scope):
  - keep the revision from the last load or save, and send it with each save;
  - **on a 409, stop autosaving** and show the message with a **Reload** button, which reloads the set from the server;
  - Validate doesn't stamp anything on a 409.

## Tests (TDD, failing first)

- **Integration** (`validation_integration_test.go`, and `patient_integration_test.go` if in scope):
  - **a stale revision → 409, and the row is unchanged:** the data, the stamp and the revision;
  - a matching revision saves and returns revision + 1;
  - the first save with `0` creates the row at revision 1; a second `0` → 409;
  - **the X-1 sequence itself:** save from "tab A", then validate from "tab B" with the old revision → 409, and A's answer is kept, unvalidated;
  - the five cases still pass;
  - a missing or non-number `revision` → 400.
- **Unit:** any pure helper added for the browser (e.g. deciding what a 409 means) gets a Vitest test.
- `make lint test test-integration` pass.

## Not in this session

- Showing "Cara is also looking at this set" (presence). It's a convenience to add when a hospital asks for it.
- Locks.
- Other routes. The chapter save already has its check, and episode PATCH and notes don't overwrite answers.
- Playwright: S03 doesn't exist yet. Ask Rahul whether S03's J4 should gain a two-tab step.

## Done when

The X-1 steps from test report 2 give the second tab a clear "changed somewhere else" message with Reload, and the first tab's answer is kept. The integration tests above pass.

## Manual test focus

- **X-1 exactly:** two tabs on one Question Set. Change an answer in tab A and let it save. Validate in tab B. **Expected:** tab B shows the message and Reload. After Reload, it shows tab A's answer, unvalidated. The POA Summary shows tab A's answer.
- **Two first saves:** both tabs open a set that has never been saved, and both change an answer. The second one to save gets the message.
- **Normal use is unchanged:** one tab corrects, validates and completes the review, with no messages. The T-07 to T-09 checks from S02 still pass.
- **If patient answers are in scope:** the patient link open in two tabs; the older tab saves after the newer one and gets the message (at phone size).
