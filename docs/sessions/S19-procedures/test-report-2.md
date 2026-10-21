# S19 test report 2 (retest)

Manual tester, automatic run (`/run-sessions`).

**Summary: 16 passed, 1 failed, 0 blocked**, plus one hand check for Rahul. This covers the 12 plan cases re-run, the 5 regression checks, and the 2 fixes from exploring that I checked again.

**Environment:**
- HEAD `b534af3`, with S19 (including the F1 fix) uncommitted in the working tree, as expected.
- 20 Oct 2026, 19:27–19:45 CEST.
- http://localhost:3000, with `make dev` run by the conductor.
- Chrome, Rahul's trk profile.
- Desktop: viewport 1014×731 at first, then 1400×673.
- Phone: the narrowest Chrome window allowed is still 500 px wide, so the phone case ran at 500×667.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-13.1 (F1) | Pass | New episode with the POA date typed as `28/10/yyyy`, then Create: the page shows "Enter the POA appointment date as a date.", sends no request and stays on the page. |
| T-13.2 (F1) | Pass | Planned date typed as `31/mm/yyyy`: "Enter the planned procedure date as a date."; no request is sent. |
| T-13.3 (F1) | Pass | Edit procedure and dates, planned date changed to `03/11/yyyy`, then Save: the same message, the form stays open, no PATCH is sent and the details are unchanged. |
| T-13.4 | Pass | No dates at all is fine (see R-1). |
| T-09 | Pass | New patient Pat ProcedureMTR1930. Total hip replacement (HIP-01) chose Demo pre-op HQ v1. Created with 03/11/2026 and 28/10/2026. The audit log shows a single `episode.create`, so the refused attempts created nothing. |
| T-14 | Pass | Knee arthroscopy, 10/11/2026, POA date cleared. Three General notes, each by Cara Clinician. Same after a reload. Cancel (with a half-typed date in the form) changed nothing. |
| T-15 | Pass | The list's Procedure, Planned date and POA date columns are right for both test episodes (—, or Knee arthroscopy, 10/11/2026, —). The POA Summary shows Procedure and Planned procedure date on both tabs. |
| T-16 | Pass | After a complete review, "Edit procedure and dates" is gone, the values still show, and the status select reads Ready for POA. |
| T-17 | Pass (at 500 px) | Patient home: one card shows "Procedure: Knee arthroscopy". The card for the episode with no procedure has no Procedure line. No dates are shown. `scrollWidth` = 500, so nothing scrolls sideways. 390 px is a hand check. |
| T-02 | Pass | "MT R2 1940 hip revision" (HIP-R2, Orthopaedics, Demo pre-op HQ) was added by **pressing Enter** in the form: "…is added." |
| T-04 | Pass | Edit on a lower row (scrolled down): **the form scrolls into view.** Changed to HIP-R3 and Trauma and orthopaedics, then **Enter** saved it: "…is saved." Still the same after a reload. |
| T-05 | Pass | Retire: "MT R2 1940 hip revision is retired: it isn't offered for new episodes."; the row moves to the bottom marked RETIRED. Bring back: "…is offered again." |
| T-06 | **Fail** (step 3 only) | Steps 1 and 4 pass, and pressing Enter twice sends only one POST (409). Spaces-only names get "Enter the procedure's name." But an **empty** name gets the browser's own bubble instead. See F1. |
| R-1 | Pass | Existing patient: search, **Choose**, no procedure and no dates. Created, and the details show — for the procedure and both dates. |
| R-2 | Pass | At 500 px the patient filled in and sent Demo pre-op HQ, and the episode went to Ready for review. Cara validated all 3 sets and completed the review, and it went to Ready for POA. The POA Summary has both tabs. |
| R-3 | Pass | Ada saved Hospital A1's own reminder days, "4, 8". The database showed `{"reminders.days": [4, 8]}`, and the page showed them after a reload. Put back to the trust's setting (`{}`). |
| R-4 | Pass | The audit log has this run's rows: "Added a procedure" and "Changed a procedure" by Alex, both DONE and REFUSED (409/400), plus Cara's episode actions and the patient's actions. |
| R-5 | Pass | Ada: Questionnaires, Episodes, Procedures, Staff, Settings, Audit. Alex: Questionnaires, Episodes, Procedures. Cara: Questionnaires, Episodes. |

## Failures

### F1 (Minor): an empty procedure name shows the browser's bubble, not "Enter the procedure's name." (T-06 step 3)

- **Who and where:** Alex Author, `/o/6ef150fb…/h/…000a/procedures`.
- **Steps:**
  1. Click **Add procedure**.
  2. Leave Name empty and click **Add procedure** (or press Enter in another field).
- **Expected (plan):** a red message: "Enter the procedure's name."
- **Actual:**
  - Chrome's native validation bubble, "Please fill in this field.", appears on Name, and nothing is sent.
  - The Name input now has the `required` attribute inside a real `<form>`. This is a side effect of the "Enter sends the form" fix.
  - If a red message is already showing (e.g. the duplicate-name message from step 1), it stays on screen next to the bubble, which is confusing.
  - A spaces-only name still gets the app's own "Enter the procedure's name." (API 400).
- **Why it matters:** the wording differs from the rest of the app. The New episode page has no `<form>` and shows its own messages. The bubble's text is in the browser's language, not the deployment's locale.
- **Console and network:** no errors; no request is sent.
- **GIF:** `S19-T06-empty-name.gif`.

## Hand checks for Rahul

- **T-17 at 390×844:** Chrome won't make the window narrower than 500 px, so the patient home was checked at 500×667. Please check it at 390 px in device emulation.

## Found while exploring

1. **The success message is out of sight after a Retire or Bring back on a lower row (Minor).**
   - The message ("…is retired: …") is at the top of the page, but the page stays scrolled where it was (`scrollY` 350). All you see is the row jumping to the bottom of the list.
   - The text is right. This is the same kind of problem as the Edit-scroll fix, but for these messages.
2. **Enter doesn't send the episode's Edit procedure and dates form.** The fix only claims this for the procedure form, so this isn't a failure. It's mentioned for consistency.
3. **Tooling note, not an app problem:** in the trk profile, 1Password attaches to the procedure **Name** field ("Unlock 1Password"). Typing into that field with the extension made the browser tools lose the tab, so I set Name with `form_input` and pressed Enter in the Code field. Everything else was typed or clicked normally.

## Console and network errors not tied to a case

- There were no console errors on the pages checked (New episode, episode page, POA Summary, Procedures).
- The extension's network log again showed the Edit procedure and dates `PATCH → 503`. The database audit log has `episode.update … 204` at the same moment, and the change was saved and shown, so this is an artefact of the extension's log, as in report 1.

## GIFs

Saved in `/Users/rahul/Downloads/`:

- `S19-T13-retest.gif`: the F1 fix. A partial POA date and a partial planned date are refused on New episode, the episode is created with full dates, and a partial date is refused in Edit procedure and dates.
- `S19-retest-procedures.gif`: the main happy path for procedures. Add and edit with Enter, the Edit form scrolling into view, retire and bring back.
- `S19-T06-empty-name.gif`: failure F1.

## Test data created

- Patient Pat ProcedureMTR1930 (made up), with 2 episodes at Hospital A1:
  - `ee1ca748…`: Knee arthroscopy, planned 10/11/2026, review complete (Ready for POA);
  - `cf350483…`: no procedure, no dates.
- Procedure "MT R2 1940 hip revision" (HIP-R3, Trauma and orthopaedics), active.
- Hospital A1's settings are back to `{}`.
