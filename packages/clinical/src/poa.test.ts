import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { poaSummary, reportedBmi } from "./poa";

const out = (id: string, text: string, category = "Cardiovascular") => [{ id, codes: [], note: { text, category } }];
const yesNo = (name: string, title: string, note?: string) => ({
  type: "radiogroup", name, title,
  choices: [{ value: `${name}_y`, text: "Yes", ...(note ? { clinicalOutputs: out(`out_${name}`, note) } : {}) }, { value: `${name}_n`, text: "No" }],
});

const about: ChapterJson = {
  pages: [
    {
      name: "p_heart", title: "Heart and blood", clinicalSummary: true,
      elements: [yesNo("q_angina", "Do you have angina?", "Angina - unspecified"), yesNo("q_anaemia", "Anaemia?", "Anemia")],
    },
    { name: "p_meds", title: "Medications", elements: [{ type: "medication", name: "q_meds", title: "Prescribed medications" }] },
    { name: "p_bmi", title: "BMI", elements: [{ type: "bmi", name: "q_bmi", title: "BMI" }] },
    { name: "p_empty", title: "Nothing here", elements: [yesNo("q_x", "X?")] },
    { name: "p_clin", title: "Clinician", elements: [{ type: "comment", name: "q_clin", title: "Clinician note", clinicianOnly: true }] },
  ],
};
const assessment: ChapterJson = { chapterVisibleIf: "{q_angina} = 'q_angina_y'", pages: [{ name: "p_a", title: "Angina follow-up", elements: [yesNo("q_stable", "Stable?", "Stable angina")] }] };
const sets = [
  { id: "s1", name: "About you", content: about },
  { id: "s2", name: "Angina", content: assessment },
];
const jo = { name: "Jo Bloggs", age: 51, sex: "male" };

const patient = {
  s1: { q_angina: "q_angina_n", q_anaemia: "q_anaemia_y", q_meds: [{ name: "Omeprazole", dosage: "20mg", frequency: "Daily" }], q_bmi: { height: 165, weight: 74 }, q_x: "q_x_n" },
};
const clinician = {
  s1: { ...patient.s1, q_angina: "q_angina_y", q_clin: "seen", cs_comments_p_heart: "  On GTN spray. " },
  s2: { q_stable: "q_stable_y" },
};

describe("poaSummary", () => {
  it("builds the validated summary from the clinician's answers, marking corrections", () => {
    const poa = poaSummary(sets, clinician, "clinician", jo, patient);
    expect(poa.map((s) => s.name)).toEqual(["About you", "Angina"]); // Angina is shown: the clinician said yes
    const [heart, meds, bmi] = poa[0].pages;
    expect(heart.title).toBe("Heart and blood");
    expect(heart.notes).toEqual([
      { text: "Angina - unspecified", questionId: "q_angina", corrected: true },
      { text: "Anemia", questionId: "q_anaemia", corrected: false },
    ]);
    expect(heart.comments).toBe("On GTN spray.");
    expect(heart.changes).toEqual([{ questionId: "q_angina", title: "Do you have angina?", patient: "No", clinician: "Yes" }]);
    expect(meds.captures).toEqual([{ questionId: "q_meds", kind: "medication", title: "Prescribed medications", rows: [{ name: "Omeprazole", dosage: "20mg", frequency: "Daily" }] }]);
    expect(bmi.captures[0].rows[0]).toMatchObject({ height: 165, weight: 74, bmi: 27.2 });
    expect(reportedBmi(poa)).toBe(27.2);
    // Pages with nothing to show are left out; a clinician-only answer isn't a correction.
    expect(poa[0].pages.map((p) => p.name)).toEqual(["p_heart", "p_meds", "p_bmi"]);
    expect(poa[1].pages[0].notes[0].text).toBe("Stable angina");
  });

  it("builds the patient's original answers without the clinician's", () => {
    const poa = poaSummary(sets, patient, "patient", jo);
    expect(poa.map((s) => s.name)).toEqual(["About you"]); // the patient said no angina
    const heart = poa[0].pages[0];
    expect(heart.notes.map((n) => n.text)).toEqual(["Anemia"]);
    expect(heart.comments).toBe("");
    expect(heart.changes).toEqual([]);
  });
});
