# S13: Patient records

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 1 of 4.
**Requirements:** PAT-01, PAT-01a, PAT-01b, PAT-02.
**Depends on:** S12 verified (or S11, if Rahul chose the features-first order).

**Ask Rahul first:**
1. **Use `national_id` + `national_id_type` instead of an `nhs_number` column** (A-20, *Proposed*)? Only the NHS number's validator is built now. **Recommended: yes.**
2. **The duplicate warning** without an NHS number (same name and date of birth). *Rewritten 11 Oct for decisions.md "finding the patient":*
   - **A match among this hospital's patients:** show it, and offer "use this record" or "create anyway".
   - **A match only at another hospital of the trust:** nothing about it may be shown, and without an NHS number it can't be linked safely. Options:
     - (a) warn "this patient may already be registered in the trust; ask for their NHS number", and allow "create anyway";
     - (b) refuse until an NHS number is given.

     **Recommended: (a).**
3. **Confirm the search details** in C2's "New episode page" design: what counts as this hospital's patient (at least one episode here), the search terms, and the 20-result limit.

## Build

- **`patients`, per trust:**
  - `org_id`;
  - `account_id NULL` (used in S15);
  - the national ID (or NHS number);
  - name, date of birth and sex;
  - mobile (E.164) and email.
  - It's unique per trust when the ID is set; a duplicate gives **409**.
- **The NHS number Modulus 11 check,** written test-first.
- **Search this hospital's patients only** (Rahul, 11 Oct): server-side, as staff type. It needs a precise term (NHS number, hospital number, or surname with date of birth) and returns at most 20 results. Details: C2's "New episode page".
- **The duplicate warning** (PAT-01a).
- **Episodes hang off the patient record,** at any of the trust's hospitals:
  - `hospital_number` is a label only;
  - a patient can have more than one open episode.
  - Today's hospital-level patients are reshaped (a fresh app, A-1).
- **Staff screens:**
  - **The New episode page** (`/h/:hid/episodes/new`) replaces the drawer, with no Existing/New patient tabs.
  - **Silent linking:** when staff type a patient's details and they match a trust record by NHS number and date of birth, the episode is linked to it. Nothing about that record is shown. A mismatch → 409 with no stored values (Rahul, 11 Oct).
  - **A patient page** shows their episodes **at this hospital only.**
- **J3 and the other journeys** create patients the new way.

## Not in this session

- invites and sign-in (S14);
- accounts (S15).

## Manual test focus

1. Create a patient with a valid NHS number.
2. **An invalid check digit** is refused, with a clear message.
3. A duplicate in the same trust is refused, but the same NHS number in trust B is allowed.
4. No NHS number → the duplicate warning appears.
5. Search by each field. A patient known only at another hospital of the trust is **not** found.
6. Episodes at two hospitals:
   - at Hospital A, creating an episode for Hospital B's patient (typing their NHS number and date of birth) links silently, with nothing about Hospital B shown;
   - each hospital sees only its own episodes.
7. A mismatched NHS number and date of birth is refused, without showing the stored values.
8. The hospital number is shown on the episode.
