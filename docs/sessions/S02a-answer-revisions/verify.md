# S02a verification

Three rounds: verification 3 (11 Oct, Verified) first, then verifications 2 and 1 below it. pending.md's links to "verify.md item 2" and "item 3" mean verification 1's items.

## Verification 3 (11 Oct)

**Verdict: Verified** (11 Oct 2026, about 22:10 BST).

- **Reviewed:** commit `b3876f9`, with S02a's work uncommitted, including the fixes for verification 2.
- **Independent:** the same session that ran verification 2. It didn't build or test S02a.
- **No manual test report covers verification 2's fix.** Test report 2 (all passed) came before it. Instead, I retested the fix and the patient cases it touches in the browser (below), along the lines of the test plan's T-12 steps 9–13, T-04 and T-06. The builder also checked it (build notes).

### Verification 2's findings

| # | Finding | Now |
| --- | --- | --- |
| 1 | **Major:** a complete section whose change failed to save still counted as complete, so the HQ could be sent without the change | **Fixed, test-first.** `isSetReadyToSend` (`packages/clinical/src/respondent.ts:44`) counts a set as complete only when the server's answers are complete and nothing on screen is unsaved. `PatientHQ.tsx:107` uses it. Its Vitest test has 5 cases, including this sequence. **Seen working in the browser** (spot-check 1). |
| 2 | **Low:** "Saved" after a failed save; technical error text | **The label is fixed:** it says **Not saved** in red (`PatientHQ.tsx:220`, `:285`). **The text is recorded in pending.md §3**, as a fix in `lib/api.ts` for every page. |
| 3 | **Low:** the clinician page is stuck if Reload itself fails | **Recorded** in pending.md §3. |
| 4 | **Low:** the patient's Reload kept an old error and had no loading state | **Fixed** (`PatientHQ.tsx:62-75`, `:133`). **Seen working** (spot-check 2). |
| 5 | **Low:** the PATCH 409's wording promises more than it checks | **Recorded** with the ASA item in pending.md §1a, as advised. |

### New findings

None that block, and nothing new worth recording:
- the new code is small: a 4-line helper, a 3-state label, and `load` clearing the error;
- the server code hasn't changed since verification 2;
- the clinician page hasn't changed since verification 2.

### Checklist

As in verification 2: scope, tests first, tenant isolation, A-20, security and docs all pass. The only additions:
- **Tests first:** the new Vitest case;
- **Docs:** pending.md §3 gains findings 2 and 3, and §1a gains finding 5.

### Test results

```
make lint   go vet -tags integration ./...  (no findings);  tsc --noEmit (no errors)
make test   ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 14 passed (14), Tests 111 passed (111)
make test-integration
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	6.172s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	0.395s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	0.884s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	0.738s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	5.811s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	0.253s
EXIT 0
```

### Browser spot-checks (Claude in Chrome, against `make dev`)

- **Setup:** the same as verification 2. The tab is "hidden", so the phone cases ran in 390 × 844 same-origin frames, by clicking the app's own controls from script. "Network down" means the frame's answer PUTs failed with "Failed to fetch".
- **Test data:** made-up episodes `S02a Verify32200` (`bb261077-…`) and `S02a Verify3…c`, both on `S02 test HQ 23:06`.

1. **Verification 2's finding 1, the same sequence as before. Pass.**
   1. Both sections were answered and saved (3 PUTs, all 200), showing "2 of 2".
   2. The network went down. The patient reopened About you and changed smoking to **Yes**. Both PUTs failed, and the label said **Not saved** in red.
   3. **All sections:** About you showed **In progress / Continue**, "1 of 2", and **Send my answers** was disabled.
   4. The network came back. Reopening showed **Yes**, and leaving sent it (PUT 200). Then it showed "2 of 2", and Send was enabled.
   5. **The submit** (204) froze `q_smokes: o_yes` at revision 3. The change is kept.
   - **The "Send your answers?" dialog** can't open in a hidden tab (no animation frames, as both test reports found). So the submit was the same `POST …/submit` the dialog sends.
2. **T-06 and finding 4. Pass.**
   - The HQ had been sent, but this frame didn't know. It still showed the old "Failed to fetch" banner.
   - A change there got a **409**, and the card "Your answers weren't saved" appeared, saying "already been sent", with Reload.
   - **Reload** showed the thank-you page, and **the old banner was gone**. There was no sideways scroll.
3. **T-04: the link on two devices. Pass.**
   - Device B saved smoking Yes (200). Stale device A then answered angina, and got a **409** and the card.
   - After Reload, A showed B's **Yes**, and A's next change saved (200), with "Saved".
   - B then left the section without changing anything, and **sent nothing**.

### Suggested commits

In this order. Each commit should pass `make lint test test-integration` on its own. The build notes say how to split the generated code for commit 1.

1. **`Refuse an episode PATCH that would put back a status that just changed`**
   - `apps/api/db/queries/episodes.sql` (`UpdateEpisode` only) and the generated `UpdateEpisode` hunks;
   - `internal/httpapi/episodes.go`, `episodes_test.go`;
   - `cast_integration_test.go` (`meanwhile`) and `episodes_integration_test.go`.
2. **`Refuse a stale save of clinician or patient answers, with Reload in the browser`**
   - **Database:** the migration `00005_answer_revisions.sql`, the rest of `episodes.sql` and the generated code;
   - **Go:** `validation.go`, `patient.go`, `validation_test.go`, `patient_test.go` (deleted);
   - **Tests:** `validation_integration_test.go`, `patient_integration_test.go`;
   - **Clinical package:** `answerSaver.ts` and `answerSaver.test.ts`, `respondent.ts` and `respondent.test.ts` (`isSetReadyToSend`), `index.ts`;
   - **Web:** `hq.ts`, `ValidateSet.tsx`, `PatientHQ.tsx`.
3. **`Docs: S02a build notes, test plan, test reports, verification and status`**
   - `docs/sessions/S02a-answer-revisions/`, `docs/sessions/README.md`;
   - the S03 and S13 briefs;
   - `docs/plans/README.md`, `f1-test-foundation.md`, `c2-patients-signin.md`;
   - `docs/decisions.md`, `docs/pending.md`.

---


## Verification 2 (11 Oct)

**Verdict: Needs fixes** (11 Oct 2026, about 21:30 BST).

- **Reviewed:** commit `b3876f9`, with S02a's work uncommitted, including the fix for verification 1's finding.
- **Independent:** a new Claude session, which didn't build or test S02a. The code review ran as its own agent, and I checked each of its points against the code.
- **What's good:**
  - X-1 is fixed, and so is the episode PATCH status write-back. I saw both work.
  - **The server side is sound:**
    - the SQL checks the revision twice;
    - the 409 writes nothing;
    - the race tests run in a fixed order and fail if the request never waits.
  - **Verification 1's own case is fixed.** A section that was never saved, and whose save failed, stays at Start with Send disabled. Its answers are sent the next time it's left.
  - Every automated test passes.
- **What blocks:** finding 1. Verification 1's fix covers a section that was never saved, but not one that was already complete. A patient who changes an answer there while the network is down can still send the HQ, and the change is lost.

### Findings, worst first

1. **Major, older than S02a, but verification 1's fix says it handles it: a complete section whose change failed to save still counts as complete, and the HQ can be sent without the change.**
   - **Where:**
     - `apps/web/src/features/patient/PatientHQ.tsx:103`: `done` judges completeness only by what the server has (`saved`), so the old, complete answers count;
     - `:182`: **Send my answers** is enabled;
     - `:10-11`: the comment says "a set whose last save failed can't look complete or be sent", which isn't true here;
     - `:242`: the section closes after a failed save that isn't a conflict, as it always has.
   - **Reproduced in the browser:**
     - **Setup:** a 390 × 844 frame, on the made-up episode `S02a Verify22124a` (`c1a66f90-…`).
     1. The patient answered both sections, and every save got a 200. The list showed "2 of 2 sections complete".
     2. Saves started failing ("Failed to fetch"). The patient reopened **About you** (**Review**) and changed **Do you smoke?** from No to **Yes**. Both PUTs failed.
     3. They clicked **All sections**. About you still showed **Complete**, "2 of 2", and **Send my answers** was enabled. The only warning was the banner "TypeError: Failed to fetch".
     4. The network came back. They clicked **Send my answers** and confirmed. The submit got a 204, and the thank-you page showed.
     5. **The server froze `q_smokes: o_no`.** The patient's "Yes" was never sent, and can't be now.
   - **Why it matters:**
     - **A clinically meaningful answer is lost.** Here, the patient now says they smoke, and the hospital never sees it.
     - **Nobody is told:**
       - the patient saw "Saved" (`:276`) and a technical banner, then "Complete" and the thank-you page;
       - the clinician sees the old answer.
     - **It's likely on a phone:** a patient goes back to correct an answer on a weak signal.
     - **It's the same family as verification 1's finding,** and the code comment says it's handled.
   - **Before S02a:** the same loss happened (the page then counted completeness from its own copy, and the submit froze the server's). So it isn't a regression. But the fix for verification 1 was meant to close this path, and closing it is small.
   - **Fix (test-first):**
     - **The rule:** a set whose draft differs from what the server has isn't complete, and Send stays disabled until it's saved.
       - Leaving the section, or reopening it, already re-sends the draft (that part works).
       - Show the tile as **In progress** / **Continue**, not Complete.
     - **The test:** pull the "which sets are complete" decision into a small pure function in `packages/clinical`, and give it a Vitest test that fails first for this sequence.
     - **Also:** while the last save failed, show **Not saved** instead of **Saved** (`:276`, finding 2). The clinician page already does this.
     - **The test plan:** add the sequence above to T-12 (step 9: change a Complete section while offline, then leave it; it must not show Complete, and Send must stay disabled).

2. **Low, older than S02a: the patient page says "Saved" after a save fails, and the error text is technical.**
   - **Where:**
     - `PatientHQ.tsx:276` shows "Saved" whenever no request is in flight;
     - the banner shows the raw "TypeError: Failed to fetch".
   - **Why it matters:** it's what lets finding 1 go unnoticed.
   - **Fix:** fix the label with finding 1. For a network failure, a plain-language message such as "Your last change wasn't saved. Check your connection." could follow later; if it isn't done now, add it to pending.md §3.

3. **Low, new in S02a: if Reload fails on the clinician page, the page is stuck on the error.**
   - **Where:** `apps/web/src/features/episodes/ValidateSet.tsx:159` calls `onReload()` without waiting or catching. If the reload's GET fails, `useEpisode` / `useEpisodeHQ` set an error that nothing clears.
   - **Effect:** the page shows only the red error. Refreshing the browser recovers.
   - **Fix:** record it in pending.md §3. It's rare, and the way out is obvious.

4. **Low, new in S02a: the patient's Reload doesn't clear an old error banner, and has no loading state.**
   - **Where:** `PatientHQ.tsx:59-69` (`load`) and `:129` (the button).
   - **Effect:** a double tap loads twice, and a banner from an earlier failure can stay above the reloaded page.
   - **Fix:** record it, or clear `error` in `load` while fixing finding 1.

5. **Low, as the brief asked: the PATCH 409's wording promises more than it checks.**
   - **Where:** `apps/api/internal/httpapi/episodes.go:254`.
   - **The gap:**
     - the message says "This episode changed since you opened it", but the guard compares with the status this request loaded a moment earlier, not the one the user's page loaded;
     - so a stale POA Summary page isn't detected; only a race of milliseconds is.
   - **Why only Low:** this is exactly the brief's fix (and verification 1's finding 2 records the wider gap). A revision on the episode row, recorded in pending.md §1a, would make the message true.
   - **No change needed now.** Rahul may prefer to reword it ("This episode changed just now…").

**Considered and dismissed:**
- **"Leaving a section while its autosave is in flight sends the same answers twice"** (`PatientHQ.tsx:236`). This is true, but the saves queue, the second gets a 200 at the next revision, and nothing is lost.
- **Points that repeat verification 1's findings 2–5:** they're already recorded in pending.md §1a (ASA overwrite, the wrong 409 reason) or accepted (a missing `revision` is a 400).

### Checklist

| Check | Result |
| --- | --- |
| **Scope** | Everything in the brief and in Rahul's three answers is done: the revision on both answer routes, Reload in both pages, the PATCH guard, and the whole-row check in `TestUpdateEpisode`. The extras (skip unchanged patient saves; every 409 offers Reload; the two-copies patient page) are recorded in decisions.md and the build notes. |
| **Tests first** | **Clinician route:** the five cases, plus a new invalid-input set (missing, `null`, string, fractional or negative revision → 400). It also covers stale revisions with the whole row compared, X-1 itself, and two first saves (via `meanwhile`). **Patient route:** the same. **Vitest:** `answerSaver.test.ts` has 7 cases. Finding 1's path has no test yet. No `e2e/` yet (S03). |
| **Tenant isolation** | The new and changed queries are keyed by episode id after `loadEpisode` (hospital-scoped) or by the patient's token. In the browser, with a valid revision: no sign-in 401, Alex 403, Bea at A 404, Bea at B with A's ids 403. The row was unchanged. |
| **Nothing hardcoded for the UK (A-20)** | The grep prints only the older `en-GB` lines in `format.ts` and `OptionLists.tsx`, already listed in saas-requirements.md §13. Nothing new. |
| **Security** | No secrets, connection strings or patient data in the diff or logs. The patient JSON has `revisions` for the patient's own sets only, and no `STAFF`, `clinicianOnly`, `clinicalOutputs`, `testCases` or clinician question names (checked in the browser). The patient routes are unchanged in what they can reach. |
| **Docs** | F1 Status, `docs/plans/README.md` (F1: "S02a tested"), `docs/sessions/README.md` (S02a: Tested), decisions.md and pending.md are updated. The scrub grep prints nothing. |

### Test results

```
make lint   go vet -tags integration ./...  (no findings);  tsc --noEmit (no errors)
make test   ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 14 passed (14), Tests 110 passed (110)
make test-integration
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	6.175s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	4.257s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	4.380s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	4.684s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	12.714s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	5.023s
EXIT 0
```

### Browser spot-checks (Claude in Chrome, against `make dev`)

- **Setup:**
  - Chrome kept the tab "hidden" and wouldn't resize the window (2560 px), as both test reports found;
  - so the phone cases ran in a **390 × 844 same-origin frame**, and the clinician cases in two **500 px frames**;
  - steps were clicks on the app's own controls from script, with every result checked in the page and the request log;
  - "Network down" means the frame's answer PUTs failed with "Failed to fetch".
- **Test data:** two made-up episodes on `S02 test HQ 23:06`:
  - `S02a Verify22124a` (`c1a66f90-…`);
  - `S02a Verify22124b` (`695f2f77-…`).
  Both were created through the API as Cara.

1. **Finding 1, at 390 × 844:** reproduced as described above.
2. **Verification 1's fix (T-12), at 390 × 844, on episode b:**
   - with the network down, answering About you failed 3 times;
   - **All sections** showed About you at **Start**, "0 of 2", and Send was disabled;
   - with the network back, reopening showed both answers, and leaving sent them (PUT 200, revision 1);
   - then About you showed **Complete**, "1 of 2", and the server had the answers.
   - **Pass.**
3. **X-1 at 500 px, as Cara, on episode b after the patient sent it:**
   - tab A changed smoking to Yes. It showed "Patient answered: No", and the PUT got a 200;
   - stale tab B clicked **Validate**. One PUT got a **409**, and the **Not saved** box showed the exact message with **Reload** (label 40 of 40 px);
   - the status said Not saved, Validate was disabled, and tab B stayed on the set;
   - a click on the locked form sent nothing;
   - **Reload** showed **Yes**, with no box and no "Validated by". The row was A's answer, revision 1, not validated.
   - **Pass.**
4. **Permissions on the clinician save, with a valid revision:**
   - no sign-in 401, Alex 403, Bea at A 404, Bea at B with A's ids 403;
   - the row was unchanged.
   - **Pass.**
5. **The patient JSON:** `revisions` is present (`{}` before any save, then the set's revision), and there's no clinician content. **Pass.**
- **Console:** no errors.

### Suggested commits (after the fix)

Unchanged from verification 1 below, with finding 1's fix and its Vitest test in commit 2. Check that each commit passes on its own.

1. `Refuse an episode PATCH that would put back a status that just changed`
2. `Refuse a stale save of clinician or patient answers, with Reload in the browser`
3. `Docs: S02a build notes, test plan, test reports, verification and status`

---

## Verification 1 (9 Oct)

**Verdict: Needs fixes** (9 Oct 2026, about 23:40 BST).

- **Reviewed:** commit `b3876f9`, with S02a's work uncommitted.
- **Not fully independent:** this verification ran in the same Claude session that built S02a. To make up for that:
  - the code review ran as a separate agent;
  - every finding below was checked against the code;
  - finding 1 was reproduced in the browser.

  Rahul may want a fresh `/verify-session S02a` in a new session after the fix.
- **What's good:**
  - X-1 is fixed, and so is the episode PATCH status write-back.
  - The server side is sound: the SQL checks the revision twice, and the race tests run in a fixed order with no fakes.
  - Every automated test passes. Manual test report 1 passed all 15 cases.
  - The report's "Rel" button fix works.
- **What blocks:** one new bug in the patient page (finding 1). After a save fails for a reason other than a conflict, the patient's answers can be silently lost.

### Findings, worst first

1. **Major, new in S02a: after a failed save, the patient's answers are never sent again.**
   - **Where:**
     - `apps/web/src/features/patient/PatientHQ.tsx:197` (`const saved = useRef(data)`) and `:225` (the skip);
     - `data` comes from `hq.answers[open.id]`, which `onChange` (`:120`) overwrites with every unsaved change.
   - **How it happens (reproduced in the browser, 390 × 844, on a made-up episode `119a4525…`):**
     1. The patient answers "Do you smoke?" while the network is down. The autosave fails ("TypeError: Failed to fetch").
     2. They click **All sections**. The save fails again, and the section closes anyway, because the failure isn't a conflict (`:237`).
     3. The network comes back. They reopen the section (it shows the answer, and "Saved") and click **All sections**.
     4. **No request is sent.** `saved` starts from `hq.answers`, which already holds the unsaved answer, so the page thinks nothing changed.
     5. The server has no answers for the section (`GET /api/p/{token}` → `"answers": {}`), but the page shows it as In progress.
   - **Why it matters:**
     - **Frozen without the answers:** if the patient then finishes and clicks **Send my answers**, the HQ is frozen without that section's answers.
     - **Nobody is told:** the clinician sees no answers, and the patient was told "Saved".
     - **It's a regression:** before S02a, leaving a section always re-sent its answers, so the next successful save recovered them.
     - **It's likely in practice:** a dropped request on a phone is common.
   - **Fix:**
     - **The baseline:** compare with what the *server* has, not with the page's local answers. One way is for `PatientHQ` to keep the last answers the server confirmed for each set, set on load and on each successful save, and give that to `SetForm` as its baseline.
     - **Test-first:** pull the "does this need saving?" decision into a small pure function in `packages/clinical`, with a Vitest test for this sequence.
   - **Also consider (Rahul's call): don't close a section whose last save failed.** That's `:237`, and it's how the page already behaved before S02a. Keeping the section open with the error would stop a failed section from looking complete. Without that, the fix above still re-sends the answers the next time the section is left.

2. **Minor, older than S02a (not fixed by the brief): two simultaneous episode PATCHes can still overwrite each other's ASA grade.**
   - **Where:** `apps/api/internal/httpapi/episodes.go:246`. S02a's guard covers the status only; the handler still fills unsent fields from the row it loaded.
   - **How:** a nurse sets Nurse ASA 2 at the same moment an anaesthetist sets Anaesthetist ASA 3. Both load the row with the other grade empty, and the last write empties it again.
   - **Why only Minor:** the browser sends only the field that changed, so the window is milliseconds on the server.
   - **Fix:** an UPDATE that sets only the fields sent (`sqlc.narg` with `coalesce`), or a revision on the episode row. Recorded in pending.md §1a.

3. **Minor, new in S02a: a save whose response is lost can put the tab in conflict with itself.**
   - **Where:** `packages/clinical/src/answerSaver.ts:22`.
   - **How:** the server commits a save, but the browser gets a network error. The saver keeps the old revision, so its next save gets a 409, and the user sees "changed somewhere else" though no other tab exists.
   - **Why only Minor:** it isn't silent (the message and Reload appear), and Reload shows what was saved. Only edits made after the lost response are lost, and the message says so.
   - **When to fix:** with finding 1 if it's easy. Otherwise record it.

4. **Low, new in S02a: the wrong 409 message if the status changes during the save.**
   - **Where:** `patient.go:141` and `validation.go:106`. Both pick the message from the episode as loaded at the start of the request.
   - **How:** if the patient submits on another device, or another clinician completes the review, while a save is in flight, that save says "changed somewhere else" instead of "already sent" or "review complete".
   - **Why Low:** it's still a 409, and Reload shows the truth.

5. **Low, considered and accepted: a missing `revision` is a 400** (`validation.go:57`).
   - A browser that loaded the page before a deploy would get 400s. Nothing is deployed and there are no users, so this doesn't apply today.
   - Once there are users, API changes need a rollout plan; that belongs with F4's pipeline work.

6. **Low, simplification: `answerSaver` repeats the editor's save queue** (`useChapters.ts:52`).
   - Two copies of the same protocol means two places to fix.
   - The editor's queue is woven into its undo and redo, so merging them is a refactor for later. It isn't needed for S02a.

**Considered and dismissed:**
- **"Each autosave re-renders the patient page"** (`PatientHQ.tsx:121`). True, but it already re-rendered on every change (`onChange`). One more render per save, every 600 ms at most, doesn't matter.

### Checklist

| Check | Result |
| --- | --- |
| **Scope** | Everything in the brief and in Rahul's three answers is done. The two extras (skip unchanged patient saves; every 409 offers Reload) are recorded in decisions.md and the build notes. The first of them causes finding 1. |
| **Tests first** | The build notes record each new test failing first. The changed routes keep their five cases, with the revision added; the clinician route has a new "invalid input" set. Both race tests fail when their guard is removed (seen by the builder for the SQL re-check; I saw the PATCH test fail before its fix). The Vitest saver test has 5 cases. No `e2e/` yet (S03). |
| **Tenant isolation** | The new and changed queries are keyed by episode id after `loadEpisode`, which is scoped to the hospital, or by the patient's token. In the browser, with a valid revision: Alex → 403, Bea at A → 404, Bea at B with A's ids → 403, no sign-in → 401. The row was unchanged. |
| **Nothing hardcoded for the UK (A-20)** | Only the two older `en-GB` lines already listed in saas-requirements.md §13. |
| **Security** | No secrets or patient data in the code, tests or logs. The patient JSON now carries `revisions` for the patient's own sets only, and still has no `STAFF`, `clinicianOnly`, `clinicalOutputs` or `testCases`. |
| **Docs** | F1 Status, `docs/plans/README.md`, `docs/sessions/README.md`, decisions.md (9 Oct) and pending.md are updated. The scrub grep prints nothing. |

### Test results

```
make lint   go vet -tags integration ./...   (no findings);  tsc --noEmit (no errors)
make test   ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 14 passed (14), Tests 108 passed (108)
make test-integration
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	4.697s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	1.234s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	1.467s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	0.988s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	4.437s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	1.691s
exit 0
```

### Browser spot-checks

- **Test data:** made-up episode `S02a Verify2342` (`119a4525-8368-4337-94af-4ec3badb6660`), on "S02 test HQ 23:06".
- **Sign-in:** through the stub, checking `/api/me` before each step.

1. **Finding 1,** at 390 × 844 in a frame. Reproduced as described above.
   - Two failed PUTs while the network was down, then none at all after it came back.
   - The server's answers stayed `{}`.
2. **The report's "Rel" fix,** in a 500 px frame as Cara, with a real stale tab:
   - another "tab" saved first, through the API, and the frame then changed an answer;
   - the **Not saved** box appeared, with the full **Reload** label (40 of 40 px), and Validate was disabled;
   - **Reload** showed the other tab's **Yes**, unvalidated;
   - the server row was unchanged (revision 1, not validated).
3. **Permissions on the changed save routes,** sending a valid revision: Alex 403, Bea at A 404, Bea at B with A's ids 403, no sign-in 401. Nothing was written.
4. **The patient JSON:** `revisions` is present, and no clinician content.

### Suggested commits (after the fix)

1. `Refuse an episode PATCH that would put back a status that just changed`
   - **Files:** `episodes.sql` (`UpdateEpisode`), the generated `UpdateEpisode` hunks, `episodes.go`, `episodes_test.go`, `cast_integration_test.go`, `episodes_integration_test.go`.
2. `Refuse a stale save of clinician or patient answers, with Reload in the browser`
   - **Files:** the migration, the rest of the queries and the generated code, `validation.go`, `patient.go`, `validation_test.go`, `patient_test.go` (deleted), both integration tests, `packages/clinical` `answerSaver`, `hq.ts`, `ValidateSet.tsx`, `PatientHQ.tsx`, and the fix for finding 1 with its test.
3. `Docs: S02a build notes, test plan, test report, verification and status`

The build notes say how to split the generated code for commit 1.
