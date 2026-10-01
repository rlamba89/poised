// Phase 1 and 2 features at runtime: calculations, scores and bands, grids, repeating groups,
// patient screens, skip rules, test cases and translations.
import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import type { ChapterJson } from "./doc";
import { computeOutputs } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { outputLines, runTestCase, setTestCases, testCasesOf, type TestCase } from "./testcases";
import { fromCsv, localesIn, missingCount, setTranslation, textEntries, toCsv } from "./translations";
import { forPatient, patientScreens } from "./viewer";

registerClinicalProperties();

const out = (id: string, note: string, extra = {}) => ({ id, codes: [], note: { text: note, category: "Respiratory" }, ...extra });

function chapter(): ChapterJson {
  return {
    pages: [
      {
        name: "p_one",
        title: "Sleep",
        elements: [
          { type: "radiogroup", name: "q_snore", title: "Do you snore?", choices: [{ value: "o_y", text: "Yes", score: 1 }, { value: "o_n", text: "No", score: 0 }] },
          { type: "checkbox", name: "q_tired", title: "Tired?", choices: [{ value: "o_day", text: "In the day", score: 1 }, { value: "o_drive", text: "When driving", score: 2 }] },
          {
            type: "expression", name: "q_stop", title: "STOP-BANG", expression: "score('q_snore', 'q_tired')", clinicianOnly: true,
            bands: [
              { id: "b_low", label: "Low", max: 1 },
              { id: "b_high", label: "High", clinicalOutputs: [out("out_h", "STOP-BANG {answer}: high risk", { flag: "red" })] },
            ],
          },
          { type: "text", inputType: "number", name: "q_h", title: "Height" },
          { type: "text", inputType: "number", name: "q_w", title: "Weight" },
          { type: "expression", name: "q_bmi", title: "BMI", expression: "bmi({q_h}, {q_w})" },
          {
            type: "matrix", name: "q_grid", title: "Symptoms", rows: [{ value: "r_c", text: "Cough" }], columns: [{ value: "o_0", text: "None" }, { value: "o_2", text: "Bad" }],
            cellOutputs: { r_c: { o_2: [out("out_c", "Bad cough")] } },
          },
        ],
      },
      {
        name: "p_two",
        title: "Operations",
        elements: [
          { type: "text", name: "q_intro", title: "Intro" },
          {
            type: "panel", name: "g_sec", title: "Your operations", patientPage: true,
            elements: [
              {
                type: "paneldynamic", name: "g_ops", title: "Operation", panelCount: 1,
                templateElements: [{ type: "radiogroup", name: "q_ga", title: "General?", choices: [{ value: "o_y", text: "Yes", clinicalOutputs: [out("out_ga", "Had a GA")] }] }],
              },
            ],
          },
          { type: "panel", name: "g_sec2", title: "Hidden heading", patientPage: true, showAsHeading: false, elements: [{ type: "text", name: "q_last", title: "Last question" }] },
          { type: "text", name: "q_clin", title: "Clinician note", clinicianOnly: true },
        ],
      },
    ],
    triggers: [{ type: "complete", expression: "{q_snore} = 'o_n'" }],
  };
}

describe("calculations, scores and bands (CAL-01/02/03)", () => {
  it("totals option scores, picks the band, and the band carries outputs", () => {
    const m = new Model(chapter());
    m.setValue("q_snore", "o_y");
    m.setValue("q_tired", ["o_day", "o_drive"]);
    expect(m.getValue("q_stop")).toBe(4);
    const r = computeOutputs(m).outputs.find((o) => o.questionId === "q_stop")!;
    expect(r).toMatchObject({ answerId: "b_high", answerLabel: "High", noteText: "STOP-BANG 4: high risk" });
    m.setValue("q_tired", []);
    expect(computeOutputs(m).outputs.some((o) => o.questionId === "q_stop")).toBe(false);
  });

  it("shows the band after the value", async () => {
    const { showBands } = await import("./questionTypes");
    const m = new Model(chapter());
    showBands(m);
    m.setValue("q_snore", "o_y");
    m.setValue("q_tired", ["o_drive"]);
    expect(m.getQuestionByName("q_stop").displayValue).toBe("3 (High)");
  });

  it("works out BMI", () => {
    const m = new Model(chapter());
    m.setValue("q_h", 180);
    m.setValue("q_w", 81);
    expect(m.getValue("q_bmi")).toBe(25);
  });
});

describe("grid disclosures (QT-10) and repeating groups (STR-04)", () => {
  it("produces outputs per grid cell and per repeating entry", () => {
    const m = new Model(chapter());
    m.setValue("q_grid", { r_c: "o_2" });
    m.setValue("g_ops", [{ q_ga: "o_y" }, { q_ga: "o_y" }]);
    const notes = computeOutputs(m).outputs.map((o) => `${o.answerLabel} → ${o.noteText}`);
    expect(notes).toEqual(["Cough: Bad → Bad cough", "Yes → Had a GA", "Yes → Had a GA"]);
  });
});

describe("patient screens (STR-03)", () => {
  it("puts each Section on its own screen, and hides a Section title when asked", () => {
    const pages = patientScreens(chapter()).pages!;
    expect(pages.map((p) => p.name)).toEqual(["p_one", "p_two", "p_two__1", "p_two__2", "p_two__3"]);
    expect(pages[2].elements!.map((e) => e.name)).toEqual(["g_sec"]);
    expect(pages[3].elements![0].title).toBeUndefined();
    const forPatients = forPatient(setTestCases(chapter(), [{ id: "t", name: "x", viewer: "patient", patient: { name: "A" }, answers: {}, expected: [] }]));
    expect(forPatients.pages!.map((p) => p.name)).toEqual(["p_one", "p_two", "p_two__1", "p_two__2"]);
    expect(forPatients.testCases).toBeUndefined();
    expect(JSON.stringify(forPatients)).not.toContain("q_stop");
  });
});

describe("skip rules (LOG-12)", () => {
  it("ends the Question Set when the condition is met", () => {
    const m = new Model(chapter());
    m.setValue("q_snore", "o_n");
    m.nextPage();
    expect(m.state).toBe("completed");
  });
});

describe("test cases (PRV-05/06)", () => {
  it("passes when the outputs match, and lists the differences when not", () => {
    const answers = { q_snore: "o_y", q_tired: ["o_drive"], q_grid: { r_c: "o_2" } };
    const m = new Model(chapter());
    m.data = answers;
    const expected = outputLines(computeOutputs(m));
    expect(expected).toEqual([
      "Flag red: STOP-BANG: High",
      "Note (Respiratory): Bad cough",
      "Note (Respiratory): STOP-BANG 3: high risk",
    ]);
    const tc: TestCase = { id: "t1", name: "High risk", viewer: "clinician", patient: { name: "Sam", age: 60 }, answers, expected };
    expect(runTestCase(chapter(), tc).pass).toBe(true);
    // A patient doesn't get the clinician-only score, so its outputs are missing.
    const asPatient = runTestCase(chapter(), { ...tc, viewer: "patient" });
    expect(asPatient.pass).toBe(false);
    expect(asPatient.missing).toEqual(["Flag red: STOP-BANG: High", "Note (Respiratory): STOP-BANG 3: high risk"]);
    expect(testCasesOf(setTestCases(chapter(), [tc]))).toEqual([tc]);
    expect(setTestCases(setTestCases(chapter(), [tc]), []).testCases).toBeUndefined();
  });
});

describe("translations (LNG-01/02/03)", () => {
  it("lists patient texts, sets translations, and round-trips CSV", () => {
    const doc = chapter();
    const entries = textEntries(doc);
    expect(entries.some((e) => e.english === "Clinician note")).toBe(false);
    expect(entries.find((e) => e.english === "Yes")?.field).toBe("Option");
    const snore = entries.find((e) => e.english === "Do you snore?")!;
    const de = setTranslation(doc, snore.path, "de", "Schnarchen Sie?");
    expect(de.pages![0].elements![0].title).toEqual({ default: "Do you snore?", de: "Schnarchen Sie?" });
    expect(localesIn(de)).toEqual(["de"]);
    expect(missingCount(de, "de")).toBe(entries.length - 1);
    const m = new Model(de);
    m.locale = "de";
    expect(m.getQuestionByName("q_snore").title).toBe("Schnarchen Sie?");

    const csv = toCsv(de, "de").replace("Do you snore?,Schnarchen Sie?", "Do you snore?,Schnarchst du?").replace(/Tired\?,$/m, "Tired?,Müde?");
    const back = fromCsv(de, "de", csv);
    expect(back.applied).toBe(2); // only the two filled-in cells
    expect(back.skipped).toBe(0);
    // A blank cell never wipes a translation added after the export.
    const later = setTranslation(back.doc, entries.find((e) => e.english === "Height")!.path, "de", "Größe");
    const again = fromCsv(later, "de", toCsv(de, "de"));
    expect(again.doc.pages![0].elements!.find((e) => e.name === "q_h")!.title).toEqual({ default: "Height", de: "Größe" });
    expect(back.doc.pages![0].elements![0].title).toEqual({ default: "Do you snore?", de: "Schnarchst du?" });
    expect(back.doc.pages![0].elements![1].title).toEqual({ default: "Tired?", de: "Müde?" });
    // Clearing a translation goes back to a plain string.
    expect(setTranslation(de, snore.path, "de", "").pages![0].elements![0].title).toBe("Do you snore?");
  });
});

describe("Question Set conditions (LOG-02)", () => {
  it("shows a set depending on earlier answers and the patient, and flags bad references", async () => {
    const { chapterConditionProblems, combineChapters, isChapterShown, setChapterCondition, testsOnlyThePatient } = await import("./questionSets");
    const first: ChapterJson = { pages: [{ name: "p_a", title: "Lungs", elements: [{ type: "radiogroup", name: "q_copd", choices: [{ value: "o_y", text: "Yes" }] }] }] };
    const second = setChapterCondition(chapter(), "{q_copd} = 'o_y' or {patientAge} >= 70");
    expect(isChapterShown(second, { q_copd: "o_y" })).toBe(true);
    expect(isChapterShown(second, { patientAge: 75 })).toBe(true);
    expect(isChapterShown(second, { patientAge: 40 })).toBe(false);
    expect(isChapterShown(chapter(), {})).toBe(true);
    const earlier = combineChapters([{ name: "Breathing", doc: first }]);
    expect(earlier.pages![0].title).toBe("Breathing › Lungs");
    expect(chapterConditionProblems(second, earlier)).toEqual([]);
    expect(chapterConditionProblems(setChapterCondition(second, "{q_snore} = 'o_y' or {q_gone} = 1"), earlier)).toEqual([
      "it tests one of its own questions",
      "it tests something that isn't in an earlier Question Set (q_gone)",
    ]);
    expect(testsOnlyThePatient("{patientAge} >= 70 and {viewer} = 'patient'")).toBe(true);
    expect(testsOnlyThePatient("{q_copd} = 'o_y'")).toBe(false);
    expect(setChapterCondition(second, undefined).chapterVisibleIf).toBeUndefined();
  });
});

describe("{answer} for dates (CLN-07)", () => {
  it("writes a month as MM/yyyy and a list of dates joined", async () => {
    const { formatAnswer } = await import("./outputs");
    const m = new Model({ elements: [
      { type: "text", inputType: "month", name: "q_m" },
      { type: "matrixdynamic", name: "q_l", dateList: true, columns: [{ name: "date", cellType: "text", inputType: "date" }] },
    ] });
    expect(formatAnswer(m.getQuestionByName("q_m"), "2025-03")).toBe("03/2025");
    expect(formatAnswer(m.getQuestionByName("q_l"), [{ date: "2020-01-31" }, { date: "2021-02-01" }])).toBe("31/01/2020, 01/02/2021");
  });

  it("produces a date list's own disclosures (matrices are not choice questions)", () => {
    const m = new Model({ elements: [
      { type: "matrixdynamic", name: "q_l", title: "Dates", dateList: true, columns: [{ name: "date", cellType: "text", inputType: "date" }],
        clinicalOutputs: [out("out_d", "Operations on {answer}")] },
      { type: "medication", name: "q_med", clinicalOutputs: [out("out_m", "Takes medication")] },
    ] });
    m.setValue("q_l", [{ date: "2020-01-31" }]);
    m.setValue("q_med", [{ name: "Aspirin" }]);
    expect(computeOutputs(m).outputs.map((o) => o.noteText)).toEqual(["Operations on 31/01/2020", "Takes medication"]);
  });
});
