# S38 test report 2 (retest)

**Summary:** 15 cases run: T-01 to T-06, T-06b, T-09, T-10, T-12, T-13 and R-01 to R-05. **15 passed, 0 blocked.** Test report 1's F-1 is fixed (T-06b), and so is its `<div>`-inside-`<p>` warning. Exploring turned up **1 failure** (Minor): when the sample patient wouldn't see the earlier Question Set, the Preview still counts the blue box's answers, so it shows a score where the real flow shows Incomplete. There is also one new console error in S38 code (Cosmetic).

**Environment:**
- Commit `a44ac2c` on master. The working tree is **not clean**: it holds S38's uncommitted build and fixes (49 changed or new files).
- Tested on 26 Oct 2026, 13:36–14:02 GMT, at http://localhost:3000 (`make dev`, API :8080).
- Chrome through Claude in Chrome, desktop size (1549 × 727).
- Users: Alex Author for authoring, and Cara Clinician for the episodes, Validate, the POA Summary and the read-only Preview.

**Test data (all made up):**
- Questionnaire **"MT S38 1340 Retest HQ"**, built through the UI:
  - v1 published, as in the plan's setup.
  - v2 published, with these changes:
    - Sleep is shown only for Female.
    - Sleep has a "Sleep pair" score (R-01) and a custom-formula Calculation (R-02).
    - Collar Yes has the disclosure "MT large collar" (R-04).
- Patients:
  - Sam Rtwo S38a: male, born 01/05/1975 (51). Two episodes on v1.
  - Sam Rtwo S38c: female, born 01/03/1986 (40). On v2.
  - Sam Rtwo S38d: male, born 01/03/1966 (60). On v2.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 | Pass | The list has the headings "Sleep › Sleep habits" (2 questions) and "This Question Set" (collar). There is no BMI in it. The card says "Total of the scores of 3 questions (2 in earlier Question Sets)". |
| T-02 | Pass | The menus are grouped Patient (Age, Sex), Measurements (BMI Calculator (BMI)), "Sleep › Sleep habits", then "This Question Set › Collar". The comparisons are =, ≠, >, ≥, <, ≤, with no "is answered". After a reload there are 3 lines, and the card says "plus 3 points lines". |
| T-03 | Pass (in passing) | No unfinished line was left after the reload. I didn't add a separate empty line this round. |
| T-04 | Pass | Low ≤ 2 and High above. High has "High risk of sleep apnoea" (Respiratory), and the card shows the disclosure badge. |
| T-05 | Pass | The blue box shows the exact text. The score is Incomplete first. It then gives 3 (High) with the note for 54 female, 4 (High) for 51 male and 3 (High) for 50 male. The form's Calculation card shows "3 (High)". |
| T-06 | Pass | Saved "High, earlier answers". After Clear answers and clearing the blue box, Load put back collar Yes and all 3 blue-box answers. Run all gave "All 1 passed". |
| **T-06b** | **Pass** | Clear answers: Incomplete, and the summary box says "No disclosures on this page yet". The blue box keeps 30 / Yes / No. Collar Yes gives 3 (High), with the note back in Clinical outputs and in the summary box. Collar No gives 2 (Low), and the note goes from both. A double-click on Clear answers is also fine. There's no `<div>`-in-`<p>` warning in the console. |
| T-09 | Pass | v1 published, and v2 later. |
| T-10 | Pass | Sam Rtwo S38a (51, male, BMI 24.2, snore Yes, tired No, collar Yes): Validate Neck shows **4 (High)** with "High risk of sleep apnoea" in the summary box. The POA Summary's Neck section lists it under "Collar summary". |
| T-12 (variant) | Pass | It ran on S38a's episode, checking the fix for re-running scores. On Validate, changing collar to No gave 3 (High) at once, and back to Yes gave 4. Correcting Sleep's tired to Yes gave **5 (High)** on Neck. Correcting snore and tired to No, with collar No, gave **2 (Low)**, and the note went from the summary box and the POA Summary. |
| T-13 | Pass | On v2, male S38d's form lists only Neck. Validate shows "Sleep risk" **Incomplete**, and the summary box has only "MT large collar", with no "High risk". |
| R-01 | Pass | "Sleep pair" in Sleep's Preview: Incomplete with one answer, and 2 (Low) with both Yes. |
| R-02 | Pass | `score('q_phnjza', 'q_eogttg')` gives 0, then 1, then 2, and never Incomplete. |
| R-03 | Pass | The v2 Sleep Preview says "The sample patient doesn't meet it, so they wouldn't see this Question Set…" for a male sample patient. Sleep is hidden for male S38d. |
| R-04 | Pass | Female S38c on v2 (BMI 39.1, all Yes): Validate Neck shows 4 (High), and the summary box lists "MT large collar" and "High risk of sleep apnoea". The POA Summary's "Collar summary" block lists both. The summary's Settings have no Logic tab. |
| R-05 | Pass | A second episode for S38a on v1 opens with both sets marked "PLEASE CHECK". Sleep says "We've filled in 2 answers from Sam's questionnaire on 26 Oct 2026". |

## Failures in detail

### F-1 (Minor): the Preview counts the earlier answers even when the sample patient wouldn't see that Question Set

**Where:** the Neck Preview of MT S38 1340 Retest HQ v2, at `/o/…fa/h/…0a/chapters/72e70059-4bf2-4ecb-8b6d-24b0c99679a3/preview`. In v2, Sleep is shown only when the patient's sex is Female. I saw it as Cara Clinician, read-only, and the same Preview code serves Alex.

**Steps:**
1. Switch to Clinician.
2. In the blue box, give BMI 36, snore Yes and tired Yes. Answer collar Yes.
3. Change the Sample patient's sex from Female to Male.

**Expected:** a male patient never sees Sleep, so Sleep's answers count as unanswered and "Sleep risk" is **Incomplete**. That is what the real flow shows for male S38d (T-13). In the Preview, at least, the blue box should say that the sample patient wouldn't see Sleep.

**Actual:**
- **The score counts Sleep's answers anyway.** With Female it shows "Sleep risk: 5 (High)". With Male it shows **6 (High)**: the male point is added, and the Sleep answers still count.
- **The disclosure shows when it shouldn't.** "High risk of sleep apnoea" stays in Clinical outputs and the summary box.
- **The blue box gives no hint** that Sleep is hidden for this sample patient.
- **Test cases are affected too.** A test case saved here keeps the answers and expects High. It would pass in the Preview, although the real HQ gives Incomplete for that patient.
  - I saw the same thing with the test case from T-06 (male, 50). On v2 it still produced "High risk of sleep apnoea". Its only mismatch was the new "MT large collar" note, so I deleted that test case before publishing v2.

**Why it matters:** an author checking a score in the Preview gets a different result from what a clinician will see. The brief says each set gets only the "earlier *shown* sets' answers". This is authoring-only and isn't seen by patients. It may be accepted as a Preview simplification, but then the blue box should say so.

**Console:** nothing tied to it.

**GIF:** `~/Downloads/S38-retest-E01-hidden-set-preview.gif`.

## Found while exploring

- **Clear answers, a double-click:** fine. The blue box keeps its answers, and the score re-counts on the next answer.
- **Back from a Preview** to the previous Validate page is fine. Its answers and summary box are unchanged.
- **Patient view of the Preview:** with the blue box filled and collar Yes, Clinical outputs lists only the ordinary disclosure ("MT large collar"). It has no Scores and no band disclosure, because the score is a clinician-only Calculation. This looks like the intended behaviour (the patient form doesn't hold clinician items), but the Clinician view of the same answers shows "High risk of sleep apnoea".
- **POA Summary after a correction:** when the Neck correction removed the only disclosure, Neck's "Collar summary" block (and its "Validated by…" line) no longer appeared on the POA Summary. Only the "Changed by the clinician" line stayed. This is probably S37's rule that a summary with no notes or comments is hidden. Noted, not S38.
- **Pre-fill labels:** "PRE-FILLS FROM 44 OTHER HQS" on "Do you snore loudly?". This comes from e2e HQs on the shared database, as in round 1.

## Console and network errors not tied to a case

- **A new React console error in S38 code, on every Preview with a blue box** (seen on page load):
  - The message is "A props object containing a "key" prop is being spread into JSX … @mantine/core/NumberInput". A second one says the same for `@mantine/core/Select`.
  - The cause is `apps/web/src/features/preview/EarlierAnswers.tsx`, line 50. `const common = { size, label, key }` is spread into `<Select {...common}>` and `<NumberInput {...common}>`. React wants `key` passed directly.
  - Cosmetic: the page works, but it shows as a console error.
- The `<div>`-inside-`<p>` warning from round 1 no longer appears.
- No unexpected 4xx or 5xx requests (checked on Validate and the Preview).

## Hand checks for Rahul

- **T-14 at a true 390 px** is still a hand check, as in round 1. Chrome's window can't go below 500 px here. I didn't re-run T-14 this round.

## GIFs

- `~/Downloads/S38-retest-T06b-clear-answers.gif`: the main happy path, T-05 to T-06b. It shows the earlier answers in the blue box, 3 (High) with the note, Clear answers, the note going and coming back, collar No giving 2 (Low), saving and loading the test case, and Run all.
- `~/Downloads/S38-retest-E01-hidden-set-preview.gif`: F-1, with the sex going Female → Male and the score going 5 (High) → 6 (High).
