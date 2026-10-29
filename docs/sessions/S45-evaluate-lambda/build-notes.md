# S45 build notes: the evaluate Lambda

[Brief](brief.md). Plan C6, part 8 of 8. **Code part built 28 Oct (builder, with no AWS access); the QA deploy, the measurements and the manual tests are still to do, with Rahul and his AWS profile** (see the last section).

## What was built

### `apps/evaluate` (TypeScript for Rahul to review)

- **`src/lambda.ts` (new): the Lambda's entry point.** Go sends one JSON payload, `{"op": "evaluate" | "publish-problems" | "shown" | "health", "input": …}`, and gets back `{"status": 200, "body": …}`. The body is exactly what the local HTTP service answers for the same path, because both call the same function (`run` in `handler.ts`). In Go terms: `makeHandler(version)` returns a closure, like a Go func returning an `http.HandlerFunc`; `handler` is that closure built with the build's version, and it's what Lambda calls (`index.handler`). It never throws: a bad payload is a 400, a failure inside the checks a 500 (Go counts anything but 200 as a failure, so the send fails safe).
- **Its log:** one line a call, `evaluate: evaluate form <form id>: 200 in 41 ms` (the op, the episode form's id only when it looks like a UUID, the status, the milliseconds), plus `handler.ts`'s existing error line (the error's kind and code locations, never its message). Never an answer, a name or a date of birth; a test checks it.
- **`src/handler.ts`:** the HTTP routing and the work are split. `run(op, input, version)` does the work for one op (unchanged logic, moved out of `handle`); `handle(method, path, body)` parses the HTTP request and calls `run`. `OP_NAMES` lists the three ops (a Go test checks Go's names match).
- **`build.mjs`:** `node build.mjs` builds the local service as before; `node build.mjs --lambda <dir>` builds `<dir>/index.js` from `src/lambda.ts` (one CommonJS file, esbuild target `node24` for the Lambda; the local service keeps `node22`). The version is the git commit as before; in the pipeline, where CodeBuild has no `.git`, it falls back to `CODEBUILD_RESOLVED_SOURCE_VERSION` (the commit CodeBuild was given), so the stored results still name the commit.
- **`packages/clinical/src/scores.ts`:** `scoreAsShown`'s cache of parsed scores is emptied once it holds 500 expressions (pending.md asked for a bound if S45 keeps a warm process: a Lambda's process serves many calls while warm).

### Go

- **`internal/evaluate/lambda.go` (new): `evaluate.Lambda`**, the second `Evaluator`. It invokes the function synchronously (AWS SDK v2 `lambda.Invoke`, `RequestResponse`) behind a one-method interface `LambdaClient`, so tests inject a fake. Same time limits as HTTP (5 s for a send and a review, 20 s for the publish checks). An error is: the invoke failing (e.g. no permission), a **function error** (Lambda's `FunctionError`: it threw, timed out or ran out of memory), an invoke status other than 200, a reply that isn't JSON, a reply status other than 200, or a body that isn't a result. No error holds the payload or the function's message.
- **`internal/evaluate/evaluate.go`:** the decoding (`evaluateVia`, `publishProblemsVia`, `shownVia`) is shared by HTTP and Lambda; HTTP's behaviour is unchanged (its decode errors now say `evaluate: shown: …` instead of `evaluate: /shown: …`).
- **`internal/config`: `EVALUATE_FUNCTION`** (the function's name or ARN). With it set, `EvaluateURL` stays empty. **Setting both is refused**; with `CONFIG_STRICT` and neither, start-up stops with "not set: EVALUATE_FUNCTION or EVALUATE_URL". `EVALUATE_URL` and its local default are unchanged.
- **`internal/serve`: `serve.Evaluator(ctx, cfg, newClient)`** picks Lambda or HTTP from the config; `Setup` uses it, so `cmd/staff-api`, `cmd/patient-api`, `cmd/api` and `cmd/jobs` all get it without changes of their own. The Lambda client is made only when `EVALUATE_FUNCTION` is set (so local dev never loads AWS credentials). The start-up line on Lambda now names it (`evaluate Lambda poised-<env>-evaluate`).
- `go.mod`: adds `aws-sdk-go-v2/service/lambda` v1.113.4 (patch bumps of the SDK core and smithy-go came with it).

### CDK (`infra/api_stack.py`)

- **The function `poised-<env>-evaluate`:** `nodejs24.x` (Rahul, 28 Oct; see below), arm64, `index.handler`, 1024 MB, 20 s, from `apps/api/dist/evaluate` (refuses to synth without `index.js`, as the Go functions do without `bootstrap`); its own log group (14 days). **No environment variables, no VPC, no secrets; its role is CDK's default, which only has `AWSLambdaBasicExecutionRole` (write its own logs).**
- **Only staff-api and patient-api may invoke it** (`grant_invoke`: `lambda:InvokeFunction` on its ARN in their roles); no resource policy, so nothing else can.
- **`EVALUATE_FUNCTION`** is set on the Go functions (a `Ref` to the function, so CloudFormation deploys it first).
- **The alarm `poised-<env>-evaluate-failures`:** a metric filter for `"EVALUATE_FAILED"` on staff-api's and patient-api's log groups (metric `Poised/<env>` `EvaluateFailures`), ≥ 1 in 5 minutes, to the existing alert topic, exactly as `DELIVERY_FAILED`'s. A Go test checks `EVALUATE_FAILED_MARKER` in `api_stack.py` is Go's `evaluate.FailedMarker`.
- Per-country values: none added; the function's name comes from the entry's name only (A-20; the made-up second country's test still passes).

### Build and pipeline

- **`make build-lambdas`** builds the three Go functions as before, then `apps/evaluate/build.mjs --lambda apps/api/dist/evaluate` (it now depends on `node_modules`). The bundle is 2.6 MB.
- **The pipeline is unchanged:** its `make synth` runs `make build-lambdas`, after `make lint test` has installed `node_modules`; esbuild's arm64 Linux binary is in the lockfile. Only a comment changed in `pipeline_stack.py`.

### Docs

`infra/README.md` (the function, the alarm, the log group, and a short "The evaluate Lambda on QA" section: a health check by hand, where the timings are, how to break it for the manual test); `docs/saas-requirements.md` §13 (the diagram says "on publish" too; the table row says arm64 and who may invoke); `.env.example`; `docs/pending.md` (two S45 items marked built).

## Decided by the builder

1. **One payload with an `op` field**, as the caller asked, rather than API Gateway-style events: the reply carries the HTTP service's status and body, so Go's decoding is shared.
2. **The jobs Lambda gets `EVALUATE_FUNCTION` too, but not the right to invoke.** Go's strict config needs one of the two settings to start and `serve.Setup` is shared; the jobs never evaluate. If one ever did, the call would fail safe and the alarm would say so (jobs' log isn't watched for it, though).
3. ~~**Node 22**~~ → **Node 24, Rahul's answer (28 Oct).** The Lambda runs on `nodejs24.x` and its bundle targets `node24`; the repo, the local service and the pipeline stay on Node 22 and move to 24 before go-live. Checked: the pinned `aws-cdk-lib` 2.273.0 has `Runtime.NODEJS_24_X`; the bundle requires no Node built-in module at all (pure JavaScript), so nothing Node 23–24 removed (`util.is*`, `SlowBuffer`, `tls.createSecurePair`, `fs.F_OK`…) is used. The bundle test checks the target; the CDK test checks `nodejs24.x`.
4. **1024 MB**, for a quicker cold start and call while the send's and publish's locks are held (Lambda gives CPU in proportion to memory). Cost is per use, so nothing when idle. To revisit with the QA measurements.
5. **Function timeout 20 s** (Go's longest wait, the publish checks).
6. **Both settings set is an error** (not "the function wins"), so a deployment can't silently call a laptop's address.
7. **The Lambda's 6 MB invoke limit is fine:** the largest HQ in the local database (HJE Full HQ, all four sets) is about 36 KB.

## Differences from the brief

- **Not done here (needs AWS and Rahul):** the QA deploy, J11 and J11b on QA, the cold-start and call-time measurements, and the manual test focus.
- The jobs Lambda has the setting but no permission (decision 2); the brief says "the config value for both".

## Questions for Rahul

1. ~~**Node 22 or 24 on Lambda?**~~ *Answered 28 Oct (Rahul): the Lambda on Node 24 now, because AWS drops Node 22 Lambdas around April 2027; the rest moves to 24 before go-live.* Built (decision 3). Note the tests run the bundle on the repo's Node, not 24, until the repo moves; the QA journeys run it on 24.

## Test results (28 Oct)

- `make lint`: passed.
- `make test`: Go all packages ok; Vitest: web 300, `apps/evaluate` 25 (12 new in `lambda.test.ts`, including one that builds the real bundle and calls it), `packages/clinical` 472 (1 new); CDK 130 passed (6 new for the function, its role, the invoke permissions, the setting and the alarm; 1 for `make build-lambdas`; 3 updated counts).
- `make test-integration`: all packages ok.
- `make e2e` against `make dev`: 100 passed, 5 skipped, 1 failed: J7 (chromium) found 3 captured texts for its random mobile (`07700 900` + 3 random digits) from earlier runs in the local database, unrelated to S45; J7 passed on a rerun (6 of 6).
- New Go unit tests: `TestLambdaEvaluate`, `TestLambdaShown`, `TestLambdaPublishProblems` (the shared fixtures through a fake Lambda client), `TestLambdaFailures` (8 ways to fail; no error holds the reply), `TestInfraAgrees` (the marker and the op names), `TestLoadEvaluateFunction`, `TestEvaluator`.
- A local run of the built bundle on the shared fixture: loads in about 35 ms, first call about 70 ms, then about 35 ms (a Mac, not Lambda).

## QA deploy and measurements

29 Oct, with Rahul (he asked about cost first: practically nothing, see below).

1. **Deploy:** `make deploy-qa PROFILE=AdministratorAccess-026090536258` passed (stacks 141 s; Neon migrated 00031 → 00036 and seeded; the web app uploaded). `poised-uk-qa-evaluate` is `nodejs24.x`, arm64, 1024 MB, 20 s, with no environment. `{"op": "health"}` by `aws lambda invoke` answers `{"status":200,"body":{"ok":true,"evaluator":"clinical@655e3ce8a31c-dirty"}}` ("-dirty" because the code isn't committed yet; the pipeline's build after the push names the commit). The three alarms (`delivery-failures`, `jobs-failed`, `evaluate-failures`) are OK and go to `poised-uk-qa-alarms`, whose email subscription is confirmed. Both APIs' start-up lines say `evaluate Lambda poised-uk-qa-evaluate`.
2. **J11 and J11b on QA:** `make e2e-users PROFILE=…`, then `make e2e E2E_BASE_URL=https://main.d33kh0gi8vk83v.amplifyapp.com ARGS="journeys/test_j11_triage.py journeys/test_j11b_high_risk.py"`: **8 passed** in 64 s.
3. **Measurements** (the evaluate log group, during that run):
   | Call | Time in the function |
   | --- | --- |
   | Cold start (`Init Duration`) | 228 ms |
   | A send's `evaluate`, first after the cold start | 347 ms (368 ms billed) |
   | A send's `evaluate`, warm | 147–182 ms |
   | Publish's `publish-problems` | 12 ms cold, then 1 ms |
   | Complete review's `shown` | under 1 ms |

   So the worst case under a lock (a cold start, then a send) is about 0.6 s plus the invoke's own round trip, **under the 1 s that would need Rahul** (S44a answer 12): the calls stay inside the locks; no provisioned concurrency. Memory used: 120–138 MB of 1024 (kept at 1024 for the CPU it brings; cost is per use).
4. **No answers in the logs:** the function's lines are only `evaluate: <op> [form <id>]: <status> in <n> ms` and Lambda's own START/END/REPORT lines.
5. pending.md's four lock items now carry these numbers.
6. **Cost** (Rahul asked): the function and its log group are pay-per-use (inside Lambda's free tier at QA's volume), the alarm is the third of CloudWatch's free 10, so QA stays near £0 when idle. `infra/README.md`'s idle-cost line now says three alarms.

7. **Manual tests:** [test report 1](test-report-1.md), 29 Oct: **all 12 passed** (T-01–T-04, R-01–R-04). Its times agree: 278 ms for the first send after a cold start, then 122–136 ms. Its phone window was 503 px wide (Chrome wouldn't go narrower); J11b's phone-size patient half covers 390 px.
8. **H-1 (Rahul's OK, 29 Oct), by a one-off pytest on QA (deleted, not committed):** the function's reserved concurrency set to 0, so every call is refused (429 after the SDK's 3 attempts). A low-risk send still worked, stayed at Ready for review, with "The automatic checks didn't run. Review this HQ by hand."; Publish was refused with 503 "The checks couldn't run. Try again in a minute."; both APIs logged `EVALUATE_FAILED` with only the form or version id and AWS's error. Mended (`delete-function-concurrency`) in the test's `finally`; the next send was auto-triaged to Ready for admission again. **The alarm `poised-uk-qa-evaluate-failures` went to ALARM at 15:03:15**, about 2 minutes after the failures at 15:01 (its email goes to the alert address, whose subscription is confirmed; Rahul checks his inbox).

Still to do: H-2, the push and a green pipeline run.

## Suggested commits

1. `S45: the evaluate Lambda's handler and bundle` — `apps/evaluate/src/{handler,lambda,lambda.test}.ts`, `apps/evaluate/build.mjs`, `packages/clinical/src/scores{,.test}.ts`.
2. `S45: Go invokes the evaluate Lambda when EVALUATE_FUNCTION is set` — `apps/api/go.{mod,sum}`, `internal/evaluate/{evaluate,lambda,lambda_test}.go`, `internal/config/*`, `internal/serve/{serve,evaluator_test}.go`, `.env.example`.
3. `S45: the evaluate function, its permissions and its alarm in CDK; make build-lambdas bundles it` — `infra/api_stack.py`, `infra/pipeline_stack.py`, `infra/tests/*`, `Makefile`.
4. `S45: docs` — `infra/README.md`, `docs/saas-requirements.md`, `docs/pending.md`, this file.
