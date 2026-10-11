# S02a test report 2 (retest after verification 1)

**10 passed, 0 failed, 0 blocked.** That's the retest list at the top of the test plan (T-03 to T-07, T-12, R-01 and R-02), plus 2 short explorations.

**Read this first: the tester was not independent.** This retest ran in the same Claude session that built S02a and fixed verification 1's finding. I kept to the tester's rules (no code changes, the plan's steps in order), but I knew the code. Rahul may want a fresh `/manual-test S02a retest` for an unbiased run.

## Environment

- **Code:** commit `b3876f9` plus S02a's uncommitted work, including the verification 1 fix. The working tree is not clean, as expected before verification.
- **When and where:** 11 Oct 2026, 20:32 to about 21:25 BST, at http://localhost:3000.
- **The API:** `/api/health` was ok, and the API was the S02a build (`GET /api/p/{token}` returns `revisions`).
- **Test data:** HQ `S02 test HQ 23:06` (published v1, from report 1), and two new made-up episodes:
  - **`S02a Retest2102a`:** T-03 to T-07;
  - **`S02a Retest2102b`:** T-12, R-01 and R-02.
- **How the browser was driven (deviations from the plan).** Chrome reported the controlled tab as **hidden** the whole time, even with the window in front, and no animation frames ran. As a result:
  - clicks on the screen and dialogs' opening animations didn't work;
  - so the steps were run by clicking the app's own buttons and answer options from script;
  - every result was confirmed in the page's text and in each frame's log of requests and status codes.
- **Specific workarounds:**
  - **Phone size (T-03 to T-06, T-12, R-01, R-02):** 390 × 844 same-origin frames, as the plan allows. "Tab A" and "tab B" were two frames side by side.
  - **T-02 setup:** the two episodes were created with the same API calls the New episode drawer makes, as Cara Clinician, because the drawer's dropdowns can't open in a hidden tab. T-02 itself passed through the UI in report 1.
  - **T-06 step 1:** the "Send your answers?" confirmation's content never appeared, because its animation needs frames. So "Send my answers" was done with the same `POST …/submit` request the dialog sends, from device A. Report 1 confirmed the dialog itself in a visible tab.
  - **T-12 "Offline":** device A's answer saves were made to fail ("Failed to fetch"), because DevTools' throttling menu can't be reached from the tool.
  - **Staff sign-in:** the same `POST /api/dev/login` the login page's buttons send. The first sign-in was a real click on Cara's button.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-03 Patient normal use (phone) | Pass | No STAFF ONLY. Smoking No saved (PUT 200, "Saved"). Opened again: About you **In progress**, No kept. All sections with no change: no request, no message. |
| T-04 Link on two devices (phone) | Pass | Run with the answers reversed (both started at Yes/Yes; B changed smoking, A changed angina). A got **Your answers weren't saved** with the exact message and **Reload** (PUT 409), no sideways scroll (388 of 388 px). Reload showed B's smoking change, and A's angina change wasn't kept. A's next change saved (200) with no message. B left unchanged: no request, no message. |
| T-05 Two first saves (phone) | Pass | Your health: A's first save 200; B's 409 with the card; Reload showed A's **Yes**. |
| T-06 Saved after sending (phone) | Pass | Submit 204, A showed the thank-you page. B, still on Your health, changed an answer: card with "Your answers have already been sent, so they can't be changed here." and Reload (PUT 409). Reload showed the thank-you page. (Submit was sent without the dialog; see Environment.) |
| **T-12 A failed save is sent again** (new) | Pass | See below. |
| T-07 Stale clinician Validate (X-1) | Pass | See below. |
| R-01 No clinician content for patient | Pass | Five patient screens (list, both sets, after answering smoking Yes) and the patient JSON: no STAFF, `clinicianOnly`, `clinicalOutputs`, `testCases`, `q_bp` or "Current smoker". |
| R-02 Leave and come back | Pass | Reloaded the device in the middle of About you: back on the list, About you **Complete**, and the answers kept. |

### T-12 in detail (verification 1's fix)
1. With saves failing, I answered both About you questions. **"Failed to fetch"** showed at the top, and both saves failed.
2. **All sections:**
   - About you shows **Start**, not Complete;
   - "0 of 2 sections complete";
   - **Send my answers is disabled.**
3. Saves working again, I dismissed the error and reopened About you. Both answers (No, No) are still shown.
4. **All sections** without changing anything:
   - **one PUT, 200**;
   - About you now shows **Complete**, with "1 of 2 sections complete".
5. Opened the link again: About you is Complete, with the same answers. The server has them.

### T-07 in detail (X-1)
1. **Tab A** changed smoking No → Yes. "Patient answered: No" appeared, then Saved (PUT 200).
2. **Tab B,** still showing No, clicked **Validate**:
   - a single PUT got **409**;
   - the **Not saved** box showed the exact message and **Reload**;
   - the status said **Not saved**, Validate was disabled, and tab B stayed on the set page.
3. **The form is locked.** I re-checked with a second stale tab: clicking the other smoking answer changed nothing and sent no request. The refused angina change stays visible until Reload, as the build notes describe.
4. **Reload** showed tab A's **Yes**, with no "Validated by" badge and no box.
5. **POA Summary:** "Do you smoke: Patient answered 'No', now 'Yes'".
6. **Episode page:** About you is **In progress**, not validated.

## Failures

None.

## Found while exploring

1. **Good: rapid changes on one device don't conflict with themselves.** Six angina changes about 0.7 s apart gave three saves, because the half-second delay merges some. All three were 200, with no message, and the server held the final answer.
2. **Not S02a, already decided: the New episode drawer.** Rahul reviewed it during this session. It will be replaced in C2 by a dedicated page with a hospital-scoped patient search and silent matching (decisions.md, 11 Oct; the C2 plan; the S13 brief). Not reported as a failure.
3. **Test environment, not the app:**
   - Chrome kept the controlled tab "hidden" even with the window in front, so no animation frames ran;
   - modals and the drawer stayed in the page but invisible, and clicks on the screen were lost;
   - timers were sometimes slowed (one scripted step took about a minute);
   - report 1 saw the same.

## Console and network

- **Console:** no errors or warnings. I captured one patient screen and one clinician screen loading.
- **Network:** every 409 was one a case caused on purpose. Successful answer saves returned 200.
  - The 503s were only on `dev/login` and `logout`. Those routes return 204, so this is the known tool quirk.

## GIFs

None. A GIF records the visible screen, and the tab was hidden throughout. Report 1's GIFs (`~/Downloads/S02a-T04.gif`, `S02a-T07.gif`) show the same flows.
