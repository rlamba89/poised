# S38 test report 3 (retest)

**Summary:** 11 cases run: T-06c (new), T-06b, T-10, T-13, T-20 (reload part), the publish check of test cases, and R-01 to R-05. **11 passed, 0 failed, 0 blocked.**
- Test report 2's F-1 is fixed (T-06c).
- Its console error is fixed too: there's no `key` prop error on the Preview any more.
- Exploring found nothing that fails. There's one small note: the blue box takes a negative BMI.

**Environment:**
- Commit `a44ac2c` on master. The working tree is **not clean**: it holds S38's uncommitted build and fixes (51 changed or new files).
- Tested on 26 Oct 2026, 14:07–14:18 GMT, at http://localhost:3000 (`make dev`, API :8080).
- Chrome through Claude in Chrome, desktop size (1549 × 783).
- Users:
  - Alex Author, for authoring, the Preview and publishing.
  - Cara Clinician, for the episodes, Validate, the POA Summary and the read-only Preview.

**Test data (all made up):**
- **"MT S38 1340 Retest HQ" v3**, from round 2's questionnaire.
  - Made with **Create new version**, so it's a copy of v2: Sleep is shown only when the patient's sex is Female, and Neck has "Sleep risk" plus "MT large collar" on collar Yes.
  - Two test cases were added on Neck: "MT 1410 male, Sleep hidden" and "MT 1410 female, High".
  - Then published.
- **Sam Rthree S38f:** female, born 01/03/1971 (55). Two episodes on v3.
- **Sam Rthree S38g:** male, born 01/03/1966 (60). One episode on v3.

## Results

| Case | Result | Note |
|---|---|---|
| **T-06c** | **Pass** | Neck Preview (Clinician): BMI 36, snore Yes, tired Yes, collar Yes. **Female 54:** "Sleep risk: 5 (High)", with "High risk of sleep apnoea" in Clinical outputs and in the summary box. **Male:** the blue box says, in red, "The sample patient wouldn't see “Sleep”, so its answers don't count." Scores shows **Incomplete**, and so does the form's Sleep risk card. The note goes from both places, and only "MT large collar" stays. **Female again:** 5 (High), and the note is back. With Male, I saved "MT 1410 male, Sleep hidden"; Load puts back Male and Incomplete. I set Female and pressed Run all: **All 1 passed**. With a female case added: **All 2 passed**. No console error about a `key` prop, or any other. |
| Publish check (part of F-1's fix) | Pass | Publishing v3 with both test cases listed no problems, and it published (v3 · published). |
| T-06b | Pass | Clear answers (Female): Incomplete and "No disclosures on this page yet". The blue box keeps 36 / Yes / Yes. Collar Yes gives 5 (High) with the note. Collar No gives 4 (High) and only the band note. That's right: the band note stays because 4 is above the Low limit of 2. |
| T-10 (on v3) | Pass | Female S38f (55; snore Yes, tired No, 170 cm / 70 kg (BMI 24.2), collar Yes) gives **3 (High)** on Validate Neck (1 + 1 + age 1). The summary box lists "MT large collar" and "High risk of sleep apnoea". After Validate, the POA Summary's Neck › "Collar summary" lists both. |
| T-20 (reload) | Pass | Reloading Validate Neck still shows "3 (High)". |
| T-13 (on v3) | Pass | Male S38g (60): the form and the Health questionnaire list only Neck. Validate Neck shows "Sleep risk" **Incomplete**, and the summary box has only "MT large collar". This matches the male result in the Preview. |
| R-01 | Pass | In Sleep's Preview, "Sleep pair" is Incomplete with one answer and **2 (Low)** with both Yes. |
| R-02 | Pass | The "Old formula" Calculation shows 0, then 1, then 2, and never Incomplete. |
| R-03 | Pass | Sleep's Preview with a male sample patient says "The sample patient doesn't meet it, so they wouldn't see this Question Set. The questions below are shown anyway." With female, it says "…meets it, so they would see this Question Set." |
| R-04 | Pass | The option disclosure "MT large collar" shows in the summary box on Validate and in the POA Summary's "Collar summary" block. I didn't recheck the summary's missing Logic tab this round (it passed in round 2, and nothing touching it has changed). |
| R-05 | Pass | A second episode for S38f on v3 opens with Sleep and Neck both marked **PLEASE CHECK**. |

## Failures in detail

None.

## Found while exploring

- **The sample patient's sex left empty** counts as "not Female": the blue box gives the same red note that Sleep is hidden, and the score is Incomplete. That's consistent with the real flow.
- **Cara (clinician) in the Preview:** the test cases are read-only ("only super clinicians and admins of the whole trust can change test cases"). Run all still works for her: All 2 passed.
- **A negative BMI in the blue box:** the BMI box takes **-5** and uses it as given, so it adds no point. This is authoring-only and can't happen in a real HQ, because the BMI there is calculated. A minimum of 0 would be tidier. Not counted as a failure.
- **A known simplification, not tested as a failure:** the build notes say a question hidden *inside* an earlier set still counts in the Preview. The real flow skips it.

## Console and network errors not tied to a case

- None. On the Preview with a blue box, the console has only Vite's and React DevTools' info lines.
- Every `/api/` request on loading the Preview returned 200.

## Hand checks for Rahul

- **T-14 at a true 390 px** is still a hand check, as in rounds 1 and 2: Chrome's window can't go below 500 px here. It wasn't re-run this round.

## GIFs

- `~/Downloads/S38-retest3-T06c-hidden-set-preview.gif`: T-06c, the main path. It shows the blue box filled in, Female at 5 (High) with the note, Male with the red note, Incomplete and no band note, Female again, saving the male test case, and Run all passing.
