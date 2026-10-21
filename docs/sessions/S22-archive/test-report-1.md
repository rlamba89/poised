# S22 test report 1

**Summary: 21 passed, 2 failed, 0 blocked** (23 cases: T-01 to T-18 and R-1 to R-5; T-18's optional step 6 not run).

- Commit: 9d37cf7, with the S22 changes uncommitted in the working tree (56 changed or new files)
- Date: 2026-10-21, 14:42 to 15:17 BST
- URL: http://localhost:3000 (API on :8080), started by the conductor (`make dev`)
- Browser: Chrome ("trk" profile) through Claude in Chrome. The window was mostly 1400×773. For phone size I used a 390px-wide frame inside the page, because the window would not resize reliably.
- Tester: automatic manual-test agent

**How I set up data.** I made the first episode (`S22 Perm`) with the New episode form, and did one HQ review fully by hand (T-08 step 5: Validate ×3, then Complete HQ review). After that, I made most setup episodes with the app's own API, from a browser tab signed in as the same user: create the episode, fill in for the patient, validate, complete the review, and change the status. I did all the behaviour under test in the UI: archive and un-archive, the lists and worklists, the patient pages, Settings, /qa/jobs and closing an account. R-4 and the patient journeys in T-09, T-13, T-14 and T-16 were done by clicking at phone size. To switch staff user, I signed out and signed in with the dev login.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Cara has no Archive button. Sue and Alex see it next to the Episode status. |
| T-02 | Pass | Enter submits the form. The grey box, ARCHIVED badge, history row and General note all match the plan. With no reason, the box has no "Reason:" and the note is "Episode archived". The reason box stops at 1,000 characters (maxLength, and also when pasted). |
| T-03 | Pass | Status dropdown is disabled, and the only button is "Un-archive: back to Discharged". There is no Edit procedure, Add comment or Add task, and the tasks message is shown. On the POA Summary the ASA grades and Save are disabled. The View page is read-only with the archived message. The invite card is hidden. Cara gets the same page, without Un-archive. |
| T-04 | Pass | In the stale tab, Add comment is refused with the exact message and nothing is saved. On hold gives "This episode changed since you opened it…" with a Reload button. Reload shows the archived page and "The episode changed, so that change is no longer offered." |
| T-05 | Pass | Anaesthetic 3→2, Procedure ≤14d 25→24, On hold 2→1, Open tasks 5→4. The episode is gone from every tab and from All. Show archived sets `archived=1`, and reload, untick and Back all behave as the plan says. The Archived status filter lists only archived episodes. On Alex's All hospitals view the episode is hidden, and shown with Show archived. |
| T-06 | Pass | Search by surname finds the patient. The patient page shows the ARCHIVED badge and opens the episode. A deep link in a new tab opens it. |
| T-07 | Pass | Shows "Archived from Cancelled", and the history row "Archived: Cancelled → Archived". Un-archive goes back to Cancelled, and the cancelled box still says "Reason: Patient withdrew". |
| T-08 | Pass | Cara has no Un-archive. An empty reason, or spaces only, gives the app's "Say why the episode is being un-archived.". With a reason the episode goes back to Discharged with the same dates; the history, the note, Add comment and Add task are all back. On hold is restored, with all four worklist counts +1. Ready for review restored, then Validate ×3 → Complete HQ review → Ready for POA. |
| T-09 | Pass | At phone size the closed message shows, the home card says CLOSED BY THE HOSPITAL, and nothing is wider than 390px. After un-archive the card is TO DO with Start, and the questionnaire opens. |
| T-10 | Pass | At +29 days, 0 archived and still Discharged. At +31 days it is archived "by Automatic (Auto-archive: 30 days after discharge)", with the history and the Automatic note. |
| T-11 | Pass | After un-archive, +2 days leaves it Discharged, and +31 days archives it again by the same rule. |
| T-12 | **Fail** (step 1) | 0 and 366 are not refused: they are silently clamped to 1 and 365 and saved. See F-1. Steps 2–5 pass: A1 archives at 7 days and A2 at 30 days, and the override is cleared afterwards. |
| T-13 | Pass | At phone size: the card, the orange warning ("Your Demo pre-op HQ for Hospital A1 hasn't been sent…"), red Close and Keep buttons, nothing wider than the screen. Keep goes back. Closing shows "Your account is closed…", and reload shows "You've been signed out". |
| T-14 | Pass | The old invite link gives "This link doesn't work". The unused sign-in link gives "This link has expired… or has already been used". A new /signin request shows "Check your email" and no new message is captured. |
| T-15 | Pass | The General note "The patient closed their account on 2026-10-21." is there. The audit log has "Patient closed their account" for Sam S22 Close. |
| T-16 | Pass | Send invite gives "Invite sent again by email.". The new link opens the questionnaire, and the new home has Close your account. |
| T-17 | Pass | The second patient was in a separate cookie jar (`http://[::1]:3000`, a stand-in for a private window). They stayed signed in after another patient closed their account. With everything sent, Close shows no orange warning, and Keep goes back. |
| R-1 | Pass | On hold, then Undo back to Ready for POA with a reason, then Cancel (Patient withdrew). Add comment works on the cancelled episode, including with an emoji. |
| R-2 | Pass | Every tab's count equalled its rows (checked all 6 tabs, twice). An Anaesthetist task added in the UI puts the episode in Anaesthetic review needed (3→4). |
| R-3 | Pass | Now + 2 days sends "Hospital A1: a reminder about your health questionnaire" to s22.remind@example.test, with the note "Day 2 reminder sent by email." (Automatic). |
| R-4 | Pass | At phone size the patient fills in both sections and sends them, and the home shows SENT with View. |
| R-5 | Pass | Picking procedure "E2E Hip 2010 194351 dd57" chose its default HQ "E2E Procedures 2010 194351 87aa v1", and the procedure shows on the episode. |
| T-18 | **Fail** (step 4 as written) | `S22 Idle` (with an email, as in the recipe) was not archived at 6 months. The same run first sent its overdue reminder and "not responding" note, stamped with the real time, and that counted as fresh activity. See F-2. With patients who have no email, the rule works exactly as the plan expects. Step 5 passes. Step 6 (optional) was not run. |

## Failures in detail

### F-2 (T-18): an idle pre-op episode with an email isn't archived by the "6 months" run. **Minor** (mostly a /qa/jobs simulation problem; see the note at the end)

Signed in as Cara at Hospital A1.

**Steps:**
1. Make `Sam S22 Idle` with the email `s22.idle@example.test`. Its last General note was at 15:06.
2. At 15:10, make `Sam S22 Busy` (`s22.busy@example.test`) and add the comment `S22 rang the patient` (15:10).
3. On /qa/jobs, set Run as of to **21/04/2027 15:08** and click Run jobs.

**Result line:** "Ran 298 jobs as of 21/04/2027, 15:08: 2 reminders sent, 2 marked "not responding", 394 episodes archived."

**Expected:** `S22 Idle` is Archived ("Automatic (Auto-archive: 6 months without activity)"), and `S22 Busy` stays HQ not complete.

**Actual:** `S22 Idle` is still **HQ not complete**. Its General notes gained two lines, both dated **21/10/2026, 15:10** (the real time of the run):
- "Day 5 reminder sent by email." (Automatic)
- "Not responding: the HQ hasn't been sent 7 days after the invite." (Automatic)

**Cause, as far as I can tell:**
- The run sends the overdue reminder and "not responding" mark before the auto-archive.
- Those notes are stamped with the real clock, not the run's "as of" time.
- The 00023 trigger sets `episodes.last_activity_at = now()` on every note, so the episode looks active at 15:10:31, which is after the cutoff (as of − 6 months = 21/10/2026 15:08).

**Confirmation:**
- I repeated the case with patients who have no email: `S22 Idle NoEmail` at 15:10:50, then `S22 Busy NoEmail` with a comment at 15:13, run as of 21/04/2027 15:12.
- The Idle one was archived "by Automatic (Auto-archive: 6 months without activity)", with the matching history row and note. The Busy one stayed HQ not complete.
- Running as of 22/04/2027 15:16 (6 months and a day) archived the Busy one too.
- So the rule itself works.

**Note for the builder:**
- In real nightly runs this only delays archiving by the not-responding day (about 7 days), because the reminder notes are real activity by design ("messages sent").
- But a "Run as of" jump can never archive an un-reminded pre-op episode in one run. Options: stamp job-made notes and activity with the as-of time; run the auto-archive before the reminders; or change the plan to use a patient with no email.
- Also to decide: should a system "Not responding" mark count as activity at all?

The GIF is `/Users/rahul/Downloads/S22-T18.gif`.

### F-1 (T-12 step 1): Settings silently clamps 0 and 366 instead of refusing them. **Minor**

Signed in as Ada Admin: Settings → Settings for **Hospital A1** → untick "Use the trust's setting" under "Archive episodes this many days after discharge".

**Steps:**
1. Type `0` and click **Save Hospital A1's settings**.
2. Then type `366` and save.

**Expected:** Both are refused with a message. Only `7` is saved.

**Actual:**
- On blur the box changes to `1` (then `365`), and the green "Hospital A1's settings were saved." appears.
- `GET /api/o/…/settings` showed `autoArchive.afterDischargeDays: 1`, then `365`.
- The API itself refuses both: a direct PATCH with 0 or 366 returns 400 "Archive episodes this many days after discharge: enter a whole number from 1 to 365."
- The clamping comes from the shared Mantine `NumberInput` with `min`/`max` in `apps/web/src/features/settings/SettingsPage.tsx`, line 217 (not changed in S22). It probably affects every number setting.
- **Risk:** an admin who mistypes 0 saves 1, without noticing, and the next night's run then archives every discharged episode at that hospital after 1 day.

There were no console errors. The GIF is `/Users/rahul/Downloads/S22-T12.gif`.

I set the value back to 7 and continued. At the end I ticked "Use the trust's setting" again, so Hospital A1 has no override (checked: `values: {}`).

## Found while exploring

- **Minor, wrong text.** After the patient closes their account, the episode's **Patient's invite** card says "No invite has been sent. The General notes say why.", although an invite was sent. The notes say "Invite sent by email." and "The patient closed their account on 2026-10-21." Seen on `Sam S22 Close` and `Sam S22 Links`. The text comes from `EpisodePage.tsx` line 201, presumably because the invite was cleared. Something like "The patient closed their account, so their link was stopped. Send a new invite if they still need it." would be correct.
- **Archived rows still show "1 open task"** under the ARCHIVED badge (Show archived list, e.g. `S22 Worklist`, `S22 TwoTabs`). This is probably fine, because the tasks can't be changed, but it can look like work is outstanding. Cosmetic, for Rahul to judge.
- **Server-side guards all hold.** I checked these by calling the API directly as the signed-in user:
  - Cara archiving or un-archiving: 403 "Only a super clinician or an admin can …".
  - On an archived episode, each of these returns 409 "This episode is archived, so it can't be changed…": complete task, add task, edit dates, save answers, re-open review. Resend invite returns 409 "…so no invite can be sent…".
  - Archiving twice: 409. Un-archiving a non-archived episode: 409 "This episode isn't archived.".
  - Blank reason: 400. Stale revision: 409 stale. A 1,001-character reason: 400 "The reason is too long (1,000 characters at most).".
  - Bea (Trust B) using a Trust A episode id: 404.
- **Double-clicks:** double-clicking Archive episode archived once (one history row, one note). Double-clicking the red Close my account closed the account once, with no error.
- **Settings for the 6-month rule.** The plan says "Archive pre-op episodes with no activity for (months)". The hospital form shows that field with "Use the trust's setting" (I didn't change it; T-18 step 6 was optional).
- **Clock note.** /qa/jobs's "Now + N days" uses the browser clock, and the results showed times about a minute different from the server's note times. That's harmless, but it's worth knowing for T-18-style tests that use "between" times.

## Console and network errors

- No app console errors or warnings were seen on the pages I tracked: the episode page, /qa/jobs and the patient pages.
- I saw no unexpected 4xx or 5xx from the UI. The expected 409s were in T-04.
- 1Password latched onto the /signin email field once. It blocked typing, and the extension then couldn't script that tab, so I closed the tab and used a fresh one with the form tool. This is not an app problem.

## GIFs

- `/Users/rahul/Downloads/S22-T02.gif`: happy path, archiving with a reason (Sue)
- `/Users/rahul/Downloads/S22-T12.gif`: F-1, 366 clamped to 365 and saved
- `/Users/rahul/Downloads/S22-T13.gif`: closing an account at phone size
- `/Users/rahul/Downloads/S22-T18.gif`: F-2, the result line and S22 Idle's General notes after the 6-month run

## Data left behind

These are all made-up `S22 …` patients, at Trust A Hospital A1 (plus `S22 A2` at Hospital A2). The 6-month runs archived almost every pre-op episode in the database, as the plan warned (394 episodes in the first run). Hospital A1's settings are back to the trust's.
