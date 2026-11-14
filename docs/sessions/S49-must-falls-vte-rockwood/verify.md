# S49 verification

## Verification 7 (14 Nov 2026)

**Verdict: Verified.** Verification 6's fix is built as asked, with tests: `draftAfter` drops every key that changes nothing on the card, and the panel works out `had` from the draft for the BMI and weight rows only. Test report 9 passed it (17 of 17), and I saw it hold in the browser. One minor finding remains (finding 1). It never under-scores and nothing wrong can be confirmed, so it doesn't hold the session back. It's in pending.md for Rahul to decide.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (28 files changed, 19 new).
- **Read:** the brief, the answers, the build notes (through report 9), verification 6 and test report 9.
- **Ran:**
  - `/code-review high`: 10 findings, weighed below.
  - `make lint test test-integration`.
  - `make e2e`.
  - A probe unit test for finding 1, deleted after.
  - Browser spot-checks against Rahul's `make dev`.

### Findings, worst first

1. **Minor: once a BMI typed into the gap is saved, changing it empties the hospital weight on screen, though a reload brings it back.** [assessments.ts:396](../../../packages/clinical/src/assessments.ts#L396) (`withDraft`'s `unpaired` call)
   - **What:** `assessmentItems` decides `filledGap` by whether the pre-fill had a value (`measurement(…).source === undefined`). `withDraft` decides it by whether the item had any source (`!items.find(…).source`). A saved BMI has source "clinician", so `withDraft` treats a change to it as a BMI typed over a pre-filled one, and the pairing rule clears the hospital weight.
   - **Probe** (precedence on, hospital weight 60, no height, BMI 20.8 typed and saved):
     - On reload the card shows BMI 20.8 (clinician) and weight 60 (Hospital weight), as Rahul's 13 Nov rule says.
     - Change the BMI to 21: **weight Missing.**
     - The same change on a card never saved keeps weight 60.
     - A confirmed card behaves the same way.
   - **Why it's minor:** the card turns Incomplete and Confirm goes off until the clinician types the weight again. It never under-scores, and nothing wrong can be confirmed. Leave the card unsaved, or save it without the weight, and the next reload shows the hospital weight again. It happens only where the hospital has a weight but no height. It does go against Rahul's 13 Nov rule ("the BMI the clinician types keeps the hospital's weight"), and the screen and a reload disagree.
   - **Fix (if wanted), test-first:** work out once in `assessmentItems` whether each measure was a pre-fill gap (a flag on the item), and have `unpaired`, `withDraft` and the panel read that flag. This is also the code review's altitude point: the pairing rule is now worked out in five places.
     - Unit cases: the probe's saved and confirmed cards; change the BMI and the hospital weight stays.
2. **Minor (code review): `sinceConfirmed`'s "The hospital has a weight but no height now" also shows when the hospital *has* a height but the BMI it gives is outside 5–150** (for example height 30 cm mis-entered). [assessments.ts (web):252](../../../apps/web/src/features/episodes/assessments.ts#L252) This needs an implausible height, and only the wording is wrong (the BMI is still Missing, as it should be). Noted only.
3. **Cleanup (optional, as verifications 3 to 6 listed):**
   - `complete` repeats `missing` ([AssessmentsPanel.tsx:116](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L116)).
   - The BMI question is defined three times.
   - `yes` and `scores` repeat the locked cut-offs.
   - `cardHints` repeats `poaLines`.
   - `draftAfter`'s loop runs on every card. Only MUST's pair needs it, and it's cheap at today's sizes.

**Weighed and not defects (from the code review):**
- Go accepts any source on an item: in pending.md since verification 1.
- Morse's two long mental-status buttons were checked at 1280 and 900 px (verification 4).
- The audit CSV's field limit: the "Listed episodes" row's size is in pending.md for Rahul (verification 3).
- The "Same as…" journey's changed check: the 50-candidate limit is a recorded gotcha (decisions.md, 7 Nov). The new check still proves the search was cleared: the first candidate is a compatible question the search had left out.

### What's sound

- **Verification 6's fix:**
  - `draftAfter` recomputes the card after each key it drops, so a drop never changes what's shown; it only shrinks the draft.
  - Both four-step sequences and verification 4's and 5's cases are unit-tested. J12 has the first sequence.
- **Browser spot-checks** (Cara, on Zia MT S49 1305 Pair's confirmed MUST: Hospital BMI 20.8, weight 60, height 170):
  1. **Verification 6's sequence.** Weight 61, then BMI 20.8 typed into the emptied box, gave both "Entered by clinician" and Confirm on. Weight 60 typed back gave Hospital BMI and Hospital weight again, shown as confirmed. Then weight 50: **BMI Missing, its box empty ("BMI"), Incomplete, Confirm off.** **Undo changes** put the confirmed card back.
  2. **Isolation.** As Cara (Hospital A1 only):
     - the episode's assessments through A1's path: 200;
     - through Hospital A2's path: 404;
     - through Trust B's path: 404;
     - Add (Clinical Frailty Scale) through A2's path: 404;
     - the patient API's `/api/p/episodes/{eid}/assessments`: 404.
  3. **The Clinical Frailty Scale card without DASI or PRISMA-7** on the episode shows no "For reference" box.
- **Unchanged since verification 6:** scope (all four instruments, the POA lines, the name fix), tests (no new routes; Go unchanged since verification 3), tenant isolation, security, docs (the plan's Status, both READMEs, `decisions.md`, `pending.md`). The A-20 grep shows nothing in S49's added lines or JSON. `repocheck` passes.

### Test results (14 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 651, web 394, evaluate 25, CDK 132, every Go package (repocheck included) |
| `make test-integration` | pass, every package (`internal/httpapi` 26 s). Run with `DB_UP=`: `docker` isn't on this shell's PATH, and Postgres was already up. |
| `make e2e` | **1 failed, 120 passed, 5 skipped** in 12 min 33 s (the 5 are the Cognito journeys, AWS only). J12 and the question-library journeys passed. The failure was J7's text-invite check: 2 captured messages where it expected 1. Re-run alone, J7 passed (6 of 6). J7 picks its mobile from 1,000 made-up numbers (`made_up_mobile`), and the local database holds thousands of captured messages, so one can repeat a number from an earlier run. This flake predates S49 and S49 doesn't touch J7. In pending.md. |

### Test data I changed

- Nothing saved. On Zia's MUST card I typed the four steps, then clicked **Undo changes**: the card is as confirmed.
- The `make e2e` run added its usual E2E episodes.
- Cara Clinician is left signed in on the Chrome tab.

### Suggested commits

As verification 1 listed below. Finding 1, if Rahul wants it fixed first, goes in commit 2 with its unit tests.

## Verification 6 (14 Nov 2026)

**Verdict: Needs fixes.** One more way round the MUST pairing rule (finding 1), seen in the browser. It under-scores and can be confirmed. Verification 5's three fixes and report 7's F-1 are built as asked, with tests, and test report 8 passed them (14 of 14).

- **Reviewed:** `31046e6` plus S49's uncommitted changes (28 files changed, 16 new).
- **Read:** the brief, build notes (through report 8), verification 5, and test reports 7 and 8.
- **Ran:** `/code-review high` (8 findings, weighed below), `make lint test test-integration`, a probe unit test (deleted after), and a browser check of finding 1 against Rahul's `make dev`.

### Findings, worst first

1. **Must fix: once the other box is put back, a box typed in earlier keeps its key in the draft, and the pairing rule stops working.** [assessments.ts:415-420](../../../packages/clinical/src/assessments.ts#L415-L420) (`draftAfter`)
   - **What:** `draftAfter` drops the key of the box being typed in when it goes back to what it showed. It never looks again at the *other* box's key. A box typed into after the pairing rule emptied it can show the hospital's number again, with its source back, once the first box is put back. Its key is still in the draft, though. The next number typed in the first box then counts as "the other is the clinician's" (verification 4's rule), so the stale value is kept as "Entered by clinician" and never cleared.
   - **Seen in the browser** (Xan MT S49 1125 Mirror's confirmed MUST: Hospital BMI 20.8, hospital weight 60, height 170; undone after):
     1. Weight 61 typed: the BMI empties.
     2. BMI 20.8 typed into the emptied box: both "Entered by clinician", as intended.
     3. Weight 60 typed back: both read Hospital BMI and Hospital weight again, as intended. The draft still holds the BMI's key.
     4. Weight 50 typed: **BMI 20.8 "Entered by clinician"** beside weight 50 "Entered by clinician", "4 of 4 filled", **"0 (Low risk), 3.8%…"**, and **Confirm is on**.
     - 50 kg at 170 cm is a BMI of 17.3, which scores 2 (High risk). This is verification 3's under-scoring again, and it can be confirmed. The label also says the clinician entered a BMI they didn't choose for this weight.
   - **The mirror** (BMI 36, weight 60 typed into the emptied box, BMI 20.8 back, then BMI 25) keeps the weight 60 as the clinician's beside BMI 25. Both sequences fail in a probe unit test against today's code.
   - **Fix, test-first:** after each change, `draftAfter` should drop *every* key whose item shows exactly as it would without that key (same value and source), not only the key just typed. Repeat until nothing changes; for two boxes one more pass is enough. Verification 4's case must stay unchanged: weight 61 then BMI 20.8 keeps both, because without the BMI's key the BMI would be Missing.
   - **Unit cases:** the two four-step sequences above, expecting BMI Missing beside weight 50 and weight Missing beside BMI 25; verification 4's and 5's cases unchanged.
   - **Journey:** one J12 step for the first sequence.
2. **Minor (from the code review): `had` calls `shownWithout`, a full `withDraft`, once per row on every render.** [AssessmentsPanel.tsx:174](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L174) VTE's 30 rows make that 30 passes per keystroke, each walking the content. That's fine today. Optionally, work out `had` only for the two measure rows (no other item is affected by another's change), or memoise it per render. Fold it into finding 1's fix if convenient.
3. **Cleanup (optional, as verifications 3 to 5 listed):** `yes` and `scores` repeat the locked cut-offs; `complete` repeats `missing`; `cardHints` repeats `poaLines`; the BMI question is defined three times.

**Weighed and not defects (from the code review):**
- On reload, a saved clinician BMI clears the hospital weight once a height has been added since. Rahul's 13 Nov answer, recorded in the build notes, says this.
- Go accepts any source on a number item: already in pending.md since verification 1.
- Morse's two long mental-status buttons were checked at 1280 and 900 px (verification 4's builder review).
- `reportedBody`'s weight with an impossible height, and the "Same as…" check: minor, already raised in verifications 1 and 2.

### What's sound

- **Verification 5's fixes:**
  - `draftAfter` for the box typed in (unit cases for both directions, verification 4's cases unchanged).
  - `shownWithout` for `had`.
  - The weight-lost sentence.
  - `filledGap` (Rahul's answer).
  - Report 7's `boxTyped`: the emptied box shows its hint. Report 8 confirmed it on two episodes and on a confirmed card.
- **Unchanged since verification 5:** scope (all four instruments, the POA lines, the name fix), tenant isolation (no new routes; `ListAssessmentForms` scoped as S48 left it), security, docs. The A-20 grep shows nothing in S49's added lines or its JSON.

### Test results (14 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 649, web 394, evaluate 25, CDK 132, every Go package (repocheck included) |
| `make test-integration` | pass, every package |
| `make e2e` | Not re-run. The builder's full run after verification 5 was 121 passed, 5 skipped; J12 and J5 passed after report 7's fix. Report 8 passed the next morning. |
| Browser | Finding 1 reproduced on Xan's confirmed MUST card, then undone ("Undo changes"; the card is as confirmed). |

### Suggested commits (after the fix is verified)

As verification 1 listed below; the fix for finding 1 goes in commit 2 with its unit tests, and its J12 step in commit 3.

## Verification 5 (13 Nov 2026)

**Verdict: Needs fixes.** One more way round the MUST pairing rule (finding 1), seen in the browser. Verification 4's three fixes are built as asked, with tests, and test report 6 passed them.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (28 files changed, 16 new).
- **Read:** the brief, answers, build notes, verifications 1 to 4, and test reports 5 and 6 (report 6: 12 of 12 passed).
- **Ran:** `/code-review high` (10 findings; it found finding 1), `make lint test test-integration`, `make e2e` against Rahul's `make dev` (started 19:07, after the fixes), and the browser spot-checks below.

### Findings, worst first

1. **Must fix: a BMI typed and typed back, then a weight typed, keeps the old BMI as "Entered by clinician".** [assessments.ts:392](../../../packages/clinical/src/assessments.ts#L392) (`withDraft`), [AssessmentsPanel.tsx:175](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L175) (the draft keeps the key)
   - **What:** the draft remembers every box that was typed in, but not the order. Verification 4's rule ("a number in this box is the clinician's once the other is theirs") therefore also fires when this box was typed in *first* and put back to the hospital's number.
   - **Seen in the browser** (Ravi's unsaved MUST: Hospital BMI 35.9, hospital weight 110; undone after):
     - BMI 36 typed, then 35.9 typed back: the row reads "Hospital BMI" again, as it should.
     - Weight 60 typed: the BMI stays **35.9, "Entered by clinician"**, beside weight 60, also "Entered by clinician". Nothing is Missing. With items 3 and 4 answered, Confirm would be on.
     - At 175 cm, 60 kg is a BMI of 19.6, which scores 1; 35.9 scores 0.
   - **Why it matters:** it's verification 3's under-scoring by a narrower route, and the label says the clinician entered a BMI they didn't choose for this weight. It needs a slip in the BMI box first (a mistype put right), so it's rare, but the card gives no sign.
   - **The mirror** (weight typed and typed back, then a BMI typed) keeps the hospital's weight as the clinician's in the same way.
   - **Fix, test-first:** when a number typed in a box leaves that item exactly as shown before (same value and source), drop its key from the draft, so the box counts as untouched. A small helper in `packages/clinical` (the draft after one change) keeps it unit-testable. Cases: BMI typed and typed back, then a weight (BMI Missing); the mirror; verification 4's cases unchanged (weight first, then the hospital's BMI typed back: both the clinician's). One J12 step.
2. **Minor: a box the pairing rule emptied says "Enter a BMI" and blocks Save once touched.** [AssessmentsPanel.tsx:174](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L174)
   - **What:** `had` is read from the card before any change, so a BMI box that the rule emptied still counts as one that "showed a value". Typing a digit in it and deleting it turns Save and Confirm off until Undo, which also discards the typed weight.
   - **From the code, not seen in the browser.** Before that keystroke the same state (weight typed, BMI Missing) can be saved, as report 5 did.
   - **Fix:** take `had` from the item as shown now (`drafted`), not `base`. Fold it into finding 1's fix.
3. **Minor: the yellow "changed since Confirm" line reads badly for MUST's weight lost.** [assessments.ts:249](../../../apps/web/src/features/episodes/assessments.ts#L249) After the HQ's answer is corrected from No to Yes, 3 kg, a confirmed MUST says "The HQ now says 3 to Unplanned weight loss in the past 3–6 months (kg) (0 when confirmed)." Better: "The HQ now gives 3 kg lost (0 when confirmed)." A number item from the HQ needs its own sentence.
4. **Cleanup (optional, as verifications 3 and 4 listed):** `yes` and `scores` repeat the locked cut-offs; `boxPlaceholder` reads the unit from the title; `complete` repeats `missing`; `cardHints` repeats `poaLines`; the BMI question is defined three times; `hqPart` walks the HQ once per card.

**Not a defect (from the code review):** with a hospital weight and no height, typing the BMI the card asks for clears the hospital weight, which must then be typed again and is stored as the clinician's. That's Rahul's 13 Nov rule working as written, and report 6's T-23 step 4 tested it. If Rahul would rather keep a pre-filled weight when the BMI had nothing pre-filled, say so and it can go in with finding 1.

### What's sound

- **Verification 4's fixes:**
  - The hospital's own number typed back after the other was typed is the clinician's (unit cases for both directions and verification 1's case; J12's MUST journey; report 6's T-23).
  - A confirmed card goes through `withDraft` unchanged while no BMI or weight changed (unit case with an empty draft and a changed choice).
  - `assessmentResult` copies the extra results before the dispose.
- **Browser spot-checks (13 Nov, signed in as Cara Clinician):**
  - The MUST case in finding 1; **Undo changes** brought back 35.9 Hospital BMI and 110 Hospital weight. Nothing saved.
  - Ravi's assessments list: 200 through Hospital A1; 404 through another hospital's path and another trust's path; 401 with no cookie.
  - The patient API (`/api/p/…/assessments`): 404.
  - I didn't sign in as Val or Bea this round. Verifications 1 and 3 and report 6 (T-14) did, and no Go code or route has changed since.
- **Scope, tests, security and A-20:** as verifications 1 to 4 found. No new routes and no new Go logic. The A-20 grep shows no S49 file but the seed.
- **Docs:** the plan's Status, both READMEs, `decisions.md` (13 Nov) and `pending.md` are current. `repocheck` passes.

### Test results (13 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 644, web 390, evaluate 25, CDK 132, every Go package |
| `make test-integration` | pass: every package, `internal/httpapi` 30 s |
| `make e2e` | pass: 120 passed, 5 skipped in 12 min 15 s (the 5 are the Cognito journeys, AWS only) |

### Test data I changed

- Nothing saved. On Ravi's unsaved MUST card I typed a BMI and a weight and clicked **Undo changes**.
- The `make e2e` run added its usual E2E episodes.
- Cara Clinician is left signed in; I closed my tab.

### Suggested commits (after the fixes are verified)

As verification 1 listed below. Findings 1 to 3 go in commit 2 with their tests, and the J12 step in commit 3.

## Verification 4 (13 Nov 2026)

**Verdict: Needs fixes.** Verification 3's new pairing rule has a dead end on the MUST card (finding 1) and is undone for one kind of confirmed card (finding 2). Both are in `withDraft`. Everything else verification 3 asked for is built as asked, with tests, and test report 5 passed it in the browser.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (27 files changed, 15 new).
- **Read:** the brief, answers, build notes, verifications 1 to 3, and test report 5 (12 of 12 passed).
- **Ran:** `/code-review high` (9 findings; it found findings 1 and 2 on its own, and I had proved finding 1 first with a throwaway Vitest case, deleted after), `make lint test test-integration`, `make e2e` against Rahul's `make dev` (started 14:17, after the fixes), and the browser spot-checks below.

### Findings, worst first

1. **Must fix: after a BMI is typed, the hospital's own weight can't be entered (and the reverse).** [assessments.ts:384](../../../packages/clinical/src/assessments.ts#L384) (`withDraft`'s `draft[i.key] === i.value`), then [assessments.ts:395](../../../packages/clinical/src/assessments.ts#L395) (`unpaired`)
   - **What:** `withDraft` treats a typed number equal to the pre-filled one as "not changed", so it keeps the pre-filled source. `unpaired` then clears it again, because the other item is the clinician's.
   - **Seen in the browser** (Ravi's episode, a new MUST form, nothing saved, undone after): hospital weight 110 and Hospital BMI 35.9. I typed BMI 30: the weight went Missing, as Rahul's rule says. I typed 110 in the weight box: the box shows 110, the row stays orange **Missing** with no message, the badge says Incomplete and Confirm is off.
   - **Proved in Vitest:**
     - Hospital weight 60 with no height (verification 1's own case: BMI Missing, so the clinician types it): BMI 20.8 typed, then weight 60 typed, gives `[[20.8,"clinician"],[null,null]]`, Incomplete. Weight 60.1 works.
     - The mirror: weight 61 typed, then the pre-filled BMI 20.8 typed back, leaves the BMI Missing.
   - **Why it matters:** in verification 1's case the card asks the clinician for a BMI. Once they type it, the true weight can't be entered on that screen. The only ways out are a slightly different number, Save and reload, or Undo.
   - **Fix, test-first:** on a card with both, when the other measure is the clinician's, a number typed in this one's box is the clinician's too, even when it equals the pre-filled number. Unit cases for both directions and for verification 1's case; a J12 step (type a BMI, type the hospital's weight back, Confirm is on and the weight reads "Entered by clinician").
2. **Should fix with finding 1: a confirmed MUST card with a typed weight beside a pre-filled BMI shows as BMI Missing.** [assessments.ts:382](../../../packages/clinical/src/assessments.ts#L382)
   - **What:** `assessmentItems` leaves a confirmed card as confirmed, but the panel reads it through `shownItems`, which calls `withDraft`, and `withDraft` applies `unpaired` to every card, an empty draft included. The unit case "a confirmed card shows exactly what was confirmed" tests `assessmentItems` only.
   - **Effect:** such a card would show Incomplete on the episode page while the POA Summary shows its confirmed score.
   - **How likely:** only a row confirmed before verification 3's fix, or one sent straight to the API (Go checks shape only). I looked in the local database: no MUST row has different sources for the two. From the code and the review's script, not seen in the browser.
   - **Fix:** don't apply `unpaired` to an unchanged confirmed card (an empty draft). Change the unit case to go through `withDraft` with `{}`.
3. **Minor: `assessmentResult` reads the extra results after `m.dispose()`.** [assessments.ts:449](../../../packages/clinical/src/assessments.ts#L449) It works with today's survey-core. Read the values before the dispose, as the code did before this session.
4. **Cleanup (optional, as verification 3 listed):** `yes` and `scores` repeat the locked cut-offs; `boxPlaceholder` reads the unit from the title; `complete` repeats `missing`; `cardHints` repeats `poaLines`; the BMI question is defined three times.

### What's sound

- **Verification 3's fixes:**
  - The pairing rule in the cases report 5 tested (a typed weight clears the BMI, a typed BMI clears the weight, both typed are kept, a saved typed weight keeps the BMI Missing after a reload).
  - The audit journey raises `csv.field_size_limit`; the question about storing every listed id is in pending.md.
  - MUST's two rounding edges are in pending.md.
  - "The hospital has a weight but no height now (BMI … when confirmed)", with its web test.
  - `ready.test.ts` checks the seeded and e2e HQ files against `ready.ts`.
- **Browser spot-checks (13 Nov, signed in as Alex Author):**
  - The MUST case in finding 1; **Undo changes** brought back 35.9 Hospital BMI and 110 Hospital weight.
  - Mona's episode: MUST, VTE and Morse each show their confirmed result and "Confirmed by Cara Clinician…".
  - Ravi's assessments list: 200 through Hospital A1; 404 through Hospital A2's path; 404 through Trust B's path; 401 with no cookie.
  - The patient API (`/api/p/…/assessments`): 404.
  - I didn't sign in as Val or Bea this round. Verifications 1 and 3 did, and no Go code or route has changed since.
- **Scope, tests, security and A-20:** as verifications 1 to 3 found. No new routes and no new Go logic. The A-20 grep shows no S49 file.
- **Docs:** the plan's Status, both READMEs, `decisions.md` (13 Nov) and `pending.md` are current. `repocheck` passes.

### Test results (13 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 640, web 390, evaluate 25, CDK 132, every Go package |
| `make test-integration` | pass: every package, `internal/httpapi` 36 s |
| `make e2e` | pass: 120 passed, 5 skipped in 11 min 34 s (the 5 are the Cognito journeys, AWS only) |

### Test data I changed

- Ravi's episode ("Ravi MT S49 1430 Snore") has a new, unsaved **MUST** form.
- The `make e2e` run added its usual E2E episodes.
- Alex Author is left signed in on the Chrome tab.

### Suggested commits (after the fixes are verified)

As verification 1 listed below. Findings 1 to 3 go in commit 2 with their tests, and the J12 step in commit 3.

## Verification 3 (13 Nov 2026)

**Verdict: Needs fixes.** One bug in MUST (finding 1, with Rahul's answer) and one journey that now fails on this Mac's database (finding 2). Verification 2's fixes are built as asked, with tests, and test report 4 passed them in the browser.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (26 files changed, 14 new).
- **Read:** the brief, answers, build notes, verifications 1 and 2, and test report 4 (13 of 13 passed).
- **Ran:** `/code-review high` (10 findings; I reproduced the two that matter with a throwaway Vitest case, deleted after), `make lint test test-integration`, `make e2e` against Rahul's `make dev` (started 12:12, after the fixes), and the browser spot-checks below.

### Rahul's answer (13 Nov, asked during this verification)

**A weight typed on the MUST card clears a pre-filled BMI, and a typed BMI clears a pre-filled weight.** The cleared item is Missing, so the clinician enters it before Confirm. Chosen over a warning line and over leaving it as built. Recorded in [decisions.md](../../decisions.md) (13 Nov).

### Findings, worst first

1. **Must fix: a weight typed on the MUST card keeps the pre-filled BMI.** [assessments.ts:379](../../../packages/clinical/src/assessments.ts#L379) (`withDraft`), [assessments.ts:361](../../../packages/clinical/src/assessments.ts#L361) (the saved clinician's item is kept and the rest re-filled)
   - **What:** the one-source rule (8 Nov) is applied to the pre-fill only. A change on the card changes that one item.
   - **Proved in Vitest:** hospital BMI 20.8 and weight 60, 2 kg lost. Typing weight 50 and No to item 4, then Save, gives `must_bmi` 20.8 (observation) beside `must_weight` 50 (clinician) and "0 (Low risk), 3.8% of body weight lost".
   - **Seen in the browser** (Tess's confirmed MUST, not saved, undone after): weight 50 shows "Entered by clinician" beside "20.8 Hospital BMI", the badge says "1 (Medium risk), 9.1% of body weight lost", and Confirm is on. At 170 cm, 50 kg is a BMI of 17.3, which scores 2 more: High risk.
   - **Why it matters:** it's verification 1's finding 1 by another route. The BMI is from a heavier weight than the one shown, so MUST under-scores, and Go accepts the Confirm (it checks shape only).
   - **Fix (Rahul's answer), test-first:**
     - On a card with both a BMI and a weight, a clinician's weight (in the draft, or kept from a save) leaves a pre-filled BMI Missing; a clinician's BMI leaves a pre-filled weight Missing. Both typed: both kept.
     - A card with a BMI only (STOP-Bang, VTE) is unchanged.
     - Unit cases in `assessments.test.ts`: the draft, the saved row re-filled, the reverse, and both typed. A J12 step: type a weight, the BMI goes Missing and Confirm is off.
     - A confirmed card still shows exactly what was confirmed.
2. **Must fix: `make e2e` fails on this database: 1 failed, 119 passed, 5 skipped.** [test_audit.py:117](../../../e2e/journeys/test_audit.py#L117)
   - **What:** `test_an_admin_finds_a_clinicians_actions_filters_them_and_exports_the_csv` stops with `_csv.Error: field larger than field limit (131072)`. It fails again when run alone.
   - **Why:** not S49's code. A "Listed episodes" audit row holds the id of every episode listed. Hospital A1 now has 3,646 episodes (e2e and test data), so that row's "Episode IDs" cell is about 138,000 characters, over Python's default limit. The builder's run at midday was just under it.
   - **Fix:** raise the limit in the journey (`csv.field_size_limit`), so the run doesn't depend on how much data the local database holds.
   - **For Rahul, in pending.md (older than S49, S11):** the export's "Episode IDs" cell for one list request grows with the hospital's list. A spreadsheet cell holds 32,767 characters, which is about 860 episodes. Should a list row store or export every id?
3. **For a clinician to check before go-live (recorded choice, not a defect):** MUST scores the weight-loss percentage after rounding it to 1 decimal place. [ready.ts:366](../../../packages/clinical/src/ready.ts#L366)
   - 89.6 kg now and 10 kg lost is 10.04%: shown "10%", scored 1 (BAPEN's >10% is 2).
   - 95.8 kg and 5 kg is 4.96%: shown "5%", scored 1 (BAPEN's <5% is 0).
   - The builder chose "scored as shown" so the card never shows 10% beside a score for over 10% (decisions.md, 7 Nov). Add these two cases to pending.md's MUST line.
4. **Minor: "There's no BMI now" can be said of a BMI that still exists.** [assessments.ts:242](../../../apps/web/src/features/episodes/assessments.ts#L242) MUST confirmed on the patient's pair, then a hospital weight alone is recorded (precedence on): the pair rule leaves the BMI Missing and the card says there's none, though STOP-Bang and VTE on the same episode still use the patient's. Better: "The hospital has a weight but no height now (BMI 25.7 when confirmed)." Fold it into finding 1's fix if it's natural.
5. **Minor: nothing ties `demo-risk-hq.json` to `ready.ts`.** [main.go:116](../../../apps/api/cmd/seed/main.go#L116) The file is a copy regenerated by hand (twice this session). After the next wording change, a fresh seed publishes a stale group, and "New version" of the Demo risk HQ then can't be published. `keys_test.go` compares keys only. A Vitest case that runs the ready-made checks over the seed and e2e files would catch it. Optional.
6. **Cleanup (optional, none blocks):**
   - `AssessmentItem.yes` and the per-item `scores` repeat cut-offs the locked expressions hold (MUST's BMI ≤ 20, VTE's age > 60 and BMI > 30), and nothing in `apps/web` reads `yes`. [assessments.ts:70](../../../packages/clinical/src/assessments.ts#L70)
   - `boxPlaceholder` reads the unit from the title's last brackets, and the panel picks its error text by comparing the placeholder with "BMI". [assessments.ts:192](../../../apps/web/src/features/episodes/assessments.ts#L192)
   - `cardHints` repeats `poaLines`' expression for a confirmed result; `complete` repeats `missing` on the next line. [AssessmentsPanel.tsx:116](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L116)
   - `hqValues` still walks the shown sets once per card (different keys each).
   - The BMI question is defined three times (left on purpose, build notes).

**Rejected from the code review:** "the 'Same as…' reopen check is too loose". While the "snore" search is on, the first candidate is "snoring", so asserting the first isn't "snoring" does prove the list was reset.

### What's sound

- **Verification 2's fixes:**
  - VTE is Incomplete while either result is (unit cases for a bleeding and a thrombosis item; J12 checks Incomplete and Confirm off).
  - `alwaysFresh` is on the four recent-time questions, and the assessments' own copies drop it, so their JSON is unchanged.
  - `pairSource` with precedence off, with its unit case.
  - The cleanups (the age and BMI placed by key, the stale comment, the placeholder).
- **Browser spot-checks (13 Nov, on Tess's first episode):**
  - Val (Hospital A2): 404 on the episode, its assessments list, Add and Confirm through A2's path, and 404 on the list through A1's path. Her own list: 200.
  - Bea (Trust B): 404 through both trusts' paths for the list, Add and Confirm. Her own list: 200.
  - The patient API (`/api/p/…/assessments`): 404.
  - The MUST case in finding 1.
- **Scope, tests, security and A-20:** as verifications 1 and 2 found. No new routes and no new Go logic. The A-20 grep shows no S49 file but the seed.
- **Docs:** the plan's Status, both READMEs, `decisions.md` and `pending.md` are current. `repocheck` passes.

### Test results (13 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 630, web 389, evaluate 25, CDK 132, every Go package |
| `make test-integration` | pass: every package, `internal/httpapi` 25 s |
| `make e2e` | **1 failed, 119 passed, 5 skipped** in 12 min 11 s (finding 2; the 5 are the Cognito journeys, AWS only). J12's 19 cases and both question-library journeys passed. |

### Test data I changed

- Nothing saved. On Tess's MUST card I typed a weight and clicked **Undo changes**.
- The `make e2e` run and the audit journey's re-run added their usual E2E episodes.
- Cara Clinician is left signed in on the Chrome tab.

### Suggested commits (after the fixes are verified)

As verification 1 listed below. Finding 1's fix goes in commit 2 with its tests, finding 2's in commit 3, and this verification's `decisions.md` entry in commit 4.

## Verification 2 (13 Nov 2026)

**Verdict: Needs fixes.** One bug in the VTE card (finding 1) and one question for Rahul about pre-fill (finding 2). Verification 1's two fixes are built as Rahul answered, with tests, and test report 3 passed them in the browser.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (26 files changed, 13 new).
- **Read:** the brief, answers, build notes, verification 1, and test reports 1 to 3 (report 3: 12 of 12 passed).
- **Ran:** `/code-review high`; I checked its findings against the code, and proved finding 1 with a throwaway Vitest case (deleted after).
- **Not done:** `make e2e` and the browser spot-checks. `make dev` wasn't running (nothing answered on :3000 or :8080). Both are owed in verification 3, after the fixes.

### Findings, worst first

1. **Must fix: the VTE card looks complete, and offers Confirm, while bleeding items are unanswered.** [AssessmentsPanel.tsx:115](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L115)
   - **What:** `complete` is "the first score has a value". VTE has two results, and the 11 bleeding items aren't inputs of the first (the thrombosis count). `assessmentResult` also drops an empty second result without saying so ([assessments.ts:420-424](../../../packages/clinical/src/assessments.ts#L420-L424)).
   - **Proved:** every thrombosis item answered and B1 left empty gives `{"score":1,"band":"Thrombosis risk: consider thromboprophylaxis","text":"1 (Thrombosis risk: consider thromboprophylaxis)"}`: not Incomplete, and no bleeding count.
   - **What the clinician sees:** a blue badge with a thrombosis result and no bleeding count, no "Confirm once every item has an answer" line, and Confirm enabled. Clicking it gets Go's 400 "Answer every item before confirming…". Nothing wrong is stored as confirmed, because Go refuses it.
   - **Why it matters:** the bleeding count is what tells staff whether drug prophylaxis is safe, and the card gives no sign it's missing. The testers answered every item in order, so no round met this.
   - **Fix, test-first:** a card is complete only when no item is missing as well (`missing` is already worked out on the next line), or `assessmentResult` returns Incomplete while any Calculation of the content is empty. Add a unit case (VTE, one bleeding item empty → Incomplete, Confirm off) and one for the reverse (a thrombosis item empty).
2. **For Rahul to decide: the new time-window questions pre-fill from an earlier HQ.** [ready.ts:351-352](../../../packages/clinical/src/ready.ts#L351-L352), [ready.ts:565](../../../packages/clinical/src/ready.ts#L565)
   - **What:** Go's pre-fill skips only questions marked `alwaysFresh` ([prefill.go:121](../../../apps/api/internal/prefill/prefill.go#L121)). The library's own "What do you weigh now?" and "pregnant" questions are marked ([library.json:82-89](../../../apps/api/cmd/seed/library.json#L82-L89)). S49's ready-made questions aren't:
     - "Have you lost weight without trying in the last 3 to 6 months?" and "About how much…(kg)?" (MUST);
     - "Have you had a fall in the last 3 months?" (Morse);
     - "Are you pregnant, or have you had a baby in the last 6 weeks?" (VTE).
   - **Effect:** a patient's answers from an HQ months ago are copied into a new episode's HQ. If they accept the pre-filled set unchanged, MUST opens with the old weight lost marked "HQ", Morse with the old fall, and VTE with the old pregnancy answer. The ready-made Group is locked, so an author can't switch on "Always ask fresh" themselves.
   - **Question:** should these four be always asked fresh?
     - (a) **Yes: mark the four `alwaysFresh` in `ready.ts`** (recommended). They ask about a recent window, as the library's weight and pregnancy questions do. The Demo risk HQ and `e2e/data/risk-hq.json` are regenerated; the assessments' own JSON should be checked for a new platform version, and the build notes should say which.
     - (b) No: leave them pre-filled, as the patient reviews pre-filled answers before sending.
   - I haven't run this in the browser; it follows from the code and the seed JSON.
3. **Minor: with precedence off, a patient's weight with no height beats a complete hospital pair.** [assessments.ts:321-325](../../../packages/clinical/src/assessments.ts#L321-L325)
   - **What:** `pairSource` counts the patient's pair as theirs when it has a weight only. MUST then opens with the BMI Missing and the patient's weight, though the hospital recorded a height and a weight. STOP-Bang and VTE on the same episode use the hospital BMI.
   - **Why it's minor:** it mirrors what Rahul agreed for the hospital's side (a weight without a height leaves the BMI for the clinician), the two numbers still come from one source, and the clinician is asked. Fix only if Rahul wants the complete pair preferred.
4. **Minor: the "Same as…" journey's reopen check is weaker.** [test_question_library.py:146](../../../e2e/journeys/test_question_library.py#L146) now passes with any two "Use this" buttons, so a filter left on after reopening would pass if it had two matches. Better: assert a candidate the "snore" search left out, by its test id.
5. **Cleanup (optional, none blocks):**
   - `VTE_ITEMS` places "Age over 60" and the BMI row by array index (`i === 1`, `i === 2`), [ready.ts:545-546](../../../packages/clinical/src/ready.ts#L545-L546). A reordered factor would move them without a compile error.
   - `vteGroup`'s comment names `vteForAssessment`, which doesn't exist (it's `withoutConditions` in `assessments.ts`), [ready.ts:553](../../../packages/clinical/src/ready.ts#L553). `CFS_SOURCE` is exported and unused, so the Frailty Scale's source and licence text is shown nowhere.
   - The number box's placeholder is "kg" for anything that isn't a BMI, [AssessmentsPanel.tsx:231](../../../apps/web/src/features/episodes/AssessmentsPanel.tsx#L231). The next number item in another unit would be mislabelled.
   - The BMI question is defined three times (STOP-Bang inline, MUST, VTE).
   - `hqValues` works out the shown sets once per card, and for the Frailty Scale, which asks for no keys.

### What's sound

- **Verification 1's fixes:** the pair rule (`pairSource`, with unit tests for precedence on and off), `visibleIf: {patientSex} <> 'male'` on VTE's three questions with the assessment's copy unconditioned, `reportedBody` giving a weight alone, the VTE labels beside their keys, and the renamed `numberInput`. Both of Rahul's answers are in `decisions.md` (8 Nov).
- **The name fix:** `ListAssessmentForms` reads `c.name` and drops the join on `questionnaires`; nothing else in the query changed, so its tenant scoping is as S48 left it.
- **A-20:** the grep shows nothing new (older comments and tests only).
- **Security and scope:** as verification 1 found; no new routes, no new Go logic.
- **Docs:** the plan's Status, both READMEs, `decisions.md` and `pending.md` are current. `repocheck` passes.

### Test results (13 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 625, web 388, evaluate 25, CDK 132, every Go package |
| `make test-integration` | pass: every package, `internal/httpapi` 26 s (Docker's `bin` had to be added to `PATH` in my shell) |
| `make e2e` | **Not run:** `make dev` wasn't running. The builder's run after verification 1's fixes was 120 passed, 5 skipped. |
| Browser spot-checks | **Not done**, for the same reason. |

### Suggested commits (after the fixes are verified)

As verification 1 listed below; the fix for finding 1 goes in commit 2 with its tests, and finding 2's (if Rahul says yes) in commits 2 and 3.

## Verification 1 (8 Nov 2026)

**Verdict: Needs fixes.** One correctness bug in MUST (finding 1), and one patient-facing change Rahul asked for during verification (finding 2). Everything else is sound: scope, tests, tenant isolation, A-20, security and docs.

- **Reviewed:** `31046e6` plus S49's uncommitted changes (26 files changed, 10 new).
- **Read:** the brief, answers, build notes, test plan, and test reports 1 and 2 (report 2: 15 of 15 passed).
- **Ran:** `/code-review high`.

### Rahul's answers (8 Nov, asked during this verification)

1. **MUST: keep BMI and weight paired.** BMI and weight always come from one source. With a hospital weight but no hospital height, fill the hospital weight and leave the BMI for the clinician (highlighted Missing, so they record a height). Never show a BMI worked out from a weight that isn't the one shown.
2. **VTE: hide the pregnancy, oestrogen contraceptive and HRT questions in the HQ for a patient recorded as male,** with tests.

Both are recorded in [decisions.md](../../decisions.md) (8 Nov).

### Findings, worst first

1. **Must fix: MUST mixes a patient-reported BMI with a hospital weight.** [assessments.ts:324](../../../packages/clinical/src/assessments.ts#L324)
   - **What:** `measurement()` picks the BMI and the weight separately. With `observations.precedence` on and a hospital weight but no hospital height, `hospitalBmi` is undefined, so item 1 takes the patient's own BMI while item 2 takes the hospital weight.
   - **Proved with a throwaway Vitest case** (deleted after): the HQ gave 165 cm and 70 kg, and the hospital gave 60 kg with no height. The card got `must_bmi` 25.7 (hq) and `must_weight` 60 (observation).
   - **Why it matters:** the two numbers describe different bodies. MUST's BMI step uses a BMI from a heavier weight than the one recorded, so it can under-score malnutrition risk. Both source labels look valid, so a clinician may not notice. A weight without a height is a normal Observations entry.
   - **Fix (Rahul's answer 1):**
     - Take the pair from one source.
     - With a hospital weight and no hospital BMI, fill the weight and leave the BMI Missing.
     - Add a unit test for that case, test-first, beside "the weight follows the precedence…" in `assessments.test.ts`. Cover both precedence on and off.
     - `sinceConfirmed` should still read sensibly when the BMI goes from a value to Missing.
2. **Must fix (Rahul's answer 2): VTE asks every patient "Are you pregnant…?".** [ready.ts:490-493](../../../packages/clinical/src/ready.ts#L490-L493)
   - **What:** `vteGroup()` has no condition on sex, so patients recorded as male are asked the pregnancy, oestrogen contraceptive and HRT questions.
   - **Fix:**
     - Hide those three for a patient recorded as male. Use the patient's sex variable, as the ready scores' sex-dependent parts do.
     - Patients recorded as female or Other still see them.
     - The assessment's pre-fill must treat a hidden question as unanswered (the clinician answers it), not as No.
     - Unit tests, plus the J12 journey or a patient case if one is cheap.
     - The seeded Demo risk HQ and `e2e/data/risk-hq.json` are written from `ready.ts`, so regenerate them. `TestSeedAndE2EQuestionnairesAgreeOnKeys` must still pass.
     - Because S48 reuses versions, the VTE assessment's content changes only if `vteGroup` is part of it (it is: `elements: () => [vteGroup(), …]`). A new platform version is then published on the next add, and existing episodes keep theirs. That's expected; say so in the build notes.
3. **Minor: MUST's weight now isn't pre-filled from an HQ that has a weight but no height.** [poa.ts:141](../../../packages/clinical/src/poa.ts#L141)
   - **What:** `reportedBody` returns the weight only together with a BMI. The clinician types it instead.
   - **Fix:** none needed. Fold it in only if the fix for finding 1 makes it natural (a weight alone is still one source).
4. **Minor: Go accepts sources that can't be true for S49's items.** [assessment.go:214](../../../apps/api/internal/assessment/assessment.go#L214)
   - **What:** `Check` stops only an observation on a choice item, so `cfs_level` with source `hq` and `must_wtloss_kg` with source `observation` are accepted. The integration case that sends `cfs_level` as `hq` is refused only because of the `patientAge` it adds ([assessments_integration_test.go:368](../../../apps/api/internal/httpapi/assessments_integration_test.go#L368)).
   - **Why it's minor:** this is provenance only. The sender is an authorised clinician at the hospital, and the score is unaffected.
   - **Fix:** optional. Tighten it (`noHq` and `measure` into the labels JSON), or leave it and correct the test's comment.
5. **Cleanup: VTE's keys are listed twice.** `VTE_LABELS` in [assessments.ts:92](../../../packages/clinical/src/assessments.ts#L92) repeats every VTE key from `ready.ts`. If they drift, `itemsOf` finds no items and the card shows none. Consider keeping the short label in `ready.ts`'s `VteFactor` tuples.
6. **Cleanup: the HQ part is worked out more than needed.**
   - **Where:** [assessments.ts:290](../../../packages/clinical/src/assessments.ts#L290).
   - **What:** `hqPart` caches by key set, so `reportedBody`, which builds a survey per shown set, runs once per assessment card (up to 8). It doesn't depend on the keys, and only cards with a `measure` need it.
   - **Fix:** cache it once per HQ.
7. **Cleanup: `bmiInput` now checks every number box** (kg too). Its name and the "the hospital BMI can't be checked" wording are BMI-only. See [assessments.ts:192](../../../apps/web/src/features/episodes/assessments.ts#L192).

**Rejected from the code review:** "Morse's two long mental-status buttons overflow a 200 px cell." In the browser they fit: the table widens the column, and both buttons show whole at desktop width.

**For a clinician to check before go-live (not defects; recorded in the build notes):**
- Morse's bands (45+ High; Morse's original paper often uses 51+).
- VTE's anticoagulant question is broader than the form's "concurrent use of anticoagulants known to increase the risk of bleeding".

### What's sound

- **Scope:** everything in the brief is built: MUST, VTE, Rockwood with hints, POA Summary lines, J12 with MUST, and S48's carried item (each form keeps its version's name, with its integration case). Falls is the Morse Fall Scale, per Rahul's answer 1.
- **Clinical content against the sources:**
  - MUST's cut-offs: BMI <18.5 scores 2 and 18.5–20 scores 1; weight loss ≥5% scores 1 and >10% scores 2; the acute disease effect scores 2.
  - The weight lost is a percentage of the weight before.
  - Morse's points: 25, 15, 0/15/30, 20, 0/10/20 and 0/15.
  - The DoH VTE form's 19 thrombosis and 11 bleeding factors, in its order.
  - The CFS 2.0 level names.
- **Tests:**
  - No new routes, so the five-case rule doesn't apply.
  - The integration cases cover the name fix, all eight assessments on one episode, and Confirm for MUST, VTE and the CFS.
  - Logic has unit tests.
  - J12 has the MUST journey.
- **Tenant isolation (checked in the browser on Rosa's episode):**
  - Val (Hospital A2): 404 on the assessments list through both A1's and A2's paths, and 404 on Confirm.
  - Bea (Trust B): 404 through both trusts' paths.
  - The patient API (`/api/p/…/assessments`): 404.
  - The changed query (`ListAssessmentForms`) only drops a join and reads `c.name`.
- **A-20:** the grep shows nothing new; its hits are older comments and tests.
- **Security:** no secrets. The patient side never shows assessment content (report 2, T-13). Rockwood isn't in `READY_SCORES`.
- **Docs:** the plan's Status, `docs/plans/README.md`, `docs/sessions/README.md` and `decisions.md` are updated. `repocheck` passes.
- **Print (report 2's hand check, done here, per Rahul's 24 Oct rule):** I rendered Rosa's POA Summary in print media at A4 width. Every label is 190 px wide, every value starts in the same column, and the VTE value wraps inside its own column.

### Test results (8 Nov)

| Check | Result |
| --- | --- |
| `make lint` | pass |
| `make test` | pass: clinical 616, web 386, evaluate 25, CDK 132, and every Go package |
| `make test-integration` | pass: every package, `internal/httpapi` 28 s |
| `make e2e` | Not re-run; the builder ran it (117 passed; the one failure was fixed and passed on its own). Run it after the fixes. |

### Test data I changed

- On Nora's episode ("Nora MT S49 1550 NoLoss"), I added a **Morse Fall Scale** form to look at its layout. It's left unsaved.
- Cara Clinician is left signed in on the Chrome tab.

### Suggested commits (after the fixes are verified)

1. **S49: an assessment form keeps its version's name.**
   - Files: `db/queries/assessments.sql`, `internal/db`, the integration case.
   - Tests: `TestAddAssessment`.
2. **S49: MUST, Morse, VTE and the Clinical Frailty Scale.**
   - Files:
     - `packages/clinical`: `ready.ts`, `assessments.ts`, `poa.ts` and their tests;
     - `internal/assessment/*.json` and its test;
     - web: `assessments.ts`, `AssessmentsPanel.tsx`, `PoaSummary.tsx` and their tests;
     - the integration cases.
   - Tests: clinical, web, `go test ./internal/assessment`, `TestListAssessments`, `TestSaveAndConfirmAssessment`.
3. **S49: Demo risk HQ; J12.**
   - Files: `cmd/seed` (`main.go`, `keys_test.go`, `demo-risk-hq.json`), `e2e/data/risk-hq.json`, J12, `test_question_library.py`.
   - Tests: `go test ./cmd/seed`, J12.
4. **S49: docs.**
