# S47 test report 2 (retest): assessments, STOP-Bang

**Retest: 17 cases re-run + 5 regressions: 22 passed, 0 failed, 0 blocked.**

- **Environment:** commit `10b1087` plus the uncommitted S47 work and the report-1 fixes (53 changed or new files in `git status`). Local `make dev` at http://localhost:3000: API :8080 healthy, evaluate :8090 `clinical@10b10878b8d9-dirty`. Thursday 5 Nov 2026, 21:00–21:30 IST. Chrome (Claude in Chrome), window 1512×805 (it wouldn't go narrower; the card sits in a fixed-width column, so it's the same width as at 1456).
- **Scope:** F-1 (T-10); the cases the builder changed after report 1 (T-08 step 3, T-13, the new T-17); the main happy paths (T-01, T-02, T-03, T-05, T-06 step 2, T-16 Back/Forward); and R-01 to R-05.
- **Not re-run:** T-04, T-07, T-09, T-11, T-12, T-14 and T-15. They passed in report 1 and the fixes don't touch them.
- **Test data (all made up, Hospital A1):**

| Episode | Patient | Setup | Used for |
| --- | --- | --- | --- |
| E1 `9100fef5` | Tom MT S47R 2105 One (M, 1960) | Demo sleep HQ, filled in for the patient; height 180, weight 120 | T-01–T-03, T-16, T-17 steps 3–4, R-01 |
| E2 `41c47595` | Eve MT S47R 2110 Two (F, 1990) | Demo sleep HQ, no height or weight, filled in | T-10, T-08, T-17 step 2, then cancelled (T-13) |
| E3 `6e72861e` | Raj MT S47R 2115 Three (M, 1955) | Demo STOP-Bang HQ, filled in | T-05, T-17 step 1, R-02 |
| E4 `85cf36ab` | Sid MT S47R 2120 Four (M, 1970) | Demo sleep HQ, not sent | T-06 step 2, then archived by Sue (T-13) |
| HQ `05ca103a` | "MT S47R 2125 R04" | Alex's new HQ | R-04 |

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-10 Two tabs (F-1)** | **Pass** | Tab 2's add: "STOP-Bang is already on this episode." shows in red above the one STOP-Bang. It stays (still there 4 s later) until its × is clicked. Step 2: the "changed somewhere else… your last change was not saved" message, with Reload; Reload shows tab 1's Yes. |
| T-01 7 of 8 filled | Pass | S, T, O, P Yes/Yes/No/Yes from HQ; B 37 Hospital BMI; A 66 and G Male from Profile; N Missing (orange); Incomplete; Confirm greyed; the Add assessment menu is gone. |
| T-02 Confirm and POA | Pass | N No: Clinician, 6 (High risk). "Confirmed by Cara Clinician on 05/11/2026, 21:03", kept on reload. POA: "Hospital BMI 37 (measured 05/11/2026, 21:02)", then "STOP-Bang 6 (High risk)", on both tabs. |
| T-03 Change needs a new Confirm | Pass | "Changed: confirm it again", 7 (High risk). Undo restored No and "Confirmed by". After Save: "Not confirmed", and the POA said "Not confirmed". After Confirm: 7 (High risk) on the POA. |
| T-05 HQ holding STOP-Bang | Pass | 8 of 8 filled; S, T, O, P, N from HQ; B 29.4 Patient reported BMI; A 71 and G Male from Profile; 5 (High risk), Not confirmed. POA "Not confirmed", then "5 (High risk)" after Confirm. |
| T-06 step 2 (HQ not sent) | Pass | E4: "2 of 8 filled: answer the 6 items highlighted." S, T, O, P, B and N are Missing. |
| **T-08 Typed BMI; a bad one** | **Pass** | B Missing. "abc" gives "A number, one decimal place", with Save and Confirm greyed. 36.5 gives Entered by clinician and the point (4). "2" and "151" each give "Between 5 and 150" under the box, with Save and Confirm greyed. "36,5" saves; after a reload it shows 36.5, Entered by clinician. By API, a PUT with 151 is a 400: "BMI: a number from 5 to 150." |
| **T-13 Read-only cards** | **Pass** | Cancelled E2 and archived E4 (by Sue): no Yes/No buttons, no BMI box, no Save or Confirm, no Add menu (no controls in the card at all). E4 says only "2 of 8 filled.", and E2 "8 of 8 filled.". |
| T-16 Reload, Back, Forward | Pass | POA, then Back, shows the episode with the assessment as saved. Forward shows the POA. An unsaved change was dropped on reload (E1, before T-17). |
| **T-17 step 1 Badges** | **Pass** | "Patient reported BMI", "Entered by clinician", "Hospital BMI", "Profile", "Clinician", "Missing" and "HQ" are all in normal case and whole. Measured: each badge's text width equals its box width (112/112 for Patient reported BMI). |
| **T-17 step 2 Back to the HQ's answer** | **Pass** | E2 S: No gives Clinician; Yes again gives HQ. Save and Undo changes go away (Save greyed, no Undo). |
| **T-17 step 3 On a confirmed one** | **Pass** | E1 S: No gives "Changed: confirm it again", 6 (High risk). Yes again gives "Confirmed by Cara Clinician…", HQ, 7. |
| **T-17 step 4 Emptying B** | **Pass** | E1 B: 40 gives Entered by clinician. Emptied: "Enter a BMI" in red, with Save and Confirm greyed. Undo changes brings back 37, Hospital BMI, and "Confirmed by…". |
| R-01 Observations | Pass | E1 weight 120 corrected to 90 ("MT retest: scale misread"). The old reading is struck through and marked "Corrected". The hospital BMI is 27.8, and the POA says 27.8. |
| R-02 Validate, Complete review | Pass | E3 (Ready for review; Demo STOP-Bang HQ, my own data rather than Demo pre-op HQ): Validate, then Complete HQ review. "Validated by Sue Super…", "HQ review completed by Sue Super…", Ready for POA. |
| R-03 Fill in for the patient | Pass | E1, E2 and E3: "Sent by Cara Clinician on the patient's behalf on 05/11/2026, …". The episode moved on (E1 and E2 auto-triaged; E3 went to Ready for review). |
| R-04 Ready-made STOP-Bang | Pass | As Alex: a new HQ, a BMI on page 1, then Ready-made score → STOP-Bang on page 2 (BMI from "BMI Calculator"). Inserted, marked Ready-made and locked; published as v1. |
| R-05 Settings ranges | Pass | As Ada, Peak flow "Usual to" 700 → 650 saved and kept on reload. Then "Use Preopsis's default" ticked again and saved: back to 700. |

## Failures

None.

## Found while exploring

1. **Minor: the score badge doesn't match an invalid or empty BMI box.**
   - While B's box holds something that isn't a value, the badge and the count go on scoring the last valid value.
   - Examples on E2 (unconfirmed, female):
     - after 36.5, typing "2" shows "4 (High risk)", as if B were still over 35;
     - typing "151" shows "3 (High risk)", because the last valid value was the "15" typed on the way;
     - emptying the box on E1 kept "7 (High risk)" and "8 of 8 filled." beside "Enter a BMI".
   - The source badge also still says "Entered by clinician" for an empty box.
   - Nothing wrong can be saved, since Save and Confirm are greyed. But the header shows a score for a value the box doesn't hold. Expected: "Incomplete" (or no score) while the box is invalid or empty.
2. **Cosmetic: the Yes/No column shifts sideways.** The Yes/No and BMI column moves when a wider source badge appears.
   - On E1 it moved about 38 px left when B became "Entered by clinician".
   - Between the cards it sits at different x positions (930, 952, 886 px), depending on the widest badge on the card.
   - The fix for note 1 (the column fits its words) seems to have made the column width depend on content.
3. **Question, still open (report 1, note 4):** after R-01, E1's POA shows "Hospital BMI 27.8" directly above "STOP-Bang 7 (High risk)". The confirmed STOP-Bang still counts B as 37. This is the builder's open question for Rahul.
4. **Odd inputs held up:**
   - "1e2" and an emoji in B get "A number, one decimal place".
   - " 36.5 " (spaces around it) is accepted as 36.5.
   - Closing the T-10 message with × removes it.
5. **Environment note (not the app):** during the run, the signed-in user in this Chrome profile changed from Cara to Ada Admin between two of my steps, with no action of mine. Something else using the same browser, or a manual sign-in, probably did it. I signed back in as Cara and re-checked the user before each step after that. It didn't affect any result.

## Console and network errors not tied to a case

- None. No console errors or warnings.
- No unexpected 4xx/5xx in the page's API calls (checked with `read_network_requests` and the page's resource timings).
- The only 4xx were the ones the cases caused: the 409 add (T-10), the 409 stale save (T-10) and my direct 400 PUT (T-08).

## GIFs

- `~/Downloads/S47-retest-happy-path.gif`: T-01 → T-02 → T-03, through Confirm and the POA Summary.
- `~/Downloads/S47-retest-T10.gif`: F-1 retest. Tab 2's add shows the message, which stays until it's closed.
