# S07 verification

## Verification 1 (14 Oct 2026, verifier, automatic run)

**Verdict: Verified.** Everything in the brief is built: `PipelineStack` with the test gates, the QA post-deploy steps (migrate, the seed with its run-twice test, the web deploy, the journeys with their results kept), the runbook, the second-region proof (done and destroyed), and the idle-cost check scheduled for 21 Oct. Both "Done when" checks are shown by the pipeline's own history, which I read myself rather than taking from the notes. The local suites pass. The pipeline's IAM is CDK Pipelines' scoped defaults plus two hand-scoped step roles, and no secret is in any build log or CodeBuild environment variable. None of the findings below blocks the session. Each is minor, or for later (C10, a second account), or comes with CDK's defaults. They're logged in [pending.md](../../pending.md) under "The pipeline (S07)".

S07's work was committed before verification (answers item 5), so I reviewed it as the range `b58a952..HEAD`: d5bb4e9, afda4cb, 1d8462c, 0ad1243, e9aecdb, 358bffe, 1d0d1a0, 8d09ca1 and 0dcfd9d.

### What was checked

- **Scope:** everything in the brief is built.
  - **Synth step** (`infra/pipeline_stack.py:36`), in order: `make lint test` → Postgres 17 from ECR Public → `make test-integration DB_UP=` → the web build → `make synth`.
  - **QA post-deploy:** Database-and-web (`db.sh`, then `deploy-web.sh`), then E2E, with `e2e/test-results` copied to S3 in `post_build`.
  - **Runbook:** in `infra/README.md`: add a country, deploy, roll back, read a failed run, move accounts.
  - **Proof:** recorded in F4 Status.
  - **Idle cost:** due 21 Oct.
  - **Not built (as the brief says):** Training and Production.
  - **Differences from the plan, all recorded in decisions.md and F4 Status:**
    - the pipeline follows `master`, not `main`;
    - migrate, seed and the web deploy are one step;
    - the failing-test check was "revert at once" rather than a branch pipeline (the plan allows either).
- **"Done when," checked in AWS** (read-only, profile `AdministratorAccess-026090536258`, eu-west-2):
  - `list-pipeline-executions`: R1 `8526b81f` (0ad1243), R2 `71edac70` (e9aecdb, WebhookV2), R4 `fc74d689` (1d0d1a0, WebhookV2) and the tester's `e789fefb` all **Succeeded**. R3 `604e8f99` (358bffe) **Failed**.
  - `list-action-executions` for R3 shows only `Source Succeeded` and `Build Synth Failed`. No UpdatePipeline, Assets or uk-qa action ran, so the failing test stopped the run before any deploy.
  - The trigger is a push to `master` only (V2 `gitConfiguration`), and there's no approval action. The connection `poised-github` is **AVAILABLE**.
  - The results bucket has `uk-qa/1/` to `uk-qa/4/` and blocks all public access.
- **The latest build logs:** I downloaded the latest Synth, Database-and-web and E2E logs and scanned them for `postgres://…@`, `neon.tech`, `npg_`, AWS keys and `aws_secret`: **0 hits**.
  - They show `78 passed`, `goose: no migrations to run`, `trusts: 2, users: 8, patients: 3`, `Deployed.` and `43 passed in 156.21s`.
  - CodeBuild project environment variables: only `RESULTS_BUCKET` (plaintext, not a secret). `db.sh` reads the URL from SSM into its own process, and the script has no `set -x`.
- **IAM** (the deployed roles' inline policies, read with `iam get-role-policy`):
  - **Synth:** only its log group, its report groups and the artifacts bucket. No lookup role and no AWS access beyond that, so the tests and `cdk synth` run with nothing they could misuse.
  - **Database-and-web:**
    - `ssm:GetParameter` on `/sj/qa/database-url` only;
    - `cloudformation:DescribeStacks` on `poised-uk-qa-web` only;
    - Amplify `CreateDeployment`, `StartDeployment` and `GetJob` on `apps/*` in its account and region (finding 5);
    - no `ssm:PutParameter`.
  - **E2E:** `s3:PutObject` on `e2eresults/uk-qa/*` only.
  - **Self-mutation, assets and the pipeline's action role:** CDK Pipelines' defaults (finding 4).
- **Self-mutation safety:** the Build stage (every test) runs before UpdatePipeline, so a change to the pipeline is tested before it's applied. The connection was approved only after S07's code was on GitHub (answers item 5).
- **Idle cost:**
  - the pipeline is V2 (per action minute), with no KMS key (`cross_account_keys=False`);
  - both buckets expire objects after 30 days and are deleted with the stack;
  - nothing in the stack is always on.
- **A-20:**
  - `PIPELINE` (account, region, repo, branch) is in `countries.py:121`. No stack names a region.
  - `test_nothing_of_the_pipelines_home_country_in_another_countrys_steps` synthesises a `ca-qa`-only pipeline.
  - The A-20 grep prints only `countries.py`, the existing `config.go` defaults and comments.
- **us-east-1** (the proof): no live `poised` stacks, no Amplify apps, no `/sj` SSM parameters, no `poised` buckets.
- **The seed:**
  - `seed()` is just `main`'s old transaction moved into a function.
  - `TestSeedTwice` (`apps/api/cmd/seed/seed_integration_test.go`) checks that every table's count is unchanged after a second run and that a tester's edits are put back.
  - No routes were added, so the five-case rule doesn't apply.
- **Security:** no secrets in the diff (the CI Postgres password `sj` is a throwaway container's). The Lifebox-org grep prints nothing. No app code changed apart from one line of sign-in text, so patient/staff separation and clinician-only stripping are untouched.
- **Docs:**
  - F4 Status, `docs/plans/README.md`, `docs/sessions/README.md`, `decisions.md` (14 Oct S07 section) and `pending.md` are updated.
  - `CLAUDE.md` lists `make deploy-pipeline`.
  - The runbook reads well. I checked its stage table, its failed-run reading and the move-accounts steps against the deployed pipeline.

### Findings (none blocking; worst first)

1. **The pipeline can only deploy to environments in its own account** (`infra/pipeline_stack.py:133`, `cross_account_keys=False`; the Database-and-web and E2E roles are in the pipeline's account).
   - A country (or QA) in another account would need `cross_account_keys=True` (a KMS key, about $1 a month) and cross-account roles for those two steps. That's a code change, against A-20's "config plus a runbook".
   - Fine today: every environment is in one account, and the runbook's "moving accounts" moves the pipeline with it. **Decide before the first environment in a different account.** Logged in pending.md.
2. **Migrations run after the new Lambdas are live** (`infra/pipeline_stack.py:143`: Database-and-web is a `post` step). A push with a migration plus code that needs it gives about 1 to 2 minutes of 500s on QA, or longer if goose fails.
   - Acceptable on QA. For C10 (Production), run goose as a `pre` step of the stage and keep migrations expand-then-contract. Logged in pending.md.
3. **Every environment in `countries.py` gets the demo seed and the journeys** (`infra/pipeline_stack.py:138`). Nothing checks `env.stage`, and there's no approval step, so a Training or Production entry added as "config only" would be seeded with the demo trusts and users.
   - Not reachable today (QA only, and C10 adds those stages). C10 must limit the seed and the journeys to QA (or fail synth otherwise). Logged in pending.md.
4. **The pipeline deploys with administrator rights in a shared account.**
   - CloudFormation's execution role from `make aws-setup` (S06) has `AdministratorAccess` (`infra/scripts/aws-setup.sh:23`). Any commit that reaches `master` can create anything in account 026090536258, which also holds another project.
   - The self-mutation role may assume any role tagged as a CDK bootstrap deploy or publishing role in the account, including other projects' bootstraps (CDK Pipelines' default condition matches by tag, not by qualifier).
   - Both are CDK defaults, and the repo is private with one writer. But they're the widest grants in the setup, and they aren't recorded as a decision. **For Rahul (not urgent):** before go-live (C10), give Poised its own account(s), or a scoped execution policy (`--cloudformation-execution-policies` with only the services Poised uses, plus a permissions boundary). Logged in pending.md.
5. **The Database-and-web role may deploy to any Amplify app in its account and region** (`infra/pipeline_stack.py:174`, `apps/*`), not just its own environment's. It could be scoped to the WebStack's app ARN. Minor; logged in pending.md.
6. **Nits (not logged):**
   - **The wait for Postgres** (`pipeline_stack.py:39`) exits 0 even if Postgres never became ready, so the failure shows up as integration-test connection errors instead.
   - **The artifact bucket reuses `RESULTS_DAYS`** (`pipeline_stack.py:114`).
   - **`lib.sh:11` falls back to `python3` on laptops too** when `infra/.venv` is missing.
   - **`PIPELINE` repeats uk-qa's account** (`countries.py:121`). The runbook's move-accounts step 3 says to change both.
   - **`make aws-setup ENV=pipeline` would write `/database-url`** (nobody is told to run it).

### Test results (14 Oct, this verification)

- `make lint`: passes.
- `make test`:
  - Go: all `ok`;
  - Vitest: **144 passed** (web) and **116 passed** (clinical);
  - pytest (CDK + Makefile): **78 passed**.
- `make test-integration`: all `ok`, including `cmd/seed` (`TestSeedTwice`) and `internal/httpapi`.
- `make e2e`: not run locally. The only app change is one line of sign-in text, and the pipeline ran all 43 journeys green against QA four times (R1, R2, R4 and `e789fefb`).
- **Spot checks on QA** (https://main.d33kh0gi8vk83v.amplifyapp.com, with curl):
  - `/api/health` → 200;
  - `/api/me` with no sign-in → 401 `{"error":"Please sign in."}`;
  - the deep link `/episodes` → 200 (the SPA);
  - `/api/config` → `en-GB`, `Europe/London`, NHS;
  - the deployed `Login-*.js` chunk has R2's text, "Development sign-in (local and QA only, until Cognito): choose a user."
- **Test report 1:** 17 of 17 passed.

### Hand checks for Rahul

1. **The connection page** (from test report 1): in **Developer Tools → Settings → Connections → `poised-github`**, check that the runbook's click-path labels (**Update pending connection**, **Install a new app**, **Connect**) match the console. Don't click Update.
2. **The CodeBuild link:** in a pipeline action's **View details**, check that the link is called **Link to execution details** (`infra/README.md`, "Reading a failed run", step 1).
3. **Idle cost, 21 Oct:** read Cost Explorer and record the figure in F4 Status (plan F4 Step 4).

### Commits

S07's work is already committed. These are on GitHub: `d5bb4e9` (the seed's run-twice test), `afda4cb` (the pipeline), `1d8462c` (the runbook), `0ad1243` (docs), `e9aecdb` (the visible change), `358bffe` (the failing test) and `1d0d1a0` (its revert).

These are local only: `8d09ca1` and `0dcfd9d` (docs), plus this verification's docs commit: verify.md, the status rows and the pending items.

No history was rewritten. **Rahul pushes:** `git push origin master` publishes the three docs commits. They're docs only, so the pipeline will run again but deploy the same code.
