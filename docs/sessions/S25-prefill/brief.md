# S25: Pre-fill and the patient's confirmation screens

**Plan:** C4 ([c4-question-keys-prefill.md](../../plans/c4-question-keys-prefill.md)), part 3 of 4.
**Requirements:** PRE-01, PRE-02, PRE-03, PRE-06; A-9 (answers never expire); PX-06. Same trust only: other trusts are S26.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S24 verified.

**Ask Rahul first:**
1. **When pre-fill happens.**
   - **Recommended:** when a Question Set is first loaded with no patient answers yet, Go fills it and stores the result as the patient's answers, in the same request. It never overwrites anything the patient typed, and runs once per set.
   - It runs the same way when staff fill in the HQ on the patient's behalf (PAT-09).
2. **Where answers come from.** **Recommended** (the plan's rules, made exact):
   - every **submitted** HQ form on the patient records linked to the same account, in the same trust, newest submit first; with no account, the same patient record only;
   - per key: the clinician's validated copy if there is one, else the patient's answer;
   - any episode status counts, archived and cancelled included (answers never expire); a closed account's old records don't (S22 unlinked them).
3. **What is never pre-filled.** **Recommended:**
   - clinician-only questions, `alwaysFresh` questions and the Profile (PRE-06);
   - a choice answer whose value isn't one of the new question's options (it was removed under S23's lock);
   - a repeating panel's rows are copied whole, keeping only the columns the new panel has;
   - "Other" text (`<key>-Comment`) travels with its key.
4. **What `prefill` stores.** The plan has `{key: {fromForm, answeredAt}}`.
   - **Recommended:** `{key: {fromForm, answeredAt, value}}` on the patient's `episode_answers` row. `answeredAt` is the source form's submit date (the one the patient sees). `value` is what was copied, so S26 can show "changed by patient" without re-reading the old form.
5. **The Question Set's opening card (PRE-02).**
   - "We've filled in 23 answers from your questionnaire on 3 Mar 2026. Please check them." With answers from several dates it names the newest: "…from your earlier questionnaires, the latest on 3 Mar 2026."
   - **Check each answer** goes through the set as normal. **Everything is still correct** goes to the first question that wasn't pre-filled, or to the next set if there's none.
   - **Record which button was pressed, and when** (per set, in `prefill`), for the clinician (S26) and the clinical-safety hazard log (OPS-04). **Recommended: yes.**
6. **The marker (PRE-03):** "From your answers on 3 Mar 2026" under each pre-filled question, until the patient changes that answer. **Recommended: yes.**

## Build

- **`internal/prefill`** in Go, test-first, a plain JSON walk (Go still never evaluates SurveyJS):
  - given the new chapter's patient questions (S23's `keys` walk, after `chapter.ForPatient`) and the earlier answers in order, it returns the data and the `prefill` map;
  - unit tests: precedence (validated over patient, newest first), every skip rule, dynamic panels row by row, `-Comment`, a removed option value.
- **`episode_answers.prefill jsonb`** (migration 00024 or next).
- **The patient routes** (`GET /api/p/episodes/{eid}` and the staff on-behalf route) run pre-fill as answered, and return `prefill` with the answers. Reads stay audited.
  - The query that finds earlier answers takes the trust's `org_id` as a parameter, so S26 can widen it deliberately and every other caller stays scoped to one trust.
  - Consider the `episode_hq` view suggested in [pending.md](../../pending.md) (the HQ-form join is copied into about 9 queries) if this adds more copies.
- **The patient's HQ** (`features/patient`): the opening card, its two buttons and the markers. Check how SurveyJS keeps values for questions a condition hides (`clearInvisibleValues`) and that a hidden pre-filled answer isn't lost or counted.
- **Integration tests:**
  - pre-fill from the same trust, clinician copy preferred, the newest form first;
  - clinician-only, `alwaysFresh` and removed-value answers are never pre-filled;
  - a second open doesn't pre-fill again, and nothing the patient typed is overwritten;
  - the `prefill` metadata and the button pressed are stored;
  - another trust's answers are never used (S26 adds the consent path).
- **End-to-end, J16** (`test_j16_prefill.py`, phone size; J10–J15 are taken by C5–C9's plans): a patient with a submitted, reviewed episode gets a new HQ, sees the card, chooses **Everything is still correct**, answers the rest and submits. S26 adds the clinician's half.

## Not in this session

- The clinician's markers (PRE-04) and other trusts (S26).
- Pre-filling assessments (C7) and PROMs baselines (C9).

## Manual test focus

Seed or make: a patient whose first episode (the Demo pre-op HQ, library keys) was submitted and reviewed, with one answer changed by the clinician; then a second episode on a different HQ sharing some keys.
1. Open the new HQ as the patient at phone size: the card counts the right number and shows the right date. Pre-filled answers carry the marker; the clinician's changed answer is the one used.
2. Everything is still correct jumps to the first unfilled question; Check each answer walks through.
3. Change a pre-filled answer: its marker goes. Leave and come back: nothing is re-filled over it.
4. Always ask fresh (weight), clinician-only questions and Profile are never pre-filled.
5. Fill in on the patient's behalf as Cara: pre-filled the same way.
6. A patient whose only earlier episode is at Trust B gets no pre-fill at Trust A.
