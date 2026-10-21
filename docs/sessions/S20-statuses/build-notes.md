# S20 build notes: every status, cancel and re-open

Builder, automatic run (`/run-sessions`), 21 Oct 2026.

## Answers (from [answers.md](answers.md), asked by the conductor on 20 Oct; also decisions.md §2)

1. **The transition table:** accepted as in the brief.
2. **Correcting a mistake:** **any clinician may undo any manual move**, one step at a time, with a reason, and again to go further back. Cancel stays final.
3. **Cancel reasons:** a fixed list (procedure cancelled, patient withdrew, moved to another hospital, other); "Other" needs a sentence. Final.
4. **Reasons:** required for cancel, re-open and undo; optional for the other moves by hand.
5. **Cancelling and the patient:** the invite links and the sessions opened from them end, follow-up jobs stop, and their home shows "Cancelled by the hospital", with no form.
6. **Re-open:** clears the review stamp and unlocks the clinician's answers; the answers and validations stay; back to Ready for review; the patient isn't asked again.

## What was built

- **The transition table in Go** (`internal/episode/episode.go`): `allowed[(from, to)] → Rule{Move, Who (least role), Patient, NeedsReason, NeedsDate}`.
  - `Check(from, to, move, role)` → the rule, `ErrIllegal` (409) or `ErrRole` (403). `role.None` is the patient (only their submit).
  - `Options(from, role)`: the moves the status control offers this user (not submit or complete-review, which have their own buttons).
  - `UndoTarget(current, latest)`: where an undo goes; `CancelReason(key, details)`: the reason as recorded ("Other: Emigrated").
  - The patient's submit, staff submitting on their behalf, and Complete HQ review go through `Check` too.
- **Migration `00021_episode_statuses.sql`:**
  - `episodes.status` CHECK adds `admitted`, `discharged`, `cancelled`;
  - `episodes` gets `revision` (int, 0), `admitted_at` and `discharged_at` (**dates**, as entered), `cancel_reason` (the list's key, CHECKed);
  - **`episode_status_events`** (the status history, EPI-03): id, `seq` (identity, exact order), episode_id, move, from_status (NULL only for create), to_status, reason, user_id (NULL for the patient and automatic changes), rule (an automatic change's rule, C6; NULL now), undone_at, created_at.
  - Down maps the new statuses to On hold rather than losing episodes.
- **The episode revision:** bumped by every change staff make (details PATCH, every status move) and by the workflow (the submit, completing the review). **Not** by the jobs' bookkeeping or a resent invite: they change nothing a page edits. Writes send the revision the page loaded; a stale one gets **409 with `code: "stale"`** ("This episode changed since you opened it. Reload to see the latest; your change was not saved."). This fixes both pending.md §1a items (two PATCHes overwriting each other's ASA grade; the 409 that compared the wrong status).
- **Routes** (`internal/httpapi/statuses.go`, each wrapped in `audited`):
  - `PATCH …/episodes/{eid}/status` `{to, reason, date, revision}` (`episode.status`): a clinician's pick, Admit (admission date) and Discharge (discharge date);
  - `POST …/episodes/{eid}/cancel` `{reason (key), details, revision}` (`episode.cancel`);
  - `POST …/episodes/{eid}/reopen` `{reason, revision}` (`episode.reopen`);
  - `POST …/episodes/{eid}/undo` `{reason, revision}` (`episode.undo`), **not in the brief's list**: answer 2 needs a route.
  - Each runs in one transaction: lock the episode, check the revision, check the table against the status as it is now, write the status, the history row and a General notes line, plus what goes with the move.
- **`PATCH …/episodes/{eid}`** (details and ASA grades) **no longer takes `status`** (a body with one is 400, so an old page can't think it saved one), requires `revision`, and refuses a cancelled episode (409).
- **`GET …/episodes/{eid}`** adds `moves` (episode.Options for this user), `undo` (`{to, label}` or null) and `statusHistory`; the episode adds `revision`, `admittedAt`, `dischargedAt`, `cancelReason`.
- **Creating an episode** records its first status (`create`) in the history. Episodes made before S20 have no history.
- **Cancel, the patient's side:** the episode's invite links are revoked (so `TouchPatientSession` ends the sessions opened from them), its open jobs are finished, `GetFollowup`, `CountReminder` and `MarkNotResponding` skip a cancelled episode (in case a runner had already claimed the job), Resend invite is refused (409), the patient's form (and staff's on-behalf form) answers **403 with `code: "cancelled"`** ("This questionnaire was cancelled by the hospital…"), and `GET /api/p/home` gives the episode the state `cancelled`.
- **Web:**
  - `StatusControl.tsx` replaces `StatusSelect.tsx` (episode page and POA Summary): the **Episode status** dropdown lists only this user's picks (plus Admitted / Discharged when legal); **Re-open HQ review**, **Undo: back to …** and **Cancel episode** buttons appear only when allowed. Every move opens a short **inline form** (no modal): the date for Admitted and Discharged (today first), the reason (required for cancel, re-open and undo; "Reason (optional)" otherwise), the cancel reason as radio buttons with "Say why" for Other. A stale page's refusal keeps what was typed, with **Reload**.
  - **Status history** card on the episode page (when, change, by, reason; "(undone)" on stepped-back rows); a grey **This episode is cancelled** alert with the reason; Admission date and Discharge date in the details; Edit procedure and dates hidden once cancelled.
  - **ASA grades** (POA Summary) send the revision; a stale save keeps the grade typed, with Reload; after a reload, a grade nobody is changing here shows the saved one.
  - **Patient home:** "Cancelled by the hospital", a sentence saying there's nothing to fill in, and no button.

### TypeScript Rahul should know about

- `features/episodes/status.ts` (unit-tested in `status.test.ts`): the API's shapes (`StatusOption`, `UndoOption`, `StatusEvent`), the cancel reasons list, `pickOptions` (which moves the dropdown lists), `statusRequest` (a filled-in form → the path and body to send, with the revision, or the first problem in the API's words), `historyText` / `historyWho` (how a history row reads). **Which moves are allowed, and to whom, is Go's**; the page shows only what `moves` and `undo` say.
- `StatusControl.tsx`: `isStale(error)` reads the 409's `code: "stale"`; `SaveError` shows a refusal, with Reload when stale.
- `features/patient/home.ts`: the `cancelled` state (no action, so no button).

## Decided by the builder (automatic mode)

- **"Any other of these five" includes Ready for POA as a target** (the accepted table, read literally): a clinician can move On hold back to Ready for POA by hand. Before S20 Ready for POA couldn't be picked; the table Rahul accepted lists the five as both sources and targets.
- **What undo can step back:** a pick, Admit and Discharge (the moves made by hand). Not the workflow's (submit; completing the review: re-open is for that), not re-open, not cancel (final), not an undo (no redo). The stack stops at the first of those. Undoing an admission or a discharge clears its date.
- **Undo is its own route** (`POST …/undo`), with the five cases; the brief listed only status, cancel and re-open, before answer 2.
- **Admin may re-open** too (the ladder: admin ≥ super clinician), as may a super clinician of that hospital only (Sue Super).
- **Dates:** the admission and discharge dates are required with their move, can't be in the future (the hospital's today), and the discharge date can't be before the admission date. Stored as `date` columns named `admitted_at` / `discharged_at`, as the brief names them.
- **Cancel details are optional for the three fixed reasons** and required for Other; the recorded reason is "Label" or "Label: details". `cancel_reason` keeps the key; the words are in the history.
- **A cancelled episode is read-only except for notes:** status moves, re-open, undo, the details PATCH, the clinician's answers and Resend are refused (409); General notes can still be added (e.g. a call with the patient).
- **The revision is required** on every staff write that a page makes (the details PATCH and the four status routes), not optional: nothing silently skips the check. **Complete HQ review doesn't send one**: its own guard is the status (Ready for review only), unchanged.
- **Messages:** illegal moves say where the episode is now ("This episode is Discharged, so its status can't be changed to On hold."); a role too low for re-open says "Only a super clinician or an admin can re-open the HQ review."
- **General notes lines** (still written for every change): "Status changed to On hold[: reason]", "Status changed to Admitted (admission date 2026-01-10)", "Episode cancelled: Patient withdrew", "HQ review re-opened: reason", "Status change undone, back to Admitted: reason"; the submit and the review keep their old lines.
- **The status forms are inline, not modals** (Claude in Chrome's hidden-tab gotcha, §4), with `noValidate`, and the date field checks `validity.badInput` (S19's lessons).
- **Seed:** unchanged (it has no episodes).

## Differences from the plan or brief

- **Undo** (answer 2) replaced the brief's recommended super-clinician step back, and has its own route.
- The brief's manual test focus 6 ("works only for a super clinician") is superseded by answer 2: any clinician.

## Tests (TDD)

- **Unit (Go):** `internal/episode/episode_test.go`: `TestCheckEveryPair` checks every pair of statuses (plus unknowns) × every move × every role and the patient against the table written out again by hand (5,760 combinations): allowed at the least role and above, `ErrRole` below, `ErrIllegal` otherwise, with the reason and date flags; `TestOptions`, `TestUndoTarget`, `TestCancelReasons`, `TestEveryStatusHasALabel`. `TestEpisodeChangeValidate` (no revision, a status in the details PATCH).
- **Integration (`statuses_integration_test.go`):**
  - `TestChangeStatus`: five cases; invalid input (no or negative revision, no target, a long reason, Admitted without a date, bad dates); a pick with and without a reason (history and note); Admitted and Discharged with their dates (discharge before admission, a future date refused); stale revision → 409 `code: "stale"`.
  - `TestChangeStatusEveryPair`: every pair through `PATCH …/status`: the table's picks, Admit and Discharge → 204 and the status moves; everything else → 409 and nothing changes.
  - `TestCancelEpisode`: five cases; every reason from every status before discharged; Other needs a sentence; final (no move, re-open, undo, PATCH or Resend after; notes still allowed); stale → 409.
  - `TestCancelEndsThePatientsSide`: the link stops (404), the session opened from it ends (401), the jobs finish, the home says `cancelled` while the same patient's other episode is untouched, and the form isn't served, saved or sent (403 `cancelled`), for the patient and for staff.
  - `TestReopenReview`: five cases (a clinician → 403); only from the five reviewed statuses; a reason required; happy path by a one-hospital super clinician: Ready for review, the stamp cleared, the clinician's answers and validation kept and saveable, the patient's form still sent; an admin may; stale → 409.
  - `TestUndoStatus`: five cases; a reason required; stale → 409; discharged → admitted → ready for admission → on hold → ready for POA, the dates cleared on the way, then 409 at the completed review; a new move after an undo is the next one undone; nothing to undo on a new or cancelled episode.
  - `TestEpisodeStatusView`: `moves`, `undo` and `statusHistory` in `GET …/episodes/{eid}` for a clinician and a super clinician, and the create, submit and complete-review rows.
  - `TestTwoTabsDontOverwriteEachOther`: Nurse ASA then Anaesthetist ASA from the same revision: the second is 409, the first kept; after a reload it goes in.
  - Updated: `TestUpdateEpisode` (revision, no status), `TestUpdateEpisodeKeepsAStatusThatJustChanged` (now the revision: the submit or the review in between → 409; two PATCHes at once → the second refused), `TestAuditRowsWritten` (the four new actions), `TestFollowupStops` (cancelling stops reminders and not responding; a job already running when it's cancelled sends nothing).
  - **Migration:** `TestEpisodeStatusesMigration` (`internal/apitest`): existing episodes keep their status at revision 0 with no history, the new statuses and the cancel reason CHECK, and Down.
- **Web (Vitest):** `status.test.ts`, `home.test.ts` (cancelled).
- **End-to-end:**
  - **J9b** (`journeys/test_j9b_cancel.py`): the patient opens their invite on a phone; Cara cancels with Other (refused without a sentence); the alert, the disabled control and the history; the patient's session has ended, the link doesn't work, and signing in by email their home shows "Cancelled by the hospital" with no button, at phone size.
  - `journeys/test_statuses.py`: walk to Discharged with both dates and undo the discharge; Sue Super re-opens (Cara has no Re-open); two tabs (ASA kept with Reload; a stale status move refused).
  - J5 confirms its status change in the new short form (`support.change_status`); `test_procedures` sends the revision with its direct PATCH.

### Results (21 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 189 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against my own API on :8091 with its own database `sj_s20`, and Vite on :3101, `E2E_BASE_URL=http://localhost:3101`): **51 passed, 5 skipped, 1 failed** (`test_procedures`: its direct PATCH had no revision); fixed, then `test_procedures.py` 4 passed. J9b, `test_statuses.py` and J5 passed.
- **Local database:** `sj` migrated to 00021 (`make migrate`); no re-seed needed.

## Test report 1 (21 Oct): 20 of 21 passed; F-1 fixed

- **F-1 (T-03 step 4): Enter in the reason box added a new line instead of sending the form.** **Fixed by making every reason box one line** (a `TextInput`): the status form's reason, Undo's and Re-open's, and Cancel's "Details" / "Say why". Enter now sends each form, as it does every other form in the app (S19's procedure and episode forms), and the plan's expectation stands. **Why not keep the textarea with Ctrl/Cmd+Enter:** a reason is a sentence, the API keeps it on one line in the history and General notes, and a second key rule would be the only one in the app. **Tests first:** `test_statuses.py` presses Enter in the reason (pick and Undo) and J9b in "Say why"; both failed before the fix.
- **Found while exploring, fixed:**
  - **A stale tab's form for a move no longer offered stayed open after Reload** (the server refused it safely). Now, when the reloaded episode no longer offers the move (or the undo, re-open or cancel), the form closes and says "The episode changed, so that change is no longer offered." Test first: `test_after_reload_a_move_no_longer_offered_closes_its_form`.
  - **The reason boxes have a 1,000-character limit**, as the API (`maxLength`); `test_statuses.py` checks a 1,001-character paste keeps 1,000.
  - **An unknown status sent to `PATCH …/status` is 400 "Choose a status."**, not 409 (it's bad input, not a conflict). Test first: `TestChangeStatus` and `TestChangeStatusEveryPair`.
  - **Sign out on the patient's refused page** (a cancelled or unknown episode): the header now has Sign out once the refusal shows. J9b signs out there.
- **Logged, not changed:** on staff's on-behalf form (`…/patient-form`) for a cancelled episode, the message is the patient's ("Please contact the hospital if you have any questions"). Cosmetic; in pending.md §3.
- **Results:** `make lint`, `make test` (Vitest 189 + 116, CDK 124), `make test-integration` all pass; `make e2e` on my own ports and database: **53 passed, 5 skipped**.

## Test report 2 (21 Oct): all 22 passed

- Every retested case and regression passed, including F-1 and the three exploration fixes.
- **Logged in pending.md §3, no code change** (as the conductor asked): a stale Undo form retargets after Reload (safe: its title names the new target, nothing is sent without a click); the ASA fields' spacing at 500 px on the POA Summary (older than S20).

## Hand checks for Rahul

- None yet.

## Suggested commits

1. **S20: the transition table, status history, revision, cancel, re-open and undo**: everything above (the Go table and its unit tests, migration 00021 and its test, the routes and their integration tests, the patient's side, the web status control and history, J9b and `test_statuses.py`, the J5 and `test_procedures` updates).
2. **docs: S20 build notes, test plan, answers, decisions, plan and session status**.
