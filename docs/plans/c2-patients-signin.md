# C2: Patients and sign-in

**Goal:** real patients. A clinician creates a patient record (identified by NHS number) and an episode. The patient gets an SMS and email with a magic link, confirms their date of birth, and sees every episode from every trust on one home page. Reminders go out on schedule.

**Requirements:** PAT-01…11, PX-08, PX-11, A-3, A-4, A-13. **Depends on:** C1.

**Not in C2:** pre-fill (C4), procedures and the other statuses (C3).

## Data

- **Control plane (London):**
  - `patient_accounts`: id, mobile, email;
  - `account_records`: account_id, org_id. A pointer only, with no clinical data.
- **Data cell (per trust):**
  - `patients`: org_id, account_id NULL, nhs_number NULL, name, dob, sex, mobile, email. Unique `(org_id, nhs_number)` where it's set.
  - `episodes.hospital_number` (a label).
  - `login_links`: token_hash, kind invite|signin, patient or account, expires_at, used_at, failed_dob_attempts.
  - `jobs`: run_at, kind, payload, done_at, attempts.
  - `sent_messages`: QA capture mode only.

## Design

- **Sign-in (A-4):**
  - The link opens `/i/:token` (invite) or `/s/:token` (sign-in). The page has a **Continue** button that sends a POST, so a link scanner opening the page can't use the link up. Then comes the date of birth.
  - The session is a signed, HTTP-only cookie with a 30-minute idle timeout.
  - Invite links work until the episode closes. Sign-in links work once, within 15 minutes. Five wrong dates of birth lock the link.
  - **Linking:** on the first successful invite, `patients.account_id` is set, matching an existing account by mobile or email.
- **Messaging:** `Deps.SMS` (AWS End User Messaging SMS) and `Deps.Email` (SES v2), with recording fakes in tests.
  - **QA uses `MESSAGING=capture`:** messages go into `sent_messages`, and a QA-only `GET /api/qa/messages?to=` lets the end-to-end tests read the link.
  - CDK fails the synth if capture mode is set on any other stage.
- **Jobs:**
  - The **jobs** Lambda, run every 5 minutes by EventBridge Scheduler (CDK). It loops over data regions; today that's only London.
  - Job kinds: send the invite, a reminder on day 2 and day 5, and flagging "not responding" on day 7.
  - **Testing jobs:** each job's *decision* is a pure function (`due(episode, now) → actions`) with unit tests. Its database writes reuse store functions already covered by the API integration tests. This follows the README's integration-test rule.
- **NHS number:** the Modulus 11 check (written TDD). Without an NHS number, staff are warned about possible duplicates with the same name and date of birth.
- **Staff completing for the patient (PAT-09):** a clinician opens the patient's forms from the episode. The answers are stored as the patient's, with `entered_by`.
- **Patient URLs:** `/p/:token` is replaced by `/i/:token`, `/s/:token`, `/home` and `/home/episodes/:id`.

## API

- **Staff:**
  - `GET/POST /api/o/{oid}/patients` (search by NHS number, name, date of birth)
  - `POST …/episodes` (sends the invite)
  - `POST …/episodes/{eid}/resend-invite`
- **Patient:**
  - `POST /api/p/links/{token}/continue`
  - `POST /api/p/links/{token}/dob`
  - `POST /api/p/signin` (mobile or email → sends a link)
  - `GET /api/p/home`
  - `GET/PUT …/forms/{fid}/answers/{cid}`
  - `POST …/submit`
  - `POST /api/p/logout`

## Tests

- **Integration, beyond the five:**
  - a bad or expired or used token → 404;
  - the 5th wrong date of birth locks the link;
  - the link alone returns no data;
  - linking joins an existing account;
  - the home page lists both trusts, and nothing from trusts not linked;
  - sign-in requests are rate-limited;
  - an SMS to a non-UK number is refused;
  - a duplicate NHS number in a trust → 409;
  - the invite is recorded by the fake SMS and email.
- **Unit:** the NHS number check, token hashing and expiry, the job decisions, and the session cookie.
- **End-to-end:**
  - **J7** (phone size): a clinician creates a patient and episode; read the captured SMS; open the link, Continue, date of birth, home, fill in and submit.
  - **J7b:** a wrong date of birth 5 times locks the link.
  - **J7c:** a returning patient signs in by email link.
  - **J3** is updated to the new URLs.

## Before Training and Production (start the paperwork early, because it takes days)

- SES production access, and a verified sending domain.
- UK SMS sender ID registration in AWS End User Messaging.

## Status

**The sign-in part was done 6 Oct 2026 as [auth first](auth-first.md) Step 5** (not committed yet): the link + Continue + date of birth, the lock after 5 wrong dates, patient sessions in the shared `sessions` table, and links stored hashed and sealed (not only hashed: clinicians can see them again). The routes are `/api/p/links/{token}/…` and `/api/p/hq…`, and the page stays `/p/:token`.

Not done yet: patient accounts, NHS number, sign-in links by mobile or email, the patient home, SMS/email, jobs and reminders.
