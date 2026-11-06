# S47 test report 4 (retest after verification 3): Rahul's answers 5–7

**7 cases: 6 passed, 1 failed, 0 blocked.** There's also one problem in the test plan itself (T-19 can't be set up as written), which I worked around.

- **Environment:**
  - commit `10b1087` plus the uncommitted S47 work (60 changed or new files in `git status`);
  - local `make dev` at http://localhost:3000, restarted at 13:06:40, after the last Go change (12:45). API :8080 healthy, evaluate :8090 `clinical@10b10878b8d9-dirty`;
  - Friday 6 Nov 2026, 13:07–13:52 IST;
  - Chrome (Claude in Chrome) at 1512 wide. The window still won't go narrower (as in reports 2 and 3), so the patient's part ran at desktop width.
- **Scope:** T-19 to T-21 (new), then the round's regression cases: T-03, T-07 and T-02's POA Summary (with T-01 as their setup).
- **Database use:**
  - T-21 step 2's `UPDATE patients …`, as the plan says. Docker wasn't on the PATH, so I ran it with the local `psql` against the same database.
  - Read-only `SELECT`s on `episode_answers`, to see what a Confirm stored.
- **Test data (all made up, Hospital A1):**

| Episode | Patient | Setup | Used for |
| --- | --- | --- | --- |
| E1 `784a8b65` | Ray MT S47R4 1310 One (M, 1 Mar 1960; changed to F, 1 Mar 1990 in T-21) | Demo sleep HQ, filled in for the patient: 170 cm, 82 kg; S, T and P Yes, O No. Auto-triaged to Ready for admission. Hospital height 180, weight 120. | T-01, T-07, T-02, T-03, T-21, exploring |
| E2 `7a905025` | Tim MT S47R4 1320 Two (M, 10 May 1960) | Demo STOP-Bang HQ, sent by the patient through the invite link: 170 cm, 82 kg; S, T and P Yes, O and N No. Stayed at Ready for review. Hospital height 180, weight 120, then corrected to 90. | T-19, T-20, exploring |

## Results

| Case | Result | Note |
| --- | --- | --- |
| **T-19 Pre-fill uses the clinician's started copy** | **Pass** (on the Demo STOP-Bang HQ; see test-plan problem 1) | E2: in Validate I changed Tired to No ("Patient answered: Yes", Saved) without validating. Add assessment → STOP-Bang: T **No**, source **HQ**, 4 (High risk). After Validate and a reload: T still No / HQ. |
| **T-20 A confirmed STOP-Bang says what's changed since** | **Pass** | E2. Step 1: B 37 Hospital BMI, Confirm gives **5 (High risk)**, "Confirmed by Cara Clinician on 06/11/2026, 13:19", no yellow box. Step 2: weight corrected 120 → 90 ("MT retest 4: scale misread"). The yellow box says "The hospital BMI is now 27.8 (37 when confirmed)." and "The score stays as confirmed…". B is still 37 / Hospital BMI, still 5, still Confirmed. Step 3: the reload keeps it all; the POA Summary says Hospital BMI 27.8 and **STOP-Bang 5 (High risk)**. Step 4: 27.8 typed in B, then Confirm: B **Entered by clinician**, **4 (High risk)**, the yellow box gone. |
| **T-21 Age and sex frozen at Confirm** | **Fail** (steps 1–3 pass; step 4 fails) | E1, confirmed at 7 (High risk). After the `UPDATE`: A 66 and G Male (Profile), still 7, and the yellow box with both lines exactly as written; the POA Summary is unchanged at 7. **Step 4:** after a change to N, A still shows **66** and G **Male**, and the score drops only by N's point. Expected: 36 and Female. Undo changes correctly puts back the confirmed state. See F-1. |
| T-01 7 of 8 filled (setup) | Pass | E1: S, T, O, P Yes/Yes/No/Yes from the HQ; B 37 Hospital BMI; A 66 and G Male from Profile; N Missing (orange); Incomplete; Confirm greyed out; the Add assessment menu gone. |
| T-07 The clinician changes a pre-filled item | Pass | E1, unconfirmed: S → No: **Clinician**. Save, then reload: S still No / Clinician, and T, O and P still HQ. With N No the score was 5; putting S back to Yes gave 6. |
| T-02 Confirm and the POA Summary | Pass | E1: Confirm at 6 (High risk), "Confirmed by Cara Clinician on 06/11/2026, 13:11", kept after reload. POA Summary: "Hospital BMI 37 (measured 06/11/2026, 13:10)", then "STOP-Bang 6 (High risk)", on both tabs. |
| T-03 A change needs a new Confirm | Pass | E1: N Yes gives "Changed: confirm it again", 7 (High risk), Confirm on. Undo: 6 and "Confirmed by…". Yes and Save: "Not confirmed", and the POA says "STOP-Bang Not confirmed". Confirm: "Confirmed by … 13:12", and the POA says 7 (High risk). |

## Failures

### F-1 (Major): a changed confirmed assessment keeps the age and sex it was confirmed with, and a new Confirm stores them again

The card keeps the frozen age and sex after the assessment is changed, which contradicts answer 7 and T-21 step 4. A Confirm made straight from that state stores the old values again. So the confirmed STOP-Bang keeps scoring a 36-year-old woman as a man over 50, and the clinician can't fix it from the card.

- **Signed in as:** Cara Clinician.
- **URL:** `/o/…fa/h/…0a/episodes/784a8b65-dc7b-42de-84c4-738bc837908a`

**Steps:**
1. Confirm a STOP-Bang for a man over 50 (E1: 7 (High risk), A 66, G Male).
2. `UPDATE patients SET date_of_birth = '1990-03-01', sex = 'female' WHERE last_name = 'MT S47R4 1310 One'`
3. Reload the episode page. So far, as expected:
   - the header says "1 March 1990 (age 36)";
   - the card keeps 66 / Male / 7;
   - the yellow box says "The patient's record now gives an age of 36 that day (66 when confirmed)." and "The patient's record now says Female (Male when confirmed)."
4. Click N's other answer (Yes → No).

**Expected (T-21 step 4, answer 7):** once changed, A and G follow the record: A **36**, G **Female**. With N No that's 7 − 3 = **4**.

**Actual:**
- A **66** / Profile, G **Male** / Profile;
- the badge says **6 (High risk)**: only N's point went;
- "Changed: confirm it again" shows;
- the note under the buttons says "The age is the patient's today." while the age shown is 66, not today's 36.

**It gets worse with Confirm:**
5. In that state, click **Confirm**.
   - The card says "Confirmed by Cara Clinician on 06/11/2026, 13:15", score 6 (High risk).
   - `episode_answers` (revision 5) stores `patientAge 66`, `patientSex "male"`, and a new `validated_at` (6 Nov), though the record now says 36 and female.
   - The yellow box comes back straight away with the same two lines and "To use a new value, change the item and confirm again". But A and G can't be changed, and confirming again takes the old values.
   - So the box can't be cleared from the card.

**What does work:**
- Undo changes puts back the confirmed state correctly: 7, "Confirmed by…", and the yellow box.
- **Save, reload, then Confirm** does use the record:
  - after Save (revision 6) and a reload, the unconfirmed card shows A **36** / Profile, G **Female** / Profile, 5 (High risk);
  - Confirm from there stored `patientAge 36`, `patientSex "female"` (revision 7).
- The Save itself stored 66/male again (revision 6). Per the build notes an unconfirmed row's values are never read, so that alone does no harm.

**The likely place to look (a tester's guess):**
- The edit draft of a confirmed row seems to start from the stored answers, `patientAge` and `patientSex` included.
- `assessmentResult` / `assessmentItems` then prefer those stored values, though the draft is no longer "confirmed and unchanged".
- The build notes' unit case "an unconfirmed row following the record" covers a saved unconfirmed row, not a confirmed row with unsaved changes.

**Console and network:** none. Every request was 200.

**GIF:** `~/Downloads/S47-retest4-T21-fail.gif`. Undo gives the confirmed 7 and the yellow box; N → No then gives 6, with A 66 and G Male still shown.

## Test-plan problem (not an app fault)

1. **T-19 can't be set up on the Demo sleep HQ.**
   - The platform default rules have `noGradeAutoTriage: true`, and the Demo sleep HQ asks nothing that gives an ASA grade.
   - So "Low risk" auto-triages every Demo sleep HQ episode to Ready for admission on send, which completes the HQ review.
   - On E1 the Sleep set offered only **View**: "The HQ review is complete, so these answers can't be changed." The status menu has no way back to Ready for review.
   - **What I did instead:** ran T-19 and T-20 on the **Demo STOP-Bang HQ**. The patient's STOP-Bang of 5 is High, so "Low risk" (`notBand` High) doesn't fire, and the episode stays at Ready for review with **Validate**.
   - T-20's expected scores still came out as written (5, then 4).
   - The plan should say the Demo STOP-Bang HQ (with a High STOP-Bang). The other option is a sleep HQ whose rules don't auto-triage.
   - Report 3's E2 (Demo sleep HQ) was auto-triaged too; it just didn't matter then.

## Found while exploring

1. **Answer 6's HQ line works.** On E2, confirmed, I changed Snoring to No in the clinician's copy (which un-validated the set, as designed). The card then said "The HQ now says No to Snoring (Yes when confirmed)." while keeping 4 and "Confirmed by…".
2. **Enter submits an observation reading.** Typing a height or weight and pressing Enter saves it. That's good for keyboard use.
3. **A saved clinician answer stays "Clinician" when set back to the HQ's value** (E1: S saved as Clinician No, then clicked Yes, which is what the HQ said: still Clinician). That seems right: it's the clinician's saved answer, unlike T-17 step 2's unsaved round trip. I note it only because it differs from that case.
4. **Double-check of T-21 step 3's age note:** before the change, the note reads "The age is the patient's on the day it was confirmed (66)." This is consistent with the stored age.

## Console and network errors not tied to a case

- **Console:** no errors or warnings, from the time tracking started (after T-20) to the end.
- **Network:** no 4xx or 5xx. All `/api/` requests in the tracked part were 200, including the set's PUT and the assessments GET.

## GIFs

All in `~/Downloads/`:
- `S47-retest4-T02-T03.gif`: the happy path: Confirm at 6, the POA Summary on both tabs, then T-03 (change, Undo, Save to Not confirmed, re-Confirm, POA 7).
- `S47-retest4-T20.gif`: Confirm at 5, the weight corrected to 90, the yellow box, the POA Summary, then 27.8 typed and confirmed at 4.
- `S47-retest4-T21-fail.gif`: F-1.
