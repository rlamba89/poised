# S16 test plan: scheduled jobs, reminders and "not responding"

**Changed for retest 1 (after test report 1, X-1):**
- **Wrong dates of birth now count per episode:** the invite's link and every reminder's link share **5 tries**, and the 5th locks **all** of them. A right date on any link starts the count again. **Only Resend invite** clears a lock. **A locked episode gets no reminders** (the General notes say "Day N reminder not sent: the patient's link is locked…"); "Not responding" still comes on day 7. New cases **T-16 to T-18**.
- The episode's lock banner now says "…so every link sent to the patient for this episode is locked, and no reminders go. Resend the invite…".
- **T-14** adds a half-typed date: it now says "Enter a date and time." (it used to do nothing).
- Not to report (logged for Rahul): the reminder arriving at the same moment as "Not responding" on catch-up; double-clicking Run jobs; real vs "as of" times in messages and notes.

For a tester who uses only the browser. Patient pages are tested at **phone size (390×844)**; staff pages at desktop size.

**What S16 adds:** after a patient's invite, the app sends **reminders** on the hospital's reminder days (days **2 and 5** after the first invite, unless the hospital's settings say otherwise) and marks the episode **"Not responding"** on day **7** if the questionnaire (HQ) hasn't been sent. Rahul's rules: **reminders go at any time of day** (no quiet hours), and **they stop once the patient starts answering** (submitting stops everything). Nothing is really sent: texts and emails are captured.

You can't wait days, so a dev/QA-only page, **Scheduled jobs** (`/qa/jobs`), runs the jobs **as of a later time**, or moves the local timer's clock ahead.

## Setup

1. In your own terminal (the conductor does this in an automatic run):
   ```sh
   docker compose up -d db
   make migrate seed
   make dev
   ```
   Leave `make dev` running. `make seed` also sets the jobs timer's clock back to real time.
2. Open **http://localhost:3000**.
3. **Staff sign-in** (the stub login page at http://localhost:3000/login, pick a name):
   - **Cara Clinician**: clinician at **Hospital A1** (Trust A).
   - **Val Clinician**: clinician at **Hospital A2** (Trust A).
   - **Ada Admin**: admin of the whole of Trust A (sees **Settings**).
   - **Bea Author**: super clinician of **Trust B** (Hospital B1).
   For two people at once, use **http://tester.localhost:3000** in the same browser: it keeps its own cookies (good for "the patient's phone" beside staff).
4. **Dev/QA helper pages (no sign-in; they exist only because messages are captured):**
   - **Captured messages:** http://localhost:3000/qa/messages lists every text and email the app "sent". Type a mobile (as stored, e.g. `+447700900601`) or an email in **To**, then **Show**. Links in messages are clickable; copy the path (`/i/…`) into the patient's browser. It links to Scheduled jobs.
   - **Scheduled jobs:** http://localhost:3000/qa/jobs.
     - **Run the jobs now, as of a later time:** the buttons **Now**, **Now + 2 days**, **Now + 5 days**, **Now + 7 days** fill the **Run as of** box (your time zone, rounded up to the next minute); you can also type any date and time. **Run jobs** then says e.g. "Ran 1 job as of …: 1 reminder sent, 0 marked "not responding"."
     - **The timer:** `make dev` runs the jobs by itself **every 60 seconds**. The card shows its last run. **Move the clock** (days and hours ahead) makes the timer act as if that much time had passed, from its next run; **Back to real time** undoes it.
   - **Important:** a run (or a moved clock) acts on **every** episode in the app, not only yours. So the counts in "Ran N jobs…" may include other episodes: **judge each case by your own patient's messages and episode**, not by the counts. **Always press Back to real time** after a case that moves the clock.
5. **Made-up data only.** Mobiles: Ofcom's drama range, `07700 900000`–`07700 900999`. Emails: anything `@example.org`. Use a **new surname per case** (e.g. `Rem01<your initials>`) and the mobile the case gives, so your messages are easy to find.
6. **Creating an episode** (most cases): as the named clinician, **Episodes → New episode** → new patient: first name `Sam`, the case's surname, date of birth `17/05/1980`, any sex, the case's mobile and/or email; any published health questionnaire; **Create episode**. The episode page opens; its General notes say "Invite sent by …".
7. **Days count from the invite**, in the hospital's time zone: "Now + 2 days" right after creating an episode is day 2. "Now + 5 days" and "Now + 7 days" likewise.

## Test cases

### T-01 Day 2: the first reminder, by text and email
1. As **Cara**: create an episode for `Sam Rem01<initials>`, mobile `07700 900601`, email `rem01.<initials>@example.org`.
2. Open **/qa/jobs** → **Now + 2 days** → **Run jobs**.
3. Captured messages, To `+447700900601`, then To your email.
**Expected:** step 2's result is green and starts "Ran … as of <two days from now>". For your patient there are now **two texts**: the invite ("Hospital A1: please complete your pre-op health questionnaire: …/i/…") and, newest first, the reminder: **"Hello Sam, a reminder from Hospital A1: please fill in your health questionnaire: …/i/…"**. Likewise **two emails**, the newest with the subject **"Hospital A1: a reminder about your health questionnaire"** and the body "Hello Sam, This is a reminder from Hospital A1: please fill in your health questionnaire before your operation: …". The reminder's link is **different** from the invite's.

### T-02 Nothing is sent twice
1. Straight after T-01, on /qa/jobs, **Now + 2 days** → **Run jobs** again. Then click **Run jobs** twice quickly.
2. Type a time **4 days** from now in **Run as of** → **Run jobs**.
3. Captured messages for `+447700900601`.
**Expected:** still exactly **two** texts and **two** emails for this patient (no second reminder).

### T-03 The episode shows the reminder, written automatically
1. As Cara, open T-01's episode (Episodes → Sam Rem01…).
**Expected:** General notes end with **"Day 2 reminder sent by text message and email."** followed by **"| Automatic on <date, time>"** (not "Patient" or a staff name). The **Patient's invite** card says "Last sent <the reminder's time>" and "Reminders go on the hospital's reminder days after the first invite, until the patient starts answering. The General notes list each one." No **Not responding** badge yet.

### T-04 The reminder's link works, and so does the invite's — phone size
1. In a phone-size window (390×844), not signed in as staff (e.g. tester.localhost signed out, or a private window), open T-01's **reminder** link (`/i/…` from the newest text) → **Continue** → date of birth `17 5 1980`.
2. Note the questionnaire opens. Close it; open the **invite's** link (the older text) → Continue → `17 5 1980`.
**Expected:** both links open Sam's questionnaire ("Hello Sam."), with no sideways scrolling. **Don't answer anything** (T-05 needs this patient not started).

### T-05 Day 5: the second reminder; day 7: "Not responding"
1. /qa/jobs → **Now + 5 days** → **Run jobs**. Check captured messages for `+447700900601`.
2. **Now + 7 days** → **Run jobs**. Check captured messages again.
3. As Cara, open (or reload) T-01's episode.
**Expected:** after step 1, a **third** text and email (the reminder again, with another new link). After step 2, **no new** message. On the episode: a red **Not responding** badge next to the status ("HQ not complete"); General notes add **"Day 5 reminder sent by text message and email."** and **"Not responding: the HQ hasn't been sent 7 days after the invite."**, each "| Automatic". Running **Now + 7 days** again adds nothing.

### T-06 Sending the answers before day 5 stops the reminders
1. As Cara: create an episode for `Sam Rem06<initials>`, mobile `07700 900606`.
2. /qa/jobs → **Now + 2 days** → **Run jobs** (the day 2 reminder arrives: two texts).
3. Phone-size patient window: open the newest link → Continue → `17 5 1980` → answer every section → **Send my answers** → confirm.
4. /qa/jobs → **Now + 5 days** → Run; then **Now + 7 days** → Run.
5. As Cara, open the episode.
**Expected:** still only **two** texts for `+447700900606` (no day 5 reminder). The episode is **Ready for review**, with **no** Not responding badge and no "Not responding" note.

### T-07 Starting the answers stops reminders, but day 7 still says "Not responding"
1. As Cara: create an episode for `Sam Rem07<initials>`, mobile `07700 900607`.
2. Phone-size patient window: open the invite link → Continue → `17 5 1980` → **Start** the first section → answer **one** question → wait for it to save → close the page (don't send).
3. /qa/jobs → **Now + 2 days** → Run; **Now + 5 days** → Run.
4. Captured messages for `+447700900607`.
5. /qa/jobs → **Now + 7 days** → Run. As Cara, open the episode.
**Expected:** step 4 shows **only the invite** (no reminders: the patient started). After step 5 the episode shows **Not responding** and the note "Not responding: the HQ hasn't been sent 7 days after the invite." (they started but didn't send). No message was sent at step 5.

### T-08 Answers entered by staff for the patient count as started
1. As Cara: create an episode for `Sam Rem08<initials>`, mobile `07700 900608`.
2. On the episode, **Fill in for the patient** → answer one question → wait for it to save → **← Episode**.
3. /qa/jobs → **Now + 2 days** → Run. Captured messages for `+447700900608`.
**Expected:** only the invite: no reminder.

### T-09 A reminder due at night still goes (no quiet hours)
1. As Cara: create an episode for `Sam Rem09<initials>`, mobile `07700 900609`.
2. /qa/jobs: in **Run as of**, type the date **3 days from today** at **22:30** → **Run jobs**.
3. Captured messages for `+447700900609`.
**Expected:** the **day 2 reminder** was sent (two texts), even though 22:30 is late evening (Rahul: reminders go at any time). Only one reminder, not two (day 5 hasn't come).

### T-10 Missed days: only the latest reminder goes, once
1. As Cara: create an episode for `Sam Rem10<initials>`, mobile `07700 900610`.
2. /qa/jobs → **Now + 7 days** → Run (straight away, skipping days 2 and 5).
3. Captured messages for `+447700900610`; open the episode as Cara.
**Expected:** exactly **one** reminder (two texts in all, not three); the General notes say **"Day 5 reminder sent by text message."** and **"Not responding: …7 days…"**; the badge shows.

### T-11 The timer sends a due reminder by itself
1. As Cara: create an episode for `Sam Rem11<initials>`, mobile `07700 900611`.
2. /qa/jobs → The timer card: it says **On**, "every 60 seconds, as of real time", and a recent **Last run** (it moves on each minute; the page refreshes it every 15 seconds).
3. **Days ahead** `2`, **Hours ahead** `1` → **Move the clock**. The card now says "as of real time + 2 days 1 hour".
4. **Don't press Run jobs.** Wait up to 2 minutes, then check captured messages for `+447700900611`.
5. **Back to real time** (the card says "as of real time" again).
**Expected:** within about a minute the reminder text for `+447700900611` appears by itself, and the timer card's **Last run** shows "as of" about two days ahead. After step 5 the clock is back on real time. *(Other episodes may also get reminders while the clock is ahead; that's expected.)*

### T-12 A hospital's own reminder days and wording
1. As **Ada Admin**: **Settings** → choose **Hospital A2** → under "Reminders and archiving" set **Send reminders on these days after the invite** to `1` and **Mark "not responding" on day** to `3`; under "Patient messages" set **Reminder text message** to `{{hospital}} reminds {{firstName}}: {{link}}` → **Save**.
2. As **Val Clinician** (Hospital A2): create an episode for `Sam Rem12<initials>`, mobile `07700 900612`.
3. /qa/jobs: **Run as of** = tomorrow at the same time plus a few minutes → Run. Captured messages for `+447700900612`.
4. **Run as of** = 3 days from now plus a few minutes → Run. As Val, open the episode.
5. As Ada, put Hospital A2's three settings back to "follow the trust" (or the defaults 2, 5 / 7 / the default text) → Save.
**Expected:** step 3: a reminder **"Hospital A2 reminds Sam: …/i/…"** after **1** day. Step 4: **Not responding** with the note "…hasn't been sent **3** days after the invite." Hospital A1's episodes (T-01) used 2/5/7 and the default text.

### T-13 No contacts, no reminders; a suspended hospital waits
1. As Cara: create an episode for `Sam Rem13<initials>` with **no mobile and no email**. Its notes say "No invite sent. The patient has no mobile or email."
2. /qa/jobs → **Now + 7 days** → Run. Open the episode.
**Expected:** no reminder note and **no Not responding** badge (the follow-up starts only when an invite reaches the patient; staff already know it didn't).
*(Suspended hospitals are covered by automated tests; testers can't suspend without the platform admin's console, which is fine to skip.)*

### T-14 The jobs page: invalid input, reload, links
1. /qa/jobs: clear **Run as of** → **Run jobs**. Then type a full date and time, clear only the **day** part → **Run jobs**.
2. **Days ahead** `91` → **Move the clock**; then Days `0`, **Hours ahead** `-1` → Move the clock.
3. Reload the page; use the **Captured messages** link and the browser's Back button; on /qa/messages use its **Scheduled jobs** link.
**Expected:** step 1: red "Enter a date and time." both times (a half-typed date isn't silently ignored). Step 2: each time red "Enter whole days and hours, 90 days at most.", and the timer card still says "as of real time". Step 3: the page reloads fine; the links go to each other's pages and Back returns.

### T-15 A Resend doesn't restart the reminder days
1. As Cara: create an episode for `Sam Rem15<initials>`, mobile `07700 900615`.
2. On the episode, **Resend invite** (a second invite text arrives).
3. /qa/jobs → **Now + 2 days** → Run. Captured messages for `+447700900615`.
**Expected:** three texts in all: invite, resent invite, and the **day 2 reminder** (days still count from the first invite). The resent invite's link works; the first invite's link no longer does (Resend replaces it, as before S16); the reminder's link works.

### T-16 Wrong dates of birth on different links share the 5 tries — phone size
1. As Cara: create an episode for `Sam Rem16<initials>`, mobile `07700 900616`.
2. /qa/jobs → **Now + 2 days** → Run. Captured messages for `+447700900616`: the invite and the day 2 reminder (two links).
3. Phone-size patient window: open the **invite's** link → Continue → enter a wrong date of birth (e.g. `18 5 1980`) **3 times** (it says 4, 3, then 2 tries left).
4. Open the **reminder's** link → Continue → a wrong date of birth: it says **1 try left**. Once more wrong: **This link is locked**.
5. Open the **invite's** link again → Continue (or the right date `17 5 1980`).
6. As Cara, open the episode.
**Expected:** step 4 continues the count from step 3 (not a fresh 4). Step 5: **This link is locked** even with the right date. Step 6: red **The link is locked** box: "…so every link sent to the patient for this episode is locked, and no reminders go. Resend the invite to send them a new link."

### T-17 A locked episode gets no reminders; Resend unlocks
1. Continue with T-16's episode. /qa/jobs → **Now + 5 days** → Run.
2. Captured messages for `+447700900616`; as Cara, open the episode.
3. On the episode, **Resend invite**. Phone-size window: open the **resent** link → Continue → a wrong date once (it says **4 tries left**), then the right date `17 5 1980`.
4. /qa/jobs → **Now + 7 days** → Run; open the episode.
**Expected:** step 2: **no** new text (still two); General notes: **"Day 5 reminder not sent: the patient's link is locked after 5 wrong dates of birth. Resend the invite to send them a new link."** (Automatic). Step 3: the lock box goes; the resent link works with a fresh 5 tries; the old invite and reminder links now say "This link doesn't work". Step 4: **Not responding** (the HQ wasn't sent), and no further reminder (day 5's was skipped, not sent late).

### T-18 A right date on any link starts the count again
1. As Cara: create an episode for `Sam Rem18<initials>`, mobile `07700 900618`. /qa/jobs → **Now + 2 days** → Run.
2. Phone-size: the invite's link → 3 wrong dates (2 tries left). Then the **reminder's** link → the right date `17 5 1980` (the questionnaire opens).
3. In a fresh window (or after Sign out), the invite's link → Continue → one wrong date.
**Expected:** step 3 says **4 tries left** (the right date reset the episode's count).

## Regression (earlier sessions)

- **R-1 (S14) Invite on create:** creating an episode still shows "Invite sent by text message and email." and the captured messages show the invite; the patient opens it with Continue + date of birth.
- **R-2 (S14) Resend locks/unlocks:** five wrong dates of birth lock the link (an episode with no reminders yet); the episode shows "The link is locked"; **Resend invite** sends a new one that works.
- **R-6 (S15) Sign-in links keep their own tries:** at `/signin` ask for a link for a patient who opened an invite before; on the `/s/…` link, a wrong date of birth says **4 tries left** whatever their invite links' count is.
- **R-3 (S15) The patient's home:** a patient who opened a link sees `/home` with their questionnaire as **To do / In progress / Sent** (phone size).
- **R-4 (S13) Patient search:** New episode → search by surname finds a patient created above (at the same hospital).
- **R-5 (S11) Audit log:** as Ada, **Audit** shows "Viewed an episode" rows for the episodes you opened. (The jobs themselves write no audit rows; that's by design.)

## Out of scope (don't report)

- **No sign-in on /qa/jobs and /qa/messages**: by design; they exist only where messages are captured (local and QA).
- The run counts include other episodes in the app (see Setup 4).
- **Quiet hours (09:00–19:00)**: Rahul chose "any time", so a reminder at night is correct.
- The "Not responding" **worklist** (C3); only the badge on the episode is built. The episodes list doesn't show it.
- Real texts or emails, AWS, EventBridge (S17).
- Suspending a trust or hospital (automated tests cover it; it needs the platform admin console).
- `make dev` logs "jobs: ran N …" lines every minute when something ran: expected.
- On catch-up (e.g. **Now + 7 days** straight away, or after re-activating a hospital), the day 5 reminder and "Not responding" come in the same run: as designed, logged for Rahul.
- Double-clicking **Run jobs** sends two requests (nothing is sent twice); messages and automatic notes show the real time, not the "as of" time.
