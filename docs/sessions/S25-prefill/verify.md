# S25 verification

## Verification 1 (23 Oct 2026, automatic run)

**Verdict: Verified.** Everything in the brief and Rahul's answers is built: pre-fill by key, then by exact wording, with S23's safeguards; the opening card with both buttons, recorded; the markers; "started" means saved or confirmed; staff on the patient's behalf. Pre-fill reads only the form's own trust. I checked that the cross-trust test fails when that scoping is removed (details below). Nothing I found fills a wrong answer silently. Every automated test passes. What's left from `/code-review high` is minor: each item is rare or needs two tabs open at once. These items are now in `pending.md`, and two of them are for S26.

### What I checked

- **Tenant isolation.**
  - `ListPrefillSources` (`apps/api/db/queries/episodes.sql`) filters on `e.org_id = org AND p.org_id = org`, takes the trust's `org_id` as a parameter, and excludes the form itself. A closed account's records drop out, because S22 cleared their `account_id`.
  - **Mutation test:** I replaced that `WHERE` with `$1 IS NOT NULL` in the generated `episodes.sql.go`, then ran `TestPrefill/another_trust's_answers_are_never_used`. The test **failed**, because trust B's `q_smokes` was filled in. I then restored the file (`cmp` confirms it is byte-identical).
  - The two new routes are wrapped in `hosp`/`session` and `audited`. In Chrome, as Cara (Hospital A1 only), a `POST …/h/<A2>/episodes/<A1 episode>/patient-form/prefill/…` returned **404**.
- **Silent wrong fills (healthcare).** I read `internal/prefill` line by line:
  - Clinician-only questions are never filled. That covers a clinician-only question, group, page or set, and a clinician-only column inside a repeating group. "Always ask fresh" questions, the profile, signatures, uploads and BMI are never filled either.
  - The kind of answer, unit, format and scale must match. A choice must map to the new question's options: by value for a key match, by exactly one option text for a wording match. Multiple choice and grids are all-or-nothing.
  - A wording match never uses a text that appears twice in either questionnaire. "Other" text travels only where the new question has Other and it's chosen.
  - With S23's lock (keyed answer shapes are frozen across the trust) and `SameWording` (it needs identical column texts), column differences in repeating lists can't arise in the same trust.
  - The SQL `PrefillPatientAnswers` uses `ON CONFLICT DO NOTHING` and runs only while the form is open, so nothing that was saved is ever overwritten.
- **The builder's two product choices.**
  - **(a) The newest earlier form that asked a question decides, with no fallback to older forms.** I agree. It's the "if in doubt, don't fill" reading.
    - The common case is a follow-up question that was hidden in the newest form ("What happened?" after "No problems this time"). Falling back would bring back an answer the patient has since contradicted.
    - The cost is only that some answers aren't filled, and the patient answers them.
  - **(b) Send waits for each pre-filled set's card.** I agree: it's the safety side, it's what PRE-02's "Please check them" asks for, and the testers found it clear.
    - It's enforced only in the browser. Go can't tell which pre-filled answers are shown, because it never evaluates conditions, so this matches the rest of the HQ's completeness checks.
  - Neither needs Rahul before S26. Both are recorded in `decisions.md` as builder's defaults, so he can reverse either one.
- **Hidden answers.** A pre-filled answer to a question its condition hides is neither counted nor lost (unit test and J16), and SurveyJS's default `clearInvisibleValues` is kept. The clinician's episode page and the POA Summary use `shownSets` and skip hidden questions, so a hidden pre-filled answer doesn't show as the patient's.
- **Before the patient sends,** the clinician's episode page shows "Waiting for the patient" for each set (checked in Chrome on an untouched pre-filled follow-up HQ). Revision-0 pre-filled rows therefore never look like typed answers to staff in S25.
- **UK hardcoding:** the grep shows only comments (`format.ts`, `patient.go`).
- **Docs:**
  - The plan's Status section, `docs/plans/README.md`, `docs/sessions/README.md`, `decisions.md`, `pending.md`, `CLAUDE.md` and `apps/web/AGENTS.md` are all updated.
  - I corrected one stale line in `CLAUDE.md`: "Started" also covers a card that has been answered (test report 1, found 1).
  - `repocheck` passes.

### Spot checks in Chrome (390 × 844 iframe, "rahul chrome")

These used the tester's patient Sam S25 MT1243 Round2, whose Demo pre-op HQ was sent, plus a new Demo follow-up HQ episode I created as Cara.
1. **The list:** both sections showed **Please check / Check**, "0 of 2 sections complete", and Send was disabled.
2. **The card on *Your health*:** "We've filled in 5 answers from your questionnaire on 23 Oct 2026. Please check them."
   - The 5 are snoring, diabetes (matched by wording), sickness after an anaesthetic, the anaesthetic problem, and its details.
   - Weight is Always ask fresh, and it was not filled (I checked the API response).
3. **Everything is still correct:** the pregnancy question's text was at the top of the screen (F1 stays fixed). The markers read "From your answers on 23 Oct 2026".
4. **Changing an answer:** setting sickness to **No** removed its marker, and the other markers stayed.
5. **Staff:** Cara's `GET …/patient-form` returned both sets' `prefill`. The same request with the wrong hospital returned 404 (above).

### Minor findings (in `pending.md`, not blocking)

All of these are from `/code-review high`; my judgement of each is below.
1. **A tab can overwrite a pre-filled set without the 409 stale check** (`SavePatientAnswers`, `episodes.sql:262-278`): a pre-filled row has revision 0, which is also what a browser sends when it has "no row yet".
   - **How it happens:** tab 1 loads while a set has nothing to fill. Then an earlier HQ is sent elsewhere, and another open pre-fills the set. Tab 1's first save then replaces the pre-filled answers with what the patient typed in tab 1.
   - **Effect:** nothing wrong is shown as the patient's. The patient's typed answers win, and other tabs get the usual 409 on their next save. But the `prefill` map then describes answers that aren't in the row, and S26's "changed by patient" would mislead.
   - **Fix (S26):** give a pre-filled row its own starting revision, or have the save refuse revision 0 against an existing row.
2. **A second, different card button returns 204, and that tab carries on as if its choice was recorded** (`prefill.go:174-186`). The server keeps the first button, as decided. This needs two tabs. Returning the stored choice would let the page follow it.
3. **An earlier repeating-list row whose only non-empty cells are in columns the new list lacks is dropped** (`prefill.go` `fitRows`, `if len(kept) > 0`). In the same trust this can't happen: S23's lock freezes columns, and a wording match needs identical columns.
   - **S26 (other trusts):** "if in doubt, don't fill" suggests refusing the whole answer if any non-empty row would be emptied.
4. **A whole Question Set hidden by a set condition is still pre-filled and sent** (`prefill.go:43`). This is the same as answers typed into a set that later became hidden: the clinician's view and the POA Summary skip hidden sets (`shownSets`).
   - **S26:** its clinician markers should mark only shown sets and questions. This is already in `pending.md` for questions.
5. **Small ones:**
   - While any patient set has no row, every open re-reads and re-walks every earlier form, and adds those episodes to the audit row again. This is already in `pending.md`.
   - `awaitsCheck` builds a second SurveyJS model per pre-filled set on each list render (`PatientHQ.tsx:213`).
   - The "started" rule is copied into `accounts.sql` and `jobs.sql`.
   - `showPrefillMarkers`' clean-up is never called (`PatientHQ.tsx:446`). The model is built once per set, so this is harmless.
6. **Hospital scope (an observation, as agreed):** pre-fill reads the trust's earlier forms at every hospital (S25 answers Q2). So staff at Hospital A1 filling in for the patient see answers that came from an A2 episode, as values in the new form, not the A2 episode itself. This is what Rahul chose; it's noted here so that S26's clinician markers can say which hospital an answer came from, if he wants that.

### Test results

- `make lint`: passed.
- `make test`:
  - Go: all ok.
  - Vitest: 218 passed (web) and 140 passed (clinical).
  - CDK: 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against `make dev`): **62 passed, 5 skipped**, no failures (J16 included).

### Hand checks for Rahul

- Optional: T-03 in Chrome's real device toolbar at 390 × 844. After **Everything is still correct** on *Your health*, the pregnancy question's text should be on screen. The tester and I used a 390-wide iframe, and J16 checks it on WebKit at iPhone 13 size on every run.

### Commits

1. `S25: pre-fill from the same trust (internal/prefill, by key then wording), episode_answers.prefill and the card's button (migration 00025), the patient's opening card, Please check and markers, started = saved or confirmed, the seeded Demo follow-up HQ`, with its tests:
   - Go unit tests (`internal/prefill`, `cmd/seed`);
   - integration tests (`prefill_integration_test.go`, fixture `prefill.json`);
   - Vitest (`prefill.test.ts`, `format.test.ts`);
   - e2e (`test_j16_prefill.py`, `data/prefill-hq.json`, README row, `support.choose`, `test_question_library.py`).
2. `docs: S25 verified — build notes, test plan and reports, verification, decisions, pending, CLAUDE.md, AGENTS.md, plan and session status`.

Next: `/build-session S26`.
