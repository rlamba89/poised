# S19: Procedures and episode dates

**Plan:** C3 ([c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md)), part 2 of 5.
**Requirements:** PRC-01, EPI-01.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S18 verified.

**Ask Rahul first:**
1. **Who manages the trust's procedure list?**
   - (a) Admins.
   - (b) Super clinicians and admins.

   **Recommended: (b),** because the list is clinical content, like the questionnaires super clinicians already manage.
2. **Hospital-only procedures?** Use the same pattern as questionnaires: `hospital_id` NULL means every hospital of the trust. **Recommended: yes.**
3. **Information content (INF-04) and the PROMs plan (PRM-01)** are in PRC-01, but they're used only from C5 and C9. **Recommended:** add those columns in C5 and C9, not now.
4. **Today's free-text `procedure` on episodes:**
   - (a) Replace it with `procedure_id` (optional). The migration makes each trust's distinct texts into procedures and links the episodes to them.
   - (b) Keep the text as well, as a "description".

   **Recommended: (a).** It means one source of truth.
5. **Removing a procedure** that episodes use: **recommended,** a **Retired** flag. A retired procedure is hidden from new episodes but stays on old ones; nothing is deleted.
6. **The default HQ:** the procedure's default comes first. If it has none, use the hospital's `defaultHq` setting (S10). The clinician can always change it. **Recommended: yes.**

## Build

- **`procedures`:** org_id, `hospital_id NULL`, name, code (optional), specialty, default_version_id (a published HQ of the trust), and retired, as answered.
  - Unique by name within a trust.
- **Routes:**
  - `GET/POST/PATCH /api/o/{oid}/procedures`.
  - Each gets the five integration cases. The tenant case: Trust B can't read or change Trust A's list. A hospital-only procedure isn't offered at the trust's other hospitals.
  - Changes are audited (S11) like the other admin changes.
- **A Procedures screen** in the trust's admin area: list, add, edit, retire.
- **Episodes:**
  - `procedure_id` and the optional `planned_date` (planned procedure date) and `poa_date` (POA appointment date). The admitted and discharged dates come in S20.
  - **The New episode page:** pick a procedure, and its default HQ is chosen. The HQ can still be changed before Create.
  - **The episode page:** the procedure and both dates can be edited until the review is complete. The HQ can't change once the patient has started, as today.
- **The procedure and its dates are shown** in the episode list, on the POA Summary, and on the patient's home (the procedure name only).
- **The seed** adds a few made-up procedures per trust, with default HQs.

## Not in this session

- Information content and the PROMs plan (C5, C9).
- Worklists that use the planned date (S21).

## Manual test focus

1. As Alex Author: add procedures (with and without a code), including one only for Hospital A1. Edit one. Retire one.
2. As Val Clinician (Hospital A2): the A1-only procedure isn't offered. As Bea Author (Trust B): Trust A's list isn't visible.
3. As Cara Clinician: on a new episode, picking a procedure selects its default HQ. Pick another HQ by hand, then create.
4. A procedure with no default HQ falls back to the hospital's default HQ.
5. Planned and POA dates: set them on create and edit them later. They show in the list and on the POA Summary, in the hospital's date format.
6. A retired procedure isn't offered, but an episode that already has it still shows it.
7. Invalid input: a duplicate name, a default HQ from another trust, or a draft HQ gives a clear message.
8. As the patient: the home shows the procedure's name.
