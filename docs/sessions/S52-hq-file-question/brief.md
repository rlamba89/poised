# S52: The HQ's file question uses the file store

**Plan:** C8 ([c8-files.md](../../plans/c8-files.md)), part 3 of 4. Not in the plan: added at the split (15 Nov) because of what the HQ's file question does today.
**Requirements:** FIL-02, FIL-03; A-14 ([saas-requirements.md](../../saas-requirements.md) §9).
*Local phase: runs on Rahul's Mac only. No AWS.*

**Depends on:** S51 verified.

**What exists today:**
- **The HQ's "Upload a file" question** (Add content → File upload) is a SurveyJS `file` with `storeDataAsText: true`, images and PDF, up to 5 MB (`packages/clinical/src/doc.ts:207`).
  - The file is stored **as base64 inside the answers** (`episode_answers`), so every save of that set carries it.
  - The clinician's view and the POA Summary show it **inside our site**, which A-14's mitigation rules out ("files are never shown inline in our origin").
  - It's never pre-filled (`keys.go`'s one-off types).
- After S50–S51: the episode's files (`files`), the storage, the type check, and the patient's uploads.

**Ask Rahul first:**
1. **What the HQ's file question does.**
   - (a) **It uploads into the episode's files.** The file becomes the patient's upload (shared, `uploaded_by_patient`), and the answer holds the file's id.
     - S50's types and 25 MB apply.
     - Staff download it as an attachment, from the set and the POA Summary, like any other file.
     - In the Preview and test cases, nothing is uploaded: the box takes a name only.
     - No existing answers are moved: local data is made up and disposable.
   - (b) Leave it as it is: base64 in the answers, 5 MB, shown inside the site.

   **Recommended: (a).** It gives one store and one set of rules, keeps the answers small, and closes A-14's gap. **With (b), S52 is dropped** and the gap goes to pending.md for C10.

## Build (with answer (a))

- **`packages/clinical`:** the `file` element without `storeDataAsText`, with S50's types and limit; an answer is a list of `{fileId, name}`. Unit tests.
- **Web:**
  - the patient's form uploads through SurveyJS's `onUploadFiles` to the S51 routes, and clears through `onClearFiles` (the file is marked removed);
  - the clinician's Validate copy and the POA Summary show the name and a Download link, never the file;
  - Fill in for the patient uploads as staff, on the patient's behalf.
- **Go:**
  - a save may name only files on that episode that the same patient (or staff on their behalf) uploaded (400 otherwise);
  - `chapter.FilterAnswers` and the evaluate service treat the answer as before (no score reads a file).
  - Integration cases for the save check.
- **E2E:** in J13, the patient answers an HQ's file question with a photo, and the clinician downloads it from the set.

## Not in this session

- The signature pad (a small drawn image, not a file; it stays in the answers).
- S3 on AWS (S53).

## Manual test focus

1. In the editor, add a File upload question; in the Preview, it takes a name and uploads nothing.
2. A patient answers it with a photo; the photo is also in the episode's files, marked "Uploaded by the patient".
3. The clinician's set and the POA Summary show the name with Download, not the picture.
4. A 26 MB file and a `.exe` are refused in plain words.
