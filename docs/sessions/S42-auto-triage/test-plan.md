# S42 test plan: auto-triage and the Auto-triaged worklist

*Changed after test report 1:*
- **F-1 fixed:** a Checked refused because the episode changed (another tab checked it) now shows the message in red on the card, above "Checked by…". Retest **T-04** step 3.
- **E-2 fixed:** Test the whole HQ, the Preview and whole-HQ test cases now apply the same guards as a real send: a rule that would auto-triage says "no auto-triage: a red or amber flag" (or "a score is Incomplete", or "the anaesthetist's ASA is required first"). The Publish window's line ends "Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first.", and the Rules page's intro says the same. See **T-02** and the new **T-15**.
- **E-1 fixed:** for an HQ sent by staff, the card says "when <name> sent the HQ on the patient's behalf"; the on-behalf confirm says "The episode moves to Ready for review, or on from there if the HQ's rules auto-triage it."; the sent page says "The answers are with the team. The episode page shows its status now." See **T-07**.
- **E-4:** publishing v2 retires v1, so **T-10 and T-13 use the newest published version** whose Low risk target is Ready for admission (v2 after T-09; do T-13 before T-12).
- Categories: use any existing category (e.g. Cardiovascular, Lifestyle); a smoker disclosure **with a note** ("Smoker", Lifestyle) makes the POA Summary list it.

For a tester who has never seen the code. Everything is in the browser, except stopping and starting the evaluate service in T-13 (a terminal).

## What this session built

When a patient (or staff for them) sends the HQ, the HQ's **ASA rules** (built in S40, worked out by the evaluate service since S41) now **act**:

- **Auto-triage.** If a rule that auto-triages fires (the default **Low risk** rule: suggested ASA I or II and no red flags → **Ready for admission**), the episode moves **straight on** from Ready for review to the rule's target, with nobody involved. The status history shows **"Auto-triaged: Ready for review → Ready for admission"** by **"Automatic (Low risk)"**, and the General notes get an **Automatic** line: **"Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission"**.
- **Guards that can't be switched off.** Even when the rule fires, the episode stays **Ready for review** if: the answers have a **red or amber flag**, a **score is Incomplete** (an input not answered), or a rule asks for the anaesthetist's ASA first. Then the General notes say **"Not auto-triaged by the rule “Low risk”: <why>. Review this HQ by hand."** If the evaluate service is down, nothing moves either ("The automatic checks didn't run…", S41).
- **The stamp.** An auto-triaged episode's page has a card **"Auto-triaged by the rule “Low risk”"**, and every Question Set says **"Auto-triaged (ASA II, no red or amber flags)"** instead of "Not started", on the episode page and on the POA Summary. The POA Summary shows the **patient's own answers** (nobody validated them).
- **The Auto-triaged worklist.** A new tab on the clinician home, after Ready for review: **"Auto-triaged"**, help line "Moved on automatically by the HQ's rules; not yet checked by a clinician." The episode stays on it until **any clinician at the hospital** clicks **Checked** on the episode's card (who and when are shown; a General notes line "Auto-triage checked").
- **The patient's message.** If the hospital has switched on the **Ready for admission** text or email (Settings), it goes **at once** when the episode is auto-triaged to Ready for admission (by default). A new setting on each HQ's **Rules** page holds it until a clinician clicks **Checked** instead.
- **Re-open (EPI-07)** works on an auto-triaged episode (super clinicians, with a reason): back to Ready for review, the card and stamps go, it leaves the Auto-triaged list, and the sets can be validated.

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
| **Ada Admin** | admin | the whole of Trust A (Settings) |
| **Cara Clinician** | clinician | Hospital A1 only |
| **Sue Super** | super clinician | Hospital A1 only (can re-open a review) |
| **Val Clinician** | clinician | Hospital A2 only |

- **Captured messages:** texts and emails aren't sent; they're listed at **http://localhost:3000/qa/messages** (newest first).
- **The patient's link:** a new episode with a mobile or email sends an invite; open the link from `/qa/messages` in a **phone-sized window (390×844)**, tap **Continue**, enter the date of birth.
- Use **made-up** patients only (e.g. "Sam MT42 <time>", born 17/05/1980, mobile `07700 9004xx`).

### The test HQ (built in T-02)

| Question (Select One, Yes / No) | Disclosure on **Yes** |
| --- | --- |
| "Do you smoke?" | SNOMED **77176002** (Smoker): fills in **ASA II** by itself |
| "Do you get chest pain when you walk?" | a note "Chest pain on exertion" (category Cardiac), **Review flag: red** |
| "Are you tired in the day?" | a note "Daytime tiredness" (category Sleep), **Review flag: amber** |
| "Do you have heart failure?" | ICD10 **I50.9**: choose **III** |

**How to add a disclosure:** in the editor, click a question's card; in the right-hand panel open **Disclosures**; **Add** next to an option; **+ Add disclosure**; add a code (the picker has a SNOMED / ICD10 switch) and/or a note with a category; **Review flag** sets amber or red; **Save output**, then **Done**.

So: **"No" to everything** = no ASA grade (counted as ASA I) → auto-triaged. **Smoker only** = ASA II → auto-triaged. **Heart failure** = ASA III → Ready for review. **Chest pain** = red flag → Low risk doesn't fire. **Tired** = amber flag → Low risk fires but the guard stops it.

## Test cases

Do them in order. T-13 stops the evaluate service and starts it again at the end.

### T-01: the Rules page's new message setting

1. As **Alex**, Questionnaires → **New**: name **"MT S42 <time>"**. Open **Rules** (the editor's header).
2. Look at the settings card at the top.
3. Tick **"When a rule auto-triages a patient, wait for a clinician to click Checked before the patient's status message goes"**; reload the page; untick it; reload.

**Expected:** the card has the S40 no-grade box (ticked) and the new box, **unticked by default**, with "Off: the hospital's message goes straight away. The episode is still on the Auto-triaged list for a clinician to check." Ticked, its description says the message goes once a clinician has checked the episode; each change is kept after a reload. The two default rules are unchanged (Low risk → Ready for admission).

### T-02: build and publish the test HQ

1. In "MT S42 <time>", Question Set **"About you"** (patient): the four questions in the table above, each **required**, with their disclosures.
2. Question Set **"Nurse checks"** (clinician): one text question.
3. **Publish**. Read the Publish window first.

**Expected:** the Publish window says "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I. Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first." (no sentence about Checked, since the T-01 box is off). It publishes.

### T-03: a low-risk patient sends the HQ on a phone: auto-triaged at once (happy path)

1. As **Cara**, **New episode**: a new made-up patient with a **mobile**, on **MT S42 <time> v1**; **Create episode**.
2. Open the invite from `/qa/messages` at **390×844**; answer **Yes** to "Do you smoke?" and **No** to the rest; **Send my answers**.
3. As Cara, open the clinician home (**Episodes**) and click the **Auto-triaged** tab.
4. Open the episode.
5. Click **POA Summary**; look at both tabs.

**Expected:**
- The patient sees the normal thank-you page; nothing wider than the phone's screen.
- The **Auto-triaged** tab (right after Ready for review) has a count and the help line "Moved on automatically by the HQ's rules; not yet checked by a clinician."; the episode is listed as **Ready for admission**. It is **not** on the Ready for review tab.
- The episode page: status **Ready for admission**; a teal card **"Auto-triaged by the rule “Low risk”"** saying "Moved to Ready for admission automatically when the patient sent the HQ (ASA II, no red or amber flags), on <date, time>. No clinician validated the answers…", then "Not yet checked by a clinician." and a **Checked** button.
- Health questionnaire: each set (About you, Nurse checks) says **"Auto-triaged (ASA II, no red or amber flags)"**, with a **View** button (no Validate, no Complete HQ review).
- Status history: "HQ sent: HQ not complete → Ready for review" by **Patient**, then **"Auto-triaged: Ready for review → Ready for admission"** by **"Automatic (Low risk)"**.
- General notes: "HQ completed by the patient", then **"Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission"**, by **Automatic**.
- The status control offers no **Undo**.
- POA Summary: a teal line "Auto-triaged (ASA II, no red or amber flags) by the rule “Low risk”. No clinician validated these answers, so this shows the patient's own."; each set's by-line starts "Auto-triaged (ASA II, no red or amber flags) · Patient on <date>"; the Smoker disclosure is listed.

### T-04: Checked takes it off the list (happy path, reload, back)

1. On the T-03 episode, click **Checked**.
2. Reload the page. Then go to **Episodes → Auto-triaged**; press the browser's **Back**, then **Forward**.
3. Open the T-03 episode in **two tabs**. Do T-03 again for a new patient (to have an unchecked one) and open that one in two tabs; click **Checked** in the first tab, then in the second tab (without reloading).

**Expected:**
- After Checked: the card says **"Checked by Cara Clinician on <date, time>."**, the button is gone, the status is still **Ready for admission**; General notes get "Auto-triage checked" by **Cara Clinician**. The same after a reload.
- The episode is no longer on the Auto-triaged tab, and the tab's count went down by one; Back/Forward keep the tab (`?worklist=auto_triaged` in the address).
- The second tab's Checked is refused in plain words: a red message on the card ("This episode changed since you opened it…"), with "Checked by Cara Clinician on …" below it and no Checked button; nothing is recorded twice.

### T-05: a high-risk patient: Ready for review, not auto-triaged

1. As Cara, a new episode on MT S42 v1; the patient answers **Yes** to "Do you have heart failure?" (ASA III), No to the rest; sends.
2. Open the episode; check the Auto-triaged and Ready for review tabs.

**Expected:** **Ready for review**; no auto-triage card; the sets say "Not started" with **Validate**; no "Auto-triaged" line in the history or notes; on **Ready for review**, not on **Auto-triaged**.

### T-06: the flags: red stops the rule, amber is stopped by the guard

1. A new episode; the patient answers **Yes** to "Do you smoke?" and **Yes** to "Do you get chest pain when you walk?" (red); sends.
2. A new episode; the patient answers **Yes** to "Do you smoke?" and **Yes** to "Are you tired in the day?" (amber); sends.

**Expected:**
- Red: **Ready for review**, nothing about auto-triage in the notes (Low risk doesn't fire with a red flag).
- Amber: **Ready for review**, with an **Automatic** note **"Not auto-triaged by the rule “Low risk”: the answers have a red or amber flag. Review this HQ by hand."**; not on Auto-triaged.

### T-07: staff filling in for the patient: auto-triaged the same

1. As Cara, a new episode on MT S42 v1; on its page, **Fill in for the patient**; answer **No** to everything; send.
2. Open the episode.

**Expected:** the confirm window says "…The episode moves to Ready for review, or on from there if the HQ's rules auto-triage it."; the sent page says "The answers are with the team. The episode page shows its status now." **Ready for admission**; the card says "…when Cara Clinician sent the HQ on the patient's behalf (no ASA grade, no red or amber flags)…", and each set "Auto-triaged (no ASA grade, no red or amber flags)"; history "HQ sent" by **Cara Clinician**, then "Auto-triaged…" by **"Automatic (Low risk)"**; notes "HQ completed on the patient's behalf" (Cara) then the auto-triage line (**Automatic**); on the Auto-triaged tab.

### T-08: the patient's message goes at once by default

1. As **Ada Admin**, **Settings → Settings for: Hospital A1**: for **Text the patient when the episode is Ready for admission**, untick "Use the trust's setting" and switch it **on**; **Save**.
2. As Cara, a new episode (with a mobile) on MT S42 v1; the patient answers No to everything and sends.
3. Open `/qa/messages` straight away, then the episode.

**Expected:** a text to the patient's mobile with the Ready for admission wording, there as soon as the HQ was sent (before anyone clicks Checked). General notes: "Ready for admission message sent by text message." by **Automatic**. The card says "Not yet checked by a clinician." with no line about the message.

### T-09: the HQ holds the message until Checked

1. As **Alex**, open MT S42 and make a **New version** (v2). On its **Rules** page, tick the T-01 box. **Publish** (the window now ends "…The patient's message goes when a clinician clicks Checked.").
2. As Cara, a new episode (mobile) on **MT S42 v2**; the patient answers No to everything; sends. Check `/qa/messages` and the episode.
3. Click **Checked**; check `/qa/messages` and the notes again.

**Expected:** after the send: **no** Ready for admission text (only the invite); the card says "Not yet checked by a clinician. The patient's message goes when you click Checked." After Checked: the text arrives, and the notes have "Auto-triage checked" (Cara) and "Ready for admission message sent by text message." (Cara).

**Clean-up:** as Ada, Settings → Hospital A1: tick "Use the trust's setting" for the switch you changed; Save.

### T-10: another hospital's episode can't be checked

1. Sign in as **Val Clinician** (Hospital A2). **New episode** on the newest **MT S42 <time>** version (it's the whole trust's; a new made-up patient); **Fill in for the patient**; answer **No** to everything; send. Copy the episode page's address.
2. Val: the **Auto-triaged** tab lists it.
3. Sign in as **Cara** (Hospital A1 only). Paste Val's episode address. Then look at Cara's Auto-triaged tab.

**Expected:** Val sees it on Auto-triaged at Hospital A2. Cara gets a "not found"-style page (no episode details, no Checked button), and her Hospital A1 tab doesn't list it.

### T-11: re-open an auto-triaged episode (Sue Super), then validate

1. As **Cara**, open an unchecked auto-triaged episode at Hospital A1 (e.g. from T-07). Look for a re-open button.
2. Sign in as **Sue Super**; open it; **Re-open HQ review** without a reason, then with the reason "MT: check by hand".
3. **Validate** each set, then **Complete HQ review**.

**Expected:**
- Cara (a clinician) has no Re-open.
- Sue: without a reason it's refused ("Say why the HQ review is being re-opened."); with it, the status is **Ready for review**; the auto-triage card and the stamps are gone (sets say "Not started" with Validate); history "HQ review re-opened: Ready for admission → Ready for review" by Sue with the reason; it leaves the Auto-triaged tab and is on **Ready for review**.
- Validating and completing the review work as usual: **Ready for POA**.

### T-12: the rule's target changed to Ready for POA

1. As **Alex**, MT S42: **New version** (v3). On **Rules**, **Edit Low risk**: Target **Ready for POA**; **Save rule**; untick the T-01 box. **Publish** (the window says "→ Ready for POA").
2. As Ada, switch Hospital A1's Ready for admission text on again (as T-08).
3. As Cara, a new episode (mobile) on **MT S42 v3**; the patient answers No to everything; sends.

**Expected:** **Ready for POA** (not Ready for admission); notes "Auto-triaged by the rule “Low risk”: no ASA grade, no red or amber flags → Ready for POA"; history "Auto-triaged: Ready for review → Ready for POA"; on the Auto-triaged tab; **no** Ready for admission text. **Clean-up** as in T-09.

### T-13: the evaluate service stopped: nothing is auto-triaged

1. Stop **only** the evaluate service: `kill $(lsof -ti tcp:8090 -sTCP:LISTEN)`. Check `curl -s 127.0.0.1:8090/health` fails, while http://localhost:3000 still works.
2. As Cara, a new episode on the newest MT S42 version that targets Ready for admission (v2, so do this before T-12); the patient answers No to everything (low risk); sends.
3. Open the episode; check the tabs.
4. **Start it again** (always, even if a step failed): `(make evaluate > /tmp/mt-s42-evaluate.log 2>&1 &)`, then `curl -s --retry 10 --retry-delay 1 --retry-connrefused 127.0.0.1:8090/health`. **Report the PID** of the copy you started (`lsof -ti tcp:8090 -sTCP:LISTEN`) in your test report.
5. Send one more low-risk HQ on that version.

**Expected:** with it stopped: the patient sees the normal thank-you page; the episode is **Ready for review**, with "The automatic checks didn't run. Review this HQ by hand." (**Automatic**), no auto-triage card or line; not on Auto-triaged. After the restart, health answers ok and the next low-risk send is auto-triaged to Ready for admission.

### T-14: the worklist with cancelled and archived episodes

1. As Cara, note the Auto-triaged tab's count; check it equals the number of rows in its list.
2. Cancel one unchecked auto-triaged episode (Cancel, reason "Patient withdrew"). Check the tab.
3. As **Sue**, archive another unchecked one. Check the tab; then un-archive it (reason "MT").

**Expected:** the count always equals the list's rows; the cancelled one leaves the list (and its page has no Checked button); the archived one leaves it, and comes back once un-archived (still unchecked).

### T-15: Test the whole HQ and the Preview say what a real send does (E-2)

1. As **Alex**, open the newest MT S42 version's **Rules → Test the whole HQ**. Answer **Yes** to "Do you smoke?" and **Yes** to "Are you tired in the day?" (amber); No to the rest.
2. Change "tired" to **No**.
3. Open About you's **Preview**; answer the same (smoker + tired).
4. On the Rules page, read the intro under the title.

**Expected:** step 1: Rules that fire "Low risk: no auto-triage: a red or amber flag"; step 2: "Low risk: auto-triage to …"; step 3: the Preview's Rules that fire says the same as step 1; step 4: the intro says a rule never auto-triages a patient with a red or amber flag or an Incomplete score, or when the anaesthetist's ASA is required first. (A real send with these answers is T-06's amber case: Ready for review.)

## Regression

- **R-01 (S20/J4):** on an HQ with no auto-triage (e.g. T-05's patient), Cara validates every set and completes the review: Ready for POA, as before.
- **R-02 (S21):** with the Ready for admission text switched on, Cara picks **Ready for admission** by hand on a reviewed episode: the text goes, as before. (Clean up the switch afterwards.)
- **R-03 (S40):** the Rules page's Test the whole HQ still shows "Suggested ASA II" and "Low risk" firing for a smoker; whole-HQ test cases still pass/fail as before.
- **R-04 (S39):** the episode page's Suggested ASA line and the POA Summary's ASA section for the T-03 episode.
- **R-05 (S41):** the stored result for a send still has no error (the "automatic checks didn't run" line appears only when the service is down).

## Out of scope (don't report)

- Tasks, flags, "anaesthetist's ASA required" and suggested statuses from rules (S43): the High risk rule's task and flag don't appear yet.
- Auto-triage when a clinician completes a review, or rules run again later.
- The default wording of the Ready for admission message (it's the hospital's setting, from S21).
- The server refusing an incomplete send (S44).
- "Edit procedure and dates" still showing on an auto-triaged episode (the review was never completed); note it if it gets in the way, but it's not this session's.
