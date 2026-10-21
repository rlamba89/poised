# S22 test report 3 (retest, round 3)

**Summary: 17 passed, 0 failed, 0 blocked.** This round re-ran:
- the case that failed in report 2 (T-18);
- the two cases the plan changed (T-10, T-12);
- the five regressions (R-1 to R-5);
- the main happy paths: T-02, T-11, T-13, and parts of T-01, T-05, T-08, T-14 and T-15.

T-18 was run last.

- **Commit:** 9d37cf7, with the S22 work and the fixes uncommitted in the working tree.
- **Date:** 2026-10-21, 15:57 to 16:13 BST.
- **URL:** http://localhost:3000, with the API on :8080. The conductor started both with `make dev`.
- **Browser:** Chrome with the "trk" profile, driven through Claude in Chrome. For phone size I used a 390px-wide iframe of the app inside the page. The patient pages measured scrollWidth 390, so nothing was wider than the screen.
- **Tester:** the automatic manual-test agent.

**Database state before I started:**
- The earlier future-dated runs had archived every episode in the database (418 archived, none in any other status).
- No hospital had an auto-archive job scheduled in 2027. Hospital A1, A2 and B1 were all due on 2026-12-23 02:00.
- The QA "run as of" pulls every hospital's auto-archive forward to the run's time anyway (`MakeAutoArchivesDue`).
- So the earlier runs did not get in the way of any case. Each case used new episodes with surnames starting `S22 R3`.

**How I set up data:**
- Things under test were done in the UI:
  - Settings;
  - /qa/jobs runs;
  - archiving and un-archiving;
  - status changes, undo, cancel, comments and tasks;
  - the worklists and Show archived;
  - the patient pages: invite, date of birth, filling in, sending, and closing the account;
  - /signin.
- R-5's episode (`S22 R3 Remind`) was made with the New episode form.
- The other setup episodes were made through the app's own API, from a browser tab signed in as the user named in each case: create, fill in for the patient, validate, complete review, then Ready for admission, Admitted and Discharged with today's date. This is the same approach as report 2.
- The T-12 setup episodes were made as Ada Admin rather than Cara or Val. Who creates them doesn't affect the case.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-12 | **Pass** | Hospital A1, "Archive episodes this many days after discharge", with "Use the trust's setting" unticked:<br>• **`7.5`**: the box still says `7.5` after clicking away. Save shows "Archive episodes this many days after discharge: enter a whole number from 1 to 365." Nothing was saved: the database still had `settings = {}`.<br>• **`0`** and **`366`**: each stays as typed and is refused with the same message.<br>• **`7`**: saved ("Hospital A1's settings were saved."), and the database had `{"autoArchive.afterDischargeDays": 7}`.<br>**Runs:**<br>• As of 29/10/2026 16:05 (+8 days): `S22 R3 A1` was archived, "Archived from Discharged on 29/10/2026, 16:05 by Automatic (Auto-archive: 7 days after discharge)". `S22 R3 A2` (Hospital A2) stayed Discharged.<br>• At +31 days: A2 was archived with the rule "Auto-archive: 30 days after discharge". I checked this in the status events table, because Cara can't open A2.<br>• The override was then cleared in the UI by re-ticking the box and saving. Both hospitals are back to `{}`. |
| T-10 | **Pass** | `S22 R3 Ten` was Discharged today at A1, with A1 on the trust's 30 days.<br>• Now + 29 days (19/11/2026 16:02): "0 episodes archived", and it is still Discharged.<br>• Now + 31 days: archived. The box reads "Archived from Discharged on 21/11/2026, 16:02 by Automatic (Auto-archive: 30 days after discharge)", which is the run's date, as the updated plan says.<br>• The last history row is "Archived: Discharged → Archived", by "Automatic (Auto-archive: 30 days after discharge)".<br>• The General note "Episode archived (Auto-archive: 30 days after discharge)" is marked Automatic, on 21/11/2026, 16:02. |
| T-11 | Pass | Sue un-archived T-10's episode.<br>• With no reason it shows the app's own "Say why the episode is being un-archived.".<br>• With the reason `S22 still needed` and Enter, it goes back to Discharged, with the history row "Un-archived: Archived → Discharged" and the note "Episode un-archived, back to Discharged: S22 still needed".<br>• Now + 2 days: still Discharged.<br>• Now + 31 days: archived again, "Auto-archive: 30 days after discharge". |
| T-02 | Pass | Sue archived `S22 R3 Hand` (Discharged), typing `S22 treatment finished` and pressing Enter.<br>• The form has the full explanation and the "Reason (optional)" box.<br>• The box reads "Archived from Discharged on 21/10/2026, 16:05 by Sue Super. Reason: S22 treatment finished. It can't be changed: a super clinician can un-archive it."<br>• The ARCHIVED badge is shown, and the status dropdown shows Archived.<br>• The last history row is "Archived: Discharged → Archived", by Sue Super, with the reason.<br>• The note says "Episode archived: S22 treatment finished".<br>• The only control left is Un-archive: back to Discharged. |
| T-01 (part) | Pass | As Cara, a Ready for POA episode has no Archive episode button. As Sue it does. |
| T-03 / server (part) | Pass | As Cara, on the archived `S22 R3 Hand`, I sent requests to the API directly:<br>• add a note: 409 "This episode is archived, so it can't be changed…";<br>• add a task: 409 with the same message;<br>• un-archive: 403 "Only a super clinician or an admin can un-archive an episode." |
| T-05 (part) | Pass | Show archived adds `?archived=1` and lists the archived S22 R3 episodes with ARCHIVED badges. Unticking it hides them, and Back ticks it again. |
| T-13 | Pass | At phone size, for `S22 R3 Close`:<br>• The **Close your account** card is shown.<br>• Close my account shows the explanation and the orange warning "Your Demo pre-op HQ for Hospital A1 hasn't been sent. If you close your account, it won't be sent.", with the red Close my account and Keep my account.<br>• Keep my account returns to the first view.<br>• I then filled in and sent the HQ (see R-4). After that, Close my account showed **no** warning, as T-17 step 4 expects.<br>• Close shows "Your account is closed…". After a reload it says "You've been signed out". |
| T-14 (steps 4, 6) | Pass | The old invite link, then Continue, shows "This link doesn't work". /signin with the closed email shows "Check your email", but /qa/messages has no new message to that address. |
| T-15 (note) | Pass | The episode's last General note is "The patient closed their account on 2026-10-21.". |
| R-1 | Pass | Ready for POA → On hold. Then "Undo: back to Ready for POA" with a reason: the history shows "(undone)" and "Undone: On hold → Ready for POA". Cancel episode with Patient withdrew gives "Cancelled: Ready for POA → Cancelled". Add comment still works on the cancelled episode. |
| R-2 | Pass | After the Anaesthetist task was added to `S22 R3 Remind` in the UI:<br>• Anaesthetic review needed shows 1, and lists 1 row: that episode.<br>• Procedure within 14 days (planned for tomorrow) shows 1 and lists 1 row.<br>• Open tasks shows 1. |
| R-3 | Pass | Now + 2 days sent "Hospital A1: a reminder about your health questionnaire" to s22r3.remind@example.test (seen on /qa/messages). The General note "Day 2 reminder sent by email." is marked Automatic. |
| R-4 | Pass | At phone size the patient answered About you and Your health and sent them through the app's own "Send your answers?" box. The thank-you page appeared, and the home then shows **SENT** with **View**. |
| R-5 | Pass | In the New episode form, choosing the procedure "E2E Hip 2010 194351 dd57 (E2E-1)" picked its default HQ, "E2E Procedures 2010 194351 87aa v1". The procedure shows on the episode. |
| T-18 | **Pass** (F-3 resolved by the plan change) | **Steps 1–2:** I made `S22 R3 Idle` at 16:08:12; its last note is "Invite sent by email.". I waited past 16:10, then made `S22 R3 Busy` and added the comment `S22 rang the patient` at 16:10:45.<br>**Steps 3–4:** run as of **21/04/2027 16:09**. The result was "1 reminder sent, 1 marked "not responding", 3 episodes archived".<br>• Idle was archived: "Archived from HQ not complete on 21/04/2027, 16:09 by Automatic (Auto-archive: 6 months without activity)". It got no reminder and no Not responding note.<br>• Busy is still HQ not complete. It gained "Day 5 reminder sent by email." and "Not responding: …7 days after the invite.", both Automatic on 21/04/2027, 16:09, as the plan allows.<br>**Step 5:** run as of **22/10/2027 16:15** (12 months and a day). The result was "2 episodes archived". Busy was archived: "Archived from HQ not complete on 22/10/2027, 16:15 by Automatic (Auto-archive: 6 months without activity)", with the matching history row and an Automatic note.<br>Optional step 6 was not re-run; it passed in report 2. |

## Failures in detail

None.

## Found while exploring

- **The General notes are out of order after a future-dated run and an un-archive.** Cosmetic, QA-only.
  - On `S22 R3 Ten`, the +31-day run's "Episode archived (Auto-archive…)" note is dated 21/11/2026.
  - The un-archive that followed it in real time is dated 21/10/2026.
  - So the notes list shows the un-archive above the archive. The status history, which is in sequence order, shows them the other way round.
  - This comes from job runs being dated with their "as of" time (the F-2 fix). It can't happen with the real nightly run.
- **An archived episode still shows the red NOT RESPONDING badge** next to ARCHIVED (`S22 R3 Busy`). This is harmless and may be intended, since the flag is history. It's for Rahul to judge whether the badge should be hidden once an episode is archived.
- **A run "as of" a later time sends only the latest reminder that's due.** `S22 R3 Busy` got the day-5 reminder only, not day 2 as well. That matches report 2 and looks intended.
- **The cross-hospital guard holds.** As Cara, GET on the Hospital A2 episode returns 404.
- **Show archived and Back:** unticking and pressing Back restores `?archived=1` with the box ticked.

## Console and network errors

- No console errors or warnings on the pages I tracked: the episodes list and the episode pages.
- All API requests on the reloaded episodes list returned 200.
- The only 4xx and 5xx responses were from my deliberate probes:
  - the 409 and 403 on the archived episode;
  - the 404 for the A2 episode as Cara;
  - two 400 "The request could not be read." responses, caused by my own probe bodies, which left out `revision`.

## GIFs

All are in `/Users/rahul/Downloads/`:
- `S22-R3-T12.gif`: 7.5 kept and refused, then 0 and 366 refused, then 7 saved.
- `S22-R3-T10.gif`: the +29 and +31 day runs and the archived box dated with the run's time.
- `S22-R3-T02.gif`: happy path, Sue archives with a reason using Enter.
- `S22-R3-T13.gif`: the patient at phone size: the warning, Keep, filling in and sending, the no-warning close, and the signed-out reload.
- `S22-R3-T18.gif`: the 6-month and 12-months-and-a-day runs, and Busy archived.

## Data left behind

- Made-up patients with surnames starting `S22 R3` at Hospital A1: Probe, A1, Ten, Hand, Undo, Remind, Close, Idle and Busy. There is also `S22 R3 A2` at Hospital A2.
- The /qa/jobs runs went as far as 22/10/2027, the same as round 2.
- Hospital A1's and Hospital A2's settings are back to the trust's (`{}`).

STATUS: ALL_PASSED
