# S23 test report 5 (retest after verification 2)

**23 passed, 0 failed, 0 blocked** (23 cases: T-01 to T-18, R-01 to R-05).

**Environment:**
- Commit `dc9bed8` on master. S23 and the fixes are uncommitted in the working tree (47 changed or new paths, as the builder left them).
- 22 Oct 2026, 19:08 to 19:45 BST.
- http://localhost:3000 (Vite), API on :8080. The conductor started it (`make dev`, migrated and seeded after the database reset).
- Chrome through Claude in Chrome (Rahul's "rahul chrome" profile, deviceId `bce1f965-…`). The desktop viewport was about 2560 CSS px wide.
- Run number `<n>` = **51**.

**How the steps were run:**
- **Creating questionnaires:** this time the New Questionnaire drawer worked. I filled Name and Description with the tool's form-fill, not by typing into the field, so the 1Password inline menu never opened. Every questionnaire in this run was made in the drawer, and everything after that in the editor UI.
- **Console snippets:** the browser tool ran the plan's `fetch` snippets in the page: T-05, T-08 and T-09 (`PUT …/content`, `POST …/publish`), T-12 (Cara) and T-17. I also ran the publish snippet on T-18 step 5 to confirm the server agrees with the window. T-12's checks for Bea, Sue and no sign-in went through `curl` with `POST /api/dev/login` cookies.
- **Switching users:** T-06 used the avatar menu's Sign out and then the stub login page. Later switches (Bea → Alex, Alex → Sue → Cara → Alex) called `POST /api/logout` and `POST /api/dev/login` from the page. This is the same request the login page sends.
- **Phone size (R-02):** I opened the invite link in a same-origin **390 px iframe**, as in earlier rounds. Inside it, `innerWidth` and `scrollWidth` were both 388, so there was no horizontal scroll.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | The placeholder "Text question", "What is your favourite colour 51?" and "Pick a fruit 51" (Apple/Banana/Cherry) all showed **New question: won't pre-fill**. The HQ published as "v1 · published", Read only. |
| T-02 | Pass | "what is your FAVOURITE colour 51" showed **Pre-fills from 1 other HQ** after the save, with no reload. The list said "T23 Source 51 · by same wording", and clicking the badge didn't select the card. **Keyboard:** Tab reached the badge with a focus ring, Enter opened the list and Escape closed it. Rewording to "food" gave New question, and rewording back gave Pre-fills. A second copy changed to Number made both cards New question. Deleting it (in-app modal) brought back Pre-fills. |
| T-03 | Pass | Both "Please give details 51" cards showed New question. After T23 Details 51 was published and Target reloaded, I deleted one. The other then showed Pre-fills from 1 other HQ, listing "T23 Details 51 · by same wording". |
| T-04 | Pass | apple/Banana/Cherry gave Pre-fills (T23 Source 51 · by same wording). Cherry→Plum gave New question, and Plum→Cherry gave Pre-fills again. |
| T-05 | Pass | The console PUT copying Source's set into Target's Question Set 2 returned `200`. After a reload, the QS2 colour and fruit cards both listed "T23 Source 51 · **by link**". The About you colour and fruit cards then showed New question (text used twice), while "Please give details 51" stayed on Pre-fills. |
| T-06 | Pass | Bea's "T23 Trust B 51" colour question showed New question, and the HQ published. As Alex, Target QS2's colour list showed only "T23 Source 51 · by link". In the Trust A Demo HQ, Smoking, "Do you smoke?" showed "Pre-fills from 34 other HQs" and the list scrolls. All 34 listed questionnaire ids opened (200) as Alex in Trust A. In the Trust B Demo HQ (as Bea), "Do you have diabetes?" showed Pre-fills from 1 other HQ ("E2E J7c B …"). The whole Trust B Demo HQ `question-keys` response named only that one questionnaire, with no T23 or Trust A names. |
| T-07 | Pass | Each change in the v2 draft was refused with the exact text, and Publish was disabled. **Cherry removed:** "Question Set 1 · “Pick a fruit 51”: the option “Cherry” was removed. A published question's answers can't change: add a new question instead (published in “T23 Source 51” v1)." ⌘Z restored it. **Adding "Date":** "…the option “Date” was added…". **Banana→Bananas:** "…the option “Banana” was reworded to “Bananas”…". After moving Cherry up and rewording the question to "Choose a fruit 51", the HQ published as "v2 · published". |
| T-08 | Pass | Target QS2 Apple→Green apple was refused: "Question Set 2 · “Pick a fruit 51”: the option “Apple” was reworded to “Green apple”. … (published in “T23 Source 51” v2)." The console publish returned `422` with the expected `error` and that single problem. After a reload it was still "v1 · draft". With Apple restored, it published. |
| T-09 | Pass | T23 Twice 51: the console copy of First into QS2 returned `200`. Publish then showed "Question Set 2 · The question ID q_csxzqy is used “Height 51” in “First” and “Height 51” in “Question Set 2”. Each question needs its own ID." with Publish disabled. The console publish returned `422` with the same problem. After I deleted the copy and added a new Text, it published. |
| T-10 | Pass | The switch sits under Clinical, off, with its explanation. Turning it on showed the orange **Always ask fresh** badge, and it stayed on through a reload and Change type → Select Many. With Clinical on, the switch and badge were hidden and no pre-fill badge showed; with Clinical off, both were back. The published v1 showed the badge, and in v2 the switch was still on. Statement, Calculation (Clinical on and off) and a plain Group had no switch. A repeatable Group offered the switch and, with the default names, showed New question. The Text inside it had no switch and no badge. |
| T-11 | Pass | A Text question in a Clinician Question Set (QS2 of T23 Fresh 51 v2, `audience: clinician` confirmed by GET) had no pre-fill, New question or Always ask fresh badge, and no switch. T23 Fresh 51 v2 was then published. |
| T-12 | Pass | **Sue:** Source 51 opened read-only, with no Create new version. The colour badge listed "T23 Target 51 · by link". **Cara:** read-only, no pre-fill badges, no console errors, and the editor sent no `question-keys` request. T23 Fresh 51 still showed the Always ask fresh badge. Cara's console fetch returned **403**. **Bea** got **404** with Trust A's address, with Trust A's org and Trust B's hospital, and with Trust B's org and hospital plus Trust A's questionnaire. No sign-in returned 401, and Sue got 200. |
| T-13 | Pass | The copied `?set=…&page=…` address of Target QS2 opened the same page with badges in a new tab. Questionnaires (the URL was confirmed as the list) followed by browser Back returned to the editor page with both badges. |
| T-14 | Pass | **Select One→Select Many:** "“Colour 51”: its kind of answer changed from single choice to multiple choice. … (published in “T23 Shape Source 51” v1)." **kg→lb:** "“Weight 51”: its unit changed from “kg” to “lb”." **Number→Date:** "its kind of answer changed from number to date". Changing back to Number cleared the unit; after I set kg again it published "v2 · published". |
| T-15 | Pass | Grid Target's Grid and Rating, made the same as Source's, both showed Pre-fills from 1 other HQ (T23 Grid Source 51 · by same wording). Setting To to 5 and renaming Swim→Cycle turned both into New question. In Grid Source v2, removing row Swim and setting To to 5 gave "2 problems": "the row “Swim” was removed" and "its scale changed from 0 to 10 by 1 to 0 to 5 by 1". After ⌘Z twice, it published v2. |
| T-16 | Pass | In T23 Sign A 51 (published) and Sign B 51 (draft), the Signature and File upload cards had no pre-fill or Always ask fresh badge, and no switch. `question-keys` for Sign B returned `{"problems":[],"questions":{}}`. A BMI block on its own page showed only PATIENT and LOCKED. |
| T-17 | Pass | T23 Odd 51: the console PUT of a Grid with string columns returned `200`. It rendered after a reload and published v1 with no error. T23 Logic 51 (R-01) published after it. `question-keys` returned 200 for Target, Grid Target, Odd, Logic, Format Target and the Demo HQ, and the editor labels still showed. |
| T-18 | **Pass** | **The key case.** **Step 1:** T23 Format Source 51 (Date "Date of your operation 51", Day, month and year; Text "Contact 51", Any text) published v1. **Step 2:** In T23 Format Target 51, both showed Pre-fills from 1 other HQ ("T23 Format Source 51 · by same wording"). Year only on the date and Email address on the text turned both into **New question: won't pre-fill**. **Step 3 (Source v2):** Year only was refused with exactly "Question Set 1 · “Date of your operation 51”: its date format changed from day, month and year to year only. A published question's answers can't change: add a new question instead (published in “T23 Format Source 51” v1)." Month and year gave "…from day, month and year to month and year…". **Step 4:** Email address gave "“Contact 51”: its answer format changed from any text to email address…". Phone number gave "…from any text to the pattern “^\+?[0-9 ()-]{7,20}$”…". Changing back to Any text published v2. Each time only one problem was listed, so the date put back was accepted. **Step 5:** v3 added a repeatable Group with a Select One "Which hospital 51" (Hospital One/Hospital Two) and published. In v4 I reworded "Hospital Two"→"Hospital 2" and the question text to "Which hospital were you in 51". It was refused with **exactly one** problem: "“Which hospital were you in 51”: the option “Hospital Two” was reworded to “Hospital 2”…(published in “T23 Format Source 51” v3)". There was no second problem about the group's column. The console publish returned `422` with the same single problem. With the option put back (new question text kept), v4 published. GIF: `/Users/rahul/Downloads/S23-r5-T18.gif`. |
| R-01 | Pass | In T23 Logic 51, "R01 dependent 51" is shown only when "R01 trigger 51" is answered. Deleting the trigger warned "Other items depend on this". Publish was refused with "“R01 dependent 51”: it tests something that no longer exists (q_ccshzw)". After ⌘Z, it published v1. |
| R-02 | Pass | As Cara, I made an episode for the made-up patient "Mt S23 Fresh Rfive" (DOB 1 Feb 1980, test email) on T23 Fresh 51 v2. The invite link from the captured messages opened at 390 px. After the date of birth (typed, then Enter), it showed T23 Fresh 51 with no "Always ask fresh" or pre-fill text in the questionnaire, and no horizontal scroll. I answered, finished the section and sent, then confirmed in the app's own modal: "Thank you, Mt". |
| R-03 | Pass | As Cara, both sets were validated ("Validated by Cara Clinician on 22/10/2026, 19:43"). The POA Summary looked as before, with no pre-fill or Always ask fresh text and no console errors. |
| R-04 | Pass | The preview of T23 Fresh 51 Question Set 1, as Patient and as Clinician, showed no badges. The Clinician view also showed the clinician-only Calculation. The only "fresh" on the page was the HQ name in the breadcrumb. |
| R-05 | Pass | In the seeded **Demo pre-op HQ** (Trust A), Create new version gave "v2 · draft". Publish with no changes said it was ready, and v2 published ("v2 · published", Read only). |

## Failures

None.

## Hand checks for Rahul

None needed.

## Found while exploring

1. **Works as designed: the lock also covers the other answer-shape switches I tried.** These are all in T23 Format Source 51's v5 draft:
   - "Allow several dates" on the published date was refused with "its kind of answer changed from date to list of dates".
   - Turning on the "Don't know" special option on the question inside the repeating group was refused with "the option “Don't know” was added".
   - Adding a Text question inside the published repeating group was refused with "“Group”: the column “Text question” was added".

   After I undid each change, only the remaining one was reported.
2. **Works as designed: the wording match ignores case, punctuation and emoji.** In T23 Format Target 51 I typed "  DATE of your operation 51!! 🙂", set back to Day, month and year. It showed Pre-fills from 1 other HQ.
3. **Cosmetic, for Rahul to judge:** the phone-format refusal shows the raw pattern to the author: "…changed from any text to the pattern “^\+?[0-9 ()-]{7,20}$”". The plan expects this ("the phone pattern"), but "phone number" would read better, like "email address" does.
4. **Not S23, for the record:** the Text question's Format list offers "NHS number" and "UK postcode". These are country-specific options in the editor (CLAUDE.md, A-20). They were probably there before S23.
5. **Still open from earlier reports, not S23:**
   - A question in a Clinician Question Set still shows the grey "PATIENT" chip.
   - The delete dialog for a Text or Number question says "with its options and disclosures".

## Console and network errors

- No console errors or warnings on the editor, preview, episode, validation, POA Summary or patient pages after tracking began.
- No unexpected 4xx or 5xx. The "503" readings for `POST /api/logout`, `POST /api/dev/login` and the patient `POST …/submit` are the tool artifact for `204` responses described in report 1. All three worked: the sign-in changed, and the HQ was sent. The expected 422, 403, 404 and 401 responses came from the plan's checks.

## GIFs and test data

- **GIF:** T-18 steps 3 and 4 (date and text format refusals, then publish), the key case: `/Users/rahul/Downloads/S23-r5-T18.gif`.
- **Test data** (all made up, run 51, Trust A unless noted):
  - **Questionnaires:**
    - Published: T23 Source 51 (v2), T23 Target 51 (v1), T23 Details 51, T23 Twice 51, T23 Fresh 51 (v2), T23 Shape Source 51 (v2), T23 Grid Source 51 (v2), T23 Sign A 51, T23 Odd 51, T23 Logic 51.
    - T23 Format Source 51 is v4 published, with a **v5 draft left** that holds an extra Text column in the repeating group (from exploring).
    - Left as drafts: T23 Format Target 51 (date reworded with an emoji from exploring), T23 Grid Target 51 and T23 Sign B 51 (with a BMI page).
  - **Trust B:** T23 Trust B 51 (published).
  - **Seeded data changed:** the **Trust A Demo pre-op HQ is now v2 published** (R-05, published unchanged). No drafts were left on seeded HQs.
  - **Episode:** "Mt S23 Fresh Rfive" at Hospital A1 on T23 Fresh 51 v2, email `mt.s23.fresh.r5@test.example`. The patient submitted, and both sets are validated. The HQ review isn't completed.
