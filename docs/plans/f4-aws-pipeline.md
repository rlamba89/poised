# F4: AWS and the pipeline (the QA environment)

**Goal:** a push to `main` runs every test, deploys to a **QA** environment on AWS, then runs the end-to-end journeys against QA. It all costs close to nothing when idle (A-11).

The architecture is in [saas-requirements.md §13](../saas-requirements.md). Training and Production stages come in C10.

**Depends on:** F1–F3.

**Not in F4:** Cognito (C1), SES/SMS (C2), the files bucket (C8), RDS (C10), and any country other than the UK (A-20). F4 still builds the country groundwork (Step 1).

## Step 0: one-off manual setup (Rahul, about 30 minutes)

These can't or shouldn't be done by CDK:

1. **An AWS account for QA**, using the UK's region, `eu-west-2`. Turn on MFA for the root user, and create an admin user/role for deploying.
2. **Bootstrap CDK:** `cdk bootstrap aws://<account>/eu-west-2`.
3. **Connect GitHub:** create an AWS CodeConnections connection to the GitHub repo and approve it in the console. Note its ARN.
4. **Neon:** a free account, a project in **AWS London (`aws-eu-west-2`)**, and a database `sj`. Copy the **direct** (not pooled) connection string.
5. **Store the secrets in SSM** (CloudFormation can't create SecureStrings):
   - `aws ssm put-parameter --name /sj/qa/database-url --type SecureString --value '<neon url>'`
   - and `/sj/qa/token-secret` (a random 32 bytes).
6. **Budget alarm:** an AWS Budget of £5/month with an email alert. The first two budgets are free.

## Step 1: CDK app (Python), in `infra/`

- **Files:** `infra/app.py`, `requirements.txt` (pinned `aws-cdk-lib`, `constructs`), `cdk.json`, and `infra/README.md`.
- **`infra/countries.py` (A-20):** the only place a region, account, web address or country default is written.
  - It holds one entry per country and environment, a small dataclass with the region, account, `app_url`, time zone, locale and phone country.
  - It starts with one entry, `uk-qa`.
  - Every stack takes an entry as input and passes its values to the Lambdas as environment variables. Nothing in a stack names `eu-west-2` or a domain.
- **CDK test (pytest + `aws_cdk.assertions`), written first:** synthesise the stages for a made-up second country (e.g. `ca-qa` in `ca-central-1`) and fail if any template contains `eu-west-2` or the UK address. Run it in the synth step.
- **`ApiStack`** (per stage):
  - Two Lambda functions, `staff-api` and `patient-api`: `provided.al2023`, **arm64**, 512 MB, code from `apps/api/dist/<fn>`.
  - Environment: `DATABASE_URL_PARAM`, `TOKEN_SECRET_PARAM`, and `DEV_LOGIN=true` **in QA only**. A CDK check fails the synth if `DEV_LOGIN` is set on any other stage.
  - An IAM role per function that can read only its own SSM parameters.
  - Log retention of 14 days.
  - **API Gateway HTTP API:** `ANY /api/p/{proxy+}` → patient-api; `ANY /api/{proxy+}` → staff-api.
  - Default throttling. Reserved concurrency for patient-api only if the account's concurrency limit allows it; new accounts have a low limit, so note it in the README.
- **`WebStack`** (per stage):
  - An Amplify app (`aws_amplify.CfnApp`) with **no repository connected** (manual deployments), and one branch, `main`.
  - **Custom rules, in this order:**
    1. `/api/<*>` → `https://<http-api-domain>/api/<*>`, **200** (reverse proxy);
    2. the SPA rule → `/index.html`, **200** (the regex form from the Amplify docs).
- **`QaStage`** = `ApiStack` + `WebStack`.

**First check, before going further:** confirm that Amplify's reverse proxy passes the `Cookie` request header and the `Set-Cookie` response header. Sign in with the dev login on QA and call `/api/me`.
- If it works, carry on.
- If it doesn't: put a custom domain on both (e.g. `qa.<domain>` for Amplify and `api.qa.<domain>` for the API), and use cookies scoped to the parent domain. Cross-site cookies are blocked by Safari, so the two must be same-site. Record which way it went under Status.

## Step 2: the pipeline (CDK Pipelines)

- **`PipelineStack`:** `pipelines.CodePipeline`, which updates itself. Source: `CodePipelineSource.connection(<repo>, "main", connection_arn=…)`.
- **Synth step (CodeBuild, arm64, privileged mode so it can run Docker):**
  1. Install Go, Node and Python at the repo's pinned versions.
  2. `make lint test`.
  3. Start Postgres (`docker run -d -p 5432:5432 postgres:<same major as compose>`), then `make test-integration`.
  4. `make build-lambdas` and `npm run build -w apps/web`.
  5. `cdk synth`.
- **QA stage, with post-deploy steps:**
  1. **Migrate:** `goose up` against the QA database (the URL comes from SSM).
  2. **Seed:** `go run ./cmd/seed`. **Make the seed idempotent** first (safe to run twice), with an integration-style test that runs it twice.
  3. **Deploy the web app:** zip `apps/web/dist`, then `aws amplify create-deployment` → upload the zip to the returned URL → `aws amplify start-deployment`, and wait for it to succeed.
  4. **End-to-end:** `make e2e-setup e2e` with `E2E_BASE_URL=https://main.<amplify-app-id>.amplifyapp.com`. Keep the Playwright traces as artifacts.
- **Failing rules:**
  - Tests failing in the synth step stop everything.
  - The end-to-end step failing marks the run failed.
  - Later stages (C10) can't be promoted from a failed run.

## Step 3: the "add a country" runbook

- Write it in `infra/README.md`: the country entry, then the one-off steps in the new region (CDK bootstrap, SSM secrets, DNS; later SES, SMS and the superadmin), then a push. The full list is in [§13 Countries](../saas-requirements.md).
- ***Proposed* proof, once:** follow the runbook to deploy a throwaway QA copy in a second region that Neon also offers (e.g. `us-east-1`), run J1–J5 against it, then destroy it. Record how long it took under Status. That's the evidence that a new country is quick.

## Step 4: idle cost

- After a week of normal use, read Cost Explorer and record the monthly figure under Status.
- **Target:** under £1/month for QA, excluding pipeline build minutes.
- Expected costs:
  - Lambda, API Gateway and Amplify hosting are pay-per-use, so close to £0 when idle;
  - SSM standard parameters are free;
  - Neon is on its free tier;
  - CloudWatch logs are small, with 14-day retention;
  - CodePipeline V2 and CodeBuild are billed per minute while running.

## Checks

- A push to `main` runs the whole pipeline to green, with no manual step.
- The QA URL works: the dev login, J1–J5 via the pipeline, and a reloaded deep link.
- A deliberately failing unit test stops the pipeline before deploy. Try it on a branch pipeline, or revert at once.
- The idle cost is recorded.
- The CDK test for a made-up second country passes, and the runbook exists.

## Status

Not started.
