# S32 test report 1: the Clinical Summary element

**28 passed, 0 failed, 0 blocked** (23 cases + R-01 to R-05).

The tester ran in the automatic run (`/run-sessions`).

**Environment**
- **Date:** 25 Oct 2026, 16:49–17:25 BST.
- **Code:** `ba98ac4` with S32's uncommitted changes (40 files changed or new, as the builder left them).
- **App:** `make dev` run by the conductor at http://localhost:3000 (API :8080).
- **Browser:** Chrome, "trk" profile, desktop window 1456×739. Phone-size cases used a same-origin 390×844 iframe.

**Test data** (all made up)
- Questionnaire **S32 test 1650**: v1 published; v2 is a draft, left unpublished on purpose (T-10).
- **Old switch draft 16:49:43** and **Old switch published 16:49:43**, made with the setup snippet.
- Patients Sam MTSthirtytwo-A, -B, -C and -D, all born 17/05/1980, with emails `mt-s32-*@example.test`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | No switch by "Clinical Page". The Add bar has **Clinical Summary**. |
| T-02 | Pass | The card sits inside Breathing, after Asthma? and Inhalers. It has the purple Clinician badge, "Breathing summary" and "Covers the Group “Breathing”". The Settings panel has only Title and a line on what it covers (plus the Logic tab), with no Disclosures, Required or Clinical. |
| T-03 | Pass | "Blood summary / Covers the Section “Blood”" and "Heart summary / Covers the page “Heart”" at the end of the page. "All changes saved" shows. |
| T-04 | Pass | The title changes to "Heart overview". Renaming the Group gives "Breathing and lungs summary" and a matching Covers line. Emptying the title goes back to "Heart summary". Restored afterwards. |
| T-05 | Pass | Clinical Summary is disabled in a repeating Group's Add bar. "Move into" never lists a repeating Group. I also checked this with a repeating Group on the same page, because the menu only lists Groups on the summary's own page. Both test Groups were deleted. |
| T-06 | Pass | After a reload, all three summaries are in place with the same titles and Covers lines. |
| T-07 | Pass | Copy adds a second "Breathing summary" in the Group, with a new id (`cs_eaoyad`). |
| T-08 | Pass | "1 problem to fix first · Question Set 1 · Two clinical summaries cover the same Group “Breathing”: keep one", and Publish is disabled. After deleting the copy, it publishes as "V1 · PUBLISHED". |
| T-09 | Pass | v2: "Two clinical summaries cover the same page “Lungs”: keep one". After deleting one, the problem is gone. |
| T-10 | Pass | "The clinical summary “Meds summary” is inside a repeating group: move it out", and Publish is disabled. v2 is left unpublished. |
| T-11 | Pass | Clinician preview: three boxes where they were placed, each with its empty-state line. With Yes answers: Breathing lists Asthma and Inhaler, Blood lists Anaemia, and Heart overview lists only Angina. No note shows twice. The Lungs page has no box. |
| T-12 | Pass | Patient preview at Phone, Tablet and Desktop, all three screens (Heart, Blood, Lungs): no summary title, box or "Clinical comments" anywhere. |
| T-13 | Pass | Fill in for the patient, then Validate shows the same three boxes. All three comments saved ("Saved"), then "Validated by Cara Clinician". |
| T-14 | Pass | Validated summary tab: Breathing summary (Asthma, Inhaler, comment), Blood summary (Anaemia, comment), Heart overview (Angina, comment) and Lungs Summary (Cough, no comment). No note appears in two blocks. The Patient answers tab shows the same blocks without comments. |
| T-15 | Pass | Clicking **Asthma** opens Validate on Heart with `?q=q_frcnsx`, and Asthma? is highlighted. The **POA Summary** link goes back to the same tab with the note still showing. |
| T-16 | Pass | All answers No and a comment only in Heart overview: the POA Summary shows only a "Heart overview" block, "Clinical comments: Checked, all clear". There are no empty blocks. |
| T-17 | Pass | The Lungs Summary's **Cough** is a link and opens Validate on the Lungs page with Cough? highlighted. |
| T-18 | Pass | The old draft shows an "Old heart summary" card ("Covers the page “Old heart”", Clinician badge) after the question, with no switch anywhere. After an edit and a reload it's still there. Stored JSON: the page flag is gone and the content now has a `clinicalsummary` element `cs_p_oldheart`. |
| T-19 | Pass | Validate on the old published version shows the box "Old heart summary" listing "Old angina note". The comment "Old comment kept" shows on the POA Summary and is still there after reopening Validate. It's stored under the old key `cs_comments_p_oldheart` (checked in the database). |
| T-20 | Pass | Old published v1 in the editor shows the summary card at the end of the page, marked "READ ONLY". There's no Add bar, no card actions and no "Saving…". The stored chapter is unchanged: revision 1 and `clinicalSummary: true` are still on the page. |
| T-21 | Pass | Sue Super on the v2 draft: "READ ONLY", no Add bar, no card buttons. All three summary cards still show. |
| T-22 | Pass | Bea: S32 test and Old switch don't appear in Trust B's list. Alex's editor URL shows "You don't have access to this hospital". The same id under Trust B shows "Questionnaire not found." The APIs return 404. |
| T-23 | Pass | The patient's own invite (from /qa/messages), DOB 17/05/1980, in a 390×844 frame: all Yes, through all three screens, then sent. No summary or "Clinical comments" text, and no sideways scroll on any screen (scrollWidth 388 = clientWidth). The patient API's JSON has no `clinicalsummary`, `cs_` or `clinicalOutputs`. |
| R-01 | Pass | The POA Summary has both tabs, Print, a status change (Ready for POA → POA complete, with Undo) and a General notes comment. Print was rendered to PDF with Playwright and shows the summary blocks and the correction note. |
| R-02 | Pass | In Validate, Asthma? changed to No: the Breathing box drops Asthma straight away. After validating again, the POA Summary's Breathing block lists only Inhaler, and the page shows "Changed by the clinician on Heart · Asthma? Patient answered “Yes”, now “No”". |
| R-03 | Pass | Patient D's second episode was pre-filled from the first ("5 answers", so the summary isn't counted). The POA Summary shows "Pre-filled on Heart" / "Pre-filled on Lungs" with each source. |
| R-04 | Pass | A Statement and a Text question were added and saved. Change type from Text to Number works. |
| R-05 | Pass | A condition pointing at a deleted question blocks Publish: "About you · “Text question”: it tests something that no longer exists (q_oldangina)". The question picker doesn't offer the summary element. |

## Failures in detail

None.

## Found while exploring

- **Cosmetic: the delete confirmation for a summary says the wrong thing.** Deleting a summary card shows: "“Clinical summary” will be deleted, with its options and disclosures."
  - It names the summary generically, not by its title (e.g. "Breathing summary").
  - A summary has no options or disclosures.
  - Seen in T-08 and T-09.
- **Cosmetic: the greyed-out Clinical Summary button doesn't say why.** Inside a repeating Group it has no tooltip or title explaining why it's disabled. BMI behaves the same way, so this may be by design.
- **Works as expected:**
  - Copying a whole Group copies its summary with a new `cs_` id, titled "Breathing (copy) summary".
  - Undo removes the copy.
  - A long title with leading spaces and an emoji is trimmed and wraps cleanly on the card.
  - A title of only spaces goes back to the default ("Heart summary"), and no `title` is stored.
- **Note, not a bug:** in the clinician view, the page-level summary (Heart overview) lines up with the last patient item in the right column (beside Blood), not at the top. The two-column layout is out of scope.

## Console and network errors

- No console errors or warnings on the editor, Preview (clinician), Validate or POA Summary after reload.
- The only non-2xx responses were these, all expected:
  - 503s on POSTs that return 204 or redirect (submit, logout, dev login, prefill). This is the known Claude in Chrome artifact (decisions.md §4), and every one of those actions succeeded.
  - 404s from T-22's deliberate cross-trust requests.
  - 401 on `/api/me` while signed out.
- **Tooling only, not app bugs:**
  - The 1Password inline menu sometimes blocked typing into the New episode form. I used `form_input` instead.
  - Mantine modal fade-outs sometimes lingered in screenshots.

## GIFs and files

- Happy path (build the summaries, T-02 to T-12): `~/Downloads/S32-happy-path-editor-preview.gif`.
- Print rendering for R-01: `/private/tmp/claude-501/-Users-rahul-poised/8735e5a4-9e31-4654-a7d9-6d69c8a4b98f/scratchpad/s32/poa.pdf`.

## Hand checks for Rahul

None.
