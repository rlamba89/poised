# S29 verification

## Verification 1 (25 Oct 2026, verifier, automatic run)

**Verdict: Verified.**

Everything in the brief is built, with tests. Test report 2 passed 17 of 17. No finding blocks the session. The findings below are small follow-ups, now in [pending.md](../../pending.md) under "From S29's verification 1".

### What was checked
- **The diff against the brief:** all six items are built:
  - the reported BMI uses only shown answers;
  - each opening-card button has its own audit action;
  - network errors have plain wording, with no banner over the cleared card;
  - archived episodes show no open-task count;
  - an archived-from-Cancelled form says cancelled;
  - "Same as…" warns before replacing a link.

  The later-join read (item 1) is left out, as the answers say. Nothing big was added that the brief didn't ask for.
- **Tests first:**
  - **Integration tests are new or changed for each Go change:** `TestPrefillChoiceAuditActions`, `TestArchivedFromCancelledFormSaysCancelled`, `TestSameAsSearch`'s `linkedTo` cases, `TestPrefillChoice` and `TestStartFresh`.
  - **No new routes.** The changed routes keep their five cases.
  - **Unit tests:** `TestLinkedTo`, `reportedBmi` (`poa.test.ts`), `api.test.ts` and `openTaskCount`.
  - **Journeys:** J5's fourth test and the audit journey's new test.
- **Tenant isolation:**
  - No new query. `GetEpisodeForm` only adds `e.archived_from`.
  - `LinkedTo` reads the same trust-scoped sources as the candidates, and a test shows another trust's library key gives `""`.
  - A spot-check showed that Trust B's admin can't see Trust A's new choice rows.
- **A-20:** the grep finds UK literals only in `infra/countries.py`, comments in `format.ts` and `patient.go`, and tests. No new literal.
- **Security:**
  - no secrets;
  - audit rows hold IDs only (the action name is the button's key);
  - the patient router is unchanged;
  - `chapter.ForPatient` is untouched.
- **Docs:**
  - decisions.md §2 has S29's entry, and item 1 is shown as reversed;
  - pending.md and sessions/README.md are updated;
  - `apps/web/AGENTS.md` is updated;
  - repocheck passes.
  - S29 has no plan of its own, so there is no plan status to update.

### Findings (none blocking, worst first)
1. **Gateway errors on AWS won't get the plain message** (`apps/web/src/lib/api.ts:34`). The plain message is given only for a 5xx with **no** JSON body. API Gateway's own 5xx replies (a Lambda timeout or throttle) are JSON, such as `{"message":"Service Unavailable"}`, with no `error` field. So on AWS they would still show "Something went wrong. Please try again." The builder followed the brief as written ("no JSON body"), and the local dev proxy's empty 502 works. A simple fix: use the plain message for any 5xx whose body has no `error`.
2. **A Reload from the "answers were cleared" card that fails for a reason other than the network shows nothing new** (`apps/web/src/features/patient/PatientHQ.tsx:277`). While the card shows, every error banner is hidden. For example, if the episode was cancelled in the meantime, the 403's message is hidden and the card's text stays the same. This is rare: it needs a Start fresh, a failed reload, then another refusal. It could be fixed by hiding only `NETWORK_ERROR` there.
3. **"Same as…" warns on almost every question carried into a later draft** (`apps/api/internal/keys/keys.go:861`):
   - `LinkedTo` counts the questionnaire's own earlier published versions;
   - so a v2 question shows "already linked to '<its own v1 wording>'".

   This is a deliberate choice, recorded in decisions.md: replacing the key does change the link to v1's answers. But it can read oddly. Rahul may want it only for keys from the library or from another HQ.
4. **Each `reportedBmi` call builds a SurveyJS model for every shown set, even sets with no BMI, height or weight question** (`packages/clinical/src/poa.ts:126`):
   - the "shown" check uses the clinician's view, which is the builder's recorded choice;
   - so a height that a condition shows only to patients would be skipped (no demo HQ does this).

   Both points are minor.
5. **A refused request still gets the old shared action and label.** A request refused before the choice is read (unknown set, bad JSON, bad choice, archived or cancelled form) gets `behalf.prefill` / `link.prefill`, "Checked pre-filled answers…". This is recorded in decisions.md, but the label can mislead on a refused row. One option is a neutral label such as "Pressed a pre-filled answers button" for new rows.
6. **A second press in another tab writes a 204 row naming the button**, though it changed nothing (the first choice stays). This is the builder's recorded choice ("the row says what the person did"). It's noted here because the audit row and the stored choice can disagree.
7. **`EpisodeList.tsx:141` uses `openTaskCount` only as a truthiness check**, and then renders `e.openTasks` directly. This is a small tidy-up.
8. **J8 is flaky, and the flake is older than S29** (`e2e/journeys/test_j8_reminders.py:31`). It picks the mobile from only 1,000 numbers (`+447700900NNN`). On a long-lived local database, a number gets reused, and `captured(mobile)` then finds another run's texts. The full run failed this way once, then J8 passed 4 times in a row. Fix it with a wider random range, or by filtering the captured texts by episode or time.
9. **No "What" filter on the Audit page.** The brief mentions one, but the page has none. The builder recorded this for Rahul (pending.md), and it isn't needed to finish the session.

### Test results (25 Oct)
- `make lint`: clean.
- `make test`:
  - Go: all ok, including repocheck;
  - Vitest: web 240 passed (23 files), clinical 161 passed (19 files);
  - CDK: 124 passed.
- `make test-integration`: all packages ok.
- `make e2e` (against the conductor's `make dev`): **70 passed, 1 failed, 5 skipped**. The failure was J8, the mobile-number collision in finding 8. Re-run alone, J8 passed 4 of 4. The 5 skipped are the Cognito journeys, which run on QA only.

### Spot-checks
Claude in Chrome couldn't be used: the only connected browser was the one the conductor said never to use. So I ran the spot-checks as temporary Playwright journeys, using the repo's harness against `make dev`, with the patient on an iPhone 13 (WebKit, 390×844). The file was deleted afterwards. **All 3 passed:**
1. **Archived from Cancelled:**
   - The patient joined, Cara cancelled the episode and Sue archived it.
   - Signed in by email, the patient's home says "Cancelled by the hospital", and the form says "This questionnaire was cancelled by the hospital…" with no "closed" text and no sideways scroll.
   - Staff get the on-behalf form's 409.
   - Another patient's episode id shows "not found".
2. **Each card button's audit label:**
   - With Trust B's answers copied in, the patient pressed Everything is still correct on Your health (204), then Start fresh on About you (204).
   - Ada's log, filtered by surname, shows one "Patient confirmed pre-filled answers", one "Patient started a Question Set fresh" and no old "Patient checked pre-filled answers".
   - Trust B's admin (Max Multi), reading his trust's audit for that surname, sees none of these rows.
3. **Network error and the archived task count:**
   - On the patient's HQ at phone size, with the HQ request aborted, the page shows "We couldn't reach the server. Check your connection and try again.", with no "Failed to fetch" or "Load failed".
   - An archived episode with an open Nurse task shows no "open task" line in the archived list and no "(N open)" on its Tasks title, and the task is still listed.

### Hand checks for Rahul
- **A real gateway error:** stop only the Go API while Vite runs, then use a staff page. It should say "We couldn't reach the server…". This is automated in J5 (a routed empty 502) and the `api` unit tests (from the build notes).

### Suggested commits (made locally, automatic mode)
1. `S29: Patient reported BMI from shown answers only`: `packages/clinical/src/poa.ts`, `poa.test.ts`, `PoaSummary.tsx`.
2. `S29: an audit action for each opening-card button`: `audit.go`, `prefill.go`, `prefill_integration_test.go`, `prefill_cross_trust_integration_test.go`, `e2e/journeys/test_audit.py`.
3. `S29: plain network-error wording, and no red banner over the cleared card`: `apps/web/src/lib/api.ts`, `api.test.ts`, `PatientHQ.tsx`, `e2e/journeys/test_j5_poa_summary.py`, `e2e/README.md`.
4. `S29: no open-task count on archived episodes`: `tasks.ts`, `tasks.test.ts`, `EpisodeList.tsx`, `TasksPanel.tsx`.
5. `S29: the cancelled message on an archived-from-Cancelled form`: `db/queries/episodes.sql`, `internal/db/episodes.sql.go`, `patient.go`, `archive_integration_test.go`.
6. `S29: "Same as…" warns before replacing a question's link`: `keys.go`, `keys_test.go`, `question_keys.go`, `library_integration_test.go`, `library.tsx`.
7. `docs: S29 verified`: this folder, `apps/web/AGENTS.md`, `docs/decisions.md`, `docs/pending.md` and `docs/sessions/README.md`.

**Next:** split plan C5 into sessions (decisions.md §5).
