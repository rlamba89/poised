# S08 test report 1

**25 passed, 0 failed, 0 blocked** (19 cases T-01 to T-19, plus 6 regression cases R-01 to R-06).

**Environment**
- Commit `603d3d5`. The working tree is not clean: it has the uncommitted S08 build changes plus planning docs (this report tests that working tree).
- Run on 12 Oct 2026, 22:48 to 23:05 BST.
- App at http://localhost:3000 (API :8080), freshly migrated and seeded by the conductor.
- Browser: Chrome through Claude in Chrome. The window can't resize, so the phone-size case ran in a 390×844 iframe.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 One hospital: straight in | Pass | Cara lands on `/o/…/h/…/episodes`. The header shows Poised, Trust A, Hospital A1, Questionnaires, Episodes, her name and a CLINICIAN badge. |
| T-02 Picker | Pass | Alex sees one Trust A card ("Super clinician, whole trust") listing A1 and A2. A1 opens Questionnaires with a SUPER CLINICIAN badge. |
| T-03 Max Multi | Pass | Two cards: Trust A has only A2 (Clinician) and no badge; Trust B is "Admin, whole trust" with B1. B1 opens Questionnaires (ADMIN). The header box is grouped by trust, and choosing A2 opens Episodes (CLINICIAN). |
| T-04 Platform admin | Pass | Shows "You don't have access to any hospital yet" and the platform-admin text. Sign out works. A Trust A URL shows "You don't have access to this hospital", and the API returns 404. |
| T-05 Ada's picker | Pass | Trust A, "Admin, whole trust"; A1 and A2 each show Admin. |
| T-06 Header switch, then Back | Pass | The `/h/` part changes (…0a → …2a) and the same trust questionnaires are listed. Back returns to A1, and a reload keeps A1. |
| T-07 Back to the picker | Pass | `/` shows both trusts for Max. |
| T-08 Each hospital sees its own episodes | Pass | Cara's A1 episode (Jo Bloggs) and Val's A2 episode (Pat Example) each appear only in their own hospital's list. Neither user gets an "All hospitals" switch. |
| T-09 All hospitals | Pass | The Hospital A1 / All hospitals switch works. With All hospitals on, the URL has `?hospitals=all`, a Hospital column appears and both episodes are listed. The choice survives a reload. Val's row opens at A2 (the header shows A2), and Back returns to the All hospitals list. The Status filter works across both hospitals (20 rows, all HQ not complete, from A1 and A2). GIF recorded. |
| T-10 Patient record is the trust's | Pass | Cara's patient picker lists Pat Example, and the A1 episode is created. Val's A2 episode for Pat still isn't in Cara's list. |
| T-11 Super clinician authors and publishes | Pass | Created "S08 test MT 2250" with a Question Set, a page and a Yes/No question. Publish (in-app modal) gave "V1 · PUBLISHED" and READ ONLY. |
| T-12 Clinician can't author | Pass | Cara sees Trust A's list with no New Questionnaire button. Ada's draft opens READ ONLY with no Publish. |
| T-13 Delete someone else's draft | Pass | The draft "S08 delete me 2255" isn't in Trust B's list. Ada's menu offers Delete, and the delete works (in-app modal). Alex's own draft offers Delete; Ada's draft "S08 Ada draft 2300" offers only Edit for Alex, and the API DELETE returns 403. |
| T-14 Different roles per trust | Pass | Max at B1 (Admin) has New Questionnaire. At A2 (Clinician) there's no New Questionnaire, drafts open READ ONLY and every settings field is disabled. |
| T-15 Episode URL from another hospital | Pass | Cara opening A2's episode URL sees "You don't have access to this hospital" and a "Choose a hospital" link, with no patient data. With `/h/` changed to A1 she sees "Episode not found." |
| T-16 Trust B URL as Ada | Pass | Shows "You don't have access to this hospital" and an empty header box ("Choose a hospital"). Trust A org with B1's hospital id gives the same refusal. The ids match Bea's own address. |
| T-17 API refuses | Pass | Trust B's hospital gives 404 `{"error":"Hospital not found."}`, and Trust B's episodes give 404 `{"error":"Trust not found."}`. Trust A's episodes return 221 rows, all with `hospitalName`. `/api/me` shows Trust A with role admin and two hospitals, each admin, with `timeZone`. |
| T-18 Old and broken addresses | Pass | `/h/…/episodes` gives the app's 404 page. `/o/nope/h/nope/episodes` gives "You don't have access to this hospital". |
| T-19 Deep link while signed out | Pass | Redirects to `/login`. |
| R-01 J1 Author | Pass | The question text survives a reload, and Preview shows it. |
| R-02 J2 Publish | Pass | Create new version gave a V2 draft. A condition on a question that was then deleted blocked publishing with "1 problem to fix first … tests something that no longer exists". After the fix, v2 published. |
| R-03 J3 Patient at phone size | Pass | In a 390×844 frame: "Hello Pat", one Question Set, no sideways scroll (scrollWidth equals clientWidth). After Send my answers: "Thank you, Pat". Cara's view then shows READY FOR REVIEW. |
| R-04 J4 Validate | Pass | After the answer was changed, "Patient answered: Patient answer MT" appeared. Validate, then Complete HQ review, gave READY FOR POA. |
| R-05 J5 POA Summary | Pass | Both tabs show. The status change to POA complete is logged, and a comment (with an emoji) appears in General notes. |
| R-06 Dates | Pass | Dates show as "2 March 1961" and "12/10/2026, 23:01". |

## Failures

None.

## Found while exploring

1. **Minor (UX): switching hospital from an Episodes page drops you on Questionnaires.**
   - As Ada at A2 Episodes, choosing Hospital A1 in the header box opens `/o/…/h/<A1>/questionnaires`, not A1's Episodes.
   - The same happens from an episode's POA Summary.
   - It appears to send you to the role's start page. Neither the plan nor the brief says what should happen, so this is not marked as a failure. A user who was working in Episodes would probably expect to stay there.
2. **Cosmetic: the clinician's "…" menu item reads "View" but keeps the pencil (edit) icon** (Max at A2 and Cara, on any questionnaire).
3. **Cosmetic: the delete-questionnaire modal flashes an empty name while closing.**
   - After Delete is confirmed, the fading modal's text reads `"" and all its Question Sets will be deleted…`.
   - It shows only during the close animation.
4. **API checks (all correct), run with `fetch` from the signed-in page:**
   - **Max at A2 (Clinician):**
     - creating a questionnaire, adding a chapter, creating a version, creating an option list and deleting Alex's draft are all refused with 403 and clear messages;
     - `/api/o/<TrustB>/h/<A2>/…` and `/api/o/<TrustA>/h/<B1>/…` return 404;
     - `/api/o/<TrustA>/episodes` returns only Hospital A2 rows.
   - **Val (A2):**
     - GET, PATCH, notes and `/hq` on Cara's A1 episode under the A2 path all return 404 "Episode not found."
     - The same episode under the A1 path returns 404 "Hospital not found."
   - **Bea (Trust B), using Trust A ids under B1:**
     - get a questionnaire, get or patch a chapter, create a version and delete Ada's draft all return 404;
     - creating an episode with Trust A's patient and HQ version returns 400 "Choose a patient and a published HQ.";
     - B1's patients, published HQs and option lists are all empty.
   - **Sam:** `/api/me` has `platformAdmin: true` and `trusts: []`.
5. **Double-clicking "Create episode"** created only one episode.
6. **Keyboard:** in the New Questionnaire drawer, Tab moves from Description to Cancel and then Save, and Enter on Save submits.
7. **Question text commits when the field loses focus,** not as you type. A reload straight after typing, without leaving the field, loses the edit. This was the same before S08 and isn't caused by this session.

## Console and network errors

- No console errors or warnings during the monitored round (Questionnaires, All hospitals, an A2 episode, the picker, a no-access Trust B URL, the editor).
- No unexpected 4xx or 5xx responses. Every API call returned 200. The no-access pages are decided in the browser from `/api/me`, without calling the API.
- Every API request appears twice in development (React StrictMode). This is expected.

## Hand checks for Rahul

None. All in-app confirmations (Publish, Delete, Send my answers) are Mantine modals, not browser dialogs.

## GIFs

- `~/Downloads/S08-T09.gif`: the main happy path (T-09), from Questionnaires to Episodes, All hospitals, a reload, opening an A2 episode, Back, and the Status filter.

## Test data created

- **Questionnaires (all Alex's except where noted):**
  - "S08 test MT 2250", v1 retired and v2 published;
  - "S08 delete me 2255", since deleted;
  - "S08 Ada draft 2300", a draft by Ada.
- **Episodes:**
  - Jo Bloggs at A1, "MT S08 Cara A1 hip";
  - Pat Example at A2, "MT S08 Val A2 knee";
  - Pat Example at A1, "MT S08 Cara A1 Pat shoulder", taken to POA complete with a comment.
