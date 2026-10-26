# S39 verification

## Verification 2 (26 Oct 2026, automatic run)

**Verdict: Verified.**

Verification 1's two findings are fixed, with tests, and I reproduced both fixes in Chrome. Its items 3 and 5 (part) are fixed; the rest are in pending.md. The rest of the session still matches the brief and Rahul's answers (see verification 1). No new route; `GET /api/codes` keeps its five cases (re-checked by hand: no sign-in 401; Cara, Val and Bea all 200, as the code list is platform data). Tenancy is unchanged: Bea (Trust B) and Val (Hospital A2) get 404 for Hospital A1's episode HQ. `asa.reason` sits inside `clinicalOutputs`, which `chapter.ForPatient` strips. The A-20 grep finds nothing new.

### Findings (none blocking; the new ones are in pending.md under "From S39's verification 2")

1. **F1 fixed:** `afterCodeAdded(asa, guide, added, declined)` (`packages/clinical/src/asa.ts:32-39`); `OutputEditor.tsx` passes its "No ASA grade" state, and picking a grade clears it. In Chrome: Smoker, then **No ASA grade**, then add COPD: the grade stays empty and No ASA grade stays selected, with COPD's example added to the box.
2. **F2 fixed:** `afterCodeRemoved` keeps `asa.emergency` (`asa.ts:48-53`). In Chrome: COPD (III) with **Emergency (E)** ticked, then add Smoker, then remove COPD: the result is **II, E still ticked**, "ASA II: Current smoker (Adults)" selected. Saved as `{grade: III, reason: "COPD"}` after picking COPD again. The stored JSON was checked through the API.
3. **(Pending) E from any disclosure can change a draft's saved test-case line** ("IV" becomes "IVE" when an ASA II emergency is also given; `testcases.ts:41`). This was a recorded builder decision. It's clinically right, but a draft's old test case would then block Publish until it's re-saved.
4. **(Pending) "From the code library" leaves a code with several examples ungraded** (`OptionsEditor.tsx:180`). The window says to choose the grade in the option's disclosure, but nothing makes the author do it. Suggest a publish warning.
5. **(Pending) The episode page's line is gated on status, not on the patient having sent** (`EpisodePage.tsx:261`). So a cancelled episode whose HQ was never sent shows a suggestion from unsent answers. This is narrow.
6. Code review's other points are already in pending.md or not real here: the half-second removal race, the missing `audience`, `hqSets` building each survey twice, and the heading written out in several places. The radio-matching point can't happen with today's guide, because no grade and example pair appears in two groups.

### Spot checks in Chrome

- Editor (Alex Author, new draft "VF S39 2015 Verify"): F1 and F2 as above. The result row says "· ASA III"; the reason line follows the choice.
- Episode "Rory MT S39 R3 2000 Retest" (Cara): the episode page says "Suggested ASA IIE: Smoker (77176002) – Current smoker (the anaesthetist confirms)". The POA Summary's Validated tab says IIE with the nurse's and anaesthetist's boxes. The Patient answers tab says IIIE with the diabetes and "(emergency)" smoker reasons, each linking to Validate at `?q=q_kfwtxq`, and has no grade boxes.
- Permissions: `GET …/episodes/{id}/hq` gives Bea (Trust B) 404 and Val (Hospital A2) 404; Cara gets 200.

### Test results

- `make lint`: clean.
- `make test`: Go ok (`asaguide`, `repocheck`); Vitest web **270 passed**, clinical **375 passed**; CDK **124 passed**.
- `make test-integration`: every package ok.
- `make e2e` (against `make dev`, the shared database): **90 passed, 5 skipped** (Cognito-only), in 31 min. `test_audit` passed this time, so verification 1's failure was the environment or a flake.

### Hand checks for Rahul

- R-05 at a real 390×844 window (the patient pages show no ASA text; checked at desktop width and in a 390 px iframe only).

### Commits

1. `S39: the ASA guide in the code search`
2. `S39: the suggested ASA with its reasons, and in clinician set conditions`
3. `S39: J5 and J10 extended`
4. `S39: docs` (not `docs/market/`)

## Verification 1 (26 Oct 2026, automatic run)

**Verdict: Needs fixes** (two Minor editor bugs in the ASA guide's rule, both small; nothing else blocks).

The session does what the brief and Rahul's answers ask: the ASA guide in Go served with the code search, the editor's guide box, the whole-HQ suggested ASA with reasons on the POA Summary (both tabs) and the episode page, the preview's reasons, and the suggested ASA in clinician set conditions. The departure from the brief (no `codes.asa` column; a Go file instead, because one code can have several grades under Rahul's answer 1) is reasoned and recorded in decisions.md. No new route; `GET /api/codes` keeps its five cases plus a happy-path subtest for `asaChoices`. Tenancy is unchanged (the guide is platform data, as the codes are). `asa.reason` lives inside `clinicalOutputs`, which `chapter.ForPatient` strips, so nothing reaches the patient. The A-20 grep finds only comments that were there before.

### Findings, worst first

1. **F1 (Minor, reproduced in Chrome): adding a code overrides the author's "No ASA grade".** `apps/web/src/features/outputs/OutputEditor.tsx:57-65` (`codeAdded`) calls `afterCodeAdded(asa, guide, choices)` with `asa` undefined after the author chose **No ASA grade**, so `afterCodeAdded` (`packages/clinical/src/asa.ts:32-37`) treats the disclosure as never graded:
   - Steps (Alex Author, draft "MT S39 1750 ASA", Select Many → Statement 3 → + Add disclosure): add SNOMED 13645005 (fills III, COPD); choose **No ASA grade**; add SNOMED 49727002 (Cough, no examples).
   - Actual: the grade goes back to **III**, "ASA III: COPD (Adults)" is selected and the reason line says "COPD". Saving would store a grade the author declined. With a code that brings several examples, it instead re-opens the required choice.
   - Why it matters: Rahul's answer 1 is "no silent single grade"; the declined grade then feeds the suggested ASA, and through it clinician set conditions.
   - Fix: when the author has chosen No ASA grade, adding a code fills nothing in by itself. Only a code that brings new examples should ask again (or leave No ASA grade standing; builder's call, say which). Unit test in `asa.test.ts` (pass the "declined" state to `afterCodeAdded`, or keep it in OutputEditor) and a J10 step.
2. **F2 (Minor): removing a code drops the Emergency (E) tick.** `packages/clinical/src/asa.ts:47-48` (`afterCodeRemoved`) replaces the grade with `afterCodeAdded(undefined, [], left)`, which always has `emergency: false`. Example: COPD (III) with E ticked, plus Smoker; remove COPD → II without E, and the suggested ASA loses its E with no sign. Fix: keep `asa.emergency` on the grade that replaces it. Unit test.
3. **(Not blocking; fix or add to pending.md) Preview and test cases decide an earlier clinician set's `{suggestedAsa}` condition differently from a real episode.** `givenEarlier` (`packages/clinical/src/questionSets.ts:68-87`) leaves `suggestedAsa` undefined, while `hqSets` starts it at 0. An earlier clinician set with "Suggested ASA is at most II" is shown in an episode but counted hidden in the preview's earlier answers and a test case's `earlier`, so a later set's cross-set score can differ. Narrow (needs a cross-set score fed by an ASA-conditioned clinician set). At least seed 0 so "no grade" matches, and note the rest in pending.md.
4. **(Not blocking; pending.md) `chapterConditionProblems` treats a missing `audience` as a patient set** (`questionSets.ts:50`). Every caller passes it today; S41's evaluate service re-checks publishing (D-1) and must pass it too, or a valid clinician set is refused. Safe direction (fails closed), but worth a line in pending.md or making the parameter required.
5. **(Not blocking) Smaller items, for pending.md:**
   - `OutputEditor.tsx:41-52`: the existing codes' examples arrive by one code search per code; a code removed before its answer arrives keeps its grade (F1 of report 1 again, in a sub-second window).
   - `EpisodePage.tsx:263` runs `hqSuggestedAsa` on every render, even while it isn't shown, and it rebuilds each set's survey after `hqSets` did; on an HQ with an ASA condition `hqSets` already has the outputs. Memoise, or have `hqSets` return them.
   - An HQ with a `{suggestedAsa}` condition now takes `hqSets`' slow path, so every set gets `earlier` contexts (S38's "older HQs get none" no longer holds for them). Harmless as far as I can see.
   - The grade list is written out again in `logic.ts:254` (use `ASA_GRADES`), and `SuggestedAsaLine` repeats `suggestedAsaText`'s wording.

### Spot checks in Chrome (Alex Author)

- Editor, new disclosure: COPD fills III with "ASA III: COPD (Adults)" and its reason line. **F1 reproduced** (above); the disclosure was cancelled, nothing saved.
- POA Summary of "MT S39 1840 Emerg": Validated tab "Suggested ASA IIE" with "Smoker (77176002) – Current smoker" and "Needs surgery today (emergency)", then the nurse's and anaesthetist's boxes; Patient answers tab (`?tab=patient`) the same suggestion without the boxes.
- `GET /api/codes?set=SNOMED&q=13645005` sends `asaChoices: [{grade: III, example: COPD, group: Adults}]`.

### Test results

- `make lint`: clean.
- `make test`: Go ok (with `internal/asaguide` and `repocheck`); Vitest web **270 passed**, clinical **372 passed**; CDK **124 passed**.
- `make test-integration`: every package ok (`httpapi` with `TestSearchCodes`' new subtest).
- `make e2e` (against `make dev`, the shared long-lived database): **89 passed, 5 skipped (Cognito-only), 1 failed**: `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all` (the audit page's "N patients" tooltip didn't appear on hover). It fails again when re-run alone. S39 touches nothing on the audit page or its route; it passed in the builder's run on a private stack, so it looks like this database's state (or a hover flake), not S39. The builder should look at it during the fix round (run it on a private stack, or against HEAD) and note it in pending.md if it isn't S39. The new J5 and J10 tests passed.

### Hand checks for Rahul

- R-05 at a real 390×844 window (the tester's window couldn't be resized; checked in a 390 px iframe).

### Suggested commits (after the fixes and a retest)

1. `S39: the ASA guide in the code search` — `apps/api/internal/asaguide/`, `apps/api/internal/httpapi/{codes.go,codes_integration_test.go}`.
2. `S39: the suggested ASA with its reasons, and in clinician set conditions` — `packages/clinical/src/{asa,asa.test,types,outputs,outputs.test,respondent,doc,logic,questionSets,publish,index}.ts`; `apps/web/src/features/outputs/{CodePicker,OutputEditor,OutputSummary}.tsx`, `features/editor/{OptionsEditor,ConditionBuilder,LogicEditor,Canvas,QuestionnaireEditor,PublishModal}.tsx`, `features/episodes/{SuggestedAsa,PoaSummary,EpisodePage}.tsx`, `features/preview/PreviewPage.tsx`.
3. `S39: J5 and J10 extended` — `e2e/journeys/{test_j5_poa_summary,test_j10_authoring}.py`, `e2e/data/asa-hq.json`, `e2e/README.md`.
4. `S39: docs` — `docs/sessions/S39-suggested-asa/`, `CLAUDE.md`, `apps/web/AGENTS.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plans/{README,c6-scores-asa-triage}.md`, `docs/sessions/README.md`. (Not `docs/market/`.)

**For the builder:** "Verification found problems: read `docs/sessions/S39-suggested-asa/verify.md`, fix them test-first, then ask me for a retest."
