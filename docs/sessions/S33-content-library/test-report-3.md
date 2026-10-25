# S33 test report 3 (retest)

**6 passed, 0 failed, 0 blocked.**

- **Environment:**
  - Code: commit `fdc963b` plus the uncommitted S33 work, so the working tree isn't clean. Since report 2, only `apps/web/src/features/content/` changed (`ContentLibrary.tsx`, `content.ts`, `content.test.ts`).
  - When and where: Sun 25 Oct 2026, 19:00–19:06 GMT, at http://localhost:3000 (API :8080, run by the conductor's `make dev`).
  - Browser: Chrome ("trk" profile) through Claude in Chrome. Automatic mode.
- **Narrow widths:** the window can't be resized, so I tested 390px and 768px in a same-origin iframe of that width. I checked that the iframe's `innerWidth` was exactly 390 and 768.
- **Scope (the builder's request):**
  - T-14 and T-15, both updated in the plan;
  - a re-run of T-01, T-02 and T-12 as happy paths, plus T-06's read-only section;
  - a quick look at 768px.
  - The R-cases weren't re-run: no code outside the content feature changed since report 2, where they passed.
- **Test data I made:**
  - Trust A: "MT S33 1900 Enter leaflet" (retired).
  - Platform: "Knee video 1905" (tags Knee, Recovery, Ortho; retired).
  - I retired no starter question. All 11 are offered.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-14 Phone size | Pass | See the detail below the table. |
| T-15 Enter in Tags; switching kind | Pass | See the detail below the table. |
| T-01 Open the page | Pass | `/admin/content`, **Content library** highlighted. **Trusts** → the list; Back → the content library, highlighted. |
| T-02 Add a platform video with tags | Pass | "Knee video 1905 is added.": 3:20, Knee (typed + Enter) and Recovery (**+ Recovery**). The checklist's Recovery went from "nothing yet" to covered (8 → 7 of 11). Retiring it put Recovery back (8 of 11). |
| T-12 Starter questions list | Pass | 11 rows, each with its text, ID and tags. Each has **Retire**, so all are offered, including "Medicines you take regularly". |
| T-06 (partial) Trust A's read-only section | Pass | **From Preopsis (read-only)** lists the 3 examples and the 2 active Fasting leaflets, with no buttons. Sam's retired "Knee video 1905" isn't listed. |

**T-14 in detail**

- **Ada at 390px:**
  - The form's fields stack at the full width: Title, link, Length and Tags, each 324px wide.
  - **Trust A's items** scrolls sideways inside its own 358px box (the table is 820px).
  - **From Preopsis** also scrolls inside a 358px box (the table is 680px).
  - None of the 106 badges is clipped or spills out of its cell, and the 31 **Retired** badges read in full.
  - The page is exactly 390px wide.
- **Sam at 390px:**
  - The top bar scrolls on its own (a 358px box, 562px of contents). **PLATFORM ADMIN**, **Trusts**, **Content library** and **Sign out** each read in full.
  - Both tables scroll in their own 358px box (the tables are 820px and 760px).
  - **MEASUREMENTS** reads in full on "How tall are you?" and "What do you weigh now?" (label 99/99px, badge 121px in a 164px cell), including with the table scrolled fully right.
  - None of the 55 badges is clipped, including every checklist badge. The page is 390px wide.

**T-15 in detail**

- **Step 1, Enter and blank text in Tags:**
  - With the Hip tag added, I pressed Enter in the empty box, then typed three spaces and pressed Enter again. Nothing was sent: there was no request.
  - Clicking Title with the spaces still typed left only the Hip chip. The hidden value was `"Hip"`.
  - I also repeated report 2's exact steps (spaces in Tags, then click the **Video** switch): no empty chip.
- **Step 2, switching the kind:**
  - On **Video**, **Add video** showed the red "Only YouTube or Vimeo links…" message (POST 400).
  - Back on **Leaflet**, the message was gone.
- **Step 3, Enter in Title:** the leaflet was added (POST 201, stored with tags `["Hip"]`). I retired it afterwards.

## Failures

None. Report 2's failure (T-14: "MEASUREME…" clipped in the Starter questions table) is fixed. Its cosmetic exploring note (an empty tag chip after blurring Tags with only spaces typed) is fixed too.

## Found while exploring

- **The edit form handles tags well:**
  - I typed spaces in the edit form's Tags and then clicked Title: no empty chip.
  - Typing `knee` when **Knee** was already a tag added nothing, so a duplicate in different case is refused.
  - `  Ortho  ` was trimmed to "Ortho".
  - Saving stored `["Knee","Recovery","Ortho"]` and showed "Knee video 1905 is saved."
- **At 768px, nothing is broken.**
  - Both pages are exactly 768px wide.
  - Each items table (820px) is slightly wider than its 736px box. The **Retire** / **Bring back** column is reached by scrolling the table sideways; **Edit** shows without scrolling.
  - The hospital top bar also scrolls sideways at 768px.
  - No badge is clipped.
  - This isn't a failure. It's noted in case Rahul wants every button visible at tablet width.

## Console and network errors not tied to a case

- No console errors or warnings.
- **Tool artefact, as in report 2:** the extension recorded these requests as **503**:
  - every PATCH (Retire);
  - `POST /api/logout`;
  - `POST /api/dev/login`.

  Each one worked in the UI: the item was retired with its message, and sign-out and sign-in happened. This isn't an app bug.
- The only real 4xx was the expected 400 in T-15 step 2.

## GIFs

- `~/Downloads/S33-T15-retest.gif`: T-15 (Enter and spaces in Tags, blur, Video → Add video → the red message, back to Leaflet, Enter in Title adds it, retire).
- `~/Downloads/S33-T02-retest3.gif`: T-02 happy path (add a platform video with a typed tag and **+ Recovery**, Recovery ticked; edit-form tag checks; retire).
