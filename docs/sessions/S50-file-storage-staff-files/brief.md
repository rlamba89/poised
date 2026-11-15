# S50: File storage and the clinician's files

**Plan:** C8 ([c8-files.md](../../plans/c8-files.md)), part 1 of 4.
**Requirements:** FIL-01 (the clinician's upload, hospital only or shared), FIL-03, FIL-05 (the staff side); A-14, A-15 ([saas-requirements.md](../../saas-requirements.md) §0, §9). The patient's side and the "file shared" message are S51's.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S49a verified (plan C7 done).

**Why first:** the storage and the `files` table come before anyone can use them. Staff upload first, so S51 has shared files for the patient to see.

**What exists today:**
- **The feature switch** `feature.files` (hospital level, opt-in, on by default; `internal/settings/settings.go:123`). Nothing reads it yet.
- **The episode page's panels** (`EpisodePage.tsx`: Tasks, Observations, Assessments, General notes) and **the POA Summary** (`PoaSummary.tsx`).
- **Patterns to copy:**
  - S46's feature switch: the routes are 403 when it's off, and the episode GET's `observationsOn` tells the web app not to call them.
  - `audited` on every patient-data route (S11).
  - S46's "entered in error" with a reason; S43's automatic General notes lines.
- **The HQ's "Upload a file" question** is a SurveyJS `file` with `storeDataAsText`: the file is stored as base64 inside the answers, up to 5 MB, images and PDF only (`packages/clinical/src/doc.ts:207`). It isn't touched here (S52).
- **The Go AWS SDK** is already used (SES, SMS, Lambda, Cognito). Its `service/s3` has `PresignClient.PresignPostObject` with extra policy conditions (checked 15 Nov 2026, v1.114).
- **Testing rule:** S3 is a recording fake behind a Go interface in the integration tests.
- **MinIO, the plan's local stand-in, stopped publishing images and binaries** (October 2025) and is unmaintained, so question 1 replaces it.

**Ask Rahul first:**
1. **Local storage** (the plan said MinIO).
   - (a) **Go's own local store** (`STORAGE=local`). Files go in a gitignored folder. A dev-only upload route checks a signed policy with the same rules S3's presigned POST enforces: the key, the type, at most 25 MB, and the expiry. A dev-only download sends `Content-Disposition: attachment`. S53 brings real S3 on AWS.
   - (b) An S3-compatible server in docker-compose (SeaweedFS, pinned), used by the real S3 client.

   **Recommended: (a).** It's how the dev login and the captured messages already work: no new service or image to trust. Real S3 is proven on QA in S53, as Cognito was in S12.
2. **Removing a file.**
   - (a) **Staff mark it Removed, with a reason.** The file stays in storage, and staff still see it, struck through, with who, when and why (like an observation entered in error). The patient no longer sees it. It's deleted only when the trust's retention period ends (C10).
   - (b) Delete it for real.

   **Recommended: (a).** A file is part of the clinical record, and the trust keeps the record for its retention years.
3. **Checking the type** (FIL-03 says the server checks it).
   - (a) **Two checks.** Before signing an upload, Go checks that the name's extension and the declared type are on the list, and the upload policy pins that type and the 25 MB limit. After the upload, when it records the file, Go reads the first bytes and checks they match the type (the PDF, JPEG, PNG, HEIC, DOC and DOCX signatures). A mismatch is removed from storage and refused (400).
   - (b) The extension and the declared type only.

   **Recommended: (a).** With no virus scan (A-14), the type list is the main guard, and a renamed file shouldn't get past it.
4. **Changing who can see a file.**
   - (a) **Staff can share a hospital-only file later, or stop sharing one** (the patient no longer sees it). Sharing sends the message (S51), and each change gets an automatic General notes line.
   - (b) Who can see it is set at upload, and never changes.

   **Recommended: (a).** A clinician often uploads first and decides to share later.

## Build

- **Migration** (next number): `files` as the plan says:
  - id, org_id, episode_id, name, content_type, size, visibility (`hospital`|`shared`), uploaded_by_user, uploaded_by_patient (S51), created_at, plus the storage key;
  - plus what questions 2 and 4 need (removed at, by and why; a visibility change's who and when).
  - Every query is scoped to the trust. A hospital-scoped user sees only their hospital's episodes.
- **`internal/storage`** (Go, test-first):
  - the `Storage` interface: sign an upload (key, type, size limit, expiry), read an object's size and first bytes, sign a short-lived download with `Content-Disposition: attachment` and the file's name, delete (for a refused upload);
  - the S3 implementation, unit-tested for the key (`{org_id}/{episode_id}/{uuid}`) and the policy conditions. It's used on AWS from S53;
  - the local store from question 1;
  - the recording fake in `internal/apitest`.
- **Config** (`internal/config`): `STORAGE` (`local`|`s3`), the bucket name (A-20: from config, never in code), the local folder. `CONFIG_STRICT` requires them.
- **Routes** (staff, `/api/o/{oid}/h/{hid}/episodes/{eid}/files…`), each `audited` (`file.upload`, `file.download`, …), with the five integration cases (happy path, no sign-in, wrong role, another trust's data → 404, invalid input):
  - `POST …/files/upload-url`: a wrong type or more than 25 MB → 400;
  - `POST …/files`: records the upload. The object must exist with the size the client gave (else 400), and pass question 3's check;
  - `GET …/files`: the list;
  - `GET …/files/{fid}/download`: a short-lived download link;
  - the visibility change and the removal (questions 2 and 4; a reason is required, 422 without one).
  - Any clinician at the hospital may upload. With `feature.files` off, the routes are 403 and the episode GET says `filesOn: false`.
- **Web:**
  - **a Files panel on the episode page:**
    - upload: choose a file, then pick **Hospital only** or **Shared with the patient**. Progress is shown, and a wrong type or too large a file is explained in plain words before anything is sent;
    - the list shows name, size, who, when and visibility;
    - Download, Share / Stop sharing, and Remove with a reason. A removed file is shown struck through.
  - **POA Summary:** a Files list (name, who, when, visibility; FIL-05).
  - Nothing is ever shown inside our site: no preview, no thumbnail (A-14).
- **E2E:** **J13** starts (`e2e/journeys/test_j13_files.py`):
  - Cara uploads a PDF hospital only and a PNG shared;
  - a text file renamed `.pdf` is refused, and so is a 26 MB file;
  - Download saves the file under its name (testers check downloads in `~/Downloads`; decisions.md);
  - the POA Summary lists both.
  - S51 adds the patient's side. Update `e2e/README.md`.
- **Docs:** CLAUDE.md's content model (files and storage), `make dev` if the local store needs anything, and decisions.md. The hazard and security log entry for A-14 goes in pending.md for C10.

## Not in this session

- The patient's side and the "file shared" message (S51).
- The HQ's file question (S52).
- S3 on AWS (S53).
- FIL-06, a clinician requesting a file (a Could, not planned).
- Virus scanning (A-14), and showing a file inside our site (never).

## Manual test focus

1. As Cara, upload a PDF hospital only and a JPG shared. Both are listed with name, size, who, when and visibility, and both download under their names.
2. A `.exe`, a text file renamed `.pdf`, and a 26 MB file are each refused with a plain reason; nothing is listed.
3. Share the PDF, then stop sharing it; remove the JPG with a reason. The General notes have the lines, and the removed file is struck through.
4. The POA Summary lists the files.
5. Val (Hospital A2) can't see Hospital A1's files, and Trust B's users can't reach them at all. With `feature.files` off for Hospital A1, the panel is gone.
6. Reload and deep links keep the panel.
