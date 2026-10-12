import { describe, expect, it } from "vitest";
import { matchRoutes } from "react-router";
import { routes } from "./routes";

// The URLs are the same as under Next.js (plan F2): each one reaches the same page, with the same params.
function page(path: string) {
  const matches = matchRoutes(routes, path);
  const last = matches?.at(-1);
  return { id: last?.route.id, params: last?.params };
}

describe("routes", () => {
  it.each([
    ["/", "home", {}],
    ["/login", "login", {}],
    ["/p/tok123", "patient", { token: "tok123" }],
    ["/h/h1/questionnaires", "questionnaires", { hospitalId: "h1" }],
    ["/h/h1/questionnaires/q1", "editor", { hospitalId: "h1", qid: "q1" }],
    ["/h/h1/chapters/c1/preview", "preview", { hospitalId: "h1", cid: "c1" }],
    ["/h/h1/episodes", "episodes", { hospitalId: "h1" }],
    ["/h/h1/episodes/e1", "episode", { hospitalId: "h1", eid: "e1" }],
    ["/h/h1/episodes/e1/poa", "poa", { hospitalId: "h1", eid: "e1" }],
    ["/h/h1/episodes/e1/sets/c1", "validate", { hospitalId: "h1", eid: "e1", cid: "c1" }],
  ])("%s is the %s page", (path, id, params) => {
    const got = page(path);
    expect(got.id).toBe(id);
    expect(got.params).toMatchObject(params);
  });

  it.each([
    "/h/h1/questionnaires",
    "/h/h1/questionnaires/q1",
    "/h/h1/chapters/c1/preview",
    "/h/h1/episodes",
    "/h/h1/episodes/e1",
    "/h/h1/episodes/e1/poa",
    "/h/h1/episodes/e1/sets/c1",
  ])("puts %s inside the hospital layout", (path) => {
    const matches = matchRoutes(routes, path) ?? [];
    const layout = matches.find((m) => m.route.id === "hospital");
    expect(layout).toBeDefined();
    // The layout has no path of its own, but it reads the hospital from the page's params.
    expect(layout?.params.hospitalId).toBe("h1");
  });

  it.each(["/", "/login", "/p/tok123", "/h/h1/episodes/e1/poa", "/nope"])(
    "%s is under the root route, whose error screen catches a page that fails to load",
    (path) => {
      const root = matchRoutes(routes, path)?.[0].route;
      expect(root?.id).toBe("root");
      expect(root?.ErrorBoundary).toBeDefined();
    },
  );

  it.each(["/nope", "/h/h1", "/h/h1/nope", "/h/h1/episodes/e1/nope", "/p", "/login/extra"])(
    "%s is not found, as it was under Next.js",
    (path) => {
      expect(page(path).id).toBe("not-found");
    },
  );
});
