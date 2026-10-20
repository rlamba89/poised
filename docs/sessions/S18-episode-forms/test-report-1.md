# S18 test report 1 (manual test, automatic run)

**Summary:** 21 passed, 0 failed, 1 blocked (R-1's CSV export: it needs a file download). R-4 (optional) not run.

**Environment**
- Commit `1036106` (S18 work uncommitted on top: 37 changed or new files in `git status`).
- Tested 20 Oct 2026, 14:36 to 15:10 CEST, on http://localhost:3000 (API :8080). The conductor ran `make dev`, freshly migrated and seeded.
- Browser: Claude in Chrome ("trk" profile). Staff were on `localhost:3000` at a 1000×623 viewport. Patients were on `tester.localhost:3000` (and, for T-10's second device, `localhost:3000`).
- **Phone size:** `resize_window` had no effect in this browser (the window stayed 1400×683, then 1000×623). So every patient case ran in a **390 px wide iframe** of the same origin, about 610 to 670 px tall. Cookies and reloads behave as in a real tab. Layout was checked at 390 px wide (`scrollWidth` = 390, no sideways scroll), but not at the full 844 px height.
- HQs used: **MT S03 1625 workflow v1** for T-01 to T-06. It has no clinical outputs. **Trust A demo HQ v1** was used from T-07 on, because it has outputs (Smoker, Non-smoker, Diabetic), so T-06's disclosure checks were also covered there. **Trust B demo HQ v1** was used for Bea.
- Test data: surnames `Form01K1436` … `Form15K1436` and mobiles `07700 900801`…`900816`. I read invite links through the captured-messages page's API (`GET /api/qa/messages?to=…`), which is the same data as `/qa/messages`.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Create an episode: shows its HQ | Pass | HQ NOT COMPLETE, "MT S03 1625 workflow v1", invite card, both notes. The patient record lists the episode. The list shows HQ + v1 + status, but no Hospital column for a single-hospital view (by design: the column shows only in "All hospitals"). |
| T-02 Patient answers half, leaves, returns (phone) | Pass | "Hello Sam", Hospital A1, About you + Your health only (no Nurse checks). Answers were kept after closing the tab, opening the link again and reloading. |
| T-03 Home: In progress, then Sent (phone) | Pass | Card: Hospital A1, HQ name, "For Sam Form01K1436", Procedure, IN PROGRESS / Continue, then SENT / View. View shows "Thank you, Sam". |
| T-04 Ready for review | Pass | Status READY FOR REVIEW. Note "HQ completed by the patient \| Patient on …". Resend invite is no longer offered. The API refuses it (409, the exact text in the plan). See the plan note below about the "sent by the patient" line. |
| T-05 Validate each set, complete review | Pass | URL `/sets/<id>`. Hint "Patient answered: Yes". "Validated by Cara Clinician on …" on all three sets. READY FOR POA. Note "HQ review completed by Cara Clinician". Sets read-only with the expected text. |
| T-06 POA Summary: both actors, change marked | Pass | Validated: "Do you smoke? Patient answered "Yes", now "No"" (T-01). On the Trust A demo HQ (T-07's episode): Validated shows **Non-smoker, CHANGED BY CLINICIAN**, and the Patient tab shows **Smoker** with no change mark. Same after a reload. |
| T-07 Fill in on the patient's behalf | Pass | `/patient-form`, blue box, patient sections only. About you was kept when coming back. "Sent for Sam Form07K1436". "Sent by Cara Clinician on the patient's behalf on …". Both sets show "Patient's answers entered by Cara Clinician". Note "HQ completed on the patient's behalf". Orange box on Validate. POA Patient tab: "Patient (entered by Cara Clinician) on …". |
| T-08 Patient + staff each fill a set | Pass | Only Your health says "entered by Cara Clinician". The patient's link shows "Thank you, Sam". |
| T-09 Stale save, clinician two tabs | Pass | Tab B: red "Not saved" box with the exact text, Reload, and Validate disabled. After Reload: tab A's "No", with no Validated badge. The episode shows About you "In progress", not validated. |
| T-10 Stale save, patient two devices (phone) | Pass | B shows "Your answers weren't saved / These answers were changed somewhere else… / Reload". After Reload, B shows A's "Yes". |
| T-11 Save after send refused (phone) | Pass | B: "These answers have already been sent, so they can't be changed here." The episode has A's answers only (revision 2; B's attempt is not stored). |
| T-12 Reminders stop after submit | Pass | 12a: invite only, no reminder notes, no badge. 12b: invite + day 2 + day 5 reminders ("Hello Sam, a reminder from Hospital A1: …"), "Automatic" notes, NOT RESPONDING badge + note. |
| T-13 Reminders stop once started | Pass (plan wording) | No reminder sent (invite only). A **Not responding** badge and note **do** appear on day 7. The plan says there should be none, but that matches the agreed S16 rule ("not responding … if the HQ isn't sent, started or not", decisions.md, 13 Oct). So the plan's expectation is wrong, not the app. |
| T-14 One patient, two trusts (phone) | Pass | Two cards, newest first: B1 (Hip replacement) above A1 (Knee replacement), both TO DO / Start. After answering one B1 question: B1 IN PROGRESS / Continue, A1 TO DO. The A1 episode has no answers (`/hq` answers = []). |
| T-15 Answers stay with their own episode | Pass | The second episode's HQ is empty. Home: the first is IN PROGRESS, the second TO DO. Each episode has its own HQ form, and only the first has answers. |
| T-16 Refused: other hospital, other trust | Pass | Val and Bea get "You don't have access to this hospital" on the episode page and the POA. The API gives 404 "Hospital not found." / "Episode not found." (also through Val's own A2 path). Signed out, the patient-form URL goes to the sign-in page. |
| T-17 Reload, deep links, Back | Pass | Episode, Validate, POA and patient-form pages all load after a reload and in a new tab. Episodes → episode → Review → Back → POA → Back all return correctly. Phone `/home` and `/home/episodes/<B1 id>` load after a reload. |
| R-1 Audit log | Pass (CSV blocked) | Ada's Audit log, filtered by patient, lists T-01 to T-07 actions: episode created and opened, answers opened and saved, patient link and DOB, "Patient sent their answers", "Saved answers for the patient", "Sent the HQ for the patient", review completed, the refused Resend (409). **Export CSV not clicked:** it downloads a file. |
| R-2 Existing record is linked | Pass | Typing T-01's details again linked to the same record (`203c7b94…`). The record lists both episodes. |
| R-3 Resend + lock | Pass | Resend: green "Invite sent again…", a second text with a new link. The old link says "This link doesn't work…" (the S14 wording). Five wrong dates: "4…1 tries left", then "This link is locked". The episode shows "The link is locked…". |
| R-4 Suspend Trust B | Not run | Optional. Skipped so as not to suspend a trust during an automatic run. |
| R-5 Authoring | Pass | Trust A demo HQ opens in the editor as "V1 · PUBLISHED" and "READ ONLY", with three Question Sets. |

## Failures in detail

None.

## Test-plan corrections for the builder (not app bugs)

1. **T-13:** expects "no Not responding badge" once the patient has started. The agreed S16 rule marks "not responding" on day 7 whenever the HQ isn't sent, **started or not** (decisions.md, 13 Oct, S16). The app follows that rule. The expectation should say the badge does appear.
2. **T-04:** expects the HQ card to say it was "sent by the patient (with a date and time)". The card has no such line for a patient's own submit (`EpisodePage.tsx` only shows "Sent by X on the patient's behalf…"). The time is in General notes ("HQ completed by the patient | Patient on …"). This was the same before S18 (the web app is unchanged).
3. **R-3:** "the old link says it's been replaced". The agreed S14 wording is "This link doesn't work. Please check it, or contact the hospital.", and that is what shows.
4. **T-01:** expects a Hospital column ("Hospital A1") in the list. It appears only in the "All hospitals" view, not for a single-hospital clinician.

## Found while exploring

- **Cross-patient and cross-hospital API checks hold after the reshape** (all 404, nothing leaked):
  - patient session GET/PUT on another patient's episode → "Episode not found.";
  - patient PUT to the clinician-only "Nurse checks" chapter → "Chapter not found.";
  - unknown chapter → 404;
  - bad JSON → 400;
  - wrong revision → 409 with the "changed somewhere else" text;
  - Val's PUT, GET and on-behalf submit on an A1 episode → 404.
- **Three simultaneous on-behalf submits** (Form12b, as Alex): one 204, two 409 "These answers have already been sent.", and exactly one "HQ completed on the patient's behalf" note. The Not responding badge cleared once it was sent.
- **Empty submits are accepted by the API.** This is the known pending item ("The API accepts Send with no answers", pending.md). While probing it I submitted **Form14K1436's A1 episode** and **Form12bK1436** with no answers, so both show "Ready for review" with empty or partial HQs. This is test data only, and the item is not reported as new.
- The episode JSON now carries `hqFormId` (new, harmless).
- Reminders carry a new link each time (`/i/…` differs per reminder), as before (S16).

## Console and network errors

- Staff tab: no console errors or warnings while tracking (T-07 onward).
- Network: every unexpected status was one I caused on purpose (the 404s and 409s above).
- The browser's network log showed **503** for `DELETE /api/qa/rate-limits`, `POST /api/logout` and `POST /api/dev/login`. Each of those actions worked in the UI: limits cleared, sign-out and sign-in worked. `curl` through :3000 returns 204 for the first two. This is probably an artefact of the extension's log, and I couldn't reproduce it.

## Hand checks for Rahul

- **R-1:** click **Export CSV** on the Audit log as Ada and confirm the file downloads. The tester doesn't download files without permission, and reading the CSV by script was blocked by the browser tool.
- **Optional:** look at one patient screen on a real phone-height window (390×844). The tester's window couldn't be resized, so phone cases ran in a 390 px wide frame.

## GIFs

- `/Users/rahul/Downloads/S18-T01-T07-happy-path.gif`: create episode, patient flow, validate, POA, on-behalf (50 frames).
- `/Users/rahul/Downloads/S18-T10-T11-stale-and-sent.gif`: two patient devices, "changed somewhere else", then "already sent".
