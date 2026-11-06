# S48: DASI, PRISMA-7 and Apfel

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), part 3 of 4.
**Requirements:** §9's assessments 2–4 ([saas-requirements.md](../../saas-requirements.md)); SCR-01 (ready-made scores in an HQ).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S47 verified.

**Why these three together:** each can be pre-filled almost entirely from the HQ (§9: DASI 12 of 12, PRISMA-7 7 of 7, Apfel 3 of 4, with post-op opioids from the clinician), and none needs observations. saas-requirements §14's phase 7 groups them with STOP-Bang.

**What exists today:** after S47, the assessment mechanism works for STOP-Bang: attach, pre-fill by key with sources, complete, Confirm, and the score and band on the POA Summary. `ready.ts` builds locked ready-made scores from `scores.ts`'s parts, and `ready.test.ts` checks STOP-Bang against its published scoring.

**Ask Rahul first:**
1. **DASI's result.** DASI gives a score (0–58.2) and an estimated functional capacity in METs.
   - (a) **The score and the METs, plus one band, "Poor functional capacity", at the published cut-off** (the builder cites the source; a clinician checks it before go-live).
   - (b) The score and the METs only, with no band.

   **Recommended: (a).** A band is what disclosures and rules can act on, as STOP-Bang's are.
2. **Where the three appear.**
   - (a) **Both as assessments and as ready-made scores an author can insert in an HQ**, from one definition (as STOP-Bang is).
   - (b) As assessments only.

   **Recommended: (a).** §9 relies on the HQ asking the same keyed questions, and an HQ that embeds the score makes the assessment complete at once.

## Build

- **Before any code: each instrument's terms.** Record the published source and the licence in each `ReadyScore` (`source`, `licence`), as STOP-Bang's are. **If any needs payment or permission for use in software, stop and ask Rahul** (STOP-Bang's licence is already a go-live item, decisions.md).
- **`packages/clinical`, test-first:** each instrument as a locked ready-made score in `ready.ts`, with its published wording, points and bands, and the Incomplete state while an item has no answer. **Its scoring is checked against worked examples from its published source** (`ready.test.ts`), including each band's edges. Keys follow STOP-Bang's pattern (`dasi_*`, `prisma7_*`, `apfel_*`).
- **As assessments** (S47's mechanism): each becomes an assessment a clinician can add, pre-filled by key, with Apfel's post-op opioids item left to the clinician. Patient sex comes from the profile, as STOP-Bang's does.
- **In the HQ** (question 2): Insert offers them beside STOP-Bang, and Publish refuses a changed one (`readyScoreProblems`).
- **From S47's verification (minor 6):** Go's list of assessments (`assessment.All`: id and name) is kept by hand beside `packages/clinical`'s `ASSESSMENTS`, and the Vitest check compares only the JSON. With three new ones, have `npm run assessments` write the id and name list too (or test that they agree).
- **From S47's verification 2 (minors 5 and 7):**
  - Go's `assessment.Check` accepts any of the three sources on any item (e.g. `observation` on Snoring). With more items, allow `observation` on number items only.
  - `poaLines` runs in render without `useMemo`, one SurveyJS model per confirmed assessment. The card's memos follow the `episode` object, so each page reload walks the HQ again. With four assessments, memoise both.
- **From S47's verifications 3 and 4 (minor, none blocking):**
  - **Two queries' trust argument is named `dollar_1`:** `PublishedQuestionnairesInOrg` (`admin.sql`) and `ListTrustPublishedContents` (`chapters.sql`) cast `$1::uuid`. Write `sqlc.arg(org_id)::uuid`, as the other queries do, then `make sqlc`.
  - **`publishedAssessment` compares only with the latest published version** (`httpapi/assessments.go`). Two builds with different wording serving at once (an overlapping deploy, a rollback) would retire each other's version on every add. Reuse any version whose content matches before publishing a new one.
  - **The pre-fill runs on every keystroke once a card is changed:** `changedItems` calls `assessmentItems` twice, and each call builds a SurveyJS model per shown HQ set. The pre-fill doesn't depend on the draft: work it out once per `from`, then apply the draft and the profile swap on top. Do this with the memo work above.
  - **Go's Confirm doesn't require the age and sex** (`patientAge`, `patientSex`) when the content names them. A hand-made Confirm without them falls back to the record on the day of Confirm, as a row from before S47's answers 7 does. Have `assessment.Unanswered` cover the profile items the content names. Its `null` test can't happen, as `Check` refuses `null` first; drop it.
  - **`label()` in `internal/assessment` names STOP-Bang's keys** (`stopbang_bmi` → "BMI"). Each new instrument would need another case. Have `assessments.ts` write a short label into the content instead.
  - **`countingHqAnswers`** (`packages/clinical`) **and `currentAnswers`** (web `hq.ts`) are now the same rule (S47 answers 5). Have `hq.ts` use the clinical one, so they can't drift apart.
  - **A renamed assessment keeps its old name** on its questionnaire row: `AssessmentQuestionnaire` inserts with `ON CONFLICT DO NOTHING`. Update the name there if a name ever changes.
- **POA Summary:** each confirmed one's score and band (DASI with its METs).
- **Integration:** attach each one; pre-fill sources per item; Confirm.
- **E2E:** extend J12: an HQ embedding DASI, sent by the patient (phone); the DASI assessment opens complete; Confirm; its METs show on the POA Summary.

## Not in this session

- MUST, falls, VTE and Rockwood (S49).
- Using DASI or PRISMA-7 as hints for Rockwood (S49).

## Manual test focus

1. Each of the three gives the published result for the source's worked examples (a tester enters them by hand).
2. An HQ embedding all three: the patient answers them (phone); each assessment opens filled, Apfel waiting only for post-op opioids.
3. The bands' edges are right (the test plan lists the values).
4. A changed ready-made score can't be published.
