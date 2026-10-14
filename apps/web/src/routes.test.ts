import { describe, expect, it } from "vitest";
import { matchRoutes } from "react-router";
import { routes } from "./routes";

// Each URL reaches its page, with its params. A hospital's pages are under /o/<trust>/h/<hospital> (S08).
function page(path: string) {
  const matches = matchRoutes(routes, path);
  const last = matches?.at(-1);
  return { id: last?.route.id, params: last?.params };
}

describe("routes", () => {
  it.each([
    ["/", "home", {}],
    ["/login", "login", {}],
    ["/auth/callback", "auth-callback", {}],
    ["/i/tok123", "invite", { token: "tok123" }],
    ["/home/episodes/e1", "patient-episode", { eid: "e1" }],
    ["/home", "patient-home", {}],
    ["/signin", "patient-signin", {}],
    ["/s/tok123", "signin-link", { token: "tok123" }],
    ["/v/tok123", "verify-contact", { token: "tok123" }],
    ["/qa/messages", "qa-messages", {}],
    ["/qa/jobs", "qa-jobs", {}],
    ["/p/tok123", "not-found", {}], // the old link (before S14) is gone
    ["/o/o1/h/h1/questionnaires", "questionnaires", { orgId: "o1", hospitalId: "h1" }],
    ["/o/o1/h/h1/questionnaires/q1", "editor", { orgId: "o1", hospitalId: "h1", qid: "q1" }],
    ["/o/o1/h/h1/chapters/c1/preview", "preview", { orgId: "o1", hospitalId: "h1", cid: "c1" }],
    ["/o/o1/h/h1/episodes", "episodes", { orgId: "o1", hospitalId: "h1" }],
    ["/o/o1/h/h1/episodes/new", "new-episode", { orgId: "o1", hospitalId: "h1" }],
    ["/o/o1/h/h1/episodes/e1", "episode", { orgId: "o1", hospitalId: "h1", eid: "e1" }],
    ["/o/o1/h/h1/patients/p1", "patient-record", { orgId: "o1", hospitalId: "h1", pid: "p1" }],
    ["/o/o1/h/h1/episodes/e1/poa", "poa", { orgId: "o1", hospitalId: "h1", eid: "e1" }],
    ["/o/o1/h/h1/episodes/e1/sets/c1", "validate", { orgId: "o1", hospitalId: "h1", eid: "e1", cid: "c1" }],
    ["/o/o1/h/h1/episodes/e1/patient-form", "patient-form", { orgId: "o1", hospitalId: "h1", eid: "e1" }],
    ["/o/o1/h/h1/staff", "staff", { orgId: "o1", hospitalId: "h1" }],
    ["/o/o1/h/h1/settings", "settings", { orgId: "o1", hospitalId: "h1" }],
    ["/o/o1/h/h1/audit", "audit", { orgId: "o1", hospitalId: "h1" }],
    ["/admin", "admin-trusts", {}],
    ["/admin/trusts/new", "admin-new-trust", {}],
    ["/admin/trusts/o1", "admin-trust", { orgId: "o1" }],
  ])("%s is the %s page", (path, id, params) => {
    const got = page(path);
    expect(got.id).toBe(id);
    expect(got.params).toMatchObject(params);
  });

  it.each([
    "/o/o1/h/h1/questionnaires",
    "/o/o1/h/h1/questionnaires/q1",
    "/o/o1/h/h1/chapters/c1/preview",
    "/o/o1/h/h1/episodes",
    "/o/o1/h/h1/episodes/new",
    "/o/o1/h/h1/episodes/e1",
    "/o/o1/h/h1/patients/p1",
    "/o/o1/h/h1/episodes/e1/poa",
    "/o/o1/h/h1/episodes/e1/sets/c1",
    "/o/o1/h/h1/episodes/e1/patient-form",
    "/o/o1/h/h1/staff",
    "/o/o1/h/h1/settings",
    "/o/o1/h/h1/audit",
  ])("puts %s inside the hospital layout", (path) => {
    const matches = matchRoutes(routes, path) ?? [];
    const layout = matches.find((m) => m.route.id === "hospital");
    expect(layout).toBeDefined();
    // The layout has no path of its own, but it reads the hospital from the page's params.
    expect(layout?.params.orgId).toBe("o1");
    expect(layout?.params.hospitalId).toBe("h1");
  });

  it.each(["/", "/login", "/i/tok123", "/home/episodes/e1", "/o/o1/h/h1/episodes/e1/poa", "/nope"])(
    "%s is under the root route, whose error screen catches a page that fails to load",
    (path) => {
      const root = matchRoutes(routes, path)?.[0].route;
      expect(root?.id).toBe("root");
      expect(root?.ErrorBoundary).toBeDefined();
    },
  );

  it.each(["/admin", "/admin/trusts/new", "/admin/trusts/o1"])("puts %s inside the admin layout (S10)", (path) => {
    const matches = matchRoutes(routes, path) ?? [];
    expect(matches.some((m) => m.route.id === "admin")).toBe(true);
    expect(matches.some((m) => m.route.id === "hospital")).toBe(false);
  });

  it.each(["/i/tok123", "/home/episodes/e1", "/qa/messages"])("keeps %s out of the staff and admin layouts (no staff sign-in)", (path) => {
    const ids = (matchRoutes(routes, path) ?? []).map((m) => m.route.id);
    expect(ids).not.toContain("hospital");
    expect(ids).not.toContain("admin");
  });

  it.each(["/nope", "/admin/nope", "/admin/trusts", "/h/h1", "/o/o1/h/h1/nope", "/o/o1/h/h1/episodes/e1/nope", "/o/o1/h/h1/patients", "/p", "/login/extra"])(
    "%s is not found, as it was under Next.js",
    (path) => {
      expect(page(path).id).toBe("not-found");
    },
  );
});
