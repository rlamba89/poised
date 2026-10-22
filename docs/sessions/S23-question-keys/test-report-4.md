# S23 test report 4 (retest after test report 3)

**22 passed, 0 failed, 0 blocked** (22 cases: T-01 to T-17, R-01 to R-05).

**Environment:**
- Commit `dc9bed8` on master, with S23 and the fixes uncommitted in the working tree (43 changed or new paths, as the builder left them).
- 22 Oct 2026, 10:35 to 14:15 BST. The run paused for a usage limit after T-17 and R-01 and resumed at about 14:00 on the same app and code.
- http://localhost:3000 (Vite), API on :8080, started by the conductor (`make dev`, migrated and seeded).
- Chrome through Claude in Chrome (deviceId `bac77f71-…`). Desktop viewport was 1014 px wide before the pause and 1523 px after it.
- Run number `<n>` = **43**.

**How the steps were run:**
- **Console snippets:** this time the browser tool ran the plan's `fetch` snippets in the page, including `PUT …/content` and `POST …/publish`. That covered T-05, T-08, T-09, T-12 (Cara) and T-17. T-12's Bea, Sue and no-sign-in checks were sent with `curl` using `POST /api/dev/login` cookies.
- **Creating questionnaires (deviation):** the New Questionnaire drawer couldn't be used reliably. Its Name field triggers the 1Password inline menu, after which every Claude-in-Chrome action on the tab fails with "Cannot access a chrome-extension:// URL of different extension". Because of this, I made the **empty** questionnaires (name and description only) with `POST …/questionnaires` through `curl` as Alex or Bea. T23 Source 43 was the exception: it was made in the drawer. Everything after that was done in the editor UI: Question Sets, pages, questions, settings, publishing and new versions. This is a tooling limit, not an app bug.
- **Phone size (R-02):** as before, `resize_window` didn't change the viewport. I ran the patient journey in a same-origin **390 px iframe** on a localhost page. Its `innerWidth` was 386, and `scrollWidth` was 386, so there was no horizontal scroll.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | The placeholder "Text question", "What is your favourite colour 43?" and "Pick a fruit 43" (Apple/Banana/Cherry) all showed **New question: won't pre-fill**. It published as "v1 · published", Read only. |
| T-02 | Pass | "what is your FAVOURITE colour 43" changed to **Pre-fills from 1 other HQ** after the save, with no reload. The list showed "T23 Source 43 · by same wording", and clicking the badge didn't select the card. **Keyboard (fix from report 3):** the badge is now a `<button>` with `aria-haspopup="dialog"`. Tab from the previous control focused it, with a visible ring. Enter opened the list and Escape closed it. Rewording: food gave New question, and colour gave Pre-fills again. A second copy changed to Number made both cards New question, and deleting it brought back Pre-fills. |
| T-03 | Pass | Both "Please give details 43" cards showed New question. After T23 Details 43 was published and Target reloaded, deleting one left the other on Pre-fills from 1 other HQ, listing "T23 Details 43 · by same wording". |
| T-04 | Pass | apple/Banana/Cherry gave Pre-fills (T23 Source 43 · by same wording). Cherry→Plum gave New question, and Plum→Cherry gave Pre-fills again. |
| T-05 | Pass | The console PUT copying Source's set into Target's Question Set 2 returned `200`. After a reload, the QS2 colour and fruit cards both showed Pre-fills, listing "T23 Source 43 · **by link**". The About you colour and fruit cards then showed New question (text used twice), while "Please give details 43" stayed on Pre-fills. |
| T-06 | Pass | Bea's "T23 Trust B 43" showed New question for the colour question (Trust A's Source wasn't counted), and it published. As Alex, `question-keys` for Target listed only T23 Source 43 for the QS2 colour key. **Trust A Demo HQ**, "Do you have diabetes?": "Pre-fills from **1136** other HQs", and the list scrolls. All 1,257 questionnaire ids in its response gave **404** when fetched as Bea in Trust B, so none is Trust B's. **Trust B Demo HQ** (as Bea), same question: 33 HQs. The response had no T23 or Trust A names, and all 33 ids opened (200) as Bea in Trust B. |
| T-07 | Pass | Each change in the v2 draft was refused with the exact text, and Publish was disabled. **Cherry removed:** "Question Set 1 · “Pick a fruit 43”: the option “Cherry” was removed. A published question's answers can't change: add a new question instead (published in “T23 Source 43” v1)." ⌘Z restored it. **Adding "Date"** gave "…the option “Date” was added…". **Banana→Bananas** gave "…the option “Banana” was reworded to “Bananas”…". By mistake I also reworded Apple→Bananas once, and that was refused correctly too. After moving Cherry up and rewording the question to "Choose a fruit 43", it published "v2 · published". |
| T-08 | Pass | Target QS2 Apple→Green apple was refused with "Question Set 2 · “Pick a fruit 43”: the option “Apple” was reworded to “Green apple”. … (published in “T23 Source 43” v2)." The console publish returned `422` with the expected `error` and that single problem, and after a reload it was still "v1 · draft". With Apple restored, it published. The different question text ("Pick" against "Choose") and option order weren't flagged. |
| T-09 | Pass | T23 Twice 43: the console copy of First into QS2 returned `200`. The window then showed "Question Set 2 · The question ID q_ggluxb is used “Height 43” in “First” and “Height 43” in “Question Set 2”. Each question needs its own ID.", and Publish was disabled. The console publish returned `422` with the same problem. After deleting the copy and adding a new Text, it published. |
| T-10 | Pass | The switch sits under Clinical, off, with its explanation. Turning it on showed the orange **Always ask fresh** badge. It stayed on through a reload, through Change type to Select Many, and through Clinical on and off (it was hidden while Clinical was on). The published v1 showed the badge, and in v2 the switch was still on. Statement, Calculation (Clinical on and off) and a plain Group had no switch. A repeatable Group offered the switch. With its default names "Group"/"Text question" it showed **New question: won't pre-fill**, which is report 3's found 1, now fixed. The Text inside it had no switch and no badge. |
| T-11 | Pass | A Text question in a Clinician Question Set (QS2 of T23 Fresh 43 v2) had no pre-fill, New question or Always ask fresh badge, and no switch. T23 Fresh 43 v2 was then published. |
| T-12 | Pass | **Sue:** Source 43 opened read-only. The colour badge "Pre-fills from 1 other HQ" listed "T23 Target 43 · by link". **Cara:** read-only, with no pre-fill badges, no `question-keys` request and no console errors. T23 Fresh 43 still showed the Always ask fresh badge. Cara's console fetch returned **403**. **Bea** with Trust A's address got **404**, with Trust A's org and Trust B's hospital **404**, and with Trust B's org and hospital and Trust A's questionnaire **404**. No sign-in returned 401, and Sue got 200. |
| T-13 | Pass | The copied `?set=…&page=…` address of Target QS2 opened the same page with badges in a new tab. Going to Questionnaires and then browser Back returned to the editor page with badges. |
| T-14 | Pass | **Select One→Select Many:** "“Colour 43”: its kind of answer changed from single choice to multiple choice. … (published in “T23 Shape Source 43” v1)." **kg→lb:** "“Weight 43”: its unit changed from “kg” to “lb”." **Number→Date:** "its kind of answer changed from number to date". Back to Number with kg, it published "v2 · published". |
| T-15 | Pass | Grid Target's Grid and Rating, created the same as Source's, both showed Pre-fills from 1 other HQ. Setting the Rating To to 5 and renaming Swim→Cycle turned both into New question. In Grid Source v2, removing row Swim and setting To to 5 gave "2 problems": "the row “Swim” was removed" and "its scale changed from 0 to 10 by 1 to 0 to 5 by 1". After ⌘Z twice, it published v2. |
| T-16 | Pass | In T23 Sign A 43 and Sign B 43, the Signature and File upload cards had no pre-fill or Always ask fresh badge, and no switch. `question-keys` returned `{"problems":[],"questions":{}}` for both. A BMI block (added on its own page) showed PATIENT and LOCKED only. |
| T-17 | Pass | T23 Odd 43: the console PUT of a Grid with string columns returned `200`. It rendered after a reload and published v1 with no error. T23 Logic 43 (R-01) then published as well. `question-keys` for Target, Grid Target, Odd, Logic and the Demo HQ all returned 200, and the editor labels still showed. |
| R-01 | Pass | In T23 Logic 43, "R01 dependent 43" is shown only when "R01 trigger 43" is answered. Deleting the trigger warned "Other items depend on this", and the header showed "1 logic problem". Publish was refused with "“R01 dependent 43”: it tests something that no longer exists (q_kdtctz)". After ⌘Z, it published v1. |
| R-02 | Pass | As Cara, I made an episode for the made-up patient "Mt S23 Fresh Rfour" (DOB 1 Feb 1980, test email) on T23 Fresh 43 v2. The invite from /qa/messages opened at 390 px (iframe). After the date of birth (typed, then Enter), it showed T23 Fresh 43 with no "fresh" or "pre-fill" text (checked in the iframe's text) and no horizontal scroll. I answered, finished the section and sent, then confirmed in the app's own modal: "Thank you, Mt". |
| R-03 | Pass | As Cara, both sets were validated ("Validated by Cara Clinician on 22/10/2026, 14:13"). The POA Summary looked as before, with nothing about pre-fill or Always ask fresh, and no console errors. |
| R-04 | Pass | The preview of T23 Fresh 43 Question Set 1, as Patient and as Clinician, showed no badges. The Clinician view also showed the clinician-only Calculation. |
| R-05 | **Pass** | **The key case.** In the seeded **Demo pre-op HQ** (Trust A) I clicked Create new version ("v2 · draft") and then Publish with no changes. The window said it was ready, with no problems, and v2 published ("v2 · published", Read only). GIF: `/Users/rahul/Downloads/S23-r4-R05.gif`. |

## Failures

None.

## Hand checks for Rahul

None needed.

## Found while exploring

1. **Works as designed: wording match ignores case and extra spaces.** In Grid Target 43, "   WHAT is your favourite   colour 43?  " (leading, double and trailing spaces) showed Pre-fills from 1 other HQ ("T23 Source 43 · by wording"). Target 43 also has this text but uses it twice, so it wasn't counted.
2. **Works as designed: a Section with its default name** gets no pre-fill badge. It isn't an answer.
3. **Report 3's found 1 (default Group names) is fixed.** See T-10.
4. **Report 3's found 2 (keyboard) is fixed.** See T-02.
5. **Change type options depend on the option count.** A Select One with three options offers only "Select Many" (no Yes/No), while one with two options also offers Yes/No. This looks intended and isn't S23.
6. **Tooling, not the app:** the 1Password extension's inline menu on the New Questionnaire Name field (and on the episode Surname field) blocks Claude-in-Chrome actions on the tab. See "How the steps were run".
7. **Still open from earlier reports, not S23:**
   - A question in a Clinician Question Set still shows the grey "PATIENT" chip.
   - The delete dialog for a Text or Number question says "with its options and disclosures".

## Console and network errors

- No console errors or warnings on the editor, preview, episode, validation, POA Summary or patient pages I checked.
- No unexpected 4xx or 5xx. The "503" readings for `POST /api/logout`, `POST /api/dev/login` and the patient `POST …/submit` are the tool artifact for `204` responses described in report 1. All three worked: sign-in changed, and the HQ was sent. The expected 422, 403, 404 and 401 responses came from the plan's checks.

## GIFs and test data

- **GIF:** R-05, the main happy path: `/Users/rahul/Downloads/S23-r4-R05.gif`.
- **Test data** (all made up, run 43, Trust A unless noted):
  - **Questionnaires:**
    - Published: T23 Source 43 (v2), T23 Target 43 (v1), T23 Details 43, T23 Twice 43, T23 Fresh 43 (v2), T23 Shape Source 43 (v2), T23 Grid Source 43 (v2), T23 Sign A 43, T23 Odd 43, T23 Logic 43.
    - Left as drafts: T23 Grid Target 43 (with an extra Text and Section from exploring) and T23 Sign B 43.
  - **Trust B:** T23 Trust B 43 (published).
  - **Seeded data changed:** the **Trust A Demo pre-op HQ is now v2 published** (R-05, published unchanged), so v1 is no longer the current version. No drafts were left on seeded HQs.
  - **Episode:** "Mt S23 Fresh Rfour" at Hospital A1 on T23 Fresh 43 v2, email `mt.s23.fresh.r4@test.example`. The patient submitted, and both sets are validated. The HQ review isn't completed.
