# S08: Trusts, hospitals and permissions

**Plan:** C1 ([c1-trusts-staff.md](../../plans/c1-trusts-staff.md)), part 1 of 5.
**Requirements:** ORG-01, STF-02, STF-03, A-5, A-20 and A-21 ([saas-requirements](../../saas-requirements.md)).
**Depends on:** S07 verified.

**Ask Rahul first** (recommendation first in each):
1. **A time zone per hospital** (A-20, *Proposed*)? **Recommended: yes.** Canada, the US and Australia span several zones, and reminders depend on local time.
2. **Roles:** today's `viewer / author / reviewer / publisher / hospital_admin / clinician` become `clinician < super_clinician < admin` (A-5). There's no "viewer", because everyone in a hospital sees its data. **Recommended:** author/reviewer/publisher → super clinician; hospital admin → admin; viewer and clinician → clinician. Reseed, since it's a fresh app (A-1).
3. **Patient record visibility:** a hospital-scoped user sees the trust's **patient record**, so they can create an episode at their hospital, but only **their hospital's episodes**. **Recommended: yes.** This is needed for S13.

## Build

- **Data** (from C1 "Data"):
  - `orgs` (trusts);
  - `hospitals`, with `org_id`, status, settings and time zone if agreed;
  - `users.cognito_sub`;
  - `memberships (user_id, org_id, hospital_id NULL = whole trust, role)`;
  - `platform_admins`;
  - questionnaires and other content owned by `org_id` + `hospital_id NULL`.
- **Routes move to `/api/o/{oid}/…`.**
  - One membership middleware re-reads the membership on every request.
  - A hospital-scoped user sees only their hospital's rows.
- **`GET /api/me`** returns the user's memberships.
- **The frontend moves to `/o/:oid/h/:hid/…`:**
  - a **hospital picker** after sign-in, when the user has more than one hospital;
  - a header showing the current trust and hospital, with a switch;
  - **"All hospitals"** on the episodes list for trust-scoped users.
- **Seed data:**
  - Trust A, with Hospitals A1 and A2;
  - Trust B, with Hospital B1;
  - a trust-wide admin for A;
  - a hospital-scoped clinician at A1;
  - a trust-wide super clinician;
  - **one user who is a member of both trusts;**
  - a dev superadmin.
- **Tests:**
  - integration tests move to the trust model (the five cases, plus C1's extras: trust A admin gets 404 on trust B; hospital-scoped isolation; the role ladder);
  - **J1–J5 move to the new URLs.**

## Not in this session

- invites (S09);
- the superadmin screens (S10);
- audit (S11);
- Cognito (S12): the dev login stays.

## Manual test focus

1. **For each seeded user,** check what they see and what they can and can't do.
2. **The picker:** switching hospitals and trusts.
3. **"All hospitals"** for a trust-wide user.
4. **Leaks:**
   - a hospital-scoped clinician opens another hospital's episode by URL;
   - the trust A admin opens a trust B URL.
5. **The role ladder:** a clinician can't publish; a super clinician can.
6. **The two-trust user** moves between trusts.
7. **Regression:** J1–J5 at the new URLs.
