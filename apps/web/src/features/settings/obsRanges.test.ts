import { describe, expect, it } from "vitest";
import { rangesDraft, rangesForSave, sameRanges, withLimit } from "./obsRanges";

const platform = {
  weight: { softMin: 35, softMax: 200, hardMin: 1, hardMax: 400 },
  heartRate: { softMin: 50, softMax: 120, hardMin: 20, hardMax: 300 },
};

describe("the observation ranges editor (S46)", () => {
  it("starts from the value, filling any measure it lacks from Preopsis's defaults", () => {
    const own = { heartRate: { softMin: 50, softMax: 100, hardMin: 30, hardMax: 150 } };
    expect(rangesDraft(own, platform)).toEqual({ heartRate: own.heartRate, weight: platform.weight });
  });
  it("keeps Go's order (measures sorted, then soft and hard), so an unchanged table compares the same", () => {
    const shuffled = { weight: { hardMax: 400, hardMin: 1, softMax: 200, softMin: 35 }, heartRate: platform.heartRate };
    expect(JSON.stringify(rangesDraft(shuffled, platform))).toBe(JSON.stringify(rangesDraft(platform, platform)));
    expect(Object.keys(rangesDraft(platform, platform))).toEqual(["heartRate", "weight"]);
  });
  it("changes one limit, leaving the rest", () => {
    const next = withLimit(rangesDraft(platform, platform), "heartRate", "hardMax", 250);
    expect(next.heartRate).toEqual({ softMin: 50, softMax: 120, hardMin: 20, hardMax: 250 });
    expect(next.weight).toEqual(platform.weight);
  });
  it("keeps what's typed even if it isn't a number yet, for the API to refuse", () => {
    expect(withLimit(rangesDraft(platform, platform), "weight", "softMin", "")).toHaveProperty("weight.softMin", "");
  });
});

describe("rangesForSave (S46 test report 1, F-3)", () => {
  it("sends what was typed as numbers, the minus sign kept, so Go can refuse -1", () => {
    const typed = { heartRate: { softMin: "50", softMax: "100", hardMin: "-1", hardMax: 150 }, weight: { softMin: "35,5", softMax: 200, hardMin: 1, hardMax: 400 } };
    expect(rangesForSave(typed, platform)).toEqual({
      heartRate: { softMin: 50, softMax: 100, hardMin: -1, hardMax: 150 },
      weight: { softMin: 35.5, softMax: 200, hardMin: 1, hardMax: 400 },
    });
  });
  it("sends anything else as typed, for Go's message", () => {
    expect(rangesForSave({ heartRate: { softMin: "abc", softMax: "", hardMin: "1e2", hardMax: 150 } }, platform)).toEqual({
      heartRate: { softMin: "abc", softMax: "", hardMin: "1e2", hardMax: 150 },
    });
  });
  it("sends only the measures that differ from Preopsis's, so the rest follow its defaults (S46 verification)", () => {
    const typed = rangesDraft(platform, platform);
    expect(rangesForSave(withLimit(typed, "heartRate", "hardMax", "250"), platform)).toEqual({
      heartRate: { softMin: 50, softMax: 120, hardMin: 20, hardMax: 250 },
    });
  });
  it("a value typed the same as the default counts as the same", () => {
    expect(rangesForSave(withLimit(rangesDraft(platform, platform), "heartRate", "hardMax", "300"), platform)).toEqual({});
  });
});

describe("sameRanges (S46 verification 2, F-1)", () => {
  // As the API returns a stored value: jsonb orders an object's keys by length, then alphabetically.
  const stored = { heartRate: { hardMax: 150, hardMin: 30, softMax: 100, softMin: 50 } };
  const draft = () => rangesDraft(stored, platform);

  it("an untouched table is the same as what's stored, whatever order the keys come in", () => {
    expect(sameRanges(draft(), stored, platform)).toBe(true);
  });
  it("a limit typed the same as the stored one is the same", () => {
    expect(sameRanges(withLimit(draft(), "heartRate", "softMax", "100"), stored, platform)).toBe(true);
  });
  it("a changed limit isn't the same", () => {
    expect(sameRanges(withLimit(draft(), "heartRate", "softMax", "105"), stored, platform)).toBe(false);
    expect(sameRanges(withLimit(draft(), "weight", "hardMax", 390), stored, platform)).toBe(false);
  });
  it("a trust that saved all ten before is the same as its table, though Save now sends fewer", () => {
    const all = { ...platform, ...stored };
    expect(sameRanges(rangesDraft(all, platform), all, platform)).toBe(true);
  });
});
