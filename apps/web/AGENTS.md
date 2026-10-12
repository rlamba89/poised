# apps/web

A plain React single-page app: **React 19 + Vite + React Router + Mantine**. There's no server rendering; everything runs in the browser. It replaced Next.js in plan F2 (session S04, 12 Oct 2026).

- **Pages and URLs:** `src/routes.tsx` lists every URL and its page (`src/pages/…`). `src/routes.test.ts` pins each URL to its page, so add a case there when you add a route.
  - Each page's code is loaded when it's first opened (the route's `lazy`), so the patient's link doesn't download the editor.
  - The pages are thin: they read the URL's params and render a component from `src/features/…`, where the real UI lives.
  - A page is keyed by its URL params (e.g. `key={eid}`), so moving to another episode starts afresh.
- **Errors:** the root route's `ErrorBoundary` (`src/pages/RouteError.tsx`) catches a page that fails to load or crashes. If the page's code couldn't be downloaded (usually after a deploy), it reloads once (`src/lib/chunkError.ts`); otherwise it shows "Something went wrong" with Reload.
- **Navigation:** `useNavigate`, `useParams`, `useLocation` and `<Link to>` from `react-router`. With Mantine: `<Button component={Link} to={…}>`.
- **API:** `src/lib/api.ts` calls `/api/…` on the same origin. `vite.config.ts` proxies `/api` to the Go API (`API_URL`, default `http://localhost:8080`), in `vite` (dev) and in `vite preview`. On AWS, Amplify does the same with a rewrite (plan F4).
- **Types:** `tsconfig.json` is the browser code, with no Node types, so `process.env` in `src/` fails the type check. `tsconfig.node.json` covers `vite.config.ts`, which runs in Node.
- **Imports:** `@/…` is `src/…`. `@poised/clinical` is the workspace package's TypeScript source, which Vite compiles.
- **Scripts:** `npm run dev` (port 3000), `build` (type-check, then `dist/`), `preview` (serves `dist/` on port 3000), `lint` (`tsc` on both tsconfigs), `test` (Vitest).
- **Nothing country-specific in the code** (A-20): the region, domain, time zone, locale and phone country come from config.
  - `main.tsx` loads `GET /api/config` once (`src/lib/config.ts`) before the first page renders. Format every date with `src/lib/format.ts`, which uses that locale; never pass a locale such as `"en-GB"` yourself.
  - A Go test (`apps/api/internal/config/literals_test.go`) fails on a locale, time-zone or region literal in `src/`.
