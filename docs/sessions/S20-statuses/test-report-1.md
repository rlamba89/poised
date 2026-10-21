# S20 test report 1

**20 passed, 1 failed, 0 blocked** (16 cases T-01–T-16, 5 regressions R-1–R-5). The one failure is minor: in T-03, pressing Enter in the reason box doesn't submit.

Environment:
- HEAD `fb2245b` with the uncommitted S20 working tree (git status not clean, as expected for a build session);
- 2026-10-21, 11:52–12:19 BST;
- http://localhost:3000 (`make dev`, run by the conductor);
- Chrome ("trk" profile) through Claude in Chrome. Automatic run.

Test data (all made up, all `Sam S20 …`, born 17/05/1980, Hospital A1):

| Name | Episode ID | Used for |
| --- | --- | --- |
| S20 Walk | f34c3368-00c7-4c2f-b630-dd279d33408e | T-01…T-06, T-10, T-11, T-15; cancelled at the end |
| S20 Review | 18d330ad-56c1-4229-958c-539cb28237b8 | T-08 |
| S20 Admit | 10532320-2f5c-4db2-a67b-8180dc74501b | T-08 |
| S20 Cancel | 94ea2899-351e-4147-ab3b-27ec1e78e8f1 | T-07, T-09 |
| S20 Remind | 132d5942-e894-4747-88f9-e5507381f456 | R-1…R-4, T-06 step 7 |
| S20 Roles | 3a9c0d91-3ac2-4877-8ac4-19fabfa4bff7 | T-12, T-13, exploring |

S20 Remind also has a second episode in A1 and one in Trust B (Hospital B1), for R-4.

**Setup done through the API rather than the UI:**
- S20 Walk and S20 Cancel were made fully through the UI.
- To save time, S20 Review, S20 Admit, S20 Roles and the extra R-4 episodes were set up with `fetch` calls from the signed-in staff page. These covered creating the episode, filling in for the patient, validating, completing the review and the first moves to Admitted.
- Every step a case actually tests was done in the UI.

The window couldn't be made narrower than 500 px wide (`resize_window` stops there). So the patient pages ran at 500×667, and the 390 px check was done by loading the pages in a 390×844 iframe and measuring `scrollWidth`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | New episode: the dropdown is disabled on HQ not complete with the hint, only Cancel episode shows, and the history has one row (Created, Cara, —). |
| T-02 | Pass | History reads Created → "HQ sent" (Cara) → "HQ review completed" (Cara). The dropdown offers POA complete, On hold, Not ready, Ready for admission. Cara sees no Re-open and no Undo. |
| T-03 | **Fail (Minor)** | Back, the empty reason, the reason in the history and notes, and Not ready's moves are all correct. But **Enter in the reason box adds a new line and doesn't submit.** Clicking Change status works, and the stray new line is trimmed. |
| T-04 | Pass | Admitted needs a date: an empty or part date gives "Enter the admission date as a date.", and a future date gives "can't be in the future". 10/01/2026 saves. Discharge before admission is refused, with the message showing the date as 2026-01-10. 12/01/2026 saves. When Discharged, the dropdown is disabled and there's no Cancel. A reload keeps everything. |
| T-05 | Pass | An empty or spaces-only reason is refused. Each undo adds an "Undone" row and marks the original "(undone)". The dates disappear as expected, and the button names the next step back. There's no Undo at Ready for POA. Cara undid Sue's move. |
| T-06 | Pass | Cara sees no Re-open. For Sue, the empty reason is refused, and re-opening goes to Ready for review: Complete HQ review and Edit procedure and dates come back, the sets show Review, and the history and notes are right. Changing the BP, validating and completing again gives Ready for POA. The patient's home still says **Sent** after a re-open (step 7, checked on S20 Remind). Ada can re-open too. |
| T-07 | Pass | "Choose why the episode is cancelled." and "Say why the episode is cancelled." both show, and the details label changes to "Say why" for Other. Back closes the form. |
| T-08 | Pass | All 4 reasons work, from HQ not complete, Ready for review, Ready for POA (cancelled from the POA Summary) and Admitted. The grey box, the disabled Cancelled dropdown, the missing buttons, the missing Edit procedure and invite card, the history row and the notes are all correct. The ASA Save is refused with "This episode is cancelled, so it can't be changed.", Add comment works, and the list shows Cancelled. The Status filter offers Admitted, Discharged and Cancelled. |
| T-09 | Pass | Reloading signs the patient out. The invite link then says "This link doesn't work". After signing in again, the home shows **Cancelled by the hospital** with the sentence and no button, with no sideways scroll at 500 px or at 390 px (iframe). `/home/episodes/<id>` and the staff `…/patient-form` both show the cancelled message. Running the jobs at +7 days sent no reminder and set no not-responding for the cancelled episode. |
| T-10 | Pass | A move made in a stale tab gives the exact stale message with Reload, keeping the choice and reason. Reload shows On hold and tab 1's history row. Cancel from a stale tab is refused the same way, and the episode is not cancelled. GIF saved. |
| T-11 | Pass | Nurse ASA II saves in tab 1. Saving Anaesthetist ASA III in stale tab 2 gives the stale message and keeps III. After Reload, Nurse shows II, and Save then keeps both (III and II after a fresh reload). |
| T-12 | Pass | HQ not complete and Ready for review offer only Cancel. Ready for POA, POA complete, On hold, Not ready and Ready for admission each offer the other four reviewed statuses, and Ready for admission adds Admitted. Admitted offers Discharged and Cancel. Discharged and Cancelled offer nothing. The server also refuses moves the table doesn't allow (409, checked with the API). |
| T-13 | Pass | Val (A2) and Bea (Trust B) get "You don't have access to this hospital" in the UI. Through Val's own hospital path the episode is "Episode not found". Through the API (`GET`, `PATCH /status`, `POST cancel/undo/reopen`) both get 404s. |
| T-14 | Pass | Cara sees Undo after her own move. Sue sees Re-open at the reviewed statuses but not on an Admitted episode. A clinician calling `POST /reopen` through the API gets 403 "Only a super clinician or an admin can re-open the HQ review." |
| T-15 | Pass | A reload keeps the status, history and dates. On the POA Summary, a move (POA complete), Undo and Cancel all work, and the episode page history shows them. Browser Back from the POA Summary shows the current status (Cancelled). |
| T-16 | Pass | Audit shows "Changed an episode's status", "Cancelled an episode", "Re-opened the HQ review" and "Undid a status change", each by the right person, including the refused ones (REFUSED 403/404/409). |
| R-1 | Pass | The patient filled in and sent the HQ from their own link (at 500 px). The episode went to Ready for review, with "HQ sent … by **Patient**" in the history. Cara validated the 3 sets and completed the review. The POA Summary shows Validated summary and Patient answers. |
| R-2 | Pass | Resend invite gives "Invite sent again by email." At +7 days, `/qa/jobs` sent a reminder for S20 Remind (not cancelled) and marked it Not responding. |
| R-3 | Pass | Before the review, Edit procedure and dates saves 01/12/2026, with the note "Planned procedure date set to 2026-12-01". |
| R-4 | Pass | The returning patient's home lists Hospital B1 **In progress**, Hospital A1 **To do** and Hospital A1 **Sent**. |
| R-5 | Pass | Audit has "Opened an episode", "Changed an episode" (the ASA saves) and "Added a note". |

## Failures

### F-1 (Minor): Enter in the reason box doesn't submit the status form (T-03 step 4)

- **Steps:**
  1. Sign in as Cara Clinician and open a Ready for POA or On hold episode, e.g. `/o/6ef150fb-16c5-4d4f-a4be-1c2b6f22638b/h/00000000-0000-4000-8000-00000000000a/episodes/f34c3368-00c7-4c2f-b630-dd279d33408e`.
  2. Pick **Not ready for admission** in Episode status.
  3. Type `Bloods abnormal` in **Reason (optional)** and press **Enter**.
- **Expected (test plan):** pressing Enter in the reason box or clicking Change status changes the status.
- **Actual:** nothing is sent. The reason box is a multi-line `Textarea`, so Enter adds a new line: the value became `"Bloods abnormal\n"`. Clicking **Change status** then works, and the server trims the new line.
- **Also affected:** the reason boxes in the Undo and Re-open forms and the Cancel form's "Details"/"Say why" are textareas too, so Enter doesn't submit those either.
- **Note:** `StatusControl.tsx` has the comment "Enter in a field sends the form", which only holds for single-line inputs (e.g. the date field).
- No console errors. GIF: `/Users/rahul/Downloads/S20-T03-enter-in-reason.gif`.

## Found while exploring

- **The stale form stays open for a move that's no longer allowed (Minor, UX).**
  1. Tab 1 opened "Change the status to Admitted" while the episode was Ready for admission.
  2. Tab 2 undid the episode back to POA complete.
  3. In tab 1, Change status gave the stale message, and Reload loaded POA complete.
  4. The form still said "Change the status to **Admitted**", although POA complete doesn't offer Admitted.
  5. Submitting gave a clear server message, "This episode is POA complete, so its status can't be changed to Admitted.", and nothing was saved.

  It's safe, but the form could close or reset after Reload when the chosen move is no longer offered.
- **Double-clicking Change status** sent one PATCH and wrote one history row. Good.
- **Odd text in the reason was handled well:** emoji and `<b>bold</b>` show as plain text, and leading and trailing spaces are trimmed. A 5,000-character reason sent through the API is refused with "The reason is too long (1,000 characters at most)." The UI has no length hint or limit, so a long reason would only fail on submit.
- **Cancelled episodes are locked on the server too** (checked with the API):
  - saving answers or completing the review gives 409 "This episode is cancelled, so it can't be changed.";
  - resending the invite gives 409 "…so no invite can be sent.";
  - undo gives 409 "There's no status change to undo.";
  - re-open, cancel again and status moves all give 409 "cancelled";
  - an unknown cancel reason key gives 400 "Choose why the episode is cancelled.";
  - an unknown status `bogus` gives 409 (not 400), with the text "can't be changed to “bogus”". This is cosmetic.
- **A year typed as `0007` in the admission date** (by mistake) was refused with "Enter the admission date as a date." Good.
- **The cancelled `/home/episodes/<id>` page** has no Sign out or hospital name in its header, unlike the other patient pages. This is cosmetic, and the "← Your questionnaires" link is there.
- **Wording on the staff on-behalf page:** for a cancelled episode, `…/patient-form` says "Please contact the hospital if you have any questions." That sentence is written for patients, but the plan expects the same message. Cosmetic.

## Console and network errors

- No console errors or warnings on the staff or patient pages (checked after reloads).
- The extension's network log showed 503 for some PATCH requests to `/status`. The app showed them as successful, and the history and Audit confirm they were saved, so these are the known false 503s.
- Every other 4xx in the log came from cases that expected a refusal (validation 400s, stale and refused 409s, cross-hospital 404s, the clinician re-open 403).

## GIFs

- Main happy path (T-03/T-04: On hold → … → Discharged): `/Users/rahul/Downloads/S20-T03-T04-happy-path.gif`
- Stale tab (T-10): `/Users/rahul/Downloads/S20-T10-stale-tab.gif`
- Failure F-1 (Enter in the reason box): `/Users/rahul/Downloads/S20-T03-enter-in-reason.gif`

## Hand checks for Rahul

- None needed. Every case ran.
- The only limit was the phone size: the browser window wouldn't go below 500 px. The 390 px layout was checked in an iframe.
