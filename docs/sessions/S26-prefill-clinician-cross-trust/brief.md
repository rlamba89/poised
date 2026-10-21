# S26: Pre-fill for the clinician, and answers from other trusts

**Plan:** C4 ([c4-question-keys-prefill.md](../../plans/c4-question-keys-prefill.md)), part 4 of 4.
**Requirements:** PRE-04; §6 point 7 of [saas-requirements.md](../../saas-requirements.md#6-pre-filling-from-earlier-hqs-agreed-need-proposed-design); ORG-04's `crossTrustPrefill` (already a trust-only setting, default off, S10). Ends plan C4.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S25 verified.

**Ask Rahul first:**
1. **What the clinician sees (PRE-04)** in Validate and on the POA Summary. **Recommended:**
   - on each pre-filled answer: "Pre-filled from *Demo pre-op HQ*, 3 Mar 2026" (plus the hospital when it's another trust's);
   - "Changed by patient" where the final answer differs from the pre-filled one, showing the old value as corrections do today;
   - per set: "Patient chose Everything is still correct" (S25's recorded button), so the reviewer knows which answers weren't re-read.
2. **Both trusts must allow it.** **Recommended:** answers from trust B are offered at trust A only when **both** have `crossTrustPrefill` on. Switching either off stops new offers; answers already copied stay.
3. **The patient's agreement.** **Recommended:**
   - before the first Question Set, a screen: "Use the answers you gave St Mary's in March 2026?" with **Yes, use them** and **No, thanks**;
   - stored per account and source trust (`patient_prefill_consents`: account, trust, yes/no, when), so it's asked once per trust;
   - **Yes** fills, from the other trust, only questions still unanswered and untouched, after same-trust pre-fill (which needs no agreement);
   - the choice can't be changed in this session (a "Change" link later, if needed).
4. **No consent screen at all when staff fill it in on the patient's behalf.** Staff can't agree for the patient, so only same-trust answers are used. **Recommended: yes.**
5. **Auditing the other trust's read.** **Recommended:** when answers are copied from trust B, an `audit_log` row is written under trust B as well (who: the patient; what: the source form ID), so trust B can see its data was used.

## Build

- **The clinician's view:** the markers from answer 1 in `ValidateSet` and `PoaSummary` (`packages/clinical/poa.ts` gets the `prefill` map beside `original`). Unit tests in `packages/clinical` for "changed by patient".
- **Cross-trust pre-fill** in `internal/prefill`: the trust rule and the consent rule, unit-tested. The other trust's answers are read with its own `org_id`, one trust per query, only after both checks pass. This is the first route that reads another trust's patient data, so the verifier reviews it for tenant isolation.
- **`patient_prefill_consents`** (next migration) and `POST /api/p/episodes/{eid}/prefill-consent` (yes/no), with the five integration cases (another trust's data here meaning another patient's episode). Audited.
- **The patient's HQ:** the consent screen, shown only when another trust has answers to offer and the patient hasn't been asked for that trust.
- **Integration tests:**
  - pre-fill from another trust only after consent, and only with both settings on;
  - "No, thanks" is remembered and the screen isn't shown again;
  - nothing the patient already typed is overwritten by the other trust's answers;
  - the on-behalf route never uses another trust's answers;
  - the audit row under the source trust.
- **End-to-end, J16 completed:** after the patient submits, Cara sees the pre-filled markers and one "Changed by patient". **J16b** (`test_j16b_prefill_cross_trust.py`, phone size): with both trusts' setting on, a patient with a Trust B episode agrees at Trust A, and Trust B's answers are filled in.
- **Plan C4 done:** its Status section and the plans table updated.

## Not in this session

- Pre-fill across countries (never: separate deployments, saas-requirements.md §13).
- Changing a consent decision; withdrawing consent.
- The hazard log entry itself (OPS-04, C10). Note in the build notes what it should say about pre-fill.

## Manual test focus

1. As Cara, validate a pre-filled HQ: each pre-filled answer names its HQ and date; the one the patient changed says "Changed by patient" with the old value; the set says which button the patient pressed. The POA Summary shows the same.
2. With `crossTrustPrefill` on for Trust A only: no consent screen, no Trust B answers. Both on: the screen names Trust B's hospital and month.
3. Yes: Trust B's answers fill only what's still empty. No: nothing from Trust B, and the screen doesn't come back on another visit or another episode at Trust A.
4. Fill in on the patient's behalf: no consent screen, same-trust answers only.
5. The audit log of each trust shows the reads (Ada for Trust A; Max for Trust B).
