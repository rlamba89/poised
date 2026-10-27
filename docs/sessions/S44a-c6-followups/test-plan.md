# S44a test plan: C6 follow-ups

**Retest after verification 2:** drag Question Sets into a new order in the editor's sidebar, and straight away edit a question in one of them (the autosave lands just after the drag): both save, with no error. Reorder twice quickly. Then Publish and try a reorder in a stale tab: "Only draft versions can be changed.". Plus a short regression: T-04, T-08, T-17 step 2, R-02.

**Retest after verification 1 (done in report 3):**
- T-04 and T-05 again.
- **New, the chain:** a patient set with Yes/No **B**, Yes/No **A** shown when B is Yes, and a text **C** shown when A is Yes. The patient answers B Yes, A Yes, C "severe", then B No, and sends. **Expected:** Cara's Validate shows only B. Correcting B to Yes shows A empty and no C answer.
- T-08's Publish, then, as Alex in a second tab while the first is still open, change a set's audience. A refused change after Publish says "Only draft versions can be changed." (hard to time by hand; the integration tests cover the race, so just check that editing a published version is refused with these words).
- T-18 (the suggested ASA) and T-02 (the ages).

**Retest after test report 1 (done in report 2):** T-06 (F1 fixed: a set whose answers were all dropped no longer says who entered them); T-16 (E4: a refused second Complete HQ review now refreshes the page to Ready for POA, keeping the message); T-09 (E3: no "?." in the message); T-17 step 2 (E1: the score card says "plus 1 line adding points"). Also re-run R-02 and R-05.

For a tester who has never seen the code. Use only the browser (and the captured-messages page).

## Setup

In your own terminal (an automatic run's conductor does this for you):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running: the API on :8080, the web app on :3000, the evaluate service on :8090
```

The local database was reset in this session (27 Oct): it holds only the seeded data and **HJE Full HQ**.

Open **http://localhost:3000**. The stub login page lists the seeded users; click one to sign in. Sign out from the header. Users used here:
- **Alex Author**: super clinician, the whole of Trust A (writes HQs, publishes, re-opens reviews).
- **Cara Clinician**: clinician at Hospital A1 (episodes, validation).
- **Val Clinician**: clinician at Hospital A2 only (another hospital of the same trust).
- **Bea Author**: super clinician of Trust B (another trust).

Helpers:
- **http://localhost:3000/qa/messages**: every captured text and email (the patient's invite link is in there).
- A patient opens their invite link, presses **Continue**, and gives their date of birth (made-up patients created on **New episode**; use date of birth 17/05/1980 so you know it).
- Patient pages are checked at **phone size, 390×844** (DevTools device toolbar, or resize the window).

**Building HQs.** Several cases need a small draft HQ. As Alex, **Questionnaires → New questionnaire**, add Question Sets and questions with the editor. To give an option points: the question's **Options** → switch on **Scores**. A **Calculation** question with **Calculate: A score total** is a score; its **Bands** are below. A Question Set's own condition is on its header ("Show this Question Set only when…"); a question's is its **Logic** tab. The **Rules** button in the editor header opens the HQ's rules; **Publish** opens the Publish window.

Unless a case says otherwise, publish an HQ only when the case asks, then make an episode for it as Cara (**New episode**, a new made-up patient with an email), and open the patient's invite from **/qa/messages**.

> If a sent HQ goes straight to **Ready for admission** (the default rules auto-triage low-risk patients), Alex can **Re-open HQ review** on the episode page to review it by hand; or, before publishing, delete the "Low risk" rule on the HQ's Rules page.

## Test cases

### Scores: a hidden question adds 0

**T-01. A score with a conditional question: hidden → a band; shown and skipped → Incomplete (Preview).**
1. As Alex, make a draft HQ "MT S44a scores" with one patient Question Set **Sleep** holding:
   - Yes/No **Do you snore?** with Scores on (Yes 1, No 0);
   - Yes/No **Is it loud?** with Scores on (Yes 2, No 0) and the condition (Logic) "Do you snore? is Yes";
   - a **Calculation** "Sleep risk", *A score total*, adding up both questions, with bands **Low** up to 1 and **High** above.
2. Open the set's **Preview**, Clinician view.
3. Answer Snore **Yes**, leave Loud unanswered. **Expected:** Sleep risk shows **Incomplete**.
4. Answer Loud **Yes**. **Expected:** **3 (High)**.
5. Change Snore to **No** (Loud disappears). **Expected:** **0 (Low)**: the hidden Loud adds 0, even though its old Yes is still in the form.
6. Look at the score's settings card in the editor. **Expected:** the help text under the score says a question a condition hides (or its whole Question Set) adds 0, and a line testing it adds nothing.

**T-02. A hidden Question Set adds 0; the same in the clinician's view of a real episode.**
1. Same HQ: add a second patient set **Nights** *before* Sleep with Yes/No **Do you wake at night?** (Scores on, Yes 1), with the Question Set condition "Patient's age is under 18". In Sleep's score, also add up **Do you wake at night?**.
2. Delete the "Low risk" rule on the Rules page (so nothing auto-triages). Publish.
3. As Cara, make an episode (patient born 17/05/1980, so 46: Nights is hidden). Patient (phone size): Snore Yes, Loud No; send.
4. As Cara, open the episode, **Validate** Sleep. **Expected:** Sleep risk **1 (Low)** (Nights hidden adds 0; not Incomplete).
5. POA Summary: **Expected:** the same score, 1 (Low).

**T-03. "Add N points when…" testing a hidden answer doesn't fire.**
1. Draft: in a score, add a points line "Add 5 points when Is it loud? is Yes" (Loud only shown when snoring).
2. Preview, Clinician: Snore Yes, Loud Yes → the 5 points count. Snore → No. **Expected:** the 5 points go (score drops), though Loud's Yes is still in the form.

### On send, keep only what the patient's form showed

**T-04. The snoring example: the follow-up's answers aren't stored; a clinician's correction shows it empty.**
1. As Alex, a draft HQ with a patient set **About you** (Yes/No **Do you snore?**, required) and a second patient set **Sleep follow-up** with the Question Set condition "Do you snore? is Yes", holding a required Yes/No **Do you stop breathing at night?** and a text **Tell us more**. Delete the "Low risk" rule. Publish.
2. As Cara, an episode for a new patient. As the patient (phone size): About you → Snore **Yes**. Sleep follow-up appears: answer **Yes** and type "loud" in Tell us more. Go back to About you, change Snore to **No** (the follow-up set disappears from the list). Send.
3. As Cara, open the episode. **Expected:** only About you is listed (Sleep follow-up hidden).
4. Validate About you and correct Snore to **Yes**; go back to the episode. **Expected:** Sleep follow-up is listed again, **Not started**.
5. Open its Validate. **Expected:** both questions are **empty**, with no "Patient answered: Yes"/"loud" under them; nothing says the patient answered them.
6. POA Summary, **Patient answers** tab. **Expected:** no answers for Sleep follow-up.

**T-05. A question hidden inside a shown set is dropped too.**
1. Draft: one patient set with Yes/No **Do you smoke?** and a text **How many a day?** whose condition is "Do you smoke? is Yes". Publish (no Low risk rule).
2. Patient: Smoke **Yes**, How many "10", then Smoke **No**. Send.
3. Cara: Validate the set, change Smoke to **Yes**. **Expected:** How many a day? is empty (no "10", no "Patient answered: 10").

**T-06. Staff filling in for the patient: the same.**
1. On a new episode of T-04's HQ, as Cara use **Fill in for the patient** on the episode page. Do T-04 step 2's answers (Yes, follow-up answered, back to No) and send.
2. **Expected:** as T-04 steps 3–5.

**T-07. Everything shown: nothing dropped.** A patient on T-04's HQ answers Snore **Yes** and the follow-up, and sends. **Expected:** Cara sees the follow-up set with the patient's answers, as before.

### Publish refuses a patient condition on clinician-only content

**T-08. A patient set conditioned on a clinician-only score.**
1. As Alex, a draft with a patient set **Sleep** (scored Yes/No question and a Calculation score **Sleep risk** with bands Low/High, the score marked **Clinician only**), then a patient set **Sleep follow-up** with the Question Set condition "Sleep risk band is High".
2. Press **Publish**. **Expected:** Publish is blocked, listing **Sleep follow-up · “Sleep follow-up” shows only when Sleep risk band is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set.**
3. Untick Clinician only on the score; Publish again. **Expected:** that problem is gone.
4. Put Clinician only back, and instead change Sleep follow-up's audience to **Clinician**. **Expected:** no such problem.

**T-09. A patient question conditioned on a clinician-only question in its own set; and on a clinician Question Set's question.**
1. In a patient set, a **Clinician only** Yes/No "Nurse: concerns?" and a patient text question whose condition is "Nurse: concerns? is Yes". Publish. **Expected:** blocked: "“<the text question>” shows only when …, but patients can't see Nurse: concerns?. Make the question visible to patients, …".
2. A clinician Question Set **Nurse checks** first, then a patient set conditioned on its question. **Expected:** blocked: "… but patients can't see …. Move the condition to a clinician Question Set, or what it tests to a patient Question Set."

### Warnings that don't block

**T-10. Auto-triage can't work: the Rules page and the Publish window warn, and Publish still works.**
1. As Alex, a draft HQ (keep the default rules, which auto-triage). A patient set with a scored Yes/No "Do you snore?"; a **clinician** Question Set **Nurse checks** with a scored Yes/No **Neck circumference over 40 cm?** and a Calculation score **Neck size risk** adding up both.
2. Open **Rules**. **Expected:** a yellow box "Auto-triage won't work for this HQ (you can still publish)" with: "Auto-triage won't move patients on for this HQ: the score “Neck size risk” needs answers a clinician enters (“Neck circumference over 40 cm?”), so it's always incomplete when the patient sends. These patients go to Ready for review."
3. Press **Publish**. **Expected:** the same line under "Before you publish (these don't stop it)", and the **Publish** button is enabled (no problems). Publish. **Expected:** it publishes.
4. On the Rules page, delete every rule with an auto-triage action (Low risk). **Expected:** the warning goes.

**T-11. An ASA guide code left ungraded: Publish warns; "No ASA grade" is remembered.**
1. As Alex, a draft: a Yes/No **Do you have diabetes?**; on **Yes**, add a disclosure with code ICD10 **E11.9** (Type 2 diabetes). The ASA guide box offers two grades. Choose **No ASA grade**, save the disclosure.
2. Reopen the disclosure. **Expected:** **No ASA grade** is still selected (before S44a it showed nothing selected).
3. Press Publish. **Expected:** no ASA warning for it.
4. Use the question's options **From the code library** (or any way that leaves a disclosure with E11.9 and no grade and no "No ASA grade"), then Publish. **Expected:** a yellow warning naming the set, the question/answer and "“Type 2 diabetes (E11.9)”, which the ASA guide grades ASA II or III, but no ASA grade was chosen. Choose one in its disclosure, or No ASA grade."; Publish is still enabled.

### The patient's thank-you page

**T-12. One neutral wording, at phone size.** As a patient (390×844), send any HQ. **Expected:** "Thank you, <first name>" and "Your answers have been sent to the hospital. We'll contact you about what happens next." Nothing about a team reviewing them. Reload the link: the same page. Also for an auto-triaged send (a seeded **Demo pre-op HQ** episode): the same words.

### The rules' suggested status

**T-13. The suggestion disappears once that status is reached.**
1. As Alex, a draft HQ; on Rules replace the rules with one: "the suggested ASA is at most II" → **suggest Ready for POA**. Publish.
2. Episode; patient answers so no ASA grade, and sends. As Cara: **Expected:** "The HQ's rules suggest: Ready for POA (the rule “…”)".
3. Validate every set and **Complete HQ review** (→ Ready for POA). **Expected:** the suggestion line is gone. The status history still shows the send.
4. On the same episode, set **On hold**. **Expected:** still gone (it reached Ready for POA before going on hold).
5. Another episode on the same HQ, sent (the line shows), then **Cancel episode**. **Expected:** no suggestion line on the cancelled episode.

### The held message goes once

**T-14. On hold → Ready for admission → Checked sends the Ready for admission message once.**
1. As Ada Admin, in **Settings** for Hospital A1, make sure the Ready for admission text/email message is on.
2. As Alex, a draft HQ keeping the Low risk rule; on Rules tick "wait for a clinician to click Checked before the patient's status message goes". Publish.
3. Episode for a new patient with a mobile; the patient sends with nothing giving an ASA grade. **Expected:** Ready for admission, auto-triaged, "The patient's message goes when you click Checked"; /qa/messages has only the invite.
4. As Cara: set **On hold**, then **Ready for admission**. **Expected:** the Ready for admission message is in /qa/messages once.
5. Click **Checked** on the auto-triage card. **Expected:** no second Ready for admission message.

### Clearer messages

**T-15. A Checked from a stale tab.** On an auto-triaged episode, open it in two tabs as Cara. Click **Checked** in the first, then in the second. **Expected:** the second shows "Someone checked this while you had it open." and the card shows "Checked by Cara Clinician on …".

**T-16. A second Complete review** (needs two tabs): open an episode Ready for review with every set validated in two tabs, click **Complete HQ review** in both. **Expected:** the second says "The HQ review is already complete." (not "…so the answers can't be changed").

### "Also when" on an author's band

**T-17. An "also when" band in the editor, the Preview and a patient's send.**
1. As Alex, a draft patient set with two scored Yes/No questions (Yes 1 each) and a score "Sleep score" adding them up, bands **Low** up to 2, **High** above.
2. Under the High band, click **Also High when…**, pick **The score** "is at least" **2**; **Add line**, pick **Patient › Sex** "is" **Male**. **Expected:** "Also High when all of these hold:" with the two lines; saved (All changes saved). Reload the editor: they're still there.
3. Preview, Clinician, sample patient male: both Yes → **2 (High)**. Sex Female → **2 (Low)**. Sex Not known → **Incomplete**.
4. Save a test case for the male answers expecting the High band's disclosure (add a disclosure to High first); change sex in the test case → it fails.
5. Publish (no Low risk rule), an episode for a male patient (sex Male on New episode); patient answers both Yes and sends. Cara Validate: **Expected:** 2 (High).
6. Delete one of the questions the "also when" tests (e.g. make a line test a question, then delete that question) and press Publish. **Expected:** a problem naming the band: "“Sleep score” band “High”: it tests something that no longer exists (…)".

### The suggested ASA

**T-18. The suggested ASA on Validate changes as an answer is corrected; and only once the patient has sent.**
1. As Alex, a draft: Yes/No **Do you smoke?** with Yes → a disclosure with SNOMED Smoker and ASA II; Yes/No **Heart failure?** with Yes → ICD10 I50.9 and ASA III. Publish (no Low risk rule).
2. Make an episode. Before the patient sends, open the episode page: **Expected:** no "Suggested ASA" line.
3. The patient answers both Yes and sends. Episode page: **Expected:** "Suggested ASA III: …".
4. Validate the heart-failure set. **Expected:** the line "Suggested ASA III: Heart failure …" above the form. Change Heart failure to **No**. **Expected:** it changes at once to "Suggested ASA II: Smoker …" (no reload). Back to Yes → ASA III again.

### The age in the hospital's time zone

**T-19. (Not testable at the right moment by hand.)** The age used on the episode page, Validate and the POA Summary is now worked out on the day of sending in the hospital's time zone (it only differs near midnight on a patient's birthday). Check only that the episode page still shows age-conditioned sets as before: on T-02's HQ, Nights stays hidden for a 46-year-old.

### Refused paths

**T-20. Another hospital, another trust.** As Val Clinician (Hospital A2) and as Bea Author (Trust B), paste the URL of T-04's episode, its Validate page and T-10's Rules page. **Expected:** not found / no access, as before; nothing about the HQ shown.

**T-21. Reload and Back.** On Validate (T-18 step 4) after correcting an answer, reload: the suggested ASA matches the saved answers. On the Rules page (T-10), reload: the warning is still there. Browser Back from Validate returns to the episode page.

## Regression (earlier sessions)

- **R-01 (S38):** STOP-Bang inserted ready-made still scores as before in the Preview (e.g. all Yes, male, 51 → High).
- **R-02 (S41/S44):** A patient whose required question is unanswered can't send; with every set complete the send works and the episode goes to Ready for review (or auto-triages).
- **R-03 (S42):** A low-risk send on a seeded Demo pre-op HQ is auto-triaged to Ready for admission and appears on the **Auto-triaged** worklist; Checked takes it off.
- **R-04 (S43):** A high-risk send (ASA III) adds the "Book anaesthetic review" task and the "Anaesthetic review" flag.
- **R-05 (S44):** Complete review with a set not validated is refused, naming the set.
- **R-06 (S11):** The trust's **Audit** page lists and exports a CSV (Ada Admin).

## Out of scope (don't report)

- The ASA grade history (S44b), the Lambda and anything on AWS (S45).
- The production policy for changes to clinical logic (C10).
- Display ages on the episode and patient pages ("age 46") still use the browser's date; only the age the HQ's rules use moved to the hospital's time zone.
- The stored result of a send (not visible in the UI) now also holds what was sent.
- Cosmetic items already in pending.md.
