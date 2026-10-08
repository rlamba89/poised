# S06: AWS: the CDK app and the QA environment

**Plan:** F4 ([f4-aws-pipeline.md](../../plans/f4-aws-pipeline.md)), Step 1 and its first check, plus A-20 (`infra/countries.py`).
**Depends on:** S05 verified, and **Rahul has done F4 Step 0**:
- the QA account;
- CDK bootstrap in `eu-west-2`;
- the CodeConnections ARN;
- Neon;
- the SSM secrets;
- the budget alarm.

**Ask Rahul first:**
- Is F4 Step 0 done? Get the account ID and the connection ARN, but **never secrets**.
- Do you approve the expected costs? Close to £0 when idle; see F4 Step 4.

## Build

- **`infra/`,** a CDK app in Python:
  - **`countries.py`** with one entry, `uk-qa`;
  - **`ApiStack`** and **`WebStack`**, each taking a country entry as input, so nothing in them names a region or domain;
  - the Amplify rewrite rules, in order: `/api/<*>` first, then the SPA rule.
- **The CDK test, written first,** with pytest and `aws_cdk.assertions`: synthesise a made-up `ca-qa` in `ca-central-1`, and fail if any template contains `eu-west-2` or the UK address.
- **Deploy QA by hand once:**
  1. `cdk deploy`;
  2. migrate and seed the Neon database;
  3. deploy the web zip with `create-deployment` / `start-deployment`.

  Write each command into `infra/README.md`.
- **F4's first check:** do cookies pass through Amplify's proxy? Record which way it went in F4 Status.

## Not in this session

- the pipeline (S07);
- Cognito (S12);
- SES and SMS (S16);
- RDS (C10).

## Done when

- The QA URL serves the app.
- The dev login works.
- A reloaded deep link works.
- The CDK test passes.

## Manual test focus

On the **QA URL**, not localhost:
- J1–J5 by hand, with the patient link at 390×844;
- reload deep links;
- `/api/health`;
- sign in and out (cookies);
- console and network errors.
