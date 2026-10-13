# S13 test report 2: patient records (retest)

**Summary: 5 passed, 0 failed, 0 blocked** (T-02, T-04, T-05, T-07, T-13), plus a privacy re-probe, which found nothing.

- **Environment:** HEAD `1361850`. The working tree is **not clean**: S13 and its fixes are uncommitted, plus planning-doc changes. Tested on 13 Oct 2026, 11:19–11:35 BST.
- **URLs:** http://localhost:3000 as Cara Clinician (Hospital A1), and http://tester.localhost:3000 as Val Clinician (Hospital A2). The conductor's `make dev` was running.
- **Data:** round 2 NHS numbers (N1 `9990003106`, N2 `9990003114`) and surnames ending ` R2`.
- **Browser:** Claude in Chrome.
  - Searches, Choose, Change patient, New episode, the patient page and every **Create episode** were clicked or typed for real.
  - To save time, some text fields were filled with a small script that sets values through React's input events. Sex and the questionnaire were always picked by clicking.
- **[API]** marks direct `fetch` calls made in the signed-in tab for the privacy probes.
- **[DB]:** one read-only `psql` query confirmed the records at the end.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 Valid NHS number (quick) | Pass | Search `999 000 3106` showed "No patients at this hospital match. Enter their details below." The new episode's card shows Hospital number A1-N1, NHS number 9990003106, 4 March 1985 (age 41), +447700900101 and nora.valid@example.test. |
| T-04 Other details checked | Pass | Mobile `12345` → "Enter a valid mobile number, or leave it empty." Email `nora@` → "Enter a valid email address, or leave it empty." **No questionnaire → "Choose a published HQ."** (F-1 is fixed: the POST now gets `400 {"error":"Choose a published HQ."}`). No first name → "Enter the patient's first name and surname." **[DB]** No "Checks R2" record or episode was created. |
| T-05 Search | Pass | `Va` sends no request and shows nothing. `Val`, `Valid R2`, `999 000 3106` and `a1-n1` find Nora (name, date of birth, NHS number, A1-N1, Choose; no contact details). `a1-n` finds nothing. `Va` + 1985-03-04 finds her. Choose shows the card and fills A1-N1. **Change patient now empties the Hospital number.** `Aud` → 20 rows and "Showing the first 20. Type more, or add the date of birth, to narrow the search." With 1980-05-17 set, `Aud` and `J` each give 20 rows and only "Showing the first 20. Type more to narrow the search." The new hint works, and round 1's cosmetic note 6 is fixed. |
| T-07 Silent link (quick) | Pass | Val made Greta Elsewhere R2 (N2, A2-SECRET) at A2. Cara's searches for N2, `A2-SECRET`, `Elsewhere R2` and `Nonhs R2` beforehand all said "No patients at this hospital match." Cara then typed `G` / `elsewhere r2` / N2 / 07700 900999 / A1-G: the episode was **created with no warning** and shows the trust's record (Greta, +447700900202, greta@example.test) with A1-G. Both patient pages have the same record id `54c93dcb…`. Cara's lists "Episodes at Hospital A1" with only A1-G, and Val's lists only A2-SECRET. **[API]** Cara's episode, patient and Episodes-list JSON contain no `A2-SECRET` and no A2 hospital id, and Val's list has no A1-G. Cara's N2 search now returns Greta with `hospitalNumber: "A1-G"`. |
| T-13 Patient page | Pass | Clicking the name on Nora's episode opens `…/patients/06e5fb2d…`, which shows the NHS number, date of birth with age, sex, mobile and email, then "Episodes at Hospital A1" with one row. The row opens the episode, and Back and reload work. **New episode now shows the card with "· Hospital number A1-N1" and fills the Hospital number field with A1-N1** (round 1's note 2 is fixed). This survives a reload. Step 4: Pat Nonhs R2 (Val's record `67f9b8c9…`) at A1 shows "Patient not found.", and so does `episodes/new?patient=<Pat>` (red box). Greta's page at A1 lists only A1-G. |

### Privacy re-probe (as asked by the conductor)

All of these were done as Cara at Hospital A1, and found nothing:

- **Search [API]:**
  - `9990003114`, `A2-SECRET`, `a2-secret`, `Elsewhere R2` and `Nonhs R2` (before A1 had any episode for those patients) all return `{"more":false,"patients":[]}`.
  - `Els` + 1970-02-03 and `Non` + 1960-06-06 return only round 1's patients that A1 has its own episodes for, never the round 2 A2-only records.
- **Possible-duplicate warning:**
  - In the UI, `pat` / `NONHS R2` / 1960-06-06 / 07700 900404 shows the orange "Possible duplicate" box: "This patient may already be registered in the trust…". It names nobody and has no "Use this record" button.
  - **[API]** The 409 body is `{"code":"possible_duplicate","error":…,"matches":[]}`. No stored mobile, email or hospital is included.
- **Mismatch [API]:** N2 with the date of birth one year off returns `409 {"code":"mismatch","error":…}` only, with no stored values.
- **Patient page and episode [API]:** Pat's record id at A1 → `404 Patient not found.`, and Pat's A2 episode id at A1 → `404 Episode not found.`

## Failures

None.

## Found while exploring

1. **After Change patient, the address still says `?patient=<id>` (Minor).**
   - Steps:
     1. On Nora's patient page, click **New episode**.
     2. Click **Change patient**. The search and form come back, but the URL still ends in `?patient=06e5fb2d…`.
   - Creating a new patient from there is correct. I typed Ursula Urlcheck R2 and the episode went to Ursula, not Nora.
   - But a **reload**, or coming back to the page via Back/Forward, chooses Nora again.
   - Suggest dropping the `patient` query parameter when Change patient is clicked.
2. **Change patient also clears a hospital number the user typed themselves (Cosmetic).**
   - Steps:
     1. Type `MY-OWN` in Hospital number.
     2. Search, and **Choose** Nora. The field correctly keeps `MY-OWN`, and the card shows A1-N1.
     3. Click **Change patient**. The field is emptied, although the user typed `MY-OWN`, not Nora's record.
   - This is the safe side of the round 1 problem, so it's only a nit.
3. **Testing note, not a product bug:** in the second tab (Val), a click by coordinates on the patient's name heading only hovered it, and the page didn't change. Clicking the same link by its element reference worked. In Cara's tab, coordinate clicks on the heading always worked. This looks like a quirk of the extension with the tab in the background, not of the app.
4. **Still present from round 1 (cosmetic, unchanged):**
   - The patient page doesn't highlight **Episodes** in the header nav.
   - The "Patient not found." page has no **← Episodes** link.
   - The New episode search table has no "Patients found" heading. The plan's wording mentions one, but the table is clear without it.

## Console and network errors

- No console errors or warnings on the New episode page, the patient page or the episode page.
- The only non-2xx responses were the ones the cases and probes caused on purpose:
  - 400s for invalid mobile, email, questionnaire and name;
  - 409s for the possible duplicate and the mismatch;
  - 404s for Pat's patient page and episode at A1.

## Hand checks for Rahul

None new. Round 1's hand checks (T-17 at a real phone size, R-04's actual file download) weren't in this retest's scope.

## GIFs

- `~/Downloads/S13-T02-happy-path-r2.gif`: T-02 happy path, typed for real.
- `~/Downloads/S13-T04-r2.gif`: no questionnaire → "Choose a published HQ." (the F-1 fix).
- `~/Downloads/S13-T05-change-patient-r2.gif`: Choose fills A1-N1, and Change patient empties it.
- `~/Downloads/S13-T13-r2.gif`: patient page → row → Back → New episode with the card and A1-N1 filled in.

STATUS: ALL_PASSED
