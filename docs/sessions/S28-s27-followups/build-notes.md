# S28 build notes: the follow-ups after S27

Builder, automatic run (`/run-sessions`), 25 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md) (25 Oct), all as recommended:
1. **When the staff-side join happens:** when staff open **Fill in for the patient** (`behalf.view`), before the form is pre-filled.
2. **What it may do:** join **only this record**, **only to an account that already exists**. No new account, no `joinMatching`, no contacts copied either way. One `account.join` row in Trust A's log names the staff member.
3. **First open only:** the **whole** pre-fill runs once per form, on its first open (this trust's answers and other trusts'); the form records it.

## What was built

### Go
- **Migration 00028** (`00028_prefill_once.sql`):
  - `episode_forms.prefilled_at`: set when the form's first open has pre-filled it. Existing forms that already have patient answers (or are sent) are marked, so they aren't pre-filled again; a form with none is pre-filled at its next open, as before.
  - `audit_log.actor_org_id`: on a staff row written into **another** trust's log, the staff member's trust, with `user_id` NULL. The check constraint now allows a staff row with either `user_id` or `actor_org_id` (never both). Down puts the old rule back as `NOT VALID` (the log is append-only, so rows written since can't be changed).
- **First open only** (`prefill.go`): `prefillForm` runs in a transaction that locks the form row (`LockFormPrefill`, `FOR NO KEY UPDATE`, so saves of other sets don't wait), does nothing if `prefilled_at` is set, otherwise fills and sets it (`MarkFormPrefilled`). A second open at the same moment waits, then finds it done. `writeForm` now reads the answers **after** the pre-fill, so it always sends what the first open stored (the old "did I fill anything?" boolean is gone).
- **The staff-side join** (`accounts.go` `joinOnBehalf`, called by `staffPatientForm` in `patient.go`): only when the record has no account, the form isn't sent and hasn't been pre-filled. In one transaction: the date-of-birth lock (same order as `joinAccount`), the record locked, `FindAccountForRecord` (S15's four-field rule), `JoinAccount`. No match: nothing happens. The `account.join` row goes in the record's trust's log, with the staff member as actor.
- **Another trust's log names only the trust** (`audit.go`):
  - `writeAudit`: a staff request's row for a trust other than the request's own (`here(r).Org`) is written with `actor_org_id` and no `user_id`. Today that's only `prefill.read` on behalf.
  - The list and CSV say **"Staff at <trust>"** in Who, with no email (`actorTrust` added to the row).
  - The **Who** filter lists each such trust after the people (`ListAuditTrusts`), with the value `trust:<id>`; `?user=trust:<id>` filters by it, and a bad id is a 400.
- **The patient's home** sends `sent` per episode, so a cancelled one the patient had sent can say so. **The patient's form** sends `archived`.

### TypeScript (for Rahul to review)
- **`packages/clinical/src/prefill.ts` `startedFreshText`:** Validate's "Answers were pre-filled here… chose “Start fresh”…" line, now shared with the POA Summary.
- **`packages/clinical/src/poa.ts`:**
  - `poaSummary` keeps a Question Set started fresh even when nothing else on it shows, so its line appears.
  - `reportedBmi(sets, answersOf)` now takes the Question Sets and a function giving each set's answers. It reads the first height and weight it finds: a BMI question's, or the platform library's `height` and `weight` questions (the demo HQs use these, which is why it showed "—" before). It rounds to one decimal place, as the BMI question does.
- **`PoaSummary.tsx`:** the BMI uses the clinician's answers for a set once validated (`validatedAt`), else the patient's, on both tabs. Each set started fresh shows the `startedFreshText` line (`data-testid="poa-started-fresh"`). `restoreSpot`'s clean-up is returned from the effect.
- **`jump.ts` `restoreSpot`** returns a clean-up that cancels the pending animation frame, so leaving the POA Summary stops it.
- **`PatientHQ.tsx`:**
  - the sent card drops "A member of the team will review them." when `archived`;
  - `load()` now returns whether it loaded; after a Start fresh whose reload fails, the page shows "The answers in this section were cleared" with **Reload** (`freshUnloaded`), instead of leaving the card with the old answers.
- **`PatientHome.tsx` / `home.ts` `closedText`:** the line under a cancelled or closed questionnaire; a cancelled one the patient had sent reads "The hospital has cancelled this. Please contact them if you have questions."
- **`status.ts` `notResponding`:** false once cancelled, as once archived.
- **`AuditPage.tsx`:** a Who entry or row with no email (staff of another trust) shows the name only.
- **`globals.css`:** `scroll-margin-top` (the top bar's height plus a little) on SurveyJS question roots inside the staff shell, so `scrollIntoView` stops below the sticky bar.

## Decided by the builder (automatic mode)
- **The join runs only when it can matter:** the record has no account, the form isn't sent, and it hasn't been pre-filled yet. A later open of an already pre-filled form doesn't join (it would change nothing the form shows). Answer 1 said "before the form is pre-filled".
- **Trust B's row stores no user id at all** (not stored and hidden): `actor_org_id` names Trust A. Trust A's own row for the same request names the person, so the two can be put together by time if ever needed.
- **The `account.join` row has no hospital** (as the patient's own join): the record is the trust's.
- **Rows written before S28 stay as they were**: a Trust B row from S27 may still name a Trust A staff member (the log is append-only).
- **Patient reported BMI is the same on both tabs** (the details block isn't per tab): the patient's own until the set is validated.
- **The BMI reads the platform library's `height`/`weight` keys** as well as a BMI question: the demo HQs ask height and weight as two questions, so without it the brief's case (a number before validation) can't happen.

## Tests

### Go integration (API → database)
- `TestJoinOnBehalf` (new, `prefill_cross_trust_integration_test.go`):
  - all four fields match (the name typed with other case and spaces): the record joins, Trust B's answers fill in, one `account.join` row under Trust A naming Cara and none under Trust B; Trust A's record takes no email, and Trust A's patient page doesn't show Trust B's;
  - only this record joins (the person's other unjoined record stays), the account takes none of Trust A's contacts, and no account is made;
  - no account to join: none is made;
  - one field different (surname, first name, date of birth, another mobile, another email): no join, nothing from Trust B, no join rows, no reads;
  - a later open after the first pre-fill doesn't join;
  - the patient's own open of an unjoined record is still a 404.
- `TestPrefillCrossTrust` (changed): on behalf, Trust B's row is staff with `actor_org_id` = Trust A and no user; no Trust B row names Cara; Trust A's `behalf.view` names her; Trust B's admin sees "Staff at Trust A" with no email or id, the Who filter offers `trust:<id>` and filters by it, `trust:abc` is a 400, the CSV has no Cara. New: a trust readable only after the first open is read no more (patient and staff); four first opens at the same moment read Trust B once and all send the answers.
- `TestPrefillRevision` (rewritten): an HQ sent after the first open fills nothing (patient, a second session, staff); the revision-1 rules as before.
- `TestCancelledHomeSaysWhetherSent` (new): `sent` on cancelled sent, cancelled unsent and archived-from-cancelled episodes.
- `TestArchivedAfterSentKeepsItsView`: the form says `archived`.
- The five cases for the changed routes: `GET …/patient-form` is `TestCompleteOnBehalf`'s (unchanged, passing); the audit list's new filter value has its invalid-input case above.

### Unit (Vitest)
- `prefill.test.ts`: `startedFreshText`.
- `poa.test.ts`: `reportedBmi` (a BMI question; the library's height and weight; patient's then validated; nothing without both); a set started fresh is kept on both tabs.
- `home.test.ts`: `closedText`. `status.test.ts`: `notResponding` once cancelled. `jump.test.ts`: `restoreSpot` scrolls once drawn, and stops once cleaned up.

### End-to-end (Playwright)
- `test_j16b_prefill_cross_trust.py`:
  - **new** `test_staff_filling_in_for_a_patient_who_never_opened_the_link_get_trust_bs_answers`: Cara's Fill in for the patient shows the card naming Hospital B1, Start fresh empties the set; Trust A's CSV has one join by Cara; Trust B's CSV says "Staff at Trust A" with no email and no Cara, and its Who list offers it; a reload adds no read row;
  - **new** `test_on_behalf_one_detail_different_gets_nothing_from_trust_b`;
  - **new** `test_start_fresh_whose_reload_fails_says_the_answers_were_cleared` (phone): the reload is made to fail with a routed 503.
- `test_j5_poa_summary.py` **new** `test_a_set_started_fresh_says_so_and_the_bmi_comes_from_the_patients_answers`: the Start fresh line on both tabs; BMI 28.4 before validation, 31.1 after validating another weight.

## Results (25 Oct)
- `make lint`: clean.
- `make test`: Go ok; Vitest 232 (clinical) and 158 (web) passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` against `make dev` on the shared `sj` database (migrated to 00028, seeded): **69 passed, 5 skipped** (the Cognito-only ones).
- `make migrate-down` then `make migrate` on `sj`: 00028 goes down and up cleanly.

## Hand checks for Rahul
None. The tester ran T-13 by failing the request in the page, and checked R-04's way back (leaving straight after Back starts the next page at the top). Both are also automated.

## Suggested commits
1. `S28: pre-fill once per form, the staff-side four-field join, and another trust's audit names only the trust` — migration 00028, `prefill.go`, `accounts.go`, `patient.go`, `audit.go`, queries and sqlc, the Go integration tests, `AuditPage.tsx`, and the J16b journeys.
2. `S28: the POA Summary's Start fresh line and reported BMI, the archived View and cancelled-after-sent wording, NOT RESPONDING once cancelled, and S27's small fixes` — `packages/clinical` (`prefill.ts`, `poa.ts` and tests), `PoaSummary.tsx`, `ValidateSet.tsx`, `PatientHQ.tsx`, `PatientHome.tsx`, `home.ts`, `status.ts`, `jump.ts`, `globals.css`, their unit tests, and the J5 journey.
3. `docs: S28 build notes, test plan, decisions, pending, session status`.

## Test report 1 (25 Oct): all passed, 18 of 18
No code changed. Its exploring notes, none failures, are logged in [pending.md](../../pending.md) ("From S28's test report 1"):
- the raw "Failed to fetch" banner above the T-13 card;
- Start fresh and Check each answer share one audit label (older than S28);
- Trust B's Who box still lists Cara from S27's rows (written before S28, append-only);
- two audit rows per staff page view in the dev build (React's double effect; not new).
