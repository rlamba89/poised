import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { canAskFresh, keyPlace } from "./keys";

describe("canAskFresh", () => {
  it("is offered on patient questions and repeating groups, not on clinician-only content, statements, calculations, groups, or one-off answers", () => {
    expect(canAskFresh({ type: "text", name: "q" })).toBe(true);
    expect(canAskFresh({ type: "paneldynamic", name: "q" })).toBe(true);
    expect(canAskFresh({ type: "text", name: "q", clinicianOnly: true })).toBe(false);
    // A Clinical Summary (S32) holds no answer of its own, so it never goes to the library either.
    for (const type of ["panel", "html", "expression", "profile", "bmi", "signaturepad", "file", "clinicalsummary"]) expect(canAskFresh({ type, name: "q" })).toBe(false);
  });
});

describe("keyPlace", () => {
  it("finds questions inside a repeating group, and clinician-only groups and pages around them", () => {
    const doc: ChapterJson = {
      pages: [
        {
          name: "p_1",
          elements: [
            { type: "text", name: "q_top" },
            { type: "panel", name: "g_1", clinicianOnly: true, elements: [{ type: "text", name: "q_in_group" }] },
            { type: "paneldynamic", name: "q_list", templateElements: [{ type: "panel", name: "g_2", elements: [{ type: "text", name: "q_deep" }] }] },
          ],
        },
        { name: "p_2", clinicianOnly: true, elements: [{ type: "text", name: "q_clin_page" }] },
      ],
    };
    expect(["q_top", "q_in_group", "q_list", "q_deep", "q_clin_page", "q_missing"].map((n) => keyPlace(doc, n))).toEqual([
      { inList: false, clinicianOnly: false },
      { inList: false, clinicianOnly: true },
      { inList: false, clinicianOnly: false },
      { inList: true, clinicianOnly: false },
      { inList: false, clinicianOnly: true },
      { inList: false, clinicianOnly: false },
    ]);
  });
});
