# C1: Trusts and staff

**Goal:** real tenancy and real staff sign-in. A platform admin creates a trust and its hospitals and invites the trust's first admin. The admin invites staff, and staff sign in through Cognito with MFA.

**Requirements:** ORG-01…06, STF-01…06, A-5, A-12, A-19 (groundwork only). **Depends on:** F1–F4.

**Not in C1:** patients (C2), and other data regions (only the groundwork is built).

## Data (a reshape is fine, because it's a fresh app: A-1)

- `orgs`: id, name, code, status, **`data_region`** (default `eu-west-2`, the only value for now), `settings jsonb`.
- `hospitals`: + `org_id`, status, `settings jsonb`.
- `users`: + `cognito_sub`.
- `memberships`: user_id, org_id, `hospital_id NULL` (NULL = the whole trust), `role` ∈ `clinician < super_clinician < admin`. This replaces today's roles.
- `platform_admins`, `staff_invites`, and `audit_log` (append-only).
- Questionnaires and other content get an `org_id` + `hospital_id NULL` owner.

## Design

- **`store.ForOrg(orgID)`** returns the trust's store. Today that's always the London pool. Every patient-data handler goes through it.
  - **No SQL joins of patient data across trusts.**
  - Control-plane tables hold no patient data.
  - These are the rules in [§13](../saas-requirements.md).
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

- Platform admin: Trusts list → New trust (name, code, contact, logo, first admin email, opt-ins; region fixed to London for now) → Hospitals.
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
- **Unit:** effective settings, the role ladder, and ID-token verification (with a fake key set).
- **End-to-end:**
  - **J6:** the platform admin creates a trust and hospital, and invites an admin; the admin invites a clinician; the clinician signs in.
  - **J1–J5** move to `/o/…` URLs.
  - In QA, the login helper signs in through Cognito's managed login, using test users whose TOTP secrets are known to the test (`pyotp`). Locally it uses the dev login.

## Status

**Partly done 6 Oct 2026 via [auth first](auth-first.md) Step 3** (not committed yet): `orgs`, `hospitals.org_id`, memberships on the ladder (trust-wide or one hospital), `internal/role`, routes under `/api/o/{oid}/h/{hid}/…` and pages under `/o/:oid/h/:hid/…`, the hospital picker grouped by trust, and a `forOrg` stub for `store.ForOrg`. Sessions are database rows, not just a cookie (auth first Step 2).

**Cognito sign-in and invites done 6 Oct via auth first Step 4** (MFA **off** for now, *pending* sign-off): [cognito.md](../cognito.md), the Staff page, `GET /api/o/{oid}/staff` and `POST …/staff/invites` (admins of the whole trust only).

Not done yet: changing or removing roles, `platform_admins` and the admin screens, `staff_invites`, `audit_log` (auth first Step 6), settings inheritance and opt-ins, and trust-owned content.
