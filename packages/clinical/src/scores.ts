// Scores (S38, SCR-01/02; CAL-01/02/03). A score totals its questions' option points and adds
// points from "Add N points when…" lines, picked from menus (Patient, Measurements, Other scores,
// and answers in this or an earlier Question Set): authors never type a formula (Rahul, 26 Oct).
//
// The form writes one SurveyJS expression, and reads it back:
//   iif(allAnswered('q_a', 'q_b', 'patientAge'), score('q_a', 'q_b') + iif({patientAge} > 50, 1, 0), '')
// A score with a missing input is **Incomplete**: it has no value, so no band, and it is never read
// as low risk (Rahul, 26 Oct). Old scores (`score('q_a', 'q_b')`) still total what is answered.
//
// A score may use questions from earlier Question Sets (by key), as a Question Set's condition can.
// Each set's survey is given the earlier shown sets' answers (`Earlier`): their values as survey
// variables (so `{q_a}` works), and their definitions, so score() and band() find options' points
// and bands. respondent.ts `earlierContexts` builds them for an HQ.
import { BinaryOperand, ConditionRunner, ConditionsParser, Const, FunctionOperand, Variable, type Operand, type SurveyModel } from "survey-core";
import { allElements, expressionsIn, namesIn, optionsOf, pagesOf, referencesIn, textOf, VARIABLES, type ChapterJson, type ElementJson } from "./doc";
import { bandFor, formatRule, isComplete, ruleOf, type Band, type Rule } from "./logic";

export type PointsLine = { points: number; rule: Rule };
export type ScoreForm = { kind: "score"; questions: string[]; lines?: PointsLine[] };

/** The function that makes a score Incomplete while one of its inputs has no answer. */
export const ALL_ANSWERED = "allAnswered";

/** A line is saved once its rule is complete and it has a number of points. */
export const isCompleteLine = (l: PointsLine) => isComplete(l.rule) && Number.isFinite(l.points);

/** The answers a score needs: its questions, then what its lines test (a band's Calculation, `q.bmi`…). */
export function scoreInputs(f: ScoreForm): string[] {
  return [...new Set([...f.questions, ...(f.lines ?? []).filter(isCompleteLine).map((l) => l.rule.left)])];
}

/**
 * The lines to save after an edit in the form, or undefined when the saved ones are unchanged: the
 * complete lines, whatever unfinished ones the author is still filling in (verification 1, F1: a
 * removed or changed line must be saved even while another is unfinished).
 */
export function linesToSave(draft: PointsLine[], saved: PointsLine[]): PointsLine[] | undefined {
  const complete = draft.filter(isCompleteLine);
  return JSON.stringify(complete) === JSON.stringify(saved) ? undefined : complete;
}

/**
 * The scored questions a score can add up, for the form's picker: each earlier set's, grouped
 * "Set › Page", then this set's ("This Question Set"), each key once. A key repeated in an older draft
 * (Publish refuses it) is listed where the survey finds it: this set's question, else the latest
 * earlier set's (applyEarlier; verification 1, F2).
 */
export function scoredQuestions(doc: ChapterJson, earlier: ChapterJson | undefined, calc: string): { group: string; items: { value: string; label: string }[] }[] {
  const isScored = (e: ElementJson) => optionsOf(e).some((o) => o.score !== undefined);
  const own = allElements(doc).filter(({ el }) => el.name !== calc && isScored(el));
  const taken = new Set(own.map(({ el }) => el.name));
  const item = (el: ElementJson) => ({ value: el.name, label: textOf(el.title) || el.name });
  const groups: { group: string; items: { value: string; label: string }[] }[] = [];
  for (const p of [...pagesOf(earlier ?? {})].reverse()) {
    const items = allElements({ pages: [p] }).filter(({ el }) => isScored(el) && !taken.has(el.name)).map(({ el }) => item(el));
    items.forEach((i) => taken.add(i.value));
    if (items.length) groups.unshift({ group: textOf(p.title), items });
  }
  if (own.length) groups.push({ group: "This Question Set", items: own.map(({ el }) => item(el)) });
  return groups;
}

const quoted = (names: string[]) => names.map((n) => `'${n}'`).join(", ");

/** The expression for a score form; "" while it has nothing to add up. */
export function formatScore(f: ScoreForm): string {
  const lines = (f.lines ?? []).filter(isCompleteLine);
  if (!f.questions.length && !lines.length) return "";
  const parts = [
    ...(f.questions.length ? [`score(${quoted(f.questions)})`] : []),
    ...lines.map((l) => `iif(${formatRule(l.rule)}, ${l.points}, 0)`),
  ];
  return `iif(${ALL_ANSWERED}(${quoted(scoreInputs(f))}), ${parts.join(" + ")}, '')`;
}

/**
 * The score form an expression was written by, or undefined when it's something else (a custom
 * formula). Only an exact round trip counts, so the form never shows a formula it didn't write.
 */
export function parseScore(expression: string | undefined): ScoreForm | undefined {
  const expr = (expression ?? "").trim();
  if (!expr.startsWith(`iif(${ALL_ANSWERED}(`)) return undefined;
  let op: Operand | null;
  try {
    op = new ConditionsParser().parseExpression(expr);
  } catch {
    return undefined;
  }
  if (!(op instanceof FunctionOperand) || op.functionName !== "iif" || op.paramValues.length !== 3) return undefined;
  const [, sum] = op.paramValues;
  const form: ScoreForm = { kind: "score", questions: [], lines: [] };
  const terms: Operand[] = [];
  const flatten = (x: Operand): void => {
    if (x instanceof BinaryOperand && x.operator === "plus") {
      flatten(x.leftOperand);
      flatten(x.rightOperand);
    } else terms.push(x);
  };
  flatten(sum);
  for (const t of terms) {
    if (!(t instanceof FunctionOperand)) return undefined;
    if (t.functionName === "score") {
      form.questions = t.paramValues.map((p) => (p instanceof Const && !(p instanceof Variable) ? String(p.correctValue) : ""));
      continue;
    }
    if (t.functionName !== "iif" || t.paramValues.length !== 3) return undefined;
    const [cond, points] = t.paramValues;
    const rule = ruleOf(cond);
    if (!rule || !(points instanceof Const) || points instanceof Variable) return undefined;
    form.lines!.push({ points: Number(points.correctValue), rule });
  }
  return formatScore(form) === expr ? form : undefined;
}

/** Whether a Calculation is a score that goes Incomplete while an input is missing. */
export const isGuardedScore = (expression: unknown) => typeof expression === "string" && expression.trim().startsWith(`iif(${ALL_ANSWERED}(`);

// ---------------------------------------------------------------- answers from earlier Question Sets

type Answers = Record<string, unknown>;

/**
 * What a Question Set's survey gets from the earlier sets shown: their answers by key (including
 * their Calculations' values), and their JSON (combineChapters), for options' points and bands.
 */
export type Earlier = { doc: ChapterJson; values: Answers };

const definitions = new WeakMap<object, Map<string, ElementJson>>();

/**
 * Gives a survey the earlier sets' answers: each becomes a survey variable, unless this set has a
 * question of that name (keys are unique in an HQ, but older HQs may repeat one: theirs wins).
 */
export function applyEarlier(model: SurveyModel, earlier: Earlier | undefined): void {
  if (!earlier) return;
  // This set's own questions, not the parts inside a BMI question or a repeating group (verification 2).
  const own = new Set(model.getAllQuestions(false, false, false).map((q) => q.name));
  const defs = new Map<string, ElementJson>();
  for (const { el } of allElements(earlier.doc)) if (!own.has(el.name)) defs.set(el.name, el);
  definitions.set(model, defs);
  for (const [name, value] of Object.entries(earlier.values)) if (!own.has(name)) model.setVariable(name, value);
  // score('q') and allAnswered('q') name their questions as text, so SurveyJS doesn't know they
  // depend on these variables: run every condition and Calculation again.
  model.runExpressions();
}

/** An earlier set's question, as score() and band() look it up when this set has no such question. */
export function earlierQuestion(survey: unknown, name: string): { el: ElementJson; value: unknown } | undefined {
  const el = survey ? definitions.get(survey as object)?.get(name) : undefined;
  return el ? { el, value: (survey as SurveyModel).getVariable(name) } : undefined;
}

/** The points an earlier set's choice question scores for its answer. */
export function earlierPoints(el: ElementJson, value: unknown): number {
  const chosen: unknown[] = Array.isArray(value) ? value : [value];
  return optionsOf(el).reduce((n, o) => n + (chosen.includes(o.value) ? Number(o.score ?? 0) : 0), 0);
}

/** Whether an answer is empty: nothing, "", an empty list or an empty object (shared with respondent.ts). */
export const isEmpty = (v: unknown) =>
  v === undefined || v === null || v === "" || (Array.isArray(v) && v.length === 0) || (typeof v === "object" && Object.keys(v as object).length === 0);

type Q = { name: string; value: unknown; isVisible: boolean; isParentVisible: boolean; getType(): string };

/**
 * Whether a score's input has an answer (`allAnswered`). A question in this set counts only while it's
 * shown (a question, group or page a condition hides isn't answered); an earlier set's answer counts
 * only if that set was shown (earlierContexts). "Unknown" isn't an answer to the patient's sex.
 * `q.bmi` is the BMI part of a BMI question's answer.
 */
export function inputAnswered(survey: SurveyModel | undefined, name: string): boolean {
  if (!survey) return false;
  const [head, ...path] = name.split(".");
  const q = survey.getQuestionByName(head) as unknown as Q | null;
  let value: unknown;
  if (q) {
    if (q.getType() !== "expression" && (!q.isVisible || !q.isParentVisible)) return false;
    value = q.value;
  } else {
    value = survey.getVariable(head);
    if (head === "patientSex" && value === "unknown") return false;
  }
  for (const p of path) value = value && typeof value === "object" ? (value as Answers)[p] : undefined;
  return !isEmpty(value);
}

/**
 * The band a Calculation's value is in, in a survey: a band's "also when" condition (`when`, e.g.
 * STOP-Bang's high-risk rule) picks it whatever the value; otherwise the first whose maximum the
 * value doesn't exceed. No value (an Incomplete score), no band.
 */
export function bandIn(survey: SurveyModel | undefined, bands: unknown, value: unknown): Band | undefined {
  if (!Array.isArray(bands) || isEmpty(value)) return undefined;
  const holds = (expr: string) => {
    if (!survey) return false;
    try {
      return new ConditionRunner(expr).runValues(survey.getFilteredValues(), survey.getFilteredProperties()) === true;
    } catch {
      return false;
    }
  };
  const also = [...(bands as Band[])].reverse().find((b) => typeof b.when === "string" && b.when.trim() && holds(b.when));
  return also ?? bandFor(bands, value);
}

/**
 * The keys a Question Set's Calculations use that it doesn't have itself: questions in earlier sets
 * (or missing ones, which the publish checks name). Empty for a set whose scores are its own.
 */
export function earlierKeysUsed(doc: ChapterJson): string[] {
  const own = namesIn(doc);
  const out = new Set<string>();
  for (const x of expressionsIn(doc)) {
    if (x.key !== "expression" && x.key !== "band") continue;
    for (const r of referencesIn(x.expr)) if (!own.has(r) && !VARIABLES.includes(r)) out.add(r);
  }
  return [...out];
}
