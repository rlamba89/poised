# S07 build notes

Builder: automatic run (`/run-sessions`), 14 Oct 2026. Answers: [answers.md](answers.md) (Rahul, 14 Oct) and decisions.md §2 (14 Oct).

**Status: tested (14 Oct, [test report 1](test-report-1.md): 17 of 17 passed; the tester's Release change, execution `e789fefb`, went green).** Previously: test plan ready. Committed locally before verification (answers item 5); Rahul pushes. Five real runs: R1, R2, R4 and the tester's green; R3 stopped at Synth by a deliberately failing test.

## Steps

| # | Step | State |
| --- | --- | --- |
| 1 | The seed is idempotent, with a run-twice integration test (plan F4 Step 2, before the pipeline seeds QA) | Done |
| 2 | `PipelineStack` (CDK Pipelines, tests first): source, synth step with the test gates, QA stage with post-deploy steps, results bucket | Done: 17 CDK tests |
| 3 | `app.py` → the pipeline holds the stages; Makefile `deploy-pipeline`/`destroy-pipeline`, `DB_UP`, hand deploys renamed to `pipeline/<env>/*` | Done |
| 4 | Deploy the pipeline (`make deploy-pipeline`) | Done 14 Oct: `poised-pipeline` (66 s); connection `poised-github` **Pending** |
| 5 | Rahul pushes `master`, then approves the connection; the first green run through QA e2e | Done: R1 green (14.5 min) |
| 6 | A trivial visible change goes green to QA; a deliberately failing unit test stops the run before deploy, then its revert | Done: R2 green, R3 stopped at Synth, R4 green |
| 7 | The runbook in `infra/README.md`: add a country, deploy, roll back, read a failed run, the connection's click-path | Done |
| 8 | The second-region proof (answers item 1: yes) | **Done 14 Oct: about 10 minutes; destroyed** |
| 9 | The idle-cost check: scheduled for 21 Oct (Rahul reads Cost Explorer) | Noted in F4 Status and pending.md |
| 10 | Test plan | Done |

## What was built

**Go (Rahul reviews):**
- `cmd/seed/main.go`: `main` now calls `seed(ctx, db, dir)`, which runs the same three loads in one transaction. Nothing else changed: the seed was already idempotent.
- `cmd/seed/seed_integration_test.go` (new, `-tags integration`): `TestSeedTwice` runs the seed into a fresh test database, checks no table is empty, has a "tester" suspend Trust A and rename Cara, runs it again, and checks every table has exactly the same number of rows and that Trust A is active and Cara is named again.

**CDK (Python, `infra/`):**
- `countries.py`: `PIPELINE = PipelineHome(account, region, repo="rlamba89/poised", branch="master")`, checked when made. `python countries.py pipeline <field>` prints one value (for the scripts).
- `pipeline_stack.py` (new): `PipelineStack` (`poised-pipeline`):
  - **the GitHub connection `poised-github`** (`AWS::CodeConnections::Connection`), created Pending;
  - **source:** `PIPELINE.repo` at `PIPELINE.branch`, through the connection;
  - **synth step** (`SYNTH_COMMANDS`, CodeBuild arm64 Amazon Linux 2023, privileged for Docker, runtimes Go 1.26 / Node 22 / Python 3.12): `make lint test` → `docker run` Postgres 17 (from ECR Public) and wait for it → `make test-integration DB_UP=` → `npm run build -w apps/web` → `make synth` (builds the Lambdas, then `cdk synth`). Outputs `infra/cdk.out` and `apps/web/dist`.
  - **per environment in `countries.py`** a stage (ApiStack, WebStack), then two post steps: **Database-and-web** (`bash infra/scripts/db.sh <env>` then `bash infra/scripts/deploy-web.sh <env>`; its role may read only the environment's database-url parameter, describe only its web stack, and deploy to Amplify in its region) and, after it, **E2E** (CodeBuild Ubuntu `standard:8.0`: `make e2e-setup`, `playwright install-deps chromium webkit`, `make e2e ARGS="--junitxml=… $E2E_ARGS"` with `E2E_BASE_URL` from WebStack's `AppUrl` output; `post_build` copies `e2e/test-results` to the **E2eResults** bucket whether it passed or failed);
  - a V2 pipeline named `poised`, no KMS key, self-mutating with the pinned CDK CLI, assets in one job, both buckets deleted with the stack (30-day expiry).
- `app.py`: makes only the pipeline stack; the stages are inside it (`pipeline/uk-qa/api` = `poised-uk-qa-api`, the same stack names, so the pipeline updates S06's stacks rather than replacing them).
- `web_stack.py`: keeps its `AppUrl` output as `app_url_output` for the E2E step.
- `scripts/lib.sh`: uses the image's `python3` when there's no `infra/.venv` (countries.py needs only the standard library); accepts `pipeline` as the entry; the CDK-version check moved to `aws-setup.sh`, the only script that runs the CDK CLI.

**Makefile:** `deploy-pipeline` and `destroy-pipeline` (both with `PROFILE=` and the account check); `deploy-stacks`/`destroy-stacks` now name `'pipeline/$(ENV)/*'`; `db-up` runs `$(DB_UP)` (default `docker compose up -d --wait db`), so the pipeline can run `make test-integration DB_UP=` against its own Postgres.

**Docs:** `infra/README.md` rewritten around the pipeline: the connection's click-path, setting up an account, the pipeline's stages, deploying, rolling back, reading a failed run (and keeping traces of a passing run with `E2E_ARGS=--tracing=on`), **adding a country**, removing one, moving accounts.

No TypeScript changed.

## Decided by the builder (automatic mode)

- **The pipeline follows `master`**, the GitHub repo's default branch and the local branch (`PIPELINE.branch` in `countries.py`). Plan F4 and the brief say `main`; renaming the branch would change every clone and the GitHub default for no gain. Amplify's branch `main` is unrelated (where the zips go).
- **The pipeline's home is in `countries.py`** (`PIPELINE`), with the UK QA's account and region, so no stack names a region (A-20).
- **The stages live inside the pipeline stack** (CDK Pipelines' usual shape), so hand deploys name `pipeline/<env>/*`. The stack names didn't change, so nothing was replaced.
- **Synth on arm64 Amazon Linux 2023 (`LARGE`), E2E on Ubuntu x86 (`standard:8.0`, `MEDIUM`).** The Lambdas are arm64 and CodeBuild's AL2023 arm image has Go 1.26, Node 22 and Python 3.12; Playwright's WebKit (the patient's iPhone in J3, J7, J7c, J8) supports only Ubuntu and Debian.
- **Postgres from ECR Public** (`public.ecr.aws/docker/library/postgres:17`; a test keeps the major version equal to `docker-compose.yml`'s): Docker Hub limits anonymous pulls and CodeBuild hosts share addresses.
- **The web app is built once, in the synth step,** and the Database-and-web step uploads that build (plan: "the builds" in synth).
- **One Database-and-web step** (migrate, seed, web) rather than three: they run in that order anyway, and one CodeBuild start saves about a minute per run.
- **Traces:** the journeys keep traces only on failure (as locally); the E2E step copies `e2e/test-results` (also `junit.xml`) to the E2eResults bucket in `post_build`, which runs even when the journeys fail. A CodeBuild report group was not used: CodeBuild creates those outside CloudFormation, so they'd outlive `cdk destroy` and break "re-create with no manual work".
- **Both buckets are deleted with the stack** (auto-delete objects, 30-day expiry), so `make destroy-pipeline` leaves nothing behind.
- **V2 pipeline, no KMS key:** V2 bills per action minute (V1 is $1 a month per pipeline); one account needs no cross-account key ($1 a month).
- **The CDK CLI version is written twice** (Makefile and `pipeline_stack.py`), with a test that they match, rather than parsing the Makefile from Python.
- **The proof's database** was a separate database (`poised_us_qa_proof`) in the existing Neon project (London), created and dropped with SQL: making a Neon project in `us-east-1` needs Rahul's Neon login. Everything in AWS was in `us-east-1`.

## The second-region proof (14 Oct, answers item 1)

Followed the runbook with a temporary `us-qa` entry in `countries.py` (account 026090536258, **us-east-1**, America/New_York, en-US, phone country US, dev login on), not committed:

| Time (BST) | Step |
| --- | --- |
| 14:01:44 | Entry added; separate Neon database created |
| ~14:03 | `make aws-setup ENV=us-qa`: Poised's CDK bootstrap in us-east-1, the database URL and the alert address in SSM (1.5 min) |
| 14:03–14:09 | `make deploy-qa ENV=us-qa`: ApiStack 60 s + WebStack 12 s, goose up to version 17 + seed (2 s), web upload (5.5 min in all, mostly the Lambda builds and the npm build) |
| 14:09–14:11 | `GET /api/config` → `en-US`, `America/New_York`; a deep link → 200; **J1, J2, J4, J5 passed; J3 failed** at its UK test data |
| 14:13–14:14 | J3 with a US mobile and the en-US date ("May 17, 1980") in a throwaway copy of the file (deleted): **passed** |
| 14:15–14:20 | Destroyed: both stacks (1 min), the SSM parameters, `poised-cdk-toolkit` and its retained assets bucket (4 object versions); the Neon database dropped; the entry removed |

- **Result: about 10 minutes of work from a new entry to J1–J5 passing**, plus 5 to destroy. No code changed.
- **J3's failure is the per-country rules working:** the US entry refused the UK mobile ("Enter a valid mobile number, or leave it empty.") and showed dates as "May 17, 1980". The journeys' test data is the UK's (J3, J7, J8 and the patient-records journey use a UK mobile or a UK-format date), noted in the runbook and pending.md.
- **Checked afterwards in us-east-1:** no stacks, no SSM parameters, no Amplify apps, no Lambda functions, no Poised log groups, no resources tagged `project=poised`, no `poised-us-qa` budget.

## The pipeline's runs (14 Oct)

| Run | Commit | Execution | Result |
| --- | --- | --- | --- |
| R1 | `0ad1243` (S07, pre-verification), started with "Release change" after Rahul pushed and approved the connection | `8526b81f` | **Green, 14:28:41–14:43:05 (14.5 min)**: Source 5 s; Synth 4 min (queue 1 min; lint + unit tests 77 s, Postgres 9 s, integration tests 25 s, web build 7 s, synth); SelfMutate 1 min; FileAssets 32 s; ApiStack 1 min 12 s; WebStack 1 min 11 s; Database-and-web 1.5 min (goose: no migrations; seed; Amplify job 2); **E2E 4.5 min: 43 passed** against QA; `junit.xml` in `s3://poised-pipeline-e2eresults…/uk-qa/1/`. No database address in any log. |
| R2 | `e9aecdb` (the visible change: the dev sign-in page's text), pushed by Rahul; started by the push (webhook) | `71edac70` | **Green, 14:48:13–14:59:31 (11.3 min)**: Synth 3 min; SelfMutate 32 s; deploys 1.5 min; Database-and-web 1 min; **E2E 4.5 min: 43 passed** (`uk-qa/2/junit.xml`). QA's sign-in page bundle has the new text ("Development sign-in (local and QA only, until Cognito): choose a user."). |
| R3 | `358bffe` (a deliberately failing Go unit test, `TestS07PipelineStopsOnAFailingTest`), pushed by Rahul | `604e8f99` | **Stopped at Build, 15:09:43–15:12:23 (2.7 min):** Synth (CodeBuild build 3) failed at `make lint test` with `--- FAIL: TestS07PipelineStopsOnAFailingTest` … `make: *** [Makefile:63: test] Error 1`. UpdatePipeline, Assets and `uk-qa` didn't run for it: their last execution stays R2's (`71edac70`), so QA kept R2's deploy. |
| R4 | `1d0d1a0` (the revert of R3), pushed by Rahul | `fc74d689` | **Green, 15:25:17–15:38:09 (12.9 min)**: Synth 3.6 min; deploys 2.4 min; Database-and-web 1 min; **E2E 4.6 min: 43 passed** (`uk-qa/3/junit.xml`). |

Nothing needed fixing: the first real run went green. **The brief's "Done when" is shown:** a push to the branch goes green all the way through QA e2e with no manual step (R2, R4), and a failing unit test stops the run before deploy (R3; plan F4's "or revert at once", not a branch pipeline).

## Tests

- **Go integration:** `TestSeedTwice` (red first: `seed` undefined; then green).
- **CDK (pytest, `infra/tests/test_pipeline.py`, 17):** the connection (GitHub, `poised-github`); the source is `PIPELINE`'s repo and branch through it; the real one follows `rlamba89/poised` `master`; V2 with no KMS key; the synth commands in order (tests → Postgres → integration → web build → synth) with `DB_UP=` and the ECR Public Postgres of compose's major version; privileged arm64 with the runtimes; the two outputs; the Makefile's CDK version used for self-mutation; per environment: deploys → Database-and-web → E2E in run order; the Database-and-web commands and least-privilege role (its region, its stack, no `ssm:PutParameter`); E2E's commands, Ubuntu image and `E2E_BASE_URL` from WebStack's output; results copied after the run, 30-day expiry; nothing of the UK in a `ca-qa`-only pipeline; buckets deleted with the stack; Poised's bootstrap; the real pipeline synthesises.
- **Makefile (pytest, `test_makefile.py`, +4):** `deploy-pipeline` and `destroy-pipeline` pass `PROFILE`; hand deploys name `pipeline/<env>/*`; `DB_UP=` skips docker compose.

**Results (14 Oct):**
- `make lint`: passes. `make test`: Go all ok; Vitest 144 (web) + 116 (clinical); CDK + Makefile **78 passed**.
- `make test-integration DB_UP=` (against the compose Postgres, as the pipeline will against its own): all ok, including `cmd/seed`.
- `npm run build -w apps/web`, `make synth`: ok (the synth's only warning is CDK's "cross-stack reference strength", as in S06).
- `make deploy-pipeline PROFILE=AdministratorAccess-026090536258`: created `poised-pipeline` (connection, pipeline, 5 CodeBuild projects, 2 buckets). The automatic first run failed at **Source** (connection Pending), as expected.
- `make e2e` against `make dev`: not run yet (no app code changed; to run before the test round).
- **The pipeline's runs:** above (R1 to R4).

## Manual testing

- **Test report 1 (14 Oct): 17 passed, 0 failed, 0 blocked** (T-01 to T-13, R-1 to R-4). The tester did the AWS checks with the read-only AWS CLI. Its one action, T-13's Release change (`e789fefb`, commit `1d0d1a0`), went green in about 12 minutes, with E2E 43 passed (`uk-qa/4/junit.xml`).
- **Hand check for Rahul** (the tester couldn't open the console): in **Developer Tools → Settings → Connections → `poised-github`**, check that the runbook's click-path labels (**Update pending connection**, **Install a new app**, **Connect**) match the console (don't click Update). In a pipeline action's **View details**, check that the CodeBuild link is called **Link to execution details** (`infra/README.md`, "Reading a failed run" step 1).
- **Exploring notes, handled as doc-only changes after the report (no code or infra changed):**
  - 1: the runbook's proof destroy time now says about 5 minutes (stacks about 1), matching F4 and these notes.
  - 2: "Reading a failed run" now says that a failed Synth log ends with a side-effect `infra/cdk.out` artifact error, and that the real failure is the `Command did not exit successfully …` line about 20 lines above it.
  - 3: the runbook no longer says the E2E step prints the QA address: the build has it as `E2E_BASE_URL`, and the log prints the S3 results path. `junit.xml` labelling every journey `[chromium]` is in pending.md.
  - 4: the runbook has "Finding the pipeline's pieces in the console" (CodeBuild projects are named by CDK, `Pipeline…`; the Lambda log groups). Naming the projects `poised-…` is in pending.md.
  - 5: test plan T-06 now says Alex Author may first choose Hospital A1 (older behaviour).
  - 6 and 7 need nothing.

## Commits

Made locally before verification (answers item 5); Rahul pushes. On `master` after `b58a952`: `d5bb4e9`, `afda4cb`, `1d8462c`, `0ad1243` (below, 1 to 4), then the checks `e9aecdb` (visible change), `358bffe` (failing test) and `1d0d1a0` (its revert), then the docs commits `8d09ca1` (the test plan and the runs) and the one after test report 1 (the runbook's doc fixes, pending.md, status).

1. **The seed's run-twice test (F4 Step 2 / S07)**: `apps/api/cmd/seed/{main.go,seed_integration_test.go}`.
2. **The pipeline (F4 Step 2 / S07)**: `infra/{pipeline_stack.py,app.py,countries.py,web_stack.py}`, `infra/scripts/{lib.sh,aws-setup.sh}`, `infra/tests/{test_pipeline.py,test_makefile.py}`, `Makefile`.
3. **The runbook (F4 Step 3 / S07)**: `infra/README.md`.
4. **Docs: S07 build notes, questions, decisions and status**: `docs/**`.
