# S48 test report 1: DASI, PRISMA-7 and Apfel

**17 passed, 0 failed, 0 blocked** (12 cases T-01–T-12, 5 regressions R-01–R-05). Two minor findings and four cosmetic notes came up while exploring (below). None of them is a plan case failing.

- **Commit:** `bb2faa4`, with S48's changes uncommitted in the working tree (46 changed or new files: the builder's work, not yet committed).
- **When:** Fri 6 Nov 2026, 19:18–19:58 IST.
- **URL:** http://localhost:3000 (`make dev`, already running; `/api/health` was ok).
- **Browser:** Claude in Chrome. The window wouldn't resize (it ignored 390×844), so the **phone cases ran in a 390×844 same-origin iframe**. Inside it the page's viewport really is 390 wide (`innerWidth` 390), so media queries apply as on a phone. A real device or a resizable window is still worth a glance.
- **How answers were entered:** some Yes/No rows and SurveyJS radios were clicked with a small in-page helper (real DOM `click()` on the app's own buttons, 80 ms apart so React re-renders between clicks) rather than one mouse click at a time. Each result was also checked on screen. The API was used directly only where noted in exploring.

## Test data (all made up)

| Episode | Patient | Used for |
| --- | --- | --- |
| `3fbe69dc…` | Ann MT S48 1920 Main, F, 1 Jan 1980 (HQ sent on a phone) | T-01–T-04, T-09, T-12 |
| `cf73996c…` | Wren MT S48 1928 Woman46, F, 1 Jan 1980 (not sent) | T-05, T-06 d/e, T-07 b/d/f, R-03 |
| `f1c33ab0…` | Mark MT S48 1931 Man46, M, 1 Jan 1980 (not sent) | T-06 c, T-07 a/c/e |
| `83e980e9…` | Edna MT S48 1933 Woman85, F, 1 Jan 1941 | T-06 a |
| `ef65fd8a…` | Iris MT S48 1934 Woman86, F, 1 Jan 1940 | T-06 b |
| `932e7735…` | Sky MT S48 1936 Other, sex Other, 1 Jan 1980 | T-08 |
| `b28f0663…` | Rita MT S48 1935 StopBang (Demo STOP-Bang HQ, filled in by Cara) | R-01 |
| `748cf0df…` | Neil MT S48 1937 Sleep (Demo sleep HQ, filled in by Cara) | R-02 |
| `b91b4cfb…` | Pia MT S48 1939 PreOp (Demo pre-op HQ, sent on a phone) | R-05 |

The questionnaires are **MT S48 ready 1945** (T-10, published), **MT S48 tamper 1950** (T-11, published) and **MT S48 stopbang 1955** (R-04, draft).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 The patient answers all three on a phone | **Pass** | 390 wide, `scrollWidth` 390 on all three pages. Only the questions show: no score, METs or opioids. Next with nothing answered marks all 12 DASI questions "Response required." The send ends on "Thank you, Ann". |
| T-02 DASI opens complete; METs on the POA Summary | **Pass** | 12 of 12, rows 1–12 with label, wording and source HQ, Yes on 1, 2, 4, 5, 7, 8, 9. Badge "34 (Poor functional capacity), 6.9 METs", all visible. No age or sex note. Confirmed by Cara, kept after reload. On both POA tabs, under Hospital BMI. |
| T-03 PRISMA-7 7 of 7, items 1–2 from the record | **Pass** | 46 and Female from Profile; 3–7 from HQ (Yes on 6); "1 (Not at increased risk)"; "The age is the patient's today." |
| T-04 Apfel 3 of 4, waiting for the opioids | **Pass** | "3 of 4 filled: answer the item highlighted." Item 4 orange and Missing, Incomplete, Confirm greyed. Yes gives source Clinician and "4 (PONV risk about 79%)", then Confirmed. POA: Apfel 4 (79%), PRISMA-7 Not confirmed. The menu then offers only STOP-Bang, and is gone once STOP-Bang is added. |
| T-05 DASI's worked examples | **Pass** | 0 of 12, all highlighted, Confirm greyed. Then 0/2.7, 4.5/3.3, 33.95/6.9, 34 (Poor)/6.9, "34.2, 6.9 METs" with no band, and "58.2, 9.9 METs". Every row Clinician, Confirm enabled. |
| T-06 PRISMA-7's edges | **Pass** | a) 85, no point: 2 (Not at increased risk). b) 86: 3 (Increased risk). c) Male: 3. d) 3 (Increased risk). e) 1 (Not at increased risk). Changed rows become Clinician. |
| T-07 Apfel's worked examples | **Pass** | a 0 (10%), b 1 (21%), c 1 (21%), d 2 (39%), e 3 (61%), f 4 (79%). |
| T-08 Sex Other leaves the sex item missing | **Pass** | PRISMA-7 item 2 and Apfel item 1 are Missing ("—"). Both Incomplete with Confirm greyed, each with the "doesn't say male or female" note. DASI has no note and scores 4.5 / 3.3 METs. |
| T-09 Change after Confirm; reload and back | **Pass** | Row 10 Yes: "Changed: confirm it again", "39.25, 7.6 METs". Undo brings back 34 and "Confirmed by…". After Save and reload: Not confirmed, row 10 Clinician, and the POA says DASI Not confirmed. Confirm, then the POA shows "DASI 39.25, 7.6 METs"; Back returns to the episode with the card as it was. |
| T-10 Insert DASI, PRISMA-7 and Apfel | **Pass** | All four offered, each with its source and licence lines; only STOP-Bang asks for a BMI; Apfel shows its "patient's two questions only" note. Groups as described (DASI points total 58.2). Questions locked, Settings read-only, Disclosures can be added. A second DASI gives "This HQ already has DASI. A ready-made score can be in an HQ only once." Published. |
| T-11 A changed ready-made score can't be published | **Pass** | The snippet was run with `alert` swapped for a return value. The window lists "Question Set 1 · DASI has been changed…" and Publish is disabled. Delete and re-insert, then it publishes. Optional: PRISMA-7 and Apfel give their own messages, and Go's publish route returns 422 with both. |
| T-12 Another hospital or trust can't see them | **Pass** | Val (A2) and Bea (Trust B) both get "You don't have access to this hospital", and the API returns 404. Val with her own hospital in the address: 404 "Episode not found." for the episode, its assessments, and a POST. |
| R-01 STOP-Bang (S47) | **Pass** | 8 of 8, "1 (Low risk)", with the age note before Confirm; the POA shows STOP-Bang 1 (Low risk). |
| R-02 STOP-Bang's neck (S47) | **Pass** | Demo sleep HQ: 7 of 8, N Missing, Confirm greyed; N No gives 8 of 8, "2 (Low risk)". |
| R-03 Observations (S46) | **Pass** | Height 180 (saved with Enter) and weight 120: Hospital BMI 37 on the card and the POA. |
| R-04 Insert STOP-Bang (S38) | **Pass** | "BMI from: BMI Calculator (BMI)", inserted; bands Low ≤ 2, Intermediate ≤ 4, High. |
| R-05 Demo pre-op HQ (S19) | **Pass** | Sent on a phone, both sections, `scrollWidth` 390 on every page, "Thank you, Pia". |

## Failures

None.

## Found while exploring

1. **Minor (data integrity): Go accepts a profile value that contradicts the record.**
   - **Steps:** Sky MT S48 1936 Other (episode `932e7735…`, sex Other), as Cara. Save PRISMA-7 with items 3–7 answered. Then send by hand `POST …/assessments/91d043b9-…/confirm` with `data.patientSex: "male"` and `sources.patientSex: "profile"`.
   - **Expected:** refused, because the record says Other.
   - **Actual:** 200. The card now shows item 2 **Male · Profile**, "4 (Increased risk of frailty)", confirmed. The POA Summary shows **PRISMA-7 4 (Increased risk of frailty)**.
   - **Why it matters:** a confirmed score appears whose "Profile" source the record doesn't back. Without the sex at all, the same Confirm is refused (400 "Answer every item before confirming: Sex has no answer."), so Go checks the item is there but not its value. It needs a hand-made request from a clinician who is already signed in. CLAUDE.md says Go only checks the shape, so this may be accepted as designed. Worth a decision: should Go check a `profile` age or sex against the record? `patientAge` is the same.
2. **Minor (wording): "answer the item highlighted" when the item can't be answered.** For sex Other, PRISMA-7 says "6 of 7 filled: answer the item highlighted." and Apfel "3 of 4…", and under the card "Confirm once every item has an answer." There is no control for the highlighted sex item. The last note does explain why ("The patient's record doesn't say male or female…"), but the headline asks for something the clinician can't do here. Possibly the same as S47's STOP-Bang G; not re-checked.
3. **Cosmetic: the badge shows "METS".** The badge styling is all capitals, so it reads "34 (POOR FUNCTIONAL CAPACITY), 6.9 METS"; the POA Summary has "METs". The unit's lower-case "s" is lost in capitals.
4. **Cosmetic: the POA Summary's assessment values sit to the right of the other values.** Under Hospital BMI, DASI, PRISMA-7 and Apfel's values start about 25 px further right than "Not measured yet…" and the rest of that column (seen at 1456 wide).
5. **Observation: unsaved changes are lost without a warning.** Changing assessment rows, then reloading or opening another page, drops them silently, with no "unsaved changes" prompt. While on the page, adding another assessment or confirming another card keeps the unsaved changes. Probably as before for these cards; noted only.
6. **Observation: option points reach the patient's browser.** `GET /api/p/episodes/{id}` includes each choice's `score` (e.g. 2.75 on DASI's Yes), though the score elements themselves are stripped and nothing is shown. Probably as for STOP-Bang; noted only.
7. **Observation: a Poor DASI doesn't stop auto-triage.** The T-01 episode (DASI 34, Poor functional capacity) was auto-triaged by "Low risk" straight to Ready for admission. The default rules don't know DASI yet, and rules are out of scope here; noted for S49 or the rules work.
8. **Checked and fine:**
   - A second add of the same assessment: 409 "DASI is already on this episode."
   - Two adds at once: one 201, one 409.
   - An unknown assessment: 400.
   - Adding an assessment with the keyboard (arrow keys and Enter) works.
   - Adding or confirming one card keeps the other cards' unsaved answers.
   - A partial Save on an Incomplete card works.
   - A confirmed card's note becomes "The age is the patient's on the day it was confirmed (46)."

## Console and network errors

No console errors or warnings on the staff tab. Every 4xx in the network log came from my own probes and was expected: Val and Bea's 404s, the 409/422 publish probes, and the duplicate-add 409s. There were no 5xx.

## GIFs

`~/Downloads/S48-T01-patient-phone.gif`: T-01, the patient's happy path at 390 wide (start → required check → three pages → send → Thank you). There were no failures, so no other GIFs.
