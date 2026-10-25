# S34: The Video element

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 4 of 6.
**Requirements:** INF-01, INF-02, A-6 ([saas-requirements.md](../../saas-requirements.md) §0, §7).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S33 verified.

**What exists today:**
- The editor's element kinds are in `packages/clinical/src/doc.ts` (`KINDS`, `kindOf`, `newElement`); the Add bar is `AddContentBar.tsx`; custom SurveyJS types are registered in `questionTypes.ts` (Medication, Admissions, BMI, Profile).
- S33's content library and its search route.
- Pre-fill (S25) copies a patient's earlier answers by key; signatures, uploads and BMI are never copied (`oneOff` in `apps/api/internal/keys/keys.go`).
- QA's web pages send `Referrer-Policy: no-referrer` (`infra/web_stack.py`). Check YouTube's current embed rules before building: its player may refuse to play without a referrer, in which case the iframe needs its own `referrerpolicy`.

**Ask Rahul first:**
1. **A video picked from the library: copy or link?**
   - (a) **The questionnaire keeps its own copy** of the link, title and length. Changing or retiring the library item later doesn't change questionnaires already made.
   - (b) The questionnaire points to the library item, so a later change shows everywhere, published versions included.

   **Recommended: (a).** Published versions stay exactly as they were signed off.
2. **"I have watched this video" in a later questionnaire.** Should a tick carry over from the patient's earlier questionnaire?
   - (a) **No, always asked again.**
   - (b) Yes, like other answers.

   **Recommended: (a).** It's about this operation.

## Build

- **`packages/clinical`, test-first:**
  - the link check (the same list as S33's Go check, kept in step by a test) and the **embed link builder**: YouTube through `https://www.youtube-nocookie.com/embed/<id>`, Vimeo through `https://player.vimeo.com/video/<id>?dnt=1` (with the private-link hash when there is one). Only a recognised video ID ever becomes an iframe;
  - a new element kind, **Video**: link, title, length, and an optional "I have watched this video" tick that can be made required (INF-02). The tick is the element's answer;
  - a publish problem for a video with no link or a link that fails the check.
- **Editor:** Video in the Add bar. Its settings: paste a link, or **Choose from library** (search by title and tag, S33's route), the tick and Required. The card shows the title and length.
- **Patient view:** the title, the length, the embedded player (captions are the video site's own), and the tick. A required tick must be ticked before the set is complete. Check it works at phone size and on WebKit.
- **Clinician view and Validate:** the video and whether the tick is ticked (read-only for the clinician). The POA Summary is unchanged.
- **Go:** the tick is never pre-filled (answer 2: add the video type to `oneOff` or the skip list), and the key checks ignore it. Unit test.
- **Integration test:** a patient can save and submit the tick's answer; a set with a required unticked video can't be submitted.
- **J10 completed** as the plan describes: the author adds a library video too, and the patient (phone size) sees the video but not the capture block. The test checks the iframe's link only, and blocks the real video site, so it runs offline.

## Not in this session

- Information blocks (S35); the patient's Information on the home and the estimated time (S36).
- Knowing whether the patient really played the video (no tracking, by design).

## Manual test focus

1. Paste a YouTube link, a `youtu.be` link and a Vimeo link: each plays in the preview through `youtube-nocookie.com` or with `dnt=1`. Any other link is refused.
2. Choose a video from the library by tag: title and length fill in. Retire the library item: the questionnaire still shows its copy.
3. Make the tick required: the patient (phone size) can't finish the set until it's ticked. Optional: they can.
4. Validate shows the tick. A second HQ for the same patient doesn't pre-fill the tick.
5. Video titles can be translated with the rest of the HQ.
