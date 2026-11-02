# S46 answers

Rahul, 29 Oct 2026 (asked at the start of `/build-session S46`).

1. **Ranges and limits: platform defaults that each trust can change** (option b, not the recommended a). The builder's reading:
   - Go holds the platform defaults: each kind's soft range (a warning; the reading can still be saved) and hard limits (refused). They're made up until a clinician checks them.
   - **A trust's whole-trust admins can change both for their trust** (a trust-level setting, in the existing settings page). Hospitals follow their trust.
   - **Some checks stay fixed in Go whatever a trust sets:** no negative values, a diastolic not above its systolic, no time in the future, and a hard range that must contain its soft range. A trust's value that breaks one is refused when it's saved (400, as for any setting).
2. **Correct and Entered in error, each with a reason** (option a, recommended), as S44b does for the ASA grades.
3. **One stored unit per kind; each country's display units come from Go's country config** (option a, recommended). The UK's display units are the stored ones, and nothing converts until a second country needs it.
4. **The C7 split is agreed:** S46 observations, S47 assessments with STOP-Bang, S48 DASI, PRISMA-7 and Apfel, and S49 MUST, falls, VTE and Rockwood.
