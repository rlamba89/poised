import { describe, expect, it } from "vitest";
import { ApiError } from "@/lib/api";
import { conflictOf, hospitalNumberOnChoose, latestHospitalNumber, searchQuery } from "./newEpisode";

// The New episode page (S13) searches only once a term is precise enough, as the API requires
// (patient.ParseSearch in Go): 3 characters, or 1 with a date of birth. Nobody can list everyone.
describe("searchQuery", () => {
  it("is null until the term is precise enough", () => {
    expect(searchQuery("", "")).toBeNull();
    expect(searchQuery("  sm ", "")).toBeNull();
    expect(searchQuery("", "1980-05-17")).toBeNull();
  });
  it("sends the trimmed term, and the date of birth when there is one", () => {
    expect(searchQuery(" smi ", "")).toBe("q=smi");
    expect(searchQuery("Li", "1980-05-17")).toBe("q=Li&dob=1980-05-17");
    expect(searchQuery("943 476 5919", "")).toBe("q=943+476+5919");
  });
});

describe("conflictOf", () => {
  it("reads the API's 409 for typed details", () => {
    const matches = [{ id: "p1", firstName: "Ada", lastName: "Lovelock", dateOfBirth: "1990-01-01", nationalId: null, hospitalNumber: "" }];
    expect(conflictOf(new ApiError(409, "may already be registered", { code: "possible_duplicate", error: "may already be registered", matches })))
      .toEqual({ code: "possible_duplicate", message: "may already be registered", matches });
    expect(conflictOf(new ApiError(409, "don't match", { code: "mismatch", error: "don't match" })))
      .toEqual({ code: "mismatch", message: "don't match", matches: [] });
  });
  it("is null for any other error, which the page shows as it is", () => {
    expect(conflictOf(new ApiError(400, "Enter a valid date of birth.", { error: "Enter a valid date of birth." }))).toBeNull();
    expect(conflictOf(new ApiError(409, "A record with this NHS number was added just now.", { error: "x" }))).toBeNull();
    expect(conflictOf(new Error("offline"))).toBeNull();
  });
});

// The Episode card's hospital number belongs to the chosen patient (S13 test report 1): choosing
// one fills in their number here (unless staff typed one), and Change patient clears it, so a
// new patient's episode never carries the previous patient's number.
describe("the hospital number when the patient changes", () => {
  it("fills in the chosen patient's number, keeping one staff typed", () => {
    expect(hospitalNumberOnChoose("", "A1-7")).toBe("A1-7");
    expect(hospitalNumberOnChoose("NEW-1", "A1-7")).toBe("NEW-1");
    expect(hospitalNumberOnChoose("", "")).toBe("");
  });
  it("from a patient's page: their newest episode's number here", () => {
    expect(latestHospitalNumber([{ hospitalNumber: "" }, { hospitalNumber: "A1-9" }, { hospitalNumber: "A1-1" }])).toBe("A1-9");
    expect(latestHospitalNumber([{ hospitalNumber: "" }])).toBe("");
    expect(latestHospitalNumber([])).toBe("");
  });
});
