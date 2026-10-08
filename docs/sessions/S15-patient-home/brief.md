# S15: Patient accounts across trusts, sign-in links and the patient home

**Plan:** C2 ([c2-patients-signin.md](../../plans/c2-patients-signin.md)), part 3 of 4.
**Requirements:** PAT-06, PAT-07, PAT-11, PX-08, A-3.
**Depends on:** S14 verified.

**Ask Rahul first:**
1. **Patients updating their contact details (PAT-10, a Should):** now or later? **Recommended: later.**
2. **Rate limits:** **recommended** 3 sign-in links per number or email per 15 minutes, and 10 per hour per IP address.

## Build

- **`patient_accounts`** (mobile in E.164, email).
- **Linking:** the first successful invite sets `patients.account_id`, matching an existing account by mobile or email (PAT-07).
- **`POST /api/p/signin`** (mobile or email) sends a **sign-in link**, `/s/:token`:
  - it works once, for 15 minutes;
  - then Continue → date of birth.
- **`GET /api/p/home`:** every open episode from **every trust**, newest first, with the hospital name. One query.
- **Logout.**
- **Rate limiting** (PAT-11). SMS goes only to the deployment's phone country, taken from config.
- **E2E:** J7c.

## Manual test focus

At **phone size**:
1. The same person (the same mobile) gets episodes in trust A and trust B → one home lists both.
2. **A returning patient** asks for a link by email and by SMS (read it from captured messages) and signs in.
3. A sign-in link used twice → refused. An expired link → refused.
4. The rate limit kicks in, with a friendly message.
5. A non-UK number is refused.
6. Log out.
7. **Another patient's episode URL shows nothing.**
