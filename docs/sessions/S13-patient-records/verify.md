# S13 verification

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Verified.** Everything in the brief is built:
- the trust's patient record with `national_id` + `national_id_type`, unique per trust;
- the NHS number's Modulus 11 check;
- the search of this hospital's patients only;
- the matching rules, including Rahul's name + date of birth + contact rule;
- episodes at any of the trust's hospitals, with the hospital number on the episode;
- the New episode page and the patient page;
- the journeys.

Every new query is scoped by `org_id`, and by the hospital wherever a hospital lists or shows a patient. Both new routes (`patient.search`, `patient.view`) and the changed `POST …/episodes` are wrapped in `audited`. Every automated suite passes, and test report 3 passed. Nothing found needs fixing before commit. The items below are follow-ups, now in [pending.md](../../pending.md) §3 ("Found by S13's verification").

### Privacy checks (the main risk)

| Check | How | Result |
| --- | --- | --- |
| Search lists this hospital's patients only | `SearchPatients` (`patients.sql`) needs `org_id` and an episode at `hospital_id`. As Cara (A1 only), searching an A2-only patient's NHS number and surname returns `[]`, and the UI says "No patients at this hospital match." | Pass |
| A patient page never shows another hospital's patient | `GetHospitalPatient` needs an episode here; `ListPatientEpisodesHere` is filtered by `hospital_id`. Cara → A2-only patient at A1: 404. Trust B's patient id on A1's route: 404. A2's and trust B's routes: 404 "Hospital not found." | Pass |
| A found patient must be this hospital's | `POST …/episodes` with `patientId` of an A2-only patient: 400, naming nobody. | Pass |
| A mismatch shows no stored values | A wrong date of birth or surname with an A2 patient's NHS number: 409 `mismatch`, a fixed message only. | Pass |
| A possible duplicate known only elsewhere names nobody | Same name + date of birth as an A2-only patient with another mobile: 409 `possible_duplicate`, `matches: []`. | Pass |
| Matching queries are trust-scoped | `PatientByNationalID` and `PatientsBornOn` take `org_id`; `here` limits what's shown to this hospital's patients (`patients.go`, `resolvePatient`). | Pass |
| Every request is audited, refusals name nobody | `audit_log` rows for each probe above: `patient.search` (200), `patient.view` (404), `episode.create` (400/409), all with empty `patient_ids`. | Pass |
| Create returns the episode id only | `createEpisode` (`episodes.go`). | Pass |
| The patient API can't reach these routes; no contacts in search results | Staff routes only (`staffRoutes`); `SearchPatients` selects no mobile or email. | Pass |
| A-20 | The grep prints only comments (`format.ts`'s examples, and `patient.go:19`'s example `+447700900123`); the identifier's label comes from config (`NATIONAL_ID_TYPE`, `GET /api/config`). The company-name grep prints nothing. | Pass |

### Findings (all minor, worst first; none blocks the commit)

1. **The `?patient=` load has no stale guard** (`apps/web/src/features/episodes/NewEpisodePage.tsx:43`). If staff click Change patient before `GET …/patients/:pid` answers, the late answer chooses that patient again. The patient's card is shown again, so it can't go unseen, and locally the request takes milliseconds. Fix: ignore the answer once the effect is cleaned up, or once `patientId` has changed.
2. **Two creates at the same moment for one patient without an NHS number make two records** (`apps/api/internal/httpapi/patients.go:159`, `resolvePatient`). Only the NHS number has a unique index. This extends the "rapid clicks" item already in pending.md: it can also happen between two members of staff. Possible fix: a transaction-level advisory lock on (trust, date of birth).
3. **The name length limit counts bytes, not characters** (`patients.go:118`, `len(...) > 100`). A name in a non-Latin script is refused well under 100 characters, whereas the search counts characters. This matters for A-20 (other countries).
4. **"an " + the identifier's label** (`apps/api/internal/patient/patient.go:150`, `TooBroad`) reads wrongly for a label such as "health card number". It's a code change for a new country, so it touches A-20; reword it ("…the surname, the NHS number…").
5. **The migration copied a patient's one hospital number onto their episodes at every hospital** (`00010_patient_records.sql:12`). Under PAT-01b, A2 could now show A1's number. The data is made up (A-1), so there's nothing to fix. It's noted only for the record.
6. **The search query ORs three conditions** (`patients.sql:21`), so Postgres can't use the new indexes for them, and it scans the trust's patients on each search. That's fine at prototype size. Before a large trust, use a UNION of three indexed branches.
7. **The Audit filter compares `national_id` without `national_id_type`** (`audit.sql:44`). It's harmless with one scheme per deployment (A-20), but inconsistent with the other queries.
8. **Code-review notes, not defects:**
   - a third copy of `Detail` (`PatientPage.tsx`);
   - two hand-written row → `patient.Record` mappings (`resolvePatient`).
   - The review also flagged "a link keeps the stored record and drops what was typed". That is Rahul's confirmed decision (answers 5), and the lost NHS number is already in pending.md.

### Test results (13 Oct)

- `make lint`: passed.
- `make test`: Go all ok (new `nationalid` and `patient` packages included); Vitest 100 passed (7 files) and 116 passed (15 files).
- `make test-integration`: every package ok (`internal/httpapi` 13.0 s).
- `make e2e` (after `make seed`): **32 passed** in 114 s.
- **Spot checks against `make dev`:** the privacy table above (API probes as Cara Clinician, A1 only), the New episode search UI for an A2-only NHS number, and Nora Valid R3's patient page (only A1's episodes). All passed.

### Hand checks for Rahul

- **The patient link in a real 390×844 window** (Chrome's device toolbar or a phone). The tester used a same-size iframe. J3 runs it in WebKit's iPhone 13 profile automatically.
- **A real Export CSV download** on the Audit screen: the endpoint returns a valid CSV.

### Suggested commits

1. **Patient records: national ID, NHS number check, search, matching and the API (C2 / S13: data and API).** `apps/api/**` (migration 00010, `internal/nationalid`, `internal/patient`, `internal/config`, `internal/httpapi`, queries, sqlc output, seed, `go.mod`/`go.sum`), `.env.example`, `CLAUDE.md` (its one S13 line).
2. **The New episode page, the patient page and journeys (C2 / S13: web and journeys).** `apps/web/**`, `e2e/**`.
3. **Docs: S13 brief, answers, build notes, test plan, test reports, verification and status.** `docs/sessions/S13-patient-records/`, `docs/plans/c2-patients-signin.md` (committed whole: it also holds the 12 Oct local-first planning note), `docs/plans/README.md`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`.
