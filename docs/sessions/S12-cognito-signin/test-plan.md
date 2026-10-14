# S12 test plan: staff sign-in with Cognito and MFA, on QA

**Changes after test report 1 (for the retest):**
- **Fixed, retest T-13 and T-14** (new): Back after a staff sign-out no longer shows the previous page (finding 1); a sign-in Poised refuses (e.g. a suspended trust) also signs out of Cognito's page, so **Sign in** asks for the password again (finding 2). Both were deployed to QA by hand (17:5x, 14 Oct).
- **T-04 step 3** now expects Back to land on `/login` with nothing of the previous page.
- **T-08, T-09 and R-05 are hand checks for Rahul** (they change shared QA data or send a real email, which the automatic tester may not). They're covered automatically too: T-08 by the new journey "a removed membership is refused on the next request" (runs locally and on QA) and the S09 API tests; T-09 by the API tests (the invite calls Cognito's `AdminCreateUser`) and H-1; R-05 by journey J4 (the clinician validates, "Patient answered"), which passes on QA.

**Everything is on QA:** https://main.d33kh0gi8vk83v.amplifyapp.com (no `make dev` needed). S12's code was deployed there by hand on 14 Oct (`make deploy-stacks`, `aws-db`, `qa-users`, `deploy-web`), so QA runs this working tree, not `master`.

## Setup

1. **Nothing to start.** Check QA answers: `curl -s https://main.d33kh0gi8vk83v.amplifyapp.com/api/health` → `{"status":"ok"}`.
2. **The test users** are the seeded staff. Each has a password and an authenticator app's secret, kept only in AWS (never in the repo, a report or the chat). From the repo root, in a terminal:
   - **The password:** `make qa-password PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'` copies it to the clipboard (it's never printed). Paste it into the password box with **Cmd+V**. Then clear the clipboard: `pbcopy < /dev/null`.
   - **The authenticator code:** `make qa-code PROFILE=AdministratorAccess-026090536258 WHO='Cara Clinician'` prints the current 6-digit code and how many seconds it's good for. If fewer than 5 seconds are left, wait and run it again.
   - **Each code works once.** If Cognito says "Invalid or expired code entered" for a code you just fetched, someone (you, or the pipeline) used it already: wait for the next one (up to 30 seconds) and fetch again.
   - If a `make` command says it can't use the AWS profile, the AWS sign-in has expired: stop and report **BLOCKED: AWS sign-in expired** (Rahul runs `aws sso login --profile AdministratorAccess-026090536258`).
   - **Never write a password, secret or code into the report.**
3. **Who's who** (emails as typed into Cognito's Email address box):

   | Name | Email | Access |
   | --- | --- | --- |
   | Cara Clinician | cara.clinician@hospital-a.example | Clinician, Hospital A1 only |
   | Val Clinician | val.clinician@trust-a.example | Clinician, Hospital A2 only |
   | Ada Admin | ada.admin@trust-a.example | Admin, all of Trust A |
   | Alex Author | alex.author@hospital-a.example | Super clinician, all of Trust A |
   | Bea Author | bea.author@hospital-b.example | Super clinician, all of Trust B (only Trust B) |
   | Max Multi | max.multi@trust-b.example | Clinician at A2, and Trust B's admin |
   | Sam Superadmin | sam.superadmin@poised.example | Platform admin (the console, `/admin`); the testers' tools |

4. **How to sign in (used by most cases):** open QA's `/login`, click **Sign in**. Cognito's page opens (its address ends in `amazoncognito.com`). Type the email, paste the password, click **Sign in**. The next page is "Authenticator app MFA": type the code, click **Sign in**. You come back to Poised.
5. **To switch user:** use Poised's **Sign out** (top right). It also signs you out of Cognito's page.
6. **The testers' tools** (captured messages `/qa/messages`, scheduled jobs `/qa/jobs`) now need you to be signed in as **Sam Superadmin**.
7. **Test data:** make up names like `MT S12 <time> Smith`. Don't invite real people. A case below invites one made-up `@example.com` address: Cognito emails it (that address doesn't exist), which uses one of the pool's ~50 emails a day, so **invite only where a case says to**.
8. **Re-activate anything you suspend** (T-07), even if the case fails.

## Test cases

### T-01 The sign-in page (no dev login)
1. Open `/login` (signed out).
2. **Expected:** the Poised title, the text "Sign in with your email address, your password and a code from your authenticator app.", a **Sign in** button, and help text about the temporary password and "Forgot your password?". **No list of users** and no "Development sign-in".
3. Open `/api/dev/users` and `/api/dev/login` in the address bar. **Expected:** both "404 page not found".

### T-02 Sign in with a password and an authenticator code
1. Sign in as **Cara Clinician** (setup step 4).
2. **Expected:** you land on Hospital A1's **Episodes** (URL `/o/…/h/…/episodes`), the header shows Trust A / Hospital A1, Clinician, and Cara's name. No console errors.
3. Reload the page. **Expected:** still signed in, same page.
4. Copy the URL, open a new tab, paste it. **Expected:** the same page, signed in (one browser session).
5. **Expected (from step 1):** Cognito's pages were at `https://poised-uk-qa-….auth.eu-west-2.amazoncognito.com/…`.

### T-03 A wrong password, then a wrong code, are refused
1. Signed out, click **Sign in**, type **Val Clinician**'s email and a made-up password (e.g. `Wrong-password-12!`), click **Sign in**.
2. **Expected:** you stay on Cognito's page with "Incorrect username or password." Nothing about whether the email exists.
3. Paste Val's real password, **Sign in**. On the code page type `000000`, **Sign in**.
4. **Expected:** "Invalid or expired code entered. Try again."; you stay there.
5. Type the real code (fetch a fresh one). **Expected:** Val lands on Hospital A2's Episodes.
6. Sign out.

### T-04 Sign out
1. Sign in as **Cara Clinician**, then click **Sign out**.
2. **Expected:** you end on Poised's `/login` (after a short hop through Cognito).
3. Press the browser's **Back** button once, then again. **Expected:** you end on `/login`; Cara's Episodes list (patient names) never shows, not even for a moment.
4. Open `/api/me` in the address bar. **Expected:** `{"error":"Please sign in."}`.
5. Click **Sign in**. **Expected:** Cognito asks for the email and **password** again (it doesn't sign Cara straight back in).

### T-05 Where each person lands
1. Sign in as **Max Multi**. **Expected:** "Choose a hospital", with Trust A (Hospital A2, Clinician) and Trust B (Hospital B1, Admin, whole trust). Pick B1: Questionnaires. Sign out.
2. Sign in as **Sam Superadmin**. **Expected:** the platform admin console, `/admin`, with the trusts list. Sign out.
3. Sign in as **Alex Author**. **Expected:** Questionnaires (or the picker if Alex has several hospitals; Alex is trust-wide, so "Choose a hospital" with A1 and A2). Sign out.

### T-06 The testers' tools need a platform admin
1. Signed out, open `/qa/messages`. **Expected:** "Sign in as a platform admin to use this page." with a **Sign in** link. No messages listed.
2. Open `/qa/jobs`. **Expected:** the same message where the timer's clock would be.
3. Open `/api/qa/messages` in the address bar. **Expected:** `{"error":"Please sign in."}`.
4. Sign in as **Cara Clinician**, open `/qa/messages`. **Expected:** "Only a platform admin can use this page." `/qa/jobs`: the same.
5. Sign out; sign in as **Sam Superadmin**; open `/qa/messages`. **Expected:** the captured messages list (newest first). Click **Reset sign-in limits**: "Sign-in limits cleared…". Open `/qa/jobs`: the timer's clock and the run form show. (Don't run jobs here; T-10 does.)

### T-07 A suspended trust's staff are refused at sign-in
1. Signed in as **Sam Superadmin**, open `/admin`, open **Trust B**, and **suspend** it. Note the "suspended" status. Sign out.
2. Sign in as **Bea Author** (Trust B only): password and code as usual.
3. **Expected:** after the code you come back to Poised's `/login`, with a red message "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust." `/api/me` says "Please sign in."
4. Sign in as **Sam Superadmin** again (wait for a new code if needed) and **re-activate Trust B**. Sign out.
5. Sign in as **Bea Author**. **Expected:** Hospital B1, signed in. Sign out.
6. **Make sure Trust B is active at the end**, even if a step failed.

### T-08 A removed membership (hand check for Rahul; automatic run: skip, mark Blocked)
1. Sign in as **Ada Admin**, open **Staff** (Trust A). Find **Val Clinician** (Hospital A2 · Clinician) and **Remove** that access (confirm in the app's own dialog). Sign out.
2. Sign in as **Val Clinician**. **Expected:** signed in, but with no hospital: "You don't have access to any hospital yet" / "Ask an admin of your trust to invite you." Opening Hospital A2's Episodes by URL (`/o/<Trust A>/h/<A2>/episodes`, copy the ids from an earlier step) shows "You don't have access to this hospital". Sign out.
3. Sign in as **Ada Admin**, **Staff**, Val → **Add access**: Hospital A2, Clinician. **Expected:** added at once (Val already has an account: no new invite email). Sign out.
4. Sign in as **Val Clinician**. **Expected:** Hospital A2's Episodes again. Sign out.

### T-09 An invite reaches Cognito (hand check for Rahul; automatic run: skip, mark Blocked)
1. Sign in as **Ada Admin**, **Staff**, **Invite**: email `mt.s12.<HHMM>@example.com` (a made-up address), name `MT S12 <HHMM> Tester`, Hospital A1, Clinician, **Send invite**.
2. **Expected:** no error; the person is listed as **Invited** with an expiry 7 days ahead. (On QA this created their user in Cognito, which emails the temporary password to that made-up address; Rahul checks a real one by hand, H-1.)
3. Click **Resend invite** on that row once. **Expected:** "Invitation sent again to … It expires on …", no error.
4. Sign out. (Don't try to sign in as them: the password is in the email.)

### T-13 Back after sign-out shows nothing (finding 1)
1. Sign in as **Alex Author**, pick **Hospital A1**, open **Episodes** (patient names show).
2. **Sign out**. You're on `/login`.
3. Press **Back**. **Expected:** the page doesn't come back from memory: you end on `/login` (or a loading spinner, then `/login`), and no patient name shows at any moment. Press Back again: the same.
4. Press **Forward**, then Back once more. **Expected:** still nothing of Alex's.
5. In a terminal: `curl -sI https://main.d33kh0gi8vk83v.amplifyapp.com/login | grep -i cache-control` → `no-store`; the same for a `/assets/…js` file → `public, max-age=31536000, immutable`.

### T-14 A refused sign-in signs out of Cognito too (finding 2)
1. As **Sam Superadmin**, suspend **Trust B** (as in T-07). Sign out.
2. Sign in as **Bea Author** (password and code). **Expected:** back on `/login` with "Your trust or hospital is suspended…".
3. Click **Sign in** again. **Expected:** Cognito's page asks for the **email and password** (it doesn't send Bea straight back). The message from step 2 is gone after this.
4. Sign in as **Sam Superadmin** from that page (it accepts another person) and **re-activate Trust B**. Sign out. **Make sure Trust B is active at the end.**

### T-10 Patients still work, with the tools as the platform admin (phone size)
1. Sign in as **Cara Clinician**. **Episodes → New episode**: a made-up patient (`MT S12 <HHMM> Smith`, born 17/05/1980, female, email `mt.s12.<HHMM>.p@example.org`), the newest published HQ, **Create episode**. **Expected:** "Invite sent by … email." Sign out.
2. Sign in as **Sam Superadmin**, open `/qa/messages`, show messages **To** that email. **Expected:** the invite, with an `/i/…` link. Copy the link.
3. Resize the window to **390×844**. Sign out of Poised (Sign out from `/admin`). Open the copied link. **Expected:** "Your health questionnaire", **Continue**, then the date of birth (17 / 5 / 1980), then the questionnaire; nothing wider than the screen.
4. Answer one question, then **Sign out** (the patient's). Resize back to desktop.

### T-11 Security headers
1. In a terminal: `curl -sI https://main.d33kh0gi8vk83v.amplifyapp.com/login`.
2. **Expected** among the headers: `x-frame-options: DENY`, `content-security-policy: frame-ancestors 'none'`, `strict-transport-security: max-age=31536000; includeSubDomains`, `x-content-type-options: nosniff`, `referrer-policy: no-referrer`.
3. The same for `/` and for a file under `/assets/` (any `.js` name from the page's source).

### T-12 Forgotten password page (no email needed)
1. Signed out, **Sign in**, then **Forgot your password?** on Cognito's page.
2. **Expected:** Cognito's reset page asks for the email address. **Don't submit it** (it would email a made-up address). Go back to Poised's `/login` with the browser's Back button: the **Sign in** button works again.

## Regression (earlier sessions, on QA)

- **R-01 (S08):** as **Max Multi**, switch from Hospital B1 to Hospital A2 in the header: the role changes to Clinician; Trust A shows only A2.
- **R-02 (S09):** as **Ada Admin**, Staff lists Trust A's people, each with their access lines; a clinician (Cara) has no Staff link.
- **R-03 (S10):** as **Sam Superadmin**, the console lists Trust A and Trust B; open Trust A: its hospitals and admins show.
- **R-04 (S11):** as **Ada Admin**, Audit: filter **Who** = Cara Clinician, **Show**: rows of Cara's actions today (from T-10).
- **R-05 (S13/S14; hand check for Rahul, automatic run: skip, mark Blocked):** as **Cara Clinician**, the episode from T-10 shows the patient, "Last sent", and the answer saved (Validate the set shows the patient's answer).
- **R-06 (S16):** as **Sam Superadmin**, `/qa/jobs`: run the jobs as of **Now + 2 days**: "Ran N jobs as of …"; `/qa/messages` To the T-10 email: a reminder is listed.

## Hand checks for Rahul (not for the tester)

- **T-08, T-09 and R-05** above (the automatic tester may not change shared QA data or send a real email).
- **T-10 at a real phone width:** the tester's window couldn't be made 390 px wide; open a patient's invite link on your phone (or a 390×844 window) once.
- **Test report 1, finding 3:** try the captured-messages **To** filter (as Sam) and the Audit **Who** filter (as Ada) by hand once (pending.md).

- **H-1, a real invite email (S12 answers item 2):** `make aws-platform-admin PROFILE=AdministratorAccess-026090536258 EMAIL=<your email> NAME="<your name>"` (or, as Ada Admin on QA, invite your email to Hospital A1). The email comes from `no-reply@verificationemail.com`, names QA's address, your email and a temporary password. Sign in: Cognito asks for a new password, then shows a QR code for your authenticator app; then you land in Poised (the console, or Hospital A1).
- **H-2, a forgotten password:** with that account, **Forgot your password?**: a code arrives by email; set a new password; sign in with it and your authenticator.
- **H-3, the pipeline:** after you push (see the build notes), the run is green: `Database-and-web` prints "8 test users can sign in…", and `E2E` passes with some journeys skipped (the dev-login-only ones).

## Out of scope (don't report)

- Cognito's own pages' look (Cognito's default style; Poised's branding is in pending.md), wording and layout.
- The ~50-a-day limit on Cognito's emails, and that Cognito's emails to made-up addresses bounce.
- Signing in as a newly invited person (their password is in an email: H-1).
- Locally (`make dev`), the dev login is unchanged and isn't part of this test.
- That a copied staff cookie keeps working after Sign out until it expires (older than S12; pending.md).
- The pipeline (H-3).
