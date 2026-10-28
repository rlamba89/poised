# S44b test report 3 (retest after verification 1)

**15 passed, 0 failed, 0 blocked.** The changed rule passes in the UI and the API: every grade after the episode's first needs a reason, even when the only earlier grade was entered in error (T-08 step 4, T-13 step 3). The database refuses deleting a grade. The happy path and all five regressions still pass.

- **Environment:** `eb1617e` plus the uncommitted S44b work (git status not clean: the S44b changes and verification 1's fixes). Tested 28 Oct 2026, 01:23–01:35, at http://localhost:3000. The conductor ran `make dev` (API :8080, web :3000, evaluate :8090), freshly migrated and seeded; `/api/health` returned ok.
- **Browser:** Claude in Chrome. A staff tab at 1499×812, and a second tab for the patient's steps. The window wouldn't resize to 390×844 (as in rounds 1 and 2), so the patient's sends ran at desktop width.
- **Signed in as:** Cara Clinician (Hospital A1) throughout.
- **Test data:**
  - **HQ:** I reused **"MT S44b 0003" v1**, which I built and published in round 1 (T-01 passed then). It's still in the database after the reseed.
  - **Patients:** all made up, born 17/05/1980, each with a new episode created through the UI and sent through the patient's invite link.
    - **H** "Sam MT44b H 0125": `f8863f99-37cb-4b93-bc25-98ea39659bf7` (heart failure Yes).
    - **H3** "Sam MT44b H3 0128": `5da19b29-6cb8-4340-90f6-48bd12e1fed6` (heart failure Yes).
    - **L** "Sam MT44b L 0129": `a247b0ec-91bf-43c4-b44f-70e23ea1a524` (smoker Yes, heart failure No).
- **Retest scope:**
  - **Changed cases:** T-08 and T-13, which verification 1 changed.
  - **Main happy paths:** T-02 to T-05 and T-15, plus T-06's refusal at Ready for admission and at Admitted.
  - **Regressions:** R-01 to R-05.
  - **T-09:** re-checked through the API.
  - **Not re-run:** T-01, T-07, T-10 (UI), T-11, T-12 and T-14. They passed in earlier rounds, and verification 1's fixes don't touch them. A stale-revision POST still gives 409.
- **Database check (confirming only):** in rolled-back transactions, `DELETE` and `UPDATE` on `anaesthetist_asa_grades` were refused by the triggers. Details are in item 2 under "Found while exploring".

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 high-risk episode, reviewed | Pass | H is at Ready for POA. The page shows the red requirement line, the Nurse dropdown with Save, and "Anaesthetist ASA grade: No grade yet" with no history. The Add box has only the grade dropdown (no reason box). The top details say "Not recorded". |
| T-03 first grade, no reason | Pass | With no grade chosen: "Choose an ASA grade.". ASA III is added with Counts and "Given by Cara Clinician on 28/10/2026, 01:26" (no reason). The line turns grey, the note says "Anaesthetist ASA grade set to 3", and the reason box appears. All the same after a reload, with the top details at ASA III. |
| T-04 re-grade needs a reason | Pass | With no reason: "Say why the grade is changed.", and ASA IV stays chosen. With a reason, ASA IV counts with ": New echo result" and ASA III stays below. The note says "…changed from 3 to 4: New echo result". |
| T-05 mark the latest entered in error | Pass | With an empty reason: "Say why the grade was entered in error.". After marking, ASA IV is struck through (line-through on its grade and given line) with "Entered in error: marked by Cara Clinician on …: Typed in the wrong box", and it has no button. ASA III counts again. The note is right, and all of it is the same after a reload. Enter submits the form. |
| T-06 steps 2 and 4 (spot check) | Pass | On H3 at Ready for admission, and again at Admitted, with only ASA III standing: marking it gave "Add the correct grade first, or move the episode back from Ready for admission." in the form, and ASA III still counts. |
| **T-08** before Ready for admission the only grade can be marked | **Pass** | H3: I added ASA II with no reason box, since it was the first grade, then marked it ("Wrong patient"). It's struck through, the page says "No grade yet", and the red line is back. **No** "No ASA grades have been added." line (checked in the page text). The **reason box now shows** ("Why is the grade changed? \*"). The status dropdown greys out "Ready for admission (add the anaesthetist's ASA grade first)", and clicking it does nothing. ASA III with an empty reason gives "**Say why the grade is changed.**". With "The correct grade" it's added and counts, showing "…01:30: The correct grade", and the note says "**Anaesthetist ASA grade set to 3: The correct grade**". The move to Ready for admission then worked. |
| **T-13** no rule requires it | **Pass** | L was auto-triaged by "Low risk" with no requirement line. I added ASA II (no reason asked) and marked it ("Wrong patient") at Ready for admission. It shows "No grade yet", and **no** "No ASA grades have been added." line. ASA I with an empty reason gives "**Say why the grade is changed.**". The API refuses it too: POST with no `reason`, or with `"   "`, gives **422** "Say why the grade is changed.", and nothing is added. With "Re-checked", ASA I counts, and the note says "Anaesthetist ASA grade set to 1: Re-checked". |
| T-15 the same grade again is a confirmation | Pass | On H, with ASA III counting (after IV was marked), I added ASA III with "Second anaesthetist agrees". It counts, and the earlier ASA III stays below. The note says "**Anaesthetist ASA grade confirmed as 3: Second anaesthetist agrees**". All the same after a reload. |
| T-09 no edit, no delete (API) | Pass | PATCH on the episode with `anaesthetistAsa` 2, 0 or null, plus the revision, gives **400** with the exact message. DELETE, PATCH and PUT on `/anaesthetist-asa/{id}` give **404**. Afterwards the grades and the revision are unchanged. |
| R-01 nurse grade | Pass | On H3: set ASA II and Save, change to ASA III and Save, then clear with × and Save. The notes say "Nurse ASA grade set to 2", "…set to 3" and "Nurse ASA grade cleared", and the top details say "Not recorded". The anaesthetist's grade wasn't touched. |
| R-02 high-risk send actions | Pass | H and H3 show the task "Book anaesthetic review" (Anaesthetist, "Added by rule: High risk"), the red Anaesthetic review flag, the red line on the episode page ("… Add it on the POA Summary.") and the three automatic notes. Clear with "Review booked" worked: "Cleared by Cara Clinician on 28/10/2026, 01:34: Review booked". |
| R-03 low-risk auto-triage | Pass | L is at Ready for admission with the "Auto-triaged by the rule “Low risk”" card. Checked gives "Checked by Cara Clinician on 28/10/2026, 01:33." |
| R-04 suggested ASA on both tabs | Pass | L's Validated summary and Patient answers tabs both show Suggested ASA II (Smoker). The Patient answers tab has no nurse or anaesthetist grade controls. |
| R-05 Undo after Admitted | Pass | On H3 (ASA III counting) I moved Ready for admission → Admitted (28/10), then used Undo back to Ready for admission ("Admitted by mistake"). It worked, and the history shows "Ready for admission → Admitted (undone)" and "Undone: Admitted → Ready for admission". |
| Delete guard (verification 1, finding 3) | Pass | Checked in the database, rolled back. `DELETE FROM anaesthetist_asa_grades …` gives "an anaesthetist's ASA grade can't be deleted", and `UPDATE … SET grade=5` gives "…can't be changed, only marked entered in error once". The counts are unchanged afterwards. |

T-03 to T-05 and T-15 ran as the main happy path (GIF below).

## Failures

None.

## Found while exploring

1. **Double-clicking Add grade** (ASA IV, reason "Double click test 🙂") added **one** grade, not two, and the emoji shows correctly in the history. A POST with a stale revision gives 409 "This episode changed since you opened it…", and grade 7 gives 400 "Choose an ASA grade from 1 to 6.".
2. **The cascade from deleting an episode:** I couldn't check it by hand. My rolled-back `DELETE FROM episodes` stopped first on an unrelated foreign key (`login_links_episode_id_fkey`), so it never reached the grades. The builder's integration test covers the cascade. Not a defect.
3. **A General notes wording, outside this session's scope:** a nurse grade change reads "Nurse ASA grade set to 3", not "changed from 2 to 3". This is the older wording; the plan's Out of scope covers the number wording, and S44b doesn't change it.

## Console and network errors

- **Console:** no errors or warnings, read with the pattern `error|Error|warn` after a reload of a POA Summary.
- **Network:** every GET returned 200. The only 4xx/5xx were the expected ones: the 409 from T-06's refused mark, the client-side "Say why…" (no request), and my deliberate API probes (422, 400, 404, 409).

## Hand checks for Rahul

- **The patient's steps at 390×844:** the window won't resize, so they ran at desktop width. S44b doesn't change the patient's side.

## GIFs

- `~/Downloads/S44b-retest3-happy-path.gif`: T-03 to T-05 and T-15 on H (the first grade, a re-grade, entered in error, "confirmed as", a reload).
- `~/Downloads/S44b-retest3-T08.gif`: T-08 on H3 (the only grade marked entered in error, the reason now required for the next grade, the move to Ready for admission).
