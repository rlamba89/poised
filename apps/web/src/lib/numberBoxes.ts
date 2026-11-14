// Every number answer is a plain box (S49a, Rahul 14 Nov: "keep it a simple input box"). On a
// desktop, scrolling over a focused <input type="number"> stepped its value (a patient's 6 kg became
// 5), and numbers feed scores. SurveyJS draws every Number question as one, in every view (the
// patient's form, Fill in for the patient, Validate, the Preview), so main.tsx installs this once
// for the whole page rather than each questionnaire's JSON changing:
//   - a wheel over the focused box takes the focus off it, so the browser doesn't step the value
//     and the page scrolls as usual;
//   - the up and down arrow keys don't step it either.
// The box stays type="number" (React 19 sets an input's type again on every update, so changing it
// in the page wouldn't hold), so SurveyJS's own checks (a number, min, max) are as before.
// globals.css hides the up/down arrows.
type Target = { tagName?: string; type?: string; blur?: () => void };
type Ev = { target: unknown; key?: string; preventDefault: () => void };
type Doc = {
  activeElement: unknown;
  addEventListener: (type: "wheel" | "keydown", handler: (e: Ev) => void, options: AddEventListenerOptions) => void;
};

const isNumberBox = (t: unknown): t is Target => (t as Target | null)?.tagName === "INPUT" && (t as Target).type === "number";

export function plainNumberBoxes(doc: Doc) {
  doc.addEventListener("wheel", (e) => {
    if (isNumberBox(e.target) && doc.activeElement === e.target) e.target.blur?.();
  }, { capture: true, passive: true });
  doc.addEventListener("keydown", (e) => {
    if (isNumberBox(e.target) && (e.key === "ArrowUp" || e.key === "ArrowDown")) e.preventDefault();
  }, { capture: true });
}

/**
 * A plain number box's text as a number: digits with one decimal point or comma; nothing for an
 * empty box or anything else. For the boxes the app draws itself as text boxes (the Preview's
 * earlier answers, S49a verification 1, finding 2), which no wheel or arrow key can step.
 */
export function typedNumber(text: string): number | undefined {
  const t = text.trim();
  return /^\d+([.,]\d*)?$/.test(t) ? Number(t.replace(",", ".")) : undefined;
}
