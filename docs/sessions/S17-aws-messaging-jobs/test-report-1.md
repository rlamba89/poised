# S17 test report 1

**Summary:** 2 passed, 1 partly passed, 0 failed, 14 blocked (staff sign-in). The test round as a whole is **blocked**.

- **Environment:** QA, https://main.d33kh0gi8vk83v.amplifyapp.com (`/api/health` → `{"status":"ok"}`). Local HEAD `b307ba5`, working tree not clean (S17's uncommitted changes; QA runs this tree, deployed by hand). Tested 14 Oct 2026, about 23:27–23:50 BST, by the manual tester in automatic mode.
- **Why blocked:** every staff case needs a Cognito sign-in with a password and an authenticator code. The tester agent isn't allowed to type or paste a password into a sign-in page on a hosted, non-local site, so it didn't sign in as Cara, Sam, Val, Bea or Ada. It ran only the cases that need no staff sign-in: the read-only AWS checks, the schedule (through the jobs Lambda's logs), and the patient's "send me a link" page. **No real messages were sent.** The tester created no episodes and changed nothing in AWS.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Invite captured | Blocked | Needs Cara's sign-in. Hand check for Rahul. |
| T-02 Schedule runs the jobs by itself | **Partly passed** (AWS side) / Blocked (UI) | The logs show a run every 15 min with no gaps, nobody pressing Run jobs (20:31:40 → 22:46:40 UTC, all at :01:40/:16:40/:31:40/:46:40). `/qa/jobs` page not seen (needs Sam). |
| T-03 Reminder from a scheduled run | Blocked | Needs Sam (Move the clock) and Cara. |
| T-04 Bounced email shows as undelivered | Blocked (UI) | Needs Cara. The AWS side was seen working from someone else's bounce episodes (see T-08.4). |
| T-05 Resend starts afresh | Blocked | Needs Cara. |
| T-06 Undelivered only for its own hospital | Blocked | Needs Val and Bea. |
| T-07 Patient view shows none of this | Blocked | Needs Sam's `/qa/messages` to get an `/i/…` link. |
| T-08.1 Schedule | **Pass** | `["ENABLED","rate(15 minutes)",0]`. |
| T-08.2 SMS configuration set | **Pass** | A `protect-…` ID, with the `poised-uk-qa-deliveries` topic as its event destination. |
| T-08.3 Protect country rules | **Pass** | GB `ALLOW`, US `BLOCK`, FR `BLOCK`. |
| T-08.4 Alarms | **Pass** | `poised-uk-qa-delivery-failures` went to `ALARM` at 23:27:41 BST, about a minute after 4 `DELIVERY_FAILED` lines (22:26:24–26 UTC). `poised-uk-qa-jobs-failed` is `OK`. |
| T-09 Sign-in link captured | Partly done, Blocked | `/signin` with a made-up `@example.com` address answered "Check your email…", with no console errors. Checking the capture needs Sam. The window didn't resize to phone size (see below). |
| R-01 Cognito sign-in / sign-out | Blocked | Password entry. |
| R-02 Invite link + date of birth | Blocked | Needs T-01's link. |
| R-03 Now + 7 days → Not responding | Blocked | Needs Sam and Cara. |
| R-04 New episode search | Blocked | Needs Cara. |
| R-05 Audit log | Blocked | Needs Ada. |

T-08 counts as 1 passed case in the summary (4 checks), and T-02's AWS side as the second pass.

## Failures

None found.

## Hand checks for Rahul

**Everything that needs a staff sign-in** has to be checked by hand, or by a run that can sign in:
- T-01, T-03, T-04, T-05, T-06, T-07;
- T-02's `/qa/jobs` page;
- T-09's capture check;
- R-01 to R-05.

Also the plan's own H-1, H-2 and H-3.

**H-2 note:** the alarm went to `ALARM` at 23:27:41 BST on 14 Oct, so an "ALARM: poised-uk-qa-delivery-failures" email should have reached Rahul's alert address around then, if the subscription is confirmed.

## Found while exploring

- **The jobs Lambda logs nothing on a scheduled run that has no due jobs.**
  - `cmd/jobs` logs "jobs: ran …" only when `sum.Jobs > 0`.
  - A cold start logs "jobs on Lambda …", but a warm run with nothing due leaves only Lambda's `START`/`END`/`REPORT` lines. So the app's own log can't show that the schedule fired. You have to read the `REPORT` lines (they were there for every slot).
  - Not wrong, but it makes T-02's "from the logs" check harder. **Cosmetic / operability.**
- **One scheduled run did work:** at 22:31:40 UTC, `ran 11 … 0 reminders, 0 not responding, 0 failed`. So 11 jobs ran and did nothing visible. This may be expected (jobs that are checked and closed or rescheduled), but it's worth a glance.
- **Someone created two bounce episodes at about 23:26 BST** (episodes `69756a84…` and `d1fd7150…`). Each got **two** `DELIVERY_FAILED` email lines about 2 seconds apart, which fits an invite plus a Resend, or two sends. The tester didn't create these, and didn't open them.
- **Signed out,** `GET /api/qa/jobs` and `GET /api/qa/messages` answer 401 (as expected).

## Console and network errors

- On `/signin` (desktop size) there were no console errors or warnings.
- `resize_window` to 390×844 reported success, but the screenshots stayed at desktop width. So T-09 ran at desktop size, not phone size. This is a tool limit, not an app problem.

## GIFs

None. No failures were found, and the main happy paths (staff) couldn't be run.
