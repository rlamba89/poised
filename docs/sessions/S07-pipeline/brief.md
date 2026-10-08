# S07: The pipeline and the "add a country" runbook

**Plan:** F4 ([f4-aws-pipeline.md](../../plans/f4-aws-pipeline.md)), Steps 2–4. This is the last part of F4.
**Depends on:** S06 verified.

**Ask Rahul first:**
- Do the *Proposed* one-time proof (F4 Step 3)?
  - That means deploying a throwaway QA copy in a second region (e.g. `us-east-1` with Neon), running J1–J5 on it, timing it, then destroying it.
  - **Recommended: yes.** It's the evidence that a new country is quick, and it costs pennies.

## Build

- **`PipelineStack`,** with CDK Pipelines, from GitHub `main`:
  - **synth step:** `make lint test`, Postgres in Docker, `make test-integration`, the CDK test, the builds, then `cdk synth`;
  - **QA stage post-deploy steps:** migrate, the idempotent seed (with a run-twice test), the web deploy, then `make e2e` against QA, keeping the traces.
- **The runbook in `infra/README.md`:** "add a country", plus how to deploy, roll back and read a failed run.
- **If agreed:** the second-region proof, with the time recorded in F4 Status.
- **The idle-cost check:** schedule it for a week later (Rahul reads Cost Explorer), and leave a note in F4 Status.

## Not in this session

- Training and Production stages (C10).

## Done when

- A push to `main` goes green all the way through QA e2e, with no manual step.
- A deliberately failing unit test on a branch pipeline stops the run before deploy.

## Manual test focus

The tester works in the **AWS console** in the browser, and Rahul does the pushes:
1. Rahul pushes a trivial visible change. The tester watches every stage go green, then sees the change on QA.
2. The e2e traces are kept as an artifact.
3. A failing test stops the deploy.
4. The tester reads the runbook as a newcomer, and reports unclear steps.
5. If the proof was done: the second region worked, and was destroyed.
