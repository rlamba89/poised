# S22 manual test plan: archive, auto-archive and closing a patient account

For a tester who hasn't seen the code: use the browser only.

**Changed after test report 2:** T-18 step 5 runs as of 12 months and a day (F-3: the plan was wrong, the rule stands); T-10's archived box shows the run's date, not now; T-12 also tries `7.5`, which now stays `7.5` in the box and is refused (it was read as 75).

**Changed after test report 1 (retest):** T-12 step 1 now names the exact refusal (F-1 fixed: the box no longer changes 0 to 1 or 366 to 365). T-18 is unchanged; F-2 is fixed (a run "as of" a later time now runs the auto-archive before the reminders, and dates what it writes as of that time). T-16 step 1 names the invite card's new wording.

## Setup

1. In your own terminal, from the repo root:
   ```sh
   docker compose up -d db
   make migrate seed
   make dev
   ```
   Leave `make dev` running. (In an automatic run, the conductor does this.)
2. Open **http://localhost:3000**. It shows the stub sign-in page: click a person's name to sign in as them. To change person, use **Sign out** (top right) and pick another.
3. **Seeded people** (made up):

   | Name | Role | Where |
   | --- | --- | --- |
   | Ada Admin | admin | the whole of Trust A (she can open Settings and the Audit log) |
   | Alex Author | super clinician | the whole of Trust A (both hospitals) |
   | Cara Clinician | clinician | Hospital A1 only |
   | Sue Super | super clinician | Hospital A1 only |
   | Val Clinician | clinician | Hospital A2 only |
   | Bea Author | super clinician | the whole of Trust B |

4. **Seeded data:** each trust has a published HQ, **Demo pre-op HQ** v1, with three Question Sets: *About you* and *Your health* (the patient's) and *Nurse checks* (the clinician's). The database also holds "E2E …" HQs, patients and episodes from the automated journeys and earlier test rounds. **Give every patient you create a surname of your own starting `S22`**, so you can find your rows.
5. **Captured messages:** texts and emails aren't sent. **http://localhost:3000/qa/messages** lists them (newest first; it can filter by address). Invite links look like `http://localhost:3000/i/…`, sign-in links `http://localhost:3000/s/…`.
6. **Scheduled jobs:** **http://localhost:3000/qa/jobs** runs the jobs as of a later time: pick a preset (**Now + 29 days**, **Now + 31 days**, …) or type a date and time into **Run as of**, then **Run jobs**. **Each run also runs every hospital's auto-archive as of that time** (by itself it runs at 02:00 at each hospital). It runs for every episode in the database, so other episodes may be archived or reminded too; only check your own. The result line says how many episodes were archived.
7. **Dates** show in en-GB (e.g. 21/10/2026). Date fields are the browser's date picker; you can type into them.
8. **Phone size** means a 390×844 window (or the browser's device toolbar at that size).

### Recipes (used by many cases)

**Make an episode** (as Cara, at Hospital A1): **Episodes** → **New episode** → a new patient (first name `Sam`, your own `S22 …` surname, date of birth `1980-05-17`, sex Female, an email such as `s22.one@example.test`), Health questionnaire **Demo pre-op HQ v1**, then **Create episode**. Its status is **HQ not complete**.

**Make it Ready for POA:** on the episode page, **Fill in for the patient**: answer every question in About you and Your health and send it (status **Ready for review**). Then **Validate** each of the three Question Sets (Nurse checks' blood pressure e.g. `120/80`) and **Complete HQ review**. Status **Ready for POA**.

**Make it Discharged today:** from Ready for POA, use the **Episode status** dropdown: **Ready for admission** → Change status; **Admitted** with today's date → Change status; **Discharged** with today's date → Change status.

## What's new, in short

- **Archive episode** (a button beside the Episode status, for **super clinicians and admins only**): from any status, with an optional reason. An archived episode is **read-only**: every control is off, and the server refuses every change. It leaves **every worklist** and the **All** list, but is still found by the search, on the patient's page, and in the list with **Show archived** ticked (or the status filter set to **Archived**). Reminders stop; the patient's questionnaire shows as **closed**.
- **Un-archive: back to <status>** (super clinicians and admins only), with a required reason: back to **exactly** the status it was archived from.
- **The daily auto-archive** (each night at 02:00 at each hospital): a **Discharged** episode **30 days after its discharge date**, and an episode **before admission (or cancelled)** with **no activity for 6 months**. Activity is any change to the episode: answers, status, notes, tasks, messages sent. A hospital can set its own days and months in **Settings** ("Archive episodes this many days after discharge", "Archive pre-op episodes with no activity for (months)"). The status history names the rule.
- **Close my account** at the bottom of the patient's home: after a confirmation (which warns if a questionnaire hasn't been sent), the patient's login goes; the hospitals keep the patient and their episodes, with a General notes line. A new invite starts a new account.

## Test cases

### A. Archive by hand

**T-01: Only a super clinician (or admin) can archive.**
1. As Cara, make an episode (recipe) and open it. **Expected:** no **Archive episode** button anywhere on the page.
2. Sign out; sign in as **Sue Super**; open the same episode (Episodes → its row). **Expected:** an **Archive episode** button beside the Episode status.
3. Sign in as **Alex Author**, open it. **Expected:** **Archive episode** is there too.

**T-02: Archive with a reason (Enter sends), and without one.**
1. As Cara, make an episode and make it **Discharged today** (recipes). Sign in as **Sue Super** and open it.
2. **Archive episode**. **Expected:** an inline form "Archive this episode", saying it can't be changed afterwards, leaves every worklist, can still be found, reminders stop, and a super clinician can un-archive it; a **Reason (optional)** box.
3. Type `S22 treatment finished` and press **Enter**. **Expected:**
   - a grey box "This episode is archived": "Archived from Discharged on <now> by Sue Super. Reason: S22 treatment finished. It can't be changed: a super clinician can un-archive it.";
   - an **ARCHIVED** badge by the name; the Episode status shows **Archived**;
   - Status history's last row: "Archived: Discharged → Archived", by Sue Super, with the reason;
   - General notes: "Episode archived: S22 treatment finished".
4. Make another episode (as Cara), sign in as Sue, **Archive episode**, and click **Archive episode** in the form with no reason. **Expected:** archived; the box has no "Reason:"; General notes "Episode archived".
5. Try pasting more than 1,000 characters into the reason box. **Expected:** it stops at 1,000.

**T-03: An archived episode can't be changed: every control is off.**
1. Open T-02's archived (Discharged) episode as **Sue Super**. **Expected:**
   - the **Episode status** dropdown is disabled; no **Cancel episode**, **Undo**, **Re-open HQ review** or **Archive episode** buttons (only **Un-archive: back to Discharged**);
   - no **Edit procedure and dates** button;
   - **Tasks** says "This episode is archived, so its tasks can't be changed." with no **Add task**, **Complete** or **Re-open**;
   - **General notes** has no **Add comment**.
2. Click **POA Summary**. **Expected:** the status dropdown disabled; no Add comment; the **Nurse ASA grade** and **Anaesthetist ASA grade** dropdowns and their **Save** buttons are disabled.
3. Back on the episode, click **View** on a Question Set. **Expected:** read-only, with "This episode is archived, so these answers can't be changed."
4. As Cara, make an episode (HQ not complete) and let Sue archive it. **Expected:** the **Patient's invite** card (Resend invite, Fill in for the patient) is not shown.
5. Sign in as **Cara** and open the archived episode. **Expected:** the same read-only page, and no **Un-archive** button.

**T-04: The server refuses changes from a page opened before the archive.**
1. As Cara, make an episode and add a task (**Add task**, `S22 call the patient`, Nurse). Make it Ready for POA (recipe).
2. Sign in as **Sue Super**. Open the episode in **two tabs**.
3. In tab 1, **Archive episode** → Archive episode.
4. In tab 2 (not reloaded): **Add comment**, type `S22 late note`, **Save comment**. **Expected:** refused, "This episode is archived, so it can't be changed. A super clinician can un-archive it." Nothing added (reload to check).
5. Still in tab 2 (not reloaded), pick **On hold** from the Episode status dropdown and **Change status**. **Expected:** refused with "This episode changed since you opened it…" and **Reload**. Click **Reload**: the page is the read-only archived page (the status form closes, saying the change is no longer offered).

**T-05: It leaves every worklist and the All list; Show archived brings it back.**
1. As Cara, make an episode with a **Planned procedure date** of tomorrow, add a task for the **Anaesthetist** team, make it Ready for POA, then set its status to **On hold**.
2. Go to **Episodes**. Note the counts on **Anaesthetic review needed**, **Procedure within 14 days**, **On hold** and **Open tasks**, and check the episode is in each of those tabs and in **All**.
3. As **Sue Super**, archive it. Go to **Episodes** (as Sue or Cara). **Expected:** each of those four counts is one less; the episode is in none of the tabs, and not in **All**.
4. On **All**, tick **Show archived**. **Expected:** the episode is listed with an **ARCHIVED** badge (the others stay too); the address has `archived=1`.
5. Reload. **Expected:** still ticked, still listed. Untick it: gone. Press **Back**: ticked again.
6. Untick it; choose **Archived** in the status filter. **Expected:** only archived episodes are listed (yours among them).
7. Sign in as **Alex Author**, choose **All hospitals**. **Expected:** the archived episode isn't in All; with **Show archived** ticked it is.

**T-06: Still found: search, the patient's page, a deep link.**
1. Copy the address of T-05's archived episode.
2. As Cara: **New episode** → search by its surname. **Expected:** the patient is found.
3. Open the patient (click the name on the episode page). **Expected:** the patient's page lists the episode with an **ARCHIVED** badge; clicking it opens the episode.
4. Paste the copied address in a new tab. **Expected:** the archived episode opens.

**T-07: Archiving a cancelled episode.**
1. As Cara, make an episode and **Cancel episode** (reason **Patient withdrew**).
2. As Sue, **Archive episode**. **Expected:** "Archived from Cancelled …"; history "Archived: Cancelled → Archived".
3. **Un-archive: back to Cancelled**, reason `S22 check`. **Expected:** back to **Cancelled**, with its "This episode is cancelled" box still saying "Reason: Patient withdrew".

### B. Un-archive

**T-08: Un-archive restores the exact status; a clinician can't.**
1. Use T-02's first archived (Discharged) episode. As **Cara**: no **Un-archive** button.
2. As **Sue Super**: **Un-archive: back to Discharged**. The form "Un-archive this episode: back to Discharged" explains reminders don't start again. Press **Un-archive** with no reason. **Expected:** "Say why the episode is being un-archived." (the app's message, not the browser's bubble).
3. Type `S22 follow-up visit booked`, press **Enter**. **Expected:** status **Discharged**, with the same **Admission date** and **Discharge date** as before; no archived box; history "Un-archived: Archived → Discharged" by Sue with the reason; General notes "Episode un-archived, back to Discharged: S22 follow-up visit booked"; **Add comment** and **Add task** are back.
4. Repeat with T-05's episode (archived from **On hold**). **Expected:** back to **On hold**, and back in the **On hold**, **Anaesthetic review needed**, **Procedure within 14 days** and **Open tasks** tabs, with the counts one higher.
5. Archive and un-archive an episode that is **Ready for review** (Fill in for the patient, sent). **Expected:** back to Ready for review, and **Complete HQ review** works again after validating.

**T-09: The patient's side of an archived episode (phone size).**
1. As Cara, make an episode with an email you'll use (e.g. `s22.phone@example.test`). From **/qa/messages**, open its invite link in a **phone-size** window (or a private window), **Continue**, date of birth `17 5 1980`. **Expected:** the questionnaire opens.
2. As Sue (in the staff window), archive the episode.
3. In the patient window, reload. **Expected:** "This questionnaire is closed, so there's nothing more to fill in. Please contact the hospital if you have any questions.", with **Your questionnaires** and **Sign out**.
4. Click **Your questionnaires**. **Expected:** the card says **CLOSED BY THE HOSPITAL**, "The hospital has closed this, so there's nothing to fill in…", with no Start/Continue button; nothing wider than the screen.
5. Sue un-archives it (reason `S22 test`). Patient reloads the home. **Expected:** **TO DO** with **Start**; the questionnaire opens again.

### C. The daily auto-archive

**T-10: 30 days after discharge: archived at +31 days, not at +29.**
1. As Cara, make an episode **Discharged today** (recipes).
2. Open **/qa/jobs**, click **Now + 29 days**, **Run jobs**. **Expected:** a result line "Ran … jobs as of …: … reminders sent, … marked "not responding", N episodes archived." Your episode (reload it) is still **Discharged**.
3. Click **Now + 31 days**, **Run jobs**. Reload the episode. **Expected:**
   - status **Archived**; the box says "Archived from Discharged on <the run's date and time, 31 days on> by Automatic (Auto-archive: 30 days after discharge)" (a job run dates what it writes with its own time);
   - Status history's last row: "Archived: Discharged → Archived", **By** "Automatic (Auto-archive: 30 days after discharge)";
   - General notes: "Episode archived (Auto-archive: 30 days after discharge)", marked **Automatic**.

**T-11: Un-archived, it isn't archived again the next night.**
1. As Sue, un-archive T-10's episode (reason `S22 still needed`). **Expected:** Discharged.
2. **/qa/jobs**: **Now + 2 days**, Run jobs. **Expected:** still **Discharged** (its 30 days now count from the un-archive).
3. **Now + 31 days**, Run jobs. **Expected:** **Archived** again, by the same rule.

**T-12: A hospital's own days; the trust's other hospital keeps 30.**
1. Sign in as **Ada Admin** → **Settings** → **Settings for: Hospital A1**. Untick "Use the trust's setting" under "Archive episodes this many days after discharge". Type `0`, click away, **Save Hospital A1's settings**. **Expected:** the box still says `0` (it isn't changed to 1), and a red message "Archive episodes this many days after discharge: enter a whole number from 1 to 365."; nothing saved. The same for `366` (not changed to 365), and for `7.5` (the box keeps `7.5`; not 75). Then `7`: saved. Optionally check another number setting (e.g. "Mark "not responding" on day" with `0`): refused the same way.
2. As Cara, make an episode **Discharged today** at Hospital A1 (`S22 A1`). As **Val Clinician** (Hospital A2), do the same at Hospital A2 (`S22 A2`).
3. **/qa/jobs**: type a **Run as of** 8 days from now, Run jobs. **Expected:** `S22 A1` is **Archived**, its history saying "Automatic (Auto-archive: 7 days after discharge)"; `S22 A2` is still **Discharged**.
4. **Now + 31 days**, Run jobs. **Expected:** `S22 A2` is archived, "Auto-archive: 30 days after discharge".
5. **Put it back:** as Ada, clear Hospital A1's override (back to the trust's value, 30).

### D. Closing a patient account

**T-13: Close my account, with the warning (phone size).**
1. As Cara, make an episode for a patient with email `s22.close@example.test` (surname `S22 Close`). Open its invite link at phone size, **Continue**, date of birth `17 5 1980`, then **Your questionnaires**.
2. Scroll down. **Expected:** a **Close your account** card: "Stop signing in to Preopsis. Your hospitals keep their records of you." and **Close my account**.
3. Click **Close my account**. **Expected:** the card explains what happens, with an orange warning "Your Demo pre-op HQ for Hospital A1 hasn't been sent. If you close your account, it won't be sent.", a red **Close my account** and **Keep my account**; nothing wider than the screen.
4. **Keep my account**. **Expected:** back to the first view; nothing changed.
5. **Close my account** → **Close my account**. **Expected:** "Your account is closed. You've been signed out, …".
6. Reload. **Expected:** "You've been signed out" (no home).

**T-14: The old links fail.**
1. As Cara, make an episode for a new patient `s22.links@example.test` (surname `S22 Links`). Open its invite link, **Continue**, date of birth `17 5 1980` (this makes their account).
2. In another window, go to **/signin**, enter `s22.links@example.test`, **Send me a link**. Don't open the link.
3. In the patient's window, go to **Your questionnaires** and close the account (**Close my account** twice).
4. Open the old **invite** link from /qa/messages, **Continue**. **Expected:** "This link doesn't work".
5. Open the unused **sign-in** link from step 2, **Continue**. **Expected:** it says the link has expired or has already been used.
6. At **/signin**, ask for a link to that email again. **Expected:** the page says to check your email, but /qa/messages shows **no new** message to it.

**T-15: Staff still see the patient and the episode, with a note.**
1. As Cara, open T-13's episode. **Expected:** the episode and patient are as before; General notes: "The patient closed their account on <today, YYYY-MM-DD>."
2. As **Ada Admin**, open the **Audit log**. **Expected:** a row "Patient closed their account" for that patient.

**T-16: A new invite starts a fresh account.**
1. On T-13's episode the **Patient's invite** card now says "The patient's link no longer works: the General notes say why (for example, they closed their account). Send a new invite if they still need it." and offers **Send invite**. Click it. **Expected:** "Invite sent again …".
2. Open the new invite link from /qa/messages (phone size), Continue, date of birth. **Expected:** the questionnaire opens; **Your questionnaires** shows it on a new home, with **Close your account** at the bottom.

**T-17: Another patient is untouched; no warning when everything is sent.**
1. Make a second patient `s22.other@example.test`; open their invite in a **different** browser window (private window), fill in and **send** the questionnaire, and go to their home.
2. Close **another** patient's account (T-13 with a third email) in the first window.
3. In the second window, reload the home. **Expected:** still signed in, their home as before.
4. In the second window, **Close my account**. **Expected:** no orange warning (their questionnaire was sent). Choose **Keep my account**.

## Regression (run these before T-18)

**R-1 (S20): cancel and undo.** As Cara, make an episode Ready for POA, pick **On hold**, then **Undo: back to Ready for POA** with a reason. **Expected:** back to Ready for POA. **Cancel episode** (Patient withdrew). **Expected:** Cancelled; **Add comment** still works on a cancelled episode.

**R-2 (S21): worklists.** On **Episodes**, a tab's count equals the number of rows its list shows (check two tabs). Add an Anaesthetist task to an un-archived episode: it appears in **Anaesthetic review needed**.

**R-3 (S16): reminders.** Make an episode with an email; **/qa/jobs** → **Now + 2 days** → Run jobs. **Expected:** a reminder email to it on /qa/messages, and "Day 2 reminder sent by email." in its General notes (Automatic).

**R-4 (S14/S15): the patient fills in and sends the HQ** at phone size, then their home shows **SENT** with **View**.

**R-5 (S19): New episode with a procedure** picks its default HQ, and the procedure shows on the episode.

## Run this last

**T-18: 6 months without activity (pre-op), and one touched last week isn't.** *(This archives nearly every un-admitted episode in the database, so run it after everything else.)*
1. As Cara, make an episode `S22 Idle` (HQ not complete). Note the time of its last General notes line (e.g. 14:37).
2. Wait until the clock shows **at least 2 minutes later**, then make episode `S22 Busy`, and add a comment to it (`S22 rang the patient`). Note that comment's time (e.g. 14:40).
3. **/qa/jobs**: type a **Run as of** of the date **6 months from today**, at a time **between** the two (e.g. 14:38), and **Run jobs**. (So `S22 Idle` has had no activity for 6 months by then, and `S22 Busy`'s comment is just under 6 months old.)
4. **Expected:** `S22 Idle` is **Archived**, "Automatic (Auto-archive: 6 months without activity)", and no reminder or "Not responding" note was added to it by that run; `S22 Busy` is still **HQ not complete** (it may get a reminder and a "Not responding" note, dated the run's time, 6 months on: that's the run's clock, not a fault).
5. Run as of **12 months and a day** from now. **Expected:** `S22 Busy` is archived too. (Not at 6 months and a day: the step-3 run sent it a reminder and marked it "Not responding", dated that run's time, and messages sent count as activity, so its 6 months count from then.)
6. Optional: as Ada, set Hospital A1's "Archive pre-op episodes with no activity for (months)" to `2`, make an episode, run as of 2 months and a day from now: it's archived. Clear the override afterwards.

## Out of scope (don't report)

- Retention and deletion after 8 years (EPI-06), and archiving when PROMs close (C9).
- Reminders don't start again after an un-archive (by design: resend the invite).
- A real 02:00 run: the timer's own nightly run can't be waited for; **Run jobs** stands in for it (the schedule is covered by automated tests).
- Dates in General notes are ISO (YYYY-MM-DD), as before.
- Other episodes in the database being archived or reminded by your /qa/jobs runs.
