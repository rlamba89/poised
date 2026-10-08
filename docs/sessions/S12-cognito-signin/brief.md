# S12: Real staff sign-in with Cognito and MFA

**Plan:** C1 ([c1-trusts-staff.md](../../plans/c1-trusts-staff.md)), part 5 of 5. C1 is done after this session.
**Requirements:** STF-05, A-12, ORG-03 (suspended trust refused), A-21.
**Depends on:** S11 verified, and QA plus the pipeline (S06–S07).

**Ask Rahul first:**
1. **The Cognito feature plan** (Lite vs Essentials). Check the current AWS docs for which plan includes managed login and TOTP MFA, and what each costs. Give your recommendation.
2. **QA test users:** a small command creates test users with a known password and TOTP secret, so the tester and e2e can sign in. Real invite emails are checked by Rahul by hand once. **Recommended: yes.**

## Build

- **`AuthStack` (CDK):**
  - takes the country entry, with callback URLs from config;
  - email as the username;
  - **TOTP MFA required;**
  - no self sign-up;
  - managed login.
- **The OAuth code flow with PKCE:**
  1. The callback page posts the code to `POST /api/auth/callback`.
  2. Go verifies the ID token (JWKS; tests use a fake key set) and maps `sub` to the user.
  3. Go sets **our own session cookie**.
- **The real `Deps.Cognito`** (`AdminCreateUser`), so invites send Cognito's email.
- **A suspended trust's users are refused at sign-in.**
- **`DEV_LOGIN` is off in QA,** and the CDK check enforces it.
- **The e2e login helper** uses Cognito with `pyotp` in QA, and the dev login locally.
- **`cmd/platform-admin`** works against QA.

## Manual test focus

All of this is on the **QA URL**, using the test users:
1. Sign in with the password and an authenticator code.
2. A wrong password and a wrong code are both refused, with clear messages.
3. **Forgotten password:** the reset flow, where Rahul reads the email.
4. **An invite:** a new user gets the email (Rahul checks), sets a password and MFA, and lands in the right hospital.
5. A removed membership and a suspended trust are both refused.
6. Sign out.
7. The dev login doesn't exist on QA.
