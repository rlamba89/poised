# S33 verification

## Verification 2 (25 Oct 2026, verifier, automatic run)

**Verdict: Verified.**

Verification 1's three fixes are in and hold, in the code, the tests and the browser. A second `/code-review high` found nothing that breaks the brief; its useful points are minor and are in pending.md ("From S33's verification 2").

### Verification 1's fixes, checked

1. **A capitalised link is saved, lower-cased (was a 500).** `content.Link` (`internal/content/content.go`) returns the trimmed link with its scheme and host in lower case, and `contentChange.apply` (`httpapi/content.go`) stores that for both kinds, on add and edit. `TestLink`, the new `video_links.json` cases and the integration case cover it.
   - Checked in Chrome through the API, as Sam:
     - `HTTPS://Example.ORG/Path/A.pdf` (leaflet) was saved as `https://example.org/Path/A.pdf`;
     - `Https://YOUTU.BE/…` (video) was added;
     - a PATCH to `HTTPS://Vimeo.com/76979871` was saved as `https://vimeo.com/76979871`.
     - `HTTPS://dailymotion.com/x` and `HTTP://example.org/a.pdf` got the plain 400 messages.
   - As Ada, in the UI: Add leaflet with `HTTPS://Example.ORG/UI.pdf` showed "… is added." and stored `https://example.org/UI.pdf`. The link field is `type="url"` with `autocapitalize="none"`.
2. **A stale error doesn't come back.** As Ada, a Retire made to fail (a one-shot 409) showed its red message; Edit hid it, and Cancel didn't bring it back. The row was unchanged.
3. **A new item can't be added retired:** `POST` with `status: "retired"` gets 400 "A new item is always active.", and nothing is added.

### Re-checked

- **Tenancy and roles (Chrome, through the API):**
  - Max (Trust B admin) doesn't see Ada's new Trust A item in Trust B's list, and his Trust B search for its tag is empty. He does see Preopsis's items.
  - Max's PATCH of it through Trust B's URL got 404; through Trust A's URL, 403. `/api/admin/content` got 403.
  - Ada's PATCH of a Preopsis item got 403.
  - Cara got 403 on list, search and add; Alex got 403 on add, and his tag search (in another case) found Trust A's item.
- **Scope, tests first, tenancy, security and A-20:** as in verification 1. Nothing in the new files matches the A-20 grep; `repocheck` passes.

### Findings (none blocking; in pending.md)

1. **For Rahul: a retired starter question's key is still offered by "Same as…" through a trust questionnaire that already holds a copy** (`httpapi/question_keys.go:199`). Only the Library source drops it. This fits decisions.md (25 Oct: "questions already inserted keep working"): the copy is the trust's own question. If retiring should stop the key being adopted anywhere, every source would have to filter it.
2. **The checklist's "+ topic" buttons ignore the 10-tag limit** (`features/content/ContentLibrary.tsx:311`): Go then refuses with "Too many tags (10 at most)." on Add. Cosmetic.
3. **An unreadable length ("1:2:3", "0:00") gets "Enter the video's length."** (`features/content/content.ts:47`) rather than a format hint.
4. **A trust's PATCH of a Preopsis item takes the row lock before its 403** (`httpapi/content.go:232`). Harmless at today's scale, like verification 1's note about the lock and the body.
5. Verification 1's notes (the seed's examples come back active; the three copies of the row mapping; `keySources` twice) stand, already in pending.md.

### Test results (25 Oct)

- `make lint`: passed.
- `make test`:
  - Go: every package ok;
  - web: 24 files, 257 tests passed;
  - clinical: 19 files, 182 passed;
  - CDK: 124 passed.
- `make test-integration`: every package ok (`internal/httpapi` 21.7s, `cmd/seed` ok).
- `make e2e` (against the conductor's `make dev`, shared database): 78 passed, 5 skipped (8m 2s).

Left behind on the shared database, all retired: "V2 S33 caps leaflet" and "V2 S33 caps video" (Preopsis), and "V2 S33 trust A caps" and "V2 S33 UI caps leaflet" (Trust A).

### Hand checks for Rahul

None.

### Commits (made, local, not pushed)

1. `S33: the content library's rules in Go (internal/content)`
2. `S33: content_items, the routes and the starter questions' retire`
3. `S33: seed the example videos`
4. `S33: the Content library pages`
5. `S33: e2e journey`
6. `docs: S33 verified …`

Next: `/build-session S34`.

## Verification 1 (25 Oct 2026, verifier, automatic run)

**Verdict: Needs fixes.**

Most of the session is sound: the brief is covered, every route has its five integration cases, the tenancy holds, and every automated test passes. One defect is in the session's central rule, the link check. A link that starts `HTTPS://` or `Https://` passes the check but fails the table's CHECK constraint, so the user gets a 500 and "Something went wrong". A phone keyboard produces `Https://` on its own. Fix 1 and 2 test-first; the rest are notes.

### Findings, worst first

1. **Must fix: a link with an upper-case or mixed-case scheme gives a 500 instead of being saved.**
   - **Where:**
     - `apps/api/internal/content/content.go:168-169` (`httpsURL`): `url.Parse` lowercases the scheme, so `u.Scheme != "https"` passes for `HTTPS://…` and `Https://…`.
     - `apps/api/internal/httpapi/content.go:186`: the raw link is stored only trimmed, not normalised.
     - `apps/api/db/migrations/00029_content_library.sql`: the column's `CHECK (url LIKE 'https://%')` is case-sensitive, so the insert fails. `serverError` then answers 500.
   - **Reproduced twice:**
     - Sam, `POST /api/admin/content` with a leaflet at `HTTPS://example.org/a.pdf`: 500, `{"error":"Something went wrong. Please try again."}`.
     - Sam's Add leaflet form in Chrome, with `HTTPS://example.org/leaflet.pdf`: the red "Something went wrong. Please try again." message.
     - The same happens for a video (`CheckVideo` uses `httpsURL` too) and on an edit (PATCH).
   - **Why it matters:** the link field (`apps/web/src/features/content/ContentLibrary.tsx:287-291`) is a plain `TextInput`. On a phone, the keyboard capitalises the first letter, so a link typed there arrives as `Https://…`. The admin then gets a server error with no hint about what is wrong.
   - **Suggested fix:** store the normalised link, with the scheme and host lower-cased (for example, have the checks return `u.String()`). Add `HTTPS://…` and `Https://…` cases to `testdata/video_links.json` (accepted), add a leaflet case to `TestCheckLeaflet`, and add an integration case that asserts the stored row starts `https://`. On the web, set `type="url"` (or `autoCapitalize="off"` with `inputMode="url"`) on the link field.

2. **Should fix: a stale error comes back after Edit → Cancel.**
   - **Where:** `ContentLibrary.tsx:90-93` (`open`). It clears the notice but not `error`. The page-level error is hidden while a form is open and shows again when it closes.
   - **When it happens:** a failed Retire or Bring back, then Edit on any row, then Cancel. The old red message reappears although nothing new failed.
   - **Fix:** clear `error` in `open` as well. One line; add a Vitest or journey check.

3. **Minor (optional): a POST's `status` is checked but then dropped.**
   - **Where:** `httpapi/content.go:224-229`. `CreateContentItem` has no status, so `{"status":"retired", …}` returns 201 with an active item, and S34's picker offers it at once.
   - The UI never sends a status on add, and the builder already chose to ignore `kind` on a PATCH, so this follows the same pattern. Refusing a non-active status on add would be clearer.

4. **Note for S34 (not a fix here): a super clinician of one hospital only gets a 403 from `GET /api/o/{oid}/content/search`** (`httpapi/content.go:105`).
   - This matches decisions.md (12 Oct, S08 answers 5): only whole-trust super clinicians and admins change the trust's questionnaires, and no hospital-owned questionnaires exist yet.
   - If S34 or a later plan lets a hospital-only super clinician edit their hospital's own questionnaire, the picker's role check has to follow `canChange`.

5. **Notes, no change needed:**
   - **The seed puts the example videos back as active on every run** (`cmd/seed/main.go:305`), even if a platform admin retired one. Starter questions keep their retirement through the seed. The builder recorded this deliberately: the examples are made-up demo data.
   - **`changeContent` takes its `FOR UPDATE` lock before reading the 16 KiB body** (`httpapi/content.go:246-260`). This is harmless at today's scale.
   - **Clean-ups `/code-review` suggested** (optional):
     - `content.Tags` duplicates `httpapi.libraryTags`;
     - the row → `contentItem` mapping is written three times;
     - `canSeeContent`/`canManageContent` could alias `canManageProcedures`/`canManageStaff`;
     - "Same as…" now walks `keySources` twice per search.

### Checked and fine

- **Scope:** all four of Rahul's answers are built:
  - the migration, the Go link check (test-first, with the cases file for S34), and the 3 × 3 routes plus the starter-question routes;
  - both screens, the checklist and the seed.
  - Nothing large was added outside the brief.
- **Tests:**
  - `TestAdminContent`, `TestTrustContent`, `TestSearchContent` and `TestAdminStarterQuestions` each have happy path, no sign-in, wrong role, another tenant and invalid input.
  - `TestRetiredStarterQuestion`, the seed test, the Vitest units and `e2e/journeys/test_content_library.py` are also there.
- **Tenancy:** every trust query has `org_id = $org OR org_id IS NULL` (platform). `GetContentItem` locks only the platform's item or the owner's, and a trust's PATCH of a platform item is a 403.
- **No UK literals added:** the A-20 grep shows only existing comments and tests.
- **Security:**
  - no patient data, so the routes are correctly left unaudited;
  - no secrets;
  - the patient router doesn't mount the routes.
- **Docs:** decisions.md, pending.md, the C5 plan, plans/README and sessions/README are updated. `repocheck` passes.

### Spot-checks in Chrome (trk profile), against `make dev`

1. **Tenancy, as Ada and Max (Trust B admin):**
   - Ada added a Trust A leaflet (201).
   - Max didn't see it in Trust B's list, and his Trust B search for its tag was empty.
   - His PATCH of it through Trust B's URL got 404; through Trust A's URL, 403 (he's a clinician there).
   - Ada's PATCH of a Preopsis item got 403 ("Preopsis's items can't be changed by a trust.").
   - After Ada retired the item, a search for its tag was empty.
2. **Roles:**
   - Cara got 403 on list, search and PATCH.
   - Alex searched by "who to call" (any case): Trust A's and the platform's items came back. His PATCH got 403, and so did `/api/admin/content`.
3. **Starter question:**
   - Sam retired "Medicines you take regularly" (204), and Alex's Insert from library search no longer offered it.
   - Sam pressed Offer again (204), and Alex was offered it again. It's left offered.
4. **The link check in the UI:** Sam's Add leaflet with `HTTPS://example.org/leaflet.pdf` showed "Something went wrong" (finding 1). Cancel closed the form cleanly.

Left behind on the shared database: one retired Trust A leaflet, "Verify S33 tenancy leaflet".

### Test results (25 Oct)

- `make lint`: passed.
- `make test`:
  - Go: every package ok;
  - web: 24 files, 257 tests passed;
  - clinical: 19 files, 182 passed;
  - CDK: 124 passed.
- `make test-integration`: every package ok (`internal/httpapi` 14.7s, `cmd/seed` ok).
- `make e2e` (against `make dev`, shared database): 77 passed, 5 skipped (6m 53s).

### Suggested commits (after the fixes; as in the build notes)

1. `S33: the content library's rules in Go (internal/content)`
2. `S33: content_items, the routes and the starter questions' retire`
3. `S33: seed the example videos`
4. `S33: the Content library pages`
5. `S33: e2e journey`
6. `docs: S33 …` (this folder and the docs; never `docs/market/` or `docs/sessions/run-state.md`)

### For the builder

"Verification found problems: read `docs/sessions/S33-content-library/verify.md`, fix them test-first, then ask me for a retest."
