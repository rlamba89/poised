import { describe, expect, it } from "vitest";
import { findElement, type ChapterJson, type ElementJson } from "./doc";
import { adoptionLosses, adoptKey, insertFromLibrary, namesInElement, optionMapFor, renameReferences, takenNames } from "./library";
import { chapterConditionProblems, combineChapters } from "./questionSets";

const smoker: ElementJson = {
  type: "radiogroup",
  name: "smoker",
  title: "Do you smoke?",
  isRequired: true,
  choices: [
    { value: "never", text: "Never", clinicalOutputs: [{ id: "out_aaaaaa", note: { text: "Never smoked" } }] as never },
    { value: "current", text: { default: "Yes", fr: "Oui" } as never },
  ],
};
const group: ElementJson = {
  type: "panel",
  name: "anaesthetic_problems_group",
  title: "Problems with anaesthetics",
  elements: [
    { type: "radiogroup", name: "anaesthetic_problems", choices: [{ value: "yes", text: "Yes" }] },
    { type: "comment", name: "anaesthetic_problems_details", visibleIf: "{anaesthetic_problems} = 'yes'" },
  ],
};
const doc = (...elements: ElementJson[]): ChapterJson => ({ pages: [{ name: "p_1", elements }] });

describe("namesInElement", () => {
  it("lists an element's name and every name inside it", () => {
    expect(namesInElement(group)).toEqual(["anaesthetic_problems_group", "anaesthetic_problems", "anaesthetic_problems_details"]);
    expect(namesInElement({ type: "paneldynamic", name: "allergies", templateElements: [{ type: "text", name: "allergies_what" }] })).toEqual([
      "allergies",
      "allergies_what",
    ]);
  });
});

describe("takenNames", () => {
  it("finds the element's names already used in any Question Set of the HQ", () => {
    const sets = [doc({ type: "text", name: "q_a" }), doc({ type: "comment", name: "anaesthetic_problems_details" })];
    expect(takenNames(sets, group)).toEqual(["anaesthetic_problems_details"]);
    expect(takenNames(sets, smoker)).toEqual([]);
  });
});

describe("insertFromLibrary", () => {
  it("inserts a copy that keeps its key and option values, with new disclosure IDs", () => {
    const before = doc({ type: "text", name: "q_a" });
    const { doc: after, name } = insertFromLibrary(before, { page: "p_1", index: 0 }, smoker);
    expect(name).toBe("smoker");
    const el = after.pages![0].elements![0];
    expect(el.name).toBe("smoker");
    expect((el.choices as { value: string }[]).map((c) => c.value)).toEqual(["never", "current"]);
    const out = (el.choices as { clinicalOutputs?: { id: string }[] }[])[0].clinicalOutputs![0];
    expect(out.id).not.toBe("out_aaaaaa");
    expect(out.id).toMatch(/^out_[a-z]{6}$/);
    expect(after.pages![0].elements![1].name).toBe("q_a");
    expect(before.pages![0].elements).toHaveLength(1); // the original chapter is unchanged
    expect((smoker.choices as { clinicalOutputs?: { id: string }[] }[])[0].clinicalOutputs![0].id).toBe("out_aaaaaa");
  });
  it("inserts a group with its questions and their condition as they are", () => {
    const { doc: after } = insertFromLibrary(doc(), { page: "p_1" }, group);
    expect(findElement(after, "anaesthetic_problems_details")?.el.visibleIf).toBe("{anaesthetic_problems} = 'yes'");
  });
});

describe("adoptKey", () => {
  const mine: ElementJson = {
    type: "radiogroup",
    name: "q_mine",
    title: "Do you currently smoke?",
    choices: [{ value: "o_aaaa", text: "Option 1", clinicalOutputs: [{ id: "out_x", note: { text: "x" } }] as never }],
  };
  it("takes the key and the options (values and wording, translations too), keeps its own title, and records where they came from", () => {
    const before = doc(mine, { type: "text", name: "q_next", visibleIf: "{q_mine} notempty" });
    const after = adoptKey(before, "q_mine", { ...smoker, showOtherItem: true, otherText: "Something else" }, { questionnaire: "Demo pre-op HQ", title: "Do you smoke?" });
    const el = findElement(after, "smoker")!.el;
    expect(findElement(after, "q_mine")).toBeUndefined();
    expect(el.title).toBe("Do you currently smoke?");
    expect(el.choices).toEqual([
      { value: "never", text: "Never" },
      { value: "current", text: { default: "Yes", fr: "Oui" } },
    ]);
    expect(el.showOtherItem).toBe(true);
    expect(el.otherText).toBe("Something else");
    expect(el.linkedFrom).toEqual({ questionnaire: "Demo pre-op HQ", title: "Do you smoke?" });
    // A condition that tested the question now tests it under its new key.
    expect(findElement(after, "q_next")!.el.visibleIf).toBe("{smoker} notempty");
  });
  it("drops an Other or None option the chosen question doesn't have", () => {
    const after = adoptKey(doc({ ...mine, showNoneItem: true, noneText: "None" }), "q_mine", smoker, { questionnaire: "Library", title: "Do you smoke?" });
    const el = findElement(after, "smoker")!.el;
    expect(el.showNoneItem).toBeUndefined();
    expect(el.noneText).toBeUndefined();
  });
  it("only takes the key (and Always ask fresh) of a question without options", () => {
    const after = adoptKey(doc({ type: "text", inputType: "number", name: "q_w", title: "Your weight", unit: "kg" }), "q_w", {
      type: "text", inputType: "number", name: "weight", title: "What do you weigh now?", unit: "kg", alwaysFresh: true,
    }, { questionnaire: "Library", title: "What do you weigh now?" });
    expect(findElement(after, "weight")!.el).toEqual({
      type: "text", inputType: "number", name: "weight", title: "Your weight", unit: "kg", alwaysFresh: true, // copied (Rahul, 23 Oct)
      linkedFrom: { questionnaire: "Library", title: "What do you weigh now?" },
    });
  });
});

describe("adoptKey after verification 1", () => {
  const mine: ElementJson = {
    type: "radiogroup",
    name: "q_mine",
    yesNo: true,
    title: "Do you smoke now?",
    choices: [
      { value: "o_y", text: "Yes" },
      { value: "o_n", text: "No", clinicalOutputs: [{ id: "out_1", note: { text: "x" } }] as never },
    ],
  };
  const from: ElementJson = {
    type: "radiogroup",
    name: "pregnant",
    alwaysFresh: true,
    title: "Is there any chance you could be pregnant?",
    choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }, { value: "not_sure", text: "I'm not sure" }],
  };
  it("maps conditions on old options to the new option with the same text (finding 1a)", () => {
    const before = doc(mine, { type: "text", name: "q_next", visibleIf: "{q_mine} = 'o_y'" }, { type: "text", name: "q_other", visibleIf: "{q_mine} anyof ['o_n', 'o_gone']" });
    const after = adoptKey(before, "q_mine", from, { questionnaire: "Library", title: "x" });
    expect(findElement(after, "q_next")!.el.visibleIf).toBe("{pregnant} = 'yes'");
    expect(findElement(after, "q_other")!.el.visibleIf).toBe("{pregnant} anyof ['no', 'o_gone']");
  });
  it("copies Always ask fresh when the chosen question has it on (finding 2, Rahul 23 Oct)", () => {
    expect(findElement(adoptKey(doc(mine), "q_mine", from, { questionnaire: "Library", title: "x" }), "pregnant")!.el.alwaysFresh).toBe(true);
    const { alwaysFresh: _fresh, ...plain } = from;
    expect(findElement(adoptKey(doc({ ...mine, alwaysFresh: true }), "q_mine", plain, { questionnaire: "Library", title: "x" }), "pregnant")!.el.alwaysFresh).toBe(true);
    expect(findElement(adoptKey(doc(mine), "q_mine", plain, { questionnaire: "Library", title: "x" }), "pregnant")!.el.alwaysFresh).toBeUndefined();
  });
  it("is no longer Yes / No when it takes options that aren't two (finding 6)", () => {
    expect(findElement(adoptKey(doc(mine), "q_mine", from, { questionnaire: "Library", title: "x" }), "pregnant")!.el.yesNo).toBeUndefined();
    const two = { ...from, choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] };
    expect(findElement(adoptKey(doc(mine), "q_mine", two, { questionnaire: "Library", title: "x" }), "pregnant")!.el.yesNo).toBe(true);
  });
});

describe("adoptionLosses (finding 1c)", () => {
  it("counts the question's disclosures that go and the conditions left testing old answers", () => {
    const mine: ElementJson = {
      type: "radiogroup", name: "q_mine", title: "Smoke?",
      choices: [{ value: "o_y", text: "Yes", clinicalOutputs: [{ id: "a" }, { id: "b" }] as never }, { value: "o_n", text: "Never", clinicalOutputs: [{ id: "c" }] as never }],
    };
    const d = doc(mine, { type: "text", name: "q_1", visibleIf: "{q_mine} = 'o_y'" }, { type: "text", name: "q_2", visibleIf: "{q_mine} = 'o_n'" });
    expect(adoptionLosses(d, "q_mine", { type: "radiogroup", name: "smoker", choices: [{ value: "yes", text: "Yes" }] })).toEqual({ disclosures: 2, conditions: 1 });
    expect(adoptionLosses(d, "q_mine", { type: "radiogroup", name: "smoker", choices: [{ value: "yes", text: "Yes" }, { value: "never", text: "Never" }] })).toEqual({ disclosures: 2, conditions: 0 });
  });
});

describe("renameReferences (finding 4)", () => {
  it("renames a question in another Question Set's condition, with its options", () => {
    const other: ChapterJson = { chapterVisibleIf: "{q_mine} = 'o_y'", pages: [{ name: "p", elements: [] }] };
    expect(renameReferences(other, new Map([["q_mine", "smoker"]]), new Map([["q_mine", new Map([["o_y", "current"]])]])).chapterVisibleIf).toBe("{smoker} = 'current'");
  });
});

describe("Same as… and other Question Sets (verification 2, finding 1)", () => {
  const mine: ElementJson = {
    type: "radiogroup", name: "q_vxa", title: "Do you smoke at all?",
    choices: [{ value: "o_1", text: "Yes" }, { value: "o_2", text: "Sometimes" }],
  };
  const snoring: ElementJson = { type: "radiogroup", name: "snoring", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] };
  const later: ChapterJson = { chapterVisibleIf: "{q_vxa} = 'o_2'", pages: [{ name: "p_2", elements: [] }] };
  it("counts another set's condition on an old answer with no same-text match among the losses", () => {
    expect(adoptionLosses(doc(mine), "q_vxa", snoring, [later])).toEqual({ disclosures: 0, conditions: 1 });
    expect(adoptionLosses(doc(mine), "q_vxa", snoring, [{ ...later, chapterVisibleIf: "{q_vxa} = 'o_1'" }])).toEqual({ disclosures: 0, conditions: 0 });
  });
  it("leaves the renamed condition for chapterConditionProblems to refuse", () => {
    const first = adoptKey(doc(mine), "q_vxa", snoring, { questionnaire: "Library", title: "Do you snore loudly?" });
    const renamed = renameReferences(later, new Map([["q_vxa", "snoring"]]), new Map([["q_vxa", optionMapFor(mine, snoring)]]));
    expect(renamed.chapterVisibleIf).toBe("{snoring} = 'o_2'");
    expect(chapterConditionProblems(renamed, combineChapters([{ name: "Smoking", doc: first }]))).toEqual([
      "it tests an answer that no longer exists (“Do you smoke at all?”: o_2)",
    ]);
  });
  it("renames a test case's answers too, with their values", () => {
    const withCase: ChapterJson = { ...doc(mine), testCases: [{ id: "tc", name: "Yes", answers: { q_vxa: "o_1", q_x: 1 }, expected: [] }] };
    const after = adoptKey(withCase, "q_vxa", snoring, { questionnaire: "Library", title: "x" });
    expect((after.testCases as { answers: Record<string, unknown> }[])[0].answers).toEqual({ snoring: "yes", q_x: 1 });
  });
});
