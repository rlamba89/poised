import { describe, expect, it } from "vitest";
import { addDays, ceilMinute, fromLocalInput, offsetMinutes, toLocalInput, whoRuns } from "./jobsTime";

describe("the jobs page's times", () => {
  it("writes a time for a date-and-time box, in the browser's zone, to the minute", () => {
    const d = new Date(2026, 9, 15, 9, 5, 42); // 15 Oct 2026 09:05:42 local
    expect(toLocalInput(d)).toBe("2026-10-15T09:05");
  });
  it("reads the box back as an exact time", () => {
    expect(fromLocalInput("2026-10-15T09:05")).toBe(new Date(2026, 9, 15, 9, 5).toISOString());
    expect(fromLocalInput("")).toBeNull();
    expect(fromLocalInput("not a date")).toBeNull();
  });
  it("rounds up to the next whole minute, so a preset is never early", () => {
    expect(ceilMinute(new Date(2026, 9, 15, 9, 5, 42, 10))).toEqual(new Date(2026, 9, 15, 9, 6));
    expect(ceilMinute(new Date(2026, 9, 15, 9, 59, 0, 1))).toEqual(new Date(2026, 9, 15, 10, 0));
    expect(ceilMinute(new Date(2026, 9, 15, 9, 5))).toEqual(new Date(2026, 9, 15, 9, 5));
  });
  it("adds whole days on the calendar", () => {
    const d = new Date(2026, 9, 13, 20, 30);
    expect(addDays(d, 2)).toEqual(new Date(2026, 9, 15, 20, 30));
    expect(d).toEqual(new Date(2026, 9, 13, 20, 30)); // unchanged
  });
  it("turns days and hours into minutes, refusing what the API would", () => {
    expect(offsetMinutes(2, 0)).toBe(2880);
    expect(offsetMinutes(0, 5)).toBe(300);
    expect(offsetMinutes(0, 0)).toBe(0);
    expect(offsetMinutes(-1, 0)).toBeNull();
    expect(offsetMinutes(1.5, 0)).toBeNull();
    expect(offsetMinutes(90, 1)).toBeNull(); // more than 90 days
    expect(offsetMinutes(90, 0)).toBe(129600);
  });
});

// Who runs the jobs by themselves (S17): the local server's timer, or on AWS the jobs Lambda's schedule.
describe("whoRuns", () => {
  it("says the local timer, in seconds", () => {
    expect(whoRuns("timer", 60)).toBe("the local server runs the jobs every 60 seconds");
  });
  it("says AWS's schedule, in minutes", () => {
    expect(whoRuns("schedule", 900)).toBe("a schedule on AWS runs the jobs every 15 minutes");
    expect(whoRuns("schedule", 60)).toBe("a schedule on AWS runs the jobs every minute");
  });
});
