# S45 verification

## Verification 1 (29 Oct 2026)

**Verdict: Verified.** The brief is done: the Node 24 arm64 function, Go's Lambda `Evaluator` chosen by `EVALUATE_FUNCTION`, CDK with the invoke rights and the alarm, `make build-lambdas` bundling it, the QA deploy, J11/J11b on QA, the measurements and the docs. Nothing below blocks. **H-2** (a green pipeline after the push) can only happen after the commit, so it stays a hand check.

### Checked independently (not taken from the notes)

- **The deployed function** (`poised-uk-qa-evaluate`): `nodejs24.x`, arm64, 1024 MB, 20 s; **no environment, no VPC, no layers**; its role has only `AWSLambdaBasicExecutionRole`, with no inline policies; **no resource policy** (`GetPolicy` → ResourceNotFound); reserved concurrency back to unset after H-1.
- **Who may invoke it:** `lambda:InvokeFunction` on its ARN appears only in staff-api's and patient-api's roles. jobs has `EVALUATE_FUNCTION` but no Lambda statement at all (builder's decision 2).
- **Its log holds no patient data:** all 364 lines of the last 24 hours, normalised, are only START/END/REPORT/INIT_START, `evaluate: evaluate form <id>`, `shown form <id>`, `publish-problems` and `health`, each with `200 in N ms`. A search for the test names, emails, mobiles, dates of birth and the answer words found nothing (the one hit was a request id containing "1971-").
- **H-1, from AWS's side:** each API logged one `EVALUATE_FAILED` line naming only the form or version id and AWS's 429 (no payload, no message). The alarm went `OK → ALARM` at 15:02:56 and `ALARM → OK` at 15:17:56, to `poised-uk-qa-alarms`, whose email subscription is confirmed. Whether the email reached the inbox is Rahul's to see.
- **J11 and J11b on QA, run by me:** 8 passed in 64 s (real browsers; J11b's patient half at phone size).
- **Browser spot-checks in Claude in Chrome were not done.** Staff sign-in on QA needs Rahul to type the Cognito password and code, and S45 changes no screen. The journeys above, run in real browsers on QA, take their place. The tester covered the screens on 29 Oct.

### Findings (minor, worst first; none blocking)

1. **The evaluate function has no reserved concurrency and shares the account's pool** ([infra/api_stack.py:236](../../../infra/api_stack.py#L236)). Each send now uses two concurrent executions (the API and the function). The QA account's limit is 1000, so this doesn't matter today. Production's account could start at 10 (infra/README's "Reserved concurrency" line). There a burst would throttle the evaluate calls: sends would go to Ready for review unchecked, and Publish would give 503. **Add this to the existing go-live item** (raise the limit, then reserve some for patient-api and for evaluate).
2. **The function's timeout is 20 s for every op, but Go waits only 5 s for a send or a review** ([infra/api_stack.py:251](../../../infra/api_stack.py#L251); `EvaluateTimeout`). A stuck call that Go has given up on keeps running, and is billed, for up to 15 s more. This is harmless at the measured 0.1–0.4 s. One function serves all three ops, so the 20 s needed for publish is the simple choice.
3. **Lambda's 6 MB request limit shows as "Try again in a minute"** ([internal/evaluate/lambda.go:71](../../../apps/api/internal/evaluate/lambda.go#L71)). A version over 6 MB would make every Publish give 503, and retrying would never help. The 20 MB `maxReply` check can't fire on Lambda, whose replies are capped at 6 MB. The largest HQ is about 36 KB (build notes, decision 7), so this is far off. Note it for C8 (files) if images ever get inlined.
4. **`scoreAsShown`'s cache empties itself completely at 500 entries** ([packages/clinical/src/scores.ts:142](../../../packages/clinical/src/scores.ts#L142)). With more than 500 distinct scores in use, it would re-parse on almost every call. Evicting only the oldest entry (`parsed.delete(parsed.keys().next().value)`) would avoid that. It's fine at today's scale.
5. **`EVALUATE_FUNCTION` refuses a qualified name such as `poised-uk-qa-evaluate:live`** ([internal/config/config.go:363](../../../apps/api/internal/config/config.go#L363)). A qualified ARN is accepted. Nothing uses aliases, so this matters only if one is added later.
6. **`make build-lambdas` doesn't empty `apps/api/dist/evaluate/` before bundling** ([Makefile:119](../../../Makefile#L119)), and CDK uploads the whole directory. The Go functions' directories work the same way, and today the directory holds only `index.js`.

Considered and dismissed:
- The jobs Lambda builds a Lambda client it never uses. That's the builder's documented decision 2, and the cost is negligible.
- `LoadDefaultConfig` is called twice on a cold start. That's milliseconds, and only on Lambda.
- The evaluator's version changes with every commit. That's S41's agreed design; on Lambda it only means a fresh deploy per push.

### Scope, tests, security, docs

- **Scope:** everything in the brief is built. There's one documented difference: jobs gets the name but not the permission. Nothing was added outside the brief, apart from the `scoreAsShown` bound that pending.md asked for if S45 kept a warm process.
- **Tests first:** Go `TestLambda*` (the shared fixtures through a fake client; 8 ways to fail), `TestLoadEvaluateFunction`, `TestEvaluator` and `TestInfraAgrees`; Vitest `lambda.test.ts` (12 tests, one of which runs the real bundle); 6 new CDK tests for the function, its role, the invoke rights, the setting and the alarm. No new routes, so no new five-case route tests are due.
- **Tenant isolation:** unchanged; the function has no database.
- **A-20 grep:** only old comments (`format.ts`, `patient.go`); nothing from S45. The function's name comes only from the country entry.
- **Security:** no secrets in the repo or in the function's environment. No answers in the function's log, the APIs' failure lines or Go's errors (checked on QA above, and by the tests). The patient router is unchanged.
- **Docs:** infra/README, saas-requirements §13, decisions.md (28 and 29 Oct), pending.md, the C6 plan's Status, plans/README and sessions/README were all updated. `repocheck` passes (in `make test`).

### Test results (29 Oct, this verification)

- `make lint`: passed.
- `make test`: Go, all packages ok; Vitest web **300 passed**, apps/evaluate **25 passed**, packages/clinical **472 passed**; CDK **130 passed**.
- `make test-integration`: all packages ok.
- `make e2e` against QA, J11 + J11b: **8 passed** in 63.6 s.
- Local `make e2e` was not re-run (`make dev` wasn't running). The builder's run on 28 Oct: 100 passed, plus J7's known flake, which passed on a rerun. The local service's change is a refactor (`handle` → `run`), covered by `apps/evaluate`'s Vitest.

### Suggested commits

1. `S45: the evaluate Lambda's handler and bundle`: `apps/evaluate/src/{handler,lambda,lambda.test}.ts`, `apps/evaluate/build.mjs`, `packages/clinical/src/scores{,.test}.ts`
2. `S45: Go invokes the evaluate Lambda when EVALUATE_FUNCTION is set`: `apps/api/go.{mod,sum}`, `apps/api/internal/evaluate/{evaluate,lambda,lambda_test}.go`, `apps/api/internal/config/{config,config_test,resolve_test}.go`, `apps/api/internal/serve/{serve,evaluator_test}.go`, `.env.example`
3. `S45: the evaluate function, its permissions and its alarm in CDK; make build-lambdas bundles it`: `infra/api_stack.py`, `infra/pipeline_stack.py`, `infra/tests/*`, `Makefile`
4. `S45: docs`: `infra/README.md`, `docs/saas-requirements.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plans/{README,c6-scores-asa-triage}.md`, `docs/sessions/README.md`, `docs/sessions/S45-evaluate-lambda/*`

`docs/market/` is untracked and not S45's work, so it's left out.

### Hand checks for Rahul

- **H-1's email:** check that "ALARM: poised-uk-qa-evaluate-failures" (about 15:03) and "OK: …" (about 15:18) reached the alert inbox. AWS shows both state changes.
- **H-2:** after the push, CodePipeline `poised` runs green, journeys included, and the function's `{"op":"health"}` names the commit without `-dirty`.
