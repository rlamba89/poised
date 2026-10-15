# S17 test plan: jobs and messaging on AWS, on QA

**Everything is on QA:** https://main.d33kh0gi8vk83v.amplifyapp.com (no `make dev` needed). S17's code was deployed there by hand on 14 Oct (`make deploy-qa`), so QA runs this working tree, not `master`. **Don't push.**

**What S17 adds, in a tester's words:**
- **The scheduled jobs run by themselves on QA** (reminders, "not responding"), every **15 minutes** (QA's setting; 5 on a real stage). `/qa/jobs` shows the last run.
- **QA still captures every text and email** (`/qa/messages`), as before. **Only messages to Rahul's own email and mobile** (the test recipients, kept in AWS) are also really sent: emails from Rahul's verified address through SES, and texts from the UK sender ID **POISED**. Nobody else can receive anything.
- **Undelivered messages:** a text or email that couldn't be sent, or that AWS reports as undelivered (a bounced email), shows on the episode as **"Invite not delivered"**, with an Automatic General notes entry, and an alarm email goes to Rahul.

## Setup

1. **Nothing to start.** Check QA answers: `curl -s https://main.d33kh0gi8vk83v.amplifyapp.com/api/health` → `{"status":"ok"}`.
2. **Sign in as on S12** (Cognito + authenticator). From the repo root, in a terminal:
   - **The password:** `make qa-password PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'` copies it to the clipboard (never printed). Paste with **Cmd+V**, then clear the clipboard: `pbcopy < /dev/null`.
   - **The code:** `make qa-code PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'` prints the current 6-digit code. Each code works once; if it's refused as used, wait for the next.
   - If a `make` command can't use the AWS profile, stop and report **BLOCKED: AWS sign-in expired**.
   - **Never write a password, secret or code into the report.**
3. **Who's who:** **Cara Clinician** (`cara.clinician@hospital-a.example`, clinician at Hospital A1) creates episodes; **Val Clinician** (`val.clinician@trust-a.example`, Hospital A2 only); **Bea Author** (`bea.author@hospital-b.example`, Trust B only); **Sam Superadmin** (`sam.superadmin@poised.example`) for the testers' tools `/qa/messages` and `/qa/jobs`.
4. **How to sign in:** `/login` → **Sign in** → Cognito's page: email, paste the password, **Sign in** → the code → **Sign in**. **Sign out** (top right) to switch user.
5. **Test data:** made-up patients named `MT S17 <time> <Surname>`, made-up mobiles in the drama range `07700 900xxx` (pick 3 random digits), and made-up emails `@example.com`. **Never type a real person's mobile or email.** The only special address is **`bounce@simulator.amazonses.com`** (AWS's own test address: it always bounces and never reaches a person), used in T-04.
6. **Leave the builder's real-send episodes alone:** patients **Sam Real…**, **Sam Text…** and **Sam Bounce…** in Hospital A1 (HQs named "S17 real send …" / "S17 real text …"). The first two have Rahul's real contacts: **never Resend their invites**, or Rahul gets a real message. Their reminders are already switched off.
7. **The HQ:** create episodes on Hospital A1's published HQ (any; e.g. "Workflow" or the first in the list).
8. **Read-only AWS checks** (T-08) use the AWS CLI with `--profile AdministratorAccess-026090536258 --region eu-west-2`. Don't change anything in AWS.

## Test cases

### T-01 A new episode's invite is captured as before (nothing really sent)
1. Sign in as **Cara**. **New episode**: a made-up patient with a mobile `07700 900xxx` and an email `mt-s17-<time>@example.com`, the HQ, **Create**.
2. **Expected:** the episode page; General notes "Invite sent by text message and email."; **no** "Invite not delivered" badge.
3. Sign out, sign in as **Sam**, open `/qa/messages`, search the mobile, then the email. **Expected:** one text and one email, each with an `/i/…` link (captured as before).

### T-02 The schedule runs the jobs by itself
1. Signed in as **Sam**, open `/qa/jobs`. **Expected:** "The timer" says "On: a schedule on AWS runs the jobs every 15 minutes, as of real time." Note **the last run** ("Last run <time>, as of <time>.").
2. Wait up to **16 minutes** (the schedule is every 15) and reload.
3. **Expected:** the last run time has moved on by itself (no one pressed **Run jobs**), and keeps moving on each 15 minutes. Note the times in the report.
4. **Expected:** nothing else changed on the page (the clock offset is as you found it).

### T-03 A reminder from a scheduled run appears in the captured messages
1. Still as **Sam** on `/qa/jobs`, under **The timer**, set **Days ahead** 2 and **Hours ahead** 1, then **Move the clock**. Note the time.
2. Wait for the next scheduled run (up to 15 minutes; **don't press Run jobs**), then reload `/qa/jobs`: the last run's "as of" is about 2 days ahead.
3. Open `/qa/messages` and search T-01's mobile. **Expected:** a second text, the **reminder** (the hospital's reminder wording with a new `/i/…` link), as well as T-01's invite.
4. Sign in as **Cara**, open T-01's episode. **Expected:** General notes has an **Automatic** "Day 2 reminder sent by text message and email."
5. **Click Back to real time** on `/qa/jobs` (Sam), even if the case failed.

### T-04 An email that bounces shows as undelivered (a real SES bounce)
1. As **Cara**, **New episode**: a made-up patient, **no mobile**, email **`bounce@simulator.amazonses.com`**, **Create**.
2. **Expected at once:** General notes "Invite sent by email."
3. Wait about a minute, reload the episode. **Expected:**
   - beside the status, a red **"Invite not delivered"** badge;
   - in **Patient's invite**, a red **"Not delivered"** box: "The invite's email wasn't delivered: the email address doesn't exist or doesn't accept email. (Sent <time>.)" and "Check the patient's mobile number and email address, then resend the invite.";
   - General notes: an **Automatic** entry "The invite's email wasn't delivered: the email address doesn't exist or doesn't accept email. Check the patient's email address, then resend the invite."
   - (If nothing shows after 5 minutes, reload once more, then report it as a failure.)
4. Reload the page and open it from the Episodes list. **Expected:** the same.
5. No console errors.

### T-05 Resend starts afresh
1. On T-04's episode, change nothing and click **Resend invite**.
2. **Expected at once:** the "Not delivered" box and badge are gone (a new round), and General notes "Invite sent again by email." (or the app's Resend wording).
3. After about a minute, reload. **Expected:** the box and badge are back, for the **new** email only (one line, "Sent" the Resend's time), and a second Automatic "wasn't delivered" note.

### T-06 Undelivered is shown only to the episode's own hospital
1. Copy T-04's episode URL. Sign out; sign in as **Val** (Hospital A2 only); open the URL.
2. **Expected:** Val can't see it (not found / no access), as for any A1 episode.
3. Sign in as **Bea** (Trust B); open the URL. **Expected:** the same refusal.

### T-07 A patient's own view never shows any of this
1. As **Sam**, open `/qa/messages`, search T-01's email, open its `/i/…` link at **phone size (390×844)**, **Continue**, type the patient's date of birth.
2. **Expected:** the patient's questionnaire as before; nothing about delivery or other messages.

### T-08 AWS's side (read-only checks)
Run each and compare. **Don't paste any email address or phone number into the report**; say "Rahul's" where one appears.
1. `aws scheduler get-schedule --name poised-uk-qa-jobs --profile AdministratorAccess-026090536258 --region eu-west-2 --query '[State,ScheduleExpression,Target.RetryPolicy.MaximumRetryAttempts]'` → `["ENABLED","rate(15 minutes)",0]`.
2. `aws pinpoint-sms-voice-v2 describe-configuration-sets --configuration-set-names poised-uk-qa --profile … --region eu-west-2 --query 'ConfigurationSets[0].[ProtectConfigurationId,EventDestinations[0].SnsDestination.TopicArn]'` → a `protect-…` ID and the `poised-uk-qa-deliveries` topic.
3. With that ID: `aws pinpoint-sms-voice-v2 get-protect-configuration-country-rule-set --protect-configuration-id <id> --number-capability SMS --profile … --region eu-west-2 --query 'CountryRuleSet.GB.ProtectStatus'` → `"ALLOW"`; the same with `CountryRuleSet.US.ProtectStatus` and `CountryRuleSet.FR.ProtectStatus` → `"BLOCK"`.
4. `aws cloudwatch describe-alarms --alarm-name-prefix poised-uk-qa --profile … --region eu-west-2 --query 'MetricAlarms[].[AlarmName,StateValue]'` → `poised-uk-qa-delivery-failures` and `poised-uk-qa-jobs-failed`. After T-04, `delivery-failures` is (or recently was) `ALARM`.

### T-09 Sign-in links and other messages still work (capture)
1. Sign out of staff. Open `/signin` (the patient's "sign in to see your questionnaires"), at phone size, type T-01's made-up email, send.
2. As **Sam**, `/qa/messages` → search it. **Expected:** a captured sign-in email with an `/s/…` link, as before.

## Hand checks for Rahul (real messages; the tester can't receive them)

- **H-1 A real text and email.** *Done by the builder on 14 Oct:* a real invite email reached Rahul's inbox, and a real invite text from **POISED** reached his phone ("got it"). The text's row has AWS's message ID and no failure. **Still for Rahul:** on the phone, open the text's link → **Continue** → the date of birth (the episode's is 17 May 1980) → the questionnaire.
- **H-2 The alarm email.** After T-04 (or H-1's), Rahul's alert address gets "ALARM: poised-uk-qa-delivery-failures" (only once he has confirmed SNS's subscription email).
- **H-3 Nobody else.** T-01's made-up contacts got nothing real (they're only captured), by design: only the test recipients are allowed.

## Regression (earlier sessions)

- **R-01 (S12)** Sign in as Cara with the password and code; Sign out returns to `/login`.
- **R-02 (S14)** T-01's invite link: **Continue**, a wrong date of birth shows the error; the right one opens the questionnaire (phone size).
- **R-03 (S16)** `/qa/jobs` → **Now + 7 days** → **Run jobs** marks T-01's episode **Not responding** (red badge) with an Automatic note.
- **R-04 (S13)** The New episode page's search finds T-01's patient by surname with the date of birth.
- **R-05 (S11)** The audit log (Ada Admin, Trust A) lists Cara's views of T-01's episode.

## Out of scope (don't report)

- Real texts or emails to anyone but Rahul (by design, QA only captures them).
- **A real undelivered text:** the UK SMS sandbox can't fail one on purpose (AWS's test numbers are American, which Poised refuses; unverified numbers are never really sent from QA). It's covered by the automated tests.
- Emails landing in spam (no domain yet), the sender ID's registration, SES/SMS production access, the $1 monthly SMS spend limit.
- The pipeline (not pushed until S17 is committed).
- The known journey flake with reused made-up mobiles (pending.md).
