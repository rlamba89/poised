# C6: Scores, ASA and auto-triage

**Goal:**
- authors build risk scores with bands into an HQ;
- disclosures carry ASA contributions;
- each HQ carries ASA rules;
- when a patient submits, the server works out the outputs and auto-triages low-risk patients.

**Requirements:** SCR-01/02/04, ASA-01…05, TSK-01 (rule-created tasks), D-1, A-8, §10. **Depends on:** C3, and C4 (score inputs by key).

## Design

- **Scores and bands (`packages/clinical`, written TDD):**
  - options carry `score`;
  - a Calculation totals the scores;
  - bands carry disclosures, and `band('<calc>')` works in conditions;
  - inputs: age and sex (profile), BMI (HQ or observations), and other answers by key.
- **ASA:**
  - `codes.asa` holds the platform default per code. An author can override it on a disclosure.
  - Suggested ASA = the highest across the episode's disclosures, with the reasons.
- **ASA rules:**
  - stored in the version JSON, e.g. `{rules:[{when:{asa:">=3"}, then:["task:Book anaesthetic review","flag:anaesthetic_review","require:anaesthetist_asa"]}]}`;
  - the platform default rules are copied into every new HQ;
  - edited in a "Rules" tab in the HQ settings (super clinicians);
  - **rules are evaluated in TypeScript**, next to the outputs, so preview, test cases and the server always agree;
  - test cases can expect a suggested ASA and the rules that fire (ASA-04).
- **The `evaluate` Lambda (D-1):**
  - Node, built from `packages/clinical` with esbuild. Input: published JSON + answers + patient context. Output: disclosures, scores, suggested ASA, fired rules.
  - Added in CDK.
  - Go calls it through `Deps.Evaluator` (a fake in tests). Locally, `npm run evaluate` serves the same handler over HTTP.
- **On submit:** Go calls evaluate, stores the result (`form_outputs`), then applies the actions:
  - **auto-triage** (status → the HQ's target, default Ready for admission, with an event naming the rule);
  - tasks, flags (`episode_flags`), and requirements (e.g. anaesthetist ASA before Ready for admission).
  - **If evaluate fails, it's safe:** no auto-triage, the status goes to Ready for review, and an error alert fires.
- **Worklist:** "Auto-triaged", for the quick check. Add a hazard-log entry (OPS-04).

## Tests

- **Unit:**
  - in TypeScript: bands, ASA max with reasons, the rule matcher, the evaluate handler;
  - in Go: applying actions from a given result.
- **Integration (with a fake evaluator):**
  - low ASA → Ready for admission, plus an event;
  - high → task + flag;
  - "require anaesthetist ASA" blocks Ready for admission until it's set;
  - evaluator failure → Ready for review, and no auto-triage;
  - publishing is blocked when a test case's expected ASA doesn't match.
- **End-to-end:**
  - **J11:** a low-risk patient submits → appears on the Auto-triaged worklist as Ready for admission.
  - **J11b:** a high-risk patient → a "Book anaesthetic review" task and a flag.

## Status

Not started.
