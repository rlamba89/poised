// The editor's model: a chapter's SurveyJS JSON edited as plain data (plan-redesign.md).
// Every edit clones the chapter, changes the clone and returns it, so the editor can keep
// earlier versions for undo. Lifebox names: a chapter is a "Question Set", outputs are
// "Disclosures".
import { ConditionsParser } from "survey-core";
import { newId } from "./ids";
import type { ClinicalOutput } from "./types";

/** A SurveyJS string: plain, or per language (`{ default: "…", de: "…" }`) once translated (LNG-01). */
export type LocString = string | Record<string, string>;

export type ChoiceJson = {
  value: string;
  text: string;
  isExclusive?: boolean;
  /** "dontknow" or "refuse": a one-click option (OPT-04). */
  special?: string;
  /** Points towards a score total (OPT-03, CAL-02). */
  score?: number;
  /** Shown only when this condition is true (OPT-05). */
  visibleIf?: string;
  clinicalOutputs?: ClinicalOutput[];
};

export type ElementJson = {
  type: string;
  name: string;
  title?: string;
  description?: string;
  isRequired?: boolean;
  clinicianOnly?: boolean;
  visibleIf?: string;
  choices?: (ChoiceJson | string)[];
  elements?: ElementJson[];
  /** A repeating group's (paneldynamic's) elements. */
  templateElements?: ElementJson[];
  html?: string;
  inputType?: string;
  clinicalOutputs?: ClinicalOutput[];
  [key: string]: unknown;
};

export type PageJson = {
  name: string;
  title?: string;
  visibleIf?: string;
  clinicalSummary?: boolean;
  elements?: ElementJson[];
  [key: string]: unknown;
};

export type ChapterJson = { pages?: PageJson[]; [key: string]: unknown };

// ---------------------------------------------------------------- element kinds

/** The Add content buttons, in Lifebox order. Number (DSG-01) and the SurveyJS types come after Date. */
export const KINDS = [
  "group", "section", "yesno", "selectone", "selectmany", "text", "date", "number",
  "rating", "grid", "calculation", "upload", "signature",
  "medication", "admissions", "bmi", "profile", "statement",
] as const;
export type Kind = (typeof KINDS)[number];

export const KIND_LABEL: Record<Kind, string> = {
  group: "Group",
  section: "Section",
  yesno: "Yes / No",
  selectone: "Select One",
  selectmany: "Select Many",
  text: "Text",
  date: "Date",
  number: "Number",
  rating: "Rating",
  grid: "Grid",
  calculation: "Calculation",
  upload: "File upload",
  signature: "Signature",
  medication: "Medication",
  admissions: "Admissions",
  bmi: "BMI",
  profile: "Profile",
  statement: "Statement",
};

/** What an element is, in Lifebox terms. */
export function kindOf(el: ElementJson): Kind {
  switch (el.type) {
    case "panel":
      return el.patientPage ? "section" : "group";
    case "paneldynamic":
      return "group";
    case "radiogroup":
      return el.yesNo ? "yesno" : "selectone";
    case "dropdown":
      return "selectone";
    case "checkbox":
    case "tagbox":
      return "selectmany";
    case "comment":
      return "text";
    case "text":
      if (el.dateFormat || el.inputType === "date" || el.inputType === "month") return "date";
      return el.inputType === "number" || el.maskType === "numeric" ? "number" : "text";
    case "matrixdynamic":
      return el.dateList ? "date" : "text";
    case "rating":
      return "rating";
    case "matrix":
      return "grid";
    case "expression":
      return "calculation";
    case "file":
      return "upload";
    case "signaturepad":
      return "signature";
    case "html":
      return "statement";
    case "medication":
    case "admissions":
    case "bmi":
    case "profile":
      return el.type;
    default:
      return "text";
  }
}

/** Groups, Sections and repeating groups hold other elements. */
export const isContainer = (el: ElementJson) => el.type === "panel" || el.type === "paneldynamic";
export const isRepeating = (el: ElementJson) => el.type === "paneldynamic";

/** The elements inside a container (a repeating group keeps them in `templateElements`). */
export function childrenOf(el: ElementJson): ElementJson[] {
  if (el.type === "paneldynamic") return (el.templateElements ??= []);
  return (el.elements ??= []);
}

/** Yes / No, Select One and Select Many: outputs and conditions are per option. */
export const isChoiceKind = (k: Kind) => k === "yesno" || k === "selectone" || k === "selectmany";
/** Text, Date, Number and Rating carry outputs on the question itself (a Calculation too, unless it has bands). */
export const hasQuestionOutputs = (k: Kind) => k === "text" || k === "date" || k === "number" || k === "rating" || k === "calculation";
/** Kinds that hold an answer (everything except groups, statements and the profile). */
export const isAnswerKind = (k: Kind) => !["group", "section", "statement", "profile"].includes(k);
/** BMI and Profile are fixed blocks: they can only be deleted (Lifebox "Locked"). */
export const isLocked = (k: Kind) => k === "bmi" || k === "profile";

/** A new element of a kind, with fresh IDs. New questions are required (QST-02). */
export function newElement(kind: Kind, taken: Set<string>): ElementJson {
  const name = freshId(kind === "group" || kind === "section" ? "g" : "q", 6, taken);
  const options = (labels: string[]) => {
    const used = new Set<string>();
    return labels.map((text) => ({ value: freshId("o", 4, used), text }));
  };
  switch (kind) {
    case "group":
      return { type: "panel", name, title: "Group", elements: [] };
    case "section":
      return { type: "panel", name, title: "Section", patientPage: true, showAsHeading: true, elements: [] };
    case "yesno":
      return { type: "radiogroup", name, yesNo: true, colCount: 0, title: "Yes / No question", isRequired: true, choices: options(["Yes", "No"]) };
    case "selectone":
      return { type: "radiogroup", name, title: "Select One question", isRequired: true, choices: options(["Option 1", "Option 2", "Option 3"]) };
    case "selectmany":
      return { type: "checkbox", name, title: "Select Many question", isRequired: true, choices: options(["Statement 1", "Statement 2", "Statement 3"]) };
    case "text":
      return { type: "text", name, title: "Text question", isRequired: true };
    case "date":
      return { type: "text", inputType: "date", name, title: "Date question", isRequired: true };
    case "number":
      return { type: "text", inputType: "number", name, title: "Number question", isRequired: true };
    case "rating":
      return { type: "rating", name, title: "Rating question", isRequired: true, rateMin: 0, rateMax: 10, minRateDescription: "None", maxRateDescription: "Worst" };
    case "grid": {
      const used = new Set<string>();
      const rows = ["Row 1", "Row 2"].map((text) => ({ value: freshId("r", 4, used), text }));
      return { type: "matrix", name, title: "Grid question", isRequired: true, rows, columns: options(["Column 1", "Column 2", "Column 3"]) };
    }
    case "calculation":
      return { type: "expression", name, title: "Calculation", expression: "", clinicianOnly: true };
    case "upload":
      return { type: "file", name, title: "Upload a file", storeDataAsText: true, acceptedTypes: "image/*,application/pdf", maxSize: 5242880 };
    case "signature":
      return { type: "signaturepad", name, title: "Please sign below", isRequired: true };
    case "medication":
      return { type: "medication", name, medicationType: "prescribed", title: "Please add the prescribed medications you are taking", isRequired: true };
    case "admissions":
      return { type: "admissions", name, title: "Please list any previous hospital admissions", isRequired: true };
    case "bmi":
      return { type: "bmi", name, title: "BMI Calculator", isRequired: true };
    case "profile":
      return { type: "profile", name, title: "Your Profile" };
    case "statement":
      return { type: "html", name, html: "<p>Panel statement</p>" };
  }
}

function freshId(prefix: string, length: number, taken: Set<string>): string {
  const id = newId(prefix, length, (c) => taken.has(c));
  taken.add(id);
  return id;
}

// ---------------------------------------------------------------- reading

/** A SurveyJS string that may be localised ({ default: "..." }). */
export function textOf(value: unknown): string {
  if (typeof value === "string") return value;
  if (value && typeof value === "object") return String((value as Record<string, unknown>).default ?? "");
  return "";
}

/** Options as objects (the Creator could save plain strings). */
export function optionsOf(el: ElementJson): ChoiceJson[] {
  return (el.choices ?? []).map((c) => (typeof c === "string" ? { value: c, text: c } : { ...c, text: textOf(c.text) || String(c.value) }));
}

export const pagesOf = (doc: ChapterJson): PageJson[] => doc.pages ?? [];

/** Every element in document order, with the page it is on. */
export function allElements(doc: ChapterJson): { el: ElementJson; page: PageJson }[] {
  const out: { el: ElementJson; page: PageJson }[] = [];
  const walk = (els: ElementJson[] | undefined, page: PageJson) => {
    for (const el of els ?? []) {
      out.push({ el, page });
      if (isContainer(el)) walk(childrenOf(el), page);
    }
  };
  for (const page of pagesOf(doc)) walk(page.elements, page);
  return out;
}

/** All names in use (pages, elements), so new IDs never clash. */
export function namesIn(doc: ChapterJson): Set<string> {
  const names = new Set<string>(pagesOf(doc).map((p) => p.name));
  for (const { el } of allElements(doc)) names.add(el.name);
  return names;
}

export type Location = { el: ElementJson; page: PageJson; siblings: ElementJson[]; index: number; parent?: ElementJson };

export function findElement(doc: ChapterJson, name: string): Location | undefined {
  const walk = (siblings: ElementJson[], page: PageJson, parent?: ElementJson): Location | undefined => {
    for (let index = 0; index < siblings.length; index++) {
      const el = siblings[index];
      if (el.name === name) return { el, page, siblings, index, parent };
      if (isContainer(el)) {
        const found = walk(childrenOf(el), page, el);
        if (found) return found;
      }
    }
  };
  for (const page of pagesOf(doc)) {
    const found = walk(page.elements ?? (page.elements = []), page);
    if (found) return found;
  }
}

export const findPage = (doc: ChapterJson, name: string) => pagesOf(doc).find((p) => p.name === name);

/** The page title, or a fallback for pages saved without one. */
export const pageTitle = (page: PageJson, index: number) => textOf(page.title) || `Page ${index + 1}`;

/** How many disclosures (outputs) an element has: on its options, grid cells or bands, or its own. */
export function countDisclosures(el: ElementJson): number {
  const kind = kindOf(el);
  if (isChoiceKind(kind)) return optionsOf(el).reduce((n, c) => n + (c.clinicalOutputs?.length ?? 0), 0);
  if (kind === "grid") {
    const cells = (el.cellOutputs ?? {}) as Record<string, Record<string, ClinicalOutput[]>>;
    return Object.values(cells).reduce((n, row) => n + Object.values(row).reduce((m, list) => m + list.length, 0), 0);
  }
  if (kind === "calculation" && Array.isArray(el.bands) && el.bands.length) {
    return (el.bands as { clinicalOutputs?: ClinicalOutput[] }[]).reduce((n, b) => n + (b.clinicalOutputs?.length ?? 0), 0);
  }
  return el.clinicalOutputs?.length ?? 0;
}

// ---------------------------------------------------------------- editing

/** Applies an edit to a copy of the chapter. */
export function edit(doc: ChapterJson, change: (draft: ChapterJson) => void): ChapterJson {
  const draft = structuredClone(doc);
  draft.pages ??= [];
  change(draft);
  return draft;
}

export function addPage(doc: ChapterJson, title?: string): { doc: ChapterJson; name: string } {
  const name = freshId("p", 6, namesIn(doc));
  const next = edit(doc, (d) => {
    d.pages!.push({ name, title: title ?? `Page ${d.pages!.length + 1}`, elements: [] });
  });
  return { doc: next, name };
}

export function updatePage(doc: ChapterJson, name: string, patch: Partial<PageJson>): ChapterJson {
  return edit(doc, (d) => {
    const page = findPage(d, name);
    if (page) assignDefined(page, keepTranslations(page, patch));
  });
}

export function movePage(doc: ChapterJson, name: string, delta: number): ChapterJson {
  return edit(doc, (d) => moveIn(d.pages!, d.pages!.findIndex((p) => p.name === name), delta));
}

export function deletePage(doc: ChapterJson, name: string): ChapterJson {
  return edit(doc, (d) => {
    d.pages = d.pages!.filter((p) => p.name !== name);
  });
}

/** Where a new element goes: a page, or a group or section on it, at an index (end if omitted). */
export type Target = { page: string; parent?: string; index?: number };

export function addElement(doc: ChapterJson, target: Target, kind: Kind): { doc: ChapterJson; name: string } {
  const el = newElement(kind, namesIn(doc));
  const next = edit(doc, (d) => {
    const list = targetList(d, target);
    list.splice(target.index ?? list.length, 0, el);
  });
  return { doc: next, name: el.name };
}

export function updateElement(doc: ChapterJson, name: string, patch: Partial<ElementJson>): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (loc) assignDefined(loc.el, keepTranslations(loc.el, patch));
  });
}

export function deleteElement(doc: ChapterJson, name: string): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (loc) loc.siblings.splice(loc.index, 1);
  });
}

export function moveElement(doc: ChapterJson, name: string, delta: number): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (loc) moveIn(loc.siblings, loc.index, delta);
  });
}

/** Moves an element to a new place (drag and drop), possibly into another group or page. */
export function moveElementTo(doc: ChapterJson, name: string, target: Target): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc || target.parent === name) return;
    if (target.parent && findElement({ pages: [{ name: "", elements: [loc.el] }] }, target.parent)) return; // not into itself
    loc.siblings.splice(loc.index, 1);
    const list = targetList(d, target);
    const sameList = list === loc.siblings;
    let index = target.index ?? list.length;
    if (sameList && target.index !== undefined && loc.index < target.index) index -= 1;
    list.splice(Math.min(index, list.length), 0, loc.el);
  });
}

/** Moves an element out of its group, to just after the group. */
export function moveOutOfGroup(doc: ChapterJson, name: string): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc?.parent) return;
    const parent = findElement(d, loc.parent.name)!;
    loc.siblings.splice(loc.index, 1);
    parent.siblings.splice(parent.index + 1, 0, loc.el);
  });
}

/**
 * The groups and Sections on an element's page it can be moved into (STR-02): not itself or
 * anything inside it, not the group it is already in, and a Section never inside another group.
 */
export function groupsToMoveInto(doc: ChapterJson, name: string): ElementJson[] {
  const loc = findElement(doc, name);
  if (!loc) return [];
  const inside = new Set<string>();
  const collect = (el: ElementJson) => {
    inside.add(el.name);
    if (isContainer(el)) childrenOf(el).forEach(collect);
  };
  collect(loc.el);
  if (kindOf(loc.el) === "section") return [];
  return allElements(doc).filter(({ el, page }) => page === loc.page && isContainer(el) && !inside.has(el.name) && el.name !== loc.parent?.name).map(({ el }) => el);
}

/**
 * Copies an element below itself (STR-07): new IDs for it, everything inside it and its
 * options. Clinical outputs are kept. Conditions inside the copy that pointed at elements
 * inside the original now point at their copies.
 */
export function copyElement(doc: ChapterJson, name: string): { doc: ChapterJson; name: string } {
  let copyName = "";
  const next = edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const [copy] = cloneWithNewIds([loc.el], namesIn(d));
    if (copy.title) copy.title = `${textOf(copy.title)} (copy)`;
    loc.siblings.splice(loc.index + 1, 0, copy);
    copyName = copy.name;
  });
  return { doc: next, name: copyName };
}

/** Copies a page below itself, like copyElement (STR-01). Its own condition is kept. */
export function copyPage(doc: ChapterJson, name: string): { doc: ChapterJson; name: string } {
  let copyName = "";
  const next = edit(doc, (d) => {
    const index = d.pages!.findIndex((p) => p.name === name);
    if (index < 0) return;
    const taken = namesIn(d);
    const page = d.pages![index];
    copyName = freshId("p", 6, taken);
    const copy: PageJson = { ...structuredClone(page), name: copyName, title: `${pageTitle(page, index)} (copy)` };
    copy.elements = cloneWithNewIds(page.elements ?? [], taken);
    d.pages!.splice(index + 1, 0, copy);
  });
  return { doc: next, name: copyName };
}

/** Moves a page to just before another page (drag and drop in the Structure tree). */
export function movePageTo(doc: ChapterJson, name: string, before: string): ChapterJson {
  return edit(doc, (d) => {
    const from = d.pages!.findIndex((p) => p.name === name);
    if (from < 0 || name === before) return;
    const [page] = d.pages!.splice(from, 1);
    const to = d.pages!.findIndex((p) => p.name === before);
    d.pages!.splice(to < 0 ? d.pages!.length : to, 0, page);
  });
}

/** Deep copies of elements with fresh IDs throughout, and conditions among them remapped. */
function cloneWithNewIds(elements: ElementJson[], taken: Set<string>): ElementJson[] {
  const copies = structuredClone(elements);
  const renamed = new Map<string, string>();
  const optionMap = new Map<string, Map<string, string>>();
  const rename = (el: ElementJson) => {
    const id = freshId(el.name.startsWith("g_") || isContainer(el) ? "g" : "q", 6, taken);
    renamed.set(el.name, id);
    if (Array.isArray(el.choices)) {
      const used = new Set<string>();
      const before = el.choices.map((c) => (typeof c === "string" ? { value: c, text: c } : c));
      const after = before.map((c) => ({ ...c, value: freshId("o", 4, used), clinicalOutputs: c.clinicalOutputs?.map(newOutputId) }));
      optionMap.set(el.name, new Map(before.map((c, i) => [String(c.value), after[i].value])));
      el.choices = after;
    }
    el.name = id;
    if (el.clinicalOutputs) el.clinicalOutputs = el.clinicalOutputs.map(newOutputId);
    if (isContainer(el)) childrenOf(el).forEach(rename);
  };
  copies.forEach(rename);
  const fix = (el: ElementJson) => {
    for (const key of EXPRESSION_KEYS) {
      if (typeof el[key] === "string") el[key] = rewriteReferences(el[key] as string, renamed, optionMap);
    }
    for (const c of el.choices ?? []) {
      if (typeof c === "object" && c.visibleIf) c.visibleIf = rewriteReferences(c.visibleIf, renamed, optionMap);
    }
    if (isContainer(el)) childrenOf(el).forEach(fix);
  };
  copies.forEach(fix);
  return copies;
}

const newOutputId = (o: ClinicalOutput): ClinicalOutput => ({ ...o, id: newId("out", 6) });

/** Element properties that hold expressions referring to other questions. */
export const EXPRESSION_KEYS = ["visibleIf", "requiredIf", "enableIf", "expression"] as const;

/**
 * Rewrites references after a copy or a rename: `{old}`, `{panel.old}`, `{old.row}`,
 * `'old'` in score() / band(), and the option IDs compared with a renamed question.
 */
export function rewriteReferences(expr: string, names: Map<string, string>, options = new Map<string, Map<string, string>>()): string {
  return expr
    .replace(/\{((?:panel\.)?)([A-Za-z0-9_]+)((?:\.[A-Za-z0-9_]+)?)\}(\s*(?:=|==|<>|!=|contains|notcontains|anyof|allof|noneof)\s*)(\[[^\]]*\]|'[^']*')/g,
      (all, prefix: string, q: string, sub: string, op: string, value: string) => {
        const map = options.get(q);
        const newValue = map ? value.replace(/'([^']*)'/g, (v, o: string) => `'${map.get(o) ?? o}'`) : value;
        return `{${prefix}${names.get(q) ?? q}${sub}}${op}${newValue}`;
      })
    .replace(/\{((?:panel\.)?)([A-Za-z0-9_]+)((?:\.[A-Za-z0-9_]+)?)\}/g, (all, prefix: string, q: string, sub: string) =>
      names.has(q) ? `{${prefix}${names.get(q)}${sub}}` : all)
    .replace(/\b(score|band)\(([^)]*)\)/g, (all, fn: string, args: string) =>
      `${fn}(${args.replace(/'([^']*)'/g, (a, n: string) => `'${names.get(n) ?? n}'`)})`);
}

/** The question names an expression refers to. */
export function referencesIn(expr: string | undefined): string[] {
  if (!expr) return [];
  const out = new Set<string>();
  for (const m of expr.matchAll(/\{(?:panel\.)?([A-Za-z0-9_]+)/g)) out.add(m[1]);
  for (const m of expr.matchAll(/\b(?:score|band)\(([^)]*)\)/g)) for (const n of m[1].matchAll(/'([^']*)'/g)) out.add(n[1]);
  return [...out];
}

// ---------------------------------------------------------------- options

export function setChoices(doc: ChapterJson, name: string, choices: ChoiceJson[]): ChapterJson {
  return updateElement(doc, name, { choices });
}

export function addChoice(doc: ChapterJson, name: string, text: string): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const choices = optionsOf(loc.el);
    const value = freshId("o", 4, new Set(choices.map((c) => c.value)));
    // New options go before the exclusive "None" option, which stays last.
    const noneAt = choices.findIndex((c) => c.isExclusive);
    choices.splice(noneAt < 0 ? choices.length : noneAt, 0, { value, text });
    loc.el.choices = choices;
  });
}

/** Turns the exclusive "None" option (Lifebox "No to all") on with a label, or off (label undefined). */
export function setNoneOption(doc: ChapterJson, name: string, label: string | undefined): ChapterJson {
  return edit(doc, (d) => {
    const loc = findElement(d, name);
    if (!loc) return;
    const choices = optionsOf(loc.el);
    const none = choices.find((c) => c.isExclusive);
    if (label === undefined) {
      loc.el.choices = choices.filter((c) => !c.isExclusive);
    } else if (none) {
      none.text = label;
      loc.el.choices = choices;
    } else {
      const value = freshId("o", 4, new Set(choices.map((c) => c.value)));
      loc.el.choices = [...choices, { value, text: label, isExclusive: true }];
    }
  });
}

// ---------------------------------------------------------------- conditions

/** One simple condition, as Lifebox writes them (the importer and older chapters use this form). */
export type Condition = { question: string; isNot: boolean; option: string };

const SIMPLE = /^\s*\{([A-Za-z0-9_]+)\}\s*(=|<>|!=|contains|notcontains)\s*'([^']*)'\s*$/;

/** Reads a simple condition; undefined when the expression is anything else (use the Code tab). */
export function parseCondition(expr: string | undefined): Condition | undefined {
  const m = expr ? SIMPLE.exec(expr) : null;
  if (!m) return undefined;
  return { question: m[1], isNot: m[2] === "<>" || m[2] === "!=" || m[2] === "notcontains", option: m[3] };
}

/** The expression for a condition. Select Many tests contain / don't contain; the others equal / not equal. */
export function formatCondition(c: Condition, on: ElementJson): string {
  const many = kindOf(on) === "selectmany";
  const op = many ? (c.isNot ? "notcontains" : "contains") : c.isNot ? "<>" : "=";
  return `{${c.question}} ${op} '${c.option}'`;
}

/** An expression with question titles and option labels in place of IDs, for display only. */
export function readableExpression(doc: ChapterJson, expr: string): string {
  const titles = new Map<string, string>();
  const labels = new Map<string, string>();
  for (const { el } of allElements(doc)) {
    titles.set(el.name, elementLabel(el));
    for (const o of optionsOf(el)) labels.set(o.value, o.text);
  }
  return expr
    .replace(/\{(?:panel\.)?([^}.]+)[^}]*\}/g, (all, name: string) => (titles.has(name) ? `“${titles.get(name)}”` : all))
    .replace(/'([^']*)'/g, (all, value: string) => (labels.has(value) ? `“${labels.get(value)}”` : titles.has(value) ? `“${titles.get(value)}”` : all));
}

/** Every expression in a chapter, with what it belongs to (pages, elements, options, skip rules). */
export function expressionsIn(doc: ChapterJson): { expr: string; owner: string; label: string; page: PageJson; el?: ElementJson; key: string }[] {
  const out: { expr: string; owner: string; label: string; page: PageJson; el?: ElementJson; key: string }[] = [];
  pagesOf(doc).forEach((p, i) => {
    if (p.visibleIf) out.push({ expr: p.visibleIf, owner: p.name, label: `Page "${pageTitle(p, i)}"`, page: p, key: "visibleIf" });
  });
  for (const { el, page } of allElements(doc)) {
    const label = `"${elementLabel(el)}"`;
    for (const key of EXPRESSION_KEYS) {
      const expr = el[key];
      if (typeof expr === "string" && expr.trim()) out.push({ expr, owner: el.name, label, page, el, key });
    }
    for (const o of optionsOf(el)) {
      if (o.visibleIf) out.push({ expr: o.visibleIf, owner: el.name, label: `${label} option "${o.text}"`, page, el, key: "choice" });
    }
  }
  for (const t of triggersOf(doc)) {
    const page = pageOfTrigger(doc, t) ?? pagesOf(doc)[0];
    if (page && t.expression) out.push({ expr: t.expression, owner: "", label: "A skip rule", page, key: "trigger" });
  }
  return out;
}

/** A short name for an element: its title, a statement's first words, or its ID. */
export function elementLabel(el: ElementJson): string {
  return textOf(el.title) || plainText(el.html ?? "").slice(0, 60) || el.name;
}

/** Everything whose conditions or calculations refer to an element (STR-08, LOG-13). */
export function dependentsOf(doc: ChapterJson, name: string): string[] {
  const out = expressionsIn(doc)
    .filter((x) => x.owner !== name && referencesIn(x.expr).includes(name))
    .map((x) => (x.key === "expression" ? `${x.label} (calculation)` : x.label));
  return [...new Set(out)];
}

// ---------------------------------------------------------------- skip rules (LOG-12)

/**
 * A SurveyJS trigger: "skip" goes to a question, "complete" ends the Question Set. `page` is the
 * page whose Logic panel the rule was added on, so it stays listed there.
 */
export type TriggerJson = { type: "skip" | "complete"; expression: string; gotoName?: string; page?: string };

export const triggersOf = (doc: ChapterJson): TriggerJson[] =>
  ((doc.triggers as TriggerJson[] | undefined) ?? []).filter((t) => t.type === "skip" || t.type === "complete");

/**
 * The page a skip rule belongs to: the page it was added on. Rules saved before that was
 * recorded belong to the latest page whose questions they test.
 */
export function pageOfTrigger(doc: ChapterJson, t: TriggerJson): PageJson | undefined {
  const own = t.page ? findPage(doc, t.page) : undefined;
  if (own) return own;
  const refs = new Set(referencesIn(t.expression));
  let found: PageJson | undefined;
  for (const { el, page } of allElements(doc)) if (refs.has(el.name)) found = page;
  return found;
}

/** The page a skip rule goes to (the page holding its target question). */
export function triggerTarget(doc: ChapterJson, t: TriggerJson): PageJson | undefined {
  return t.gotoName ? findElement(doc, t.gotoName)?.page : undefined;
}

/** Replaces the chapter's skip rules. */
export function setTriggers(doc: ChapterJson, triggers: TriggerJson[]): ChapterJson {
  return edit(doc, (d) => {
    if (triggers.length) d.triggers = triggers;
    else delete d.triggers;
  });
}

/** The first answerable question on a page, which a "skip to page" rule goes to. */
export function firstQuestionOn(page: PageJson): ElementJson | undefined {
  const walk = (els: ElementJson[]): ElementJson | undefined => {
    for (const el of els) {
      if (isContainer(el) && !isRepeating(el)) {
        const inner = walk(childrenOf(el));
        if (inner) return inner;
      } else if (el.type !== "html") return el;
    }
  };
  return walk(page.elements ?? []);
}

/** True if SurveyJS can parse the expression (the Logic tab's Code view). */
export function isValidExpression(expr: string): boolean {
  return !!new ConditionsParser().parseExpression(expr);
}

/** Values a condition can test besides questions: the sample patient and who is viewing. */
export const VARIABLES = ["viewer", "patientName", "patientAge", "patientSex"];

/** Names an expression refers to that are not questions in this chapter (likely typos). */
export function unknownReferences(doc: ChapterJson, expr: string, variables: string[] = VARIABLES): string[] {
  const names = namesIn(doc);
  return referencesIn(expr).filter((r) => !names.has(r) && !variables.includes(r));
}

// ---------------------------------------------------------------- statements

/** Statement text ↔ HTML: one paragraph per line, escaped. */
export function textToHtml(text: string): string {
  return text
    .split(/\n/)
    .map((line) => `<p>${line.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;") || "&nbsp;"}</p>`)
    .join("");
}

export function plainText(html: string): string {
  return html
    .replace(/<\/p>\s*<p>/g, "\n")
    .replace(/<br\s*\/?>/g, "\n")
    .replace(/<[^>]+>/g, "")
    .replace(/&nbsp;/g, "")
    .replace(/&lt;/g, "<")
    .replace(/&gt;/g, ">")
    .replace(/&quot;/g, '"')
    .replace(/&#0?39;/g, "'")
    .replace(/&amp;/g, "&")
    .trim();
}

// ---------------------------------------------------------------- helpers

function targetList(d: ChapterJson, target: Target): ElementJson[] {
  if (target.parent) {
    const parent = findElement(d, target.parent)?.el;
    if (parent && isContainer(parent)) return childrenOf(parent);
  }
  const page = findPage(d, target.page);
  if (!page) throw new Error(`No page ${target.page}`);
  return (page.elements ??= []);
}

function moveIn<T>(list: T[], index: number, delta: number) {
  const to = index + delta;
  if (index < 0 || to < 0 || to >= list.length) return;
  const [item] = list.splice(index, 1);
  list.splice(to, 0, item);
}

/** Keys holding text a patient sees, which can be translated (LNG-01). */
export const LOCALIZABLE_KEYS = [
  "title", "description", "html", "placeholder", "requiredErrorText", "otherText", "selectAllText",
  "minRateDescription", "maxRateDescription", "panelAddText", "panelRemoveText", "addRowText",
] as const;

/** Item lists whose `text` can be translated. */
export const LOCALIZABLE_LISTS = ["choices", "rows", "columns"] as const;

/**
 * Editing the English text of something already translated keeps its translations:
 * a plain string in the patch becomes the `default` of the existing object.
 */
function keepTranslations(target: Record<string, unknown>, patch: Record<string, unknown>): Record<string, unknown> {
  const out = { ...patch };
  for (const key of LOCALIZABLE_KEYS) {
    if (typeof out[key] === "string") out[key] = withEnglish(target[key], out[key] as string);
  }
  for (const key of LOCALIZABLE_LISTS) {
    const before = target[key];
    if (!Array.isArray(out[key]) || !Array.isArray(before)) continue;
    const old = new Map(before.filter((c) => c && typeof c === "object").map((c) => [String(c.value), c.text]));
    out[key] = (out[key] as unknown[]).map((c) =>
      c && typeof c === "object" && typeof (c as ChoiceJson).text === "string"
        ? { ...c, text: withEnglish(old.get(String((c as ChoiceJson).value)), (c as ChoiceJson).text) }
        : c,
    );
  }
  return out;
}

/** The new English text, keeping any translations the old value had. */
export function withEnglish(old: unknown, text: string): LocString {
  if (old && typeof old === "object" && !Array.isArray(old)) {
    const others = Object.entries(old as Record<string, string>).filter(([k]) => k !== "default");
    if (others.length) return { ...Object.fromEntries(others), default: text };
  }
  return text;
}

/** Object.assign, except an undefined value removes the key (so the JSON stays clean). */
function assignDefined(target: Record<string, unknown>, patch: Record<string, unknown>) {
  for (const [k, v] of Object.entries(patch)) {
    if (v === undefined) delete target[k];
    else target[k] = v;
  }
}
