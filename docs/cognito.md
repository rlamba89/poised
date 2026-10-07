# Cognito: staff sign-in (development pool)

Staff sign in through Amazon Cognito's managed login (decision A-12; [plans/auth-first.md](plans/auth-first.md) Step 4). Cognito says who someone is. What they may do comes from our `memberships` table.

**MFA is off for now.** This is *pending* Rahul's sign-off; see [decisions.md](decisions.md), 6 Oct. Switch it on in the pool before any real patient data. It needs no code change.

## The development pool

It was created on 6 Oct 2026 with the AWS CLI in `eu-west-2`, in the AWS account the developer confirmed. No CDK yet: F4 brings infrastructure as code. Every resource is tagged `Project=poised`.

| Setting | Value |
| --- | --- |
| User pool | `poised-dev-staff` (`eu-west-2_Z24T6xpDK`), tier Essentials (needed for managed login) |
| Sign-in | Email is the username. Invites only (`AllowAdminCreateUserOnly`); no self sign-up |
| Password | At least 12 characters, with upper case, lower case and a number. Temporary passwords last 7 days |
| MFA | Off |
| Recovery | Verified email |
| Deletion protection | On |
| Managed login domain | `https://poised-dev-4ee8920c.auth.eu-west-2.amazoncognito.com` (managed login version 2, Cognito's default branding) |
| App client | `poised-web-local` (`1v9e5lr0t3sq1ll9tav7138us3`). Public, so no secret: authorization code with PKCE. Scopes `openid email profile`. ID and access tokens last 60 minutes |
| Callback URL | `http://localhost:3000/auth/callback` (Cognito allows plain `http` only for localhost) |
| Sign-out URL | `http://localhost:3000/login` |

## Local setup

Add these to `.env` (`.env.example` lists them):

```sh
COGNITO_POOL_ID=eu-west-2_Z24T6xpDK
COGNITO_CLIENT_ID=1v9e5lr0t3sq1ll9tav7138us3
COGNITO_DOMAIN=https://poised-dev-4ee8920c.auth.eu-west-2.amazoncognito.com
COGNITO_REDIRECT_URI=http://localhost:3000/auth/callback
COGNITO_LOGOUT_URI=http://localhost:3000/login
AWS_PROFILE=<an SSO profile for that account>   # only invites call AWS; run aws sso login when it expires
```

Without `COGNITO_POOL_ID`, only the development login works.

## How it works

1. `/login` sends the browser to the managed login, with a PKCE challenge and a state. The verifier waits in `sessionStorage`.
2. Cognito returns to `/auth/callback?code=…`. The page posts the code and the verifier to `POST /api/auth/callback`.
3. Go swaps the code at the token endpoint and checks the ID token: RS256 with the pool's keys, the issuer, the audience (the app client), `token_use=id`, a verified email, and expiry (`internal/cognito`).
4. Go finds the user by Cognito subject. For someone who existed before Cognito, it matches the email once and saves the subject. No membership in an active trust → refused.
5. Go starts our own session (the `sessions` table). Sign-out ends it, then sends the browser to Cognito's `/logout`, so the next sign-in asks for the password again.

## Inviting staff

A trust-wide admin uses the **Staff** page, which calls `POST /api/o/{oid}/staff/invites`:
- **A new email:** Cognito creates the person and emails them a temporary password (Cognito's own email: about 50 a day; production uses SES, C2).
- **An email already in the pool:** nothing is sent, and the person only gains the new membership (STF-02).

## Recreating or removing it

The exact CLI calls are `create-user-pool`, `create-user-pool-domain --managed-login-version 2`, `create-user-pool-client` and `create-managed-login-branding --use-cognito-provided-values`, with the settings above. To remove the pool, first turn off deletion protection (`update-user-pool --deletion-protection INACTIVE`), then delete the domain, then the pool.
