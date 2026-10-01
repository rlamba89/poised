// Conditions as data (LOG-03/04/05/06): the Builder edits a group of rules joined by AND or OR,
// and reads expressions with SurveyJS's own parser, so the Builder and the Code tab always agree.
import { ArrayOperand, BinaryOperand, ConditionsParser, Const, FunctionOperand, UnaryOperand, Variable, type Operand } from "survey-core";
import {
  allElements, childrenOf, elementLabel, expressionsIn, isAnswerKind, isRepeating, isValidExpression, kindOf, optionsOf,
  pagesOf, readableExpression, textOf, unknownReferences, referencesIn, type ChapterJson, type ElementJson, type PageJson,
} from "./doc";

export type Op = "eq" | "neq" | "anyof" | "noneof" | "contains" | "notcontains" | "allof" | "gt" | "gte" | "lt" | "lte" | "empty" | "notempty";
export type Value = string | number | string[];
/** One test: `left` is a question ID (or `panel.<id>`, `<grid>.<row>`, a patient variable). */
export type Rule = { left: string; fn?: "age" | "band"; op: Op; value?: Value };
/** Rules joined by AND / OR, nested to any depth. `not` negates the group: "None of" (or) or "Not all of" (and). */
export type Group = { join: "and" | "or"; not?: boolean; items: (Rule | Group)[] };

export const isGroup = (x: Rule | Group): x is Group => "items" in x;

const SYMBOL: Record<Op, string> = {
  eq: "=", neq: "<>", gt: ">", gte: ">=", lt: "<", lte: "<=", contains: "contains", notcontains: "notcontains",
  anyof: "anyof", allof: "allof", noneof: "noneof", empty: "empty", notempty: "notempty",
};
const FROM_SURVEYJS: Record<string, Op> = {
  equal: "eq", notequal: "neq", greater: "gt", greaterorequal: "gte", less: "lt", lessorequal: "lte", contains: "contains",
  notcontains: "notcontains", anyof: "anyof", allof: "allof", noneof: "noneof", empty: "empty", notempty: "notempty",
};
const INVERSE: Partial<Record<Op, Op>> = {
  eq: "neq", neq: "eq", contains: "notcontains", notcontains: "contains", anyof: "noneof", noneof: "anyof",
  empty: "notempty", notempty: "empty", gt: "lte", lte: "gt", gte: "lt", lt: "gte",
};
const NO_VALUE: Op[] = ["empty", "notempty"];

// ---------------------------------------------------------------- writing

/** A rule is complete when it has a question and, unless it tests "answered", a value. */
export function isComplete(r: Rule): boolean {
  if (!r.left) return false;
  if (NO_VALUE.includes(r.op)) return true;
  if (Array.isArray(r.value)) return r.value.length > 0;
  return r.value !== undefined && r.value !== "" && !(typeof r.value === "number" && Number.isNaN(r.value));
}

/**
 * True when every rule, at any depth, is filled in. The Builder saves only then: a rule being
 * changed (say, to another question) leaves the saved condition as it was until it is complete.
 */
export function isAllComplete(g: Group): boolean {
  return g.items.every((x) => (isGroup(x) ? x.items.length > 0 && isAllComplete(x) : isComplete(x)));
}

/** The SurveyJS expression for a group; undefined when it has no complete rules ("Always"). */
export function formatLogic(g: Group): string | undefined {
  const parts = g.items
    .map((item) => {
      if (!isGroup(item)) return isComplete(item) ? formatRule(item) : undefined;
      const inner = formatLogic(item);
      if (!inner) return undefined;
      return item.not || completeCount(item) <= 1 ? inner : `(${inner})`;
    })
    .filter((x): x is string => !!x);
  if (!parts.length) return undefined;
  const joined = parts.join(` ${g.join} `);
  return g.not ? `!(${joined})` : joined;
}

const completeCount = (g: Group): number => g.items.filter((x) => (isGroup(x) ? !!formatLogic(x) : isComplete(x))).length;

function formatRule(r: Rule): string {
  const left = r.fn === "age" ? `age({${r.left}})` : r.fn === "band" ? `band('${r.left}')` : `{${r.left}}`;
  if (NO_VALUE.includes(r.op)) return `${left} ${SYMBOL[r.op]}`;
  return `${left} ${SYMBOL[r.op]} ${literal(r.value!)}`;
}

function literal(v: Value): string {
  if (Array.isArray(v)) return `[${v.map(literal).join(", ")}]`;
  if (typeof v === "number") return String(v);
  return `'${v.replace(/\\/g, "\\\\").replace(/'/g, "\\'")}'`;
}

// ---------------------------------------------------------------- reading

/**
 * The rules in an expression; an empty group for no expression, undefined for anything the
 * Builder can't show (arithmetic, other functions, negated groups): those stay in the Code tab.
 */
export function parseLogic(expr: string | undefined): Group | undefined {
  if (!expr?.trim()) return { join: "and", items: [] };
  const op = new ConditionsParser().parseExpression(expr);
  const item = op ? toItem(op) : undefined;
  if (!item) return undefined;
  // A negated group at the top is kept as a sub-group, so the top level stays plain All / Any.
  return isGroup(item) && !item.not ? item : { join: "and", items: [item] };
}

function toItem(op: Operand): Rule | Group | undefined {
  if (op instanceof BinaryOperand && op.isConjunction) {
    const join = op.operator === "or" ? "or" : "and";
    const flat = (x: Operand): (Rule | Group | undefined)[] =>
      x instanceof BinaryOperand && x.isConjunction && x.operator === op.operator ? [...flat(x.leftOperand), ...flat(x.rightOperand)] : [toItem(x)];
    const items = [...flat(op.leftOperand), ...flat(op.rightOperand)];
    return items.every(Boolean) ? { join, items: items as (Rule | Group)[] } : undefined;
  }
  if (op instanceof BinaryOperand) {
    const o = FROM_SURVEYJS[op.operator];
    const subject = subjectOf(op.leftOperand);
    const value = valueOf(op.rightOperand);
    if (!o || !subject || value === undefined) return undefined;
    return { ...subject, op: o, value };
  }
  if (op instanceof UnaryOperand) {
    if (op.operator === "negate") {
      const inner = toItem(op.expression);
      if (inner && !isGroup(inner) && INVERSE[inner.op]) return { ...inner, op: INVERSE[inner.op]! };
      if (inner && isGroup(inner) && !inner.not) return { ...inner, not: true };
      return undefined;
    }
    const o = FROM_SURVEYJS[op.operator];
    const subject = subjectOf(op.expression);
    return o && subject ? { ...subject, op: o } : undefined;
  }
  return undefined;
}

function subjectOf(op: Operand): Pick<Rule, "left" | "fn"> | undefined {
  if (op instanceof Variable) return { left: op.variable };
  if (op instanceof FunctionOperand && op.paramValues.length === 1) {
    const [p] = op.paramValues;
    if (op.functionName === "age" && p instanceof Variable) return { left: p.variable, fn: "age" };
    if (op.functionName === "band" && p instanceof Const && !(p instanceof Variable)) return { left: String(p.correctValue), fn: "band" };
  }
  return undefined;
}

function valueOf(op: Operand): Value | undefined {
  if (op instanceof ArrayOperand) {
    const values = op.values.map(valueOf);
    return values.every((v) => typeof v === "string" || typeof v === "number") ? values.map(String) : undefined;
  }
  if (op instanceof Const && !(op instanceof Variable)) {
    const v = op.correctValue;
    return typeof v === "string" || typeof v === "number" ? v : undefined;
  }
  return undefined;
}

/** The Builder shows any condition made of rules and groups, nested to any depth. */
export function builderCanShow(g: Group | undefined): g is Group {
  return !!g;
}

// ---------------------------------------------------------------- read-only (LOG-11)

/** "Read-only when C" is stored as SurveyJS's `enableIf = !(C)`. */
export const readOnlyToEnableIf = (expr: string | undefined) => (expr ? `!(${expr})` : undefined);

export function enableIfToReadOnly(enableIf: string | undefined): string | undefined {
  const m = enableIf ? /^\s*!\s*\(([\s\S]*)\)\s*$/.exec(enableIf) : null;
  return m && isValidExpression(m[1]) ? m[1].trim() : enableIf ? `!(${enableIf})` : undefined;
}

// ---------------------------------------------------------------- what a condition can test

export type SubjectType = "single" | "multi" | "number" | "date" | "text" | "answered";
export type Subject = {
  /** The Select value: `left`, or `band:<id>` for a score band. */
  id: string;
  left: string;
  fn?: "band";
  label: string;
  /** Where it is, for grouping the list: a page title or "Patient and viewer". */
  group: string;
  type: SubjectType;
  options?: { value: string; label: string }[];
};

export type ValueInput = "none" | "option" | "options" | "number" | "date" | "text";
export type OpChoice = { key: string; label: string; op: Op; fn?: "age"; input: ValueInput };

const ANSWERED: OpChoice[] = [
  { key: "notempty", label: "is answered", op: "notempty", input: "none" },
  { key: "empty", label: "is not answered", op: "empty", input: "none" },
];

/** The comparisons offered for each kind of question (LOG-03). */
export const OPS: Record<SubjectType, OpChoice[]> = {
  single: [
    { key: "eq", label: "is", op: "eq", input: "option" },
    { key: "neq", label: "is not", op: "neq", input: "option" },
    { key: "anyof", label: "is any of", op: "anyof", input: "options" },
    { key: "noneof", label: "is none of", op: "noneof", input: "options" },
    ...ANSWERED,
  ],
  multi: [
    { key: "contains", label: "includes", op: "contains", input: "option" },
    { key: "notcontains", label: "does not include", op: "notcontains", input: "option" },
    { key: "anyof", label: "includes any of", op: "anyof", input: "options" },
    { key: "allof", label: "includes all of", op: "allof", input: "options" },
    ...ANSWERED,
  ],
  number: [
    { key: "eq", label: "=", op: "eq", input: "number" },
    { key: "neq", label: "≠", op: "neq", input: "number" },
    { key: "gt", label: ">", op: "gt", input: "number" },
    { key: "gte", label: "≥", op: "gte", input: "number" },
    { key: "lt", label: "<", op: "lt", input: "number" },
    { key: "lte", label: "≤", op: "lte", input: "number" },
    ...ANSWERED,
  ],
  date: [
    { key: "lt", label: "is before", op: "lt", input: "date" },
    { key: "gt", label: "is after", op: "gt", input: "date" },
    { key: "eq", label: "is on", op: "eq", input: "date" },
    { key: "age:gte", label: "was at least … years ago", op: "gte", fn: "age", input: "number" },
    { key: "age:lt", label: "was less than … years ago", op: "lt", fn: "age", input: "number" },
    ...ANSWERED,
  ],
  text: [...ANSWERED, { key: "eq", label: "is exactly", op: "eq", input: "text" }, { key: "contains", label: "contains", op: "contains", input: "text" }],
  answered: ANSWERED,
};

/** The comparison a rule uses, in the list for its subject. */
export function opChoiceOf(subject: Subject, r: Rule): OpChoice | undefined {
  return OPS[subject.type].find((o) => o.op === r.op && (o.fn ?? undefined) === (r.fn === "age" ? "age" : undefined));
}

const PATIENT = "Patient and viewer";
const VARIABLE_SUBJECTS: Subject[] = [
  { id: "patientAge", left: "patientAge", label: "Patient's age", group: PATIENT, type: "number" },
  {
    id: "patientSex", left: "patientSex", label: "Patient's sex", group: PATIENT, type: "single",
    options: [{ value: "female", label: "Female" }, { value: "male", label: "Male" }],
  },
  {
    id: "viewer", left: "viewer", label: "Who is viewing", group: PATIENT, type: "single",
    options: [{ value: "patient", label: "Patient" }, { value: "clinician", label: "Clinician" }],
  },
];

/** What one element offers to conditions; `prefix` is "panel." inside the same repeating group. */
function subjectsOfElement(el: ElementJson, group: string, prefix = ""): Subject[] {
  const kind = kindOf(el);
  const label = textOf(el.title) || el.name;
  const left = prefix + el.name;
  const base = { id: left, left, label, group };
  const opts = () => optionsOf(el).map((o) => ({ value: o.value, label: o.text }));
  switch (kind) {
    case "yesno":
    case "selectone":
      return [{ ...base, type: "single", options: opts() }];
    case "selectmany":
      return [{ ...base, type: "multi", options: opts() }];
    case "number":
    case "rating":
      return [{ ...base, type: "number" }];
    case "calculation": {
      const out: Subject[] = [{ ...base, type: "number" }];
      const bands = bandsOf(el);
      if (bands.length) {
        out.push({ id: `band:${el.name}`, left: el.name, fn: "band", label: `${label} band`, group, type: "single", options: bands.map((b) => ({ value: b.id, label: b.label })) });
      }
      return out;
    }
    case "date":
      if (el.type === "text" && el.inputType === "date") return [{ ...base, type: "date" }];
      if (el.dateFormat === "y") return [{ ...base, type: "number" }];
      return [{ ...base, type: "answered" }];
    case "text":
      return [{ ...base, type: "text" }];
    case "grid": {
      const columns = optionsOfList(el.columns);
      return optionsOfList(el.rows).map((row) => ({
        id: `${left}.${row.value}`, left: `${left}.${row.value}`, label: `${label} › ${row.label}`, group, type: "single" as const, options: columns,
      }));
    }
    case "bmi":
      return [{ ...base, id: `${left}.bmi`, left: `${left}.bmi`, label: `${label} (BMI)`, type: "number" }];
    default:
      return isAnswerKind(kind) ? [{ ...base, type: "answered" }] : [];
  }
}

const optionsOfList = (list: unknown): { value: string; label: string }[] =>
  (Array.isArray(list) ? list : []).map((c) =>
    typeof c === "string" ? { value: c, label: c } : { value: String(c.value), label: textOf(c.text) || String(c.value) },
  );

/** A Calculation's score bands (CAL-02). */
export type Band = { id: string; label: string; max?: number; clinicalOutputs?: import("./types").ClinicalOutput[] };
export const bandsOf = (el: ElementJson): Band[] => (Array.isArray(el.bands) ? (el.bands as Band[]) : []);

/** The band a value falls in: the first whose maximum it doesn't exceed (the last band has none). */
export function bandFor(bands: unknown, value: unknown): Band | undefined {
  if (!Array.isArray(bands) || value === "" || value === null || value === undefined) return undefined;
  const n = Number(value);
  if (Number.isNaN(n)) return undefined;
  return (bands as Band[]).find((b) => b.max === undefined || b.max === null || n <= b.max);
}

/**
 * What a condition may test (LOG-05, LOG-08): questions that come earlier (for a page, on
 * earlier pages; for a skip rule, on its page or earlier), then the patient and the viewer.
 * Inside a repeating group, its own earlier questions are tested per entry (`{panel.<id>}`),
 * and questions inside other repeating groups can't be tested.
 */
export function subjectsFor(doc: ChapterJson, of: { element: string } | { page: string } | { upToPage: string }): Subject[] {
  const pages = pagesOf(doc);
  const titleOf = (p: PageJson) => textOf(p.title) || `Page ${pages.indexOf(p) + 1}`;
  const repeatingOf = repeatingParents(doc);
  const out: Subject[] = [];
  if ("element" in of) {
    const owner = repeatingOf.get(of.element);
    for (const { el, page } of allElements(doc)) {
      if (el.name === of.element) break;
      const group = repeatingOf.get(el.name);
      if (group && group !== owner) continue;
      out.push(...subjectsOfElement(el, titleOf(page), group ? "panel." : ""));
    }
  } else {
    const limit = pages.findIndex((p) => p.name === ("page" in of ? of.page : of.upToPage));
    const last = "page" in of ? limit - 1 : limit;
    for (const { el, page } of allElements(doc)) {
      if (pages.indexOf(page) > last || repeatingOf.has(el.name)) continue;
      out.push(...subjectsOfElement(el, titleOf(page)));
    }
  }
  return [...out, ...VARIABLE_SUBJECTS];
}

/** Each element inside a repeating group, with that group's name. */
function repeatingParents(doc: ChapterJson): Map<string, string> {
  const out = new Map<string, string>();
  const walk = (els: ElementJson[], group?: string) => {
    for (const el of els) {
      if (group) out.set(el.name, group);
      if (el.type === "panel" || isRepeating(el)) walk(childrenOf(el), isRepeating(el) ? el.name : group);
    }
  };
  for (const p of pagesOf(doc)) walk(p.elements ?? []);
  return out;
}

/** The subject a rule tests, looked up anywhere in the chapter (for describing it). */
export function subjectOfRule(doc: ChapterJson, r: Pick<Rule, "left" | "fn">): Subject | undefined {
  const variable = VARIABLE_SUBJECTS.find((s) => s.left === r.left);
  if (variable) return variable;
  const name = r.left.replace(/^panel\./, "").split(".")[0];
  const found = allElements(doc).find(({ el }) => el.name === name);
  if (!found) return undefined;
  const prefix = r.left.startsWith("panel.") ? "panel." : "";
  const all = subjectsOfElement(found.el, "", prefix);
  return r.fn === "band" ? all.find((s) => s.fn === "band") : all.find((s) => s.left === r.left && !s.fn);
}

// ---------------------------------------------------------------- plain words (LOG-06)

export type Part = { text: string; strong?: boolean };

/** A condition in plain words, e.g. "Do you smoke? **is** Yes **and** Patient's age **≥** 65". */
export function describeLogic(doc: ChapterJson, expr: string | undefined): { parts: Part[] } | { raw: string } | undefined {
  if (!expr?.trim()) return undefined;
  const g = parseLogic(expr);
  const parts = g && describeGroup(doc, g, false);
  return parts ? { parts } : { raw: readableExpression(doc, expr) };
}

function describeGroup(doc: ChapterJson, g: Group, nested: boolean): Part[] | undefined {
  const out: Part[] = [];
  for (const [i, item] of g.items.entries()) {
    if (i > 0) out.push({ text: g.not ? (g.join === "or" ? "or" : "and") : g.join, strong: !g.not });
    const parts = isGroup(item) ? describeGroup(doc, item, true) : describeRule(doc, item);
    if (!parts) return undefined;
    out.push(...parts);
  }
  if (g.not) return [{ text: g.join === "or" ? "none of" : "not all of", strong: true }, { text: "(" }, ...out, { text: ")" }];
  if (nested && g.items.length > 1) return [{ text: "(" }, ...out, { text: ")" }];
  return out;
}

function describeRule(doc: ChapterJson, r: Rule): Part[] | undefined {
  const subject = subjectOfRule(doc, r);
  if (!subject) return undefined;
  const choice = opChoiceOf(subject, r);
  const opText = choice?.label ?? SYMBOL[r.op];
  const parts: Part[] = [{ text: subject.label }];
  if (choice?.fn === "age") {
    const [before, after] = opText.split("…");
    return [...parts, { text: before.trim(), strong: true }, { text: String(r.value) }, { text: after.trim(), strong: true }];
  }
  parts.push({ text: opText, strong: true });
  if (r.value === undefined) return parts;
  const label = (v: string) => subject.options?.find((o) => o.value === v)?.label ?? (subject.type === "date" ? formatIsoDate(v) : v);
  const values = Array.isArray(r.value) ? r.value.map(label).join(r.op === "allof" ? " and " : " or ") : label(String(r.value));
  parts.push({ text: values });
  return parts;
}

const formatIsoDate = (v: string) => {
  const m = /^(\d{4})-(\d{2})-(\d{2})$/.exec(v);
  return m ? `${m[3]}/${m[2]}/${m[1]}` : v;
};

// ---------------------------------------------------------------- problems (LOG-08)

export type LogicProblem = { label: string; message: string; owner: string };

/**
 * Conditions that can't work: unreadable, testing a question that no longer exists, or
 * testing a question that comes later (forward and circular dependencies).
 */
export function logicProblems(doc: ChapterJson): LogicProblem[] {
  const order = allElements(doc).map(({ el }) => el.name);
  const pageIndex = new Map(allElements(doc).map(({ el, page }) => [el.name, pagesOf(doc).indexOf(page)]));
  const out: LogicProblem[] = [];
  for (const x of expressionsIn(doc)) {
    if (!isValidExpression(x.expr)) {
      out.push({ label: x.label, message: "its condition can't be read", owner: x.owner });
      continue;
    }
    const unknown = unknownReferences(doc, x.expr);
    if (unknown.length) out.push({ label: x.label, message: `it tests something that no longer exists (${unknown.join(", ")})`, owner: x.owner });
    if (x.key === "expression") continue;
    const refs = referencesIn(x.expr).filter((r) => order.includes(r));
    if (x.key === "trigger") {
      // A skip rule may test its own page and earlier pages.
      if (refs.some((r) => pageIndex.get(r)! > pagesOf(doc).indexOf(x.page))) {
        out.push({ label: x.label, message: "it tests a question on a later page", owner: x.owner });
      }
      continue;
    }
    const late = x.el
      ? refs.filter((r) => order.indexOf(r) >= order.indexOf(x.el!.name))
      : refs.filter((r) => pageIndex.get(r)! >= pagesOf(doc).indexOf(x.page));
    if (late.length) out.push({ label: x.label, message: "it tests a question that comes later or itself", owner: x.owner });
  }
  return out;
}
