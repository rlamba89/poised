import { describe, expect, it } from "vitest";
import { bmiOf, counting, hospitalBmi, latest, outsideUsual, type ObservationRanges, type Reading } from "./observations";

let n = 0;
const r = (kind: string, value: number | null, takenAt: string, more: Partial<Reading> = {}): Reading => ({
  id: `r${++n}`, kind, value, value2: null, unit: null, note: null, takenAt, enteredAt: takenAt, enteredByName: "Cara Clinician",
  corrects: null, reason: null, supersededBy: null, errorAt: null, errorByName: null, errorReason: null, ...more,
});

describe("bmiOf", () => {
  it("is weight over height squared, to one decimal place, as the HQ's bmi() is", () => {
    expect(bmiOf(170, 82.5)).toBe(28.5);
    expect(bmiOf(180, 100)).toBe(30.9);
  });
  it("is undefined without a positive height and weight", () => {
    expect(bmiOf(0, 80)).toBeUndefined();
    expect(bmiOf(170, undefined)).toBeUndefined();
    expect(bmiOf(Number.NaN, 80)).toBeUndefined();
  });
});

describe("the readings that count", () => {
  const a = r("weight", 82, "2026-10-29T09:00:00Z", { supersededBy: "x" });
  const b = r("weight", 80, "2026-10-29T09:00:00Z", { corrects: a.id, reason: "Scale" });
  const c = r("weight", 70, "2026-10-29T10:00:00Z", { errorAt: "2026-10-29T10:05:00Z", errorReason: "Wrong patient" });
  it("leaves out the corrected and the entered in error", () => {
    expect(counting([a, b, c]).map((x) => x.id)).toEqual([b.id]);
  });
  it("latest is the newest measured that counts, by when it was measured", () => {
    const d = r("weight", 79, "2026-10-28T09:00:00Z"); // measured earlier, entered later
    expect(latest([a, b, c, d], "weight")?.id).toBe(b.id);
    expect(latest([a, b, c, d], "height")).toBeUndefined();
  });
});

describe("times from the API come in mixed forms (S46 verification, F-5)", () => {
  // 10:30 at +01:00 is 09:30 UTC: earlier than 10:00Z, though it sorts later as text.
  const early = r("weight", 80, "2026-10-20T10:30:00+01:00");
  const late = r("weight", 90, "2026-10-20T10:00:00Z");
  it("latest compares them as times", () => {
    expect(latest([late, early], "weight")?.id).toBe(late.id);
  });
  it("the hospital BMI says when, the later of the two as a time", () => {
    const h = r("height", 170, "2026-10-20T10:30:00+01:00");
    expect(hospitalBmi([h, late])?.at).toBe(late.takenAt);
  });
  it("a tie on the measured time goes to the later entered", () => {
    const a = r("weight", 80, "2026-10-20T10:00:00Z", { enteredAt: "2026-10-20T11:30:00+01:00" });
    const b = r("weight", 90, "2026-10-20T10:00:00Z", { enteredAt: "2026-10-20T11:00:00Z" });
    expect(latest([b, a], "weight")?.id).toBe(b.id);
  });
});

describe("hospitalBmi", () => {
  it("uses the latest height and the latest weight that count, and says when", () => {
    const h = r("height", 170, "2026-10-20T09:00:00Z");
    const w1 = r("weight", 90, "2026-10-21T09:00:00Z");
    const w2 = r("weight", 82.5, "2026-10-29T09:00:00Z");
    expect(hospitalBmi([h, w1, w2])).toEqual({ bmi: 28.5, at: "2026-10-29T09:00:00Z" });
  });
  it("follows a correction", () => {
    const h = r("height", 170, "2026-10-20T09:00:00Z");
    const w = r("weight", 90, "2026-10-21T09:00:00Z", { supersededBy: "z" });
    const fixed = r("weight", 82.5, "2026-10-21T09:00:00Z", { corrects: w.id, reason: "Typo" });
    expect(hospitalBmi([h, w, fixed])?.bmi).toBe(28.5);
  });
  it("is undefined without both", () => {
    expect(hospitalBmi([r("weight", 82, "2026-10-29T09:00:00Z")])).toBeUndefined();
    expect(hospitalBmi([])).toBeUndefined();
  });
});

describe("outsideUsual (the soft warning)", () => {
  const ranges: ObservationRanges = {
    heartRate: { softMin: 50, softMax: 120, hardMin: 20, hardMax: 300 },
    systolic: { softMin: 90, softMax: 180, hardMin: 40, hardMax: 300 },
    diastolic: { softMin: 50, softMax: 110, hardMin: 20, hardMax: 200 },
  };
  it("names each value outside its usual range, inclusive at the edges", () => {
    expect(outsideUsual(ranges, ["heartRate"], [72])).toEqual([]);
    expect(outsideUsual(ranges, ["heartRate"], [120])).toEqual([]);
    expect(outsideUsual(ranges, ["heartRate"], [121])).toEqual(["heartRate"]);
    expect(outsideUsual(ranges, ["systolic", "diastolic"], [200, 120])).toEqual(["systolic", "diastolic"]);
  });
  it("says nothing outside the hard limits: the server refuses those with its own reason", () => {
    expect(outsideUsual(ranges, ["heartRate"], [-3])).toEqual([]);
    expect(outsideUsual(ranges, ["heartRate"], [301])).toEqual([]);
    expect(outsideUsual(ranges, ["heartRate"], [300])).toEqual(["heartRate"]);
    // One value refused is enough: no warning for the other.
    expect(outsideUsual(ranges, ["systolic", "diastolic"], [200, 250])).toEqual([]);
  });
  it("says nothing about a value it can't read or a measure with no range", () => {
    expect(outsideUsual(ranges, ["heartRate"], [undefined])).toEqual([]);
    expect(outsideUsual(ranges, ["glucose"], [30])).toEqual([]);
  });
});
