# S39: The suggested ASA and its reasons

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 2 of 8.
**Requirements:** ASA-01 ([saas-requirements.md](../../saas-requirements.md) §8), CLN-17 ([requirements.md](../../requirements.md)); pending.md §1a's "a suggested ASA from disclosures" on the POA Summary. ASA-05 is question 2.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S38 verified.

**What exists today:**
- A disclosure can carry an ASA grade I–VI with E (`ClinicalOutput.asa` in `packages/clinical/src/types.ts`; the ASA select in `features/outputs/OutputEditor.tsx`). There's no default from the code.
- `computeOutputs` (`outputs.ts`) returns `suggestedAsa`: the highest grade in **one** Question Set, with no reasons. Only the editor's preview shows it (`PreviewPage.tsx`), and test cases compare it ("Suggested ASA III", `testcases.ts` `outputLines`).
- The `codes` table (migration 00002; seeded from the incumbent's codes by `cmd/seed`) has no ASA. `GET /api/codes` (`internal/httpapi/codes.go`) searches it for the code picker (`features/outputs/CodePicker.tsx`).
- The POA Summary (`features/episodes/PoaSummary.tsx`, built from `poa.ts`) shows the nurse's and anaesthetist's ASA grades (`episodes.nurse_asa`, `anaesthetist_asa`, 1–6), but no suggested grade.

**Ask Rahul first:**
1. **Where each code's usual ASA grade comes from.** When an author adds "Heart failure (I50)" to an answer, the ASA box could fill in "III" by itself (the author can still change it).
   - (a) **A file in the repo lists the usual grade for some codes.** For now a few made-up ones for testing; a clinician fills in real ones before go-live. No screen.
   - (b) A superadmin screen to search a code and set its grade.

   **Recommended: (a).** Simplest; a screen can come when a clinician maintains the list.
2. **Using the suggested ASA in conditions,** e.g. show a clinician's "Anaesthetic plan" Question Set only when the suggested ASA is III or more.
   - (a) **Not now.** Clinician sets can already be shown by the patient's answers.
   - (b) Yes, in Question Set conditions of clinician sets.

   **Recommended: (a).** It was asked for a capture block, which was dropped (25 Oct).

## Build

- **The code's default ASA:** a nullable `asa` (1–6) on `codes` (next migration), loaded from the file in question 1 by `make seed` (idempotent). `GET /api/codes` returns it; integration test for the field (the route's five cases already exist; add the field to the happy path).
- **The editor:** adding a code to a disclosure that has no ASA yet fills in the code's grade; the author can change or clear it. **The grade is copied into the disclosure** (as a code's `display` is, so the form stands alone and a published version never changes when the file does). Unit test where the logic is plain TypeScript.
- **`packages/clinical`, test-first:**
  - **the suggested ASA for the whole HQ**: the highest grade across every shown Question Set's disclosures, with **the reasons**: each disclosure at that grade, in words ("ASA III: Heart failure (I50); Angina"). Built on S38's helper, so earlier sets' answers count the same way everywhere;
  - unit tests: no grades (none suggested), ties, E (emergency) kept, a hidden set's disclosures don't count, a band's disclosure counts.
- **Where clinicians see it:**
  - the **POA Summary**, in the ASA grades section: "Suggested ASA III" with its reasons, above the nurse's and anaesthetist's grades (both tabs: from the validated answers and from the patient's);
  - the episode page's Validate view, as one line at the top.
- **The preview** keeps its one-set suggested ASA, and adds the reasons.
- **E2E:** extend J5 (`e2e/journeys/test_j5_poa_summary.py`): the POA Summary shows the suggested ASA with its reason.

## Not in this session

- Rules and auto-triage (S40–S43); the suggested ASA in conditions (question 2).
- Changing the nurse's or anaesthetist's ASA controls.
- Storing the suggested ASA (S41 stores it at submit).

## Manual test focus

1. Add a code with a default grade to a disclosure: the ASA box fills in; change it; it keeps your change. A code with no default leaves it empty.
2. Two sets, one giving ASA II and one ASA III: the POA Summary says "Suggested ASA III" with the right reason.
3. Hide the ASA III set by its condition: the suggestion drops to II.
4. A clinician's change on Validate that removes a disclosure changes the validated tab's suggestion, not the patient's tab.
5. An episode whose answers give no ASA grade: "No suggested ASA", not an error.
