# S44b test report 2 (retest)

**14 passed, 0 failed, 0 blocked.** F1 from report 1 is fixed. The changed cases (T-08, T-13), the new case (T-15) and all five regressions pass.

- **Environment:** `eb1617e` plus the uncommitted S44b work (git status not clean: the S44b changes). Tested 28 Oct 2026, 00:48–00:59, at http://localhost:3000. The conductor ran `make dev` (API :8080, web :3000, evaluate :8090; `/health` reported `clinical@eb1617e71c5e`). Claude in Chrome: a desktop tab at 1512×861 and a second tab stuck at 500×757, which I used for the narrow checks and the patient's steps.
- **Signed in as:** Cara Clinician (Hospital A1) throughout.
- **Test data:**
  - **HQ:** I reused **"MT S44b 0003" v1**, which I built and published through the UI in round 1 (T-01, passed then). It has the plan's two questions and disclosures and the default rules, so I didn't rebuild it.
  - **Patients** (all born 17/05/1980, made up):
    - **H** "Sam MT44b H 0050": episode `f7e3be0b-8e98-41e6-9bb1-36388b6ab080`, at Ready for POA.
    - **H3** "Sam MT44b H3 0053": `0162fa12-21a8-4020-8034-d10bc0860904`, at Ready for admission.
    - **L** "Sam MT44b L 0054": `eeb846c8-ff5b-4f3a-ad2c-c07f016b6191`, at Ready for admission.
- **Retest scope:**
  - **Re-run:** the case that failed (T-14), the changed cases (T-08, T-13) and the new case (T-15).
  - **Happy paths:** T-02 to T-05, plus T-06 step 2.
  - **Regressions:** R-01 to R-05.
  - **Not re-run:** T-07, T-09 (UI part), T-10, T-11 and T-12. They passed in round 1, and the fixes don't touch them. I re-checked T-09's API refusal while exploring.
- **Print (T-14):**
  - **How:** I rendered H's POA Summary with Playwright, signed in as Cara (`emulate_media("print")`, then `page.pdf()`).
  - **The PDF:** `/private/tmp/claude-501/-Users-rahul-poised/c92da616-784f-4500-966e-bd8e6e97afd6/scratchpad/s44b/poa-retest.pdf`.
- **Phone size:** as in round 1, the window wouldn't resize to 390×844, so the patient's three sends ran at 500 px. All three worked.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 high-risk episode, reviewed | Pass | H is at Ready for POA. All of these show: the red requirement line; "No ASA grades have been added." (correct here, since nothing is listed); the Nurse dropdown with Save; "Anaesthetist ASA grade: No grade yet" with no history; and an Add box with only the grade dropdown. The top details say "Not recorded". |
| T-03 first grade, no reason | Pass | With no grade chosen: "Choose an ASA grade.". Adding ASA III shows it with Counts and "Given by Cara Clinician on 28/10/2026, 00:51" (no reason), and the line turns grey. The note says "Anaesthetist ASA grade set to 3", and the reason box appears. |
| T-04 re-grade needs a reason | Pass | With no reason: "Say why the grade is changed.", and ASA IV stays chosen. With a reason, ASA IV counts with ": New echo result" and ASA III stays below. The note says "…changed from 3 to 4: New echo result". |
| T-05 mark the latest entered in error | Pass | With an empty reason: "Say why the grade was entered in error.". After marking, ASA IV is struck through with "Entered in error: marked by Cara Clinician on …: Typed in the wrong box" and has no button. ASA III counts again, and the note is right. All the same after a reload. |
| T-06 step 2 (spot check) | Pass | H3 at Ready for admission with only ASA III counting: marking it gave "Add the correct grade first, or move the episode back from Ready for admission." in the form, and ASA III still counts. |
| **T-08** before Ready for admission the only grade can be marked | Pass | H3: I added ASA II and marked it ("Wrong patient"). It's struck through, "No grade yet" shows, and the requirement line is red again. **No "No ASA grades have been added." line** (checked in the page text: absent). The Ready for admission option is greyed with "(add the anaesthetist's ASA grade first)", and clicking it does nothing. ASA III was then added with **no reason box**, and the move to Ready for admission worked. |
| **T-13** no rule requires it | Pass | L was auto-triaged by "Low risk" with no requirement line. I added ASA II, then marked it ("Wrong patient"; Enter submits) at Ready for admission. "Anaesthetist ASA grade: No grade yet" shows, and **no "No ASA grades have been added." line**, including after a reload. |
| **T-14** print and narrow window | **Pass** (was Fail) | **Print:** "Anaesthetist ASA grade: ASA III", then ASA III (Counts), ASA IV struck through (with its entered-in-error line) and ASA III. No Add box, no Entered in error buttons. **At 500 px:** both Entered in error buttons read in full, including beside the "Given by … 00:52: Second anaesthetist agrees" line, which wraps to two lines. Each button is 122 px wide; the label's scrollWidth and clientWidth are both 92, so it isn't clipped. The page doesn't scroll sideways (document width 500 = viewport). The Add box wraps cleanly. F1 is fixed. |
| **T-15** the same grade again is a confirmation | Pass | On H, with ASA III counting (after IV was marked), I added ASA III with "Second anaesthetist agrees". It's accepted, counts, and the earlier ASA III stays below. The General notes say "**Anaesthetist ASA grade confirmed as 3: Second anaesthetist agrees**". All the same after a reload and in print. |
| R-01 nurse grade | Pass | On H3 I set ASA II and saved, then cleared it with × and saved. The notes say "Nurse ASA grade set to 2" and "Nurse ASA grade cleared". (The "change" step passed in round 1, and I didn't repeat it.) |
| R-02 high-risk send actions | Pass | H shows the task "Book anaesthetic review" (Anaesthetist, "Added by rule: High risk"), the red Anaesthetic review flag, the red line on the episode page ("… Add it on the POA Summary.") and the three automatic notes. Clear with "Review booked" worked: the flag reads "Cleared by Cara Clinician on … Review booked". |
| R-03 low-risk auto-triage | Pass | L is at Ready for admission with the "Auto-triaged by the rule “Low risk”" card. Checked gives "Checked by Cara Clinician …", and the status history shows the auto-triage line. |
| R-04 suggested ASA on both tabs | Pass | L's Validated summary and Patient answers tabs both show Suggested ASA II with its reason. The Patient answers tab has no nurse or anaesthetist grades. |
| R-05 Undo after Admitted | Pass | On H3 (ASA III counting) I moved Ready for admission → Admitted (admitted 27/10), then used Undo back to Ready for admission ("Admitted by mistake"). It worked, and the history marks the Admitted row "(undone)". |
| T-09 API (exploring) | Pass | PATCH on the episode with `anaesthetistAsa` 2 or null, plus the revision, gives 400 with the exact message. DELETE `/anaesthetist-asa/{id}` gives 404, and GET `/anaesthetist-asa` gives 405. |

The count of 14 is every row above except the T-09 exploring row. T-03 to T-05 also ran as the main happy path.

## Failures

None.

## Found while exploring

1. **A long unbroken reason** (115 characters, no spaces) breaks across lines inside the history entry at 500 px. The Entered in error button beside it stays full width, and nothing overflows. Enter in the reason box submits Add grade.
2. **PATCH without `revision`** (and with `anaesthetistAsa`) gives 400 "The request could not be read." rather than the specific anaesthetist message. It's only reachable by hand-written API calls, and with a revision the specific message comes back. Not a defect, just noted.
3. **The extension's network log shows 503 for every 204 No Content response:** complete-review, entered-in-error, status PATCH and flag clear. Each of these actions worked in the UI and stayed after a reload. The Go handlers write `http.StatusNoContent`, and the evaluate service's `/health` returned ok. This is a tool artefact, not an app error.

## Console and network errors

- **Console:** no errors or warnings in either tab (read with the pattern `error|Error|warn`).
- **Network:** every GET returned 200. The only 4xx were the expected ones: the 409 from T-06's refused mark, and my deliberate API probes. The "503"s are the 204 artefact explained in item 3 above.

## Hand checks for Rahul

- **The patient's steps at 390×844:** the window won't resize, so they ran at 500 px. They aren't changed by S44b.

## GIFs

- `~/Downloads/S44b-retest-happy-path.gif`: T-03 to T-05 and T-15 on H (first grade, re-grade, entered in error, "confirmed as", reload).
