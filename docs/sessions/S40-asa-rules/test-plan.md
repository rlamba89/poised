# S40 test plan: ASA rules on the HQ

*Changed after test report 1:* the Preview's Rules that fire now uses the clinician's view in both views (T-08 step 5 in the Patient view); "at least I" and "at most III–VI" read "I to VI" / "I to III…" (T-03); saving a test case under an existing name replaces it, test case names are unique, Enter saves, and a test case's bin asks first (new T-09b).

For a tester who has never seen the code. Use only the browser.

## What this session built

Each HQ (questionnaire) version now has **ASA rules**. A rule has a name, a **when** (the suggested ASA compared with a grade, plus optional extras) and a **then** (actions from a fixed list). **Nothing acts on the rules yet**: later sessions auto-triage, add tasks and flags when a patient sends the HQ. This session lets authors write, see, preview and test them.

1. **Default rules.** A new HQ (and a new version of an HQ that had none) starts with two:
   - **Low risk:** "When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission";
   - **High risk:** "When the suggested ASA is III or more: add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission".
2. **The Rules page:** a **Rules** button in the editor's header. The rules in plain words, with **Add rule**, Edit (pencil), Delete (bin), and ↑ ↓ to reorder. Every change saves at once. A setting: **"Patients whose answers give no ASA grade (usually fit and well) count as ASA I, so the rules can auto-triage them"** (on by default; off, such a patient is never auto-triaged, but a matching rule's other actions still apply).
   - **When:** the suggested ASA *is at most / is at least / is exactly* I–VI; optional ticks: *and there are no red flags*, *and a score is in a band*, *and a disclosure has the code*, *and the procedure is one of*.
   - **Then:** *Auto-triage* (to Ready for admission, Ready for POA or POA complete), *Add a task* (title and team), *Add a flag* (a name), *Require the anaesthetist's ASA*, *Suggest a status*.
   - Read-only for those who can't change the HQ, and on a published version.
3. **Whole-HQ test cases:** **Test the whole HQ** (on the Rules page) shows the patient's Question Sets as tabs; answer them as a made-up patient and see **What the rules see** (the suggested ASA and the rules that fire), then **Save as test case**. The Rules page lists each case with pass/fail, re-checked after every change; a failing one blocks Publish.
4. **Preview:** under Suggested ASA, **Rules that fire** for that one Question Set's answers.
5. **Publish window:** one line saying what the version auto-triages, e.g. "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I."

Rules look at what the patient sends: the patient's answers (a clinician Question Set's own questions never count).

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

- **Alex Author**: super clinician, the whole of Trust A (writes HQs).
- **Sue Super**: super clinician, Hospital A1 only (views the trust's HQs, can't change them).
- **Cara Clinician**: clinician, Hospital A1.
- **Bea Author**: super clinician, Trust B.

### Codes to use (from the seeded code list)

| Code set | Search for | What it gives |
| --- | --- | --- |
| SNOMED | `77176002` (Smoker) | fills in ASA II ("Current smoker") by itself |
| ICD10 | `I50.9` (Heart failure unspecified) | offers ASA III or IV: choose **III** |
| SNOMED | `49727002` (Cough) | no ASA grade |

### How to add a disclosure

In the editor, click a question's card; in the right-hand panel open the **Disclosures** tab; click **Add** next to an option; **+ Add disclosure**; add a code (the picker has a SNOMED / ICD10 switch) and/or a note with a category; **Review flag** sets amber or red; **Save output**, then **Done**.

### The HQ to build first (T-00)

As **Alex**, Questionnaires → **New**: name **"MT S40 <time>"** (e.g. "MT S40 1415"), any description. Then:

1. Question Set **"About you"** (patient):
   - Select One **"Do you smoke?"**, Yes / No. Disclosure on **Yes**: code SNOMED 77176002 (ASA II fills in).
   - Select One **"Do you get chest pain when you walk?"**, Yes / No. Disclosure on **Yes**: a note "Chest pain on exertion", category e.g. Cardiac, **Review flag: red**. No code.
   - Select One **"How often do you snore?"**, options **Never** (score 0), **Sometimes** (score 1), **Every night** (score 3).
   - A **Calculation** "Snore score" adding up "How often do you snore?", with **bands**: **Low** up to **1**, **High** above. (In its Settings, **Add band** twice.)
2. Question Set **"Your heart"** (patient): Select One **"Have you got heart failure?"**, Yes / No. Disclosure on **Yes**: ICD10 I50.9, choose **ASA III**.
3. Question Set **"Nurse checks"** (Who for: **Clinician**): Select One **"Frail?"**, Yes / No. Disclosure on **Yes**: a note "Frail" (category General), ASA grade **IV** typed by hand.

Wait until the header says "All changes saved".

## Test cases

### The Rules page

**T-01 A new HQ shows the two default rules, in plain words**
1. In MT S40's editor, click **Rules** in the header.
2. *Expected:* the address ends `/questionnaires/<id>/rules`; title "← MT S40 … / Rules" and "V1 · DRAFT". The setting is ticked. Two rules, numbered: **1. Low risk** "When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission" and **2. High risk** with the wording in "What this session built". "Test cases for the whole HQ: No test cases yet."
3. Reload the page. *Expected:* the same. Click **← MT S40 …**: back in the editor. Browser Back: the Rules page again.

**T-02 Edit a rule: the target, kept after a reload**
1. On Low risk click the pencil (Edit). *Expected:* a form with Name "Low risk", "the suggested ASA [is at most] [II]", "and there are no red flags" ticked, and one action: **Auto-triage** → **Ready for admission**.
2. Change the target to **Ready for POA**, **Save rule**. *Expected:* Low risk now ends "auto-triage to Ready for POA". Reload: still so.
3. Edit again, set it back to **Ready for admission**, Save. *Expected:* the default wording again.
4. Edit, change something, then **Cancel**. *Expected:* nothing changed.

**T-03 Add a rule with each kind of "when"**
1. **Add rule**. Name **"Snorer"**; the suggested ASA **is at least I**; tick **and a score is in a band**: score **Snore score**, bands **High**; **+ Add action** → **Add a flag**: "Sleep apnoea". Save rule.
   *Expected:* "3. Snorer: When the suggested ASA is I to VI and Snore score is High: flag “Sleep apnoea”".
2. **Add rule** "Heart code": **is exactly III**; tick **and a disclosure has the code** → **Heart failure unspecified (I50.9)** (the list offers only codes on this HQ's disclosures: Smoker and Heart failure); action **Suggest a status** → **On hold**. *Expected:* "When the suggested ASA is III and a disclosure has the code Heart failure unspecified (I50.9): suggest On hold".
3. **Add rule** "Hip": **is at most VI**; tick **and the procedure is one of** → choose **Total hip replacement** and one more; action **Require the anaesthetist's ASA**. *Expected:* "When the suggested ASA is I to VI and the procedure is Total hip replacement or <the other>: require the anaesthetist's ASA before Ready for admission".
4. **Add rule** "Red": **is at least I**, tick **and there are no red flags**; actions **Add a task** "Phone the patient", team **Nurse**, and **Auto-triage** → **POA complete**. *Expected:* "…and there are no red flags: add the task “Phone the patient” for the nurse team; auto-triage to POA complete".
5. Reload. *Expected:* all six rules, in that order, same words.

**T-04 Each kind of action, and a refused rule in plain words**
1. **Add rule** "Bad", leave the actions empty, **Save rule**. *Expected:* a red message in the form: "Rule “Bad”: add at least one action." The form stays open; the list is unchanged.
2. **+ Add action** (it's **Add a task**); leave the task box empty; Save. *Expected:* "Rule “Bad”: Enter the task."
3. Change it to **Add a flag**, leave the name empty; Save. *Expected:* "Rule “Bad”: name the flag, on one line (100 characters at most)."
4. Clear the Name; give the flag a name; Save. *Expected:* "Give each rule a name."
5. Name it **"low risk"** (another rule's name, other case); Save. *Expected:* "Two rules are called “low risk”: give each its own name."
6. Name it **"Bad"** again; make two actions both **Auto-triage**; Save. *Expected:* "Rule “Bad”: auto-triage only once."
7. **Cancel**. *Expected:* no "Bad" rule was added.

**T-05 Reorder and delete**
1. On rule 3 press ↑. *Expected:* it becomes 2; reload keeps the order. The first rule's ↑ and the last rule's ↓ are disabled.
2. Press the bin on "Hip". *Expected:* "Delete “Hip”? Delete / Keep it". **Keep it**: nothing changes. Bin again → **Delete**: it's gone, and stays gone after reload.

**T-06 The no-grade setting**
1. Untick the setting. *Expected:* a line under it: "Off: they're never auto-triaged; a clinician reviews them." Reload: still off.
2. Tick it again (on, for the next cases).

**T-07 Who can change rules**
1. Sign in as **Sue Super** (Hospital A1), Questionnaires → MT S40 → **Rules**. *Expected:* the rules in words, a **Read only** badge and "Read only: only super clinicians and admins of the whole trust can change the trust's rules."; no Add rule, no pencils, bins or arrows; the setting's tick is greyed out. **Test the whole HQ** opens, but says "Read only…" instead of Save as test case.
2. Same as **Cara Clinician**: the editor shows a **Rules** button; the Rules page is read-only the same way.
3. As **Bea Author** (Trust B), paste MT S40's Rules address (from Alex's browser, `/o/<Trust A>/h/<A1>/questionnaires/<id>/rules`). *Expected:* Bea isn't let in to Trust A's hospital (as for any Trust A page). Change only the trust and hospital part of the address to Bea's own Hospital B1 (keep `/questionnaires/<id>/rules`). *Expected:* "Questionnaire not found." Nothing of Trust A's rules shows.

### The Preview

**T-08 Rules that fire for one Question Set's answers**
1. As Alex, open **About you** → **Preview** (Patient view).
2. Answer nothing. *Expected:* under the outputs, **Rules that fire** "For this Question Set's answers alone…" lists **Low risk: auto-triage to Ready for admission** and, if T-03's "Red" exists, "Red: …" (no ASA grade counts as ASA I). "Snorer" isn't listed.
3. Answer **Do you smoke? Yes**. *Expected:* Suggested ASA II (Smoker…); Rules that fire still includes **Low risk**.
4. Answer **chest pain: Yes** (a red flag). *Expected:* **Low risk** and "Red" are gone from Rules that fire (only rules without "no red flags" stay, e.g. none or Snorer).
5. Answer snoring **Every night**. *Expected:* **Snorer: flag “Sleep apnoea”** fires, in the Patient view too, even when Snore score is a Clinical (clinician-only) Calculation and so isn't listed under Scores there; in the Clinician view Snore score shows "3 (High)".
6. Open **Your heart** → Preview, answer **heart failure: Yes**. *Expected:* Suggested ASA III; **High risk** fires with its three actions; "Heart code: suggest On hold" fires too.
7. "Hip" (procedure) never fires in the Preview (no procedure there).

### Whole-HQ test cases

**T-09 Test the whole HQ, and save a test case**
1. Rules page → **Test the whole HQ**. *Expected:* tabs **About you** and **Your heart** only (not Nurse checks); a sample patient (Sam Sample, 54, Female), a Procedure box ("None"); **What the rules see**: "No suggested ASA", and Rules that fire: Low risk (and Red, if present).
2. In About you answer **smoke: Yes**, **chest pain: No**, **snore: Never**. *Expected:* "Suggested ASA II" with "Smoker (77176002) – Current smoker"; **Low risk: auto-triage to Ready for admission**.
3. Tab **Your heart** → heart failure **No**. Back to About you: the answers are still there.
4. Name it **"Smoker"**, **Save as test case**. *Expected:* "“Smoker” passes"; the address gets `?case=…`.
5. **← Rules**. *Expected:* under Test cases for the whole HQ: "All 1 passed"; **Smoker** with a green **pass** and its lines, e.g. "Rule fires: Low risk (auto-triage to Ready for admission)" and "Suggested ASA II" (plus any other rule that fires, e.g. Red).
6. Make a second case: Test the whole HQ → **heart failure: Yes** → save **"Heart"**. *Expected:* Suggested ASA III; High risk fires (and Heart code). It passes on the Rules page.
7. Reload the Rules page. *Expected:* both cases, both pass.

**T-09b Saving a case again replaces it; Enter saves; the bin asks first**
1. On the Rules page, **Open** "Heart". *Expected:* the name box says "Heart" and the button reads **Replace test case**, with "A test case has this name: saving replaces it…".
2. Change an answer (heart failure **No**), click in the name box and press **Enter**. *Expected:* it saves ("“Heart” passes"). **← Rules**: still one "Heart" (not two), with the new lines (Suggested ASA … without High risk).
3. On the test page type a new name and press Enter: a new case is added.
4. On the Rules page press a test case's bin. *Expected:* "Delete “…”? Delete / Keep it". Keep it: nothing changes; Delete: it's gone.

**T-10 Open a saved case**
1. On "Smoker" click **Open**. *Expected:* the test page with the same sample patient and answers (smoke Yes), "“Smoker” passes".
2. **Clear answers**. *Expected:* the answers go, the address loses `?case=`, and the result goes back to "No suggested ASA".

**T-11 A changed rule makes a case fail, and blocks Publish**
1. Edit **Low risk**: **is at most I**. Save. *Expected:* Low risk reads "When the suggested ASA is I and there are no red flags…"; **Smoker** turns red **fail**, "1 of 2 failed". Click its name: "− expected, not produced: Rule fires: Low risk (auto-triage to Ready for admission)".
2. Back in the editor, **Publish**. *Expected:* the window lists "**Rules** · Test case “Smoker” fails"; Publish is disabled.
3. Set Low risk back to **is at most II**. *Expected:* Smoker passes again; the Publish window has no problem.
4. Change Low risk's target to **Ready for POA**. *Expected:* Smoker fails (its line names Ready for admission). Set it back.
5. Delete the "Smoker" case with its bin (as Alex). *Expected:* it's gone after reload. (Make it again as in T-09 if you need it later.)

**T-12 A rule that can't fire blocks Publish**
1. In the editor, delete the **"Snore score"** Calculation (or remove its **High** band).
2. Open Rules. *Expected:* an orange box "These rules can't fire, and block Publish": "Rule “Snorer”: its score isn't in this HQ any more" (or "a band it tests isn't in “Snore score” any more"). Snorer's words show the score's ID instead of its name.
3. Publish window: *Expected:* the same problem under **Rules**; Publish disabled.
4. Delete the "Snorer" rule. *Expected:* the warning is gone. (Similarly, removing the I50.9 code from its disclosure makes "Heart code" say "no disclosure in this HQ has the code I50.9, so the rule can't fire".) Leave the HQ with no rule problems.

**T-13 The no-grade setting in the test page**
1. Untick the setting on the Rules page.
2. Test the whole HQ: answer **smoke: No**, **chest pain: No**, heart failure **No**. *Expected:* "No suggested ASA"; Rules that fire: **Low risk: no auto-triage: no ASA grade** (and "Red: add the task “Phone the patient” for the nurse team; no auto-triage: no ASA grade" if it exists).
3. Tick the setting again: the same answers give "Low risk: auto-triage to Ready for admission".

**T-14 Clinician sets don't count**
1. On the test page there's no Nurse checks tab. (A nurse's ASA IV on "Frail?" can never count, since the patient doesn't answer it.)

### Publishing

**T-15 The Publish window says what the version auto-triages**
1. With the default rules (Low risk to Ready for admission) and the setting on, open **Publish**. *Expected:* below the usual text: "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I." (If "Red" still exists, its part is added after a "; ", e.g. "ASA I–VI, no red flags → POA complete". Delete "Red" first for a clean line.)
2. Untick the setting, reopen Publish. *Expected:* "… Patients with no ASA grade are never auto-triaged." Tick it again.
3. Delete Low risk's auto-triage action temporarily (edit it, bin the action, add a flag instead, save). *Expected:* "This version doesn't auto-triage anyone." Put Low risk back as it was.

**T-16 After publishing: read-only; a new version copies the rules**
1. Make sure the test cases pass and there are no problems; **Publish**.
2. Open **Rules**. *Expected:* "V1 · PUBLISHED", **Read only**, "Read only: this version is published. To change its rules, create a new version."; no buttons; the test cases still listed; Test the whole HQ works but can't save.
3. Back in the editor, **Create new version**, then **Rules**. *Expected:* "V2 · DRAFT", the same rules (including any you added, in the same order), the same setting and the same test cases, all passing, and editable.

**T-17 A seeded HQ has the default rules**
1. Open **Demo pre-op HQ** (published, seeded) → **Rules**. *Expected:* the two default rules, read-only.

### Edge cases

**T-18 Two tabs: a stale save is refused**
1. Open the v2 Rules page in two tabs.
2. In tab 1, tick or untick the setting. In tab 2 (not reloaded), edit Low risk's target and Save.
3. *Expected:* tab 2 shows "Someone else changed these rules since you opened them. Reload to see their changes; your latest change was not saved." After reloading tab 2, it shows tab 1's change and the rule unchanged.

**T-19 Deep links**
1. Paste `…/questionnaires/<id>/rules/test` into a new tab. *Expected:* the test page opens. `…/rules/test?case=<an id that doesn't exist>`: the page opens with no case loaded (no error).

## Regression

**R-01 Per-set test cases** (S08, S38): in any draft's Preview, answer, **Save answers** as a test case, **Run all**: it passes; still saved per Question Set (unchanged by S40).
**R-02 Publish** an HQ with no failing cases (any draft): publishes as before; **Create new version** works.
**R-03 The suggested ASA** (S39): in the Preview, answers giving ASA II and III show "Suggested ASA III" with its reasons; a clinician Question Set's condition can still offer "Suggested ASA is at least…".
**R-04 Editor basics** (S04): add a question, rename it, undo; "All changes saved".
**R-05 A patient sends an HQ** (S14): Cara creates an episode on a published HQ, the patient (phone size, 390×844) fills it in from the captured invite (http://localhost:3000/qa/messages) and sends; the episode becomes **Ready for review** (rules don't act yet).

## Out of scope (don't report)

- Nothing happens when a patient sends: no auto-triage, no task, no flag, no requirement (S42, S43).
- The server doesn't re-check rules or their test cases at publish (S41); only the browser blocks Publish.
- Rules on the nurse's ASA; trust-wide rules shared by several HQs; a screen for the platform defaults.
- The Rules and test pages aren't patient pages, so phone size isn't required (they should still be usable at 390px wide).
- A rule's procedure list shows only this hospital's procedures.
