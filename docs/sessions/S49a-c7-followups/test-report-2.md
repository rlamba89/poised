# S49a test report 2 (retest)

**13 passed, 0 failed, 0 blocked** (9 test cases re-run + 4 regression cases).

- **Code:** `a23b907`, with the S49a changes and the verification 1 fixes uncommitted in the working tree (not clean, as expected before verification).
- **When:** 14 Nov 2026, 18:22–19:25 IST.
- **Where:** http://localhost:3000, Chrome (Claude in Chrome). `make dev` was restarted at 18:22, after the fixes' last edits (`audit.go` 17:46, `EarlierAnswers.tsx` 17:55), so the API ran the new Go.
- **Data:** all test data was made up:
  - Through the UI: the draft "MT S49a 1830 earlier numbers"; episodes "Val MT S49a 1840 Validate" (07700 900271, `mt-s49a-1840@example.com`) and "Mia MT S49a 1850 Must" (07700 900272, `mt-s49a-1850@example.com`).
  - The Setup's 900-episode audit row: it wasn't in the database, so I ran the plan's INSERT myself (with `psql`, the same statement) at 18:23.

**Scope:** report 1 had no failures or blocked cases. This round re-ran:
- the cases the builder changed after verification 1 (T-06 step 3 new, T-05 rewritten, T-12 step 3);
- the audit CSV (T-08, T-09), because its Go changed (finding 3);
- the main happy paths (T-01, T-04, T-07, the patient's form);
- the regressions touched on the way.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Weight 60 only. MUST opens BMI Missing, weight 60 "Hospital weight". Typed BMI 20.8 ("Entered by clinician"), weight unchanged; saved; after a reload 20.8 / 60. **Changed to 22: the weight stays 60 "Hospital weight"** (the fix). Saved, reloaded: 22 / 60. 19 then **Undo changes**: back to 22 / 60. GIF `S49a-retest-T01.gif`. |
| T-04 | Pass | Fill in for the patient ("MT S44c units 1615"): typed 70, the wheel down and up and Up×2 / Down×3 leave 70; `appearance: textfield`, no arrows. |
| T-05 | Pass | Run as the changed plan says, on an HQ that isn't auto-triaged: "MT S44c units 1615" (only a High risk rule), sent via Fill in for the patient. The episode stayed **Ready for review**. Validate → Alcohol: 10, no arrows; the wheel and Up/Down leave 10. Changed to 12 and pressed **Tab**: one PUT, 200. After a reload it shows 12, with "Patient answered: 10". |
| T-06 step 3 | Pass | **New case (finding 2).** Built in the editor as Alex: draft "MT S49a 1830 earlier numbers". Set 1 has "Drinks a week" (Number, page 1) and a BMI (page 2). Set 2 has a Calculation with "Add 1 point when Drinks a week > 10" and "Add 1 point when BMI Calculator (BMI) > 20", all from the menus. Set 2's Preview (Clinician): the blue box lists both earlier questions as plain boxes (`type="text"`, `inputMode="decimal"`, no arrows). Typed 14 and 20.8; Up×2, Down×3 and the wheel down and up leave **14** and **20.8**, and the page scrolls. The score is **2**; with 9 it's 1. `abc`, `1e3` and `0x10` show "A number" under the box and the score goes Incomplete; ` 12.5 ` (spaces) is taken as 12.5. GIF `S49a-T06-earlier-answers.gif`. |
| T-07 | Pass | MUST card: BMI, Weight now and Weight lost are `type="text"` with no arrows; Up/Down and the wheel leave 22. |
| T-08 | Pass | **With the plan's made-up 900-episode row.** As Ada, Who = Cara: on screen one row, "Listed episodes · 900 patients". **Export CSV** (From 14/11/2026, 90 MB) has two rows, "Listed episodes (part 1 of 2)" and "(part 2 of 2)", with the same Time, Who, Email, Hospital, Route and Result. Part 1 holds 862 names / ids / episode ids, part 2 holds 38. Together that's **900 patient ids and 900 episode ids, none repeated**; the episode ids match the database row's exactly. The longest cell in the file is 32,754 UTF-16 units (≤ 32,767). Rows that aren't lists have no "part". |
| T-09 | Pass | Max Multi: Trust B's Audit shows neither Cara nor a "900 patients" row; Trust B's CSV filtered to Cara is header only. Trust A's CSV is **403** "Only the trust's admins can see the audit log." |
| T-12 steps 2–3 | Pass | Morse on Mia's episode, at a 1512 px viewport. **3: Ambulatory aid** (step 3, as changed): the list's left edge is 617 = the box's 617, the same width (409 px), 8 px under the box. Picked "Crutches, a stick or a walking frame": it shows in the box. **5: Gait**: the same alignment; picked **Weak**, which shows in the box. 6: Mental status is two buttons. (Steps 4–5 passed in report 1 and weren't touched since.) |
| Patient's form (T-03 subset) | Pass | Mia's invite → date of birth → Demo risk HQ → "Weight, falls and clots": weight 70, Up/Down and the wheel leave 70, no arrows. At desktop width only: see the environment note. |
| R-04 | Pass | Audit filters Who and From work and add `user` and `from` to the address; Export CSV works; times in the CSV are `+05:30` (the browser's zone). |
| R-05 | Pass | /qa/messages shows Mia's email and text invite (and Val's), and the link opens her Demo risk HQ after the date of birth. |
| R-01, R-02, R-03 | Not re-run | Passed in report 1. The verification fixes didn't touch the MUST pairing, STOP-Bang or the patient form's saves. T-01 above re-checks the MUST card. |

## Failures

None.

## Found while exploring

1. **Cosmetic, not new: the editor's BMI button is greyed out with no reason given.** A BMI must be alone on its page (`AddContentBar.tsx`), so on a page with any question the button is just disabled. Hovering shows no tooltip. An author has to guess that a new page is needed.
2. **Cosmetic, not new: the assessment cards' number boxes still have no `inputMode`** (MUST BMI, Weight now, Weight lost: `inputMode` is empty), so a tablet shows the full keyboard. Verification 1's finding 2 says the Preview's new boxes use `inputMode="decimal"`, "as the assessment cards do", but the cards don't. Same as report 1, exploring 4.
3. **Cosmetic: the Preview's earlier-answers error reads just "A number"** under a box holding `abc`. Other number checks in the app say a full sentence. It's in the author's Preview only.
4. **Scale, not new (report 1, exploring 5): today's audit CSV for Cara alone was 90 MB, 12,070 rows.** Most of it is "Listed episodes" rows split into 5 or 6 parts each (1,093 part rows today), from the e2e runs and Episodes list visits.

## Environment notes

- **Chrome wouldn't resize the window to 390×844** (the resize call reports success; the viewport stayed 1512 wide), so I checked the patient's form at desktop width. Report 1 ran T-03 at 500 px, and J3b covers 390×844.
- **The viewport changed size twice mid-session** (1000 → 1512 → 1456 wide), apparently from outside the test. One click landed off a dialog button because of it; after the retry, the dialog's button was clicked by reference.

## Console and network errors

None. No console errors or warnings were captured on the pages checked. Every `/api/` request returned 2xx, except:
- the 403 that T-09 expected;
- one 400 "Enter valid dates." from a hand-typed CSV URL of mine (`from=2026-11-14` without a time). That was my URL, not the page's: the page's own Export CSV worked.

## GIFs

Saved in `~/Downloads`:
- `S49a-T06-earlier-answers.gif`: the new case. Earlier answers 14 and 20.8 survive the arrow keys and the wheel; the score follows; odd input.
- `S49a-retest-T01.gif`: main happy path. The MUST BMI is saved, reloaded and changed, and the hospital weight stays.
