# S43 test plan: tasks, flags and the anaesthetist's ASA from rules

*Changed after test report 1:*
- **F-1 fixed:** after **Reload** on a stale Clear, the "Why is it cleared?" form closes when its flag is no longer standing. Retest **T-07** step 3.
- **Exploring note 3 fixed:** the reason box is focused when the Clear form opens, so typing goes straight into it (see T-05).
- **T-06's expected text corrected:** an A1 URL opened by someone outside Hospital A1 shows "You don't have access to this hospital" (S08's guard); with their own hospital in the URL, "Episode not found.".

For a tester who has never seen the code. Everything is in the browser, except stopping and starting the evaluate service in T-12 (a terminal).

## What this session built

Each HQ has **ASA rules** (S40): the default **High risk** rule says "When the suggested ASA is III or more: add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission". Until now only the auto-triage action did anything (S42). Now, **when a patient (or staff for them) sends the HQ**, every rule that fires also acts:

- **A task:** added to the episode's **Tasks** panel for the rule's team, saying **"Added by rule: High risk on …"**. An open task for the anaesthetist team puts the episode on the **Anaesthetic review needed** worklist (S21). The same rule's task isn't added twice while it's still open.
- **A flag:** a red badge (e.g. **Anaesthetic review**) next to the patient's name on the episode page and under the name in the episode lists, plus a **Flags** card on the episode page. **Any clinician at the hospital clears it with a short reason** (Rahul's answer): who, when and why stay on the card, and the General notes get a line.
- **The anaesthetist's ASA required:** while the episode has no **Anaesthetist ASA grade**, a move to **Ready for admission** is refused ("Add the anaesthetist's ASA grade first."), and the status control says why. Saving the anaesthetist's grade (POA Summary) meets it. The nurse's grade doesn't.
- **A suggested status:** a line on the episode, e.g. **"The HQ's rules suggest: Ready for POA (the rule “…”)"**; the status doesn't change by itself.
- Each task, flag and requirement gets an **Automatic** line in the General notes. If the evaluate service is down when the HQ is sent, none of this happens (and the notes say "The automatic checks didn't run…", S41).

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. Sign in on the stub login page by clicking a name (sign out from the top right to switch):

| User | Role | Where |
| --- | --- | --- |
| **Alex Author** | super clinician | the whole of Trust A (writes and publishes HQs) |
| **Cara Clinician** | clinician | Hospital A1 only |
| **Sue Super** | super clinician | Hospital A1 only (re-opens and archives) |
| **Val Clinician** | clinician | Hospital A2 only |
| **Bea Author** | super clinician | the whole of Trust B |

- **Captured messages:** texts and emails aren't sent; they're listed at **http://localhost:3000/qa/messages** (newest first).
- **The patient's link:** a new episode with a mobile or email sends an invite; open the link from `/qa/messages` in a **phone-sized window (390×844)**, tap **Continue**, enter the date of birth.
- Use **made-up** patients only (e.g. "Sam MT43 <time>", born 17/05/1980, a made-up email such as `mt43.<time>@example.org`).

### The test HQ (built in T-01)

| Question (Select One, Yes / No, required) | Disclosure on **Yes** |
| --- | --- |
| "Do you smoke?" | SNOMED **77176002** (Smoker): fills in **ASA II** by itself; add a note "Smoker" (category Lifestyle) |
| "Do you have heart failure?" | ICD10 **I50.9**: choose **III**; add a note "Heart failure" (category Cardiovascular) |

**How to add a disclosure:** in the editor, click a question's card; in the right-hand panel open **Disclosures**; **Add** next to an option; **+ Add disclosure**; add a code (the picker has a SNOMED / ICD10 switch) and a note with a category; **Save output**, then **Done**.

So: **heart failure Yes** = ASA III → **High risk** fires. **Smoker only** = ASA II → **Low risk** fires (auto-triaged to Ready for admission, as in S42; no task or flag).

## Test cases

Do them in order.

### T-01: build and publish the test HQ

1. As **Alex**, Questionnaires → **New**: name **"MT S43 <time>"**.
2. Question Set **"About you"** (patient): the two questions in the table, each **required**, with their disclosures.
3. Question Set **"Nurse checks"** (clinician): one text question.
4. Open **Rules** (the editor's header) and read the two rules. Go back and **Publish**.

**Expected:** the Rules page has **Low risk** and **High risk**; High risk reads "When the suggested ASA is III or more: add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission". It publishes.

### T-02: a high-risk patient sends the HQ on a phone (happy path)

1. As **Cara** (Hospital A1), **New episode** on "MT S43 <time>" for a made-up patient with an email.
2. Open the invite from `/qa/messages` in a **390×844** window. Answer **Smoke: No**, **Heart failure: Yes**. Send.
3. As **Cara**, open the episode list: look at the **Anaesthetic review needed** tab and the **Auto-triaged** tab.
4. Open the episode.

**Expected:**
- The patient's pages fit the phone (no sideways scroll) and say nothing about rules, tasks or flags.
- The episode is on **Anaesthetic review needed** (its count includes it), **not** on Auto-triaged. Its row shows a red **Anaesthetic review** badge under the patient's name, and **Ready for review**.
- The episode page: a red **Anaesthetic review** badge beside the patient's name; a **Flags** card with "Anaesthetic review" and "Added by the rule “High risk” on <date, time>" and a **Clear** button; no Auto-triaged card.
- **Tasks:** "Book anaesthetic review", **Anaesthetist**, "Added by rule: High risk on <date, time>", with **Complete**.
- Under the status control, in red: **"Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”). Add it on the POA Summary."**
- **General notes**, each marked **Automatic**: "Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review", "Flag added by the rule “High risk”: Anaesthetic review", "The rule “High risk” requires the anaesthetist's ASA grade before Ready for admission".

### T-03: Ready for admission is refused until the anaesthetist's grade is saved

1. On the T-02 episode, **Validate** each Question Set (keep the answers), then **Complete HQ review** (→ Ready for POA).
2. Open the **Episode status** dropdown.
3. Pick **On hold** (it works), then open the dropdown again.
4. Open the **POA Summary**; on the validated tab, set **Nurse ASA grade** to ASA III and **Save**. Go back to the episode and look at the dropdown again.

**Expected:** the dropdown lists **"Ready for admission (add the anaesthetist's ASA grade first)"** greyed out: it can't be picked. The red line stays under the status control. Other moves (On hold, Ready for POA…) work. The nurse's grade doesn't change anything. On the POA Summary's ASA grades section, a red line says "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”)."

### T-04: saving the anaesthetist's grade meets it

1. On the POA Summary, set **Anaesthetist ASA grade** to ASA III and **Save** (its own Save button).
2. Use the browser's **Back** to the episode. Pick **Ready for admission** from the dropdown and confirm.
3. Reload.

**Expected:** after saving, the POA Summary's line turns grey: "The anaesthetist's ASA grade is added, as the rule “High risk” asks." On the episode the red line has gone, and **Ready for admission** is offered normally; the move works and is kept after the reload.

### T-05: clear the flag with a reason

1. On the same episode's **Flags** card, click **Clear**.
2. Click **Clear flag** with the reason empty.
3. Type a reason (e.g. "Reviewed by Dr Example") and press **Enter**.
4. Reload. Go to the episode list (All, then Anaesthetic review needed).

**Expected:** step 2 says "Say why the flag is cleared." and nothing is cleared. After step 3 the red badge beside the name is gone; the card keeps the flag (grey) with "Cleared by Cara Clinician on <date, time>: Reviewed by Dr Example", and no Clear button. The General notes say "Flag cleared: Anaesthetic review: Reviewed by Dr Example" by Cara. All kept after the reload. The list row has no flag badge. (The episode stays on Anaesthetic review needed while its task is open: the flag and the task are separate.)

**Also:** when the Clear form opens, the reason box is focused (typing goes into it). **Back** on the open Clear form closes it without clearing.

### T-06: someone at another hospital or trust can't see or clear it

1. Make another high-risk episode (as T-02) and copy its URL. Don't clear its flag.
2. Sign in as **Val** (Hospital A2) and open the copied URL.
3. Sign in as **Bea Author** (Trust B) and open the copied URL.
4. As **Val**, look at Hospital A2's episode list.

**Expected:** steps 2 and 3 show "You don't have access to this hospital" (the URL names Hospital A1; S08's guard), with no patient details or flag; with their own hospital in the URL instead, "Episode not found.". The episode isn't in Val's or Bea's lists. Back as Cara, the flag is still standing.

### T-07: a second tab's Clear is told the episode changed

1. As **Cara**, open the T-06 episode in **two tabs**.
2. In tab 1, clear its flag with a reason.
3. In tab 2 (not reloaded), click **Clear**, type a reason and send.

**Expected:** tab 2 shows "This episode changed since you opened it. Reload to see the latest; your change was not saved." with **Reload**, keeping the typed reason. After Reload, the flag shows as cleared by Cara (tab 1's reason), with no Clear button, and the Clear form has closed.

### T-08: completing the rule's task takes it off Anaesthetic review needed

1. On the T-02 episode, **Complete** the "Book anaesthetic review" task.
2. Open the **Anaesthetic review needed** tab.

**Expected:** the task shows Done, "Completed by Cara Clinician on …" (and still "Added by rule: High risk"). The episode has left Anaesthetic review needed. The status and the cleared flag are unchanged.

### T-09: a low-risk patient: no task, flag or requirement

1. **New episode** on "MT S43 <time>". On a phone, answer **Smoke: Yes**, **Heart failure: No**. Send.
2. As **Cara**, open the episode.

**Expected:** auto-triaged as in S42 (the "Auto-triaged by the rule “Low risk”" card, Ready for admission, on the Auto-triaged tab). **No** flag badge or Flags card, **no** rule task, **no** red requirement line, and no "Task added…" / "Flag added…" lines in the General notes.

### T-10: a rule that suggests a status shows the line and changes nothing

1. As **Alex**, open "MT S43 <time>" and click **Create new version**. Open **Rules**, **Edit Low risk**: remove its auto-triage action and add **Suggest a status: Ready for POA**. **Save rule**. Publish version 2.
2. As **Cara**, **New episode** on **version 2**. On a phone, answer **Smoke: Yes**, **Heart failure: No**. Send.
3. Open the episode; reload it.

**Expected:** the episode is **Ready for review** (not auto-triaged; no Auto-triaged card). Under the status control, in blue: **"The HQ's rules suggest: Ready for POA (the rule “Low risk”)"**, kept after the reload. Nothing else moves; the dropdown still starts at Ready for review.

### T-11: a cancelled or archived episode's flag can't be cleared

1. Make another high-risk episode (as T-02).
2. As **Sue Super**, **Archive episode** (a reason is optional).
3. Look at the Flags card.

**Expected:** the flag badge and the card are shown, but there's **no Clear** button. (The same for a cancelled one, if you try that too.)

### T-12: the evaluate service down: none of the rules' actions

1. In a terminal, find the evaluate service: `lsof -nP -iTCP:8090 -sTCP:LISTEN`, and stop that process (`kill <PID>`).
2. **New episode** on "MT S43 <time>" **version 2** (publishing it retired version 1) for a new patient; on the phone answer heart failure **Yes** and send.
3. Open the episode.
4. **Start the evaluate service again**: in a terminal at the repo, `make evaluate` (leave it running), and check `lsof -nP -iTCP:8090 -sTCP:LISTEN` shows one listener. **Report the PID of the copy you started**, so it can be stopped afterwards.

**Expected:** the episode is **Ready for review**, the General notes say "The automatic checks didn't run. Review this HQ by hand.", and there's **no** flag, rule task or requirement line.

*(Version 2's Low risk suggests instead of auto-triaging; its High risk is unchanged.)*

## Regression

- **R-01 (S21):** add a task by hand ("Add task", Nurse team): it says "Added by Cara Clinician on …"; complete and re-open it.
- **R-02 (S20):** an episode with no rule requirement (e.g. the T-09 one, re-opened by Sue and reviewed, or any reviewed seeded episode) moves to Ready for admission as before.
- **R-03 (S42):** the T-09 episode's **Checked** works and it leaves the Auto-triaged tab.
- **R-04 (S40):** the Rules page of a new HQ still shows the two default rules and saves an edit.
- **R-05 (S22):** **Show archived** in the episode list shows the T-11 episode, with its flag badge.

## Out of scope (don't report)

- **Sending the HQ a second time:** an HQ is sent once; nothing re-sends it yet (rules running again later is a later session). A re-open doesn't run the rules again, so it adds no task or flag. The automated tests check that a second send wouldn't add the same open task or standing flag twice, and that a cleared flag would come back.
- A worklist for each flag name (the anaesthetists' tasks already have Anaesthetic review needed).
- Server-side checks on send and on Complete review (S44); the Lambda (S45).
- The patient's thank-you page wording after an auto-triaged send (in pending.md from S42).
