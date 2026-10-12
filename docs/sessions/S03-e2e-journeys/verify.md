# S03 verification

**Verdict: Verified** (12 Oct 2026). Nothing blocks the commit. Six minor items are listed below. Each is a small fix in S03's own files, and none affects what the journeys prove today.

- **Commit:** `165c7b8`, with S03's changes uncommitted.
- **App:** `make dev` in Rahul's terminal, http://localhost:3000.
- **Read:** brief, build notes, test plan, test report 1 (16 passed, T-04 partly blocked), and F1 Steps 3–4.

## What I checked

- **Scope:** everything in the brief is done.
  - `e2e/` has pinned requirements, `conftest.py`, `journeys/` and a README.
  - Data is set up through the API with unique names, and elements are found by role and label.
  - J1–J5 are there, J3's patient runs on an iPhone 13, and J4 has the two-tab step.
  - `make e2e-setup` and `make e2e` exist, and traces are kept on failure.
  - The root README has a Tests section, and `manual-test-plan.md` has the Automated notes.
  - **App code is unchanged:** `git status apps packages` is empty, and no `data-testid` was added.
  - `spikes/ui-workflow.ts` was never in the repo (it's gitignored and isn't on disk), so there was nothing to delete. The build notes record this.
- **Tests first:** no new routes or logic, so no new integration or unit tests are owed. The journeys test existing behaviour; the build notes explain the TDD exception, and the broken-label check stands in for "watch it fail".
- **Tenant isolation:** no query changed.
- **UK hardcoding (A-20):**
  - The grep prints 3 lines: `en-GB` in `apps/web/src/lib/format.ts:1,6` and `apps/web/src/features/editor/OptionLists.tsx:71`.
  - They're older than S03 (`c8789d7`), and S05's brief already plans to replace them with the locale from `GET /api/config`.
  - S03's own files (`e2e/`) have none.
- **Security:**
  - There are no secrets in the repo.
  - The test data is made up (`Sam`, `1980-05-17`, hospital number `E2E`).
  - The patient JSON is still stripped (spot-check 2 below).
- **Docs:**
  - The plan Status, the plans README, the sessions README and `decisions.md` §1 and §4 are updated (I've now set them to Verified / Done).
  - The org-name grep prints nothing.
- **The one unexplained `make e2e` failure in 32 (build notes):**
  - I looked for a race in J3's leave-and-come-back step. There isn't one: `PatientHQ.tsx`'s `back()` waits for the save before closing the set, so the reload can't overtake it.
  - The code review also found no cause.
  - I didn't see a failure in my runs (5 runs, 18 journeys, all passed).
  - Nothing more to do unless it shows again; the trace will then say where.

## Findings (all minor, worst first)

1. **`E2E_BASE_URL` in `.env` would silently beat the command line.**
   - **Where:** `Makefile:2` (`-include .env` plus `export`) together with the new commented line in `.env.example:9`.
   - **Why it matters:** once someone uncomments that line, `E2E_BASE_URL=https://qa.example make e2e` (`e2e/README.md`, "Run") runs against localhost and passes, with no warning. Make's variables from `.env` override the environment. Today's `.env` doesn't set it, so nothing is wrong yet.
   - **Fix:** drop the line from `.env.example`, or say there to set it only on the command line.
2. **`make e2e-setup` hides its hint when `python3.12` isn't installed.**
   - **Where:** `Makefile:55`.
   - **Why it matters:** on a machine with only `python3.13` (or a CodeBuild image, which F4 will use), make prints `python3.12: No such file or directory`, never the "run: make e2e-setup PYTHON=…" hint written for exactly that case.
   - **Fix:** check `command -v $(PYTHON)` first.
3. **J3 checks for sideways scrolling only on the section list, not inside an open set.**
   - **Where:** `e2e/journeys/test_j3_patient.py:48`.
   - **What I tested:**
     - The code review said the check can pass even when the page overflows. That's wrong: I added a 900 px element to the body on an emulated iPhone 13, and it failed (`scrollWidth 900 > innerWidth 390`).
     - Inside an open set, both of About you's pages measured 390 = 390 and looked right in screenshots. So there's no bug today.
   - **Why it still matters:** a wide SurveyJS question is the likelier overflow, and J3 wouldn't catch it.
   - **Fix:** repeat the assert after `Start`.
4. **The `phone` fixture's teardown isn't in `try/finally`.**
   - **Where:** `e2e/conftest.py:86–88`.
   - **Why it matters:** if the private `_artifacts_recorder` changes on a pytest-playwright upgrade (the README warns it might), the WebKit browser is left running and the error hides J3's result.
5. **`-p no:cacheprovider` turns off `--lf`.**
   - **Where:** `e2e/pytest.ini:5`.
   - **Why it matters:** `make e2e ARGS=--lf` ("re-run the last failure") quietly runs everything. That's the natural next step after a rare failure like the one in the build notes.
   - **Fix:** remove the option and gitignore `.pytest_cache/` instead, or note it in the README.
6. **Cleanups, optional:**
   - J4 and J5 repeat the same five-line set-up (publish, episode, the patient answers, submit). It could be one `support` helper.
   - The "this option is checked" locator is written out 4 times in J3 and J4.
   - `Api.__init__` doesn't dispose its request context if sign-in fails (`support.py:63–64`). That only matters when the run is already failing.

**Not a finding:** the code review said the conftest `base_url` fixture makes pytest-playwright's `--base-url` option do nothing. That's true, but the plan's interface is `E2E_BASE_URL`, which the README documents.

**Still open from the test report (T-04):** nobody has watched the headed run. Both headed runs pass (`5 passed`), and a verifier session can't see the windows either. **Rahul:** watch `make e2e ARGS="--headed --slowmo 300"` once, when convenient. It doesn't block the commit.

## Test results

```
make lint              go vet -tags integration ./... and tsc --noEmit: no findings
make test              ok auth, chapter, episode, httpapi, lifebox; Vitest: Test Files 14 passed (14), Tests 111 passed (111)
make test-integration  ok apitest 10.0s, auth 1.6s, chapter 0.6s, episode 1.1s, httpapi 9.6s, lifebox 1.4s
make e2e (×3)          5 passed in 23.04s / 5 passed in 17.49s / 5 passed in 18.45s
```

- `make test-integration` first stopped at `db-up` because `docker` wasn't on this session's PATH. With `/Applications/Docker.app/Contents/Resources/bin` added, it passed. It's an environment issue, not S03's.
- **"Done when" 2, repeated:**
  - I relabelled the list's `Send my answers` in `PatientHQ.tsx:186` as `Submit`, ran J3, then restored the file (`git diff apps/web` is empty).
  - J3 failed at `expect(send).to_be_disabled()`, with "element(s) not found", the locator, and an aria snapshot of the patient's page.
  - It kept 3 traces (API, Chromium, WebKit).

## Spot-checks

| # | What | How | Result |
| --- | --- | --- | --- |
| 1 | J3's failure is readable when the patient's button changes | Broke the label (above) | Pass |
| 2 | Clinician content stripped for the patient | Chrome, `GET /api/p/{token}` on a fresh made-up episode | Pass: only About you and Your health; no "Nurse checks", `q_bp`, `clinicalOutputs`, `testCases` or `clinicianOnly` |
| 3 | Patient form at phone size, inside a set (J3 checks only the list) | WebKit, iPhone 13 profile, About you pages 1 and 2, plus screenshots | Pass: 390 = 390 on both pages, nothing cut off, full-width buttons |
| 4 | Another hospital's data | Chrome, as Bea Author (Hospital B) on a Hospital A episode | Pass: the API gives 404 "Hospital not found." under Hospital A's path and 403 under her own; the page says "You don't have access to this hospital."; no console errors |

Chrome was full-screen (2560 wide), so `resize_window` had no effect, as in the test report. That's why spot-check 3 used Playwright's WebKit iPhone profile, which is closer to a real phone anyway.

## Suggested commits

Docs go in their own commit.

1. **`End-to-end journeys J1–J5 with Playwright and pytest, and make e2e (F1 Step 3)`**
   - `e2e/` (all of it), `Makefile`, `.env.example`.
2. **`Docs: a Tests section, and which manual tests the journeys automate (F1 Step 4)`**
   - `README.md`, `CLAUDE.md`, `docs/manual-test-plan.md`, `docs/plan-workflow.md`.
3. **`Docs: S03 build notes, test plan, test report, verification and status; F1 done`**
   - `docs/sessions/S03-e2e-journeys/*`, `docs/sessions/README.md`, `docs/plans/README.md`, `docs/plans/f1-test-foundation.md`, `docs/decisions.md`, `docs/pending.md`.

`reports/` and `research_notes/` aren't S03's (decisions.md §5), so they stay out of these commits.
