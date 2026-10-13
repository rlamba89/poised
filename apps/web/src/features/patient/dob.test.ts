import { describe, expect, it } from "vitest";
import { dobValue } from "./dob";

describe("dobValue", () => {
  it("turns day, month and year into the API's YYYY-MM-DD", () => {
    expect(dobValue("27", "3", "1985")).toBe("1985-03-27");
    expect(dobValue(" 01 ", "12", "2001")).toBe("2001-12-01");
  });
  it("refuses anything that isn't a whole date", () => {
    for (const [d, m, y] of [["", "3", "1985"], ["27", "", "1985"], ["27", "3", ""], ["27", "3", "85"], ["x", "3", "1985"], ["32", "1", "1985"], ["1", "13", "1985"], ["0", "1", "1985"], ["31", "2", "1985"], ["1.5", "2", "1985"]]) {
      expect(dobValue(d, m, y), `${d}/${m}/${y}`).toBeNull();
    }
  });
  it("knows leap years", () => {
    expect(dobValue("29", "2", "2000")).toBe("2000-02-29");
    expect(dobValue("29", "2", "1900")).toBeNull();
  });
});
