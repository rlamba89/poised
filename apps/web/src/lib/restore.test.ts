import { describe, expect, it, vi } from "vitest";
import { reloadWhenRestored } from "./restore";

// A page the browser restores from its back/forward cache shows what it showed before (e.g. a
// signed-out clinician's patient list). Reloading it asks the API again, which signs them out.
describe("reloadWhenRestored", () => {
  function fakeWindow() {
    const handlers: ((e: { persisted: boolean }) => void)[] = [];
    return {
      reload: vi.fn(),
      win: {
        addEventListener: (type: string, h: (e: { persisted: boolean }) => void) => {
          if (type === "pageshow") handlers.push(h);
        },
        location: { reload: vi.fn() },
      },
      fire: (persisted: boolean) => handlers.forEach((h) => h({ persisted })),
    };
  }

  it("reloads a page restored from the back/forward cache", () => {
    const w = fakeWindow();
    reloadWhenRestored(w.win);
    w.fire(true);
    expect(w.win.location.reload).toHaveBeenCalledOnce();
  });

  it("leaves a normal page load alone", () => {
    const w = fakeWindow();
    reloadWhenRestored(w.win);
    w.fire(false);
    expect(w.win.location.reload).not.toHaveBeenCalled();
  });
});
