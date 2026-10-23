# S24 build notes: the question library and "Same as…"

Builder, automatic run (`/run-sessions S23 S26`), 22 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md); all five as recommended. From S23 they build on: pre-fill matches **by key, then by exact wording**, so "Same as…" is mainly for questions worded differently that mean the same; a published keyed question's **whole answer is frozen**, compared strictly with every published question holding that key in the trust, so "Same as…" copies the key **and the options** exactly. The brief's first item (S23 verification 3) was done first, test-first.

## What was built

### 1. A medication question's type is part of its frozen answer (first, test-first)
- `keys.formatOf` reads `medicationType`: "prescribed medication" (also when missing, the editor's default), "non-prescribed medication", "recreational drugs". So `answerChanges` refuses a change ("its medication type changed from prescribed medication to recreational drugs"), `SameWording` needs the same type, and `keys.Compatible` compares it.
- Tests: `TestMedicationTypeIsPartOfTheAnswer` (keys), the integration case "a published medication block's type is frozen" in `TestPublishKeyChecks`. Reproduced first: the unit test failed on all four types before the change.

### 2. Go: `internal/keys` (unit-tested)
- **`Compatible(mine, theirs)`** (answers 4 and 5, Go only): the same kind of answer. Single with single (option buttons, dropdown, Yes/No buttons), multiple with multiple: compatible whatever their options, since adopting copies them. Any other kind must already have the same unit, date or text format, medication type, scale, rows and labels, since nothing else is copied and S23's lock would refuse a difference. Never repeating lists ("insert them from the library instead"), signatures, uploads, the BMI block, or anything without an answer. Each refusal is a sentence for the author, e.g. "Its answer is single choice; this question's is text.", "Its unit is “kg”; this question's is “lb”."
- **`Candidates(mine, here, sources, search)`**: the keys "Same as…" lists. One per key, from the trust's published questionnaires and the library; only a patient's top-level answers (as pre-fill uses them); never the question's own key; matched by text (normalised) or key; adoptable ones first, then by text; at most 50. Each has `compatible`, `reason`, `inThisHQ` ("This HQ already has a question with this ID, in “About you”."), the sources' names, the kind and option texts to show, and the `element` JSON to copy from (the library's if it has it). A key the trust published in answers that disagree can't be adopted.
- **`Element(raw)`** walks one element on its own; `Question.Raw` keeps each element's own JSON from the walk (no extra cost: the bytes were already read).

### 3. Go: the library
- **Migration 00024 `library_questions`**: id, `org_id` (NULL = the platform's), key, title (for search), element (jsonb; its `name` must be the key), tags, created_by, created_at. A key is unique per owner (two partial unique indexes); a trust's item can't reuse a platform key (checked in Go: the platform's are only added by the seed).
- **Routes** (`library.go`), whole-trust super clinicians and admins only (answers 2), not audited (no patient data):
  - `GET /api/o/{oid}/library-questions?q=`: the platform's and the trust's, by title; `q` matches the title or a tag in any case (not a SQL pattern).
  - `POST /api/o/{oid}/library-questions` `{element, tags}`: one question (not a group, statement, calculation, profile, signature, upload or BMI block), saved with its key and options; its conditions (`visibleIf`, `enableIf`, `requiredIf`) and `linkedFrom` are dropped, since they name its own questionnaire. Refused (409) if the library has the key, or if the question isn't as the trust published it (S23's lock, under the same advisory lock as publishing).
  - `DELETE …/{id}`: the trust's own only; the platform's → 403; questionnaires keep their copies.
- **"Same as…"**: `POST /api/o/{oid}/h/{hid}/questionnaires/{qid}/question-keys/search` `{question, q}` → `{candidates}`. Only those who may change the questionnaire.
- **Seed**: `cmd/seed/library.json` is the starter library (answer 1), loaded by `seedLibrary` with the codes (every deployment, not just the demo data): added or updated by key, any other platform item removed. The **Demo pre-op HQ** (`demo-hq.json`) is rebuilt from library questions: smoker, alcohol units, height, weight (Always ask fresh) in "About you"; diabetes (the trust's own), snoring, pregnant (Always ask fresh), sickness after an anaesthetic, the two anaesthetic-problem groups, allergies and regular medicines in "Your health"; the clinician set unchanged.

### 4. TypeScript (`packages/clinical/src/library.ts`, for Rahul to review)
- `insertFromLibrary(doc, target, element)`: inserts a copy that keeps every name and option value; only its disclosures get new IDs (as a copied question's do).
- `takenNames(sets, element)`: the element's names (a group's questions included) already used in any Question Set of the HQ; the editor refuses an insert or an adoption with one.
- `adoptKey(doc, name, from, linkedFrom)`: the question takes `from`'s name and, if it has options, `from`'s options (values and wording, translations, its Other/None options) without their disclosures; it keeps its own text and everything else; conditions and skip rules in the Question Set that tested it are renamed; `linkedFrom: {questionnaire, title}` records where from.
- `linkedFrom` is a hidden SurveyJS property (`questionTypes.ts`) and in `fields.ts`'s `KEEP` list, like `alwaysFresh`.
- Tests: `library.test.ts`, `fields.test.ts` (linkedFrom survives a type change and a SurveyJS save).

### 5. The editor (`apps/web/src/features/editor`)
- `library.tsx` (new): **LibraryWindow** ("Insert from the question library": search by text or tag, Insert, Remove for the trust's own with a confirm; "Preopsis" or "This trust" badge, tags, kind, ID), **SaveToLibraryWindow** (tags, then "Saved."), **SameAsWindow** (candidates with kind, options, "In …", **Use this** only on compatible ones; the others greyed with Go's reason).
- `AddContentBar`: **From library** after the Add content buttons, for whole-trust super clinicians and admins on a draft; not inside a repeating group; disabled on a BMI page.
- `SettingsPanel`: **Same as…** (a patient's answer pre-fill can copy, not a repeating list, draft only) and **Save to library** (any answer-holding question, published ones too, for whole-trust super clinicians and admins), and "Linked to “…” in …" when a key was adopted.
- `QuestionnaireEditor`: the three windows; Insert and Use this are refused in the window when the key is already in the HQ (checked across every Question Set); after Use this the question stays selected under its new ID.

### 6. Docs
- PNL-06 in `requirements.md` (library inserts keep their IDs); `saas-requirements.md` §6 points 2–4 (built in S24; the medication type is frozen); the e2e README's new row.

## Decided by the builder (automatic mode)
- **"yes/no + details" is a group** (`anaesthetic_problems_group`: the Yes/No `anaesthetic_problems` and a details box `anaesthetic_problems_details` shown on Yes; the same for `family_…`). A library item is one element, so the group is the item and both questions keep their keys. The editor already builds this pattern; no new SurveyJS feature. Groups can't be *saved* to the library from the editor (saved groups are PNL-06's later part).
- **The library's starter wording** is in `library.json`; option values are readable (`never`, `ex`, `current`, `yes`, `no`, `not_sure`); allergies and regular medicines start empty ("Leave this empty if you have none").
- **Routes:** "Same as…" is under the questionnaire (`…/h/{hid}/questionnaires/{qid}/question-keys/search`, the brief said `/api/o/{oid}/question-keys/search`), as S23's GET was: it reuses the questionnaire's scoping and knows which HQ to check "already in this HQ" against.
- **Who may list the library:** whole-trust super clinicians and admins only (as for saving and removing). A hospital-only super clinician (Sue) can't change the trust's questionnaires, so has no use for it.
- **Compatible, other than choices, needs the same unit, format, medication type, scale, rows and labels** (answers 4 said "a date with a date"): adopting copies only the key for those, so a different date format would be refused at Publish anyway.
- ~~**Adopting keeps the question's `yesNo` display and its own disclosures are dropped with its old options;** … the editor's logic-problem check shows it.~~ *Wrong, corrected after verification 1 (below): no check flagged it.*
- **Save to library refuses a question that differs from the trust's published one with that key** (409), so a library copy can always be published.
- **The Same as… list** shows greyed candidates (repeating lists included) rather than hiding them, so the author sees why.
- **The local `sj` database was reset** (dropped, migrated, seeded) so it has the new Demo pre-op HQ; HJE Full HQ was exported first and put back (same ids, content checked identical by md5). A full dump from before is in the builder's scratchpad (`sj-before-s24-reset.dump`), temporary.

## Not done / follow-ups (in pending.md)
- **QA's Neon database keeps its old Demo pre-op HQ:** the seed never changes a published demo HQ. S25's demo on QA needs the new one: reset QA's demo HQ, or publish a new version from the library by hand.

## Tests
- Go unit: `internal/keys` (`TestMedicationTypeIsPartOfTheAnswer`, `TestCompatible`, `TestCandidates`), `cmd/seed/keys_test.go` (`TestStarterLibrary`: 11 items, unique keys, weight and pregnant Always ask fresh, the Demo HQ shares ≥8 keys with the library and asks them exactly as it does).
- Integration: `library_integration_test.go` (`TestListLibrary`, `TestSaveToLibrary`, `TestRemoveFromLibrary`, `TestSameAsSearch`, each with happy path, no sign-in, wrong role, another trust, invalid input), the medication case in `TestPublishKeyChecks`, `cmd/seed` `TestSeedTwice` (the platform library back as seeded).
- Vitest: `library.test.ts`, `fields.test.ts`.
- E2E: `journeys/test_question_library.py`.

## Results
- `make lint`: passed.
- `make test`: Go (all packages ok), Vitest 204 (web) + 128 (clinical), CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` on my own instance (database `sj_s24`, API :8094, Vite :3104, `E2E_BASE_URL=http://localhost:3104`): **59 passed, 5 skipped** (the QA-only Cognito journeys).

## Suggested commits
1. `S24: a medication question's type is part of its frozen answer (keys.formatOf), with its tests` (S23 verification 3).
2. `S24: the question library (platform starter set + the trust's own), Insert from library keeping the key, Save to library, "Same as…" with keys.Compatible and keys.Candidates; the Demo pre-op HQ from library questions; PNL-06 changed` with its tests (Go unit and integration, Vitest, `test_question_library.py`) and the session docs.

## Test report 1 (22 Oct): 19 of 19 passed; two "found while exploring" items fixed
- **Found 1 (fixed, in scope): Save to library was offered on, and saved, a question inside a repeating group.** On its own, a row question's key would mean a different answer (a single answer, not a list's rows), which pre-fill must never mix. Now:
  - the editor offers **Save to library** on the repeating group, not on its questions (`SettingsPanel`, `keyPlace(...).inList`);
  - Go refuses (409) a question whose key is a row question of a repeating list the trust has published ("…is a question of a repeating list in “X”, so it can't go in the library on its own. Save the repeating group instead."), and any key already used *inside* a library item (a group's or a list's question), not only an item's own key.
  - Tests: the integration case "a question of a repeating list can't go in on its own" in `TestSaveToLibrary` (failed first with 201); `test_question_library.py` checks the button is on the group and not on `allergies_what`.
- **Found 2 (fixed, cosmetic):** the library and Same as… windows kept the last search when reopened. Each opening now starts with an empty search (and Same as… asks again). The e2e journey checks both.
- **Found 3** (only one refusal reason shown) and **found 4** (the Demo HQ's pregnancy question has no condition on sex) are recorded in pending.md; no change.

**Results after the fixes:** `make lint` passed; `make test` (Go all ok, Vitest 204 + 128, CDK 124) passed; `make test-integration` 23 packages ok; `make e2e` on my own instance (`sj_s24`, :8094/:3104): **59 passed, 5 skipped**.

## Test report 2 (22 Oct): all passed (19 of 19)
Both report-1 fixes work. Its notes, none a defect in S24's scope:
- **Note 1:** T-05 step 9's expected message was the library-inside refusal; when the trust has published that repeating list, the (clearer) "is a question of a repeating list in …" refusal comes first. The test plan is corrected.
- **Notes 2, 4 and 5** (an empty library title is accepted; Same as… can replace a question's existing library or published key; the empty-group text in the read-only editor; HJE Full HQ's local description) are in pending.md for Rahul to judge. Replacing a key is an author's deliberate act in a draft, compatible by `keys.Compatible`, so nothing unsafe reaches pre-fill.

**S24 is tested**; next is `/verify-session S24`.

## Suggested commits (final)
1. `S24: a medication question's type is part of its frozen answer (keys.formatOf)`, with `TestMedicationTypeIsPartOfTheAnswer` and the `TestPublishKeyChecks` case.
2. `S24: the question library (platform starter set + the trust's own), Insert from library keeping the key, Save to library (never a repeating list's row question), "Same as…" with keys.Compatible and keys.Candidates; the Demo pre-op HQ from library questions; PNL-06 changed`, with its tests (Go unit and integration, `cmd/seed` tests, Vitest `library.test.ts`/`fields.test.ts`, e2e `test_question_library.py` and its README row).
3. `docs: S24 tested — build notes, test plan and reports, decisions, pending, plan and session status`.

## Verification 1 (23 Oct): fixes

All three findings and minor items 4–7 (7 in part), test-first:
1. **Conditions on the old options** (`packages/clinical`):
   - (a) `adoptKey` maps each old option value to the chosen question's option with the same text (`optionMapFor`, through `rewriteReferences`' option map), in conditions, option conditions, skip rules and the Question Set condition.
   - (b) **`logicProblems` reports a condition that compares a choice question with a value it doesn't have** ("it tests an answer that no longer exists (“Do you smoke?”: o_9)"), so the header's logic problems and Publish refuse it (`unknownAnswers` in `logic.ts`; Other and None count; grid rows and choices from another question aren't judged). It guards any edit, not only Same as…. Scanned the e2e data, the seed's Demo HQ, the integration fixtures and the local HJE Full HQ: none is flagged. One older unit test compared a Yes/No question with `'x'`; it now uses a real option.
   - (c) **The Same as… window says what would be lost before Use this** (`adoptionLosses`): "Its own disclosures on N options will be removed. M conditions test its old answers and will need changing (Publish is refused until then)." with **Use this anyway** / Cancel.
2. **Always ask fresh is copied** (Rahul, 23 Oct, `answers.md`): Go's `Candidate.alwaysFresh` says when any of the key's questions is fresh; the row shows "Always ask fresh, like the original"; `adoptKey` switches it on; the author may switch it off.
3. **The search sees every copy of a key** (`keys.Candidates`): a key is kept if its key or any copy's text matches, so a differently worded published copy can't be skipped (`TestCandidatesSearchSeesEveryCopy`).
4. **Other Question Sets' conditions are renamed too** (`renameReferences` applied by the editor to every other loaded set that changes).
5. **`adoptCandidate` waits for every Question Set to load**, as Insert does.
6. **A Yes / No question taking options that aren't two is no longer Yes / No** (`yesNo` cleared).
7. `libraryElement` uses `keys.OneOff`; the rest is in pending.md.

Tests: `logic.test.ts` (answers that don't exist), `library.test.ts` (mapping, fresh, Yes/No, losses, other sets), `doc.test.ts` (a real option value), `keys_test.go` `TestCandidatesSearchSeesEveryCopy`; the e2e journey now checks the fresh badge on the candidate and the card, switches it off, then sees the pre-fill label.

**Results after verification 1's fixes:** `make lint` passed; `make test` (Go all ok, Vitest 204 web + 135 clinical, CDK 124) passed; `make test-integration` 23 packages ok; `make e2e` on my own instance (`sj_s24`, :8094/:3104): 58 passed, 1 failed, 5 skipped. The failure was `test_statuses.py::test_a_stale_tab_is_told_and_keeps_its_input` (S20's, nothing S24 touches). It passed twice on its own (4 of 4 each time), so it's a timing flake under the full run's load. The library journey passed.

## Suggested commits (after verification 1)
1. `S24: a medication question's type is part of its frozen answer (keys.formatOf)`, with its tests.
2. `S24: the question library and "Same as…"` (as before, now with: options mapped by text, Always ask fresh copied, the losses warning, the unknown-answer logic problem, the search over every copy, other sets renamed), with its tests.
3. `docs: S24 verified — …` once verified.

## Test report 3 (23 Oct): all passed (21 of 21)
Verification 1's fixes work in the browser (T-15, T-16 and the retested T-07 to T-10). Its exploring items are cosmetic or minor (the squeezed **Use this** button on rows with a long "In …" list; Save to library reopening on "Saved."; a removed answer's raw value on a card); with one test round left, they're recorded in pending.md, not changed. **S24 is tested**; next is `/verify-session S24` (verification 2).

## Verification 2 (23 Oct): fix
**Finding 1: a later Question Set's condition renamed to an answer the new question lacks hid the set, unflagged** (a regression from verification 1's fix 4). Test-first:
- **`chapterConditionProblems` reports an answer that doesn't exist** (`unknownAnswers` on the earlier sets), so the editor's logic problems and Publish refuse it, after any edit, not only Same as… (`questionSets.test.ts`, new).
- **`adoptionLosses` counts every other Question Set's conditions too** (the editor passes them), so the window warns before Use this (`library.test.ts`).
- **Similar cross-set gaps checked:** element conditions, calculations and skip rules can only test their own set (`unknownReferences` flags any other), and disclosures hold no conditions, so the Question Set condition was the only one. **Test cases' answers** are now renamed with the key (values mapped), so a set's saved test cases keep testing the adopted question (`library.test.ts`).
- E2E: `test_same_as_and_a_later_question_sets_condition` (the warning, then Publish refuses the set's condition). Test plan: T-16 step 8.
- The cosmetic items from test report 3 stay in pending.md.

**Results:** `make lint` passed; `make test` (Go all ok, Vitest 204 web + 140 clinical, CDK 124) passed; `make test-integration` 23 packages ok; `make e2e` on my own instance (`sj_s24`, :8094/:3104): **60 passed, 5 skipped**.

## Test report 4 (23 Oct): all passed (21 of 21), the last round
Verification 2's fix works in the browser (T-16 step 8), and the new check also guards plain edits (deleting an option a later set's condition tests). Its exploring items are minor and recorded in pending.md, with no code change (no rounds left): ⌘Z after Same as… restores only the current Question Set (Publish still refuses the other set's renamed condition); Save to library reopening on "Saved."; the squeezed Use this button. **S24 is tested**; next is `/verify-session S24` (verification 3).
