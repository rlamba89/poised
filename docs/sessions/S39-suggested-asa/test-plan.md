# S39 test plan: the ASA guide and the suggested ASA

*Changed after verification 1:* new T-03b and T-05c. *Changed after test report 1:* T-05 step 2 (a disclosure edited later now offers its existing codes' examples too), and new T-05b (removing a code) and T-06 step 3 (an option made from a code can be given its grade from the guide).

For a tester who has never seen the code. Use only the browser.

## What this session built

1. **The ASA guide.** When an author adds a code to a disclosure, the editor offers the ASA's own published examples for that code (ASA statement, 2020): one example fills in the ASA grade, with the example's words as its reason; several (the grade depends on how severe or well controlled it is, e.g. diabetes II or III) must be chosen from before the disclosure can be saved. The author can always change it.
2. **The suggested ASA for the whole HQ**, with its reasons: the highest ASA grade across every shown Question Set's disclosures, and each disclosure at that grade in words. It shows on the **POA Summary** (both tabs, in the ASA grades section, above the nurse's and anaesthetist's grades) and as **one line at the top of the episode page's Health questionnaire card** (where the sets are validated).
3. **The preview** keeps its one-set suggested ASA and now lists the reasons.
4. **A clinician Question Set's condition can test the suggested ASA** (e.g. show "Anaesthetic plan" only when it's III or more). Patient sets can't.

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

- **Alex Author**: super clinician, the whole of Trust A (writes HQs).
- **Cara Clinician**: clinician, Hospital A1 (episodes, Validate, POA Summary).
- **Sue Super**: super clinician, Hospital A1 only (views HQs, can't change them).
- **Bea Author**: super clinician, Trust B.

Captured patient emails and texts are at **http://localhost:3000/qa/messages** (the invite link is in the email).

### Codes to use (from the seeded code list)

| Code set | Search for | Code | What the ASA guide offers |
| --- | --- | --- | --- |
| SNOMED | `13645005` (Chronic obstructive lung disease) | 13645005 | one example: ASA III, "COPD" |
| SNOMED | `77176002` (Smoker) | 77176002 | one example: ASA II, "Current smoker" |
| ICD10 | `E11.9` (Type 2 diabetes mellitus without complications) | E11.9 | ASA II "Well-controlled diabetes" (Adults), ASA III "Poorly controlled diabetes" (Adults), ASA II "Non-insulin-dependent diabetes" (Children) |
| ICD10 | `I50.9` (Heart failure unspecified) | I50.9 | ASA III "Moderate reduction of ejection fraction", ASA IV "Severe reduction of ejection fraction" (Adults), ASA IV "Congestive heart failure" (Children) |
| SNOMED | `49727002` (Cough) | 49727002 | none |

The code picker has a SNOMED / ICD10 switch above the search box.

### How to add a disclosure (for the tester)

In the editor, click a question's card; in the right-hand panel open the **Disclosures** tab; click **Add** (or the 🔗 number) next to an option; in the window click **+ Add disclosure**; add codes and/or a note, then **Save output**, then **Done**.

## Test cases

### The ASA guide in the editor

**T-01 One example fills in the grade, with its reason.**
1. As Alex, create a new HQ (Questionnaires → New), add a Question Set "About you" with a **Select One** question "Do you have lung disease?" with options Yes and No.
2. Add a disclosure to **Yes**. In the code picker choose **SNOMED**, search `13645005`.
3. *Expected:* the result row says "· ASA III" after the category.
4. Click the code. *Expected:* **ASA grade** shows **III** by itself; a box "ASA from the ASA guide" lists "ASA III: COPD (Adults)" (selected) and "No ASA grade"; under it: "Shown with the suggested ASA as: “COPD” (from the ASA guide)…".
5. **Save output**. *Expected:* the list shows the code, the grey badge **ASA III** and "(COPD)".
6. **Done**, reload the page, reopen the disclosures for Yes. *Expected:* still ASA III (COPD).

**T-02 Several examples: nothing filled in; the author must choose.**
1. On the **No** option (or another question), **+ Add disclosure**, switch to **ICD10**, search `E11.9`, click it.
2. *Expected:* **ASA grade** stays empty. The "ASA from the ASA guide" box (pale yellow) lists the three diabetes examples (two Adults, one Children) and "No ASA grade", and says the grade depends on how severe or well controlled it is.
3. Click **Save output** without choosing. *Expected:* a red message "Choose the ASA grade from the ASA guide, or No ASA grade." The disclosure isn't saved.
4. Choose "ASA III: Poorly controlled diabetes (Adults)". *Expected:* ASA grade shows III; the reason line says “Poorly controlled diabetes”; the red message goes.
5. **Save output**. *Expected:* the list shows ASA III and "(Poorly controlled diabetes)".

**T-03 "No ASA grade" is a choice.**
1. New disclosure, ICD10 `E11.9` again, choose **No ASA grade**, then **Save output**.
2. *Expected:* it saves with no ASA badge.

**T-03b (verification 1) "No ASA grade" stands when another code is added.** New disclosure: SNOMED `13645005` (fills III), choose **No ASA grade**, then add SNOMED `49727002` (Cough). *Expected:* the grade stays empty and No ASA grade stays selected. Add ICD10 `E11.9` too: its examples join the box, nothing is filled in, and Save isn't refused (No ASA grade still chosen).

**T-05c (verification 1) The Emergency tick survives removing a code.** New disclosure: COPD (III), tick **Emergency (E)**, add Smoker, remove COPD. *Expected:* **II** with the Emergency tick still on, reason “Current smoker”.

**T-04 Changing the grade by hand.**
1. New disclosure, SNOMED `13645005` (fills III, reason COPD).
2. Change **ASA grade** to **IV** in its list. *Expected:* the reason line disappears (IV is the author's own grade, not the guide's).
3. Change it back to **III**. *Expected:* the reason “COPD” comes back.
4. Clear the grade (the ✕ in the box). *Expected:* no grade, no reason. Save: no ASA badge.
5. With ICD10 `I50.9` added to a new disclosure, pick **IV** by hand from the grade list. *Expected:* no reason (two examples are at IV: the Adults' and the Children's), saved as ASA IV.

**T-05 A code with no example; a disclosure that already has a grade.**
1. New disclosure, SNOMED `49727002` (Cough). *Expected:* no "· ASA" hint on the result row, no guide box, ASA grade empty.
2. Edit a saved disclosure that already has ASA III (T-01's COPD). *Expected:* the guide box lists "ASA III: COPD (Adults)", selected. Add SNOMED `77176002` (Smoker). *Expected:* the grade stays **III** (not changed to II); the box now also lists "ASA II: Current smoker (Adults)". Choosing the Smoker option changes the grade to II with its reason.
3. Cancel.

**T-05b Removing a code takes away the grade that came from it.**
1. New disclosure: add SNOMED `13645005` (III, COPD), then SNOMED `77176002` (Smoker): the grade stays III.
2. Remove the COPD code with its ✕. *Expected:* the grade becomes **II** with the reason “Current smoker” (Smoker's only example).
3. Instead remove Smoker (with COPD's grade III kept): *expected:* III (COPD) stays.
4. A disclosure with COPD (III) and ICD10 `E11.9`: remove COPD. *Expected:* no grade, and Save asks you to choose from the diabetes examples.
5. A grade typed by hand (e.g. V) stays when a code is removed.

**T-06 Adding options from the code library.**
1. On a Select Many question's Settings, click **From the code library** (above the options). Add SNOMED `13645005` and ICD10 `E11.9`, then **Add 2 options**.
2. *Expected:* the window's text says a code with one example gets its ASA grade and that a severity-dependent one is chosen in the option's disclosure. The COPD option's disclosure has **ASA III (COPD)**; the diabetes option's has **no** grade.
3. Edit the diabetes option's disclosure. *Expected:* the "ASA from the ASA guide" box lists the diabetes examples (none selected; Save isn't refused). Choose "ASA III: Poorly controlled diabetes (Adults)": III with that reason; save. Picking **III** from the grade list instead also gives that reason.

**T-07 Read-only.** As **Sue Super**, open the same draft and a question's disclosures. *Expected:* she sees the disclosures with their ASA badge and reason, but no Add, Edit or Delete (as before S39).

### The preview

**T-08 The preview's suggested ASA with reasons.**
1. Open the Question Set's **Preview**, switch to Clinician, answer Yes to the lung question.
2. *Expected:* the Clinical outputs panel shows **Suggested ASA** "ASA III", and under it "Chronic obstructive lung disease (13645005) – COPD".
3. If a test case was saved before with "Suggested ASA III", it still passes.

### A clinician set's condition on the suggested ASA

**T-09 Offered on a clinician set only, and described in words.**
1. In the same HQ add a second Question Set "Anaesthetic plan"; set **Who is it for?** to **Clinician**; add a Long text question "Plan".
2. On its Question Set page, **Display this Question Set → Conditionally**, open the **Question** list. *Expected:* under "Patient and viewer" there's **Suggested ASA**. Choose it; the comparison offers "is at least", "is at most", "is"; the value offers I–VI. Choose **is at least III**.
3. *Expected:* the note under the condition says a clinician's Question Set can be shown by the suggested ASA (none counts as lower than I). Reload: the condition is kept.
4. On a **patient** set ("About you") choose Conditionally and open the Question list. *Expected:* no "Suggested ASA".
5. Switch "Anaesthetic plan" to **Who is it for? → Patient**. *Expected:* the header's problems and the **Publish** window list "Question Set condition: only a clinician's Question Set can test the suggested ASA", and Publish is blocked; the condition still shows "Suggested ASA is at least III" (so it can be removed). Switch it back to Clinician: the problem goes.

### Clinicians: the episode page and the POA Summary

Build this HQ as Alex (or reuse T-09's), then publish it:
- **About you** (patient): "Do you smoke?" Yes/No; Yes has a disclosure with SNOMED `77176002` (ASA II, Current smoker).
- **Your heart** (patient): "Have you been told you have heart failure?" Yes/No; Yes has a disclosure with ICD10 `I50.9`, choose "ASA III: Moderate reduction of ejection fraction (Adults)".
- **Anaesthetic plan** (clinician): shown when Suggested ASA is at least III (T-09).

**T-10 Two sets, II and III: the POA Summary says ASA III with the right reason.**
1. As Cara, create an episode on this HQ for a made-up patient with a made-up email; open the invite link from /qa/messages, enter the date of birth, answer Smoke **Yes** and Heart failure **Yes**, send.
2. Open the episode page. *Expected:* at the top of the **Health questionnaire** card: "**Suggested ASA III**: Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction (the anaesthetist confirms)". The Question Sets list includes **Anaesthetic plan**.
3. Click **POA Summary**. In the **Pre operative assessment ASA grades** section: "Suggested ASA III", one reason (the heart failure line; not the smoking one), "Suggested from the answers. The nurse and the anaesthetist give the ASA grade.", then the nurse's and anaesthetist's grade boxes.
4. Click the **Patient answers** tab. *Expected:* the same section shows the same suggestion and reason, **without** the nurse's and anaesthetist's boxes.

**T-11 The reason is a link.** On the POA Summary click the reason. *Expected:* Validate opens for "Your heart" at that question, highlighted, with a "POA Summary" link; Back returns to the same tab and place.

**T-12 A clinician's change on Validate changes the validated tab only.**
1. As Cara, Validate "About you" as it is, then on "Your heart" change heart failure to **No** and Validate.
2. *Expected:* POA Summary **Validated summary**: "Suggested ASA II" with "Smoker (77176002) – Current smoker". **Patient answers**: still "Suggested ASA III" with the heart failure reason.
3. The episode page's line says "Suggested ASA II: Smoker (77176002) – Current smoker", and **Anaesthetic plan** is no longer in the list.
4. Change heart failure back to Yes on Validate: the plan set and ASA III come back.

**T-13 Hide the ASA III set by its condition: the suggestion drops to II.**
1. As Alex make a **new version** of the HQ; give **Your heart** the condition "Do you smoke? is Yes" (Display this Question Set → Conditionally). Publish.
2. New episode on it; the patient answers Smoke **No**, and (Your heart is now hidden) sends.
   *Expected:* the episode page line and the POA Summary say **No suggested ASA: no answer gives an ASA grade.** (not an error).
3. Another episode: the patient answers Smoke **Yes** and Heart failure **Yes** and sends: ASA III, as T-10. Then as Cara, on Validate, change Smoke to **No** and Validate "About you". *Expected:* "Your heart" drops out of the episode page's list, and the validated tab says **No suggested ASA** (the heart failure answer is still held, but its set is hidden, so it doesn't count); the Patient answers tab still says ASA III.

**T-14 No grade at all.** An episode whose answers give no grade (Smoke No, Heart failure No). *Expected:* episode page "No suggested ASA: no answer gives an ASA grade" (without "the anaesthetist confirms"); POA Summary, both tabs: "No suggested ASA: no answer gives an ASA grade." The nurse's and anaesthetist's grades still work on the validated tab (choose a grade, Save, reload).

**T-15 Emergency (E).** As Alex add, to a third option or question, a disclosure with a note and ASA **I** with **Emergency (E)** ticked (e.g. "Do you need surgery today?" Yes). Publish; a patient answers it Yes with Smoke Yes. *Expected:* POA Summary "**Suggested ASA IIE**", reasons: the smoking line and the emergency one ending "(emergency)". The preview with both answers shows "ASA IIE" and both lines.

**T-16 Reload and deep links.** On the POA Summary's Patient answers tab, reload: still that tab (`?tab=patient`) with its suggestion. Open the episode page's address in a new tab: the line is there.

**T-17 Another trust.** As **Bea Author** (Trust B), search a code in a disclosure (SNOMED `13645005`): the same ASA guide choice (the guide is the platform's). Trust B can't open Trust A's episode addresses (copy one from Cara's session: "not found").

## Regression

**R-01 (S38) Scores across sets.** An HQ whose second set's score uses a first set's answers still shows the right total and band on Validate and the POA Summary.
**R-02 (S27) POA Summary links.** A disclosure note on the validated tab still opens Validate at its question, and Back returns to the same place.
**R-03 (S20) ASA grades.** On the validated tab, set the Nurse ASA grade to III, Save, reload: kept; the General notes say "Nurse ASA grade set to 3".
**R-04 (PRV-05) Test cases.** A saved test case expecting "Suggested ASA III" still passes, and a failing one still blocks Publish.
**R-05 (S14/S15) The patient at phone size (390×844).** The patient's set list never shows the clinician's Anaesthetic plan, and nothing about ASA appears to the patient; no sideways scroll.

## Out of scope (don't report)

- Rules and auto-triage (S40–S43); storing the suggested ASA (S41).
- Changes to the nurse's and anaesthetist's ASA controls.
- The ASA guide's clinical content: it's made up for testing from the ASA's examples and a clinician checks it before go-live. Codes not in the table above may have no examples.
- A score's band offering ASA guide choices (bands have no codes); the author types the grade by hand there.
- The Validate page itself has no suggested ASA line (it's on the episode page and the POA Summary).
