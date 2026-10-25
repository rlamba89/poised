# S34 verification

## Verification 1 (25 Oct 2026, automatic run)

**Verdict: Verified.**

Everything the brief asked for is built, and it follows Rahul's answers.md (25 Oct). Test report 1 passed all 24 cases. All automated tests pass. I found no security problem. The findings below are minor and recorded in pending.md ("From S34's verification 1"). None of them blocks the commit.

### What I checked

- **Scope:** everything in the brief is done:
  - the link check and the embed-link builder (`packages/clinical/src/video.ts`);
  - the Video kind, the tick and Required;
  - the publish problems;
  - the editor's settings and the library picker;
  - the patient, clinician and Validate views;
  - Go never pre-fills the tick (`keys.notAnswered`), and Go refuses a send while a required tick is unticked;
  - J10 is completed, plus `test_video.py`.

  The builder decided two things themselves, both listed in the build notes:
  - a video needs a length to publish;
  - the Go 409 for an unticked required tick.

  Both stay within the brief's intent. Nothing large was added that the brief didn't ask for.
- **The embed link is built only from a recognised ID (security):** `embedUrl` builds the address from a checked ID alone, and nothing else from the link reaches it:
  - a YouTube ID must match `^[A-Za-z0-9_-]{11}$`;
  - a Vimeo ID must match `^[0-9]{1,12}$`;
  - a private Vimeo hash must match `^[0-9a-f]{6,20}$`.

  I checked this in the browser. I saved a draft through the API with 10 hostile links:
  - `javascript:`;
  - `data:`;
  - `evil.example/embed/<id>`;
  - `youtube.com.evil.example`;
  - `user@youtube.com`;
  - `youtu.be\@evil.example`;
  - an encoded quote in `v=`;
  - an encoded `&autoplay=1` in a Vimeo `h=`;
  - `/../` in a Vimeo path;
  - an embed link with `?autoplay=1&origin=evil`.

  In the Preview, 9 of them showed "This video can't be shown" with no iframe. The last one (a real YouTube embed link) became exactly `https://www.youtube-nocookie.com/embed/aqz-KE-bpKQ`, with its query dropped. Go doesn't check `videoUrl` when content is saved (plan 2.1: publish checks run in the browser), so this builder is the only guard, and it holds.
- **The link list is shared with Go:** `video.test.ts` imports `apps/api/internal/content/testdata/video_links.json`, the same file Go's `CheckVideo` test runs, so the two checks can't drift apart.
- **Referrer policy:** every iframe has `referrerpolicy="strict-origin-when-cross-origin"` and `allow="encrypted-media; picture-in-picture; fullscreen"`. QA's pages send `Referrer-Policy: no-referrer`, and its CSP sets only `frame-ancestors`, so the iframe's own policy applies and no infra change is needed. Whether a real video plays on QA is a hand check (below).
- **Tests first:** S34 adds no new route. The submit change is covered by `TestVideoTick`:
  - happy path;
  - "invalid input" (no answer, `false`);
  - staff on the patient's behalf;
  - a second HQ doesn't pre-fill the tick;
  - an HQ without videos.

  The sign-in, role and tenancy cases of the submit and save routes are unchanged and still covered by `TestSubmitPatientHQ`, `TestSavePatientAnswers` and `TestCompleteOnBehalf`. The unit tests are `video.test.ts` (66), `picker.test.ts`, `keys_test.go` and `chapter/video_test.go`. The build notes admit that `video.ts`'s first version was written alongside its tests.
- **Tenant isolation:** no new query. `uncheckedTicks` reads the form's own version chapters and answers, and the form is already resolved inside the trust or hospital scope. I checked the picker's search route in the browser:
  - Alex: 200;
  - Sue (one hospital): 403;
  - Cara: 403;
  - Bea (Trust B): 404.
- **Clinician-only content:** `RequiredTicks` runs on `chapter.ForPatient` output, so a clinician-only video is never required of the patient.
- **A-20:** the grep finds only comments that were there before S34. `repocheck` passes.
- **Docs:** the plan's Status section, `plans/README.md`, `sessions/README.md`, `decisions.md` (S34 entry) and `pending.md` were all updated.

### Browser spot-checks (Chrome, trk profile)

1. **Hostile links → no iframe** (above). The editor cards say "Not a YouTube or Vimeo link" for each of them.
2. **The required tick at phone size (Preview, Phone frame):** the player fits inside the card. **Complete** without the tick gives "Please tick “I have watched this video” to go on."
3. **Clinician view:** the tick is `disabled` with `aria-required="false"`, and "The patient hasn't ticked this." shows.
4. **Choose from library:** the tags show in full. The tag "The anaesthetic" leaves one video, and **Use this** copied its title, link and 10:35, so the card now reads "YouTube · Length 10:35".
5. **Roles on the search route:** as above.

### Findings (all minor, none blocking)

1. **A stale Length error** (`apps/web/src/features/editor/VideoSettings.tsx:75`). On a video with no saved length, type something unreadable ("abc"), tab out, then clear the field and tab out. The field is now empty, but it still shows "Type the length as minutes and seconds…" instead of "Add the video's length." `onSave("")` doesn't reset `lengthDraft`, and `seconds` doesn't change, so the reset effect never runs. I reproduced this in Chrome. It's cosmetic: the card still says "No length yet" and Publish is still refused. The fix is to call `setLengthDraft(null)` before `patch({ videoLength: undefined })`.
2. **The picker briefly shows the last search when reopened** (`VideoSettings.tsx:118–139`). Reopening clears `search`, but `debounced` keeps the old words for 250 ms, so the old results show for a moment. Each opening also sends the unfiltered search twice: once for the tags, once for the list. It's cosmetic and a small amount of extra traffic.
3. **Go's 409 message always says the default “I have watched this video”** (`apps/api/internal/httpapi/patient.go`, `uncheckedTicks`), even when the tick's words are translated. The browser's message uses the tick's own words, and the browser stops the patient first. Rahul sees this message only from a direct API call.
4. **`RequiredTicks` ignores a fixed `visible: false` or `readOnly: true`** (`apps/api/internal/chapter/video.go:19`). Only expression conditions make a tick "conditional". The editor never writes either property on an element today. If an imported or hand-edited HQ ever has a required tick inside a fixed-hidden or read-only container, Go would refuse every send. One cheap guard is to treat `visible == false` and `readOnly == true` as conditions too.
5. **A narrow race on the tick check:** the check reads the answers inside the submit transaction without a row lock. A save from a second device that unticks between the check and `SubmitPatientHQ` could freeze an unticked required tick. The browser checks it too, so this is very unlikely and does no harm.
6. **Small tidy-ups:**
   - `ElementCard.tsx:291` types "I have watched this video" again instead of using `DEFAULT_TICK_TEXT`;
   - `content.ts` re-exports `formatLength`/`parseLength` instead of its callers importing them from `@poised/clinical`;
   - `packages/clinical/tsconfig.json` had its `lib`/`include` arrays reformatted. Only `resolveJsonModule` matters.

### Test results (25 Oct)

- `make lint`: passed.
- `make test`:
  - Go: every package ok (`repocheck` included);
  - web: 25 files, **260 passed**;
  - clinical: 20 files, **248 passed**;
  - CDK: **124 passed**.
- `make test-integration`: every package ok.
- `make e2e` (against the conductor's `make dev`): **79 passed, 5 skipped, 1 failed**. The failure was `test_j7_patient_invites.py::test_j7_invite_by_text_and_email_then_date_of_birth_and_send`: 5 captured texts to its random `07700 900nnn` mobile instead of 1. This is the known flake recorded in pending.md: J7's mobile is one of 1,000 on a well-used database. S34 doesn't touch J7 or messaging. The re-run failed the same way once, then **6 passed**. J10 and `test_video.py` passed.

### Hand checks for Rahul

- **A real YouTube and a real Vimeo video play on QA**, where the pages send `Referrer-Policy: no-referrer`. The iframe's own `strict-origin-when-cross-origin` should make this work.
- **Full screen on a real phone** (iPhone Safari and Android).
- **A Vimeo video really plays.** The tester played only YouTube.

### Test data left behind

- The draft questionnaire "VERIFY S34 evil links" (Trust A, created by Alex Author) is still in the local database. It has never been published, and it holds the hostile-link videos. Delete it from the questionnaire list if it gets in the way.

### Commits

These are the builder's suggested commits, made locally on `master` (not pushed). `docs/market/` and `run-state.md` are left out.

1. `S34: the Video element in packages/clinical`
2. `S34: Go never pre-fills a video's tick and holds back an unticked required one`
3. `S34: the editor's Video settings, library picker and the player`
4. `S34: J10 completed and the video journey`
5. `docs: S34 verified — build notes, test plan and report, verification, decisions, pending, session status`

Next: `/build-session S35`.
