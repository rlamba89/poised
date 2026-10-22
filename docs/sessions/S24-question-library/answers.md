# S24 answers

Asked by the conductor (`/run-sessions S23 S26`), 22 Oct 2026.

**Context from S23's answers:** pre-fill matches **by key, then by exact wording** (see [S23's answers.md](../S23-question-keys/answers.md)). So "Same as…" is mainly for questions **worded differently that mean the same** ("Do you smoke?" and "Do you currently smoke?"). Also, a published keyed question's options are **frozen completely**, values and wording (S23 Q2).

## Q1. The platform's starter library

**Answer: use the recommended list.** `smoker` (never, ex-smoker, current), `alcohol_units` (number), `snoring` (yes/no), `ponv_history` (yes/no), `anaesthetic_problems` (yes/no + details), `family_anaesthetic_problems` (yes/no + details), `height` (cm), `weight` (kg, Always ask fresh), `pregnant` (Always ask fresh), and `allergies` and `regular_medicines` as repeating panels. Made-up but sensible wording, seeded into every deployment (`org_id` NULL).

## Q2. Who looks after the trust's library

**Answer: whole-trust super clinicians and admins** (recommended). Save to library from the editor; remove from a library window. Platform items are seeded only (a superadmin screen comes with C5).

## Q3. What "Same as…" does

Re-asked after the matching change. **Answer: copy the key and the options** (recommended). Adopting replaces the question's `name` and its options (values and wording) with the chosen question's, and keeps its own title. Because of S23's freeze, its options can't be edited after that. `linkedFrom` records where it came from, for information only.

## Q4. What counts as compatible

**Answer: the same kind of answer** (recommended). Single choice (radio, dropdown, yes/no buttons) with single choice; multiple choice with multiple choice; a number with a number of the same unit; a date with a date; text with text. Repeating panels aren't offered for "Same as…".

## Q5. One `keys.Compatible`, in Go

**Answer: Go only** (recommended). The search route returns each candidate with `compatible` and a reason.
