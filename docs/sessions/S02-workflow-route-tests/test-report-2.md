# S02 test report 2 (retest)

**17 passed, 0 failed, 0 blocked** (17 cases: T-01 to T-13, R-01 to R-04). Report 1's F-1 ("Patient answered: false") is **fixed**. A Yes / No correction now reads "No" and "Yes" on the validation screen and on the POA Summary.

**One Major problem found while exploring, older than S02 (X-1):** when two tabs have the same Question Set open, a stale tab's **Validate** silently overwrites an answer the other tab has just saved. Details are under "Found while exploring".

## Environment

- **Commit:** `7e6e535`. The working tree is **not clean**: the S02 changes are still uncommitted, the same files as in report 1.
- **When:** Thu 8 Oct 2026, 16:19–16:45 BST.
- **URL:** http://localhost:3000 (API on :8080; `/api/health` returned `{"status":"ok"}`).
- **Browser:** Chrome, driven by Claude in Chrome.
- **Test data created:**
  - HQ **"S02 test HQ 16:20"** (published, v1), id `60deed5b-796f-4456-88a5-b5e4174320c9`.
  - An episode for **Jo Bloggs**: `/h/…0a/episodes/fedd92d7-53ce-495c-aee3-a156387bfb75`. It ended Ready for admission.
  - Patient **S02 Test1622** and their episode `88077f36-b995-434d-82d4-3a50a79e936b`. After exploring, it's Ready for review, with About you and Your health validated.
  - The draft "S02 regression 1636" was created and then deleted (R-02).

### Where my testing differed from the plan

- **Retest scope:** the HQ script changed, so I re-ran the whole plan from T-01 on a new HQ. I didn't re-run only the failed cases.
- **Phone size (T-04 to T-06):** resizing the window didn't take effect for the first tab, which stayed 2560 px wide. A later tab ended at the window's minimum width of 500 px.
  - So I used the plan's fallback: a **390 × 844 same-origin iframe** (inner width 390, scroll width 390).
  - Touch, the address bar and the on-screen keyboard weren't tested.
- **"New private window" (T-04, T-05):** I signed all staff users out and opened the link in a new tab. For T-05, I closed that tab and opened the link in another new tab.
  - **I didn't clear any site storage.**
- **Console (T-01, T-13):** I ran the script and the fetch calls with the page's JavaScript tool, not by typing them into DevTools. The code was the same.
- **Checks through the API**, each confirming something I'd already seen in the UI:
  - `GET /api/p/{token}` (T-04);
  - `GET …/patients`, to check that no stray patient was made (T-03);
  - `GET /api/me`, to confirm who was signed in (sign-out check and R-04).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Make the test HQ | Pass | Printed `Created and published: S02 test HQ 16:20`. Listed as Published v1. It opens read-only ("v1 · Published") with About you, Your health and Nurse checks. |
| T-02 Episode for an existing patient | Pass | Only published versions are offered (S01 publish 14:25 v1, S02 test HQ 15:51 v1 and 16:20 v1). No drafts. Every field, the link and Copy ("Copied") work, the status dropdown is disabled, and the note "Episode created" (Cara Clinician, 16:21) appears. |
| T-03 New patient, with refusals | Pass | A future date of birth gives "Enter a valid date of birth." and the drawer stays open. Once corrected, the episode opens for S02 Test1622. The filter Ready for review shows neither episode. The refused try made no stray patient (checked with the API). |
| T-04 Patient, part-filled, at phone size | Pass | The page says "Hello Jo". Only About you and Your health are listed, and nothing scrolls sideways. Angina now shows as **Yes / No radio buttons** (the editor's shape). There's no "STAFF" or "Current smoker" anywhere, even in the hidden page code. The patient API sends no `clinicianOnly`, `clinicalOutputs`, `testCases`, Nurse checks, `q_ecg`, `q_advice` or `q_bp`. |
| T-05 Leave, come back, submit | Pass | In a new tab, About you is still Complete, with Yes and No kept. Send is enabled at 2 of 2. The "Send your answers?" dialog leads to "Thank you, Jo", which stays after a reload. There's no STAFF or Current smoker. |
| T-06 Bad patient link | Pass | A changed token and `/p/abc` both show "This link doesn't work. Please check it, or contact the hospital." |
| T-07 Clinician validates, with a correction | **Pass (was Fail)** | Status Ready for review. Complete HQ review is disabled, with the hint. The answers are pre-filled with Yes and No. The STAFF ONLY questions and page are shown, and "Current smoker" is in the Lifestyle summary. Changing angina to Yes shows **"Patient answered: No"**. All three sets end "Validated by Cara Clinician on 08/10/2026, 16:31", and Complete HQ review becomes enabled. |
| T-08 Complete the review; then read-only | Pass | The status becomes Ready for POA. About you is read-only, says "The HQ review is complete, so these answers can't be changed.", and has no Validate. "Patient answered: No" is still shown. Reload, Back and Forward all work. |
| T-09 POA Summary, both tabs, notes | **Pass (was Fail)** | The details are right. The three notes appear with their times. About you shows Current smoker and **Do you have angina? Patient answered "No", now "Yes"**. The Patient answers tab shows only "Current smoker". Notes one and two stay in order after a reload, and spaces alone can't be saved. "Nurse ASA grade set to 2", "Status changed to Ready for admission" and Print are all there. |
| T-10 Status locked before review | Pass | The dropdown is truly `disabled` on the episode page and the POA Summary. The orange "The HQ review isn't complete yet. This shows the answers as they stand." note shows. |
| T-11 Wrong role | Pass | Alex has no Episodes link. Alex and Val both get "Only clinicians can see patients and episodes." (the API returns 403, not 404). |
| T-12 Another hospital | Pass | With the Hospital A address, Bea sees "You don't have access to this hospital." With Hospital B's id, she sees "Only clinicians can see patients and episodes." No patient details are shown either way. |
| T-13 Null field refused (S02 fix) | Pass | As Cara, `{"status": null}` and `{"procedure": null}` → `400 {"error":"The request could not be read."}`. After a reload, Ready for admission and Knee replacement are unchanged. |
| R-01 Sign in and out | Pass, with a note | All four users land on their start page with their name in the header. One sign-out didn't take (see "Found while exploring", item 1). |
| R-02 Create and save a questionnaire | Pass | "S02 regression 1636" got a set, a page and a Yes / No question. "All changes saved" showed, and everything was kept after a reload. Deleted with the in-app confirmation. |
| R-03 A viewer can't edit | Pass | Val has no New Questionnaire. A published HQ and a draft (S01 publish 14:25) both open Read only, with no Publish, Create new version or Add controls. |
| R-04 Hospital scope for questionnaires | Pass | As Bea (confirmed with `/api/me`), the Hospital A address shows "You don't have access to this hospital." With her own hospital id, she sees "Questionnaire not found." |

## Failures in detail

None.

## Found while exploring

1. **Sign out didn't sign out once (again unconfirmed; already in pending.md §3).**
   - Steps:
     1. As Bea, at the end of T-12, I was on the refusal page `/h/…0b/episodes/fedd92d7…`.
     2. I clicked **Sign out**, waited about 1 s, clicked **Cara Clinician**, and then opened the episode address.
   - The page showed **Bea Author** and "You don't have access to this hospital."
   - My first T-13 fetches therefore ran as Bea. They got `404 {"error":"Hospital not found."}` and changed nothing.
   - I repeated the same steps twice more, recording the GIF and the network log. Both times it worked: `POST /api/logout` was sent, `/api/me` then returned 401, and Cara signed in.
   - This matches report 1's item 3. Both times it happened right after Sign out on a refusal page reached by pasting an address. I couldn't capture the failing try, so there's no GIF.
2. **A patient section counts as complete without "Finish this section"** (Minor, possibly by design).
   - Steps:
     1. On S02 Test1622's patient link, open About you and answer both questions.
     2. Don't click Finish. Reload instead.
   - The list then shows About you as **Complete**, with "1 of 2 sections complete".
   - So "complete" seems to mean "every required question answered", not "finished". If that's intended, it's fine. The T-04 wording suggests finishing is what marks it done.
3. **The "Send your answers?" dialog stayed open after the page behind it changed** (Cosmetic, timing).
   - My double-click on the dialog's button landed just below it as the dialog animated in. That closed the dialog, and the click opened About you.
   - The dialog then reappeared over the About you page, and confirming it there submitted normally.
   - A real user is unlikely to hit this.
4. **The staff header wraps at narrow widths.** At a 500 px window, "Cara Clinician / Sign out" drops to a second line and crowds the "← S02 Test1622" back link. Staff pages aren't phone-targeted, so this is cosmetic.

Things I tried that worked fine:
- **Required questions:** "Finish this section" with nothing answered shows "Response required." on both questions.
- **Reloading in the middle of a section:** the answers are kept.
- **Double-clicking:**
  - "Finish this section": one transition;
  - the dialog's **Send my answers**: only **one** `POST …/submit`, then the thank-you page;
  - **Validate**: one validation stamp, and no duplicate note.
- **Corrections, both ways** (S02 Test1622):
  - Smoking No → Yes shows "Patient answered: No", and "Current smoker" appears.
  - Angina Yes → No shows **"Patient answered: Yes"**.
  - Changing smoking back to the patient's answer removes its note, and "Current smoker" goes away.
- **Emoji and leading spaces** in a clinician text field (`   ECG ok 🫀 made up`) are kept exactly as typed.
- **Opening the patient link while a staff user is signed in:** the patient view is the same, with no STAFF content.

### X-1 (Major, older than S02): a stale tab's Validate silently overwrites a newer answer

Tested later the same evening (21:52–21:56), once the browser extension reconnected.

- **Signed in as:** Cara Clinician, Hospital A, in two tabs.
- **URL (both tabs):** `/h/…0a/episodes/88077f36-b995-434d-82d4-3a50a79e936b/sets/2b73cddb-870b-4377-8dba-2f633abbf368` ("Your health" for S02 Test1622).
- **Steps:**
  1. Open the set in tab A and in tab B. Both show diabetes **Yes**.
  2. In tab A, change it to **No**. The autosave `PUT …/answers/2b73cddb…` succeeds, and the page says "Saved".
     - Through the API, the clinician copy is now `o_dian` (No), unvalidated, updated at 21:55:08.
  3. In tab B, which still shows **Yes**, click **Validate**.
- **Expected:** tab B is told that the answers changed elsewhere (a 409, like a stale revision on chapters), and nothing is overwritten.
- **Actual:**
  - Tab B's `PUT` is accepted. The clinician copy becomes `o_diay` (**Yes**), "Validated by Cara Clinician on 08/10/2026, 21:55" (21:55:35).
  - Tab A still shows **No** and "Saved", with no warning, so its change was lost without notice.
  - The POA Summary would show the Yes.
- **How often:** every time. It also happened at 21:52, the first time I tried it.
- **Why Major:** it's wrong clinical data, with nothing to show the clinician that anything went wrong. Two clinicians reviewing the same episode would hit the same thing.
- **Not an S02 regression:** S02 didn't change the answer routes. Unlike chapters, answer saves seem to have no stale-revision check.
- **GIF:** `S02-two-tabs-lost-update.gif` (tab B's Validate and the result).
- **Console and network:** no errors. The PUTs show as 503, which is the known 204 quirk.

**An aside on the tool:** during this check, Rahul rejected one batch of browser actions. Its first actions had already run, which is why the 21:52 stamp exists. I then repeated the steps one at a time, with the network log.

## Console and network errors not tied to a case

- **Console:** no error or warning messages were captured. The tool only records messages after its first call, so I reloaded the POA Summary and the T-10 pages to capture page load. Both were clean.
- **Network:** nothing unexpected.
  - The 4xx responses were all deliberate: 403 for the wrong role (T-11, T-12), Bea's 404 "Hospital not found." (T-13's first try), and T-13's 400s.
  - The 503s were on `logout`, `dev/login`, `submit` and the answer autosave and validate PUTs, which is the known 204 quirk.
  - Every GET is sent twice, which is normal for React in dev mode.
- **Patient set page:** the "Saved" label still shows before any answer is given. This is already recorded.

## GIFs

In `~/Downloads/`:
- `S02-two-tabs-lost-update.gif` (114 KB): X-1, stale tab B's Validate overwriting tab A's change.
- `S02-retest-happy-path.gif` (3.6 MB, 50 frames): signing in as Cara, then T-07 to T-09. It shows the validation with **"Patient answered: No"**, completing the review, and the POA Summary with **Patient answered "No", now "Yes"**.

No plan case failed. The sign-out issue (item 1) didn't recur on the recorded retries.
