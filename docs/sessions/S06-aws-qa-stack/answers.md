# S06: answers

Asked by the conductor (`/run-sessions S06 S17`) on 14 Oct 2026, before the AWS-phase run started.

1. **Which AWS account?** Account `026090536258` (SSO profile `AdministratorAccess-026090536258`, region eu-west-2) also hosts another project's staging stacks (Asgard).
   - **Answer: use it for now** (Rahul): it's for his own testing until the demo; before sales it moves to another account. **The CDK must need no manual AWS work, so the stacks can be deleted here and re-created in a new account at any time.** Never touch the other project's stacks.
2. **Full automation (widens S06, Rahul's choice: "both"):**
   - an idempotent `make aws-setup` that bootstraps CDK and stores the Neon URL in SSM (`/sj/qa/database-url`, from an env var; skipped if already present);
   - **CDK generates the token secret in Secrets Manager** (no script step; about £0.30/month per secret);
   - **the £5/month budget alarm is in CDK**;
   - `infra/README.md` documents moving to a new account. Only the GitHub connection approval, the Neon sign-up and SES/SMS production access stay manual.
3. **F4 Step 0:** Rahul does his parts (Neon project in AWS London with database `sj`; the Neon URL into SSM; the GitHub CodeConnections connection). CDK is already bootstrapped in eu-west-2.
4. **Costs:** approved (close to £0 idle, target under £1/month, plus pennies per pipeline run).
5. **Permissions for this run:** builders and verifiers may **deploy and destroy Poised's own QA stacks** in this account (never other stacks), and **push to GitHub** (`rlamba89/poised`). **No real messages** without a separate yes.
6. **No domain yet:** QA uses Amplify's default `…amplifyapp.com` address. A real domain comes before the demo.

7. **`make aws-setup` done** (Rahul, 14 Oct): `/sj/qa/database-url` and `/sj/alert-email` are in SSM (eu-west-2). The Neon project is in AWS London.

## 14 Oct 2026: AWS set-up done

Rahul ran `make aws-setup PROFILE=AdministratorAccess-026090536258 NEON_DATABASE_URL=… ALERT_EMAIL=…`; it finished with "Done". The conductor confirmed `/sj/qa/database-url` (SecureString) and `/sj/alert-email` exist in eu-west-2 (values not read).
8. **QA is open to anyone with the URL** (test report 1: the dev login API, the captured messages with live links, the rate-limit reset and /qa/jobs need no sign-in; no security headers).
   - **Answer: leave it until S12** (Rahul, 14 Oct): S12's real login closes it. Keep the QA URL private meanwhile. The rest goes into S12 (see S12 answers item 3).
