// Outputs saved without `codes` or a note `category`, as the e2e fixtures and older content have
// them ({id, note: {text}}). Every reader must cope (S08 test reports 2 and 3).
import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import type { ChapterJson } from "./doc";
import { computeOutputs, validateOutput } from "./outputs";
import { normalizeOutput } from "./properties";
import { publishProblems } from "./publish";
import { outputLines, runTestCase, type TestCase } from "./testcases";
import type { ClinicalOutput } from "./types";
import { registerClinicalProperties } from "./properties";

registerClinicalProperties();

// The shape of e2e/data/workflow-hq.json: a note with no category, and no codes at all.
const fixture = (testCases: TestCase[] = []): ChapterJson => ({
  pages: [
    {
      name: "p_1",
      elements: [
        {
          type: "radiogroup", name: "q_smokes", title: "Do you smoke?",
          choices: [
            { value: "o_yes", text: "Yes", clinicalOutputs: [{ id: "out_smoker", note: { text: "Smoker" } }] },
            { value: "o_no", text: "No", clinicalOutputs: [{ id: "out_bare" }] },
          ],
        },
        {
          type: "expression", name: "q_score", title: "Score", expression: "1",
          bands: [{ id: "b_1", label: "Low", clinicalOutputs: [{ id: "out_band", note: { text: "Low score" } }] }],
        },
        {
          type: "matrix", name: "q_grid", title: "Grid", rows: [{ value: "r1", text: "Row" }], columns: [{ value: "c1", text: "Col" }],
          cellOutputs: { r1: { c1: [{ id: "out_cell", note: { text: "Cell note" } }] } },
        },
      ],
    },
  ],
  ...(testCases.length ? { testCases } : {}),
} as unknown as ChapterJson);

const smoker = (expected: string[]): TestCase => ({
  id: "tc_1", name: "Smoker", viewer: "clinician", patient: { name: "Sam" }, answers: { q_smokes: "o_yes", q_grid: { r1: "c1" } }, expected,
});

describe("outputs without codes or a note category", () => {
  it("normalizes to no codes and an empty category", () => {
    expect(normalizeOutput({ id: "o", note: { text: "Smoker" } } as unknown as ClinicalOutput)).toEqual({
      id: "o", codes: [], note: { text: "Smoker", category: "" },
    });
    expect(normalizeOutput({ id: "o" } as unknown as ClinicalOutput)).toEqual({ id: "o", codes: [] });
  });

  it("computes them, from options, bands and grid cells, with codes always a list", () => {
    const m = new Model(fixture());
    m.data = { q_smokes: "o_yes", q_grid: { r1: "c1" } };
    const { outputs } = computeOutputs(m);
    expect(outputs.map((o) => o.output.id).sort()).toEqual(["out_band", "out_cell", "out_smoker"]);
    for (const o of outputs) expect(o.output.codes).toEqual([]);
  });

  it("writes test-case lines without a category in brackets", () => {
    const m = new Model(fixture());
    m.data = { q_smokes: "o_yes", q_grid: { r1: "c1" } };
    expect(outputLines(computeOutputs(m))).toEqual(["Note: Cell note", "Note: Low score", "Note: Smoker"]);
  });

  it("runs saved test cases and the publish check on them", () => {
    const tc = smoker(["Note: Cell note", "Note: Low score", "Note: Smoker"]);
    expect(runTestCase(fixture(), tc).pass).toBe(true);
    expect(publishProblems([{ name: "About you", doc: fixture([tc]) }])).toEqual([]);
  });

  it("validates them without throwing", () => {
    expect(validateOutput({ id: "o" } as unknown as ClinicalOutput)).toEqual(["Add at least one code or a clinical note."]);
    expect(validateOutput({ id: "o", note: { text: "Smoker" } } as unknown as ClinicalOutput)).toEqual(["Choose a category for the note."]);
  });
});
