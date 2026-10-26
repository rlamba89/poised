# S39 build notes: the ASA guide and the suggested ASA

Builder, automatic run, 26 Oct 2026. Rahul's answers ([answers.md](answers.md)) replace the brief's questions:
1. **The usual grade comes from a file built from the ASA's own published examples** (the 2020 statement); where the grade depends on severity the **author picks which**; no silent single grade; the anaesthetist gives the final grade.
2. **The suggested ASA can be used in conditions on clinician sets** (not the brief's recommendation).

## What was built

### Go: the ASA guide (`apps/api/internal/asaguide`, new)

- `guide.json` (embedded): the ASA statement's examples (adults, children), each with its grade, its words, whose example it is (Adults, Children, Pregnancy) and the seeded codes it covers, e.g. diabetes codes → II "Well-controlled diabetes" and III "Poorly controlled diabetes" (adults), II/III non-insulin/insulin-dependent (children); COPD → III only; heart failure → III/IV (adults), IV (children); MI, CVA, TIA, stents → III (more than 3 months) or IV (less); angina → III or IV "ongoing cardiac ischaemia"; unstable angina → IV; sepsis → IV; smoker → II; alcohol → II social drinker or III dependence.
- `ChoicesFor(set, code)`: the examples for a code, lowest grade first, `[]` for none.
- `GET /api/codes` (`internal/httpapi/codes.go`) sends each row with `asaChoices: [{grade, example, group}]`.
- Tests: `asaguide_test.go` (choices, ordering, exact match, a copy; **every code in the file is in `db/seed/codes/codes.csv`**, every grade I–VI, ids unique); `TestSearchCodes` gets a happy-path subtest for the field (the route's five cases already existed).

### `packages/clinical` (Vitest, test-first)

- **`asa.ts` (new; types only imported, so `outputs.ts` and `respondent.ts` both use it):**
  - `AsaChoice`; `afterCodeAdded(asa, guide, added)`: the editor's rule (keep a grade already there; one example → its grade and words as `reason`; several → nothing, `mustChoose`); `gradePicked(grade, emergency, guide)`: a grade picked by hand takes the guide's words only when exactly one example has that grade.
  - `suggestAsa(outputs)`: the highest grade, each disclosure at it as a reason (each once), **E from any disclosure** (one below the top listed "(emergency)"). A reason in words: the codes ("Heart failure unspecified (I50.9)"), else the note, else "question: answer", plus " – <the guide's example>".
  - `suggestedAsaText`, `asaHeading` ("Suggested ASA III", "Suggested ASA IIIE"), `ASA_GRADES`, `asaRank`.
- `types.ts`: a disclosure's `asa` gets an optional `reason` (copied, so a published HQ never changes).
- `outputs.ts`: `computeOutputs(...).suggestedAsa` is `suggestAsa`'s (with reasons). Test-case lines are unchanged ("Suggested ASA III").
- `respondent.ts`:
  - **`hqSuggestedAsa(sets, answers, viewer, patient)`**: the whole HQ's, on `hqSets` (S38's helper), so a hidden set or question doesn't count and a score uses earlier sets' answers as everywhere.
  - `hqSets`: when a set's condition tests `{suggestedAsa}`, each set's condition sees the suggested ASA from the sets shown before it, as a number (III → 3), **0 for none**.
- `doc.ts` `SUGGESTED_ASA`; `logic.ts`: a subject type `asa` ("Suggested ASA" is at least / at most / is I–VI, stored as a number), offered by `subjectsFor(…, {suggestedAsa: true})` and described in words; `questionSets.ts` `chapterConditionProblems(doc, earlier, audience)`: a patient set's condition testing it is a problem; `publishProblems` takes each set's `audience`.
- Tests: `asa.test.ts` (16: the editor's rules; whole HQ: II + III → III with both III reasons, none, E kept, a hidden set and a hidden question, a band's disclosure, ties, the patient's view, the preview's; the condition: shown from III, 0 for none, offered only with the flag, in words, publish problem).

### `apps/web`

- **Editor:** `CodePicker` shows "· ASA II or III" on a result and passes its choices (`onAdded`). `OutputEditor`: an **"ASA from the ASA guide"** box (radio list of the added codes' examples, plus **No ASA grade**), filled in by one example, pale yellow and required ("Choose the ASA grade from the ASA guide, or No ASA grade.") when there are several; the reason line under it. `OutputSummary` shows the reason. "From the code library" (options from codes) gives single-example codes their grade.
- **Clinician set conditions:** `Canvas` passes `suggestedAsa` to the Question Set's condition builder when the set isn't a patient set (and a note says what it is); `ConditionBuilder`'s value input `asa`. The header's problems and the Publish window check the audience.
- **`episodes/SuggestedAsa.tsx` (new):** the POA Summary's ASA section shows "Suggested ASA III", each reason (a link to Validate at its question, as S27's answers), and "Suggested from the answers. The nurse and the anaesthetist give the ASA grade."; **on both tabs** (validated: the answers that count, clinician view; patient: the patient's sets and answers). The nurse's and anaesthetist's grades stay on the validated tab only. "No suggested ASA: no answer gives an ASA grade." when none.
- **Episode page:** one line at the top of the Health questionnaire card (the Validate list) once the patient has sent: "Suggested ASA III: … (the anaesthetist confirms)".
- **Preview:** the reasons under the one-set suggested ASA.

### E2E

- `e2e/data/asa-hq.json`: About you (smoker, ASA II), Your heart (heart failure, ASA III with its example), Anaesthetic plan (clinician, shown when `{suggestedAsa} >= 3`).
- **J5** `test_the_suggested_asa_with_its_reasons_on_both_tabs`: the episode page's line and the plan set; the POA Summary's ASA III and reason on both tabs (the smoking reason not listed), the grades only on the validated tab; the reason opens Validate; Cara corrects heart failure → validated ASA II (the smoking reason), patient's still III, the plan set gone; smoking corrected too → "No suggested ASA".
- **J10** `test_the_asa_guide_fills_in_or_asks_for_the_grade`: E11.9's examples, nothing filled, Save refused, III chosen with its reason; COPD fills III by itself, changed by hand to IV (reason gone); kept after reload; the clinician set's condition shows "Suggested ASA / is at least / III"; a patient set's list doesn't offer it.

## Decided by the builder (automatic mode)

1. **No `codes.asa` column or migration** (the brief's): one code can have several grades under Rahul's answer, so the guide is a Go file served with the code search. Nothing to seed; the S41 evaluate service doesn't need it (the grade is copied into the disclosure).
2. **The guide's content** follows the ASA's 2020 examples; codes are mapped only where an example clearly applies. Codes whose only example is a **single children's one** (AF, AICD, autism, cancer, thrombophilia in pregnancy) are left out, so a child's grade never fills in an adult form by itself. **A clinician must check the file before go-live** (pending.md).
3. **Several examples → Save is refused until the author picks one or "No ASA grade"**: "no silent single grade", and no silently missed one either.
4. **The reason's words:** "codes (code) – example", else the note, else "question: answer". Joined with "; " in one line.
5. **E (emergency) from any disclosure counted**, not only the top grade's (before S39 an E below the top was dropped). Written "IIIE", as test cases already write it.
6. **`{suggestedAsa}` is 0 when there's no grade**, so "is at most II" includes no grade and "is at least III" doesn't. It counts the sets shown before the clinician set (as every Question Set condition does), in the viewer's view.
7. **"The episode page's Validate view"** read as the episode page's Health questionnaire card (where each set's Validate button is), shown once the patient has sent. The Validate page of one set has no line (it would go stale as the clinician edits).
8. **The POA Summary's patient tab now has the ASA section**, with only the suggestion (the grade boxes stay on the validated tab).
9. **Not done:** the research's "BMI bands with ASA in the starter library" (no BMI Calculation in the library; not in Rahul's answer). A band's disclosure gets its grade by hand.

## TypeScript for Rahul to review

- `packages/clinical/src/asa.ts`: `afterCodeAdded`, `gradePicked` (the editor's rule) and `suggestAsa` (the highest grade, the reasons, E).
- `packages/clinical/src/respondent.ts`: `hqSuggestedAsa`, and the `{suggestedAsa}` part of `hqSets`.
- `apps/web/src/features/outputs/OutputEditor.tsx`: the guide box and the Save check.

## Tests

- `make lint`: clean.
- `make test`: Go ok (new `internal/asaguide`); Vitest: clinical **371** passed (16 new in `asa.test.ts`), web 270 passed; CDK 124 passed.
- `make test-integration`: every package ok (`TestSearchCodes` with the new subtest).
- `make e2e` (every journey, on a private stack on :3093/:8093 on the shared `sj` database, stopped afterwards): **89 passed, 5 skipped (Cognito-only), 1 failed**: J7's text count found 3 texts to its made-up mobile (the known reused-fictional-number flake noted in S38), not S39. Re-run alone, J7: **6 passed**. The new J5 and J10 tests passed.

## Hand checks for Rahul

- None expected beyond the test plan.

## Suggested commits

1. `S39: the ASA guide in the code search` — `apps/api/internal/asaguide/{guide.json,asaguide.go,asaguide_test.go}`, `apps/api/internal/httpapi/{codes.go,codes_integration_test.go}`.
2. `S39: the suggested ASA with its reasons, and in clinician set conditions` — `packages/clinical/src/{asa,asa.test,types,outputs,outputs.test,respondent,doc,logic,questionSets,publish,index}.ts`; `apps/web/src/features/outputs/{CodePicker,OutputEditor,OutputSummary}.tsx`, `features/editor/{OptionsEditor,ConditionBuilder,LogicEditor,Canvas,QuestionnaireEditor,PublishModal}.tsx`, `features/episodes/{SuggestedAsa,PoaSummary,EpisodePage}.tsx`, `features/preview/PreviewPage.tsx`.
3. `S39: J5 and J10 extended` — `e2e/journeys/{test_j5_poa_summary,test_j10_authoring}.py`, `e2e/data/asa-hq.json`.
4. `S39: docs` — this folder, `CLAUDE.md`, `apps/web/AGENTS.md`, `docs/decisions.md`, `docs/pending.md`, plan C6's status, the plans and sessions tables.

## Test report 1 (22 of 22 passed; 2 Minor failures found while exploring, fixed test-first)

- **F1: removing the code a grade came from left its grade and reason** (saved as "Smoker · ASA III (COPD)"). New `afterCodeRemoved(asa, removed, left)` (`asa.ts`): a grade taken from one of the removed code's examples goes with it unless a code left offers the same example; the codes left then decide as when adding (one example fills in, several must be chosen). The author's own grade, or another code's, is kept. `OutputEditor` applies it when a code's ✕ is clicked.
- **F2: an option's disclosure made by "From the code library" (or any disclosure edited later) offered no ASA guide choice**, so the window's "choose it in the option's disclosure" couldn't be followed, and a grade picked by hand got no reason. `OutputEditor` now looks up the examples of the codes already on the disclosure when it opens (`asaChoicesOf` in `CodePicker.tsx`, the code search by its code; no new route), so the guide box, a grade picked by hand and removing a code work for them too. Opening an older disclosure with no grade shows the box but doesn't require a choice (only adding a code does).
- **C1:** choosing a grade now clears "Choose the ASA grade…". **C2:** the episode page's "No suggested ASA" line no longer says "(the anaesthetist confirms)". **C3:** a patient set whose condition tests the suggested ASA (a publish problem) still shows "Suggested ASA" in its condition, so it can be seen and removed.
- **Not changed:** the POA Summary saying "No answers produce anything for the summary yet." while the ASA section lists a code-only disclosure (the summary lists notes, as before S39); "Patients see" and the PATIENT label on a clinician set (as before). R-05 at a true 390×844 window is a hand check for Rahul (the e2e patient journeys run at iPhone size).
- Tests: `asa.test.ts` "removing the code a grade came from…" (7 cases); J10 `test_the_asa_guide_fills_in_or_asks_for_the_grade` now edits the diabetes disclosure later (its example checked, another chosen with its reason) and removes COPD after adding Smoker (the grade becomes II, "Current smoker").
- After the fixes: `make lint` clean; `make test`: clinical 372, web 270, CDK 124 passed, Go ok; e2e on a private stack (stopped afterwards): J10 (all), J5, J1, J4: all passed.

## Test report 2 (retest; all passed)

- Every case re-run passed, with report 1's F1, F2 and C1–C3 confirmed fixed. R-05's unsent set list was checked at 390 px in an iframe (the window couldn't be resized); a true 390×844 window stays a hand check for Rahul.
- Exploring note, not changed: a disclosure saved before the F1 fix keeps its stale reason ("Smoker · ASA III (COPD)", the tester's draft). A new one can't get there; dropping unoffered reasons on open could also drop valid ones after a guide change. In pending.md.
- No code change after report 2. Finished: the session row says **Tested**; plan C6's status and the plans table say it awaits verification; decisions.md and pending.md are current.

## Verification 1 (needs fixes), fixed test-first

- **F1: after "No ASA grade", adding a code brought the declined grade back.** `afterCodeAdded(asa, guide, added, declined)`: once declined, adding a code fills in nothing and asks nothing; its examples join the box and **No ASA grade stands** (builder's choice of the verifier's two: the author already answered, and can still pick a grade from the box). `OutputEditor` passes its "No ASA grade" state; picking a grade from the list now also clears that state (the radio no longer shows No ASA grade beside a grade).
- **F2: removing a code dropped the Emergency (E) tick.** `afterCodeRemoved` keeps `asa.emergency` on the grade that replaces it.
- **Item 3:** `givenEarlier` (the preview's and test cases' earlier answers) starts `{suggestedAsa}` at 0, as `hqSets` does, so "is at most II" is decided as for an episode with no grade. The rest (no grade can be worked out from answers given by hand) is in pending.md.
- **Item 5:** the episode page works out the suggested ASA once per load (`useMemo`), and only once the patient has sent; `logic.ts` uses `ASA_GRADES`. Item 4 and the rest of 5 (the missing audience, the half-second removal race, the slow path, the line's wording) are in pending.md.
- **The audit journey (`test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`) isn't S39's:** S39 changes nothing on the audit page or its route, and with S39's code on a private stack (:3093/:8093, the same shared `sj` database) the whole audit file passed (with J5 and the J10 ASA test: **12 passed**). So the verifier's failure against `make dev` is that environment or a hover flake. The run took 20 minutes: the audit journeys are slow on this long-lived database.
- Tests: `asa.test.ts` "after the author chose No ASA grade…", "removing a code keeps the Emergency (E) tick…", "answers given by hand… count as no grade" (clinical **375** passed); J10 adds COPD, No ASA grade, then Cough: the grade stays empty with No ASA grade checked.
- After the fixes: `make lint` clean; `make test`: clinical 375, web 270, CDK 124 passed, Go ok; e2e (private stack, stopped): J10's ASA test, J5 (all), test_audit (all): 12 passed.

## Test report 3 (retest after verification 1; all passed)

- Every check passed; verification 1's F1 and F2 confirmed fixed.
- Two exploring notes, judged intended by the tester and left as built, both in pending.md for Rahul: a saved "No ASA grade" isn't remembered when the disclosure is reopened (nothing wrong is saved); "(emergency)" marks only a reason below the top grade.
- No code change after report 3. Finished: the session row says **Tested** (test report 3); plan C6's status and the plans table say it awaits verification 2; decisions.md and pending.md are current.

