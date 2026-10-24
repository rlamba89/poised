# S27 manual test plan: the follow-ups after plan C4

**Changed after verification 2:** only an automated journey (`test_audit.py`) and the card's wording when every answer came from **two or more** other hospitals. The wording now reads "…from your earlier questionnaires at Hospital B1 and Hospital C1…", with no "including". A unit test checks it; it can't be made by hand, because the seed has only two trusts. **No manual retest is needed.** If the conductor wants one, re-run T-01/T-02 (the card naming Hospital B1) as regression.
- **Note after test report 4:** the mixed wording ("…including one at Hospital B1…", this trust's answers with another trust's) can't be reached with the seeded questionnaires either. Trust A's own history from the Demo HQs covers every question Trust B's would fill. Both "including" wordings are checked by unit tests only.

**Changed after verification 1 (retest these, plus T-01/T-02 and T-07 as regression):**
- **V-1 (new). After Start fresh, a card still open elsewhere is refused.**
  1. Make a person as in T-01 (Trust B sent, then a Trust A follow-up episode, e.g. "S27 V1").
  2. As the patient, open the Trust A questionnaire in **two tabs** and open **Your health** in both, so the card shows in each.
  3. In tab 2, press **Start fresh**, then **Yes, start fresh**: the set opens empty.
  4. In tab 1, press **Everything is still correct**. **Expected:** a red message, "The answers in this section were cleared with “Start fresh” somewhere else, so there's nothing here to check. Please reload the page.", and the card stays. **Check each answer** gets the same.
  5. Reload tab 1. **Expected:** Your health opens empty, with no card.
  6. Repeat with **Cara's Fill in for the patient** as tab 1 (the patient presses Start fresh in their own tab). **Expected:** the same refusal for Cara.
- **V-2 (new). All answers from Hospital B1, from two dates.**
  1. Make a person with **two** sent Trust B Demo pre-op HQs (Bea creates the second after the first is sent).
     - In the second, answer only **Your health** (change snoring), and leave About you unanswered.
     - That gives Trust A's follow-up answers from two dates, all from Hospital B1, as long as the person has no Trust A history.
  2. Create their Trust A follow-up episode. **Expected:** a card reads "…from your earlier questionnaires **at Hospital B1**, the latest on …", with no "including".
  - If the two dates are the same day, the card reads "…from your questionnaire at Hospital B1 on …". That's also right: it just means the case can't show two dates today.
  - **Note after test report 3:** this recipe can't give two dates on any day. Sending the second HQ freezes its copied About you answers into it, so Trust A reads every answer from the second HQ. The two-date wording ("…from your earlier questionnaires at Hospital B1, the latest on …") is checked by the unit test only.
- **V-3 (automated only): no Start fresh when the only other-trust answer is hidden.** This needs a questionnaire whose Trust B answer sits on a question a condition hides. It's checked by a unit test; don't build one by hand.

**Note after test report 2:** with the corrected T-07, the patient's own open (after their date of birth, the section list loads the form) already copies Trust B's answers. So T-08's T-07 audit row is by the **patient**, not Cara. A staff-attributed read needs a person who joined through an **earlier** Trust A episode, then a second Trust A episode that Cara fills in. The tester checked this; it's attributed to Cara Clinician.

**Changed after test report 1:**
- **T-07 corrected (F1):** staff filling in for the patient get another trust's answers only once the patient's record at this trust has **joined their account**. That happens when the patient enters their date of birth on this trust's link (S15). Go never matches records across trusts by typed details alone. The step now has the patient open the Trust A invite and enter their date of birth, **without opening a section**, before Cara fills in. A new step checks that a person who hasn't done so gets nothing from Trust B.
- **T-13 (F2):** the Cancelled email's **subject** is neutral too: "<hospital>: your pre-operative assessment was cancelled".

**For the tester:** you've never seen the code. Use only the browser (and, where a step says so, a JSON page opened in a browser tab, or the browser's console). Report each case as Pass, Fail (with what you saw) or Blocked (why). The brief is [brief.md](brief.md), and Rahul's answers are in [answers.md](answers.md).

## What this session changed
1. **Answers from another trust are copied without asking.** S26 needed both trusts to switch on **Offer answers patients gave other trusts**, and asked the patient "Use the answers you gave …?" first. Both are gone.
   - A patient with a Trust B history now opens a Trust A questionnaire already filled in from Trust B, wherever Trust A's own history doesn't cover a question.
   - A section holding Trust B's answers opens on the usual card, which now **names the hospital** ("We've filled in 9 answers from your questionnaire at Hospital B1 on 3 Mar 2026. Please check them.").
   - The card has a third button, **Start fresh**. It asks first, then clears that section's pre-filled answers, so the section opens empty. It can't be undone, and it's offered only until something in the section changes.
   - Staff filling in for the patient get the same answers, card and button.
   - A suspended trust's answers are never used.
2. **An archived questionnaire the patient had sent** stays **Sent**, with a read-only **View**, on their home. It used to be "Closed by the hospital". An archived episode no longer shows the red **NOT RESPONDING** badge.
3. **The default Cancelled text and email** say "…has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." They used to say "…cancelled your pre-operative health questionnaire, so you don't need to fill it in…".
4. **The publish refusal names a text format in words**, e.g. "its answer format changed from any text to phone number", not the raw pattern.
5. **Clickable answers on the POA Summary.** On both tabs, these open **Validate** for that Question Set, scrolled to the question and briefly highlighted:
   - every answer (disclosure) line;
   - each Medication / Admissions / BMI table's title;
   - each line in a "Pre-filled on …" box;
   - each "Changed by the clinician" line.

   Back (the browser's, or Validate's **← POA Summary** link) returns to the same tab and the same place.

## Setup

**Rahul's terminal** (the conductor does this in an automatic run):
```sh
docker compose up -d db
make migrate seed      # takes the database to migration 00027 and seeds the demo data
make dev               # leave it running
```

**Open:** http://localhost:3000

**Staff sign-in:** the stub login page lists the seeded users; click a name to sign in. Sign out (top right) before signing in as someone else, or use a second browser profile or an incognito window. The users you need:
- **Cara Clinician**: a clinician at Hospital A1 (Trust A). Creates Trust A episodes, validates them, and fills in for patients.
- **Ada Admin**: an admin of the whole of Trust A. Uses Trust A's **Settings** and **Audit**.
- **Alex Author**: a super clinician of the whole of Trust A. Edits and publishes questionnaires.
- **Sue Super**: a super clinician at Hospital A1. Archives episodes.
- **Bea Author**: a super clinician of the whole of Trust B. Creates Trust B episodes at **Hospital B1** (pick Hospital B1 in the hospital picker).
- **Max Multi**: an admin of the whole of Trust B (and a clinician at A2). Uses Trust B's **Settings** and **Audit**: pick Hospital B1 in the picker.
- **Sam Superadmin**: the platform admin (`/admin`). Opens the testers' pages.

**Patients:** there's no patient sign-in page. A new episode sends the patient an invite email, which is captured, not sent.
- Open **http://localhost:3000/qa/messages** (sign in as **Sam Superadmin** if it asks).
- Find the newest email to the patient's address, and open its link (`/i/…`).
- Press **Continue**, then enter the patient's date of birth.
- **Test patients at phone size (390 × 844):** Chrome's device toolbar (iPhone 12 Pro, or a 390 × 844 custom size), or a window that narrow.

**The same person in two trusts:** create the episode at each trust with the **same first name, surname, date of birth and email**. When the patient enters their date of birth from the second trust's invite, both records join one account (S15). Use a fresh made-up email and a surname you'll recognise for each person, e.g. `s27.t01@example.test` and `S27 T01`.

**The seeded HQs:**
- **Demo pre-op HQ** (in both trusts) has two patient sections:
  - **About you:** smoking, alcohol, height, weight;
  - **Your health:** diabetes, snoring, pregnancy, sickness after an anaesthetic, anaesthetic problems and what happened, family problems, allergies, regular medicines.
- **Demo follow-up HQ** (Trust A only) asks most of the same questions, plus "Can you climb two flights of stairs without stopping?".
- Weight and pregnancy are **Always ask fresh**, so they're never pre-filled.
- Answer every required question, so you can send.

**Downloads and printing:** a downloaded file lands in `~/Downloads`; open it and check its contents. To check a print layout, use Print → **Save as PDF**, then open the PDF and look at it.

**Allowed in this test:** suspending and re-activating a trust (T-09). Put it back afterwards.

## Test cases

### Another trust's answers, without asking (phone size for the patient)

**T-01. A Trust B history fills a Trust A HQ with no question, and the card names Hospital B1**
1. As **Bea** (Hospital B1), create an episode on the **Demo pre-op HQ** for a new patient: Sam "S27 T01", date of birth 17/05/1980, a fresh email. Note the episode's address (its id is the last part of the URL).
2. As the patient (the invite link, at phone size), answer:
   - **About you:** smoker "I used to smoke", alcohol 6, height 170, weight 82;
   - **Your health:** diabetes No, snoring **Yes**, pregnant No, sick after an anaesthetic No, problems No, family No.

   Then **send**.
3. As **Cara** (Hospital A1), create an episode on the **Demo follow-up HQ** for a new patient with the **same** first name, surname, date of birth and email.
4. As the patient, open the **new** invite (phone size) and enter the date of birth.

**Expected:**
- **No question** comes before the sections ("Use your earlier answers?" never shows).
- **About you** and **Your health** both say **Please check**, and **Send my answers** is disabled.
- Opening **Your health** shows the card: "We've filled in N answers from your questionnaire at **Hospital B1** on <today's date>. Please check them.", with three buttons: **Check each answer**, **Everything is still correct** and **Start fresh**.
- **Check each answer** shows diabetes No and snoring Yes already chosen, each with "From your answers on <date>" under it. Pregnancy is empty.
- Nothing is wider than the screen (no sideways scrolling).

**T-02. Start fresh empties one section, asks first, and stays empty**

Continue with T-01's patient.
1. Go back to the sections (**All sections**). Open **About you**; the card shows. Press **Start fresh**.
2. **Expected:** the buttons give way to "Clear these answers? You'll answer every question in this section again. This can't be undone.", with **Yes, start fresh** and **Keep the answers**.
3. Press **Keep the answers**. **Expected:** the card's three buttons are back, and nothing has changed.
4. Press **Start fresh**, then **Yes, start fresh**.
5. **Expected:**
   - The section opens empty: no card, smoking not chosen, no "From your answers on…" markers.
   - The answers you then give save as usual.
6. Go back to the sections without answering anything, and reload the page.
7. **Expected:**
   - **About you** shows **Start** (or In progress, if you answered something), not Please check, and opening it shows no card and no copied answers: it isn't filled in again.
   - **Your health** still has its answers.
8. Finish both sections and **send**.

**T-03. Start fresh is only for another trust's answers**
1. As **Cara**, create an episode on the **Demo pre-op HQ** for a new patient, e.g. "S27 T03" (a fresh email). As the patient, answer both sections and send.
2. As Cara, create a second episode for **the same patient** (find them by search) on the **Demo follow-up HQ**. As the patient, open its invite.
3. **Expected:** the sections say Please check, and the card reads "…from your questionnaire on <date>…" **with no hospital named**, and only **Check each answer** and **Everything is still correct**: **no Start fresh**.

**T-04. Start fresh after the section changed elsewhere: refused, nothing cleared**
1. Make a person as in T-01 (Trust B sent, then a Trust A follow-up episode), e.g. "S27 T04". As the patient, open the Trust A invite and open **Your health**, so the card shows (tab 1).
2. In a **second tab** of the same browser, open the same questionnaire (copy the address of tab 1). Open **Your health**, press **Check each answer**, and change **snoring to No** (wait for "Saved").
3. Back in tab 1, press **Start fresh**, then **Yes, start fresh**.
4. **Expected:**
   - A red message at the top: "The answers in this section have changed since you opened it, so they weren't cleared. Please reload the page."
   - The confirmation closes, and the card is still there.
   - After a reload, Your health still has the answers (with snoring No): nothing was cleared.

**T-05. Trust A's responses never name Trust B's records**
1. Signed in as **Cara**, open in a browser tab: `http://localhost:3000/api/o/<Trust A's id>/h/<Hospital A1's id>/episodes/<T-01's Trust A episode id>/hq`. The ids are in Cara's address bar on the episode page: `/o/<trust>/h/<hospital>/episodes/<episode>`.
2. **Expected:**
   - The JSON's patient rows have `prefill` entries with `"otherTrust": true`, `"hospital": "Hospital B1"` and `"questionnaire": "Demo pre-op HQ"`, and **no `fromForm`**.
   - **About you**'s patient row has `"prefillChoice": "start_fresh"`.
   - Search the page (Ctrl/Cmd+F) for **Trust B's episode id** from T-01 step 1: it isn't there.

### The clinician's markers

**T-06. Validate and the POA Summary say where each answer came from, and that a section was started fresh**
1. As **Cara**, open T-01's Trust A episode, and open **Your health** (Validate).
2. **Expected:**
   - A line above the set: "N answers here were pre-filled from the patient's earlier questionnaires… The patient chose “Check each answer” on <date>."
   - Under **snoring**: "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust), <date> (matched by link)".
   - Under **diabetes**: "…(matched by same wording)".
3. Open **About you** (Validate). **Expected:**
   - The line reads "Answers were pre-filled here from the patient's earlier questionnaires. The patient chose “Start fresh” on <date>, so the pre-filled answers were cleared and every answer here was given afresh."
   - **No** "Pre-filled from…" marker on any question, even one answered the same as at Trust B.
4. Open the **POA Summary**. **Expected:** Your health's pages have a "Pre-filled on …" box naming Hospital B1 (another trust); About you has none.

**T-07. Staff filling in for the patient get the same answers, card and Start fresh**
1. Make a person with a sent Trust B HQ, as in T-01 steps 1–2 (e.g. "S27 T07"). As **Cara**, create their Trust A follow-up episode.
2. As the patient, open the **Trust A** invite, press **Continue** and enter the date of birth. That joins the Trust A record to their account. Then **close the tab without opening any section**.
3. As Cara, on the episode page, press **Fill in for the patient**.
4. **Expected:**
   - Both sections say Please check.
   - Your health's card reads "We've filled in N answers from Sam's questionnaire at Hospital B1 on <date>. Please check them with Sam.", with **Start fresh**.
5. In **About you**, press **Start fresh**. **Expected:** "Clear these answers? You'll answer every question in this section again with Sam. This can't be undone." Press **Yes, start fresh**: the section opens empty.
6. Answer everything and **Send the answers**. Open **About you** (Validate). **Expected:** "Cara Clinician chose “Start fresh” with the patient on …".
7. **Not joined yet:** make another person with a sent Trust B HQ (e.g. "S27 T07c"), and their Trust A episode. **Don't** open the Trust A invite. As Cara, press **Fill in for the patient**. **Expected:** both sections say **Start**, with nothing from Trust B, and Trust B's audit log has no "read to pre-fill" row for them. As decided, this isn't a failure: Go doesn't match records across trusts by typed details alone.

**T-08. The audit log: Trust B sees the reads, Trust A sees only its own records**
1. As **Max** (Hospital B1), open **Audit**, with today's dates, and apply.
2. **Expected:**
   - Rows "Patient's answers read to pre-fill their HQ at another trust" (no "(they agreed)"), for the patients of T-01, T-04 and T-07.
   - T-01's and T-04's are by the patient; **T-07's is by Cara Clinician**.
3. **Export CSV.** **Expected:** a CSV file in `~/Downloads`; open it. It has the same rows, with that action's label, the patient and the Hospital B1 episode.
4. As **Ada**, open Trust A's **Audit**. **Expected:** Trust A's rows for those episodes (opened their questionnaire, and so on) name only Trust A's patients and episodes, never a Hospital B1 one.

**T-09. A suspended trust's answers aren't used**
1. Make a person with a sent Trust B HQ (as T-01 steps 1–2, e.g. "S27 T09"). Don't create the Trust A episode yet.
2. As **Sam Superadmin**, open `/admin`, open **Trust B** and **suspend** it.
3. As **Cara**, create their Trust A follow-up episode. As the patient, open its invite.
4. **Expected:** nothing from Trust B. Both sections say **Start**, with no card.
5. As Sam, **re-activate Trust B**. **Expected:** Trust B's staff and patients work again (e.g. Bea's episode list loads).

**T-10. The setting is gone from Settings and the superadmin console**
1. As **Ada**, open **Settings**. **Expected:** no "Offer answers patients gave other trusts" among the trust's settings (Keep episode data for (years) is still there).
2. As **Max** (Hospital B1), the same for Trust B.
3. As **Sam Superadmin**, open `/admin`, then Trust A. **Expected:** no such setting there either.

### The archived episode

**T-11. A sent, then archived questionnaire: Sent, with a read-only View**
1. As **Cara**, create an episode on the **Demo pre-op HQ** for a new patient, e.g. "S27 T11". As the patient (phone size), answer both sections and **send**.
2. As **Sue Super**, open the episode and **Archive episode**.
3. As the patient, open the invite link again (or go to `http://localhost:3000/home` in the patient's tab).
4. **Expected:**
   - The link opens "Thank you, Sam. Your answers have been sent to the hospital…", with no Send button and nothing to fill in.
   - **Your questionnaires** (the home) shows it as **Sent** with **View**, not "Closed by the hospital". View opens the same thank-you page.
5. As Cara, create another episode for a new patient (e.g. "S27 T11b") and, **without the patient sending anything**, have Sue archive it. As that patient, open their invite. **Expected:** "This questionnaire is closed…", and their home says **Closed by the hospital** (as before).

**T-12. An archived episode hides NOT RESPONDING**
1. As **Cara**, create an episode for a new patient (e.g. "S27 T12"). Don't answer it.
2. Open `http://localhost:3000/qa/jobs` (as **Sam Superadmin** if asked) and run the jobs **as of 8 days from now**.
3. As Cara, open the episode. **Expected:** the red **Not responding** badge shows beside the status.
4. As **Sue Super**, archive it. **Expected:** the badge is gone (ARCHIVED shows). Reload: still gone.
5. Un-archive it (Sue, with a reason). **Expected:** the badge is back.

### The Cancelled message

**T-13. The default Cancelled text and email are neutral, from any status**
1. As **Ada**, open **Settings**. Under the status messages, switch on **Text the patient when the episode is cancelled** and **Email the patient when the episode is cancelled** for the whole trust, and save.
2. **Expected:** the **Cancelled text message** and **Cancelled email** templates read "Hello {{firstName}}, {{hospital}} has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." They don't mention a questionnaire to fill in.
3. As **Cara**, cancel four episodes (with any reason), one in each of these statuses:
   - HQ not complete (new, not answered);
   - Ready for review (sent);
   - Ready for POA (review completed);
   - Ready for admission.
4. On `/qa/messages`, check each patient's newest email, and their newest text if you gave them a mobile number. **Expected:** "Hello Sam, Hospital A1 has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." (the email with its greeting on a line of its own), reading correctly in every case. The email's **subject** is "Hospital A1: your pre-operative assessment was cancelled".
5. **Switch both settings back off** (Ada, Settings).

### The format names

**T-14. The publish refusal says "phone number"**
1. As **Alex**, create a new questionnaire (e.g. "S27 formats"), a Question Set, and a **Text** question "Your contact number" (Format: Any text). **Publish** it.
2. **Create new version**. In the draft, set that question's **Format** to **Phone number**. Press **Publish**.
3. **Expected:** the Publish window refuses it, naming the question: "…its answer format changed from any text to **phone number**". No pattern such as `^\+?[0-9 ()-]…` appears.
4. Try **NHS number**, then **UK postcode**. **Expected:** "…to NHS number", "…to UK postcode".
5. Try **Custom pattern** with `^[0-9]+$`. **Expected:** "…to the pattern “^[0-9]+$”" (a custom pattern is still shown as itself).
6. Set it back to **Any text**. **Expected:** it publishes.

### Clickable POA Summary answers

**T-15. Answers open Validate at the question; Back returns to the same place**

The seeded Demo pre-op HQ has one disclosure: diabetes **Yes** gives "Diabetic".
1. As **Cara**, create an episode on the **Demo pre-op HQ** for a new patient (e.g. "S27 T15"). As the patient, answer both sections with **diabetes Yes**, snoring No and sick after an anaesthetic No, and send.
2. As Cara, validate **About you** as it is. In **Your health**, change **snoring to Yes** (first page) and **"Have you ever been sick… after an anaesthetic?" to Yes** (second page, Anaesthetics). Validate **Nurse checks** too.
3. Make the window short (about 500 px high), so the summary scrolls. Open the **POA Summary** and scroll down to **Your health**. **Expected:** "Diabetic" under Health Summary, and the "Changed by the clinician on Health" and "…on Anaesthetics" boxes, all as links.
4. Click **Diabetic**. **Expected:**
   - **Validate** for Your health opens, with "Do you have diabetes?" **in the middle of the screen**, ringed in yellow for a moment.
   - A **← POA Summary** link shows beside the patient's name.
5. Press the browser's **Back**. **Expected:** the POA Summary's **Validated summary** tab, scrolled so **Diabetic** is where it was on the screen.
6. Click the snoring line in **"Changed by the clinician on Health"**. **Expected:** Validate at "Do you snore loudly?", highlighted. Press **← POA Summary**. **Expected:** back at the same place.
7. Click the line in **"Changed by the clinician on Anaesthetics"**. **Expected:** Validate opens **on the Anaesthetics page**, at that question, highlighted.
8. Back on the POA Summary, switch to **Patient answers**. **Expected:** the address ends `?tab=patient`. Click **Diabetic** there. **Expected:** Validate at diabetes. Press **Back**. **Expected:** the **Patient answers** tab, at the same place. Reload: still Patient answers.
9. Open T-01's Trust A episode's POA Summary (T-06), and click a line in a **"Pre-filled on …"** box. **Expected:** Validate at that question.
10. Open a Validate address directly in a new tab, e.g. `…/episodes/<id>/sets/<set id>?q=snoring` (copy it from step 6 and open it fresh). **Expected:** it opens at snoring, highlighted, with **no** "← POA Summary" link (it wasn't opened from there).

**T-16. Print renders the POA Summary to PDF**
1. On T-15's POA Summary (Validated summary), press **Print**, then **Save as PDF**. Open the PDF.
2. **Expected:**
   - The summary's content, with each answer line readable: the links print as plain text, with no extra symbols or addresses.
   - Nothing is cut off at the sides.

   (The print-designed PDF is plan C9: tabs and web elements still showing is known, not a failure.)

## Regression (earlier sessions)
- **R-01 (S25):** T-03's same-trust follow-up: **Everything is still correct** jumps to the first question not filled in (pregnancy), and Send works once both sections are done.
- **R-02 (S26):** Validate's "Changed by patient: pre-filled “…”" still shows under a pre-filled answer the patient changed (e.g. in T-04, snoring changed to No).
- **R-03 (S23):** in T-14's draft, rewording an option of a published Select One question is still refused at Publish, naming the option.
- **R-04 (S20–S22):** cancelling and archiving an episode still make it read-only. The patient's home says **Cancelled by the hospital** for a cancelled one.
- **R-05 (S04/J4):** a clinician's correction in Validate still shows "Patient answered: …" and saves.

## Out of scope (don't report)
- The print-designed POA Summary PDF (plan C9). Today's Print shows the web page's tabs and elements.
- Merging one person's two accounts (C10).
- IG/DPO sign-off on copying across trusts (C10).
- The patient's View of a sent questionnaire is the thank-you page, not a list of their answers: as before S27.
- Known older items in `docs/pending.md`, e.g. a false "Patient answered: no answer" under a list's column in Validate.
