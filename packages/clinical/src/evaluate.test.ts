import { describe, expect, it } from "vitest";
import sent from "../../../apps/api/internal/evaluate/testdata/evaluate.json";
import publishing from "../../../apps/api/internal/evaluate/testdata/publish_problems.json";
import reviewing from "../../../apps/api/internal/evaluate/testdata/shown.json";
import { checkPublish, evaluate, EvaluateInputError, shownSetsFor, type EvaluateInput, type PublishInput, type ShownInput } from "./evaluate";

const input = sent.input as unknown as EvaluateInput;
const [ABOUT, SLEEP, TRAVEL, NURSE] = input.sets.map((s) => s.id);

describe("evaluate: what the server works out when an HQ is sent (S41)", () => {
  const r = evaluate(input, "test");

  it("gives the same result as the shared fixture Go's client tests against", () => {
    expect(r).toEqual(sent.output);
  });

  it("names the evaluator's version", () => {
    expect(r.evaluator).toBe("test");
  });

  it("shows the sets the clinician's view shows: a set its condition hides isn't there", () => {
    expect(r.shown).toEqual([ABOUT, SLEEP, NURSE]);
    expect(r.sets.map((s) => s.id)).toEqual([ABOUT, SLEEP, NURSE]);
    expect(r.shown).not.toContain(TRAVEL);
  });

  it("gives each set's disclosures, scores and bands", () => {
    const about = r.sets.find((s) => s.id === ABOUT)!;
    expect(about.outputs.map((o) => o.output.id)).toEqual(["d_smoke", "d_heart", "d_chest"]);
    expect(about.scores).toEqual([{ questionId: "q_score", title: "Snore score", value: 5, band: { id: "b_high", label: "High" }, incomplete: false }]);
  });

  it("never counts answers to a clinician's set: the rules look at what the patient sends", () => {
    const nurse = r.sets.find((s) => s.id === NURSE)!;
    expect(nurse.outputs).toEqual([]);
    expect(r.suggestedAsa?.grade).toBe("III"); // not the nurse set's IV
  });

  it("gives the suggested ASA with its reasons", () => {
    expect(r.suggestedAsa).toEqual({
      grade: "III", emergency: false,
      reasons: [{ text: "Heart failure unspecified (I50.9) – Moderate reduction of ejection fraction", questionId: "q_heart", setId: ABOUT }],
    });
  });

  it("lists the red and amber flags", () => {
    expect(r.flags).toEqual([
      { level: "red", setId: ABOUT, questionId: "q_chest", text: "Do you get chest pain?: Yes" },
      { level: "amber", setId: SLEEP, questionId: "q_tired", text: "Are you tired in the day?: Yes" },
    ]);
  });

  it("gives the rules that fire, in order, with their actions", () => {
    expect(r.rules.map((f) => f.name)).toEqual(["High risk", "Snorers"]);
    expect(r.rules[0]).toEqual({
      id: "high_risk", name: "High risk", held: false,
      actions: [
        { kind: "task", title: "Book anaesthetic review", team: "anaesthetist" },
        { kind: "flag", name: "Anaesthetic review" },
        { kind: "requireAnaesthetistAsa" },
      ],
      line: "Rule fires: High risk (add the task “Book anaesthetic review” for the anaesthetist team; flag “Anaesthetic review”; require the anaesthetist's ASA before Ready for admission)",
    });
  });

  it("says whether each set the patient's own form shows is complete (for S44)", () => {
    expect(r.patientSets).toEqual([{ id: ABOUT, complete: true }, { id: SLEEP, complete: false }]);
  });

  it("with no rules (a version published before S40), no rule fires", () => {
    expect(evaluate({ ...input, rules: null }, "test").rules).toEqual([]);
  });

  it("a rule on the procedure sees the episode's procedure", () => {
    const rules = { noGradeAutoTriage: true, testCases: [], rules: [{ id: "p", name: "Knees", when: { asa: { op: ">=" as const, grade: 1 }, procedures: ["proc_knee"] }, then: [{ kind: "flag" as const, name: "Knee" }] }] };
    expect(evaluate({ ...input, rules, procedureId: "proc_knee" }, "test").rules.map((f) => f.name)).toEqual(["Knees"]);
    expect(evaluate({ ...input, rules, procedureId: null }, "test").rules).toEqual([]);
  });

  it("holds an auto-triage the guards stop, and says why (S42, test report 1, E-2)", () => {
    const rules = { noGradeAutoTriage: true, testCases: [], rules: [{ id: "low", name: "Low", when: { asa: { op: ">=" as const, grade: 1 } }, then: [{ kind: "autoTriage" as const, target: "ready_for_admission" as const }] }] };
    const [low] = evaluate({ ...input, rules }, "test").rules; // the fixture's answers have a red and an amber flag
    expect(low).toEqual({ id: "low", name: "Low", actions: [], held: true, heldReason: "flag", line: "Rule fires: Low (no auto-triage: a red or amber flag)" });
  });

  it("works out a score's points for age and sex from the patient context", () => {
    const sets = [{ id: "s", name: "S", audience: "patient", content: { pages: [{ name: "p", elements: [
      { type: "expression", name: "q_age", title: "Age points", expression: "iif({patientAge} > 50, 1, 0) + iif({patientSex} = 'male', 1, 0)", bands: [{ id: "b_0", label: "None", max: 0 }, { id: "b_1", label: "Some" }] },
    ] }] } }];
    const old = evaluate({ ...input, sets, answers: {}, patient: { age: 70, sex: "male" } }, "test");
    expect(old.sets[0].scores[0].value).toBe(2);
    const young = evaluate({ ...input, sets, answers: {}, patient: { age: 30, sex: "female" } }, "test");
    expect(young.sets[0].scores[0].value).toBe(0);
  });

  it("refuses an input that isn't a sent HQ", () => {
    expect(() => evaluate({} as EvaluateInput, "test")).toThrow(EvaluateInputError);
    expect(() => evaluate({ ...input, sets: [{ id: "x" }] } as unknown as EvaluateInput, "test")).toThrow(EvaluateInputError);
    expect(() => evaluate({ ...input, answers: [] } as unknown as EvaluateInput, "test")).toThrow(EvaluateInputError);
  });
});

describe("checkPublish: the publish checks the server runs (S41)", () => {
  it.each(publishing.cases)("$name", (c) => {
    expect(checkPublish(c.input as unknown as PublishInput)).toEqual(c.output);
  });

  it("refuses an input that isn't a version", () => {
    expect(() => checkPublish({} as PublishInput)).toThrow(EvaluateInputError);
  });
});

describe("shownSetsFor: the sets Complete review needs validated (S44)", () => {
  const review = reviewing.input as unknown as ShownInput;

  it("gives the same result as the shared fixture Go's client tests against", () => {
    expect(shownSetsFor(review)).toEqual(reviewing.output);
  });

  it("uses the answers that count now: the clinician's copy shows Travel and hides Sleep", () => {
    expect(shownSetsFor(review).shown).toEqual([ABOUT, TRAVEL, NURSE]);
    // With the patient's own answers (they smoke, snore 5 nights), Sleep is shown and Travel isn't.
    expect(shownSetsFor({ ...review, answers: input.answers }).shown).toEqual([ABOUT, SLEEP, NURSE]);
  });

  it("counts a clinician set's answers, which a later clinician set's condition may test", () => {
    const sets = [
      { id: "n1", name: "Nurse", audience: "clinician", content: { pages: [{ name: "p", elements: [{ type: "text", name: "q_bp", title: "BP" }] }] } },
      { id: "n2", name: "High BP", audience: "clinician", content: { chapterVisibleIf: "{q_bp} = 'high'", pages: [{ name: "p", elements: [{ type: "text", name: "q_more", title: "More" }] }] } },
    ];
    expect(shownSetsFor({ sets, answers: { n1: { q_bp: "high" } }, patient: {} }).shown).toEqual(["n1", "n2"]);
    expect(shownSetsFor({ sets, answers: { n1: { q_bp: "low" } }, patient: {} }).shown).toEqual(["n1"]);
  });

  it("sees the patient's age, as the episode page does", () => {
    const sets = [{ id: "s", name: "Older", audience: "patient", content: { chapterVisibleIf: "{patientAge} > 60", pages: [{ name: "p", elements: [{ type: "text", name: "q", title: "Q" }] }] } }];
    expect(shownSetsFor({ sets, answers: {}, patient: { age: 70 } }).shown).toEqual(["s"]);
    expect(shownSetsFor({ sets, answers: {}, patient: { age: 40 } }).shown).toEqual([]);
  });

  it("refuses an input that isn't an HQ's sets and answers", () => {
    expect(() => shownSetsFor({} as ShownInput)).toThrow(EvaluateInputError);
    expect(() => shownSetsFor({ ...review, answers: [] } as unknown as ShownInput)).toThrow(EvaluateInputError);
    expect(() => shownSetsFor({ ...review, patient: { age: "old" } } as unknown as ShownInput)).toThrow(EvaluateInputError);
  });
});
