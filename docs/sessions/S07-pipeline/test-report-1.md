# S07 test report 1

**17 passed, 0 failed, 0 blocked** (2 console-wording sub-checks left as hand checks for Rahul, see below).

- **Environment:** repo HEAD `8d09ca1` (working tree not clean: the skills, the S12 brief, and S17/run-sessions/reports/research_notes are untracked or modified, none of them S07 code). `origin/master` = `1d0d1a0`.
- **When:** 14 Oct 2026, 15:39–15:56 BST.
- **App:** QA at https://main.d33kh0gi8vk83v.amplifyapp.com (Chrome, through Claude in Chrome).
- **AWS:** account 026090536258, eu-west-2 (plus us-east-1 for T-11). Automatic mode: the Chrome browser wasn't used for the AWS console, so every AWS check was done with the **read-only AWS CLI** (`AWS_PROFILE=AdministratorAccess-026090536258`): describe/list/get calls, CodeBuild/CloudWatch log reads, S3 list/head/get of `junit.xml`. The only change was T-13's single `aws codepipeline start-pipeline-execution --name poised` (15:39:44 BST, execution `e789fefb`).
- **Secrets:** all five build logs I downloaded, plus T-13's Database-and-web log, were scanned for database URLs, Neon hosts, passwords and AWS keys. None were found, and none were printed.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 connection and source | Pass | `poised-github`: GitHub, **AVAILABLE**. Source action `rlamba89_poised` (CodeStarSourceConnection), `FullRepositoryId` `rlamba89/poised`, branch `master`. |
| T-02 R2 green end to end | Pass | `71edac70`, `e9aecdb`: Source → Synth → SelfMutate → FileAssets → api.Prepare/Deploy → web.Prepare/Deploy → Database-and-web → E2E, all Succeeded, in that order. The pipeline definition has no Approval action. |
| T-03 Synth runs every test first | Pass | R2 Synth (build #2): `make lint test`, Go `ok` lines, Vitest 144 + 116 passed, pytest 78 passed, `docker run … postgres:17` (ECR Public), `make test-integration` (incl. `cmd/seed`, `cmd/platform-admin`), `npm run build -w apps/web`, `make synth`, in that order. Build SUCCEEDED. `ARM_CONTAINER`, `amazonlinux-aarch64-standard:3.0`, privileged. |
| T-04 migrate, seed, web deploy | Pass | Account check line exactly as expected; `goose: no migrations to run. current version: 17`; `trusts: 2, users: 8, patients: 3`; `Deploying job 3 to https://main.d33kh0gi8vk83v.amplifyapp.com …`, `Deployed.` No DB address or password. |
| T-05 journeys against QA | Pass | 13 journey files (audit, J1–J8, J7c, patient records, staff, trusts); `43 passed in 151.57s`; ends with the S3 line `…/uk-qa/2/`. Chromium and WebKit both installed. The QA address appears only in the build's `E2E_BASE_URL` env var, not in the log (see exploring #3). |
| T-06 the change is on QA | Pass | Sign-in text is "Development sign-in (local and QA only, until Cognito): choose a user." Alex Author → Hospital A1 → Questionnaires list. Loading the Episodes deep link directly opens it (no 404). |
| T-07 e2e results kept | Pass | Bucket `poised-pipeline-e2eresults63613c7a-kr11vd6eyxar`: `uk-qa/1/`, `/2/`, `/3/` (and `/4/` from T-13) each hold `junit.xml` (R2: tests=43, failures=0, errors=0). Lifecycle: one rule, Expiration **30 days**, enabled. Public access fully blocked. |
| T-08 failing test stops the run | Pass | `604e8f99` (`358bffe`): Source ok, **Synth Failed**; no UpdatePipeline/Assets/uk-qa actions exist for it. The log shows `--- FAIL: TestS07PipelineStopsOnAFailingTest`, `FAIL github.com/rlamba89/poised/apps/api/internal/clock`, `make: *** [Makefile:63: test] Error 1`, build FAILED. Amplify jobs: #3 (R2, 14:54) then #4 (R4, 15:33), with nothing during R3. |
| T-09 revert green again | Pass | `fc74d689` (`1d0d1a0`): every action Succeeded; E2E build #3 `43 passed`; `uk-qa/3/junit.xml` present. |
| T-10 runbook as a newcomer | Pass (with notes) | (a) to (e) are each there as numbered steps; every make target named exists in the Makefile, and the bucket and project names match. "Reading a failed run" for R3 leads to `TestS07PipelineStopsOnAFailingTest` via "scroll up to the first FAIL". The console button labels ("Update pending connection", "Link to execution details") couldn't be compared with the real console: hand check. Small doc notes in exploring #1, #2 and #4. |
| T-11 second-region proof destroyed | Pass | F4 Status and the runbook record about 10 minutes. us-east-1: no live `poised*` CloudFormation stacks (`poised-us-qa-web`, `-api` and `poised-cdk-toolkit` are DELETE_COMPLETE, created 14:02–14:04, deleted 14:14–14:15 BST); no Amplify apps; no `poised` Lambda functions; no `/sj/…` or `poised` SSM parameters; no `cdk-poised-assets-…-us-east-1` bucket; no Poised log groups. |
| T-12 costs nothing idle | Pass | `pipelineType: V2`. The artifact bucket uses SSE-S3 (AES256). The only customer-managed KMS keys in eu-west-2 belong to another project (asgard), or are an undescribed key from 2024 that predates Poised: none for the pipeline. |
| T-13 Release change | Pass | Started 15:39:44 BST (`e789fefb`, trigger StartPipelineExecution, commit `1d0d1a0`). Every action Succeeded by 15:51:58 (about 12 min). E2E `43 passed in 156.21s`, `uk-qa/4/junit.xml`. DB step: no migrations, seed counts as before, Amplify job 5 deployed. Afterwards QA works as Cara Clinician (Episodes lists the run's new E2E episodes). No other click or retry. |
| R-1 health/config | Pass | `/api/health` 200 `{"status":"ok"}`; `/api/config` has `en-GB`, `Europe/London` (NHS number scheme). |
| R-2 episode reload | Pass | Cara Clinician → Episodes → first episode → reload: it stays on the episode. |
| R-3 /api/me after sign-out | Pass | Signed out, then opened `/api/me` in the address bar: `{"error":"Please sign in."}`, status 401 (confirmed with fetch). |
| R-4 phone-size invite | Pass | The window wouldn't resize, so I used a 390×844 iframe. I opened the newest captured invite (`/i/p3Qq…`): "Your health questionnaire … Continue", with no name, hospital or other patient details, and no horizontal scroll (scrollWidth 388). |

## Failures

None.

## Hand checks for Rahul

1. **T-10 / T-01, the console's wording:** in **Developer Tools → Settings → Connections → `poised-github`**, check that the runbook's click-path labels (**Update pending connection**, the GitHub Apps / "Install a new app" step, **Connect**) match what the console shows. Don't click Update. In a pipeline action's **View details**, check that the CodeBuild link is called **Link to execution details**, as `infra/README.md` "Reading a failed run" step 1 says. I could only check the CLI side: the connection is AVAILABLE, and the names match.

## Found while exploring

1. **The runbook's destroy time disagrees with the build notes (cosmetic).** `infra/README.md` "Proof" says it "was then destroyed in about 3 minutes". F4's Status says "(5 min)", and the build notes say "14:15–14:20 … plus 5 to destroy". CloudFormation shows the stacks deleted between 14:14:52 and 14:15:59 BST, so "3" may be just the stacks.
2. **A failed Synth log's last line is a different error (minor, docs).** In R3 the log ends with `no matching base directory path found for infra/cdk.out` (UPLOAD_ARTIFACTS failing because synth never ran). The real cause, `Command did not exit successfully make lint test …`, is about 20 lines above it. The runbook says "The log's end shows which command". A newcomer could chase the cdk.out message first, so one line saying "ignore a following `infra/cdk.out` artifact error" would help.
3. **The E2E log never prints the address it tests (cosmetic).** pytest runs with the default output (dots per file), and the base URL is only in the build's environment variables (`E2E_BASE_URL=https://main.d33kh0gi8vk83v.amplifyapp.com`, seen through `batch-get-builds`). The runbook says the `AppUrl` is "also printed in the pipeline's E2E step", but only the S3 results path is printed. Also, `junit.xml` labels all 43 tests `[chromium]`, although the patient journeys use WebKit through the `phone` fixture (`e2e/conftest.py`), so the report doesn't show which journeys ran on WebKit.
4. **Most CodeBuild projects aren't named `poised…` (cosmetic, docs).** `PipelineBuildSynthCdkBuildP-…`, `PipelineAssetsFileAsset5D8C-…`, `PipelineukqaDatabaseandwebA-…` and `PipelineukqaE2E9D841BB0-…` (only `poised-selfupdate` is). The Lambda log groups are `poised-uk-qa-api-staffapilogs…` rather than `/aws/lambda/poised-…`. In a shared account, the test plan's "look only at things named `poised…`" doesn't find them. The runbook's "the `…E2E…` project" does work.
5. **Alex Author lands on "Choose a hospital" (Hospital A1/A2) first, not the Questionnaires list (older behaviour).** The Questionnaires list opens after choosing a hospital. The test plan's wording is slightly off; this isn't an S07 problem.
6. **The session cookie survives a redeploy (good).** Cara's session from before T-13 still worked after the redeploy (the token secret isn't rotated by a deploy).
7. **`/api/<unknown>` returns a plain-text 404 and an unknown asset returns 404,** while an unknown page path returns the SPA (200 text/html), as the rewrite rules intend.

## Console and network errors

- No console errors or warnings on the pages visited.
- **A tool quirk, not an app bug:** Claude in Chrome's network log reported `503` for `POST /api/dev/login` and `POST /api/logout`, but sign-in and sign-out both worked. A direct `fetch('/api/logout', {method:'POST'})` returned **204** (`ok: true`, via CloudFront and API Gateway), so the extension mis-reports these 204 No Content responses.

## GIFs

- `~/Downloads/S07-T06.gif`: the main happy path (QA sign-in page with R2's text → Alex Author → hospital → Questionnaires → Episodes deep link).
- No failures, so there are no failure GIFs.
