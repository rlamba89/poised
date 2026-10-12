# S04 build notes: the web app on React + Vite + React Router

**Builder session, 12 Oct 2026.** Plan: [F2](../../plans/f2-react-vite.md), Steps 1–4.

## Questions asked

- **Rename the UI from "Lifebox" to "Poised" in S04?** Rahul: **yes** (12 Oct). Recorded in [decisions.md](../../decisions.md) §2.
  - **Renamed:** the browser tab title (`index.html`), the header beside the hospital picker (`HospitalLayout.tsx`), the sign-in page's heading ("Lifebox Authoring" → "Poised", `Login.tsx`) and the patient page's top bar (`PatientHQ.tsx`).
  - **Not renamed:** code comments that cite Lifebox as layout evidence.

## Before starting

- S03 was **Verified** (12 Oct).
- `git status`: only the untracked `reports/` and `research_notes/`, which are a known open item (decisions §5), not unreviewed code. Left alone.

## Steps done

### Step 1: the Vite scaffold

- **Pinned (exact, like the rest of the repo):** `vite` 8.3.4, `@vitejs/plugin-react` 6.1.2, `react-router` 8.4.0, `@types/react-dom` 19.3.0, `vitest` 4.1.11 (the version `packages/clinical` already uses).
- `apps/web/index.html`, `src/main.tsx` (the Mantine provider, global CSS and `RouterProvider`), `vite.config.ts` (dev and preview on port 3000; `/api` proxied to `API_URL`, default `http://localhost:8080`; `@/` → `src/`).
- `package.json` scripts: `dev` = `vite`, `build` = `tsc --noEmit && vite build`, `preview` = `vite preview`, `lint` = `tsc --noEmit`, `test` = `vitest run`. `"type": "module"` (Vite warns without it).

### Step 2: routes

- `src/routes.tsx`: one route table with the same URLs as before. Each Next.js page file was moved with `git mv` to `src/pages/…`, so its history follows.
- **Test first:** `src/routes.test.ts` checks that every URL reaches the right page with the right params, that the hospital pages are inside the hospital layout, and that unknown URLs are "not found". It failed before `routes.tsx` existed; it passes now (17 cases).
- A **not-found page** (`src/pages/NotFound.tsx`) with the same text as Next.js's: "404 / This page could not be found."

### Step 3: the Next.js imports replaced

- `useRouter().push/replace` → `useNavigate()`; `useParams`/`usePathname` → React Router's `useParams`/`useLocation`; `<Link href>` → `<Link to>`; `next/dynamic` → a plain import (see differences, 1).
- The `"use client"` line was removed from every file (it meant something only to Next.js, and Vite's bundler warns about it).

### Step 4: Next.js removed

- Deleted `next`, `next.config.ts`, `next-env.d.ts`, `src/app/` and `.next/`. `tsconfig.json` lost the Next plugin and gained Vite's types.
- Updated `apps/web/AGENTS.md` (which `apps/web/CLAUDE.md` includes) to describe the Vite app; the root `README.md`, `CLAUDE.md`, `Makefile` comment, `e2e/README.md`, and two stale lines in `docs/saas-requirements.md` and `docs/manual-test-plan.md`.
- `.gitignore` gains `dist/` (the build output). The `.next/` and `next-env.d.ts` lines stay, because `spikes/` is still a Next.js app.
- `make dev` is unchanged (`npm run dev -w apps/web`).

## Differences from the plan

1. **Every page's code loads when it's first opened** (React Router's `lazy` on each route), instead of plain imports plus `React.lazy` for the four SurveyJS pages.
   - **Why:** with plain imports, the first page anyone opened downloaded every screen: 2.3 MB of JavaScript, including the editor on the patient's phone. Next.js split the code per page by itself. With `lazy`, the first load is 0.53 MB, and SurveyJS (1.5 MB) loads only on pages that run a survey.
   - So the four SurveyJS pages import their component plainly; the route is what's lazy.
2. **The hospital layout has no path of its own;** each hospital page has its full path (`/h/:hospitalId/episodes`…).
   - **Why:** otherwise `/h/<id>` alone would show an empty hospital page. Under Next.js it was "not found", and it still is.
3. **Pages are keyed by their URL params** (e.g. `key={eid}`), and the hospital layout by the hospital.
   - **Why:** Next.js started a page afresh when its URL's ID changed; React Router keeps the same component and only changes its props. Keying keeps Next.js's behaviour, so nothing from one episode or hospital lingers on the next.
4. **`<ScrollRestoration />`** added to the root.
   - **Why:** Next.js scrolled a new page to the top and restored the position on Back. Without this, React Router keeps the old scroll position.
5. **The editor's `?set=` URL sync keeps the router's history state.** `QuestionnaireEditor.tsx` called `history.replaceState(null, …)`, which would wipe React Router's key for that history entry (used by Back and scroll restoring). It now passes `history.state` through. The URL behaviour is unchanged.
6. **Node 22.22 or newer is now required** (was 20.9). React Router 8 needs it. Node 20 has been end-of-life since April 2026; Rahul's Mac has 23.7. README and CLAUDE.md say so.
7. **`build.chunkSizeWarningLimit` is 1600 kB,** so the known SurveyJS chunk doesn't warn on every build.
8. **`npm run start` is gone;** `npm run preview` serves the production build on :3000.

## TypeScript for Rahul to review

The TypeScript is mostly moved code. The new parts:
- **`src/routes.tsx`:** a list of `{ id, path, lazy }` objects, one per page. `lazy: async () => ({ Component: (await import("@/pages/Login")).Login })` means "when this URL is first opened, download that file and use its `Login` component". The `id`s exist so the test can name pages.
- **`src/routes.test.ts`:** calls React Router's own `matchRoutes(routes, url)`, the same function the browser uses, and checks which route `id` and params come out. It needs no browser.
- **`src/pages/*.tsx`:** each is 5–8 lines: read the IDs from the URL (`useParams()`), and render the feature component with them. `as { hospitalId: string }` tells TypeScript the IDs are always there, because the route's path guarantees it.
- **`src/pages/hospital/HospitalLayout.tsx`:** the old `h/[hospitalId]/layout.tsx`, with `<Outlet />` where the page goes (Next.js passed it in as `children`).
- **`src/main.tsx`:** starts the app: Mantine, then the router.
- **`vite.config.ts`:** the dev server's port, the `/api` proxy, and the `@/` shortcut.

## Bundle size (production build, 12 Oct 2026)

| | Raw | Gzipped |
| --- | --- | --- |
| **First load** (any page: the app shell, Mantine, React, the router) | 0.76 MB (0.53 MB JS + 0.24 MB CSS) | 0.20 MB |
| **All JavaScript** | 2.57 MB | 0.66 MB |
| **All CSS** | 0.77 MB | 0.09 MB |
| SurveyJS + `packages/clinical` (loaded by the editor, preview, patient, validate and POA pages) | 1.51 MB JS + 0.52 MB CSS | 0.34 MB + 0.05 MB |

## Test results (12 Oct 2026)

- `make lint test test-integration`: **exit 0**. Go vet and `tsc` clean; Go unit tests ok; Vitest: `packages/clinical` 14 files, 111 tests passed; `apps/web` 1 file, 17 tests passed; integration: all 6 packages ok.
- `make e2e` against `vite` (dev), **3 runs in a row:** `5 passed in 27.38s`, `5 passed in 22.33s`, `5 passed in 24.44s`. **No journey or selector changed** (`git diff -- e2e/journeys` is empty).
- `make e2e` against `vite build` + `vite preview`: `5 passed in 12.85s`.
- **Deep link reload after `vite build`:** with `vite preview` on :3000, `/h/<id>/episodes/<eid>/poa` and `/p/<token>` loaded and reloaded correctly (headless Chromium), and `curl` of a deep link returns `index.html`.
- **Console sweep** (a scratch Playwright script, not committed): every screen opened and reloaded (login, questionnaires, editor, preview, not-found ×2, episodes, episode, POA, three validate screens, the patient at 390 × 844), in dev and in preview. **No console errors or warnings, and no page errors.**

## Suggested commits

S04 is one plan (F2) whose Steps 1–4 can't be committed separately with green tests: the app doesn't run until the routes and imports are moved (Steps 1–3), and Step 4 mostly deletes. So:

1. **`Web app on React + Vite + React Router; Next.js removed; UI renamed to Poised (F2 Steps 1–4)`**
   - `apps/web/**`, `package-lock.json`, `.gitignore`, `Makefile`, `README.md`, `CLAUDE.md`, `e2e/README.md`, `docs/saas-requirements.md`, `docs/manual-test-plan.md`.
   - Tests: `apps/web/src/routes.test.ts`; J1–J5 unchanged and passing.
2. **`Docs: S04 decisions, build notes, test plan and status`**
   - `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/f2-react-vite.md`, `docs/sessions/README.md`, `docs/sessions/S04-react-vite/*`.

## Manual testing

- Test plan: [test-plan.md](test-plan.md).

### Report 1 (12 Oct): 22 passed, 1 failed

- **Failure 1, T-16 step 4: Back doesn't return a long list to its scroll position.** **Not a regression, so no code change.**
  - **Checked:** I ran the pre-S04 Next.js build (`ac8ae27`, in a temporary worktree, `next dev` on :3100 against the same API) beside the Vite build. The same Playwright script (Cara: Episodes list, scroll 1000 px, open row 26, Back) gave `after_back=0` on **both**.
  - **Why:** the list's rows are fetched after the page appears, so when the position is restored the page is still short. Keeping it would need cached list data, which is a behaviour change outside S04.
  - **The test plan was wrong** (my expectation, not the app's). T-16 now expects the top after Back.
- **Plan notes from the tester, corrected in the plan:** T-05 needs **Add Page** first; T-06's Required is already on; T-14's Patient answers tab shows the disclosure, not "Yes".
- **Print (T-14 step 5):** the tester stubbed `window.print` (the real dialog freezes the extension). **Rahul to click Print once by hand.** The print CSS is unchanged by S04.
- **Found while exploring, items 1–7:** none comes from S04 (the code behind each is unchanged apart from the moved imports). Not fixed here; for Rahul to decide whether to log them in `docs/pending.md`:
  1. the stale-editor banner's body says "chapter";
  2. the delete modal shows an empty name while it fades;
  3. the hospital picker is blank on a hospital you can't access;
  4. a refused editor address has no header or way back;
  5. Alex sees **New episode** above "Only clinicians can see patients and episodes.";
  6. the printed POA Summary would include Add comment, the tabs and chevrons;
  7. an empty Question Set can be published.
- **Duplicate GETs in development:** React StrictMode runs effects twice in development, as Next.js did. Not in production builds.

### Report 2 (12 Oct): 10 passed, 0 failed

- Retested T-16 (corrected), T-14 step 5, R-01 to R-05 and three happy paths: **all passed**, no console errors. No code changed after report 1.
- **The tester withdrew two report-1 items** as tool effects (the Chrome tab was hidden, which slows animations): the delete modal's empty name, and missed first clicks. Exploring item 2 above is therefore not a bug.
- **New, not from S04:** the POA Summary still lists a clinician's change to a question that a later change hid (e.g. "Do you vape?" changed, then smoking set to No). It comes from how answers to hidden questions are kept (`packages/clinical`, unchanged in S04). Added to decisions §5 as an open item.
- **Still for Rahul by hand:** click **Print** once on a POA Summary.
- **S04 passed manual testing.** Next: `/verify-session S04`.

## Verification 1 (12 Oct): Needs fixes, now fixed

- **F-1 (must fix): a page whose code can't be downloaded showed React Router's raw "Unexpected Application Error!" screen.** Caused by S04's per-page loading: it happens after a deploy (the old file names are gone) or on a patchy signal. Next.js recovered with a full page load.
  - **Tests first:** `src/lib/chunkError.test.ts` (recognising the failed-download message in Chrome, Firefox and Safari, and Vite's CSS preload error; reloading at most once in 10 seconds; not reloading when storage can't be used) and new `routes.test.ts` cases (every page is under the root route, which has an error screen). Both failed first.
  - **Fix:** the root route's `ErrorBoundary` is `src/pages/RouteError.tsx`.
    - A failed download reloads the page once. A `sessionStorage` timestamp stops a loop: no second automatic reload within 10 seconds.
    - Otherwise, or if the reload didn't help, it shows "Something went wrong", **Reload** and a link to the start page.
    - Vite's `vite:preloadError` listener isn't needed: that error reaches the route's error screen anyway.
  - **Checked by hand in the production build** (`vite build`, `vite preview --port 3100`, Playwright blocking the editor's file): blocked **once** (as after a deploy): one automatic reload, then the editor opens. Blocked **always**: exactly one reload, then the message; with the file back, **Reload** opens the editor.
- **Minor ones fixed now (cheap):**
  - **F-3:** browser code no longer gets Node's types. `tsconfig.node.json` type-checks `vite.config.ts` alone with them; `lint` and `build` run both. A probe file using `process.env` in `src/` now fails `tsc`.
  - **F-4 (part):** the dead `typeof window === "undefined"` check in `EpisodePage.tsx` is gone.
  - **F-5:** `index.html` has `data-mantine-color-scheme="light"`, as Next.js's layout set.
  - **F-6:** the hospital-layout test loops over all seven hospital pages.
- **Logged in [pending.md](../../pending.md) §3:** F-2 (the editor's `replaceState`), the rest of F-4 (default exports, `useParams` casts), F-7 (no loading cue on a page's first open).
- **Results after the fixes:** `make lint test test-integration` exit 0 (`apps/web` Vitest 2 files, 41 tests; `packages/clinical` 14 files, 111 tests; integration all ok). `make e2e` 3 runs in a row: `5 passed in 27.41s`, `21.69s`, `21.68s`.
- **Report 3 (12 Oct, retest after verification 1): 5 passed, 0 failed, 1 blocked.** T-01, T-08, T-12 (phone size), T-15 and R-01 passed, with no console errors and no unexpected self-reloads in about 60 page loads.
  - **T-19 is blocked:** Claude in Chrome can't block a network request. **Rahul deferred it to a hand check** (12 Oct). The same behaviour is covered by the unit tests and my Playwright check in the production build (above). The test plan's T-19 now names current Chrome's panel (**Request conditions**).
  - **Found while exploring:** Enter doesn't save a test case in the preview (you must click **Save answers**). `TestCases.tsx` changed only by losing its `"use client"` line, so it's older than S04. Logged in `pending.md` §3.
- **Commits:** as suggested above, with the F-1 fix in commit 1 (`apps/web/src/lib/chunkError*.ts`, `apps/web/src/pages/RouteError.tsx`, `apps/web/tsconfig.node.json`) and `docs/pending.md` in commit 2.
