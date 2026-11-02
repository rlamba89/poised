import { afterEach, describe, expect, it } from "vitest";
import type { Reading } from "@poised/clinical";
import { formatDateTime } from "@/lib/format";
import { resetConfigForTests } from "@/lib/config";
import { fromLocalInput, toLocalInput } from "@/lib/localTime";
import { correctionRequest, errorLine, hospitalBmiRow, measuredAt, historyLine, newestFirst, readingRequest, unitOf, valueText, warningText, type ObservationsView } from "./observations";

const view: ObservationsView = {
  readings: [],
  maxNote: 20,
  kinds: [
    { key: "weight", label: "Weight", measures: ["weight"], note: false },
    { key: "bp", label: "Blood pressure", measures: ["systolic", "diastolic"], note: false },
    { key: "heartRate", label: "Heart rate", measures: ["heartRate"], note: false },
    { key: "glucose", label: "Blood glucose", measures: ["glucose"], note: false },
    { key: "ecg", label: "ECG", measures: [], note: true },
  ],
  measures: [
    { key: "weight", label: "Weight", unit: "kg", decimals: 1 },
    { key: "systolic", label: "Systolic", unit: "mmHg", decimals: 0 },
    { key: "diastolic", label: "Diastolic", unit: "mmHg", decimals: 0 },
    { key: "heartRate", label: "Heart rate", unit: "beats/min", decimals: 0 },
    { key: "glucose", label: "Blood glucose", unit: "mmol/L", decimals: 1 },
  ],
  ranges: {
    weight: { softMin: 35, softMax: 200, hardMin: 1, hardMax: 400 },
    systolic: { softMin: 90, softMax: 180, hardMin: 40, hardMax: 300 },
    diastolic: { softMin: 50, softMax: 110, hardMin: 20, hardMax: 200 },
    heartRate: { softMin: 50, softMax: 120, hardMin: 20, hardMax: 300 },
  },
};

const at = "2026-10-29T09:00:00Z";
const r = (over: Partial<Reading>): Reading => ({
  id: "r1", kind: "weight", value: 82.5, value2: null, unit: "kg", note: null, takenAt: at, enteredAt: at, enteredByName: "Cara Clinician",
  corrects: null, reason: null, supersededBy: null, errorAt: null, errorByName: null, errorReason: null, ...over,
});

afterEach(() => resetConfigForTests());

describe("valueText", () => {
  it("shows the value with its unit, to the measure's decimals", () => {
    expect(valueText(r({}), view)).toBe("82.5 kg");
    expect(valueText(r({ value: 80 }), view)).toBe("80.0 kg");
    expect(valueText(r({ kind: "bp", value: 128, value2: 82, unit: "mmHg" }), view)).toBe("128/82 mmHg");
    expect(valueText(r({ kind: "ecg", value: null, unit: null, note: "Sinus rhythm" }), view)).toBe("Sinus rhythm");
  });
  it("names a kind it doesn't know by its key, rather than failing", () => {
    expect(valueText(r({ kind: "new", value: 3, unit: "x" }), view)).toBe("3 x");
  });
});

describe("unitOf (A-20)", () => {
  it("is the stored unit without the country's config", () => {
    expect(unitOf("glucose", "mmol/L")).toBe("mmol/L");
  });
});

describe("the history's words", () => {
  it("says who entered it and when", () => {
    expect(historyLine(r({}))).toBe(`Measured ${formatDateTime(at)} · entered by Cara Clinician`);
  });
  it("adds the reason on a correction", () => {
    expect(historyLine(r({ corrects: "r0", reason: "Scale not zeroed" }))).toBe(`Measured ${formatDateTime(at)} · corrected by Cara Clinician: Scale not zeroed`);
  });
  it("says who marked it entered in error, when and why", () => {
    const e = r({ errorAt: "2026-10-29T10:00:00Z", errorByName: "Val Clinician", errorReason: "Wrong patient" });
    expect(errorLine(e)).toBe(`Entered in error: marked by Val Clinician on ${formatDateTime("2026-10-29T10:00:00Z")}: Wrong patient`);
  });
  it("lists the newest measured first", () => {
    const old = r({ id: "a", takenAt: "2026-10-28T09:00:00Z" });
    const now = r({ id: "b" });
    expect(newestFirst([old, now]).map((x) => x.id)).toEqual(["b", "a"]);
  });
  it("newest first compares times, not text (S46 verification, F-5)", () => {
    // 10:30 at +01:00 is 09:30 UTC: earlier than 10:00Z, though it sorts later as text.
    const early = { id: "early", takenAt: "2026-10-20T10:30:00+01:00", enteredAt: "2026-10-20T10:30:00+01:00" } as Reading;
    const late = { id: "late", takenAt: "2026-10-20T10:00:00Z", enteredAt: "2026-10-20T10:00:00Z" } as Reading;
    expect(newestFirst([early, late]).map((x) => x.id)).toEqual(["late", "early"]);
  });
});

describe("warningText (OBS-03's soft range)", () => {
  it("is empty inside the usual range", () => {
    expect(warningText(view, "heartRate", [72])).toBe("");
  });
  it("names the value and the usual range outside it", () => {
    expect(warningText(view, "heartRate", [220])).toBe("Heart rate 220 beats/min is outside the usual range (50 to 120).");
    expect(warningText(view, "bp", [200, 82])).toBe("Systolic 200 mmHg is outside the usual range (90 to 180).");
  });
});

describe("warningText for a note (S46 test report 1, F-1)", () => {
  it("is empty, even if the kind's measures came as null", () => {
    const old = { ...view, kinds: view.kinds.map((k) => (k.note ? { ...k, measures: null as unknown as string[] } : k)) };
    expect(warningText(old, "ecg", [])).toBe("");
  });
});

describe("readingRequest", () => {
  const when = "2026-10-29T09:30";
  it("turns the form into the API's body, in the browser's time zone", () => {
    expect(readingRequest(view, "weight", ["82.5"], "", when)).toEqual({ body: { kind: "weight", value: 82.5, takenAt: fromLocalInput(when) } });
    expect(readingRequest(view, "bp", ["128", "82"], "", when)).toEqual({ body: { kind: "bp", value: 128, value2: 82, takenAt: fromLocalInput(when) } });
    expect(readingRequest(view, "ecg", [], "  Sinus rhythm ", when)).toEqual({ body: { kind: "ecg", note: "Sinus rhythm", takenAt: fromLocalInput(when) } });
  });
  it("refuses a time that isn't one, rather than failing with no message (S46 verification 2, F-2)", () => {
    const refused = { error: "When was it measured? Enter the date and time." };
    expect(readingRequest(view, "weight", ["80"], "", "20266-10-31T15:52")).toEqual(refused);
    expect(readingRequest(view, "weight", ["80"], "", "2026-13-45T15:52")).toEqual(refused);
    expect(readingRequest(view, "weight", ["80"], "", "")).toEqual(refused);
  });
  it("says a note is too long, rather than cutting it (S46 test report 4)", () => {
    const refused = { error: "The note is too long (20 characters at most)." };
    expect(readingRequest(view, "ecg", [], "x".repeat(21), when)).toEqual(refused);
    expect(correctionRequest(view, "ecg", [], "x".repeat(21), "Typo")).toEqual(refused);
    expect(readingRequest(view, "ecg", [], "😀".repeat(20), when)).toHaveProperty("body"); // characters, as Go counts them
  });
  it("asks for what's missing before sending anything", () => {
    expect(readingRequest(view, "", [], "", when)).toEqual({ error: "Choose what was measured." });
    expect(readingRequest(view, "weight", [""], "", when)).toEqual({ error: "Enter the weight." });
    expect(readingRequest(view, "bp", ["128", ""], "", when)).toEqual({ error: "Enter the diastolic." });
    expect(readingRequest(view, "weight", ["eighty"], "", when)).toEqual({ error: "Weight: enter a number." });
    expect(readingRequest(view, "ecg", [], "  ", when)).toEqual({ error: "Enter the ECG note." });
    expect(readingRequest(view, "weight", ["80"], "", "")).toEqual({ error: "When was it measured? Enter the date and time." });
  });
  it("refuses a diastolic not below the systolic before any warning (F-2)", () => {
    expect(readingRequest(view, "bp", ["80", "120"], "", when)).toEqual({ error: "The diastolic must be lower than the systolic." });
    expect(readingRequest(view, "bp", ["90", "90"], "", when)).toEqual({ error: "The diastolic must be lower than the systolic." });
    expect(correctionRequest(view, "bp", ["80", "120"], "", "x")).toEqual({ error: "The diastolic must be lower than the systolic." });
  });
  it("reads only plain decimal numbers: not 1e2 or 0x10", () => {
    expect(readingRequest(view, "heartRate", ["1e2"], "", when)).toEqual({ error: "Heart rate: enter a number." });
    expect(readingRequest(view, "heartRate", ["0x10"], "", when)).toEqual({ error: "Heart rate: enter a number." });
    expect(readingRequest(view, "heartRate", [" 75 "], "", when)).toEqual({ body: { kind: "heartRate", value: 75, takenAt: fromLocalInput(when) } });
  });
  it("accepts a comma as the decimal mark", () => {
    expect(readingRequest(view, "weight", ["82,5"], "", when)).toEqual({ body: { kind: "weight", value: 82.5, takenAt: fromLocalInput(when) } });
  });
});

describe("correctionRequest", () => {
  it("keeps the reading's kind and needs a reason", () => {
    expect(correctionRequest(view, "weight", ["80.5"], "", "Scale not zeroed")).toEqual({ body: { value: 80.5, reason: "Scale not zeroed" } });
    expect(correctionRequest(view, "weight", ["80.5"], "", "  ")).toEqual({ error: "Say why the reading is corrected." });
    expect(correctionRequest(view, "weight", [""], "", "x")).toEqual({ error: "Enter the weight." });
  });
});

describe("the date-time box", () => {
  it("round-trips a time in the browser's own zone, to the minute", () => {
    const d = new Date(2026, 9, 29, 9, 30);
    expect(toLocalInput(d)).toBe("2026-10-29T09:30");
    expect(new Date(fromLocalInput("2026-10-29T09:30") ?? "").getTime()).toBe(d.getTime());
  });
});

describe("measuredAt (test report 1: the time could lag the save)", () => {
  const now = new Date(2026, 9, 29, 17, 20);
  it("is the time of Save while the box hasn't been changed", () => {
    expect(measuredAt(false, "2026-10-29T17:05", now)).toBe("2026-10-29T17:20");
  });
  it("is what the clinician typed once they changed it", () => {
    expect(measuredAt(true, "2026-10-28T08:00", now)).toBe("2026-10-28T08:00");
  });
});

describe("the Hospital BMI row on the POA Summary (S46 verification 2, F-3)", () => {
  const readings = [
    { kind: "height", value: 180, takenAt: "2026-10-29T09:00:00Z", enteredAt: "2026-10-29T09:00:00Z" },
    { kind: "weight", value: 100, takenAt: "2026-10-29T09:00:00Z", enteredAt: "2026-10-29T09:00:00Z" },
  ] as Reading[];
  it("shows the BMI once it's loaded", () => {
    expect(hospitalBmiRow({ ...view, readings }, "")).toContain("30.9");
  });
  it("says so when the readings couldn't be loaded, so it doesn't read as observations being off", () => {
    expect(hospitalBmiRow(null, "We couldn't reach the server.")).toBe("Couldn't be loaded");
  });
  it("has no row with observations off, or while loading", () => {
    expect(hospitalBmiRow("off", "")).toBeNull();
    expect(hospitalBmiRow(null, "")).toBeNull();
  });
  it("keeps the readings it has if a later load fails", () => {
    expect(hospitalBmiRow({ ...view, readings }, "We couldn't reach the server.")).toContain("30.9");
  });
});
