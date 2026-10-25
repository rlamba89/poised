# S30 test report 1

**Summary:** 9 passed, 0 failed, 0 blocked (5 cases and 4 regression checks).

**Environment:**
- Commit `c29edd8`. The working tree isn't clean: it holds S30's uncommitted changes (`api.ts`, `PatientHQ.tsx`, `prefill.ts`, tests, docs, e2e) and the untracked `docs/market/` and the S30 folder.
- Sunday 25 Oct 2026, 15:31–15:50 GMT.
- http://localhost:3000, with the API on :8080. The conductor ran `make dev`.
- Browser: Claude in Chrome, "trk" profile.

**How the cases were run:**
- **Changed fetch:** (A), (B) and (C) were applied with the page-scripting tool, exactly as the plan gives them.
- **Phone size:** the window can't be resized, so patient pages ran inside a same-origin 390 × 844 iframe. The fetch overrides went on the iframe's `contentWindow`, and "F5" was `contentWindow.location.reload()`.
- **Overflow check:** in every phone-size check, `scrollWidth` was 390 and `clientWidth` was 390.

**Test data:** all made up, created through the UI.
- **Patients** (all "Sam", date of birth 17/05/1980, female):
  - `S30 T02 1531`, `s30.t02.1531@example.test`: Trust A only, Demo pre-op HQ v5.
  - `S30 T04 1531`, `s30.t04.1531@example.test`: Trust B episode on Demo pre-op HQ, sent by the patient. Then two Trust A episodes on Demo follow-up HQ. The second one was made for R-02.
  - `S30 T05 1531`, `s30.t05.1531@example.test`: Trust B episode sent by the patient. Its Trust A episode was cancelled in T-05.
- **Comments** saved on T02's POA Summary: "MT S30 T01 after reload".

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Staff page, gateway JSON 502 | Pass | **Save comment** shows "We couldn't reach the server. Check your connection and try again." Episodes, Questionnaires and the POA Summary (reached with Back) show the same message. After a reload the comment saves. |
| T-02 Patient page, gateway JSON 502 (phone) | Pass | Changing an answer shows the plain message as a dismissible banner, with "Not saved". No "Something went wrong" and no overflow. A reload works again. |
| T-03 The API's own message wins | Pass | Patient: the "Your answers weren't saved" card says "Made-up refusal from the API." with **Reload**. Staff comment: "Made-up refusal from the API." in red. Wrong date of birth: "That date of birth doesn't match our records… You have 4 tries left." |
| T-04 Start fresh, reload fails: card only (phone) | Pass | Offline: only the cleared card, with no red banner. **Reload** while offline: still only the card. Back online: About you opens empty, with no card or banner. With the read-only 502 on Your health: only the card. |
| T-05 Reload refused because cancelled: message shows | Pass | Cara's fill-in for the patient, offline, then **Start fresh**: only the card. After the episode is cancelled in a second tab and the reads are back on, **Reload** shows the red "This questionnaire was cancelled by the hospital, so there's nothing more to fill in…" above the card, and the card stays. |
| R-01 Rejected fetch | Pass | **Save comment** shows "We couldn't reach the server…", not "Failed to fetch". |
| R-02 Plain Start fresh | Pass | On a new T04 Trust A episode, as Cara for the patient: About you opens empty, with no card or banner. After a reload it shows **Start** (no answers) and opens empty. |
| R-03 Patient opens a cancelled episode | Pass | The T05 patient signed in through the Trust B invite. Their home shows the Hospital A1 card "Cancelled by the hospital" with its message. Opening `/home/episodes/<T05 A>` shows the cancelled message. |
| R-04 /qa/jobs, Now + 2 days | Pass | "Ran 96 jobs as of 27/10/2026, 15:47: 14 reminders sent, 0 marked "not responding", 0 episodes archived." Captured messages lists new reminder emails at the top. |

## Failures

None.

## Found while exploring

Nothing that needs a fix. What I checked:

1. **Other gateway answers on Save comment** (staff POA Summary):
   - A 500 with the API's `{"error":"API own 500 message."}` shows that message. This is correct.
   - A 504 with an HTML body shows the network message.
   - A 503 with an empty body shows the network message.
   - A 404 with the gateway's `{"message":"Not Found"}` shows "Something went wrong. Please try again." That matches the brief: only 5xx answers without the API's message count as the network message.
2. **The cleared card after T-04's 502:**
   - With an API 409 on reads, **Reload** puts "Made-up refusal from the API." in a red banner above the card. That fits the T-05 design.
   - Switching back to the 502 and pressing **Reload** removes the banner and leaves only the card again.
3. **The patient's "Your questionnaires" link under (A):** the plain network message, at phone size.
4. **T-05 extra:** pressing **Reload** again after the cancelled banner shows keeps the same single banner and the card. Nothing is duplicated.
5. **Cosmetic, a question rather than a fault:** in T-05 the cancelled banner sits above a card that still says "The questionnaire couldn't be loaded again just now. Please try again." with a **Reload** button. The plan says the card stays, so this isn't a failure. Once the episode is known to be cancelled, though, "Please try again" and **Reload** offer an action that can't succeed.
6. **Not the app (tester tooling):** typing into a just-focused staff form field sometimes failed with "Cannot access a chrome-extension:// URL of different extension". Another extension's popup in this Chrome profile, probably autofill, caused it. Using `form_input` avoided it. A click right after that sometimes needed repeating.

## Console and network errors

- No console errors or warnings were captured.
- Console and network tracking only started late in the run, on `/qa/jobs` and `/qa/messages`. Every 4xx/5xx seen during the cases was simulated on purpose in the browser. The real API answered normally throughout.

## GIFs

Saved in `~/Downloads/`:
- `S30-T01.gif`: the staff 502 path, then the reload and save.
- `S30-T02-T03.gif`: the patient 502 at phone size, then the API's 409 message.
- `S30-T04.gif`: Start fresh with failing reloads, and the read-only 502.
- `S30-T05.gif`: the cancelled-episode banner above the card.

## Hand checks for Rahul

None from this round.
