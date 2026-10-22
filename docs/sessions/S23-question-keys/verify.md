# S23 verification

## Verification 3 (22 Oct 2026, automatic run; the extra round Rahul allowed)

**Verdict: Verified**, with one gap carried into S24 as its first build item (finding 1).

Verification 2's five items are fixed, and I checked each in the code and, for 1 and 2, on the running app:
1. **Date and text formats are frozen and part of the wording match** (`keys.go:218` `formatOf`, compared in `answerChanges` `keys.go:487` and `SameWording` `keys.go:606`). It reads what the editor writes (`fields.ts` `dateField`, `setTextFormat`). Reproduced as Alex: "T23 V3 Date V3-38223" v1 (day, month and year) published; v2 as Year only → **422** "its date format changed from day, month and year to year only"; as Month and year → **422** likewise. The editor's Publish window showed the same problem with Publish disabled.
2. **A change inside a repeating group is reported once** (`keys.go:508`). Reproduced: "T23 V3 Group V3-38223" v2 rewording a child's option → **422 with exactly one problem**, and `question-keys` listed one problem.
3. **`refused` clears once the keys are re-fetched** (`PublishModal.tsx`): after a 422 it re-fetches; the fresh list (compared as text, null → list) replaces the refusal. Read in the code; not reproducible by hand (it needs the trust to change between two requests).
4. **`Duplicates` skips unnamed elements** (`keys.go:389`), with `TestDuplicatesSkipUnnamed`.
5. **`decisions.md`:** the superseded bullets are struck through and marked; `saas-requirements.md` §6 point 4 says the whole answer is frozen.

### Findings (worst first)

1. **A medication question's type isn't part of the frozen answer.** `internal/keys` never reads `medicationType` (`keys.go:186` `kindOf` gives kind `medication`; `formatOf` gives ""). The editor offers Prescribed / Non-prescribed / Recreational (`SettingsPanel.tsx:130`).
   - **Reproduced:** "T23 V3 Meds V3-38223" v1 (Prescribed) published; v2, the same key as **Recreational**, **published (204)** with no problems. Two medication blocks of different types with the same text would also match by wording.
   - **Why it matters:** Rahul's "freeze all kind of answers" covers it: S25 copies by key, and would put a patient's prescribed medicines into a recreational-drugs list. Found by `/code-review high`.
   - **Why it doesn't block S23:** nothing copies answers until S25, the local data is disposable, and the fix is a few lines in `formatOf` with tests. **Carried into S24's brief as its first build item** (test-first: `TestLockFreezesFormats`, `TestSameWordingFormats`, a case in `TestPublishKeyChecks`; `keys.Compatible` should compare it too), and into `pending.md`. Rahul: if you'd rather it were fixed in S23 itself, say so before S24 starts.

**Minor (in `pending.md`):**

2. **The badges can go stale until the next content save:** changing a Question Set's audience, or adding or deleting a set, doesn't re-fetch `question-keys` (`QuestionnaireEditor.tsx`, the `savedCount` effect), and the editor fetches it twice as it opens. Publish re-fetches as its window opens, so publishing is unaffected.
3. **If the re-fetch after a 422 fails,** the refusal is cleared and the window says ready; clicking Publish gives the same 422. Go still refuses.
4. **`keyPlace` walks the document for every card on every render** (`ElementCard.tsx` `KeyBadge`). Fine at today's sizes.

### Reviewed and accepted (no action)

- `/code-review high` also raised, each already recorded: the save/publish race (`pending.md`, verification 1 item 8); trust-wide questionnaire names to a hospital-scoped super clinician (`decisions.md`); a key published in disagreeing shapes is blocked for good (Rahul's "keep the strict rule", after test report 3); the route's cost on every save (`pending.md`; add a key table if slow); `placeholders` is a hand copy of `newElement`'s titles (now with a comment in `doc.ts`). The medication and admissions default titles aren't placeholders, deliberately: they're real wording, and with finding 1 fixed a medication match also needs the same type.
- No other editor setting I found changes what an answer means: a number's range and decimals, text length, date range, rows allowed and selection limits are limits (recorded in `decisions.md`); display settings (columns, rating style) don't change the answer.

### Checks

- **Scope:** the brief, following the answers where they differ (Q2, Q5, after verification 1, after test report 3, after verification 2). Nothing big outside it; the e2e tolerance of a later Demo HQ version and the obsolete note on `S02-…/make-test-hq.js` are small and reasoned.
- **Tests first:** `internal/keys` unit tests including `TestWalkFormats`, `TestLockFreezesFormats`, `TestSameWordingFormats`, `TestLockRepeatingGroupOnce`, `TestDuplicatesSkipUnnamed`; `TestQuestionKeys` has the five cases (happy, no sign-in, wrong role, another trust, invalid input); `TestPublishKeyChecks` has the format case; Vitest; e2e `test_question_keys.py`.
- **Tenant isolation:** `ListTrustPublishedContents` is scoped by `q.org_id`; another trust's HQs never count (integration tests; test report 5 T-06, T-12).
- **A-20 grep:** nothing in the new or changed code.
- **Security:** no secrets; no patient data in the route (correctly not audited); patient router unchanged; `chapter.ForPatient` unchanged; `repocheck` passes.
- **Docs:** session row **Verified**, plan C4 "In progress: S23 verified", `decisions.md` (verification 3 line), `pending.md`, S24 brief.

### Test results

- `make lint`: passed.
- `make test`: Go all ok (incl. `repocheck`, `keys`); Vitest web 204 passed (20 files), clinical 120 passed (16 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 12.6 s, keys 1.7 s), exit 0.
- `make e2e` (against the running `make dev`): **58 passed, 5 skipped** (the QA-only Cognito journeys), in 3 min 14 s.
- **Browser (Chrome, as Alex):** the date-format, medication and repeating-group reproductions above (via the API from the signed-in page); the Publish window for the date HQ's v2 showed Go's one problem with Publish disabled. No console errors.

### Test data left by the verifier

Trust A, made up: "T23 V3 Date V3-38223" (v1 published, v2 draft), "T23 V3 Meds V3-38223" (v1 Prescribed and v2 Recreational, both published under one key; only the newest version counts, so it blocks nothing), "T23 V3 Group V3-38223" (v1 published, v2 draft).

### Hand checks for Rahul

- None new.

### Commits

1. `cfdfadf` S23: question keys — internal/keys, publish refuses key problems (422), GET …/question-keys, Always ask fresh, the editor's pre-fill label (PRE-05), with its tests.
2. `docs: S23 verified …` (this file and the rest of the docs; its hash is in the run's status line).

Next: `/build-session S24` (its brief now starts with finding 1).

## Verification 2 (22 Oct 2026, automatic run; the last the run allows)

**Verdict: Needs fixes** (one finding, small).

Verification 1's nine items are fixed as described in the build notes, and I checked each one in the code: the whole-answer lock (`answerChanges`), signatures/uploads/BMI out of pre-fill (`source`, `canAskFresh`), the wording rule for grids/ratings/lists, the lenient walk (`obj`), Go-only key checks shown in the Publish window, one read of the draft, and items 8–9 recorded. Every automated test passes. But Rahul's answer after verification 1 was "it should freeze all kind of answers", and **one kind of answer the editor offers as a simple setting still changes under a published key: a date's format.**

### Findings (worst first)

1. **A published date question can change between "Day, month and year", "Month and year" and "Year only", and still publish.** `apps/api/internal/keys/keys.go:193` (`kindOf`): all three are kind `date`, and `answerChanges` (`keys.go:440`) compares nothing else for a single date. `SameWording` (`keys.go:552`) also matches them by wording.
   - **Why it matters:** the stored answer changes shape: `"1990-05-01"` (date), `"1990-05"` (month) or `1990` (a number; `fields.ts` `dateField` makes Year only an `inputType: number` with `dateFormat: "y"`). S25 copies by key ("same key: always a match") and by wording, so it would put a full date into a year box. This is the same class as verification 1's number → date, which Rahul answered by freezing every kind of answer.
   - **Reproduced** on the running app (as Alex): "T23 Verify2 Date V2-1792675124" v1 published `q_v2date…` as Day, month and year; v2, the same key as Year only, **published (204)** with `question-keys` `problems: []`. In the editor, v3 (draft) Year only → Month and year: the Publish window said ready, with no problems. Turning on "Allow several dates" was refused correctly ("its kind of answer changed from date to list of dates"). A unit probe gave `LockProblems` = none for dmy → y and dmy → my, and `SameWording(dmy, y)` = true.
   - A list of dates is already covered (its column's `cellType:inputType` differs: `text:date`, `text:month`, `text:number`).
   - **Fix:** make a single date's format part of its answer: e.g. kind `date` with a format (`dmy`, `my`, `y`, read as `fields.ts` `dateSettingsOf` does), compared in `answerChanges` ("its date format changed from day, month and year to year only") and in `SameWording`. Unit tests in `keys_test.go` (`TestLockFreezesTheWholeAnswer`, `TestSameWording…`) and a case in `TestPublishKeyChecks`.
   - **Recommended in the same change: a text question's format** (Any text, Email, Phone, NHS number, Postcode, Custom: `inputType` and the email/regex validator, `fields.ts` `textFormatOf`) also defines what the answer can be, and an earlier "Phone number" answer copied into an "Email address" box would fail its check. Lock it too, or, if the builder thinks that's too wide, record why in `decisions.md`. A number's range and decimal places are limits, not the kind of answer; leave them unlocked and say so in the decisions entry.

**Minor (fix if cheap, else add to `pending.md`):**

2. **A change inside a repeating group is reported twice.** `keys.go:178` walks a dynamic panel's children as keyed questions, and `keys.go:453` also compares them as the panel's columns. Rewording an option of a child gives one problem for the child and one for "the column …" of the group. Report it once (e.g. skip the column comparison for a `paneldynamic`, whose children are checked on their own, or skip `InList` children in `LockProblems`).
3. **After a 422, `refused` sticks** (`apps/web/src/features/editor/PublishModal.tsx:46`): `problems` falls back to `refused` whenever Go's refetched list is empty, so Publish stays blocked until the window is reopened. Rare (it needs the trust to change between the two requests); clear `refused` when a fresh `keyProblems` arrives.
4. **`Duplicates` counts elements without a `name`** (`keys.go:346`): two unnamed elements (only possible through an import or the API) give "The question ID  is used twice". Skip empty names.
5. **`decisions.md`'s S23 entry still has superseded bullets:** the second ("options differ (added, removed or reworded)") and fifth ("grids, ratings and Yes/No switches have no options here") were replaced by the whole-answer lock after verification 1. Mark them superseded, as the 21 Oct entry was. `saas-requirements.md` §6 point 4's 22 Oct note mentions only options; add the whole answer.

### Reviewed and accepted (no action)

- **Q3 ("the editor refuses a duplicate … and Go's publish refuses one too"): met.** The editor never makes a duplicate (copying gives new IDs). If one arrives (API, hand-edited JSON), the Publish window shows Go's problem naming both sets, and Publish is disabled before the author clicks (T-09). One implementation in Go is simpler than two, and fits CLAUDE.md's "keep TypeScript to the frontend".
- **"Compared with every published question holding that key":** Go compares a draft with the newest published or retired version of **each** of the trust's questionnaires holding the key (`publishedQuestions`). Older versions agree with it by construction since the lock, and Rahul's answer after test report 3 says "keep the strict rule, as built". Fine.
- `/code-review high` also raised: a key published in disagreeing shapes blocks that key for good (Rahul chose this; old local data was cleaned); trust-wide names to a hospital-scoped super clinician (recorded, accepted in verification 1); the save/publish race (in `pending.md`); the route's cost on every save (in `pending.md`: add a key table if slow); and non-object chapter content making `Walk` fail (not reachable: `chapter.ValidateContent` refuses it on save). Also minor: `keyPlace` walks the doc for every card (fine at today's sizes) and Go's placeholder titles copy `doc.ts`'s `newElement` titles with nothing tying them together (worth a comment in `doc.ts` pointing at `keys.go`).

### Checks

- **Scope:** everything in the brief, following the answers where they differ (Q2, Q5, "after verification 1", "after test report 3"). Not in scope (library, pre-fill) untouched.
- **Tests first:** `internal/keys` unit tests (walk shapes, duplicates, whole-answer lock, disagreeing shapes, wording, one-off answers); `TestQuestionKeys` has the five cases (happy, no sign-in 401, wrong role 403, another trust 404 both ways, invalid ids); `TestPublishKeyChecks`, `TestNewVersionKeepsKeys`; `cmd/seed/keys_test.go`; Vitest for `alwaysFresh`; e2e `test_question_keys.py`.
- **Tenant isolation:** `ListTrustPublishedContents` is scoped by `q.org_id`. Checked on the running app: question-keys as Cara 403, Sue 200, Bea 404 (Trust A's path and her own trust's path with A's questionnaire), no sign-in 401; the patient router has no such route (404).
- **A-20 grep:** nothing in the new or changed files.
- **Security:** no secrets; the route has no patient data (not audited, correctly); `chapter.ForPatient` unchanged; `alwaysFresh` is a harmless hidden property in patient JSON. `repocheck` passes.
- **Docs:** session row "Tested", plan C4 "In progress: S23 tested", decisions and pending updated (see item 5).

### Test results

- `make lint`: passed.
- `make test`: Go all ok; Vitest web 204 passed (20 files), clinical 120 passed (16 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 18.2 s, keys 2.5 s), exit 0.
- `make e2e` (against the running `make dev` on the shared `sj` database): **56 passed, 2 failed, 5 skipped** (the QA-only Cognito journeys). Both failures are local data, not S23 code: J3 and `test_procedures` expect "Demo pre-op HQ v1", and test report 4's R-05 published the seeded Demo HQ as **v2**, so the picker shows "Demo pre-op HQ v2". The builder's run on a fresh database passed 58. *For the retest:* run `make e2e` on a fresh database (as the builder does), or reset `sj`; and consider making those two journeys pick "Demo pre-op HQ" by name rather than "v1" (worth a `pending.md` line).
- **Browser (Chrome, as Alex):** the date-format reproduction above; the Publish window showed Go's kind-of-answer problem with Publish disabled; the card's "New question: won't pre-fill" label for a key only this HQ has published. No console errors.

### Test data left by the verifier

"T23 Verify2 Date V2-1792675124" (Trust A, Hospital A1 list): v1 and v2 published, v3 draft with "Allow several dates" on. Made up; delete or ignore.

### Hand checks for Rahul

- None new.

### Suggested commits (once fixed and verified)

1. `S23: question keys — internal/keys (walk, duplicates, whole-answer lock, wording matches), publish refuses key problems (422), GET …/question-keys (matches and problems), Always ask fresh, the editor's pre-fill label (PRE-05)`, with its tests: Go unit (`internal/keys`, `cmd/seed/keys_test.go`) and integration (`question_keys_integration_test.go`, the adjusted `trusts_integration_test.go`), Vitest (`fields.test.ts`, `publish.test.ts`), the e2e journey `test_question_keys.py` and its `e2e/README.md` row.
2. `docs: S23 verified — build notes, test plan and reports, answers for S23–S26, verification, decisions, pending, plan and session status`, including the conductor's recording of the S23–S26 answers (`decisions.md` 22 Oct entry, `saas-requirements.md` §6, `sessions/README.md`, `plans/c4`, the `answers.md` files of S24–S26) and the obsolete note on `S02-workflow-route-tests/make-test-hq.js`.

**For the builder:** "Verification found problems: read `docs/sessions/S23-question-keys/verify.md`, fix them test-first, then ask me for a retest."

## Verification 1 (22 Oct 2026, automatic run)

**Verdict: Needs fixes.**

The build matches the brief and Rahul's 22 Oct answers: options are frozen (values and wording), keys must be unique across the HQ, the label matches by key or by wording, and there's "Always ask fresh". Every automated test passes, the route has its five integration cases, and tenant isolation holds. But this session exists to make keys trustworthy **before** S25 copies answers by them. Four gaps let a key, or a wording match, claim the same meaning when it doesn't. One malformed but valid chapter also turns publishing into a 500. Each fix is small. I reproduced all four against a freshly seeded scratch API (`sj_verify23`, since dropped).

### Findings (worst first)

1. **A published key can change its kind of answer, and the HQ still publishes.** `apps/api/internal/keys/keys.go:285` (`LockProblems`) and `packages/clinical/src/keys.ts` (`optionLockProblems`) compare options only.
   - Reproduced: HQ A published `q_kind` as Select One (Yes/No) and `q_wt` as Number (kg). HQ B used the same keys, with `q_kind` as Select Many with the same options and `q_wt` as Date. **Both published (204).**
   - Why it matters: S25 copies by key with "same key: always a match". It would put a single value into a multiple-choice question, or a weight into a date. That's exactly the meaning drift that Q2 ("frozen completely") is meant to stop.
   - **Fix (recommended, in the spirit of Q2):** for a published key, the lock also refuses a change in the **kind of answer** (Go's `Kind`: single, multiple, text, number, date, dates, list, …) and in a number's **unit**. The message should name the question, e.g. "“Weight”: its kind of answer changed from number to date. A published question's kind of answer can't change (published in …)". Use the same message in TypeScript. Add unit and integration tests, as `TestPublishKeyChecks` does for options.
   - Also lock the parts of the answer the builder left out as "not options" (build notes, "Decided by the builder"): a **grid's rows and columns** (the stored answer is `{row: column}`) and a **rating's scale** (`rateMin`, `rateMax`, `rateValues`). Rahul's answer froze "the options", and these are a grid's and a rating's options. If the builder thinks this is too wide, record it as a question for Rahul in `answers.md` and lock at least the kind.
2. **Signature and file-upload questions match by wording.** `keys.go:407` (`source`) and `keys.go:349` (`placeholders`).
   - Reproduced: two HQs, each with the editor's default signature "Please sign below". The draft's signature showed "Pre-fills from 1 other HQ · by same wording".
   - Why it matters: a signature attests to one document. Copying it into another HQ (S25 reuses `Matches`) breaks "If in doubt, don't fill". An uploaded file is the same kind of one-off evidence.
   - **Fix:** `signaturepad` and `file` never take part in pre-fill. Leave them out of `source` (no label, by key or by wording), and leave them out of `canAskFresh` (`packages/clinical/src/keys.ts:90`), because they're never pre-filled anyway. Add unit tests.
3. **Grids and ratings match by wording without comparing what can be answered.** `keys.go:358` (`SameWording`).
   - Reproduced: "How bad is your pain?" on 0–10 matched the same text on 1–5. "How often do you" grids with different rows (Run / Swim) and columns (Never / Always) matched too. Both showed "by same wording".
   - Why it matters: Rahul's rule is the same kind of answer with every option text the same. A 1–5 scale can't hold an 8.
   - **Fix:** for `matrix`, compare row texts and column texts. For `rating`, compare `rateMin`, `rateMax` and `rateStep`, or the `rateValues` texts. The simpler alternative, which "If in doubt, don't fill" also allows, is that grids and ratings never match by wording, only by key.
4. **One valid but differently shaped chapter makes `keys.Walk` fail, so publish returns 500.** `keys.go:101`: `Columns` is decoded as `[]struct{Name, Title}`. SurveyJS also allows plain strings (`"columns": ["Never", "Sometimes"]`), the shape SurveyJS itself writes for an item with no separate text.
   - Reproduced: a draft with such a grid saved fine (200), but **Publish answered 500** instead of publishing or giving a 422.
   - Why it matters: `ListTrustPublishedContents` walks every published chapter in the trust. If such a chapter is ever published (imported, or published before S23), **every publish and every `question-keys` request in that trust returns 500**.
   - **Fix:** decode `columns` (and any other list the walk reads) as `[]json.RawMessage`, accepting a string or an object as `options` already does. Add a unit test with string columns.

**Minor (fix if cheap, else add to `pending.md`):**

5. **After Go's 422, the Publish window keeps stale `published` keys.** `apps/web/src/features/editor/PublishModal.tsx:53`: `refused` clears when the window is reopened, but the stale keys still pass, so it shows "ready" again. Re-fetch the keys after a 422, for example through `onPublished`, or with a `refresh` callback that bumps `useQuestionKeys`.
6. **Publish reads the draft's chapters twice inside the trust lock.** `apps/api/internal/httpapi/questionnaires.go:220`: `ListChapters` is used only for its length, and `keyProblems` then runs `ListVersionContents`. Use one of them.
7. **TypeScript and Go differ on edge cases.**
   - `keys.ts:36` turns a choice with no `value` into the value "undefined", where Go skips it.
   - `canAskFresh` leaves out `bmi`, but Go counts a BMI question as a pre-fill source.
   - Make them agree. The file says the messages are "Go's, word for word".
8. **A pre-existing race, now more visible:** a chapter save that has passed `canEdit` (`chapters.go:117`, status still draft) can commit between the key checks and `PublishVersion`. The race existed before S23, and the browser's checks had the same gap. Record it in `pending.md`; no fix is needed now.
9. **The question-keys route lists the names of all the trust's questionnaires to a hospital-scoped super clinician,** including other hospitals' own HQs. This was a builder decision (build notes). No patient data is involved, and pre-fill is trust-wide, so it's acceptable. Record it in `decisions.md` with the S23 entry if it isn't there already.

Already in `pending.md` (no action): response size, keyboard access to the badge, and the label not warning about a link whose options changed.

### Checks

- **Scope:** everything in the brief is built, following the answers where they differ (Q2, Q5). The route moved under `…/h/{hid}/questionnaires/{qid}/question-keys`; this is reasoned in the build notes and fine.
- **Tests first:** `internal/keys` and `keys.test.ts` have unit tests. `TestQuestionKeys` covers the happy path, no sign-in, wrong role (hospital and trust clinician), another trust (404 both ways) and invalid ids. `TestPublishKeyChecks` and `TestNewVersionKeepsKeys` are there. The e2e journey is `e2e/journeys/test_question_keys.py`.
- **Tenant isolation:** `ListTrustPublishedContents` is scoped by `q.org_id`. Through the API, Bea gets 404 on Trust A's questionnaire.
- **A-20 grep:** nothing new in the diff or the new files.
- **Security:** nothing is secret. There's no patient data in the route (correctly not audited). The patient router is unchanged. `alwaysFresh` is a harmless hidden property in patient JSON.
- **Org-name grep:** clean.
- **Docs:** the session's row is "Tested". `decisions.md` has the 22 Oct answers, and `pending.md` has test report 1's items.

### Test results

- `make lint`: passed.
- `make test`:
  - Go: all ok.
  - Vitest: web 204 passed (20 files), clinical 130 passed (16 files).
  - CDK: 124 passed.
- `make test-integration`: every package ok (httpapi 13.2 s, keys 3.3 s), exit 0.
- `make e2e`: not re-run by the verifier, because the verdict is Needs fixes and the retest re-runs it. The builder's run: 58 passed, 5 skipped.
- **Scratch API (fresh migrate and seed, :8097):**
  - **R-05 passed:** Demo pre-op HQ, Create new version, Publish unchanged: **204**. `question-keys` was 200 with 3 questions and 4 published keys.
  - question-keys: Cara 403, Bea 404, no sign-in 401.
  - The four reproductions above.
- **Browser (Chrome, as Alex, at :3000):**
  - T23 Target 41, Question Set 2: both cards show "Pre-fills from 1 other HQ". The list says "T23 Source 41 · by link".
  - T23 Fresh 41 v2: the orange "Always ask fresh" badge is shown. The statement and the clinician calculation have no label. The repeating group says "New question: won't pre-fill", and its child has no label.
  - No console errors.

### Hand checks for Rahul (carried over; they don't block)

- R-02 at 390×844: the patient's invite for an HQ with "Always ask fresh" on looks as before. The tester's window wouldn't resize.

### Suggested commits (once fixed and verified)

1. `S23: question keys — internal/keys (walk, duplicates, option lock, wording matches), publish refuses key problems (422), GET …/question-keys, Always ask fresh, the editor's pre-fill label (PRE-05)`, with its tests (Go unit and integration, Vitest, the e2e journey).
2. `docs: S23 verified — build notes, test plan and reports, answers for S23–S26, verification, decisions, pending, plan and session status`. This includes the conductor's recording of the S23–S26 answers: the `decisions.md` 22 Oct entry, `saas-requirements.md` §6, `sessions/README.md`, `plans/c4`, and the `answers.md` files of S24–S26.

**For the builder:** "Verification found problems: read `docs/sessions/S23-question-keys/verify.md`, fix them test-first, then ask me for a retest."
