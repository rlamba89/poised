# S12 build notes: staff sign-in with Cognito and MFA

Builder, automatic run (`/run-sessions`), 14 Oct 2026.

## Answers (answers.md, decisions.md §2 14 Oct)

1. **Feature plan: Essentials.** Checked the AWS docs on 14 Oct: managed login exists only in **Essentials and Plus** (Lite has only the classic hosted UI); TOTP MFA is in every plan; **Lite and Essentials are both free to 10,000 monthly active users** (permanent free tier), then $0.0055 vs $0.015 per user; Plus has no free tier. So the cheapest plan with managed login is Essentials, at £0 until 10,000 active staff. Clear-cut, so not asked. Sources: [feature plans](https://docs.aws.amazon.com/cognito/latest/developerguide/cognito-sign-in-feature-plans.html), [pricing](https://aws.amazon.com/cognito/pricing/).
2. **QA test users: yes** (built as `cmd/qa-users`, QA only).
3. **Widened:** the testers' tools need a signed-in platform admin; Amplify sends security headers. Both built.

## What was built

**Go (`apps/api`)**
- `internal/config`: the staff pool, `STAFF_POOL_ID`, `STAFF_POOL_CLIENT_ID`, `STAFF_POOL_DOMAIN` (all or none locally; required under `CONFIG_STRICT`; on AWS read from SSM via `*_PARAM`, like the web address). **`DEV_LOGIN` is refused under `CONFIG_STRICT`.** `GET /api/config` gains `staffSignIn: {devLogin, cognito: {authorizeUrl, logoutUrl, clientId, redirectUri, signedOutUri} | null}` (nothing secret: a public client with PKCE).
- `internal/cognito`:
  - `Verifier` (ID token: RS256 against the pool's keys, issuer, audience = our client, `token_use=id`, expiry, verified email) and `JWKS` (fetched once per Lambda instance, refetched at most once a minute for an unknown key id);
  - `AWS`, the real client: `AdminCreateUser` (email marked verified; an existing pool user is fine), `ResendInvite` (creates a missing user; nothing for someone who set a password), `Exchange` (the token endpoint, with the PKCE verifier; a refused code is `ErrCode`);
  - `TOTP` (RFC 6238), used only by `cmd/qa-users`;
  - `Recorder` gained `Code`/`Exchange` for tests.
- `internal/httpapi`: **`POST /api/auth/callback {code, verifier}`** (only when there's a pool): exchange, verify, find the user by `cognito_sub` else by the verified email (then store the sub), then `startStaffSession` (shared with the dev login: suspended → 403, expired invite → 403, then our cookie). Never creates an account. **`qaTool`** guards every `/api/qa/*` route: open with the dev login on, else a signed-in platform admin (401/403).
- `internal/serve`: the real `cognito.AWS` when the pool is configured.
- `cmd/platform-admin`: the real pool when `STAFF_POOL_ID` is set (`make aws-platform-admin`).
- **`cmd/qa-users`** (`setup`, `code <who>`, `password <who>`): QA only (refuses any other `STAGE`); the seed's staff (fixed ids) get a pool user, a generated password and an enrolled authenticator; stored in the SecureString `/sj/qa/test-users`. Idempotent.
- Queries `GetUserByCognitoSub`, `SetCognitoSub` (the `cognito_sub` column existed since S08).

**CDK (`infra`)**
- **`auth_stack.py`** (`poised-<env>-auth`, after WebStack): the pool (Essentials, email username not changeable, TOTP required, no self sign-up, email recovery, Cognito's own email, invitation and reset templates, 7-day temporary password, QA: deleted with the stack; other stages: retained with deletion protection), the managed login domain (version 2) and branding, the web client (code + PKCE, no secret, callbacks from the web address; `ALLOW_ADMIN_USER_PASSWORD_AUTH` on QA only), the three SSM parameters, staff-api's `AdminCreateUser` on this pool.
- `api_stack.py`: no `DEV_LOGIN`; reads the pool's parameters. `web_stack.py`: security headers (`X-Frame-Options`, CSP `frame-ancestors 'none'`, HSTS, `nosniff`, `no-referrer`). `countries.py`: `dev_login` removed. `stage.py`: the auth stack.
- `pipeline_stack.py`: on QA, Database-and-web runs `qa-users.sh setup` (Cognito admin actions only on the pool tagged `poised-env=<env>`, `PutParameter` only on `test-users`); E2E runs `e2e-users.sh` first (read-only on `test-users`).
- Scripts `qa-users.sh`, `e2e-users.sh`, `platform-admin.sh`; Makefile `qa-users`, `qa-code`, `qa-password`, `e2e-users`, `aws-platform-admin`; `deploy-qa` includes `qa-users`. `.qa-users.json` is gitignored.

**Web (`apps/web`, TypeScript: explained below)**
- `src/lib/signin.ts`, `src/pages/Login.tsx`, `src/pages/AuthCallback.tsx` (route `/auth/callback`), every staff **Sign out** through `signOut`, `src/features/qa/access.ts` for the tools' "Sign in as a platform admin" message.

**E2E (`e2e`)**
- `support.py` signs in by the deployment's `staffSignIn`: the dev stub locally, Cognito's managed login on QA (once per user per run, cookie reused; a refused code retries once with the next one); the tools as Sam Superadmin (`qa_request`, `qa_page`). `captured`, `invite_path`, `link_path` lost their `request` argument. `pyotp==2.9.0`.
- Marks: `dev_login` (`test_staff.py`, `test_j6_onboarding.py`), `cognito_only` (new **`test_cognito_signin.py`**: password + code, wrong ones refused, sign-out also from Cognito, suspended trust refused, no dev login, tools need an admin, security headers).
- J1 and the trusts journey sign in with `sign_in` instead of clicking the dev list; J7 and J8 use the tools through `qa_page`.

## The TypeScript, for Rahul

- **`src/lib/signin.ts`:**
  - `startSignIn` makes two random 43-character strings, a *verifier* and a *state*, keeps them in this tab's `sessionStorage`, and sends the browser to Cognito's `/oauth2/authorize` with the SHA-256 of the verifier (`challengeOf`).
  - `finishSignIn` reads Cognito's redirect (`?code=…&state=…`), refuses it unless the state is the one this tab kept (so another site can't push a code into Poised), and returns the code with the verifier, once.
  - `logoutUrl` / `signOut`: Poised's `POST /api/logout`, then Cognito's `/logout?client_id=…&logout_uri=<app>/login`, so Cognito's own session ends too (otherwise the next Sign in would skip the password).
- **`Login.tsx`** shows **Sign in** when the config has Cognito, and the dev list when it has the dev login (both locally if both are on). A refused sign-in arrives with its message in the navigation state.
- **`AuthCallback.tsx`** posts `{code, verifier}` to `/api/auth/callback`, then `/api/me`, then goes to the user's start page; a failure goes to `/login` with the message. A module-level map makes the page's double effect run (React's development StrictMode) finish the sign-in once, since a code works once.
- **`features/qa/access.ts`** turns 401/403/404 from the tools' API into a sentence.
- No token ever reaches the browser: only Go sees the ID token, and the browser keeps Poised's own HttpOnly cookie, as with the dev login.

## Decided by the builder (automatic mode)

- **AuthStack after WebStack, pool IDs through SSM** (no cycle; the invitation can name the app's address). See decisions.md.
- **Find the user by sub, else by the verified email, and re-store the sub.** Safe because only admins create pool users (email marked verified), there's no self sign-up and the client can't change the email; and it keeps sign-in working after the QA pool is re-created (a new sub for the same email).
- **No account is created at sign-in** (someone Cognito knows but Poised doesn't gets 403 "There's no Poised account for …").
- **The tools stay open where the dev login is on** (local): anyone can sign in as the superadmin there, so a check adds nothing, and the local journeys and testers keep working unchanged.
- **Variables `STAFF_POOL_*`, not `COGNITO_*`:** Rahul's `.env` holds another project's `COGNITO_*`, which make exports; with the first names `make dev` stopped ("not set: COGNITO_USER_POOL_ID").
- **The pool's domain prefix** includes 8 hex of the account's SHA-256 (prefixes are unique across accounts in a region).
- **The CSP is only `frame-ancestors 'none'`** (the brief's ask); a full CSP could break SurveyJS's and Mantine's inline styles, and isn't needed for framing.
- **`Referrer-Policy: no-referrer`** added with the others (S14 suggested it for the hosting).
- **Journeys that invite and then sign in as the invitee run only locally** (`dev_login`): on QA the password is in an email, and Cognito's email is ~50 a day. Their API paths are integration-tested; the email is H-1.
- **The scripts hand the AWS CLI's credentials to Go** when a profile is named (the Go SDK couldn't refresh the SSO token the CLI could).
- **Test fix in `test_audit.py`:** it read the list before the filter's reload finished (on QA the counts happened to match). It now waits for the patient's rows to go.

## Deployed to QA by hand (14 Oct)

`make deploy-stacks`, `aws-db`, `qa-users`, `deploy-web` (PROFILE=AdministratorAccess-026090536258), twice (the second after the `STAFF_POOL_*` rename). Pool `eu-west-2_XoA7mik1F` (Essentials, MFA on, COGNITO_DEFAULT email, tagged `poised-env=uk-qa`); managed login at `https://poised-uk-qa-35d61cce.auth.eu-west-2.amazoncognito.com`. `make aws-platform-admin EMAIL=sam.superadmin@poised.example` reached QA's database and pool ("The account exists; it is a platform admin.").

**⚠ The pipeline still has S07's definition.** Until S12 is committed and pushed, **any push to `master` (including the three S07 docs commits waiting) would redeploy QA from `master`: the old Lambdas with the open dev login.** AuthStack would stay but be unused. So: **push only after S12 is committed** (see "What Rahul pushes").

## Test results (14 Oct)

- `make lint test`: go vet + tsc clean; Go unit tests pass; Vitest 155 + 116 passed; CDK tests **92 passed**.
- `make test-integration`: every package `ok` (new: `TestCognitoCallback` (9 cases), `TestCognitoCallbackOnlyWithAPool`, `TestQAToolsNeedAPlatformAdminWithoutTheDevLogin`, `cmd/qa-users`; `TestConfig` now strict with Cognito).
- `make e2e` locally against `make dev` (started by the builder for the run, then stopped): **43 passed, 5 skipped** (the Cognito-only journeys).
- `make e2e` against QA (`E2E_BASE_URL=https://main.d33kh0gi8vk83v.amplifyapp.com`, after `make e2e-users`): **35 passed, 13 skipped** (the dev-login-only journeys) in 2 min 22 s.

## Suggested commits (Rahul or the conductor commits after verification)

1. **Go: the staff pool config, the Cognito client, ID-token check and `POST /api/auth/callback`; no dev login under strict; the tools need a platform admin without the dev login** (`apps/api/internal/{config,cognito,httpapi,apitest,serve}`, `db/queries/users.sql` + `internal/db`, `cmd/platform-admin`, `go.mod`/`go.sum`, `.env.example`), with their unit and integration tests.
2. **`cmd/qa-users`** and its tests.
3. **CDK: AuthStack, security headers, no `dev_login`, the pipeline's test-users steps**, scripts and Makefile targets (`infra/`, `Makefile`, `.gitignore`), with the CDK tests.
4. **Web: Cognito sign-in (PKCE), the callback page, sign-out through Cognito, the tools' sign-in message** (`apps/web/`), with Vitest.
5. **E2E: sign-in by deployment, the Cognito journey, marks** (`e2e/`).
6. **Docs:** decisions.md, pending.md, plans and sessions READMEs, c1 plan status, infra/README, e2e/README, apps/web/AGENTS.md, CLAUDE.md, this folder (brief/answers changes from the planning run included).

## What Rahul pushes (after the verifier commits S12), and H-3

**Before:** QA already runs S12's code, deployed by hand. Don't push anything to `master` until S12's commits are on it: S07's pipeline definition would put the open dev login back on QA.

**Push:** from the repo root, `git push origin master`. That sends the three S07 docs commits that are waiting (`8d09ca1`, `0dcfd9d`, `ab90771`) and S12's commits from the verifier, all in one push. The pipeline runs once, for the newest commit.

**What the first run should show (H-3)**, in **CodePipeline → `poised`**, about 15 to 20 minutes:
1. **Source:** the newest S12 commit.
2. **Build / Synth:** green. `make lint test` (CDK tests **93 passed**, Vitest 160 + 116), `make test-integration` (every package `ok`), the web build, `make synth`.
3. **UpdatePipeline / SelfMutate:** runs, because the pipeline itself changed (the QA test-users step, the E2E step's users file, and their permissions). The run then **restarts from Source by itself**. That's expected, not a failure.
4. **Assets:** uploads the Lambdas.
5. **`uk-qa` deploys:** `poised-uk-qa-api` and `-web` (no changes, or tiny ones, after the hand deploys) and **`poised-uk-qa-auth`** (no change).
6. **`Database-and-web`:** goose "no migrations to run", the seed ("trusts: 2, users: 8, patients: 3"), then **"8 test users can sign in; their passwords and secrets are in SSM /sj/qa/test-users."** (it keeps the same passwords and secrets), then "Deployed." for the web app.
7. **`E2E`:** "The test users' sign-ins are in e2e/.qa-users.json (8 users).", then the journeys signing in through Cognito: **36 passed, 13 skipped**. The skipped ones are `test_staff.py` and `test_j6_onboarding.py` (dev login only) on Chromium. The log ends with the S3 address of the results.

**If `Database-and-web` fails with AccessDenied** on a `cognito-idp:Admin…` action (the policy's `aws:ResourceTag/poised-env` condition), QA stays usable: the test users from the hand-run `make qa-users` are still there. Then `infra/pipeline_stack.py` needs fixing.

**If `E2E` fails on sign-in** with "Invalid or expired code" twice, someone signed in as the same test user in that minute. **Retry failed actions** on the stage.

**Afterwards:** QA should still need credentials. `/api/dev/users` should be 404, and `/login` should show only **Sign in**.

## Test report 1 (14 Oct): 15 passed, 0 failed, 3 blocked

Fixed test-first (the conductor's direction: hospital workstations are shared):
1. **Finding 1: Back after a staff sign-out showed the previous page (patient names) from the browser's back/forward cache.**
   - Amplify sends **`Cache-Control: no-store` on every page**, and `public, max-age=31536000, immutable` on `/assets/` (hashed names; they were `max-age=0` before, so this also saves a revalidation per file). CDK test `test_pages_are_never_kept_for_back_after_sign_out`.
   - **Sign out empties the page** (`blankPage`) before leaving for Cognito's sign-out, and **a page restored from the back/forward cache reloads** (`src/lib/restore.ts`, installed in `main.tsx`). Vitest for both.
   - Checked on QA with Chromium's back/forward cache switched on: after Sign out, Back lands on `/login` with the old page's state gone. Journey `test_cognito_signin.py` now goes Back after sign-out.
   - **Gotcha:** Amplify shows changed custom headers only after the branch's next deployment (`make deploy-web`); the pipeline does that on every run.
2. **Finding 2: a sign-in Poised refused (suspended trust) left Cognito's session, so Sign in went straight back in as the refused person.** `AuthCallback` now calls `refuseSignIn`: the message waits in the tab's `sessionStorage`, and the browser goes through Cognito's `/logout` back to `/login`, which shows it once. Vitest; the suspended-trust journey now clicks Sign in after the refusal and expects the password page (on QA: passed). Staff Sign out already ended Cognito's session (T-04 passed); it's unchanged apart from `blankPage`.
3. **Finding 3 (the S11/S15 filters under automation):** in pending.md as a hand re-check.

**T-08, T-09, R-05 (blocked: shared QA data or a real email):** hand checks for Rahul. Automated cover: **T-08** by a new journey in `test_trusts.py` ("a removed membership is refused on the next request": Val's only access is removed while she's signed in, her reload shows "You don't have access to this hospital" and the API 404s, then it's given back; runs locally and on QA) plus S09's `TestRemoveStaffAccess`/`TestRemoveStaff` and the callback's "removed from every trust" case; **T-09** by `TestCognitoCallback` (the invite calls `CreateUser`) and S09's invite tests, the real email being H-1; **R-05** by J4 (validates, "Patient answered"), which passes on QA.

Deployed to QA by hand again (`deploy-stacks`: web headers; `deploy-web`). Test plan: T-13, T-14 added for the retest; T-04 step 3 sharpened.

**Results after the fixes (14 Oct):** `make lint test`: clean; Vitest 160 + 116, CDK **93** passed. `make test-integration`: all `ok`. Journeys locally: **44 passed, 5 skipped**. On QA: **36 passed, 13 skipped**.

## Test report 2 (14 Oct, retest): 9 passed, 0 failed, 3 blocked (hand checks)

Both findings from report 1 are fixed on QA: T-13 (Back after sign-out), T-04 (tightened), T-14 (a refused sign-in also signs out of Cognito), plus spot checks of T-01, T-02, T-06, T-07 and T-11. **Exploring 1** is in pending.md, not fixed in S12 (no code change at this stage, the conductor's direction): a second open tab keeps showing patient names after a sign-out in another tab until a click, then shows a red "Please sign in." box instead of going to `/login`. It's older than S12. Exploring 2 was the test tool, not the app.

## Hand checks for Rahul

- **T-08, T-09, R-05** (test-plan.md), and the patient's pages at a real phone width (the tester's window couldn't be resized), and the two filters (pending.md, finding 3).

- **H-1:** a real invite email end to end (test-plan.md).
- **H-2:** a forgotten-password reset by email.
- **H-3:** the first pipeline run after the push.
