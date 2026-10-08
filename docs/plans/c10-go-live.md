# C10: Go-live

**Goal:** Training and Production environments that are ready for the first customer and real patient data.

**Requirements:** OPS-01…06, NFR-01/03/05/07/09/10, D-2 (RDS from the first customer). **Depends on:** everything the first customer's contract needs.

## Infrastructure

- **Separate AWS accounts** for Training and Production, added as stages in the pipeline, with a **manual approval** before Production. QA must be green, including the end-to-end tests, before Training can be promoted.
- **Countries (A-20):** Production has one stage per country, each from its entry in `infra/countries.py`. Only the UK exists at go-live. Another country is added with the runbook in `infra/README.md`.
- **Database:**
  - **RDS for PostgreSQL** in our own account (`DataStack`, in the deployment's region; eu-west-2 for the UK): the smallest Graviton instance, encrypted, with point-in-time restore. Multi-AZ if the contract needs it.
  - Production **starts empty on RDS**. QA can stay on Neon.
  - The Lambdas move into the VPC, and outbound traffic goes through VPC endpoints or a NAT gateway (this fixed cost is accepted now).
- **Domains:** custom domains with TLS (ACM), one per country: `uk.<domain>` for the UK (A-20). The cookie domain is settled in F4.
- **Off in Training and Production:** `DEV_LOGIN` and `MESSAGING=capture` (CDK checks this). SES production access and the SMS sender ID must be approved (started in C2).

## Operations

- **Alarms (OPS-05):** Lambda errors, failed jobs, failed SMS and email deliveries, the database, and spend.
- **Undelivered invites** are shown to the hospital.
- **Backups:** a restore is tested before go-live and then every quarter (NFR-10).
- **Break-glass access** for platform staff: time-limited, and written to the audit log (OPS-06).
- **Runbooks:** deploying, rolling back, restoring the database, rotating secrets, and suspending a trust.

## Compliance (start well before the date)

- DCB0129 hazard log and a clinical safety officer (OPS-04). It covers:
  - auto-triage straight to admission;
  - pre-fill;
  - identity linking;
  - no virus scanning.
- NHS DSPT, Cyber Essentials Plus, a DPIA template for customers, and the data processing agreement (OPS-03).
- An independent pen test (NFR-05).

## Tests

- The end-to-end suite runs against Training after each promotion.
- A restore drill.
- A short load check of the patient journey at the contract's expected peak.

## Status

Not started.
