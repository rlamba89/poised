# S26 build notes: pre-fill for the clinician, and answers from other trusts

Builder, automatic run (`/run-sessions S23 S26`), 23 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md) (22 Oct). They match the brief's recommendations, plus one addition:
- **Q1:** the clinician sees, on each pre-filled answer, its questionnaire and date, the hospital when it's another trust's, **and whether it matched by link or by wording**; "Changed by patient" with the pre-filled answer; and which button the patient pressed per set.
- **Q2:** both trusts' `crossTrustPrefill` must be on. Switching either off stops new offers, and answers already copied stay.
- **Q3:** the patient is asked once per source trust, before the first Question Set ("Use the answers you gave St Mary's in March 2026?"), and the answer is stored in `patient_prefill_consents`. Yes fills only what's still empty, after same-trust pre-fill. It can't be changed.
- **Q4:** on the patient's behalf, same-trust answers only and no question, even after a Yes.
- **Q5:** reading another trust's answers writes an audit row under that trust too.

The conductor added the three "For S26" items from S25's verification ([pending.md](../../pending.md)) to the scope. All three are done, test-first (below).

## What was built

### Go: `internal/prefill` (unit-tested)
- **Other trusts' forms come after all of the form's own trust's.** `Source.OtherTrust` marks them. `Fill` sorts the own trust's forms first (newest first), then the other trusts' (newest first). "The newest form that asked decides" (S25) therefore means: if any of the own trust's forms asked the question, it decides, even with no answer; otherwise the newest other-trust form that asked decides. This is answers.md Q3's "only questions still unanswered … after same-trust pre-fill", read as "if in doubt, don't fill" (an own-trust form that asked and got no answer, e.g. the question was hidden then, isn't overridden by another trust's).
- **`Entry` stores `questionnaire`** (always), plus **`otherTrust` and `hospital`** for another trust's answer. The clinician's marker reads them from the map, so **trust A's staff never read trust B's records** to label an answer. Rows that S25 filled have no `questionnaire`; the marker then says "an earlier HQ".
- **A repeating-list row whose only answers are in columns the new list lacks refuses the whole answer** (S25 verify, item 3). Before, such a row was dropped silently. This applies in every trust; within one trust it can't arise (S23's lock). Rows that were empty to begin with are still skipped.

### Go: storage
**Migration 00026** (`00026_prefill_consents.sql`):
- **`patient_prefill_consents`:** `(account_id, org_id)` is the primary key (one answer per account and source trust), with `agreed`, `decided_at` and `episode_id` (where it was asked). Closing the account deletes it (`ON DELETE CASCADE`; S22 deletes the account).
- **A pre-filled row starts at revision 1, not 0** (S25 verify, item 1). A browser sends 0 for "no row yet", so a tab that loaded a set before it was pre-filled now gets the stale 409 instead of overwriting it. Without this, the `prefill` map would describe answers no longer in the row. Existing pre-filled rows move up by one. The answers trigger is switched off for that update, so it doesn't count as episode activity (S22's auto-archive).
- **`episode_answers.started` is a generated column:** saved past the revision the row was created at (0 for a row the patient made, 1 for a pre-filled one), or a card answered. `ListHome` (accounts.sql) and `GetFollowup` (jobs.sql) now use it, so there's one definition instead of two copies (S25 verify, item 5).

### Go: routes and queries
- **`ListPrefillOffers`** (new) lists the other trusts that could pre-fill, with no answers: each **active** trust other than the form's own, where a record joined to the person's account has a sent HQ. For each it returns the trust's settings, the newest sent HQ's hospital and date (for the question), and the person's answer so far. Only the person's own records are included.
- **`ListPrefillSources`** (S25) also returns the questionnaire's name, the hospital's id and its name. It still takes exactly one `org_id`.
- **`crossTrust`** (prefill.go) applies Q2 and Q3, for the patient's own form only:
  - nothing at all unless the form's own trust has `crossTrustPrefill` on (`hospitalSettings`, a trust-only setting);
  - then each offered trust whose own setting is on (`settings.Effective`): not yet asked (a question), Yes (a source) or No (skipped).
- **`prefillForm`** (S25) now runs in this order:
  1. **For the patient only**, any question still to be answered stops the fill: nothing is filled, and the GET returns `prefillConsent: [{orgId, hospital, submittedAt}]`.
  2. Otherwise it reads the form's own trust's sources, then **each agreed trust with its own `ListPrefillSources` call (one trust per query)**, and fills once from all of them.
  3. Another trust's episodes go in a **`prefill.read` audit row under that trust**, at the source episode's hospital (actor: the patient). They never go in the form's own trust's row.
  4. **On the patient's behalf (staff)**, `crossTrust` isn't called at all.
- **`POST /api/p/episodes/{eid}/prefill-consent`** `{orgId, agree}` is a patient session only, audited as `link.prefill_consent` under the form's trust. It returns:
  - **204**: recorded;
  - **400**: unreadable, an `orgId` that isn't a uuid, or no `agree`;
  - **404**: not the person's episode;
  - **409**: the HQ was sent; the trust isn't being asked about (its own trust, a trust with no answers, a setting switched off, or already answered); or answered at the same moment in another tab.
- **`ListEpisodeAnswers`** also returns `prefillChoiceByName` (staff who pressed the card's button with the patient), for the clinician's line.
- The `crossTrustPrefill` help text now says both trusts must have it on, and that it's never used when staff fill in for the patient.

### TypeScript (for Rahul to review)
- **`packages/clinical/src/prefill.ts`** (new, Vitest `prefill.test.ts`). It holds the shared types (`PrefillEntry`, `SetPrefill`, moved here from the web's patient code) and the clinician's rules:
  - `prefillMark(question, entries, patientAnswers)`: one question's marker, or none. Clinician-only questions are never marked. **`changed`** is true when the patient's final answer (or its "Other" text) differs from the pre-filled value, including an answer they cleared. It also gives `was` and `now` in words (`describeAnswer`, as corrections use).
  - `prefillMarks(model, …)`: the markers for **the questions the model shows** (`getAllQuestions(true)`), so a pre-filled answer its condition hides isn't marked (S25 verify, item 4).
  - `prefillSourceText`: "Pre-filled from Demo pre-op HQ, 3 Mar 2026 (matched by link)". Another trust's: "… from Demo pre-op HQ at Hospital B1 (another trust), … (matched by same wording)".
  - `prefillChoiceText`: "The patient chose “Everything is still correct” on …, so these pre-filled answers weren't checked one by one.", "The patient chose “Check each answer” on …", or "Cara Clinician chose … with the patient on …".
- **`poa.ts`** `poaSummary(…, original, prefill)`:
  - each page gets `prefilled` (the markers above, compared with the patient's answers);
  - a page that has only pre-filled answers is kept;
  - each set carries its `prefill` for the button line;
  - only sets shown (`shownSets`), as before.
- **`apps/web`:**
  - `features/episodes/hq.ts` builds `hq.prefill` from the patient rows.
  - `features/episodes/prefillOrigins.ts` adds the marker under each question SurveyJS draws in **ValidateSet**. Drawn means shown, so only shown questions are marked. Under each it puts the source line and, where changed, "Changed by patient: pre-filled “Yes”" in orange.
  - **ValidateSet** shows a line above the set: "6 answers here were pre-filled from the patient's earlier questionnaires, and the patient changed 1. The patient chose …".
  - **PoaSummary** shows, per page, a "Pre-filled on <page>" box listing each answer's source, with a **Changed by patient** badge and "pre-filled “Yes”, patient answered “No”". Under the set, it shows the button line. Both tabs.
  - **PatientHQ**: when the GET sends `prefillConsent`, a card replaces the list of sets. It reads "Use your earlier answers? Use the answers you gave Hospital B1 in October 2026?", with a short explanation, **Yes, use them** and **No, thanks**. It posts the answer, then loads the HQ again (filled, or not). It asks about one trust at a time.
  - `lib/format.ts` `formatMonth` gives "October 2026" in the deployment's locale.

## Decided by the builder (automatic mode)
- **While a consent question is open, nothing is pre-filled for the patient, not even from their own trust.** The question comes before the list (Q3), so the patient can't open a set meanwhile. The next open then fills every set from all its sources at once. That keeps "nothing typed is overwritten" simple (fills still only go into sets with no row), and no second merge into existing rows is needed.
  - **The cost:** if staff opened the form on the patient's behalf first (same trust only), or the patient started typing before a setting was switched on, those sets already have rows and get nothing from the other trust. "If in doubt, don't fill."
- **The question is shown only if some patient set has no answers yet**, the form isn't sent, and the other trust has a sent HQ for the person's account. **Deciding to ask reads no answers.** It reads only that trust's settings, the newest sent HQ's hospital and date, and the person's earlier answer to the question. So the patient may say Yes and then find nothing matched. We accept that over reading another trust's answers before the patient agrees.
- **A suspended trust's answers are never offered** (`o.status = 'active'`), like its questionnaires on the patient home (R-6).
- **Every open that reads another trust's answers writes a `prefill.read` row under it**, including when nothing matched in the end. While some set still has no row (a set with nothing to fill gets none, S25), each open reads again and writes again. That's accurate, but repetitive.
- **The audit row names the source episode** (`episode_ids`) and the patient's record at that trust (`patient_ids`), not the form id. `audit_log` has no form column, and an episode has one HQ form today (C7 adds more). This is how answers.md Q5's "the source form ID" is recorded. If Rahul wants the form id itself, it needs a new column.
- **The consent is per account and source trust** (Q3), not per pair of trusts. A Yes to Trust B's answers given at Trust A also lets Trust C use them, if both C and B have the setting on. The question names Trust B's hospital and month, not where it's asked.
- **409 for a trust that isn't being asked about**, rather than 400, because it's usually the state that changed (answered in another tab, a setting switched off). The page shows the message and loads again.
- **A row whose only non-empty cells are in a clinician-only column also refuses the list.** Such columns are never copied, so a list where the patient left a row empty but the clinician filled its check isn't pre-filled. This is rare, and on the safe side.

## Not built (as the brief says)
Pre-fill across countries; changing or withdrawing a consent decision; the hazard log entry itself (OPS-04, C10).

**What the hazard log (OPS-04) should say about pre-fill**, for C10:
- **Hazard:** a wrong or out-of-date earlier answer is copied into a new HQ, and the patient confirms it without reading it. Example: "Everything is still correct" over answers from another trust's form a year ago.
- **Controls in place:**
  - **matching:** by key or exact wording only, with every safeguard ("if in doubt, don't fill"); the newest form that asked decides, and the own trust's forms come before other trusts'; Always ask fresh on questions that change (weight, pregnancy); clinician-only questions are never copied;
  - **for the patient:** the opening card and a marker on every pre-filled answer;
  - **for the clinician:** where each answer came from, how it matched and whether the patient changed it, plus the button the patient pressed (recorded with its time and who pressed it);
  - **across trusts:** both trusts' setting and the patient's explicit Yes, never on their behalf, audited under both trusts.
- **Residual risk:** a patient who presses "Everything is still correct" without reading. The clinician is told this happened, per set.

## Tests
- **Go unit (`internal/prefill`):**
  - `TestOtherTrusts`:
    - the own trust decides when one of its forms asked, even when another trust's form is newer;
    - an own-trust form that asked with no answer means the question isn't filled from another trust;
    - several other trusts: the newest decides;
    - the JSON fields.
  - `TestEmptiedRowRefusesTheList`.
- **Integration (`internal/httpapi/prefill_cross_trust_integration_test.go`):**
  - `TestPrefillCrossTrust`:
    - offered only with both settings on (all four combinations), and nothing filled or audited under trust B before the answer;
    - **Yes:** filled by key and by wording, marked as another trust's with its hospital and questionnaire, revision 1, the consent row, a `prefill.read` row under trust B at its hospital, and no trust B record ids in any trust A row;
    - a second open reads trust B no more; another episode at trust A isn't asked again;
    - switching trust B off stops new offers, and copied answers stay;
    - **No, thanks:** remembered, not asked again at another episode, no `prefill.read` rows, can't be changed (409);
    - **own trust first, nothing typed overwritten:** a set typed in before the answer keeps its answers, and smoking comes from trust A's own form, not trust B's;
    - **on behalf:** never trust B's answers, even after a Yes;
    - **a suspended trust:** not offered.
  - `TestPrefillConsentRoute`, the five cases plus more:
    - no sign-in: 401;
    - wrong role (staff, no patient session): 401;
    - another patient's episode, and the person naming an episode that isn't theirs: 404;
    - invalid input: 400 for a bad body, a bad `orgId` or no answer; 409 for the form's own trust or an unknown trust;
    - happy path: recorded, audited; asking again with either answer is a 409;
    - once sent: 409.
  - `TestPrefillRevision` (S25 verify, item 1): a stale tab's revision-0 save gets the 409, the pre-filled answers stay, the first save makes revision 2, and "started" only then.
  - S25's `TestPrefill` and `TestPrefillChoice` now expect revision 1 for pre-filled rows.
  - **Mutation checks** (run by hand, then reverted): the test fails if any of these is removed:
    - the other trust's setting check;
    - the own trust's setting check;
    - the "not on the patient's behalf" check;
    - the separate audit row for the other trust.
- **Migration (`internal/apitest`):** `TestPrefillConsentsMigration` checks that:
  - revisions move up by one;
  - `started` is right for each case;
  - the last activity isn't moved;
  - one consent per account and trust;
  - closing the account deletes its consents;
  - Down puts the revisions back.
- **Vitest:**
  - `packages/clinical/src/prefill.test.ts` (8): markers for shown questions only, "changed" (including "Other" text and a cleared answer), the texts, and `poaSummary` with pre-fill (only shown sets, both tabs, a page with only pre-filled answers kept);
  - `apps/web/src/lib/format.test.ts`: `formatMonth`.
- **E2E:**
  - `test_j16_prefill.py` is now J16 completed. After the patient sends, Cara's Validate page shows:
    - the set line (6 pre-filled, 1 changed, "Everything is still correct");
    - the diabetes marker "(matched by same wording)" and the snoring marker "(matched by link)";
    - one "Changed by patient: pre-filled “Yes”";
    - no marker on pregnancy.

    The POA Summary shows the same.
  - `test_j16b_prefill_cross_trust.py` (new, phone size). Ada and Max switch the setting on, and the fixture puts it back:
    - **Yes:** "Use the answers you gave Hospital B1 in <month>?", then both sets are Please check, with the card and markers; not asked after a reload; the sources name Hospital B1 as another trust's;
    - **No, thanks:** nothing filled, and not asked again.

## Results
- `make lint`: passed.
- `make test`: Go all ok; Vitest 219 (web) + 150 (clinical); CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` on my own instance (database `sj_s26`, API :8096, Vite :3106): **64 passed, 5 skipped, 0 failed**.
- **The local `sj` database is still at migration 00025.** The conductor's restart (`make migrate seed`) takes it to 00026. No reset is needed, and HJE Full HQ is untouched.

## Suggested commits
1. `S26: pre-fill from other trusts (both trusts' setting and the patient's Yes, once per trust: patient_prefill_consents, POST …/prefill-consent; one trust per query; prefill.read audited under the source trust; never on the patient's behalf), the clinician's pre-fill markers in Validate and the POA Summary (PRE-04), pre-filled rows start at revision 1 and "started" is one generated column (migration 00026), an emptied list row refuses the answer`, with its tests:
   - Go unit tests (`internal/prefill`);
   - integration tests (`prefill_cross_trust_integration_test.go`; `prefill_integration_test.go` updated; `TestPrefillConsentsMigration`);
   - Vitest (`packages/clinical/src/prefill.test.ts`, `format.test.ts`);
   - e2e (`test_j16_prefill.py` completed, `test_j16b_prefill_cross_trust.py`, README rows).
2. `docs: S26 …` (build notes, test plan and reports, decisions, pending, CLAUDE.md, AGENTS.md, plan C4 and session status) once verified.

## Test report 1 (23 Oct): 20 of 20 passed; exploring notes

1. **A false "Patient answered: no answer" under a list's column in Validate:** older than S26 (it's on an episode with no pre-fill too). Recorded in pending.md, not changed here.
2. **A bad `orgId` got "Choose “Yes, use them” or “No, thanks”."** **Fixed, test first:** it now gets "We couldn't tell which answers this is about. Please reload the page."; a missing answer keeps the old message (`TestPrefillConsentRoute` checks both).
3. **Answers copied from Trust B become Trust A's own once sent,** and later fill Trust A HQs marked only as Trust A's, even after Trust B switched off. This is how answers.md Q2 says it should be ("answers already copied stay"), but the clinician loses where the value first came from. **Recorded in pending.md for Rahul** (options: carry the original source through, or a hazard-log line); not changed.
4. **Trust B's audit said its answers were "used to pre-fill"** when Trust A's own answers had covered every question, so the log claimed a use that didn't happen. In healthcare an audit row must say what happened. **Fixed, test first:** the label is now "Patient's answers **read** to pre-fill their HQ at another trust (they agreed)". The row is still written on every read, as decided (`TestPrefillCrossTrust` checks the label Trust B's admin sees).
5. **The question is asked even when nothing will be used:** as decided (deciding to ask reads no answers).

**Results after the fixes:** `make lint`, `make test` and `make test-integration` pass. The two fixes change one label and one message and touch no page, so e2e wasn't re-run; J16b doesn't check either text. **T-12 and T-13 need a retest:** the test plan names the new texts.

## Test report 2 (23 Oct): all passed
- **8 of 8:** T-12 (the "read to pre-fill" label under Trust B) and T-13 (the bad `orgId` message) are fixed, with no regressions.
- **Exploring:** round 1's false "Patient answered: no answer" under a list column is still there. It's older than S26 and already in pending.md. Refused consent requests are audited with their status, as other refused requests are. Nothing new to fix.
- **S26 is tested.** Plan C4 becomes **Done** once S26 is verified. The plan's Status and the plans table say so.

## Hand checks for Rahul
None. Every case was run in the browser or with `curl`.

## Suggested commits (final)
1. `S26: pre-fill from other trusts (both trusts' setting and the patient's Yes, once per trust: patient_prefill_consents, POST …/prefill-consent; one trust per query; prefill.read audited under the source trust; never on the patient's behalf), the clinician's pre-fill markers in Validate and the POA Summary (PRE-04), pre-filled rows start at revision 1 and "started" is one generated column (migration 00026), an emptied list row refuses the answer`, with its tests:
   - Go unit tests (`internal/prefill`);
   - integration tests (`prefill_cross_trust_integration_test.go`, `prefill_integration_test.go`, `TestPrefillConsentsMigration`);
   - Vitest (`packages/clinical/src/prefill.test.ts`, `format.test.ts`);
   - e2e (`test_j16_prefill.py`, `test_j16b_prefill_cross_trust.py`, README rows).
2. `docs: S26 tested — build notes, test plan and reports, decisions, pending, CLAUDE.md, AGENTS.md, plan C4 and session status`.

## Verification 1 (23 Oct): fixes

All test first.

1. **Another trust's key match copied option values without checking their wording.** This is clinical safety. S23's lock freezes a keyed question's options only **within one trust**. Trust B's `smoker` with `o1` = "Current smoker" became trust A's `o1` = "Never", and the marker said "(matched by link)".
   - **Fix:** another trust's forms now match **by wording only, never by key** (`earlierForm.find` skips the key loop for `otherTrust`). That brings the wording match's safeguards:
     - the same question text, kind and option texts;
     - never a text used twice in either questionnaire;
     - every choice mapped by its option's **text**, for single and multiple choice, Yes/No labels, a grid's rows and columns, and a list's columns and their options.
   - **The marker says what happened:** every other-trust answer is "matched by same wording".
   - This is stricter than the verifier's smallest fix (key match, choices by text). "If in doubt, don't fill": across trusts the same key is only a coincidence of naming (imports, library copies edited before publishing), and answers.md expected most cross-trust matches to be by wording anyway. **For Rahul:** a library question one trust rewords no longer pre-fills across trusts.
   - **Tests:**
     - `TestOtherTrustsByWordingOnly`: swapped values, a reworded option, a reworded question, a text used twice, multiple choice, Yes/No labels the other way round, a grid, a list column's options;
     - `TestPrefillCrossTrust` now uses a trust B HQ whose `q_smokes` values mean the opposite (`crosstrust-b.json`): trust B's "Yes" (`o_no` there) arrives as trust A's "Yes" (`o_yes`);
     - J16b asserts "wording".
2. **Trust B's form id was stored in trust A's `prefill` map and sent to trust A's staff.**
   - **Fix:** `Entry.FromForm` is empty (omitted) for another trust's answer. Trust A keeps only the questionnaire's name, the hospital, the date and how it matched (answers.md Q1). Trust B's own `prefill.read` audit row records the source episode.
   - **Tests:** `TestPrefillCrossTrust` checks that none of trust B's ids (form, episode, record, trust, hospital, questionnaire, version) appears in trust A's stored answers or in the staff `…/hq` response; `TestOtherTrusts` too.
   - **Other leaks checked:**
     - data keys and values are trust A's own names and values;
     - trust A's audit rows hold only trust A's ids (tested);
     - `prefillConsent`'s `orgId` goes only to the patient, never to staff;
     - `patient_prefill_consents` isn't read by any staff route.
   - **Local test data:** the copies made before the fix in the local `sj` and `sj_s26` databases had `fromForm` stripped with one `UPDATE` each (8 rows each, trigger off). Their old "key" labels stay as they were.
3. **One trust's unreadable settings 500'd the patient's form at another trust.** Fixed: that trust is skipped and logged (`crossTrust`). Test: the "unreadable settings" case of `TestPrefillCrossTrust`.
4. **The consent card hid its own error:** `load()` cleared it. Fixed: the message is set after the reload. Test: J16b's `test_a_refused_answer_keeps_its_message` (it failed before the fix).
5. **From the notes:**
   - **fixed:** a stale `-Comment` alone in a list row no longer counts as an answer that refuses the list (`TestStaleCommentInARow`);
   - **fixed:** several other trusts are asked about newest first;
   - **recorded in pending.md:** cells dropped from a row without a word, the repeated reads, and ValidateSet's count being fixed at load.

**Hazard log (OPS-04), added:** cross-trust matches are by wording only, because keys aren't held to one lock across trusts.

**Results after the fixes:**
- `make lint`, `make test` and `make test-integration` pass.
- `make e2e` on my instance (`sj_s26`, :8096/:3106): **64 passed, 1 failed, 5 skipped**. The failure was `test_audit.py`'s tooltip check, the known S11 flake; the file passed alone (5 of 5). J16 and J16b passed, 5 of 5, including the new refused-answer test.
- **`internal/repocheck` failed on verify.md**: line 24 named the incumbent (twice). I changed it to "incumbent" (decisions.md §1: never name it in the repo); nothing else in verify.md changed.
- **Retest:** T-07 (the Trust B markers now say "matched by same wording"; no `fromForm` in the staff API). The rest is unchanged.

## Rahul's answer after verification 1 (23 Oct): platform library keys plus wording

**Rule:** for another trust's answers, a key counts only if it's a **platform library key** (`library_questions` with `org_id` NULL). Even then every choice is mapped by its option's text, and a text that doesn't match means no fill. Everything else matches by exact wording with all the safeguards, as built after verification 1.

**Built test first:**
- **`prefill.Fill` takes the platform library's keys.** `find` matches another trust's question by key only for one of them, and `fit` maps another trust's choices by text whatever the match.
- **`prefill.LibraryKeys`** collects each platform library element's key and its children's. The "yes/no + details" groups keep their children's keys (S24), so `anaesthetic_problems` counts.
- **The new query `ListPlatformLibraryElements`** is read only when another trust's answers are used.
- **The marker** says "matched by link" for a library key and "matched by same wording" otherwise.

**Tests:**
- **Unit, `TestOtherTrustsPlatformLibraryKeys`:**
  - a library key reworded at one trust still matches by link, with its choice mapped by text;
  - a library key whose chosen option's text isn't in the new question isn't filled;
  - a trust's own colliding key never matches by key;
  - the form's own trust is as before.
- **Unit, `TestLibraryKeys`.**
- **Integration (`TestPrefillCrossTrust`):**
  - trust B's `q_smokes` (its own key, worded "Do you smoke now?", with its values swapped) isn't filled;
  - with a platform library row for `q_smokes`, it's filled as trust A's "Yes" (`o_yes`), matched by key.
- **J16b:** snoring (a starter-library key) is "key"; diabetes stays "wording".

**Results:**
- `make lint`, `make test` and `make test-integration` pass.
- J16 and J16b pass on my instance, 5 of 5.
- `make e2e` on my instance (`sj_s26`, :8096/:3106): **65 passed, 5 skipped, 0 failed**.
- **Retest:** T-07, as the test plan's note at the top describes.

## Test report 3 (23 Oct): all passed
- **14 of 14:**
  - T-07 with the library-key rule (library questions "matched by link", diabetes "matched by same wording", no `fromForm`);
  - the consent error staying on screen;
  - no regressions.
- **Exploring, nothing new to fix:**
  - the false "Patient answered: no answer" under a list column is older than S26 and in pending.md;
  - Trust B's "read … (they agreed)" row when Trust A's own answers covered everything is expected (every read is logged; the label says "read");
  - two tabs on the consent card behave.
- **S26 is tested again**, for verification 2. Plan C4 becomes **Done** once S26 is verified.

## Suggested commits (final)
1. `S26: pre-fill from other trusts`, with its tests:
   - **What it does:**
     - both trusts' setting and the patient's Yes, once per trust (`patient_prefill_consents`, `POST …/prefill-consent`);
     - another trust's question matches by key only for a platform library key, else by exact wording, its choices always by option text;
     - one trust per query, `prefill.read` audited under the source trust, and no trust B id kept;
     - never on the patient's behalf;
     - the clinician's pre-fill markers in Validate and the POA Summary (PRE-04);
     - pre-filled rows start at revision 1, and "started" is one generated column (migration 00026);
     - an emptied list row refuses the answer.
   - **Its tests:**
     - Go unit tests (`internal/prefill`);
     - integration tests (`prefill_cross_trust_integration_test.go`, fixture `crosstrust-b.json`, `prefill_integration_test.go`, `TestPrefillConsentsMigration`);
     - Vitest (`packages/clinical/src/prefill.test.ts`, `format.test.ts`);
     - e2e (`test_j16_prefill.py`, `test_j16b_prefill_cross_trust.py`, README rows).
2. `docs: S26 verified — build notes, test plan and reports, verification, decisions, pending, CLAUDE.md, AGENTS.md, plan C4 done and session status`.
