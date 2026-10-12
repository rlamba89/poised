# F2: Frontend on React + Vite + React Router

**Goal:** replace Next.js with a plain React single-page app (D-5), with **no change in behaviour or URLs**. F1's end-to-end journeys J1–J5 must pass unchanged, and they are the proof.

**Why:**
- Amplify Hosting supports Next.js only up to 15, and we're on 16.3.
- Every page is already a client component, so no Next.js server feature is used.
- A plain React app on Amplify needs no server compute.

The details are in [saas-requirements.md §13](../saas-requirements.md).

**Depends on:** F1 (the journeys J1–J5 must exist and pass first).

**Not in F2:** UI changes, new pages, or Amplify itself (that's F4).

## What exists today

- `apps/web` has 55 TS/TSX files. 20 of them import Next.js:
  - `next/navigation` (17 imports);
  - `next/link` (6);
  - `next/dynamic` (4).
- **Pages:** `src/app/**/page.tsx`, with two layouts: the root, and `h/[hospitalId]`.
- **API proxy:** `/api/*` is proxied to the Go API by a rewrite in `next.config.ts` (`API_URL`, default `http://localhost:8080`).
- `src/features/*` holds the real UI and doesn't change.

## Step 1: Vite scaffold beside the routes

- Add exact-pinned versions (as the repo already does) of `vite`, `@vitejs/plugin-react` and `react-router`.
- `apps/web/index.html` and `src/main.tsx`: the Mantine provider and global styles (moved from `app/layout.tsx`), plus `RouterProvider`.
- `vite.config.ts`:
  - dev server on **port 3000** (unchanged, so `E2E_BASE_URL` stays the same);
  - `server.proxy['/api']` → `process.env.API_URL ?? 'http://localhost:8080'`, which replaces the Next.js rewrite;
  - `packages/clinical` is resolved from the workspace. Vite compiles TypeScript itself, so `transpilePackages` isn't needed.
- `package.json` scripts: `dev` = `vite`, `build` = `tsc --noEmit && vite build`, `preview` = `vite preview --port 3000`, `lint` = `tsc --noEmit`.

## Step 2: routes

`src/routes.tsx` holds one `createBrowserRouter` with **the same paths** as today:

| Path | Today's file |
| --- | --- |
| `/` | `app/page.tsx` |
| `/login` | `app/login/page.tsx` |
| `/p/:token` | `app/p/[token]/page.tsx` |
| `/h/:hospitalId` (layout, `<Outlet/>`) | `app/h/[hospitalId]/layout.tsx` |
| `…/questionnaires`, `…/questionnaires/:qid` | `app/h/[hospitalId]/questionnaires/…` |
| `…/chapters/:cid/preview` | `app/h/[hospitalId]/chapters/[cid]/preview/page.tsx` |
| `…/episodes`, `…/episodes/:eid`, `…/episodes/:eid/poa`, `…/episodes/:eid/sets/:cid` | `app/h/[hospitalId]/episodes/…` |

Move each page component to `src/pages/…` with the same body. Add a "not found" route.

## Step 3: replace the Next.js imports

| Next.js | Replacement |
| --- | --- |
| `useRouter().push/replace` | `useNavigate()` |
| `useParams()` | `useParams()` from `react-router` |
| `usePathname()` | `useLocation().pathname` |
| `useSearchParams()` | `useSearchParams()` from `react-router` (it returns a pair) |
| `<Link href>` | `<Link to>` |
| `dynamic(() => import(…), { ssr: false })` | A plain import (nothing renders on a server any more), or `React.lazy` + `<Suspense>` where the bundle is large (the SurveyJS renderer) |

Any small pure helper written along the way gets a Vitest unit test first (TDD).

## Step 4: remove Next.js

- Delete `next`, `next.config.ts`, `next-env.d.ts` and `src/app/`. Keep `tsconfig` but drop the Next plugin.
- Update `apps/web/CLAUDE.md` and `AGENTS.md`, which describe Next.js, and the root `README.md`.
- `make dev` is unchanged (`npm run dev -w apps/web`).

## Checks

- `make lint` and `make test` pass. `make test-integration` is unaffected.
- **J1–J5 pass unchanged**, 3 runs in a row. If a journey needs a change, it must be because a selector depended on Next.js, never because behaviour changed. Explain any such change in the commit.
- After `vite build`, `vite preview` reloads a deep link (e.g. `/h/<id>/episodes/<eid>/poa`) correctly. Amplify gets the same rewrite in F4.
- The bundle size is noted under Status (to compare later).

## Status

**Done** (12 Oct 2026): **S04 verified** ([verify.md](../sessions/S04-react-vite/verify.md), verification 2). Verification 1's fix (a page whose code can't be downloaded) was retested in report 3 and checked by the verifier in the production build, which also covered T-19. Steps 1–4 are done; details in [S04's build notes](../sessions/S04-react-vite/build-notes.md).

- **Checks:** `make lint test test-integration` pass; J1–J5 pass unchanged, 3 runs in a row against `vite`, and once against `vite build` + `vite preview`; a deep link (`/h/<id>/episodes/<eid>/poa`) reloads correctly under `vite preview`.
- **Bundle size (production build, 12 Oct):** first load 0.76 MB raw (0.53 MB JS + 0.24 MB CSS), 0.20 MB gzipped. All JS 2.57 MB raw (0.66 MB gzipped); all CSS 0.77 MB (0.09 MB). SurveyJS with `packages/clinical` is one 1.51 MB chunk, loaded only by pages that run a survey.
- **Differences from the plan:**
  - every page's code loads when first opened (the route's `lazy`), instead of plain imports plus `React.lazy` for the SurveyJS pages, because plain imports made every first load 2.3 MB, the patient's phone included;
  - the hospital layout has no path of its own, so `/h/<id>` alone stays "not found";
  - pages are keyed by their URL IDs, and `<ScrollRestoration />` is added, to keep Next.js's behaviour;
  - Node 22.22+ is required (React Router 8);
  - the UI is renamed from "Lifebox" to "Poised" (Rahul, 12 Oct).
