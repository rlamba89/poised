import { describe, expect, it } from "vitest";
import { registerClinicalProperties } from "@poised/clinical";
import { clinicianModel } from "./clinicianView";

registerClinicalProperties();

describe("the clinician view", () => {
  it("never draws a chapter's survey-level HTML (S35 verification 1, F3)", () => {
    const m = clinicianModel({ completedHtml: "<img src=x onerror=alert(1)>", loadingHtml: "<b>x</b>", pages: [{ name: "p1", elements: [] }] });
    expect(m.completedHtml).not.toContain("<img");
    expect(m.loadingHtml).not.toContain("<b>");
  });
});
