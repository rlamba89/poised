# S10: Superadmin console: trusts, hospitals and settings

**Plan:** C1 ([c1-trusts-staff.md](../../plans/c1-trusts-staff.md)), part 3 of 5.
**Requirements:** ORG-01…05, A-21, OPS-06 (no patient data for the superadmin).
**Depends on:** S09 verified.

**Ask Rahul first:**
1. **Logos** need file uploads (C8). **Recommended:** leave them out until C8.
2. **Settings:** build the `settings.Effective` mechanism, the opt-ins (ORG-05) and contact details now. Every other setting is added by the plan that first uses it. **Recommended: yes**, so there are no settings that do nothing yet.

## Build

- **`cmd/platform-admin add <email>`** (A-21). It uses the Cognito fake locally and real Cognito from S12 on.
- **`/api/admin/…` routes,** for superadmins only (the five cases each):
  - create and list trusts;
  - add hospitals;
  - `PATCH` to suspend, re-activate, and set opt-ins.
- **Admin screens:**
  - **the trusts list;**
  - **New trust:** name, code, contact, and the first admin's email, which sends an invite using S09;
  - **hospitals:** name, address, patient-facing phone and email, and time zone;
  - **suspend and re-activate:** sign-in is refused while suspended, and the data is kept;
  - **opt-ins:** the trust sets a default, and each hospital follows it or decides for itself.
- **Trust settings screen** (for trust admins), with per-hospital overrides.
  - `settings.Effective(platform, org, hospital)` is written test-first.
- **E2E J6** (locally, dev login): the superadmin creates a trust and a hospital, and invites an admin. That admin invites a clinician, and the clinician signs in.

## Not in this session

- logos;
- real Cognito (S12);
- audit (S11).

## Manual test focus

1. **Onboard a trust end to end:** trust → hospitals → first admin → that admin invites a clinician → the clinician signs in.
2. **A non-superadmin can't reach the admin area,** through either the UI or the API URLs.
3. **Suspend a trust:** its users can't sign in. Re-activate it: they can again.
4. **Opt-in inheritance:** trust off with a hospital on, and the reverse.
5. **A duplicate trust code** is refused.
6. **The superadmin sees no patient data anywhere.**
