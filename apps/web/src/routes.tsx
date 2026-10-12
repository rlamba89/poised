// Every page and its URL: the same URLs as under Next.js (plan F2). Each route has an id for routes.test.ts.
// Each page's code loads when it's first opened (`lazy`), so the patient's link doesn't download the editor.
import type { RouteObject } from "react-router";
import { Root } from "@/pages/Root";
import { NotFound } from "@/pages/NotFound";
import { RouteError } from "@/pages/RouteError";
import { HospitalLayout } from "@/pages/hospital/HospitalLayout";

const h = "/h/:hospitalId";

export const routes: RouteObject[] = [
  {
    id: "root",
    element: <Root />,
    // Shown while the first page's code loads: nothing, as before.
    HydrateFallback: () => null,
    // Any page that fails to load or crashes: reload once if its code is missing, else a message.
    ErrorBoundary: RouteError,
    children: [
      { id: "home", path: "/", lazy: async () => ({ Component: (await import("@/pages/Home")).Home }) },
      { id: "login", path: "/login", lazy: async () => ({ Component: (await import("@/pages/Login")).Login }) },
      { id: "patient", path: "/p/:token", lazy: async () => ({ Component: (await import("@/pages/Patient")).Patient }) },
      {
        // No path of its own, so /h/:hospitalId alone isn't a page (it wasn't under Next.js either).
        id: "hospital",
        element: <HospitalLayout />,
        children: [
          {
            id: "questionnaires",
            path: `${h}/questionnaires`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Questionnaires")).Questionnaires }),
          },
          {
            id: "editor",
            path: `${h}/questionnaires/:qid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Editor")).Editor }),
          },
          {
            id: "preview",
            path: `${h}/chapters/:cid/preview`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Preview")).Preview }),
          },
          {
            id: "episodes",
            path: `${h}/episodes`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Episodes")).Episodes }),
          },
          {
            id: "episode",
            path: `${h}/episodes/:eid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Episode")).Episode }),
          },
          {
            id: "poa",
            path: `${h}/episodes/:eid/poa`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Poa")).Poa }),
          },
          {
            id: "validate",
            path: `${h}/episodes/:eid/sets/:cid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/ValidateSetPage")).ValidateSetPage }),
          },
        ],
      },
      { id: "not-found", path: "*", element: <NotFound /> },
    ],
  },
];
