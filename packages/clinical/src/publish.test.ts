import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { publishProblems } from "./publish";
import type { TestCase } from "./testcases";

const smoker = (testCases: TestCase[] = [], visibleIf?: string): ChapterJson => ({
  pages: [
    {
      name: "p_1",
      elements: [
        {
          type: "radiogroup", name: "q_smoke", title: "Do you smoke?",
          choices: [
            { value: "o_y", text: "Yes", clinicalOutputs: [{ id: "out_1", codes: [], note: { text: "Smoker", category: "Lifestyle" } }] },
            { value: "o_n", text: "No" },
          ],
        },
        { type: "text", name: "q_many", title: "How many a day?", ...(visibleIf ? { visibleIf } : {}) },
      ],
    },
  ],
  ...(testCases.length ? { testCases } : {}),
});

const smokes = (expected: string[]): TestCase => ({
  id: "tc_1", name: "Smoker", viewer: "patient", patient: { name: "Sam" }, answers: { q_smoke: "o_y" }, expected,
});

describe("publishProblems", () => {
  it("passes a clean questionnaire with passing test cases", () => {
    const doc = smoker([smokes(["Note (Lifestyle): Smoker"])], "{q_smoke} = 'o_y'");
    expect(publishProblems([{ name: "Lifestyle", doc }])).toEqual([]);
  });

  it("reports a failing test case", () => {
    const doc = smoker([smokes([])]);
    expect(publishProblems([{ name: "Lifestyle", doc }])).toEqual([{ set: "Lifestyle", message: "Test case “Smoker” fails" }]);
  });

  it("reports logic problems with the Question Set's name", () => {
    const doc = smoker([], "{q_gone} = 'o_y'");
    const problems = publishProblems([{ name: "A" , doc: smoker() }, { name: "Lifestyle", doc }]);
    expect(problems).toHaveLength(1);
    expect(problems[0].set).toBe("Lifestyle");
  });
});
