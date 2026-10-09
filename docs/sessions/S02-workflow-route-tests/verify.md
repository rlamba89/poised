# S02 verification

**Verdict: Verified.** Two independent passes on 8 Oct 2026, both at commit `7e6e535` with S02's work uncommitted:
- **Pass 1:** about 22:15–22:40 BST.
- **Pass 2:** about 22:30–22:45 BST. It re-reviewed the diff, re-ran every test and the code review, and did its own browser checks on a new episode. No code had changed between the passes.

**Summary:**
- **Scope:** the whole brief is done, and nothing large was added.
  - Every route in `router.go` (37) has an integration test.
  - The episode PATCH `null` crash is fixed test-first.
  - The recover middleware is in place.
- **Tests:** every automated test passes. Manual test report 2 passed all 17 cases.
- **Findings:** none blocks S02.
  - The worst problems are older than S02 and already recorded (X-1, and the status write-back).
  - The findings in new code are Minor: one can't happen today, and the rest are gaps in what the tests prove, not bugs.

## Findings, worst first

1. **Major, older than S02, already recorded:** a stale tab's Validate silently overwrites a newer clinician answer (test-report-2 X-1).
   - **Where:** `PUT …/episodes/{eid}/answers/{cid}` has no revision check.
   - **Why it matters:** wrong clinical data, with nothing to warn the clinician.
   - **Status:** S02 is test-only, so it rightly didn't change behaviour. It's scheduled as session **S02a** ([brief](../S02a-answer-revisions/brief.md)): a revision check, so a stale save gets a 409.
2. **Minor, older than S02 (recorded in pending.md §1a):** `updateEpisode` writes the status and ASA grades it loaded back to the database, even when the PATCH changes only details.
   - **Where:** `apps/api/internal/httpapi/episodes.go:246`, and `apps/api/db/queries/episodes.sql:57-60` (`UPDATE episodes SET status = $2, … WHERE id = $1`).
   - **How it happens:**
     1. A clinician's PATCH that changes only details (e.g. `{"procedure": …}`) loads the episode.
     2. In the milliseconds before its UPDATE, Complete HQ review or the patient's submit changes the status.
     3. The PATCH then puts the old status back. The episode ends up Ready for review with `review_completed_at` set, or HQ not complete with `patient_submitted_at` set.
     4. Pass 2's code review adds a second case: two clinicians who set ASA grades at the same moment overwrite each other.
   - **Why only Minor:** the browser sends only the fields it changes (`StatusSelect.tsx:17`, `PoaSummary.tsx:235`), so the window is milliseconds on the server, not a stale page. It's the same family as X-1.
   - **Fix:** add `AND status = <the loaded status>` to the UPDATE, and answer 409 when no row matches. S02a's brief leaves episode PATCH out on purpose. Rahul can add it to S02a (small, and the same idea) or leave it in pending.md §1a.
3. **Minor, new in S02, can't happen today:** after a panic, `recoverPanics` sends the 500 with any headers the handler had already set.
   - **Where:** `apps/api/internal/httpapi/middleware.go:50` writes through `w` without clearing `w.Header()`.
   - **Why it matters:** a handler that sets a cookie (or a Location or Cache-Control header) and then panics before `WriteHeader` would send that cookie with the "Something went wrong" error.
   - **Why it can't happen today:** both handlers that set cookies (`auth_handlers.go:46` and `:54`) call `WriteHeader` straight after `SetCookie`, with nothing in between that could panic.
   - **When to fix:** before C1 adds the Cognito session code. Clear the header map before `serverError`, test-first. This is in pending.md §1a.
4. **Minor, new in S02, test gap (pass 2):** after the refusals, `TestUpdateEpisode` checks only `Status` and `NurseAsa` against the row before.
   - **Where:** `apps/api/internal/httpapi/episodes_integration_test.go:469`.
   - **What it misses:** that a refused `{"procedure": null}`, `{"anaesthetic": null}`, `{"consultant": null}` or a bad `anaesthetistAsa` left those columns unchanged.
   - **Why it's only Minor:**
     - the sign-in, role and hospital refusals send only a status change, and status *is* checked;
     - the null cases are refused before any write;
     - pass 2 checked in the browser that the row is unchanged after each of them.
   - **Fix, when this test is next touched:** compare the whole row with `before`. Then the build notes' "every refusal asserts nothing was written" holds for every column.
5. **Accepted, documented by the builder:** `startedWriter` hides `net/http`'s internal "request too large" hook and direct `http.Flusher` type assertions.
   - **Where:** `middleware.go:57-73`.
   - An oversized body still gets a 400; both passes checked a 20 KB PATCH in the browser. The connection just isn't closed as early.
   - `Unwrap` keeps `http.ResponseController` working.
   - **Related (pass 2):** a `Flush` through `http.ResponseController` commits the response without setting `started`. A panic after it would get an error appended instead of an abort.
   - Nothing streams or flushes today. Revisit both if SSE or streaming is added.
6. **Low, logging only:** the panic log uses `r.Pattern`, which is empty if a panic comes before the mux matches a route. That can't happen with today's wiring, where `recoverPanics` wraps the mux directly. The log would then say "panic in " with no route.
7. **Low, test hygiene (pass 2):** `TestRecoverPanicsKeepsTokensOutOfTheLog` (`middleware_test.go:43`) points the package-wide logger at a `strings.Builder`, then resets it to `os.Stderr`.
   - It's safe today: the test isn't parallel, and no other unit test in the package logs in the background.
   - Adding `t.Parallel()` to it later would cause a data race.

**Considered and dismissed:**
- **Plain `string` fields instead of pointers, so `null` keeps the old value.** S01 rejected this on purpose (a 400 is explicit, and "nothing may break silently"), and S02 follows S01.
- **"`null` means 400 for text but 'keep' for an ASA grade."** That's how the ASA grades worked before S02; S02 changed only the four text fields, as its brief asked. Pass 2 checked that a body mixing the two, `{"procedure": null, "nurseAsa": 2}`, gets a 400 and sets no grade.
- **An unknown status gets 409, not 400.** Older than S02, and already in pending.md §3.
- **"The patient link has no other-tenant case."** The link routes are keyed only by the token, which finds exactly one episode, and the query has no hospital in it. So the same-hospital "another episode" cases prove the scoping just as well. The save route does test a Question Set of Hospital B's HQ (`patient_integration_test.go:131`).
- **`validate()` trims in place.** That's the same convention as S01's `chapterDetails.validate`.

## Checklist

| Check | Result |
| --- | --- |
| **Scope** | Done, as the brief asked. The two browser-enforced rules are tested as built on the server (Rahul's decision, decisions.md 8 Oct), with the gap recorded for C6. |
| **Tests first** | Per the build notes, `TestEpisodeChangeValidate` and four `TestRecoverPanics…` tests failed first. Every route test has happy path, no sign-in, wrong role, another hospital and invalid input, adapted for the token routes and the sign-in stub as the build notes' table explains. Refusals assert that nothing was written, with item 4's gap. |
| **Tenant isolation** | S02 adds no queries. In the browser, Bea (Hospital B) got 404 at Hospital A's address and 403 with A's ids under B. Alex (wrong role) got 403 on the clinician routes. |
| **Nothing hardcoded for the UK (A-20)** | The grep finds only `en-GB` in `apps/web/src/lib/format.ts` and `features/editor/OptionLists.tsx`. Both are older than S02 and already listed in saas-requirements.md §13. |
| **Security** | No secrets, connection strings or patient data in the new code or tests. The panic log names the route pattern, never the path (which holds the patient's token). The patient JSON has no `STAFF`, `clinicianOnly`, `clinicalOutputs`, `testCases`, clinician question names or clinician answers. A staff route without a cookie → 401. A patient save that includes clinician keys stores only the patient's answer. |
| **Docs** | F1 Status, `docs/plans/README.md`, `docs/sessions/README.md`, decisions.md and pending.md were updated by the builder. **Fixed in pass 2:** decisions.md §5 still said "S02 is next"; it now says S02a is next, then S03. The scrub grep prints nothing. |

## Test results (pass 2)

```
make lint   go vet -tags integration ./...   (no findings);  tsc --noEmit (no errors)
make test   ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 13 passed (13), Tests 103 passed (103)
make test-integration
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	1.973s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	0.548s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	0.300s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	1.010s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	10.972s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	0.780s
exit 0
```

- **No `make e2e` yet;** it comes with S03.

## Browser spot-checks (Claude in Chrome, against `make dev`)

### Pass 2: on a new episode

- **What I used:**
  - **The episode:** `e8363f1c-084b-4dc9-8052-d3aa4236631f`, for a made-up patient "VS S02 2235 Test", on "S02 test HQ 16:20".
  - **How I signed in:** through the stub. I read `/api/me` before every step, and each step ran as the expected user.
- **What I left behind (test data only):**
  - the made-up patient and the episode;
  - the consultant on that episode is set to "Dr Trimmed";
  - one patient answer: smoking is Yes.

1. **The S02 fix, as Cara.**
   - These all got `400 {"error":"The request could not be read."}`:
     - `{"status": null}`, `{"procedure": null}`, `{"anaesthetic": null}` and `{"consultant": null}`;
     - `{"procedure": null, "nurseAsa": 2}`;
     - broken JSON;
     - a 20 KB body.
   - Afterwards, the status, procedure, anaesthetic and consultant were unchanged, no ASA grade was set, and no event was written.
   - A valid `{"consultant": "  Dr Trimmed  "}` → 204, and it was stored trimmed.
2. **The patient link, signed out.**
   - **The patient JSON:** no `STAFF`, `clinicianOnly`, `clinicalOutputs`, `testCases`, `q_ecg`, `q_bp` or `q_advice`. Only the two patient sets are listed.
   - **The token:** a bad one → 404.
   - **A staff route without a cookie** → 401.
3. **The patient link at 390 × 844,** in an iframe, because the window couldn't be resized (the tester's agreed workaround).
   - The page fits with no sideways scrolling.
   - "About you" shows only patient questions. Answering Yes to smoking reveals no staff content.
4. **Answer keys and permissions.**
   - **A patient save with clinician keys:** I sent `q_ecg`, `q_advice` and a made-up key with the patient's answer, and got 204. Only `{"q_smokes": "o_yes"}` was stored.
   - **Bea (Hospital B):**
     - at A's address → 404;
     - under B with A's episode id → 403, for both GET and PATCH.
   - **Alex (an author, not a clinician):** 403 on the episode GET, the PATCH, a note, and the patients list.
   - **Afterwards, as Cara:** the procedure was unchanged, and the episode still had only its "Episode created" event.

The recover middleware can't be triggered in the running app without a code change. Its four unit tests cover it, and the builder saw it working with the fix removed.

### Pass 1: on the S02 test episodes

1. **The S02 fix, as Cara:**
   - `{"status": null}`, `{"consultant": null}`, a 20 KB body and broken JSON → 400, and nothing changed;
   - a valid PATCH → 204, stored trimmed.
2. **The patient link** (both S02 test episodes, no cookie):
   - no clinician content;
   - a staff route without a cookie → 401;
   - a patient save to the clinician set → 404.
3. **Permissions:**
   - Alex: 403 on list, get, PATCH, notes and complete review;
   - Bea: 404 on A's routes, and 403 under B with A's id.
4. **A side effect, test data only:**
   - **What happened:** one early check ran as Cara instead of Alex, because a Sign out click at screen coordinates didn't run.
   - **What it changed on Jo Bloggs `fedd92d7…`:**
     - the procedure was briefly set to "hacked", then put back to "Knee replacement";
     - a note "x" was added. Notes are append-only, so it stays.
   - Every click by element reference worked. This matches pending.md §3's unconfirmed sign-out item, and is most likely the testing tool.
   - Pass 2 avoided this by signing in through the API and checking `/api/me` before every step.

## Suggested commits

In this order. Each passes `make test` and `make test-integration` on its own.

1. `Fix a crash when an episode PATCH sends a null field, and recover from handler panics`
   - `apps/api/internal/httpapi/episodes.go`, `episodes_test.go`, `middleware.go`, `middleware_test.go`, `router.go`.
2. `Integration tests for the episode, patient-link, clinician and sign-in routes (F1 Step 2, part 2 of 2)`
   - `apps/api/internal/apitest/testdata/workflow.json`;
   - `apps/api/internal/httpapi/episodes_integration_test.go`, `patient_integration_test.go`, `validation_integration_test.go`, `auth_integration_test.go`.
3. `Docs: S02 build notes, test plan, test reports, verification and status`
   - `docs/sessions/S02-workflow-route-tests/` (all files, including this one);
   - `docs/sessions/S02a-answer-revisions/brief.md`;
   - `docs/sessions/README.md`, `docs/plans/README.md`, `docs/plans/f1-test-foundation.md`, `docs/decisions.md`, `docs/pending.md`.

`reports/` and `research_notes/` stay untracked, as decided (decisions.md §5).
