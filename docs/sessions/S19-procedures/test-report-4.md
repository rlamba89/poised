# S19 test report 4 (retest)

Manual tester, automatic run (`/run-sessions`), round 4.

**Summary: 9 passed, 0 failed, 1 blocked (a hand check).**
- Report 3's failure F1 **passes**: Retire on a lower row now jumps straight to the top, where the message is, and stays there.
- The 8 cases that report 3 couldn't run because the browser disconnected all pass: T-04, Enter in Edit procedure and dates, and R-1 to R-5. R-4 was checked in the Audit page.
- T-17 at 390×844 is still a hand check for Rahul. The window couldn't be resized: it stayed 2560 px wide.

**Environment:**
- HEAD `b534af3`, with S19 uncommitted in the working tree, as expected.
- 21 Oct 2026, 11:00 to 11:25 BST.
- http://localhost:3000, with `make dev` run by the conductor. API health was ok.
- Chrome, Rahul's trk profile (deviceId bac77f71…), viewport 2560×1318.
- The database still had data from earlier runs, such as E2E procedures and HQs, "MT R2 1940 hip revision" and "MT R3 0440 hip revision". All test data below was created fresh.

## Results

| Case | Result | Note |
| --- | --- | --- |
| **F1 retest: Retire on a lower row (report 3)** | **Pass** | From scrollY 500 and from scrollY 400, Retire on "MT R4 1100 hip revision" goes to 0 within 100–200 ms and stays at 0 for the next 4–8 s, after the list reloads. The message "MT R4 1100 hip revision is retired: it isn't offered for new episodes." is in view. Tried twice. |
| T-05: Bring back from the bottom of the list | Pass | From scrollY 1777 it goes straight to 0 and shows "…is offered again." |
| T-02: add (setup for the cases below) | Pass | "MT R4 1100 hip revision is added." |
| T-04: edit a procedure | Pass | Edit on the bottom (retired) row scrolls the edit form into view, filled with the procedure's values. Code was changed to HIP-R4B, Specialty to Trauma and orthopaedics and Default HQ to Demo pre-op HQ. **Enter in Code saves it:** "MT R4 1100 hip revision is saved." After a reload the row still shows HIP-R4B, Trauma and orthopaedics and Demo pre-op HQ. **Cancel** (after typing a specialty on MT R2 1940) closes the form and changes nothing. **Save changes** (MT R2 1940, code to HIP-R2) gives "…is saved." |
| Enter in **Edit procedure and dates** | Pass | Cara, on episode Pat Mtrfourxb: Planned date typed as 10/11/2026, then Enter. The form closes, and the details show 10/11/2026. General notes: "Planned procedure date set to 2026-11-10" by Cara Clinician. |
| R-1 (S13): episode for an existing patient | Pass | Searched for "Mtrfourxb", clicked **Choose**, picked HQ Demo pre-op HQ v1, with no procedure and no dates. Created: the details show — for Procedure, Planned procedure date and POA appointment date. |
| R-2 (S14, S18): patient sends, clinician completes | Pass | The patient opened the invite link, confirmed the date of birth (2/4/1975), filled About you and Your health, and sent them; the page showed "Thank you, Pat". The episode became **Ready for review**. Cara validated all 3 sets (Blood pressure 120/80) and clicked **Complete HQ review**: the status became Ready for POA, and **Edit procedure and dates** disappeared (T-16 behaviour). The POA Summary shows both tabs, with Procedure Total hip replacement, 10/11/2026 and 28/10/2026. |
| R-3 (S10): Settings | Pass | As Ada, Settings for Hospital A1: reminder days, untick "Use the trust's setting", type `3, 6`, save. The page said "Hospital A1's settings were saved." After a reload, Hospital A1 shows 3, 6 (in the database: `{"reminders.days": [3, 6]}`). Restored afterwards: the box re-ticked and saved, and the settings are back to `{}`. |
| R-4 (S11): Audit page | Pass | As Ada, Audit, Who = Alex Author: this run's "Added a procedure" (11:01) and five "Changed a procedure" rows (11:01–11:03), all DONE, for the whole trust. Earlier rows show REFUSED (400) / (409) for the invalid adds. The episode actions by Cara are listed as before: Opened an episode, Opened the answers, Saved the clinician's answers, Completed the HQ review. |
| R-5 (S08/S09): header links | Pass | Ada: Questionnaires, Episodes, Procedures, Staff, Settings, Audit. Alex: Questionnaires, Episodes, Procedures. Cara: Questionnaires, Episodes only. |
| T-17 at 390×844 | Blocked: hand check for Rahul | `resize_window` reported success, but the page stayed 2560 px wide. At desktop width the patient home shows "Procedure: Total hip replacement" on that episode's card, no procedure line on the card without one, and no dates. |

## Failures

None.

## Hand checks for Rahul

- **T-17 at 390×844:** check the patient home in device emulation at 390 px: the procedure line, and that nothing scrolls sideways. This is carried over from reports 2 and 3, because the trk window can't be resized from the extension.

## Found while exploring

1. **(Cosmetic) The top of the edit form sits under the fixed header.** Click Edit on a lower row of Procedures. The form scrolls into view, but its heading "Edit MT R4 1100 hip revision" ends up 17 px from the top of the window, under the 56 px fixed header. The Name field (top at 80 px) and everything below it are visible, so it works. The heading just can't be seen until you scroll up a little.
2. **(Tool note, not a bug) 1Password in trk.** Clicking New episode's **First name** field and typing made every extension call fail with "Cannot access a chrome-extension:// URL of different extension", until the tab was reloaded. Filling the text fields with the form tool avoided it.

## Console and network errors not tied to a case

- No console errors on the Procedures, Episodes, Settings or Audit pages.
- The extension's network log shows `POST /api/logout` and `POST /api/dev/login` as **503**. Signing out and in still worked every time.
  - The same requests return 204 / 404 from curl, and `fetch('/api/logout')` from the page returns 204.
  - So this looks like the extension's label for a request cut off when the page navigates away, not a server error. It's noted here only in case it shows up elsewhere.

## GIFs

Saved in Rahul's Chrome downloads folder (normally `/Users/rahul/Downloads/`):
- `S19-R4-procedures.gif`: the procedures happy path. It covers adding a procedure, Retire on a lower row jumping to the top (the F1 retest), Bring back from the bottom, Edit on the bottom row, saving with Enter, a reload, Cancel and Save changes.

## Test data created

- Procedure **"MT R4 1100 hip revision"** (HIP-R4B, Trauma and orthopaedics, every hospital, Demo pre-op HQ). It is **retired**.
- "MT R2 1940 hip revision": its code is now **HIP-R2** (was HIP-R3).
- Patient **Pat Mtrfourxb** (born 2/4/1975, 07700 900555, pat.s19r4@example.test), with two Hospital A1 episodes:
  - `268f2fd8…`: Total hip replacement, 10/11/2026 and 28/10/2026. Its HQ was sent, validated and completed, and it is now **Ready for POA**.
  - `ae0dc2be…`: no procedure and no dates, **HQ not complete**.
- Hospital A1's settings were changed for R-3 and then restored to the trust's settings (`{}`).
