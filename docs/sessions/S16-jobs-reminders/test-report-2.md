# S16 test report 2 (retest): scheduled jobs, reminders and "not responding"

**Summary: 25 passed, 0 failed, 0 blocked.** Report 1's failure (X-1) is fixed. Wrong dates of birth now count per episode across the invite link and every reminder link, and the fifth wrong date locks them all together. A locked episode gets no reminders, only a staff Resend unlocks it, and the staff banner shows the real lock state. The new cases T-16 to T-18, the extended T-14 and R-6 all pass. I re-ran the core reminder cases T-01 to T-13 and T-15, and the regression checks R-1 to R-5; all pass. I also probed the lock hard, across the invite link, reminder links, open sessions and parallel requests, and found no way past it through the invite or reminder links. The one way past it is S15's sign-in link, which is by design and is noted below.

- **Environment:** `661cdff`, with S16's work (round 2) not yet committed. The working tree is dirty, as the conductor expected. Tested on 13 Oct 2026, 21:41–22:04 BST, against `make dev` at http://localhost:3000 (API :8080) with the one-minute jobs timer on. The patient was on http://tester.localhost:3000.
- **Browser:** Claude in Chrome, run by the tester in automatic mode.
- **Phone size:** `resize_window` reports success, but the viewport stays 1512×805, the same as in report 1. So I ran the patient pages in a **390 px-wide iframe** (390×780) on tester.localhost. Nothing scrolled sideways (`scrollWidth` was at most 390).
- **Test data:** all made up, born 17/05/1980.
  - Through the UI: `Sam Rem16Mtb` and `Rem18Mtb`.
  - Through Cara's own API session, to save time on setup (I said so each time): `Rem19Mtb` to `Rem23Mtb`, and `Rem01Mtc` to `Rem15Mtc`, `RemSubMtc`.
  - Every patient action, the jobs page, Resend, the episode pages, settings and the audit log were used through the UI. The exceptions are the parallel and direct date-of-birth probes, which are marked "(API)".
  - Mobiles were `+44 7700 9009xx`, picked to avoid numbers with earlier messages. Emails: `rem01.mtc@example.org` and `rem23.mtb@example.org`.
- **Database:** read-only `SELECT`s only, to confirm counts, notes, `dob_failed_attempts`, link lock state and the clock.
- **Shared data I changed, then put back:**
  - Hospital A2's settings: `{}` again ("Use the trust's setting").
  - Timer clock: **Back to real time**; the offset is 0, confirmed in the database.

## Results

| Case | Result | Note |
|---|---|---|
| **T-16** Shared tries across links (phone width) | Pass | 3 wrong on the invite link: 4, 3, then 2 tries left. The reminder link continued at **1 try left**, then "This link is locked". The invite link then shows locked at Continue. The right date (API) is 403 `locked` on both links. Staff banner: "…so every link sent to the patient for this episode is locked, and no reminders go. Resend the invite…". |
| **T-17** Locked: no reminders; Resend unlocks | Pass | At +5 days, still 2 texts. Note: "Day 5 reminder not sent: the patient's link is locked after 5 wrong dates of birth. Resend the invite to send them a new link. \| Automatic". After Resend the lock box is gone. The resent link: one wrong gives **4 tries left**, then the right date opens "Hello Sam". The old invite and reminder links say "This link doesn't work". At +7: **Not responding** badge and note, with no further text (3 in all). |
| **T-18** A right date starts the count again | Pass | 3 wrong on the invite link (2 left). The right date on the reminder link opens the HQ. After Sign out, one wrong on the invite link gives **4 tries left**. |
| **T-14** Jobs page: invalid input, reload, links | Pass | Day part cleared only: red "Enter a date and time." (fixed since report 1). Fully empty: the same. Days 91 and hours −1: red "Enter whole days and hours, 90 days at most.", and the card stays on real time. Reload, the links both ways and Back all work. |
| **R-6** Sign-in links keep their own tries | Pass | Episode `Rem18Mtb` locked (count 5). On its `/s/…` sign-in link, one wrong date gives **4 tries left**, and the right date opens `/home`. |
| T-01 Day 2 reminder, text and email | Pass | `Rem01Mtc`: 2 texts and 2 emails. Wording and subject ("Hospital A1: a reminder about your health questionnaire") as in the plan. The reminder's link is new. |
| T-02 Nothing sent twice | Pass | +2 days again: "Ran 0 jobs". As of 17/10 21:58 (day 4): "Ran 0 jobs". Still 2 texts and 2 emails. |
| T-03 Episode shows the reminder | Pass | "Day 2 reminder sent by text message and email. \| Automatic on …", the invite-card text, and no badge. |
| T-04 Both links work (phone width) | Pass | The reminder link and the invite link each go Continue → date of birth → "Hello Sam". No sideways scroll. |
| T-05 Day 5, then day 7 | Pass | `Rem01Mtc`: a third text and email at +5. Nothing new at +7. **NOT RESPONDING** badge and both notes. +7 again: "Ran 0 jobs". |
| T-06 Sending stops reminders | Pass | `Rem06Mtc`: sent through the reminder link. Still 2 texts after +5 and +7. **READY FOR REVIEW**, with no badge and no note. |
| T-07 Starting stops reminders; day 7 still flags | Pass | `Rem07Mtc`: one answer saved. Only the invite at +2 and +5. At +7: flagged, with no message. |
| T-08 Staff answers count as started | Pass | `Rem08Mtc`, after Fill in for the patient: only the invite at +2. |
| T-09 Reminder at night | Pass | `Rem09Mtc`, as of 16/10 22:30: "1 reminder sent". 2 texts. |
| T-10 Catch-up: only the latest, once | Pass | `Rem10Mtc`, +7 straight away: one reminder ("Day 5 reminder sent by text message."), "Not responding", and 2 texts in all. |
| T-11 The timer sends by itself | Pass | Clock moved 2 days 1 hour. The reminder for `+447700900931` arrived at the next tick (21:57:10, "as of 15/10 22:57") without Run jobs. Back to real time: the card says "as of real time" and the offset is 0. |
| T-12 Hospital's own days and wording | Pass | A2 set to days `1`, not responding `3`, and `{{hospital}} reminds {{firstName}}: {{link}}`. Nothing before day 1. "Hospital A2 reminds Sam: …/i/…" as of 14/10 22:05. As of 16/10 22:05: "Not responding: … **3** days after the invite." Settings restored to `{}`. |
| T-13 No contacts, no follow-up | Pass | `Rem13Mtc`: "No invite sent…". No job, no note and no badge after +7. |
| T-15 Resend doesn't restart the days | Pass | `Rem15Mtc`: invite, resent invite and the day 2 reminder (3 texts). The first link is 404 and the resent and reminder links work (API Continue). |
| R-1 Invite on create | Pass | Created through the UI: "Invite sent by text message.", Continue and date of birth work. |
| R-2 Lock and Resend (no reminders yet) | Pass | `Rem20/22/23Mtb` locked before any reminder. The banner shows. Resend on `Rem23Mtb` sends a link that works ("4 tries left" after one wrong). |
| R-3 Patient home | Pass | `/home` at 390 px: "Hospital A1 · TO DO · Trust A demo HQ · For Sam Rem18Mtb", the contacts card, and no sideways scroll. |
| R-4 Patient search | Pass | "Rem16Mtb" finds Sam Rem16Mtb, 17 May 1980. |
| R-5 Audit log | Pass | As Ada: "Opened an episode", "Resent the patient's invite", and the patients' REFUSED (401/403) date-of-birth rows. No rows for the jobs, as designed. |
| X-1 (report 1) Lock is per link | **Fixed** | Covered by T-16 to T-18 and the probes below. |

## Failures

None.

## Probing the lock (beyond the plan)

All of these behaved correctly:

1. **A count carries over to links sent later.** `Rem19Mtb` had 3 wrong dates on the invite link before day 2. The day 2 reminder's link then said "**1 try left**" after one wrong date. A reminder doesn't reset the count.
2. **Three live links lock together.** `Rem19Mtb` had the invite, day 2 and day 5 links live, with 4 tries used. One wrong date on the day 5 link locked all three: Continue gives 403 on the others, and so does the right date (API).
3. **Locked before day 2, then Resend, then day 5 goes.** `Rem23Mtb`, text and email:
   - +2 days: "Day 2 reminder not sent…".
   - Resend: "Invite sent again by text message and email".
   - +5 days: "Day 5 reminder sent by text message and email".
   - +7 days: "Not responding".
   
   The resent link and the day 5 link share a fresh count (4, then 3 tries left), and the first link is 404.
4. **Locked through day 7, no Resend.** `Rem20/21/22Mtb`: notes "Day 2 reminder not sent…" and "Day 5 reminder not sent…", then "Not responding", with the invite as the only message.
5. **An open session ends when another link locks the episode.** The patient signed in on `Rem18Mtb`'s reminder link, which reset the count to 0. Five wrong dates were then sent on the invite link from a different browser origin (API). The patient's next save on their open page gave "You've been signed out — Not saved…".
6. **Parallel wrong dates.** 12 wrong dates sent at once on `Rem21Mtb` (API) gave exactly **4** "tries left" answers (401) and 8 "locked" answers (403). The lock holds at the fifth.
7. **An unreadable date isn't a try.** `1980-02-30` and an empty date give 400 "Enter your date of birth as a real date…", and the count stayed at 4.
8. **Old links after a Resend don't count.** A wrong date on a replaced link is 404, and the episode's count doesn't change.

## Found while exploring

1. **The S15 sign-in link gets the patient into a locked episode, for Rahul.** This is by design, but the staff banner now says more than is true.
   - On `Rem18Mtb`, every invite and reminder link was locked. The patient then used **Get a sign-in link** with their mobile. The `/s/…` link has its own five tries (R-6, as decided), so a right date opened `/home`, and they could open and **answer** the locked episode's questionnaire ("Saved").
   - Meanwhile the staff page still said "…so **every link** sent to the patient for this episode is locked…".
   - This follows the recorded decisions: "Sign-in links (S15) keep their own per-link count", and the session belongs to the account (pending.md, "A patient session opens every episode…").
   - The sign-in link only goes to the contact on record, so it doesn't widen what a link-holder can guess.
   - Worth a line in the banner ("…they can still sign in with their mobile or email"), or a decision on whether a lock should also stop sign-in for that episode. Not reported as a failure.
2. **The "signed out" text points to a locked link (Cosmetic, older than S16).** After the lock ended an open session, the page said "For your security, you've been signed out. Please open the link in your text message or email again." Those links are now locked, so the patient lands on "This link is locked". "Get a sign-in link" is offered just below, so they aren't stuck.
3. **The stored count can go past 5 (Cosmetic, no effect).** After the 12 parallel wrong dates, `episodes.dob_failed_attempts` was **9**, although only 4 tries were answered as "tries left" and the links locked at the fifth. Requests that were already waiting still added to the count. A Resend sets it back to 0 and the lock is unaffected, so this has no visible effect. Mentioned for whoever reads the column.
4. **Not to report, already logged:** double-clicking Run jobs, real vs "as of" times in notes, and the reminder arriving with "Not responding" on catch-up. I saw each again and left them out, as the plan says.
5. **Test-tool behaviour, not app bugs:**
   - Clicks and typing in a tab that wasn't in front were sometimes lost. Two episode forms came out blank, a Resend click did nothing and a Run jobs click did nothing. Each worked after a screenshot brought the tab forward. Because of one lost Resend, `Rem20Mtb` couldn't serve the "Resend, then day 5" probe, so I repeated it with `Rem23Mtb`.
   - The extension's network log showed **503** for `POST …/submit` both times a patient sent their answers. The page's own `performance` entries show **204**. The audit log shows "Patient sent their answers — DONE", and the episode is Ready for review. The 503 is a recording error in the tool, not a server response.

## Console and network errors

- No console errors or warnings on the patient pages, the episode pages or the jobs and messages pages.
- No unexpected 4xx or 5xx responses. The expected ones:
  - 401 and 403 from wrong dates and locks;
  - 404 from replaced links;
  - 400 from unreadable dates.
- The tool-reported 503 on submit is explained above: the browser itself recorded 204.

## GIFs

- `~/Downloads/S16-T16-shared-tries-r2.gif`: T-16 and the start of T-17. Wrong dates on the invite link (4, 3, then 2 tries left), then on the reminder link (1 try left, then locked), then the invite link locked too, then the staff banner "…every link sent to the patient for this episode is locked, and no reminders go."

STATUS: ALL_PASSED
