# C1: Trusts and staff

**Goal:** real tenancy and real staff sign-in. A platform admin creates a trust and its hospitals and invites the trust's first admin. The admin invites staff, and staff sign in through Cognito with MFA.

**Requirements:** ORG-01…06, STF-01…06, A-5, A-12, A-20 (nothing hardcoded for the UK), A-21 (superadmin). **Depends on:** F1–F4.

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

Not started.
