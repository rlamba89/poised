// Showing or hiding a whole Question Set (chapter) by a condition (LOG-02). The condition sits
// in the chapter's own JSON as `chapterVisibleIf`, so the definition stays complete (OUT-01).
// It may test questions in earlier Question Sets and the patient, never its own questions.
import { ConditionRunner } from "survey-core";
import { allElements, edit, isValidExpression, pageTitle, pagesOf, referencesIn, VARIABLES, type ChapterJson } from "./doc";
import { unknownAnswers } from "./logic";

export const chapterConditionOf = (doc: ChapterJson): string | undefined =>
  typeof doc.chapterVisibleIf === "string" && doc.chapterVisibleIf.trim() ? doc.chapterVisibleIf : undefined;

export function setChapterCondition(doc: ChapterJson, expr: string | undefined): ChapterJson {
  return edit(doc, (d) => {
    if (expr) d.chapterVisibleIf = expr;
    else delete d.chapterVisibleIf;
  });
}

/**
 * The earlier Question Sets as one chapter, so the condition builder can offer their
 * questions, grouped by "Set › Page", and describe the condition in plain words.
 */
export function combineChapters(sets: { name: string; doc: ChapterJson }[]): ChapterJson {
  return {
    pages: sets.flatMap((s) => pagesOf(s.doc).map((p, i) => ({ ...p, title: `${s.name} › ${pageTitle(p, i)}` }))),
  };
}

/** Whether a Question Set is shown, given the answers so far (all sets) and the patient variables. */
export function isChapterShown(doc: ChapterJson, values: Record<string, unknown>): boolean {
  const expr = chapterConditionOf(doc);
  if (!expr) return true;
  return new ConditionRunner(expr).runValues({ ...values });
}

/**
 * Why a Question Set's condition can't work: unreadable, testing something not in an earlier set, or
 * testing an answer an earlier choice question doesn't have.
 */
export function chapterConditionProblems(doc: ChapterJson, earlier: ChapterJson): string[] {
  const expr = chapterConditionOf(doc);
  if (!expr) return [];
  if (!isValidExpression(expr)) return ["its condition can't be read"];
  const known = new Set(allElements(earlier).map(({ el }) => el.name));
  const own = new Set(allElements(doc).map(({ el }) => el.name));
  const refs = referencesIn(expr).filter((r) => !VARIABLES.includes(r));
  const out: string[] = [];
  if (refs.some((r) => own.has(r))) out.push("it tests one of its own questions");
  const missing = refs.filter((r) => !known.has(r) && !own.has(r));
  if (missing.length) out.push(`it tests something that isn't in an earlier Question Set (${missing.join(", ")})`);
  // An answer an earlier question doesn't have would hide the set from every patient (S24 verification 2).
  const gone = unknownAnswers(earlier, expr);
  if (gone.length) out.push(`it tests an answer that no longer exists (${gone.join("; ")})`);
  return out;
}

/** True when the condition tests only the patient and the viewer, so a one-set preview can decide it. */
export const testsOnlyThePatient = (expr: string) => referencesIn(expr).every((r) => VARIABLES.includes(r));
