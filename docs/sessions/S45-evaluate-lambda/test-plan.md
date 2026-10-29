# S45 test plan: the evaluate Lambda, on QA

**Everything is on QA:** https://main.d33kh0gi8vk83v.amplifyapp.com (no `make dev` needed). S45's working tree was deployed there by hand on 29 Oct (`make deploy-qa`), so QA runs this tree, not `master`. What changed: the automatic checks on a sent HQ (and on Publish and Complete review) now run in their own AWS function instead of a local program. **To the user nothing should look different**: auto-triage, tasks and flags work as they do locally.

## Setup

1. **Nothing to start.** Check QA answers: `curl -s https://main.d33kh0gi8vk83v.amplifyapp.com/api/health` → `{"status":"ok"}`.
2. **Signing in** is Cognito's, as in S12 ([S12's test plan](../S12-cognito-signin/test-plan.md), setup steps 2–5): open `/login`, **Sign in**, type the email, paste the password, then the authenticator code. From the repo root:
   - the password: `make qa-password PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'` (copied to the clipboard, never printed; paste with **Cmd+V**, then `pbcopy < /dev/null`);
   - the code: `make qa-code PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'`. Each code works once; if it's refused, wait for the next one.
   - If a `make` command can't use the AWS profile, stop and report **BLOCKED: AWS sign-in expired**.
   - **Never write a password, secret or code into the report.**
3. **Who:** Cara Clinician (`cara.clinician@hospital-a.example`, Hospital A1) and Alex Author (`alex.author@hospital-a.example`, all of Trust A). The patient's invite link is on the captured-messages page `/qa/messages`, which needs **Sam Superadmin** (`sam.superadmin@poised.example`); open it in a second browser profile or window so Cara stays signed in.
4. **The HQ:** the journeys published several HQs named **`E2E J11b <code>`** in Trust A, with the default rules and auto-triage on. Use the newest (Alex Author → Questionnaires). It has two Question Sets: **About you** (Do you smoke? Do you get chest pain when you walk? How often do you snore?) and **Your heart** (Have you been told you have heart failure?).
5. **Test data:** made-up patients only, named `MT S45 <time> <surname>`, email `mt-s45-<time>@example.com`, mobile `07700 900` + 3 digits. Date of birth: any adult's.
6. **Patient steps at phone size (390×844).**

**To create an episode** (used below): as Cara, Hospital A1 → Episodes → **New episode**: First name, Surname, Date of birth, Sex, Mobile, Email; **Health questionnaire**: the `E2E J11b` HQ from step 4; **Create**. Then as Sam, `/qa/messages`: open the newest invite to that email and copy its link into the phone-size window; type the date of birth to get in.

## Test cases

### T-01 A low-risk patient is auto-triaged (brief 1)
1. Create an episode (above), `MT S45 <time> Low`.
2. At phone size, as the patient: **About you**: smoke **Yes**, chest pain **No**, snore **Never**; **Finish this section**. **Your heart**: heart failure **No**; **Finish this section**. **Send my answers**, confirm.
3. **Expected:** "Thank you, <first name>"; nothing about rules, ASA or triage on the patient's screen.
4. As Cara, Episodes → the **Auto-triaged** worklist.
5. **Expected:** the episode is there at **Ready for admission**. Open it: the card names the rule **Low risk**; the history says the rule moved it; the General notes do **not** say "The automatic checks didn't run."
6. Click **Checked**. **Expected:** who and when are shown; it leaves the Auto-triaged list.

### T-02 A high-risk patient gets the task and the flag (brief 1)
1. Create an episode, `MT S45 <time> High`.
2. As the patient: About you: smoke **No**, chest pain **No**, snore **Never**; Your heart: heart failure **Yes**. Send.
3. As Cara: **Expected:** the episode is on **Anaesthetic review needed**, at **Ready for review** (not auto-triaged), with the red **Anaesthetic review** flag on its row.
4. Open it. **Expected:** the flag says "Added by the rule “High risk” on …"; the task **Book anaesthetic review** for the Anaesthetist team, "Added by rule: High risk"; the line saying Ready for admission needs the anaesthetist's ASA grade first. General notes list the task, the flag and the requirement; **no** "The automatic checks didn't run."

### T-03 Complete review and Publish still work (the other two calls)
1. As Cara, on T-02's episode, validate each Question Set, then **Complete review**. **Expected:** it completes (no "Validate every Question Set first", no error).
2. As Alex Author, open the newest `E2E J11b` HQ, **New version**, change one question's wording in a Question Set, **Publish**. **Expected:** the Publish window opens, says what the version auto-triages, and publishing works. (Then leave it; the journeys make new HQs each run.)

### T-04 The function's logs hold no answers (brief 3)
1. In a terminal (the log group's name is CDK's own, so look it up first):
   `G=$(aws logs describe-log-groups --profile AdministratorAccess-026090536258 --region eu-west-2 --log-group-name-prefix poised-uk-qa-api-evaluatelogs --query 'logGroups[0].logGroupName' --output text)`, then
   `aws logs tail "$G" --since 1h --profile AdministratorAccess-026090536258 --region eu-west-2`.
2. **Expected:** only lines like `evaluate: evaluate form <id>: 200 in 160 ms`, `publish-problems: 200 in 1 ms`, `shown form <id>: 200 in 0 ms`, and Lambda's START / END / REPORT lines. **No** patient name, date of birth, email, mobile, answer text ("Yes", "heart failure") or question wording.

## Hand checks for Rahul (they change QA's AWS setup or need his inbox)

### H-1 A broken function: the note, and the alarm email (brief 2)
Run by the builder (Claude) with Rahul's OK, not by the tester:
1. **Break it:** `aws lambda put-function-concurrency --function-name poised-uk-qa-evaluate --reserved-concurrent-executions 0 --profile AdministratorAccess-026090536258 --region eu-west-2` (the function then refuses every call; nothing is deleted).
2. Create an episode and send T-01's low-risk answers.
3. **Expected:** the send works for the patient ("Thank you"); the episode stays at **Ready for review** (not auto-triaged), and its General notes say **"The automatic checks didn't run. Review this HQ by hand."** A Publish (T-03 step 2) is refused with "The checks couldn't run. Try again in a minute." and publishes nothing.
4. **Expected within about 5 minutes:** an email from AWS Notifications to the alert address, "ALARM: poised-uk-qa-evaluate-failures".
5. **Mend it:** `aws lambda delete-function-concurrency --function-name poised-uk-qa-evaluate --profile AdministratorAccess-026090536258 --region eu-west-2`. Send another low-risk HQ: it's auto-triaged again (T-01). The alarm goes back to OK (a second email, "OK: …").

### H-2 A green pipeline after the push (brief 4)
After Rahul commits and pushes: CodePipeline → `poised` runs green, journeys included, and the function's health check names the commit (`aws lambda invoke --function-name poised-uk-qa-evaluate --cli-binary-format raw-in-base64-out --payload '{"op":"health"}' out.json --profile … --region eu-west-2` → `"evaluator":"clinical@<commit>"`, without `-dirty`).

## Regression

- **R-01** S42: J11's low-risk path is T-01; the Auto-triaged worklist count goes down after Checked.
- **R-02** S43: clear T-02's flag with a reason (as Cara): the badge goes; who, when and why stay.
- **R-03** S44b: on T-02's POA Summary, add the anaesthetist's ASA III; then move the episode to **Ready for admission**: allowed.
- **R-04** S44c: as a new patient, About you: smoke Yes, then No: nothing else changes, and the send is still auto-triaged.

## Out of scope

- Anything about speed you notice by feel (the numbers are in the build notes).
- The training and production stages (C10); the rest of the app on QA beyond the cases above.
- J7's occasional failure locally (a random mobile seen before; pending.md).
