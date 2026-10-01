import { describe, expect, it } from "vitest";
import { Model, type QuestionHtmlModel } from "survey-core";
import { registerClinicalProperties } from "./properties";
import { refreshClinicalSummaries, withClinicalSummaries } from "./summary";
import { stripClinicianOnly } from "./viewer";

registerClinicalProperties();

const chapter = {
  pages: [
    {
      name: "p_one",
      title: "Smoking",
      clinicalSummary: true,
      elements: [
        {
          type: "radiogroup",
          name: "q_smoke",
          title: "Do you smoke?",
          choices: [
            { value: "o_yes", text: "Yes", clinicalOutputs: [{ id: "a", codes: [], note: { text: "Current smoker", category: "Lifestyle" } }] },
            { value: "o_no", text: "No" },
          ],
        },
      ],
    },
    { name: "p_two", title: "Other", elements: [{ type: "text", name: "q_t" }] },
  ],
};

describe("clinical summary", () => {
  it("adds a clinician-only summary to pages that have it on", () => {
    const json = withClinicalSummaries(chapter);
    const els = json.pages[0].elements as { name: string; clinicianOnly?: boolean; title?: string }[];
    expect(els.at(-1)).toMatchObject({ name: "cs_p_one", clinicianOnly: true, title: "Smoking summary" });
    expect(json.pages[1].elements).toHaveLength(1);
    // Patients never receive it.
    expect(JSON.stringify(stripClinicianOnly(json))).not.toContain("cs_p_one");
  });

  it("lists the notes this page's answers produce", () => {
    const m = new Model(withClinicalSummaries(chapter));
    refreshClinicalSummaries(m);
    const notes = m.getQuestionByName("cs_notes_p_one") as QuestionHtmlModel;
    expect(notes.html).toContain("No disclosures");
    m.setValue("q_smoke", "o_yes");
    refreshClinicalSummaries(m);
    expect(notes.html).toContain("<li>Current smoker</li>");
  });
});
