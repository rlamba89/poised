// STOP-Bang as published (S38): the scoring criteria of the updated questionnaire (Chung et al.,
// Chest 2016; stopbang.ca), each as a worked example. These are the ready-made score's test cases:
// they run with `make test`, and the score can't be changed in an HQ (readyScoreProblems).
import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { publishProblems } from "./publish";
import { registerClinicalProperties } from "./properties";
import { showBands } from "./questionTypes";
import { insertReadyScore, readyScoreAround, readyScoreProblems, STOP_BANG_KEYS as K, stopBangGroup } from "./ready";
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
