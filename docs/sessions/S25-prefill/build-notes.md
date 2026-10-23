# S25 build notes: pre-fill and the patient's confirmation screens

Builder, automatic run (`/run-sessions S23 S26`), 23 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md); where they differ from the brief, the build follows them:
- **Matching is by key first, then by exact wording** (S23 answers): `internal/prefill` reuses S23's `keys.SameWording` and `keys.RepeatedTexts`, so it needs each earlier form's chapter JSON (question and option texts), not just its answer keys. A wording match maps a choice answer by option **text** to the new question's own value.
- `prefill` stores `{key: {fromForm, answeredAt, value, matchedBy}}` (Q4 plus `matchedBy`), and the button pressed per set (Q5).
- Everything else as recommended: pre-fill on the first open of each set, by the patient or by staff for them; every submitted HQ form of the patient's records joined to the same account in the same trust (no account: the record only), newest submit first, any episode status; the clinician's validated copy first; never clinician-only, Always ask fresh or the profile; the opening card with both buttons; the marker.

## What was built

### Go: `internal/prefill` (new, unit-tested)
`prefill.Fill(target, sets, sources)`: for the new form's version (every set, so texts used twice anywhere in it are known) and the patient's earlier forms, the pre-filled data and the `prefill` map for each set asked for. A plain JSON walk on top of `internal/keys`; it never evaluates SurveyJS.
- A question takes part if `keys.TakesPart` (a patient's top-level answer, not a signature, upload or BMI block; S23's `source`, now exported) and it isn't **Always ask fresh**.
- For each such question, the earlier forms newest submit first: the first form that asked the **same key**, else the **same wording** (`keys.SameWording`, never a text used twice in either questionnaire), **decides**.
- Its answer comes from the set's **validated clinician copy** if there is one, else the patient's. If it's empty, or doesn't fit, **nothing is filled** (no older form is tried).
- **Fitting** (`fit`): the same kind of answer, unit, format and scale; a choice value must be one of the new question's options (by key: the same value; by wording: the option with the same text, written as the new question's own value, numbers kept as numbers); multiple choice all or nothing; a grid's rows and columns likewise; a repeating group's or list's rows copied whole, keeping only the new one's columns (matched by name, or by text for a wording match), each cell fitted as a question; anything else (a medication list) only into the same answer (`keys.SameAnswer`, the lock's comparison, now exported).
- **"Other" and comment text** (`<key>-Comment`) travel with their question, only where the new question has Other (and it's chosen) or a comment box; recorded as `comment` in the entry.
- **Clinician-only questions inside a repeating group** are neither filled nor copied from (found while testing with the integration fixture's clinician-only "dose check" column).

### Go: storage and routes
- **Migration 00025** (`episode_answers`): `prefill jsonb` (the map, never changed afterwards), `prefill_choice` (`check_each` / `still_correct`), `prefill_choice_at`, `prefill_choice_by` (staff for the patient; NULL: the patient).
- **Queries:** `ListPrefillSources` (one query, the trust's `org_id` as a parameter, newest first: each submitted HQ form's chapters with the patient's answers and the validated clinician copy); `PrefillPatientAnswers` (insert as revision 0, only while the form is open, `ON CONFLICT DO NOTHING`); `RecordPrefillChoice` (first button kept). `ListEpisodeAnswers` returns the new columns. sqlc maps a nullable `jsonb` to `json.RawMessage`.
- **`writeForm`** (the patient's `GET /api/p/episodes/{eid}` and staff's `GET …/patient-form`) pre-fills the open form's patient sets that have no patient row, then answers with `prefill: {chapterId: {answers, choice, choiceAt}}`. Still audited (`link.view` / `behalf.view`); the earlier forms' episodes are added to that row (their answers were read).
- **New routes** (audited): `POST /api/p/episodes/{eid}/prefill/{cid}` (`link.prefill`) and `POST /api/o/{oid}/h/{hid}/episodes/{eid}/patient-form/prefill/{cid}` (`behalf.prefill`), body `{choice}`. 204; the first button is kept and a repeat is 204; 400 for another choice; 404 for a set that isn't the form's patient set; 409 when nothing was pre-filled in it or the form has been sent.
- **"Started"** (reminders, `GetFollowup`, and the home's In progress, `ListHome`) now means a patient row with **revision > 0**, or (after test report 1) **a card answered**. A pre-filled set the patient hasn't touched doesn't stop reminders.

### TypeScript (for Rahul to review)
- **`apps/web/src/features/patient/prefill.ts`** (new, Vitest `prefill.test.ts`):
  - `cardText`: the card's sentence, one date or "the latest on …", and "Sam's … with Sam" for staff.
  - `shownPrefilled`: the questions shown (SurveyJS's `getAllQuestions(true)`) that still hold their pre-filled answer. Only these are counted, so an answer a condition hides isn't counted.
  - `firstUnfilled`: where "Everything is still correct" goes; statements, calculations and the details box are skipped.
  - `awaitsCheck`: whether a set still waits for its card.
  - `stillPrefilled`: the answer equals the copied value.
  - `showPrefillMarkers`: adds and removes the marker under each question, as the clinician's corrections do (`corrections.ts`).
- **`PatientHQ.tsx`:**
  - the list shows **Please check** / **Check** on a set waiting for its card, which isn't complete until then (so Send stays off);
  - `SetForm` opens on the card, records the button (`POST …/prefill/{cid}`), then shows the survey: at the first page (Check each answer), or at the first unfilled question, focused (Everything is still correct), or opens the next set (or the list) when nothing is left.
  - The markers are added when the model is built, so the first draw has them.
- **`lib/format.ts`** `formatDay` ("3 Mar 2026" in en-GB); **`globals.css`** `.sj-prefill-marker`.

### Seed
**Demo follow-up HQ** (`cmd/seed/demo-followup-hq.json`, Trust A only, fixed ids `…0301`):
- it asks the Demo pre-op HQ's smoking, alcohol, height, weight, snoring, pregnancy, sickness after an anaesthetic, the anaesthetic-problem pair, allergies and regular medicines **exactly** (same IDs);
- "Do you have diabetes?" under an ID of its own (`fu_diabetes`), for a wording match;
- a new required question ("Can you climb two flights of stairs…"), the details box, and the Nurse checks set.

The brief's manual test focus asked for a second HQ sharing keys. `cmd/seed/keys_test.go` checks it agrees with the Demo HQ and the e2e data on shared keys, and what `prefill.Fill` copies from one to the other.

## Hidden answers (`clearInvisibleValues`), as the brief asked
SurveyJS's default (`onComplete`) is unchanged.
- A pre-filled answer to a question its condition hides **stays in the answers**: it's saved as it is, and back when the question shows again (unit test and J16).
- It's cleared only if the patient finishes the section with it still hidden, as for any answer.
- It isn't counted: the card counts the questions shown; outputs and the POA Summary already skip hidden questions.
- If the patient sends without finishing that section, a hidden pre-filled answer is sent like any hidden typed answer, and stays out of outputs and the POA Summary as before.

## Decided by the builder (automatic mode)
- **The newest earlier form that asked a question decides.** If it has no answer there (skipped, or hidden then), or the answer doesn't fit, nothing is filled, rather than an older answer that may no longer be true ("if in doubt, don't fill"). Within a form, a key match beats a wording match; across forms, the newest wins.
- **The clinician's validated copy is used whole, per set:** an answer the clinician cleared isn't filled from the patient's. (The clinician's copy starts as the patient's answers, so a missing key was removed by the clinician.)
- **A set with nothing to fill gets no row**, so a later open may fill it if the patient still hasn't saved anything there. "Once per set" holds for every set something was filled in, and nothing saved is ever filled over. This also leaves earlier tests' row counts as they were.
- **The button is stored in columns beside `prefill`**, not inside the map (a question named like the field can't collide). `by` is kept too.
- **A pre-filled set must be checked before it counts as complete:** Send stays disabled until each set showing pre-filled answers has had its card answered (safety: the patient is asked to check). A set whose pre-filled answers are all hidden needs no card.
- **"Everything is still correct" with nothing left** opens the next set shown; after the last set, the list.
- **Staff wording:** "We've filled in 3 answers from Sam's questionnaire on … Please check them with Sam." and "From Sam's answers on …".
- **Pre-fill reads only the form's own trust** (`ListPrefillSources` takes `org_id`); the open's audit row lists the earlier episodes read. S26 widens it deliberately.
- **Two opens at the same moment** (two tabs, or React's double effect in development, found by J16): the second stores nothing but re-reads, so it sends what the first stored (integration test with `meanwhile`).
- **`support.choose` (e2e) now matches the radio exactly:** "No" also matched "I'm not sure". **`test_question_library.py`** no longer expects the candidate's sources to start "In Demo pre-op HQ", since the follow-up HQ also asks `pregnant`.
- **The TypeScript key walk isn't repeated:** pre-fill runs only in Go; the browser only reads `prefill`.

## Tests
- **Go unit:**
  - `internal/prefill/prefill_test.go`: by key; precedence (clinician copy, a cleared answer, newest first, the newest form that asked decides, an older form when the newer didn't ask); never filled (clinician-only on a question, group, page or set; fresh; profile; statements; signature, upload, BMI; only the sets asked; a clinician-only source); choice values (removed value, multiple with Other text, one misfit, Other text without Other, comment box, Yes/No switch, grid); wording (mapped value, numeric value, missing option text, different kind, unit, repeated texts on either side, placeholder text); repeating groups and lists (by key, fewer columns, by wording, column texts differ, a misfit cell, a row question alone, clinician-only row question).
  - `cmd/seed/keys_test.go`: the follow-up HQ agrees on keys; `TestFollowUpHQPrefillsFromTheDemoHQ`.
- **Integration:** `internal/httpapi/prefill_integration_test.go`:
  - `TestPrefill`:
    - the same trust's answers, the clinician's copy first, by key and wording; not fresh or clinician-only;
    - revision 0, `prefill` stored; the home shows To do and reminders don't count it as started;
    - a second open doesn't pre-fill again, and nothing typed is overwritten;
    - two opens at once;
    - a set answered before the first open;
    - newest first, from a cancelled episode, with the audit row's earlier episodes;
    - joined records count, others don't;
    - another trust's answers are never used;
    - staff for the patient;
    - a sent form is never pre-filled.
  - `TestPrefillChoice`: the five cases for both routes (no sign-in; another patient's session or another hospital's clinician; another trust; invalid choice, body, set ids, nothing pre-filled), the first button kept, staff marked, audited, 409 once sent.
- **Vitest:** `features/patient/prefill.test.ts` (13), `lib/format.test.ts` (`formatDay`).
- **E2E:** `journeys/test_j16_prefill.py` (two tests, phone size; README row).

## Results
- `make lint`: passed.
- `make test`: Go all ok; Vitest 218 (web) + 140 (clinical); CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` on my own instance (database `sj_s25`, API :8095, Vite :3105): **60 passed, 1 failed, 5 skipped** on the first full run. The failure was `test_question_library.py`, which expected only "In Demo pre-op HQ" (fixed above); it and `test_question_keys.py` then passed, and J16 passed 5 times running.
- The local `sj` database is migrated to 00025 and seeded with the follow-up HQ. HJE Full HQ is untouched.

## Suggested commits
1. `S25: pre-fill from the same trust (internal/prefill, by key then wording), episode_answers.prefill and the card's button (migration 00025), the patient's opening card, Please check and markers, the seeded Demo follow-up HQ`, with its tests:
   - Go unit (`internal/prefill`, `cmd/seed`);
   - integration (`prefill_integration_test.go`, fixture `prefill.json`);
   - Vitest (`prefill.test.ts`, `format.test.ts`);
   - e2e (`test_j16_prefill.py`, `data/prefill-hq.json`, README row, `support.choose`, `test_question_library.py`).
2. `docs: S25 …` (build notes, test plan and reports, decisions, pending, plan and session status) once verified.

## Test report 1 (23 Oct): 18 of 19 passed; fixes

1. **F1: after "Everything is still correct" the question's text was scrolled off the top of a phone screen** (T-03).
   - **Reproduced** in J16 at iPhone 13 size: a new check that the question's top is on screen failed at −32 px.
   - **Cause:** SurveyJS's `question.focus()` scrolls to the input it focuses; then SurveyJS settles its phone layout just after first drawing, which moves the content.
   - **Fix:** `focusQuestion` (`prefill.ts`) scrolls the question itself to the top, focuses its input without scrolling again, and keeps it there while the page's size settles, for a second at most or until the patient scrolls or types.
   - J16 now checks the question's top is on screen after that second (passes 4 runs of 4).
2. **Found 1: confirming didn't count as started.** A section answered with "Everything is still correct" was Complete on the list, but the home said To do and the reminders still went. That's wrong: the patient has acted on it.
   - **Fix:** "started" (`ListHome`, `GetFollowup`) is now a patient row with revision > 0 **or a card answered** (`prefill_choice`). An untouched pre-filled HQ still gets reminders.
   - Test first: `TestPrefillChoice`'s happy path now checks the home says In progress and the reminders see it as started after the button (it failed before the fix).
3. **Found 2: the follow-up HQ's details box was headed "profile"** (no title in the seed).
   - **Fix:** it's titled **Your details**. A new seed test, `TestSeededHQsHaveTitles`, fails on any untitled element the patient sees.
   - The seed never changes a published HQ, so the copies already in the local `sj` and `sj_s25` databases were given the title with one `UPDATE` each (local test data only, cosmetic, no answers affected).
   - The lowercase "Sex female" comes from the profile block's template (older than S25); it's in pending.md.
4. Found 3 (behaviour as expected) and the "Pre-fills from 100 other HQs" label (S23's) need no change.

**Results after the fixes:**
- `make lint`: passed.
- `make test`: Go all ok; Vitest 218 + 140; CDK 124.
- `make test-integration`: 24 packages ok.
- `make e2e` on my own instance (`sj_s25`, :8095/:3105): 61 passed, 1 failed, 5 skipped. The failure was `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`: its hover tooltip didn't appear. It passed on its own, and the whole file passed once and failed once more on reruns. It's S11's audit page, which S25 doesn't touch: a timing flake.

## Test report 2 (23 Oct): all passed
- **Result:** 19 of 19, with the fixes for F1, found 1 and found 2 confirmed in the browser (T-03 also at 375 × 667).
- **Exploring:** nothing new. "Sex female" in lower case and S23's "Pre-fills from 100 other HQs" stay in pending.md.
- **S25 is tested**; next is `/verify-session S25`.

## Hand checks for Rahul
- **Optional:** F1 in Chrome's real device toolbar at 390 × 844. After **Everything is still correct** on *Your health*, the pregnancy question's text should be on screen.
  - The testers used a 390-wide iframe, because resizing the window had no effect. That method reproduced the bug in report 1 and passes in report 2.
  - J16 (iPhone 13, WebKit) checks it on every run.

## Suggested commits (final)
1. `S25: pre-fill from the same trust (internal/prefill, by key then wording), episode_answers.prefill and the card's button (migration 00025), the patient's opening card, Please check and markers, started = saved or confirmed, the seeded Demo follow-up HQ`, with its tests:
   - Go unit tests (`internal/prefill`, `cmd/seed`);
   - integration tests (`prefill_integration_test.go`, fixture `prefill.json`);
   - Vitest (`prefill.test.ts`, `format.test.ts`);
   - the e2e journey (`test_j16_prefill.py`, `data/prefill-hq.json`, its README row, `support.choose`, `test_question_library.py`).
2. `docs: S25 tested — build notes, test plan and reports, decisions, pending, CLAUDE.md, apps/web/AGENTS.md, plan and session status`.
