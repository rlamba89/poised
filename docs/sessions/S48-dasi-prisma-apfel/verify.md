# S48 verification: DASI, PRISMA-7 and Apfel

## Verification 1 (6 Nov 2026)

**Verdict: Verified.** Everything the brief asked for is built, including all nine carry-overs from S47's verifications. Every automated suite passes, test report 1 passed all 17 cases, and my own spot checks passed. The findings below are minor or need a decision from Rahul. None of them blocks the commit.

### Rahul's answers (7 Nov)

- **Finding 1:** refuse a mismatch, as a fix commit on top of S48, before S49. **Built 7 Nov, test-first:**
  - `assessment.ProfileProblem` (unit test `TestProfileProblem`), called by `storeAssessment` on every save and Confirm, with the age worked out today in the hospital's time zone (as the browser does). The reply is a 400: "Sex doesn't match the patient's record. Reload the page and try again." or "Age doesn't match the patient's record (63 today). …"
  - Integration case: "an age or sex that isn't the record's is 400, and nothing is stored".
  - The assessment tests now give their patient a record that backs what they send (`asmEpisode`, `recordAs`).
- **Finding 2:** each form keeps its own version's name. Carried into S49's brief.
- **Commit:** the five suggested commits, local only.

### What I checked

- **Scope against the brief:**
  - DASI, PRISMA-7 and Apfel are locked ready-made scores (`ready.ts`), each with its `source` and `licence`.
  - Each is also an assessment from the same definition (`assessments.ts` `DEFS`).
  - The editor's Insert offers all four, and `readyScoreProblems` locks each one.
  - The POA Summary shows each result, with DASI's METs.
  - The S47 carry-overs are done:
    - `list.json`;
    - the observation source on number items only;
    - `poaLines` memoised;
    - the pre-fill worked out once;
    - the two `dollar_1` queries;
    - matching-version reuse;
    - `Unanswered` covering the age and sex;
    - `labels` in the content;
    - `currentAnswers` using `countingHqAnswers`;
    - the name upsert.
  - Nothing large was added outside the brief.
- **Scoring against the sources:**
  - **DASI:** the 12 weights total 58.2, and METs = (0.43 × DASI + 9.6) / 3.5. I checked 0 → 2.7, 34 → 6.9 and 58.2 → 9.9 by hand.
  - **PRISMA-7:** one point per Yes, plus 1 for over 85 and 1 for male.
  - **Apfel:** female, non-smoker, PONV or motion sickness, and opioids, at 10, 21, 39, 61 and 79%.
  - PRISMA-7 scoring question 6's Yes is the builder's decision, recorded in decisions.md and in pending.md for a clinician to check.
- **Tests first:** S48 adds no route. The existing assessment routes keep their five cases, and the new integration cases cover:
  - adding all four;
  - reuse of a matching version;
  - a rename;
  - the list in order;
  - Apfel's sources;
  - Confirm of each, including the 400s for a missing age and for missing opioids;
  - an age Apfel doesn't name;
  - an observation on a Yes/No item.

  `ready.test.ts` checks every band's edges. J12 has the phone journey.
- **Tenant isolation:** S48 adds no new query on patient data. The two queries changed to `sqlc.arg(org_id)` keep their scoping.
- **A-20:** the grep finds only older comments and test or config files. Nothing in S48's files.
- **Security:**
  - No secrets in the change.
  - No answers are logged.
  - Clinician-only content is still stripped (spot check 2).
  - The patient router is unchanged.
- **Docs:**
  - The plan's status line, `plans/README.md`, `sessions/README.md`, `decisions.md` (the answers and the builder's choices) and `pending.md` (licences, wording checks, the tester's findings) are all updated.
  - `repocheck` passes.

### Findings (worst first)

1. **Decision for Rahul: Go accepts a `profile` age or sex that contradicts the patient's record** (test report 1, finding 1; `apps/api/internal/assessment/assessment.go` `Check`).
   - A hand-made Confirm with `patientSex: "male"` for a patient recorded as Other gave a confirmed PRISMA-7 of 4, shown as from "Profile".
   - It takes a signed-in clinician sending a crafted request, and Go has the date of birth, sex and time zone it would need to compare.
   - *Recommended:* refuse a mismatch with a 400 in a small follow-up (S49 or a fix commit). It's already in pending.md.
2. **A rename relabels every existing form, confirmed ones included** (`apps/api/db/queries/assessments.sql:26`, read by `ListAssessmentForms` at `:57` as `q.name`).
   - The brief asked for the name to be updated, so this is as asked. The effect is that an old episode's card and POA line take the new name too.
   - If each form should keep its version's name, read `c.name` (the chapter name `PublishAssessmentVersion` stores) instead. Low risk: no rename is planned.
3. **A matching retired version is reused, but its status isn't changed back** (`apps/api/internal/httpapi/assessments.go:142`).
   - After a rollback, new forms go onto a version marked retired while a version no build serves stays "published".
   - This is the builder's decision 11, and harmless today, because nothing reads an assessment's status to choose a version. Revisit if reports ever do.
4. **STOP-Bang forms on version 1 (from before S48) get the long-title fallback in Go's messages** (`assessment.go:159` `label`): "BMI (kg/m²): a number…", and the neck question's first 40 characters. The build notes acknowledge it. Cosmetic.
5. **Apfel's score has no `clinicianOnly`** (`packages/clinical/src/ready.ts:299`), unlike STOP-Bang's, DASI's and PRISMA-7's scores.
   - It has no effect today: `apfelClinicianPart` is used only in the assessment's content, never in an HQ.
   - Add `clinicianOnly: true` for consistency, in case it's ever reused.
6. **`itemsOf` guesses the definition from the content's keys** (`packages/clinical/src/assessments.ts:140`) rather than using the row's `assessment` id.
   - Correct for the four today, but a future variant whose keys are a subset of another's would match the wrong definition.
   - Passing the id would remove the guess.
7. **Two caches depend on object identity** (web `assessments.ts:93` `hqParts` keeps `from.hq` stable so that clinical's `WeakMap` can hit).
   - Correct as written, and tested ("is worked out once").
   - A future spread of `from.hq` would silently bring back the per-keystroke cost.
8. **A blank line was left in JSX** where the sex note was removed (`apps/web/src/features/episodes/AssessmentsPanel.tsx:189`). Trivial.
9. **Cosmetic, from test report 1 (already in pending.md):**
   - the badge's capitals turn "METs" into "METS" (I saw it too, at 542 px wide);
   - the POA Summary's assessment values sit about 25 px right of the column;
   - "answer the item highlighted" appears when the highlighted item is the record's sex.

### Spot checks (Claude in Chrome, `make dev`, 6 Nov)

1. **Another hospital or trust (API):**
   - The test episode was the tester's Ann (`3fbe69dc…`, HQ sent, DASI and Apfel confirmed).
   - Cara gets 200, with four assessments, `available` empty and each `content.labels` present.
   - Val (A2), Bea (Trust B) and Max Multi (A2 clinician, Trust B admin) each get **404**:
     - on Hospital A1's path;
     - on each of their own hospitals' paths, for the list, an add (POST) and the episode itself.
2. **What the patient receives:**
   - I ran Go's `chapter.ForPatient` on the seeded Demo fitness HQ, using a throwaway test that I deleted afterwards.
   - Removed: `dasi_score`, `dasi_mets`, `prisma7_score`, `apfel_*` opioids and score, "METs" and "Poor functional".
   - Kept: the patient's questions.
3. **The cards on screen** (Ann's episode, 542 px wide, `scrollWidth` 542, so no sideways scroll):
   - DASI "39.25, 7.6 METS", confirmed, 12 of 12, sources HQ and Clinician (after the tester's T-09).
   - PRISMA-7 "1 (Not at increased risk)", items 1 and 2 from Profile, 3–7 from the HQ.
   - Apfel "4 (PONV risk about 79%)", item 4 from Clinician, confirmed.
   - STOP-Bang (version 2) Incomplete, "2 of 8 filled".
4. **Go's messages from `labels`.** These were refused hand-made requests, and the revisions were unchanged afterwards:
   - STOP-Bang Confirm without the neck: 400 "Answer every item before confirming: Neck has no answer."
   - BMI 300: 400 "BMI: a number from 5 to 150."
   - An observation source on PRISMA-7's Yes/No item: 400 "Say where each answer came from."
   - PRISMA-7 Confirm without the sex: 400 "…Sex has no answer."
5. **The POA Summary:** under Hospital BMI it shows "DASI 39.25, 7.6 METs · PRISMA-7 Not confirmed · Apfel PONV 4 (PONV risk about 79%) · STOP-Bang Not confirmed".

The test browser was left signed in as Cara (it had been Alex Author).

### Test results

- `make lint`: pass.
- `make test`:
  - Go: every package ok, including `repocheck`;
  - web: 378 passed;
  - evaluate: 25 passed;
  - clinical: 562 passed;
  - CDK: 132 passed.
- `make test-integration`: every package ok, including `internal/httpapi` (37 s). The first attempt failed only because `docker` wasn't on this shell's PATH. Rerun with Docker.app's bin on the PATH, it passed.
- `make e2e` (against Rahul's `make dev`): **116 passed, 1 failed, 5 skipped** (the Cognito journeys, AWS only) in 11 min 24 s.
  - J12, with S48's new DASI phone journey, passed.
  - The one failure was **J7** (patient invites): "expected 1 captured text, got 3". This is the known flake (pending.md:127): `made_up_mobile()` picks from 1,000 numbers, and the dev database has more than 3,000 episodes.
  - J7 hasn't changed since S14. **Rerun alone, it passed (6 of 6).** Not S48's.

### Hand checks for Rahul

- None needed. The tester ran the phone cases in a 390×844 iframe, because Chrome wouldn't resize. A glance on a real phone at the Demo fitness HQ is still worthwhile.

### Suggested commits

1. **S48: DASI, PRISMA-7 and Apfel as ready-made scores; the editor's insert**
   - Files:
     - `packages/clinical/src/{ready.ts,ready.test.ts,questionTypes.ts,scores.ts,rules.ts}`;
     - `apps/web/src/features/editor/{ReadyScoreWindow.tsx,QuestionnaireEditor.tsx}`.
2. **S48: the three as assessments; labels and list.json for Go; the pre-fill worked out once**
   - Files:
     - `packages/clinical/src/{assessments.ts,assessments.test.ts}`;
     - `packages/clinical/scripts/assessments.mjs`;
     - `apps/api/internal/assessment/{stopbang,dasi,prisma7,apfel,list}.json`;
     - `apps/web/src/features/episodes/{assessments.ts,assessments.test.ts,AssessmentsPanel.tsx,PoaSummary.tsx,hq.ts}`.
3. **S48: Go: the list from list.json, labels, the checks; version reuse, the name, the org_id arguments**
   - Files:
     - `apps/api/internal/assessment/{assessment.go,assessment_test.go}`;
     - `apps/api/internal/httpapi/{assessments.go,assessments_integration_test.go}`;
     - `apps/api/db/queries/{assessments,admin,chapters}.sql`;
     - `apps/api/internal/db/*`.
4. **S48: Demo fitness HQ; J12**
   - Files:
     - `apps/api/cmd/seed/{main.go,keys_test.go,demo-fitness-hq.json}`;
     - `e2e/data/fitness-hq.json`;
     - `e2e/journeys/test_j12_observations.py`.
5. **S48: docs**
   - Files:
     - `docs/{decisions.md,pending.md}`;
     - `docs/plans/{README.md,c7-observations-assessments.md}`;
     - `docs/sessions/{README.md,S48-dasi-prisma-apfel/*}`;
     - `CLAUDE.md`;
     - `apps/web/AGENTS.md`.

Commits 1–3 depend on each other: Go's tests read the JSON that commit 2 writes, and commit 2's code uses commit 1's. Each builds with the ones before it, in this order.
