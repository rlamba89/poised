import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import { findElement, kindOf, optionsOf, updateElement, type ChapterJson, type ElementJson } from "./doc";
import {
  convertibleKinds, convertKind, dateSettingsOf, numberSettingsOf, setDateSettings, setNumberSettings, setRepeating, setSpecialOption,
  setTextFormat, textFormatOf, setChoiceDisplay, choiceDisplayOf,
} from "./fields";
import { registerClinicalProperties } from "./properties";

registerClinicalProperties();

const one = (el: ElementJson, ...more: ElementJson[]): ChapterJson => ({ pages: [{ name: "p_a", elements: [el, ...more] }] });
const el = (doc: ChapterJson, name: string) => findElement(doc, name)!.el;
const errorsFor = (json: ChapterJson, name: string, value: unknown) => {
  const m = new Model(json);
  m.setValue(name, value);
  m.validate();
  return m.getQuestionByName(name).errors.map((e) => `${e.notificationType ?? "error"}: ${e.getText()}`);
};

describe("text format (QT-04)", () => {
  it("writes a validator and reads the format back from it", () => {
    const q: ElementJson = { type: "text", name: "q_nhs", validators: [{ type: "text", maxLength: 20 }] };
    const patch = setTextFormat(q, "nhs");
    const next = { ...q, ...patch };
    expect(textFormatOf(next)).toEqual({ format: "nhs", message: "Please enter a 10-digit NHS number." });
    expect((next.validators as unknown[]).length).toBe(2); // the other validator is kept
    expect(errorsFor(one(next), "q_nhs", "12345")).toContain("error: Please enter a 10-digit NHS number.");
    expect(errorsFor(one(next), "q_nhs", "943 476 5919")).toEqual([]);
    const custom = { ...q, ...setTextFormat(q, "custom", "^[A-Z]{3}$", "Three capitals") };
    expect(textFormatOf(custom)).toEqual({ format: "custom", pattern: "^[A-Z]{3}$", message: "Three capitals" });
    expect(textFormatOf({ ...q, ...setTextFormat(q, "none") }).format).toBe("none");
    // Choosing "Custom pattern" sticks before a pattern is typed, and starts from the NHS rule.
    const fromNhs = { ...next, ...setTextFormat(next, "custom") };
    expect(textFormatOf(fromNhs)).toMatchObject({ format: "custom", pattern: "^\\d{3} ?\\d{3} ?\\d{4}$" });
    const empty = { ...q, ...setTextFormat(q, "custom") };
    expect(textFormatOf(empty)).toMatchObject({ format: "custom", pattern: "" });
    expect(errorsFor(one(empty), "q_nhs", "anything")).toEqual([]);
  });
});

describe("number settings (QT-06, VAL-03)", () => {
  it("round-trips range, decimals, unit and a soft warning", () => {
    const q: ElementJson = { type: "text", inputType: "number", name: "q_w" };
    const s = { min: 2, max: 400, decimals: 1, unit: "kg", warnMin: 30, warnMax: 200, warnText: "That weight looks unusual" };
    const next = { ...q, ...setNumberSettings(q, s) };
    expect(numberSettingsOf(next)).toEqual(s);
    expect(kindOf(next)).toBe("number");
    expect(errorsFor(one(next), "q_w", 500)).toContain("error: Please enter a number at least 2 and at most 400.");
    expect(errorsFor(one(next), "q_w", 250)).toEqual(["warning: That weight looks unusual"]);
    const plain = { ...next, ...setNumberSettings(next, {}) };
    expect(plain).toMatchObject({ inputType: "number" });
    expect(plain.validators).toBeUndefined();
  });
});

describe("date settings (QT-05)", () => {
  it("rebuilds the question for each format, range and a list of dates", () => {
    const doc = one({ type: "text", inputType: "date", name: "q_d", title: "When?", isRequired: true, visibleIf: "{x} = 1" });
    const past = setDateSettings(doc, "q_d", { format: "dmy", range: "past", multiple: false });
    expect(el(past, "q_d")).toEqual({ type: "text", inputType: "date", name: "q_d", title: "When?", isRequired: true, visibleIf: "{x} = 1", min: "1900-01-01", maxValueExpression: "today()" });
    const years = setDateSettings(doc, "q_d", { format: "y", range: "future", multiple: false });
    expect(dateSettingsOf(el(years, "q_d"))).toEqual({ format: "y", range: "future", multiple: false });
    expect(kindOf(el(years, "q_d"))).toBe("date");
    const list = setDateSettings(doc, "q_d", { format: "my", range: "any", multiple: true });
    expect(el(list, "q_d").type).toBe("matrixdynamic");
    expect(dateSettingsOf(el(list, "q_d"))).toEqual({ format: "my", range: "any", multiple: true });
    expect(kindOf(el(list, "q_d"))).toBe("date");
    const m = new Model(list);
    m.setValue("q_d", [{ date: "2020-03" }, { date: "2021-04" }]);
    expect(m.getValue("q_d")).toHaveLength(2);
  });
});

describe("options (OPT-04, QT-02, QT-07)", () => {
  const many: ElementJson = { type: "checkbox", name: "q_m", choices: [{ value: "o_a", text: "A" }, { value: "o_n", text: "None", isExclusive: true }] };

  it("adds one-click options at the end, exclusive in Select Many, and removes them", () => {
    let doc = setSpecialOption(one(many), "q_m", "dontknow", true);
    doc = setSpecialOption(doc, "q_m", "refuse", true);
    const opts = optionsOf(el(doc, "q_m"));
    expect(opts.map((o) => o.text)).toEqual(["A", "None", "Don't know", "Prefer not to say"]);
    expect(opts[2]).toMatchObject({ special: "dontknow", isExclusive: true });
    doc = setSpecialOption(doc, "q_m", "dontknow", false);
    expect(optionsOf(el(doc, "q_m")).map((o) => o.text)).toEqual(["A", "None", "Prefer not to say"]);
  });

  it("shows options as buttons or as a list", () => {
    const list = { ...many, ...setChoiceDisplay(many, "list") };
    expect(list.type).toBe("tagbox");
    expect(choiceDisplayOf(list)).toBe("list");
    expect(kindOf(list)).toBe("selectmany");
    expect({ ...list, ...setChoiceDisplay(list, "buttons") }.type).toBe("checkbox");
  });
});

describe("repeating groups (STR-04)", () => {
  it("turns a group into a repeating group and back, with conditions per entry", () => {
    const doc = one({
      type: "panel", name: "g_ops", title: "Operation",
      elements: [
        { type: "radiogroup", name: "q_ga", choices: [{ value: "o_y", text: "Yes" }] },
        { type: "text", name: "q_why", visibleIf: "{q_ga} = 'o_y' or {q_out} = 1" },
      ],
    });
    const rep = setRepeating(doc, "g_ops", { min: 1, max: 5, addText: "Add operation" });
    const g = el(rep, "g_ops");
    expect(g).toMatchObject({ type: "paneldynamic", minPanelCount: 1, maxPanelCount: 5, panelAddText: "Add operation" });
    expect(g.elements).toBeUndefined();
    expect(el(rep, "q_why").visibleIf).toBe("{panel.q_ga} = 'o_y' or {q_out} = 1");
    expect(kindOf(g)).toBe("group");
    const m = new Model(rep);
    m.setValue("g_ops", [{ q_ga: "o_y" }, {}]);
    const panels = (m.getQuestionByName("g_ops") as unknown as { panels: { getQuestionByName: (n: string) => { isVisible: boolean } }[] }).panels;
    expect(panels.map((p) => p.getQuestionByName("q_why").isVisible)).toEqual([true, false]);
    const back = setRepeating(rep, "g_ops", undefined);
    expect(el(back, "g_ops")).toMatchObject({ type: "panel" });
    expect(el(back, "q_why").visibleIf).toBe("{q_ga} = 'o_y' or {q_out} = 1");
  });
});

describe("changing type", () => {
  it("offers compatible kinds and rewrites conditions that test the question", () => {
    const yes: ElementJson = { type: "radiogroup", yesNo: true, name: "q_y", choices: [{ value: "o_y", text: "Yes" }, { value: "o_n", text: "No" }] };
    expect(convertibleKinds(yes)).toEqual(["selectone", "selectmany"]);
    const doc = one(yes, { type: "text", name: "q_t", visibleIf: "{q_y} <> 'o_y'" });
    const many = convertKind(doc, "q_y", "selectmany");
    expect(el(many, "q_y")).toMatchObject({ type: "checkbox", choices: yes.choices });
    expect(el(many, "q_t").visibleIf).toBe("{q_y} notcontains 'o_y'");
    const back = convertKind(many, "q_y", "yesno");
    expect(el(back, "q_t").visibleIf).toBe("{q_y} <> 'o_y'");
    expect(convertibleKinds({ type: "text", name: "q" })).toEqual(["number", "date"]);
    expect(kindOf(el(convertKind(updateElement(doc, "q_t", {}), "q_t", "date"), "q_t"))).toBe("date");
  });
});

describe("calculation formulas (CAL-01/03)", () => {
  it("round-trips the formulas the form builds", async () => {
    const { calcFormOf, formatCalc } = await import("./fields");
    for (const expr of ["score('q_a', 'q_b')", "age({q_dob})", "bmi({q_h}, {q_w})", "{q_a} * 2"]) expect(formatCalc(calcFormOf(expr))).toBe(expr);
    expect(calcFormOf("score('q_a', 'q_b')")).toEqual({ kind: "score", questions: ["q_a", "q_b"] });
    expect(calcFormOf("")).toEqual({ kind: "score", questions: [] });
    // The chosen kind is kept while its formula is still empty, and a custom formula stays custom.
    expect(calcFormOf("", "years")).toEqual({ kind: "years", question: "" });
    expect(calcFormOf("", "bmi")).toEqual({ kind: "bmi", height: "", weight: "" });
    expect(calcFormOf("", "custom")).toEqual({ kind: "custom", expression: "" });
    expect(calcFormOf("score('q_a')", "custom")).toEqual({ kind: "custom", expression: "score('q_a')" });
  });
});

describe("saved option lists (OPT-07)", () => {
  it("saves options without IDs, and copies them in with new IDs", async () => {
    const { applySavedOptions, toSavedOptions } = await import("./fields");
    const src: ElementJson = {
      type: "checkbox", name: "q_a",
      choices: [
        { value: "o_gen", text: "General", score: 2, visibleIf: "{x} = 1", clinicalOutputs: [{ id: "out_a", codes: [] }] },
        { value: "o_dk", text: "Don't know", special: "dontknow", isExclusive: true },
      ],
    };
    const saved = toSavedOptions(src);
    expect(saved).toEqual([
      { text: "General", score: 2, clinicalOutputs: [{ id: "out_a", codes: [] }] },
      { text: "Don't know", special: "dontknow", isExclusive: true },
    ]);
    const target = one({ type: "radiogroup", name: "q_b", choices: [{ value: "o_old", text: "Old" }] });
    const added = optionsOf(el(applySavedOptions(target, "q_b", saved, "add"), "q_b"));
    expect(added.map((o) => o.text)).toEqual(["Old", "General", "Don't know"]);
    expect(added[1].value).not.toBe("o_gen");
    expect(added[1].clinicalOutputs![0].id).not.toBe("out_a");
    expect(added[2].isExclusive).toBeUndefined(); // only Select Many has exclusive options
    const replaced = optionsOf(el(applySavedOptions(target, "q_b", saved, "replace"), "q_b"));
    expect(replaced.map((o) => o.text)).toEqual(["General", "Don't know"]);
  });
});
