# S38 manual test plan: scores across Question Sets

*Retest after verification 2 (test report 5): F4 is fixed. Run **V-04 to V-06** below, then a quick pass of T-10, T-12, T-13 and R-03.*

*Retest after verification 1 (test report 4): F1–F3 are fixed. Run **V-01 to V-03** below, then a quick pass of T-02, T-05, T-10, T-15–T-17.*

*Retest after test report 2: its F-1 is fixed (a hidden earlier set's answers given in the Preview no longer count; the blue box says the sample patient wouldn't see that set). T-06c checks it.*

*Retest after test report 1: F-1 is fixed (the Preview's **Clear answers** keeps the earlier answers given in the blue box counting, and empties the summary box). T-06b checks it. Everything else is unchanged.*

For a tester who has never seen the code. Use only the browser. Report each case as Pass, Fail or Blocked, with what you saw.

**What's new:**
- A **score** (a Calculation that adds up option points) can use questions from **earlier Question Sets**, not only its own. Never later ones.
- A score can get **"Add N points when…" lines**, picked from menus: **Patient** (Age, Sex), **Measurements** (a BMI), **Other scores** (another score's band), and the answers to questions in this or an earlier Question Set. Nobody types a formula.
- A score with an input not answered shows **Incomplete**: it has no total and no band, so its band's disclosure doesn't appear. An input counts as not answered when it's empty, when a condition hides it, or when its Question Set is hidden.
- The age a score uses is the patient's age **on the day they sent their answers**.
- **STOP-Bang** is a **ready-made score**: inserted whole, with its published questions, points and bands, and locked.
- A Question Set's **Preview** can't see the earlier sets' answers: it says so, and lets you type them in to try the score.

## 1. Setup

1. In your own terminal (an automatic run does this for you):
   ```sh
   docker compose up -d db && make migrate seed && make dev
   ```
   Leave `make dev` running.
2. Open **http://localhost:3000** in Chrome. Sign in from the stub login page by clicking a user's name. Sign out from the avatar menu to switch user.
   - **Alex Author** (super clinician, the whole of Trust A): makes and publishes questionnaires.
   - **Cara Clinician** (clinician, Hospital A1): creates episodes, fills in for the patient, validates, opens the POA Summary.
3. **Captured messages:** http://localhost:3000/qa/messages lists the texts and emails the app "sent" (nothing is really sent). A patient's invite link is in their email there.
4. **How to make a scored question:** add a **Yes / No** question; in its Settings, under the options, switch on **Scores** and type a score beside each option (Yes **1**, No **0**).
5. **Preparation (as Alex Author, Hospital A1):** **Questionnaires** → new questionnaire **"S38 Scores HQ"** with two Question Sets, both for the **Patient**:
   - **Sleep**, with two pages:
     - page **Sleep habits**: Yes / No **"Do you snore loudly?"** and Yes / No **"Are you tired in the day?"**, both scored (Yes 1, No 0);
     - page **Size**: a **BMI** block (it must be alone on its page).
   - **Neck**, page **Collar**: Yes / No **"Is your collar large?"**, scored (Yes 1, No 0), and a **Clinical Summary** (the Add bar's **Clinical Summary** button).
   Don't publish yet.

## 2. Test cases

### Building a score in the editor

**T-01 The score's questions include earlier sets'**
1. In **Neck › Collar**, press **Calculation** on the page's Add bar. In its Settings, give it the question text **"Sleep risk"**.
2. **Calculate** says "A score total". Open **Add up the scores of**.

Expected: the list has a heading **"Sleep › Sleep habits"** with the two Sleep questions, and a heading **"This Question Set"** with "Is your collar large?". The BMI and unscored questions aren't there. Pick all three. The card says "Total of the scores of 3 questions (2 in earlier Question Sets)".

**T-02 Points lines from menus**
1. Under **Add points when…**, press **Add points line**. Open the second dropdown (placeholder "Select…").
2. Look at the menus, then pick **Age**, comparison **>**, value **50**.
3. Add two more lines: **Sex** **is** **Male**, and **BMI Calculator (BMI)** (under **Measurements**) **>** **35**.
4. Reload the page and select the Calculation again.

Expected: the dropdown is grouped **Patient** (Age, Sex), **Measurements** (BMI Calculator (BMI)), then "Sleep › Sleep habits" and "This Question Set › Collar" with the questions. No comparison says "is answered". Nothing asks for a formula. After the reload, the three lines are still there, each "Add 1 point when…". The card says "…, plus 3 points lines".

**T-03 An unfinished line isn't saved**
1. Press **Add points line**, pick **Age** but leave the value empty. Wait for "All changes saved", reload.

Expected: after the reload only the three finished lines are there.

**T-04 Bands and a disclosure**
1. In the Calculation's Settings, press **Add band** twice: **Low** up to **2**, **High** above.
2. **Disclosures** tab → **High band** → **Add** → **+ Add disclosure**: Note **"High risk of sleep apnoea"**, a Category (e.g. Respiratory) → **Save output** → **Done**.

Expected: the card shows the two bands, and High has one disclosure.

### The preview of one Question Set

**T-05 The preview says the earlier answers are missing, and lets you give them**
1. With **Neck** open in the editor, press **Preview** in the header. Switch to **Clinician**.
2. Read the blue box above the form. Answer "Is your collar large?" **Yes**. Look at **Clinical outputs → Scores**.
3. In the blue box, give "Do you snore loudly?" **Yes (1 pt)**, "Are you tired in the day?" **No (0 pts)** and the BMI **30**.
4. Change the Sample patient: Age **51**, Sex **Male**; then Age **50**.

Expected: the box says "Uses answers from earlier Question Sets, which count as unanswered here. To try them, give them below." At step 2 Scores says **"Sleep risk: Incomplete"**, and the outputs have no "High risk of sleep apnoea". After step 3, with the default sample patient (54, female): **"Sleep risk: 3 (High)"** (1 + 0 + 1 + 1 for age) and the disclosure shows. Age 51, male: **4 (High)**. Age 50, male: **3 (High)**: no point at 50. The Calculation in the form shows the same value with its band, e.g. "3 (High)".

**T-06 A test case keeps the earlier answers**
1. With the answers of T-05, name a test case "High, earlier answers" and press **Save answers**.
2. Press **Clear answers**, clear the blue box's answers, then **Load** the test case.
3. Press **Run all**.

Expected: Load puts back this set's answer and the blue box's answers; the score is back to its value. Run all: **All 1 passed**.

**T-06b Clear answers keeps the earlier answers (test report 1, F-1)**
1. In Neck's Preview (Clinician), give the blue box snore **Yes**, tired **No**, BMI **30**, and answer collar **Yes**: 3 (High), with the note in the Clinical outputs and in the summary box.
2. Press **Clear answers**, then answer collar **Yes** again without touching the blue box. Then change collar to **No**.

Expected: after Clear answers the score says Incomplete and the summary box has no note. The blue box still shows its answers, and they count: collar Yes gives **3 (High)** with the note back in both places; collar No gives **2 (Low)**, and the note goes. No `<div>`-inside-`<p>` warning appears in the console.

**T-06c A hidden earlier set's answers don't count in the Preview (test report 2, F-1)**
1. Give **Sleep** the condition **Patient's sex is Female** (in a draft). Open **Neck**'s Preview (Clinician), give the blue box snore **Yes**, tired **Yes**, BMI **36**, and answer collar **Yes**.
2. Set the Sample patient's Sex to **Male**, then back to **Female**.
3. With Male, save a test case; set Female; press **Run all**.

Expected: Female: a High score with the note. Male: the blue box says **"The sample patient wouldn't see “Sleep”, so its answers don't count."**, the score is **Incomplete** and the note goes; Female again brings both back. The male test case passes on Run all (it expected no note). No console error about a `key` prop.

### Publishing

**T-07 A score that uses a later Question Set is refused**
1. In the editor, move **Neck** above **Sleep** (the Structure panel's move up, or drag).
2. Look at the header's problem warning, then press **Publish**.
3. Move **Neck** back below **Sleep**.

Expected: at step 2 the problems say **"Sleep risk": it uses a question in a later Question Set (…)** naming the Sleep keys, and Publish is disabled. At step 3 the problem goes.

**T-08 A deleted question is refused**
1. In **Sleep**, delete "Are you tired in the day?".
2. Read the confirmation, then **Cancel**.

Expected: the "Delete this?" window lists **"Neck · "Sleep risk" (calculation)"** under "Other items depend on this". (If you delete it for real: the header and Publish say "Sleep risk": it tests something that no longer exists (…). Undo with Ctrl+Z / Cmd+Z.)

**T-09 It publishes**
1. With nothing wrong, **Publish**.

Expected: it publishes (v1 · published).

### Patients and clinicians

Create each episode as **Cara Clinician** (Hospital A1 → **New episode** → new patient), on **S38 Scores HQ**, with a made-up email. Fill the HQ in with **Fill in for the patient** on the episode page (or the invite link, T-14), then **Send**. Validate from the episode page's **Health questionnaire** card.

**T-10 A score from both sets' answers, with its band's disclosure on Validate and the POA Summary**
1. Patient **Sam S38a**, male, born **51 years ago** (e.g. 01/05/1975). Answers: snore **Yes**, tired **No**, height **170**, weight **70**; collar **Yes**. Send.
2. Validate **Neck**. Look at "Sleep risk" and the summary box.
3. Press **Validate**, then open the **POA Summary**.

Expected: Validate shows **"Sleep risk" 4 (High)** (1 + 0 + 1, age 1, male 1; BMI 24.2 adds nothing), and the summary box lists **"High risk of sleep apnoea"**. The POA Summary's Neck section lists it too.

**T-11 Age 50 gets no point**
1. Patient **Sam S38b**, **female**, born **exactly 50 years ago today** (e.g. on 26 Oct 2026: 26/10/1976). Answers: snore **Yes**, tired **No**, any height and weight with a BMI under 35, collar **No**. Send, validate Neck.

Expected: **1 (Low)** (snore only; 50 isn't over 50, female, BMI under 35), and no "High risk" note in the summary box or on the POA Summary.

**T-12 BMI over 35 gets the point; a clinician's correction counts**
1. Patient **Sam S38c**, female, born 40 years ago: snore **Yes**, tired **Yes**, height **160**, weight **100** (BMI 39.1), collar **No**. Send.
2. Validate **Neck**: note the score. Then Validate **Sleep**: change tired to **No** (the correction note appears), press Validate, and open Neck's Validate again.

Expected: step 2 first shows **3 (High)** (1 + 1 + BMI 1). After correcting Sleep, Neck shows **2 (Low)**, and the "High risk" note is gone from Neck's summary box.

**T-13 A hidden Question Set's answers don't count: the score is Incomplete**
1. As Alex: **Create new version** of S38 Scores HQ. Give **Sleep** a condition (Question Set view → Display this Question Set: **Conditionally**) **Patient's sex is Female**. Publish v2.
2. As Cara: patient **Sam S38d**, **male**, born 60 years ago, on v2 (the new episode picks the published version). Fill in Neck (collar **Yes**), send. Validate Neck.

Expected: Sleep isn't in the patient's form nor in the Health questionnaire list. Validate shows "Sleep risk" **Incomplete**, and the summary box has no "High risk of sleep apnoea" (it says no disclosures yet). The POA Summary has no such note.

**T-14 On a phone (390 × 844): the patient never sees the score**
1. Resize the window to 390 × 844 (DevTools device mode). Patient **Sam S38e** (any details): open their invite link from /qa/messages, Continue, date of birth.
2. Answer both sets.

Expected: the questions fit the screen with no sideways scroll; "Sleep risk", its value and "Incomplete" are never shown to the patient; they can send.

### STOP-Bang, ready-made

**T-15 Insert STOP-Bang**
1. As Alex, a new questionnaire **"S38 STOP-Bang HQ"**: Question Set **Measures** (Patient) with a page holding only a **BMI** block; Question Set **Sleep check** (Patient) with an empty page.
2. On Sleep check's page, press **Ready-made score** on the Add bar.
3. Read the window; look at **BMI from**; press **Insert**.

Expected: the window names STOP-Bang, its source (Chung et al., Chest 2016, the updated version) and its licence (University Health Network). **BMI from** has "BMI Calculator (BMI)" already chosen (it's the only one). Inserted: a Group **STOP-Bang** with a **Ready-made: STOP-Bang** badge and the licence line; five Yes / No questions (snoring, tiredness, observed stopped breathing, high blood pressure, a large neck with 17 in / 43 cm and 16 in / 41 cm), each "Yes 1 pt", and a Clinician **STOP-Bang score** with bands **Low risk**, **Intermediate risk**, **High risk**.

**T-16 Its parts are locked; disclosures and the Group's condition aren't**
1. Hover a STOP-Bang question's card; click it.
2. Look at its Settings tabs and fields. Try dragging it out of the Group.
3. Click the **STOP-Bang score** card and read its Settings.
4. On a question, **Disclosures** → **Yes** → add a disclosure; save.
5. Click the **STOP-Bang** Group's title: Logic → give it a condition (e.g. Patient's age ≥ 18).

Expected: a STOP-Bang question's card has no Copy, Move or Delete buttons and can't be dragged. Its Settings show a blue **"Ready-made: STOP-Bang"** note ("can't be changed… You can add disclosures."), every field is greyed out, and there is **no Logic** tab, but **Disclosures** works (step 4 saves). The score's Settings show the bands and **"Also High risk when: Yes to 2 or more of the 4 STOP questions, and male, a BMI over 35 or a large neck"**. The Group's Add bar has no buttons inside STOP-Bang; the Group can take a condition, and can be moved or deleted as a whole. Its Copy button isn't there.

**T-17 Only one STOP-Bang per HQ; with no BMI it can't be inserted**
1. Press **Ready-made score** again in this HQ and **Insert**.
2. In a new questionnaire with no BMI anywhere, press **Ready-made score**.

Expected: step 1: "This HQ already has STOP-Bang. A ready-made score can be in an HQ only once." Step 2: an orange note asks to add a BMI question first, and Insert is disabled.

**T-18 A changed STOP-Bang can't be published**
1. In S38 STOP-Bang HQ (still a draft), open DevTools → **Console**, paste the snippet below, press Enter, then reload the page.
2. Press **Publish**.
3. Delete the STOP-Bang Group, insert it again, Publish.

```js
(async () => {
  const m = location.pathname.match(/^\/o\/([^/]+)\/h\/([^/]+)\/questionnaires\/([^/]+)/);
  if (!m) return alert("Open the questionnaire in the editor first.");
  const api = `/api/o/${m[1]}/h/${m[2]}`;
  const { chapters } = await (await fetch(`${api}/questionnaires/${m[3]}`)).json();
  for (const c of chapters) {
    const ch = await (await fetch(`${api}/chapters/${c.id}`)).json();
    const g = (ch.content.pages ?? []).flatMap((p) => p.elements ?? []).find((e) => e.readyScore === "stopbang");
    if (!g) continue;
    g.elements[0].choices[0].score = 2;
    const res = await fetch(`${api}/chapters/${c.id}/content`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content: ch.content, revision: ch.revision }) });
    return alert(res.ok ? `Done: STOP-Bang's snoring Yes now scores 2 in “${c.name}”. Reload the page.` : `Refused: ${res.status}`);
  }
  alert("No STOP-Bang in this questionnaire.");
})();
```

Expected: step 2: the Publish window lists **"Sleep check · STOP-Bang has been changed: a ready-made score can't be changed. Delete it and insert it again."** and Publish is disabled. Step 3 publishes.

**T-19 STOP-Bang's results, as published**
1. Publish S38 STOP-Bang HQ (T-18). As Cara, episodes on it, each filled in for the patient and sent, then Validate **Sleep check** and read **STOP-Bang score**:
   - **a)** female, born 55 years ago, BMI 25 (e.g. 170 cm, 72 kg): Yes to tired and blood pressure only → **3 (Intermediate risk)** (2 STOP + age).
   - **b)** male, born 40 years ago, BMI 25: Yes to snoring and tired only → **3 (High risk)** (2 STOP + male).
   - **c)** female, born 40 years ago, BMI 36 (e.g. 170 cm, 104 kg): Yes to snoring and tired → **3 (High risk)**.
   - **d)** female, born 30 years ago, BMI 22: all No → **0 (Low risk)**.
   - **e)** sex **Not known**, born 40 years ago, BMI 25: snoring Yes, the rest No → **Incomplete** (the patient's sex is one of its inputs).

Expected: as listed.

### Reload, deep links, Back

**T-20 Reload and Back keep everything**
1. On Validate of T-10's Neck, reload; press Back to the episode, then Forward.
2. Open the Neck Preview's URL in a new tab.

Expected: "4 (High)" and the note come back after the reload and after Back/Forward. The Preview in a new tab shows the blue box again (its earlier answers start empty).

### After verification 1

**V-01 A removed line is saved while another is unfinished (F1)**
1. On a score with two finished lines (e.g. Age > 50 and Sex is Male), press **Add points line** and leave it empty.
2. Remove the **Sex** line. Wait for "All changes saved"; reload; select the score again.
3. Change the Age line's points to **2** while an empty line is open; reload.

Expected: after step 2 the card says "plus 1 points line" and, after the reload, only the Age line is there. After step 3 the Age line says 2 points after the reload.

**V-02 A key repeated in two Question Sets doesn't break the editor (F2)**
1. In a draft, give a question in the second set the same ID as a scored question in the first (the API or Code tab is the only way; or use the snippet below in the Console with the questionnaire open, then reload).
2. Select a score in the second set; open **Add up the scores of**, then a points line's dropdown.

```js
(async () => {
  const m = location.pathname.match(/^\/o\/([^/]+)\/h\/([^/]+)\/questionnaires\/([^/]+)/);
  if (!m) return alert("Open the questionnaire in the editor first.");
  const api = `/api/o/${m[1]}/h/${m[2]}`;
  const { chapters } = await (await fetch(`${api}/questionnaires/${m[3]}`)).json();
  if (chapters.length < 2) return alert("It needs two Question Sets.");
  const first = await (await fetch(`${api}/chapters/${chapters[0].id}`)).json();
  const q = (first.content.pages ?? []).flatMap((p) => p.elements ?? []).find((e) => (e.choices ?? []).some((c) => c.score !== undefined));
  if (!q) return alert("The first Question Set has no scored question.");
  const second = await (await fetch(`${api}/chapters/${chapters[1].id}`)).json();
  second.content.pages[0].elements.push({ ...q, title: `${q.title} (asked again)` });
  const res = await fetch(`${api}/chapters/${chapters[1].id}/content`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content: second.content, revision: second.revision }) });
  alert(res.ok ? `Done: “${q.title} (asked again)” in ${chapters[1].name} repeats ID ${q.name}. Reload the page.` : `Refused: ${res.status}`);
})();
```

Expected: no "Something went wrong" page; each list shows that ID once (the second set's "(asked again)" question). Publish still refuses the repeated ID.

**V-03 STOP-Bang refuses a repeated fixed key (F3)**
1. In a draft HQ without STOP-Bang but with a BMI block, run the snippet below in the Console (it gives the first Question Set's first question the ID `stopbang_snoring`), reload, then press **Ready-made score** → **Insert**.

```js
(async () => {
  const m = location.pathname.match(/^\/o\/([^/]+)\/h\/([^/]+)\/questionnaires\/([^/]+)/);
  if (!m) return alert("Open the questionnaire in the editor first.");
  const api = `/api/o/${m[1]}/h/${m[2]}`;
  const { chapters } = await (await fetch(`${api}/questionnaires/${m[3]}`)).json();
  const ch = await (await fetch(`${api}/chapters/${chapters[0].id}`)).json();
  const q = ch.content.pages[0].elements[0];
  q.name = "stopbang_snoring";
  const res = await fetch(`${api}/chapters/${chapters[0].id}/content`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content: ch.content, revision: ch.revision }) });
  alert(res.ok ? `Done: “${q.title}” now has the ID stopbang_snoring. Reload the page.` : `Refused: ${res.status}`);
})();
```

Expected: it's refused: "This HQ already has a question with the ID stopbang_snoring…".

**V-04 A set shown by a score follows a changed earlier answer (verification 2, F4)**
1. As Alex, a new questionnaire "S38 Follow-up HQ", three Question Sets for the Patient:
   - **Snoring**: Yes / No "Do you snore loudly?", scored (Yes 1, No 0);
   - **Tiredness**: Yes / No "Are you tired in the day?", and a Calculation **"Snoring score"** adding up "Do you snore loudly?", with its **Clinical** switch **off** (so patients have it too);
   - **Sleep follow-up**: Yes / No "Has anyone seen you stop breathing?"; Question Set condition **Snoring score ≥ 1** (Display this Question Set: Conditionally).
   Publish.
2. As Cara, an episode on it; open the patient's invite link (/qa/messages), Continue, date of birth.
3. Snoring: **Yes**; finish. Tiredness: **Yes**; finish. Look at the list.
4. Open Snoring again (Review) and change to **No**; finish. Look at the list.
5. Change Snoring back to **Yes**.

Expected: step 3: **Sleep follow-up** is listed, and Send stays disabled until it's answered. Step 4: Sleep follow-up **goes** from the list (though Tiredness wasn't opened again), and Send is enabled. Step 5: it's back, and Send is disabled again.

**V-05 The clinician sees the same sets**
1. Send V-04's answers with Snoring **No** (the follow-up gone). As Cara, open the episode.

Expected: the Health questionnaire card lists Snoring and Tiredness only; the POA Summary has no Sleep follow-up section.

**V-06 Deleting STOP-Bang warns about scores that use its questions**
1. In a draft with STOP-Bang in set 2, add a score in set 3 that adds up a STOP-Bang question. Delete the STOP-Bang Group (Cancel at the confirmation).

Expected: "Other items depend on this" lists the set 3 score.

## 3. Regression

**R-01 A score inside one Question Set, as before:** in any draft, a Calculation "A score total" of two scored questions in the same set, with bands, works in the Preview (e.g. both Yes → "2 (…)"); with only one answered it shows **Incomplete** (new rule).

**R-02 An older score formula still adds what's answered:** a Calculation set to **A custom formula** `score('<key 1>', '<key 2>')` (the two questions' IDs, from their Settings) shows the total of what's answered (e.g. one Yes → 1), never Incomplete.

**R-03 Question Set conditions (S08/LOG-02):** a set shown only for "Patient's sex is Female" still hides for a male patient, and its Preview note ("The sample patient doesn't meet it…") is unchanged.

**R-04 Clinical Summaries (S32/S37):** a disclosure from an ordinary question's option still shows in its summary box on Validate and in its POA Summary block; a summary still has no Logic tab.

**R-05 Pre-fill (S25):** a second episode for the same patient on the same HQ still opens with "Please check" and the pre-filled answers.

## 4. Out of scope

- The suggested ASA from scores, rules, auto-triage and the server's evaluate service (S39–S44). A score's Incomplete state isn't acted on yet.
- Other ready-made scores (Apfel and the rest: plan C7); observations as a BMI source (C7); using a score from another questionnaire or episode.
- An "also when" condition on bands for authors' own scores (only STOP-Bang has one).
- The exact wording of STOP-Bang against the licensed form (a clinician checks it before go-live).
