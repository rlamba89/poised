# Lifebox Authoring on SurveyJS — Requirements

Sep 30, 2026 · @Rahul Lamba

## 1. Purpose and scope

We are building a new clinical questionnaire authoring tool on SurveyJS. It must do everything the current Lifebox authoring tool does, fix what it does badly, and add what a clinical authoring product needs next.

This document states **what** the tool must do. It does not say how to build it.

### What this document is not

- It is not a design. It does not prescribe a data model, database, tables, APIs, API style, frontend or backend framework, hosting or cloud services. The build team decides those.
- It is not a copy of Lifebox's internal structure. Lifebox's node types, tables and expression grammar are not requirements. Only the behaviour users rely on is.

### Where the requirements come from

1. **Parity:** what clinical editors can do in the Lifebox authoring tool today.
2. **Extended:** Lifebox capabilities that are half-built, restricted or broken today, and must work properly. Examples: preview, undo, ASA grade, more than one output per answer.
3. **New:** what a clinical authoring product should have, much of it cheap with SurveyJS. Examples: sign-off workflow, test cases, translations, per-hospital libraries.

### In scope

- Designing forms: structure, questions, answer options, conditional display, translations and branding.
- Clinical meaning: the codes, clinical notes, ASA grades and review flags that answers produce.
- Preview and testing of forms as a patient and as a clinician.
- Form lifecycle: draft, review, clinical sign-off, publish, versions, copy, export and import.
- Hospitals, roles, shared templates, the code library, reports and audit.

### Out of scope

- Patients and clinicians filling in forms inside the Lifebox product, episodes, patient records and generating clinical documents.
- Integration with any existing Lifebox system. The tool must still output everything a future integration would need (OUT requirements, section 11).

### How to read the requirement tables

| Column | Values |
| --- | --- |
| Origin | **Parity**, **Extended** or **New**, as above |
| SurveyJS | **Built-in**: SurveyJS provides it; we configure it. **Extend**: we add it on top of SurveyJS without changing SurveyJS itself. **Outside**: not a form-engine concern, such as workflow or the code library. |
| Priority | **Must** (first release), **Should** (next release), **Could** (later) |

Each requirement has a stable ID (for example, `STR-03`) for tracking and tests.

## 2. Users and roles

Lifebox has one authoring permission (clinical editor) and no hospital scoping. The new tool needs separate roles so that authoring, clinical sign-off and publishing are held by different people.

| Role | Scope | Can do |
| --- | --- | --- |
| Viewer | Hospital | Browse forms, preview them, read reports |
| Author | Hospital | Create and edit draft forms, test cases and library items; submit for review |
| Clinical reviewer | Hospital | Review drafts, comment, approve or send back. Cannot approve their own edits. |
| Publisher | Hospital | Publish approved forms, retire versions, promote between environments |
| Hospital admin | Hospital | Manage users and roles, branding, allowed question types; delete drafts |
| Code librarian | Platform | Add, retire and import codes; manage code categories and note categories |
| Template owner | Platform | Author and publish master templates that hospitals copy |
| Platform admin | Platform | Manage hospitals and platform settings |

A person can hold different roles in different hospitals. Two groups of people use the forms without using the authoring tool: patients and clinicians. Authors design for both groups.

## 3. Glossary

| Term | Meaning |
| --- | --- |
| Questionnaire | A versioned health questionnaire that a hospital assigns to a patient episode. Lifebox calls it "HQ". |
| Chapter | A part of a questionnaire completed in one go, such as "Heart and circulation". It has a name and an icon, and is either for patients, for clinicians, or a clinician document. Lifebox calls it a "question set". |
| Page | One screen of questions |
| Group | A titled set of questions shown together on a page |
| Repeating group | A group the respondent can add several times ("add another medication") |
| Clinician-only | A question or text that only clinicians see and answer |
| Clinical output | Anything an answer produces for the clinical record: codes, a clinical note, an ASA grade or a review flag. Lifebox calls this a "disclosure". |
| Clinical note | A sentence written into the patient's notes when an answer is given, such as "Takes ramipril 5mg once daily". Lifebox calls it an "episode note". |
| Note category | The heading a clinical note is filed under, such as Cardiovascular or Allergies |
| Code | An entry from a clinical coding system: SNOMED CT, ICD-10, OPCS-4 or dm+d |
| Code set | A coding system |
| Code category | A clinical grouping of codes |
| ASA grade | American Society of Anesthesiologists physical status class, I to VI, with E for emergency |
| Review flag | An amber or red marker telling a reviewing clinician to look at an answer |
| Clinical panel | A ready-made block of questions with fixed clinical meaning: medication, previous admissions, BMI, allergies, profile |
| Clinical summary | A clinician-only box on a page, listing that page's clinical notes, where the clinician adds comments |
| Stable ID | An identifier for a question or answer option that never changes once published, so answers stay comparable across versions |
| Master template | A questionnaire published centrally that hospitals copy and adapt |
| Test case | A saved set of example answers, with the clinical outputs they should produce |

## 4. What SurveyJS is expected to do

SurveyJS replaces Lifebox's custom form model, form renderer and form editor. The new tool should rely on SurveyJS's own form structure and designer, not re-create Lifebox's page, section and group hierarchy.

We add clinical meaning on top of SurveyJS using its supported extension mechanisms. **SurveyJS itself is never modified.**

### 4.1 Expected from SurveyJS as it comes

| Capability | What we rely on SurveyJS for |
| --- | --- |
| Visual form designer | Drag and drop designer, toolbox, property panel, undo and redo, copy and delete of elements |
| Form structure | Pages, groups (panels), repeating groups, one-question-per-screen mode, progress bar, table of contents |
| Question types | Single choice, multiple choice, dropdown, searchable dropdown, yes/no, text, long text, number, date, rating, matrix, file upload, signature, ranking, display text, calculated value |
| Answer options | Add, edit, reorder and remove options; exclusive "none" option; "other" option; conditional options |
| Logic | Show or hide questions, groups and pages; required-if; enable-if; skip and complete triggers; a visual logic editor for non-developers; typed expressions with custom functions |
| Validation | Required, numeric range, text length, pattern, number of selections, custom messages |
| Calculations | Values calculated from other answers, shown or hidden |
| Languages | Translatable text for every patient-facing string; a translation editor; right-to-left support |
| Look and feel | Themes, hospital branding, responsive layout for phone, tablet and desktop, accessibility |
| Preview | Running the form inside the designer, on simulated devices |
| Read-only display | Showing a completed form with its answers |
| Partial completion | Saving and restoring answers part-way through |
| Developer view | The form definition as editable JSON |
| Change events | An event for every answer change and every designer change |

### 4.2 To be added on top of SurveyJS

| Capability | Why SurveyJS does not cover it |
| --- | --- |
| Clinical metadata on questions and answer options (codes, notes, ASA, flags, scores) | SurveyJS has no clinical concepts. These must be addable to any question or option and editable in the designer's property panel. |
| Clinical output calculation | Working out which codes and notes a set of answers produces is our rule, not a form feature |
| Clinician-only content | SurveyJS has no notion of who is viewing. The form must behave differently for patients and clinicians. |
| Clinical panels | Medication, admissions, BMI, allergies and profile are our reusable blocks, built from SurveyJS parts |
| Clinical functions in logic | Age, BMI, score totals and "option selected" checks for use in conditions and calculations |
| Chapters | A questionnaire made of several chapters, each with an icon, audience and its own progress |
| Stable IDs and publish rules | Preventing a published question's identity from being changed |
| Workflow, versions, hospitals, code library, reports, audit | Not form-engine concerns |

### 4.3 Constraints on using SurveyJS

- Every clinical addition must appear in the designer like any built-in setting. Authors should not need to leave the designer to add clinical meaning.
- A form definition must be complete on its own. It carries its clinical metadata, so any system that reads it can compute the same outputs.
- Upgrading SurveyJS must not require re-doing our additions.
- The SurveyJS designer requires a commercial licence. The PDF export add-on, if used, needs its own licence.

## 5. Form library, lifecycle and sign-off

### 5.1 Form library (FRM)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| FRM-01 | Users see a list of their hospital's questionnaires with name, description, version, status, a "promoted" marker, last changed (date and person) and creator. Opening one takes them to it. | Parity | Outside | Must |
| FRM-02 | Users can search the list by name and description, filter by status, sort by name, version or last changed, and page through results. "Clear filters" clears every filter. | Parity; fixes a defect | Outside | Must |
| FRM-03 | Users can search inside questionnaires by question text, option text, code or clinical note, and jump to the match | New | Outside | Should |
| FRM-04 | Authors create a questionnaire with a name and description, both required. Two drafts in one hospital cannot share a name. | Parity | Outside | Must |
| FRM-05 | Authors duplicate any questionnaire as a new one under a new name | Parity | Outside | Must |
| FRM-06 | Drafts can be deleted, after confirmation, by their creator or a hospital admin. Published versions cannot be deleted, only retired. | Parity | Outside | Must |
| FRM-07 | A questionnaire contains ordered chapters. Authors add, rename, describe, reorder, copy and delete chapters, with confirmation before deleting. Each chapter has an icon chosen from a searchable set and an audience: patient, clinician, or clinician document. | Parity; description and audience not editable today | Extend | Must |
| FRM-08 | Authors can copy a chapter from another questionnaire | Extended (Lifebox supports it behind the scenes, but not on screen) | Outside | Should |

### 5.2 Lifecycle and versions (LCY)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| LCY-01 | A questionnaire version is Draft, In review, Approved, Published or Retired | Parity (Draft, Published) + New | Outside | Must |
| LCY-02 | Only drafts can be edited. Other versions open read-only, with an obvious "Create new version" action. | Extended (Lifebox opens published versions as editable, then rejects the save) | Built-in (read-only designer) | Must |
| LCY-03 | Changes save automatically, with a visible saving and saved state. No edit is lost on refresh or network loss. | Parity | Built-in | Must |
| LCY-04 | If two people edit the same chapter at once, the second save is refused with a clear message instead of silently overwriting the first | New (Lifebox: last save wins) | Outside | Must |
| LCY-05 | Every edit can be undone and redone | Extended (Lifebox's undo never works) | Built-in | Must |
| LCY-06 | "Create new version" makes the next version number under the same name, copies all content and keeps every stable ID. There is only one draft per questionnaire at a time. | Parity | Outside | Must |
| LCY-07 | A question or option that has been published keeps its stable ID forever. The ID cannot be edited or reused for something else. Deleting it in a later version is allowed. | Parity | Extend | Must |
| LCY-08 | Authors can compare any two versions and see added, removed and changed items, with changes to clinical outputs highlighted | New | Outside | Should |
| LCY-09 | Publishers can retire a published version. Retired versions stay viewable. | New | Outside | Should |
| LCY-10 | Each version can have effective-from and effective-to dates | Parity (stored, never editable) | Outside | Could |

### 5.3 Review, sign-off and publishing (SGN)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| SGN-01 | An author submits a draft for review with a note. A clinical reviewer approves it with a sign-off statement or returns it with comments. | New | Outside | Must |
| SGN-02 | The approver cannot be anyone who edited that version. The tool records who approved, when and their statement. | New (clinical safety) | Outside | Must |
| SGN-03 | Reviewers can comment on any question, option or clinical output. Comments are threaded and can be resolved. | New | Extend | Should |
| SGN-04 | Publishing is blocked, with a list of problems that each link to the item, when: a condition refers to a missing question or a later one; a single-choice question has fewer than 2 options or a multi-choice fewer than 1; a clinical note has no category; a code is missing or retired; a published stable ID has changed; a page has more than one clinical summary; or a test case fails. | Extended (Lifebox checks some on save and none on publish) | Outside | Must |
| SGN-05 | The publish confirmation names the questionnaire, the version number and the environment. Afterwards, the user is told what happens next. | Parity | Outside | Must |

## 6. Form structure, questions and answer options

### 6.1 Structure (STR)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| STR-01 | A chapter contains ordered pages. Authors add, rename, reorder, copy and delete pages. | Parity | Built-in | Must |
| STR-02 | Authors group questions under a heading on a page. Groups can contain groups. | Parity | Built-in | Must |
| STR-03 | Patients see short screens and clinicians see the whole page. The author marks which groups form one patient screen. For each such group, the author chooses whether the group title is the screen heading or the first question's text is. | Parity | Extend | Must |
| STR-04 | Authors can make any group repeatable ("add another"), with a minimum and maximum number of entries | New | Built-in | Must |
| STR-05 | A chapter can be set to show patients one question per screen | New | Built-in | Should |
| STR-06 | Authors reorder questions and groups by drag and drop and by move up or down, including into another group | Parity | Built-in | Must |
| STR-07 | Copying a question or group copies all its settings, options and clinical outputs. Conditions inside the copy point to the copied questions. | Extended (Lifebox drops outputs and leaves conditions pointing at the original) | Extend | Must |
| STR-08 | Deleting anything asks for confirmation. If other conditions, calculations or clinical outputs depend on it, the warning lists them. | New | Extend | Must |
| STR-09 | The designer shows on each element: who sees it (patient or clinician), whether it is conditional and on what, how many clinical outputs it has, and whether it is locked | Parity | Extend | Must |
| STR-10 | An outline of chapters, pages and groups lets authors jump to any part and reorder it | Parity | Built-in + Extend | Must |
| STR-11 | A clinical panel can require a page to itself. BMI does. | Parity | Extend | Should |

### 6.2 Settings every question has (QST)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| QST-01 | Question text, and an optional description that can also be cleared | Parity; clearing fixes a defect | Built-in | Must |
| QST-02 | Required or optional. New questions are required by default; optional ones show "(optional)". | Parity | Built-in | Must |
| QST-03 | Clinician-only on or off (section 10) | Parity | Extend | Must |
| QST-04 | A stable ID that authors can see but not change after publishing (LCY-07) | Parity | Extend | Must |
| QST-05 | Placeholder text and help text | New | Built-in | Should |

### 6.3 Question types (QT)

| ID | Type | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- | --- |
| QT-01 | Yes / No | Two options, Yes and No. Labels are editable and translatable. Each option can carry clinical outputs. | Parity; editable labels are new | Extend | Must |
| QT-02 | Single choice | At least 2 options. Shown as buttons, or as a dropdown for long lists. | Parity | Built-in | Must |
| QT-03 | Multiple choice | At least 1 option. Optional exclusive "None of these" option (Lifebox "No to all") with an editable label and its own clinical outputs. Optional "Select all". | Parity | Built-in | Must |
| QT-04 | Text | Short (one line) or long (several lines), with optional maximum length and input format (for example, phone number) | Parity; format is new | Built-in | Must |
| QT-05 | Date | Day, month and year; month and year; or year only. The earliest date is 01/01/1900, and the date can be restricted to past or future. Authors can allow several dates as a list. | Parity (full date, multiple); other formats and limits are Extended | Built-in | Must |
| QT-06 | Number | Minimum, maximum, decimal places and a unit label | New | Built-in | Must |
| QT-07 | Searchable list | Pick one or several from a long list, including lists from the code library (for example, medicines) | New | Built-in + Extend | Should |
| QT-08 | Display text | Information only; several paragraphs; can be clinician-only | Parity | Built-in | Must |
| QT-09 | Clinical summary | A clinician-only comments box, at most one per page, that shows the page's clinical notes above it | Parity | Extend | Must |
| QT-10 | Grid | Rows by columns of choices, where each cell's choice can carry clinical outputs | New | Built-in + Extend | Should |
| QT-11 | Rating or pain scale | For example 0 to 10, with end labels | New | Built-in | Should |
| QT-12 | File or photo upload | With allowed types and a size limit | New | Built-in | Should |
| QT-13 | Signature, ranking | — | New | Built-in | Could |
| QT-14 | Clinical panels | Medication, admissions, BMI, profile, allergies (section 9) | Parity + New | Extend | Must |

Hospital admins can hide question types their authors should not use (New, Should).

### 6.4 Answer options (OPT)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| OPT-01 | Authors add, edit, reorder by drag and drop, and delete options, directly on the form | Parity | Built-in | Must |
| OPT-02 | Each option has a stable ID separate from its label, so correcting a label does not change what the answer means | Parity | Extend | Must |
| OPT-03 | Each option can carry clinical outputs (section 8) and a score | Extended (outputs exist today; scores are new) | Extend | Must |
| OPT-04 | "Don't know" and "Prefer not to say" can be added in one click and carry their own clinical outputs | New | Built-in + Extend | Should |
| OPT-05 | An option can be shown or hidden by a condition | New | Built-in | Should |
| OPT-06 | An "Other, please specify" option opens a text box | New | Built-in | Should |
| OPT-07 | A list of options (for example, anaesthetic types) can be saved once and reused in many questions | New | Extend | Could |

## 7. Conditional display and navigation (LOG)

Today, Lifebox authors can show or hide a question or page only when a single earlier option is or is not chosen. The new tool must let clinical authors express real clinical conditions without developer help.

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| LOG-01 | Authors can show or hide a question, group, display text or answer option depending on earlier answers | Parity (questions); options are new | Built-in | Must |
| LOG-02 | Authors can show or hide a whole page, and a whole chapter | Parity (page); chapter is new | Built-in + Extend | Must |
| LOG-03 | A condition can test: an option chosen or not chosen; any or all of several options; a number compared with a value; a date before or after a value; an age; a text answer present or empty; a calculated value; the patient's age or sex; and who is viewing (patient or clinician) | Extended (Lifebox tests chosen options only) | Built-in + Extend | Must |
| LOG-04 | Conditions can be combined with AND and OR, and grouped | Extended (Lifebox's builder allows one condition) | Built-in | Must |
| LOG-05 | Non-developers build conditions by picking a question, a comparison and a value from lists, without typing syntax | Parity | Built-in | Must |
| LOG-06 | Each conditional element shows its condition in plain words, for example "Shown when *Do you smoke?* is *Yes*". Clicking it opens the condition. | Parity | Extend | Must |
| LOG-07 | Advanced authors can type a condition. It is checked as they type, for syntax and for references to unknown questions. | Parity | Built-in | Must |
| LOG-08 | A condition can depend only on questions that come earlier. Circular or forward dependencies are flagged immediately and block publishing. | Parity | Extend | Must |
| LOG-09 | When a question is hidden, its answer is ignored: it does not show other questions and does not produce clinical outputs | Parity | Built-in + Extend | Must |
| LOG-10 | A question can be required only when a condition is met | New | Built-in | Must |
| LOG-11 | A question can be read-only when a condition is met | New | Built-in | Should |
| LOG-12 | Authors can skip the respondent to a later question or page, or end a chapter early, when a condition is met | New | Built-in | Should |
| LOG-13 | Each question lists everything that depends on it: conditions, calculations and clinical outputs | New | Extend | Should |
| LOG-14 | Conditions keep working after an element is copied, moved or its label edited | Extended | Extend | Must |

## 8. Clinical outputs from answers (CLN)

Clinical outputs are what makes this a clinical tool rather than a survey tool. An output is what a given answer means for the patient's record: codes, a clinical note, an ASA grade or a review flag.

In Lifebox today, an author can attach one output to each answer option, or to a text or date question. The output is a note with a category (the answer can be inserted into the note) and optionally one SNOMED CT code plus one ICD-10 code. ASA grade is shown but cannot be set. Outputs from hidden questions are not produced.

### 8.1 Defining outputs

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| CLN-01 | Authors attach clinical outputs to any answer option of Yes/No, single-choice, multiple-choice, searchable-list and grid questions, including "None of these" and "Don't know" | Parity | Extend | Must |
| CLN-02 | Authors attach clinical outputs to a question without options (text, number, date, clinical panels). The output is produced when the question is answered. | Parity (text, date); others Extended | Extend | Must |
| CLN-03 | An output can be limited to a condition on the answer, for example BMI over 40, or an admission in the last 12 months | New | Extend | Should |
| CLN-04 | Authors can define an output produced by a combination of answers across questions, for example "diabetic" and "takes insulin" | New (Lifebox planned "rule sets" but never built them) | Extend | Should |
| CLN-05 | One option or question can have several outputs | Extended (Lifebox allows one) | Extend | Must |
| CLN-06 | An output contains any combination of: codes (any number, from any code set); a clinical note with a note category; an ASA grade (I to VI, optionally E); a review flag (amber or red). It needs at least a code or a note, and a note needs a category. | Parity (note, one SNOMED CT + one ICD-10 code); multiple codes, ASA and flag are Extended | Extend | Must |
| CLN-07 | A clinical note can include the answer and other answers. Examples: the typed text, the number, a date as dd/MM/yyyy, a list of dates, the chosen option labels, or each row of a repeating group ("Ramipril 5mg once daily"). The editor shows a worked example as the author types. | Parity (inserting one text or date answer); the rest is Extended | Extend | Must |
| CLN-08 | Clinical notes and codes are never translated. They stay as authored when the form is shown in another language. | New | Extend | Must |
| CLN-09 | Authors find codes by typing part of the code or description, within a chosen code set. If a code is missing and the author has permission, they create it on the spot. | Parity | Extend + Outside | Must |
| CLN-10 | A question can record what it asks as a SNOMED CT concept (for example, "History of surgery"), separate from the codes its answers produce | New | Extend | Should |
| CLN-11 | Each question can be assigned to a heading of the clinical summary document (for example, Cardiovascular), so that answers group correctly in any summary later produced | New | Extend | Should |

### 8.2 Seeing and checking outputs

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| CLN-12 | Every option and question shows how many outputs it has, directly on the form. Clicking the count opens them for editing. An option with none shows "+ Add". | Parity | Extend | Must |
| CLN-13 | One screen lists every output in a chapter or questionnaire: the question, the triggering answer, the note, category, codes, ASA and flag. It can be filtered and edited in place. | New | Extend | Should |
| CLN-14 | Outputs are produced only for questions the respondent could see. A clinician-only question produces outputs only when answered by a clinician. | Parity | Extend | Must |
| CLN-15 | The same answers always produce the same outputs, whether in preview, in test cases, or in any other system that reads the form definition | New | Extend | Must |
| CLN-16 | Outputs with the same codes are recognised as the same clinical outcome across all questionnaires, for reports | Parity | Outside | Should |
| CLN-17 | When several outputs carry an ASA grade, the highest is shown as the suggested grade | New | Extend | Should |

## 9. Clinical panels, calculations and validation

### 9.1 Clinical panels (PNL)

A clinical panel is one item in the toolbox with fixed wording and structure. Authors can change only the settings listed for it. Its answers keep the same shape in every version, so data stays comparable over time.

| ID | Panel | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- | --- |
| PNL-01 | Medication | The author chooses prescribed, non-prescribed or recreational; the question wording follows the choice. The respondent adds rows of medicine name, dosage (hint "For example, 500mg") and frequency (hint "For example, twice daily"), all required. New: pick the medicine name from a coded medicines list, optional route, and a "I take no medicines" answer. | Parity + New | Extend | Must |
| PNL-02 | Previous admissions | Rows of reason, hospital, year (1900 to the current year) and anaesthetic type: General, Local, Sedation, Spinal / Epidural, Other, Not applicable, Don't know | Parity | Extend | Must |
| PNL-03 | BMI | Height in cm and weight in kg; BMI is calculated to 1 decimal place and shown. Messages: "Enter a valid height", "Enter a valid weight". New: entry in feet, inches, stones and pounds, and a warning outside a plausible range. | Parity + New | Extend | Must |
| PNL-04 | Profile | Shows the patient's name, date of birth, email, mobile and gender from the hospital record, read-only, with a note on how to correct them. The preview uses a sample patient. | Parity | Extend | Must |
| PNL-05 | Allergies | Rows of allergen (from a coded list), reaction and severity (mild, moderate, severe, anaphylaxis), plus "No known allergies" | New | Extend | Should |
| PNL-06 | Library items | Template owners and hospitals save any group as a named library item. It appears in their toolbox, and inserting it adds a copy with new stable IDs. | New | Extend | Should |
| PNL-07 | Validated scoring tools | STOP-BANG, PHQ-9, GAD-7, Clinical Frailty Scale and Caprini, with locked wording, scoring and codes | New | Extend | Could |

Authors can attach clinical outputs to any panel (CLN-02).

### 9.2 Calculations (CAL)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| CAL-01 | Authors can show a value calculated from other answers, such as BMI, a score total or an age. The value is stored with the answers. | Parity (BMI) + New | Built-in | Must |
| CAL-02 | Options can carry scores. Authors can total them and define score bands (for example, STOP-BANG 0–2 low, 3–4 intermediate, 5–8 high) for use in conditions and clinical outputs. | New | Extend | Should |
| CAL-03 | Clinical functions are available in conditions and calculations: age from date of birth, BMI, time since a date, and "option chosen" | New | Extend | Must |

### 9.3 Validation (VAL)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| VAL-01 | Required questions show "This is a required field." when left empty. Multiple choice shows "Select at least one option". | Parity | Built-in | Must |
| VAL-02 | Authors can set number ranges, text length, allowed patterns, date ranges and the minimum or maximum number of choices | Parity (fixed rules) + New (author-set) | Built-in | Must |
| VAL-03 | Authors can set soft warnings that let the respondent continue, for example "That weight looks unusual — please check it" | New | Built-in + Extend | Should |
| VAL-04 | Validation messages are editable and translatable | New | Built-in | Should |
| VAL-05 | A "Don't know" or "Prefer not to say" answer satisfies a required question | New | Built-in | Should |

## 10. Patient and clinician experience, preview and testing

### 10.1 Who sees what (VEW)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| VEW-01 | Patient view: clinician-only questions, text and summaries are hidden; questions come in short screens (STR-03); clinical page titles are hidden | Parity | Extend | Must |
| VEW-02 | Clinician view: everything is visible, including clinician-only items and clinical summaries. The clinician can answer or change any answer, and sees a whole page per screen. | Parity | Extend | Must |
| VEW-03 | Review view: a completed form, read-only, with review flags highlighted and each answer's clinical outputs shown beside it | Extended (Lifebox has a clinical review mode in its renderer, but authors cannot see it) | Built-in + Extend | Should |
| VEW-04 | Further viewer types can be added later, for example hospital administrator, each with its own visibility rules | Parity (Lifebox has this viewer type) | Extend | Could |

### 10.2 Respondent experience (PX)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| PX-01 | Forms work on phone, tablet and desktop, from 320 px wide | Parity | Built-in | Must |
| PX-02 | Forms meet WCAG 2.2 AA: keyboard use, screen reader labels, contrast, focus order and announced errors | New as a stated requirement | Built-in + verification | Must |
| PX-03 | Back and Next navigation keeps earlier answers | Parity | Built-in | Must |
| PX-04 | Progress is shown per chapter and within a chapter. Chapters show their icon. | Parity (icons) + New (progress) | Built-in + Extend | Should |
| PX-05 | A respondent can stop part-way and resume later on the same page | Parity | Built-in | Should |
| PX-06 | Questions can be pre-filled from known information, and the respondent can change them | New | Built-in | Should |
| PX-07 | Hospital branding: logo, colours and font, previewed live | New | Built-in | Should |

### 10.3 Languages (LNG)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| LNG-01 | Every patient-facing text can be translated, starting with English, German, Spanish and French | Extended (Lifebox stores translations but has no screen for them) | Built-in | Must |
| LNG-02 | A translation screen shows all texts side by side per language, and flags missing translations. Missing text falls back to English. | New | Built-in | Must |
| LNG-03 | Translations can be exported for a translator and imported back | New | Built-in | Should |
| LNG-04 | Clinical notes, codes and other clinician-facing metadata are excluded from translation (CLN-08) | New | Extend | Must |

### 10.4 Preview and testing (PRV)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| PRV-01 | Authors preview any chapter as a patient or as a clinician, switching with one control | Extended (Lifebox has the switch in code but no control) | Built-in + Extend | Must |
| PRV-02 | Preview on phone, tablet and desktop sizes, and in any translated language | Extended | Built-in | Must |
| PRV-03 | While previewing, a side panel shows live the codes, clinical notes, ASA grade and flags that the current answers produce | New | Extend | Must |
| PRV-04 | Preview uses a sample patient (name, age, sex) that the author can change, so conditions on age or sex can be tried | New | Extend | Must |
| PRV-05 | Authors save a preview's answers as a named test case, with the outputs they expect | New | Extend + Outside | Should |
| PRV-06 | Running all test cases shows pass or fail, and for each failure the difference between expected and actual outputs. Any failure blocks publishing (SGN-04). | New | Outside | Should |
| PRV-07 | Test cases carry over to new versions | New | Outside | Should |
| PRV-08 | Test cases hold made-up data only, and the tool reminds authors not to enter real patient details | New | Outside | Must |

## 11. Hospitals, code library, data exchange, reports and audit

### 11.1 Hospitals and templates (TEN)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| TEN-01 | Every questionnaire, library item and test case belongs to one hospital or to the master library. Users see only the hospitals they belong to. | New (Lifebox authoring is shared by everyone) | Outside | Must |
| TEN-02 | A user in several hospitals can switch between them | New | Outside | Must |
| TEN-03 | A hospital copies a master template and adapts it. The copy remembers which template and version it came from. | New | Outside | Should |
| TEN-04 | When a master template publishes a new version, hospitals using it are told and shown what changed. They choose whether to adopt it. | New | Outside | Should |
| TEN-05 | Each hospital sets its allowed question types, languages and branding | New | Built-in + Outside | Should |

### 11.2 Code library (COD)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| COD-01 | A codes screen lets users search by code or description, filter by code set, code category and status, sort any column and page through results | Parity | Outside | Must |
| COD-02 | A code has its code, description, full name, code set, code category, billable flag and status (active or retired) | Parity + status | Outside | Must |
| COD-03 | Code sets at launch: SNOMED CT and ICD-10. Later: OPCS-4, dm+d medicines and local codes. | Parity + New | Outside | Must (first two); Should (others) |
| COD-04 | Creating a code needs code, description, code category and code set, plus a billable choice. The code format is checked for its code set, for example the SNOMED CT check digit and the ICD-10 pattern. | Parity; format checks are new | Outside | Must |
| COD-05 | A code never changes once created. A wrong code is retired, not deleted. Retired codes stay valid in published versions but cannot be added to drafts. | Parity (unchangeable) + New (retire) | Outside | Must |
| COD-06 | Code categories and note categories are managed lists. Lifebox has 13 categories, such as Cardiovascular, Respiratory, Allergies and Anaesthetic; note categories use the same list plus "Unassigned". Only code librarians change them. | Parity | Outside | Must |
| COD-07 | When a code is not in the library, authors can look it up in a national terminology service and add it | New | Outside | Should |
| COD-08 | Code librarians can bulk-import codes from a spreadsheet, seeing what will be added before confirming | New | Outside | Should |
| COD-09 | When an author picks a SNOMED CT code, the tool suggests the matching ICD-10 code from the official map | New | Outside | Could |

### 11.3 Designer set-up (DSG)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| DSG-01 | The toolbox uses clinical names in this order: Group, Yes / No, Select One, Select Many, Text, Date, Number, Searchable list, Medication, Admissions, BMI, Profile, Statement, then Library. Types a hospital does not allow are hidden. | Parity (names and order) | Built-in | Must |
| DSG-02 | The settings panel shows a short, clinician-friendly set of settings, with clinical settings first. Advanced settings are shown only to users allowed to use them. | New | Built-in + Extend | Must |
| DSG-03 | The designer links to the user guide | Parity | Extend | Should |
| DSG-04 | Errors are shown in plain language, for example "A draft questionnaire with this name already exists" | Parity | Outside | Must |

### 11.4 Export, import and migration (EXP)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| EXP-01 | A questionnaire version can be exported as one file containing all its content and every code and category it uses, so another installation can import it | Parity | Outside | Must |
| EXP-02 | Importing a file creates a new questionnaire and never changes existing content. Codes are matched by code set and code, and missing ones are added. | Parity | Outside | Must |
| EXP-03 | A published version can be promoted from QA to Training to Production, or copied within an environment as a test draft named "(Test)". The source is marked "promoted", and the user sees how many chapters and codes were moved. | Parity | Outside | Should |
| EXP-04 | Existing Lifebox questionnaires can be imported and behave the same: same questions, options, conditions and clinical outputs, with the same outputs for the same answers. Anything that cannot be converted is listed. | New (migration) | Outside | Must |
| EXP-05 | Developers can view and edit the form definition as JSON, with errors shown before it is applied | New | Built-in | Should |
| EXP-06 | A form can be printed or saved as PDF with its clinical outputs, for offline clinical review | New | Built-in (add-on) or Outside | Should |
| EXP-07 | A form can be exported in the FHIR Questionnaire standard | New | Outside | Could |

### 11.5 What the tool must output (OUT)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| OUT-01 | The form definition is complete on its own: structure, wording, translations, conditions, calculations and all clinical metadata | New | Built-in + Extend | Must |
| OUT-02 | Answers are recorded against each question's stable ID. Chosen options are recorded by their stable IDs, not their labels. | Parity | Built-in + Extend | Must |
| OUT-03 | Every set of answers records the questionnaire, version and chapter it was given against | New | Extend | Must |
| OUT-04 | For any set of answers, the tool can list the clinical outputs: which question and answer triggered each one, its codes, clinical note and category, ASA grade and flag | Parity | Extend | Must |
| OUT-05 | Each clinical panel's answers keep a fixed shape. For example, medication is a list of name, dosage and frequency. | Parity | Extend | Must |

### 11.6 Reports (RPT)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| RPT-01 | A clinical outputs report lists every output across questionnaires: questionnaire, version, chapter, question, triggering answer, clinical note, category, codes, ASA and flag. It can be filtered by questionnaire, chapter, category, code set present or missing, ASA and audience, and exported to a spreadsheet. It includes outputs that have a note but no codes. | Parity; fixes filters that do nothing and missing note-only outputs | Outside | Must |
| RPT-02 | For any code, users see where it is used: questionnaire, version and question | New | Outside | Should |
| RPT-03 | For a chapter, users see which answer options and questions have no clinical output, to support clinical review | New | Outside | Could |

### 11.7 Audit (AUD)

| ID | Requirement | Origin | SurveyJS | Priority |
| --- | --- | --- | --- | --- |
| AUD-01 | Every change, status change, approval, export and import is recorded: who, when, which hospital, and what changed (before and after). Records cannot be changed or deleted. | New (Lifebox records only the last editor) | Outside | Must |
| AUD-02 | Users see the change history of a questionnaire and can restore an earlier draft state as a new change | New | Outside | Should |

## 12. Non-functional requirements (NFR)

Numeric targets are proposals for the team to confirm.

| ID | Area | Requirement | Priority |
| --- | --- | --- | --- |
| NFR-01 | Clinical safety | The product is developed under DCB0129, with a clinical safety officer and a hazard log. The log covers wrong code, missed output, wrong visibility and wrong translation, and traces each mitigation to requirement IDs and tests. | Must |
| NFR-02 | Parity proof | Every Lifebox questionnaire in scope is imported, and a set of real-world answer scenarios gives the same clinical outputs as Lifebox (EXP-04) | Must |
| NFR-03 | Data protection | The authoring tool holds no real patient data. It complies with UK GDPR and the NHS Data Security and Protection Toolkit. | Must |
| NFR-04 | Access | Single sign-on with multi-factor authentication. Role and hospital rules are enforced by the system, not just hidden on screen. One hospital can never see another's content. | Must |
| NFR-05 | Security testing | Independent penetration test before first live use and yearly after | Must |
| NFR-06 | Integrity | Published versions, codes and audit records cannot be altered by any user or route | Must |
| NFR-07 | Audit retention | Audit records are kept for at least 8 years and can be exported | Must |
| NFR-08 | Performance | A chapter with 60 pages and 600 questions opens in the designer in under 3 seconds. Saves complete in under 1 second. Code search shows results in under 300 ms across 500,000 codes. Preview outputs update within 200 ms of an answer. | Must |
| NFR-09 | Availability | 99.5% during UK working hours, with planned maintenance announced in advance | Should |
| NFR-10 | Recovery | No more than 15 minutes of work lost, and service restored within 4 hours, after a failure. Restores are tested every quarter. | Should |
| NFR-11 | Accessibility | Forms meet WCAG 2.2 AA (PX-02). The designer is usable by keyboard. | Must |
| NFR-12 | Browsers | Current and previous versions of Chrome, Edge, Safari and Firefox; iOS Safari and Android Chrome for forms | Must |
| NFR-13 | Environments | Separate QA, Training and Production environments, with promotion between them (EXP-03) | Should |
| NFR-14 | Upgrades | SurveyJS can be upgraded without re-doing our additions, and a regression suite proves forms and outputs are unchanged | Must |
| NFR-15 | Licensing | Commercial SurveyJS licences are in place for the designer and any add-ons used | Must |

## 13. Release scope and acceptance gates

Release 1 must match Lifebox before anything new ships. Proving parity on real Lifebox content is its exit gate.

| Release | Contains | Accepted when |
| --- | --- | --- |
| Release 1: parity | Every **Must** requirement | Every Lifebox questionnaire in scope imports and gives the same outputs for the same answers (NFR-02). Every Must requirement passes its acceptance test. The clinical safety officer signs the hazard log. |
| Release 2: clinical plus | Every **Should** requirement | A hospital copies a master template, adapts it, passes its test cases, gets clinical sign-off and promotes it to Production |
| Later | **Could** requirements, chosen by hospital demand | Agreed per item |

A requirement is done when its behaviour is demonstrated against the wording in its row, and a repeatable test exists for it.

## 14. Open questions

- [ ] **Patient screens.** Is each marked group exactly one patient screen, with the whole page as one clinician screen (STR-03)? Lifebox behaves this way, but the rule is not written down.
- [ ] **Chapters.** Are chapters always completed and submitted separately, or can a respondent move freely between them?
- [ ] **ASA grade.** Lifebox dropped ASA from its outputs in 2024 and never let authors set it. Is it wanted? Who acts on the suggested grade (CLN-17)?
- [ ] **Terminology source.** Which national source is authoritative for SNOMED CT UK edition and dm+d (COD-07)?
- [ ] **Master templates.** Who may author them? Can hospitals change the clinical outputs of a copied template, or only its wording and branding?
- [ ] **Migration.** Which Lifebox questionnaires must be imported? Must their past answers stay comparable with new ones?
- [ ] **Clinician-only options.** Lifebox has no clinician-only answer options. Are they needed?
- [ ] **Downstream use.** Who will consume the clinical outputs later, and what do they need beyond OUT-01 to OUT-05?
- [ ] **Review flags.** Are amber and red enough, or is a third level needed?
- [ ] **Viewer types.** Is a hospital administrator view needed in Release 1 (VEW-04)?
