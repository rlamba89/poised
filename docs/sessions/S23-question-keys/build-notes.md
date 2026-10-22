# S23 build notes: question keys in the editor

Builder, automatic run (`/run-sessions S23 S26`), 22 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md). Two differ from the brief, and the build follows the answers:
- **Q2: a published keyed question's options are frozen completely**, values **and** wording: no option added, removed or reworded (the brief recommended "add and reword, never remove").
- **Q5: the label counts matches by key and by exact wording, and says how** ("by link" / "by same wording"); the walk returns each question's text and option texts.

The rest as recommended: keys found by walking the trust's published and retired chapter JSON on each request (no table); keys unique across an HQ (editor and publish); "Always ask fresh" on patient questions.

## What was built

### Go: `internal/keys` (new, unit-tested)
- `Walk(content)` lists every element of a chapter except groups (panels), at any depth, dynamic-panel children included (`InList` names the dynamic panel). Each `Question` has name, SurveyJS type, **kind of answer** (single, multiple, boolean, text, number, date, dates, list, or the type: bmi, medication…), unit, **text** (default language), **options** (value + default-language text; SurveyJS's "Other" and "None" items count), a repeating list's **column texts**, `Answerable` (false for statements, calculations, profile, image), `ClinicianOnly` (on the question, a group or page around it, or the set's audience), `AlwaysFresh`.
- `WalkSets` walks a version's sets in order and marks a clinician set's questions clinician-only.
- `Duplicates`: a problem for each extra use of a key in the HQ, naming both sets ("…used “A” in “Set 1” and “B” in “Set 2”…", or "twice in “Set 1”").
- `LockProblems`: compares each draft question's options with each **published** occurrence of its key (one per questionnaire: the newest published or retired version that holds it) and reports every option **added, removed or reworded**, naming the question, the option and where it was published. Reordering options, translating them, and rewording the question itself are allowed.
- `Normalize`, `SameWording`, `RepeatedTexts`, `Matches`: the PRE-05 matching. Same key always matches; otherwise the same normalised text (case, punctuation and extra spaces ignored), the same kind of answer (and unit for numbers), the same option texts in any order (choices), the same column texts (repeating lists). A text used twice in either questionnaire never matches by wording; a question with no text never does. Only top-level, answerable, non-clinician-only questions take part. S25's pre-fill can reuse these.

### Go: routes
- **Publish** (`POST …/questionnaires/{qid}/publish`) now runs in a transaction under the trust's advisory lock (`LockOrgQuestionnaires`), so two publishes in one trust can't each pass the check against the other's old options. Between loading the chapters and publishing, it runs `Duplicates` and `LockProblems` against the trust's published and retired versions, and refuses with **422** `{error, problems: [{set, message}]}`.
- **`GET /api/o/{oid}/h/{hid}/questionnaires/{qid}/question-keys`** (new): for the questionnaire's latest version, `questions` (each patient question's matches among the trust's *other* questionnaires, `[]` for a new one) and `published` (each key's options as published in the trust, the questionnaire's own versions included). Super clinician and above **at that hospital** (so Sue Super, a hospital super clinician who can view the editor, sees the label); not audited (no patient data).
- Queries `ListVersionContents`, `ListTrustPublishedContents` (chapters.sql).

### TypeScript (`packages/clinical`, for Rahul to review)
- `keys.ts` (new): the same two checks as Go, word for word, so the Publish window shows them early: `duplicateKeyProblems(sets)` and `optionLockProblems(sets, published)` (published = Go's `published` map). `keyOptions` reads options as Go does. `canAskFresh(el)` says whether an element may have the switch (not clinician-only, not a group, statement, calculation, profile or BMI). `keyPlace(doc, name)` says whether a question is inside a repeating group or a clinician-only group/page (no label there).
- `publishProblems(sets, published = {})` adds the two key checks after the existing ones.
- `alwaysFresh` is a hidden SurveyJS property on every question (`questionTypes.ts`), and is in `fields.ts`'s `KEEP` list, so a type change or a date-settings change keeps it.

### Web editor (`apps/web/src/features/editor`)
- `questionKeys.tsx` (new): `useQuestionKeys` fetches the route when the editor opens and again each time all changes are saved (so a new or reworded question gets its label without a reload); `PrefillLabel` draws **"Pre-fills from N other HQs"** (a teal badge; click → a scrollable list "Name · by link / by same wording") or **"New question: won't pre-fill"**. Clinicians (below super clinician) don't fetch it and see no label.
- `ElementCard.tsx`: the badge next to PATIENT. **"Always ask fresh"** (orange) replaces the label when on. None on clinician-only content, in a clinician set, or inside a repeating group.
- `SettingsPanel.tsx`: the **"Always ask fresh"** switch under Clinical, where `canAskFresh` and the place allow.
- `PublishModal.tsx`: passes `published` to `publishProblems`; a 422 from Go shows Go's problem list in the same box.

## Decided by the builder (automatic mode)
- **The route is under the hospital and the questionnaire** (`…/h/{hid}/questionnaires/{qid}/question-keys`), not the brief's `/api/o/{oid}/question-keys?questionnaire=`: the existing `loadQuestionnaire` scoping and 404s apply, and a hospital-scoped super clinician (who views the editor) is allowed by the usual hospital role. It still lists all of the trust's questionnaires (pre-fill works across the trust's hospitals). A hospital's own questionnaires can't be made in the UI today, so no names leak across hospitals in practice.
- **Which "published" occurrence a draft is checked against:** for each questionnaire of the trust (this one included), the newest published or retired version holding the key. Older versions of the same questionnaire are consistent with it by construction from now on; content published before S23 may differ between versions, and checking every version could make a key impossible to publish.
- **The lock applies to every keyed question with options,** clinician-only ones and dynamic-panel children included (their answers are stored too). A published question without options that gains some (or loses all, e.g. Select One → Text) is a change. Grid rows/columns, rating scales and Yes/No switches (`boolean`) have no "options" here.
- **Option order may change; option text is compared exactly (trimmed) in the default language;** translations may change.
- **Duplicates** count every element but groups (`panel`), at any depth: questions, statements, calculations and dynamic-panel children.
- **The editor's placeholder texts never match by wording** ("Text question", "Select One question", "Yes / No question"… as `newElement` names a new question): nobody chose that wording. Found by the e2e run: a new "Text question" matched an earlier run's published, untouched one. ("If in doubt, don't fill.")
- **Wording matches** need the same kind of answer *and* the same unit for numbers; a repeating group and a repeating list (matrixdynamic) never match each other.
- **The label is refreshed after every save** (the brief said "fetched once when the editor opens"), so new and reworded questions are right without a reload. The route is cheap at the trust's size.
- **"Always ask fresh" is offered on a repeating group (dynamic panel)** but not on questions inside one (rows are copied whole, S25 answers Q3), nor in a clinician set or under a clinician-only group or page.
- **Integration test fixtures:** `basic.json` and `workflow.json` both use `q_smokes` with different meanings (a Yes/No switch vs a choice with options). Two `trusts_integration_test.go` cases published one where the other was already published in the trust, which the lock now rightly refuses: one now drafts `workflow.json`, the other empties its draft's content (commented there).

## Tests
- `apps/api/internal/keys/keys_test.go`: Walk (every type, depth, clinician-only inheritance, options, columns), WalkSets, Duplicates, LockProblems (removed, added, reworded, reordered, translated, choice ↔ no options, inside a repeating group, two sources), Normalize, SameWording, RepeatedTexts, Matches.
- `apps/api/internal/httpapi/question_keys_integration_test.go`:
  - `TestQuestionKeys`: happy path (by key, by wording, a new question, clinician-only excluded, published options; trust B's identical HQs never count), a hospital super clinician, no sign-in, wrong role (clinician at the hospital and trust-wide), another trust (404 both ways), invalid ids.
  - `TestPublishKeyChecks`: same HQ's new version with a removed / added / reworded option refused (version stays draft); rewording the question and reordering options publishes; a different HQ of the trust refused; another trust's HQs never lock; a key in two sets refused naming both.
  - `TestNewVersionKeepsKeys`: every key survives New version.
- `packages/clinical/src/keys.test.ts`, `fields.test.ts` ("Always ask fresh" survives type and date changes and a SurveyJS save), `publish.test.ts` (an older case used one doc twice, now a duplicate key; given its own keys).
- `e2e/journeys/test_question_keys.py`: the label "by link" and "New question", Always ask fresh on/badge/reload, a reworded published option refused in the Publish window, put back and published.

## Results
- `make lint`: passed.
- `make test`: Go ok, Vitest 204 (web) + 130 (clinical), CDK 124 passed.
- `make test-integration`: every package ok (22).
- `make e2e` (own instance: database `sj_s23`, API :8093, Vite :3103, `E2E_BASE_URL=http://localhost:3103`): **58 passed, 5 skipped** (the QA-only Cognito journeys). The first full run failed the new journey once: a new "Text question" matched an earlier run's published one by wording, which led to the placeholder safeguard above; after it, all passed.

## Suggested commits
1. `S23: question keys — internal/keys (walk, duplicates, option lock, wording matches), publish refuses key problems (422), GET …/question-keys, Always ask fresh, the editor's pre-fill label (PRE-05)` with all the tests above, plus the session docs.

## Test report 1 (22 Oct): all passed
17 passed, 0 failed. Blocked by the tester's tools, not failures:
- **R-05** (the session's permission check refused changing the seeded Demo pre-op HQ). Covered by `TestPublishKeyChecks`'s "the question reworded and its options reordered publishes" and `TestPublishQuestionnaire`'s new-version publish.
- **R-02 at 390×844** (the window wouldn't resize). The patient's view has no S23 change; the desktop check passed and the phone journeys (J3, J7) pass in `make e2e`.

The tester's "found while exploring" items are recorded in [pending.md](../../pending.md) §3 ("From S23's test report 1"); none needed a code change. T-06 step 4 of the test plan is corrected (Trust B's demo HQ may have no Smoking page).

## Hand checks for Rahul
- None: R-02 at phone size passed in test report 3 (a 390 px frame), and R-05 in test report 4.

## Verification 1 (22 Oct): fixes

All nine findings in [verify.md](verify.md), test-first. Item 1 follows Rahul's answer "after verification 1" in [answers.md](answers.md): **freeze every kind of answer**.

1. **The lock freezes the whole answer** (`keys.LockProblems` → `answerChanges`): the kind of answer (single, multiple, Yes/No switch, text, number, date, list of dates, repeating list, grid, rating, signature, file upload…), the unit, the options (values and wording), a Yes/No switch's labels, a grid's rows and columns, a rating's scale (From/To/step, or its own values), a repeating list's columns (key, text, kind, options). A changed kind is reported alone (the rest follows from it). Order may still change, and so may translations and the question's text. A key published on a statement or calculation locks nothing. The message is now "“Q”: <change>. A published question's answers can't change: add a new question instead (published in …)." Tests: `TestLockFreezesTheWholeAnswer` (13 changes, 3 non-changes), `TestPublishKeyChecks` ("kind of answer or unit").
2. **Signatures, file uploads and the BMI block never take part in pre-fill** (`keys.source`; `canAskFresh` in TypeScript agrees): no label, no switch. The BMI block is included because its weight changes and, being a fixed block, an author can't set "Always ask fresh" on it (*builder's default*, "if in doubt, don't fill"; this also settles item 7's BMI difference). Test: `TestOneOffAnswersNeverPrefill`.
3. **Wording matches compare everything the answer can hold:** options, a grid's rows and columns, a rating's scale, a list's columns with their kinds, and the unit for any kind. Test: `TestSameWordingGridsAndRatings`.
4. **The walk is lenient** (`obj` helpers): a field of an unexpected shape reads as empty and non-object items are skipped, so only content that isn't a JSON object is an error. Plain-string rows, columns and rating values are both value and text. Tests: `TestWalkOddShapes`, `TestWalkAnswerShapes`, and the integration case "a published chapter with plain-string grid columns doesn't stop publishing".
5. **The Publish window always has current key problems:** they are now **Go's**, from `GET …/question-keys` (`problems`, for a draft), fetched again as the window opens and after a 422; the window says "Checking the Question Sets…" until they arrive. *Builder's choice:* this replaces the TypeScript copies of the two checks (`duplicateKeyProblems`, `optionLockProblems`, removed), which would otherwise have had to copy the whole answer comparison; one implementation in Go, and less TypeScript for Rahul. The route no longer returns `published`, which also removes the ~1 MB response size noted in pending.md. The draft is always saved before Publish can be clicked, so Go's view of it is current.
6. **Publish reads the draft once** (`ListVersionContents`, also used for the "no Question Set" check).
7. **TypeScript and Go agree:** the TypeScript option reading is gone (item 5); `canAskFresh` and Go's `source` exclude the same types.
8. The pre-existing race (a save between the checks and the publish) is recorded in pending.md.
9. The route listing all of the trust's questionnaire names to a hospital-scoped super clinician is recorded in decisions.md.

**Results after the fixes:** `make lint` passed; `make test` (Go 18 packages ok, Vitest 204 web + 120 clinical, CDK 124) passed; `make test-integration` 22 packages ok; `make e2e` on a fresh own instance (`sj_s23`, :8093/:3103): **58 passed, 5 skipped**. The test plan has a retest note at the top and new cases T-14 to T-17.

## Test report 3 (22 Oct): R-05 failed; fixes

**R-05** (an unchanged new version of the seeded Demo pre-op HQ refused) came from old local data, not the rule: three **"S02 test HQ …"** questionnaires, made on 8–9 Oct by the S02 manual-test helper `docs/sessions/S02-workflow-route-tests/make-test-hq.js` (before the lock), published `q_diabetes` as a Yes/No switch and as choices with other values, and `q_angina` in two shapes. Rahul kept the strict rule (`answers.md`, "After test report 3"). So:
- **Cleaned the local `sj` database** (disposable, decisions.md §1): deleted the three S02 test HQs (no episodes), the tester's v2 draft of the Demo pre-op HQ and the v3 draft of T23 Fresh 42. A scan of every trust's published keys (a throw-away Go program, deleted) found no other key published in shapes that disagree. Checked through the API on `sj`: a new version of Trust A's Demo pre-op HQ has **no key problems** (that draft was then removed again, so the Demo HQ is v1 only).
- **Stopped it recurring:** the helper script uses API paths from before S08 and can't run; it's now marked obsolete with the reason. The e2e journeys and the seed are the other things that publish into `sj`: a new unit test, `cmd/seed/keys_test.go` (`TestSeedAndE2EQuestionnairesAgreeOnKeys`), fails if `demo-hq.json` and any `e2e/data/*.json` publish a shared key with different answers.
- **One problem per question when the published copies disagree** (`keys.LockProblems` groups the published copies by answer, `publishedShapes`): "“Q”: the trust has already published this question with different answers (in A v1; B v1 and C v2), so it can't be published again as it is. Add a new question instead, or ask for the old questionnaires to be tidied up." Tests: `TestLockWhenPublishedShapesDisagree`, and the integration case "a key published in shapes that disagree is one problem".
- **Found 1:** the editor's default names **"Group"** and **"Section"** (and "Calculation", "Upload a file", "Please sign below") are placeholders too; a repeating "Group" with a "Text question" no longer matches by wording (`TestSameWording`).
- **Found 2:** the pre-fill badge is a **button** (keyboard focus; Enter opens the list; Escape closes it wherever the focus is). The e2e journey checks it.
- Found 5 (the test plan's page name) is corrected; found 6 (the delete dialog says "options" for any question) is in pending.md; found 7 was already there.

**Results after these fixes:** `make lint` passed; `make test` (Go 20 packages, Vitest 204 web + 120 clinical, CDK 124) passed; `make test-integration` 23 packages ok; `make e2e` on a fresh own instance (`sj_s23`, :8093/:3103): **58 passed, 5 skipped**. The local `sj` database is migrated, seeded and cleaned as above (the Demo pre-op HQ is v1 published only, and its new version has no key problems).

## Test report 4 (22 Oct): all passed
22 of 22 (T-01 to T-17, R-01 to R-05), R-05 included. Its "found while exploring" items are as designed or not S23 (the PATIENT chip in a clinician set and the delete dialog's "options" are in pending.md). **S23 is tested**; next is `/verify-session S23`.

## Suggested commits (updated)
1. `S23: question keys — internal/keys (walk, duplicates, whole-answer lock, wording matches), publish refuses key problems (422), GET …/question-keys (matches and problems), Always ask fresh, the editor's pre-fill label (PRE-05)`, with its tests: Go unit (`internal/keys`, `cmd/seed/keys_test.go`) and integration (`question_keys_integration_test.go`, the adjusted `trusts_integration_test.go`), Vitest, the e2e journey `test_question_keys.py`.
2. `docs: S23 tested — build notes, test plan and reports, answers for S23–S26, verification, decisions, pending, plan and session status` (includes the conductor's recording of the S23–S26 answers, and the obsolete note on the S02 helper script).

## Verification 2 (22 Oct): fixes

Rahul's answer "after verification 2" (`answers.md`): a date's format and a text question's input format are part of the frozen answer and of the wording match.
1. **`Question.Format`** (`keys.formatOf`): a date's format (day, month and year / month and year / year only, read as `fields.ts` `dateSettingsOf` does) or a text question's input format (any text, email address, "the pattern “…”" from its regex validator, or another `inputType`), from the validators the editor writes (`setTextFormat`). `answerChanges` reports "its date format changed from … to …" / "its answer format changed from … to …"; `SameWording` requires the same format. A list of dates' column kind now includes its date format. Text and long-answer (comment) are the same answer. A number's range and decimal places stay unlocked (limits, not the kind of answer; decisions.md). Tests: `TestWalkFormats`, `TestLockFreezesFormats`, `TestSameWordingFormats`, and the integration case "a published date's format and a text question's input format are frozen".
2. **A change inside a repeating group is reported once:** a dynamic panel's columns are compared only for a question added or removed; its questions are checked as questions of their own, so rewording one's text is allowed, as for any question. Test: `TestLockRepeatingGroupOnce`.
3. **The Publish window drops Go's refusal once it has re-checked:** a fresh `keyProblems` (compared as text) clears `refused`, so an empty re-check no longer leaves Publish blocked.
4. **`Duplicates` skips elements without a name.** Test: `TestDuplicatesSkipUnnamed`.
5. **`decisions.md`:** the two superseded S23 bullets are marked superseded (struck through, not deleted); the verification 2 answer is added. `saas-requirements.md` §6 point 4 says the whole answer is frozen.
- Also: a comment in `doc.ts`'s `newElement` points at Go's `placeholders`, which copy its default titles.

**E2E on the local data:** J3 and `test_procedures` expected "Demo pre-op HQ v1"; they now accept any published version of the seeded Demo pre-op HQ (`^Demo pre-op HQ v\d+$`), since a tester may publish a new one (R-05). **The local `sj` database was reset** (dropped, migrated, seeded); HJE Full HQ, which exists only locally, was exported first and put back as the same draft (same id, the same four Question Sets, content checked identical). A full `pg_dump` from before the reset is in the builder's scratchpad (`sj-before-s23-reset.dump`), temporary.

**Results after these fixes:** `make lint` passed; `make test` (Go 20 packages, Vitest 204 web + 120 clinical, CDK 124) passed; `make test-integration` 23 packages ok; `make e2e` **on the reset `sj` itself** (my API on :8093 and Vite on :3103 against it): **58 passed, 5 skipped**. On a scratch database where the Demo pre-op HQ was then published as **v2** (Create new version → Publish unchanged: 204, so R-05 holds on a fresh seed), J3 and `test_procedures` passed (5 tests). `sj` is left migrated and seeded, with HJE Full HQ restored, the Demo pre-op HQ at v1, and the e2e run's "E2E …" questionnaires (as every e2e run leaves).

## Test report 5 (22 Oct): all passed
23 of 23 (T-01 to T-18, R-01 to R-05). Its "found while exploring" items: the lock also refused "Allow several dates", a "Don't know" option and a question added inside a repeating group (as designed); the raw phone pattern in the message and the editor's country-specific Format options are in pending.md (Rahul to judge; the latter predates S23). Rahul's commit dc9bed8 (no incumbent name anywhere, `internal/repocheck`) landed under this work: `go test -count=1 ./internal/repocheck` passes and no S23 file names it; `make lint`, `make test` (Go 20 packages, Vitest 204 + 120, CDK 124) and `make test-integration` (23 packages) pass on the current tree. **S23 is tested**; next is `/verify-session S23`.

## Suggested commits (final)
1. `S23: question keys — internal/keys (walk, duplicates, whole-answer lock incl. date and text formats, wording matches), publish refuses key problems (422), GET …/question-keys (matches and problems), Always ask fresh, the editor's pre-fill label (PRE-05)`, with its tests: Go unit (`internal/keys`, `cmd/seed/keys_test.go`) and integration (`question_keys_integration_test.go`, the adjusted `trusts_integration_test.go`), Vitest (`keys.test.ts`, `fields.test.ts`, `publish.test.ts`), the e2e journey `test_question_keys.py` with its `e2e/README.md` row, and the J3 / `test_procedures` tolerance of a later Demo HQ version.
2. `docs: S23 verified — build notes, test plan and reports, answers for S23–S26, verification, decisions, pending, plan and session status`, including the conductor's recording of the S23–S26 answers and the obsolete note on `S02-workflow-route-tests/make-test-hq.js`.
