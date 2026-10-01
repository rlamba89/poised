// Settings for each type of question, kept as plain SurveyJS JSON (QT-04/05/06, VAL-02/03,
// STR-04, OPT-04/06). Each setting reads from and writes to the SurveyJS properties
// themselves (validators, masks, input types), so there is never a second copy of a rule.
import {
  childrenOf, edit, findElement, isContainer, kindOf, optionsOf, type ChapterJson, type ChoiceJson,
  type ElementJson, type Kind,
} from "./doc";
import { newId } from "./ids";
import type { ClinicalOutput } from "./types";

type Validator = { type: string; [key: string]: unknown };
const validatorsOf = (el: ElementJson): Validator[] => (Array.isArray(el.validators) ? (el.validators as Validator[]) : []);

// ---------------------------------------------------------------- text format (QT-04)

export const TEXT_FORMATS = {
  none: { label: "Any text" },
  email: { label: "Email address", message: "Please enter a valid email address." },
  phone: { label: "Phone number", regex: "^\\+?[0-9 ()-]{7,20}$", message: "Please enter a valid phone number." },
  nhs: { label: "NHS number", regex: "^\\d{3} ?\\d{3} ?\\d{4}$", message: "Please enter a 10-digit NHS number." },
  postcode: { label: "UK postcode", regex: "^[A-Za-z]{1,2}\\d[A-Za-z\\d]? ?\\d[A-Za-z]{2}$", message: "Please enter a valid postcode." },
  custom: { label: "Custom pattern", message: "Please check the format." },
} as const;
export type TextFormat = keyof typeof TEXT_FORMATS;

/** The format a text question checks, read back from its validators. */
export function textFormatOf(el: ElementJson): { format: TextFormat; pattern?: string; message?: string } {
  const v = validatorsOf(el).find((x) => x.type === "email" || x.type === "regex");
  if (!v) return { format: "none" };
  const message = typeof v.text === "string" ? v.text : undefined;
  if (v.type === "email") return { format: "email", message };
  // "custom" is marked, because a custom pattern may start as a copy of a known one.
  if (el.textFormat === "custom") return { format: "custom", pattern: String(v.regex ?? ""), message };
  const known = (Object.entries(TEXT_FORMATS) as [TextFormat, { regex?: string }][]).find(([, f]) => f.regex === v.regex);
  return known ? { format: known[0], message } : { format: "custom", pattern: String(v.regex ?? ""), message };
}

/**
 * The validators and input type for a format. Switching to "custom" starts from the current
 * format's pattern (an NHS number rule becomes an editable pattern, not nothing).
 */
export function setTextFormat(el: ElementJson, format: TextFormat, pattern?: string, message?: string): Partial<ElementJson> {
  if (format === "custom" && pattern === undefined) {
    const current = textFormatOf(el);
    pattern = current.pattern ?? (TEXT_FORMATS[current.format] as { regex?: string }).regex ?? "";
  }
  const others = validatorsOf(el).filter((x) => x.type !== "email" && x.type !== "regex");
  const f = TEXT_FORMATS[format] as { regex?: string; message?: string };
  const text = message || f.message;
  let v: Validator | undefined;
  if (format === "email") v = { type: "email", text };
  // A custom pattern is kept even while empty (it then checks nothing), so the choice sticks.
  else if (format === "custom") v = { type: "regex", regex: pattern ?? "", text };
  else if (f.regex) v = { type: "regex", regex: f.regex, text };
  const validators = v ? [...others, v] : others;
  return {
    validators: validators.length ? validators : undefined,
    inputType: format === "email" ? "email" : format === "phone" ? "tel" : undefined,
    textFormat: format === "custom" ? "custom" : undefined,
  };
}

// ---------------------------------------------------------------- number (QT-06, VAL-02, VAL-03)

export type NumberSettings = {
  min?: number;
  max?: number;
  /** Decimal places allowed; undefined allows any. */
  decimals?: number;
  unit?: string;
  /** Outside this range the patient sees a warning but can carry on (VAL-03). */
  warnMin?: number;
  warnMax?: number;
  warnText?: string;
};

const isWarning = (v: Validator) => v.notificationType === "warning";

export function numberSettingsOf(el: ElementJson): NumberSettings {
  const range = validatorsOf(el).find((v) => v.type === "numeric" && !isWarning(v));
  const warn = validatorsOf(el).find((v) => v.type === "numeric" && isWarning(v));
  const mask = el.maskSettings as { precision?: number } | undefined;
  return {
    min: range?.minValue as number | undefined,
    max: range?.maxValue as number | undefined,
    decimals: el.maskType === "numeric" ? (mask?.precision ?? 0) : undefined,
    unit: el.unit as string | undefined,
    warnMin: warn?.minValue as number | undefined,
    warnMax: warn?.maxValue as number | undefined,
    warnText: warn?.text as string | undefined,
  };
}

/**
 * Decimal places use SurveyJS's numeric input mask (the box won't take more); with no limit
 * the question stays a number input. Ranges are numeric validators: an error, and a warning.
 */
export function setNumberSettings(el: ElementJson, s: NumberSettings): Partial<ElementJson> {
  const others = validatorsOf(el).filter((v) => v.type !== "numeric");
  const has = (n: number | undefined) => typeof n === "number" && !Number.isNaN(n);
  const validators: Validator[] = [...others];
  if (has(s.min) || has(s.max)) {
    const range = [has(s.min) && `at least ${s.min}`, has(s.max) && `at most ${s.max}`].filter(Boolean).join(" and ");
    validators.push(clean({ type: "numeric", minValue: s.min, maxValue: s.max, text: `Please enter a number ${range}.` }));
  }
  if (has(s.warnMin) || has(s.warnMax)) {
    validators.push(clean({ type: "numeric", minValue: s.warnMin, maxValue: s.warnMax, notificationType: "warning", text: s.warnText || "That looks unusual. Please check it." }));
  }
  const masked = has(s.decimals);
  return {
    inputType: masked ? undefined : "number",
    maskType: masked ? "numeric" : undefined,
    maskSettings: masked ? { precision: s.decimals, allowNegativeValues: !has(s.min) || s.min! < 0 } : undefined,
    validators: validators.length ? validators : undefined,
    unit: s.unit || undefined,
  };
}

function clean<T extends Record<string, unknown>>(o: T): T {
  for (const k of Object.keys(o)) if (o[k] === undefined || (typeof o[k] === "number" && Number.isNaN(o[k]))) delete o[k];
  return o;
}

// ---------------------------------------------------------------- date (QT-05)

export type DateFormat = "dmy" | "my" | "y";
export type DateRange = "any" | "past" | "future";
export type DateSettings = { format: DateFormat; range: DateRange; multiple: boolean };

export const DATE_FORMATS: Record<DateFormat, string> = { dmy: "Day, month and year", my: "Month and year", y: "Year only" };

export function dateSettingsOf(el: ElementJson): DateSettings {
  const field = el.type === "matrixdynamic" ? ((el.columns as ElementJson[] | undefined)?.[0] ?? el) : el;
  const format: DateFormat = field.dateFormat === "y" ? "y" : field.inputType === "month" ? "my" : "dmy";
  const range: DateRange = field.maxValueExpression ? "past" : field.minValueExpression ? "future" : "any";
  return { format, range, multiple: el.type === "matrixdynamic" };
}

/** The input for one date: a date or month picker, or a year as a number. Never before 1900. */
function dateField(format: DateFormat, range: DateRange): Record<string, unknown> {
  const now = format === "y" ? "currentYear()" : "today()";
  const limits = range === "past" ? { maxValueExpression: now } : range === "future" ? { minValueExpression: now } : {};
  if (format === "y") return { inputType: "number", dateFormat: "y", min: 1900, ...limits };
  if (format === "my") return { inputType: "month", min: "1900-01", ...limits };
  return { inputType: "date", min: "1900-01-01", ...limits };
}

const KEEP = ["name", "title", "description", "isRequired", "clinicianOnly", "visibleIf", "requiredIf", "enableIf", "clinicalOutputs", "requiredErrorText", "descriptionLocation"];

/** Rebuilds a Date question for new settings. Several dates are a list with one date per row. */
export function setDateSettings(doc: ChapterJson, name: string, s: DateSettings): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const kept = Object.fromEntries(Object.entries(loc.el).filter(([k]) => KEEP.includes(k)));
    const field = dateField(s.format, s.range);
    const next: ElementJson = s.multiple
      ? {
          ...(kept as ElementJson), type: "matrixdynamic", dateList: true, rowCount: 1, minRowCount: 1, addRowText: "Add another date",
          removeRowText: "Remove", columns: [{ name: "date", title: " ", cellType: "text", ...field }],
        }
      : { ...(kept as ElementJson), type: "text", ...field };
    loc.siblings[loc.index] = next;
  });
}

// ---------------------------------------------------------------- options (QT-02/03, OPT-04/06)

export const SPECIAL_OPTIONS = { dontknow: "Don't know", refuse: "Prefer not to say" } as const;
export type SpecialOption = keyof typeof SPECIAL_OPTIONS;

/**
 * Adds or removes a one-click option. It is an ordinary option (so it can carry disclosures and
 * satisfies a required question, VAL-05), exclusive in Select Many, and kept at the end.
 */
export function setSpecialOption(doc: ChapterJson, name: string, special: SpecialOption, on: boolean): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const choices = optionsOf(loc.el).filter((c) => c.special !== special);
    if (on) {
      const value = newId("o", 4, (id) => choices.some((c) => c.value === id));
      const option: ChoiceJson = { value, text: SPECIAL_OPTIONS[special], special };
      if (kindOf(loc.el) === "selectmany") option.isExclusive = true;
      choices.push(option);
    }
    loc.el.choices = orderOptions(choices);
  });
}

/** Ordinary options first, then "None of the above", then the one-click options. */
export function orderOptions(choices: ChoiceJson[]): ChoiceJson[] {
  const rank = (c: ChoiceJson) => (c.special ? 2 : c.isExclusive ? 1 : 0);
  return [...choices].sort((a, b) => rank(a) - rank(b));
}

/** Select One as buttons or a dropdown; Select Many as checkboxes or a searchable list (QT-02, QT-07). */
export type ChoiceDisplay = "buttons" | "list";
export const choiceDisplayOf = (el: ElementJson): ChoiceDisplay => (el.type === "dropdown" || el.type === "tagbox" ? "list" : "buttons");

export function setChoiceDisplay(el: ElementJson, display: ChoiceDisplay): Partial<ElementJson> {
  const many = kindOf(el) === "selectmany";
  if (display === "list") return { type: many ? "tagbox" : "dropdown", searchEnabled: true, colCount: undefined };
  return { type: many ? "checkbox" : "radiogroup", searchEnabled: undefined, placeholder: undefined };
}

// ---------------------------------------------------------------- repeating groups (STR-04)

export type RepeatSettings = { min: number; max?: number; addText: string };

export const repeatSettingsOf = (el: ElementJson): RepeatSettings => ({
  min: Number(el.minPanelCount ?? 1),
  max: el.maxPanelCount === undefined ? undefined : Number(el.maxPanelCount),
  addText: typeof el.panelAddText === "string" ? el.panelAddText : "Add another",
});

/**
 * Turns a group into a repeating group (a SurveyJS dynamic panel) or back. Conditions between
 * questions inside it are tested per entry, so `{q}` becomes `{panel.q}` inside, and back.
 */
export function setRepeating(doc: ChapterJson, name: string, settings: RepeatSettings | undefined): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc || !isContainer(loc.el)) return;
    const el = loc.el;
    const children = childrenOf(el);
    const inside = new Set<string>();
    const collect = (els: ElementJson[]) => els.forEach((e) => (inside.add(e.name), isContainer(e) && collect(childrenOf(e))));
    collect(children);
    const toPanel = settings !== undefined;
    const rename = (expr: string) =>
      toPanel
        ? expr.replace(/\{([A-Za-z0-9_]+)((?:\.[A-Za-z0-9_]+)?)\}/g, (all, q: string, sub: string) => (inside.has(q) ? `{panel.${q}${sub}}` : all))
        : expr.replace(/\{panel\.([A-Za-z0-9_]+)((?:\.[A-Za-z0-9_]+)?)\}/g, "{$1$2}");
    const fix = (e: ElementJson) => {
      for (const key of ["visibleIf", "requiredIf", "enableIf", "expression"] as const) if (typeof e[key] === "string") e[key] = rename(e[key] as string);
      if (isContainer(e)) childrenOf(e).forEach(fix);
    };
    children.forEach(fix);
    if (toPanel) {
      delete el.elements;
      Object.assign(el, {
        type: "paneldynamic", templateElements: children, minPanelCount: settings.min || undefined, maxPanelCount: settings.max,
        panelCount: Math.max(1, settings.min), panelAddText: settings.addText || undefined, panelRemoveText: "Remove",
      });
      if (settings.max === undefined) delete el.maxPanelCount;
      if (!settings.min) delete el.minPanelCount;
    } else {
      for (const k of ["templateElements", "minPanelCount", "maxPanelCount", "panelCount", "panelAddText", "panelRemoveText"]) delete el[k];
      Object.assign(el, { type: "panel", elements: children });
    }
  });
}

// ---------------------------------------------------------------- changing type (editor comfort)

/** The kinds an element can become without losing its options or answers' meaning. */
export function convertibleKinds(el: ElementJson): Kind[] {
  const kind = kindOf(el);
  if (kind === "selectone" || kind === "selectmany" || kind === "yesno") {
    const two = optionsOf(el).filter((o) => !o.isExclusive && !o.special).length === 2;
    return (["yesno", "selectone", "selectmany"] as Kind[]).filter((k) => k !== kind && (k !== "yesno" || two));
  }
  if (kind === "text" || kind === "number" || kind === "date") return (["text", "number", "date"] as Kind[]).filter((k) => k !== kind);
  return [];
}

/**
 * Changes a question's type, keeping its text, options, disclosures and logic. Conditions that
 * test it switch between "is" (one answer) and "includes" (several answers).
 */
export function convertKind(doc: ChapterJson, name: string, to: Kind): ChapterJson {
  const from = findElement(doc, name)?.el;
  if (!from || !convertibleKinds(from).includes(to)) return doc;
  const wasMany = kindOf(from) === "selectmany";
  return edit(doc, (d) => {
    const loc = findElement(d, name)!;
    const el = loc.el;
    const keep = Object.fromEntries(Object.entries(el).filter(([k]) => [...KEEP, "choices"].includes(k))) as ElementJson;
    let next: ElementJson;
    switch (to) {
      case "yesno":
        next = { ...keep, type: "radiogroup", yesNo: true, colCount: 0 };
        break;
      case "selectone":
        next = { ...keep, type: "radiogroup" };
        break;
      case "selectmany":
        next = { ...keep, type: "checkbox" };
        break;
      case "text":
        next = { ...keep, type: "text" };
        break;
      case "number":
        next = { ...keep, type: "text", inputType: "number" };
        break;
      default:
        next = { ...keep, type: "text", inputType: "date", min: "1900-01-01" };
    }
    if (Array.isArray(next.choices)) {
      // Only Select Many has exclusive options; there, the one-click options are exclusive again.
      next.choices = (next.choices as ChoiceJson[]).map(({ isExclusive, ...c }) =>
        to === "selectmany" && (isExclusive || c.special) ? { ...c, isExclusive: true } : c,
      );
    }
    loc.siblings[loc.index] = next;
    const isMany = to === "selectmany";
    if (wasMany === isMany || !Array.isArray(next.choices)) return;
    // `{q} = 'o'` ↔ `{q} contains 'o'` wherever this question is tested.
    const swap = (expr: string) =>
      expr.replace(new RegExp(`(\\{(?:panel\\.)?${name}\\}\\s*)(=|==|<>|!=|contains|notcontains)`, "g"), (all, left: string, op: string) => {
        const not = op === "<>" || op === "!=" || op === "notcontains";
        return left + (isMany ? (not ? "notcontains" : "contains") : not ? "<>" : "=");
      });
    const fix = (e: ElementJson) => {
      for (const key of ["visibleIf", "requiredIf", "enableIf"] as const) if (typeof e[key] === "string") e[key] = swap(e[key] as string);
      for (const c of e.choices ?? []) if (typeof c === "object" && c.visibleIf) c.visibleIf = swap(c.visibleIf);
      if (isContainer(e)) childrenOf(e).forEach(fix);
    };
    for (const p of d.pages ?? []) {
      if (p.visibleIf) p.visibleIf = swap(p.visibleIf);
      (p.elements ?? []).forEach(fix);
    }
  });
}

// ---------------------------------------------------------------- calculations (CAL-01/02/03)

/** The formulas the Calculation form builds; anything else is a custom formula. */
export type CalcForm =
  | { kind: "score"; questions: string[] }
  | { kind: "years"; question: string }
  | { kind: "bmi"; height: string; weight: string }
  | { kind: "custom"; expression: string };

/**
 * The form for a Calculation. `kind` is the type the author chose (stored as `calcKind`): an empty
 * formula can't say which it is, and a custom formula may look like one the form builds.
 */
export function calcFormOf(expression: string | undefined, kind?: string): CalcForm {
  const expr = (expression ?? "").trim();
  if (kind === "custom") return { kind: "custom", expression: expr };
  let m = /^score\(([^)]*)\)$/.exec(expr);
  if (m && (!kind || kind === "score")) return { kind: "score", questions: [...m[1].matchAll(/'([^']*)'/g)].map((x) => x[1]) };
  m = /^age\(\{([A-Za-z0-9_]+)\}\)$/.exec(expr);
  if (m && (!kind || kind === "years")) return { kind: "years", question: m[1] };
  m = /^bmi\(\{([A-Za-z0-9_]+)\},\s*\{([A-Za-z0-9_]+)\}\)$/.exec(expr);
  if (m && (!kind || kind === "bmi")) return { kind: "bmi", height: m[1], weight: m[2] };
  if (!expr) return emptyCalc(kind);
  return { kind: "custom", expression: expr };
}

/** A new, empty form of a kind (a score total when none is given). */
export function emptyCalc(kind?: string): CalcForm {
  switch (kind) {
    case "years":
      return { kind: "years", question: "" };
    case "bmi":
      return { kind: "bmi", height: "", weight: "" };
    case "custom":
      return { kind: "custom", expression: "" };
    default:
      return { kind: "score", questions: [] };
  }
}

/** The JSON for a form: the formula, and the chosen kind so an unfinished one keeps its type. */
export const calcPatch = (f: CalcForm): Partial<ElementJson> => ({ expression: formatCalc(f), calcKind: f.kind });

export function formatCalc(f: CalcForm): string {
  switch (f.kind) {
    case "score":
      return f.questions.length ? `score(${f.questions.map((q) => `'${q}'`).join(", ")})` : "";
    case "years":
      return f.question ? `age({${f.question}})` : "";
    case "bmi":
      return f.height && f.weight ? `bmi({${f.height}}, {${f.weight}})` : "";
    case "custom":
      return f.expression;
  }
}

// ---------------------------------------------------------------- saved option lists (OPT-07)

/** An option as a saved list keeps it: no ID and no condition, which belong to one question. */
export type SavedOption = { text: string; score?: number; special?: string; isExclusive?: boolean; clinicalOutputs?: ClinicalOutput[] };

/** A question's options, ready to save as a reusable list. */
export function toSavedOptions(el: ElementJson): SavedOption[] {
  return optionsOf(el).map(({ text, score, special, isExclusive, clinicalOutputs }) =>
    JSON.parse(JSON.stringify({ text, score, special, isExclusive, clinicalOutputs })),
  );
}

/**
 * Copies a saved list into a question (OPT-07): each option gets a new stable ID and its
 * disclosures new IDs, so later changes to the list never change this question (OPT-02).
 * "add" keeps the current options; "replace" drops them.
 */
export function applySavedOptions(doc: ChapterJson, name: string, saved: SavedOption[], mode: "add" | "replace"): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const many = kindOf(loc.el) === "selectmany";
    const kept = mode === "add" ? optionsOf(loc.el) : [];
    const taken = new Set(kept.map((c) => c.value));
    const added = saved.map((s) => {
      const value = newId("o", 4, (id) => taken.has(id));
      taken.add(value);
      const option: ChoiceJson = { value, text: s.text };
      if (s.score !== undefined) option.score = s.score;
      if (s.special) option.special = s.special;
      if (many && (s.isExclusive || s.special)) option.isExclusive = true;
      if (s.clinicalOutputs?.length) option.clinicalOutputs = s.clinicalOutputs.map((o) => ({ ...o, id: newId("out", 6) }));
      return option;
    });
    loc.el.choices = orderOptions([...kept, ...added]);
  });
}
