import { beforeAll, describe, expect, it } from "vitest";
import { modelFor, registerClinicalProperties, type ChapterJson } from "@poised/clinical";
import { awaitsCheck, cardText, firstUnfilled, offersStartFresh, shownPrefilled, stillPrefilled, type PrefillEntry, type SetPrefill } from "./prefill";

beforeAll(() => registerClinicalProperties());

const vars = { name: "Sam Example", age: 40, sex: "female" };

// About you: smoking (pre-filled), the cigarettes a day only for smokers (pre-filled, hidden
// while the answer is "no"), weight (Always ask fresh, so never pre-filled), then a new question.
const set: ChapterJson = {
  pages: [
    {
      name: "p1",
      elements: [
        { type: "html", name: "intro", html: "<p>About you</p>" },
        { type: "radiogroup", name: "smoker", title: "Do you smoke?", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] },
        { type: "text", name: "per_day", title: "How many a day?", visibleIf: "{smoker} = 'yes'" },
      ],
    },
    { name: "p2", elements: [{ type: "text", name: "weight", title: "Your weight" }, { type: "text", name: "stairs", title: "Can you climb stairs?" }] },
  ],
};

const entry = (value: unknown, answeredAt = "2026-03-03T10:00:00Z"): PrefillEntry => ({ fromForm: "f1", answeredAt, value, matchedBy: "key" });
const prefill = (answers: Record<string, PrefillEntry>, choice: SetPrefill["choice"] = null): SetPrefill => ({ answers, choice, choiceAt: null });

function model(data: Record<string, unknown>) {
  const m = modelFor(set, "patient", vars);
  m.data = data;
  return m;
}

describe("stillPrefilled (the marker shows until the patient changes the answer)", () => {
  it("compares the answer with the value copied, ignoring key order", () => {
    expect(stillPrefilled(entry("yes"), "yes")).toBe(true);
    expect(stillPrefilled(entry("yes"), "no")).toBe(false);
    expect(stillPrefilled(entry([{ a: 1, b: 2 }]), [{ b: 2, a: 1 }])).toBe(true);
    expect(stillPrefilled(undefined, "yes")).toBe(false);
  });
});

describe("shownPrefilled", () => {
  const p = prefill({ smoker: entry("no"), per_day: entry(20) });
  it("counts only the questions the patient can see: a hidden pre-filled answer isn't counted…", () => {
    const m = model({ smoker: "no", per_day: 20 });
    expect(shownPrefilled(m, p).map((q) => q.name)).toEqual(["smoker"]);
  });
  it("…and isn't lost: it's there when its question shows", () => {
    const m = model({ smoker: "no", per_day: 20 });
    expect(m.data.per_day).toBe(20);
    m.setValue("smoker", "yes");
    expect(m.data.per_day).toBe(20);
    expect(shownPrefilled(m, p).map((q) => q.name)).toEqual(["per_day"]); // smoker was changed
  });
  it("nothing without pre-fill", () => {
    expect(shownPrefilled(model({}), undefined)).toEqual([]);
  });
});

describe("firstUnfilled (where Everything is still correct goes)", () => {
  it("the first question shown that wasn't pre-filled, on any page", () => {
    const m = model({ smoker: "no", per_day: 20 });
    expect(firstUnfilled(m, prefill({ smoker: entry("no") }))?.name).toBe("weight");
  });
  it("one whose pre-filled answer was changed counts as unfilled", () => {
    const m = model({ smoker: "yes" });
    expect(firstUnfilled(m, prefill({ smoker: entry("no") }))?.name).toBe("smoker");
  });
  it("none when every question shown was pre-filled", () => {
    const all = prefill({ smoker: entry("no"), weight: entry(70), stairs: entry("yes") });
    expect(firstUnfilled(model({ smoker: "no", weight: 70, stairs: "yes" }), all)).toBeNull();
  });
});

describe("awaitsCheck (the set waits for the card's answer before it counts as complete)", () => {
  const content = set;
  it("while pre-filled answers show and no button was pressed", () => {
    expect(awaitsCheck(content, { smoker: "no" }, prefill({ smoker: entry("no") }), vars)).toBe(true);
  });
  it("not once a button was pressed", () => {
    expect(awaitsCheck(content, { smoker: "no" }, prefill({ smoker: entry("no") }, "still_correct"), vars)).toBe(false);
  });
  it("not when every pre-filled answer is hidden or changed", () => {
    expect(awaitsCheck(content, { smoker: "yes", per_day: 5 }, prefill({ smoker: entry("no"), per_day: entry(20) }), vars)).toBe(false);
    expect(awaitsCheck(content, {}, undefined, vars)).toBe(false);
  });
});

describe("cardText", () => {
  const day = (iso: string) => iso.slice(0, 10);
  it("one questionnaire's date", () => {
    expect(cardText([entry("a"), entry("b", "2026-03-03T15:00:00Z")], day)).toBe(
      "We've filled in 2 answers from your questionnaire on 2026-03-03. Please check them.",
    );
    expect(cardText([entry("a")], day)).toBe("We've filled in 1 answer from your questionnaire on 2026-03-03. Please check it.");
  });
  it("several dates: the latest", () => {
    expect(cardText([entry("a", "2026-06-01T09:00:00Z"), entry("b", "2026-03-03T10:00:00Z")], day)).toBe(
      "We've filled in 2 answers from your earlier questionnaires, the latest on 2026-06-01. Please check them.",
    );
  });
  it("staff filling it in for the patient", () => {
    expect(cardText([entry("a"), entry("b")], day, "Sam")).toBe(
      "We've filled in 2 answers from Sam's questionnaire on 2026-03-03. Please check them with Sam.",
    );
  });
});

// S27 (answers 1–2): another trust's answers are copied without asking; the card names the hospital
// and offers Start fresh.
describe("another trust's answers", () => {
  const day = (iso: string) => iso.slice(0, 10);
  const theirs = (value: unknown, hospital = "Hospital B1", answeredAt = "2026-03-03T10:00:00Z"): PrefillEntry => ({
    ...entry(value, answeredAt), fromForm: "", otherTrust: true, hospital, matchedBy: "wording",
  });
  it("the card names the hospital", () => {
    expect(cardText([theirs("a"), theirs("b")], day)).toBe(
      "We've filled in 2 answers from your questionnaire at Hospital B1 on 2026-03-03. Please check them.",
    );
    expect(cardText([theirs("a")], day, "Sam")).toBe(
      "We've filled in 1 answer from Sam's questionnaire at Hospital B1 on 2026-03-03. Please check it with Sam.",
    );
  });
  it("mixed with the trust's own answers, or from several hospitals: every hospital is named", () => {
    expect(cardText([entry("a", "2026-06-01T09:00:00Z"), theirs("b")], day)).toBe(
      "We've filled in 2 answers from your earlier questionnaires, including one at Hospital B1, the latest on 2026-06-01. Please check them.",
    );
    expect(cardText([entry("a", "2026-06-01T09:00:00Z"), theirs("b"), theirs("c", "Hospital C1")], day)).toBe(
      "We've filled in 3 answers from your earlier questionnaires, including ones at Hospital B1 and Hospital C1, the latest on 2026-06-01. Please check them.",
    );
    // Every answer from those two other hospitals: no "including" (verification 2, other finding 1).
    expect(cardText([theirs("a"), theirs("b", "Hospital C1")], day)).toBe(
      "We've filled in 2 answers from your earlier questionnaires at Hospital B1 and Hospital C1, the latest on 2026-03-03. Please check them.",
    );
  });
  it("all from one other hospital, on several dates: named, with no 'including' (verification 1, finding 2)", () => {
    expect(cardText([theirs("a", "Hospital B1", "2026-06-01T09:00:00Z"), theirs("b")], day)).toBe(
      "We've filled in 2 answers from your earlier questionnaires at Hospital B1, the latest on 2026-06-01. Please check them.",
    );
  });
  it("Start fresh is offered only when an answer the card counts is another trust's, and not once a button was pressed", () => {
    // The card counts the answers shown (shownPrefilled), as cardText does.
    const shown = (p: SetPrefill, data: Record<string, unknown>) => shownPrefilled(model(data), p).map((q) => p.answers[q.name]);
    const mixed = prefill({ smoker: entry("yes"), per_day: theirs(20) });
    expect(offersStartFresh(mixed, shown(mixed, { smoker: "yes", per_day: 20 }))).toBe(true);
    const own = prefill({ smoker: entry("no") });
    expect(offersStartFresh(own, shown(own, { smoker: "no" }))).toBe(false);
    const pressed = prefill({ smoker: theirs("no") }, "check_each");
    expect(offersStartFresh(pressed, [theirs("no")])).toBe(false);
    expect(offersStartFresh(undefined, [])).toBe(false);
    // Another trust's answer on a question its condition hides: the card names no hospital, so no
    // Start fresh, which would clear this trust's own answers too (verification 1, finding 3).
    const hidden = prefill({ smoker: entry("no"), per_day: theirs(20) });
    expect(offersStartFresh(hidden, shown(hidden, { smoker: "no", per_day: 20 }))).toBe(false);
  });
  it("after Start fresh nothing counts as pre-filled, even an answer the same as the copied one", () => {
    const fresh = prefill({ smoker: theirs("no") }, "start_fresh");
    const m = model({ smoker: "no" });
    expect(shownPrefilled(m, fresh)).toEqual([]);
    expect(firstUnfilled(m, fresh)?.name).toBe("smoker");
    expect(awaitsCheck(set, { smoker: "no" }, fresh, vars)).toBe(false);
  });
});
