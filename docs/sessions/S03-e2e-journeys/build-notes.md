# S03 build notes: end-to-end journeys J1–J5

**Plan:** F1 Steps 3 and 4. **Builder session:** 12 Oct 2026.

## Start

- **Previous session:** S02a is Verified and committed (`165c7b8`). The working tree held only the untracked `reports/` and `research_notes/`, which aren't S03's (decisions.md §5 lists them).
- **Questions for Rahul:** none in the brief. **Python:** `python3.12` (3.12.9) and `python3.13` are installed through Homebrew; plain `python3` is 3.10, so `make e2e-setup` uses `python3.12` by default (`PYTHON=` overrides it).

## What was built

### Step 3: the suite (`e2e/`)

| File | What it does |
| --- | --- |
| `requirements.txt` | Pinned: pytest 9.1.1, pytest-playwright 0.10.0, playwright 1.63.0. |
| `pytest.ini` | Runs `journeys/`, headless, `--tracing retain-on-failure`, artifacts in `e2e/test-results/` (gitignored). |
| `conftest.py` | Fixtures: `base_url` (`E2E_BASE_URL`, default `http://localhost:3000`); a session check that stops the run at once if `/api/health` doesn't answer; desktop viewport 1440×900; `as_user(name)` and `patient_link(episode)` API sessions; `phone`, an iPhone 13 in WebKit with its trace kept. |
| `support.py` | Data set-up through the real API (`Api`, `Patient`), `sign_in` (puts the dev-stub cookie in a browser context), and two UI helpers, `choose` (SurveyJS options) and `pick` (Mantine dropdowns). |
| `data/workflow-hq.json` | The HQ for J3–J5: About you (two pages; "Do you smoke?" with notes Smoker / Non-smoker), Your health, and Nurse checks (clinician). |
| `data/logic-problem-hq.json` | J2's draft: "How many a day?" shows only if `q_gone`, a question that doesn't exist, is Yes. |
| `journeys/test_j1…j5_*.py` | One journey each (table in `e2e/README.md`). |
| `README.md` | Set-up, running, reading a trace, test data, how to write a journey, the `data-testid`s used. |
| `.gitignore` | `.venv/`, `test-results/`, `__pycache__/`. |

**Makefile:** `make e2e-setup` (checks Python ≥ 3.12, makes `e2e/.venv`, installs the requirements, `playwright install chromium webkit`) and `make e2e` (`pytest` in `e2e/`; extra options through `ARGS`, e.g. `ARGS=--headed`). `.env.example` documents `E2E_BASE_URL`.

### Step 4: docs

- **Root `README.md`:** a Tests section with the three targets and what each covers; the layout lists `e2e/`; the stale "tests are unit-level" note is replaced.
- **`docs/manual-test-plan.md`:** an "Automated journeys" section, and an *Automated: J<n>* or *Partly automated: J<n>* note on MT-01, 02, 06, 08, 46, 51 and 64–68, each saying what the journey doesn't check.
- **`CLAUDE.md`:** the commands list has `make e2e-setup` and `make e2e`, and how to run one journey; the "planned targets" line is gone.
- **`docs/plan-workflow.md`:** marks `spikes/ui-workflow.ts` as retired.

## The Python, for Rahul

The suite is Python (the F1 plan's choice). Reading order:
1. **A journey file,** e.g. `journeys/test_j4_clinician_validates.py`. pytest runs every function whose name starts `test_`. Its parameters are **fixtures**: pytest builds each one before the test and tears it down after. `page` is a browser tab, from pytest-playwright; `as_user` and `patient_link` come from `conftest.py`.
2. **`expect(locator).to_…`** waits up to 15 seconds for the condition, then fails with a message. That's what makes the journeys wait for the app rather than sleep.
3. **`support.py`:** `Api` is like the Go `apitest` client: it signs in as a seeded user and calls the API (`call("POST", path, body)`). `draft_hq` creates a questionnaire from a `data/` file, Question Set by Question Set, the way the editor does.
4. **`conftest.py`:** the fixtures. `yield` in a fixture is its `defer`: the code after it runs when the test ends.

## Differences from the plan or brief

1. **`spikes/ui-workflow.ts` didn't exist to delete.** It was gitignored and never committed, and this machine doesn't have it. The journeys were written from the app and the plans instead. `plan-workflow.md` now says it's retired.
2. **No `data-testid` was added,** so the app is unchanged. Everything is found by role or label, except three ids that already existed (`set-<name>` on the patient's section cards, and the two save-status texts), listed in `e2e/README.md`.
3. **The patient runs in WebKit** (Safari's engine) with Playwright's iPhone 13 profile, not in Chromium at phone size. It's the closer match to a real iPhone, and `make e2e-setup` installs WebKit as the plan says. To keep the phone's trace on failure, the `phone` fixture registers its context with pytest-playwright's private `_artifacts_recorder` fixture; `e2e/README.md` warns to check it on upgrade.
4. **Sign-in:** J1 signs in through the stub sign-in page (it's part of the author's journey). The others sign in through `POST /api/dev/login`, which puts the cookie in the browser context, so the UI is used only for the journey.
5. **J3, J4 and J5 each set up their own data** through the API, so they never depend on each other: J4 sends the patient's answers through the patient-link API; J5 also validates every set and completes the review through the API.
6. **J3 checks a little more than the F1 table:** the status dropdown is disabled and "Episode created" is in the notes before the patient starts, and reopening the link after sending shows the thank-you page. That lets the manual-test-plan notes for MT-65 and MT-66 say "Automated".
7. **J2's logic problem** is a condition that tests a question that no longer exists (set up in the draft through the API), fixed in the Logic tab's builder. MT-46's problem (a question moved above the one it tests) needs a drag, which is slower and more fragile to automate.
8. **`make e2e` takes options through `ARGS`** (e.g. `make e2e ARGS="--headed --slowmo 300"`), and `make e2e-setup` takes `PYTHON`. Neither is in the plan.
9. **TDD:** the journeys test behaviour that already exists, so there was nothing to make fail first. Instead, J3 was checked against a broken label (below), as the brief asks. J1 and J4 also failed on an ambiguous selector while being written, and showed readable messages; J2, J3 and J5 passed first time.

## Gotchas found (for decisions.md §4)

- **SurveyJS hides the real radio input** behind a drawn one, so Playwright's `check()` on the radio waits forever ("intercepts pointer events"). Tap the option's text, as a person does (`support.choose`).
- **`make dev` adds a Next.js tools button** whose accessible name contains "Next", so `get_by_role("button", name="Next")` matches two buttons. Use `exact=True`. It goes with plan F2.
- **pytest-playwright 0.10 traces API request contexts too,** so each failed journey has one trace per browser plus one for the API set-up, named `trace-1.zip`… in no fixed order.
- **For F4's pipeline** (`make e2e-setup e2e` against QA): CodeBuild must have Python 3.12+ called `python3.12`, or pass `PYTHON=`. The journeys sign in with the dev stub, so QA needs it until S12 changes sign-in.

## Checks

**"Done when" 1: `make e2e` passes headless 3 runs in a row.** Run against `make dev` (Rahul's), 12 Oct:

```
============================== 5 passed in 19.12s ==============================
============================== 5 passed in 15.93s ==============================
============================== 5 passed in 16.25s ==============================
```

After J3 gained its extra checks, `make e2e` ran 29 more times: 28 passed (including 20 in a row), and **one failed, unexplained**. That run's output wasn't captured (only `make: *** [e2e] Error 1` was printed), and the 27 runs after it, the last 20 kept with logs and traces in case of failure, all passed. If it shows again, the trace in `e2e/test-results/` will say which journey and step. **Tester and verifier: please note any `make e2e` failure in full.**

**"Done when" 2: breaking the patient submit button's label makes J3 fail, with a readable trace.** In `PatientHQ.tsx`, the list's `Send my answers` button was relabelled `Submit` (then restored; `git diff apps/web` is empty):

```
E       AssertionError: Locator expected to be disabled
E       Error: element(s) not found
E         - Expect "to_be_disabled" get_by_role("button", name="Send my answers") with timeout 15000ms
E       Aria snapshot:
E       - paragraph: Lifebox
…
journeys/test_j3_patient.py: send = patient.get_by_role("button", name="Send my answers"); expect(send).to_be_disabled()
```

The output names the line and the missing button, with the page's aria snapshot. `e2e/test-results/<J3>/` kept three traces: the API set-up, the clinician's Chromium, and the patient's WebKit (phone-sized screenshots, the failing step in red).

**`make lint test test-integration`** (exit 0):

```
make lint          go vet -tags integration ./... (no findings); tsc --noEmit (no errors)
make test          ok auth, chapter, episode, httpapi, lifebox;  Vitest: Test Files 14 passed (14), Tests 111 passed (111)
make test-integration
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	1.990s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	0.661s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	1.115s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	0.890s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	5.546s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	1.342s
```

## Test report 1 (12 Oct): 16 passed, 0 failed, T-04 partly blocked

No failures, so no app code changed. Fixed from the report's minor items, all in S03's own files:

| Item | Change |
| --- | --- |
| Minor 1: the README's trace count is wrong | `e2e/README.md` now says one trace per browser plus one per API session (J4 and J5 have three), and shows a real folder name. |
| Minor 2: pytest noise around "app not running" | The check moved from a session fixture to a `pytest_sessionstart` hook that raises `pytest.UsageError`, so the run prints one line: `ERROR: The app isn't answering at …`. Seen with `E2E_BASE_URL=http://localhost:3999 make e2e`. |
| Cosmetic 1: J3 never asserts the clinician's set is hidden | J3 now checks "Nurse checks" never appears on the patient's page, plus "Hello Sam." and no sideways scrolling at iPhone 13 size. MT-66's note is reworded. |
| Test plan: T-09's steps can't be done in order | Step 2 now moves the question below "Do you smoke?" first; noted at the top of the test plan. |

**Not changed:**
- **T-04 (watching the headed run):** the tester couldn't see the windows. The headed and slow runs passed (`5 passed`). Rahul to watch `make e2e ARGS="--headed --slowmo 300"` once; the verifier can too.
- **"Found while exploring" items 1–5** are older than S03 and outside its scope (the brief allows no app changes). They're in `docs/pending.md` §3. Item 6 ("Saved" before any change) was already there.
- **"Date of birth is filled as 1980-05-17":** that's how Playwright fills a date box in every browser; a person's typing gives the same value.

After the changes, `make e2e` passed three more runs in a row (`5 passed` in 21.2 s, 18.3 s, 19.0 s).

## Suggested commits

1. **`End-to-end journeys J1–J5 with Playwright and pytest, and make e2e (F1 Step 3)`**
   - `e2e/` (all of it), `Makefile`, `.env.example`.
2. **`Docs: a Tests section, and which manual tests the journeys automate (F1 Step 4)`**
   - `README.md`, `CLAUDE.md`, `docs/manual-test-plan.md`, `docs/plan-workflow.md`.
3. **`Docs: S03 build notes, test plan, test reports, verification and status`**
   - `docs/sessions/S03-e2e-journeys/*`, `docs/sessions/README.md`, `docs/plans/README.md`, `docs/plans/f1-test-foundation.md`, `docs/decisions.md`, `docs/pending.md`.
