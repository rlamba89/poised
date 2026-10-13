# S16 verification

## Verification 1 (13 Oct 2026, verifier, automatic run)

**Verdict: Verified.** The brief is fully built, with Rahul's answers 1–5 applied. All automated tests pass. The spot checks in the browser match the build notes. None of the findings below blocks the session. Each is minor or limited to QA, and each is logged in [pending.md](../../pending.md) §3 for later.

### What was checked

- **Scope:** everything in the brief is built:
  - the `jobs` table with `FOR UPDATE SKIP LOCKED`;
  - `jobs.RunDue(ctx, now)`;
  - the pure `Due` function, with 21 unit cases, test-first;
  - the day 2 and day 5 reminders, "not responding" on day 7, and no double sending;
  - the local timer in `cmd/api`, which `JOBS_TIMER=false` switches off;
  - the `/qa/jobs` "run as of" page;
  - capture-only messages.

  Brief item 4 (the 09:00–19:00 window) was dropped, per answer 1. Nothing large was added beyond the brief. The extras are the testers' clock offset and the per-episode date-of-birth lock, which came from test report 1's X-1 under the conductor's direction.
- **Lambda-safe jobs:**
  - **No server memory or disk.** `RunDue` keeps nothing between calls and reads everything from Postgres.
  - **Overlapping runners.** `ClaimDueJob` claims one row at a time with `FOR UPDATE SKIP LOCKED` and moves `run_at` on by a 10-minute lease, so no row lock is held while the job runs. `TestFollowupRunnersAtOnce` runs four runners at once and gets one reminder per episode.
  - **Counted before sent.** `CountReminder` is a compare-and-set: `reminders_sent < count AND patient_submitted_at IS NULL`. It runs before `Inviter.Send`, so a reminder goes at most once. If the process crashes in between, the reminder is lost rather than sent twice, as recorded in pending.md.
  - **Bounded runs.** Each call runs at most 500 jobs.
- **Time zones and DST:** day *n* is `invited_at.In(zone).AddDate(0,0,n)`, using the hospital's zone or the deployment default (A-20). It is unit-tested across the March clock change and integration-tested with Pacific/Auckland.
- **Tenant scoping:**
  - Each job is keyed by one episode. `GetFollowup` and `GetInviteDetails` read that episode's own hospital and trust, so a job never reads or writes another episode's data.
  - The QA responses carry counts only.
  - Spot check: Bea Author (Trust B) gets 404 on the Trust A episode, under both hospitals' paths.
- **The QA controls are impossible outside capture mode:**
  - `/api/qa/jobs*` are mounted only when `Messaging == capture`, in the same block as `/api/qa/messages`.
  - `Tick` applies the clock offset only when `qaClock`, which is set from capture mode. `tick_integration_test` shows the offset is ignored without capture.
  - Today capture is the only mode `config.Load` allows. S17's brief already says the CDK refuses capture outside QA.
- **Lock rule (report 1's X-1 fix):**
  - Wrong tries are counted per episode in `FailInviteDob`.
  - The fifth try locks every live invite link.
  - A locked episode gets no reminder: `Due.Withheld`, with a note.
  - Resend resets the count.
- **Security:**
  - No secrets in the diff.
  - Log lines carry only episode and job IDs, never patient data.
  - Reminder texts use the hospital's templates (first name, hospital name, link), as the invites do.
  - The staff routes are unchanged.
- **Hardcoding (A-20):** the grep prints only existing comments in `apps/web/src/lib/format.ts` and `internal/patient/patient.go`. Nothing new.
- **The Lifebox grep:** prints nothing.
- **Docs:**
  - decisions.md (S16 section and answers 3–5), pending.md, the C2 plan Status and both READMEs are updated.
  - The `messages()` move to `settings.Effective(…, own)` is the same as S14's `hospitalOnly(own)`, because `Effective` already drops non-hospital keys.

### Findings (worst first; none blocks)

1. **A Resend can hand out a link that's already locked (race, fails closed)** — `apps/api/internal/invite/invite.go:161` with `apps/api/db/queries/links.sql` (`FailInviteDob`).
   - **What happens:** `Send` stores the new link before sending. If the patient's fifth wrong try lands while the messages are going, `FailInviteDob` locks the new link too. `settle` then resets the episode's count but not that link's `locked_at`.
   - **Effect:** the patient's fresh link says "locked", and staff see the banner and must resend again.
   - **Fix:** in `settle`, for First/Again, also clear the new link's `locked_at` and `failed_dob_attempts`. This is safe because `settle` holds the episode row lock, which `FailInviteDob`'s episode update waits on.
   - **Why it's minor:** the window is short and the result is safe; only a step is wasted.
2. **A reminder link made just after a lock comes out unlocked (race, fails open, negligible)** — `apps/api/internal/jobs/jobs.go:148`.
   - **What happens:** `GetFollowup` reads `locked=false`. The fifth wrong try then locks the existing links before `Send` creates the reminder's link.
   - **Why it's negligible:** the link goes only to the patient's own contacts, and the episode's count is already at 5 or more, so one wrong try on it locks it again.
   - **Root fix, if wanted:** a single `episodes.dob_locked_at` that the link checks read, instead of copying the lock onto each link.
3. **QA "run as of" uses times that a tester chose (QA only):**
   - `apps/api/internal/httpapi/qajobs.go:37` runs `RunDue` with `r.Context()`. A tester who closes the tab mid-run can leave a reminder counted but not sent. `context.WithoutCancel` would fix it, as Resend already does.
   - `apps/api/internal/jobs/jobs.go:92` takes the lease and the backoff from `now`, which is the as-of time. A job that fails during a "+7 days" run is hidden from the real-time timer until real time reaches that day.
4. **`GET /api/qa/jobs` reports `cfg.JobsTimer`, not whether this process runs the timer** — `apps/api/internal/httpapi/qajobs.go:58`.
   - staff-api and patient-api never run the timer but would say "On". Locally only `cmd/api` serves the page, so it's right today.
   - **For S17:** pass the effective value in `Deps`.
5. **The staff "Last sent …" shows the newest reminder's time** — `apps/api/db/queries/links.sql` (`GetEpisodeInvite`).
   - Reminder links are live invite links too. Seen in the browser: the invite went at 22:15 and the reminder at 22:16, and the card said "Last sent 22:16".
   - This is cosmetic: the General notes list each send.
6. **If `settle` fails after the messages went, the note says "not sent" and a first invite doesn't start the follow-up** — `apps/api/internal/invite/invite.go:130`. This is a database failure between sending and recording. The same pattern was in S14 before; S16 only adds the follow-up start.
7. **J7 (and J8) can fail by chance (test flake, older than S16)** — `e2e/journeys/test_j7_patient_invites.py:22`, `test_j8_reminders.py:31`.
   - **What happens:** the made-up mobile is one of only 1,000 numbers (`07700 900nnn`), and the local database already holds captured texts to 177 of them. Roughly one run in six reuses a number that already has messages, so the journey's "exactly 1 text" check fails.
   - This verification's `make e2e` run hit it: J7 found 3 texts. A rerun of J7 passed (6 passed).
   - **Fix:** filter the captured messages by time or by the episode's link, or use a wider made-up range.
8. **Cleanups (not bugs):**
   - The step that decodes the trust's and hospital's settings and combines them with `settings.Effective` is repeated in `jobs.schedule`, `invite.messages` and `trust_settings`. A `settings.ForHospital` helper would replace all three.
   - `signinDob` repeats the same `FailLinkDob` call in two branches.
   - `invite.Sender.now()` repeats the server's default clock.

### Spot checks in the browser (Claude in Chrome, against the conductor's `make dev`)

1. **Reminder, no double send:**
   - A new Hospital A1 episode with email only.
   - `/qa/jobs` → **Now + 2 days** → **Run jobs**: "Ran 13 jobs … 13 reminders sent". This episode got exactly one reminder, with the subject "Hospital A1: a reminder about your health questionnaire".
   - The same time again: `jobs: 0`, still 2 messages.
   - The General notes say "Day 2 reminder sent by email. | Automatic".
2. **The shared date-of-birth lock and a withheld reminder:**
   - Three wrong dates on the invite link, then two on the reminder's link: the fifth is 403 "locked", and the right date is then refused on both links.
   - As of +5 days: no new message, `remindersSent` is 2, and the note "Day 5 reminder not sent: the patient's link is locked…" appears.
   - The staff banner says "…every link … is locked, and no reminders go".
3. **"Not responding":** as of +7 days, the red **NOT RESPONDING** badge appears beside "HQ NOT COMPLETE", with the automatic note.
4. **Resend clears the lock:**
   - Resend → "Invite sent again by email". Both old links return 404.
   - The new link → Continue → the right date of birth → the patient's HQ home ("Hello Vera…").
   - The window could not be resized to 390×844 (it stayed 1512 wide). The phone size is covered by J8 at iPhone 13.
5. **Another trust:** Bea Author gets 404 on the episode, including its new follow-up fields.

The jobs clock offset was left at 0. I set no offset.

### Test results (13 Oct)

- `make lint`: passes.
- `make test`: Go all ok; Vitest 143 (web) + 116 (clinical) passed.
- `make test-integration`: every package ok (httpapi 14.8 s, jobs 15.1 s).
- `make e2e` (after `make seed`, against `make dev`): **42 passed, 1 failed**. The failure is J7's count of captured texts, from a reused made-up mobile (finding 7). Rerunning `test_j7_patient_invites.py`: **6 passed**.

### Hand checks for Rahul

- None new. The patient screens at 390×844 in a real window are still the existing S11 hand check.

### Suggested commits

1. **Jobs, reminders and "not responding": data, Go and API tests (C2 / S16: data and API)**
   - Migrations `00016_jobs.sql` and `00017_episode_dob_lock.sql`.
   - Queries and the generated `internal/db`.
   - `internal/jobs`, `internal/invite`, `internal/settings`, `internal/config`, `internal/serve`.
   - In `internal/httpapi`: `invites`, `router`, `qajobs`, `patient`, `accounts`, and the `jobs` and `patient` integration tests.
   - `cmd/*/main.go` and `.env.example`.
2. **The scheduled-jobs page, "Not responding" on the episode, and J8 (C2 / S16: web and journeys)**
   - `apps/web/src/features/qa/*`, `apps/web/src/pages/QaJobs.tsx`, `routes.tsx` and `routes.test.ts`.
   - The episode page, General notes and types.
   - `apps/web/AGENTS.md`.
   - `e2e/journeys/test_j8_reminders.py` and `e2e/README.md`.
3. **Docs: S16 brief, answers, build notes, test plan, test reports, verification and status**
   - S16's session folder, decisions.md, pending.md, the C2 plan, `docs/plans/README.md` and `docs/sessions/README.md`.
   - These files mix in 12 Oct planning changes and are committed whole.
