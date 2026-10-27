# S44a test report 2 (retest)

**9 passed, 0 failed, 0 blocked.** This round re-ran T-06, T-16, T-09, T-17 step 2, R-02 and R-05, plus three happy paths: T-01, T-04 steps 3–6 (inside T-06) and T-07 (inside R-02).

- Commit `97e62e2`, working tree **dirty**: S44a's changes are still uncommitted.
- 27 Oct 2026, 21:24–21:33 GMT.
- Local: web http://localhost:3000, API :8080, evaluate :8090 (`clinical@97e62e2e699f-dirty`).
- Tester: automatic run (Claude in Chrome). Rahul wasn't watching.

**How the test data was made.**
- **Episodes:** both new episodes were made through **New episode** as Cara:
  - "Rob MT S44a R2 T06 2124" (`eb189149-f499-4eb0-aebe-ce5e4ca7fec0`);
  - "Pia MT S44a R2 R02 2130" (`7a502883-5f6c-41a7-8496-39287c7a087a`).
- **Answers:**
  - T-06's were entered with **Fill in for the patient**.
  - R-02's were entered by the patient, through the invite link from /qa/messages and the date of birth.
- **HQs:** T-09 and T-17 step 2 reused the HQs from round 1 ("MT S44a pubchk2" and "MT S44a scores").
- **API calls from the signed-in browser** were used only for the server-side refusals:
  - R-02: the patient's submit with a set incomplete;
  - R-05: Complete review with sets not validated.

  In both cases the UI button is disabled.
- **Database reads:** one read of `episode_answers`, only to confirm T-06.

**Phone size.** The Chrome window still wouldn't resize (`resize_window` reported success, but the viewport stayed 1512 px wide). This round's patient pages were checked at desktop width. Round 1 checked the thank-you page at 390×844 with Playwright, and those pages haven't changed since.

## Results

| Case | Result | Note |
|---|---|---|
| T-06 | Pass | **F1 fixed.** After Fill in for the patient (Snore Yes, follow-up Yes + "loud", Snore back to No, Smoking No, send), only About you and Smoking are listed. Then Cara corrected Snore to Yes and validated. **Sleep follow-up** now shows "Not started" with no "Patient's answers entered by Cara Clinician". Its Validate page has no "entered on the patient's behalf" banner, and both questions are empty ("Patient answered: no answer" appears only once a clinician answers). POA Summary Patient answers: nothing for it. The frozen row is `{}` with `updated_by` NULL. About you and Smoking, which keep answers, still say "entered by Cara Clinician", as they should. |
| T-04 (steps 3–6, inside T-06) | Pass | The hidden set appears, Not started and empty, after the correction. |
| T-16 | Pass | **E4 fixed.** Two tabs on the same episode with every set validated. Complete HQ review in tab 1 moves the episode to Ready for POA. In stale tab 2 it shows the red "The HQ review is already complete." and the page refreshes: Ready for POA badge, "HQ review completed by Cara Clinician on 27/10/2026, 21:27.", no Complete HQ review button, and the rows show **View**. The status history has a single completion row. |
| T-09 | Pass | **E3 fixed.** Publish on "MT S44a pubchk2" lists two problems, with no "?.": "Make the question visible to patients" and "Move the condition to a clinician Question Set" (full text below). Publish is disabled. |
| T-17 step 2 (E1) | Pass | **E1 fixed.** The score card reads "Total of the scores of 3 questions (1 in earlier Question Sets), plus 1 line adding points". |
| T-01 | Pass | Preview, Clinician, on "MT S44a scores" Sleep: Snore Yes → **Incomplete**; Loud Yes → **8 (High)** (1+2 plus the 5-point line); Snore No → **0 (Low)**, with Loud hidden. |
| R-02 | Pass | Patient (Pia): About you Snore Yes, Smoking No, Sleep follow-up left unstarted → "2 of 3 sections complete", **Send my answers** disabled. The patient API's submit returns 409 "Please finish “Sleep follow-up” before sending your answers.". After completing the follow-up, the send works. The thank-you page reads "Thank you, Pia" / "Your answers have been sent to the hospital. We'll contact you about what happens next.", and is the same after a reload. The episode is **Ready for review**, with all three sets listed (T-07: everything shown, nothing dropped). |
| R-05 | Pass | Complete review with two sets not validated → 409 "Validate every Question Set first: “Sleep follow-up” and “Smoking” aren't validated." The button is disabled in the UI. |
| T-20 (spot check) | Pass | Val Clinician (A2) opening T-06's Validate URL sees "You don't have access to this hospital". |

T-09's two problems in full:
- "Health · “Anything else to tell us?” shows only when Nurse: concerns? is Yes, but patients can't see Nurse: concerns? Make the question visible to patients, or move the condition to a clinician Question Set."
- "Later · “Later” shows only when Nurse: patient needs review? is Yes, but patients can't see Nurse: patient needs review? Move the condition to a clinician Question Set, or what it tests to a patient Question Set."

## Failures

None.

## Found while exploring

- Enter submits the patient's date-of-birth form. The stale-tab Complete HQ review (a double submit across tabs) gives one completion row only.
- **Note:** with a clinician answer on a set the patient never saw (T-06's Sleep follow-up), Validate shows "Patient answered: no answer" under the changed question. That's accurate and doesn't suggest the patient gave answers.

## Console and network errors

- No console errors or warnings on the episode, Validate, Preview or patient pages checked.
- The only 4xx were the intended ones:
  - R-02's 409;
  - R-05's 409;
  - T-16's refused second Complete review (409, shown as the message).

## GIFs

All are in `~/Downloads`:
- `S44a-r2-T06.gif`: Fill in for the patient, then the hidden set with no "entered by" marker.
- `S44a-r2-T16.gif`: the stale tab's refused Complete HQ review, refreshed.
- `S44a-r2-T01.gif`: main happy path, the score in the Preview.

## Hand checks for Rahul

None.
