# S08 test report 3 (retest 2)

**13 passed, 1 failed, 0 blocked.**

This round re-ran:
- the new cases T-25, T-26 and T-27;
- the spot checks T-01, T-02, T-05, T-09 (All hospitals) and T-15;
- all six regression cases, R-01 to R-06.

The F-1 fix works: the Preview is read-only for Sue Super and Cara Clinician. One Minor failure remains (F-1 below): on the "E2E …" questionnaires, **Save answers** does nothing when the answers produce an output. It's the same no-`codes` problem that T-27 fixed for the Outputs panel, but in the test-case code.

**Environment**
- Commit `603d3d5`. The working tree is not clean: it holds the uncommitted S08 build and fixes plus planning docs, and this report tests that working tree.
- Run on 12 Oct 2026, 23:41 to 23:55 BST.
- App at http://localhost:3000 (API :8080). The conductor had freshly migrated and seeded it, and `/api/health` returned ok.
- Browser: Chrome through Claude in Chrome, in a 1280×773 tab.
  - Narrow and phone widths ran in iframes of 390, 500 and 700px.
  - The first click after a page load was sometimes ignored until a screenshot was taken. This is a harness quirk (also in report 2), not an app bug.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-25 Preview read-only (F-1) | **Fail** (step 4 only) | Steps 1–3 pass. Sue and Cara both see "Read only: only super clinicians and admins of the whole trust can change test cases.". There's no name box, no Save answers and no delete icon on the existing test case. **Load** and **Run all** work ("All 1 passed"). The header button reads **Open in editor**, which opens the editor READ ONLY. On a published v1, Alex sees "Read only: test cases can be changed only in a draft version." In step 4, Alex's **Save answers** works on a questionnaire built in the editor ("MT asthma yes" saved), but silently does nothing on the plan's "E2E trust-wide …" draft. See F-1. |
| T-26 Header in a narrow window | Pass | Cara on Episodes in 390, 500 and 700px frames. The header is its own sideways scroller (scrollWidth 716, clientWidth 354/464/664) with a visible scrollbar. After a real scroll in the 390px frame, **Episodes**, the CLINICIAN badge and **Sign out** were all visible. Clicking **Sign out** there signed out. Clicking **Questionnaires** in the 500px frame opened Questionnaires. The rest of the page is wider than 390px only because of the Episodes table (out of scope). |
| T-27 Small fixes | Pass | **Step 1:** on my own draft, a Text question shown "when MT do you have asthma? is Yes" lost its source question to a delete. Publish then showed "2 problems to fix first" (the missing question, and the test case that now fails), and Cancel closed it. A MutationObserver saw only that text, never "Patients can then be given this version…". **Step 2:** as Sue, choosing Yes for "Do you smoke?" on "E2E trust-wide 1210 233929 d4f9" shows Clinical outputs → Notes "Smoker", with no crash and no console errors. |
| T-01 Cara straight in | Pass | Cara goes straight to `/o/…/h/…0a/episodes`. The header shows Trust A, Hospital A1 and a CLINICIAN badge. |
| T-02 Alex's picker | Pass | Trust A "Super clinician, whole trust", with A1 and A2. |
| T-05 Ada's picker | Pass | Trust A "Admin, whole trust", with A1 and A2 each "Admin". |
| T-09 All hospitals (spot check) | Pass | Ada's `?hospitals=all` shows the switch and a Hospital column with both A1 and A2 rows. A2's episode opens at `/h/…2a/…` with the header showing Hospital A2. |
| T-15 Leak by URL (spot check) | Pass | Cara pasting Ada's A2 episode URL sees "You don't have access to this hospital" and Choose a hospital, with no patient data. API check: `GET /api/o/<A>/h/<A2>/episodes` as Cara gives 404 "Hospital not found.". `GET /api/o/<A>/episodes` as Cara returns only her A1 episodes (239 rows, all A1). |
| R-01 J1 Author | Pass | Alex created "MT S08 2346 preview" with a Question Set, a page, a Yes/No question "MT do you have asthma?", a disclosure on Yes ("MT asthma", Allergies) and a conditional Text question. All of it was kept after a reload, and the Preview shows it. |
| R-02 J2 Publish | Pass | Publishing was blocked with the problems listed (T-27). Undo brought the question back, and publishing then worked: V1 · PUBLISHED, READ ONLY. **Create new version** gave V2 · DRAFT. |
| R-03 J3 Patient at phone size | Pass | Cara created an episode for Robin Sample with "MT S08 2346 preview v1". In a 390×844 frame the page showed "Hello Robin" and one Question Set. The patient answered No, then Send my answers (in-app modal) gave "Thank you, Robin". scrollWidth 386 = clientWidth, so no sideways scroll. Cara's view after a reload: READY FOR REVIEW. |
| R-04 J4 Validate | Pass | Changing the answer to Yes showed "Patient answered: No" and revealed the conditional Text question. After Validate and **Complete HQ review**: READY FOR POA. |
| R-05 J5 POA Summary | Pass | Validated summary shows "MT asthma" and "Changed by clinician" lines. Patient answers shows "No answers produce anything…", which is correct for the patient's No. Setting the status to POA complete is logged. The comment "MT S08 retest3 comment" appears in General notes. |
| R-06 Dates | Pass | Dates show as "21 July 1990" and "12/10/2026, 23:49". |

## Failures

### F-1 · Minor: Save answers silently does nothing on questionnaires whose outputs have no codes (the E2E fixtures)

T-27 made the Preview's **Clinical outputs** panel cope with an output that has no `codes`. The test-case code in `packages/clinical` still assumes `codes` is always there. So on any "E2E …" questionnaire (T-25 step 4 names "the same draft"), a super clinician can't save a test case whose answers produce an output.

- **Signed in as:** Alex Author (whole trust), Hospital A1.
- **URL:** `/o/6ef150fb-…/h/…0a/chapters/ab67f99d-e7e2-45df-bc4e-dee8e21dd73e/preview` (first Question Set of the draft "E2E trust-wide 1210 233929 d4f9").
- **Steps:**
  1. Open the Preview.
  2. Choose **Yes** (or **No**) for "Do you smoke?". Clinical outputs shows "Smoker" (or "Non-smoker").
  3. Type a test case name, e.g. "MT S08 2345 smoker".
  4. Click **Save answers**.
- **Expected:** the test case is saved and listed.
- **Actual:**
  - Nothing happens. No message, no request is sent, and the chapter's revision stays at 1.
  - Console (each click): `TypeError: o.output.codes is not iterable at outputLines (packages/clinical/src/testcases.ts:28) at add (src/features/preview/TestCases.tsx:51)`.
  - The fixture's outputs are `{id, note:{text}}`, with no `codes` and no `note.category`. In `outputLines`, `for (const c of o.output.codes)` throws. The `Note (${o.output.note.category})` line would also print "undefined" for these.
- **Not affected:** questionnaires built in the editor. The editor always writes `codes`, and it requires a category, so the same steps on "MT S08 2346 preview" saved "MT asthma yes".
- **Probably also affected (not tried):** **Run all** and the publish check's test-case run on such a chapter, because both go through `outputLines`.
- **GIF:** `~/Downloads/S08-T25.gif`. It shows choosing Yes, the "Smoker" output and Save answers, and nothing happening after the click.

## Found while exploring

1. **Cosmetic, probably from before S08: the editor's "Delete this?" dialog for a question loses its text as it fades out.**
   - Steps: select a question card, click the bin icon, then click **Cancel** (or **Delete**).
   - A MutationObserver recorded the dialog's text: first the full message ("“MT do you have asthma?” will be deleted, with its options and disclosures. Other items depend on this…"), then only "Cancel Delete", with an empty body, during the fade-out.
   - It's the same kind of problem as the T-24 fix (questionnaire delete) and report 2's Publish-dialog flash, but in the editor's question-delete dialog.
2. The "E2E J2 …" questionnaire in the list is a **v2 draft whose broken condition is already fixed**, because the J2 journey published v1 and created v2. Its description ("publishing is blocked until it's fixed") no longer matches, so I built my own broken draft for T-27 step 1. This is a test-data note, not a bug.
3. As Cara, `GET /api/o/<Trust A>/episodes` (the trust-wide list) returns 200 with only her own hospital's episodes. That looks intended (scoped, not refused); I note it only because T-17 describes this route for trust-wide staff.

## Console and network errors

- The only console errors were the `o.output.codes is not iterable` exceptions in F-1.
- No unexpected 4xx or 5xx responses. The only 404 was the deliberate A2 probe as Cara (T-15).

## Hand checks for Rahul

None. Every confirmation used (Publish, delete question, Send my answers) is an in-app modal.

## GIFs

- `~/Downloads/S08-T25.gif`: F-1. Alex on the E2E draft Preview chooses Yes, the "Smoker" output appears, he clicks Save answers, and nothing happens.

## Test data created

- **Questionnaire** "MT S08 2346 preview" (Alex):
  - v1 published;
  - v2 draft;
  - Question Set 1 has "MT do you have asthma?" (a Yes disclosure "MT asthma", Allergies) and a conditional Text question;
  - test case "MT asthma yes".
- **Episode:** Robin Sample at Hospital A1, "MT S08 retest3 hip", HQ "MT S08 2346 preview v1".
  - The patient answered No.
  - Cara validated it (changed to Yes, Text "MT mild") and set it to POA complete.
  - Comment: "MT S08 retest3 comment".
- **Not saved:** the test case "MT S08 2345 smoker" on the E2E draft (F-1).
