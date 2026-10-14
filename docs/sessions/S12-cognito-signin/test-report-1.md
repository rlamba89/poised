# S12 test report 1 — staff sign-in with Cognito and MFA, on QA

**Summary:** 15 passed, 0 failed, 3 blocked (hand checks). No Blocker or Major defects found.

**Environment**
- URL: https://main.d33kh0gi8vk83v.amplifyapp.com (QA, uk-qa, eu-west-2)
- Commit at test start: `ab90771` (working tree had the session's uncommitted changes, as per the run's git status; QA runs the deployed S12 working tree, not `master`)
- Date/time: 14 Oct 2026, ~17:04–17:45 BST
- Tester: automatic (manual-test, `/run-sessions`), Claude in Chrome + read-only AWS CLI + curl
- Cognito pool/domain observed: `poised-uk-qa-…amazoncognito.com` (`/api/config` clientId `5lroegtdtvfubefn41spl1gd9c`); API Gateway `https://y4bo1a1lb1.execute-api.eu-west-2.amazonaws.com`

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Sign-in page (no dev login) | Pass | Poised title, the expected help text, Sign in button, "temporary password / Forgot your password?" help. No user list, no "Development sign-in". `/api/dev/users` and `/api/dev/login` → 404. |
| T-02 Sign in with password + code | Pass | Cara lands on Hospital A1 Episodes, header Trust A / Hospital A1 / Clinician / Cara. Reload stays signed in; 2nd tab on same URL signed in (one session). No console errors. Cognito pages on the `amazoncognito.com` domain. |
| T-03 Wrong password, then wrong code | Pass | "Incorrect username or password." (no account-existence leak). Real password → MFA; `000000` → "Invalid or expired code entered. Try again."; real code → Val lands on Hospital A2 Episodes. |
| T-04 Sign out | Pass (1 minor, see exploring) | Ends on `/login` after a hop through Cognito. `/api/me` → `{"error":"Please sign in."}`. Clicking Sign in re-prompts for email **and** password (no auto-login). Back button caveat below. |
| T-05 Where each person lands | Pass | Max Multi → picker: Trust A (Hospital A2, Clinician) + Trust B (ADMIN, WHOLE TRUST; Hospital B1, Admin); picking B1 → Questionnaires. Sam → `/admin` console (trusts list). Alex → picker: Trust A (SUPER CLINICIAN, WHOLE TRUST; Hospital A1 & A2). |
| T-06 Testers' tools need a platform admin | Pass | Signed out: "Sign in as a platform admin to use this page." + Sign in link on `/qa/messages` and `/qa/jobs`; `/api/qa/messages` → 401. As Cara: "Only a platform admin can use this page." on both. As Sam: captured list shows; **Reset sign-in limits** → "Sign-in limits cleared: links can be asked for again."; `/qa/jobs` shows the timer and run form. |
| T-07 Suspended trust refused at sign-in | Pass | Suspended Trust B (in-app confirm modal). Bea → after code returns to `/login` with red "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust."; `/api/me` → 401. Re-activated Trust B ("Trust B is active again. Its staff can sign in."). **Trust B left ACTIVE.** |
| T-08 A removed membership | **Blocked** | Removing Val's Hospital A2 access was refused by the auto-mode safety classifier ("Modify Shared Resources"). Not retried. Hand check for Rahul. |
| T-09 An invite reaches Cognito | **Blocked** | Inviting a staff member creates a Cognito user and sends an email (shared-resource mutation, same class the classifier blocked in T-08; also spends the ~50/day email pool). Not attempted in auto mode. Hand check for Rahul (also overlaps H-1). |
| T-10 Patients still work (phone size) | Pass (phone-size caveat) | Cara created episode "MT S12 1714 Smith" (DOB 17 May 1980, email `mt.s12.1714.p@example.org`, newest published HQ); "Invite sent by email." in General notes. As Sam, captured messages filtered To that email → the invite with `/i/…` link. Patient link → "Your health questionnaire" → Continue → "Confirm it's you" (DOB 17/5/1980, Hospital A1) → questionnaire (sections "About you" + "Your health"; clinician "Nurse checks" section correctly hidden). Answered "Do you smoke? No" (Saved), then patient Sign out → "You've been signed out." See phone-size note below. |
| T-11 Security headers | Pass | `x-frame-options: DENY`, `content-security-policy: frame-ancestors 'none'`, `strict-transport-security: max-age=31536000; includeSubDomains`, `x-content-type-options: nosniff`, `referrer-policy: no-referrer` all present on `/login`, `/`, a `/assets/*.js` file (and `/api/health`). |
| T-12 Forgotten password page | Pass | Cognito reset page "Forgot your password? Enter your email address…". Not submitted. Back returns to Poised `/login` and the Sign in button works again (redirects to Cognito). |
| R-01 (S08) Hospital switch | Pass | Max switched Hospital B1 → Hospital A2 in header: role changed ADMIN → CLINICIAN, nav reduced to Questionnaires/Episodes; Trust A dropdown shows only A2. |
| R-02 (S09) Staff list | Pass | Ada's Staff lists Trust A people (Ada, Alex, Cara, Max, QA06, Sue, Val) with access-and-role lines. Clinicians have no Staff link (Cara/Max as clinician show no Staff nav). |
| R-03 (S10) Console | Pass | `/admin` lists Trust A and Trust B (among many E2E trusts). Trust A page shows its hospitals (A1, A2) and admin (Ada). |
| R-04 (S11) Audit | Pass (filter-narrow caveat) | Audit shows Cara Clinician's actions today, including "Created an episode — MT S12 1714 Smith" (17:15) and opening its answers (from T-10). See filter note below. |
| R-05 (S13/S14) Validate shows patient answer | **Blocked / not reached** | The T-10 patient only *saved* one answer and did not "Send my answers", so the HQ is not submitted and clinician validation isn't available yet; "Validate the set"/"Complete the HQ review" also mutates (would hit the classifier). Hand check for Rahul. |
| R-06 (S16) Scheduled jobs | Pass | As Sam, ran jobs as of Now + 2 days → "Ran 12 jobs as of 16/10/2026, 17:19: 12 reminders sent, 0 marked 'not responding'." Captured messages then showed a reminder to `mt.s12.1714.p@example.org` ("Hello MT S12 1714", 17:18). |

## Security probes (the conductor's priority) — all pass

Done unauthenticated via curl, in-browser, and against the API Gateway URL directly; plus a read-only AWS CLI check of the API's routes.

- **Old dev login is gone:** `POST /api/dev/login` → 404, `GET /api/dev/users` → 404 — both via the Amplify domain **and** against `https://y4bo1a1lb1.execute-api.eu-west-2.amazonaws.com` directly. `/api/config` reports `staffSignIn.devLogin: false`.
- **`/api/qa/*` require a session:** unauthenticated `GET /api/qa/messages`, `/api/qa/jobs`, `/api/qa/client`, `POST /api/qa/jobs/run`, `PUT /api/qa/jobs/clock`, `DELETE /api/qa/rate-limits` all → 401 `{"error":"Please sign in."}` (Amplify and API Gateway direct).
- **Protected data:** `/api/me`, `/api/admin/orgs` → 401 unauthenticated.
- **API Gateway URL directly** enforces the same auth; the patient Lambda rejects staff routes (`/api/me` → 404 on `/api/p/*` function). No open bypass found.
- **Forged session cookie** (`sj_token=` random JWT) → 401.
- **Suspended trust, wrong role, other tenant:** suspended trust refused at sign-in (T-07); role change on hospital switch (R-01); platform-admin-only tools refuse a clinician (T-06).
- **Sign-out ends the session:** reload of a staff page after sign-out → `/login`; `/api/me` → 401 (T-04, T-07).
- **Security headers** present everywhere checked (T-11).

## Found while exploring

1. **Minor — stale page visible via Back after staff sign-out (bfcache).** After signing out, the browser Back button re-renders the previous Episodes page *with patient names* from the back/forward cache. The session is genuinely dead (reloading that page redirects to `/login`; `/api/me` → 401; API calls 401), so no live data is served — but on a shared workstation the next person pressing Back could glimpse the previous clinician's patient list. T-04 step 3's "no staff data" expectation is technically not met because of this cached render. Consider `Cache-Control: no-store` on authenticated SPA responses. (Related note: a persistently-valid copied cookie after sign-out is already tracked in pending.md and out of scope.)
2. **Minor / UX — Cognito's own session survives a Poised-side rejection.** When sign-in is rejected on our side (suspended trust), Cognito still holds the user's authenticated session. Clicking "Sign in" again silently re-authenticates as the same (rejected) user and bounces straight back to `/login` with the suspended message — there's no visible way to sign in as someone else until Cognito's `/logout` is hit. Expected SSO behaviour, but it traps the user; worth considering a "sign in as a different user" affordance or a Cognito logout on our rejection.
3. **QA-tool filters didn't narrow when set programmatically.** The captured-messages "To" filter and the audit "Who" filter did not narrow the list when the value was injected via automation (form_input / typed) and submitted; the messages filter *did* narrow correctly when the value was committed and **Show** clicked by hand. This looks like a React controlled-input / automation-timing interaction rather than an app bug, but a quick hand re-check of both filters is worth it. These are S15/S11 tools, not S12.

## Console and network errors

None observed beyond the expected 401/404/409 responses for the negative cases above. No unexpected 4xx/5xx or console errors on the signed-in pages.

## Hand checks for Rahul

- **T-08 (removed membership):** the automated safety classifier blocked removing/re-adding Val's staff access. Please run T-08 by hand.
- **T-09 (invite reaches Cognito):** staff invite is a shared-resource mutation that also sends a real Cognito email; not attempted in auto mode. Please run by hand (overlaps H-1).
- **R-05 (validate shows the patient's answer):** not reachable because the T-10 patient saved but didn't submit the HQ, and validation mutates. Please validate by hand after submitting a patient HQ.
- **T-10 phone size:** `resize_window` to 390×844 reported success but the rendered viewport stayed ~1512px wide in this environment, so true phone-width rendering couldn't be confirmed. The patient pages were centered and showed no horizontal overflow at the captured width. Please eyeball the patient journey at phone size.
- **H-1 / H-2 / H-3** remain yours as stated in the plan.

## Test data created on QA (made-up)

- Episode "MT S12 1714 Smith" in Hospital A1 (id `72abbef1-1e5f-4099-b2ea-781511f77ebc`), email `mt.s12.1714.p@example.org` — patient answered one question, not submitted.
- R-06 ran scheduled jobs as of Now+2days (a one-shot "run as of"; the live timer/clock was **not** changed), generating 12 captured reminders.
- Trust B was suspended and **re-activated** (left ACTIVE).

## GIFs

- `~/Downloads/S12-T02-signin.gif` — the main happy path (Cara's password + authenticator sign-in to Hospital A1 Episodes). No failures to record.
