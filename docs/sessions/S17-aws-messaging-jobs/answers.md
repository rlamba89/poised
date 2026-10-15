# S17: answers

Asked by the conductor (`/run-sessions S06 S17`) on 14 Oct 2026, before the AWS-phase run started.

1. **SES and SMS are both in the sandbox** in account 026090536258 (SES: 200/day, verified addresses only, no identity yet; SMS: sandbox tier, $1 spend limit, no sender ID). **Answer: proceed in this account**; S17 tests in the sandbox to Rahul's own verified email and phone only. Production access is requested later, in whichever account becomes the demo account.
2. **No domain yet:** test email goes from a verified address of Rahul's.
3. **Still to ask when S17 starts:** permission for the real test messages (not granted yet), the sender address, SMS sender ID, the alarm address and the test recipients (kept out of the repo).

## Asked when S17 started (14 Oct, by the conductor)

4. **Real messages:** **yes, to Rahul only** (recommended): a handful of test emails and texts to his own verified email and phone, in the SES/SMS sandbox. Nobody else can receive anything. QA otherwise stays in capture mode.
5. **How his email and phone reach AWS:** **Rahul tells the builder when asked** (in chat). They must still never be written into the repo, logs or committed docs; store them only in SSM (or pass them to the verification calls directly).
6. **Delivery-failure alarms:** **the same address as the budget alarm** (`/sj/alert-email`, from `make aws-setup`).
7. **QA jobs interval** (Neon free tier; asked 14 Oct): **15 minutes** on QA (about 60 compute-hours a month). Training and Production stay at 5 minutes.
8. **Push plan** (conductor, 14 Oct): no push until S17 is committed; then one push covers S07's docs, S12 and S17.
9. **SMS sender** (Q4, 14 Oct; the account has no origination identity, so AWS refuses even the sandbox verification code): **try a UK alphanumeric sender ID "Poised" through CDK** (recommended). If AWS requires UK registration first or charges any monthly fee, release it at once and fall back to: no real text in S17 (logged in pending.md for the demo account).
10. **S17's staff-side test cases** (test report 1 was blocked: agents won't type passwords into Cognito's hosted login on QA): **test them locally** on `make dev` (same code, the dev login, captured messages) (Rahul, 15 Oct). QA's AWS parts stay proven by test report 1's T-08/T-02 and the builder's real sends.
