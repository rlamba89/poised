# S17: questions for Rahul (builder, 14 Oct 2026)

## Round 3 (after the sender ID): answered 14 Oct

*Rahul received the code from POISED; the mobile is verified, and one real invite text reached him ("got it"). No open questions.*

Q4 is answered (answers 9). The UK sender ID **POISED** now exists through CDK, with a **$0.00 monthly fee**. It isn't registered: AWS accepted it unregistered, and the docs say registration is needed only for MEF-protected names.

**AWS has texted Rahul's mobile a verification code from POISED.** Rahul, please reply with that 6-digit code. If no text arrived within about 5 minutes, say so: that means the UK needs registration after all, so the builder releases the sender ID and falls back to option 2.

After the code: the builder verifies the mobile and sends one real test invite text to it, which should arrive from "POISED". Then the test plan follows.

## Round 2 (after the deploy)

**Where it stands:** S17 is on QA (deployed by hand, 14 Oct). The schedule runs the jobs by itself every 15 minutes (answers 7). Rahul's email and mobile are in SSM only.
- **SES:** the sender identity (Rahul's email) is already **verified**, so no link to click. **A real invite email was sent to Rahul's address** (it may be in spam).
- **The bounce test works end to end.** An episode for `bounce@simulator.amazonses.com` showed "Invite not delivered" with its note within about a second of the send.
- **The alarm subscription** to the alert address is already **confirmed**.
- **SMS is blocked:** AWS refuses to text anything, even the sandbox's verification code to Rahul's mobile: `NO_ORIGINATION_IDENTITIES_FOUND`. The account has no sender ID or phone number to send from. The app handles it: the invite's text shows as "couldn't be sent", the episode shows it as undelivered, and the alarm counts it.

### Q4. How should QA send a real text?

1. **(Recommended) Request a UK sender ID, "Poised", through CDK.** Set `sms_sender_id="Poised"` on `uk-qa`; MessagingStack already makes it (`AWS::SMSVOICE::SenderId`). Then the builder:
   - texts Rahul's mobile its sandbox verification code;
   - asks for the code;
   - sends the one real test text.

   **Unknowns:**
   - AWS's table says UK sender IDs need **registration** (a console form with company details, which takes days). Its UK page says that applies to MEF-protected names only.
   - A sender ID may carry a **monthly lease fee**, which AWS shows only once one is requested.

   **If you agree:** the builder requests it, reads its fee and registration status, and **releases it at once if there's any fee, or if it can't send without registration.** Then it falls back to option 2.
2. **No real text in S17.** The real email, the bounce and the SMS failure path are proven. The SMS code paths are covered by the automated tests. A real text waits for the demo account, with a registered sender ID and production access. The builder adds that to pending.md and writes the test plan now.
3. **A UK phone number (long code)** instead of a sender ID. It has a monthly fee and also needs UK registration. Not recommended.

---

## Round 1 (answered 14 Oct: answers.md items 7–8; contacts given in chat)


S17 is built and every automated test passes locally ([build-notes.md](build-notes.md)). To put it on QA and send the one real test message, the builder needs the following. **Answer in chat: none of your contacts is written into the repo, a log or a doc.**

## Q1. Your email address and mobile (needed to deploy)

- **Your email address** is used twice: as **the address patients' emails come from** on QA (SES; stored only in SSM `/sj/qa/email-from`, read by CloudFormation at deploy time), and as **a test recipient**.
- **Your mobile** (UK, e.g. `+447…`) is **a test recipient**.
- The test recipients go only into the SSM SecureString `/sj/qa/messaging-test-recipients`. QA still captures every message; only messages addressed to these two (and SES's own bounce-test addresses) are really sent. Nobody else can receive anything from QA.
- **Recommended:** the same personal address as sender and recipient (one SES verification). If you'd rather send from a different address of yours, give both.
- Note: an email "from" a Gmail address sent by SES isn't signed by Gmail, so it may land in **spam**; Gmail's policy (`p=none`) still delivers it. A domain fixes that later.

**What happens next (one more stop):** the builder deploys QA by hand (MessagingStack, the jobs Lambda and its schedule) and starts the sandbox verifications. Then it asks you, in one go, to:
1. click **SES's verification link** (to your email);
2. send back the **code texted to your mobile** (SMS sandbox);
3. click **SNS's "Confirm subscription"** email for the delivery-failure alarm (sent to `/sj/alert-email`).

## Q2. How often the jobs run on QA (Neon's free tier)

The brief says every 5 minutes. **Each run wakes QA's Neon database, which sleeps only after 5 minutes idle, so a 5-minute schedule keeps it awake all month:** about 180 compute-hours at Neon's smallest size, against the free plan's **100 a month**. Past that, Neon stops the database until the next month (QA down), unless the project is on a paid plan. The interval is a per-entry setting (`jobs_every_minutes` in `countries.py`); Training and Production use 5 (RDS doesn't sleep).

1. **(Recommended) QA every 60 minutes, after S17's test.** The builder deploys with 5 for the test (a day or two costs about 6 compute-hours a day), then sets QA to 60. Neon wakes about once an hour (roughly 20 compute-hours a month). Reminders are counted in days, and testers use **Run jobs now** on `/qa/jobs` as before; only "the schedule fired by itself" takes up to an hour to see.
2. **Keep 5 minutes on QA.** Upgrade QA's Neon project to a paid plan (usage-based; roughly $20–30 a month always-on, to be checked on Neon's pricing page), or accept QA stopping around mid-month.
3. **QA every 15 minutes:** Neon awake about a third of the time (roughly 60 compute-hours), close to the limit with testing and pipeline runs on top.

## Q3. SMS sender ID (no action now)

No sender ID is registered (answers item 3), so the texts go without one and AWS picks how to send. **Recommended:** try it at the real test. If the UK sandbox refuses texts without a sender ID, the builder comes back with the options (a UK sender ID needs registration in the console, which takes days; `sms_sender_id` in `countries.py` then uses it).

## Before you answer: pushing

The pipeline last ran S07's code; S12 isn't pushed yet. **If you're pushing S12, push it now and let that run finish before answering.** After the builder deploys S17 by hand, **don't push again until S17 is verified and committed**: a push of code without S17 would take its jobs Lambda and schedule off QA in the middle of testing.
