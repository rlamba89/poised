# S24 test report 2 (retest)

**19 passed, 0 failed, 0 blocked** (14 cases + 5 regression). Both fixes from report 1 work.

- **Environment:** commit `f845a98` with S24's uncommitted work in the tree (git status not clean: 42 changed or new files, as expected before verification). Date: 22 Oct 2026, 21:31–22:05 BST. URL: http://localhost:3000 (API :8080, health ok).
- **Database:** not reset since report 1. Report 1's data was still there, and the Demo pre-op HQ was v2 published in Trust A. Its "Do you smoke?" has the three library options, so the case wasn't blocked. This run published **v3** (T-05 and R-02 changes reverted first, then R-05).
- **Browser:** Chrome ("rahul chrome" profile) through Claude in Chrome, desktop at about 1456–1512 px wide. The phone case (T-13) ran in a 390×844 same-origin iframe.
- **Test data:** run number 2131.
  - Trust A: `S24 Lib 2131` (draft), `S24 Lib B 2131` (published v1) and `S24 Meds 2131` (published v2).
  - Episode "Sam S24 2131" (Knee arthroscopy, made-up mobile 07700 900131, answers sent).
  - Trust library item `S24 job question 2131` (ID `q_vjbnjc`), saved and then removed in T-06.
  - In Trust B I only read data (Bea's existing draft `S24 Lib Bea 2056`). Nothing was changed there.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Insert keeps the ID; second insert refused | Pass | `smoker` was inserted with its 3 options and the Settings panel showed "ID smoker". The second insert, in Question Set 2, was refused inside the window under the row ("This HQ already has a question with the ID smoker. A question can be in an HQ only once (copy it to ask it again under a new ID)."). After a reload the ID was still `smoker`. GIF: S24-T01-retest.gif. |
| T-02 Search; group; repeating group; **step 9 (retest)** | Pass | `ANAESTHETIC` returned the 3 rows. `measurements` returned height and weight. `zzz` showed "No library questions match." The group inserted with `anaesthetic_problems` and `anaesthetic_problems_details` and "Displays when … is Yes". Your allergies inserted as a repeating group. Weight showed the Always ask fresh badge and kg. In Preview, choosing Yes showed the details question. **Step 9:** both windows opened with an empty search every time: after Insert, after ×, after Escape and after a click outside. I typed a search before closing each time (library: "snore", "tall"; Same as…: "smoke", "allerg"). The appending bug from report 1 didn't happen. |
| T-03 Where From library isn't offered | Pass | The repeating group's own bar had no From library; the page bar had one. From library was greyed on a BMI page. The published Demo HQ had no Add content bar. Sue and Cara got a read-only editor with no From library, Same as… or Save to library. `GET library-questions` returned 403 for both, and Sue's POST also returned 403. |
| T-04 Save to library, never seen by another trust | Pass | Saved with the tag Work ("Saved. The trust's authors can now insert it into any questionnaire, keeping its ID."). The row showed the "This trust" badge, the WORK tag and Remove; platform rows had no Remove. Ada saw it. Bea didn't, in the UI or the console (`?q=job` returned 0 items, while all 11 platform items were listed). Bea's request to Trust A's library returned 404. |
| T-05 Save to library refusals; **step 9 (retest)** | Pass | `smoker` was refused with "The library already has a question with the ID smoker." The job question was refused with "…ID q_vjbnjc." The reworded diabetes "No" (Demo v3 draft) was refused with the full sentence, then reverted. Statement, Calculation, Signature, File upload, Group and Section have no Save to library; the BMI block can't be selected. The repeating group has the button. **Step 9:** "What are you allergic to?" inside the repeating group now has **no Save to library** (and no Same as…). The console POST was refused with 409, but the wording differs from the plan; see "Found while exploring" 1. |
| T-06 Remove; inserted copies stay | Pass | The red box appeared, Cancel kept the item, and Remove → Remove gave "No library questions match." After a reload, both questionnaires still hold `q_vjbnjc` (checked in the UI and through the chapters API). |
| T-07 Same as… (main path) | Pass | Before: "New question: won't pre-fill". After Use this: "ID smoker", "Linked to “Do you smoke?” in Demo pre-op HQ", its own text kept, the 3 options copied. The label read "Pre-fills from 2 other HQs" (Demo pre-op HQ and S24 Lib B 2056, both "by link"). All of it survived a reload. Adding an option was allowed while editing, and ⌘Z removed it. GIF: S24-T07-retest.gif. |
| T-08 Incompatible candidates greyed with reason | Pass | The Text question's `smoker` row showed the ID refusal (allowed by the plan). The kind sentence "Its answer is single choice; this question's is text." was confirmed in `S24 Meds 2131`, which has no `smoker`. lb showed "Its unit is “kg”; this question's is “lb”." for both weight candidates; after changing to kg, Use this was enabled. Allergies showed the repeating-list sentence. Select Many showed "Its answer is single choice; this question's is multiple choice." on the diabetes row; the smoker row there showed the ID reason. |
| T-09 Same as… refused when the ID is already in the HQ | Pass | "This HQ already has a question with this ID, in “About you”." Use this was disabled. |
| T-10 Deleting an adopted option refused at Publish | Pass | "“Are you a smoker at the moment?”: the option “I used to smoke” was removed. A published question's answers can't change: add a new question instead (published in “Demo pre-op HQ” v3 and “S24 Lib B 2056” v1)." Publish was disabled. After ⌘Z it published v1 with ID `smoker` and "Linked to…". |
| T-11 Medication type frozen | Pass | Recreational was refused with "…its medication type changed from prescribed medication to recreational drugs. A published question's answers can't change…". Set back to Prescribed, v2 published. |
| T-12 Seeded Demo HQ uses library questions | Pass | The structure matched (About you: Smoking and alcohol, Height and weight; Your health: Health, Anaesthetics, Allergies and medicines; Nurse checks). The UI showed `smoker`, the weight and pregnancy badges, and the alcohol hint. The chapters API confirmed `alcohol_units` (units a week), `height` (cm), `weight` (kg, fresh), `q_diabetes`, `snoring`, `pregnant` (fresh), `ponv_history`, both anaesthetic groups, `allergies` and `regular_medicines`. Trust B's Demo HQ (API, as Bea) has the same keys. |
| T-13 Demo HQ on a phone (390×844) | Pass | Knee arthroscopy picked **Demo pre-op HQ v3**, the latest published version (the plan says v1 because it assumes a fresh database). The SMS link and date of birth worked, and there were 2 sections with the expected questions and hint. Yes showed "What happened with your anaesthetic?". The allergy and medicine lists started empty, and their buttons showed the boxes. There was no horizontal overflow (scrollWidth = clientWidth = 388). After the "Send your answers?" box, the patient saw "Thank you, Sam". |
| T-14 Reload, deep link and Back | Pass | The `?set=&page=` link opened in a new tab with "ID smoker" and "Linked to…". Back from the open library window went to the Questionnaires list; Forward returned to the editor without the window. |
| R-01 (S23) pre-fill label, Always ask fresh | Pass | "Pre-fills from N other HQs" with "by link". New questions show "New question: won't pre-fill". Always ask fresh switched on and its badge showed. |
| R-02 (S23) rewording refused at Publish | Pass | "Your health · “Do you have diabetes?”: the option “No” was reworded to “Not at all”…". Publish was disabled. |
| R-03 (S19) Total hip replacement → Demo pre-op HQ | Pass | It picked Demo pre-op HQ **v3**, the latest published (checked as Cara). |
| R-04 (S14/J3) patient sends an HQ on a phone | Pass | Covered by T-13. |
| R-05 (S23) Demo HQ new version publishes | Pass | v3 published after the T-05 and R-02 rewording was put back. |

## Failures

None.

## Found while exploring

1. **Note (test plan wording, not an app fault): T-05 step 9 gives a different refusal sentence.**
   - **What I did:** as Alex, in the console: POST `library-questions` with `{type: "text", name: "allergies_what", title: "What are you allergic to?"}`.
   - **What happened:** 409, "“What are you allergic to?” is a question of a repeating list in “Demo pre-op HQ”, so it can't go in the library on its own. Save the repeating group instead."
   - **What the plan expects:** "The library already has a question with the ID allergies_what, in “Your allergies”."
   - The refusal works, and this sentence is the clearer one. Only the plan's expected text differs.
   - **Related checks:**
     - `anaesthetic_problems_details` (a question inside a plain group) gave "The library already has a question with the ID anaesthetic_problems_details, in “Problems with anaesthetics”."
     - `allergies_reaction` gave the repeating-list sentence.
2. **Note: a library question with an empty title is accepted through the API.** A console POST with `title: ""` and a new ID returned 201. The row shows the ID as its title (SurveyJS's default), so nothing breaks. I removed it afterwards.
3. **Checks that worked as expected:**
   - Library search with leading and trailing spaces ("   smoke  ") found Do you smoke?.
   - "😀 smoke", `%`, `_` and `%_'"` returned "No library questions match." (no wildcard matching).
   - Save to library on the repeating group "Your allergies" (already in the library) was refused with "The library already has a question with the ID allergies."
   - DELETE of a library id that doesn't exist returned 404, "Library question not found."
   - A published questionnaire still offers **Save to library** (no Same as…), as the build notes say.
4. **Note: Same as… is offered on a question that already has a library key.** On "What do you weigh now?" (ID `weight`, inserted from the library), Same as… listed "Your weight" (ID `q_muihne`, S24 Lib B 2056) with Use this enabled. Using it would replace the library key with another one. The plan doesn't cover this; I'm noting it for the builder.
5. **Cosmetic, outside S24's scope:**
   - In the read-only editor (Sue, Cara), an empty Group or Section still reads "Add content or drag and drop existing content in to this Group".
   - The seeded "HJE Full HQ" description names the incumbent product. It's local data kept from before the reset.

## Console and network errors

- No console errors or warnings in the tabs I tracked.
- No unexpected 4xx or 5xx responses reached the page. The only real 4xx were the 403, 404 and 409 responses I triggered on purpose.
- **Recording oddity, not an app fault:** Claude in Chrome's network log showed status 503 for several mutating requests: the patient's submit, logout, dev login, the library DELETE and a publish. Each of these succeeded in the UI and in the data:
  - "Thank you, Sam" appeared;
  - sign-out and sign-in worked;
  - the item disappeared from the library;
  - the version became "V1 · PUBLISHED".
  
  A fetch from the page to the same DELETE route returned the real status (404 for an unknown id). I don't believe these 503s are real.

## GIFs

- `~/Downloads/S24-T01-retest.gif`: the main library path (insert, then the refused second insert, then the reload).
- `~/Downloads/S24-T07-retest.gif`: the main Same as… path.

## Hand checks for Rahul

None.
