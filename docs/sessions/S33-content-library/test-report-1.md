# S33 test report 1

**18 passed, 1 failed, 0 blocked.**

- Environment: commit `fdc963b` plus the uncommitted S33 work (working tree not clean: the S33 files listed in `git status`), Sun 25 Oct 2026 18:09–18:35 GMT, http://localhost:3000 (API :8080, run by the conductor's `make dev`), Chrome ("trk" profile) through Claude in Chrome. Automatic mode.
- Phone size: the window can't be resized, so 390px was tested with a same-origin 390×720 iframe of the page.
- Test data made: platform items "Knee video 1810" (retired), "Fasting leaflet 1822" (active, Fasting), "Good … 1815" ×5 (accepted-link checks, all retired); Trust A "A hip video 1830", "A who-to-call leaflet 1830" (active), "Consent leaflet 1835", "🦵 Emoji leaflet 1845", "Dbl click video 1852", "len leaflet" (all retired); Trust B "B video 1840" (Hip, active); questionnaire "MT S33 1900 library" (draft); library question "MT S33 1905 own question"; procedure "MT S33 1910 procedure" (retired). "Medicines you take regularly" was retired and then **offered again**: it is offered at the end of the round.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Open the page | Pass | `/admin/content`, title, Content library highlighted; Trusts → Back → back on the library with it highlighted; deep-link reload works. |
| T-02 Add a platform video with tags | Pass | "Knee video 1810 is added."; Video, 3:20, KNEE, RECOVERY; link opens in a new tab (`target=_blank`, `noopener`); Recovery went from "nothing yet" to covered (8 → 7 of 11). |
| T-03 Only YouTube/Vimeo | Pass | All four bad links: the plain red message, each POST 400, nothing added. All six accepted forms added (youtu.be in T-02; watch, embed, vimeo, private vimeo, player.vimeo here). |
| T-04 Other refusals | Pass | "Enter a title."; empty / `0:00` / `3:75` / `soon` → "Enter the video's length."; `300` → "…between 1 second and 4 hours."; `leaflet.pdf` and `http://…` → "Enter the leaflet's web address, starting https://". All 400, nothing added; Cancel closes. |
| T-05 Leaflet; edit; retire; bring back | Pass | Leaflet has no Length field, shows "—". Edit showed 3:20; 4:05 and Knee removed, saved with Enter in Length. Retire → message, last row, greyed, Retired badge, Bring back; Recovery back to "nothing yet". Bring back → "offered again". Retired again. |
| T-06 Trust A's library | Pass | `/o/…/h/…/content`, Trust A wording, "(Preopsis's items count too)", Trust A's items, "From Preopsis (read-only)" without buttons; Sam's retired video not listed. Switching to Hospital A2 keeps the page; reload keeps it. |
| T-07 Trust A video and leaflet | Pass | Private Vimeo link, 12:05, Hip; who-to-call leaflet; Who to call ticked; edit to Recovery + 12:30 saved; `film.mp4` on Edit refused with the T-03 message. |
| T-08 Checklist count | Pass | Consent "nothing yet", 6 of 11 → leaflet tagged by typing `consent` → covered, 5 of 11 → retired → nothing yet, 6 of 11. |
| T-09 Other trust never sees Trust A's | Pass | Bea: Preopsis's items only, no Add buttons, "Only the trust's admins can add or change items." Max added "B video 1840"; not on Trust A's page. |
| T-10 Who may do what | Pass | Alex reads, no buttons, the line shown; Cara and Sue: no menu item, URL gives "Only super clinicians and admins of the whole trust can see the content library"; Ada on `/admin/content`: "Only Preopsis's platform admins can open this page". API: Cara 403 ×4; Alex 200/403/200/403; signed out 401 ×4 (plus `/api/admin/library-questions` 401); Bea → Trust A's list 404. |
| T-11 Search | Pass | `tag=Hip` → only "A hip video 1830" (not Max's Hip video, not retired E2E Hip video); `tag=fasting` and `tag=FASTING` → the example + "Fasting leaflet 1822"; `q=example` → the three examples; `""` → 6 active items of Preopsis's and Trust A, none of Trust B's, none retired. Bea `tag=Hip` → `[]`. |
| T-12 Starter questions list | Pass | 11 questions, text, ID and tags, each with Retire; no trust questions. |
| T-13 Retire a starter question | Pass | Inserted (ID `regular_medicines`) and saved; Sam's Retire gave the exact message, row last, Retired, Offer again; draft still has it with the same ID; From library "medicines" → "No library questions match.", others still offered; Offer again → "… is offered again." and From library offers it again. Left offered. |
| T-14 Phone size | **Fail** | Form fields stack and the hospital top bar scrolls sideways, but at 390px the table's **Tags column shrinks to ~33px and every tag (and the Retired badge) is cut to one letter** ("H.", "R.", "W"), even after scrolling sideways. Sam's admin top bar doesn't scroll: its labels are clipped ("PLATFO…", "Tru", "Content"). See below. |
| R-01 (S10) Trusts | Pass | List, Trust A with Hospitals A1/A2, New trust form opens. |
| R-02 (S19) Procedures | Pass | Alex added and retired "MT S33 1910 procedure". |
| R-03 (S24) From library / Save to library | Pass | "Do you smoke?" inserted with ID `smoker`; own question saved ("Saved. The trust's authors can now insert it…") and listed in From library with a **THIS TRUST** badge (the plan says "under 'Trust A'"; the badge is S24's wording). |
| R-04 (S24) Same as… | Pass | Search "smoke" offers "Do you smoke?" · ID `smoker` (the starter question). |
| R-05 (S32) Clinical Summary | Pass | Added from the Add bar: "Page 1 summary", Clinician badge, Clinical comments box. |

## Failures

### T-14: at phone width the items table's tags are unreadable, and the admin top bar clips its labels (Minor)

- **Who / where:** Ada Admin on `/o/00000000-0000-4000-8000-0000000000fa/h/00000000-0000-4000-8000-00000000000a/content`; Sam Superadmin on `/admin/content`. Viewport 390px wide (same-origin iframe).
- **Steps:** open either page at 390px wide; scroll to the items table (and on Sam's page also the Starter questions table); scroll sideways.
- **Expected (plan):** "the tables can be read (scrolling sideways is fine)" and "the top bar scrolls sideways".
- **Actual:**
  - The page does scroll sideways (document 503px wide for Ada, 520px for Sam), but the Tags column is squeezed to about 33px. Each tag badge is 33px wide and truncated: HIP → "H.", RECOVERY → "R.", WHO TO CALL → "W", CONSENT → "C.", KNEE → "K.". The Retired badge beside a title is cut to "R.." or shows empty. The width doesn't grow when scrolling sideways, so tags can't be read at all on a phone. Titles and links also wrap to a few characters per line.
  - Sam's console top bar (`AdminLayout`) doesn't scroll on its own like the hospital one (`overflow-x: visible` there vs `auto` in the hospital layout). The PLATFORM ADMIN badge shows "PLATFO…", the nav labels are clipped to "Tru" and "Content", and Sign out sits beyond the right edge (x 407–456), reachable only by scrolling the whole page sideways. With only "Trusts" before S33 it fit, so adding **Content library** is what tips it over.
- **No console errors.** Screenshots: `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-n6rM6I/screenshot-1792952308462-0.png` (Ada's table), `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-n6rM6I/screenshot-1792952531512-1.png` (Sam's top bar). GIF: `~/Downloads/S33-T14.gif`.

## Found while exploring

- **Enter in an empty Tags box submits the form.** The hint says "Press Enter after each", so typing a tag, Enter, then Enter again (or Enter on spaces only) submits the whole form. With a complete form it adds the item straight away (it happened to me with "🦵 Emoji leaflet 1845"); with an incomplete one the error shows. Enter on a duplicate tag ("knee" after "Knee") is ignored correctly and doesn't submit. Possibly intended (Enter submits from any field, T-05 relies on it from Length); worth a look.
- **A stale error stays when switching Video → Leaflet** in the Add form: "The video's length must be between 1 second and 4 hours." stays under the leaflet form until the next submit. Cosmetic.
- **Ada on `/admin/content` (and `/admin`) sees the PLATFORM ADMIN badge** in the top bar of the refusal page. It's the same on `/admin` (S10), so not new. Cosmetic.
- **Sam on a hospital URL** gets "You don't have access to this hospital" with no top bar or Sign out on that page (S08 behaviour, not new).
- Checks that behaved well:
  - Title and link with leading and trailing spaces are trimmed.
  - Emoji in a title works.
  - Duplicate tags in a different case are dropped.
  - Double-clicking **Add video** adds the item once.
  - API refusals give plain messages:
    - a 1000-character title → "The title is too long (200 characters at most).";
    - 11 tags → "Too many tags (10 at most).";
    - a 41-character tag → "A tag is too long (40 characters at most).";
    - kind `audio` → "Choose a video or a leaflet.";
    - `javascript:` link and `https://user:pw@…` link → the leaflet message.
  - A leaflet POSTed with `lengthSeconds` is stored with none.
  - Max (Trust B admin) PATCHing a Preopsis item through Trust B's path → 403 "Preopsis's items can't be changed by a trust."
  - Ada PATCHing Trust B's item through Trust A's path → 404.
  - Max switching to Hospital A2 (clinician there) from the content page lands on Episodes, not the library.
- Alex (read-only) sees Trust A's **retired** items in the list too (greyed, no buttons). That seems right for a reader; mentioned only because the plan doesn't say.

## Console and network errors not tied to a case

None. No console errors or warnings on any page visited. The only 4xx responses were the ones the cases provoked.

## Environment notes (not app bugs)

- The 1Password extension sometimes put its overlay on a form field (New questionnaire, Add procedure), and while it was there the browser tools couldn't type or click. I filled those two forms with `form_input` and a script click instead.
- Mantine modals open and close slowly in the tool's background tab, so some clicks landed during the animation and were repeated.

## GIFs

- `~/Downloads/S33-T02-happy-path.gif`: T-01 → T-02 (open the library, add a platform video with tags, checklist ticks Recovery).
- `~/Downloads/S33-T13-starter-question.gif`: Sam retires "Medicines you take regularly", then offers it again.
- `~/Downloads/S33-T14.gif`: the 390px table with truncated tags (one frame).
