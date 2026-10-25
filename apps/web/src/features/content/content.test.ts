import { describe, expect, it } from "vitest";
import { addTag, bodyOf, cleanTags, changedFields, formOf, formatLength, parseLength, type ContentItem } from "./content";

describe("a video's length (S33)", () => {
  it("shows as minutes and seconds, or hours, minutes and seconds", () => {
    expect(formatLength(200)).toBe("3:20");
    expect(formatLength(5)).toBe("0:05");
    expect(formatLength(3725)).toBe("1:02:05");
    expect(formatLength(null)).toBe("");
  });
  it("is typed as minutes:seconds, hours:minutes:seconds or whole minutes", () => {
    expect(parseLength("3:20")).toBe(200);
    expect(parseLength(" 12:05 ")).toBe(725);
    expect(parseLength("1:02:05")).toBe(3725);
    expect(parseLength("4")).toBe(240);
    expect(parseLength("")).toBeNull();
    expect(parseLength("3:75")).toBeNull();
    expect(parseLength("1:60:00")).toBeNull();
    expect(parseLength("abc")).toBeNull();
    expect(parseLength("0:00")).toBeNull();
  });
});

describe("the tags typed", () => {
  it("drops blank ones and repeats in any case, and trims them (test report 2: spaces left an empty chip)", () => {
    expect(cleanTags(["Hip", "   ", " Knee ", "hip", ""])).toEqual(["Hip", "Knee"]);
  });
});

describe("the ready-made tags", () => {
  it("adds a checklist topic once, in any case", () => {
    expect(addTag(["Knee"], "Fasting")).toEqual(["Knee", "Fasting"]);
    expect(addTag(["fasting"], "Fasting")).toEqual(["fasting"]);
  });
});

describe("the add and edit form", () => {
  const item: ContentItem = {
    id: "c1", platform: false, kind: "video", title: "Hip", url: "https://youtu.be/aqz-KE-bpKQ", lengthSeconds: 200,
    tags: ["Hip"], status: "active", updatedAt: "2026-10-25T10:00:00Z", updatedByName: "Ada",
  };
  it("sends a video's length in seconds, and none for a leaflet", () => {
    expect(bodyOf(formOf(item))).toEqual({ kind: "video", title: "Hip", url: "https://youtu.be/aqz-KE-bpKQ", lengthSeconds: 200, tags: ["Hip"] });
    expect(bodyOf({ ...formOf(item), kind: "leaflet" })).toEqual({ kind: "leaflet", title: "Hip", url: "https://youtu.be/aqz-KE-bpKQ", tags: ["Hip"] });
  });
  it("sends a length it can't read as null, so Go says what's wrong", () => {
    expect(bodyOf({ ...formOf(item), length: "soon" }).lengthSeconds).toBeNull();
  });
  it("sends only what changed on an edit, never the kind", () => {
    const before = bodyOf(formOf(item));
    expect(changedFields(before, bodyOf({ ...formOf(item), title: "Hip video" }))).toEqual({ title: "Hip video" });
    expect(changedFields(before, bodyOf({ ...formOf(item), tags: ["Hip", "Recovery"] }))).toEqual({ tags: ["Hip", "Recovery"] });
    expect(changedFields(before, bodyOf(formOf(item)))).toEqual({});
  });
});
