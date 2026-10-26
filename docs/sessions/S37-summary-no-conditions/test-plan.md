# S37 manual test plan: no conditions on summary boxes

For a tester who has never seen the code. Use only the browser. Report each case as Pass, Fail or Blocked, with what you saw.

**What's new (Rahul, 26 Oct):** a **Clinical Summary** box can't have a condition, because a hidden box would hide its notes.
- The editor offers **no Logic** on a Clinical Summary: its settings have no **Logic** tab, and its card never says "Displays when…".
- **Publish refuses** a draft whose summary has a condition. The Publish window names the Question Set and the summary: "<Question Set> · The clinical summary “<title>” has a condition: remove it (a summary is always shown)".
- A summary can only get a condition from outside the editor (the API, or JSON made by hand). Such a summary's card shows its "Displays when" box, its Settings show an orange warning, and its **Logic** tab is offered **only so the condition can be removed** (Display: Always, or the Code tab). Once removed, the Logic tab goes away and Publish works. Deleting the summary works too.
- Every other element (questions, Groups, Sections, Information, Video, pages) still offers Logic as before.
- Published versions are never rewritten.

## 1. Setup

1. In your own terminal (an automatic run does this for you):
   ```sh
   docker compose up -d db && make migrate seed && make dev
   ```
   Leave `make dev` running.
2. Open **http://localhost:3000** in Chrome. Sign in from the stub login page by clicking a user's name (sign out from the avatar menu to switch):
   - **Alex Author** (super clinician, the whole of Trust A): makes and publishes questionnaires.
   - **Sue Super** (super clinician, Hospital A1 only): views the trust's questionnaires but can't change them.
3. **Giving a summary a condition** (there is no way in the editor, on purpose). With the questionnaire open in the editor (its URL is `…/o/<trust>/h/<hospital>/questionnaires/<id>`), open Chrome's DevTools → **Console**, paste the snippet below and press Enter. It finds the first Clinical Summary in the questionnaire, gives it the condition `{patientAge} >= 18` through the API, and shows an alert: "Done: a Clinical Summary in “<Question Set>” now has a condition. Reload the page." Press OK, then **reload the page**.
   ```js
   (async () => {
     const m = location.pathname.match(/^\/o\/([^/]+)\/h\/([^/]+)\/questionnaires\/([^/]+)/);
     if (!m) return alert("Open the questionnaire in the editor first.");
     const api = `/api/o/${m[1]}/h/${m[2]}`;
     const { chapters } = await (await fetch(`${api}/questionnaires/${m[3]}`)).json();
     const find = (els) => { for (const el of els ?? []) { if (el.type === "clinicalsummary") return el; const f = find(el.elements ?? el.templateElements); if (f) return f; } };
     for (const c of chapters) {
       const ch = await (await fetch(`${api}/chapters/${c.id}`)).json();
       const s = (ch.content.pages ?? []).map((p) => find(p.elements)).find(Boolean);
       if (!s) continue;
       s.visibleIf = "{patientAge} >= 18";
       const res = await fetch(`${api}/chapters/${c.id}/content`, { method: "PUT", headers: { "Content-Type": "application/json" }, body: JSON.stringify({ content: ch.content, revision: ch.revision }) });
       alert(res.ok ? `Done: a Clinical Summary in “${c.name}” now has a condition. Reload the page.` : `Refused: ${res.status} ${await res.text()}`);
       return;
     }
     alert("No Clinical Summary found in this questionnaire.");
   })();
   ```
4. **Preparation:** as **Alex Author** (Hospital A1) → **Questionnaires** → make a new questionnaire **"S37 Summary HQ"** with one Question Set **About you** (Who is it for? **Patient**). On its first page (title it `Heart`) add:
   - a **Yes / No** question "Do you get chest pain?";
   - a **Group** "Breathing" holding a **Yes / No** question "Do you have asthma?".
   Don't publish yet.

## 2. Test cases

### The editor offers no Logic on a summary

**T-01 A new summary has no Logic tab**
1. In "S37 Summary HQ", on the Heart page, press the page's **Clinical Summary** button (the Add bar at the bottom).
2. Look at the settings column on the left, with the new summary selected.
3. Look at the summary's card.

Expected: the settings column has **Settings** only: no **Logic** tab (and no Disclosures). It shows the Title and the "Only clinicians see this box…" help, with no orange warning. The card says "Covers the page “Heart”" and has **no "Displays when"** box.

**T-02 A summary inside a Group**
1. Click the **Breathing** Group's title to select it, then press the Group's own **Clinical Summary** button.

Expected: as T-01: the new "Breathing summary" has no Logic tab, and its card no "Displays when" box.

**T-03 Other elements still offer Logic**
1. Click the "Do you get chest pain?" card.
2. Click the **Breathing** Group's title.
3. Click the summary again, then the question again.

Expected: the question and the Group each show **Settings | Disclosures | Logic** (the Group: Settings | Logic) as before, and their Logic tab opens the Builder with "Display: Always / Conditionally". Moving between the summary and the question shows and hides the Logic tab without errors; the summary always opens on Settings.

**T-04 Give a question a condition, as before**
1. Select "Do you have asthma?" → **Logic** → Display: **Conditionally** → make it show when "Do you get chest pain?" is **Yes**.

Expected: the question's card shows "Displays when … Yes", as before. This doesn't affect the summaries (they keep no Logic, and Publish isn't blocked by this).

**T-05 A draft with no condition on any summary publishes**
1. Delete the **Breathing summary** (its card's Delete, then Delete in the confirmation), so one page summary remains.
2. Wait for "All changes saved". Press **Publish** at the top.

Expected: the window says "Patients can then be given this version…", with no problems listed. **Don't publish yet**: press **Cancel** (you'll use this draft next).

### Publish refuses a summary with a condition

**T-06 Publish names the summary**
1. With "S37 Summary HQ" open in the editor, run the Console snippet (Setup 3); OK the alert; reload the page.
2. Open About you → Heart. Look at the summary's card.
3. Press **Publish**.

Expected: the summary's card now shows a "Displays when" box (Patient age is at least 18, or similar wording). The Publish window lists **1 problem to fix first**: "**About you** · The clinical summary “Heart summary” has a condition: remove it (a summary is always shown)", and its **Publish** button is disabled.

**T-07 The summary's Settings say why**
1. Cancel the window. Click the summary's card (not its "Displays when" box).

Expected: its Settings show an orange warning: "A Clinical Summary can't have a condition, so this version can't be published. Remove the condition in the Logic tab (Display: Always, or the Code tab), or delete the summary." A **Logic** tab is now offered (only because it has a condition).

**T-08 The name follows a typed title**
1. In the summary's Settings, change its **Title** to `Heart checks` and press Tab. Wait for "All changes saved".
2. Press **Publish**.

Expected: the problem now reads "… The clinical summary “Heart checks” has a condition: …". Cancel.

**T-09 Remove the condition in the Code tab**
1. Select the summary → **Logic** → **Code**. The "Visible if" box holds `{patientAge} >= 18`.
2. Empty the box and press **Save logic**. Wait for "All changes saved".

Expected: the Logic tab disappears and the column shows **Settings** again; the orange warning is gone; the card's "Displays when" box is gone. **Publish** now shows no problems.

**T-10 Remove it with Display: Always instead**
1. Run the snippet again; OK; reload. Open About you → Heart.
2. Click the summary's "Displays when" box on its card.
3. In the Logic tab's **Builder**, set **Display** to **Always**.

Expected: clicking the "Displays when" box opens the summary's Logic tab. After Always: as T-09 (no Logic tab, no warning, no box, Publish shows no problems).

**T-11 Or delete the summary**
1. Run the snippet again; OK; reload. Open About you → Heart.
2. Press **Publish**: the problem is listed. Cancel.
3. Delete the summary (card's Delete → Delete).
4. Press **Publish**.

Expected: no problems listed. Now press **Publish** in the window: version 1 is published, and the editor shows it read-only.

**T-12 Reload and the back button**
1. Make a new version of "S37 Summary HQ" (**New version**), add a page **Clinical Summary** again, run the snippet, OK, reload.
2. Press **Publish**: the problem is listed. Cancel.
3. Go back with the browser's back button to the questionnaire list, then forward again (or open the questionnaire again from the list).

Expected: the problem and the warning are still there after the reload and after coming back (they come from the saved draft, not from the page's memory). Then remove the condition (T-09) and publish version 2.

### Who sees what

**T-13 A read-only viewer**
1. Repeat step 1 of T-12 on a new version 3 (a summary with a condition, not published). Sign out; sign in as **Sue Super** (Hospital A1).
2. Open "S37 Summary HQ" → About you → Heart, and click the summary's card.

Expected: Sue sees the editor read-only: the card's "Displays when" box, and a Logic tab whose fields are disabled; **no** orange warning (she can't change it). She can't Publish (no Publish button, or it's refused as before for her role).

## 3. Regression

**R-01 Two summaries in one Group are still refused (S32)**
As Alex, on any draft: add two Clinical Summaries in the same Group → Publish lists "Two clinical summaries cover the same Group “…”: keep one".

**R-02 Page Logic still works**
In the structure column on the left, use a page's menu → **Logic**: the page's Display and skip rules open as before.

**R-03 The logic problems button still works**
Give a question a condition testing a question **below** it (Logic → Display: Conditionally → pick a later question). The editor's orange "conditions won't work" button appears; clicking the problem opens that question's Logic tab. Then remove the condition.

**R-04 Video and Information still offer Logic**
Add a **Video** and an **Information** block: each has a Logic tab with Display Always / Conditionally.

## 4. Out of scope (don't report)

- Other notes about summaries in pending.md: a summary reaching a repeating group indirectly, the importer making two summaries in one place, "Heart Summary" vs "Heart summary" capitalisation.
- The API itself still saves a summary with a condition, and **the API's own publish (without the editor) doesn't check summaries**: the editor's Publish window is the check, as for logic problems, test cases and two summaries.
- Publish problems are plain text (none of them link to their item yet).
- Published versions that already have a summary with a condition are left as they are.
