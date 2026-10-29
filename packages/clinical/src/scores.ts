// Scores (S38, SCR-01/02; CAL-01/02/03). A score totals its questions' option points and adds
// points from "Add N points when…" lines, picked from menus (Patient, Measurements, Other scores,
// and answers in this or an earlier Question Set): authors never type a formula (Rahul, 26 Oct).
//
// The form writes one SurveyJS expression, and reads it back:
//   iif(allAnswered('q_a', 'q_b', 'patientAge'), score('q_a', 'q_b') + iif({patientAge} > 50, 1, 0), '')
// A score with a missing input is **Incomplete**: it has no value, so no band, and it is never read
// as low risk (Rahul, 26 Oct). An input a condition hides (the question, its page or its whole
// Question Set) adds 0 instead, and a line testing it doesn't fire (Rahul, 27 Oct; S44a). Old scores
// (`score('q_a', 'q_b')`) still total what is answered.
//
// A score may use questions from earlier Question Sets (by key), as a Question Set's condition can.
// Each set's survey is given the earlier shown sets' answers (`Earlier`): their values as survey
// variables (so `{q_a}` works), and their definitions, so score() and band() find options' points
// and bands. respondent.ts `earlierContexts` builds them for an HQ.
import { BinaryOperand, ConditionRunner, ConditionsParser, Const, FunctionOperand, Variable, type Operand, type SurveyModel } from "survey-core";
import { allElements, expressionsIn, namesIn, optionsOf, pagesOf, referencesIn, textOf, VARIABLES, type ChapterJson, type ElementJson } from "./doc";
import { bandFor, formatRule, isComplete, isGroup, parseLogic, pointsSubjects, repeatingParents, ruleOf, type Band, type Rule, type Subject } from "./logic";

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
  // A question inside a repeating group has an answer per row, which score() can't read: left out,
  // as pointsSubjects does (S44a, S38 verification 3).
  const repeating = new Set([...repeatingParents(doc).keys(), ...(earlier ? repeatingParents(earlier).keys() : [])]);
  const isScored = (e: ElementJson) => !repeating.has(e.name) && optionsOf(e).some((o) => o.score !== undefined);
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

/**
 * The parsed score forms, by expression text. A browser tab sees a handful; the evaluate Lambda's
 * process (S45) lives across many calls and trusts, so the cache is emptied once it holds this many.
 */
export const MAX_PARSED_SCORES = 500;
const parsed = new Map<string, ScoreForm | null>();

/** How many expressions the cache holds (for its test). */
export const parsedScoreCount = () => parsed.size;

/**
 * A score's expression as it runs in a survey now (S44a): an "Add N points when…" line that tests an
 * input a condition hides is left out, so it doesn't fire (a hidden answer adds 0, whatever its old
 * value). Anything the score form didn't write runs as it is. Called for every Calculation by
 * SurveyJS (questionTypes.ts registers it), so every view and the evaluate service agree.
 */
export function scoreAsShown(survey: SurveyModel, expression: string): string {
  if (!isGuardedScore(expression)) return expression;
  if (!parsed.has(expression)) {
    if (parsed.size >= MAX_PARSED_SCORES) parsed.clear();
    parsed.set(expression, parseScore(expression) ?? null);
  }
  const f = parsed.get(expression);
  if (!f?.lines?.length) return expression;
  const lines = f.lines.filter((l) => !inputHidden(survey, l.rule.left));
  if (lines.length === f.lines.length) return expression;
  return formatScore({ ...f, lines }) || "0";
}

// ---------------------------------------------------------------- answers from earlier Question Sets

type Answers = Record<string, unknown>;

/**
 * What a Question Set's survey gets from the earlier sets shown: their answers by key (including
 * their Calculations' values), and their JSON (combineChapters), for options' points and bands.
 */
export type Earlier = {
  doc: ChapterJson;
  values: Answers;
  /**
   * The earlier sets' question keys a condition hid (S44a): in a set its condition hid, or hidden
   * inside a shown set. A score input among them adds 0 and doesn't make the score Incomplete.
   */
  hidden?: string[];
};

const definitions = new WeakMap<object, Map<string, ElementJson>>();
const hiddenEarlier = new WeakMap<object, Set<string>>();

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
  hiddenEarlier.set(model, new Set((earlier.hidden ?? []).filter((n) => !own.has(n))));
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
 * Whether a condition hides a score's input (S44a, Rahul 27 Oct: it then adds 0 points): a question
 * in this set that it, its group or its page hides (a Calculation is always worked out, so never
 * hidden), or an earlier set's question its set or a condition there hid (`Earlier.hidden`).
 * `q.bmi` is the BMI question `q`.
 */
export function inputHidden(survey: SurveyModel | undefined, name: string): boolean {
  if (!survey) return false;
  const head = name.split(".")[0];
  const q = survey.getQuestionByName(head) as unknown as Q | null;
  if (q) return q.getType() !== "expression" && (!q.isVisible || !q.isParentVisible);
  return hiddenEarlier.get(survey)?.has(head) ?? false;
}

/**
 * Whether a score's input has an answer (`allAnswered`). An input a condition hides counts as
 * answered, adding 0 (S44a; inputHidden); a shown one left unanswered makes the score Incomplete.
 * An earlier set's answer counts only if that set was shown (earlierContexts). "Unknown" isn't an
 * answer to the patient's sex; for the ready-made STOP-Bang, neither is "other" (`otherSexUnscored`,
 * S44c, Rahul 28 Oct: its male point can't be decided, so it fails safe). `q.bmi` is the BMI part of
 * a BMI question's answer.
 */
export function inputAnswered(survey: SurveyModel | undefined, name: string, otherSexUnscored = false): boolean {
  if (!survey) return false;
  if (inputHidden(survey, name)) return true;
  const [head, ...path] = name.split(".");
  const q = survey.getQuestionByName(head) as unknown as Q | null;
  let value: unknown;
  if (q) {
    value = q.value;
  } else {
    value = survey.getVariable(head);
    if (head === "patientSex" && (value === "unknown" || (otherSexUnscored && value === "other"))) return false;
  }
  for (const p of path) value = value && typeof value === "object" ? (value as Answers)[p] : undefined;
  return !isEmpty(value);
}

/**
 * The band a Calculation's value is in, in a survey: a band's "also when" condition (`when`, e.g.
 * STOP-Bang's high-risk rule, or an author's own: S44a) picks it whatever the value; otherwise the
 * first whose maximum the value doesn't exceed. No value (an Incomplete score), no band. An input a
 * condition hides has no answer for "also when" (S44a), whatever its old value.
 */
export function bandIn(survey: SurveyModel | undefined, bands: unknown, value: unknown): Band | undefined {
  if (!Array.isArray(bands) || isEmpty(value)) return undefined;
  const holds = (expr: string) => {
    if (!survey) return false;
    try {
      const values = { ...survey.getFilteredValues() };
      for (const r of referencesIn(expr)) if (inputHidden(survey, r)) delete values[r];
      return new ConditionRunner(expr).runValues(values, survey.getFilteredProperties()) === true;
    } catch {
      return false;
    }
  };
  const also = [...(bands as Band[])].reverse().find((b) => typeof b.when === "string" && b.when.trim() && holds(b.when));
  return also ?? bandFor(bands, value);
}

// ---------------------------------------------------------------- "also when" on a band (S44a)

/**
 * What a band's "also when" can test (S44a, Rahul 27 Oct): the score itself, then what an "Add N
 * points when…" line can (Patient, Measurements, Other scores, answers), from the same menus.
 */
export function bandWhenSubjects(doc: ChapterJson, earlier: ChapterJson | undefined, calc: string): Subject[] {
  return [{ id: calc, left: calc, label: "The score", group: "This score", type: "number" }, ...pointsSubjects(doc, earlier, calc)];
}

/** A band's "also when" from its rules (all must hold), or undefined while none is complete. */
export function formatBandWhen(rules: Rule[]): string | undefined {
  const done = rules.filter(isComplete).map(formatRule);
  return done.length ? done.join(" and ") : undefined;
}

/**
 * The rules of a band's "also when" the editor can show, or undefined for one it can't (a ready-made
 * score's, which has "or"): those are shown as words, read-only.
 */
export function parseBandWhen(when: string | undefined): Rule[] | undefined {
  if (!when?.trim()) return [];
  let g;
  try {
    g = parseLogic(when);
  } catch {
    return undefined;
  }
  if (!g || g.not || (g.join !== "and" && g.items.length > 1) || g.items.some(isGroup)) return undefined;
  const rules = g.items as Rule[];
  return formatBandWhen(rules) === when ? rules : undefined;
}

/**
 * The inputs a score's bands' "also when" tests (S44a), besides the score itself: one left
 * unanswered makes the score Incomplete, as its own inputs do, since the band can't be decided.
 */
export function bandWhenInputs(calc: string, bands: unknown): string[] {
  if (!Array.isArray(bands)) return [];
  const out = new Set<string>();
  for (const b of bands as Band[]) {
    if (typeof b.when !== "string") continue;
    for (const m of b.when.matchAll(/\{(?:panel\.)?([A-Za-z0-9_.]+)\}/g)) out.add(m[1]);
    for (const m of b.when.matchAll(/\bband\('([^']*)'\)/g)) out.add(m[1]);
  }
  out.delete(calc);
  for (const v of ["viewer", "patientName"]) out.delete(v);
  return [...out];
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
