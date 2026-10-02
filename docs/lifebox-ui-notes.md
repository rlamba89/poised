# Lifebox Author UI: notes for the redesign

These notes describe the current Lifebox Author tool, which we are copying. They cover layout and behaviour only, not colours or fonts.

**Sources:**
- Author: `web-app/Author/src`
- DesignSystem (DS): `web-app/DesignSystem/lib/systems/Questionnaire`
- API: `web-app/API/nextgen/author`
- The Training environment in Chrome. Screenshots are still to come, because page interaction was blocked.

## 1. Naming

| Lifebox | Poised today | Notes |
| --- | --- | --- |
| Questionnaire, "HQ" (Health Questionnaire) | Questionnaire | HQ *is* the questionnaire, not a template. |
| Question Set (the patient app calls it a "chapter") | Chapter | Has a name, description, icon and sequence. |
| Page ("Clinical Page") | SurveyJS page | Has a name and `renderIf`. |
| Section (group with `PATIENT_PAGE`) | — | A dashed container on the page. Patients probably see each Section as its own page. |
| Group (plain group) | — | A card with nested questions. |
| Question / option | Question / choice | |
| Disclosure (`RuleDisclosureMapping`) | Clinical output | Holds: Group (note category), ICD10, SNOMED, document text (`%s` inserts the answer), ASA (not built yet). There are no flags. |
| Clinical summary (`ClinicalPageSummary`) | — | One per page, switched on or off by a toggle in the page header. |
| `isClinical` (Clinician badge) | `clinicianOnly` | |

## 2. Questionnaire list

- **Header:** a nav bar with Questionnaires | Codes | Questions.
- **Title bar:** "Questionnaires" with a [+ New Questionnaire] button on the right.
- **Filters:** Search, a State filter (Draft/Published) and a "Clear filters" link.
- **Table:** sorted server-side and paged at 10 rows (default sort: updated desc).
  - Columns: Name (in bold, with the description underneath), Version, Last modified ("date by user"), State badge (plus a Promoted badge), Created by, and a ⋮ menu.
- **Row actions (⋮):**
  - Draft: Edit, Duplicate as new HQ, Publish, Delete.
  - Published: View, Create version from this, Duplicate as new HQ, Promote.
- **Create and clone** open a **right-side drawer** with Name, Description and Save. Saving opens the editor.
- **Publish and Delete** use a centred confirmation modal.

## 3. Questionnaire editor: layout

```
+---------------------------------------------------------------------------+
| Questionnaires / <name> (Saving...)                            (?) [RL]   |  sticky header, breadcrumb
+---------------------------+-----------------------------------------------+
| LEFT (1/3)                | CANVAS (2/3, grey, scrolls)                   |
| Structure            [↶]  |  [icon] Clinical Page    Clinical summary [o-]|
| v Question Set 1     (:)  |  Page 1  [✎]                                  |
|   [📄] Page 1        (:)  |  [⑂ Displays when Q is X]   <- page logic chip|
|      ├ Section A          |  +-----------------------------------+        |
|      └ Section B          |  | [Patient]          [⧉][↑][↓][🗑]   |        |  question card,
|   [📄] Page 2        (:)  |  | Question text (optional)          |        |  max-width ~680, centred
|   [+ Add Page]            |  | ( ) Yes  [🔗2]                    |        |
| > Question Set 2     (:)  |  | ( ) No                            |        |
| [+ Add Question Set]      |  | [⑂ Displays when Q1 is Yes]       |        |
|                           |  +-----------------------------------+        |
|                           |  Add content: [Group][Section][Yes/No]...     |
+---------------------------+-----------------------------------------------+
```

- **Two columns only.** When a question is selected, the **Settings panel replaces the Structure tree** in the left column. Closing it with ✕ brings the tree back. No third column.
- **Autosave on every edit:** there's no Save button, and "(Saving...)" appears in the breadcrumb while it saves.
- **Undo** is an icon in the Structure header.

### 3.1 Structure tree (left)
- **Question Sets:** each set is an accordion. Opening one selects it, and the canvas shows the set header. The ⋮ menu has Move up/down and Delete.
- **Pages:** each page row has a file icon, and the active page is highlighted. The ⋮ menu has Move up/down, **Logic** and Delete. Pages can be drag-sorted.
- **Sections:** if a page has Sections, the page row expands to list them. Clicking one scrolls the canvas to it.
- **Add buttons:** [+ Add Page] under each set ("Page N") and [+ Add Question Set] at the bottom ("Question Set N").

### 3.2 Canvas header
- **When a set is selected:** a "Question Set" label, the name (inline edit with ✎ then ✗/✓) and an **Icon** card with a preview and a searchable icon dropdown. No questions are shown.
- **When a page is selected:** a "Clinical Page" label, and a **Clinical summary toggle** on the right. Below are the inline-editable page name and the page logic chip, then the question cards, then the Add content bar.

### 3.3 Question card (canvas)
- **Header row:** a **[Clinician]** badge (accent) if clinical, otherwise **[Patient]**. Clone, Move up, Move down and Delete appear on the right on hover or selection.
- **Body:** a read-only lookalike of the real patient control (DS component). It shows:
  - the title, with "(optional)" if the question isn't required
  - a branch icon if conditional
  - the description
  - the control itself
- **Disclosure chip `[🔗 n]`** next to an option or input, **only when there is a disclosure**. Clicking it opens the disclosure modal.
- **Conditional cards:**
  - a coloured left border
  - a chip "⑂ Displays when `<question>` is / is not `<option>`"; clicking it opens the Logic tab
- **Selected card:** a thicker border and a grey background.
- **Locked types (BMI, Profile):** a 🔒 Locked badge. They can't be selected or moved; only Delete works.
- **Group:** a card holding nested cards and its own Add content bar. When empty it says "Add content or drag and drop existing content in to this Group".
- **Section:** a dashed container with an inline-editable name. Its ⋮ menu has Move up/down, Delete and a **"Show as patient page heading"** toggle. Clicking a Section selects it, and the page's Add buttons then insert into it.
- **Drag and drop** reorders cards.

### 3.4 Add content bar
- **Order:** Group, Section, Yes / No, Select One, Select Many, Text, Date, Medication, Admissions, BMI, Profile, Statement.
- Section isn't offered inside a Group or Section.
- BMI must be alone on its page.
- **Defaults for new questions:** required = true, clinical = false.
  - Select One: Option 1–3. Select Many: Statement 1–3.
  - Text: short. Date: day/month/year.
  - Statement: "Panel statement".

### 3.5 Settings panel (left column, tabs **Settings | Disclosures | Logic**, ✕ to close)

**Settings** starts with a type icon and name, then these fields:

| Type | Fields |
| --- | --- |
| All questions | Name (textarea), Description (textarea), **Required** toggle, **Clinical** toggle |
| Text | Input length: Short / Long |
| Date | Type: Day/Month/Year, **Add multiple** toggle |
| Select One / Many | **Options** list with [+ Add]: drag handle, inline edit ✎, delete 🗑 (min 2 for One, 1 for Many) |
| Select Many | **Special options:** "No to all" toggle with an editable label (mutually exclusive) |
| Yes / No | Basic fields only; labels are fixed |
| Medication | Type: Prescribed / Non-prescribed (rewrites the question name) |
| Group | Name and Description only |
| Statement | Text and the Clinical toggle |

**Disclosures**
- For option types, the tab lists each option with `[+ Add]` or `[🔗 n]`. For Text and Date there's a single row with the question name.
- Other types show "not supported yet".
- The disclosure modal (centred) has these fields:
  - **Group** (note category, required)
  - **ICD10** and **SNOMED** type-aheads side by side
  - **Document text** (required; `%s` inserts the answer)
  - ASA (disabled)
- Its buttons are Cancel, Save and Remove.

**Logic**
- The sub-tabs are **Builder | Code**.
- **Builder:**
  1. Display: **Always / Conditionally**.
  2. A **question picker popover** that drills through Page → Group → Question. Only earlier Yes/No, Select One and Select Many questions can be picked.
  3. **Is / Is not** plus an option. Yes/No uses Yes/No radios instead.
  - Only one condition is supported. Anything more complex says "use the Code tab".
- **Code:** a raw expression editor.
- Page logic uses the same UI and is opened from the page's ⋮ → Logic. For a page, the picker scope is the pages of the current set.

## 4. Clinician questions and the clinician box (runtime)

- **In the editor,** clinician questions look the same as patient ones; only the badge differs: **[Clinician]** vs **[Patient]**.
- **At runtime (hospital and clinician view)** the page is a **2-column grid**:
  - **Left:** patient questions.
  - **Right:** every clinical item (clinician questions and statements, plus the clinical summary) stacked as **tinted boxes**.
  - An optional Workplan drawer can open at the far right.

```
+--------------------------------------+---------------------------+
| Page title                           | +-----------------------+ |
| Patient Q1  ( ) Yes ( ) No           | | Clinician Q  (box)    | |
| Patient Q2  [............]           | +-----------------------+ |
|                                      | +-----------------------+ |
|                                      | | <Page> summary (box)  | |
|                                      | |  • disclosure note 1  | |
|                                      | |  Clinical comments    | |
|                                      | |  [textarea]           | |
|                                      | +-----------------------+ |
| [Next →]                             |                           |
+--------------------------------------+---------------------------+
```

- **Patient view:** one column, with the clinical items hidden.
- **Clinical summary box:**
  - Title: "`<Page name>` summary".
  - A bullet list of the disclosure notes produced on this page.
  - "Clinical comments" with a multiline input. Its hint says comments show on the POA summary.

## 5. HQ (patient-facing Health Questionnaire)

- **Landing page:**
  - Title: "Health Questionnaire".
  - A progress bar: "n of N completed".
  - **To do** and **Done** groups of **chapter cards**. Each card shows the set icon (a ✓ when done), the set name and an "In progress" badge.
- **Opening a chapter** shows one page at a time inside a white tile, with "← Back" and "Next →" buttons. The title is hidden for patients.
- **Question rendering:**

| Type | How it renders |
| --- | --- |
| Yes / No | Horizontal radio blocks |
| Select One | Full-width vertical radio blocks |
| Select Many | Checkboxes, then "or", then the None option |
| Text | Input or textarea |
| Date | Date input, or a repeater ("Add multiple") |
| Medication | Repeater: "No medication added yet" and [Add medication]. The add/edit modal asks for name, dosage and frequency. |
| Admissions | Repeater: reason, hospital, year, anaesthetic type |
| BMI | Height, weight, then "Your BMI is X" |
| Profile | Read-only patient details |
| Statement | Paragraphs |

## 6. Gaps against Poised (to confirm with screenshots)

1. **Questionnaire page:** replace the chapter list and modal with the **Structure tree + canvas** editor. Question Set = chapter; pages and sections are in the tree.
2. **Canvas:** replace the stock SurveyJS Creator canvas with **cards** showing a Patient/Clinician badge, the disclosure chip `[🔗 n]` and a "Displays when" chip.
3. **Settings panel:** replace the Creator property grid with a **left-column Settings | Disclosures | Logic panel**.
4. **Outputs modal:** align it with the Disclosure modal fields. Our flags and ASA go beyond what Lifebox has.
5. **Clinician questions:**
   - Rename "Clinician only" to a **Clinical** toggle.
   - In clinician preview, render clinical items in a **right-hand column of boxes** rather than inline.
   - Add the **Clinical summary** page toggle and box.
6. **New HQ screen:** a chapter-card landing page with progress, plus a one-page-at-a-time runner with Back/Next.

## 7. Seen on the live site (HJE Full HQ screenshots, 1 Oct 2026)

These come from 26 of Rahul's screenshots. They **confirm** sections 2–3 and add the details below. The full question content is in [hje-full-hq-content.md](hje-full-hq-content.md).

### Structure column
- **Question Set rows:** chevron, name, ⋯. Expanding a set shows its pages indented, each with a file icon. The **selected page is a dark filled bar** (white text, ⋯ on the right).
- **Buttons:** an outlined `+ Add Page` under the last page of an open set, and an outlined `+ Add Question Set` at the bottom.
- **Sets in this HQ (11):** About you, Medication and Allergies, Medical and Anaesthetic history, Heart and Blood, Lungs, Organs, Digestive health, Musculoskeletal, Neurological & mental health, Infection risks, Support, lifestyle & discharge planning.
- **No Sections or Groups** are used anywhere in the four sets seen.

### Canvas, set selected
- A small "[set icon] Question Set" label, then the set name as a heading.
- A full-width white card with:
  - a round grey icon preview
  - **Icon Name**, with the hint "Set to 'Folder' for the default icon"
  - a dropdown with values like `User01` or `Medication`

### Canvas, page selected
- **Top left:** "[file icon] Clinical Page", with the page name in bold underneath.
- **Top right:** a **Clinical summary** toggle. It's on for every page seen.
- **Page logic:** if the page has some, a single-line chip appears under the name: `[branch icon] Displays when <question text> is <option>`. Example: the Pacemaker page shows when the device question "is Pacemaker". Page logic can point at a question on **another page** of the same set.
- **Cards:** a single centred column about 450px wide, with roughly 16px between cards.
- **Add content:** the bar sits under the last card and wraps onto two lines when narrow.
  - **BMI** is greyed out on any page that already has content.
  - On the BMI page, **every other button** is greyed out.

### Question card
- **Badge**, first line:
  - `PATIENT`: small grey caps on a light grey background.
  - `CLINICIAN`: small white caps on a **purple** background.
- **Conditional card:** an **orange left border**, plus an orange branch icon before the title (or on its own line for statements).
- **Title** in bold, followed by a small grey "(optional)" when the question isn't required. The **description** sits below in normal text.
- **Displays when box:** for conditional questions, the last thing in the card is a bordered box with `[icon] Displays when | <full question text> | is / is not | <option label>`. It wraps into columns when the question text is long.
- **No hover actions** are visible in the screenshots, which were taken without hovering.

### How each type looks on the canvas
| Type | Look |
| --- | --- |
| Yes / No | Two small bordered radio blocks side by side, "Yes ○" and "No ○". Each is followed by its own `🔗 2` chip, but only if that option has disclosures (some have one on Yes only, some on No only). |
| Select One | The same radio blocks **stacked vertically**, one per option, each with its chip. Examples: GTN (Yes / Partially / No / …) and "Do you need translation…" (Yes / No stacked). |
| Select Many | Small checkboxes with a label and chip each, then a plain "or" line, then the None option with its own label ("No known allergies", "None of the above", "No to all" …). The None option can carry a chip too. |
| Text (short) | A small single-line input (~130px), with a chip if there are disclosures. |
| Text (long) | A small resizable textarea (~140×40), with a chip if there are disclosures. |
| Date | A single-line input. The format hint is in the description: "(Month/Year)" or "(Day/Month/Year)". |
| Statement | The badge, an empty title line, then the paragraph text. Statements are used for **both patient info** ("To ensure your safety…") and **clinician guidance** ("Review the patient's… This entry will appear on the POA Summary."). |
| Medication | A grey panel reading "No medication added yet" with an outlined `+ Add medication` button. |
| Admissions | A grey panel reading "No admission added yet" with `+ Add admission`. |
| BMI | A "BMI Calculator" heading and description, then "What is your height?" [0] cm with a "Switch to feet and inches" link, and "What is your weight?" [0] kg with "Switch to stones and pounds". |
| Profile | A grey panel listing Name, Date of birth, Email address, Mobile phone number and Gender in bold, then an ⓘ note: "If any of these details are incorrect, please contact your hospital…". No Locked badge is visible. |

### Disclosure chip
A small green-outlined pill with a link icon and a count. It shows on almost every option and clinician textbox, and nearly always reads **2**.

### Clinician questions: how they are used
In the editor they sit **inline** in the page flow, mixed with patient cards; there's no separate box. The only differences are the purple badge and the content. Three patterns recur:
1. **Clinician statements:** guidance text, often conditional on a patient answer. Examples: "Patient consents to…", "Learning disability disclosed – review…".
2. **Clinician questions with coded outcomes:** Select Many or Select One where each option has disclosures. Example: "Does the patient require a pacemaker review…".
3. **Clinician free text** headed "This entry will appear on the POA Summary.", (optional), with disclosures.

### Not covered by the screenshots yet
- The Settings | Disclosures | Logic panel (shown when you click a card)
- The disclosure modal (shown when you click a `🔗 2` chip)
- The ⋯ menus
- The questionnaire list
- Sets 5–11
- The patient and clinician runtime views
