> **Added after verification 6 (14 Nov):** T-25 (on MUST, a box typed in earlier counts as untouched again once the other box is put back). Restart `make dev` first; no seed needed.
>
> **Changed after test report 7 (13 Nov):** T-24 steps 2 and 3 now also expect the box marked Missing to be empty (F-1). Restart `make dev` first; no seed needed.
>
> **Added after verification 5 (13 Nov):** T-24 (a box typed in and put back counts as untouched; an emptied box doesn't block Save; the weight-lost line), and **T-23 step 4 changed** (Rahul: a BMI typed where the hospital has no height keeps the hospital's weight). Restart `make dev` first; no seed needed.
>
> **Added after verification 4 (13 Nov):** T-23 (on MUST, the hospital's own number typed back after the other was typed counts). Restart `make dev` first; no seed needed.
>
> **Added after verification 3 (13 Nov):** T-22 (a weight typed on MUST clears the pre-filled BMI). Restart `make dev` first; no seed needed.
>
> **Added after verification 2 (13 Nov):** T-19 (VTE stays Incomplete until every item is answered), T-20 (the recent-time questions aren't copied into a patient's next HQ), T-21 (MUST, precedence off, hospital pair). Restart `make dev` first; `make migrate seed` isn't needed (your local Demo risk HQ was updated in place again).
>
> **Added after verification 1 (8 Nov):** T-17 (MUST with a hospital weight but no height) and T-18 (a man isn't asked three VTE questions). **Restart `make dev` first** (the evaluate service changed too); `make migrate seed` isn't needed (your local Demo risk HQ was updated in place).
>
> **Changed after test report 1 (8 Nov):** T-01 no longer counts the seeded HQ's own name ("Demo risk HQ"); T-03 says a value typed back to the one shown keeps its source; T-12 also checks the VTE value lines up; T-10 checks the menu shows a level whole.

# S49 test plan: MUST, Morse falls, VTE and the Clinical Frailty Scale

**What's new:** four more assessments on the episode page's **Assessments** card, after STOP-Bang, DASI, PRISMA-7 and Apfel PONV:
- **MUST** (malnutrition screening): 4 items. **1 BMI** and **2 Weight now** come from the clinician's height and weight readings (Observations); **3 Unplanned weight loss in the past 3–6 months (kg)** from the patient's HQ (a "No" there is 0 kg); **4 Acute disease effect** is the clinician's Yes/No (Yes scores 2). Scoring: BMI over 20 → 0, 18.5–20 → 1, under 18.5 → 2; weight lost as a % of the weight before (now + lost): under 5% → 0, 5–10% → 1, over 10% → 2. Result **0 Low risk, 1 Medium risk, 2 or more High risk**, plus the %, e.g. "1 (Medium risk), 9.1% of body weight lost".
- **Morse Fall Scale** (falls risk): 6 items, points: **1** History of falling (Yes 25), **2** Secondary diagnosis (Yes 15), **3** Ambulatory aid (Nothing 0, Crutches/stick/frame 15, Furniture 30), **4** IV or heparin lock (Yes 20), **5** Gait (Normal 0, Weak 10, Impaired 20), **6** Mental status (Knows own ability 0, Overestimates/forgets limits 15). The patient answers 1 and 3 in the HQ. Bands: **0–24 Low risk, 25–44 Moderate risk, 45+ High risk**.
- **VTE risk** (the Department of Health's form): 30 Yes/No risk factors, **T1–T19 thrombosis**, **B1–B11 bleeding**. T2 "Age over 60" is the patient's record; T5 "Obesity (BMI over 30)" is a BMI box from the readings; 9 come from the patient's HQ; the rest are the clinician's. The result counts the thrombosis ticks: **0 → "No thrombosis risk factors ticked"**, **1 or more → "Thrombosis risk: consider thromboprophylaxis"**, then "Bleeding risk factors: N".
- **Clinical Frailty Scale** (Rockwood): one item, the **Level**, picked from a menu of 9 ("1: Very Fit" … "9: Terminally Ill"). Result e.g. "5 (Living with Mild Frailty)". It shows **DASI's and PRISMA-7's confirmed results** in a blue "For reference" box, when the episode has them confirmed. It's never in an HQ.

An author can insert **MUST, Morse Fall Scale and VTE risk** in an HQ as ready-made parts: **only the patient's questions, with no score**. The Clinical Frailty Scale isn't offered there.

## Setup

In Rahul's own terminal, from the repo root:

```sh
docker compose up -d db
make migrate seed
make dev
```

**Restart `make dev` if it's already running** (the Go API, the evaluate service and the web app all changed), and run `make migrate seed` (it adds the new demo HQ). Leave it running. Open **http://localhost:3000**. The sign-in page lists the seeded users; click one to sign in (Sign out, top right, to switch).

| User | Who |
| --- | --- |
| **Cara Clinician** | clinician, Hospital A1 only |
| **Val Clinician** | clinician, Hospital A2 only (same trust) |
| **Alex Author** | super clinician, the whole of Trust A (writes questionnaires) |
| **Ada Admin** | admin, the whole of Trust A (Settings) |
| **Bea Author** | super clinician, the whole of Trust B |

**The seeded HQs for this session (Trust A, made up):**
- **Demo risk HQ** (new): one Question Set, **"Weight, falls and clots"**, three pages: **height and weight** plus MUST's two questions ("Have you lost weight without trying in the last 3 to 6 months?" and, only after Yes, "About how much…(kg)?"); **Falls** (a fall in the last 3 months; what you use to walk); **Blood clots and bleeding** (9 Yes/No questions).
- **Demo fitness HQ** (S48): DASI, PRISMA-7 and Apfel's questions. Used for the Frailty Scale's hints.

**Making an episode:** as Cara, Hospital A1 → **Episodes** → **New episode**: first name, surname (e.g. `MT S49 <time>`), the date of birth and sex the test names, mobile `07700 900` + 3 digits, an email like `mt-s49-<time>@example.com`, the HQ the test names, **Create**. The episode page opens.

**The patient's answers:** open **http://localhost:3000/qa/messages** (no sign-in locally), open the newest invite to the episode's email **at phone size (390×844)**, type the date of birth, answer the set and **send**. (Or, as Cara, **Fill in for the patient**, answer and send.) The assessments use the patient's answers only once they're sent.

**Readings:** on the episode page's **Observations** card, **Add a reading** → Height (cm) / Weight (kg) → **Save reading**. **Correct** on a reading asks for the new value and "Why is it corrected?".

**Ages:** today is 7 Nov 2026. Born **1 Jan 1965** is **61**; born **1 Jan 1966** is **60**; born **1 Jan 1980** is 46.

## Test cases

### The patient's side

#### T-01 The patient answers the Demo risk HQ on a phone
1. As Cara, make an episode on **Demo risk HQ** for a made-up **woman born 1 Jan 1965**. Keep it: T-02, T-04, T-07 and T-08 continue it.
2. Open the invite at **390×844**, type the date of birth, **Start** "Weight, falls and clots".
3. Page 1: height **165**, weight **70**. "Have you lost weight without trying…?" **No** first: no "how much" question shows. Change to **Yes**: "About how much weight have you lost without trying (kg)?" appears; type **6**. **Next**.
4. Page 2: "Have you had a fall in the last 3 months?" **Yes**; "Do you use anything to help you walk?" **Crutches, a stick or a walking frame**. **Next**.
5. Page 3: **Yes** to "Are you taking hormone replacement therapy (HRT)?" and to "Do you take medicine to thin your blood…?"; **No** to the other 7. **Finish this section**, **Send my answers**, confirm.

**Expected:** no sideways scrolling on any page. The patient sees only questions: no score, no "MUST", "Morse", band or percentage anywhere, and no result such as "High risk" (the seeded HQ's own name, "Demo risk HQ", doesn't count). Required questions need an answer before Next. The send says "Thank you". On the patient's home afterwards there's **nothing about assessments**.

### MUST

#### T-02 MUST opens 3 of 4: BMI and weight from the readings, weight lost from the HQ
1. Continue T-01. As Cara, on the episode page, **Add a reading**: Height **170**, then Weight **60**. The hospital BMI reads **20.8**.
2. **Add assessment** → **MUST**.
3. Click **No** on item 4. **Save**.

**Expected:** step 2: "3 of 4 filled: answer the item highlighted." **1 BMI** box **20.8**, source **Hospital BMI**; **2 Weight now** box **60**, **Hospital weight**; **3 Unplanned weight loss…(kg)** box **6**, **HQ**; **4 Acute disease effect**, the wording "Acute disease effect: is the patient acutely ill and there has been or is likely to be no nutritional intake for >5 days?", highlighted orange, **Missing**. Badge **Incomplete**, Confirm greyed out. Step 3: item 4 **Clinician**; badge **1 (Medium risk), 9.1% of body weight lost** (BMI 20.8 → 0; 6 of 66 kg = 9.1% → 1). After Save: "Not confirmed".

#### T-03 MUST's published cut-offs, typed by hand
1. Continue T-02's card (or any MUST card). Type into boxes 1, 2 and 3 and set item 4 as each row says, reading the badge (don't Save).

| 1 BMI | 2 Weight now | 3 Weight lost | 4 Acute | Expected badge |
| --- | --- | --- | --- | --- |
| 22 | 70 | 0 | No | **0 (Low risk), 0% of body weight lost** |
| 34 | 100 | 0 | No | **0 (Low risk)…** (over 30 is still 0) |
| 20.1 | 60 | 0 | No | **0 (Low risk)…** |
| 20 | 60 | 0 | No | **1 (Medium risk)…** (18.5–20 is 1) |
| 18.5 | 55 | 0 | No | **1 (Medium risk)…** |
| 18.4 | 55 | 0 | No | **2 (High risk)…** |
| 22 | 95.1 | 4.9 | No | **0 (Low risk), 4.9% of body weight lost** |
| 22 | 95 | 5 | No | **1 (Medium risk), 5% of body weight lost** |
| 22 | 90 | 10 | No | **1 (Medium risk), 10% of body weight lost** |
| 22 | 89.9 | 10.1 | No | **2 (High risk), 10.1% of body weight lost** |
| 22 | 70 | 0 | Yes | **2 (High risk), 0%…** |
| 17 | 44 | 6 | Yes | **6 (High risk), 12% of body weight lost** |

2. Box mistakes: type **101** in box 3, then **-1**, then **abc**, then empty it; type **0** in box 2.

**Expected:** each badge as the table says; a typed item's source is **Entered by clinician** (typing back the value it showed keeps its old source, as since S47). Step 2: box 3 says "Between 0 and 100" for 101, "A number, one decimal place" for -1 and for abc, and emptied, "Enter a number"; box 2 says "Between 1 and 400" for 0. While a box is wrong, Save and Confirm are greyed out. **Undo changes** brings back T-02's values.

#### T-04 A corrected weight changes MUST before Confirm; after Confirm it's only noted
1. Continue T-02 (saved, not confirmed; Undo any T-03 typing).
2. On the Observations card, **Correct** Weight 60.0 kg → **55**, reason "Made-up: outdoor clothes", **Save correction**.
3. Read the MUST card. **Confirm**. Reload.
4. **Correct** the weight again → **58**, with a reason.

**Expected:** step 3: without a reload, box 1 shows **19** and box 2 **55** (sources still Hospital BMI / Hospital weight); item 4 keeps **No (Clinician)**; badge **2 (High risk), 9.8% of body weight lost**. After Confirm: "Confirmed by Cara Clinician on …", kept after reload. Step 4: the badge **stays** 2 (High risk), 9.8%…; a yellow box says "The hospital BMI is now 20.1 (19 when confirmed)." and "The hospital weight is now 58 (55 when confirmed)." with "The score stays as confirmed…".

#### T-05 A "No" to weight loss is 0 kg; no readings means missing BMI and weight
1. Make a new episode on **Demo risk HQ** (anyone). As the patient, height 160, weight 64, **No** to weight loss, answer the rest anything, send.
2. As Cara, **don't** add readings. **Add assessment** → **MUST**.

**Expected:** item 3 box **0**, source **HQ**. Items 1 and 2 show the patient's own: **25** (Patient reported BMI) and **64** (Patient reported weight). Item 4 missing.

#### T-06 The trust's precedence also decides MUST's weight
1. Continue T-05. As Cara, add Height **160**, Weight **70** (hospital BMI 27.3). Reload: MUST items 1 and 2 show **27.3 Hospital BMI** and **70 Hospital weight**.
2. As **Ada Admin**, Settings: turn **off** "Observations take precedence in scores". Save.
3. As Cara, reload the episode.

**Expected:** step 3: items 1 and 2 show **25 Patient reported BMI** and **64 Patient reported weight**. (Turn the setting back **on** afterwards.)

### Morse Fall Scale

#### T-07 Morse: 2 of 6 from the HQ, then the published worked example (65)
1. Continue T-01's episode. **Add assessment** → **Morse Fall Scale**.
2. Answer: **2** No, **4** No, **5** **Weak** (menu), **6** **Overestimates or forgets their limits**.
3. Then try: 1 No, 3 Nothing, 5 Normal…, 6 Knows…: then 4 Yes; then 4 No and 1 Yes; then 1 Yes and 4 Yes; then 3 **I hold on to furniture** and 5 **Impaired** (1 No, 4 No).
4. Put back step 2's answers (1 Yes, 3 Crutches…) and **Confirm**.

**Expected:** step 1: "2 of 6 filled: answer the 4 items highlighted." **1 History of falling** Yes (**HQ**); **3 Ambulatory aid** a menu showing "Crutches, a stick or a walking frame" (**HQ**); 2, 4, 5, 6 **Missing**. Item 5 (Gait) is a menu with 3 choices; item 6 has two buttons. Step 2: badge **65 (High risk)**. Step 3 badges: all zero → **0 (Low risk)**; IV only → **20 (Low risk)**; fall only → **25 (Moderate risk)**; fall + IV → **45 (High risk)**; furniture + impaired → **50 (High risk)**. Step 4: confirmed.

### VTE

#### T-08 VTE: the patient's factors, the age and the BMI pre-filled
1. Continue T-01's episode (61, HRT and anticoagulant Yes; hospital BMI now 20.1 after T-04). **Add assessment** → **VTE risk**.
2. Read the card. Then answer **No** to every missing item.
3. Type **31** in T5's box; then **30**; then **30.1**.
4. Click **Yes** on **B1 Active bleeding**. **Confirm**.

**Expected:** step 1: "11 of 30 filled: answer the 19 items highlighted." Rows **T1–T19** then **B1–B11**. **T2 Age over 60** shows **61**, **Profile**; **T5 Obesity (BMI over 30)** a box with **20.1**, **Hospital BMI**; T1, T4, T7–T11 and **B3 Anticoagulants**, **B7 Inherited bleeding disorders** from the **HQ** (T8 HRT **Yes**, B3 **Yes**, the rest No). Clinician items show the form's own wording (e.g. "Total anaesthetic + surgical time > 90 minutes"); a note says "The age is the patient's today." Step 2: badge **2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1** (age 61 and HRT; the anticoagulant). Step 3: 31 → **3 (…)**; 30 → **2 (…)**; 30.1 → **3 (…)**. Step 4: "Bleeding risk factors: 2"; confirmed.

#### T-09 VTE: 60 isn't over 60; nothing ticked
1. Make an episode on **Demo risk HQ** for a made-up **man born 1 Jan 1966** (60). As the patient: No to all 9 clot questions, the rest anything; send.
2. As Cara, add Height 180, Weight 70. **Add assessment** → **VTE risk**; answer No to every missing item.

**Expected:** T2 shows **60**; badge **0 (No thrombosis risk factors ticked), Bleeding risk factors: 0**.

### Clinical Frailty Scale

#### T-10 The level, and DASI's and PRISMA-7's results as hints
1. Make an episode on **Demo fitness HQ** for a **woman born 1 Jan 1980**; as the patient answer DASI Yes to 1, 2, 4, 5, 7, 8, 9 (No to the rest), PRISMA-7 No to all but question 6 (Yes), Apfel anything; send.
2. As Cara, **Add assessment** → **Clinical Frailty Scale**.
3. **Add assessment** → **DASI**, **Confirm** it. Look at the Frailty Scale card.
4. **Add assessment** → **PRISMA-7**, **Confirm** it.
5. On the Frailty Scale card pick **5: Living with Mild Frailty**; then **9: Terminally Ill**; then **4: Living with Very Mild Frailty**; **Confirm**.
6. On DASI, click **No** on item 1, **Save** (DASI is now not confirmed).

**Expected:** step 2: "0 of 1 filled", one row **1 Level**, wording "Clinical Frailty Scale level, from the patient's usual state (about two weeks before)", a **Choose** menu listing the 9 levels "1: Very Fit" … "9: Terminally Ill", **Missing**; **no** blue box. Step 3: a blue **For reference** box: "DASI: 34 (Poor functional capacity), 6.9 METs (confirmed …)". Step 4: a second line "PRISMA-7: 1 (Not at increased risk) (confirmed …)". Step 5 badges: **5 (Living with Mild Frailty)**, **9 (Terminally Ill)**, **4 (Living with Very Mild Frailty)**, the closed menu showing each level whole (try **8: Living with Very Severe Frailty** too); source **Clinician**; confirmed. Step 6: the DASI line is gone from the box; the PRISMA-7 line stays. No hint box on any other card.

#### T-11 The Frailty Scale is never in an HQ; MUST, Morse and VTE insert as the patient's part
1. As **Alex Author**, open a draft questionnaire (or make one), a Question Set's Add bar → **Ready-made score**.
2. Open the score menu. Pick **MUST**, read the window, **Insert**. Do the same for **Morse Fall Scale** and **VTE risk** (a new page or set each is fine).
3. **Preview** the set.

**Expected:** the menu offers **STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse Fall Scale, VTE risk**, and **not** the Clinical Frailty Scale. Each window shows the source and licence (MUST: "© BAPEN…", with BAPEN's "reproduced here with the kind permission of BAPEN…" line; Morse: "© Janice Morse…"; VTE: "© Crown copyright 2010…") and a note that in an HQ it's the patient's questions only, with no score. Inserted: MUST is a Group "Weight loss" with 2 questions, Morse "Falls" with 2, VTE "Blood clots" with 9; no Calculation. Their cards can't be edited one by one (as STOP-Bang's). The Preview shows the questions only.

### Added after verification 1

#### T-17 MUST: a hospital weight without a height leaves the BMI for the clinician
1. Make an episode on **Demo risk HQ** (a woman born 1 Jan 1980). As the patient: height **165**, weight **70**, weight loss **No**, the rest anything; send.
2. As Cara, **Add a reading**: Weight **60** only (no height). **Add assessment** → **MUST**.
3. Add Height **170**. Then mark the height **Entered in error** (with a reason).
4. As **Ada Admin**, turn **off** "Observations take precedence in scores"; as Cara, reload. (Turn it back **on** afterwards.)

**Expected:** step 2: **1 BMI** is **Missing** (orange, empty box) and **2 Weight now** shows **60, Hospital weight**; the patient's own 25.7 is **not** used. Step 3: with the height, item 1 shows **20.8, Hospital BMI**; after the height is entered in error, item 1 is Missing again and item 2 still 60. Step 4: items 1 and 2 show **25.7 Patient reported BMI** and **70 Patient reported weight** (both the patient's).

#### T-18 A man isn't asked about pregnancy, contraception or HRT
1. Make an episode on **Demo risk HQ** for a **man born 1 Jan 1966**. Open the invite at **390×844**, answer pages 1 and 2, and read page 3 (Blood clots and bleeding). Answer it and send.
2. As Cara, **Add assessment** → **VTE risk**.
3. Repeat step 1 for a patient recorded as **Other** (sex Other), and for a **woman**.

**Expected:** step 1: page 3 has **6** questions; none of "Are you pregnant…", "Are you taking hormone replacement therapy (HRT)?", "Are you using a contraceptive that contains oestrogen…". Step 2: **T8 Hormone replacement therapy**, **T9 Oestrogen-containing contraceptive** and **T11 Pregnancy…** are **Missing** (for Cara to answer), not No; answering Yes on T8 counts in the result. Step 3: Other and the woman see all **9** questions.

### Added after verification 2

#### T-19 VTE stays Incomplete until every item, bleeding ones too, is answered
1. On any Demo risk HQ episode, **Add assessment** → **VTE risk**. Answer **No** to every missing **T** item, but leave one **B** item (e.g. B1) unanswered.
2. Answer B1.

**Expected:** step 1: the badge says **Incomplete** (no thrombosis result), "Confirm once every item has an answer." shows, **Confirm** is greyed out, and B1 is highlighted. Step 2: the badge shows both results ("… Bleeding risk factors: N") and Confirm is enabled. The same holds with a T item left instead.

#### T-20 Recent-time questions aren't copied into the next HQ
1. Make an episode on **Demo risk HQ** for a made-up patient; as the patient answer **Yes** to "lost weight…" with **5** kg, **Yes** to "Have you had a fall in the last 3 months?", and for a woman **Yes** to "Are you pregnant…?"; the rest anything; send.
2. As Cara, make a **second** episode for the **same patient** (same details) on **Demo risk HQ**. Open the patient's new invite.

**Expected:** step 2: the set opens with the other answers pre-filled (height, the walking aid, the other clot questions) but **"Have you lost weight without trying…?", "About how much…", "Have you had a fall in the last 3 months?" and "Are you pregnant…?" are empty**, to be answered afresh (like the library's "What do you weigh now?").

#### T-21 MUST with precedence off: the hospital's full pair beats a patient's weight alone
1. As **Alex Author**, publish a small HQ holding the library's **weight** question (no height) and the ready-made **MUST**. Make an episode on it; as the patient give a weight of **68** and No to weight loss; send.
2. As **Ada Admin**, turn **off** "Observations take precedence in scores".
3. As Cara, add Height **170** and Weight **60**. **Add assessment** → **MUST**. Then mark the height **Entered in error**.

**Expected:** step 3: items 1 and 2 show **20.8 Hospital BMI** and **60 Hospital weight**. After the height is entered in error: item 1 **Missing**, item 2 **68 Patient reported weight** (a patient's weight alone beats a hospital weight alone). Turn the setting back **on** afterwards.

### Added after verification 3

#### T-22 A weight or BMI typed on MUST leaves the other for the clinician
1. On any Demo risk HQ episode with a height **170** and weight **60** recorded, **Add assessment** → **MUST** (items 1 and 2: 20.8 Hospital BMI, 60 Hospital weight). Answer item 4 **No**.
2. Type **50** in item 2 (Weight now).
3. Type **17.3** in item 1 (BMI).
4. **Undo changes**. Then type **18** in item 1 only.
5. **Undo changes**, **Confirm**. Then type **50** in item 2.

**Expected:** step 2: item 1's box empties, **Missing** (orange); badge **Incomplete**; Confirm greyed out, with "Confirm once every item has an answer." Step 3: both **Entered by clinician**; a full result (e.g. "2 (High risk), 3.8% of body weight lost" with 2 kg lost: BMI 17.3 scores 2, 2 of 52 kg scores 0; or as your HQ gave). Step 4: item 2's box empties, **Missing**; item 1 18, Entered by clinician. Step 5: after Confirm, typing 50 again empties item 1 and shows "Changed: confirm it again"; **Undo changes** brings back the confirmed values.

### Added after verification 4

#### T-23 On MUST, the hospital's own number typed back after the other was typed counts
1. On a Demo risk HQ episode with a height **170** and weight **60** recorded, **Add assessment** → **MUST** (20.8 Hospital BMI, 60 Hospital weight). Answer item 4 **No**.
2. Type **30** in item 1 (BMI). Then type **60** in item 2 (the hospital's own weight, typed back).
3. **Undo changes**. Type **61** in item 2. Then type **20.8** in item 1 (the hospital's own BMI, typed back).
4. **Undo changes**. Mark the height **Entered in error** (item 1 goes **Missing**, item 2 stays 60 Hospital weight). Type **20.8** in item 1. **Confirm**.
5. Reload the page.

**Expected:** step 2: after 30, item 2 empties and goes **Missing**; after 60, item 2 shows **Entered by clinician** (not Missing), the badge shows a score (e.g. "0 (Low risk), …" with no weight lost), and **Confirm** is on. Step 3: after 20.8, both items **Entered by clinician**, a score, Confirm on. Step 4 (changed after verification 5): after 20.8, item 1 **Entered by clinician** and item 2 **stays 60 Hospital weight** (not Missing); Confirm works ("Confirmed by …"). Step 5: the confirmed card shows the same values and result, not Missing.

### Added after verification 5

#### T-24 On MUST, a box typed in and put back counts as untouched
1. On a Demo risk HQ episode with a height **170** and weight **60** recorded, **Add assessment** → **MUST** (20.8 Hospital BMI, 60 Hospital weight). Answer item 4 **No**.
2. Type **36** in item 1, then type **20.8** back. Then type **50** in item 2.
3. **Undo changes**. Type **61** in item 2, then **60** back. Then type **17.3** in item 1.
4. **Undo changes**. Type **50** in item 2 (item 1 goes Missing). Click in item 1's empty box, type **1**, then delete it.
5. **Undo changes**, **Confirm**. Then open the HQ's **Weight, falls and clots** Question Set to validate it (the clinician's copy), change the weight lost (e.g. from 2 to **3** kg, or No to Yes and 3), save, and go back to the episode page. (If the episode was auto-triaged, its HQ review is closed: sign in as **Sue Super** and **Re-open HQ review** first; test report 7.)

**Expected:** step 2: after 20.8 back, item 1 reads **Hospital BMI**; after 50, item 1 is **Missing** (not "Entered by clinician" 20.8) and **its box is empty** (the grey "BMI" hint, not 20.8; test report 7, F-1), the badge **Incomplete**. Step 3: after 17.3, item 2 is **Missing** and its box is empty (not 60). Step 4: no "Enter a BMI" message, and **Save** stays on. Step 5: the MUST card's yellow line reads like "The HQ now gives 3 kg for Unplanned weight loss in the past 3–6 months (2 kg when confirmed)."

#### T-25 On MUST, putting the other box back makes the earlier box untouched again
1. On a Demo risk HQ episode with a height **170** and weight **60** recorded, **Add assessment** → **MUST** (20.8 Hospital BMI, 60 Hospital weight). Answer item 4 **No** (a confirmed card works too).
2. Type **61** in item 2 (item 1 goes Missing). Type **20.8** in item 1's empty box. Then type **60** back in item 2. Then type **50** in item 2.
3. **Undo changes**. Type **36** in item 1 (item 2 goes Missing). Type **60** in item 2's empty box. Then type **20.8** back in item 1. Then type **25** in item 1.
4. **Undo changes**. Type **61** in item 2, then **20.8** in item 1 (verification 4's case).

**Expected:** step 2: after 20.8, both **Entered by clinician**; after 60 back, **Hospital BMI** and **Hospital weight** again; after 50, item 1 is **Missing** with an empty box (not 20.8 "Entered by clinician"), the badge **Incomplete**, Confirm off. Step 3: after 20.8 back, both hospital sources again; after 25, item 2 is **Missing** with an empty box, Incomplete. Step 4: both stay **Entered by clinician** (20.8 and 61), with a score and Confirm on.

### The POA Summary, the patient, and refusals

#### T-12 The POA Summary shows each confirmed result
1. Continue T-01's episode (MUST, Morse and VTE confirmed in T-04, T-07, T-08). Add **Clinical Frailty Scale**, don't confirm.
2. Open **POA Summary** (both tabs).

**Expected:** under Hospital BMI: **MUST 2 (High risk), 9.8% of body weight lost**, **Morse Fall Scale 65 (High risk)**, **VTE risk 3 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 2** (or as you left it), **Clinical Frailty Scale Not confirmed**. Each value lines up with the others: the VTE value wraps onto a second line but starts in the same column as Hospital BMI's (report 1, F-1).

#### T-13 The patient never sees an assessment
1. After T-12, open the T-01 patient's link again at **390×844** (a new invite or sign-in via **/signin**), their home and their sent HQ.

**Expected:** nothing about MUST, Morse, VTE, the Frailty Scale, scores, bands or readings anywhere on the patient's pages.

#### T-14 Another hospital's or trust's staff can't reach them
1. Copy T-01's episode page URL. Sign in as **Val Clinician** (Hospital A2) and open it. Then as **Bea Author** (Trust B).

**Expected:** neither sees the episode (not found / no access), so no assessments.

#### T-15 Reload, deep link, back
1. On T-01's episode, change MUST item 4 to Yes (don't save). Reload.
2. Open the POA Summary, then the browser's **Back**.

**Expected:** step 1: the unsaved change is gone; the card is as last confirmed (a change isn't kept without Save). Step 2: the episode page, with the assessments as saved.

### Cancelled or archived

#### T-16 A cancelled episode's assessments are read-only
1. Cancel T-09's episode (Cancel, a reason).

**Expected:** the VTE card shows its values as text (no buttons, boxes or menus), no Save, Confirm or Add assessment.

## Regression (earlier sessions)

- **R-01 (S46)** Observations: add a weight of **-3**: refused with Go's reason; a heart rate of 220 warns and saves with "Save it anyway".
- **R-02 (S47)** STOP-Bang on a **Demo STOP-Bang HQ** or **Demo sleep HQ** episode opens with its items filled and sources (S, T, O, P from the HQ, B Hospital BMI, A and G Profile), confirms, and shows on the POA Summary.
- **R-03 (S48)** DASI on a **Demo fitness HQ** episode: "12 of 12 filled.", **34 (Poor functional capacity), 6.9 METs** (as T-10).
- **R-04 (S48)** Apfel opens 3 of 4, waiting only for "Opioids after surgery"; Yes gives **4 (PONV risk about 79%)** for a woman who doesn't smoke and had PONV.
- **R-05 (S48)** The Ready-made score window still asks for a BMI for STOP-Bang only.

## Out of scope (don't report)

- The **wording** of the instruments' items and bands (ours, close to the published forms; a clinician checks it before go-live), and their **licences** (go-live items).
- MMSE or any other cognitive screen; assessments feeding the ASA rules or auto-triage.
- That unsaved card changes are lost on reload without a warning (as before S49).
- That an assessment's name changes only on new forms after a rename (checked by an automated test; there's no way to rename one in the app).
- The VTE card's length (30 items, as the form has).
