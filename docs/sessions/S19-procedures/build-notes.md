# S19 build notes: procedures and episode dates

Builder, automatic run (`/run-sessions`), 20 Oct 2026.

## Answers (from [answers.md](answers.md), asked by the conductor on 20 Oct; also decisions.md §2)

1. **Who manages the list:** (b) super clinicians and admins (of the whole trust).
2. **Hospital-only procedures:** yes, `hospital_id` NULL = every hospital of the trust.
3. **INF-04 information content and PRM-01 PROMs plan:** add the columns **now** (not the recommendation).
4. **The free-text procedure:** (a) replaced by an optional `procedure_id`. The conductor's note: carry the texts across or re-seed, consistently with S18. **Carried across** (below).
5. **Removing a procedure in use:** a Retired flag; nothing is deleted.
6. **Default HQ:** the procedure's, then the hospital's `defaultHq`; the clinician can always change it.

## What was built

- **Migration `00020_procedures.sql`:**
  - **`procedures`**: id, org_id, `hospital_id` (a composite FK to the trust's own hospitals, as questionnaires have), name (unique per trust on `lower(name)`, retired ones included), code, specialty, `default_questionnaire_id` (ON DELETE SET NULL), `info_content` and `proms_plan` (jsonb, CHECK they're lists, default `[]`), retired, updated_by, created_at, updated_at.
  - **`episodes`** gets `procedure_id` (optional), `planned_date` and `poa_date` (both `date`, optional), and loses the free-text `procedure`: each trust's distinct texts (trimmed, any case once) become its procedures, for every hospital and with no default HQ, and the episodes are linked to them. Down puts the text back from the names.
- **Queries:** `db/queries/procedures.sql` (new: the list, the offered list, the offered check, get-for-update, create, update, a hospital's and its trust's settings); `episodes.sql` (`CreateEpisode`, `ListEpisodes`, `GetEpisode`, `UpdateEpisode` carry the procedure and dates; the procedure's name keeps the JSON name `procedure`; `ListPublishedVersions` adds `questionnaire_id`); `patients.sql` and `accounts.sql` (`ListHome`) read the procedure's name.
- **Go, `internal/httpapi/procedures.go`** (new):
  - `GET /api/o/{oid}/procedures`: a whole-trust member gets every procedure, a member of some hospitals the trust's and those hospitals' own; retired ones last.
  - `GET /api/o/{oid}/procedures?hospital={hid}`: what a new episode there is offered (not retired; the trust's and that hospital's own), each with **`defaultVersionId`**, the HQ it picks there: its own default's published version if that hospital uses it, else the hospital's effective `defaultHq` setting's, else null. A member of another hospital gets 404.
  - `POST` and `PATCH /api/o/{oid}/procedures/{prid}`: whole-trust super clinicians and admins only (403 otherwise). PATCH changes the fields sent; `hospitalId` or `defaultQuestionnaireId` null clears it. A changed hospital must be the trust's; a changed default HQ one of the trust's published questionnaires. Duplicate name → 409. Both are wrapped in `audited` (`procedure.create`, `procedure.update`).
- **Go, `episodes.go`:** create takes `procedureId`, `plannedDate`, `poaDate` (each optional; the procedure must be offered at this hospital now); PATCH takes the same, until the HQ review is complete (then a change is a 409, "The procedure and its dates can't be changed once the HQ review is complete."). An episode keeps a retired procedure; a new one must be offered. Each change writes a General notes entry. `GET …/published-hqs` now marks the hospital's default HQ (`isDefault`) and gives each one's `questionnaireId`.
- **Seed (`cmd/seed`):** one published **Demo pre-op HQ** per trust (`demo-hq.json`, embedded; made up, the e2e journeys' HQ; created once and never reset) and six procedures, put back as seeded on each run:
  - Trust A: Total hip replacement (HIP-01, Orthopaedics, Demo HQ), Knee arthroscopy (KNEE-02, Orthopaedics, Demo HQ), Laparoscopic cholecystectomy (General surgery, **no default HQ**), Cataract surgery (EYE-01, Ophthalmology, **Hospital A1 only**, Demo HQ);
  - Trust B: Total hip replacement (HIP-01, Demo HQ B), Inguinal hernia repair (General surgery, Demo HQ B).
- **Web:**
  - **Procedures page** `…/procedures` (`features/procedures/ProceduresPage.tsx`), in the header for whole-trust super clinicians and admins (`canManageProcedures`); a deep link from anyone else shows a refusal. Add and Edit are an inline form (name, code, specialty, offered at, default HQ); Retire / Bring back on each row.
  - **New episode:** a Procedure dropdown (searchable) from the trust's list for this hospital; picking one chooses its default HQ; the page opens with the hospital's default HQ; Planned procedure date and POA appointment date (date inputs).
  - **Episode page:** Planned procedure date and POA appointment date in the details; **Edit procedure and dates** (inline form, `EditProcedure.tsx`) until the HQ review is complete. A retired procedure shows as "(no longer offered)" in its dropdown and is kept.
  - **Episode list and POA Summary:** the procedure and both dates. **Patient home:** the procedure's name (unchanged code; the name now comes from the list).

### TypeScript Rahul should know about

- `features/procedures/procedures.ts` (unit-tested in `procedures.test.ts`): the API's shapes, and four small rules: `procedureLabel` ("Name (CODE)"), `initialHQ` (the hospital's default HQ, marked `isDefault` by Go), `hqForProcedure` (after a pick: the procedure's `defaultVersionId` from Go if it's one of the offered HQs, else keep the current choice), `procedureChoices` (the offered procedures, plus the episode's own if it's no longer offered), and `changedFields` (an edit PATCHes only what changed, so an unchanged default HQ that's since lost its published version isn't checked again). **The rule "own default, else the hospital's" is in Go**, not TypeScript.
- `lib/format.ts` `formatOptionalDate`: a `YYYY-MM-DD` date in the deployment's short style ("03/11/2026" in en-GB), or "—".

## Decided by the builder (automatic mode)

- **The default HQ is stored as a questionnaire, not a version** (the plan and brief say `default_version_id`). A new episode gets its published version at the time. Storing a version would strand the default the moment a new version is published (publishing retires the old one), and S10's `defaultHq` setting already names a questionnaire.
- **The procedure-then-hospital default is resolved in Go** (`?hospital=` gives each procedure's `defaultVersionId`), so the brief's "picking a procedure fills the default HQ" is an API → database test. This also **wires S10's `defaultHq` into the New episode page for the first time**: the page opens with the hospital's default HQ selected.
- **A super clinician of one hospital only can't manage the list**, not even for their hospital's own procedures: managing is for the whole trust's super clinicians and admins (answers 1, read as for the trust's questionnaires). They and clinicians see the list only as the New episode choices.
- **Specialty is optional** (the brief made only the code optional). Procedures carried across have no specialty, and requiring one would block retiring them.
- **INF-04 and PRM-01 (answers 3):** the columns `info_content` and `proms_plan` are JSON lists, `[]` for now, returned by `GET …/procedures` but **not writable or shown** yet. Their items (leaflets need file storage, C8; PROMs instruments, C9) aren't defined, so any shape chosen now would be a guess.
- **Answer 4: carried across, not re-seeded.** Migration 00020 turns each trust's distinct free texts into procedures and links the episodes. Unlike S18's reshape, this is cheap and exact, and keeps any episode made since migration 00019 (locally, and on QA after the push).
- **General notes entries** for each change to the procedure and dates ("Procedure set to …", "Procedure cleared", "Planned procedure date set to 2026-11-03", "POA appointment date cleared"), as ASA changes have. The date in the note is ISO, since notes are stored text.
- **Dates:** any valid date from 1900 to 2199; no rule between the two dates (the brief has none).
- **The seed adds a published demo HQ per trust**, so the seeded procedures can have a default HQ (the brief's "with default HQs") and a tester can make an episode without publishing an HQ first.
- **Audit:** the brief says "audited like the other admin changes", but Staff and Settings changes aren't in the audit log today. The procedure changes are (one row per POST or PATCH, no patient or episode IDs; "Added a procedure", "Changed a procedure").
- **Names are unique in the trust including retired ones** ("The trust already has a procedure with this name (retired ones included)."), so a retired name is brought back rather than duplicated.

## Differences from the plan or brief

- The default HQ column is `default_questionnaire_id`, not `default_version_id` (above). Recorded in the plan's Status.
- PRC-01 says picking a procedure "fills in" the information content and PROMs plan too; those wait for C5 and C9 (only the columns exist).

## Tests (TDD)

Written before the code they test, and seen failing, then passing (the migration itself was written just before its test):
- **Migration:** `TestProceduresMigration` (`internal/apitest`): a database at 00019 with episodes in two trusts, texts differing in case and spaces, and a blank one; after 00020, each trust's distinct procedures, the links, no `procedure` column, the unique name in any case, the list CHECKs, the hospital FK, and Down then Up.
- **Routes (five cases each):** `TestListProcedures`, `TestCreateProcedure`, `TestUpdateProcedure` (`procedures_integration_test.go`): who sees what, `?hospital=` with the default HQ (own, the hospital's, the trust's, none when unpublished), 401, 403 for a clinician and a one-hospital super clinician, 404 for another trust, 400s (name, lengths, another trust's or a draft HQ, another trust's hospital, unreadable), 409 duplicate, partial PATCH, null clears, retire and bring back, audit rows.
- **Episodes:** `TestCreateEpisode` (procedure and dates stored; retired, another hospital's own, another trust's, unknown and malformed procedures refused; bad dates refused), `TestUpdateEpisode` (procedure and dates before the review with their notes; a retired one kept; 409 after the review), `TestGetEpisode` and `TestListEpisodes` (the name, id and dates), `TestListPublishedHQs` (`isDefault`, the hospital's over the trust's). The tests that sent the old `procedure` text now send `procedureId` (`audit`, `invites`, `patients`, `accounts`).
- **Seed:** `TestSeedTwice` checks the demo HQs and procedures, and that a retired seeded procedure is put back.
- **Unit:** `TestReadEpisodeDates`, `TestEpisodeDatesChanges` (Go); `procedures.test.ts`, `auth.test.ts` (`canManageProcedures`, switching hospital on Procedures), `format.test.ts` (`formatOptionalDate`), `routes.test.ts` (the new route).
- **End-to-end:** new `journeys/test_procedures.py` (Alex adds, edits, a duplicate is refused, Cara's pick chooses the default HQ and can change it, Val at A2 isn't offered A1's own, retire; Sue Super has no Procedures; Cara edits the procedure and dates, sees them in the list and on the POA Summary, and can't after the review). J3 now picks a procedure (the HQ follows it, then she picks the journey's HQ) and the dates; J7c checks the patient's home shows "Procedure: Knee arthroscopy" at phone size. `support.Api.episode` gives episodes the seeded Knee arthroscopy.

### Results (20 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 175 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against my own API on :8091 and Vite on :3011, `E2E_BASE_URL=http://localhost:3011`): **46 passed, 5 skipped, 1 failed**; the failure, J7's first case, **passed on a re-run** (6 passed). It's S18's known flake (J7 picks a random mobile among 1,000 that earlier runs have often used). An earlier run also had `test_audit`'s "list row" case fail once and pass on a re-run (5 passed): its page reads the newest audit rows, which other journeys write at the same moment.
- **Local database:** migrated to 00020 and re-seeded (`make migrate seed`).

## Test report 1 (20 Oct): 22 of 23 passed; F1 fixed

- **F1 (T-13): a partly typed date was dropped silently.** A browser date input typed only in part (e.g. "28/10/yyyy") has the value "", the same as no date, so New episode created the episode without it and Edit procedure and dates saved without it. **Fixed:** both forms now check the inputs' `validity.badInput` before sending and show the API's wording ("Enter the POA appointment date as a date."), sending nothing (`features/episodes/dates.ts`, `partlyTypedDate`). **Tests first:** `test_a_partly_typed_date_is_refused_not_dropped` in `e2e/journeys/test_procedures.py` (failed before the fix: the episode was created) and `dates.test.ts`. The server can't see this case: the browser never sends the half-typed text.
- **Found while exploring, in S19's scope, fixed:**
  - Edit on a lower row of Procedures opened the form out of sight: the form now scrolls itself into view.
  - Enter didn't send the Add/Edit procedure form: it's a real `<form>` now, and the journey's duplicate-name step presses Enter. A second Enter while saving is ignored.
- **Found while exploring, not S19's, logged in [pending.md](../../pending.md) §3 ("From the S19 manual test"):** the header overflows for a whole-trust admin below about 1280 px (partly older; the Procedures link adds about 114 px); the New episode HQ select isn't searchable (older).
- Exploring note 6 (the carried-across procedures in both trusts' lists) is migration 00020 working as intended.
- **Results after the fixes:** `make lint` passed; `make test` Go ok, Vitest 177 + 116, CDK 124; `make test-integration` all ok; `make e2e` on my own ports: 46 passed, 5 skipped, 2 failed (`test_audit` list row and J8). Both passed on a re-run (6 passed). The machine's load average was about 180 during that run (requests logged "context canceled"), and J8 hit the known reused-mobile flake (pending.md, S18 item 1).

## Test report 2 (20 Oct): 16 of 17 passed; F1 fixed

- **F1 (T-06 step 3): an empty name got Chrome's "Please fill in this field." bubble**, not the app's "Enter the procedure's name.", and an earlier red message could stay beside it. A side effect of report 1's "Enter sends the form" fix (a real `<form>` with a `required` Name). **Fixed:** the form is `noValidate`, so the API's message shows, and each attempt clears the last message. **Test first:** the procedures journey now empties the name after the duplicate-name refusal, presses Enter, and expects "Enter the procedure's name." with the duplicate message gone (it failed before the fix).
- **Found while exploring, small and in S19's scope, fixed:**
  - After Retire or Bring back on a lower row, the page scrolls to the top, where the message is.
  - Enter sends **Edit procedure and dates** too (also a `noValidate` form); the journey presses Enter in the POA date to save.
- Exploring note 3 (1Password on the Name field) is the tester's browser profile, not the app.
- **Results after the fixes:** `make lint` passed; `make test` Go ok, Vitest 177 + 116, CDK 124; `make test-integration` all ok; `make e2e` on my own ports: 47 passed, 5 skipped, 1 failed (`test_patient_records`' linked-silently case), which passed on a re-run (4 passed). The machine's load average was 165 to 230 during the run, which took 11 minutes.

## Test report 3 (21 Oct): report 2's fix passes; F1 fixed; 8 cases blocked by the browser

- **F1: Retire on a lower row still didn't bring its message into view** (the smooth scroll stopped part-way, 3 of 3). The list reload right after the save changed the page and cut the smooth scroll short. **Fixed:** the scroll now runs after each new message and again once the list has reloaded, and it's instant, to the top of the page (the message is there, under the fixed header). **Test first:** the procedures journey shrinks the window to 300 px high, scrolls the row below the fold, Retires it, and expects the message fully in view, still in view a second later, with the page at the top. It then does Bring back and Retire again. This failed before the fix with "viewport ratio 0", the tester's symptom. The journey passed 3 runs of 3 after it.
- **The cases blocked by the browser disconnecting, run in e2e instead:**
  - T-04: the procedures journey now opens Edit on a row below the fold (300 px window), expects the form in view, and saves with Enter.
  - Enter in Edit procedure and dates: the dates journey saves with Enter in the POA date.
  - R-1: `test_patient_records` (search, Choose, a second episode). R-2: J3 (the patient fills in and sends), J4 and J5 (validate, complete the review, both POA Summary tabs). R-3: J6 (settings). R-4: `test_audit`. R-5: `test_trusts` and `test_staff` (who sees which header links), plus `test_procedures` (Sue has no Procedures).
- T-17 at 390 px is still a hand check for Rahul.
- **Results after the fix:** `make lint` passed; `make test` Go ok, Vitest 177 + 116, CDK 124; `make test-integration` all ok; `make e2e` on my own ports: 47 passed, 5 skipped, 1 failed. The failure was J7's first case, the known reused-mobile flake, and it passed on a re-run (6 passed).

## Test report 4 (21 Oct): all passed

- Every case passed (9), with T-17 at 390 px blocked as a hand check.
- Its one exploring note (cosmetic: after Edit on a lower row, the form's heading sits under the fixed header) is logged in [pending.md](../../pending.md) §3, "From the S19 manual test", item 3. No code changed.
- **Finish:** the session is **Tested** in `docs/sessions/README.md`; the plan's Status and `docs/plans/README.md` say S19 tested, awaiting verification; decisions.md §2 has the S19 entry, with the gotchas from the test rounds.

## Hand checks for Rahul

- **T-17 at 390×844:** the tester's Chrome window couldn't go below 500 px wide, so the patient home was checked at 500 px. Check it at 390 px in device emulation. (J7c checks the same home at phone size automatically.)

## Known limits (minor, for the verifier)

- A procedure picked on the New episode page could be retired by someone else before Create; the server then refuses with "Choose a procedure from the trust's list for this hospital." (no stale-list race beyond that).
- The audit row for a procedure change records the route, not which procedure (the audit log holds patient and episode IDs only).
- The Procedures page's hospital choices are the trust's hospitals in `/api/me` (a whole-trust member's are every active one); a procedure of a hospital since suspended still shows its name.

## Suggested commit

One plan step, one commit:

```
S19: procedures and episode dates — the trust's procedure list picks the default HQ

Migration 00020 adds the trust's procedures (hospital-only or every hospital, code, specialty,
default HQ, retired, and the empty info-content and PROMs-plan lists), carries each trust's
free-text procedures across into it, and gives episodes procedure_id, planned_date and poa_date.
Routes GET/POST/PATCH /api/o/{oid}/procedures (whole-trust super clinicians and admins change it,
audited); ?hospital= gives each procedure the HQ it picks there (its own, else the hospital's
defaultHq). Episodes take the procedure and dates on create, and on edit until the review is
complete. Web: the Procedures page, the New episode picker, the episode's Edit procedure and dates,
and the dates in the list and on the POA Summary. The seed adds a demo HQ and procedures per trust.
Tests: the migration, the three routes' five cases, the episode create/update/get/list changes,
the seed, unit tests, and the procedures journey.
```

Files: `apps/api/db/migrations/00020_procedures.sql`, `apps/api/db/queries/{procedures,episodes,patients,accounts}.sql`, `apps/api/internal/db/*` (generated), `apps/api/internal/httpapi/{procedures,episodes,router,audit}.go`, `apps/api/internal/httpapi/{procedures,episodes,audit,invites,patients,accounts}_integration_test.go`, `apps/api/internal/httpapi/episodes_test.go`, `apps/api/internal/apitest/migrations_integration_test.go`, `apps/api/cmd/seed/{main.go,demo-hq.json,seed_integration_test.go}`, `apps/web/src/features/procedures/*`, `apps/web/src/features/episodes/{types.ts,EpisodeList.tsx,NewEpisodePage.tsx,EpisodePage.tsx,EditProcedure.tsx,PoaSummary.tsx,dates.ts,dates.test.ts}`, `apps/web/src/lib/{auth.tsx,auth.test.ts,format.ts,format.test.ts}`, `apps/web/src/pages/hospital/{HospitalLayout.tsx,Procedures.tsx}`, `apps/web/src/{routes.tsx,routes.test.ts}`, `apps/web/AGENTS.md`, `e2e/support.py`, `e2e/journeys/{test_procedures.py,test_j3_patient.py,test_j7c_returning_patient.py}`, `CLAUDE.md`, `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/c3-episode-lifecycle.md`, `docs/sessions/README.md`, `docs/pending.md`, `docs/sessions/S19-procedures/*`.
