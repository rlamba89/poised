# S42 test report 2 (retest): auto-triage and the Auto-triaged worklist

**Summary: 17 passed, 0 failed, 0 blocked** (12 cases + 5 regression checks). Test report 1's F-1, E-1 and E-2 are fixed.

- **Environment:** HEAD `cb8857f` with the S42 work uncommitted (dirty tree). 27 Oct 2026, 02:28–02:44 GMT.
- **App:** the conductor's `make dev` (web http://localhost:3000, API :8080, evaluate service 127.0.0.1:8090). Rows from earlier runs were still in the database (e.g. round 1's MT S42 episodes on the worklists). They made no difference: every check below uses this round's own data.
- **Browser:** Chrome (Claude in Chrome) for every staff screen. The window stayed 500 px wide (`resize_window` had no effect), so staff screens were seen at a narrow width. They all read fine.
- **Tester (automatic run):** manual-test agent.

## How I tested (and where I deviated)

- **Retest scope:**
  - every case touched by report 1's fixes: T-04 (F-1), T-07 (E-1), and T-02 and T-15 (E-2);
  - the happy paths T-01, T-03 and T-08, plus T-05, T-06, T-11, T-13 and T-14;
  - all five regression checks.
  - Not re-run: T-09, T-10 and T-12. They passed in round 1, and this round's changes don't touch them.
- **The test HQ's content was built through the API.** "MT S42r 0229" was built with the same requests the editor sends (create the HQ, add the Question Sets, set the audiences, save the content), using the plan's four questions and disclosures. The Smoker disclosure has a note, so the POA Summary lists it.
  - The Rules page settings (T-01), Test the whole HQ, the Preview (T-15) and Publish (T-02) were all done in the browser.
  - The editor's question-building UI was tested in round 1.
- **Patient sends:** real Chromium at 390×844 (Playwright, `is_mobile`), through the patient's UI: invite link, Continue, date of birth, answers, Send.
  - Every send got the thank-you page, with no sideways scroll and no console errors.
  - Script: `/private/tmp/claude-501/-Users-rahul-poised/e354aaca-05a3-4639-ae42-c43325b43018/scratchpad/mt2/patient.py`.
- **Episodes:**
  - T-03's episode was created on the New episode page.
  - Later episodes were created through the API as Cara (the same request the New episode page sends).
  - T-07's Fill in for the patient, Checked, re-open, validate, archive, cancel and the status change were all done in the browser.
- **Settings:** Hospital A1's Ready for admission text was switched on (T-08, R-02) and then put back to "Use the trust's setting". Afterwards no Hospital A1 setting is overridden.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Rules page message setting | Pass | No-grade box ticked, new box unticked by default with "Off: the hospital's message goes straight away…". Ticked: "On: the hospital's message (for example Ready for admission) goes once a clinician has checked the episode on the Auto-triaged list." Both kept after a reload. Default rules unchanged. |
| T-02 Publish window (E-2) | Pass | "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I. Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first." No Checked sentence. Published. |
| T-03 Low-risk patient on a phone | Pass | Thank-you page fits 390 px. On Auto-triaged (help line shown), not on Ready for review. Teal card shown with "Not yet checked by a clinician." and Checked. Both sets "Auto-triaged (ASA II, no red or amber flags)" with View. History rows by Patient and "Automatic (Low risk)". Notes by Automatic. No Undo. POA Summary: teal line, by-line "Auto-triaged (ASA II, no red or amber flags) · Patient on …", and Smoker listed, on both tabs. |
| T-04 Checked, reload, Back/Forward, two tabs (F-1) | **Pass** | After Checked: "Checked by Cara Clinician on 27/10/2026, 02:33.", button gone, still Ready for admission, note "Auto-triage checked" (Cara). Same after a reload. Off the tab, and the count went down by one. Back/Forward keep `?worklist=auto_triaged`. **Second tab:** a red message on the card, "This episode changed since you opened it. Reload to see the latest; your change was not saved.", with "Checked by Cara Clinician on …" below it and no Checked button. Only one "Auto-triage checked" note. |
| T-05 High risk (heart failure) | Pass | Ready for review, Suggested ASA III. Sets "Not started" with Validate. No auto-triage lines. On Ready for review, not on Auto-triaged. |
| T-06 Red / amber | Pass | Red: Ready for review, no auto-triage note. Amber: Ready for review with the Automatic note "Not auto-triaged by the rule “Low risk”: the answers have a red or amber flag. Review this HQ by hand." Neither is on Auto-triaged. |
| T-07 Fill in for the patient (E-1) | **Pass** | Confirm window: "…The episode moves to Ready for review, or on from there if the HQ's rules auto-triage it." Sent page: "The answers are with the team. The episode page shows its status now." Card: "…when Cara Clinician sent the HQ on the patient's behalf (no ASA grade, no red or amber flags)…". Sets stamped "Auto-triaged (no ASA grade, …)". "HQ sent" by Cara Clinician, then Automatic (Low risk). Notes "HQ completed on the patient's behalf" (Cara), then the auto-triage line (Automatic). On Auto-triaged. |
| T-08 Message at once by default | Pass | The Ready for admission text was captured 2 s after the invite, with no Checked. Note "Ready for admission message sent by text message." by Automatic. Card "Not yet checked by a clinician." with no message line. |
| T-11 Re-open (Sue Super) | Pass | Cara has no Re-open. Sue without a reason gets "Say why the HQ review is being re-opened." With "MT: check by hand": Ready for review, the card is gone, sets "Not started" with Validate, history "HQ review re-opened: Ready for admission → Ready for review" (Sue, with the reason), off Auto-triaged. Validate ×2 and Complete HQ review gave Ready for POA. |
| T-13 Evaluate service stopped | Pass | Stopped only the evaluate service (node PID 10941). Health failed, while web and API stayed up. The patient got the normal thank-you page. The episode is Ready for review with "The automatic checks didn't run. Review this HQ by hand." (Automatic), no card, not on Auto-triaged. Restarted with `make evaluate`: health ok. The next low-risk send was auto-triaged to Ready for admission. |
| T-14 Cancelled / archived | Pass | The count equalled the rows every time (13, 12, 11, 12, 11). Sue's archived episode left the list and came back unchecked after un-archive ("MT"). The cancelled one ("Patient withdrew") left the list, and its page has no Checked. |
| T-15 Test the whole HQ and Preview (E-2) | **Pass** | Smoker + tired: "Low risk: no auto-triage: a red or amber flag". Tired → No: "Low risk: auto-triage to Ready for admission". Chest pain Yes: "None". Saved amber test case "MT amber smoker" passes. Preview (About you), smoker + tired: "Low risk: no auto-triage: a red or amber flag", with the amber flag listed. The Rules intro says a rule never auto-triages with a red or amber flag, an Incomplete score, or when the anaesthetist's ASA is required first. |
| R-01 Review by hand | Pass | T-05: Validate both sets, Complete HQ review: Ready for POA. |
| R-02 Hand-picked Ready for admission text | Pass | T-05, Ready for POA → Ready for admission by hand. The text was captured, with the note "Ready for admission message sent by text message." (Cara). |
| R-03 Test the whole HQ (S40) | Pass | Smoker: "Suggested ASA II · Smoker (77176002) – Current smoker", Low risk fires. Whole-HQ test case saved and passing. |
| R-04 Suggested ASA, POA ASA section | Pass | T-03: "Suggested ASA II: Smoker (77176002) – Current smoker (the anaesthetist confirms)". POA "Pre operative assessment ASA grades: Suggested ASA II". |
| R-05 No error on a normal send | Pass | "The automatic checks didn't run" appeared only on the T-13 send made while the service was down. |

## Failures

None.

## Found while exploring

- **API checks on `POST …/episodes/{id}/triage-checked`** (done to confirm what the UI hides):
  - Val (Hospital A2) on an A1 episode: 404.
  - No sign-in: 401.
  - On an episode that wasn't auto-triaged: 409, "This episode wasn't auto-triaged, so there's nothing to check."
  - A bad body: 400, "The request could not be read."
  - A stale revision: 409 stale.
  - None of these checked the episode.
- **Cosmetic: the stale message says to reload when the card is already up to date.** After a refused Checked, the red message says "Reload to see the latest; your change was not saved." But the card already shows the latest ("Checked by Cara Clinician on …"), so reloading changes nothing. This is the app's standard stale wording, and it's clear enough. Not a failure.
- **The patient is told two different things.** On an auto-triaged send, the thank-you page says "A member of the team will review them.", and 2 s later the hospital's text says the assessment is complete and they're ready for their operation.
  - The plan expects "the normal thank-you page", and the text's wording is the hospital's setting (out of scope).
  - Noted for Rahul as a possible product question.
- **An archived auto-triaged episode still shows the teal auto-triage card,** without the "Not yet checked" line or the Checked button. After un-archive, both come back. This seems right.

## Console and network errors

- **Console:** none on the episode page, the POA Summary or the worklists, and none in any 390×844 patient run.
- **Network:** only the expected 409 (T-04's stale click), plus the exploratory API calls above. No unexplained 503 this round (round 1's E-3 didn't recur).

## Evaluate service

- At the end exactly **one** evaluate service is listening on 127.0.0.1:8090: **node PID 19687**, health `{"ok":true,…}`.
- It's the copy I started in T-13 with `make evaluate` (log `/tmp/mt-s42-evaluate.log`). Its processes:
  - **make** 19612
  - **npm run evaluate** 19621
  - **npm run serve** 19647
  - **sh** 19681
  - **node** 19687
- The conductor's original listener (node PID 10941, parent `sh` 10935) was stopped in T-13.
- The `make dev` API and web were never touched.

## GIFs

- `~/Downloads/S42-retest-T03-T04-happy-path.gif`: T-03's episode page, Checked, the worklist with Back/Forward, and T-04's refused second-tab Checked with its red message.

Patient screenshots (390×844) are in `/private/tmp/claude-501/-Users-rahul-poised/e354aaca-05a3-4639-ae42-c43325b43018/scratchpad/mt2/` (`t03-*.png`, …).

## Hand checks for Rahul

- None.
