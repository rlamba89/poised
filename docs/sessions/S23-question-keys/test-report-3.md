# S23 test report 3 (retest after verification 1)

**21 passed, 1 failed, 0 blocked** (22 cases: T-01 to T-17, R-01 to R-05).

**Environment:**
- Commit `05a1a5b` on master, with S23 and the verification 1 fixes uncommitted in the working tree (40 changed or new paths, as the builder left them).
- 22 Oct 2026, 09:47–10:25 BST.
- http://localhost:3000 (Vite), API on :8080, started by the conductor (`make dev`, migrated and seeded).
- Chrome through Claude in Chrome (deviceId `bac77f71-…`), desktop viewport 1400×823.
- Run number `<n>` = **42**.

**How the console steps were run:** as in report 1, the `PUT …/content` and `POST …/publish` snippets (T-05, T-08, T-09, T-12, T-17) were sent with `curl` to the local API, using cookies from `POST /api/dev/login` for Alex, Bea, Cara and Sue. The requests, ids and bodies were the same as the snippets. Every result was then checked in the browser.

**Phone size (R-02):** `resize_window` again didn't change the viewport while I needed it (it took effect several minutes later, at 500 px wide). So I ran the patient journey inside a same-origin **390×780 iframe** on a localhost page. Media queries apply to the iframe's own viewport, and its `innerWidth` was 386 with no horizontal scroll.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | "Text question" (the placeholder) and then "What is your favourite colour 42?" showed the grey **New question: won't pre-fill** next to PATIENT. The same was true of "Pick a fruit 42" (Apple/Banana/Cherry). Published as "v1 · published", Read only. |
| T-02 | Pass | "what is your FAVOURITE colour 42" changed to **Pre-fills from 1 other HQ** after the save, with no reload. The list showed "T23 Source 42 · by same wording". Clicking the badge on an unselected card didn't select it. Rewording to "food" gave New question, and rewording back gave Pre-fills. With a second copy changed to Number, both cards showed New question. Deleting the Number brought back Pre-fills. |
| T-03 | Pass | Both "Please give details 42" cards showed New question. After T23 Details 42 was published and the page reloaded, deleting one left the other on Pre-fills from 1 other HQ, listing "T23 Details 42 · by same wording". |
| T-04 | Pass | apple/Banana/Cherry gave Pre-fills (T23 Source 42 · by same wording). Cherry→Plum gave New question, and Plum→Cherry gave Pre-fills again. |
| T-05 | Pass | Copying Source's set into Target's Question Set 2 returned `200`. After a reload, both QS2 cards showed Pre-fills, with the list "T23 Source 42 · **by link**". The colour and fruit cards in About you now showed New question (text used twice), while QS2's still matched by link. |
| T-06 | Pass | Bea's "T23 Trust B 42" showed New question: Trust A's Source wasn't counted. As Alex, Target QS2's colour list showed only T23 Source 42. Trust A Demo HQ, "Do you have diabetes?": **1139 other HQs**, and the list scrolls. The API response named no Trust B questionnaire. Trust B Demo HQ (as Bea), "Do you have diabetes?": 33 HQs, and the response had no T23 Source, T23 Target or other Trust A names. |
| T-07 | Pass | v2 draft. Each change was refused with the new wording, and Publish was disabled each time. **Cherry removed**: "Question Set 1 · “Pick a fruit 42”: the option “Cherry” was removed. A published question's answers can't change: add a new question instead (published in “T23 Source 42” v1)." ⌘Z restored Cherry. **"Date" added**: "…the option “Date” was added…". **Banana→Bananas**: "…the option “Banana” was reworded to “Bananas”…". After moving Cherry up and rewording the question to "Choose a fruit 42", it published "v2 · published". |
| T-08 | Pass | Target QS2 Apple→Green apple was refused with "Question Set 2 · “Pick a fruit 42”: the option “Apple” was reworded to “Green apple”. … (published in “T23 Source 42” v2)." The API publish returned **422** with the expected `error` and this problem, and it stayed "draft 1". After Apple was restored it published. Source v2 has a different option order (Apple, Cherry, Banana), and that wasn't flagged, as expected. |
| T-09 | Pass | "T23 Twice 42": copying First into QS2 returned `200`. The window showed "Question Set 2 · The question ID q_gmccdh is used “Height 42” in “First” and “Height 42” in “Question Set 2”. Each question needs its own ID." with Publish disabled. The API returned **422** with the same problem. After deleting the QS2 copy and adding a new Text question, it published. |
| T-10 | Pass | The "Always ask fresh" switch sits under Clinical, off, with its explanation. Turning it on showed the orange badge and "All changes saved". It stayed on through a reload, through Change type to Select Many, and through Clinical on and off (with Clinical on, the badge and switch were hidden and no pre-fill badge showed). The published v1 showed the badge, and in v2 the switch was still on. Statement, Calculation (Clinical on or off) and Group had no switch. A repeatable Group offered it, and a Text inside that Group had no switch and no badge. See Found while exploring 1. |
| T-11 | Pass | In a Clinician Question Set, the Text question had no pre-fill, New question or Always ask fresh badge, and no switch. |
| T-12 | Pass | **Sue:** Source was read-only. "Pre-fills from 1 other HQ" listed "T23 Target 42 · by link". **Cara:** Source was read-only, with no pre-fill badges, no `question-keys` request and no console errors. T23 Fresh 42 still showed the Always ask fresh badge. Cara's GET `question-keys` returned **403**. Bea with Trust A's address got **404**, and with Trust B's hospital and Trust A's questionnaire also **404**. With no sign-in it returned 401, and Sue got 200. |
| T-13 | Pass | The copied `?set=…&page=…` address opened the same page with badges in a new tab. Questionnaires then browser Back returned to the editor page, with badges. |
| T-14 | Pass | **Select One→Select Many** was refused with "“Colour 42”: its kind of answer changed from single choice to multiple choice. … (published in “T23 Shape Source 42” v1)." **kg→lb**: "its unit changed from “kg” to “lb”". **Number→Date**: "its kind of answer changed from number to date". Changing back to Number cleared the unit, and that was refused too ("from “kg” to none"). With kg set again, it published "v2 · published". |
| T-15 | Pass | Grid Target's Grid and Rating, created the same as Source's, both showed Pre-fills from 1 other HQ (T23 Grid Source 42 · by same wording). Setting the Rating's To to 5 and renaming row Swim→Cycle turned both into New question. In Source v2, removing row Swim and setting To to 5 was refused with 2 problems: "the row “Swim” was removed" and "its scale changed from 0 to 10 by 1 to 0 to 5 by 1". After ⌘Z twice (restoring To 10 and the row), it published v2. |
| T-16 | Pass | In T23 Sign A 42 and T23 Sign B 42, the Signature and File upload cards had no pre-fill, New question or Always ask fresh badge, and no switch. `question-keys` for both returned `{"problems":[],"questions":{}}`. The BMI block showed PATIENT and LOCKED only, with no settings panel. |
| T-17 | Pass | T23 Odd 42: the PUT of a Grid with string columns returned `200`. It rendered after a reload and **published** with no error. Then T23 Logic 42 (R-01) published as well. `question-keys` for Target, Grid Target, Odd and Demo HQ all returned 200, and the editor labels still showed. |
| R-01 | Pass | In T23 Logic 42, "R01 dependent 42" is shown only when "R01 trigger 42" is answered. Deleting the trigger warned that the dependent relies on it, the header showed "1 logic problem", and publishing was refused with "“R01 dependent 42”: it tests something that no longer exists (q_gdatgd)". After ⌘Z it published v1. |
| R-02 | Pass | Cara made an episode for the made-up patient "Mt S23 Fresh Rthree" on T23 Fresh 42 v2, with a test email. The invite from /qa/messages opened at **390 px** (iframe; see Environment). After the date of birth (typed, then Enter) it showed T23 Fresh 42 with no "fresh" or "pre-fill" text anywhere (checked in the iframe's text) and no horizontal scroll. I answered, finished the section and sent, then confirmed in the app's own modal: "Thank you, Mt". |
| R-03 | Pass | As Cara, both sets were validated ("Validated by Cara Clinician"). The POA Summary looked as before, with nothing about pre-fill or Always ask fresh, and no console errors. |
| R-04 | Pass | The preview of T23 Fresh 42 Question Set 1, as Patient and as Clinician, showed no badges. The Clinician view also showed the clinician-only Calculation. |
| R-05 | **Fail** | A new version of the seeded **Demo pre-op HQ** (Trust A), left unchanged, is **refused with 5 problems** on "Do you have diabetes?". See Failure 1. |

## Failures

### 1. Major: an unchanged new version of the seeded Demo pre-op HQ can't be published (R-05)

**Signed in as:** Alex Author.
**URL:** http://localhost:3000/o/6ef150fb-16c5-4d4f-a4be-1c2b6f22638b/h/00000000-0000-4000-8000-00000000000a/questionnaires/00000000-0000-4000-8000-000000000101

**Steps:**
1. Open Demo pre-op HQ (v1, published).
2. Click **Create new version**. It shows "v2 · draft".
3. Without changing anything, click **Publish**.

**Expected:** the window says it's ready, and v2 publishes. Its keys and options are the same as its own v1.

**Actual:** "5 problems to fix first", and Publish is disabled. The API publish also returns **422** with the same 5 problems:
- Your health · “Do you have diabetes?”: its kind of answer changed from Yes/No switch to single choice. … (published in “S02 test HQ 15:51” v1).
- Your health · “Do you have diabetes?”: the option “Yes” was removed. … (published in “S02 test HQ 16:20” v1 and “S02 test HQ 23:06” v1).
- … the option “No” was removed. (same HQs)
- … the option “Yes” was added. (same HQs)
- … the option “No” was added. (same HQs)

**Why (checked through the API):** the key `q_diabetes` is published in several Trust A HQs with **different answer shapes**, all published before the lock existed:

| HQ | `q_diabetes` |
| --- | --- |
| Demo pre-op HQ v1 | radiogroup Yes=`o_yes` / No=`o_no` |
| S02 test HQ 15:51 v1 | **boolean** (Yes/No switch) |
| S02 test HQ 16:20 v1 (and 23:06) | radiogroup Yes=`o_diay` / No=`o_dian` |

The lock compares the draft against **every** published HQ that holds the key, so no single shape can satisfy them all. The Demo HQ, and each of the S02 HQs, is now permanently unpublishable without dropping the question.

The messages also contradict each other: "Yes was removed" and "Yes was added" appear for the same question, which leaves an author no way forward.

The S02 HQs are old local test data. But the same thing will happen to any trust whose questionnaires already disagree on a key before this session's lock: imported or older data, and the local demo database used by the e2e journeys.

**Possible direction (for the builder, not a fix):** a question that is unchanged from the HQ's **own** previous published version arguably shouldn't be refused because of other HQs that already disagreed. It may also help to merge "removed"/"added" pairs per source HQ.

**GIF:** `/Users/rahul/Downloads/S23-R05.gif`

**Console and network:** no console errors. The only failed request was `POST …/publish` → 422, from the API check.

## Hand checks for Rahul

None needed. Report 1's hand checks are now covered: R-02 at phone size (390 px iframe) and R-05 (it ran, and failed).

## Found while exploring

1. **Minor: an editor-placeholder repeating group matches by wording.** In T23 Fresh 42 v2, I added a Group, switched on Repeatable and added a Text question inside it, keeping the default names "Group" and "Text question". The Group's badge became **"Pre-fills from 1 other HQ"**, listing "T23 Fresh 41 · by same wording" (report 1 built the same default group there). T-01 says the editor's own placeholder text never matches by wording. Here a list whose title and column are both placeholders does. Before the inner Text was added, the empty repeating group showed New question.
2. **Minor (keyboard), still open from report 1:** the pre-fill badge is a `div` with `tabIndex -1`, and its `role=button` wrapper also has `tabIndex -1`. So a keyboard user can't open the list, and **Escape doesn't close** an open list; only clicking elsewhere does. (Escape does close the Publish window.)
3. **Works as designed:**
   - Changing only "Always ask fresh" and "Required" on a published key (T23 Fresh 42 v3 draft) leaves the Publish window ready.
   - Clearing the unit, by changing type away and back, is refused ("from “kg” to none").
   - Moving options is allowed.
4. **Better than report 1:** `question-keys` for the Trust A Demo HQ is now about **370 KB** (it was about 0.97 MB) and 108 ms locally. The response no longer has a `published` part. Small HQs return under 400 B.
5. **Test plan detail:** T-06 step 3 asks for Trust A Demo HQ's "Smoking" page. Trust A's Demo HQ has no such page either: it has About you, Your health (page Health) and Nurse checks. The key list includes `q_smokes`. I checked "Do you have diabetes?" in the UI and the whole response through the API instead.
6. **Cosmetic, not S23:** the delete dialog for a Number or Text question says "… will be deleted, with its options and disclosures", although these have no options.
7. **Not S23 (as in report 1):** a question in a Clinician Question Set still shows the grey "PATIENT" chip on its card.

## Console and network errors

- No console errors or warnings on the editor, preview, episode, validation, POA Summary or patient pages I checked.
- No unexpected 4xx or 5xx. The "503" readings for `POST /api/logout` and `POST /api/dev/login` are the tool artifact for `204` responses described in report 1 (sign-in and sign-out worked). The expected 422, 403, 404 and 401 responses came from the API checks.

## GIFs and test data

- **GIFs:**
  - Main happy path (T-01 and T-02): `/Users/rahul/Downloads/S23-r3-T01-T02-happy-path.gif`
  - Failure: `/Users/rahul/Downloads/S23-R05.gif`
- **Test data**, all made up, run 42, Trust A unless noted:
  - **Questionnaires:**
    - T23 Source 42 (v2 published), T23 Target 42 (v1), T23 Details 42, T23 Twice 42.
    - T23 Fresh 42 (v2 published, plus a **v3 draft**: I clicked "Create new version" by mistake, then used it to check that "Always ask fresh" and "Required" aren't frozen).
    - T23 Shape Source 42 (v2), T23 Grid Source 42 (v2), T23 Grid Target 42 (draft), T23 Sign A 42, T23 Sign B 42 (draft), T23 Odd 42, T23 Logic 42.
  - **Trust B:** T23 Trust B 42.
  - **Seeded data changed:** **Demo pre-op HQ (Trust A) now has a v2 draft**, left by R-05 because it can't be published. v1 is still the published version.
  - **Episode:** "Mt S23 Fresh Rthree" at Hospital A1 on T23 Fresh 42 v2, email `mt.s23.fresh.r3@test.example`. Both sets are validated; the HQ review isn't completed.
