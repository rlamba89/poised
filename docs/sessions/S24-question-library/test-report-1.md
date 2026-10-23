# S24 test report 1

**19 passed, 0 failed, 0 blocked** (14 cases + 5 regression).

- **Environment:** commit `f845a98` with S24's uncommitted work in the tree (git status not clean: 41 changed or new files, as expected before verification). Date: 22 Oct 2026, 20:56–21:30 BST. URL: http://localhost:3000 (API :8080, health ok). The database had been reset: the Demo pre-op HQ showed "Do you smoke?" with three options.
- **Browser:** Chrome ("rahul chrome" profile) through Claude in Chrome, desktop at 1512 px wide. `resize_window` didn't change the viewport, so the phone case (T-13) ran in a 390 px-wide same-origin iframe.
- **Test data:** run number 2056. Trust A: `S24 Lib 2056` (draft), `S24 Lib B 2056` (published v1), `S24 Meds 2056` (published v2) and episode "Sam S24 2056" (Knee arthroscopy, mobile 07700 900256). Trust B: `S24 Lib Bea 2056` (draft). The Demo pre-op HQ is now **v2 published** in Trust A (from R-05).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Insert keeps the ID; second insert refused | Pass | `smoker` with its 3 options. The second insert, in another Question Set, was refused inside the window with the expected text. After a reload the ID was still `smoker`. |
| T-02 Search; group; repeating group | Pass | `ANAESTHETIC` returned the 3 rows. `measurements` returned height and weight. `zzz` showed "No library questions match." The anaesthetic group had `anaesthetic_problems` and `anaesthetic_problems_details` with "Displays when … is Yes". The allergies repeating group inserted. The weight question showed the Always ask fresh badge and kg. In Preview, choosing Yes showed the details question. |
| T-03 Where From library isn't offered | Pass | It wasn't offered in the repeating group's own bar, was greyed on a BMI page, and was absent on the published Demo HQ. Sue and Cara got a read-only editor with no From library, Same as… or Save to library. `GET library-questions` returned 403 for both. Sue's POST, DELETE and Same as… search also returned 403. |
| T-04 Save to library, never seen by another trust | Pass | Saved with the tag Work. The row showed the "This trust" badge, the WORK tag and Remove, and a search for "work" matched by tag. Ada saw it. Bea didn't, in the UI or the console (`?q=job` returned `[]`), but she saw all 11 platform questions. Bea's requests to Trust A's library returned 404. |
| T-05 Save to library refusals | Pass | `smoker` was refused with "The library already has a question with the ID smoker." The job question was refused with "…ID q_fdeccf." A reworded diabetes "No" (Demo v2 draft) was refused with the full sentence ("…was reworded to “Not at all”…"), then reverted. Statement, Calculation, Signature, File upload, BMI, Group and Section have no Save to library. The repeating group has one. |
| T-06 Remove; inserted copies stay | Pass | The red box appeared, Cancel kept the item, and Remove → Remove emptied the list. The copies in both questionnaires kept `q_fdeccf` after a reload. |
| T-07 Same as… (main path) | Pass | The card read "New question: won't pre-fill" before. After Use this: ID smoker, "Linked to “Do you smoke?” in Demo pre-op HQ", the question's own text kept, the 3 options copied, and "Pre-fills from 1 other HQ" listing "Demo pre-op HQ · by link". All of it survived a reload. GIF: S24-T07.gif. |
| T-08 Incompatible candidates greyed with reason | Pass | In `S24 Lib B`, the Text question showed "This HQ already has a question with this ID, in “Question Set 1”" for `smoker` (the plan allows this). The kind reason ("Its answer is single choice; this question's is text.") was confirmed in `S24 Meds`, where `smoker` isn't in the HQ. lb vs kg showed "Its unit is “kg”; this question's is “lb”." Changing the unit to kg enabled Use this. Allergies showed the repeating-list sentence. Select Many showed "…this question's is multiple choice." |
| T-09 Same as… refused when the ID is already in the HQ | Pass | "This HQ already has a question with this ID, in “About you”." with Use this disabled. Same as… was absent on a Clinical question, a question inside the repeating group, the repeating group itself, a Statement and a Calculation. |
| T-10 Deleting an adopted option refused at Publish | Pass | The exact sentence appeared and Publish was disabled. After ⌘Z the option came back with value `ex`, and the questionnaire published with ID `smoker`. |
| T-11 Medication type frozen | Pass | Recreational was refused with "…its medication type changed from prescribed medication to recreational drugs…". Setting it back to Prescribed published v2. |
| T-12 Seeded Demo HQ uses library questions | Pass | The structure matched. `smoker` and the weight badge were checked in the UI. The other keys and units were checked through the chapters API: `alcohol_units` (units a week), `height` (cm), `weight` (kg, fresh), `q_diabetes`, `snoring`, `pregnant` (fresh), `ponv_history`, both anaesthetic groups, `allergies` and `regular_medicines`. Trust B's Demo HQ has the same keys (API, as Bea). |
| T-13 Demo HQ on a phone (390 px) | Pass | Knee arthroscopy picked Demo pre-op HQ v1. The SMS link and date of birth worked, and there were 2 sections. Yes showed the details question. The allergy and medicine lists started empty and their add buttons showed the boxes. There was no horizontal overflow (scrollWidth = clientWidth = 388). The patient saw "Thank you, Sam". |
| T-14 Reload, deep link and Back | Pass | The `?set=&page=` link opened in a new tab with ID smoker and "Linked to…". Back from the open library window left the editor for the list, and Forward came back without the window. |
| R-01 (S23) pre-fill label, Always ask fresh | Pass | "Pre-fills from N other HQs" with "by link". A new question shows "New question: won't pre-fill". Always ask fresh switched on and its badge showed. |
| R-02 (S23) rewording refused at Publish | Pass | "“Do you have diabetes?”: the option “No” was reworded to “Not at all”…", and Publish was disabled. |
| R-03 (S19) Total hip replacement → Demo pre-op HQ v1 | Pass | Checked as Cara. |
| R-04 (S14/J3) patient sends an HQ on a phone | Pass | Covered by T-13. |
| R-05 (S23) Demo HQ new version, unchanged, publishes | Pass | v2 published after the T-05 and R-02 changes were reverted. |

## Failures

None.

## Found while exploring

1. **Minor: Save to library is offered on a question inside a repeating group, and it saves.**
   - **Steps:** as Alex, in `S24 Lib 2056` → Question Set 2 → the "Your allergies" repeating group → select "What are you allergic to?" (ID `allergies_what`) → Save to library → Save to library.
   - **What happened:** "Saved." It became a trust library item, "What are you allergic to?" (Text, ID `allergies_what`), that can be inserted on its own. Inserting it into the same HQ was refused because the ID is already there. In any other HQ it would become a standalone top-level question that shares its key with a repeating-list row field.
   - **Why it matters:** Same as… is deliberately not offered inside a repeating group. The plan doesn't say whether Save to library should be, so this may need a decision.
   - **Cleanup:** I removed the item afterwards.
2. **Cosmetic: the search box sometimes keeps the previous search.** The "Insert from the question library" and "Same as…" windows sometimes reopen with the earlier search text. When that happens, typing appends to it ("smokesmoke", "allergweigh"), and "allergweigh" showed "No published or library questions match." Other times the box was empty. It isn't consistent within one page visit.
3. **Note: only one refusal reason is shown.** When a candidate is refused for two reasons (its ID is already in this HQ, and it's a different kind of answer), only the ID reason is shown. The plan allows this; I'm noting it for the builder.
4. **Content note: the pregnancy question is shown to a male patient.** The seeded Demo HQ asks "Is there any chance you could be pregnant?" of a patient recorded as Male, because the question has no condition on sex. Not in this session's scope.
5. **Checks that worked as expected:**
   - Double-clicking Insert added the question once.
   - Removing a platform library question through the API returned 403 ("The platform's starter questions can't be removed.").
   - Alex's Same as… search on Trust B's questionnaire returned 404.
   - A malformed POST to library-questions returned 400.

## Console and network errors

- No console errors or warnings in the tabs I tracked.
- No unexpected 4xx or 5xx responses. The only ones were the 403, 404 and 400 responses I triggered on purpose in the console.

## GIFs

- `/Users/rahul/Downloads/S24-T01.gif`: the main library path (insert, then the refused second insert).
- `/Users/rahul/Downloads/S24-T07.gif`: the main Same as… path.

## Hand checks for Rahul

None.
