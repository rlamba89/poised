# S53: The files bucket on AWS

*AWS phase (12 Oct): runs with Rahul there. Never in an automatic run unless Rahul names it.*

**Plan:** C8 ([c8-files.md](../../plans/c8-files.md)), part 4 of 4. C8 is done after this session.
**Requirements:** FIL-03, A-14, A-15, A-20 ([saas-requirements.md](../../saas-requirements.md) §0, §9, §13).
**Depends on:** S52 verified (or dropped). Rahul there, with the AWS profile (`PROFILE=…`).

**What exists today (after S50–S52):**
- `internal/storage`'s S3 implementation, unit-tested but never run against S3, and the local store used everywhere so far.
- QA on AWS: `infra/api_stack.py` (the Go Lambdas, the per-country values from `infra/countries.py`), the web stack (its headers send no `connect-src`, so an upload straight to S3 isn't blocked), and the pipeline that deploys QA and runs the journeys on it.

**Ask Rahul first:** nothing. The plan settles the bucket: one per country deployment, its name from config, keys `{org_id}/{episode_id}/{uuid}`.

## Build

- **CDK (Python, `infra/`):** the bucket per country deployment, its name from `infra/countries.py` (A-20):
  - all public access blocked, encryption at rest (S3-managed keys; KMS is a C10 question), TLS only;
  - CORS allowing a POST from the web app's address only, and the download's GET;
  - no lifecycle rule yet (retention is C10's);
  - `staff-api` and `patient-api` may put, get and delete objects in it, and nothing else may.
  - CDK tests (`infra/tests`) for the bucket, its policy, CORS and the permissions.
- **Config:** `STORAGE=s3` and the bucket name for both APIs on AWS. `CONFIG_STRICT` holds.
- **On QA:**
  - deploy (by hand first, then the pipeline at the next push); J13 passes on QA;
  - check by hand that S3 itself refuses a 26 MB upload and a changed type, past the signed policy;
  - check that a download comes from S3's address with `Content-Disposition: attachment`;
  - note the results in the build notes.
- **Docs:** `infra/README.md` (the bucket), saas-requirements §13 if anything differs.

## Not in this session

- Training and Production stages, KMS keys, lifecycle and retention, the hazard log entry (C10).

## Manual test focus

1. On QA, J13's journey by hand: a clinician shares a file, the patient (phone size) downloads it and uploads a photo, and the clinician downloads that.
2. A tampered upload (a larger file or a different type than the one signed) is refused by S3.
3. The bucket can't be read without a signed link, and an old link has expired.
4. The pipeline's run after the push is green, J13 included.
