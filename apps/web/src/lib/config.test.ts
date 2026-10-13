import { afterEach, describe, expect, it, vi } from "vitest";
import { loadConfig, locale, nationalIdLabel, resetConfigForTests } from "./config";

function serve(status: number, body: unknown) {
  const fetch = vi.fn(async () => new Response(JSON.stringify(body), { status }));
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

afterEach(() => {
  vi.unstubAllGlobals();
  resetConfigForTests();
});

describe("loadConfig", () => {
  it("reads GET /api/config once and keeps the locale", async () => {
    const fetch = serve(200, { locale: "en-CA", timeZone: "America/Toronto" });
    expect(await loadConfig()).toEqual({ locale: "en-CA", timeZone: "America/Toronto" });
    expect(fetch).toHaveBeenCalledOnce();
    expect(fetch.mock.calls[0]).toContain("/api/config");
    expect(locale()).toBe("en-CA");
  });

  it("has no locale of its own before it's loaded: the browser's default is used", () => {
    expect(locale()).toBeUndefined();
  });

  it("fails loudly when the API doesn't answer, and keeps no locale", async () => {
    serve(503, { error: "The database is not reachable." });
    await expect(loadConfig()).rejects.toThrow("The database is not reachable.");
    expect(locale()).toBeUndefined();
  });

  it("names the deployment's patient identifier (S13), with a neutral name until it's loaded", async () => {
    expect(nationalIdLabel()).toBe("National ID");
    serve(200, { locale: "en-CA", timeZone: "America/Toronto", nationalId: { type: "nhs", label: "NHS number" } });
    await loadConfig();
    expect(nationalIdLabel()).toBe("NHS number");
  });
});
