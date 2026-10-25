# S33 test report 2 (retest)

**4 passed, 1 failed, 0 blocked.**

- Environment: commit `fdc963b` plus the uncommitted S33 work (working tree not clean), Sun 25 Oct 2026 18:40–18:48 GMT, http://localhost:3000 (API :8080, run by the conductor's `make dev`), Chrome ("trk" profile) through Claude in Chrome. Automatic mode.
- Phone size: the window can't be resized, so 390px was tested with a same-origin 390×844 iframe of the page.
- Scope (the builder's request): T-14 (rewritten), the new T-15, and a re-run of T-02, T-05 and T-07.
- Test data made:
  - Platform: "Knee video 1842" (retired), "Fasting leaflet 1845" (active, Fasting).
  - Trust A: "A hip video 1850" (active, Hip + Recovery, 12:30), "A who-to-call leaflet 1851" (active, Who to call), "Enter leaflet 1855" (retired).
  - I also retired round 1's "A who-to-call leaflet 1830", so that T-07 could show **Who to call** going from "nothing yet" to covered. "A who-to-call leaflet 1851" now covers that topic.
- No starter question was retired this round.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-02 Add a platform video with tags | Pass | "Knee video 1842 is added."; Video, 3:20, KNEE + RECOVERY; the link has `target=_blank` and `rel="noopener noreferrer"`; Recovery went from "nothing yet" to covered (8 → 7 of 11). |
| T-05 Leaflet; edit; retire; bring back | Pass | The leaflet form has no Length field and the row shows "—". Edit showed 3:20; I removed Knee and changed the length to 4:05, then Enter in Length gave "… is saved." Retire gave the plain message, and the row moved last, greyed, with a RETIRED badge and Bring back; Recovery went back to "nothing yet". Bring back gave "… is offered again." and Recovery covered. Retired again at the end. |
| T-07 Trust A video and leaflet | Pass | "A hip video 1850" was added under Trust A's items (private Vimeo link, 12:05, HIP). The who-to-call leaflet ticked **Who to call** from "nothing yet" to covered. The edit to + Recovery and 12:30 saved. `https://example.org/film.mp4` on Edit was refused with the T-03 message. |
| T-14 Phone size | **Fail** | Step 1 (Ada) passes: the form's fields stack; both tables scroll sideways inside their own box (358px wide, contents 820/680px); none of the 62 badges is cut off; the page is 390px wide. Step 2 (Sam): the top bar now scrolls on its own and every label reads in full. The items table is fine, but in the **Starter questions** table the **MEASUREMENTS** tag is cut to "MEASUREME…" on both rows, even when the table is scrolled fully right. See below. |
| T-15 Enter in Tags; switching kind | Pass | With the Hip tag added, Enter in the empty box and Enter after spaces only sent nothing (no request) and the form stayed open. Video → Add video gave the red "Only YouTube or Vimeo links…" (POST 400). Back to Leaflet: the message was gone. Enter in Title added the leaflet (POST 201, tags `["Hip"]`). Retired afterwards. One cosmetic side effect is under "Found while exploring". |

## Failures

### T-14: in the Starter questions table at phone width, the "Measurements" tag is cut off (Minor)

- **Who / where:** Sam Superadmin on `/admin/content`, 390px wide (same-origin iframe).
- **Steps:**
  1. Open `/admin/content` at 390px.
  2. Scroll down to **Starter questions**.
  3. Scroll that table sideways to the right end.
  4. Look at the tags of "How tall are you?" and "What do you weigh now?".
- **Expected (plan, step 2):** the Starter questions table behaves as in step 1: every tag reads in full.
- **Actual:** both tags show "MEASUREME…".
  - The table does scroll inside its own box (box 358px, table 760px wide).
  - But its Tags column is only about 128px wide, so the Badge label is clipped with an ellipsis (label scrollWidth 99 vs clientWidth 86).
  - The other tags in that table (LIFESTYLE, ANAESTHETIC, PREGNANCY, …) fit. They wrap to two lines where there are two.
  - At desktop width the same badge reads in full: the column is 218px and the label isn't clipped.
  - Only this table is affected. The items tables on both pages had no clipped badge.
- **No console errors.** Screenshot: `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-n6rM6I/screenshot-1792953798942-2.png`.

## Found while exploring

- **Spaces-only text in Tags becomes an empty tag chip when the box loses focus** (Cosmetic).
  - Steps: in T-15, I typed three spaces in Tags and pressed Enter (correctly nothing happened). Then I clicked the **Video** switch.
  - The Tags box then showed a second, empty chip ("×" with no text). The hidden value was `"Hip,   "`.
  - The blank tag was not saved: the POST stored `tags: ["Hip"]`. So it is only a confusing empty chip until you submit.
  - This is probably Mantine TagsInput accepting the typed text on blur.
  - Screenshot: `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-n6rM6I/screenshot-1792953958291-3.png`.
- Enter in an empty Tags box no longer submits in the **Video** form either (I checked it while adding "A hip video 1850", as well as in T-15's leaflet form).
- Round 1's admin top-bar clipping is fixed. The bar scrolls inside its own box (358px box, 562px contents). **PLATFORM ADMIN**, **Trusts**, **Content library** and **Sign out** read in full when scrolled to, and the page itself stays 390px wide. The hospital top bar also scrolls on its own.

## Console and network errors not tied to a case

- No console errors or warnings.
- **Tool artefact, not an app bug:** the network log reported every PATCH (Retire / Bring back) as **503**, but the UI showed success and the item's status changed. The same PATCH sent with `fetch` from the page returned **204**. Treat the 503s as a recording artefact of the extension.
- The only real 4xx was the expected 400 in T-15 step 2.

## GIFs

- `~/Downloads/S33-T02-retest.gif`: T-02 happy path (add a platform video with tags; Recovery gets ticked).
- `~/Downloads/S33-T15.gif`: T-15 (Enter in an empty Tags box, switch to Video and back, Enter in Title adds the leaflet, retire). It also shows the empty tag chip.
