# S24 test report 4 (retest after verification 2, the last round)

**21 passed, 0 failed, 0 blocked** (16 cases + 5 regression). Verification 2's fix works in the browser:
- **The cross-set hole is closed (T-16 step 8):**
  - When "Same as…" leaves a later Question Set's condition testing an answer with no match, the window warns first.
  - After adopting, the set's logic-problem button and the Publish window both refuse it.
  - Changing the condition to "is Yes" clears the problem.
  - With "is Yes" from the start, there's no warning and the condition follows to `snoring` = Yes.
- **The same check guards a plain edit too** (found while exploring): deleting an option that a later set's condition tests is refused at Publish.

**Environment**
- **Code:** commit `f845a98` with S24's uncommitted work in the tree. Git status isn't clean (50 changed or new files), as expected before verification.
- **When and where:** 23 Oct 2026, 10:47–11:14 BST, on http://localhost:3000. The API on :8080 reported health ok.
- **Database:** not reset since report 3. The Demo pre-op HQ was v4 published; this run published **v5** (R-05).
- **Browser:** Chrome ("rahul chrome" profile) through Claude in Chrome, desktop at 1512 px wide. The phone case (T-13) ran in a 390×844 same-origin iframe.

**Test data (run number 1050), all in Trust A**
- **Questionnaires:**
  - `S24 Cross 1050`: draft, T-16 step 8, unmapped condition. The "Occasional smokers" condition is now "is Yes".
  - `S24 Cross Yes 1050`: draft, T-16 step 8, mapped condition. It also holds the inserted `q_zugsrf`.
  - `S24 Lib B 1050`: published v1, T-07, T-08 and T-10.
  - `S24 Lib 1050`: draft, T-01, T-02, T-04, T-05, T-09, T-15, T-16 and the exploring below. It has open logic problems, left on purpose.
  - `S24 Meds 1050`: published v2, T-11.
  - Demo pre-op HQ: now v5.
- **Episode:** "Sam S24 1050", Knee arthroscopy, made-up mobile 07700 900105. The answers were sent.
- **Trust library item:** `S24 job question 1050` (ID `q_zugsrf`). It was saved and then removed in T-06.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Insert keeps the ID; second insert refused | Pass | `smoker` was inserted into "About you" with 3 options and "ID smoker". Inserting it again in Question Set 2 was refused under the row: "This HQ already has a question with the ID smoker. A question can be in an HQ only once (copy it to ask it again under a new ID)." |
| T-02 Search; group; repeating group; empty search | Pass | `ANAESTHETIC` found the 3 rows, `measurements` found height and weight, and `zzz` showed "No library questions match." Problems with anaesthetics inserted with "Displays when … is Yes". Your allergies inserted as a repeating group. Weight showed the Always ask fresh badge and kg. Both windows opened with an empty search each time. Preview (step 8) wasn't rerun. |
| T-03 Where From library isn't offered | Pass (partly rerun) | The selected repeating group's own bar has no From library; the page's bar has one. As Cara: the editor was read-only, with no Same as…, Save to library or From library, and `GET library-questions` returned 403. Sue and the BMI page weren't rerun (both passed in report 3). |
| T-04 Save to library; another trust can't see it | Pass | Saved with the tag Work: "Saved. The trust's authors can now insert it…". The row had the This trust badge, WORK and Remove. Ada's `?q=job` lists it. Bea's Trust B `?q=job` returned `[]` (11 platform items in all), and her GET of Trust A's library returned 404. I checked Ada and Bea through the API from the signed-in page, not in their library windows. |
| T-05 Save to library refusals | Pass | `smoker`: "The library already has a question with the ID smoker." The job question: "…the ID q_zugsrf." The reworded diabetes "No" (Demo v5 draft) gave the full "isn't the same as the trust's published one … reworded to “Not at all”" sentence; I reverted it. The console POST of `allergies_what` returned 409 with the repeating-list sentence. |
| T-06 Remove; inserted copies stay | Pass | The red "Remove from the library?" box appeared. Cancel kept the item; Remove → Remove gave "No library questions match." `q_zugsrf` is still in `S24 Lib 1050` and `S24 Cross Yes 1050` (checked through the chapters API). |
| **T-07 Same as… main path** | Pass | It started as "New question: won't pre-fill". After Use this, with no warning: "ID smoker", "Linked to “Do you smoke?” in Demo pre-op HQ", its own text kept, and the 3 options taken. The label read "Pre-fills from 4 other HQs" (Demo pre-op HQ and S24 Lib B 0959, 2056 and 2131, all "by link"). All of it survived a reload. An added option was undone with ⌘Z. GIF: S24-T07-retest4.gif. |
| **T-08 Incompatible candidates greyed** | Pass | Text: "Its answer is single choice; this question's is text." (`smoker` showed the "already has a question with this ID" reason). With lb: "Its unit is “kg”; this question's is “lb”."; with kg, Use this was enabled. Your allergies showed the repeating-list sentence. Select Many: "…single choice; this question's is multiple choice." |
| **T-09 Refused when the ID is in this HQ** | Pass | "This HQ already has a question with this ID, in “About you”." Use this was disabled. With Clinical switched on, Same as… isn't shown. |
| **T-10 Deleting an adopted option refused at Publish** | Pass | "Question Set 1 · “Are you a smoker at the moment?”: the option “I used to smoke” was removed. A published question's answers can't change…". It names Demo pre-op HQ v4 and three S24 Lib B HQs, and Publish was disabled. After ⌘Z it published v1 with "ID smoker" and "Linked to…". |
| T-11 Medication type frozen | Pass | Recreational in v2 was refused: "…its medication type changed from prescribed medication to recreational drugs…". Set back to Prescribed, v2 published. |
| T-12 Seeded Demo HQ uses library questions | Pass (spot-check) | Demo v5: About you, Your health and Nurse checks. Health holds `q_diabetes`, `snoring` and `pregnant` (Always ask fresh). The patient form (T-13) showed every library question. Trust B's HQ wasn't rechecked. |
| T-13 Demo HQ on a phone (390×844) | Pass | Knee arthroscopy picked **Demo pre-op HQ v5**. The SMS link and date of birth led to 2 sections. The patient saw 3 smoke options, the alcohol hint and "units a week", cm and kg. Yes showed "What happened with your anaesthetic?". The allergy and medicine lists started empty, and their Add buttons showed the boxes. scrollWidth = clientWidth = 388. The patient saw "Thank you, Sam". |
| T-14 Deep link, Back and Forward | Pass | The `?set=&page=` link opened in a new tab and showed "ID smoker" and "Linked to…". Back from the open library window left the editor; Forward came back without the window. |
| **T-15 Same as… copies Always ask fresh** | Pass | The row showed "Always ask fresh, like the original". After Use this: ID `pregnant`; Yes / No / I'm not sure; the orange badge; **Select One**; Always ask fresh on. Switched off: "Pre-fills from 1 other HQ" (Demo pre-op HQ · by link). Switched on again: the badge came back. GIF: S24-T15-retest4.gif. |
| **T-16 Same as… and conditions and disclosures (steps 1–6)** | Pass | The warning read "Its own disclosures on 1 option will be removed. 1 condition tests its old answers and will need changing (Publish is refused until then)." Cancel left the question unchanged. After Use this anyway: ID `snoring`; "How many a day?" kept `{snoring} = 'yes'`; the disclosure was gone. The header showed "1 logic problem", "“Why did you stop?”: it tests an answer that no longer exists (“Do you smoke at all?”: o_fqkr)", and Publish was disabled with the same line. GIF: S24-T16-retest4.gif. |
| **T-16 step 8 (verification 2): a later Question Set's condition** | Pass | See below. GIF: S24-T16-step8-retest4.gif. |
| R-01 (S23) pre-fill label; Always ask fresh | Pass | "Pre-fills from N other HQs" with "by link". New questions say "New question: won't pre-fill". Always ask fresh switched both ways (T-15). |
| R-02 (S23) rewording refused at Publish | Pass | "Your health · “Do you have diabetes?”: the option “No” was reworded to “Not at all”…". Publish was disabled. |
| R-03 (S19) Total hip replacement → Demo HQ | Pass | As Cara, it picked "Demo pre-op HQ v5". |
| R-04 (S14/J3) patient sends an HQ on a phone | Pass | Covered by T-13. |
| R-05 (S23) Demo HQ new version publishes | Pass | v5 was created, the rewording was typed back to "No", and v5 published. |

**T-16 step 8 in detail**, as Alex in `S24 Cross 1050`:
- **Setup:**
  - Set "Smoking" has "Do you smoke at all?" with Yes / Sometimes.
  - Set "Occasional smokers" (a page and a Text question) has the Question Set condition "Do you smoke at all? is Sometimes".
- **Same as… → `snore` → Use this:** the window warned "1 condition tests its old answers and will need changing (Publish is refused until then)." **Use this anyway** gave ID `snoring` (Yes / No), labelled "Pre-fills from 1 other HQ".
- **The refusal:**
  - **Publish:** "1 problem to fix first: Occasional smokers · Question Set condition: it tests an answer that no longer exists (“Do you smoke at all?”: o_uure)". Publish was disabled.
  - **The set itself:** opening "Occasional smokers" showed "1 logic problem" in the header: "This Question Set: it tests an answer that no longer exists (“Do you smoke at all?”: o_uure)". The condition's option box was empty.
- **After changing the condition to "is Yes":** the header problem cleared, and Publish offered "Publish version 1?" with Publish enabled (I cancelled). The stored condition is `{snoring} = 'yes'`.
- **The mapped variant** (`S24 Cross Yes 1050`, the condition "is Yes" before adopting): Use this adopted at once with no warning. The second set's condition was stored as `{snoring} = 'yes'`, and Publish showed no problems (I cancelled).

## Failures

None.

## Found while exploring

1. **Minor: ⌘Z after "Same as…" restores only the current Question Set.**
   - **What happens:** another set's condition that the adoption renamed keeps the new key, which is no longer in the HQ.
   - **Steps** (Alex, `S24 Lib 1050`):
     1. Question Set 3 has the condition "Do you currently smoke? is Option 2". That question is in "About you".
     2. On "Do you currently smoke?": Same as… → `sick` → **Have you ever been sick…** (`ponv_history`) → Use this anyway. The warning counted 1 condition, as expected.
     3. Set 3's condition is now `{ponv_history} = 'o_qunt'`.
     4. Click the canvas and press ⌘Z.
   - **Expected:** both the question and set 3's condition go back.
   - **Actual:**
     - The question goes back to `q_jpqhzo` with Option 1 / 2.
     - Set 3's condition stays `{ponv_history} = 'o_qunt'`, now naming a key that isn't in the HQ.
     - It isn't silent: Publish refuses it with "Question Set 3 · Question Set condition: it tests something that isn't in an earlier Question Set (ponv_history)". The author has to fix the condition by hand.
2. **Good: the verification 2 check also guards plain edits, not only "Same as…".**
   - **Steps:** in `S24 Lib 1050`, set 3's condition was "Do you currently smoke? is Option 3", and I deleted Option 3 from that question in "About you".
   - **Result:** Publish listed "Question Set 3 · Question Set condition: it tests an answer that no longer exists (“Do you currently smoke?”: o_sawk)", and the set's header button showed it after a reload.
3. **Minor (already in pending.md): the Save to library window still reopens on "Saved." for the same question.**
   - This time, selecting another question and coming back didn't reset it. Report 3 said it did.
   - A page reload reset it. Saving again was then correctly refused: "The library already has a question with the ID q_zugsrf."
4. **Cosmetic (already in pending.md):** the squeezed **Use this** button on `q_smokes` rows, whose source list names about 40 "E2E …" HQs.
5. **Note:** in the Same as… list, the `q_smokes` row now also shows "Always ask fresh, like the original". So at least one E2E HQ marks `q_smokes` fresh, and the "fresh if any copy is fresh" rule shows it. That's as designed; it comes from the automated tests' data.
6. **Note:** on the T-08 Text question, the `smoker` row showed only the "already has a question with this ID" reason, not the kind mismatch. The plan accepts either refusal.

## Console and network errors

- **Console:** no errors or warnings on the editor tab once I started tracking (late in the run, during the exploring).
- **Network:** every `/api/` request I checked returned 200, including the content PUTs on two sets and the `question-keys/search` POSTs.
- **4xx I triggered on purpose:** 403 for Cara's library GET, 404 for Bea's GET of Trust A's library, and 409 for the repeating-list POST.

## GIFs

Saved in `~/Downloads/`:
- `S24-T16-step8-retest4.gif`: verification 2's new case. It shows the warning, Use this anyway, the Publish refusal, the set's logic problem, and the condition changed to Yes.
- `S24-T07-retest4.gif`: the main Same as… path.
- `S24-T15-retest4.gif`: Always ask fresh is copied.
- `S24-T16-retest4.gif`: the disclosure and condition warning, Cancel, Use this anyway, and the logic problem at Publish.

## Hand checks for Rahul

None.
