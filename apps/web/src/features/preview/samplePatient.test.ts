import { describe, expect, it } from "vitest";
import { SAMPLE_SEXES } from "./samplePatient";
import { SEXES } from "@/features/episodes/types";

// S44c test report 1, F1: an author must be able to try every sex a real patient can have (e.g.
// "Other", which makes STOP-Bang Incomplete) in the Preview and Test the whole HQ.
describe("the sample patient's sex", () => {
  it("offers every sex a patient record can have", () => {
    expect(SAMPLE_SEXES.map((s) => s.value)).toEqual(["female", "male", "other", "unknown"]);
    expect(SAMPLE_SEXES).toEqual(SEXES);
  });
});
