# S01 manual test plan: authoring regression

**What changed in S01:** almost nothing you can see. S01 added automated API → database tests and fixed two small server bugs. This plan checks that authoring still works end to end, plus the two fixes (T-12, T-13).

## 1. Setup

**Rahul runs these in his own terminal before testing, and leaves the last one running:**

```sh
docker compose up -d db
make migrate seed
make dev
```

- **The app:** http://localhost:3000. If it doesn't load, stop and report it. Don't try to start or fix the servers.
- **Sign-in:** the sign-in page lists the users; click a name. There's no password. **Sign out** is at the top right.

  | User | Hospital | Can do |
  | --- | --- | --- |
  | Alex Author | Hospital A | write and publish questionnaires |
  | Val Viewer | Hospital A | read only |
  | Bea Author | Hospital B | write questionnaires, in Hospital B only |

- **Data rules:**
  - Use made-up text only.
  - Name everything you create `S01 test <time>`, e.g. `S01 test 14:05`. Don't change questionnaires you didn't create (e.g. **HJE Full HQ**).
- **Autosave:** the editor saves by itself. The header shows "Saving…", then "All changes saved". Wait for "All changes saved" before reloading or leaving. Settings text boxes save when you click out of them.
- **Terms:** a **Question Set** is a section of a questionnaire. **Disclosures** are clinical outputs (codes, a note, a flag) on an answer.
- **Console:** keep DevTools open and report any red error with the case ID. Expected and not errors:
  - two `401 (Unauthorized)` on the sign-in page;
  - a `409 (Conflict)` in T-03 (duplicate name);
  - a `400 (Bad Request)` in T-13 (the fix being tested).
- **Terminal step:** T-14 asks you to run one command in a terminal and copy its last lines.

## 2. Test cases

### T-01: Sign in, and the questionnaire list
1. Open http://localhost:3000 and click **Alex Author**.
2. **Expected:** the questionnaire list for Hospital A opens. The header shows a **Hospital** picker on Hospital A, Alex's name and **Sign out**. The address is `/h/<a long id>/questionnaires`. Note that id: it's Hospital A's, and later cases use it.

### T-02: Create a questionnaire
1. Click **New Questionnaire**. A drawer opens.
2. Click **Save** with Name and Description empty. **Expected:** it's refused with a message, and nothing is created.
3. Enter Name `S01 test <time>` and any Description. Click **Save**.
4. **Expected:** the editor opens, with the name in the header and a "draft" badge.

### T-03: Duplicate draft name is refused
1. Click **← Questionnaires**. Click **New Questionnaire** and enter exactly the same name as in T-02, but in CAPITALS, with any description. Click **Save**.
2. **Expected:** the message "A draft questionnaire with this name already exists." No second questionnaire appears in the list.

### T-04: Search the list
1. On the list, type part of your questionnaire's name in **Search by name or description**.
2. **Expected:** the list narrows to matching rows, including yours.
3. Search for a word from your Description. **Expected:** your questionnaire is found.
4. Clear the search box. **Expected:** the full list is back.

### T-05: Add a Question Set, a page and a question, and keep them after reload
1. Open your questionnaire. Click **Add Question Set**. **Expected:** "Question Set 1" appears in the Structure tree on the left.
2. Rename it to `Breathing` with the pencil, and type a description, then click out of the box.
3. Click **Add Page**. Click the new page, and rename it to `Lungs` with its pencil.
4. Under "Add content:", click **Yes / No**. In the Settings panel, set the question text to `Do you smoke?` and click out of the box.
5. Under "Add content:", click **Select Many**. Set its text to `Which lung conditions do you have?` and give it three options: `Asthma`, `COPD`, `Sleep apnoea`.
6. Wait for "All changes saved". Note the address: it ends in `?set=…&page=…`.
7. Reload the page (⌘R / Ctrl+R).
8. **Expected:** you're back on the `Lungs` page of `Breathing`, with both questions and all three options exactly as you left them.

### T-06: Deep link and the back button
1. Copy the editor's full address from T-05.
2. Click **← Questionnaires**, then press the browser's **Back** button. **Expected:** the editor opens again with your questionnaire, and nothing is lost.
3. Open a new tab, paste the copied address and press Enter. **Expected:** the same Question Set and page open, with both questions.
4. Close the extra tab.

### T-07: Saved option lists
1. On the **Select Many** from T-05, click **Save as a list**. Name it `S01 list <time>` and click **Save list**. **Expected:** "Saved …" is shown, then click **Done**.
2. Add a **Select One** question. Click **Use a saved list**. **Expected:** your list is shown with its three options as chips, Alex's name and today's date.
3. Choose "Add to the current options" and click **Add**. **Expected:** Asthma, COPD and Sleep apnoea are added to the Select One.
4. Reload once "All changes saved" shows. **Expected:** the Select One still has the three options.
5. Open **Use a saved list** again and delete your list with its trash icon. **Expected:** it disappears from the list, and both questions keep their options.

### T-08: Code search in a disclosure
1. On the **Yes / No** question from T-05, open the **Disclosures** tab and click **Add** next to **Yes**, then **+ Add disclosure**.
2. Under Codes, search SNOMED for `smok`. **Expected:** "Smoker" (77176002) and "Ex-smoker" (8517006).
3. Clear the box and search for the start of a code number: `7717`. **Expected:** "Smoker" (77176002).
4. Search for `asthma`. **Expected:** codes with "asthma" anywhere in the description, including "Mild asthma", "Moderate asthma" and "Severe asthma".
5. Pick one code, type the Note `Current smoker`, choose Category "Lifestyle", click **Save output**, then **Done**.
6. **Expected:** the card shows a 🔗 1 chip next to Yes. Reload: the chip is still there.

### T-09: Publish, then "Create new version"
1. Back on the list, create a second questionnaire named `S01 publish <time>`. Published questionnaires can't be deleted, so keep it tiny.
2. Add a Question Set, a page, and one **Yes / No** question with text `Ready?`. Wait for "All changes saved".
3. Click **Publish**. **Expected:** a dialog opens. If it lists problems, Publish stays disabled; it should list none for this questionnaire. Click **Publish**.
4. **Expected:** the header shows "v1 · published" and "Read only". There are no Add content buttons, and the settings fields can't be changed.
5. Reload. **Expected:** still "v1 · published", read only.
6. Click **Create new version**. **Expected:** "v2 · draft", with the same Question Set, page and `Ready?` question, editable again.
7. Back on the list. **Expected:** the row for `S01 publish <time>` shows version 2, draft.

### T-10: Val Viewer can't edit
1. Click **Sign out** and sign in as **Val Viewer**.
2. **Expected:** the list shows Hospital A's questionnaires, including yours, but there's no **New Questionnaire** button. A row's ⋯ menu says **View**, not Edit, and has no Delete.
3. Open `S01 test <time>`. **Expected:** a "Read only" badge in the header. No Add content buttons, no Add Page or Add Question Set, and the settings fields are disabled.
4. Copy the address of this editor page, for T-11.

### T-11: Bea Author (Hospital B) can't open Hospital A's questionnaire
1. Sign out and sign in as **Bea Author**. **Expected:** Hospital B's list opens, and your `S01 …` questionnaires are **not** in it.
2. Paste the address copied in T-10 (Hospital A's editor) into the address bar and press Enter.
3. **Expected:** "You don't have access to this hospital." and none of the questionnaire's content.
4. Now take Bea's own list address (`/h/<Hospital B's id>/questionnaires`) and add `/` plus the questionnaire id from the T-10 address (the long id after `/questionnaires/`). Open it.
5. **Expected:** an error such as "Questionnaire not found." and none of the questionnaire's content. Hospital A's questionnaire is never shown under Hospital B.
6. Sign out and sign back in as **Alex Author**.

### T-12: A very large page number in the list doesn't break (bug fix)
1. As Alex Author, open this address, putting Hospital A's id from T-01 in place of `<A>`:
   `http://localhost:3000/api/h/<A>/questionnaires?page=99999999999`
2. **Expected:** the browser shows a small block of JSON text with `"items":[]` and a `"page"` number. It must **not** say "Something went wrong."
3. Change the end to `?page=abc`. **Expected:** JSON with `"page":1` and your questionnaires in `"items"`.

### T-13: A broken chapter update is refused, not dropped (bug fix)
Skip this case if you can't use the DevTools console, and say so in the report.
1. As Alex Author, open `S01 test <time>` on the `Breathing` Question Set. From the address, copy Hospital A's id (after `/h/`) and the Question Set's id (after `set=`, up to `&`).
2. Open DevTools → Console and run, with the two ids filled in:
   ```js
   fetch("/api/h/<A>/chapters/<SET>", {method: "PATCH", headers: {"Content-Type": "application/json"}, body: '{"name": null}'}).then(async r => console.log(r.status, await r.text()))
   ```
3. **Expected:** it prints `400` and a message such as "The request could not be read." It must not print a network error such as "Failed to fetch".
4. Reload the editor. **Expected:** the Question Set is still called `Breathing`.

### T-14: The integration tests pass (terminal)
1. In a terminal, in the project folder, run `make test-integration`.
2. **Expected:** it ends without `FAIL`. Every line starting `ok` is fine. Copy the last 8 lines into the report, including the `internal/apitest` and `internal/httpapi` lines.

### T-15: Clean up
1. As Alex Author, delete `S01 test <time>` from the list (⋯ → **Delete** → confirm). **Expected:** it disappears.
2. Try to delete `S01 publish <time>` (its ⋯ menu offers Delete, because its latest version is a draft). **Expected:** it's refused with "Only drafts can be deleted. Published versions can be retired instead." Leave it.
3. List in the report what you created and what you deleted.

## 3. Regression

These are earlier work, checked briefly.

- **R-01:** As Alex Author, click **Preview** on `S01 test <time>` (before deleting it in T-15). **Expected:** the patient view shows `Do you smoke?`; answer Yes and the outputs panel shows "Current smoker".
- **R-02:** Open **HJE Full HQ** and click through two Question Sets. Don't change anything. **Expected:** cards render, and the console shows no errors.
- **R-03:** Sign in as **Cara Clinician**. **Expected:** you land on **Episodes**, and an existing episode (if there is one) opens without errors.
- **R-04:** Two tabs on the same Question Set of `S01 test <time>`: rename a question in tab 1 and wait for "All changes saved"; then rename a different question in tab 2. **Expected:** tab 2 shows "Someone else changed this Question Set" with a **Reload** button, and tab 1's change isn't lost.

## 4. Out of scope (don't report)

- The automated tests themselves: how they're written or named.
- The header still says "Lifebox" in places. That's a known item, renamed later.
- Anything about episodes, the patient link or the POA Summary beyond R-03. S02 covers those.
- Known limits listed at the end of [docs/manual-test-plan.md](../../manual-test-plan.md).
- Leftover published test questionnaires from earlier sessions; they can't be deleted.
