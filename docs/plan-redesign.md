# Redesign plan: Lifebox-style editor, clinician display, HJE Full HQ

Agreed 1 Oct 2026. We copy the **layout and behaviour** of the Lifebox Author tool, not its colours or fonts. What we're copying is described in [lifebox-ui-notes.md](lifebox-ui-notes.md).

## Decisions
- **Our own editor UI, no SurveyJS Creator.**
  - The editor edits the chapter's SurveyJS JSON directly as plain data, with undo kept as snapshots.
  - survey-core still runs the preview and the outputs calculation, so `packages/clinical` stays the single place for clinical rules.
  - This also removes the need for a Creator licence.
- **Lifebox terms are used in the UI:**
  - **Question Set** = our chapter.
  - **Page** = a SurveyJS page.
  - **Disclosures** = our clinical outputs. We keep our richer output model (codes, note + category, ASA, flag) and show it in Lifebox's layout.
- **Storage is unchanged:** one SurveyJS JSON per chapter. Page settings live in that JSON:
  - name: `p_xxxxxx`
  - title
  - `visibleIf`
  - `clinicalSummary`
- **"Create the HQ" means importing HJE Full HQ** from a Lifebox JSON export (see `tools/export-lifebox-hq.js`) into our tool.

## Mapping Lifebox types to SurveyJS JSON
| Add button | JSON |
| --- | --- |
| Group | `panel` |
| Section | `panel` with `patientPage: true` and `showAsHeading` (STR-03) |
| Yes / No | `radiogroup` with `yesNo: true`; the two options have stable IDs and are shown side by side |
| Select One | `radiogroup` |
| Select Many | `checkbox`; "None" is a choice with `isExclusive: true` |
| Text | `text` (Short) or `comment` (Long) |
| Date | `text` with `inputType: "date"` |
| Number | `text` with `inputType: "number"` (kept from DSG-01) |
| Medication | custom type `medication`: a repeating list of name, dosage and frequency, plus `medicationType` |
| Admissions | custom type `admissions`: reason, hospital, year, anaesthetic type |
| BMI | custom type `bmi`: height, weight and the calculated BMI; must be alone on its page |
| Profile | custom type `profile`: read-only patient details from the sample patient |
| Statement | `html` |
| Clinical summary | a page flag, `clinicalSummary: true`, switched by the toggle in the page header |

- **Clinician items:** a `clinicianOnly` element shows a purple **CLINICIAN** badge on its card. In the Settings panel, the switch for it is labelled **Clinical**.
- **Logic:** `visibleIf` on elements and pages.
  - The builder writes one condition:
    - `{q} = 'o'` or `{q} <> 'o'` for Select One and Yes / No
    - `{q} contains 'o'` or `{q} notcontains 'o'` for Select Many
  - The Code tab edits the raw expression.
  - Cards show "Displays when `<question>` is / is not `<option>`" and fall back to the raw expression when it isn't one simple condition.
  - AND/OR in the builder (LOG-04) comes later.

## Steps (each ends runnable, with tests where there's logic)

1. **Editor model** (`packages/clinical`, with Vitest). Pure functions over chapter JSON:
   - add, copy (new IDs, outputs kept), move, delete and update elements and pages
   - the type catalogue with defaults
   - parse and format a simple condition
   - list the questions an element may depend on
2. **Questionnaire editor shell.** Replace the questionnaire page with:
   - the header (← Questionnaires / name, saving state)
   - the **Structure** column: Question Set accordions → pages, ⋯ menus, Add Page, Add Question Set
   - the canvas:
     - when a set is selected: name and icon card
     - when a page is selected: Clinical Page header, Clinical summary toggle, page logic chip
   - Autosave per chapter, using the existing revision check, and undo.
3. **Cards.** One read-only lookalike per type, with:
   - a PATIENT or CLINICIAN badge, and an orange border plus "Displays when" box for conditional elements
   - `🔗 n` chips
   - hover actions: clone, up, down, delete
   - the Add content bar, with the BMI rules
4. **Settings panel.** **Settings | Disclosures | Logic** tabs in the left column, replacing the tree while a card is selected.
   - **Settings:** fields per type, options list, "None" special option.
   - **Disclosures:** a row per option, with a modal in Lifebox's layout.
   - **Logic:** Always / Conditionally builder, plus a Code tab.
5. **New question types** (registered in `packages/clinical`, so the preview renders them): Medication, Admissions, BMI, Profile, Section.
6. **Clinician display in the preview:**
   - Patient view stays one column.
   - Clinician view puts patient items on the left and clinician items on the right as boxes, plus the page's **Clinical summary** box (this page's notes and a comments field).
7. **HJE Full HQ import.** A Go command converts the Lifebox export into questionnaire → chapters → SurveyJS JSON:
   - text records → strings
   - `renderIf`/`visibleIf` → our expressions
   - disclosures → outputs, matching codes in our library
8. **Remove the Creator** (the designer route and `survey-creator-*` packages), then update the README demo script and tests.

## Status (1 Oct 2026)

Steps 1–8 are built.
- **Tests:** `make test` and `make lint` pass. The editor, preview and import were checked in a headless browser.
- **HJE Full HQ:** imported from the screenshot transcription. That covers sets 1–4 with no disclosures.
- **Still to do:** import the real Lifebox export, to get all 11 sets and their disclosure codes.
- **Done since, in plan-features.md:** Sections as patient screens (STR-03), several dates, and AND/OR in the Builder (LOG-04).
