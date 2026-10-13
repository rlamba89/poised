# S10 build notes: the superadmin console, trusts, hospitals and settings

Builder: automatic run (`/run-sessions`), 13 Oct 2026. Answers: [answers.md](answers.md). Both widen the brief at Rahul's choice: **a logo URL field now** (upload in C8), and **every ORG-04 setting now**, trust-only and hospital-overridable.

## What was built

### Data (migration `00008_platform_admin.sql`; up, down, up checked on the local database)
- `orgs`: `contact_name`, `contact_email`, `contact_phone` (who Poised deals with at the trust; never shown to patients). Codes are unique whatever their case (`orgs_code_upper_key`).
- `hospitals`: `address`. Hospital names are unique within a trust, whatever their case (`hospitals_org_name_key`).
- Everything else a trust or hospital has is a **setting**, stored as a flat JSON object in the existing `orgs.settings` / `hospitals.settings` (`{"contact.phone": "0100 …", "feature.proms": true}`). A key that isn't stored is "not set here".

### Go
- **`internal/settings`** (written test-first): `Defs` (every ORG-04 setting with its label, group, level, kind, default and limits), `Effective(platform, org, hospital)`, `Validate(key, value)`, `Platform()`, `Decode`, `Apply`, `IsOptIn`, `PhoneOK`.
  - `Effective`: the hospital's own value where the setting is hospital-overridable and the hospital has set it, else the trust's, else the platform default. Trust-only keys ignore any hospital value; unknown keys are ignored.
  - **The settings** (defaults in brackets):
    - trust-only: `retentionYears` (8), `crossTrustPrefill` (off);
    - opt-ins (ORG-05), hospital-overridable, **set by the platform admin only**: `feature.assessments` (on), `feature.proms` (off), `feature.files` (on), `feature.observations` (on), `instrument.eq5d5l`, `instrument.oxfordHip`, `instrument.oxfordKnee` (all off);
    - hospital-overridable: `branding.logoUrl` (""), `contact.phone`, `contact.email` (""), `defaultHq` (none), `whatHappensNext` (a default text), `reminders.days` ([2, 5]), `reminders.notRespondingDay` (7), `autoArchive.afterDischargeDays` (30), `autoArchive.inactiveMonths` (6), and the patient message templates `templates.inviteSms`, `templates.inviteEmail`, `templates.reminderSms`, `templates.reminderEmail` (default texts).
  - Templates may use only `{{firstName}}`, `{{hospital}}` and `{{link}}`, and must include `{{link}}`. Logo addresses must be http(s). Phones are checked for shape only (digits, spaces, `+ ( ) -`), with no country's numbering plan (A-20).
- **Routes** (each with the five integration cases; `internal/httpapi/admin.go`, `trust_settings.go`):
  - Platform admins only (`requirePlatformAdmin`, re-read on every request; anyone else signed in 403, nobody 401):
    - `GET /api/admin/orgs`: every trust with its hospital count. No patient data (OPS-06).
    - `POST /api/admin/orgs` `{name, code, contactName, contactEmail, contactPhone?, logoUrl?, adminEmail, adminName?, optIns?}` → 201: the trust, plus the first admin invited by **S09's invite** (now `inviteTo` in `staff.go`): a whole-trust admin, Cognito (local stand-in) called last in the transaction. A duplicate code → 409.
    - `GET /api/admin/orgs/{oid}`: the trust, its hospitals (own and effective opt-ins, time zone), its whole-trust admins with invite status, and the opt-in definitions.
    - `PATCH /api/admin/orgs/{oid}` `{status?, optIns?}`: suspend or re-activate; set the trust's opt-ins (null = back to the default).
    - `POST /api/admin/orgs/{oid}/hospitals` `{name, address, phone?, email?, logoUrl?, timeZone?, optIns?}` → 201. Phone, email and logo become the hospital's own settings; empty follows the trust. `timeZone` "" = the deployment's; otherwise an IANA zone from Go's built-in database ("Local" refused). Duplicate name in the trust → 409.
    - `PATCH /api/admin/orgs/{oid}/hospitals/{hid}` `{status?, optIns?}` (null = follow the trust). Another trust's hospital → 404.
    - `POST /api/admin/orgs/{oid}/admins` `{email, name?}` → 201: invite another whole-trust admin (an expired first admin, or a trust left without one).
    - `GET /api/admin/opt-ins`: the opt-in definitions, for the New trust form.
  - Whole-trust admins only (like Staff, S09): `GET /api/o/{oid}/settings` (definitions, the trust's values and effective values, each hospital's own and effective values, the trust's published questionnaires for the default HQ) and `PATCH /api/o/{oid}/settings` `{hospitalId: null|id, values: {key: value|null}}`. Every change is checked before anything is written: unknown key 400, opt-in 403, trust-only key for a hospital 400, bad value 400, a default HQ that isn't one of the trust's published questionnaires 400, another trust's hospital 400. `hospitalId` is required (a missing one is never read as the whole trust, as in S09).
- **Suspension (ORG-03) is enforced** (S08 stored it only):
  - `RolesAtHospital`, `ListOrgMemberships` and `ListMyHospitals` leave out suspended trusts and hospitals, so a signed-in member's next request is 404 and `/api/me` drops them.
  - **Sign-in** (the dev login; S12's Cognito must do the same) is refused, 403 "Your trust or hospital is suspended…", when someone has memberships and every one is in a suspended trust or hospital (`SignInState`). Someone with another active trust signs in and keeps it.
  - **The patient's link** answers 403 "This questionnaire isn't available at the moment…" while its trust or hospital is suspended.
  - A suspended hospital's episodes leave "All hospitals". Nothing is deleted.
- **`/api/me` lists a trust that has no hospitals yet** (pending.md §3, S08's verification): `ListMyHospitals` left-joins hospitals; the trust comes back with `hospitals: []`.
- **`cmd/platform-admin add <email> [name]`** (A-21) and `make platform-admin EMAIL=… NAME=…`: creates the account if there's none, adds the `platform_admins` row and, for a new account, calls Cognito (`cognito.Local` until S12). Idempotent; a Cognito failure leaves nothing.
- **Seed:** each run puts the demo trusts and hospitals back to active and gives them contacts and addresses; their settings are left alone. Sam Superadmin was already seeded.

### Web app
- **`/admin`** (trusts list), **`/admin/trusts/new`** (New trust), **`/admin/trusts/:orgId`** (the trust) inside `pages/admin/AdminLayout.tsx`, which shows "Only Poised's platform admins can open this page" to anyone else (and asks the admin API nothing).
- **`/o/:orgId/h/:hospitalId/settings`**: the trust's Settings, linked in the header for whole-trust admins (`canManageSettings`). Switching hospital on Settings stays on Settings.
- Sign-in sends a platform admin with no hospital to `/admin`; the hospital picker shows a platform admin a link to the console, and a trust with no hospitals yet with a note. A platform admin who is also staff sees "Platform admin" in the hospital header.

**TypeScript for Rahul to review:**
- `src/lib/auth.tsx`: `startPath` (platform admin with no hospital → `/admin`), `canManageSettings` (= `canManageStaff`: the trust-wide role must be admin).
- `src/pages/admin/AdminLayout.tsx`: the header and the platform-admin check (from `/api/me`'s `platformAdmin`).
- `src/features/admin/`: `types.ts` (the JSON shapes of `admin.go`), `TrustList.tsx` (a table), `NewTrustForm.tsx` (one form; the opt-in checkboxes start at each default from `GET /api/admin/opt-ins`), `TrustPage.tsx` (one `change()` helper sends every PATCH/POST and shows the trust it returns; `OptInTable` has a switch per opt-in for the trust and a Follow trust / On / Off select per hospital, each saved at once; `AddHospitalForm` lists the browser's IANA zones with "Default (<deployment zone>)" first, so no zone is written in the code).
- `src/features/settings/SettingsPage.tsx`: draws one field per setting **kind** from the definitions Go sends, so it has no list of settings of its own. Each editable setting has a "Use the trust's setting" (or, for the trust, "Use Poised's default") tick box; Save sends only what changed (`null` for a setting put back to following). Opt-ins and, at a hospital, trust-only settings are shown as badges.

## Decided by the builder (automatic mode)
1. **The opt-ins are set only by the platform admin**; trust admins see them on Settings but can't change them (the API answers 403). PRM-08 says the platform admin sets them, and they can be sold as add-ons.
2. **Only whole-trust admins see and change Settings**, as for Staff (S09); a hospital-only admin gets 403. Follows Rahul's S08 rule that trust-level things need a whole-trust role.
3. **Defaults** as listed above: the core features (assessments, files, observations) on; PROMs and the licensed instruments off (opt-in); EPI-05/06 and C2's numbers for the rest.
4. **The trust's contact (name, email, phone) is for Poised only** (columns). The **patients'** phone and email are the `contact.*` settings: the trust's default, which a hospital overrides; the Add hospital form's phone and email set the hospital's own.
5. **The logo is the `branding.logoUrl` setting**: the trust's (New trust form, Settings) and optionally a hospital's (Add hospital, Settings). http(s) only.
6. **Suspension**: sign-in is refused only when **all** of someone's memberships are suspended; signed-in people lose access on their next request (404); the patient's link is closed (403); a suspended hospital's episodes leave "All hospitals". A suspended hospital is hidden from everyone, whole-trust staff included.
7. **Two small routes beyond the brief:** `POST /api/admin/orgs/{oid}/admins` (re-invite a first admin whose invitation expired, or recover a trust without a working admin, as S09's verification expected) and `GET /api/admin/opt-ins` (the New trust form's checkboxes come from Go, not a copy in TypeScript).
8. **Codes** are 2–12 letters, digits or hyphens, kept in capitals, unique whatever their case. **Hospital names** are unique within a trust. **The address is required**; a hospital's phone and email are optional (it follows the trust's).
9. **Message templates**: four now (invitation and reminder, each by text and email). Only `{{firstName}}`, `{{hospital}}`, `{{link}}`; `{{link}}` is required. Email subjects and EPI-08's per-status templates are left to the sessions that send them (S14–S16, C3).
10. **The default HQ** must be one of the trust's questionnaires with a published version.
11. **`cmd/platform-admin`'s test uses the apitest database harness** (`cmd/platform-admin/main_integration_test.go`). Strictly the rules allow only API → database integration tests; the command has no API, and a database test was the only honest check. It also signs the new admin in through the API.

## Differences from the plan
- C1 lists `PATCH …` for suspend and opt-ins: built as `PATCH /api/admin/orgs/{oid}` and `PATCH /api/admin/orgs/{oid}/hospitals/{hid}`.
- Not built (follow-ups in [pending.md](../../pending.md) §3): a licence-holder note per instrument (PRM-08); promoting someone who already has another whole-trust role from the console (409 today). (Editing details was added in round 1, Rahul's answer 3.)
- Every `make e2e` run adds three or four "E2E Trust …" trusts and "Zoe E2E …" accounts; trusts can't be deleted. Reset the local database when the lists get long.

## Tests
- **Unit:** `internal/settings` (`TestEveryKeyHasAValidDefault`, `TestEffective` incl. ORG-05 inheritance both ways, `TestValidate` 17 good / 28 bad values, `TestOptInKeys`, `TestDecodeAndApply`); web `auth.test.ts` (start path, empty trusts, who changes settings), `routes.test.ts` (four new URLs; the admin layout; `/admin/nope` and `/admin/trusts` not found).
- **Integration (written first, failing):**
  - `admin_integration_test.go`: `TestAdminListTrusts` (counts, no patient data, 401, 403 for a trust admin and a clinician, a removed platform-admin row → 403 next request), `TestAdminCreateTrust` (settings stored, first admin invited with a 7-day invite and one `CreateUser`, that admin's `/api/me` lists the trust with no hospitals; existing account → membership only; duplicate code any case → 409 with nothing written; 401; 403; 16 invalid inputs + broken JSON; Cognito failing → 500, nothing written), `TestAdminGetTrust`, `TestAdminAddHospital` (own time zone; duplicate name 409, same name in another trust fine; 11 invalid inputs), `TestAdminSuspendTrust` (next request 404, sign-in 403 with no cookie, patient link 403, a two-trust member keeps the other, data kept; re-activate), `TestAdminSuspendHospital` (hospital-only member refused; whole-trust admin keeps A2, loses A1, All hospitals drops A1's episodes; another trust's hospital 404), `TestAdminOptIns` (trust off/hospital on and the reverse, null resets, status + opt-ins together, invalid), `TestAdminInviteTrustAdmin`, `TestAdminOptInDefs`, `TestPlatformAdminSeesNoPatientData` (every trust/hospital route 404 for the superadmin).
  - `settings_integration_test.go`: `TestGetTrustSettings`, `TestPatchTrustSettings` (trust values followed by both hospitals; a hospital's own; null back to the trust; trust-only for a hospital 400; opt-ins 403; another trust's admin 404, its hospital or questionnaire 400; invalid inputs write nothing).
  - `routers_integration_test.go`: the patient router doesn't serve `/api/admin/…` or settings; the staff router does.
  - `cmd/platform-admin`: new email, idempotent re-run, existing account, invalid emails, Cognito failing.
- **End-to-end (`e2e/journeys/test_j6_onboarding.py`, 5 journeys):** **J6** (Sam creates a trust and a hospital in the UI and invites an admin; the admin, in their own browser, lands in the hospital, sees Staff and Settings, invites a clinician; the clinician signs in to Episodes; Sam sees the admin Active); suspend → the admin's sign-in is refused with the message → re-activate → they sign in; a trust admin can't reach `/admin`; a duplicate code is refused in the form; a trust admin sets the trust's phone and gives one hospital its own (kept after reload; the other follows; opt-ins shown as set by Poised). `support.Api` takes `hospital=None` for the platform admin and gained `new_trust()` and `new_hospital()`.

Results (13 Oct, 02:2x):
- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web 74 passed, clinical 116 passed.
- `make test-integration`: all packages ok (incl. `cmd/platform-admin`).
- `make e2e` against `make dev`: **22 passed** (J1–J5, trusts, staff, J6).

## Round 1 (test report 1: 18 of 19 passed; T-15 failed with two Minor issues) and Rahul's answers 3–4

**Fixes, each with a failing test first:**
- **F1: at a hospital, the trust-only settings were missing.** `SettingsPage.tsx` skipped a group whose settings were all trust-level. Now the "Whole trust" group is shown at a hospital, with "Set for the whole trust" badges. Test: J6's settings journey checks the retention badge (and that no input is there).
- **F2: the "saved" message appeared ~2,000px above the Save button**, and an old "saved" stayed beside a later "Nothing has changed.". The Settings result is now one message, shown beside Save (kept in `SettingsPage` because the form starts afresh after a save). On the console's trust page each change's message is shown by the part it came from: the top (trust actions), the Hospitals table, or the opt-ins table. Tests: the journeys check each message is in the viewport, and that "Nothing has changed." replaces "saved".

**Found while exploring, taken:**
- The console's "already has that access" refusal said "Change it in the list instead.", which the console hasn't got. It now says "… already has access to the whole trust. The trust's admins can change their role on the Staff screen." (`inviteTo` takes the refusal's wording). Test: `TestAdminInviteConflictMessage`.
- The start page said "Choose a hospital" when the user's trusts have none; now "Your trusts".
- Switching hospital from a page you were refused (Max on Trust A's Settings → Hospital B1, where he's admin) went to Questionnaires. The rule is now one tested function, `switchPath` in `src/lib/auth.tsx`: Staff and Settings are kept wherever the user manages the target trust, Episodes and Questionnaires as before. Tests: `auth.test.ts`, "switching hospital from the header".
- The default-HQ list can be searched.
- The S01 editor's title saving on blur was seen again; it's already in pending.md (S02a), and that entry now says so.

**Test data:** `make seed` now also resets the demo trusts' and hospitals' settings (to `{}`, Poised's defaults) and the hospitals' time zones, as it resets their status, names and codes, so each round starts clean.

**Rahul's answer 3: the platform admin edits trust and hospital details at any time (built):**
- `PATCH /api/admin/orgs/{oid}` also takes `name`, `code`, `contactName`, `contactEmail`, `contactPhone`, `logoUrl`; `PATCH /api/admin/orgs/{oid}/hospitals/{hid}` also takes `name`, `address`, `timeZone`, `phone`, `email`, `logoUrl`. Only the fields sent change, checked with the same rules as creating; "" for a logo, phone or email removes it (a hospital then follows the trust's), and "" for a time zone is the deployment's. A code or hospital name already taken → 409. A status-only PATCH isn't held up by details from before S10 (a hospital with no address).
- Tests (written first): `TestAdminEditTrustDetails`, `TestAdminEditHospitalDetails`: happy path (fields trimmed, the rest unchanged, opt-ins kept), the duplicate (409; its own code or name in another case is fine), no sign-in, wrong role (a trust admin), another trust's hospital (404), invalid input (nothing changes). E2E: `test_the_platform_admin_edits_a_trusts_and_a_hospitals_details`.
- UI: **Edit details** beside Suspend on the trust page, and **Edit** on each hospital's row, open filled-in dialogs (`TrustForm`, and `HospitalForm`, which Add hospital now shares).

**Rahul's answer 4:** opt-ins stay platform admin only, as built.

**Test plan:** updated for the tester's two deviations (T-05 step 7 uses another trust than Trust B; T-08 uses Zoe Clinician One rather than Cara, so T-13 step 3 works), the fixes, and new cases T-20 and T-21.

Results (13 Oct, round 1):
- `make lint`: clean. `make test`: Go ok; Vitest web 77, clinical 116 passed.
- `make test-integration`: all packages ok.
- `make e2e` against `make dev`: **23 passed**.

## Round 2 (test report 2: all passed)

Every case re-run passed, including T-15 (F1, F2), T-20 and T-21. No code changed after it. Its three "found while exploring" items (the logo message says https:// but http:// is accepted; the dev sign-in refusal shown off-screen; an explicit `false` stored when a trust's opt-in is turned back off) are logged in [pending.md](../../pending.md) §3.

## Verification 1 (Needs fixes): fixes, each test-first

1. **A platform admin who is also staff of a suspended trust was refused sign-in**, so couldn't reach `/admin` to re-activate it. `devLogin` now calls `signInAllowed`, which lets platform admins in; their suspended trusts still stay out of `/api/me` and every trust and hospital route. Test: `TestAdminSuspendTrust`, "a platform admin who is also its staff still signs in, and can re-activate it" (failed with 403 first).
2. **A trust whose hospitals are all suspended was shown as having "no hospitals in Poised yet"** to its whole-trust staff, who could still sign in, while its hospital-only staff were refused. Now `ListMyHospitals` lists a trust with no hospital rows only when it has **no hospitals at all**, and `SignInState` counts a whole-trust membership as usable only when the trust has an active hospital or none yet. So both kinds of staff are treated alike: left out, and refused sign-in if they have nothing else. Test: `TestAdminSuspendHospital`, "a trust whose only hospital is suspended is left out…" (failed first). The start page's "no hospitals yet" note is now only ever shown for a new trust.
3. **`make platform-admin` broke on a name with an apostrophe:** `$(NAME)` is now quoted. Checked: `make platform-admin EMAIL=… NAME="Sean O'Brien"` created "Sean O'Brien" (the check account was then deleted).

Findings 4–9 are logged in [pending.md](../../pending.md) §3. The test plan has new cases T-22 and T-23 for fixes 1 and 2.

Results (13 Oct, after the fixes): `make lint` clean; `make test` Go ok, Vitest web 77 and clinical 116 passed; `make test-integration` all ok; `make e2e` **23 passed**.

## Retest after verification 1 (test report 3: all passed)

T-22, T-23, T-12 and T-13 passed (R-1 rechecked). No code changed after it. Its two small items (the start page's wording for a trust whose hospitals are all suspended; the console header's "Your hospitals" link not refreshed after suspend or re-activate) are logged in [pending.md](../../pending.md) §3.

## Hand checks for Rahul
- None left: the verifier ran `make platform-admin` (first run, second run, no EMAIL). Run it once for your own email when you want your own superadmin account.

## Suggested commits (Rahul commits after verification)
1. **Settings, the superadmin console API and suspension (C1 / S10: data and API)** (with round 1's editing of details): `apps/api/db/migrations/00008_platform_admin.sql`, `apps/api/db/queries/{admin.sql,users.sql,episodes.sql}`, `apps/api/internal/db/*` (generated), `apps/api/internal/settings/`, `apps/api/internal/httpapi/{admin.go,trust_settings.go,staff.go,auth_handlers.go,patient.go,router.go,admin_integration_test.go,settings_integration_test.go,routers_integration_test.go}`, `apps/api/internal/apitest/fixtures.go`, `apps/api/cmd/platform-admin/`, `apps/api/cmd/seed/main.go`, `Makefile`.
2. **The admin console and Settings screens (S10: web and journeys)**: `apps/web/src/**` (auth, config, routes, Home, layouts, `features/admin/`, `features/settings/`, `pages/admin/`, `pages/hospital/Settings.tsx`), `apps/web/AGENTS.md`, `e2e/support.py`, `e2e/conftest.py`, `e2e/journeys/test_j6_onboarding.py`, `e2e/README.md`.
3. **Docs: S10 decisions, build notes, test plan and status**: `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S10-superadmin-console/*`, `CLAUDE.md`, plus S10's share of the 12 Oct planning-doc changes (its `brief.md`, `answers.md`).
