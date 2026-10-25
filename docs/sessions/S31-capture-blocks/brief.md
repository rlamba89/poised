# S31: Clinician capture blocks

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 1 of 6.
**Requirements:** CAP-01, CAP-02 ([saas-requirements.md](../../saas-requirements.md) §8); A-7 (§0).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S30 verified.

**Why first:** it's the smallest part of C5 and touches only the editor, the clinician view and the POA Summary. It also starts journey J10, which the later sessions extend.

**What exists today:**
- A Group can already be switched to **Clinical** (`clinicianOnly`, `SettingsPanel.tsx`'s `ClinicalSwitch`). A Section can't. Go's `chapter.ForPatient` (`apps/api/internal/chapter/patient.go`) and `viewer.ts`'s `stripClinicianOnly` remove anything `clinicianOnly` before a patient sees it.
- In the clinician view (`features/preview/clinicianView.ts`, used by the preview and `ValidateSet.tsx`), clinician items are tinted and sit in the right-hand column, beside the patient question above them (`.sj-clinician-layout` in `globals.css`).
- The POA Summary (`packages/clinical/src/poa.ts`, `features/episodes/PoaSummary.tsx`) shows each page's notes and comments, the changes, the pre-filled answers, and tables for the Medication, Admissions and BMI answers. Those tables are already called `captures` in `poa.ts`; **name the new thing differently** (e.g. `captureBlocks`) so the two don't get mixed up.

**Ask Rahul first:**
1. **How a capture block relates to today's "Clinical" switch on a Group.**
   - (a) **A new switch, "Clinician capture", on Groups and Sections.** A capture block is a full-width tinted box that clinicians fill in, and its answers go on the POA Summary. Today's "Clinical" switch stays as it is, for a small clinician box beside a patient question.
   - (b) Today's "Clinical" switch on a Group becomes the capture switch: every clinician-only Group is shown as a capture box and listed on the POA Summary.

   **Recommended: (a).** Nothing already built changes, and the two do different jobs.

## Build

- **`packages/clinical`, test-first:**
  - a hidden `capture:boolean` property on panels (`questionTypes.ts`, beside `patientPage`);
  - an editor action that switches capture on or off. On sets `capture: true` **and** `clinicianOnly: true` on the block itself, so Go's strip and `stripClinicianOnly` remove it whole and a capture Section never leaves the patient an empty screen. (The plan says "on every child"; setting it on the block does the same job. Record the difference in the build notes. A question dragged out of the block becomes a patient question again.)
  - `poaSummary` lists each capture block that has answers: its title, then each answered question as "question: answer" (`describeAnswer`), in order, empty ones left out. Validated tab only: the patient never sees these questions.
- **Editor:** the switch on Groups and Sections (not repeating groups), a "Clinician capture" badge and tint on the card and in the Structure tree. The block can hold any question type.
- **Clinician view:** the capture block is a full-width tinted box across both columns (a `sj-capture` class from `clinicianModel`), with its title. Clinicians fill it in in Validate like any clinician question, and its answers save with the clinician's copy.
- **POA Summary:** one block per capture block, under its title, with S27's clickable answer links.
- **Go:** `chapter.ForPatient` also drops any element with `capture: true`, even if `clinicianOnly` is missing (a hand-made JSON can't leak it). A difference from the plan, which says the strip doesn't change: record it.
- **Integration test:** a published HQ with a capture Section and a capture Group: the patient's JSON (`GET /api/p/episodes/{eid}`, and the staff on-behalf route) has neither, nor their questions; a patient save that names one of their questions has that answer dropped (`FilterAnswers`).
- **J10 starts** (`e2e/journeys/test_j10_authoring.py`): an author adds a capture Section and publishes; the patient (phone size) never sees it; the clinician fills it in during Validate; the POA Summary shows its answers under its title.

## Not in this session

- Clinical Summary elements (S32), videos and the content library (S33, S34), Information blocks (S35).
- Score panels inside a capture block: they come with C6. Showing a capture block only when ASA ≥ 3 (ASA-05) is C6 too.
- Changing clinician-only Question Sets or "Clinical" Groups.

## Manual test focus

1. Switch a Section to Clinician capture, add three questions of different types, save, reload: the badge and tint stay.
2. Preview as patient: the Section is gone, with no empty screen. Preview as clinician: a full-width tinted box.
3. Publish, send to a patient, fill in on a phone-size window: the capture block never shows. Staff filling in on the patient's behalf don't see it either.
4. Validate: fill in the capture block. The POA Summary's validated tab lists its answers under its title; each opens the question. The patient tab doesn't show it.
5. Drag a question out of the capture block: it's a patient question again. Switch capture off: the block is a normal Section.
6. An HQ without capture blocks looks exactly as before in every view.
