# S19 test report 1

Manual tester, automatic run (`/run-sessions`).

**Summary: 22 passed, 1 failed, 0 blocked** (18 plan cases + 5 regression checks), plus one hand check for Rahul.

**Environment:**
- HEAD `b534af3`, with S19 uncommitted in the working tree, as expected.
- 20 Oct 2026, 16:02–19:12 CEST. The run was paused twice by network outages; nothing was lost.
- http://localhost:3000, with `make dev` run by the conductor.
- Chrome, Rahul's trk profile.
- Desktop: viewport 1126×819 at first, then 1400×683.
- Phone: the narrowest Chrome window allowed is 500 px wide, so 390 px couldn't be set (see the hand check).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Shows Trust A's 4 seeded procedures with the right code, specialty, "Offered at" and default HQ. It also shows 3 procedures migration 00020 carried across from older free-text episodes: Hip replacement, Knee replacement and Knee replacement (left), all for every hospital with the hospital's default HQ. The retired E2E ones are last. No Trust B procedures. |
| T-02 | Pass | "Test hip revision is added."; the row shows HIP-02, Orthopaedics, Every hospital, Demo pre-op HQ. |
| T-03 | Pass | Test grommets shows —, —, Hospital A1, The hospital's default. |
| T-04 | Pass | "Test hip revision is saved."; HIP-03 and Trauma and orthopaedics are still there after a reload. Cancel changed nothing. |
| T-05 | Pass | The Retire and Bring back messages are exact. The retired row moves to the bottom, marked RETIRED. Retired again afterwards. |
| T-06 | Pass | Duplicate names are refused, whatever the case and including a retired one: the API returns 409 and the page shows the exact message. An empty name and a spaces-only name both get "Enter the procedure's name." Renaming Test grommets to "Knee Arthroscopy" is refused and it keeps its name. The Default HQ dropdown lists exactly Trust A's 1,243 published questionnaires, checked against the database: no draft-only ones (e.g. "S08 Ada draft 2300") and nothing of Trust B's. |
| T-07 | Pass | Ada has Procedures and added Test tonsillectomy. Cara and Sue have no link. Sue's deep link shows "Only super clinicians and admins of the whole trust can manage its procedures" and no list. |
| T-08 | Pass | Bea sees only Trust B's procedures: Total hip replacement, Inguinal hernia repair, and Trust B's own carried-across Hip replacement and Knee replacement. Pasting Trust A's address shows "You don't have access to this hospital". |
| T-09 | Pass | The list is right: names with the code in brackets, and no retired ones. Picking Total hip replacement chose Demo pre-op HQ v1. A hand-picked E2E HQ was kept. The episode page shows Total hip replacement, 03/11/2026 and 28/10/2026. |
| T-10 | Pass | With Hospital A1's default HQ set, the page opens with it chosen. Total hip replacement → Demo pre-op HQ v1. Laparoscopic cholecystectomy → back to the hospital's default. Clearing the procedure with × left the HQ unchanged. Cleaned up: Hospital A1 follows the trust's settings again (`settings = {}`). |
| T-11 | Pass | Val (A2): "No procedure of that name" for Cataract and for grommets. The other 7 are offered: the 3 seeded trust-wide procedures, the 3 carried across, and Test tonsillectomy. Before grommets was retired, the API's offered list for A2 also had no Cataract or grommets. |
| T-12 | Pass | After the retire, grommets isn't offered. The episode still shows Test grommets on its page and in the list. The edit form shows "Test grommets (no longer offered)". Changing only the POA date to 05/11/2026 saved and kept the procedure. |
| T-13 | **Fail** | A partly typed date is silently dropped and the episode is still created. See F1. |
| T-14 | Pass | Shows Knee arthroscopy, 10/11/2026 and — for the POA date. The 3 General notes appear, each by Cara Clinician. Cancel changed nothing. Still the same after a reload. |
| T-15 | Pass | The list has Procedure, Planned date and POA date columns in dd/mm/yyyy, or —. The POA Summary header shows Procedure, Planned procedure date and POA appointment date (Cataract surgery, 12/11/2026, 30/10/2026) on both tabs. |
| T-16 | Pass | After a complete review, "Edit procedure and dates" is gone, the values still show, and the status select is enabled. A PATCH sent directly from the browser console (an extra check) was refused with 409 and "The procedure and its dates can't be changed once the HQ review is complete." |
| T-17 | Pass (at 500 px) | The patient's home cards show "Procedure: Cataract surgery", "Procedure: Test grommets" and "Procedure: Knee arthroscopy". No planned or POA dates are shown. No sideways scroll at 500 px (`scrollWidth` = 500). 390 px is a hand check. |
| T-18 | Pass | Reload, Back and Forward each render the right page. Switching from A1 to A2 on Procedures stays on Procedures with the same 22-row list. |
| R-1 | Pass | Existing patient: search, **Choose**, no procedure and no dates. Created, and the details show — for all three. |
| R-2 | Pass | The patient filled in and sent Demo pre-op HQ, and the episode went to Ready for review. Cara validated all 3 sets and completed the review, and it went to Ready for POA. The POA Summary has both tabs. |
| R-3 | Pass | Ada saved Hospital A1's own reminder days, "3, 6", and they showed after a reload. Put back to the trust's setting afterwards. |
| R-4 | Pass | The audit log has "Added a procedure" and "Changed a procedure" rows by Alex (DONE, plus REFUSED (409) and (400) for the T-06 attempts) and Ada's add. Episode actions show as before. |
| R-5 | Pass | Staff, Settings and Audit appear only for Ada. Everyone has Questionnaires and Episodes. |

## Failures

### F1 (Minor): a partly typed date is silently dropped and the episode is still created (T-13)

- **Who and where:** Cara Clinician, Hospital A1, `/o/6ef150fb…/h/…000a/episodes/new`.
- **Steps:**
  1. Search `ProcedureMT1603` and click **Choose**.
  2. Set Health questionnaire to **Demo pre-op HQ v1**.
  3. In **POA appointment date**, type only the day and month, so the field shows `28/10/yyyy`.
  4. Click **Create episode**.
- **Expected:** Create says "Enter the POA appointment date as a date." (or the browser refuses it), and no episode is created.
- **Actual:**
  - POST `…/episodes` returns 201. The episode is created with POA appointment date "—", and there is no message.
  - The browser marks the field `validity.badInput = true` with `value = ""`, so the page treats it as empty.
  - The same happens with the **Planned procedure date** typed as `31/mm/yyyy` (episode `d27aee2c…`, created with Planned date —).
  - **Edit procedure and dates** behaves the same: a partial planned date and **Save** closes the form with no message and no change.
- **Why it matters:** the clinician thinks they typed a date, and it's lost without a word. It does show as "—" afterwards.
- **Console and network:** no errors.
- **GIF:** `S19-T13.gif`.

## Hand checks for Rahul

- **T-17 at 390×844:** the narrowest Chrome window allowed is 500 px wide, so I tested at 500×667. Please check the patient home at 390 px in device emulation.

## Found while exploring

1. **Edit on a lower row opens the form out of sight (Minor).** On Procedures, Edit on a row below the fold opens the form at the top of the page and doesn't scroll to it. All you see is every row's buttons turning grey.
2. **Enter doesn't submit the Add/Edit procedure form (Minor, keyboard).** Typing a name and pressing Enter does nothing; you have to click **Add procedure**.
3. **The header overflows for Ada at 1126 px (Minor).**
   - Sign out sits at x≈1175–1254 in a 1126 px viewport, off-screen. The page doesn't scroll sideways, so Sign out can't be reached.
   - The new Procedures link adds about 114 px. Even without it, Sign out would end at about 1140 px, so this is partly older.
   - It fits at 1280 px and above.
4. **The Health questionnaire select on New episode isn't searchable.** Typing does nothing, which is hard with 1,243 options locally. This is older, not S19. The Default HQ select on Settings is searchable.
5. **Leading and trailing spaces are trimmed, and emoji are kept,** in procedure names ("MT S19 1910 Ünicode 🦴 knee"). Double-clicking **Add procedure** sent only one POST.
6. **The carried-across procedures** (migration 00020) show in both trusts' lists, with no code, no specialty and the hospital's default HQ. This works as the build notes describe. Mentioned so the seeded-list counts in T-01 and T-08 make sense.

## Console and network errors not tied to a case

- No console errors on the pages checked.
- One `PATCH …/procedures/8bfbe657… → 503` showed in the extension's network log around the second network outage. That procedure is Test grommets. The database's audit log has a single 204 for that retire, and the page showed the success message, so it looks like an artefact of the outage, not an app error.

## GIFs

Saved in `/Users/rahul/Downloads/`:

- `S19-T02-T06-procedures.gif`: the main happy path for managing procedures.
- `S19-T09-new-episode.gif`: a new episode with a procedure and dates.
- `S19-T13.gif`: failure F1.

## Test data created

- Patient Pat ProcedureMT1603, with 3 episodes at Hospital A1:
  - `778b76ae…`: Knee arthroscopy;
  - `d27aee2c…`: Test grommets;
  - `154f37c8…`: Cataract surgery, review complete.
- Procedures: Test hip revision (retired), Test grommets (retired), Test tonsillectomy, and "MT S19 1910 Ünicode 🦴 knee".
- Hospital A1's settings are back to `{}`.
