import { describe, expect, it, vi } from "vitest";
import { plainNumberBoxes, typedNumber } from "./numberBoxes";

// S49a (Rahul, 14 Nov): a number answer is a plain box. Scrolling over a focused one never changes
// its value (a patient's 6 kg became 5), and neither do the arrow keys.
describe("plainNumberBoxes", () => {
  type Handler = (e: { target: unknown; key?: string; preventDefault: () => void }) => void;
  function fakeDocument() {
    const handlers: Record<string, Handler[]> = {};
    const doc = {
      activeElement: null as unknown,
      addEventListener: (type: string, h: Handler) => (handlers[type] ??= []).push(h),
    };
    const fire = (type: string, target: unknown, key?: string) => {
      const e = { target, key, preventDefault: vi.fn() };
      handlers[type]?.forEach((h) => h(e));
      return e;
    };
    return { doc, fire };
  }
  const box = (type: string) => ({ tagName: "INPUT", type, blur: vi.fn() });

  it("a wheel over the focused number box takes the focus off it, so the browser doesn't step its value", () => {
    const { doc, fire } = fakeDocument();
    plainNumberBoxes(doc);
    const weight = box("number");
    doc.activeElement = weight;
    const e = fire("wheel", weight);
    expect(weight.blur).toHaveBeenCalledOnce();
    expect(e.preventDefault).not.toHaveBeenCalled(); // the page still scrolls
  });

  it("leaves a number box without the focus, and other boxes, alone", () => {
    const { doc, fire } = fakeDocument();
    plainNumberBoxes(doc);
    const weight = box("number");
    const name = box("text");
    doc.activeElement = name;
    fire("wheel", weight);
    fire("wheel", name);
    expect(weight.blur).not.toHaveBeenCalled();
    expect(name.blur).not.toHaveBeenCalled();
  });

  it("the up and down arrow keys don't change a number box; other keys and boxes are as before", () => {
    const { doc, fire } = fakeDocument();
    plainNumberBoxes(doc);
    const weight = box("number");
    expect(fire("keydown", weight, "ArrowUp").preventDefault).toHaveBeenCalledOnce();
    expect(fire("keydown", weight, "ArrowDown").preventDefault).toHaveBeenCalledOnce();
    expect(fire("keydown", weight, "5").preventDefault).not.toHaveBeenCalled();
    expect(fire("keydown", box("text"), "ArrowUp").preventDefault).not.toHaveBeenCalled();
  });
});

describe("typedNumber: a plain number box's text (S49a verification 1, finding 2)", () => {
  it("reads digits with one decimal point or comma", () => {
    expect(typedNumber("70")).toBe(70);
    expect(typedNumber(" 70.5 ")).toBe(70.5);
    expect(typedNumber("20,8")).toBe(20.8);
    expect(typedNumber("20.")).toBe(20); // halfway through typing 20.8: no error yet
  });
  it("gives nothing for an empty box or anything else", () => {
    for (const t of ["", "  ", "abc", ".5", "1e3", "-5", "1.2.3"]) expect(typedNumber(t)).toBeUndefined();
  });
});
