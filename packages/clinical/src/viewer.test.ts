import { describe, expect, it } from "vitest";
import { stripClinicianOnly } from "./viewer";

describe("stripClinicianOnly", () => {
  const json = {
    pages: [
      {
        name: "page1",
        elements: [
          { type: "text", name: "q_a" },
          { type: "html", name: "q_note", clinicianOnly: true, html: "<p>For clinicians</p>" },
          {
            type: "panel",
            name: "g_mixed",
            elements: [
              { type: "text", name: "q_b" },
              { type: "text", name: "q_c", clinicianOnly: true },
              { type: "panel", name: "g_clin", clinicianOnly: true, elements: [{ type: "text", name: "q_d" }] },
            ],
          },
          { type: "text", name: "q_e", clinicianOnly: false },
        ],
      },
      { name: "page2", elements: [{ type: "text", name: "q_f", clinicianOnly: true }] },
    ],
  };

  it("removes clinician-only questions and groups at any depth", () => {
    expect(stripClinicianOnly(json)).toEqual({
      pages: [
        {
          name: "page1",
          elements: [
            { type: "text", name: "q_a" },
            { type: "panel", name: "g_mixed", elements: [{ type: "text", name: "q_b" }] },
            { type: "text", name: "q_e", clinicianOnly: false },
          ],
        },
        { name: "page2", elements: [] },
      ],
    });
  });

  it("does not change the original", () => {
    const before = JSON.stringify(json);
    stripClinicianOnly(json);
    expect(JSON.stringify(json)).toBe(before);
  });

  it("handles top-level elements and an empty survey", () => {
    expect(stripClinicianOnly({ elements: [{ name: "q_a", clinicianOnly: true }, { name: "q_b" }] })).toEqual({
      elements: [{ name: "q_b" }],
    });
    expect(stripClinicianOnly({})).toEqual({});
  });
});
