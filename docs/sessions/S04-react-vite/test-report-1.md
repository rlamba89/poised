# S04 test report 1

**22 passed, 1 failed, 0 blocked** (23 cases: T-01 to T-18 and R-01 to R-05). The one failure is Minor: scroll position isn't restored on Back (T-16 step 4).

- **Commit:** `ac8ae27`. The working tree is not clean: S04's changes are uncommitted (about 76 paths in `git status`).
- **When:** Mon 12 Oct 2026, 17:13–17:47 BST.
- **App:** http://localhost:3000 (API :8080 healthy), with `make dev` running in Rahul's terminal.
- **Browser:** Claude in Chrome, desktop window (2560 px wide at first, then 1512 × 757).
- **Phone size:** Chrome wouldn't resize below its minimum window width. Following the plan, I used a same-origin 390 × 844 iframe, so the patient page really had a 390 px viewport.
- **Test data (all made up):**
  - questionnaire `S04 Test1714`: v1 published, v2 draft;
  - patient `S04 Test1714`, episode `ebc27d03…`, now at POA complete;
  - patient `S04 Test1733`, episode `d8dea9a2…`, ready for review, used for R-02 and R-03;
  - questionnaire `S04 Delete1742`, created and then deleted (R-05).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 New app, name Poised | Pass | The tab title is **Poised**, and `/icon.svg` is the blue tick. The page shows the heading Poised, "Development sign-in: choose a user." and 4 user cards with emails. The source is a short page (608 bytes) with `<div id="root">` and `/src/main.tsx`, and no `_next`. The old name appeared on no screen in any case. |
| T-02 Sign-in, start page | Pass | Alex lands on `/h/<A>/questionnaires` with the header "Poised · Hospital A · Questionnaires (highlighted) · Alex Author · Sign out" and no Episodes. `/` takes Alex back to the list. Cara lands on `/h/<A>/episodes` with Episodes highlighted. Signed-out `/` goes to `/login`, with the 401 from `/api/me` as expected. |
| T-03 Signed-out deep link | Pass | It goes to `/login`, and no `/api/h/…` request is made on the way (I checked the network log). |
| T-04 List, new questionnaire | Pass | Search "E2E J3" narrowed the list, and clearing it restored it. Creating it opened the full-screen editor with "v1 · draft" and a **Questionnaires** link. The link went back to the list, where the row showed DRAFT, and clicking the row reopened the editor. |
| T-05 Structure tree, canvas | Pass | Add Question Set added `?set=`, and renaming worked. Content was kept after a reload, with `?set` kept. Switching sets changes `?set` and adds no history entries (`history.length` stayed 13). Back went to the list, and Forward came back to About you. Plan note: you need **Add Page** first; see "Test plan notes". |
| T-06 Settings, Logic | Pass | The Settings, Disclosures and Logic tabs show. The Builder condition "Do you smoke? is Yes" gives the orange border and "Displays when…". The Code tab shows `{q_emrfvn} = 'o_apey'`. Required and the condition were both kept after a reload, and the panel closes. Required was already on: new questions default to Required (see notes). |
| T-07 Disclosures | Pass | Yes → SNOMED 77176002 Smoker, note "Patient smokes", category Lifestyle (the category is required, and leaving it out shows a clear message). The card shows a green chip with **1**, kept after a reload. |
| T-08 Preview, test cases | Pass | The preview at `/h/<A>/chapters/<id>/preview` has the Poised header. Yes reveals "Do you vape?", and Clinical outputs show 77176002 Smoker and Patient smokes. Patient/Clinician and Phone/Tablet/Desktop all change the layout, with no console errors. Test case `Smoker` → Run all → "All 1 passed", and it was still listed after a reload. Edit → editor with `?set`. Preview, then Back → editor on About you. |
| T-09 Publish, versions | Pass | Publish dialog → "v1 · PUBLISHED" + "READ ONLY", with no Add content and no editable fields. Create new version → "v2 · DRAFT", the same content, and nothing selected. The list shows version 2 / DRAFT. Val Viewer opens it Read only, with no Publish or Add content. |
| T-10 Episodes list, new episode | Pass | The status filter (POA complete) narrowed it to 48 rows, all POA COMPLETE, and clearing it restored all 176. New episode with a new patient → `/h/<A>/episodes/<id>`. The Episodes link → list (my episode at the top) → row → episode. |
| T-11 Episode page | Pass | The patient link starts `http://localhost:3000/p/`. About you and Your health say "Waiting for the patient". After a reload: the same episode and the same link. |
| T-12 Patient link at 390 × 844 | Pass | The dark **Poised** bar, "Hello S04", and the section tiles. `scrollWidth` = 390, so there's no sideways scroll. Yes → vape appears, No, "Saved". After a reload the answers were kept: the tile said COMPLETE and Review showed Yes/No. Send my answers → in-app confirm → "Thank you, S04", still shown after a reload. A tampered token gives the red "This link doesn't work. Please check it, or contact the hospital.", with the expected `/api/p/…` 404. |
| T-13 Validate a Question Set | Pass | "READY FOR REVIEW". Validate → `/episodes/<eid>/sets/<cid>` with the patient's answers, the same after a reload. Changed to No ("Patient answered: Yes") → Validate → back on the episode, "Validated by Cara Clinician". Your health validated too. Back → the previous (validate) screen, not an error. Complete HQ review → READY FOR POA. |
| T-14 POA Summary | Pass (Print partly checked) | `/poa` with the Episodes / Episode breadcrumbs and the two tabs. Validated: "Changed by the clinician on Page 1 · Do you smoke? Patient answered “Yes”, now “No”". Patient answers: no change box, and the Yes disclosure ("Patient smokes") listed. Status → POA complete added "Status changed to POA complete". The same after a reload. Episode / Back / Episodes all go where they say. **Print:** I didn't open the real print dialog, because it freezes the browser extension. I stubbed `window.print` (Print calls it once) and read the print CSS: it hides the breadcrumbs, Print, the status picker and the header. **Rahul, please click Print once and look at the preview.** |
| T-15 Every address reloads / new tab | Pass | All 9 rows open directly and reload correctly, with no 404, blank page or sign-in prompt, and no console errors: `/login` signed out; the list, the editor with `?set`, and the preview as Alex; the episodes list, the episode, `sets/<cid>` and `/poa` as Cara; and `/p/<token>` signed out at 390 px. |
| T-16 Back, Forward, scrolling | **Fail** (steps 1–3 pass) | Back ×2 and Forward ×2 between the list, the episode and the POA Summary are correct. A newly opened page starts at the top. **But Back does not return to the scroll position:** see Failure 1. |
| T-17 Not-found page | Pass | All 5 addresses show **404** / "This page could not be found." with the title Poised, the same after a reload, and no console messages. Back → the previous page. |
| T-18 Refused paths | Pass | Bea lands on Hospital B, and the picker lists only Hospital B. The Hospital A list, editor and POA addresses all say "You don't have access to this hospital.", and no Hospital A API call was made. Alex on the episode (and on `/poa`) sees "Only clinicians can see patients and episodes.", with the expected 403. |
| R-01 Two editors | Pass | Tab A changed the smoking question's text and saved. Tab B's edit got "Someone else changed this Question Set" + **Reload** and "Not saved". Reload showed tab A's text and dropped tab B's. |
| R-02 Two validate tabs | Pass | Tab A changed vape → Yes (PUT 200, Saved). Tab B's change got "Not saved … changed somewhere else" + **Reload**, a 409, and Validate disabled. Reload showed tab A's answers, so nothing from tab B was saved. |
| R-03 Patient on two "devices" | Pass | Two 390 px frames on one fresh link. A answered Yes and saved, then B answered No: "Your answers weren't saved" + **Reload**, and Reload showed Yes. (Cara was still signed in during this; the patient API uses only the link.) |
| R-04 Duplicate name | Pass | "A draft questionnaire with this name already exists." in the dialog, with POST 409. Nothing was created. |
| R-05 Delete a draft | Pass | ⋯ → Delete → the in-app "Delete questionnaire?" modal → Delete. The row is gone. The DELETE shows as 503, which is the known 204 quirk. |

## Failures

### 1. Back doesn't return to the scroll position on long lists (T-16 step 4): Minor

- **User:** Cara Clinician (Episodes list) and Alex Author (Questionnaires list).
- **URL:** `/h/00000000-0000-4000-8000-00000000000a/episodes` (and `/questionnaires`).
- **Steps:**
  1. Open the Episodes list (176 rows) and scroll down about 1000 px.
  2. Click an episode row: the episode opens at the top, which is correct.
  3. Press the browser's Back.
- **Expected:** back on the list at about where you were scrolled (about 1000 px).
- **Actual:** the list opens at the very top (`scrollY` = 0, both times I tried). The page is long enough by then (`scrollHeight` 11284). The Questionnaires list behaves the same (500 → 0).
- **Console and network:** nothing unexpected.
- **GIF:** `~/Downloads/S04-T16.gif`.
- **Note for the builder:** `pages/Root.tsx` uses `<ScrollRestoration />`. The rows are fetched after the page mounts, so the page is probably still short when the scroll position is restored. I couldn't compare with the Next.js build, so I can't say whether it worked before S04.

## Found while exploring

Apart from the last item, none of these comes from S04, as far as I can tell (the code behind them is unchanged in the diff). I list them for completeness.

1. **The stale-editor banner says "chapter"** (Cosmetic). The R-01 heading says "Question Set", but the body says "Someone else changed this **chapter** since you opened it…". Users don't see the word "chapter" anywhere else.
2. **The delete modal flashes an empty name as it closes** (Cosmetic). After confirming, the fading modal reads `"" and all its Question Sets will be deleted…`.
3. **The hospital picker goes blank on a hospital you can't access** (Cosmetic). For example, Cara on `/h/<B>/episodes`: the picker is empty beside "You don't have access to this hospital."
4. **Refused editor addresses have no header** (Minor UX). Bea on `/h/<A>/questionnaires/<qid>` sees only the red alert on a blank page, with no Poised header, Sign out or link back. The list and POA addresses keep the header.
5. **Alex sees a "New episode" button** above "Only clinicians can see patients and episodes." on `/h/<A>/episodes` (Cosmetic).
6. **The printed POA Summary will still include screen-only controls** (Cosmetic, unchanged from before S04). The print CSS hides only `.sj-no-print` and the header. So **Add comment**, the **Validated summary / Patient answers** tabs and the section chevrons would print.
7. **An empty Question Set can be published** (Minor, not S04). "Your health" had no pages and still published. The patient sees it as COMPLETE, and the validate screen shows SurveyJS's default "The survey doesn't contain any visible elements." plus "The patient didn't answer this Question Set."
8. **These worked well:**
   - double-clicking Save in New Questionnaire sent only one POST;
   - a stale patient tab changing an answer after the submit gets "Your answers weren't saved. Your answers have already been sent…" (409);
   - an unknown episode ID or a non-UUID gives "Episode not found.";
   - Back after the validate screen, and reloads in the middle of a flow, all behaved.
9. **Not tried:** emoji and leading spaces in a questionnaire name. My click to open the dialog didn't register (see the tool notes), and I stopped there.

## Test plan notes (for the builder, not app bugs)

- **T-05 step 3:** "Add content:" appears only after **Add Page** (a new Question Set has no page). The plan doesn't mention this step.
- **T-06 step 2:** new questions are already **Required** (`doc.ts` sets `isRequired: true`), so "tick Required" is a no-op.
- **T-14 step 3:** the Patient answers tab shows the disclosure the Yes produces ("Patient smokes"), not the literal word "Yes". That matches the design but not the plan's wording.

## Console and network errors not tied to a case

- **Console:** no red errors or yellow warnings on any screen, in either tab, all session. I confirmed the capture works by logging a test warning.
- **Network:** only the expected codes:
  - 401 on `/api/me` while signed out;
  - 403 or 404 where a case caused it;
  - 409 in R-01 to R-04;
  - 503 for 204 responses (logout, delete, submit), which is the known tool quirk.
- **Duplicate GETs:** every page makes its GETs twice (for example `/api/me` and `/api/dev/users`). I assume that's React StrictMode in development, and harmless.

## Tool notes

- **Phone size:** Chrome's minimum window width stopped a 390 px window, so I used same-origin 390 × 844 iframes, as the plan allows.
- **Missed clicks:** a few times the first click after switching tabs or resizing didn't register. Twice it was a sign-in card, where no `/api/dev/login` request was sent; another time it was a card or button. Clicking again always worked. I believe this is the extension, not the app.

## GIFs

- `~/Downloads/S04-happy-path-authoring.gif`: the main happy path (sign-in → new questionnaire → Question Sets → questions → Logic → disclosure → preview → test case → publish → new version → Val read-only).
- `~/Downloads/S04-T16.gif`: Failure 1 (scroll position lost on Back).
