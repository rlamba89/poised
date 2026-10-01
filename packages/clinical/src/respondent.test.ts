import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { Model } from "survey-core";
import { ageFrom, describeAnswer, isSetComplete, patientVariables, sameAnswer, shownSets } from "./respondent";

const smoking: ChapterJson = {
  pages: [{ name: "p_1", elements: [{ type: "radiogroup", name: "q_smoke", isRequired: true, choices: [{ value: "o_y", text: "Yes" }, { value: "o_n", text: "No" }] }] }],
};
const clinicianNote: ChapterJson = {
  pages: [{ name: "p_1", elements: [{ type: "comment", name: "q_note", isRequired: true, clinicianOnly: true }] }],
};
const pat = { name: "Jo Bloggs", age: 51, sex: "male" };

describe("respondent", () => {
  it("works out age on the day", () => {
    expect(ageFrom("1974-12-11", new Date(2026, 9, 1))).toBe(51);
    expect(ageFrom("1974-10-01", new Date(2026, 9, 1))).toBe(52);
    expect(patientVariables({ firstName: "Jo", lastName: "Bloggs", dateOfBirth: "1974-12-11", sex: "male" }, new Date(2026, 9, 1))).toEqual(pat);
  });

  it("is complete when every required visible question is answered", () => {
    expect(isSetComplete(smoking, {}, "patient", pat)).toBe(false);
    expect(isSetComplete(smoking, { q_smoke: "o_n" }, "patient", pat)).toBe(true);
    // A required clinician-only question doesn't hold the patient up, but does hold up the clinician.
    expect(isSetComplete(clinicianNote, {}, "patient", pat)).toBe(true);
    expect(isSetComplete(clinicianNote, {}, "clinician", pat)).toBe(false);
  });

  it("shows a Question Set only when its condition is met by earlier sets", () => {
    const sets = [
      { id: "a", content: smoking },
      { id: "b", content: { ...smoking, chapterVisibleIf: "{q_smoke} = 'o_y'" } },
      { id: "c", content: { ...smoking, chapterVisibleIf: "{patientAge} >= 65" } },
    ];
    expect(shownSets(sets, { a: { q_smoke: "o_n" } }, "patient", pat).map((s) => s.id)).toEqual(["a"]);
    expect(shownSets(sets, { a: { q_smoke: "o_y" } }, "patient", pat).map((s) => s.id)).toEqual(["a", "b"]);
    expect(shownSets(sets, {}, "patient", { ...pat, age: 70 }).map((s) => s.id)).toEqual(["a", "c"]);
  });
});

describe("corrections", () => {
  it("compares answers ignoring key order and empty values", () => {
    expect(sameAnswer({ a: 1, b: 2 }, { b: 2, a: 1 })).toBe(true);
    expect(sameAnswer(undefined, "")).toBe(true);
    expect(sameAnswer(undefined, [])).toBe(true);
    expect(sameAnswer("o_y", "o_n")).toBe(false);
    expect(sameAnswer(["o_a", "o_b"], ["o_b", "o_a"])).toBe(false); // order of ticks is kept as given
  });

  it("describes answers in words", () => {
    const m = new Model({
      elements: [
        { type: "radiogroup", name: "q_s", choices: [{ value: "o_y", text: "Yes" }] },
        { type: "matrixdynamic", name: "q_meds", columns: [{ name: "name" }, { name: "dose" }] },
      ],
    });
    expect(describeAnswer(m.getQuestionByName("q_s"), "o_y")).toBe("Yes");
    expect(describeAnswer(m.getQuestionByName("q_s"), undefined)).toBe("no answer");
    expect(describeAnswer(m.getQuestionByName("q_meds"), [{ name: "Omeprazole", dose: "20mg" }, { name: "Paracetamol" }])).toBe("Omeprazole 20mg; Paracetamol");
  });
});
