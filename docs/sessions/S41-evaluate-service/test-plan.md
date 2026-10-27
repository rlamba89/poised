# S41 test plan: the evaluate service, run locally

*Changed after test report 1:* F-1 is fixed: in T-09/T-10 the "automatic checks didn't run" line shows **"Automatic"**, whoever sent the HQ. Retest T-09 and T-10.

For a tester who has never seen the code. Most cases use the browser; a few read the database or call the API with `curl`, as each step says.

## What this session built

When an HQ (health questionnaire) is sent, by the patient or by staff for them, the server now works out **itself** what the answers give, and stores it: the Question Sets shown, the disclosures, scores and bands, the **suggested ASA** with its reasons, the red and amber flags, the **ASA rules that fire**, and whether each of the patient's sets is complete. A new small program, the **evaluate service**, does the working out (it runs the same code the browser uses). **Nothing acts on the result yet** (auto-triage comes in S42): the episode still goes to **Ready for review** as before, and nothing new shows on screen when it works.

- `make dev` now starts **three** processes: the API (:8080), the web app (:3000) and the evaluate service (**127.0.0.1:8090**).
- **If the evaluate service can't be reached** when an HQ is sent, the send still works for the patient; the episode is Ready for review, and the **General notes** get a line: **"The automatic checks didn't run. Review this HQ by hand."**
- **Publishing** now also asks the evaluate service for the same checks the Publish window shows (logic, test cases, rules and whole-HQ test cases), so a version with a failing test case can't be published even by calling the API directly. If the service can't be reached, publishing is refused with **"The checks couldn't run. Try again in a minute."**

## Setup

**In your own terminal** (the conductor does this in an automatic run):

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

- **Alex Author**: super clinician, the whole of Trust A (writes and publishes HQs).
- **Cara Clinician**: clinician, Hospital A1 (creates episodes, fills in for patients).

Check the evaluate service is up: `curl -s 127.0.0.1:8090/health` answers `{"ok":true,"evaluator":"clinical@…"}`.

### Reading the database

Some cases read the stored result. Run this in a terminal from the repo folder (if `docker` isn't found, put `PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"` in front). Replace `<EPISODE>` with the episode's id (the last part of the episode page's URL, `…/episodes/<EPISODE>`):

```sh
docker compose exec -T db psql -U sj -d sj -x -c "
SELECT o.id, o.created_at, o.evaluator_version, o.error,
       o.result->'suggestedAsa'->>'grade' AS asa,
       (SELECT string_agg(r->>'text', ' | ') FROM jsonb_array_elements(o.result->'suggestedAsa'->'reasons') r) AS asa_reasons,
       (SELECT string_agg(r->>'line', ' | ') FROM jsonb_array_elements(o.result->'rules') r) AS rules_that_fire,
       (SELECT string_agg((f->>'level') || ': ' || (f->>'text'), ' | ') FROM jsonb_array_elements(o.result->'flags') f) AS flags,
       (SELECT string_agg(d->'output'->'codes'->0->>'display', ', ') FROM jsonb_array_elements(o.result->'sets') s, jsonb_array_elements(s->'outputs') d) AS coded_disclosures,
       jsonb_array_length(o.result->'shown') AS sets_shown,
       o.result->'patientSets' AS patient_sets
FROM form_outputs o JOIN episode_forms f ON f.id = o.episode_form_id
WHERE f.episode_id = '<EPISODE>' ORDER BY o.id;"
```

### Calling the API as Alex (for T-07, T-08)

```sh
JAR=/tmp/mt-s41.jar
curl -s -c $JAR -X POST localhost:3000/api/dev/login -H 'Content-Type: application/json' -d '{"userId":"00000000-0000-4000-8000-0000000000a1"}' -o /dev/null -w '%{http_code}\n'   # 204: signed in as Alex
# The editor's URL is http://localhost:3000/o/<ORG>/h/<HOSP>/questionnaires/<QID>
Q=localhost:3000/api/o/<ORG>/h/<HOSP>/questionnaires/<QID>
V=$(curl -s -b $JAR $Q | python3 -c 'import sys,json; print(json.load(sys.stdin)["questionnaire"]["versionId"])'); echo $V
curl -s -b $JAR -X POST $Q/publish -H 'Content-Type: application/json' -d "{\"versionId\":\"$V\"}" -w '\nHTTP %{http_code}\n'
```

### Codes to use (from the seeded code list)

| Code set | Search for | What it gives |
| --- | --- | --- |
| SNOMED | `77176002` (Smoker) | fills in ASA II by itself |
| ICD10 | `I50.9` (Heart failure unspecified) | offers ASA III or IV: choose **III** |

### How to add a disclosure

In the editor, click a question's card; in the right-hand panel open the **Disclosures** tab; click **Add** next to an option; **+ Add disclosure**; add a code (the picker has a SNOMED / ICD10 switch) and/or a note with a category; **Review flag** sets amber or red; **Save output**, then **Done**.

## Test cases

Do them in order: T-09 to T-11 stop the evaluate service, and T-12 starts it again.

### T-01: make dev starts the evaluate service

1. In a terminal: `lsof -nP -iTCP:8090 -sTCP:LISTEN` and `curl -s 127.0.0.1:8090/health`.

**Expected:** a `node` process listens on 127.0.0.1:8090; health answers `{"ok":true,"evaluator":"clinical@<commit>…"}`.

### T-02: build and publish an HQ through the evaluate service (happy path)

1. As **Alex**, Questionnaires → **New**: name **"MT S41 <time>"** (e.g. "MT S41 1415").
2. Question Set **"About you"** (patient):
   - Select One **"Do you smoke?"**, Yes / No. Disclosure on **Yes**: SNOMED 77176002 (ASA II fills in).
   - Select One **"Do you get chest pain when you walk?"**, Yes / No. Disclosure on **Yes**: a note "Chest pain on exertion", category e.g. Cardiac, **Review flag: red**.
   - Select One **"Do you have heart failure?"**, Yes / No. Disclosure on **Yes**: ICD10 I50.9, choose **III**.
3. Question Set **"Nurse checks"** (clinician): any one text question.
4. Leave the default rules (the editor's **Rules** button shows Low risk and High risk).
5. Click **Publish** → **Publish**.

**Expected:** the Publish window shows no problems, and the version publishes as before (read-only, "Create new version" offered). Nothing new is shown.

### T-03: a sent HQ's result is stored (patient on a phone)

1. As **Cara**, **New episode** for a new made-up patient (e.g. "MT S41 <time> Smith", any date of birth, a made-up mobile **+447700900###** and email), on the HQ from T-02. Note the episode's id from the URL.
2. Open the captured messages page (**http://localhost:3000/qa/messages**) and open the invite link, at **390×844**. Continue, confirm the date of birth.
3. Answer About you: smoke **Yes**, chest pain **No**, heart failure **Yes**. Send the answers.
4. Back as Cara, open the episode.
5. Run the database query (setup) with this episode's id.

**Expected:**
- The patient sees the normal "sent" screen; the episode is **Ready for review**, with "HQ completed by the patient" in the General notes and **no** "automatic checks" line.
- One row: `evaluator_version` `clinical@…`, `error` empty; `asa` **III**; `asa_reasons` "Heart failure unspecified (I50.9) – …"; `rules_that_fire` "Rule fires: High risk (add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission)"; `flags` empty; `coded_disclosures` "Smoker, Heart failure unspecified"; `sets_shown` 2; `patient_sets` one entry with `"complete": true`.
- The episode page's "Suggested ASA III" line matches what was stored.

### T-04: the stored result matches the answers (a low-risk patient, sent on the patient's behalf)

1. As **Cara**, a second new episode on the same HQ (a new made-up patient).
2. On the episode page, **Fill in for the patient**: smoke **Yes**, chest pain **No**, heart failure **No**. Send.
3. Run the query for this episode.

**Expected:** the send works (Ready for review; "HQ completed on the patient's behalf"). The row: `asa` **II**, reason "Smoker (77176002) – …"; `rules_that_fire` "Rule fires: Low risk (auto-triage to Ready for admission)"; no error. **The episode is still Ready for review** (nothing acts on the rule yet: S42).

### T-05: a red flag is stored, and stops Low risk

1. A third episode on the same HQ, filled in for the patient: smoke **Yes**, chest pain **Yes**, heart failure **No**. Send.
2. Run the query.

**Expected:** `asa` **II**; `flags` "red: Do you get chest pain when you walk?: Yes"; `rules_that_fire` empty (Low risk needs no red flags; High risk needs ASA III). No error.

### T-06: an answer to the clinician's set never counts; a second send isn't evaluated twice

1. On the T-03 episode, Cara validates **Nurse checks** (answer anything).
2. Run the query for the T-03 episode again.

**Expected:** still exactly **one** row, unchanged (validating doesn't re-evaluate; only sending does).

### T-07: publish through the API is refused while a test case fails

1. As **Alex**, open the T-02 HQ and click **Create new version**.
2. **Rules** → **Test the whole HQ**: answer smoke **Yes** (others No) → What the rules see shows "Suggested ASA II" and Low risk. **Save as test case** "MT smoker".
3. Back on the Rules page, edit **Low risk**: change "is at most II" to **I**. Save. The test case now fails (red), and the Publish window lists "Rules · Test case “MT smoker” fails" (don't click Publish).
4. Call publish with `curl` as in the setup ("Calling the API as Alex"), with this HQ's ids.

**Expected:** `HTTP 422` with `"error":"This version can't be published until these problems are fixed."` and the problem `{"set":"Rules","message":"Test case “MT smoker” fails"}`. Reload the editor: the version is still a **draft**.

### T-08: the same version publishes once the case passes

1. Put Low risk back to **at most II** (the case passes again).
2. Call publish with `curl` again.

**Expected:** `HTTP 204`; the editor (reloaded) shows the version published.

### T-09: with the evaluate service stopped, sending still works (patient's view)

1. Stop **only** the evaluate service: `kill $(lsof -ti tcp:8090 -sTCP:LISTEN)`. Check `curl -s 127.0.0.1:8090/health` now fails, while http://localhost:3000 still works.
2. As **Cara**, a new episode on the T-08 HQ (new made-up patient). Open its invite link at **390×844**, answer anything, **send**.
3. As Cara, open the episode. Run the query for it.

**Expected:**
- The patient sees the normal **sent** screen (no error).
- The episode is **Ready for review**; its General notes have "HQ completed by the patient" **and** "The automatic checks didn't run. Review this HQ by hand.", shown as written by **Automatic** (like a reminder's line), not the patient or Cara.
- One row with `evaluator_version` empty, every result column empty, and `error` naming the connection (e.g. "…connection refused"); the error holds **no answer** of the patient.
- *If you can see the API's output* (the `make dev` terminal): one line containing `EVALUATE_FAILED form <id>: …`, with no answers in it. (In an automatic run you may not see it: then mark this one bullet as a hand check; an automated test covers it.)

### T-10: with the service stopped, filling in for the patient also still works

1. As **Cara**, another new episode; **Fill in for the patient**, answer, send.

**Expected:** as T-09: sent, Ready for review, the "automatic checks didn't run" line, an error row.

### T-11: with the service stopped, publishing is refused

1. As **Alex**, open any draft (e.g. create a new version of the T-08 HQ). Click **Publish** → **Publish** in the window.
2. Call publish with `curl` for it as well.

**Expected:** the window shows **"The checks couldn't run. Try again in a minute."**, and the draft is not published; `curl` gives `HTTP 503` with the same message.

### T-12: start it again; everything works as before

1. Start the evaluate service again in the background: `(make evaluate > /tmp/mt-s41-evaluate.log 2>&1 &)`, then `curl -s --retry 10 --retry-delay 1 --retry-connrefused 127.0.0.1:8090/health`.
2. Publish the T-11 draft from the Publish window.
3. Send one more HQ (any way); run the query.

**Expected:** health answers ok; the draft publishes; the new send stores a result row (no error) and no "automatic checks" line.

## Regression

- **R-01 (S40):** the Rules page, Test the whole HQ and the Publish window's "This version auto-triages: …" line work as before.
- **R-02 (S39):** the episode page's "Suggested ASA" line and the POA Summary's ASA section for the T-03 episode.
- **R-03 (S34):** a required "I have watched this video" tick still blocks sending until ticked (any HQ with a video, or skip if none is handy).
- **R-04 (S18–S22):** Cara validates every set of the T-04 episode and completes the review; it moves on as before.
- **R-05:** the Publish window still blocks on a logic problem (e.g. a condition testing a deleted question) before calling the server.

## Out of scope

- Acting on the result: auto-triage, the Auto-triaged worklist, tasks, flags or required ASA from rules (S42, S43). The episode staying **Ready for review** after a Low-risk send is expected.
- Refusing an incomplete send, or checking Complete review on the server (S44).
- The Lambda on AWS and its alarm (S45).
- Nothing on screen shows the stored result yet, apart from the "automatic checks didn't run" note.
