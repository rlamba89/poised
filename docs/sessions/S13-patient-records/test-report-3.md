# S13 test report 3: patient records (retest, round 3)

**Summary: 3 passed, 0 failed, 0 blocked** (T-18, T-05, T-13), plus about 10 minutes of Back/Forward/reload exploration around the New episode page. That exploration found nothing that fails a case.

- **Environment:** HEAD `1361850`. The working tree is **not clean**: S13 and its fixes are uncommitted, plus planning-doc changes. Tested on 13 Oct 2026, 11:34–11:41 BST.
- **URLs:**
  - http://localhost:3000 as Cara Clinician (Hospital A1).
  - http://tester.localhost:3000 as Max Multi (Hospital A2 clinician, Trust B admin), for switching hospitals.
  - The conductor's `make dev` was running.
- **Data:** round 3. Nora Valid R3 (N1 `9990004102`, A1-N1) was created through the UI for this round. One extra episode for her ("MT R3 dblclick") came from the double-click probe. The round 2 records (Greta Elsewhere R2, Pat Nonhs R2) were only read, for T-13 step 4.
- **Browser:** Claude in Chrome.
  - Every click and keystroke went through the real UI.
  - Reloads were done with `location.reload()`.
  - Addresses, field values and `history.length` were read with small read-only page scripts.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-18 Change patient forgets the patient | Pass | **Step 1:** on Nora's patient page, **Episodes** is highlighted in the header. **Step 2:** New episode opens `…/episodes/new?patient=a2a046f6…`, showing the card and Hospital number A1-N1. **Step 3:** Change patient makes the address `…/episodes/new` with no `?patient=`. It replaces the history entry (`history.length` stays the same) and empties the Hospital number. After a reload: the search and the New patient form, with no patient chosen. **Step 4:** Back goes to Nora's patient page. Forward goes to New episode with no patient chosen, and all fields are empty. This holds with and without a reload before Back. **Step 5:** `…/patients/<id with the last character changed>` and `…/patients/not-a-uuid` both show **← Episodes** above a red "Patient not found." The link goes to `…/episodes`. **Switching hospital:** as Max, switching to B1 from Greta's A2 patient page goes to B1's **Episodes**. So does switching from `…/episodes/new?patient=…` at A2. |
| T-05 Search (quick) | Pass | `Va` searches nothing. `Val` shows the Patients found table. `999 000 4102` and `Valid R3` find Nora R3 only. `a1-n1` finds Nora R1/R2/R3 (all A1-N1). `a1-n` → "No patients at this hospital match. Enter their details below." `Va` + 1985-03-04 finds the three Noras. No contact details appear in the rows. Choose shows the card and fills A1-N1. Change patient brings back the search and form, and empties the Hospital number. `Aud` → 20 rows and "Showing the first 20. Type more, or add the date of birth, to narrow the search." |
| T-13 Patient page (quick) | Pass | Clicking the name on Nora's episode opens her patient page. It shows the NHS number, date of birth with age, sex, mobile and email, then "Episodes at Hospital A1". A row opens its episode. Back and reload work. New episode shows the card with "· Hospital number A1-N1" and fills the field with it, and this survives a reload. **Step 4:** Pat Nonhs R2 (A2-only record `67f9b8c9…`) at A1 shows "Patient not found." Greta Elsewhere R2's page at A1 lists only A1-G. |

## Failures

None.

## Found while exploring

None of these fails a case. They're notes for the builder.

1. **Choosing from the search doesn't survive a reload, unlike arriving from the patient page.** Search `Valid R3` → **Choose** leaves the address at `…/episodes/new` with no `?patient=`, so a reload drops the choice. Arriving from the patient page's New episode keeps `?patient=` and survives a reload. This is consistent with "Change patient forgets the patient", and the plan doesn't ask for it. *Cosmetic / by design?*
2. **Change patient keeps what was typed in the Episode section** (e.g. Procedure "Hip test" stays) and clears only the Hospital number. This seems right. It's noted only for the record.
3. **Back after creating an episode from `?patient=`** returns to the New episode page with the patient still chosen and an empty Episode form. Creating again from there would make a second episode, which is allowed. That's normal browser behaviour, not a bug.
4. **`…/episodes/new?patient=<a record with no episode here>`** shows the empty search and New patient form, with a red "Patient not found." at the bottom just above **Create episode**, and the address keeps `?patient=`. This is unchanged from round 2 and acceptable. The message sits far from the top, so it's easy to miss on a short screen. *Cosmetic.*
5. **Double-clicking Create episode** made exactly **one** episode. Nora's patient page lists two A1 episodes: the original one and the probe.
6. **Cancel** on `…/episodes/new?patient=…` goes to the Episodes list, not back to the patient's page. This is consistent with T-01. Arriving from a patient's page, some users might expect Cancel to return there. *Cosmetic suggestion only.*

## Console and network errors

- No console errors or warnings during the tracked T-18 run.
- Every tracked `/api/` request returned 200 (patient and published-HQs requests).
- Network tracking started late, after the not-found checks. So the "Patient not found." responses in T-18 step 5 and T-13 step 4 weren't captured, though the UI showed the expected messages.

## GIFs

- `~/Downloads/S13-T-18-r3.gif` (main happy path):
  - patient page → New episode → Change patient → reload → Back → Forward;
  - the not-found page and ← Episodes;
  - the search and Choose;
  - Back/Forward again.
