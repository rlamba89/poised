# S29 manual test plan: the follow-ups after S28

**Changed after test report 2 (25 Oct):** T-11 step 6 and R-05 now use "Do you snore loudly?" as the example candidate. The pregnancy question named before is Always ask fresh, so its label reads "Always ask fresh", not "Pre-fills from N other HQs" (as designed).

**For the tester:** you've never seen the code. Use only the browser (and, where a step says so, the browser's JavaScript console or your page-scripting tool). Report each case as Pass, Fail (with what you saw) or Blocked (why). The brief is [brief.md](brief.md), and Rahul's answers are in [answers.md](answers.md).

## What this session changed
1. **Patient reported BMI uses only answers that are shown.** On the POA Summary, a height, weight or BMI whose question, page, group or section is hidden by a condition no longer feeds the number, before validation (the patient's answers) and after it (the validated ones).
2. **Each button on the pre-fill card has its own words in the audit log** (Check each answer, Everything is still correct, Start fresh), by staff and by the patient:
   - staff: "Checked each pre-filled answer with the patient", "Confirmed pre-filled answers with the patient", "Started a Question Set fresh for the patient";
   - patient: "Patient chose to check each pre-filled answer", "Patient confirmed pre-filled answers", "Patient started a Question Set fresh".

   Rows written before this session keep the old shared words ("Checked pre-filled answers with the patient", "Patient checked pre-filled answers").
3. **One plain message when the server can't be reached:** "We couldn't reach the server. Check your connection and try again." on every page, never the browser's "Failed to fetch". After **Start fresh** whose reload fails, only the "The answers in this section were cleared" card shows, with no red banner above it.
4. **No open-task count on archived episodes:** the episode list (with **Show archived**) and the episode's **Tasks** title show no count once archived; the tasks are still listed.
5. **An episode archived from Cancelled** shows the patient the cancelled message ("This questionnaire was cancelled by the hospital…") on its form, not "This questionnaire is closed". Archived from any other status, it's as before.
6. **"Same as…" warns before replacing a link:** for a question whose ID already comes from the library or a published questionnaire, the window says "This question is already linked to “<that question's text>”. Using this replaces that link." A new question shows no warning.

## Setup

**Rahul's terminal** (the conductor does this in an automatic run):
```sh
docker compose up -d db
make migrate seed      # no new migration in S29; seeds the demo data
make dev               # leave it running
```

**Open:** http://localhost:3000

**Staff sign-in:** the stub login page lists the seeded users; click a name to sign in. Sign out (top right) before signing in as someone else, or use a second browser profile or an incognito window. The users you need:
- **Cara Clinician**: a clinician at Hospital A1 (Trust A). Creates episodes, fills in for patients, cancels, adds tasks.
- **Sue Super**: a super clinician at Hospital A1. Archives episodes.
- **Alex Author**: a super clinician of the whole of Trust A. Writes and publishes questionnaires.
- **Ada Admin**: an admin of the whole of Trust A. Uses Trust A's **Audit**.
- **Bea Author**: a super clinician of the whole of Trust B. Creates Trust B episodes at **Hospital B1** (pick Hospital B1 in the hospital picker).
- **Sam Superadmin**: the platform admin. Opens the testers' pages.

**Patients:** there's no patient sign-in page. A new episode sends the patient an invite email, which is captured, not sent.
- Open **http://localhost:3000/qa/messages** (sign in as **Sam Superadmin** if it asks).
- Find the newest email to the patient's address, and open its link (`/i/…`).
- Press **Continue**, then enter the patient's date of birth.
- **Test patient pages at phone size (390 × 844):** Chrome's device toolbar, or a window that narrow.

**A person with a Trust B history (used by T-02 to T-04 and T-06):**
1. As **Bea** (Hospital B1), create an episode on the **Demo pre-op HQ** for a new patient: first name **Sam**, a surname you'll recognise (e.g. "S29 T02"), date of birth **17/05/1980**, and a fresh made-up email (e.g. `s29.t02@example.test`).
2. As the patient (that invite link, phone size), answer:
   - **About you:** smoker "I used to smoke", alcohol 6, height 170, weight 82;
   - **Your health:** diabetes No, snoring **Yes**, pregnant No, sick after an anaesthetic No, problems No, family No.

   Then **send**.
3. As **Cara** (Hospital A1), create an episode on the **Demo follow-up HQ** for a new patient with the **same** first name, surname, date of birth and email.

Opening that Trust A episode (as the patient through its invite, or as Cara with **Fill in for the patient**) fills both sections from Trust B. Each section then opens on a card with **Check each answer**, **Everything is still correct** and **Start fresh** (Start fresh asks "Yes, start fresh" first).

**Making the network fail in the browser (T-05, T-06):** the app sends every request through the page's `fetch`. In the browser's JavaScript console (DevTools → Console), or with your page-scripting tool, run on the page:
- **Every request fails:** `window.fetch = () => Promise.reject(new TypeError("Failed to fetch"))`
- **Only reads fail (GET), changes still go through:** `const f0 = window.fetch; window.fetch = (u, i) => (i && i.method && i.method !== "GET") ? f0(u, i) : Promise.reject(new TypeError("Failed to fetch"))`

A full page reload (F5) puts the network back. If neither the console nor a scripting tool is available to you, mark those cases **Blocked (needs a blocked request)**; an automated journey covers them.

## Test cases

### The audit log: each card button in its own words

**T-01. The new words are in Trust A's audit log** *(read-only check of the screen)*
1. As **Ada**, open **Audit** (top bar) and press **Show** with no filters.
2. **Expected:** the page loads as before (Who, From, To, Patient filters; a **What** column). Nothing to check yet: the rows come in T-02 to T-04.

**T-02. The patient's three buttons (phone size)**
1. Make **two** people with a Trust B history (Setup): "S29 T02a" and "S29 T02b".
2. As patient **T02a** (the Trust A invite, phone size): open **Your health** → **Check each answer**; go back with **All sections**; open **About you** → **Everything is still correct**.
3. As patient **T02b** (the Trust A invite, phone size): open **About you** → **Start fresh** → **Yes, start fresh**. **Expected:** the section opens empty, with no card and no red banner.
4. As **Ada**, open **Audit**, pick **Who: Patients (their own link)**, type "S29 T02" in **Patient**, press **Show**.
5. **Expected:**
   - one row **Patient chose to check each pre-filled answer** (T02a);
   - one row **Patient confirmed pre-filled answers** (T02a);
   - one row **Patient started a Question Set fresh** (T02b);
   - **no** row "Patient checked pre-filled answers" for these patients.
6. **Export CSV** with the same filters, and open it from `~/Downloads`. **Expected:** the Action column has the same three wordings.

**T-03. Staff's three buttons, for the patient**
1. Make **two** more people with a Trust B history: "S29 T03a" and "S29 T03b". **Don't** open their Trust A invites as the patient.
2. As **Cara**, open T03a's Trust A episode → **Fill in for the patient**. In **Your health** press **Check each answer**; back with **All sections**; in **About you** press **Everything is still correct**.
3. As **Cara**, open T03b's episode → **Fill in for the patient** → **About you** → **Start fresh** → **Yes, start fresh**.
4. As **Ada**, open **Audit**, pick **Who: Cara Clinician (…)**, type "S29 T03" in **Patient**, press **Show**.
5. **Expected:**
   - one row **Checked each pre-filled answer with the patient**;
   - one row **Confirmed pre-filled answers with the patient**;
   - one row **Started a Question Set fresh for the patient**;
   - each by **Cara Clinician**, at Hospital A1; **no** "Checked pre-filled answers with the patient" rows for these patients.

**T-04. Pressing a second button in another tab** *(edge case)*
1. Make a person with a Trust B history ("S29 T04"). As the patient, open the Trust A invite in **two tabs** (phone size).
2. Tab 1: **Your health** → **Everything is still correct**. Tab 2 (still showing the card): **Check each answer**.
3. **Expected:** both work without an error, and Ada's Audit (Patient "S29 T04") has one **Patient confirmed pre-filled answers** and one **Patient chose to check each pre-filled answer** row: each row says which button was pressed. (The section keeps the first choice; that's unchanged.)

### Network errors

**T-05. A staff page says it plainly**
1. As **Cara**, open any episode's **POA Summary** (e.g. T03a's).
2. Make every request fail (Setup, "Every request fails").
3. In **General notes** press **Add comment**, type anything, press **Save comment**.
4. **Expected:** a red message **"We couldn't reach the server. Check your connection and try again."**; nowhere "Failed to fetch".
5. Reload (F5). **Expected:** the page works again; Save comment saves.
6. Repeat steps 2–4 on another staff page of your choice (e.g. the episode page: **Add task**, type a title, choose a team, **Add**). **Expected:** the same message.

**T-06. A patient page, and Start fresh whose reload fails (phone size)**
1. Make a person with a Trust B history ("S29 T06") and open the Trust A invite as the patient (phone size).
2. Open **Your health**, press **Check each answer**. Make every request fail (Setup). Change one answer (e.g. snoring to No).
3. **Expected:** a red message "We couldn't reach the server. Check your connection and try again." (no "Failed to fetch"). Reload the page (the change may be lost; that's fine).
4. Back on the section list, open **About you**. Make **only reads** fail (Setup, "Only reads fail"). Press **Start fresh** → **Yes, start fresh**.
5. **Expected:** the card **"The answers in this section were cleared"** with "The questionnaire couldn't be loaded again just now. Please try again." and **Reload**, and **no red banner** above it. Nothing is wider than the screen.
6. Reload the page (F5). **Expected:** About you is empty (started fresh), with no card.

### Archived episodes

**T-07. No open-task count once archived**
1. As **Cara**, create an episode for a new patient ("S29 T07"). On the episode page, **Add task** (any title, team Nurse). **Expected:** the Tasks title says **(1 open)**; in the episode list the row says **1 open task** under the status.
2. As **Sue Super**, open the episode and **Archive episode** (with a reason if asked).
3. **Expected:** on the episode page the **Tasks** title has **no** "(1 open)", the task is still listed, and it says the tasks can't be changed.
4. Open the episode list, tick **Show archived**. **Expected:** the archived row shows ARCHIVED and **no** "1 open task".
5. Un-archive it (Sue, with a reason). **Expected:** "(1 open)" and "1 open task" are back.
6. *(Cancelled is unchanged:)* a **cancelled** episode with an open task still shows its count.

**T-08. Archived from Cancelled: the patient sees the cancelled message**
1. As **Cara**, create **three** episodes on the **Demo pre-op HQ** for the **same** new patient ("S29 T08", a fresh email): E1, E2, E3.
2. As the patient, open E1's invite (phone size). Note the address in the browser bar: `/home/episodes/<E1's id>`. Do the same for E2 (open its invite; note `/home/episodes/<E2's id>`), and **answer and send E2**.
3. As **Cara**, **cancel** E1 and E2 (any reason). As **Sue**, **archive** both.
4. As the patient (phone size), open **E3's** invite and enter the date of birth. Then type E1's noted address (`http://localhost:3000/home/episodes/<E1's id>`) into the browser bar.
5. **Expected:** "This questionnaire was cancelled by the hospital, so there's nothing more to fill in. Please contact the hospital if you have any questions." Not "This questionnaire is closed".
6. Do the same with E2's address (sent before it was cancelled). **Expected:** the same cancelled message.
7. **Your questionnaires** (the home). **Expected:** E1 and E2 both **Cancelled by the hospital** (unchanged).

**T-09. Archived from another status keeps "closed" (and a sent one its View)**
1. Same patient as T-08. As **Cara**, create two more episodes for them on the **Demo pre-op HQ**: E4 and E5. As the patient, open each invite once and note `/home/episodes/<id>` for both; **answer and send E5**.
2. As **Cara**, open E5, validate both sections (Validate each Question Set) and complete the HQ review.
3. As **Sue**, archive E4 (still HQ not complete) and E5 (validated).
4. As the patient (E3's session from T-08, or open E3's invite again), go to E4's address. **Expected:** "This questionnaire is closed, so there's nothing more to fill in. Please contact the hospital…".
5. Go to E5's address. **Expected:** the read-only View of what was sent ("Thank you, Sam. Your answers have been sent to the hospital.", with no promise of a review), as before S29. The home shows E5 as **Sent** with **View** and E4 as **Closed by the hospital**.

### Patient reported BMI

**T-10. A height hidden by a condition doesn't feed the BMI**
1. As **Alex Author**, create a new questionnaire (e.g. "S29 BMI test") with one patient Question Set "About you" on one page:
   - a Select One (Yes/No) question **"Do you know your height?"**;
   - **From library**: **"How tall are you?"** (ID `height`) and **"What do you weigh now?"** (ID `weight`);
   - on "How tall are you?", a condition (Logic): shown only when "Do you know your height?" is **Yes**.

   Publish it.
2. As **Cara**, create an episode on it for a new patient ("S29 T10"). As the patient (phone size): answer **Yes**, height **170**, weight **82**, then change the first answer to **No** (the height question disappears), and **send**.
3. As Cara, open the episode's **POA Summary**. **Expected:** **Patient reported BMI** is empty ("—" or blank), on both tabs, though the patient once typed 170.
4. Open **About you** (Validate): change "Do you know your height?" to **Yes**, enter height **170** if it's empty, and **Validate**. **Expected:** the POA Summary shows **28.4**, on both tabs.
5. Validate again with "Do you know your height?" changed to **No** (the height still held but hidden). **Expected:** the BMI is empty again, on both tabs.
6. *(Regression)* An episode on the **Demo pre-op HQ** with height 170 and weight 82 sent still shows **28.4**.

### "Same as…"

**T-11. A question linked from the library warns; a new one doesn't**
1. As **Alex Author**, open a draft questionnaire (e.g. a **New version** of "S29 BMI test" from T-10, or a new questionnaire with a patient Question Set).
2. **From library**: insert **"Do you smoke?"**. On its card, open **Same as…**.
3. **Expected:** under the window's introduction, an orange line: **This question is already linked to “Do you smoke?”. Using this replaces that link.** The candidates list is as before.
4. Close the window. Add a **new** Select One question (type any text). Open **Same as…** on it. **Expected:** **no** "already linked" line.
5. On T-10's new version, open **Same as…** on **"How tall are you?"**. **Expected:** "This question is already linked to “How tall are you?”…".
6. On the new question from step 4, pick a candidate with **Use this** (e.g. "Do you snore loudly?", which isn't Always ask fresh). Then open **Same as…** on that question again. **Expected:** now it warns "This question is already linked to “Do you snore loudly?”…" (the text of the question its ID now comes from).
7. Close and reopen the window on the step-4-style new question (or another new one). **Expected:** no warning left over from before.

## Regression (earlier sessions)
- **R-01 (S28):** Cara's **Fill in for the patient** for a person with a Trust B history (who never opened Trust A's link) still fills in Trust B's answers, the card naming Hospital B1 (already exercised in T-03).
- **R-02 (S28):** the POA Summary still shows a section started fresh with "…chose “Start fresh”…" on both tabs (T-03b's episode after Cara sends it), and the BMI from the patient's answers before validation (T-10 step 6).
- **R-03 (S27):** clicking an answer on the POA Summary opens Validate at that question, and Back returns to the same tab.
- **R-04 (S22):** an archived episode is read-only for staff (status disabled, no Add comment, no Add task), and un-archive restores it (T-07).
- **R-05 (S24):** "Same as…" still gives a new question the chosen ID and options, keeps its own text, and its label then says **Pre-fills from N other HQs** (T-11 step 6, with a candidate that isn't Always ask fresh, such as "Do you snore loudly?": an Always ask fresh one, such as the pregnancy question, makes the label **Always ask fresh** instead, as designed).
- **R-06 (S11):** Trust A's Audit **Who**, **Patient** and date filters, and **Export CSV**, still work (T-02, T-03).

## Out of scope (don't report)
- An **action ("What") filter** on the Audit page: there isn't one (the brief's "What filter" is read as the What column; noted for Rahul).
- A second pre-fill after a later join (Rahul, 25 Oct: an edge case, not built).
- The catch-up reminder alongside Not responding (kept as designed).
- Merging one person's two accounts; IG/DPO sign-off (C10).
- Audit rows written **before** this session, which keep the old shared words.
- Each staff page view writing two audit rows in the dev build (React's development double-fetch; known).
- Known older items in `docs/pending.md`.
