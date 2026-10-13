# C1: Trusts and staff

**Goal:** real tenancy and real staff sign-in. A platform admin creates a trust and its hospitals and invites the trust's first admin. The admin invites staff, and staff sign in through Cognito with MFA.

**Requirements:** ORG-01…06, STF-01…06, A-5, A-12, A-20 (nothing hardcoded for the UK), A-21 (superadmin). **Local first (12 Oct, A-22):** S08–S11 run locally with the dev login. Cognito (S12) comes in the AWS phase.

**Depends on:** F1–F4.

**Not in C1:** patients (C2), and other countries. Only the config groundwork is built (A-20).

## Data (a reshape is fine, because it's a fresh app: A-1)

- `orgs`: id, name, code, status, `settings jsonb`. *(The `data_region` column was dropped on 8 Oct, because each country is its own deployment: A-20.)*
- `hospitals`: + `org_id`, status, `settings jsonb`, and **`timezone`** (defaults to the deployment's; *Proposed*, A-20).
- `users`: + `cognito_sub`.
- `memberships`: user_id, org_id, `hospital_id NULL` (NULL = the whole trust), `role` ∈ `clinician < super_clinician < admin`. This replaces today's roles.
- `platform_admins`, `staff_invites`, and `audit_log` (append-only).
  - **`platform_admins` is filled by hand (A-21):** only Rahul, with no screen. A small command (`cmd/platform-admin add <email>`) creates the Cognito user and the row. Run it once in each environment and country. Locally, the seed adds a dev superadmin.
- Questionnaires and other content get an `org_id` + `hospital_id NULL` owner.

## Design

- **Tenant isolation:** every patient-data query is scoped by `org_id`, and the "another trust's data" integration case proves it on every route. *(`store.ForOrg` and its region routing were dropped on 8 Oct: A-20.)*
- **Nothing hardcoded for the UK (A-20):**
  - per-country values (web address, Cognito pool and client, time zone, locale) come from `internal/config`;
  - the web app reads them from `GET /api/config`;
  - the rules are in [§13 Countries](../saas-requirements.md).
- **Routes move** to `/api/o/{oid}/…`, with `hospital_id` on rows. A hospital-scoped user sees only their hospital; a trust-scoped user sees every hospital in the trust. The frontend becomes `/o/:oid/h/:hid/…`.
- **Staff sign-in:**
  - A Cognito staff user pool (CDK `AuthStack`): email as the username, **TOTP MFA required**, no self sign-up. Cognito's **managed login pages** handle the password, MFA and reset screens, so we build none of them.
  - The frontend redirects there (OAuth code flow with PKCE). The callback page posts the code to `POST /api/auth/callback`.
  - Go exchanges the code, verifies the ID token, maps `sub` → user, and sets **our own HTTP-only session cookie**. That's the same cookie model patients will use, and no API Gateway authorizer is needed.
  - Each request re-reads the membership, so removing access takes effect at once (STF-04).
- **Invites:** `POST /api/o/{oid}/staff/invites` creates the Cognito user (`AdminCreateUser`, behind a `Deps.Cognito` interface with a fake in tests) and the membership. If the email already exists, it only adds the membership (STF-02).
- **Settings:** `settings.Effective(platformDefaults, org, hospital)`. Only keys on the "hospital can override" list come from the hospital. This includes the opt-ins (ORG-05). Written TDD.
- **The dev login** stays for local development only. `DEV_LOGIN` is turned off in QA once C1 ships.

## API (each with the five integration cases)

- Platform admin: `POST/GET /api/admin/orgs`, `POST /api/admin/orgs/{oid}/hospitals`, `PATCH …` (suspend, opt-ins).
- Trust admin:
  - `GET/PATCH /api/o/{oid}/settings`
  - `GET /api/o/{oid}/staff`, `POST …/staff/invites`, `PATCH/DELETE …/staff/{uid}`
  - `GET /api/o/{oid}/audit` (export as CSV)
- `GET /api/me`: the user's memberships.

## Screens

- Platform admin: Trusts list → New trust (name, code, contact, logo, first admin email, opt-ins) → Hospitals (with a time zone, defaulting to the deployment's).
- Hospital picker, and a header showing the trust and hospital.
- Staff list and invite.
- Trust settings, with per-hospital overrides.

## Tests

- **Integration, beyond the standard five:**
  - an admin of trust A gets 404 for anything of trust B;
  - a hospital-scoped clinician can't see another hospital's episodes;
  - the role ladder (a clinician can't publish; a super clinician can't invite);
  - inviting an existing email adds a membership only;
  - a removed membership → 403 on the next request;
  - audit rows are written for patient-data reads;
  - a suspended trust → sign-in refused.
- **Unit:**
  - effective settings;
  - the role ladder;
  - ID-token verification (with a fake key set);
  - `internal/config` refuses to start outside local dev when a per-country value is missing.
- **End-to-end:**
  - **J6:** the platform admin creates a trust and hospital, and invites an admin; the admin invites a clinician; the clinician signs in.
  - **J1–J5** move to `/o/…` URLs.
  - In QA, the login helper signs in through Cognito's managed login, using test users whose TOTP secrets are known to the test (`pyotp`). Locally it uses the dev login.

## Status

**In progress: S08 (trusts, hospitals, memberships, role ladder, hospital picker) verified 13 Oct 2026 (test report 4 all passed; [verify.md](../sessions/S08-trusts-permissions/verify.md)). S09 (staff list, invites, change role, remove access) verified 13 Oct (test report 2 all passed; [verify.md](../sessions/S09-staff-invites/verify.md), [build notes](../sessions/S09-staff-invites/build-notes.md)). S10 (superadmin console, suspend, opt-ins, settings, editing details) tested 13 Oct (test report 2 all passed), verification 1's fixes retested (test report 3 all passed; [build notes](../sessions/S10-superadmin-console/build-notes.md)), verified 13 Oct ([verify.md](../sessions/S10-superadmin-console/verify.md), verification 2).** **S11 (audit log and CSV export) tested 13 Oct (test report 1 all passed); verification 1's and test report 2's fixes retested (test report 3 all passed); verified 13 Oct** ([verify.md](../sessions/S11-audit-log/verify.md), verification 2; [build notes](../sessions/S11-audit-log/build-notes.md)). S12 (Cognito, AWS phase) comes later.

Differences from the plan so far (S08; details in [decisions.md](../decisions.md) §2 and the [S08 build notes](../sessions/S08-trusts-permissions/build-notes.md)):
- Hospital routes are `/api/o/{oid}/h/{hid}/…`; the only trust route so far is `GET /api/o/{oid}/episodes` ("All hospitals").
- A removed membership gets **404** on the next request, not 403: non-members can't probe trusts or hospitals (NFR-04).
- `hospitals.timezone` NULL means the deployment's default.
- Patient records moved to the trust in S08 (Rahul's answer, 12 Oct), ahead of C2.
- The trust's questionnaires and option lists can be changed only by whole-trust super clinicians and admins (Rahul, 12 Oct).

Differences and choices in S09 (details in [decisions.md](../decisions.md) §2, 13 Oct "S09"):
- Only **whole-trust admins** manage staff; an admin of one hospital can't.
- A person can hold **several memberships in one trust** (Rahul, 13 Oct). Each is changed or removed at `PATCH/DELETE …/staff/{uid}/access/{scope}` (`scope` = `trust` or a hospital id); `DELETE …/staff/{uid}` removes them all. The plan's `PATCH …/staff/{uid}` became the per-membership PATCH.
- The membership is added when the invite is made. The 7-day expiry is the deadline for a **new account's first sign-in** (as Cognito's temporary password); after it, sign-in is refused until the admin resends the invite.
- The last whole-trust admin can't be removed or demoted. Emails are unique whatever their case.

Differences and choices in S10 (details in [decisions.md](../decisions.md) §2, 13 Oct "S10"):
- Settings are `internal/settings` (`Defs`, `Effective`); **every ORG-04 setting** exists now (Rahul, 12 Oct). The opt-ins are set by the platform admin only (Rahul, 13 Oct).
- The platform admin's `PATCH` routes also **edit a trust's and a hospital's details** at any time (Rahul, 13 Oct). Two extra routes: `POST /api/admin/orgs/{oid}/admins` and `GET /api/admin/opt-ins`.
- Trust settings (`GET/PATCH /api/o/{oid}/settings`) are for whole-trust admins only, as Staff is.
- Suspension is enforced: no new sign-in when all of a person's memberships are suspended, no access on the next request, and the patient's link closed.

Differences and choices in S11 (details in [decisions.md](../decisions.md) §2, 13 Oct "S11"):
- One `audit_log` row per patient-data **request** (Rahul, 12 Oct), written by the `audited` middleware with IDs only; "no row, no data" (the response is held until the row is written).
- `GET /api/o/{oid}/audit` is for whole-trust admins only; the CSV is `?format=csv` on the same route. Reading and exporting the log are logged too.
- The patient's own link is logged (actor `patient`). Refusals after the membership check are logged with their status; earlier ones aren't.
