import { describe, expect, it, vi } from "vitest";
import { isChunkLoadError, reloadOnce } from "./chunkError";

describe("isChunkLoadError", () => {
  it.each([
    ["Chrome", "Failed to fetch dynamically imported module: http://x/assets/Editor-abc.js"],
    ["Firefox", "error loading dynamically imported module: http://x/assets/Editor-abc.js"],
    ["Safari", "Importing a module script failed."],
    ["Vite CSS preload", "Unable to preload CSS for /assets/units-abc.css"],
  ])("recognises %s's message", (_, message) => {
    expect(isChunkLoadError(new TypeError(message))).toBe(true);
  });

  it.each([new Error("Cannot read properties of undefined"), "Failed to fetch dynamically imported module", null, undefined, { status: 404 }])(
    "ignores anything else: %s",
    (error) => {
      expect(isChunkLoadError(error)).toBe(false);
    },
  );
});

function memoryStorage() {
  const m = new Map<string, string>();
  return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v) };
}

describe("reloadOnce", () => {
  it("reloads the first time", () => {
    const reload = vi.fn();
    expect(reloadOnce(memoryStorage(), reload, 1_000_000)).toBe(true);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("doesn't reload again within 10 seconds, so a chunk that's really gone can't loop", () => {
    const storage = memoryStorage();
    const reload = vi.fn();
    reloadOnce(storage, reload, 1_000_000);
    expect(reloadOnce(storage, reload, 1_009_999)).toBe(false);
    expect(reload).toHaveBeenCalledOnce();
  });

  it("reloads again after 10 seconds (e.g. the next deploy)", () => {
    const storage = memoryStorage();
    const reload = vi.fn();
    reloadOnce(storage, reload, 1_000_000);
    expect(reloadOnce(storage, reload, 1_010_001)).toBe(true);
    expect(reload).toHaveBeenCalledTimes(2);
  });

  it("doesn't reload when storage can't be used, because it couldn't stop a loop", () => {
    const broken = {
      getItem: () => {
        throw new Error("SecurityError");
      },
      setItem: () => {
        throw new Error("SecurityError");
      },
    };
    const reload = vi.fn();
    expect(reloadOnce(broken, reload, 1_000_000)).toBe(false);
    expect(reloadOnce(null, reload, 1_000_000)).toBe(false);
    expect(reload).not.toHaveBeenCalled();
  });
});
