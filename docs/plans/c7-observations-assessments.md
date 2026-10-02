# C7: Observations and assessments

**Goal:** clinicians record observations. Assessments open pre-filled from the HQ, the observations and the profile, and the clinician completes them.

**Requirements:** OBS-01…04, §9 (assessments). **Depends on:** C4 (keys) and C6 (scores and bands).

## Data

- `observations`: episode_id, kind, value, value2 (diastolic), unit, taken_at, entered_by, superseded_by. One row per reading; a correction supersedes the old row instead of overwriting it.
- **Assessments** are platform library questionnaires of kind `assessment`, with locked wording. They're attached as `episode_forms` of kind `assessment`.

## Design

- **Observations panel** on the episode:
  - add a reading, see the history, correct a reading;
  - soft range warnings and hard limits (no negatives);
  - **hospital BMI** on the POA Summary, next to the patient's own figure;
  - the `obs_precedence` setting decides which BMI feeds scores.
- **Assessment pre-fill:**
  - items get their values by key from the HQ answers (C4), plus `obs.*` and the profile;
  - each item shows its **source** (HQ / observation / profile / clinician), and missing items are highlighted;
  - the clinician completes and confirms;
  - the score and band appear on the POA Summary.
- **Order:**
  1. **STOP-Bang**
  2. DASI
  3. PRISMA-7
  4. Apfel PONV
  5. MUST
  6. Falls
  7. VTE
  8. Rockwood (clinician-only)
  - MMSE is deferred (A-17).
  - **Ship STOP-Bang alone first if time is short.**

## Tests

- **Unit (written TDD):**
  - each instrument's scoring and bands, checked against worked examples from its published source;
  - BMI;
  - choosing the pre-fill source (obs_precedence).
- **Integration:**
  - observation create, history and supersede;
  - a negative value → 400;
  - attaching an assessment;
  - pre-fill sources reported per item;
  - another trust → 404.
- **End-to-end:** **J12**. A clinician records height, weight and blood pressure. STOP-Bang opens with 7 of 8 items filled; enter the neck size; the score and band show on the POA Summary next to the hospital BMI.

## Status

Not started.
