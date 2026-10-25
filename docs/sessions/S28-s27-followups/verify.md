# S28 verification

## Verification 1 (25 Oct 2026, verifier, automatic run)

**Verdict: Verified.**

I reviewed the whole working-tree change against the brief and [answers.md](answers.md), ran `/code-review high` and weighed what it found, ran every automated suite, and spot-checked the riskiest functions myself in Chrome and Postgres against the conductor's `make dev`.

### Scope
Everything in the brief is built:
- the staff-side four-field join, as answered: only this record, only to an existing account, with no contacts copied;
- another trust's audit log naming only the trust;
- pre-fill once per form (migration 00028);
- the Start fresh line on the POA Summary;
- the archived View's wording, the cancelled-after-sent home card, and NOT RESPONDING hidden once cancelled;
- the reported BMI before validation;
- S27's three small fixes.

Nothing large was added beyond the brief. Reading the platform library's `height`/`weight` in `reportedBmi` is the builder's choice, recorded in the build notes. Without it the brief's case can't happen on the demo HQs.

### Findings (none blocking, worst first)

1. **The staff-side join can be made from details staff type** (`apps/api/internal/httpapi/accounts.go:110`, called from `patient.go:585` on a GET). Anyone at the hospital who knows a person's name, date of birth and mobile or email can join a new record to that person's account, and so see their other trusts' answers. This is Rahul's agreed design (24 Oct, "Risk accepted"), and the IG/DPO review before go-live (C10) already lists it. I'm noting it again because `/code-review` ranked it first. No change.
2. **Once per form also covers a first open that found nothing to read** (`prefill.go:54`). Suppose staff open Fill in for the patient first, while the record matches no account (for example the mobile differs). The form is then marked pre-filled. If the patient later opens their own link, which joins their account, they get nothing from other trusts. This follows from answer 3 ("later opens never read earlier HQs again"), and `TestPrefillCrossTrust` tests it on purpose. It may surprise Rahul, so I've added it to [pending.md](../../pending.md).
3. **`reportedBmi` reads answers whether or not their question or set is shown** (`packages/clinical/src/poa.ts:118`). The old version used the summary's captures, which only covered shown questions in shown sets. A BMI, or a height and weight, left in data after its condition hid it could still give the "Patient reported BMI". The demo HQs ask height and weight unconditionally, so this is minor. Logged in pending.md.
4. **The `prefill.read` rows are added before the pre-fill's transaction commits** (`prefill.go:95`). If the transaction fails, or the patient goes away mid-fill, the source trust's row is still written and the next open reads again. The read did happen, so the row is accurate, and this is rare. Minor.
5. **`prefilled_at` is the database's `now()`, not `Deps.Clock`** (`apps/api/db/queries/episodes.sql:257`). No test or rule reads the time, only whether it's set. Minor.
6. **Cosmetic:**
   - `closedText` is called twice in one expression (`apps/web/src/features/patient/PatientHome.tsx:71`);
   - the BMI formula now exists in `reportedBmi` as well as in the BMI question.

`/code-review` also suggested making Start fresh's POST return the form, so it needs no second GET. That's a design preference and isn't needed.

### Checks
- **Tests first:** the changed routes have integration cases.
  - The join: `TestJoinOnBehalf`. All four fields match: joined. Each field different in turn: no join. Only this record is joined, no account is made, no contacts are copied. A later open doesn't join. Not this patient's: 404.
  - The audit's `trust:` filter: `TestPrefillCrossTrust`, including `trust:abc` → 400.
  - The patient's home and form flags: `TestCancelledHomeSaysWhetherSent` and `TestArchivedAfterSentKeepsItsView`.
  - The five cases for `GET …/patient-form` stay in `TestCompleteOnBehalf`.
  - The logic has Vitest cases, and the journeys are J16b (three new) and J5 (one new).
- **Tenant isolation:**
  - Other trusts are still read one trust per query.
  - `joinOnBehalf` runs only after `loadStaffForm`'s hospital check, and never for a cancelled or archived episode, because `loadForm` refuses both first.
  - `ListAuditTrusts` and the `actor_org_id` filter are scoped by `org_id`.
- **Nothing hardcoded for the UK:** the A-20 grep finds only comments, config, seed and tests.
- **Security:**
  - No secrets were added.
  - Trust B's rows and CSV carry no Trust A user ID, name or email.
  - The patient API gets no staff routes.
  - Clinician-only stripping is unchanged.
- **Docs:**
  - `decisions.md` has the 25 Oct builder decisions and answers;
  - `pending.md` has S28's notes;
  - `docs/sessions/README.md`, `CLAUDE.md` and `apps/web/AGENTS.md` are updated;
  - `repocheck` passes.

### Spot-checks (Chrome, "rahul chrome", against `make dev` on migration 00028)
1. **Permissions before the join.** As Cara I made a Hospital A1 episode for a new record matching account `s28t04b` on all four fields.
   - Val (Hospital A2) got **404** for its `patient-form`, both under A2 and under A1. Signed out: **401**.
   - Postgres afterwards: the record still had no `account_id`, and the form had no `prefilled_at`. Nothing was joined or read, and nothing was written to Trust A's log.
2. **The join and first open only.** Cara's Fill in for the patient showed both sets as **Please check**. Your health's card read "We've filled in 5 answers from Sam's questionnaire at Hospital B1 on 25 Oct 2026. Please check them with Sam."
   - Postgres: the record joined the existing account, the account's contacts were unchanged, and `prefilled_at` was set.
   - For this record, the page's double load and a full reload made exactly one `account.join` row (Trust A, Cara) and one `prefill.read` row (Trust B: `user_id` NULL, `actor_org_id` = Trust A).
3. **Trust B's log** (Max Multi, Trust B admin).
   - The API row for that read: Who "Staff at Trust A", email "", `userId` null.
   - The Who list offers `trust:<Trust A id>` as "Staff at Trust A", and filtering by it gives only "Staff at Trust A" rows. `?user=trust:abc` is **400**.
   - The filtered CSV's row has Who "Staff at Trust A", an empty Email and no "cara".
   - The Audit page shows the same: the screenshot listed 10 "Staff at Trust A" rows, the newest being Sam S28 T04b.
   - Max (a clinician only at Trust A) gets **403** on Trust A's audit.
   - Trust B's Who list still names Cara once. That's from S27's rows, written before S28, as the tester also noted.

The patient pages at phone size were covered by test report 1 (T-09, T-10, T-13 at 390 wide) and by J16b's phone journey, so I didn't repeat them.

### Test results (25 Oct)
- `make lint`: clean.
- `make test`: Go ok (including `repocheck`); Vitest 232 passed (clinical) and 158 passed (web); CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against `make dev`): **69 passed, 5 skipped** (the Cognito-only journeys) in 4 min 47 s.

### Hand checks for Rahul
None.

### Suggested commits
1. `S28: pre-fill once per form, the staff-side four-field join, and another trust's audit names only the trust`
   - Go: migration 00028, `prefill.go`, `accounts.go`, `patient.go`, `audit.go`, the queries and sqlc output, and the Go integration tests.
   - Web: `AuditPage.tsx`.
   - Journeys: J16b.
2. `S28: the POA Summary's Start fresh line and reported BMI, the archived View and cancelled-after-sent wording, NOT RESPONDING once cancelled, and S27's small fixes`
   - `packages/clinical`: `prefill.ts` and `poa.ts`, with their tests.
   - Web: `PoaSummary.tsx`, `ValidateSet.tsx`, `PatientHQ.tsx`, `PatientHome.tsx`, `home.ts`, `status.ts`, `jump.ts`, `globals.css`, and their unit tests.
   - Journeys: J5.
3. `docs: S28 verified`
   - The brief, answers, build notes, test plan, test report and this verification.
   - `decisions.md`, `pending.md`, `docs/sessions/README.md`, `CLAUDE.md` and `apps/web/AGENTS.md`.
