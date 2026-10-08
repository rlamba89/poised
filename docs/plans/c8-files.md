# C8: Files

**Goal:** clinicians and patients share files on an episode.

**Requirements:** FIL-01…05 (FIL-06 is a Could), A-14 (no virus scan), A-15 (25 MB). **Depends on:** C2.

## Data and storage

- One S3 bucket **per country deployment** (CDK `DataStack`; the bucket name comes from config, A-20), with keys `{org_id}/{episode_id}/{uuid}`.
- Locally, MinIO in docker-compose.
- `files`: id, org_id, episode_id, name, content_type, size, visibility hospital|shared, uploaded_by_user, uploaded_by_patient, created_at.

## Design

- **Upload:**
  - `POST …/files/upload-url` returns a **presigned POST**, with conditions enforcing the type list (pdf, jpg, png, heic, doc, docx) and `content-length-range` ≤ 25 MB;
  - the client uploads straight to S3, then calls `POST …/files` to record it;
  - Go checks that the object exists and its size.
- **Download:** a short-lived **presigned GET** with `Content-Disposition: attachment`, so files are never shown inside our site (the A-14 mitigation).
- **Patients:**
  - patients upload with `visibility=shared` (it's always visible to the hospital);
  - patients see only `shared` files;
  - sharing a file with the patient sends a notification through the C2 messaging.
- `Deps.Storage` hides S3, with a fake in tests.

## Tests

- **Integration:**
  - a wrong type → 400;
  - more than 25 MB is refused by the presign conditions;
  - a patient never sees hospital-only files;
  - another trust → 404;
  - recording a file whose object is missing → 400;
  - a notification is sent when a file is shared.
- **Unit:** key building, and the presign conditions.
- **End-to-end:** **J13**. A clinician uploads one hospital-only file and one shared file. The patient (at phone size) sees only the shared one, and uploads a photo. The clinician sees it on the episode and in the POA Summary's files list.

## Status

Not started.
