# S17: Jobs and messaging on AWS

*AWS phase (12 Oct): runs with Rahul there.*

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 5 of 5. C2 is done after this session.
**Requirements:** A-13, PAT-03 and PAT-11 for real, OPS-05 (failed deliveries).
**Depends on:** S12 verified. **Rahul has SES production access and the SMS sender registration** (the paperwork started in C2).

**Ask Rahul first:**
1. **The sending domain and SMS sender ID** to use, and the address for delivery-failure alarms.
2. **Who receives the one real test message,** e.g. Rahul's own phone and email.

## Build

- **`cmd/jobs`:** a Lambda entry point that calls the same `jobs.RunDue` as S16's local timer.
- **CDK, per country entry:**
  - an **EventBridge Scheduler** rule every 5 minutes for the jobs Lambda;
  - **`MessagingStack`:** the SES identity (sending domain), and the SMS configuration with a protect configuration allowing only the country's numbers;
  - **CDK refuses `MESSAGING=capture` on any stage except QA**, and refuses real sending on QA unless it's named in the country entry.
- **The real `Deps.SMS` and `Deps.Email`** (AWS End User Messaging SMS and SES v2), behind the same interfaces as the fakes.
- **Delivery failures** are recorded on the message and shown to the hospital as an undelivered invite (OPS-05), with an alarm.

## Done when

- On QA, the schedule fires by itself, and reminders appear in the captured messages.
- With real sending switched on for one test run, Rahul receives a real SMS and email, and the link works on his phone.

## Manual test focus

On the **QA URL**:
1. Create an episode. The invite is captured, and after 5–10 minutes the schedule has run by itself.
2. Rahul switches on real sending for his own number and address. He gets the SMS and email; the link → Continue → date of birth works on a real phone.
3. A message to an invalid number shows as undelivered to the hospital.
