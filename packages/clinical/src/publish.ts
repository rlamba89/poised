// Checks before publishing (SGN-04, without sign-off yet): logic problems (a score may use earlier
// Question Sets' questions, S38), two clinical summaries
// covering the same Group, Section or page (S32), a video without a YouTube or Vimeo link or a length
// (S34), and failing test cases in every Question Set.
// They run in the browser (plan 2.1); publishing is blocked while any are found.
// The question-key checks (S23) are Go's: the editor shows them from GET …/question-keys.
import type { ChapterJson } from "./doc";
import { logicProblems } from "./logic";
import { registerClinicalProperties } from "./properties";
import { chapterConditionProblems, combineChapters } from "./questionSets";
import { readyScoreProblems } from "./ready";
import { summaryProblems } from "./summary";
import { runTestCase, testCasesOf } from "./testcases";
import { videoProblems } from "./video";

export type PublishProblem = { set: string; message: string };

/** Every problem that blocks publishing, Question Set by Question Set, in order. */
export function publishProblems(sets: { name: string; doc: ChapterJson }[]): PublishProblem[] {
  registerClinicalProperties(); // test cases need the clinical properties and question types
  const out: PublishProblem[] = [];
  sets.forEach((s, i) => {
    const earlier = combineChapters(sets.slice(0, i));
    const later = combineChapters(sets.slice(i + 1));
    for (const m of chapterConditionProblems(s.doc, earlier)) out.push({ set: s.name, message: `Question Set condition: ${m}` });
    // A score may use earlier sets' questions, never later ones (S38).
    for (const p of logicProblems(s.doc, { earlier, later })) out.push({ set: s.name, message: `${p.label}: ${p.message}` });
    for (const m of summaryProblems(s.doc)) out.push({ set: s.name, message: m });
    for (const m of readyScoreProblems(s.doc)) out.push({ set: s.name, message: m });
    for (const m of videoProblems(s.doc)) out.push({ set: s.name, message: m });
    for (const tc of testCasesOf(s.doc)) {
      if (!runTestCase(s.doc, tc, sets.slice(0, i)).pass) out.push({ set: s.name, message: `Test case “${tc.name}” fails` });
    }
  });
  return out;
}
