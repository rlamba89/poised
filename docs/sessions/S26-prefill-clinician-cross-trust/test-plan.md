# S26 manual test plan: pre-fill for the clinician, and answers from other trusts

**Changed after verification 1 (and Rahul's answer):**
- **Another trust's answers match by key only for a platform library key** (the starter library's, the same in every trust), and otherwise by exact wording. Every choice is mapped by its option's text. A trust's own key never matches another trust's by key.
  - In **T-07** the questions the seeded Demo HQs took from the platform library (smoking, alcohol, height, snoring, sickness after an anaesthetic, the anaesthetic problems, allergies, medicines) say "**(matched by link)**".
  - "Do you have diabetes?" says "**(matched by same wording)**" (it's a trust's own key).
- **Trust A keeps no Trust B id** in its answers (no `fromForm` on Trust B entries; T-07 checks the staff API).
- A refused consent answer keeps its message on screen (automated: J16b; hard to cause by hand).

**Changed after test report 1:**
- **T-12:** Trust B's audit row now reads "Patient's answers **read** to pre-fill their HQ at another trust (they agreed)" (found 4).
- **T-13:** a bad `orgId` now gets "We couldn't tell which answers this is about. Please reload the page." (found 2).

**For the tester:** you've never seen the code. Use only the browser (and, where a step says so, the browser's console). Report each case as Pass, Fail (with what you saw) or Blocked (why). The brief is [brief.md](brief.md), and Rahul's answers are in [answers.md](answers.md).

## What this session added
1. **The clinician's pre-fill markers (PRE-04).** When a patient's answers were filled in from an earlier questionnaire (S25), the clinician now sees:
   - on **Validate**, under each pre-filled answer: where it came from ("Pre-filled from Demo pre-op HQ, 23 Oct 2026 (matched by link)" or "(matched by same wording)"), plus the hospital when it was another trust's;
   - "Changed by patient: pre-filled “Yes”" where the patient changed a pre-filled answer;
   - a line above the set: how many answers were pre-filled, how many the patient changed, and which button of the opening card the patient pressed ("Everything is still correct" or "Check each answer");
   - on the **POA Summary**, the same: a "Pre-filled on <page>" box per page, a **Changed by patient** badge, and the button line.
2. **Answers from another trust.** Trust B's answers are offered at Trust A only when **both** trusts have **Offer answers patients gave other trusts** switched on (Settings). The patient is asked once per trust, before the first section: "Use the answers you gave Hospital B1 in October 2026?", with **Yes, use them** and **No, thanks**.
   - **Yes:** the questions the patient's Trust A history doesn't cover are filled from Trust B.
   - **No:** nothing from Trust B, and the patient isn't asked again.
   - **On the patient's behalf:** staff filling in for the patient never get Trust B's answers, and never see the question.
   - **The audit log:** reading Trust B's answers is logged under Trust B as well.

## Setup

**Rahul's terminal** (the conductor does this in an automatic run):
```sh
docker compose up -d db
make migrate seed      # takes the database to migration 00026 and seeds the demo data
make dev               # leave it running
```

**Open:** http://localhost:3000

**Staff sign-in:** the stub login page lists the seeded users; click a name to sign in. Sign out (top right) before signing in as someone else, or use a second browser profile or an incognito window. The users you need:
- **Cara Clinician**: a clinician at Hospital A1 (Trust A). Creates Trust A episodes and validates them.
- **Ada Admin**: an admin of the whole of Trust A. Uses Trust A's **Settings** and **Audit**.
- **Bea Author**: a super clinician of the whole of Trust B. Creates Trust B episodes at **Hospital B1** (pick Hospital B1 in the hospital picker).
- **Max Multi**: an admin of the whole of Trust B (and a clinician at A2). Uses Trust B's **Settings** and **Audit**: pick Hospital B1 in the picker.

**Patients:** there's no patient sign-in page. A new episode sends the patient an invite email, which is captured, not sent.
- Open **http://localhost:3000/qa/messages** (sign in as **Sam Superadmin** if it asks), find the newest email to the patient's address, and open its link (`/i/…`).
- Then press **Continue** and enter the patient's date of birth.
- **Test patients at phone size (390 × 844)**: Chrome's device toolbar (iPhone 12 Pro or a 390 × 844 custom size), or a window that narrow.

**The same person in two trusts:** create the episode at each trust with the **same first name, surname, date of birth and email**. When the patient enters their date of birth from the second trust's invite, both records join one account (S15). Use a fresh made-up email for each person, e.g. `s26.t05@example.test`, and a surname you'll recognise, e.g. `S26 T05`.

**Settings:** the setting is in **Settings** (whole-trust admins only), under the trust's settings: **Offer answers patients gave other trusts**. It's off by default. Each trust's admin switches their own trust's.
- **Before you start, check that it's off in both trusts** (Ada for Trust A; Max at Hospital B1 for Trust B).
- **Put both back to off at the end.**

**Answering the seeded HQs:** the **Demo pre-op HQ** (both trusts) has two patient sections:
- **About you:** smoking, alcohol, height, weight;
- **Your health:** diabetes, snoring, pregnancy, sickness after an anaesthetic, anaesthetic problems and what happened, family problems, allergies, regular medicines.

The **Demo follow-up HQ** (Trust A only) asks most of the same questions, plus "Can you climb two flights of stairs without stopping?". Its "Do you have diabetes?" matches the pre-op HQ's by wording only. Weight and pregnancy are **Always ask fresh**: they're never pre-filled. Answer every required question, so you can send.

## Test cases

### The clinician's markers (one trust)

**T-01. Validate shows where each pre-filled answer came from (PRE-04)**
1. As **Cara** (Hospital A1), create an episode on the **Demo pre-op HQ** for a new patient, e.g. Sam "S26 T01", date of birth 17/05/1980, a fresh email.
2. As the patient (invite link, phone size), answer **About you** (smoker: I used to smoke, alcohol 6, height 170, weight 82). Then answer **Your health**:
   - diabetes: No; snoring: No; pregnant: No;
   - **sick after an anaesthetic: Yes**;
   - problem with an anaesthetic: Yes, with "I was sick all night";
   - family: No;
   - one allergy: Penicillin / A rash.

   Send it.
3. As **Cara**, open the episode and validate each section, changing **snoring to Yes** in Your health. Validate Nurse checks, then complete the review.
4. As Cara, create a second episode for **the same patient** (find them by search) on the **Demo follow-up HQ**.
5. As the patient, open the new invite (phone size). In **Your health**, press **Everything is still correct**. Change **"Have you ever been sick… after an anaesthetic?" to No**, answer pregnancy, and finish the section.
6. Open **About you**, press **Check each answer**, fill in weight and stairs, and finish. **Send**.
7. As Cara, open the second episode and click **Your health** (Validate).

**Expected:**
- A line above the set: "6 answers here were pre-filled from the patient's earlier questionnaires, and the patient changed 1. The patient chose “Everything is still correct” on <today>, so these pre-filled answers weren't checked one by one."
- Under **Do you have diabetes?**: "Pre-filled from Demo pre-op HQ, <today, e.g. 23 Oct 2026> (matched by same wording)".
- Under **Do you snore loudly?**: "… (matched by link)", and the answer is **Yes** (the clinician's correction was the one used).
- Under **sick after an anaesthetic**: the source line, plus **"Changed by patient: pre-filled “Yes”"** in orange, and the answer shown is No.
- **Pregnancy** has no marker. **Problem with an anaesthetic**, **What happened** and **Your allergies** (next page) have markers without "Changed".
- **About you:** the line says "… The patient chose “Check each answer” on <today>." **Smoking, alcohol and height** have markers. Weight and stairs have none.

**T-02. The POA Summary shows the same**
1. From T-01's second episode, open **POA Summary**.

**Expected:**
- **Validated summary**, section **Your health**:
  - the button line ("The patient chose “Everything is still correct” on …");
  - a box **"Pre-filled on Page 1"** listing diabetes (same wording), snoring, sickness, problem and what happened, each "Pre-filled from Demo pre-op HQ, <date> (matched by …)";
  - the sickness line has a **CHANGED BY PATIENT** badge and "pre-filled “Yes”, patient answered “No”";
  - a **"Pre-filled on Page 2"** box with Your allergies.
- **About you:** its own boxes and the "Check each answer" line.
- **Patient answers** tab: the same pre-filled boxes and badge.
- Nothing about pregnancy or weight.

**T-03. Only questions shown are marked**
1. Use a reviewed-but-not-completed episode, i.e. one at **Ready for review**: repeat T-01 steps 4–6 for the same patient (a third episode) and don't validate.
2. As Cara, open **Your health** (Validate). Change **"Have you ever had a problem with an anaesthetic?"** to **No**.
3. Change it back to **Yes**.

**Expected:**
- **Step 2:** "What happened with your anaesthetic?" disappears, and its marker with it. No marker shows for a hidden question.
- **Step 3:** the question and its marker come back. The line above the set still counts what was shown when the page opened (6); a reload recounts.

**T-04. Staff pressed the card's button with the patient**
1. As Cara, create another Demo follow-up HQ episode for T-01's patient. On the episode page choose **Fill in for the patient**.
2. In Your health press **Check each answer**, answer pregnancy, and finish both sections. **Send the answers**.
3. Open Validate for Your health.

**Expected:**
- The line says "… Cara Clinician chose “Check each answer” with the patient on <today>."
- The orange "entered by Cara Clinician on the patient's behalf" note is there as before.
- The markers are as in T-01, without any "Changed by patient" unless you changed one.

### Another trust's answers

For T-05 to T-12, make **one new person** per case (or as each case says) with **only a Trust B history**:
1. As **Bea** at **Hospital B1**, create an episode on Trust B's **Demo pre-op HQ** for e.g. Sam "S26 T05", 17/05/1980, a fresh email.
2. As the patient (invite link), answer both sections (snoring **Yes**, diabetes **No**, the rest as you like) and **send**.
3. As **Cara** at Hospital A1, create an episode on the **Demo follow-up HQ** for the **same name, date of birth and email** (a new patient for Trust A).

**T-05. Not offered unless both trusts allow it**
1. Make the person as above. **Ada** switches Trust A's setting **on**; Trust B's stays **off**.
2. As the patient, open Trust A's invite (phone size).
3. **Max** switches Trust B's setting **on** and **Ada** switches Trust A's **off**. The patient reloads.

**Expected:**
- **Steps 2 and 3:** no question about other answers. The sections show **Start** (nothing is filled in), and there's no "Please check".

**T-06. Both on: the question names Trust B's hospital and month (phone size)**
1. Continue from T-05: **Ada** switches Trust A's setting back **on** (both are now on). The patient reloads.

**Expected:**
- Before any section: a card **"Use your earlier answers?"**:
  - "Use the answers you gave **Hospital B1** in **<this month and year, e.g. October 2026>**?";
  - a short explanation;
  - **Yes, use them** and **No, thanks**.
- No section tiles show yet.
- At 390 × 844, no sideways scrolling, and both buttons are full width and readable.
- "Your questionnaires" (back to the home) still works. Coming back shows the question again.

**T-07. Yes fills from Trust B**
1. Continue from T-06: press **Yes, use them**.
2. Open **Your health**.
3. Finish both sections (answer pregnancy, weight and stairs) and **send**.
4. As Cara, open the episode and **Your health** (Validate).

**Expected:**
- **After Yes:** the sections show **Please check**, and the card is gone. A reload doesn't ask again.
- **Your health:** the card says "We've filled in N answers from your questionnaire on <the day Trust B's HQ was sent>". Diabetes (No) and snoring (Yes) are filled, each with "From your answers on <date>".
- **Validate (step 4):** each pre-filled answer says "Pre-filled from Demo pre-op HQ **at Hospital B1 (another trust)**, <date>":
  - followed by "**(matched by link)**" for the platform library's questions (snoring, sickness after an anaesthetic…);
  - followed by "**(matched by same wording)**" for diabetes.
- **As Cara**, in the browser, open `http://localhost:3000/api/o/<trust A id>/h/<hospital A1 id>/episodes/<episode id>/hq` (the ids are in the episode page's address). The Trust B entries in `prefill` have `"otherTrust": true` and a `hospital`, and **no `fromForm`**.

**T-08. Trust A's own answers come first**
1. Make a person with **both** histories: first T-01's steps 1–3 at Trust A (snoring **No**, reviewed), then a Trust B Demo pre-op HQ for the same person with snoring **Yes**, sent.
   - **Keep both settings off** while the Trust B one is filled in. Otherwise Trust B asks about Trust A's answers.
   - Then switch **both on**.
2. As Cara, create a Demo follow-up HQ episode at Trust A for this person. The patient opens it and presses **Yes, use them**.
3. Open Your health.

**Expected:**
- Snoring shows **Trust A's** answer (No, or what Cara validated), not Trust B's Yes.
- After sending, Validate shows "Pre-filled from Demo pre-op HQ, <date> (matched by link)", **without** "at Hospital B1 (another trust)", for the questions Trust A's HQ asked.

**T-09. No, thanks is remembered**
1. Make a new person (Trust B history, both settings on), and open Trust A's invite. Press **No, thanks**.
2. Reload.
3. As Cara, create another Demo follow-up HQ episode for the same person, and open its invite.

**Expected:**
- **Step 1:** the sections show **Start**, nothing is filled in, and there's no card.
- **Steps 2 and 3:** no question again; nothing from Trust B.

**T-10. Switching a setting off after a Yes**
1. Use T-07's person (they said Yes). **Max** switches Trust B's setting **off**.
2. As Cara, create another Demo follow-up HQ episode for them, and open its invite.
3. Look at T-07's episode again (Validate, or the patient's own view if it wasn't sent).

**Expected:**
- **Step 2:** nothing filled from Trust B, and no question.
- **Step 3:** the answers already copied are still there, with their markers.
- Switch Trust B's setting back **on** afterwards.

**T-11. On the patient's behalf: never Trust B's answers**
1. Use T-07's person (they said Yes), with both settings on. As Cara, create another Demo follow-up HQ episode for them.
2. On the episode page choose **Fill in for the patient**.

**Expected:**
- No question about other answers.
- **Nothing from Trust B** is filled in: no "Please check", unless the person also has a Trust A history (T-08's person would get Trust A's answers only).

**T-12. The audit log of each trust shows the reads**
1. As **Ada**, open Trust A's **Audit**, filtered to today.
2. As **Max** (Hospital B1), open Trust B's **Audit**, filtered to today.

**Expected:**
- **Trust A:** a row **"Patient answered whether to use another trust's answers"** for each Yes/No (T-07, T-09) naming the Trust A patient. **No Trust B patient is named** in Trust A's rows.
- **Trust B:** rows **"Patient's answers read to pre-fill their HQ at another trust (they agreed)"**, by the patient, at **Hospital B1**, naming the person's **Trust B** record. They're there for T-07's person (and T-08's), and **none for T-09's person** (they said No).

**T-13. Refused requests (browser console)**

On a patient page of T-09's person (signed in as the patient), open DevTools → Console. Run each line, replacing `<eid>` with the episode id from the address bar (`/home/episodes/<eid>`) and `<trustB>` with any uuid:
```js
const post = (eid, body) => fetch(`/api/p/episodes/${eid}/prefill-consent`, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify(body)}).then(async r => [r.status, await r.text()]);
await post("<eid>", {orgId: "<trustB>", agree: true});        // 409: nothing to answer about (or already answered)
await post("<eid>", {orgId: "not-a-uuid", agree: true});      // 400: "We couldn't tell which answers this is about. Please reload the page."
await post("<eid>", {orgId: "<trustB>"});                     // 400: "Choose “Yes, use them” or “No, thanks”."
await post("00000000-0000-0000-0000-000000000000", {orgId: "<trustB>", agree: true}); // 404
```
Then, in a **new incognito window** (not signed in), at http://localhost:3000, run the `post` line again with any ids.

**Expected:**
- The statuses as commented, each with a plain-English `error` message.
- **Signed out:** **401**.

**T-14. A stale tab can't overwrite a pre-filled section**
1. As Cara, create two episodes for a new Trust A patient: **E1** on the Demo pre-op HQ, then **E2** on the Demo follow-up HQ (same patient). Two invite emails are captured; open both links in the same browser (two tabs).
2. **Tab 1:** open E2 (the follow-up). Nothing is pre-filled yet (E1 isn't sent). Open **Your health** and leave it open without answering.
3. **Tab 2:** open E1, answer both sections, and send.
4. **Tab 3** (or tab 2 again): open E2 from the home or its link. Your health is now **Please check** (pre-filled from E1).
5. **Tab 1:** answer a question in Your health.

**Expected:**
- **Tab 1** says its answers weren't saved because they changed elsewhere ("Your answers weren't saved", with **Reload**). It does **not** silently replace the pre-filled answers.
- After **Reload**, the pre-filled answers and the card are there.

**T-15. The setting's help text**
1. As Ada, open **Settings** and find **Offer answers patients gave other trusts**.

**Expected:**
- The help reads: "Used only when the other trust has it on too, and the patient is always asked first. Never when staff fill in an HQ for the patient."

## Regression (earlier sessions)

**R-01. Same-trust pre-fill for the patient (S25).** In T-01 step 5 (phone size):
- the card says "We've filled in 6 answers from your questionnaire on <today>. Please check them.";
- **Everything is still correct** jumps to the pregnancy question, with its text on screen;
- changing an answer removes its marker.

**R-02. "Started" (S25/S16).** For a patient whose new Trust A HQ was pre-filled, **before** they touch anything:
- their **home** (Your questionnaires) shows the HQ as **To do**;
- after they answer one question, or press a card's button, it shows **In progress**.

**R-03. The clinician's corrections (S04/J5).** On T-01's POA Summary, Validated tab: **About you** or **Your health** still shows **Changed by clinician** where Cara changed an answer, with "Patient answered “…”, now “…”".

**R-04. A patient with no history (S14).** A brand-new patient's Demo pre-op HQ:
- no question, no cards, no markers;
- they can fill it in and send as before;
- Cara's Validate shows no pre-fill line.

**R-05. Staff on behalf (PAT-09).** Filling in for a patient with no history still works, and the episode moves to Ready for review on send.

## Out of scope: don't report
- Changing or withdrawing a consent decision (not built: S26 answers 3).
- Pre-fill across countries, and the hazard log entry.
- "Sex female" in lower case in the details box (known, in pending.md).
- A tab where you press a different card button after another tab pressed one: the server keeps the first (known, in pending.md).
- If the person has records in more than one other trust, the question is asked one trust at a time. Only Trust B exists in the seed, so this can't be tested by hand.
- QA (AWS): local only.
