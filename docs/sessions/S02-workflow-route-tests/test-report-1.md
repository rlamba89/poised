# S02 test report 1

**15 passed, 2 failed, 0 blocked** (17 cases: T-01 to T-13, R-01 to R-04). Both failures have the same cause and are Minor.

## Environment

- **Commit:** `7e6e535`. The working tree is **not clean**: the S02 changes are uncommitted (episodes.go, middleware.go, router.go, the new `*_integration_test.go` files, docs and the session files).
- **When:** Thu 8 Oct 2026, 15:51–16:12 BST.
- **URL:** http://localhost:3000 (API on :8080; `/api/health` returned `{"status":"ok"}`).
- **Browser:** Chrome, driven by Claude in Chrome.
- **Test data created:**
  - HQ **"S02 test HQ 15:51"** (published, v1);
  - an episode for **Jo Bloggs**: `/h/…0a/episodes/6b880600-e6aa-4802-84e2-8a71f25363ea`;
  - patient **S02 Test1553** and their episode `c0e89d76-ce16-4d1c-b4c2-bc1b2b937d12`.
  - The draft "S02 regression 1609" was created and then deleted (R-02).

### How I tested that differs from the plan

- **Phone size (T-04 to T-06):** the extension couldn't resize the Chrome window. It reported success, but the page stayed 2560 px wide, even after Rahul un-maximised the window.
  - Instead, I loaded the patient link in a **390 × 844 same-origin iframe**. The page inside sees a real 390 px viewport, so its media queries and sideways scrolling behave as on a phone.
  - It is not a real phone-size window, though: touch, the address bar and the on-screen keyboard weren't tested.
- **"New private window" (T-04, T-05):** the extension can't open incognito. I signed all staff users out and opened the patient link in a fresh tab.
  - For T-05, I cleared local and session storage before reopening. **Side effect, my mistake:** `localhost:3000` storage also held keys from Rahul's other local apps (asgard, twindle, Clerk and Cognito/OIDC sign-in data). They are gone now. None of them belonged to Poised.
- **Console (T-01, T-13):** I ran the scripts with the page's JavaScript tool instead of typing them into DevTools. The code was the same; only the way it was entered differs.
- **Checks through the API:** I fetched `GET /api/p/{token}` (T-04), tried a second submit and a later answer change (T-05), and read one chapter's JSON (R-02). Each time I was confirming something I'd already seen in the UI.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Make the test HQ | Pass | Printed `Created and published: S02 test HQ 15:51`. Listed as Published v1. It opens Read only with About you, Your health and Nurse checks. |
| T-02 Episode for an existing patient | Pass | Only published versions are listed: S02 test HQ and "S01 publish 14:25 v1", whose v1 is published. The draft-only HJE Full HQ isn't listed. All fields, the link and Copy ("Copied") work, the status dropdown is disabled, and the "Episode created" note appears. |
| T-03 New patient, with refusals | Pass | A date of birth in the future gives "Enter a valid date of birth." and the drawer stays open. Once corrected, the episode opens. The filter Ready for review shows neither episode. No stray patient was created by the refused try. |
| T-04 Patient, part-filled, at phone size | Pass | Greeting "Hello Jo". Only About you and Your health are listed. No sideways scroll (scroll width 390). No "STAFF" or "Current smoker" anywhere, even in the hidden DOM. The patient API sends no `clinicianOnly`, `clinicalOutputs`, `testCases`, Nurse checks, `q_ecg` or `q_advice`. |
| T-05 Leave, come back, submit | Pass | About you is still Complete, with Yes and No kept. Send is enabled only once 2 of 2 sets are done. The in-page confirmation is followed by "Thank you, Jo", which stays after a reload. A second submit and a later answer change both get **409**. |
| T-06 Bad patient link | Pass | A changed token and `/p/abc` both show "This link doesn't work. Please check it, or contact the hospital." No questions are shown. |
| T-07 Clinician validates, with a correction | **Fail (Minor)** | Everything works except the correction note, which reads **"Patient answered: false"**, not "No". See F-1. |
| T-08 Complete the review; then read-only | Pass | Status Ready for POA. About you is read-only with the expected message and no Validate. Reload, Back and Forward all show correctly. |
| T-09 POA Summary, both tabs, notes | **Fail (Minor)** | The change reads **"Patient answered "false", now "true""**, where the plan expects No → Yes (F-1). Everything else passes: both tabs, both notes in order, spaces-only can't be saved, notes kept after reload, "Nurse ASA grade set to 2", "Status changed to Ready for admission", and Print. |
| T-10 Status locked before review | Pass | The dropdown is truly `disabled` on the episode page and the POA Summary. The orange "The HQ review isn't complete yet…" note shows. |
| T-11 Wrong role | Pass | Alex has no Episodes link. Alex and Val both get "Only clinicians can see patients and episodes." on the episode and POA addresses. |
| T-12 Another hospital | Pass, with a note | No patient details are shown. The Hospital A address says **"You don't have access to this hospital."**, not "not found" (see "Found while exploring", item 4). With Hospital B's id, it says "Only clinicians can see patients and episodes." |
| T-13 Null field refused (S02 fix) | Pass | `{"status": null}` and `{"procedure": null}` → `400 {"error":"The request could not be read."}`. I also tried `anaesthetic` and `consultant` null: the same 400. After a reload, the status (Ready for admission) and procedure (Knee replacement) are unchanged. |
| R-01 Sign in and out | Pass, with a note | All four users land on their start page with their name in the header. Sign out was intermittent (see "Found while exploring", item 3). |
| R-02 Create and save a questionnaire | Pass | Created "S02 regression 1609" with a set, a page and a Yes / No question. "All changes saved" shows, and everything is kept after a reload. Deleted with the in-app confirmation. |
| R-03 A viewer can't edit | Pass | Val has no New Questionnaire. A draft (HJE Full HQ) and a published HQ both open Read only, with no Publish or add controls. |
| R-04 Hospital scope for questionnaires | Pass, with a note | Bea, on the Hospital A address, gets "You don't have access to this hospital." With Bea's own hospital id, she gets "Questionnaire not found." Nothing from Hospital A is shown. |

## Failures in detail

### F-1 (Minor): a corrected Yes / No answer shows "false" and "true" instead of "No" and "Yes"

This causes both T-07 and T-09.

- **Signed in as:** Cara Clinician, Hospital A.
- **URLs:**
  - `/h/00000000-0000-4000-8000-00000000000a/episodes/6b880600-e6aa-4802-84e2-8a71f25363ea/sets/d6cdc3c5-bbcd-4d88-8330-71fd30c9b458`
  - …`/poa`
- **Steps:**
  1. Do T-01 to T-05. The patient answers **No** to "Do you have angina?".
  2. As Cara, open the episode, click **Validate** on **About you**, and change angina to **Yes**.
- **Expected:**
  - under the question: "Patient answered: No";
  - on the POA Summary: angina No → Yes.
- **Actual:**
  - under the question: **"Patient answered: false"**;
  - on the POA Summary (Validated summary, "Changed by the clinician on Lifestyle"): **Do you have angina? Patient answered "false", now "true"**.
  - It's the same on the read-only view after the review is complete.
- **Console and network:** no errors.
- **GIF:** `S02-T07.gif`.
- **Context, for the builder:**
  - The question is a SurveyJS `boolean` (made by `make-test-hq.js`), and the patient saw a No / Yes switch for it.
  - The editor's own **Yes / No** question is stored as a `radiogroup` with `yesNo: true` and Yes / No choices (I checked R-02's saved JSON). I found no `"boolean"` in the app code or the HQ importer.
  - So editor-made HQs probably don't hit this. Either the answer description needs to handle `boolean` (true/false → the labels, defaulting to Yes / No), or the test script should use the editor's Yes / No shape. That's the builder's call.

## Found while exploring

1. **Double-clicking "Save comment" saves the note twice (Minor).**
   - Steps: as Cara, on any episode, click Add comment, type text, and **double-click** Save comment.
   - Two identical notes appear, each from its own `POST …/notes` → 201. Reproduced twice, on episode `c0e89d76…`.
   - Notes are append-only, so the duplicate can't be removed, and it shows on the POA Summary.
   - Expected: the button is disabled while saving. GIF: `S02-double-click-note.gif`.
2. **The patient link's response includes more personal data than the page uses.**
   - `GET /api/p/{token}` returns `patient: {firstName, lastName, dateOfBirth, sex}`, but the patient pages only show the first name.
   - Not an S02 regression. But the target design signs patients in with **magic link + date of birth**. If this response still includes `dateOfBirth` then, anyone with the link could read the answer to the check. Worth a decision before plan C-series patient sign-in.
3. **Sign out sometimes doesn't sign you out (Minor, not reliably reproducible).**
   - 2 of my ~10 sign-outs left the old user signed in. Both happened after clicking **Sign out** on a refusal page reached by pasting an address: Alex on `…/poa`, then Bea on `…/h/…0b/episodes/…/poa`.
   - The login page appeared, but the next page still showed the old user (Alex, then Bea).
   - Retrying on the same pages worked, and `POST /api/logout` was sent each time I checked. It could be a timing issue on my side, so treat it as unconfirmed.
4. **Different wording for another hospital's address.**
   - The plan expects "not found" for T-12 step 2 and R-04. The app says "You don't have access to this hospital."
   - It's a refusal and leaks nothing about patients. It does confirm that the hospital id exists, which a "not found" wouldn't. Plan or app: builder's call.
5. **The questionnaire refusal pages have no header** (R-04). The page shows only the red message, with no header, Sign out or way back. The episode refusal pages do have the header.
6. **The hospital picker is blank** on the "You don't have access to this hospital" episode page (T-12 step 2). Cosmetic.
7. **"No episodes yet." shows when a filter matches nothing** (T-03 step 6), even though episodes exist. "No episodes match this filter" would be clearer. Cosmetic.
8. **An unknown status is a 409, not a 400.**
   - `PATCH …/episodes/{id}` with `{"status": "bogus"}` or `{"status": ""}` gets 409 "The status can't be changed to that now." A value that isn't a status at all is arguably a 400.
   - `{"procedure": 123}` and a body that isn't JSON both correctly get 400. `{}` gets 204 and adds no note. Very low priority.
9. **The patient's set page says "Saved" before any answer is given** (T-04 step 4). Cosmetic.
10. **The delete dialog's text flickers to `"" and all its Question Sets…`** as it closes (R-02). Cosmetic.
11. **The Nurse and Anaesthetist ASA grade controls work before the review is complete** (T-10's POA Summary), while Episode status is locked. Possibly by design; noting it.

Things I tried that worked fine:
- unknown or malformed episode ids ("Episode not found.");
- a set's address on an unsubmitted episode (read-only, "The patient hasn't sent their answers yet.");
- Validate with a required clinician question empty: refused with "Some required questions have no answer yet." and the field highlighted;
- emoji and leading spaces in a note: the emoji is kept and the leading spaces are trimmed;
- Back and Forward on the episode and set pages.

## Console and network errors not tied to a case

- **Console:** no messages were captured on either tab. That includes the 401s on the sign-in page that the plan expects, so the tool may not record network-error lines. I saw no red errors in the pages themselves.
- **Network:** nothing unexpected.
  - The 4xx responses were all deliberate: the T-13 probes, bad ids, Hospital B lookups, and the 409s on resubmitting.
  - The 503s are on `dev/login`, `logout`, the autosave PUTs, `complete-review` and `DELETE`, which is the known 204 quirk.
  - Every GET is sent twice, which is normal for React in dev mode.

## GIFs

All in `~/Downloads/`:
- `S02-happy-path.gif`: T-02 to T-07 (create the episode, the patient flow in the phone frame, and validation). 50 frames.
- `S02-T07.gif`: F-1, "Patient answered: false".
- `S02-double-click-note.gif`: the duplicate note from a double-click.
