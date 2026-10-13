# S13 test report 1: patient records

**Summary: 21 passed, 1 failed, 0 blocked** (17 cases + 5 regression checks).

- **Environment:** HEAD `1361850`, working tree **not clean** (S13 is uncommitted, plus planning-doc changes). Tested 13 Oct 2026, 10:48–11:20 BST, at http://localhost:3000 (Cara, Ada) and http://tester.localhost:3000 (Val, Max, Bea, Ada, Alex). The conductor's `make dev` had freshly migrated and seeded the database.
- **Data:** round 1 NHS numbers (N1 `9990002118`, N2 `9990002126`, N3 `9990002134`), surnames ending ` R1`.
- **Browser:** Claude in Chrome. Each case was run through the UI. To save time, some form fields were filled with a small script that sets values through React's input events and picks Mantine options by clicking them. Every *Create* that a case depends on was clicked for real.
- **API calls:** I also made direct API calls (`fetch` in the signed-in tab) to probe for privacy leaks, as the conductor asked. Each place this was done is marked **[API]**.
- **Database:** read-only `psql` queries were used to choose search terms and to confirm the records at the end.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 New episode page | Pass | A full page at `…/episodes/new` with the three cards and no tabs. Reload, Back, Forward, **← Episodes** and **Cancel** all work. |
| T-02 Valid NHS number | Pass | The search said "No patients at this hospital match…". The card showed A1-N1, 9990002118, +447700900101 and nora.valid@example.test. |
| T-03 Wrong NHS number | Pass | The exact red message for a wrong check digit and for 9 digits. The form kept its values, and no "Wrong R1" episode or record was created (checked in the DB). |
| T-04 Other details checked | **Fail** | The mobile, email and first-name messages are correct. **With no questionnaire chosen, the message is "The request could not be read."**, not "Choose a published HQ." See F-1. |
| T-05 Search | Pass | `Va` sends no request. `Val`, `Valid R1`, `999 000 2118` and `a1-n1` all find Nora's row (name, date of birth, NHS number, A1-N1, Choose; no contacts). `a1-n` finds nothing. `Va` plus the date of birth finds her. Choose shows the card and fills A1-N1; Change patient brings the form back. The "first 20" line is correct, but the plan's `Sam` hint doesn't work (see notes). |
| T-06 Other hospital's patient not found | Pass | Searches for N2 (with and without spaces), `A2-SECRET`, `a2-secret`, `Elsewhere R1`, `Els`+DOB and `E`+DOB all say "No patients at this hospital match." **[API]** The bodies are `{"patients":[]}`. Greta's record id at A1 gives 404 "Patient not found.", and so does A2's episode id at A1. Creating with `patientId` = Greta gives the same 400 as a random UUID, so it isn't an oracle. |
| T-07 Silent link | Pass | Created with no warning. The page shows A1-G and the trust's record (Greta, +447700900202, greta@example.test). It never shows A2-SECRET, "A2" or A2's episode, in the UI **or [API]** in the episode, patient and episode-list JSON. Both patient pages show the same record id `176609b3…`. Each hospital lists only its own episode. A1's search for N2 now finds Greta with A1-G. |
| T-08 Mismatch | Pass | "The details don't match" with the exact text, for both the date of birth and the surname. **[API]** The 409 body is `{code:"mismatch", error}` only, with no stored values. Nothing was created. |
| T-09 One record per trust | Pass | Val's second Greta episode went to the same record, which now lists two A2 episodes. Max at B1: the search for N2 found nothing, and creating Bert with N2 made a new Trust B record. Nothing about Trust A appeared. **[API]** Max asking for Greta through B's org, mixed org/hospital ids, or A2 under org B all get 404. |
| T-10 Possible duplicate elsewhere | Pass | The orange box with the exact text. **[API]** `matches: []`. No stored mobile or email, no hospital and no "Use this record" button. Create anyway made a new record (`50f08503…` ≠ Val's `d558e12f…`) with only Cara's mobile. |
| T-11 Link by mobile or email | Pass | `+44 7700 900505` and `MO.CONTACT@example.test` both linked silently to Val's record `db1f58ad…` and show Val's email. Cara's patient page lists 2 A1 episodes. Nothing about A2 is shown. |
| T-12 Match at this hospital | Pass | The box shows Ola (born 9 Sept 1999, NHS —, A1-OLA) with **Use this record** and **Create anyway**. Use this record → card plus A1-OLA → created. Ola's page lists 2 episodes. |
| T-13 Patient page | Pass | Name, NHS number, date of birth with age, sex, mobile and email, then "Episodes at Hospital A1". A row opens its episode, and Back and reload work. **New episode** opens `?patient=` with the patient's card. Pat (Val's record) at A1 shows "Patient not found."; so does `?patient=<Pat>` on New episode. Greta at A1 lists only A1-G. |
| T-14 Hospital number on the episode | Pass | A1-N1 appears under the name in the list, on the episode card, and on the POA Summary header and table. The POA Summary also shows the NHS number and mobile. |
| T-15 Other hospital / trust refused | Pass | Cara at the A2 New episode page and at A2's patient page sees "You don't have access to this hospital". Bea at B1 finds nothing for N1, `999 000 2118`, `Valid R1`, `A1-N1` or `Elsewhere R1`. **[API]** Bea creating N1 with another date of birth, or Pat Nonhs R1 with no NHS number, in Trust B gets 201 with no mismatch or duplicate warning, so Trust A's records don't affect Trust B. |
| T-16 Audit | Pass | Who = Cara shows "Searched patients", "Opened a patient" and "Created an episode" rows, with patients named. Refused rows show "—". Patient = `Valid R1`, or `999 000 2118` (in the URL), finds Nora's rows, including 6 "Searched patients". **[API]** Cara gets 403 on the audit API. |
| T-17 Patient link at phone size | Pass* | "Hello Nora.", no sideways scroll (scrollWidth 386 = width). Both sections were filled in and sent ("Thank you, Nora"). *The Chrome window couldn't be resized (it stays at 2560 px), so the patient link was loaded in a **390×844 same-origin iframe**. This is a close stand-in, but not a real device or the device toolbar. |
| R-01 Episodes list, filter, All hospitals | Pass | The Ready for POA filter shows only that status. Ada's All hospitals has a Hospital column (Greta: A2-AGAIN/A2, A1-G/A1, A2-SECRET/A2). |
| R-02 Validate, complete, POA | Pass | Validated all 3 Question Sets on Nora's episode, then Complete HQ review → Ready for POA. The POA Summary opens. |
| R-03 Staff and Settings | Pass | Both open for Ada. |
| R-04 Export CSV | Pass | **[API]** The link's URL returns 200 `text/csv` (2,350 lines, including "Searched patients" rows). I didn't trigger a real browser download. |
| R-05 Editor and Preview | Pass | Alex opened Trust A demo HQ (read-only v1) and the About you Preview. |

## Failures

### F-1 (Minor): "No questionnaire chosen" says "The request could not be read."

- **Signed in as:** Cara Clinician, Hospital A1.
- **URL:** `http://localhost:3000/o/6ef150fb-…/h/00000000-0000-4000-8000-00000000000a/episodes/new`
- **Steps:**
  1. Open New episode.
  2. Type First name `Wendy`, Surname `Checks R1`, date of birth `01/01/1980`, Sex Female.
  3. Leave **Health questionnaire** empty.
  4. Click **Create episode**.
- **Expected:** "Choose a published HQ." (test plan T-04.3).
- **Actual:** the red box says **"The request could not be read."**
- **Cause, from the network:** the page sends `"versionId":""`, and the API answers `400 {"error":"The request could not be read."}`. The empty string seems to fail JSON decoding into a UUID before the "choose an HQ" check runs. This is the same message a malformed request gets, so the user can't tell what to fix.
- **Console:** no console errors.
- **GIF:** `~/Downloads/S13-T04.gif`.

## Found while exploring

1. **Change patient keeps the old patient's hospital number (Minor, S13 scope).** After **Choose** fills Hospital number with `A1-N1`, clicking **Change patient** brings back the search and the New patient form, but the Episode card's **Hospital number still says A1-N1**. A clinician who then types a different new patient would create that episode under Nora's hospital number unless they notice. Suggest clearing it, or only keeping it if the user typed it.
2. **`?patient=` from the patient page doesn't fill the hospital number.**
   - The patient-page **New episode** card reads "Born 9 September 1999 · NHS number —", with no "· Hospital number A1-OLA".
   - The Hospital number field stays empty.
   - Choosing the same patient from the search shows the number and fills the field.
   - This is inconsistent, but not wrong. Cosmetic/Minor.
3. **Design note on privacy (expected per S08/answers, so not a failure).** The rules confirm that a record exists, and a successful link reveals it:
   - The NHS-number **mismatch** reply tells a clinician the trust holds that NHS number.
   - A **silent link** then reveals the whole trust record (first name, mobile, email).
   - So a clinician who knows an NHS number and surname could try dates of birth until one links, and then see the record. Each try leaves an audit row ("Created an episode" REFUSED (409)), and a success creates a real episode.
   - Likewise, name + date of birth + a guessed mobile links and reveals the stored email (T-11).
   - Rahul may want a rate limit or an audit alert on repeated mismatches later.
4. **Synthetic rapid clicks send several POSTs.**
   - Three `click()` calls in the same tick sent three POSTs. For a new NHS number the unique index caught it: one 201 and two 409 "A record with this NHS number was added just now. Try again.".
   - A **real double-click** sent only one POST and created one episode, so users are fine.
   - Without an NHS number there's no unique index. Disabling the button synchronously would remove the gap. Low risk.
5. **The test plan's T-05 hint doesn't work.** "`Sam` + date of birth `1980-05-17`" finds nothing, because the automated runs' "Sam …" patients have **Sam as their first name**, and the search matches surnames. The "first 20" line was checked with `J` + `1980-05-17` and with `Aud` (both returned 20 rows and the line). This is a test-plan fix, not a product bug.
6. **The "first 20" line still says "or add the date of birth"** when a date of birth is already set. Cosmetic.
7. **The Audit screen's Patient box placeholder** says "Name or hospital number", but the NHS number works too. Cosmetic.
8. **The patient page doesn't highlight "Episodes"** in the header nav (the episode pages do). The "Patient not found." page has no **← Episodes** link. Cosmetic.
9. **A non-UK mobile is accepted** (`+1 202 555 0143` → 201). Per the build notes this is deliberate until S14 (PAT-11). Mentioned for completeness.
10. **Other edge inputs behaved well [API]:**
    - These were refused with clear messages: a name over 100 characters, a future date, 31 Feb, 1800-01-01, a letter O in the NHS number, a 51-character hospital number, an unknown sex, and a name of only spaces.
    - Emoji were accepted.
    - Padding on the name and NHS number (`"  elsewhere   r1 "`, `" 999 000 2126 "`) still linked to Greta.
    - Searches for `%%%`, `___` and `%`+DOB return nothing; a 300-character search → "The search is too long."

## Console and network errors

- No console errors or warnings on the New episode page, patient page, episode page, POA Summary or Audit screen.
- The only unexpected non-2xx response was F-1's 400. The others were the 400s, 404s and 409s that the cases and probes deliberately caused.

## Hand checks for Rahul

- **T-17 on a real phone-size window:** if you want certainty beyond the 390 px iframe, check it in Chrome's device toolbar or on WebKit.
- **R-04:** that clicking **Export CSV** actually saves a file. The endpoint itself returned a valid CSV.

## GIFs

- `~/Downloads/S13-T02-happy-path.gif`: T-02 happy path, typed for real.
- `~/Downloads/S13-T04.gif`: F-1.

STATUS: FAILURES 1
