# S47 verification

## Verification 4 (6 Nov 2026)

**Verdict: Verified.** Rahul's answers 5–8 are built as he chose. Report 4's F-1 is fixed, and I checked it end to end in the browser on the Save path, which report 5 didn't use. Every suite passes. Nothing blocks. The minor points below, and verification 3's five, which hadn't been filed, are now in the S48 brief and pending.md.

### What this round covered

These are the changes since verification 3, and what I checked:
- **Answer 5: the pre-fill uses a set's clinician copy once started.** `countingHqAnswers` ([clinical assessments.ts:101](../../../packages/clinical/src/assessments.ts#L101)) is now the same rule as `currentAnswers` in `hq.ts`. It has Vitest cases (not started, started, validated), and J12 and report 4's T-19 cover it.
- **Answer 6: a confirmed assessment says what's changed since.** `sinceConfirmed` ([assessments.ts:188](../../../apps/web/src/features/episodes/assessments.ts#L188)) compares against a fresh pre-fill with the record as of the Confirm day. It's shown only while the card is confirmed and unchanged. It has 7 web cases, and J12 and T-20 cover it.
- **Answer 7: age and sex frozen at Confirm.**
  - `assessmentAnswers` stores `patientAge` and `patientSex` (source `profile`). `assessmentItems` and `assessmentResult` prefer the stored values on a confirmed row, so the card and `poaLines` both use them. An older row falls back to the record on the Confirm day.
  - Go's `Check` ([assessment.go:143](../../../apps/api/internal/assessment/assessment.go#L143)) takes `profile` only on those two items, and only where the content names `{patientAge}` / `{patientSex}`. The age must be a whole number from 0 to 150, and the sex male or female.
  - It's tested by `TestCheck`'s new cases and an integration case on both routes.
- **F-1 (report 4): once changed, A and G follow the record.** `changedItems` ([assessments.ts:124](../../../apps/web/src/features/episodes/assessments.ts#L124)) does it, with a web test of the save body. Report 5 passed it on Confirm.
- **Answer 8: a record-only change can't be cleared without changing another item.** Left as built, as Rahul chose. The button is in pending.md.

### Findings, worst first

None block. All are minor, from `/code-review high` on the whole working tree, weighed and checked against the code.

1. **Go's Confirm doesn't require the age and sex** ([assessments.go:213](../../../apps/api/internal/httpapi/assessments.go#L213)).
   - `Unanswered` walks only the content's questions. A hand-made Confirm without `patientAge` / `patientSex` is accepted, and its score then falls back to the record on the Confirm day, as a pre-answers-7 row's does.
   - The browser always sends them, because Confirm needs every item, G included. The key rule trusts the clinician's browser.
   - **Filed in the S48 brief:** have `Unanswered` cover the profile items the content names. Its `null` test can't happen (`Check` refuses `null` first), so drop it.
2. **The age a Confirm stores is the browser's "today" when the card's items were last built,** not at the click ([AssessmentsPanel.tsx:94](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L94)).
   - A card left open across midnight before a 51st birthday stores 50 against a Confirm dated the birthday.
   - It's verification 2's minor 4, now on the stored value. The next load's yellow box says so ("…an age of 51 that day (50 when confirmed)"), so it isn't silent.
   - **pending.md's entry is updated** (it said no score input was stored).
3. **Efficiency:** once a card is changed, every keystroke runs the pre-fill twice (`changedItems` → `assessmentItems` ×2), and each run builds a SurveyJS model per shown HQ set. **Filed in the S48 brief with the memo work.**
4. **`countingHqAnswers` and `hq.ts`'s `currentAnswers` are now the same rule in two places.** **S48 brief:** have `hq.ts` use the clinical one.
5. **`label()` names STOP-Bang's keys in Go** ([assessment.go:127](../../../apps/api/internal/assessment/assessment.go#L127)), and **a renamed assessment keeps its old questionnaire name** (`AssessmentQuestionnaire`'s `ON CONFLICT DO NOTHING`). Both are in the **S48 brief**.

**Verification 3's minors, now filed:** the builder hadn't filed them.
- 1 (`dollar_1`), 2 (`publishedAssessment` and the latest version) and 5 (keystrokes) are in the S48 brief.
- 3 (`updated_by` NULL) and 4 (a `null` BMI's message) are in pending.md.

**Weighed and accepted (no change):** `useAssessments` copies `useObservations`, `assessmentsOn` copies `observationsOn`, and `lockedObservations` wraps `lockedEpisode`. These were accepted in verifications 1–3; a shared helper can come when a third feature flag does.

### Checklist

- **Scope:** answers 5–8 and F-1, nothing else. There's no new route, query or migration since verification 3.
- **Tests first:** the build notes record each new case red first. These are clinical (`countingHqAnswers`, frozen age and sex, the old-row fallback), web (`sinceConfirmed` ×7, `changedItems`' save body, `poaLines` against a corrected record), Go (`TestCheck` ×8, and the integration case on both routes) and J12 (`test_a_confirmed_stop_bang_says_what_has_changed_since`). The five cases per route are unchanged and pass.
- **Tenant isolation:** no query changed since verification 3. I re-checked by API (below).
- **A-20:** the grep finds only the earlier comments in `lib/format.ts` and `patient.go`.
- **Security:**
  - No secrets.
  - The 400s name the item ("Age", "Sex"), never a value.
  - The assessments stay staff-only (the patient's path is a 404).
- **Docs:**
  - decisions.md has "S47 verification 3: Rahul's answers" with the builder's choices, and answers.md has 5–8.
  - CLAUDE.md and apps/web/AGENTS.md describe the stored age and sex.
  - The C7 plan, plans/README.md and sessions/README.md are updated (sessions/README.md now says Verified).
  - pending.md and the S48 brief hold the minors.
  - `repocheck` passes.

### Spot-checks in the browser (Chrome, as Cara, Hospital A1; local `make dev`, the API started at 13:06 after the last Go change at 12:45)

I used report 5's made-up patients: Tom "MT S47R5 1436 Two" and Rex "MT S47R5 1432 One".

| Check | Result |
| --- | --- |
| **Answer 7 and F-1 on the Save path:** confirm Tom's STOP-Bang (male, 68), change his record to female and 1 Mar 1990 (`psql`), reload, type B 36, press **Save** (not Confirm) | • After the Confirm, revision 2 stored `patientAge 68`, `patientSex "male"`.<br>• After the reload, the header says "1 March 1990 (age 36)". The card keeps 68 / Male, **4 (High risk)** and "Confirmed by Cara Clinician…". The yellow box has the age and sex lines and "Once any item is changed, the age and sex are the record's."<br>• With B 36: A **36**, G **Female**, B "Entered by clinician", **3 (High risk)**, "Changed: confirm it again", "The age is the patient's today.", and the box gone.<br>• Save: "Not confirmed". Revision 3 stores `patientAge 36` / `profile` and `patientSex "female"`. The POA Summary says **STOP-Bang Not confirmed**.<br>**Pass.** |
| **A confirmed score against a later record change (POA Summary):** Rex (confirmed in report 5 with 36 / female stored), his record put back to male, 1 Mar 1960 | The POA Summary still says **STOP-Bang 5 (High risk)**, under "Hospital BMI 37". From the record it would be 7. **Pass.** |
| **Go's profile checks (hand-made API bodies on Tom's row)** | `profile` on Snoring, and `clinician` on the age: 400 "Say where each answer came from."<br>Age 50.5 and age 200: 400 "Age: a number from 0 to 150."<br>Sex "other" (Confirm): 400 "Sex: that answer isn't one this assessment takes."<br>Age "36" as text (Confirm): 400.<br>The revision stayed at 3. **Pass.** |
| **Other hospitals and trusts (API, curl)** | Through A1's path, list and confirm: Bea (Trust B), Val (A2) and Max (A2, and Trust B admin) each get 404 "Hospital not found.". Through each one's own hospitals: 404 "Episode not found.". With no sign-in: 401. The patient's `/api/p/episodes/{eid}/assessments`: 404. **Pass.** |

Tom's and Rex's episodes keep these changes (made-up test data, local only).

### Test results (6 Nov, on the working tree)

- `make lint test`: **pass**.
  - packages/clinical 518, web 374, evaluate 25, CDK 132;
  - every Go package ok.
- `make test-integration`: **pass**. Every package ok; `internal/httpapi` 32.5 s. Docker's bin was added to PATH, as before.
- `make e2e` (against `make dev`, after `DELETE /api/qa/rate-limits`): **116 passed, 5 skipped** (the Cognito journeys, AWS only) in 12 min 14 s, J12 included.

### Suggested commits

These are verification 1's, with verification 3's changes. Answers 5–8 and F-1 go in their plan step's commit.
1. **S47: assessments as platform questionnaires; the add, save and Confirm routes; observations.precedence.**
   - Migration 00038 and the queries (`assessments.sql`, `questionnaires.sql`, `chapters.sql`, `episodes.sql`, `admin.sql`) with the regenerated `internal/db`;
   - `internal/assessment` (`Check`'s profile items with `TestCheck`; `stopbang.json` goes in commit 2);
   - `httpapi/assessments.go`, `episodes.go`, `observations.go` (`lockedEpisode`), the router, the audit labels, and settings with their tests;
   - `assessments_integration_test.go` and `apitest/migrations_integration_test.go`.
2. **S47: STOP-Bang's pre-fill and score in packages/clinical; the JSON for Go.** `assessments.ts` (with answers 5 and 7), `assessments.test.ts`, `index.ts`, `package.json`, `package-lock.json`, `scripts/assessments.mjs` and `apps/api/internal/assessment/stopbang.json`.
3. **S47: the Assessments card, the POA Summary's line, the demo HQs; J12.**
   - The web app: `AssessmentsPanel.tsx`, `assessments.ts` (`sinceConfirmed`, `changedItems`) and its tests, `EpisodePage.tsx`, `ObservationsPanel.tsx`, `PoaSummary.tsx`;
   - the seed: `cmd/seed/main.go`, `demo-sleep-hq.json`, `demo-stopbang-hq.json` and `keys_test.go`;
   - e2e: `e2e/data/*.json`, `test_j12_observations.py`, `support.py` and `e2e/README.md`.
4. **S47: docs.**
   - decisions.md, pending.md, the plans and the sessions README;
   - the S47 folder and the S48 brief;
   - CLAUDE.md and apps/web/AGENTS.md.

`docs/market/` is untracked and isn't part of S47. Leave it out of these commits.

## Verification 3 (6 Nov 2026)

**Verdict: Verified as built, but back to the builder for Rahul's three answers (below).** Rahul asked for them in S47 before it's committed. After they're built, a re-verification (verification 4) covers only those changes.

### Rahul's answers (6 Nov, after this verification): to build into S47, test-first

Recorded in [decisions.md](../../decisions.md) (6 Nov, "S47 verification 3: Rahul's answers") and [answers.md](answers.md) (answers 5–7).

1. **The pre-fill uses the clinician's copy of a set as soon as they've started it,** like the rest of the episode page (`currentAnswers` in `apps/web/src/features/episodes/hq.ts`). Change `countingHqAnswers` ([clinical assessments.ts:98](../../../packages/clinical/src/assessments.ts#L98)), with Vitest cases: not started (the patient's), started but not validated (the clinician's), validated.
2. **A confirmed assessment says when a value it used has changed since,** e.g. "The hospital BMI is now 24.7 (37 when confirmed)", or the same for an HQ answer. It's one line on the card; nothing changes until a clinician confirms again. It needs a tested helper comparing the confirmed answers with a fresh pre-fill, and a J12 check (confirm, correct the weight, see the line).
3. **Age and sex are frozen at Confirm.** Store them with the confirmed answers (source `profile`), and use the stored values for a confirmed score: the card, `poaLines`, and the "say so" line ("The patient's record now says …").
   - Go's `assessment.Check` must accept `profile` for those two items only.
   - The integration test must cover a Confirm with them, and refuse `profile` on another item.
   - Unit tests must show a confirmed score unmoved by a changed date of birth or sex.
   - A confirmed row from before this change has no stored age or sex; fall back to the record on the day of Confirm, as today.

**Retest:** these change what the card shows, so the builder writes a short test plan for a manual round (the three changes, plus a regression of T-03, T-07 and the POA Summary). Then re-verify.

Verification 2's blocking finding is fixed, and I checked it in the browser. Minor 2 is fixed too. Minors 3–7 are filed (Rahul's question, pending.md, the S48 brief). A fresh `/code-review high` of the whole working tree found nothing blocking; its new points are minor and listed below.

### Verification 2's findings

| Finding | Now |
| --- | --- |
| 1. The POA Summary dropped a confirmed assessment when the assessments failed to load | **Fixed; checked in the browser.** `poaLines` takes the error ([assessments.ts:177](../../../apps/web/src/features/episodes/assessments.ts#L177)). It's unit-tested (error, loading, off, confirmed, not confirmed), with J12 `test_the_poa_summary_says_when_the_assessments_did_not_load`. |
| 2. A BMI box that started empty couldn't be emptied again | **Fixed; checked in the browser.** `had` now comes from `base[n]` ([AssessmentsPanel.tsx:158](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L158)), and an emptied draft value falls back to the base item in `withDraft`. |
| 3. Which HQ answers the pre-fill uses mid-validation | **Rahul's question**, in decisions.md and the build notes. It doesn't block: the brief says "validated". |
| 4, 6 | In [pending.md](../../pending.md) ("From S47's verification 2"). |
| 5, 7 | In the S48 brief. |

### New findings (all minor, none blocking)

From `/code-review high`, weighed and checked against the code:

1. **Two queries' trust argument is now named `dollar_1`.** `PublishedQuestionnairesInOrg` ([admin.sql:69](../../../apps/api/db/queries/admin.sql#L69)) and `ListTrustPublishedContents` ([chapters.sql:100](../../../apps/api/db/queries/chapters.sql#L100)) cast `$1::uuid`, so sqlc names the Go parameter `dollar_1`, where it was `orgID`.
   - The behaviour is the same, but the tenancy argument is less clear to a reader.
   - **Fix:** write `sqlc.arg(org_id)::uuid`, as the other edited queries do. It's one line each plus `make sqlc`; do it in S48, or now if convenient.
2. **`publishedAssessment` compares only with the latest published version** ([assessments.go:142](../../../apps/api/internal/httpapi/assessments.go#L142)).
   - If two builds holding different STOP-Bang wording serve at once (an overlapping Lambda deploy, or a rollback), each add retires the other's version and publishes its own, back and forth.
   - That makes extra, identical versions, but no wrong data: each episode keeps the version it was given.
   - **Fix (S48 or later):** reuse any version whose content matches, published or retired, before publishing a new one.
3. **`questionnaire_versions.updated_by` and `chapters.updated_by` may now be NULL for a trust's rows too** ([00038_assessments.sql:23](../../../apps/api/db/migrations/00038_assessments.sql#L23)). Only `questionnaires` has the owner check.
   - Every Go write passes a non-null `uuid.UUID` (the casts), and the foreign key refuses a zero one, so only hand-written SQL could leave one NULL.
   - Such a version would drop out of the trust's list, because the editor join is an inner join.
   - **Fix:** a trigger or check tying a NULL `updated_by` to a platform questionnaire. It's not urgent; put it in pending.md.
4. **A JSON `null` answer gets the BMI's limits message** ([assessment.go:167](../../../apps/api/internal/assessment/assessment.go#L167)): `fits`/`isNumber` read `null` as 0. Only a hand-made request sends one, and it's still refused (400). It's cosmetic.
5. **Efficiency:** once a confirmed assessment is changed, each keystroke in the BMI box rebuilds the items from the HQ (one SurveyJS model per shown set), because `from` differs from `shownFrom` ([AssessmentsPanel.tsx:110](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L110)). Fold it into the S48 memo work, already in that brief.

**Weighed and accepted (no change)**

- **A confirmed score follows a later change to the patient record's age or sex**, as only the BMI and Yes/No items are stored. This is the builder's documented decision 2 ("as the HQ's scores do"). Rahul may want it in the same thought as his open question on a confirmed assessment beside a newer reading.
- **An unconfirmed saved row is pre-filled afresh except the clinician's items.** This was accepted in verification 2. An unconfirmed row shows no score on the POA Summary, and Confirm sends what's shown.
- **`useAssessments` copies `useObservations`.** This was accepted in verification 1. S48's brief has the memo work, and a shared hook can come with it.

### Checklist

- **Scope:** unchanged, plus the fixes. There's nothing beyond the brief.
- **Tests first:** both fixes have unit tests and a J12 check, which fail without them (build notes). The five cases per route are unchanged and pass.
- **Tenant isolation:** no query changed in meaning since verification 1. I checked by API in the browser (below).
- **A-20:** the grep finds only the earlier comments in `lib/format.ts` and `patient.go`, plus `infra/tests`' list of UK values.
- **Security:** nothing new. The assessments' routes are on the staff router only, and the patient's path is a 404.
- **Docs:**
  - decisions.md has "After verification 2" and Rahul's two open questions;
  - pending.md and the S48 brief hold the filed minors;
  - the C7 plan, plans/README.md and sessions/README.md are updated (sessions/README.md now says Verified);
  - `repocheck` passes.

### Spot-checks in the browser (Chrome, as Cara, Hospital A1; local `make dev`, the API started after the last Go change)

| Check | Result |
| --- | --- |
| **The POA Summary normally:** "Sam J12 SB 0611 112428 8a84" (confirmed STOP-Bang, hospital BMI 37) | "Hospital BMI 37 (measured 06/11/2026, 11:24)", then **"STOP-Bang 7 (High risk)"**. **Pass.** |
| **The same with `…/assessments` failing** (the page's `fetch` answering 500, then moving within the app) | The line under Hospital BMI reads **"Assessments: Couldn't be loaded"**. **Pass (verification 2's finding 1).** |
| **A BMI box that started empty:** "Sam J12 SB BMI 0611 112316 f07a" (unconfirmed, no BMI). N → No, type 30, then empty the box. | With 30: "Entered by clinician", 4 of 8. Emptied: **B Missing**, 3 of 8, **no "Enter a BMI"**, and N's change kept, so Save and Undo changes stay on. Nothing was saved. **Pass (minor 2).** |
| **Another trust and the patient's side (API):** Bea (Trust B) on that confirmed episode | Through Trust A's path: 404 "Hospital not found."; through her own B1 path: 404 "Episode not found."; `/api/p/episodes/{eid}/assessments`: 404. **Pass.** |

### Test results (6 Nov, on the working tree)

- `make lint test`: **pass**.
  - packages/clinical 514, web 365, evaluate 25, CDK 132;
  - every Go package ok.
- `make test-integration`: **pass**. Every package ok; `internal/httpapi` 21.2 s. Docker's bin was added to PATH, as before.
- `make e2e` (against `make dev`): **113 passed, 2 failed, 5 skipped** (the Cognito journeys, AWS only) in 11 min 2 s. J12 passed in full.
  - The 2 failures were J7c `test_j7c_twins_on_one_phone_are_asked_their_name` and J9b `test_j9b_cancel_with_a_reason_ends_the_patients_side`. Both were refused by the sign-in's per-network limit: "There have been too many requests from this network…", 10 per IP per hour. The builder's J12 runs this morning had used it up.
  - After `DELETE /api/qa/rate-limits`, both passed (2 passed in 14.6 s). So it's the environment, not the code.

### Suggested commits

They're the same as in verification 2 below, with these changes:
- verification 2's fixes go in commit 3 (the web app; `poaLines`'s tests and the J12 check with it);
- commit 1's queries are those verification 2 lists: `assessments.sql`, `questionnaires.sql`, `chapters.sql`, `episodes.sql` and `admin.sql`. The build notes also name `patients.sql` and `accounts.sql`, but neither changed.

## Verification 2 (5 Nov 2026)

**Verdict: Needs fixes.** Every verification 1 finding is fixed, and I checked finding 1 end to end in the browser (below). One new finding blocks: a few lines in the web app, test-first. Everything else here is minor.

### Verification 1's findings

| Finding | Now |
| --- | --- |
| 1. A re-confirm used the old Confirm day's age | **Fixed; checked in the browser.** `scoreDay` ([assessments.ts:84](../../../apps/web/src/features/episodes/assessments.ts#L84)). Details in the spot-checks below. |
| 2. Confirm before the observations or the HQ loaded | **Fixed.** `inputsState` ([assessments.ts:93](../../../apps/web/src/features/episodes/assessments.ts#L93)) has unit tests, J12 has `test_stop_bang_waits_for_the_observations`, and T-18 passed in report 3. |
| 3. Go accepted a Confirm with no answers | **Fixed.** `assessment.Unanswered` ([assessment.go:106](../../../apps/api/internal/assessment/assessment.go#L106)) only walks the JSON, which the key rule allows. `Check` still refuses `""`. |
| 4. A pre-filled BMI outside 5–150 | **Fixed** (`fits`, [clinical assessments.ts:139](../../../packages/clinical/src/assessments.ts#L139)). |
| 5. esbuild missing from `packages/clinical`'s package.json | **Fixed.** |
| 6. Go's list of assessments kept by hand | **Moved to the S48 brief.** |

### Findings, worst first

**Must fix**

1. **The POA Summary silently leaves out a confirmed assessment when the assessments fail to load.** See [PoaSummary.tsx:95](../../../apps/web/src/features/episodes/PoaSummary.tsx#L95) and [assessments.ts:162](../../../apps/web/src/features/episodes/assessments.ts#L162).
   - `poaLines(assessments.view, e)` returns `[]` while `view` is null, and nothing on the page reads `assessments.error`.
   - **Reproduced in the browser:** Ray "MT S47R3 2240 One" has a confirmed "STOP-Bang 6 (High risk)". With `…/assessments` failing, the POA Summary shows no STOP-Bang line and no error anywhere on the page.
   - The Hospital BMI line just above it handles the same failure: it says "Couldn't be loaded" (`hospitalBmiRow`).
   - **Why it matters:** the POA Summary is what the anaesthetist reads and prints. A missing High-risk STOP-Bang there looks like "no assessment done".
   - **Fix:** give `poaLines` the error, as `hospitalBmiRow` gets it. While the view is null and there's an error, show one line, e.g. "Assessments: Couldn't be loaded".
   - **Test:** a `poaLines` unit test (error, loading, off, confirmed, not confirmed). A J12 check is cheap if it reuses the failing-route setup of `test_stop_bang_waits_for_the_observations`.
   - **Retest:** this is the only change, so the builder's tests plus my re-verify are enough. No full manual round is needed unless the builder changes more.

**Minor (non-blocking; fix while there, or put in pending.md)**

2. **A BMI box that started empty can't be emptied again after a value is typed in it.** See [AssessmentsPanel.tsx:214](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L214) (found by reading the code).
   - `had` is `!!drafted[n].source`, which is true once the draft holds the typed value. Clearing the box then shows "Enter a BMI" and blocks Save and Confirm.
   - The only way out is Undo changes, which also throws away the other items' changes (e.g. N).
   - **Fix:** when a box that started empty is emptied, remove its key from the draft (`had` from `base[n]`), so B is simply missing again.
3. **Question for Rahul: which HQ answers the pre-fill uses mid-validation.** See [clinical assessments.ts:101](../../../packages/clinical/src/assessments.ts#L101).
   - `countingHqAnswers` takes the clinician's copy only once a set is **validated**, as the brief says and the build notes record.
   - The rest of the episode page counts the clinician's copy as soon as they've **started**: the Question Sets card, the suggested ASA (S39) and Complete review's `/shown` (`currentAnswers` in `hq.ts`).
   - **Example:** a clinician changes S from No to Yes while validating and hasn't pressed Validate. The suggested ASA uses Yes, but the assessment pre-fills S as No, with the source "HQ".
   - **Options:**
     - (a) **Recommended:** use the clinician's copy once started, like the rest of the page. The same clinician confirms the assessment.
     - (b) Keep the validated copy only, as the brief says.
4. **"Today" is the browser's clock, but the Confirm day is the server's `now()`.** See [AssessmentsPanel.tsx:104](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L104).
   - On a confirm within seconds of midnight on the patient's birthday, or from a browser whose clock is off by a day, the score shown and the score confirmed can differ by A's point.
   - This is very rare, and the key rule stores no score. Note it in pending.md.
5. **Go accepts any of the three sources for any item,** e.g. `observation` on Snoring ([assessment.go:145](../../../apps/api/internal/assessment/assessment.go#L145)). Only a hand-made request could send that, since the clinician's browser is trusted. When S48 adds more items, consider allowing `observation` on number items only.
6. **After one failed HQ reload, `useEpisodeHQ` keeps its error even when a later reload succeeds.** The card then stays blocked, saying "Reload the page to save or confirm", which is the remedy, so it fails safe. The hook dates from before S47; note it in pending.md.
7. **Small efficiency points:**
   - `poaLines` runs in render without `useMemo`, building one SurveyJS model per confirmed assessment on every render.
   - The card's memos follow the `episode` object, so every page reload walks the HQ again.
   - This is fine at one assessment. Revisit in S48, when there are four.

**Weighed and accepted (no change)**

- **An unconfirmed saved row is pre-filled afresh except for the clinician's items,** so a saved HQ or observation value can follow a newer reading. This is the documented design (decisions.md, 5 Nov). An unconfirmed row never shows a score on the POA Summary, and Confirm sends what's shown.
- **`useAssessments` copies `useObservations`.** This was accepted in verification 1.
- **Rahul's open question** (a confirmed assessment beside a newer reading) is still open and doesn't block.

### Checklist

- **Scope:** unchanged since verification 1, plus the fixes. There's nothing beyond the brief.
- **Tests first:**
  - The fixes have tests: `scoreDay`, `inputsState`, `TestUnanswered` and a `TestSaveAndConfirmAssessment` case, the BMI-limits case in `assessmentItems`, and J12 `test_stop_bang_waits_for_the_observations`.
  - The five cases per route were checked in verification 1 and still pass.
- **Tenant isolation:** no query changed since verification 1. The fixes are browser-only, plus one Go check that runs after `loadEpisode` and `GetAssessmentForm`.
- **A-20:** the grep finds only the earlier comments in `lib/format.ts` and `patient.go`.
- **Security:** nothing new. `Unanswered`'s 400 names the item ("BMI"), never an answer.
- **Docs:**
  - decisions.md has "After verification 1".
  - The build notes, the C7 plan, plans/README.md and sessions/README.md are updated.
  - Minor 6 is in the S48 brief.
  - `repocheck` passes.

### Spot-checks in the browser (Chrome, as Cara, Hospital A1; local `make dev`)

| Check | Result |
| --- | --- |
| **Verification 1's finding 1, end to end** (no one had reproduced it). I back-dated the Confirm on Ray's made-up STOP-Bang (born 14/03/1958) to 01/03/2008 in the local database, when he was 49. | **Pass.**<br>• Confirmed and unchanged, the card showed A = 49, "4 (High risk)", "Confirmed … 01/03/2008".<br>• After N → Yes it showed A = 68, "6 (High risk)", "Changed: confirm it again" and the note "The age is the patient's today."<br>• Confirm gave "Confirmed by Cara Clinician on 05/11/2026, 23:16", 6 (High risk), and the POA Summary said **STOP-Bang 6 (High risk)**, the score the card showed.<br>• Before the fix the card would have shown 5. |
| **The POA Summary with `…/assessments` failing** (the page's `fetch` replaced, then moved within the app, as in report 3) | **Fail (finding 1).** The STOP-Bang line disappeared, with no error. The Hospital BMI line above it handles its own failure. |
| **Verification 1's finding 2** | Covered by report 3's T-18 and the J12 check. Not repeated. |

Ray's episode now holds my re-Confirm (made-up test data, local only).

### Test results (5 Nov, on the working tree)

- `make lint test`: **pass**.
  - packages/clinical 514, web 364, evaluate 25, CDK 132;
  - every Go package ok.
- `make test-integration`: **pass**. Every package ok; `internal/httpapi` 21.7 s.
  - Docker's bin had to be added to PATH again, as in verification 1.
- `make e2e` (against `make dev`): **114 passed, 5 skipped** (the Cognito journeys, AWS only) in 10 min 35 s, on 6 Nov.
  - J12's 13 checks passed in both runs.
  - The first run (5–6 Nov, around 01:40–02:00) had 12 timeouts in journeys S47 doesn't touch: staff, trusts, question keys, the library, video, and one J16b check. The machine's load average was about 11, and the servers were healthy.
  - The second run, after a rest, passed everything. So the timeouts were the environment, not the code.

### Suggested commits (after the fix and a re-verify)

They're the same as in verification 1 below. The fix goes in commit 3 (the web app).

## Verification 1 (5 Nov 2026)

**Verdict: Needs fixes.** One small correctness bug in the score a clinician confirms (finding 1) and one small guard (finding 2). Both are a few lines each, test-first, in the web app. Everything else is sound: the Go side, tenant isolation, the migration, the patient never seeing an assessment, the docs, and the two fixes after test report 2 (checked by hand below).

### Findings, worst first

**Must fix**

1. **Re-confirming a changed assessment shows a score worked out with the old Confirm day's age, but Confirm stamps today.** See [AssessmentsPanel.tsx:92](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L92) and [:175](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L175).
   - `from` passes `row.confirmedAt` to `prefillFrom`, so while a confirmed assessment shows "Changed: confirm it again", A (age over 50) and the result badge use the patient's age on the *earlier* Confirm day. The note under the card says "on the day it was confirmed (50)".
   - Pressing **Confirm** stamps `validated_at = now()`. After that, `poaLines` and the reloaded card work the score out with *today's* age.
   - **Failure:** the patient was 50 at the first Confirm and is 51 now. The clinician changes N, sees "4 (Intermediate risk)" and confirms. The POA Summary then says "5 (High risk)", a score the clinician never saw.
   - The same mismatch happens for any change made after the age crosses a band edge. It's rare but deterministic, and it's the score a clinician signs.
   - **Fix:** use the Confirm day's age only while the assessment is shown as confirmed and unchanged. Once it's changed (or not confirmed), use today's, which is what Confirm will record. Keep `base` (the shown values, for working out `changed`) as it is.
   - **Test (Vitest):** a helper that picks the day, e.g. `scoreDay(row, changed)`, with a case for a patient who crossed 50 since the first Confirm. Add a J12 check if it's cheap.

**Should fix (small)**

2. **The card can be confirmed before the observations have loaded, or after their load failed.** The BMI then quietly comes from the patient's HQ answer, against the trust's `observations.precedence`. See [EpisodePage.tsx:186](../../../apps/web/src/features/episodes/EpisodePage.tsx#L186) and [AssessmentsPanel.tsx:168](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L168).
   - `readings` is `undefined` both with observations off and while they're loading or failed. `assessmentItems` then uses the patient's BMI (`own`), labelled "Patient reported BMI". The label is honest, but nothing says the hospital BMI wasn't considered.
   - While the HQ is loading, S, T, O, P and the patient's BMI are all missing, so Confirm is off. The gap only opens when the HQ has loaded and the observations haven't (slow or failed).
   - **Fix:** when `observationsOn` is true, treat the card as still loading (no Save or Confirm, or no items yet) until the observations have loaded. If their load fails, say so on the card. Do the same for the HQ's load error (`hq.error`).
   - **Test:** a unit test on whatever decides "ready". J12 isn't needed.

**Minor (non-blocking; fix now if convenient, or note for S48)**

3. **Go doesn't check that a Confirm has its answers.** See [assessments.go:208](../../../apps/api/internal/httpapi/assessments.go#L208).
   - A direct `POST …/confirm` with `{data:{}, sources:{}, revision:N}` stamps `validated_at`, and the POA Summary then shows "Incomplete" beside "Confirmed by …". The browser never sends that (Confirm needs every item), and a stale tab gets the 409.
   - Checking that every stored question has an answer is only JSON walking, so the key rule allows it. Recommended for S48, when DASI and the others make the shape richer.
4. **A pre-filled BMI isn't checked against the BMI question's limits (5–150).** See [packages/clinical/src/assessments.ts:137-141](../../../packages/clinical/src/assessments.ts#L137-L141).
   - A patient's HQ BMI outside them would show as valid, and Go would refuse the Confirm with "BMI: a number from 5 to 150." That message is clear, so this is cosmetic.
   - It's unlikely, as both BMIs come from limited heights and weights.
5. **`packages/clinical`'s new `assessments` script imports esbuild, but the package doesn't list it** ([package.json:12](../../../packages/clinical/package.json#L12)). It works only because `apps/evaluate`'s esbuild is hoisted. Add `"esbuild": "0.28.2"` to its devDependencies.
6. **Go's list of assessments** (`All`, [assessment.go:32](../../../apps/api/internal/assessment/assessment.go#L32)) is kept by hand beside `packages/clinical`'s, and the Vitest check compares only the JSON. When S48 adds DASI, PRISMA-7 and Apfel, consider having the script write the id and name list too.

**Weighed and accepted (no change)**

- **A confirmed score follows later changes to the patient record's sex or date of birth.** No score is stored, by the key rule. The builder recorded this in decisions.md (5 Nov), and the HQ's scores behave the same way (S38).
- **The `::uuid` casts in the trust's questionnaire queries hide the new nullability from sqlc.** Every one of those queries filters `q.org_id = $org`, so it can never reach a platform row. A future query without that filter that scans `org_id` would fail loudly, not silently. Recorded in decisions.md.
- **`useAssessments` copies the shape of `useObservations`.** That's acceptable for two cards.

**Still Rahul's open question** (build notes, report 1 note 4, decisions.md): should a confirmed assessment say when a value it used has changed since (e.g. "The hospital BMI is now 27.8 (37 when confirmed)")? It doesn't block verification.

### Checklist

- **Scope:** everything in the brief is built. That covers the four answers, the platform assessment with versions, the attach, pre-fill with sources, save and Confirm with the revision check, the POA line, `feature.assessments`, the setting, J12, and nothing extra. Two differences are recorded in the build notes: the setting's name (`observations.precedence`) and the JSON written from `packages/clinical`.
- **Tests first:** each of the 4 routes has the five cases (happy path, 401, another hospital's clinician (the role cases here, as any clinician may), another trust's 404, invalid 400). It also has cancelled or archived (409), turned off (403), stale (409), duplicate add (409), a new version on changed wording, `TestPatientNeverSeesAssessments`, `TestAssessmentsAudited` and `TestAssessmentsMigration` (rolling back and forward). Unit tests cover `assessmentItems` (each source, precedence both ways, an unanswered question, a hidden one), `bmiInput`, `withoutInvalid`, `withDraft`, `assessment.Check` and the JSON comparison.
- **Tenant isolation:**
  - Every route goes through `requireHospital` → `loadEpisode` (episode in this hospital and trust).
  - `GetAssessmentForm` checks both `episode_id` and `kind = 'assessment'`.
  - Every existing form query filters `kind = 'hq'`, including `ListPrefillSources`, the patient's routes, the jobs, the links and the accounts queries.
  - Every trust questionnaire query filters `org_id`, so the platform rows never appear in a trust's lists.
- **A-20:** the grep finds only comments in `lib/format.ts` and `patient.go` (from earlier sessions). Nothing new.
- **Security:**
  - no secrets;
  - no answers in logs (the 500 log names only `what`);
  - assessment routes are on the staff router only;
  - the assessment's chapter is `audience = 'clinician'`;
  - all four routes are `audited`.
- **Docs:** decisions.md, the C7 plan's Status, plans/README.md, sessions/README.md, CLAUDE.md and apps/web/AGENTS.md are updated. `repocheck` passes.

### Spot-checks in the browser (Chrome, as Cara, Hospital A1; local `make dev`)

| Check | Result |
| --- | --- |
| **B's box after report 2's fixes (not retested by hand before):** Ann "MT S47 1736 Two" (unconfirmed, female, hospital BMI 27.8). | 36.5 gives "Entered by clinician", 3 (High risk), 8 of 8. Then "2" gives **B Missing**, "Between 5 and 150", "7 of 8 filled: answer the item highlighted.", **Incomplete**, and Save and Confirm both off. The BMI box and the Yes/No buttons stayed at x = 866 throughout (the fixed column). Nothing was saved. **Pass.** |
| **Another hospital, another trust (API):** Val (A2) and Bea (Trust B) on Ann's assessments. | Through A1's path, list, add, save and confirm are all 404. Through Val's own A2 path and Bea's own B1 path with Trust A's ids: 404 "Episode not found." **Pass.** |
| **Form ids:** Cara on Ann's episode with Tom's assessment id, and with Ann's HQ form id. | Both 404 "Assessment not found." **Pass.** |
| **Invalid and stale (API):** an unknown key, then a Confirm at an old revision. | 400 "An answer doesn't belong to this assessment."; 409 "These answers were changed somewhere else…". **Pass.** |
| **Re-confirm age (finding 1):** read from the code and the card's own note ("on the day it was confirmed (N)"), which shows the old day while changed. | **Confirmed by reading.** It wasn't reproduced in the browser, as it needs a Confirm stamped on an earlier day. |

### Test results (5 Nov, on the working tree)

- `make lint test`: **pass**.
  - packages/clinical 513 passed, web 357 passed, evaluate 25 passed, CDK 132 passed;
  - every Go package ok, including `internal/assessment`, `settings`, `cmd/seed` and `repocheck`.
- `make test-integration`: **pass**. Every package ok; `internal/httpapi` 28.6 s.
  - Docker wasn't on the shell's PATH for `make`'s `db-up`, so it was run with `/Applications/Docker.app/Contents/Resources/bin` added. That's an environment issue, not the code.
- `make e2e` (against `make dev`): **113 passed, 5 skipped** (the Cognito journeys, AWS only) in 10 min 52 s.

### Suggested commits (after the fixes and a re-verify)

1. **S47: assessments as platform questionnaires; the add, save and Confirm routes; observations.precedence.**
   - Migration 00038 and the queries (`assessments.sql`, and the casts in `questionnaires.sql`, `chapters.sql`, `episodes.sql`, `admin.sql`) with the regenerated `internal/db`;
   - `internal/assessment` (Go code and tests; `stopbang.json` goes in commit 2);
   - `httpapi/assessments.go`, the router, the audit labels, `assessmentsOn`, `lockedEpisode`; settings;
   - `assessments_integration_test.go` and the migration test.
2. **S47: STOP-Bang's pre-fill and score in packages/clinical; the JSON for Go.** `assessments.ts`, `assessments.test.ts`, `index.ts`, `package.json`, `scripts/assessments.mjs` and `apps/api/internal/assessment/stopbang.json`.
3. **S47: the Assessments card, the POA Summary's line, the demo HQs; J12.** `AssessmentsPanel.tsx`, `assessments.ts` and its tests, `EpisodePage.tsx`, `ObservationsPanel.tsx`, `PoaSummary.tsx`; the seed's two HQs and `keys_test.go`; `e2e/data/*.json`, `test_j12_observations.py` and `e2e/README.md`.
4. **S47: docs.** decisions.md, the plans, the sessions README and the session folder, CLAUDE.md and apps/web/AGENTS.md.

`docs/market/` is untracked and isn't part of S47. Leave it out of these commits.
