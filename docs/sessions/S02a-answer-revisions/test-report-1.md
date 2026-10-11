# S02a test report 1

**15 passed, 0 failed, 0 blocked** (11 cases + 4 regression checks). One cosmetic issue found while exploring.

**Environment**
- Commit `b3876f9` plus the uncommitted S02a work. The working tree is not clean, as expected before verification.
- Tested 9 Oct 2026, 23:05–23:28 BST, at http://localhost:3000.
- Rahul restarted `make dev` at 23:05, so the API was rebuilt. `/api/health` returned ok.
- Browser: Claude in Chrome.
- **Phone size:** the browser window wouldn't resize (it stayed 2560 px wide), so phone-size cases (T-03 to T-06) ran in **390 × 844 same-origin iframes**, as the plan allows. Two side by side served as device A and device B.
- Test data:
  - HQ `S02 test HQ 23:06` (published v1, then a draft v2 for R-04).
  - Episode 1: `S02a Test2307` (T-02 to T-09, R-03).
  - Episode 2: `S02a Test2318` (T-10).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Make the test HQ | Pass | Script printed `Created and published: S02 test HQ 23:06`. Listed as published v1 with About you, Your health and Nurse checks. |
| T-02 Create an episode | Pass | Status HQ not complete, with Patient link and Copy. |
| T-03 Patient normal use (phone) | Pass | No STAFF ONLY. Smoking No saved (PUT 200). Reopened: In progress, No kept. All sections with no change sent no save. |
| T-04 Link on two devices (phone) | Pass | The stale device got the "Your answers weren't saved" card with the exact message and Reload (PUT 409). No sideways scroll (scrollWidth = clientWidth = 386). Reload showed smoking Yes and angina unanswered. Answering again saved (200). The other device leaving unchanged sent nothing. |
| T-05 Two first saves (phone) | Pass | B's first save → 409 and the card. Reload showed A's diabetes answer. |
| T-06 Saved after sending (phone) | Pass | Card said "Your answers have already been sent, so they can't be changed here." Reload showed the thank-you page. |
| T-07 Stale clinician Validate (X-1) | Pass | See below. |
| T-08 Two first saves, clinician set | Pass | B: Not saved box (409). Reload showed A's 120/80. |
| T-09 Normal clinician use | Pass | "Patient answered" notes; Validated by Cara Clinician with time; Complete HQ review → Ready for POA, with the General note; POA Summary showed both corrections. |
| T-10 Stale tab after review complete | Pass | Not saved box with "The HQ review is already complete, so the answers can't be changed." Reload showed the set read-only with "The HQ review is complete, so these answers can't be changed." |
| T-11 Refusals | Pass | Alex: "Only clinicians can see patients and episodes." (403). Bea: "You don't have access to this hospital." (403). No answers shown. |
| R-01 No clinician content for patient | Pass | No STAFF ONLY or "Current smoker" on any patient screen. The patient API response had no `STAFF ONLY`, `clinicalOutputs`, `testCases` or `q_bp` (checked with a fetch in the page). |
| R-02 Leave and come back | Pass | Reloading a device mid-section went back to the list, with both sections Complete and answers kept. |
| R-03 Episode details still change | Pass | Nurse ASA set to 2, then cleared; status → POA complete. Each was saved, with no message, and General notes recorded all three. |
| R-04 Editor stale-tab check | Pass, roles swapped | The editor saves on blur. Tab B blurred first and saved (200). Stale tab A then got "Someone else changed this chapter since you opened it. Reload to see their changes; your latest edits were not saved." with Reload (409). This is the same check as written, with the tabs the other way round. |

### T-07 in detail (the bug S02a fixes)
1. Tab A changed smoking Yes → No. "Patient answered: Yes" appeared, then Saved.
2. Tab B, still showing Yes, clicked **Validate**:
   - B sent a single PUT, which got **409**, and nothing else.
   - The red **Not saved** box appeared with the exact message and Reload. The status said **Not saved**.
   - **Validate** was disabled, and clicking **No** didn't change anything.
   - Tab B stayed on the set page.
3. Reload in B showed tab A's **No**, with no "Validated by" badge.
4. The POA Summary showed "Do you smoke? Patient answered 'Yes', now 'No'", and About you stayed **In progress**, not validated.
5. Keyboard check: Reload can be reached with Tab and works with Enter. The Not saved box has `role="alert"`.

## Failures

None.

## Found while exploring

1. **Cosmetic: the Reload button's label is cut to "Rel" in a narrow clinician window (about 500 px).**
   - Where: the clinician set page (`/h/{hid}/episodes/{eid}/sets/{sid}`), signed in as Cara Clinician, in a window about 500 px wide.
   - Steps: make a stale save (T-07 or T-08). The Reload button inside the red Not saved box shows only "Rel".
   - Cause: the label needs 40 px, but the flex row shrinks the button to 48 px wide, leaving 18 px for the label.
   - At desktop width (1512 px) the label shows in full.
   - It still works, and its accessible name is "Reload".
   - Probably needs `flex-shrink: 0` on the button, or a wrapping layout.
   - This isn't the "staff header wraps at narrow widths" item already in pending.md; it's the new S02a alert.
2. **Observation, no severity (please confirm it's intended).**
   - After a refused clinician save, the page keeps showing the unsaved answer until Reload.
   - In T-10, the page showed smoking **Yes** and the Lifestyle summary listed **Current smoker**, both from the refused change.
   - The red box and the locked form make that clear, so I didn't count it as a failure.
3. **Pre-existing, not S02a: the editor says "All changes saved" while a title edit is still in the focused field.**
   - The save happens only on blur.
   - In R-04, tab A said "All changes saved" for several seconds with an unsaved title, and then lost the conflict.
   - A stale editor tab therefore only finds out when the field loses focus.
4. **Good: no self-conflicts.**
   - Several times one tab sent two saves back-to-back (an autosave, then Validate a moment later): T-09 About you, and T-10 Nurse checks. Both saves got 200 each time.
   - So the saver's revision chaining works within one tab.
5. **Expected by the plan:** a second patient tab sitting on the list keeps a stale section status ("In progress" after the other device finished). It only finds out on save.

**Not explored, because the browser tool's input became unreliable** (see below):
- double-clicking Validate and Send my answers;
- odd text (emoji, very long text) in clinician comments;
- the back button after a conflict.

## Console and network errors not tied to a case

- **Console:** no red errors and no warnings from the app. There were only React DevTools and HMR info lines.
- **Network:** every 4xx was expected:
  - the deliberate 409s;
  - 403s in T-11;
  - 401s on the sign-in page.
- 503s appeared only on 204 routes (sign-in, sign-out, submit, complete-review). This is the known tool quirk.
- Successful answer saves returned **200**, as the plan says.

**Test-environment notes (not app bugs):**
- From about T-10 onwards, the tool's mouse clicks and some key presses were often lost, mostly just after a page load:
  - clicking Validate, a sign-in name, or New episode did nothing;
  - text typed just after a set page loaded was lost.
- I checked:
  - the button wasn't disabled;
  - the same Validate fired from script worked at once (PUT 200, back to the episode).
- `document.visibilityState` reported `hidden` for every tab, even after Rahul brought Chrome to the front.
- Screenshots of the iframes also lagged behind the page.
- I retried by element reference, keyboard or script. I've only marked a case as passing where the result was confirmed in the page and in the network log.

## GIFs

- `~/Downloads/S02a-T04.gif`: the main patient happy path and conflict, on two devices.
- `~/Downloads/S02a-T07.gif`: X-1, a stale clinician tab's Validate refused, then Reload and the POA Summary.

There are no failures, so there are no failure GIFs.
