# S21 test report 1

**Summary: 26 passed, 0 failed, 0 blocked** (21 plan cases T-01 to T-21, plus R-1 to R-5).

**Environment**
- HEAD `de4da42`, plus the uncommitted S21 work (the working tree isn't clean: S21's new and changed files).
- 21 Oct 2026, 13:22 to 13:52 BST. http://localhost:3000, with the API on :8080 (`make dev`, run by the conductor).
- Chrome, "trk" profile (Claude in Chrome), automatic run.

**How the test data was made**
- Through the UI:
  - T-01's episode, and every task added, completed and re-opened;
  - every status change, Undo, Cancel and settings change;
  - S21 Review's fill-in and validation;
  - the patient journey in R-2.
- About 20 setup episodes (`S21 Stale`, `Anaes`, `Nurse`, `Today`, `Day14`, `Day15`, `Yesterday`, `NoDate`, `Ready`, `Hold`, `Text`, `Off`, `Fresh`, `CancelMail`, `MobOnly`, `Link`, `Atwo`, `AtwoSoon`, `AtwoCancel`) were created with in-page `fetch` calls. Each call sent the same request body that the New episode page sends.
- Five of those episodes were taken to Ready for POA with the same requests the UI makes (patient-form answers, submit, validate, complete-review). So was `S21 Quiet`'s fill-in in T-07.3.
- The action each case checks (the task, status or setting change) was always done in the UI.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Add a task for each team | Pass | "No tasks yet."; "Enter the task." / "Choose the team." are the app's own messages (Enter key); badge, "Added by Cara Clinician on …", "(1 open)"…"(3 open)"; General notes line per team; input stops at 200 (maxLength 200); order kept after reload. |
| T-02 Complete and re-open by different people | Pass | Sue: struck through, Done badge, "Completed by Sue Super …", moved below, "(2 open)", note by Sue. Cara re-open: "Re-opened by Cara Clinician …", back in its added position, "(3 open)". Re-complete: re-open line gone. |
| T-03 Stale tab | Pass | Complete in stale tab → "This episode changed since you opened it…" + Reload; Reload shows it done. Typed title/team kept across the stale message and Reload; Add task then works. Task in tab 1, then On hold in tab 2 → same stale message. |
| T-04 Cancelled episode's tasks | Pass | "This episode is cancelled, so its tasks can't be changed.", task listed, no Add/Complete/Re-open; in neither Open tasks nor Anaesthetic review needed. |
| T-05 Tabs, counts, address | Pass | All seven tabs; All selected with status filter; each tab has a help line (native `title` tooltip); `?worklist=open_tasks`, reload keeps it; Back → All, Forward → Open tasks; direct URL selects it; `?worklist=nonsense` → red "Choose a worklist.", All still works. |
| T-06 Ready for review | Pass | In the list once sent; gone after Complete HQ review. |
| T-07 Not responding | Pass | Jobs "Now + 7 days": `S21 Quiet` got the red badge and is in the tab; `S21 Fresh` (made after) isn't; after the fill-in Quiet leaves Not responding and is in Ready for review. |
| T-08 Anaesthetic review / Open tasks | Pass | Anaes in both; Nurse only in Open tasks; completing removes from both; re-opening puts it back. |
| T-09 Procedure within 14 days | Pass | Today and Day14 in; Day15, Yesterday, NoDate out. `S21 Ready` (+3) in, leaves after Ready for admission. |
| T-10 On hold | Pass | In On hold (count 1); after Not ready for admission the count is 0 and it's gone. |
| T-11 Every count matches its list | Pass | Checked three times (Cara A1, Ada All hospitals, final Cara A1): every tab count = rows, including 0 ("No episodes in this worklist."). |
| T-12 Hospitals | Pass | Cara: no Atwo / AtwoSoon in any tab; Val's URL → "Episode not found." (and "You don't have access to this hospital" on the A2 path). Ada, All hospitals: tab says "Procedure soon", Hospital column, Atwo in Anaesthetic + Open tasks, AtwoSoon in Procedure soon, counts match; tab stays selected switching A1 ↔ All. Bea: no Trust A episodes, and Trust A task/worklist APIs give 404. |
| T-13 A hospital's own days | Pass | A2 set to 1 → tab reads "Procedure within 1 day" and AtwoSoon (2 days) left; A1 still 14; ticked back → 14 and AtwoSoon back. (The A2 check was done as Ada viewing Hospital A2, which uses the same endpoint as Val.) |
| T-14 New settings, off by default | Pass | Worklists (14) and "Messages when the status changes" with the four switch/template pairs, `{{link}}` optional help, switches off; Hospital A1: every "Use the trust's setting" ticked. |
| T-15 Template validation | Pass | Status template without `{{link}}` saved; invitation without `{{link}}` refused ("…must include {{link}}…"); `{{name}}` refused ("{{name}} isn't a placeholder…"). |
| T-16 Ready for admission sends the text | Pass | Note "Ready for admission message sent by text message."; /qa/messages "Hospital A1: Sam, you are ready for your operation (S21 test)." to +447700900211, no email; Undo + pick again sends a second text; Admitted then Undo sends nothing. |
| T-17 Switch off, nothing sent | Pass | `S21 Off` → Ready for admission: no message line in notes, only the invite text on /qa/messages. |
| T-18 Cancelled email; hospital that follows trust | Pass | "Cancelled message sent by email."; email to s21.cancel@example.test, subject "Hospital A1: your health questionnaire was cancelled", default wording; no text. Val cancelling an A2 episode with an email: nothing sent, no note. |
| T-19 Missing contact noted | Pass | "Cancelled message not sent. The patient has no email address."; nothing new for 07700 900213. |
| T-20 `{{link}}` opens sign-in (phone) | Pass | Email "Hello Sam, Hospital A1 cancelled this. Your questionnaires: http://localhost:3000/signin"; the page is the patient's Sign in page, 390 px wide with no sideways scroll, no episode data. See note 1: the window couldn't be resized. |
| T-21 Audit | Pass | "Opened an episode's tasks", "Added a task", "Completed or re-opened a task" by Cara, Sue and Val. |
| Clean-up | Done | A1: every changed setting ticked back to "Use the trust's setting" and saved; reload shows all ticked and the switches off. A2's days were already back to the trust's. |
| R-1 Status control (S20) | Pass | A pick, Admitted with a date, Undo with reasons, Cancel; Status history shows "(undone)" rows and reasons. |
| R-2 Patient journey (S14, S18) | Pass | The `S21 Fresh` invite link → Continue → date of birth → both sections → Send my answers, done in a 390 px frame. Then Ready for review; Cara validated and completed; the POA Summary shows Validated summary and Patient answers. |
| R-3 Invite and Resend (S16, S17) | Pass | New episodes send invites (/qa/messages); Resend invite → "Invite sent again by email." |
| R-4 Edit procedure and dates (S19) | Pass | Planned date saved, note "Planned procedure date set to 2026-10-28", shows in the list and puts the episode into Procedure within 14 days. |
| R-5 All hospitals (S08) | Pass | Ada has Hospital A1 / All hospitals with the status filter under All; Cara has no switch. |

## Failures

None.

## Found while exploring (not failures)

1. **Cosmetic, General notes dates (S19 and S20 lines):** two notes show ISO dates where the page elsewhere uses en-GB dates: "Planned procedure date set to 2026-10-28" and "Status changed to Admitted (admission date 2026-10-21)". This is older behaviour, not new in S21. Mentioned for consistency only.
2. **UX, "← Episodes" goes to All:** opening an episode from a worklist and clicking "← Episodes" lands on **All**, not the worklist you came from. The browser Back button does return to the worklist.
3. **Minor, a cancelled episode's open count:** its Tasks title still reads "(1 open)" when an open task is left. That's accurate, but such tasks can never be done now.
4. **Hover help is a native tooltip:** each tab's hover help is a native `title` attribute. "All" has none. Fine, but slow to show, and screen readers may not announce it.
5. **API and edge checks, all fine:**
   - a title of spaces only gives "Enter the task.";
   - leading and trailing spaces are trimmed;
   - emoji and `<b>` are stored and shown as plain text (escaped);
   - a double-click on Complete sends one PATCH only;
   - a task, then a status change in the same tab (and the reverse) works without a stale message;
   - a task ID from another episode, through this episode's path, gives 404 "Task not found.";
   - a 201-character title or an unknown team gives 400 with clear messages;
   - a POST without `revision` gives the generic 400 "The request could not be read.".
6. **A hospital-scoped clinician on the trust endpoints:** `GET /api/o/{oid}/worklists` and `?worklist=` on the trust list return 200 for Cara (A1 only). The counts are A1's only, so nothing leaks. It matches S08's trust-list scoping.

## Console and network errors

- No console errors or warnings on the episode page, the worklists or Settings.
- Network: the only 4xx/5xx were expected:
  - the 404s and 400s from the probes above;
  - the extension's known false 503s on sign-in, sign-out and some PATCH/POST calls. The app showed success each time, and the rows and notes confirm it.

## Notes for Rahul

1. **The window couldn't be resized.** `resize_window` reported success but the window stayed 1300 px wide (it may be full-screen). So T-20 and R-2 were checked in a 390×844 iframe inside the page (same origin, same cookies). It's a fair check of the layout, but not a real device toolbar.

## GIFs

- `/Users/rahul/Downloads/S21-T01.gif`: the main happy path (adding tasks, the validation messages, the 200-character cap).
- `/Users/rahul/Downloads/S21-T16.gif`: the Ready for admission text, Undo and pick again, Admitted then Undo.
