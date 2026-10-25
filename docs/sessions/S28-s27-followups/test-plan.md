# S28 manual test plan: the follow-ups after S27

**For the tester:** you've never seen the code. Use only the browser (and, where a step says so, a JSON page or a downloaded CSV). Report each case as Pass, Fail (with what you saw) or Blocked (why). The brief is [brief.md](brief.md), and Rahul's answers are in [answers.md](answers.md).

## What this session changed
1. **Filling in for a patient who never opened this trust's link.** When staff press **Fill in for the patient**, before the form is first pre-filled, a patient record at this trust that hasn't joined a sign-in account joins the account whose **date of birth, mobile or email, first name and surname** all match it (case and spaces don't matter). Then the other trust's answers fill in, with the card naming its hospital and **Start fresh**, as for the patient.
   - It joins only that record, only to an account that already exists. It never makes an account, never joins the person's other records, and copies no mobile or email either way.
   - Trust A's audit log gets one row "Patient's record joined their sign-in account", by the staff member.
   - One detail different: no join, and nothing from the other trust.
2. **Another trust's audit log names only the trust.** When Trust A staff cause a read of Trust B's answers, Trust B's log (screen, CSV and **Who** filter) shows **Staff at Trust A**, with no name or email. Trust A's own log still names the person.
3. **A form is pre-filled on its first open only**, from this trust and other trusts alike. Later opens never read earlier questionnaires again, so Trust B's log has one read row per form.
4. **The POA Summary says when a section was started fresh**, on both tabs, with Validate's wording.
5. **Patient reported BMI** on the POA Summary comes from the patient's own height and weight until About you is validated, then from the validated ones. (Before, it showed "—" for the demo questionnaires.)
6. **Wording and the badge:**
   - an archived episode's read-only View no longer says "A member of the team will review them.";
   - a cancelled questionnaire the patient had already sent says, on their home, "The hospital has cancelled this. Please contact them if you have questions." An unsent one keeps "…so there's nothing to fill in…";
   - the red **Not responding** badge is hidden once an episode is cancelled (as it already was once archived).
7. **Small fixes:**
   - on a staff page, a question brought to the top ("Everything is still correct") stops below the top bar, with its title showing;
   - if the page can't load again after **Start fresh**, it says the answers were cleared, with **Reload** (automated only, see T-13);
   - leaving the POA Summary straight after Back no longer scrolls the next page (automated only).

## Setup

**Rahul's terminal** (the conductor does this in an automatic run):
```sh
docker compose up -d db
make migrate seed      # takes the database to migration 00028 and seeds the demo data
make dev               # leave it running
```

**Open:** http://localhost:3000

**Staff sign-in:** the stub login page lists the seeded users; click a name to sign in. Sign out (top right) before signing in as someone else, or use a second browser profile or an incognito window. The users you need:
- **Cara Clinician**: a clinician at Hospital A1 (Trust A). Creates Trust A episodes, fills in for patients, cancels.
- **Ada Admin**: an admin of the whole of Trust A. Uses Trust A's **Audit**.
- **Sue Super**: a super clinician at Hospital A1. Archives episodes.
- **Bea Author**: a super clinician of the whole of Trust B. Creates Trust B episodes at **Hospital B1** (pick Hospital B1 in the hospital picker).
- **Max Multi**: an admin of the whole of Trust B (and a clinician at A2). Uses Trust B's **Audit**: pick Hospital B1 in the picker.
- **Sam Superadmin**: the platform admin. Opens the testers' pages.

**Patients:** there's no patient sign-in page. A new episode sends the patient an invite email, which is captured, not sent.
- Open **http://localhost:3000/qa/messages** (sign in as **Sam Superadmin** if it asks).
- Find the newest email to the patient's address, and open its link (`/i/…`).
- Press **Continue**, then enter the patient's date of birth.
- **Test patient pages at phone size (390 × 844):** Chrome's device toolbar, or a window that narrow.

**A person with a Trust B history (used by T-01 to T-07):**
1. As **Bea** (Hospital B1), create an episode on the **Demo pre-op HQ** for a new patient: first name **Sam**, a surname you'll recognise (e.g. "S28 T01"), date of birth **17/05/1980**, a fresh made-up email (e.g. `s28.t01@example.test`) **and** a made-up mobile (e.g. `07700 900101`).
2. As the patient (that invite link, phone size), answer:
   - **About you:** smoker "I used to smoke", alcohol 6, height 170, weight 82;
   - **Your health:** diabetes No, snoring **Yes**, pregnant No, sick after an anaesthetic No, problems No, family No.

   Then **send**.
3. As **Cara** (Hospital A1), create an episode on the **Demo follow-up HQ** for a new patient with the **same** first name, surname, date of birth and email, but **no mobile**.
4. **Don't open the Trust A invite as the patient.** This is the point: the patient never opens Trust A's link.

**The seeded HQs:** the Demo pre-op HQ (both trusts) has **About you** (smoking, alcohol, height, weight) and **Your health** (diabetes, snoring, pregnancy, anaesthetic questions). The Demo follow-up HQ (Trust A) asks most of the same. Weight and pregnancy are **Always ask fresh**, so they're never pre-filled.

**Downloads:** a downloaded file lands in `~/Downloads`; open it and check its contents.

## Test cases

### The four-field join on the patient's behalf

**T-01. Cara fills in for a patient who never opened Trust A's link: Trust B's answers, the card names Hospital B1, and Start fresh works**
1. Make a person with a Trust B history (Setup, "S28 T01").
2. As **Cara**, open the Trust A episode and press **Fill in for the patient**.
3. **Expected:**
   - A blue banner "You're filling in Sam S28 T01's health questionnaire for them…".
   - **About you** and **Your health** both say **Please check**.
4. Open **Your health**. **Expected:** the card reads "We've filled in N answers from Sam's questionnaire at **Hospital B1** on <today>. Please check them with Sam.", with **Check each answer**, **Everything is still correct** and **Start fresh**.
5. Press **Check each answer**. **Expected:** diabetes No and snoring Yes are already chosen, each with a marker under it. Go back with **All sections**.
6. Open **About you**, press **Start fresh**, then **Yes, start fresh**. **Expected:** the section opens empty (smoking not chosen), with no card.
7. Reload the page. **Expected:** About you says **Start** (still empty); Your health keeps its answers.

**T-02. The audit logs: Trust A names Cara for the join; Trust B names only "Staff at Trust A"**
1. As **Ada**, open Trust A's **Audit**, type T-01's surname in **Patient**, and press **Show**.
2. **Expected:** exactly **one** row "Patient's record joined their sign-in account", by **Cara Clinician**, and the rows for opening the HQ to fill in for them, by Cara.
3. As **Max** (Hospital B1), open Trust B's **Audit**, type T-01's surname in **Patient**, and press **Show**.
4. **Expected:**
   - One row "Patient's answers read to pre-fill their HQ at another trust", whose **Who** says **Staff at Trust A**, with **no** name and **no** email under it.
   - The patient's own rows from step 2 of Setup are by "Patient (their link)".
   - Nothing in Trust B's log says "Cara" or shows her email.
5. Open the **Who** box. **Expected:** an entry **Staff at Trust A** (with no email in brackets). Pick it, clear the Patient box, and press **Show**. **Expected:** only rows by Staff at Trust A.
6. **Export CSV** (with the Staff at Trust A filter). Open the file from `~/Downloads`. **Expected:** the rows' **Who** column says "Staff at Trust A" and the **Email** column is empty; no Cara anywhere in the file.

**T-03. After the join, Trust A shows no contact only Trust B holds**
1. As **Cara**, open T-01's patient (the patient's name on the episode page, or search for them in New episode).
2. **Expected:** the patient's details show the **email** only. **No mobile** appears, though Trust B has one (07700 900101).
3. As **Bea** (Hospital B1), open the Trust B episode. **Expected:** its mobile and email are as Bea entered them (nothing changed).

**T-04. One detail different: no join, and nothing from Trust B**
1. Make a person with a Trust B history (Setup steps 1–2, e.g. "S28 T04"). In step 3, create the Trust A episode with the same details except the **surname** spelt differently (e.g. "S28 T04x").
2. As Cara, press **Fill in for the patient**. **Expected:** both sections say **Start**; nothing is filled in, and no card.
3. Repeat with another person ("S28 T04b") whose Trust A episode has the same name and date of birth, **no email**, and a **different mobile** (e.g. `07700 900199`). **Expected:** the same: nothing from Trust B.
4. As **Ada** (Trust A's Audit) and **Max** (Trust B's Audit), search for both surnames. **Expected:** no "Patient's record joined their sign-in account" row for the Trust A records, and no "read to pre-fill" row in Trust B's log for them.

**T-05. Opening the same form again reads Trust B no more**
1. Use T-01's episode. As Cara, open **Fill in for the patient** again (and reload it once).
2. As **Max**, open Trust B's Audit for T-01's surname. **Expected:** still exactly **one** "read to pre-fill" row (the one from T-01), not one per open.

**T-06. The first open is the only pre-fill, from this trust too**
1. As **Cara**, create an episode on the **Demo follow-up HQ** for a brand-new patient (e.g. "S28 T06", a fresh email). Press **Fill in for the patient**. **Expected:** both sections say **Start** (no history yet). Go back to the episode **without answering**.
2. As Cara, create a second episode for the **same patient** (pick them in the New episode search) on the **Demo pre-op HQ**. As the patient (that invite, phone size), answer both sections and **send**.
3. As Cara, open the **first** episode and press **Fill in for the patient**. **Expected:** both sections still say **Start**: the questionnaire sent after the first open doesn't fill it in. (Before S28 it would have.)
4. As the patient, open the first episode's invite (phone size). **Expected:** the same, nothing filled in.

### The POA Summary

**T-07. A section started fresh says so, on both tabs**
1. Finish T-01's form as Cara: answer **About you** (smoker, alcohol, height **170**, weight **82**) and confirm **Your health**, then **Send the answers**.
2. Open the episode's **POA Summary**.
3. **Expected:** under **About you**, the line "Answers were pre-filled here from the patient's earlier questionnaires. Cara Clinician chose “Start fresh” with the patient on <date>, so the pre-filled answers were cleared and every answer here was given afresh." It's the same wording as **Validate** shows for About you (open it to compare).
4. Switch to **Patient answers**. **Expected:** the same line under About you.

**T-08. Patient reported BMI before and after validation**
1. On T-07's POA Summary, before validating anything: **Expected:** **Patient reported BMI** shows **28.4** (82 kg at 170 cm), on both tabs.
2. As Cara, open **About you** (Validate), change the weight to **90**, and **Validate**.
3. Back on the POA Summary. **Expected:** **31.1** on both tabs.
4. An episode whose patient gave no height (e.g. T-06's first, or any unanswered one): **Expected:** "—".

### Wording and the badge (phone size for the patient)

**T-09. An archived episode's View doesn't promise a review**
1. As **Cara**, create an episode on the **Demo pre-op HQ** for a new patient ("S28 T09"). As the patient (phone size), answer both sections and **send**. **Expected:** "Thank you, Sam. Your answers have been sent to the hospital. A member of the team will review them."
2. As **Sue Super**, open the episode and **Archive episode**.
3. As the patient, open the invite link again (or **Your questionnaires** → **View**).
4. **Expected:** "Thank you, Sam. Your answers have been sent to the hospital." with **no** "A member of the team will review them.". Nothing is wider than the screen.

**T-10. A cancelled questionnaire the patient had sent**
1. As **Cara**, create two episodes on the **Demo pre-op HQ** for the **same** new patient ("S28 T10"), and a third one for them that you leave alone.
2. As the patient, answer and **send** the first; leave the second unanswered.
3. As Cara, **cancel** the first and the second (any reason).
4. As the patient (phone size), open the **third** episode's invite, then **Your questionnaires**.
5. **Expected:**
   - The sent one: **Cancelled by the hospital**, with "The hospital has cancelled this. Please contact them if you have questions."
   - The unsent one: **Cancelled by the hospital**, with "The hospital has cancelled this, so there's nothing to fill in. Please contact them if you have any questions."
   - Nothing is wider than the screen.

**T-11. NOT RESPONDING is hidden once cancelled**
1. As **Cara**, create an episode for a new patient ("S28 T11"). Don't answer it.
2. Open `http://localhost:3000/qa/jobs` (as **Sam Superadmin** if asked) and run the jobs **as of 8 days from now**.
3. As Cara, open the episode. **Expected:** the red **Not responding** badge beside the status.
4. **Cancel** the episode (any reason). **Expected:** the badge is gone. Reload: still gone.
5. (Archived still hides it too: archive it as Sue. **Expected:** no badge.)

### The small fixes

**T-12. "Everything is still correct" on a staff page shows the question's title**
1. Use a full-size desktop window. Make a person with a Trust B history (Setup, "S28 T12"), and as Cara press **Fill in for the patient**.
2. Open **Your health** and press **Everything is still correct**.
3. **Expected:** the form moves to the first question not filled in (the pregnancy question, which is always asked fresh). Its **title** ("Is there any chance you could be pregnant?") is fully visible **below** the top bar (the teal Preopsis bar), not hidden under it.

**T-13. A failed reload after Start fresh (automated only)**
- Making the page fail to load just after Start fresh needs a blocked network request, which a tester can't do by hand. An automated journey covers it (the page says "The answers in this section were cleared", with **Reload**). Mark it **Blocked (automated)** unless your tools can block one request.

## Regression (earlier sessions)
- **R-01 (S27):** a patient with a Trust B history who **opens** the Trust A invite themselves (enter the date of birth) still gets Trust B's answers, the card naming Hospital B1, and Start fresh; Trust B's audit row for it is by "Patient (their link)".
- **R-02 (S25):** a same-trust follow-up opened by the patient is pre-filled at its first open; **Everything is still correct** jumps to the first question not filled in (phone size).
- **R-03 (S11):** Trust A's Audit **Who** filter still lists Trust A's staff as "Name (email)", and filtering by **Cara Clinician** shows only her rows; the CSV still downloads.
- **R-04 (S27):** on the POA Summary, clicking an answer opens Validate at the question, and Back returns to the same tab and place.
- **R-05 (S20–S22):** a cancelled episode stays read-only for staff, and its patient's form says it was cancelled.

## Out of scope (don't report)
- Merging one person's two accounts (C10).
- IG/DPO sign-off on copying across trusts and on the staff-side join (C10).
- The print-designed POA Summary PDF (plan C9).
- Rows written **before** this session: an older Trust B audit row may still name a Trust A staff member (the log can't be changed). Judge only rows made during this test.
- Known older items in `docs/pending.md`.
