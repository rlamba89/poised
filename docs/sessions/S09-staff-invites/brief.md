# S09: Staff invites, roles and removing access

**Plan:** C1 ([c1-trusts-staff.md](../../plans/c1-trusts-staff.md)), part 2 of 5.
**Requirements:** STF-01, STF-02, STF-04.
**Depends on:** S08 verified.

**Ask Rahul first:** nothing new. The plan's choices stand: invites expire after 7 days, and the scope is the whole trust or one hospital.

## Build

- **`staff_invites`,** plus a **`Deps.Cognito` interface** with a recording fake. Real Cognito comes in S12.
  - Locally, an invited user appears in the dev login list.
- **Routes** (admin only, with the five cases each):
  - `GET /api/o/{oid}/staff`;
  - `POST …/staff/invites`, taking email, scope and role;
  - `PATCH …/staff/{uid}`, to change the role or scope;
  - `DELETE …/staff/{uid}`.
- **Inviting an existing email only adds a membership** (STF-02). There's never a second account.
- **Removing a membership ends access on the next request** (STF-04).
- **The invite expires after 7 days.** The clock is behind an interface, so tests can move it.
- **The Staff screen:** a list, invite, change role, and remove. Only admins can see it.

## Not in this session

- real email or Cognito (S12);
- the superadmin's first-admin invite, which reuses this in S10.

## Manual test focus

1. An admin invites a clinician to Hospital A1. The invitee signs in with the dev login and sees only A1.
2. Inviting a trust B user into trust A adds a membership, and no duplicate account appears.
3. Changing someone's role changes what they can do.
4. Removing someone means their next action is refused.
5. A clinician can't see Staff, and a super clinician can't invite.
6. An invalid email is refused.
7. **An expired invite:** the test plan may age it with one SQL step, and must say so.
