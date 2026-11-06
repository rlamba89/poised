# S47 test report 3 (retest after verification 1): assessments, STOP-Bang

**Retest: 9 cases + 5 regressions + 3 spot checks: 17 passed, 0 failed, 0 blocked.**

- **Environment:**
  - commit `10b1087` plus the uncommitted S47 work, report 2's fixes and verification 1's fixes (57 changed or new files in `git status`);
  - local `make dev` at http://localhost:3000: API :8080 healthy, evaluate :8090 `clinical@10b10878b8d9-dirty`;
  - Thursday 5 Nov 2026, 22:36–22:52 IST;
  - Chrome (Claude in Chrome) at 1456–1512 wide. The window still won't go narrower, as in report 2, so the patient's part ran at desktop width.
- **Scope:**
  - **T-18**, the new case for verification 1: findings 1 and 2, and minor 3;
  - report 2's two exploring notes, fixed after it: B's box (T-08, T-17 step 4) and the column moving;
  - the main happy paths: T-01, T-02, T-03, T-05, T-11 and T-15;
  - R-01 to R-05.
- **Not re-run:** T-04, T-06, T-07, T-09, T-10, T-12, T-13, T-14 and T-16. They passed in reports 1 and 2, and verification 1's fixes don't touch them.
- **Test data (all made up, Hospital A1):**

| Episode | Patient | Setup | Used for |
| --- | --- | --- | --- |
| E1 `d83138cb` | Ray MT S47R3 2240 One (M, 1958) | Demo STOP-Bang HQ, filled in for the patient (170 cm, 85 kg); no readings | T-18, T-05, R-02, R-03, exploring |
| E2 `46f4dfa1` | Tim MT S47R3 2245 Two (M, 1960) | Demo sleep HQ, sent by the patient through the invite link (170 cm, 82 kg); hospital height 180, weight 120 | T-01–T-03, B's box, T-11, R-01 |
| HQ `51117686` | "MT S47R3 2250 R04" | Alex's new HQ | R-04 |

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-18 step 1: observations fail to load** | **Pass** | See note 1 below for how the request was made to fail. The Observations card said "We couldn't reach the server…". The STOP-Bang card still showed all 8 items, with "The observations didn't load, so the hospital BMI can't be checked. Reload the page to save or confirm." in red. Confirm and Save were greyed out, and stayed greyed out after I changed N (only Undo changes came up). |
| **T-18 step 2: unblocked and reloaded** | **Pass** | The message had gone and Confirm was active. Confirm gave "Confirmed by Cara Clinician on 05/11/2026, 22:40". |
| **T-18 step 3: Confirm with no answers (API)** | **Pass** | This was before the UI Confirm. A `POST …/assessments/7e965267…/confirm` with `{data:{}, sources:{}, revision:0}` from the page's console got **400 "Answer every item before confirming: BMI has no answer."** The assessment row was the same before and after (revision 0, not confirmed). The audit log shows it as "Confirmed an assessment … REFUSED (400)". |
| **Finding 1: the age after a change** | **Pass** | E1 confirmed: the note said "The age is the patient's on the day it was confirmed (68)." After N was changed to Yes it said "The age is the patient's today." with "Changed: confirm it again" and 6 (High risk). After Undo changes, it was back to the Confirm-day note. After the re-Confirm, the POA Summary said 6 (High risk), the same score the card showed. An earlier Confirm day can't be set up by hand. |
| **Report 2 note 1: B's box (T-08 / T-17 step 4)** | **Pass** | E2 with N answered (6, High risk): <br>• "2": B **Missing**, "Between 5 and 150", "7 of 8 filled: answer the item highlighted.", **Incomplete**, Save and Confirm off. <br>• Box emptied: "Enter a BMI", B Missing, Incomplete, both off. <br>• "36.5": Entered by clinician, 8 of 8, 6 (High risk), both on. <br>• "36.5x": "A number, one decimal place", B Missing, Incomplete, both off. <br>• Undo changes: 37, Hospital BMI. |
| **Report 2 note 2: the column moving** | **Pass** | The BMI box stayed at x = 866 through every state above (Hospital BMI, Missing, Entered by clinician), on both cards. |
| T-01 7 of 8 filled | Pass | E2: S, T, O, P Yes/Yes/No/Yes from the HQ; B 37 Hospital BMI; A 66 and G Male from Profile; N Missing (orange); Incomplete; Confirm greyed out; the Add assessment menu has gone. |
| T-02 Confirm and POA | Pass | N No: Clinician, 6 (High risk). "Confirmed by Cara Clinician on 05/11/2026, 22:43", kept after reload. POA Summary: "Hospital BMI 37 (measured 05/11/2026, 22:42)", then "STOP-Bang 6 (High risk)", on both tabs. |
| T-03 A change needs a new Confirm | Pass | N Yes: "Changed: confirm it again", 7 (High risk), Confirm on. Undo: back to 6 and "Confirmed by…". Yes again and Save: "Not confirmed", and the POA said "STOP-Bang Not confirmed". Confirm: "Confirmed by Cara Clinician on 05/11/2026, 22:44"; the POA said 7 (High risk). |
| T-05 HQ holding STOP-Bang | Pass | E1: "8 of 8 filled."; S, T, O, P and N from the HQ; B 29.4 Patient reported BMI; A 68 and G Male from Profile; 5 (High risk); Not confirmed; Confirm on. |
| T-11 The patient never sees it | Pass | As Tim, the patient home and episode page show only the HQ and "Thank you, Tim". The patient's `GET /api/p/episodes/46f4dfa1…` reply holds only the HQ's own questions and the patient's own answers: no neck item, no clinician's answers, no assessment. |
| T-15 Audit log | Pass | As Ada: "Added an assessment", "Opened an episode's assessments", "Saved an assessment's answers" and "Confirmed an assessment", each naming Cara and the patient. The refused Confirm shows "REFUSED (400)". |
| R-01 Observations | Pass | E2 weight 120 corrected to 90 ("MT retest 3: scale misread"). The old reading is struck through and marked Corrected. The hospital BMI is 27.8, and the POA says 27.8. |
| R-02 Validate, Complete review | Pass | E1 (Ready for review; my own Demo STOP-Bang HQ episode, not Demo pre-op HQ): Validate, then Complete HQ review. "Validated by Cara Clinician…", "HQ review completed by Cara Clinician on 05/11/2026, 22:46.", Ready for POA. |
| R-03 Fill in for the patient | Pass | E1: "Sent by Cara Clinician on the patient's behalf on 05/11/2026, 22:38." The episode moved to Ready for review. |
| R-04 Ready-made STOP-Bang | Pass | As Alex: a new HQ with a BMI on page 1, then Ready-made score → STOP-Bang on page 2 (BMI from "BMI Calculator"). It was inserted, marked Ready-made and locked (the Name is greyed out). Published as v1 (read only). |
| R-05 Settings ranges | Pass | As Ada: Peak flow "Usual to" 700 → 650 saved, and kept after reload. Then "Use Preopsis's default" ticked again and saved: back to 700, ticked, greyed out. "Observations take precedence in scores" is listed under Observations with its help text. |

## Failures

None.

## Found while exploring

1. **How T-18 step 1 was done (method note, not a fault).**
   - Claude in Chrome can't use DevTools' request blocking.
   - Instead I replaced `window.fetch` in the page with one that rejects any `…/observations` request, as a blocked request does ("Failed to fetch").
   - Then I left the episode page and came back with Episodes and Back, so the app loaded the page again without a full reload.
   - One observations request was refused this way. A real reload afterwards cleared the replacement (step 2).
   - The result is the same as blocking the request. Rahul may still want to try the DevTools block by hand once.
2. **Double-clicks are safe.** On E1, a double-click on Save sent one PUT (200), and a double-click on Confirm sent one confirm POST (200). There was no 409 and no error message, and the card ended "Confirmed by Cara Clinician on 05/11/2026, 22:50".
3. **Clicking the answer already chosen does nothing.** Nothing is marked changed and Save stays off, which is as it should be.
4. **Still Rahul's open question (report 1 note 4, report 2 note 3).** After R-01, E2's POA shows "Hospital BMI 27.8" just above "STOP-Bang 7 (High risk)". The card's B still says 37, Hospital BMI: the confirmed STOP-Bang keeps the value it was confirmed with. Nothing on the card says the hospital BMI has changed since.

## Console and network errors not tied to a case

- **Console:** no errors or warnings, from the time console tracking started (R-04 onwards).
- **Network:**
  - no unexpected 4xx or 5xx;
  - the only refusal was my direct 400 Confirm (T-18 step 3);
  - the only failed request was the observations request I made fail (T-18 step 1);
  - every assessment request in the exploring was 200.

## GIFs

- `~/Downloads/S47-retest3-T18-confirm.gif`: T-18 step 2 and the happy path. Confirm after the observations load, "Confirmed by…", a change ("Changed: confirm it again", the age note says today), the re-Confirm, and the POA Summary's "STOP-Bang 6 (High risk)".
