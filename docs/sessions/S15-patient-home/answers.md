# S15: answers

Asked by the conductor (`/run-sessions`) on 12 Oct 2026, before the run started.

1. **Patients updating their own contact details (PAT-10, a Should).**
   - **Answer: now, in S15.** Patients can change their email or mobile from patient home. A new contact is verified before it's used. (The recommendation had been later.) This **widens the brief**; Rahul chose it.
2. **Rate limits on sign-in links.**
   - **Answer (recommended):** at most 3 sign-in links per mobile or email in 15 minutes, and 10 per hour per IP address.

## Asked during the run (13 Oct, by the conductor, during test round 1)

3. **Account matching:** a trust's record joins an account only when the date of birth AND the mobile or email match (people sharing a phone or email stay separate).
   - **Answer: DOB + contact, as built** (recommended).
4. **Sign-in is for returning patients only** (an account exists only once someone proved their DOB on an invite).
   - **Answer: returning only, as built** (recommended).
5. **Contact changes update every trust's copy (the old contact is told); verification links last 24 hours.**
   - **Answer: both as built** (recommended).
6. **Twins** (found by test report 1, Major privacy): two people with the same date of birth and the same contact (e.g. twins on a parent's phone) were merged into one account.
   - **Answer: also match the first name** (recommended). A record joins an account only when DOB + contact + first name (ignoring case and spaces) match. A different spelling just stays unjoined and uses its own invite links (fails safe). Also label each questionnaire card with the patient's name.
7. **The smaller gaps from test report 1.**
   - **Answer: fix all five:** (1) adding a contact where there was none must notify the existing contact; (2) the email box must refuse a mobile and the mobile box an email; (3) a record that joins later gets the account's confirmed contact; (4) write an audit row when a record joins an existing account; (5) tighten email validation (reject emoji and over-long addresses).
8. **Same first name, DOB and contact, different surname** (found by test report 2, Major privacy): two such patients became one account.
   - **Answer: require the surname too** (recommended). A record joins an account only when DOB + contact + first name + surname (ignoring case and spaces) match. A surname change or typo just keeps a separate account (fails safe). This supersedes item 6's first-name-only rule.
9. **Report 2's small wording issues** (greeting casing, "keep using your current mobile" when adding one, the first-name step's wording and focus, a misleading "date of birth doesn't match" after a contact change, a 300-character first name's generic error).
   - **Answer: fix them in the same round** (recommended).
10. **Verification 1, finding 1** (Major privacy): someone who shares a contact and date of birth with an existing account holder, but hasn't opened their own invite yet (e.g. an unjoined twin), signs in to the other person's account, because the name step only runs when two or more accounts match.
    - **Answer: accept it; don't change sign-in** (Rahul, 13 Oct): "the twin use case is a very rare scenario; let's not mess up other things because of a 0.00001% chance." Record it as an accepted risk in `decisions.md` and `pending.md`. It needs the same date of birth and access to the shared phone or inbox.
