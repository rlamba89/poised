# S17 build notes: jobs and messaging on AWS

Builder, automatic run (`/run-sessions`), 14 Oct 2026. Answers: [answers.md](answers.md) (items 1–6) and decisions.md §2 (14 Oct).

## Where it stands

- **Answers 7–8 (14 Oct):** QA jobs every **15 minutes** (Neon's free plan), Training/Production 5; no push until S17 is committed. Rahul's email and mobile were given in chat and are only in SSM (`/sj/qa/email-from`; the SecureString `/sj/qa/messaging-test-recipients`) and the verification calls.
- **On QA (deployed by hand, 14 Oct ~20:45–21:00 BST):** `make aws-setup EMAIL_FROM=…`, `make deploy-qa`, then `make deploy-stacks deploy-web` again (the jobs page's schedule text), then `make test-recipients`.
  - **The schedule fires by itself:** the jobs Lambda's first run at 19:47 UTC ("ran 11 … 0 failed"), `rate(15 minutes)`, ENABLED, no retries.
  - **SES:** the sender identity (Rahul's email) is verified, and the alarm topic's email subscription is confirmed (neither needed a click).
  - **One real send** (a temporary journey, deleted afterwards): **a real invite email went to Rahul**. An episode for `bounce@simulator.amazonses.com` showed "Invite not delivered" plus its Automatic note **about 1 second after the send**: SES → SNS → jobs Lambda → `episode_messages`. **The delivery-failure alarm went to ALARM** about 2 minutes later (2 `DELIVERY_FAILED` lines).
  - **SMS is blocked:** AWS won't text anything, not even the sandbox's verification code, with `NO_ORIGINATION_IDENTITIES_FOUND` (no sender ID or number in the account). From the Lambda, `SendTextMessage` reports `AccessDeniedException`; the policy simulator allows the configuration set's ARN and implicitly denies `*`, so with no origination AWS seems to authorise against a resource outside the account. Revisit once a sender ID exists. The app shows the text as "couldn't be sent" and as undelivered. **Rahul's choice is in [questions.md](questions.md) round 2 (Q4).**
- **Answers 9 (14 Oct): a UK sender ID through CDK.** `uk-qa` has `sms_sender_id="POISED"`. **CloudFormation's `AWS::SMSVOICE::SenderId` takes capitals only** (`^[A-Z0-9_-]+$`; "Poised" failed early validation), so `countries.py` now checks for capitals.
  - The deployed sender ID `POISED/GB` shows **`MonthlyLeasingPrice` 0.00** and **`Registered: false`**. The pricing page lists no sender ID fee for the UK, only long codes at $2 a month.
  - **The sandbox verification code was sent from it to Rahul's mobile** (AWS accepted it). Waiting for Rahul to report the code, which also shows whether an unregistered UK sender ID delivers. `test-recipients.sh` now sends the code from the entry's sender ID.
- **Answers 9, done (14 Oct):** Rahul got the code from POISED, so the mobile is verified (`make test-recipients-verify`). **One real invite text went to it from POISED, and Rahul confirmed it arrived.** On AWS's side: the episode's `episode_messages` row holds a real End User Messaging message ID (not `capture-…`) with no failure, no `DELIVERY_FAILED` line was logged, and no failure event came back. The configuration set publishes only failure events, so a success shows as their absence. **An unregistered UK alphanumeric sender ID delivers.** The earlier `AccessDeniedException` is gone now that the account has an origination identity.
- **A failed sign-in-link send now answers 202, like "no account"** (pending.md's "For S17" item from S15 verification 1, finding 4). Before, it was a 502 only for contacts with an account, which gave away that the account exists. Test: `TestSignin/a send that fails…` (written failing first).
- **The real-send episodes' reminders are switched off** (their `followup` jobs set done on QA's database), so a tester moving QA's clock can't text or email Rahul again. The test plan says to leave those episodes alone.
- **Fixed on the way:**
  - `test-recipients.sh` used a wrong filter name;
  - AWS's not-found errors print the number, so they're now hidden;
  - a failed SMS code no longer stops the email's step;
  - `make test-recipients` / `-off` now give the three functions a new description so they start afresh (they read the list at cold start; the first real send went to capture only because the warm functions predated the list).
- **Added after the deploy:** `JOBS_SCHEDULE_MINUTES` (set by CDK; `JOBS_TIMER=false` on AWS). `GET /api/qa/jobs` returns `runner: "schedule"` with its interval, and `/qa/jobs` says "On: a schedule on AWS runs the jobs every 15 minutes" (Vitest `whoRuns`, integration `TestQAJobsOnASchedule`, a CDK test).

## Testing

- **Test report 1 (QA, 14 Oct): blocked.** The tester agent won't type passwords into Cognito's hosted login on QA, so every staff case was blocked. Two checks passed: T-08 (the read-only AWS checks: schedule, SMS configuration set, protect configuration GB ALLOW / others BLOCK, alarms) and T-02's AWS side (the jobs Lambda ran every 15 minutes with nobody pressing Run jobs). No failures.
- **Answers 10 (Rahul, 15 Oct):** the staff-side cases were tested **locally** on `make dev` (same code, the dev login, captured messages). QA's AWS parts stand on report 1 and the builder's real sends.
- **Test report 2 (local, 15 Oct): all passed** (11 passed). T-04 steps 3–5 and T-05's second half were partly blocked: locally nothing bounces, so the undelivered badge, box and note can't appear. They're covered by the automated tests and by the builder's QA bounce-simulator run (14 Oct), and are hand checks below.
- **Found while exploring:** three cosmetic items, logged in pending.md (no code change):
  - S16's Run jobs summary can count more results than jobs;
  - the invite card's "Last sent" moves on with a reminder;
  - `cmd/jobs` logs nothing on a run with no due jobs.

## Hand checks for Rahul (on QA, signed in through Cognito)

1. **T-04 and T-05 on QA:** as Cara, create an episode with no mobile and the email `bounce@simulator.amazonses.com`.
   - Within about a minute, the episode shows the red **"Invite not delivered"** badge and the **"Not delivered"** box ("The invite's email wasn't delivered: the email address doesn't exist or doesn't accept email."), plus an Automatic General note.
   - **Resend invite** clears them, and after about a minute they're back for the new email only.
2. **T-06 with an undelivered row:** with that episode's URL, Val (Hospital A2) and Bea (Trust B) are refused (404), as for any A1 episode.
3. **H-1, the real text's link:** on the phone, open the link in POISED's text, then **Continue**, then the date of birth (17 May 1980), and the questionnaire opens. (The text and the real email already arrived on 14 Oct.)
4. **H-2, the alarm email:** an "ALARM: poised-uk-qa-delivery-failures" email at the alert address. The alarm fired on 14 Oct, the last time at 23:27 BST.
5. **H-3, nobody else:** nothing real goes to made-up contacts (by design; the journeys' and testers' messages were only captured).
6. **The `/qa/jobs` page on QA** (as Sam) says "On: a schedule on AWS runs the jobs every 15 minutes", and its last run moves on by itself.

## What was built

### Go

- **`cmd/jobs`** (new): the jobs Lambda. Called by the **schedule** (any event without SNS `Records`) it runs `jobs.Runner.Tick(now)`, the same as S16's local timer (with the testers' clock offset where messages are captured, so `/qa/jobs` works on QA as locally, and notes the run there). Called by the **SNS delivery topic** it records undelivered messages (`internal/delivery`). Run locally (`go run ./cmd/jobs`) it runs the due jobs once. `make build-lambdas` builds it.
- **`internal/serve.Setup`** (refactor): config, pool, senders and Cognito, shared by `serve.Main` and `cmd/jobs`. `serve.OnLambda()` exported.
- **`internal/messaging`:**
  - **The senders return the message's ID** (`SendSMS/SendEmail → (id, error)`): SES's or End User Messaging SMS's, `capture-<sent_messages.id>` when captured, `rec-<n>` for the test fake.
  - **`messaging.SES`** (SES v2 `SendEmail`: plain text, from `"Poised" <EMAIL_FROM>`, through the configuration set) and **`messaging.SMSAWS`** (End User Messaging SMS v2 `SendTextMessage`: transactional, through the configuration set, `SMS_SENDER_ID` as the origination identity when set), each behind a one-method client interface with a fake in the tests.
  - **A refused send logs `DELIVERY_FAILED …`** (the alarm's marker) and returns an error with **only AWS's error code**: AWS's messages can hold the address or number, which mustn't reach the logs.
  - **`messaging.TestRecipients`** (QA): captures every message and also really sends the ones to the named recipients (and SES's mailbox simulator, `@simulator.amazonses.com`).
- **`internal/config`:** `MESSAGING` is `capture` or `send`. `Sending` (`EMAIL_FROM`, `SES_CONFIGURATION_SET`, `SMS_CONFIGURATION_SET`, optional `SMS_SENDER_ID`) is required with `send`, and with `MESSAGING_TEST_RECIPIENTS` (capture only; refused with `send`). `EMAIL_FROM` and `MESSAGING_TEST_RECIPIENTS` can come from SSM (`…_PARAM`); a **missing test-recipients parameter means none** (`config.ErrNotFound`), so deleting it switches QA's real sending off without breaking the functions. Errors never print a recipient.
- **`episode_messages`** (migration 00018): one row per text or email about an episode (invite/Resend or reminder), with the sender's ID; `failed_at` + `failure` (words for staff) when it couldn't be sent or AWS reports it undelivered. `invite.Sender` records each message (the same time for a send's text and email).
- **`internal/delivery`:** `Parse` reads SES events (Bounce: Permanent/Transient/other, Reject, Rendering Failure) and SMS events (the ten `TEXT_*` failure types); `Record` marks the message and adds an **Automatic General notes entry** ("The invite's text message wasn't delivered: the mobile number isn't valid. Check the patient's mobile number, then resend the invite."), in one transaction, once (SNS can repeat); a message that isn't an episode's (a sign-in link) is only logged for the alarm. `HandleSNS` skips unreadable events and returns database errors (SNS retries).
- **`GET …/episodes/{eid}`** adds **`undelivered`**: failed messages since the latest invite or Resend (a later Resend starts afresh).

### Web

- **The episode page:** a red **"Invite not delivered"** badge beside the status, and a **"Not delivered"** alert in the invite card listing each ("The invite's text message wasn't delivered: … (Sent …)") with "Check the patient's mobile number and email address, then resend the invite." (`features/episodes/undelivered.ts`, Vitest).
  - **TypeScript for Rahul:** `undelivered.ts` is one type (the API's row) and one function building the sentence; `EpisodePage.tsx` shows the badge when the list isn't empty and the HQ isn't sent, and the alert inside the invite card.

### CDK (`infra/`)

- **`countries.py`:** `messaging` (`capture`/`send`), `qa_test_recipients`, `sms_sender_id`, `jobs_every_minutes` (5; **15 on `uk-qa`**, answers 7), and `sends`. **Refused at synth:** `capture` on Training or Production; `send` on QA (real sending on QA must be named: `qa_test_recipients=True`, which sends to the test recipients only); `qa_test_recipients` outside QA. **`uk-qa` has `qa_test_recipients=True`.**
- **`MessagingStack`** (`poised-<env>-messaging`, only when `sends`): SES email identity (the address from SSM `<prefix>/email-from`, a deploy-time parameter), SES configuration set + event destination (bounce, reject, renderingFailure → SNS); SMS protect configuration (**the entry's phone country ALLOW, the other 244 BLOCK**; `sms_countries.py`, fetched from AWS 14 Oct), SMS configuration set (with it, and an event destination for the ten failure types → SNS); a sender ID when the entry names one; the topic `poised-<env>-deliveries` with a policy for `ses.amazonaws.com` and `sms-voice.amazonaws.com` (this account only).
- **`ApiStack`:** the **jobs Lambda** (`poised-<env>-jobs`, 3-minute timeout); the **EventBridge Scheduler** schedule `poised-<env>-jobs`, `rate(<jobs_every_minutes> minutes)`, no retries; the jobs Lambda **subscribed to the delivery topic**; the sending settings and rights (`ses:SendEmail` on identities + the configuration set, `sms-voice:SendTextMessage`) only when the entry sends; **alarms** to `poised-<env>-alarms` (email to `/sj/alert-email`): `DeliveryFailures` (a metric filter on `DELIVERY_FAILED` in all three functions' logs) and the jobs Lambda's `Errors`.
- **Scripts:** `make aws-setup … EMAIL_FROM=…` stores `<prefix>/email-from` for an entry that sends. **`make test-recipients`** (`scripts/test-recipients.sh`, QA only): stores the list in the SecureString `/sj/qa/messaging-test-recipients` and starts the sandbox verifications (SES identity per address, unless it already is one; SMS verified destination + code); `-verify CODE=…`, `-status`, `-off`. Never prints a contact.
- **`infra/README.md`** updated (stacks, manual steps 3–4, aws-setup, "Texts and emails on QA", tidy-up lists).

## Decided by the builder (automatic mode)

1. **QA keeps capturing, and really sends only to named test recipients** (`messaging.TestRecipients`), rather than switching QA to `send` for a test run. **Why:** switching would break the journeys and testers (they read the captured messages, and the made-up mobiles would fail in the sandbox), and this guarantees nobody but Rahul receives anything from QA, even after the account leaves the sandbox. The brief's "refuses real sending on QA unless named in the country entry" is `qa_test_recipients=True`; plain `send` is refused on QA.
2. **SES's mailbox simulator is always an allowed recipient on QA,** so "a message to an invalid address shows as undelivered" can be tested for real (SES really bounces it, the event comes back through SNS) without anyone receiving it. **An SMS can't be failed for real in the UK sandbox** (AWS's SMS simulator numbers are US, which PAT-11 refuses), so the SMS failure path is covered by the tests (Go unit + integration, the Lambda handler's integration test).
3. **Undelivered = couldn't be sent, or AWS reported it undelivered,** listed since the latest invite or Resend; each also gets an Automatic General note (as S16's reminders do), and the alarm.
4. **One delivery topic for both services,** handled by the jobs Lambda (no fifth function).
5. **The alarm counts log lines** (a metric filter on `DELIVERY_FAILED`), so a refused send (in any function) and a reported failure both count, with no metrics code in Go. A second alarm on **jobs Lambda errors** (OPS-05 names failed jobs too; cheap).
6. **SNS email subscription for the alarms** (the budget's address): AWS asks to confirm it once.
7. **The protect configuration lists every country** (245, from AWS): AWS's default is ALLOW, so a country left out would be allowed. Fetched by creating and deleting a temporary protect configuration (`poised-temp-country-list`) on 14 Oct.
8. **No SMS sender ID for now** (none registered, answers item 3): AWS picks the origination; whether UK texts go without one in the sandbox is checked at the real test (questions.md Q3).
9. **The test `test_a_second_country_has_nothing_of_the_uk`** still checks ca-qa's stacks; ca-qa (capture only) has no MessagingStack. A sending second country's protect configuration names GB as `BLOCK`, which is correct.

## Tests

- **Go unit:** `messaging` (SES/SMS request building, configuration sets, sender ID; a refused send logged with the marker and no contact; test recipients), `delivery` (every event type → reason; notes; **infra's SMS event list and alarm marker match Go's**, read from the `.py` files), `config` (sending, test recipients, refusals, SSM names, a missing test-recipients parameter).
- **Go integration:** `httpapi/undelivered_integration_test.go` (messages recorded with IDs; a reported failure shows on the episode and in the notes once; other hospital/trust 404; a non-episode message changes nothing; a failed Resend shows its own failures; a good Resend clears; an undelivered reminder; capture IDs), `cmd/jobs` (the schedule's call runs the jobs with the QA clock; an SNS call records the failure; no QA clock outside capture).
- **Vitest:** `undelivered.test.ts`.
- **CDK:** `tests/test_messaging.py` (countries refusals; the jobs function and schedule; capture-only has no sending; MessagingStack's identity, configuration sets, protect configuration, topic policy, sender ID; the subscription; the functions' settings and rights; the alarms); existing tests updated for three functions.
- **Final results (14 Oct, after the deploy and the sign-in fix):**
  - `make lint` ok.
  - `make test`: Go ok (17 packages); Vitest 163 + 116 passed; CDK 123 passed.
  - `make test-integration`: all ok.
  - QA redeployed with this tree (`make deploy-stacks`).
  - `make e2e E2E_BASE_URL=<QA>`: **36 passed, 13 skipped** (the dev-login-only journeys).
- **First results (14 Oct, before the deploy):** `make lint` ok; `make test`: Go ok, Vitest 161 + 116 passed, CDK 122 passed; `make test-integration`: all ok; `make synth` ok (four stacks in uk-qa); `make e2e` against `make dev`: 43 passed, 5 skipped, 1 failed (J7's `to_have_count(1)`: the known reused-mobile flake, pending.md; J7 passed 6/6 on a re-run after `make seed`).

## Suggested commits (Rahul commits after verification)

1. **S17: episode_messages, senders return message IDs, undelivered messages on the episode** (migration 00018, queries, `messaging` IDs + `TestRecipients`, `invite` recording, `delivery`, `GET …/episodes/{eid}` `undelivered`, the episode page; tests: messaging, delivery, undelivered integration, Vitest).
2. **S17: SES and End User Messaging SMS senders, MESSAGING=send and QA's test recipients; a failed sign-in send answers 202** (`messaging/aws.go`, `config` sending and `JOBS_SCHEDULE_MINUTES`, `serve.Setup`, `requestSignin`, `/qa/jobs`'s schedule text; tests).
3. **S17: the jobs Lambda** (`cmd/jobs`, `make build-lambdas`; its integration test).
4. **S17: CDK MessagingStack, the jobs schedule, delivery events and alarms; test recipients' scripts** (`countries.py`, `messaging_stack.py`, `sms_countries.py`, `api_stack.py`, `stage.py`, scripts, Makefile, README; CDK tests).
5. **Docs: S17** (build notes, test plan, decisions, plan status).
