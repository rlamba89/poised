# S36 build notes: the patient's Information, and the estimated time

Builder, automatic run, 26 Oct 2026. Plan C5's last session.

## Answers used

[answers.md](answers.md) (Rahul, 25 Oct):
1. **(a)** an Information section on the episode's card on the patient's home.
2. **(b), not the brief's recommendation: the procedure's list is copied onto the episode when it's made.** A booked patient keeps the list from when they were booked; changing the procedure's list later changes only new episodes.
3. **(a)** 15 seconds for each question the patient sees, plus each video's length, rounded up to whole minutes (at least 1), **with the seconds per question a setting**: Go's `internal/config` reads `SECONDS_PER_QUESTION` (default 15), and the web app gets it from `GET /api/config`. No number is written anywhere else.

Where answers.md differs from brief.md (manual test focus 3: "existing episodes follow it"), answers.md wins: existing episodes keep their list.

## What was built

### Go

- **Config:** `SECONDS_PER_QUESTION` (1 to 600, default 15, also under `CONFIG_STRICT`: it isn't a per-country value), in `Config.SecondsPerQuestion` and `GET /api/config`'s `secondsPerQuestion`. `.env.example` mentions it.
- **Migration 00030** `episode_info_items`: an episode's own list (`episode_id`, `content_item_id`, `from_procedure`, `position`, `added_by`, `added_at`; one row per item per episode). `procedures.info_content` (S19) holds the procedure's list as a JSON array of content item ids, in order.
- **Procedures** (`procedures.go`): POST and PATCH take `infoContent` (ids, in order; at most 50; no repeats). Each must be Preopsis's or the trust's own content item, and active; an item retired since it was put in the list may stay, so the list can be changed around it. `GET /api/o/{oid}/procedures` adds `infoItems` (each item's id, kind, title and status) next to `infoContent`.
- **Episodes** (`episodes.go`): a new episode with a procedure gets a copy of the procedure's list (`CopyProcedureInfo`), in the same transaction. **Changing the episode's procedure** (Edit procedure and dates) removes the items copied from the old procedure and copies the new one's; items a clinician added stay. The episode update is now in a transaction for this.
- **New routes** (`information.go`), each audited:
  - `GET /api/o/{oid}/h/{hid}/episodes/{eid}/information` (`information.list`): the episode's items in order (with status, where each came from and who added it) and `library`, the content library's active items that can be added.
  - `POST …/information` `{"contentId": id}` (`information.add`): adds one of the library's active items (Preopsis's or the trust's) last. Already there: 409. Cancelled or archived episode: 409. A General notes line, "Information added: <title>".
  - `DELETE …/information/{cid}` (`information.remove`): removes any item (copied or added). "Information removed: <title>".
  - `GET /api/p/episodes/{eid}/information` (`link.information`, the patient's session): the episode's **active** items (id, kind, title, url, length) and the hospital's `phone` and `email` for patients (its effective `contact.phone` and `contact.email` settings) and name. Only the patient's own episodes (404 otherwise), as the form.
  - **Another trust's items never show:** every read joins only Preopsis's items and the episode's own trust's, so even an item that got into a list some other way isn't sent.

### `packages/clinical` (TypeScript, for Rahul to review)

- **`estimate.ts`** (new): `estimate(doc)` counts the questions the patient may see and adds up the videos' lengths; `estimatedMinutes(doc, secondsPerQuestion)` is `ceil((questions × seconds + video seconds) / 60)`, at least 1, or `undefined` when there's no usable seconds per question (the config didn't load: then nothing is shown, rather than a guess); `estimateText(4)` is "About 4 minutes".
  - What counts, as Go's `chapter.ForPatient` strips it: no clinician-only page, Group or question, no Clinical Summary. Information blocks, the profile, calculations, Groups and Sections aren't answered, so they don't count; nor does a question set to `visible: false`. A question shown only under a condition **does** count (the patient might see it; conditions aren't evaluated for an estimate). A repeating group's questions count once. A video adds its length and isn't a question itself, tick or not.

### `apps/web` (TypeScript)

- `lib/config.ts`: `secondsPerQuestion()` from `GET /api/config` (undefined until loaded).
- **The patient's Question Set tiles** (`PatientHQ.tsx`): "About N minutes" under each set's name. The tile's row now wraps on a phone and the status badge and button never shrink, so "Complete" and "Review" are never cut short (the 390px finding noted after S34 and S35, pending.md). The badge is `lib/WholeBadge.tsx`, moved out of the content library page so both use it.
- **The editor** (`Canvas.tsx`, the Question Set header): "· About N minutes for the patient" for a patient set; nothing for a clinician set.
- **Procedures page:** Edit scrolls the form's start into view (it was "nearest", which showed the end of the now taller form). The form has **Information for patients**: the list in order, each with Remove (a retired one marked "Retired"), and a searchable **Add a video or leaflet** picker of the library's active items not yet in the list. The table has an **Information** column ("1 video, 1 leaflet"). `changedFields` now compares lists by their items, so an unchanged list isn't sent.
- **Episode page:** a new **Information for the patient** card (`features/information/EpisodeInformation.tsx`): each item (a link to it, its kind and length, "From the procedure" or "Added for this patient by …", "Retired" when it is), Remove, and **Add a video or leaflet for this patient**. Read-only when cancelled or archived. Reloads when the episode changes (a new procedure brings its list).
- **Patient home:** each card (not cancelled or closed) has an **Information** button below its action row that opens the list (`features/information/PatientInformation.tsx`): each video with its title, length and the player (S34's `videoEmbedFor`: built only from a recognised YouTube or Vimeo id, `youtube-nocookie.com` or Vimeo with `dnt=1`), each leaflet as a link opening in a new tab, then "Questions? Contact <hospital>" with the phone (a `tel:` link of its digits) and the email (a `mailto:` link, only for a plain address). **Nothing is put on the page as HTML:** titles, the phone and the email are plain text (React escapes them), as S35 requires for anything authors or staff wrote; a leaflet is a link only if its address starts `https://`.
- `features/information/info.ts`: the types and the small rules (`infoSummary`, `addable`, `kindLabel`, `telHref`, `mailtoHref`).

### End-to-end

- **J10 extended** (`test_j10_authoring.py::test_procedure_information_and_estimated_time`, data `e2e/data/information-time-hq.json`): Ada adds two trust leaflets and Hospital A1's phone and email; Alex gives a new procedure a library video and one leaflet on the Procedures page ("1 video, 1 leaflet"); a new episode for it; the patient, on a phone (WebKit): "About 4 minutes" (five questions, a 150-second video; a clinician-only question and a summary don't count) and "About 1 minute"; the Complete badge and Review button whole; on the home, Information: the player's address, "Length 10:35", the leaflet link (new tab), the phone and email links, not the other leaflet, no sideways scroll. Cara adds the other leaflet on the episode page ("Added for this patient by Cara Clinician"); the patient sees it. The editor's header says "About 4 minutes for the patient", and nothing for the clinician set.

## Decided by the builder (automatic mode)

- **Copied items keep pointing at the library item** (a row per item), not a frozen copy of its title and link. So an edit to a library item (a fixed link) reaches booked patients, and **retiring an item hides it from every patient** (brief focus 4); bringing it back shows it again. The list itself (which items) is what's copied.
- **Retired items are copied too** when an episode is made, so retiring and bringing back an item behaves the same for every episode.
- **Changing the episode's procedure swaps the copied items** for the new procedure's (the old procedure's videos would be wrong for the new operation); a clinician's own additions stay. Clearing the procedure removes the copied items.
- **A clinician may remove any item** from one episode, copied or added (the brief says "adding and removing an episode's items").
- **Adding or removing an item doesn't send or bump the episode's revision** (unlike tasks): each is a single add or remove that can't overwrite anything; a repeat add is a 409, a repeat remove a 404.
- **The patient's Information is fetched when they open it** (its own route), so videos load only when wanted.
- **The patient's Information route answers like their form** for a cancelled or closed episode (the form's 403); the home doesn't offer Information on those cards.
- **Conditional questions count** in the estimate (see above): the simplest rule that's never an underestimate; it can be tuned later.
- **The editor shows the estimate on the Question Set's own header**, which the brief names, not on each page.
- **No seed change.** `make seed` already clears each procedure's list and each hospital's settings; the tester adds a leaflet, a list and the hospital's contacts in the app.

## Tests (each written before its code, run once the code compiled)

- **TypeScript:** `packages/clinical/src/estimate.test.ts` (14: what counts and what doesn't, every page, nested and repeating groups, videos, rounding, the seconds per question followed, five more questions raise it, at least 1, undefined without config, the text); `apps/web/src/lib/config.test.ts` (seconds per question from the API only); `features/information/info.test.ts` (5); `features/procedures/procedures.test.ts` (lists compared by items).
  - Honest note: `estimate.test.ts` ran and failed (no module) before `estimate.ts` existed; `changedFields`'s list test failed before its fix. The config and `info.ts` tests were written first but run only with their code.
- **Go:** `internal/config` (`TestLoadSecondsPerQuestion`: default 15 locally and under strict, 20 from the environment, five bad values refused; `Public` carries it); `TestConfig` (routers) now expects `secondsPerQuestion` from the environment; **`internal/httpapi/information_integration_test.go`** (new): `TestProcedureInformation` (the five cases, plus a retired item kept while the list changes around it, an admin, clearing, POST), `TestEpisodeGetsItsProcedureInformation` (copied in order with retired items; none without a procedure; a later change to the procedure's list leaves made episodes alone and reaches new ones; changing the episode's procedure swaps the copied items and keeps a clinician's; clearing it; other changes leave the list alone), `TestListEpisodeInformation`, `TestAddEpisodeInformation`, `TestRemoveEpisodeInformation` (the five cases each, audit rows, General notes, 409 on cancelled and archived episodes), `TestPatientInformation` (active items in order, contact details, nothing staff-only, audited as the patient; a retired item brought back shows again; no sign-in; staff signed in get 401; another person's episode in this trust or another 404; bad ids 404; another trust's item in the list never sent).

### Results (26 Oct)

- `make lint`: passed.
- `make test`: Go all ok; web 27 files, 268 passed (then 28 files with `info.test.ts`); clinical 22 files, 302 passed; CDK 124 passed.
- `make test-integration`: every package ok (after `TestConfig` was updated for the new field).
- `make e2e` (every journey, on a dev stack on spare ports 3093/8093 on the shared `sj` database, stopped afterwards): **82 passed, 5 skipped, 1 failed**: `test_procedures.py` (S19) found the Name box off screen after Edit in a 300-pixel-high window: the form is taller now (the Information list), and it was scrolled into view with `block: "nearest"`, which shows the end of a form taller than the window. Fixed: it scrolls to the form's start (`block: "start"`, with a margin for the fixed header). Then `test_procedures.py`, J10 and J3: **7 passed**.
- **Checked at phone size by me** (WebKit, iPhone 13, screenshots): the tiles ("About 4 minutes", "About 1 minute"; Complete and Review whole, the row wrapping), the home's Information open (player, leaflet link, phone and email links, no sideways scroll); and on a desktop the Procedures form's list and picker, the episode's Information card and the editor's header.

## Hand checks for Rahul

- **A real video plays** in the patient's Information on a real phone (the journeys block the video sites and check the player's address).
- **The phone number's link offers to call** on a real phone, and the email link opens a mail app.
- **`SECONDS_PER_QUESTION` set to another number** (e.g. 30 in `.env`, then restart `make dev`) changes every estimate. The tester can't change the server's environment; covered by `TestLoadSecondsPerQuestion`, `TestConfig` and `estimate.test.ts`.

## Suggested commits

1. `S36: the estimated time` — `packages/clinical/src/{estimate,estimate.test,index}.ts`; `apps/api/internal/config/{config,config_test}.go`, `internal/httpapi/routers_integration_test.go`, `.env.example`; `apps/web/src/lib/{config,config.test}.ts`, `lib/WholeBadge.tsx`, `features/content/ContentLibrary.tsx`, `features/patient/PatientHQ.tsx`, `features/editor/Canvas.tsx`.
2. `S36: the procedure's and the episode's Information, and the patient's` — `apps/api/db/migrations/00030_episode_information.sql`, `db/queries/{procedures,information}.sql`, `internal/db/*` (generated), `internal/httpapi/{procedures,episodes,information,router,audit}.go`, `internal/httpapi/information_integration_test.go`; `apps/web/src/features/information/{info,info.test}.ts`, `{EpisodeInformation,PatientInformation}.tsx`, `features/procedures/{ProceduresPage.tsx,procedures.ts,procedures.test.ts}`, `features/episodes/EpisodePage.tsx`, `features/patient/PatientHome.tsx`, `apps/web/AGENTS.md`.
3. `S36: J10 extended` — `e2e/journeys/test_j10_authoring.py`, `e2e/data/information-time-hq.json`.
4. `S36: docs` — this folder, `docs/{decisions,pending}.md`, plan and session status, `CLAUDE.md`.

## Test rounds

- Test plan written 26 Oct ([test-plan.md](test-plan.md)).
- **Test report 1** (26 Oct): **all passed, 28 of 28** (21 cases, 6 regression checks and the setup). **S36 tested.**
  - Exploring 1, fixed (wording only): the Content library's intro said "anything that already uses it keeps its copy", no longer true for Information; it now says questionnaires keep their own copy of a video, but the item disappears from every patient's Information (`ContentLibrary.tsx`). No journey checks that line; `tsc` and the web tests pass (268).
  - Exploring 2–4, no code change (pending.md, "From S36's test report 1"): "About 1 minute" for a set with nothing to answer (the agreed "at least 1" rule; a question for Rahul); a cancelled or closed home card asks the patient to contact the hospital without showing its details; the episode's Information card on a cancelled episode doesn't say why it can't be changed.
  - Exploring 5–6: checks that passed (parallel adds, repeat removes, bad bodies, other trusts' items, keyboard pickers), and how a cancelled episode's patient was reached (the sign-in link): nothing to do.
  - The report has no new hand checks; the three above still apply.
