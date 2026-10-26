import { describe, expect, it } from "vitest";
import { Model, type QuestionHtmlModel } from "survey-core";
import type { ChapterJson, ElementJson } from "./doc";
import { registerClinicalProperties } from "./properties";
import {
  logicOffered, refreshClinicalSummaries, summariesIn, summaryOwners, summaryProblems, upgradeSummaries, withClinicalSummaries,
} from "./summary";
import { stripClinicianOnly } from "./viewer";

registerClinicalProperties();

const out = (id: string, text: string) => [{ id, codes: [], note: { text, category: "Lifestyle" } }];
const yesNo = (name: string, note: string): ElementJson => ({
  type: "radiogroup", name, title: `${name}?`,
  choices: [{ value: `${name}_y`, text: "Yes", clinicalOutputs: out(`out_${name}`, note) }, { value: `${name}_n`, text: "No" }],
});
const summary = (name: string, title?: string): ElementJson => ({ type: "clinicalsummary", name, clinicianOnly: true, ...(title ? { title } : {}) });

// The old page switch (before S32).
const legacy: ChapterJson = {
  pages: [
    { name: "p_one", title: "Smoking", clinicalSummary: true, elements: [yesNo("q_smoke", "Current smoker")] },
    { name: "p_two", title: "Other", elements: [{ type: "text", name: "q_t" }] },
  ],
};

// A page summary, a Group with its own, a nested Group inside it with none, a Section with its own,
// and a second page with a Group summary only.
const scoped: ChapterJson = {
  pages: [
    {
      name: "p_heart", title: "Heart",
      elements: [
        yesNo("q_angina", "Angina"),
        {
          type: "panel", name: "g_breath", title: "Breathing",
          elements: [
            yesNo("q_asthma", "Asthma"),
            { type: "panel", name: "g_inner", title: "Inhalers", elements: [yesNo("q_inhaler", "Uses an inhaler")] },
            summary("cs_breath"),
          ],
        },
        { type: "panel", name: "g_sec", title: "Blood", patientPage: true, elements: [yesNo("q_anaemia", "Anaemia"), summary("cs_sec", "Blood checks")] },
        summary("cs_page"),
      ],
    },
    {
      name: "p_two", title: "Lungs",
      elements: [
        yesNo("q_cough", "Cough"),
        { type: "panel", name: "g_copd", title: "COPD", elements: [yesNo("q_copd", "COPD"), summary("cs_copd")] },
      ],
    },
  ],
};

describe("the old page switch", () => {
  it("is read as a page summary at the end of the page, named cs_<page>", () => {
    const doc = upgradeSummaries(legacy);
    expect(doc.pages![0].clinicalSummary).toBeUndefined();
    expect(doc.pages![0].elements!.at(-1)).toEqual({ type: "clinicalsummary", name: "cs_p_one", clinicianOnly: true });
    expect(doc.pages![1].elements).toHaveLength(1);
    expect(legacy.pages![0].clinicalSummary).toBe(true); // the original isn't changed
    // Its comments keep the old answer key.
    expect(summariesIn(doc)[0]).toMatchObject({ name: "cs_p_one", title: "Smoking summary", commentsId: "cs_comments_p_one", notesId: "cs_notes_p_one" });
  });

  it("leaves content without the switch as it is (the same object)", () => {
    expect(upgradeSummaries(scoped)).toBe(scoped);
  });
});

describe("which notes each summary lists", () => {
  const owners = summaryOwners(scoped);
  it("a Group's summary lists its own questions, nested Groups included", () => {
    expect(owners.get("q_asthma")).toBe("cs_breath");
    expect(owners.get("q_inhaler")).toBe("cs_breath");
  });
  it("a Section's summary lists the Section's questions", () => {
    expect(owners.get("q_anaemia")).toBe("cs_sec");
  });
  it("the page's summary lists only what no nearer summary does", () => {
    expect(owners.get("q_angina")).toBe("cs_page");
    expect([...owners].filter(([, s]) => s === "cs_page").map(([q]) => q)).toEqual(["q_angina", "g_breath", "g_sec"]);
  });
  it("a question outside every summary has none", () => {
    expect(owners.get("q_cough")).toBeUndefined();
    expect(owners.get("q_copd")).toBe("cs_copd");
  });
  it("titles default to the Group's, Section's or page's name", () => {
    expect(summariesIn(scoped).map((s) => [s.name, s.title, s.page, s.scope])).toEqual([
      ["cs_breath", "Breathing summary", "p_heart", "Group “Breathing”"],
      ["cs_sec", "Blood checks", "p_heart", "Section “Blood”"],
      ["cs_page", "Heart summary", "p_heart", "page “Heart”"],
      ["cs_copd", "COPD summary", "p_two", "Group “COPD”"],
    ]);
  });
});

describe("the clinician view", () => {
  it("puts each summary box where the author placed it, with notes and comments, for clinicians only", () => {
    const json = withClinicalSummaries(scoped);
    const group = json.pages![0].elements![1];
    expect(group.elements![2]).toMatchObject({
      type: "panel", name: "cs_breath", title: "Breathing summary", clinicianOnly: true,
      elements: [{ type: "html", name: "cs_notes_breath" }, { type: "comment", name: "cs_comments_breath" }],
    });
    expect(json.pages![0].elements!.at(-1)).toMatchObject({ name: "cs_page", title: "Heart summary" });
    expect(JSON.stringify(stripClinicianOnly(json))).not.toContain("cs_");
  });

  it("reads the old page switch too", () => {
    const json = withClinicalSummaries(legacy);
    expect(json.pages![0].elements!.at(-1)).toMatchObject({ name: "cs_p_one", title: "Smoking summary", clinicianOnly: true });
    expect(json.pages![1].elements).toHaveLength(1);
  });

  it("lists in each box only its own notes", () => {
    const m = new Model(withClinicalSummaries(scoped));
    const html = (id: string) => (m.getQuestionByName(id) as QuestionHtmlModel).html;
    refreshClinicalSummaries(m, scoped);
    expect(html("cs_notes_page")).toContain("No disclosures on this page yet");
    expect(html("cs_notes_breath")).toContain("No disclosures in this group yet");
    m.data = { q_angina: "q_angina_y", q_asthma: "q_asthma_y", q_inhaler: "q_inhaler_y", q_anaemia: "q_anaemia_y" };
    refreshClinicalSummaries(m, scoped);
    expect(html("cs_notes_page")).toBe(`<ul class="sj-summary-notes"><li>Angina</li></ul>`);
    expect(html("cs_notes_breath")).toBe(`<ul class="sj-summary-notes"><li>Asthma</li><li>Uses an inhaler</li></ul>`);
    expect(html("cs_notes_sec")).toBe(`<ul class="sj-summary-notes"><li>Anaemia</li></ul>`);
  });

  it("an old page switch's box lists the page's notes", () => {
    const m = new Model(withClinicalSummaries(legacy));
    m.setValue("q_smoke", "q_smoke_y");
    refreshClinicalSummaries(m, legacy);
    expect((m.getQuestionByName("cs_notes_p_one") as QuestionHtmlModel).html).toContain("<li>Current smoker</li>");
  });
});

describe("summaryProblems", () => {
  it("none for one summary per Group, Section and page", () => {
    expect(summaryProblems(scoped)).toEqual([]);
  });
  it("two summaries covering the same Group, Section or page", () => {
    const doc = structuredClone(scoped);
    doc.pages![0].elements![1].elements!.push(summary("cs_breath2"));
    doc.pages![1].elements!.unshift(summary("cs_lungs"), summary("cs_lungs2"));
    expect(summaryProblems(doc)).toEqual([
      "Two clinical summaries cover the same Group “Breathing”: keep one",
      "Two clinical summaries cover the same page “Lungs”: keep one",
    ]);
  });
  it("the old switch and a page summary are two on the page", () => {
    const doc = structuredClone(legacy);
    doc.pages![0].elements!.unshift(summary("cs_new"));
    expect(summaryProblems(doc)).toEqual(["Two clinical summaries cover the same page “Smoking”: keep one"]);
  });
  it("a summary inside a repeating group", () => {
    const doc: ChapterJson = { pages: [{ name: "p", title: "Meds", elements: [
      { type: "paneldynamic", name: "g_list", title: "Medicines", templateElements: [{ type: "text", name: "q_m" }, summary("cs_in_list")] },
    ] }] };
    expect(summaryProblems(doc)).toEqual(["The clinical summary “Medicines summary” is inside a repeating group: move it out"]);
  });
  // S37 (Rahul, 26 Oct): a hidden summary would hide its notes, so a summary can't have a condition.
  it("a summary with a condition: shown, required or read-only", () => {
    const doc = structuredClone(scoped);
    const els = doc.pages![0].elements!;
    els[1].elements![2].visibleIf = "{q_asthma} = 'q_asthma_y'";
    els[2].elements![1].enableIf = "{q_anaemia} = 'q_anaemia_y'";
    els[3].requiredIf = "{q_angina} = 'q_angina_y'";
    expect(summaryProblems(doc)).toEqual([
      "The clinical summary “Breathing summary” has a condition: remove it (a summary is always shown)",
      "The clinical summary “Blood checks” has a condition: remove it (a summary is always shown)",
      "The clinical summary “Heart summary” has a condition: remove it (a summary is always shown)",
    ]);
  });
  it("an empty condition is no condition", () => {
    const doc = structuredClone(scoped);
    doc.pages![0].elements![3].visibleIf = "";
    expect(summaryProblems(doc)).toEqual([]);
  });
});

// S37: the editor offers no Logic on a summary, except to remove a condition it already has.
describe("logicOffered", () => {
  it("not on a summary", () => {
    expect(logicOffered(summary("cs_a"))).toBe(false);
    expect(logicOffered({ ...summary("cs_a"), visibleIf: " " })).toBe(false);
  });
  it("on a summary that already has a condition, so it can be removed", () => {
    expect(logicOffered({ ...summary("cs_a"), visibleIf: "{q} = 'y'" })).toBe(true);
    expect(logicOffered({ ...summary("cs_a"), requiredIf: "{q} = 'y'" })).toBe(true);
  });
  it("on every other element, as before", () => {
    expect(logicOffered({ type: "text", name: "q" })).toBe(true);
    expect(logicOffered({ type: "panel", name: "g", elements: [] })).toBe(true);
    expect(logicOffered({ type: "html", name: "i", html: "" })).toBe(true);
  });
});
