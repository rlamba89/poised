# S06 test plan: the app on AWS (the QA environment)

For a tester who uses only the browser. **Test on the QA web address, not localhost.** Patient pages at **phone size (390×844)**, staff pages at desktop size.

**What S06 changed:** the app now also runs on AWS. The web pages are served by **Amplify Hosting**, and every `/api/…` request is passed by Amplify to the Go API, which runs as two **Lambda** functions (one for staff, one for the patient's link) behind API Gateway. The database is **Neon** (Postgres in AWS London). **Nothing should look or behave differently from `make dev`**, apart from:
- the web address;
- the very first request after a few idle minutes can take about a second longer (a "cold start");
- the **jobs timer doesn't run on AWS yet** (S17): **Run jobs** on the Scheduled jobs page still works, but "Move the clock" has no effect.

Small changes also in S06:
- the session cookies are **Secure** (sent only over https);
- after a deploy, a page whose code changed reloads itself once;
- a missing file under `/assets/` is a real "not found" rather than the app;
- the rate limit on patient sign-ins now counts each browser's own address (new QA-only page `/api/qa/client` shows it).

## Setup

1. **Nothing to start:** the app is already deployed. **Open https://main.d33kh0gi8vk83v.amplifyapp.com** (call it **QA** below; every path in this plan is relative to it, e.g. **QA/login**).
2. The data is made up. The database was freshly seeded, but the automated journeys have also run there, so you'll see questionnaires and patients named "E2E …". Ignore them unless a case uses them.
3. **Staff sign-in:** QA/login lists the seeded users; click a name. Useful ones:

   | User | Can do |
   | --- | --- |
   | **Alex Author** | super clinician of Trust A (picks Hospital A1 or A2): authoring and publishing |
   | **Cara Clinician** | clinician at **Hospital A1** (Trust A): episodes, validating, POA Summary |
   | **Val Clinician** | clinician at **Hospital A2** (Trust A) only |
   | **Ada Admin** | admin of the whole of Trust A (Staff, Settings, Audit) |
   | **Bea Author** | super clinician of **Trust B** (Hospital B1) |
   | **Sam Superadmin** | the platform superadmin console |

4. **QA-only helper pages (no sign-in):**
   - **Captured messages:** QA/qa/messages lists every text and email the app "sent" (nothing is really sent). Type a mobile as stored (e.g. `+447700900611`) or an email in **To**, then **Show**.
   - **Scheduled jobs:** QA/qa/jobs. **Run jobs** with **Now + 2 days** etc. works. The timer card and "Move the clock" do nothing on AWS: don't report that.
5. **Made-up contacts only:**
   - Mobiles from `07700 900600`–`07700 900699`.
   - Emails `…@example.org`.
   - Use a new surname per case, e.g. `Qa06<your initials><case>`.
6. **Patient and staff in one browser:** staff and patient sessions can coexist in one browser. If one gets in the way, sign out (staff: **Sign out**; patient: **Sign out** on the patient's home).
7. **Browser notes:**
   - The Claude in Chrome network log shows a `204` as `503`; ignore a 503 on sign-in, sign-out or other 204 routes.
   - Two `401`s from `/api/me` on the login page are expected.

## Test cases

### The platform

**T-01 Health and config**
1. Open QA/api/health, then QA/api/config (signed out).
**Expected:**
- health shows `{"status":"ok"}`;
- config shows `{"locale":"en-GB","timeZone":"Europe/London","nationalId":{"type":"nhs","label":"NHS number"}}`;
- the address stays on QA (no redirect to an `amazonaws.com` address).

**T-02 Sign in and out (cookies through Amplify)**
1. Open QA/login, click **Cara Clinician**.
2. **Expected:**
   - you land on Hospital A1's **Episodes** (or the hospital picker, then pick Hospital A1);
   - the network log shows `POST /api/dev/login` (204, shown as 503) then `GET /api/me` **200**.
3. Open QA/api/me in a new tab. **Expected:** JSON naming Trust A and Hospital A1.
4. If your tools can show response headers, check the login response's `Set-Cookie`. **Expected:** `sj_token=…; Path=/; …; HttpOnly; Secure; SameSite=Lax`.
5. Click **Sign out**. **Expected:**
   - you're back on the login page;
   - QA/api/me now answers **401**;
   - pressing the browser's **Back** button doesn't show Cara's data (a reload sends you to the login page).

**T-03 Reloading deep links (staff)**
Signed in as Cara, open each of these, then **reload**, and also paste its address into a **new tab**:
- the Episodes list;
- an episode's page;
- an episode's **POA Summary** (any "E2E J5 …" episode, or T-11's).

Then, as Alex Author: a questionnaire's editor (QA/o/…/h/…/questionnaires/…) and a Question Set's **Preview**.

**Expected:**
- every reload and new tab shows the same page, with no blank page and no "Something went wrong";
- the address bar keeps the full address.

Signed out, paste the episode address. **Expected:** the login page (after sign-in you can reach it again).

**T-04 Not-found addresses**
1. Open QA/assets/does-not-exist.js. **Expected:** a plain "not found" (404) from Amplify, **not** the app's page.
2. Open QA/api/no-such-route. **Expected:** a 404 (`404 page not found`), not the app.
3. Open QA/no/such/page. **Expected:** the app's own "not found" page.

**T-05 The patient's address for rate limits**
1. Open https://checkip.amazonaws.com in a tab and note your public address.
2. Open QA/api/qa/client.
**Expected:**
- `address` is **your** public address from step 1, not an Amazon address;
- `forwardedFor` starts with your address, followed by two Amazon addresses.

### J1–J5 by hand

**T-06 J1: author a questionnaire and preview it**
1. As **Alex Author**: pick **Hospital A1** → **Questionnaires** → **New Questionnaire**.
2. Name it `QA06 <initials> J1`, add a description, then create it.
3. In the editor:
   - add a Question Set (e.g. "About you");
   - add a page;
   - add a Yes/No question "Do you smoke?";
   - save.
4. **Reload** the editor.
5. Open the Question Set's **Preview**.
**Expected:**
- after the reload, the set, page and question are still there;
- the preview shows "Do you smoke?" with Yes/No.

**T-07 J2: publish and create a new version**
1. Still as Alex, in T-06's questionnaire, click **Publish** (top bar).
2. In the dialog, publish version 1.
3. Then click **Create new version**.
**Expected:**
- the dialog lists any problems first (none for T-06's questionnaire), then publishes;
- the questionnaire shows version 1 **published** (read-only);
- **Create new version** opens a draft version 2 with the same content.

Optional: add a question whose condition tests a question you then delete; Publish should list "1 problem to fix first" and stay disabled until it's fixed.

**T-08 J3: the patient fills in their questionnaire on a phone (390×844)**
1. As **Cara**: **Episodes → New episode** → new patient:
   - first name `Sam`, surname `Qa06<initials>J3`;
   - date of birth `17/05/1980`, any sex;
   - mobile `07700 900611`, email `qa06j3.<initials>@example.org`;
   - a made-up NHS number if asked (the form says if it's invalid);
   - **Health questionnaire:** any published one whose name starts **"E2E J3"** (it has three Question Sets: About you, Your health, Nurse checks).
   - Click **Create episode**.
2. **Expected:** the episode page says "Invite sent by text message and email."
3. QA/qa/messages, To `+447700900611` → **Show**. **Expected:** the invite text "Hospital A1: please complete your pre-op health questionnaire: https://main.d33kh0gi8vk83v.amplifyapp.com/i/…". **The link is on the QA address**, not localhost.
4. At **390×844**, open the link:
   - tap **Continue**;
   - enter date of birth **17 5 1980**;
   - answer the first Question Set part-way;
   - **reload** the page;
   - finish every set, then **Send my answers**.
5. **Expected:**
   - the reload keeps your answers and the session;
   - the patient pages fit the phone width (no sideways scrolling);
   - after sending: a thank-you;
   - as Cara, the episode's status is **Ready for review**.

**T-09 J3 refused paths (phone size)**
1. Create another episode as in T-08 (surname `Qa06<initials>J3b`, mobile `07700 900612`). Open its link at 390×844 → **Continue** → enter a **wrong** date of birth (`1 1 1990`).
2. Open a made-up link QA/i/not-a-real-token.
**Expected:**
- step 1 is refused with a plain message and the number of tries left (5 in all), and the questionnaire doesn't open;
- step 2 says the link isn't valid, with no stack trace or blank page.

**T-10 J4: the clinician validates and completes the review**
1. As **Cara**, open T-08's episode (Ready for review).
2. Open each Question Set in turn. In "About you", **change one answer** from the patient's, then **Validate**.
3. Validate the other sets, then **Complete review**.
**Expected:**
- the changed answer shows a "Patient answered …" note;
- each set shows as validated;
- after **Complete review** the status is **Ready for POA**.

**T-11 J5: the POA Summary**
1. On T-10's episode, open **POA Summary**.
2. Look at both tabs.
3. Change the episode's **status**, adding a comment.
**Expected:**
- one tab shows the patient's answers and the other the validated answers, with T-10's change marked;
- the status changes;
- the comment appears in the episode's **General notes**;
- a **Print** button is there (don't print).
- Then **reload** the POA Summary: it's unchanged.

**T-12 Wrong role and another trust's data**
1. Copy the address of T-08's episode page (Hospital A1).
2. Sign out, sign in as **Val Clinician** (Hospital A2 only) and paste it.
3. Sign out, sign in as **Bea Author** (Trust B) and paste it.
**Expected:** both are refused (a "no access" or not-found message). No patient details show, in the page or in the network responses.

**T-13 Console and network errors**
Throughout T-02–T-12, keep an eye on the browser console and the network log.
**Expected:**
- no unexpected red console errors;
- no failed requests (4xx/5xx) other than the ones a case expects: the 401s on the login page, T-04's 404s, T-09's refusals, and 204s shown as 503.

## Regression (earlier sessions, now on AWS)

**R-1 Staff list (S09):** as **Ada Admin**, open **Staff**. **Expected:** Trust A's staff with their roles. Invite a new staff member `qa06.<initials>@example.org` as clinician at Hospital A1; they appear as invited.

**R-2 Superadmin console (S10):** sign in as **Sam Superadmin**. **Expected:** the trusts list (Trust A, Trust B, and any "E2E …" trusts); opening Trust A shows its hospitals. Don't suspend anything.

**R-3 Audit log (S11):** as **Ada Admin**, open **Audit**. **Expected:** rows for T-08–T-11 (episode created, answers viewed, review completed…) with Cara's name.

**R-4 Patient records (S13):** as **Cara**, search patients for `Qa06<initials>`. **Expected:** T-08's and T-09's patients, each with their episode.

**R-5 Reminders by "Run jobs" (S16):** create an episode as in T-08 (surname `Qa06<initials>R5`, mobile `07700 900615`). Then QA/qa/jobs → **Now + 2 days** → **Run jobs**. **Expected:** "Ran … 1 reminder sent" (counts may include others). Captured messages for `+447700900615` show the invite and the reminder ("Hello Sam, a reminder from Hospital A1…"), with links on the QA address.

**R-6 Returning patient sign-in (S15), phone size:**
1. At 390×844, after T-08, open QA/signin.
2. Ask for a link by mobile `07700 900611`.
3. Open the link from the captured messages, and enter the date of birth.
**Expected:** the patient's home lists T-08's questionnaire as sent.

## Out of scope (don't report)

- **The web address:** Amplify's `…amplifyapp.com` address (a real domain comes later).
- **The dev login:** stays on QA until Cognito (S12).
- **The jobs timer** and **Move the clock** on the Scheduled jobs page: no timer on AWS until S17.
- **Speed:** the first request after an idle spell taking about a second (Lambda cold start, Neon waking).
- **Messaging:** no real texts or emails (captured only).
- **The reload-once after a deploy:** it needs a deploy during the test; it's covered by an automated test.
- **The data:** "E2E …" questionnaires, trusts and patients left by the automated journeys.
- **Browser support:** Safari or other browsers.
