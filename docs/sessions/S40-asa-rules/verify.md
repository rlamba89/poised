# S40 verification

## Verification 2 (26 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1 found F1 and F2, and both are fixed. So is test report 3's save race. I confirmed all three myself, through the API and in Chrome. Test report 4 passed all 14 cases. The second review found nothing blocking. Its smaller points are now in pending.md ("From S40's verification 2").

### What I checked

- **F1 (`apps/api/internal/httpapi/rules.go:32-42`):** the PUT has to name the version the page loaded.
  - No `versionId` gives 400. Another version than the latest gives 409, "This version has changed since you opened it. Reload and try again.", before anything is saved.
  - `TestSaveRules` has the subtest: publish, make a new version, then a stale PUT gets 409 and v2 keeps its two defaults at revision 0.
- **F2 and test report 3's F-1 (`features/rules/oneAtATime.ts`, `useHq.ts`):** only one save runs at a time.
  - Each save uses the revision the last one gave back, held in a ref.
  - Every control that saves is disabled through `saving`.
  - `oneAtATime.test.ts` and J11's `test_one_change_saves_at_a_time` cover it.
- **The scope is unchanged since verification 1:**
  - Rahul's four answers are applied as recorded.
  - The route has its five integration cases, plus the stale, older-version and published cases.
  - Tenancy holds: `loadQuestionnaire` is scoped by trust and hospital, and the procedures a rule names are checked against the trust.
  - The route isn't audited, which is right because it holds no patient data.
- **The A-20 grep** finds nothing new from S40. Every line it prints was already there (format.ts comments, tests, seed, docs). `repocheck` passes.
- **The review's points, weighed:**
  - Already in pending.md from verification 1: the Preview reads the latest version's rules; the Publish window checks the rules from when the editor opened; `audience === "patient"`; unmemoised models; the held wording written three times.
  - New, not blocking, added to pending.md:
    - Test the whole HQ rebuilds its survey after a save or a keystroke in the sample patient's details. Answers are kept, but a multi-page set goes back to page 1.
    - In the clinician view, the Preview's Rules that fire counts the author's answers to clinician-only questions.
    - The seed puts the rules on demo versions that are already published. This is builder's decision 10 and the data is made up, but QA is seeded too.
    - The test page has its own copy of the pass/fail comparison.
  - Not reachable: the review thought `useHq`'s saver could keep the first render's `base` after a hospital switch. It can't, because `switchPath` sends a questionnaire page to the hospital's questionnaire list, which unmounts the page.

### Spot checks in Chrome (signed in through the dev login)

On **MT S40 2225** v4 (draft, rules revision 5), through the API from the page:
- **Alex:**
  - with no `versionId`: **400** "The rules could not be read.";
  - with another `versionId`: **409** "This version has changed since you opened it…";
  - with a stale revision: **409** "Someone else changed these rules…".
- **Cara:** PUT **403**. **Sue (A1 only):** PUT **403**, with the plain message.
- **Bea (Trust B):** GET and PUT **404**.
- **Signed out:** **401**.
- Afterwards the rules were unchanged, still at revision 5.

In the page:
- **One save at a time:** I held the rules PUT for 4 s and unticked the setting.
  - While it was held, Add rule, every ↑ ↓, the pencils, the bins and the test case bins were disabled.
  - I removed `disabled` from ↓ by hand and clicked it. No second PUT went out (1 in total).
  - Afterwards the controls were enabled again and no error showed. I ticked the setting back on (2 PUTs in total), and both rules were unchanged.
- **Sue's view** is "V4 · DRAFT", READ ONLY, with the plain line and only the disabled tick.
- **Test the whole HQ as Alex:** no suggested ASA, and Low risk fires (auto-triage to Ready for admission).
- I left the setting as I found it. Alex is signed in.

### Test results

- `make lint`: clean.
- `make test`: Go ok. Vitest: web **276 passed** (29 files), clinical **391 passed** (26 files). CDK **124 passed**.
- `make test-integration`: every package ok (httpapi 14.5 s).
- `make e2e` (against `make dev`, the shared database): **92 passed, 5 skipped (Cognito-only), 1 failed**, in 14.4 min. J11 passes.
  - The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known oversized audit CSV ("Cannot create a string longer than 0x1fffffe8"). S40 changes no audit code, and the failure is in pending.md.
  - Verification 1's second audit failure (the "N patients" tooltip) passed this time.

### Hand checks for Rahul

- None.

### Commits

1. `S40: ASA rules stored with the version, checked by Go`
2. `S40: the rule matcher, words and whole-HQ test cases in packages/clinical`
3. `S40: the Rules page, Test the whole HQ, the Preview's rules and the Publish line`
4. `S40: J11 starts`
5. `S40: docs` (not `docs/market/`)

**Next:** `/build-session S41`.

## Verification 1 (26 Oct 2026, automatic run)

**Verdict: Needs fixes.** There is one Minor bug, F1: a rules save isn't tied to the version the page loaded. I reproduced it through the API. The fix is small. Nothing else blocks.

The session does what the brief and Rahul's answers ask:
- the platform defaults (a Go file) on new HQs, new versions, the import and the seed;
- `questionnaire_versions.rules` with a revision;
- `PUT …/rules`, where Go checks the shape only and never evaluates;
- the matcher, its words and the whole-HQ test cases in `packages/clinical`;
- the Rules page, Test the whole HQ, the Preview's "Rules that fire", and the Publish window's line and block;
- J11's authoring half.

Answers 2 and 4 depart from the brief's recommendations, and that is recorded in decisions.md and the plan's Status. The new route has all five integration cases plus the stale and published cases. Tenancy holds: `loadQuestionnaire` is scoped by trust and hospital, and procedures are checked against the trust. The route isn't audited, which is right because it holds no patient data. The A-20 grep finds only lines that were already there (comments and the infra tests). `repocheck` passes.

### Findings, worst first

1. **F1 (Minor, reproduced): a rules save goes to whichever version is latest, not to the version the page loaded.**
   - **Where:** `apps/api/internal/httpapi/rules.go:20-51` loads the questionnaire's latest version (`loadQuestionnaire`) and saves by `rules_revision` alone. A new draft's `rules_revision` always starts at 0 (`CreateNextVersion`, `apps/api/db/queries/questionnaires.sql:92-108`).
   - **What happens:** a Rules page (or Test the whole HQ) opened on draft v1 at revision 0 stays open. Meanwhile v1 is published and v2 is created. The stale page's next save is stored on **v2**, while the page still says "V1 · DRAFT".
   - **Reproduced through the API as Alex:** I created "VF S40 stale …" (`4722a88d-acc7-47e9-a302-410b900352fb`), added a set, read v1's rules (revision 0), published v1 and created v2. A PUT with revision 0 and only Low risk got **200**, and **v2 lost High risk**.
   - **Why it matters:**
     - The rules decide auto-triage from S42 on.
     - Any edit someone made to v1 after the page loaded is silently dropped from v2.
     - Publish already guards against this by sending `versionId`; chapter saves are bound to their chapter. This route breaks that pattern.
   - **Fix (test first):** send the loaded `versionId` with the PUT (`useHq.ts` `save`, which both pages use). Go answers 409 when it isn't the latest version, with plain words such as "This version changed since you opened it…" (a published version keeps its "Only draft versions can be changed."). Add an integration subtest in `TestSaveRules` for the case above (publish, new version, stale PUT → 409, v2 unchanged), and a missing or other `versionId` → 400 or 409.
2. **F2 (Minor; fix while there, or pending.md): saving the rule form doesn't set `busy`.** `apps/web/src/features/rules/RulesPage.tsx:52-58` (`saveRule`) calls `save` directly, not `change`. While a Save rule PUT is in flight, the ↑ ↓ buttons, the no-grade tick and the bins stay enabled. A second PUT then carries the same revision and gets a 409 "Someone else changed these rules…", although nobody else did, and that change is lost.
3. **(Not blocking; pending.md) The Publish window checks the rules loaded when the editor opened.** `QuestionnaireEditor.tsx` passes `ruleSetOf(q.rules)`, and Publish sends no rules revision. In the same tab this is fine, because going to Rules and back remounts the editor. With a second tab, though, Publish can be enabled over rules that changed since. This is the same as chapters today; S41/S44's server check closes it.
4. **(Not blocking; pending.md) The Preview reads the rules of the HQ's latest version** (`PreviewPage.tsx:163`, `GET …/questionnaires/{qid}`), not those of the version the previewed chapter belongs to. It matters only when an older version's chapter is previewed by its URL.
5. **(Not blocking; pending.md) Smaller items:**
   - `HqTestPage.tsx:39` filters with `audience === "patient"`, not `isPatientSet`. It's the same today because every chapter from the API has an audience.
   - `publishProblems(sets, rules)` runs every whole-HQ case, building SurveyJS models, on each render of the Publish window; it isn't memoised.
   - The Preview's `setFiredRules` and the test page rebuild models on every answer.
   - The held wording ("no auto-triage: no ASA grade") is written in `HqTestPage.tsx`/`PreviewPage.tsx` as well as in `firedLine`.
   - Procedure names come from this hospital's offered list (already in pending.md).

### Spot checks in Chrome (Alex, Bea, Sue, Cara; signed in through the dev login)

- **E-7 (after test report 2):** on MT S40 2112 v3, I opened "Smoker" on Test the whole HQ and picked Cataract surgery, so Hip fires and "Smoker fails: it expected …" shows. I then saved it as a new case, "VF Cataract smoker". Its stored lines are `Suggested ASA II`, `Rule fires: Low risk (…)`, `Rule fires: Hip (…)`: the ASA line first, then the rules in their order. Older cases keep their sorted lines and still pass, as the build notes say.
- **E-8:** typing "catar" in the Procedure box filters the list down to "Cataract surgery".
- **Permissions through the API** on Trust A's rules:
  - Bea (Trust B): GET and PUT get **404**, through Trust A's URL and through her own.
  - Sue (Hospital A1 only): GET 200, PUT **403** with the plain message.
  - Cara: PUT 403.
  - Signed out: 401.
  - Alex, with a task that has no title: 400 "Rule “T”: Enter the task.".
- **F1**, as above.
- Left behind (made-up, local): the "VF Cataract smoker" case on MT S40 2112 v3 and the "VF S40 stale …" HQ.

### Test results

- `make lint`: clean.
- `make test`: Go ok (51 packages, including `asarules` and `repocheck`). Vitest: web **274 passed** (28 files), clinical **391 passed** (26 files). CDK **124 passed**.
- `make test-integration`: every package ok.
- `make e2e` (against `make dev`, the shared database): **90 passed, 5 skipped (Cognito-only), 2 failed**, in 9.5 min. J11 passes. Both failures are in `test_audit.py`, and neither is S40's, because S40 changes no audit code or audit journey:
  - `test_another_trusts_admin_sees_nothing_of_trust_a` fails on the known ~435 MB CSV: "Cannot create a string longer than 0x1fffffe8".
  - `test_a_list_row_names_ten_patients_and_counts_them_all` fails because the "N patients" tooltip on Cara's newest "Listed episodes" row doesn't appear within 15 s. It failed again when run alone. This is likely the same cause: one row lists thousands of e2e episodes on this long-lived database. Add it to pending.md's audit-size line; it should pass on a reset database or a private stack.

### Hand checks for Rahul

- None.

### Suggested commits (after the fix)

1. `S40: ASA rules stored with the version, checked by Go`: migration 00031, `internal/asarules`, the queries and sqlc output, `rules.go`, router, create/new version/import/seed; `asarules_test.go`, `rules_integration_test.go`, `TestSeedTwice`.
2. `S40: the rule matcher, words and whole-HQ test cases in packages/clinical`: `rules.ts`, `respondent.ts` (`hqResult`), `publish.ts`, `index.ts`; `rules.test.ts`.
3. `S40: the Rules page, Test the whole HQ, the Preview's rules and the Publish line`: `features/rules/`, `pages/hospital/Rules.tsx`, `routes.tsx`, editor, Preview, PublishModal; `routes.test.ts`.
4. `S40: J11 starts`: `e2e/data/rules-hq.json`, `test_j11_triage.py`, `e2e/README.md`.
5. `S40: docs`: session folder, decisions, pending, plans, sessions README, CLAUDE.md, `apps/web/AGENTS.md` (not `docs/market/`).

**For the builder:** "Verification found problems: read `docs/sessions/S40-asa-rules/verify.md`, fix them test-first, then ask me for a retest."
