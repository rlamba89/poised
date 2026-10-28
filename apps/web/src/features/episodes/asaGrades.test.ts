import { describe, expect, it } from "vitest";
import { formatDateTime } from "@/lib/format";
import { addGradeRequest, asaLabel, needsReason, countingGrade, errorLine, givenLine, newestFirst, type AnaesthetistGrade } from "./asaGrades";

const g = (over: Partial<AnaesthetistGrade>): AnaesthetistGrade => ({
  id: "g1", grade: 3, givenAt: "2026-10-27T09:00:00Z", givenByName: "Cara Clinician", reason: null,
  errorAt: null, errorByName: null, errorReason: null, ...over,
});

// S44b: the anaesthetist's ASA grade is a history; the one that counts is the latest not entered in error.
describe("countingGrade", () => {
  it("is null with no grades, or every grade entered in error", () => {
    expect(countingGrade([])).toBeNull();
    expect(countingGrade([g({ errorAt: "2026-10-27T10:00:00Z" })])).toBeNull();
  });
  it("is the latest grade, or the one before when the latest was entered in error", () => {
    const first = g({ id: "a", grade: 3 });
    const second = g({ id: "b", grade: 4, givenAt: "2026-10-27T10:00:00Z", reason: "New echo result" });
    expect(countingGrade([first, second])?.id).toBe("b");
    expect(countingGrade([first, { ...second, errorAt: "2026-10-27T11:00:00Z" }])?.id).toBe("a");
  });
});

describe("the history's words", () => {
  it("lists the newest first", () => {
    expect(newestFirst([g({ id: "a" }), g({ id: "b" })]).map((x) => x.id)).toEqual(["b", "a"]);
  });
  it("says who gave a grade and when, and why for a re-grade", () => {
    expect(givenLine(g({}))).toBe(`Given by Cara Clinician on ${formatDateTime("2026-10-27T09:00:00Z")}`);
    expect(givenLine(g({ reason: "New echo result" }))).toBe(`Given by Cara Clinician on ${formatDateTime("2026-10-27T09:00:00Z")}: New echo result`);
  });
  it("says a grade carried over from before the history was recorded then", () => {
    expect(givenLine(g({ givenByName: null }))).toBe(`Recorded before the history (${formatDateTime("2026-10-27T09:00:00Z")})`);
  });
  it("says who marked a grade entered in error, when and why", () => {
    expect(errorLine(g({}))).toBe("");
    expect(errorLine(g({ errorAt: "2026-10-27T11:00:00Z", errorByName: "Val Clinician", errorReason: "Wrong patient" })))
      .toBe(`Entered in error: marked by Val Clinician on ${formatDateTime("2026-10-27T11:00:00Z")}: Wrong patient`);
  });
  it("names grades as on the rest of the page", () => {
    expect([asaLabel(1), asaLabel(6), asaLabel(null)]).toEqual(["ASA I", "ASA VI", "Not recorded"]);
  });
});

describe("addGradeRequest", () => {
  it("needs a grade", () => {
    expect(addGradeRequest(null, "", false, 4)).toEqual({ error: "Choose an ASA grade." });
  });
  it("sends the first grade with no reason", () => {
    expect(addGradeRequest("3", "  ", false, 4)).toEqual({ body: { grade: 3, reason: "", revision: 4 } });
  });
  it("needs a reason for every grade after the episode's first, even after an error (verification 1)", () => {
    expect(needsReason([])).toBe(false);
    expect(needsReason([g({})])).toBe(true);
    expect(needsReason([g({ errorAt: "2026-10-27T10:00:00Z" })])).toBe(true);
  });
  it("needs a reason once a grade exists", () => {
    expect(addGradeRequest("4", "  ", true, 5)).toEqual({ error: "Say why the grade is changed." });
    expect(addGradeRequest("4", " New echo result ", true, 5)).toEqual({ body: { grade: 4, reason: "New echo result", revision: 5 } });
  });
});
