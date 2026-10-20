# S18 test plan: episode forms (a full regression of the episode workflow)

**Corrected after test report 1 (all passed):** four expectations didn't match the agreed behaviour, and the app was right: T-01 (the Hospital column shows only in "All hospitals"), T-04 (no "sent by the patient" line on the card; the time is in General notes), T-13 (day 7 still marks "Not responding" once started, S16's rule), R-3 (the old link's S14 wording). No code changed.

For a tester who uses only the browser. Patient pages are tested at **phone size (390×844)**; staff pages at desktop size.

**What S18 changes:** nothing you can see. Behind the scenes, an episode's health questionnaire (HQ) and its answers now belong to a **form** of the episode, so later sessions can add more forms (assessments, PROMs) to one episode. The addresses and screens are the same as before.

**So this is a full regression:** every step of the episode workflow must work exactly as it did before S18. Report anything that behaves differently from what each case expects, any error message, any blank page, and any red error in the browser console other than the expected noise (two 401s from `/api/me` on the login page; 409s in the cases that expect "changed somewhere else").

**Expected, not a bug:** the S18 database change **removed every episode** made before it (they were all made-up test data). The Episodes list starts empty or holds only episodes made since. Patients, their records, questionnaires, staff and trusts are all still there.

## Setup

1. In your own terminal (the conductor does this in an automatic run):
   ```sh
   docker compose up -d db
   make migrate seed
   make dev
   ```
   Leave `make dev` running.
2. Open **http://localhost:3000**.
3. **Staff sign-in** (the stub login page at http://localhost:3000/login, pick a name):
   - **Cara Clinician**: clinician at **Hospital A1** (Trust A).
   - **Val Clinician**: clinician at **Hospital A2** (Trust A).
   - **Alex Author**: super clinician of the whole of Trust A (can publish questionnaires).
   - **Bea Author**: super clinician of **Trust B** (Hospital B1).
   For two people at once, use **http://tester.localhost:3000** in the same browser: it keeps its own cookies (good for "the patient's phone" beside staff).
4. **Dev/QA helper pages (no sign-in):**
   - **Captured messages:** http://localhost:3000/qa/messages lists every text and email the app "sent" (nothing really goes out). Type a mobile as stored (e.g. `+447700900801`) or an email in **To**, then **Show**. Links in messages are clickable; copy the path (`/i/…`, `/s/…`) into the patient's browser. Press **Reset sign-in limits** before T-07.
   - **Scheduled jobs:** http://localhost:3000/qa/jobs runs the reminder jobs **as of a later time** (buttons **Now + 2 days**, **Now + 5 days**, **Now + 7 days**, then **Run jobs**). A run acts on every episode in the app, so judge each case by **your own patient's** messages and episode, not by the counts. If you use **Move the clock**, press **Back to real time** afterwards.
5. **Made-up data only.** Mobiles: Ofcom's drama range, `07700 900000`–`07700 900999`; use the ones each case gives, with a **new surname per case** (e.g. `Form01<your initials>`) so your messages are easy to find. Emails: anything `@example.org`.
6. **The questionnaire to use (Trust A):** on **New episode**, pick a published HQ that has the Question Sets **About you**, **Your health** (the patient's) and **Nurse checks** (the clinician's), e.g. one named `S02 test HQ …` or `MT S03 1625 workflow`. If there's none, sign in as **Alex Author**, create a questionnaire with two patient Question Sets (one required question each, e.g. a Yes/No "Do you smoke?") and one clinician Question Set, and publish it. Write down its name; the cases call it **"the HQ"**.
7. **Creating an episode** (most cases): as Cara, **Episodes → New episode** → new patient: first name `Sam`, the case's surname, date of birth `17/05/1980`, any sex, the case's mobile and/or email; **the HQ**; procedure `Knee replacement`; **Create episode**. The episode page opens; its General notes say "Episode created" and "Invite sent by …".

## Test cases

### T-01 Create an episode: it shows its HQ (desktop)
1. As **Cara**: create an episode for `Sam Form01<initials>`, mobile `07700 900801`, email `form01.<initials>@example.org`.
2. Look at the episode page, then **Episodes** (the list).
3. Open **Patients** (or the patient search), find `Form01<initials>`, and open the patient's record.

**Expected:**
- Step 1: the episode page shows the patient's name, status **HQ not complete**, the HQ's name and version (v1), the **Patient's invite** card ("Last sent …"), and General notes "Episode created" by Cara and the invite's note.
- Step 2: the list shows the episode with the patient, the HQ's name and **v1**, status HQ not complete. (A Hospital column shows only in the trust-wide **All hospitals** view, not for a one-hospital clinician.)
- Step 3: the patient's record lists this episode with the HQ's name and v1.

### T-02 The patient signs in, answers half, leaves and comes back — phone size
1. Captured messages, To `+447700900801`: the invite text ("Hospital A1: please complete your pre-op health questionnaire: …/i/…").
2. In a **phone-size** window (390×844), not signed in as staff (tester.localhost signed out, or a private window), open the invite link → **Continue** → date of birth `17 5 1980` → **Continue**.
3. Fill in **About you** completely (e.g. Do you smoke? **Yes**). Wait for **Saved**.
4. Start **Your health**: answer one question only. Wait for **Saved**.
5. Close the tab. Open the invite link again in a new phone-size tab, Continue, date of birth.
6. Reload the page once more.

**Expected:** step 2: "Hello Sam", Hospital A1, the HQ's patient sections only (**no Nurse checks**). Steps 5 and 6: **every answer from steps 3 and 4 is still there**, and you can carry on. No sideways scrolling at any step.

### T-03 The patient home shows it in progress, then sent — phone size
1. In the phone window from T-02, tap **← Your questionnaires** (top left).
2. Note the card for Hospital A1. Open it again with **Continue**, finish **Your health**, then **Send the answers** → confirm.
3. Tap **← Your questionnaires** again (or open `/home`).

**Expected:**
- Step 1: `/home` shows **Hello Sam** and a card for **Hospital A1** (Procedure: Knee replacement), the HQ's name, **For Sam Form01<initials>**, badge **In progress**, button **Continue**.
- Step 2: a thank-you page ("Thank you, Sam" or similar: the answers are with the team).
- Step 3: the card's badge is now green **Sent** and its button **View** (not "In progress" or "To do").
- Opening the sent questionnaire again shows it as already sent; its answers can't be changed.

### T-04 After the patient's submit: Ready for review (desktop)
1. As **Cara**, open T-01's episode (reload it if open).

**Expected:** status **Ready for review**. General notes include **"HQ completed by the patient"** with no staff name, and its date and time. (The health questionnaire card has a "Sent by …" line only when staff sent it on the patient's behalf, T-07.) **Resend invite**, if the card still offers it, is refused with "The patient's answers have already been sent, so there's nothing to invite them to." and no new text arrives.

### T-05 Validate each Question Set, then complete the review
1. On T-01's episode, click **Validate** on **About you**.
2. Change **Do you smoke?** to **No**. Note the "Patient answered: Yes" hint. Click **Validate**.
3. Validate **Your health** with no changes.
4. Open **Nurse checks**, fill in its required questions, **Validate**.
5. Click **Complete HQ review**.

**Expected:** step 1 opens `/h/<Hospital A1>/episodes/<id>/sets/<set id>` with the patient's answers. Step 2: back on the episode page, About you shows **Validated by Cara Clinician** and the time. Steps 3–4 likewise. Step 5: status **Ready for POA**; General notes say "HQ review completed by Cara Clinician". The sets now open read-only ("The HQ review is complete, so these answers can't be changed.").

### T-06 The POA Summary shows both actors' answers, with the change marked
1. On T-01's episode, open **POA Summary**.
2. Look at **About you** in **Validated summary**, then open the **Patient answers** tab.
3. Reload the page.

**Expected:** the POA Summary at `/h/<Hospital A1>/episodes/<id>/poa`. **Validated summary:** the smoking answer is Cara's **No**, **marked as changed by the clinician** (showing what the patient said). **Patient answers:** the patient's original **Yes** (and what it produces, e.g. a "smokes" disclosure), with no change marked. Both tabs show the **Nurse checks** answers only where they were shown before S18 (Validated summary). After reload: the same.

### T-07 Fill in the HQ on the patient's behalf
1. As Cara, create an episode for `Sam Form07<initials>`, email `form07.<initials>@example.org` (no mobile).
2. On the **Patient's invite** card, click **Fill in for the patient**.
3. Fill in **About you**, wait for **Saved**. Go **Back to the episode**, then **Fill in for the patient** again.
4. Fill in **Your health**, then **Send the answers** → confirm. Click **Back to the episode**.
5. Validate **About you**, then open the **POA Summary**'s **Patient answers** tab.

**Expected:**
- Step 2: the address ends `/episodes/<id>/patient-form`; a blue box says you're filling it in for Sam Form07…, saved as theirs, marked as entered by you; only the patient's sections.
- Step 3: About you's answers are still there when you come back.
- Step 4: "Sent for Sam Form07…"; on the episode: status **Ready for review**; the health questionnaire card says **"Sent by Cara Clinician on the patient's behalf on <date>"**; under About you and Your health: **"Patient's answers entered by Cara Clinician"**; General notes: **"HQ completed on the patient's behalf"** by Cara.
- Step 5: the Validate page has the orange box "The patient's answers here were entered by Cara Clinician on the patient's behalf."; the POA Summary's Patient tab headings say **"Patient (entered by Cara Clinician) on <date>"**.

### T-08 Patient and staff each fill a section: only staff's is marked
1. As Cara, create an episode for `Sam Form08<initials>`, mobile `07700 900808`.
2. As the patient (phone size, the invite link, Continue, `17 5 1980`): fill in **About you** only. Leave it.
3. As Cara: **Fill in for the patient** → About you shows the patient's answers → fill in **Your health** → **Send the answers**.
4. Back to the episode.

**Expected:** **Your health** says "Patient's answers entered by Cara Clinician"; **About you** has **no** "entered by" line. The patient opening their link again sees the HQ as sent ("Thank you, Sam").

### T-09 A stale save in a second tab: "changed somewhere else" (S02a), clinician
1. As Cara, create an episode for `Sam Form09<initials>`, mobile `07700 900809`; as the patient (phone size) fill in both sets and **Send the answers**.
2. As Cara, open the episode, click **Validate** on **About you** (tab A). Copy the address into **tab B**.
3. In tab A, change **Do you smoke?**. Wait for **Saved**.
4. In tab B (still showing the old answer), click **Validate**.
5. Click **Reload** in tab B.

**Expected:** step 4: a red **Not saved** box: "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved." with **Reload**; Validate disabled; tab B stays on the set page. Step 5: tab B shows tab A's answer and **no** "Validated by" badge. On the episode page About you is **not** validated.

### T-10 A stale save on two devices: "changed somewhere else", patient — phone size
1. As Cara, create an episode for `Sam Form10<initials>`, mobile `07700 900810`.
2. Open the invite link in **two** phone-size windows (A and B: e.g. a private window and tester.localhost), each with Continue and `17 5 1980`. Both show About you empty.
3. In A, answer **Do you smoke?**. Wait for **Saved**.
4. In B, answer the same question differently.

**Expected:** B shows the **Not saved** message "These answers were changed somewhere else…" with a way to reload; after reloading, B shows A's answer. Nothing else is lost.

### T-11 A patient's save after the answers were sent is refused — phone size
1. Keep window B from T-10 open (or open the link again in a second window before sending). In window A, finish both sets and **Send the answers**.
2. In window B, change an answer.

**Expected:** B can't save it: it says the answers have already been sent (e.g. "These answers have already been sent, so they can't be changed here."). Cara's episode shows the answers from A only.

### T-12 Reminders stop after the submit (S16's "run jobs as of" page)
1. As Cara, create two episodes: `Sam Form12a<initials>` with mobile `07700 900812`, and `Sam Form12b<initials>` with mobile `07700 900813`.
2. As patient **12a** (phone size): open the invite, Continue, `17 5 1980`, fill in everything and **Send the answers**.
3. Leave **12b** untouched.
4. Open **/qa/jobs** → **Now + 2 days** → **Run jobs**. Then **Now + 7 days** → **Run jobs**.
5. Captured messages for `+447700900812`, then for `+447700900813`. Open both episodes as Cara.

**Expected:** **12a** (sent): **only the invite** text; no reminder; its episode has no reminder notes and no **Not responding** badge. **12b** (untouched): the invite plus a **reminder** ("Hello Sam, a reminder from Hospital A1: …"); its General notes include "Day 2 reminder sent by text message…" (Automatic), and after the day-7 run a **Not responding** badge and note.

### T-13 Reminders stop once the patient starts (not only on submit)
1. As Cara, create an episode for `Sam Form13<initials>`, mobile `07700 900814`.
2. As the patient (phone size): answer **one** question in About you, wait for **Saved**, close it (don't send).
3. **/qa/jobs** → **Now + 2 days** → **Run jobs**; then **Now + 7 days** → **Run jobs**.
4. Captured messages for `+447700900814`; open the episode as Cara.

**Expected:** only the invite text; **no reminder** (starting stops reminders, S16). After the day-7 run the episode **does** get the **Not responding** badge and note: S16's rule marks it on day 7 whenever the HQ hasn't been sent, started or not.

### T-14 One patient, two trusts: both on one home — phone size
1. Press **Reset sign-in limits** on /qa/messages.
2. As **Cara** (Hospital A1): create an episode for `Sam Form14<initials>`, date of birth `17/05/1980`, mobile `07700 900815`, procedure `Knee replacement`.
3. As **Bea** (Hospital B1): **New episode** → the same first name, surname, date of birth and mobile; any Trust B published HQ; procedure `Hip replacement` → **Create episode**. (If B1 offers no HQ, create and publish one as Bea first: one patient Question Set with one question.)
4. Phone size, not signed in as staff: open **Hospital A1's** invite link → Continue → `17 5 1980`. Tap **← Your questionnaires**.
5. Tap **Start** on the Hospital B1 card, answer one question, wait for **Saved**, go back to the home.

**Expected:** step 4: `/home` shows **two** cards, newest first: **Hospital B1** (Procedure: Hip replacement) above **Hospital A1** (Procedure: Knee replacement), each with its HQ's name, **For Sam Form14…**, **To do**, **Start**. Step 5: the B1 card shows **In progress** / **Continue**; the A1 card is still **To do**. As Cara, the A1 episode shows no answers from B1's questionnaire.

### T-15 Answers stay with their own episode
1. As Cara, create **two** episodes for the same new patient `Sam Form15<initials>` (mobile `07700 900816`), both on the HQ (create the second from the patient's record or by typing the same details again).
2. As the patient: open the **first** episode's invite link, answer **About you**, wait for **Saved**.
3. From the patient home (← Your questionnaires), open the **second** episode.

**Expected:** the second episode's HQ is **empty**: the first one's answers don't appear there. On the home, the first is **In progress** and the second **To do**. As Cara, only the first episode shows the patient's answers.

### T-16 Refused: another hospital and another trust
1. Copy the address of T-01's episode page (`/h/<Hospital A1>/episodes/<id>`) and of its POA Summary.
2. Sign in as **Val Clinician** (Hospital A2) and paste each address.
3. Sign in as **Bea Author** (Trust B) and paste each address.
4. Signed out (or in a private window), paste the patient-form address (`…/episodes/<id>/patient-form`).

**Expected:** Val and Bea each get a refusal ("You don't have access to this hospital." or "Episode not found"), never the patient's details or answers. Signed out: the sign-in page.

### T-17 Reload, deep links and Back
For T-07's episode, as Cara, open each address, **reload** it, and paste it into a **new tab**: the episode page; a set's Validate page (`…/sets/<set id>`); the POA Summary; the patient-form page. Then click through Episodes → episode → Validate → Back → POA Summary → Back.
On the phone, reload `/home` and `/home/episodes/<id>` (from T-14).

**Expected:** each loads the same content after a reload and in a new tab; Back returns to the previous screen each time; no error or blank page.

## Regression (earlier sessions)

- **R-1 (S11, the audit log):** as **Ada Admin**, open the trust's **Audit log**: the actions from T-01 to T-07 appear (episode viewed, answers saved, submitted…), with IDs only. **Export CSV** downloads (or offers) a file.
- **R-2 (S13, patient records):** on **New episode**, type a patient whose details match an existing record (e.g. T-01's patient again): it links to the same record (no duplicate), and the patient record lists both episodes.
- **R-3 (S14, invites):** on a fresh episode, **Resend invite**: a new text arrives and the old link no longer works ("This link doesn't work. Please check it, or contact the hospital."); five wrong dates of birth on a link lock it, and the episode shows the lock.
- **R-4 (S10, suspend):** not needed unless time allows: as Sam Superadmin, suspending Trust B hides T-14's B1 card from the patient home; **un-suspend it afterwards**.
- **R-5 (authoring):** as Alex, open the HQ in the editor (Questionnaires → the HQ): it opens; the published version is read-only.

## Out of scope (don't report)

- **Episodes made before S18 are gone** (see the top). Expected.
- Procedures, planned dates, new statuses, cancel, archive, tasks and worklists: later sessions (S19–S22).
- Any new screen: S18 adds none.
- The known items in `docs/pending.md` (e.g. the POA Summary listing a clinician's change to a question that a later change hid).
- The real print dialog (no tester can open it).
