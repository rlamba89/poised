# S38 test report 4 (retest after verification 1)

**Summary:** 13 cases run: V-01, V-02, V-03 (verification 1's F1–F3), and a quick pass of T-01 to T-05, T-09, T-10, T-15 to T-17, plus the reload part of T-20. **13 passed, 0 failed, 0 blocked.**
- F1 is fixed: removing a line or changing its points while another line is unfinished is saved.
- F2 is fixed: a key repeated in two Question Sets no longer crashes the editor.
- F3 is fixed: STOP-Bang refuses a repeated fixed key.
- Exploring found nothing that fails.

**Environment:**
- Commit `a44ac2c` on master. The working tree is **not clean**: it holds S38's uncommitted build and fixes (53 changed or new files).
- Tested on 26 Oct 2026, 14:59–15:16 GMT, at http://localhost:3000 (`make dev`, API :8080). The database was freshly migrated and seeded.
- Chrome through Claude in Chrome, desktop size (1549 × 783).
- Users:
  - Alex Author, for authoring, the Preview and publishing.
  - Cara Clinician, for the episode, Fill in for the patient, Validate and the POA Summary.

**Test data (all made up):**
- **"MT S38 1500 Scores HQ"**, built through the editor as the plan's setup describes:
  - **Sleep** › Sleep habits (snore and tired, both scored 1/0) and › Size (BMI);
  - **Neck** › Collar: collar (scored 1/0), a Clinical Summary, "Sleep risk", and an extra "MT V01 score" for V-01.
  - "Sleep risk" has the three questions, the lines Age > 50, Sex is Male and BMI > 35, bands Low ≤ 2 and High above, and the disclosure "High risk of sleep apnoea" (Cardiovascular) on High.
  - v1 is published. v2 is a draft, used for V-02 and V-03: the snippets gave it a repeated key and a `stopbang_snoring` ID, so it can't be published.
- **"MT S38 1500 STOP-Bang HQ"** v1 draft: Measures (BMI) and Sleep check (STOP-Bang). It has a disclosure "MT high blood pressure" on the pressure question's Yes, and the Group's condition "Patient's age ≥ 18".
- **Sam MTfour S38a**: male, born 01/05/1975 (51). One episode on Scores HQ v1.

## Results

| Case | Result | Note |
|---|---|---|
| **V-01** | **Pass** | "MT V01 score" had two lines, Age > 50 and Sex is Male. I added an empty line and removed the Sex line. The card at once said "plus 1 points line", and the header said "All changes saved". I checked the API: the chapter's expression had only the Age line (`iif({patientAge} > 50, 1, 0)`). I then set the Age line to **2** points with the empty line still open and reloaded. After the reload, only the Age line is there, at "Add 2 points when", and the empty line is gone. On Validate, the score is 3 for a 51-year-old with collar Yes (1 + 2), which confirms the 2 points were saved. |
| **V-02** | **Pass** | In the v2 draft I ran the snippet; it said "Do you snore loudly? (asked again)" in Neck repeats ID q_cxodlp. I reloaded and selected "Sleep risk": no "Something went wrong" page. **Add up the scores of** lists "Are you tired in the day?" under Sleep and "Is your collar large?" and "Do you snore loudly? (asked again)" under This Question Set: the ID appears once, as this set's copy. The points line menu is the same: Sleep › Sleep habits has only "Are you tired…", and This Question Set › Collar has the collar and "(asked again)". Publish is refused: "1 problem to fix first. Neck · The question ID q_cxodlp is used "Do you snore loudly?" in "Sleep" and "Do you snore loudly? (asked again)" in "Neck". Each question needs its own ID." The Publish button is disabled. There were no console errors. |
| **V-03** | **Pass** | In the same v2 draft (it has a BMI and no STOP-Bang), I ran the snippet, reloaded, and pressed **Ready-made score** → **Insert**. It's refused in red: "This HQ already has a question with the ID stopbang_snoring. A question can be in an HQ only once (copy it to ask it again under a new ID)." Nothing was inserted: the page's text after Cancel has no STOP-Bang. |
| T-01 | Pass | **Add up the scores of** shows the heading "Sleep › Sleep habits" with the two Sleep questions, and "This Question Set" with the collar. The BMI isn't listed. With all three picked: "Total of the scores of 3 questions (2 in earlier Question Sets)". |
| T-02 | Pass | The menu groups are Patient (Age, Sex), Measurements (BMI Calculator (BMI)), "Sleep › Sleep habits" and "This Question Set › Collar". A later Calculation's menu also has Other scores, with "Sleep risk (This Question Set › Collar)". The comparisons are =, ≠, >, ≥, <, ≤: there is no "is answered", and no formula box. After a reload, the three lines are there, each "Add 1 point when", and the card says "…plus 3 points lines". |
| T-03 | Pass | I added a fourth line with Age and no value, and reloaded. Only the three finished lines were there. |
| T-04 | Pass | Add band twice gives Low ≤ 2 and High above. The High band's disclosure "High risk of sleep apnoea" (Cardiovascular) saved, and the card shows a disclosure count of 1. |
| T-05 | Pass | Neck's Preview (Clinician) shows the blue box: "Uses answers from earlier Question Sets, which count as unanswered here. To try them, give them below." With collar Yes only, the score is "Sleep risk: Incomplete" and there's no note. With BMI 30, snore Yes and tired No, it's **3 (High)** for the sample patient (54, female), and the note shows in Clinical outputs and in the summary box. With Male and 51 it's **4 (High)**; with Male and 50 it's **3 (High)**. The Calculation in the form shows "3 (High)". There were no console errors. |
| T-09 | Pass | v1 published with no problems (v1 · published). |
| T-10 | Pass | Sam MTfour S38a: male, 51. I filled in for the patient: snore Yes, tired No, 170 cm and 70 kg (the form showed BMI 24.2), collar Yes. I sent the answers. The patient form never shows "Sleep risk". Validate Neck shows **"Sleep risk" 4 (High)**, with "High risk of sleep apnoea" in the summary box. After Validate, the POA Summary's Neck › "Collar summary" lists "High risk of sleep apnoea". |
| T-20 (reload) | Pass | Reloading Validate Neck still shows "4 (High)" and the note. |
| T-15 | Pass | The window names STOP-Bang, its source ("Chung F, Abdullah HR, Liao P… Chest 2016;149(3):631–8", the updated version) and its licence (UHN, stopbang.ca). **BMI from** is already "BMI Calculator (BMI)". Insert gives a Group "STOP-Bang" with the **Ready-made: STOP-Bang** badge and the licence line. It has the five Yes / No questions, each "Yes 1 pt" (the neck one says 17 in / 43 cm and 16 in / 41 cm), and a Clinician **STOP-Bang score** with LOW RISK ≤ 2, INTERMEDIATE RISK ≤ 4 and HIGH RISK ABOVE. |
| T-16 | Pass | A question card has no Copy, Move or Delete buttons. Dragging it out of the Group does nothing. Its Settings show the blue "Ready-made: STOP-Bang" note and greyed-out fields, with only Settings and Disclosures tabs (no Logic). A disclosure on the pressure question's Yes saved. The score's Settings include "Also High risk when: Yes to 2 or more of the 4 STOP questions, and male, a BMI over 35 or a large neck". The Group has move up, move down and delete, but no Copy, and its Logic tab took the condition "Patient's age ≥ 18" (shown as a condition icon on the Group's title). No Add bar appears inside the Group. |
| T-17 | Pass | Step 2, run first while the HQ had no BMI: an orange note says "It uses the patient's BMI. Add a BMI question (or a BMI Calculation) to this Question Set or an earlier one first." Insert is disabled. Step 1, after STOP-Bang was inserted: "This HQ already has STOP-Bang. A ready-made score can be in an HQ only once." |

## Failures in detail

None.

## Found while exploring

- **V-02's card wording:** with the repeated key, the Sleep risk card says "Total of the scores of 3 questions (1 in earlier Question Sets)", because the repeated key now counts as this set's question. That matches "this set's wins", and Publish refuses the HQ anyway. Not a failure.
- **"plus 1 points line"** is the plan's wording, but "1 points" reads oddly. Cosmetic, and the plan expects it.
- **Selection stuck once (not reproduced):** in the STOP-Bang HQ, after the T-16 drag attempt, clicks on other cards (the score, the pressure question) didn't change the selection until I reloaded. I then repeated the drag attempt on a fresh page, and selecting another card worked normally. It might come from the browser tool's synthetic drag. Not counted as a failure.
- **A later Calculation's points menu** offers "Other scores" with the earlier Calculation in the same set. "Sleep risk" itself doesn't list the later "MT V01 score". That's right: it uses no later items.

## Console and network errors not tied to a case

- None in the console on the editor, the Preview, the patient form, Validate or the POA Summary.
- The browser tool reported `503` for `POST /api/logout`, `POST /api/dev/login` and the set-rename `PATCH /chapters/{id}`. These calls worked: the renames were saved (I checked with the API), and sign-in and sign-out worked. With curl, `POST /api/logout` returns **204**, both through :3000 and on :8080. So these are 204 No Content responses that the tool misreports, not server errors.

## Hand checks for Rahul

- None new this round. T-14 at a true 390 px is still not re-run here: the browser can't go below 500 px. J10's patient journey covers it at iPhone size in WebKit.

## GIFs

All in `~/Downloads/`:
- `S38-V01.gif`: a line removed and a line's points changed while another line is unfinished, then the reload.
- `S38-V02.gif`: the editor with a repeated key, both menus, and the Publish refusal.
- `S38-V03.gif`: STOP-Bang refusing `stopbang_snoring`.
- `S38-T10.gif`: the main happy path (Validate Neck 4 (High), then the POA Summary).
