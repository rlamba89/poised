import { describe, expect, it } from "vitest";
import { ItemValue, Model, type PanelModel, type QuestionCheckboxModel } from "survey-core";
import {
  GROUP_ID, OPTION_ID, QUESTION_ID, assignGroupId, assignMissingOptionIds, assignOptionId, assignQuestionId, newId,
} from "./ids";

describe("newId", () => {
  it("makes a prefixed id of lowercase letters", () => {
    for (let i = 0; i < 100; i++) {
      expect(newId("q", 6)).toMatch(QUESTION_ID);
      expect(newId("o", 4)).toMatch(OPTION_ID);
    }
  });

  it("retries until the id is free", () => {
    const seen: string[] = [];
    const id = newId("o", 4, (candidate) => {
      seen.push(candidate);
      return seen.length < 3;
    });
    expect(seen).toHaveLength(3);
    expect(id).toBe(seen[2]);
  });
});

describe("assignQuestionId", () => {
  it("renames the question and keeps the label it showed", () => {
    const m = new Model({ elements: [{ type: "text", name: "question1" }] });
    const q = m.getQuestionByName("question1");
    assignQuestionId(q);
    expect(q.name).toMatch(QUESTION_ID);
    expect(q.title).toBe("question1");
  });

  it("keeps an existing title", () => {
    const m = new Model({ elements: [{ type: "text", name: "question1", title: "Do you smoke?" }] });
    const q = m.getQuestionByName("question1");
    assignQuestionId(q);
    expect(q.title).toBe("Do you smoke?");
  });

  it("never reuses a name in the survey", () => {
    const m = new Model({ elements: Array.from({ length: 50 }, (_, i) => ({ type: "text", name: `question${i}` })) });
    for (const q of m.getAllQuestions()) assignQuestionId(q);
    const names = m.getAllQuestions().map((q) => q.name);
    expect(new Set(names).size).toBe(50);
  });
});

describe("assignGroupId", () => {
  it("renames the group and keeps its label", () => {
    const m = new Model({ elements: [{ type: "panel", name: "panel1" }] });
    const p = m.getPanelByName("panel1") as PanelModel;
    assignGroupId(p);
    expect(p.name).toMatch(GROUP_ID);
    expect(p.title).toBe("panel1");
  });
});

describe("option ids", () => {
  const checkbox = (choices: unknown[]) =>
    new Model({ elements: [{ type: "checkbox", name: "q_a", choices }] }).getQuestionByName("q_a") as QuestionCheckboxModel;

  it("gives a new option an id unique in its question and keeps its label", () => {
    const q = checkbox([{ value: "o_aaaa", text: "Yes" }]);
    const item = new ItemValue("Item 2");
    q.choices.push(item);
    assignOptionId(item, q.choices);
    expect(item.value).toMatch(OPTION_ID);
    expect(item.value).not.toBe("o_aaaa");
    expect(item.text).toBe("Item 2");
  });

  it("assigns only missing, invalid or duplicate ids", () => {
    const q = checkbox([
      { value: "o_keep", text: "Keep" },
      "Plain",
      { value: "o_dupe", text: "First" },
      { value: "o_dupe", text: "Second" },
      { value: "o_bad1", text: "Digit" },
    ]);
    assignMissingOptionIds(q.choices);
    const [keep, plain, first, second, digit] = q.choices;
    expect(keep.value).toBe("o_keep");
    expect(plain.value).toMatch(OPTION_ID);
    expect(plain.text).toBe("Plain");
    expect(first.value).not.toBe(second.value);
    expect(digit.value).toMatch(OPTION_ID);
    expect(q.choices.map((c) => c.text)).toEqual(["Keep", "Plain", "First", "Second", "Digit"]);
    expect(new Set(q.choices.map((c) => c.value)).size).toBe(5);
  });
});
