# S48: DASI, PRISMA-7 and Apfel

**Plan:** C7 ([c7-observations-assessments.md](../../plans/c7-observations-assessments.md)), part 3 of 4.
**Requirements:** §9's assessments 2–4 ([saas-requirements.md](../../saas-requirements.md)); SCR-01 (ready-made scores in an HQ).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S47 verified.

**Why these three together:** each can be pre-filled almost entirely from the HQ (§9: DASI 12 of 12, PRISMA-7 7 of 7, Apfel 3 of 4, with post-op opioids from the clinician), and none needs observations. saas-requirements §14's phase 7 groups them with STOP-Bang.

**What exists today:** after S47, the assessment mechanism works for STOP-Bang: attach, pre-fill by key with sources, complete, Confirm, and the score and band on the POA Summary. `ready.ts` builds locked ready-made scores from `scores.ts`'s parts, and `ready.test.ts` checks STOP-Bang against its published scoring.

**Ask Rahul first:**
1. **DASI's result.** DASI gives a score (0–58.2) and an estimated functional capacity in METs.
   - (a) **The score and the METs, plus one band, "Poor functional capacity", at the published cut-off** (the builder cites the source; a clinician checks it before go-live).
   - (b) The score and the METs only, with no band.

   **Recommended: (a).** A band is what disclosures and rules can act on, as STOP-Bang's are.
2. **Where the three appear.**
   - (a) **Both as assessments and as ready-made scores an author can insert in an HQ**, from one definition (as STOP-Bang is).
   - (b) As assessments only.

   **Recommended: (a).** §9 relies on the HQ asking the same keyed questions, and an HQ that embeds the score makes the assessment complete at once.

## Build

- **Before any code: each instrument's terms.** Record the published source and the licence in each `ReadyScore` (`source`, `licence`), as STOP-Bang's are. **If any needs payment or permission for use in software, stop and ask Rahul** (STOP-Bang's licence is already a go-live item, decisions.md).
- **`packages/clinical`, test-first:** each instrument as a locked ready-made score in `ready.ts`, with its published wording, points and bands, and the Incomplete state while an item has no answer. **Its scoring is checked against worked examples from its published source** (`ready.test.ts`), including each band's edges. Keys follow STOP-Bang's pattern (`dasi_*`, `prisma7_*`, `apfel_*`).
- **As assessments** (S47's mechanism): each becomes an assessment a clinician can add, pre-filled by key, with Apfel's post-op opioids item left to the clinician. Patient sex comes from the profile, as STOP-Bang's does.
- **In the HQ** (question 2): Insert offers them beside STOP-Bang, and Publish refuses a changed one (`readyScoreProblems`).
- **POA Summary:** each confirmed one's score and band (DASI with its METs).
- **Integration:** attach each one; pre-fill sources per item; Confirm.
- **E2E:** extend J12: an HQ embedding DASI, sent by the patient (phone); the DASI assessment opens complete; Confirm; its METs show on the POA Summary.

## Not in this session

- MUST, falls, VTE and Rockwood (S49).
- Using DASI or PRISMA-7 as hints for Rockwood (S49).

## Manual test focus

1. Each of the three gives the published result for the source's worked examples (a tester enters them by hand).
2. An HQ embedding all three: the patient answers them (phone); each assessment opens filled, Apfel waiting only for post-op opioids.
3. The bands' edges are right (the test plan lists the values).
4. A changed ready-made score can't be published.
