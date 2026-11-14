// Assessments (S47): STOP-Bang on an episode, pre-filled from the HQ, the observations and the
// profile, each item with its source; the clinician completes and confirms it.
import { describe, expect, it } from "vitest";
import { readFileSync } from "node:fs";
import goStopBang from "../../../apps/api/internal/assessment/stopbang.json";
import {
  ASSESSMENTS, assessmentAnswers, assessmentContent, assessmentHints, assessmentItems, assessmentResult, countingHqAnswers, filledCount, STOP_BANG_BMI,
  draftAfter, shownWithout, withDraft, type PrefillFrom,
} from "./assessments";
import type { ChapterJson } from "./doc";
import {
  APFEL_KEYS, apfelGroup, CFS_KEYS, DASI_KEYS, dasiGroup, insertReadyScore, MORSE_KEYS, morseGroup, MUST_KEYS, mustGroup, PRISMA7_KEYS, prisma7Group, STOP_BANG_KEYS as K,
  stopBangGroup, VTE_KEYS, vteGroup,
} from "./ready";

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
  it("are STOP-Bang, DASI, PRISMA-7, Apfel PONV, MUST, Morse, VTE and the Clinical Frailty Scale, in the plan's order (S49)", () => {
    expect(ASSESSMENTS).toEqual([
      { id: "stopbang", name: "STOP-Bang" }, { id: "dasi", name: "DASI" }, { id: "prisma7", name: "PRISMA-7" }, { id: "apfel", name: "Apfel PONV" },
      { id: "must", name: "MUST" }, { id: "morse", name: "Morse Fall Scale" }, { id: "vte", name: "VTE risk" }, { id: "rockwood", name: "Clinical Frailty Scale" },
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

// ---------------------------------------------------------------- S49

const WEIGHT_LOSS = { [MUST_KEYS.lost]: "o_yes", [MUST_KEYS.lostKg]: 6 };

describe("MUST on an episode (S49)", () => {
  const must = assessmentContent("must")!;
  const at = (answers: Record<string, unknown>, o: Partial<PrefillFrom> = {}): PrefillFrom => ({ ...hqWith([mustGroup()], answers), ...o });
  it("fills 3 of 4: the hospital BMI and weight from the observations, the weight lost from the HQ; the acute disease effect is the clinician's", () => {
    const items = assessmentItems(must, at(WEIGHT_LOSS, { hospitalBmi: 21.5, hospitalWeight: 64 }));
    expect(items.map((i) => [i.letter, i.key, i.kind, i.source ?? null, i.value ?? null])).toEqual([
      ["1", MUST_KEYS.bmi, "number", "observation", 21.5],
      ["2", MUST_KEYS.weight, "number", "observation", 64],
      ["3", MUST_KEYS.lostKg, "number", "hq", 6],
      ["4", MUST_KEYS.acute, "choice", null, null],
    ]);
    expect(items.map((i) => i.measure ?? null)).toEqual(["bmi", "weight", null, null]);
  });
  it("a No in the HQ is 0 kg lost; the amount isn't asked then", () => {
    const items = assessmentItems(must, at({ [MUST_KEYS.lost]: "o_no", [MUST_KEYS.lostKg]: 12 }, { hospitalBmi: 21.5, hospitalWeight: 64 }));
    expect(items[2]).toMatchObject({ value: 0, source: "hq" });
  });
  it("a Yes without how much, or no answer, is missing", () => {
    expect(assessmentItems(must, at({ [MUST_KEYS.lost]: "o_yes" }))[2].source).toBeUndefined();
    expect(assessmentItems(must, at({}))[2].source).toBeUndefined();
  });
  it("the weight follows the precedence as the BMI does: the patient's own when the trust says so", () => {
    const set = { pages: [{ name: "p_1", title: "Set", elements: [mustGroup(), { type: "bmi", name: "q_bmi", title: "Height and weight" }] }] };
    const from: PrefillFrom = {
      hq: { sets: [{ id: "s1", name: "Set", content: set }], answers: { s1: { ...WEIGHT_LOSS, q_bmi: { height: 170, weight: 70 } } }, patient: { name: "Pat", age: 70, sex: "female" } },
      patient: { name: "Pat", age: 70, sex: "female" }, hospitalBmi: 21.5, hospitalWeight: 64, obsPrecedence: false,
    };
    const items = assessmentItems(must, from);
    expect(items.slice(0, 2).map((i) => [i.value, i.source])).toEqual([[24.2, "hq"], [70, "hq"]]);
    expect(assessmentItems(must, { ...from, obsPrecedence: true }).slice(0, 2).map((i) => [i.value, i.source])).toEqual([[21.5, "observation"], [64, "observation"]]);
  });
  describe("the BMI and the weight come from one source (S49 verification 1, Rahul's answer 1)", () => {
    const set = { pages: [{ name: "p_1", title: "Set", elements: [mustGroup(), { type: "bmi", name: "q_bmi", title: "Height and weight" }] }] };
    const patient = { name: "Pat", age: 70, sex: "female" };
    const with_ = (o: Partial<PrefillFrom>, body: Record<string, unknown> = { q_bmi: { height: 165, weight: 70 } }): PrefillFrom => ({
      hq: { sets: [{ id: "s1", name: "Set", content: set }], answers: { s1: { ...WEIGHT_LOSS, ...body } }, patient }, patient, obsPrecedence: true, ...o,
    });
    const pair = (from: PrefillFrom) => assessmentItems(must, from).slice(0, 2).map((i) => [i.value ?? null, i.source ?? null]);
    it("on: a hospital weight without a height fills the weight and leaves the BMI for the clinician, never the patient's own BMI", () => {
      expect(pair(with_({ hospitalWeight: 60 }))).toEqual([[null, null], [60, "observation"]]);
    });
    it("off: the patient's own pair, whatever the hospital has", () => {
      expect(pair(with_({ hospitalWeight: 60, obsPrecedence: false }))).toEqual([[25.7, "hq"], [70, "hq"]]);
    });
    it("off, with no height or weight from the patient: the hospital's, the BMI missing without a height", () => {
      expect(pair(with_({ hospitalWeight: 60, obsPrecedence: false }, {}))).toEqual([[null, null], [60, "observation"]]);
    });
    it("on, with neither from the hospital: the patient's pair", () => {
      expect(pair(with_({}))).toEqual([[25.7, "hq"], [70, "hq"]]);
    });
    it("a patient's weight without a height is still their weight (verification 1, finding 3)", () => {
      const weightOnly = { pages: [{ name: "p_1", title: "Set", elements: [mustGroup(), { type: "text", inputType: "number", name: "weight", title: "Weight" }] }] };
      const from: PrefillFrom = { hq: { sets: [{ id: "s1", name: "Set", content: weightOnly }], answers: { s1: { ...WEIGHT_LOSS, weight: 68 } }, patient }, patient, obsPrecedence: true };
      expect(pair(from)).toEqual([[null, null], [68, "hq"]]);
    });
    it("off: a patient's weight without a height gives way to the hospital's full pair, but beats a hospital weight alone (verification 2, Rahul)", () => {
      const weightOnly = { pages: [{ name: "p_1", title: "Set", elements: [mustGroup(), { type: "text", inputType: "number", name: "weight", title: "Weight" }] }] };
      const from: PrefillFrom = { hq: { sets: [{ id: "s1", name: "Set", content: weightOnly }], answers: { s1: { ...WEIGHT_LOSS, weight: 68 } }, patient }, patient, hospitalWeight: 60, obsPrecedence: false };
      expect(pair(from)).toEqual([[null, null], [68, "hq"]]);
      expect(pair({ ...from, hospitalBmi: 20.8 })).toEqual([[20.8, "observation"], [60, "observation"]]);
    });
    it("STOP-Bang, with a BMI only, still takes the patient's BMI when the hospital has no height", () => {
      const sb = assessmentItems(content, from({ hospitalBmi: undefined }));
      expect(item(sb, STOP_BANG_BMI)).toMatchObject({ value: 30, source: "hq" });
    });
  });
  it("a corrected weight changes the BMI and the weight before Confirm, but not once confirmed", () => {
    const items = assessmentItems(must, at(WEIGHT_LOSS, { hospitalBmi: 21.5, hospitalWeight: 64 }));
    const saved = { ...assessmentAnswers(withDraft(must, items, { [MUST_KEYS.acute]: "o_no" })), confirmed: false };
    const corrected = assessmentItems(must, at(WEIGHT_LOSS, { hospitalBmi: 18.4, hospitalWeight: 55 }), saved);
    expect(corrected.slice(0, 2).map((i) => i.value)).toEqual([18.4, 55]);
    expect(corrected[3]).toMatchObject({ value: "o_no", source: "clinician" });
    const confirmed = assessmentItems(must, at(WEIGHT_LOSS, { hospitalBmi: 18.4, hospitalWeight: 55 }), { ...saved, confirmed: true });
    expect(confirmed.slice(0, 2).map((i) => i.value)).toEqual([21.5, 64]);
  });
  it("gives the score, its band and the weight lost", () => {
    const items = withDraft(must, assessmentItems(must, at(WEIGHT_LOSS, { hospitalBmi: 19.5, hospitalWeight: 64 })), { [MUST_KEYS.acute]: "o_no" });
    expect(items.map((i) => i.yes)).toEqual([true, undefined, undefined, false]); // a BMI of 20 or less scores; the weights only together
    expect(assessmentResult(must, assessmentAnswers(items).data, { name: "Pat" }).text).toBe("2 (High risk), 8.6% of body weight lost");
  });
  it("names its items for Go's messages", () => {
    expect(must.labels).toEqual({ [MUST_KEYS.bmi]: "BMI", [MUST_KEYS.weight]: "Weight now", [MUST_KEYS.lostKg]: "Weight lost", [MUST_KEYS.acute]: "Acute disease effect" });
  });
});

describe("MUST: a typed weight or BMI leaves the other one for the clinician (S49 verification 3, Rahul 13 Nov)", () => {
  const must = assessmentContent("must")!;
  const at = (o: Partial<PrefillFrom> = {}): PrefillFrom => ({ ...hqWith([mustGroup()], { [MUST_KEYS.lost]: "o_yes", [MUST_KEYS.lostKg]: 2 }), hospitalBmi: 20.8, hospitalWeight: 60, ...o });
  const pair = (items: ReturnType<typeof assessmentItems>) => items.slice(0, 2).map((i) => [i.value ?? null, i.source ?? null]);
  it("a weight typed on the card: the pre-filled BMI is Missing, and the score Incomplete", () => {
    const items = withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.weight]: 50, [MUST_KEYS.acute]: "o_no" });
    expect(pair(items)).toEqual([[null, null], [50, "clinician"]]);
    expect(assessmentResult(must, assessmentAnswers(items).data, { name: "Pat" }).text).toBe("Incomplete");
  });
  it("a BMI typed on the card: the pre-filled weight is Missing", () => {
    expect(pair(withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.bmi]: 17.3 }))).toEqual([[17.3, "clinician"], [null, null]]);
  });
  it("both typed: both kept", () => {
    expect(pair(withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.bmi]: 17.3, [MUST_KEYS.weight]: 50 }))).toEqual([[17.3, "clinician"], [50, "clinician"]]);
  });
  it("a saved clinician's weight, filled afresh: the BMI stays Missing", () => {
    const saved = { data: { [MUST_KEYS.weight]: 50, [MUST_KEYS.bmi]: 20.8 }, sources: { [MUST_KEYS.weight]: "clinician", [MUST_KEYS.bmi]: "observation" } as const, confirmed: false };
    expect(pair(assessmentItems(must, at(), saved))).toEqual([[null, null], [50, "clinician"]]);
  });
  it("a confirmed card shows exactly what was confirmed, on the card too (verification 4, finding 2)", () => {
    const saved = { data: { [MUST_KEYS.weight]: 50, [MUST_KEYS.bmi]: 17.3 }, sources: { [MUST_KEYS.weight]: "clinician", [MUST_KEYS.bmi]: "observation" } as const, confirmed: true };
    const items = assessmentItems(must, at(), saved);
    expect(pair(items)).toEqual([[17.3, "observation"], [50, "clinician"]]);
    expect(pair(withDraft(must, items, {}))).toEqual([[17.3, "observation"], [50, "clinician"]]);
    expect(pair(withDraft(must, items, { [MUST_KEYS.acute]: "o_yes" }))).toEqual([[17.3, "observation"], [50, "clinician"]]);
  });
  describe("the hospital's own number typed back after the other was typed is the clinician's (verification 4, finding 1)", () => {
    it("a BMI typed, then the hospital's weight typed back", () => {
      const items = withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.bmi]: 30, [MUST_KEYS.weight]: 60, [MUST_KEYS.acute]: "o_no" });
      expect(pair(items)).toEqual([[30, "clinician"], [60, "clinician"]]);
      expect(assessmentResult(must, assessmentAnswers(items).data, { name: "Pat" }).text).not.toBe("Incomplete");
    });
    it("a weight typed, then the pre-filled BMI typed back", () => {
      expect(pair(withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.weight]: 61, [MUST_KEYS.bmi]: 20.8 }))).toEqual([[20.8, "clinician"], [61, "clinician"]]);
    });
    it("a hospital weight with no height: the BMI typed keeps the hospital's weight (verification 5, Rahul 13 Nov)", () => {
      const items = assessmentItems(must, at({ hospitalBmi: undefined }));
      expect(pair(items)).toEqual([[null, null], [60, "observation"]]);
      expect(pair(withDraft(must, items, { [MUST_KEYS.bmi]: 20.8 }))).toEqual([[20.8, "clinician"], [60, "observation"]]);
      // Saved, then filled afresh: still the hospital's weight beside the clinician's BMI.
      const saved = { ...assessmentAnswers(withDraft(must, items, { [MUST_KEYS.bmi]: 20.8 })), confirmed: false };
      expect(pair(assessmentItems(must, at({ hospitalBmi: undefined }), saved))).toEqual([[20.8, "clinician"], [60, "observation"]]);
      // A weight typed over it is the clinician's, the BMI kept.
      expect(pair(withDraft(must, items, { [MUST_KEYS.bmi]: 20.8, [MUST_KEYS.weight]: 58 }))).toEqual([[20.8, "clinician"], [58, "clinician"]]);
    });
    it("the pre-filled number typed back with the other untouched stays where it came from", () => {
      expect(pair(withDraft(must, assessmentItems(must, at()), { [MUST_KEYS.weight]: 60 }))).toEqual([[20.8, "observation"], [60, "observation"]]);
    });
  });
  describe("a box typed in and put back counts as untouched (verification 5, finding 1)", () => {
    const type = (items: ReturnType<typeof assessmentItems>, steps: [string, unknown][]) =>
      steps.reduce((d, [k, v]) => draftAfter(must, items, d, k, v), {} as Record<string, unknown>);
    it("a BMI typed and typed back, then a weight: the BMI is Missing", () => {
      const items = assessmentItems(must, at());
      const back = type(items, [[MUST_KEYS.bmi, 36], [MUST_KEYS.bmi, 20.8]]);
      expect(back).toEqual({});
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.bmi, 36], [MUST_KEYS.bmi, 20.8], [MUST_KEYS.weight, 50]])))).toEqual([[null, null], [50, "clinician"]]);
    });
    it("the mirror: a weight typed and typed back, then a BMI: the weight is Missing", () => {
      const items = assessmentItems(must, at());
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.weight, 61], [MUST_KEYS.weight, 60], [MUST_KEYS.bmi, 17.3]])))).toEqual([[17.3, "clinician"], [null, null]]);
    });
    it("verification 4's cases are unchanged: the other typed first, then the hospital's number typed back", () => {
      const items = assessmentItems(must, at());
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.weight, 61], [MUST_KEYS.bmi, 20.8]])))).toEqual([[20.8, "clinician"], [61, "clinician"]]);
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.bmi, 30], [MUST_KEYS.weight, 60]])))).toEqual([[30, "clinician"], [60, "clinician"]]);
    });
    it("a box emptied goes back to as shown", () => {
      const items = assessmentItems(must, at());
      expect(type(items, [[MUST_KEYS.bmi, 36], [MUST_KEYS.bmi, undefined]])).toEqual({});
    });
    it("the other box put back as well: weight 61, BMI 20.8, weight 60 back, then weight 50: the BMI is Missing (verification 6)", () => {
      const items = assessmentItems(must, at());
      const back = type(items, [[MUST_KEYS.weight, 61], [MUST_KEYS.bmi, 20.8], [MUST_KEYS.weight, 60]]);
      expect(back).toEqual({});
      expect(pair(withDraft(must, items, back))).toEqual([[20.8, "observation"], [60, "observation"]]);
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.weight, 61], [MUST_KEYS.bmi, 20.8], [MUST_KEYS.weight, 60], [MUST_KEYS.weight, 50]])))).toEqual([[null, null], [50, "clinician"]]);
    });
    it("the mirror: BMI 36, weight 60, BMI 20.8 back, then BMI 25: the weight is Missing (verification 6)", () => {
      const items = assessmentItems(must, at());
      expect(type(items, [[MUST_KEYS.bmi, 36], [MUST_KEYS.weight, 60], [MUST_KEYS.bmi, 20.8]])).toEqual({});
      expect(pair(withDraft(must, items, type(items, [[MUST_KEYS.bmi, 36], [MUST_KEYS.weight, 60], [MUST_KEYS.bmi, 20.8], [MUST_KEYS.bmi, 25]])))).toEqual([[25, "clinician"], [null, null]]);
    });
    it("shownWithout: a box the pairing rule emptied showed nothing; a pre-filled one its value (finding 2)", () => {
      const items = assessmentItems(must, at());
      expect(shownWithout(must, items, { [MUST_KEYS.weight]: 50 }, MUST_KEYS.bmi)?.source).toBeUndefined();
      expect(shownWithout(must, items, {}, MUST_KEYS.bmi)).toMatchObject({ value: 20.8, source: "observation" });
      expect(shownWithout(must, items, { [MUST_KEYS.bmi]: 36 }, MUST_KEYS.bmi)).toMatchObject({ value: 20.8, source: "observation" });
    });
  });
  it("a card with a BMI only (STOP-Bang) is as before", () => {
    const items = withDraft(content, assessmentItems(content, from({ hospitalBmi: 36.4 })), { [K.neck]: "o_no" });
    expect(item(items, STOP_BANG_BMI)).toMatchObject({ value: 36.4, source: "observation" });
  });
});

describe("VTE is Incomplete while any item is missing, either result's (S49 verification 2, finding 1)", () => {
  const vte = assessmentContent("vte")!;
  const all = (except: string) => ({
    ...Object.fromEntries([...VTE_KEYS.thrombosis, ...VTE_KEYS.bleeding].filter((k) => k !== except).map((k) => [k, "o_no"])),
    [VTE_KEYS.bmi]: 24, patientAge: 70,
  });
  it("a bleeding item empty: Incomplete, though every thrombosis item is answered", () => {
    expect(assessmentResult(vte, all("vte_active_bleeding"), { name: "Pat", age: 70 })).toEqual({ text: "Incomplete" });
  });
  it("a thrombosis item empty: Incomplete", () => {
    expect(assessmentResult(vte, all("vte_dehydration"), { name: "Pat", age: 70 })).toEqual({ text: "Incomplete" });
  });
  it("every item: both results", () => {
    expect(assessmentResult(vte, all(""), { name: "Pat", age: 70 }).text).toBe("1 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0");
  });
});

describe("VTE's questions a man isn't asked (S49 verification 1, Rahul's answer 2)", () => {
  const vte = assessmentContent("vte")!;
  const male = { name: "Pat", age: 64, sex: "male" };
  it("are missing for the clinician, not No, even if an answer is held", () => {
    const items = assessmentItems(vte, hqWith([vteGroup()], yesTo(VTE_KEYS.patient, ["vte_hrt", "vte_pregnancy"]), male));
    const of = (k: string) => items.find((i) => i.key === k)!;
    for (const k of ["vte_pregnancy", "vte_contraceptive", "vte_hrt"]) expect(of(k).source).toBeUndefined();
    expect(of("vte_cancer")).toMatchObject({ value: "o_no", source: "hq" });
  });
  it("count on the card when the clinician ticks them: the assessment asks them of everyone", () => {
    const items = withDraft(vte, assessmentItems(vte, { ...hqWith([vteGroup()], yesTo(VTE_KEYS.patient, []), male), hospitalBmi: 24 }),
      Object.fromEntries([...VTE_KEYS.thrombosis, ...VTE_KEYS.bleeding].filter((k) => !VTE_KEYS.patient.includes(k) || ["vte_pregnancy", "vte_contraceptive", "vte_hrt"].includes(k))
        .map((k) => [k, k === "vte_hrt" ? "o_yes" : "o_no"])));
    expect(assessmentResult(vte, assessmentAnswers(items).data, male).text).toBe("2 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 0");
  });
});

describe("the Morse Fall Scale on an episode (S49)", () => {
  const morse = assessmentContent("morse")!;
  it("fills the patient's two items from the HQ; the clinician's four are missing", () => {
    const items = assessmentItems(morse, hqWith([morseGroup()], { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.aid]: "o_aid_walker" }));
    expect(items.map((i) => [i.letter, i.label, i.source ?? null])).toEqual([
      ["1", "History of falling", "hq"], ["2", "Secondary diagnosis", null], ["3", "Ambulatory aid", "hq"], ["4", "IV or heparin lock", null],
      ["5", "Gait", null], ["6", "Mental status", null],
    ]);
    expect(items[2]).toMatchObject({ kind: "choice", value: "o_aid_walker", yes: true });
    expect(items[2].options!.map((o) => o.text)).toEqual(["Nothing", "Crutches, a stick or a walking frame", "I hold on to furniture"]);
  });
  it("with the clinician's answers gives the trial's worked example: 65, high risk", () => {
    const items = withDraft(morse, assessmentItems(morse, hqWith([morseGroup()], { [MORSE_KEYS.falls]: "o_yes", [MORSE_KEYS.aid]: "o_aid_walker" })), {
      [MORSE_KEYS.secondary]: "o_no", [MORSE_KEYS.iv]: "o_no", [MORSE_KEYS.gait]: "o_gait_weak", [MORSE_KEYS.mental]: "o_mental_forgets",
    });
    expect(items.map((i) => i.yes)).toEqual([true, false, true, false, true, true]);
    expect(assessmentResult(morse, assessmentAnswers(items).data, { name: "Pat" }).text).toBe("65 (High risk)");
  });
});

describe("VTE on an episode (S49)", () => {
  const vte = assessmentContent("vte")!;
  it("has the tool's 30 risk factors: thrombosis T1–T19, bleeding B1–B11", () => {
    const items = assessmentItems(vte, hqWith([vteGroup()], {}));
    expect(items.map((i) => i.letter)).toEqual([...Array.from({ length: 19 }, (_, n) => `T${n + 1}`), ...Array.from({ length: 11 }, (_, n) => `B${n + 1}`)]);
    expect(items[1]).toMatchObject({ key: "patientAge", label: "Age over 60" });
    expect(items[4]).toMatchObject({ key: VTE_KEYS.bmi, label: "Obesity (BMI over 30)", measure: "bmi" });
  });
  it("pre-fills the patient's own risk factors from the HQ, the age from the record and the BMI from the observations", () => {
    const answers = { ...yesTo(VTE_KEYS.patient, ["vte_hrt", "vte_anticoagulant"]) };
    const items = assessmentItems(vte, { ...hqWith([vteGroup()], answers, { name: "Pat", age: 64, sex: "female" }), hospitalBmi: 31.2 });
    const filled = items.filter((i) => i.source);
    expect(Object.fromEntries(filled.map((i) => [i.key, i.source]))).toEqual({
      patientAge: "profile", [VTE_KEYS.bmi]: "observation", ...Object.fromEntries(VTE_KEYS.patient.map((k) => [k, "hq"])),
    });
    expect(filled.filter((i) => i.yes).map((i) => i.key)).toEqual(["patientAge", VTE_KEYS.bmi, "vte_hrt", "vte_anticoagulant"]);
    expect(filledCount(items)).toBe(2 + 9);
  });
  it("with every other item No: 3 thrombosis ticks, 1 bleeding", () => {
    const answers = yesTo(VTE_KEYS.patient, ["vte_hrt", "vte_anticoagulant"]);
    const from = { ...hqWith([vteGroup()], answers, { name: "Pat", age: 64, sex: "female" }), hospitalBmi: 31.2 };
    const rest = Object.fromEntries([...VTE_KEYS.thrombosis, ...VTE_KEYS.bleeding].filter((k) => !VTE_KEYS.patient.includes(k)).map((k) => [k, "o_no"]));
    const items = withDraft(vte, assessmentItems(vte, from), rest);
    expect(assessmentResult(vte, assessmentAnswers(items).data, from.patient).text).toBe("3 (Thrombosis risk: consider thromboprophylaxis), Bleeding risk factors: 1");
  });
});

describe("the Clinical Frailty Scale on an episode (S49)", () => {
  const cfs = assessmentContent("rockwood")!;
  it("is one item, the clinician's: the level, from nine", () => {
    const items = assessmentItems(cfs, hqWith([], {}));
    expect(items.map((i) => [i.letter, i.label, i.kind, i.source ?? null])).toEqual([["1", "Level", "choice", null]]);
    expect(items[0].options).toHaveLength(9);
    const picked = withDraft(cfs, items, { [CFS_KEYS.level]: "o_cfs_6" });
    expect(assessmentResult(cfs, assessmentAnswers(picked).data, { name: "Pat" }).text).toBe("6 (Living with Moderate Frailty)");
  });
  it("shows DASI's and PRISMA-7's results as hints; the others have none", () => {
    expect(assessmentHints("rockwood")).toEqual(["dasi", "prisma7"]);
    expect(assessmentHints("must")).toEqual([]);
  });
  it("is never filled from an HQ, whatever it holds", () => {
    const items = assessmentItems(cfs, hqWith([{ type: "radiogroup", name: CFS_KEYS.level, title: "Level?", choices: [{ value: "o_cfs_2", text: "2" }] }], { [CFS_KEYS.level]: "o_cfs_2" }));
    expect(items[0].source).toBeUndefined();
  });
});
