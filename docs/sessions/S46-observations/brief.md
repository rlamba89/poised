# S46: Observations

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), part 1 of 4.
**Requirements:** OBS-01, OBS-02, OBS-03, OBS-04's hospital BMI ([saas-requirements.md](../../saas-requirements.md) §9). OBS-04's precedence setting is S47's.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S45 verified (plan C6 done).

**Why first:** the assessments (S47–S49) pre-fill from the observations (`obs.*`), and STOP-Bang needs the hospital BMI.

**What exists today:**
- **The feature switch** `feature.observations` (hospital level, opt-in, on by default; `internal/settings/settings.go:106`). Nothing reads it yet.
- **The POA Summary** shows "Patient reported BMI" (`apps/web/src/features/episodes/PoaSummary.tsx:150`), taken from a BMI question or the library's `height`/`weight` questions (`packages/clinical/src/poa.ts`; decisions.md, "Patient reported BMI"). BMI itself is `bmi()` in `packages/clinical` (`questionTypes.ts`).
- **A history pattern to copy:** S44b's anaesthetist's ASA grades (`anaesthetist_asa_grades`, migration 00035): rows that are never changed, a reason for a re-grade, and "entered in error" marks, with triggers refusing edits.
- The episode page's panels (Tasks, flags, General notes) and the role ladder (`internal/role`).

**Ask Rahul first:**
1. **Ranges and limits** (OBS-03). Who sets each kind's soft range (a warning; the reading can still be saved) and hard limit (refused)?
   - (a) **One platform list in Go** (a file, like the ASA guide), sent to the web app with the readings. The values are made up until a clinician checks them.
   - (b) Platform defaults that each trust can change in its settings.

   **Recommended: (a).** Simple; a trust setting can come later if a trust asks.
2. **Correcting a reading** (OBS-02). The old row is always kept and marked superseded.
   - (a) **Correct** (a new value that supersedes the old one) **and Entered in error** (no new value), **each with a reason**, as S44b does for ASA grades.
   - (b) Correct only, with no reason.

   **Recommended: (a).** It matches the ASA grade history clinicians already use, and the reason explains the change.
3. **Units** (A-20: never hardcode the UK). Blood glucose is mmol/L in the UK and mg/dL in the US, and temperature is °C or °F.
   - (a) **Store one unit per kind** (cm, kg, mmHg, mmol/L, °C…). **Each country's display units come from Go's country config;** the UK's are the stored ones, and nothing converts until a second country needs it.
   - (b) Metric only, with the units fixed in the code.

   **Recommended: (a).** A new country is then config, not code (A-20), at the cost of one config field per unit that differs.

## Build

- **Migration** (next number): `observations` as the plan says (episode_id, kind, value, value2 for the diastolic, unit, taken_at, entered_by, superseded_by), plus what question 2 needs (the reason; an entered-in-error mark; who and when). Rows are never edited or deleted, except the one supersede or entered-in-error mark, as S44b's triggers do. Every query is scoped to the trust, and a hospital-scoped user sees only their hospital's episodes.
- **The kinds** (OBS-01): height (cm), weight (kg), blood pressure (systolic/diastolic; several readings), heart rate, SpO₂, temperature, respiratory rate, blood glucose, peak flow, plus ECG and urinalysis as notes (text). **BMI is never entered.** It's worked out from the latest height and the latest weight still counting on this episode, by `packages/clinical`'s `bmi()`, the same function the HQ uses.
- **Go, test-first** (`internal/observation`: the kinds, the units, the ranges as question 1 answers; unit tests at each limit's edge):
  - `GET /api/o/{oid}/h/{hid}/episodes/{eid}/observations`: the readings with their history, plus the kinds, ranges and units;
  - `POST …/observations`: add a reading. Outside a hard limit (a negative value, a diastolic above its systolic, a time in the future) → 400 with a plain reason;
  - `POST …/observations/{obsid}/correct` and `POST …/observations/{obsid}/entered-in-error`, as question 2 answers (a reason is required: 422 without one; acting on a reading that's already superseded: 409).
  - **Each route is `audited`** (patient data), and each gets the five integration cases (happy path, no sign-in, wrong role, another trust's data, invalid input). Any clinician at the hospital may record a reading. With `feature.observations` off for the hospital, the routes refuse (the builder picks the code and records it) and the panel is hidden.
- **Web:** an **Observations panel** on the episode page. It adds a reading (now by default, or an earlier time), warns softly outside the plausible range ("That's outside the usual range: save anyway?"), lists the readings newest first (value, unit, when, who), shows a corrected reading struck through with its correction, reason, who and when, and shows the hospital BMI. **POA Summary:** "Hospital BMI" next to "Patient reported BMI", with when it was measured.
- **E2E:** **J12** starts (`e2e/journeys/test_j12_observations.py`). Cara records height, weight and two blood pressures; a negative weight is refused; a heart rate of 220 warns, then saves; a correction keeps the old reading struck through; the POA Summary shows the hospital BMI next to the patient's. S47 adds STOP-Bang. Update `e2e/README.md`.

## Not in this session

- The trust setting for which BMI feeds the scores (OBS-04's second half), and anything that pre-fills or scores from observations: S47.
- Observations in the patient's view (patients never see them) or entered by the patient.
- Observations on POA documents (C9, DOC-01). Charts or trends across episodes.

## Manual test focus

1. As Cara, record each kind once. The list shows the value, unit, time and who.
2. A negative value and a future time are refused with a plain reason. An implausible heart rate warns, and can still be saved.
3. Correct a weight and mark a blood pressure entered in error, each with a reason. Both stay in the history, struck through, and the hospital BMI follows the corrected weight.
4. The POA Summary shows the hospital BMI beside the patient's own.
5. Val (Hospital A2) can't see or add Hospital A1's readings, and Trust B's users can't reach them at all. With `feature.observations` off for Hospital A1, the panel is gone.
6. Reload and deep links keep the panel and its history.
