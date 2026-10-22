# S04 test report 3 (retest after verification 1)

**5 passed, 0 failed, 1 blocked.** I ran the cases the plan asks for after the "Something went wrong" change: T-19, T-01, T-08, T-15, T-12 and R-01. T-19 is blocked because my tools can't block a network request.

**Environment:**
- **Commit:** `ac8ae27`. The working tree is not clean: S04 is still uncommitted (84 paths in `git status`).
- **When:** Mon 12 Oct 2026, 20:36–20:55 BST.
- **App:** http://localhost:3000 (API :8080 healthy), with `make dev` running in Rahul's terminal.
- **Browser:** Claude in Chrome with a 1512 × 757 viewport. Phone size used a same-origin 390 × 844 iframe, as the plan allows.
- **Window:** the Chrome window was behind other windows at first (`visibilityState: hidden`), so real clicks often didn't register. Rahul brought it to the front part-way through. After that, every click worked first time. While it was hidden, I signed Cara out once by calling the Sign out button's own click handler from the page; that was a set-up step, not a case.
- **Test data (all made up):** patient `S04 Retest2040`, episode `d076f457…` on `S04 Retest2001 v1`, now sent by the patient. Test case `Smoker 2045` on `S04 Test1714` v2 "About you". R-01 edited the smoking question on `S04 Retest2001` v2 (it now ends "(A) 2050").

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-19 Page code can't be downloaded | **Blocked (deferred)** | It needs DevTools' Network request blocking, which Claude in Chrome can't use. I can't fake it from the page either: the fix reloads the page once, and any page-side block I set up is lost on that reload. **Deferred (12 Oct, 21:05):** Rahul will run T-19 by hand later. In current Chrome, the panel is DevTools → **Request conditions** (it was "Network request blocking"): add the rule `*Editor*` set to block, tick **Disable cache** in the Network tab, and reload the Questionnaires list before step 3. Until then, T-19 stays open. Indirect evidence: in about 60 page loads, reloads and back/forward moves this session, the app never reloaded by itself and never showed "Something went wrong" or "Unexpected Application Error!". |
| T-01 New app, name Poised | Pass | Signed out on `/login`: the title is **Poised** with `/icon.svg`, the heading Poised, "Development sign-in: choose a user." and 4 users with emails. The old name appears nowhere. The source has `<div id="root">` and `/src/main.tsx`, and no `_next`. (It is 658 bytes instead of 608 because of Vite's `?t=` cache-busting stamp on `main.tsx`.) No console errors. |
| T-08 Preview, test cases | Pass | Editor (About you) → **Preview** → `/h/<A>/chapters/<id>/preview` with the Poised header. Yes reveals vape, and Clinical outputs show 77176002 Smoker / Patient smokes. Patient/Clinician and Phone/Tablet/Desktop change the layout. Saved test case `Smoker 2045` → Run all → "All 2 passed". It was still listed after a reload. **Edit** → editor with `?set=`. Preview, then Back → the editor on About you. No console errors. |
| T-15 Every address reloads / new tab | Pass | Every row opens directly and reloads correctly, with no 404, blank page, unexpected sign-in or unexpected self-reload, and no console errors: <ul><li>`/login` signed out;</li><li>`/p/<token>` signed out at 390 px;</li><li>the Questionnaires list, the editor with `?set=` and the preview as Alex;</li><li>the Episodes list, the episode, `sets/<cid>` and `/poa` as Cara.</li></ul>`sets/<cid>` also opened correctly when pasted into a brand-new tab. `/nope`, `/h/<A>/nope` and `/p` still show the 404 page. |
| T-12 Patient link at 390 × 844 | Pass | Signed out, using the fresh link. The dark **Poised** bar, "Hello S04", and `scrollWidth` 390, so no sideways scroll. Yes → vape appears, No, "Saved". After a reload: COMPLETE, and Review shows Yes / No. **Finish this section** → back to the list → Send my answers → in-app confirm → "Thank you, S04", the same after a reload. A tampered token gives the red "This link doesn't work. Please check it, or contact the hospital." GIF saved. |
| R-01 Two editors | Pass | Two side-by-side editors on the v2 draft's About you. A changed the smoking question's text and saved ("All changes saved"). B's edit got "Someone else changed this Question Set" + **Reload** and "Not saved". Reload showed A's text, and B's edit was dropped. |

## Failures

None. T-19 is blocked, not failed. Rahul has deferred it to a later hand check, so it is still open.

## Found while exploring

1. **Enter doesn't save a test case** (Minor, keyboard). In the preview's Test cases box, typing a name and pressing Enter does nothing; you have to click **Save answers**. I don't know if it was the same before S04. The test-case code only lost its `"use client"` line in this session, so it probably was.
2. **No unexpected reloads.** The new reload-on-failure code never fired during normal use. I went through sign-in, sign-out, deep links, Back/Forward, reloads inside iframes and the 404 page.
3. **Background window:** while the window was behind others, clicks often missed and drawers opened very slowly. The viewport also jumped from 2560 to 1512 px part-way through a form, which closed the New episode drawer. It kept my entered values when reopened, which is good. Once the window was in front, none of this happened again, so it was the tool, not the app.

## Console and network errors not tied to a case

- **Console:** no errors or warnings on any screen I opened.
- **Network:** only the expected codes:
  - the 401 from `/api/me` while signed out;
  - the 404 for the tampered patient link;
  - the 409 in R-01.

## GIFs

- `~/Downloads/S04-retest2-T12-patient.gif`: T-12 (the patient link at 390 px, from Start to Thank you and the tampered link).
