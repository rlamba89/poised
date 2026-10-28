# S44b test report 1

**18 passed, 1 failed (cosmetic), 0 blocked.**

- **Environment:** `eb1617e` plus the uncommitted S44b work (git status not clean: the S44b changes, as listed by the conductor). Tested 28 Oct 2026, 00:02–00:25, at http://localhost:3000 (`make dev` run by the conductor, freshly migrated and seeded). Claude in Chrome, desktop window 1512×861. A second tab opened at 500×757, so T-10 and T-14's narrow check used it.
- **Test data:**
  - HQ **"MT S44b 0003"** (v1 published).
  - Patients (all born 17/05/1980, made up):
    - **H** "Sam MT44b H 0010": episode `96fb54c3…`, now archived.
    - **H2** "Sam MT44b H2 0013": `5bb243bf…`, now cancelled.
    - **H3** "Sam MT44b H3 0015": `ef5c683c…`, at Ready for admission.
    - **L** "Sam MT44b L 0021": `f34f78a7…`, at Ready for admission.
- **Deviation:** the window couldn't be resized to 390×844. `resize_window` reported success, but the viewport stayed at 1512 px. So the patient's steps (open the invite, date of birth, answer, send) ran at desktop width. The patient side isn't S44b's subject, and every send worked.
- **API checks (T-09, T-11, and exploring):** I ran these with `fetch` from the browser's console while signed in, as the plan says.
- **Print (T-14):** I rendered the print layout to a PDF with Playwright (`emulate_media("print")` then `page.pdf()`, signed in as Cara). The file is at `/private/tmp/claude-501/-Users-rahul-poised/c92da616-784f-4500-966e-bd8e6e97afd6/scratchpad/s44b/poa.pdf`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 build and publish the test HQ | Pass | Two required Select One questions, each with its disclosure: Smoker 77176002 (ASA II from the guide, Lifestyle) and I50.9 (ASA III chosen, Cardiovascular). A clinician set "Nurse checks". Published with the default rules. |
| T-02 high-risk episode, reviewed | Pass | The episode is at Ready for POA. The red requirement line, the Nurse dropdown with Save, "Anaesthetist ASA grade: No grade yet", no history, and an Add box with only the grade dropdown all show. The top details say "Not recorded". The status option "Ready for admission (add the anaesthetist's ASA grade first)" is greyed. |
| T-03 first grade, no reason | Pass | Clicking Add with no grade gives "Choose an ASA grade." Adding ASA III gives the Counts badge and "Given by Cara Clinician on …" with no reason. The line turns grey ("…is added, as the rule “High risk” asks."). The top details say ASA III, and the notes say "Anaesthetist ASA grade set to 3". The reason box appears. All the same after a reload. |
| T-04 re-grade needs a reason | Pass | With no reason: "Say why the grade is changed.", and ASA IV stays chosen. With a reason, ASA IV is listed first with Counts and ": New echo result", and ASA III stays below with its own button. The top details say ASA IV. The note says "…changed from 3 to 4: New echo result". |
| T-05 mark the latest entered in error | Pass | With the box empty: "Say why the grade was entered in error.", and Back closes the form. After marking, ASA IV is struck through, badged "Entered in error", shows "Entered in error: marked by Cara Clinician on …: Typed in the wrong box", and has no button. ASA III counts again. The note says "Anaesthetist ASA grade 4 marked entered in error: Typed in the wrong box". All the same after a reload. |
| T-06 the only grade can't go at Ready for admission, Admitted or Discharged | Pass | The move to Ready for admission worked. Marking ASA III was refused at Ready for admission, at Admitted (admitted yesterday) and at Discharged, each time in the form: "Add the correct grade first, or move the episode back from Ready for admission." ASA III still counted each time. |
| T-07 the latest can go when an earlier grade falls back | Pass | H2 got ASA III, then ASA IV ("Second opinion"), then moved to Ready for admission. Marking IV ("Wrong patient") worked and III counted again. Marking III was then refused with the expected message. Enter submits the form. |
| T-08 before Ready for admission the only grade can be marked | Pass | Marking ASA II worked: it's struck through, "No grade yet" shows, and the line is red again. The status option is greyed, and clicking it does nothing. ASA III was added with no reason box ("set to 3"), and the move to Ready for admission then worked. |
| T-09 no edit, no delete | Pass | The UI offers only Add grade and Entered in error. PATCH with `anaesthetistAsa` 2, 0 and null each gave 400 with the exact message. DELETE, PATCH and PUT on `/anaesthetist-asa/{id}` gave 404. The grades, the revision (6) and the General notes didn't change. |
| T-10 two tabs | Pass | Tab 1 added IV ("Tab one"). In tab 2, adding V ("Tab two") gave "This episode changed since you opened it. Reload to see the latest; your change was not saved." with a Reload button, and the typed values stayed. After Reload, tab 1's IV showed and V/"Tab two" were still in the form. Add then worked: V counts and IV sits below it. |
| T-11 other hospitals and trusts | Pass | Val and Bea each saw "You don't have access to this hospital". Every POST gave 404: via A1 "Hospital not found.", and via their own hospital "Episode not found.". Cara then saw H's grades unchanged. |
| T-12 cancelled and archived are read-only | Pass | Cancelled H2 and archived H (archived by Sue) show the history, struck-through entries included, but no Add box and no Entered in error. The grade that counts is kept. The API also refuses both actions with 409 ("This episode is cancelled/archived, so it can't be changed…"). |
| T-13 no rule requires it | Pass | L was auto-triaged to Ready for admission with no requirement line. ASA II was added, and marking it entered in error worked at Ready for admission. "Anaesthetist ASA grade: No grade yet" shows. |
| T-14 print and narrow window | **Fail (cosmetic)** | Print is right: the counting grade and the history with the strike-through show, and the Add box and buttons don't. At 500 px the Add box wraps well, but the **Entered in error** button is cut to "Entered in err" on rows whose given-by line wraps. See F1. |
| R-01 nurse grade | Pass | Set ASA II and saved, then ASA III and saved, then cleared with × and saved. The notes say "Nurse ASA grade set to 2", "…set to 3" and "Nurse ASA grade cleared". |
| R-02 high-risk send actions | Pass | The task "Book anaesthetic review" (anaesthetist), the red Anaesthetic review flag, and the red line on the episode page ("… Add it on the POA Summary.") all appear. Clearing the flag with a reason worked ("Cleared by … Review booked"). |
| R-03 low-risk auto-triage | Pass | The episode is at Ready for admission with the "Auto-triaged by the rule “Low risk”" card. Checked works ("Checked by Cara Clinician …"). |
| R-04 suggested ASA on both tabs | Pass | Suggested ASA II with its reason shows on both tabs. The Patient answers tab has no nurse or anaesthetist grades. |
| R-05 Undo after Admitted | Pass | On H (ASA III counting), Undo back to Ready for admission with a reason worked, and so did admitting again. |

## Failures

### F1 (Cosmetic): the "Entered in error" button is cut off at about 500 px

- **Steps:**
  1. Sign in as Cara Clinician and open H3's POA Summary (`/o/…fa/h/…0a/episodes/ef5c683c-6adf-4a4b-9215-05666eb1ac22/poa`) in a window about 500 px wide.
  2. Look at a history entry whose "Given by Cara Clinician on 28/10/2026, 00:18: Tab one" line wraps, which is any entry with a reason.
- **Expected:** the button reads "Entered in error" in full, as it does on the entries without a reason. The plan says nothing should be "cut off badly".
- **Actual:** the button shrinks to about 111 px, and its label is clipped to "Entered in err" (label scrollWidth 92 against clientWidth 81). It sits a little further right than the full-width button on the next row. The page doesn't scroll sideways (document width 500 = viewport).
- **Likely cause:** the button isn't kept from shrinking in the entry's flex row (it needs `flex-shrink: 0`, or Mantine's `style={{ flexShrink: 0 }}`). This is the same class of problem as pending.md's known "Cle" Clear-button item.
- **Console/network:** none.
- **GIF:** `~/Downloads/S44b-T10-T14-narrow.gif` (its last frames show it).

## Found while exploring

1. **Re-adding the grade that already counts is allowed, and the note reads "changed from 3 to 3".** On H I added ASA III again with a reason while ASA III counted. The new entry counts, and the General notes say "Anaesthetist ASA grade changed from 3 to 3: 🫀 Post-op review". This might be intended, for example a second anaesthetist confirming the grade. If so, the wording could be "confirmed as 3". Otherwise the Add could refuse a grade equal to the one that counts. Neither the brief nor the plan covers this. **Minor, for Rahul to decide.**
2. **"No ASA grades have been added." shows above a history that has a struck-through grade** (H3 and L after marking their only grade). It's literally true, since no grade counts and there's no nurse grade, but it sits just above a listed (struck) anaesthetist's grade. Something like "No ASA grade counts yet." might read better. **Cosmetic.**
3. **Double-clicking Add grade added only one grade.** Correct.
4. **A reason with leading and trailing spaces and an emoji** ("   🫀 Post-op review   ") was trimmed and shown correctly, both in the history and in the notes.
5. **API edge cases all behave:**
   - Grade 0 or 7 gives 400 "Choose an ASA grade from 1 to 6."
   - A string grade gives 400.
   - No reason, or only spaces, while a grade counts gives 422 "Say why the grade is changed."
   - A stale revision gives 409 stale.
   - Marking an already-marked grade gives 409 "This grade is already marked entered in error."
   - A random or invalid grade id gives 404 "Grade not found."
   - Another episode's grade id under H3's path gives 404, and that grade is unchanged.
   - A 2,000-character reason gives 400 "The reason is too long (1,000 characters at most)." The reason box has `maxlength=1000`.
   - A 5,000-character reason gives 400 "The request could not be read.", which is a generic message for an oversize body. You can't reach it from the UI.
6. **Back and Forward between the episode and its POA Summary** reload cleanly. Every API call returned 200, and the console showed no errors or warnings.
7. **Print also shows screen-only controls:** the tabs, Add comment, and the empty Nurse ASA dropdown with its "ASA grade" placeholder. The first two are already in pending.md. The empty nurse dropdown in print is older than S44b, and the top details already say "Not recorded".

## Hand checks for Rahul

- **The patient's steps at 390×844:** the window wouldn't resize, so the four patient sends ran at desktop width. These steps aren't changed by S44b.

## Console and network errors

None seen. I read the console with the pattern `error|Error|warn` and the API requests after reloads and Back/Forward: all GETs returned 200, and no unexpected 4xx or 5xx appeared. Only the deliberate API probes above returned 4xx.

## GIFs

- `~/Downloads/S44b-happy-path.gif`: T-03 to T-05 (first grade, re-grade, entered in error, reload).
- `~/Downloads/S44b-T10-T14-narrow.gif`: T-10 in the 500 px tab (stale message, Reload, add), ending on F1's clipped buttons.
