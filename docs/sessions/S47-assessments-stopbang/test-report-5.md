# S47 test report 5 (retest after test report 4): F-1, age and sex after a change

**6 cases: 6 passed, 0 failed, 0 blocked.** Report 4's F-1 is fixed.

- **Environment:**
  - commit `10b1087` plus the uncommitted S47 work (61 changed or new files in `git status`). The web files `AssessmentsPanel.tsx`, `assessments.ts` and `assessments.test.ts` changed at 14:07–14:08.
  - local `make dev` at http://localhost:3000 (not restarted, as the plan says: a web change only). API :8080 healthy, evaluate :8090 `clinical@10b10878b8d9-dirty`.
  - Friday 6 Nov 2026, 14:28–14:39 IST.
  - Chrome (Claude in Chrome), desktop width (1456–1512).
- **Scope:** T-21 in full with the extended step 4, then the round's regression cases T-03 and T-07, with T-01, T-02 and T-05 as setup and happy paths.
- **Database use:**
  - T-21 step 2's `UPDATE patients …`, as the plan says. Docker isn't on the PATH, so I ran it with the local `psql` against the same database.
  - Read-only `SELECT`s on `episode_answers`, `episode_forms`, `form_outputs` and `audit_log`, to confirm what a Confirm stored and to look into the 503 below.
- **Test data (all made up, Hospital A1, as Cara Clinician):**

| Episode | Patient | Setup | Used for |
| --- | --- | --- | --- |
| E1 `1ab64790` | Rex MT S47R5 1432 One (M, 1 Mar 1960; changed to F, 1 Mar 1990 in T-21) | Demo sleep HQ, filled in for the patient: 170 cm, 82 kg; S, T and P Yes, O No. Auto-triaged to Ready for admission. Hospital height 180, weight 120. | T-01, T-02, T-03, T-21, exploring |
| E2 `0349709e` | Tom MT S47R5 1436 Two (M, 15 Jul 1958) | Demo STOP-Bang HQ, filled in: 170 cm, 82 kg; S, T and P Yes, O and N No. Ready for review. No readings. | T-05 (happy path, not confirmed), T-07 |
| E3 `78f5c157` | Ian MT S47R5 1442 Three (M, 2 Feb 1970) | Demo STOP-Bang HQ, filled in: no height or weight; all five No. | Only to look into the 503 below |

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-21 Age and sex frozen at Confirm, with the extended step 4** | **Pass** | E1, confirmed at 7 (High risk), N Yes. After the `UPDATE` and a reload: the header says "1 March 1990 (age 36)"; the card keeps A 66 and G Male (Profile) and 7; the yellow box has both lines as written; the hint now ends "Once any item is changed, the age and sex are the record's."; the POA Summary still says 7 (High risk). **Step 4:** N Yes → No gives A **36**, G **Female** (both Profile), **4 (High risk)**, "Changed: confirm it again", and the note "The age is the patient's today."; the yellow box goes. Undo changes puts back 66 / Male / 7 and the yellow box. N → No again, then **Confirm**: "Confirmed by Cara Clinician on 06/11/2026, 14:33", 4 (High risk). After a reload: 36 / Female, 4, Confirmed, **no yellow box**, and the note says "…on the day it was confirmed (36).". POA Summary: **STOP-Bang 4 (High risk)**, Age 36. The stored row (revision 4) has `patientAge 36` and `patientSex "female"`, with `validated_at` set. |
| T-03 A change needs a new Confirm | Pass | E1: N Yes gives "Changed: confirm it again", 7 (High risk), Confirm on. Undo gives No, 6 and "Confirmed by… 14:30". Yes and Save give "Not confirmed", and the POA says "STOP-Bang Not confirmed". Confirm gives "Confirmed by … 14:32", and the POA says 7 (High risk). |
| T-07 The clinician changes a pre-filled item | Pass | E2, not confirmed: S Yes → No gives **Clinician** and a score one lower (5 → 4). After Save and a reload: S is still No / Clinician; T, O, P and N are still HQ; Save is greyed out again. |
| T-01 7 of 8 filled (setup) | Pass | E1: "No assessments yet." before the add. Then "7 of 8 filled: answer the item highlighted."; S, T, O and P are Yes/Yes/No/Yes from the HQ; B 37 Hospital BMI; A 66 and G Male from Profile; N Missing (orange); Incomplete; Confirm and Save greyed out. |
| T-02 Confirm and the POA Summary (setup and regression) | Pass | E1: N No gives Clinician and 6 (High risk). Confirm gives "Confirmed by Cara Clinician on 06/11/2026, 14:30", with Confirm greyed out. The POA Summary shows "Hospital BMI 37 (measured 06/11/2026, 14:30)" with "STOP-Bang 6 (High risk)" under it, on both tabs. |
| T-05 A complete HQ, waiting for Confirm (happy path) | Pass (first part only) | E2: "8 of 8 filled."; all five Yes/No items HQ; B 28.4 **Patient reported BMI** (no readings); A 68 and G Male Profile; nothing highlighted; 5 (High risk); Not confirmed; Confirm on. I then used it for T-07, not Confirm. |

## Failures

None.

## Found while exploring

1. **A double-click on Confirm sends one request.** On E1, with N changed to Yes, a double-click on Confirm sent a single `POST …/confirm` (200). The card showed 5 (High risk) on 36 / Female, "Confirmed … 14:39", and no errors.
2. **The 503 on "Send the answers" is the recording tool's mistake, not the app's.** For both Fill in for the patient sends that the tool recorded (E2 and E3), Claude in Chrome's network log says `POST …/patient-form/submit` **503**. But:
   - the screen showed "Sent for Ian MT S47R5 1442 Three…" (E3);
   - the episodes moved to Ready for review;
   - `episode_forms.submitted_at` was set, and `form_outputs` holds the evaluator's result;
   - the server's audit rows say `behalf.submit` **204** (rows 90814 and the one at 09:09:03Z).

   The tool seems to record a 204 No Content reply as 503. I'm noting it so the next tester doesn't chase it.
3. **Audit rows with 4xx from before this run:** `assessment.add` 409 (×3), `observation.add` 400 (×2), `observation.correct` 409 (×2) and `assessment.list` 499, all by Cara at 14:09–14:12 IST, before I started. They look like the builder's e2e or test runs, not part of this round.

## Console and network errors not tied to a case

- **Console:** no errors or warnings once tracking started (from T-07's reload to the end).
- **Network:** no 4xx or 5xx, apart from the 503 explained above. All the assessments requests (POST 201; PUT, confirm and GET 200) were fine.

## GIFs

- `~/Downloads/S47-retest5-T21.gif`: the main happy path and the fix. It covers T-01's add (7 of 8, N Missing), T-02's Confirm at 6, the POA Summary, T-03 (change, Undo, Save, re-Confirm at 7), and T-21 with the yellow box. The recording hit its 50-frame limit, so it may stop before the end of T-21 step 4. The T-21 row above records what step 4 showed.
