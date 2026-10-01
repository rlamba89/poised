# Plan: SurveyJS features in our editor (phases 1 and 2)

Agreed 1 Oct 2026. These are features the SurveyJS Creator offered and our Lifebox-style editor (plan-redesign.md) dropped, plus requirements still open. Everything stays plain SurveyJS JSON, and the editor shows each feature with a form, never code.

## Phase 1: open "Must" requirements

1. **Condition builder v2** (LOG-03/04/05/06/10/11, OPT-05). One builder is used for every condition.
   - Rules are joined by **All / Any**, with one level of sub-groups.
   - Comparisons depend on the question type:
     - **Options:** is, is not, is any of, is none of, includes, includes all of.
     - **Numbers:** = ≠ > ≥ < ≤.
     - **Dates:** before, after, on, at least N years ago.
     - **Text and any question:** answered, empty.
   - It can also test:
     - the patient's age and sex, and who is viewing
     - calculated values and score bands
     - grid rows
   - It is used for:
     - show if (elements, options, pages)
     - required if
     - read-only if (`enableIf = !(…)`)
     - skip rules
   - Cards show the condition in plain words.
   - The parser is SurveyJS's own (`ConditionsParser`), so the Builder and the Code tab always agree.
2. **Settings and validation for each type** (QT-04/05/06, VAL-02/03/04, QST-05):
   - **Number:** min, max, decimal places, unit, and a soft-warning range (a validator with `notificationType: "warning"`).
   - **Text:** maximum length, and a format: email, phone, NHS number, UK postcode, or a custom pattern. These are regex/email validators, so there's no second copy of the rule.
   - **Date:** day/month/year, month/year or year only; past or future only; never earlier than 1900; several dates as a list.
   - **Select Many:** minimum and maximum ticks.
   - **All questions:** placeholder, an editable required message, and where the description sits.
3. **Calculated values** (CAL-01/03): a new **Calculation** element (`expression`). Its formula is built from a form:
   - a score total
   - years since a date
   - BMI from two numbers
   - a custom formula

   It is stored with the answers, and can be hidden from patients.
4. **Repeating groups** (STR-04): a group can be made repeatable. It is stored as `paneldynamic` with a minimum, a maximum and an "Add" label.
5. **Patient screens** (STR-03, STR-05):
   - In the patient view, every Section becomes its own screen.
   - A Question Set can also be set to one question per screen (`questionsOnPageMode`).
6. **Preview** (PRV-02): phone, tablet and desktop widths, plus a language picker.

## Phase 2: "Should" requirements

7. **One-click options** (OPT-04/06, QT-03, VAL-05):
   - "Don't know" and "Prefer not to say" are ordinary options marked `special`, so they keep their own disclosures and satisfy a required question.
   - "Other (please specify)" uses `showOtherItem`.
   - "Select all" uses `showSelectAllItem`.
8. **More types and displays:**
   - Select One as a dropdown (`dropdown`); Select Many as a searchable list (`tagbox`). Options can be imported from the code library, and each one gets its code as a disclosure (QT-07).
   - Rating (QT-11).
   - Grid, with disclosures per cell (QT-10).
   - File upload (QT-12).
   - Signature (QT-13).
9. **Scores and bands** (OPT-03, CAL-02):
   - Options carry a `score`.
   - A Calculation can total the scores and define bands. Each band carries disclosures and can be used in conditions through the `band('<calc>')` function.
10. **Skip rules** (LOG-12): "skip to page" and "end this Question Set" are stored as SurveyJS `triggers`.
11. **Editor comforts:**
    - redo (LCY-05)
    - editing the title directly on the selected card
    - changing a question's type where the answers are compatible
    - moving an element to another page
    - "Used by" on each question (LOG-13)
    - logic problems flagged in the header (LOG-08)
12. **Translations** (LNG-01/02/03/04):
    - A Translate view per Question Set shows English beside the chosen language and flags gaps.
    - CSV export and import.
    - Texts are stored as SurveyJS localisable strings (`{ "default": …, "de": … }`).
    - Clinical notes and codes are never offered for translation.
13. **Test cases** (PRV-05/06/07/08):
    - In the preview, the current answers, viewer and sample patient can be saved as a named case with the outputs they produce.
    - "Run all" shows pass or fail, and the difference for each failure.
    - Cases are stored in the chapter JSON under `testCases`, so they carry over to new versions. They are removed before patients get the form.
    - Blocking publishing waits for the publish workflow, which doesn't exist yet.

The Lifebox importer also stops warning about "several dates" and imports them as a date list.

## Status (1 Oct 2026)

Both phases are built.
- **Tests:** `make test` and `make lint` pass (84 Vitest tests and the Go tests).
- **Browser checks:** `spikes/ui-features.ts` and `ui-features2.ts` drive every feature in a headless browser with no page errors. The earlier editor, preview and HQ checks still pass.
- **SurveyJS quirk fixed:** a Rating or File upload that changes width just after rendering (the clinician layout does this) left SurveyJS resize timers running on the replaced survey. The preview now disposes the old survey once the new one is showing.
- **Still open:**
  - Publish blocking (needs a publish workflow).
  - Disclosures on "Other".

## Follow-ups (1 Oct 2026)

- **Builder depth (LOG-04):** groups nest to any depth. A group can match All, Any, None or Not all of its items, so the `!(… or …)` conditions Lifebox imports also open in the Builder. Only arithmetic and other functions need the Code tab.
- **Question Set conditions (LOG-02):**
  - `chapterVisibleIf` in the chapter JSON tests questions in earlier Question Sets and the patient.
  - It is set on the Question Set screen and marked in the Structure tree.
  - The header flags conditions that test the set's own questions or missing ones.
  - `isChapterShown()` evaluates it for a runtime.
  - The preview explains it, and decides it when it tests only the patient.
- **Moving into a group (STR-02):** the card's ⋯ menu offers **Move into** each group and Section on the page, plus **Move out of** the current group. A Section never goes inside a group.
- **Saved option lists (OPT-07):**
  - **Storage:** an `option_lists` table per hospital (migration 00003). The routes are `GET`/`POST /api/h/{hid}/option-lists`, then `PUT`/`DELETE /api/h/{hid}/option-lists/{id}`, for authors only.
  - **Using:** a question's options can be saved under a name, and any Select One or Select Many question can add or replace its options from a list.
  - **Copies:** the options are copied with new IDs and disclosure IDs (OPT-02), so changing a list never changes questions that used it.

## Fixes from the manual test run (1 Oct 2026)

| Case | Problem | Fix |
| --- | --- | --- |
| MT-56 | Changing a rule's question deleted the condition and flipped Display to Always | The Builder saves only once every rule is complete (`isAllComplete`). Until then, the saved condition stays as it was. |
| MT-27 | Calculate always fell back to "A score total" | The chosen type is stored as `calcKind`, so an empty or custom formula keeps its type. |
| MT-21 | "Custom pattern" couldn't be chosen | Custom is stored as `textFormat: "custom"`, and it starts from the previous format's pattern. |
| MT-54 | The clinician column was 158 px wide | The grid columns are `minmax(0, 1fr)`, and Calculations show their band after the value (`showBands`). |
| MT-47 | Skip rules moved to another page | Each rule records its `page` and stays there. The header flags a rule that tests a later page. The "Then" label now says when the rule acts. |
| MT-50 | CSV import counted blank cells, and they could wipe later translations | Blank cells are ignored. |
| MT-35 | Wrong example in a Number disclosure | The example is chosen by question type. |

Found while fixing these:
- Disclosures on date lists, Medication and Admissions never fired, because matrices were treated as choice questions.
- `{answer}` didn't format month-and-year dates or lists of dates.

All are covered by unit tests, and checked in a browser by `spikes/ui-fixes.ts`.
