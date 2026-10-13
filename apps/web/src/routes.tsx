// Every page and its URL. Each route has an id for routes.test.ts. A hospital's pages are under
// /o/<trust>/h/<hospital> (plan C1, S08).
// Each page's code loads when it's first opened (`lazy`), so the patient's link doesn't download the editor.
import type { RouteObject } from "react-router";
import { Root } from "@/pages/Root";
import { NotFound } from "@/pages/NotFound";
import { RouteError } from "@/pages/RouteError";
import { HospitalLayout } from "@/pages/hospital/HospitalLayout";
import { AdminLayout } from "@/pages/admin/AdminLayout";

const h = "/o/:orgId/h/:hospitalId";

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
        // No path of its own, so /o/:orgId/h/:hospitalId alone isn't a page.
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
            // The New episode page (S13): find the patient, or type their details, then the episode.
            id: "new-episode",
            path: `${h}/episodes/new`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/NewEpisode")).NewEpisode }),
          },
          {
            id: "episode",
            path: `${h}/episodes/:eid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Episode")).Episode }),
          },
          {
            // A patient's page: the trust's record and their episodes at this hospital only (S13).
            id: "patient-record",
            path: `${h}/patients/:pid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Patient")).Patient }),
          },
          {
            id: "poa",
            path: `${h}/episodes/:eid/poa`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Poa")).Poa }),
          },
          {
            id: "staff",
            path: `${h}/staff`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Staff")).Staff }),
          },
          {
            id: "settings",
            path: `${h}/settings`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Settings")).Settings }),
          },
          {
            id: "audit",
            path: `${h}/audit`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/Audit")).Audit }),
          },
          {
            id: "validate",
            path: `${h}/episodes/:eid/sets/:cid`,
            lazy: async () => ({ Component: (await import("@/pages/hospital/ValidateSetPage")).ValidateSetPage }),
          },
        ],
      },
      {
        // The platform admin's console (S10): trusts and their hospitals. No patient data (OPS-06).
        id: "admin",
        element: <AdminLayout />,
        children: [
          { id: "admin-trusts", path: "/admin", lazy: async () => ({ Component: (await import("@/pages/admin/Trusts")).Trusts }) },
          {
            id: "admin-new-trust",
            path: "/admin/trusts/new",
            lazy: async () => ({ Component: (await import("@/pages/admin/NewTrust")).NewTrust }),
          },
          {
            id: "admin-trust",
            path: "/admin/trusts/:orgId",
            lazy: async () => ({ Component: (await import("@/pages/admin/Trust")).Trust }),
          },
        ],
      },
      { id: "not-found", path: "*", element: <NotFound /> },
    ],
  },
];
