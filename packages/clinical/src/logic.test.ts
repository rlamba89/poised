import { describe, expect, it } from "vitest";
import { Model } from "survey-core";
import { addPage, type ChapterJson } from "./doc";
import {
  builderCanShow, describeLogic, enableIfToReadOnly, formatLogic, logicProblems, parseLogic, readOnlyToEnableIf, subjectsFor,
  type Group,
} from "./logic";
import { registerClinicalProperties } from "./properties";

registerClinicalProperties();

function chapter(): ChapterJson {
  return {
    pages: [
      {
        name: "p_one",
        title: "About you",
        elements: [
          { type: "radiogroup", yesNo: true, name: "q_smoke", title: "Do you smoke?", choices: [{ value: "o_yes", text: "Yes", score: 1 }, { value: "o_no", text: "No" }] },
          { type: "checkbox", name: "q_cond", title: "Conditions", choices: [{ value: "o_asth", text: "Asthma", score: 2 }, { value: "o_copd", text: "COPD", score: 3 }] },
          { type: "text", inputType: "number", name: "q_cigs", title: "How many a day?" },
          { type: "text", inputType: "date", name: "q_dx", title: "When diagnosed?" },
          { type: "matrix", name: "q_grid", title: "Symptoms", rows: [{ value: "r_cough", text: "Cough" }], columns: [{ value: "o_none", text: "None" }, { value: "o_bad", text: "Bad" }] },
          {
            type: "expression", name: "q_score", title: "Risk score", expression: "score('q_smoke', 'q_cond')",
            bands: [{ id: "b_low", label: "Low", max: 2 }, { id: "b_high", label: "High" }],
          },
          {
            type: "paneldynamic", name: "g_ops", title: "Operations",
            templateElements: [
              { type: "radiogroup", name: "q_gen", title: "General anaesthetic?", choices: [{ value: "o_y", text: "Yes" }, { value: "o_n", text: "No" }] },
              { type: "text", name: "q_prob", title: "Any problems?", visibleIf: "{panel.q_gen} = 'o_y'" },
            ],
          },
          { type: "comment", name: "q_more", title: "Tell us more" },
        ],
      },
    ],
  };
}

describe("formatLogic and parseLogic", () => {
  it("round-trips groups of rules with AND, OR and a sub-group", () => {
    const g: Group = {
      join: "and",
      items: [
        { left: "q_smoke", op: "eq", value: "o_yes" },
        { join: "or", items: [{ left: "q_cigs", op: "gt", value: 10 }, { left: "q_cond", op: "anyof", value: ["o_asth", "o_copd"] }] },
        { left: "q_dx", fn: "age", op: "gte", value: 5 },
        { left: "q_more", op: "notempty" },
        { left: "q_score", fn: "band", op: "eq", value: "b_high" },
      ],
    };
    const expr = formatLogic(g)!;
    expect(expr).toBe(
      "{q_smoke} = 'o_yes' and ({q_cigs} > 10 or {q_cond} anyof ['o_asth', 'o_copd']) and age({q_dx}) >= 5 and {q_more} notempty and band('q_score') = 'b_high'",
    );
    expect(parseLogic(expr)).toEqual(g);
    expect(builderCanShow(parseLogic(expr))).toBe(true);
  });

  it("reads Lifebox's simple form and the importer's negations", () => {
    expect(parseLogic("{q_a} = 'o_b'")).toEqual({ join: "and", items: [{ left: "q_a", op: "eq", value: "o_b" }] });
    expect(parseLogic("!({q_a} contains 'o_b')")).toEqual({ join: "and", items: [{ left: "q_a", op: "notcontains", value: "o_b" }] });
  });

  it("leaves only arithmetic and other functions to the Code tab", () => {
    expect(parseLogic("{q_a} + {q_b} > 3")).toBeUndefined();
    expect(parseLogic("iif({q_a} = 'x', 1, 2) = 1")).toBeUndefined();
  });

  it("nests groups to any depth, and reads negated groups as None of / Not all of", () => {
    const deep = "{a} = 1 and ({b} = 2 or ({c} = 3 and ({d} = 4 or {e} empty)))";
    expect(builderCanShow(parseLogic(deep))).toBe(true);
    expect(formatLogic(parseLogic(deep)!)).toBe(deep);
    const none = parseLogic("!({q_a} = 'x' or {q_b} = 'y')")!;
    expect(none).toEqual({ join: "and", items: [{ join: "or", not: true, items: [{ left: "q_a", op: "eq", value: "x" }, { left: "q_b", op: "eq", value: "y" }] }] });
    expect(formatLogic(none)).toBe("!({q_a} = 'x' or {q_b} = 'y')");
    expect(formatLogic(parseLogic("{q_c} = 'z' and !({q_a} = 'x' and {q_b} = 'y')")!)).toBe("{q_c} = 'z' and !({q_a} = 'x' and {q_b} = 'y')");
  });

  it("drops incomplete rules, and is empty with none", () => {
    expect(formatLogic({ join: "and", items: [{ left: "q_a", op: "eq" }, { left: "", op: "eq", value: "x" }] })).toBeUndefined();
    expect(formatLogic({ join: "or", items: [{ left: "q_a", op: "empty" }, { join: "and", items: [{ left: "q_b", op: "gt", value: 1 }] }] }))
      .toBe("{q_a} empty or {q_b} > 1");
    expect(parseLogic("")).toEqual({ join: "and", items: [] });
  });

  it("writes conditions SurveyJS evaluates as intended", () => {
    const json = chapter();
    json.pages![0].elements!.push({ type: "text", name: "q_t", visibleIf: formatLogic(parseLogic("{q_cond} allof ['o_asth', 'o_copd'] and {patientAge} >= 65")!) });
    const m = new Model(json);
    m.setVariable("patientAge", 70);
    m.setValue("q_cond", ["o_asth"]);
    expect(m.getQuestionByName("q_t").isVisible).toBe(false);
    m.setValue("q_cond", ["o_asth", "o_copd"]);
    expect(m.getQuestionByName("q_t").isVisible).toBe(true);
  });
});

describe("isAllComplete", () => {
  it("is false while any rule, at any depth, is unfinished", async () => {
    const { isAllComplete } = await import("./logic");
    expect(isAllComplete({ join: "and", items: [{ left: "q_a", op: "eq", value: "x" }] })).toBe(true);
    expect(isAllComplete({ join: "and", items: [{ left: "q_b", op: "eq" }] })).toBe(false); // question changed, no value yet
    expect(isAllComplete({ join: "and", items: [{ left: "q_a", op: "empty" }, { join: "or", items: [{ left: "", op: "eq" }] }] })).toBe(false);
    expect(isAllComplete({ join: "and", items: [] })).toBe(true); // everything removed: "Always"
  });
});

describe("read-only (LOG-11)", () => {
  it("stores read-only when C as enableIf = !(C) and reads it back", () => {
    expect(readOnlyToEnableIf("{q_a} = 'x'")).toBe("!({q_a} = 'x')");
    expect(enableIfToReadOnly("!({q_a} = 'x')")).toBe("{q_a} = 'x'");
    expect(enableIfToReadOnly("{q_a} = 'x'")).toBe("!({q_a} = 'x')");
    expect(enableIfToReadOnly(undefined)).toBeUndefined();
  });
});

describe("subjectsFor (LOG-03, LOG-05)", () => {
  it("offers earlier questions with types, grid rows, bands, then the patient", () => {
    const s = subjectsFor(chapter(), { element: "q_more" });
    const byId = Object.fromEntries(s.map((x) => [x.id, x]));
    expect(byId.q_smoke.type).toBe("single");
    expect(byId.q_cond.type).toBe("multi");
    expect(byId.q_cigs.type).toBe("number");
    expect(byId.q_dx.type).toBe("date");
    expect(byId["q_grid.r_cough"]).toMatchObject({ type: "single", label: "Symptoms › Cough" });
    expect(byId["band:q_score"]).toMatchObject({ fn: "band", options: [{ value: "b_low", label: "Low" }, { value: "b_high", label: "High" }] });
    expect(byId.q_gen).toBeUndefined(); // inside a repeating group
    expect(s.slice(-3).map((x) => x.id)).toEqual(["patientAge", "patientSex", "viewer"]);
    expect(s.some((x) => x.id === "q_more")).toBe(false);
  });

  it("inside a repeating group, offers its earlier questions per entry", () => {
    const s = subjectsFor(chapter(), { element: "q_prob" });
    expect(s.find((x) => x.label === "General anaesthetic?")?.left).toBe("panel.q_gen");
  });

  it("for a page, offers questions on earlier pages only; for a skip rule, its own page too", () => {
    const two = addPage(chapter());
    expect(subjectsFor(two.doc, { page: "p_one" }).filter((x) => x.group !== "Patient and viewer")).toEqual([]);
    expect(subjectsFor(two.doc, { page: two.name }).some((x) => x.id === "q_smoke")).toBe(true);
    expect(subjectsFor(two.doc, { upToPage: "p_one" }).some((x) => x.id === "q_smoke")).toBe(true);
  });
});

describe("describeLogic (LOG-06)", () => {
  it("puts a condition in plain words", () => {
    const text = (expr: string) => {
      const d = describeLogic(chapter(), expr);
      return d && "parts" in d ? d.parts.map((p) => (p.strong ? `*${p.text}*` : p.text)).join(" ") : d?.raw;
    };
    expect(text("{q_smoke} = 'o_yes'")).toBe("Do you smoke? *is* Yes");
    expect(text("{q_cond} anyof ['o_asth', 'o_copd'] or {patientAge} >= 65")).toBe("Conditions *includes any of* Asthma or COPD *or* Patient's age *≥* 65");
    expect(text("{q_dx} < '2020-01-31' and ({q_more} notempty or band('q_score') = 'b_high')")).toBe(
      "When diagnosed? *is before* 31/01/2020 *and* ( Tell us more *is answered* *or* Risk score band *is* High )",
    );
    expect(text("age({q_dx}) >= 5")).toBe("When diagnosed? *was at least* 5 *years ago*");
    expect(text("{q_cigs} * 2 > 3")).toBe("“How many a day?” * 2 > 3");
    expect(text("!({q_smoke} = 'o_yes' or {q_cigs} > 10)")).toBe("*none of* ( Do you smoke? *is* Yes or How many a day? *>* 10 )");
    expect(describeLogic(chapter(), "")).toBeUndefined();
  });
});

describe("logicProblems (LOG-08)", () => {
  it("flags unreadable conditions, missing questions, and forward or circular references", () => {
    const doc = chapter();
    doc.pages![0].elements![0].visibleIf = "{q_more} notempty"; // forward
    doc.pages![0].elements![1].visibleIf = "{q_gone} = 'x'"; // missing
    doc.pages![0].elements![2].visibleIf = "{q_cigs} > 1"; // itself
    doc.pages![0].elements![3].visibleIf = "{q_smoke} ="; // unreadable
    const problems = logicProblems(doc).map((p) => `${p.label}: ${p.message}`);
    expect(problems).toEqual([
      '"Do you smoke?": it tests a question that comes later or itself',
      '"Conditions": it tests something that no longer exists (q_gone)',
      '"How many a day?": it tests a question that comes later or itself',
      '"When diagnosed?": its condition can\'t be read',
    ]);
    expect(logicProblems(chapter())).toEqual([]);
  });
});
