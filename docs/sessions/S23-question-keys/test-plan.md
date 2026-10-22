# S23 test plan: question keys in the editor

**Retest after verification 2 (22 Oct, the last round).** Changes since test report 4:
- **The local database was reset** to migrated + seeded (HJE Full HQ put back as it was): earlier runs' questionnaires (T23 … 41–43, E2E …) are gone, and the Demo pre-op HQ is v1 again. Use a new run number.
- **A date's format and a text question's format are frozen once published, and part of the wording match** (Rahul, "after verification 2" in `answers.md`). New case **T-18**.
- **A change inside a repeating group is reported once** (on the question inside it), and rewording a question inside a repeating group is allowed, as for any question (T-18 step 5).
- **The Publish window drops Go's refusal once it has re-checked** (rare: it needs the trust to change between two requests; no case).

**Retest after test report 3 (22 Oct).** Changes since report 3:
- **R-05 now passes on the local database:** the three old "S02 test HQ …" questionnaires (made on 8–9 Oct by an S02 helper script, before the lock) published `q_diabetes` and `q_angina` with other answers; they're removed, with your v2 draft of the Demo pre-op HQ and the v3 draft of T23 Fresh 42. Rahul kept the strict rule (`answers.md`, "After test report 3").
- **When the trust's published copies of a key disagree, Publish gives one problem** ("…the trust has already published this question with different answers (in …), so it can't be published again as it is…"), not "Yes was removed" with "Yes was added". There's no such data left locally to see it; an automated test covers it.
- **The editor's default names "Group" and "Section" never match by wording** (report 3, found 1), like "Text question".
- **The pre-fill badge is a button:** Tab to it, Enter opens the list, Escape closes it (report 3, found 2; T-02 step 3).

**Retest after verification 1 (22 Oct).** Changes since test report 1:
- **The publish lock now freezes the whole answer** (Rahul, "after verification 1" in `answers.md`): a published question's **kind of answer** (e.g. Select One → Select Many, Number → Date), **unit**, **grid rows and columns**, **rating scale**, a repeating list's **columns**, and its options (values and wording, as before). New cases **T-14** and **T-15**.
- **The problem message changed** to "… A published question's answers can't change: add a new question instead (published in …)."
- **Signature and File upload questions (and the BMI block) never pre-fill:** no pre-fill badge and no "Always ask fresh" switch on them. New case **T-16**.
- **Grids and ratings match by wording only with the same rows/columns or scale** (T-15).
- **The Publish window gets the key problems from the server** as it opens ("Checking the Question Sets…" for a moment). A published chapter in an unusual but valid shape no longer stops publishing (T-17).
- Run all cases again with a new run number; T-01 to T-13 are otherwise unchanged.

**For the tester:** you test only through the browser (and, where a case says so, the browser's JavaScript console). You haven't seen the code. Report each case as Pass, Fail or Blocked, with what you saw.

**What this session built:**
- Every question's **key** is its ID (shown as "ID q_…" at the top of its Settings panel). Pre-fill (a later session) will copy a patient's earlier answer into a new questionnaire when a question has **the same key**, or failing that **exactly the same wording**.
- **The pre-fill label** on each patient question's card in the editor: **"Pre-fills from N other HQs"** (click it to see which, each "by link" = same key, or "by same wording"), or **"New question: won't pre-fill"**. Only the trust's *other* questionnaires with a published (or older, retired) version count; never another trust's.
- **Always ask fresh:** a switch in a patient question's Settings, under Clinical. When on, the card shows an orange **Always ask fresh** badge instead of the label.
- **Publish is refused** (the Publish window lists the problems, and the API refuses too):
  - when a question that was **published** (in this HQ's earlier version, or in any other HQ of the trust with the same key) now has **different options**: an option **added, removed or reworded**. Changing the *order* of options, or the *question's own text*, is allowed.
  - when **one key is used twice** in the HQ (in two Question Sets, or twice in one). The editor never makes one itself, so you make one through the console.

**Note (differs from the brief's manual focus 1 and 2):** Rahul decided on 22 Oct that a published question's options are **frozen completely**. So *adding* or *rewording* an option is refused too, not only removing one (`answers.md`, Q2).

## 1. Setup

Rahul (or the conductor) runs, in his own terminal, and leaves `make dev` running:

```sh
docker compose up -d db && make migrate seed && make dev
```

Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

| User | Role | Use for |
| --- | --- | --- |
| **Alex Author** | super clinician, the whole of Trust A | authoring and publishing |
| **Sue Super** | super clinician, Hospital A1 only | sees the trust's questionnaires read-only |
| **Cara Clinician** | clinician, Hospital A1 only | sees questionnaires read-only, no label |
| **Bea Author** | super clinician, the whole of Trust B | another trust |

Sign out from the avatar menu (top right of the editor) or the app's menu, then click another name.

**The local database already holds many questionnaires named "E2E …"** (left by automated tests) in both trusts. Most share keys and wording with the seeded **Demo pre-op HQ**, so the Demo HQ's questions pre-fill from many HQs. That's expected. The cases below make their own questionnaires with a **run number** `<n>` (pick one, e.g. `1`, and use the same one throughout) so the counts are predictable.

**Making a questionnaire:** Questionnaires → **New Questionnaire** → Name and Description → Save. In the editor: **Add Question Set**, then **Add Page** on it, then add a question from the buttons on the page (**Text**, **Select One**, …). The selected question's **Settings** panel is on the left; **Question text** saves when the box loses focus. Option texts are edited in the Settings panel ("Option 1", "Option 2"…). The save status ("All changes saved") is in the header.

**Console snippets** (cases T-05, T-08, T-09, T-12): open the browser's DevTools console on an editor page. A Question Set's id is in the address bar as `?set=<id>` when that set is selected in the left Structure tree. `base` below is the start of the editor's address, `/o/<trust>/h/<hospital>`; the snippets work it out from the page you're on.

## 2. Test cases

### T-01: A brand-new question says "New question: won't pre-fill"
1. As **Alex Author** at Hospital A1, create questionnaire **"T23 Source <n>"** (any description).
2. Add a Question Set, a page, and a **Text** question. Set its Question text to **"What is your favourite colour <n>?"**.
3. **Expected:** the question's card shows a grey outlined badge **"New question: won't pre-fill"**, next to the PATIENT badge. (Even while it still has the editor's own text, "Text question": that placeholder never matches by wording, though other HQs may have one too.)
4. Add a **Select One** question with Question text **"Pick a fruit <n>"**, and options **Apple**, **Banana**, **Cherry** (rename the default options; add one if needed).
5. **Expected:** it says "New question: won't pre-fill" too.
6. Click **Publish** → the window says it's ready → **Publish**. **Expected:** "v1 · published", Read only.

### T-02: The same wording in another HQ: "Pre-fills from 1 other HQ", by same wording
1. As Alex, create **"T23 Target <n>"**, with a Question Set (on the set's own page, rename it **"About you"**), a page and a **Text** question worded **"what is your FAVOURITE colour <n>"** (different case, no question mark).
2. Wait for "All changes saved". **Expected:** within a moment the card shows a teal badge **"Pre-fills from 1 other HQ"** (no reload needed).
3. Click the badge. **Expected:** a small list: **"T23 Source <n> · by same wording"**. Click elsewhere to close it; clicking the badge doesn't select the card. **Keyboard:** Tab to the badge (it gets a focus ring), press **Enter**: the list opens; press **Escape**: it closes.
4. Change the Question text to **"What is your favourite food <n>?"** and click away. **Expected:** after the save, the badge becomes **"New question: won't pre-fill"**. Change it back to "What is your favourite colour <n>?": **"Pre-fills from 1 other HQ"** again.
5. Add another **Text** question with the same words, **"What is your favourite colour <n>?"**, and change it to a **Number** (Settings → Change type → Number). **Expected:** after the save, **both** say "New question: won't pre-fill": the number is a different kind of answer, and the text is now used twice in this questionnaire, which never matches by wording (the safeguard in T-03).
6. Delete the Number question. **Expected:** the first goes back to "Pre-fills from 1 other HQ".

### T-03: A text used twice never matches by wording
1. In **T23 Source <n>** you can't edit (published). Instead, in **T23 Target <n>** add two **Text** questions, both worded **"Please give details <n>"**.
2. Create and publish a third questionnaire **"T23 Details <n>"** with one Text question **"Please give details <n>"**.
3. Back in T23 Target <n> (reload it). **Expected:** both "Please give details <n>" questions say **"New question: won't pre-fill"**, because the text is used twice in this questionnaire. Delete one of them: after the save, the other says **"Pre-fills from 1 other HQ"** (T23 Details <n>, by same wording).

### T-04: Choice questions match by wording only with the same option texts
1. In **T23 Target <n>**, add a **Select One** **"Pick a fruit <n>"** with options **apple**, **Banana**, **Cherry** (lower-case "apple" is fine: case is ignored).
2. **Expected:** "Pre-fills from 1 other HQ" → T23 Source <n> · by same wording.
3. Rename option "Cherry" to **"Plum"**. **Expected:** "New question: won't pre-fill" (an option text differs). Rename it back to "Cherry".

### T-05: The same key in another HQ: "by link"
1. In **T23 Target <n>**, click **Add Question Set** (a second set, "Question Set 2"), add a page to it, select the set in the Structure tree and note its id from the address bar (`?set=<target set id>`).
2. Open **T23 Source <n>**, select its Question Set and note its id (`?set=<source set id>`).
3. On the Target editor page, in the console, run (with the two ids):
   ```js
   const base = location.pathname.split("/questionnaires")[0];
   const src = await fetch(`/api${base}/chapters/<source set id>`).then(r => r.json());
   const dst = await fetch(`/api${base}/chapters/<target set id>`).then(r => r.json());
   await fetch(`/api${base}/chapters/<target set id>/content`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify({content: src.content, revision: dst.revision})}).then(r => r.status)
   ```
   **Expected:** `200`.
4. **Reload** the Target editor and open Question Set 2's page. **Expected:** its "What is your favourite colour <n>?" and "Pick a fruit <n>" cards each say **"Pre-fills from 1 other HQ"**, and the list says **"T23 Source <n> · by link"** (the same key wins over the same wording).
5. **Expected, the safeguard again:** Target now uses each of these texts twice (in About you *and* in Question Set 2). So About you's "what is your FAVOURITE colour <n>" and "Pick a fruit <n>" now say **"New question: won't pre-fill"** (same wording, but used twice here), while Question Set 2's still say "by link" (the same key always matches).

### T-06: Another trust's HQs never count
1. Sign in as **Bea Author** (Trust B). Create **"T23 Trust B <n>"** with a **Text** question **"What is your favourite colour <n>?"**, and publish it.
2. Sign in as **Alex Author** and open **T23 Target <n>**'s Question Set 2. **Expected:** "What is your favourite colour <n>?" still says **"Pre-fills from 1 other HQ"**, and the list has only **T23 Source <n>**: never T23 Trust B <n>.
3. As Alex, open the seeded **Demo pre-op HQ** (published), Question Set **About you** → page **Smoking** (open About you in the Structure tree to see its pages). Click the "Do you smoke?" card's badge. **Expected:** every name listed is a Trust A questionnaire (the Demo pre-op HQ of Trust B, and Trust B's "E2E … B" questionnaires, are never listed). The list scrolls if long.
4. As Bea, open Trust B's **Demo pre-op HQ** → a patient question (e.g. Your health → Health → "Do you have diabetes?"; *corrected after test report 1: Trust B's demo HQ may have no Smoking page*). **Expected:** no Trust A questionnaire (no "T23 Source <n>", "T23 Target <n>") in its list.

### T-07: Publish refuses a removed, added or reworded option of a published question (brief focus 1 and 2)
1. As Alex, open **T23 Source <n>** (v1 published). Click **Create new version**. **Expected:** "v2 · draft".
2. Open "Pick a fruit <n>". Delete option **Cherry** (the red bin beside it).
3. Click **Publish**. **Expected:** the window shows "1 problem to fix first" with: **"<Set name> · “Pick a fruit <n>”: the option “Cherry” was removed. A published question's answers can't change: add a new question instead (published in “T23 Source <n>” v1)."**, and **Publish** is disabled.
4. Cancel. Undo the delete (Ctrl/⌘+Z, or add it back: it must be the same option, so undo is easier). Check "Cherry" is back. Add a fourth option **"Date"**. Publish. **Expected:** refused: **"the option “Date” was added"**. Cancel and delete "Date".
5. Rename **"Banana"** to **"Bananas"**. Publish. **Expected:** refused: **"the option “Banana” was reworded to “Bananas”"**. Cancel and rename it back to "Banana".
6. Move "Cherry" up (the up arrow beside it), and change the **Question text** to **"Choose a fruit <n>"**. Publish. **Expected:** ready → **Publish** works: "v2 · published".

### T-08: The same lock across HQs, and Go refuses even without the editor
1. As Alex, open **T23 Target <n>** (still a draft). In Question Set 2, open "Pick a fruit <n>" (copied by key from Source in T-05). Rename option **"Apple"** to **"Green apple"**.
2. Click **Publish**. **Expected:** refused: **"“Pick a fruit <n>”: the option “Apple” was reworded to “Green apple”. A published question's answers can't change: add a new question instead (published in “T23 Source <n>” v2)."** (the newest version of Source that holds the question).
3. Cancel. In the console on the Target editor, run:
   ```js
   const base = location.pathname.split("/questionnaires")[0];
   const qid = location.pathname.split("/questionnaires/")[1];
   const d = await fetch(`/api${base}/questionnaires/${qid}`).then(r => r.json());
   await fetch(`/api${base}/questionnaires/${qid}/publish`, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({versionId: d.questionnaire.versionId})}).then(async r => [r.status, await r.json()])
   ```
   **Expected:** `422`, with `error` "This version can't be published until these problems are fixed." and a `problems` list that includes the "Green apple" problem. Reload: still "v1 · draft".
4. Rename it back to **"Apple"**. Publish (the window). **Expected:** it publishes.

### T-09: A key used twice in one HQ is refused, naming both Question Sets (brief focus 3)
1. As Alex, create **"T23 Twice <n>"**: Question Set "First" with a page and a Text question **"Height <n>"**; then **Add Question Set** "Question Set 2" with a page and any question. Note both sets' ids.
2. On this editor page, in the console, copy First's content into Question Set 2 (the snippet in T-05 with `<source set id>` = First, `<target set id>` = Question Set 2). **Expected:** `200`. Reload.
3. Click **Publish**. **Expected:** refused: **"Question Set 2 · The question ID q_… is used “Height <n>” in “First” and “Height <n>” in “Question Set 2”. Each question needs its own ID."** Publish disabled.
4. The console publish snippet (T-08 step 3) on this HQ. **Expected:** `422`, the same problem in `problems`.
5. Delete the question in Question Set 2 and add a new one. Publish. **Expected:** it publishes.

### T-10: Always ask fresh: switch, badge, save, reload, type change, new version (brief focus 4)
1. As Alex, create **"T23 Fresh <n>"** with a **Select One** "Are you pregnant <n>?" (Yes / No, any texts).
2. Select it. **Expected:** in Settings, under **Clinical**, a switch **"Always ask fresh"** with a line explaining it, off.
3. Switch it on. **Expected:** the card's teal/grey pre-fill badge is replaced by an orange **"Always ask fresh"** badge; "All changes saved".
4. Reload the page. **Expected:** the badge and the switch are still on.
5. Settings → **Change type** → **Select Many**. **Expected:** still on.
6. Switch on **Clinical**. **Expected:** the "Always ask fresh" switch and badge disappear (clinician-only questions are never pre-filled), and no pre-fill badge shows either. Switch Clinical off: they're back, still on.
7. Publish. **Expected:** the read-only v1 still shows the badge. **Create new version**. **Expected:** v2 shows the badge, and its switch is still on.
8. In v2, add a **Statement**, a **Group** and a **Calculation**. (A Group made Repeatable with a Text question in it, both keeping the editor's default names "Group" and "Text question", says **"New question: won't pre-fill"** even if another HQ has the same defaults.) Select each. **Expected:** no "Always ask fresh" switch on any of them. Switch the Group's **Repeatable** on: now it **does** offer "Always ask fresh" (a repeating list is an answer); a question you add *inside* it doesn't, and has no pre-fill badge (its answers travel in the list's rows).

### T-11: Nothing pre-fill related in a clinician Question Set
1. In T23 Fresh <n> v2, **Add Question Set**; on the set's own page (select the set in the Structure tree) set **Who is it for?** to the clinician option; add a page and a Text question.
2. **Expected:** the question's card has no "Pre-fills…", "New question" or "Always ask fresh" badge, and its Settings have no "Always ask fresh" switch.

### T-12: Who sees the label
1. As **Sue Super** (Hospital A1), open **T23 Source <n>**. **Expected:** read-only, and the cards show the pre-fill badges (e.g. "Pre-fills from 1 other HQ" on "What is your favourite colour <n>?"); clicking one lists the HQs.
2. As **Cara Clinician**, open T23 Source <n>. **Expected:** read-only; no pre-fill badges (but no error either). Any "Always ask fresh" badge (T23 Fresh <n>) still shows.
3. As Cara, in the console on that editor page:
   ```js
   const base = location.pathname.split("/questionnaires")[0];
   const qid = location.pathname.split("/questionnaires/")[1];
   await fetch(`/api${base}/questionnaires/${qid}/question-keys`).then(r => r.status)
   ```
   **Expected:** `403`.
4. As **Bea Author**, run the same fetch with the Trust A ids written in (copy Alex's editor address: `/api/o/<trust A>/h/<hospital A1>/questionnaires/<qid>/question-keys`). **Expected:** `404`. With Trust B's hospital in the address and Trust A's questionnaire id: `404`.

### T-13: Reload, deep link and Back
1. As Alex, in **T23 Target <n>**, select Question Set 2's page; the address has `?set=…&page=…`. Copy it.
2. Open the copied address in a new tab. **Expected:** the same page, with the badges shown.
3. Go to Questionnaires and press the browser's **Back**. **Expected:** back on the editor page, badges shown.

### T-14: A published question's kind of answer and unit are frozen (after verification 1)
1. As Alex, create **"T23 Shape Source <n>"** with a **Number** question **"Weight <n>"** whose Unit (Settings) is **kg**, and a **Select One** **"Colour <n>"** (any two options). Publish.
2. **Create new version**. Change "Colour <n>" to **Select Many** (Settings → Change type). Click **Publish**. **Expected:** refused: **"“Colour <n>”: its kind of answer changed from single choice to multiple choice. A published question's answers can't change: add a new question instead (published in “T23 Shape Source <n>” v1)."** Cancel and change it back to **Select One**.
3. Change "Weight <n>"'s Unit to **lb**. Publish. **Expected:** refused: **"its unit changed from “kg” to “lb”"**. Set it back to **kg**.
4. Change "Weight <n>" to a **Date** (Change type). Publish. **Expected:** refused: **"its kind of answer changed from number to date"**. Change it back to Number with unit kg.
5. Publish. **Expected:** it publishes ("v2 · published").

### T-15: Grids and ratings: frozen once published, and matched by wording only when they're the same
1. As Alex, create **"T23 Grid Source <n>"** with a **Grid** **"How often do you <n>"** (rows **Run**, **Swim**; columns **Never**, **Always**) and a **Rating** **"How bad is your pain <n>?"** (From **0**, To **10**). Publish.
2. Create **"T23 Grid Target <n>"** with a Grid of the same text, rows and columns, and a Rating of the same text, 0 to 10. **Expected:** both say **"Pre-fills from 1 other HQ"** (T23 Grid Source <n> · by same wording).
3. In the Target, change the Rating's **To** to **5**, and rename the Grid's row "Swim" to **"Cycle"**. **Expected:** after the saves, both say **"New question: won't pre-fill"**.
4. Back in **T23 Grid Source <n>**: **Create new version**, delete the Grid's row **Swim**, and change the Rating's To to **5**. Publish. **Expected:** refused with **"the row “Swim” was removed"** and **"its scale changed from 0 to 10 by 1 to 0 to 5 by 1"**. Put them back (undo the row's deletion with Ctrl/⌘+Z: a row added again by hand gets a new ID, which is another change), set To back to 10, and Publish: it publishes.

### T-16: Signatures, uploads and the BMI block never pre-fill
1. As Alex, create **"T23 Sign A <n>"** with a **Signature** ("Please sign below") and a **File upload** question **"Upload your letter <n>"**. Publish.
2. Create **"T23 Sign B <n>"** with the same two questions (same texts).
3. **Expected:** in both questionnaires, the Signature and File upload cards have **no** pre-fill badge ("Pre-fills…" or "New question") and no "Always ask fresh" badge; selecting each shows **no** "Always ask fresh" switch. A **BMI** block shows no pre-fill badge either.

### T-17: A published chapter in an unusual shape doesn't stop publishing
1. As Alex, create **"T23 Odd <n>"** with one Question Set and a page. Note the set id (`?set=`).
2. In the console on its editor page, save a Grid whose columns are plain strings (a shape SurveyJS accepts):
   ```js
   const base = location.pathname.split("/questionnaires")[0];
   const id = "<set id>";
   const dst = await fetch(`/api${base}/chapters/${id}`).then(r => r.json());
   await fetch(`/api${base}/chapters/${id}/content`, {method: "PUT", headers: {"Content-Type": "application/json"}, body: JSON.stringify({revision: dst.revision, content: {pages: [{name: "p_odd", elements: [{type: "matrix", name: "q_odd_<n>", title: "Odd grid <n>", rows: ["A"], columns: ["Never", "Sometimes"]}]}]}})}).then(r => r.status)
   ```
   **Expected:** `200`. Reload, then **Publish**. **Expected:** it publishes (no error).
3. Then publish any other draft in Trust A (e.g. a new small questionnaire). **Expected:** it publishes; the editor's labels still show (no "Something went wrong").

### T-18: A date's format and a text question's format are frozen, and part of the wording match (after verification 2)
1. As Alex, create **"T23 Format Source <n>"** with a **Date** question **"Date of your operation <n>"** (Settings: format **Day, month and year**) and a **Text** question **"Contact <n>"** (Settings → Format: **Any text**). Publish.
2. Create **"T23 Format Target <n>"** with the same two questions, same texts and formats. **Expected:** both say **"Pre-fills from 1 other HQ"** (T23 Format Source <n> · by same wording). Change the Target's date to **Year only** and the text's Format to **Email address**. **Expected:** after the saves, both say **"New question: won't pre-fill"**.
3. In **T23 Format Source <n>**: **Create new version**. Change the date's format to **Year only**. Click **Publish**. **Expected:** refused: **"“Date of your operation <n>”: its date format changed from day, month and year to year only. A published question's answers can't change: add a new question instead (published in “T23 Format Source <n>” v1)."** Change it to **Month and year**: refused, "…from day, month and year to month and year". Put it back to Day, month and year.
4. Change "Contact <n>"'s Format to **Email address**. Publish. **Expected:** refused: **"its answer format changed from any text to email address"**. To **Phone number**: refused, "…to the pattern “…”" (the phone pattern). Put it back to **Any text**, then Publish: it publishes.
5. In the same HQ, **Create new version** again. Add a **Group**, switch on **Repeatable**, add a **Select One** inside it with two options, and Publish (v3). **Create new version**; reword one option of the question inside the group, and reword the question's own text. Publish. **Expected:** exactly **one** problem, about the question inside the group ("the option “…” was reworded to “…”"), not a second one about the group's column. Put the option back: it publishes (the question's own text may change).

## 3. Regression

- **R-01 (J2):** a draft whose condition tests a deleted question is still refused with its logic problem, and publishes once fixed (e.g. make a Text question visible only when another question is answered, then delete that other question: "1 logic problem").
- **R-02 (patient view, phone size):** create an episode on **T23 Fresh <n>** (v2 published, with "Always ask fresh" on) for a made-up patient at Hospital A1, open the patient's invite link from **/qa/messages** in a **390×844** window, sign in with the date of birth. **Expected:** the questionnaire looks as before: no "Always ask fresh", no "Pre-fills" text anywhere; the question can be answered and the HQ submitted.
- **R-03 (POA Summary):** as Cara, validate that episode's sets and open the POA Summary. **Expected:** as before, nothing new on it.
- **R-04 (preview):** Preview a set of T23 Fresh <n> as Patient and as Clinician. **Expected:** as before; no badges in the preview.
- **R-05 (new version):** Create new version of the seeded **Demo pre-op HQ** and publish it unchanged. **Expected:** it publishes (its keys and options are the same as published).

## 4. Out of scope (don't report)

- Pre-fill itself: no answers are copied yet (S25). The label only says what *will* pre-fill.
- The question library, "Insert from library" and "Same as…" (S24).
- The editor offering no way to give a question an existing key (that's S24's "Same as…"); the console is the only way in this session.
- Copying a question or a Question Set inside an HQ gives new IDs (expected).
- Grids (matrix rows and columns), rating scales and Yes/No switches have no "options" for the lock in this session.
- The many "E2E …" questionnaires in the lists (left by automated tests).
