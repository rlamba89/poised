# S04 test report 2 (retest)

**10 passed, 0 failed, 0 blocked.** I re-ran:
- T-16 in full (step 4 corrected in the plan);
- T-14 step 5 (Print), as far as I can test it;
- the regression cases R-01 to R-05;
- three main happy paths.

**Environment:**
- **Commit:** `ac8ae27`. The working tree is not clean: S04 is still uncommitted (77 paths in `git status`). No code changed since report 1, as the plan's note says.
- **When:** Mon 12 Oct 2026, 20:00–20:14 BST.
- **App:** http://localhost:3000 (API :8080 healthy), with `make dev` running in Rahul's terminal.
- **Browser:** Claude in Chrome with a 1512 × 757 viewport. Phone-size checks used same-origin 390 px iframes, as the plan allows.
- **Test data (all made up):**
  - questionnaire `S04 Retest2001`: v1 published, v2 draft;
  - patient `S04 Retest2015`, episode `d3fa5cf7…`, now at Ready for POA;
  - `S04 Delete2012`, created and then deleted.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-16 Back, Forward, scrolling | Pass | **Steps 1–3:** list → episode → POA Summary; Back ×2 gave the episode, then the list; Forward ×2 gave the episode, then the POA Summary. **Step 4:** the list scrolled to 1000 px → an episode opens at the top (`scrollY` 0) → Back shows the list at the top, as the corrected plan expects. **Step 5:** the episode scrolled a little (18 px; the page is barely taller than the window) → the POA Summary opens at the top → Back shows the episode, with nothing broken. No console errors. |
| T-14 step 5 Print | Pass (dialog not opened) | Unchanged from report 1. With `window.print` stubbed, Print calls it once. The print CSS hides the breadcrumbs, Print, the status picker, the ASA Save buttons and the header. I didn't open the real print dialog, because it freezes the browser extension. **Rahul: please click Print once by hand.** |
| R-01 Two editors | Pass | On the v2 draft, tab A changed the smoking question's text and saved. Tab B's edit got "Someone else changed this Question Set" + **Reload** and "Not saved". Reload showed tab A's text and dropped tab B's. |
| R-02 Two validate tabs | Pass | Two side-by-side frames on About you. A changed vape → Yes and saved. B changed smoking → No: "Not saved … changed somewhere else" + **Reload**, with Validate disabled. Reload showed A's answers and nothing from B. |
| R-03 Patient on two "devices" | Pass | Two 390 px frames on a fresh link. A answered Yes and saved. B answered No: "Your answers weren't saved" + **Reload**, and Reload showed Yes. |
| R-04 Duplicate name | Pass | "A draft questionnaire with this name already exists." in the dialog, with POST 409. Nothing was created. |
| R-05 Delete a draft | Pass | ⋯ → Delete → the "Delete questionnaire?" modal names `S04 Delete2012` → Delete. The row is gone, and the other rows remain. |
| Happy path 1: authoring | Pass | Sign in → New Questionnaire → Question Set renamed "About you" → Add Page → two Yes/No questions → Logic "show vape when smoke is Yes" (orange border, "Displays when…") → Preview (Yes reveals vape) → Edit → Publish → "v1 · PUBLISHED" + Read only → Create new version → "v2 · DRAFT". GIF saved. |
| Happy path 2: patient at 390 px | Pass | A new episode → patient link → "Hello S04", `scrollWidth` 390 → answers → All sections shows COMPLETE → Send my answers → in-app confirm → "Thank you, S04", still shown on a later visit. |
| Happy path 3: clinician to POA | Pass | READY FOR REVIEW → Validate → smoking changed to No ("Patient answered: Yes") → Validate → "Validated by Cara Clinician" → Complete HQ review → READY FOR POA → POA Summary shows "Changed by the clinician on Page 1 · Do you smoke? Patient answered “Yes”, now “No”". |

## Failures

None.

## Found while exploring

1. **Two items in report 1 were caused by my tools, not the app; please disregard them.** In this run, `document.visibilityState` was `hidden` because the Chrome window wasn't in front, so Chrome slows the page's animations. That explains:
   - report 1's "the delete modal flashes an empty name as it closes" (Cosmetic 2): the closing animation was simply frozen part-way;
   - report 1's tool note about first clicks not registering.

   Both happened again here and cleared after a longer wait.
2. **The POA Summary lists a change to a question the condition now hides** (Minor, not from S04). In R-02, the clinician set "Do you vape?" to Yes. Changing smoking to No then hid that question, but the validated answer was kept. The POA Summary still says "Do you vape? Patient answered “No”, now “Yes”" for a question the clinician can no longer see. This comes from how answers to hidden questions are stored (the clinical logic, which S04 didn't touch). It may be worth a look later.

## Console and network errors not tied to a case

- **Console:** no errors or warnings. I reloaded the Episodes list, the POA Summary, the validate screen, the patient page and a 404 address with capture on, and checked that a test warning was captured.
- **Network:** only the 409s that R-02 to R-04 expect.

## GIFs

- `~/Downloads/S04-retest-happy-path-authoring.gif`: happy path 1 (authoring through publish and new version).
- `~/Downloads/S04-retest-T16.gif`: T-16 (Back, Forward and scrolling).
