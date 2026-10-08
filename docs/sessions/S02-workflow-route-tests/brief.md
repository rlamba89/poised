# S02: Integration tests for the episode, patient and clinician routes

**Plan:** F1 ([f1-test-foundation.md](../../plans/f1-test-foundation.md)), Step 2 for the remaining route groups. This is part 2 of 3 of F1.
**Depends on:** S01 verified.

**Ask Rahul first:** nothing.

## Build

Integration tests with the five cases, plus F1's "must also cover" items, for:
- **patients and episodes:**
  - creating an episode needs a published version;
  - creating writes a "created" event;
  - status changes write events.
- **the patient link (`/api/p/{token}`):**
  - **the patient JSON has no `clinicianOnly` elements, `clinicalOutputs` or `testCases`;**
  - answer keys that aren't patient questions are dropped;
  - answers are frozen after submit;
  - a bad token gives 404.
- **clinician answers, validate and complete review:**
  - the first write copies the patient's answers;
  - validated sets are stamped;
  - complete review is refused until every set is validated;
  - it's read-only afterwards.
- **notes:** append-only, with the user and time recorded.
- **F1's check:** removing the clinician-content strip in `chapter/patient.go` makes a test fail. Try it, then revert.
- **Fix a known crash, found by S01's code review** (added 8 Oct with Rahul's yes):
  - `PATCH /api/h/{hid}/episodes/{eid}` with `status`, `procedure`, `anaesthetic` or `consultant` set to `null` dereferences a nil pointer, and the connection drops with no response. Refuse it with a 400, test-first, as in S01's chapter fix.
  - Add a small **recover middleware** in `router.go`, so any handler panic is logged and answered with the plain-language 500 (`writeError`), not a dropped connection.

Bugs found are handled as in S01.

## Not in this session

- Playwright (S03);
- behaviour changes;
- replacing the old `fakeQ` tests (they're left alone, per testing rule 4).

## Done when

`make test-integration` covers every route in `internal/httpapi/router.go` with the five cases, and it passes.

## Manual test focus

A regression of the whole workflow:
- create an episode;
- open the patient link **at phone size**, fill in part of it, leave, come back, and submit;
- **check that no clinician-only content is visible to the patient;**
- the clinician corrects an answer and validates each set;
- complete the review;
- the POA Summary, both tabs;
- add a note.
