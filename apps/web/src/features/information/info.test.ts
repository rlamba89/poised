import { describe, expect, it } from "vitest";
import { addable, infoSummary, kindLabel, mailtoHref, telHref } from "./info";

const item = (id: string, kind: "video" | "leaflet", title = id) => ({ id, kind, title });

describe("infoSummary", () => {
  it("counts the videos and leaflets", () => {
    expect(infoSummary([])).toBe("");
    expect(infoSummary([item("a", "video")])).toBe("1 video");
    expect(infoSummary([item("a", "leaflet"), item("b", "leaflet")])).toBe("2 leaflets");
    expect(infoSummary([item("a", "video"), item("b", "leaflet"), item("c", "video")])).toBe("2 videos, 1 leaflet");
  });
});

describe("addable", () => {
  it("is the library's items not in the list yet, by title", () => {
    const library = [item("b", "leaflet", "Fasting"), item("a", "video", "Anaesthetic"), item("c", "video", "Consent")];
    expect(addable(library, ["c"]).map((i) => i.id)).toEqual(["a", "b"]);
  });
});

describe("kindLabel", () => {
  it("names the kind", () => {
    expect(kindLabel("video")).toBe("Video");
    expect(kindLabel("leaflet")).toBe("Leaflet");
  });
});

describe("telHref and mailtoHref", () => {
  it("call the hospital's number: digits and a leading + only", () => {
    expect(telHref("0100 555 0123")).toBe("tel:01005550123");
    expect(telHref("+44 (0)100-555")).toBe("tel:+440100555");
    expect(telHref("")).toBe("");
    expect(telHref("call us")).toBe("");
  });
  it("write to its address, when it is one", () => {
    expect(mailtoHref("poa@hospital.example")).toBe("mailto:poa@hospital.example");
    expect(mailtoHref("")).toBe("");
    expect(mailtoHref("not an address")).toBe("");
    expect(mailtoHref("a@b.c?subject=x")).toBe("");
  });
});
