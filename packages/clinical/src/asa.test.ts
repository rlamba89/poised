import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import { afterCodeAdded, afterCodeRemoved, gradePicked, suggestedAsaText, type AsaChoice } from "./asa";
import type { ChapterJson } from "./doc";
import { computeOutputs } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { chapterConditionProblems, givenEarlier } from "./questionSets";
import { publishProblems } from "./publish";
import { hqSuggestedAsa, shownSets } from "./respondent";
import { describeLogic, OPS, parseLogic, subjectsFor } from "./logic";
import type { ClinicalOutput } from "./types";

registerClinicalProperties();

const well: AsaChoice = { grade: "II", example: "Well-controlled diabetes", group: "Adults" };
const poorly: AsaChoice = { grade: "III", example: "Poorly controlled diabetes", group: "Adults" };
const copd: AsaChoice = { grade: "III", example: "COPD", group: "Adults" };

describe("the ASA guide in the editor (S39)", () => {
  it("fills in the grade when the code has one example, with its words as the reason", () => {
    expect(afterCodeAdded(undefined, [], [copd])).toEqual({
      asa: { grade: "III", emergency: false, reason: "COPD" }, guide: [copd], mustChoose: false,
    });
  });

  it("asks the author to choose when the grade depends on severity, and fills in nothing", () => {
    expect(afterCodeAdded(undefined, [], [well, poorly])).toEqual({ asa: undefined, guide: [well, poorly], mustChoose: true });
    // A second code with one example doesn't decide it: the guide now has three.
    expect(afterCodeAdded(undefined, [well, poorly], [copd])).toEqual({ asa: undefined, guide: [well, poorly, copd], mustChoose: true });
  });

  it("leaves a grade the disclosure already has, and a code with no example changes nothing", () => {
    const kept = { grade: "IV" as const, emergency: true };
    expect(afterCodeAdded(kept, [], [copd])).toEqual({ asa: kept, guide: [copd], mustChoose: false });
    expect(afterCodeAdded(undefined, [], [])).toEqual({ asa: undefined, guide: [], mustChoose: false });
    // The same example from two codes is listed once.
    expect(afterCodeAdded(undefined, [copd], [copd]).guide).toEqual([copd]);
  });

  it("after the author chose No ASA grade, adding a code fills in nothing and asks nothing (verification 1, F1)", () => {
    // COPD would fill in III; Cough brings no example: the declined grade must not come back.
    expect(afterCodeAdded(undefined, [copd], [], true)).toEqual({ asa: undefined, guide: [copd], mustChoose: false });
    // A code with several examples adds them to the box, but No ASA grade stands.
    expect(afterCodeAdded(undefined, [copd], [well, poorly], true)).toEqual({ asa: undefined, guide: [copd, well, poorly], mustChoose: false });
  });

  it("removing a code keeps the Emergency (E) tick on the grade that replaces it (verification 1, F2)", () => {
    const smoker: AsaChoice = { grade: "II", example: "Current smoker", group: "Adults" };
    expect(afterCodeRemoved({ grade: "III", emergency: true, reason: "COPD" }, [copd], [smoker]))
      .toEqual({ asa: { grade: "II", emergency: true, reason: "Current smoker" }, mustChoose: false });
  });

  it("removing the code a grade came from takes the grade and its words away, then works as adding the rest (test report 1, F1)", () => {
    const smoker: AsaChoice = { grade: "II", example: "Current smoker", group: "Adults" };
    const fromCopd = { grade: "III" as const, emergency: false, reason: "COPD" };
    // COPD removed, Smoker left: Smoker's only example fills in.
    expect(afterCodeRemoved(fromCopd, [copd], [smoker])).toEqual({ asa: { grade: "II", emergency: false, reason: "Current smoker" }, mustChoose: false });
    // Diabetes left: the author must choose again.
    expect(afterCodeRemoved(fromCopd, [copd], [well, poorly])).toEqual({ asa: undefined, mustChoose: true });
    // Nothing left with an example: no grade.
    expect(afterCodeRemoved(fromCopd, [copd], [])).toEqual({ asa: undefined, mustChoose: false });
    // Another code still offers the same example: kept.
    expect(afterCodeRemoved(fromCopd, [copd], [copd])).toEqual({ asa: fromCopd, mustChoose: false });
    // A grade that didn't come from the removed code (the author's own, or another code's) is kept.
    const own = { grade: "IV" as const, emergency: true };
    expect(afterCodeRemoved(own, [copd], [smoker])).toEqual({ asa: own, mustChoose: false });
    const fromSmoker = { grade: "II" as const, emergency: false, reason: "Current smoker" };
    expect(afterCodeRemoved(fromSmoker, [copd], [smoker])).toEqual({ asa: fromSmoker, mustChoose: false });
    expect(afterCodeRemoved(undefined, [copd], [well, poorly])).toEqual({ asa: undefined, mustChoose: false });
  });

  it("a grade picked takes the guide's words when one example has that grade, else none", () => {
    expect(gradePicked("III", false, [well, poorly])).toEqual({ grade: "III", emergency: false, reason: "Poorly controlled diabetes" });
    expect(gradePicked("III", true, [well, poorly, copd])).toEqual({ grade: "III", emergency: true }); // two at III: the author's own
    expect(gradePicked("V", false, [well, poorly])).toEqual({ grade: "V", emergency: false });
    expect(gradePicked(null, false, [well])).toBeUndefined();
  });
});

const out = (id: string, extra: Partial<ClinicalOutput> = {}): ClinicalOutput => ({ id, codes: [], ...extra });
const heartFailure = out("o_hf", {
  codes: [{ set: "ICD10", code: "I50.9", display: "Heart failure unspecified" }],
  asa: { grade: "III", emergency: false, reason: "Moderate reduction of ejection fraction" },
});
const angina = out("o_angina", { note: { text: "Angina", category: "Cardiac" }, asa: { grade: "III", emergency: false } });
const smoker = out("o_smoker", { codes: [{ set: "SNOMED", code: "77176002", display: "Smoker" }], asa: { grade: "II", emergency: false, reason: "Current smoker" } });
const urgent = out("o_urgent", { note: { text: "Needs surgery today", category: "Surgical" }, asa: { grade: "I", emergency: true } });

const yesNo = (name: string, outputs: ClinicalOutput[], extra: Record<string, unknown> = {}) => ({
  type: "radiogroup", name, title: name, ...extra,
  choices: [{ value: "y", text: "Yes", clinicalOutputs: outputs }, { value: "n", text: "No" }],
});
const set = (id: string, elements: unknown[], extra: Record<string, unknown> = {}) =>
  ({ id, name: id, content: { pages: [{ name: `${id}_p`, elements }], ...extra } as ChapterJson });
const patient = { name: "Jo Bloggs", age: 60, sex: "male" };

describe("the suggested ASA for the whole HQ (S39, ASA-01)", () => {
  const hq = [
    set("lifestyle", [yesNo("q_smoke", [smoker])]),
    set("heart", [yesNo("q_hf", [heartFailure]), yesNo("q_angina", [angina])]),
  ];

  it("is the highest grade across every shown set, with each disclosure at that grade as a reason", () => {
    const s = hqSuggestedAsa(hq, { lifestyle: { q_smoke: "y" }, heart: { q_hf: "y", q_angina: "y" } }, "clinician", patient);
    expect(s?.grade).toBe("III");
    expect(s?.emergency).toBe(false);
    expect(s?.reasons.map((r) => r.text)).toEqual(["Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction", "Angina"]);
    expect(s?.reasons[0]).toMatchObject({ setId: "heart", questionId: "q_hf" });
    expect(suggestedAsaText(s)).toBe("Suggested ASA III: Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction; Angina");
  });

  it("suggests none when no answer gives a grade", () => {
    const s = hqSuggestedAsa(hq, { lifestyle: { q_smoke: "n" }, heart: { q_hf: "n" } }, "clinician", patient);
    expect(s).toBeUndefined();
    expect(suggestedAsaText(s)).toBe("No suggested ASA");
  });

  it("a lower set alone gives its grade", () => {
    expect(suggestedAsaText(hqSuggestedAsa(hq, { lifestyle: { q_smoke: "y" } }, "clinician", patient))).toBe("Suggested ASA II: Smoker (77176002) – Current smoker");
  });

  it("keeps E (emergency) from any disclosure, and says which", () => {
    const withUrgent = [...hq, set("surgery", [yesNo("q_today", [urgent])])];
    const s = hqSuggestedAsa(withUrgent, { lifestyle: { q_smoke: "y" }, surgery: { q_today: "y" } }, "clinician", patient);
    expect(s).toMatchObject({ grade: "II", emergency: true });
    expect(suggestedAsaText(s)).toBe("Suggested ASA IIE: Smoker (77176002) – Current smoker; Needs surgery today (emergency)");
  });

  it("leaves out a hidden set's disclosures, and a hidden question's", () => {
    const hidden = [
      set("lifestyle", [yesNo("q_smoke", [smoker])]),
      set("heart", [yesNo("q_hf", [heartFailure]), yesNo("q_angina", [angina], { visibleIf: "{q_hf} = 'y'" })], { chapterVisibleIf: "{q_smoke} = 'y'" }),
    ];
    const answers = { lifestyle: { q_smoke: "n" }, heart: { q_hf: "y", q_angina: "y" } };
    expect(hqSuggestedAsa(hidden, answers, "clinician", patient)).toBeUndefined();
    const s = hqSuggestedAsa(hidden, { lifestyle: { q_smoke: "y" }, heart: { q_hf: "n", q_angina: "y" } }, "clinician", patient);
    expect(s?.grade).toBe("II"); // angina is held but hidden
  });

  it("counts a score band's disclosure", () => {
    const bmi = set("measures", [
      { type: "text", inputType: "number", name: "q_bmi", title: "BMI" },
      {
        type: "expression", name: "c_bmi", title: "BMI band", expression: "{q_bmi}",
        bands: [
          { id: "b_ok", label: "Under 40", max: 39.9 },
          { id: "b_high", label: "40 or more", clinicalOutputs: [out("o_bmi", { note: { text: "BMI {answer}", category: "Lifestyle" }, asa: { grade: "III", emergency: false, reason: "Morbid obesity (BMI 40 or more)" } })] },
        ],
      },
    ]);
    expect(suggestedAsaText(hqSuggestedAsa([bmi], { measures: { q_bmi: 41 } }, "clinician", patient))).toBe("Suggested ASA III: BMI 41 – Morbid obesity (BMI 40 or more)");
    expect(hqSuggestedAsa([bmi], { measures: { q_bmi: 30 } }, "clinician", patient)).toBeUndefined();
  });

  it("ties: every disclosure at the top grade, each once", () => {
    const twice = [set("a", [yesNo("q_1", [angina]), yesNo("q_2", [angina])])];
    expect(hqSuggestedAsa(twice, { a: { q_1: "y", q_2: "y" } }, "clinician", patient)?.reasons.map((r) => r.text)).toEqual(["Angina"]);
  });

  it("the patient's view leaves out clinician-only questions", () => {
    const withOnly = [set("a", [yesNo("q_smoke", [smoker]), yesNo("q_hf", [heartFailure], { clinicianOnly: true })])];
    const answers = { a: { q_smoke: "y", q_hf: "y" } };
    expect(hqSuggestedAsa(withOnly, answers, "clinician", patient)?.grade).toBe("III");
    expect(hqSuggestedAsa(withOnly, answers, "patient", patient)?.grade).toBe("II");
  });

  it("the one-set suggested ASA (the preview's) has the reasons too", () => {
    const m = new Model(hq[1].content);
    m.data = { q_hf: "y" };
    expect(computeOutputs(m).suggestedAsa).toMatchObject({ grade: "III", emergency: false, reasons: [{ text: "Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction" }] });
  });
});

describe("the suggested ASA in a clinician set's condition (S39 answers 2)", () => {
  const plan = set("plan", [{ type: "comment", name: "q_plan", title: "Anaesthetic plan" }], { chapterVisibleIf: "{suggestedAsa} >= 3" });
  const hq = [set("lifestyle", [yesNo("q_smoke", [smoker])]), set("heart", [yesNo("q_hf", [heartFailure])]), plan];

  it("shows the set only when the sets before it suggest ASA III or more", () => {
    const ids = (answers: Record<string, Record<string, unknown>>) => shownSets(hq, answers, "clinician", patient).map((s) => s.id);
    expect(ids({ lifestyle: { q_smoke: "y" }, heart: { q_hf: "y" } })).toEqual(["lifestyle", "heart", "plan"]);
    expect(ids({ lifestyle: { q_smoke: "y" }, heart: { q_hf: "n" } })).toEqual(["lifestyle", "heart"]);
    expect(ids({})).toEqual(["lifestyle", "heart"]); // no grade at all is 0
    // "At most II" includes no grade.
    const low = [hq[0], set("fast", [], { chapterVisibleIf: "{suggestedAsa} <= 2" })];
    expect(shownSets(low, {}, "clinician", patient).map((s) => s.id)).toEqual(["lifestyle", "fast"]);
  });

  it("answers given by hand (the preview, a test case) count as no grade, as an episode does (verification 1)", () => {
    const fast = { name: "Fast track", doc: { chapterVisibleIf: "{suggestedAsa} <= 2", pages: [{ name: "p", elements: [{ type: "text", name: "q_x" }] }] } as ChapterJson };
    expect(givenEarlier([fast], { q_x: "a" }, patient, "clinician").hidden).toEqual([]);
    expect(givenEarlier([{ ...fast, doc: { ...fast.doc, chapterVisibleIf: "{suggestedAsa} >= 3" } }], { q_x: "a" }, patient, "clinician").hidden).toEqual(["Fast track"]);
  });

  it("is offered, in words, only to a clinician set's condition", () => {
    const earlier = { pages: [] };
    expect(subjectsFor(earlier, { upToPage: "" }).some((s) => s.left === "suggestedAsa")).toBe(false);
    const asa = subjectsFor(earlier, { upToPage: "" }, { suggestedAsa: true }).find((s) => s.left === "suggestedAsa");
    expect(asa).toMatchObject({ label: "Suggested ASA", type: "asa" });
    expect(OPS.asa.map((o) => o.label)).toEqual(["is at least", "is at most", "is"]);
    expect(parseLogic("{suggestedAsa} >= 3")).toEqual({ join: "and", items: [{ left: "suggestedAsa", op: "gte", value: 3 }] });
    const words = describeLogic(earlier, "{suggestedAsa} >= 3");
    expect(words && "parts" in words ? words.parts.map((p) => p.text).join(" ") : words).toBe("Suggested ASA is at least III");
  });

  it("is a problem in a patient set's condition, and fine in a clinician set's", () => {
    const earlier = { pages: [] };
    expect(chapterConditionProblems(plan.content, earlier, "clinician")).toEqual([]);
    expect(chapterConditionProblems(plan.content, earlier, "patient")).toEqual(["only a clinician's Question Set can test the suggested ASA"]);
    expect(publishProblems([{ name: "Plan", doc: plan.content, audience: "patient" }])).toEqual([
      { set: "Plan", message: "Question Set condition: only a clinician's Question Set can test the suggested ASA" },
    ]);
    expect(publishProblems([{ name: "Plan", doc: plan.content, audience: "clinician" }])).toEqual([]);
  });
});
