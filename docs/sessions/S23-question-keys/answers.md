# S23 answers

Asked by the conductor (`/run-sessions S23 S26`), 22 Oct 2026. Rahul asked for more context first, and the conductor re-asked the questions with the background of plan C4.

## Rahul's change to the design: pre-fill matches by key, then by exact wording

Rahul's aim: **if a patient answered something before, they shouldn't have to answer it again.** Everything that can safely be copied is copied, and the patient can change any answer. Key-only matching (the plan) would leave questions with the same wording but different keys un-filled, e.g. two HQs written separately, or another trust's HQs. So he chose a **hybrid** (affects S23, S25 and S26):

1. **Same key:** always a match (library inserts, "Same as…", new versions).
2. **Otherwise, exact wording:** the same question text (ignoring case, spaces and punctuation), the same kind of answer, and, for choices, every option **text** the same. Safeguards:
   - a text used more than once in either questionnaire (e.g. "Please give details") never matches by wording;
   - a choice answer is copied only if its option text exists in the new question (mapped from text to the new question's own value);
   - a repeating list matches only if the list's text and its column texts match.
3. **"If in doubt, don't fill."** Authors can stop any question pre-filling with "Always ask fresh". The patient sees a marker on every pre-filled answer, and the clinician sees where each came from and how it matched (S26).

In S23 this changes only the label (Q5) and the walk, which must also return each question's text and option texts.

## Q1. Where Go finds a trust's keys

**Answer: (a), walk the JSON on demand** (recommended). Walk the trust's published and retired chapters on each request; no `question_keys` table. Add one only if it gets slow.

## Q2. What "option values are locked" means

**Answer: (b), freeze the option list completely**, not the recommended (a). Once a keyed question is published, a later publish of any HQ in the trust holding that key must have **exactly the same options**: no adding, no removing.

**Follow-up: wording is frozen too** (Rahul chose this over "rewording allowed"). The option **texts** may not change either. So the publish check compares each option's value **and text** with the published one, and refuses any difference, naming the question and the option.

## Q3. Keys unique across the whole HQ

**Answer: yes, in the editor and at publish** (recommended). The editor refuses a duplicate across the HQ's sets, and Go's publish refuses one, naming both sets.

## Q4. "Always ask fresh"

**Answer: yes, on patient questions** (recommended). A switch beside Clinician only, with a badge on the card, on every answerable question except clinician-only ones.

## Q5. The PRE-05 label

Rahul first asked: "is it not going to auto fill for the patient if the question and option matches?" That led to the hybrid above.

**Answer: count both kinds of match, and say how** (recommended). "Pre-fills from 3 other HQs"; clicking lists each questionnaire and whether it matched **by link (key)** or **by same wording**. Otherwise "New question: won't pre-fill". Computed by Go once when the editor opens, from the trust's published and retired questionnaires (other questionnaires, not versions). Another trust's HQs never count.

## After verification 1 (22 Oct): what else the publish lock freezes

Asked by the conductor, after verify.md item 1 found that a published key could change its kind of answer (single → multiple choice, number → date) and still publish.

**Answer (Rahul): "it should freeze all kind of answers. Any kind of answer."** Once a keyed question is published, **everything that defines what an answer can be is frozen**, for every kind of question, not only the options: the kind of answer (single choice, multiple choice, number, date, text, signature, file, grid, rating, repeating panel…), the unit, a grid's rows and columns, a rating's scale, a repeating panel's columns, and the options (values and wording, as Q2). Changing any of them needs a new question with a new key. Publish refuses with 422 naming the question and what changed.

## After test report 3 (22 Oct): what a draft is compared against

R-05 failed: a new version of the seeded Demo pre-op HQ, unchanged, was refused, because old local test HQs ("S02 test HQ …") published `q_diabetes` with different answer shapes before the lock existed.

Rahul: **don't worry about the old data**; it isn't used anywhere. What matters is that nothing can change published data, now or after a later change to the logic. A published HQ version stays read-only (it already is), and changes go into a new version or a copy, because changing an HQ that episodes have answered would be a medical risk.

**Answer: keep the strict rule, as built.** A keyed question in a draft is compared with **every** published question holding that key in the trust (not just the HQ's own last version). Conflicting old test data is cleaned up, not accommodated: the local database is disposable (decisions.md §1).

## After verification 2 (22 Oct): date and text formats are part of the answer

Verification 2 found a date question could switch between full date, month and year, and year only after publishing. Rahul's "freeze all kind of answers" (after verification 1) covers it: **a date's format and a text question's input format (email, phone, NHS number, number…) are part of the frozen answer shape, and of the wording match.** Rahul allowed S23 one more fix round, retest and verification beyond the run's limits (22 Oct).
