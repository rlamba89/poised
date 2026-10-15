# S17 verification

## Verification 1 (15 Oct 2026, verifier, automatic run)

**Verdict: Verified.** Plan C2 is done.

What I checked:
- the whole working tree against the brief, plan C2 and answers 1–10;
- `/code-review high`, with each finding weighed below;
- every suite;
- read-only checks on QA's AWS account;
- three spot checks in the browser, run locally.

No finding blocks the session. They're all minor, and they're logged in pending.md (§3, "From S17").

### Scope

Everything in the brief is built:
- **`cmd/jobs`** calls the same `jobs.Runner.Tick` as S16's timer.
- **The EventBridge Scheduler schedule** runs every `jobs_every_minutes`: 15 on QA (answers 7), 5 elsewhere, with no Scheduler retries.
- **`MessagingStack`** holds:
  - the SES identity and configuration set;
  - the SMS configuration set, with a protect configuration that allows GB and blocks the other 244 codes;
  - the sender ID POISED;
  - the delivery topic.
- **CDK refuses unsafe settings at synth:** `capture` outside QA, `send` on QA, and `qa_test_recipients` outside QA.
- **The real senders** (SES v2 and End User Messaging SMS v2) sit behind the same interfaces as the fakes.
- **Delivery failures (OPS-05)** are recorded in `episode_messages`. They show as the badge and box on the episode, with an Automatic note, and they raise the alarm.

**"Real sending switched on" is narrower than the brief.** It's the builder's `TestRecipients` design: QA captures everything, and really sends only to the SSM list and SES's mailbox simulator.
- That's safer than the brief's literal reading. The reason is recorded in decisions.md.
- It's backed by a real email, a real text from POISED, and a real SES bounce on 14 Oct.

**Extra work, all small and justified:**
- the sign-in 202 fix (S15's pending item);
- `JOBS_SCHEDULE_MINUTES` for `/qa/jobs`.

### AWS checks (read-only, 15 Oct, account 026090536258, eu-west-2)

- **Schedule** `poised-uk-qa-jobs`: `ENABLED`, `rate(15 minutes)`, Scheduler `MaximumRetryAttempts` 0.
- **The jobs Lambda:** 180 s timeout. Its environment has the sending settings, `JOBS_TIMER`=false and `JOBS_SCHEDULE_MINUTES`.
- **IAM** (one role per function):
  - `ssm:GetParameter` on QA's seven named parameters only;
  - `ses:SendEmail` on `identity/*` and the `poised-uk-qa` configuration set;
  - `sms-voice:SendTextMessage` on the account's sms-voice resources (finding 7);
  - the token secret.
- **Sender ID** POISED/GB: `MonthlyLeasingPrice` 0.00, `Registered` false (as answers 9).
- **SMS spend limits:** $1 enforced for every kind of message.
- **The stacks' outputs** carry no contact: messaging's export of the topic; api's `ApiUrl`, `TokenSecretArn` and exports; web's `AppId` and `AppUrl`; auth's `UserPoolId` and `ManagedLoginUrl`.
- **Not read:** no SecureString values were read, and no message was sent.

### Privacy

I grepped every changed and new file, outside `.claude/skills`, `reports/` and `research_notes/`, for:
- Rahul's email;
- any UK mobile outside the drama range `07700 900…`;
- six-digit codes.

**Nothing was found.** The only hits are the words "Rahul's Gmail address" (pending.md) and "a Gmail address" (questions.md), neither of which gives the address. Recipients live only in `/sj/qa/messaging-test-recipients` (SecureString). The sender address lives only in `/sj/qa/email-from`, which a deploy-time SSM parameter reads; it isn't a stack output.

### Other checks

- **Tenant isolation:**
  - `ListUndeliveredMessages` is keyed by the episode, and it runs only inside `getEpisode` after the hospital-scoped load.
  - `FailEpisodeMessage` is keyed by AWS's message ID; it's a system path with no user input.
  - Integration tests cover another hospital and another trust (404).
- **A-20:** the UK grep finds only config.go's existing defaults (config is allowed). `PHONE_COUNTRY` drives the protect configuration. `sms_sender_id` is per entry.
- **Lifebox organisation name:** none found.

### Findings (all minor, worst first)

1. **The CDK and Go comments say a failed jobs run isn't retried, but Lambda retries it twice.**
   - The comments are at `infra/api_stack.py:186`, `apps/api/internal/delivery/delivery.go:118` and `delivery.go:146`.
   - EventBridge Scheduler and SNS both invoke the jobs Lambda asynchronously. The function has no EventInvokeConfig (checked on QA), so **Lambda's own async retries (2) apply**.
   - **For the schedule:** a failed Tick runs again about 1 and 3 minutes later. That's harmless, because S16's `FOR UPDATE SKIP LOCKED` lease makes runs overlap-safe.
   - **For SNS:** those Lambda retries are what actually re-run a delivery event after a database error. SNS itself doesn't retry a function error, so the comment "SNS tries again" is wrong about where the retry comes from.
   - **Fix:** correct the comments. If the behaviour should be explicit, set `retry_attempts` and `max_event_age` on the function. Keep the SNS path retrying.
2. **One bad entry stops all three Lambdas on QA** (`infra/scripts/test-recipients.sh:43-44`).
   - `make test-recipients` stores `TEST_RECIPIENTS` unchecked, then recycles the functions.
   - Go's `config.Load` refuses any entry that isn't E.164 or a bare address. So one typo (`+44 7700 …` with spaces, or `07700…`) makes all three QA Lambdas fail at start-up, until it's fixed.
   - **Fix:** validate the list in the script, with the same rules, before `put-parameter`.
3. **A lost record loses the delivery report** (`apps/api/internal/invite/invite.go:183-196`).
   - The `episode_messages` row is written after the send, and a failed insert is only logged.
   - A later bounce for that message then finds no row: it's logged as "not an episode's", and the episode never shows it.
   - This is rare: it needs a database error at that moment. **Fix:** record before sending, or return the error.
4. **A repeated delivery event raises the alarm again** (`delivery.go:128`).
   - SNS delivers at least once, and Lambda retries. A repeat logs `DELIVERY_FAILED … already noted`, which the metric filter counts, so the alarm can email again for the same message.
   - **Fix:** log the repeat without the marker.
5. **A failure event that can't be read isn't counted by the alarm** (`delivery.go:153`).
   - Examples: a Bounce with no `mail.messageId`, or a payload format that changes.
   - It's logged as `delivery: …` without the marker, so a broken pipeline goes unnoticed.
   - **Fix:** log it with the marker, or a second marker with its own alarm.
6. **The badge can mislead** (`apps/web/src/features/episodes/EpisodePage.tsx:62-65`).
   - It says "Invite not delivered" even when only a reminder failed.
   - It also shows in statuses where the invite card, with its "Not delivered" box and Resend, isn't shown (the card is only for `hq_not_complete`).
   - Cosmetic. Possible wording: "Message not delivered".
7. **The IAM grants are account-wide rather than least-privilege** (`infra/api_stack.py:124-135`).
   - `sms-voice:SendTextMessage` is on `arn:aws:sms-voice:<region>:<account>:*`, and `ses:SendEmail` is on `identity/*`.
   - That's acceptable in a Poised-only account. Before the account holds anything else, narrow them to:
     - the configuration set;
     - the sender ID;
     - the SES identity.

     The builder's simulator note, that with no origination AWS authorises against a non-account resource, should be re-tested now that POISED exists.
8. **Every send failure on `POST /api/p/signin` now answers 202** (`apps/api/internal/httpapi/accounts.go:161`), as pending.md asked. That includes a database failure while making the link, which gets no alarm because there's no `DELIVERY_FAILED` marker. It's logged. Consider marking non-AWS failures too.
9. **Nits:**
   - `apps/api/internal/serve/lambda.go:5`: the imports aren't gofmt-sorted (`"os"` sits between `"net"` and `"net/http"`).
   - `undelivered.ts` repeats Go's `delivery.Note` wording, so the API could return the sentence instead.
   - `serve.senders` loads the AWS config a second time on each cold start.
   - `TestRecipients.allowed` normalises entries that config has already normalised.
10. **A new e2e flake:** J3 (`test_j3_patient.py:74`, "In progress" after "All sections") failed once in the full run, then passed 3 of 3 alone. Nothing in S17 touches the patient's pages.

### Spot checks (local `make dev`, Chrome)

**1. The undelivered display, which test report 2 couldn't produce.**
- As Cara, I created a made-up episode (`VS17 Bouncecheck`, no mobile, `bounce@simulator.amazonses.com`) through the API.
- I then marked its captured email failed in the local database, exactly as `FailEpisodeMessage` and `AddAutomaticEpisodeEvent` do.
- **Expected and seen:**
  - the red "Invite not delivered" badge;
  - the "Not delivered" box, with "The invite's email wasn't delivered: the email address doesn't exist or doesn't accept email. (Sent 15/10/2026, 09:14.)" and the advice line.
- **Pass.**

**2. Resend starts afresh.**
- Resend cleared the badge and the box ("Invite sent again by email.").
- After I failed the new email, `undelivered` held only the Resend's row (sent 09:15:33).
- **Pass.**

**3. Another hospital's and another trust's data, with an undelivered row present.**
- Val (Hospital A2 only) got a 404 "Hospital not found." on the A1 path, and a 404 "Episode not found." through her own A2 path.
- Bea (Trust B) got a 404 on the A1 path and a 404 through her own B1 path.
- **Pass.**

### Test results

- `make lint`: ok.
- `make test`:
  - Go ok, 17 packages;
  - Vitest 163 + 116 passed;
  - CDK (pytest) 123 passed.
- `make test-integration`: all ok, including `cmd/jobs`, `delivery`, `httpapi` (undelivered) and `config`.
- `make migrate seed && make e2e` (against the conductor's `make dev`): **43 passed, 5 skipped, 1 failed.** The failure was J3's flake (finding 10). J3 then passed 3 of 3 alone.

### Hand checks for Rahul (on QA, signed in through Cognito)

These are from the build notes and the test reports.
1. **T-04 and T-05 on QA:** a `bounce@simulator.amazonses.com` episode shows the badge, the box and the note within about a minute. Resend clears them, and they come back for the new email only.
2. **T-06 with an undelivered row:** Val and Bea are refused (404). (I checked this locally.)
3. **H-1:** on the phone, the POISED text's link → **Continue** → 17 May 1980 → the questionnaire.
4. **H-2:** the "ALARM: poised-uk-qa-delivery-failures" email at the alert address (the alarm fired on 14 Oct).
5. **H-3:** nobody but Rahul received anything real (by design).
6. **`/qa/jobs` as Sam on QA** says "On: a schedule on AWS runs the jobs every 15 minutes", and its last run moves on by itself.

### Pushing after this commit

- QA was deployed by hand from this tree. The pipeline still holds S07's definition, and GitHub is 9 commits behind (S07's docs and S12).
- **With S17 committed, one push is safe:** it brings S07's docs, S12 and S17 together.
- **What the first pipeline run should show:**
  1. The self-mutate step updates the pipeline to S17's stage (MessagingStack first).
  2. Synth builds three Lambdas (`make build-lambdas`: staff-api, patient-api, jobs).
  3. The uk-qa stage deploys:
     - `poised-uk-qa-messaging`: no change, or tags only;
     - `-api`: at most new Lambda code;
     - `-web` and `-auth`.

     It should create nothing new, since every stack already exists from the hand deploy.
  4. The test gates run as in S07, plus S12's Cognito sign-in (S12's H-3).
- **Afterwards:**
  - the schedule is still `rate(15 minutes)`;
  - `make test-recipients-status` still shows 2 recipients;
  - no `DELIVERY_FAILED` lines from the journeys, which use made-up contacts and are only captured.

### Commits (made locally; not pushed)

The build notes suggested four code commits. The Go changes are interlocked: the senders' new signatures touch `accounts.go`, `invite.go` and `serve.go` together, and `delivery`'s test reads `infra/*.py`. So that no commit is left that doesn't build or pass its own tests, they're made as three:

1. **S17: the CDK MessagingStack, the jobs schedule and Lambda, delivery events and alarms; the test recipients' scripts:**
   - `countries.py`, `messaging_stack.py`, `sms_countries.py`, `api_stack.py` and `stage.py`;
   - the scripts, the Makefile (with `build-lambdas` now building jobs) and `infra/README.md`;
   - the CDK tests.

   It comes first because the Go tests check its SMS event list and alarm marker.
2. **S17: the jobs Lambda, the SES and End User Messaging SMS senders, QA's test recipients, and undelivered messages on the episode:**
   - `cmd/jobs`;
   - migration 00018 and the queries;
   - `messaging` (message IDs, `aws.go`, `TestRecipients`);
   - `delivery`;
   - `invite` recording;
   - `config` and `serve.Setup`;
   - `GET …/episodes/{eid}` `undelivered`;
   - the sign-in 202;
   - `/qa/jobs`'s schedule text;
   - the episode page;
   - every Go, integration and Vitest test.
3. **Docs: S17 verified (verification 1), plan C2 done:**
   - the session folder;
   - decisions;
   - pending;
   - the plan status;
   - both READMEs.
