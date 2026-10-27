# S41 test report 1

**17 cases: 15 passed, 2 failed, 0 blocked** (T-01–T-12 and R-01–R-05). Both failures are the same Minor bug, F-1. Run in automatic mode.

- **Code:** `1ec4c87` plus uncommitted S41 work (the tree isn't clean: 36 changed or new files). The evaluator reports `clinical@1ec4c87a3274-dirty`.
- **When:** 26 Oct 2026, 23:49 to 27 Oct 2026, 00:15.
- **Where:** http://localhost:3000, with the conductor's `make dev` (API :8080, web :3000, evaluate 127.0.0.1:8090). Browser: Chrome through Claude in Chrome.
- **Users:** Alex Author and Cara Clinician; the patient through the captured invite link.
- **Test data (all made up):**
  - HQ: "MT S41 2350" (`034586f7-9429-4c30-a1ab-927b80487598`), versions 1 to 3 published and version 4 left as a draft with a logic problem.
  - Episodes:
    - T-03: `1e86ebd6-…`
    - T-04: `5daf9b7a-…`
    - T-05: `f2f067d7-…`
    - T-09: `dedf63a3-…`
    - T-10: `13c58e0e-…`
    - T-12: `9b2cf5e6-…`
    - R-03: `37adf5cf-…` and `fde2c17f-…`
- **Phone size:** I couldn't test it. `resize_window` reported success, but the window stayed 1500×763 (probably full screen). So T-03 and T-09, the patient sends, were done at desktop width. The patient screens didn't change in S41.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 make dev starts the evaluate service | Pass | `node` on 127.0.0.1:8090; health gave `{"ok":true,"evaluator":"clinical@1ec4c87a3274-dirty"}` |
| T-02 build and publish an HQ | Pass | Built as written. The Publish window showed no problems plus the auto-triage line. V1 published, read-only, with "Create new version" offered |
| T-03 a sent HQ's result is stored (patient) | Pass | Normal "Thank you" screen. Ready for review, with "HQ completed by the patient" and no "automatic checks" line. Row: ASA III with the I50.9 reason, High risk fires (exact line), no flags, "Smoker, Heart failure unspecified", 2 sets shown, one patient set complete. The episode page's "Suggested ASA III" line matches. (Desktop width, see above) |
| T-04 low-risk patient, sent on their behalf | Pass | "HQ completed on the patient's behalf". Row: ASA II with the Smoker reason, Low risk fires, no error. The episode stays Ready for review |
| T-05 a red flag is stored and stops Low risk | Pass | ASA II; flags "red: Do you get chest pain when you walk?: Yes"; no rules fire; no error |
| T-06 a clinician's set never counts; no second evaluation | Pass | After Nurse checks was validated, still one row (id 39), unchanged. I also sent again through the API (patient and staff routes): both gave 409 "These answers have already been sent." and still one row |
| T-07 publish through the API is refused while a test case fails | Pass | The "MT smoker" case saved (ASA II, Low risk). After Low risk was changed to "at most I", the case failed, and the Publish window listed "Rules · Test case "MT smoker" fails" with Publish disabled. curl gave `HTTP 422` with the exact error and problem. Still V2 draft after a reload |
| T-08 the same version publishes once the case passes | Pass | Put back to II, so the case passes. curl gave `HTTP 204` and the editor shows V2 published |
| T-09 service stopped: the patient's send still works | **Fail (Minor)** | Everything works except one bullet: the "automatic checks didn't run" note shows **"\| Patient on …"**, not "Automatic" like other automatic lines. See F-1. The "Thank you" screen, Ready for review, the error row (no version, no result, "…connection refused", no answer in it) and the `EVALUATE_FAILED form <id>: …` log line (no answers) are all right |
| T-10 service stopped: filling in for the patient still works | **Fail (Minor)** | Same as T-09: it sends, goes to Ready for review, stores the error row and logs. But the note shows **"\| Patient"** even though Cara sent it (F-1) |
| T-11 service stopped: publishing is refused | Pass | The window showed "The checks couldn't run. Try again in a minute." and it stayed a draft. curl gave `HTTP 503` with the same message. Log: `EVALUATE_FAILED publish checks, version f5c33072-…` |
| T-12 start it again; everything works | Pass | Health ok. The V3 draft published from the window. A new send (on behalf, heart failure Yes) stored row 44: ASA III, High risk, no error, and no "automatic checks" line |
| R-01 (S40) Rules page, Test the whole HQ, the auto-triage line | Pass | Used throughout T-02, T-07 and T-08: rules, editing a rule, test cases going PASS/FAIL, and the line "This version auto-triages: ASA I–II, no red flags → Ready for admission…" |
| R-02 (S39) Suggested ASA on the episode page and POA Summary | Pass | T-03 episode: "Suggested ASA III: Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction" on both |
| R-03 (S34) a required video tick blocks sending | Pass | "Shots 2510 202004 2798" (its tick is required): Finish showed "Please tick "I have watched this video" to go on.", and the server's submit gave 409 with the tick message, with no row stored. After ticking, it sent. (The "E2E Video 2610 …" HQs have a tick that isn't required, so they send without it, as expected) |
| R-04 (S18–S22) validate every set and complete the review | Pass | T-04 episode: both sets validated, Complete HQ review gave Ready for POA. Still one row |
| R-05 Publish window blocks on a logic problem | Pass | The V4 draft's condition tests a later question, so the window lists "2 problems to fix first" and Publish is disabled. No publish request was sent |

## Failures in detail

### F-1 (Minor): "The automatic checks didn't run" is shown as written by "Patient", including when staff sent the HQ

- **Cases:** T-09 (patient sends) and T-10 (staff send on the patient's behalf).
- **Signed in as:** Cara Clinician.
- **URLs:**
  - http://localhost:3000/o/00000000-0000-4000-8000-0000000000fa/h/00000000-0000-4000-8000-00000000000a/episodes/dedf63a3-457f-43c4-8ea7-530c1a391d31 (T-09)
  - …/episodes/13c58e0e-73af-4980-bea8-9bcd56e5c157 (T-10)

**Steps:**
1. Stop only the evaluate service.
2. Send an HQ, either by the patient's link or with Fill in for the patient.
3. Open the episode and look at the General notes.

**Expected:** the plan says the line is "written by nobody, like other automatic lines". Those lines show **"Automatic on …"** (e.g. "Day 2 reminder sent…", "Not responding…").

**Actual:**
- `The automatic checks didn't run. Review this HQ by hand.` shows **"| Patient on 27/10/2026, 00:04"**.
- After a staff send (T-10) it also says **"| Patient"**, right under "HQ completed on the patient's behalf | Cara Clinician". Someone reviewing the episode would think the patient wrote it.

**Database (confirmation):**
- The `episode_events` row has `user_id` NULL and **`automatic = false`**.
- `GeneralNotes.tsx` shows `ev.automatic ? "Automatic" : (ev.userName ?? "Patient")`.
- The line comes from `evaluateSent` in `apps/api/internal/httpapi/patient.go` (`q.AddEpisodeEvent(… Kind: "event", Text: checksDidntRun)`), which doesn't mark it automatic.
- The integration test only checks `User == nil`, so it doesn't catch this.

**Console/network:** none.

**GIF:** `~/Downloads/S41-T09-evaluate-down-patient-send.gif` (the patient's send while the service is down).

## Found while exploring

- **Three submits at once** on one form (staff route, fired together from the page): one 204 and two 409s "These answers have already been sent.", with exactly **one** `form_outputs` row. Good.
- **The evaluate service called directly** (127.0.0.1:8090):
  - `{}` gives 400 "sets must be a list".
  - Text that isn't JSON gives 400 "The body isn't JSON."
  - GET /evaluate gives 405, and an unknown path gives 404.
  - An 11 MB body gives 413 "The body is too big."
  - A set whose content is a string gives 400 "each set needs its id, name, audience and content". The value I put in the content was **not echoed back** and **nothing was logged**.
  - It listens on 127.0.0.1 only.
- **Logs:** while the service was stopped, `make dev` kept the API and web running. Its own evaluate process exited (npm logged "code 143") and stayed down until I started it again.
- **The editor won't save a condition naming a question that doesn't exist** (Code tab: "Unknown question: q_deleted1"), so for R-05 I used a condition on a later question instead.
- **Editor, not S41 (cosmetic or flaky):**
  - **"Save output" in New Disclosure:** the first time I clicked it while the code search list was still open, the dialog closed and **nothing was saved**. Clicking it again after picking the code worked every time.
  - **The ASA guide list:** picking a code adds this list under the form, which moves the buttons down. A click aimed at "ASA III" landed on "ASA IV: Congestive heart failure (Children)" (my mis-click, which I fixed with Edit). Worth knowing when a layout shifts.
- **The card badge on a clinician set's question** reads "PATIENT" (Nurse checks, whose audience is Clinician). This was there before S41, maybe on purpose (it may describe who can see the question). I'm only noting it.

## Console and network errors not tied to a case

- **Console:** no errors or warnings on the episode, editor, Rules or Test pages.
- **Network:** the Chrome extension's request log reports **503** for several POSTs that actually returned 204 (dev login, logout, and the T-12 submit after the restart). curl against the same routes gives 204, and the UI and database show they worked, so this is a reporting error in the tool, not the app. The real 503s were the two publish attempts in T-11, which were expected.

## GIFs

All three are in `~/Downloads/`:

- `S41-T03-happy-path.gif`: the patient's link, date of birth, answers and send.
- `S41-T09-evaluate-down-patient-send.gif`: the patient's send with the service stopped (F-1).
- `S41-T11-publish-checks-couldnt-run.gif`: Publish refused with "The checks couldn't run…".

## State left behind

- **The evaluate service:** exactly one is listening on 127.0.0.1:8090. It's **the copy I started separately** (`nohup make evaluate`, log `/tmp/mt-s41-evaluate.log`):
  - `make` is PID 19854.
  - `node dist/evaluate.cjs` is **PID 19930**.
  - The evaluate process that `make dev` started was stopped in T-09 and isn't running.
  - My copy isn't in `make dev`'s process group, so stopping `make dev` won't stop it. Kill PID 19854 (or 19930) before running `make dev` again, or the new evaluate process will find the port taken.
- **Data:** "MT S41 2350" V4 is left as a draft with a logic problem. The trust's suspension state wasn't touched.

## Hand checks for Rahul

- The patient screens at phone size (390×844) for T-03 and T-09: the window couldn't be resized in this run. S41 doesn't change those screens.
