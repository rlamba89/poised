# S06: questions for Rahul (14 Oct 2026)

**Answered 14 Oct:** Rahul ran `make aws-setup` (see answers.md); QA is deployed.

## 1. The Neon URL and the budget alarm's email address aren't in AWS yet

Everything that can be built and tested without them is done (build-notes.md), and Poised's own CDK bootstrap (`poised-cdk-toolkit`) is now in account 026090536258, eu-west-2. The first deploy needs two values in SSM:
- `/sj/qa/database-url`: the Neon database's connection string. The Lambdas read it at start-up. It's a secret, so you put it there yourself; I never see it.
- `/sj/alert-email`: where the £5 budget alarm emails go. CloudFormation reads it when the budget is created, so it's not written in the repo.

**What to do (about 10 minutes):**
1. **Neon** (if not done): sign up at neon.tech (free plan), create a project in region **AWS London (`aws-eu-west-2`)** with a database named **`sj`**. On the project's **Connect** dialog, turn **Connection pooling off** and copy the connection string (the host must not contain `-pooler`). It looks like `postgresql://<user>:<password>@ep-….eu-west-2.aws.neon.tech/sj?sslmode=require…`.
2. In **your own terminal**, in the repo:
   ```sh
   aws sso login --profile AdministratorAccess-026090536258   # if your SSO session has expired
   make aws-setup PROFILE=AdministratorAccess-026090536258 NEON_DATABASE_URL='<Neon direct URL>' ALERT_EMAIL='<your email address>'
   ```
   Keep the single quotes. `PROFILE=` picks the AWS profile and wins over the `AWS_PROFILE=sid-dev` in your `.env` (added 14 Oct, after your first try failed on `sid-dev`). The script first prints "AWS profile: AdministratorAccess-026090536258" and "AWS account: 026090536258", and stops before changing anything if a value is missing.
   It should end with "Done". Step 4 will say the cost tag isn't known to Billing yet: that's expected on the first run.
3. Tell the conductor "Answers are in answers.md" (or add a line to answers.md saying it's done).

**Options:**
- **A (recommended): do the above.** I then deploy QA (`make deploy-qa`), migrate and seed Neon, upload the web app, check cookies through Amplify's proxy, measure the proxies in `X-Forwarded-For`, and write the test plan for the QA URL.
- **B: skip Neon for now.** Not possible for S06: the QA URL can't serve the app without a database.
- **C: use a different alert address mechanism** (e.g. no budget email, only the console). Not recommended: the budget alarm was your requirement, and the address is the only way it can reach you.
