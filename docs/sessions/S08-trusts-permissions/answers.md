# S08: answers

Asked by the conductor (`/run-sessions`) on 12 Oct 2026, before the run started.

1. **A time zone per hospital** (proposed under A-20)?
   - **Answer: yes** (recommended).
2. **Role mapping to `clinician < super_clinician < admin`** (A-5).
   - **Answer: map as proposed** (recommended): author/reviewer/publisher → super clinician; hospital_admin → admin; viewer and clinician → clinician. Reseed.
3. **The `dev` branch** (7 Oct, commit `557be2c`).
   - **Answer: ignore it entirely.** Don't read it or reuse it; build from the plans. (The recommendation had been "reference only".)
4. **Patient record visibility.** Re-asked with an example, because the first wording wasn't clear: a patient has episodes at Hospitals A and B of one trust, and Cara works only at A.
   - **Answer: Cara sees the trust's patient record** (name, date of birth, contacts), so she can open an episode at A without retyping, **but only A's episodes.** Nothing about B's episodes is visible. (Recommended.)

5. **May a super clinician whose membership is for one hospital only edit and publish the trust's shared questionnaires?** (Raised by the builder; built as yes. Asked by the conductor during test round 1, 12 Oct.)
   - **Answer: no, trust-wide only.** Only super clinicians or admins with a whole-trust membership may edit or publish the trust's questionnaires (and, by the same rule, its option lists). A hospital-only super clinician can view them but not change them. (The recommendation had been "yes, as built".)
