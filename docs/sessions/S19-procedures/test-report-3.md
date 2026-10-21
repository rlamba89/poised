# S19 test report 3 (retest)

Manual tester, automatic run (`/run-sessions`).

**Summary: 7 passed, 1 failed, 8 blocked.**
- The trk Chrome profile's extension kept disconnecting. After about 08:00 it dropped for good (no browsers listed for over 10 minutes), so the regression cases and the episode-side retest could not be run. They are listed as Blocked below.
- The fix from report 2 (F1, the empty procedure name) **passes**.
- One of the two explored fixes fails: **Retire on a lower row still leaves the success message out of sight.**

**Environment:**
- HEAD `b534af3`, with S19 uncommitted in the working tree, as expected.
- 20 Oct 2026 23:45 BST to 21 Oct 08:15 BST. There were long pauses while the browser extension was disconnected.
- http://localhost:3000, with `make dev` run by the conductor. The API was still healthy at the end.
- Chrome, Rahul's trk profile, 1400×729.
- The database was **not** freshly seeded. It still holds data from earlier runs, such as report 2's "MT R2 1940 hip revision", and Test hip revision and Test grommets, both retired.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Alex at Hospital A1: Trust A's list, with retired ones last and marked RETIRED. No Trust B procedure is listed. |
| T-06.1 | Pass | `total HIP replacement` gives "The trust already has a procedure with this name (retired ones included)." (POST 409). |
| T-06.2 | Pass | The retired name `Test hip revision` gives the same message. |
| **T-06.3 (report 2's F1)** | **Pass** | With Name empty, clicking **Add procedure** shows no browser bubble. The app's red "Enter the procedure's name." **replaces** the duplicate message (POST 400). Pressing Enter in Code does the same. |
| T-06.4 | Pass | Renaming "MT R2 1940 hip revision" to `Knee arthroscopy` and saving gives the duplicate message (PATCH 409), and the database name is unchanged. Emptying Name in the edit form and pressing Enter in Code gives "Enter the procedure's name.", which replaces it (PATCH 400). Cancel closes the form. |
| T-06.5 | Pass | Default HQ offers 1323 HQs. That matches the database's count of Trust A questionnaires with a published version. None of the 246 draft-only questionnaires or Trust B's appear. |
| T-02 | Pass | "MT R3 0440 hip revision" (HIP-R3X, Orthopaedics, Demo pre-op HQ) was added by pressing Enter in Code: "…is added." |
| T-05 (Bring back scroll) | Pass | Bring back on a row at the bottom of the list (scrollY 1651) shows "MT R3 0440 hip revision is offered again." and scrolls smoothly to the top, where the message is. Checked twice. |
| **T-05 (Retire scroll, report 2's exploring note 1)** | **Fail** | The retire works and its message is set, but the page doesn't scroll up. See F1. |
| T-04 (Edit form scrolls into view; Enter saves) | Blocked | The extension disconnected just as Edit was clicked. Report 2 passed it. |
| Enter in **Edit procedure and dates** (report 2's exploring note 2) | Blocked | Browser disconnected. |
| R-1 | Blocked | Browser disconnected. |
| R-2 | Blocked | Browser disconnected. |
| R-3 | Blocked | Browser disconnected. |
| R-4 | Blocked (DB only) | The audit log has this run's rows, by Alex Author: `procedure.create` 201/409/400 and `procedure.update` 204/409/400. They weren't checked in the Audit page. |
| R-5 | Blocked | Browser disconnected. Partly seen: Alex's header shows Questionnaires, Episodes and Procedures. |
| T-17 at 390×844 | Blocked | Hand check for Rahul, carried over from report 2. |

## Failures

### F1 (Minor): after Retire on a lower row, the page doesn't scroll to the message

- **Who and where:** Alex Author, `/o/6ef150fb…/h/…000a/procedures`.
- **Steps:**
  1. Scroll down until an active row near the bottom of the active list, e.g. "MT R3 0440 hip revision", is in view (scrollY about 490).
  2. Click **Retire** on that row.
- **Expected:** "MT R3 0440 hip revision is retired: it isn't offered for new episodes." comes into view, because the page scrolls to the top (the fix after report 2).
- **Actual:**
  - The procedure is retired (PATCH 204) and moves to the retired section, and the message is rendered at the top.
  - The page stays where it was. In a 100 ms trace of `scrollY`, it starts to move (488 → 483, or 500 → 429) and then stops or snaps back to about 500.
  - It happened all 3 times: twice clicking the element and once clicking by screen position after a fresh reload.
- **Contrast:** **Bring back** from the bottom of the list does scroll smoothly to 0, also every time. Running `window.scrollTo({top: 0, behavior: "smooth"})` by hand on the same page also scrolls to 0. So smooth scrolling itself works, and something about the Retire case cancels it.
  - A guess, not checked: the list reloads and re-renders while the smooth scroll is running (`load()` runs before `scrollTo`), or scroll anchoring holds the view on the clicked row as it moves.
- **Console and network:** no errors. PATCH 204.
- **GIF:** `S19-retire-scroll.gif`.

## Hand checks for Rahul

- **T-17 at 390×844:** check the patient home at 390 px in device emulation (carried over from report 2).
- The cases blocked by the extension dropping can be run again in the next round: T-04, Enter in Edit procedure and dates, and R-1 to R-5.

## Found while exploring

- Nothing new. The time went on the retest and on waiting for the extension to reconnect.

## Console and network errors not tied to a case

- None seen on the Procedures page. The only 4xx responses were the 409 and 400 that the T-06 steps expect.

## GIFs

Saved in `/Users/rahul/Downloads/`:
- `S19-T06-retest3.gif`: T-06 steps 1 to 4, including the F1 fix (an empty name gives the app's message, which replaces the duplicate one).
- `S19-retire-scroll.gif`: the main procedures happy path (add with Enter, Bring back scrolling up) and failure F1 (Retire not scrolling).

## Test data created

- Procedure "MT R3 0440 hip revision" (HIP-R3X, Orthopaedics, Demo pre-op HQ, every hospital). It is now **retired**.
- "MT R2 1940 hip revision" is unchanged. Both rename attempts were refused.
