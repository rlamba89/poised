# S13: answers

Asked by the conductor (`/run-sessions`) on 12 Oct 2026, before the run started.

1. **`national_id` + `national_id_type` instead of an `nhs_number` column** (proposed under A-20)?
   - **Answer: yes, the generic `national_id`** (recommended). Only the NHS-number validator is built now.
2. **The duplicate check without an NHS number.**
   - Rahul's note: in Lifebox, **email** has been what helps here; name + date of birth is the identifier in practice, and the NHS number is often empty. He asked for a proposal.
   - **Answer (the conductor's proposal, chosen):** on Create, the server compares the new patient with the trust's records. Names are compared ignoring case and spaces, email in lower case, mobile in international format.
     - **Same first name, surname and date of birth, AND the same email or mobile** → link silently to that trust record, as with a matching NHS number. Nothing about that record is shown.
     - **Same name and date of birth, but a different or missing email/mobile** → warn "this patient may already be registered in the trust; check their NHS number or contact details", and allow **Create anyway**. Nothing about the other record is shown.
     - The earlier rules still apply: a match among this hospital's patients is shown, with "use this record" or "create anyway"; a matching NHS number + date of birth links silently; an NHS-number mismatch gets a 409.
   - Rejected: linking on name + date of birth alone (two people in a large trust can share both, and their records would merge); never linking without an NHS number (duplicates would be common).
3. **The patient search** on the New episode page.
   - **Answer: confirmed as in C2** (recommended). Only this hospital's patients (at least one episode here). Terms: a valid NHS number (exact), a hospital number (exact), surname + date of birth, or at least 3 letters of a surname. At most 20 results (name, date of birth, NHS number).

## Asked during the run (13 Oct, by the conductor, during test round 1)

4. **The new Go library `github.com/nyaruka/phonenumbers`** (the Go port of Google's libphonenumber), for E.164 mobiles in any country.
   - **Answer: keep it** (recommended). It keeps a new country config-only (A-20); hand-written parsing would end up UK-only.
5. **On a silent link, do typed details overwrite the trust's stored record?**
   - **Answer: no, keep the stored record** (recommended), as built. The pages then show the trust's stored name and contacts. Updating contacts is a separate, deliberate action.
