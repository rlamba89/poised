# S03 test plan: end-to-end journeys J1–J5

**Changed after report 1 (12 Oct):** T-09 step 2 now says to move "What is your job?" below "Do you smoke?" first. T-05's "app not running" message is now a single `ERROR:` line.

**What S03 built:** an automated end-to-end suite in `e2e/` (Playwright + pytest) with five user journeys, J1–J5, and two `make` targets: `make e2e-setup` and `make e2e`. The app itself didn't change.

So this round has two halves:
1. **Part A: use the suite as a newcomer would,** following `e2e/README.md` literally. Report any step that's unclear, wrong or surprising. **This plan says you may use a terminal** for Part A: run exactly the commands given, from the repository root (`/Users/rahul/poised`).
2. **Part B: do J1–J5 by hand in the browser,** and report anything a real user does or sees that the automation doesn't.

## Setup

**Rahul, in your own terminal, before testing** (leave `make dev` running):

```sh
docker compose up -d db && make migrate seed && make dev
```

- **URL:** http://localhost:3000
- **Sign-in:** the stub sign-in page lists the seeded users; click a name. There's no password.

  | User | Hospital | Role |
  | --- | --- | --- |
  | Alex Author | Hospital A | author and publisher |
  | Val Viewer | Hospital A | viewer (read-only) |
  | Cara Clinician | Hospital A | clinician (episodes) |
  | Bea Author | Hospital B | author |

- **Name your own data** `MT S03 <time> …`, e.g. a questionnaire `MT S03 1432 workflow` and a patient `Sam MT1432`. Made-up details only.
- **Data the suite leaves behind:** each `make e2e` run adds questionnaires named `E2E J1 …`, `E2E J2 …` and so on, and patients named `Sam J3 …`, `Sam J4 …`, to Hospital A. That's expected (published questionnaires can't be deleted).
- **Expected console noise:** two 401s from `/api/me` on the sign-in page, and a 409 in T-12 and R-03, where the app refuses a stale save on purpose.

## Part A: the suite, following `e2e/README.md`

Open `e2e/README.md` and follow it as written. Where a step below says "per the README", type what the README tells you, not what this plan guesses.

**T-01: Set-up instructions**
1. Read the README's "Set up (once)" section.
2. Check your Python: `python3.12 --version`.
3. Run the set-up command the README gives (`make e2e-setup`).
4. Expected:
   - It finishes with no error. It may take a minute the first time (it downloads browsers); later runs are quicker.
   - `e2e/.venv/` exists.
   - Report any step where you had to guess.

**T-02: Set-up refuses an old Python, clearly**
1. Run `make e2e-setup PYTHON=python3` (on this Mac, `python3` is 3.10).
2. Expected: it stops at once with "e2e needs Python 3.12+; run: make e2e-setup PYTHON=<path to python3.12+>", and changes nothing.

**T-03: Headless run**
1. With `make dev` running, run every journey headless, per the README.
2. Expected:
   - It finishes in about 20 seconds, with no browser windows.
   - The last line says `5 passed`.
   - Note the exact command and the time it took.

**T-04: Headed run, watching**
1. Run the journeys in visible browsers, per the README (`make e2e ARGS=--headed`).
2. Expected:
   - Browser windows open and close as the journeys run: a desktop-sized Chromium window for staff, and during J3 a **phone-sized** WebKit (Safari engine) window for the patient.
   - `5 passed`.
3. Then run the slow, visible version from the README's table, and watch J3 and J4 in particular. Report anything a real user wouldn't do (for example, typing into a box that isn't visible).
4. If you can't see the windows from your session, say so in the report, and ask Rahul to watch the slow run and tell you what he saw.

**T-05: One journey, and the app not running**
1. Run only J3, per the README's table. Expected: `1 passed`.
2. Run `E2E_BASE_URL=http://localhost:3999 make e2e` (nothing listens there).
3. Expected: the run stops almost at once with one line, "ERROR: The app isn't answering at http://localhost:3999/api/health. Start it first: `make dev`." No journey runs, and there's no long wait.

**T-06: Reading a trace**
1. Make a passing journey keep its trace: `make e2e ARGS="journeys/test_j4_clinician_validates.py --tracing on"`.
2. Find the trace files in `e2e/test-results/`, as the README's "Reading a trace" section describes.
3. Open the Chromium one with the README's `show-trace` command. (A desktop window opens; if you can't see it, ask Rahul to.)
4. Expected:
   - The Actions list shows the journey's steps, such as clicking **Validate** and **Reload**.
   - Clicking a step shows the page before and after it.
   - The Network tab shows the `PUT …/answers/…` requests, one with a **409**.
   - Report anything in the README's description that doesn't match what you see.
5. Run `make e2e` once more afterwards. Expected: `e2e/test-results/` is cleared (passing journeys keep nothing).

**T-07: The rest of the docs**
1. Read the root `README.md` "Tests" section, and `docs/manual-test-plan.md`'s new "Automated journeys" section and its *Automated: J<n>* notes (on MT-01, 02, 06, 08, 46, 51 and 64–68).
2. Expected: the three `make` targets and what each covers are clear, the links work, and each note matches what you saw the journey do in T-04.

## Part B: J1–J5 by hand

Do each journey yourself in the browser, the way a real user would. Afterwards, compare with the journey's file in `e2e/journeys/` (its comments say what it does), and note any difference: a step the automation skips, does differently, or does in a way no user could.

**T-08: J1, Author (desktop)**
1. Open http://localhost:3000. You land on the sign-in page; click **Alex Author**. Expected: the Questionnaires list.
2. **New Questionnaire**, name `MT S03 <time> J1`, any description, **Save**. Expected: the editor, with "v1 · draft" in the header.
3. **Add Question Set**, then **Add Page**, then under "Add content" click **Text**.
4. In the Settings panel on the left, set **Question text** to "What is your job?", then click out of the box. Wait for "All changes saved".
5. Reload the page. Expected: the same page opens again, and the card shows "What is your job?".
6. Click **Preview**. Expected: the Patient view shows a "What is your job?" box.
7. Press the browser's **Back** button. Expected: back in the editor on the same page.

**T-09: J2, Publish blocked by a logic problem (desktop)**
1. In your J1 questionnaire, add a second question (**Yes / No**) on the same page, with the text "Do you smoke?".
2. Move "What is your job?" **below** "Do you smoke?" first (card **Move down**), because a condition can only test earlier questions. Then open it (click its card), go to **Logic**, choose **Conditionally**, and make it show when "Do you smoke?" is Yes. Wait for "All changes saved".
3. Use the card's **Move down** / **Move up** buttons (or drag) to put "What is your job?" **above** "Do you smoke?". Expected: an orange "1 logic problem" button in the header.
4. Click **Publish**. Expected: the dialog lists the problem, and its **Publish** button is disabled. **Cancel**.
5. Click "1 logic problem", then the problem. Expected: that question's Logic tab opens.
6. Fix it: move the question back below "Do you smoke?". Expected: the logic-problem button goes.
7. **Publish** → **Publish**. Expected: "v1 · published" and "Read only".
8. **Create new version**. Expected: "v2 · draft", and the Question Set and its questions are still there.
9. (J2 does the same with a different problem: a condition testing a question that no longer exists, fixed in the Logic tab. Both are fine; report only if the UI behaves differently.)

**T-10: J3, Patient on a phone**
1. As **Alex Author**, make and publish a small questionnaire `MT S03 <time> workflow`:
   - Question Set **About you** (Patient), with two pages: page 1 a **Yes / No** "Do you smoke?", page 2 a **Yes / No** "Do you drink alcohol?".
   - Question Set **Your health** (Patient) with a **Yes / No** "Do you have diabetes?".
   - Question Set **Nurse checks**, set to **Clinician** on the Question Set screen, with a **Text** question "Blood pressure".
   - **Publish** it.
2. Sign out, and sign in as **Cara Clinician**. **New episode** → **New patient**: First name `Sam`, Last name `MT<time>`, a date of birth, Sex, your workflow HQ **v1**, Procedure "Knee replacement". **Create episode**.
3. Expected: the episode page shows "HQ not complete", a disabled status dropdown, "Episode created" in General notes, and a **Patient link** with **Copy**. Click **Copy**: the button says "Copied".
4. Resize the browser to **390×844**, and open the link in a new tab (paste it; the patient isn't signed in, so a private window is best if you can).
5. Expected: "Hello Sam", two sections (**About you**, **Your health**; never Nurse checks), "0 of 2 sections complete", and **Send my answers** disabled.
6. **Start** About you, answer "Do you smoke?", then tap **All sections**. Expected: About you shows "In progress".
7. Reload the page. Tap **Continue**. Expected: your answer is still selected. Tap **Next**, answer page 2, tap **Finish this section**. Expected: "Complete".
8. Do **Your health** the same way. Expected: "2 of 2 sections complete", and **Send my answers** is enabled.
9. **Send my answers**, then confirm. Expected: "Thank you, Sam". Reload: still the thank-you page.
10. Back in Cara's tab (desktop size), reload the episode. Expected: "Ready for review", and the Patient link has gone.
11. Throughout, check the phone layout: nothing cut off, no sideways scrolling, buttons big enough to tap.

**T-11: J4, Clinician validates (desktop)**
1. On T-10's episode, as Cara: **Complete HQ review** is disabled.
2. **Validate** About you. Change "Do you smoke?" to the other answer. Expected: "Patient answered: …" appears under it, and the status near **Validate** goes "Saving…" then "Saved".
3. **Validate**. Expected: back on the episode, About you shows "Validated by Cara Clinician on …".
4. **Validate** Your health without changing anything.
5. Open **Nurse checks** and click **Validate** with Blood pressure empty. Expected: "Some required questions have no answer yet." (new questions are required by default; if you switched Required off, it validates instead). Fill it in and **Validate**.
6. **Complete HQ review**. Expected: "Ready for POA", "HQ review completed by Cara Clinician" in General notes, and each set's button now says **View**.

**T-12: J4's two-tab step (desktop)**
1. Set up a fresh episode as in T-10 steps 2–9 (a new patient `Sam MT<time>b`, same workflow HQ), so it's Ready for review.
2. As Cara, open **Validate** for About you in **two tabs** (open it, then copy the URL into a second tab).
3. In tab 1, change "Do you smoke?" and wait for "Saved".
4. In tab 2 (which still shows the old answer), click **Validate**.
5. Expected: a red **Not saved** box: "These answers were changed somewhere else since you opened them. Reload to see the latest; your last change was not saved.", with a **Reload** button, and **Validate** disabled.
6. Click **Reload**. Expected: tab 2 now shows tab 1's answer, with "Patient answered: …" under it.
7. **Validate** in tab 2. Expected: About you is "Validated by Cara Clinician".

**T-13: J5, POA Summary (desktop)**
1. On T-11's episode, click **POA Summary**.
2. **Validated summary** tab. Expected: the patient details, General notes, and under About you the page summary with **Changed by clinician** where a disclosure note is shown, and an orange box "Changed by the clinician on …" listing "Do you smoke? Patient answered “…”, now “…”".
   - Your T-10 questions have no disclosure notes, so you may see only the orange box. That's fine. (J5's questionnaire has notes, "Smoker" and "Non-smoker".)
3. **Patient answers** tab. Expected: the patient's original answers' summary, with no "Changed by clinician" and no orange box.
4. Change **Episode status** to **POA complete**. Expected: General notes adds "Status changed to POA complete".
5. **Add comment**, type `MT S03 comment`, **Save comment**. Expected: it appears in General notes with Cara's name and the time.
6. Reload. Expected: the status and the comment are kept.
7. The **Print** button is there (click it if you like; cancel the print dialog).

## Regression

**R-01: Hospital scope (S01).** Sign in as **Bea Author**. Expected: none of the `MT S03 …` or `E2E …` questionnaires are in her list.

**R-02: Viewer is read-only (S01).** Sign in as **Val Viewer** and open `MT S03 <time> J1`. Expected: "Read only", and nothing can be edited.

**R-03: The patient's stale save (S02a).** On a fresh episode's patient link (not yet sent), open the link in **two tabs first**, both at phone size. Then, in tab 1, answer "Do you smoke?" in About you and go back to **All sections**. In tab 2, open About you and answer the same question differently. Expected: tab 2 shows "Your answers weren't saved" with **Reload**; after Reload it shows tab 1's answer.

**R-04: Read-only after the review (S02).** On T-11's episode (review complete), open **About you** with **View**. Expected: "The HQ review is complete, so these answers can't be changed.", there's no **Validate** button, and the answers can't be changed.

## Out of scope (don't report)

- **Breaking the patient's "Send my answers" label to make J3 fail.** That needs a code change; the builder did it, and the verifier repeats it.
- The app's "Lifebox" branding, and the Next.js tools button at the bottom left (both go in later sessions).
- The `E2E …` questionnaires and `Sam J…` patients the suite leaves in Hospital A.
- The order of the trace files (`trace-1.zip`, `trace-2.zip`, …), which isn't fixed; the README says so.
- Features not built yet: patient records, invites, sign-in by magic link, AWS.
