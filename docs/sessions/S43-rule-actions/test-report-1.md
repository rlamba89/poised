# S43 test report 1

**16 passed, 1 failed, 0 blocked** (17 cases: T-01 to T-12, R-01 to R-05).

- **Environment:** commit `f769916` with S43's changes not yet committed (the working tree isn't clean: the build's 39 changed and new files). Tested 27 Oct 2026, 03:35–04:05 GMT, at http://localhost:3000 (the conductor's `make dev`: API :8080, evaluate :8090). Chrome through Claude in Chrome.
- **Phone size:** the window couldn't be resized: `resize_window` reported success, but the viewport stayed 1400 px wide. So I ran every patient journey inside a 390×700 iframe of the real invite link on the same origin. That gives the same CSS width as a 390 px phone. `scrollWidth` equalled `clientWidth` (390), so there was no sideways scroll.
- **Test data (all made up):**
  - HQ "MT S43 0336" (v1, then v2).
  - HQ "MT S43 R04 0400".
  - Episodes for patients "Sam MT43 0340 High" (T-02 to T-05, T-08), "… 0348 Tabs" (T-06, T-07), "… 0351 Low" (T-09, R-01 to R-03), "… 0354 Arch" (T-11, R-05), "… 0402 Suggest" (T-10), "… 0405 NoEval" (T-12) and "… 0408 Staff" (exploring). All were born 17/05/1980, with emails `mt43.*@example.org`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Build and publish the HQ | Pass | "About you" (patient) has the 2 required Yes/No questions. Smoker 77176002 filled in ASA II by itself, and I chose III for I50.9, with notes and categories. "Nurse checks" is the clinician set. The Rules page shows Low risk and High risk with the exact High risk wording. It published. |
| T-02 High-risk patient sends on a phone | Pass | The patient pages fit 390 px and say nothing about rules, tasks or flags. The episode is on Anaesthetic review needed (59 → 60), not on Auto-triaged. The row has the red badge and Ready for review. On the episode page: the badge, a Flags card ("Added by the rule “High risk” on 27/10/2026, 03:42", Clear), no Auto-triaged card, and the task (Anaesthetist, "Added by rule: High risk on …", Complete). The red requirement line is word for word. All 3 Automatic notes are there. No console errors. |
| T-03 Ready for admission refused | Pass | After the review, the dropdown greys out "Ready for admission (add the anaesthetist's ASA grade first)", and clicking it does nothing. On hold works. The nurse's ASA III changes nothing, and the POA Summary's red line shows. I checked the API too: PATCH status → `409 "Add the anaesthetist's ASA grade first."`, and `moves` carries `blocked`. |
| T-04 Anaesthetist's grade meets it | Pass | After Save, the line turns grey: "The anaesthetist's ASA grade is added, as the rule “High risk” asks." On Back, the red line has gone and Ready for admission is offered normally. The move works and is kept after a reload. |
| T-05 Clear the flag with a reason | Pass | Empty → "Say why the flag is cleared." Spaces only → the same message. "Reviewed by Dr Example" + Enter clears it: no badge, and the card shows the grey flag with "Cleared by Cara Clinician on 27/10/2026, 03:46: Reviewed by Dr Example" and no Clear. The note is by Cara. All kept after a reload. The list row has no badge, and the episode stays on Anaesthetic review needed while its task is open. Back closes the form without clearing. |
| T-06 Other hospital or trust can't see or clear it | Pass, with a note | Val and Bea get **"You don't have access to this hospital"**, not the plan's "Episode not found.", because the copied URL names Hospital A1. That is the existing S08 hospital guard. No patient details or flag are shown. With each user's own hospital id in the URL, the page says "Episode not found." (API 404). The episode isn't in Val's or Bea's lists. Back as Cara, the flag is still standing. Val's clear POST also gets 404 with her hospital id or with A1's. |
| T-07 Second tab's Clear is told the episode changed | **Fail (Minor)** | The stale message, Reload and the kept reason all work. But after **Reload**, the Clear form stays open under the now-cleared flag. See failure 1. |
| T-08 Completing the rule's task | Pass | The task shows Done, "Completed by Cara Clinician on 27/10/2026, 03:50" and still "Added by rule: High risk". The episode left Anaesthetic review needed. The status (Ready for admission) and the cleared flag are unchanged. |
| T-09 Low-risk patient | Pass | The "Auto-triaged by the rule “Low risk”" card shows, the episode is Ready for admission and on the Auto-triaged tab. There is no flag, Flags card, rule task, red line, or "Task added…"/"Flag added…" note. |
| T-10 Suggested status | Pass | v2's Low risk reads "suggest Ready for POA", and the publish dialog says "This version doesn't auto-triage anyone." The episode is Ready for review with no Auto-triaged card. The blue line "The HQ's rules suggest: Ready for POA (the rule “Low risk”)" shows and is kept after a reload. Nothing else moved. |
| T-11 Archived episode's flag can't be cleared | Pass | Sue archived it. The badge and the Flags card show, with no Clear. The API clear → 409 "This episode is archived, so it can't be changed…". I didn't try a cancelled episode. |
| T-12 Evaluate service down | Pass | I stopped PID 55370 (the `make dev` copy). The episode on v2 is Ready for review, and the notes say "The automatic checks didn't run. Review this HQ by hand." There is no flag, rule task or requirement line. I restarted the service with `make evaluate` (see below), and it has one listener on :8090. |
| R-01 Task by hand | Pass | "MT43 call patient", Nurse team: "Added by Cara Clinician on …". Complete, then Re-open ("Re-opened by Cara Clinician on …"). |
| R-02 No rule requirement → Ready for admission | Pass | Sue re-opened the T-09 episode, validated both sets, completed the review (→ Ready for POA), then moved it to Ready for admission. Nothing was greyed out. |
| R-03 Checked on the auto-triaged episode | Pass | "Checked by Cara Clinician on …". It left the Auto-triaged tab. |
| R-04 Rules page of a new HQ | Pass | The new HQ shows the two default rules. Renaming High risk to "High risk MT" saved and was kept after a reload. |
| R-05 Show archived | Pass | The T-11 episode is listed as Archived, with its red Anaesthetic review badge. Without Show archived it isn't listed. |

## Failures in detail

### 1. Minor: after Reload on a stale Clear, the form stays open under a cleared flag

- **Signed in as:** Cara Clinician (Hospital A1).
- **URL:** `/o/…fa/h/…0a/episodes/98119e88-25a9-405f-8936-92a52306da75`
- **Steps:**
  1. Open the same high-risk episode in two tabs.
  2. In tab 1, Clear the flag with "Tab one reason".
  3. In tab 2 (not reloaded), click Clear, type "Tab two reason" and press Enter.
  4. Tab 2 shows the stale message. Click **Reload**.
- **Expected:** after Reload, the flag shows as cleared by Cara (tab 1's reason), with no Clear button, as the plan says.
- **Actual:**
  - The flag card is updated correctly: grey, "Cleared by Cara Clinician on 27/10/2026, 03:49: Tab one reason".
  - But the "Why is it cleared?" form is still open below it, holding "Tab two reason", with a live **Clear flag** button.
  - Clicking it shows "This flag is already cleared.", so the server refuses correctly and no data is wrong.
  - You have to click Back to close the form.
- **Suggested fix:** close the form when the flag it belongs to is no longer standing after a reload.
- **Errors:** no console errors. The only 409 was the expected stale one, then "already cleared".
- **GIF:** `~/Downloads/S43-T07.gif`

## Found while exploring

1. **Staff "Fill in for the patient" (the second send route) works.**
   - On v2, with heart failure Yes, it added the task, the flag and the requirement, with the same three Automatic notes.
   - Double-clicking the confirm "Send the answers" made one send only: one task and one flag (checked in the API).
   - This also confirmed that the restarted evaluate service works.
2. **Flag reason handling is right.**
   - Leading and trailing spaces are trimmed. "Reviewed 👍 by Dr Émile" is stored and shown correctly.
   - The input has `maxlength=1000`. The API refuses 1,001 characters with "The reason is too long (1,000 characters at most)."
   - No revision → 400. An unknown flag id → 404 "Flag not found.".
3. **Cosmetic or UX: the Clear form's reason box isn't focused when it opens.**
   - After clicking Clear, focus stays on the page, so typing goes nowhere.
   - Spaces scroll the page instead. I hit this once: the typed reason was lost and the page jumped.
   - Other inline forms might behave the same. Focusing the box on open would help keyboard users.
4. **Cosmetic, not this session's:** the General notes say "Nurse ASA grade set to 3" and "Anaesthetist ASA grade set to 3", while the page says "ASA III" everywhere else.
5. **Plan wording (T-06):** someone outside Hospital A1 who opens an A1 URL sees "You don't have access to this hospital", not "Episode not found." (the S08 guard; earlier reports note the same). The plan's expected text could say so. It isn't a leak.
6. **Phone resize:** Chrome ignored `resize_window`, so I used the iframe harness described above. Someone may want to hand-check the patient flow on a real phone-sized window, but layout at 390 px CSS width was fine.

## Console and network errors not tied to a case

- None. The episode page made each GET twice on load (the React dev StrictMode pattern), all 200.

## GIFs

- `~/Downloads/S43-T02-T05-happy-path.gif`: the high-risk happy path, from the lists and the episode page through validation, the blocked dropdown, the POA Summary grades and clearing the flag.
- `~/Downloads/S43-T07.gif`: failure 1.

## Evaluate service (T-12)

- I stopped the `make dev` copy (node PID 55370). Its `npm run evaluate` parent (55268) exited with it. The API and web kept running.
- I restarted it with `make evaluate` in the background. Exactly one listener is on 127.0.0.1:8090: node **71776**.
- The separately started copy's PIDs: zsh 71693 → make **71704** → npm **71710** → npm (serve) **71736** → sh **71770** → node **71776**.

## Hand checks for Rahul

- None needed. (Optional: the patient flow in a real 390×844 window, since the window couldn't be resized here.)
