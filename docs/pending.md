# Pending items

State on 1 Oct 2026. Plans: [plan-redesign.md](plan-redesign.md), [plan-features.md](plan-features.md) and, next, [plan-workflow.md](plan-workflow.md) (the episode workflow, which comes before the items below). Requirement IDs refer to [requirements.md](requirements.md).

## 1. Blocking now

| Item | Why it matters | Next step |
| --- | --- | --- |
| Docker Desktop doesn't start its engine, and `docker` commands hang | Postgres is down, so the app can't run | Open Docker Desktop and answer whatever it's waiting for (update, sign-in or licence) |
| Migration 00003 (`option_lists`) is not applied | Saved option lists fail until it is | `make migrate` |
| The last four features haven't been tried in a browser: nested logic, Question Set conditions, Move into group, saved option lists | Only unit and handler tests cover them so far | `make dev`, then run `spikes/ui-features3.ts`, then [manual-test-plan.md](manual-test-plan.md) |
| Nothing is committed (redesign, phase 1, phase 2 and the follow-ups) | A large working tree is at risk | Commit when the checks above pass |

## 2. Waiting on input

| Item | Detail |
| --- | --- |
| Real HJE Full HQ export | Today's copy has only Question Sets 1–4, transcribed from screenshots, with no disclosures. Run `tools/export-lifebox-hq.js` in the Lifebox Author tool and import the file with `go run ./cmd/import-lifebox` to get all 11 sets and their disclosure codes (EXP-04). |
| Saved option lists: copied or linked? | Built as copies: a question keeps its options when the list changes, which protects stable IDs (OPT-02). Confirm that's what's wanted. |

## 3. Small gaps in what's built

- **Disclosures on "Other (please specify)":** not possible yet.
- **Question Set conditions:**
  - The one-set preview can't see answers from earlier sets. It explains the condition but decides it only when it tests just the patient.
  - There is no whole-questionnaire preview yet.
- **Questionnaire list:** no status filter or sorting (FRM-02). Search and paging work.
- **Lifebox import:**
  - Conditions that cross Question Sets on pages and questions are still dropped with a warning.
  - Lifebox has no Question Set-level conditions to import.

## 4. Not started: Must

| Area | Requirements | Notes |
| --- | --- | --- |
| Versions | LCY-01, 02, 06, 07 | Draft → In review → Approved → Published → Retired. Published versions are read-only, with "Create new version" that keeps stable IDs. |
| Review, sign-off, publish | SGN-01, 02, 04, 05 | The approver must not be an editor of that version. Publishing must be blocked by logic problems and failing test cases. Both checks already exist in the editor and preview. |
| Duplicate questionnaire | FRM-05 | |
| Code library screen | COD-01, 02, 04, 05 | Today codes can only be picked inside disclosures. |
| Export and import between installations | EXP-01, 02, 03 | One file with all codes, and promotion from QA to Training to Production. |
| Audit trail | AUD-01 | |
| Accessibility check | PX-02 (WCAG 2.2 AA) | SurveyJS mostly covers it, but nobody has verified it. |
| Answers recorded against a version | OUT-03 | Needed once real patients fill in forms. |

## 5. Not started: Should

- **Outputs:**
  - an all-outputs screen (CLN-13)
  - outputs from combinations of answers (CLN-04)
  - outputs limited by a condition on the answer (CLN-03)
- **Panels:** Allergies (PNL-05), and library items: saved groups (PNL-06).
- **Versions:** compare (LCY-08) and retire (LCY-09).
- **Reviews and reports:**
  - reviewer comments on items (SGN-03)
  - reports: outputs across questionnaires, where a code is used, and options with no output (RPT-01/02/03)
- **Hospitals:** master templates and adopting their updates (TEN-03/04), and per-hospital settings (TEN-05).
- **Respondent experience:**
  - print or PDF with outputs (EXP-06)
  - the review view (VEW-03)
  - resuming later (PX-05)
  - pre-filling answers (PX-06)
  - hospital branding (PX-07)
- **Validated scoring tools** (PNL-07, Could): STOP-BANG, PHQ-9 and others, with locked wording.
