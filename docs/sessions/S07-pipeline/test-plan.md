# S07 test plan: the pipeline and the "add a country" runbook

For a tester who has never seen the code. You use **QA in Chrome** and **read-only checks in the AWS console** (and GitHub, read-only).

**What you may and may not do in AWS:**
- **Look at anything named `poised…`** (pipeline, build logs, S3 results, CloudFormation stacks, the connection).
- **You may click "Release change" on the pipeline `poised` once** (T-13), which re-runs the latest commit. Nothing else that starts, stops, retries, edits or deletes anything.
- **Never push to GitHub.** Pushes are Rahul's; the four runs below were made for you to inspect.

## Setup

- **Nothing to run locally.** The app under test is QA: **https://main.d33kh0gi8vk83v.amplifyapp.com**
- **AWS console:** account **026090536258**, region **Europe (London) eu-west-2** (top-right region picker). Sign in through the SSO profile `AdministratorAccess-026090536258` (Rahul's Chrome is normally already signed in).
- **The account also holds another project's resources.** Look only at things named `poised…`.
- **Where things are:**
  - The pipeline: **CodePipeline → Pipelines → `poised`** (https://eu-west-2.console.aws.amazon.com/codesuite/codepipeline/pipelines/poised/view?region=eu-west-2).
  - Its build logs: **CodeBuild → Build history** (or from a pipeline action's **View details**).
  - The journeys' results: **S3 → the bucket whose name starts `poised-pipeline-e2eresults`**.
  - The GitHub connection: **Developer Tools → Settings → Connections → `poised-github`**.
  - The runbook: https://github.com/rlamba89/poised/blob/master/infra/README.md (or `infra/README.md` in the repo).
- **Signing in to QA:** the dev login at the QA address lists the seeded users; click one (e.g. Alex Author, Cara Clinician).

## Runs to inspect

| Run | Commit | Execution ID (starts with) | What it is | Result |
| --- | --- | --- | --- | --- |
| R1 | `0ad1243` | `8526b81f` | S07's own code: the first real run (started with Release change) | Every stage green; E2E 43 passed |
| R2 | `e9aecdb` | `71edac70` | A visible change: the dev sign-in page's text now reads "Development sign-in (local and QA only, until Cognito): choose a user." | Every stage green; E2E 43 passed |
| R3 | `358bffe` | `604e8f99` | A deliberately failing Go unit test (`TestS07PipelineStopsOnAFailingTest`) | Stopped at Build (Synth); nothing after it ran |
| R4 | `1d0d1a0` | `fc74d689` | The revert of R3 | Every stage green; E2E 43 passed |

The journeys' results are in the results bucket under `uk-qa/1/` (R1), `uk-qa/2/` (R2) and `uk-qa/3/` (R4): the folder is the E2E build's number, and R3 never reached E2E.

The pipeline's **History** lists executions with their source commit (the short commit and message) and ID; open one to see its stages.

## Test cases

### T-01: the connection and the source

1. Open **Developer Tools → Settings → Connections**.
2. Find `poised-github`.

**Expected:** status **Available**, provider GitHub. In the pipeline `poised`, the **Source** action is `rlamba89_poised`, and its details show the repo `rlamba89/poised` and branch `master`.

### T-02: a push goes green all the way through QA e2e (R2)

1. Open the pipeline `poised`, then **History** (left menu, or "View history").
2. Find run R2 (its source commit is in the table above; the commit message is shown).
3. Open it.

**Expected:**
- Every stage succeeded, in this order: **Source → Build (Synth) → UpdatePipeline (SelfMutate) → Assets (FileAssets) → uk-qa** (`poised-uk-qa-api.Prepare`, `.Deploy`, `poised-uk-qa-web.Prepare`, `.Deploy`, `Database-and-web`, `E2E`).
- No action needed anyone to click anything (no manual approval stage exists).

### T-03: the synth step runs every test before any deploy (R2)

1. In run R2, open **Build → Synth → View details → (the CodeBuild link)**, then the build log ("Build logs" tab; "View entire log" if it's cut).
2. Look for, in this order: `make lint test`, the Go test lines (`ok  github.com/rlamba89/poised/…`), Vitest's `Tests  … passed`, pytest's `… passed` (the CDK tests), `docker run … postgres:17`, `make test-integration`, more `ok …` lines (the integration tests, including `cmd/seed`), `npm run build -w apps/web`, `make synth`.

**Expected:** all present in that order, none failed, and the build **Succeeded**. The build ran on an arm64 image (Build details → Environment: `ARM_CONTAINER`, `amazonlinux-aarch64-standard:3.0`, privileged).

### T-04: migrate, seed and the web deploy (R2)

1. In run R2, open **uk-qa → Database-and-web → View details**, then its log.

**Expected:** the log shows the AWS account check (`AWS account: 026090536258 (uk-qa is in 026090536258, region eu-west-2)`), goose (`goose: no migrations to run` or `successfully migrated`), the seed (`trusts: 2, users: 8, patients: 3`), then `Deploying job N to https://main.d33kh0gi8vk83v.amplifyapp.com …` and `Deployed.` **No database address or password appears anywhere in the log.**

### T-05: the journeys run against QA (R2)

1. In run R2, open **uk-qa → E2E → View details**, then its log.

**Expected:** pytest runs against the QA address, lists the journeys (J1 to J8, patient records, trusts) with Chromium and WebKit, and ends `N passed`. At the end: "The journeys' JUnit report, and a trace of each that failed: s3://poised-pipeline-e2eresults…/uk-qa/<build number>/".

### T-06: the change is on QA (R2)

1. Open https://main.d33kh0gi8vk83v.amplifyapp.com in a new tab (reload with Shift if it was open).

**Expected:** the sign-in page says "Development sign-in (local and QA only, until Cognito): choose a user." (R2's change; before it said "Development sign-in: choose a user."). Sign in as **Alex Author**: the Questionnaires list opens. Reload a deep link (e.g. the Episodes page's address): it opens, not a 404.

### T-07: the e2e results are kept as an artifact (R2 and R3/R4)

1. Open **S3 → the `poised-pipeline-e2eresults…` bucket → `uk-qa/`**.
2. Open the folder named with R2's E2E build number (from T-05).

**Expected:** `junit.xml` is there (a passing run keeps no traces; that's by design: traces are kept only for failed journeys). Its **Management → Lifecycle rules** shows objects expire after **30 days**.
*If a run's journeys ever failed, its folder also has a folder per failed journey with `trace-N.zip` files.*

### T-08: a failing test stops the run before deploy (R3)

1. In the pipeline's **History**, open run R3.

**Expected:**
- **Build (Synth) failed**; **UpdatePipeline, Assets and uk-qa did not run** for R3.
- Its Synth log ends with the failing test (`--- FAIL: …`, then `FAIL github.com/rlamba89/poised/…`) and `make: *** … Error`, and the build status **Failed**.
- QA wasn't changed by it: the uk-qa stage's last run is R2's (or R4's, after it).

### T-09: the revert goes green again (R4)

**Expected:** run R4 succeeded in every stage, like T-02, and its E2E folder in the results bucket has a `junit.xml`.

### T-10: the runbook, read as a newcomer

1. Read `infra/README.md` from top to bottom, as someone who has to (a) add a country, (b) deploy, (c) roll back, (d) read a failed run, and (e) move to a new AWS account.

**Expected:** each is there as numbered steps, using only things you could find (console paths, make targets). Report any step that's unclear, assumes knowledge, or names something you couldn't find in the console (e.g. a bucket or project name). Check in particular:
- "The manual steps": the GitHub connection's click-path matches what you see in T-01's console pages (don't click "Update").
- "Reading a failed run": following it for R3 gets you to the failing test's name.

### T-11: the second-region proof was done and destroyed

1. Read F4's Status (`docs/plans/f4-aws-pipeline.md`) and the runbook's "Proof" paragraph: the time is recorded.
2. Switch the console region to **US East (N. Virginia) us-east-1**. Check **CloudFormation → Stacks**, **Amplify**, **Lambda → Functions**, **Systems Manager → Parameter Store**, **S3** (bucket names containing `poised` and `us-east-1`).

**Expected:** about 10 minutes recorded; in us-east-1 nothing named `poised…` remains (no stacks, apps, functions, `/sj/…` parameters, or `cdk-poised-assets-…-us-east-1` bucket). Switch back to eu-west-2 afterwards.

### T-12: the pipeline costs nothing when idle

1. Open the pipeline `poised` and look at its summary at the top (don't click Edit).

**Expected:** **Pipeline type: V2** (billed per action minute; nothing while idle). In **KMS → Customer managed keys** there's no key for the pipeline. (The idle-cost figure itself is read by Rahul on 21 Oct: out of scope.)

### T-13: Release change re-runs the latest commit (the only thing you may start)

1. In the pipeline `poised`, click **Release change**, then **Release** in the dialog. Note the time.
2. Watch the stages for about 12 to 15 minutes (refresh the page now and then).

**Expected:** a new execution of the latest commit (`1d0d1a0` unless Rahul has pushed since) goes green in every stage, including E2E, with no other click. QA works afterwards (sign in as Cara Clinician).
*If it fails, don't retry: note the failed action and its log's last lines in the report.*

## Regression

- **R-1 (S06):** at the QA address, `/api/health` shows `ok`; `/api/config` shows `en-GB` and `Europe/London`.
- **R-2 (S06):** dev login as **Cara Clinician**, open **Episodes**, open any episode, reload: it stays on the episode.
- **R-3 (S06):** sign out, then open `/api/me` in the address bar: 401.
- **R-4 (S14):** at phone size (390×844), open the captured messages page (`/qa/messages`), open the newest patient invite link, and check the "Continue" page shows without any patient details before the date of birth.

## Out of scope

- The idle-cost figure (Rahul, 21 Oct), Training and Production stages (C10), Cognito (S12), real messages (S17).
- QA being open to anyone (accepted until S12).
- The journeys' UK test data in another country (pending.md).
- CDK's "cross-stack reference strength" warning in the Synth log.
