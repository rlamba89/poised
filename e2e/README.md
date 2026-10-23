# End-to-end journeys

User journeys through the real app in real browsers, with Playwright and pytest (plan [F1](../docs/plans/f1-test-foundation.md), Step 3). Each journey follows one person's task from start to finish; it isn't a test of one screen.

| ID | File | Journey | Browser |
| --- | --- | --- | --- |
| J1 | `journeys/test_j1_author.py` | **Author:** signs in, creates a questionnaire, adds a Question Set, a page and a question, reloads and finds it kept. The preview shows the question. | Desktop Chromium |
| J2 | `journeys/test_j2_publish.py` | **Publish:** publishing is blocked by a logic problem. The author follows the header's problem list to the condition, fixes it, publishes, then clicks **Create new version**. | Desktop Chromium |
| J3 | `journeys/test_j3_patient.py` | **Patient:** the clinician creates an episode, which texts the patient their invite (captured, S14). The patient opens the link on a phone, taps Continue, confirms their date of birth, fills in every Question Set (leaving part-way and coming back), and sends it. The episode is Ready for review. | Clinician: desktop Chromium. Patient: **iPhone 13** (WebKit, Safari's engine) |
| J4 | `journeys/test_j4_clinician_validates.py` | **Clinician validates:** corrects one answer ("Patient answered: Yes" appears), validates every Question Set and completes the review. **Two tabs:** a stale tab's Validate shows "changed somewhere else" with **Reload**, and Reload shows the newer answer. | Desktop Chromium |
| J5 | `journeys/test_j5_poa_summary.py` | **POA Summary:** both tabs, with the corrected answer marked on the validated one. The status changes and General notes records it; a comment is added; **Print** is there. | Desktop Chromium |
| J6 | `journeys/test_j6_onboarding.py` | **Onboarding a trust (S10):** the platform admin creates a trust and a hospital and invites its first admin; that admin, in their own browser, invites a clinician, who signs in. Also: suspending a trust refuses its admin's sign-in until it's re-activated; only platform admins reach `/admin`; a duplicate code is refused; a trust admin gives one hospital its own phone number. | Desktop Chromium |
| J7 | `journeys/test_j7_patient_invites.py` | **Patient invites (S14):** J7: a clinician creates a patient and episode; the captured-messages page shows the text and the email; the patient opens the text's link (nothing about them shows before their date of birth), gets the date wrong once, then right, and fills in and sends the HQ; the link still opens later. J7b: five wrong dates of birth lock the link; the clinician sees the lock, resends, and only the new link works. **On the patient's behalf** (PAT-09): a clinician fills in and sends the HQ; "entered by" shows on the episode and the POA Summary. **Leaving straight after typing:** Sign out (patient) and the on-behalf form's ← Episode link keep the text and choice just entered (`data/medicines-hq.json`). **Session ended:** after a Resend, the patient's next change shows "Your latest change in About you wasn't saved…". | Clinician: desktop Chromium. Patient: **iPhone 13** (WebKit) |
| J7c | `journeys/test_j7c_returning_patient.py` | **A returning patient (S15):** Sam has Trust A's questionnaire (their first visit made their account) and a new one from Trust B to the same email; they ask for a sign-in link by email at `/signin`, open it, Continue, date of birth, and their home lists both trusts' questionnaires, newest first; they open one and come back with **Your questionnaires**; another episode's address shows "Episode not found."; Sign out; the used link says it has expired. **Changing their email** (PAT-10): Change on the home sends a link to the new address; Confirm there changes the home and the hospital's record, and the old address is told. **Twins** (S15 answers 6): two accounts on one email with one date of birth; the sign-in link asks for the first name and surname, and the home shows only that twin's questionnaire, "For Lee". | Patient: **iPhone 13** (WebKit) |
| J8 | `journeys/test_j8_reminders.py` | **Reminders and "not responding" (S16):** a new episode's invite; on the scheduled-jobs page (`/qa/jobs`, capture mode only) the jobs run as of now + 2 days send a reminder by text and email with a new link (also on the captured-messages page), and running them again sends nothing; the episode's General notes say so, written "Automatic"; the patient opens the **reminder's** link and starts answering, so now + 5 days sends nothing; now + 7 days marks the episode **Not responding** (no message), until the patient sends their answers. The jobs page runs every due job, so other episodes may get reminders too; only this patient's messages are checked. | Clinician: desktop Chromium. Patient: **iPhone 13** (WebKit) |
| J9 | `journeys/test_j9_lifecycle.py` | **The full lifecycle (S22):** an episode with a procedure; the patient sends the HQ; Cara completes the review and moves it to Ready for admission, Admitted and Discharged, with the dates. Cara can't archive; **Sue Super archives it** with a reason (Enter sends): it's read-only (status disabled, no Add comment, no Add task), it leaves the episode list until **Show archived** is ticked (kept across a reload), and Cara can't un-archive. Sue **un-archives** it ("Say why…" first): back to Discharged exactly. The jobs run as of 31 days on: the **daily auto-archive** archives it, and the history says "Automatic (Auto-archive: 30 days after discharge)". (J9b, cancel, is `test_j9b_cancel.py`.) The run as of 31 days runs every due job, so other episodes may get reminders or be archived too. | Desktop Chromium |
| Close | `journeys/test_close_account.py` | **A patient closes their account (S22, EPI-09):** from their home, "Close my account" asks first and warns that the unsent questionnaire won't be sent ("Keep my account" backs out); confirmed, "Your account is closed", a reload finds them signed out, and the old invite link no longer works. Cara sees "The patient closed their account on …" in the General notes; her new invite starts a fresh account. | Clinician: desktop Chromium. Patient: **iPhone 13** (WebKit) |
| Tasks | `journeys/test_tasks_worklists.py` | **Tasks, worklists and status messages (S21):** Cara adds a task for the anaesthetist team (Enter sends; "Choose the team." first); on the clinician home the episode is in the **Open tasks** and **Anaesthetic review needed** tabs (not On hold), the tab's count is its list's length, and the tab survives Back and reload; she completes the task ("Completed by Cara Clinician") and the episode leaves Open tasks; re-opened, it comes back. **A status message:** with Hospital A1's Ready for admission email switched on (by the API, switched back off after), moving a reviewed episode there sends the captured email in the hospital's wording. | Desktop Chromium |
| Question keys | `journeys/test_question_keys.py` | **Question keys (S23):** in a draft sharing keys with an earlier published HQ, "Do you smoke?" says **Pre-fills from N other HQs** and lists the earlier HQ "by link"; a new question says **New question: won't pre-fill**; **Always ask fresh** replaces the label with its badge and survives a reload; rewording a published option is refused in the Publish window, naming the question and the option; put back, it publishes. | Desktop Chromium |
| Library | `journeys/test_question_library.py` | **The question library and "Same as…" (S24):** **From library** inserts the platform's "Do you smoke?" keeping its key `smoker`, and a second insert into the same HQ is refused; **Save to library** puts a new question in the trust's library with a tag (Trust B's author never sees it), and **Remove** takes it out (the HQ keeps its question); **Same as…** gives a new Select One question the key `pregnant` (the seeded Demo pre-op HQ asks it) and its options, keeping its own text, and its label then says **Pre-fills from N other HQs**, "by link"; a text question's choice candidate is greyed with its reason; deleting an adopted option is refused at Publish (S23's lock). Needs the seed's S24 Demo pre-op HQ. | Desktop Chromium |
| Patients | `journeys/test_patient_records.py` | **Patient records (S13):** on the New episode page, a clinician at Hospital A1 types the details of a patient known only at A2 (A1's search can't find them) and the episode links silently, showing nothing of A2; a wrong NHS number and a mismatched date of birth are refused without the stored details; a possible duplicate known elsewhere names nobody (Create anyway), and one at this hospital offers Use this record; the search by surname, NHS number, hospital number and short surname + date of birth, and a second open episode. | Desktop Chromium |
| Cognito | `journeys/test_cognito_signin.py` | **Staff sign-in on QA (S12), QA only:** a clinician signs in with a password and an authenticator code and signs out (out of the managed login too); a wrong password and a wrong code are refused; a suspended trust's staff are refused at sign-in; there's no dev login and the testers' tools need a platform admin; the web app's security headers. | Desktop Chromium |
| Trusts | `journeys/test_trusts.py` | **Trusts and hospitals (S08):** a member of two trusts picks a hospital and switches trust from the header, with their role in each; a trust admin's **All hospitals** list; a clinician at Hospital A1 can't open Hospital A2's episode by its URL; a super clinician of one hospital can view but not change the trust's questionnaires, in the editor or the Preview; the header fits a 390px window. | Desktop Chromium (one case at 390px) |

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
| Run against QA (Cognito sign-in, S12) | `make e2e-users PROFILE=<profile>` once, then `make e2e E2E_BASE_URL=<QA's address>` |

A run takes about 20 seconds. Each test prints a dot as it passes, and the last line says how many passed. If the app isn't running, the run stops at once with one line: "ERROR: The app isn't answering at …".

## How the journeys sign in

`support.py` asks the app how staff sign in (`GET /api/config`, S12):

- **Locally** (`make dev`), with the dev stub, as before.
- **On QA**, through Cognito's managed login in a browser: the seeded user's email and password, then a code from their authenticator secret (`pyotp`). The sign-ins come from `e2e/.qa-users.json` (gitignored; `make e2e-users`, or the pipeline's E2E step), or the file named by `E2E_TEST_USERS`. Each user signs in once per run and the session cookie is reused, because Cognito takes each authenticator code only once.
- **The testers' tools** (`captured()`, `/qa/messages`, `/qa/jobs`) are used as Sam Superadmin (`qa_request()`, `qa_page(page)`), since on QA they need a platform admin.
- **Marks:** `dev_login` journeys (`test_staff.py`, `test_j6_onboarding.py`: they sign in as people they've just invited, whose temporary password is in Cognito's email) are skipped on QA; `cognito_only` journeys (`test_cognito_signin.py`: password + code, wrong ones refused, a suspended trust refused, sign-out, no dev login, the tools need a platform admin, the security headers) run only there.

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
- **Nothing is deleted.** Published questionnaires can't be deleted, so `E2E …` questionnaires and `Sam J…` patients collect in Trust A (Hospital A1). The local database is disposable: rebuild it when it gets cluttered ([decisions.md](../docs/decisions.md) §4).
- **Made-up data only.** The questionnaires are in `data/`.

## Writing a journey

- **Find things as a person would:** by role and name (`get_by_role("button", name="Validate")`) or by label (`get_by_label("First name")`). This checks the page is accessible too.
- **`data-testid` only where an element has no accessible name.** List each one below.
- **Answer SurveyJS questions with `choose(page, "Do you smoke?", "Yes")`.** SurveyJS hides the real radio button behind a drawing of one, so the helper taps the option's text, as a person does, then checks the radio is selected.
- **Pick from a dropdown (Mantine Select) with `pick(page, "Sex", "Female")`.**
- **`exact=True` for a "Next" button** was needed while `make dev` added a Next.js tools button whose name contained "Next". Plan F2 removed Next.js (S04, 12 Oct); the journeys keep `exact=True`, which is harmless.
- **Patient journeys use the `phone` fixture,** an iPhone 13 in WebKit. Its trace is kept through pytest-playwright's private `_artifacts_recorder` fixture, so check `conftest.py` still works after upgrading pytest-playwright.
- **Wait for saves through the page,** never with a fixed sleep: `expect(page.get_by_test_id("save-status")).to_have_text("All changes saved")`, or `page.expect_response(...)` around the change (as J4 does).

### `data-testid` attributes used

None were added for these journeys. These already existed:

| Test id | Where | Why |
| --- | --- | --- |
| `set-<Question Set name>` | The patient's section cards (`PatientHQ.tsx`) | A card has no accessible name, and every card has a **Start** button. |
| `save-status` | The editor's header: "All changes saved" (`QuestionnaireEditor.tsx`) | Plain text, with no role. |
| `validate-save-status` | The Validate page's "Saved" (`ValidateSet.tsx`) | Plain text, with no role. |
| `invite` | The episode's **Patient's invite** card (`EpisodePage.tsx`, S14) | A card has no accessible name. |
| `dob-error` | The wrong-date-of-birth message on the patient's link (`InviteLink.tsx`, S14) | An alert with no name; J7 reads the tries left. |
| `captured-message` | One message on the captured-messages page (`CapturedMessages.tsx`, S14) | A card has no accessible name. |
| `on-behalf-banner` | "You're filling in …'s health questionnaire for them" (`PatientHQ.tsx`, S14) | An alert with no name. |
| `hq-set-<Question Set name>` | A row of the episode's Health questionnaire table (`EpisodePage.tsx`) | Already there; the on-behalf journey reads its "entered by". |
| `jobs-run-result` | The scheduled-jobs page's "Ran 1 job as of …" (`ScheduledJobs.tsx`, S16) | An alert with no name; J8 waits for it after each run. |
| `not-responding` | The episode's **Not responding** badge (`EpisodePage.tsx`, S16) | A badge has no role. |

## Files

```
e2e/
  requirements.txt   pinned pytest, pytest-playwright, playwright
  pytest.ini         headless; traces kept on failure, in test-results/
  conftest.py        fixtures: base_url, as_user, patient_link, phone
  support.py         API set-up (Api, Patient), the captured invites (invite_path), and the choose / pick helpers
  data/              made-up questionnaires the journeys set up
  journeys/          one file per journey
```
