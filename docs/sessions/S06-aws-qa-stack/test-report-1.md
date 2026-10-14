# S06 test report 1: the app on AWS (QA)

**19 passed, 0 failed, 0 blocked** (T-01 to T-13, R-1 to R-6). T-07's optional "1 problem to fix first" check was not run.

**Environment**
- **URL:** https://main.d33kh0gi8vk83v.amplifyapp.com (Amplify + API Gateway + Lambda + Neon). No localhost was used.
- **Code:** `git rev-parse --short HEAD` = `743fd55`. The working tree is **not clean**: S06's uncommitted changes (about 50 paths). The deployed build is whatever the conductor deployed from it.
- **When:** Wed 14 Oct 2026, 12:09 to 12:35 BST.
- **Browser:** Claude in Chrome, desktop window 1400×757. The window wouldn't go below 500 px wide, so the patient pages ran in a **390 px-wide same-origin iframe** (390×750, cut down from 844 to fit the viewport).
- **Test data (all made up):**
  - questionnaire `QA06 MT J1` (v1 published, v2 draft);
  - patients `Sam Qa06MTJ3` (+447700900611, qa06j3.mt@example.org), `Sam Qa06MTJ3b` (+447700900612) and `Sam Qa06MTR5` (+447700900615), all born 17/05/1980;
  - staff invite `qa06.mt@example.org` ("QA06 MT Clinician", clinician, Hospital A1).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Health and config | Pass | `{"status":"ok"}`; config is exactly as expected. The address stayed on QA. |
| T-02 Sign in and out | Pass | Login took Cara straight to Hospital A1's Episodes: `POST /api/dev/login` (503 = 204), then `/api/me` 200. `/api/me` in a new tab named Trust A / Hospital A1. Set-Cookie (checked with curl) is `sj_token=…; Path=/; Max-Age=43200; HttpOnly; Secure; SameSite=Lax`. After Sign out: the login page, `/api/me` 401 ("Please sign in."), and Back showed no data. |
| T-03 Deep links | Pass | Reload and fresh load both worked for the Episodes list, an episode (E2E J5), its POA Summary, Alex's editor and a Question Set Preview. No blank page, and the full address was kept. Signed out, the episode address went to `/login`, and the episode could be reached again after sign-in (see exploring note 6). |
| T-04 Not-found addresses | Pass | `/assets/does-not-exist.js`: 404 from S3/CloudFront with an empty body (Chrome shows its own error page). `/api/no-such-route`: `404 page not found`. `/no/such/page`: the app's "404 This page could not be found." |
| T-05 Client address | Pass | checkip = `90.200.59.251`; `/api/qa/client` gives `address` `90.200.59.251` and `forwardedFor` `90.200.59.251, 64.252.85.x, 3.11.170.31` (mine, then two Amazon addresses). Extra: a spoofed `X-Forwarded-For: 1.2.3.4` is ignored, and `address` stays my real IP. |
| T-06 J1 author and preview | Pass | Created `QA06 MT J1` with the Question Set "About you", Page 1 and the Yes/No "Do you smoke?". "All changes saved"; all of it was still there after a reload. The Preview shows the question with Yes/No. |
| T-07 J2 publish and new version | Pass | The Publish dialog ("Publish version 1?") had no problems listed. v1 became PUBLISHED / READ ONLY, and **Create new version** opened V2 · DRAFT with the same set, page and question. |
| T-08 J3 patient on phone | Pass | The episode notes say "Invite sent by text message and email." The captured SMS reads "Hospital A1: please complete your pre-op health questionnaire: https://main.d33kh0gi8vk83v.amplifyapp.com/i/…" (QA address). At 390 px: Continue, DOB 17 5 1980 (submitted with Enter), answered part-way, reloaded: still signed in, with "About you" IN PROGRESS and the answer kept. Finished both sets, then Send my answers, then the confirm modal, then "Thank you, Sam". scrollWidth 390 = clientWidth 390, so no sideways scroll. Staff side: READY FOR REVIEW. |
| T-09 J3 refused paths | Pass | Wrong DOB 1 1 1990: "That date of birth doesn't match our records… You have 4 tries left." and the questionnaire didn't open (401). `/i/not-a-real-token`, then Continue: "This link doesn't work. Please check it, or contact the hospital." (404). No stack trace. |
| T-10 J4 validate | Pass | Changed "Do you smoke?" from Yes to No, which shows "Patient answered: Yes". Validated all 3 sets (Nurse checks BP 120/80). Complete HQ review gives READY FOR POA. |
| T-11 J5 POA Summary | Pass | **Validated summary** shows "Non-smoker", "CHANGED BY CLINICIAN… Patient answered "Yes", now "No"". **Patient answers** shows "Smoker". Changing the status to On hold logged "Status changed to On hold". The comment "QA06 MT: on hold pending GP letter" appears in General notes. The Print button is there (not clicked). A reload changed nothing. |
| T-12 Wrong role / other trust | Pass | Val (A2 only) and Bea (Trust B) pasting the A1 episode address both see "You don't have access to this hospital". The network log shows only `/api/config` and `/api/me`, with no episode request. Direct API checks by both: `404 Hospital not found` / `404 Episode not found`. No patient details leaked. |
| T-13 Console / network | Pass | No console errors or warnings once tracking started. Every 4xx/5xx was expected: 503s on 204 routes (login, logout, submit, complete-review, PATCH status), 401 on `/api/me` signed out, 401 on the wrong DOB, and 404s in T-04, T-09 and T-12. |
| R-1 Staff list | Pass | Ada sees Trust A's 6 staff with their roles. The invite of `qa06.mt@example.org` as clinician at A1 shows "Invitation sent… expires on 21/10/2026" and the row reads INVITED · EXPIRES 21/10/2026. |
| R-2 Superadmin console | Pass | Sam's list: Trust A, Trust B and 5 "E2E …" trusts. Trust A shows Hospital A1 and A2. Nothing was changed (see exploring note 8). |
| R-3 Audit log | Pass | Ada's Audit lists T-08 to T-11 with Cara's name: "Created an episode", "Opened the answers", "Saved the clinician's answers", "Completed the HQ review", "Changed an episode", "Added a note". It also has the patient rows (opened link, entered DOB, saved, sent) and Val's REFUSED (404). |
| R-4 Patient records | Pass | New episode, then a search for `Qa06MT` finds Sam Qa06MTJ3 and Sam Qa06MTJ3b. Each patient page lists its episode (ON HOLD / HQ NOT COMPLETE). |
| R-5 Reminders by Run jobs | Pass | Now + 2 days, then Run jobs: "Ran 14 jobs as of 16/10/2026, 12:31: 13 reminders sent, 0 marked "not responding"". Captured messages for +447700900615 show the invite and "Hello Sam, a reminder from Hospital A1: please fill in your health questionnaire: https://main.d33kh0gi8vk83v.amplifyapp.com/i/…". |
| R-6 Returning patient sign-in | Pass | At 390 px: `/signin`, mobile `07700 900611`, then "Check your text messages". The captured SMS "Poised: your sign-in link… https://main.d33kh0gi8vk83v.amplifyapp.com/s/…" leads to Continue and DOB 17 5 1980. The home "Hello Sam" lists Hospital A1 · E2E J3 … · For Sam Qa06MTJ3 · **SENT**. |

## Failures

None.

## Found while exploring

These are not failures. Several are about QA being reachable by anyone on the internet, which the conductor asked to be raised for Rahul.

1. **Anyone on the internet can sign in as any staff user on QA (for Rahul).**
   - `GET /api/dev/users` (no sign-in) lists every account, including the superadmin and E2E staff, with emails. `POST /api/dev/login {"userId":…}` returns a session cookie for any of them; I checked with curl from outside the browser. The seeded IDs are predictable (`00000000-0000-4000-8000-0000000000a3` is Cara).
   - The plan puts the dev login out of scope until S12. It's still worth knowing that QA's patient data (all made up) and the superadmin console are effectively public until Cognito arrives. Consider an Amplify basic-auth password or an IP allow-list on QA until S12.
2. **The QA helper pages are public too (for Rahul).**
   - `/qa/messages` (and `GET /api/qa/messages?to=`) **lists every captured text and email when "To" is empty**, including live invite links (`/i/…`), sign-in links (`/s/…`) and contact-change links (`/v/…`). Opening any of them still needs the patient's DOB, but the 5-tries limit can be undone by the public **Reset sign-in limits** button.
   - `/qa/jobs` lets anyone run the jobs (send reminders, mark "not responding") or move the clock.
   - Known: pending.md §3 notes capture-mode links in plain text and the S17 follow-up. This note only adds that on AWS these pages are exposed to the internet.
3. **A staff invite can be "accepted" through the dev login.** After R-1, "QA06 MT Clinician" appeared in the dev login list. Clicking it (by accident, because the list shifted by one) signed me in as that clinician at Hospital A1, which counts as their first sign-in. This is the S09 design ("the dev login now"), so it's not a bug, but it widens point 1.
4. **No security headers on the web app.** The responses for `/` have no `Strict-Transport-Security`, `X-Frame-Options` / CSP `frame-ancestors` or `X-Content-Type-Options`, so the app can be framed by another site. HTTP does redirect to HTTPS (301). Amplify `customHeaders` would add them, a hardening item for before real users. API responses are not cached by CloudFront (`x-cache: Miss` / `Error` every time), which is good.
5. **The patient cookie is correct:** `poised_patient=…; Path=/api/p; Max-Age=1800; HttpOnly; Secure; SameSite=Strict`. I checked this with curl, and the DOB step also wrote an "entered their date of birth" audit row on Sam Qa06MTR5's episode.
6. **The login page doesn't return you to the deep link.** Signed out, pasting the episode address goes to `/login`. After signing in you land on the Episodes list rather than the episode you asked for. This is older behaviour, not S06.
7. **Reloading the patient's questionnaire mid-set goes back to the set list**, not the page you were on. The answers and session are kept, and "Continue" reopens the set. This is older behaviour.
8. **I opened a dropdown on the superadmin page by mistake** (Trust A, Features, PROMs for Hospital A1). I closed it with Escape and it still reads "Follow trust (Off)". Nothing was saved.
9. **The captured-messages "To" box has no accessible label of its own:** its accessible name is the placeholder text, not "To". This is cosmetic and older.
10. **Tool note, not the app:** clicks at the very top of the page (the header's Sign out, at y≈27) were often not delivered by the browser tool. A document-level listener saw no pointer events at all, and clicks lower on the page or by element reference worked. Sign out always worked when the click arrived.
11. **The patient sign-in rejects nonsense input:** emoji in the mobile/email box gives "Enter a real mobile number or email address."

## Console and network errors not tied to a case

None. Every 503 in the network log was on a 204 route (as the plan notes), and every 4xx was one a case expected.

## Hand checks for Rahul

None required. Optional: the phone cases ran in a 390 px iframe rather than a real 390×844 window. Everything fitted and there was no sideways scroll.

## GIFs

- Happy path (T-08 patient journey at 390 px: link, DOB, answer, reload, finish, send, thank you): `/Users/rahul/Downloads/S06-T08-happy-path.gif`.
- There are no failure GIFs because nothing failed.
