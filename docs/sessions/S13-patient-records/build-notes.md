# S13 build notes: patient records

Builder, automatic run (`/run-sessions`), 13 Oct 2026. Rahul's answers are in [answers.md](answers.md).

## What was built

### Data (migration `00010_patient_records.sql`; up, down, up checked on the local database)

- **`patients`** (the trust's record): `national_id` + `national_id_type` (NULL together; A-20), `account_id` (NULL; S15 adds `patient_accounts` and the foreign key), `mobile` (renamed from `phone`; E.164 or `''`, checked by the database), `email` (lower case), `created_at`. **Unique `(org_id, national_id_type, national_id)`** where set: another trust may hold the same NHS number.
- **`episodes.hospital_number`** (PAT-01b): moved from the patient. The migration copies each patient's number onto their episodes; phones that weren't already E.164 are dropped (made-up data only, A-1).
- **Indexes** for the search and the duplicate check: surname prefix (`lower(last_name) text_pattern_ops`) with date of birth, date of birth, `episodes (hospital_id, patient_id)`, and `episodes (hospital_id, lower(hospital_number))`.

### Go

- **`internal/nationalid`:** one validator per identifier scheme. Only `nhs` exists: the **Modulus 11 check** (written test-first), and `Normalise` strips spaces and hyphens.
- **`internal/config`:** **`NATIONAL_ID_TYPE`** (local default `nhs`; required with `CONFIG_STRICT=true`; must be a scheme `internal/nationalid` knows). `GET /api/config` now also returns `nationalId: {type, label}`, so the web app labels the field ("NHS number") without a UK literal.
- **`internal/patient`** (no database, unit-tested): `Mobile` (E.164 via `github.com/nyaruka/phonenumbers`, a number without a country code read as the deployment's `PHONE_COUNTRY`), `Email`, `NameKey` (case and spaces ignored), **`Decide`** (the matching rules, below) and **`ParseSearch`** (the search term rules).
- **Routes** (`internal/httpapi/patients.go`, `episodes.go`), each wrapped in `audited`:
  - `GET /api/o/{oid}/h/{hid}/patients?q=&dob=` (`patient.search`): **this hospital's patients only** (at least one episode here). `q` is an NHS number (exact, when its check digit is valid), a hospital number here (exact, any case) or the start of a surname; `dob` narrows it. Needs 3 characters, or 1 with a date of birth (else 400). Returns `{patients: [{id, firstName, lastName, dateOfBirth, nationalId, hospitalNumber}], more}`, at most 20. No contacts.
  - `GET /api/o/{oid}/h/{hid}/patients/{pid}` (`patient.view`): the trust's record and the patient's **episodes at this hospital only**; 404 for a patient with no episode here.
  - `POST /api/o/{oid}/h/{hid}/episodes` now takes **either** `patientId` (one of this hospital's patients; anyone else is 400) **or** `patient` (typed details) plus `createAnyway`, and `hospitalNumber`. One transaction: find or create the record, then the episode. The response is the episode's id only.
  - **Removed:** `POST …/patients` (a patient is created with their first episode) and the trust-wide patient list behind the old drawer's dropdown.
- **The matching rules** (`patient.Decide`), on Create with typed details:
  1. **NHS number found in the trust:** date of birth and surname (ignoring case and spaces) agree → **link silently**; otherwise **409 `mismatch`** with no stored values, nothing written. First names may differ (Sam/Samantha).
  2. **No NHS number typed, or one the trust doesn't have:** candidates are the trust's records with the same first name, surname and date of birth (and, when an NHS number was typed, no different NHS number of their own).
     - a candidate with the **same email or mobile** → **link silently**;
     - otherwise any candidate → **409 `possible_duplicate`**. `matches` lists only candidates that are **this hospital's** patients (to "use this record"); one known only elsewhere is never shown. **`createAnyway: true`** creates a new record.
  3. Otherwise a new record.
  - A race (two people typing the same new NHS number at once) gets a 409 "A record with this NHS number was added just now. Try again." from the unique index (tested with `meanwhile`).
- **Audit:** new actions `patient.search` ("Searched patients") and `patient.view` ("Opened a patient"); the old `patient.list`/`patient.create` labels stay for old rows. The Audit screen's patient label shows the NHS number instead of the hospital number; its **Patient** filter matches a hospital number on any of the patient's episodes and an NHS number (typed with or without spaces).
- **Seed:** Jo Bloggs (NHS 9990001006), Pat Example (no NHS number) and Robin Sample (NHS 9990001014), with E.164 mobiles and emails, reset each run. They have no episodes in a fresh database.

### Web app

TypeScript for Rahul to review, in plain terms:
- **`features/episodes/NewEpisodePage.tsx`** (`/o/:orgId/h/:hospitalId/episodes/new`): replaces the drawer. A search card (searches 0.3 s after typing stops, only once `searchQuery` says the term is precise enough), a **New patient** form, an **Episode** card, then **Create episode**. A `?patient=<id>` address (from a patient page) starts with that patient chosen. On a 409 it shows a red "The details don't match" box, or an orange "Possible duplicate" box with **Use this record** (this hospital's matches only) and **Create anyway**.
- **`features/episodes/newEpisode.ts`** (+ test): `searchQuery` (the API's "precise enough" rule, so the page never sends a broad search) and `conflictOf` (reads the 409's `code` and `matches`).
- **`features/episodes/PatientPage.tsx`** (`…/patients/:pid`): the record, then "Episodes at <hospital>", and **New episode** for this patient. The episode page's heading links to it.
- `lib/api.ts`: `ApiError` now keeps the response body (for the 409's `code`). `lib/config.ts`: `nationalIdLabel()`.
- The episode page and POA Summary show **NHS number**, **Mobile** and **Email**; "Phone number" is now "Mobile".

### Journeys (`e2e/`)

- **J3** creates its patient on the New episode page (with an NHS number and mobile) and checks the cleaned-up values.
- **New `journeys/test_patient_records.py`:** silent linking from A1 to an A2 patient (nothing of A2 shown, A1's search can't find them first); a wrong check digit and a mismatch; a possible duplicate known elsewhere (Create anyway) and one at this hospital (Use this record); the search by surname, NHS number, hospital number and short surname + date of birth, and a second open episode.
- `support.py`: `nhs_number()` (a random valid made-up number) and `Api.episode(...)` now creates the patient with the episode.
- The audit journey no longer uses the removed `POST …/patients`.

## Decided by the builder (automatic mode)

- **The national ID scheme is a per-deployment setting, `NATIONAL_ID_TYPE`** (default `nhs` locally), with the label sent in `GET /api/config`. **Reason:** A-20 (no NHS-only identifier hardcoded); saas-requirements §13 already lists "Patient identifier scheme: env var".
- **Mobile numbers use `github.com/nyaruka/phonenumbers`** (the Go port of Google's libphonenumber) to turn what staff type into E.164. It checks the number's **length for its country**, not that it's a real, allocated mobile: Ofcom's made-up 07700 900xxx numbers must pass, and libphonenumber calls them invalid. **PAT-11's "mobiles in the deployment's country only"** is left to sending (S14). **Reason:** every country's trunk prefix and calling code, with no table of our own.
- **Linking never changes the trust's record.** What staff typed (another first name, another mobile) is not written over the stored details, and a record without an NHS number doesn't gain one from a link. **Reason:** a silent link must not silently edit someone else's data; updating contacts is PAT-10 (S15).
- **After a silent link, the episode and patient pages show the trust's record** (its first name and contacts), as the S08 answer says (a hospital's staff see the trust's patient record, but only their hospital's episodes). Nothing about it is shown **before** the episode exists (the create response is the episode id only), and nothing about other hospitals' episodes or hospital numbers ever is.
- **"This hospital's patient" = at least one episode at this hospital** (C2's design, confirmed in answers 3). A patient chosen by id must be one; anyone else is 400.
- **A typed NHS number the trust doesn't have** still goes through the name + date of birth + contact rules, but only against records **without** an NHS number (a record with a different NHS number is someone else).
- **Several possible-duplicate matches** at this hospital are all listed; linking by contact picks the oldest matching record.
- **The search's hospital number is an exact match** (any case); the surname is a prefix; `%` and `_` are matched as text.
- **The hospital number is limited to 50 characters**, names to 100.
- **`createAnyway` doesn't override a link or a mismatch:** a matching NHS number (or name + date of birth + contact) always links, and an NHS-number mismatch is always refused.

## Differences from the plan

- C2's API listed `GET /api/h/{hid}/patients?q=`; it's `GET /api/o/{oid}/h/{hid}/patients?q=&dob=` (S08's URL shape), with the date of birth as its own parameter rather than parsed out of one text box (dates are typed in the browser's own format).
- Added `GET …/patients/{pid}` for the brief's patient page.
- The duplicate rule extends the plan (Rahul's answer 2): name + date of birth + the same email or mobile links silently.

## Tests

- **Unit:** `internal/nationalid` (Modulus 11, including check digits 0 and 10), `internal/patient` (`Mobile`, `Email`, `NameKey`, `Decide`: 13 cases, `ParseSearch`), `internal/config` (`NATIONAL_ID_TYPE`), `internal/httpapi` (`details`); web `newEpisode.test.ts`, `config.test.ts`, `routes.test.ts`.
- **Integration (API → database), new in `patients_integration_test.go`:**
  - `TestSearchPatients`: NHS number with spaces (and exactly the fields returned), hospital number, surname prefix (this hospital's only), short surname + date of birth; a patient known only at A2 is never found at A1 (but is at A2); 20 + `more`; no sign-in, wrong role (A2 clinician at A1: 404), another trust (and trust B finds its own record with the same NHS number); too broad, bad date, too long, bad ids; `%`/`_` not wildcards.
  - `TestGetPatient`: the record and A's episodes only (A2's never in the body); a patient without an episode here is 404; the four refusals.
  - `TestCreateEpisodeForANewPatient`: the five cases; 14 invalid inputs (bad check digit names the NHS number, bad mobile/email, both or neither patient forms…) writing nothing; values cleaned up; no NHS number or contacts (PAT-01a); a second open episode (PAT-02); a found patient must be this hospital's.
  - `TestCreateEpisodeMatchesTheTrustsRecord`: links silently to A2's patient (no new record, no stored value in the response, the record unchanged, A's search now finds them by A's number); mismatch on date of birth or surname (409 `mismatch`, no stored values, nothing written); same NHS number in another trust is a new record; name + date of birth + mobile / + email link; a possible duplicate known elsewhere names nobody (also for a record with no episodes) and Create anyway creates; one at this hospital is shown; the concurrent-insert race.
  - Updated: `TestAuditRowsWritten` (search, view, create with typed details, a possible duplicate records nobody), `TestListAudit`, `TestAuditListCapsPatientsPerRow` (now through the episode list), `TestAuditFailsClosed`, `TestHospitalScopedClinicianSeesOnlyTheirHospital`, `TestConfig`, `TestPatientRouter`. Removed `TestListPatients` and `TestCreatePatient` (their routes are gone).
- **Results (13 Oct):** `make lint` passed; `make test`: Go ok, Vitest 98 + 116 passed; `make test-integration`: all packages ok; `make e2e`: **32 passed** (104 s).

## Round 1 (test report 1: 21 passed, 1 failed): fixes, each test-first

- **F-1, no questionnaire said "The request could not be read."** The page sends `"versionId": ""`, which failed JSON decoding into a UUID. `createEpisode` now reads `versionId` as text and parses it, so empty, missing or malformed all get **"Choose a published HQ."** Test: `TestCreateEpisodeForANewPatient/invalid_input` (written first, failed with the old message).
- **Exploring 1, Change patient kept the previous patient's hospital number.** It now empties the Episode's Hospital number (the number was that patient's). The search starts afresh too. Tests: Vitest `hospitalNumberOnChoose`; the search journey checks the field empties.
- **Exploring 2, New episode from a patient's page didn't fill the hospital number.** It now uses the patient's newest hospital number here (from `GET …/patients/{pid}`'s episodes), in the card and the field, like choosing from the search. Tests: Vitest `latestHospitalNumber`; the search journey opens New episode from the patient page.
- **Exploring 6:** with a date of birth set, the "first 20" line no longer suggests adding one.
- **Exploring 5:** the test plan's T-05 hint now uses surnames (`Aud`, `J` + date of birth).
- **Logged in `docs/pending.md`, not changed:** the by-design date-of-birth guessing exposure (for Rahul to decide; a per-user mismatch limit suggested), rapid synthetic clicks sending several POSTs, and the cosmetic items 7–8. Item 9 (a non-UK mobile accepted) is deliberate until S14 (PAT-11).
- **Rahul confirmed** both builder choices during the round (answers 4–5): keep `nyaruka/phonenumbers`, and a silent link keeps the stored record. Recorded in decisions.md.
- **Results:** `make lint` passed; `make test`: Go ok, Vitest 100 + 116; `make test-integration`: ok; `make e2e` (after `make seed`): 31 passed + the fixed journey re-run, `test_patient_records.py` 4 passed.

## Round 2 (test report 2: all passed): exploring items fixed, each test-first

- **After Change patient the address kept `?patient=<id>`,** so a reload or Back/Forward chose the old patient again (risk: the wrong patient). Change patient now removes it (`replace`, so no history entry keeps it). The New episode page is now keyed by the hospital only, so dropping the parameter doesn't wipe what was typed in the Episode card. Test: the search journey checks the address, a reload with no patient chosen, and Back.
- **The patient page highlights Episodes in the header** (`HospitalLayout.tsx`, `nav(…, also)`), and switching hospital from a patient's page goes to the other hospital's Episodes (`switchPath`; Vitest first, which failed with "/questionnaires"). Test: the journey checks the header link's variant.
- **"Patient not found." has a ← Episodes link** (`PatientPage.tsx`).
- **Logged in pending.md, not changed:** Change patient also empties a typed hospital number (the coordinator's call), and the search table has no visible heading.
- **Test plan:** new case T-18.
- **Results:** `make lint` passed; `make test`: Go ok, Vitest 100 + 116; `make test-integration`: ok; `make e2e` (after `make seed`): **32 passed**.

## Round 3 (test report 3: all 3 re-run cases passed)

- No code changes. Its "found while exploring" items (a search choice not kept in the address, the position of "Patient not found." on `?patient=`, Cancel going to Episodes) are cosmetic and logged in `docs/pending.md`; the double-click probe made exactly one episode.
- S13 passed manual testing; next: verification.

## Hand checks for Rahul

- **T-17 in a real 390×844 window** (Chrome's device toolbar or WebKit): the tester used a same-size iframe. J3 runs it in WebKit's iPhone 13 profile automatically.
- **R-04:** that clicking **Export CSV** saves a file (the endpoint returned a valid CSV).

## Suggested commits (Rahul commits after verification)

1. **Patient records: national ID, NHS number check, search, matching and the API (C2 / S13: data and API)** — `apps/api/**` (migration 00010, `internal/nationalid`, `internal/patient`, `internal/config`, `internal/httpapi`, queries, sqlc output, seed, `go.mod`/`go.sum`), `.env.example`, with their unit and integration tests.
2. **The New episode page, the patient page and journeys (C2 / S13: web and journeys)** — `apps/web/**`, `e2e/**`.
3. **Docs: S13 brief, answers, build notes, test plan, reports, verification and status** — `docs/sessions/S13-patient-records/`, `docs/plans/`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`, `e2e/README.md`, `apps/web/AGENTS.md`.
