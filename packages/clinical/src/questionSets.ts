// Showing or hiding a whole Question Set (chapter) by a condition (LOG-02). The condition sits
// in the chapter's own JSON as `chapterVisibleIf`, so the definition stays complete (OUT-01).
// It may test questions in earlier Question Sets and the patient, never its own questions.
import { ConditionRunner } from "survey-core";
import { allElements, childrenOf, edit, isContainer, elementLabel, expressionsIn, isValidExpression, pageTitle, pagesOf, referencesIn, SUGGESTED_ASA, SUMMARY_TYPE, textOf, VARIABLES, type ChapterJson, type ElementJson } from "./doc";
import { describeLogic, unknownAnswers } from "./logic";
import type { Earlier } from "./scores";

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
 * testing an answer an earlier choice question doesn't have. Only a clinician set (`audience` other
 * than "patient") may test the suggested ASA (S39 answers 2): the patient's form has no disclosures.
 */
export function chapterConditionProblems(doc: ChapterJson, earlier: ChapterJson, audience?: string): string[] {
  const expr = chapterConditionOf(doc);
  if (!expr) return [];
  if (!isValidExpression(expr)) return ["its condition can't be read"];
  const known = new Set(allElements(earlier).map(({ el }) => el.name));
  const own = new Set(allElements(doc).map(({ el }) => el.name));
  const all = referencesIn(expr);
  const refs = all.filter((r) => !VARIABLES.includes(r) && r !== SUGGESTED_ASA);
  const out: string[] = [];
  if (all.includes(SUGGESTED_ASA) && (!audience || audience === "patient")) out.push("only a clinician's Question Set can test the suggested ASA");
  if (refs.some((r) => own.has(r))) out.push("it tests one of its own questions");
  const missing = refs.filter((r) => !known.has(r) && !own.has(r));
  if (missing.length) out.push(`it tests something that isn't in an earlier Question Set (${missing.join(", ")})`);
  // An answer an earlier question doesn't have would hide the set from every patient (S24 verification 2).
  const gone = unknownAnswers(earlier, expr);
  if (gone.length) out.push(`it tests an answer that no longer exists (${gone.join("; ")})`);
  return out;
}

/** The clinician-only elements of a set (S44a): all of a clinician set's, else those marked so (or inside one). */
export function clinicianOnlyIn(doc: ChapterJson, audience: string | undefined): Map<string, ElementJson> {
  const out = new Map<string, ElementJson>();
  const walk = (els: ElementJson[] | undefined, hidden: boolean) => {
    for (const el of els ?? []) {
      const h = hidden || el.clinicianOnly === true || el.type === SUMMARY_TYPE;
      if (h) out.set(el.name, el);
      if (isContainer(el)) walk(childrenOf(el), h);
    }
  };
  for (const p of pagesOf(doc)) walk(p.elements, audience !== undefined && audience !== "patient");
  return out;
}

const VERBS: Record<string, string> = { visibleIf: "shows only when", enableIf: "can be answered only when", requiredIf: "is required only when", choice: "shows only when", trigger: "runs only when" };

/**
 * Conditions the patient's form must work out that test something patients can't see (S44a, Rahul
 * 27 Oct: Publish refuses them): a patient Question Set's own condition, or one on a page, question
 * or option in it, testing a clinician-only element (of this set or an earlier one) or anything in
 * an earlier clinician Question Set. Go strips those before the form reaches the patient, so the
 * condition could never hold there, while the clinician's view would work it out. One message each,
 * e.g. "“Sleep follow-up” shows only when Sleep risk band is High, but patients can't see Sleep
 * risk. Make the score visible to patients, or move the condition to a clinician Question Set."
 */
export function patientConditionProblems(sets: { name: string; doc: ChapterJson; audience?: string }[]): { set: string; message: string }[] {
  const out: { set: string; message: string }[] = [];
  const unseen = new Map<string, { el: ElementJson; inClinicianSet: boolean }>();
  sets.forEach((s, i) => {
    const patientSet = !s.audience || s.audience === "patient";
    const own = clinicianOnlyIn(s.doc, s.audience);
    if (patientSet) {
      const here = new Map([...unseen, ...[...own].map(([k, el]) => [k, { el, inClinicianSet: false }] as const)]);
      const whole = combineChapters([...sets.slice(0, i), s]);
      const check = (holder: string, verb: string, expr: string, before: Map<string, { el: ElementJson; inClinicianSet: boolean }>) => {
        if (!isValidExpression(expr)) return; // named by the other checks
        const hit = [...new Set(referencesIn(expr))].filter((r) => before.has(r)).map((r) => before.get(r)!);
        if (!hit.length) return;
        const d = describeLogic(whole, expr);
        const words = !d ? expr : "parts" in d ? d.parts.map((p) => p.text).join(" ") : d.raw;
        const names = hit.map((h) => elementLabel(h.el)).join(", ");
        const kind = (el: ElementJson) => (el.type === "expression" ? "the score" : el.type === "panel" ? "the group" : "the question");
        const fix = hit.some((h) => h.inClinicianSet)
          ? "Move the condition to a clinician Question Set, or what it tests to a patient Question Set."
          : `Make ${hit.length > 1 ? "them" : kind(hit[0].el)} visible to patients, or move the condition to a clinician Question Set.`;
        const end = /[?!.]$/.test(names) ? "" : "."; // "can't see Concerns?." reads badly (test report 1, E3)
        out.push({ set: s.name, message: `${holder} ${verb} ${words}, but patients can't see ${names}${end} ${fix}` });
      };
      const cond = chapterConditionOf(s.doc);
      if (cond) check(`“${s.name}”`, VERBS.visibleIf, cond, unseen);
      for (const x of expressionsIn(s.doc)) {
        if (!(x.key in VERBS) || (x.el && own.has(x.el.name))) continue; // a clinician-only element's own condition is the clinician's
        const holder = x.key === "trigger" ? "A skip rule" : x.key === "choice" ? x.label.replace(/"([^"]*)"/g, "“$1”") : !x.el ? `Page “${textOf(x.page.title) || x.page.name}”` : `“${elementLabel(x.el)}”`;
        check(holder, VERBS[x.key], x.expr, here);
      }
    }
    for (const [k, el] of own) unseen.set(k, { el, inClinicianSet: !patientSet });
  });
  return out;
}

/** True when the condition tests only the patient and the viewer, so a one-set preview can decide it. */
export const testsOnlyThePatient = (expr: string) => referencesIn(expr).every((r) => VARIABLES.includes(r));

/**
 * What a set placed after `sets` gets from answers given by hand (S38: the Preview's earlier answers,
 * a test case's `earlier`), as earlierContexts does for an HQ, but without building each set's
 * survey: each given answer goes with the set holding its key, and a set its condition hides for
 * this patient gives nothing (`hidden` names it; test report 2, F-1).
 */
export function givenEarlier(
  sets: { name: string; doc: ChapterJson }[],
  given: Record<string, unknown>,
  patient: { age?: number; sex?: string },
  viewer: string,
): { earlier: Earlier; hidden: string[] } {
  // The suggested ASA can't be worked out from answers given by hand: 0 (none), as hqSets starts it,
  // so an earlier clinician set's "is at most II" is decided as for an episode with no grade (verification 1).
  let seen: Record<string, unknown> = { patientAge: patient.age, patientSex: patient.sex, viewer, [SUGGESTED_ASA]: 0 };
  const values: Record<string, unknown> = {};
  const hidden: string[] = [];
  const hiddenKeys: string[] = [];
  for (const s of sets) {
    const names = new Set(allElements(s.doc).map(({ el }) => el.name));
    const own = Object.fromEntries(Object.entries(given).filter(([k]) => names.has(k)));
    if (!isChapterShown(s.doc, seen)) {
      hidden.push(s.name);
      hiddenKeys.push(...names); // they add 0 to a score (S44a)
      continue;
    }
    seen = { ...seen, ...own };
    Object.assign(values, own);
  }
  return { earlier: { doc: combineChapters(sets), values, ...(hiddenKeys.length ? { hidden: hiddenKeys } : {}) }, hidden };
}
