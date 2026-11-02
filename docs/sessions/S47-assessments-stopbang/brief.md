# S47: Assessments, starting with STOP-Bang

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), part 2 of 4.
**Requirements:** §9 "Assessments through the HQ" ([saas-requirements.md](../../saas-requirements.md); marked *Proposed* there, agreed as plan C7's design on 2 Oct), OBS-04's precedence setting, ORG-05 (`feature.assessments`).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S46 verified.

**Why STOP-Bang alone first:** the plan says to ship it first if time is short. It's the most widely used pre-op screen, 7 of its 8 items come from the HQ, observations and profile, and S38 already built it as a locked ready-made score (`packages/clinical/src/ready.ts`, checked by `ready.test.ts`).

**What exists today:**
- `episode_forms.kind` already allows `assessment` (migration 00019). Only the HQ form is ever created, and a unique index allows one HQ per episode.
- **STOP-Bang** as a ready-made Group (`ready.ts`: `STOP_BANG`, `STOP_BANG_KEYS`), with its bands and the Incomplete state; an author can insert it in an HQ.
- **Keys and pre-fill** (C4): answers matched by question key (`internal/prefill`), the platform library's keys, and markers saying where a pre-filled answer came from.
- **Observations** (S46), with the hospital BMI. The patient's age and sex come from the patient record (`{patientAge}`, `{patientSex}`).
- `feature.assessments` (hospital level, on by default) exists but nothing reads it.
- **Key rule:** outputs are worked out in the browser from `packages/clinical`; Go stores and enforces. A clinician's browser is trusted, so an assessment needs no evaluate call.

**Ask Rahul first:**
1. **How an assessment gets onto an episode.**
   - (a) **A clinician adds it** from the hospital's list on the episode page ("Add assessment → STOP-Bang").
   - (b) Every episode gets STOP-Bang automatically when `feature.assessments` is on.
   - (c) Each procedure lists the assessments its episodes get (like its default HQ).

   **Recommended: (a)** now. (c) can follow once trusts say which procedures need which assessments.
2. **STOP-Bang's neck item** (the one item the HQ can't answer). J12 says "enter the neck size".
   - (a) **The clinician answers the item Yes or No**, as published (collar 43 cm for men, 41 cm for women).
   - (b) A neck circumference observation in cm; the item follows from it and the patient's sex.

   **Recommended: (a).** It's the published wording, and it adds no new observation kind.
3. **Which BMI feeds the score** (OBS-04: "a trust setting says whether observations take precedence in scores").
   - (a) **A trust setting, `obs_precedence`, on by default:** the hospital BMI when there is one, else the patient's. Each item says which it used.
   - (b) Always the hospital BMI when there is one, with no setting.

   **Recommended: (a).** The requirement names the setting, and the default does what (b) does.
4. **An HQ that already holds STOP-Bang** (§9: "the assessment is already complete").
   - (a) **It opens with every item filled from the HQ, and the clinician still confirms** (one click).
   - (b) It's marked complete without the clinician.

   **Recommended: (a).** A clinician confirms every assessment, and the confirmation records who and when.

## Build

- **Assessments as platform library forms:** a Go file (or seed) defines each assessment as locked SurveyJS JSON built from `ready.ts`'s parts, so the editor, the HQ and the assessment share one definition, wording, scoring and keys. STOP-Bang only in this session.
- **Attaching** (question 1): an `episode_forms` row of kind `assessment` (at most one of each assessment per episode; the builder decides the index). The route is `audited`, with the five integration cases. Patients never see assessment forms (check `chapter.ForPatient` and the patient routes refuse them).
- **Pre-fill, test-first** (`packages/clinical`, Vitest): each item's value and its **source**:
  - **HQ:** by key, from the HQ's answers that count now (the clinician's validated copy, else the patient's);
  - **observation:** `obs.*` (the hospital BMI), as question 3 says;
  - **profile:** age and sex;
  - **clinician:** what the clinician enters.

  Missing items are highlighted. Unit tests cover each source, the precedence both ways, an unanswered HQ question, and a hidden one (it counts as unanswered).
- **Completing:** the clinician fills the rest, then **Confirm** (who and when). The clinician's answers are stored as the form's clinician answers (`episode_answers`), with the revision check. Changing an item after Confirm needs a new Confirm.
- **POA Summary:** each confirmed assessment's score and band (STOP-Bang "5 (High)"), beside the hospital BMI. An unconfirmed one says "Not confirmed".
- **Integration:** attach; pre-fill sources reported per item; save and confirm; a stale revision → 409; another trust → 404; `feature.assessments` off → refused.
- **E2E:** **J12** completes (`test_j12_observations.py`). Cara records height and weight. STOP-Bang opens with 7 of 8 items filled, each with its source. She answers the neck item, then confirms. The score and band show on the POA Summary next to the hospital BMI.

## Not in this session

- DASI, PRISMA-7, Apfel (S48); MUST, falls, VTE, Rockwood (S49).
- Assessments feeding the rules or auto-triage (rules run on the patient's send only).
- Assessments on POA documents (C9).

## Manual test focus

1. Add STOP-Bang to an episode whose HQ asked S, T, O and P. It opens with 7 of 8 filled, each showing its source, and the neck item highlighted.
2. Record a hospital BMI over 35 after the patient's own under 35. With `obs_precedence` on, the BMI item uses the hospital's; off, the patient's.
3. Confirm: the score and band appear on the POA Summary. Change an item: it needs confirming again.
4. The patient never sees the assessment. Val (Hospital A2) can't open Hospital A1's, and Trust B's users get a 404.
5. An HQ that embeds STOP-Bang: the assessment opens complete, waiting for Confirm.
