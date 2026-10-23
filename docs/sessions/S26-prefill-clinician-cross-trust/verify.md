# S26 verification

## Verification 2 (23 Oct 2026, verifier, automatic run)

**Verdict: Verified.** Verification 1's findings are fixed, and the whole S26 change was reviewed again against the brief, answers.md and Rahul's rule after verification 1. **Plan C4 is Done.**

### Verification 1's findings: all fixed

1. **Another trust's key match copied option values without checking their meaning:** fixed under Rahul's rule.
   - `earlierForm.find` (`apps/api/internal/prefill/prefill.go:202`) matches another trust's question by key only if the key is a platform library key. Otherwise it matches by exact wording, with the "text used twice" and `SameWording` safeguards.
   - `fit` gets `byKey = by == "key" && !f.otherTrust` (`prefill.go:132`), so another trust's choices are always mapped by option text: single, multiple, Yes/No, rating, grid rows and columns, list columns and their options, and a repeating group's children.
   - The platform library keys come from `ListPlatformLibraryElements` (`org_id IS NULL`), read only when another trust is used; `LibraryKeys` includes a library group's children.
   - Tested: `TestOtherTrustsPlatformLibraryKeys`, `TestOtherTrustsByWordingOnly`, `TestLibraryKeys`, and `TestPrefillCrossTrust` (trust B's own `q_smokes` with swapped values isn't filled; with a platform library row it's filled as trust A's `o_yes`, "key").
2. **Trust B's form id in trust A's rows and responses:** fixed. `Entry.FromForm` is empty (`omitempty`) for another trust (`prefill.go:142`). `TestPrefillCrossTrust` checks that none of trust B's seven ids appears in trust A's stored answers or in the staff `…/hq` response.
3. **Unreadable settings 500'd the form:** fixed. `crossTrust` logs and skips that trust (`apps/api/internal/httpapi/prefill.go:141`); tested.
4. **The consent card hid its error:** fixed. The message is set after the reload (`PatientHQ.tsx`, `answerConsent`); J16b's `test_a_refused_answer_keeps_its_message`.
5. **Notes:** the stale `-Comment` row (`TestStaleCommentInARow`) and the order of several trusts (newest first) are fixed; the rest are in pending.md.

### Tenant isolation, reviewed again

- **One trust per query:** `ListPrefillSources` filters `e.org_id` and `p.org_id` by the one trust asked about (`episodes.sql`), and it's called once per agreed trust.
- **Only after both checks:** this trust's setting (`hospitalSettings`), then each other trust's own setting and the patient's stored Yes (`crossTrust`). Deciding whether to ask (`ListPrefillOffers`) reads only the other trust's settings and its newest sent HQ's hospital and date, never answers. While a question is open nothing is filled.
- **Staff on the patient's behalf never use another trust:** `crossTrust` is skipped when `enteredBy` is set (`prefill.go:60`). Tested ("staff filling in for the patient never use another trust's answers, even after a Yes").
- **No trust B identifiers in trust A's data or responses:**
  - in the local database, no other-trust entry has `fromForm`;
  - as Cara in the browser, T-07 (R3)'s `…/hq` holds only trust A's three chapter ids, with 9 `otherTrust` entries;
  - no audit row in any trust names another trust's episode or patient (SQL over the whole `audit_log`).
- **Audit under the source trust:** `prefill.read` rows are written under trust B at the source episode's hospital (8 locally). Trust A's own row gets `link.prefill_consent` and only trust A's ids.
- **Hospital scoping still holds:** Max (a clinician at Hospital A2) gets a 404 for T-07's episode at Hospital A1.

### Findings

Nothing blocks. The `/code-review high` findings were weighed, and these minor items went to pending.md ("From S26's verification 2"):
1. **What the patient's Yes covers (Rahul to judge).** The question names the newest HQ's hospital and month, but a Yes covers every HQ the person sent that trust, for every later HQ. That's what answers.md Q3 says ("per account and source trust"). The card doesn't say that later HQs will use them too.
2. **Hazard log: a platform library key one trust rewords** before its first publish still matches "by link". Choices are mapped by text, so only a reworded Yes/No that now means something else would carry. This is the residual risk of Rahul's rule. Keys are generated (`q_xxxxxx`, also on import), so a trust's own question can't take a library key by accident. The build notes' hazard-log line "by wording only" (line 216) predates the rule.
3. **A repeating group's row with only a clinician-only cell refuses the whole list** (`fitRows`): on the safe side.
4. **`PrefillEntry.fromForm`** (`packages/clinical/src/prefill.ts:16`) should be optional; nothing reads it.

Also from the review, and already recorded or decided:
- repeated reads and `prefill.read` rows on every open while a set has no row (pending.md, verification 1);
- ValidateSet's count fixed at load (pending.md);
- staff opening first leaves a partly filled set closed to another trust (build notes; "if in doubt, don't fill");
- the consent decision is audited under the form's trust only (answers.md Q5 asks for the read under the source trust, which is done).

### Test results (verifier's own runs)

- `make lint`: passed.
- `make test`: Go all ok (`internal/repocheck` ok); Vitest 219 web and 150 clinical passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (on `make dev`): 63 passed, 5 skipped, 2 failed. The two failures are the known flakes, S11's audit tooltip check and S20's stale-tab check, and both pass when run alone (2 passed). J16 and J16b passed.
- **Settings afterwards:** `crossTrustPrefill` is `false` in Trust A and Trust B.
- **A-20 grep:** only comments (`format.ts`, `patient.go`).
- **Browser spot-checks** (Claude in Chrome, "rahul chrome"; read-only, no setting changed):
  - Cara's Validate on T-07 (R3): Trust B's answers say "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust), 23 Oct 2026". Library questions (snoring, sickness, anaesthetic problems) say "(matched by link)", and diabetes says "(matched by same wording)". Pregnancy has no marker.
  - The staff `…/hq` has no `fromForm` and no trust B uuid.
  - Max at Hospital A2: 404 for that episode.

### Hand checks for Rahul

None.

### Suggested commits

1. `S26: pre-fill from other trusts and the clinician's pre-fill markers`, with its tests (as the build notes' final suggestion lists).
2. `docs: S26 verified — build notes, test plan and reports, verification, decisions, pending, CLAUDE.md, AGENTS.md, plan C4 done and session status`.

### Next step

Plan C4 is Done. The next plan, C5, isn't split into sessions yet.

## Verification 1 (23 Oct 2026, verifier, automatic run)

**Verdict: Needs fixes.** Two problems must be fixed, test first, before S26 can be Verified and plan C4 marked Done:
- another trust's answers matched **by key** are copied **by option value without checking the option's wording**, so a patient's answer can change meaning on the way (a clinical-safety bug, reproduced below);
- trust B's form ID is stored in trust A's rows and sent to trust A's staff.

Everything else holds up well:
- **Tenant isolation:** the patient's answers are read one trust per query, by that trust's `org_id`, and only after both trusts' `crossTrustPrefill` checks and the patient's Yes.
- **Staff on the patient's behalf** never get another trust's answers.
- **Audit:** the `prefill.read` row is written under the source trust, and trust A's own audit rows hold only trust A's IDs.
- **The consent route** has its five integration cases.

### Findings, worst first

1. **Must fix: another trust's "by key" match copies option values without checking that the options mean the same thing.**
   - **Where:**
     - `apps/api/internal/prefill/prefill.go:194` (`find` returns "key" when `s.Name == q.Name`);
     - `fit` / `fitOne` / `fitChoice` with `byKey` true keep the stored value as it is.
   - **Why it's wrong:**
     - Within one trust, copying by value is safe because S23's lock freezes a keyed question's options (values *and wording*) across the trust's HQs.
     - That lock is **per trust**. Nothing makes trust B's `smoker` options mean the same as trust A's.
     - answers.md assumes keys cross trusts "only through the platform library". That's not true here: the seeded Demo pre-op HQ in each trust shares every key. The same would happen when two trusts import the same incumbent HQ, or import different incumbent versions with the same IDs, or when one trust edits a library question's options before its first publish.
     - In the local database, **6 of the 7** answers copied from Trust B on T-07's episode matched **by key**. Only one matched by wording.
   - **Reproduced with a throwaway unit test** (deleted afterwards):
     - At trust B, `smoking` has options `o1` = "Current smoker" and `o2` = "Never". At trust A, the same key has `o1` = "Never" and `o2` = "Current smoker".
     - The patient answered "Current smoker" (`o1`) at trust B.
     - Trust A's form got `{"smoking":"o1"}`, which reads as **"Never"**, and the clinician's marker said "(matched by link)".
   - **Suggested fix (smallest):** for a source with `OtherTrust`, fit a key match's choices **by option text**, as a wording match already does: pass `byKey = by == "key" && !f.otherTrust` into `fit`. The question still matches by key, and the clinician still sees "matched by link". The same applies to:
     - grids (rows and columns by text);
     - repeating lists (columns by text);
     - Yes/No labels;
     - a rating's options.
   - **Tests:**
     - a unit test like the one above for each choice-like kind (single, multiple, Yes/No, grid, list column);
     - an integration case in `TestPrefillCrossTrust` with trust B's HQ relabelling an option under a shared key.
   - **Hazard log note for C10:** cross-trust key matches don't get the publish lock's guarantee.

2. **Must fix: trust B's form ID is stored in trust A's rows and sent to trust A's staff.** This goes beyond what answers.md Q1 allows.
   - **Where:**
     - `prefill.Entry.FromForm` (`apps/api/internal/prefill/prefill.go:70`, set at `:132`) is trust B's `episode_forms.id` for another trust's answer;
     - it's stored in trust A's `episode_answers.prefill`;
     - `ListEpisodeAnswers` returns it to trust A's staff.
   - **Seen in the browser:** as Ada (Trust A) on T-07's episode, `GET …/episodes/2e8e59ec…/hq` returns `fromForm: "4ecd0ce8-…"` on all 7 of Trust B's entries. In the database, that ID is a Trust B form (org `…00fb`).
   - **Why it matters:** answers.md Q1 lets trust A's clinician see the source's questionnaire name, date and hospital, and how it matched. A record ID of trust B's isn't on that list.
     - Nothing reads `fromForm` (Go or web), so it isn't needed.
     - The builder's own check, "no trust B record ids in any trust A row", covers audit rows only.
     - Trust B's own `prefill.read` row already records the source episode, which is enough for an investigation.
   - **Suggested fix:**
     - leave `fromForm` empty (`omitempty`) for another trust's entry;
     - assert in `TestPrefillCrossTrust` that trust A's stored `prefill` and the staff `…/hq` response contain no trust B ID (the happy path currently asserts the opposite: `smokes.FromForm != x.inB.FormID` at line 157, and again at line 209);
     - `TestOtherTrusts` (`prefill_test.go`) should expect it empty too.

3. **Should fix (low): one trust's unreadable settings break the patient's form at every other trust.**
   - **Where:** `crossTrust` (`apps/api/internal/httpapi/prefill.go`, line 130, the `settings.Decode` error) returns the error, so `writeForm` sends a 500.
   - **Why it matters:** `crossTrustOn` says "Unreadable is off". Settings are validated on write, so this is unlikely. If it happened, though, one trust's bad JSON would stop the patient opening their HQ at another trust.
   - **Suggested fix:** skip that trust (log it) instead of failing.

4. **Should fix (low): the consent card hides its own error.**
   - **Where:** `apps/web/src/features/patient/PatientHQ.tsx`, `answerConsent`. It calls `setError(...)`, then always runs `load()`, and `load()` clears the error on success.
   - **What the patient sees:** after a 500, the same card comes back with no message, and pressing Yes again looks like nothing happened.
   - **Suggested fix:** set the error after the reload, or don't clear it on this path.

**Notes** (not required for this session; pending.md is enough):
- **A list row holding only stale text refuses the whole list.** `fitRows` (`prefill.go`, around line 462) counts any non-empty key as "answered", including a leftover `<col>-Comment` that isn't copied. Such a row then refuses the whole list, in the same trust too. It's on the safe side but stricter than S25. Counting only the cells that would be copied would fix it.
- **A row that loses only some cells still drops them without a word** (S25's rule: "columns the new one lacks are left out"). This matters more across trusts.
- **Reads repeat on every open.** After a Yes, every open re-reads each agreed trust and writes a new `prefill.read` row under it, for as long as some patient set has no row. The builder recorded this as accepted. Trust B's log will grow with page loads.
- **ValidateSet's count is fixed at load.** The "N answers here were pre-filled… changed M" line is counted once when the page loads, while the markers redraw. A clinician's change that shows or hides a pre-filled question leaves the count stale.
- **Trusts are asked about in an arbitrary order.** With several other trusts, `ListPrefillOffers`' `DISTINCT ON (e.org_id)` asks about them in UUID order, not the newest first.

### The builder's own choices, against Rahul's answers

- **"While a consent question is open, nothing is pre-filled, not even from the patient's own trust"** compared with answer Q3 ("Yes fills … after same-trust pre-fill (which needs no agreement)"): **consistent; no question for Rahul.**
  - Q3 also puts the question *before the first Question Set*, so the patient can't open a set while it's open.
  - When it's answered, the next open fills from every source at once, with the form's own trust first (`Fill` sorts own-trust forms first, and an own-trust form that asked decides).
  - No answers is equivalent to "No": same-trust pre-fill still happens on the next open.
  - The only difference from a literal "fill same-trust first, then merge" is an edge case: sets that already have a row (staff opened the form first, or the patient typed before a setting was switched on) get nothing from the other trust. That follows "if in doubt, don't fill", and the build notes record it.
- **The consent is per account and source trust, so a Yes given at trust A also lets trust C use trust B's answers:** this is what Q3 says ("stored per account and source trust"), and the question names only the source. It's consistent. Rahul may still want to know.
- **The audit row names the source episode and record, not the form** (Q5 said "the source form ID"): acceptable. An episode has one HQ form today, and `audit_log` has no form column.
- **The question is asked even when nothing will match in the end** (deciding to ask reads no answers): this is the right way round for privacy.

### Test results (verifier's own runs)

- `make lint`: passed.
- `make test`:
  - Go: all ok (`internal/repocheck` ok);
  - Vitest: 219 web and 150 clinical passed;
  - CDK: 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (on `make dev`): 62 passed, 5 skipped, 2 failed. The two failures are the known flakes, S11's audit row check and S20's stale-tab check, and both pass when run alone (2 passed). J16 and J16b passed. Afterwards `crossTrustPrefill` is off in Trust A and Trust B; J16b's fixture leaves an explicit `false` rather than clearing the override, which has the same effect.
- A-20 grep: only comments (in `format.ts` and `patient.go`).
- Browser spot-checks (Claude in Chrome, "rahul chrome"; read-only, so no setting was changed):
  - As Ada (Trust A), T-07's episode: the HQ response holds Trust B's form ID (finding 2).
  - Trust A's audit log names no Trust B record.
  - Finding 1 was reproduced with a unit test instead, as above.

### Hand checks for Rahul

None.

### Next step

Verification found problems: read `docs/sessions/S26-prefill-clinician-cross-trust/verify.md`, fix them test first, then ask for a retest.
