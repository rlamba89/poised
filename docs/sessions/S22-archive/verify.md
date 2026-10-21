# S22 verification

## Verification 1 (21 Oct 2026, automatic run)

**Verdict: Verified.** Nothing blocks. The findings below are small or need two things to happen in the same instant; the ones worth doing later are in pending.md §3 ("From S22's verification"). S22 is C3's last session, so **plan C3 is Done**.

Reviewed: the whole working tree against `9d37cf7` (49 changed files, 14 new), the brief, answers.md, the build notes, the test plan and test reports 1–3 (report 3: 17/17 passed). `/code-review high` was run and its findings weighed below.

### Findings (none blocking; worst first)

1. **Closing an account can deadlock with an archive or cancel of one of its episodes at the same instant.** `closeAccount` (`apps/api/internal/httpapi/accounts.go:681`) locks the records' `jobs` rows (`FinishRecordsFollowups`, line 709) and only then the episodes (the "closed their account" note's insert, line 715, whose FK and `touch_episode` trigger lock the episode row). `applyStatus` for Archive or Cancel, and the auto-archive (`jobs.Runner.archive`), lock the episode first (`LockEpisodeStatus` / `AutoArchiveEpisode`) and then its `jobs` rows (`FinishEpisodeJobs`, `statuses.go:346`). If both run in the same few milliseconds, Postgres aborts one: a clean rollback and a 500, so the person tries again and nothing is half-done. Fix later: in `closeAccount`, lock the records' episodes (`SELECT … FOR UPDATE` in id order) before touching `jobs`/`login_links`.
2. **A patient's sent questionnaire loses its View once archived** (`accounts.go:406`: archived → `closed`; `patient.go:330`: the form answers 403 `archived`). Archived normally happens 30 days after discharge, so from then on the patient can't read the answers they sent, which they could before. This follows the builder's documented reading of answers 2 ("read-only"), but read-only needn't hide reading. **For Rahul:** keep it (recommended only if the patient's copy is meant to end with the episode), or show archived-after-sent as "Sent" with View (read-only), which I'd recommend.
3. **A patient's autosave racing an archive can still land.** `loadForm` checks the status without a lock (`patient.go:330`), and `SavePatientAnswers` (`apps/api/db/queries/episodes.sql:222`) checks only that the form is open, which archiving leaves open. An autosave that passed the check just before an archive committed is written to the now-archived episode. Harmless (un-archive restores exactly, and staff can't change it), but the SQL could also require the episode not to be archived.
4. **`RunDue` stops all follow-ups if adding a hospital's auto-archive job fails** (`apps/api/internal/jobs/jobs.go:64`). The likely cause, a bad zone, can't happen (hospital zones are validated in `admin.go` and tzdata is built in), and a database error would fail the follow-ups anyway, so this is robustness only: logging and carrying on would keep reminders going.
5. **Smaller points:**
   - An episode archived from Cancelled: the patient's home says "Cancelled by the hospital" (`accounts.go:402`) but an old link's form says "This questionnaire is closed" (`patient.go:330` checks only `archived`). Both are true; the wording differs.
   - Closing an account writes its General note on archived episodes too, and it counts as activity (restarting the 6 months on pre-op or cancelled episodes). Both read as intended: staff should see the note, and it is a change to the episode.
   - The /qa/jobs "as of" run pushes every hospital's real 02:00 auto-archive to after the "as of" time (`RescheduleJob` with `NextArchiveRun(asOf)`), so on QA the nightly run stops until then. Capture mode only; already in the spirit of pending.md's F-2 notes.
   - `ListArchiveCandidates` reads every pre-op, discharged and cancelled episode of the hospital each night and Go decides; `ClaimDueJob`'s new `ORDER BY (kind <> 'auto_archive'), run_at` sorts the due jobs on each claim. Fine at today's sizes.
   - The patient cookie is cleared by a copied `SetCookie` line in `closeAccount`, and the archived read-only check is hand-written in four places (`addEpisodeNote`, `resendInvite`, `loadForm`, `notInReview`) beside the new `readOnly()`. `TestArchivedEpisodeIsReadOnly` covers every write route, so a missed one would fail it.

### Checks

- **Migration 00023 (edited in place during the build):** on a fresh database (`sj_v22`): up to 23, seed, down one, up again, down to 0, up again: all clean. `TestArchiveMigration` covers the carry-over of `last_activity_at`, each trigger, the CHECKs, `in_worklist`, and Down putting archived episodes back. Down drops the archive/un-archive history rows and tolerates a database migrated before the edit (`IF EXISTS` on `touch_episode_now` and `seq`).
- **`last_activity_at` triggers:** five `AFTER` triggers (`episode_events`, `episode_status_events`, `episode_messages` with the row's own `created_at`, never backwards; `tasks` and `episode_answers` with `now()`). `TestLastActivity` checks each write moves it and another episode's doesn't. Triggers rather than handler code is the right call here.
- **Job runs dated with their own time (F-2):** `AddAutomaticEpisodeEvent` and `AddStatusEvent` take the run's `at`; the reminder, not-responding, auto-archive and undelivered-report notes all pass it; auto-archives are claimed before follow-ups. `TestJobRunsUseTheirOwnTime` covers it. On a real clock `at` is now.
- **Auto-archive schedule and A-20:** one `auto_archive` job per hospital (unique partial index), added by `RunDue` for any hospital without one, at `NextArchiveRun` = 02:00 in the hospital's zone, else `cfg.Country.TimeZone` (the runner's `DefaultZone`). No zone literal outside tests; `TestAutoArchiveSchedule` uses a UTC+14 hospital. `ArchiveDue` is pure and unit-tested (+29/+30/+31 days, the hospital's 7 days, the hospital's date not UTC's, un-archived since, 6 months less a minute, every pre-op status, cancelled, admitted never).
- **Scope:** everything in the brief is done: archive and un-archive with `archived_from`, read-only (409 on every other write), worklists and lists, Show archived, the daily auto-archive, closing an account, J9 and the close-account journey. Extra: `?archived=true` and `unarchive` in the episode JSON, both needed and recorded in the plan's differences.
- **Tests first:** both new staff routes have the five cases plus stale, every-status and exact-restore cases (`archive_integration_test.go`); the patient route has no-sign-in, a clinician's cookie, an ended session, the happy path with rows, another patient untouched, and the shared-contact case (`close_account_integration_test.go`).
- **Tenant isolation:** archive/un-archive go through `requireHospital` + `loadEpisode`; `ListArchiveCandidates` and `AutoArchiveEpisode` are per hospital/episode; `closeAccount` touches only the signed-in account's records. Checked live (below).
- **Audit:** `episode.archive`, `episode.unarchive` and `account.close` are wrapped in `audited`; close writes one row per trust.
- **A-20 grep:** only the existing comments in `format.ts` and `patient.go`.
- **Security:** no secrets; logs carry IDs only; the close-account note has a date only; the patient router adds only `/api/p/account/close`; `chapter.ForPatient` is untouched.
- **Docs:** plan C3 Status and differences, plans/README, sessions/README, decisions.md and pending.md updated; the Lifebox organisation grep prints nothing.

### Test results (21 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 204 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok.
- `make e2e` against my own instance (API :8093, Vite :3103, database `sj_v22`): **57 passed, 5 skipped** (the QA-only Cognito journeys). My first run failed every journey because my database had no users (my own down-to-0 check had emptied it); after re-seeding it, all passed.

### Spot-checks (Chrome, "trk" profile, against `make dev` on :3000)

- **Permissions:** as Cara (clinician), un-archive → 403 "Only a super clinician or an admin can un-archive an episode."; a note and archiving again on an archived episode → 409 with the archived message.
- **Another trust:** as Bea (Trust B), un-archive of a Trust A episode → 404 by Trust A's path ("Hospital not found.") and by Trust B's path ("Episode not found.").
- **Invalid and stale:** as Sue, a blank reason → 400 "Say why the episode is being un-archived."; an old revision → 409 `stale`.
- **Exact restore, worklists:** as Sue in the UI, un-archived `Sam S22 Worklist` with Enter: back to **On hold** (status history "unarchive", the note), and back in On hold, Anaesthetic review, Procedure within 14 days and Open tasks (each count 1). Archived again with the UI form, no reason: every worklist count 0, `archivedFrom` on_hold, `unarchive` offered, and it's in `?archived=true`.
- Not repeated: the patient pages at 390×844 (the testers did them at phone width in a frame; it stays a hand check below).

### Hand checks for Rahul

- **Phone size in a real window:** T-09 and T-13 (the closed questionnaire; Close your account) in a real 390×844 window. The testers used a 390px frame inside the page.

### Suggested commits

1. **S22: archive, auto-archive and closing a patient account**: migration 00023 and its test, the queries and sqlc output, `internal/episode` (table, `ArchiveDue`, `NextArchiveRun`) and their unit tests, the routes and handlers, the jobs runner's auto-archive, the close-account route, their integration tests, the web changes, the Settings number box fix, J9, the close-account journey and the journeys adjusted.
2. **docs: S22 verified**: brief answers, build notes, test plan and reports, verification, decisions, pending, plan C3 Done, session status.
