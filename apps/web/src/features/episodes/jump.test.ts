import { afterEach, describe, expect, it, vi } from "vitest";
import { questionOf, restoreSpot, saveSpot, takeSpot, validateAt } from "./jump";

// S27 (answers 5): the POA Summary's answers open Validate at their question, and Back returns to the same place.
describe("validateAt and questionOf", () => {
  it("names the Question Set and the question, and reads it back", () => {
    const link = validateAt("/o/t/h/h/episodes/e1", "set1", "q smokes/1");
    expect(link).toBe("/o/t/h/h/episodes/e1/sets/set1?q=q%20smokes%2F1");
    expect(questionOf(link.slice(link.indexOf("?")))).toBe("q smokes/1");
    expect(questionOf("")).toBe("");
  });
});

describe("saveSpot and takeSpot (where the clinician was on the POA Summary)", () => {
  const memory = () => {
    const m = new Map<string, string>();
    return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), removeItem: (k: string) => void m.delete(k) };
  };
  it("kept per episode, and taken once", () => {
    const store = memory();
    saveSpot(store, "e1", { anchor: "set1:note:q1", top: 240 });
    expect(takeSpot(store, "e2")).toBeNull();
    expect(takeSpot(store, "e1")).toEqual({ anchor: "set1:note:q1", top: 240 });
    expect(takeSpot(store, "e1")).toBeNull();
  });
  it("no storage, or storage that throws or holds rubbish: nothing, and no error", () => {
    expect(() => saveSpot(null, "e1", { anchor: "a", top: 1 })).not.toThrow();
    expect(takeSpot(null, "e1")).toBeNull();
    const broken = { getItem: () => { throw new Error("blocked"); }, setItem: () => { throw new Error("blocked"); }, removeItem: () => {} };
    expect(() => saveSpot(broken, "e1", { anchor: "a", top: 1 })).not.toThrow();
    expect(takeSpot(broken, "e1")).toBeNull();
    const store = memory();
    store.setItem("poa-spot:e1", "{not json");
    expect(takeSpot(store, "e1")).toBeNull();
  });
});

// S28 (S27 verification 3): leaving the POA Summary stops the way back trying to scroll.
describe("restoreSpot", () => {
  afterEach(() => vi.unstubAllGlobals());
  // A screen where the answer isn't drawn yet: each try asks for another frame, run by hand here.
  const screen = (drawn: () => boolean) => {
    const frames = new Map<number, () => void>();
    let next = 0;
    const scrollBy = vi.fn();
    vi.stubGlobal("requestAnimationFrame", (cb: () => void) => (frames.set(++next, cb), next));
    vi.stubGlobal("cancelAnimationFrame", (id: number) => frames.delete(id));
    vi.stubGlobal("CSS", { escape: (s: string) => s });
    vi.stubGlobal("document", { querySelector: () => (drawn() ? { getBoundingClientRect: () => ({ top: 500 }) } : null) });
    vi.stubGlobal("window", { scrollBy });
    const run = () => {
      const due = [...frames.entries()];
      frames.clear();
      for (const [, cb] of due) cb();
    };
    return { frames, run, scrollBy };
  };
  it("scrolls the answer back to where it was, once it's drawn", () => {
    let drawn = false;
    const s = screen(() => drawn);
    restoreSpot({ anchor: "a", top: 200 });
    s.run();
    s.run();
    drawn = true;
    s.run();
    expect(s.scrollBy).toHaveBeenCalledWith(0, 300);
    expect(s.frames.size).toBe(0);
  });
  it("stops trying once cleaned up, so the next page isn't scrolled", () => {
    let drawn = false;
    const s = screen(() => drawn);
    const stop = restoreSpot({ anchor: "a", top: 200 });
    s.run();
    stop();
    expect(s.frames.size).toBe(0);
    drawn = true;
    s.run();
    expect(s.scrollBy).not.toHaveBeenCalled();
  });
});
