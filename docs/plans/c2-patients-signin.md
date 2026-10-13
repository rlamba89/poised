# C2: Patients and sign-in

**Goal:** real patients. A clinician creates a patient record (identified by NHS number) and an episode. The patient gets an SMS and email with a magic link, confirms their date of birth, and sees every episode from every trust on one home page. Reminders go out on schedule.

**Requirements:** PAT-01…11, PX-08, PX-11, A-3, A-4, A-13, A-20. **Local first (12 Oct, A-22):**
- **S13–S16 run locally.** Messages are only captured, and a timer in the local server calls `jobs.RunDue`.
- **S17, in the AWS phase:** the jobs Lambda, the EventBridge schedule, and real SES/SMS.

**Depends on:** C1.

**Not in C2:** pre-fill (C4), procedures and the other statuses (C3).

## Data

All of these are in the deployment's one database. *(On 8 Oct the London control-plane / data-cell split and the `account_records` table were dropped: A-20.)*

- `patient_accounts`: id, mobile (E.164), email.
- `patients`: org_id, account_id NULL, nhs_number NULL, name, dob, sex, mobile, email. Unique `(org_id, nhs_number)` where it's set.
  - *Proposed (A-20):* `national_id` + `national_id_type` instead of `nhs_number`, so another country's identifier needs only a new validator. Only the NHS number is built.
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
  - The **jobs** Lambda, run every 5 minutes by EventBridge Scheduler (CDK). It serves its own deployment only; each country has its own schedule (A-20).
  - Job kinds: send the invite, a reminder on day 2 and day 5, and flagging "not responding" on day 7.
  - **Testing jobs:** each job's *decision* is a pure function (`due(episode, now) → actions`) with unit tests. Its database writes reuse store functions already covered by the API integration tests. This follows the README's integration-test rule.
- **NHS number:** the Modulus 11 check (written TDD). Without an NHS number, staff are warned about possible duplicates with the same name and date of birth. The warning shows none of the other record's details (decisions.md, 11 Oct).
- **The New episode page** (Rahul, 11 Oct; decisions.md). It replaces today's drawer and its Existing/New patient tabs.
  - **The page:** a dedicated page, `/h/:hid/episodes/new`. It has a patient search, then one form: patient details, then episode details.
  - **Search is for this hospital's patients only:** patients with at least one episode at this hospital. Creating a patient always creates an episode here.
    - **Never shown:** a patient known only at another hospital of the trust, and anything from another trust.
  - **Built for millions of records:**
    - **no dropdown:** the server searches as staff type, and returns at most 20 results (name, date of birth, NHS number);
    - **a precise term is needed:** a valid NHS number (exact), a hospital number (exact), a surname with a date of birth, or at least 3 letters of a surname. Nobody can list everyone;
    - **indexed:** on the identifier, and on surname with date of birth. Scoped to the hospital in the query itself.
  - **Not found:** staff type the patient's details from the referral. On Create, the server links **silently** to the trust's existing record when the NHS number and date of birth agree.
    - **Nothing is shown about that record:** not its hospitals, episodes or stored values.
    - **A mismatch** (same NHS number, different date of birth or surname) gets a 409 that says the details don't match. It shows no stored values, and nothing is created.
- **Staff completing for the patient (PAT-09):** a clinician opens the patient's forms from the episode. The answers are stored as the patient's, with `entered_by`.
- **Patient URLs:** `/p/:token` is replaced by `/i/:token`, `/s/:token`, `/home` and `/home/episodes/:id`.

## API

- **Staff:**
  - `GET /api/h/{hid}/patients?q=` (search this hospital's patients: NHS number, hospital number, or surname with date of birth; at most 20)
  - `POST /api/h/{hid}/episodes` with the patient's details or a found patient's id (creates the trust record when there's no match, links silently when there is). *Changed 11 Oct from a trust-wide `/api/o/{oid}/patients`: search is hospital-scoped.*
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
  - **patient search:**
    - doesn't return a patient known only at another hospital of the same trust, or anything from another trust;
    - refuses a term that's too broad;
    - returns at most 20 results;
  - **creating an episode for details that match another hospital's patient:**
    - it links to that record;
    - the response and the page show none of that record's data;
  - **a mismatch** (same NHS number, different date of birth) → 409, the message holds no stored values, and nothing is written;
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

- **S13 (patient records): built 13 Oct; tested 13 Oct (test report 3, all passed); verified 13 Oct ([verify.md](../sessions/S13-patient-records/verify.md)).** [Build notes](../sessions/S13-patient-records/build-notes.md). Differences from the plan:
  - `national_id` + `national_id_type` (agreed), the scheme set per deployment by `NATIONAL_ID_TYPE`; only `nhs` is built.
  - The search is `GET /api/o/{oid}/h/{hid}/patients?q=&dob=` (S08's URL shape), the date of birth its own parameter. A patient page, `GET …/patients/{pid}`, was added for the brief.
  - Rahul's duplicate rule (S13 answers 2): the same name, date of birth **and** email or mobile links silently; the same name and date of birth otherwise warns, with Create anyway.
  - `POST …/patients` is gone: a patient is created with their first episode.
- **S14 (patient invites and magic-link sign-in): built 13 Oct; tested 13 Oct (test report 1 all passed; Rahul's session hardening then built and retested in test report 2, all passed); verification 1's fixes built and retested 13 Oct, test report 3 all passed; verification 2's fix built and retested 13 Oct (test report 4 passed; its X-1, Back within 0.6 s of typing, skipped by Rahul as an edge case); **verified 13 Oct** ([verify.md](../sessions/S14-patient-invites/verify.md), verification 3).** [Build notes](../sessions/S14-patient-invites/build-notes.md). Differences from the plan:
  - The patient's form routes are `/api/p/episodes/{eid}` (+ `/answers/{cid}`, `/submit`), not `…/forms/{fid}/…`: an episode has one HQ until C3. Staff's on-behalf form is `…/episodes/{eid}/patient-form`.
  - Only `/home/episodes/:eid` of the patient's URLs is built (where the HQ opens); `/home` is S15. Linking to a patient account on the first invite is S15's too.
  - Patient sessions are held in the database (`patient_sessions`, S14 answers 4), tied to the invite link, so Sign out, a Resend or a lock ends them at once.
  - Sending happens in the request after the episode is saved; S16 moves it to jobs. A failed send is noted on the episode, never a failed Create.
- **S15 (patient accounts, sign-in links, the home): built 13 Oct; test report 1 (23 passed, 1 blocked: R-6 a hand check); test report 2 (27 passed); **tested 13 Oct (test report 3: all passed, R-6 a hand check for Rahul)**; verification 1 (13 Oct): **needs fixes** ([verify.md](../sessions/S15-patient-home/verify.md)); finding 2 fixed and retested 13 Oct (test report 4, all passed); finding 1 accepted by Rahul as a rare risk (answers 10); **verification 2 (13 Oct): verified**.** [Build notes](../sessions/S15-patient-home/build-notes.md). Differences from the plan:
  - `patient_accounts` also holds the **date of birth** and the **first name's and surname's keys**: a record joins the account with the same date of birth, the same mobile or email **and** the same first name and surname (S15 answers 6 and 8), so people sharing a phone or email, twins included, stay apart. A sign-in link goes to a contact; the date of birth picks the account.
  - At an invite's or a sign-in link's right date of birth, the person's other unjoined records with that date and a contact that just reached them join too, so a later trust's questionnaire appears on the home at once.
  - `POST /api/p/signin {contact}` (one field); the sign-in link's routes are `/api/p/signin/{token}/continue|dob`. **PAT-10 built now** (S15 answers 1): `POST /api/p/contact` sends a link to the new contact, `POST /api/p/contact/{token}/confirm` changes the account and every trust's record, and tells the old contact.
  - Patient sessions belong to the account, so they open every joined record's episodes, in every trust.
