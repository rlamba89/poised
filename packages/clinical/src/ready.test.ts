// STOP-Bang as published (S38): the scoring criteria of the updated questionnaire (Chung et al.,
// Chest 2016; stopbang.ca), each as a worked example. These are the ready-made score's test cases:
// they run with `make test`, and the score can't be changed in an HQ (readyScoreProblems).
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import type { ChapterJson } from "./doc";
import { publishProblems } from "./publish";
import { registerClinicalProperties } from "./properties";
import { showBands } from "./questionTypes";
import { APFEL_KEYS, apfelClinicianPart, apfelGroup, cfsElements, DASI_KEYS, MORSE_KEYS, morseClinicianPart, morseGroup, MUST_KEYS, mustClinicianPart, mustGroup, VTE_KEYS, vteClinicianPart, vteGroup, dasiGroup, PRISMA7_KEYS, prisma7Group, READY_SCORES, insertReadyScore, readyScoreAround, readyScoreProblems, STOP_BANG_KEYS as K, stopBangGroup } from "./ready";
import { earlierContexts } from "./respondent";
import { renameReferences } from "./library";
import { modelFor, type SamplePatient } from "./testcases";

registerClinicalProperties();

const measures: ChapterJson = { pages: [{ name: "p_1", title: "Measurements", elements: [{ type: "bmi", name: "q_bmi", title: "BMI" }] }] };
const sleep = (): ChapterJson => insertReadyScore({ pages: [{ name: "p_2", title: "Sleep", elements: [] }] }, { page: "p_2" }, "stopbang", "q_bmi.bmi").doc;

type Example = { name: string; patient: SamplePatient; yes: (keyof typeof K)[]; bmi?: number; total: number | "Incomplete"; band?: string };
const E = (name: string, age: number | undefined, sex: string, yes: Example["yes"], bmi: number | undefined, total: Example["total"], band?: string): Example => ({
  name, patient: { name: "Sam Sample", age, sex }, yes, bmi, total, band,
});

/** Height 170 cm; the weight that gives the BMI wanted. */
const bmiAnswer = (bmi: number) => ({ height: 170, weight: Math.round(bmi * 1.7 * 1.7 * 10) / 10 });

const EXAMPLES: Example[] = [
  E("Yes to 0–2 is low risk", 45, "female", ["snoring"], 28, 1, "Low risk"),
  E("Nothing at all is low risk", 30, "female", [], 22, 0, "Low risk"),
  E("Yes to 3–4 is intermediate (only 1 STOP question)", 51, "male", ["snoring"], 30, 3, "Intermediate risk"),
  E("Yes to 4: STOP 1, male, BMI over 35, large neck", 40, "male", ["snoring", "neck"], 36, 4, "Intermediate risk"),
  E("Yes to 5–8 is high risk", 60, "female", ["snoring", "tired", "observed", "pressure"], 30, 5, "High risk"),
  E("All eight", 70, "male", ["snoring", "tired", "observed", "pressure", "neck"], 40, 8, "High risk"),
  E("2 or more STOP and male is high risk", 40, "male", ["snoring", "tired"], 25, 3, "High risk"),
  E("2 or more STOP and BMI over 35 is high risk", 40, "female", ["snoring", "tired"], 36, 3, "High risk"),
  E("2 or more STOP and a large neck is high risk", 40, "female", ["snoring", "observed", "neck"], 25, 3, "High risk"),
  E("2 STOP with age over 50 only stays intermediate", 55, "female", ["tired", "pressure"], 25, 3, "Intermediate risk"),
  E("Age 50 isn't over 50", 50, "female", ["snoring"], 25, 1, "Low risk"),
  E("BMI 35 isn't over 35", 40, "female", ["snoring", "tired"], 35, 2, "Low risk"),
  E("No BMI is Incomplete", 40, "male", ["snoring"], undefined, "Incomplete"),
  E("No age is Incomplete", undefined, "male", ["snoring"], 25, "Incomplete"),
  E("Sex not known is Incomplete", 40, "unknown", ["snoring"], 25, "Incomplete"),
  // S44c (Rahul, 28 Oct): "Other" can't decide the male point either, so it fails safe.
  E("Sex Other is Incomplete", 40, "other", ["snoring"], 25, "Incomplete"),
];

function run(ex: Example, unanswered?: keyof typeof K) {
  const hq = [{ id: "s1", name: "Measurements", content: measures }, { id: "s2", name: "Sleep", content: sleep() }];
  const s2: Record<string, unknown> = {};
  for (const q of ["snoring", "tired", "observed", "pressure", "neck"] as const) if (q !== unanswered) s2[K[q]] = ex.yes.includes(q) ? "o_yes" : "o_no";
  const answers = { s1: ex.bmi === undefined ? {} : { q_bmi: bmiAnswer(ex.bmi) }, s2 };
  const m = modelFor(hq[1].content, "clinician", ex.patient, earlierContexts(hq, answers, "clinician", ex.patient).get("s2"));
  showBands(m);
  m.data = s2;
  const q = m.getQuestionByName(K.score) as unknown as { value: unknown; formatedValue: string };
  const out = { value: q.value, shown: q.formatedValue };
  m.dispose();
  return out;
}

describe("STOP-Bang, as published", () => {
  for (const ex of EXAMPLES) {
    it(ex.name, () => {
      const r = run(ex);
      expect(r.shown).toBe(ex.total === "Incomplete" ? "Incomplete" : `${ex.total} (${ex.band})`);
    });
  }
  it("is Incomplete while any question is unanswered", () => {
    expect(run(EXAMPLES[2], "neck").shown).toBe("Incomplete");
  });
});

describe("the ready-made STOP-Bang in an HQ", () => {
  it("publishes as inserted, with a condition on the Group and disclosures on its options and bands", () => {
    const doc = sleep();
    expect(readyScoreProblems(doc)).toEqual([]);
    expect(publishProblems([{ name: "Measurements", doc: measures }, { name: "Sleep", doc }])).toEqual([]);
    const group = doc.pages![0].elements![0];
    group.visibleIf = "{patientAge} >= 18";
    (group.elements![5].bands as { clinicalOutputs?: unknown[] }[])[2].clinicalOutputs = [{ id: "out_1", codes: [], note: { text: "STOP-Bang high risk", category: "Airway" } }];
    (group.elements![3].choices as { clinicalOutputs?: unknown[] }[])[0].clinicalOutputs = [{ id: "out_2", codes: [{ set: "SNOMED", code: "38341003", display: "Hypertension" }] }];
    expect(readyScoreProblems(doc)).toEqual([]);
  });

  it("can't be changed: wording, points, bands or the BMI it uses", () => {
    const changes: ((g: ReturnType<typeof stopBangGroup>) => void)[] = [
      (g) => (g.elements![0].title = "Do you snore?"),
      (g) => ((g.elements![1].choices as { score?: number }[])[0].score = 2),
      (g) => ((g.elements![5].bands as { max?: number }[])[0].max = 3),
      (g) => (g.elements![5].expression = String(g.elements![5].expression).replace("> 50", ">= 50")),
      (g) => g.elements!.splice(3, 1),
      (g) => delete g.readyScoreBmi,
    ];
    for (const change of changes) {
      const doc = sleep();
      change(doc.pages![0].elements![0]);
      expect(readyScoreProblems(doc)).toEqual(["STOP-Bang has been changed: a ready-made score can't be changed. Delete it and insert it again."]);
    }
  });

  it("follows a rename of its BMI question (verification 2)", () => {
    const renamed = renameReferences(sleep(), new Map([["q_bmi", "bmi_measured"]]));
    expect(renamed.pages![0].elements![0].readyScoreBmi).toBe("bmi_measured.bmi");
    expect(readyScoreProblems(renamed)).toEqual([]);
  });

  it("knows which elements belong to it", () => {
    const doc = sleep();
    expect(readyScoreAround(doc, K.snoring)?.score.name).toBe("STOP-Bang");
    expect(readyScoreAround(doc, K.group)?.score.name).toBe("STOP-Bang");
    expect(readyScoreAround(measures, "q_bmi")).toBeUndefined();
  });
});

// ---------------------------------------------------------------- DASI (S48)
// Hlatky MA et al., Am J Cardiol 1989;64:651–4: twelve Yes/No items, each Yes its published weight
// (0–58.2); VO2peak = 0.43 × DASI + 9.6 mL/kg/min, and METs = VO2peak / 3.5. Band (S48 answers 2):
// "Poor functional capacity" at 34 or less (Wijeysundera DN et al., BJA 2020;124(3):261–70).
const DASI_WEIGHTS = [2.75, 1.75, 2.75, 5.5, 8, 2.7, 3.5, 8, 4.5, 5.25, 6, 7.5];

/** A survey of one ready-made Group, answered; its Calculations as shown (score, METs…). */
function shown(group: ReturnType<typeof dasiGroup>, data: Record<string, unknown>, patient: SamplePatient = { name: "Sam Sample", age: 60, sex: "female" }) {
  const m = modelFor({ pages: [{ name: "p_1", title: "Fitness", elements: [group] }] }, "clinician", patient);
  showBands(m);
  m.data = data;
  const out: Record<string, string> = {};
  for (const q of m.getAllQuestions()) if (q.getType() === "expression") out[q.name] = (q as unknown as { formatedValue: string }).formatedValue;
  m.dispose();
  return out;
}

const dasiAnswers = (yes: number[]) => Object.fromEntries(DASI_KEYS.items.map((k, i) => [k, yes.includes(i + 1) ? "o_yes" : "o_no"]));

describe("DASI, as published", () => {
  it("has the twelve items with their published weights, 58.2 in all", () => {
    const g = dasiGroup();
    const items = g.elements!.filter((e) => e.type === "radiogroup");
    expect(items.map((e) => e.name)).toEqual(DASI_KEYS.items);
    expect(items.map((e) => (e.choices as { value: string; score: number }[]).find((c) => c.value === "o_yes")!.score)).toEqual(DASI_WEIGHTS);
    expect(items.every((e) => (e.choices as { value: string; score: number }[]).find((c) => c.value === "o_no")!.score === 0)).toBe(true);
  });
  const EXAMPLES: [string, number[], string, string][] = [
    ["Every item Yes: 58.2, 9.9 METs", [1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12], "58.2", "9.9 METs"],
    ["Every item No: 0, 2.7 METs (Poor)", [], "0 (Poor functional capacity)", "2.7 METs"],
    ["34 is Poor (the cut-off: 34 or less)", [1, 2, 4, 5, 7, 8, 9], "34 (Poor functional capacity)", "6.9 METs"],
    ["33.95 is Poor", [1, 2, 3, 5, 6, 7, 8, 9], "33.95 (Poor functional capacity)", "6.9 METs"],
    ["34.2 is over the cut-off: no band", [1, 3, 4, 5, 6, 8, 9], "34.2", "6.9 METs"],
    ["Self-care and walking indoors only: 4.5", [1, 2], "4.5 (Poor functional capacity)", "3.3 METs"],
  ];
  for (const [name, yes, score, mets] of EXAMPLES) {
    it(name, () => {
      const r = shown(dasiGroup(), dasiAnswers(yes));
      expect(r[DASI_KEYS.score]).toBe(score);
      expect(r[DASI_KEYS.mets]).toBe(mets);
    });
  }
  it("is Incomplete while any item is unanswered, with no METs", () => {
    const data = dasiAnswers([1, 2, 3]);
    delete data[DASI_KEYS.items[11]];
    const r = shown(dasiGroup(), data);
    expect(r[DASI_KEYS.score]).toBe("Incomplete");
    expect(r[DASI_KEYS.mets]).toBe("");
  });
});

// ---------------------------------------------------------------- PRISMA-7 (S48)
// Raîche M, Hébert R, Dubois MF. Arch Gerontol Geriatr 2008;47(1):9–18; the wording and scoring of
// the PRISMA-7 questionnaire as BCGuidelines.ca prints it (2017): one point per "Yes" (question 6
// too), and "3 or more 'yes' answers … indicates an increased risk of frailty". Questions 1 and 2
// (older than 85, male) are the patient's record's.
const prismaAnswers = (yes: number[]) => Object.fromEntries(PRISMA7_KEYS.items.map((k, i) => [k, yes.includes(i + 3) ? "o_yes" : "o_no"]));

describe("PRISMA-7, as published", () => {
  it("asks questions 3 to 7 in the published wording; 1 and 2 are the age and sex", () => {
    const titles = prisma7Group().elements!.filter((e) => e.type === "radiogroup").map((e) => e.title);
    expect(titles).toEqual([
      "In general, do you have any health problems that require you to limit your activities?",
      "Do you need someone to help you on a regular basis?",
      "In general, do you have any health problems that require you to stay at home?",
      "If you need help, can you count on someone close to you?",
      "Do you regularly use a stick, walker or wheelchair to move about?",
    ]);
  });
  const EXAMPLES: [string, number | undefined, string, number[], string][] = [
    ["No Yes at all", 70, "female", [], "0 (Not at increased risk)"],
    ["2 Yes is not increased risk", 70, "female", [3, 4], "2 (Not at increased risk)"],
    ["3 Yes is increased risk", 70, "female", [3, 4, 5], "3 (Increased risk of frailty)"],
    ["Question 6's Yes scores too", 70, "female", [3, 4, 6], "3 (Increased risk of frailty)"],
    ["Male counts 1", 70, "male", [3, 4], "3 (Increased risk of frailty)"],
    ["Older than 85 counts 1", 86, "female", [3, 4], "3 (Increased risk of frailty)"],
    ["85 isn't older than 85", 85, "female", [3, 4], "2 (Not at increased risk)"],
    ["Every one", 90, "male", [3, 4, 5, 6, 7], "7 (Increased risk of frailty)"],
    ["No age is Incomplete", undefined, "female", [3], "Incomplete"],
    ["Sex not known is Incomplete", 70, "unknown", [3], "Incomplete"],
    ["Sex Other is Incomplete (it can't decide the male point)", 70, "other", [3], "Incomplete"],
  ];
  for (const [name, age, sex, yes, want] of EXAMPLES) {
    it(name, () => {
      expect(shown(prisma7Group(), prismaAnswers(yes), { name: "Sam Sample", age, sex })[PRISMA7_KEYS.score]).toBe(want);
    });
  }
  it("is Incomplete while a question is unanswered", () => {
    const data = prismaAnswers([3, 4, 5]);
    delete data[PRISMA7_KEYS.items[4]];
    expect(shown(prisma7Group(), data)[PRISMA7_KEYS.score]).toBe("Incomplete");
  });
});

// ---------------------------------------------------------------- Apfel (S48)
// Apfel CC et al. A simplified risk score for predicting postoperative nausea and vomiting.
// Anesthesiology 1999;91(3):693–700: female, non-smoker, a history of PONV or motion sickness, and
// post-op opioids, 1 point each; with 0, 1, 2, 3 or 4 of them, PONV in 10%, 21%, 39%, 61% and 79%.
// In an HQ it's the patient's two questions only, with no score (S48 answers 4): the opioids item
// is the clinician's, in the Apfel assessment.
describe("Apfel, as published", () => {
  const page = (): ChapterJson => ({ pages: [{ name: "p_1", title: "PONV", elements: [apfelGroup(), ...apfelClinicianPart()] }] });
  const answers = (smokes: boolean, ponv: boolean, opioids: boolean) => ({
    [APFEL_KEYS.smoking]: smokes ? "o_yes" : "o_no", [APFEL_KEYS.ponv]: ponv ? "o_yes" : "o_no", [APFEL_KEYS.opioids]: opioids ? "o_yes" : "o_no",
  });
  const score = (sex: string, data: Record<string, unknown>) => {
    const m = modelFor(page(), "clinician", { name: "Sam Sample", age: 40, sex });
    showBands(m);
    m.data = data;
    const out = (m.getQuestionByName(APFEL_KEYS.score) as unknown as { formatedValue: string }).formatedValue;
    m.dispose();
    return out;
  };
  const EXAMPLES: [string, string, boolean, boolean, boolean, string][] = [
    ["None of the four: about 10%", "male", true, false, false, "0 (PONV risk about 10%)"],
    ["Female only: about 21%", "female", true, false, false, "1 (PONV risk about 21%)"],
    ["A non-smoker only: about 21%", "male", false, false, false, "1 (PONV risk about 21%)"],
    ["Two: about 39%", "female", false, false, false, "2 (PONV risk about 39%)"],
    ["Three: about 61%", "male", false, true, true, "3 (PONV risk about 61%)"],
    ["All four: about 79%", "female", false, true, true, "4 (PONV risk about 79%)"],
  ];
  for (const [name, sex, smokes, ponv, opioids, want] of EXAMPLES) {
    it(name, () => expect(score(sex, answers(smokes, ponv, opioids))).toBe(want));
  }
  it("is Incomplete without the opioids item, or a sex of Other or Not known", () => {
    const data: Record<string, unknown> = answers(false, true, true);
    delete data[APFEL_KEYS.opioids];
    expect(score("female", data)).toBe("Incomplete");
    expect(score("other", answers(false, true, true))).toBe("Incomplete");
    expect(score("unknown", answers(false, true, true))).toBe("Incomplete");
  });
  it("in an HQ is the patient's two questions, with no score of its own", () => {
    const g = apfelGroup();
    expect(g.elements!.map((e) => [e.name, e.type])).toEqual([[APFEL_KEYS.smoking, "radiogroup"], [APFEL_KEYS.ponv, "radiogroup"]]);
    expect(apfelClinicianPart().map((e) => e.name)).toEqual([APFEL_KEYS.opioids, APFEL_KEYS.score]);
  });
});

describe("every ready-made score in an HQ", () => {
  const ids = ["stopbang", "dasi", "prisma7", "apfel", "must", "morse", "vte"];
  it("is offered: STOP-Bang, DASI, PRISMA-7, Apfel, and the patient's parts of MUST, Morse and VTE", () => {
    expect(READY_SCORES.map((r) => r.id)).toEqual(ids);
  });
  for (const id of ids) {
    const inserted = (): ChapterJson => insertReadyScore({ pages: [{ name: "p_2", title: "Set", elements: [] }] }, { page: "p_2" }, id, "q_bmi.bmi").doc;
    it(`${id}: publishes as inserted, and can't be changed`, () => {
      expect(readyScoreProblems(inserted())).toEqual([]);
      expect(publishProblems([{ name: "Measurements", doc: measures }, { name: "Set", doc: inserted() }])).toEqual([]);
      const changed = inserted();
      changed.pages![0].elements![0].elements![0].title = "Changed wording";
      expect(readyScoreProblems(changed)).toHaveLength(1);
      const points = inserted();
      const scored = points.pages![0].elements![0].elements!.find((e) => Array.isArray(e.choices))!;
      ((scored.choices as { score?: number }[])[0]).score = 9;
      expect(readyScoreProblems(points)).toHaveLength(1);
    });
  }
});

// ---------------------------------------------------------------- MUST (S49)
// BAPEN's 'Malnutrition Universal Screening Tool' flowchart (© BAPEN): Step 1 BMI kg/m² >20 (>30
// obese) = 0, 18.5–20 = 1, <18.5 = 2; Step 2 unplanned weight loss in past 3–6 months % <5 = 0,
// 5–10 = 1, >10 = 2; Step 3 "If patient is acutely ill and there has been or is likely to be no
// nutritional intake for >5 days" Score 2; Step 4 add them: 0 Low Risk, 1 Medium Risk, 2 or more
// High Risk. In an HQ it's the patient's weight-loss questions only (S49 answers 3).
describe("MUST, as published", () => {
  const page = (): ChapterJson => ({ pages: [{ name: "p_1", title: "MUST", elements: mustClinicianPart() }] });
  const run = (bmi: number | undefined, weight: number | undefined, lost: number | undefined, acute: boolean | undefined) => {
    const data: Record<string, unknown> = {};
    if (bmi !== undefined) data[MUST_KEYS.bmi] = bmi;
    if (weight !== undefined) data[MUST_KEYS.weight] = weight;
    if (lost !== undefined) data[MUST_KEYS.lostKg] = lost;
    if (acute !== undefined) data[MUST_KEYS.acute] = acute ? "o_yes" : "o_no";
    const m = modelFor(page(), "clinician", { name: "Sam Sample", age: 60, sex: "female" });
    showBands(m);
    m.data = data;
    const q = (n: string) => (m.getQuestionByName(n) as unknown as { formatedValue: string }).formatedValue;
    const out = { score: q(MUST_KEYS.score), pct: q(MUST_KEYS.pct) };
    m.dispose();
    return out;
  };
  const EXAMPLES: [string, number, number, number, boolean, string][] = [
    ["BMI over 20, no weight lost, not acutely ill: 0, low risk", 22, 70, 0, false, "0 (Low risk)"],
    ["BMI over 30 (obese) is 0", 34, 100, 0, false, "0 (Low risk)"],
    ["BMI 20.1 is over 20: 0", 20.1, 60, 0, false, "0 (Low risk)"],
    ["BMI 20 is 18.5–20: 1, medium risk", 20, 60, 0, false, "1 (Medium risk)"],
    ["BMI 18.5 is 18.5–20: 1", 18.5, 55, 0, false, "1 (Medium risk)"],
    ["BMI 18.4 is under 18.5: 2, high risk", 18.4, 55, 0, false, "2 (High risk)"],
    ["4.9% weight lost is under 5: 0", 22, 95.1, 4.9, false, "0 (Low risk)"],
    ["5% weight lost is 5–10: 1", 22, 95, 5, false, "1 (Medium risk)"],
    ["10% weight lost is 5–10: 1", 22, 90, 10, false, "1 (Medium risk)"],
    ["10.1% weight lost is over 10: 2", 22, 89.9, 10.1, false, "2 (High risk)"],
    ["Acutely ill with no intake for over 5 days: 2", 22, 70, 0, true, "2 (High risk)"],
    ["All three steps at their highest: 6", 17, 44, 6, true, "6 (High risk)"],
  ];
  for (const [name, bmi, weight, lost, acute, want] of EXAMPLES) {
    it(name, () => expect(run(bmi, weight, lost, acute).score).toBe(want));
  }
  it("shows the weight lost as a percentage of the weight before, to one decimal place", () => {
    expect(run(22, 94, 6, false).pct).toBe("6% of body weight lost");
    expect(run(22, 63.5, 4.2, false).pct).toBe("6.2% of body weight lost");
  });
  it("is Incomplete while the BMI, the weight, the weight lost or the acute disease effect is missing", () => {
    expect(run(undefined, 70, 0, false).score).toBe("Incomplete");
    expect(run(22, undefined, 0, false).score).toBe("Incomplete");
    expect(run(22, 70, undefined, false).score).toBe("Incomplete");
    expect(run(22, 70, 0, undefined).score).toBe("Incomplete");
  });
  it("words Step 3 as BAPEN requires", () => {
    const acute = mustClinicianPart().find((e) => e.name === MUST_KEYS.acute)!;
    expect(acute.title).toBe("Acute disease effect: is the patient acutely ill and there has been or is likely to be no nutritional intake for >5 days?");
  });
  it("in an HQ is the patient's weight-loss questions, with no score: how much only when they say Yes", () => {
    const g = mustGroup();
    expect(g.elements!.map((e) => [e.name, e.type])).toEqual([[MUST_KEYS.lost, "radiogroup"], [MUST_KEYS.lostKg, "text"]]);
    expect(g.elements![1]).toMatchObject({ inputType: "number", min: 0, max: 100, visibleIf: `{${MUST_KEYS.lost}} = 'o_yes'` });
    expect(mustClinicianPart().map((e) => e.name)).toEqual([MUST_KEYS.bmi, MUST_KEYS.weight, MUST_KEYS.lostKg, MUST_KEYS.acute, MUST_KEYS.pct, MUST_KEYS.score]);
  });
});

// ---------------------------------------------------------------- Morse Fall Scale (S49)
// Morse JM, Morse RM, Tylko SJ. Can J Aging 1989;8(4):366–77, as the Fall TIPS trial shows it
// (Dykes PC et al., JAMA 2010;304(17):1912–8, Figure 1): history of falls 25, secondary diagnosis
// 15, ambulatory aid 0/15/30, IV or heparin lock 20, gait 0/10/20, mental status 0/15 (0–125). The
// figure's worked example: a fall, a walker, weak gait and forgets limitations = 65.
describe("the Morse Fall Scale, as published", () => {
  const page = (): ChapterJson => ({ pages: [{ name: "p_1", title: "Falls", elements: [morseGroup(), ...morseClinicianPart()] }] });
  const score = (data: Record<string, unknown>) => {
    const m = modelFor(page(), "clinician", { name: "Sam Sample", age: 70, sex: "female" });
    showBands(m);
    m.data = data;
    const out = (m.getQuestionByName(MORSE_KEYS.score) as unknown as { formatedValue: string }).formatedValue;
    m.dispose();
    return out;
  };
  const none = { [MORSE_KEYS.falls]: "o_no", [MORSE_KEYS.secondary]: "o_no", [MORSE_KEYS.aid]: "o_aid_none", [MORSE_KEYS.iv]: "o_no", [MORSE_KEYS.gait]: "o_gait_normal", [MORSE_KEYS.mental]: "o_mental_oriented" };
  it("has the published points for each item", () => {
    const points = Object.fromEntries([morseGroup(), ...morseClinicianPart()].flatMap((e) => (e.elements ?? [e]))
      .filter((e) => Array.isArray(e.choices)).map((e) => [e.name, (e.choices as { score: number }[]).map((c) => c.score)]));
    expect(points).toEqual({
      [MORSE_KEYS.falls]: [25, 0], [MORSE_KEYS.aid]: [0, 15, 30], [MORSE_KEYS.secondary]: [15, 0], [MORSE_KEYS.iv]: [20, 0],
      [MORSE_KEYS.gait]: [0, 10, 20], [MORSE_KEYS.mental]: [0, 15],
    });
  });
  const EXAMPLES: [string, Record<string, string>, string][] = [
    ["Nothing: 0, low risk", {}, "0 (Low risk)"],
    ["The trial's worked example: 65, high risk", { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.aid]: "o_aid_walker", [MORSE_KEYS.gait]: "o_gait_weak", [MORSE_KEYS.mental]: "o_mental_forgets" }, "65 (High risk)"],
    ["An IV line only: 20, low risk", { [MORSE_KEYS.iv]: "o_yes" }, "20 (Low risk)"],
    ["A fall only: 25, moderate risk", { [MORSE_KEYS.falls]: "o_yes" }, "25 (Moderate risk)"],
    ["A fall and a secondary diagnosis: 40, moderate", { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.secondary]: "o_yes" }, "40 (Moderate risk)"],
    ["A fall and an IV line: 45, high risk", { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.iv]: "o_yes" }, "45 (High risk)"],
    ["Furniture and impaired gait: 50", { [MORSE_KEYS.aid]: "o_aid_furniture", [MORSE_KEYS.gait]: "o_gait_impaired" }, "50 (High risk)"],
    ["Everything: 125", { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.secondary]: "o_yes", [MORSE_KEYS.aid]: "o_aid_furniture", [MORSE_KEYS.iv]: "o_yes", [MORSE_KEYS.gait]: "o_gait_impaired", [MORSE_KEYS.mental]: "o_mental_forgets" }, "125 (High risk)"],
  ];
  for (const [name, change, want] of EXAMPLES) {
    it(name, () => expect(score({ ...none, ...change })).toBe(want));
  }
  it("is Incomplete while any item is unanswered", () => {
    const data: Record<string, unknown> = { ...none };
    delete data[MORSE_KEYS.gait];
    expect(score(data)).toBe("Incomplete");
  });
  it("in an HQ is the patient's two questions (a fall, a walking aid), with no score", () => {
    expect(morseGroup().elements!.map((e) => e.name)).toEqual([MORSE_KEYS.falls, MORSE_KEYS.aid]);
    expect(morseClinicianPart().map((e) => e.name)).toEqual([MORSE_KEYS.secondary, MORSE_KEYS.iv, MORSE_KEYS.gait, MORSE_KEYS.mental, MORSE_KEYS.score]);
  });
});

// ---------------------------------------------------------------- VTE (S49)
// Department of Health, Risk assessment for venous thromboembolism (VTE), 2010 (© Crown copyright;
// NICE NG89's resources): "Any tick for thrombosis risk should prompt thromboprophylaxis according to
// NICE guidance"; "Any tick [for bleeding risk] should prompt clinical staff to consider if bleeding
// risk is sufficient to preclude pharmacological intervention." Age > 60 and Obesity (BMI >30 kg/m²)
// are ticks of their own.
describe("the DoH VTE risk assessment, as published", () => {
  const page = (): ChapterJson => ({ pages: [{ name: "p_1", title: "VTE", elements: [vteGroup(), ...vteClinicianPart()] }] });
  const allNo = () => Object.fromEntries([...VTE_KEYS.thrombosis, ...VTE_KEYS.bleeding].map((k) => [k, "o_no"]));
  const run = (age: number | undefined, bmi: number | undefined, yes: string[] = []) => {
    const data: Record<string, unknown> = { ...allNo(), ...Object.fromEntries(yes.map((k) => [k, "o_yes"])) };
    if (bmi !== undefined) data[VTE_KEYS.bmi] = bmi;
    const m = modelFor(page(), "clinician", { name: "Sam Sample", age, sex: "male" });
    showBands(m);
    m.data = data;
    const q = (n: string) => (m.getQuestionByName(n) as unknown as { formatedValue: string }).formatedValue;
    const out = { score: q(VTE_KEYS.score), bleeding: q(VTE_KEYS.bleedingCount) };
    m.dispose();
    return out;
  };
  it("lists every risk factor the tool has, in its order: 11 and 8 for thrombosis, 7 and 4 for bleeding", () => {
    expect(VTE_KEYS.thrombosis).toHaveLength(19 - 2); // age and BMI are the profile's and the BMI item
    expect(VTE_KEYS.bleeding).toHaveLength(11);
    const titles = [vteGroup(), ...vteClinicianPart()].flatMap((e) => e.elements ?? [e]).map((e) => e.name);
    expect(new Set(titles).size).toBe(titles.length);
  });
  it("no tick: 0, no thrombosis risk factors", () => expect(run(45, 24).score).toBe("0 (No thrombosis risk factors ticked)"));
  it("age 61 is over 60; age 60 isn't", () => {
    expect(run(61, 24).score).toBe("1 (Thrombosis risk: consider thromboprophylaxis)");
    expect(run(60, 24).score).toBe("0 (No thrombosis risk factors ticked)");
  });
  it("a BMI of 30.1 is over 30; 30 isn't", () => {
    expect(run(45, 30.1).score).toBe("1 (Thrombosis risk: consider thromboprophylaxis)");
    expect(run(45, 30).score).toBe("0 (No thrombosis risk factors ticked)");
  });
  it("counts each thrombosis tick; the bleeding ticks are counted apart", () => {
    const r = run(65, 32, [VTE_KEYS.thrombosis[0], VTE_KEYS.thrombosis[12], VTE_KEYS.bleeding[2]]);
    expect(r.score).toBe("4 (Thrombosis risk: consider thromboprophylaxis)");
    expect(r.bleeding).toBe("Bleeding risk factors: 1");
  });
  it("is Incomplete without the age, the BMI or any item", () => {
    expect(run(undefined, 24).score).toBe("Incomplete");
    expect(run(45, undefined).score).toBe("Incomplete");
  });
  it("in an HQ doesn't ask a patient recorded as male about pregnancy, an oestrogen contraceptive or HRT (verification 1)", () => {
    const shownTo = (sex: string) => {
      const m = modelFor({ pages: [{ name: "p_1", title: "Clots", elements: [vteGroup()] }] }, "patient", { name: "Sam Sample", age: 50, sex });
      const names = m.getAllQuestions(true).map((q) => q.name);
      m.dispose();
      return names;
    };
    const asked = ["vte_hrt", "vte_contraceptive", "vte_pregnancy"];
    expect(shownTo("male").filter((n) => asked.includes(n))).toEqual([]);
    expect(shownTo("male")).toHaveLength(6);
    for (const sex of ["female", "other", "unknown"]) expect(shownTo(sex)).toEqual(VTE_KEYS.patient);
  });
  it("in an HQ is the patient's own risk factors, with no score", () => {
    const g = vteGroup();
    expect(g.elements!.every((e) => e.type === "radiogroup")).toBe(true);
    expect(g.elements!.map((e) => e.name)).toEqual(VTE_KEYS.patient);
  });
});

// ---------------------------------------------------------------- Rockwood (S49)
// The Clinical Frailty Scale, version 2.0 (2020; © Dalhousie University): nine levels, 1 Very Fit
// to 9 Terminally Ill; 2020 renamed 2 "Fit", 4 "Living with Very Mild Frailty" and 5–8 "Living
// with…" mild, moderate, severe and very severe frailty. A clinician's judgement: never in an HQ.
describe("the Clinical Frailty Scale", () => {
  it("is the nine levels of version 2.0, the level its score", () => {
    const [level, score] = cfsElements();
    expect((level.choices as { text: string; score: number }[]).map((c) => [c.score, c.text])).toEqual([
      [1, "1: Very Fit"], [2, "2: Fit"], [3, "3: Managing Well"], [4, "4: Living with Very Mild Frailty"], [5, "5: Living with Mild Frailty"],
      [6, "6: Living with Moderate Frailty"], [7, "7: Living with Severe Frailty"], [8, "8: Living with Very Severe Frailty"], [9, "9: Terminally Ill"],
    ]);
    const m = modelFor({ pages: [{ name: "p_1", title: "CFS", elements: cfsElements() }] }, "clinician", { name: "Sam Sample", age: 80, sex: "male" });
    showBands(m);
    const shownScore = () => (m.getQuestionByName(score.name) as unknown as { formatedValue: string }).formatedValue;
    m.data = {};
    expect(shownScore()).toBe("Incomplete");
    m.data = { [level.name]: "o_cfs_5" };
    expect(shownScore()).toBe("5 (Living with Mild Frailty)");
    m.data = { [level.name]: "o_cfs_9" };
    expect(shownScore()).toBe("9 (Terminally Ill)");
    m.dispose();
  });
  it("can't be inserted in an HQ", () => {
    expect(READY_SCORES.some((r) => r.id === "rockwood")).toBe(false);
  });
});

// S49 verification 2 (Rahul, 13 Nov): questions about a recent time are asked afresh, never copied
// from an earlier HQ, as the library's "What do you weigh now?" and pregnancy questions.
describe("the ready-made parts' recent-time questions", () => {
  const fresh = (g: ReturnType<typeof mustGroup>) => g.elements!.filter((e) => e.alwaysFresh === true).map((e) => e.name);
  it("are always asked fresh: MUST's weight loss and amount, Morse's fall, VTE's pregnancy", () => {
    expect(fresh(mustGroup())).toEqual([MUST_KEYS.lost, MUST_KEYS.lostKg]);
    expect(fresh(morseGroup())).toEqual([MORSE_KEYS.falls]);
    expect(fresh(vteGroup())).toEqual(["vte_pregnancy"]);
  });
});

// S49 verification 3, finding 5: the seeded Demo risk HQ and the e2e copy are written from this
// file by hand; a wording change here without regenerating them would seed a "changed" ready-made part.
describe("the HQ files holding ready-made parts", () => {
  for (const file of ["../../../apps/api/cmd/seed/demo-risk-hq.json", "../../../apps/api/cmd/seed/demo-fitness-hq.json", "../../../e2e/data/risk-hq.json", "../../../e2e/data/fitness-hq.json"]) {
    it(`${file.split("/").pop()} holds them as ready.ts builds them`, () => {
      const hq = JSON.parse(readFileSync(new URL(file, import.meta.url), "utf8")) as { chapters: { content: ChapterJson }[] };
      for (const c of hq.chapters) expect(readyScoreProblems(c.content)).toEqual([]);
    });
  }
});
