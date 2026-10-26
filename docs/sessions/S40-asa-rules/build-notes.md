# S40 build notes: ASA rules on the HQ

Builder, automatic run, 26 Oct 2026. Rahul's answers ([answers.md](answers.md)) replace the brief's questions where they differ:
1. **A new version of an older questionnaire gets the default rules** (as the brief recommended).
2. **A patient whose answers give no ASA grade: a setting on each HQ decides whether they can be auto-triaged, on by default** (not the brief's (a) alone).
3. **Rules run once, when the patient sends the HQ** (as recommended).
4. **Rules are tested with whole-HQ test cases** (not the brief's recommendation: the per-set test cases are unchanged, and the HQ gets test cases of its own).

**Nothing acts on the rules yet** (S42 and S43 do); they're stored, shown, previewed and tested.

## What was built

### Go

- **Migration 00031** (`questionnaire_versions.rules jsonb`, `rules_revision int`): one JSON object per version: `{noGradeAutoTriage, rules: [...], testCases: [...]}`. NULL only for versions published before S40 (no rules). The migration gives **existing drafts** the defaults (a copy of the file's JSON as on 26 Oct); published and retired versions keep none.
- **`internal/asarules` (new):** `defaults.json` (embedded): **Low risk** (ASA ≤ 2 and no red flags → auto-triage to Ready for admission) and **High risk** (ASA ≥ 3 → task "Book anaesthetic review" for the anaesthetist team, flag "Anaesthetic review", require the anaesthetist's ASA), setting on, no test cases. `Check(raw)` reads a set with unknown fields refused and returns the set to store (names and titles trimmed, each action with only its own fields) or a plain problem: a name for each rule (unique, any case), the ASA compared `>=`, `<=` or `=` with 1–6, a band with its score and at least one band, a code with its set and code, at least one procedure, at least one action, known kinds, **auto-triage once per rule, to Ready for admission, Ready for POA or POA complete**, a task as `episode.TaskInput` ("Rule “High risk”: Enter the task."), a flag's name, a suggested status from Ready for POA, POA complete, On hold, Not ready for admission, Ready for admission; ids `[A-Za-z0-9_-]{1,40}`, unique; at most 50 rules and 100 test cases, 512 KB; test cases with a name, the sample patient's age 0–120, answers an object. `ProcedureIDs()` lists the procedures named (rules and test cases).
- **Routes:** `GET …/questionnaires/{qid}` now sends `questionnaire.rules` and `rulesRevision`. **`PUT /api/o/{oid}/h/{hid}/questionnaires/{qid}/rules`** `{rules, revision}` (`internal/httpapi/rules.go`): `canEdit` (403 for those who can't change the HQ's chapters, 409 on a published version), `asarules.Check` (400 in its words), the procedures must be the trust's (400), then saved only if the revision is current and the version still a draft (`SaveVersionRules`; 409 stale, or 409 "Only draft versions can be changed."). Answers `{revision, rules}` as stored. Not audited (no patient data).
- **Defaults given:** `createQuestionnaire` (`CreateVersion` with `rules`), `createVersion` (`CreateNextVersion`: `coalesce(prev.rules, defaults)`, so the published version's rules are copied, or the defaults for an older one), `cmd/import-hq`, and the seed's demo HQs (`INSERT … ON CONFLICT DO UPDATE SET rules = coalesce(…)`, so an already-seeded database gets them too).
- **Tests:** `asarules_test.go` (the defaults pass their own check; each refusal in plain words; every kind of when and action kept and trimmed; the rule limit). Integration (`rules_integration_test.go`): **TestNewHQGetsTheDefaultRules** (the GET sends the two defaults, revision 0; a clinician reads them; another trust 404), **TestNewVersionCopiesTheRules** (a published version's own rules; an older one's → the defaults, the published one untouched), **TestSaveRules** (happy path with rows: trimmed, revision 1, the GET; no sign-in 401; a clinician and a one-hospital super clinician 403; another trust 404, and a rule naming another trust's procedure 400; bad shapes 400 in plain words, a bad id 404, revision unchanged; stale 409; published 409). `TestSeedTwice` checks every demo version has the two default rules after both runs.

### `packages/clinical` (Vitest, test-first; TypeScript for Rahul to review)

**`rules.ts` (new).** What it does, in plain terms:
- **Types:** a `RuleSet` is `{noGradeAutoTriage, rules, testCases}`; an `AsaRule` is `{id, name, when, then}`; `when` has `asa: {op, grade}` plus optional `noRedFlags`, `band: {score, bands}`, `code`, `procedures`; `then` is a list of actions (`autoTriage` with a target, `task` with title and team, `flag` with a name, `requireAnaesthetistAsa`, `suggestStatus` with a status). The lists of targets, statuses and teams (with their labels) copy Go's.
- **`firedRules(set, input)`** is the matcher. `input` is what `computeOutputs` (one set) or `hqResult` (the HQ) gives: the suggested ASA, the disclosures produced, the scores, and the episode's procedure. Each rule fires when the suggested ASA compares as it says (**no grade counts as ASA I**), and every extra holds: no disclosure with a red flag; the score's band is one listed; a disclosure has the code; the procedure is one listed (no procedure, as in the Preview, never matches). Every rule that matches fires, in order. **With the HQ's setting off and no grade, a rule's auto-triage is taken out** (`held`), its other actions stay.
- **Words:** `ruleWords` ("When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission"), `actionWords`, `firedLine` ("Rule fires: Low risk (auto-triage to Ready for admission)"; held: "… (no auto-triage: no ASA grade)"), `autoTriageSummary` for the Publish window ("This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I."; or "… are never auto-triaged."; or "This version doesn't auto-triage anyone."). `ruleNames` gives scores' and procedures' names; a missing one shows its id.
- **What an HQ offers a rule:** `codesIn` (every code on a disclosure: options, the question, bands, grid cells) and `scoresWithBands` (Calculations with bands), from every set.
- **The whole HQ:** `hqRuleResult(sets, ruleSet, answersByKey, patient, procedure?)` splits the answers by key into each set (`answersBySet`; **only the patient's sets get answers**), runs `hqResult` (below) on every set **in the clinician's view** (so clinician-only scores worked out from the patient's answers count), and gives the sets shown, the suggested ASA, the rules that fire and the test case lines (`hqLines`: "Suggested ASA II" or "No suggested ASA", then each `firedLine`, sorted). `runHqTestCase` compares them with a case's saved lines (missing / unexpected), as per-set test cases do. `ruleProblems`: a rule whose score, band or code the HQ no longer has ("…can't fire").
- `ruleSetOf(raw)`: the stored JSON as a `RuleSet`, or null.

**Changed:** `respondent.ts` **`hqResult`** (new): S39's `hqSuggestedAsa` generalised to give the shown sets, every disclosure (with its set), the scores and the suggested ASA; `hqSuggestedAsa` now calls it. `publish.ts` **`publishProblems(sets, rules?)`**: with rules, adds `ruleProblems` and each failing whole-HQ test case, under the set name "Rules" ("Rules · Test case “Smoker” fails"). Without rules, as before.

**Tests:** `rules.test.ts` (15), reading Go's `defaults.json` itself so both sides use one file: the defaults on both sides of ASA II/III; a red flag stops Low risk (amber doesn't); no grade → Low risk fires and auto-triages; the setting off → held, its line; `>=`, `<=`, `=`; the band, code and procedure extras (and none in the Preview); several rules fire in order; the words of the defaults and of every when and action; the Publish line (on, off, none); the codes and scores offered; the whole-HQ result ignores an answer to a clinician set, and counts a clinician-only score worked out from the patient's answers; a case passes, then fails when the ASA limit or the target changes; `publishProblems` with a failing case and `ruleProblems`.

### `apps/web`

- **Editor header:** a **Rules** button (for everyone who opens the editor) to `…/questionnaires/:qid/rules`.
- **Rules page** (`features/rules/RulesPage.tsx`, `RuleForm.tsx`, `useHq.ts`; route `rules`): the setting ("Patients whose answers give no ASA grade (usually fit and well) count as ASA I, so the rules can auto-triage them"); the rules numbered, in plain words, with ↑ ↓, Edit and Delete ("Delete “X”?" / Keep it); **Add rule**. The form: name; "the suggested ASA [is at most | is at least | is exactly] [I–VI]"; ticks for "and there are no red flags", "and a score is in a band" (score and bands from the HQ), "and a disclosure has the code" (codes on the HQ's disclosures), "and the procedure is one of" (this hospital's procedures); the actions, each a kind and its fields, **+ Add action** and a bin. Every change saves the whole set at once; Go's refusal shows in the form (or above the list), and nothing changes. A warning lists rules that can't fire. **Test cases for the whole HQ:** each with pass/fail (run after every change), its lines, Open and Delete; **Test the whole HQ**. Read-only (no buttons, the tick disabled, a line why) for those who can't change the HQ and on a published version; a version published before S40 says it has no rules.
- **Test the whole HQ** (`HqTestPage.tsx`, route `rules-test`, `?case=<id>`): the patient's sets as tabs ("not shown" on a set a condition hides), each set's survey as the patient sees it (with the sets before it, for scores across sets); a sample patient (name, age, sex) and a procedure; **What the rules see**: the suggested ASA with its reasons and the rules that fire with their actions; **Save as test case** (name). An opened case says whether it passes, and what it expected when not. Clear answers.
- **Preview:** under Suggested ASA, **Rules that fire** "for this Question Set's answers alone" (None when none), when the version has rules.
- **Publish window:** the rules' problems and failing whole-HQ test cases block it; one line says what the version auto-triages.

### E2E

- `e2e/data/rules-hq.json`: About you (smoker → ASA II; chest pain on walking → red flag), Your heart (heart failure → ASA III), Nurse checks (clinician).
- **J11** `test_j11_triage.py` (authoring half): the two defaults in words; Low risk's target to Ready for POA (kept after reload) and back; a task with no title refused in plain words; Test the whole HQ as a smoker → ASA II, Low risk; saved "Smoker", passing on the Rules page with its lines; the Preview's Rules that fire, and a red flag stopping Low risk; the Publish window's line; Low risk lowered to ASA I → the case fails, Publish blocked ("Rules · Test case “Smoker” fails"); Sue Super read-only. `e2e/README.md` updated.

## Decided by the builder (automatic mode)

1. **Rules and their whole-HQ test cases are one JSON object in `questionnaire_versions.rules`**, saved whole with one revision. Simplest: the Rules page and the test page save the same thing, and S41 sends one object to the evaluator.
2. **The no-grade setting is about auto-triage only:** no grade counts as ASA I; off, a matching rule's auto-triage is taken out and its other actions stay. (Rahul's words: "whether they can be auto-triaged".)
3. **Rules look at the patient's answers as the clinician's view shows them before any validation** (S41's brief: "the same result the clinician's view shows for the patient's answers before any validation"): every Question Set, with answers only in the patient's sets. So a **clinician-only score worked out from the patient's answers** (STOP-Bang's) can drive a rule, and a clinician set's own questions never count (the patient doesn't answer them). The rule form offers every set's codes and banded scores; `ruleProblems` checks the whole HQ.
4. **"No red flags" means no red flag** (the brief's and A-8's words); amber doesn't stop it. S42's fixed safety checks add "no amber or red flags" for auto-triage.
5. **Whole-HQ test cases keep answers by question key** (unique in an HQ since S23), so they carry over to new versions, whose chapter ids change. They're always on the patient's view.
6. **A test case line names the rule and its actions in words**, so a changed limit, target, task, flag or name fails it (the brief: "a changed rule makes a saved case fail"). The suggested ASA is a line too (ASA-04).
7. **Whole-HQ test cases run by themselves** on the Rules page after every change (no Run all), and on Publish.
8. **Roman numerals** in the rules' words ("is I or II", "ASA I–II"), as the app writes ASA everywhere else; the brief's examples used 1–2.
9. **A rule that can't fire blocks Publish** (its score, band or code is gone from the HQ), like a condition that can't work.
10. **Existing drafts got the defaults in the migration** (a copy of the JSON in the SQL); published and retired versions stay without rules. The seed gives its demo versions the defaults, even though they're published: they're made-up data, and S42's J11 needs a seeded HQ with rules.
11. **The procedures offered on the Rules page** are this hospital's (`GET /api/o/{trust}/procedures?hospital=`), as New episode lists them; Go checks every procedure named is the trust's.
12. **Not audited:** the rules route has no patient data (the brief).

## Differences from the plan or brief

- The brief's question 4 answer (whole-HQ test cases) replaces "a set's test case also lists the rules its answers alone would fire": per-set test cases are unchanged.
- Rules are stored in their own column, not "the version JSON" (a version has none); the plan's Status section says so.
- Go's publish still checks only question keys (S41 adds the rules through the evaluator). pending.md has it.

## Test results (26 Oct)

- `make lint`: go vet (with the integration tag) and tsc in every workspace: clean.
- `make test`: Go all ok; Vitest: web 274 passed (28 files), clinical 390 passed (26 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 19 s).
- `make e2e` against a private stack on :3010/:8090 (the conductor's `make dev` left alone): **90 passed, 5 skipped, 1 failed**: `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, which reads Trust A's whole audit CSV, now about 435 MB on the long-lived local database ("Listed episodes" rows name every episode listed), and Playwright can't hold it ("Cannot create a string longer than 0x1fffffe8"). Not caused by S40 (no audit change); a reset local database passes it. In pending.md. J11 passes.

## Test report 1 (23 of 24 passed; F-1 and the exploring findings fixed, 26 Oct)

- **F-1 (Minor): the Preview's "Rules that fire" in the Patient view ignored clinician-only scores** (a new Calculation is Clinical by default), so it said a rule wouldn't fire when it would on send. **Fixed:** `setFiredRules(rules, content, answers, patient, earlier)` (`rules.ts`) works the set's answers out in the clinician's view, as the rules see them on send, whichever view the Preview shows; the Preview uses it. Tests first: `rules.test.ts` ("the Preview's rules use the clinician's view…"), and J11's second test (`test_the_preview_fires_rules_on_a_clinician_only_score_in_the_patient_view`; `rules-hq.json` gained a clinician-only Snore score with bands).
- **E-1:** "is at most VI" now reads "is I to VI" (and "at most III" "is I to III"; "at least I" "is I to VI"). Unit test.
- **E-2: saving a case again under its name replaces it** (same id and place), and opening a case fills in its name, with the button **Replace test case** and a line saying so. **Go refuses two test cases of one name** ("Two test cases are called “Heart”: give each its own name."; `asarules_test.go`). An HQ that already holds two cases of one name (the tester's MT S40 2112 v2) refuses every save until one is deleted (deleting is a save that removes the duplicate, so it works).
- **E-3:** a test case's bin asks first ("Delete “…”? Delete / Keep it"), as a rule's does. **E-4:** Enter in the name box saves (a form). **E-5:** the form's refusal goes once the rule is changed. J11 covers E-2–E-4.
- **E-6** (a question in a Clinician set shows a PATIENT badge in the editor): not S40's; in pending.md for a later look.
- **Hand check for Rahul:** R-05 (a patient sends an HQ) at a real 390×844 window; the tester's Chrome couldn't go below 500 px.
- After the fixes: `make lint`, `make test` (clinical 391 passed), `make test-integration` all pass; J11 (2 tests), J1, J2, J10, Video, Information and Trusts journeys pass against a private stack.

## Test report 2 (all 14 passed, 26 Oct)

- **E-7 (cosmetic):** a whole-HQ test case's lines were sorted, so the ASA line came first or last. Now the ASA line comes first, then the rules in their order (`hqLines`; `rules.test.ts` first). Cases are compared as a set, so saved ones still pass.
- **E-8 (cosmetic):** Test the whole HQ's Procedure box now filters as you type (`searchable`). New episode's Health questionnaire box (S13/S19's) is the same plain select: in pending.md.
- R-05 at phone size was covered by J3 on an emulated iPhone 13, so no hand check is left.

## Verification 1 (Needs fixes: F1, F2; fixed 26 Oct)

- **F1 (Minor): a rules save went to whichever version was latest.** A page opened on draft v1 (revision 0) saved onto v2 after v1 was published and v2 made, since a new draft's revision starts at 0 again. **Fixed:** the PUT sends the `versionId` the page loaded (`useHq.ts`, used by both pages); Go refuses a missing one (400, "The rules could not be read.") and another than the latest (409, "This version has changed since you opened it. Reload and try again.", as publish says). Test first: `TestSaveRules` "a page opened on an older version: 409, the new draft unchanged" and a body with no version (400).
- **F2 (Minor): Save rule didn't set busy,** so ↑ ↓, the tick and the bins could send a second save with the same revision (a false 409). **Fixed:** `saveRule` sets busy; the confirm Delete buttons are disabled while busy too. No automated test (a race on a click in flight); a hand check isn't needed.
- The non-blocking review points are in pending.md ("From S40's verification 1").

## Test report 3 (13 of 14 passed; F-1 fixed, 26 Oct)

- **F-1 (Minor): an open rule form's Save rule (and Add rule) stayed enabled while another change was saving,** so a quick click sent a second save with the old revision: a false "Someone else changed these rules", and the new rule lost. **Fixed at the source:** `useHq`'s save goes through **`oneAtATime`** (`features/rules/oneAtATime.ts`): while a save is in flight another sends nothing and returns "Wait until the last change has saved, then try again."; each save uses the revision the last one gave back (a ref, not a render's copy). The hook's `saving` is the page's `busy`: the rule form's Save rule, Add rule, every row control, the tick, both confirm Delete and Keep it buttons, and Test the whole HQ's Save are disabled while it's true. Tests first: `oneAtATime.test.ts` (Vitest: a second save refused while one is in flight, allowed after; free again after an error) and J11's `test_one_change_saves_at_a_time` (the setting's PUT held by Playwright: the form's Save rule and Edit are disabled and only one PUT goes out; once it lands the rule saves, both kept after a reload). The e2e test fails with RuleForm's fix taken out.
- **Exploring:** "Keep it" during a held delete is now disabled too. The test page and the Preview show SurveyJS's own "The survey doesn't contain any visible elements." for an empty Question Set: in pending.md.
- After the fix: `make lint`, `make test` (web 276, clinical 391), `make test-integration` pass; J11 (3 tests) and J2 pass on a private stack.

## Test report 4 (all 14 passed, 26 Oct)

- Report 3's F-1 confirmed fixed (held saves, deletes and reorders with forms open; no false 409). No test rounds left, so no code changed after it.
- Its exploring notes (Add rule with an Edit form open drops the unsaved edit; Enter in the rule's Name box; Cancel during a save) are in pending.md.

## Suggested commits

1. **S40: ASA rules stored with the version, checked by Go** — migration 00031, `internal/asarules`, the queries, `PUT …/rules`, defaults on create, new version, import and seed; `asarules_test.go`, `rules_integration_test.go`, `TestSeedTwice`.
2. **S40: the rule matcher, words and whole-HQ test cases in packages/clinical** — `rules.ts`, `hqResult`, `publishProblems(sets, rules)`; `rules.test.ts`.
3. **S40: the Rules page, Test the whole HQ, the Preview's rules and the Publish line** — `features/rules/`, the routes, the editor, the Preview, the Publish window; `routes.test.ts`.
4. **S40: J11 starts** — `e2e/data/rules-hq.json`, `test_j11_triage.py`, `e2e/README.md`.
5. **S40: docs** — build notes, test plan, decisions, pending, plans, sessions, CLAUDE.md, AGENTS.md.

## Hand checks for Rahul

- None (test report 2: R-05 at phone size is covered by J3 on an emulated iPhone 13).

## Verification 2 (Verified, 26 Oct)

- F1, F2 and test report 3's F-1 confirmed fixed (API and Chrome). The review's smaller points are in pending.md ("From S40's verification 2"). `make lint test test-integration` pass; `make e2e` 92 passed, 5 skipped, 1 failed (the known audit CSV size, not S40's).
