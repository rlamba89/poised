> **Changed after verification 1 (14 Nov):** T-06 step 3 added (the Preview's earlier answers' number boxes). Restart `make dev` first (the audit CSV's Go changed); no seed needed.
>
> **Changed after test report 1 (14 Nov, all passed):** T-05 now uses an HQ that isn't auto-triaged on send (a sent Demo risk HQ goes straight to Ready for admission, so its sets are read-only), and presses Tab before reloading (Validate saves a number when the box loses focus). T-12 step 3 checks item 3 (Ambulatory aid): Mental status is two buttons, not a menu.

# S49a test plan: C7 follow-ups

**What's new (six small changes):**
1. **MUST:** where the hospital has a weight but no height, the BMI the clinician types keeps the hospital's weight beside it. That already worked the first time; now it also holds after the BMI is **saved, the page reloaded and the BMI changed**.
2. **Number boxes:** every number answer is a plain box. Scrolling the mouse wheel over it (while it has the cursor in it) never changes its value, the up and down arrow keys don't either, and it has **no little up/down arrows**. Everywhere: the patient's form, Fill in for the patient, the clinician's Validate, the Preview, and the assessment cards. Its checks (a number, a smallest and largest value) are as before.
3. **The audit CSV:** a row listing more episodes than a spreadsheet cell holds (about 860) is split across rows in the CSV, "part 1 of 2" and so on. The audit screen and the log itself are unchanged.
4. **An assessment where the patient's record says Other or Not known for sex** (PRISMA-7, Apfel, STOP-Bang): the card's headline says *"The patient's record has no male or female sex, so this score can't be worked out."* instead of asking to "answer the item highlighted".
5. (A journey test fix only: nothing to test by hand.)
6. **An assessment card's menu** (e.g. Morse's Gait) opens lined up under its box, not off to the right.

## Setup

In Rahul's own terminal, from the repo root:

```sh
docker compose up -d db
make migrate seed
make dev
```

**Restart `make dev` if it's already running** (the Go API changed). Leave it running.

**For T-08 only**, also run this once in Rahul's terminal: it adds one made-up audit row (Cara listing 900 made-up episodes) to Trust A's log. The log can't be deleted from, so it stays; it's made-up data in a local database.

```sh
docker compose exec -T db psql -U sj -d sj -c "INSERT INTO audit_log (at, org_id, hospital_id, actor, user_id, action, route, status, patient_ids, episode_ids) SELECT now(), h.org_id, h.id, 'staff', u.id, 'episode.list', 'GET /api/o/{oid}/h/{hid}/episodes', 200, (SELECT array_agg(gen_random_uuid()) FROM generate_series(1, 900)), (SELECT array_agg(gen_random_uuid()) FROM generate_series(1, 900)) FROM hospitals h JOIN orgs o ON o.id = h.org_id, users u WHERE o.name = 'Trust A' AND h.name = 'Hospital A1' AND u.email = 'cara.clinician@hospital-a.example'"
```

Open **http://localhost:3000**. The sign-in page lists the seeded users; click one to sign in (Sign out, top right, to switch).

| User | Who |
| --- | --- |
| **Cara Clinician** | clinician, Hospital A1 only |
| **Alex Author** | super clinician, the whole of Trust A (writes questionnaires) |
| **Ada Admin** | admin, the whole of Trust A (the Audit page) |
| **Bea Author** / **Max Multi** | Trust B (Max: admin of the whole of Trust B) |

**The seeded HQs used (Trust A, made up):**
- **Demo risk HQ:** one Question Set, **"Weight, falls and clots"**: page 1 has "How tall are you?" (cm), "What do you weigh now?" (kg), "Have you lost weight without trying…?" and, after Yes, "About how much weight have you lost without trying (kg)?" (0 to 100).
- **Demo fitness HQ:** DASI, PRISMA-7 and Apfel's questions.

**Making an episode:** as Cara, Hospital A1 → **Episodes** → **New episode**: first name, surname (e.g. `MT S49a <time>`), date of birth, the sex the test names, mobile `07700 900` + 3 digits, an email like `mt-s49a-<time>@example.com`, the HQ the test names, **Create**. The episode page opens.

**The patient's form:** open **http://localhost:3000/qa/messages** (no sign-in locally), type the episode's email in **To**, **Show**, open the newest invite's link, **Continue**, type the date of birth.

**Readings:** the episode page's **Observations** card → **Add a reading** → Weight (kg) (or Height (cm)) → **Save reading**.

**How to "scroll over a box":** click in the box so the cursor is in it, leave the mouse pointer over the box, and turn the mouse wheel (or two-finger scroll on a trackpad) a few notches down and up. In Chrome before this session that changed the number by 1 per notch.

## Test cases

### 1. MUST keeps the hospital weight beside a saved typed BMI

#### T-01 A saved BMI typed where the hospital has no height, changed after a reload
1. As Cara, make an episode on **Demo risk HQ** (any adult, Female). Don't send the patient's answers.
2. On **Observations**, add a **Weight** of **60** only (no height).
3. **Add assessment → MUST**. Expected: 1 BMI **Missing** (empty), 2 Weight now **60**, "Hospital weight".
4. Type **20.8** in **1: BMI**. Expected: BMI "Entered by clinician"; the weight stays **60**, "Hospital weight".
5. **Save**. Expected: "Not confirmed". **Reload the page.** Expected: BMI **20.8** "Entered by clinician", weight **60** "Hospital weight".
6. Change the BMI to **22**. **Expected (the fix):** the weight **stays 60, "Hospital weight"**; it doesn't go Missing or empty.
7. **Save**, reload. Expected: BMI **22** "Entered by clinician", weight **60** "Hospital weight": the same as the screen showed before the reload.
8. Change the BMI again to **19**, then press **Undo changes**. Expected: back to 22 and 60, as saved.

#### T-02 The pairing rule is otherwise unchanged (the hospital has a height)
1. As Cara, on a new Demo risk HQ episode, add **Height 170** and **Weight 60** (hospital BMI 20.8).
2. **Add assessment → MUST**: BMI **20.8** "Hospital BMI", weight **60** "Hospital weight".
3. Type **50** in **2: Weight now**. Expected: the BMI goes **Missing** with an empty box; the result **Incomplete**; Confirm greyed out.
4. **Undo changes**, then type **17.3** in **1: BMI**. Expected: the weight goes **Missing** with an empty box.
5. **Save**, reload, change the BMI to **18**. Expected: the weight is still **Missing** (here the BMI was typed over the hospital's own BMI, so the old rule stands).

### 2. Number boxes: no wheel, no arrows

Use **Chrome** (it's where the wheel changed numbers).

#### T-03 The patient's form at phone size (390×844)
1. As Cara, make an episode on **Demo risk HQ**. Open the patient's form (Setup) with the window at **390×844**. Start "Weight, falls and clots".
2. Look at "How tall are you?" and "What do you weigh now?". Expected: plain boxes, **no up/down arrows** inside them (also when the pointer is over the box).
3. Type **70** in the weight. With the cursor still in it, scroll over it (Setup) down and up several notches. Expected: it **stays 70**; the page scrolls as usual.
4. Click back in the weight box and press the **Up arrow** key twice and the **Down arrow** key three times. Expected: it **stays 70**.
5. Type **165** in the height, wait 2 seconds (the save line shows "Saved"), reload. Expected: 165 and 70 are kept.
6. Answer **Yes** to "Have you lost weight without trying…?" and type **150** in "About how much…(kg)?", then **Next**. Expected: the form refuses it as before (a message about the largest value, 100) and stays on the page. Change it to **6**: **Next** works.
7. Try typing the letter **e** or **+** in a number box. Expected: nothing appears (as before).
8. Finish the set and **Send my answers**. Expected: "Thank you"; no sideways scrolling on any page.

#### T-04 Fill in for the patient (desktop)
1. As Cara, on a new Demo risk HQ episode, press **Fill in for the patient** and start the set.
2. Repeat T-03 steps 2–4 in the weight box. Expected: no arrows; the wheel and arrow keys don't change it.

#### T-05 The clinician's Validate copy
1. As Cara, open an episode whose sent HQ has a number question and **isn't auto-triaged** (a sent Demo risk HQ goes straight to Ready for admission, with read-only sets): e.g. an HQ Alex makes with one number question and no Low risk rule. Open the set to **Validate**.
2. In the number box, click in it, scroll over it, press Up/Down. Expected: the number stays; no arrows.
3. Change it, press **Tab** (Validate saves a number when its box loses focus), wait for "Saved", reload. Expected: the change is kept.

#### T-06 The Preview
1. As Alex, open **Demo risk HQ** in the editor → **Preview** of "Weight, falls and clots".
2. In the weight box: type 80, scroll over it, press Up/Down. Expected: stays **80**; no arrows.
3. **The earlier answers' boxes (verification 1, finding 2):** as Alex, make a draft with two Question Sets: the first with a Number question (e.g. "Drinks a week") and a BMI question, the second with a Calculation that uses both (e.g. its own question's points plus "Add 1 point when" the earlier number or BMI is over a value). Open the **second** set's **Preview** (Clinician view). In the blue "Uses answers from earlier Question Sets" box, type **14** in the number and **20.8** in the BMI; in each, press Up/Down and scroll over it. Expected: they stay **14** and **20.8**; no arrows; the score uses them.

#### T-07 The assessment cards' number boxes
1. As Cara, on T-01's or T-02's episode, on the **MUST** card click in **1: BMI** (or **3: Weight lost**), scroll over it and press Up/Down. Expected: its number doesn't change; no arrows (these were already plain boxes).

### 3. The audit CSV

#### T-08 A list of 900 episodes splits across rows in the CSV
(Needs the Setup's extra command.)
1. As **Ada**, Hospital A1 → **Audit**. Filter **Who: Cara Clinician**, press **Show**.
2. Expected on screen: the newest row is **Listed episodes**, with **900** patients (the screen shows the first few and the count, as before), and it's **one row**.
3. Press **Export CSV** and open the file in a text editor or spreadsheet. (Or, if the download can't be opened: in the page's console, `await (await fetch(location.pathname.replace(/\/h\/[^/]+\/audit$/, '/audit').replace(/^\/o\//, '/api/o/') + '?format=csv&user=' + new URLSearchParams(location.search).get('user'))).text()` and read the result; the Audit page's address holds the `user` filter.)
4. Expected: that request is **two rows**: Action **"Listed episodes (part 1 of 2)"** and **"Listed episodes (part 2 of 2)"**, with the **same** Time, Who, Email, Hospital, Route and Result. The patients' names (each "(record not found)", as the ids are made up), their ids and the episode ids are shared between the two rows, together **900 each**, none repeated. No cell is longer than 32,767 characters (in a spreadsheet: the file opens without a warning about cells being cut short).
5. Expected: the other rows (short lists, opened episodes…) have **no "part"** in their Action, as before.

#### T-09 Refused: the audit isn't anyone's but a whole-trust admin's, and not another trust's
1. As **Cara**, there's no **Audit** in the menu; typing the Audit address of Hospital A1 shows it's not available to her (as before).
2. As **Max Multi** (admin of Trust B), Trust B's **Audit** doesn't show Cara or the 900-episode row.

### 4. The record's sex is Other or Not known

#### T-10 PRISMA-7, Apfel and STOP-Bang say the record has no male or female sex
1. As Cara, make an episode on **Demo fitness HQ** for a made-up patient with **Sex: Other**, born 1 Jan 1950. Don't send the patient's answers.
2. **Add assessment → PRISMA-7**. Expected: the headline under the card's title reads "*N of 7 filled: answer the M items highlighted. The patient's record has no male or female sex, so this score can't be worked out.*" where M counts only the items she can answer (not the sex). The sex item is still marked Missing; the result Incomplete.
3. Answer every other item (Yes/No). Expected: the headline is "*6 of 7 filled. The patient's record has no male or female sex, so this score can't be worked out.*", with **no** "answer the item highlighted"; the result stays **Incomplete**; Confirm greyed out.
4. **Add assessment → Apfel PONV**, then **STOP-Bang**. Expected: each headline ends with the same sentence; the sex isn't counted in "answer the … highlighted". There's no second line under the card saying the same thing again.
5. Repeat steps 1–2 for a patient with **Sex: Not known**. Expected: the same sentence.

#### T-11 Male or female: as before
1. As Cara, make a Demo fitness HQ episode with **Sex: Female** and add PRISMA-7. Expected: "*… filled: answer the … highlighted.*" with no sentence about the record's sex.
2. On T-10's Other patient, **DASI** (which doesn't use the sex): Expected: no sentence about the sex.

### 6. An assessment card's menu

#### T-12 The menu opens lined up under its box
1. As Cara, on any episode, with the window about **1456×900**, **Add assessment → Morse Fall Scale**.
2. Open **5: Gait**'s menu. Expected: the list opens **directly under its box**, its left edge lined up with the box's left edge (before: about 115 px to the right), at least as wide as the box. Pick **Weak**: it shows in the box.
3. Open **3: Ambulatory aid**'s menu. Expected: the same. (6: Mental status is two buttons, not a menu.)
4. **Add assessment → Clinical Frailty Scale**, open **1: Level**. Expected: the list opens lined up with the box's left edge and shows **"8: Living with Very Severe Frailty"** whole (not cut short). Pick it: the box shows it whole.
5. Make the window narrower (about 1000 px wide) and open the Gait menu again. Expected: the list still opens under its box and is fully on screen.

## Regression (re-check what earlier sessions built)

- **R-01 (S49):** MUST with height 170 and weight 60 opens 3 of 4 (Hospital BMI 20.8, Hospital weight 60, HQ weight lost); "4: No" gives "1 (Medium risk), …% of body weight lost"; **Confirm** works.
- **R-02 (S47):** STOP-Bang on a Female patient pre-fills from the HQ and the hospital BMI as before; Confirm, then the POA Summary shows "STOP-Bang N (…)".
- **R-03 (S44c):** in the patient's form, a number edited (e.g. weight 120 → 125 with Backspace then 5) keeps a follow-up it still shows.
- **R-04 (S11):** the Audit page's filters (Who, days, Patient) and Export CSV still work for short lists; times are in the browser's time zone.
- **R-05 (S14):** a new episode's invite shows on /qa/messages for its email and mobile, and its link opens the patient's form.

## Out of scope (don't report)

- Firefox and Safari never changed a number on the wheel; only Chrome (and Edge) did. Check T-03–T-07 in Chrome.
- The made-up audit row's patient names read "(record not found)": its ids are made up.
- The settings page's and the jobs page's number boxes (they're settings, not answers; Mantine's own boxes with their own arrows).
- The clinician checks of wording and bands, the licences and other go-live items (pending.md, for C10).
- A retired starter question's key still offered by "Same as…" through a trust's own copies (as built, Rahul 14 Nov).
