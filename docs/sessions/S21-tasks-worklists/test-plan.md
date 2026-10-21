# S21 manual test plan: tasks, worklists and status messages

For a tester who hasn't seen the code: use the browser only.

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
   | Ada Admin | admin | the whole of Trust A (both hospitals; she can open Settings) |
   | Alex Author | super clinician | the whole of Trust A |
   | Cara Clinician | clinician | Hospital A1 only |
   | Sue Super | super clinician | Hospital A1 only |
   | Val Clinician | clinician | Hospital A2 only |
   | Bea Author | super clinician | the whole of Trust B |

4. **Seeded data:** each trust has a published HQ, **Demo pre-op HQ** v1, with three Question Sets: *About you* and *Your health* (the patient's) and *Nurse checks* (the clinician's). The database also holds "E2E …" HQs, patients and episodes from the automated journeys, and episodes from earlier test rounds; they show in the lists and the counts too. **Give every patient you create a surname of your own starting `S21`**, so you can find your rows.
5. **Captured messages:** texts and emails aren't sent. **http://localhost:3000/qa/messages** lists them (newest first; it can filter by address).
6. **Scheduled jobs:** **http://localhost:3000/qa/jobs** runs the reminders and "not responding" as of a later time (**Now + 7 days**, then **Run jobs**). It runs every due job, so other episodes may change too.
7. **Dates** show in the deployment's locale, en-GB locally (e.g. 21/10/2026). Date fields are the browser's date picker; you can type into them. "Today + 14 days" means the calendar date 14 days after today.

### Recipes (used by many cases)

**Make an episode** (as Cara, at Hospital A1): **Episodes** → **New episode** → a new patient (first name `Sam`, your own `S21 …` surname, date of birth `1980-05-17`, sex Female, and an email such as `s21.one@example.test` and/or a mobile such as `07700 900101`), Health questionnaire **Demo pre-op HQ v1**, optionally a **Planned procedure date**, then **Create episode**. Its status is **HQ not complete**.

**Make it Ready for review:** on the episode page, **Fill in for the patient**: answer every question in About you and Your health, and send it. Back on the episode, the status is **Ready for review**.

**Make it Ready for POA:** then **Validate** each of the three Question Sets (Nurse checks' blood pressure e.g. `120/80`), and **Complete HQ review**. The status is **Ready for POA**, and the **Episode status** dropdown offers moves by hand (On hold, Ready for admission, …).

## What's new, in short

- **Tasks** on the episode page (a **Tasks** card below the status): **Add task** (a title and a team: Nurse, Anaesthetist or Admin), **Complete**, **Re-open**. Anyone at the hospital may do any of these. Each task says who added it and when, and who completed or re-opened it.
- **Worklists** on the clinician home (**Episodes**): tabs **All**, **Ready for review**, **Not responding**, **Anaesthetic review needed**, **Procedure within 14 days**, **On hold** and **Open tasks**, each with a count. The tab is in the address (`?worklist=…`). Hover a tab for what's in it; the selected one's line shows under the tabs.
- **Status messages to the patient:** when an episode is moved to **Ready for admission**, or **cancelled**, the patient can be texted and/or emailed the hospital's wording. Each of the four messages has its own switch in **Settings**, **off by default**.

### The worklists' rules

| Tab | An episode is in it when |
| --- | --- |
| Ready for review | its status is Ready for review |
| Not responding | it has the red **Not responding** badge (from the scheduled jobs) and the patient hasn't sent the HQ |
| Anaesthetic review needed | it has an open task for the **Anaesthetist** team |
| Procedure within 14 days | its planned procedure date is from today to today + 14 days (the hospital's setting), and it isn't Ready for admission, Admitted, Discharged or Cancelled |
| On hold | its status is On hold |
| Open tasks | it has any task that isn't done |

A cancelled episode is in none of them.

## Test cases

### Tasks

**T-01: Add a task for each team.**
1. As Cara, make an episode (recipe) and stay on its page. **Expected:** a **Tasks** card says "No tasks yet." with **Add task**.
2. **Add task**. A form opens with **Task** and **Team** (Choose a team / Nurse / Anaesthetist / Admin). Leave both empty and press **Enter** in Task. **Expected:** "Enter the task." (the app's own message, not the browser's bubble); nothing added.
3. Type `S21 book anaesthetic review`, leave the team, press **Enter**. **Expected:** "Choose the team."
4. Choose **Anaesthetist**, press **Enter** (or **Add task**). **Expected:** the form closes; the task is listed with an **Anaesthetist** badge, "Added by Cara Clinician on <now>" and a **Complete** button; the title says "(1 open)". General notes: "Task added for the anaesthetist team: S21 book anaesthetic review".
5. Add `S21 check bloods` for **Nurse** and `S21 send the letter` for **Admin**. **Expected:** three tasks, "(3 open)", each with its team.
6. Try pasting a title over 200 characters. **Expected:** the box stops at 200.
7. Reload the page. **Expected:** the same three tasks, in the order added.

**T-02: Complete and re-open, by different people.**
1. On T-01's episode, **Sign out** and sign in as **Sue Super** (Hospital A1). Open the episode (Episodes → the row). Click **Complete** on the Nurse task.
- **Expected:** the task is struck through, with a **Done** badge, "Completed by Sue Super on <now>", a **Re-open** button, and moves below the open ones; "(2 open)". General notes: "Task completed: S21 check bloods" by Sue Super.
2. Sign in as **Cara** again, open the episode, click **Re-open** on it.
- **Expected:** it's open again (no strike-through), "Re-opened by Cara Clinician on <now>" under "Added by …"; "(3 open)". General notes: "Task re-opened: S21 check bloods".
3. Complete it again as Cara. **Expected:** "Completed by Cara Clinician …" (the re-open line is gone while it's done).

**T-03: A stale tab is told, and keeps what was typed.**
1. As Cara, open the T-01 episode in two browser tabs.
2. In tab 1, **Complete** the Admin task. In tab 2 (not reloaded), click **Complete** on the same task.
- **Expected:** tab 2 shows "This episode changed since you opened it. Reload to see the latest; your change was not saved." with **Reload**; Reload shows the task completed by Cara.
3. In tab 2 open **Add task** and type a title and team; in tab 1 add any task; in tab 2 press **Add task**. **Expected:** the same "changed since you opened it" message, the typed title and team still there; **Reload**, then **Add task** again adds it.
4. (A task change is a change to the episode.) Make a Ready for POA episode, open it in two tabs, add a task in tab 1, then in tab 2 pick **On hold** in the status and **Change status**. **Expected:** tab 2 is told "This episode changed since you opened it…" with Reload.

**T-04: A cancelled episode's tasks can't be changed.**
1. Make an episode, add an **Anaesthetist** task, then **Cancel episode** (reason Patient withdrew).
- **Expected:** the Tasks card says "This episode is cancelled, so its tasks can't be changed.", lists the task, and has no **Add task**, **Complete** or **Re-open** buttons.
2. On **Episodes**, check **Open tasks** and **Anaesthetic review needed**. **Expected:** this episode is in neither.

### Worklists on the clinician home

**T-05: The tabs, their counts and the address.**
1. As Cara, open **Episodes**. **Expected:** under the title, the tabs **All**, Ready for review, Not responding, Anaesthetic review needed, **Procedure within 14 days**, On hold, Open tasks, each worklist with a number. **All** is selected, with the **All statuses** filter under it. Hovering a tab shows a line saying what's in it.
2. Click **Open tasks**. **Expected:** the address ends `?worklist=open_tasks`; the line "At least one task that isn't done, for any team." shows instead of the status filter; T-01's episode is listed, its Status column saying "2 open tasks" (or as many as are open).
3. **Reload**. **Expected:** still on Open tasks, same list. Press browser **Back**: you're on All. **Forward**: Open tasks again.
4. Copy the address, open it in a new tab. **Expected:** Open tasks is selected there too.
5. Edit the address to `?worklist=nonsense`. **Expected:** a red "Choose a worklist." message; the page doesn't break; clicking **All** works.

**T-06: Ready for review.**
1. Make an episode (surname `S21 Review`) and make it **Ready for review** (recipe). **Expected:** it's in **Ready for review**.
2. Make it **Ready for POA** (validate and complete the review). **Expected:** it has left Ready for review.

**T-07: Not responding.**
1. Make an episode with an email (surname `S21 Quiet`). Go to **/qa/jobs**, **Now + 7 days**, **Run jobs**.
- **Expected:** back on the episode, the red **Not responding** badge; it's in the **Not responding** tab.
2. Make an episode (surname `S21 Fresh`) and don't run the jobs for it. **Expected:** it's not in Not responding (just made, no badge). (The jobs page runs every due job, so run it before making this one.)
3. On `S21 Quiet`, **Fill in for the patient** and send it. **Expected:** it leaves Not responding and appears in Ready for review.

**T-08: Anaesthetic review needed and Open tasks.**
1. Make an episode `S21 Anaes` with an **Anaesthetist** task. **Expected:** in **Anaesthetic review needed** and in **Open tasks**.
2. Make an episode `S21 Nurse` with only a **Nurse** task. **Expected:** in Open tasks, **not** in Anaesthetic review needed.
3. Complete `S21 Anaes`'s task. **Expected:** it leaves both. Re-open it: it's back in both.

**T-09: Procedure within 14 days, and the ones that just miss it.** Make these episodes, each with a **Planned procedure date** set on the New episode page:
   - `S21 Today`: today; `S21 Day14`: today + 14 days; `S21 Day15`: today + 15 days; `S21 Yesterday`: yesterday; `S21 NoDate`: no date.
1. Open **Procedure within 14 days**. **Expected:** `S21 Today` and `S21 Day14` are listed; `S21 Day15`, `S21 Yesterday` and `S21 NoDate` aren't.
2. Make one more, `S21 Ready`, planned today + 3 days, and take it to **Ready for POA** (recipe). **Expected:** it's in the tab. Pick **Ready for admission** in the status. **Expected:** it has left the tab.

**T-10: On hold, and counts that follow a status move.**
1. Take an episode `S21 Hold` to **Ready for POA** and pick **On hold**. Note the **On hold** tab's count. **Expected:** it's in On hold.
2. Pick **Not ready for admission** on it, then come back to **Episodes**. **Expected:** the On hold count is one lower, and the episode isn't in On hold.

**T-11: Every count matches its list.**
1. For each of the six worklist tabs, click it and count the rows. **Expected:** the number on the tab equals the rows listed, every time (including tabs with 0, which say "No episodes in this worklist.").

**T-12: Hospitals: Cara sees only A1; a trust-wide admin sees both.**
1. Sign in as **Val Clinician** (Hospital A2). Make an episode `S21 Atwo` there with an **Anaesthetist** task, and one `S21 AtwoSoon` with a planned date today + 2 days.
2. Sign in as **Cara**. **Expected:** in no tab (and not in All) does `S21 Atwo` or `S21 AtwoSoon` appear; the counts are A1's only. Opening Val's episode by its address (copy it from Val's session) gives "Episode not found."
3. Sign in as **Ada Admin**, pick Hospital A1, open **Episodes**, choose **All hospitals**. **Expected:** the tab says **Procedure soon** (each hospital has its own days), and the lists hold both hospitals' episodes (a Hospital column shows which): `S21 Atwo` in Anaesthetic review needed and Open tasks, `S21 AtwoSoon` in Procedure soon, and Cara's A1 episodes too. Switching back to **Hospital A1** shows A1's only. The worklist tab stays selected when you switch.
4. Sign in as **Bea Author** (Trust B). **Expected:** none of Trust A's episodes in any tab.

**T-13: A hospital's own number of days.**
1. As **Ada Admin**, **Settings** → **Settings for: Hospital A2** → in **Worklists**, untick "Use the trust's setting" for **"Procedure within N days" worklist: number of days** and set **1**; **Save**.
2. As **Val**, open **Episodes**. **Expected:** the tab now reads **Procedure within 1 day** and `S21 AtwoSoon` (2 days away) has left it. (As Cara at A1 it still reads 14.)
3. Put it back: as Ada, tick "Use the trust's setting" again and **Save**. **Expected:** Val's tab reads 14 again.

### Settings and status messages

**T-14: The new settings, off by default.**
1. As **Ada Admin**, **Settings** (the whole trust). **Expected:** a **Worklists** section (the number of days, 14) and a **Messages when the status changes** section with, in pairs: "Text the patient when the episode is Ready for admission" (off) and "Ready for admission text message"; the same for email; then the same two for **cancelled**. Each template's help says `{{link}}` is optional and opens the patient's sign-in page. The four switches are **off**.
2. Pick **Hospital A1**. **Expected:** the same settings, each "Use the trust's setting" ticked.

**T-15: A status template needn't hold {{link}}; an invite's must.**
1. At **Hospital A1**, untick "Use the trust's setting" for **Ready for admission text message**, replace it with `{{hospital}}: {{firstName}}, you are ready for your operation (S21 test).`, untick it for **Text the patient when the episode is Ready for admission** and switch it **on**; **Save**. **Expected:** "Hospital A1's settings were saved."
2. Untick it for **Invitation text message**, remove `{{link}}` from it, **Save**. **Expected:** refused, saying the message must include `{{link}}`; nothing is saved. Put the invitation back to "Use the trust's setting" and Save.
3. Try `{{name}}` in the Ready for admission text. **Expected:** refused: "{{name}} isn't a placeholder…".

**T-16: Ready for admission sends the hospital's text.**
1. As Cara, make an episode with a mobile `07700 900211` and an email (surname `S21 Text`) and take it to **Ready for POA**. Pick **Ready for admission**, **Change status**.
- **Expected:** General notes: "Ready for admission message sent by text message." On **/qa/messages**, the newest text to +447700900211 is "Hospital A1: Sam, you are ready for your operation (S21 test)." No status email (its switch is off).
2. Undo it (**Undo: back to Ready for POA**, with a reason) and pick Ready for admission again. **Expected:** another text is sent (it's a new move to Ready for admission).
3. Pick **Admitted** (today), then **Undo: back to Ready for admission**. **Expected:** nothing new on /qa/messages: undoing an admission doesn't message the patient.

**T-17: With the switch off, nothing is sent.**
1. As Ada, Hospital A1: switch **off** "Text the patient when the episode is Ready for admission"; Save.
2. As Cara, take another episode with a mobile (`07700 900212`, `S21 Off`) to Ready for admission. **Expected:** no text on /qa/messages for it apart from its invite; General notes have no "message" line.

**T-18: The cancelled email, and a hospital that follows the trust.**
1. As Ada, Hospital A1: switch **on** "Email the patient when the episode is cancelled" (keep the default wording); Save.
2. As Cara, make an episode with an email (`s21.cancel@example.test`) and **Cancel episode** (Procedure cancelled). **Expected:** General notes "Cancelled message sent by email."; /qa/messages shows an email to s21.cancel@example.test, subject "Hospital A1: your health questionnaire was cancelled", saying Hospital A1 has cancelled the questionnaire and there's no need to fill it in. No text (its switch is off).
3. As **Val** (Hospital A2, which follows the trust: off), cancel an A2 episode with an email. **Expected:** nothing sent for it.

**T-19: A missing contact is noted.**
1. With A1's cancelled **email** switch still on, as Cara make an episode with a **mobile only** (`07700 900213`, no email) and cancel it. **Expected:** General notes "Cancelled message not sent. The patient has no email address."; nothing new on /qa/messages for it.

**T-20 (phone size): the {{link}} opens the sign-in page.**
1. As Ada, Hospital A1: change **Cancelled email** to `Hello {{firstName}}, {{hospital}} cancelled this. Your questionnaires: {{link}}`; Save.
2. As Cara, cancel another episode with an email. On /qa/messages, copy the link from that email (it ends `/signin`).
3. Open it in a **390×844** window (or the browser's device toolbar). **Expected:** the patient's sign-in page ("sign-in link by mobile or email"), fitting the width with no sideways scrolling; no part of the episode shows without signing in.

**T-21: Audit.**
1. As **Ada**, **Audit**. **Expected:** the task actions appear: "Opened an episode's tasks", "Added a task", "Completed or re-opened a task", by the people who did them.

**Clean-up (after T-14–T-20):** as Ada, Settings → Hospital A1: tick "Use the trust's setting" for every setting you changed, and Save. **Expected:** all four switches show off again.

## Regression (earlier sessions)

- **R-1 (S20):** the status control: a pick with a reason, Admitted with a date, Undo, Cancel; the Status history shows them.
- **R-2 (S14, S18):** the patient opens their own invite link (from /qa/messages) at phone size, confirms the date of birth, fills in and sends the HQ; the episode becomes Ready for review; Cara validates and completes the review; the POA Summary shows both tabs.
- **R-3 (S16, S17):** a new episode's invite and **Resend invite** still work; the invite card's messages are unchanged.
- **R-4 (S19):** **Edit procedure and dates** before the review saves, with its General notes lines; the planned date shows in the list.
- **R-5 (S08):** **All hospitals** for Ada still lists both hospitals with the status filter; Cara has no All hospitals switch.

## Out of scope (don't report)

- Tasks created by rules, e.g. "Book anaesthetic review" automatically (C6); the Auto-triaged worklist (C6).
- Archive and un-archive (S22): archived episodes leaving the worklists is S22's.
- User-defined (saved) worklists; assigning a task to a named person (S21 answers 1: teams only).
- Status messages for statuses other than Ready for admission and Cancelled.
- "E2E …" HQs, patients and episodes, and other testers' episodes, in the lists and counts.
