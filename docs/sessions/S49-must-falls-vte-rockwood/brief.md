# S49: MUST, falls risk, VTE and Rockwood

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), part 4 of 4. C7 is done after this session.
**Requirements:** §9's assessments 5–8 ([saas-requirements.md](../../saas-requirements.md)). MMSE stays out (A-17).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S48 verified.

**What exists today:** after S48, four assessments (STOP-Bang, DASI, PRISMA-7, Apfel) on S47's mechanism, each also a ready-made score for the HQ. Observations (S46) give weight, height and the hospital BMI.

**Ask Rahul first:**
1. **Falls risk.** §9 says "confirm which tool the customer uses", and there's no customer yet.
   - (a) **Leave falls risk out until a trust names its tool.**
   - (b) Build one common published tool now (the builder proposes it, with its terms).

   **Recommended: (a).** A falls tool nobody asked for may not be the one a trust uses.
2. **Licences.** MUST (BAPEN), the VTE risk assessment (the Department of Health's, under NICE NG89) and the Clinical Frailty Scale (Dalhousie University) each have their own terms for use in software. The builder checks them first.
   - (a) **Build only what's free to use in software; list the rest as go-live items**, as STOP-Bang's licence is.
   - (b) Build all of them now, and settle the licences before go-live.

   **Recommended: (a).** Nothing is built that might have to be taken out.
3. **MUST's weight loss** needs an earlier weight. Observations are per episode.
   - (a) **Ask the patient in the HQ** (MUST's own question about unplanned weight loss), and take today's BMI from the observations.
   - (b) Use readings from the patient's earlier episodes.

   **Recommended: (a).** MUST asks the patient this anyway, and (b) reads across episodes for the first time.

## Build

- **Each instrument test-first,** as in S48: locked wording, points and bands in `ready.ts`, its source and licence recorded, and its scoring checked against its published worked examples.
- **MUST** (question 3): BMI from the observations (S47's precedence), weight loss from the HQ, and the acute-disease effect left to the clinician.
- **VTE:** the patient's risk factors pre-filled from the HQ; the procedure's and the clinician's items left to the clinician.
- **Rockwood CFS** (clinician-only, never in an HQ): the clinician picks the level, with DASI's and PRISMA-7's confirmed results shown as hints when the episode has them.
- **Falls** as question 1 says.
- **POA Summary:** each confirmed result.
- **E2E:** extend J12 with MUST, with BMI from the observations.
- **From S48's verification 1 (Rahul, 7 Nov):** each assessment form keeps the name of its own version. Today `ListAssessmentForms` reads `q.name`, which S48's upsert changes on a rename, so an old confirmed card and its POA line take the new name. Read the version's chapter name (`c.name`, stored per version by `PublishAssessmentVersion`) instead, with an integration case: rename, then the old episode keeps the old name and a new one shows the new name.

## Not in this session

- MMSE (A-17) and any other cognitive screen.
- Assessments feeding the rules.

## Manual test focus

1. Each instrument gives the published result for its source's worked examples.
2. MUST takes the hospital BMI; a correction to the weight (S46) changes it before Confirm.
3. Rockwood shows the DASI and PRISMA-7 hints when they're confirmed, and nothing when they're not.
4. Rockwood never appears in an HQ, and the patient never sees any assessment.
