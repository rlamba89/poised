import { afterEach, describe, expect, it, vi } from "vitest";
import { loadConfig, resetConfigForTests } from "./config";
import { formatDate, formatDateTime, formatOptionalDate, formatShortDate } from "./format";

// The dates follow the deployment's locale from GET /api/config (A-20), not a hardcoded one.
async function withLocale(locale: string) {
  vi.stubGlobal("fetch", async () => new Response(JSON.stringify({ locale, timeZone: "UTC" })));
  await loadConfig();
}

afterEach(() => {
  vi.unstubAllGlobals();
  resetConfigForTests();
});

// Midday UTC is the same calendar day in every time zone from UTC-11 to UTC+11, so these
// don't depend on the machine's zone.
const noon = "2026-10-12T12:00:00Z";

describe("formatDate (a date of birth)", () => {
  it("in en-GB", async () => {
    await withLocale("en-GB");
    expect(formatDate("1980-05-17")).toBe("17 May 1980");
  });
  it("in en-US", async () => {
    await withLocale("en-US");
    expect(formatDate("1980-05-17")).toBe("May 17, 1980");
  });
  it("never moves the day, whatever the locale's zone", async () => {
    await withLocale("en-CA");
    expect(formatDate("1980-01-01")).toBe("January 1, 1980");
  });
});

describe("formatDateTime", () => {
  it("in en-GB: day first", async () => {
    await withLocale("en-GB");
    expect(formatDateTime(noon)).toMatch(/^12\/10\/2026, \d\d:\d\d$/);
  });
  it("in en-US: month first", async () => {
    await withLocale("en-US");
    expect(formatDateTime(noon)).toMatch(/^10\/12\/26, \d{1,2}:\d\d\s?[AP]M$/);
  });
});

describe("formatShortDate", () => {
  it("in en-GB", async () => {
    await withLocale("en-GB");
    expect(formatShortDate(noon)).toBe("12/10/2026");
  });
  it("in en-US", async () => {
    await withLocale("en-US");
    expect(formatShortDate(noon)).toBe("10/12/2026");
  });
});

describe("formatOptionalDate (an episode's planned and POA dates, S19)", () => {
  it("in the deployment's locale, never moving the day", async () => {
    await withLocale("en-GB");
    expect(formatOptionalDate("2026-11-03")).toBe("03/11/2026");
    await withLocale("en-US");
    expect(formatOptionalDate("2026-11-03")).toBe("11/3/26");
  });
  it("is a dash for none", async () => {
    await withLocale("en-GB");
    expect(formatOptionalDate(null)).toBe("—");
    expect(formatOptionalDate("")).toBe("—");
  });
});
