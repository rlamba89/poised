import { describe, expect, it } from "vitest";
import type { ChapterJson, ElementJson } from "./doc";
import { estimate, estimatedMinutes, estimateText } from "./estimate";

const YT = "https://www.youtube.com/watch?v=aqz-KE-bpKQ";
const q = (name: string, extra: Partial<ElementJson> = {}): ElementJson => ({ type: "text", name, title: name, ...extra });
const set = (...elements: ElementJson[]): ChapterJson => ({ pages: [{ name: "p1", elements }] });

describe("estimate: what the patient gets", () => {
  it("counts each question the patient sees", () => {
    expect(estimate(set(q("a"), q("b"), { type: "radiogroup", name: "c", choices: ["Yes", "No"] }))).toEqual({ questions: 3, videoSeconds: 0 });
  });

  it("counts the questions inside Groups, Sections and repeating groups (a repeating group's once)", () => {
    const doc = set(
      { type: "panel", name: "g1", elements: [q("a"), { type: "panel", name: "g2", patientPage: true, elements: [q("b")] }] },
      { type: "paneldynamic", name: "g3", templateElements: [q("c"), q("d")] },
    );
    expect(estimate(doc).questions).toBe(4);
  });

  it("counts the questions on every page", () => {
    expect(estimate({ pages: [{ name: "p1", elements: [q("a")] }, { name: "p2", elements: [q("b"), q("c")] }] }).questions).toBe(3);
  });

  it("leaves out clinician-only questions, Groups and pages, and Clinical Summaries", () => {
    const doc: ChapterJson = {
      pages: [
        { name: "p1", elements: [q("a"), q("secret", { clinicianOnly: true }), { type: "panel", name: "g", clinicianOnly: true, elements: [q("b"), q("c")] },
          { type: "clinicalsummary", name: "cs_1", clinicianOnly: true }, { type: "clinicalsummary", name: "cs_2" }] },
        { name: "p2", clinicianOnly: true, elements: [q("d"), q("e")] },
      ],
    };
    expect(estimate(doc).questions).toBe(1);
  });

  it("leaves out what isn't answered: Information, the profile, calculations and hidden questions", () => {
    const doc = set(q("a"), { type: "html", name: "i", markdown: "Hello" }, { type: "profile", name: "pr" }, { type: "expression", name: "calc", expression: "1" },
      q("hidden", { visible: false }));
    expect(estimate(doc).questions).toBe(1);
  });

  it("counts questions shown only under a condition (the patient might see them)", () => {
    expect(estimate(set(q("a"), q("b", { visibleIf: "{a} = 'x'" }))).questions).toBe(2);
  });

  it("adds each video's length; the video isn't a question, even with its tick", () => {
    const doc = set(q("a"), { type: "video", name: "v1", videoUrl: YT, videoLength: 125, watchTick: true }, { type: "video", name: "v2", videoUrl: YT, videoLength: 60 },
      { type: "video", name: "v3", videoUrl: YT }, { type: "video", name: "v4", videoUrl: YT, videoLength: 30, clinicianOnly: true });
    expect(estimate(doc)).toEqual({ questions: 1, videoSeconds: 185 });
  });

  it("an empty set or none at all", () => {
    expect(estimate({})).toEqual({ questions: 0, videoSeconds: 0 });
    expect(estimate(set())).toEqual({ questions: 0, videoSeconds: 0 });
  });
});

describe("estimatedMinutes", () => {
  it("is the seconds per question for each question plus the videos, rounded up to whole minutes", () => {
    expect(estimatedMinutes(set(q("a"), q("b"), q("c"), q("d")), 15)).toBe(1); // 60 s
    expect(estimatedMinutes(set(q("a"), q("b"), q("c"), q("d"), q("e")), 15)).toBe(2); // 75 s
    expect(estimatedMinutes(set(q("a"), { type: "video", name: "v", videoUrl: YT, videoLength: 200 }), 15)).toBe(4); // 215 s
  });

  it("follows the seconds per question it's given (the deployment's setting)", () => {
    const doc = set(q("a"), q("b"), q("c"), q("d"));
    expect(estimatedMinutes(doc, 30)).toBe(2);
    expect(estimatedMinutes(doc, 10)).toBe(1);
  });

  it("five more questions raise it", () => {
    const ten = set(...Array.from({ length: 10 }, (_, i) => q(`q${i}`)));
    const fifteen = set(...Array.from({ length: 15 }, (_, i) => q(`q${i}`)));
    expect(estimatedMinutes(ten, 15)).toBe(3);
    expect(estimatedMinutes(fifteen, 15)).toBe(4);
  });

  it("is at least 1 minute", () => {
    expect(estimatedMinutes(set(), 15)).toBe(1);
    expect(estimatedMinutes(set(q("a")), 15)).toBe(1);
  });

  it("is undefined without a usable seconds per question (the config didn't load)", () => {
    expect(estimatedMinutes(set(q("a")), undefined)).toBeUndefined();
    expect(estimatedMinutes(set(q("a")), 0)).toBeUndefined();
    expect(estimatedMinutes(set(q("a")), Number.NaN)).toBeUndefined();
  });
});

describe("estimateText", () => {
  it("says about how long", () => {
    expect(estimateText(1)).toBe("About 1 minute");
    expect(estimateText(4)).toBe("About 4 minutes");
    expect(estimateText(undefined)).toBe("");
  });
});
