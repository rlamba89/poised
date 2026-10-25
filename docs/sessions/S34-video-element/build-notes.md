# S34 build notes: the Video element

Builder, automatic run, 25 Oct 2026.

## Answers used

[answers.md](answers.md) (Rahul, 25 Oct), both as recommended: (1) a video picked from the library is **copied** into the questionnaire (link, title, length), so later library changes don't reach it; (2) the "I have watched this video" tick is **never pre-filled**.

S33's notes for S34 (pending.md): the TypeScript link check runs Go's `video_links.json` (done); the picker's role check (see "Decided by the builder").

## What was built

### `packages/clinical` (TypeScript, for Rahul to review)

- **`video.ts`** (new):
  - `checkVideo(link)`: the same rules as Go's `content.CheckVideo` (YouTube watch / `youtu.be` / embed / `youtube-nocookie.com` embed; Vimeo `vimeo.com/<id>`, private `vimeo.com/<id>/<hash>`, `player.vimeo.com/video/<id>[?h=<hash>]`; https only, no user name or port). Written with a plain pattern, not the browser's `URL`, so it reads links the way Go does. **Its test runs every case in `apps/api/internal/content/testdata/video_links.json`**, the file Go's test runs, so the two can't drift apart.
  - `embedUrl` / `videoEmbedFor`: the player's address, built **only from a checked ID**: `https://www.youtube-nocookie.com/embed/<id>`, or `https://player.vimeo.com/video/<id>?dnt=1` (plus `&h=<hash>` for a private link). Anything else gives no player at all.
  - `formatLength` / `parseLength` moved here from the web's `content.ts` (which now re-exports them), `tickPatch` (the tick on with its words, or off with Required too), `chooseLibraryVideo` (copies a library item's link, title and length; the old title's translations go, being of another video), `videoProblems` (publish: no link, a link that isn't a YouTube or Vimeo video, no length).
  - `QuestionVideoModel`, registered as SurveyJS type `video`: properties `videoUrl`, `videoLength` (seconds), `watchTick`, `tickText` (translatable). Its answer is `true` when ticked; unticking clears it. It's required only when it has the tick and Required; **for clinicians it's read-only and never required**. A missing required tick says "Please tick “I have watched this video” to go on." (or the author's own message).
- **`doc.ts`:** the kind `video` ("Video"), `VIDEO_TYPE`, `newElement("video")` (title "Video", no link yet); not an "answer kind" (its tick has its own settings; no Logic "Required when" or conditions on it); never moved into a repeating group. `tickText` is a translatable key.
- **`keys.ts`:** no "Always ask fresh", "Same as…" or "Save to library" on a video. **`translations.ts`:** the tick's words are listed as "Tick". **`publish.ts`:** adds `videoProblems`.

### `apps/web` (TypeScript)

- **`features/video/VideoQuestion.tsx`:** how SurveyJS draws a Video, registered once by importing it in `PatientHQ`, `ValidateSet` and `PreviewPage`: "Length 3:20", the player (an iframe at 16:9, the full width up to 720px), the tick, and for clinicians "The patient ticked this." / "The patient hasn't ticked this.". The iframe has `referrerPolicy="strict-origin-when-cross-origin"`: **YouTube's player needs to know which site embeds it** (YouTube's "Required minimum functionality" page asks for the referrer and recommends exactly this policy), and QA's pages send `Referrer-Policy: no-referrer`. The iframe's own policy wins for its request; QA's CSP sets only `frame-ancestors`, so nothing in `infra/` changes.
- **`features/editor/VideoSettings.tsx`:** Title, Link (with "Plays from YouTube (privacy-enhanced mode)" or "Vimeo (no tracking)", or the problem), **Choose from library**, Length (m:ss), the tick switch, and Required (only with the tick). The library window searches `GET /api/o/<trust>/content/search` by words (`q`) and by tag buttons (`tag`), lists videos only, and **Use this** copies one in. `features/video/picker.ts` shapes the search (unit-tested).
- **Editor:** **Video** in the Add bar (greyed inside a repeating group); its card shows the title, "YouTube · Length 10:35" (or "No link yet", "Not a YouTube or Vimeo link", "No length yet", in red) and the tick with "(optional)" when not required.
- `globals.css`: the player and tick styles.

### Go

- **`internal/keys`:** `video` joins `notAnswered`, so its tick is never pre-filled (by key or wording), linked with "Same as…", saved to the library, or locked at publish.
- **`internal/chapter/video.go`:** `RequiredTicks(content)`: the required ticks every patient sees (no condition on the video, a group or page around it, or the whole Question Set; not clinician-only; not in a repeating group), and `Unticked`.
- **`internal/httpapi/patient.go` (submit):** the patient's and staff's "Send" is refused (409, "Please tick “I have watched this video” for “Your anaesthetic” before sending your answers.") while such a tick isn't `true` in the saved answers. **Why only those:** Go never evaluates conditions, so a tick behind one is left to the browser's check, as every other required question is (plan 2.1).

### End-to-end

- **J10 completed** (`test_j10_authoring.py`): the author adds a library video by its tag (title and length fill in) and makes its tick required; the patient on a phone (WebKit) sees the player's address only, can't finish until ticked; Cara sees it ticked, read-only; the POA Summary doesn't mention it. There's no capture block (S31 dropped), so the "patient doesn't see the capture block" part of the plan isn't there.
- **`test_video.py`** (new): pasted links (refused site, Publish refused naming the set, length), the Preview's YouTube and private-Vimeo addresses, the title and tick in German; the patient at phone size (player inside the screen at about 16:9, no sideways scroll; an optional tick doesn't hold them up); the clinician sees "hasn't ticked".
- `support.block_video_sites(context)`: YouTube and Vimeo answered with an empty page, so the journeys run offline and check only the player's address.

## Decided by the builder (automatic mode)

- **A video must have a length to publish**, as a library video must (S33), so S36's estimated time always has it. (The brief named only the link; the length is the same reason as S33's.)
- **The tick's words are stored with the element** (`tickText`, "I have watched this video") when the author turns the tick on, so they're translated with the HQ (the brief's "titles can be translated"). The author can't reword them in Settings (the Translate view only).
- **Choose from library is offered to whole-trust super clinicians and admins only**, as the question library is (S24), matching the search route's check. Anyone who can edit a questionnaire can paste a link. S33's note: new questionnaires are always the trust's, so a hospital-only super clinician edits none today.
- **The Go submit check holds back only ticks with no condition** (above). A required tick behind a condition is checked by the browser.
- **No video inside a repeating group** (its tick would be asked once per row), as for Clinical Summaries.
- **No Clinical (clinician-only) switch on a video:** a video is for the patient.
- **The clinician sees "The patient ticked this." / "The patient hasn't ticked this."** under the read-only tick, so it's clear on Validate.
- **The library window's tags are buttons** (not a dropdown, like S33's checklist topics), with whole labels; its rows' buttons don't shrink at narrow widths.

## Tests (each new test failed first, then passed)

- **TypeScript:** `packages/clinical/src/video.test.ts` (66: every link case from Go's file, the player addresses, never an address from an unknown link, lengths, the kind, no Always ask fresh, no repeating group, publish problems, the survey question: required only with the tick, read-only and not required for clinicians, only a tick is an answer, the message, translations, the library copy); `apps/web/src/features/video/picker.test.ts`.
  - Honest note: `video.ts`'s first version (link check, embed, model) was written in the same step as its test file; the later pieces (`chooseLibraryVideo`, the required message) and the Go and web parts were test-first.
- **Go unit:** `keys_test.go` `TestVideoTickIsNeverPrefilledNorLocked`; `chapter/video_test.go` (`TestRequiredTicks`, the whole-set condition, `TestUnticked`).
- **Go integration:** `httpapi/video_integration_test.go` `TestVideoTick` (fixture `apitest/testdata/video.json`): the patient's JSON keeps the videos; a required tick with no answers, or `false`, can't be sent (409, the message names only the required video; nothing written), nor by staff on the patient's behalf; ticked, it's saved and sent (Ready for review); a second HQ for the same patient pre-fills their other answer but not the tick (checked to fail with `video` taken out of `notAnswered`); an HQ without videos is sent as before. The routes' sign-in, role and tenancy cases are unchanged and covered by `TestSavePatientAnswers`, `TestSubmitPatientHQ` and `TestCompleteOnBehalf`.

### Results (25 Oct)

- `make lint`: passed.
- `make test`: Go all ok; web 25 files, 260 passed; clinical 20 files, 248 passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (all journeys, against a dev stack on spare ports 3091/8091 on the shared `sj` database, stopped afterwards): **79 passed, 5 skipped, 1 failed**: `test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`, because I was running screenshot scripts as Cara at the same time (its "Listed episodes" row wasn't the newest). Run again alone: `test_audit.py` **6 passed**. J10 and `test_video.py` passed again after the last changes (3 passed).

## Hand checks for Rahul

- **Full screen on a real phone** (iPhone Safari and Android): the tester's 390px frame couldn't go full screen.
- **A Vimeo video really plays** (the tester played YouTube; Vimeo's player loaded but wasn't played).
- **A real video plays on QA** (YouTube and Vimeo), where the pages send `Referrer-Policy: no-referrer`: the iframe's own `strict-origin-when-cross-origin` should let YouTube's player work. Locally there's no such header, and the journeys block the video sites, so only QA shows it.
- **On a real iPhone (Safari)**, a video plays and goes full screen. The journeys use WebKit at iPhone size, but blocked video sites.
- **Go's refusal of an unticked required video** (409) can't be reached from the browser, which stops first; it's covered by `TestVideoTick`.

## Suggested commits

1. `S34: the Video element in packages/clinical` — `packages/clinical/src/{video,doc,keys,translations,publish,questionTypes,index}.ts`, `video.test.ts`, `tsconfig.json` (JSON imports for the cases file).
2. `S34: Go never pre-fills a video's tick and holds back an unticked required one` — `internal/keys/keys.go` + test, `internal/chapter/video.go` + test, `internal/httpapi/patient.go`, `httpapi/video_integration_test.go`, `apitest/testdata/video.json`.
3. `S34: the editor's Video settings, library picker and the player` — `apps/web/src/features/{video,editor}/…`, `features/content/content.ts`, the three survey pages' import, `globals.css`, `apps/web/AGENTS.md`.
4. `S34: J10 completed and the video journey` — `e2e/journeys/{test_j10_authoring,test_video}.py`, `e2e/support.py`, `e2e/data/video-hq.json`, `e2e/README.md`.
5. `S34: docs` — this folder, `docs/pending.md`, plan and session status, `docs/decisions.md`.

## Test rounds

- Test plan written 25 Oct ([test-plan.md](test-plan.md)).
- **Test report 1** (25 Oct): **all passed, 24 of 24.** **S34 tested.**
  - **T-09's step 1 was wrong in the plan:** Ada can't retire a Preopsis item (it's read-only on a trust's page). The tester used a trust video instead and every expected result held; the plan's T-09 now does that. No code change.
  - **Exploring notes** (no code change, as the conductor asked), recorded in pending.md ("From S34's test report 1"): the patient's set-list badges and Continue button cut short at 390px (before S34); the required tick has no marker of its own; a link's `&t=` start time is dropped (by design); YouTube Shorts links are refused (a question for Rahul); the Clinical Summary delete wording. The tester also confirmed Go's 409 for an unticked required video, and the search route's 403/404 for Sue and Bea, with in-page API calls.
