# S48 build notes: DASI, PRISMA-7 and Apfel

[Brief](brief.md) · [Answers](answers.md) · [Test plan](test-plan.md). Plan C7, part 3 of 4. Built 6 Nov 2026.

## Answers (Rahul, 6 Nov)

All four recommended options ([answers.md](answers.md)):
1. build all three now; the licences are go-live items;
2. DASI: score, METs, and one band, "Poor functional capacity", at 34 or less;
3. both assessments and ready-made scores for the HQ;
4. the HQ's Apfel is the patient's two questions, with no score.

## Each instrument's terms (before any code, as the brief asks)

| | Source used | Licence |
| --- | --- | --- |
| **DASI** | Hlatky MA et al., Am J Cardiol 1989;64:651–4: 12 items and weights (58.2 in all); METs = (0.43 × DASI + 9.6) / 3.5. Band: Wijeysundera DN et al., BJA 2020;124(3):261–70 (34 is the threshold; "better than 34" had lower risk, so Poor is 34 or less). | © Mark Hlatky; Greenlight Health (formerly Pattern Health) is the exclusive licensor, "based on use case"; no fee published. **Rahul asked: build now, licence at go-live.** |
| **PRISMA-7** | Raîche M, Hébert R, Dubois MF, Arch Gerontol Geriatr 2008;47:9–18; wording and scoring from the BCGuidelines.ca form (2017): "Are you older than 85 years?", "Are you male?", 3–7 as worded; "3 or more 'yes' answers … increased risk of frailty". **Every Yes scores, question 6 too** (the form, and Hoffmann et al. BMJ Open 2020). | © CESS Sherbrooke; an electronic version needs its permission (the CESS user guide wouldn't open: 403). **Go-live item.** |
| **Apfel** | Apfel CC et al., Anesthesiology 1999;91:693–700: female, non-smoker, history of PONV or motion sickness, post-op opioids; 10%, 21%, 39%, 61%, 79% for 0–4. The full text and Gan 2020's guidelines wouldn't open (paywalled), so the bands are named by the published incidence, not low/moderate/high. | None found. |

Each `ReadyScore` in `ready.ts` carries its `source` and `licence`, shown in the Insert window. pending.md lists the licences and the wording checks for a clinician.

## What was built

### TypeScript (for Rahul to review)

- **`packages/clinical/src/ready.ts`:**
  - `ReadyScore` gains **`build(bmi)`**, the Group as inserted, and an optional **`note`**. `insertReadyScore` and `readyScoreProblems` use it, so they no longer name STOP-Bang. The publish check is the same lock as before: anything but the Group's condition and the disclosures is a change.
  - **`dasiGroup()`:** 12 Yes/No questions (`dasi_selfcare` … `dasi_strenuous_sports`), each Yes its weight. The score is `iif(allAnswered(…), round(score(…), 2), '')`: the rounding stops a sum of decimals missing the cut-off by 0.0000001. **`dasi_mets`** is `round((0.43 × score + 9.6) / 3.5, 1)`, shown as "6.9 METs" (SurveyJS's `format` and fraction digits). The score and METs are clinician-only, like STOP-Bang's score.
  - **`prisma7Group()`:** questions 3–7 (`prisma7_limit`, `_help`, `_home`, `_count_on`, `_aid`), 1 point per Yes, plus "Add 1 point when" lines for age over 85 and male, built with the score form (`formatScore`) as STOP-Bang's are. Bands: "Not at increased risk" (0–2) and "Increased risk of frailty" (3+).
  - **`apfelGroup()`:** the HQ part, two questions (`apfel_smoking`: a **No** scores 1, the non-smoker's point; `apfel_ponv`), no score. **`apfelClinicianPart()`:** the opioids question (`apfel_opioids`) and the score (female 1 point), only in the assessment. Five bands, "PONV risk about 10%" … "79%".
  - **`STRICT_SEX_SCORES`:** STOP-Bang's, PRISMA-7's and Apfel's scores, where a sex of Other makes the score Incomplete (S44c's rule; `questionTypes.ts` used to name STOP-Bang only).
  - `READY_SCORES` = STOP-Bang, DASI, PRISMA-7, Apfel PONV.
- **`packages/clinical/src/assessments.ts`:** one definition per assessment (`DEFS`: id, name, its elements, its items in the published order). STOP-Bang's is unchanged.
  - `assessmentItems` works for any of them: the definition is found from the content's questions (`itemsOf`).
  - Items are numbered "1"… for the three new ones. A profile item (age, sex) has its own threshold: older than 85, male, female.
  - **`labels`** (new, in the content) gives what Go's messages call each item. Most use the card's label; STOP-Bang's B and N keep "BMI" and "Neck", and the profile's "Age" and "Sex".
  - **`assessmentResult`** adds the assessment's other Calculations as they show: "34 (Poor functional capacity), 6.9 METs".
  - **The HQ part of the pre-fill is worked out once per HQ** (`hqPart`, a `WeakMap` on `from.hq`): the shown-question walk and the patient reported BMI. Before, every keystroke built a SurveyJS model per HQ set, twice (S47 verification 4).
- **`scripts/assessments.mjs`** writes `list.json` (id and name, in order) beside each assessment's JSON. Go reads its list from it, and the Vitest check compares every file.
- **`rules.ts`:** the "(not in this HQ)" names stay STOP-Bang's only (the default rules name it). Offering every ready-made score there would have widened the Rules page (3 tests showed it).
- **Web:**
  - `ReadyScoreWindow` shows the score's `note`, and the editor's insert uses `build`.
  - `assessments.ts`:
    - `prefillFrom` keeps one HQ-part object per loaded HQ and patient (`hqPart`), so the clinical cache works across the cards, a re-confirm's other day and each keystroke;
    - `cardNotes`: the age note only for an assessment with an age item, and the "doesn't say male or female" note only where a sex item is missing. DASI has neither; before, the sex note would have shown on DASI.
  - `PoaSummary`: `poaLines` in a `useMemo`.
  - `hq.ts` `currentAnswers` is now `countingHqAnswers` from `@poised/clinical`: one rule.

### Go

- **`internal/assessment`:**
  - `All` comes from `list.json`, no longer a hand-kept list.
  - `label()` reads the content's `labels`, else the title's first words (a STOP-Bang version from before S48). No STOP-Bang names in Go.
  - **`Check`:** an `observation` source only on a number item (S47 verification 2, minor 5). An empty (`null`) answer is refused; before, a `null` age read as 0 and passed.
  - **`Unanswered`** covers the age and sex wherever the content names them (`{patientAge}`), and its `null` test is gone (Check refuses it first). A hand-made Confirm without them is now 400 ("Age has no answer").
- **`publishedAssessment`:** `MatchingAssessmentVersion` reuses the newest version, published or retired, whose content matches, so two builds serving at once don't republish on every add. **`AssessmentQuestionnaire`** upserts the name (`ON CONFLICT … DO UPDATE SET name`), so a rename takes effect with its new version.
- **The two `dollar_1` queries** (`PublishedQuestionnairesInOrg`, `ListTrustPublishedContents`) now use `sqlc.arg(org_id)::uuid`. Their Go call sites are unchanged.
- **Seed:** **Demo fitness HQ** (Trust A, made up, ids 7xx): one set holding the ready-made DASI, PRISMA-7 and Apfel's questions. It's added to the seed's keys test.
- **No migration.** STOP-Bang's assessment JSON changed (it gained `labels`), so the next add publishes its version 2; episodes keep version 1, whose messages fall back to the titles' first words.

### E2E

- **J12:** `test_an_hq_with_dasi_sent_on_a_phone_opens_dasi_complete`, at phone size (WebKit, iPhone 13):
  - the patient sends `e2e/data/fitness-hq.json` (the three, as seeded), and sees no score;
  - DASI opens 12 of 12 from the HQ with "34 (Poor functional capacity), 6.9 METs" (the badge whole at 1456 wide), and is confirmed;
  - PRISMA-7 opens 7 of 7 with "1 (Not at increased risk)";
  - Apfel opens 3 of 4 with item 4 missing; "4: Yes" gives "4 (PONV risk about 79%)", and it's confirmed;
  - the POA Summary shows DASI's METs, Apfel's risk and PRISMA-7 Not confirmed.
- **An S47 check changed:** after STOP-Bang is added, the Add assessment menu stays (it offers the others) but no longer offers STOP-Bang.

## Decided by the builder

1. **PRISMA-7 scores question 6's Yes,** as the BC form and the CESS-derived sources do ("3 or more yes answers"), though a No (no one to count on) reads as the riskier answer. A clinician checks it (pending.md).
2. **"Older than 85" is over 85:** 85 doesn't score, 86 does (the form's wording).
3. **PRISMA-7's lower band is named "Not at increased risk"** (the source names only 3+). It's needed: a band with no maximum would otherwise catch every value.
4. **Apfel's bands are named by the published incidence** ("PONV risk about 39%"). Apfel 1999 gives incidences, not categories, and the 2020 guidelines couldn't be read. A rule can still act on each band.
5. **Apfel's question wording is ours** (Apfel names risk factors): "Do you smoke?", "Have you ever felt or been sick after an anaesthetic, or do you get travel sickness (motion sickness)?", and the clinician's "Will the patient be given opioids after surgery?". The smoking question's No scores (the non-smoker).
6. **DASI's wording is Hlatky's as commonly reproduced** (US terms: "a block or two", "yardwork"); the licensed version may differ.
7. **The DASI score is rounded to 2 places** and the METs to 1. 34 is Poor (34 or less).
8. **Items are numbered 1… on the new cards** (STOP-Bang keeps its letters), in the published order, with PRISMA-7's age and sex as items 1 and 2.
9. **Apfel needs no BMI or age; DASI no age or sex,** so those cards have no age note.
10. **The Rules page's "(not in this HQ)" stays STOP-Bang's only** (above).
11. **A version with the same content is reused whether published or retired;** statuses aren't changed back. A new version is published only when no version matches.

## Differences from the brief

- None in scope. The brief's "Insert offers them beside STOP-Bang" is built. For Apfel, Insert gives only the patient's two questions with no score (answers 4), so "Publish refuses a changed one" there covers those two questions.

## Test results (6 Nov)

- `make lint`: pass.
- `make test`: pass. clinical 562 (was 518), web 378 (was 374), evaluate 25, CDK 132, every Go package.
- `make test-integration`: pass. New assessment cases:
  - all four on one episode, each its own platform questionnaire;
  - a matching older version reused;
  - a renamed questionnaire;
  - the four offered, in order;
  - Apfel's sources per item;
  - Confirm of DASI, PRISMA-7 (400 "Age has no answer" first) and Apfel (400 "Opioids after surgery has no answer" first);
  - an age Apfel doesn't name, and an observation on a Yes/No item, refused.
- **Shown red first:** the ready-made scores' tests (DASI, PRISMA-7, Apfel, the strict sex with the old one-score rule), the assessments' tests, the web tests (`cardNotes`, the shared HQ part), Go's `TestAllFromTheList`, `TestCheck` (observation, `null` age) and `TestUnanswered`, and the reuse and rename integration cases.
- `make e2e`: **117 passed, 5 skipped** (the Cognito journeys, AWS only) in 10 min 47 s. It was run on a second local stack (API :8082, evaluate :8092, web :3002), as Rahul's `make dev` predates the change. **Restart `make dev` before the manual round.**

## Test report 1 (6 Nov): all passed

17 of 17 (T-01–T-12, R-01–R-05). The phone cases ran in a 390×844 iframe, because Chrome's window wouldn't resize. No code was changed after the report. Its exploring notes are open items in [pending.md](../../pending.md) ("From S48's test report 1"):
- **Finding 1, for Rahul:** Go accepts a `profile` age or sex that differs from the patient's record. A hand-made Confirm with `patientSex: "male"` on a patient recorded as Other gave a confirmed PRISMA-7 of 4.
  - Go checks that the item is there, not its value. Go can work the age out (it has the date of birth and the hospital's time zone) and compare the sex, so it could refuse a mismatch (400).
  - **Recommended:** refuse it. A "Profile" source should always be the record's.
- **Finding 2, for Rahul:** for a sex of Other or Not known, the card says "answer the item highlighted", though nothing on the card can answer it. The note under the card says why. **Recommended:** say "the patient's record has no male or female" in the headline instead. It's the same on S47's STOP-Bang G.
- **Findings 3 and 4 (cosmetic):** the result badge's capitals turn "METs" into "METS" (`tt="none"` would keep the case), and the POA Summary's assessment values sit about 25 px right of the other values.
- **Notes 5–7:**
  - unsaved card changes are lost on reload without a warning, as before S48;
  - option points reach the patient's browser, as for STOP-Bang;
  - a Poor DASI doesn't hold auto-triage: the default rules don't know DASI. For the rules work (S49 or later).

## Suggested commits (Rahul commits after verification)

1. **S48: DASI, PRISMA-7 and Apfel as ready-made scores; the editor's insert.** `ready.ts`, `questionTypes.ts`, `scores.ts` (comment), `rules.ts`, `ready.test.ts`, `ReadyScoreWindow.tsx`, `QuestionnaireEditor.tsx`. Tests: `ready.test.ts`, `rules.test.ts`.
2. **S48: the three as assessments; labels and list.json for Go; the pre-fill worked out once.** `assessments.ts`, its tests, `scripts/assessments.mjs`, `apps/api/internal/assessment/*.json`, the web `assessments.ts`, `AssessmentsPanel.tsx`, `PoaSummary.tsx`, `hq.ts` and their tests. Tests: clinical and web `assessments.test.ts`.
3. **S48: Go: the list from list.json, labels, the checks; version reuse, the name, the org_id arguments.** `internal/assessment`, `httpapi/assessments.go`, `db/queries/{assessments,admin,chapters}.sql`, `internal/db`, the integration tests. Tests: `go test ./internal/assessment`, `TestAddAssessment`, `TestListAssessments`, `TestSaveAndConfirmAssessment`.
4. **S48: Demo fitness HQ; J12.** `cmd/seed` (main, keys test, `demo-fitness-hq.json`), `e2e/data/fitness-hq.json`, J12. Tests: `go test ./cmd/seed`, J12.
5. **S48: docs.**
