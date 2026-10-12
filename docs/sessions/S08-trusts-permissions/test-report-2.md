# S08 test report 2 (retest)

**20 passed, 1 failed, 0 blocked.** The run covered the new cases T-20 to T-24, the changed cases T-03 and T-06, the spot checks T-01, T-02, T-05, T-12 and T-14, and all six regression cases R-01 to R-06. One Minor failure was found while exploring the new whole-trust rule (F-1).

**Environment**
- Commit `603d3d5`. The working tree is not clean: it holds the uncommitted S08 build and retest changes plus planning docs, and this report tests that working tree.
- Run on 12 Oct 2026, 23:11 to 23:36 BST.
- App at http://localhost:3000 (API :8080). The conductor had freshly migrated and seeded it.
- Browser: Chrome through Claude in Chrome.
  - The first tab was stuck at 500×757 because the window wouldn't resize, so header items past 500px were clicked with a JS `click()`. That is how the Episodes link was reached in T-23 and T-03.
  - New tabs opened at 1280×717 and were used for the desktop cases.
  - The phone-size case ran in a 390×844 iframe.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-20 Super clinician of one hospital: view only | Pass | Sue lands straight on A1 Questionnaires with a SUPER CLINICIAN badge. Alex's draft "S08 trust-wide 2315" is listed, and there's no New Questionnaire button. The row's … menu shows only View, with an eye icon. The draft opens READ ONLY: no Publish, no Add Question Set, no card toolbar, every settings field disabled, and the Translate cells can't be edited. The published "E2E trusts …" has no Create new version. (The Preview page is not read-only: see F-1.) |
| T-21 API refuses Sue | Pass | The plan's POST `/questionnaires` gives `403 Forbidden {"error":"Only super clinicians and admins of the whole trust can change the trust's questionnaires."}`, and POST `/publish` gives the same 403 message. DELETE on Alex's draft gives 403 "Only the person who created this questionnaire or an admin of the whole trust can delete it." The DB shows no "x MT Sue" questionnaire, and the draft is unchanged (read-only `psql` check). |
| T-22 Whole-trust staff still can | Pass | Alex's draft is editable for Alex, and Publish (in-app modal) gives V1 · PUBLISHED with Create new version. Ada's … menu on Alex's draft "S08 Alex draft 2322" offers Edit and Delete, and the delete works. |
| T-23 Switching keeps the section | Pass | Ada: A1 Episodes → choose A2 → A2 Episodes. A2 Questionnaires → choose A1 → A1 Questionnaires. A1 POA Summary → choose A2 → A2 Episodes list. Back returns to the A1 POA Summary, and a reload keeps it. GIF recorded. |
| T-24 Delete dialog keeps the name | Pass | Alex deleted "S08 fade 2320". A MutationObserver recorded every text the modal showed while opening and closing: it always read "“S08 fade 2320” and all its Question Sets will be deleted…" and never `""`. The row is gone. The same was seen for Ada's delete. |
| T-03 Max Multi (changed step 5) | Pass | The picker shows Trust A with only A2 (Clinician) and Trust B "Admin, whole trust" with B1. B1 Episodes → choose A2 lands on `/o/<TrustA>/h/…2a/episodes`. The header reads Trust A, CLINICIAN, and there's no All hospitals switch. The hospital box is grouped by trust. |
| T-06 Switch, Back, reload | Pass | Run as Ada, not Alex: A2 Questionnaires → A1 Questionnaires keeps the section and the same trust list. Back and reload were checked on the POA Summary (in T-23). |
| T-01 Cara straight in | Pass | `/o/…/h/…0a/episodes`, Trust A, Hospital A1, CLINICIAN badge. |
| T-02 Alex's picker | Pass | Trust A "Super clinician, whole trust" with A1 and A2. A1 opens Questionnaires. |
| T-05 Ada's picker | Pass | Trust A "Admin, whole trust"; A1 and A2 each show Admin. |
| T-12 Clinician can't author, View icon | Pass | Cara has no New Questionnaire. The … menu shows View with an eye icon. The draft "E2E trust-wide …" opens READ ONLY with no Publish. |
| T-14 Max at A2 | Pass | No New Questionnaire, and the … menu shows View with an eye icon. |
| R-01 J1 Author | Pass | Alex created "S08 trust-wide 2315" with a Question Set, a page and a Yes/No question "MT do you smoke?". It was kept, since Sue and the patient later saw it. Preview of v2 shows the question. |
| R-02 J2 Publish | Pass | Create new version gave a V2 draft. A Text question shown "when MT do you smoke? is Yes", whose source question was then deleted, blocked Publish with "1 problem to fix first … it tests something that no longer exists (q_mgqunl)". Setting Display to Always fixed it, and v2 published. |
| R-03 J3 Patient at phone size | Pass | Cara created an episode for Robin Sample with "S08 trust-wide 2315 v1". In a 390×844 frame the page showed "Hello Robin" and one Question Set, with no sideways scroll (scrollWidth 388 = clientWidth). After Send my answers it showed "Thank you, Robin", and Cara's view became READY FOR REVIEW. |
| R-04 J4 Validate | Pass | After the answer was changed to Yes, "Patient answered: No" appeared. Validate, then Complete HQ review, gave READY FOR POA. |
| R-05 J5 POA Summary | Pass | Both tabs show. Changing the status to On hold is logged, and the comment "MT S08 retest comment" appears in General notes. |
| R-06 Dates | Pass | Dates show as "21 July 1990" and "12/10/2026, 23:23". |
| Sue: other hospital, /api/me | Pass | Sue at `/o/<TrustA>/h/<A2>/episodes` sees "You don't have access to this hospital". `/api/me` returns Trust A with `role: null` and only Hospital A1 (`super_clinician`). Her A1 Episodes list has New episode and no All hospitals switch. |
| Console / network round as Sue | Pass | Opening the list, a draft and its sets gave no console errors, and every API call returned 200. |
| F-1 (new rule, Preview) | **Fail** | See below. |

## Failures

### F-1 · Minor: Preview lets a view-only user try to save test cases (the save is refused)

The new rule says Sue (and clinicians) can view the trust's questionnaires but not change them. The editor follows this, but the **Preview** page, reached with the editor header's **Preview** button, still offers editing controls. The server refuses the save correctly, so no data is changed.

- **Signed in as:** Sue Super (Hospital A1 only).
- **URL:** `/o/6ef150fb-…/h/…0a/chapters/0528e236-3717-42e3-a788-07f4757d9953/preview` (the first Question Set of the draft "E2E trust-wide 1210 230935 520d").
- **Steps:**
  1. Open the draft from Questionnaires (READ ONLY).
  2. Open a Question Set and click **Preview** in the header.
  3. Under **Test cases**, type "MT Sue test case".
  4. Click **Save answers**.
- **Expected:** a view-only user isn't offered Save answers (nothing can be changed), in the same way the editor hides its editing controls. The **Edit** button at the top right is also odd for her: it leads to a read-only editor.
- **Actual:**
  - The name box and **Save answers** are enabled.
  - Clicking Save answers sends `PUT …/chapters/0528e236…/content` and gets **403**.
  - The panel then shows a red "Only super clinicians and admins of the whole trust can change the trust's questionnaires."
- **Cause seen in the code:** nothing in `apps/web/src/features/preview/` (TestCases, PreviewPage) or `pages/hospital/Preview.tsx` uses `canChange`. The same gap should apply to any clinician, not only Sue.
- **Console/network:** only the expected 403. No console errors.

## Found while exploring

1. **Minor: the staff header doesn't fit below about 768px wide.**
   - The new trust name and hospital box push **Episodes** and **Sign out** past the right edge, and there's no way to scroll to them.
   - Measured in iframes on `/o/…/h/…/episodes`:
     - at 390, 500 and 600px the header is 762px wide;
     - Sign out sits at x≈761 at those widths, so it is unreachable;
     - at 500px, Episodes (x 493–574) is cut off too.
   - At 768px and wider everything fits.
   - The plan has no staff phone-size case, so this is not counted as a failure. A clinician on a small tablet, or in a narrow browser window, can't reach Episodes or Sign out.
2. **Cosmetic, probably from before S08: cancelling a blocked Publish dialog flashes the "ready" version.**
   - Open Publish on a draft with a logic problem, then click **Cancel**.
   - While the modal fades out, it switches to "Patients can then be given this version. It becomes read-only…", with an enabled-looking blue **Publish** button. A MutationObserver recorded both texts.
   - This is the same kind of problem as the T-24 fix, in a different dialog.
3. **Minor, from before S08 (not caused by this session): Preview crashes on the e2e fixture HQs.**
   - In the Preview of the "E2E trust-wide 1210 230935 520d" draft's first set (or any "E2E …" HQ built the same way), choosing **Yes** for "Do you smoke?" replaces the page with **Something went wrong**.
   - Console: `TypeError: Cannot read properties of undefined (reading 'map')` at `OutputsPanel` (`PreviewPage.tsx` line 254, `o.output.codes.map`).
   - The fixture's `clinicalOutputs` are `{id, note:{text}}`, with no `codes` (and no `note.category`), and `OutputsPanel` assumes `codes` is always an array.
   - Either the e2e fixture or the panel's defensiveness needs fixing. Questionnaires authored in the editor were fine.
4. **API spot checks as Sue:** create, publish and delete were all refused with 403 (T-21). I tried to probe the other change routes (add or delete a chapter, reorder, rename, save content, new version, option-list create/update/delete) in one batch. The session's permission guard stopped it, because it would have modified shared data, so it was not run.
   - Reading the code instead: `deleteChapter` and the other chapter and option-list handlers go through `canEdit` / `canChange`.
   - Saving content was shown refused in the browser (F-1, 403).
5. **Test harness note (not an app bug):** in the stuck 500px tab, the first click after a navigation was often ignored until a screenshot had been taken, because the page wasn't painting in the background.
   - The extension also reported `POST /api/dev/login` as 503 while the sign-in actually succeeded.
   - `curl` to the same endpoint, through Vite and directly, returns 204.
   - Neither is an app problem.
6. Questions authored by Alex show in the patient view and to Sue straight away. Trust-owned questionnaires are listed with `hospitalId: null`.

## Console and network errors

- None tied to the cases, apart from the crash in exploring item 3, which happened only in the Preview of an e2e fixture HQ.
- The only unexpected status was the 403 in F-1, which is the server correctly refusing the save.

## Hand checks for Rahul

None. Every confirmation used (Publish, Delete, Send my answers, delete question) is an in-app Mantine modal.

## GIFs

- `~/Downloads/S08-T23.gif` (main happy path, T-23): Ada goes from A1 Questionnaires → Episodes → switches to A2 (Episodes) → Questionnaires → switches to A1 (Questionnaires) → Episodes → an episode → POA Summary → switches to A2 (Episodes).
- No GIF for F-1. It is a single click whose result (the red message and the 403) is described above.

## Test data created

- **Questionnaires:**
  - "S08 trust-wide 2315" (Alex): v1 published, then v2 published with a Text question.
  - "S08 fade 2320" (Alex): deleted.
  - "S08 Alex draft 2322" (Alex): deleted by Ada.
- **Episode:** Robin Sample at A1, "MT S08 retest Robin knee", HQ "S08 trust-wide 2315 v1". The patient answered; Cara validated it (answer changed to Yes), set it On hold and added the comment "MT S08 retest comment".
