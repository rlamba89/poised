import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import {
  KINDS, addChoice, copyPage, isValidExpression, movePageTo, unknownReferences, addElement, addPage, copyElement, countDisclosures, deleteElement, deletePage,
  dependentsOf, findElement, formatCondition, kindOf, moveElement, moveElementTo, movePage, referencesIn, rewriteReferences,
  newElement, optionsOf, parseCondition, plainText, setNoneOption, textToHtml, updateElement, updatePage, groupsToMoveInto,
  moveOutOfGroup, type ChapterJson,
} from "./doc";
import { GROUP_ID, OPTION_ID, QUESTION_ID } from "./ids";

/** A chapter with one page, a Yes / No question and a text question shown when it's Yes. */
function sample(): ChapterJson {
  return {
    pages: [
      {
        name: "p_one",
        title: "First",
        elements: [
          { type: "radiogroup", yesNo: true, name: "q_smoke", title: "Do you smoke?", choices: [{ value: "o_yes", text: "Yes", clinicalOutputs: [{ id: "out_a", codes: [] }] }, { value: "o_no", text: "No" }] },
          { type: "text", name: "q_many", title: "How many?", visibleIf: "{q_smoke} = 'o_yes'" },
        ],
      },
    ],
  };
}

describe("kinds", () => {
  it("round-trips every kind through newElement and kindOf", () => {
    for (const kind of KINDS) expect(kindOf(newElement(kind, new Set()))).toBe(kind);
  });

  it("gives new elements stable IDs and makes questions required", () => {
    const yes = newElement("yesno", new Set());
    expect(yes.name).toMatch(QUESTION_ID);
    expect(yes.isRequired).toBe(true);
    expect(optionsOf(yes).map((o) => o.text)).toEqual(["Yes", "No"]);
    for (const o of optionsOf(yes)) expect(o.value).toMatch(OPTION_ID);
    expect(newElement("group", new Set()).name).toMatch(GROUP_ID);
    expect(newElement("statement", new Set()).isRequired).toBeUndefined();
  });

  it("loads every kind into a SurveyJS model", () => {
    const doc: ChapterJson = { pages: [{ name: "p_a", elements: KINDS.filter((k) => !["medication", "admissions", "bmi", "profile"].includes(k)).map((k) => newElement(k, new Set())) }] };
    expect(new Model(doc).getAllQuestions().length).toBeGreaterThan(0);
  });
});

describe("editing", () => {
  it("never changes the chapter it was given", () => {
    const doc = sample();
    const before = JSON.stringify(doc);
    updateElement(doc, "q_smoke", { title: "Changed" });
    addPage(doc);
    deleteElement(doc, "q_many");
    expect(JSON.stringify(doc)).toBe(before);
  });

  it("adds pages and elements with fresh IDs", () => {
    const { doc, name } = addPage(sample());
    expect(doc.pages!.map((p) => p.title)).toEqual(["First", "Page 2"]);
    const added = addElement(doc, { page: name }, "selectmany");
    expect(findElement(added.doc, added.name)?.page.name).toBe(name);
  });

  it("adds into a group at an index", () => {
    const g = addElement(sample(), { page: "p_one", index: 0 }, "group");
    const q = addElement(g.doc, { page: "p_one", parent: g.name }, "text");
    expect(findElement(q.doc, q.name)?.parent?.name).toBe(g.name);
    expect(q.doc.pages![0].elements![0].name).toBe(g.name);
  });

  it("removes a patch key when its value is undefined", () => {
    const doc = updateElement(sample(), "q_many", { visibleIf: undefined });
    expect(findElement(doc, "q_many")?.el).not.toHaveProperty("visibleIf");
  });

  it("moves elements and pages, ignoring moves past the ends", () => {
    let doc = moveElement(sample(), "q_many", -1);
    expect(doc.pages![0].elements!.map((e) => e.name)).toEqual(["q_many", "q_smoke"]);
    doc = moveElement(doc, "q_many", -1);
    expect(doc.pages![0].elements![0].name).toBe("q_many");
    const two = addPage(doc).doc;
    expect(movePage(two, "p_one", 1).pages![1].name).toBe("p_one");
  });

  it("moves an element into a group, and not into itself", () => {
    const g = addElement(sample(), { page: "p_one" }, "group");
    const moved = moveElementTo(g.doc, "q_smoke", { page: "p_one", parent: g.name });
    expect(findElement(moved, "q_smoke")?.parent?.name).toBe(g.name);
    const self = moveElementTo(moved, g.name, { page: "p_one", parent: g.name });
    expect(findElement(self, g.name)?.parent).toBeUndefined();
  });

  it("moves into a group with the menu, and back out after the group", () => {
    let doc = addElement(sample(), { page: "p_one" }, "group").doc;
    const g = doc.pages![0].elements![2].name;
    const sec = addElement(doc, { page: "p_one" }, "section");
    doc = sec.doc;
    expect(groupsToMoveInto(doc, "q_smoke").map((e) => e.name)).toEqual([g, sec.name]);
    expect(groupsToMoveInto(doc, sec.name)).toEqual([]); // a Section never goes inside a group
    expect(groupsToMoveInto(doc, g).map((e) => e.name)).toEqual([sec.name]);
    doc = moveElementTo(doc, "q_smoke", { page: "p_one", parent: g });
    expect(groupsToMoveInto(doc, "q_smoke").map((e) => e.name)).toEqual([sec.name]); // not its own group
    expect(groupsToMoveInto(moveElementTo(doc, g, { page: "p_one", parent: sec.name }), sec.name)).toEqual([]);
    doc = moveOutOfGroup(doc, "q_smoke");
    expect(doc.pages![0].elements!.map((e) => e.name)).toEqual(["q_many", g, "q_smoke", sec.name]);
  });

  it("deletes pages", () => {
    expect(deletePage(sample(), "p_one").pages).toEqual([]);
  });
});

describe("copyElement (STR-07)", () => {
  it("copies with new IDs, keeps outputs and puts the copy below", () => {
    const { doc, name } = copyElement(sample(), "q_smoke");
    const els = doc.pages![0].elements!;
    expect(els[1].name).toBe(name);
    expect(name).toMatch(QUESTION_ID);
    const copy = optionsOf(els[1]);
    expect(copy[0].value).not.toBe("o_yes");
    expect(copy[0].clinicalOutputs).toHaveLength(1);
    expect(copy[0].clinicalOutputs![0].id).not.toBe("out_a");
  });

  it("points conditions inside a copied group at the copies", () => {
    const g = addElement(sample(), { page: "p_one" }, "group");
    let doc = moveElementTo(g.doc, "q_smoke", { page: "p_one", parent: g.name });
    doc = moveElementTo(doc, "q_many", { page: "p_one", parent: g.name });
    const copied = copyElement(doc, g.name);
    const group = findElement(copied.doc, copied.name)!.el;
    const [q, follow] = group.elements!;
    expect(follow.visibleIf).toBe(`{${q.name}} = '${optionsOf(q)[0].value}'`);
    // The original is untouched.
    expect(findElement(copied.doc, "q_many")!.el.visibleIf).toBe("{q_smoke} = 'o_yes'");
  });
});

describe("options", () => {
  it("adds options before the exclusive None option", () => {
    let doc = setNoneOption(sample(), "q_smoke", "Neither");
    doc = addChoice(doc, "q_smoke", "Sometimes");
    expect(optionsOf(findElement(doc, "q_smoke")!.el).map((o) => o.text)).toEqual(["Yes", "No", "Sometimes", "Neither"]);
  });

  it("renames and removes the None option", () => {
    let doc = setNoneOption(sample(), "q_smoke", "None");
    doc = setNoneOption(doc, "q_smoke", "No to all");
    expect(optionsOf(findElement(doc, "q_smoke")!.el).filter((o) => o.isExclusive).map((o) => o.text)).toEqual(["No to all"]);
    doc = setNoneOption(doc, "q_smoke", undefined);
    expect(optionsOf(findElement(doc, "q_smoke")!.el).some((o) => o.isExclusive)).toBe(false);
  });

  it("reads options saved as plain strings", () => {
    expect(optionsOf({ type: "checkbox", name: "q_x", choices: ["a"] })).toEqual([{ value: "a", text: "a" }]);
  });

  it("counts disclosures on options or on the question", () => {
    expect(countDisclosures(findElement(sample(), "q_smoke")!.el)).toBe(1);
    expect(countDisclosures({ type: "text", name: "q_t", clinicalOutputs: [{ id: "x", codes: [] }] })).toBe(1);
  });
});

describe("conditions", () => {
  it("parses and formats the simple forms", () => {
    expect(parseCondition("{q_a} = 'o_b'")).toEqual({ question: "q_a", isNot: false, option: "o_b" });
    expect(parseCondition("{q_a} notcontains 'o_b'")).toEqual({ question: "q_a", isNot: true, option: "o_b" });
    expect(parseCondition("{q_a} = 'o_b' and {q_c} = 'o_d'")).toBeUndefined();
    const many = newElement("selectmany", new Set());
    expect(formatCondition({ question: "q_a", isNot: true, option: "o_b" }, many)).toBe("{q_a} notcontains 'o_b'");
    expect(formatCondition({ question: "q_a", isNot: false, option: "o_b" }, newElement("yesno", new Set()))).toBe("{q_a} = 'o_b'");
  });

  it("lists what depends on an element: conditions, option conditions, calculations", () => {
    expect(dependentsOf(sample(), "q_smoke")).toEqual(['"How many?"']);
    expect(dependentsOf(sample(), "q_many")).toEqual([]);
    const doc = addElement(sample(), { page: "p_one" }, "calculation");
    const withCalc = updateElement(doc.doc, doc.name, { title: "Score", expression: "score('q_smoke')" });
    expect(dependentsOf(withCalc, "q_smoke")).toEqual(['"How many?"', '"Score" (calculation)']);
  });

  it("finds and rewrites references in every form", () => {
    expect(referencesIn("{q_a} = 'o_b' and {panel.q_c} > 1 or {q_g.r_x} = 'o_y' or score('q_d', 'q_e') > 2")).toEqual(["q_a", "q_c", "q_g", "q_d", "q_e"]);
    const names = new Map([["q_a", "q_z"], ["q_d", "q_w"]]);
    const options = new Map([["q_a", new Map([["o_b", "o_n"], ["o_c", "o_m"]])]]);
    expect(rewriteReferences("{q_a} anyof ['o_b', 'o_c'] and band('q_d') = 'x' and {panel.q_a} empty", names, options))
      .toBe("{q_z} anyof ['o_n', 'o_m'] and band('q_w') = 'x' and {panel.q_z} empty");
  });
});

describe("translations kept while editing (LNG-01)", () => {
  it("keeps other languages when the English text changes", () => {
    let doc = updateElement(sample(), "q_smoke", { title: { default: "Do you smoke?", de: "Rauchen Sie?" } as never });
    doc = updateElement(doc, "q_smoke", { title: "Do you smoke now?" });
    expect(findElement(doc, "q_smoke")!.el.title).toEqual({ default: "Do you smoke now?", de: "Rauchen Sie?" });
    doc = updateElement(doc, "q_smoke", {
      choices: [{ value: "o_yes", text: { default: "Yes", de: "Ja" } as never }, { value: "o_no", text: "No" }],
    });
    doc = updateElement(doc, "q_smoke", { choices: optionsOf(findElement(doc, "q_smoke")!.el).map((o) => ({ ...o, text: o.text + "!" })) });
    expect(findElement(doc, "q_smoke")!.el.choices).toEqual([{ value: "o_yes", text: { default: "Yes!", de: "Ja" } }, { value: "o_no", text: "No!" }]);
    doc = updatePage(updatePage(doc, "p_one", { title: { default: "First", fr: "Premier" } as never }), "p_one", { title: "One" });
    expect(doc.pages![0].title).toEqual({ default: "One", fr: "Premier" });
  });
});

describe("skip rules (LOG-12)", () => {
  it("stay on the page they were added on, and older ones go to the latest page they test", async () => {
    const { pageOfTrigger, setTriggers, triggersOf } = await import("./doc");
    const { logicProblems } = await import("./logic");
    (await import("./properties")).registerClinicalProperties();
    const two = addPage(sample());
    const own = { type: "complete" as const, expression: "{q_smoke} = 'o_no'", page: two.name };
    const old = { type: "complete" as const, expression: "{q_smoke} = 'o_no'" };
    expect(pageOfTrigger(two.doc, own)?.name).toBe(two.name);
    expect(pageOfTrigger(two.doc, old)?.name).toBe("p_one");
    expect(new Model(setTriggers(two.doc, [own])).jsonErrors ?? []).toEqual([]);
    expect(triggersOf(setTriggers(two.doc, [own]))).toEqual([own]);
    // A rule on page 1 that tests a question on page 2 is flagged.
    const q = addElement(two.doc, { page: two.name }, "yesno");
    const late = setTriggers(q.doc, [{ type: "complete", expression: `{${q.name}} = 'x'`, page: "p_one" }]);
    expect(logicProblems(late).map((p) => p.message)).toEqual(["it tests a question on a later page"]);
  });
});

describe("statements", () => {
  it("round-trips text through HTML, escaped", () => {
    const text = "Line one\nA < B & C";
    expect(textToHtml(text)).toBe("<p>Line one</p><p>A &lt; B &amp; C</p>");
    expect(plainText(textToHtml(text))).toBe(text);
  });
});

describe("pages", () => {
  it("copies a page with new IDs and its conditions remapped", () => {
    const { doc, name } = copyPage(sample(), "p_one");
    expect(doc.pages!.map((p) => p.title)).toEqual(["First", "First (copy)"]);
    const [q, follow] = doc.pages![1].elements!;
    expect(q.name).not.toBe("q_smoke");
    expect(follow.visibleIf).toBe(`{${q.name}} = '${optionsOf(q)[0].value}'`);
    expect(doc.pages![1].name).toBe(name);
  });

  it("moves a page before another", () => {
    let doc = addPage(sample()).doc;
    doc = addPage(doc).doc;
    const [a, b, c] = doc.pages!.map((p) => p.name);
    expect(movePageTo(doc, c, a).pages!.map((p) => p.name)).toEqual([c, a, b]);
  });
});

describe("expression checks", () => {
  it("validates syntax and finds unknown questions", () => {
    expect(isValidExpression("{q_smoke} = 'o_yes' and {q_many} > 3")).toBe(true);
    expect(isValidExpression("{q_smoke} =")).toBe(false);
    expect(unknownReferences(sample(), "{q_smoke} = 'o_yes' or {q_typo} > 1 or {viewer} = 'patient'")).toEqual(["q_typo"]);
  });
});
