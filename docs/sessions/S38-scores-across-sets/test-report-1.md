# S38 test report 1

**Summary:** 25 cases run (T-01 to T-20, R-01 to R-05). **25 passed, 0 blocked.** Exploring turned up **1 failure** (Minor): in a Question Set's Preview, **Clear answers** stops the earlier sets' answers from counting, and the summary box keeps showing a stale disclosure. There is also one new console error, and a few small observations.

**Environment:**
- Commit `a44ac2c` on master. The working tree is **not clean**: it holds S38's uncommitted build (48 changed or new files).
- Tested on 26 Oct 2026, 12:42–13:30 GMT, at http://localhost:3000 (`make dev`, API :8080).
- Chrome through Claude in Chrome.
- Users: Alex Author for authoring, Cara Clinician for the episodes and Validate, Alex for the R-04 episode, and one patient through the invite link (T-14).

**Test data (all made up):**
- Questionnaires:
  - "MT S38 1245 Scores HQ": v1 and v2 published, v3 a draft.
  - "MT S38 1255 STOP-Bang HQ": v1 published.
  - "MT S38 1340 Regression HQ": v1 published.
- Patients: Sam S38a, S38b, S38c, S38d and S38e Mtone; Sam S38 SBa to SBe Mtone; Sam S38 Reg Mtone.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 | Pass | The list has the headings "Sleep › Sleep habits" (2 questions) and "This Question Set" (collar). It doesn't list the BMI. The card says "Total of the scores of 3 questions (2 in earlier Question Sets)". |
| T-02 | Pass | The menus are grouped Patient (Age, Sex), Measurements (BMI Calculator (BMI)), "Sleep › Sleep habits", then "This Question Set › Collar". The comparisons are =, ≠, >, ≥, <, ≤ for numbers and is, is not, is any of, is none of for Sex, with no "is answered". After a reload there are 3 lines, and the card says "plus 3 points lines". |
| T-03 | Pass | A line with Age and no value isn't kept after a reload. |
| T-04 | Pass | Low ≤ 2 and High above. High's disclosure ("High risk of sleep apnoea", Respiratory) is saved, and the card shows "LOW ≤ 2, HIGH ABOVE, 1". |
| T-05 | Pass | The blue box shows the exact text. The score is Incomplete with no note, then 3 (High) with the note for 54 female, 4 (High) for 51 male and 3 (High) for 50 male. The form's card matches each time. |
| T-06 | Pass | Load puts back the collar answer and all 3 blue-box answers (3 (High)). Run all shows "All 1 passed". See F-1 for what happens after Clear answers. |
| T-07 | Pass | With Neck first, the header says "1 logic problem": "Sleep risk": it uses a question in a later Question Set (q_osafgn, q_pfvdon, q_muvbll). Publish is disabled, and it also lists the test case as failing. Moving Neck back clears the problem. |
| T-08 | Pass | "Delete this?" lists "Neck · "Sleep risk" (calculation)". After a real delete, the Publish window says "it tests something that no longer exists (q_muvbll)". Undo restores the question. (Observation O-2: the header showed no warning while on Sleep.) |
| T-09 | Pass | v1 published. |
| T-10 | Pass | Validate Neck shows 4 (High), and the summary box lists "High risk of sleep apnoea". The POA Summary's Neck section lists it under "Collar summary". |
| T-11 | Pass | The patient was born 26/10/1976, so 50 today: 1 (Low), with no note on Validate or the POA Summary. |
| T-12 | Pass | 3 (High) at first. After changing tired to No on Sleep (the "Patient answered: Yes" note shows) and validating, Neck shows 2 (Low) and the box says "No disclosures on this page yet". |
| T-13 | Pass | v2 has Sleep shown only for Female. For male Sam S38d the patient form and the HQ card list only Neck. Validate shows "Sleep risk" Incomplete and "No disclosures on this page yet", and the POA Summary has no such note. |
| T-14 | Pass | The patient flow through the invite link works (Continue, date of birth by Enter, both sets, sent). There is no sideways scroll, and "Sleep risk", "Incomplete" and the value never appear. The patient API's JSON has no Calculation or expression. **Note:** Chrome won't make the window narrower than **500 px**, so I tested at 500 × 701. At 390 px I checked only the final "Thank you" screen, in an iframe (scrollWidth 386 = clientWidth). |
| T-15 | Pass | The window names STOP-Bang, the updated version (17 in / 43 cm, 16 in / 41 cm), Chung et al., Chest 2016;149(3):631–8 and the UHN licence. BMI from is already set to "BMI Calculator (BMI)". The inserted Group has its badge and the licence line, 5 questions each at "1 pt", and a Clinician "STOP-Bang score" with Low risk ≤ 2, Intermediate risk ≤ 4 and High risk above. |
| T-16 | Pass | The question cards have no Copy, Move or Delete buttons. Dragging one out of the Group does nothing. The Settings show the blue note, every field is greyed out (typing is ignored), and there is no Logic tab. A Disclosure on snoring Yes saves. The score's Settings show "Also High risk when: Yes to 2 or more of the 4 STOP questions, and male, a BMI over 35 or a large neck". The Group has Move up, Move down and Delete but no Copy, has nothing to add inside it, and saves a condition (age ≥ 18). |
| T-17 | Pass | A second insert says "This HQ already has STOP-Bang. A ready-made score can be in an HQ only once." With no BMI, an orange note appears and Insert is disabled. |
| T-18 | Pass | I ran the snippet's logic with `return` instead of `alert`, so that no browser dialog opened. Publish then lists "Sleep check · STOP-Bang has been changed: a ready-made score can't be changed. Delete it and insert it again." and is disabled. After deleting and reinserting STOP-Bang, it publishes. |
| T-19 | Pass | a) 3 (Intermediate risk); b) 3 (High risk); c) 3 (High risk) at BMI 36; d) 0 (Low risk); e) Incomplete with sex Not known. |
| T-20 | Pass | Reload, Back and Forward on T-10's Neck Validate keep 4 (High) and the note. The Neck Preview URL in a new tab shows the blue box with its earlier answers empty. |
| R-01 | Pass | Within one set: 2 (Low) with both Yes, and Incomplete with one answered. |
| R-02 | Pass | `score('q_alpard', 'q_okgtry')` gives 0, then 1, then 2, and never shows Incomplete. |
| R-03 | Pass | For a male sample patient, v2 Sleep's Preview says "The sample patient doesn't meet it, so they wouldn't see this Question Set…". The set is hidden for male Sam S38d (T-13). |
| R-04 | Pass | An ordinary Yes disclosure ("MT drinks alcohol") shows in the summary box in the Preview and on Validate, and in the POA Summary's "Page 1 summary" block. The summary's Settings have no Logic tab. |
| R-05 | Pass | A second episode for Sam S38c on the same HQ (v2) opens with both sets marked "PLEASE CHECK". Sleep shows "We've filled in 2 answers from Sam's questionnaire on 26 Oct 2026". |

## Failures in detail

### F-1 (Minor): after Clear answers, the Preview ignores the earlier answers still shown in the blue box, and the summary box keeps a stale disclosure

**Where:** the Neck Preview of MT S38 1245 Scores HQ, as Alex Author. It happens on any set whose score uses earlier sets. Example URL: `/o/…fa/h/…0a/chapters/3b68db35-3e25-49f2-8679-62e1024df982/preview`.

**Steps:**
1. Switch to Clinician.
2. In the blue box give BMI 30, snore Yes (1 pt) and tired No (0 pts). Answer collar Yes. Scores shows 3 (High) and the summary box shows "High risk of sleep apnoea", which is right.
3. Press **Clear answers**.
4. Answer collar **Yes** (or No) again, without touching the blue box.

**Expected:** one of two things.
- The blue box's answers still count, since they're still shown: with collar Yes the score is back to 3 (High), or 2 (Low) for collar No as 50 male.
- Or Clear answers clears the blue box too.

In both cases the summary box should match Clinical outputs.

**Actual:**
- **The blue box answers are ignored after step 3.** The blue box still shows 30, Yes (1 pt) and No (0 pts), but:
  - the score stays **Incomplete** whatever collar answer you give;
  - Clinical outputs says "The current answers produce no outputs".
- **They count again only once you edit one of them.** Changing the BMI to 31 brings back 3 (High).
- **The summary box goes stale.** Straight after step 3 it still lists **"High risk of sleep apnoea"**, while Scores says Incomplete and Clinical outputs says no outputs. It stays that way after the blue-box answers are cleared one by one. It clears only when an answer in the set changes.
- **Ordinary disclosures aren't affected.** A one-set HQ (MT S38 1340 Regression HQ) clears its summary box correctly on Clear answers, so the problem is new in S38's earlier-answers code.

**Console:** no error tied to it.

**GIF:** `~/Downloads/S38-E01-clear-answers.gif`.

## Found while exploring

- **O-1, the "1 points line" wording:** the card says "plus 1 points line" for one line, against "plus 2 points lines" for two. "1 point line" may read better. Cosmetic.
- **O-2, header problems depend on the open set:** T-08's note says the header should show "tests something that no longer exists" after the delete. While the Sleep page was open the header showed no warning; only the Publish window named the problem. In T-07 the header warning appeared only once a Question Set was opened, not on the plain structure view. This may be by design (the header shows the open set's problems), but it doesn't match the plan's wording.
- **O-3, one failed attempt to create an episode:** on New episode, after Choose for an existing patient (S38c) and picking the HQ, one press of Create episode sent me back to a blank New episode form with no episode made. It worked at once on a second try. I couldn't reproduce it, and my early click may have landed on "Change patient".
- **Double-clicking Insert:** double-clicking Insert in the Ready-made score window inserts only one STOP-Bang (checked in the chapter JSON). Good.
- **BMI in the blue box:** it accepts −5 (Preview only, so harmless).
- **Pre-fill labels:** the new question "Do you snore loudly?" shows "PRE-FILLS FROM 40 OTHER HQS", and STOP-Bang's questions show "PRE-FILLS FROM 3 OTHER HQS". These come from wording and key matches with earlier e2e HQs on this shared database, and were expected.
- **The Validate button:** on a two-page set (Sleep), it validates the whole set from page 1 without visiting the BMI page. That's as before, not S38.

## Console and network errors not tied to a case

- **A new React warning in S38 code, on every Preview with Scores:** "In HTML, `<div>` cannot be a descendant of `<p>`. This will cause a hydration error." It comes from `OutputsPanel` → `preview-sc…` (Scores): a Mantine `Badge` (the INCOMPLETE badge) sits inside `Text` (a `<p>`). Cosmetic, but it shows as a console error.
- No unexpected 4xx or 5xx requests.

## Hand checks for Rahul

- **T-14 at a true 390 px:** Chrome's window can't go below 500 px here, so the question pages were checked at 500 px. Hand check them at 390 × 844 in DevTools device mode.

## GIFs

- `~/Downloads/S38-T05-happy-path.gif`: the main happy path, T-05 and T-06: the Preview with the earlier answers, the score bands, saving and loading a test case, and Run all.
- `~/Downloads/S38-E01-clear-answers.gif`: F-1.
