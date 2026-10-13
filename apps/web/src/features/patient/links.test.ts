import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api";
import { isPatientPage, linkStopped } from "./links";

describe("linkStopped", () => {
  it("a locked link says so, and a sign-in link offers a new one", () => {
    const e = new ApiError(403, "This link is locked…", { code: "locked" });
    expect(linkStopped(e, "invite")).toMatchObject({ title: "This link is locked", newLink: false });
    expect(linkStopped(e, "signin")).toMatchObject({ title: "This link is locked", newLink: true });
  });
  it("a used or expired sign-in link offers a new one", () => {
    const e = new ApiError(404, "This sign-in link has expired…", { code: "expired" });
    expect(linkStopped(e, "signin")).toMatchObject({ title: "This link has expired", message: "This sign-in link has expired…", newLink: true });
  });
  it("an unknown link doesn't work; a suspended hospital isn't available", () => {
    expect(linkStopped(new ApiError(404, "nope"), "invite")).toMatchObject({ title: "This link doesn't work", newLink: false });
    expect(linkStopped(new ApiError(404, "nope"), "signin")).toMatchObject({ title: "This link doesn't work", newLink: true });
    expect(linkStopped(new ApiError(403, "closed"), "invite")).toMatchObject({ title: "Not available at the moment" });
  });
  it("anything else isn't a stop: the page shows the message and lets them try again", () => {
    expect(linkStopped(new ApiError(401, "wrong date"), "invite")).toBeNull();
    expect(linkStopped(new ApiError(500, "oops"), "signin")).toBeNull();
    expect(linkStopped(new Error("network"), "signin")).toBeNull();
  });
});

describe("isPatientPage", () => {
  it.each([
    ["/home", true],
    ["/home/episodes/e1", true],
    ["/i/tok", true],
    ["/s/tok", true],
    ["/v/tok", true],
    ["/signin", true],
    ["/", false],
    ["/login", false],
    ["/o/o1/h/h1/episodes", false],
    ["/homes", false],
  ])("%s → %s", (path, want) => {
    expect(isPatientPage(path)).toBe(want);
  });
});
