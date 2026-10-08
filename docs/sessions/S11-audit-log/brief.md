# S11: Audit log

**Plan:** C1 ([c1-trusts-staff.md](../../plans/c1-trusts-staff.md)), part 4 of 5.
**Requirements:** STF-06.
**Depends on:** S10 verified.

**Ask Rahul first:**
1. **What gets a row?** **Recommended:** one row per request that reads or changes patient data (list screens included), not one per record. That gives who, what (route and record IDs), when, trust and hospital.
2. **How long to keep it?** **Recommended:** for as long as the trust's retention period. Nothing is deleted for now.

## Build

- **`audit_log`, append-only.** A database rule or trigger refuses UPDATE and DELETE.
- **Go middleware writes the rows.** No patient values go in, only IDs.
- **`GET /api/o/{oid}/audit`,** with filters (user, date and patient) and a **CSV export**. Admins only.
- **The Audit screen,** for trust admins.
- **Integration tests:**
  - rows are written for patient-data reads and changes;
  - another trust's admin sees none of them;
  - UPDATE and DELETE fail.

## Manual test focus

1. As a clinician, open patients and episodes and change an answer. Then, as the admin, find those actions in the audit list.
2. Filter by user and by date.
3. Export the CSV, open it, and check that it matches the list.
4. A clinician can't see the Audit screen.
5. Trust B's admin sees nothing of trust A.
