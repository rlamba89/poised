# S25 manual test plan: pre-fill and the patient's confirmation screens

For a tester who hasn't seen the code: use the browser only.

**Changed after test report 1 (retest):**
- **T-03:** at phone size the pregnancy question's **text** must now be on screen after the jump (F1 fixed).
- **T-13** has a new step 4: confirming a section with a card button counts as started (found 1, fixed).
- The follow-up HQ's details box is headed **Your details** (found 2).
- **R-03** adds a confirmed-only HQ, which must get **no** day-2 reminder.

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
   | Cara Clinician | clinician | Hospital A1 only (Trust A) |
   | Alex Author | super clinician | the whole of Trust A |
   | Val Clinician | clinician | Hospital A2 only (Trust A) |
   | Bea Author | super clinician | the whole of Trust B (Hospital B1) |
   | Max Multi | clinician at Hospital A2; admin of the whole of Trust B | |
   | Ada Admin | admin | the whole of Trust A (Audit log) |

4. **Seeded questionnaires (Trust A):**
   - **Demo pre-op HQ** (v1): *About you* (Do you smoke?, alcohol units, height, weight), *Your health* (Do you have diabetes?, Do you snore loudly?, Is there any chance you could be pregnant?, sickness after an anaesthetic, "Have you ever had a problem with an anaesthetic?" with "What happened with your anaesthetic?" shown only on Yes, the family question, allergies, regular medicines) and *Nurse checks* (the clinician's: Blood pressure).
   - **Demo follow-up HQ** (v1, new in S25): *About you* (the patient's details box, Do you smoke?, alcohol units, height, weight, and a new question "Can you climb two flights of stairs without stopping?"), *Your health* ("Do you have diabetes?" — the same wording as the Demo pre-op HQ but a question of its own, snoring, pregnancy, sickness after an anaesthetic, the anaesthetic-problem pair, allergies, regular medicines) and *Nurse checks* (Blood pressure).
   - **Weight** and **pregnancy** are **Always ask fresh**: never pre-filled.
   - Trust B has its own **Demo pre-op HQ**.
5. **Captured messages:** texts and emails aren't sent. **http://localhost:3000/qa/messages** lists them, newest first (it can filter by address). Invite links look like `http://localhost:3000/i/…`. Open a patient's link in a **separate private/incognito window** (or another browser profile) so the staff sign-in and the patient's session don't mix.
6. **Phone size** means a 390×844 window (or the browser's device toolbar at that size). Everything the patient sees should be tested at phone size.
7. **Dates** show in en-GB, e.g. "23 Oct 2026".
8. **Give every patient you create a surname of your own starting `S25`** (e.g. `S25 Round1`) and an email of your own (e.g. `s25.round1@example.test`), so your rows are easy to find. Use a new surname and email for each recipe run, so each round starts clean.

### Recipes

**R-A: a reviewed first episode** (Trust A, as Cara at Hospital A1):
1. **Episodes → New episode** → a new patient: first name `Sam`, your `S25 …` surname, date of birth `1980-05-17`, sex Female, your email. Health questionnaire **Demo pre-op HQ v1**. **Create episode**.
2. Open the invite from **/qa/messages** (to your email) in the patient window, at phone size: **Continue**, date of birth 17 / 5 / 1980, **Continue**.
3. *About you*: Do you smoke? **I used to smoke**; alcohol units **6**; height **170**; weight **82**. **Finish this section**.
4. *Your health*: diabetes **No**; snore loudly **No**; pregnant **No**; sick after an anaesthetic **Yes**; problem with an anaesthetic **Yes**, what happened `I was sick all night`; family **No**; allergies: add one, `Penicillin` / `A rash`; regular medicines: none. **Finish this section**. **Send my answers** → confirm.
5. As Cara, open the episode: **Validate** *About you* as it is; in *Your health* change **Do you snore loudly?** to **Yes** (the clinician's correction), then Validate; *Nurse checks*: blood pressure `120/80`, Validate. **Complete HQ review**.

**R-B: the second episode** (as Cara): **Episodes → New episode** → search your surname, pick the patient (or type exactly the same details: the episode links to the same record), Health questionnaire **Demo follow-up HQ v1**, **Create episode**. A new invite goes to the same email; on /qa/messages it's the newest.

## What's new, in short

- When a patient (or staff for them) **first opens** an HQ, each of the patient's Question Sets with no answers yet is **filled in from the patient's earlier sent HQs at the same trust**: by the same question (same ID) or, failing that, by **exactly the same wording** (with safeguards). The newest sent HQ is used; where the clinician corrected an answer in the review, the clinician's answer is used.
- **Never pre-filled:** "Always ask fresh" questions (weight, pregnancy), clinician-only questions and Question Sets, the patient's details box, signatures and uploads; and answers from **another trust**.
- On the HQ's list, a pre-filled section shows **Please check** with a **Check** button, and doesn't count as complete until the patient answers its card.
- Opening it shows a **card**: "We've filled in N answers from your questionnaire on 3 Mar 2026. Please check them." (several dates: "…from your earlier questionnaires, the latest on …"), with **Check each answer** (walks the section from the start) and **Everything is still correct** (goes to the first question that wasn't filled in, or to the next section if there's none). The button pressed is recorded.
- Under each pre-filled answer: **"From your answers on 3 Mar 2026"**, until the patient changes that answer.
- Pre-fill happens **once per section**: nothing the patient typed is ever filled over.

## Test cases

### A. The patient's list and card

**T-01: Pre-filled sections wait to be checked.**
1. Do R-A, then R-B. Open the newest invite (R-B's) in the patient window at phone size; Continue; date of birth.
2. **Expected:** heading **Demo follow-up HQ**; both *About you* and *Your health* show **Please check** with a **Check** button; "0 of 2 sections complete"; **Send my answers** is disabled. No sideways scrolling.

**T-02: The card counts the right answers and shows the right date.**
1. From T-01, tap **Check** on *Your health*.
2. **Expected:** a card "Your earlier answers": "We've filled in **6 answers** from your questionnaire on **<the day you sent R-A's HQ>**. Please check them." with **Check each answer** and **Everything is still correct**. The questions aren't shown yet. (The 6: diabetes, snoring, sickness after an anaesthetic, problem with an anaesthetic, what happened, allergies. Not pregnancy.)

**T-03: Everything is still correct jumps to the first question not filled in.**
1. On the card, tap **Everything is still correct**.
2. **Expected:** the section's first page, scrolled to "Is there any chance you could be pregnant?" (nothing chosen), with the **question's text visible at the top of the screen** at phone size (not scrolled past it). Above it: diabetes **No**, snoring **Yes** (the clinician's correction, not the patient's No). Below: sickness **Yes**, problem **Yes**, what happened `I was sick all night`.
3. **Expected:** under diabetes, snoring, sickness, problem and what happened: **"From your answers on <date>"**; none under pregnancy.

**T-04: Changing an answer takes its marker away; hidden answers aren't lost.**
1. Answer pregnancy **No**. Change sickness after an anaesthetic to **No**. **Expected:** its marker goes; the others stay.
2. Change "Have you ever had a problem with an anaesthetic?" to **No**. **Expected:** "What happened…" disappears, and the problem question's marker goes.
3. Change it back to **Yes**. **Expected:** "What happened…" is back with `I was sick all night` and its marker; the problem question's marker is back too (its answer is the pre-filled one again).
4. **Next**: allergies has the `Penicillin` / `A rash` row with a marker; regular medicines is empty. **Finish this section**. **Expected:** *Your health* is **Complete**.

**T-05: Leaving and coming back fills nothing in again.**
1. Reload the page. **Expected:** *Your health* still **Complete**; tap **Review**: no card; sickness after an anaesthetic is still **No** (your change), pregnancy **No**.
2. Tap **All sections**.

**T-06: Check each answer walks the section.**
1. Tap **Check** on *About you*. **Expected:** the card says "We've filled in **3 answers**…" (smoking, alcohol, height).
2. Tap **Check each answer**. **Expected:** the first page: your details box headed **Your details** (name, age, sex; no marker), Do you smoke? **I used to smoke** with a marker, alcohol units **6** with a marker.
3. **Next**. **Expected:** height **170** with a marker; **weight is empty** (Always ask fresh); the stairs question is empty.
4. Fill weight `80`, stairs **Yes**, **Finish this section**. **Expected:** **Complete**; "2 of 2 sections complete"; **Send my answers** enabled.

**T-07: The card shows once; leaving it unanswered keeps the section waiting.**
1. Do R-B again (a third episode on the follow-up HQ for the same patient) and open its invite.
2. Tap **Check** on *About you*; on the card tap **All sections** (top left) without choosing. **Expected:** *About you* still **Please check**.
3. Tap **Check** again, then **Everything is still correct**. **Expected:** it goes to page 2, at the weight question (the first one not filled in; the details box doesn't count).
4. **All sections**, then **Continue**/**Check** on *About you* again. **Expected:** no card this time.

**T-08: The earlier answers come from the newest sent HQ.**
1. Send T-06's HQ (**Send my answers** → confirm). **Expected:** "Thank you, Sam".
2. As Cara, R-B again (another follow-up episode); open its invite and *Your health*.
3. **Expected:** the card's date is today (the follow-up HQ just sent); "Have you ever been sick… after an anaesthetic?" is now **No** (your T-04 change), not Yes.

**T-09: Everything is still correct with nothing left moves on.**
1. As **Alex Author**: **Questionnaires → New questionnaire** (name `S25 Short`), one Question Set; use **From library** to insert "Do you smoke?" (the platform's). Publish it.
2. As Cara, a new episode for the same patient on **S25 Short v1**; open its invite; **Check** the set.
3. **Expected:** "We've filled in 1 answer from your questionnaire on <date>. Please check it."
4. **Everything is still correct**. **Expected:** back on the list (it was the only section), **Complete**, **Send my answers** enabled.

### B. Staff, other trusts, and what's never filled in

**T-10: Filling in for the patient gets the same pre-fill.**
1. As Cara, R-B again (a new follow-up episode); on its episode page **Fill in for the patient**.
2. **Expected:** both sections **Please check**. *About you* → card: "We've filled in 3 answers from **Sam's** questionnaire on <date>. Please check them **with Sam**."
3. **Everything is still correct**. **Expected:** page 2, at weight; height's marker reads **"From Sam's answers on <date>"**.
4. Now open that episode's invite as the patient. **Expected:** *About you* doesn't show a card any more (the button was recorded); *Your health* still **Please check**.

**T-11: Another trust's answers are never used.**
1. As **Bea Author** (Hospital B1): New episode for a new patient: `Sam`, surname `S25 Cross <n>`, born `1980-05-17`, Female, email `s25.cross<n>@example.test`, HQ **Demo pre-op HQ**. As the patient, fill in and send it (any answers).
2. As Cara (Trust A): New episode for a new patient with **exactly the same details**, HQ **Demo follow-up HQ v1**. Open its invite as the patient (same date of birth).
3. **Expected:** both sections show **Start** (not Please check); opening them shows no card and no markers; nothing is filled in.

**T-12: Clinician-only answers are never pre-filled.**
1. Send T-10's HQ (as Cara: complete both sections and **Send the answers**), or any follow-up HQ from these tests.
2. As Cara, open that episode and **Validate** *Nurse checks*. **Expected:** Blood pressure is **empty** (R-A's `120/80` was the clinician's).

**T-13: The patient's home doesn't call an untouched pre-filled HQ "started".**
1. R-B again; open its invite; don't open any section; tap **Your questionnaires**.
2. **Expected:** the new follow-up questionnaire shows **To do** with **Start** (as a new one does), not "In progress".
3. Open it, change one answer in a section, come back to **Your questionnaires**. **Expected:** now "In progress".
4. R-B again; open its invite; tap **Check** on a section and **Everything is still correct** (change nothing); come back to **Your questionnaires**. **Expected:** "In progress" (confirming counts as starting).

**T-14: Deep links and the back button.**
1. In a pre-filled section showing its card, reload. **Expected:** back on the list, the section still **Please check**.
2. Press the browser's Back from the HQ to the home and Forward again. **Expected:** no error; the list as before.

## Regression (earlier sessions)

**R-01 (S14):** a new patient with no earlier HQ: their invite opens the HQ with **Start** on both sections, no card; filling in and sending works as before.
**R-02 (S02a):** open one follow-up HQ in two patient tabs, answer a question in one, then in the other: the second tab says the answers changed somewhere else, with **Reload**.
**R-03 (S16):** at **/qa/jobs**, run the jobs as of **Now + 2 days**: an untouched pre-filled HQ's patient gets a reminder (it isn't counted as started); T-13 step 4's HQ (a card answered, nothing typed) gets **no** reminder. Use a separate patient (surname and email) for each so the messages can be told apart, and check only your patients' messages on /qa/messages.
**R-04 (S20/S22):** cancel one of your follow-up episodes as Cara: the patient's link says it was cancelled (no form, no pre-fill).
**R-05 (S23/S24):** as Alex, open the **Demo follow-up HQ** in the editor: "Do you have diabetes?" shows **Pre-fills from N other HQs**, and its list names **Demo pre-op HQ · by same wording** (other test HQs may be listed too); "Do you smoke?" lists **Demo pre-op HQ · by link**.

## Out of scope

- What the clinician sees about pre-filled answers in Validate and on the POA Summary (markers, "Changed by patient", the button pressed): **S26**.
- Pre-fill from another trust with the patient's agreement, and the consent screen: **S26**.
- Pre-filling assessments (C7) and PROMs (C9).
- The editor's pre-fill label and "Always ask fresh" switch (S23), and the library (S24), except R-05.
