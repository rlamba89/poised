# Auth first: real sign-in, sessions and trusts

**Agreed with the developer on 6 Oct 2026. Rahul (product owner) has not signed off yet:** it runs before F1–F4 and turns MFA off for now. See [decisions.md](../decisions.md) §2, 6 Oct. It takes parts of [F1](f1-test-foundation.md), [C1](c1-trusts-staff.md) and [C2](c2-patients-signin.md).

## Context

Today's login is a dev stub (`apps/api/internal/httpapi/auth_handlers.go`). `GET /api/dev/users` lists every user. `POST /api/dev/login` signs in as any of them with no password and returns a 12 h HS256 token in the `sj_token` cookie (`internal/auth/token.go`). Logout only clears the cookie. Patients use `/p/{token}`, and that link alone returns their HQ.

Poised is becoming a multi-trust SaaS, so this replaces the stub with:
- **Staff:** Amazon Cognito (email + password), then our own session cookie (decision A-12, plan C1).
- **Patients:** link + Continue + date of birth (A-4, PAT-04/05).
- **Sessions in Postgres:** revocable, with an idle timeout.
- **Trusts above hospitals,** with the new role ladder (C1, A-5).

**Agreed with the user this session (6 Oct 2026):**
- Scope is auth and login only. AWS deploy, React/Vite and Lambdas are out (F2–F4 untouched).
- **MFA is off for now.** This deviates from A-12 and STF-05, where MFA is required. To be switched on before real patient data (C10).
- Cognito will be used; an AWS account will be provided. Only a Cognito user pool is needed, with no other AWS resources.
- Trusts come in this work too.
- Sessions are stored in the database. This is a proposal: C1 and C2 only say "our own cookie".

**Deviations from the agreed roadmap (record in `docs/decisions.md` via `/update-wiki`, manager sign-off pending):** C1 before F1–F4; MFA off; DB sessions; patient sign-in is only the link + date-of-birth part of C2. NHS number, patient accounts, SMS/email, jobs and the patient home stay in C2.

## Conventions to keep (checked in the code)

- **Go:**
  - `gofmt`.
  - A package doc comment, and a doc comment on every function, starting with its name.
  - Errors wrapped with context (`fmt.Errorf("verify token: %w", err)`).
  - User-facing errors are full sentences sent through `writeError`; server faults go through `serverError(w, "what", err)` (`middleware.go`).
  - Stdlib `net/http` only.
  - SQL lives in `apps/api/db/queries/*.sql`; regenerate with `make sqlc` and never edit `internal/db`.
  - Non-members get 404, never 403.
- **TypeScript:** `strict`, tests next to the source, a one-line header comment per file, and exact-pinned versions.
- **Commits:** one per step, with its tests. Tests are written first (TDD).

## Step 0: minimal integration harness (prerequisite)

The testing rules (`docs/plans/README.md`) require new route tests to be API → real Postgres. No such harness exists today. F1 is "Not started", there are no build tags, and there is no `apitest` package.

Build **only F1 Step 1**:
- `apps/api/internal/apitest` behind `//go:build integration`.
- A template database migrated with goose as a library.
- `apitest.New(t)` creates a database per test from the template.
- Helpers `c.LoginAs`, `c.Do` and `c.DB`.
- `make test-integration`, and `TEST_DATABASE_URL` added to `.env.example`.

F1's remaining steps (tests for every existing route, the Playwright suite) stay in F1. The existing `fakeQ` tests (`internal/httpapi/httpapi_test.go`) stay as they are, per rule 4.

## Step 1: hardening (no new behaviour)

- `DEV_LOGIN` env var. When it isn't `true`, both `GET /api/dev/users` and `POST /api/dev/login` return 404. They are registered in `router.go`.
- Cookie flags:
  - `Secure` when not on localhost (an env flag);
  - `SameSite=Lax` on both set and clear (`auth_handlers.go` logout sets no SameSite today);
  - rename the cookie off `sj_` to `__Host-poised_session` where `Secure` allows it.
- CSRF: reject POST, PUT, PATCH and DELETE whose `Origin` is not our own (a small middleware in `middleware.go`).
- **Tests:** integration tests for each flag state, the Origin check, and the cookie attributes.

## Step 2: database sessions

- **Migration `00005_sessions.sql`.** Table `sessions`:
  - `id_hash` (sha256 of a 32-byte random id; the raw id lives only in the cookie)
  - `kind` (`staff` | `patient`)
  - `user_id` or `episode_id`
  - `created_at`, `last_seen_at`, `expires_at`
- **Lifetimes:**
  - Staff: 8 h maximum, 30 min idle.
  - Patients: 30 min idle (C2). Staff values are a proposal, pending the manager.
- Replace `auth.Issue`/`auth.Verify` with `auth.NewSessionID` and `auth.HashID`. Reuse the `crypto/rand` + base64url pattern from `episode.NewToken` (`internal/episode/episode.go`).
- `requireUser`:
  - looks up the session by hash;
  - checks expiry and idle time;
  - updates `last_seen_at`, throttled to one write a minute;
  - puts the user in the context.
- Logout deletes the row. Add a "sign out everywhere" query for later use.
- Today `requireUser` (`middleware.go:29`) only verifies the JWT and never reads the DB. After this step it reads the session row on every request.
- **Existing fake tests mint cookies with `auth.Issue(secret, userA, …)`** (`httpapi_test.go:68` `do`, `versions_test.go:70` `call`, `optionlists_test.go:40` `doList`). Give the test server a session lookup it can fake (a small interface, like `RoleLister` in `middleware.go:24`), so these helpers keep working without a DB.
- `TOKEN_SECRET` is no longer needed for sessions. Keep it until the dev login moves to sessions too, then drop it.
- **Tests:**
  - **Unit:** id hashing and expiry maths.
  - **Integration:** login → request → logout → the same cookie gets 401; the idle timeout; the absolute timeout.

## Step 3: trusts and the role ladder (data part of C1)

- **Migration `00006_trusts.sql`:**
  - `orgs` (id, name, code, status, `data_region` default `eu-west-2`, `settings jsonb`);
  - `hospitals.org_id`;
  - `platform_admins`;
  - `memberships` reshaped to (user_id, org_id, hospital_id NULL = whole trust, role ∈ `clinician` | `super_clinician` | `admin`);
  - questionnaires and option lists get `org_id`.
  - It's a fresh app (A-1), so reshaping is allowed; seed data is rewritten.
- **Role ladder in Go:** one ordered type with `atLeast(role)` (`saas-requirements.md:91`). Existing checks map as follows (`hasRole` call sites in `chapters.go`, `episodes.go`, `optionlists.go`, `questionnaires.go`):
  - `clinician` → clinician;
  - `author` / `publisher` / `reviewer` → **super_clinician** (`saas-requirements.md:87`: super clinicians author, review and publish; marked *Proposed*);
  - `hospital_admin` → admin;
  - `viewer` has no equivalent. A-5 says everyone in a hospital sees all its data, so any member can read.
- **`store.ForOrg(orgID)`:** for now it returns the one pool. Every patient-data handler goes through it (§13 rules 1–3).
- **Routes move to `/api/o/{oid}/h/{hid}/…`, and pages to `/o/:oid/h/:hid/…`:**
  - `requireHospital` becomes `requireMembership`, which resolves the org and hospital and allows trust-scoped members.
  - Frontend impact: 31 path literals in 14 files (`lib/auth.tsx`, `app/h/[hospitalId]/…`, `EpisodePage.tsx`, `QuestionnaireEditor.tsx`, `useChapters.ts`, `OptionLists.tsx` and others).
  - `Me` gains orgs; the hospital switcher is grouped by trust.
  - The `Role` type in `lib/auth.tsx` and the role gates (`QuestionnaireList.tsx`, `QuestionnaireEditor.tsx`, `app/h/[hospitalId]/layout.tsx`) switch to the ladder.
- **Seed (`cmd/seed/main.go`):**
  - Today it seeds Hospital A and B (fixed UUIDs `…0a` and `…0b`), Alex (author + publisher), Val (viewer), Cara (clinician) and Bea (author, Hospital B), plus 3 patients in Hospital A.
  - It is idempotent (`ON CONFLICT`).
  - Put A and B in **two trusts**, so isolation can be tested by hand.
  - Map the users onto the ladder: Alex → super_clinician, Cara → clinician, Bea → super_clinician in trust B. Add one trust admin.
  - Val (viewer) has no equivalent; it becomes a clinician or is removed. **Ask.**
- Role checks to convert (the exact sites):
  - `canEdit` (`chapters.go:31`, author)
  - `readOptionList` / `deleteOptionList` (`optionlists.go:108,126`)
  - `questionnaires.go:62,149,173,215`
  - `clinicianOnly` (`episodes.go:22`), with its callers in `episodes.go` and `validation.go`
  - The 403 messages name the old roles ("Only authors can …") and must be reworded.
- **Tests:**
  - integration tests for the five cases on every route that moves;
  - trust A admin → 404 on trust B;
  - a hospital-scoped clinician can't see another hospital;
  - the ladder (a clinician can't publish);
  - a removed membership → refused on the next request.

## Step 4: staff sign-in with Cognito (MFA off)

- **Manual setup (manager or owner):**
  - one Cognito user pool in `eu-west-2`, email as the username, no self sign-up, MFA **off**;
  - managed login;
  - an app client with authorization-code + PKCE, and callback `http://localhost:3000/auth/callback`. Confirm against the AWS Cognito docs that an http localhost callback is allowed.
  - Recorded in a short `docs/` note (no CDK, since F4 is out of scope).
- **Go:**
  - `POST /api/auth/callback {code, verifier}` exchanges the code at the Cognito token endpoint.
  - It verifies the ID token: RS256, JWKS from the pool's issuer URL, plus `iss`, `aud` and `exp`.
  - It maps `sub` to `users.cognito_sub` (first login matches by verified email), then creates a staff session (Step 2).
  - **New dependency:** a JWKS/JWT verifier, either the already-present `golang-jwt/jwt/v5` plus a small JWKS fetch, or a library. Choose the smaller one and ask first (CLAUDE.md: keep it simple).
- **Invites:**
  - `POST /api/o/{oid}/staff/invites` → Cognito `AdminCreateUser` + a membership.
  - Behind a `Cognito` interface with a recording fake in tests (`aws-sdk-go-v2`, cognito-identity-provider only).
  - An existing email adds only a membership (STF-02).
- **Web:**
  - `/login` redirects to Cognito with a PKCE verifier kept in `sessionStorage`.
  - A new `/auth/callback` page posts the code.
  - The dev user list is shown only when `/api/dev/users` answers.
- **Tests:**
  - **Unit:** ID-token verification against a fake key set (good, expired, wrong `aud`, wrong `iss`, wrong key).
  - **Integration:**
    - the callback with a fake token exchanger creates a session;
    - an unknown `sub` with no invite → refused;
    - a suspended trust → refused;
    - the invite flow, using the fake Cognito.

## Step 5: patient sign-in (link + Continue + date of birth)

- **Migration `00007_login_links.sql`:** `login_links`
  - `token_hash`, `kind` (`invite`)
  - `episode_id`
  - `expires_at` (NULL = until the episode closes), `used_at`
  - `failed_dob_attempts`, `locked_at`
- **Must fix first:** `patientHQ` returns `patient.dateOfBirth` to anyone holding the link (`internal/httpapi/patient.go:85`, from `GetEpisodeByToken`, `episodes.sql:72-81`). Remove it from the patient response, or the DOB check proves nothing. The DOB to compare against is `patients.date_of_birth` (`pgtype.Date`).
- Today the token is stored in plain text (`episodes.patient_token`, `00004_episodes.sql`) and sent to clinicians (`GetEpisode`, `episodes.sql:46` → `patientToken`).
- Episode creation (`episodes.go:136`) stores only the hash. Reuse `episode.NewToken` for the raw value. The clinician sees the link once, at creation, because a hash can't be shown again. Today `EpisodePage.tsx:44` rebuilds it from `patientToken` on every visit. **Open question** (see below).
- **Routes:**
  - `POST /api/p/links/{token}/continue`: checks the link exists and isn't locked, and returns no data.
  - `POST /api/p/links/{token}/dob {dob}`: compares with `patients.dob`. Five wrong attempts lock the link; a match creates a patient session (Step 2).
  - Existing `/api/p/{token}/…` routes switch to `/api/p/…`, authorised by the patient session, not the URL token (`internal/httpapi/patient.go`).
  - `chapter.ForPatient` and `FilterAnswers` stay as they are.
- **Rate limit:** per link and per IP on `/continue` and `/dob` (in-memory for local; note it for Lambda later).
- **Web:** `app/p/[token]/page.tsx` shows a Continue screen, then a date-of-birth screen, then `PatientHQ`. `PatientHQ.tsx` calls `/api/p/...` without the token.
- **Tests (integration):**
  - the link alone returns no HQ;
  - a wrong date of birth 5× locks the link;
  - a correct date of birth → a session, then the HQ loads;
  - a bad or locked token → 404;
  - the patient session can't reach staff routes.

## Step 6: audit log

- `audit_log` (append-only): who (staff user or patient session), action, org, hospital, entity, when.
- Written for sign-in success, failure and lockout, and for every patient-data read or write (STF-06). Export comes later with C1's admin screens.
- **Tests:** rows are written for the login events and for one patient-data read.

## Not in this plan

- AWS deploy, CDK, Lambdas and React/Vite (F2–F4).
- MFA, and Entra/NHSmail SSO.
- SMS/email sending (the link is printed to the API log locally).
- NHS number, patient accounts across trusts, the patient home and reminders (C2).
- Settings inheritance and opt-ins (C1).

## Open questions for the manager (answer before the step that needs them)

1. **Authoring role:** is `super_clinician` = author + reviewer + publisher, as `saas-requirements.md:87` (*Proposed*) says? This decides the Step 3 mapping.
2. **Route move to `/api/o/{oid}/…` in this work** (Step 3, 31 frontend paths), or keep `/api/h/{hid}` and resolve the trust from the hospital?
3. **Staff session lengths:** 8 h maximum and 30 min idle?
4. **Patient link visibility:** shown once at creation (stored as a hash only), or kept viewable (stored encrypted)?
5. **The AWS account and Cognito pool:** who creates them, and when (Step 4 is blocked until then)?
6. **When does MFA go on?** Proposed: before any real patient data (C10).
7. **Read-only users:** the new ladder has no `viewer`. Is a read-only staff role still needed, or is every member at least a clinician?

## Critical files

- **Go:**
  - `apps/api/internal/auth/token.go` (replaced)
  - `internal/httpapi/{middleware.go, auth_handlers.go, router.go, patient.go, episodes.go, questionnaires.go, chapters.go, optionlists.go}`
  - `internal/episode/episode.go` (reuse `NewToken`)
  - `cmd/api/main.go` (env: `DEV_LOGIN`, Cognito settings)
  - `cmd/seed/main.go`
  - `db/migrations/` (new 00005–00008), `db/queries/` (new `sessions.sql`, `links.sql`, `orgs.sql`)
- **Web:**
  - `apps/web/src/lib/{api.ts, auth.tsx}`
  - `app/login/page.tsx`, a new `app/auth/callback/page.tsx`
  - `app/p/[token]/page.tsx`
  - `app/h/[hospitalId]/layout.tsx` (moves under `/o/[orgId]/`)
  - `features/patient/PatientHQ.tsx`, `features/episodes/EpisodePage.tsx`
- **Docs:** `docs/decisions.md` (the deviations above), the Status sections of `docs/plans/c1-trusts-staff.md` and `c2-patients-signin.md`, `docs/plans/README.md` (status table), `.env.example`.

## Verification

- `make test`, `make lint` and `make test-integration` pass after every step. Each step's new tests fail before its code exists.
- **Manual, in the browser (`make dev`):**
  1. With `DEV_LOGIN` unset, `/api/dev/users` → 404.
  2. Staff: `/login` → Cognito → callback → hospital picker. Logout, then reuse the old cookie with curl → 401.
  3. Trust isolation: sign in as a trust B user; trust A URLs → "not found".
  4. Patient: create an episode and open the link in a private window → Continue → wrong date of birth 5× → locked. A new link with the right date of birth → HQ. The clinician-only set is never shown.
  5. Idle: wait past the idle timeout (shortened by env in dev) → asked to sign in again.
- Run `/update-wiki` at the end of each step.

## Status

| Step | Status |
| --- | --- |
| 0. Integration harness | Done 6 Oct (not committed yet) |
| 1. Hardening | Done 6 Oct (not committed yet) |
| 2. Database sessions | Done 6 Oct (not committed yet) |
| 3. Trusts and the role ladder | Done 6 Oct (not committed yet) |
| 4. Staff sign-in with Cognito | Done 6 Oct (not committed yet); pool details in [cognito.md](../cognito.md) |
| 5. Patient sign-in | Done 6 Oct (not committed yet) |
| 6. Audit log | Not started |

**Answers to the open questions (6 Oct, from the developer):** 1. yes, `super_clinician` authors, reviews and publishes; 2. routes moved now to `/api/o/{oid}/h/{hid}/…` and `/o/:oid/h/:hid/…`; 3. staff sessions 8 h / 30 min idle (configurable); 4. the link stays viewable, **sealed** (AES-GCM, key `LINK_KEY`), not shown once; 7. no read-only role: `viewer` is gone. 5. the developer created the Cognito pool in the AWS account they confirmed (6 Oct). Question 6 (when MFA goes on) is still open.

**Differences from the plan:**
- Step 1: `APP_ORIGIN` added. Behind the web app's `/api` proxy the API sees `Host: localhost:8080`, so a browser sending only `Origin` was refused until the app's origin was trusted. The cookie rename moved to Step 2.
- Step 2: the `sessions` table has no `kind` / `episode_id` yet (they come with patient sessions in Step 5). No "sign out everywhere" query yet (nothing calls it). Expired rows aren't cleaned up yet; that's a job for C2's `jobs` worker.
- Step 3: no `platform_admins` table and no hospital `status` / `settings` yet (no screen or route uses them). Questionnaires and option lists stay owned by a hospital (no trust-level content yet). `forOrg` (the `store.ForOrg` of §13) returns the one pool; the patient link routes and episode events still use it directly until C2 gives the link a trust. `clinicianOnly` was removed: with `viewer` gone, every member is at least a clinician.
- Step 5:
  - The link is sealed so clinicians can see it again (question 4), not shown only once. `login_links` keeps `token_hash` (to look it up) and `token_sealed`. Links made before this have no sealed copy: they still work, and the episode page offers "Make a new link".
  - `POST …/episodes/{eid}/patient-link` replaces the link. The old one stops working, and its patient sessions end.
  - Patient routes: `POST /api/p/links/{token}/continue` and `/dob`, then `GET /api/p/hq`, `PUT /api/p/hq/answers/{cid}`, `POST /api/p/hq/submit` and `POST /api/p/logout`. These replace `/api/p/{token}/…`. C2's `forms/{fid}` routes need `episode_forms` (C3). The patient's page URL stays `/p/:token`; C2 renames it.
  - Patient sessions use the `sessions` table (`episode_id` instead of `user_id`) and their own cookie, `poised_patient`. One browser can then hold a staff and a patient session, and neither is taken for the other.
  - The date of birth is still in the HQ response, because the patient has just confirmed it (patient age drives conditions). It is no longer reachable with the link alone.
  - **No per-IP rate limit.** Behind the `/api` proxy every request comes from one address, and on Lambda each instance would count separately. The lock after 5 wrong dates of birth, and 256-bit link tokens, are the real guard. Throttling belongs at API Gateway (F4).
  - The old fake tests for patient routes and `createEpisode` were replaced by integration tests (testing rule 4).
- Step 4:
  - The pool was made with the AWS CLI, not CDK (F4 is out of scope); [cognito.md](../cognito.md) records every setting. The claude.ai AWS connector needed re-authorising, so the developer's own SSO profile was used.
  - The ID token is checked with `golang-jwt/jwt/v5` (already in the module) plus a small JWKS fetch. There's no OIDC library; it's in `internal/cognito`. Invites use the AWS SDK for Go v2 (`cognitoidentityprovider` only).
  - **Staff list and invites need an admin of the whole trust.** A hospital-scoped admin is refused (403). C1 left this open; it's the simplest rule, and hospital admins can come later.
  - An invite with an email already in Cognito sends nothing and only adds the membership (STF-02). The same scope twice → 409.
  - `GET /api/auth/config` tells the login page which ways to sign in exist. The dev login still shows when `DEV_LOGIN=true`.
  - Sign-out also ends the Cognito session (its `/logout`), so the password is asked again.
  - Tested end to end against the real pool with a made-up user, created with `--message-action SUPPRESS` so no email went out: invite (existing-user path), managed login, callback, Staff page, sign-out. The user was deleted afterwards.
  - Not done: changing or removing a member's role, resending an invite, and an audit of sign-ins (Step 6).
- Integration tests cover every hospital route for no sign-in (401) and another trust (404), plus scope, ladder, removal/suspension and `/api/me`. The full five cases per route (happy path and invalid input for each) are still F1 Step 2.
