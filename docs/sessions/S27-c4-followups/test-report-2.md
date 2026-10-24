# S27 test report 2 (retest)

**Summary: 10 passed, 0 failed, 0 blocked.** That's the two cases that failed in report 1 (T-07 with its corrected steps, T-13), the five regression cases R-01 to R-05, and three main happy paths re-run (T-01/T-02, T-03, T-15).

- **Environment:** `d8d183b` plus the uncommitted S27 work (git status not clean: the builder's changes and migration 00027). The conductor ran the app with `make dev` (web http://localhost:3000, API http://localhost:8080, database at 00027). 24 Oct 2026, 12:47–13:02 BST.
- **Browser:** Claude in Chrome, Rahul's profile.
- **Tester:** manual-test agent, automatic run, retest mode.

## How I tested (deviations from the plan)

- **Phone size:** same as round 1. Chrome's window can't be resized, so the patient steps for T-01/T-02, T-03 and R-01 ran in a **390 × 780 iframe** of the real app, opened on a `localhost:3000` page. No sideways scroll: `scrollWidth == clientWidth`.
  - T-07's patient step (entering the date of birth only) ran at desktop size in a normal tab.
- **Setting up test data through the app's own API:** where a case needed earlier steps done first, I scripted them against the same API routes the UI uses (dev login, `POST …/episodes`, and the patient's `continue` / `dob` / `answers` / `submit`, plus `complete-review`). This covered:
  - the Trust B sent HQs for T-07, T-07c and T-01r;
  - the Trust A follow-up episodes for T-07, T-07c and T-01r;
  - R-01's first sent HQ;
  - T-13's four episodes, two of which were taken to Ready for POA.

  Everything under test ran in the browser UI:
  - the patient entering their date of birth on the Trust A link;
  - Fill in for the patient, the card, Start fresh and Send;
  - Validate;
  - moving an episode to Ready for admission;
  - all four cancellations;
  - Ada's Settings, both on and back off;
  - R-01's second episode, created by searching for the patient;
  - archiving;
  - Publish;
  - the POA Summary.
- **Captured messages and audit rows:** I read captured messages from `GET /api/qa/messages` as Sam Superadmin; this is the data `/qa/messages` shows. I read Trust B's audit rows from the Audit CSV export (`GET /api/o/<Trust B>/audit?format=csv`) as Max Multi.

## Results

| Case | Result | Note |
|---|---|---|
| T-07 (corrected) Staff filling in get the same answers, card and Start fresh | Pass | Patient "S27 T07r 1250" pressed Continue on the Trust A link and entered the date of birth, then left without opening a section. Cara pressed Fill in for the patient and saw the following. **Both sets:** Please check. **Your health card:** "We've filled in 4 answers from Sam's questionnaire at Hospital B1 on 24 Oct 2026. Please check them with Sam." with Start fresh. **About you → Start fresh:** "Clear these answers? You'll answer every question in this section again with Sam. This can't be undone." After Yes, the set opened empty. Cara answered everything and pressed Send the answers (in-page modal) to get "Sent for Sam S27 T07r 1250". **Validate About you:** "Cara Clinician chose “Start fresh” with the patient on 24 Oct 2026, so the pre-filled answers were cleared…". |
| T-07 step 7 (not joined) | Pass | For "S27 T07c 1250", the Trust A invite was never opened. Fill in for the patient showed both sets as **Start**, with nothing copied. Trust B's audit has no "read to pre-fill" row for T07c. |
| T-13 Neutral Cancelled text and email, subject too | Pass | **Settings:** both default templates read "…has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." **Episodes cancelled in the UI:** one in each status (HQ not complete, Ready for review, Ready for POA, Ready for admission). **Each email:** subject "Hospital A1: your pre-operative assessment was cancelled"; body "Hello Sam,\n\nHospital A1 has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." **SMS** (T13a, +447700900132): "Hello Sam, Hospital A1 has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." **Afterwards:** both switches put back to "Use Preopsis's default" and saved. |
| T-01/T-02 happy path (patient at 390 px) | Pass | For "S27 T01r 1300", no question came before the sections. Both sets said Please check, and Send was disabled. **Your health card:** "…from your questionnaire at Hospital B1 on 24 Oct 2026…", with the 3 buttons. **Check each answer:** diabetes No and snoring Yes, each marked "From your answers on…"; pregnancy empty. Changed snoring to No. **About you:** Start fresh → Keep the answers brought the 3 buttons back. **Everything is still correct:** jumped to weight (focused). Sent: "Thank you, Sam". No sideways scroll. |
| T-03 happy path (same trust: no hospital, no Start fresh) | Pass | "S27 R01 1300", Demo follow-up HQ created by searching for the patient. The card reads "We've filled in 4 answers from your questionnaire on 24 Oct 2026. Please check them." with only 2 buttons. |
| T-15 happy path (clickable POA Summary answers) | Pass | **"Changed by the clinician on Page 1" line:** opens Validate at `?q=ponv_history`, ringed and in mid-screen (top 302, bottom 503 of 805), with the "POA Summary" back button. **Browser Back:** returns to scrollY 923.5 with the link at the same y (552). **Patient answers tab** (`?tab=patient`), "Do you snore loudly?" pre-filled line: opens Validate at `?q=snoring`. **"POA Summary" back button:** returns to the Patient answers tab at the same scroll (605) and position. **Direct `?q=snoring` URL:** no POA Summary link. |
| R-01 Everything is still correct jumps to the first empty question; Send works | Pass | Same-trust follow-up: Your health jumped to pregnancy (focused) and About you to weight. Sent. |
| R-02 "Changed by patient: pre-filled “…”" | Pass | T01r's snoring shows "Changed by patient: pre-filled “Yes”". The header says "4 answers here were pre-filled… and the patient changed 1." The POA Summary has "CHANGED BY PATIENT pre-filled “Yes”, patient answered “No”". |
| R-03 Rewording a published option is refused | Pass | Publishing "S27 formats 1235" v3: "the option “Option 2” was reworded to “Option two”. A published question's answers can't change…". Publish is disabled. Cancelled. |
| R-04 Cancel and archive make it read-only; the home says Cancelled | Pass | **Cancelled R01 episode:** View only; no Validate or Fill in. **Patient's home:** CANCELLED BY THE HOSPITAL on the pre-op HQ, and SENT with View on the follow-up. **Archived T07c (by Sue):** ARCHIVED, "It can't be changed", only Un-archive; no Fill in or Validate. |
| R-05 A correction shows "Patient answered: …" and saves | Pass | In T01r's Your health, I changed sickness after an anaesthetic to Yes. It shows "Patient answered: No". Validated, and the change shows as "Patient answered “No”, now “Yes”" on the POA Summary. |

## Failures

None.

## Found while exploring

1. **Test-plan inconsistency: T-07 and T-08 no longer agree (not a product bug).**
   - The corrected T-07 has the patient enter their date of birth on the Trust A link first.
   - But after the date of birth, the app always lands on the section list, and that loads the form. So **the patient's own open already copies Trust B's answers**, before Cara presses Fill in.
   - Trust B's audit therefore records T07r's read as **"Patient (their link)"**, not Cara Clinician. That breaks T-08's expectation "T-07's is by Cara Clinician".
   - To check that a genuine on-behalf read works and is attributed to Cara:
     1. I made a **second** Trust A episode (Demo pre-op HQ) for the already-joined T01r person.
     2. Cara pressed Fill in for the patient.
     3. The card read "We've filled in 5 answers from Sam's earlier questionnaires, including one at Hospital B1, the latest on 24 Oct 2026. Please check them with Sam.", with Start fresh.
     4. Trust B's audit shows "Cara Clinician … Patient's answers read to pre-fill their HQ at another trust … GET …/patient-form".
   - So the behaviour is right. Only the plan's T-07 set-up can't produce a staff-attributed read. A plan that wants one needs the patient to have joined through an **earlier** Trust A episode.
2. **Staff Fill in: the jump target sits partly under the sticky header (cosmetic, low confidence).** On desktop, "Everything is still correct" in Your health focused pregnancy correctly. But the question's title was just above the visible area, under the staff top bar, with only its options showing. At phone size on the patient side the question lands at the top with its title visible.
3. **Wording on the patient's home for a cancelled questionnaire that had been sent (cosmetic, older than S27).** The card says "The hospital has cancelled this, so there's nothing to fill in. Please contact them if you have any questions." That's slightly odd when the patient already sent it. It's the home card, not S27's message template.
4. **Patient reported BMI shows "—"** on the T01r POA Summary, though About you has height 170 and weight 80. The HQ review isn't complete and About you isn't validated yet, so this is probably by design (validated answers only). Noting it only in case it isn't.

## Console and network errors

- No console errors or warnings on the POA Summary or Validate.
- Every `/api/` request on those pages returned 200.
- No unexpected 4xx or 5xx during the run.

## GIFs

- `~/Downloads/S27-T07-retest.gif`: T-07 corrected, from Fill in for the patient (Please check, the card naming Hospital B1, Start fresh "with Sam") through Send to Validate's Start fresh line.
- `~/Downloads/S27-T01-retest.gif`: the main patient happy path at 390 px.
- `~/Downloads/S27-T15-retest.gif`: clickable POA Summary answers, Back and the POA Summary button.

## Hand checks for Rahul

None.

## Test data left behind

- **Patients** (all made-up `s27.*@example.test` emails):
  - "S27 T07r 1250" and "S27 T07c 1250", each with a Trust B sent HQ and a Trust A follow-up (T07c's is archived);
  - "S27 T13a–d 1300", all cancelled; T13a has mobile +447700900132;
  - "S27 T01r 1300": a Trust B sent HQ, a sent Trust A follow-up (Your health validated) and a Trust A pre-op HQ that Cara opened on the patient's behalf (not sent);
  - "S27 R01 1300": one cancelled pre-op HQ and one sent follow-up.
- **Trust A's cancelled-message settings:** back to "Use Preopsis's default".
- **Trust B:** untouched (Active).

STATUS: ALL_PASSED
