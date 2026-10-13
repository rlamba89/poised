# S16 build notes: scheduled jobs, reminders and "not responding" (local)

Builder, automatic run (`/run-sessions`), 13 Oct 2026. Rahul's answers are in [answers.md](answers.md):
1. **Reminders go at any time** a reminder is due: no 09:00–19:00 window (the recommendation had been the window).
2. **Starting stops reminders:** once the patient opens the HQ and answers anything (the recommendation had been "only submitting").

Because of answer 1, the brief's manual test focus 4 ("a run at 22:00 hospital time sends nothing") doesn't apply: the test plan checks the opposite, that a reminder due at night goes.

## What was built

### Data (migration `00016_jobs.sql`; up, down, up checked on the local database)

- **`jobs`**: `kind` (only `followup` so far), `episode_id`, `payload`, `run_at`, `done_at`, `attempts`, `last_error`. One open follow-up per episode (a unique partial index). Indexed on `run_at` where not done.
- **`episodes`**: `invited_at` (the **first** invite that reached the patient: the reminder days count from it), `reminders_sent` (how many reminder days have been dealt with), `not_responding_at`.
- **`episode_events.automatic`**: a General notes entry written by the jobs. The page shows **"Automatic"** as its author (a NULL user used to mean "Patient").
- **`qa_jobs_clock`** (one row): the testers' offset for the local timer's clock, and the timer's last run. **`make seed` sets the offset back to 0.**

### Go

- **`internal/jobs/due.go`: `Due(followup, schedule, now) → actions`**, the pure decision, written test-first (`due_test.go`, 21 cases):
  - day *n* is the invite's local time *n* calendar days later **in the hospital's time zone** (S08), so a reminder keeps its time of day across a clock change (tested for the March change, and for another zone);
  - a reminder on each of `reminders.days` (ORG-04; 2 and 5 by default) **while the patient hasn't started**; if runs were missed and several days have passed, **only the latest goes, once**;
  - **"not responding" once, on `reminders.notRespondingDay`** (7), if the HQ hasn't been sent, **whether or not the patient started** (EPI-02 says "no submit");
  - nothing once the HQ is sent, or if no invite ever reached the patient; while the trust or hospital is suspended, nothing, and it looks again in an hour;
  - it always says when to look again (always after now), or that the follow-up is finished.
- **`internal/jobs/jobs.go`: `Runner.RunDue(ctx, now)`**, one function that keeps nothing in memory and touches no disk (Lambda-ready, S17):
  - **claims one due job at a time with `FOR UPDATE SKIP LOCKED`**, moving its `run_at` on by a 10-minute lease, so no two runners run the same job and nothing stays locked while it runs (an earlier version held the row lock for the whole job and deadlocked when four runners shared a 4-connection pool: found by the concurrency test);
  - reads the episode and its hospital's settings and zone, asks `Due`, then does it. **A reminder is counted on the episode (`CountReminder`, a compare-and-set) before it's sent**, so it's sent at most once even if two runners decide together; a send that then fails is written to the General notes and not retried (as for an invite: staff can resend);
  - then reschedules the job at `Due`'s next time, finishes it, or (on an error) fails it with a backoff (5 minutes doubling to 6 hours), logged;
  - at most 500 jobs per call (the rest wait for the next run).
- **`Runner.Tick(ctx, real, qaClock)`** is one scheduled run: `RunDue` at the real time, plus the testers' offset when `qaClock` (MESSAGING=capture only), noting the run. **`Runner.Every`** is the local timer.
- **The local timer:** `cmd/api` (`make dev`) runs `Tick` **every minute**, starting at start-up; **`JOBS_TIMER=false` switches it off** (`internal/config`; default on). `cmd/staff-api` and `cmd/patient-api` never run it. The start-up log says "jobs timer every 1m0s" or "off".
- **`internal/invite`** (moved out of `internal/httpapi`, so the jobs can send without an HTTP request): `Sender.Send(ctx, episode, by, kind)` with kinds `First` (creating the episode), `Again` (Resend) and **`Reminder`**. A reminder uses the hospital's **reminder templates** (S10's `templates.reminderSms`/`reminderEmail`), subject "<hospital>: a reminder about your health questionnaire", and a **new link that's added beside the patient's earlier ones** (they keep working). The first invite that goes sets `invited_at` and creates the follow-up job; a resend changes neither. The invite's and Resend's behaviour and wording are unchanged (their tests pass untouched).
- **General notes (automatic):** "Day 2 reminder sent by text message and email." (or "… not sent. The patient has no mobile or email." etc.), and "Not responding: the HQ hasn't been sent 7 days after the invite."
- **`GET …/episodes/{eid}`** now also returns `invitedAt`, `remindersSent`, `notRespondingAt`; each event has `automatic`.
- **QA routes (MESSAGING=capture only, no sign-in, like `/api/qa/messages`; the responses hold counts, never patient data):**
  - `POST /api/qa/jobs/run {asOf}` (RFC 3339) → `{asOf, jobs, reminders, notResponding, failed}`; a bad or missing time → 400;
  - `GET /api/qa/jobs` → `{timer, intervalSeconds, offsetMinutes, lastTickAt, lastTickAsOf}`;
  - `PUT /api/qa/jobs/clock {offsetMinutes}` (0 to 90 days) → 204; else 400.
- `settings.Get(values, key, &dst)` reads a setting's value (used for the schedule).

### Web app

TypeScript for Rahul to review, in plain terms:
- **`features/qa/ScheduledJobs.tsx` (`/qa/jobs`, no sign-in):** "Run the jobs now, as of a later time": buttons **Now / Now + 2 days / Now + 5 days / Now + 7 days** fill a date-and-time box (your zone, rounded up to the next minute so a preset is never early), **Run jobs** shows "Ran 1 job as of …: 1 reminder sent, 0 marked "not responding"." **The timer:** on/off, its last run, and **Move the clock** by days and hours, or **Back to real time**. It reloads the timer's state every 15 seconds. Times are helpers in `jobsTime.ts` (tested in `jobsTime.test.ts`).
- **`/qa/messages`** links to it, and it to the messages.
- **Episode page:** a red **"Not responding"** badge beside the status while the HQ hasn't been sent; the invite card adds "Reminders go on the hospital's reminder days after the first invite, until the patient starts answering. The General notes list each one." **General notes:** automatic entries show "| Automatic on …".
- `routes.tsx`: `/qa/jobs` (pinned in `routes.test.ts`).

### Journeys

- **J8** (`e2e/journeys/test_j8_reminders.py`): an episode's invite; the jobs page as of now + 2 days → a reminder by text and email with a new link (also on the captured-messages page); the same again → nothing; the episode's notes ("Day 2 reminder sent…", "Automatic"); the patient opens the **reminder's** link on an iPhone 13 and starts answering → now + 5 days sends nothing; now + 7 days → "Not responding" on the episode, no message; the patient sends their answers → the badge goes.

## Decided by the builder (automatic mode)

- **One follow-up job per episode**, rescheduled to its next due time, rather than one row per reminder: the schedule is read from the hospital's settings at each run, so a changed setting applies to episodes already invited.
- **The days count from the first invite that reached the patient** (`invited_at`), not the episode's creation; a Resend doesn't restart them. If the first invite couldn't go (no contacts), the count starts at the first successful Resend. **Reason:** the requirement is "days since invite"; a patient who was never reached can't be reminded.
- **Day n = the same local time n calendar days later in the hospital's zone** (not n × 24 hours). **Reason:** the per-hospital zone (S08) exists for exactly this; with no time window (answer 1) a reminder simply keeps the invite's time of day.
- **A reminder's link is new, and the patient's earlier links keep working** (unlike a Resend, which revokes them). **Reason:** a patient who taps the original invite after the reminder must still get in, and a reminder must not end a session they have open. Only a token's hash is stored, so the old link can't be resent.
- **Missed runs send only the latest due reminder**, once (no burst of several reminders).
- **"Started" = any patient answers saved for the episode, including staff entering them on the patient's behalf** (PAT-09: staff on the phone are completing it). Opening the link and confirming the date of birth without answering doesn't count (answer 2: "answers anything").
- **"Not responding" still comes on day 7 for a patient who started but didn't send** (EPI-02: "no submit"); it shows until the HQ is sent, then the badge goes (the timestamp stays).
- **Suspended trusts or hospitals get nothing** while suspended (their links are closed, ORG-03); the follow-up looks again hourly and catches up once active.
- **The invite is still sent in the Create request**, not moved to a job (C2's S14 status line said "S16 moves it to jobs", but the brief doesn't list it): staff keep the immediate "Invite sent by …" note, and nothing changes for S14's tested behaviour. S17 can move it if Rahul wants (pending.md).
- **The jobs write no audit rows.** They aren't a person viewing or changing data (S11's actors are staff or patient); what they did is in the episode's General notes. If Rahul wants a "system" actor in the audit log, that's a small follow-up (pending.md).
- **The QA controls are served only with `MESSAGING=capture`** (local and QA), like the captured messages: they exist nowhere real messages are sent (S17's CDK check refuses capture outside QA). **Two controls:** "run as of" (the brief's), and a **clock offset for the timer**, so a tester can see the timer itself send a reminder within a minute (the brief's test focus 5) without waiting days.
- **`JOBS_TIMER`** (default on) is read by `internal/config`; only `cmd/api` acts on it.

## Round 1 (test report 1: 20 passed, 1 failed)

**X-1 (Minor, security): reminders multiplied the date-of-birth tries.** S14 locks a link after five wrong dates of birth until staff resend; S16's reminders add links beside the invite, each with its own five tries, and a reminder after a lock handed the patient a fresh working link. Fixed test-first as the conductor directed (S14's intent kept):
- **Tries are counted per episode** (migration **`00017_episode_dob_lock.sql`**: `episodes.dob_failed_attempts`; existing episodes take their live links' highest count, and one locked live link locks the others). `FailInviteDob` counts a wrong date against the episode and, at the fifth, sets `locked_at` on **every** live invite link of the episode, so all the existing checks (Continue, date of birth, and sessions opened from any of them via `TouchPatientSession`) see the lock. `PassInviteDob`: a right date on any link resets the episode's count. Sign-in links (S15) keep their per-link count (`FailLinkDob`). The unused `PassLinkDob` was removed.
- **Only a Resend clears the lock:** `invite.Send` (First/Again) resets the count when it revokes the older links (`ResetEpisodeDob`); a reminder doesn't.
- **A locked episode gets no reminders:** `Due` has `Followup.Locked` and `Actions.Withheld`. A reminder day that passes while locked is **counted but not sent, and not sent later**; the General notes say "Day 5 reminder not sent: the patient's link is locked after 5 wrong dates of birth. Resend the invite to send them a new link." **After a Resend, the later reminder days carry on** from the first invite (e.g. locked before day 2, resent on day 3: day 2's is skipped, day 5's goes). "Not responding" still comes on day 7.
- **Staff banner:** `GetEpisodeInvite`'s `lockedAt` is the episode's lock (the newest live link's, or any other live link's). The banner now reads "…so every link sent to the patient for this episode is locked, and no reminders go. Resend the invite to send them a new link."
- **Tests first:** 3 new `Due` cases; `TestEpisodeDobLock` (integration): wrong dates on the invite and the reminder's link share the count and lock both at the fifth (right date, Continue refused on each; no live unlocked link; staff told); a right date on either link resets the count; another episode's tries don't count; the lock ends a session opened from the other link; a locked episode gets no reminder (a note, the day counted, no new link) and still "not responding"; Resend clears it (4 tries left on the new link, the old one 404) and day 5's reminder then goes. `readLink` in `patient_integration_test.go` reads the episode's count for invite links. The other routes' five cases are unchanged: the patient's link has no staff sign-in or role; "another trust" is "another episode's tries don't count"; invalid input (an unreadable date) still isn't a try (S14's test, passing).
- **Also (exploring 7):** the jobs page's form has `noValidate`, so a half-typed **Run as of** shows "Enter a date and time." instead of doing nothing.
- **Logged in pending.md:** the reminder at the same moment as "Not responding" on catch-up (for Rahul), and the cosmetic notes (double-click Run jobs, real vs "as of" times, double audit rows in dev).
- **Results:** `make lint`, `make test` (Vitest 143 + 116), `make test-integration` all pass; `make e2e` (after `make migrate seed`) **43 passed**.

## Round 2 (test report 2: 25 passed, all)

Tested 13 Oct. No code changed. Rahul's answers 3–5 (answers.md) confirm the audit choice, staff answers counting as started, and that a locked episode stays reachable through an S15 sign-in link. Logged in pending.md: the banner rewording ("the invite links are locked"), the signed-out card pointing to a now-locked link, and the stored try count passing 5 under parallel requests (no effect).

## Tests

- **Unit:** `internal/jobs/due_test.go` (21 cases), `internal/settings` (`TestGet`), `internal/config` (`JOBS_TIMER`), `apps/web/src/features/qa/jobsTime.test.ts`, `routes.test.ts`.
- **Integration (API → database)**, `internal/httpapi/jobs_integration_test.go`, through `POST /api/qa/jobs/run`: the first invite starts the follow-up; nothing before day 2; day 2's reminder by text and email with the reminder template and a new link, both links working; the same time again and a later time before day 5 send nothing; the episode route's new fields and the automatic note; day 5; day 7's flag, once, no message, the job finished; submitting stops everything; starting stops reminders but not the flag; staff answers count as started; a resend doesn't restart it; no invite, no follow-up; a suspended hospital waits, then catches up; a hospital's own days, template and zone (Pacific/Auckland); **four runners at once send one reminder per episode**; invalid input to all three QA routes (400); the clock read, moved and set back; `JOBS_TIMER=false` shown. The QA routes have no sign-in, so the no-sign-in, wrong-role and other-trust cases don't apply (as for `/api/qa/messages`).
- **`internal/jobs/tick_integration_test.go`:** `Tick` (the timer's run, which has no route) with the apitest database harness, a stated exception like `cmd/platform-admin`'s: nothing due on real time; with the clock 2 days ahead the reminder goes and the run is noted; without capture mode the offset is ignored.
- **End-to-end:** J8 (above).
- **Checked by hand against `make dev`:** the timer with the clock moved 2 days ahead sent a waiting reminder 25 seconds later; the jobs page renders as described.

### Results (13 Oct)

- `make lint`: passes (go vet with the integration tag, tsc in every workspace).
- `make test`: Go all ok; Vitest 143 (web) + 116 (clinical) passed.
- `make test-integration`: every package ok (httpapi 13.4 s, jobs 13.7 s).
- `make e2e` (against `make dev`, after `make migrate seed`): **43 passed** in 188 s, J8 included.

## Suggested commits

1. **Jobs, reminders and "not responding": data, Go and API tests (C2 / S16: data and API)** — `00016_jobs.sql`, `00017_episode_dob_lock.sql`, `db/queries/{jobs,episodes,links}.sql`, `internal/db/*` (generated), `internal/jobs/*`, `internal/invite/*`, `internal/httpapi/{invites,router,qajobs,patient,accounts}.go`, `internal/httpapi/{jobs,patient}_integration_test.go`, `internal/settings/*`, `internal/config/*`, `internal/serve/serve.go`, `cmd/*/main.go`, `cmd/seed/main.go`, `.env.example`.
2. **The scheduled-jobs page, "Not responding" on the episode, and J8 (C2 / S16: web and journeys)** — `apps/web/src/features/qa/*`, `apps/web/src/pages/QaJobs.tsx`, `apps/web/src/routes*.ts*`, `apps/web/src/features/episodes/{EpisodePage,GeneralNotes,types}.tsx?`, `apps/web/AGENTS.md`, `e2e/journeys/test_j8_reminders.py`, `e2e/README.md`.
3. **Docs: S16 brief, answers, build notes, test plan, test reports, verification and status.**
