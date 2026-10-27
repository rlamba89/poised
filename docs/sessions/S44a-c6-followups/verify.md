# S44a verification

## Verification 3 (27 Oct 2026, automatic run; the extra round Rahul allowed)

**Verdict: Verified.** Verification 2's one finding is fixed test-first and holds under load: a reorder no longer deadlocks with a content save or a second reorder. Every earlier fix still holds. `/code-review high` found nothing new that blocks: three small wording and stamp issues are recorded in pending.md ("From S44a's verification 3"), and the rest were already there or were wrong.

### Verification 2's finding, checked

- **The reorder and save deadlock** (`apps/api/db/queries/chapters.sql`, `LockDraftVersion`): it now takes `FOR NO KEY UPDATE`, and the comment says why. Checked against the lock rules:
  - The reorder takes the version's row first and only then moves sets. So a save either finishes first (the reorder waits for its SHARE) or waits in its CTE before it touches its set. Neither side holds what the other needs, so they can't deadlock.
  - `TouchVersion` in the same transaction needs only the lock the reorder already holds, so there's no upgrade.
  - NO KEY UPDATE still conflicts with Publish's `FOR UPDATE`, so a reorder after Publish is still refused.
  - Two reorders now queue on the row instead of deadlocking.
  - A new set's foreign key takes KEY SHARE, which doesn't conflict.
- **The integration case** `TestPublishAsksTheEvaluator/a reorder while a Question Set save holds the version: it waits, then works` follows the order verification 2 described: SHARE held, the reorder waits, the "save" updates the moved set and commits, then 204 and the new order. It passes.
- **In the running app** (Chrome, API calls from the signed-in page, a new trust-wide draft as Alex):
  - **25 rounds**, each firing 6 requests at once: a reorder, a content save of each of the 3 sets, a rename and a second reorder. All 150 came back 204 or 200, with no 500. Afterwards each set was at revision 25, in the second reorder's order.
  - **Publish raced** 3 saves, a reorder and an audience change. Publish got 204. All five changes got 409 "Only draft versions can be changed.". The published sets kept their content, audience and order (checked in the database).
  - **Permissions on a new draft (v2) at Hospital A1:** Sue (A1-only super clinician) and Cara (clinician) got 403 on reorder, content save, rename, delete and add. At Hospital A2, which neither belongs to, they got 404, and so did Bea (Trust B). Nothing changed.

### Earlier fixes, still holding

- **Verification 1, findings 1 to 4 and 6** were reviewed in verification 2. The code is unchanged since then apart from the one-word fix: `dateIn`'s Gregorian calendar, `shownAnswers` repeating until stable, every Question Set change only into a draft, `notDraft` after a save refused by Publish, and `Result.Stored()`.
- **Verification 1's findings 5 and 7, and verification 2's minor items,** are in pending.md, and the duplicate "Built in S44a" on the S43 row is gone.

### Findings (minor, recorded in pending.md, not blocking)

1. **A published version's "last changed by" can name the author of a save that landed just before Publish** (`apps/api/internal/httpapi/chapters.go:290`, `touch` → `TouchVersion` has no draft check). The touch runs as its own statement after the save, so it can wait for Publish and then overwrite the publisher's `updated_by`. This is older than S44a, and it's only a stamp.
2. **A clinician's save that waited for a Complete review is refused with "someone else saved" wording** (`apps/api/internal/httpapi/validation.go:103`). The message comes from the status read before the wait. The save is still refused correctly, so nothing wrong is stored.
3. **Renaming or deleting a set another author just deleted says "Only draft versions can be changed."** (`chapters.go:140` and `:226`, where 0 rows is read as "published meanwhile"). The window is narrow, and it's only the wording.

**`/code-review high`, findings not taken:**
- The held message lost after Undo or un-archive, Publish's lock held through the evaluate call, and the efficiency items are already in pending.md for Rahul and S45.
- "`live` is not reset when the set changes" is wrong: the route keys `ValidateSet` by `eid/cid`, as verification 2 found.
- "`reachedOrPassed` reads the whole history for On hold and Not ready" is what the brief asks for (part 6).

### Checks

- **A-20:** the grep finds only comments and E.164 test numbers. `go test ./internal/repocheck` passes in `make test`.
- **Tenant isolation:** no new route. The changed queries keep their trust scope through the handlers' loaders, and the spot checks above confirm it.
- **Docs:** decisions.md, pending.md, CLAUDE.md, the C6 plan and both READMEs are updated. The rows are now set to Verified.

### Test results

- `make lint test test-integration`: **pass** (exit 0).
  - Vitest: web 288, evaluate 13, clinical 444.
  - CDK: 124 passed.
  - Go: every package ok, including httpapi's integration tests (26.9 s).
- `make e2e` against the conductor's `make dev`: **98 passed, 5 skipped** (7 min 35 s), with `test_audit` included.

### Hand checks for Rahul

- The patient's pages at 390×844 in a real resized window (the chain, T-04, T-12). The tester's window stayed at 500 px. Round 1 checked the thank-you page at 390×844 with Playwright.
- Drop `sj_before_s44a` once you're happy with the reset (the command is in build-notes.md).

### Commits

The build notes' ten commits by part can't be made cleanly without splitting hunks, because many files carry several parts (`scores.ts`, `respondent.ts`, `episodes.sql`, the generated `internal/db`). The build notes allow folding them, so the work is committed by layer instead, each commit with its tests:
1. `S44a: packages/clinical`: hidden score inputs add 0, keeping only what the form showed, the publish refusal, the authors' warnings, "also when" bands, the picker and the age in a time zone, with the shared fixtures.
2. `S44a: the API`: the send keeps only what was shown, the four verifiers' defects with the version and episode locks, the clearer messages and the hospital's time zone, with the integration tests.
3. `S44a: the web app`: the thank-you wording, the suggested status, the warnings, the suggested ASA on Validate, No ASA grade remembered and the "also when" editor.
4. `S44a: e2e journeys`: the audit CSV read as a stream, and the journeys' new checks.
5. `S44a: docs`.

Committed locally on 27 Oct (automatic mode, not pushed): 2279169, 13c6689, 4e477b3, d9c294d, and the docs commit that holds this file.

## Verification 2 (27 Oct 2026, automatic run)

**Verdict: Needs fixes.** One finding, and it is new: the fix for verification 1's finding 3 adds a deadlock between reordering Question Sets and saving a set's content (finding 1 below). It takes one word to fix and one integration case to test. Everything else from verification 1 holds: findings 1, 2, 3, 4 and 6 are fixed test-first, and findings 5 and 7, plus finding 2's question for Rahul, are recorded in pending.md.

### Findings, worst first

1. **A reorder deadlocks with a content save, or with a second reorder** (`apps/api/db/queries/chapters.sql`, `LockDraftVersion`; `apps/api/internal/httpapi/chapters.go:166` and `:186`).
   - `reorderChapters` takes the version's row `FOR SHARE` (`LockDraftVersion`), updates the chapters' positions, and then `UPDATE`s that same version row in the same transaction (`TouchVersion`). That `UPDATE` has to upgrade the lock, so it waits for every other `FOR SHARE` holder.
   - `SaveChapterContent` holds the version `FOR SHARE` (in its CTE) while it waits for its chapter's row. If the reorder has already moved that chapter, each statement waits for the other.
   - **Reproduced** in a scratch database (dropped afterwards) on the dev Postgres 17, with the same lock sequence and a 1 s gap:
     - With the reorder taking `FOR SHARE`, the save failed with `ERROR: deadlock detected`, and the reorder committed.
     - With the reorder taking `FOR NO KEY UPDATE`, the save waited, then `UPDATE 1`, and both committed.
   - Two reorders at once deadlock the same way: both take SHARE, A moves a chapter, B waits on it, and A's `TouchVersion` waits on B's SHARE.
   - **Why it matters:** Postgres aborts one of the two. Either the author's autosave fails with a 500 (`serverError("save content")`), or their drag fails with a 500. Before S44a neither statement locked the version, so this is a regression this session introduced. The window is short, but a debounced autosave landing just after a drag in the sidebar is ordinary editing.
   - **Fix, test first:** make `LockDraftVersion` take `FOR NO KEY UPDATE`, as `SaveClinicianAnswers` does for the same reason (builder's decision 4). It still conflicts with Publish's `FOR UPDATE`, so the reorder is still refused once the version is published. It serialises against saves' `FOR SHARE` rather than deadlocking with them. Fix the query's comment too.
   - **Integration case** (in the style of `TestPublishAsksTheEvaluator`'s race cases):
     1. A test transaction takes the version row `FOR SHARE`, as a save in progress would.
     2. Start the reorder and check that it waits.
     3. The test transaction then updates one of the chapters, as the save's own update would, and commits.
     4. The reorder should then return 204. As built, one side gets a deadlock error.
2. **Minor, record in pending.md (not blocking):**
   - **The wording when Complete review is refused** (`apps/api/internal/httpapi/validation.go:177`, `reviewNotOpen`). "The HQ review is already complete." is also what an episode on On hold or Not ready gets, and so does one auto-triaged past Ready for review, which nobody reviewed. The old `notInReview` wording was equally loose for those statuses, so this isn't a regression. A plainer line for them would be "This episode isn't waiting for review any more."
   - **Publish holds the version's row `FOR UPDATE` for the whole call to the evaluate service** (`questionnaires.go:226`). Every save to that draft waits for it, for up to the client's timeout. This is the brief's own design ("what's checked is what's published"), and answer 12 already says to measure the evaluate calls under locks in S45. Add Publish's call to what S45 measures.
   - **The `parsed` cache in `scoreAsShown` is a module Map that is never cleared** (`packages/clinical/src/scores.ts:123`). It's keyed by expression text, so in the long-running evaluate service it grows with every distinct score expression it sees. It's small in practice, and a Lambda (S45) recycles it. Bound it if S45 keeps a warm process.

Checked and fine (verification 1's fixes):
- **Finding 1, the calendar:** `dateIn` sets `calendar: "gregory"` and `numberingSystem: "latn"`, and takes an optional locale. The tests cover th-TH, fa-IR, ar-SA and ja-JP-u-ca-japanese.
- **Finding 2, the chain:** `shownAnswers` repeats `shownAnswersOnce` until nothing changes. Each pass only drops answers, and it stops after 50 passes. The B→A→C chain gives `{qB: "no"}` and doesn't change when run again. The shared fixture has `q_nap_hours`, and Go's `keepSent` still only removes keys.
- **Finding 3, other changes racing Publish:**
  - `CreateChapter`, `UpdateChapterMeta` and `DeleteChapter` each hold the version `FOR SHARE` and write only into a draft. Each takes a single statement, so there's no lock upgrade.
  - The reorder takes `LockDraftVersion`; that's finding 1 above.
  - A spot check in Chrome: Alex's PATCH and DELETE on a set of a published J10 HQ both gave 409 "Only draft versions can be changed.". Bea (Trust B) and Cara (another hospital) got 404, and Sue (A1-only super clinician) got 403.
- **Finding 4, the save message:** `saveContent` re-reads the version's status after no row, and answers `notDraft` when the version is no longer a draft.
- **Finding 6, the stored copy:** `form_outputs` gets `Result.Stored()`, which is the reply without `sent`. It's unit-tested, and the integration test checks the stored row.
- **`/code-review high`:** one suggested finding was wrong. It said the Validate page's `live` answers carry over between sets, but `ValidateSetPage` keys `ValidateSet` by `eid/cid`, so each set mounts afresh. Its efficiency findings are already in pending.md (verification 1, finding 7). The held message lost after Undo is pending.md's question for Rahul.
- **A-20:** the only added lines that match are E.164 test numbers in integration tests. `go test ./internal/repocheck` passes as part of `make test`.
- **Docs:** the sessions README and plans README say Tested, waiting for verification 2. pending.md's S43 row (line 53) says "Built in S44a" twice. Remove one when fixing.

### Test results

- `make lint test test-integration`: **pass** (exit 0).
  - Vitest: web 288, evaluate 13, clinical 444.
  - CDK: 124 passed.
  - Go: every package ok, including the integration tests (httpapi 33.5 s).
- `make e2e` against the conductor's `make dev`: **98 passed, 5 skipped** (6 min 50 s). `test_audit` passed too, with its stream reading.

### Spot checks

1. **Question Set changes on a published version, and the permissions** (Chrome, API calls run in the page): see "Finding 3" above.
2. **The reorder and save lock sequence**, reproduced in a scratch Postgres database: see finding 1.
3. **Not repeated, because test report 3 covered them with GIFs:** the patient's chain send and what Validate shows after it, a stale tab's audience change after Publish, and the suggested ASA on Validate.

### Hand checks for Rahul

- The patient's pages at 390×844 in a real resized window (the chain, T-04, T-12). Round 1 checked the thank-you page with Playwright.
- Drop `sj_before_s44a` once you're happy with the reset (the command is in build-notes.md).

### Suggested commits (once Verified)

As in verification 1: the build notes' list of commits 1–10, with the fix for finding 1 in commit 7 (the verifiers' defects, with the Publish race cases).

**For the builder:** "Verification found problems: read `docs/sessions/S44a-c6-followups/verify.md`, fix them test-first, then ask me for a retest."

## Verification 1 (27 Oct 2026, automatic run)

**Verdict: Needs fixes.** Two real defects in the clinical logic (findings 1 and 2). Each is small to fix, and each gives a patient a wrong result without anyone noticing. The rest of the session is sound. The brief's nine parts are all built, the tests are thorough, every test suite passes, and the spot checks passed. Findings 3 to 7 are minor: fix 3 and 4 while you're in there (they're cheap), and record 5 to 7 in pending.md.

### Findings, worst first

1. **The age is wrong in a browser whose locale uses another calendar** (`packages/clinical/src/respondent.ts:25`, `dateIn`).
   - `Intl.DateTimeFormat(undefined, …)` takes the browser's default locale, and with it that locale's calendar. `numberingSystem: "latn"` is pinned, but `calendar` isn't.
   - **Checked in Node:** today in `Europe/London` gives `th-TH` year **2569** (Buddhist calendar) and `fa-IR` **1405/8/5** (Persian). `en-GB` and `ar-SA` give 2026.
   - **Why it matters:** `hqPatient` now always passes a time zone (the hospital's or the deployment's), so for that clinician every view gets the wrong `{patientAge}`: about 543 years too old under th-TH, or a negative age under fa-IR. That changes "age over 50" points lines, age-conditioned Question Sets, the suggested ASA on Validate and the episode page, and the POA Summary. They then no longer match Go's result, which the evaluate service worked out. It's a regression: before S44a, `getFullYear()` was always Gregorian. One deployment per country (A-20) makes non-English locales a real prospect.
   - **Fix, test first:** add `calendar: "gregory"` to the options. Add a unit test that calls `dateIn`/`ageFrom` through a formatter with a Buddhist-calendar locale. Either pass a locale such as `th-TH` into a helper the test can reach, or test that `dateIn` gives the Gregorian year whatever the default locale is, for example by stubbing `Intl.DateTimeFormat`'s resolved locale.
2. **A chained follow-up's answer is frozen, but every later view hides it, and its flag no longer fires** (`packages/clinical/src/respondent.ts:173`, `shownAnswers`; used by `evaluate.ts:112`).
   - `shownAnswers` works out visibility once, on the answers as given (builder's decision 3). Take a set where B = yes shows A and A = yes shows C. The patient answers B yes, A yes, C "severe", then changes B to no.
   - On the patient's screen, A is hidden but its stale value keeps C shown. `shownAnswers` drops A and keeps C.
   - **Checked with a throwaway Vitest case:** given `{qB: "no", qA: "yes", qC: "severe"}`, the result is `{qB: "no", qC: "severe"}`. Running `shownAnswers` again on that gives `{qB: "no"}`, so the result isn't stable.
   - **Why it matters:**
     - The frozen row keeps C. Every later view (Validate, the POA Summary) works C's visibility out without A, so C is hidden and the clinician never sees the answer.
     - `evaluate()` now works its result out from these filtered answers, so C's red or amber flag, its disclosures and any rule on it don't fire, and auto-triage can go ahead.
     - Before S44a, the evaluate service saw A's stale value, so C was shown and its flag fired. For this edge case, S44a turns a flagged answer into one that's stored but invisible.
   - **Fix, test first (recommended):** repeat the drop until nothing changes. Each pass works out the conditions on the answers the last pass kept, so what's frozen is exactly what every view shows. That still follows Rahul's rule (an answer hidden by a condition is dropped) and stays in this one function. Unit test: the chain above gives `{qB: "no"}`, and `shownAnswers` run on its own result gives the same answers back. Add the chain to the shared `evaluate.json` fixture if that's cheap.
   - **For Rahul (pending.md, not this session):** the patient's own form still shows C while A's stale value stands, because SurveyJS keeps a hidden question's value until the survey is completed. So with the fix, an answer the patient saw on screen is dropped. The alternative is for the patient's form to clear an answer when its question is hidden (SurveyJS's `clearInvisibleValues: "onHidden"`). The screen and the stored answers would then always agree, but the patient would lose a follow-up's answers when they change the question it follows from. That's a product choice, so ask him.
3. **Other Question Set changes don't wait for Publish** (`apps/api/db/queries/chapters.sql:35-38` `UpdateChapterMeta`, plus `CreateChapter`, `DeleteChapter` and `SetChapterPosition`; `apps/api/internal/httpapi/chapters.go:117,129,203`).
   - Only `SaveChapterContent` takes the version's row `FOR SHARE` and saves only into a draft. The other writes check the draft status in Go first (`canEdit`), and then write with no status guard and no lock.
   - So a set's **audience** changed from clinician to patient, or a set deleted, while Publish runs can land after Publish's checks, or even after it has committed. A patient set that was never checked could then be published, which goes against "what's checked is what's published", the point of this fix.
   - The timing is narrow, so this is minor. **Cheap fix:** give those four queries the same `draft … FOR SHARE OF v` guard, and add an integration case for an audience change racing Publish, like the content-save case in `TestPublishAsksTheEvaluator`.
4. **A content save refused because Publish happened says "Someone else changed this chapter"** (`apps/api/internal/httpapi/chapters.go:253`).
   - Since S44a, `SaveChapterContent` also returns no row when the version was published while the save waited. `saveContent` turns every `ErrNoRows` into `staleRevisionMsg`, so the author reloads expecting someone else's edits.
   - **Cheap fix:** after no row, re-read the chapter's version status, as `rules.go` does, and answer `notDraft` ("Only draft versions can be changed.") when it's no longer a draft. Add an integration case, which can be the existing race case's assertion on the message.
5. **A held auto-triage message is now lost after an undo or an un-archive** (`apps/api/db/queries/episodes.sql:152`).
   - Clearing `triage_message_pending` on every change made by hand is what the brief asked for (S42's decision 6), and it fixes the double send. But some moves made by hand send no message of their own. Take an episode auto-triaged to Ready for POA with `messageAfterCheck`: put On hold, then Undo (or archive, then un-archive), and it's back at Ready for POA with nothing pending. Checked then sends nothing, so the patient is never told their status.
   - Not a defect against the brief. **Record it in pending.md for Rahul.** The likely answer is "keep the message pending when the change lands back on the auto-triaged status", but it's his choice.
6. **Data minimisation: `form_outputs.result` now holds a full copy of the patient's sent answers** (`apps/api/internal/evaluate/evaluate.go:88`, stored by `evaluateSent` via `AddFormOutput`).
   - Go reads `sent` once, in `keepSent`, and nothing reads it back from `form_outputs`. The copy widens where patient answers live, which matters for subject access, retention and deletion, and later corrections never touch it.
   - The builder chose "stored as it came". **Suggested:** drop `sent` from `Raw` before storing it, or record it in pending.md as a question for Rahul.
7. **Minor efficiency, for pending.md:**
   - Each send builds the patient-view surveys about three times: `shownAnswers` → `hqSets`, then once more per set, then `hqSets` again for `patientSets`. `hqSets` already works out each set's hidden names, so it could return them.
   - Validate reruns `hqSuggestedAsa` over the whole HQ on every answer change (`ValidateSet.tsx:103`). This is builder's decision 9, and is fine at today's sizes.
   - The Publish window sends one `/codes` search per distinct code (`PublishModal.tsx:70`). The warning also quietly disappears if the search doesn't return that exact code.
   - `keepSent` re-reads `ListEpisodeAnswers`, which `evaluateSent` loaded a moment before.

Checked and fine:
- **Scope:** every part of the brief, 0 to 9, is built as asked. The builder's differences are sensible and recorded:
  - The old database was renamed, not dropped.
  - `SaveClinicianAnswers` takes `FOR NO KEY UPDATE`, not `FOR SHARE`, because of the deadlock with the touch trigger. The reasoning holds.
  - The content-save guard was added.
  - A band's "also when" inputs count as score inputs.
- **Go never evaluates:** `keepSent` only removes the keys the service left out, never adds or changes an answer, and keeps everything when the service fails or names no set (S41's fail-safe).
- **The pre-fill entries:** they're dropped with their answers, `{}` stays `{}`, and `updated_by` is cleared when nothing is left (test report 1's F1).
- **The SQL:** the `FOR SHARE` CTE in `SaveChapterContent` and the `FOR NO KEY UPDATE OF e` on `INSERT … SELECT … ON CONFLICT` are valid, and the integration tests cover both orders.
- **The new wordings:** `triageChecked` checks "already checked" before the revision check, so a stale second tab gets "Someone checked this while you had it open.", as test report 2 confirmed. `reviewNotOpen` keeps `notInReview`'s wording for HQ not complete, cancelled and archived.
- **`suggestionLines`/`reachedOrPassed`:** they follow answer 6, and `status.test.ts` covers the main path, the history route for On hold and Not ready, and an undone change.
- **The beforeExpressionRunning wrap:** SurveyJS's prototype is wrapped once, inside `registerClinicalFunctions`, and the original is still called first. `parsed` caches by expression text.
- **Tenant isolation:** no new route was added. `GET …/episodes/{eid}` gains `hospitalTimeZone` through a join on the episode's own hospital.
- **A-20:** the grep finds only comments and E.164 examples. The time zone comes from the API or `deploymentTimeZone()`, never a literal.
- **Repo check:** passes, as part of `make test`.
- **Docs:** decisions.md (27 Oct) and pending.md (items struck, pointing here) are updated, and the plan, plans README and sessions README are at "Tested".

### Test results

- `make lint test`: **pass**. Vitest: web 288, evaluate 13, clinical 442. CDK: 124 passed. Go unit tests: all ok.
- `make test-integration`: **pass** (every package ok; httpapi 22 s).
- `make e2e`: **not run this round.** The verdict was already Needs fixes, and the conductor's `make dev` was serving the builder's code. The builder's run after test report 1 was 97 passed, 1 failed, 5 skipped, the failure being `test_audit`'s hover tooltip, which passed when run alone. Verification 2 runs the full suite.

### Spot checks (Chrome, the conductor's `make dev`)

1. **Another trust or hospital's episode:** an A1 Ready-for-review episode. The tests were API calls run in the page.

   | Who | `GET …/episodes/{eid}` | `POST …/triage-checked` |
   | --- | --- | --- |
   | Cara (A1) | 200, `hospitalTimeZone: null` (the deployment's) | not tried |
   | Bea (Trust B) | 404 | 404 |
   | Val (A2) | 404 | 404 |

2. **`dateIn` under other calendars:** Node's `Intl` gives th-TH 2569 and fa-IR 1405, which is finding 1.
3. **`shownAnswers` on a three-question chain:** a throwaway Vitest case, since deleted, gave the unstable result in finding 2.
4. **Already covered by the tester, so not repeated:**
   - the thank-you words, at 390×844 in round 1 (Playwright);
   - the Rules page and Publish warnings;
   - Publish refusing a patient condition on clinician-only content;
   - the stale second-tab Complete review;
   - the held message going once.

### Hand checks for Rahul

- The patient's pages at 390×844 in a real resized window (T-04, T-12). Round 1 checked the thank-you page with Playwright.
- Drop the old local database `sj_before_s44a` once you're happy with the reset (the command is in build-notes.md).

### Suggested commits (once Verified)

The build notes' list stands: commits 1–9 by part, then 10 for docs. Add the fixes for findings 1–4 to the commits they belong to (1 → part 8, 2 → part 2, 3 and 4 → part 7).

**For the builder:** "Verification found problems: read `docs/sessions/S44a-c6-followups/verify.md`, fix them test-first, then ask me for a retest."
