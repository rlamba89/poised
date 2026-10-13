# S14 manual test plan: patient invites and magic-link sign-in

For a tester who hasn't seen the code. Use only the browser.

**Changed after verification 2:** leaving the patient's form no longer drops the answer just typed (T-21, T-22). **For the retest, run T-21 and T-22 (round 4's mobiles and `R4` surnames), plus a quick T-06 step 6 and T-07.**

**Changed after verification 1:** a patient whose session ended mid-form is now told what wasn't saved (T-19); a failed Resend no longer breaks the working link (automated: it needs a failing message provider); the captured-messages page's **To** placeholder is country-neutral (T-20). **For the retest, run T-19 and T-20 (round 3's mobiles and `R3` surnames), plus a quick T-04, T-07 and T-15.**

**Changed after test report 1 (all passed):** Rahul decided on its "found while exploring" items 1 and 3. **Patient sessions are now held on the server**, tied to the invite link they were opened with: **Sign out**, a **Resend** or a **lock** ends an open session at once (even a copied cookie stops working), and the cookie no longer holds the patient's record id. Patient responses are **never cached** (`Cache-Control: no-store`) and the app's page has **no-referrer**. New cases **T-15 to T-18**. **For the retest, run T-15 to T-18 (round 2's mobiles and `R2` surnames), plus a quick T-02, T-04, T-06 (step 6) and T-07.**

**What S14 built:**
1. **Creating an episode sends the patient an invite** by **text message (SMS)** and **email**, whichever the patient's record has. Nothing is really sent here: every message is **captured** and shown on the **captured-messages page** (`/qa/messages`). Staff can **Resend invite** from the episode.
2. **The patient's link** (`/i/<code>`) shows **nothing about the patient** by itself. The patient taps **Continue** (they then see only the hospital's name), then enters their **date of birth**. The right date opens their questionnaire; **five wrong dates lock the link** (staff must resend). The old "patient link with a Copy button" is gone.
3. **The patient's questionnaire** (`/home/episodes/<id>`) works as before (sections, saving as you go, Send my answers) but shows **no date of birth, NHS number, mobile or email**. They stay signed in for 30 minutes after their last action, and can **Sign out**.
4. **Staff can fill in the questionnaire on the patient's behalf** (e.g. on the phone). Those answers are marked **"entered by <staff member>"** on the episode, the Validate page and the POA Summary.

## Setup

**Rahul, in your own terminal** (the automatic run does this itself):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

**Open:** http://localhost:3000. It goes to the development sign-in page: click a person to sign in as them. **Sign out** is in the page header.

**A second person at the same time:** open **http://tester.localhost:3000** in another tab. It's the same app but keeps its own sign-ins, so you can be two people at once.

**Seeded people:**

| Person | What they are |
| --- | --- |
| **Cara Clinician** | Clinician at **Hospital A1** only (Trust A) |
| **Val Clinician** | Clinician at **Hospital A2** only (Trust A) |
| **Ada Admin** | Admin of the whole of Trust A (sees **Settings** and **Audit**) |
| **Bea Author** | Super clinician of the whole of Trust B |
| **Max Multi** | Clinician at Hospital A2 **and** admin of Trust B (Hospital B1) |
| **Sam Superadmin** | The platform admin (the console at `/admin`) |

**Published questionnaire:** use **"Trust A demo HQ"** (v1). Its patient sections are **About you** ("Do you smoke?", then "How often do you drink alcohol?" on a second page) and **Your health** ("Do you have diabetes?"). It also has a clinician-only section, **Nurse checks**, which the patient must never see.

**Dev-only helper: the captured-messages page, http://localhost:3000/qa/messages.** No sign-in. It lists the newest texts and emails, newest first. Type a mobile (as `+447700900…`) or an email in **To** and click **Show** to see only that person's. The link in each message is clickable.

**The patient's phone:** the patient's pages are for phones. Use a window **390 × 844** (resize the browser window, or the device toolbar) for every patient step. The patient's link opens on **http://localhost:3000/i/…**; to be the patient in a **fresh browser** (no session yet), open the same path on **http://tester.localhost:3000/i/…** instead, or use a private window.

**Made-up data only.** Every patient below is **Sam**, born **17 May 1980** (enter it as day `17`, month `5`, year `1980`), with a surname you make unique: **add your round to every surname** (e.g. `Invite R1`). **Mobiles:** Ofcom's drama range only, and **use your round's set** so a retest never meets an earlier round's messages:

| Round | M1 | M2 | M3 | M4 | M5 |
| --- | --- | --- | --- | --- | --- |
| 1 | 07700 900101 | 07700 900102 | 07700 900103 | 07700 900104 | 07700 900105 |
| 2 | 07700 900201 | 07700 900202 | 07700 900203 | 07700 900204 | 07700 900205 |
| 3 | 07700 900301 | 07700 900302 | 07700 900303 | 07700 900304 | 07700 900305 |
| 4 | 07700 900401 | 07700 900402 | 07700 900403 | 07700 900404 | 07700 900405 |

They're stored in the international format: `07700 900101` becomes `+447700900101` (type that in the captured-messages page's **To**). **Emails:** `s14.r<round>.<letter>@example.org`, e.g. `s14.r1.a@example.org`.

**Creating an episode** (used by many cases): as **Cara Clinician**, **Episodes → New episode**. Ignore the search; under **New patient** type First name `Sam`, the surname, Date of birth `1980-05-17`, Sex any, and the Mobile and/or Email the case gives (leave NHS number empty). Under **Episode** choose **Trust A demo HQ v1**, then **Create episode**. You land on the episode's page.

## Test cases

### T-01 Creating an episode sends a text and an email (PAT-03)

1. As Cara, create an episode for `Sam Invite R<n>` with mobile **M1** and email `s14.r<n>.a@example.org`.
2. On the episode page, look at **General notes** and the **Patient's invite** card.
3. Open **http://localhost:3000/qa/messages** (another tab). In **To** type `+447700900<n>01` → **Show**. Then type the email → **Show**.

**Expected:**
- General notes: "Episode created", then **"Invite sent by text message and email."**, both by Cara Clinician.
- **Patient's invite** card: "Last sent <date, time>…", buttons **Resend invite** and **Fill in for the patient**. There is **no** "Patient link" box or **Copy** button any more.
- The **text message** to `+447700900<n>01` reads exactly **"Hospital A1: please complete your pre-op health questionnaire: http://localhost:3000/i/…"** (a long code at the end).
- The **email** to the address has the subject **"Hospital A1: your health questionnaire"**, starts "Hello Sam," and ends with the **same link** as the text.

### T-02 The link, Continue, four wrong dates of birth, the right one, fill in and send (phone, PAT-04)

Use T-01's episode. At **390 × 844**:
1. Open the link from T-01's text message.
2. Look at the page, then tap **Continue**.
3. Enter a wrong date of birth (e.g. `18` `5` `1980`) and tap **Continue**. Repeat with three more wrong dates (4 wrong in all), reading the red message each time.
4. Enter the right one: `17` `5` `1980` → **Continue**.
5. Look at the questionnaire page. Then fill in **About you** (both pages) and **Your health**, and **Send my answers** → confirm.

**Expected:**
- Step 1: "Your health questionnaire … Your hospital has asked you to fill in a health questionnaire…" and **Continue**. **Nothing about the patient**: no name, no hospital, no questionnaire name.
- Step 2: **"Confirm it's you"**, "**Hospital A1** needs your date of birth to make sure it's you.", three boxes **Day**, **Month**, **Year**. The bar at the top says Poised and **Hospital A1**. Still no name.
- Step 3: each wrong date shows "That date of birth doesn't match the hospital's records… You have **4 tries left**." then **3**, **2**, **1 try**.
- Step 4: the questionnaire opens (the address becomes `/home/episodes/…`): the questionnaire's name, "Hello Sam.", **About you** and **Your health** (never **Nurse checks**), "0 of 2 sections complete", **Sign out** in the top bar.
- **Nowhere** on these pages: the date of birth (17 May 1980 / 1980), the mobile, the email, an NHS number or a hospital number.
- Step 5: "Thank you, Sam". On Cara's episode page (reload) the status is **Ready for review** and the **Patient's invite** card is gone.
- Nothing is wider than the phone's screen; no sideways scrolling.

### T-03 Five wrong dates of birth lock the link (PAT-05)

1. As Cara, create an episode for `Sam Locked R<n>` with mobile **M2** (no email). Find its text on the captured-messages page (`+447700900<n>02`).
2. At 390 × 844, open the link → **Continue** → enter a wrong date 5 times.
3. Now enter the **right** date (`17` `5` `1980`).
4. Open the same link again → **Continue**.
5. As Cara, reload the episode's page.

**Expected:**
- The 5th wrong date shows **"This link is locked"**: "…entered wrongly too many times. Please contact the hospital to send you a new link." (no date boxes any more).
- Step 3 isn't possible from that screen; step 4 shows **"This link is locked"** straight after Continue, even though the right date was never tried.
- Step 5: the **Patient's invite** card shows a red **"The link is locked"** box: "The date of birth was entered wrongly 5 times (<time>). Resend the invite…".
- General notes for this episode say "Invite sent by text message." (no email: the patient has none).

### T-04 Resend the invite (PAT-03, PAT-05)

Continue with T-03's episode.
1. As Cara, click **Resend invite** on the **Patient's invite** card.
2. On the captured-messages page, show `+447700900<n>02` again.
3. At 390 × 844, open the **old** (first) link → **Continue**.
4. Open the **new** link → **Continue** → `17` `5` `1980`.

**Expected:**
- Step 1: a green "Invite sent again by text message." above the buttons; the red locked box is gone; "Last sent" shows the new time; General notes gain "Invite sent again by text message." by Cara.
- Step 2: a **second** text, newest first, with a **different** link.
- Step 3: **"This link doesn't work"** ("…Please check it, or contact the hospital.").
- Step 4: the questionnaire opens.

### T-05 The link alone shows no data (a fresh browser)

1. Copy the address of the questionnaire page from T-04 step 4 (`…/home/episodes/<id>`).
2. Open it in a **fresh browser**: change `localhost` to **`tester.localhost`** (or use a private window).
3. In that fresh browser, open T-04's **new** link (again on `tester.localhost`, i.e. `http://tester.localhost:3000/i/…`). Look at the page, then **Continue**.

**Expected:**
- Step 2: **"You've been signed out"** ("For your security, you've been signed out. Please open the link in your text message or email again."). No questionnaire, no name, no answers.
- Step 3: before Continue, nothing about the patient; after Continue, **only the hospital's name** and the date-of-birth boxes, until the right date is entered.

### T-06 Reload and Back in the middle of the flow (phone)

Create an episode for `Sam Reload R<n>` with mobile **M3**. At 390 × 844, with its link:
1. Open the link → **Continue** → on the date-of-birth screen, **reload** the page.
2. **Continue** → the right date → the questionnaire. **Reload**.
3. Open **About you**, answer "Do you smoke?", wait for **Saved**, then tap **All sections**. **Reload**.
4. Press the browser's **Back** button.
5. Open the link again → **Continue** → the right date.
6. Tap **Sign out** in the top bar. Then **reload**.

**Expected:**
- Step 1: back to the first screen with **Continue** (the date isn't remembered; nothing about the patient shows).
- Step 2: after the reload you're still in the questionnaire (signed in for 30 minutes after your last action).
- Step 3: **About you** shows **In progress**, and your answer is kept when you open it again.
- Step 4: Back leaves the questionnaire (to the page before the link, or nothing); it **never** shows the date-of-birth form again.
- Step 5: the questionnaire opens with the answer kept.
- Step 6: **"You've been signed out"** ("You've signed out. To come back, open the link…"); after the reload it still says "You've been signed out", with no questionnaire.

### T-07 Staff fill in the questionnaire on the patient's behalf (PAT-09)

1. As Cara, create an episode for `Sam Phone R<n>` with email `s14.r<n>.b@example.org` (no mobile).
2. On the **Patient's invite** card, click **Fill in for the patient**.
3. Fill in **About you** and **Your health**, then **Send the answers** → confirm in "Send the answers?".
4. Click **Back to the episode**.
5. Click **Validate** on **About you**. Go back, then open **POA Summary**.

**Expected:**
- Step 2: the address ends `/episodes/<id>/patient-form`; a blue box says "You're filling in **Sam Phone R<n>**'s health questionnaire for them. The answers are saved as theirs, marked as entered by you."; the patient's sections only (no **Nurse checks**).
- Step 3: **"Sent for Sam Phone R<n>"** … "The answers are with the team for review." and **Back to the episode**.
- Step 4: status **Ready for review**. Health questionnaire card: "Sent by Cara Clinician on the patient's behalf on <date>", and under **About you** and **Your health**: **"Patient's answers entered by Cara Clinician"**. General notes: **"HQ completed on the patient's behalf"** by Cara Clinician.
- Step 5: the Validate page has an orange box "The patient's answers here were entered by Cara Clinician on the patient's behalf." The POA Summary's section headings say **"Patient (entered by Cara Clinician) on <date>"** on the Patient tab.

### T-08 Patient and staff each fill a section: only staff's is marked

1. As Cara, create an episode for `Sam Mixed R<n>` with mobile **M4**.
2. As the patient (390 × 844, its link, Continue, right date): fill in **About you** only. Leave it.
3. As Cara: **Fill in for the patient** → fill in **Your health** → **Send the answers**.
4. Back to the episode.

**Expected:** **Your health** says "Patient's answers entered by Cara Clinician"; **About you** has **no** "entered by" line. The patient opening their link again sees "Thank you, Sam".

### T-09 Who gets which message (PAT-03, PAT-11)

As Cara, create three episodes:
1. `Sam Email R<n>` with **email only** `s14.r<n>.c@example.org`.
2. `Sam Abroad R<n>` with mobile **`+1 202 555 0123`** (an American number) and email `s14.r<n>.d@example.org`.
3. `Sam Nobody R<n>` with **no mobile and no email**.

**Expected:**
- 1: General notes "Invite sent by email."; one email on the captured-messages page, no text.
- 2: General notes **"Invite sent by email. No text message: texts go only to mobile numbers in this country."**; the captured-messages page has the email but **no text to `+12025550123`**.
- 3: General notes **"No invite sent. The patient has no mobile or email."**; the card says "No invite has been sent. The General notes say why." and its button (**Send invite**) is greyed out; **Fill in for the patient** still works.

### T-10 Refused: other hospitals, other trusts, other patients, signed out

Use T-02's or T-04's episode id (from Cara's episode address, `…/episodes/<id>`).
1. As **Val Clinician** (Hospital A2), open `http://localhost:3000/o/<Trust A id>/h/<Hospital A1 id>/episodes/<id>/patient-form` (Cara's address with `/patient-form` added).
2. As **Bea Author** (Trust B), open the same address.
3. Sign out of staff entirely, then open the same address.
4. As the patient of T-06 (signed in on the phone), change the address to `/home/episodes/<T-07's episode id>` (another patient's).
5. Open `http://localhost:3000/p/anything` (the old style of link).

**Expected:**
- 1 and 2: a "not found / no access" page; no questionnaire, no patient name.
- 3: the sign-in page.
- 4: an error ("Episode not found."), no questionnaire or name.
- 5: Page not found.

### T-11 The hospital's own invite wording (S10 templates)

1. As **Ada Admin**, open **Settings**. Under the hospitals, set **Hospital A1**'s **Invitation text message** to `{{hospital}} test R<n>: {{link}}` and save.
2. As Cara, create an episode for `Sam Template R<n>` with mobile **M5**.
3. Look at its text on the captured-messages page.
4. As Ada, put Hospital A1's **Invitation text message** back (clear it, so it follows the trust) and save.

**Expected:** step 3's text reads "Hospital A1 test R<n>: http://localhost:3000/i/…". The trust's default **Invitation text message** on Settings now reads "{{hospital}}: please complete your pre-op health questionnaire: {{link}}" (PAT-03's wording; changed in S14).

### T-12 The audit log records the patient's link and staff on-behalf actions

1. As **Ada Admin**, open **Audit**. **Who: Patients (their own link)** → **Show**.
2. **Who: Cara Clinician** → **Show**.

**Expected:**
- 1: rows for the patients above: **"Patient opened their link"**, **"Patient entered their date of birth"** (the wrong ones marked **Refused (401)**, the locking one **Refused (403)**), "Patient opened their questionnaire", "Patient saved answers", "Patient sent their answers". No address on any row contains the link's code.
- 2: among Cara's: **"Resent the patient's invite"**, **"Opened the patient's HQ to fill in for them"**, **"Saved answers for the patient"**, **"Sent the HQ for the patient"**, each naming the patient.

### T-13 Phone layout of every patient screen (390 × 844)

Walk through the first screen, the date-of-birth screen (with a red message showing), the locked screen, "This link doesn't work", the questionnaire list, a section, the "Send your answers?" box, "Thank you", and "You've been signed out".

**Expected:** no sideways scrolling; the three date boxes fit on one line; buttons are full-size and tappable; the top bar shows Poised, the hospital's name (once known) and Sign out without overlapping.

### T-14 A suspended trust closes the link (ORG-03)

1. As **Max Multi** (Trust B admin; pick **Hospital B1**), create an episode on **Trust B demo HQ** for `Sam Suspended R<n>` with email `s14.r<n>.e@example.org`.
2. As **Sam Superadmin**, in `/admin`, **suspend Trust B**.
3. As the patient, open that email's link → **Continue**.
4. As Sam Superadmin, **re-activate Trust B**. Open the link again → **Continue**.

**Expected:** step 3 shows **"Not available at the moment"** ("This questionnaire isn't available at the moment. Please contact the hospital."); step 4 shows "Hospital B1 needs your date of birth…".

### T-15 A Resend ends a session that's already open (new after report 1)

1. As Cara, create an episode for `Sam Ended R<n>` with mobile **M1** and email `s14.r<n>.f@example.org`.
2. At 390 × 844, open its link → **Continue** → the right date. The questionnaire shows. Open **About you** and answer "Do you smoke?" (wait for **Saved**), then **All sections**.
3. As Cara, click **Resend invite** on the episode.
4. Back on the phone, **reload** the questionnaire (or open a section and change an answer).
5. Open the **new** link from the captured messages → Continue → the right date.

**Expected:**
- Step 4: **"You've been signed out"** (no questionnaire), because the link the session was opened with has been replaced.
- Step 5: the questionnaire opens; the "Do you smoke?" answer from step 2 is kept.

### T-16 A lock ends a session that's already open (new after report 1)

1. As Cara, create an episode for `Sam LockEnds R<n>` with mobile **M2**.
2. At 390 × 844 on **http://localhost:3000**, open its link → Continue → the right date. The questionnaire shows.
3. In a **fresh browser** (the same link on **http://tester.localhost:3000/i/…**, or a private window), tap Continue and enter a **wrong** date 5 times, until "This link is locked".
4. Back in the first browser, **reload** the questionnaire.

**Expected:** step 4 shows **"You've been signed out"**. Cara's episode page shows the red **"The link is locked"** box; **Resend invite** gives a new link that works.

### T-17 Sign out ends the session on the server, even for a copied cookie (new after report 1; a probe, as in report 1)

1. As Cara, create an episode for `Sam CopiedCookie R<n>` with mobile **M3**. Open its link, Continue, the right date.
2. In the browser's developer tools (Application → Cookies → `http://localhost:3000`), copy the value of the **`poised_patient`** cookie (path `/api/p`). Note what it holds: paste its middle part (between the two dots) into a base64 decoder if you can.
3. Tap **Sign out** in the questionnaire's top bar.
4. Send `GET http://localhost:3000/api/p/episodes/<episode id>` with that **copied** cookie (e.g. `curl -i -H 'Cookie: poised_patient=<value>' …`).

**Expected:**
- Step 2: the decoded part has `"aud":["patient"]`, an expiry and a `sub` that is **not** the patient's record id (the id in Cara's `…/patients/<id>` address).
- Step 4: **401** "For your security, you've been signed out…", not the questionnaire.
- *If your tools can't read or send the cookie, mark this Blocked: an automated test covers it.*

### T-18 Nothing cached, no referrer (new after report 1)

1. On a patient page (the link's first screen, or the questionnaire), check the network responses of `/api/p/…` requests (Continue, the date of birth, the questionnaire, a save).
2. On the link's first screen (`/i/…`), run `document.querySelector('meta[name=referrer]').content` in the console (or view the page source).

**Expected:**
- 1: every `/api/p/…` response has the header **`Cache-Control: no-store`** (also the 401s).
- 2: **`no-referrer`**.
- *If your tools can't show response headers, mark step 1 Blocked: an automated test covers it.*

### T-19 A change refused because the session ended is named (new after verification 1)

1. As Cara, create an episode for `Sam Lost R<n>` with mobile **M4**.
2. At 390 × 844, open its link → Continue → the right date. Open **About you**, answer "Do you smoke?" **No**, and wait for **Saved**.
3. As Cara, click **Resend invite** on the episode.
4. On the phone, change the answer to **Yes**.
5. Open the **new** link from the captured messages → Continue → the right date → **About you**.
6. *(Send)* Answer every question in both sections (they save), then as Cara **Resend invite** again, then on the phone tap **Send my answers** → confirm.

**Expected:**
- Step 4: **"You've been signed out"** with an orange **Not saved** box: "Your latest change in **About you** wasn't saved. When you open the link again, please check your answers in that section.", and below it "…Please open the link in your text message or email again."
- Step 5: "Do you smoke?" shows **No** (the change was indeed not saved, as the box said).
- Step 6: "You've been signed out" with "Your answers haven't been sent yet. When you open the link again, check them and tap Send my answers." Cara's episode is still **HQ not complete**.

### T-20 The captured-messages page's placeholder (new after verification 1)

Open http://localhost:3000/qa/messages.

**Expected:** the **To** box's hint reads "A mobile number (as stored, with its country code) or an email address": no UK number.

### T-21 Sign out straight after typing keeps the answer (new after verification 2)

1. As **Alex Author**, create and publish a small questionnaire `Medicines R<n>`: one Question Set (audience **Patient**) with a **text** question "What medicines do you take?" and a **Yes/No** question "Do you take blood thinners?" (neither required).
2. As Cara, create an episode on it for `Sam Typing R<n>` with mobile **M5**.
3. At 390 × 844, open the link → Continue → the right date → the section. Type `Warfarin 5mg` in the text box and, **without tapping anywhere else**, tap **Sign out** at once.
4. Open the link again → Continue → the right date → the section.
5. Tap **Yes** for blood thinners and, within a second, tap **Sign out**. Open the link again and the section.

**Expected:** step 3 shows "You've been signed out" with **no** Not saved box; step 4 shows `Warfarin 5mg`; step 5 shows **Yes** kept. (While typing, the status may briefly say "Saving…".)

### T-22 Leaving the on-behalf form straight after typing keeps the answer (new after verification 2)

1. As Cara, create an episode on `Medicines R<n>` for `Sam Leaving R<n>` (email `s14.r<n>.h@example.org`). Click **Fill in for the patient**, open the section, type `Aspirin 75mg` and click **← Episode** at once.
2. Open **Fill in for the patient** again and the section.

**Expected:** step 1 lands on the episode, whose Health questionnaire card says "Patient's answers entered by Cara Clinician"; step 2 shows `Aspirin 75mg`.

## Regression (earlier sessions)

- **R-01 (S13) New episode search:** as Cara, **New episode**, search `Invite R<n>`: T-01's patient is found; choosing them and creating a second episode works, and that episode also sends an invite.
- **R-02 (S02/S04) Validate and complete the review:** on T-02's episode, validate all three Question Sets (change one answer) and **Complete HQ review**; the status becomes **Ready for POA**.
- **R-03 (S11) Audit filters:** filter by Patient `Invite R<n>`; only that patient's rows show.
- **R-04 (S08) Hospital switch:** as Ada, switch from Hospital A1 to A2 on an Episodes page; A2's episodes show.
- **R-05 (S10) Settings save:** as Ada, change and restore a hospital's phone number on **Settings**.

## Out of scope (don't report)

- **Real texts and emails:** nothing is sent; messages are only captured (sending is S17, on AWS).
- **The 30-minute idle timeout itself:** an automated test covers it; don't wait 30 minutes.
- **Send with no answers through the API directly** (report 1, item 4): the API can't check required answers until C6; logged in pending.md.
- **"Entered by" staying on a set the patient later changes, and the bare "Episode not found." page for someone else's episode** (report 1, items 5 and 7): logged in pending.md.
- **Sign-in links by mobile or email, the patient's home page (`/home`) and their other trusts' episodes:** S15. `/home` alone is "Page not found" for now.
- **Reminders and "not responding":** S16.
- **Changing a patient's mobile or email:** not built yet (S15).
- **An episode "closing":** no status closes an episode yet (C3), so an invite link works for as long as the episode exists.
- **The captured-messages page has no sign-in:** by design; it exists only where nothing is really sent.
- **Expected console errors:** a 401 after each wrong date of birth or on "You've been signed out", 403 on a locked link, 404 on a dead link; two 401s from `/api/me` on the staff sign-in page.
- **Every clinician at the hospital sees Fill in for the patient:** by design (everyone at a hospital is at least a clinician).
- The Claude in Chrome network log shows a `204 No Content` (e.g. Send my answers, Sign out) as 503: ignore it.
