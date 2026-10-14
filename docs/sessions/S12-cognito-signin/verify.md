# S12 verification

## Verification 1 (14 Oct 2026, automatic run)

**Verdict: Verified.** None of the findings blocks the session. Every one is Minor, needs a transient failure or an operator's mistake to show, and can't be used by an outsider. They're recorded in [pending.md](../../pending.md) §3 ("From S12's verification").

**The security points the conductor asked about all hold:**
- The Cognito callback uses PKCE (S256, a 43-character verifier and state in the tab's `sessionStorage`) and checks the state.
- The ID token is checked for its RS256 signature against the pool's JWKS, the issuer, the audience (our client), `token_use=id`, the expiry (required) and `email_verified`. `alg none`, HS256-with-the-public-key, other keys, other pools and other clients are all refused, with unit and integration tests.
- No account is created at sign-in.
- Suspended trusts and expired invites are refused through the shared `startStaffSession`.
- The dev login can't be reached on AWS:
  - Go refuses `DEV_LOGIN` under `CONFIG_STRICT`;
  - `countries.py` has no `dev_login`;
  - a CDK test covers it;
  - both Lambdas' environments have no `DEV_LOGIN`;
  - `/api/dev/*` is 404 through Amplify and through the `execute-api` address.
- `/api/qa/*` returns 401 without a sign-in through both addresses, and 403 for anyone who isn't a platform admin (integration test).
- The test users' secrets are only in the SSM SecureString `/sj/qa/test-users`. The stack outputs are the pool ID and the login URL only, and `e2e/.qa-users.json` is gitignored with mode 0600.
- staff-api's role may only call `cognito-idp:AdminCreateUser`, on this one pool.
- The security headers are present on pages and on `/assets/`.
- Sign-out goes through Cognito's `/logout`.
- Locally, `/api/config` reports `cognito: null`, so `.env`'s unrelated `COGNITO_*` values don't leak in (the variables are `STAFF_POOL_*`).

### Findings (all Minor, worst first)

1. **A failure after a successful callback leaves the Poised session alive** (`apps/web/src/pages/AuthCallback.tsx:36-41`).
   - If `POST /api/auth/callback` succeeds (the cookie is set) but `GET /api/me` then fails, the page treats it as a refused sign-in: it signs out of Cognito and shows "sign in again" on `/login`. Our 12-hour cookie stays valid, though.
   - On a shared workstation, the next person to open `/` is signed in as the previous one.
   - Needs a transient `/me` failure straight after the callback.
   - **Fix:** in the `catch`, `POST /api/logout` before `refuseSignIn`.
2. **Sign-out skips Cognito's `/logout` when `GET /api/config` failed at page load** (`apps/web/src/lib/config.ts:43-44`, used by `signOut` in `src/lib/signin.ts`).
   - `staffSignIn()` falls back to `{devLogin: true, cognito: null}`, which is fail-open on AWS. Our cookie is cleared, but the managed login's session isn't, so the next Sign in on that computer goes straight back in as the previous person (test report 1's finding 2).
   - Needs the config fetch to fail (a 5-second timeout on a cold start) on a page where someone is signed in.
   - **Fix:** retry `loadConfig()` in `signOut` before deciding, or never assume the dev login when the config is missing.
3. **Every OAuth error from the token endpoint becomes "sign in again", with no log line** (`apps/api/internal/cognito/aws.go:130`; `auth_handlers.go:130-133` doesn't log `ErrCode`).
   - `invalid_client`, `unauthorized_client` and a `redirect_uri` mismatch are deployment faults, not the user's. After a domain change (APP_URL changes but the app client's callback doesn't), every staff sign-in would fail with nothing in CloudWatch.
   - **Fix:** only `invalid_grant` is `ErrCode`; log the rest as server errors, and log the `ErrCode` reason too.
4. **The email fallback re-links an account that already has a different `cognito_sub`, silently** (`apps/api/internal/httpapi/auth_handlers.go:144-156`).
   - This was the builder's decision, so a re-created QA pool keeps working. It's safe today: only admins create pool users, the email is verified and can't be changed by the user, and there's no self sign-up.
   - On a real stage, a pool user deleted and re-created for the same email would quietly take over the account's link, with no audit row.
   - **Decide with C10:** audit or log a re-link, or allow it only where `cognito_sub IS NULL` outside QA.
5. **JWKS: the lock is held across the fetch, and a failed fetch isn't rate-limited** (`apps/api/internal/cognito/idtoken.go:106-124`).
   - While Cognito's JWKS is unreachable, concurrent callbacks queue and each waits up to 5 s.
   - It's reached only after a successful code exchange, so an outsider can't drive it. Set `fetched` on failure too.
6. **`cmd/qa-users setup` saves to SSM only at the end** (`apps/api/cmd/qa-users/main.go:168`).
   - If one user fails after others were re-created, their new authenticator secrets are lost. The next run then keeps the stale stored secret, because the user is found and enrolled.
   - QA tooling only: deleting the user in the pool and re-running fixes it. Save after each user.
7. **`make aws-platform-admin` for an email that already has a Poised account doesn't create its pool user** (`apps/api/cmd/platform-admin/main.go`, `add`). It reports success, but that person can't sign in on AWS. It's an operator tool: say so in its output, or call `CreateUser` (which is idempotent) anyway.
8. **Pipeline:** the Database-and-web role's Cognito admin actions depend on `aws:ResourceTag/poised-env` (`infra/pipeline_stack.py:176-188`). Cognito's IAM page lists ABAC as "Partial", so this is proven only by the first pipeline run (H-3). The build notes give the fallback.
9. **Nits:**
   - `internal/serve/serve.go:61`'s comment still says `COGNITO_*`;
   - `internal/config/config_test.go` isn't gofmt-clean (the `TestLoadCognito` map);
   - the same `signOut(() => navigate("/login"…), () => api("/logout"…))` is in four places (Home, AdminLayout, HospitalLayout, QuestionnaireEditor);
   - `infra/scripts/lib.sh`'s mode changed from 644 to 755 (harmless).

**Checked and not a problem:** `/code-review` thought Amplify's `'**'` and `'assets/**'` header patterns might both apply to the bundle. On QA, `/assets/index-79YIFPht.js` has one `cache-control: public, max-age=31536000, immutable`, and pages have `no-store`.

### Test results (14 Oct)

- **`make lint test`:** go vet and tsc clean. Every Go package ok. Vitest: 160 and 116 passed. CDK: **93 passed**.
- **`make test-integration`:** every package `ok`, including `TestCognitoCallback` (9 cases), `TestCognitoCallbackOnlyWithAPool`, `TestQAToolsNeedAPlatformAdminWithoutTheDevLogin` and `cmd/qa-users`.
- **`make seed`, then `make e2e` locally** (against a `make dev` I started and then stopped): **42 passed, 2 failed, 5 skipped**.
  - The two failures were J7 (`to_have_count(1)` saw 5) and J8 (4 texts, not 2). Both are the known made-up-mobile collision ([pending.md](../../pending.md): "J7 and J8 flake", 1 of 1,000 numbers against a local database full of old texts). It isn't S12's.
  - J7 passed when re-run; J8 failed again on the next number, then passed.
- **`make e2e` against QA: not run by the verifier.** It signs in with the test users' passwords on Cognito's hosted domain, a non-local host where I don't enter credentials. The builder's run (36 passed, 13 skipped) and the pipeline's E2E step on the first push (H-3) cover it.
- **Read-only AWS checks** (profile AdministratorAccess-026090536258, eu-west-2):
  - **Pool `eu-west-2_XoA7mik1F`:** ESSENTIALS, MFA ON (TOTP only), first factor PASSWORD only (no passkey bypass of MFA), admin-create-only, COGNITO_DEFAULT email, tagged `poised-env=uk-qa`.
  - **Client:** no secret; `code` flow only; scopes `openid email`; callback `/auth/callback` and logout `/login` on QA's address; `PreventUserExistenceErrors` on; no writable attributes. `ALLOW_ADMIN_USER_PASSWORD_AUTH` is on, as intended for QA.
  - **Lambdas:** both have `CONFIG_STRICT=true`, no `DEV_LOGIN`, and `STAFF_POOL_*_PARAM`.
  - **SSM:** `test-users` is a SecureString; the pool parameters are String (not secret).
  - **Stacks:** api, web and auth are all UPDATE_COMPLETE, with no secret outputs.
- **Unauthenticated probes, through Amplify and `execute-api`:**
  - `/api/dev/users` and `/api/dev/login` → 404;
  - all six `/api/qa/*` routes, `/api/me` and `/api/admin/orgs` → 401;
  - a forged cookie → 401;
  - the callback with a bad code → 401, and with a short verifier → 400.
- **Browser spot checks** (Claude in Chrome, no credentials entered):
  1. QA `/login` shows only **Sign in**. Clicking it goes to `poised-uk-qa-…amazoncognito.com/login` with `code_challenge_method=S256`, a 43-character challenge and state, scope `openid email`, and the right `redirect_uri`.
  2. A forged `/auth/callback?code=…&state=…` (a state this tab never made) is refused: the page goes through Cognito's `/logout` back to `/login` with "That sign-in didn't finish. Please sign in again."
  3. QA `/qa/messages` signed out shows "Sign in as a platform admin to use this page." with a Sign in link.
  4. Local `/login` shows only the development list ("local only") and no Cognito button.

### Hand checks for Rahul

- **T-08, T-09, R-05** ([test-plan.md](test-plan.md)): they change shared QA data or send a real email.
- **H-1:** a real invite email, end to end.
- **H-2:** a forgotten-password reset by email.
- **H-3:** the first pipeline run after the push ([build-notes.md](build-notes.md), "What Rahul pushes").
- The patient's pages at a real 390×844 window, and the To/Who filters by hand ([pending.md](../../pending.md), S12).

### Commits (made by the verifier, automatic mode)

1. Go: the staff pool config, the Cognito client, the ID-token check, `POST /api/auth/callback`, no dev login under strict, and the tools need a platform admin (with unit and integration tests).
2. `cmd/qa-users` (with tests).
3. CDK: AuthStack, security headers, no `dev_login`, the pipeline's test-users steps, the scripts and Makefile targets (with the CDK tests).
4. Web: Cognito sign-in (PKCE), the callback page, sign-out through Cognito, back/forward-cache reload, and the tools' sign-in message (with Vitest).
5. E2E: sign-in by deployment, the Cognito journey, and the marks.
6. Docs: decisions, pending, the plans and sessions READMEs, C1's status, the infra and e2e READMEs, AGENTS.md, CLAUDE.md, and this session's folder.

Not committed: `.claude/skills/*` (the conductor's), `docs/sessions/S17-aws-messaging-jobs/`, `reports/`, `research_notes/`, `run-state.md` and `e2e/.qa-users.json`.
