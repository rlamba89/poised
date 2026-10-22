# S25 answers

Asked by the conductor (`/run-sessions S23 S26`), 22 Oct 2026.

## The matching rule (changed by Rahul, 22 Oct)

Pre-fill matches **by key first, then by exact wording**, with safeguards. The full rule is in [S23's answers.md](../S23-question-keys/answers.md). Rahul's aim: if a patient answered something before, they shouldn't have to answer it again. `internal/prefill` therefore needs, for each earlier form, the **chapter JSON it was answered on** (question texts and option texts), not only its answer keys. A wording match maps a choice answer by option **text** to the new question's own value. Unit-test both kinds of match and every safeguard: a duplicated text never matches by wording; a choice whose option text is missing; a different kind of answer; a repeating list whose column texts differ. When in doubt, don't fill.

## Q1. When pre-fill happens

**Answer: on the first open of each Question Set** (recommended). With no patient answers yet, Go fills it and stores the result as the patient's answers in the same request. Once per set; never overwrites what the patient typed. The same when staff fill it in on the patient's behalf (PAT-09).

## Q2. Where answers come from

**Answer: as recommended.** Every **submitted** HQ form on the patient records linked to the same account, in the same trust, newest submit first (no account: the same patient record only). Per question: the clinician's validated copy if there is one, else the patient's answer. Any episode status counts, archived and cancelled included; a closed account's old records don't.

## Q3. What is never pre-filled

**Answer: as recommended.** Clinician-only questions, `alwaysFresh` questions and the Profile (PRE-06). A choice answer whose value (or, for a wording match, option text) isn't in the new question. A repeating panel's rows are copied whole, keeping only the columns the new panel has. "Other" text (`<key>-Comment`) travels with its question.

## Q4. What `prefill` stores

**Answer: store the copied value** (recommended): `{key: {fromForm, answeredAt, value}}` on the patient's `episode_answers` row, `answeredAt` being the source form's submit date. **Also store how it matched** (`matchedBy`: `"key"` or `"wording"`), because the clinician is shown it (S26 Q1).

## Q5. The opening card (PRE-02)

**Answer: yes, the card and both buttons, and record the button** (recommended). "We've filled in 23 answers from your questionnaire on 3 Mar 2026. Please check them." (several dates: "…from your earlier questionnaires, the latest on 3 Mar 2026"). **Check each answer** walks through the set; **Everything is still correct** goes to the first question that wasn't pre-filled, or to the next set. Record which button was pressed, and when, per set, in `prefill`.

## Q6. The marker (PRE-03)

**Answer: yes** (recommended). "From your answers on 3 Mar 2026" under each pre-filled question, until the patient changes that answer.
