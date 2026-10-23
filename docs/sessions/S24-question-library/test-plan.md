# S24 test plan: the question library and "Same as…"

**Retest after verification 2 (23 Oct; the last round, report 4).** A **Question Set condition** (a later set shown only for some answer) that tests an answer the question no longer has is now a problem: the header's logic problems (when that set is open) and the **Publish** window refuse it ("Question Set condition: it tests an answer that no longer exists (…)"). Same as… counts such conditions in **other** Question Sets in its warning, and renames test cases' answers too. New **T-16 step 8**. Retest **T-07 to T-10, T-15, T-16** and the regression cases; the rest if time allows.

**Retest after verification 1 (23 Oct).** Changes to "Same as…":
- **Always ask fresh is copied:** when the chosen question has it on (e.g. `pregnant`, `weight`), its row says "Always ask fresh, like the original" and the adopted question gets the orange **Always ask fresh** badge; the author may switch it off (Rahul, 23 Oct). T-07 below uses `smoker` (not fresh); new case **T-15**.
- **Conditions follow the options:** a condition that tested an old option now tests the new option **with the same text**; one that can't be mapped is a **logic problem** ("it tests an answer that no longer exists …"), which blocks Publish. Before **Use this**, the window warns what will be lost (disclosures, conditions) with **Use this anyway** / Cancel. New case **T-16**.
- Other Question Sets' conditions on the question are renamed too; a Yes / No question taking three options becomes Select One.
Retest T-07 to T-10, T-15 and T-16 (and the regression cases).

**Corrected after test report 2 (22 Oct):** T-05 step 9's expected message (either refusal sentence is right).

**Retest after test report 1 (22 Oct).** Report 1 passed 19 of 19; two of its "found while exploring" items are fixed:
- **Save to library isn't offered on a question inside a repeating group** (it is on the repeating group itself), and the server refuses saving a repeating list's row question on its own (T-05 step 9).
- **The library and Same as… windows open with an empty search** each time (T-02 step 9).
Run all cases again with a new run number.

**For the tester:** you test only through the browser (and, where a case says so, the browser's JavaScript console). You haven't seen the code. Report each case as Pass, Fail or Blocked, with what you saw.

**What this session built:**
- **The question library:** questions that keep their **ID** (their "key", shown as "ID …" at the top of a question's Settings panel) wherever they're inserted. Two questions with the same ID are the same question, so a later session can fill in a patient's earlier answer. There are two parts:
  - the **platform's starter library** ("Preopsis" badge): 11 made-up questions with readable IDs (`smoker`, `alcohol_units`, `snoring`, `ponv_history`, `anaesthetic_problems…`, `family_anaesthetic_problems…`, `height`, `weight`, `pregnant`, `allergies`, `regular_medicines`). It can't be changed or removed in the app;
  - **the trust's own library** ("This trust" badge): questions saved from the editor with **Save to library**, and removed from the library window. Only **super clinicians and admins of the whole trust** use the library.
- **From library** (in the editor's "Add content" bar): search by question text or tag, then **Insert**. The inserted question **keeps its ID and option values**. Inserting a question whose ID is already in the questionnaire (in any Question Set) is refused.
- **Same as…** (a button at the top of a patient question's Settings panel): lists questions the trust has **published** and the library's, by text. **Use this** gives the selected question the chosen one's **ID and options** (values and wording); it keeps its own question text, and shows "Linked to “…” in …". Only questions with the **same kind of answer** can be used (single choice with single choice, multiple with multiple, a number with the same unit, a date with the same format, text with text); the others are greyed out with the reason. A question whose ID is already in this questionnaire can't be used either.
- Because of the previous session's lock, once a question is published, its answers (options, unit, format…) can't change in any later publish in the trust. **A medication question's type (Prescribed / Non-prescribed / Recreational) is now part of that too.**
- **The seeded Demo pre-op HQ** now uses library questions.

## 1. Setup

Rahul (or the conductor) runs, in his own terminal, and leaves `make dev` running:

```sh
docker compose up -d db && make migrate seed && make dev
```

**The local database was reset by the builder on 22 Oct** (migrated and seeded; HJE Full HQ kept). The seeded **Demo pre-op HQ** (v1, published, in Trust A and in Trust B) is the new one, made from library questions. If it still shows "Do you smoke?" with only **Yes / No** options, the database wasn't reset: report it as Blocked.

Open **http://localhost:3000**. Sign in on the stub login page by clicking a name:

| User | Role | Use for |
| --- | --- | --- |
| **Alex Author** | super clinician, the whole of Trust A | authoring, the library, Same as… |
| **Ada Admin** | admin, the whole of Trust A | an admin can use the library too |
| **Sue Super** | super clinician, Hospital A1 only | sees the trust's questionnaires read-only, no library |
| **Cara Clinician** | clinician, Hospital A1 only | no library |
| **Bea Author** | super clinician, the whole of Trust B | another trust |

Sign out from the avatar menu (top right of the editor) or the app's menu, then click another name.

**Making a questionnaire:** Questionnaires → **New Questionnaire** → Name and Description → Save. In the editor: **Add Question Set**, then **Add Page** on it, then add questions from the **Add content** buttons under the page (**Text**, **Select One**, **Number**, …, and the new **From library** at the end). Click a question's card to open its **Settings** panel on the left; **Question text** saves when the box loses focus. Option texts are edited in the Settings panel. The save status ("All changes saved") is in the header. Use a **run number** `<n>` of your choice in names (e.g. `S24 Lib <n>`).

**Publishing:** **Publish** (top right) → the window → **Publish**. A published questionnaire is read-only; **Create new version** makes an editable draft copy.

**Console snippets:** open DevTools on any page of the app while signed in. `trust` is Trust A's id, from the address bar (`/o/<trust>/h/<hospital>/…`):

```js
const trust = location.pathname.split("/")[2];
```

## 2. Test cases

### T-01: Insert from library keeps the ID; a second insert is refused
1. As **Alex**, make a questionnaire `S24 Lib <n>` with a Question Set "About you" and a page.
2. Click **From library** in the Add content bar. A window "Insert from the question library" opens, listing the platform's questions with a **Preopsis** badge, their kind, ID and tags.
3. Type `smoke` in Search. Only **Do you smoke?** (ID `smoker`, tags Lifestyle, Smoking) is left. Click **Insert**.
4. **Expected:** the window closes; the card "Do you smoke?" shows the options **No, I have never smoked / I used to smoke / Yes, I smoke now**; its Settings panel says **ID smoker**. "All changes saved".
5. Add a second Question Set and page, open **From library** there, and Insert **Do you smoke?** again.
6. **Expected:** refused inside the window, under that row: "This HQ already has a question with the ID smoker. A question can be in an HQ only once…". Nothing is added (close the window with ×).
7. Reload the page and re-open the first page: the question is still there with **ID smoker**.

### T-02: Searching the library; a group and a repeating group
1. In `S24 Lib <n>`, open **From library**. Search `ANAESTHETIC` (capitals): **Anaesthetics in your family**, **Have you ever been sick, or felt very sick, after an anaesthetic?** and **Problems with anaesthetics** are listed (any case matches).
2. Search `measurements` (a tag): **How tall are you?** and **What do you weigh now?**.
3. Search `zzz`: "No library questions match."
4. Insert **Problems with anaesthetics** ("Group of 2 questions").
5. **Expected:** a Group card holding "Have you ever had a problem with an anaesthetic?" (Yes / No, ID `anaesthetic_problems`) and "What happened with your anaesthetic?" (ID `anaesthetic_problems_details`) with **Displays when … is Yes**.
6. Insert **Your allergies** ("Group (repeatable)"): a repeating group card with "What are you allergic to?" and "What happens when you have it?".
7. Insert **What do you weigh now?**: its card shows the orange **Always ask fresh** badge; the unit is kg (Settings).
9. Close the window with ×, reopen **From library**: the Search box is empty. Do the same with a question's **Same as…** window: empty each time it opens.
8. Click **Preview** (top right) for this Question Set: the questions show; choosing **Yes** for "Have you ever had a problem with an anaesthetic?" shows "What happened with your anaesthetic?".

### T-03: Where From library isn't offered
1. In `S24 Lib <n>`, click the repeating group "Your allergies" (T-02): its own Add content bar (inside the group) has **no From library** button. The page's bar has one.
2. A page holding a **BMI** block: add a new page, click **BMI** first; **From library** is greyed out on that page.
3. Open the seeded **Demo pre-op HQ** (published, read-only): no Add content bar, so no From library.
4. Sign in as **Sue Super**, open `S24 Lib <n>`: read-only, no From library, and a question's Settings panel has **no Same as…** and **no Save to library**.
5. As **Cara Clinician**, open Questionnaires (if she can) or the same editor address: no From library, Same as… or Save to library.
6. As Cara, in the console: `await fetch(\`/api/o/${trust}/library-questions\`).then(r => r.status)` → **403**. As Sue: also **403**.

### T-04: Save to library (the trust's own), never seen by another trust
1. As **Alex**, in `S24 Lib <n>`, add a **Text** question with text `S24 job question <n>`.
2. In its Settings panel click **Save to library**. A window shows the question and its ID. Add the tag `Work` (type it, press Enter), then **Save to library**.
3. **Expected:** "Saved. The trust's authors can now insert it…". Click **Done**.
4. Open **From library**, search `job question`: the row has a teal **This trust** badge, the tag **Work**, and a **Remove** button. Platform rows have no Remove.
5. Sign in as **Ada Admin** (admin of Trust A), open a draft questionnaire, **From library**, search `job question`: it's listed.
6. Sign in as **Bea Author** (Trust B), make or open a draft in Trust B, **From library**, search `job question`: **not listed**. The platform's questions are listed.
7. As Bea, in the console on a Trust B page: `await fetch(\`/api/o/${trust}/library-questions?q=job\`).then(r => r.json())` → no `S24 job question <n>`.

### T-05: Save to library refusals
1. As **Alex**, in `S24 Lib <n>`, select the **Do you smoke?** question (ID `smoker`, from T-01) and click **Save to library** → **Save to library**.
2. **Expected:** refused in the window: "The library already has a question with the ID smoker."
3. Save `S24 job question <n>` (T-04) again: refused the same way ("…the ID q_…").
4. Open the seeded **Demo pre-op HQ** → **Create new version**. In "Your health" → Health, select **Do you have diabetes?** and reword its option **No** to `Not at all` (saves on leaving the box).
5. Click **Save to library** → **Save to library**.
6. **Expected:** refused: "This question isn't the same as the trust's published one, so it can't go in the library. “Do you have diabetes?”: the option “No” was reworded to “Not at all”…".
7. Undo the rewording (Ctrl/⌘+Z, or type `No` back) so this draft can still be published (R-05).
8. Statements, Calculations, Signature, File upload and the BMI block have **no Save to library** button; nor does a Group or Section. A repeating group (Your allergies) has one.
9. Select **What are you allergic to?** inside the "Your allergies" repeating group (T-02): **no Save to library** button. As Alex, in the console, try saving it anyway: `await fetch(\`/api/o/${trust}/library-questions\`, {method: "POST", headers: {"Content-Type": "application/json"}, body: JSON.stringify({element: {type: "text", name: "allergies_what", title: "What are you allergic to?"}})}).then(r => r.json())` → refused (409). The message is "“What are you allergic to?” is a question of a repeating list in “Demo pre-op HQ”, so it can't go in the library on its own. Save the repeating group instead." when a published HQ of the trust has that repeating list, else "The library already has a question with the ID allergies_what, in “Your allergies”." *(Corrected after test report 2: both refusals are right; the first is checked first.)*

### T-06: Remove a trust library question; inserted copies stay
1. As **Alex**, make a second questionnaire `S24 Lib B <n>` with a Question Set and page; **From library** → search `job question` → **Insert**. The card shows `S24 job question <n>` with the same ID as in T-04.
2. Open **From library** again, search `job question`, click **Remove** on its row.
3. **Expected:** a red box "Remove from the library?" saying questionnaires keep their copy. Click **Cancel**: nothing changes. Click **Remove** then the box's **Remove**: the row disappears ("No library questions match.").
4. Close the window. **Expected:** the question in `S24 Lib B <n>` (and in `S24 Lib <n>`) is unchanged, with the same ID. Reload: still there.

### T-07: Same as… a published question in another HQ (the main path)
1. As **Alex**, in `S24 Lib B <n>`, add a **Select One** question with the text `Are you a smoker at the moment?`. Note its ID (q_…) and its options (Option 1, 2, 3).
2. Its card shows **New question: won't pre-fill**.
3. In its Settings panel click **Same as…**. A window explains it, with a Search box and a list.
4. Search `smoke`. **Do you smoke?** (ID `smoker`, "In Demo pre-op HQ, the library", options listed) has an enabled **Use this**. Click it.
5. **Expected:**
   - the window closes; the Settings panel now says **ID smoker** and "Linked to “Do you smoke?” in Demo pre-op HQ";
   - the question text is still **Are you a smoker at the moment?**;
   - the options are now **No, I have never smoked / I used to smoke / Yes, I smoke now**;
   - after "All changes saved", the card's label says **Pre-fills from 1 other HQ** (more if other published HQs of Trust A have `smoker`); clicking it lists **Demo pre-op HQ · by link**.
6. Reload the page: the ID, options, text and "Linked to…" are kept.
7. Try to change the options (e.g. add one, or reword one) — allowed while editing; it's refused at Publish (T-10). Undo it (Ctrl/⌘+Z) before going on.

### T-08: Incompatible candidates are greyed with their reason
In `S24 Lib B <n>`, as **Alex**:
1. Add a **Text** question `How much do you smoke?`. **Same as…** → search `smoke`: **Do you smoke?** is greyed, **Use this** is disabled, and it says "Its answer is single choice; this question's is text." (It may also say "This HQ already has a question with this ID…" if `smoker` is already in this HQ from T-07: that's also a refusal.)
2. Add a **Number** question `Your weight`; in Settings set its unit to `lb`. **Same as…** → search `weigh`: **What do you weigh now?** is greyed: "Its unit is “kg”; this question's is “lb”." Change the unit to `kg`, close and re-open **Same as…**, search `weigh`: now **Use this** is enabled (don't use it).
3. With the Number question selected, **Same as…** → search `allerg`: **Your allergies** is greyed: "Repeating lists can't be linked with “Same as…”: insert them from the library instead."
4. A **Select Many** question: **Same as…** → search `smoke`: greyed, "Its answer is single choice; this question's is multiple choice."

### T-09: Same as… refused when the ID is already in this HQ
1. In `S24 Lib <n>` (which has `smoker` from T-01), add a **Select One** question `Do you currently smoke?`.
2. **Same as…** → search `smoke`: **Do you smoke?** is greyed: "This HQ already has a question with this ID, in “About you”." (the Question Set where `smoker` is). **Use this** is disabled.
3. Where is Same as… not offered: a question in a Question Set whose audience is **Clinician**; a question switched to **Clinical**; a question inside a repeating group; a repeating group itself; a Statement or Calculation. Check two or three of these.

### T-10: Deleting an adopted option is refused at Publish (S23's lock)
1. In `S24 Lib B <n>` (T-07's `smoker`), select the question and **delete** the option **I used to smoke** (its bin icon).
2. Make sure nothing else in this questionnaire has a problem (delete the questions T-08 left, if any, or keep them: they don't block).
3. Click **Publish**.
4. **Expected:** the Publish window lists "“Are you a smoker at the moment?”: the option “I used to smoke” was removed. A published question's answers can't change: add a new question instead (published in “Demo pre-op HQ” v1…)." and **Publish** is disabled.
5. Cancel. Undo the deletion (Ctrl/⌘+Z) until the three options are back, then **Publish** again: it publishes (if T-08's questions are complete). The card keeps **ID smoker**.

### T-11: A medication question's type is frozen once published
1. As **Alex**, make `S24 Meds <n>` with one Question Set and page; add a **Medication** question (type **Prescribed**, the default, in Settings → Medication type). **Publish** it.
2. **Create new version**. Select the medication question and set **Medication type** to **Recreational**.
3. **Publish**. **Expected:** refused: "“Please add the prescribed medications you are taking”: its medication type changed from prescribed medication to recreational drugs. A published question's answers can't change…". Publish disabled.
4. Set it back to **Prescribed**: Publish is allowed (publish it).

### T-12: The seeded Demo pre-op HQ uses library questions
1. As **Alex**, open **Demo pre-op HQ** (v1 published). In the Structure tree: "About you" (pages **Smoking and alcohol**, **Height and weight**), "Your health" (**Health**, **Anaesthetics**, **Allergies and medicines**), "Nurse checks".
2. Click questions and check their IDs in Settings: `smoker`, `alcohol_units` (unit "units a week"), `height` (cm), `weight` (kg, **Always ask fresh** badge), `q_diabetes`, `snoring`, `pregnant` (**Always ask fresh**), `ponv_history`, the two anaesthetic groups, `allergies`, `regular_medicines`.
3. Trust B's **Demo pre-op HQ** (as Bea) has the same questions.

### T-13: The new Demo HQ on a patient's phone (390×844)
1. As **Cara Clinician**, **New episode**: a new made-up patient (e.g. first name `Sam`, surname `S24 <n>`, date of birth 17/05/1980, a made-up mobile `07700 900<3 digits>`), procedure **Knee arthroscopy** (its default HQ is **Demo pre-op HQ v1**), **Create episode**.
2. Open **http://localhost:3000/qa/messages**, find the text to that mobile and open its link in a window sized **390×844** (DevTools device toolbar). **Continue**, enter the date of birth.
3. **Expected:** "About you" and "Your health" (2 sections). In About you: "Do you smoke?" with three options; "About how many units of alcohol…" (a number, with its hint text); "How tall are you?" and "What do you weigh now?" (numbers). In Your health: "Have you ever had a problem with an anaesthetic?" — choosing **Yes** shows "What happened with your anaesthetic?"; "Your allergies" and "Medicines you take regularly" start empty with **Add an allergy** / **Add a medicine** buttons, and adding one shows the boxes. Nothing is wider than the screen.
4. Fill in both sections and **Send my answers**: "Thank you, Sam". (No pre-fill yet: that's the next session.)

### T-14: Reload, deep link and Back
1. In `S24 Lib B <n>`, select the `smoker` question; copy the address (it has `?set=…&page=…`). Open it in a new tab: the same page opens; the card shows **ID smoker** when clicked, and "Linked to…" is in Settings.
2. Open **From library** and press the browser's **Back**: you leave the editor for the previous page (the window isn't a page of its own); **Forward** comes back to the editor without the window.

### T-15: Same as… copies Always ask fresh (verification 1)
1. As **Alex**, in a draft, add a **Yes / No** question `Might you be pregnant?`. **Same as…** → search `pregnant`.
2. **Expected:** the **Is there any chance you could be pregnant?** row has an orange "Always ask fresh, like the original" badge. **Use this**.
3. **Expected:** ID `pregnant`, options Yes / No / I'm not sure, the card shows the orange **Always ask fresh** badge, and Settings shows **Select One** (not Yes / No, since it has three options) with **Always ask fresh** switched on.
4. Switch **Always ask fresh** off: the card now says **Pre-fills from N other HQs** (Demo pre-op HQ, by link). Switch it on again: allowed either way.

### T-16: Same as… and the question's conditions and disclosures (verification 1)
1. As **Alex**, in a draft, add a **Select One** question `Do you smoke at all?` and reword its options to `Yes` and `Never` and `Sometimes` (Option 1, 2, 3).
2. Add a **Text** question `How many a day?` after it; in its **Logic** tab make it display when `Do you smoke at all?` **is** `Yes`. Add another **Text** question `Why did you stop?` displaying when it **is** `Sometimes`.
3. On `Do you smoke at all?`, open **Disclosures** and add one to the option `Never` (any note).
4. **Same as…** → search `snore` → **Do you snore loudly?** (Yes / No) → **Use this**.
5. **Expected:** instead of adopting at once, the row warns: "Its own disclosures on 1 option will be removed. 1 condition tests its old answers and will need changing (Publish is refused until then)." **Cancel** leaves the question as it was.
6. **Use this** again → **Use this anyway**. **Expected:** ID `snoring`; "How many a day?" still displays when it **is** Yes (mapped by its text); "Why did you stop?" is now a **logic problem** in the header ("…it tests an answer that no longer exists (“Do you smoke at all?”: o_…)"), and **Publish** is refused until it's changed or deleted.
7. If the questionnaire has a second Question Set whose **Question Set condition** tested `Do you smoke at all?`, it now names the new question (optional).
8. **A later Question Set shown only for an answer with no match (verification 2):** make a new draft with Question Set **"Smoking"** (a page, a **Select One** question `Do you smoke at all?` with options `Yes` and `Sometimes`) and a second Question Set **"Occasional smokers"** (a page and any question). Select "Occasional smokers" in the Structure tree and give it the **Question Set condition** "Do you smoke at all? **is** Sometimes". Go back to the question in "Smoking" → **Same as…** → search `snore` → **Do you snore loudly?** → **Use this**.
   - **Expected:** the warning says "1 condition tests its old answers and will need changing (Publish is refused until then)." (Sometimes has no match among Yes / No.) **Use this anyway**: ID `snoring`.
   - Open **Publish**: refused, listing "Occasional smokers · Question Set condition: it tests an answer that no longer exists (“Do you smoke at all?”: o_…)". Publish is disabled.
   - Open "Occasional smokers" in the Structure tree: the header's logic-problem button lists the same problem. Change its condition to "is Yes" (or remove it): the problem goes, and Publish is allowed.
   - With "is **Yes**" instead before adopting (Yes has a match), there's no warning, and the condition follows to `snoring` = Yes with no problem.

## 3. Regression

- **R-01 (S23):** in a draft that shares a key with a published HQ, the card says **Pre-fills from N other HQs** and its list says "by link"; a new question says **New question: won't pre-fill**; **Always ask fresh** can be switched on and its badge shows.
- **R-02 (S23):** in a new version of a published HQ, rewording a published option is refused at Publish, naming the question and the option; put back, it publishes.
- **R-03 (S19):** New episode with the procedure **Total hip replacement** picks **Demo pre-op HQ v1** as its HQ.
- **R-04 (S14/J3):** a patient fills in and sends an HQ on a phone (T-13 covers this with the Demo HQ).
- **R-05 (S23):** the seeded **Demo pre-op HQ** → **Create new version** → change nothing → **Publish**: it publishes (v2). (If you did T-05's draft, undo its change and publish that one.)

## 4. Out of scope (don't report)

- **Pre-filling** a patient's answers (S25) and the clinician's markers (S26). The label only says where an answer *would* come from.
- **Saving a group of questions** to the library (later, PNL-06), and a superadmin screen for the platform's library (C5).
- A **Same as…** for repeating lists (by design: they come from the library).
- The old "Option 1/2/3" disclosures and conditions of a question are dropped when it takes another question's options (by design; the logic-problem button flags any condition left testing an old option).
- QA (AWS): its Demo pre-op HQ is the old one until it's reset (recorded in pending.md).
- Existing "E2E …" questionnaires made by automated tests.
