import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { Model } from "survey-core";
import { modelFor } from "./testcases";
import { registerClinicalProperties } from "./properties";
import { ageFrom, dateIn, describeAnswer, isSetComplete, isSetReadyToSend, patientVariables, sameAnswer, shownSets, shownToPatient, loadForPatient, clearedAnswers, type Answers } from "./respondent";

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

  it("works out the age on a day in the hospital's time zone, as Go does (S44a)", () => {
    // 23:30 UTC on 10 Dec: already 11 Dec (the birthday) in Auckland, still 10 Dec in Los Angeles.
    const at = new Date(Date.UTC(2026, 11, 10, 23, 30));
    expect(ageFrom("1974-12-11", at, "Pacific/Auckland")).toBe(52);
    expect(ageFrom("1974-12-11", at, "America/Los_Angeles")).toBe(51);
    expect(ageFrom("1974-12-11", at, "UTC")).toBe(51);
    expect(patientVariables({ firstName: "Jo", lastName: "Bloggs", dateOfBirth: "1974-12-11", sex: "male" }, at, "Pacific/Auckland").age).toBe(52);
    // Verification 1, finding 1: a browser whose locale has another calendar still gets the Gregorian date.
    for (const locale of ["th-TH", "fa-IR", "ar-SA", "ja-JP-u-ca-japanese"]) {
      expect(dateIn(at, "Pacific/Auckland", locale)).toEqual({ y: 2026, m: 12, d: 11 });
    }
    // A zone the browser doesn't know: its own date, as before.
    expect(ageFrom("1974-12-11", new Date(2026, 9, 1), "Not/AZone")).toBe(51);
  });

  it("is complete when every required visible question is answered", () => {
    expect(isSetComplete(smoking, {}, "patient", pat)).toBe(false);
    expect(isSetComplete(smoking, { q_smoke: "o_n" }, "patient", pat)).toBe(true);
    // A required clinician-only question doesn't hold the patient up, but does hold up the clinician.
    expect(isSetComplete(clinicianNote, {}, "patient", pat)).toBe(true);
    expect(isSetComplete(clinicianNote, {}, "clinician", pat)).toBe(false);
  });

  // S02a verify.md, verification 2 finding 1: the server had a complete set, the patient changed an
  // answer and the save failed. The set still counted as complete, so the HQ was sent without the change.
  it("is ready to send only when the server's answers are complete and nothing on screen is unsaved", () => {
    const saved = { q_smoke: "o_n" };
    expect(isSetReadyToSend(smoking, saved, undefined, pat)).toBe(true); // nothing typed since loading
    expect(isSetReadyToSend(smoking, saved, { q_smoke: "o_n" }, pat)).toBe(true); // what's on screen is saved
    expect(isSetReadyToSend(smoking, saved, { q_smoke: "o_y" }, pat)).toBe(false); // a change that didn't save
    expect(isSetReadyToSend(smoking, {}, { q_smoke: "o_n" }, pat)).toBe(false); // complete on screen only
    expect(isSetReadyToSend(smoking, undefined, undefined, pat)).toBe(false); // nothing saved yet
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

// S44c (Rahul, 28 Oct): what the patient's form showed for the answers they sent, so a question it
// didn't show gets no "Patient answered" line on Validate or the POA Summary.
describe("shownToPatient", () => {
  const yn = (name: string, extra: Record<string, unknown> = {}) => ({ type: "radiogroup", name, choices: [{ value: "y", text: "Yes" }, { value: "n", text: "No" }], ...extra });
  const one: ChapterJson = { pages: [
    { name: "p1", elements: [yn("q_b"), yn("q_a", { visibleIf: "{q_b} = 'y'" }), yn("q_c", { visibleIf: "{q_a} = 'y'" }), { type: "comment", name: "q_nurse", clinicianOnly: true }] },
    { name: "p2", visibleIf: "{q_b} = 'y'", elements: [yn("q_p2")] },
  ] };
  const two: ChapterJson = { chapterVisibleIf: "{q_b} = 'y'", pages: [{ name: "p1", elements: [yn("q_two")] }] };
  const nurse: ChapterJson = { pages: [{ name: "p1", elements: [yn("q_n")] }] };
  const hq = [
    { id: "s1", name: "One", content: one, audience: "patient" },
    { id: "s2", name: "Two", content: two, audience: "patient" },
    { id: "s3", name: "Nurse", content: nurse, audience: "clinician" },
  ];
  const p = { name: "Sam", age: 40, sex: "female" };
  const names = (m: Map<string, Set<string>>) => Object.fromEntries([...m].map(([k, v]) => [k, [...v]]));

  it("B No: only B; the follow-ups, the hidden page and the hidden set aren't shown; no clinician set", () => {
    expect(names(shownToPatient(hq, { s1: { q_b: "n" } }, p))).toEqual({ s1: ["q_b"] });
  });
  it("B Yes, A Yes: the chain, the page and the set are shown (answered or not)", () => {
    expect(names(shownToPatient(hq, { s1: { q_b: "y", q_a: "y" } }, p))).toEqual({ s1: ["q_b", "q_a", "q_c", "q_p2"], s2: ["q_two"] });
  });
});
// S44c (Rahul, 28 Oct, answer 1): the patient's form clears an answer once a condition hides its
// question (or its page, group or whole Question Set), so what's on screen is what's saved.
// Verification 2, F3: a change to a typed answer (text or number) clears at the autosave, after the
// pause in typing, so a number edited 20 → 2 → 25 doesn't wipe the follow-up it shows; any other
// change (a choice) clears at once, so a chain never shows a follow-up through a hidden answer.
describe("clearing hidden answers in the patient's form", () => {
  const yn = (name: string, extra: Record<string, unknown> = {}) => ({ type: "radiogroup", name, choices: [{ value: "y", text: "Yes" }, { value: "n", text: "No" }], ...extra });
  const chain: ChapterJson = { pages: [
    { name: "p1", elements: [yn("q_b"), yn("q_a", { visibleIf: "{q_b} = 'y'" }), { type: "text", name: "q_c", isRequired: true, visibleIf: "{q_a} = 'y'" }] },
    { name: "p2", visibleIf: "{q_b} = 'y'", elements: [{ type: "panel", name: "g", elements: [yn("q_g")] }] },
  ] };
  const p = { name: "Sam", age: 40, sex: "female" };
  const form = (doc: ChapterJson, data: Answers) => {
    const m = modelFor(doc, "patient", p);
    const clearing = loadForPatient(m, data);
    return { m, clearing };
  };

  it("the chain: B Yes → A Yes → C answered → B No clears A and C (and the hidden page's group) at once, each as a change", () => {
    const { m } = form(chain, {});
    m.setValue("q_b", "y");
    m.setValue("q_a", "y");
    m.setValue("q_c", "loud");
    m.setValue("q_g", "y");
    const changed: string[] = [];
    m.onValueChanged.add((_, o) => changed.push(o.name));
    m.setValue("q_b", "n");
    expect(m.data).toEqual({ q_b: "n" });
    expect(changed).toEqual(expect.arrayContaining(["q_a", "q_c", "q_g"]));
    // B Yes, A Yes, then B No before anything is typed in C: C (required) isn't left showing.
    m.setValue("q_b", "y");
    expect(m.getQuestionByName("q_a").isVisible && m.getQuestionByName("q_a").isEmpty()).toBe(true);
    m.setValue("q_a", "y");
    m.setValue("q_b", "n");
    expect(m.getQuestionByName("q_c").isVisible).toBe(false);
    expect(m.validate(false)).toBe(true);
    m.dispose();
  });

  it("a typed number edited 20 → 2 → 25 before the save keeps the follow-up; left at 2, the save clears it", () => {
    const units: ChapterJson = { pages: [{ name: "p1", elements: [
      { type: "text", inputType: "number", name: "q_units" }, { type: "comment", name: "q_why", visibleIf: "{q_units} > 14" },
    ] }] };
    const { m, clearing } = form(units, { q_units: 20, q_why: "Wine with dinner" });
    m.setValue("q_units", 2); // Backspace: hidden for an instant
    expect(m.data).toEqual({ q_units: 2, q_why: "Wine with dinner" });
    m.setValue("q_units", 25);
    clearing.atSave();
    expect(m.data).toEqual({ q_units: 25, q_why: "Wine with dinner" });
    m.setValue("q_units", 2);
    clearing.atSave();
    expect(m.data).toEqual({ q_units: 2 });
    m.dispose();
  });

  // Verification 3, F4: only a choice clears at once. A BMI's height or weight (a composite whose
  // boxes are typed) and a Calculation a typed number feeds wait for the save too.
  it("a BMI weight edited 120 → 12 → 125 keeps a BMI follow-up; left at 12, the save clears it", () => {
    registerClinicalProperties();
    const bmi: ChapterJson = { pages: [{ name: "p1", elements: [
      { type: "bmi", name: "q_bmi" }, { type: "comment", name: "q_why", visibleIf: "{q_bmi.bmi} > 35" },
    ] }] };
    const { m, clearing } = form(bmi, { q_bmi: { height: 170, weight: 120 }, q_why: "Diet plan" });
    expect(m.getQuestionByName("q_why").isVisible).toBe(true);
    m.setValue("q_bmi", { height: 170, weight: 12 });
    expect(m.data.q_why).toBe("Diet plan");
    m.setValue("q_bmi", { height: 170, weight: 125 });
    clearing.atSave();
    expect(m.data.q_why).toBe("Diet plan");
    m.setValue("q_bmi", { height: 170, weight: 12 });
    clearing.atSave();
    expect(m.data.q_why).toBeUndefined();
    m.dispose();
  });

  it("a number feeding a Calculation, edited 42 → 4 → 45, keeps its follow-up", () => {
    const neck: ChapterJson = { pages: [{ name: "p1", elements: [
      { type: "text", inputType: "number", name: "q_neck" },
      { type: "expression", name: "q_big", expression: "iif({q_neck} > 40, 1, 0)" },
      { type: "comment", name: "q_more", visibleIf: "{q_neck} > 40" },
    ] }] };
    const { m, clearing } = form(neck, { q_neck: 42, q_more: "Snores" });
    m.setValue("q_neck", 4);
    m.setValue("q_neck", 45);
    clearing.atSave();
    expect(m.data.q_more).toBe("Snores");
    m.dispose();
  });

  it("a typed change that shows a question, answered and hidden again with no pause, is cleared at the save", () => {
    const units: ChapterJson = { pages: [{ name: "p1", elements: [
      { type: "text", inputType: "number", name: "q_units" }, { type: "comment", name: "q_why", visibleIf: "{q_units} > 14" },
    ] }] };
    const { m, clearing } = form(units, { q_units: 2 });
    m.setValue("q_units", 20);
    m.setValue("q_why", "Wine");
    m.setValue("q_units", 2);
    clearing.atSave();
    expect(m.data).toEqual({ q_units: 2 });
    m.dispose();
  });

  // Verification 1, F1: with an "isn't" condition, a question shown while the set is empty is hidden
  // once its answers load. Loading mustn't clear it (no change, nothing saved): only a later hide or Finish.
  it("loading answers clears nothing, even under an \"isn't\" condition", () => {
    const not: ChapterJson = { pages: [{ name: "p1", elements: [yn("q_b"), { type: "text", name: "q_x", visibleIf: "{q_b} <> 'n'" }, yn("q_y")] }] };
    const m = modelFor(not, "patient", p);
    const changed: string[] = [];
    m.onValueChanged.add((_, o) => changed.push(o.name));
    const clearing = loadForPatient(m, { q_b: "n", q_x: "pre-filled" });
    m.setValue("q_y", "y"); // another change, then its save
    clearing.atSave();
    expect(m.data).toEqual({ q_b: "n", q_x: "pre-filled", q_y: "y" });
    expect(changed).toEqual(["q_y"]);
    m.doComplete();
    expect(m.data).toEqual({ q_b: "n", q_y: "y" });
    m.dispose();
  });

  it("an answer hidden when the form opens (e.g. pre-filled) stays until the set is finished", () => {
    const { m, clearing } = form(chain, { q_b: "n", q_a: "y" });
    clearing.atSave();
    expect(m.data).toEqual({ q_b: "n", q_a: "y" });
    m.doComplete();
    expect(m.data).toEqual({ q_b: "n" });
    m.dispose();
  });
});

describe("clearedAnswers: other Question Sets a change hides", () => {
  const yn = (name: string, extra: Record<string, unknown> = {}) => ({ type: "radiogroup", name, choices: [{ value: "y", text: "Yes" }, { value: "n", text: "No" }], ...extra });
  const s1: ChapterJson = { pages: [{ name: "p1", elements: [yn("q_snore")] }] };
  const s2: ChapterJson = { chapterVisibleIf: "{q_snore} = 'y'", pages: [{ name: "p1", elements: [yn("q_loud"), yn("q_wake", { visibleIf: "{q_loud} = 'y'" })] }] };
  const s3: ChapterJson = { chapterVisibleIf: "{q_wake} = 'y'", pages: [{ name: "p1", elements: [yn("q_tired")] }] };
  const s4: ChapterJson = { pages: [{ name: "p1", elements: [yn("q_other")] }] };
  const hq = [{ id: "s1", content: s1 }, { id: "s2", content: s2 }, { id: "s3", content: s3 }, { id: "s4", content: s4 }];
  const p = { name: "Sam", age: 40, sex: "female" };
  const before = { s1: { q_snore: "y" }, s2: { q_loud: "y", q_wake: "y" }, s3: { q_tired: "y" }, s4: { q_other: "n" } };

  it("Snoring back to No: the follow-up set and the set it showed are cleared; others untouched", () => {
    const after = { ...before, s1: { q_snore: "n" } };
    expect(clearedAnswers(hq, before, after, p, "s1")).toEqual({ s2: {}, s3: {} });
  });
  it("nothing hidden: nothing to clear", () => {
    expect(clearedAnswers(hq, before, { ...before, s4: { q_other: "y" } }, p, "s4")).toEqual({});
  });
  it("only what this change hid: an answer already hidden before (e.g. pre-filled) is left for the send", () => {
    const hiddenAlready = { ...before, s1: { q_snore: "n" }, s3: {} };
    expect(clearedAnswers(hq, hiddenAlready, { ...hiddenAlready, s4: { q_other: "y" } }, p, "s4")).toEqual({});
  });
});
