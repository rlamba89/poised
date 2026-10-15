# S17 test report 2 (retest, local)

**Summary:** 11 passed, 0 failed, 2 partly blocked (T-04 and T-05: the "not delivered" display can't be produced locally). T-08 and T-02's AWS side weren't redone; they passed on QA in test report 1.

- **Environment:** local `make dev` (run by the conductor), at http://localhost:3000 (Cara) and http://tester.localhost:3000 (Sam, Val, Bea, Ada). The patient's view was at http://patient.localhost:3000, in a 390×844 iframe. API `/api/health` → `{"status":"ok"}`.
  - HEAD `b307ba5`; the working tree isn't clean (S17's uncommitted changes).
  - Tested 15 Oct 2026, 08:03–09:05 BST, by the manual tester in automatic mode.
- **Why local:** answers.md item 10. Locally, staff sign in with the dev login and messages are only captured (`MESSAGING=capture`). Locally the jobs run on the server's 60-second timer, not on AWS's 15-minute schedule.
- **Test data:**
  - `MT S17 0805 Lockwood`: born 12 Mar 1975, mobile 07700 900417, `mt-s17-0805@example.com`. This is T-01's episode, `c61801fd…`.
  - `MT S17 0808 Bouncer`: born 4 Jul 1968, no mobile, `bounce@simulator.amazonses.com`. This is T-04's episode, `a011f4c5…`.
  - Both are in Hospital A1, on HQ "MT S03 1625 workflow v1". Nothing was really sent.
- **The timer's clock was set back to real time** after T-03 (offset 0, confirmed by `GET /api/qa/jobs` at 09:04).

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Invite captured | **Pass** | General notes say "Invite sent by text message and email.", with no "Invite not delivered" badge. `/qa/messages` has one text and one email, each with the same `/i/…` link. The database (read-only) has two `episode_messages` invite rows with `capture-2067` / `capture-2068` and no failure. |
| T-02 Jobs run by themselves (`/qa/jobs` page) | **Pass** (local timer) | "The timer" reads "On: the local server runs the jobs every 60 seconds, as of real time." The last run moved on by itself (08:05 → 08:06 → 09:02, …), and nobody pressed Run jobs. AWS's 15-minute schedule text can't show locally; QA's schedule passed in report 1. |
| T-03 Reminder from a timer run | **Pass** | Days 2 + Hours 1 → Move the clock. The page then said "as of real time + 2 days 1 hour", and the next run was "as of 17/10/2026, 09:07". `/qa/messages` had the reminder text with a new `/i/…` link, next to the invite. Cara's General notes had an **Automatic** "Day 2 reminder sent by text message and email." The reminder rows are in `episode_messages` (kind `reminder`). Back to real time was clicked afterwards. |
| T-04 Bounce shows as undelivered | **Partly passed / Blocked** | Steps 1–2 pass: "Invite sent by email.", with no badge and `undelivered: []` from the API. Steps 3–5 are blocked: locally nothing bounces (capture never fails), so the badge, the "Not delivered" box and the Automatic note can't appear. The episode opened from the list (step 4) with no console errors and no failed requests (step 5). **Hand check for Rahul** (see below). |
| T-05 Resend starts afresh | **Partly passed / Blocked** | Resend showed "Invite sent again by email." in the card and in General notes. A new `episode_messages` invite row (`capture-2088`) was recorded. The undelivered half (the box cleared, then back for the new email only) is blocked, as for T-04. |
| T-06 Only the episode's own hospital | **Pass** | **Val** (A2 only) opened T-04's URL and saw "You don't have access to this hospital". The API answered 404 "Hospital not found.", and 404 "Episode not found." through her own A2 path. **Bea** (Trust B) got the same page, and the API answered 404 on the A1 path and 404 through her own B1 path. Note: the episode had no undelivered rows at the time (see T-04). |
| T-07 Patient view shows none of this | **Pass** | The `/i/…` link from T-01's latest email was opened at 390×844: Continue → date of birth → "MT S03 1625 workflow" with "Hello MT S17 0805…" and its two sections. Nothing about delivery, messages or notes appeared. |
| T-08 AWS side | Not redone | It passed on QA in report 1 (this round is local, and the brief says not to touch AWS). |
| T-09 Sign-in link captured | **Pass** | At 390×844, `/signin` → T-01's email → Enter showed "Check your email…". As Sam, `/qa/messages` had "Your Poised sign-in link" with an `/s/…` link. |
| R-01 Sign in / Sign out | **Pass** (dev login) | Locally, Cara signs in with the dev login, and Sign out returns to `/login`. Cognito's own sign-in passed on QA in S12's verification. |
| R-02 Invite link + DOB (phone) | **Pass** | A wrong date of birth (13 3 1975) showed "That date of birth doesn't match our records… You have 4 tries left." The right one opened the questionnaire. Enter submits the form. |
| R-03 Now + 7 days → Not responding | **Pass** | "Ran 17 jobs as of 22/10/2026, 09:03: 14 reminders sent, 17 marked "not responding"." T-01's episode shows a red **NOT RESPONDING** badge, with Automatic notes "Day 5 reminder sent…" and "Not responding: the HQ hasn't been sent 7 days after the invite." No "Invite not delivered" badge appeared, which is correct because nothing failed. |
| R-04 New episode search | **Pass** | "Lockwood" with 12/03/1975 found exactly "MT S17 0805 Lockwood, 12 March 1975" with **Choose**. |
| R-05 Audit log | **Pass** | Ada Admin → Hospital A1 → Audit lists Cara's "Opened an episode", "Opened the answers" and "Searched patients" for MT S17 0805 Lockwood (09:02–09:03), and her views of MT S17 0808 Bouncer. |

## Failures

None.

## Hand checks for Rahul

- **T-04 steps 3–5 and T-05's second half: an undelivered invite on the episode page.**
  - These need a reported delivery failure. Locally, messages are only captured, so none ever fails, and the app has no tester's tool to report one.
  - I tried a local stand-in for SES's report, by feeding a bounce event to the jobs function on this machine. The permission system refused it (it writes to the shared local database), so I didn't pursue it. Nothing was written: the attempts that did run stopped at the configuration step, before touching the database.
  - So the badge, the "Not delivered" box, its wording, the Automatic note, and Resend clearing then re-showing it were **not seen in a browser this round**.
  - They're covered by the builder's automated tests (`undelivered_integration_test.go`, `cmd/jobs`'s integration test, `undelivered.test.ts`). The builder saw them on QA on 14 Oct: the bounce episode showed "Invite not delivered" about 1 s after the send.
  - **Rahul:** on QA, signed in as Cara, run T-04 and T-05 as written (the `bounce@simulator.amazonses.com` episode, then Resend). Also check T-06 against that episode once it has an undelivered row.
- **T-06 with an undelivered row present** (see above). With no undelivered rows, the refusals are whole-episode 404s, so nothing could leak.
- The plan's own **H-1, H-2 and H-3** (real messages; unchanged from report 1).

## Found while exploring

- **Each Run jobs summary can count more results than jobs:** "Ran 17 jobs …: 14 reminders sent, 17 marked "not responding"". This reads oddly, because 14 + 17 > 17. One follow-up job can both send the day-5 reminder and mark the episode not responding when "as of" jumps 7 days, so it's probably right. The wording could say "17 episodes checked". **Cosmetic.** S16's wording, not S17's.
- **The invite card's "Last sent" moves on when a reminder goes**: 08:05 → 08:07 after the day-2 reminder, then 09:02 after the run. That's arguably right ("last sent" = the last message), but the line sits under "Patient's invite" and reads as the invite's own time. **Cosmetic.** S16 behaviour, not S17's.
- **Already in pending.md (not new):** each staff episode view writes two audit rows (React StrictMode in `make dev`). Seen again in R-05.
- Signed out, local `GET /api/qa/jobs` and `/api/qa/messages` answer 200. This is by design locally: capture mode with the dev login. QA answered 401 in report 1.

## Console and network errors

- None. Every `/api/` request on the episode, new-episode, list, `/qa/jobs`, `/qa/messages` and audit pages answered 200. The expected exceptions were Val's and Bea's 404s in T-06, and 401s from a check I made by hand of `/api/p/…` outside the patient's session.
- The episode page, the list and the patient pages logged no console errors or warnings.

## GIFs

- `~/Downloads/S17-retest-happy-path.gif`: from creating T-01's episode, through `/qa/messages`, the timer reminder, the bounce episode and Resend, to Not responding (50 frames). There were no failures to record.
