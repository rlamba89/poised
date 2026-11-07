// Assessments (S47): STOP-Bang on an episode, pre-filled from the HQ, the observations and the
// profile, each item with its source; the clinician completes and confirms it.
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import goStopBang from "../../../apps/api/internal/assessment/stopbang.json";
import {
  ASSESSMENTS, assessmentAnswers, assessmentContent, assessmentItems, assessmentResult, countingHqAnswers, filledCount, STOP_BANG_BMI,
  withDraft, type PrefillFrom,
} from "./assessments";
import type { ChapterJson } from "./doc";
import { APFEL_KEYS, apfelGroup, DASI_KEYS, dasiGroup, insertReadyScore, PRISMA7_KEYS, prisma7Group, STOP_BANG_KEYS as K, stopBangGroup } from "./ready";

const content = assessmentContent("stopbang")!;

/** An HQ asking S, T, O and P (a trust's own questions with STOP-Bang's keys), and a BMI question. */
const hqSet = (extra: Partial<Record<string, unknown>> = {}): ChapterJson => ({
  pages: [{
    name: "p_1", title: "Sleep", elements: [
      { type: "radiogroup", name: K.snoring, title: "Do you snore loudly?", choices: [{ value: "o_yes", text: "Yes" }, { value: "o_no", text: "No" }] },
      { type: "radiogroup", name: K.tired, title: "Tired?", choices: [{ value: "o_yes", text: "Yes" }, { value: "o_no", text: "No" }] },
      { type: "radiogroup", name: K.observed, title: "Observed?", choices: [{ value: "o_yes", text: "Yes" }, { value: "o_no", text: "No" }], ...extra },
      { type: "radiogroup", name: K.pressure, title: "Blood pressure?", choices: [{ value: "o_yes", text: "Yes" }, { value: "o_no", text: "No" }] },
      { type: "bmi", name: "q_bmi", title: "Height and weight" },
    ],
  }],
});
const STOP = { [K.snoring]: "o_yes", [K.tired]: "o_yes", [K.observed]: "o_no", [K.pressure]: "o_yes" };
/** Height 170 cm; the weight that gives the BMI wanted. */
const bmiAnswer = (bmi: number) => ({ height: 170, weight: Math.round(bmi * 1.7 * 1.7 * 10) / 10 });

const from = (o: Partial<PrefillFrom> & { answers?: Record<string, unknown>; set?: ChapterJson } = {}): PrefillFrom => ({
  hq: { sets: [{ id: "s1", name: "Sleep", content: o.set ?? hqSet() }], answers: { s1: { ...STOP, q_bmi: bmiAnswer(30), ...o.answers } }, patient: { name: "Pat", age: 62, sex: "male" } },
  patient: o.patient ?? { name: "Pat", age: 62, sex: "male" },
  hospitalBmi: o.hospitalBmi,
  obsPrecedence: o.obsPrecedence ?? true,
});
const item = (items: ReturnType<typeof assessmentItems>, key: string) => items.find((i) => i.key === key)!;

describe("the STOP-Bang assessment's definition", () => {
  it("is offered, by its id and name", () => {
    expect(ASSESSMENTS[0]).toEqual({ id: "stopbang", name: "STOP-Bang" });
    expect(assessmentContent("nope")).toBeUndefined();
  });
  it("is the ready-made score's Group, word for word, with a BMI the clinician can see filled", () => {
    const els = content.pages![0].elements!;
    expect(els.find((e) => e.name === K.group)).toEqual(stopBangGroup(STOP_BANG_BMI));
    expect(els.find((e) => e.name === STOP_BANG_BMI)).toMatchObject({ type: "text", inputType: "number" });
  });
  it("is what Go stores (apps/api/internal/assessment/stopbang.json): regenerate it with `npm run assessments -w packages/clinical`", () => {
    expect(goStopBang).toEqual(JSON.parse(JSON.stringify(content)));
  });
});

describe("pre-filling STOP-Bang", () => {
  it("fills 7 of 8 from an HQ that asked S, T, O and P, a hospital BMI and the profile; the neck is missing", () => {
    const items = assessmentItems(content, from({ hospitalBmi: 36.4 }));
    expect(items.map((i) => i.key)).toEqual([K.snoring, K.tired, K.observed, K.pressure, STOP_BANG_BMI, "patientAge", K.neck, "patientSex"]);
    expect(items.map((i) => i.source ?? null)).toEqual(["hq", "hq", "hq", "hq", "observation", "profile", null, "profile"]);
    expect(filledCount(items)).toBe(7);
    expect(item(items, K.snoring).value).toBe("o_yes");
    expect(item(items, K.observed).value).toBe("o_no");
    expect(item(items, STOP_BANG_BMI).value).toBe(36.4);
    expect(item(items, "patientAge").value).toBe(62);
    expect(item(items, "patientSex").value).toBe("male");
    expect(item(items, K.neck).value).toBeUndefined();
  });

  it("an HQ question not answered is missing", () => {
    const items = assessmentItems(content, from({ answers: { [K.tired]: undefined } }));
    expect([item(items, K.tired).value, item(items, K.tired).source]).toEqual([undefined, undefined]);
  });

  it("a question a condition hides counts as unanswered, though its answer is held", () => {
    const set = hqSet({ visibleIf: `{${K.snoring}} = 'o_no'` });
    const items = assessmentItems(content, from({ set }));
    expect([item(items, K.observed).value, item(items, K.observed).source]).toEqual([undefined, undefined]);
  });

  it("a value that isn't one of the assessment's options is missing", () => {
    const items = assessmentItems(content, from({ answers: { [K.snoring]: "o_sometimes" } }));
    expect(item(items, K.snoring).source).toBeUndefined();
  });

  it("an HQ with the ready-made STOP-Bang fills every item, the neck too", () => {
    const set = insertReadyScore({ pages: [{ name: "p_1", title: "Sleep", elements: [{ type: "bmi", name: "q_bmi", title: "BMI" }] }] }, { page: "p_1" }, "stopbang", "q_bmi.bmi").doc;
    const items = assessmentItems(content, from({ set, answers: { [K.neck]: "o_no", q_bmi: bmiAnswer(31) } }));
    expect(filledCount(items)).toBe(8);
    expect(item(items, K.neck)).toMatchObject({ value: "o_no", source: "hq" });
    expect(item(items, STOP_BANG_BMI)).toMatchObject({ value: 31, source: "hq" });
  });

  describe("which BMI (obs_precedence)", () => {
    it("on: the hospital BMI when there is one", () => {
      expect(item(assessmentItems(content, from({ hospitalBmi: 36.4, obsPrecedence: true })), STOP_BANG_BMI)).toMatchObject({ value: 36.4, source: "observation" });
    });
    it("off: the patient's own", () => {
      expect(item(assessmentItems(content, from({ hospitalBmi: 36.4, obsPrecedence: false })), STOP_BANG_BMI)).toMatchObject({ value: 30, source: "hq" });
    });
    it("on, without a hospital BMI: the patient's", () => {
      expect(item(assessmentItems(content, from({ obsPrecedence: true })), STOP_BANG_BMI)).toMatchObject({ value: 30, source: "hq" });
    });
    it("off, without the patient's: the hospital's", () => {
      expect(item(assessmentItems(content, from({ hospitalBmi: 36.4, obsPrecedence: false, answers: { q_bmi: undefined } })), STOP_BANG_BMI))
        .toMatchObject({ value: 36.4, source: "observation" });
    });
    it("neither: missing", () => {
      expect(item(assessmentItems(content, from({ answers: { q_bmi: undefined } })), STOP_BANG_BMI).source).toBeUndefined();
    });
  });

  describe("the profile", () => {
    it("no age: missing", () => {
      expect(item(assessmentItems(content, from({ patient: { name: "Pat", sex: "male" } })), "patientAge").source).toBeUndefined();
    });
    it("sex Other or Not known: missing (it can't decide the male point, as in the HQ)", () => {
      for (const sex of ["other", "unknown"]) {
        expect(item(assessmentItems(content, from({ patient: { name: "Pat", age: 40, sex } })), "patientSex").source).toBeUndefined();
      }
      expect(item(assessmentItems(content, from({ patient: { name: "Pat", age: 40, sex: "female" } })), "patientSex")).toMatchObject({ value: "female", source: "profile" });
    });
  });

  describe("the clinician's own answers", () => {
    it("are kept over the pre-fill until confirmed; the rest is filled afresh", () => {
      const saved = { data: { [K.neck]: "o_yes", [K.snoring]: "o_no", [K.tired]: "o_no" }, sources: { [K.neck]: "clinician", [K.snoring]: "clinician", [K.tired]: "hq" } as const, confirmed: false };
      const items = assessmentItems(content, from({ hospitalBmi: 36.4 }), saved);
      expect(item(items, K.neck)).toMatchObject({ value: "o_yes", source: "clinician" });
      expect(item(items, K.snoring)).toMatchObject({ value: "o_no", source: "clinician" });
      expect(item(items, K.tired)).toMatchObject({ value: "o_yes", source: "hq" }); // the HQ as it is now
      expect(filledCount(items)).toBe(8);
    });
    it("once confirmed, are what was confirmed, with their sources", () => {
      const data = { ...STOP, [K.neck]: "o_no", [STOP_BANG_BMI]: 36.4 };
      const sources = { [K.snoring]: "hq", [K.tired]: "hq", [K.observed]: "hq", [K.pressure]: "hq", [K.neck]: "clinician", [STOP_BANG_BMI]: "observation" } as const;
      const items = assessmentItems(content, from({ obsPrecedence: false, answers: { [K.snoring]: "o_no" } }), { data, sources, confirmed: true });
      expect(item(items, K.snoring)).toMatchObject({ value: "o_yes", source: "hq" });
      expect(item(items, STOP_BANG_BMI)).toMatchObject({ value: 36.4, source: "observation" });
    });
  });
});

describe("the age and sex, frozen at Confirm (S47 answers 7)", () => {
  const confirmed = {
    data: { ...STOP, [K.neck]: "o_no", [STOP_BANG_BMI]: 36.4, patientAge: 62, patientSex: "male" },
    sources: {
      [K.snoring]: "hq", [K.tired]: "hq", [K.observed]: "hq", [K.pressure]: "hq", [K.neck]: "clinician", [STOP_BANG_BMI]: "observation",
      patientAge: "profile", patientSex: "profile",
    } as const,
    confirmed: true,
  };
  // The record corrected since: a later date of birth (age 40) and the sex.
  const now = from({ patient: { name: "Pat", age: 40, sex: "female" } });
  it("a confirmed assessment shows the age and sex it was confirmed with", () => {
    const items = assessmentItems(content, now, confirmed);
    expect(item(items, "patientAge")).toMatchObject({ value: 62, source: "profile", yes: true });
    expect(item(items, "patientSex")).toMatchObject({ value: "male", source: "profile", yes: true });
  });
  it("its score doesn't move with a changed date of birth or sex", () => {
    const at = { name: "Pat", age: 40, sex: "female" };
    expect(assessmentResult(content, confirmed.data, at).text).toBe("6 (High risk)"); // S, T, P, B, over 50, male
    expect(assessmentResult(content, { ...confirmed.data, patientAge: undefined, patientSex: undefined }, at).text).toBe("4 (High risk)");
  });
  it("one confirmed before they were stored uses the record as given (the day of Confirm's)", () => {
    const { patientAge: _a, patientSex: _s, ...old } = confirmed.data;
    const items = assessmentItems(content, now, { ...confirmed, data: old });
    expect(item(items, "patientAge")).toMatchObject({ value: 40, source: "profile" });
    expect(assessmentResult(content, old, { name: "Pat", age: 62, sex: "male" }).text).toBe("6 (High risk)");
  });
  it("one not confirmed follows the record", () => {
    const items = assessmentItems(content, now, { ...confirmed, confirmed: false });
    expect(item(items, "patientAge")).toMatchObject({ value: 40, source: "profile" });
    expect(item(items, "patientSex")).toMatchObject({ value: "female", source: "profile" });
  });
});

describe("the HQ's answers that count", () => {
  // S47 answers 5 (Rahul, 6 Nov): the clinician's copy as soon as they've started it, as the rest of
  // the episode page (currentAnswers in apps/web's hq.ts).
  it("are the clinician's copy once they've started it, validated or not, else the patient's", () => {
    const sets = [{ id: "validated" }, { id: "started" }, { id: "notStarted" }];
    const answers = countingHqAnswers(sets, { validated: { x: 1 }, started: { x: 2 }, notStarted: { x: 3 } }, {
      validated: { data: { x: 10 }, validatedAt: "2026-11-05T10:00:00Z" }, started: { data: { x: 20 }, validatedAt: null },
    });
    expect(answers).toEqual({ validated: { x: 10 }, started: { x: 20 }, notStarted: { x: 3 } });
  });
});

describe("the score and band", () => {
  const data = (yes: string[], neck: "o_yes" | "o_no", bmi: number) => ({
    [K.snoring]: yes.includes("s") ? "o_yes" : "o_no", [K.tired]: yes.includes("t") ? "o_yes" : "o_no",
    [K.observed]: yes.includes("o") ? "o_yes" : "o_no", [K.pressure]: yes.includes("p") ? "o_yes" : "o_no", [K.neck]: neck, [STOP_BANG_BMI]: bmi,
  });
  it("as published: 5 is High risk", () => {
    expect(assessmentResult(content, data(["s", "t", "p"], "o_no", 36.4), { name: "Pat", age: 62, sex: "female" })).toEqual({ score: 5, band: "High risk", text: "5 (High risk)" });
  });
  it("Low risk", () => {
    expect(assessmentResult(content, data(["s"], "o_no", 25), { name: "Pat", age: 40, sex: "female" })).toEqual({ score: 1, band: "Low risk", text: "1 (Low risk)" });
  });
  it("2 STOP and male is High risk whatever the total", () => {
    expect(assessmentResult(content, data(["s", "t"], "o_no", 25), { name: "Pat", age: 40, sex: "male" })?.text).toBe("3 (High risk)");
  });
  it("a missing item is Incomplete", () => {
    const d: Record<string, unknown> = data(["s"], "o_no", 25);
    delete d[K.neck];
    expect(assessmentResult(content, d, { name: "Pat", age: 40, sex: "male" })).toEqual({ text: "Incomplete" });
    expect(assessmentResult(content, data(["s"], "o_no", 25), { name: "Pat", age: 40, sex: "unknown" })).toEqual({ text: "Incomplete" });
  });
});

describe("each item's point, and the clinician's changes on screen", () => {
  it("says which items score (a Yes, a BMI over 35, over 50, male)", () => {
    const items = assessmentItems(content, from({ hospitalBmi: 35, patient: { name: "Pat", age: 51, sex: "female" } }));
    expect(items.map((i) => i.yes)).toEqual([true, true, false, true, false, true, undefined, false]);
  });
  it("a pre-filled BMI outside the question's limits isn't used: the other BMI, else missing (verification 1)", () => {
    expect(item(assessmentItems(content, from({ hospitalBmi: 151 })), STOP_BANG_BMI)).toMatchObject({ value: 30, source: "hq" });
    expect(item(assessmentItems(content, from({ hospitalBmi: 151, answers: { q_bmi: bmiAnswer(4) } })), STOP_BANG_BMI).source).toBeUndefined();
  });
  it("the BMI carries its limits, for the box to check (report 1)", () => {
    expect(item(assessmentItems(content, from()), STOP_BANG_BMI)).toMatchObject({ min: 5, max: 150 });
    expect(item(assessmentItems(content, from()), K.neck).min).toBeUndefined();
  });
  it("a change back to the value shown keeps where it came from (report 1)", () => {
    const items = withDraft(content, assessmentItems(content, from({ hospitalBmi: 30 })), { [K.snoring]: "o_yes", [STOP_BANG_BMI]: 30 });
    expect(item(items, K.snoring)).toMatchObject({ value: "o_yes", source: "hq" });
    expect(item(items, STOP_BANG_BMI)).toMatchObject({ value: 30, source: "observation" });
  });
  it("a change is the clinician's; the profile can't be changed", () => {
    const items = withDraft(content, assessmentItems(content, from({ hospitalBmi: 30 })), { [K.neck]: "o_yes", [K.snoring]: "o_no", [STOP_BANG_BMI]: 36, patientAge: 20 });
    expect(item(items, K.neck)).toMatchObject({ value: "o_yes", source: "clinician", yes: true });
    expect(item(items, K.snoring)).toMatchObject({ value: "o_no", source: "clinician", yes: false });
    expect(item(items, STOP_BANG_BMI)).toMatchObject({ value: 36, source: "clinician", yes: true });
    expect(item(items, "patientAge")).toMatchObject({ value: 62, source: "profile" });
  });
  it("stores each answer with its source, the age and sex too (S47 answers 7: frozen at Confirm)", () => {
    const items = withDraft(content, assessmentItems(content, from({ hospitalBmi: 36.4 })), { [K.neck]: "o_no" });
    expect(assessmentAnswers(items)).toEqual({
      data: { ...STOP, [STOP_BANG_BMI]: 36.4, patientAge: 62, [K.neck]: "o_no", patientSex: "male" },
      sources: {
        [K.snoring]: "hq", [K.tired]: "hq", [K.observed]: "hq", [K.pressure]: "hq", [STOP_BANG_BMI]: "observation", patientAge: "profile",
        [K.neck]: "clinician", patientSex: "profile",
      },
    });
  });
  it("the items' points add up to the published score, for every patient and answer", () => {
    for (const [age, sex, bmi] of [[62, "male", 36.4], [40, "female", 25], [50, "male", 35], [51, "female", 35.1]] as const) {
      for (const neck of ["o_yes", "o_no"]) {
        const items = withDraft(content, assessmentItems(content, from({ hospitalBmi: bmi, patient: { name: "Pat", age, sex } })), { [K.neck]: neck });
        const points = items.filter((i) => i.yes).length;
        expect(assessmentResult(content, assessmentAnswers(items).data, { name: "Pat", age, sex }).score).toBe(points);
      }
    }
  });
});

// ---------------------------------------------------------------- DASI, PRISMA-7 and Apfel (S48)

/** One HQ Question Set holding these elements; the answers given. */
const hqWith = (elements: ChapterJson["pages"] extends (infer P)[] | undefined ? P extends { elements?: infer E } ? E : never : never, answers: Record<string, unknown>, patient = { name: "Pat", age: 70, sex: "female" }): PrefillFrom => ({
  hq: { sets: [{ id: "s1", name: "Set", content: { pages: [{ name: "p_1", title: "Set", elements }] } }], answers: { s1: answers }, patient },
  patient,
  obsPrecedence: true,
});
const yesTo = (keys: string[], yes: string[]) => Object.fromEntries(keys.map((k) => [k, yes.includes(k) ? "o_yes" : "o_no"]));

describe("the assessments offered (S48)", () => {
  it("are STOP-Bang, DASI, PRISMA-7 and Apfel PONV, in that order", () => {
    expect(ASSESSMENTS).toEqual([
      { id: "stopbang", name: "STOP-Bang" }, { id: "dasi", name: "DASI" }, { id: "prisma7", name: "PRISMA-7" }, { id: "apfel", name: "Apfel PONV" },
    ]);
  });
  it("each is what Go stores, and so is the list (regenerate with `npm run assessments -w packages/clinical`)", () => {
    const dir = new URL("../../../apps/api/internal/assessment/", import.meta.url);
    expect(JSON.parse(readFileSync(new URL("list.json", dir), "utf8"))).toEqual(ASSESSMENTS);
    for (const { id } of ASSESSMENTS) expect(JSON.parse(readFileSync(new URL(`${id}.json`, dir), "utf8"))).toEqual(JSON.parse(JSON.stringify(assessmentContent(id))));
  });
  it("each names its items with short labels, for Go's messages", () => {
    expect(assessmentContent("stopbang")!.labels).toMatchObject({ stopbang_bmi: "BMI", stopbang_neck: "Neck", stopbang_snoring: "Snoring" });
    expect(assessmentContent("dasi")!.labels).toMatchObject({ dasi_run: "Run a short distance" });
    expect(assessmentContent("apfel")!.labels).toMatchObject({ apfel_opioids: "Opioids after surgery" });
  });
});

describe("DASI on an episode", () => {
  const dasi = assessmentContent("dasi")!;
  it("an HQ with the ready-made DASI fills all 12, each from the HQ", () => {
    const items = assessmentItems(dasi, hqWith([dasiGroup()], yesTo(DASI_KEYS.items, ["dasi_selfcare", "dasi_walk_indoors"])));
    expect(items.map((i) => i.letter)).toEqual(["1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11", "12"]);
    expect(items.every((i) => i.source === "hq")).toBe(true);
    expect(filledCount(items)).toBe(12);
    expect(items.map((i) => i.yes)).toEqual([true, true, false, false, false, false, false, false, false, false, false, false]);
  });
  it("gives the score, its band and the METs", () => {
    const yes = ["dasi_selfcare", "dasi_walk_indoors", "dasi_stairs", "dasi_run", "dasi_moderate_work", "dasi_heavy_work", "dasi_yardwork"];
    expect(assessmentResult(dasi, yesTo(DASI_KEYS.items, yes), { name: "Pat", age: 70, sex: "female" }))
      .toEqual({ score: 34, band: "Poor functional capacity", text: "34 (Poor functional capacity), 6.9 METs" });
    expect(assessmentResult(dasi, yesTo(DASI_KEYS.items, DASI_KEYS.items), { name: "Pat" }).text).toBe("58.2, 9.9 METs");
  });
  it("an item the HQ didn't ask is missing", () => {
    const items = assessmentItems(dasi, hqWith([], {}));
    expect(filledCount(items)).toBe(0);
    expect(assessmentResult(dasi, {}, { name: "Pat" }).text).toBe("Incomplete");
  });
});

describe("PRISMA-7 on an episode", () => {
  const prisma = assessmentContent("prisma7")!;
  it("an HQ with the ready-made PRISMA-7 fills all 7: questions 3–7 from the HQ, the age and sex from the profile", () => {
    const items = assessmentItems(prisma, hqWith([prisma7Group()], yesTo(PRISMA7_KEYS.items, ["prisma7_help", "prisma7_aid"]), { name: "Pat", age: 87, sex: "female" }));
    expect(items.map((i) => [i.letter, i.source])).toEqual([["1", "profile"], ["2", "profile"], ["3", "hq"], ["4", "hq"], ["5", "hq"], ["6", "hq"], ["7", "hq"]]);
    expect(items.map((i) => i.yes)).toEqual([true, false, false, true, false, false, true]);
    expect(assessmentResult(prisma, assessmentAnswers(items).data, { name: "Pat", age: 87, sex: "female" }).text).toBe("3 (Increased risk of frailty)");
  });
  it("85 isn't older than 85; a sex of Other leaves question 2 missing", () => {
    const items = assessmentItems(prisma, hqWith([prisma7Group()], {}, { name: "Pat", age: 85, sex: "other" }));
    expect(items[0]).toMatchObject({ value: 85, source: "profile", yes: false });
    expect(items[1].source).toBeUndefined();
  });
});

describe("Apfel PONV on an episode", () => {
  const apfel = assessmentContent("apfel")!;
  const from = hqWith([apfelGroup()], { [APFEL_KEYS.smoking]: "o_no", [APFEL_KEYS.ponv]: "o_yes" }, { name: "Pat", age: 40, sex: "female" });
  it("an HQ with Apfel's questions fills 3 of 4: only the post-op opioids is left, for the clinician", () => {
    const items = assessmentItems(apfel, from);
    expect(items.map((i) => [i.letter, i.label, i.source ?? null])).toEqual([
      ["1", "Female", "profile"], ["2", "Smoking", "hq"], ["3", "PONV or travel sickness before", "hq"], ["4", "Opioids after surgery", null],
    ]);
    expect(items.map((i) => i.yes)).toEqual([true, true, true, undefined]); // a No to smoking is a non-smoker's point
    expect(assessmentResult(apfel, assessmentAnswers(items).data, from.patient).text).toBe("Incomplete");
  });
  it("with the clinician's answer, gives the score and the published risk", () => {
    const items = withDraft(apfel, assessmentItems(apfel, from), { [APFEL_KEYS.opioids]: "o_yes" });
    expect(items[3]).toMatchObject({ source: "clinician", yes: true });
    expect(assessmentResult(apfel, assessmentAnswers(items).data, from.patient).text).toBe("4 (PONV risk about 79%)");
  });
});
