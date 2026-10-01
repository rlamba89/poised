import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import { registerClinicalProperties } from "./properties";
import { newElement } from "./doc";

registerClinicalProperties();

describe("Lifebox question types", () => {
  it("loads Medication, Admissions, BMI and Profile from the editor's JSON", () => {
    const m = new Model({
      pages: [{ name: "p_a", clinicalSummary: true, elements: ["medication", "admissions", "bmi", "profile"].map((k) => newElement(k as never, new Set())) }],
    });
    expect(m.getAllQuestions().map((q) => q.getType())).toEqual(["medication", "admissions", "bmi", "profile"]);
    expect(m.pages[0].getPropertyValue("clinicalSummary")).toBe(true);
    expect(m.getAllQuestions()[0].getPropertyValue("medicationType")).toBe("prescribed");
  });

  it("works out the BMI from height and weight", () => {
    const m = new Model({ elements: [{ type: "bmi", name: "q_bmi" }] });
    m.setValue("q_bmi", { height: 180, weight: 81 });
    expect(m.getValue("q_bmi").bmi).toBe(25);
  });

  it("keeps medication rows as the answer", () => {
    const m = new Model({ elements: [{ type: "medication", name: "q_med" }] });
    m.setValue("q_med", [{ name: "Aspirin", dosage: "75mg", frequency: "daily" }]);
    expect(m.getValue("q_med")).toHaveLength(1);
  });

  it("keeps the Yes / No and Section flags through a save", () => {
    const json = {
      pages: [{ name: "p_a", elements: [
        { ...newElement("yesno", new Set()) },
        { ...newElement("section", new Set()), showAsHeading: false },
      ] }],
    };
    const saved = new Model(json).toJSON();
    expect(saved.pages[0].elements[0].yesNo).toBe(true);
    expect(saved.pages[0].elements[1].patientPage).toBe(true);
    expect(saved.pages[0].elements[1].showAsHeading).toBe(false);
  });
});
