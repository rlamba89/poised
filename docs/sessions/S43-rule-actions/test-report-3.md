# S43 test report 3 (retest after verification 1)

**11 passed, 0 failed, 0 blocked.** This round re-checked verification 1's four fixes (build-notes.md, "Verification 1", items 1 to 4), then ran a quick pass of the main flow (T-02, T-03, T-04, T-05, T-08, T-09).

- **Environment:** commit `f769916`, with S43's changes still uncommitted (46 changed and new files). Tested 27 Oct 2026, 04:51–05:06 GMT, at http://localhost:3000, on the conductor's `make dev` (API :8080, evaluate :8090). Chrome through Claude in Chrome.
- **Window size:** the staff tab's viewport was 500 px wide and `resize_window` didn't change it. The patient's tab was 1400 px wide. This session doesn't change the patient's screens, and earlier rounds checked them at 390 px.
- **Captured messages:** I read the invite links from `GET /api/qa/messages`, the API behind `/qa/messages`.
- **Test data (all made up):**
  - HQ "MT S43 0336" version 4, published by Alex from v3. Its **High risk** rule also adds the task "Book anaesthetic review" for the **nurse** team, and the flag "MT43 long flag name …", exactly 100 characters. A new rule **"MT Airway"** (ASA III or more) adds the flag "Anaesthetic review".
  - Patients "Sam MT43 R3 0505 Two" (fixes 1, 2 and 4, plus T-03, T-04 and T-05), "Sam MT43 R3 0510 Backstop" (fix 3, plus T-08) and "Sam MT43 R3 0515 Low" (T-09). All were born 17/05/1980, with emails `mt43.r3.*@example.org`.
- **Database use (fix 3 only):** the rules page refuses a bad flag name, so the backstop can't be reached through the UI. To test it, I edited v4's `questionnaire_versions.rules` in SQL. I added three flags to MT Airway: one of 101 characters, one with a newline, and "MT43 backstop ok". I sent one HQ, then put the rules back exactly as published (MT Airway: flag "Anaesthetic review" only).

## Results

| Case | Result | Note |
| --- | --- | --- |
| Fix 1: same task title for two teams | Pass | The send added both tasks: "Book anaesthetic review" for the **Anaesthetist** and for the **Nurse** team, each "Added by rule: High risk on 27/10/2026, 04:58". Each has its own Automatic note. The episode is on **Anaesthetic review needed** (count 73 → 74), with "2 open" tasks. |
| Fix 1 (follow-on): completing only the anaesthetist's task | Pass | On "…0510 Backstop", I completed only the Anaesthetist task. The episode left Anaesthetic review needed and stayed on Open tasks, because the nurse task is still open. I checked this through the list's API (`?worklist=anaesthetic_review` and `?worklist=open_tasks`). |
| Fix 2: Undo greyed out while the anaesthetist's ASA is missing | Pass | Path: Ready for POA → anaesthetist ASA III saved → Ready for admission → On hold, then I cleared the anaesthetist's grade on the POA Summary. The page then shows "Undo: back to Ready for admission (add the anaesthetist's ASA grade first)", greyed out (`disabled`). Clicking it does nothing. The red requirement line shows, and the dropdown greys out "Ready for admission (add the anaesthetist's ASA grade first)". After saving ASA III again, the button reads "Undo: back to Ready for admission" and works: with a reason, the status went back to Ready for admission. |
| Fix 3: flag names over 100 characters or with a newline | Pass | **Rules page:** a 100-character flag name saves. At 101 it's refused: "Rule “High risk”: name the flag, on one line (100 characters at most)." **Backstop (SQL-injected rules):** the patient's send succeeded ("Thank you, Sam"). The 101-character and newline flags were left out, while "MT43 backstop ok" and the other flags were added, with their notes. No error. |
| Fix 4: a repeated flag name shows once | Pass | High risk and MT Airway both add "Anaesthetic review". The page header and the list row each show **one** "Anaesthetic review" badge (plus the long flag). The list's API sends `flags: ["Anaesthetic review", "MT43 long flag name …"]`. The Flags card lists both, one per rule, each with its own Clear. After clearing MT Airway's flag, the header still shows the badge, since High risk's is still standing. No console warnings, including no duplicate-key warning. |
| T-02 High-risk send (quick pass) | Pass | Red badge, Flags card, rule tasks, red requirement line and Automatic notes are all shown as the plan says. There's no Auto-triaged card. The episode is on Anaesthetic review needed as Ready for review. |
| T-03 Ready for admission refused | Pass | After Complete HQ review (→ Ready for POA), the red line is under the status control, and the POA Summary has its own red line. |
| T-04 The anaesthetist's grade meets it | Pass | After saving, the POA line turns grey: "The anaesthetist's ASA grade is added, as the rule “High risk” asks." On Back, Ready for admission was offered and the move worked. |
| T-05 Clear a flag | Pass | The reason box is focused when the form opens, and Enter clears. The card turns grey: "Cleared by Cara Clinician on 27/10/2026, 05:02: MT43 R3 airway checked". |
| T-08 Completing the rule's task | Pass | The task shows "Done", "Completed by Cara Clinician on 27/10/2026, 05:04", and still "Added by rule: High risk". |
| T-09 Low-risk patient | Pass | Smoke Yes, heart failure No → "Auto-triaged by the rule “Low risk”", Ready for admission. There are no flags, Flags card, rule task or red line, and no "Task added" or "Flag added" notes. |

## Failures in detail

None.

## Found while exploring

1. **Cosmetic: a long flag name squeezes its Clear button** on the Flags card at 500 px. With the 100-character flag, the badge takes the row and the button shows only "Cle". The header and list badges truncate with "…", which looks fine.
2. **Cosmetic: the Undo label is cut off at narrow widths.** At 500 px, the greyed "Undo: back to Ready for admission (add the anaesthetist's ASA grade first)" ends "…ASA grad". The red line underneath still explains why.
3. **Minor: the order of one send's flags and tasks changes from episode to episode.** All of a send's rows share the same `created_at`, so `ORDER BY created_at, id` falls back to a random UUID order.
   - Episode "…0505 Two" listed its flags MT Airway, High risk (long), High risk, with the nurse task first.
   - Episode "…0510 Backstop" listed MT Airway, High risk, High risk (long), MT Airway, with the anaesthetist task first.
   - The rules' order would read better. Nothing is wrong or missing.
4. **Minor (accessibility):** the two rule tasks' buttons both have the accessible name "Complete: Book anaesthetic review". A screen-reader user can't tell the team apart from the button alone.
5. **Not this session's (S20):** after an Undo back to Ready for admission, the next Undo offered was "Undo: back to Ready for POA". That is the change before the undone one, not "back to On hold". This may be intended (undo walks back through the history). I'm noting it only.

## Console and network errors not tied to a case

- None. No console errors or warnings on the episode page or the list, including with the repeated flag name. Every `/api/` request answered 200. A few `pending` requests were cancelled when I navigated away.

## GIFs

- `~/Downloads/S43-retest3-undo-blocked.gif`: fix 2. The anaesthetist's grade is cleared on the POA Summary, then the episode shows the greyed-out Undo, the greyed dropdown item and the red line. The grade is re-added, the Undo is enabled, and the undo back to Ready for admission works.

## Evaluate service

- I didn't stop it. One listener is on 127.0.0.1:8090: node PID 4410, the conductor's copy. I started no separate copy.

## Hand checks for Rahul

- None needed. (Optional, as before: the patient flow in a real 390×844 window.)
