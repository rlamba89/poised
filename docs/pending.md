# Pending items

State on 1 Oct 2026. Plans: [plan-redesign.md](plan-redesign.md), [plan-features.md](plan-features.md) and, next, [plan-workflow.md](plan-workflow.md) (the episode workflow, which comes before the items below). Requirement IDs refer to [requirements.md](requirements.md).

## 1. Blocking now

Nothing. The episode workflow ([plan-workflow.md](plan-workflow.md)) is built and committed.

**After a full disk:** if Postgres won't start and logs `bogus data in lock file`, remove the stale lock and recreate the container. The data is safe in its volume:

```sh
docker run --rm -v sj-demo_dbdata:/data alpine rm /data/postmaster.pid
docker compose up -d --force-recreate db
```

## 1a. Episode workflow: next items

- **POA Summary sections not built yet:**
  - Audit log and Work plan
  - comments per section
  - Observations, Investigation results, Assessments
  - Files, and Send to EPR
  - hospital-reported BMI
  - a suggested ASA from disclosures
- **Statuses:** admitted, discharged, recovery and archive are not built yet.
- **Patients:**
  - accounts, and invitations by email or SMS
  - patient search
  - Short HQ
- **Re-opening:** a review can't be re-opened after it is complete.

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
- **Found by the S01 manual test, 8 Oct** (all older than S01; details in [test-report-1.md](sessions/S01-integration-harness/test-report-1.md)):
  - **The delete dialog shows the previous error.** After a refused delete ("Only drafts can be deleted…"), opening Delete on another questionnaire still shows that message. The delete itself works.
  - **The list API's `total` is 0 on a page past the last one,** and every item carries a stray `total`. Both come from `count(*) OVER ()` in `ListQuestionnaires`.
  - **The access-error pages are dead ends.** "You don't have access to this hospital." and "Questionnaire not found." have no header, no Sign out and no way back.
  - **"chapter" shows in user-facing text:** the two-editor conflict message says "Someone else changed this chapter…" under a "Question Set" title.
  - **Small:**
    - Enter doesn't submit "Save these options as a list".
    - A saved list is deleted with one click, with no confirmation.
    - The viewer's ⋯ → View uses the pencil icon.
    - The State badge truncates to "D…" below about 760 px.
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
