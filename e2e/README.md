# End-to-end journeys

User journeys through the real app in real browsers, with Playwright and pytest (plan [F1](../docs/plans/f1-test-foundation.md), Step 3). Each journey follows one person's task from start to finish; it isn't a test of one screen.

| ID | File | Journey | Browser |
| --- | --- | --- | --- |
| J1 | `journeys/test_j1_author.py` | **Author:** signs in, creates a questionnaire, adds a Question Set, a page and a question, reloads and finds it kept. The preview shows the question. | Desktop Chromium |
| J2 | `journeys/test_j2_publish.py` | **Publish:** publishing is blocked by a logic problem. The author follows the header's problem list to the condition, fixes it, publishes, then clicks **Create new version**. | Desktop Chromium |
| J3 | `journeys/test_j3_patient.py` | **Patient:** the clinician creates an episode and copies the link. The patient opens it on a phone, fills in every Question Set (leaving part-way and coming back), and sends it. The episode is Ready for review. | Clinician: desktop Chromium. Patient: **iPhone 13** (WebKit, Safari's engine) |
| J4 | `journeys/test_j4_clinician_validates.py` | **Clinician validates:** corrects one answer ("Patient answered: Yes" appears), validates every Question Set and completes the review. **Two tabs:** a stale tab's Validate shows "changed somewhere else" with **Reload**, and Reload shows the newer answer. | Desktop Chromium |
| J5 | `journeys/test_j5_poa_summary.py` | **POA Summary:** both tabs, with the corrected answer marked on the validated one. The status changes and General notes records it; a comment is added; **Print** is there. | Desktop Chromium |

## Set up (once)

You need Python 3.12 or newer. On a Mac with Homebrew: `brew install python@3.12`.

```sh
make e2e-setup
```

This makes a virtual environment in `e2e/.venv`, installs the pinned packages in `requirements.txt`, and downloads Chromium and WebKit for Playwright. If your Python 3.12+ isn't called `python3.12`, name it: `make e2e-setup PYTHON=/path/to/python3.13`.

## Run

The journeys run against the app you already have running. In another terminal, start it as usual and leave it running:

```sh
docker compose up -d db
make migrate seed
make dev
```

Then:

| To | Type |
| --- | --- |
| Run every journey, headless | `make e2e` |
| Watch them in visible browsers | `make e2e ARGS=--headed` |
| Watch slowly (300 ms per action) | `make e2e ARGS="--headed --slowmo 300"` |
| Run one journey | `make e2e ARGS=journeys/test_j3_patient.py` |
| Run against another address | `E2E_BASE_URL=https://qa.example make e2e` |

A run takes about 20 seconds. Each test prints a dot as it passes, and the last line says how many passed. If the app isn't running, the run stops at once with one line: "ERROR: The app isn't answering at …".

## When a journey fails

pytest prints:
- the journey's line that failed, such as `expect(send).to_be_disabled()`;
- what it was waiting for, and for how long (15 seconds at most);
- an **aria snapshot**: the page as a screen reader sees it, at the moment it failed.

That's often enough to see what changed, such as a button with a new label.

### Reading a trace

A failed journey keeps its Playwright **traces** in `e2e/test-results/<journey name>/`. A trace is a recording of the run: every action, with screenshots and the page's HTML before and after, plus the console and network requests. Passing journeys keep nothing, and each run clears the folder first.

There's one trace per browser, plus one per API session that set up data: one for each user (Alex creates the questionnaire, Cara the episode) and one for the patient's link. So J4 and J5 each have three API traces and one browser trace. J1 has a single `trace.zip`; the others have `trace-1.zip`, `trace-2.zip` and so on, in no fixed order. API traces have no screenshots, and in J3 the patient's iPhone trace is the one with phone-shaped screenshots.

The folder is named after the whole test, e.g. `e2e/test-results/journeys-test-j4-clinician-validates-py-test-clinician-validates-every-set-and-completes-the-review-chromium/`.

Open one with:

```sh
e2e/.venv/bin/playwright show-trace e2e/test-results/<journey name>/trace-2.zip
```

In the viewer:
1. **Actions,** on the left, lists every step. The failed one is red; click it.
2. **Before** and **After,** at the top, show the page at that step. You can inspect them like a live page.
3. **Call** shows what the step waited for. **Console** and **Network** show the browser's errors and the API requests, with their responses.

## Test data

- **Set up through the API** (`support.py`), signed in as the seeded users with the dev stub, so the browser is used only for the journey itself. J4 and J5 set up the patient's answers this way; J3 is where a patient fills them in.
- **Every name has a unique suffix,** such as `E2E J3 1210 150457 c0d4` (day, month, time, random). Runs never collide, and journeys don't depend on each other or on their order.
- **Nothing is deleted.** Published questionnaires can't be deleted, so `E2E …` questionnaires and `Sam J…` patients collect in Hospital A. The local database is disposable: rebuild it when it gets cluttered ([decisions.md](../docs/decisions.md) §4).
- **Made-up data only.** The questionnaires are in `data/`.

## Writing a journey

- **Find things as a person would:** by role and name (`get_by_role("button", name="Validate")`) or by label (`get_by_label("First name")`). This checks the page is accessible too.
- **`data-testid` only where an element has no accessible name.** List each one below.
- **Answer SurveyJS questions with `choose(page, "Do you smoke?", "Yes")`.** SurveyJS hides the real radio button behind a drawing of one, so the helper taps the option's text, as a person does, then checks the radio is selected.
- **Pick from a dropdown (Mantine Select) with `pick(page, "Sex", "Female")`.**
- **Use `exact=True` for a "Next" button.** `make dev` adds a Next.js tools button whose name contains "Next". (It goes when plan F2 replaces Next.js.)
- **Patient journeys use the `phone` fixture,** an iPhone 13 in WebKit. Its trace is kept through pytest-playwright's private `_artifacts_recorder` fixture, so check `conftest.py` still works after upgrading pytest-playwright.
- **Wait for saves through the page,** never with a fixed sleep: `expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")`, or `page.expect_response(...)` around the change (as J4 does).

### `data-testid` attributes used

None were added for these journeys. These already existed:

| Test id | Where | Why |
| --- | --- | --- |
| `set-<Question Set name>` | The patient's section cards (`PatientHQ.tsx`) | A card has no accessible name, and every card has a **Start** button. |
| `save-status` | The editor's header: "All changes saved" (`QuestionnaireEditor.tsx`) | Plain text, with no role. |
| `validate-save-status` | The Validate page's "Saved" (`ValidateSet.tsx`) | Plain text, with no role. |

## Files

```
e2e/
  requirements.txt   pinned pytest, pytest-playwright, playwright
  pytest.ini         headless; traces kept on failure, in test-results/
  conftest.py        fixtures: base_url, as_user, patient_link, phone
  support.py         API set-up (Api, Patient) and the choose / pick helpers
  data/              made-up questionnaires the journeys set up
  journeys/          one file per journey
```
