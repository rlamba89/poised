# S35 manual test plan: the Information element

**Changes after verification 1 (retest these):**
- **An answer shown back in a block is plain text** (F1). The setup script's set now also has a question "Anything else we should know?" and a block "You told us: **{q_note}**" (re-run the script on a new questionnaire). New case T-23.
- **Numbered lists keep their numbers** (F2): new case T-24.
- **Named characters** such as `&eacute;` in an old statement show as the character (F5): new case T-25.
- An old statement's **translations keep their lines** when it's first edited (F4): new case T-26.
- Survey-level HTML (`completedHtml` and the like) is dropped: nothing a tester can reach in the browser (the editor never writes it); unit-tested.

**Changes after test report 1:** F1 fixed (Information headings are now the same size in the editor's preview and in every survey: about 20px for `#`, 18px for `##`, 16px for `###`). T-13, T-18 and the new T-20 check this. A **blank line** in the text is now a gap, as in old statements (T-21). Hand-made HTML opens in the Text box with its words on separate lines (T-22).

For a tester who has never seen the code. Use only the browser. Report each case as Pass, Fail or Blocked, with what you saw.

**What changed:** the editor's **Statement** block is now called **Information**. Authors type simple marks in its text (`# Heading`, `**bold**`, `*italic*`, `- list`, `1. list`, `[words](https://…)`, `![what it shows](https://…)`) and see a live preview. Patients and clinicians see it formatted. Anything else typed (HTML, a `javascript:` link) shows as plain text, and nothing in a questionnaire can run code in the browser. Old statements look and work as before.

## 1. Setup

1. In your own terminal (an automatic run does this for you):
   ```sh
   docker compose up -d db && make migrate seed && make dev
   ```
   Leave `make dev` running.
2. Open **http://localhost:3000** in Chrome. Sign in from the stub login page by clicking a user's name:
   - **Alex Author** (super clinician, the whole of Trust A): edits questionnaires.
   - **Cara Clinician** (clinician, Hospital A1): creates episodes and validates.
   - **Val Clinician** (clinician, Hospital A2 only).
   - **Sue Super** (super clinician, Hospital A1 only: can view but not change the trust's questionnaires).
   - **Bea Author** (super clinician, Trust B).
3. **Captured messages:** http://localhost:3000/qa/messages lists the invite texts and emails (nothing is really sent). The patient's invite link is in there.
4. **Phone size:** for every patient case use Chrome's device toolbar (or a window) at **390 × 844**.
5. **Making the test questionnaire ("S35 test HQ").** Old statements and hand-made HTML can't be typed in the editor, so you add them with a short script:
   1. Sign in as **Alex Author** (Hospital A1). Questionnaires → **New questionnaire**, name it `S35 test HQ <your initials + time>`, create it. You're now in its editor (the address ends `/questionnaires/<id>`).
   2. Open Chrome's DevTools → **Console**, paste the script below, press Enter. The page reloads with a new Question Set **"Before you come in"**, page **"Your visit"**, holding:
      - **i_old**: an old statement ("Please arrive 30 minutes early." / "Bring a dressing gown & slippers."), saved the way statements were before this session (and the way the incumbent import writes them);
      - **i_bad**: HTML nobody could type in the editor (a picture with `onerror`, a `javascript:` link "bad link", a `<script>`, a paragraph "Click me" with `onclick`). Each would set `window.__xss` if it ever ran;
      - **i_parking**: a formatted block: heading "Parking", bold "car park B", a two-item list, links "the map" (https) and "call us" (tel:), and a wide picture of a mountain (from Wikimedia, https), with a German translation of its first two lines;
      - a required question "Will someone drive you home?";
      - a question "Anything else we should know?" (q_note) and a block **i_echo**, "You told us: **{q_note}**", which shows the answer back (added after verification 1).

```js
(async () => {
  const CONTENT = {"pages": [{"name": "p_info", "title": "Your visit", "elements": [{"type": "html", "name": "i_old", "html": "<p>Please arrive 30 minutes early.</p><p>Bring a dressing gown &amp; slippers.</p>"}, {"type": "html", "name": "i_bad", "html": "<p>Imported text</p><img src=x onerror=\"window.__xss=1\"><a href=\"javascript:window.__xss=2\">bad link</a><script>window.__xss=3</script><p onclick=\"window.__xss=4\">Click me</p>"}, {"type": "html", "name": "i_parking", "markdown": {"default": "## Parking\nPark in **car park B**.\n- Pay at the machine\n- Keep your ticket\nSee [the map](https://example.org/map) or [call us](tel:+10000000000).\n![The car park entrance](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Fronalpstock_big.jpg/1280px-Fronalpstock_big.jpg)", "de": "## Parken\nParken Sie im **Parkhaus B**."}}, {"type": "radiogroup", "name": "q_lift", "title": "Will someone drive you home?", "isRequired": true, "choices": [{"value": "o_yes", "text": "Yes"}, {"value": "o_no", "text": "No"}]}, {"type": "comment", "name": "q_note", "title": "Anything else we should know?"}, {"type": "html", "name": "i_echo", "markdown": "You told us: **{q_note}**"}]}]};
  const [, base, qid] = location.pathname.match(/^(\/o\/[^/]+\/h\/[^/]+)\/questionnaires\/([^/?#]+)/);
  const call = async (method, path, body) => {
    const r = await fetch("/api" + base + path, { method, headers: { "Content-Type": "application/json" }, body: body && JSON.stringify(body) });
    if (!r.ok) throw new Error(method + " " + path + " " + r.status);
    return r.status === 204 ? null : r.json();
  };
  const { id } = await call("POST", `/questionnaires/${qid}/chapters`, { name: "Before you come in" });
  await call("PATCH", `/chapters/${id}`, { audience: "patient" });
  const { revision } = await call("GET", `/chapters/${id}`);
  await call("PUT", `/chapters/${id}/content`, { content: CONTENT, revision });
  location.reload();
})();
```

   If the script fails, report it as Blocked with the console's message.

## 2. Test cases

### The editor

**T-01: Statement is now Information.**
1. As Alex, open the S35 test HQ, Question Set "Before you come in", page "Your visit".
2. Look at the page's **Add content** bar (below the cards).

Expected: there's an **Information** button (with an "i" icon) where Statement used to be, and no "Statement" button anywhere.

**T-02: An old statement looks as before and opens as its plain text.**
1. Look at the first card (i_old).
2. Click its text to select it. Look at the Settings panel on the left.

Expected: the card shows "Please arrive 30 minutes early." and "Bring a dressing gown & slippers." on two lines, as plain text. Settings' header says **Information**. The **Text** box holds exactly those two lines (no `<p>` tags, `&` not `&amp;`). Below it: **How to format** (a short list of marks), a **Preview** showing the same two lines, and the **Clinical** switch.

**T-03: Editing an old statement keeps its text.**
1. With i_old selected, click at the end of the Text box and add a third line: `**Thank you**`.
2. Watch the Preview while you type, then click outside the box (or press Tab).
3. Reload the page and open i_old again.

Expected: the Preview shows "Thank you" in bold as you type. After leaving the box, the card shows the two old lines plus **Thank you** in bold, and the header says "All changes saved". After reload, the Text box shows the three lines as you typed them (the two old ones unchanged).

**T-04: Hand-made HTML is cleaned in the editor too.**
1. Look at the second card (i_bad). In the Console, type `window.__xss` and press Enter.

Expected: the card shows "Imported text", "bad link" and "Click me" as plain text. There's no broken-picture icon, and "bad link" isn't a link. `window.__xss` is `undefined`.

**T-05: A new Information block with every mark (the live preview).**
1. Click **Information** in the page's Add bar (the one below all the cards). A new card appears, selected.
2. Replace the Text box's contents with (copy all of it):
   ```
   # Fasting
   Do **not** eat after *midnight*.
   - Water is fine
   - Black coffee is fine
   1. Shower in the morning
   2. Remove nail varnish
   [Our fasting page](https://example.org/fasting)
   Email [the team](mailto:poa@example.org)
   ![A mountain](https://upload.wikimedia.org/wikipedia/commons/thumb/3/3f/Fronalpstock_big.jpg/1280px-Fronalpstock_big.jpg)
   ```
3. Watch the Preview, then click outside the box.

Expected: the Preview shows, as you type: the heading "Fasting"; "not" in bold and "midnight" in italics; a bulleted list of two; a numbered list of two; "Our fasting page" and "the team" as links; the mountain picture, as wide as the preview and no wider. After leaving the box, the card on the canvas shows the same, and "All changes saved".

**T-06: Typed HTML and unsafe links show as text (no pop-up).**
1. In the same block, add these three lines at the end, then click outside the box:
   ```
   <script>alert(1)</script>
   [x](javascript:alert(1))
   <img src=x onerror=alert(1)>
   ```
2. Check the Preview, the card and the Console (`window.__xss`, and any errors).

Expected: no pop-up, ever. All three lines show exactly as typed, as plain text, in the Preview and on the card ("x" is not a link; there's no broken picture). An orange note under the hints says it's "Shown as text, not as a link or picture: javascript:alert(1)…" and what a link and a picture must start with. Nothing new in the Console beyond the usual noise.

**T-07: Other refused links and pictures.**
1. Add a line `[plain web](http://example.org)` and a line `![old picture](http://example.org/a.png)`. Leave the box.

Expected: both show as the text typed (not a link, not a picture), and the orange note lists `http://example.org` and `http://example.org/a.png`.

**T-08: Undo and the Clinical switch.**
1. Use the editor's Undo once (its Undo button, or Ctrl/Cmd+Z with no text box focused).
2. Turn the block's **Clinical** switch on, then off.

Expected: Undo brings back the text before your last edit (card and Text box). The Clinical switch works as for other blocks (a "Clinician" marker on the card while on).

**T-09: Translating an Information block.**
1. Click **Translate** (top right), choose **German**.
2. Find the rows whose field badge says **INFORMATION**.
3. In the German box of your new "Fasting" block, type `# Nüchtern` and a second line `Nichts **essen** nach Mitternacht.`, then click outside.
4. Close Translate, click **Preview**, set **Language** to German.

Expected: in Translate, the old statement's row shows its two lines in English as plain text; the "Parking" block already has German (`## Parken`…); the English column shows the marks as typed (e.g. `**car park B**`), so the translator can keep them. In the Preview in German: the Parking block shows the heading "Parken" and "Parkhaus B" in bold; your block shows the heading "Nüchtern" and "essen" in bold; the old statement (not translated) shows in English. Back in English, everything is English. (Patients can't choose a language yet: that's known, not a failure.)

**T-10: Preview at desktop size.**
1. In Preview (English), scroll through "Your visit".

Expected: every block formatted as in the editor, the mountain pictures no wider than the form, links underlined in teal, i_bad shows only its words, no pop-up, `window.__xss` is `undefined`.

**T-11: Publish.**
1. Click **Publish** → **Publish**.

Expected: it publishes ("v1 · published"); no problem is listed about the Information blocks (an unsafe link isn't a publish problem; it just shows as text).

**T-12: Who can change it.**
1. Sign in as **Sue Super** (Hospital A1), open the S35 test HQ (published; or any draft of the trust).
2. Select an Information card.

Expected: Sue can see the block and its preview, but the Text box is disabled (greyed) and she can't add or change blocks (as for every other block: her role can't change the trust's questionnaires).

### The patient (phone size, 390 × 844)

**T-13: The patient sees Information formatted.**
1. As **Cara Clinician** (Hospital A1), create a **New episode** for a made-up patient (any name, date of birth, an email you make up), with the **S35 test HQ v1** as the Health questionnaire.
2. Open http://localhost:3000/qa/messages, find the email by its address, open the invite link **in a 390 × 844 window**. Continue, type the date of birth, Continue.
3. Start **Before you come in**.

Expected: the old statement's two lines (and "Thank you" in bold, from T-03). Headings are no bigger than the set's title ("Fasting" about 20px, "Parking" about 18px). i_bad's words only ("Imported text", "bad link" not a link, "Click me"); no broken picture. "Parking" as a heading, "car park B" bold, the two bullets, "the map" and "call us" as links. Your "Fasting" block: heading, bold, italics, both lists, the two links, the lines from T-06/T-07 as plain text. **Both mountain pictures fit inside the screen**, and the page doesn't scroll sideways. No pop-up at any point.

**T-14: Links work for the patient.**
1. Tap **the map**. Then tap **Our fasting page**.
2. Long-press or hover **call us** and **the team** to see their addresses.

Expected: each https link opens in a **new tab** (the form stays open in the first tab). "call us" is `tel:+10000000000` and "the team" is `mailto:poa@example.org`.

**T-15: Nothing runs.**
1. Tap "Click me" (i_bad) and "bad link". In DevTools Console run `window.__xss`.

Expected: nothing happens on either tap; `window.__xss` is `undefined`.

**T-16: The patient finishes and sends.**
1. Answer "Will someone drive you home?", **Finish this section**, **Send my answers**, confirm.

Expected: "Thank you, …". Information blocks never ask for an answer.

**T-17: Reload and back.**
1. Before sending (use a second episode if you already sent), reload the form at phone size; then go back to the sections list and open the set again.

Expected: the blocks show formatted the same way after reload and after going back.

### The clinician

**T-18: Validate shows the blocks, cleaned.**
1. As Cara, open the episode → **Validate** for "Before you come in".

Expected: the same blocks, formatted, with i_bad cleaned and no pop-up. The POA Summary doesn't list the Information blocks.

### After test report 1

**T-20: Heading sizes match.** In the editor, select the "Parking" block: note the size of "Parking" in the Settings Preview. Open **Preview**, then the patient's form at 390 × 844, then Validate.

Expected: "Parking" (`##`) is the same size everywhere (about 18px), and "Fasting" (`#`) about 20px; neither is bigger than the set's title ("Before you come in").

**T-21: Blank lines are gaps.** In a new Information block type `Line one`, an empty line, then `Line two`; leave the box.

Expected: the Preview and the card both show a gap between the two lines, as an old statement with an empty line does. In the patient's form (after publishing a new version) the gap is there too.

**T-22: Hand-made HTML opens as separate lines.** Select i_bad (on a draft) and look at its Text box; then Translate (German).

Expected: the Text box shows "Imported text", "bad link" and "Click me" on three lines (not run together); Translate's English column shows the same three lines. Long text in Translate's English column wraps inside its column.

### After verification 1

**T-23: An answer shown back is plain text (security).**
1. Make a new S35 test HQ with the (updated) setup script, publish it, and open it as a patient (390 × 844) as in T-13.
2. In "Anything else we should know?" type `[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)` and press Tab. Also try `# Big` on one line and `two` on a second line.
3. Finish and send; then as Cara open **Validate**.

Expected: the block shows "You told us:" followed by exactly what was typed, in bold, as plain text on one line: no link "Sign in again", no picture, no heading. The same in Validate. In DevTools, no `<a>` or `<img>` pointing at evil.example anywhere on the page.

**T-24: Numbered steps keep their numbers.**
1. In a draft's Information block type: `1. Stop eating at midnight`, an empty line, `2. Take your usual tablets`, an empty line, `3. Arrive at 7am`. Then add a block with `3. Three` and `4. Four`.

Expected: the preview, the card, the Preview and the patient's form show 1, 2 and 3 (with gaps between), and 3, 4 in the second block; never "1. 1. 1.".

**T-25: Named characters in an old statement.**
1. On a draft, add by API (as the setup script does) or ask the conductor for a block with `html` `<p>caf&eacute; &pound;5 it&rsquo;s</p>`; or skip if you can't, and say so.

Expected: the card, the Preview and the patient show "café £5 it’s", and the Text box shows the same characters.

**T-26: An old statement's translation keeps its lines.**
1. In Translate (German), give i_old a two-line German text (e.g. `Bitte früh kommen.` / `Bademantel mitbringen.`). Close Translate.
2. Edit i_old's English text (add a word) and leave the box. Open Translate (German) again.

Expected: the German is still on two lines (not run together), and the Preview in German shows two lines.

### Refused and other trusts

**T-19: Another trust and another hospital.**
1. As **Bea Author** (Trust B), go to Questionnaires.
2. As **Val Clinician** (Hospital A2), try opening the S35 test episode's address from T-13 (copy it from Cara's browser).

Expected: Bea doesn't see the S35 test HQ. Val gets "not found" for the Hospital A1 episode. (Nothing new here: the routes didn't change.)

## 3. Regression

**R-01: Clinical Summaries (S32).** In any draft, add a **Clinical Summary** inside a Group: its card says "Covers the Group …". As a clinician validating an episode with summaries, the summary box lists its notes (e.g. a demo HQ answer with a disclosure) as a bulleted list.

**R-02: Video (S34).** Add a **Video** in a draft, choose one from the library: its card shows "YouTube · Length …"; the Preview shows the player.

**R-03: The Profile block.** In a draft, add **Profile**; in Preview it shows "Name", "Age" and "Sex" with the sample patient's details, in bold labels, and its note below.

**R-04: An existing demo HQ.** As Cara, create an episode on a demo HQ (e.g. the trust's default), open it as the patient at phone size: the questions show and can be answered as before.

**R-05: Translate view for other texts.** In Translate (German), a question's title can still be translated and shows in the Preview in German.

## 4. Out of scope (don't report)

- Uploading pictures or leaflets (plan C8); a picture must be an https link for now.
- The patient's Information on their home page, and the estimated time (S36).
- Patients choosing a language (no screen for it yet).
- Importing an incumbent HQ (`import-hq` is a command; Rahul checks it by hand).
- The usual console noise: two 401s from `/api/me` on the login page; 409s in conflict tests; a 503 shown for a 204 in the extension's network log.
