import { describe, expect, it } from "vitest";
import { Model, type QuestionCheckboxModel } from "survey-core";
import { isClinicianOnly, outputsOf, registerClinicalProperties, setOutputs } from "./properties";
import type { ClinicalOutput } from "./types";

registerClinicalProperties();

const smoker: ClinicalOutput = {
  id: "out_1",
  codes: [{ set: "SNOMED", code: "77176002", display: "Smoker" }],
  note: { text: "Current smoker", category: "Lifestyle" },
  flag: "amber",
};

describe("clinical properties", () => {
  it("round-trips outputs as a plain array in JSON", () => {
    const json = {
      elements: [
        { type: "text", name: "q_perday", clinicalOutputs: [smoker] },
        {
          type: "checkbox",
          name: "q_cond",
          clinicianOnly: true,
          choices: [{ value: "o_none", text: "None of these", isExclusive: true, clinicalOutputs: [smoker] }],
        },
      ],
    };
    const survey = new Model(json);
    expect(outputsOf(survey.getQuestionByName("q_perday"))).toEqual([smoker]);
    const cond = survey.getQuestionByName("q_cond") as QuestionCheckboxModel;
    expect(outputsOf(cond.choices[0])).toEqual([smoker]);
    expect(isClinicianOnly(cond)).toBe(true);
    expect(survey.toJSON()).toEqual({ pages: [{ name: "page1", elements: json.elements }] });
  });

  it("omits the property when there are no outputs", () => {
    const survey = new Model({ elements: [{ type: "text", name: "q_a" }] });
    const q = survey.getQuestionByName("q_a");
    setOutputs(q, [smoker]);
    setOutputs(q, []);
    expect(outputsOf(q)).toEqual([]);
    expect(survey.toJSON().pages[0].elements[0]).toEqual({ type: "text", name: "q_a" });
  });

  it("does not share the caller's array", () => {
    const survey = new Model({ elements: [{ type: "text", name: "q_a" }] });
    const q = survey.getQuestionByName("q_a");
    const items = [smoker];
    setOutputs(q, items);
    items.pop();
    expect(outputsOf(q)).toEqual([smoker]);
  });

  it("keeps the value when it is written back through the property (Creator undo/redo)", () => {
    const survey = new Model({ elements: [{ type: "text", name: "q_a" }] });
    const q = survey.getQuestionByName("q_a") as unknown as Record<string, unknown>;
    setOutputs(q as never, [smoker]);
    const box = q.clinicalOutputs;
    q.clinicalOutputs = undefined;
    expect(outputsOf(q as never)).toEqual([]);
    q.clinicalOutputs = box;
    expect(outputsOf(q as never)).toEqual([smoker]);
  });
});
