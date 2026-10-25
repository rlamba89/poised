# S33 build notes: the content library

Builder, automatic run, 25 Oct 2026.

## Answers used

[answers.md](answers.md) (Rahul, 25 Oct), all as recommended: (1) a few made-up example videos pointing to one public test video; (2) a leaflet is a link to a PDF on the web until C8; (3) free tags, with PX-13's checklist topics as ready-made tags and the page showing which topics have nothing yet; (4) a simple superadmin list of the starter questions where one can be retired.

## What was built

### Go

- **Migration 00029** (`content_library.sql`): `content_items` (id, `org_id` NULL = Preopsis's, kind `video|leaflet`, title, url (`https://` only), `length_seconds` (a video has one, a leaflet none: a CHECK), tags, status `active|retired`, created/updated by and at). And `library_questions.retired` for the starter questions.
- **`internal/content`** (new package, unit-tested): `CheckVideo` (YouTube watch / `youtu.be` / embed / `youtube-nocookie.com` embed; Vimeo `vimeo.com/<id>`, the private link `vimeo.com/<id>/<hash>`, `player.vimeo.com/video/<id>[?h=<hash>]`; https only, no user name or port), `CheckLeaflet` (any https link with a host), `Tags` (trim, drop repeats in any case, at most 10 of 40 characters), and `Checklist` / `Covered` (PX-13's 11 topics: Appointment details, Fasting, Medicines, What to bring, Screening, Prehab, Stopping smoking, Consent, The anaesthetic, Recovery, Who to call).
  - **The video link cases are a file**, `internal/content/testdata/video_links.json` (accepted links with what they play, and refused ones). S34's TypeScript check should run the same file in Vitest: that's the test that keeps the two lists in step.
- **Routes** (`internal/httpapi/content.go`, `router.go`), not audited (no patient data):
  - `GET/POST /api/admin/content`, `PATCH /api/admin/content/{cid}`: Preopsis's items, platform admins only. The GET lists retired ones too, last, with the checklist.
  - `GET /api/admin/library-questions`, `PATCH /api/admin/library-questions/{lid}` `{retired}`: the starter questions (answer 4).
  - `GET/POST /api/o/{oid}/content`, `PATCH /api/o/{oid}/content/{cid}`: the trust's own (retired included) plus Preopsis's **active** items, with the checklist; read by whole-trust super clinicians and admins, changed by whole-trust admins only. A trust admin's PATCH of a Preopsis item is a 403.
  - `GET /api/o/{oid}/content/search?tag=&q=`: active items of Preopsis's and the trust's; `tag` a whole tag (any case), `q` words in the title or a tag. Whole-trust super clinicians and admins (the authors).
- **Retired starter questions:** `ListLibraryQuestions` takes `with_retired`. Insert from library (and its search) and "Same as…"'s candidates skip retired ones; "Same as…"'s `linkedTo`, the save-to-library key check, and cross-trust pre-fill's platform keys (`ListPlatformLibraryElements`, unchanged) still count them, so questions already inserted keep working exactly as before.
- **Seed:** three example videos (fixed ids `…0501`–`…0503`), all "Example: …", Preopsis's, pointing to the Blender Foundation's "Big Buck Bunny" on YouTube (`aqz-KE-bpKQ`, checked to exist, 10:35), tagged with checklist topics (The anaesthetic, Fasting, Prehab) and "Example". Each run puts them back as seeded. `seedLibrary` doesn't touch `retired`.

### `apps/web` (TypeScript, for Rahul to review)

- `features/content/content.ts`: the types, and small helpers: `formatLength` (seconds → "3:20" / "1:02:05"), `parseLength` ("3:20", "1:02:05" or whole minutes → seconds; null if unreadable, so Go gives its message), `addTag`, the form ↔ request body (`bodyOf`, `formOf`) and `changedFields` (an edit sends only what changed). Unit-tested.
- `features/content/ContentLibrary.tsx`: one page component for both screens: **Add video / Add leaflet** (inline form, not a modal), the checklist card, the items table (Edit, Retire / Bring back for those who may change), Preopsis's items read-only on the trust's page, and on the admin page the **Starter questions** list (Retire / Offer again).
- Pages `pages/admin/Content.tsx` (`/admin/content`) and `pages/hospital/Content.tsx` (`…/content`); routes and `routes.test.ts`; the top bars: **Content library** in the console (Trusts is no longer highlighted on it) and in the hospital layout for whole-trust super clinicians and admins (`canSeeContent`; `canManageContent` = whole-trust admins); `switchPath` keeps the page when switching hospital.

### End-to-end

- `e2e/journeys/test_content_library.py`: Sam adds a platform video (a Dailymotion link refused in plain words), Ada adds a trust video and leaflet and edits one, the checklist ticks "Who to call", Bea never sees Trust A's, Alex's search finds the platform's and the trust's by tag, Cara has no library and its URL refuses her, retiring takes items out of the search; and Sam retires a starter question ("Medicines you take regularly"), Insert from library's API stops offering it, then he offers it again.

## Decided by the builder (automatic mode)

- **A video's length is required** (1 second to 4 hours), so S36's estimated time always has it. It's typed as `m:ss`, `h:mm:ss` or whole minutes.
- **A leaflet's link is any https link**, not only one ending `.pdf` (trusts often serve PDFs from links without it).
- **Retiring is reversible** for content items (**Bring back**) and starter questions (**Offer again**): a PATCH of the status, as Procedures do. Nothing is ever deleted.
- **An item's kind can't change** after it's added (the link check differs); a PATCH ignores `kind`.
- **Whole-trust super clinicians see the trust's library page read-only** (the brief: "may read them"); clinicians and hospital-only super clinicians don't see it.
- **A retired Preopsis item isn't shown on trusts' pages** (only the admin's).
- **A retired starter question stays retired through `make seed`** (and the pipeline's seed on QA): it's a platform admin's decision, not test data. The example videos, being made-up demo data, are put back on each seed.
- **The tag box has no suggestion dropdown;** the checklist topics are buttons under it ("+ Fasting"), because the dropdown covered the form's buttons (and testers' hidden tabs can't open dropdowns).
- **No uniqueness on titles** (two trusts, or Preopsis and a trust, may both have "Fasting").

## Tests (test-first: each new test failed first, then passed)

### Go
- `internal/content/content_test.go`: `TestCheckVideo` (the cases file), `TestCheckLeaflet`, `TestChecklist`, `TestTags`.
- `internal/httpapi/content_integration_test.go`: `TestAdminContent`, `TestTrustContent`, `TestSearchContent`, `TestAdminStarterQuestions`, each with the five cases (happy path, no sign-in, wrong role, another tenant's data, invalid input), asserting responses and rows.
- `internal/httpapi/library_integration_test.go`: `TestRetiredStarterQuestion` (Insert from library and "Same as…" skip it; `linkedTo` and the key check still count it).
- `cmd/seed/seed_integration_test.go`: the examples come back as seeded, a retired starter question stays retired, and the second run leaves the same rows.

### TypeScript (Vitest)
- `features/content/content.test.ts`; `lib/auth.test.ts` (who sees and changes the library; switching hospital); `routes.test.ts` (both new pages, in their layouts).

### Results (25 Oct)
- `make lint`: passed.
- `make test`: Go all ok; web 24 files, 256 tests passed; clinical 19 files, 182 passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (all journeys, Chromium, against a dev stack on spare ports): **75 passed, 5 skipped** (77 after test report 1's fixes) on a fresh database and again on the shared `sj` database (decisions §4: run it on the shared one too).

## Hand checks for Rahul

None expected: everything in this session can be done in the browser. The example videos' link was checked to exist (YouTube's oEmbed answered "Big Buck Bunny 60fps 4K - Official Blender Foundation Short Film").

## Suggested commits

1. `S33: the content library's rules in Go (internal/content)` — `internal/content/*` with its tests and cases file.
2. `S33: content_items, the routes and the starter questions' retire` — migration 00029, `db/queries/content.sql`, `library.sql`, generated `internal/db`, `httpapi/content.go`, `router.go`, `library.go`, `question_keys.go`, their integration tests.
3. `S33: seed the example videos` — `cmd/seed/main.go` and its test.
4. `S33: the Content library pages` — `apps/web` (features/content, pages, routes, layouts, auth) with their Vitest tests.
5. `S33: e2e journey and docs` — `e2e/journeys/test_content_library.py`, this folder, the docs.

## Test rounds

- Test plan written 25 Oct ([test-plan.md](test-plan.md)).
- **Test report 1** (25 Oct): 18 passed, 1 failed.
  - **T-14 (fixed, test-first):** at 390px the tags column shrank to one letter per tag, and the console's top bar clipped its labels. The items and starter-question tables now scroll sideways in their own box (`Table.ScrollContainer`, wide enough for whole tags), the Retired badge no longer shrinks, and `AdminLayout`'s top bar scrolls sideways like the hospital one. New journey `test_at_phone_width_tags_and_the_top_bar_can_be_read` (failed first: "the tag 'Who to call' is cut short", then the Retired badge).
  - **Exploring, Enter in an empty Tags box sent the form: fixed.** The hint says "Press Enter after each", so a second Enter is a natural slip that could add an unfinished item. Enter with nothing typed in Tags now does nothing; Enter in any other field still sends the form. Covered by `test_enter_in_an_empty_tags_box_doesnt_send_the_form` (failed first: the form had closed).
  - **Exploring, a video's error stayed after switching to Leaflet: fixed** (switching the kind clears the message; same journey).
  - **Exploring, the PLATFORM ADMIN badge on the refusal page and Sam on a hospital URL:** S08/S10 behaviour, out of scope; no change. **Alex seeing the trust's retired items** (greyed, no buttons) is intended: readers see the same list as admins.
  - After the fixes: `make lint`, `make test`, `make test-integration` pass; all e2e journeys **77 passed, 5 skipped** on the shared `sj` database.
- **Test report 2** (25 Oct, retest): 4 passed, T-14 failed.
  - **T-14 (fixed, test-first):** in Sam's Starter questions table at 390px, "Measurements" was still cut to "MEASUREME…". The cause: Mantine's Badge hides overflow, so a narrow column could squeeze any badge. Every badge on both screens (tags, Retired, checklist topics) is now a `WholeBadge` (overflow visible, no shrinking), so the column grows to fit it instead. The journey now checks **every** badge and button label on both pages (with the add form open) at 390, 768 and 1440px, and that neither page scrolls sideways (`test_at_phone_width_every_tag_badge_and_label_reads_in_full`; it failed first on Sam's two "Measurements").
  - **Exploring, spaces then leaving the Tags box made an empty chip** (not saved): **fixed**. The box's tags are cleaned on every change (`cleanTags`: trimmed, no blanks, no repeats in any case; unit-tested), and the Enter journey checks only "Hip" remains after leaving the box (failed first: two chips).
  - After the fixes: `make lint`, `make test` (web 257), `make test-integration` pass; all e2e journeys **77 passed, 5 skipped** on the shared `sj` database.
- **Test report 3** (25 Oct, retest): **all passed** (6 of 6). The 768px observation (Retire reached by scrolling the table sideways) is recorded in pending.md, no code change, as the conductor asked. **S33 tested.**

## Verification 1 fixes (25 Oct)

[verify.md](verify.md): Needs fixes. Each fix was test-first (the new test failed, then passed).

1. **Must: `HTTPS://` / `Https://` gave a 500.** `url.Parse` lowercases the scheme, so the check passed, but the table's case-sensitive `CHECK (url LIKE 'https://%')` refused the raw link. Now Go stores the link as `content.Link` gives it: trimmed, scheme and host in lower case (path, query and fragment unchanged), for videos and leaflets, on add and edit. Tests: `TestLink` and new leaflet cases in `TestCheckLeaflet`; three capitalised links added to `video_links.json` (accepted) and `HTTP://…` (refused); integration "a capital in the link's scheme or host … is saved in lower case" asserts the stored rows. The web's link field is now `type="url"` with `autoCapitalize="none"` (a phone's URL keyboard, no capital).
2. **Should: a failed Retire's error came back after Edit → Cancel.** Opening a form now clears the page's error. Journey `test_a_capitalised_link_is_saved_and_an_old_error_doesnt_come_back` (it makes the Retire fail with a stubbed 409; I checked it fails without the fix).
3. **Minor: a POST with `status: "retired"` made an active item.** Now refused (400, "A new item is always active."); integration invalid-input case.
4. **Note for S34** (search refuses a hospital-only super clinician): recorded in pending.md.
5. **Notes:** the duplicate tag rule is gone (`library.go` uses `content.Tags`, same limits and messages; the library tests pass unchanged). Left as they are, recorded in pending.md: the row → item mapping written three times, the `canSeeContent`/`canManageContent` aliases, "Same as…" walking `keySources` twice, and the lock taken before reading the body.

After the fixes: `make lint`, `make test`, `make test-integration` pass; all e2e journeys **78 passed, 5 skipped** on the shared `sj` database (on a spare-port stack, stopped afterwards).

- **Test report 4** (25 Oct, the last retest after verification 1): **all passed** (9 of 9). Its exploring note (a failed Retire low on a long list shows its error off-screen) is in pending.md, no code change, as the conductor asked. **S33 tested.**
