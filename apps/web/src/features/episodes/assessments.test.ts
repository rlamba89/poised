import { describe, expect, it } from "vitest";
import {
  assessmentAnswers, assessmentContent, assessmentResult, CFS_KEYS, DASI_KEYS, MORSE_KEYS, MUST_KEYS, mustGroup, STOP_BANG_BMI, STOP_BANG_KEYS as K, type Reading,
} from "@poised/clinical";
import { formatDateTime } from "@/lib/format";
import { boxPlaceholder, boxTyped, numberInput, cardHints, cardNotes, changedItems, confirmText, sinceConfirmed, filledText, inputsState, patientOn, scoreDay, withoutInvalid, poaLines, prefillFrom, saveRequest, shownItems, sourceText, valueText, type AssessmentRow, type AssessmentsView } from "./assessments";
import type { EpisodeHQ } from "./hq";
import type { Episode } from "./types";

const content = assessmentContent("stopbang")!;
const episode = { firstName: "Pat", lastName: "Patient", dateOfBirth: "1960-03-01", sex: "male", patientSubmittedAt: "2026-11-01T10:00:00Z", hospitalTimeZone: "UTC" } as Episode;
const yesNo = (name: string) => ({ type: "radiogroup", name, title: name, choices: [{ value: "o_yes", text: "Yes" }, { value: "o_no", text: "No" }] });
const hq: EpisodeHQ = {
  chapters: [{
    id: "s1", name: "Sleep", description: "", icon: "", audience: "patient",
    content: { pages: [{ name: "p1", elements: [yesNo(K.snoring), yesNo(K.tired), yesNo(K.observed), yesNo(K.pressure)] }] },
  }],
  patient: { s1: { [K.snoring]: "o_yes", [K.tired]: "o_yes", [K.observed]: "o_no", [K.pressure]: "o_yes" } },
  enteredBy: {},
  clinician: {},
  prefill: {},
};
// The clinician started validating (Validate starts from the patient's answers) and changed Tired,
// but hasn't pressed Validate: their copy counts already (S47 answers 5).
const started: EpisodeHQ = {
  ...hq,
  clinician: {
    s1: {
      chapterId: "s1", actor: "clinician", data: { ...hq.patient.s1, [K.tired]: "o_no" }, updatedAt: "", validatedAt: null, updatedByName: "Cara", revision: 1,
    },
  },
  prefill: {},
};
const reading = (kind: string, value: number): Reading => ({
  id: kind, kind, value, value2: null, unit: null, note: null, takenAt: "2026-11-05T09:00:00Z", enteredAt: "2026-11-05T09:00:00Z",
  enteredByName: "Cara", corrects: null, reason: null, supersededBy: null, errorAt: null, errorByName: null, errorReason: null,
});
const readings = [reading("height", 170), reading("weight", 105.2)]; // BMI 36.4
const row = (o: Partial<AssessmentRow> = {}): AssessmentRow => ({
  id: "f1", assessment: "stopbang", name: "STOP-Bang", createdAt: "", versionNo: 1, chapterId: "c1", content,
  data: null, sources: null, revision: 0, updatedAt: null, confirmedAt: null, updatedByName: null, ...o,
});

describe("an assessment as shown", () => {
  it("is pre-filled from the HQ's answers that count, the hospital BMI and the profile: 7 of 8, the neck missing", () => {
    const items = shownItems(row(), prefillFrom(hq, episode, readings, true), {});
    expect(items.map((i) => [i.letter, valueText(i), sourceText(i)])).toEqual([
      ["S", "Yes", "HQ"], ["T", "Yes", "HQ"], ["O", "No", "HQ"], ["P", "Yes", "HQ"],
      ["B", "36.4", "Hospital BMI"], ["A", "66", "Profile"], ["N", "—", "Missing"], ["G", "Male", "Profile"],
    ]);
  });
  it("the clinician's copy counts once they've started it, validated or not (S47 answers 5)", () => {
    const tired = (h: EpisodeHQ) => valueText(shownItems(row(), prefillFrom(h, episode, readings, true), {}).find((i) => i.key === K.tired)!);
    expect(tired(hq)).toBe("Yes"); // not started: the patient's
    expect(tired(started)).toBe("No");
    expect(tired({ ...started, clinician: { s1: { ...started.clinician.s1, validatedAt: "2026-11-02T10:00:00Z" } } })).toBe("No");
  });
  it("before the patient sends the HQ, their saved answers don't count", () => {
    const items = shownItems(row(), prefillFrom(hq, { ...episode, patientSubmittedAt: null }, readings, true), {});
    expect(items.filter((i) => i.source === "hq")).toEqual([]);
  });
  it("without observations the BMI is missing (the HQ asked for none)", () => {
    expect(sourceText(shownItems(row(), prefillFrom(hq, episode, undefined, true), {}).find((i) => i.key === STOP_BANG_BMI)!)).toBe("Missing");
  });
  it("a save sends each answer with its source and the row's revision", () => {
    const items = shownItems(row({ revision: 3 }), prefillFrom(hq, episode, readings, true), { [K.neck]: "o_no" });
    expect(saveRequest(row({ revision: 3 }), items)).toEqual({
      // The age and sex too: frozen at Confirm (S47 answers 7).
      data: { [K.snoring]: "o_yes", [K.tired]: "o_yes", [K.observed]: "o_no", [K.pressure]: "o_yes", [STOP_BANG_BMI]: 36.4, patientAge: 66, [K.neck]: "o_no", patientSex: "male" },
      sources: {
        [K.snoring]: "hq", [K.tired]: "hq", [K.observed]: "hq", [K.pressure]: "hq", [STOP_BANG_BMI]: "observation", patientAge: "profile", [K.neck]: "clinician", patientSex: "profile",
      },
      revision: 3,
    });
  });
});

describe("Confirm and the POA Summary", () => {
  const data = { [K.snoring]: "o_yes", [K.tired]: "o_yes", [K.observed]: "o_no", [K.pressure]: "o_yes", [STOP_BANG_BMI]: 36.4, [K.neck]: "o_no" };
  const view = (r: AssessmentRow): AssessmentsView => ({ assessments: [r], available: [], obsPrecedence: true });
  it("a confirmed one shows its score and band, with the patient's age on the day of Confirm", () => {
    const confirmed = row({ data, sources: {}, revision: 2, confirmedAt: "2026-11-05T10:00:00Z", updatedByName: "Cara Clinician" });
    expect(poaLines(view(confirmed), episode, "")).toEqual([{ name: "STOP-Bang", text: "6 (High risk)" }]);
    expect(confirmText(confirmed)).toBe(`Confirmed by Cara Clinician on ${formatDateTime("2026-11-05T10:00:00Z")}`);
  });
  it("a confirmed one keeps the age and sex it was confirmed with, whatever the record says now (S47 answers 7)", () => {
    const frozen = row({ data: { ...data, patientAge: 66, patientSex: "male" }, sources: {}, revision: 2, confirmedAt: "2026-11-05T10:00:00Z" });
    const corrected = { ...episode, dateOfBirth: "1990-03-01", sex: "female" }; // 36 and female: 4 (High risk) if it followed
    expect(poaLines(view(frozen), corrected, "")).toEqual([{ name: "STOP-Bang", text: "6 (High risk)" }]);
  });
  it("one not confirmed says so", () => {
    expect(poaLines(view(row({ data, sources: {}, revision: 1 })), episode, "")).toEqual([{ name: "STOP-Bang", text: "Not confirmed" }]);
    expect(confirmText(row())).toBe("Not confirmed");
  });
  it("none with assessments off, or before they're loaded", () => {
    expect(poaLines("off", episode, "")).toEqual([]);
    expect(poaLines(null, episode, "")).toEqual([]);
  });
  it("one line saying so when they couldn't be loaded, never silently none (verification 2)", () => {
    expect(poaLines(null, episode, "Server error")).toEqual([{ name: "Assessments", text: "Couldn't be loaded" }]);
    expect(poaLines("off", episode, "Server error")).toEqual([]);
  });
});

describe("the BMI box (report 1)", () => {
  const b = { min: 5, max: 150 };
  it("takes a number with at most one decimal place, a comma allowed", () => {
    expect(numberInput("36.5", b)).toEqual({ value: 36.5 });
    expect(numberInput(" 36,5 ", b)).toEqual({ value: 36.5 });
    expect(numberInput("40", b)).toEqual({ value: 40 });
  });
  it("says what's wrong with anything else, and gives no value", () => {
    expect(numberInput("abc", b)).toEqual({ error: "A number, one decimal place" });
    expect(numberInput("36.55", b)).toEqual({ error: "A number, one decimal place" });
    expect(numberInput("-5", b)).toEqual({ error: "A number, one decimal place" });
  });
  it("says the limits for a number outside them", () => {
    expect(numberInput("2", b)).toEqual({ error: "Between 5 and 150" });
    expect(numberInput("151", b)).toEqual({ error: "Between 5 and 150" });
    expect(numberInput("150", b)).toEqual({ value: 150 });
  });
  it("an empty box is neither a value nor a mistake", () => {
    expect(numberInput("  ", b)).toEqual({});
  });
});
describe("what a number box shows (S49 test report 7, F-1)", () => {
  const bmi = { key: "bmi", letter: "1", label: "BMI", kind: "number", min: 5, max: 150, measure: "bmi" } as const;
  it("keeps what the clinician typed while the item holds it", () => {
    expect(boxTyped("20.8", { ...bmi, value: 20.8, source: "observation" })).toBe("20.8");
    expect(boxTyped("36", { ...bmi, value: 36, source: "clinician" })).toBe("36");
  });
  it("drops a value typed earlier once the pairing rule has emptied the item: the box is empty", () => {
    expect(boxTyped("20.8", { ...bmi, value: undefined, source: undefined })).toBeNull();
  });
  it("keeps a mistake or an emptied box, which the item never held", () => {
    expect(boxTyped("1", { ...bmi, value: undefined, source: undefined })).toBe("1");
    expect(boxTyped("", { ...bmi, value: undefined, source: undefined })).toBe("");
  });
  it("not typed in: nothing", () => {
    expect(boxTyped(null, { ...bmi, value: 20.8, source: "observation" })).toBeNull();
  });
});

describe("how many are filled", () => {
  const items = shownItems(row(), prefillFrom(hq, episode, readings, true), {});
  it("asks for the missing items", () => {
    expect(filledText(items, false)).toBe("7 of 8 filled: answer the item highlighted.");
    expect(filledText(shownItems(row(), prefillFrom(hq, episode, undefined, true), {}), false)).toBe("6 of 8 filled: answer the 2 items highlighted.");
  });
  it("on a cancelled or archived episode, only counts them", () => {
    expect(filledText(items, true)).toBe("7 of 8 filled.");
  });
  it("all filled", () => {
    expect(filledText(shownItems(row(), prefillFrom(hq, episode, readings, true), { [K.neck]: "o_no" }), false)).toBe("8 of 8 filled.");
  });
});

describe("a box holding something that isn't a value (report 2)", () => {
  const items = shownItems(row(), prefillFrom(hq, episode, readings, true), { [K.neck]: "o_no" });
  it("counts as missing, so the count and the score don't use the value typed before", () => {
    const shown = withoutInvalid(items, { [STOP_BANG_BMI]: true });
    const b = shown.find((i) => i.key === STOP_BANG_BMI)!;
    expect([b.value, b.source, b.yes]).toEqual([undefined, undefined, undefined]);
    expect(sourceText(b)).toBe("Missing");
    expect(filledText(shown, false)).toBe("7 of 8 filled: answer the item highlighted.");
  });
  it("a box that's fine again changes nothing", () => {
    expect(withoutInvalid(items, { [STOP_BANG_BMI]: false })).toEqual(items);
  });
});

describe("the day a score's age is worked out on (verification 1, finding 1)", () => {
  // Made up: born 1 March 1960, so 50 on the first Confirm (1 June 2010) and over 60 today.
  const confirmed = row({ data: { [K.neck]: "o_no" }, sources: { [K.neck]: "clinician" }, confirmedAt: "2010-06-01T10:00:00Z", updatedByName: "Cara" });
  it("is the day it was confirmed while it's shown as confirmed and unchanged", () => {
    expect(scoreDay(confirmed, false)).toBe("2010-06-01T10:00:00Z");
    expect(patientOn(episode, scoreDay(confirmed, false)).age).toBe(50);
  });
  it("is today once it's changed: the day Confirm will record", () => {
    expect(scoreDay(confirmed, true)).toBeUndefined();
    expect(patientOn(episode, scoreDay(confirmed, true)).age).toBeGreaterThan(60);
  });
  it("is today while it isn't confirmed", () => {
    expect(scoreDay(row(), false)).toBeUndefined();
  });
});

describe("whether the card's inputs are in (verification 1, finding 2)", () => {
  const view = { readings: [], ranges: {} } as never;
  const loadedHq = { hq, error: "" };
  it("ready once the HQ and the observations have loaded", () => {
    expect(inputsState(loadedHq, { view, error: "" })).toEqual({ ready: true });
  });
  it("ready with observations off: there's no hospital BMI to wait for", () => {
    expect(inputsState(loadedHq, { view: "off", error: "" })).toEqual({ ready: true });
  });
  it("not ready while either is loading", () => {
    expect(inputsState({ hq: null, error: "" }, { view, error: "" })).toEqual({ ready: false });
    expect(inputsState(loadedHq, { view: null, error: "" })).toEqual({ ready: false });
  });
  it("says which didn't load", () => {
    expect(inputsState(loadedHq, { view: null, error: "Server error" })).toEqual({
      ready: false, problem: "The observations didn't load, so the hospital's readings can't be checked. Reload the page to save or confirm.",
    });
    expect(inputsState({ hq: null, error: "Server error" }, { view, error: "" })).toEqual({
      ready: false, problem: "The health questionnaire didn't load, so its answers can't be checked. Reload the page to save or confirm.",
    });
  });
});

describe("a confirmed assessment beside newer values (S47 answers 6)", () => {
  const confirmedAt = "2026-11-05T10:00:00Z";
  const data = {
    [K.snoring]: "o_yes", [K.tired]: "o_yes", [K.observed]: "o_no", [K.pressure]: "o_yes", [STOP_BANG_BMI]: 36.4, patientAge: 66, [K.neck]: "o_no", patientSex: "male",
  };
  const sources = {
    [K.snoring]: "hq", [K.tired]: "hq", [K.observed]: "hq", [K.pressure]: "hq", [STOP_BANG_BMI]: "observation", patientAge: "profile", [K.neck]: "clinician", patientSex: "profile",
  } as const;
  const confirmed = row({ data, sources, revision: 2, confirmedAt });
  /** The card's lines: the items as confirmed against a fresh pre-fill (the record as of the day of Confirm). */
  const lines = (h: EpisodeHQ, e: Episode, r: Reading[] | undefined) => {
    const f = prefillFrom(h, e, r, true, confirmedAt);
    return sinceConfirmed(shownItems(confirmed, f, {}), shownItems(row(), f, {}));
  };
  it("nothing while nothing it used has changed", () => {
    expect(lines(hq, episode, readings)).toEqual([]);
  });
  it("a corrected weight: the hospital BMI now", () => {
    expect(lines(hq, episode, [reading("height", 170), reading("weight", 71.4)])).toEqual(["The hospital BMI is now 24.7 (36.4 when confirmed)."]);
  });
  it("the hospital BMI gone: the patient reported BMI is used now, or none", () => {
    const withBmi: EpisodeHQ = {
      ...hq,
      chapters: [{ ...hq.chapters[0], content: { pages: [{ name: "p1", elements: [...hq.chapters[0].content.pages![0].elements!, { type: "bmi", name: "q_bmi", title: "Height and weight" }] }] } }],
      patient: { s1: { ...hq.patient.s1, q_bmi: { height: 170, weight: 86.7 } } },
    };
    expect(lines(withBmi, episode, undefined)).toEqual(["The patient reported BMI is now 30 (Hospital BMI 36.4 when confirmed)."]);
    expect(lines(hq, episode, undefined)).toEqual(["There's no BMI now (36.4 when confirmed)."]);
  });
  it("an HQ answer changed, or no longer given", () => {
    const changed = { ...hq, patient: { s1: { ...hq.patient.s1, [K.snoring]: "o_no" } } };
    expect(lines(changed, episode, readings)).toEqual(["The HQ now says No to Snoring (Yes when confirmed)."]);
    const { [K.snoring]: _s, ...rest } = hq.patient.s1!;
    expect(lines({ ...hq, patient: { s1: rest } }, episode, readings)).toEqual(["The HQ no longer answers Snoring (Yes when confirmed)."]);
  });
  it("the clinician's own answers aren't compared: nothing they used has changed", () => {
    const c = row({ data: { ...data, [K.snoring]: "o_no" }, sources: { ...sources, [K.snoring]: "clinician" }, revision: 2, confirmedAt });
    const f = prefillFrom(hq, episode, readings, true, confirmedAt);
    expect(sinceConfirmed(shownItems(c, f, {}), shownItems(row(), f, {}))).toEqual([]);
  });
  it("the patient's record corrected: their age that day, or their sex", () => {
    expect(lines(hq, { ...episode, dateOfBirth: "1990-03-01" }, readings)).toEqual(["The patient's record now gives an age of 36 that day (66 when confirmed)."]);
    expect(lines(hq, { ...episode, sex: "female" }, readings)).toEqual(["The patient's record now says Female (Male when confirmed)."]);
    expect(lines(hq, { ...episode, sex: "other" }, readings)).toEqual(["The patient's record no longer says male or female (Male when confirmed)."]);
  });
  it("once changed, the age and sex follow the record, so a new Confirm stores them (report 4, F-1)", () => {
    // Confirmed for a man of 66; the record is then corrected to a woman born 1990, and N changed.
    const corrected = { ...episode, dateOfBirth: "1990-03-01", sex: "female" };
    const today = prefillFrom(hq, corrected, readings, true); // once changed: today, as a Confirm records
    const items = changedItems(confirmed, today, { [K.neck]: "o_yes" });
    const value = (k: string) => items.find((i) => i.key === k);
    expect([value("patientAge")?.value, value("patientSex")?.value, value("patientSex")?.source]).toEqual([36, "female", "profile"]);
    expect(value(K.neck)).toMatchObject({ value: "o_yes", source: "clinician" });
    // The rest stays as confirmed: the BMI it was confirmed with, until that item is changed.
    expect(value(STOP_BANG_BMI)).toMatchObject({ value: 36.4, source: "observation" });
    expect(assessmentResult(content, assessmentAnswers(items).data, today.patient).text).toBe("5 (High risk)");
    expect(saveRequest(confirmed, items)).toMatchObject({ data: { patientAge: 36, patientSex: "female" }, sources: { patientAge: "profile", patientSex: "profile" } });
  });
  it("a birthday since isn't a change: the age is the record's on the day of Confirm", () => {
    const before = row({ data: { ...data, patientAge: 65 }, sources, revision: 2, confirmedAt: "2026-02-15T10:00:00Z" }); // 66 on 1 March
    const f = prefillFrom(hq, episode, readings, true, before.confirmedAt!);
    expect(sinceConfirmed(shownItems(before, f, {}), shownItems(row(), f, {}))).toEqual([]);
  });
});

describe("DASI, PRISMA-7 and Apfel on the card and the POA Summary (S48)", () => {
  const dasi = assessmentContent("dasi")!;
  const yes = ["dasi_selfcare", "dasi_walk_indoors", "dasi_stairs", "dasi_run", "dasi_moderate_work", "dasi_heavy_work", "dasi_yardwork"];
  const dasiData = Object.fromEntries(DASI_KEYS.items.map((k) => [k, yes.includes(k) ? "o_yes" : "o_no"]));
  it("the POA Summary gives DASI's score, band and METs", () => {
    const r = row({ assessment: "dasi", name: "DASI", content: dasi, data: dasiData, sources: {}, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    expect(poaLines({ assessments: [r], available: [], obsPrecedence: true }, episode, "")).toEqual([{ name: "DASI", text: "34 (Poor functional capacity), 6.9 METs" }]);
  });
  it("the card's notes name only the age and sex an assessment uses", () => {
    const dasiItems = shownItems(row({ assessment: "dasi", content: dasi }), prefillFrom(hq, episode, readings, true), {});
    expect(cardNotes(dasiItems, undefined)).toEqual([]);
    const unknownSex = { ...episode, sex: "unknown" } as Episode;
    const apfel = shownItems(row({ assessment: "apfel", content: assessmentContent("apfel")! }), prefillFrom(hq, unknownSex, readings, true), {});
    expect(cardNotes(apfel, undefined)).toEqual(["The patient's record doesn't say male or female, so the score stays Incomplete, as in the HQ."]);
    const stopBang = shownItems(row(), prefillFrom(hq, episode, readings, true), {});
    expect(cardNotes(stopBang, undefined)).toEqual(["The age is the patient's today."]);
    expect(cardNotes(stopBang, "2026-11-05T10:00:00Z")).toEqual(["The age is the patient's on the day it was confirmed (66)."]);
  });
});

describe("the pre-fill's HQ part is worked out once (S47 verification 4)", () => {
  it("is the same for any day, BMI and setting while the HQ and the patient are unchanged", () => {
    const a = prefillFrom(hq, episode, readings, true).hq;
    expect(prefillFrom(hq, { ...episode }, undefined, false, "2026-11-05T10:00:00Z").hq).toBe(a);
  });
  it("is worked out afresh for a reloaded HQ, a send, or a corrected record", () => {
    const a = prefillFrom(hq, episode, readings, true).hq;
    expect(prefillFrom({ ...hq }, episode, readings, true).hq).not.toBe(a);
    expect(prefillFrom(hq, { ...episode, patientSubmittedAt: null }, readings, true).hq).not.toBe(a);
    expect(prefillFrom(hq, { ...episode, dateOfBirth: "1970-03-01" }, readings, true).hq).not.toBe(a);
  });
});

describe("MUST, Morse, VTE and the Clinical Frailty Scale on the card (S49)", () => {
  const must = assessmentContent("must")!;
  const mustHq: EpisodeHQ = {
    ...hq,
    chapters: [{ id: "s1", name: "Weight", description: "", icon: "", audience: "patient", content: { pages: [{ name: "p1", elements: [mustGroup()] }] } }],
    patient: { s1: { [MUST_KEYS.lost]: "o_yes", [MUST_KEYS.lostKg]: 6 } },
  };
  const mustRow = (o: Partial<AssessmentRow> = {}) => row({ assessment: "must", name: "MUST", content: must, ...o });
  it("MUST takes the hospital BMI and weight from the readings, and the weight lost from the HQ", () => {
    const items = shownItems(mustRow(), prefillFrom(mustHq, episode, readings, true), {});
    expect(items.map((i) => [i.value ?? null, sourceText(i)])).toEqual([[36.4, "Hospital BMI"], [105.2, "Hospital weight"], [6, "HQ"], [null, "Missing"]]);
  });
  it("a corrected weight changes MUST's BMI and weight before Confirm", () => {
    const corrected = [reading("height", 170), { ...reading("weight", 105.2), supersededBy: "w2" }, { ...reading("weight", 60), id: "w2" }];
    const items = shownItems(mustRow(), prefillFrom(mustHq, episode, corrected, true), {});
    expect(items.slice(0, 2).map((i) => i.value)).toEqual([20.8, 60]);
  });
  it("a confirmed MUST says when the hospital weight has changed since", () => {
    const data = { [MUST_KEYS.bmi]: 36.4, [MUST_KEYS.weight]: 105.2, [MUST_KEYS.lostKg]: 6, [MUST_KEYS.acute]: "o_no" };
    const sources = { [MUST_KEYS.bmi]: "observation", [MUST_KEYS.weight]: "observation", [MUST_KEYS.lostKg]: "hq", [MUST_KEYS.acute]: "clinician" } as const;
    const confirmed = mustRow({ data, sources, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    const from = prefillFrom(mustHq, episode, [reading("height", 170), reading("weight", 100)], true);
    expect(sinceConfirmed(shownItems(confirmed, from, {}), shownItems(mustRow(), from, {}))).toEqual([
      "The hospital BMI is now 34.6 (36.4 when confirmed).", "The hospital weight is now 100 (105.2 when confirmed).",
    ]);
    const lessLost = { ...mustHq, patient: { s1: { [MUST_KEYS.lost]: "o_no" } } };
    expect(sinceConfirmed(shownItems(confirmed, prefillFrom(lessLost, episode, readings, true), {}), shownItems(mustRow(), prefillFrom(lessLost, episode, readings, true), {})))
      .toEqual(["The HQ now gives 0 kg for Unplanned weight loss in the past 3–6 months (6 kg when confirmed)."]);
  });
  it("a hospital weight without a height: the weight, and the BMI left for the clinician (verification 1)", () => {
    const weightOnly = [reading("weight", 60)];
    const items = shownItems(mustRow(), prefillFrom(mustHq, episode, weightOnly, true), {});
    expect(items.slice(0, 2).map((i) => [i.value ?? null, sourceText(i)])).toEqual([[null, "Missing"], [60, "Hospital weight"]]);
    const data = { [MUST_KEYS.bmi]: 36.4, [MUST_KEYS.weight]: 105.2, [MUST_KEYS.lostKg]: 6, [MUST_KEYS.acute]: "o_no" };
    const sources = { [MUST_KEYS.bmi]: "observation", [MUST_KEYS.weight]: "observation", [MUST_KEYS.lostKg]: "hq", [MUST_KEYS.acute]: "clinician" } as const;
    const confirmed = mustRow({ data, sources, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    const from = prefillFrom(mustHq, episode, weightOnly, true);
    expect(sinceConfirmed(shownItems(confirmed, from, {}), shownItems(mustRow(), from, {}))).toEqual([
      "The hospital has a weight but no height now (BMI 36.4 when confirmed).", "The hospital weight is now 60 (105.2 when confirmed).",
    ]);
  });
  it("confirmed on the patient's pair, then a hospital weight alone: says the hospital has no height, not that there's no BMI (verification 3)", () => {
    const withBody: EpisodeHQ = {
      ...mustHq,
      chapters: [{ ...mustHq.chapters[0], content: { pages: [{ name: "p1", elements: [mustGroup(), { type: "bmi", name: "q_bmi", title: "Height and weight" }] }] } }],
      patient: { s1: { [MUST_KEYS.lost]: "o_no", q_bmi: { height: 165, weight: 70 } } },
    };
    const data = { [MUST_KEYS.bmi]: 25.7, [MUST_KEYS.weight]: 70, [MUST_KEYS.lostKg]: 0, [MUST_KEYS.acute]: "o_no" };
    const sources = { [MUST_KEYS.bmi]: "hq", [MUST_KEYS.weight]: "hq", [MUST_KEYS.lostKg]: "hq", [MUST_KEYS.acute]: "clinician" } as const;
    const confirmed = mustRow({ data, sources, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    const from = prefillFrom(withBody, episode, [reading("weight", 66)], true);
    expect(sinceConfirmed(shownItems(confirmed, from, {}), shownItems(mustRow(), from, {}))).toEqual([
      "The hospital has a weight but no height now (BMI 25.7 when confirmed).", "The hospital weight is now 66 (Patient reported weight 70 when confirmed).",
    ]);
  });
  it("says the readings can't be checked when they didn't load", () => {
    expect(inputsState({ hq, error: "" }, { view: null, error: "boom" }).problem).toBe(
      "The observations didn't load, so the hospital's readings can't be checked. Reload the page to save or confirm.");
  });
  it("a weight lost of 0 is a value, and the box says the item's limits", () => {
    expect(numberInput("0", { min: 0, max: 100 })).toEqual({ value: 0 });
    expect(numberInput("101", { min: 0, max: 100 })).toEqual({ error: "Between 0 and 100" });
    expect(numberInput("0", { min: 5, max: 150 })).toEqual({ error: "Between 5 and 150" });
  });
  it("a choice of several shows the option's words", () => {
    const morse = shownItems(row({ assessment: "morse", content: assessmentContent("morse")! }), prefillFrom(hq, episode, readings, true), { [MORSE_KEYS.aid]: "o_aid_walker" });
    expect(valueText(morse[2])).toBe("Crutches, a stick or a walking frame");
  });
  it("the POA Summary gives each confirmed result", () => {
    const cfs = row({ assessment: "rockwood", name: "Clinical Frailty Scale", content: assessmentContent("rockwood")!, data: { [CFS_KEYS.level]: "o_cfs_4" }, sources: {}, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    expect(poaLines({ assessments: [cfs], available: [], obsPrecedence: true }, episode, "")).toEqual([{ name: "Clinical Frailty Scale", text: "4 (Living with Very Mild Frailty)" }]);
  });
  describe("the Clinical Frailty Scale's hints", () => {
    const cfs = row({ id: "f9", assessment: "rockwood", name: "Clinical Frailty Scale", content: assessmentContent("rockwood")! });
    const dasiData = Object.fromEntries(DASI_KEYS.items.map((k) => [k, "o_no"]));
    const dasi = row({ id: "f2", assessment: "dasi", name: "DASI", content: assessmentContent("dasi")!, data: dasiData, sources: {}, revision: 1, confirmedAt: "2026-11-05T10:00:00Z" });
    const prisma = row({ id: "f3", assessment: "prisma7", name: "PRISMA-7", content: assessmentContent("prisma7")!, data: {}, sources: {}, revision: 1 });
    it("are DASI's and PRISMA-7's results once confirmed", () => {
      expect(cardHints(cfs, { assessments: [cfs, dasi, prisma], available: [], obsPrecedence: true }, episode)).toEqual([
        `DASI: 0 (Poor functional capacity), 2.7 METs (confirmed ${formatDateTime("2026-11-05T10:00:00Z")})`,
      ]);
    });
    it("none when neither is confirmed, and none on other assessments", () => {
      expect(cardHints(cfs, { assessments: [cfs, prisma], available: [], obsPrecedence: true }, episode)).toEqual([]);
      expect(cardHints(dasi, { assessments: [cfs, dasi], available: [], obsPrecedence: true }, episode)).toEqual([]);
    });
  });
});

describe("a number box's placeholder (S49 verification 2)", () => {
  it("is BMI for a BMI, else the unit its wording ends with, else nothing", () => {
    const items = shownItems(row({ assessment: "must", content: assessmentContent("must")! }), prefillFrom(hq, episode, readings, true), {});
    expect(items.slice(0, 3).map(boxPlaceholder)).toEqual(["BMI", "kg", "kg"]);
    expect(boxPlaceholder({ key: "x", letter: "1", label: "Peak flow", kind: "number", title: "Peak flow" })).toBe("");
  });
});
