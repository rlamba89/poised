# S24 test report 3 (retest after verification 1)

**21 passed, 0 failed, 0 blocked** (16 cases + 5 regression). Verification 1's fixes all work in the browser:
- **T-15:** "Same as…" copies Always ask fresh.
- **T-16:** conditions follow the options by their text, an unmappable one becomes a logic problem that blocks Publish, and the warning comes before "Use this".
- **T-16 step 7:** another Question Set's condition is renamed too.
- **T-15:** a Yes / No question that takes three options becomes Select One.

**Environment**
- **Code:** commit `f845a98` with S24's uncommitted work in the tree. Git status isn't clean (47 changed or new files), as expected before verification.
- **When and where:** 23 Oct 2026, 09:59–10:28 BST, on http://localhost:3000. The API on :8080 reported health ok.
- **Database:** not reset since report 2. The Demo pre-op HQ was v3 published in Trust A, and its "Do you smoke?" has the three library options, so nothing was blocked. This run published **v4** (R-05).
- **Browser:** Chrome ("rahul chrome" profile) through Claude in Chrome, desktop at about 1549 px wide. The phone case (T-13) ran in a 390×844 same-origin iframe.

**Test data (run number 0959)**
- **Trust A questionnaires:**
  - `S24 Lib 0959`: draft, id `bf48a007…`;
  - `S24 Lib B 0959`: published v1, `c97909e5…`;
  - `S24 Meds 0959`: published v2, `ad07d0f4…`;
  - Demo pre-op HQ: now v4.
- **Episode:** "Sam S24 0959", Knee arthroscopy, made-up mobile 07700 900959. The answers were sent.
- **Trust library item:** `S24 job question 0959` (ID `q_uebtsk`). It was saved and then removed in T-06.
- **Trust B:** I only read data there, through Bea's existing draft `S24 Lib Bea 2056`.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Insert keeps the ID; second insert refused | Pass | `smoker` was inserted with its 3 options and "ID smoker". The card said "Pre-fills from 3 other HQs". The second insert, in Question Set 2, was refused under its row: "This HQ already has a question with the ID smoker. A question can be in an HQ only once (copy it to ask it again under a new ID)." After a reload it was still `smoker`. GIF: S24-T01-retest3.gif. |
| T-02 Search; group; repeating group; empty search on reopen | Pass | `ANAESTHETIC` found 3 rows, `measurements` found height and weight, and `zzz` showed "No library questions match." Problems with anaesthetics inserted with "Displays when … is Yes". Your allergies inserted as a repeating group. Weight showed the Always ask fresh badge and kg. Both windows opened with an empty search after × and after Escape. In Preview, Yes showed "What happened with your anaesthetic?". |
| T-03 Where From library isn't offered | Pass | The repeating group's own bar has no From library. From library was greyed on the BMI page. The published Demo HQ has no Add content bar. Sue and Cara got a read-only editor with no From library, Same as… or Save to library, and `GET library-questions` returned 403 for both. |
| T-04 Save to library; another trust can't see it | Pass | Saved with the tag Work ("Saved. The trust's authors can now insert it…"). The row had the "This trust" badge, WORK and Remove. Ada saw it. Bea didn't, in the UI or with `?q=job` (0 items; the 11 platform items are listed), and her GET of Trust A's library returned 404. |
| T-05 Save to library refusals | Pass | `smoker` was refused with "The library already has a question with the ID smoker." The job question was refused with "…ID q_uebtsk." The reworded diabetes "No" (Demo v4 draft) gave the full "isn't the same as the trust's published one" sentence; I reverted it. Statement, Calculation, Signature, File upload and Group have no Save to library; the repeating group has one. "What are you allergic to?" has neither Same as… nor Save to library. The console POST returned 409 with the repeating-list sentence (accepted by the plan). |
| T-06 Remove; inserted copies stay | Pass | The red "Remove from the library?" box appeared. Cancel kept the item; Remove → Remove gave "No library questions match." Both questionnaires still hold `q_uebtsk` after a reload, checked in the UI and through the chapters API. |
| T-07 Same as… main path | Pass | Before: "New question: won't pre-fill". After Use this, with no warning (nothing to lose): "ID smoker" and "Linked to “Do you smoke?” in Demo pre-op HQ". The question kept its own text and took the 3 options. Always ask fresh stayed off. The label read "Pre-fills from 3 other HQs" (Demo pre-op HQ, S24 Lib B 2056 and S24 Lib B 2131, all "by link"). All of it survived a reload. An added option was undone with ⌘Z. GIF: S24-T07-retest3.gif. |
| T-08 Incompatible candidates greyed with the reason | Pass | Text: `smoker` showed the ID reason, and `q_smokes` showed "Its answer is single choice; this question's is text." With lb, the weight candidates showed "Its unit is “kg”; this question's is “lb”."; with kg, Use this was enabled. Your allergies showed the repeating-list sentence. Select Many showed "…single choice; this question's is multiple choice." and "Its answer is text; this question's is multiple choice." |
| T-09 Refused when the ID is already in the HQ | Pass | "This HQ already has a question with this ID, in “About you”." Use this was disabled. Same as… isn't offered when Clinical is on, on a repeating list's row question, on the repeating group itself, or on a Statement or Calculation (also Signature and File upload). |
| T-10 Deleting an adopted option refused at Publish | Pass | "Question Set 1 · “Are you a smoker at the moment?”: the option “I used to smoke” was removed. A published question's answers can't change…" It names Demo pre-op HQ v3, S24 Lib B 2056 v1 and S24 Lib B 2131 v1, and Publish was disabled. After ⌘Z it published v1 with "ID smoker" and "Linked to…". |
| T-11 Medication type frozen | Pass | Recreational in v2 was refused: "…its medication type changed from prescribed medication to recreational drugs. A published question's answers can't change…". Set back to Prescribed, v2 published. |
| T-12 Seeded Demo HQ uses library questions | Pass | The structure matches the plan. I checked the IDs through the UI and the chapters API: `smoker`, `alcohol_units` (units a week), `height` (cm), `weight` (kg, alwaysFresh), `q_diabetes`, `snoring`, `pregnant` (alwaysFresh), `ponv_history`, both anaesthetic groups, `allergies` and `regular_medicines`. Trust B's Demo HQ (as Bea) has the same keys, with weight and pregnant fresh. |
| T-13 Demo HQ on a phone (390×844) | Pass | Knee arthroscopy picked Demo pre-op HQ **v3**, the latest at the time. The SMS link and date of birth worked. The patient saw 2 sections, the 3 smoke options, the alcohol hint and "units a week", and cm and kg. Yes showed the details box. The allergy and medicine lists started empty, and their Add buttons showed the boxes. Nothing overflowed (scrollWidth = clientWidth = 388). The patient saw "Thank you, Sam". |
| T-14 Reload, deep link and Back | Pass | The `?set=&page=` link opened in a new tab with "ID smoker" and "Linked to…". Back from the open library window went to the Questionnaires list; Forward came back to the editor without the window. |
| **T-15 Same as… copies Always ask fresh** | Pass | The `pregnant` row showed the orange "Always ask fresh, like the original". After Use this: ID `pregnant`; Yes / No / I'm not sure; the orange card badge; Settings showed **Select One** (stored as a radiogroup without `yesNo`) with Always ask fresh on. Switched off, the card said "Pre-fills from 1 other HQ" (Demo pre-op HQ · by link). Switched on, the badge came back. GIF: S24-T15-retest3.gif. |
| **T-16 Same as… and conditions and disclosures** | Pass | See the notes below this table. GIF: S24-T16-retest3.gif. |
| R-01 (S23) pre-fill label; Always ask fresh | Pass | "Pre-fills from N other HQs" with "by link". New questions say "New question: won't pre-fill". Always ask fresh can be switched both ways and its badge shows (T-15). |
| R-02 (S23) rewording refused at Publish | Pass | "Your health · “Do you have diabetes?”: the option “No” was reworded to “Not at all”…". Publish was disabled. |
| R-03 (S19) Total hip replacement → Demo pre-op HQ | Pass | It picked Demo pre-op HQ v3 (checked as Cara). |
| R-04 (S14/J3) patient sends an HQ on a phone | Pass | Covered by T-13. |
| R-05 (S23) Demo HQ new version publishes | Pass | v4 was created; the T-05/R-02 rewording was typed back to "No"; v4 published. |

**T-16 in detail.** I set it up in `S24 Lib 0959` Question Set 2:
- "Do you smoke at all?" with the options Yes / Never / Sometimes, and a disclosure on Never.
- "How many a day?", displayed when it's Yes.
- "Why did you stop?", displayed when it's Sometimes.
- For step 7, I added Question Set 3, displayed when "Do you smoke at all?" is Yes.

What happened:
- **The warning:** Same as… → `snoring` → Use this warned "Its own disclosures on 1 option will be removed. 1 condition tests its old answers and will need changing (Publish is refused until then)."
- **Cancel:** the question was unchanged.
- **Use this anyway:** ID `snoring`, options Yes / No, and "Linked to “Do you snore loudly?”". The disclosure is gone from the stored JSON.
- **The conditions:**
  - "How many a day?" still displays when it's **Yes**. It's stored as `{snoring} = 'yes'`.
  - Question Set 3's condition was renamed and mapped to `{snoring} = 'yes'`.
- **The logic problem:**
  - The header showed "1 logic problem": "“Why did you stop?”: it tests an answer that no longer exists (“Do you smoke at all?”: o_glyj)".
  - The Publish window listed the same problem, and Publish was disabled.
  - After I changed that condition to "No", the problem cleared and the Publish window allowed it. I cancelled without publishing.

## Failures

None.

## Found while exploring

1. **Cosmetic: the "Use this" button in the Same as… list is squashed to a sliver ("U") on rows with a long source list.**
   - **Where:** for example, `q_smokes` and `q_diabetes`, which are published in about 40 "E2E …" questionnaires.
   - **What happens:** the list of HQ names takes the row's width and the button shrinks to about 15 px.
   - Seen in T-07, T-08 and T-09 (Alex, `S24 Lib B 0959` and `S24 Lib 0959`). Today it comes from the automated tests' data, but a real trust with many HQs would see the same.
2. **Minor: the Save to library window keeps its last "Saved." state on the same question.**
   - **Steps:** after a successful save, click Done, then click **Save to library** again on the same question.
   - **What happens:** the window reopens showing "Saved. The trust's authors can now insert it…" with Done, not the form.
   - Selecting another question resets it, and saving again is refused anyway. Harmless, but inconsistent with the other windows, which now start fresh each time.
3. **Note (not S24): my sign-in expired mid-test.**
   - At 10:02, the From library window showed a red "Please sign in." box and `/api/me` returned 401.
   - The cause is the 12-hour session TTL: the browser's cookie came from report 2's sign-in at about 22:02 the night before.
   - The editor stayed open and showed no other sign that the session had ended. Signing in again worked, and nothing was lost (the last change had saved).
4. **Note: an unmappable condition shows its raw value on the card.**
   - After T-16's adoption, the card reads "Displays when Do you smoke at all? is o_glyj", and the Logic tab's value box is empty.
   - The logic-problem message flags it, so it isn't silent. A friendlier label such as "(a removed answer)" may read better.
5. **Checks that worked as expected:**
   - Double-clicking Use this, on a question with nothing to lose, adopts once.
   - Switching Clinical on hides Same as…, and switching it off shows it again.
   - Adopting a three-option question with no disclosures or conditions shows no warning and drops the old third option (by design).
   - Typing in a window's search before the list loads isn't lost. One miss I saw was my own click below the box.
6. **Cosmetic, outside S24 (since 1 Oct):** the page header in the editor reads "Clinical Page" even on a patient Question Set's page (`Canvas.tsx`).

## Console and network errors

- No console errors or warnings in the tabs I tracked.
- No unexpected 4xx or 5xx responses. The only 4xx were the ones I triggered on purpose: 403 for Sue and Cara, 404 for Bea's request to Trust A, 409 for the repeating-list POST, and the expired session's 401 described above.

## GIFs

Saved in `~/Downloads/`:
- `S24-T01-retest3.gif`: the main library path.
- `S24-T07-retest3.gif`: the main Same as… path.
- `S24-T15-retest3.gif`: Always ask fresh is copied.
- `S24-T16-retest3.gif`: the warning, Cancel, Use this anyway, and the logic problem at Publish.
