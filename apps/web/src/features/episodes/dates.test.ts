import { describe, expect, it } from "vitest";
import { partlyTypedDate } from "./dates";

const input = (badInput: boolean) => ({ validity: { badInput } });

describe("partlyTypedDate", () => {
  it("names the first date typed only in part", () => {
    expect(partlyTypedDate([
      { label: "planned procedure date", input: input(false) },
      { label: "POA appointment date", input: input(true) },
    ])).toBe("Enter the POA appointment date as a date.");
  });
  it("is empty when every date is whole or empty, or not drawn yet", () => {
    expect(partlyTypedDate([{ label: "planned procedure date", input: input(false) }, { label: "POA appointment date", input: null }])).toBe("");
  });
});
