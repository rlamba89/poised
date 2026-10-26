# S36 verification

## Verification 1 (26 Oct 2026, automatic run)

**Verdict: Verified.**

The brief and answers.md are met: the patient's Information on the episode's home card (answer 1), the procedure's list copied onto the episode when it's made (answer 2), and the estimated time with the seconds per question from Go's `internal/config` (`SECONDS_PER_QUESTION`, default 15, sent by `GET /api/config`; no other place holds the number: answer 3). Every new route has the five integration cases, is audited, and is scoped to the trust and hospital. Patient-facing text from authors and staff is plain text (React-escaped); links are built only from checked parts (S34's `videoEmbedFor`, https-only leaflets enforced by the database, digit-only `tel:`, plain-address `mailto:`). The only HTML path is S35's sanitised `InformationQuestion`, unchanged.

### Findings (worst first; none blocks)

1. **Question for Rahul (answer 2's reading).** answers.md says "the list is copied onto the episode when it's made, so a booked patient keeps the list from when they were booked". The builder copies the **list** (which items) but each row still points at the library item (`episode_info_items.content_item_id`, `apps/api/db/migrations/00030_episode_information.sql:17`). So a later **edit** of a library item (its title or link) reaches booked patients, and **retiring** an item hides it from every patient (the brief's "Retired items stop showing" and manual-test focus 4 ask for that). Changing a *procedure's* list leaves booked episodes alone, as answered. This is a reasonable reading that satisfies both the brief and answer 2; recorded in pending.md for Rahul to confirm rather than as a fix.
2. **Minor: the estimate counts questions inside a hidden Group or page.** `packages/clinical/src/estimate.ts:30` skips a `visible: false` element itself but still walks its children, and line 36 checks only `clinicianOnly` on pages. A hidden Group of 8 questions would add 8 × 15 s. Rare in authored content (conditions use `visibleIf`, which deliberately counts). pending.md.
3. **Minor: `telHref` keeps a trunk 0 after a country code.** `apps/web/src/features/information/info.ts:54`: "+44 (0)20 7946 0000" becomes `tel:+4402079460000`, which dials wrongly while the text looks right. The hospital types its own number; pending.md.
4. **Minor: the Procedures table's Information summary counts retired items** (`ProceduresPage.tsx:144`), which patients don't see. The form marks them "Retired"; pending.md.
5. **Minor: add/remove read the episode's status before writing, without a lock** (`information.go:96`, `:132`): a cancel committed in between lets one add or remove through. Same pattern as other single-row episode changes; low impact. pending.md.
6. **Minor: changing an episode's procedure swaps its copied Information silently** — the General notes record the procedure change but not the list change (`episodes.go`, `swapProcedureInfo`). pending.md.
7. **Fixed by the verifier (docs only):** the builder's CLAUDE.md edit had pulled the "Answers are stored per form, per actor" bullet out of the Content model list (its indent was lost), so the Pre-fill and Other trusts bullets nested under it. Indent restored.
8. Cleanups noted by `/code-review high`, not worth a round: the episode and procedure pickers repeat the same searchable Select (`EpisodeInformation.tsx:98`, `ProceduresPage.tsx:297`); `listEpisodeInfo` resends the whole library after each add or remove; `estimateText(estimatedMinutes(...))` computed twice per render in `Canvas.tsx` and `PatientHQ.tsx`. A `WholeBadge` caveat (function `style` and caller `styles` dropped) doesn't affect any current caller.

### Checked

- **Scope:** everything in the brief is built (INF-04 procedure list, a clinician's per-episode items, the patient's view with the hospital's phone and email, PX-10 on the tiles and in the editor, J10 extended). Extra: the 390px tile fix (badge and button no longer cut short), from pending.md, small and tested by J10.
- **Tests first:** `information_integration_test.go` has the five cases for the procedure PATCH/POST, the episode's list, add and remove, and the patient's read, plus copy-on-create, swap-on-procedure-change and other-trust items. `estimate.test.ts` (14), `info.test.ts`, `config.test.ts`, `TestLoadSecondsPerQuestion`.
- **Tenant isolation:** every read joins only Preopsis's items and the episode's own trust's (`information.sql`, `procedures.sql`'s `info_items`); staff routes use `hosp(...)` + `loadEpisode`; the patient's route `loadPatientForm` (own episodes only).
- **A-20 grep:** only pre-existing comments (`format.ts`, `patient.go`) and infra tests; nothing new. `15` appears only in Go's `DefaultSecondsPerQuestion` and comments.
- **Security:** no secrets; no patient data in notes or audit rows (General notes name the item's title only, as tasks do); the patient router has only `/api/p/…/information`.
- **Docs:** decisions.md, pending.md, CLAUDE.md, apps/web/AGENTS.md, plan and session status updated by the builder; this verification sets S36 Verified and plan C5 Done.

### Spot-checks (Chrome, trk profile, against `make dev`)

1. **Another trust (Bea, Trust B):** a Trust A episode's Information by Trust B's path: GET and DELETE **404** "Episode not found."; Trust A's path: **404** "Hospital not found."; Trust A's leaflet into a Trust B procedure's list: **400**; onto a Trust B episode: **400**; Trust B's library offers no Trust A item.
2. **Hospital scope (Cara, Hospital A1 only):** an A1 episode through Hospital A2's path: **404**. Her A1 episode's list: 200.
3. **Copy on create (database):** Knee arthroscopy's list now has 1 item; an episode made before the tester shortened it still has its 2 copied items; later ones have 1. No `episode_info_items` row anywhere points at another trust's item (0 rows).
4. **Episode page card** (desktop): "Information for the patient", the procedure's video with "Video · 10:35 · From the procedure", Remove and the picker.
5. The patient's phone-size view (tiles "About N minutes", whole badge and button, Information with player, leaflet and contact links, no sideways scroll) is covered by J10 in WebKit at iPhone 13, which passed in this run, and by the tester's T-06/T-07; not repeated by hand.

### Test results (26 Oct)

- `make lint`: passed.
- `make test`: Go all ok; web 27 files, 268 passed; clinical 22 files, 302 passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e`: **83 passed, 5 skipped** (13 min 44 s).

### Hand checks for Rahul

- A real video plays in the patient's Information on a real phone.
- The phone link offers to call and the email link opens a mail app on a real phone.
- `SECONDS_PER_QUESTION=30` in `.env`, restart `make dev`: every estimate changes.

### Suggested commits

1. `S36: the estimated time` — `packages/clinical/src/{estimate,estimate.test,index}.ts`; `apps/api/internal/config/{config,config_test}.go`, `apps/api/internal/httpapi/routers_integration_test.go`, `.env.example`; `apps/web/src/lib/{config,config.test}.ts`, `apps/web/src/lib/WholeBadge.tsx`, `apps/web/src/features/content/ContentLibrary.tsx`, `apps/web/src/features/patient/PatientHQ.tsx`, `apps/web/src/features/editor/Canvas.tsx`.
2. `S36: the procedure's and the episode's Information, and the patient's` — migration 00030, `db/queries/{procedures,information}.sql`, `internal/db/*`, `internal/httpapi/{procedures,episodes,information,router,audit}.go`, `information_integration_test.go`; `apps/web/src/features/information/*`, `features/procedures/*`, `features/episodes/EpisodePage.tsx`, `features/patient/PatientHome.tsx`, `apps/web/AGENTS.md`.
3. `S36: J10 extended` — `e2e/journeys/test_j10_authoring.py`, `e2e/data/information-time-hq.json`.
4. `S36: docs` — this folder, `docs/{decisions,pending}.md`, plan and session status, `CLAUDE.md`.
