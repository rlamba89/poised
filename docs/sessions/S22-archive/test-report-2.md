# S22 test report 2 (retest)

**Summary: 14 passed, 1 failed, 0 blocked.** I ran 15 cases: the three changed cases (T-12, T-16, T-18), the five regressions (R-1 to R-5), and seven main happy paths (T-02, T-10, T-13, T-15 and the parts of T-03, T-08 and T-09 I re-checked while exploring). T-18's optional step 6 was also run.

- **Commit:** 9d37cf7, with the S22 changes and the fixes for test report 1 uncommitted in the working tree.
- **Date:** 2026-10-21, 15:31 to 15:49 BST.
- **URL:** http://localhost:3000 (API on :8080). The conductor started it with `make dev`.
- **Browser:** Chrome with the "trk" profile, driven through Claude in Chrome. The window was 1400×773. For phone size I used a 390px-wide iframe of the app inside the page.
- **Tester:** the automatic manual-test agent.

**How I set up data:**
- I made two episodes with the New episode form: `S22 R2 Close`, which also covered R-5, and the patient's journey in T-13, T-16 and R-4.
- For the other setup episodes I used the app's own API, from a browser tab signed in as the user under test: create, fill in for the patient, validate, complete the review, then Ready for admission, Admitted and Discharged.
- Everything under test was done in the UI: archiving, Settings, /qa/jobs, the patient pages, closing the account, undo and cancel, comments, tasks and worklists.
- New patients have surnames starting `S22 R2`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 | Pass | Sue archived `S22 R2 Hand` (Discharged) by pressing Enter with the reason `S22 treatment finished`. The grey box, ARCHIVED badge, last history row "Archived: Discharged → Archived" by Sue Super with the reason, and the note "Episode archived: S22 treatment finished" are all correct. With no reason, the box has no "Reason:" and the note says "Episode archived". The invite card is hidden. |
| T-10 | Pass | At +29 days nothing was archived (0) and the episode stayed Discharged. At +31 days it shows "Archived from Discharged on 21/11/2026, 15:34 by Automatic (Auto-archive: 30 days after discharge)", with the matching history row and an Automatic note. These are now dated with the run's time, not the real time (see the notes below the table). |
| T-12 | **Pass** (F-1 fixed) | `0` stays `0` when you click away. Save shows "Archive episodes this many days after discharge: enter a whole number from 1 to 365." and nothing is saved (`values: {}`). `366` behaves the same way. "Mark "not responding" on day" = `0` gives "…enter a whole number from 1 to 90." `7` saves. A run as of +8 days archived `S22 R2 A1` ("Automatic (Auto-archive: 7 days after discharge)") and left `S22 R2 A2` at Hospital A2 Discharged. At +31 days A2 was archived ("30 days after discharge"). The override was then cleared: both hospitals are back to `values: {}`. |
| T-13 | Pass | At phone size: the Close your account card, then the orange warning "Your Demo pre-op HQ for Hospital A1 hasn't been sent…", with red Close and Keep buttons. Nothing is wider than 390px (scrollWidth 388). Keep returns to the first view. Close shows "Your account is closed…", and after a reload "You've been signed out". |
| T-15 | Pass (notes) | The General note says "The patient closed their account on 2026-10-21." (by Patient). I didn't re-check the audit log this round. |
| T-16 | **Pass** | The invite card says, word for word, "The patient's link no longer works: the General notes say why (for example, they closed their account). Send a new invite if they still need it." and has **Send invite**. Clicking it gives "Invite sent again by email." At phone size the new link opens the questionnaire. The new home shows it as TO DO, with Close your account at the bottom. |
| R-1 | Pass | Ready for POA → On hold, then "Undo: back to Ready for POA" with a reason (history shows "(undone)" and "Undone: On hold → Ready for POA"). Cancel episode with Patient withdrew gives CANCELLED and the cancelled box. Add comment works on the cancelled episode. |
| R-2 | Pass | On hold: 2 in the tab and 2 rows. Procedure within 14 days: 25 and 25 rows. An Anaesthetist task added in the UI to `S22 R2 Close` took Anaesthetic review needed from 4 to 5, and the episode is listed there. Open tasks went from 7 to 8. |
| R-3 | Pass | Now + 2 days sent "Hospital A1: a reminder about your health questionnaire" to s22r2.remind@example.test (shown on /qa/messages). The General note "Day 2 reminder sent by email." is marked Automatic. |
| R-4 | Pass | At phone size the patient answered both sections, sent them through the app's own "Send your answers?" box, and the home shows SENT with View. |
| R-5 | Pass | Picking procedure "E2E Hip 2010 194351 dd57 (E2E-1)" chose its default HQ, "E2E Procedures 2010 194351 87aa v1". I then changed the HQ to Demo pre-op HQ v1 for T-13. The procedure shows on the episode. |
| T-18 | **Fail** (step 5 only) | Steps 1–4 pass, so F-2 is fixed. `S22 R2 Idle` (with an email) was archived with no reminder or "Not responding" note. `S22 R2 Busy` stayed HQ not complete. Step 5 fails: Busy is **not** archived at 6 months and a day. See F-3. Optional step 6 passes: with A1 set to 2 months, a new episode was archived at 2 months and a day ("Auto-archive: 2 months without activity"), and the override was cleared afterwards. |

**Notes on behaviour that changed since report 1, as the F-2 fix intends:**
- Everything a job run writes is dated with the run's "as of" time, not the real time. That covers the archive box, the status history row and the General notes, including reminders and "Not responding".
- For example, T-10's box now reads "on 21/11/2026, 15:34". The plan's T-10 says "on <now>", so its wording is slightly out of date. This is not a fault.
- The captured message on /qa/messages still shows the real time (21/10/2026, 15:43), while the note for the same reminder is dated 23/10/2026, 15:44. That's harmless, but the two don't match.

## Failures in detail

### F-3 (T-18 step 5): `S22 Busy` isn't archived at "6 months and a day". **Minor.** The plan's expectation looks out of date after the F-2 fix; the app behaves as designed.

I was signed in as Cara at Hospital A1.

**Steps:**
1. 15:43:34: made `Sam S22 R2 Idle` (s22r2.idle@example.test). Its last note is "Invite sent by email." at 15:43.
2. 15:46: made `Sam S22 R2 Busy` (s22r2.busy@example.test) and added the comment `S22 rang the patient` at 15:46:23.
3. On /qa/jobs, set Run as of to **21/04/2027 15:45** and ran the jobs. The result was "Ran 297 jobs as of 21/04/2027, 15:45: 2 reminders sent, 2 marked "not responding", 405 episodes archived."
   - Idle was archived ("Automatic (Auto-archive: 6 months without activity)"). It got no reminder or Not responding note. **This passes.**
   - Busy is still HQ not complete. It gained "Day 5 reminder sent by email." and "Not responding: the HQ hasn't been sent 7 days after the invite.", both "Automatic on 21/04/2027, 15:45". **This passes too**, as the plan allows.
4. Set Run as of to **22/04/2027 15:50** (6 months and a day) and ran the jobs.

**Expected (plan, step 5):** `S22 Busy` is archived too.

**Actual:**
- The result line was "Ran 295 jobs as of 22/04/2027, 15:50: 0 reminders sent, 0 marked "not responding", 0 episodes archived."
- Busy is still **HQ not complete**.

**Why:**
- Since the F-2 fix, step 3's reminder and Not responding notes are dated 21/04/2027 15:45, and they count as activity (answers 3: messages sent count).
- So Busy's last activity is now 21/04/2027, and its 6 months run from then.
- To confirm this, I ran as of **22/10/2027 15:50**. Busy was then archived ("Auto-archive: 6 months without activity", 2 episodes archived).
- In a real nightly run this can't happen the same way: the day-5 reminder and the day-7 mark would be dated days 5 and 7, so the archive would come about 6 months after day 7. That is consistent with answer 3.

**For the builder:**
- This looks like a test-plan fix, not a code fix. For example, step 5 could run as of "6 months and a day after step 3's run" (22/10/2027), or say that step 3's reminders restart the clock.
- If Rahul wants a system-written "Not responding" mark or reminder **not** to count as activity, that is a product decision.

There were no console errors. The GIF is `/Users/rahul/Downloads/S22-R2-T18.gif` (steps 3–5 on /qa/jobs).

## Found while exploring

- **Settings, typing "7.5"** into "Archive episodes this many days after discharge" gives **75** in the box, because the decimal point is dropped as you type. I didn't save it. Mantine's whole-number box does this. It's visible in the box, but an admin who types 7.5 and doesn't look would save 75 days. **Cosmetic/Minor**, for Rahul to judge. I re-ticked "Use the trust's setting", so nothing changed.
- **Message time and note time differ** in a job run "as of" a later time: /qa/messages shows the real time, and the General note shows the run's time (see the notes under the results table). This is QA-only and harmless.
- **The cross-hospital guard holds.** As Cara (Hospital A1 only), Hospital A2's episode returns 404 by the A2 path and by the A1 path, and un-archiving it returns 404 "Episode not found.".
- **As Cara, an archived episode** shows no Un-archive, Add comment, Add task or Edit procedure, and no enabled buttons apart from Sign out.
- **The Run as of box on /qa/jobs** starts at "now + 2 days" when you open the page, not at now. That is fine, but worth knowing.

## Console and network errors

- No app console errors or warnings on the pages I tracked: the episode page, Settings, /qa/jobs and the episodes list.
- The only 4xx responses were my deliberate cross-hospital probes (404).
- `POST /api/logout` and `POST /api/dev/login` showed 503 in the extension's network log, although sign-out and sign-in worked. These are the known false 503s.
- 1Password latched onto the New episode form once ("Cannot access a chrome-extension:// URL"). I closed that tab and used the form tool in a fresh tab. This is not an app problem.

## GIFs

All are in `/Users/rahul/Downloads/`:
- `S22-R2-T02.gif`: happy path, Sue archives with a reason using Enter.
- `S22-R2-T12.gif`: 0 and 366 refused and kept as typed, then "not responding" 0 refused, then 7 saved.
- `S22-R2-T13.gif`: closing an account at phone size, with the warning, Keep, Close and the signed-out reload.
- `S22-R2-T18.gif`: F-3, the 6-month runs on /qa/jobs.

## Data left behind

- All the test patients are made up, with surnames starting `S22 R2`, at Trust A Hospital A1, plus `S22 R2 A2` at Hospital A2.
- The /qa/jobs runs went as far as 22/10/2027. As the plan warned, they archived almost every pre-op episode in the database (405 in the 21/04/2027 run). They also left automatic notes dated in the future on the remaining episodes.
- Hospital A1's and Hospital A2's settings are back to the trust's (`values: {}`).
