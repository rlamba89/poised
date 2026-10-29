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

**Done 29 Oct 2026 (S45 verified, verification 1; H-2, a green pipeline after the push, still to do). Split into sessions on 26 Oct 2026: S38 verified (26 Oct, verification 3, after verifications 1–2's fixes); S39 verified (26 Oct, verification 2); S40 verified (26 Oct, verification 2, after verification 1's fixes); S41 verified (27 Oct, verification 1); S42 verified (27 Oct, verification 1); S43 verified (27 Oct, verification 2, after verification 1's fixes); S44 verified (27 Oct, verification 1); S44a verified (27 Oct, verification 3, after verifications 1–2's fixes); S44b verified (28 Oct, verification 2, after verification 1's fixes; for verification 2: the anaesthetist's ASA grade as a history; no differences from its brief, the builder's choices are in its build notes); S44c added 28 Oct (C6's last open items, Rahul's answers of 28 Oct; tested 28 Oct; verification 1 needed fixes (F1, F2), done; test report 3 all passed; verification 2 needed one fix (F3), done; test report 4 all passed; verification 3 needed fixes (F4, F5, Rahul's answer 8), done; test report 5 all passed; verified 28 Oct, verification 4), then S45 (AWS phase, with Rahul; code built 28 Oct; deployed to QA by hand and measured 29 Oct; tested 29 Oct, test report 1 all passed, hand check H-1 done; verified 29 Oct, verification 1).** S39 differs from this plan: there's no `codes.asa`; the ASA guide (the ASA's own examples per code, a Go file) gives a code one or several grades, and the author picks where it depends on severity. The briefs are in [docs/sessions/](../sessions/README.md); Rahul answered their "Ask Rahul first" questions on 26 Oct. S38 also built the ready-made STOP-Bang (his answer to its question 2) and the Incomplete state.
- **S38:** scores that use answers from earlier Question Sets, and points for age, sex and BMI (SCR-01/02; most of SCR-01 was already built);
- **S39:** each code's usual ASA grade, and the suggested ASA for the whole HQ with its reasons (ASA-01);
- **S40:** the rules on the HQ: defaults, the Rules page, preview and test cases (ASA-02/03/04); J11 starts. *Differs from this plan:* rules are in their own column (`questionnaire_versions.rules`, with whole-HQ test cases; Rahul's answer 4), not "the version JSON"; the defaults are a Go file;
- **S41:** the `evaluate` service run locally (`npm run evaluate`, behind Go's `Evaluator` interface with a fake in tests); on send Go stores the result in `form_outputs`; it fails safe; publish re-checks through it. *Differs:* publish's refusal is the existing 422 with the problems (not 400); `form_outputs` keeps a row per send; the service is `apps/evaluate` (Node `node:http` on 127.0.0.1, esbuild), the functions `packages/clinical/src/evaluate.ts`;
- **S42:** auto-triage on send, the Auto-triaged worklist; J11. *Differs:* the auto-triage move goes from Ready for review (after the Submit event) to the target, as a system-only move outside the role table; a per-HQ `messageAfterCheck` setting (Rahul's answer 2: the message goes at once by default); the guards (no red or amber flag, no Incomplete score, no anaesthetist's ASA required) can't be switched off; a re-open ends the auto-triage;
- **S43:** rule-made tasks, flags, the anaesthetist's ASA requirement, suggested statuses; J11b;
- **S44:** the server refuses an incomplete send and an unchecked Complete review (two pending.md gaps). *Differs:* Complete review asks the service a third question, `POST /shown` (which sets are shown for the answers that count now), since `/evaluate` counts only the patient's answers; the send's check runs after the freeze in the same transaction and a refusal rolls it back (so nothing is frozen and the answers checked are the ones that would be);
- **S44a (added 27 Oct):** the follow-ups Rahul decided on 27 Oct (hidden score inputs add 0, a send keeps only what the form showed, Publish refuses patient conditions on clinician-only content, the auto-triage warning, authors' "also when" bands) and the four defects the verifiers found;
- **S44b (added 27 Oct):** the anaesthetist's ASA grade as a history (never changed or cleared);
- **S45 (AWS phase):** the Lambda, its permissions and its alarm on QA.

Proposed in the briefs, differing from or adding to this plan (for Rahul to confirm when the sessions start):
- **Rules are stored in a new `questionnaire_versions.rules` column**, since a version has no JSON of its own; Go holds the platform defaults as a file and checks only the shape.
- **Go's publish asks the evaluator for every browser publish check** (not only test cases), and refuses publishing while the evaluator can't be reached (S41).
- **The evaluator also runs on Complete review**, only to check that every shown set is validated (S44, as §10 of saas-requirements says).
- **The validated score library (SCR-04) moves to C7** if Rahul agrees (S38 question 2).
