# S51: Patients and files

**Plan:** C8 ([c8-files.md](../../plans/c8-files.md)), part 2 of 4.
**Requirements:** FIL-01 (the patient is notified when a file is shared), FIL-02, FIL-05 (the patient's uploads in the staff lists) ([saas-requirements.md](../../saas-requirements.md) §9).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS: messages are captured.*

**Depends on:** S50 verified.

**What exists today (after S50):**
- The `files` table and `internal/storage` (the local store; the fake in tests). The staff Files panel, and the POA Summary's Files list.
- **The patient's home** (`PatientHome.tsx`): one card per episode, with Information under it (S36, `/api/p/episodes/{eid}/information`). The patient's routes are `/api/p/…`, behind the patient session, each `audited`.
- **Messages to the patient:**
  - the status messages (S21, EPI-08: hospital switches and templates in Settings, "Messages when the status changes"). A send that fails is noted, never refused (`statusMessage` in `httpapi/statuses.go`);
  - `/qa/messages` shows what was captured.

**Ask Rahul first:**
1. **The "file shared" message** (FIL-01).
   - (a) **On for every hospital by default**, by SMS and email to whichever contacts the patient has.
     - One message per share: several files shared at once send one.
     - Its template sits in Settings beside the status messages.
     - The text never names the file. It says the hospital has shared a document and links to the patient's sign-in.
   - (b) Off by default, like the status messages.

   **Recommended: (a).** FIL-01 says the patient is notified (a Must). The file's name stays out of an SMS.
2. **Where the patient sees files.**
   - (a) **A Files section under each episode's card on their home**, like Information. It lists the files shared with them and their own uploads, with **Add a file**: a photo from the phone's camera, or a file.
   - (b) A page of its own.

   **Recommended: (a).** It's where the patient already finds everything about that operation.
3. **Can the patient remove their own upload?**
   - (a) **No.** They ask the hospital, and staff can mark it removed (S50).
   - (b) Yes, while the episode is open.

   **Recommended: (a).** It's simpler, and the hospital may already have acted on the file.

## Build

- **Patient routes** (`/api/p/episodes/{eid}/files…`), each `audited`, using S50's storage and checks:
  - the list: the shared files plus their own uploads, never a hospital-only or removed file;
  - the upload link and the record. A patient's upload is always `shared` and marked `uploaded_by_patient` (FIL-02);
  - the download.
  - Integration cases: happy path, no session, another patient's episode → 404, another trust's → 404, invalid input. A hospital-only file never appears, and its download is 404.
  - The builder sets which episode states allow an upload, and until when shared files stay listed (e.g. upload while open, list until archived), and records it.
- **The share message:** sent when a file is uploaded as shared or shared later (S50's question 4), as question 1 answers. It goes through the existing messaging (captured locally). A failed send is noted on the episode, never refused. There's a setting and template if question 1 says so, with Go unit tests for the template.
- **Web:**
  - the patient's Files section at phone size (390×844), as question 2 answers. HEIC from an iPhone and JPG from Android are accepted. Type and size errors are in plain words before anything is sent;
  - **staff side:** a patient's upload shows "Uploaded by the patient" in the Files panel and the POA Summary.
- **E2E:** **J13** completes:
  - a clinician uploads one hospital-only file and one shared file;
  - the message is captured;
  - the patient (at phone size) sees only the shared one, downloads it, and uploads a photo;
  - the clinician sees the photo on the episode and in the POA Summary's files list.
  - Update `e2e/README.md`.

## Not in this session

- The HQ's file question (S52). S3 on AWS (S53).
- FIL-06 (a file request in the patient's to-do list; a Could).
- Messages between the patient and the hospital (A-15: no messaging yet).

## Manual test focus

1. As Cara, share a file with a made-up patient. One message is captured (SMS and email as the contacts allow), and it doesn't name the file.
2. The patient, signed in at phone size, sees only that file (not the hospital-only one), downloads it, and uploads a photo and a PDF. A 26 MB file and a `.exe` are refused in plain words.
3. Cara sees both uploads marked "Uploaded by the patient", on the episode and on the POA Summary.
4. Cara stops sharing the file, or removes it: the patient no longer sees it after a reload.
5. A second patient can't see the first one's files (a typed URL is 404).
