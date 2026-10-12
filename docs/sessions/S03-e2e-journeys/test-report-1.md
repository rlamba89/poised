# S03 test report 1

**16 passed, 0 failed, 1 partly blocked** (T-04: the visible browsers couldn't be watched from the tester session).

- **Commit:** `165c7b8`. The working tree is not clean: S03's changes are uncommitted (`e2e/`, the Makefile, README and docs, as listed in `git status`).
- **When:** Mon 12 Oct 2026, 16:10–16:32 BST.
- **App:** http://localhost:3000 (API :8080 healthy), with `make dev` running in Rahul's terminal.
- **Browser:** Claude in Chrome, desktop window at 1440 wide. Part A used a terminal, as the plan allows.
- **Test data:**
  - questionnaires `MT S03 1612 J1` and `MT S03 1625 workflow`;
  - patients `Sam MT1625` (episode at POA complete) and `Sam MT1625b` (About you validated, the rest not).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Set-up instructions | Pass | Python 3.12.9. `make e2e-setup` finished with no error in 1.7 s, re-using the existing `e2e/.venv`. No guessing needed. |
| T-02 Old Python refused | Pass | It printed the exact message and exited with code 2. `e2e/.venv` was unchanged (still 3.12.9, same modified time). |
| T-03 Headless run | Pass | `make e2e`: `5 passed in 23.59s` (later runs took 19.9 s). No windows. |
| T-04 Headed run | Partly blocked | `make e2e ARGS=--headed` gave `5 passed in 27.97s`, and `ARGS="--headed --slowmo 300"` gave `5 passed in 39.49s`. The windows open on Rahul's screen, which I can't see. **Rahul, please watch the slow run** (see below). From reading the code, nothing in J3 or J4 looks like something a user couldn't do (see "Automation vs a real user"). |
| T-05 One journey; app not running | Pass | J3 alone: `1 passed in 7.98s`. With `E2E_BASE_URL=http://localhost:3999` it stopped in 0.6 s with "The app isn't answering at http://localhost:3999/api/health. Start it first: `make dev`." No journey ran. There's some noise around the message (see Minor 2). |
| T-06 Reading a trace | Pass | `--tracing on` kept 4 traces for J4: `trace-4.zip` is the browser trace (113 screenshots, 1440×900) and 1–3 are API traces with no screenshots. I opened it with the README's `show-trace` plus `--port 9323`, so it showed in Chrome, then stopped it. The Actions list includes Validate and Reload, Before and After show the page around a click, and Network filtered on "answers" shows PUT 200, **409** (in red, on page#2), 200, 200, 200. The next `make e2e` removed `e2e/test-results/`. One README mismatch: see Minor 1. |
| T-07 The rest of the docs | Pass | The root README's Tests table covers all three targets clearly, and the links and anchor resolve. The *Automated: J<n>* notes match what the journeys do (one wording nit: Cosmetic 1). |
| T-08 J1 Author | Pass | Signed in, created the questionnaire, and the header showed "v1 · draft". Set → page → Text → "What is your job?" gave "All changes saved". After a reload the card was kept, Preview (Patient) showed the box, and Back returned to the same page. |
| T-09 J2 Publish blocked | Pass | The steps as written can't be followed in order (see **Test plan issue**). Done the reachable way, it works: "1 logic problem" button, the Publish dialog lists "…tests a question that comes later or itself" with **Publish** disabled, the problem opens the Logic tab, and moving the question back clears it. Then "v1 · published" + "Read only", and **Create new version** gave "v2 · draft" with both questions and the condition kept. |
| T-10 J3 Patient on a phone | Pass | Episode page as expected; **Copy** changed to "Copied". The patient page at 390 px: "Hello Sam", About you and Your health only (no Nurse checks), "0 of 2 sections complete", Send disabled, and no sideways scroll (scrollWidth 388 = clientWidth 388). Leave and come back gave "In progress", and the answer was kept after a reload. Then Complete ×2, "2 of 2", "Thank you, Sam", still shown after a reload. Cara then saw "Ready for review" and the link gone. *How:* Chrome was full-screen, so I loaded the link in a 390×844 frame (see Environment notes). |
| T-11 J4 Clinician validates | Pass | Complete HQ review was disabled. Changing to No showed "Patient answered: Yes" and "Saved" (PUT 200); "Saving…" was too quick to see. Validate gave "Validated by Cara Clinician on …". Nurse checks with an empty answer gave "Some required questions have no answer yet."; I filled it and validated. Complete HQ review gave "Ready for POA", the note "HQ review completed by Cara Clinician", and **View** buttons. |
| T-12 J4 two tabs | Pass | Tab 1 changed the answer to No and showed "Saved". Tab 2's Validate showed a red **Not saved** box with the exact text and **Reload**, with Validate disabled. Reload showed No and "Patient answered: Yes". Validate then gave "Validated by Cara Clinician". |
| T-13 J5 POA Summary | Pass | Validated summary: the details, General notes, and the orange box "Changed by the clinician on Page 1 / Do you smoke? Patient answered “Yes”, now “No”". Patient answers: no box ("No answers produce anything for the summary yet.", as expected with no disclosure notes). Status → POA complete added a note, and the comment `MT S03 comment` was saved with Cara's name and the time. Both were kept after a reload, and **Print** is there. |
| R-01 Hospital scope | Pass | Bea Author: "No questionnaires found." |
| R-02 Viewer read-only | Pass | Val Viewer on `MT S03 1612 J1`: "v2 · draft", "Read only", with no Add content, Publish or card controls. |
| R-03 Patient stale save | Pass | Two phone-width views loaded first. View 1 answered Yes and went back to All sections; view 2 answered No and got "Your answers weren't saved" + **Reload**, and Reload showed Yes. |
| R-04 Read-only after review | Pass | View About you: "The HQ review is complete, so these answers can't be changed.", no Validate button, and clicking Yes did nothing. |

## Failures

None.

## Issues found (none blocks S03)

### Minor 1: The README's trace count is wrong for J3–J5
- **Where:** `e2e/README.md`, "Reading a trace".
- **What it says:** "There's one trace per browser, plus one for the data set up through the API."
- **What I saw:** J4 kept **4** traces: one browser trace, plus one per API session (Alex sets up the HQ, Cara the episode, and the patient link the answers).
- **Suggest:** "one per browser, plus one per API session used to set up data (J4 and J5 have three)". It would also help to show the real folder name once, e.g. `journeys-test-j4-clinician-validates-py-test-…-chromium/`: `<journey name>` suggests something short like `test_j4`.

### Minor 2: The "app not running" message is wrapped in pytest noise
- **Steps:** `E2E_BASE_URL=http://localhost:3999 make e2e`
- **What it prints:** a dangling `journeys/test_j1_author.py` line, then `no tests ran`, then `! _pytest.outcomes.Exit: The app isn't answering … !`.
- **Effect:** the message is correct and quick, but a newcomer reads past two odd lines first. Cosmetic.

### Test plan issue: T-09's steps can't be done in the order written (not an app bug)
- **The problem:** T-09 starts from J1's questionnaire, where "What is your job?" was added first, so it is already **above** "Do you smoke?". Step 2 asks for a condition on "Do you smoke?", but the Logic picker only offers *earlier* questions ("Only questions that come earlier can be tested…"), so "Do you smoke?" isn't offered.
- **What I did:** moved "What is your job?" down first, set the condition, then moved it up for step 3.
- **Suggest for next time:** add "move it below Do you smoke? first" to step 2.

### Cosmetic 1: The note on MT-66 undersells J3
- **The note:** "Clinician-only questions aren't in its HQ."
- **What J3 does:** J3's HQ *does* have a clinician Question Set (Nurse checks). J3 checks it's hidden only indirectly, through "0 of 2 sections complete"; it never asserts that "Nurse checks" is absent.
- **Suggest:** reword the note, or add an explicit `expect(...Nurse checks...).to_be_hidden()`. That is MT-66's main privacy expectation.

## Automation vs a real user (Part B comparison)

- **J1:** the same as by hand. J1 presses Tab to leave the box, and I clicked out of it; both save.
- **J2:**
  - J2 uses a different logic problem (a missing question, set up through the API). The plan says that's fine.
  - The UI behaves the same for both kinds of problem, except as described under "Found while exploring", item 1.
- **J3:**
  - **Date of birth:** J3 fills the date box with `1980-05-17`, but a user types `dd/mm/yyyy` into Chrome's date control. This works either way.
  - **The link:** J3 reads the link from the clipboard and opens it with `goto`, where a user would paste it. Equivalent.
  - **"Hello Sam":** J3 checks the heading (the HQ name), not "Hello Sam".
  - **Layout:** nothing in J3 checks the phone layout (no sideways scroll, nothing cut off). By hand it was fine at 390 px.
- **J4:** the same steps as by hand. Its setup (the patient's answers) goes through the API, as the README says.
- **J5:** the same as by hand.

## Found while exploring

1. **Logic tab for a "comes later" problem shows an empty condition.**
   - Clicking the logic problem opens the Logic tab, but the condition shows as an empty "Select a question…" picker. There's no hint of what's wrong, while the card still says "Displays when Do you smoke? is Yes".
   - An author following "click the problem to fix it" sees a blank condition, not the cause.
   - Minor UX. This existed before S03, and J2's "missing question" problem shows the ID instead.
2. **A question card in a Clinician Question Set is badged "PATIENT".** "Blood pressure" in Nurse checks, a set marked Clinician, shows a PATIENT badge on its card. Confusing for authors. Cosmetic, and it existed before S03.
3. **Undo is unreachable while a question is selected.**
   - The Settings panel replaces the Structure panel, and Undo lives in Structure.
   - After a mis-click on **Copy**, I had to close Settings before I could undo.
   - Minor UX, and it existed before S03.
4. **The New episode "Health questionnaire" dropdown can't be searched.**
   - With the dozens of `E2E …` HQs that the suite leaves behind, finding your own HQ means scrolling a long list. Typing does nothing.
   - The README says to rebuild the database when it gets cluttered, but each `make e2e` run adds 4 more published HQs.
   - Minor.
5. **Clicks straight after a page loads were sometimes ignored.**
   - This happened three times: New Questionnaire, Sign out, and Val Viewer on the stub sign-in. A second click a moment later worked.
   - It's probably clicking before the page finished loading its scripts (dev mode). It may be partly due to my tool's timing, so it's worth one look, not a bug report.
6. **Validate page says "Saved" before anything is changed.** Cosmetic, and it existed before S03.
7. **Things that worked:**
   - A stale patient tab answering *after* the HQ was sent got "Your answers weren't saved. Your answers have already been sent, so they can't be changed here." + Reload.
   - Double-clicking the patient's confirm **Send my answers** sent once and showed the thank-you page.
   - Bea opening Hospital A's episode URL got "You don't have access to this hospital."
   - Bea opening Hospital A's questionnaire under her own hospital's path got "Questionnaire not found."
   - Enter in the New Questionnaire Description adds a newline (it's a multi-line box), as expected.

## Console and network errors

- **Console:** no errors or warnings in the tabs I used. The extension only starts listening when first asked, so the first few page loads weren't covered.
- **Network:** the extension listed **503** for every request that returns an empty 204 (PATCH chapter, publish, logout, dev login, complete-review, PATCH episode). The UI behaved as if each succeeded. With curl, `POST /api/logout` returns `204 No Content` through both :3000 and :8080, so this is the extension's reporting, not the app. There were no other unexpected 4xx/5xx.

## Environment notes

- **Phone size:** Chrome was in full-screen (2560×1296), so `resize_window` to 390×844 had no effect at first.
  - For T-10 and R-03 I loaded the patient link in 390×844 frames on a page of the same app, so the page laid out at phone width.
  - This doesn't emulate touch or Safari; J3 covers WebKit on an iPhone 13.
  - The window did shrink later, so I resized it back to 1440 for the desktop cases.
- **Private window:** not available to my tools. The patient page opened in the same browser as Cara's session; the patient page uses only its link, so this made no difference here.

## For Rahul: T-04, please watch the slow run

Run `make e2e ARGS="--headed --slowmo 300"` and watch J3 and J4. Check:
1. A desktop Chromium window opens for staff.
2. During J3, a **phone-sized** WebKit window opens for the patient.
3. Nothing is typed or clicked somewhere that isn't visible.

## GIFs

In `~/Downloads/`:
- `S03-T08-J1-happy.gif`: J1 by hand.
- `S03-T09-J2-publish.gif`: the logic problem, publish, and new version.
- `S03-T10-J3-patient-happy.gif`: the episode, the patient at phone width, and sending.
- `S03-T12-J4-two-tabs.gif`: the stale tab's "Not saved" and Reload.
- `S03-R03-patient-stale-save.gif`: the patient's two tabs.
