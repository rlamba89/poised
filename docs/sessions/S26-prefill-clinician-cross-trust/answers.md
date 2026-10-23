# S26 answers

Asked by the conductor (`/run-sessions S23 S26`), 22 Oct 2026.

**Context:** pre-fill matches **by key, then by exact wording** ([S23's answers.md](../S23-question-keys/answers.md)). Across trusts that matters most: another trust's HQs share keys only through the platform library, so most cross-trust matches will be by wording. The same safeguards apply.

## Q1. What the clinician sees (PRE-04)

**Answer: all three** (recommended):
- on each pre-filled answer: "Pre-filled from *Demo pre-op HQ*, 3 Mar 2026", plus the hospital when it's another trust's, **and whether it matched by link or by wording**;
- "Changed by patient" where the final answer differs from the pre-filled one, showing the old value as corrections do today;
- per set: "Patient chose Everything is still correct" (S25's recorded button).

## Q2. Both trusts must allow it

**Answer: both trusts' `crossTrustPrefill` on** (recommended). Switching either off stops new offers; answers already copied stay.

## Q3. The patient's agreement

**Answer: ask once per trust** (recommended). Before the first Question Set: "Use the answers you gave St Mary's in March 2026?" with **Yes, use them** / **No, thanks**. Stored per account and source trust (`patient_prefill_consents`). Yes fills only questions still unanswered and untouched, after same-trust pre-fill. The choice can't be changed in this session.

## Q4. On the patient's behalf

**Answer: same-trust answers only, no consent screen** (recommended). Another trust's answers are never used on the patient's behalf, even if the patient said Yes earlier.

## Q5. Auditing the other trust's read

**Answer: yes, audit both trusts** (recommended). An `audit_log` row under the source trust as well (who: the patient; what: the source form ID).

## After verification 1 (23 Oct): how another trust's answers match

Verification 1 found that a key match across trusts copied an option value without checking its meaning: each trust makes its own keys, and two trusts can both have a `q_smokes` that mean different things. The builder then made other trusts' answers match by exact wording only.

**Answer (Rahul): platform library keys plus wording** (recommended). For answers from another trust, a key counts only if it's a **platform library key** (`library_questions` with `org_id` NULL, the same for every trust), and even then every choice is mapped by its option **text** (no fill if a text doesn't match). Every other question must match by exact wording, with all the wording safeguards. The clinician's marker says how it matched ("by link" for a library key, "by same wording" otherwise). Asked by the conductor.
