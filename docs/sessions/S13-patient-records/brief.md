# S13: Patient records

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 1 of 4.
**Requirements:** PAT-01, PAT-01a, PAT-01b, PAT-02.
**Depends on:** S12 verified (or S11, if Rahul chose the features-first order).

**Ask Rahul first:**
1. **Use `national_id` + `national_id_type` instead of an `nhs_number` column** (A-20, *Proposed*)? Only the NHS number's validator is built now. **Recommended: yes.**
2. **The duplicate warning** without an NHS number: show possible matches (same name and date of birth) and let staff choose "use this record" or "create anyway". **Recommended: yes.**

## Build

- **`patients`, per trust:**
  - `org_id`;
  - `account_id NULL` (used in S15);
  - the national ID (or NHS number);
  - name, date of birth and sex;
  - mobile (E.164) and email.
  - It's unique per trust when the ID is set; a duplicate gives **409**.
- **The NHS number Modulus 11 check,** written test-first.
- **Search** by NHS number, name and date of birth.
- **The duplicate warning** (PAT-01a).
- **Episodes hang off the patient record,** at any of the trust's hospitals:
  - `hospital_number` is a label only;
  - a patient can have more than one open episode.
  - Today's hospital-level patients are reshaped (a fresh app, A-1).
- **Staff screens:** patient search and create, and a patient page showing their episodes. A hospital-scoped user sees only their hospital's episodes (S08's answer).
- **J3 and the other journeys** create patients the new way.

## Not in this session

- invites and sign-in (S14);
- accounts (S15).

## Manual test focus

1. Create a patient with a valid NHS number.
2. **An invalid check digit** is refused, with a clear message.
3. A duplicate in the same trust is refused, but the same NHS number in trust B is allowed.
4. No NHS number → the duplicate warning appears.
5. Search by each field.
6. Episodes at two hospitals; the hospital-scoped view.
7. The hospital number is shown on the episode.
