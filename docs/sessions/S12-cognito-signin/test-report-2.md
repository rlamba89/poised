# S12 test report 2 (retest): staff sign-in with Cognito and MFA, on QA

**Summary:** 9 passed, 0 failed, 3 blocked (hand checks for Rahul, unchanged from report 1). Both report-1 findings are fixed on QA. No Blocker, Major or Minor defects in S12's cases; one Minor item found while exploring.

**Environment**
- URL: https://main.d33kh0gi8vk83v.amplifyapp.com (QA, uk-qa, eu-west-2), redeployed by hand after report 1
- Commit at test start: `ab90771`, working tree not clean (the session's uncommitted changes; QA runs that working tree, not `master`)
- Date/time: 14 Oct 2026, about 17:49 to 17:58 BST
- Tester: automatic (manual-test, `/run-sessions`, retest), using Claude in Chrome and curl. No AWS resources were changed.
- Served bundle: `/assets/index-79YIFPht.js`

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-13 Back after sign-out shows nothing | Pass | Alex → Hospital A1 → Episodes (patient names showed) → Sign out → `/login`. Back: the Episodes URL reloaded and redirected to `/login`; no patient name in the DOM (`innerText` checked after each step; the navigation type was `reload`, not a bfcache restore). Back again, then Forward and Back once more: `/login` each time, nothing of Alex's. Step 5: `/login` and `/` send `cache-control: no-store`; `/assets/index-79YIFPht.js` sends `public, max-age=31536000, immutable`. |
| T-04 Sign out (tightened step 3) | Pass | Cara signed in → Sign out → `/login` after the Cognito hop. I added a `pagehide` listener before signing out. It recorded `persisted: false` and an **empty page (0 characters of text)** at unload, so the page that left was already blank and wasn't kept in the back/forward cache. Back → `/login` (Cara's list never showed). A second Back went to Cognito's empty sign-in form from earlier in the history (no data, nothing auto-filled). `/api/me` → `{"error":"Please sign in."}`. **Sign in** → Cognito asked for the email **and password** again. |
| T-14 A refused sign-in signs out of Cognito too | Pass | As Sam: suspended Trust B (in-app modal) → Sign out. Bea: password + code → back on `/login` with "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust."; `/api/me` → 401. **Sign in** again → Cognito showed the empty email and password form (it didn't send Bea straight back). Signed in as Sam from that page, which accepted the other person. Re-activated Trust B ("Trust B is active again. Its staff can sign in.") → Sign out → `/login` **without** the suspended message. **Trust B left ACTIVE.** |
| Quick sign-in / sign-out | Pass | Five Cognito sign-ins (Alex, Cara ×2, Sam ×2, Bea ×2), each with password and authenticator code, all landed correctly. Alex → picker (A1, A2), Cara → A1 Episodes, Sam → `/admin`, Bea → Hospital B1. Reloading a signed-in page (now `no-store`) stays signed in. Every sign-out ended on `/login`, and each next **Sign in** asked for the password (the shared-workstation check). Double-clicking **Sign out** is harmless. No console errors. |
| T-06 Testers' tools need a platform admin (spot check) | Pass | Signed out: `/qa/messages` and `/qa/jobs` show "Sign in as a platform admin to use this page." with a Sign in link; `/api/qa/messages` and `/api/qa/jobs` → 401 `{"error":"Please sign in."}`. As Cara: "Only a platform admin can use this page." on both. As Sam: the captured messages list (newest first), and `/qa/jobs` shows the run form and the timer. I didn't click Reset sign-in limits or run any jobs this round. |
| T-07 Suspended trust refused at sign-in (spot check) | Pass | Done together with T-14. Refused while suspended, with the right message. After re-activation, Bea signs in to Trust B / Hospital B1 (Questionnaires, Super clinician). Trust B is active at the end. |
| T-11 Security headers (curl, re-checked) | Pass | `/login`: `x-frame-options: DENY`, `content-security-policy: frame-ancestors 'none'`, `strict-transport-security: max-age=31536000; includeSubDomains`, `x-content-type-options: nosniff`, `referrer-policy: no-referrer`, plus `cache-control: no-store`. The `.js` asset also has `x-frame-options: DENY`. |
| T-01 Sign-in page (seen in passing) | Pass | Same as report 1: Poised title, the help text and the Sign in button. No user list. |
| T-02 Sign in, reload (seen in passing) | Pass | Covered by the quick sign-in row: reloading Cara's Episodes keeps her signed in on the same page. |
| T-08 A removed membership | Blocked | Hand check for Rahul: it changes shared QA staff data. Covered by the new journey and the S09 API tests, per the plan. |
| T-09 An invite reaches Cognito | Blocked | Hand check for Rahul: it sends a real Cognito email. |
| R-05 Validate shows the patient's answer | Blocked | Hand check for Rahul (per the plan). |

The other report-1 passes (T-03, T-05, T-10, T-12, R-01 to R-04, R-06) weren't re-run, as the conductor's retest scope says. Nothing in this fix touches them beyond sign-in and sign-out, which the rows above cover.

## Failures

None.

## Found while exploring

1. **Minor: a second open tab keeps showing patient data after a sign-out in another tab.** Steps: as Bea, open Hospital B1's Episodes in tab 2, then click **Sign out** in tab 1. Tab 2 still shows the full Episodes list (patient names) until someone clicks in it. Clicking Questionnaires or Episodes then shows a red "Please sign in." box under the header. The header still says "Bea Author · Super clinician" with a Sign out button, and the page doesn't go to `/login`. The session really is over (every API call returns 401), so no new data loads. But on a shared workstation, a list left open in another tab stays readable, and the page doesn't send the user to sign in. A possible fix: on any 401 from the staff API, go to `/login`, or tell other tabs about the sign-out (the `storage` event or a BroadcastChannel) so they clear their pages. This existed before S12 and isn't in `docs/pending.md`.
2. **(Tooling, not the app)** In the background tab, the extension's clicks sometimes didn't reach the page until I took a screenshot. A capturing listener on the document recorded no `pointerdown` or `click`. This made **Sign in** look dead at first. It isn't an app bug: once the tab had been screenshotted, the same click opened Cognito every time.

## Console and network errors

No console errors or warnings on any signed-in page this round. The only 4xx responses were the expected ones: 401 from `/api/me` and `/api/qa/*` when signed out or refused, and 401 "Please sign in." in the stale second tab (exploring item 1).

## Test data changed on QA

- Trust B was suspended and then **re-activated**: it's ACTIVE now (seen on its console page, and Bea signed in afterwards).
- No episodes, staff, invites or jobs were created or run this round.

## Hand checks for Rahul

- **T-08, T-09, R-05**: these change shared QA data or send a real email (as the plan says).
- **T-10 at a real phone width, report 1's finding 3 (the To and Who filters), and H-1 to H-3**: still yours, as listed in the plan.

## GIFs

- `~/Downloads/S12-T13-back-after-signout.gif`: Alex signs in, goes to Episodes, signs out, then Back and Forward; it ends on `/login` each time.
- `~/Downloads/S12-T14-refused-signin-signs-out-cognito.gif`: suspend Trust B, Bea refused, Sign in asks for the password again, Sam re-activates Trust B, sign out (the message has gone).
