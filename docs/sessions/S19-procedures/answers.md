# S19 answers

Asked by the conductor (`/run-sessions S18 S22`), 20 Oct 2026.

1. **Who manages the procedure list?** **(b) Super clinicians and admins** (recommended).
2. **Hospital-only procedures** (`hospital_id` NULL = every hospital of the trust): **yes** (recommended).
3. **INF-04 information content and PRM-01 PROMs plan columns:** **add them now** (not the recommendation of waiting for C5/C9).
4. **Today's free-text `procedure`:** **(a) replace it with an optional `procedure_id`** (recommended). *Conductor's note:* S18 chose reshape-and-re-seed over migrating, so builder: carry existing texts across, or re-seed, consistently with S18's approach.
5. **Removing a procedure in use:** **a Retired flag** (recommended). It's hidden from new episodes and stays on old ones, and nothing is deleted.
6. **Default HQ:** **yes** (recommended). The procedure's default, then the hospital's `defaultHq`, and the clinician can always change it.
