# S48 test plan: DASI, PRISMA-7 and Apfel

**What's new:** three more assessments beside STOP-Bang, each also a **ready-made score** an author can insert in an HQ:
- **DASI** (fitness): 12 Yes/No questions ("Can you…?"), each Yes worth its own points (0–58.2). The result is the score, a band **"Poor functional capacity"** at **34 or less** (no band above 34), and the **METs** (an estimate of fitness), e.g. "34 (Poor functional capacity), 6.9 METs".
- **PRISMA-7** (frailty): 7 Yes/No items, 1 point per Yes. Items 1 and 2 (older than 85, male) come from the patient's record; the patient answers 3–7. **3 or more** is "Increased risk of frailty"; below is "Not at increased risk".
- **Apfel PONV** (sickness after an anaesthetic): 4 items, 1 point each: female (record), non-smoker, had PONV or travel sickness before, opioids after surgery. In an HQ it's **only the patient's two questions, with no score**. On the episode, the clinician answers the 4th item (opioids). The result names the published risk, e.g. "2 (PONV risk about 39%)".

On the episode page's **Assessments** card, **Add assessment** now offers STOP-Bang, DASI, PRISMA-7 and Apfel PONV. Each opens pre-filled from the HQ's answers (once sent) and the patient's record, each item with its source; the clinician completes and **Confirm**s. The POA Summary shows each one's result.

## Setup

In Rahul's own terminal, from the repo root:

```sh
docker compose up -d db
make migrate seed
make dev
```

**Restart `make dev` if it's already running** (the Go API, the evaluate service and the web app all changed). Leave it running. Open **http://localhost:3000**. The sign-in page lists the seeded users; click one to sign in (Sign out, top right, to switch).

| User | Who |
| --- | --- |
| **Cara Clinician** | clinician, Hospital A1 only |
| **Val Clinician** | clinician, Hospital A2 only (same trust) |
| **Alex Author** | super clinician, the whole of Trust A (writes questionnaires) |
| **Bea Author** | super clinician, the whole of Trust B |

**The seeded HQ for this session:** **Demo fitness HQ** (Trust A, made up): one Question Set, **"Fitness and recovery"**, with three pages: the ready-made **DASI** (12 questions), **PRISMA-7** (questions 3–7) and **Apfel**'s two questions (smoking; sickness after an anaesthetic or travel sickness).

**Making an episode:** as Cara, Hospital A1 → **Episodes** → **New episode**: first name, surname (e.g. `MT S48 <time>`), the date of birth and sex the test names, mobile `07700 900` + 3 digits, an email like `mt-s48-<time>@example.com`, HQ **Demo fitness HQ**, **Create**. The episode page opens.

**The patient's answers:** open **http://localhost:3000/qa/messages** (no sign-in needed locally), open the newest invite to the episode's email **at phone size (390×844)**, type the date of birth, answer the set and **send** the answers. (Or, as Cara on the episode page, **Fill in for the patient**, answer, and send.) The assessments use the patient's answers only once they're sent.

**Ages:** today is 6 Nov 2026. Born **1 Jan 1941** is 85; born **1 Jan 1940** is 86; born **1 Jan 1980** is 46.

## Test cases

### The patient's side (focus 2)

#### T-01 The patient answers all three on a phone
1. As Cara, make an episode on **Demo fitness HQ** for a made-up **woman born 1 Jan 1980**.
2. Open the invite at **390×844**, type the date of birth, **Start** "Fitness and recovery".
3. Page 1 (DASI): answer **Yes** to questions 1, 2, 4, 5, 7, 8 and 9 (self-care, walk indoors, stairs or a hill, run, moderate housework, heavy housework, yardwork), **No** to the rest. **Next**.
4. Page 2 (PRISMA-7): **No** to all but "If you need help, can you count on someone close to you?" (**Yes**). **Next**.
5. Page 3: "Do you smoke?" **No**; "Have you ever felt or been sick after an anaesthetic…" **Yes**. **Finish this section**, then **Send my answers** and confirm.

**Expected:** every page fits the screen with no sideways scrolling. The patient sees only the questions: no "DASI score", no METs, no PRISMA-7 score, nothing about opioids. Each DASI question is required (Next asks for an answer first). The send says "Thank you".

### The assessments (focus 1–3)

#### T-02 DASI opens complete, and its METs reach the POA Summary
1. Continue T-01. As Cara, reload the episode page. **Add assessment** → **DASI**.
2. Read the card. Click **Confirm**. Reload.
3. Open **POA Summary**.

**Expected:** a DASI card: "12 of 12 filled." Rows numbered **1** to **12**, each with a short label (Self-care, Walk indoors… Strenuous sports), the question's wording, Yes/No buttons (Yes on 1, 2, 4, 5, 7, 8, 9) and source **HQ**. The badge reads **34 (Poor functional capacity), 6.9 METs**, all of it visible. No age or sex note under the card. After Confirm: "Confirmed by Cara Clinician on …", kept after the reload. The POA Summary, under Hospital BMI, lists **DASI 34 (Poor functional capacity), 6.9 METs** on both tabs.

#### T-03 PRISMA-7 opens 7 of 7: items 1 and 2 from the record
1. Continue. **Add assessment** → **PRISMA-7**.

**Expected:** "7 of 7 filled." Rows: **1** Older than 85 → **46**, source **Profile**; **2** Male → **Female**, **Profile**; **3–7** from the **HQ** (No, No, No, **Yes** on 6, No). The badge reads **1 (Not at increased risk)** (question 6's Yes scores, as published). A note says "The age is the patient's today."

#### T-04 Apfel opens 3 of 4, waiting only for the opioids
1. Continue. **Add assessment** → **Apfel PONV**.
2. Click **Yes** on item 4. Then **Confirm**.
3. Open the POA Summary.

**Expected:** step 1: "3 of 4 filled: answer the item highlighted." **1** Female → Female, **Profile**; **2** Smoking → No, **HQ**; **3** PONV or travel sickness before → Yes, **HQ**; **4** Opioids after surgery is highlighted (orange), **Missing**. The badge says **Incomplete** and **Confirm** is greyed out. Step 2: item 4's source is **Clinician**, the badge **4 (PONV risk about 79%)**, then "Confirmed by Cara Clinician…". The POA Summary lists **Apfel PONV 4 (PONV risk about 79%)**, and **PRISMA-7 Not confirmed** (T-03 wasn't confirmed). The **Add assessment** menu now offers only **STOP-Bang**; once that's added too, the menu is gone.

#### T-05 DASI's worked examples, by hand (focus 1 and 3)
1. As Cara, make an episode on **Demo fitness HQ** (anyone) and **don't** send the HQ. **Add assessment** → **DASI**: "0 of 12 filled", all 12 highlighted, badge **Incomplete**, Confirm greyed out.
2. Click the Yes/No buttons for each row below, reading the badge after each (No to every row not listed):

| Yes to | Expected badge |
| --- | --- |
| none (No to all 12) | **0 (Poor functional capacity), 2.7 METs** |
| 1 and 2 | **4.5 (Poor functional capacity), 3.3 METs** |
| 1, 2, 3, 5, 6, 7, 8, 9 | **33.95 (Poor functional capacity), 6.9 METs** |
| 1, 2, 4, 5, 7, 8, 9 | **34 (Poor functional capacity), 6.9 METs** (the cut-off: 34 or less is Poor) |
| 1, 3, 4, 5, 6, 8, 9 | **34.2, 6.9 METs** (over 34: no band) |
| all 12 | **58.2, 9.9 METs** |

**Expected:** as in the table. Each changed row's source is **Clinician**. Confirm is enabled once all 12 are answered.

#### T-06 PRISMA-7's edges: older than 85, male, 3 points (focus 3)
Make an episode on **Demo fitness HQ** for each patient below, don't send the HQ, **Add assessment** → **PRISMA-7**, then answer items 3–7 as listed (each changed row becomes Clinician):

| Patient | Items 3–7 | Expected |
| --- | --- | --- |
| a) woman born 1 Jan 1941 (85) | Yes to 3 and 4 | item 1 shows **85**, no point; **2 (Not at increased risk)** |
| b) woman born 1 Jan 1940 (86) | Yes to 3 and 4 | item 1 shows **86**, a point; **3 (Increased risk of frailty)** |
| c) man born 1 Jan 1980 (46) | Yes to 3 and 4 | item 2 **Male**, a point; **3 (Increased risk of frailty)** |
| d) woman born 1 Jan 1980 | Yes to 3, 4 and 5 | **3 (Increased risk of frailty)** |
| e) woman born 1 Jan 1980 | Yes to 3 only | **1 (Not at increased risk)** |

#### T-07 Apfel's worked examples (focus 1)
Episodes on **Demo fitness HQ**, HQ not sent, **Add assessment** → **Apfel PONV**, then answer items 2–4:

| Patient | Smoking (2) | Sickness before (3) | Opioids (4) | Expected |
| --- | --- | --- | --- | --- |
| a) man | Yes | No | No | **0 (PONV risk about 10%)** |
| b) woman | Yes | No | No | **1 (PONV risk about 21%)** |
| c) man | No | No | No | **1 (PONV risk about 21%)** (a non-smoker's point) |
| d) woman | No | No | No | **2 (PONV risk about 39%)** |
| e) man | No | Yes | Yes | **3 (PONV risk about 61%)** |
| f) woman | No | Yes | Yes | **4 (PONV risk about 79%)** |

#### T-08 A sex of Other leaves the sex item missing
1. Make an episode on **Demo fitness HQ** for a patient whose sex is **Other** (or use the seeded **Robin Sample**). Don't send.
2. Add **PRISMA-7** and **Apfel PONV**; answer every Yes/No item.
3. Add **DASI**.

**Expected:** PRISMA-7's item 2 and Apfel's item 1 are highlighted, **Missing**; both badges stay **Incomplete**, Confirm greyed out, each with the note "The patient's record doesn't say male or female, so the score stays Incomplete, as in the HQ." DASI has no such note and gives its score once its 12 are answered.

#### T-09 A change after Confirm needs a new Confirm; reload and back
1. On T-02's episode, change DASI's row 10 to **Yes**.
2. **Undo changes**. Change it to **Yes** again, **Save**. Reload.
3. **Confirm**. Open the POA Summary, then the browser's **Back**.

**Expected:** step 1: "Changed: confirm it again"; the badge **39.25, 7.6 METs** (no band). Undo puts back 34 and "Confirmed by…". After Save and reload: "Not confirmed", row 10 **Clinician**, the POA Summary would say Not confirmed. After Confirm the POA Summary says **DASI 39.25, 7.6 METs**; Back returns to the episode page with the card as it was.

### The HQ editor (focus 4)

#### T-10 Insert DASI, PRISMA-7 and Apfel as ready-made scores
1. As **Alex Author**, Trust A → Questionnaires → a new questionnaire **"MT S48 ready <time>"** with one Patient Question Set and an empty page.
2. Press **Ready-made score** on the Add bar. Open the **Score** list.
3. Choose **DASI**, read the window, **Insert**. Do the same for **PRISMA-7** and **Apfel PONV** (each on the page).
4. Press Ready-made score → **DASI** → Insert again.

**Expected:** the list offers **STOP-Bang, DASI, PRISMA-7, Apfel PONV**. Each shows its source and licence line (DASI: Hlatky 1989 and Wijeysundera 2020, "Licensed by Greenlight Health…"; PRISMA-7: Raîche 2008, "© CESS…"; Apfel: Apfel 1999, "A published risk score: no licence found."). Only STOP-Bang asks for a BMI. Apfel shows a note: "In an HQ it's the patient's two questions only, with no score…". Inserted:
- **DASI:** a Group with 12 Yes/No questions (Yes shows its points: 2.75, 1.75, … 7.5), a Clinician **DASI score** with the band "Poor functional capacity", and **DASI METs**;
- **PRISMA-7:** five Yes/No questions (1 pt each) and a Clinician **PRISMA-7 score** with two bands;
- **Apfel:** a Group "Sickness after an anaesthetic" with just two questions and no score.

Each is locked like STOP-Bang (no Copy, Move or Delete on its questions; read-only Settings; Disclosures work). Step 4: "This HQ already has DASI. A ready-made score can be in an HQ only once." The draft publishes.

#### T-11 A changed ready-made score can't be published
1. Make another draft like T-10 with **DASI** inserted. With it open in the editor, open DevTools → **Console**, paste the snippet below, press Enter, then reload.
2. Press **Publish**.
3. Delete the DASI Group, insert it again, Publish.

```js
(async () => {
  const id = "dasi"; // or "prisma7", "apfel"
  const m = location.pathname.match(/^\/o\/([^/]+)\/h\/([^/]+)\/questionnaires\/([^/]+)/);
  if (!m) return alert("Open the questionnaire in the editor first.");
  const api = `/api/o/${m[1]}/h/${m[2]}`;
  const { chapters } = await (await fetch(`${api}/questionnaires/${m[3]}`)).json();
  for (const c of chapters) {
    const ch = await (await fetch(`${api}/chapters/${c.id}`)).json();
    const g = (ch.content.pages ?? []).flatMap((p) => p.elements ?? []).find((e) => e.readyScore === id);
    if (!g) continue;
    g.elements[0].choices[0].score = 2;
    const res = await fetch(`${api}/chapters/${c.id}/content`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content: ch.content, revision: ch.revision }) });
    return alert(res.ok ? `Done: ${id}'s first question's Yes now scores 2 in “${c.name}”. Reload the page.` : `Refused: ${res.status}`);
  }
  alert(`No ${id} in this questionnaire.`);
})();
```

**Expected:** step 2: the Publish window lists "**<set> · DASI has been changed: a ready-made score can't be changed. Delete it and insert it again.**" and Publish is disabled. Step 3 publishes. (Optional: repeat with `"prisma7"` or `"apfel"`, giving "PRISMA-7 has been changed…" / "Apfel PONV has been changed…".)

### Refused paths

#### T-12 Another hospital or trust can't see them
1. Copy the address of T-02's episode page.
2. Sign in as **Val Clinician** (Hospital A2) and open it. Then as **Bea Author** (Trust B).

**Expected:** neither sees the episode or its assessments ("not found" page or the hospital picker); nothing about DASI is shown.

## Regression (earlier sessions)

- **R-01 STOP-Bang (S47):** an episode on **Demo STOP-Bang HQ**, the patient answers and sends (height 170, weight 82, snoring Yes, the rest No). **Add assessment** → **STOP-Bang**: "8 of 8 filled", the badge e.g. "1 (Low risk)"; Confirm; the POA Summary shows **STOP-Bang 1 (Low risk)**. A note says "The age is the patient's today." before Confirm.
- **R-02 STOP-Bang's neck message (S47):** an episode on **Demo sleep HQ**, sent; STOP-Bang opens 7 of 8, N missing, Confirm greyed out until N is answered.
- **R-03 Observations (S46):** record height 180 and weight 120 on any episode: the hospital BMI 37 shows on the card and the POA Summary.
- **R-04 Insert STOP-Bang (S38):** in a draft with a BMI question, **Ready-made score** → STOP-Bang still asks "BMI from" and inserts; its score's bands are Low, Intermediate and High risk.
- **R-05 The Demo pre-op HQ (S19):** a new episode on the Demo pre-op HQ; the patient fills in and sends on a phone with no sideways scroll.

## Out of scope (don't report)

- MUST, falls, VTE and Rockwood (S49), and DASI or PRISMA-7 as hints for Rockwood.
- Whether the wording, bands or cut-offs are clinically right (a clinician checks them before go-live; the licences are go-live items).
- The Rules page's "(not in this HQ)" choices list STOP-Bang only.
- Adding an assessment automatically (per procedure), or removing one.
- Anything about the incumbent product.
