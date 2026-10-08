# S16: Scheduled jobs, reminders and messaging on AWS

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 4 of 4. C2 is done after this session.
**Requirements:** reminders on day 2 and day 5, and "not responding" on day 7 (section 5.1), plus A-13.
**Depends on:** S15 verified. Rahul should have started SES production access and SMS sender registration.

**Ask Rahul first:**
1. **Send reminders only between 09:00 and 19:00 in the hospital's time zone?** **Recommended: yes.** It needs the per-hospital time zone (S08).
2. **What stops reminders?** **Recommended:** submit stops them. Starting the HQ doesn't.

## Build

- **The `jobs` table.** Due rows are claimed with `FOR UPDATE SKIP LOCKED`.
- **`cmd/jobs`:** a Lambda entry point, which can also run locally.
- **`due(episode, now) → actions`,** a pure function written test-first, covering:
  - the reminders on day 2 and day 5;
  - the **"not responding" flag on day 7**, shown on the episode (the worklist itself is C3);
  - no double sending.
- **A dev/QA-only "run jobs as of <date and time>" page,** so testers can jump days ahead.
- **CDK:**
  - an EventBridge Scheduler rule every 5 minutes, per country entry;
  - **`MessagingStack`:** the SES identity and SMS configuration.
  - QA stays in capture mode. Real sending is switched on when the approvals arrive (Rahul checks it once).

## Manual test focus

1. Create an episode. Run jobs as of +2 days → reminder 1 is captured. +5 days → reminder 2. +7 days → the "Not responding" flag.
2. Submit before day 5 → no more reminders.
3. Run the jobs twice for the same time → nothing is sent twice.
4. **The time window:** a run at 22:00 hospital time sends nothing, and the next morning's run sends it.
5. On QA: the schedule fires by itself (check the captured messages after 5–10 minutes).
