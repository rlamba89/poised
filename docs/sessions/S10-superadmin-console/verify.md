# S10 verification

## Verification 2 (13 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1's three fixes are correct and tested first. Findings 4–9 are logged in pending.md. Test report 3 passed, and every automated suite passes. The session as a whole matches the brief plus Rahul's answers 1–4.

### Verification 1's fixes, checked

1. **Platform admin locked out by suspension: fixed.**
   - `devLogin` now calls `signInAllowed` (`apps/api/internal/httpapi/auth_handlers.go:25`), which lets a platform admin in before `SignInState` is checked. The note for S12 now names `signInAllowed`.
   - Test: `TestAdminSuspendTrust`, "a platform admin who is also its staff still signs in, and can re-activate it" (`admin_integration_test.go:484`). It checks the 204, the trust missing from `/api/me`, a 404 on its hospital, the re-activation and the 200 afterwards.
   - **Re-checked against `make dev`:**
     - Sam suspended Verify S10 Trust 1. Vera Verifier S10 (a platform admin and admin of that trust) still signed in (204), with `"trusts":[]`, and got 404 on its hospital's episodes.
     - She re-activated the trust herself (200), and the hospital then answered 200.
2. **"No hospitals yet" for a trust whose hospitals are all suspended: fixed.**
   - `ListMyHospitals` (`apps/api/db/queries/users.sql`) now gives the no-hospital row only when the trust has no hospitals at all.
   - `SignInState` (`admin.sql:73`) counts a whole-trust membership as usable only when the trust has an active hospital or none yet. So whole-trust and hospital-only staff now agree.
   - Test: `TestAdminSuspendHospital` (`admin_integration_test.go:555`).
   - **Re-checked:**
     - I suspended VS10 Hospital 1, Trust 1's only hospital. Vince TwoTrusts still signed in through Trust 2, and `/api/me` listed only Trust 2. After re-activation it listed both.
     - Zoe Admin Nine, of the brand-new QA Trust Nine, still gets the trust with `hospitals: []`.
3. **`make platform-admin` with an apostrophe: fixed.** `Makefile:53` quotes `"$(EMAIL)"` and `"$(NAME)"`. An empty NAME reaches the command as `""`, which `add` turns into the email, as before.

### Findings, worst first

All are minor. Each is logged in [pending.md](../../pending.md) §3, "From S10's verification 2". None blocks Verified.

1. **A trust whose hospitals are all suspended still answers its trust-level routes.**
   - **Where:** `ListOrgMemberships` (`users.sql`) checks only the trust's status. `ListMyHospitals` and `SignInState` now treat such a trust as unusable.
   - **The effect:** a whole-trust member who is already signed in, or who signs in through another trust, can still reach `/api/o/{oid}/staff` and `/settings`.
   - **Why it's minor:** these routes aren't reachable from the UI (the pages sit under a hospital), the trust itself isn't suspended, and no patient data is involved.
   - This adds to verification 1's finding 8: one "usable membership" helper would remove the drift.
2. **An empty hospital text setting means two things.**
   - Trust Settings stores `""` as a hospital's own value (`trust_settings.go` `check`). The console's `textSettings` (`admin.go:325`) treats `""` as "follow the trust".
   - So the console can show a hospital as using the trust's phone when it has none.
   - Nothing shows contact details to patients until S15. Decide which meaning `""` has by then.
3. **Editing a trust made before S10 runs the full new-trust checks** (`admin.go` `adminUpdateOrg`).
   - A migrated trust (code `migrated-<uuid>`, empty contact) can't be renamed until its code and contact are also fixed.
   - This is local data only. Status changes aren't affected.
4. **`"optIns": {}` alongside other changes is refused** with "Nothing to change." (`statusPatch.validate`). The UI never sends it.
5. **The journeys depend on the seeded Sam having no memberships.**
   - The first `make e2e` run failed two J6 journeys. The product is right: test report 3 had left Sam as admin of QA Trust Seven, so his sign-in went to that hospital, as designed.
   - `make seed` resets the demo users' memberships. After it, J6 passed, and so did the whole suite.
   - Run `make seed` before `make e2e` after a manual round.

**From `/code-review high`, already logged after verification 1:** the seed's `ON CONFLICT (id)`, New trust storing every opt-in, three email checks, and the extra read of the trust row on a PATCH (efficiency only).

### Checklist (whole session)

- **Scope:** everything in the brief and answers 1–4 is built, as in verification 1.
- **Tests:** every new route has its five cases. `internal/settings` has unit tests, J6 is in e2e, and the patient router serves no admin or settings routes.
- **Tenant isolation:** admin routes are scoped by `org_id`. The superadmin gets 404 on every trust and hospital route.
- **A-20:** the grep finds only the comments in `apps/web/src/lib/format.ts`, which predate S10.
- **Security:**
  - No secrets were added.
  - Ada gets 403 on `/api/admin/orgs`, `/api/admin/opt-ins` and `/api/admin/orgs/{oid}`. With no sign-in, the answer is 401.
  - In the browser, Ada at `/admin` sees "Only Poised's platform admins can open this page".
- **Docs:**
  - decisions.md has both fix decisions (13 Oct).
  - The pending items are logged.
  - The organisation-name grep is clean.
  - Status is updated to Verified below.

### Test results (13 Oct, on the working tree)

- `make lint`: go vet (integration tag) and tsc clean.
- `make test`: Go ok; Vitest web 77 passed, clinical 116 passed.
- `make test-integration`: every package ok, incl. `cmd/platform-admin`, `internal/httpapi` and `internal/settings`.
- `make e2e` against the conductor's `make dev`:
  - first run: 2 failed and 21 passed (finding 5, test data);
  - after `make seed`: **23 passed** in 62.6 s.

### Spot checks

1. A platform admin who is staff of a suspended trust signs in and re-activates it (API, as Vera). **Pass.**
2. Whole-trust staff of a trust whose only hospital is suspended: the trust drops out, and the new trust with no hospitals is still listed. **Pass.**
3. A non-superadmin on the admin API and on `/admin` in Chrome (Ada, at tester.localhost). **Pass.**
4. Ada's trust Settings at Hospital A1 renders, with the opt-ins shown as set by Poised. **Pass.**

**Test data:** I ran `make seed`, which put the demo trusts, hospitals, settings and the demo users' memberships back. So Sam is no longer admin of QA Trust Seven. Everything I suspended was re-activated.

### Hand checks for Rahul

- None. `make platform-admin` was checked in verification 1.

### Commits

These are as suggested in the build notes:
1. Settings, the superadmin console API and suspension (C1 / S10: data and API).
2. The admin console and Settings screens (S10: web and journeys).
3. Docs: S10. This includes S10's `brief.md` and `answers.md`, plus whole files that also hold 12 Oct planning changes.

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Needs fixes.** One real bug in suspension (finding 1) that takes a few lines to fix, plus two small ones. Everything else matches the brief and Rahul's answers 1–4: the console, New trust with S09's invite, hospitals, editing details, suspend and re-activate, platform-admin-only opt-ins, every ORG-04 setting with `settings.Effective`, the trust Settings screen, `cmd/platform-admin`, and J6. All the automated tests pass.

Judged against the brief plus [answers.md](answers.md), which widened it at Rahul's choice: a logo URL, all of ORG-04, editing details at any time, and opt-ins for the platform admin only.

### Findings, worst first

**Fix before Verified:**

1. **A platform admin who is also staff of a suspended trust is locked out of the console.**
   - **Where:** `apps/api/internal/httpapi/auth_handlers.go:48-55` (`devLogin`'s `SignInState` check).
   - **The bug:** sign-in is refused when all of someone's memberships are suspended, and platform admins aren't exempt.
   - **Reproduced against `make dev`:**
     - I made Vera Verifier S10 a platform admin with `make platform-admin`.
     - Sam invited her as an admin of "Verify S10 Trust 1" (`POST /api/admin/orgs/{oid}/admins`), then suspended that trust.
     - `POST /api/dev/login` for Vera now answers 403 "Your trust or hospital is suspended…".
     - So she can't reach `/admin` to re-activate the trust she suspended. I re-activated it afterwards.
   - **Why it matters:** the build expects platform admins to be staff too (the "Platform admin" header link, Home's console card). Rahul is likely to be both, for example as a demo trust's first admin. If he's the only platform admin, the only way back is SQL. The comment also says S12's Cognito sign-in "must refuse the same way", which would carry the bug forward.
   - **Fix (test-first):**
     - Skip the refusal when `IsPlatformAdmin` is true. Their suspended memberships are still left out of `/api/me` and of every trust and hospital route, as now.
     - Add an integration case to `TestAdminSuspendTrust`: a platform admin who is a member of the suspended trust still signs in and can re-activate it.
     - Update the note for S12 at `auth_handlers.go:16`.
2. **A trust whose hospitals are all suspended is described as having "no hospitals in Poised yet".**
   - **Where:** `apps/api/db/queries/users.sql` `ListMyHospitals` (the LEFT JOIN, with `h.status = 'active'` in its ON clause), and `apps/web/src/pages/Home.tsx:67-71`.
   - **The bug:** for a whole-trust member, suspended hospitals become the same `hospitals: []` row as a brand-new trust. Home then says the platform admin hasn't added the hospitals yet, which is untrue.
   - **The rules also differ:** a hospital-only member of those hospitals is refused at sign-in, while the whole-trust member signs in to an empty trust.
   - **Fix:** tell the two cases apart. For example, leave the trust out of `/api/me` when it has hospitals but none of them is active, so the hospital-only and whole-trust cases agree. Or change the wording to cover both cases. Either is small. Alternatively, log it in pending.md with a reason.
3. **`make platform-admin` breaks on a name with an apostrophe.**
   - **Where:** `Makefile:53`. `$(NAME)` isn't quoted, so `NAME="Sean O'Brien"` gives the shell an unmatched quote.
   - Spaces already work, because the command joins its arguments.
   - **Fix:** quote it, as `"$(NAME)"`. This is the command Rahul runs by hand.

**Log in pending.md (no fix needed in S10):**

4. **`make seed` can fail after hospitals are renamed in the console.**
   - **Where:** `apps/api/cmd/seed/main.go` (the hospital and trust upserts use `ON CONFLICT (id)` only).
   - **The bug:** suppose A2 is renamed "Hospital A1" after A1 was renamed. Re-seeding A1's name then hits `hospitals_org_name_key`, and the reset aborts. The same applies to trust codes and `orgs_code_upper_key`.
   - It only affects test data, but the seed is the "start each round clean" tool.
5. **The default HQ can only be a trust-level questionnaire.**
   - **Where:** `apps/api/db/queries/admin.sql` `PublishedQuestionnairesInOrg` (`hospital_id IS NULL`).
   - **The problem:** a hospital's own published HQ can't be its default, though the setting is hospital-overridable.
   - Nothing reads it until S13, so decide it there.
6. **New trusts store every opt-in explicitly.**
   - **Where:** `apps/web/src/features/admin/NewTrustForm.tsx:25`, which sends all seven keys at their defaults.
   - **The problem:** console-made trusts never follow a later change to a Poised default, and the opt-in table has no "use Poised's default" control. This relates to the explicit-`false` item already logged from report 2.
   - **Fix:** send only the keys that were changed, when convenient.
7. **Email checks now exist three times, with slightly different rules.**
   - **Where:** `normalizeEmail` (staff.go), `settings.Validate` Email (`settings.go:248`), and `cmd/platform-admin/main.go:59`.
   - For example, settings accepts `help@trust.`, which `normalizeEmail` refuses.
   - **Fix:** one shared checker.
8. **Suspension is enforced by a `status = 'active'` filter repeated in six queries.**
   - **The risk:** S13's patient records and S16's reminder job must remember it, or patients of a suspended trust get reminders whose links answer 403.
   - Note it in those briefs, or move it to a single view or helper.
9. **Small duplication:** the seed and `apitest`'s `PlatformAdmin` fixture still insert into `platform_admins` with raw SQL, though the `AddPlatformAdmin` query now exists.

**Checked and not a problem:** `/code-review high` suggested that switching hospital across trusts on Settings keeps the first trust's drafts and could save them into the second trust.
- `HospitalLayout` keys `AuthProvider` by `hospitalId`, so the page remounts.
- I reproduced the switch in the browser with a person who is admin of two trusts. Trust 2's form showed none of Trust 1's values, and Save said "Nothing has changed." with nothing written.

### The review checklist

- **Scope:** every item in the brief is built, plus answers 1–4.
  - The two extra routes (`POST …/admins`, `GET /api/admin/opt-ins`) are small and justified in the build notes.
  - Logos are a URL field, as Rahul chose.
- **Tests:**
  - Every new route has its five cases (`admin_integration_test.go`, `settings_integration_test.go`).
  - `internal/settings` has unit tests, written first.
  - The patient router doesn't serve `/api/admin/…` or settings (`routers_integration_test.go`).
  - J6 is in `e2e/journeys/test_j6_onboarding.py` (5 journeys).
- **Tenant isolation:**
  - Admin hospital routes are scoped by `org_id` (another trust's hospital is 404; checked by the tester and by the tests).
  - Settings PATCH is scoped to `orgID(r)`.
  - The superadmin gets 404 on every trust and hospital route. I checked patients, episodes, settings and staff for Trust A as Sam: all 404.
- **A-20:**
  - The grep finds only comments in `apps/web/src/lib/format.ts`, which predate S10.
  - Time zones come from the browser's IANA list with the deployment's as the default, and phones are checked for shape only.
- **Security:**
  - No secrets were added.
  - The sign-in refusal and the patient link's 403 carry no patient data.
  - `requirePlatformAdmin` is re-read on every request. Ada gets 403 on all three admin URLs I tried, and an anonymous request gets 401.
- **Docs:**
  - The C1 plan Status, `docs/plans/README.md`, `docs/sessions/README.md` ("Tested") and `docs/decisions.md` are updated.
  - The organisation-name grep is clean.

### Test results (13 Oct, on the working tree)

- `make lint`: go vet (integration tag) and tsc clean.
- `make test`: Go ok (incl. `internal/settings`); Vitest web 77 passed, clinical 116 passed.
- `make test-integration`: every package ok, incl. `cmd/platform-admin` and `internal/httpapi`.
- `make e2e` against the conductor's `make dev`: **23 passed** in 69.8 s.

### Spot checks (Chrome and the API against `make dev`)

1. **Settings across two trusts** (from the code review): as Vince TwoTrusts, an admin of Verify S10 Trusts 1 and 2. **Not a bug** (see above).
2. **Platform admin lockout on suspension:** **reproduced** (finding 1).
3. **Non-superadmin on the admin API:**
   - Ada (a trust admin) gets 403 "Only Poised's platform admins can do this." on `GET /api/admin/orgs`, `/admin/opt-ins` and `/admin/orgs/{TA}`.
   - With no sign-in, the answer is 401.
4. **The superadmin sees no patient data:**
   - Sam gets 404 on Trust A's patients, episodes, settings and staff.
   - `GET /api/admin/orgs/{TA}` holds no patient fields.
5. **Hand check `make platform-admin`:**
   - The first run prints "Created the account and made it a platform admin.", and the second "The account exists; it is a platform admin.".
   - With no EMAIL, it prints the usage and exits 2.
   - **This hand check is done.**

**Test data left in the dev database:** "Verify S10 Trust 1/2" (VS10-1, VS10-2), VS10 Hospital 1/2, Vince TwoTrusts, and Vera Verifier S10 (a platform admin, admin of Trust 1). Trust 1 is active again. Trusts can't be deleted; `make seed` doesn't touch them.

### Hand checks for Rahul

- None left. I ran `make platform-admin` myself (above). Run it once for your own email when you want your own superadmin account.

### Suggested commits (after the fixes)

These are as in the build notes:
1. **Settings, the superadmin console API and suspension (C1 / S10: data and API).**
2. **The admin console and Settings screens (S10: web and journeys).**
3. **Docs: S10 decisions, build notes, test plan, test reports, verification and status.** This includes S10's `brief.md` and `answers.md`.

For the builder: "Verification found problems: read `docs/sessions/S10-superadmin-console/verify.md`, fix them test-first, then ask me for a retest."
