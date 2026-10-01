# Manual test plan: questionnaire editor and preview

Work through these cases in Chrome, on the running app, the way a careful manual tester would. Record a result for every case. Background on the product: [plan-redesign.md](plan-redesign.md) and [plan-features.md](plan-features.md).

## Before you start

- **The app** runs at **http://localhost:3000**.
  - If the page doesn't load, stop and report it; don't try to start or fix the servers.
  - Someone needs to have run `docker compose up -d db`, `make migrate` and `make dev`.
- **Sign-in:** click a user's name on the sign-in page; there's no password.

  | User | Hospital | Role |
  | --- | --- | --- |
  | Alex Author | Hospital A | author (can edit) |
  | Val Viewer | Hospital A | viewer (read-only) |
  | Bea Author | Hospital B | author |

- **Data rules:**
  - Use made-up data only. Never type a real person's details.
  - Create your own questionnaire for testing, named `Manual test <today's date> <time>`.
  - Don't change **HJE Full HQ** or any other questionnaire you didn't create. MT-59 only opens HJE Full HQ to look.
  - Stay on localhost:3000. Don't visit or sign in to any Lifebox site.
- **Autosave:** the editor saves on its own. The header shows "Saving…" and then "All changes saved".
  - Wait for "All changes saved" before reloading or leaving a page.
  - Text boxes in the Settings panel save when you click out of them (on blur), not as you type.
- **Terms:**
  - A **Question Set** is a chapter.
  - **Disclosures** are clinical outputs (codes, a note, an ASA grade, a flag) attached to an answer.
  - **Clinical** means clinician-only.
- **Watch the browser console** throughout. Report any red error, with the case ID.
  - Two `401 (Unauthorized)` errors on the sign-in page are expected; ignore them.
  - A `409 (Conflict)` error is expected when the app refuses a change on purpose: in MT-03 (duplicate name) and MT-61 (two editors). Ignore it there.
  - A Next.js "Issue" badge at the bottom left also counts as an error. Open it and copy the message.

## Priority: the four untested features

Cases marked **NEW, untested** cover features that no one has tried in a browser yet:

| Feature | Cases | What they need first |
| --- | --- | --- |
| Moving into a group | MT-17 | A questionnaire with a Question Set, a page and some content: MT-02, MT-06, MT-08, MT-11 |
| Saved option lists | MT-30 to MT-33 | A Select One and a Select Many: MT-11 |
| Nested logic | MT-41 | The earlier questions listed at the start of section I |
| Showing a whole Question Set | MT-48, MT-56 | Two Question Sets with a Select Many in the first: MT-06, MT-10, MT-11 |

Work through the plan in order, because each section builds on the ones before it. Never skip these cases. If time runs short, do just the cases in the last column, then these.

## How to report

For each case, give **Pass**, **Fail** or **Blocked**, plus a one-line note. For each Fail:
- the step that failed
- what you expected and what happened
- a screenshot
- any console error

At the end, list the questionnaires you created and confirm you deleted them (MT-63).

Report the **known limits** at the end of this document only if they behave differently from how they're described there.

---

## A. Sign-in and hospitals

**MT-01: Sign in as an author**
1. Open http://localhost:3000 and click **Alex Author**.
2. Expected:
   - The questionnaire list for Hospital A opens.
   - The header shows "Lifebox Authoring", a **Hospital** picker showing Hospital A, the user's name and **Sign out**.

## B. Questionnaire list

**MT-02: Create a questionnaire**
1. Click **New Questionnaire**. A drawer opens on the right.
2. Try **Save** with an empty Name or Description. Expected: it's refused, with a message.
3. Enter Name `Manual test <date> <time>` and a Description, then click **Save**.
4. Expected: the editor opens, with the questionnaire name in the header next to a "draft" badge.

**MT-03: Duplicate draft name**
1. Go back to the list (**← Questionnaires**). Create another questionnaire with the exact same name.
2. Expected: the error "A draft questionnaire with this name already exists."

**MT-04: Search, open and delete**
1. Type part of your questionnaire's name in **Search by name or description**. Expected: the list narrows to matching rows.
2. Click the row. Expected: the editor opens.
3. Back on the list, open the row's ⋯ menu. Expected: **Edit** and **Delete**.
4. Don't delete yet; that happens in MT-63.

**MT-05: Hospital scope**
1. Click **Sign out**, then sign in as **Bea Author**.
2. Expected: your test questionnaire is not in Bea's list (Hospital B).
3. Sign out, then sign back in as Alex Author.

## C. Structure: Question Sets and pages

Use your test questionnaire, signed in as Alex Author.

**MT-06: Add a Question Set**
1. Click **Add Question Set**.
2. Expected:
   - "Question Set 1" appears in the Structure tree.
   - The right side shows the Question Set screen: name with a pencil, Icon Name, Description, **Who is it for?**, **Patients see**, a progress bar switch, and **Display this Question Set**.

**MT-07: Edit the Question Set**
1. Rename it with the pencil to `Breathing`.
2. Pick an icon.
3. Type a description and click out of the box.
4. Reload the page once "All changes saved" shows.
5. Expected: the name, icon and description are kept, and the same Question Set is still selected (the URL holds it).

**MT-08: Pages**
1. Click **Add Page** twice. Expected: "Page 1" and "Page 2", and clicking a page shows its canvas ("Clinical Page", the page name with a pencil, and a **Clinical summary** switch).
2. Rename Page 1 to `Lungs` with the pencil.
3. Open Page 2's ⋯ menu. Check that **Move up**, **Copy** and **Delete** work: Copy adds "… (copy)", and Delete asks for confirmation.
4. Drag a page above another in the tree. Expected: the order changes.

**MT-09: Undo and redo**
1. Make any change, such as renaming a page.
2. Click **Undo** (↶ next to "Structure"). Expected: the change is reversed.
3. Click **Redo** (↷). Expected: it's applied again.
4. Click on the canvas, away from any text box. Press ⌘Z (Mac) or Ctrl+Z, then ⌘⇧Z or Ctrl+Y. Expected: the same undo and redo.
5. Click into a text box and press ⌘Z. Expected: only the text box's own undo; the questionnaire is not undone.

**MT-10: A second Question Set and reordering**
1. Add another Question Set and rename it `Sleep`.
2. Open its ⋯ menu and click **Move up**, then **Move down**. Expected: the order changes and survives a reload.

## D. Adding content and the rules for it

On the `Lungs` page.

**MT-11: Every type can be added**
1. Under "Add content:", click each button once: Group, Section, Yes / No, Select One, Select Many, Text, Date, Number, Rating, Grid, Calculation, File upload, Signature, Medication, Admissions, Profile, Statement.
2. After each, close the settings panel (✕).
3. Expected:
   - Each makes a card that looks like the patient's question.
   - Questions show a PATIENT badge.
   - Calculation shows a purple CLINICIAN badge (it's clinician-only by default).
   - Profile shows a "Locked" badge.
4. Skip BMI here; the next case covers it.

**MT-12: BMI must be alone on its page**
1. On the `Lungs` page, which has content, look at the **BMI** button. Expected: it's disabled.
2. Add a new empty page and click **BMI**. Expected: a BMI card appears and every other Add content button is now disabled.
3. Delete that page.

**MT-13: A Section can't go inside a group**
1. Click a **Group** card to select it. An Add content bar appears inside it.
2. Expected: **Section** is disabled there.
3. Add a **Text** inside the group. Expected: the new card sits inside the group.

**MT-14: Delete with dependents**
1. Add a Yes / No question, then a Text question. On the Text question, open the **Logic** tab, set **Display** to Conditionally, and pick the Yes / No question, **is**, **Yes**.
2. Delete the Yes / No card with its trash icon.
3. Expected:
   - The confirmation lists the Text question under "Other items depend on this".
   - Cancel keeps it.

## E. Cards

**MT-15: Hover actions**
1. Hover a question card. Expected: Copy, Move up, Move down, ⋯ (More actions) and Delete appear.
2. **Copy:** a copy appears below, titled "… (copy)".
3. **Move up / Move down:** the card moves. The buttons are disabled at the ends.

**MT-16: Edit the title on the card**
1. Click a question card to select it. Its title becomes an editable line with a dashed underline.
2. Type a new title and press Enter.
3. Expected: the title updates, and so does the **Question text** box in the Settings panel.
4. The other way round: change **Question text** in Settings and click out. Expected: the card's title updates.

**MT-17: Move into a group and out again · NEW, untested**
1. With a Group on the page, hover a question outside it, open **⋯ More actions** and look under **Move into**.
2. Expected:
   - The Group and the Section are listed, each labelled "(Group)" or "(Section)".
   - Choosing the Group moves the question inside it.
3. Open the same question's ⋯ menu again. Expected: **Move out of "Group"**. Choose it; the question appears right after the group.
4. Open a **Section** card's ⋯ menu. Expected: no Move into entries (a Section can't go inside a group).

**MT-18: Move to page**
1. On a question's ⋯ menu, under **Move to page**, pick another page.
2. Expected: the card leaves this page and appears at the end of that page.

**MT-19: Drag and drop**
1. Drag a card above another card. Expected: it's placed there.
2. Drag a card onto an empty group's dashed area. Expected: it moves into the group.

## F. Settings for each type

Select the card, then use the **Settings** tab on the left.

**MT-20: Settings every question has**
1. On a Text question:
   - Change **Question text** and **Description**.
   - Set **Show the description** to "Under the answer" (it appears once there's a description).
2. Switch **Required** off. Expected: the card shows "(optional)".
3. Switch it on and type a **Message when it's missing**.
4. Switch **Clinical** on. Expected: the card's badge becomes the purple CLINICIAN.
5. Switch **Clinical** off again; MT-51 needs this question on the patient's form.

**MT-21: Text**
1. Set **Input length** to Long. Expected: the card shows a taller box.
2. Back on Short:
   - **Placeholder** `e.g. 943 476 5919`
   - **Maximum length** 12
   - **Format** "NHS number"
3. Expected: the card shows the placeholder and the badges "NHS number" and "max 12".
4. Set **Format** to "Custom pattern". Expected: the **Pattern** box appears, starting with the NHS number rule.
5. Enter the pattern `^[A-Z]{3}$` with a message. Expected: no error, and Format stays on "Custom pattern".
6. Set **Format** back to "NHS number"; MT-51 uses it.

**MT-22: Number**
1. On a Number question, set:
   - **Minimum** 2, **Maximum** 400
   - **Decimal places** 1
   - **Unit** kg
   - **Soft warning** Above 200
2. Expected: the card shows "kg", "2 to 400", "1 dp" and a yellow "soft warning".
3. Reload. Expected: all the values are kept.

**MT-23: Date**
1. On a Date question, try each **Format**: Day, month and year; Month and year; Year only. Expected: the card's box shows dd/mm/yyyy, mm/yyyy and yyyy.
2. Choose **Allowed dates** "Past only". Expected: a "past only" badge.
3. Switch on **Allow several dates**. Expected: the card shows an "Add another date" button.
4. Switch **Allow several dates** off again and set **Format** back to "Day, month and year". MT-27 and MT-40 need a single full date: a question that takes several dates is offered only as "is answered" in conditions, and not at all in "Years since a date".

**MT-24: Select One and Select Many**
1. On Select One:
   - Edit the option labels.
   - Reorder with the arrows. Expected: the card follows.
   - Add an option.
   - Delete an option. Deleting is disabled below 2 options.
2. Set **Show options as** to "Dropdown list". Expected: the card shows a dropdown box, and a **Placeholder** field appears.
3. On Select Many:
   - Switch on **No to all**. Expected: "None of the above" appears after "or".
   - Switch on **Don't know** and **Prefer not to say**. Expected: they appear after "None of the above".
   - Switch on **Other (please specify)**. Expected: an "Other (please specify)" row with a text box.
   - Switch on **Select all**. Expected: an italic "Select all" row at the top.
4. On Select Many, set **At least** 1 and **At most** 2. Expected: "Ticks: 1–2" under the options.
5. Set **Show options as** to "Searchable list". Expected: the card shows a search box.

**MT-25: Scores**
1. On a Yes / No question, switch on **Scores** and give Yes the score 1.
2. On a Select Many, switch on **Scores** and give two options 2 and 3.
3. Expected: the cards show "1 pt", "2 pts" and "3 pts" next to the options.

**MT-26: Rating, Grid, Upload**
1. **Rating:**
   - Set **From** 0, **To** 10, **Low label** "No pain" and **High label** "Worst pain".
   - Expected: the card shows the boxes 0 to 10, with the labels under them.
2. **Grid:**
   - Add a row and a column, rename them, and reorder them.
   - Expected: the card's table follows.
3. **File upload:**
   - Set **Allowed files** "PDFs only", **Largest file (MB)** 2 and **Allow several files**.
   - Expected: the card says "PDFs, up to 2 MB, several files".

**MT-27: Calculation and bands**
1. Make sure some questions have scored options (MT-25).
2. On the Calculation, choose **Calculate** "A score total", then in **Add up the scores of** pick the scored questions.
3. Expected: the card says "Total of the scores of N questions".
4. Click **Add band** twice. Expected: two bands, "Low ≤ 2" and "High above".
5. Rename the first band to `Low risk`. Change its maximum to 3. Expected: the card's badges update.
6. Change **Calculate** to "Years since a date". Pick a Date question in **Date** (it must be "Day, month and year"). Expected: the card says "Years since a date".
7. Change it to "A custom formula" and type `{q_doesnotexist} * 2`. Expected: an "Unknown question" error under the box.
8. Set **Calculate** back to "A score total" and pick the scored questions again; MT-45 and MT-54 use it.

**MT-28: Repeatable group and Section heading**
1. Select a Group and switch on **Repeatable**. Set **At least** 1, **At most** 5 and **Add button** "Add operation".
2. Expected: the card shows a blue "Repeats 1–5" badge and an "Add operation" button.
3. Switch Repeatable off. Expected: back to a normal group, with its questions kept.
4. Select a Section. Expected: a **Show as patient page heading** switch, and no Description or Clinical.

**MT-29: Change type**
1. On a Select One with exactly two ordinary options, click **Change type** in the panel header.
2. Expected: Yes / No and Select Many are offered.
3. Change to Select Many. Expected: checkboxes, with the options kept.
4. Do the same for a Text question. Expected: Number and Date are offered.

## G. Saved option lists

**MT-30: Save a list · NEW, untested**
1. On a Select Many with options `Asthma`, `COPD` and `Sleep apnoea`, click **Save as a list**.
2. Enter the name `Lung conditions <time>` and click **Save list**.
3. Expected: "Saved "…"", then **Done**.

**MT-31: Use a list · NEW, untested**
1. Add a Select One and click **Use a saved list**.
2. Expected: the list is shown with its options as chips, the user's name and a date.
3. With "Add to the current options" selected, click **Add**. Expected: the three options are added after the existing ones.
4. Open it again, choose "Replace the current options" and click **Replace**.
5. Expected:
   - A warning about conditions shows before you click.
   - Afterwards only the three saved options remain.

**MT-32: The list is a copy · NEW, untested**
1. Add an option to the first question, so the list really changes. Then save its options again under the **same name**.
2. Expected: a warning that a list with this name exists, and the button reads "Replace saved list". Save.
3. Expected: the second question's options don't change.

**MT-33: Delete a list · NEW, untested**
1. In **Use a saved list**, click the trash icon on your list.
2. Expected: it disappears, and the questions keep their options.

## H. Disclosures (clinical outputs)

**MT-34: On an option**
1. On a Yes / No, open the **Disclosures** tab and click **Add** next to Yes.
2. Click **+ Add disclosure**.
3. Under Codes, search SNOMED for `smok` and pick a code.
4. Type the Note `Current smoker` and choose Category "Lifestyle".
5. Set the flag to Amber and click **Save output**, then **Done**.
6. Expected: the card shows a green 🔗 1 chip next to Yes. Clicking the chip reopens the disclosure.

**MT-35: On a question**
1. On a Number, add a disclosure with the note `Smokes {answer} per day`.
2. Expected:
   - After saving, the modal's list shows the disclosure with an example ("e.g. …") with the answer filled in.
   - The card shows 🔗 1 next to the box.

**MT-36: On a grid cell and a band**
1. On a Grid, open **Disclosures**. Expected: one row per cell, grouped under each row's heading.
2. Add a disclosure to one cell. Expected: 🔗 1 in that cell on the card.
3. On a Calculation with bands, open **Disclosures**. Expected: one row per band. Add one to the high band.

**MT-37: Options from the code library**
1. On a Select Many, click **From the code library**, search for and pick two codes, then click **Add 2 options**.
2. Expected: two options named after the codes, each already showing 🔗 1.

**MT-38: A disclosure needs content**
1. Try **Save output** with no code and no note. Expected: it's refused, with a message.

## I. Logic

Use the Logic tab of a selected card. Make sure earlier questions exist: a Yes / No "Do you smoke?", a scored Select Many, a Number, a Date and a Calculation with bands. Then add a Text question at the end for these tests.

**MT-39: One condition**
1. On the Text, set **Display** to Conditionally.
2. Pick the question "Do you smoke?", the comparison **is** and the value **Yes**.
3. Expected:
   - The card gets an orange left border and a branch icon.
   - It shows a box: "**Displays when** Do you smoke? **is** Yes".
   - Clicking that box opens the Logic tab.

**MT-40: Comparisons follow the question type**
1. Add conditions on different kinds of question and open the comparison list each time. Expected:
   - **Yes / No or Select One:** is, is not, is any of, is none of, is answered, is not answered.
   - **Select Many:** includes, does not include, includes any of, includes all of, …
   - **Number:** =, ≠, >, ≥, <, ≤, …
   - **Date:** is before, is after, is on, was at least … years ago, was less than … years ago, …
   - **Text:** is answered, is not answered, is exactly, contains.
2. In the question list, check that "Patient's age", "Patient's sex" and "Who is viewing" are offered under "Patient and viewer".
3. With a Calculation that has bands, check that "<name> band" is offered with the band names as values.

**MT-41: AND, OR and nested groups · NEW, untested**
1. Build this:
   - Do you smoke? is Yes
   - **Add group**:
     - Patient's age ≥ 65
     - **Add group inside**, set to **Match None**: Patient's sex is Male, and Who is viewing is Clinician.
2. Expected:
   - Each group is a bordered box, and you can nest them.
   - Each box has Match All / Any / None / Not all.
   - The card's sentence shows the groups in brackets, with "none of ( … )" for the None group.
3. Open the **Code** tab. Expected: an expression with `and`, `or` and `!( … )` that matches what you built.
4. Remove a whole group with its trash icon. Expected: the sentence updates.

**MT-42: Required when and read-only when**
1. On the same Text, set **Required** to "Only when" and build a condition.
2. Expected:
   - The card shows "Required when …".
   - In Settings, the Required switch is replaced by a note pointing to the Logic tab.
3. Set **Read-only** to "When" and build a condition. Expected: the card shows "Read-only when …".
4. Set both back to Never or Always. Expected: the boxes go.

**MT-43: Conditions on options**
1. On a Select Many's Logic tab, under **Options**, click **Add condition** for one option.
2. In the modal, add Patient's sex is Female and click **Done**.
3. Expected: that option shows the condition in plain words in the Logic tab, and an orange branch icon on the card.

**MT-44: Code tab checks**
1. In the **Code** tab, type `{q_smoke} =` and look at the error. Expected: "This isn't a valid expression."
2. Type `{q_nothere} = 'x'`. Expected: "Unknown question: q_nothere".
3. Click **Reset**. Expected: back to the saved expression.

**MT-45: Used by**
1. Open the Logic tab of "Do you smoke?". Expected: **Used by** lists each item whose condition tests it, plus its disclosure count.

**MT-46: Logic problems in the header**
1. Drag the conditional Text question above "Do you smoke?".
2. Expected:
   - An orange "1 logic problem" button appears in the header.
   - Clicking it lists "…: it tests a question that comes later or itself".
   - Clicking the item opens that question's Logic tab.
3. Undo. Expected: the button goes.

**MT-47: Page logic and skip rules**
1. On page 2's ⋯ menu, choose **Logic**.
2. Set **Display** to Conditionally, on a question from page 1. Expected:
   - The page's row in the tree shows a branch icon.
   - The page canvas shows "Displays when …".
3. Under **Skip rules**, click **Add skip rule** and build "Do you smoke? is No".
4. Expected: the rule stays on Page 2, with **Then** set to "End this Question Set". Keep it; MT-57 uses it.
5. Page 2 is the last page, so its rule offers no "Skip to". Open **Logic** on the `Lungs` page's ⋯ menu, add the same rule there and change **Then** to "Skip to “Page 2”". Expected: it's kept.
6. **Remove** the `Lungs` rule.

**MT-48: Showing a whole Question Set · NEW, untested**
1. Select the second Question Set (`Sleep`). Under **Display this Question Set**, choose Conditionally.
2. Expected: the question list offers questions from the first set, grouped "Breathing › Lungs", plus the patient.
3. Pick a Select Many question from Breathing, **includes**, and an option.
4. Expected: the `Sleep` row in the Structure tree gets an orange branch icon.
5. Select the first set (`Breathing`) and set its Display to Conditionally. Expected:
   - Only "Patient and viewer" questions are offered.
   - A note says it's the first Question Set.
   - Set it back to Always.
6. Move `Sleep` above `Breathing` (⋯ → Move up). Expected:
   - With `Sleep` selected, the header shows a logic problem: "This Question Set: it tests something that isn't in an earlier Question Set".
   - Move it back. Expected: the problem goes.

## J. Translate

**MT-49: Translate texts**
1. With a Question Set selected, click **Translate** in the header.
2. Expected: a table with Where, English and a language column, defaulting to German.
3. Expected: rows for page names, question texts, options, descriptions and messages. Clinician-only questions and disclosure notes are **not** listed.
4. Type a German translation for one question and click out of the box. Expected:
   - That row is no longer flagged red.
   - The Missing count goes down.
5. Switch to "Missing". Expected: only untranslated rows.
6. Close Translate. Change that question's English text in Settings. Reopen Translate. Expected: the German translation is still there.

**MT-50: CSV export and import**
1. Click **Export CSV**. Expected: a CSV downloads, with columns id, where, field, english, de.
2. Fill in a translation in the CSV's `de` column and click **Import CSV** to load it back. Expected:
   - The message counts only the cells you filled in ("Imported 1 translation."); blank cells are ignored.
   - The table shows the new translation, and the one from MT-49 is still there.

## K. Preview

Click **Preview** in the editor header, with a Question Set selected.

**MT-51: Patient view**
1. First, in the editor, add a Text inside the Section: an empty Section has no screen of its own. Then open the preview. Expected:
   - Clinician-only questions and statements aren't shown.
   - Page titles aren't shown.
   - Each Section is on its own screen (use **Next**).
2. On a Number with a unit, check that the unit shows after the box.
3. Enter 250 where the soft warning is "Above 200". Expected: a warning, but **Next** still works.
4. Enter 500 where the maximum is 400. Expected: an error that blocks **Next**.
5. In an NHS number field, type `123`. Expected: "Please enter a 10-digit NHS number." when moving on.
6. Leave a required question empty. Expected: the required message (yours from MT-20, if set).

**MT-52: Logic at runtime**
1. Answer "Do you smoke?" Yes and No. Expected: the conditional question appears and disappears as built.
2. In **Sample patient**, change Age and Sex. Expected: conditions on the patient react straight away.
3. In the editor, make a Text question required only when "Do you smoke?" is Yes (**Logic** tab → **Required** → Only when). In the preview, check it's required with Yes and not with No.

**MT-53: Outputs panel**
1. Answer the questions that carry disclosures.
2. Expected: **Clinical outputs** lists the codes, notes by category, "Suggested ASA" when set, and the review flags.
3. Change the answers. Expected: the outputs update straight away.

**MT-54: Clinician view**
1. First, in the editor, switch on **Clinical summary** on the `Lungs` page. Then switch the preview to **Clinician**. Expected:
   - Page titles are shown.
   - Clinician items sit in a right-hand column with a tinted background. Both columns are about the same width, and no words break mid-word.
   - With **Clinical summary** on, a yellow summary box lists the page's clinical notes above "Clinical comments".
2. Check the Calculation shows its value and band, for example "3 (Low risk)". Calculations are clinician-only by default.
3. Switch between Patient and Clinician several times. Expected: no errors appear (watch the console and the Next.js Issue badge).

**MT-55: Screen sizes and language**
1. Click **Phone**, then **Tablet**, then **Desktop**. Expected: the form narrows inside a device frame and still works. Long ratings may turn into a dropdown on Phone.
2. With a German translation saved (MT-49), choose **German** in the language picker. Expected: translated texts show in German, and the rest in English.

**MT-56: Question Set condition banner · NEW, untested**
1. Preview the `Sleep` set (from MT-48). Expected:
   - An orange banner: "This Question Set is shown only when …", in plain words.
   - A note that it depends on answers in earlier Question Sets.
2. Change `Sleep`'s condition to test only Patient's age ≥ 70.
3. Preview it with sample age 54. Expected: a red banner saying the sample patient doesn't meet it.
4. Change the sample age to 75. Expected: the banner says the patient meets it.

**MT-57: Skip rule at runtime**
1. In the editor, set Page 2's **Display** back to Always (MT-47 made it depend on "Do you smoke?"). Otherwise answering No hides Page 2 and the preview ends anyway, rule or no rule.
2. Check Page 2 still has the skip rule from MT-47 ("Do you smoke? is No" → End this Question Set). Recreate it if you removed it.
3. In the preview, answer No and fill in page 1's other required questions.
4. Expected: the button at the bottom reads **Complete** instead of **Next** (the rule's condition is met). Click it: the preview ends ("End of the chapter preview.").

**MT-58: Test cases**
1. Answer the form, then type a name in **Test cases** and click **Save answers**. Expected: the case is listed.
2. Click **Run all**. Expected: "All 1 passed" and a green "pass".
3. Go back to the editor and change the note text of a disclosure your answers trigger. Return to Preview (reload it) and click **Run all**.
4. Expected:
   - The case fails.
   - Clicking it lists "− expected, not produced: …" and "+ produced, not expected: …".
5. Click **Load** on the case. Expected: the saved answers, viewer and sample patient come back.
6. Delete the case with its trash icon.

## L. Other checks

**MT-59: HJE Full HQ opens (look only, don't edit)**
1. Open **HJE Full HQ** from the list. Click through each Question Set and a few pages.
2. Expected:
   - Cards render, and conditional cards show "Displays when …".
   - The header shows no logic problems.
   - The console shows no errors.
3. Don't change anything. If you change something by mistake, use Undo straight away and report it.

**MT-60: Viewer is read-only**
1. Sign in as **Val Viewer**.
2. Expected: there's no **New Questionnaire** button, and a row's ⋯ menu says **View** rather than Edit.
3. Open your test questionnaire, which has content by now. Expected:
   - The header shows a "Read only" badge.
   - There are no Add content buttons, no hover actions on cards, no Add Page or Add Question Set, and the settings fields are disabled.
4. Sign back in as Alex Author.

**MT-61: Two editors at once**
1. Open your test questionnaire in two tabs on the same Question Set.
2. In tab 1, rename a question and wait for "All changes saved".
3. In tab 2, rename a different question.
4. Expected: tab 2 shows "Someone else changed this Question Set", with a **Reload** button. Nothing is silently overwritten.

**MT-62: Leaving with unsaved changes**
1. Change a text box and immediately try to close the tab or navigate away.
2. Expected: the browser asks whether to leave, while the save is still pending.

## Clean-up

**MT-63: Delete what you created**
1. As Alex Author, delete every `Manual test …` questionnaire from the list (⋯ → Delete → confirm).
2. Also delete your saved option lists if MT-33 left any.
3. Report the names you deleted.

---

## Known limits (not bugs)

- **Question Set conditions in a one-set preview:** the preview can't see answers from earlier Question Sets. It explains the condition but doesn't apply it, unless the condition tests only the patient.
- **"Other (please specify)"** can't carry disclosures.
- **No publishing:** there's no publish, versions or sign-off yet. Every questionnaire stays a draft.
- **Questionnaire list:** no status filter or sorting.
- **HJE Full HQ** has only Question Sets 1–4 and no disclosures (transcribed from screenshots).
- **Skip rules saved before 1 Oct 2026** have no page recorded, so they're listed under the latest page they test, not the page they were added on.
