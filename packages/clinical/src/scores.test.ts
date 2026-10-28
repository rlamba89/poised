import { describe, expect, it } from "vitest";
import type { ChapterJson, ElementJson } from "./doc";
import { calcFormOf, formatCalc } from "./fields";
import { logicProblems, pointsSubjects } from "./logic";
import { computeOutputs } from "./outputs";
import { poaSummary } from "./poa";
import { publishProblems } from "./publish";
import { registerClinicalProperties } from "./properties";
import { combineChapters, givenEarlier } from "./questionSets";
import { earlierContexts, isSetComplete, isSetReadyToSend, shownSets } from "./respondent";
import { applyEarlier, bandWhenSubjects, formatBandWhen, formatScore, linesToSave, parseBandWhen, parseScore, scoredQuestions, type PointsLine, type ScoreForm } from "./scores";
import { modelFor, runTestCase, type SamplePatient, type TestCase } from "./testcases";
import { renameReferences } from "./library";
import { showBands } from "./questionTypes";

registerClinicalProperties();

const yesNo = (name: string, title: string, extra: object = {}) => ({
  type: "radiogroup", name, title, yesNo: true, isRequired: true,
  choices: [{ value: "o_yes", text: "Yes", score: 1 }, { value: "o_no", text: "No", score: 0 }], ...extra,
});

/** Set 1: two scored questions, one only shown to smokers, and a BMI question. */
const sleep: ChapterJson = {
  pages: [{
    name: "p_1", title: "Sleep", elements: [
      yesNo("q_snore", "Do you snore?"),
      yesNo("q_tired", "Are you tired in the day?"),
      { type: "radiogroup", name: "q_smoke", title: "Do you smoke?", choices: [{ value: "o_y", text: "Yes" }, { value: "o_n", text: "No" }] },
      yesNo("q_cough", "Do you cough?", { visibleIf: "{q_smoke} = 'o_y'" }),
      { type: "bmi", name: "q_bmi", title: "BMI" },
    ],
  }],
};

const high = { id: "out_high", codes: [], note: { text: "High risk of sleep apnoea", category: "Airway" } };
const risk = (form: ScoreForm, extra: object = {}): ElementJson => ({
  type: "expression", name: "q_risk", title: "Sleep risk", clinicianOnly: true, calcKind: "score", expression: formatScore(form),
  bands: [{ id: "b_low", label: "Low", max: 2 }, { id: "b_high", label: "High", clinicalOutputs: [high] }], ...extra,
});

/** The STOP-Bang-like score of S38's brief: two questions from set 1, one here, age, sex and BMI. */
const form: ScoreForm = {
  kind: "score",
  questions: ["q_snore", "q_tired", "q_neck"],
  lines: [
    { points: 1, rule: { left: "patientAge", op: "gt", value: 50 } },
    { points: 1, rule: { left: "patientSex", op: "eq", value: "male" } },
    { points: 1, rule: { left: "q_bmi.bmi", op: "gt", value: 35 } },
  ],
};
const neck = (calc: ElementJson): ChapterJson => ({ pages: [{ name: "p_2", title: "Neck", elements: [yesNo("q_neck", "Is your neck large?"), calc] }] });
const later: ChapterJson = { pages: [{ name: "p_3", elements: [yesNo("q_later", "Later question")] }] };

const sets = (two = neck(risk(form)), one = sleep) => [
  { id: "s1", name: "Sleep", content: one },
  { id: "s2", name: "Neck", content: two },
];
const man51: SamplePatient = { name: "Sam Sample", age: 51, sex: "male" };

/** Set 2's score for these answers, in the clinician's view, as every view builds it. */
function scoreFor(answers: Record<string, Record<string, unknown>>, patient = man51, hq = sets()) {
  const ctx = earlierContexts(hq, answers, "clinician", patient);
  const m = modelFor(hq[1].content, "clinician", patient, ctx.get("s2"));
  showBands(m);
  m.data = answers.s2 ?? {};
  const q = m.getQuestionByName("q_risk");
  const out = { value: q.value, display: (q as unknown as { formatedValue: string }).formatedValue, result: computeOutputs(m) };
  m.dispose();
  return out;
}

const allYes = { s1: { q_snore: "o_yes", q_tired: "o_yes", q_smoke: "o_n", q_bmi: { height: 170, weight: 104 } }, s2: { q_neck: "o_yes" } };

describe("the score form (S38)", () => {
  it("writes and reads back points lines", () => {
    const expr = formatScore(form);
    expect(expr).toBe(
      "iif(allAnswered('q_snore', 'q_tired', 'q_neck', 'patientAge', 'patientSex', 'q_bmi.bmi'), " +
        "score('q_snore', 'q_tired', 'q_neck') + iif({patientAge} > 50, 1, 0) + iif({patientSex} = 'male', 1, 0) + iif({q_bmi.bmi} > 35, 1, 0), '')",
    );
    expect(parseScore(expr)).toEqual(form);
    expect(calcFormOf(expr)).toEqual(form);
    expect(formatCalc(calcFormOf(expr))).toBe(expr);
    // An earlier answer and another score's band.
    const more: ScoreForm = {
      kind: "score", questions: [],
      lines: [{ points: 2, rule: { left: "q_smoke", op: "eq", value: "o_y" } }, { points: 1, rule: { left: "q_other", fn: "band", op: "eq", value: "b_high" } }],
    };
    expect(parseScore(formatScore(more))).toEqual(more);
  });

  it("leaves out unfinished lines, and reads anything else as a custom formula", () => {
    expect(formatScore({ kind: "score", questions: [], lines: [{ points: 1, rule: { left: "patientAge", op: "gt" } }] })).toBe("");
    expect(parseScore("score('q_a')")).toBeUndefined();
    expect(parseScore("iif(allAnswered('q_a'), score('q_a') * 2, '')")).toBeUndefined();
    expect(calcFormOf("iif(allAnswered('q_a'), score('q_a') * 2, '')")).toEqual({ kind: "custom", expression: "iif(allAnswered('q_a'), score('q_a') * 2, '')" });
  });
});

describe("a score across Question Sets (S38)", () => {
  it("adds earlier sets' points, this set's, age, sex and BMI, with the band and its disclosure", () => {
    const r = scoreFor(allYes);
    expect(r.value).toBe(6);
    expect(r.display).toBe("6 (High)");
    expect(r.result.outputs.map((o) => o.noteText)).toEqual(["High risk of sleep apnoea"]);
    expect(r.result.scores).toEqual([{ questionId: "q_risk", title: "Sleep risk", value: 6, band: { id: "b_high", label: "High" }, incomplete: false }]);
  });

  it("works out again when the earlier answers are given after the answers here (the preview)", () => {
    const hq = sets();
    const m = modelFor(hq[1].content, "clinician", man51);
    m.data = allYes.s2;
    expect(m.getQuestionByName("q_risk").value ?? "").toBe("");
    applyEarlier(m, earlierContexts(hq, allYes, "clinician", man51).get("s2"));
    expect(m.getQuestionByName("q_risk").value).toBe(6);
    applyEarlier(m, { doc: combineChapters([{ name: "Sleep", doc: sleep }]), values: { ...allYes.s1, q_tired: undefined } });
    expect(m.getQuestionByName("q_risk").value ?? "").toBe("");
  });

  it("works out when this set is answered after the earlier answers are given (the preview after Clear answers)", () => {
    const hq = sets();
    const m = modelFor(hq[1].content, "clinician", man51);
    applyEarlier(m, earlierContexts(hq, allYes, "clinician", man51).get("s2"));
    m.setValue("q_neck", "o_yes");
    expect(m.getQuestionByName("q_risk").value).toBe(6);
    // A change made here (a clinician's correction on Validate) is counted at once.
    m.setValue("q_neck", "o_no");
    expect(m.getQuestionByName("q_risk").value).toBe(5);
  });

  it("uses an earlier set's height though a BMI question here has a height inside it (verification 2)", () => {
    const here: ChapterJson = { pages: [{ name: "p_2", elements: [{ type: "bmi", name: "q_bmi2" }, risk({ kind: "score", questions: [], lines: [{ points: 1, rule: { left: "height", op: "gt", value: 160 } }] })] }] };
    const m = modelFor(here, "clinician", man51, { doc: { pages: [{ name: "p_1", elements: [{ type: "text", inputType: "number", name: "height" }] }] }, values: { height: 170 } });
    expect(m.getQuestionByName("q_risk").value).toBe(1);
  });

  it("gives a point over 50, not at 50; for male, not female; for BMI over 35, not at 35", () => {
    const some = { s1: { ...allYes.s1, q_tired: "o_no" }, s2: { q_neck: "o_no" } }; // snore only: 1
    expect(scoreFor(some).value).toBe(4);
    expect(scoreFor(some, { ...man51, age: 50 }).value).toBe(3);
    expect(scoreFor(some, { ...man51, sex: "female" }).value).toBe(3);
    // 170 cm, 101.2 kg is a BMI of 35.0: not over 35.
    expect(scoreFor({ ...some, s1: { ...some.s1, q_bmi: { height: 170, weight: 101.2 } } }).value).toBe(3);
    // An author's own score: sex "Other" is an answer, with no male point (only the ready-made
    // STOP-Bang is Incomplete for it, S44c: ready.test.ts).
    expect(scoreFor(some, { ...man51, sex: "other" }).value).toBe(3);
  });

  it("is Incomplete while an input has no answer, with no band and no disclosure", () => {
    for (const r of [
      scoreFor({ s1: { ...allYes.s1, q_tired: undefined }, s2: allYes.s2 }), // an earlier answer
      scoreFor({ s1: allYes.s1, s2: {} }), // this set's
      scoreFor(allYes, { ...man51, age: undefined }), // the patient's age
      scoreFor(allYes, { ...man51, sex: "unknown" }), // sex not known
      scoreFor({ s1: { ...allYes.s1, q_bmi: undefined }, s2: allYes.s2 }), // the BMI
    ]) {
      expect(r.value ?? "").toBe("");
      expect(r.display).toBe("Incomplete");
      expect(r.result.outputs).toEqual([]);
      expect(r.result.scores).toEqual([{ questionId: "q_risk", title: "Sleep risk", incomplete: true }]);
    }
  });

  it("adds 0 for a question a condition hides, but is Incomplete for one shown and skipped (S44a)", () => {
    // q_cough is shown only to smokers; its old answer stays in the data but adds nothing.
    const withCough = neck(risk({ kind: "score", questions: ["q_snore", "q_cough"] }));
    expect(scoreFor({ s1: { q_snore: "o_yes", q_smoke: "o_y", q_cough: "o_yes" } }, man51, sets(withCough)).value).toBe(2);
    const hiddenCough = scoreFor({ s1: { q_snore: "o_yes", q_smoke: "o_n", q_cough: "o_yes" } }, man51, sets(withCough));
    expect(hiddenCough.value).toBe(1);
    expect(hiddenCough.display).toBe("1 (Low)");
    expect(hiddenCough.result.scores).toEqual([{ questionId: "q_risk", title: "Sleep risk", value: 1, band: { id: "b_low", label: "Low" }, incomplete: false }]);
    // Shown (a smoker) and not answered: Incomplete.
    expect(scoreFor({ s1: { q_snore: "o_yes", q_smoke: "o_y" } }, man51, sets(withCough)).display).toBe("Incomplete");
  });

  it("adds 0 for every question of a Question Set a condition hides, and its lines don't fire (S44a)", () => {
    // Set 1 hidden: snoring, tiredness and the BMI line add nothing; the neck, age and sex do (3: High).
    const hidden = { ...sleep, chapterVisibleIf: "{patientAge} < 18" };
    const r = scoreFor({ s1: allYes.s1, s2: { q_neck: "o_yes" } }, man51, sets(neck(risk(form)), hidden));
    expect(r.value).toBe(3);
    expect(r.display).toBe("3 (High)");
    // This set's own question still counts: unanswered, Incomplete.
    expect(scoreFor({ s1: allYes.s1, s2: {} }, man51, sets(neck(risk(form)), hidden)).display).toBe("Incomplete");
  });

  it("doesn't fire a points line that tests a hidden answer, in this set or an earlier one (S44a)", () => {
    // Earlier: q_cough hidden (not a smoker) with an old Yes; "is answered" would fire on an old value too.
    const earlierLine = neck(risk({
      kind: "score", questions: ["q_neck"],
      lines: [{ points: 5, rule: { left: "q_cough", op: "eq", value: "o_yes" } }, { points: 7, rule: { left: "q_cough", op: "empty" } }],
    }));
    expect(scoreFor({ s1: { q_smoke: "o_n", q_cough: "o_yes" }, s2: { q_neck: "o_no" } }, man51, sets(earlierLine)).value).toBe(0);
    expect(scoreFor({ s1: { q_smoke: "o_y", q_cough: "o_yes" }, s2: { q_neck: "o_no" } }, man51, sets(earlierLine)).value).toBe(5);
    // This set: the Calculation comes first on the page, before the question its line tests.
    const own: ChapterJson = {
      pages: [{
        name: "p_2", elements: [
          risk({ kind: "score", questions: ["q_neck", "q_pillows"], lines: [{ points: 4, rule: { left: "q_pillows", op: "eq", value: "o_yes" } }] }),
          yesNo("q_neck", "Is your neck large?"),
          yesNo("q_pillows", "More than two pillows?", { visibleIf: "{q_neck} = 'o_yes'" }),
        ],
      }],
    };
    const m = modelFor(own, "clinician", man51);
    m.data = { q_neck: "o_yes", q_pillows: "o_yes" };
    expect(m.getQuestionByName("q_risk").value).toBe(6);
    m.setValue("q_neck", "o_no"); // q_pillows goes, its old Yes stays in the data
    expect(m.getQuestionByName("q_risk").value).toBe(0);
    m.setValue("q_neck", "o_yes");
    expect(m.getQuestionByName("q_risk").value).toBe(6);
    m.dispose();
    const fresh = modelFor(own, "clinician", man51);
    fresh.data = { q_neck: "o_no", q_pillows: "o_yes" };
    expect(fresh.getQuestionByName("q_risk").value).toBe(0);
    fresh.dispose();
  });

  it("adds 0 for a hidden page's question", () => {
    const paged: ChapterJson = {
      pages: [
        { name: "p_2", elements: [yesNo("q_neck", "Is your neck large?"), risk({ kind: "score", questions: ["q_neck", "q_extra"] })] },
        { name: "p_x", visibleIf: "{q_neck} = 'o_yes'", elements: [yesNo("q_extra", "Extra")] },
      ],
    };
    const m = modelFor(paged, "clinician", man51);
    m.data = { q_neck: "o_no", q_extra: "o_yes" };
    expect(m.getQuestionByName("q_risk").value).toBe(0);
    m.setValue("q_neck", "o_yes");
    expect(m.getQuestionByName("q_risk").value).toBe(2);
    m.dispose();
  });

  it("uses an earlier set's score band, and an earlier answer, in points lines", () => {
    const first: ChapterJson = {
      pages: [{
        name: "p_1", elements: [
          ...sleep.pages![0].elements!,
          risk({ kind: "score", questions: ["q_snore", "q_tired"] }, { name: "q_first", bands: [{ id: "b_lo", label: "Low", max: 1 }, { id: "b_hi", label: "High" }] }),
        ],
      }],
    };
    const second = neck(risk({
      kind: "score", questions: [],
      lines: [{ points: 3, rule: { left: "q_first", fn: "band", op: "eq", value: "b_hi" } }, { points: 1, rule: { left: "q_smoke", op: "eq", value: "o_y" } }],
    }));
    const hq = sets(second, first);
    expect(scoreFor({ s1: { ...allYes.s1, q_smoke: "o_y" } }, man51, hq).value).toBe(4);
    expect(scoreFor({ s1: { ...allYes.s1, q_tired: "o_no" } }, man51, hq).value).toBe(0);
  });

  it("works in the patient's view, the set's completeness and the POA Summary", () => {
    const hq = sets(neck(risk(form, { clinicianOnly: false })));
    const ctx = earlierContexts(hq, allYes, "patient", man51);
    const m = modelFor(hq[1].content, "patient", man51, ctx.get("s2"));
    m.data = allYes.s2;
    expect(m.getQuestionByName("q_risk").value).toBe(6);
    m.dispose();
    // A question shown only for a high score is required once the earlier answers make it high.
    const followUp = neck(risk(form));
    followUp.pages![0].elements!.push({ type: "comment", name: "q_why", isRequired: true, visibleIf: "band('q_risk') = 'b_high'" });
    const hq2 = sets(followUp);
    const ctx2 = earlierContexts(hq2, allYes, "clinician", man51);
    expect(isSetComplete(followUp, allYes.s2, "clinician", man51, ctx2.get("s2"))).toBe(false);
    expect(isSetComplete(followUp, allYes.s2, "clinician", man51)).toBe(true); // without them it's Incomplete, so not shown
    // The POA Summary lists the band's disclosure.
    const poa = poaSummary(sets(), allYes, "clinician", man51);
    expect(poa.flatMap((s) => s.pages.flatMap((p) => p.blocks.flatMap((b) => b.notes.map((n) => n.text))))).toEqual(["High risk of sleep apnoea"]);
  });

  it("keeps an older single-set score as it was: the total of what is answered", () => {
    const old: ChapterJson = {
      pages: [{
        name: "p_1", elements: [
          yesNo("q_a", "A"), yesNo("q_b", "B"),
          { type: "expression", name: "q_old", expression: "score('q_a', 'q_b')", bands: [{ id: "b_l", label: "Low", max: 1 }, { id: "b_h", label: "High" }] },
        ],
      }],
    };
    // Nothing in it uses another set, so its sets get nothing from earlier ones.
    const hq = [{ id: "s1", name: "Sleep", content: sleep }, { id: "s2", name: "Old", content: old }];
    expect(earlierContexts(hq, { s1: allYes.s1 }, "clinician", man51).size).toBe(0);
    const m = modelFor(old, "clinician", man51);
    showBands(m);
    m.data = { q_a: "o_yes" };
    expect(m.getQuestionByName("q_old").value).toBe(1);
    expect(m.getQuestionByName("q_old").displayValue).toBe("1 (Low)");
  });
});

describe("checking a score that uses other Question Sets (S38)", () => {
  const two = neck(risk(form));
  it("allows earlier sets' questions, and flags later or missing ones", () => {
    const earlier = combineChapters([{ name: "Sleep", doc: sleep }]);
    expect(logicProblems(two, { earlier, later })).toEqual([]);
    // Without the earlier sets, they look missing (as before S38).
    expect(logicProblems(two).map((p) => p.message)).toEqual(["it tests something that no longer exists (q_bmi, q_snore, q_tired)"]);
    const bad = neck(risk({ ...form, questions: ["q_snore", "q_later", "q_gone"] }));
    expect(logicProblems(bad, { earlier, later }).map((p) => p.message)).toEqual([
      "it tests something that no longer exists (q_gone)",
      "it uses a question in a later Question Set (q_later)",
    ]);
    // An earlier answer that no longer exists.
    const gone = neck(risk({ kind: "score", questions: [], lines: [{ points: 1, rule: { left: "q_smoke", op: "eq", value: "o_gone" } }] }));
    expect(logicProblems(gone, { earlier }).map((p) => p.message)).toEqual(["it tests an answer that no longer exists (“Do you smoke?”: o_gone)"]);
    // Only a Calculation may use an earlier set: a question's condition still may not.
    const cond = neck(risk(form));
    cond.pages![0].elements![0] = { ...cond.pages![0].elements![0], visibleIf: "{q_snore} = 'o_yes'" };
    expect(logicProblems(cond, { earlier }).map((p) => p.message)).toEqual(["it tests a question in another Question Set (q_snore)"]);
  });

  it("blocks publishing a score that uses a later set's question or a deleted one", () => {
    const named = (docs: ChapterJson[]) => docs.map((doc, i) => ({ name: `Set ${i + 1}`, doc }));
    expect(publishProblems(named([sleep, two]))).toEqual([]);
    expect(publishProblems(named([two, sleep])).map((p) => p.message)).toEqual(['"Sleep risk": it uses a question in a later Question Set (q_bmi, q_snore, q_tired)']);
    const deleted = { pages: [{ ...sleep.pages![0], elements: sleep.pages![0].elements!.filter((e) => e.name !== "q_tired") }] };
    expect(publishProblems(named([deleted, two])).map((p) => p.message)).toEqual(['"Sleep risk": it tests something that no longer exists (q_tired)']);
  });

  it("renames a key in another set's score", () => {
    const renamed = renameReferences(two, new Map([["q_bmi", "bmi_measured"], ["q_snore", "snoring"]]));
    expect(String(renamed.pages![0].elements![1].expression)).toContain("allAnswered('snoring', 'q_tired', 'q_neck', 'patientAge', 'patientSex', 'bmi_measured.bmi')");
    expect(String(renamed.pages![0].elements![1].expression)).toContain("score('snoring', 'q_tired', 'q_neck')");
    expect(String(renamed.pages![0].elements![1].expression)).toContain("{bmi_measured.bmi} > 35");
  });
});

describe("test cases with earlier sets' answers (S38)", () => {
  const two = neck(risk(form));
  const earlier = [{ name: "Sleep", doc: sleep }];
  const tc: TestCase = {
    id: "t_1", name: "High", viewer: "clinician", patient: man51, answers: { q_neck: "o_yes" },
    earlier: { q_snore: "o_yes", q_tired: "o_yes", q_bmi: { height: 170, weight: 104, bmi: 36 } },
    expected: ["Note (Airway): High risk of sleep apnoea"],
  };
  it("runs a case with the earlier answers it gives", () => {
    expect(runTestCase(two, tc, earlier).pass).toBe(true);
    // Without them, the score is Incomplete, so no disclosure.
    expect(runTestCase(two, { ...tc, earlier: undefined }, earlier).missing).toEqual(["Note (Airway): High risk of sleep apnoea"]);
  });
  it("leaves out the answers of an earlier set the case's patient wouldn't see (test report 2, F-1), counting them 0 (S44a)", () => {
    const womenOnly = [{ name: "Sleep", doc: { ...sleep, chapterVisibleIf: "{patientSex} = 'female'" } }];
    // Snoring, tiredness and the BMI add 0: neck, age and sex give 3, High.
    expect(runTestCase(two, tc, womenOnly).pass).toBe(true);
    expect(runTestCase(two, { ...tc, patient: { ...man51, age: 40 } }, womenOnly).missing).toEqual(["Note (Airway): High risk of sleep apnoea"]);
    const given = givenEarlier(womenOnly, tc.earlier!, man51, "clinician");
    expect(given.hidden).toEqual(["Sleep"]);
    expect(given.earlier.values).toEqual({});
    expect(given.earlier.hidden).toEqual(expect.arrayContaining(["q_snore", "q_tired", "q_bmi"]));
    expect(givenEarlier(womenOnly, tc.earlier!, { ...man51, sex: "female" }, "clinician")).toEqual({
      earlier: { doc: combineChapters(womenOnly), values: tc.earlier }, hidden: [],
    });
  });

  it("publishes with the case passing, and not when it fails", () => {
    const withCase = { ...two, testCases: [tc] };
    expect(publishProblems([{ name: "Sleep", doc: sleep }, { name: "Neck", doc: withCase }])).toEqual([]);
    // A woman of 40 who doesn't snore and isn't tired: 2 points (neck and BMI), Low.
    const failing = { ...two, testCases: [{ ...tc, patient: { ...man51, age: 40, sex: "female" }, earlier: { ...tc.earlier, q_tired: "o_no", q_snore: "o_no" } }] };
    expect(publishProblems([{ name: "Sleep", doc: sleep }, { name: "Neck", doc: failing }]).map((p) => p.message)).toEqual(["Test case “High” fails"]);
  });
});

describe("what points lines can test (S38)", () => {
  it("offers Patient, Measurements, Other scores, then answers grouped Set › Page", () => {
    const earlier = combineChapters([{ name: "Sleep", doc: sleep }]);
    const subjects = pointsSubjects(neck(risk(form)), earlier, "q_risk");
    expect(subjects.map((s) => [s.group, s.label])).toEqual([
      ["Patient", "Age"],
      ["Patient", "Sex"],
      ["Measurements", "BMI (BMI)"],
      ["Sleep › Sleep", "Do you snore?"],
      ["Sleep › Sleep", "Are you tired in the day?"],
      ["Sleep › Sleep", "Do you smoke?"],
      ["Sleep › Sleep", "Do you cough?"],
      ["This Question Set › Neck", "Is your neck large?"],
    ]);
  });
});

describe("saving points lines (verification 1)", () => {
  const age: PointsLine = { points: 1, rule: { left: "patientAge", op: "gt", value: 50 } };
  const male: PointsLine = { points: 1, rule: { left: "patientSex", op: "eq", value: "male" } };
  const unfinished: PointsLine = { points: 1, rule: { left: "", op: "eq" } };
  it("saves a removed or changed line though another line is unfinished (F1)", () => {
    expect(linesToSave([unfinished], [age])).toEqual([]);
    expect(linesToSave([{ ...age, points: 2 }, unfinished], [age])).toEqual([{ ...age, points: 2 }]);
    expect(linesToSave([age, male, unfinished], [age])).toEqual([age, male]);
  });
  it("saves nothing while only unfinished lines changed", () => {
    expect(linesToSave([age, unfinished], [age])).toBeUndefined();
    expect(linesToSave([age], [age])).toBeUndefined();
  });
});

describe("a key repeated in two Question Sets (verification 1, F2)", () => {
  // An older draft may repeat a key; Publish refuses it, but the editor mustn't break.
  const twice = neck(risk(form));
  twice.pages![0].elements!.push(yesNo("q_snore", "Do you snore (asked again)?"));
  const earlier = combineChapters([{ name: "Sleep", doc: sleep }]);
  it("lists each key once in the points lines' menu, this set's question winning", () => {
    const subjects = pointsSubjects(twice, earlier, "q_risk");
    expect(subjects.filter((s) => s.id === "q_snore").map((s) => s.label)).toEqual(["Do you snore (asked again)?"]);
    expect(new Set(subjects.map((s) => s.id)).size).toBe(subjects.length);
  });
  it("lists each key once among the questions to add up", () => {
    const groups = scoredQuestions(twice, earlier, "q_risk");
    const values = groups.flatMap((g) => g.items.map((i) => i.value));
    expect(new Set(values).size).toBe(values.length);
    expect(groups).toEqual([
      { group: "Sleep › Sleep", items: [{ value: "q_tired", label: "Are you tired in the day?" }, { value: "q_cough", label: "Do you cough?" }] },
      { group: "This Question Set", items: [{ value: "q_neck", label: "Is your neck large?" }, { value: "q_snore", label: "Do you snore (asked again)?" }] },
    ]);
  });
});

describe("the questions a score can add up (S44a, S38 verification 3)", () => {
  it("leaves out scored questions inside a repeating group, which score() can't read", () => {
    const listed: ChapterJson = { pages: [{ name: "p_1", title: "Meds", elements: [
      yesNo("q_top", "Top level"),
      { type: "panel", name: "g_1", elements: [yesNo("q_in_group", "In a group")] },
      { type: "paneldynamic", name: "q_list", templateElements: [yesNo("q_in_list", "In a list")] },
    ] }] };
    const here = scoredQuestions(listed, undefined, "q_risk").flatMap((g) => g.items.map((i) => i.value));
    expect(here).toEqual(["q_top", "q_in_group"]);
    const earlier = scoredQuestions(neck(risk(form)), listed, "q_risk").find((g) => g.group === "Meds")!.items.map((i) => i.value);
    expect(earlier).toEqual(["q_top", "q_in_group"]);
  });
});

describe("Question Set conditions on a score that uses earlier sets (verification 2, F4)", () => {
  // Set 1: q_a. Set 2: score s from q_a. Set 3: shown only when s >= 1.
  const one: ChapterJson = { pages: [{ name: "p_1", elements: [yesNo("q_a", "A?")] }] };
  const two: ChapterJson = {
    pages: [{ name: "p_2", elements: [yesNo("q_b", "B?"), { type: "expression", name: "s", clinicianOnly: true, expression: formatScore({ kind: "score", questions: ["q_a"] }) }] }],
  };
  const three: ChapterJson = { chapterVisibleIf: "{s} >= 1", pages: [{ name: "p_3", elements: [{ ...yesNo("q_c", "Follow-up?"), isRequired: true }] }] };
  const hq = [
    { id: "s1", name: "One", content: one },
    { id: "s2", name: "Two", content: two },
    { id: "s3", name: "Three", content: three },
  ];
  const ids = (answers: Record<string, Record<string, unknown>>, viewer: "patient" | "clinician" = "clinician") => shownSets(hq, answers, viewer, man51).map((x) => x.id);

  it("hides a set when an earlier answer changed after the score was saved", () => {
    // Set 2 was saved with s = 1; then the patient changed set 1 to No: the score is 0 now.
    expect(ids({ s1: { q_a: "o_no" }, s2: { q_b: "o_yes", s: 1 } })).toEqual(["s1", "s2"]);
  });

  it("shows a set when an earlier answer changed after the score was saved", () => {
    expect(ids({ s1: { q_a: "o_yes" }, s2: { q_b: "o_yes", s: 0 } })).toEqual(["s1", "s2", "s3"]);
    // Never saved at all (set 2 opened by nobody yet): worked out from set 1.
    expect(ids({ s1: { q_a: "o_yes" } })).toEqual(["s1", "s2", "s3"]);
  });

  it("holds Send back until the set the worked-out score shows is answered", () => {
    const answers = { s1: { q_a: "o_yes" }, s2: { q_b: "o_yes", s: 0 } };
    const shown = shownSets(hq, answers, "clinician", man51);
    const ctx = earlierContexts(hq, answers, "clinician", man51);
    const ready = shown.map((x) => isSetReadyToSend(x.content, answers[x.id as keyof typeof answers], undefined, man51, ctx.get(x.id)));
    expect(shown.map((x) => x.id)).toEqual(["s1", "s2", "s3"]);
    expect(ready).toEqual([true, true, false]);
  });

  it("keeps older HQs as they were: a set's saved values decide", () => {
    const own: ChapterJson = { pages: [{ name: "p_2", elements: [yesNo("q_b", "B?"), { type: "expression", name: "s", expression: "score('q_b')" }] }] };
    const old = [{ id: "s1", name: "One", content: one }, { id: "s2", name: "Two", content: own }, { id: "s3", name: "Three", content: three }];
    expect(shownSets(old, { s2: { q_b: "o_no", s: 1 } }, "clinician", man51).map((x) => x.id)).toEqual(["s1", "s2", "s3"]);
  });
});

describe("an author's band with \"also when\" (S44a)", () => {
  // Snoring points 0–2; High from 3, and also when the score is 2 or more and the patient is male.
  const when = formatBandWhen([{ left: "q_score", op: "gte", value: 2 }, { left: "patientSex", op: "eq", value: "male" }])!;
  const doc = (w = when): ChapterJson => ({ pages: [{ name: "p_1", elements: [
    yesNo("q_snore", "Do you snore?"), yesNo("q_tired", "Tired?"),
    { type: "expression", name: "q_score", title: "Sleep score", calcKind: "score", expression: formatScore({ kind: "score", questions: ["q_snore", "q_tired"] }),
      bands: [{ id: "b_low", label: "Low", max: 2 }, { id: "b_high", label: "High", when: w, clinicalOutputs: [high] }] },
  ] }] });
  const run = (patient: SamplePatient, data: Record<string, unknown>, d = doc()) => {
    const m = modelFor(d, "clinician", patient);
    showBands(m);
    m.data = data;
    const q = m.getQuestionByName("q_score");
    const out = { display: (q as unknown as { formatedValue: string }).formatedValue, result: computeOutputs(m) };
    m.dispose();
    return out;
  };

  it("writes the rules and reads them back; a ready-made score's (with or) can't be edited", () => {
    expect(when).toBe("{q_score} >= 2 and {patientSex} = 'male'");
    expect(parseBandWhen(when)).toEqual([{ left: "q_score", op: "gte", value: 2 }, { left: "patientSex", op: "eq", value: "male" }]);
    expect(parseBandWhen(undefined)).toEqual([]);
    expect(parseBandWhen("{a} = 'x' or ({b} = 'y' and {c} = 'z')")).toBeUndefined();
    expect(formatBandWhen([{ left: "", op: "eq" }])).toBeUndefined();
  });

  it("offers the score itself, then what a points line can test", () => {
    const subjects = bandWhenSubjects(doc(), undefined, "q_score");
    expect(subjects[0]).toEqual({ id: "q_score", left: "q_score", label: "The score", group: "This score", type: "number" });
    expect(subjects.map((x) => x.left)).toEqual(expect.arrayContaining(["patientAge", "patientSex", "q_snore"]));
  });

  it("picks the band whatever the score's own limits, with its disclosure", () => {
    const yes2 = { q_snore: "o_yes", q_tired: "o_yes" };
    expect(run(man51, yes2).display).toBe("2 (High)");
    expect(run(man51, yes2).result.outputs.map((o) => o.output.id)).toEqual(["out_high"]);
    expect(run({ ...man51, sex: "female" }, yes2).display).toBe("2 (Low)");
    expect(run(man51, { q_snore: "o_yes", q_tired: "o_no" }).display).toBe("1 (Low)");
  });

  it("is Incomplete while what the band tests has no answer (the sex not known)", () => {
    expect(run({ ...man51, sex: "unknown" }, { q_snore: "o_yes", q_tired: "o_yes" }).display).toBe("Incomplete");
  });

  it("runs in test cases (and so the Preview)", () => {
    const tc: TestCase = { id: "t", name: "Male", viewer: "clinician", patient: man51, answers: { q_snore: "o_yes", q_tired: "o_yes" }, expected: ["Note (Airway): High risk of sleep apnoea"] };
    expect(runTestCase(doc(), tc).pass).toBe(true);
    expect(runTestCase(doc(), { ...tc, patient: { ...man51, sex: "female" } }).pass).toBe(false);
  });

  it("names a band's \"also when\" testing something deleted, at Publish", () => {
    const gone = formatBandWhen([{ left: "q_gone", op: "eq", value: "o_yes" }])!;
    expect(publishProblems([{ name: "Sleep", doc: doc(gone) }]).map((p) => p.message)).toEqual([
      "\"Sleep score\" band \"High\": it tests something that no longer exists (q_gone)",
    ]);
  });
});
