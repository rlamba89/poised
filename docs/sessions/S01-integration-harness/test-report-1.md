# S01 test report 1

**Summary: 19 passed, 0 failed, 0 blocked** (15 test cases + 4 regression cases). Exploring turned up 1 minor UI bug and 1 minor API quirk, both older than S01 (details below). S01 changed no UI code, and its list change only caps the page number.

**Environment**
- Commit `223e69b`. The working tree is **not clean**: 24 uncommitted changes, which are the S01 work under test.
- 8 Oct 2026, 14:14–14:36 BST.
- App http://localhost:3000, API http://localhost:8080 (`/api/health` → `{"status":"ok"}`), started by Rahul with `make dev`.
- Chrome through Claude in Chrome. Desktop cases ran at 1440×757; phone width (390) was only looked at while exploring.
- Users: Alex Author, Val Viewer, Bea Author, Cara Clinician. Hospital A is `00000000-0000-4000-8000-00000000000a` and Hospital B is `…00000000000b`.

**Where I used something other than the UI**
- **T-12 and T-13:** the plan asks for these. I ran T-13's `fetch` through the page's JavaScript context, which is the same as the DevTools console.
- **T-14:** a terminal.
- **To confirm a network-log oddity:** in-page `fetch` and `curl` (see "Console and network").
- **Reading code, without changing it:**
  - checked for native `confirm()` and `beforeunload`, which freeze the browser tool;
  - read `git diff` to tell whether findings are new in S01.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 Sign in, list | Pass | Hospital picker on Hospital A, name and Sign out shown, URL `/h/…0a/questionnaires`. |
| T-02 Create questionnaire | Pass | Empty Save refused by the browser's "Please fill in this field." The editor opened with "v1 · draft". |
| T-03 Duplicate name (capitals) | Pass | "A draft questionnaire with this name already exists." No second row. |
| T-04 Search | Pass | A name fragment, a description word (`pelican`) and clearing the search all behave correctly. |
| T-05 Question Set, page, questions, reload | Pass | Breathing → Lungs, Yes/No + Select Many with 3 options. All kept after reload, including the Question Set description. Renaming with Enter also works. |
| T-06 Deep link and Back | Pass | Back from the list reopens the editor; the deep link in a new tab opens the same set and page. |
| T-07 Saved option lists | Pass | Save, Use (chips, "Alex Author, 08/10/2026"), Add, reload and delete all work. See the note about default options below. |
| T-08 Code search in a disclosure | Pass | `smok` → Smoker 77176002 and Ex-smoker 8517006; `7717` → Smoker; `asthma` → Asthmatic, Mild, Severe and Moderate asthma. 🔗 1 chip still there after reload. |
| T-09 Publish, then new version | Pass | Publish dialog listed no problems. "v1 · published" and "Read only" survive a reload. v2 draft keeps the same question ID (`q_rkdonn`) and is editable. List shows version 2, draft. |
| T-10 Val Viewer read-only | Pass | No New Questionnaire. ⋯ shows only View. Editor shows "Read only": no Add Page, Add Question Set or Add content, fields disabled, no Add on the Disclosures tab. |
| T-11 Bea can't open Hospital A's questionnaire | Pass | Her list is empty. A's URL → "You don't have access to this hospital." B's URL with A's id → API 404 and "Questionnaire not found." No content shown. |
| T-12 Huge page number | Pass | `page=99999999999` → `{"items":[],"page":107374182,"pageSize":20,"total":0}`. `page=abc` and `page=-5` → page 1 with all 3 items. A 23-digit page is capped the same way. See exploring finding 2 about `total`. |
| T-13 Broken chapter PATCH | Pass | `{"name": null}` → `400 {"error":"The request could not be read."}`, not a network error. After reload the set is still `Breathing` and the description is unchanged. Extra bodies were refused correctly too (see below). |
| T-14 `make test-integration` | Pass | Exit 0, no `FAIL`. Last 8 lines below. |
| T-15 Clean up | Pass | `S01 test 14:15` deleted. `S01 publish 14:25` refused with "Only drafts can be deleted. Published versions can be retired instead." |
| R-01 Preview | Pass | Answering Yes to "Do you smoke?" shows SNOMED 77176002 Smoker, Lifestyle, "Current smoker". |
| R-02 HJE Full HQ | Pass | Opened About you → Patient Information, then Medication and Allergies → Prescribed medication. Cards render, no console errors, and only GETs were sent (nothing saved). |
| R-03 Cara Clinician | Pass | Lands on Episodes. There are no episodes yet, so there was none to open. |
| R-04 Two tabs | Pass | Tab 1 saved "Tab one question". Tab 2's edit showed "Someone else changed this Question Set" with Reload and "Not saved". After Reload, tab 1's change is there. |

**T-13: extra request bodies I tried.** All were refused correctly or harmless, and the name and description stayed the same.
- `{}` → 204, with no change.
- `{"name": ""}` and `{"name": "   "}` → 400 "Enter a name for the chapter."
- `{"name": 123}`, `{"description": null}`, `not json` and `{"name": null, "description": "x"}` → 400 "The request could not be read."

**T-14, last 8 lines:**
```
?   	github.com/rlamba89/poised/apps/api/cmd/seed	[no test files]
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	1.105s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	1.158s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	0.325s
?   	github.com/rlamba89/poised/apps/api/internal/db	[no test files]
ok  	github.com/rlamba89/poised/apps/api/internal/episode	1.386s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	3.392s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	0.909s
```

My first run failed straight away with `make: docker: No such file or directory`, because my shell didn't have Docker Desktop's CLI (`/Applications/Docker.app/Contents/Resources/bin`) on its `PATH`. That's my environment, not the project. The run above added it to `PATH` and passed. Rahul's terminal clearly has `docker`, since `docker compose up -d db` worked for him.

## Failures

None.

## Found while exploring

None of these are new in S01: S01 changed no `apps/web` files, and the list query is unchanged.

1. **Minor: the delete dialog shows the previous error.**
   - **Steps:** as Alex Author, ⋯ → Delete on a questionnaire that has a published version (`S01 publish 14:25`). Click Delete; it's refused with "Only drafts can be deleted. Published versions can be retired instead." Click Cancel. Then ⋯ → Delete on a different questionnaire that is only a draft (`S01 emoji 🫁 14:40`).
   - **Expected:** a clean "Delete questionnaire?" dialog.
   - **Actual:** the red "Only drafts can be deleted…" message is already showing, above text that says this questionnaire will be deleted. Clicking Delete does delete it, so the message is stale and misleading.
   - **Where:** URL `/h/…0a/questionnaires`, signed in as Alex Author.
   - **Screenshot:** `~/Downloads/S01-X01-stale-delete-error.png`. The GIF is `S01-T15-cleanup-stale-delete-error.gif`.
2. **Minor (API): `total` is 0 on any page past the last.**
   - `GET …/questionnaires?page=2` with 3 questionnaires returns `"total":0`; `page=1` returns `"total":3`.
   - Every item also carries a stray `"total":3` field.
   - **Cause:** both come from `count(*) OVER () AS total` in the list query, which counts only the rows actually returned on that page.
   - **Effect:** none in the UI today with fewer than 20 rows. A pager that relies on `total` would show 0 results on an out-of-range page.
3. **Minor (keyboard): Enter doesn't submit "Save these options as a list".** Typing a name and pressing Enter does nothing; you have to click Save list. Elsewhere Enter does submit, for example renaming a Question Set or a page.
4. **Cosmetic: the access-error pages are dead ends.** "You don't have access to this hospital." and "Questionnaire not found." show alone on a blank page, with no header, no Sign out and no link back. To sign out I had to go to `/` by hand.
5. **Cosmetic: "chapter" appears in text users see.**
   - The conflict banner's body reads "Someone else changed this **chapter** since you opened it…", while its title says "Question Set".
   - The API error "Enter a name for the **chapter**." also uses it, but the UI doesn't seem to show that one: a blank rename just keeps the old name.
6. **Cosmetic: narrow screens.** The State badge shrinks to "D…" at about 760 px wide and at phone width. At phone width the header wraps, and "Alex Author / Sign out" crowds under the hospital picker. Authoring is desktop-first, so this is low priority.
7. **Cosmetic:** Val Viewer's ⋯ → **View** item uses the pencil (edit) icon.
8. **Note, a product question:**
   - Deleting a saved option list happens on one click of the trash icon, with no confirmation and no undo. Deleting a questionnaire asks first.
   - A new Select One starts with placeholder options Option 1–3, so "Add to the current options" leaves them in place and the question ends up with 6 options. That's as the plan says, but most authors will want "Replace".
9. **Note:** Question Set names have no length limit. 303 characters were accepted, and the tree and header wrap them cleanly.
10. **Low confidence, dev mode only: some clicks right after a full page load were lost.**
    - Twice, Sign out clicked 2–3 s after a full page load did nothing, and no `/api/logout` request was sent. Clicking again a few seconds later worked.
    - Once, Select One clicked just after a dialog closed did nothing.
    - This looks like slow hydration under `next dev`. Worth watching for when the Playwright tests (S03) land.
11. **Low confidence:** the ⋯ button is a small target and the row itself can be clicked, so one click that landed just beside ⋯ opened the editor instead of the menu.
12. **Low:** the New Questionnaire drawer sometimes opened about 1 s late. In that window, my click landed on a row's ⋯ behind it, and that row's Edit/Delete menu then appeared on top of the open drawer. Escape closed it.

**Checked and fine:**
- A name of spaces only is refused by the server ("Enter a name for the questionnaire.").
- Leading and trailing spaces are trimmed, and emoji names work.
- Double-clicking Save creates only one questionnaire (one POST).
- Renaming a Question Set to blank keeps the old name.
- Cara Clinician gets a read-only view of questionnaires.
- Cara's and Bea's lists show only their own hospital.

**Not tested:** reloading in the middle of an unsaved edit. The editor sets a `beforeunload` "unsaved changes" prompt, and native dialogs freeze the browser tool. Rahul could try this by hand: type in a question text box, then press ⌘R before clicking out.

## Console and network

- **Console:** no errors or warnings in any case I checked (T-01, T-05, T-09, R-02, R-03, R-04, and the Bea and Cara pages).
- **Network-log artifact, not an app bug:** the Claude in Chrome network log reports every `204 No Content` as **503**: option-list DELETE, `POST …/publish` and `POST /api/logout`. I confirmed the real status is 204 in two ways:
  - an in-page `fetch` DELETE through the Next.js proxy returned `204 No Content`, while the log for that same request said 503;
  - `curl -X DELETE` through `:3000` and directly to `:8080` both returned `HTTP/1.1 204 No Content`.

  Non-204 errors are reported correctly, for example a 404 for an unknown list id. Future testers should ignore 503s on 204 routes in that log.
- **Duplicate GETs:** most GETs appear twice (for example `/api/me` and `questionnaires?q=&page=1`). This is expected from React Strict Mode in development.

## Test data

**Created:**
- Questionnaire `S01 test 14:15` (Hospital A).
- Questionnaire `S01 publish 14:25` (v1 published, v2 draft).
- Questionnaire `S01 emoji 🫁 14:40` (exploring).
- Option lists `S01 list 14:15` and `S01 list2 14:15` (UI).
- Option list `S01 js 14:15` (in-page fetch).
- Option lists `S01 curl direct` and `S01 curl proxy` (curl, used to check the 204/503 question).

**Deleted:** `S01 test 14:15`, `S01 emoji 🫁 14:40`, and all five option lists. Hospital A's saved option lists are empty again.

**Left:** `S01 publish 14:25` (v2 draft over a published v1). It can't be deleted, as expected. **HJE Full HQ** was not changed.

## GIFs and screenshots

All are in `~/Downloads/`:
- `S01-T02-T08-happy-path.gif`: create, Question Set, page, questions, reload, saved list, disclosure code search.
- `S01-R04-two-tab-conflict.gif`: the two-tab conflict banner and Reload.
- `S01-T15-cleanup-stale-delete-error.gif`: clean-up, including exploring finding 1.
- `S01-X01-stale-delete-error.png`: still of exploring finding 1.
