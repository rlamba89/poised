# S15 test plan: patient accounts across trusts, sign-in links and the patient home

**Changed for retest 3 (after verification 1):**
- **T-24 (new):** "← Your questionnaires" saves the answer just typed before leaving; if the session has ended, it stays and says what wasn't saved.
- Not to report: an unjoined twin (sharing a contact and date of birth with an account holder) signing in with a sign-in link reaches the other person's account. **Rahul accepted it as a rare risk** (answers 10); sign-in is unchanged.

**Changed for retest 2 (after test report 2, Rahul's answers 8–9):**
- **The surname counts too:** an account now matches by **date of birth, mobile or email, first name and surname** (each ignoring case and spaces). New case **T-23** (same first name, birthday and phone, different surnames: two people). **T-19**'s name step now asks for **First name and Surname**, with the focus in First name.
- **T-22 uses a fresh patient** with no Trust B record (as test report 2 did).
- Wording: the greeting is capitalised ("Hello Sam" even if a hospital typed "sam"); adding a mobile says "Open it to finish adding your mobile number."; the home's Sign out says how to come back; a sign-in link to a contact the account has since changed away from says "This link has expired"; the name step's messages ("…date of birth and name don't match… check them…", the lock "…the details were entered wrongly…") and a name over 100 characters ("That name is too long…").
- Records joined by the old first-name-only rule with another surname (e.g. Kim OtherRT, " sam " CaseRT) were unjoined by a migration; they join the right account at their next invite.

**Changed for retest 1 (after test report 1, Rahul's answers 6–7):**
- **Reset the sign-in limits yourself:** the captured-messages page (`/qa/messages`) has **Reset sign-in limits**. Press it at the start, before T-15, and before T-18 (every request from this machine counts as one network, 10 an hour; test report 1 ran out by T-14).
- **New cases T-19–T-22:** twins (the first name now counts), adding a contact tells the existing one, each contact box takes only its own kind, a record joining later gets the account's contact. T-10 adds emoji and over-long emails. R-6 is a hand check for Rahul.
- Each home card now says whose questionnaire it is ("For Sam …"), and the procedure reads "Procedure: …".
- An account now matches by **date of birth, mobile or email, and first name** (ignoring case and spaces).

For a tester who uses only the browser. Every patient page is tested at **phone size (390×844)**; staff pages at desktop size.

## Setup

1. In your own terminal (the conductor does this in an automatic run):
   ```sh
   docker compose up -d db
   make migrate seed
   make dev
   ```
   Leave `make dev` running. `make seed` also clears the sign-in rate-limit counts.
2. Open **http://localhost:3000**.
3. **Staff sign-in** (the stub login page at http://localhost:3000/login, pick a name):
   - **Cara Clinician**: clinician at **Hospital A1** (Trust A).
   - **Val Clinician**: clinician at **Hospital A2** (Trust A).
   - **Bea Author**: super clinician of **Trust B** (Hospital B1).
   - **Alex Author**: super clinician of Trust A (publishes questionnaires).
   For two people at once, use **http://tester.localhost:3000** in the same browser: it keeps its own cookies (good for "the patient's phone" beside staff).
4. **Captured messages:** http://localhost:3000/qa/messages lists every text and email the app "sent" (nothing really goes out). Its **Reset sign-in limits** button clears the rate-limit counts (local and QA only); press it first, and wherever a case says so. Type a mobile (as stored, e.g. `+447700900123`) or an email in **To**, then **Show**. Links in messages are clickable; copy the path (`/i/…`, `/s/…`, `/v/…`) into the patient's browser.
5. **Made-up data only.** Mobiles: Ofcom's drama range, `07700 900000`–`07700 900999`. Emails: anything `@example.org`.
6. **Patient pages:** `/signin` (ask for a sign-in link), `/s/<code>` (sign-in link), `/home` (the patient's home), `/home/episodes/<id>` (a questionnaire), `/v/<code>` (confirm a new mobile or email), `/i/<code>` (an invite, as in S14).
7. **Each trust needs a published questionnaire.** Trust A's seeded one is fine. For Trust B, if Hospital B1's **New episode** page offers no health questionnaire, sign in as Bea, open **Questionnaires**, create one with one patient Question Set and one question, and publish it.

**How accounts work (so the cases make sense):** a patient's records in different trusts join **one account** when they prove their **date of birth** through a link that reached them at a **mobile or email** the record holds. The match needs the **same date of birth, the same mobile or email, and the same first name and surname** (each ignoring case and spaces). Two people sharing a phone stay separate: different dates of birth, or different first names (twins), or different surnames. Use the **same first name and surname** wherever a case means one person.

## Test cases

### T-01 One home lists both trusts (the same mobile) — phone size
1. As **Cara** (Hospital A1): **New episode** → new patient: first name `Sam`, surname `Home<your initials>`, date of birth `17/05/1980`, sex any, mobile `07700 900501`, no email; any questionnaire, procedure `Knee replacement` → **Create episode**.
2. As **Bea** (Hospital B1, in tester.localhost or after signing out): **New episode** → the same details (`Sam`, same surname, `17/05/1980`, mobile `07700 900501`), procedure `Hip replacement` → **Create episode**.
3. Captured messages, To `+447700900501`: two invites, one from Hospital A1, one from Hospital B1.
4. In a phone-size window (not signed in as staff, e.g. a private window or tester.localhost after signing out), open **Hospital A1's** invite link → **Continue** → date of birth `17 5 1980` → the questionnaire opens.
5. Tap **← Your questionnaires** (top left).
**Expected:** `/home` shows **Hello Sam** and **two** cards, newest first: **Hospital B1** (Procedure: Hip replacement) above **Hospital A1** (Procedure: Knee replacement), each with its questionnaire's name, **For Sam Home<initials>**, **To do**, and **Start**. No date of birth, NHS number or full mobile anywhere. Under **Your contact details**: Mobile number `•••501`, Email "Not given". No sideways scrolling.

### T-02 Open a questionnaire from the home, in either trust, and come back
1. From T-01's home, tap **Start** on the **Hospital B1** card.
2. Answer one question, wait for **Saved**, tap **← Your questionnaires**.
**Expected:** the B1 questionnaire opens (bar shows "Hospital B1"); back on the home its badge is **In progress** and its button **Continue**. Open it again: the answer is there. Repeat for Hospital A1.

### T-03 A returning patient signs in by email — phone size
1. As Cara: New episode for a new patient `Sam`, surname `Return<initials>`, date of birth `03/04/1975`, email `return.<initials>@example.org`, no mobile.
2. In the phone window, open the invite from captured messages → Continue → `3 4 1975` → it opens. Tap **Sign out** (top right).
3. Go to **http://localhost:3000/signin**. Type the email in capitals with spaces around it → **Send me a link**.
4. **Expected:** "Check your email", saying a link was sent *if* Poised is used with that address, that it works once within 15 minutes, plus **Send another link**.
5. Captured messages for the email: the newest is "Your Poised sign-in link", with a `/s/…` link and "15 minutes". Open the link in the phone window.
6. **Expected:** "Sign in to Poised" and **Continue** only: nothing about the patient. Tap **Continue**: "Confirm it's you … Poised needs your date of birth". Enter `3 4 1975` → **Continue**.
7. **Expected:** lands on `/home` (not the questionnaire), **Hello Sam**, the Hospital A1 questionnaire listed.

### T-04 A returning patient signs in by text message — phone size
1. Use T-01's patient. Sign out from their home (or use a fresh window). Go to `/signin`, type `07700 900501` → **Send me a link**.
2. **Expected:** "Check your text messages".
3. Captured messages for `+447700900501`: a text "Poised: your sign-in link. It works once, within 15 minutes: …". Open its link → Continue → `17 5 1980`.
**Expected:** the home with both trusts' questionnaires (as T-01).

### T-05 A sign-in link works once
1. After T-03 or T-04, open the **same** sign-in link again (paste it in the phone window) → **Continue**.
**Expected:** "This link has expired" with "This sign-in link has expired or has already been used. Please ask for a new one." and a **Get a new sign-in link** button that goes to `/signin`. The session from the first use still works: `/home` still shows the questionnaires (if you didn't sign out).

### T-06 An expired sign-in link is refused
1. Ask for a sign-in link (T-03's email), but **don't open it for 16 minutes**. (Do other cases meanwhile.)
2. Open it → **Continue**.
**Expected:** "This link has expired", as T-05. (An automated test also covers this with a moved clock.)

### T-07 Wrong dates of birth on a sign-in link
1. Ask for a new sign-in link for T-03's email. Open it → Continue → enter `1 1 1999`.
**Expected:** "That date of birth doesn't match our records… You have 4 tries left." Four more wrong dates → "This link is locked … Please ask for a new sign-in link." with **Get a new sign-in link**. Even the right date is then refused. A new link (from `/signin`) works.

### T-08 The rate limit per contact, with a friendly message
1. At `/signin`, ask for a link for T-03's email **three times** within 15 minutes (use **Send another link**). Count the ones you've already asked for in T-03/T-05/T-07 if within 15 minutes.
2. Ask a fourth time.
**Expected:** a red message: "You've asked for 3 links in the last 15 minutes. Please use the newest one, or wait 15 minutes and try again." No fourth email in captured messages. A **different** contact (e.g. T-01's mobile) still works.

### T-09 A mobile outside the country is refused
1. At `/signin`, type `+1 202 555 0143` → **Send me a link**.
**Expected:** "We can only send text messages to mobile numbers in this country. Please use your email address instead." Nothing in captured messages for it. Invalid input (`hello`, `123`, `a@b`, empty) gets a clear red message too.

### T-10 Asking for an unknown contact reveals nothing
1. At `/signin`, type `nobody.<initials>@example.org` → **Send me a link**.
**Expected:** the same "Check your email" screen as T-03 (nobody can tell whether an account exists). Captured messages: nothing for that address.
   Also try `😀😀@example.org` and an address with 70 letters before the `@`: **Expected:** "Enter a real email address…", nothing sent.
2. As Cara, create an episode for a new patient with email `fresh.<initials>@example.org` and **don't** open its invite. At `/signin` ask for that email.
**Expected:** "Check your email", but nothing is sent: a patient who has never opened an invite uses the invite (the screen says so in grey).

### T-11 Sign out
1. On the home, tap **Sign out**.
**Expected:** "You've been signed out" with **Get a sign-in link**. Reloading `/home`, or opening `/home/episodes/<id>` of one of theirs, stays signed out (no data).

### T-12 Another patient's episode address shows nothing — phone size
1. As Cara, open **T-01's** episode at Hospital A1 and copy the episode id: the last part of the URL `/o/…/h/…/episodes/<id>`.
2. In T-03's patient window, open `/home/episodes/<that id>`.
**Expected:** "Episode not found." inside the Poised bar, with **← Your questionnaires**; none of T-01's details. Also try `/home/episodes/abc`.

### T-13 Two people sharing a mobile stay apart
1. As Cara, create a patient `Kim <initials>`, date of birth `04/05/2016`, mobile `07700 900501` (T-01's mobile), no email. Open the invite from the newest text to `+447700900501` → Continue → `4 5 2016`.
2. Tap **← Your questionnaires**.
**Expected:** Kim's home shows only Kim's questionnaire, **Hello Kim**, not Sam's two.
3. Sign out. `/signin` with `07700 900501` → open the text's link → Continue → `17 5 1980`.
**Expected:** Sam's home (both trusts), not Kim's. Same link, Kim's date → Kim's home (use a fresh link: one link works once).

### T-14 A record joins when the patient signs in (trust B invited later)
1. As Bea, create an episode at Hospital B1 for T-03's patient: `Sam`, `Return<initials>`, `03/04/1975`, email `return.<initials>@example.org`. Don't open its invite.
2. As the patient, sign in by email (T-03 steps 3–6).
**Expected:** the home now lists **both** Hospital B1 and Hospital A1.

### T-15 Change the email from the home (PAT-10) — phone size
0. On `/qa/messages`, press **Reset sign-in limits**.
1. Signed in as T-03's patient, on the home under **Your contact details**, tap **Change** beside Email address. Type `new.<initials>@example.org` → **Send a link to confirm**.
**Expected:** green "We've sent a link to new.… Open it to finish the change. Until then, we'll keep using your current email address." Reload: the email still shows the **old** one (`r•••@example.org`).
2. Captured messages for the new address: "Confirm your new email address for Poised" with a `/v/…` link. Open it (any window, signed in or not) → **Confirm**.
**Expected:** "Your email address has been updated" with **Go to your questionnaires**. The home now shows `n•••@example.org`. Captured messages for the **old** address: "Your Poised contact details were changed".
3. As Cara, open that patient's episode at Hospital A1 (and as Bea at B1 if T-14 was done).
**Expected:** the patient's email is the new one at both hospitals. General notes unchanged; the trust's **Audit** page (as Ada Admin) shows "Patient changed their contact details".
4. Open the `/v/…` link again → Confirm. **Expected:** "This link has expired" (it works once).
5. `/signin` with the **new** email works; with the **old** one, nothing is sent.

### T-16 Change or add the mobile; refused inputs
1. On the home, **Add** (or **Change**) the mobile: `07700 900777` → Send a link to confirm. Captured messages for `+447700900777`: a text with a `/v/…` link; open it → Confirm → "Your mobile number has been updated"; the home shows `•••777`.
2. Try changing the email to the **current** one, to `hello`, and the mobile to `+1 202 555 0143`.
**Expected:** a red message each time ("That's already your email address.", a "real email/mobile" message, the "in this country" message). **Cancel** closes the box.

### T-17 Reload, deep links and Back
1. Reload `/home`: still signed in, same list.
2. Open `/home` in a new tab of the same phone window: works (same session).
3. From a questionnaire, use the browser's **Back**: returns to the home.
4. In a window with no session, open `/home`: "You've been signed out" with **Get a sign-in link** (no data).
5. Reload a `/s/…` page after the date of birth: you're on `/home` (Back doesn't return to the date-of-birth form).

### T-18 The rate limit per network (do this LAST)
All requests from this machine count as one network: after this case, sign-in links and contact changes are refused for an hour, until **Reset sign-in limits** is pressed on `/qa/messages`.
0. On `/qa/messages`, press **Reset sign-in limits** ("Sign-in limits cleared").
1. At `/signin`, ask for links for ten **different** made-up addresses (`ip1.<initials>@example.org` … `ip10…`).
2. Ask once more.
**Expected:** "There have been too many requests from this network. Please try again in an hour." Nothing sent.
3. Press **Reset sign-in limits** on `/qa/messages`, then ask once more. **Expected:** "Check your email" again.

### T-19 Twins on one phone stay two people (S15 answers 6) — phone size
1. As Cara, create **Kim Twin<initials>** and **Lee Twin<initials>**, both born `04/05/2016`, both with mobile `07700 900601`, no email (different first names, so no duplicate warning).
2. Open the newest text to `+447700900601` that is **Kim's** invite (Hospital A1; open each in turn, newest is Lee's) → Continue → `4 5 2016` → **← Your questionnaires**.
**Expected:** "Hello Kim", one card, **For Kim Twin…**; Lee's questionnaire isn't there.
3. Open Lee's invite in the same way. **Expected:** "Hello Lee", only Lee's card.
4. Sign out. `/signin` with `07700 900601` → open the text's sign-in link → Continue → `4 5 2016`.
**Expected:** a red message: "More than one person with this date of birth uses this mobile number. Please also enter your first name and surname." with **First name** (the cursor already in it) and **Surname** boxes. Enter `Lee` / `Someone` → "That date of birth and name don't match our records. Please check them and try again. You have 4 tries left." Enter ` LEE ` / `twin<initials>` in capitals → Lee's home only. A 101-character first name (paste one) can't be typed past 100 characters.
5. A different spelling stays apart: as Cara, create **Sammy Home<initials>** (T-01's date of birth and mobile, `17/05/1980`, `07700 900501`). Sign in as T-01's Sam (by text). **Expected:** Sam's home doesn't list Sammy's questionnaire.

### T-20 Adding a contact tells the existing one (S15 answers 7)
1. Signed in as a patient with an email but **no mobile** (e.g. T-03's), tap **Add** beside Mobile number, type `07700 900888` → Send a link to confirm. Open the text's `/v/…` link → Confirm.
**Expected:** "Your mobile number has been updated". Captured messages for the patient's **email**: "Your Poised contact details were added" ("The mobile number you use for Poised was added. If you didn't do this…").

### T-21 Each contact box takes only its own kind (S15 answers 7)
1. On the home, **Change** the email and type `07700 900555` → Send a link to confirm. **Expected:** "Enter an email address, for example name@example.com." Nothing sent.
2. **Change** the mobile and type `someone@example.org`. **Expected:** "Enter a mobile number." Nothing sent.

### T-22 A record that joins later gets the account's contact (S15 answers 7)
1. Use a **fresh patient** with no Trust B record: as Cara, create **Pat Fill<initials>** (`01/02/1985`, email `fill.<initials>@example.org`, no mobile); open the invite → date of birth → home; **Add** the mobile `07700 900883` and confirm it.
2. As Bea, create an episode at Hospital B1 for **Pat Fill<initials>**, `01/02/1985`, with only the **email**, no mobile.
3. Sign in as Pat by email (a new sign-in link) → the home lists the Hospital B1 questionnaire.
4. As Bea, open that episode. **Expected:** the patient's mobile is `+447700900883` (filled from their account); the email is unchanged. Max Multi's Audit page (Trust B's admin) shows "Patient's record joined their sign-in account".

### T-23 Same first name, birthday and phone, different surnames: two people (S15 answers 8) — phone size
1. As Cara, create **Kim Alpha<initials>** and **Kim Beta<initials>**, both `06/07/2015`, both mobile `07700 900631`, no email.
2. Open Kim Alpha's invite (the older of the two texts to `+447700900631`) → `6 7 2015` → **← Your questionnaires**. **Expected:** only Kim Alpha's card ("For Kim Alpha…").
3. Open Kim Beta's invite the same way. **Expected:** only Kim Beta's card.
4. Sign out. `/signin` with `07700 900631` → the sign-in link → Continue → `6 7 2015`. **Expected:** the name step (as T-19). `Kim` / `beta<initials>` → Kim Beta's home only.
5. The same person typed differently joins: as Bea (Hospital B1), create ` kim ` / `ALPHA<initials>` (extra spaces, other case), `06/07/2015`, mobile `07700 900631`. Sign in as Kim Alpha (name step: `Kim` / `Alpha<initials>`). **Expected:** her home lists both Hospital A1 and Hospital B1, and greets **Hello Kim** (capitalised).
6. A link to a contact changed since: ask for a sign-in link for T-03's patient's email, then (before opening it) change that patient's email from their home and confirm the change. Open the first link → Continue → the right date of birth. **Expected:** "This link has expired…", not "date of birth doesn't match".

### T-24 Leaving for the home straight after typing keeps the answer (verification 1) — phone size
1. As Cara, create a patient with an email (any details) on a questionnaire with a **text** question (the seeded one, or any with a free-text box). Open the invite → date of birth → open a section with a text box.
2. Type `Warfarin 5mg` and, **at once** (within half a second, the cursor still in the box), tap **← Your questionnaires**.
**Expected:** the home shows that questionnaire **In progress** straight away (no reload needed). Open it again: the text is there.
3. In that section, as Cara on the episode press **Resend invite** (this ends the patient's session). As the patient, change the text to `Warfarin 3mg` and at once tap **← Your questionnaires**.
**Expected:** the page doesn't go to the home: "You've been signed out" with the orange **Not saved** box naming the section ("Your latest change in … wasn't saved."), and **Get a sign-in link**.

## Regression (earlier sessions)

- **R-1 (S14):** a new invite: Continue shows only the hospital's name; a wrong date says "4 tries left"; the right one opens the questionnaire; fill in and **Send my answers** → "Thank you, Sam". On the home that questionnaire is then **Sent** with **View**.
- **R-2 (S14):** five wrong dates lock an invite link; Cara sees "The link is locked" and **Resend invite**; the old link "doesn't work", the new one opens.
- **R-3 (S14):** Cara's **Fill in for the patient** (PAT-09) still works, marked "entered by".
- **R-4 (S13):** New episode search at Hospital A1 finds only A1's patients; another trust's patient is never shown.
- **R-5 (S11):** Ada Admin's **Audit** page lists "Patient opened their home" rows for Trust A's hospitals after a patient opens their home (Trust B's rows appear only in Trust B's log, as Max Multi).
- **R-6 (S10):** suspend Trust B (Sam Superadmin, `/admin`): T-01's home then lists only Hospital A1; re-activate it afterwards. *A hand check for Rahul (test report 1: the tester's permissions refuse Suspend); mark it Blocked if refused again.*

## Out of scope (don't report)

- An unjoined twin's sign-in reaching the other twin's account (an accepted risk, answers 10).

- Reminders and "not responding" (S16); procedures, statuses and closing episodes (C3); a logo, files or videos on the home (later plans).
- Real text messages or emails (captured only until S17).
- A patient who has **never** opened an invite can't use `/signin` (by design: their invite link works until the episode closes).
- Removing a mobile or email altogether (only change or add).
- The browser's Back, a reload or closing the tab within ~0.6 s of typing in a questionnaire (S14's X-1, skipped by Rahul).
- Times shown in the browser's time zone; "Sent to you" is the date the episode was created.
