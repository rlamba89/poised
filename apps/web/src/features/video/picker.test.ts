import { describe, expect, it } from "vitest";
import { searchPath, tagsOf, videosOf, type Found } from "./picker";

const item = (over: Partial<Found>): Found => ({
  id: "1", platform: true, kind: "video", title: "Example", url: "https://youtu.be/aqz-KE-bpKQ", lengthSeconds: 635, tags: [], ...over,
});

describe("the video picker", () => {
  it("lists videos only, never leaflets", () => {
    const list = [item({ id: "v" }), item({ id: "l", kind: "leaflet", lengthSeconds: null })];
    expect(videosOf(list).map((i) => i.id)).toEqual(["v"]);
  });

  it("offers each video tag once, in any case, in order", () => {
    const list = [item({ tags: ["Fasting", "Example"] }), item({ tags: ["fasting", "The anaesthetic"] }), item({ kind: "leaflet", tags: ["Leaflets only"] })];
    expect(tagsOf(list)).toEqual(["Example", "Fasting", "The anaesthetic"]);
  });

  it("searches by words and a whole tag", () => {
    expect(searchPath("/o/t1", "", null)).toBe("/o/t1/content/search");
    expect(searchPath("/o/t1", " hip & knee ", "The anaesthetic")).toBe("/o/t1/content/search?q=hip%20%26%20knee&tag=The%20anaesthetic");
  });
});
