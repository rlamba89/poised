# S24 verification

## Verification 3 (23 Oct 2026, automatic run; the last)

**Verdict: Verified.** Verification 2's finding is fixed and works: a later Question Set's condition left testing an answer the question no longer has is flagged in the set's header and refused at Publish, and the Same as… window counts it before **Use this**. Every automated test passes, and a scan of every questionnaire version in the local database found no false positives from the new checks. What's left (from `/code-review high` and test report 4) is minor, and none of it is silent: Publish refuses each case that could reach a patient, or it's cosmetic. It's recorded in `pending.md`, as S23's verification 3 did with its leftover.

### Verification 2's finding, checked

- **1 (a later Question Set hidden by "Same as…"): fixed.**
  - `chapterConditionProblems` (`packages/clinical/src/questionSets.ts:39-54`) now runs `unknownAnswers` on the earlier sets, so the header's logic problems and `publishProblems` (`publish.ts:17`) both refuse it. It guards any edit (test report 4 deleted an option a later set tested: refused).
  - `adoptionLosses` (`library.ts:91-113`) counts every other set's conditions, and the editor passes the other loaded sets (`QuestionnaireEditor.tsx:566-569`). Test cases' answers are renamed with the key and their values mapped (`library.ts:136-147`).
  - Unit tests: `questionSets.test.ts` (new) and `library.test.ts`. E2E: `test_same_as_and_a_later_question_sets_condition`.
  - **Checked in Chrome as Alex** with a variant the tests don't cover, in a throwaway draft `S24 Verify3 5773` (deleted afterwards, 204):
    - Set "Snoring" had "Do you snore at night?" (Yes `o_a` / Sometimes `o_b` / No `o_c`). Set "Regular snorers" was shown when `{q_snz} anyof ['o_a', 'o_b']`, and set "Non snorers" when `{q_snz} <> 'o_c'`.
    - Same as… → `snoring` → **Use this**: the window warned "1 condition tests its old answers…". It counted only the `anyof` one, because "No" maps to `no`.
    - After **Use this anyway**, the stored conditions were `{snoring} anyof ['yes', 'o_b']` and `{snoring} <> 'no'`.
    - **Publish** showed "1 problem to fix first: Regular snorers · Question Set condition: it tests an answer that no longer exists (“Do you snore at night?”: o_b)", with Publish disabled. I cancelled.
- **No false positives:** I ran `logicProblems` and `chapterConditionProblems` (with each set's earlier sets, as `publishProblems` does) over all 122 versions (220 patient Question Sets) in the local database. That covers the Demo pre-op HQ v1–v5, the HJE Full HQ draft, the e2e and tester data. The only hits were the two problems test report 4 left on purpose in `S24 Lib 1050`. The scan was a throwaway test file, deleted.

### Minor findings (in `pending.md`, not blocking)

All are from `/code-review high`, weighed here. None reaches a patient silently.

1. **`unknownAnswers` compares values case-sensitively and only reads `{q} op 'v'`** (`logic.ts:440-460`); `adoptionLosses` and `rewriteReferences` read with their own regexes. So:
   - **Too strict:** a hand-written `{q} = 'Yes'` against the value `yes` works in SurveyJS (it compares case-insensitively) but is flagged. Publish refuses it, so it's not silent.
   - **Not caught:** a reversed `'o_2' = {q}` or a double-quoted value isn't judged. The condition builder never writes these; they'd only come from the Code tab. None of the 79 stored expressions locally has either.
   - Moving all three onto SurveyJS's `ConditionsParser` (as `logic.ts` already does elsewhere) would make them one.
2. **Saving a repeating group to the library checks only the group's own key** (`apps/api/internal/httpapi/library.go:96-139`). Its row questions' keys aren't checked against the library, or against the trust's published questions (`namesOf(mine)` is the top-level key only). So the library can hold a row key twice. An HQ using it is still refused at Publish by S23's lock.
3. **A saved question's option conditions and a repeating group's row conditions are kept** (`library.go:183`, only the top-level conditions are dropped). This is verification 1's item 7. A condition naming the source HQ's questions is refused at Publish ("tests something that no longer exists").
4. **`keys.Compatible` passes any choice question, but `adoptKey` copies options only when the chosen question has a `choices` array** (`keys.go` `Compatible`, `library.ts:64`). A published question with `choicesFromQuestion` would give its key without its options. Publish's lock refuses it. Rare: no starter or Demo question does this.
5. **Small ones:**
   - A Yes/No question adopting two options that aren't Yes/No stays "Yes / No" (`library.ts:73`). The label is wrong; the answers are right.
   - The seed adds a platform key without checking that no trust's library already has it (`cmd/seed/main.go` `seedLibrary`). This only matters if `library.json` gains a key later.
   - A search of only punctuation lists every candidate: "?" returned 42.
   - `renameIn` repeats `doc.ts`'s rewrite loop.
6. **From test report 4 (already in `pending.md`):**
   - ⌘Z after Same as… restores only the current set. Publish refuses the other set's renamed condition.
   - Save to library reopens on "Saved.".
   - The squeezed **Use this** button.

### Checks

- **Scope:** everything in the brief and the answers is built. Since verification 2, only the fix above changed: `questionSets.ts`, `library.ts`, `QuestionnaireEditor.tsx`, the tests, the e2e journey and the docs. No Go code changed.
- **Tests first:** `TestListLibrary`, `TestSaveToLibrary`, `TestRemoveFromLibrary` and `TestSameAsSearch` each have the five cases. The logic has unit tests (`keys_test.go`, `logic.test.ts`, `library.test.ts`, `questionSets.test.ts`, `fields.test.ts`, `cmd/seed`). Journeys are in `e2e/journeys/test_question_library.py`.
- **Tenant isolation, rechecked in Chrome through the API:**
  - Sue and Cara got 403 listing and saving to Trust A's library.
  - Bea got 404 for both, and her Trust B library lists only the 11 platform items.
  - The library queries are scoped by `org_id`; Same as…'s sources by `trustVersions`.
- **A-20 grep:** nothing new (only the earlier comments in `format.ts` and `patient.go`).
- **Security:**
  - no secrets;
  - the library holds no patient data, so it's correctly not audited;
  - the patient router and `chapter.ForPatient` are unchanged;
  - `repocheck` passes.
- **Docs:** `decisions.md`, `pending.md` (with the items above), `requirements.md` PNL-06, `saas-requirements.md` §6, the plan's status, `docs/plans/README.md`, the sessions table (now Verified) and the e2e README.

### Test results

- `make lint`: passed.
- `make test`: Go all ok (including `keys`, `cmd/seed`, `repocheck`). Vitest: web 204 passed (20 files), clinical 140 passed (18 files). CDK: 124 passed.
- `make test-integration`: every package ok, exit 0.
- `make e2e` (against the running `make dev`): **60 passed, 5 skipped** (the QA-only Cognito journeys), in 3 min 28 s.
- **Browser (Chrome, "rahul chrome"):**
  - the variant above as Alex;
  - the permission and other-trust checks;
  - the punctuation search.

### Test data left by the verifier

None. The draft `S24 Verify3 5773` was created through the API and deleted (204), and nothing was published. Nothing was saved to the library (every attempted save was refused). The browser was signed back in as Alex.

### Hand checks for Rahul

- None.

### Commits

1. `S24: a medication question's type is part of its frozen answer (keys.formatOf)`, with `TestMedicationTypeIsPartOfTheAnswer` and the `TestPublishKeyChecks` case.
2. `S24: the question library and "Same as…"`, with its tests (Go unit and integration, `cmd/seed`, Vitest, `test_question_library.py` and its README row).
3. `docs: S24 verified`: the build notes, test plan, reports, verification, decisions, pending, and the plan and session status.

## Verification 2 (23 Oct 2026, automatic run)

**Verdict: Needs fixes.** Verification 1's findings 1–3 and minor items 4–6 are fixed and work (the option mapping, the unknown-answer logic problem, Always ask fresh copied, the search over every copy, the loading guard, Yes / No cleared), and every automated test passes. But the fix for minor item 4 (renaming other Question Sets' conditions) opened a silent hole of the same kind as finding 1, one level up: **a whole Question Set can be hidden from patients with nothing flagging it.** Fix it test-first (small), then retest T-16 and the new case below.

### Findings (worst first)

1. **"Same as…" can silently hide a later Question Set: an unmapped option in another set's Question Set condition is renamed to the new key, and no check flags it.**
   - **Where:**
     - `apps/web/src/features/editor/QuestionnaireEditor.tsx:247-253` (`adoptCandidate`) now renames every other set's conditions with the option map, so `{q_old} = 'o_2'` becomes `{snoring} = 'o_2'` when no new option has o_2's text;
     - `packages/clinical/src/questionSets.ts:35-47` (`chapterConditionProblems`) checks only the *names* a Question Set condition tests, never their values;
     - `packages/clinical/src/logic.ts:451` (`unknownAnswers`) judges only questions in the same Question Set (`findElement(doc, …)`), so it can't see a question in an earlier set;
     - `packages/clinical/src/library.ts:95-106` (`adoptionLosses`) counts conditions in the current set only, so the window says nothing.
   - **Reproduced** with a throwaway Vitest case (deleted) and **in Chrome as Alex** in a throwaway draft `S24 Verify2 cross-set` (deleted afterwards):
     - Set "Smoking": "Do you smoke at all?" (Yes `o_1` / Sometimes `o_2`). Set "Occasional smokers": displayed when that question is Sometimes (`{q_vxa} = 'o_2'`).
     - Same as… → `snoring` → **Use this**: no warning (the window counted 0 conditions). The question became `snoring` (Yes / No).
     - The second set's condition was stored as `{snoring} = 'o_2'`. The header showed no logic problem, and **Publish offered "Publish version 1?" with no problems** (I cancelled).
   - **A regression from verification 1's fix 4.** Before it, the other set's condition kept naming `q_vxa`, and Publish refused it ("it tests something that isn't in an earlier Question Set"). Now the rename makes it look valid.
   - **Why it matters:** a Question Set that's never shown is worse than finding 1's single follow-up question. Example: "Women's health" shown when "Sex" is Female; the author links "Sex" to a library key whose options are worded "Female" → "Woman". The HQ publishes cleanly and no patient ever gets that set.
   - **Fix, test-first:**
     - (a) **`chapterConditionProblems` reports an answer that doesn't exist**, using `unknownAnswers(earlier, expr)` ("it tests an answer that no longer exists (…)"). The editor's logic problems and Publish (`publish.ts:18`) both use it, so both then refuse it. Like finding 1(b), this guards any edit, not only Same as…. Unit case in `questionSets.test.ts`; a case in `library.test.ts` for the adoption.
     - (b) **The Same as… warning counts other sets' conditions too:** `adoptionLosses` takes every loaded set (or the editor adds up the other sets'), so the window says "1 condition tests its old answers…" before **Use this**.
     - Retest: T-16 step 7 with a Question Set condition on an option that has **no** same-text match (e.g. Sometimes → `snoring`).

**Minor (record in `pending.md` if not fixed):**

2. **The cosmetic items from test report 3 stand** (the squeezed **Use this**, Save to library reopening on "Saved.", a removed answer's raw value on a card). Already in `pending.md`; I saw the squeezed button too (`q_diabetes`, `q_smokes`). Not blocking.

### Verification 1's findings, checked

- **1 (conditions on old options):** fixed within the set. `adoptKey` maps by text (`optionMapFor`); `unknownAnswers` flags the rest in the header and at Publish; the window warns before **Use this anyway**. Read the code and the tests (`logic.test.ts`, `library.test.ts`); test report 3's T-16 shows it in the browser. (Across sets: finding 1 above.)
- **2 (Always ask fresh):** fixed as Rahul answered. `Candidate.alwaysFresh` is true if any copy is fresh, and `adoptCandidate` passes it to `adoptKey` (`QuestionnaireEditor.tsx:244`). T-15 passed.
- **3 (the search hides a disagreeing copy):** fixed. Every copy's text counts (`keys.go` `Candidates`, `f.matches`), with `TestCandidatesSearchSeesEveryCopy`.
- **4–6:** done (other sets renamed, the loading guard, `yesNo` cleared). **7:** `libraryElement` uses `keys.OneOff`; the rest is in `pending.md`.
- `pending.md` and `build-notes.md` are corrected where finding 1 said they were wrong.

### Checks

- **Scope:** unchanged from verification 1. Everything in the brief and the answers is built, including "After verification 1" (Always ask fresh is copied).
- **Tests first:** `TestListLibrary`, `TestSaveToLibrary`, `TestRemoveFromLibrary` and `TestSameAsSearch` each have the five cases. The new logic has unit tests (`logic.test.ts`, `library.test.ts`, `keys_test.go`).
- **Tenant isolation:** the library queries and Same as…'s sources are still scoped by `org_id` / `trustVersions`. Since verification 1 (which checked Bea, Max, Cara, Sue and Ada in Chrome), `library.go` changed only to use `keys.OneOff`.
- **A-20 grep:** nothing new (only the earlier comments in `format.ts` and `patient.go`, and the infra tests' UK lists).
- **Security:** no secrets; no patient data in the library (correctly not audited); the patient router and `chapter.ForPatient` are unchanged; `repocheck` passes.
- **Docs:** `decisions.md`, `pending.md`, `requirements.md` PNL-06, `saas-requirements.md` §6, the plan's status, `docs/plans/README.md`, the sessions table (Tested) and the e2e README are updated.

### Test results

- `make lint`: passed.
- `make test`: Go all ok (including `keys`, `cmd/seed`, `repocheck`). Vitest: web 204 passed (20 files), clinical 135 passed (17 files). CDK: 124 passed.
- `make test-integration`: every package ok, exit 0.
- `make e2e` (against the running `make dev`): **59 passed, 5 skipped** (the QA-only Cognito journeys), in 3 min 30 s.
- **Browser (Chrome, "rahul chrome", as Alex):** finding 1 reproduced end to end (Same as…, the stored condition, the clean Publish window, cancelled).

### Test data left by the verifier

None. The draft `S24 Verify2 cross-set` was created through the API, used, and deleted (204). Nothing was published.

### Hand checks for Rahul

- None.

### For the builder

"Verification found problems: read `docs/sessions/S24-question-library/verify.md`, fix them test-first, then ask me for a retest."

### Suggested commits (once verified)

As in verification 1:
1. `S24: a medication question's type is part of its frozen answer (keys.formatOf)`, with `TestMedicationTypeIsPartOfTheAnswer` and the `TestPublishKeyChecks` case.
2. `S24: the question library and "Same as…"`, with its tests (Go unit and integration, `cmd/seed`, Vitest, `test_question_library.py` and its README row).
3. `docs: S24 verified`: the build notes, test plan, reports, verification, decisions, pending, and the plan and session status.

## Verification 1 (22 Oct 2026, automatic run)

**Verdict: Needs fixes.** The library, Insert from library, Save to library, the routes' permissions and tenant isolation, the medication-type lock and the seeded Demo HQ are sound, and every automated test passes. Two problems in **"Same as…"** reach patients (findings 1 and 2), and one in its search is a small bug (finding 3). Fix them test-first, then retest T-07 to T-10 and the new cases.

### Findings (worst first)

1. **"Same as…" silently breaks conditions that tested the question's old options. A follow-up question then never shows to patients, and nothing flags it.**
   - **Where:** `packages/clinical/src/library.ts:48-65` (`adoptKey`). It renames `{q_old}` to `{smoker}` but keeps the old option values, so `{q_old} = 'o_1'` becomes `{smoker} = 'o_1'`, and `o_1` no longer exists. `rewriteReferences` (`doc.ts:479`) already takes an option-value map, but `renameReferences` doesn't pass one.
   - **Reproduced** with a throwaway Vitest case (deleted afterwards). A radiogroup `q_a` (o_1 "Yes", o_2 "No") has a follow-up `visibleIf: "{q_a} = 'o_1'"`. After `adoptKey(…, smoker)` the follow-up's condition is `{smoker} = 'o_1'`, and `logicProblems` returns `[]`. So Publish isn't blocked either: `publish.ts:19` runs `logicProblems`, which checks only unknown *questions*, not unknown values.
   - **The build notes and docs say otherwise.** `build-notes.md:51` says "the editor's logic-problem check shows it", and `pending.md:66` says "the logic-problem button flags them". Neither is true.
   - **Why it matters:** an author who links "Do you smoke now?" (with "How many a day?" shown on Yes) gets an HQ that publishes cleanly but never asks "How many a day?". The same applies to skip rules and to option-level `visibleIf`. The question's own option disclosures (flags, notes, ASA) are also dropped with no warning (`library.ts:52-56`), so the clinician summary silently loses them.
   - **Fix, test-first:**
     - (a) In `adoptKey`, map each old option value to the new option with exactly the same text, where there is one, through `rewriteReferences`' option map.
     - (b) Make `logicProblems` report a condition that compares a choice question with a value it doesn't have ("it tests an answer that no longer exists"), so Publish refuses what (a) couldn't map. This is a general check that also guards other edits.
     - (c) In the Same as… window, before **Use this**, say what will be lost: "Its own disclosures on N options will be removed; M conditions test its old answers."
     - Correct `pending.md:66` and `build-notes.md:51`.

2. **"Same as…" doesn't carry "Always ask fresh". Linking to `pregnant` or `weight` makes a question that *will* pre-fill.**
   - **Reproduced in Chrome as Alex** in the draft `S24 Lib 2131`:
     - I added a Yes/No question and used **Same as…** on `pregnant` (the library's and Demo HQ's, both Always ask fresh).
     - The question took `pregnant` and its three options. **Always ask fresh stayed off**, and the card said **"Pre-fills from 1 other HQ"**.
     - I deleted the question afterwards.
   - **Where:** `library.ts:48-65` (only the name and options are copied). `Candidate` (`keys.go`) doesn't say whether the chosen question is Always ask fresh.
   - **Why it matters:** the starter library marks pregnancy and today's weight Always ask fresh so they're never pre-filled (S24 Q1, PRE-06). S25 pre-fills by key, and its answer Q3 skips only questions that are themselves `alwaysFresh`. A linked copy would put a months-old "not pregnant" or weight in front of the patient. Rahul's rule is "if in doubt, don't fill".
   - **Question for Rahul:** should adopting copy Always ask fresh?
     - **Recommended:** yes. When the chosen question (library or published) is Always ask fresh, the adopted question gets `alwaysFresh: true` too, shown in the window ("Always ask fresh, like the original"). The author can still switch it off deliberately.
     - **Alternative:** leave S24 as is, and have S25 treat a key as fresh if any published question or library item with it is fresh.
     - The builder should do the recommended option unless Rahul says otherwise.

3. **The Same as… search can hide a disagreeing published copy, so a candidate that will be refused at Publish shows as adoptable.**
   - **Where:** `apps/api/internal/keys/keys.go:859` (`Candidates`). The text filter runs only on a key's *first* occurrence. If the trust's published copy of a key has text that doesn't match the search, it's skipped; the library's copy then creates the candidate alone. That bypasses both the "published with different answers" check and the sources list.
   - **Example:** the trust published `smoker` as "Smoking status" (Yes/No), and the library has "Do you smoke?" (three options). Searching "do you smoke" offers `smoker` as compatible; with an empty search it's correctly refused.
   - **How likely:** today only if the platform's starter wording or options change after a trust published them. S23's lock still refuses at Publish, so nothing unsafe reaches patients, but the author gets a wrong "Use this".
   - **Fix:** collect every source's questions per key first, then keep a key if *any* of its texts or the key matches the search. Add a unit case beside `TestCandidates`.

**Minor (fix if cheap, or record in `pending.md`):**

4. **Renames reach only the current Question Set.** `library.ts:65` renames references only in the current set, so another set's Question Set condition (`chapterVisibleIf`) that tested the old key is left naming it. Publish refuses it (`chapterConditionProblems`), so it isn't silent, but the editor could apply the rename to the other sets as it does here.
5. **`adoptCandidate` lacks the loading guard.** `QuestionnaireEditor.tsx:237` doesn't have the "still loading" guard that `insertLibraryQuestion` (`:223`) has, so a key in a not-yet-loaded set can be missed. Publish's `Duplicates` check catches it.
6. **A Yes/No question can end up with three options.** A Yes/No-buttons question adopting a three-option key keeps `yesNo: true` with three options: its Settings panel says "Yes / No" over three options. (Seen in Chrome with `pregnant`; it renders, but the type label is wrong.) Clear `yesNo` when the adopted options aren't two.
7. **Library code repeats itself in places.**
   - `libraryElement` (`library.go:180`) hard-codes `signaturepad/file/bmi`, which is `keys.oneOff`; it also drops only top-level conditions, so a saved repeating group keeps its row questions' `visibleIf`.
   - `keySources` (`question_keys.go`) repeats `publishedQuestions`' newest-version dedupe.
   - Same as… re-walks every published version of the trust on each search keystroke. Fine at today's sizes; already noted for S23's route in `pending.md`.

### Reviewed and accepted (no action)

- **The seed never replaces a published demo HQ**, so QA keeps the old one. This is already in `pending.md`, and the local database was reset by the builder.
- **Inserting a library item whose key the trust published in a different shape** is refused at Publish by S23's lock, not flagged in the library window. This only happens if the platform's starter set changes after a trust published it. Worth a badge later.
- **"Same as…" can replace an existing key, and the library accepts an empty title:** both are in `pending.md` from test report 2, for Rahul to judge.

### Checks

- **Scope:** everything in the brief and the answers is built. That covers the medication type (first, test-first), `library_questions`, the three routes, Insert from library keeping the key, Save to library, Same as… with `keys.Compatible`/`Candidates` in Go only, the seeded starter library (11 items) and the Demo HQ from library keys, and PNL-06. Two route differences are reasoned in the build notes: Same as… sits under the questionnaire, and listing is limited to whole-trust super clinicians and admins.
- **Tests first:**
  - `TestListLibrary`, `TestSaveToLibrary`, `TestRemoveFromLibrary` and `TestSameAsSearch` each have the five cases.
  - `internal/keys` has `TestMedicationTypeIsPartOfTheAnswer`, `TestCompatible` and `TestCandidates`; `TestPublishKeyChecks` has the medication case.
  - Also: `cmd/seed` `TestStarterLibrary` and `TestSeedTwice`, Vitest `library.test.ts`, and the e2e journey `test_question_library.py`.
- **Tenant isolation:** the library queries are scoped by `org_id` (`org_id IS NULL OR org_id = $org`, and delete by `org_id`). The Same as… sources come from the trust's own versions (`trustVersions`). Checked in Chrome, all through the API from the signed-in page:
  - **Bea** got 404 for Trust A's library (GET), 404 for deleting Trust A's item (through either trust's URL), 404 for Trust A's Same as… search, and saw none of Trust A's items in Trust B (11 platform items only).
  - **Max** (clinician at Hospital A2, admin of Trust B) got 403 for Trust A's library and 200 for Trust B's.
  - **Cara and Sue** got 403 for GET; Cara also got 403 for delete, and Sue for the Same as… search.
  - **Ada** got 403 removing a platform item and 204 removing the trust's own.
  - **Saving** returned 201 and dropped the question's `visibleIf`.
- **A-20 grep:** nothing new (only earlier comments in `format.ts` and `patient.go`).
- **Security:**
  - no secrets;
  - the library has no patient data, so it's correctly not audited;
  - the patient router is unchanged;
  - `chapter.ForPatient` is unchanged;
  - `repocheck` passes.
- **Docs:** `decisions.md`, `pending.md` (but see finding 1), `requirements.md` PNL-06, `saas-requirements.md` §6, the plan's status, the sessions table (Tested) and the e2e README are all updated.

### Test results

- `make lint`: passed.
- `make test`: Go all ok (including `keys`, `cmd/seed`, `repocheck`). Vitest: web 204 passed (20 files), clinical 128 passed (17 files). CDK: 124 passed.
- `make test-integration`: every package ok, exit 0.
- `make e2e` (against the running `make dev`): **59 passed, 5 skipped** (the QA-only Cognito journeys), in 4 min 12 s.
- **Browser (Chrome, "rahul chrome"):** the permission and other-trust checks above, and finding 2 reproduced and then cleaned up.

### Test data left by the verifier

None: the library item `q_verify…` was saved and then removed, and the test Yes/No question in `S24 Lib 2131` was deleted.

### Hand checks for Rahul

- None.

### For the builder

"Verification found problems: read `docs/sessions/S24-question-library/verify.md`, fix them test-first, then ask me for a retest."

### Suggested commits (once verified)

1. `S24: a medication question's type is part of its frozen answer (keys.formatOf)`, with `TestMedicationTypeIsPartOfTheAnswer` and the `TestPublishKeyChecks` case.
2. `S24: the question library and "Same as…"`, covering the platform starter set and the trust's own, Insert from library keeping the key, Save to library, `keys.Compatible`/`Candidates`, the Demo pre-op HQ from library questions, and the PNL-06 change. It goes with its tests: Go unit and integration, `cmd/seed`, Vitest, and `test_question_library.py` with its README row.
3. `docs: S24 verified`: the build notes, test plan, reports, verification, decisions, pending, and the plan and session status.
