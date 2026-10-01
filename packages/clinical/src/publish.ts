// Checks before publishing (SGN-04, without sign-off yet): logic problems and failing test cases
// in every Question Set. They run in the browser (plan 2.1); publishing is blocked while any are found.
import type { ChapterJson } from "./doc";
import { logicProblems } from "./logic";
import { registerClinicalProperties } from "./properties";
import { chapterConditionProblems, combineChapters } from "./questionSets";
import { runTestCase, testCasesOf } from "./testcases";

export type PublishProblem = { set: string; message: string };

/** Every problem that blocks publishing, Question Set by Question Set, in order. */
export function publishProblems(sets: { name: string; doc: ChapterJson }[]): PublishProblem[] {
  registerClinicalProperties(); // test cases need the clinical properties and question types
  const out: PublishProblem[] = [];
  sets.forEach((s, i) => {
    const earlier = combineChapters(sets.slice(0, i));
    for (const m of chapterConditionProblems(s.doc, earlier)) out.push({ set: s.name, message: `Question Set condition: ${m}` });
    for (const p of logicProblems(s.doc)) out.push({ set: s.name, message: `${p.label}: ${p.message}` });
    for (const tc of testCasesOf(s.doc)) {
      if (!runTestCase(s.doc, tc).pass) out.push({ set: s.name, message: `Test case “${tc.name}” fails` });
    }
  });
  return out;
}
