# S02a build notes

**Builder session:** 9–11 Oct 2026. **Status:** verification 2 said Needs fixes; finding 1 is fixed. Next: `/manual-test S02a retest` in a new session, then `/verify-session S02a`.

## Before building

- **S02:** Verified and committed (`9e2c574`, `71abc99`, `b3876f9`). The tree was clean apart from `reports/` and `research_notes/`, which are untracked on purpose.
- **Rahul's answers (9 Oct),** recorded in decisions.md §2 and in the brief:
  1. **The patient's autosave gets the same check:** yes.
  2. **Also fix the episode PATCH status write-back** (S02's verify.md item 2): yes, in its own commit.
  3. **S03's J4 gains a two-tab step:** yes. Added to S03's brief and to F1's journey table.

## Steps

1. [x] **The episode PATCH can't put back a status that just changed.**
   - **Test-first:** `TestUpdateEpisodeKeepsAStatusThatJustChanged` failed with a 204 (the old status put back) for both races: the review being completed, and the patient's submit.
   - **The fix:** `UpdateEpisode` now has `AND status = <the status the handler loaded>`. No row → 409: "This episode changed since you opened it. Reload to see the latest; your change was not saved."
   - **Also closed S02's test gap** (its verify.md item 4): after its refusals, `TestUpdateEpisode` now compares the whole row with the row before. A new case covers changing only the details before the review.
2. [x] **Answer saves carry a revision; a stale one gets a 409** (clinician and patient).
   - **Migration** `00005_answer_revisions.sql`: `episode_answers.revision int NOT NULL DEFAULT 0`. Up, down and up again were checked on the dev database.
   - **Reading:**
     - `GET …/episodes/{eid}/hq` returns each row's `revision`;
     - `GET /api/p/{token}` returns a new `revisions` map (set id → revision), for the patient's own sets only.
   - **Saving:** the body is `{data, validated, revision}`.
     - A missing, `null`, non-number or negative `revision` → 400.
     - On success → `200 {"revision": n}` (was 204).
     - On a stale revision → 409, and nothing is written.
   - **One SQL statement per save,** checking the revision twice:
     - the `INSERT … SELECT` runs only if `revision` equals the row's current revision (or 0 when there's no row);
     - `ON CONFLICT … DO UPDATE … WHERE episode_answers.revision = <revision>` checks it again against the row as it is at that moment, so of two saves sent at once, only the first is applied.
   - **Which 409 message:**
     - **clinician:** "The patient hasn't sent…" or "The HQ review is already complete…" if the episode isn't Ready for review, else the stale message;
     - **patient:** "Your answers have already been sent…" once submitted, else the stale message;
     - **the stale message** is the brief's: "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved."
3. [x] **Browser:** both answer pages send the revision, stop saving after a 409, and offer **Reload**. The TypeScript is explained below.
4. [x] `make lint test test-integration` passes. Test plan written.

## Tests

**Integration** (API → database). Each was seen failing first, for the expected reason.

- **`TestSaveClinicianAnswers`:**
  - the first save makes the row at revision 1, and each save after that returns the next revision;
  - **a stale revision → 409, and the row is unchanged** (data, stamp, by, revision). Three stale cases: an older revision, a newer one, and 0 when a row exists;
  - **a revision sent for a set with no row** → 409, and no row is made;
  - **the X-1 sequence:** tab A saves from revision n; tab B validates from n → 409. A's answer is kept, unvalidated, at revision n + 1;
  - **two first saves at once** → the second gets a 409 (see `meanwhile` below);
  - a missing, `null`, string, fractional or negative revision → 400;
  - the five cases and the old rules (only while Ready for review; the stamp cleared by a later save) still pass.
- **`TestSavePatientAnswers`:**
  - the same checks, for the link open on two devices;
  - a save after submit still gets "already been sent".
- **`TestEpisodeHQ` and `TestPatientHQ`:** the revisions are returned. The patient gets none for a set they haven't answered, and none of the clinician's.
- **`TestUpdateEpisodeKeepsAStatusThatJustChanged`:** the PATCH race, through the real API.

**`meanwhile`** (`cast_integration_test.go`) makes a race happen in a fixed order. No fakes and no sleeps-and-hope:
1. It writes the conflicting change in a transaction it doesn't commit yet, so Postgres keeps that row locked.
2. It sends the request. The request reads the old row, then waits for the lock when it writes.
3. Once `pg_stat_activity` shows the request waiting, it commits.
4. It returns the request's response.
- The test fails if the request never waits, so it can't pass by accident.
- **Checked:** with the `ON CONFLICT … WHERE` re-check removed, only "two first saves at once" fails, with a 200 instead of a 409. So both halves of the SQL are needed.
- The `httpapi` integration package passed 3 runs in a row.

**Unit (Vitest):** `packages/clinical/src/answerSaver.test.ts`, 5 tests:
- each save sends the last revision;
- a set with no answers starts at 0;
- an autosave and Validate sent together run one after the other;
- after a 409 nothing more is sent;
- other errors aren't treated as conflicts.

**Old fake-querier tests removed** (testing rule 4: replaced when their area is next touched). Everything they checked is in the integration tests:
- `TestUpdateEpisode` in `episodes_test.go`;
- `TestSaveClinicianAnswers` in `validation_test.go` (`TestCompleteReview` stays: that route didn't change);
- all of `patient_test.go`. Its cases (bad tokens, stripping, dropped keys, the clinician set → 404, the submit event, frozen after submit) are in `patient_integration_test.go`.

## The TypeScript (for Rahul)

**`packages/clinical/src/answerSaver.ts`** is new, about 40 lines. It's in `packages/clinical` because that's where Vitest runs, and both answer pages use it. It has no React.
- `answerSaver(startRevision, send)` returns an object with `save(body)`.
- **Saves run one at a time.** Each `save` waits for the previous one to finish, then sends the revision the previous one returned.
  - **Why:** without this, a tab's own autosave and Validate could be in flight together with the same revision, and the second would get a 409 from its own tab.
  - It's the same queue idea the editor already uses (`useChapters.ts`).
- **After a 409 it stops:**
  - every later `save` returns `{ok: false, conflict: true, message}` without sending anything;
  - other failures (a network error, a 500) return `conflict: false`, and the next save is still sent;
  - it spots a 409 by the error's `status`, which the web app's `ApiError` carries.

**`ValidateSet.tsx`** (clinician):
- **Saving:**
  - it makes one saver per set, starting from the row's revision (0 if the clinician has no row);
  - autosave and **Validate** both go through it.
- **On a conflict:**
  - the form becomes read-only;
  - **Validate** is disabled, and the status says **Not saved**;
  - a red **Not saved** box shows the message and a **Reload** button.
- **Reload** fetches the episode and the answers again, and rebuilds the page: React's `key` changes, so the old one is thrown away.

**`PatientHQ.tsx`** (patient):
- **Revisions:**
  - the page reads `revisions` from `GET /api/p/{token}`;
  - each section's saver starts from its revision and lasts until Reload, so reopening a section carries on from the last save (changed after verification 1, below).
- **On a conflict:**
  - the section is replaced by a card, "Your answers weren't saved", with the message and **Reload**;
  - Reload fetches the HQ again and reopens the same section with the latest answers;
  - a 409 for "already sent" shows the same card, and Reload then shows the thank-you page. This replaces the old "Reload the page to see them." text added to the error.
- **Two smaller changes, needed for the check to work properly:**
  - **Leaving a section without changing anything no longer saves** (if the server already has those answers; changed after verification 1). Before, "All sections" and "Finish this section" always saved. Without this change, a tab left open on another device would either get the conflict message for doing nothing, or (before S02a) quietly overwrite newer answers with stale ones. The clinician page already skipped unchanged saves.
  - **A save still waiting when a section closes is cancelled,** so it can't run later and bring the conflict card back after Reload.
- **Not done on the clinician page:** cancelling there would lose a change made just before clicking the back link, which today is still saved.

`hq.ts`: the answers row type gains `revision`.

**Changed after verification 1** (finding 1, below): the patient page now keeps two copies of the answers apart.
- **What the server has:**
  - the answers loaded with the HQ, then each save the server accepts;
  - each set has one saver for the life of the page (until Reload), kept in a `useRef` map, and it tracks these (`saver.confirmed`) together with the set's revision;
  - **the section list, "Complete", "In progress" and "Send my answers" go by these.**
- **What's on screen** (`drafts`): what an open set shows.
- **A set saves whenever its draft differs from what the server has.** So a failed save is sent again the next time the set is left, and a set that's simply left unchanged sends nothing.
- `answerSaver` gained `confirmed` for this: it starts as the loaded answers, and changes only when a save succeeds.

## Differences from the brief

- **The patient page skips saves when nothing changed** (above). It's a small behaviour change the brief didn't list, but the revision check needs it: otherwise every stale tab that's simply closed would get the message.
- **Every 409 on an answers save stops autosaving and offers Reload,** not only the stale-revision one. That includes "the review is already complete" (clinician) and "already sent" (patient). In each case the page is out of date, and Reload shows the truth.
- **The race tests use `meanwhile`** rather than two real concurrent HTTP requests, which would pass or fail by timing.

## Test report 1 (9 Oct): all passed

[test-report-1.md](test-report-1.md): 11 test cases and 4 regression checks passed, including X-1 exactly (T-07). It also found three things while exploring:

1. **Fixed (cosmetic, new in S02a):** in a window about 500 px wide, the **Reload** button in the clinician's Not saved box shrank to "Rel".
   - **The fix:** the button no longer shrinks (`flexShrink: 0`, `ValidateSet.tsx`).
   - **Untested by automation:** there's no automated test for layout, so the verifier should check it at about 500 px.
2. **Intended:** after a refused clinician save, the page keeps showing the unsaved answer (and any summary built from it) until Reload.
   - **Why:** the form is locked, Validate is disabled, and the red box says the change wasn't saved.
   - Reloading on its own would throw away what the clinician typed before they've read why.
3. **Older than S02a, recorded in pending.md §3:** the editor says "All changes saved" while a title edit is still in the focused field, because it saves on blur.

## Verification 1 (9 Oct): Needs fixes

[verify.md](verify.md).

- **Finding 1 (Major, new in S02a): after a failed save, the patient's answers were never sent again.**
  - **The cause:** the "skip unchanged saves" check compared with the page's own copy of the answers, which already held the unsaved answer.
  - **Fixed:**
    - the check now compares with what the server has accepted (`answerSaver`'s new `confirmed`);
    - the section list, "Complete" and "Send my answers" now go by what the server has, so a set whose save failed can't look complete or be sent.
  - **Test-first:** two Vitest cases in `answerSaver.test.ts` failed first:
    - `confirmed` changes only when the server accepts a save;
    - a refused save doesn't change it.
  - **Seen in the browser,** at 390 × 844, with failing PUTs while answering and leaving a section:
    - offline, the section stayed "Start", "0 of 2 sections complete", and Send was disabled;
    - back online, reopening showed the answers, and leaving sent them (PUT 200);
    - then it showed Complete, and the server had them.
  - **Re-checked with two "devices":** one device's own saves across closing and reopening a set don't conflict; a stale device still gets the card; after Reload its next change saves.
  - **The verifier's "keep a failed section open?" question** isn't needed now: the section closes as before, but it can't count as complete or be sent until a save succeeds.
- **Findings 2–4 (Minor or Low), recorded in pending.md §1a, not fixed here:**
  - **2:** two simultaneous ASA-grade PATCHes. Older than S02a.
  - **3:** a save whose response is lost makes the tab conflict with itself. It isn't silent: the message appears.
  - **4:** the 409's wording when the status changes during the save.
- **Findings 5–6:** accepted by the verifier. No change.

## Test report 2 (11 Oct): all passed

[test-report-2.md](test-report-2.md) covers the retest list: T-03 to T-07, the new T-12, R-01 and R-02.

- **T-12** confirms verification 1's fix in the browser:
  - a failed save keeps the section at Start, and Send disabled;
  - the answers are re-sent when the section is next left, and only then does it show Complete.
- **T-07** (X-1) still passes. The clinician page's locked form also refused a real change and sent nothing.
- **No code changed after it.**
- **Two caveats, both stated in the report:**
  - **Not independent:** the retest ran in this builder session (Rahul ran it here).
  - **Scripted clicks:** Chrome kept the tab "hidden", so the steps were scripted clicks on the app's own controls, with every result checked in the page and the request log. The "Send your answers?" dialog couldn't show, so its request was sent directly.
- **Recommendation:** run the verification in a **new** Claude session. This builder session also ran verification 1 and test report 2.

## Verification 2 (11 Oct): Needs fixes

[verify.md](verify.md), "Verification 2". It was run independently, in a new session.

- **Finding 1 (Major, older than S02a; verification 1's fix claimed to cover it): a complete section whose change failed to save still counted as complete,** and the HQ could be sent without the change.
  - **Fixed:**
    - a set counts as complete only when the server's answers are complete **and** what's on screen has been saved (`isSetReadyToSend` in `packages/clinical/src/respondent.ts`);
    - the list shows such a set as In progress, and **Send my answers** stays disabled;
    - the header comment in `PatientHQ.tsx` now says this accurately.
  - **Test-first:** `respondent.test.ts`, "is ready to send only when the server's answers are complete and nothing on screen is unsaved", failed first (5 cases).
  - **Seen in the browser,** at 390 × 844, with the verifier's sequence:
    1. Both sections complete (2 of 2). Saves failing, smoking changed in About you: the label says **Not saved**.
    2. Leaving it: About you is In progress, 1 of 2, and Send is disabled.
    3. Back online, reopening shows the change (still Not saved). Leaving sends it (PUT 200).
    4. Then it's Complete, 2 of 2, and Send is enabled. The server has the change.
- **Finding 2 (Low):**
  - **Fixed:** the patient's save label now says **Not saved** (red) after a failed save, or when a reopened set has an unsaved change.
  - **Recorded in pending.md §3:** the raw "Failed to fetch" text. The fix belongs in `lib/api.ts`, for every page.
- **Finding 3 (Low):** recorded in pending.md §3 (the clinician page is stuck if Reload itself fails).
- **Finding 4 (Low), fixed:** the patient's Reload clears an earlier error message, and shows it's loading (which also stops a double tap loading twice).
- **Finding 5 (Low):** no change, as the verifier advised. It's recorded with the ASA item in pending.md §1a, since an episode revision would fix both.

## Test results (11 Oct, after verification 2's fix)

`make lint test test-integration` (exit 0): no vet or tsc findings; Vitest 14 files, **111 tests** passed; every Go package `ok`, with and without `-tags integration`.

## Test results (9 Oct, after the fix)

`make lint test test-integration` (exit 0):

```
go vet -tags integration ./...      (no findings)
tsc --noEmit                        (no errors)
make test:  ok auth, chapter, episode, httpapi, hqimport;  Vitest: Test Files 14 passed (14), Tests 110 passed (110)
make test-integration:
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	5.601s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	0.829s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	1.292s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	1.969s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	5.260s
ok  	github.com/rlamba89/poised/apps/api/internal/hqimport	1.738s
```

- **`make dev` must be restarted** after this session: it runs the API with `go run`, so a running one still has the old code.
- **The dev database is already migrated to version 5** (this session ran `make migrate`).

## Suggested commits

In this order.

1. **Refuse an episode PATCH that would put back a status that just changed**
   - `apps/api/db/queries/episodes.sql` (`UpdateEpisode` only);
   - the generated `internal/db/episodes.sql.go` and `querier.go` (the `UpdateEpisode` hunks);
   - `internal/httpapi/episodes.go`, `episodes_test.go` (fake test removed);
   - `cast_integration_test.go` (`meanwhile`) and `episodes_integration_test.go`.
2. **Refuse a stale save of clinician or patient answers, with Reload in the browser**
   - **Database:** `db/migrations/00005_answer_revisions.sql`, the rest of `episodes.sql` and the generated code;
   - **Go:** `validation.go`, `patient.go`, `validation_test.go`, `patient_test.go` (deleted);
   - **Tests:** `validation_integration_test.go`, `patient_integration_test.go`;
   - **Clinical package:** `packages/clinical/src/answerSaver.ts`, `answerSaver.test.ts`, `index.ts`;
   - **Web:** `apps/web/src/features/episodes/hq.ts`, `ValidateSet.tsx`, `features/patient/PatientHQ.tsx`.
3. **Docs: S02a build notes, test plan, status and decisions**
   - `docs/sessions/S02a-answer-revisions/`, `docs/sessions/S03-e2e-journeys/brief.md`, `docs/sessions/README.md`;
   - `docs/plans/f1-test-foundation.md`, `docs/decisions.md`, `docs/pending.md`.

**How to split the generated code:** commit 1 needs only the `UpdateEpisode` hunks of `episodes.sql.go`, `querier.go` and `episodes.sql`. The simplest way:
1. Stash the other query changes.
2. Run `make sqlc`.
3. Commit 1.
4. Restore the changes and run `make sqlc` again for commit 2.

The verifier should check that each commit passes on its own.
