# S45: The evaluate Lambda on AWS

*AWS phase (12 Oct): runs with Rahul there. Never in an automatic run unless Rahul names it.*

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 8 of 8. C6 is done after this session.
**Requirements:** D-1, D-4 ([saas-requirements.md](../../saas-requirements.md) §0, §10, §13); OPS-05's alarm pattern.
**Depends on:** S44 verified. Rahul there, with the AWS profile (`PROFILE=…`).

**What exists today:**
- QA on AWS (S06, S07): `infra/api_stack.py` builds the three Go Lambdas (`FUNCTIONS`, from `make build-lambdas`), with the per-country values from `infra/countries.py`; the pipeline (`infra/pipeline_stack.py`) runs `make test`, builds, deploys QA and runs the e2e journeys on it.
- The delivery-failure alarm: a metric filter on log lines starting `DELIVERY_FAILED`, an alarm, and an SNS topic emailing the address in SSM `/sj/alert-email` (`infra/api_stack.py`, `infra/budget.py`).
- After S41–S44: Go's `internal/evaluate` has the `Evaluator` interface, an HTTP client for the local service (`EVALUATE_URL`) and a fake; the handler is a pure function in `packages/clinical`, served locally by a small Node program; failures log `EVALUATE_FAILED`.

**Ask Rahul first:** nothing. The plan settles the Lambda (D-1, D-4, Node 22 or newer); the alarm follows the existing delivery-failure alarm and its address.

## Build

- **The Lambda:** a Node 22+ (arm64) function `evaluate`, its code bundled by esbuild from the same handler (`make build-lambdas` builds it too). No database, no network access it doesn't need, no environment secrets. Its log never holds answers.
- **Go:** a Lambda-invoke `Evaluator` (AWS SDK `lambda:InvokeFunction`), chosen by config (`EVALUATE_FUNCTION` in `internal/config`; `EVALUATE_URL` stays for local). `CONFIG_STRICT` requires one of them. Unit test with a fake Lambda client, using S41's shared fixture.
- **CDK (Python, `infra/`):** the function, with its own role; `staff-api` and `patient-api` may invoke it (and nothing else may); the config value for both; the `EVALUATE_FAILED` metric filter and alarm on both APIs' logs, to the existing alert topic. CDK tests (`infra/tests`) for the function, the permissions and the alarm.
- **Pipeline:** the build step bundles it; nothing else changes.
- **On QA:** deploy (by hand first, then the pipeline at the next push); J11 and J11b pass on QA. Measure the cold start and the call time; note them in the build notes.
- **Docs:** `infra/README.md` (the new function and alarm), saas-requirements §13 if anything differs.

## Not in this session

- Training and Production stages (C10).
- Calling the evaluator from anything but the two APIs.

## Manual test focus

1. On QA, a low-risk patient's send is auto-triaged; a high-risk one gets the task and flag.
2. Break the function on purpose (e.g. remove the invoke permission in a test deploy): sends go to Ready for review with the note, and the alarm email arrives.
3. The function's logs hold no answers, names or dates of birth.
4. The pipeline's run after the push is green, journeys included.
