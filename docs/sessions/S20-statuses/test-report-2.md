# S20 test report 2 (retest)

**22 passed, 0 failed, 0 blocked.** That's 17 retested cases and 5 regressions. F-1 from report 1 is fixed: Enter now sends every reason form. The three changes listed at the top of the test plan all work: the reason boxes are one line with a 1,000-character limit, a stale form closes after Reload when its move is no longer offered, and the patient's cancelled page has Sign out.

Environment:
- HEAD `fb2245b` with the uncommitted S20 working tree (git status not clean, as expected before the verifier commits).
- 2026-10-21, 12:28–12:45 BST.
- http://localhost:3000 (`make dev`, run by the conductor).
- Chrome ("trk" profile) through Claude in Chrome. Automatic run.

Window sizes:
- The first tab was stuck at 500×723 (`resize_window` didn't change it). I used it for the patient (phone-size) pages and as the second, stale tab.
- A second tab opened at 1300×773, and I used it for the staff desktop pages.

Test data (all made up: `Sam S20 Retest …`, born 17/05/1980, Hospital A1):

| Name | Episode ID | Used for |
| --- | --- | --- |
| S20 Retest | 12482ae4-4da4-4991-95f5-ec03303d2b14 | T-01…T-06, T-10, T-11, T-14, T-15, R-1; cancelled from Admitted at the end (Moved to another hospital) |
| S20 Retest Other | 54cfc5b1-a1b5-4035-98c7-a8ecab586262 | T-07, T-08 (Other) |
| S20 Retest Cancel | 6de7acb4-449a-4d8c-93a8-ec0a6851977b | T-08 (Patient withdrew), T-09 |
| S20 Retest Remind | 24cb77a9-9185-45fc-9185-2dca64a1e9bd | R-2, R-3, T-09 step 10 |

R-4 used the patient **S20 Remind** (`s20.remind@example.test`) from report 1.

**Setup done through the API:**
- S20 Retest was made fully in the UI: New episode, then the patient's own link (R-1), then Cara's validation and review.
- To save time, I created the other three episodes with a `fetch` `POST …/episodes` from Cara's signed-in page.
- I also moved S20 Retest from Ready for POA to Ready for admission to Admitted through the API (as Ada), to set up the T-14 check.
- Every step that a case tests was done in the UI.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | New episode: the dropdown shows HQ not complete, disabled, with the hint. Cancel episode is the only button. The history has one row: Created, Cara, —. |
| T-02 | Pass | The patient sent the HQ from their own link, so the history reads Created → "HQ sent … by **Patient**" → "HQ review completed" (Cara). The dropdown offers POA complete, On hold, Not ready and Ready for admission. Cara sees no Re-open and no Undo. |
| T-03 (F-1) | **Pass (fixed)** | The reason box is now one line. **Enter in the reason box sends the form:** "Bloods abnormal" is saved to the history and to General notes ("Status changed to Not ready for admission: Bloods abnormal"). Enter with an empty, optional reason also sends. Back closes the form. An empty reason gives "—" and "Status changed to On hold". Not ready's moves are correct. |
| T-04 | Pass | A future admission date is refused: "The admission date can't be in the future." 10/01/2026 saves (I sent it with Enter in the reason box), and the details show Admission date 10/01/2026. A discharge date of 09/01 gives "can't be before the admission date (2026-01-10)" (sent with Enter in the date field). 12/01 saves. When Discharged, the dropdown is disabled and there's no Cancel. |
| T-05 | Pass | **Undo with Enter:** an empty reason gives "Say why the status change is being undone.", then Enter with "Pressed by mistake" undoes the move. The Discharge date disappears, and the Undo button names the next step back. |
| T-06 | Pass | As Sue, **Re-open with Enter:** an empty reason gives "Say why the HQ review is being re-opened.", then Enter with a reason re-opens. The status goes to Ready for review, Complete HQ review and Edit procedure and dates come back, the sets show Review, and the history and notes are correct. Sue's Complete HQ review gives Ready for POA again. |
| T-07 | Pass | Enter in "Details" with no reason chosen gives "Choose why the episode is cancelled." Other with an empty "Say why" gives "Say why the episode is cancelled." (both sent with Enter). |
| T-08 | Pass | Three reasons were checked:<br>• Other + "Moved abroad" from HQ not complete, sent with **Enter in Say why**;<br>• Patient withdrew;<br>• Moved to another hospital + "Leeds General" from **Admitted** (Ada, with Enter).<br>The grey box, the reason text, the history row and the notes are all correct. On the cancelled POA Summary, ASA Save is refused with "This episode is cancelled, so it can't be changed." |
| T-09 | Pass | Run at 500 px: the patient's link was open when the episode was cancelled.<br>• Reloading signs the patient out.<br>• The invite link then gives "This link doesn't work".<br>• After signing in again with /signin, the home shows **Cancelled by the hospital** with the sentence and no button. `scrollWidth` is 500, so nothing scrolls sideways.<br>• **Step 8 (new):** `/home/episodes/<id>` shows the cancelled message, and the header now has **Sign out**, which signs the patient out.<br>• Step 9: the staff `…/patient-form` page shows the cancelled message.<br>• Step 10: running the jobs at +7 days sent no reminder and set no not-responding for either cancelled episode. |
| T-10 | Pass | • **Step 3:** a stale move gives the exact stale message with Reload. After Reload, the form keeps the choice ("On hold") and the reason ("Stale keep me"), and the dropdown shows the current status (POA complete). Enter then saves it.<br>• **Step 4:** a stale Cancel (Enter in Details) is refused the same way, and the episode is not cancelled.<br>• **Step 5 (new):** tab 1 had "Change the status to Admitted" open while tab 2 undid the episode to Not ready for admission. In tab 1, Change status gives the stale message. After Reload, **the form closes and shows "The episode changed, so that change is no longer offered."**, and the dropdown shows Not ready for admission. GIF saved. |
| T-11 | Pass | Nurse ASA II saved in tab 1. In the stale tab 2, saving Anaesthetist ASA III gives the stale message and keeps III. After Reload, Nurse shows II, and Save keeps both: the API shows `nurseAsa 2`, `anaesthetistAsa 3`. |
| T-12 | Pass | The offered moves match the table at Ready for POA, On hold, Not ready, Ready for admission (which adds Admitted), Admitted (Discharged only) and Discharged (none).<br>Through the API:<br>• `PATCH /status {"to":"bogus"}` gives **400 "Choose a status."** (the fix);<br>• `{"to":"admitted"}` from Ready for POA gives 409 "This episode is Ready for POA, so its status can't be changed to Admitted.";<br>• a 1,001-character reason gives 400 "The reason is too long (1,000 characters at most)." |
| T-13 | Pass | Val (A2) at the A1 episode address sees "You don't have access to this hospital". Through the API, her `GET`, `PATCH /status` and `POST /cancel` all get 404. (Bea wasn't re-run; she passed in report 1.) |
| T-14 | Pass | Cara sees Undo after her own move. Sue sees Re-open at POA complete. On an **Admitted** episode, Ada (an admin) sees Undo and Cancel but **no Re-open**, and the API's `moves` list has no `reopen`. |
| T-15 | Pass | The POA Summary shows the same status control (Re-open and Cancel for Sue), and the episode page history shows every move. |
| T-16 | Pass | Ada's Audit shows "Changed an episode's status", "Cancelled an episode", "Re-opened the HQ review" (Sue) and "Undid a status change", each by the right person. The refused ones show as REFUSED (409) or (400), e.g. "Undid a status change … REFUSED (409)" for the stale undo. |
| Reason length | Pass | Every reason input has `maxlength=1000`. Typing 1,020 characters keeps exactly 1,000. |
| R-1 | Pass | The patient filled in and sent the HQ from their own link. The episode became Ready for review, Cara validated the 3 sets and completed the review, and the POA Summary shows the Validated summary and Patient answers tabs. |
| R-2 | Pass | Resend invite gives "Invite sent again by email.". At +7 days, `/qa/jobs` sent a reminder to S20 Retest Remind (not cancelled) and marked it Not responding. |
| R-3 | Pass | Before the review, Edit procedure and dates saves 01/12/2026, with the note "Planned procedure date set to 2026-12-01". |
| R-4 | Pass | The returning patient's home lists Hospital B1 **In progress**, Hospital A1 **To do** and Hospital A1 **Sent**. |
| R-5 | Pass | Audit has "Opened an episode", "Changed an episode" (the ASA saves) and the status actions. |

## Failures

None.

## Found while exploring

- **A stale Undo whose target changed** (no fault):
  1. In stale tab 2, the Undo form "back to Not ready for admission" was refused as stale.
  2. After Reload, the form **stayed open** with the reason kept, but its title changed to "Undo the last status change: back to **POA complete**". This is because undo is still offered, just for a different move.
  3. Nothing was saved without a second click, and the new title says what will happen, so this is safe.

  A tester could still miss that the target changed. It might be clearer to close this form too, like the "no longer offered" case. This is a suggestion, not a defect.
- **The 1Password extension in trk took over the patient /signin email field** ("Cannot access a chrome-extension:// URL"). I filled it with `form_input` and sent it with the form's submit. This is a tool problem, not an app problem.
- **POA Summary at 500 px** (cosmetic, from an earlier session): when the two ASA grade fields wrap onto two lines, the "Anaesthetist ASA grade" field sits about 20 px further right than "Nurse ASA grade".
- **Double effects:** none seen. Every move wrote exactly one history row.
- **No-access attempts aren't in the audit log:** Val's refused calls through the **A1** address (not her hospital) don't appear in Audit. Calls through her own A2 address do (report 1). This matches decisions.md: `audited` runs inside `requireHospital`, so non-members are refused before the audit. It's by design, and outside S20.

## Console and network errors

- Console: no errors or warnings on the staff or patient pages. I checked after reloading the episode, the POA Summary and the patient's home.
- Network: every 4xx I saw was one a case expected (stale 409s, validation 400s, cross-hospital 404s). There were no 5xx.

## GIFs

- Main happy path (T-03/T-04: On hold → Not ready with Enter → Ready for admission → Admitted → Discharged): `/Users/rahul/Downloads/S20-retest-happy-path.gif`
- Stale tab (T-10, including step 5: the form closes with "no longer offered", and a stale move keeps its reason): `/Users/rahul/Downloads/S20-retest-T10-stale-tab.gif`

## Hand checks for Rahul

- None needed.
