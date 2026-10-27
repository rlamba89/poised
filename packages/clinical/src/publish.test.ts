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
    const other: ChapterJson = { pages: [{ name: "p_2", elements: [{ type: "text", name: "q_other", title: "Other" }] }] }; // keys unique in the HQ (S23)
    const problems = publishProblems([{ name: "A", doc: other }, { name: "Lifestyle", doc }]);
    expect(problems).toHaveLength(1);
    expect(problems[0].set).toBe("Lifestyle");
  });

  // S32 (SGN-04): two clinical summaries covering the same Group, Section or page.
  it("reports two summaries in one Group, naming the set", () => {
    const doc: ChapterJson = { pages: [{ name: "p_1", title: "Heart", elements: [
      { type: "panel", name: "g_b", title: "Breathing", elements: [
        { type: "text", name: "q_b", title: "Breathless?" },
        { type: "clinicalsummary", name: "cs_one", clinicianOnly: true },
        { type: "clinicalsummary", name: "cs_two", clinicianOnly: true },
      ] },
      { type: "clinicalsummary", name: "cs_page", clinicianOnly: true },
    ] }] };
    expect(publishProblems([{ name: "About you", doc }])).toEqual([
      { set: "About you", message: "Two clinical summaries cover the same Group “Breathing”: keep one" },
    ]);
  });

  // S37 (Rahul, 26 Oct): a Clinical Summary can't have a condition.
  it("reports a summary with a condition, naming the set and the summary", () => {
    const doc: ChapterJson = { pages: [{ name: "p_1", title: "Heart", elements: [
      { type: "radiogroup", name: "q_b", title: "Breathless?", choices: ["Yes", "No"] },
      { type: "clinicalsummary", name: "cs_page", title: "Heart checks", clinicianOnly: true, visibleIf: "{q_b} = 'Yes'" },
    ] }] };
    expect(publishProblems([{ name: "About you", doc }])).toEqual([
      { set: "About you", message: "The clinical summary “Heart checks” has a condition: remove it (a summary is always shown)" },
    ]);
  });
  it("passes a summary without a condition", () => {
    const doc: ChapterJson = { pages: [{ name: "p_1", title: "Heart", elements: [
      { type: "radiogroup", name: "q_b", title: "Breathless?", choices: ["Yes", "No"] },
      { type: "clinicalsummary", name: "cs_page", clinicianOnly: true },
    ] }] };
    expect(publishProblems([{ name: "About you", doc }])).toEqual([]);
  });
});

describe("a patient's condition on something patients can't see (S44a)", () => {
  const sleep: ChapterJson = {
    pages: [{
      name: "p_1", title: "Sleep", elements: [
        { type: "radiogroup", name: "q_snore", title: "Do you snore?", choices: [{ value: "o_y", text: "Yes", score: 1 }, { value: "o_n", text: "No", score: 0 }] },
        {
          type: "expression", name: "q_risk", title: "Sleep risk", clinicianOnly: true, calcKind: "score", expression: "iif(allAnswered('q_snore'), score('q_snore'), '')",
          bands: [{ id: "b_low", label: "Low", max: 0 }, { id: "b_high", label: "High" }],
        },
        { type: "comment", name: "q_why", title: "Why do you snore?", visibleIf: "band('q_risk') = 'b_high'" },
        { type: "comment", name: "q_note", title: "Nurse's note", clinicianOnly: true, visibleIf: "band('q_risk') = 'b_high'" },
      ],
    }],
  };
  const followUp = (cond: string): ChapterJson => ({ chapterVisibleIf: cond, pages: [{ name: "p_1", elements: [{ type: "text", name: "q_more", title: "More" }] }] });
  const nurse: ChapterJson = { pages: [{ name: "p_1", elements: [{ type: "radiogroup", name: "q_neck", title: "Neck over 40 cm?", choices: [{ value: "o_y", text: "Yes" }] }] }] };

  it("refuses a patient Question Set conditioned on a clinician-only score, saying how to fix it", () => {
    const p = publishProblems([{ name: "Sleep", doc: sleep, audience: "patient" }, { name: "Sleep follow-up", doc: followUp("band('q_risk') = 'b_high'"), audience: "patient" }]);
    expect(p).toContainEqual({
      set: "Sleep follow-up",
      message: "“Sleep follow-up” shows only when Sleep risk band is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set.",
    });
  });

  it("refuses a patient question conditioned on a clinician-only element in its own set, but not a clinician-only one", () => {
    const messages = publishProblems([{ name: "Sleep", doc: sleep, audience: "patient" }]).map((x) => x.message);
    expect(messages).toEqual([
      "“Why do you snore?” shows only when Sleep risk band is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set.",
    ]);
  });

  it("refuses a patient set conditioned on a clinician Question Set's question", () => {
    const p = publishProblems([{ name: "Nurse checks", doc: nurse, audience: "clinician" }, { name: "Neck", doc: followUp("{q_neck} = 'o_y'"), audience: "patient" }]);
    expect(p.map((x) => x.message)).toEqual([
      "“Neck” shows only when Neck over 40 cm? is Yes, but patients can't see Neck over 40 cm? Move the condition to a clinician Question Set, or what it tests to a patient Question Set.",
    ]);
  });

  it("allows the same condition on a clinician Question Set, and on a score patients see", () => {
    expect(publishProblems([{ name: "Sleep", doc: sleep, audience: "patient" }, { name: "Nurse", doc: followUp("band('q_risk') = 'b_high'"), audience: "clinician" }]).map((x) => x.set)).toEqual(["Sleep"]);
    const visible = structuredClone(sleep);
    delete visible.pages![0].elements![1].clinicianOnly;
    expect(publishProblems([{ name: "Sleep", doc: visible, audience: "patient" }, { name: "Follow-up", doc: followUp("band('q_risk') = 'b_high'"), audience: "patient" }])).toEqual([]);
  });

  it("names a page's and an option's condition", () => {
    const doc = structuredClone(sleep);
    doc.pages![0].elements!.splice(2, 2);
    doc.pages!.push({ name: "p_2", title: "More sleep", visibleIf: "band('q_risk') = 'b_high'", elements: [
      { type: "radiogroup", name: "q_pillows", title: "Pillows?", choices: [{ value: "o_1", text: "One" }, { value: "o_3", text: "Three", visibleIf: "band('q_risk') = 'b_high'" }] },
    ] });
    expect(publishProblems([{ name: "Sleep", doc, audience: "patient" }]).map((x) => x.message.split(" shows")[0])).toEqual(["Page “More sleep”", "“Pillows?” option “Three”"]);
  });
});
