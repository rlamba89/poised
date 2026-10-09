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
- **Major: a stale tab's Validate silently overwrites newer clinician answers** (found in S02 testing, 8 Oct; older than S02; [test-report-2.md](sessions/S02-workflow-route-tests/test-report-2.md) X-1).
  - Two tabs (or two clinicians) open the same Question Set. Tab A changes an answer, and it autosaves. Tab B, still showing the old answer, clicks **Validate**.
  - Tab B's save is accepted and stamped "Validated", tab A's change is lost, and nobody is told. The POA Summary shows tab B's answer.
  - **Cause:** `PUT …/episodes/{eid}/answers/{cid}` has no stale-revision check, unlike chapter saves (409).
  - **Fix:** a revision check on answer saves (409, nothing written), decided by Rahul on 8 Oct. Scheduled as session **S02a** ([brief](sessions/S02a-answer-revisions/brief.md)), after S02 and before S03.
- **An episode PATCH can put back a status that just changed** (found by S02's verification, 8 Oct; older than S02).
  - `updateEpisode` writes the status it loaded, even when only details change (`UPDATE episodes SET status = $2 … WHERE id = $1`).
  - If Complete HQ review or the patient's submit lands in the milliseconds between the load and the write, the old status comes back, while `review_completed_at` or `patient_submitted_at` stays set.
  - **Fix with the lost update above:** add `AND status = <loaded status>` to the UPDATE, and answer 409 when no row matches. Details in [verify.md](sessions/S02-workflow-route-tests/verify.md) item 2.
- **Before C1: after a panic, the recover middleware sends headers the handler had already set** (found by S02's verification, 8 Oct).
  - `recoverPanics` writes the 500 without clearing `w.Header()`, so a cookie set before a panic would go out with the error.
  - No handler can do that today: both cookie handlers write straight after `SetCookie`. Fix it test-first before the Cognito session code (C1). Details in [verify.md](sessions/S02-workflow-route-tests/verify.md) item 3.
- **A test gap: `TestUpdateEpisode` doesn't check every column after its refusals** (found by S02's second verification, 8 Oct).
  - It compares only `Status` and `NurseAsa` with the row before. A refused null procedure, anaesthetic or consultant, or a bad `anaesthetistAsa`, isn't proved to leave the row unchanged.
  - No bug today. Compare the whole row the next time this test is touched (S02a is likely to). Details in [verify.md](sessions/S02-workflow-route-tests/verify.md) item 4.
- **Complete review isn't checked on the server** (found by S02, 8 Oct). `POST …/complete-review` succeeds while the episode is Ready for review, even if no Question Set is validated. Only the browser disables the button until every visible set is validated. Enforce it on the server in C6, where the `evaluate` Lambda can work out which sets are visible.

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
- **Found by the S02 manual test, 8 Oct** (all older than S02; details in [test-report-1.md](sessions/S02-workflow-route-tests/test-report-1.md)):
  - **A double-click on "Save comment" saves the note twice.** Notes are append-only, so the duplicate can't be removed, and it shows on the POA Summary. The button should be disabled while it saves.
  - **The patient link's response (`GET /api/p/{token}`) includes the patient's last name, date of birth and sex,** though the page shows only the first name. **Decide before C2 (S14):** if patients sign in with magic link + date of birth, this response must not include the date of birth, or anyone holding the link can read the answer to the check.
  - **Sign out sometimes seemed not to sign out** (2 of about 10, on refusal pages reached by pasting an address). Unconfirmed; it may have been the testing tool's timing.
  - **Episode refusal pages:** the hospital picker is blank on "You don't have access to this hospital." (The questionnaire refusal pages have no header at all; see S01's item above.)
  - **"No episodes yet." shows when a status filter matches nothing,** though episodes exist.
  - **A PATCH with an unknown episode status gets 409** ("can't be changed to that now"), where 400 would be more accurate.
  - **The ASA grade controls work before the HQ review is complete,** while Episode status is locked. Check whether that's wanted.
  - **Small:** the patient's set page says "Saved" before any answer; the delete dialog's text flickers to `""` as it closes.
  - **From the retest (report 2):**
    - **A patient's section counts as complete once every required question is answered,** even without "Finish this section". Probably by design; confirm.
    - **"Send your answers?" can reappear over a section** if a click lands behind it as it animates in. Confirming it still submits correctly.
    - **The staff header wraps at narrow widths** (about 500 px), crowding the back link. Staff pages aren't phone-targeted.
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
