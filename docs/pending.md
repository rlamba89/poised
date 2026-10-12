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
- **Two simultaneous episode PATCHes can overwrite each other's ASA grade** (found by S02a's verification, 9 Oct; older than S02a).
  - S02a's guard covers only the status. The handler still fills the fields a PATCH didn't send from the row it loaded, so of two PATCHes in the same milliseconds (Nurse ASA and Anaesthetist ASA), the last empties the other's grade.
  - **Fix:** an UPDATE that sets only the fields sent (`sqlc.narg` with `coalesce`), or a revision on the episode row. Details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md) item 2.
  - **Related (S02a verification 2, finding 5):** the PATCH 409 says "This episode changed since you opened it", but it compares with the status the request loaded a moment earlier, not the one the user's page loaded. A stale POA Summary page isn't detected. A revision on the episode row would fix both, and make the message true.
- **A save whose response is lost can make the tab conflict with itself** (found by S02a's verification, 9 Oct).
  - **How it happens:** the server commits an answers save, but the browser gets a network error. The saver keeps the old revision, so the tab's next save gets a 409, and the user sees "changed somewhere else" though no other tab exists.
  - **Not silent:** the message and Reload show, and Reload shows what was saved.
  - **Fix when it matters:** after a failed save, read the set's current revision and data, and carry on if the server already has this tab's answers. Details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md) item 3.
- **An answers save's 409 can give the wrong reason** (found by S02a's verification, 9 Oct; low).
  - **How it happens:** if the patient submits on another device, or another clinician completes the review, while a save is in flight, that save says "changed somewhere else" instead of "already sent" or "review complete".
  - **Why it's low:** it's still a 409, and Reload shows the truth.
  - **Where:** `patient.go` and `validation.go` pick the message from the episode as loaded at the start of the request.
- **Before C1: after a panic, the recover middleware sends headers the handler had already set** (found by S02's verification, 8 Oct).
  - `recoverPanics` writes the 500 without clearing `w.Header()`, so a cookie set before a panic would go out with the error.
  - No handler can do that today: both cookie handlers write straight after `SetCookie`. Fix it test-first before the Cognito session code (C1). Details in [verify.md](sessions/S02-workflow-route-tests/verify.md) item 3.
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
- **Found by S02a's second verification, 11 Oct** (details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md), "Verification 2"):
  - **Network errors show technical text** (finding 2, older than S02a): a failed fetch shows "TypeError: Failed to fetch" (patient banner) or "Failed to fetch" (staff). `apps/web/src/lib/api.ts` could turn a network failure into a plain message, such as "We couldn't reach the server. Check your connection." The patient page already says **Not saved** next to the section.
  - **If Reload fails on the clinician's set page, the page is stuck on the error** (finding 3, new in S02a): `ValidateSet.tsx` doesn't wait for or catch the reload, and the error from `useEpisode` / `useEpisodeHQ` is never cleared. Refreshing the browser recovers.
- **Found by the S02a manual test, 9 Oct** (older than S02a; [test-report-1.md](sessions/S02a-answer-revisions/test-report-1.md) item 3):
  - **The editor says "All changes saved" while a title edit is still in the focused field.** It saves only when the field loses focus. So a stale editor tab finds out about a conflict only then, and an edit can look saved when it isn't.
- **Found by the S03 manual test, 12 Oct** (all older than S03; details in [test-report-1.md](sessions/S03-e2e-journeys/test-report-1.md), "Found while exploring"):
  - **A "comes later" logic problem opens an empty-looking condition.** Clicking the problem opens the Logic tab with a blank "Select a question…" picker, because the tested question isn't in the earlier-questions list, while the card still says "Displays when Do you smoke? is Yes". The author isn't shown the cause.
  - **A question in a Clinician Question Set is badged PATIENT** on its card ("Blood pressure" in Nurse checks).
  - **Undo can't be reached while a question is selected:** the Settings panel replaces the Structure panel, where Undo lives.
  - **The New episode "Health questionnaire" dropdown can't be searched.** Each `make e2e` run publishes four more `E2E …` HQs in Hospital A, so the list grows; rebuild the local database when it's cluttered. C2 rebuilds this page.
  - **Clicks straight after a page loads were sometimes ignored** (3 times, in dev mode). Possibly the testing tool's timing; unconfirmed.
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
