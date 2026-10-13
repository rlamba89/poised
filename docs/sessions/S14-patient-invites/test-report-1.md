# S14 test report 1: patient invites and magic-link sign-in

**Summary: 19 passed, 0 failed, 0 blocked** (14 cases + 5 regression).

**Environment:**
- HEAD `14981f1`, with S14's changes uncommitted in the working tree. The tree also has planning-doc changes that aren't part of S14.
- 13 Oct 2026, 12:22–12:58 BST.
- http://localhost:3000 and http://tester.localhost:3000, served by the conductor's `make dev` with MESSAGING=capture.
- Claude in Chrome, automatic mode.

**How the phone size was done:** the browser window wouldn't go below about 507 px. Every patient step therefore ran in a **390 × 844 iframe**, served from the same origin as the page under test. The one exception is T-06's reload and Back steps. They needed real top-level history, so they ran in a full-width tab.

**Round 1 data:** surnames `… R1`, mobiles 07700 900101–105 and emails `s14.r1.a`–`e@example.org`.

**Where I used tools other than the UI:**
- I read some captured texts and emails with `curl /api/qa/messages?to=…`. This is the same data as the captured-messages page, which I also checked in the UI for T-01.
- The security probes in T-03, T-10 and "Found while exploring" used curl or `fetch`.
- I used the database read-only to look up the Nurse checks chapter id and to confirm that Settings were restored.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Create → text + email | Pass | Notes say "Episode created" and then "Invite sent by text message and email." Text wording is exact. The email subject and "Hello Sam," are right, and its link matches the text's. The card has no Copy box. |
| T-02 Continue, 4 wrong, right, fill, send (phone) | Pass | The first screen is generic. After Continue, only "Hospital A1" shows. Tries left went 4 → 3 → 2 → 1. The page moved to `/home/episodes/…`. The episode JSON has no DOB, mobile, email, NHS number or Nurse checks (checked with `fetch`). "Thank you, Sam" showed, the status became Ready for review and the card disappeared. scrollWidth was 388. |
| T-03 5 wrong → locked | Pass | The 5th wrong date shows "This link is locked". A right date sent by API afterwards also gets 403 locked. Reopening the link and pressing Continue shows "locked" straight away. Staff see the red box with "5 times (…12:27)". The note says "Invite sent by text message." |
| T-04 Resend | Pass | The green "Invite sent again by text message." appears, the red box goes, and a new note is added. The second text has a different link. The old link shows "This link doesn't work"; the new one opens the questionnaire. |
| T-05 Fresh browser | Pass | `/home/episodes/…` on tester.localhost shows "You've been signed out". The link shows nothing before Continue, and only the hospital's name after it. |
| T-06 Reload / Back (phone) | Pass | Reload on the date-of-birth form returns to the Continue screen. Reload inside the questionnaire stays signed in. About you shows In progress and keeps the answer. Back leaves to the page before the link and never shows the date-of-birth form. Sign out shows "You've signed out…", and a reload still shows signed out. |
| T-07 Staff fill in on behalf | Pass | The banner, "Sent for Sam Phone R1" and "Sent by Cara Clinician on the patient's behalf" all show. Both sets say "Patient's answers entered by Cara Clinician". The note "HQ completed on the patient's behalf" is there, as are the orange Validate box and "Patient (entered by Cara Clinician) on …" on the POA Patient answers tab. |
| T-08 Mixed patient/staff | Pass | Only Your health is marked "entered by". The patient reopening the link sees "Thank you, Sam". |
| T-09 Who gets which message | Pass | Email only: email sent, no text. +1 202 number: the note has the PAT-11 wording and no text went to +12025550123. No contacts: the note says "No invite sent…", Send invite is greyed out, and Fill in still works. |
| T-10 Refused access | Pass | Val (A2) and Bea (Trust B) both get "You don't have access to this hospital", and the API returns 404 with either hospital id. Signed out goes to `/login`. Another patient's episode shows "Episode not found." `/p/anything` is a 404 page. |
| T-11 Hospital invite template | Pass | The text read "Hospital A1 test R1: http://localhost:3000/i/…". The trust default is PAT-03's wording. A1 was restored to follow the trust (database `settings` is `{}`). |
| T-12 Audit | Pass | Patient rows: opened link, DOB (401 ×4 and 403 for the lock on Locked R1), opened questionnaire, saved and sent. Cara has resend, opened-to-fill, saved-for and sent-for rows, each naming the patient. 0 of 500 patient rows contain any link code; the route is stored as `/api/p/links/{token}/…`. |
| T-13 Phone layout | Pass | First screen, DOB form with the red message, locked, doesn't work, list, section, the "Send your answers?" dialog, Thank you and Signed out were all checked. scrollWidth was 388 each time and nothing was wider than 390. The date boxes fit on one row and the top bar doesn't overlap. |
| T-14 Suspended trust | Pass | While suspended, Continue shows "Not available at the moment". An already-open session and the DOB step both get 403. After re-activating: "Hospital B1 needs your date of birth…". |
| R-01 Search and second episode | Pass | "Invite R1" found, chosen and created. The new episode's note is "Invite sent by text message and email." |
| R-02 Validate and complete review | Pass | Smoking changed to Yes (with "Patient answered: No" shown). All 3 sets validated, and the status became Ready for POA. |
| R-03 Audit patient filter | Pass | `?patient=Invite+R1` shows only Sam Invite R1 rows, plus list rows that include that patient. |
| R-04 Hospital switch | Pass | A1 → A2 lists A2's episodes. |
| R-05 Settings save | Pass | A2's phone set to `01632 960123`, saved and restored (`settings` is `{}`). |

## Failures

None.

## Found while exploring

Security first. None of these is a failed case. They're listed for Rahul to decide on.

1. **Signing out, a resend or a lock doesn't end a session already open on a phone.** Minor; security hardening. The patient session is a stateless JWT, as designed in build notes "no server state". Two consequences:
   - After **Sign out** (`POST /api/p/logout` → 204, cookie cleared), a copy of the old cookie value still worked: `GET /api/p/episodes/<own>` returned **200**.
   - A **Resend** or a **lock** ends the *link*, but not a session someone already opened with it. If a link went to the wrong phone and the date of birth was guessed, Resend doesn't remove that person. They keep access while active, because every request issues the token again.

   The cookie is HttpOnly and SameSite=Strict, so this needs the cookie value to be stolen. Options:
   - a per-patient "sessions valid after" timestamp, checked in `requirePatient`;
   - bind the session to the link and check `revoked_at` and `locked_at`.
2. **What the HQ response contains after the date of birth.** For Rahul to confirm, not a bug: `patient: {firstName, name, age, sex}`.
   - **Full name (with surname) and age** are returned. The builder chose this on purpose (build notes, "Decided by the builder") for the survey's `{patientName}`/`{patientAge}`. The page itself shows only "Hello Sam.".
   - Rahul's rule is "no DOB or identifiers the patient didn't type", so he may want the surname, and maybe the age, sent only when the questionnaire uses them.
   - The session JWT payload also holds the patient record id (`sub`). The page's JavaScript can't read it (HttpOnly), but anyone holding the cookie can decode it.
3. **No `Cache-Control: no-store` on patient data responses** (`GET /api/p/episodes/{eid}`), and **no `Referrer-Policy`** on the `/i/<token>` page. Advisory, for the hosting step.
   - On a shared phone, the browser may cache the HQ JSON.
   - If a questionnaire ever loads an outside image or link while the address bar holds `/i/<token>`, the token could leak in the Referer header.

   Today the page loads no outside resources, and the address changes to `/home/episodes/…` after the right date.
4. **The server accepts Send with no answers.** For awareness; probably by design. While probing, I sent `POST /api/p/episodes/<Locked R1>/submit` straight to the API with nothing filled in. It returned **204**, and the episode is now Ready for review with empty answers.
   - The web app blocks this, because "Send my answers" stays disabled until every section is complete.
   - The API can't check required answers because Go doesn't evaluate SurveyJS. C6's `evaluate` Lambda could.
5. **"Entered by" stays on a set when the patient changes it afterwards.** By design (build notes). Example: Cara saved Reload R1's smoking answer, and the patient then changed the alcohol answer and sent. **About you** still says "Patient's answers entered by Cara Clinician", although part of the set is the patient's own.
6. **Things that worked well under probing:**
   - Ten parallel wrong dates of birth on a fresh link gave exactly 4 × 401 and 6 × 403. The count stopped at 5 and the link locked; the right date afterwards got 403.
   - Impossible dates (31/2/1980, a 2-digit year) are refused in the page with a clear message and don't use up a try. Malformed dates sent by API get 400 and don't count either.
   - Tampered tokens are refused with 401: `alg:none`, a wrong signature, or a patient token sent as the staff cookie (`/api/me`).
   - A patient session can't save to the clinician-only Nurse checks chapter (404), or to another patient's episode (404 for GET, PUT and submit).
   - The staff episode JSON has no token or hash, and `login_links` stores only `token_hash`.
   - Staff and patient editing the same set at once: the patient's stale save got 409, with a clear message ("Your answers weren't saved… Reload") and a working Reload button.
   - Double-clicking "Send my answers" in the dialog sent only one submit.
7. **"Episode not found." page for a patient opening someone else's episode.** Cosmetic. The page shows only the red message, with no Poised bar or Sign out.
8. **Captured messages hold the full link in plain text** (`sent_messages`, `/qa/messages`). Expected in capture mode, which is local only. Noted so S17 makes sure capture can never run in a real deployment.

**Tooling note, not an app bug:** Claude in Chrome often dropped the first click after a page load. In those cases a `click` listener on the page received no event at all. Real clicks weren't affected.

## Console and network errors

- No console errors or warnings on the patient pages (wrong date and signed-out flows) or on the staff episode page.
- Network errors were only the expected ones:
  - 401 after a wrong date of birth or when signed out;
  - 403 for locked or suspended;
  - 404 for a dead link or another patient's episode;
  - 409 for a stale save;
  - 204 responses shown as 503 by the extension (a known tool quirk).

## Recordings

- `~/Downloads/S14-T02-happy-path.gif`: T-02 at phone size. It shows the link, Continue, 4 wrong dates, the right date, filling both sections and Send my answers → "Thank you, Sam".
- No failures to record.

STATUS: ALL_PASSED
