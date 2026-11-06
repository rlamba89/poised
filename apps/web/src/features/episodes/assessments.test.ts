import { describe, expect, it } from "vitest";
import { assessmentAnswers, assessmentContent, assessmentResult, STOP_BANG_BMI, STOP_BANG_KEYS as K, type Reading } from "@poised/clinical";
import { formatDateTime } from "@/lib/format";
import { bmiInput, changedItems, confirmText, sinceConfirmed, filledText, inputsState, patientOn, scoreDay, withoutInvalid, poaLines, prefillFrom, saveRequest, shownItems, sourceText, valueText, type AssessmentRow, type AssessmentsView } from "./assessments";
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
    expect(bmiInput("36.5", b)).toEqual({ value: 36.5 });
    expect(bmiInput(" 36,5 ", b)).toEqual({ value: 36.5 });
    expect(bmiInput("40", b)).toEqual({ value: 40 });
  });
  it("says what's wrong with anything else, and gives no value", () => {
    expect(bmiInput("abc", b)).toEqual({ error: "A number, one decimal place" });
    expect(bmiInput("36.55", b)).toEqual({ error: "A number, one decimal place" });
    expect(bmiInput("-5", b)).toEqual({ error: "A number, one decimal place" });
  });
  it("says the limits for a number outside them", () => {
    expect(bmiInput("2", b)).toEqual({ error: "Between 5 and 150" });
    expect(bmiInput("151", b)).toEqual({ error: "Between 5 and 150" });
    expect(bmiInput("150", b)).toEqual({ value: 150 });
  });
  it("an empty box is neither a value nor a mistake", () => {
    expect(bmiInput("  ", b)).toEqual({});
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
      ready: false, problem: "The observations didn't load, so the hospital BMI can't be checked. Reload the page to save or confirm.",
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
