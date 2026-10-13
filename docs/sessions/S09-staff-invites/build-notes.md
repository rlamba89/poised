# S09 build notes: staff invites, roles and removing access

Builder: automatic run (`/run-sessions`), 13 Oct 2026. Answers: [answers.md](answers.md) (nothing new; invites expire after 7 days, scope is the whole trust or one hospital).

## What was built

### Data (migration `00007_staff_invites.sql`)
- `staff_invites`: trust, `hospital_id` (NULL = whole trust), user, role, `invited_by`, `created_at`, `expires_at`, `accepted_at`. A composite foreign key keeps the hospital in the invite's trust.
- **Emails are unique whatever their case:** `users_email_key` is now a unique index on `lower(email)` (was a case-sensitive constraint). New accounts are stored in lower case. Down migration included.

### Go
- **`internal/clock`:** `Clock` (`Now()`), `Real`, and `Fake` (tests move it with `Add`).
- **`internal/cognito`:** `Client` with `CreateUser(email)` and `ResendInvite(email)` (AdminCreateUser and its RESEND in S12). `Local` (used by `make dev`/the servers) only logs; `Recorder` is the test fake.
- **`httpapi.Deps`** gains `Cognito` (required: `newServer` panics without it, so a server can't silently lose invitations) and `Clock` (nil = the system clock). `internal/serve` passes `cognito.Local{}`. `apitest` gives every test a `Fake` clock and a `Recorder` (`c.Clock`, `c.Cognito`), and responses now expose headers (`r.Header`).
- **Routes** (`internal/httpapi/staff.go`, behind `requireOrg`; only whole-trust admins, everyone else in the trust 403, non-members 404):
  - `GET /api/o/{oid}/staff`: one row per person: `userId, name, email, access: [{hospitalId, hospitalName, role}], status (active|invited|expired), inviteExpiresAt` (since round 1).
  - `POST /api/o/{oid}/staff/invites` `{email, name?, hospitalId|null, role}` → 201 with the person + `existingAccount`.
  - `PATCH /api/o/{oid}/staff/{uid}/access/{scope}` `{hospitalId|null, role}` → 200 with the person: changes **one** membership (`scope` = `trust` or a hospital id). Both fields are required; a missing `hospitalId` is never read as "whole trust". (Round 1; it was `PATCH …/staff/{uid}`.)
  - `DELETE /api/o/{oid}/staff/{uid}/access/{scope}` → 204: removes one membership (round 1).
  - `DELETE /api/o/{oid}/staff/{uid}` → 204: removes all of the person's memberships in the trust.
  - Each change runs in a transaction holding a per-trust advisory lock (`LockOrgStaff`).
- **The dev login checks invites** (`checkInvites`, which S12's Cognito sign-in must call too): an account waiting for its first sign-in may sign in while one invite is open, which accepts them all; once every one has expired it gets **403 "Your invitation has expired. Ask your trust's admin to invite you again."** and no cookie.

### How invites work
- **A new email:** a new account (name = the optional name, else the email), `cognito.CreateUser`, the membership at once, and an open invite expiring in 7 days.
- **An existing account that has signed in, or was seeded (STF-02):** only the membership; the invite row is recorded as accepted at once; no Cognito call.
- **An existing account still waiting for its first sign-in** (e.g. invited by another trust, or expired here): the membership is added or replaced, a new open invite, and `cognito.ResendInvite`. This is what **Resend invite** does.
- **Someone already in the trust:** a scope they don't hold is **added** (round 1); a scope they hold is 409 "… already has that access. Change it in the list instead." (or, if they haven't signed in yet, its role is updated and the invite resent).
- **Removing** deletes one membership, or all of the person's memberships in that trust. The account and the invite rows stay (past actions keep their author).
- **The last whole-trust admin** can't be removed, demoted, or narrowed to one hospital (409).

### Web app
- **Staff** link in the header for whole-trust admins only (`canManageStaff(trust)` in `src/lib/auth.tsx`). Page `/o/:orgId/h/:hospitalId/staff` (`src/pages/hospital/Staff.tsx` → `src/features/staff/StaffPage.tsx`). Switching hospital on Staff stays on Staff within the trust.
- The page: the list (Name, Email, Access, Role, Status badge; "(you)"), **Invite** dialog (email, optional name, access, role), **Change** dialog, **Remove** confirmation, **Resend invite** on invited/expired rows. Messages come back as a green status line. Changing or removing your own access sends you to `/`.
- Non-admins opening the URL see "Only the trust's admins can manage staff" (no API call).

**TypeScript for Rahul to review:**
- `src/lib/auth.tsx`: `canManageStaff(trust)` is `atLeast(trust.role, "admin")`: the trust-wide role, so an admin of one hospital (trust role null) can't.
- `src/features/staff/StaffPage.tsx`: one file, four parts. `StaffPage` loads `GET /api/o/<trust>/staff` and shows the table; `InviteModal`, `ChangeModal` and `RemoveModal` each send one request and report back; `AccessFields` is the shared scope + role pickers, where the value `"trust"` becomes `hospitalId: null` (`accessBody`). The modals keep the last row's name while they fade out (the same trick as S08's delete dialog).
- `src/pages/hospital/HospitalLayout.tsx`: the Staff link, and keeping Staff when switching hospital.

## Decided by the builder (automatic mode)
1. **Only whole-trust admins manage staff.** An admin of one hospital gets 403. Follows Rahul's S08 rule that only whole-trust staff change trust-level things, and the routes are trust routes (`/api/o/{oid}/staff`).
2. ~~**One membership per person per trust.**~~ *Superseded 13 Oct by Rahul's answer ([answers.md](answers.md)): several memberships per trust. See Round 1.*
3. **The membership is added when the invite is made; the 7-day expiry is the deadline for a new account's first sign-in.** This matches Cognito: AdminCreateUser's temporary password lasts 7 days (TemporaryPasswordValidityDays), and an expired one needs a RESEND. An expired invitee keeps their row (Status "Invite expired") but can't sign in until invited again.
4. **The invite takes an optional name** (the brief lists email, scope and role). Without one the email is the name; the dev login and the staff list would otherwise have only the email to show.
5. **The last whole-trust admin can't be removed or demoted** (409). Otherwise a trust could lock itself out, and only the superadmin (S10) could fix it.
6. **Emails are case-insensitive** (unique on `lower(email)`), so `Ann@X` and `ann@x` can't become two accounts (STF-02).
7. **The Cognito call is the last step inside the transaction**, so a Cognito failure rolls everything back (500). If the commit itself then fails, the Cognito account is left alone and inviting again resends to it (S12 must treat "already exists" that way).
8. **Token checks still use the system clock;** only invites use `Deps.Clock`. Moving the fake clock 8 days therefore doesn't sign test users out.
9. **The Staff page lives under the hospital layout** (`/o/:orgId/h/:hospitalId/staff`) rather than a new trust layout, so the header, hospital box and sign-out stay as they are. Its API is the trust route.
10. **Testing two people at once:** a second host name keeps its own cookie. The first plan used `http://[::1]:3000`; Claude in Chrome won't open it, so since round 1 it's `http://tester.localhost:3000` (the tester's find).

## Differences from the plan
- None in scope. The plan's "removed membership → 403" stays a 404, as decided in S08 (NFR-04).
- The e2e journeys leave their invitees ("Zoe E2E …", no access after clean-up) in the dev sign-in list, because accounts are never deleted. Logged in [pending.md](../../pending.md) §3.

## Tests
- **Unit:** `internal/clock` (`TestFake`); `httpapi/staff_test.go`: `TestNormalizeEmail`, `TestStaffAccessParse`; web `src/lib/auth.test.ts` ("who manages staff"), `src/routes.test.ts` (the staff URL, inside the hospital layout).
- **Integration (`staff_integration_test.go`, written first and failing):**
  - `TestListStaff`: happy path, no sign-in, wrong role (super clinician, clinician, admin of one hospital), another trust (both ways), a bad trust id.
  - `TestInviteStaff`: a new email (account in lower case, membership, open 7-day invite by Ada, one `CreateUser`, in the dev login list, signs in and sees only A1, invite accepted, row Active); whole trust + name defaulting to the email; **STF-02** (existing account in any case: same account, user count unchanged, accepted invite, no Cognito call, both trusts work); already on staff → 409; re-inviting an expired, never-signed-in account replaces its access and resends; no sign-in; wrong role (checked before the body is read); another trust (URL 404, its hospital in the body 400); 12 invalid inputs; Cognito failing → 500 with nothing written.
  - `TestStaffInviteExpires`: after 7 days + 1 minute the dev login is 403 with no cookie and the list says "expired"; within 6 days it's accepted for good; seeded users are unaffected.
  - `TestUpdateStaff`: a role change takes effect on the next request (403 → 201 creating a questionnaire); a scope change moves them (A1 404, A2 200); two memberships collapse to one; the last-admin guard (and that it passes with a second admin); no sign-in; wrong role, including self-promotion; another trust (URL, person, hospital); invalid input.
  - `TestRemoveStaff`: **STF-04** (next request 404, `/api/me` has no trusts, account kept, gone from the list); only this trust's access goes; the last-admin guard and removing yourself; no sign-in; wrong role; another trust; invalid input.
  - `routers_integration_test.go`: the patient router doesn't serve the staff routes; the staff router does.
- **End-to-end (`e2e/journeys/test_staff.py`, 5 journeys):** invite → the invitee (own browser context) signs in and sees only A1 → removed → next click refused; a role change gives New Questionnaire; an existing account (invited by Max to Trust B, then by Ada to Trust A in capitals) → "already had an account", one entry in the sign-in list; clinicians and whole-trust super clinicians don't see Staff; an invalid email is refused. `support.Api` gained `o()` and `invite()`.

Results (13 Oct, 00:3x):
- `make lint`: go vet and tsc clean.
- `make test`: Go ok; Vitest web 61 passed, clinical 116 passed.
- `make test-integration`: all packages ok.
- `make e2e` against `make dev`: **15 passed** (J1–J5, trusts, staff).
- Migration checked on the local database: up, down, up; `make seed` after it.

## Suggested commits (Rahul commits after verification)
1. **Staff invites, change role and remove access (C1 / S09: data and API)**: `apps/api/db/migrations/00007_staff_invites.sql`, `apps/api/db/queries/staff.sql`, `apps/api/internal/db/*` (generated), `apps/api/internal/clock/`, `apps/api/internal/cognito/`, `apps/api/internal/httpapi/{router.go,staff.go,auth_handlers.go,staff_test.go,staff_integration_test.go,routers_integration_test.go}`, `apps/api/internal/apitest/apitest.go`, `apps/api/internal/serve/serve.go`.
2. **The Staff screen (S09: web and journeys)**: `apps/web/src/**` (auth, routes, layout, `features/staff/`, `pages/hospital/Staff.tsx`), `e2e/support.py`, `e2e/journeys/test_staff.py`.
3. **Docs: S09 decisions, build notes, test plan and status**: `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/c1-trusts-staff.md`, `docs/sessions/README.md`, `docs/sessions/S09-staff-invites/*`, plus S09's share of the 12 Oct planning-doc changes (its `brief.md`, `answers.md`).

## Round 1 (test report 1: 22 passed, 1 failed) and Rahul's answers (13 Oct)
- **F-1 (the Invite dialog kept the last email, access and role):** fixed test-first. e2e `test_the_invite_dialog_starts_afresh_each_time` (red first) opens it fresh, after Cancel and after an invite. The Staff page now has one `Modal` whose form is remounted with a `key` bumped on every open, so every dialog (Invite, Add access, Change, Remove) starts from its defaults or the row's own values. The modal keeps the last dialog's content while it fades out.
- **Answer 1: several memberships per trust** (supersedes builder decision 2). Built test-first:
  - Integration (red first): `TestListStaff` (one row per person with both of Cara's hospitals), `TestInviteStaff` (an existing member invited to another hospital → added; to the whole trust on top → added; the same access → 409; a waiting account re-invited elsewhere keeps its access and gets a resend; resending to the same access adds nothing and takes a new role), new `TestUpdateStaffAccess` (role change on the next request; moving one membership leaves the other; moving onto a held scope → 409; last-admin guard; no sign-in; wrong role; another trust's URL, person and hospital; invalid bodies, a missing person or membership, a bad scope) and new `TestRemoveStaffAccess` (one goes, the others stay; the last one ends the trust access; the whole-trust admin's membership is protected but their other ones aren't; no sign-in; wrong role; another trust; invalid scope). `TestRemoveStaff` (whole person) now removes two memberships at once.
  - Go: new queries `MembershipRole`, `UpdateMembership`, `DeleteMembership` (`hospital_id IS NOT DISTINCT FROM` the scope); handlers `updateStaffAccess`, `removeStaffAccess`, `staffScope` ("trust" or a uuid, else 404); `staffRowsOf` folds membership rows into people; the invite adds or updates one membership instead of replacing them all. The person-level PATCH was dropped.
  - Web: one row per person; each access on its own line with **Change** and **Remove**; per person **Add access** (the invite form with the email fixed, offering only scopes they don't have), **Resend invite**, **Remove from trust**. The Change dialog offers only the access's own scope and ones they don't hold.
  - e2e: `test_someone_can_work_at_several_hospitals_of_a_trust` (Add access, two lines, remove one, they land in the other); the other journeys use the new button names.
  - Test plan: T-02, T-03, T-05–T-11 updated; new T-18 (several accesses) and T-19 (F-1); the second address is `tester.localhost`.
- **Answer 2:** the other choices are confirmed; nothing changed.
- **TypeScript for Rahul (round 1):** `StaffPage.tsx` now keeps one `dialog` value saying which dialog is open (`invite`, `add`, `change`, `removeAccess`, `removePerson`) and renders the matching form (`InviteForm`, `ChangeForm`, `RemoveForm`) inside one `Modal`; `opening` is the counter used as the form's `key` (F-1). `scopeOf(access)` is the URL's scope (`"trust"` or the hospital id).
- Results: `make lint` clean; `make test` (web 61, clinical 116) and `make test-integration` pass; `make e2e` **17 passed**.

## Round 2 (test report 2: 24 passed, 0 failed)
- Everything passed, including the reworked T-02–T-11, the new T-18 and T-19, and the regression cases. No code changed after round 1.
- The report's "found while exploring" items are logged in [pending.md](../../pending.md) §3: the Remove dialog saying someone loses a hospital they keep through whole-trust access (and "access at the whole trust" wording); a change to your own access always going to the picker with no message, even Add access; and the "already had an account" wording for someone already on this trust's staff. Its note on T-18 step 11 (clicking the page you're already on makes no request) is a test-plan wording point only.
- Status: **Tested**; then **Verified** 13 Oct ([verify.md](verify.md)).

## Hand checks for Rahul
- None yet.
