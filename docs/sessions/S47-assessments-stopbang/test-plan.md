# S47 test plan: assessments, starting with STOP-Bang

**What's new:** a clinician can add **STOP-Bang** (a sleep apnoea screen with 8 Yes/No items: Snoring, Tired, Observed, Pressure, BMI over 35, Age over 50, Neck, Gender male) to an episode, from the episode page's new **Assessments** card. It opens **pre-filled**:
- S, T, O and P (and the neck, if asked) from the patient's HQ, once they've sent it;
- B from the **hospital BMI** (the Observations card) or the patient's own BMI from the HQ, as the trust's setting **"Observations take precedence in scores"** says (on by default: the hospital BMI first);
- A and G from the patient's record (their age and sex).

Each item shows where its value came from (**HQ**, **Hospital BMI** / **Patient reported BMI**, **Profile**, **Clinician**), and a missing item is highlighted (orange, "Missing"). The clinician answers what's missing or changes any item, then **Confirm**s, which records who and when. Changing an item after Confirm needs a new Confirm. The **POA Summary** shows the score and band (e.g. "6 (High risk)") beside the hospital BMI, or "Not confirmed". Patients never see assessments.

## Setup

In Rahul's own terminal, from the repo root:

```sh
docker compose up -d db
make migrate seed
make dev
```

Leave `make dev` running. Open **http://localhost:3000**. The sign-in page lists the seeded users; click one to sign in as them (Sign out, top right, to switch).

| User | Who |
| --- | --- |
| **Cara Clinician** | clinician, Hospital A1 only |
| **Val Clinician** | clinician, Hospital A2 only (same trust) |
| **Bea Author** | super clinician, the whole of Trust B |
| **Ada Admin** | admin, the whole of Trust A (Settings, Audit) |
| **Sue Super** | super clinician, Hospital A1 only (can archive) |
| **Sam Superadmin** | platform admin (the console at `/admin`) |

**Two seeded HQs for this session** (Trust A, both made up):
- **Demo sleep HQ**: one Question Set, "Sleep": the patient's height and weight, then four of STOP-Bang's questions (snoring, tired, observed, blood pressure). The assessment opens with 7 of 8 filled: the neck is missing.
- **Demo STOP-Bang HQ**: "Sleep": height and weight, then the whole ready-made STOP-Bang (neck included). The assessment opens with all 8 filled, waiting for Confirm.

**Making an episode:** as Cara, Hospital A1 → **Episodes** → **New episode**: first name, surname (e.g. `MT S47 <time>`), date of birth, sex, mobile `07700 900` + 3 digits, an email like `mt-s47-<time>@example.com`, the HQ named in the test, **Create**. The episode page opens.

**The patient's answers:** open **http://localhost:3000/qa/messages** (no sign-in needed locally), open the newest invite to the episode's email **at phone size (390×844)**, type the date of birth, answer the Sleep set and **send** the answers. (Or, as Cara on the episode page, **Fill in for the patient**, answer, and send.) The assessment uses the patient's answers only once they're sent.

On the episode page the **Assessments** card is below **Observations**. To record the hospital BMI, use the Observations card: **Add a reading** → Height → e.g. 180 → **Save reading**, then Weight.

## Test cases

### T-01 STOP-Bang opens with 7 of 8 filled, each with its source (focus 1)
1. As Cara, make an episode on **Demo sleep HQ** for a made-up **man born 1960** (age 66).
2. As the patient: height 170, weight 82; snoring **Yes**, tired **Yes**, observed **No**, blood pressure **Yes**; send.
3. As Cara, reload the episode page. The Assessments card says "No assessments yet." Record height **180** and weight **120** (hospital BMI 37).
4. **Add assessment** → **STOP-Bang**.

**Expected:** a STOP-Bang card: "7 of 8 filled: answer the item highlighted." Rows S, T, O, P show Yes, Yes, No, Yes with source **HQ**; B shows **37** in a box, source **Hospital BMI**; A shows **66**, **Profile**; G shows **Male**, **Profile**; **N** is highlighted (orange) with source **Missing**. The result badge says **Incomplete**; **Confirm** is greyed out; it says "Not confirmed". The **Add assessment** menu is gone (one STOP-Bang per episode).

### T-02 Confirm: who and when, and the POA Summary (focus 3)
1. Continue T-01. On N click **No**.
2. Click **Confirm**.
3. Reload. Open **POA Summary**.

**Expected:** after step 1, N's source is **Clinician** and the badge says **6 (High risk)**. After Confirm: "Confirmed by Cara Clinician on <date and time>", and Confirm is greyed out. Reload keeps it all. The POA Summary's details list **Hospital BMI 37 (measured …)** and right under it **STOP-Bang 6 (High risk)**, on both tabs.

### T-03 A change needs a new Confirm (focus 3)
1. Continue T-02. Back on the episode page, click **Yes** on N.
2. Click **Undo changes**. Then click **Yes** on N again and **Save**.
3. Open the POA Summary. Then back to the episode page and **Confirm**.

**Expected:** step 1: "Changed: confirm it again", badge **7 (High risk)**, Confirm active. Undo puts back No and "Confirmed by…". After Save: "Not confirmed"; the POA Summary says **STOP-Bang Not confirmed**. After Confirm: "Confirmed by Cara Clinician…", and the POA Summary says **7 (High risk)**.

### T-04 Which BMI: the trust's setting (focus 2)
1. As Cara, make an episode on **Demo sleep HQ** (any adult). As the patient: height 170, weight **82** (BMI 28.4, under 35), all four questions answered; send.
2. As Cara: record height **180**, weight **120** (hospital BMI 37, over 35). **Add assessment** → STOP-Bang. Don't confirm.
3. As **Ada Admin**, Hospital A1 → **Settings** → under **Observations**, turn **Observations take precedence in scores** off → save.
4. As Cara, reload the episode page.
5. As Ada, turn the setting back **on**, save.

**Expected:** step 2: B shows 37, **Hospital BMI**. Step 3: the setting is listed under Observations, with its help text. Step 4: B shows **28.4**, **Patient reported BMI**, and the badge's score is one lower. Step 5 (reload as Cara): back to 37, Hospital BMI. *Note:* a confirmed assessment keeps what was confirmed; the setting changes only one not confirmed yet.

### T-05 An HQ holding STOP-Bang opens complete, waiting for Confirm (focus 5)
1. As Cara, make an episode on **Demo STOP-Bang HQ**. As the patient: height and weight, every STOP-Bang question; send.
2. As Cara, **Add assessment** → STOP-Bang (no readings recorded yet).
3. Open the POA Summary, then go back and **Confirm**.

**Expected:** "8 of 8 filled." All five Yes/No items say **HQ**, B says **Patient reported BMI**, A and G **Profile**; nothing highlighted; a score and band show; it says **Not confirmed** and Confirm is active. The POA Summary says **STOP-Bang Not confirmed**. After Confirm, "Confirmed by Cara Clinician…" and the POA Summary shows the score and band.

### T-06 Before the patient sends, the HQ gives nothing
1. Make an episode on Demo sleep HQ. As the patient, answer the four questions but **don't send**.
2. As Cara, add STOP-Bang.
3. As the patient, send. As Cara, reload.

**Expected:** step 2: S, T, O, P and B are **Missing** (highlighted): "2 of 8 filled" (A and G), or 3 of 8 if you recorded a hospital BMI. Step 3: S, T, O and P come from the **HQ** (and B from the patient's BMI, if no hospital BMI).

### T-07 The clinician changes a pre-filled item
1. On an unconfirmed STOP-Bang from the HQ (e.g. T-04's), click **No** on S (the patient said Yes), then **Save**.
2. Reload.

**Expected:** S says **Clinician**, the score one lower. After reload S is still No / Clinician; T, O and P still say HQ.

### T-08 A BMI typed by the clinician; a bad one
1. Make an episode on Demo sleep HQ; as the patient **skip** height and weight, answer the four questions, send. Record no readings. Add STOP-Bang.
2. In B's box type **abc**. Then **36.5**.
3. Type **2**, then **151**. Then **36,5** and click **Save**.

**Expected:** step 1: B is **Missing**. Step 2: "abc" shows "A number, one decimal place" under the box, and Save and Confirm are greyed out; 36.5 makes B **Entered by clinician** and scores the point. Step 3: 2 and 151 show "Between 5 and 150" under the box, with Save and Confirm greyed out (report 1), and B shows **Missing**, so the count and score don't use the value typed before (report 2); 36,5 saves as 36.5. (Go refuses a value outside the limits too, with "BMI: a number from 5 to 150.", seen only by calling the API directly.)

### T-09 Sex Other or Not known
1. Make an episode on Demo sleep HQ for a patient whose sex is **Other** (or **Not known**). Answer and send; add STOP-Bang.

**Expected:** G is **Missing** (highlighted), a note says the record doesn't say male or female so the score stays Incomplete; the badge says **Incomplete** and Confirm stays greyed out even with every other item answered.

### T-10 Two tabs
1. Open one episode's page in two tabs. In tab 1, add STOP-Bang. In tab 2 (not reloaded), **Add assessment** → STOP-Bang.
2. Reload tab 2. In tab 1, answer N and **Save**. In tab 2 (not reloaded), answer N differently and **Save**.

**Expected:** step 1, tab 2: "STOP-Bang is already on this episode." shows in red at the top of the Assessments card and **stays** (it has a close button), and the card shows the one STOP-Bang (F-1, report 1). Step 2, tab 2: "These answers were changed somewhere else since you opened them…" with **Reload**; Reload shows tab 1's answer.

### T-11 The patient never sees an assessment (phone size)
1. With an episode from T-01 or T-05 (STOP-Bang confirmed), open the patient's link at **390×844**, and their home.

**Expected:** only the HQ's Question Set(s); no STOP-Bang assessment, no "Assessments", no clinician's answers, anywhere.

### T-12 Other hospitals and other trusts
1. Copy the URL of an A1 episode with STOP-Bang (and of its POA Summary).
2. Sign in as **Val Clinician** (Hospital A2) and open both URLs (with A1's id in them). Then as **Bea Author** (Trust B).

**Expected:** not found / no access for both, as for any other trust's or hospital's episode; nothing of the assessment shows.

### T-13 Cancelled and archived episodes are read-only
1. Cancel an episode that has a STOP-Bang (Episode status → Cancel, with a reason). Open it.
2. As **Sue Super**, archive another one that has STOP-Bang (from a status it can be archived from). Open it.

**Expected:** the STOP-Bang card shows its items and Confirm state, with no Yes/No buttons, no BMI box, no Save or Confirm, and no Add assessment menu. The count says only "N of 8 filled.", with no "answer the items highlighted" (report 1).

### T-14 Assessments switched off
1. As **Sam Superadmin**, `/admin` → Trust A → turn **Assessments** off for **Hospital A1** → save.
2. As Cara, open an A1 episode that has STOP-Bang, and its POA Summary.
3. Turn it back on.

**Expected:** step 2: no Assessments card; no STOP-Bang line on the POA Summary; everything else unchanged. Step 3: the card and its STOP-Bang are back, as they were.

### T-15 The Audit log
1. As **Ada Admin**, Trust A → **Audit**, filter to today and to one episode used above (or to Cara).

**Expected:** rows "Opened an episode's assessments", "Added an assessment", "Saved an assessment's answers" and "Confirmed an assessment", each naming Cara and the episode.

### T-16 Reload, deep links and Back
1. On an episode page with STOP-Bang, make a change (don't save) and reload. Open the POA Summary, press the browser's **Back**, then **Forward**.
2. Paste an episode page URL into a new tab.

**Expected:** an unsaved change is dropped on reload (the saved state shows). Back returns to the episode page with the assessment as saved; Forward returns to the POA Summary. The pasted URL opens the episode page with its Assessments card.

### T-17 Retest of report 1's exploring notes
1. On any STOP-Bang at a desktop width (about 1456 wide), look at the source badges, especially **Patient reported BMI** and **Entered by clinician**.
2. On an unconfirmed STOP-Bang whose S came from the HQ, click S's other answer, then the HQ's answer again.
3. On a confirmed one, do the same with any item.
4. In B's box, type a valid BMI, then empty the box.

**Expected:** step 1: every badge shows its words whole, in normal case, and the Yes/No buttons and the BMI box stay put as a source changes (report 2). Step 2: S goes to **Clinician**, then back to **HQ**; Save and Undo changes go away again. Step 3: "Changed: confirm it again" shows, then goes back to "Confirmed by …". Step 4: "Enter a BMI" under the box, B **Missing**, with Save and Confirm greyed out; **Undo changes** brings the value back.

### T-18 Retest of verification 1
1. On an episode with the Demo STOP-Bang HQ sent (8 of 8 filled), open DevTools → Network, block the request URL pattern `*/observations`, and reload.
2. Unblock it and reload.
3. By API (or DevTools' console, `fetch`), POST `…/assessments/{fid}/confirm` with `{"data": {}, "sources": {}, "revision": <current>}`.

**Expected:** step 1: the card still shows its items, with "The observations didn't load, so the hospital BMI can't be checked. Reload the page to save or confirm." in red, and Save and Confirm greyed out. Step 2: the message has gone, and Confirm works. Step 3: 400 "Answer every item before confirming: BMI has no answer.", and nothing changes. (Finding 1, the age on a re-confirm, can't be set up by hand: it needs a Confirm on an earlier day. It's covered by a unit test.)

## Retest after verification 3 (Rahul's answers 5–7, 6 Nov)

**Restart `make dev` first** (Ctrl-C, then `make dev`): Go changed. Then run T-19 to T-21, plus T-03, T-07 and T-02's POA Summary as regression.

### T-19 The pre-fill uses the clinician's started copy (answer 5)
1. Make an episode with the **Demo STOP-Bang HQ** (not the Demo sleep HQ: its episodes are auto-triaged by the default Low risk rule, which completes the review); the patient sends S, T and P Yes, O and N No (a High STOP-Bang, so it stays at Ready for review).
2. As Cara, open the Sleep set (**Validate**) and change Tired to **No**. Don't press Validate. Go back to the episode page.
3. **Add assessment → STOP-Bang**.
4. Then validate the set, and reload the episode page.

**Expected:** step 3: T is **No**, source **HQ** (the clinician's copy counts once started, like the Question Sets card and the suggested ASA). Step 4: T is still No.

### T-20 A confirmed STOP-Bang says what's changed since (answer 6)
1. Continue T-19. Record height 180 and weight 120 (hospital BMI 37), answer N **No** and **Confirm**: **5 (High risk)**.
2. On the Observations card, **Correct** the weight to 90, with a reason.
3. Reload. Open the POA Summary.
4. Back on the episode page, in B's box type 27.8 and **Confirm**.

**Expected:**
- Step 1: no yellow box.
- Step 2: a yellow box on the card: **"The hospital BMI is now 27.8 (37 when confirmed)."** and "The score stays as confirmed…". B still shows 37 / Hospital BMI, the score is still **5 (High risk)**, and it still says "Confirmed by Cara Clinician…".
- Step 3: reload keeps all that, and the POA Summary says **STOP-Bang 5 (High risk)**.
- Step 4: B says **Entered by clinician**, the score is **4 (High risk)**, and the yellow box has gone.

### T-21 Age and sex are frozen at Confirm (answer 7)
There's no screen that changes a patient's date of birth or sex, so step 2 changes them in the database (made-up data only).
1. Confirm a STOP-Bang for a man over 50, e.g. T-02's (6 (High risk), A and G scoring). Note his surname.
2. In a terminal at the repo root: `docker compose exec db psql -U sj -c "UPDATE patients SET date_of_birth = '1990-03-01', sex = 'female' WHERE last_name = '<surname>'"`
3. Reload the episode page. Open the POA Summary.
4. Back on the episode page, click N's other answer, then **Undo changes**.

**Expected:**
- Step 3: A still shows his age when confirmed and G **Male**, both Profile. The score is unchanged (6 (High risk)). The yellow box says **"The patient's record now gives an age of 36 that day (66 when confirmed)."** and **"The patient's record now says Female (Male when confirmed)."** The POA Summary is unchanged.
- Step 4: once changed, A and G follow the record (36, Female) and the score drops; Undo changes puts back the confirmed values.

### Regression for this round
- **T-03** (a change needs a new Confirm), **T-07** (the clinician changes a pre-filled item), and **T-02's POA Summary** line, as written.

## Retest after test report 4 (F-1, 6 Nov)

No restart needed (a web change only; Vite reloads it). Run T-21 again in full, with step 4 extended:

4. Back on the episode page, click N's other answer. Check A, G and the score. Then **Confirm**, and reload.

**Expected, step 4:**
- once N is changed: A **36** and G **Female** (Profile), and the score is worked out with them (from T-02's 6 with N Yes → No: S, T, P and B only, **4 (High risk)**);
- the note under the buttons says "The age is the patient's today.";
- after Confirm and the reload: still 36 / Female and that score, "Confirmed by…", and **no yellow box**; the POA Summary shows the new score.

Before step 4 (after step 3), the yellow box's hint also says "Once any item is changed, the age and sex are the record's."

Regression: T-03 and T-07 as written.

## Regression

- **R-01 (S46) Observations:** add a weight, correct it with a reason: the old reading is struck through and the hospital BMI follows; the POA Summary's Hospital BMI matches.
- **R-02 (S20, S44) Validate and Complete review:** on a sent Demo pre-op HQ episode (Ready for review), validate each set and **Complete HQ review**: the Health questionnaire card shows "Validated by…" and the status moves on.
- **R-03 (PAT-09) Fill in for the patient:** on a new episode, **Fill in for the patient**, answer and send: the episode moves on, and "Sent by Cara Clinician on the patient's behalf…" shows.
- **R-04 (S38) The ready-made STOP-Bang in the editor:** as **Alex Author**, make a new HQ, add a BMI question, then **Ready-made score → STOP-Bang**: it's inserted, its cards can't be edited, and the HQ publishes.
- **R-05 (S46) Settings:** as Ada, the Observations group still has the ranges table; change one limit and save, then put it back to the default.

## Out of scope

- Other assessments (DASI, PRISMA-7, Apfel: S48; MUST, falls, VTE, Rockwood: S49). The Add assessment menu lists only STOP-Bang.
- Assessments added automatically, or per procedure (later, S47 answers 1).
- Assessments feeding the HQ's rules or auto-triage; assessments on POA documents (C9).
- A neck measurement in cm (the neck is Yes/No, S47 answers 2).
- The wording, scoring and bands of STOP-Bang itself (checked against its published source in S38).
