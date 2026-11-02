import { describe, expect, it } from "vitest";
import { fromLocalInput, toLocalInput } from "./localTime";

describe("a date-and-time box's times", () => {
  it("writes a time for the box, in the browser's zone, to the minute", () => {
    const d = new Date(2026, 9, 15, 9, 5, 42); // 15 Oct 2026 09:05:42 local
    expect(toLocalInput(d)).toBe("2026-10-15T09:05");
  });
  it("reads the box back as an exact time", () => {
    expect(fromLocalInput("2026-10-15T09:05")).toBe(new Date(2026, 9, 15, 9, 5).toISOString());
    expect(fromLocalInput("")).toBeNull();
    expect(fromLocalInput("not a date")).toBeNull();
  });
  it("gives null for what a box can hold but no date can be", () => {
    expect(fromLocalInput("20266-10-31T15:52")).toBeNull(); // a mistyped five-digit year
    expect(fromLocalInput("2026-13-45T15:52")).toBeNull();
  });
});
