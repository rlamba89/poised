# S43 test report 2 (retest)

**14 passed, 0 failed, 0 blocked** (re-run: T-07, which failed in report 1; T-05's focus check; T-06's corrected wording; the happy paths T-02, T-03, T-04, T-08, T-09, T-11; and every regression case, R-01 to R-05).

- **Environment:** commit `f769916`, with S43's changes still uncommitted (41 changed and new files). Tested 27 Oct 2026, 04:10–04:29 GMT, at http://localhost:3000, on the conductor's `make dev` (API :8080, evaluate :8090). Chrome through Claude in Chrome.
- **Not re-run:** T-01, T-10 and T-12 passed in report 1, and the builder's changes don't touch them. I didn't stop the evaluate service. One listener is still on 127.0.0.1:8090 (node PID 77145, the conductor's copy). I started no copy of my own.
- **Phone size:** `resize_window` again reported success, but the viewport stayed 800 px wide. As in report 1, I ran the patient journeys inside a 390×650 iframe of the real invite link, on the same origin. `scrollWidth` equalled `clientWidth` (388), so there was no sideways scroll.
- **Captured messages:** I read the invite links from `GET /api/qa/messages`, the API behind `/qa/messages`.
- **Test data (all made up):**
  - HQ "MT S43 0336": I published version 3 from v2, with Low risk set back to auto-triage, for T-09.
  - New HQ "MT S43 R2 R04 0425" (R-04).
  - Patients "Sam MT43 R2 0412 High" (T-02 to T-05, T-08), "… 0415 Tabs" (T-06, T-07), "… 0420 Arch" (T-11, R-01, R-05; sent through **Fill in for the patient**) and "… 0427 Low" (T-09, R-02, R-03). All were born 17/05/1980, with emails `mt43.r2.*@example.org`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-07 Second tab's Clear (F-1 retest) | **Pass** | Tab 2 shows the stale message, with Reload, and keeps "Tab two reason". After **Reload** the flag shows grey, "Cleared by Cara Clinician on 27/10/2026, 04:17: Tab one reason". There is no Clear button and the form has closed (no text box left on the page). The General notes have only one "Flag cleared" line. |
| T-05 Clear the flag (focus fix) | Pass | When the Clear form opens, the reason box has focus: typing "R2 typed straight in" went into it. **Back** closes the form without clearing. With an empty reason, "Say why the flag is cleared." "Reviewed by Dr Example" + Enter clears the flag: the badge goes, and the card reads "Cleared by Cara Clinician on 27/10/2026, 04:13: Reviewed by Dr Example", with no Clear button. The note by Cara is there. All of this is kept after a reload. The list row has no badge, and the episode stays on Anaesthetic review needed while its task is open. |
| T-06 Other hospital or trust (corrected wording) | Pass | With the copied A1 URL, Val and Bea each see "You don't have access to this hospital", with no patient details or flag. With their own hospital in the URL, each sees "Episode not found." The episode isn't in Val's (A2) list or Bea's (B1) list. As Bea, a clear POST with A1's ids gets 404 "Hospital not found." and with B1's ids 404 "Episode not found.". Back as Cara, the flag is still standing. |
| T-02 High-risk patient sends on a phone | Pass | The patient pages fit 390 px and say nothing about rules, tasks or flags. The episode page shows the red badge, and the Flags card reads "Added by the rule “High risk” on 27/10/2026, 04:12" with a Clear button. There's no Auto-triaged card. The task reads "Book anaesthetic review", Anaesthetist, "Added by rule: High risk on …", with Complete. The red requirement line is word for word. All three Automatic notes are there. The episode is on Anaesthetic review needed (86) as Ready for review. |
| T-03 Ready for admission refused | Pass | After the review (→ Ready for POA), the dropdown greys out "Ready for admission (add the anaesthetist's ASA grade first)". Clicking it does nothing. The red line stays, and the POA Summary shows its own red line. |
| T-04 Anaesthetist's grade meets it | Pass | After saving ASA III, the POA line turns grey: "The anaesthetist's ASA grade is added, as the rule “High risk” asks." On browser Back the red line has gone, and Ready for admission is offered normally. The move works and is kept after a reload. |
| T-08 Completing the rule's task | Pass | The task shows Done, "Completed by Cara Clinician on 27/10/2026, 04:14", and still "Added by rule: High risk". The episode has left Anaesthetic review needed. The status and the cleared flag are unchanged. |
| T-09 Low-risk patient (v3) | Pass | The card reads "Auto-triaged by the rule “Low risk”", the episode is Ready for admission, and it's on the Auto-triaged tab. There is no flag badge, Flags card, rule task or red line, and no "Task added…" or "Flag added…" note. |
| T-11 Archived episode's flag | Pass | Sue archived the episode. The badge and the Flags card still show, with no Clear button. |
| R-01 Task by hand | Pass | "MT43 R2 call patient", Nurse team: "Added by Cara Clinician on …". I completed it, then re-opened it: "Re-opened by Cara Clinician on …". |
| R-02 No rule requirement → Ready for admission | Pass | Sue re-opened the T-09 episode, validated both sets and completed the review (→ Ready for POA). She then moved it to Ready for admission; nothing was greyed out. The status history is right. |
| R-03 Checked | Pass | "Checked by Cara Clinician on 27/10/2026, 04:25". The episode left the Auto-triaged tab. |
| R-04 Rules page of a new HQ | Pass | It shows the two default rules. Renaming High risk to "High risk R2" saved and was kept after a reload. |
| R-05 Show archived | Pass | The T-11 episode isn't listed without Show archived. With it, the episode is listed as Archived, with its red Anaesthetic review badge. |

## Failures in detail

None.

## Found while exploring

1. **A stale Clear while the flag is still standing keeps the form open, which is right.** Tab 2 opened Clear and typed "Stale standing reason". Tab 1 then added a task, which changes the episode. Tab 2's Enter gave the stale message. After Reload, the form stayed open with the reason kept and a live Clear flag button, because the flag is still standing. The new task showed in tab 2. I didn't send that clear.
2. **A stale Clear on an episode that was archived meanwhile.** Tab 2 was still showing the episode as not archived. Its Clear gave the stale message, and nothing was cleared. After Reload the page shows the episode as archived, the form has closed, and there's no Clear button.
3. **Staff "Fill in for the patient" with heart failure Yes** added the task, the flag and the requirement, just as a patient send does. The episode page says "Sent by Cara Clinician on the patient's behalf".
4. **Cosmetic, seen again (report 1, note 4; not this session's):** the General notes say "Anaesthetist ASA grade set to 3", while the page says "ASA III" everywhere else.
5. **Small observation:** in the episode list, an archived episode's row doesn't show "1 open task" under its status, even though its tasks are still open. Other rows do. This is probably intended, since an archived episode leaves every worklist. I'm noting it only in case it isn't.

## Console and network errors not tied to a case

- None. No console errors or warnings. Every `/api/` request on the episode page returned 200; each GET ran twice, which is the dev StrictMode pattern.

## GIFs

- `~/Downloads/S43-retest-happy-path.gif`: the high-risk episode page, the Clear form's focus and Back, the empty-reason message, clearing with Enter, the reload, the worklist and completing the task.
- `~/Downloads/S43-T07-retest.gif`: F-1's retest, the two-tab stale Clear, then Reload closing the form.

## Evaluate service

- I didn't stop it. One listener is on 127.0.0.1:8090: node PID 77145, the conductor's copy. No copy was started separately in this round.

## Hand checks for Rahul

- None needed. (Optional: the patient flow in a real 390×844 window, since the window couldn't be resized here. Layout at 390 px CSS width was fine.)
