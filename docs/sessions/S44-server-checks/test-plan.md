# S44 test plan: the server's checks on send and on Complete review

For a tester who has never seen the code. Most cases use the browser. A few send a request straight to the API from the browser's console (as a person bypassing the web page would), and two stop the evaluate service from a terminal, as each step says.

## What this session built

The web app already stops two things in the browser: **Send my answers** stays disabled until every section the patient sees is complete, and **Complete HQ review** stays disabled until every Question Set shown is validated. Now **the server checks both too**, so a request sent straight to the API can't get round them:

- **Sending an HQ** (the patient's own send, and staff's **Fill in for the patient** send) with a section the patient's form shows still unfinished is refused: **"Please finish “Your health” before sending your answers."** (every unfinished section is named, e.g. "“About you” and “Your health”"). Nothing is sent: the episode stays **HQ not complete**, the patient can carry on.
- **Complete HQ review** while a Question Set shown isn't validated is refused: **"Validate every Question Set first: “About you” isn't validated."** (several: "“Your health” and “Nurse checks” aren't validated."). A set hidden by its condition doesn't need validating. Which sets are shown follows the answers that count now: the clinician's corrected answers where they've validated or started a set, else the patient's.
- **If the evaluate service can't be reached**, both work as before this session (the browser's checks still stand): the send goes through (with S41's "The automatic checks didn't run…" line), and Complete review goes through. The API's log gets an `EVALUATE_FAILED` line.

Nothing new appears on screen in normal use.

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running: it starts the API (:8080), the web app (:3000) and the evaluate service (127.0.0.1:8090). Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

- **Alex Author**: super clinician, the whole of Trust A (writes and publishes HQs).
- **Cara Clinician**: clinician, Hospital A1 (creates episodes, fills in for patients, validates).
- **Val Clinician**: clinician, Hospital A2 only (for a refused case).

Check the evaluate service is up: `curl -s 127.0.0.1:8090/health` answers `{"ok":true,…}`.

**Patient links:** the invite email is on **http://localhost:3000/qa/messages** (captured, never sent). Open the link in a separate window at **390×844**; the made-up date of birth is the one you typed for the patient.

### Sending a request straight to the API (from the browser's console)

Use the browser tab that's already signed in (the patient's tab for a patient's request, Cara's for staff), open the developer console, and run the line given; it prints the status and the API's answer. IDs come from the page's address: a staff page is `http://localhost:3000/o/<ORG>/h/<HOSP>/episodes/<EID>`; the patient's form is `http://localhost:3000/home/episodes/<EID>`.

```js
// the patient's send (in the patient's tab)
fetch('/api/p/episodes/<EID>/submit', {method: 'POST'}).then(async r => r.status + ' ' + await r.text())
// staff sending for the patient (in Cara's tab)
fetch('/api/o/<ORG>/h/<HOSP>/episodes/<EID>/patient-form/submit', {method: 'POST'}).then(async r => r.status + ' ' + await r.text())
// Complete HQ review (in Cara's tab)
fetch('/api/o/<ORG>/h/<HOSP>/episodes/<EID>/complete-review', {method: 'POST'}).then(async r => r.status + ' ' + await r.text())
```

A refusal prints e.g. `409 {"error":"Please finish …"}`; success prints `204 ` (nothing after it). Before running a send line, give the page a second to save the last answer (it saves as you go).

### The test HQ (build it once, as Alex, before T-05)

Questionnaires → **New Questionnaire**, name "S44 test <today's time>". Give it three Question Sets, in this order:

1. **About you** (Who is it for? **Patient**): one **required** single-choice question "Do you snore?" with options Yes and No.
2. **Sleep** (Patient): one **required** single-choice question "Are you tired in the day?" (Yes / No). **Display this Question Set → Conditionally**: "Do you snore?" is **Yes**.
3. **Nurse checks** (Who is it for? **Clinician**): one **required** text question "Blood pressure". **Display this Question Set → Conditionally**: "Do you snore?" is **Yes**.

Then open **Rules** (the editor header) and **untick** "Patients whose answers give no ASA grade (usually fit and well) count as ASA I, so the rules can auto-triage them", so a send goes to **Ready for review** (not straight to Ready for admission). Publish.

## Test cases

Do T-11 and T-12 last: they stop the evaluate service, and T-12 starts it again.

### T-01: the normal send in the app still works (phone)

1. As **Cara**, **New episode** for a new made-up patient on the seeded **Demo pre-op HQ** (with an email, so the invite is captured).
2. Open the invite at **390×844**, enter the date of birth, answer every section, **Send my answers**.

**Expected:** the normal sent screen, no error. As Cara, the episode is **Ready for review** (or, since the seeded demo HQ's rules auto-triage a patient with no ASA grade, **Ready for admission** with the Auto-triaged note; both are fine: the send was accepted).

### T-02: a patient's send with a section unfinished is refused, naming it

1. As Cara, a new episode on **Demo pre-op HQ**; open the invite at 390×844; answer **About you** fully; leave **Your health** untouched. ("Send my answers" stays disabled: that's the browser's check.)
2. In the patient's tab, run the patient's send line (Setup).
3. Reload the patient's page. As Cara, open the episode.

**Expected:**
- `409 {"error":"Please finish “Your health” before sending your answers."}`.
- The patient's page still shows the sections to fill in (not the sent screen); About you's answers are still there.
- Cara's episode page: **HQ not complete**; no "HQ completed by the patient" line in the General notes; the Status history has no Submit.

### T-03: two sections unfinished: both named; then finishing works

1. On a new episode on **Demo pre-op HQ**, open the invite, answer **one** question of About you only, and run the patient's send line.
2. Then answer every section in the page and press **Send my answers**.

**Expected:** step 1: `409` with "Please finish “About you” and “Your health” before sending your answers."; step 2: sent normally, and Cara sees the episode moved on (as T-01).

### T-04: staff sending for the patient is refused the same

1. As Cara, a new episode on **Demo pre-op HQ**; **Fill in for the patient**; answer About you only.
2. In Cara's tab run the staff send line for that episode.
3. Back on the Fill in page, complete Your health and send it from the page.

**Expected:** step 2: `409` with "Please finish “Your health” before sending your answers."; the episode is still **HQ not complete** with no "completed on the patient's behalf" line. Step 3: sent normally.

### T-05: a section the patient doesn't see doesn't need finishing

(Uses the test HQ.)
1. As Cara, a new episode on the **S44 test** HQ; open the invite at 390×844. Answer **Do you snore? = No**. Only About you is listed (Sleep is hidden). Press **Send my answers**.
2. A second episode on the same HQ: answer **Do you snore? = Yes**, leave **Sleep** unanswered, and run the patient's send line.

**Expected:** step 1: sent; Cara sees **Ready for review**. Step 2: `409` with "Please finish “Sleep” before sending your answers."; the episode stays HQ not complete.

### T-06: Complete review with sets not validated is refused, naming them

(Uses the test HQ, whose sends stay **Ready for review**.)
1. Send an S44 test HQ as the patient with **snore = Yes** and Sleep answered. As Cara, open the episode: the card lists **About you**, **Sleep** and **Nurse checks**. Validate **About you** only (open it, **Validate**).
2. In Cara's tab run the Complete review line for that episode.
3. Validate **Sleep** too; run the line again.
4. Validate **Nurse checks** (type a blood pressure); press **Complete HQ review** on the page.

**Expected:** step 2: `409 {"error":"Validate every Question Set first: “Sleep” and “Nurse checks” aren't validated."}`, and the episode stays **Ready for review** (no "HQ review completed" line). Step 3: `409` with "… “Nurse checks” isn't validated." Step 4: completes, **Ready for POA**, "HQ review completed by Cara Clinician".

### T-07: Complete review with nothing validated

1. Send another S44 test HQ with **snore = Yes** and Sleep answered. As Cara, with nothing validated, run the Complete review line.

**Expected:** `409` naming **all three**: "Validate every Question Set first: “About you”, “Sleep” and “Nurse checks” aren't validated." Status unchanged.

### T-08: a set hidden by its condition doesn't need validating

1. As Cara, open T-05 step 1's episode (S44 test HQ, snore = No, Ready for review). The Health questionnaire card lists only **About you** (Nurse checks is hidden).
2. Validate **About you** without changing the answer. Press **Complete HQ review**.

**Expected:** completes: **Ready for POA**. (Nurse checks never needed validating.)

### T-09: the clinician's corrected answers decide which sets count

1. Send another S44 test HQ as the patient with **snore = No** (Ready for review).
2. As Cara, open **About you**, change "Do you snore?" to **Yes**, and **Validate**. Back on the episode page the card now lists **Sleep** and **Nurse checks** too, not validated.
3. Run the Complete review line.
4. Validate Sleep (answer it) and Nurse checks (type a blood pressure); press **Complete HQ review**.

**Expected:** step 3: `409` with "Validate every Question Set first: “Sleep” and “Nurse checks” aren't validated."; status unchanged. Step 4: completes.

### T-10: refused paths are as before

1. As **Val Clinician** (Hospital A2), in Val's tab run the Complete review line for a Hospital A1 Ready-for-review episode (use Hospital A1's ORG/HOSP and the episode's EID).
2. Signed out (a private window with no sign-in), run `fetch('/api/p/episodes/<EID>/submit', {method:'POST'}).then(r => r.status)`.
3. In Cara's tab, run the Complete review line on an episode that's still **HQ not complete**.
4. In the patient's tab of an episode already sent (T-01), run the patient's send line again.

**Expected:** 1: `404` (Episode not found). 2: `401`. 3: `409` "The patient hasn't sent their answers yet." 4: `409` "These answers have already been sent."

### T-11: with the evaluate service stopped, both work as before

1. Stop **only** the evaluate service: `kill $(lsof -ti tcp:8090 -sTCP:LISTEN)`. Check `curl -s 127.0.0.1:8090/health` now fails while http://localhost:3000 still works.
2. On a new Demo pre-op episode, answer About you only and run the patient's send line.
3. As Cara, open that episode (Ready for review). With nothing validated, run the Complete review line.

**Expected:**
- Step 2: `204` (accepted, as before this session). The episode is **Ready for review** with "The automatic checks didn't run. Review this HQ by hand." in the General notes (Automatic).
- Step 3: `204`: **Ready for POA**, "HQ review completed by Cara Clinician".
- *If you can see the API's output* (the `make dev` terminal): an `EVALUATE_FAILED form <id>: …` line for the send and an `EVALUATE_FAILED complete review, form <id>: …` line for the review, with no answers in them. (In an automatic run you may not see it: mark this bullet as a hand check; automated tests cover it.)

### T-12: start it again; the checks are back

1. Start the evaluate service again in the background: `(make evaluate > /tmp/mt-s44-evaluate.log 2>&1 &)`, then `curl -s --retry 10 --retry-delay 1 --retry-connrefused 127.0.0.1:8090/health`. **Note the PID** of the new copy (`lsof -nP -iTCP:8090 -sTCP:LISTEN`) and put it in your report, so the conductor can stop it before restarting `make dev`.
2. Repeat T-02 on a new episode.

**Expected:** health answers ok; the send is refused again with "Please finish “Your health” …".

## Regression

- **R-01 (S34):** an HQ with a required "I have watched this video" tick still refuses a send until ticked (skip if none is handy).
- **R-02 (S41):** after T-01's send, the episode page shows the Suggested ASA line as before (if the HQ gives one) and nothing new appears.
- **R-03 (S42):** a Demo pre-op HQ sent completely by the patient (T-01) is auto-triaged to **Ready for admission** with the Auto-triaged note, and appears on the **Auto-triaged** worklist.
- **R-04 (S43):** the Flags card and the rule task still appear for a high-risk send, if you have one handy (skip otherwise).
- **R-05 (S20):** **Re-open review** (as Alex) on a Ready-for-POA episode, then Complete HQ review again: works (its sets are still validated).

## Out of scope

- Storing a result or re-running rules on Complete review.
- The Lambda on AWS (S45).
- The web app's own disabled buttons (unchanged); new screens (none).
