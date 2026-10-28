# S44c test plan: C6's last open items

**Retest after verification 3 (test report 5).** A new migration (00036) came with it: run `make migrate` before `make dev`.
- **F4, a BMI's weight:** as Alex, a new HQ "MT S44c weight" with one patient set **Weight**: page 1 a **BMI** question; page 2 a long text **Tell us about your weight** whose Logic is "BMI is greater than 35". Delete Low risk; Publish. As the patient at phone size: height **170**, weight **120**, **Next**, type "Diet plan"; wait for **Saved**. **Previous**; correct the weight to **125** (click in the box, Backspace, then 5); wait for Saved; **Next**. **Expected:** "Diet plan" is still there; **reload**: still there. Then weight **60** (BMI under 35) and wait: the follow-up page's answer goes (Next shows it hidden or empty; reload agrees).
- **F4, a number that feeds a score:** on an HQ with STOP-Bang and a number (e.g. "How many units…" from report 4's MT S44c units) the report-4 units case once more: 20 → 25 keeps "Tell us more".
- **The chain still clears at once:** T-01 steps 4–7 at phone size (choices clear immediately).
- **F5:** as Alex, on a draft HQ with your own banded score (e.g. a Calculation "Frailty" with bands), edit Low risk: tick "and a score isn't in a band", choose Frailty / its top band; Save rule. Then delete the Frailty Calculation from the set and open **Publish**. **Expected:** a problem "Rule “Low risk”: its score isn't in this HQ any more" and Publish refused. An HQ without STOP-Bang and the default rules still publishes (no problem for STOP-Bang).
- **Answer 8, Start fresh after the app clears:** it needs a patient with another trust's earlier answers in a set that a choice in another set hides and shows again, which takes a cross-trust set-up (S26/S27). If you can't set that up by hand, mark it covered by the integration test `TestStartFresh` ("a clearing save by the form leaves Start fresh open…").
- Short regression: T-03, R-05.


**Retest after verification 2 (test report 4):**
- **F3, a typed number's follow-up:** as Alex, a new HQ "MT S44c units" with one patient set **Alcohol**: a number question **How many units of alcohol a week?** and a long-text **Tell us more about your drinking** whose Logic is "units is greater than 14". Delete Low risk; Publish. As the patient, at phone size: units **20**, Tell us more "Wine with dinner"; wait for **Saved**. Click in the units box and correct it to **25** (Backspace, then 5). **Expected:** "Tell us more" keeps "Wine with dinner" (it may blink away for an instant while you type); after **Saved**, **reload**: still 25 and "Wine with dinner". Now change units to **2** and wait: the follow-up goes; reload: units 2, and changing it back to 20 shows the follow-up **empty**.
- **The chain still clears at once:** T-01 steps 4–7 at phone size (B No hides A and C at once; B Yes, A Yes, then B No before typing in C: **Finish this section** works, with no "Response required" on a hidden C).
- **Fill in for the patient:** the units case once more, as Cara on the on-behalf form.
- Short regression: T-03, T-12 step 2, R-05 (pre-fill).


**Retest after verification 1:**
- **F1 (opening a set cleared answers):** the case needs an answer that's already hidden when the set opens (e.g. pre-filled into a question an "is not" condition hides), which the UI can't set up by hand; the unit test `respondent.test.ts` "loading answers clears nothing, even under an "isn't" condition" covers it. In the browser, check nothing regressed: on MT S44c chain, a patient **Review**s a finished Breathing set: the status stays **Saved** (no "Saving…" on opening), and B Yes → No still clears A and C at once. The same through **Fill in for the patient**.
- **F2:** on an HQ with a score that has bands **and no STOP-Bang**, open Rules → edit **Low risk**: the "Score not in a band" list offers the HQ's score **and** "STOP-Bang score (not in this HQ)". Choose the HQ's score, then open the list again: **STOP-Bang score (not in this HQ)** is still there; choose it back and pick **High risk**; Save rule. Words read "…STOP-Bang score isn't High risk (not in this HQ)…". On an HQ with **no** banded score, the "and a score isn't in a band" box can be ticked and offers STOP-Bang.
- Short regression: T-01 (the chain at phone size), T-07, T-10 steps 1–2.


**Changed after test report 2:** T-14 steps 2–3 can't be done in the UI (at Ready for review the status is locked until the review is complete), so the "isn't waiting for review any more" wording is covered by the integration tests (`TestCompleteReview`) and checked through the API; only step 4 is a browser step.

**Retest after test report 1** (5 passed; F1 fixed; 15 cases blocked by the browser):
- **F1 fixed:** the Sample patient's **Sex** in the set's **Preview** and on **Test the whole HQ** now offers Female, Male, **Other** and **Not known**. Run **T-11** in full (step 1 in the Preview's **Clinician view**, where the score's value shows; also choose Other on Test the whole HQ: Low risk is then held, "no auto-triage: a score is Incomplete").
- **Cosmetic fixed:** on the Rules page, editing Low risk on an HQ without STOP-Bang, the score box is wider, so "STOP-Bang score (not in this HQ)" fits (T-10 step 1).
- **The blocked cases:** T-04, T-05, T-06, T-08 (the score values: use the Preview's Clinician view, as Test the whole HQ shows only the rules), T-09, T-10 steps 3–4, T-11 step 3, T-12, T-13, T-14, T-15, and R-01 to R-05.
- T-01 to T-03 and T-07 passed; no need to repeat them. The HQs and episodes from report 1 can be reused.

For a tester who has never seen the code. Use only the browser (and the captured-messages page).

## Setup

In your own terminal (an automatic run's conductor does this for you):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running: the API on :8080, the web app on :3000, the evaluate service on :8090
```

Open **http://localhost:3000**. The stub login page lists the seeded users; click one to sign in. Sign out from the header. Users used here:
- **Alex Author**: super clinician, the whole of Trust A (writes HQs, publishes, archives, re-opens reviews).
- **Cara Clinician**: clinician at Hospital A1 (episodes, validation, status changes).
- **Ada Admin**: admin of Trust A (Settings).

Helpers:
- **http://localhost:3000/qa/messages**: every captured text and email (the patient's invite link is in there).
- A patient opens their invite link, presses **Continue**, and gives their date of birth. Create made-up patients on **New episode** with date of birth **17/05/1980** and an email, so you can find their invite.
- Patient pages are checked at **phone size, 390×844** (DevTools device toolbar, or resize the window).

**Building HQs.** As Alex, **Questionnaires → New questionnaire**, add Question Sets and questions in the editor. A question's condition is its **Logic** tab; a Question Set's own condition is on its header ("Show this Question Set only when…"). **Ready-made score** on an Add bar inserts STOP-Bang (it needs a BMI question first, in the same or an earlier set). The **Rules** button in the editor header opens the HQ's rules; **Publish** opens the Publish window.

> A new HQ gets the platform's default rules, which **auto-triage** a low-risk patient on send (straight to Ready for admission). For cases that need the review by hand, Alex can **Re-open HQ review** on the episode page, or delete the "Low risk" rule on the HQ's Rules page before publishing.

**The "chain" HQ used by T-01 to T-03 and T-11.** As Alex, a new HQ **"MT S44c chain"** with two patient Question Sets:
1. **Breathing**, holding:
   - Yes/No **Do you get short of breath?** (B), required;
   - Yes/No **Is it worse at night?** (A), required, Logic: shown when B is Yes;
   - Text **What wakes you?** (C), required, Logic: shown when A is Yes.
2. **Pillows**, with the Question Set condition "**Is it worse at night?** is Yes", holding Yes/No **Do you use more than two pillows?** (D), required.

On its Rules page, **delete the "Low risk" rule** (so a send stays at Ready for review), then **Publish**.

## Test cases

### 1. The patient's form clears an answer as soon as its question is hidden

**T-01. The chain, at phone size (390×844).**
1. As Cara, New episode for a new made-up patient (email, DOB 17/05/1980) on **MT S44c chain v1**. Open the invite from /qa/messages in a phone-size window; Continue; date of birth.
2. **Expected:** the list shows only **Breathing** ("0 of 1 sections complete"); Pillows isn't listed.
3. Start Breathing: B **Yes**, A **Yes**, type "Coughing" in C. **Finish this section**. **Expected:** Pillows is now listed. Start it, answer D **Yes**, Finish. "2 of 2 sections complete".
4. **Review** Breathing; change B to **No**. **Expected:** A and C disappear at once; the status beside "All sections" goes "Saving…" then **Saved**.
5. **All sections**. **Expected:** Pillows is no longer listed; "1 of 1 sections complete".
6. **Reload** the page. **Review** Breathing. **Expected:** B shows No; A and C are not shown.
7. Change B to **Yes**. **Expected:** A appears **empty** (neither Yes nor No ticked). Answer A **Yes**: C appears **empty**. Finish, then go back to the list: **Pillows** is listed again and its **Start** opens D **empty** (not the old Yes).
8. Put B back to **No**, finish, and **Send my answers**. **Expected:** "Thank you, Sam". Nothing is wider than the screen at any step.

**T-02. Leaving straight after a change still clears and saves.**
1. Another episode on MT S44c chain v1. As the patient: B Yes, A Yes, C "x", Finish; Pillows D Yes, Finish.
2. Review Breathing, change B to **No**, and **immediately** press **All sections** (don't wait for "Saved").
3. Reload. **Expected:** Pillows isn't listed; Review Breathing shows only B (No). Changing B to Yes shows A empty; Pillows (after A Yes) is empty.

**T-03. Staff's "Fill in for the patient" clears the same way.**
1. A new episode on MT S44c chain v1. As Cara, on the episode page use **Fill in for the patient** (the on-behalf form).
2. Do T-01's steps 3–7 there (desktop size is fine). **Expected:** the same: A, C and Pillows' answer go at once when B is No, and come back empty.
3. Press **← Episode**. **Expected:** you're back on the episode page; nothing is lost or refused.

### 2. A held auto-triage message stays held after Undo or un-archive

Setup for T-04 to T-06:
- As **Ada Admin**, **Settings → Settings for: Hospital A1**: for **Text the patient when the episode is Ready for admission**, untick "Use the trust's setting" and switch it **on**; **Save**.
- As Alex, a new HQ **"MT S44c held"** with one patient set (one Yes/No question with no disclosures). On its **Rules** page, tick **"When a rule auto-triages a patient, wait for a clinician to click Checked before the patient's status message goes"**. Publish.
- For each case, as Cara create an episode on MT S44c held for a new patient **with a mobile** (e.g. 07700 900611, a new one each time), and send the HQ as the patient. **Expected after each send:** the episode is **Ready for admission**, auto-triaged; its card says "Not yet checked by a clinician. The patient's message goes when you click Checked."; /qa/messages has only the invite for that mobile.

**T-04. On hold, then Undo, then Checked: the message goes once.**
1. As Cara, set the status to **On hold**. **Expected:** the auto-triage card no longer says the message goes when you click Checked (it doesn't fit On hold).
2. **Undo** (with a reason). **Expected:** back at Ready for admission; no new text; the card again says "The patient's message goes when you click Checked."
3. Click **Checked**. **Expected:** /qa/messages has **one** Ready for admission text for that mobile; General notes "Ready for admission message sent by text message.".

**T-05. Archived, then un-archived, then Checked: the message goes once.**
1. As **Alex**, **Archive** the episode. **Expected:** no text; the card doesn't say the message waits for Checked.
2. **Un-archive** (with a reason). **Expected:** back at Ready for admission; the card says "The patient's message goes when you click Checked."
3. As Cara, **Checked**. **Expected:** one Ready for admission text.

**T-06. On hold, then Ready for admission by hand, then Checked: still once (S44a's fix).**
1. Set **On hold**, then set **Ready for admission** by hand. **Expected:** the Ready for admission text arrives now.
2. Click **Checked**. **Expected:** no second text (still exactly one).

**Clean-up:** as Ada, Settings → Hospital A1: tick "Use the trust's setting" again; Save.

### 3. A High STOP-Bang holds back the default Low risk auto-triage

**T-07. A new HQ with STOP-Bang: the rule in words.**
1. As Alex, new HQ **"MT S44c stopbang"**: one patient set **Health** with a **BMI** question, and in the same set **Ready-made score → STOP-Bang** (BMI from that question). Don't change the rules.
2. Open **Rules**. **Expected:** the Low risk rule reads "When the suggested ASA is I or II and there are no red flags **and STOP-Bang score isn't High risk**: auto-triage to Ready for admission".
3. **Edit** Low risk. **Expected:** "and a score isn't in a band" is ticked, with **STOP-Bang score** and **High risk**. Cancel.
4. Open **Publish** (don't publish yet). **Expected:** the line about auto-triage says "ASA I–II, no red flags, STOP-Bang score not High risk → Ready for admission".

**T-08. The Preview and Test the whole HQ: High → no auto-triage; Low → auto-triage.**
1. On the Rules page, **Test the whole HQ**, and also the set's **Preview** in the **Clinician view** (the only one of the two that shows the score's value). Sample patient male, age 60. BMI: height 170, weight 116 (BMI over 35). Answer every STOP-Bang question **Yes**.
2. **Expected:** STOP-Bang shows **8 (High risk)** (Preview); **no** rule fires (no "Low risk") in either.
3. Now a female patient age 30, BMI 22 (170 cm, 64 kg), every STOP-Bang question **No**. **Expected:** STOP-Bang **0 (Low risk)**; "Low risk" fires: auto-triage to Ready for admission.

**T-09. A real send with a High STOP-Bang stays at Ready for review.**
1. Publish MT S44c stopbang. As Cara, an episode for a new **male** patient (DOB 17/05/1980), sent by the patient with a BMI over 35 and every STOP-Bang question Yes.
2. **Expected:** the episode is **Ready for review** (not auto-triaged).
3. Another episode, a **female** patient, BMI 22, every STOP-Bang answer No. **Expected:** auto-triaged to **Ready for admission** by Low risk.

**T-10. An HQ without STOP-Bang still auto-triages on Low risk.**
1. As Alex, a new HQ with one patient set and one Yes/No question (no disclosures). Open **Rules**. **Expected:** Low risk reads "…and STOP-Bang score isn't High risk (not in this HQ): auto-triage to Ready for admission". Editing it shows the score as "STOP-Bang score (not in this HQ)".
2. Publish. **Expected:** no publish problem about the rule; the Publish line reads "ASA I–II, no red flags → Ready for admission".
3. An episode, sent by the patient. **Expected:** auto-triaged to Ready for admission.
4. On a draft (a new version, or another new HQ), edit Low risk: untick "and a score isn't in a band", Save rule. **Expected:** saved; the words no longer mention STOP-Bang. Tick it again, choose a score and no band, Save rule. **Expected:** refused with "…choose the score and at least one of its bands." in the form.

### 4. STOP-Bang is Incomplete for sex "Other"

**T-11. Sex Other → Incomplete; the help text.**
1. On MT S44c stopbang (or a draft copy), the set's **Preview**, **Clinician view**: sample patient sex **Other** (the Sex list offers Female, Male, Other and Not known), age 40, BMI 25, answer every STOP-Bang question. **Expected:** STOP-Bang shows **Incomplete** (no band). Change sex to **Female**: a number and a band. **Not known**: Incomplete again.
   Then **Test the whole HQ** with the same answers and sex **Other**. **Expected:** "Low risk" fires held: "no auto-triage: a score is Incomplete".
2. In the editor, select the STOP-Bang score card and open its settings. **Expected:** under the score's help text: "A patient whose sex is Not known or Other makes STOP-Bang Incomplete."
3. A real episode for a patient whose sex is **Other**, sent with every STOP-Bang question answered. **Expected:** it is **not** auto-triaged (Ready for review: a score is Incomplete). On the POA Summary or Validate, the STOP-Bang score reads Incomplete.

### 5. No "Patient answered" line on a question the patient never saw

**T-12. Validate and the POA Summary.**
1. An episode on MT S44c chain v1; the patient answers B **No** and sends (Pillows never shown).
2. As Cara, on the episode page **Validate** Breathing. Change B to **Yes**. **Expected:** under B: "Patient answered: No". A appears: answer **Yes**; **no** "Patient answered" line under A. C appears: type "Coughing"; no line under C. **Validate**.
3. **Expected:** back on the episode page, **Pillows** is now listed (Not started), and **Complete HQ review** is disabled ("Validate every Question Set…").
4. Validate **Pillows**: answer D **Yes**. **Expected:** no "Patient answered" line at all. Validate. Complete HQ review is now enabled.
5. Re-open Breathing (**Review**). **Expected:** exactly one "Patient answered" line (under B).
6. **POA Summary**, Validated summary tab. **Expected:** "Changed by the clinician on Breathing" lists only "Do you get short of breath? Patient answered “No”, now “Yes”" (nothing for A or C); there's no "Changed by the clinician" box for Pillows.
7. Compare: a question the patient **saw but skipped** still gets a line. (If your HQ has an optional question that was shown to the patient, answering it on Validate shows "Patient answered: no answer". Optional: skip if there's none.)

### 6. Small fixes

**T-13. Renaming or deleting a Question Set another author just deleted.**
1. As Alex, a draft HQ with three Question Sets. Open the editor in **two tabs**.
2. In tab 1, delete the third set. In tab 2 (not reloaded), rename that same set (its header's settings) and save. **Expected:** refused with "**This Question Set has been deleted.**".
3. In tab 2, try to delete it. **Expected:** the same message.
4. Publish the HQ in tab 1, then in tab 2 rename another set. **Expected:** "Only draft versions can be changed." (unchanged).

**T-14. Complete review on an episode no longer waiting for review.**
1. An episode on MT S44c chain v1, sent by the patient (B No). As Cara, Validate Breathing as is, so **Complete HQ review** is enabled. Keep this tab open.
2. *(Integration test only: `TestCompleteReview`.)* An episode that moved on before anyone reviewed it (On hold, Not ready, or auto-triaged straight past Ready for review) is refused with "**This episode isn't waiting for review any more.**" The UI can't reach this except through a race (the status is locked at Ready for review); to check the wording, an auto-triaged episode's `POST …/complete-review` answers 409 with it.
3. In a second tab, same episode as step 1: **Complete HQ review**, then set **On hold**. Back in the first tab (not reloaded), click **Complete HQ review**. **Expected:** "The HQ review is already complete."

**T-15. A clinician's save after the review was completed elsewhere.**
1. An episode at Ready for review, every set validated. Open a set's **Validate** in tab 1.
2. In tab 2, **Complete HQ review**.
3. In tab 1, change an answer. **Expected:** "The HQ review is already complete, so the answers can't be changed." (not "changed somewhere else").

(The fourth small fix, Publish's "last changed by" never overwritten by a save that waited for it, can't be timed by hand: an integration test covers it.)

## Regression

- **R-01 (S44a):** a send keeps only what the form showed: on MT S44c chain, a patient who answered B Yes, A Yes and then B No sends; Validate Breathing shows only B.
- **R-02 (S44b):** on a sent episode's POA Summary, **Add grade** (anaesthetist's ASA III), then a second grade with a reason: the history shows both; the latest **Counts**.
- **R-03 (S42):** a low-risk send on an HQ with the default rules and no "wait for Checked" (Hospital A1's Ready for admission text switched on, as in section 2): the text goes at once, before Checked.
- **R-04 (S43):** an HQ whose answer gives ASA III (a disclosure with grade III): on send, the High risk rule adds the task "Book anaesthetic review" and the flag "Anaesthetic review".
- **R-05 (S25):** a patient whose earlier HQ answered a question gets it pre-filled in a new HQ with the same question; the card and "Check each answer" work as before.

## Out of scope

- Softer wording of the auto-triage status message (before go-live, with the trust).
- The evaluate Lambda (S45).
- HQs created before this session keep their old rules (no STOP-Bang test in their Low risk rule): only new HQs, and new versions of HQs without rules, get the new defaults.
- A pre-filled answer to a question that is hidden when the form opens stays until the patient finishes that section (and a send drops it anyway): not a failure.
- Checked clicked while the episode is On hold ends the held message (no text then, nor after an Undo): the clinician has checked it; not a failure.
