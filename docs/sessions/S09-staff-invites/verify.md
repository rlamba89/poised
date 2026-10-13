# S09 verification

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Verified.**

Everything in the brief is built and tested: `staff_invites`, `Deps.Cognito` (recording fake) and `Deps.Clock` (fake in tests), the four staff routes (plus the per-membership PATCH/DELETE that Rahul's "several memberships per trust" answer needed), STF-02 (an existing email only gets a membership), STF-04 (removal ends access on the next request), the 7-day expiry, and the Staff screen for whole-trust admins only. Test report 2 passed all 24 cases. No finding blocks the session; the ones below are logged in [pending.md](../../pending.md) §3.

### Findings (worst first; none blocking)

1. **The last-admin guard counts whole-trust admins who have never signed in,** even ones whose invite has expired. `apps/api/db/queries/staff.sql:74` (`CountOtherTrustAdmins`), used by `keepsAnAdmin` in `apps/api/internal/httpapi/staff.go:360`. The only working admin can demote or remove themselves once they've invited another admin who then never signs in. The trust can't manage its staff until S10's superadmin steps in. **For Rahul (later):** should "another admin" mean one who has signed in?
2. **Invite state is per account, across trusts.** `AwaitingFirstSignIn` and `checkInvites` (`staff.go:274`, `staff.go:553`) look at all of an account's invites. So when trust A re-invites an account that never signed in, its first sign-in also accepts trust B's expired invite. The B membership that B granted then works. This follows the agreed rule (access starts at the invite; the 7 days is only the first sign-in's deadline), and real Cognito accounts are per user pool anyway. The staff list's invite join (`staff.sql:18`) also isn't limited to the trust, so "Invited · expires …" can show another trust's deadline. Revisit with S12.
3. **An invite reveals whether an email has an account anywhere,** and the name another trust stored for it: `existingAccount` and the stored name in the 201 body, and the name in the 409 text (`staff.go:286`). This is inherent in STF-02's "never a second account", but it leaks a little across tenants.
4. **For S12:** the Cognito call runs inside the invite transaction (`staff.go:315`), holding the trust's staff lock and a pooled connection for the network round trip (builder decision 7, deliberate). Also, `serve.Main` always wires `cognito.Local{}` (`apps/api/internal/serve/serve.go:33`), so a deployed stack without the real client would accept invites that send nothing. That's fine in the local phase, but S12 must replace it.
5. **The admin check runs before the staff lock** (in `requireOrg`; `staffAdmin` at `staff.go:156`). A request that has already passed it can commit just after its caller is demoted. A very narrow race.
6. **Already logged by the tester** (pending.md §3): the Remove dialog wording when whole-trust access remains; a change to your own access always goes to the picker, even Add access; and the "already had an account" wording for existing members.
7. **Doc fix made by the verifier:** `docs/decisions.md` said re-inviting a waiting account "replaces its access here". Since round 1 it adds the access and keeps the rest, so the wording now says that.

### Review checks

- **Scope:** complete. One addition beyond the brief: per-membership routes, from Rahul's 13 Oct answer. An optional name on invites is builder decision 4.
- **Tests first:** each of the five staff routes has happy path, no sign-in, wrong role, another trust and invalid input (`staff_integration_test.go`: `TestListStaff`, `TestInviteStaff`, `TestUpdateStaffAccess`, `TestRemoveStaffAccess`, `TestRemoveStaff`), plus `TestStaffInviteExpires` (fake clock) and Cognito failure → nothing written. There are unit tests (`TestNormalizeEmail`, `TestStaffAccessParse`, `clock.TestFake`, web `canManageStaff`, the route table) and 7 e2e journeys in `e2e/journeys/test_staff.py`.
- **Tenant isolation:** every staff query is scoped by `org_id`. A hospital in the body is checked against the trust (`HospitalInOrg`), and the composite foreign keys keep invites and memberships within their trust. Another trust's URL is 404.
- **A-20:** the grep finds only comments in `apps/web/src/lib/format.ts` (from before S09).
- **Security:**
  - no secrets;
  - nothing personal is logged (`cognito.Local` logs only the email, locally);
  - the patient router doesn't serve `/api/o/{oid}/staff` (`routers_integration_test.go`);
  - no change to what patients receive.
- **Docs:**
  - the plan Status, `docs/plans/README.md`, `docs/sessions/README.md`, `decisions.md` and `pending.md` are updated;
  - the scrub grep prints nothing.

### Test results (13 Oct)

- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web **61 passed**, clinical **116 passed**.
- `make test-integration`: all 9 packages ok (httpapi 5.8 s).
- `make e2e` against the conductor's `make dev`: **17 passed** in 52 s.

### Spot checks (Claude in Chrome, against `make dev`)

1. **Wrong role:** Alex (whole-trust super clinician) gets 403 "Only the trust's admins can manage staff." on GET staff and on POST invite (whole-trust admin). No account was made.
2. **Another trust:** as Ada, Trust B's GET staff, PATCH Max's access and DELETE Max each give 404 "Trust not found.". Inviting to Hospital B1, or moving Max to it, gives 400. A PATCH without `hospitalId` gives 400 (never read as the whole trust).
3. **Last admin:** Ada demoting herself, or removing herself, gives 409 with the right message.
4. **Invite → sign in → remove (STF-01, STF-04),** through the UI:
   - Ada invited `Vera.Verify@Test.example` to A1 as a Clinician. The message gave the lower-cased email and a 20/10/2026 expiry, and the row showed Invited.
   - Vera signed in at `tester.localhost` and landed on A1 Episodes as a Clinician. `/api/me` lists only A1, Staff gives 403 and A2 gives 404.
   - Ada used Remove from trust ("Vera Verify no longer has access to Trust A."). Vera's next click (an episode) showed "Hospital not found.".
   - In the database, Vera's account is kept, with 0 memberships and the invite accepted.

### Hand checks for Rahul

- None.

### Commits (made by the verifier, automatic mode)

1. `Staff invites, change role and remove access (C1 / S09: data and API)`: the migration, queries, generated `internal/db`, `internal/clock`, `internal/cognito`, the httpapi staff handlers, router, dev-login invite check, `apitest`, `serve`, with their unit and integration tests.
2. `The Staff screen (C1 / S09: web and journeys)`: `apps/web` (auth, routes, layout, `features/staff`, `pages/hospital/Staff.tsx`, AGENTS.md) and `e2e/support.py`, `e2e/journeys/test_staff.py`.
3. `Docs: S09 ...`: `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, and `docs/sessions/S09-staff-invites/*`. S09's `brief.md` also carries a 12 Oct planning line ("Local phase"), and is committed whole.

Next session: `/build-session S10`.
