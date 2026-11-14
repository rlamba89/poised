# S49 build notes: MUST, Morse falls, VTE and the Clinical Frailty Scale

[Brief](brief.md) · [Answers](answers.md) · [Test plan](test-plan.md). Plan C7, part 4 of 4. Built 7 Nov 2026.

## Answers (Rahul, 7 Nov)

([answers.md](answers.md))
1. **Falls: build one now** (not the brief's recommendation). The builder proposed the **Morse Fall Scale**.
2. **Build MUST, VTE and the Clinical Frailty Scale now; the licences are go-live items** (as S48).
3. **MUST's weight loss is asked in the HQ;** today's weight and BMI come from the observations.

## Each instrument's terms and source (checked before any code)

| | Source used | Licence |
| --- | --- | --- |
| **MUST** | BAPEN's MUST flowchart (`must_page3.pdf`, read): Step 1 BMI >20 (>30 obese) = 0, 18.5–20 = 1, <18.5 = 2; Step 2 unplanned weight loss in past 3–6 months % <5 = 0, 5–10 = 1, >10 = 2; Step 3 "If patient is acutely ill and there has been or is likely to be no nutritional intake for >5 days" Score 2; Step 4 0 Low, 1 Medium, 2 or more High Risk. The explanatory booklet's worked examples couldn't be read (the PDF is images), so the tests check every boundary instead. | © BAPEN. Free for non-commercial use; **commercial use needs permission, a signed licence and a fee** (the form lists £20,000 + VAT per country for an EPR that trusts pay for). Cut-offs and scores unchanged; called a *screening* tool; BAPEN's attribution line (in `MUST.source`). |
| **Morse Fall Scale** | Morse JM et al., Can J Aging 1989;8(4):366–77, as the Fall TIPS trial shows it (Dykes PC et al., JAMA 2010;304(17):1912–8, Figure 1, read): falls 25, secondary diagnosis 15, aid 0/15/30, IV or heparin lock 20, gait 0/10/20, mental status 0/15. The figure's worked example (a fall, a walker, weak, forgets limitations) is 65, tested. | © Janice Morse. "Free of charge but permission from the author is required" (Fall TIPS FAQ). |
| **VTE** | Department of Health, *Risk assessment for venous thromboembolism (VTE)*, 2010 (read in full): the thrombosis and bleeding risk factors, each worded as the form has them; "Any tick for thrombosis risk should prompt thromboprophylaxis according to NICE guidance"; "Any tick [bleeding] should prompt clinical staff to consider if bleeding risk is sufficient to preclude pharmacological intervention." | © Crown copyright 2010: reusable free under the Open Government Licence (attribution). |
| **Clinical Frailty Scale** | Dalhousie's CFS page and guidance (2020, read): version 2.0's nine levels; 2020 renamed 2 "Fit", 4 "Living with Very Mild Frailty", 5–8 "Living with…" mild, moderate, severe, very severe frailty; 1 Very Fit, 3 Managing Well, 9 Terminally Ill from the guidance. | © Dalhousie University. Permission through its portal; "an EMR vendor who wishes to add a tool to their menu of offerings may require a license" (fees by duration and scope). |

AHRQ's, LOINC's and NICE's pages refused automated reads (403); the copies above were read instead.

## What was built

### TypeScript (for Rahul to review)

- **`packages/clinical/src/ready.ts`** (one definition per instrument, as S48):
  - **MUST:** `mustGroup()` is the HQ part: "Have you lost weight without trying in the last 3 to 6 months?" and, shown only after Yes, "About how much…(kg)?" (0–100). `mustClinicianPart()` is the assessment: BMI (5–150), weight now (1–400), weight lost (0–100), the acute disease effect (Yes scores 2, worded as BAPEN requires), the weight lost as a % of the weight before (lost × 100 / (now + lost), 1 decimal place, shown "9.1% of body weight lost") and the score, which uses that rounded %.
  - **Morse:** `morseGroup()` (the HQ part: a fall in 3 months, what you use to walk) and `morseClinicianPart()` (secondary diagnosis, IV, gait, mental status, the score 0–125; bands 0–24 Low, 25–44 Moderate, 45+ High).
  - **VTE:** the form's 30 risk factors in its order. `vteGroup()` is the patient's 9 (our wording, for a patient); `vteClinicianPart()` the BMI box, the clinician's factors (the form's wording) and two results: the **thrombosis count** (each tick, plus age over 60 and a BMI over 30, as the form's own ticks; 0 "No thrombosis risk factors ticked", 1+ "Thrombosis risk: consider thromboprophylaxis") and the **bleeding count** ("Bleeding risk factors: 1").
  - **Clinical Frailty Scale:** `cfsElements()`: one choice of 9 levels ("5: Living with Mild Frailty"), the score is the level, banded by its name. **Not** in `READY_SCORES`, so it can't be inserted in an HQ.
  - `READY_SCORES` gains MUST, Morse and VTE (the patient's parts only, each with a `note`, as Apfel).
- **`packages/clinical/src/assessments.ts`:**
  - An item's kind **`yesNo` is now `choice`** (Yes/No, or one of several: Morse's aid, gait, mental status; the CFS level).
  - A number item says what fills it: **`measure: "bmi" | "weight"`** (the hospital's or the patient's own, by the trust's precedence, as STOP-Bang's BMI was), or **`fromHq`** (MUST's weight lost: a No is 0, a Yes is the amount). **`noHq`** keeps the CFS level from ever being pre-filled.
  - `PrefillFrom` gains **`hospitalWeight`** (the latest weight that counts). The HQ part also gives the patient's own weight (`reportedBody` in `poa.ts`, beside `reportedBmi`).
  - **`hints`**: the CFS's are DASI and PRISMA-7 (`assessmentHints`).
  - VTE's items are lettered **T1–T19** and **B1–B11**; the others 1…; each has a short label for Go's messages ("Weight lost", "Acute disease effect", "Age").
- **Web (`features/episodes`):**
  - `assessments.ts`: `prefillFrom` passes the hospital weight; `sourceText` says "Hospital weight" / "Patient reported weight" / "HQ"; `sinceConfirmed` names the BMI or the weight; the number box takes 0 where the item allows it; **`cardHints`** gives the CFS card its "DASI: … (confirmed …)" lines from the confirmed rows only.
  - `AssessmentsPanel.tsx`: a choice of more than two options is a menu (Select); the number box's placeholder is "BMI" or "kg"; the hints show in a blue **For reference** box; an item's wording isn't repeated when it's the same as its label.

### Go

- **`ListAssessmentForms`** reads the version's chapter name (`c.name`), not the questionnaire's (S48 verification 1, Rahul 7 Nov): a form keeps its own version's name.
- `internal/assessment`: the four new JSON files and `list.json`, written by `npm run assessments`. **No Go logic changed:** `Check`, `Unanswered` and `ProfileProblem` handle them as they are (the observation source on a number item, the HQ's 0 kg, VTE's age).
- **Seed:** **Demo risk HQ** (Trust A, made up, ids 8xx): one set holding the library's height and weight, and MUST's, Morse's and VTE's patient parts; added to the seed's keys test. `e2e/data/risk-hq.json` is the same set. Both were written from `ready.ts` by a one-off script, not kept.

### E2E

- **J12:** `test_must_takes_the_hospital_bmi_and_follows_a_corrected_weight`: the patient sends the risk HQ (6 kg lost); Cara records 170 cm and 60 kg; MUST opens 3 of 4 (Hospital BMI 20.8, Hospital weight 60, HQ 6); "4: No" gives "1 (Medium risk), 9.1% of body weight lost"; the weight corrected to 55 changes the card to 19 and 55 and "2 (High risk), 9.8%…" before Confirm; the POA Summary shows it beside the hospital BMI.

## Decided by the builder

1. **MUST's weight lost is a percentage of the weight before** (today's + what was lost), rounded to 1 decimal place, and scored as shown (so 4.96% shows and scores as 5.0%).
2. **MUST in the assessment holds the weight lost in kg, not the HQ's Yes/No;** the Yes/No stays in the HQ only. A No pre-fills 0 kg.
3. **MUST's BMI and weight follow the trust's precedence** (the hospital's, else the patient's own), as STOP-Bang's BMI.
4. **Morse's bands are 0–24 Low, 25–44 Moderate, 45+ High,** as commonly used (Morse leaves the cut-offs to each hospital; the Fall TIPS trial's high-risk sign was at 45). The patient's two questions and the clinician's labels for options ("Knows their own ability", "Overestimates or forgets their limits") are our wording.
5. **The VTE form's mobility step is left out:** in pre-op every patient is a surgical patient, for whom the form says "assess for thrombosis and bleeding risk". Its result is the counts, not a score the form defines; the band names are ours, from the form's instructions.
6. **VTE's patient questions are our wording** (e.g. "Do you take medicine to thin your blood…?"), pre-filling the form's items; the clinician sees both the label and the patient's question.
7. **The CFS shows the level names only,** not the published one-line descriptions (copyright; the clinician knows the scale). Its hints are DASI's and PRISMA-7's results **as confirmed**, with the date; a DASI saved since (so not confirmed) gives no hint.
8. **Names:** "MUST", "Morse Fall Scale", "VTE risk", "Clinical Frailty Scale"; the CFS's id is `rockwood`.

## Differences from the brief

- **Falls risk is built** (Morse), by Rahul's answer 1; the brief recommended leaving it out.
- None otherwise. The brief's "acute-disease effect left to the clinician", "the procedure's and the clinician's items left to the clinician" and "Rockwood clinician-only, with hints" are built as worded.

## Test results (7 Nov)

- `make lint`: pass.
- `make test`: pass. clinical 616 (was 562), web 386 (was 378), evaluate 25, CDK 132, every Go package.
- `make test-integration`: pass. New: each form keeps its own version's name; all eight on one episode; Confirm of MUST (observation sources, the HQ's 0 kg), VTE (400 "Age has no answer" first) and the CFS (an age refused; level 5 confirmed); the list of eight.
- **Shown red first:** the name test (the old episode showed the new name), the ready-made tests for all four, the pre-fill tests, Go's `TestAllFromTheList`, the web tests (`cardHints`, the hospital weight, the 0 kg box).
- `make e2e`: **117 passed, 1 failed, 5 skipped** (the Cognito journeys, AWS only) in 11 min 25 s, on a second local stack (API :8082, evaluate :8092, web :3002), as Rahul's `make dev` predates the change. The failure was S24's `test_library_and_same_as`: on reopening "Same as…" it expected the "snoring" question in the unfiltered list, but the list shows the first 50 (compatible first, then by title) and the Demo risk HQ's questions ("About how much…", "Are you…", "Do you have…") now fill it ahead of "Do you snore…". The product is right; the check now looks for more than the one "snore" match. **That journey passed on its own afterwards** (2 passed). **Restart `make dev` and run `make migrate seed` before the manual round.**

## Test report 1 (8 Nov): 20 of 21, one cosmetic failure

- **F-1 (fixed):** the POA Summary's VTE value started about 64 px left of the other values: its label column shrank when the value was long. The label keeps its width now (`flexShrink: 0` in `PoaSummary`'s `Detail`); the value wraps in its own column. New J12 case `test_a_long_result_lines_up_on_the_poa_summary` (red first: 869 vs 933 px).
- **Exploring 2 (fixed):** the Frailty Scale's closed menu cut off "4: Living with Very Mild F…". A menu's cell is 270 px wide now (200 for buttons and boxes). The same J12 case checks level 8 shows whole (red first).
- **Exploring 1 (for Rahul, not changed):** on a desktop, scrolling the mouse wheel over a focused number box changes the patient's answer (6 became 5). It's the browser's behaviour for every number question in every HQ, older than S49, so it's in pending.md, not fixed here.
- **Exploring 3–6 (expected, not changed):**
  - a comma is a decimal point in an assessment's number box ("5,5" is 5.5), on purpose since S47;
  - "Demo risk HQ" is the seed's name, which the patient sees; real trusts name their HQs (the test plan no longer counts the HQ's name);
  - typing back the value shown keeps its source (HQ), as S47 decided ("a change back to the value shown keeps where it came from"); the test plan now says so;
  - the Settings switch shown off under "Use Preopsis's default" is older than S49.
- Tests after the fixes: web 386, J12 and J5 (23 passed).

## Test report 2 (8 Nov): all passed

15 of 15 (T-01–T-04, T-07, T-08, T-10, T-12, T-13, R-01–R-05). No code changed after it.
- **Hand check for Rahul:** the POA Summary's **print** layout with the long VTE value (Print preview); the F-1 fix is a flex rule, and the tester didn't print.
- Narrow widths (390 and 768 px) have no sideways scroll; the VTE value wraps onto several lines in its column. Staff pages aren't phone-first.

## Verification 1 (8 Nov): needs fixes; fixed 8 Nov

([verify.md](verify.md); Rahul's two answers there.)
1. **MUST's BMI and weight from one source (Rahul's answer 1).** A card with both (MUST) now takes them as a pair: the hospital's whenever it has a weight (by the precedence), so with a weight but no height the BMI is left **Missing** for the clinician, never the patient's own BMI beside the hospital weight. With precedence off, the patient's pair; without one, the hospital's. A card with a BMI only (STOP-Bang, VTE) is as before. `sinceConfirmed` says "There's no BMI now (36.4 when confirmed)." Tests: clinical "the BMI and the weight come from one source" (red first: on, and the patient's weight without a height), web "a hospital weight without a height".
2. **VTE: a patient recorded as male isn't asked about pregnancy, an oestrogen contraceptive or HRT in the HQ (Rahul's answer 2).** Those three carry `visibleIf: {patientSex} <> 'male'` in `vteGroup`; female, Other and Not known see them. The assessment's own copy has **no** such conditions (`withoutConditions`), so the clinician answers all 30 and a man's ticked HRT counts; a question the HQ hid pre-fills nothing (Missing). Because of that, **the VTE assessment's JSON is unchanged: no new platform version is published.** Tests: `ready.test.ts` (shown per sex, red first), clinical "VTE's questions a man isn't asked", J12 `test_a_man_is_not_asked_about_pregnancy_contraception_or_hrt` (phone). The Demo risk HQ and `e2e/data/risk-hq.json` were regenerated (keys test passes). **Rahul's local database:** the seed doesn't overwrite a published HQ, so the seeded Demo risk HQ's one set (`…0811`) was updated in place with the new content (local only; the database is disposable, decisions §1).
3. **Finding 3 (folded in):** `reportedBody` gives the patient's weight alone when they gave no height, so MUST's weight fills from it (BMI Missing).
4. **Finding 4 (left, test corrected):** Go still accepts any of hq / observation / clinician on a non-profile item (only an observation on a choice is refused). It's provenance from an authorised clinician; the score is unaffected. The integration case now sends the CFS level as clinician's, so it's refused only for the age it names. Noted in pending.md.
5. **Finding 5:** VTE's short labels live beside its keys in `ready.ts` (`VTE_ITEMS`); `assessments.ts` no longer repeats them.
6. **Finding 6:** the patient's own BMI and weight are worked out once per HQ (`reportedPart`), and only for a card with a `measure`.
7. **Finding 7:** `bmiInput` is `numberInput`; the load message says "the hospital's readings can't be checked".
- Tests after the fixes: `make lint`, `make test` (clinical 625, web 388, evaluate 25, CDK, Go), `make test-integration` pass; J12 19 passed; the full `make e2e` **120 passed, 5 skipped** (Cognito, AWS only) in 11 min 54 s, on the second stack.

## Test report 3 (8 and 13 Nov): all passed

12 of 12 (T-17, T-18, five happy paths, R-01–R-05). No code changed after it. The round paused on 8 Nov when the Mac ran out of memory (not the app) and finished on 13 Nov.

## Verification 2 (13 Nov): needs fixes; fixed 13 Nov

([verify.md](verify.md); Rahul's two answers asked by the builder, in decisions.md.)
1. **VTE looked complete with a bleeding item unanswered** (finding 1). `assessmentResult` is now Incomplete while any of the content's results is empty (VTE's bleeding count has inputs its first result hasn't), and the card's `complete` also needs no item missing, so Confirm stays off. Tests: clinical "VTE is Incomplete while any item is missing" (red first: the bleeding case), J12's VTE case checks Incomplete and Confirm off with one bleeding item left (red against the old code, as the verifier proved).
2. **Always asked fresh (Rahul: yes).** `alwaysFresh` on MUST's two questions, Morse's fall and VTE's pregnancy in `ready.ts`; the assessments' copies drop it (`withoutConditions`, now used for Morse too), so **their JSON is unchanged: checked equal to the versions published in the local database, no new platform version**. The Demo risk HQ and `e2e/data/risk-hq.json` were regenerated (keys test passes), and Rahul's local seeded set (`…0811`) updated in place again. Test: `ready.test.ts` "the ready-made parts' recent-time questions" (red first). Go's pre-fill already skips `alwaysFresh` questions (S25's tests).
3. **Precedence off prefers the hospital's full pair over a patient's weight alone (Rahul: yes).** `pairSource`. Test: clinical "off: a patient's weight without a height gives way…" (red first).
4. **The "Same as…" reopen check** now asserts the list starts with a compatible question, not "snoring" (the one the search kept).
5. **Cleanups:** `VTE_ITEMS` places the age and BMI by key (after active cancer and known thrombophilias), not by index; the stale `vteForAssessment` comment fixed; `CFS_SOURCE` (unused) became a comment; the number box's placeholder is the unit its wording ends with (`boxPlaceholder`, tested), not "kg" for everything; the Frailty Scale card no longer walks the HQ (it asks for no keys). The BMI question defined three times is left (each instrument's own wording and limits).
- Tests after the fixes: `make lint`, `make test` (clinical 630, web 389, evaluate 25, CDK, Go), `make test-integration` pass; J12 and the question-library journeys 21 passed on the second stack. Full `make e2e`: 120 passed, 5 skipped in 741.11s (0:12:21) (second stack).

## Test report 4 (13 Nov): all passed

13 of 13 (T-19, T-20, T-21, five happy paths, R-01–R-05). No code changed after it. Its exploring note 3, "0 pts" beside MUST's Yes and No in the editor, is expected: the question carries points of 0 (it scores nothing in an HQ), and the editor shows every option's points, as for Apfel's smoking question; the patient never sees them.

## Verification 3 (13 Nov): needs fixes; fixed 13 Nov

([verify.md](verify.md); Rahul's answer there and in decisions.md.)
1. **A weight typed on the MUST card kept the pre-filled BMI (Rahul: the typed one clears the other).** `unpaired` in `assessments.ts`: on a card with both a BMI and a weight, a clinician's weight leaves a pre-filled BMI Missing and a clinician's BMI a pre-filled weight; both typed are kept; applied to the pre-fill (a saved, unconfirmed row too) and to `withDraft`; a confirmed card is as confirmed; STOP-Bang and VTE unchanged. Tests: six clinical cases (red first: the draft, the reverse and the saved row), J12's MUST journey types a weight (BMI Missing, Incomplete, Confirm off), then a BMI, then Undo.
2. **The audit journey failed on this database's size** (not S49's code): it raises `csv.field_size_limit`. The question of storing every listed id is in pending.md for Rahul.
3. **MUST's rounding edges** (10.04% and 4.96%) are in pending.md's MUST line for a clinician.
4. **"There's no BMI now"** now reads "The hospital has a weight but no height now (BMI 25.7 when confirmed)." when the BMI is left Missing beside a hospital weight (web test, red first; the verification 1 case's line changed to match).
5. **The HQ files holding ready-made parts are checked against `ready.ts`** (`ready.test.ts`: the Demo risk and fitness HQs, seeded and e2e), so a wording change without regenerating them fails `make test`.
6. Cleanups in finding 6 left (optional): `yes` and `scores` (tests use them), the placeholder's unit, the repeats.
- Tests after the fixes: `make lint`, `make test` (clinical 640, web 390, evaluate 25, CDK, Go), `make test-integration` pass; J12 and the audit journeys 25 passed on the second stack. Full `make e2e`: 120 passed, 5 skipped in 792.98s (0:13:12) (second stack).

## Test report 5 (13 Nov): all passed

12 of 12 (T-22, R-01–R-05, six happy paths). No code changed after it. T-22's example result in the test plan was wrong ("4 (High risk)"; with 2 kg lost it's "2 (High risk), 3.8%…"), corrected. Its exploring notes: (1) a MUST card confirmed on typed values shows no "hospital BMI is now…" note when a reading changes, as intended (neither value came from the readings); (2) an incomplete card can be saved, as every card since S47 (Save keeps a draft, only Confirm needs every item); (3) the New episode HQ menu isn't searchable, older than S49 and already in pending.md.

## Verification 4 (13 Nov): needs fixes; fixed 13 Nov

([verify.md](verify.md).)
1. **The hospital's own number typed back after the other was typed was cleared again.** `withDraft` in `assessments.ts`: on a card with a BMI and a weight, once the other is the clinician's (typed now, or saved as theirs), a number in this box is the clinician's too, even when it equals the pre-filled one. Typed back with the other untouched, it keeps its source (as T-03). Tests (red first): both directions, verification 1's case (hospital weight, no height: BMI typed, then the weight), and the untouched case; J12's MUST journey types BMI 30, then the hospital's weight 55 back (Entered by clinician, a score, Confirm on).
2. **A confirmed card with a typed weight beside a pre-filled BMI showed BMI Missing on the card.** `withDraft` now runs `unpaired` only when a BMI or weight changed on the screen. The unit case goes through `withDraft` with an empty draft and with a choice changed (red first). It can't be made in the browser (no such row exists; it needs a row confirmed before verification 3 or sent straight to the API), so T-23 doesn't cover it.
3. **`assessmentResult` read the extra results after `m.dispose()`.** It copies each value and text before the dispose. No new test (today's survey-core keeps them after a dispose, so a test can't fail first); the existing DASI and VTE result cases cover it.
4. Cleanups in finding 4 left (optional), as after verification 3.
- Tests after the fixes: `make lint`, `make test` (clinical 644, web 390, evaluate 25, CDK 132, Go), `make test-integration` pass; J12 19 passed on the second stack. Full `make e2e`: 120 passed, 5 skipped in 851.36s (0:14:11) (second stack).
- **Builder's own `/code-review high` after the fixes (Rahul asked, 13 Nov):** 8 points, none in this round's changes. Fixed: CLAUDE.md's Scores entry and `ReadyScoreWindow.tsx`'s comment now name MUST, Morse and VTE. Checked and not a bug: Morse's two long mental-status answers as buttons fit their column at 1280 and 900 px (the table widens it; screenshots taken). Already recorded: MUST's rounding edges and Go's permissive sources (pending.md), the optional cleanups (verification 3 and 4) and the weaker Same as… check (verification 1).

## Test report 6 (13 Nov): all passed

12 of 12 (T-23, R-01–R-05, six happy paths). No code changed after it. Its exploring notes: verification 4's own example works (1); a confirmed card with both numbers typed keeps its state (2); the Gait menu's open list starts right of its box (3, cosmetic, in pending.md); the New episode HQ menu isn't searchable (4, older, already in pending.md).

## Verification 5 (13 Nov): needs fixes; fixed 13 Nov

([verify.md](verify.md); Rahul's answer in decisions.md.)
1. **A box typed in and put back still counted as typed** (BMI 36, then 35.9 back, then a weight: the old BMI stayed "Entered by clinician"). `draftAfter` in `packages/clinical`'s `assessments.ts` gives the screen's changes after one more: a value that leaves the item exactly as it showed before (same value and source) drops its key. The panel's `onChange` uses it. Tests (red first): BMI typed and back then a weight (BMI Missing), the mirror, verification 4's cases unchanged, an emptied box; J12's MUST journey types 36 then 19 back, then a weight (BMI Missing). The journey step was checked to fail with the old `onChange`.
2. **A box the rule emptied asked for a value once touched.** `had` is now `shownWithout(...)`: the item as shown with every change but its own, so a box the pairing rule emptied showed nothing (not `drafted`, which would make a box the clinician typed into "had a value" and block its emptying). Unit case; J12 types 1 and deletes it in the emptied BMI box (no "Enter a BMI", Save on).
3. **The weight-lost line:** "The HQ now gives 0 kg for Unplanned weight loss in the past 3–6 months (6 kg when confirmed)." (web test changed first).
- **Rahul's answer (asked by the builder):** a BMI typed where the pre-fill has none keeps the hospital's weight (`unpaired`'s `filledGap`: in `withDraft` the item had no source before the screen's changes; in `assessmentItems` the pre-fill now gives none, so a height added later makes the weight the clinician's to retype). Unit cases (typed, saved and filled afresh, a weight then typed over it); a new J12 journey (weight 60 only, BMI 20.8 typed: Hospital weight 60 kept). T-23 step 4 changed to match.
4. Cleanups in finding 4 left (optional), as after verifications 3 and 4.
- Tests after the fixes: `make lint`, `make test` (clinical 649, web 390, evaluate 25, CDK 132, Go), `make test-integration` pass; J12's MUST journeys pass on the second stack. Full `make e2e`: 121 passed, 5 skipped in 763.16s (0:12:43) (second stack; one more journey).

## Test report 7 (13 Nov): 11 of 12; F-1 fixed

([test-report-7.md](test-report-7.md).)
- **F-1 (Minor): a MUST box the pairing rule marked Missing still showed the number typed in it earlier** (36 then 20.8 back in the BMI, then a weight: BMI Missing but its box showed 20.8; the mirror showed 60). What was stored was right. The number box keeps what the clinician typed while they type; `boxTyped` (web `assessments.ts`) drops it once the item no longer holds that value (a valid number typed, the item now missing: only the pairing rule does that), so the box goes empty with the item. A mistake or an emptied box is kept as before. TypeScript for Rahul: `AssessmentsPanel.tsx`'s `ItemValue` now reads `const typed = boxTyped(typing, i)` instead of using its own typed text straight. Tests (red first): four web unit cases; J12's MUST journey checks the BMI box is empty after the weight, and a new mirror step (weight 61 then 55 back, BMI 17.3: the weight's box empty). The journey step was checked to fail without the fix ("unexpected value 19").
- Its exploring notes: (1) T-24 step 5 needs a super clinician to re-open an auto-triaged HQ review, the existing rule (the test plan's step says to validate; a tester can sign in as Sue Super); (2) saving a half-typed card after the HQ changed takes the HQ's new value, as intended; (3) the Gait menu's list, (4) the New episode HQ menu: both already in pending.md.
- Tests after the fix: `make lint`, `make test` (clinical 649, web 394, evaluate 25, CDK 132, Go) pass; J12 20 passed and J5 5 passed on the second stack. Not re-run: `make test-integration` and the full `make e2e` (only the web number box changed; the journeys using assessment cards are J12 and J5).

## Test report 8 (14 Nov): all passed

14 of 14 (T-24 with its changed steps 2 and 3, R-01–R-05, eight happy paths and earlier cases); F-1 confirmed fixed. No code changed after it. Its exploring notes: (1) both boxes typed keep both, as intended; (2) a saved weight can't be emptied back to the hospital's (the S47 rule; in pending.md for a clinician); (3) a confirmed card hides its "since Confirm" line while it has unsaved changes (S47 answers 6's `!changed`; in pending.md for a clinician); (4) T-24 step 5 needs a super clinician on an auto-triaged episode (the test plan says so since report 7); (5) the Gait menu, already in pending.md.

## Verification 6 (14 Nov): needs fixes; fixed 14 Nov

([verify.md](verify.md).)
1. **A box typed in earlier kept its key once the other box was put back** (weight 61, BMI 20.8 typed into the emptied box, weight 60 back, then weight 50: BMI 20.8 stayed "Entered by clinician" beside 50, "0 (Low risk)", Confirm on). `draftAfter` in `packages/clinical`'s `assessments.ts` now drops **every** key whose removal changes nothing on the card (each item the same value and source without it), repeating until none drops; before, it looked only at the box just typed in. TypeScript for Rahul: a loop over the draft's keys that calls `withDraft` with and without each key and compares the two lists. Tests (red first): both four-step sequences (the BMI Missing beside 50; the mirror, the weight Missing beside BMI 25); verification 4's and 5's cases pass unchanged. J12's MUST journey has the first sequence (weight 56, BMI 19, weight 55 back, weight 50: BMI Missing, its box empty, Incomplete, Confirm off); it was checked to fail with the old `draftAfter` ("unexpected value Entered by clinician").
   - **A side effect, intended:** beside a BMI the clinician saved, the hospital's own weight typed back is now the clinician's (verification 4's rule), where before it was dropped as unchanged. The value is the same; only the badge differs.
2. **`had` is worked out from the draft only for the BMI and weight rows** (`AssessmentsPanel.tsx`); every other row reads the card before changes, as no other item depends on another box.
3. Cleanups in finding 3 left (optional), as after verifications 3 to 5.
- Test plan: T-25 added.
- Tests after the fixes: `make lint`, `make test` (clinical 651, web 394, evaluate 25, CDK 132, Go), `make test-integration` pass. Full `make e2e`: 121 passed, 5 skipped in 827.65s (0:13:47) (second stack).

## Test report 9 (14 Nov): all passed

17 of 17 (T-25 on an unsaved and a confirmed card, T-24, T-23 step 2, R-01–R-05, and the patient's side, MUST, Morse, VTE with T-19, the POA Summary, T-14 and reload/back). No code changed after it. Its exploring notes: a longer keyboard chain behaves as T-25 (1); emptying a pre-filled box asks for a value, the S47 rule (2, in pending.md since report 8); a stale save in a second tab gets the 409 and Reload (3); a double-clicked Confirm sends one request (4); "20,8" counts as 20.8 but the box keeps showing what was typed, as intended (`numberInput` allows a comma since report 1; the box never rewrites a clinician's typing) (5); the Gait menu (6, in pending.md); auto-triage on send (7, as designed).

## Suggested commits (Rahul commits after verification)

1. **S49: an assessment form keeps its version's name.** `db/queries/assessments.sql`, `internal/db`, the integration case. Tests: `TestAddAssessment`.
2. **S49: MUST, Morse, VTE and the Clinical Frailty Scale.** `packages/clinical` (`ready.ts`, `assessments.ts`, `poa.ts`, their tests), `apps/api/internal/assessment/*.json` and its test, the web `assessments.ts`, `AssessmentsPanel.tsx`, `PoaSummary.tsx` (report 1, F-1) and tests, the integration cases. Tests: clinical, web, `go test ./internal/assessment`, `TestListAssessments`, `TestSaveAndConfirmAssessment`.
3. **S49: Demo risk HQ; J12.** `cmd/seed` (main, keys test, `demo-risk-hq.json`), `e2e/data/risk-hq.json`, J12, and the "Same as…" journey's reopen check (`test_question_library.py`). Tests: `go test ./cmd/seed`, J12, `test_question_library.py`.
4. **S49: docs.**
