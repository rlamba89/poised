# S12: answers

Asked by the conductor (`/run-sessions S06 S17`) on 14 Oct 2026, before the AWS-phase run started.

1. **The Cognito feature plan:** the builder checks the current AWS docs and recommends the cheapest plan with managed login and TOTP MFA (answer recorded when the builder asks, if it isn't clear-cut).
2. **QA test users:** **yes** (recommended). A small command creates QA staff users with a known password and TOTP secret for testers and e2e; it refuses any other stage. Rahul checks one real invite email by hand once.
3. **Widened by Rahul (14 Oct, after S06 test report 1):** S12 also puts the QA-only tool pages (captured messages, `/qa/jobs`, the rate-limit reset, `/api/qa/client`) behind a signed-in platform admin, and adds security headers to the web app (anti-framing via CSP `frame-ancestors`/X-Frame-Options, HSTS, X-Content-Type-Options). After S12, nothing on QA is usable without credentials.
