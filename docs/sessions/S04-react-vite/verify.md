# S04 verification: the web app on React + Vite + React Router

# Verification 2 (12 Oct 2026): **Verified**

**Re-verification** after the builder fixed verification 1 (below) and test report 3. Reviewed the whole uncommitted working tree again (HEAD `ac8ae27`), ran every test, re-ran `/code-review high` and checked the F-1 fix myself in the production build.

## Verification 1's findings

- **F-1 (must fix): fixed and checked.**
  - The root route's `ErrorBoundary` is [RouteError.tsx](../../../apps/web/src/pages/RouteError.tsx), with the logic in [chunkError.ts](../../../apps/web/src/lib/chunkError.ts). The unit tests (`chunkError.test.ts`) and the new `routes.test.ts` cases cover it.
  - **I checked it myself** in `vite build` + `vite preview --port 3100`, as Alex, in Claude in Chrome. This covers **T-19**, which report 3 left for Rahul's hand check: hiding the file has the same effect as blocking the request.
    1. **File gone for good** (the editor's chunk hidden, then a questionnaire clicked): exactly **one** automatic reload (my page marker was gone, and `poised:chunk-reload` was set once). Then **Something went wrong** / "This page couldn't be loaded…" / **Reload** / **Go to the start page**. After 5 seconds the same page was still there, so there was no loop.
    2. **File put back, then Reload:** the editor opened, with no console errors.
    3. **A deploy while the app is open** (the chunk renamed and the reference to it updated, so the open app still asked for the old name): one automatic reload, then the editor opened from the new file, with no message.
- **F-3, F-5, F-6: fixed.**
  - `tsconfig.node.json` now type-checks `vite.config.ts` alone, and `src/` has only `vite/client` types.
  - `data-mantine-color-scheme="light"` is back on `<html>`.
  - The layout test loops over all seven hospital pages.
- **F-4: partly fixed.** The dead `typeof window` check is gone.
- **F-2, the rest of F-4, and F-7:** logged in [pending.md](../../pending.md) §3, as agreed.

## New findings (all minor; logged in pending.md §3, none blocks)

From `/code-review high` on the fix, weighed by me:

1. **The 10-second reload window can loop on a very slow connection.** [chunkError.ts:26](../../../apps/web/src/lib/chunkError.ts#L26).
   - If one round (reload, load the app, the page's file fails) takes longer than 10 s, every failure reloads again, so the page reloads forever and the message never shows.
   - It needs a file that fails slowly every time (a stalled phone connection). A missing file after a deploy fails at once, which I checked.
   - **Fix:** clear the flag only after a page renders, or widen the window. Do it in S06, before the first deploy.
2. **Safari's wording when HTML is served in place of a missing `.js` file isn't recognised.** [chunkError.ts:7](../../../apps/web/src/lib/chunkError.ts#L7).
   - Safari words it "'text/html' is not a valid JavaScript MIME type".
   - It matters only if Amplify's SPA rewrite (F4) sends missing `/assets/*.js` to `index.html`. Amplify's standard SPA rewrite excludes `.js` and `.css`, which gives a real 404 that every browser reports as recognised.
   - **For S06:** keep `/assets/*` out of the rewrite, or add Vite's `vite:preloadError` listener as a backstop.
3. **"Go to the start page" sends a patient to the staff sign-in.** [RouteError.tsx:39](../../../apps/web/src/pages/RouteError.tsx#L39).
   - The error screen is shared by `/p/:token`, and `/` sends anyone who isn't signed in to `/login` (the dev user list).
   - It shows only after a patient page crashes or can't load; **Reload** is the main button.
   - **Fix:** hide the link on `/p/…` routes. Do it with S14/S15 (the patient's sign-in and home).
4. **Dev only:** StrictMode runs the error screen's effect twice, so "Something went wrong" flashes before the automatic reload. This doesn't happen in a production build, as I checked.
5. **Small:**
   - `HospitalLayout` (the staff shell) is in the first-load bundle, so the patient's link downloads it. It could be lazy too.
   - The Vite proxy key `/api` also matches `/api-…` paths; the old Next.js rewrite matched only `/api/…`. It applies only in dev and preview.

**Not findings:**
- Keying `AuthProvider` by hospital refetches `/api/me` on a hospital switch. That's deliberate, and the same as Next.js's layout (difference 3 in the build notes).
- The three `en-GB` lines are older than S04 and moving in S05.

## Checks

- **Scope:** the brief is complete. The fixes add only the error screen and the type-check split, and nothing else changed.
- **Tests first:**
  - `chunkError.test.ts` and `routes.test.ts` cover the new logic.
  - There are no new API routes.
  - `git diff -- e2e/journeys` is empty.
- **Tenant isolation and security:** no Go changes. The API is unchanged, and so are clinician-only stripping and the patient/staff split. Verification 1's browser checks (another hospital's POA, the patient link at phone size) still stand. Report 3 re-ran T-12 and T-15.
- **A-20 grep:** only the three older `en-GB` lines, planned for S05.
- **Organisation-name grep:** prints nothing.
- **Docs:** the F2 Status, `docs/plans/README.md` (F2 **Done**), `docs/sessions/README.md` (S04 **Verified**), `docs/decisions.md` §5 and `docs/pending.md` are updated in this session.

## Test results (12 Oct 2026, verification 2)

- `make lint test`: **pass**.
  - `apps/web` Vitest: *Test Files 2 passed (2), Tests 41 passed (41)*.
  - `packages/clinical`: *Test Files 14 passed (14), Tests 111 passed (111)*.
  - `go vet` and both `tsc` runs are clean.
- `make test-integration`: **pass**. `ok` for `apitest`, `auth`, `chapter`, `episode`, `httpapi` and `lifebox`. *EXIT 0.*
- `make e2e`, 3 runs in a row against `make dev`: *5 passed in 23.89s*, *5 passed in 22.12s*, *5 passed in 21.52s*.
- `npm run build -w apps/web`: **pass** (both type checks, then `vite build`).

## Still for Rahul by hand

- Click **Print** once on a POA Summary.

## Suggested commits

1. **`Web app on React + Vite + React Router; Next.js removed; UI renamed to Poised (F2 Steps 1–4)`**
   - Files: `apps/web/**`, `package-lock.json`, `.gitignore`, `Makefile`, `README.md`, `CLAUDE.md`, `e2e/README.md`, `docs/saas-requirements.md`, `docs/manual-test-plan.md`.
   - Tests: `apps/web/src/routes.test.ts`, `apps/web/src/lib/chunkError.test.ts`; J1–J5 unchanged.
   - Steps 1–4 can't be green on their own, so they go in one commit (build notes, "Suggested commits").
2. **`Docs: S04 decisions, build notes, test plan, test reports, verification and status; F2 done`**
   - Files: `docs/decisions.md`, `docs/pending.md`, `docs/plans/README.md`, `docs/plans/f2-react-vite.md`, `docs/sessions/README.md`, `docs/sessions/S04-react-vite/*`, `docs/sessions/S06-aws-qa-stack/brief.md` (the deploy-hardening note).

`reports/` and `research_notes/` stay untracked (decisions §5).

---

# Verification 1 (12 Oct 2026): Needs fixes

**Verifier session, 12 Oct 2026.** Reviewed the uncommitted working tree on `master` (HEAD `ac8ae27`) against the [brief](brief.md), [F2](../../plans/f2-react-vite.md), the build notes, the test plan and test reports 1–2.

## Verdict: **Needs fixes** (one small fix)

The migration is done well. The brief is fully covered, every test passes, J1–J5 are unchanged and pass 3 runs in a row, and my spot checks in the browser found no regressions. **One problem must be fixed before commit (F-1).** S04 introduced it by making every page load lazily: when a page's code file is missing, the user gets a raw error page with no way out. Everything else below is minor and can wait.

## Findings, worst first

### Must fix

- **F-1: a page whose code can't be downloaded shows React Router's raw error screen.** [apps/web/src/routes.tsx:12](../../../apps/web/src/routes.tsx#L12) (the root route has no `ErrorBoundary`/`errorElement`).
  - **Reproduced** with the production build (`vite build`, then `vite preview` on :3100). I opened the Questionnaires list, deleted the editor's chunk from `dist/assets/` as a redeploy would, and clicked a questionnaire. The whole screen became:
    > **Unexpected Application Error!** Failed to fetch dynamically imported module: …/assets/Editor-….js
    
    There's no header, no link back and no hint to reload.
  - **Why it matters:**
    - Every page is now loaded on first open (difference 1 in the build notes), so this happens to anyone who has the app open across a deploy and then opens a page they haven't opened yet. From S06/S07 on, the pipeline deploys QA often.
    - It also happens to a patient on a patchy phone signal.
    - Next.js recovered from this by doing a full page load. That's a behaviour change, which the brief doesn't allow.
  - **Fix (test-first):** give the root route an error element.
    - If the error is a failed dynamic import, reload the page **once**. Guard it with a `sessionStorage` flag so it can't loop, wrapped in try/catch.
    - Otherwise, show a plain "Something went wrong" message with **Reload** and a link home.
    - Vite's documented alternative is `window.addEventListener("vite:preloadError", () => location.reload())` in `main.tsx`. That alone still leaves render errors on the raw screen, so the error element is still needed.
    - Pin it with a `routes.test.ts` case (the root route has an error element) and a unit test of the "is this a failed chunk load" check.
    - Re-check by hand the way I did: build, preview, hide a chunk, click.

### Minor (don't block; fix now if cheap, or log in `docs/pending.md`)

- **F-2: the editor writes `?set=`/`&page=` with `window.history.replaceState`, behind the router's back.** [QuestionnaireEditor.tsx:93](../../../apps/web/src/features/editor/QuestionnaireEditor.tsx#L93).
  - Passing `history.state` through keeps React Router's key, and I confirmed in the browser that Back and Forward work: editor → set + page → Preview → Back returns to the same set and page → Back returns to the list, with `history.state` keeping `key`/`idx`.
  - But the router's own `location.search` goes stale, and the code relies on the shape of the router's history state.
  - The cleaner way is `useSearchParams` + `setSearchParams(…, { replace: true, preventScrollReset: true })`. That's worth doing when this file is next touched.
- **F-3: `tsconfig.json` gives all browser code Node's types.** [apps/web/tsconfig.json:15](../../../apps/web/tsconfig.json#L15).
  - `"types": ["vite/client", "node"]` lets `process.env.X` in `src/` pass `make lint` and then crash in the browser.
  - Nothing does that today (I grepped). A separate `tsconfig.node.json` for `vite.config.ts` would close the gap.
- **F-4: leftovers from Next.js.**
  - The `typeof window === "undefined"` guard at [EpisodePage.tsx:43](../../../apps/web/src/features/episodes/EpisodePage.tsx#L43) can't be true any more.
  - `PatientHQ`, `PoaSummary`, `ValidateSet` and `PreviewPage` keep default exports that existed only for `next/dynamic`.
  - The pages use `useParams() as {…}` casts, so a renamed `:param` would compile and pass `undefined`.
  - All three are cosmetic.
- **F-5: `index.html` lost Mantine's colour-scheme attribute.** [apps/web/index.html:2](../../../apps/web/index.html#L2).
  - The old layout had `ColorSchemeScript`/`mantineHtmlProps`. Adding `data-mantine-color-scheme="light"` to `<html>` restores it.
  - The effect today is at most a blank frame before React mounts.
- **F-6: one route test claims more than it checks.** [routes.test.ts:31](../../../apps/web/src/routes.test.ts#L31) is titled "puts every hospital page inside the hospital layout", but it checks one URL. Loop it over the hospital cases.
- **F-7: no "loading" cue when you open a page for the first time.**
  - The old page stays until the new page's code arrives. On the first open of a SurveyJS page that's 0.34 MB gzipped.
  - Not noticeable locally, and the test plan accepts it. Worth a thin progress bar (`useNavigation().state`) before patients use it on phones.

### Checked and fine

- **Scope:**
  - F2 Steps 1–4 are all done, with the Poised rename as agreed. There's nothing outside the brief.
  - The builder's 8 differences from the plan are each justified and recorded.
  - Every former Next.js URL maps to the same page (`routes.test.ts`, 17 cases).
  - `/h/<id>` alone and unknown paths are "not found".
- **Tests first:** no new routes or Go logic, so no new integration cases are needed. `routes.test.ts` covers the route table. `git diff -- e2e/journeys` is empty.
- **Tenant isolation (browser):**
  - As Bea (Hospital B), Hospital A's POA Summary deep link shows only "You don't have access to this hospital." The page never requested the episode.
  - A direct `fetch` of that episode as Bea returns **404**.
- **Patient link at 390 × 844** (a same-origin iframe, because the window wouldn't resize):
  - The section list fits (`scrollWidth` 388 = viewport), and the "Poised" top bar shows.
  - Only `PatientHQ` and its SurveyJS code load; the editor's code doesn't.
  - A wrong token shows "This link doesn't work…", not the 404 page.
  - No console errors.
- **Not found:** `/h/<A>` and `/h/<A>/episodes/x/nope` both show "404 / This page could not be found.", with the title Poised.
- **Deep link under `vite preview`:** `curl` of `/h/<A>/episodes/<eid>/poa` returns `200 text/html` (index.html).
- **A-20 grep:** three `en-GB` lines (`lib/format.ts:1,6`, `OptionLists.tsx:70`). All are older than S04 (from `aea5c9b`), and [S05's brief](../S05-lambda-entrypoints/brief.md) already plans to move them to `GET /api/config`. Not an S04 finding.
- **Security:**
  - No secrets or connection strings are added.
  - The API proxy is the dev/preview server only.
  - No patient data is in logs.
  - The Go code is unchanged, so clinician-only stripping and the patient/staff route split are unaffected.
- **Docs:**
  - The F2 Status, `docs/plans/README.md`, `docs/sessions/README.md`, `docs/decisions.md` (S04 entries, gotchas updated, open items), `apps/web/AGENTS.md`, `CLAUDE.md` and the README are all updated.
  - The organisation-name grep prints nothing.

### Still open (not a finding)

- **Print:** neither tester could open the real print dialog. **Rahul to click Print once** on a POA Summary.
- **The 7 "found while exploring" items** (decisions §5) all come from before S04. Decide whether to log them in `docs/pending.md`.

## Test results (12 Oct 2026, this session)

- `make lint test`: **pass**.
  - `go vet` and `tsc` are clean, and the Go unit tests are ok.
  - Vitest: `apps/web` *Test Files 1 passed, Tests 17 passed*; `packages/clinical` *Test Files 14 passed, Tests 111 passed*.
- `make test-integration`: **pass**. `ok` for `apitest`, `auth`, `chapter`, `episode`, `httpapi` and `lifebox`.
  - My shell didn't have `docker` on its PATH, so I re-ran with Docker Desktop's bin added. That's an environment quirk, not a code problem.
- `make e2e` against `make dev` (Vite), 3 runs in a row: *5 passed in 27.35s*, *5 passed in 24.30s*, *5 passed in 22.41s*.
- `npm run build -w apps/web`: built in 0.8 s. The chunk sizes match the build notes (`index` 325 kB, SurveyJS `src` 1,506 kB).
- **Browser spot checks** (Claude in Chrome): another hospital's POA, not-found pages, the patient link at phone size and with a bad token, the editor's `?set=` with Preview and Back, and a missing chunk in the production build. All behaved as before S04 **except F-1**.

## Suggested commits (after F-1 is fixed)

S04 is one plan (F2). Steps 1–4 can't be green separately (the app doesn't run until routes and imports have both moved), so I agree with the builder's split, with the F-1 fix folded into the first commit:

1. **`Web app on React + Vite + React Router; Next.js removed; UI renamed to Poised (F2 Steps 1–4)`**
   - Files: `apps/web/**`, `package-lock.json`, `.gitignore`, `Makefile`, `README.md`, `CLAUDE.md`, `e2e/README.md`, `docs/saas-requirements.md`, `docs/manual-test-plan.md`.
   - Tests: `apps/web/src/routes.test.ts` (with the F-1 cases); J1–J5 unchanged.
2. **`Docs: S04 decisions, build notes, test plan, test reports, verification and status`**
   - Files: `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/f2-react-vite.md`, `docs/sessions/README.md`, `docs/sessions/S04-react-vite/*`.

`reports/` and `research_notes/` stay untracked (decisions §5).
