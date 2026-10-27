import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import defaults from "../../../apps/api/internal/asarules/defaults.json";
import type { ChapterJson } from "./doc";
import { computeOutputs } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { publishProblems } from "./publish";
import {
  actionWords, autoTriageSummary, codesIn, setFiredRules, firedLine, firedWords, firedRules, hqRuleResult, ruleNames, ruleProblems, ruleWords, ruleSetOf, runHqTestCase, scoresWithBands,
  type AsaRule, type HqTestCase, type RuleSet,
} from "./rules";

registerClinicalProperties();

const DEFAULTS = defaults as RuleSet;
// S42 (test report 1, E-2): the guards every send applies, in the Publish window's line.
const NEVER = "Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first.";
const copy = (): RuleSet => structuredClone(DEFAULTS);

const smoker = { set: "SNOMED" as const, code: "77176002", display: "Smoker" };
const heart = { set: "ICD10" as const, code: "I50.9", display: "Heart failure" };

// A patient set: smoking gives ASA II, heart failure ASA III, chest pain a red flag; a score with bands.
const aboutYou: ChapterJson = {
  pages: [{
    name: "p_1",
    elements: [
      { type: "radiogroup", name: "q_smoke", title: "Do you smoke?", choices: [
        { value: "o_y", text: "Yes", clinicalOutputs: [{ id: "d1", codes: [smoker], asa: { grade: "II", emergency: false } }] },
        { value: "o_n", text: "No" },
      ] },
      { type: "radiogroup", name: "q_heart", title: "Heart failure?", choices: [
        { value: "o_y", text: "Yes", clinicalOutputs: [{ id: "d2", codes: [heart], asa: { grade: "III", emergency: false } }] },
        { value: "o_n", text: "No" },
      ] },
      { type: "radiogroup", name: "q_chest", title: "Chest pain?", choices: [
        { value: "o_y", text: "Yes", clinicalOutputs: [{ id: "d3", codes: [], note: { text: "Chest pain", category: "Heart" }, flag: "red" }] },
        { value: "o_n", text: "No" },
      ] },
      { type: "text", inputType: "number", name: "q_snores", title: "Nights snoring" },
      { type: "expression", name: "q_score", title: "Snore score", expression: "{q_snores}", bands: [
        { id: "b_low", label: "Low", max: 2 }, { id: "b_high", label: "High" },
      ] },
    ],
  }],
};
// A clinician set: its ASA IV never counts for the rules (the patient doesn't fill it in).
const nurse: ChapterJson = {
  pages: [{ name: "p_1", elements: [{ type: "radiogroup", name: "q_frail", choices: [
    { value: "o_y", text: "Yes", clinicalOutputs: [{ id: "d4", codes: [], note: { text: "Frail", category: "General" }, asa: { grade: "IV", emergency: false } }] },
  ] }] }],
};
const sets = [
  { id: "s1", name: "About you", content: aboutYou, audience: "patient" },
  { id: "s2", name: "Nurse checks", content: nurse, audience: "clinician" },
];
const patient = { name: "Sam Sample", age: 54, sex: "female" };

/** What one set's answers produce, as the Preview gives the rules. */
function resultOf(answers: Record<string, unknown>, procedureId?: string) {
  const m = new Model(aboutYou);
  m.data = answers;
  const r = computeOutputs(m);
  return { suggestedAsa: r.suggestedAsa, outputs: r.outputs, scores: r.scores ?? [], procedureId };
}
const names = (set: RuleSet, input: Parameters<typeof firedRules>[1]) => firedRules(set, input).map((f) => f.rule.name);

describe("the rule matcher (S40, ASA-02/03)", () => {
  it("the defaults: ASA II with no red flags is Low risk; ASA III is High risk", () => {
    const low = firedRules(DEFAULTS, resultOf({ q_smoke: "o_y" }));
    expect(low.map((f) => f.rule.name)).toEqual(["Low risk"]);
    expect(low[0].actions).toEqual([{ kind: "autoTriage", target: "ready_for_admission" }]);
    const high = firedRules(DEFAULTS, resultOf({ q_smoke: "o_y", q_heart: "o_y" }));
    expect(high.map((f) => f.rule.name)).toEqual(["High risk"]);
    expect(high[0].actions.map((a) => a.kind)).toEqual(["task", "flag", "requireAnaesthetistAsa"]);
  });

  it("a red flag stops Low risk", () => {
    expect(names(DEFAULTS, resultOf({ q_smoke: "o_y", q_chest: "o_y" }))).toEqual([]);
    // An amber flag doesn't: the rule says red.
    const amber = resultOf({ q_smoke: "o_y" });
    amber.outputs.push({ ...amber.outputs[0], output: { id: "x", codes: [], flag: "amber" } });
    expect(names(DEFAULTS, amber)).toEqual(["Low risk"]);
  });

  // S42 (test report 1, E-2): the guards a real send applies can't be switched off, so the Preview, Test
  // the whole HQ and test cases hold the auto-triage as the server does, and say why.
  it("holds the auto-triage, saying why, as a real send does: an amber flag, an Incomplete score, the anaesthetist's ASA", () => {
    const amber = resultOf({ q_smoke: "o_y" });
    amber.outputs.push({ ...amber.outputs[0], output: { id: "x", codes: [], flag: "amber" } });
    const [a] = firedRules(DEFAULTS, amber);
    expect(a).toMatchObject({ held: true, heldReason: "flag", actions: [] });
    expect(firedLine(a)).toBe("Rule fires: Low risk (no auto-triage: a red or amber flag)");
    expect(firedWords(a)).toBe("no auto-triage: a red or amber flag");

    const incomplete = resultOf({ q_smoke: "o_y" });
    incomplete.scores.push({ questionId: "q_stopbang", title: "STOP-Bang", incomplete: true });
    const [i] = firedRules(DEFAULTS, incomplete);
    expect(i).toMatchObject({ held: true, heldReason: "incomplete", actions: [] });
    expect(firedLine(i)).toBe("Rule fires: Low risk (no auto-triage: a score is Incomplete)");

    const asa = copy();
    asa.rules.push({ id: "d", name: "Diabetes", when: { asa: { op: "<=", grade: 6 } }, then: [{ kind: "requireAnaesthetistAsa" }] });
    const [low, diabetes] = firedRules(asa, resultOf({ q_smoke: "o_y" }));
    expect(low).toMatchObject({ held: true, heldReason: "anaesthetistAsa", actions: [] });
    expect(firedLine(low)).toBe("Rule fires: Low risk (no auto-triage: the anaesthetist's ASA is required first)");
    expect(diabetes).toMatchObject({ held: false, actions: [{ kind: "requireAnaesthetistAsa" }] });
    // Only Ready for admission waits for the anaesthetist's ASA.
    asa.rules[0].then = [{ kind: "autoTriage", target: "ready_for_poa" }, { kind: "flag", name: "Day case" }];
    const [poa] = firedRules(asa, resultOf({ q_smoke: "o_y" }));
    expect(poa.held).toBe(false);
    // A held rule keeps its other actions.
    amber.outputs.length = 1;
    amber.outputs.push({ ...amber.outputs[0], output: { id: "x", codes: [], flag: "amber" } });
    expect(firedRules(asa, amber)[0]).toMatchObject({ held: true, heldReason: "flag", actions: [{ kind: "flag", name: "Day case" }] });
    expect(firedWords(firedRules(asa, amber)[0])).toBe("flag “Day case”; no auto-triage: a red or amber flag");
    // A rule with no auto-triage is never held.
    expect(firedRules(asa, amber)[1].held).toBe(false);
  });

  it("no ASA grade counts as ASA I, so a fit patient is auto-triaged by default (S40 answers 2)", () => {
    const fit = resultOf({ q_smoke: "o_n" });
    expect(fit.suggestedAsa).toBeUndefined();
    const [f] = firedRules(DEFAULTS, fit);
    expect(f.rule.name).toBe("Low risk");
    expect(f.held).toBe(false);
    expect(f.actions).toEqual([{ kind: "autoTriage", target: "ready_for_admission" }]);
  });

  it("with the HQ's setting off, a patient with no ASA grade is never auto-triaged", () => {
    const off = { ...copy(), noGradeAutoTriage: false };
    const [f] = firedRules(off, resultOf({ q_smoke: "o_n" }));
    expect(f.rule.name).toBe("Low risk");
    expect(f.held).toBe(true);
    expect(f.heldReason).toBe("noGrade");
    expect(f.actions).toEqual([]);
    expect(firedLine(f)).toBe("Rule fires: Low risk (no auto-triage: no ASA grade)");
    // A patient with a grade is auto-triaged as before.
    expect(firedRules(off, resultOf({ q_smoke: "o_y" }))[0].actions).toEqual([{ kind: "autoTriage", target: "ready_for_admission" }]);
  });

  it("compares the ASA at least, at most and exactly", () => {
    const rule = (op: ">=" | "<=" | "=", grade: number): RuleSet => ({
      noGradeAutoTriage: true, testCases: [], rules: [{ id: "r", name: "R", when: { asa: { op, grade } }, then: [{ kind: "requireAnaesthetistAsa" }] }],
    });
    const two = resultOf({ q_smoke: "o_y" });
    expect(names(rule("=", 2), two)).toEqual(["R"]);
    expect(names(rule("=", 3), two)).toEqual([]);
    expect(names(rule(">=", 2), two)).toEqual(["R"]);
    expect(names(rule("<=", 1), two)).toEqual([]);
  });

  it("the extras: a score band, a disclosure code and the episode's procedure", () => {
    const withWhen = (extra: Partial<AsaRule["when"]>): RuleSet => ({
      noGradeAutoTriage: true, testCases: [],
      rules: [{ id: "r", name: "R", when: { asa: { op: "<=", grade: 6 }, ...extra }, then: [{ kind: "flag", name: "Look" }] }],
    });
    const band = withWhen({ band: { score: "q_score", bands: ["b_high"] } });
    expect(names(band, resultOf({ q_snores: 5 }))).toEqual(["R"]);
    expect(names(band, resultOf({ q_snores: 1 }))).toEqual([]);
    expect(names(band, resultOf({}))).toEqual([]); // no value, no band
    const code = withWhen({ code: heart });
    expect(names(code, resultOf({ q_heart: "o_y" }))).toEqual(["R"]);
    expect(names(code, resultOf({ q_smoke: "o_y" }))).toEqual([]);
    const proc = withWhen({ procedures: ["p_hip", "p_knee"] });
    expect(names(proc, resultOf({}, "p_knee"))).toEqual(["R"]);
    expect(names(proc, resultOf({}, "p_eye"))).toEqual([]);
    expect(names(proc, resultOf({}))).toEqual([]); // no procedure (the Preview)
  });

  it("the Preview's rules use the clinician's view, whichever view is shown (test report 1, F-1)", () => {
    const hidden = structuredClone(aboutYou);
    hidden.pages![0].elements!.find((e) => e.name === "q_score")!.clinicianOnly = true;
    const band: RuleSet = { noGradeAutoTriage: true, testCases: [],
      rules: [{ id: "r", name: "Snorer", when: { asa: { op: "<=", grade: 6 }, band: { score: "q_score", bands: ["b_high"] } }, then: [{ kind: "flag", name: "Sleep" }] }] };
    // The answers as the Patient view gives them: no value for the clinician-only score.
    expect(setFiredRules(band, hidden, { q_snores: 5 }, patient).map((f) => f.rule.name)).toEqual(["Snorer"]);
    expect(setFiredRules(band, hidden, { q_snores: 1 }, patient)).toEqual([]);
  });

  it("every rule that matches fires, in the rules' order", () => {
    const set = copy();
    set.rules.unshift({ id: "any", name: "Everyone", when: { asa: { op: ">=", grade: 1 } }, then: [{ kind: "suggestStatus", status: "ready_for_poa" }] });
    expect(names(set, resultOf({ q_heart: "o_y" }))).toEqual(["Everyone", "High risk"]);
  });
});

describe("rules in plain words", () => {
  const n = ruleNames([aboutYou], [{ id: "p_hip", name: "Hip replacement" }, { id: "p_knee", name: "Knee replacement" }]);

  it("the defaults", () => {
    expect(ruleWords(DEFAULTS.rules[0], n)).toBe("When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission");
    expect(ruleWords(DEFAULTS.rules[1], n)).toBe(
      "When the suggested ASA is III or more: add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission",
    );
  });

  it("each kind of when and action", () => {
    const rule: AsaRule = {
      id: "r", name: "R",
      when: { asa: { op: "=", grade: 3 }, band: { score: "q_score", bands: ["b_low", "b_high"] }, code: heart, procedures: ["p_hip", "p_knee"] },
      then: [{ kind: "autoTriage", target: "ready_for_poa" }, { kind: "suggestStatus", status: "on_hold" }],
    };
    expect(ruleWords(rule, n)).toBe(
      "When the suggested ASA is III and Snore score is Low or High and a disclosure has the code Heart failure (I50.9) and the procedure is Hip replacement or Knee replacement: auto-triage to Ready for POA; suggest On hold",
    );
    expect(actionWords({ kind: "task", title: "Call", team: "nurse" })).toBe("add the task “Call” for the nurse team");
    // A score or procedure no longer there is named by its key.
    expect(ruleWords({ ...rule, when: { asa: { op: "<=", grade: 1 }, band: { score: "q_gone", bands: ["b_x"] }, procedures: ["p_x"] } }, n))
      .toContain("the suggested ASA is I and q_gone is b_x and the procedure is p_x");
    // Every "at most" and "at least" reads as a range (test report 1, E-1).
    const asa = (op: ">=" | "<=", grade: number) => ruleWords({ ...rule, when: { asa: { op, grade } } }, n).split(":")[0];
    expect(asa("<=", 6)).toBe("When the suggested ASA is I to VI");
    expect(asa("<=", 3)).toBe("When the suggested ASA is I to III");
    expect(asa(">=", 1)).toBe("When the suggested ASA is I to VI");
    expect(asa(">=", 4)).toBe("When the suggested ASA is IV or more");
  });

  it("the Publish window's line says what the version does automatically", () => {
    expect(autoTriageSummary(DEFAULTS, n)).toBe("This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I. " + NEVER);
    const off = { ...copy(), noGradeAutoTriage: false };
    off.rules[0].then = [{ kind: "autoTriage", target: "ready_for_poa" }];
    expect(autoTriageSummary(off, n)).toBe("This version auto-triages: ASA I–II, no red flags → Ready for POA. Patients with no ASA grade are never auto-triaged. " + NEVER);
    expect(autoTriageSummary({ ...copy(), rules: [DEFAULTS.rules[1]] }, n)).toBe("This version doesn't auto-triage anyone.");
    // S42 answers 2: the HQ can hold the patient's message until a clinician clicks Checked.
    expect(autoTriageSummary({ ...copy(), messageAfterCheck: true }, n)).toBe(
      "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I. " + NEVER +
        " The patient's message goes when a clinician clicks Checked.",
    );
  });

  it("reads the message setting as stored: off unless true (a set from before S42 has none)", () => {
    expect(DEFAULTS.messageAfterCheck).toBe(false);
    expect(ruleSetOf({ ...DEFAULTS })?.messageAfterCheck).toBe(false);
    expect(ruleSetOf({ noGradeAutoTriage: true, rules: [], testCases: [] })?.messageAfterCheck).toBe(false);
    expect(ruleSetOf({ ...DEFAULTS, messageAfterCheck: true })?.messageAfterCheck).toBe(true);
    expect(ruleSetOf({ ...DEFAULTS, messageAfterCheck: "yes" })?.messageAfterCheck).toBe(false);
  });

  it("lists the codes and the banded scores the patient's sets could give", () => {
    expect(codesIn([aboutYou]).map((c) => c.code)).toEqual(["77176002", "I50.9"]);
    expect(scoresWithBands([aboutYou]).map((s) => [s.key, s.title, s.bands.map((b) => b.label)])).toEqual([["q_score", "Snore score", ["Low", "High"]]]);
  });
});

describe("whole-HQ test cases (S40 answers 4)", () => {
  const tc = (answers: Record<string, unknown>, expected: string[]): HqTestCase => ({ id: "t1", name: "Smoker", patient, answers, expected });

  it("work out the suggested ASA and the rules from the patient's answers only", () => {
    const r = hqRuleResult(sets, DEFAULTS, { q_smoke: "o_y", q_frail: "o_y" }, patient);
    expect(r.asa?.grade).toBe("II"); // an answer to the clinician set doesn't count: the patient never gives it
    expect(r.shown).toEqual(["s1", "s2"]);
    // The ASA line first, then the rules in their order (test report 2, E-7).
    expect(r.lines).toEqual(["Suggested ASA II", "Rule fires: Low risk (auto-triage to Ready for admission)"]);
    expect(hqRuleResult(sets, DEFAULTS, { q_smoke: "o_n" }, patient).lines).toEqual(["No suggested ASA", "Rule fires: Low risk (auto-triage to Ready for admission)"]);
  });

  it("a clinician-only score worked out from the patient's answers counts (e.g. STOP-Bang)", () => {
    const hidden = structuredClone(aboutYou);
    const score = hidden.pages![0].elements!.find((e) => e.name === "q_score")!;
    score.clinicianOnly = true;
    const band: RuleSet = { noGradeAutoTriage: true, testCases: [],
      rules: [{ id: "r", name: "Snorer", when: { asa: { op: "<=", grade: 6 }, band: { score: "q_score", bands: ["b_high"] } }, then: [{ kind: "flag", name: "Sleep" }] }] };
    const r = hqRuleResult([{ id: "s1", name: "About you", content: hidden, audience: "patient" }], band, { q_snores: 5 }, patient);
    expect(r.fired.map((f) => f.rule.name)).toEqual(["Snorer"]);
  });

  it("pass, then fail when a rule changes (its ASA limit, or its target)", () => {
    const expected = hqRuleResult(sets, DEFAULTS, { q_smoke: "o_y" }, patient).lines;
    const t = tc({ q_smoke: "o_y" }, expected);
    expect(runHqTestCase(sets, DEFAULTS, t).pass).toBe(true);
    const limit = copy();
    limit.rules[0].when.asa.grade = 1;
    const r = runHqTestCase(sets, limit, t);
    expect(r.pass).toBe(false);
    expect(r.missing).toEqual(["Rule fires: Low risk (auto-triage to Ready for admission)"]);
    const target = copy();
    target.rules[0].then = [{ kind: "autoTriage", target: "ready_for_poa" }];
    expect(runHqTestCase(sets, target, t).pass).toBe(false);
  });

  it("a failing whole-HQ test case blocks Publish, as do rules that can't fire", () => {
    const set = copy();
    set.testCases = [tc({ q_smoke: "o_y" }, ["Suggested ASA II"])];
    const forPublish = sets.map((s) => ({ name: s.name, doc: s.content, audience: s.audience }));
    expect(publishProblems(forPublish, set)).toEqual([{ set: "Rules", message: "Test case “Smoker” fails" }]);
    set.testCases = [];
    set.rules.push({ id: "gone", name: "Gone", when: { asa: { op: "<=", grade: 6 }, band: { score: "q_gone", bands: ["b"] }, code: { set: "ICD10", code: "X1", display: "X" } }, then: [{ kind: "requireAnaesthetistAsa" }] });
    set.rules.push({ id: "band", name: "Band", when: { asa: { op: "<=", grade: 6 }, band: { score: "q_score", bands: ["b_gone"] } }, then: [{ kind: "requireAnaesthetistAsa" }] });
    expect(ruleProblems(set, sets)).toEqual([
      "Rule “Gone”: its score isn't in this HQ any more",
      "Rule “Gone”: no disclosure in this HQ has the code X1, so the rule can't fire",
      "Rule “Band”: a band it tests isn't in “Snore score” any more",
    ]);
    expect(publishProblems(forPublish).length).toBe(0); // no rules given: as before S40
  });
});
