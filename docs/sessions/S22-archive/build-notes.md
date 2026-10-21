# S22 build notes: archive, auto-archive and closing a patient account

Builder, automatic run (`/run-sessions`), 21 Oct 2026.

## Answers (from [answers.md](answers.md), asked by the conductor on 20 Oct; also decisions.md §2)

1. **Who archives and un-archives by hand:** **a super clinician for both** (not the brief's recommendation of any clinician archiving); un-archive needs a reason.
2. **Read-only:** every write to an archived episode gets 409 except un-archive. Archived episodes leave every worklist, and stay in search, on the patient page and in the All list behind "Show archived".
3. **"No activity":** `episodes.last_activity_at`, moved forward by every write (answers, status, notes, tasks, messages sent).
4. **The daily job:** kind `auto_archive`, one row per hospital, at 02:00 in the hospital's time zone, scheduling its own next run. "After the last PROMs form closes" waits for C9.
5. **Closing a patient account (EPI-09):** as in the brief.

## What was built

### Data (migration `00023_archive.sql`)

- **`episodes.status` may be `archived`**, and **`archived_from`** holds the status it came from, set exactly while archived (a CHECK).
- **`episode_status_events.move`** may be `archive` or `unarchive`; an automatic archive names its rule in `rule` (the column S20 added) with no user.
- **`episodes.last_activity_at`** (NOT NULL), carried over from each episode's newest row (notes, status history, tasks, messages, answers, or its creation). **Five triggers keep it current** (`touch_episode`): a row added to `episode_events` (every status change, note, task change, invite, reminder and message writes one), `episode_status_events`, `tasks` (insert or update), `episode_messages`, and `episode_answers` (insert or update: the patient's autosaves and the clinician's). Triggers rather than code in each handler, so a write path added later can't forget it.
- **`jobs`**: kind `auto_archive`, with a new `hospital_id`; one open auto-archive per hospital (a unique index).
- **`in_worklist`** (S21) leaves out archived episodes, as it did cancelled ones.
- Down puts archived episodes back where they were archived from.

### Archive and un-archive (EPI-04; `internal/httpapi/statuses.go`)

- **The transition table** (`internal/episode`) has `archive` from every status to Archived and `unarchive` from Archived to any status, both **super clinician** (the A-5 ladder, so admins too); un-archive needs a reason. `Options` lists the archive (the page shows it as a button) but not un-archive, whose one target is `archived_from`: `GET …/episodes/{eid}` adds **`unarchive: {to, label}`** for someone who may.
- **`POST …/archive` {reason?, revision}** and **`POST …/unarchive` {reason, revision}**, audited (`episode.archive`, `episode.unarchive`), through S20's `applyStatus`: the revision (stale → 409 `stale`), then the table (a clinician → 403 "Only a super clinician or an admin can archive/un-archive an episode."), the status, `archived_from`, a status event, a General notes line ("Episode archived: <reason>", "Episode un-archived, back to <status>: <reason>"). A cancelled episode keeps its cancel reason through both.
- **Archive stops the follow-up** (reminders, "not responding"), as cancel does; the jobs also skip archived episodes. **It keeps the patient's invite link**, which opens nothing while archived (the form answers 403 `code: "archived"`, "This questionnaire is closed…"), so un-archive restores everything as it was. **Un-archive doesn't restart reminders** (staff can resend the invite).
- **Read-only (answers 2):** every other write is 409 "This episode is archived, so it can't be changed. A super clinician can un-archive it.": the details PATCH, notes (a cancelled episode still takes notes; an archived one doesn't), status, cancel, re-open, undo, archive again, tasks, the clinician's answers, Complete HQ review, Resend invite, and the on-behalf form (save and submit; staff get 409, the patient 403 `archived`).
- **Lists:** `GET …/episodes` (and the trust's) leave archived episodes out unless **`?archived=true`** or `?status=archived`. Search and the patient's page are unchanged (they find them).
- **The patient's home:** an archived questionnaire is **`closed`** ("Closed by the hospital", no button); one archived after being cancelled stays `cancelled`.

### The daily auto-archive (EPI-05; `internal/episode/archive.go`, `internal/jobs`)

- **`episode.ArchiveDue(facts, policy, now) → (rule, due)`**, pure and written test-first:
  - **Discharged:** due on the hospital's date **discharge date + N** (N = `autoArchive.afterDischargeDays`, default 30): so +30 and +31 days are due, +29 isn't. **An episode un-archived since counts its N days from the un-archive**, so it isn't archived again the next night.
  - **Before admission (hq_not_complete … ready_for_admission) or cancelled:** due once **M months** (`autoArchive.inactiveMonths`, default 6) have passed since `last_activity_at`, in the hospital's zone.
  - **Admitted**, archived or anything else: never.
  - The rule as recorded: "Auto-archive: 30 days after discharge", "Auto-archive: 6 months without activity" ("1 day", "1 month" singular).
- **`episode.NextArchiveRun(now, zone)`**: 02:00 at the hospital, strictly after now.
- **The job** (`jobs.Runner.autoArchive`): reads the hospital's zone and effective settings (its own, else the trust's, else the platform's: the S10 settings already had both keys), lists its discharged, pre-op and cancelled episodes, asks `ArchiveDue`, and archives each due one in its own transaction **only if nothing changed it since it was read** (same status and `last_activity_at`): status, `archived_from`, revision + 1, a status event with the rule and no user, an **Automatic** General notes line "Episode archived (<rule>)", and its follow-up finished. Then it reschedules itself to the next 02:00. Nothing is archived while the trust or hospital is suspended.
- **Each `RunDue` first gives every hospital without one its auto-archive job**, due at its next 02:00. So new hospitals, the seed and a fresh database need nothing else; the local timer and the jobs Lambda (S17) already call `RunDue`, so nothing new is scheduled.
- **The testers' "run the jobs as of" (`POST /api/qa/jobs/run`, capture mode only) also runs every hospital's auto-archive as of that time**, whatever its next 02:00 (`RunAllAsOf`), so a tester's cases don't depend on the order they run them in. The summary adds `autoArchives` (jobs run) and `archived` (episodes).

### Closing a patient account (EPI-09; `internal/httpapi/accounts.go`)

- **`POST /api/p/account/close`** on the patient's session, audited (`account.close`: a row in each trust whose records it unlinked). In one transaction:
  - the trust's records leave the account (`patients.account_id = NULL`); records and episodes stay;
  - their **invite links are revoked** (an unsent HQ won't be sent) and their **reminders finished** (each would send a new link);
  - **unused sign-in links** to the account's mobile or email are revoked, **unless another account uses that contact** (twins on one phone, S15): theirs carry on;
  - the account's **sessions** and its **links to confirm a new contact** are deleted, then **the account**;
  - every episode of those records gets a General notes line, **"The patient closed their account on YYYY-MM-DD."** (the date at its hospital).
- The cookie is cleared. A later invite (staff's Send/Resend) makes a new account as usual (S15).

### Web

- **Status control** (`StatusControl.tsx`): **Archive episode** (when `moves` has it) and **Un-archive: back to <status>** (when `unarchive` is set) buttons, each with an inline form (no modal), `noValidate`, Enter sends, one-line reason (1,000 characters): optional for archive, required for un-archive ("Say why the episode is being un-archived.").
- **Episode page:** a grey "This episode is archived" box (from what, when, by whom or which rule, the reason); every control off: no Edit procedure, no invite card, no Add comment, Tasks read-only, the status dropdown disabled. **POA Summary:** no Add comment, the ASA grades disabled (also for a cancelled episode, whose grades the API already refused). **Validate page:** "This episode is archived, so these answers can't be changed."
- **Episode list:** **Show archived** beside the status filter on All, kept in the URL as `?archived=1`; the status filter has **Archived**.
- **Patient home:** the `closed` state; **Close your account** card at the bottom with an inline confirmation, warning about any questionnaire not yet sent, then "Your account is closed".
- **/qa/jobs:** says what the auto-archive does, the result line counts episodes archived, and presets **Now + 29 days** and **Now + 31 days**.

### TypeScript Rahul should know about

- `features/episodes/status.ts`: the `archive` / `unarchive` moves and forms; `statusRequest` turns them into `POST /archive` and `/unarchive` (unit-tested in `status.test.ts`). The history words them "Archived: … → Archived", "Un-archived: …", and an automatic change's "By" is "Automatic (<rule>)".
- `features/patient/home.ts`: the `closed` state and `unsentWarning` (the confirmation's warning, naming each questionnaire not yet sent; unit-tested in `home.test.ts`).
- `features/patient/PatientHome.tsx`: `CloseAccount` (the card and its confirmation) and `AccountClosed` (the page after).
- `features/episodes/EpisodeList.tsx`: the Show archived box (`?archived=1` → the API's `archived=true`). Which episodes are left out is Go's.

## Decided by the builder (automatic mode)

- **Archiving keeps the invite link and un-archiving doesn't restart the reminders.** Archive must be exactly undoable, so the link isn't revoked (the patient's form is closed while archived instead); reminders after un-archive would arrive at once for days long past, so staff resend the invite if needed.
- **Cancelled episodes are archived by the "no activity" rule** (M months), like pre-op ones: the lifecycle (§5.1) has "cancelled → archived", and nothing else would ever archive them. **Admitted episodes never are** (neither rule fits).
- **Un-archiving restarts the clock:** a discharged episode's N days count from the later of discharge and its last un-archive; the "no activity" months count from the un-archive anyway (it's a write). Otherwise the next night would archive it again.
- **`last_activity_at` is kept by triggers**, not by each handler, so it can't be missed by a later write path (healthcare: nothing breaks silently). The jobs' own notes ("not responding", reminders) count as activity, as messages sent do (answers 3).
- **Notes on an archived episode are refused too** (answers 2 lists notes), unlike a cancelled episode's.
- **Archive and un-archive are stops for Undo** (S20): neither can be undone by Undo; they're each other's way back.
- **A job run's writes are dated with the run's time, and a run takes the auto-archives before the follow-ups** (test report 1, F-2; above).
- **The QA "run jobs as of" runs every hospital's auto-archive** as well as what's due, so testers needn't run the cases in date order. The real schedule (02:00 at each hospital, its own zone) is covered by `TestAutoArchiveSchedule`.
- **Nothing is archived while a trust or hospital is suspended** (as S16's follow-ups do nothing then).
- **Closing an account revokes the records' invite links and finishes their reminders**, so "won't be sent" holds; staff can still send a new invite, which makes a new account. **A sign-in link to a contact another account shares is kept.**
- **The patient's home shows an archived questionnaire as "Closed by the hospital"** with no button (the form is closed), and one archived after cancelling as cancelled.
- **The closing note's date** is ISO (YYYY-MM-DD), as the General notes' other dates (S19).
- **Wording** (made up, plain): "This episode is archived, so it can't be changed. A super clinician can un-archive it."; the patient's "This questionnaire is closed, so there's nothing more to fill in. Please contact the hospital if you have any questions."

## Differences from the plan or brief

- **The brief's `due(episode, settings, now)`** is `episode.ArchiveDue(ArchiveFacts, ArchivePolicy, now)`, beside `episode.NextArchiveRun`.
- **One query parameter the brief didn't name:** `GET …/episodes?archived=true` for "Show archived"; and `unarchive` in the episode's JSON.
- **The QA run also runs the auto-archive** (above), and its summary has two more counts.
- **ASA grades on the POA Summary are disabled for a cancelled episode too** (the API already refused them since S20; the page now shows it).

## Tests (TDD)

- **Unit (Go):**
  - `internal/episode/episode_test.go`: the transition table, every pair × every move × every role, now with archive and un-archive; `Options` (archive for a super clinician and admin only, never un-archive); `UndoTarget` (archive and un-archive are stops).
  - `internal/episode/archive_test.go`: **`TestArchiveDue`** (30/31 days due, 29 not; the hospital's 7 days, 8 due and 6 not; the hospital's date, not UTC's; un-archived since; 6 months due, less a minute not, a note last week not; the hospital's 2 months; every pre-op status; cancelled; admitted never; already archived; no settings) and **`TestNextArchiveRun`** (UTC, London in summer and winter, strictly after).
- **Integration:**
  - `apitest` **`TestArchiveMigration`**: `last_activity_at` carried over from the newest row; each trigger moves it; the CHECKs (archived_from, the auto-archive job's hospital, one job per hospital, the moves); `in_worklist` leaves archived out; Down puts archived episodes back.
  - `archive_integration_test.go`: **`TestArchive`** and **`TestUnarchive`** (the five cases each: no sign-in 401, a clinician 403 and the other hospital 404, another trust 404, invalid input 400; stale 409; the happy paths with the rows, the history, the notes, the follow-up stopped, the audit rows; from every status; **un-archive restores the exact previous status, dates and cancel reason, from every status**); **`TestArchivedEpisodeIsReadOnly`** (**every write → 409**, for a clinician and a super clinician, nothing changed; the patient's form 403 `archived`; still read, on the patient's page and in the search; un-archived, writable again); **`TestArchivedLeavesWorklists`** (every worklist, its count, All, `?archived=true`, `?status=archived`, All hospitals; back after un-archive); **`TestPatientHomeShowsArchived`**.
  - `auto_archive_integration_test.go`: **`TestAutoArchive`** (29 days: only the 7-day hospital's; 31 days: archived with the rule, an Automatic note, no user, shown on the page; un-archived isn't re-archived the next night but is 31 days later; 6 months untouched archived, a note added through the API a week before isn't, admitted isn't, cancelled is; A2's 2 months; suspended: nothing; a stale page refused after); **`TestAutoArchiveSchedule`** (one job per hospital at its next 02:00 in its zone; a new hospital gets one; no duplicates); **`TestLastActivity`** (the patient's answers, a note, a task, a message sent, the clinician's answers, a status change each move it; another episode's write doesn't).
  - `close_account_integration_test.go`: **`TestCloseAccount`** (no sign-in, a clinician, an ended session: 401; the account, its sessions and verify links go; the records and episodes stay, unlinked, each noted; reminders stop; an audit row per trust; the old session, invite link and sign-in link fail, and no new sign-in link is sent; staff still read them; another patient untouched; a new invite makes a new account) and **`TestCloseAccountKeepsASharedContactsLinks`**.
  - Existing tests adjusted: `setStatus` sets `archived_from` for Archived; the jobs tests' `runJobs` counts the follow-ups only (`Jobs - AutoArchives`), since each run now also runs the auto-archives.
- **Web (Vitest):** `status.test.ts` (archive and un-archive requests, the history's words, an automatic rule), `home.test.ts` (`closed`, `unsentWarning`).
- **End-to-end:** **J9** `journeys/test_j9_lifecycle.py` (the full lifecycle: created with a procedure → patient submits → review → Ready for admission → Admitted → Discharged → Sue archives (Cara can't) → read-only, out of the list until Show archived → un-archive → the auto-archive 31 days on, naming its rule) and `journeys/test_close_account.py` (the patient closes their account at iPhone 13 size, with the warning; signed out; old invite link fails; staff see the note; a new invite starts a new account).

### Results (21 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 204 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against my own API on :8092 with its own database `sj_s22`, and Vite on :3102, `E2E_BASE_URL=http://localhost:3102`): **57 passed, 5 skipped** (the QA-only Cognito journeys). In the first full run S20's `test_a_stale_tab_is_told_and_keeps_its_input` failed once at a Mantine Select (the Save stayed disabled); it passed alone and in a second full run: a flake, not this session's.
- Checked headless (Playwright screenshots, not the testers' Chrome): the archived episode page, the archive form, the list with Show archived, the patient's closed questionnaire and the Close your account confirmation at iPhone 13 size.
- **Local database:** `sj` migrated to 00023 (`make migrate`); no re-seed needed. My `sj_s22` database is dropped and my servers stopped.

## Test report 1 (21 Oct): 21 of 23 passed, 2 failed; both fixed test-first

- **F-1 (T-12): Settings changed 0 to 1 and 366 to 365 and saved them.** The shared number box (`SettingsPage.tsx`, Mantine `NumberInput` with min/max) clamped on blur, so the API never saw the bad value. Now `clampBehavior="none"`: the value stays as typed and Save shows the API's refusal. Every number setting uses that one box; the other `NumberInput`s (Preview's age, the editor) aren't settings. *Test first:* the S10 settings journey (`test_j6_onboarding.py`) now types 0 and 366 for a hospital's archive days and expects the API's message, the box unchanged, and nothing saved; it failed (the box showed 1) before the fix.
- **F-2 (T-18): an idle emailed episode wasn't archived by a run "as of" 6 months on.** In that one run the overdue reminder and "not responding" went first, written with the real clock; the 00023 trigger counted them as activity after the cutoff. Two fixes, as the conductor asked (clock consistency; answer 3 unchanged: messages still count as activity):
  - **What a job run writes is dated with the run's time**: its General notes (`AddAutomaticEpisodeEvent` takes `at`), the auto-archive's status event (`AddStatusEvent` takes an optional `at`), and an undelivered report's note (its own time). On a real clock that's now.
  - **`last_activity_at` follows the row's own time** where it has one (notes, status history, messages: `touch_episode` uses `NEW.created_at`, never moving it back); tasks and answers use now() (`touch_episode_now`, `touch_episode_of_form`).
  - **A run takes the hospitals' auto-archives before the follow-ups** (`ClaimDueJob` orders `kind <> 'auto_archive'` first), so the archive decides on the activity there has been, and an episode it archives isn't reminded in the same run (the follow-up skips archived episodes).
  - **`episode_events.seq`** (an identity column, in 00023) orders notes written at the same time (a reminder, then "not responding"), since they now share the run's time.
  - *Test first:* `TestJobRunsUseTheirOwnTime` (an invited idle episode is archived as of 6 months and a minute on, with no reminder sent; a day-2 run's reminder note and the activity carry the run's time; the archive's history row too). It failed before the fix.
  - 00023 was edited in place (not yet committed or deployed); the local `sj` database was migrated down and up again, and its Down tolerates a database migrated before the edit.
- **Found while exploring:**
  - Fixed: after a patient closes their account, the invite card said "No invite has been sent". It now says "The patient's link no longer works: the General notes say why (for example, they closed their account). Send a new invite if they still need it." (when an invite reached the patient before; the close-account journey checks it).
  - Logged in pending.md §3 (no change): archived rows still show "1 open task".
- **Results after the fixes:** `make lint` passed; `make test` (Go, Vitest 204 + 116, CDK 124) passed; `make test-integration` every package ok; `make e2e` against my own instance (`sj_s22`, :8092/:3102): **57 passed, 5 skipped**. In one full run S11's `test_a_list_row_names_ten_patients_and_counts_them_all` failed once; it passed alone and in the next full run (a flake, not this session's).

## Test report 2 (21 Oct): 14 passed, 1 failed (the plan, not the code)

- **F-3 (T-18 step 5): `S22 Busy` wasn't archived at 6 months and a day.** As designed: the step-3 run sent it a reminder and marked it "Not responding", dated that run's time (F-2), and messages sent count as activity (answers 3). The plan was wrong, so **step 5 now runs as of 12 months and a day**; T-10's box now shows the run's date. No code change.
- **Found while exploring, fixed test-first:** typing `7.5` in a Settings number box showed `75` (Mantine's whole-number box drops the point). The box now takes decimals and the API refuses anything but a whole number with its message ("…: enter a whole number from 1 to 365."). *Tests:* `settings_test.go` refuses `7.5`; the S10 settings journey types `7.5` and expects the box to keep it and Save to be refused (it failed, showing 7, before the fix).
- **Logged in pending.md §3:** /qa/messages shows a reminder's real send time while its note has the run's time (testers' tool only).
- **Results after the fix:** `make lint` passed; `make test` (Go, Vitest 204 + 116, CDK 124) passed; `make test-integration` every package ok; `make e2e` on my own instance: **57 passed, 5 skipped**.

## Test report 3 (21 Oct): all 17 passed

- Retest of T-18 (F-3), T-10 and T-12 as changed, R-1–R-5 and the main happy paths: all passed.
- Logged in pending.md §3 (no code change, as the conductor asked): General notes look out of order after a future-dated /qa/jobs run and then a real-time change (testers' tool only); an archived episode still shows the NOT RESPONDING badge.

## Hand checks for Rahul

- **Phone size in a real window:** the testers used a 390px frame inside the page (their window wouldn't resize). Check T-09 and T-13 (the closed questionnaire, Close your account) in a real 390×844 window.

## Suggested commits

1. **S22: archive, auto-archive and closing a patient account**: migration 00023 and its test, the queries, `internal/episode` (table, `ArchiveDue`, `NextArchiveRun`) and their unit tests, the routes and handlers, the jobs runner's auto-archive, the close-account route, their integration tests, the web changes, J9 and the close-account journey.
2. **docs: S22 build notes, test plan, decisions, plan and session status**.
