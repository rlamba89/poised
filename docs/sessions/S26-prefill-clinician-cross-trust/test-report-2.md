# S26 test report 2 (retest)

**8 passed, 0 failed, 0 blocked.** This retest covers the two fixes made after test report 1 (T-12 and T-13), plus the happy paths T-01, T-07 and T-09, and regressions R-01 and R-02.

**Environment**
- Commit `08c7645`. The working tree isn't clean: it holds the uncommitted S26 build plus the two fixes (46 changed or new files).
- 23 Oct 2026, 14:16–14:24 BST, on http://localhost:3000 (API on :8080). The conductor's `make dev` was running on the latest code.
- **Browser:** Claude in Chrome, Rahul's "rahul chrome" profile.
  - Patient screens ran in a same-origin 390 × 844 iframe. It was scaled up on screen only, so the layout stayed 390 wide.
  - Staff screens ran at desktop size.
- **Settings:** `crossTrustPrefill` was off in both trusts at the start (`trust: {}`).
  - Both were switched on through `PATCH /api/o/{oid}/settings`, with Ada's and Max's dev-login cookies.
  - At the end, both overrides were cleared again (`trust: {}`, effective `false`), and I checked this with the API.

**What I did through the API rather than the UI** (setup only):
- A script used the e2e helpers and the real API to set up three people:
  - **T-01's person:** Trust A's first Demo pre-op HQ, answered, sent and reviewed by Cara, who corrected snoring to Yes.
  - **T-07's and T-09's people:** their Trust B Demo pre-op HQs (answered and sent), and their Trust A Demo follow-up HQ episodes.
- **Through the UI:**
  - T-01's second episode: Cara searched for the patient and created it.
  - Every patient step: the invite link, Continue, the date of birth, the consent card, the opening card, the answers and Send.
  - Validate, the POA Summary and both trusts' Audit pages.
- A few SurveyJS **Next** and **Finish this section** buttons, and one radio button (pregnancy), were clicked with a script inside the patient iframe. The page handled these exactly as a tap.

## Results

| Case | Result | Note |
|---|---|---|
| **T-12** Audit in both trusts (fix) | Pass | **Trust B** (UI as Max at Hospital B1, filtered "S26 R2T"): one row "Patient's answers **read** to pre-fill their HQ at another trust (they agreed)", by "Patient (their link)", at Hospital B1, for T-07's person. The API shows it's their **Trust B** record (`f4a8a33e…`). There's none for T-09's person, who said No. Round 1's T-08 row now shows the new wording too. **Trust A** (UI as Ada): "Patient answered whether to use another trust's answers" for T-07 and T-09, both DONE, at Hospital A1. The Trust A rows name only Trust A records: none of the Trust B patient or episode ids appear in the API response. |
| **T-13** Refused requests (fix) | Pass | Run from the patient's own signed-in page (`/home/episodes/6ef0e88c…`). <ul><li>Already answered: 409. An unknown uuid: 409. Both say "This question has already been answered, or no longer applies. Please reload the page."</li><li>`not-a-uuid`, an empty orgId, a missing orgId, and an orgId with leading spaces: **400 "We couldn't tell which answers this is about. Please reload the page."** (the fix).</li><li>A missing `agree`: 400 "Choose “Yes, use them” or “No, thanks”."</li><li>`agree: "yes"`, a numeric orgId, or broken JSON: 400 "The request could not be read."</li><li>The zero uuid, and another person's episode: 404 "Episode not found."</li><li>Signed out, and with a staff cookie (`curl`): 401.</li></ul> |
| T-01 Validate markers (happy path) | Pass | Your health: "6 answers here were pre-filled … the patient changed 1. The patient chose “Everything is still correct” on 23 Oct 2026, so these pre-filled answers weren't checked one by one." Diabetes "(matched by same wording)". Snoring is **Yes** "(matched by link)", which is Cara's correction. Sickness is No, with "Changed by patient: pre-filled “Yes”". Pregnancy has no marker. Problem and What happened have markers, and so does Your allergies on page 2. About you: "3 answers … The patient chose “Check each answer” on 23 Oct 2026." Smoking and alcohol have markers on page 1. The POA Summary shows the "Pre-filled on Page …" boxes. |
| R-01 Same-trust card | Pass | "We've filled in 6 answers from your questionnaire on 23 Oct 2026. Please check them." **Everything is still correct** jumps to the pregnancy question. Changing sickness to No removed its marker. |
| R-02 Started | Pass | The pre-filled HQ showed **Please check** on both sections before anything was touched. |
| T-06 The question (phone) | Pass | "Use your earlier answers?" / "Use the answers you gave Hospital B1 in October 2026?", the explanation, and two full-width buttons at 390 wide. No section tiles. |
| T-07 Yes fills from Trust B | Pass | After Yes: both sections show Please check. A reload doesn't ask again. The card says "We've filled in 4 answers from your questionnaire on 23 Oct 2026". Diabetes No and snoring Yes, each with "From your answers on 23 Oct 2026". After Send, Validate shows "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust), 23 Oct 2026 (matched by link)", and diabetes "(matched by same wording)". |
| T-09 No, thanks remembered | Pass | Both sections show **Start**. After a reload: no question, still Start. |

## Failures
None.

## Found while exploring
1. **Still there, and older than S26 (round 1's found 1):** Validate shows a false correction marker on a list's optional column.
   - **Where:** T-01's second episode, Your health page 2, under "What happens when you have it?" in **Your allergies**.
   - **What it shows:** "Patient answered: no answer".
   - Not an S26 regression.
2. **Checked, no problem:** T-13's request with an orgId but no `agree` still gets "Choose “Yes, use them” or “No, thanks”." The orgId message is used only when the orgId itself is missing or bad, which reads correctly.
3. **Observation:** each refused T-13 request adds a row to Trust A's audit, e.g. "Patient answered whether to use another trust's answers", with the result "REFUSED (400)" or "(409)". This is consistent with how other refused requests are logged.

## Console and network errors
- No console errors or warnings on the staff Validate page and the POA Summary after reloading them.
- I saw no unexpected 4xx or 5xx. The only refusals were T-13's deliberate ones.

## Hand checks for Rahul
None. T-13's "new incognito window" step was done with `curl` and no cookies (401), which tests the same thing.

## GIFs
- `/Users/rahul/Downloads/S26-retest2-T01-happy-path.gif`: Cara creates the follow-up episode. The patient goes through the invite and date of birth, then "Everything is still correct", changes sickness and sends. Then Cara's Validate.
- `/Users/rahul/Downloads/S26-retest2-T07-cross-trust-yes.gif`: the cross-trust question, then Yes, Please check, the opening card, Send, and Validate with "Hospital B1 (another trust)".

## Test data created
- **Trust A:**
  - "Sam S26 R2T01 1416": two episodes, the pre-op HQ (reviewed) and the follow-up HQ (sent).
  - "Sam S26 R2T07 1416": the follow-up HQ, sent.
  - "Sam S26 R2T09 1416": the follow-up HQ, not started.
  - All are born 17/05/1980, with emails `s26r2.*@example.test`.
- **Trust B:** "Sam S26 R2T07 1416" and "Sam S26 R2T09 1416", each with a sent Demo pre-op HQ.
