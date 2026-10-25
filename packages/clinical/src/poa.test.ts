import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import type { Answers } from "./respondent";
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
    // The old page switch is read as a page summary named cs_<page>, so its comments keep their key (S32).
    expect(heart.blocks).toEqual([{
      name: "cs_p_heart", title: "Heart and blood summary", comments: "On GTN spray.",
      notes: [
        { text: "Angina - unspecified", questionId: "q_angina", corrected: true },
        { text: "Anemia", questionId: "q_anaemia", corrected: false },
      ],
    }]);
    expect(heart.changes).toEqual([{ questionId: "q_angina", title: "Do you have angina?", patient: "No", clinician: "Yes" }]);
    expect(meds.captures).toEqual([{ questionId: "q_meds", kind: "medication", title: "Prescribed medications", rows: [{ name: "Omeprazole", dosage: "20mg", frequency: "Daily" }] }]);
    expect(bmi.captures[0].rows[0]).toMatchObject({ height: 165, weight: 74, bmi: 27.2 });
    // Pages with nothing to show are left out; a clinician-only answer isn't a correction.
    expect(poa[0].pages.map((p) => p.name)).toEqual(["p_heart", "p_meds", "p_bmi"]);
    expect(poa[1].pages[0].blocks).toMatchObject([{ title: "Angina follow-up Summary", notes: [{ text: "Stable angina" }] }]); // no summary: the page's own block
  });

  it("builds the patient's original answers without the clinician's", () => {
    const poa = poaSummary(sets, patient, "patient", jo);
    expect(poa.map((s) => s.name)).toEqual(["About you"]); // the patient said no angina
    const heart = poa[0].pages[0];
    expect(heart.blocks.map((b) => [b.title, b.notes.map((n) => n.text), b.comments])).toEqual([["Heart and blood summary", ["Anemia"], ""]]);
    expect(heart.changes).toEqual([]);
  });
});

// S28 (Rahul, 24 Oct): the Patient reported BMI uses the patient's own answers until the set is
// validated, then the validated ones; from a BMI question, or the library's height and weight.
describe("reportedBmi", () => {
  const sized: ChapterJson = {
    pages: [{ name: "p_size", title: "Height and weight", elements: [
      { type: "text", inputType: "number", name: "height", title: "How tall are you?" },
      { type: "text", inputType: "number", name: "weight", title: "What do you weigh now?" },
    ] }],
  };
  it("from a BMI question's height and weight", () => {
    expect(reportedBmi(sets, (id) => (patient as Record<string, Answers | undefined>)[id], jo)).toBe(27.2);
  });
  it("from the library's height and weight questions", () => {
    expect(reportedBmi([{ id: "s1", name: "About you", content: sized }], () => ({ height: 170, weight: 80 }), jo)).toBe(27.7);
  });
  it("the answers it's given: the patient's before validation, the validated ones after", () => {
    const set = [{ id: "s1", name: "About you", content: sized }];
    const patientsOwn = { s1: { height: 170, weight: 80 } };
    const validated = { s1: { height: 170, weight: 90 } };
    expect(reportedBmi(set, (id) => patientsOwn[id as "s1"], jo)).toBe(27.7);
    expect(reportedBmi(set, (id) => validated[id as "s1"], jo)).toBe(31.1);
  });
  it("nothing without both a height and a weight", () => {
    const set = [{ id: "s1", name: "About you", content: sized }];
    expect(reportedBmi(set, () => ({ height: 170 }), jo)).toBeUndefined();
    expect(reportedBmi(set, () => ({ height: 0, weight: 80 }), jo)).toBeUndefined();
    expect(reportedBmi(set, () => undefined, jo)).toBeUndefined();
  });

  // S29 (Rahul, 25 Oct): a hidden answer never feeds a clinical number, as with disclosures.
  describe("uses only answers that are shown", () => {
    const gated = (extra: Record<string, unknown> = {}): ChapterJson => ({
      pages: [{ name: "p_size", title: "Height and weight", elements: [
        yesNo("q_measured", "Do you know your height and weight?"),
        { type: "text", inputType: "number", name: "height", title: "How tall are you?", visibleIf: "{q_measured} = 'q_measured_y'", ...extra },
        { type: "text", inputType: "number", name: "weight", title: "What do you weigh now?" },
      ] }],
    });
    const held = { height: 170, weight: 80 };
    it("skips a height its condition hides, before validation and after", () => {
      const set = [{ id: "s1", name: "About you", content: gated() }];
      const patientsOwn = { s1: { ...held, q_measured: "q_measured_n" } };
      const validated = { s1: { ...held, weight: 90, q_measured: "q_measured_n" } };
      expect(reportedBmi(set, (id) => patientsOwn[id as "s1"], jo)).toBeUndefined();
      expect(reportedBmi(set, (id) => validated[id as "s1"], jo)).toBeUndefined();
      expect(reportedBmi(set, () => ({ ...held, q_measured: "q_measured_y" }), jo)).toBe(27.7); // shown again
    });
    it("skips a BMI question, page or group a condition hides", () => {
      const bmiHidden: ChapterJson = { pages: [{ name: "p", elements: [
        yesNo("q_ask", "Ask?"), { type: "bmi", name: "q_bmi", title: "BMI", visibleIf: "{q_ask} = 'q_ask_y'" },
      ] }] };
      const pageHidden: ChapterJson = { pages: [
        { name: "p1", elements: [yesNo("q_ask", "Ask?")] },
        { name: "p2", visibleIf: "{q_ask} = 'q_ask_y'", elements: [{ type: "text", inputType: "number", name: "height" }, { type: "text", inputType: "number", name: "weight" }] },
      ] };
      const groupHidden: ChapterJson = { pages: [{ name: "p", elements: [
        yesNo("q_ask", "Ask?"),
        { type: "panel", name: "g_size", visibleIf: "{q_ask} = 'q_ask_y'", elements: [{ type: "text", inputType: "number", name: "height" }, { type: "text", inputType: "number", name: "weight" }] },
      ] }] };
      const one = (content: ChapterJson) => [{ id: "s1", name: "S", content }];
      expect(reportedBmi(one(bmiHidden), () => ({ q_ask: "q_ask_n", q_bmi: { height: 170, weight: 80 } }), jo)).toBeUndefined();
      expect(reportedBmi(one(pageHidden), () => ({ q_ask: "q_ask_n", ...held }), jo)).toBeUndefined();
      expect(reportedBmi(one(groupHidden), () => ({ q_ask: "q_ask_n", ...held }), jo)).toBeUndefined();
      expect(reportedBmi(one(groupHidden), () => ({ q_ask: "q_ask_y", ...held }), jo)).toBe(27.7);
    });
    it("skips a Question Set its condition hides, and goes on to the next one shown", () => {
      const sized: ChapterJson = { pages: [{ name: "p", elements: [
        { type: "text", inputType: "number", name: "height" }, { type: "text", inputType: "number", name: "weight" },
      ] }] };
      const asked = [
        { id: "s0", name: "First", content: { pages: [{ name: "p0", elements: [yesNo("q_ask", "Ask?")] }] } },
        { id: "s1", name: "Hidden", content: { ...sized, chapterVisibleIf: "{q_ask} = 'q_ask_y'" } },
        { id: "s2", name: "Shown", content: sized },
      ];
      const answers: Record<string, Answers> = { s0: { q_ask: "q_ask_n" }, s1: { height: 170, weight: 80 }, s2: { height: 170, weight: 90 } };
      expect(reportedBmi(asked, (id) => answers[id], jo)).toBe(31.1);
    });
  });
});

// S28 (Rahul, 24 Oct): a set started fresh is in the summary, for its line, though nothing else shows.
describe("a set started fresh", () => {
  it("is kept with its pre-fill, on both tabs", () => {
    const fresh = { s1: { answers: {}, choice: "start_fresh" as const, choiceAt: "2026-04-01T09:00:00Z" } };
    const empty = { s1: { q_x: "q_x_n" } };
    expect(poaSummary(sets, empty, "patient", jo, undefined, fresh).map((s) => [s.id, s.pages.length, s.prefill?.choice])).toEqual([["s1", 0, "start_fresh"]]);
    expect(poaSummary(sets, empty, "clinician", jo, empty, fresh).map((s) => s.id)).toEqual(["s1"]);
    expect(poaSummary(sets, empty, "patient", jo)).toEqual([]);
  });
});

// S32 (SUM-01): one block per summary, with its own title, notes and comments; each note once,
// under the nearest summary; notes no summary covers in a block named after the page.
describe("several clinical summaries", () => {
  const summary = (name: string, title?: string) => ({ type: "clinicalsummary", name, clinicianOnly: true, ...(title ? { title } : {}) });
  const content: ChapterJson = {
    pages: [
      {
        name: "p_heart", title: "Heart",
        elements: [
          yesNo("q_angina", "Angina?", "Angina"),
          { type: "panel", name: "g_breath", title: "Breathing", elements: [yesNo("q_asthma", "Asthma?", "Asthma"), summary("cs_breath")] },
          summary("cs_page", "Heart overview"),
        ],
      },
      { name: "p_lungs", title: "Lungs", elements: [yesNo("q_cough", "Cough?", "Cough"), { type: "panel", name: "g_copd", title: "COPD", elements: [yesNo("q_copd", "COPD?", "COPD"), summary("cs_copd")] }] },
    ],
  };
  const one = [{ id: "s1", name: "About you", content }];
  const said = { s1: { q_angina: "q_angina_y", q_asthma: "q_asthma_y", q_cough: "q_cough_y", q_copd: "q_copd_y" } };
  const validated = { s1: { ...said.s1, cs_comments_breath: "Uses a blue inhaler", cs_comments_page: "Refer to cardiology" } };
  const blocks = (poa: ReturnType<typeof poaSummary>) =>
    poa[0].pages.map((p) => [p.title, p.blocks.map((b) => [b.title, b.notes.map((n) => n.text), b.comments])]);

  it("the validated summary: each summary's own notes and comment", () => {
    expect(blocks(poaSummary(one, validated, "clinician", jo, said))).toEqual([
      ["Heart", [["Breathing summary", ["Asthma"], "Uses a blue inhaler"], ["Heart overview", ["Angina"], "Refer to cardiology"]]],
      ["Lungs", [["Lungs Summary", ["Cough"], ""], ["COPD summary", ["COPD"], ""]]],
    ]);
  });

  it("the patient's answers: the same blocks, without comments", () => {
    expect(blocks(poaSummary(one, said, "patient", jo))).toEqual([
      ["Heart", [["Breathing summary", ["Asthma"], ""], ["Heart overview", ["Angina"], ""]]],
      ["Lungs", [["Lungs Summary", ["Cough"], ""], ["COPD summary", ["COPD"], ""]]],
    ]);
  });

  it("a summary with only a comment still shows; one with nothing doesn't", () => {
    const quiet = { s1: { q_angina: "q_angina_n", q_asthma: "q_asthma_n", cs_comments_breath: "Checked" } };
    expect(blocks(poaSummary(one, quiet, "clinician", jo, quiet))).toEqual([["Heart", [["Breathing summary", [], "Checked"]]]]);
  });
});
