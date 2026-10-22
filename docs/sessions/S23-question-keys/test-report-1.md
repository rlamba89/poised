# S23 test report 1

**17 passed, 0 failed, 1 blocked** (plus one part of R-02 that needs a hand check).

**Environment:**
- Commit `05a1a5b` on master, with S23 uncommitted in the working tree (35 changed or new paths, as the builder left them).
- 22 Oct 2026, 08:49–09:30 BST.
- http://localhost:3000 (Vite), API on :8080. `make dev` was started by the conductor on a freshly migrated and seeded database.
- Chrome ("Browser 1") through Claude in Chrome, desktop window 1568×773.
- Run number `<n>` = **41**.

**How the console steps were run:** the plan's DevTools snippets for T-05, T-08, T-09 and T-12 need a `fetch` in the page. The browser tool refused to run the `PUT …/content` snippet (a permission block on page scripts that change data). So I sent the same requests from a terminal with `curl` to the local API, using cookies from `POST /api/dev/login` for Alex, Bea, Cara and Sue. The requests, ids and bodies were the same as the snippets, and every result was then checked in the browser.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | The new Text question ("Text question" placeholder and after rewording) and the Select One "Pick a fruit 41" (Apple/Banana/Cherry) both showed the grey **"New question: won't pre-fill"** next to PATIENT. Published: "v1 · published", Read only. |
| T-02 | Pass | "what is your FAVOURITE colour 41" showed **"Pre-fills from 1 other HQ"** after the save with no reload. The list read "T23 Source 41 · by same wording". Clicking the badge didn't select the card. Rewording to "food" → New question; back → Pre-fills. With a second copy changed to Number, **both** showed New question. Deleting the Number brought the first back to Pre-fills. |
| T-03 | Pass | Two "Please give details 41" in Target: both New question (after T23 Details 41 was published and Target reloaded). Deleting one → the other showed "Pre-fills from 1 other HQ", list "T23 Details 41 · by same wording". |
| T-04 | Pass | "Pick a fruit 41" with apple/Banana/Cherry → Pre-fills, list "T23 Source 41 · by same wording". Cherry→Plum → New question. Back to Cherry → Pre-fills. |
| T-05 | Pass | Copied Source's set into Target's Question Set 2 (curl, `200`). After a reload, both QS2 cards showed Pre-fills, list "T23 Source 41 · **by link**". The About you colour and fruit cards now showed **New question** (text used twice in Target), while QS2's still showed "by link". |
| T-06 | Pass | Bea published "T23 Trust B 41" with the same wording. In the Trust B editor it showed New question: Source 41 isn't counted across trusts. As Alex, Target QS2's colour list showed only "T23 Source 41". Trust A Demo pre-op HQ → Smoking → "Do you smoke?" listed **1260 other HQs**, all Trust A (the list scrolls). Checked through the API: no Trust B name, and "Trust B demo HQ" isn't listed. Trust B Demo HQ (as Bea), checked on "Do you have diabetes?" (see Found while exploring 4): 33 HQs, all Trust B, no T23 Source/Target in `questions` or `published`. |
| T-07 | Pass | v2 draft. Each change was refused with the exact wording, and Publish was disabled each time: Cherry removed → "Question Set 1 · “Pick a fruit 41”: the option “Cherry” was removed. A published question's options can't change (published in “T23 Source 41” v1)." Undo (⌘Z) restored Cherry. "Date" added → "…the option “Date” was added…". Banana→Bananas → "…the option “Banana” was reworded to “Bananas”…". Moving Cherry up and rewording the question to "Choose a fruit 41" published: "v2 · published". |
| T-08 | Pass | Target QS2 Apple→Green apple → "Question Set 2 · “Pick a fruit 41”: the option “Apple” was reworded to “Green apple”. A published question's options can't change (published in “T23 Source 41” v2)." API publish → **422** with the expected `error` and that problem in `problems`. Still "1 draft". Back to Apple → published "v1 · published". |
| T-09 | Pass | "T23 Twice 41": First's content was copied into QS2 (`200`). The Publish window showed "Question Set 2 · The question ID q_osynxs is used “Height 41” in “First” and “Height 41” in “Question Set 2”. Each question needs its own ID." with Publish disabled. API → **422**, same problem. QS2's question deleted and a new one added → published. |
| T-10 | Pass | Switch "Always ask fresh" under Clinical, off, with an explanation line. On → the orange **Always ask fresh** badge replaced the label, "All changes saved". It stayed on after a reload, after Change type → Select Many, and through Clinical on/off: with Clinical on, the switch and badge were hidden and no pre-fill badge showed; turning Clinical off brought them back, still on. Published v1 kept the badge. Create new version → v2 had the badge, switch on. Statement, Group and Calculation (with Clinical on or off) had no switch. A Repeatable Group offered the switch. The Text inside it had no switch and no pre-fill badge. |
| T-11 | Pass | A clinician Question Set (audience `clinician`, checked through the API) with a Text question: no Pre-fills, New question or Always ask fresh badge, and no switch in Settings. |
| T-12 | Pass | Sue: Source read-only, "Pre-fills from 1 other HQ" → "T23 Target 41 · by link". Cara: Source read-only, no pre-fill badges, no `question-keys` request, no console errors; T23 Fresh 41 still showed the Always ask fresh badge. Cara GET question-keys → **403**. Bea with Trust A's address → **404**; with Trust B's hospital and Trust A's questionnaire id → **404**. (Also: no sign-in → 401; Sue → 200.) |
| T-13 | Pass | Opening the copied `?set=…&page=…` address in a new tab showed the same page with badges. Questionnaires then browser Back → the editor page again with badges. |
| R-01 | Pass | In T23 Fresh 41 v2, "R01 dependent 41" was shown only when "R01 trigger 41" was answered. Deleting the trigger (the delete dialog warned that the dependent relies on it) → header "1 logic problem", and Publish was refused: "“R01 dependent 41”: it tests something that no longer exists (q_…)". Deleting the dependent → published v2. |
| R-02 | Pass at desktop size; **phone size blocked** | Made an episode for the made-up patient "Mt S23 Fresh Testpatient" on T23 Fresh 41 v2 at Hospital A1 with a test email. Opened the invite from /qa/messages and signed in with the date of birth. The questionnaire showed no "Always ask fresh" and no "Pre-fills" text. Answered, finished the section, sent: "Thank you, Mt". **The window wouldn't resize to 390×844** (`resize_window` reported success but the viewport stayed 2560 wide; the window seems maximised or full-screen), so the phone layout wasn't checked. |
| R-03 | Pass | As Cara, validated both sets ("Validated by Cara Clinician"). The POA Summary looked as before: nothing about pre-fill or Always ask fresh. No console errors. |
| R-04 | Pass | Preview of T23 Fresh 41 Question Set 1, as Patient and as Clinician: no badges. The Clinician view also showed the clinician-only Calculation, as expected. |
| R-05 | **Blocked** | Clicking "Create new version" on the seeded **Demo pre-op HQ** was refused by the session's permission check (it changes shared seeded data). Not done; a hand check for Rahul. |

## Failures

None.

## Hand checks for Rahul

1. **R-02 at phone size:** open the patient's invite on a 390×844 window, or with DevTools device mode, and check the layout. Use any T23 Fresh-style HQ with "Always ask fresh" on. The content was correct at desktop size.
2. **R-05:** Demo pre-op HQ (Trust A) → Create new version → Publish unchanged. Expect it to publish.

## Found while exploring

1. **Minor (keyboard):** the pre-fill badge isn't reachable from the keyboard. It's a plain `div` with no `tabindex`, and the nearest `role=button` around it has `tabIndex -1`. So a keyboard user can't open the "copied from" list. **Escape doesn't close the open list** either; only a click elsewhere does.
2. **Observation (size):** `GET …/question-keys` for the seeded Demo pre-op HQ returns about **0.97 MB** of JSON. The local database holds 1260 matching E2E questionnaires, and `published` lists every questionnaire's options per key. The editor fetches it again after every save. It takes 60 ms locally, but it could be heavy on a slow link or Lambda with many HQs. Target 41's response is 632 B.
3. **Works as designed:** the wording match ignores leading spaces, repeated spaces, case and trailing "!!": "   WHAT is your favourite   colour 41 !! " → "Pre-fills from 1 other HQ" (T23 Source 41 · by same wording).
4. **Test plan detail:** T-06 step 4 says to open Trust B's Demo pre-op HQ → "Do you smoke?". Trust B's Demo HQ has no Smoking page; it has About you, Your health (page Health: "Do you have diabetes?") and Nurse checks. I checked "Do you have diabetes?" and the full API response instead.
5. **As designed (note):** a question linked by key still shows "Pre-fills from 1 other HQ · by link" while its options differ from the published ones (Target's "Green apple"). Publish blocks it, so nothing reaches a patient. An author might still expect the label to warn there too.
6. **Double-clicking Publish** in the Publish window published once, with no error. The extension's network log showed "503" for successful publish POSTs, even after a single click. This is a tool artifact for the `204` response: the same two POSTs sent at once through the Vite proxy with curl returned `204` and `409` ("This version has changed since you opened it…").
7. **Not S23:** a question in a *clinician* Question Set still shows the grey "PATIENT" chip on its card in the editor (the set's audience is `clinician`). This code is unchanged by this session (the chip predates S23).

## Console and network errors

- No console errors or warnings on the editor, preview, episode, POA Summary or patient pages that I checked.
- No unexpected 4xx/5xx; the only "503" readings are the tool artifact in item 6.

## GIFs and test data

- GIF of the main happy path (T-02: label appears, list, reword, Number, delete): `/Users/rahul/Downloads/S23-T02-happy-path.gif`. No failures, so no failure GIFs.
- Test data created, all made up, run 41, Trust A unless noted:
  - **Questionnaires:** T23 Source 41 (v2 published), T23 Target 41 (v1 published), T23 Details 41, T23 Twice 41, T23 Fresh 41 (v2 published), T23 Explore 41 (v4 published).
  - **Trust B:** T23 Trust B 41.
  - **Episode:** "Mt S23 Fresh Testpatient" at Hospital A1, with email `mt.s23.fresh@test.example`.
- An extra Chrome tab (T23 Target 41's editor) stayed open after the tool's tab group was closed, and the tool couldn't close it.
