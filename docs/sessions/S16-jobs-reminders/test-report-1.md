# S16 test report 1: scheduled jobs, reminders and "not responding"

**Summary: 20 passed, 1 failed, 0 blocked.** The 15 plan cases and 5 regression checks all pass. The failure comes from exploring: a reminder adds another working link, so the S14 "five wrong dates of birth lock the link" limit now applies to each link separately (X-1, Minor, security-related).

- **Environment:** `661cdff` with S16's work not yet committed (the working tree is dirty, as the conductor expected). Tested on 13 Oct 2026, 20:54–21:30 BST, against `make dev` at http://localhost:3000 (API :8080) with the one-minute jobs timer on. The patient was on http://tester.localhost:3000.
- **Browser:** Claude in Chrome, run by the tester in automatic mode.
- **Phone size:** this Chrome window can't be resized: `resize_window` reports success, but the viewport stays 1280×657. So the patient pages were run in a **390 px-wide iframe** on tester.localhost, 390×640, the same width as the 390×844 the plan asks for. No page scrolled sideways (`scrollWidth` was 390 every time).
- **Test data:** made-up patients `Sam Rem01Mt` to `Rem17Mt`, born 17/05/1980, mobiles `+44 7700 9008xx`. I used 9008xx because some of the plan's 9006xx numbers already had messages from earlier runs. One email address: `rem01.mt@example.org`.
- **Database:** I used read-only `SELECT`s to confirm message counts, `invited_at` and `run_at`. The plan's checks were also done in the UI.
- **Shared data I changed, then put back:**
  - Hospital A2's settings: back to "Use the trust's setting" (`settings = {}`).
  - Hospital A2: suspended, then re-activated (`active`).
  - Timer clock offset: back to 0.
  - Jobs: none left open.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Day 2 reminder, text and email | Pass | Green "Ran 1 job as of 15/10/2026, 20:57: 1 reminder sent…". Two texts and two emails, worded exactly as the plan says. The reminder's link is new. Text and email share one link. |
| T-02 Nothing sent twice | Pass | Running +2 days again, double-clicking Run jobs and running at +4 days all give "Ran 0 jobs". Still 2 texts and 2 emails. Run jobs also submits with Enter. |
| T-03 Episode shows the reminder | Pass | "Day 2 reminder sent by text message and email. \| Automatic on…". The invite-card text is right and there's no badge. The invite and reminder went out in the same minute, so "Last sent" couldn't show the difference here; T-05 confirms it moves to the reminder's time. |
| T-04 Both links work (phone width) | Pass | The reminder link and the invite link each go Continue → date of birth → "Hello Sam." No sideways scroll. |
| T-05 Day 5 reminder; day 7 "Not responding" | Pass | Third text and email, with a new link. Nothing new at +7. Red **NOT RESPONDING** badge, both automatic notes, "Last sent" moved to 20:59. Running +7 again: "Ran 0 jobs". |
| T-06 Sending before day 5 stops reminders | Pass | Still 2 texts. Ready for review, no badge, no "Not responding" note. |
| T-07 Starting stops reminders; day 7 still flags | Pass | Only the invite at +2 and +5. At +7: badge and note, no message. |
| T-08 Staff answers count as started | Pass | Only the invite after +2 days. |
| T-09 Reminder due at night still goes | Pass | Run as of 16/10 22:30 → "1 reminder sent". Two texts, only one reminder. |
| T-10 Missed days: only the latest, once | Pass | +7 straight away: one reminder ("Day 5 reminder sent by text message."), then "Not responding", badge shown. |
| T-11 The timer sends by itself | Pass | After moving the clock 2 days 1 hour, the reminder arrived about 3 s later at the next tick, without Run jobs. The card said "Last run 21:07, as of 15/10/2026, 22:07". Back to real time worked and the button went grey. |
| T-12 Hospital's own days and wording | Pass | A2 set to days 1 / NR 3 / `{{hospital}} reminds {{firstName}}: {{link}}`. Nothing as of 14/10 21:05. "Hospital A2 reminds Sam: …/i/…" as of 14/10 21:15. As of 16/10 21:15: "Not responding: … 3 days after the invite." Settings restored afterwards. |
| T-13 No contacts, no follow-up | Pass | "No invite sent…". After +7: no reminder note, no badge. |
| T-14 Jobs page: invalid input, reload, links | Pass | Empty date: red "Enter a date and time." Days 91 and hours −1: red "Enter whole days and hours, 90 days at most.", and the clock stayed on real time (offset 0). Reload, the links both ways, Back and Forward all work. |
| T-15 Resend doesn't restart the days | Pass | Three texts: invite, resent invite, day 2 reminder. `invited_at` is still the first invite's time. The first invite link gives "This link doesn't work"; the resent link and the reminder link both work. |
| R-1 Invite on create | Pass | "Invite sent by text message and email."; Continue and date of birth open it. |
| R-2 Lock and Resend | Pass | Five wrong dates of birth → "This link is locked"; the episode shows "The link is locked…"; Resend sends a new link that works. |
| R-3 Patient home | Pass | `/home` at 390 px: "Hospital A1 · TO DO · Trust A demo HQ · For Sam Rem11Mt", contact details card, no sideways scroll. |
| R-4 Patient search | Pass | Searching "Rem07Mt" finds Sam Rem07Mt, 17 May 1980. |
| R-5 Audit log | Pass | As Ada: "Opened an episode" rows for the Rem episodes. There are no rows for the jobs themselves, as designed. |
| X-1 (exploring) The date-of-birth lock is per link, and reminders add links | **Fail (Minor)** | See below. |

## Failures

### X-1 (Minor, security-related): reminders multiply the patient's date-of-birth tries, and "The link is locked" is shown while other links still work

Since S14, five wrong dates of birth lock the link, and staff must press **Resend invite** to give the patient another try. Resend revokes every older link. S16's reminders instead add a new link and keep the older ones working. The lock is per link (`login_links.locked_at`, `failed_dob_attempts`). So an episode can now have up to three working links with the default days 2 and 5, or up to six if a hospital sets five reminder days. Each link has its own five tries.

**Steps** (signed in as Cara Clinician, Hospital A1; patient on tester.localhost):
1. Create `Sam Rem11Mt` with mobile `07700 900811`. Let the day 2 and day 5 reminders go: I used the timer and then **Now + 7 days**. There are now three working links: the invite, day 2 and day 5.
2. Open the newest (day 5) link and enter a wrong date of birth 5 times. You get "This link is locked".
3. Open the episode as Cara. The red box says "**The link is locked.** The date of birth was entered wrongly 5 times (…). Resend the invite to send the patient a new link."
4. Open the **original invite link**, `/i/jszuTIYUn-…`. It still shows "Confirm it's you". A wrong date of birth says "**You have 4 tries left**", a fresh count. The right date of birth (17 5 1980) signs the patient in and opens the questionnaire.

**Expected:** what S14 promised. Five wrong tries for the episode lock it until staff resend. If the per-link design is kept on purpose, the staff page shouldn't say the link is locked while other links still work. Alternatives: lock all of the episode's links together, or count failed tries per episode. This is Rahul's or the builder's call.

**Actual:** each live link gets 5 tries, so 15 with the defaults. The staff banner describes only the newest link (`GetEpisodeInvite`, `ORDER BY created_at DESC LIMIT 1`).

**More of the same:**
- On `Sam Rem15Mt`, the invite link was locked after five wrong tries. The next scheduled reminder then sent a fresh link that worked straight away.
- A locked patient therefore gets back in with no staff action. Staff still see "The link is locked … Resend the invite" until the patient uses the new link.

**Other details:** the audit log records each try (REFUSED 401 ×4, then 403). There were no console errors and no failed requests.

**GIF:** `~/Downloads/S16-X1-locked-link-other-links-open.gif`.

## Found while exploring

1. **Suspended hospital (the conductor's probe).** I suspended Hospital A2 in the platform-admin console (as Sam Superadmin) and checked `Sam Rem16Mt` (A2, invited 21:23).
   - **While suspended:** runs at +2 and +7 days sent nothing and set no flag. The job said it would look again an hour later (`run_at` +1 h). The patient's link showed "Not available at the moment".
   - **After re-activating:** a run as of +7 days +1 h sent **one** reminder ("Day 5") and set "Not responding" in the same run.
   - This is what I expected from the build notes ("nothing while suspended … catches up once active"). One thing for Rahul: on catch-up, the patient gets a reminder at the same moment the episode is flagged "Not responding".
   - Hospital A2 is active again.
2. **Time zone and clock change (the conductor's probe).**
   - I set A2's reminder days to `14` (not responding: 20) and invited `Sam Rem17Mt` at 13/10 21:26:59 BST. Day 14 is after the UK clocks go back on 25 Oct.
   - The job was scheduled for **27/10 21:26:59 GMT**, the same local time. Simply adding 14 × 24 hours would have given 20:26:59.
   - As of 27/10 21:00: nothing sent. As of 21:30: the reminder went.
   - Local times work as designed. Settings were restored and the job finished.
3. **Hospital and trust separation:**
   - every A2 reminder used "Hospital A2" (and A2's own template in T-12);
   - every A1 reminder used "Hospital A1";
   - the e2e B1 episodes' reminders say "Hospital B1".
   
   No episode got another hospital's wording.
4. **Resend after a reminder** also revokes the reminder's link: it gives "This link doesn't work". This matches "Resend replaces it".
5. **The badge clears on staff send:** filling in `Sam Rem07Mt` (flagged "Not responding") for the patient and pressing **Send the answers** → Ready for review, and the badge is gone.
6. **Double-clicking "Run jobs" sends two requests:** the button isn't disabled while a run is in progress. It does no harm (the second request finds nothing due, so nothing is sent twice). Cosmetic.
7. **Partly cleared "Run as of":** if only the day is cleared, Run jobs makes no request (probably the browser's own validation), and the previous green or red banner stays. Clearing every part gives the right red message. Cosmetic.
8. **Real time vs "as of" time:**
   - captured messages and the automatic General-notes entries show the real time, not the "as of" time;
   - the episode's `not_responding_at` holds the "as of" time.
   
   Fine for a test helper. A tester might expect "Automatic on 18/10…" for a day 5 reminder. Cosmetic, probably as designed.
9. **Outside S16:** each staff page view writes two audit rows ("Opened an episode" ×2 and "Opened the answers" ×2). This looks like a double fetch on the page and is older than S16.
10. **Test-tool behaviour, not app bugs:** the tester's tabs run in the background. A click or typing done right after a navigation was sometimes lost, and a Mantine modal stayed half-faded until a screenshot brought the tab forward. Re-doing the action after a screenshot always worked. API responses took under 15 ms.

## Console and network errors

No console errors or warnings on the jobs, messages, episode, settings or patient pages. No unexpected 4xx or 5xx responses. I sent direct API probes to check the validation, and these 400s were expected:
- `PUT /api/qa/jobs/clock` with −60, 129601, 1.5, `{}` and non-JSON → 400.
- `POST /api/qa/jobs/run` with `bad`, `{}` and a time with no zone → 400, "Give the date and time to run the jobs as of, e.g. 2026-10-15T09:30:00Z."

## GIFs

- `~/Downloads/S16-T01.gif`: the main happy path. Run as of +2 days, then the reminder text and email in the captured messages.
- `~/Downloads/S16-X1-locked-link-other-links-open.gif`: X-1. The staff page says "The link is locked", while the original invite link still accepts dates of birth ("4 tries left") and signs the patient in.

STATUS: FAILURES 1
