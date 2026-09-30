import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import { computeOutputs, formatAnswer, validateOutput } from "./outputs";
import { registerClinicalProperties } from "./properties";
import type { ClinicalOutput } from "./types";
import { setViewer, stripClinicianOnly } from "./viewer";

registerClinicalProperties();

const out = (id: string, extra: Partial<ClinicalOutput> = {}): ClinicalOutput => ({ id, codes: [], ...extra });
const smoker = out("out_smoker", {
  codes: [{ set: "SNOMED", code: "77176002", display: "Smoker" }],
  note: { text: "Current smoker", category: "Lifestyle" },
  flag: "amber",
});
const perDay = out("out_perday", { note: { text: "Smokes {answer} per day", category: "Lifestyle" } });

/** The demo chapter: smoking, how many (shown when Yes), a clinician-only question. */
const chapter = {
  pages: [
    {
      name: "page1",
      elements: [
        {
          type: "radiogroup",
          name: "q_smoke",
          title: "Do you smoke?",
          choices: [
            { value: "o_yes", text: "Yes", clinicalOutputs: [smoker] },
            { value: "o_no", text: "No" },
          ],
        },
        { type: "text", inputType: "number", name: "q_perday", title: "How many per day?", visibleIf: "{q_smoke} = 'o_yes'", clinicalOutputs: [perDay] },
        {
          type: "text",
          name: "q_obs",
          title: "Clinician observation",
          clinicianOnly: true,
          clinicalOutputs: [out("out_obs", { note: { text: "Observed: {answer}", category: "Unassigned" } })],
        },
      ],
    },
  ],
};

const survey = (json: object, data: Record<string, unknown>, viewer?: "patient" | "clinician") => {
  const m = new Model(json);
  if (viewer) setViewer(m, viewer);
  m.data = data;
  return m;
};
const ids = (m: Model) => computeOutputs(m).outputs.map((o) => o.output.id);

describe("computeOutputs", () => {
  it("produces option and question outputs, with {answer} filled in", () => {
    const { outputs } = computeOutputs(survey(chapter, { q_smoke: "o_yes", q_perday: 20 }));
    expect(outputs).toEqual([
      { questionId: "q_smoke", questionTitle: "Do you smoke?", answerId: "o_yes", answerLabel: "Yes", output: smoker, noteText: "Current smoker" },
      { questionId: "q_perday", questionTitle: "How many per day?", answerLabel: "20", output: perDay, noteText: "Smokes 20 per day" },
    ]);
  });

  it("ignores a hidden question's answer (LOG-09)", () => {
    // q_perday keeps its value in survey.data once hidden.
    const m = survey(chapter, { q_smoke: "o_yes", q_perday: 20 });
    m.setValue("q_smoke", "o_no");
    expect(m.data.q_perday).toBe(20);
    expect(ids(m)).toEqual([]);
  });

  it("ignores answers inside a hidden group, including nested groups", () => {
    const json = {
      elements: [
        { type: "radiogroup", name: "q_show", choices: [{ value: "o_y", text: "Y" }, { value: "o_n", text: "N" }] },
        {
          type: "panel",
          name: "g_outer",
          visibleIf: "{q_show} = 'o_y'",
          elements: [
            { type: "text", name: "q_a", clinicalOutputs: [out("out_a")] },
            { type: "panel", name: "g_inner", elements: [{ type: "text", name: "q_b", clinicalOutputs: [out("out_b")] }] },
          ],
        },
      ],
    };
    expect(ids(survey(json, { q_show: "o_y", q_a: "x", q_b: "y" }))).toEqual(["out_a", "out_b"]);
    expect(ids(survey(json, { q_show: "o_n", q_a: "x", q_b: "y" }))).toEqual([]);
  });

  it("counts clinician-only answers for clinicians only (CLN-14)", () => {
    const data = { q_smoke: "o_no", q_obs: "pale" };
    expect(ids(survey(chapter, data, "clinician"))).toEqual(["out_obs"]);
    // Even if the full JSON reaches a patient model, clinician-only questions don't count.
    expect(ids(survey(chapter, data, "patient"))).toEqual([]);
    expect(ids(survey(stripClinicianOnly(chapter), data, "patient"))).toEqual([]);
  });

  it("skips questions inside a clinician-only group for patients", () => {
    const json = {
      elements: [{ type: "panel", name: "g_clin", clinicianOnly: true, elements: [{ type: "text", name: "q_a", clinicalOutputs: [out("out_a")] }] }],
    };
    expect(ids(survey(json, { q_a: "x" }, "patient"))).toEqual([]);
    expect(ids(survey(json, { q_a: "x" }, "clinician"))).toEqual(["out_a"]);
  });

  it("produces the outputs of an exclusive 'None of these' option", () => {
    const json = {
      elements: [
        {
          type: "checkbox",
          name: "q_cond",
          choices: [
            { value: "o_asth", text: "Asthma", clinicalOutputs: [out("out_asth")] },
            { value: "o_none", text: "None of these", isExclusive: true, clinicalOutputs: [out("out_none")] },
          ],
        },
      ],
    };
    expect(ids(survey(json, { q_cond: ["o_none"] }))).toEqual(["out_none"]);
  });

  it("produces outputs for each chosen option of a multi-select, with all labels as {answer}", () => {
    const json = {
      elements: [
        {
          type: "checkbox",
          name: "q_meds",
          choices: [
            { value: "o_asp", text: "Aspirin", clinicalOutputs: [out("out_asp", { note: { text: "Takes {answer}", category: "Medication" } })] },
            { value: "o_war", text: "Warfarin", clinicalOutputs: [out("out_war")] },
            { value: "o_met", text: "Metformin", clinicalOutputs: [out("out_met")] },
          ],
        },
      ],
    };
    const { outputs } = computeOutputs(survey(json, { q_meds: ["o_asp", "o_met"] }));
    expect(outputs.map((o) => [o.output.id, o.answerLabel])).toEqual([
      ["out_asp", "Aspirin"],
      ["out_met", "Metformin"],
    ]);
    expect(outputs[0].noteText).toBe("Takes Aspirin, Metformin");
  });

  it("formats a date answer as dd/MM/yyyy", () => {
    const json = {
      elements: [{ type: "text", inputType: "date", name: "q_op", clinicalOutputs: [out("out_op", { note: { text: "Last operation {answer}", category: "Anaesthetic" } })] }],
    };
    expect(computeOutputs(survey(json, { q_op: "2019-07-04" })).outputs[0].noteText).toBe("Last operation 04/07/2019");
  });

  it("produces several outputs on one option, in order", () => {
    const json = {
      elements: [{ type: "radiogroup", name: "q_a", choices: [{ value: "o_y", text: "Yes", clinicalOutputs: [out("out_1"), out("out_2"), out("out_3")] }] }],
    };
    expect(ids(survey(json, { q_a: "o_y" }))).toEqual(["out_1", "out_2", "out_3"]);
  });

  it("produces nothing for unanswered questions", () => {
    expect(ids(survey(chapter, {}))).toEqual([]);
  });

  it("suggests the highest ASA grade (CLN-17)", () => {
    const json = {
      elements: [
        {
          type: "checkbox",
          name: "q_a",
          choices: [
            { value: "o_1", text: "A", clinicalOutputs: [out("a", { asa: { grade: "II", emergency: false } })] },
            { value: "o_2", text: "B", clinicalOutputs: [out("b", { asa: { grade: "IV", emergency: false } })] },
            { value: "o_3", text: "C", clinicalOutputs: [out("c", { asa: { grade: "IV", emergency: true } })] },
          ],
        },
      ],
    };
    expect(computeOutputs(survey(json, { q_a: ["o_1", "o_2"] })).suggestedAsa).toEqual({ grade: "IV", emergency: false });
    expect(computeOutputs(survey(json, { q_a: ["o_1", "o_2", "o_3"] })).suggestedAsa).toEqual({ grade: "IV", emergency: true });
    expect(computeOutputs(survey(chapter, { q_smoke: "o_yes" })).suggestedAsa).toBeUndefined();
  });
});

describe("validateOutput (CLN-06)", () => {
  it("needs a code or a note, and a note needs a category", () => {
    expect(validateOutput(out("x"))).toEqual(["Add at least one code or a clinical note."]);
    expect(validateOutput(out("x", { note: { text: "Note", category: "" } }))).toEqual(["Choose a category for the note."]);
    expect(validateOutput(smoker)).toEqual([]);
    expect(validateOutput(out("x", { codes: smoker.codes }))).toEqual([]);
  });
});

describe("formatAnswer", () => {
  it("leaves text and numbers as they are and maps option IDs to labels", () => {
    const m = new Model(chapter);
    expect(formatAnswer(m.getQuestionByName("q_perday"), 7)).toBe("7");
    expect(formatAnswer(m.getQuestionByName("q_obs"), "pale")).toBe("pale");
    expect(formatAnswer(m.getQuestionByName("q_smoke"), "o_yes")).toBe("Yes");
    expect(formatAnswer(m.getQuestionByName("q_smoke"), undefined)).toBe("");
  });
});
