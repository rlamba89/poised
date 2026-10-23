import { describe, expect, it } from "vitest";
import type { ChapterJson } from "./doc";
import { chapterConditionProblems, combineChapters } from "./questionSets";

// A Question Set condition that tests an answer an earlier question doesn't have would hide the
// whole set from every patient (S24 verification 2, finding 1): the editor and Publish refuse it.
describe("chapterConditionProblems: answers that don't exist", () => {
  const first: ChapterJson = {
    pages: [{ name: "p_1", elements: [{ type: "radiogroup", name: "snoring", title: "Do you snore loudly?", choices: [{ value: "yes", text: "Yes" }, { value: "no", text: "No" }] }] }],
  };
  const earlier = combineChapters([{ name: "Sleep", doc: first }]);
  const second = (chapterVisibleIf: string): ChapterJson => ({ chapterVisibleIf, pages: [{ name: "p_2", elements: [] }] });
  it("flags a value the earlier question doesn't have", () => {
    expect(chapterConditionProblems(second("{snoring} = 'o_2'"), earlier)).toEqual(["it tests an answer that no longer exists (“Do you snore loudly?”: o_2)"]);
    expect(chapterConditionProblems(second("{snoring} anyof ['yes', 'gone']"), earlier)).toEqual(["it tests an answer that no longer exists (“Do you snore loudly?”: gone)"]);
  });
  it("accepts the question's own values and the patient variables", () => {
    expect(chapterConditionProblems(second("{snoring} = 'yes' and {patientSex} = 'female'"), earlier)).toEqual([]);
  });
});
