# S32 test plan: the Clinical Summary element

For a tester who uses only the browser. Everything is made up; no real patient data.

## Setup

**In your own terminal** (the automatic run's conductor does this for you):

```sh
docker compose up -d db
make migrate seed
make dev        # leave it running
```

Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

| Who | Role | Use for |
| --- | --- | --- |
| **Alex Author** | super clinician, the whole of Trust A | building and publishing questionnaires |
| **Cara Clinician** | clinician, Hospital A1 only | episodes, Validate, POA Summary |
| **Bea Author** | super clinician, the whole of Trust B | another trust |
| **Sue Super** | super clinician, Hospital A1 only (views but can't change the trust's questionnaires) | read-only editor |

Pick **Trust A → Hospital A1** after signing in (Bea: Trust B's hospital).

**Helpers:**
- **Fill in for the patient:** on an episode's page, the button **Fill in for the patient** opens the patient's form (as staff, on their behalf). Or open the patient's own invite link from **http://localhost:3000/qa/messages** (captured texts and emails), with date of birth **17/05/1980** if you made the patient with that date.
- **A questionnaire made before S32 (the old page switch).** The page switch is gone from the editor, so make two such questionnaires with this snippet. Sign in as **Alex Author**, open **Questionnaires** at Hospital A1, then run this in the page (browser console / JavaScript tool). It returns the two names it made: `Old switch draft HH:MM:SS` (a draft) and `Old switch published HH:MM:SS` (published). Each has one Question Set "About you" with one page "Old heart", where "Old: do you get chest pain?" Yes gives the note "Old angina note", and the page's old switch is on.

```js
(async () => {
  const base = "/api" + location.pathname.match(/\/o\/[^/]+\/h\/[^/]+/)[0];
  const j = async (method, path, body) => {
    const r = await fetch(base + path, { method, headers: { "Content-Type": "application/json" }, body: body && JSON.stringify(body) });
    if (!r.ok) throw new Error(`${method} ${path} ${r.status} ${await r.text()}`);
    return r.status === 204 ? null : r.json();
  };
  const content = { pages: [{ name: "p_oldheart", title: "Old heart", clinicalSummary: true, elements: [{
    type: "radiogroup", name: "q_oldangina", title: "Old: do you get chest pain?", isRequired: true,
    choices: [{ value: "o_y", text: "Yes", clinicalOutputs: [{ id: "out_oldangina", codes: [], note: { text: "Old angina note", category: "Cardiovascular" } }] }, { value: "o_n", text: "No" }],
  }] }] };
  const make = async (name, publish) => {
    const q = await j("POST", "/questionnaires", { name, description: "Made up: the old page switch (S32 testing)" });
    const c = await j("POST", `/questionnaires/${q.id}/chapters`, { name: "About you" });
    const { revision } = await j("GET", `/chapters/${c.id}`);
    await j("PUT", `/chapters/${c.id}/content`, { content, revision });
    if (publish) {
      const { questionnaire } = await j("GET", `/questionnaires/${q.id}`);
      await j("POST", `/questionnaires/${q.id}/publish`, { versionId: questionnaire.versionId });
    }
    return name;
  };
  const stamp = new Date().toISOString().slice(11, 19);
  return [await make(`Old switch draft ${stamp}`, false), await make(`Old switch published ${stamp}`, true)];
})()
```

**What "summary" means here:** a **Clinical Summary** is a yellow box only clinicians see. It lists the disclosure notes from the part of the page it covers, above a "Clinical comments" box. It covers its Group or Section if it's inside one, otherwise the whole page. **Each note shows once, under the nearest summary** (Rahul's answer): a note from a question inside a Group that has its own summary shows in the Group's summary, not in the page's.

## Build the test questionnaire (used by most cases)

As **Alex Author**, Hospital A1 → **Questionnaires** → **New Questionnaire**, name it `S32 test <time>`. Then:
1. **Add Question Set** (it's "Question Set 1"), **Add Page**. Rename the page **Heart** (the pencil by the page name).
2. Add a **Yes / No** question: text **Chest pain?** In its **Disclosures** tab, on **Yes**, add a note **Angina** (any category).
3. Add a **Group**, named **Breathing**. Select the Group, and in its own Add bar add a **Yes / No**: **Asthma?**, with the note **Asthma** on Yes.
4. Inside the Breathing Group, add a second **Group** named **Inhalers**, with a **Yes / No**: **Uses an inhaler?**, note **Inhaler** on Yes.
5. Add a **Section** (on the page, not in a Group) named **Blood**, with a **Yes / No**: **Anaemia?**, note **Anaemia** on Yes.
6. Add a second page **Lungs** with a **Yes / No**: **Cough?**, note **Cough** on Yes. (No summary goes on this page.)

## Test cases

### The editor

**T-01 · The page switch is gone; Clinical Summary is in the Add bar**
1. Open the Heart page of the test questionnaire.
2. Look at the top of the page area.
3. Look at the page's Add content bar.

Expected: there's no "Clinical summary" switch by "Clinical Page". The Add bar has a **Clinical Summary** button.

**T-02 · A summary inside a Group**
1. Select the **Breathing** Group; in its own Add bar click **Clinical Summary**.

Expected: a card appears inside the Group, after Asthma? and the Inhalers Group, with a purple **Clinician** badge, the title **Breathing summary**, the line **Covers the Group “Breathing”**, and a "Clinical comments" box. The Settings panel opens on the left: **Clinical Summary**, a **Title** box showing "Breathing summary", and a line explaining what it covers. There's no Disclosures tab and no Required or Clinical switch.

**T-03 · A summary in a Section, and one on the page**
1. Select the **Blood** Section; add a **Clinical Summary** in it.
2. Click an empty part of the page (nothing selected); in the page's Add bar at the bottom click **Clinical Summary**.

Expected: the Section's card says **Blood summary**, **Covers the Section “Blood”**. The page's card is at the end of the page: **Heart summary**, **Covers the page “Heart”**. "All changes saved" shows in the header.

**T-04 · Changing a summary's title**
1. Select the **Heart summary** card. In Settings change **Title** to `Heart overview`, then click elsewhere.
2. Rename the Breathing Group to `Breathing and lungs` (select the Group, Settings → Name).
3. Select the Heart overview card again; empty its Title box and click elsewhere.

Expected: 1: the card's title is "Heart overview". 2: the Group's summary card now says **Breathing and lungs summary** and "Covers the Group “Breathing and lungs”" (an untitled summary follows the name). 3: the page summary goes back to **Heart summary**. Then set it to `Heart overview` again for the next cases, and rename the Group back to `Breathing`.

**T-05 · Not offered inside a repeating group**
1. Add a new **Group** on the Lungs page, select it, and in Settings switch **Repeatable** on.
2. Look at the repeating Group's own Add bar.
3. Select any existing summary card, open its **⋯ More actions** menu.

Expected: 2: **Clinical Summary** is greyed out (disabled) in the repeating Group's Add bar. 3: "Move into" never lists the repeating Group. Delete the repeating Group afterwards.

**T-06 · Reload keeps everything**
1. Reload the editor page.

Expected: the three summaries are where they were, with the same titles and "Covers…" lines.

**T-07 · Copy a summary**
1. On the Breathing summary card click **Copy**.

Expected: a second summary card appears in the Breathing Group (titled "Breathing summary" as well). Keep it for T-08.

### Publishing

**T-08 · Two summaries in the same Group block Publish, naming the Question Set**
1. With the copied summary still in the Breathing Group, click **Publish** in the header.
2. Read the problems.
3. Cancel. Delete the copied summary (its card's bin, then **Delete**). Click **Publish** again.

Expected: 2: the dialog says "1 problem to fix first" and lists **Question Set 1 · Two clinical summaries cover the same Group “Breathing”: keep one**; Publish is disabled. 3: no problem is listed; **Publish** works and the header says "v1 · published".

**T-09 · Two summaries on the same page block Publish too**
1. Create a new version (**Create new version**). On the Lungs page add two **Clinical Summary** elements on the page itself (not in a Group).
2. Click **Publish**.

Expected: "Question Set 1 · Two clinical summaries cover the same page “Lungs”: keep one". Delete one of them; the problem goes. (Delete both before going on, or leave one: either is fine, but the later cases assume v1.)

**T-10 · A summary inside a repeating group blocks Publish**
1. In the v2 draft, add a **Group** named `Meds` on the Lungs page, select it and add a **Clinical Summary** inside it (allowed: it isn't repeating yet).
2. Select the Meds Group and switch **Repeatable** on.
3. Click **Publish**.

Expected: 3: "Question Set 1 · The clinical summary “Meds summary” is inside a repeating group: move it out", and Publish is disabled. Leave the v2 draft unpublished (the later cases use v1).

### The clinician's view

**T-11 · Preview, as a clinician: each box where it was placed, each listing only its own notes**
1. As Alex, open the published v1 → **Preview** on Question Set 1. Switch the view to **Clinician** (Desktop).
2. Look at the empty boxes.
3. Answer **Yes** to Chest pain?, Asthma?, Uses an inhaler? and Anaemia?.
4. Go to page 2 (Lungs) and answer **Yes** to Cough?.

Expected: 2: three yellow boxes on the Heart page: **Breathing summary** inside the Breathing Group (left column, below the questions), **Blood summary** inside the Blood Section, **Heart overview** for the page (right column). Empty ones say "No disclosures in this group yet." / "…in this section yet." / "No disclosures on this page yet." 3: Breathing summary lists **Asthma** and **Inhaler** (the nested Inhalers Group has no summary of its own, so the nearest is Breathing's); Blood summary lists **Anaemia**; Heart overview lists only **Angina**. No note shows twice. 4: the Lungs page has no box at all.

**T-12 · The patient view never shows a summary**
1. In the same preview switch to **Patient**, at **Phone**, then **Tablet**, then **Desktop**.

Expected: no yellow box, no "summary" title and no "Clinical comments" box on any screen, at any size.

### Episodes: Validate and the POA Summary

**T-13 · Validate: a comment in each summary**
1. As **Cara Clinician** (Hospital A1): **Episodes → New episode**, a made-up patient (e.g. Sam S32-<time>, born 17/05/1980, any mobile or email), on **S32 test … v1**. Create.
2. **Fill in for the patient**: Yes to Chest pain?, Asthma?, Uses an inhaler?, Anaemia? and Cough?; finish and send.
3. Back on the episode: **Validate** Question Set 1.
4. Type `Uses a blue inhaler` in the Breathing summary's comments, `Iron tablets` in the Blood summary's, `Refer to cardiology` in Heart overview's. Wait for "Saved". Click **Validate**.

Expected: 3: the same three boxes as in T-11, where the author placed them, each listing only its own notes. 4: saved and validated ("Validated by Cara Clinician").

**T-14 · The POA Summary: one block per summary**
1. Open the episode's **POA Summary** (Validated summary tab).
2. Look at Question Set 1.
3. Switch to **Patient answers**.

Expected: 2: under Question Set 1: **Breathing summary** (Asthma, Inhaler; "Clinical comments: Uses a blue inhaler"), **Blood summary** (Anaemia; "Clinical comments: Iron tablets"), **Heart overview** (Angina; "Clinical comments: Refer to cardiology"), and for the Lungs page a block **Lungs Summary** with **Cough** (no summary covers it, so it's named after the page) and no comments. No note appears in two blocks. 3: the same blocks and notes, without the comments.

**T-15 · Clickable notes still work (S27)**
1. On the Validated summary tab click the note **Asthma**.
2. Use the **POA Summary** link (or the browser's Back).

Expected: 1: Validate opens on the Heart page with "Asthma?" scrolled into view and highlighted. 2: back on the POA Summary, same tab, with the note still on screen.

**T-16 · A summary with only a comment still shows**
1. New episode on S32 test v1; fill in for the patient: **No** to everything; send.
2. Validate: type `Checked, all clear` only in Heart overview's comments; Validate.
3. Open the POA Summary.

Expected: a block **Heart overview** with only "Clinical comments: Checked, all clear"; no empty blocks for the other summaries.

**T-17 · A note on a page with no summary still shows**
Covered in T-14 (the **Lungs Summary** block). Also check: on the T-13 episode, the Lungs page's Cough note is listed (and linked) in the POA Summary.

### Content made before S32 (the old page switch)

**T-18 · Opening an old draft shows a summary element at the end of the page**
1. Run the snippet in Setup (as Alex). Open **Old switch draft …** in the editor, Question Set "About you", page "Old heart".
2. Look at the cards. Select the summary card.
3. Change anything small (e.g. the question's text), wait for "All changes saved", and reload.

Expected: 2: after "Old: do you get chest pain?" there's a summary card **Old heart summary**, "Covers the page “Old heart”", with the Clinician badge; no switch anywhere. 3: still there after the reload.

**T-19 · An old published version: Validate and POA Summary still show its summary and saved comments**
1. As **Cara**: New episode on **Old switch published …** v1. Fill in for the patient: **Yes**; send.
2. Validate "About you": the box **Old heart summary** at the end of the page lists **Old angina note**. Type `Old comment kept` in its comments. Validate.
3. Open the POA Summary.
4. Reload Validate (open it again from the episode).

Expected: 2: the box is there with the note. 3: a block **Old heart summary** with "Old angina note" and "Clinical comments: Old comment kept". 4: the comment is still in the box.

**T-20 · Opening an old published version in the editor changes nothing**
1. As Alex, open **Old switch published …** (v1, published, read-only) in the editor, page "Old heart".

Expected: the summary card shows at the end of the page (read-only: no Add bar, no delete), and the header says read-only. No "Saving…" appears.

### Refused paths and other trusts

**T-21 · A read-only author can't add a summary**
1. Sign in as **Sue Super** (views but can't change the trust's questionnaires), open **S32 test …**'s draft (or v2 draft) in the editor.

Expected: no Add bar, so no Clinical Summary button; the summary cards show with no Copy/Delete actions.

**T-22 · Another trust can't see the questionnaire**
1. Sign in as **Bea Author** (Trust B), Questionnaires.

Expected: S32 test … isn't listed; its editor URL (copied from Alex's session) shows not found.

### Phone size

**T-23 · The patient's own form at phone size (390×844) never shows a summary**
1. Make a new episode on S32 test v1 for a patient with an email; open their invite from **/qa/messages** in a window sized 390×844; Continue, date of birth 17/05/1980.
2. Answer every question Yes, page by page, to the end; send.

Expected: no summary title, yellow box or "Clinical comments" on any screen; no sideways scrolling.

## Regression (earlier sessions)

**R-01 · J5's POA Summary basics (S20, S27):** on any reviewed episode, the POA Summary shows both tabs, Print, status change and a General notes comment, as before.

**R-02 · Corrections marked (plan-workflow):** in T-13's episode, change Asthma? to No in Validate and validate again: the POA Summary's Validated tab drops Asthma from the Breathing block, and lists "Changed by the clinician on Heart" with the change.

**R-03 · Pre-fill markers (S26):** a returning patient's pre-filled answers still show "Pre-filled on <page>" on the POA Summary (any episode where pre-fill happened; skip if none to hand).

**R-04 · Statements and other cards unchanged:** in the editor, add a Statement and a Text question: they look and save as before; "Change type" on the Text question still works.

**R-05 · The publish checks still catch a logic problem (J2):** a question whose condition points at a deleted question still blocks Publish with its message.

## Out of scope (don't report)

- Videos, the content library and Information blocks (S33–S35).
- Which notes a disclosure makes (unchanged).
- Capture blocks (S31 was dropped).
- The two-column clinician layout itself (a Group holding a clinician-only question still moves to the right column, as before).
- The 503 shown for 204 responses in Claude in Chrome's network log (known, decisions.md §4).
