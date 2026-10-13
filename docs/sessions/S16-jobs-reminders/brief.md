# S16: Scheduled jobs and reminders (local)

*Local phase (12 Oct): runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 4 of 5. The jobs Lambda, the EventBridge schedule and real SES/SMS are S17 (AWS phase).
**Requirements:** reminders on day 2 and day 5, and "not responding" on day 7 (section 5.1).
**Depends on:** S15 verified.

**Ask Rahul first:**
1. **Send reminders only between 09:00 and 19:00 in the hospital's time zone?** **Recommended: yes.** It needs the per-hospital time zone (S08).
2. **What stops reminders?** **Recommended:** submit stops them. Starting the HQ doesn't.

## Build

- **The `jobs` table.** Due rows are claimed with `FOR UPDATE SKIP LOCKED`.
- **`jobs.RunDue(ctx, now)`:** one function that claims and runs the due jobs. Locally, **a timer in `cmd/api` calls it every minute** (it can be switched off). In S17, a Lambda calls the same function, so nothing here may depend on the server staying up.
- **`due(episode, now) → actions`,** a pure function written test-first, covering:
  - the reminders on day 2 and day 5;
  - the **"not responding" flag on day 7**, shown on the episode (the worklist itself is C3);
  - no double sending.
- **A dev/QA-only "run jobs as of <date and time>" page,** so testers can jump days ahead.
- **All messages stay in capture mode** (S14). Nothing is really sent.

## Manual test focus

1. Create an episode. Run jobs as of +2 days → reminder 1 is captured. +5 days → reminder 2. +7 days → the "Not responding" flag.
2. Submit before day 5 → no more reminders.
3. Run the jobs twice for the same time → nothing is sent twice.
4. **The time window:** a run at 22:00 hospital time sends nothing, and the next morning's run sends it.
5. With the local timer on, a due reminder appears in the captured messages within a couple of minutes, without using the "run jobs" page.
