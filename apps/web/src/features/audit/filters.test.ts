import { describe, expect, it } from "vitest";
import { auditQuery, badDays, namesOf, readFilters, withoutBadDays } from "./filters";

// The Audit screen's filters live in the page's URL (?user=&from=&to=&patient=), as days and text.
// auditQuery turns them into the API's query: the days become the browser's own midnights, so a
// day means the same on the screen, in the list and in the CSV, with no time zone in the code.
describe("the audit filters (S11)", () => {
  it("reads the URL, ignoring anything that isn't a filter", () => {
    expect(readFilters(new URLSearchParams("user=u1&from=2026-10-01&to=2026-10-02&patient=Smith&x=1"))).toEqual({
      user: "u1", from: "2026-10-01", to: "2026-10-02", patient: "Smith",
    });
    expect(readFilters(new URLSearchParams(""))).toEqual({ user: "", from: "", to: "", patient: "" });
  });

  it("turns days into the browser's midnights: from the start of `from` to the end of `to`", () => {
    const q = auditQuery({ user: "u1", from: "2026-10-01", to: "2026-10-02", patient: " Smith " });
    expect(q.get("user")).toBe("u1");
    expect(q.get("patient")).toBe("Smith");
    expect(new Date(q.get("from")!).getTime()).toBe(new Date(2026, 9, 1).getTime());
    expect(new Date(q.get("to")!).getTime()).toBe(new Date(2026, 9, 3).getTime()); // the day after `to`
  });

  it("leaves out the filters that aren't set", () => {
    expect(auditQuery({ user: "", from: "", to: "", patient: "  " }).toString()).toBe("");
  });

  it("adds the CSV format and the browser's time zone for the export", () => {
    const q = auditQuery({ user: "", from: "", to: "", patient: "" }, { csv: true, timeZone: "Pacific/Auckland" });
    expect(q.get("format")).toBe("csv");
    expect(q.get("tz")).toBe("Pacific/Auckland");
  });

  it("ignores a day that isn't a date rather than sending nonsense", () => {
    expect(auditQuery({ user: "", from: "soon", to: "2026-02-30x", patient: "" }).toString()).toBe("");
  });

  it("says when a day isn't a real day, so the screen can say so rather than ignore it", () => {
    const f = { user: "", from: "", to: "", patient: "" };
    expect(badDays(f)).toBe(false);
    expect(badDays({ ...f, from: "2026-10-01", to: "2026-10-02" })).toBe(false);
    expect(badDays({ ...f, from: "2026-02-30" })).toBe(true);
    expect(badDays({ ...f, to: "soon" })).toBe(true);
    expect(badDays({ ...f, from: "2026-10-01", to: "2026-13-01" })).toBe(true);
  });

  it("names the patients the API sent, and says how many more the row touched", () => {
    expect(namesOf(["A", "B"], 2)).toBe("A, B");
    expect(namesOf(["A", "B"], 332)).toBe("A, B and 330 more");
  });

  it("empties a day that isn't real, keeping the other filters, so the form can be applied again", () => {
    const f = { user: "u1", from: "2026-02-30", to: "2026-03-05", patient: "Smith" };
    expect(withoutBadDays(f)).toEqual({ user: "u1", from: "", to: "2026-03-05", patient: "Smith" });
    expect(withoutBadDays({ ...f, from: "2026-03-01" })).toEqual({ ...f, from: "2026-03-01" });
  });
});
