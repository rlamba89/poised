import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import type { RuleSet } from "./rules";
import { formatScore } from "./scores";
import { autoTriageWarnings, publishWarnings, ungradedAsaWarnings, type AsaGuide } from "./warnings";

const yesNo = (name: string, title: string, extra: object = {}) => ({
  type: "radiogroup", name, title, ...extra, choices: [{ value: "o_y", text: "Yes", score: 1 }, { value: "o_n", text: "No", score: 0 }],
});
const score = (name: string, title: string, questions: string[], extra: object = {}) => ({
  type: "expression", name, title, calcKind: "score", expression: formatScore({ kind: "score", questions }), ...extra,
});
const about: ChapterJson = { pages: [{ name: "p_1", elements: [yesNo("q_snore", "Do you snore?"), score("q_snore_pts", "Snore points", ["q_snore"])] }] };
const nurse: ChapterJson = {
  pages: [{ name: "p_1", elements: [yesNo("q_neck", "Neck circumference over 40 cm?"), score("q_neck_risk", "Neck size risk", ["q_snore", "q_neck"])] }],
};
const triage: RuleSet = {
  noGradeAutoTriage: true, testCases: [],
  rules: [{ id: "low", name: "Low risk", when: { asa: { op: "<=", grade: 2 } }, then: [{ kind: "autoTriage", target: "ready_for_admission" }] }],
};
const message = "Auto-triage won't move patients on for this HQ: the score “Neck size risk” needs answers a clinician enters (“Neck circumference over 40 cm?”), so it's always incomplete when the patient sends. These patients go to Ready for review.";

describe("autoTriageWarnings (S44a)", () => {
  it("names a score that needs a clinician Question Set's answer, and its inputs", () => {
    expect(autoTriageWarnings([{ name: "About you", doc: about, audience: "patient" }, { name: "Nurse", doc: nurse, audience: "clinician" }], triage)).toEqual([message]);
  });

  it("names one that needs a clinician-only question in a patient set", () => {
    const doc: ChapterJson = { pages: [{ name: "p_1", elements: [
      yesNo("q_snore", "Do you snore?"), yesNo("q_neck", "Neck circumference over 40 cm?", { clinicianOnly: true }), score("q_neck_risk", "Neck size risk", ["q_snore", "q_neck"]),
    ] }] };
    expect(autoTriageWarnings([{ name: "About you", doc, audience: "patient" }], triage)).toEqual([message]);
  });

  it("follows a score that uses such a score's band", () => {
    const later: ChapterJson = { pages: [{ name: "p_2", elements: [{
      type: "expression", name: "q_total", title: "Total", calcKind: "score",
      expression: formatScore({ kind: "score", questions: [], lines: [{ points: 2, rule: { left: "q_neck_risk", fn: "band", op: "eq", value: "b_hi" } }] }),
    }] }] };
    const w = autoTriageWarnings([{ name: "About you", doc: about, audience: "patient" }, { name: "Nurse", doc: nurse, audience: "clinician" }, { name: "More", doc: later, audience: "patient" }], triage);
    expect(w).toHaveLength(2);
    expect(w[1]).toContain("the score “Total” needs answers a clinician enters (“Neck circumference over 40 cm?”)");
  });

  it("says nothing when the patient answers every input, or no rule auto-triages", () => {
    expect(autoTriageWarnings([{ name: "About you", doc: about, audience: "patient" }], triage)).toEqual([]);
    const noTriage: RuleSet = { ...triage, rules: [{ ...triage.rules[0], then: [{ kind: "flag", name: "Check" }] }] };
    const sets = [{ name: "About you", doc: about, audience: "patient" }, { name: "Nurse", doc: nurse, audience: "clinician" }];
    expect(autoTriageWarnings(sets, noTriage)).toEqual([]);
    expect(autoTriageWarnings(sets, null)).toEqual([]);
    expect(publishWarnings(sets, triage)).toEqual([message]);
  });
});

describe("ungradedAsaWarnings (S44a)", () => {
  const diabetes = { set: "ICD10" as const, code: "E11.9", display: "Type 2 diabetes" };
  const guide: AsaGuide = (c) => (c.code === "E11.9" ? [
    { grade: "II", example: "Well-controlled diabetes", group: "Adults" },
    { grade: "III", example: "Poorly controlled diabetes", group: "Adults" },
  ] : []);
  const withOutput = (output: object): ChapterJson => ({ pages: [{ name: "p_1", elements: [{
    type: "radiogroup", name: "q_dm", title: "Do you have diabetes?", choices: [{ value: "o_y", text: "Yes", clinicalOutputs: [{ id: "d_1", ...output } as never] }, { value: "o_n", text: "No" }],
  }] }] });

  it("warns about a code the guide grades when no grade was chosen", () => {
    expect(ungradedAsaWarnings([{ name: "Health", doc: withOutput({ codes: [diabetes] }) }], guide)).toEqual([
      "Health: “Do you have diabetes?” answered “Yes” has “Type 2 diabetes (E11.9)”, which the ASA guide grades ASA II or III, but no ASA grade was chosen. Choose one in its disclosure, or No ASA grade.",
    ]);
  });

  it("says nothing once a grade or a deliberate No ASA grade is chosen, or for a code the guide doesn't grade", () => {
    expect(ungradedAsaWarnings([{ name: "H", doc: withOutput({ codes: [diabetes], asa: { grade: "II", emergency: false } }) }], guide)).toEqual([]);
    expect(ungradedAsaWarnings([{ name: "H", doc: withOutput({ codes: [diabetes], asaNone: true }) }], guide)).toEqual([]);
    expect(ungradedAsaWarnings([{ name: "H", doc: withOutput({ codes: [{ set: "SNOMED", code: "1", display: "Other" }] }) }], guide)).toEqual([]);
  });

  it("is part of the Publish window's warnings once the guide is known", () => {
    const sets = [{ name: "H", doc: withOutput({ codes: [diabetes] }) }];
    expect(publishWarnings(sets, null)).toEqual([]);
    expect(publishWarnings(sets, null, guide)).toHaveLength(1);
  });
});
