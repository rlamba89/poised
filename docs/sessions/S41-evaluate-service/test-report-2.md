# S41 test report 2 (retest)

**Retest: 15 passed, 0 failed, 0 blocked, 2 not rerun** (they passed in report 1). **F-1 is fixed**: "The automatic checks didn't run…" now shows as **"Automatic"** after both kinds of send. Run in automatic mode.

- **Code:** `1ec4c87` plus the uncommitted S41 work (the tree isn't clean: 37 changed or new files). The evaluator reports `clinical@1ec4c87a3274-dirty`.
- **When:** 27 Oct 2026, 00:21 to 00:40.
- **Where:** http://localhost:3000, on the conductor's `make dev` (API :8080, web :3000, evaluate 127.0.0.1:8090). Browser: Chrome through Claude in Chrome.
- **Users:** Cara Clinician and Alex Author; patients through their captured invite links.
- **Test data (all made up):**
  - HQ: the round-1 HQ "MT S41 2350" (`034586f7-…`). This round I fixed V4's logic problem and published V4 (T-11/T-12), then made V5 and published it (T-07/T-08). No draft is left.
  - Episodes:
    - T-03: `a71aab1e-…`
    - T-04 and R-04: `76b21ab0-…`
    - T-09: `c49cd1f4-…`
    - T-10: `4afe52a3-…`
    - T-12: `e3b32b67-…`
    - R-03: `14238bbe-…`
- **Phone size:** Chrome won't make its window narrower than **500 px**, so the browser cases ran at 500×701 or at desktop width. To cover 390×844, I opened the T-09 patient's link again in headless Playwright Chromium at 390×844 and took a screenshot of the "Thank you, Kim" screen. The layout is fine and there's no sideways scroll (`/private/tmp/claude-501/-Users-rahul-poised/e354aaca-05a3-4639-ae42-c43325b43018/scratchpad/phone-sent.png`).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 make dev starts the evaluate service | Pass | `node` listening on 127.0.0.1:8090. Health gave `{"ok":true,"evaluator":"clinical@1ec4c87a3274-dirty"}` |
| T-02 build and publish an HQ | Not rerun | Passed in report 1. The publish path ran again in T-08 (API) and T-12 (Publish window) |
| T-03 a sent HQ's result is stored (patient) | Pass | Normal "Thank you, Pat" screen. Ready for review, with "HQ completed by the patient" and no "automatic checks" line. Row 47: ASA III with the I50.9 reason; the exact High risk line; no flags; "Smoker, Heart failure unspecified"; 2 sets shown; one patient set complete. "Suggested ASA III" on the episode page matches |
| T-04 low-risk patient, sent on their behalf | Pass | "HQ completed on the patient's behalf". Row 48: ASA II "Smoker (77176002) – Current smoker"; Low risk fires; no error. Still Ready for review |
| T-05 a red flag is stored and stops Low risk | Not rerun | Passed in report 1. T-12's row also stores the red flag correctly |
| T-06 a clinician's set never counts; no second evaluation | Pass | The T-04 episode still has exactly one row (48) after Cara validated both sets and completed the review (R-04). The T-03 episode also has one row |
| T-07 publish through the API is refused while a test case fails | Pass | On V5, "Test the whole HQ" showed smoke Yes → "Suggested ASA II", Low risk. I saved it as "MT smoker retest". After Low risk was changed to "at most I", both cases failed. The Publish window listed "2 problems to fix first" with Publish disabled. curl gave `HTTP 422` with the exact error and both `{"set":"Rules","message":"Test case “…” fails"}` problems |
| T-08 the same version publishes once the case passes | Pass | Put back to II: "All 2 passed". curl gave `HTTP 204`; the API shows V5 `published` |
| T-09 service stopped: the patient's send still works | **Pass (F-1 fixed)** | Only the evaluate process was killed: health refused, while the web and API stayed up. The patient saw "Thank you, Kim". The episode went to Ready for review. Notes: "HQ completed by the patient \| Patient", then "The automatic checks didn't run. Review this HQ by hand. **\| Automatic** on 27/10/2026, 00:28". Database: that event has `automatic = t` and `user_id` NULL. Row 49: no version, empty results, error "evaluate: /evaluate: Post "http://127.0.0.1:8090/evaluate": dial tcp …: connection refused" (no answers in it). The API log has `EVALUATE_FAILED form 8d900838-…: …connection refused`, with no answers |
| T-10 service stopped: filling in for the patient still works | **Pass (F-1 fixed)** | "Sent for Ray…". Ready for review. "HQ completed on the patient's behalf \| Cara Clinician", then the line "**\| Automatic**", not "Patient". Row 50 is an error row. The log has `EVALUATE_FAILED form ef453fba-…` |
| T-11 service stopped: publishing is refused | Pass | I fixed V4's logic problem (I set "Nurse observations" to always show), then clicked Publish → Publish. The window showed "The checks couldn't run. Try again in a minute." and V4 stayed a draft. curl gave `HTTP 503` with the same message. Log: `EVALUATE_FAILED publish checks, version 2d034f8d-…` |
| T-12 start it again; everything works | Pass | I restarted it with `make evaluate` in the background, and health was ok. Clicking Publish in the same window published V4 (read-only, "Create new version" offered). A new send on V4 (on behalf; smoke No, chest pain Yes, heart failure Yes) stored row 51: ASA III, High risk, red flag, no error, and no "automatic checks" line |
| R-01 (S40) Rules page, Test the whole HQ, the auto-triage line | Pass | Rules page, editing a rule, test cases going PASS→FAIL→PASS, saving a case from Test the whole HQ. The Publish line read "This version auto-triages: ASA I–II, no red flags → Ready for admission…" and changed to "ASA I" while the rule said I |
| R-02 (S39) Suggested ASA on the episode page and POA Summary | Pass | T-03 episode: "Suggested ASA III: Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction" on both |
| R-03 (S34) a required video tick blocks sending | Pass | "Shots 2510 202004 2798", filled in by Cara. "Finish this section" without the tick showed "Please tick "I have watched this video" to go on." After ticking, it finished and sent. Row 52 was stored with no error |
| R-04 (S18–S22) validate every set and complete the review | Pass | T-04 episode: About you and Nurse checks validated. Complete HQ review moved it to **Ready for POA** |
| R-05 Publish window blocks on a logic problem | Pass | V4 (before my fix) listed "2 problems to fix first" (an answer that no longer exists; a question that comes later), with Publish disabled. This worked while the evaluate service was down, so the check runs in the browser |

## Failures in detail

None. F-1 from report 1 is fixed (T-09 and T-10 above).

## Found while exploring

- **Nothing about the stored result reaches the patient.** While the service was down, I fetched `GET /api/p/episodes/<T-09 id>` from the patient's session:
  - its keys are only `answers, archived, chapters, hospital, patient, prefill, questionnaireName, revisions, submitted`;
  - none of "suggestedAsa", "rules", "flags", "evaluate", "connection refused" or the "automatic checks" text appear in it.
- **Note order:** "HQ completed by the patient" and "The automatic checks didn't run…" have the **same `created_at`** to the microsecond. Both times they still showed in the right order, but nothing in the timestamp itself sets that order. This is only an observation, not a failure.
- **Logs:**
  - `make dev`'s own evaluate process exited with npm "code 143" when I stopped it. The API and web kept running.
  - The API log never contained an answer value. It only had the `EVALUATE_FAILED` lines with form or version ids.
- **Editor (not S41):** in "Fill in for the patient", the clinician-only "Nurse observations" text box on the Nurse checks set looks greyed out before you click into it, but typing works.

## Console and network errors not tied to a case

- **Console:** no errors or warnings on the staff and patient tabs I checked (the episode page, the patient form, Rules and Test the whole HQ).
- **Network:** the extension's request log again reports **503** for the R-03 `POST …/patient-form/submit`, which actually worked: the UI showed "Sent for Vic…", and row 52 plus Ready for review are in the database. This is the same tool reporting problem as in report 1. The only real 503s were the two T-11 publish attempts, which were expected.

## GIFs

All three are in `~/Downloads/`:

- `S41-retest-T03-happy-path.gif`: the patient's link, date of birth, answers, send and "Thank you".
- `S41-retest-T09-evaluate-down-automatic-note.gif`: the patient's send with the service stopped, then the General notes showing "| Automatic".
- `S41-retest-T11-publish-checks-couldnt-run.gif`: Publish refused with "The checks couldn't run…".

## State left behind

- **The evaluate service:** exactly one is listening on 127.0.0.1:8090. It's **the copy I started separately** (`make evaluate`, log `/tmp/mt-s41-evaluate.log`):
  - `make` is PID **36030**, `sh -c node build.mjs && node dist/evaluate.cjs` is PID **36102**, and `node dist/evaluate.cjs` is PID **36108**. The npm wrappers in between are PIDs 36040 and 36068.
  - `make dev`'s own evaluate process was stopped in T-09 and isn't running. The API and web from `make dev` are still up.
  - My copy isn't in `make dev`'s process group, so stopping `make dev` won't stop it. Kill 36030 (or 36108) before running `make dev` again.
- **Data:** "MT S41 2350" has V5 published and no draft. I didn't touch any trust's suspension state.

## Hand checks for Rahul

- None needed. I covered phone size with a headless Playwright screenshot at 390×844, because the Chrome window can't go below 500 px wide.
