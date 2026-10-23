import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { poaSummary } from "./poa";
import { prefillChoiceText, prefillMarks, prefillSourceText, type PrefillEntry, type SetPrefill } from "./prefill";
import { registerClinicalProperties } from "./properties";
import { modelFor } from "./testcases";

registerClinicalProperties(); // clinicianOnly is a custom property (the web app registers it at start)

const jo = { name: "Jo Bloggs", age: 51, sex: "male" };
const day = (iso: string) => iso.slice(0, 10);

const health: ChapterJson = {
  pages: [
    {
      name: "p1", title: "Your health",
      elements: [
        { type: "radiogroup", name: "snoring", title: "Do you snore loudly?", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] },
        { type: "radiogroup", name: "problems", title: "Any problems with an anaesthetic?", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] },
        { type: "comment", name: "problems_details", title: "What happened?", visibleIf: "{problems} = 'yes'" },
        { type: "checkbox", name: "allergy", title: "Allergies", showOtherItem: true, choices: [{ value: "nuts", text: "Nuts" }] },
        { type: "comment", name: "nurse_note", title: "Nurse note", clinicianOnly: true },
      ],
    },
  ],
};

const entry = (value: unknown, more: Partial<PrefillEntry> = {}): PrefillEntry => ({
  fromForm: "f1", answeredAt: "2026-03-03T10:00:00Z", value, matchedBy: "key", questionnaire: "Demo pre-op HQ", ...more,
});

describe("prefillMarks (PRE-04)", () => {
  const entries = {
    snoring: entry("no"),
    problems: entry("yes"),
    problems_details: entry("Sick all night", { matchedBy: "wording" }),
    allergy: entry(["other"], { comment: "Plasters" }),
    nurse_note: entry("never pre-filled, but if it were"),
  };

  it("marks each pre-filled question shown, and says which the patient changed, with the pre-filled answer", () => {
    const patient = { snoring: "yes", problems: "yes", problems_details: "Sick all night", allergy: ["other"], "allergy-Comment": "Plasters" };
    const m = modelFor(health, "clinician", jo);
    m.data = patient;
    const marks = prefillMarks(m, entries, patient);
    expect(marks.map((x) => [x.questionId, x.changed])).toEqual([
      ["snoring", true],
      ["problems", false],
      ["problems_details", false],
      ["allergy", false],
    ]); // never a clinician-only question
    expect(marks[0]).toMatchObject({ title: "Do you snore loudly?", was: "No", now: "Yes" });
  });

  it("an 'Other' text the patient changed is a change", () => {
    const patient = { snoring: "no", problems: "no", allergy: ["other"], "allergy-Comment": "Latex" };
    const m = modelFor(health, "patient", jo);
    m.data = patient;
    expect(prefillMarks(m, entries, patient).find((x) => x.questionId === "allergy")?.changed).toBe(true);
  });

  it("only questions shown: a pre-filled answer its condition hides isn't marked", () => {
    const patient = { snoring: "no", problems: "no", problems_details: "Sick all night" };
    const m = modelFor(health, "clinician", jo);
    m.data = patient;
    expect(prefillMarks(m, entries, patient).map((x) => x.questionId)).not.toContain("problems_details");
  });

  it("an answer the patient cleared is a change", () => {
    const m = modelFor(health, "clinician", jo);
    m.data = {};
    expect(prefillMarks(m, { snoring: entry("no") }, {})).toMatchObject([{ questionId: "snoring", changed: true, was: "No", now: "no answer" }]);
  });

  it("nothing without a prefill map", () => {
    const m = modelFor(health, "clinician", jo);
    expect(prefillMarks(m, undefined, {})).toEqual([]);
  });
});

describe("the clinician's words", () => {
  it("names the questionnaire, the date and how it matched; another trust's hospital", () => {
    expect(prefillSourceText(entry("no"), day)).toBe("Pre-filled from Demo pre-op HQ, 2026-03-03 (matched by link)");
    expect(prefillSourceText(entry("no", { matchedBy: "wording", questionnaire: "Pre-op", otherTrust: true, hospital: "St Mary's Hospital" }), day)).toBe(
      "Pre-filled from Pre-op at St Mary's Hospital (another trust), 2026-03-03 (matched by same wording)",
    );
    expect(prefillSourceText(entry("no", { questionnaire: undefined }), day)).toBe("Pre-filled from an earlier HQ, 2026-03-03 (matched by link)");
  });

  it("says which button the patient pressed, or that staff pressed it with them", () => {
    const set = (choice: SetPrefill["choice"], by?: string): SetPrefill => ({ answers: {}, choice, choiceAt: "2026-04-01T09:00:00Z", choiceByName: by ?? null });
    expect(prefillChoiceText(set("still_correct"), day)).toBe(
      "The patient chose “Everything is still correct” on 2026-04-01, so these pre-filled answers weren't checked one by one.",
    );
    expect(prefillChoiceText(set("check_each"), day)).toBe("The patient chose “Check each answer” on 2026-04-01.");
    expect(prefillChoiceText(set("check_each", "Cara Clinician"), day)).toBe("Cara Clinician chose “Check each answer” with the patient on 2026-04-01.");
    expect(prefillChoiceText(set(null), day)).toBe("");
  });
});

describe("poaSummary with pre-filled answers", () => {
  const sets = [
    { id: "s1", name: "Your health", content: health },
    { id: "s2", name: "Hidden", content: { chapterVisibleIf: "{snoring} = 'never'", pages: [{ name: "p2", title: "Hidden page", elements: [{ type: "text", name: "h", title: "H?" }] }] } as ChapterJson },
  ];
  const prefill: Record<string, SetPrefill> = {
    s1: { answers: { snoring: entry("no"), problems: entry("yes") }, choice: "still_correct", choiceAt: "2026-04-01T09:00:00Z", choiceByName: null },
    s2: { answers: { h: entry("x") }, choice: null, choiceAt: null, choiceByName: null },
  };
  const patient = { s1: { snoring: "yes", problems: "yes", problems_details: "Bad" }, s2: { h: "x" } };

  it("lists each page's pre-filled answers, the patient's changes, and the set's button; only sets shown", () => {
    const clinician = { s1: { ...patient.s1, snoring: "no" } }; // the clinician put it back
    const poa = poaSummary(sets, clinician, "clinician", jo, patient, prefill);
    expect(poa.map((s) => s.id)).toEqual(["s1"]);
    expect(poa[0].prefill?.choice).toBe("still_correct");
    expect(poa[0].pages[0].prefilled.map((m) => [m.questionId, m.changed, m.was])).toEqual([
      ["snoring", true, "No"], // the patient's answer differs from the pre-filled one, whatever the clinician did after
      ["problems", false, "Yes"],
    ]);
  });

  it("the patient's tab marks the same", () => {
    const poa = poaSummary(sets.slice(0, 1), patient, "patient", jo, undefined, prefill);
    expect(poa[0].pages[0].prefilled.map((m) => m.questionId)).toEqual(["snoring", "problems"]);
  });

  it("a page whose only content is pre-filled answers is kept", () => {
    const quiet: ChapterJson = { pages: [{ name: "q1", title: "Quiet", elements: [{ type: "text", name: "t", title: "T?" }] }] };
    const poa = poaSummary([{ id: "q", name: "Quiet", content: quiet }], { q: { t: "x" } }, "patient", jo, undefined, { q: { answers: { t: entry("x") }, choice: null, choiceAt: null, choiceByName: null } });
    expect(poa[0].pages.map((p) => p.name)).toEqual(["q1"]);
  });
});
