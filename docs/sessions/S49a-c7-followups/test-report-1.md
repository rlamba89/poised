# S49a test report 1

**17 passed, 0 failed, 0 blocked** (12 test cases + 5 regression cases).

- **Code:** `a23b907`, with the S49a changes uncommitted in the working tree (23 changed files: not clean, as expected before verification).
- **When:** 14 Nov 2026, 15:52–16:20 IST.
- **Where:** http://localhost:3000, Chrome (Claude in Chrome). `make dev` had been restarted at 15:52.
- **Data:** all test data was made up through the UI: episodes "MT S49a 1600–1645 …", mobiles 07700 900161–169, emails `mt-s49a-16xx@example.com`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Typed BMI 20.8, saved, reloaded, then changed to 22: the weight stays 60 "Hospital weight". After a second save and reload it shows 22 / 60. Changing it to 19 then **Undo changes** returns 22 / 60. GIF `S49a-T01.gif`. |
| T-02 | Pass | Height 170 + weight 60 open MUST at 20.8 "Hospital BMI" and 60. Typing 50 empties the BMI (Missing, Incomplete, Confirm greyed). After Undo, typing 17.3 empties the weight. After a save, a reload and a change to 18, the weight is still Missing. |
| T-03 | Pass | Viewport **500×701**, not 390×844: Chrome won't make the window narrower than 500 px. No up/down arrows, whether hovered or focused (`appearance: textfield`). The wheel down and up leaves 70 and the page scrolls. Up×2 and Down×3 leave 70. 165 and 70 are kept after a reload. 150 is refused with "The value should not be greater than 100"; 6 lets Next through. Typing `e` and `+` does nothing. "Thank you, Ria", no sideways scroll. GIF `S49a-T03.gif`. |
| T-04 | Pass | In Fill in for the patient: no arrows, and the wheel and arrow keys leave 70. |
| T-05 | Pass (see note) | **Couldn't be run on T-03's episode:** the Demo risk HQ is always auto-triaged on send, so the set is read-only ("The HQ review is complete…"). Run on a new episode with the HQ "MT S44c units 1615", whose only rule is High risk: the number box held 10, with no arrows, and the wheel and arrow keys left it at 10. Changed to 12, it is kept after a reload ("Patient answered: 10" shown). The save happens only when the box loses focus; see exploring 1. |
| T-06 | Pass | Alex → Demo risk HQ → Preview of "Weight, falls and clots": weight 80 with no arrows, and the wheel and arrow keys leave 80. |
| T-07 | Pass | MUST card: BMI and weight-lost boxes are `type="text"`, with no arrows. The wheel and Up/Down leave 22, and Save stays greyed. |
| T-08 | Pass (real data) | The made-up 900-episode row wasn't in the database (the Setup INSERT hadn't been run). I tested on Cara's own **real** `episode.list` rows from today instead: 4,383 episodes and 4,164 patients each. **Screen:** one row, "Listed episodes · 4164 patients". **CSV:** each request is 6 rows, "Listed episodes (part 1 of 6)" … "(part 6 of 6)", with the same Time, Who, Email, Hospital, Route and Result on each. Each part holds 862 names and 862 ids side by side; every patient id and episode id appears once (4,164 / 4,383 unique). The longest cell is 32,754 characters. Short lists (235 ids) and other rows have no "part". |
| T-09 | Pass | Cara has no Audit menu. Her Audit address shows "Only the trust's admins can see the audit log", and the CSV API returns 403. Max (Trust B admin): Trust B's Audit has no Cara rows, a CSV filtered to Cara is empty (header only), and Trust A's CSV returns 403. |
| T-10 | Pass | Sex Other, born 1 Jan 1950. **PRISMA-7:** "1 of 7 filled: answer the 5 items highlighted. The patient's record has no male or female sex, so this score can't be worked out."; with every other item answered it reads "6 of 7 filled. The patient's record has…", Incomplete, Confirm greyed. **Apfel:** "0 of 4 filled: answer the 3 items highlighted. The patient's…". **STOP-Bang:** "1 of 8 filled: answer the 6 items highlighted. The patient's…". None has a second line repeating it. **Not known:** PRISMA-7 shows the same sentence. |
| T-11 | Pass | Female: PRISMA-7 "2 of 7 filled: answer the 5 items highlighted.", with no sex sentence. DASI on the Other patient: "0 of 12 filled: answer the 12 items highlighted.", with no sex sentence. |
| T-12 | Pass (see note) | At 1456 wide, Morse's Gait list starts at the box's left edge (x 589 = 589) and is as wide as the box (409 px); "Weak" shows in the box. **Step 3:** "6: Mental status" is two buttons, not a menu, so I checked item 3's menu (Ambulatory aid) instead: also lined up. Frailty "1: Level": the list starts at x 748 = the box, is 300 px wide (box 250), and "8: Living with Very Severe Frailty" shows whole in the list and in the box. **Step 5:** the window would no longer resize, so I loaded the episode in a 996 px-wide same-origin frame: Gait's list is still under its box and fully on screen. |
| R-01 | Pass | MUST on Ria's episode (height 170, weight 60, HQ weight lost 6) opens "3 of 4 filled" with Hospital BMI 20.8, Hospital weight 60 and HQ 6. Answering 4: No gives "1 (Medium risk), 9.1% of body weight lost"; Confirmed by Cara Clinician. |
| R-02 | Pass | Female, Demo STOP-Bang HQ sent via Fill in for the patient, hospital height 160 and weight 100 (BMI 39.1). STOP-Bang is pre-filled: S, T, O, P and N from the HQ, B from the Hospital BMI 39.1, A and G from the Profile. 8 of 8, "5 (High risk)"; Confirm works. The POA Summary shows "STOP-Bang 5 (High risk)". |
| R-03 | Pass | In the patient's Review, height 165 → Backspace → 8 = 168 keeps the weight-lost follow-up (6). It's a related check: no follow-up in the Demo risk HQ depends on a number. |
| R-04 | Pass | Audit filters Who and Patient ("MT S49a 1645") work; Export CSV works. Times are in the browser's time zone (IST on screen; `+05:30` in the CSV). |
| R-05 | Pass | /qa/messages shows the new episode's email and text invite, and the link opens the patient's form. |

## Failures

None.

## Found while exploring

1. **Minor (probably not new in S49a): in Validate, "Saved" shows while a changed number hasn't been saved.** In a clinician's Validate copy, a number is saved only when the box loses focus. The patient's form saves while typing (`textUpdateMode = "onTyping"` is set only in `PatientHQ.tsx`).
   - **Steps:** Cara → Val MT S49a 1620 Validate → Alcohol → Validate. Change 10 to 12 and leave the cursor in the box. The header still says "Saved", and no PUT is sent (network log). Reload: the box shows 10 and the change is lost. Press Tab before reloading instead: a PUT returns 200 and 12 is kept.
   - **Why it matters:** T-05 step 3 ("wait for it to save, reload") loses the change unless the tester clicks away first. A clinician could think a change was saved when it wasn't.
   - **Side effect of S49a:** scrolling the wheel over the focused box now blurs it, which does trigger the save.
   - GIF `S49a-T05-validate-saves-on-blur.gif`.
2. **Test plan: T-05 can't run as written.** Every seeded Demo HQ uses the default rules with `noGradeAutoTriage: true`, and none of the Demo HQs with a number question has an answer that gives an ASA grade or a red flag. So a sent Demo risk HQ (even with a fall: Yes) goes straight to Ready for admission, and its sets are read-only. A Validate test needs an HQ without the Low risk rule (I used "MT S44c units 1615") or a seeded answer that holds it back.
3. **Cosmetic: the Preview's "Sample patient → Age" box keeps its up/down arrows**, and Up/Down step it (54 → 55 → 54); the wheel doesn't. The same `NumberInput` is on the HQ test page (`HqTestPage.tsx:167`). It isn't an answer, so it's outside "every number answer", but it sits beside the plain answer boxes.
4. **Cosmetic, not new: the assessment cards' number boxes have no `inputMode`,** so a tablet shows the full keyboard. Observations use `inputMode="decimal"`.
5. **Scale, not new in S49a: the audit CSV for Cara alone was 441 MB (101,255 rows).**
   - Locally, each Episodes list visit logs every id in the hospital (about 4,380). Each page load logs it **twice**, with two identical rows in the same second (likely the dev double fetch).
   - The split works, but a real trust's export of a busy clinician will be huge. Worth noting for C10.
6. **Not new: the Audit page's Patient filter puts the patient's name in the address** (`…/audit?user=…&patient=MT+S49a+1645`), so it lands in browser history and any proxy logs. This is the S11 design; flagging it for privacy review.
7. **Minor, not new: searching Questionnaires for "Demo risk" didn't show "Demo risk HQ"** among the results on the page; dozens of "E2E J12 …" HQs matched by description. Searching "library's height" found it. The result list may be capped.
8. **Environment:**
   - Chrome wouldn't make the window narrower than 500 px, so T-03 ran at 500 px, not 390 px.
   - Later in the session the window stopped resizing at all, so T-12 step 5 used a 996 px frame.

## Console and network errors

None. No console errors or warnings were captured, and every `/api/` request returned 2xx, apart from the 403s that T-09 expected.

## GIFs

Saved in `~/Downloads`:
- `S49a-T01.gif` (main happy path: MUST BMI saved, reloaded and changed)
- `S49a-T03.gif` (patient's form: no arrows, wheel, arrow keys, the 100 limit, send)
- `S49a-T05-validate-saves-on-blur.gif` (exploring 1)
