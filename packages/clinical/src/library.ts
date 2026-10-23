// The question library and "Same as…" (plan C4, session S24), as edits to a Question Set's JSON.
// Go decides what may be inserted or adopted (apps/api/internal/keys: Compatible, Candidates; the
// library routes); these functions only make the change in the editor:
//   - insertFromLibrary puts a library question in, keeping its key and its options' values, so
//     every HQ it's in shares the key and pre-fill can copy an earlier answer to it;
//   - adoptKey gives a question another question's key and options ("Same as…"), keeping its text.
import {
  EXPRESSION_KEYS, childrenOf, edit, expressionsIn, findElement, isContainer, optionsOf, rewriteReferences, triggersOf, type ChapterJson,
  type ElementJson, type Target,
} from "./doc";
import { newId } from "./ids";
import type { ClinicalOutput } from "./types";

/** Where an adopted key came from (`linkedFrom`): for information only. */
export type LinkedFrom = { questionnaire: string; title: string };

/** An element's name and every name inside it (a group's or a repeating group's questions). */
export function namesInElement(el: ElementJson): string[] {
  return [el.name, ...(isContainer(el) ? childrenOf(structuredClone(el)).flatMap(namesInElement) : [])];
}

/** The element's names that a Question Set of the HQ already uses: inserting it would repeat a key. */
export function takenNames(sets: ChapterJson[], el: ElementJson): string[] {
  return namesInElement(el).filter((n) => sets.some((d) => findElement(structuredClone(d), n)));
}

/**
 * Inserts a copy of a library question at the target, keeping every name and option value. Its
 * disclosures get new IDs, as a copied question's do. The caller checks takenNames first.
 */
export function insertFromLibrary(doc: ChapterJson, target: Target, element: ElementJson): { doc: ChapterJson; name: string } {
  const copy = structuredClone(element);
  renewOutputIds(copy);
  const next = edit(doc, (d) => {
    const parent = target.parent ? findElement(d, target.parent)?.el : undefined;
    const list = parent && isContainer(parent) ? childrenOf(parent) : (d.pages!.find((p) => p.name === target.page)!.elements ??= []);
    list.splice(target.index ?? list.length, 0, copy);
  });
  return { doc: next, name: copy.name };
}

/** The properties that, with `choices`, make up a choice question's options. */
const OPTION_KEYS = ["showOtherItem", "otherText", "hasOther", "showNoneItem", "noneText", "hasNone"];

/**
 * "Same as…": the question takes the chosen question's key (its `name`) and, if it has options, the
 * chosen question's options (values and wording, translations included, and its Other and None
 * options), without their disclosures. It keeps its own text and everything else, except that it's
 * no longer Yes / No when the new options aren't two, and it's Always ask fresh when the chosen
 * question is (Rahul, 23 Oct; the author may switch it off). Conditions and skip rules in this
 * Question Set that tested it now test the new key, and an old option is mapped to the new one with
 * the same text where there is one (optionMapFor); the editor renames other Question Sets' with
 * renameReferences. Go has already said they're compatible.
 */
export function adoptKey(doc: ChapterJson, name: string, from: ElementJson, linkedFrom: LinkedFrom): ChapterJson {
  return edit(doc, (d) => {
    const el = findElement(d, name)?.el;
    if (!el) return;
    const options = new Map([[name, optionMapFor(el, from)]]);
    if (Array.isArray(from.choices)) {
      el.choices = structuredClone(from.choices).map((c) => {
        if (typeof c !== "object") return c;
        const { clinicalOutputs: _outputs, visibleIf: _condition, ...rest } = c;
        return rest;
      });
      for (const k of OPTION_KEYS) {
        if (from[k] === undefined) delete el[k];
        else el[k] = structuredClone(from[k]);
      }
      if (el.choices.length !== 2) delete el.yesNo;
    }
    if (from.alwaysFresh) el.alwaysFresh = true;
    el.name = from.name;
    el.linkedFrom = linkedFrom;
    renameIn(d, new Map([[name, from.name]]), options);
  });
}

/** Each of the question's option values mapped to the chosen question's option with the same text. */
export function optionMapFor(el: ElementJson, from: ElementJson): Map<string, string> {
  const map = new Map<string, string>();
  const theirs = optionsOf(from);
  for (const o of optionsOf(el)) {
    const same = theirs.find((t) => t.text.trim() === o.text.trim());
    if (same) map.set(String(o.value), String(same.value));
  }
  return map;
}

/**
 * What adopting `from`'s key would lose, for the window to say before Use this: the question's
 * options that have disclosures (they go with the options), and the conditions in this Question Set
 * and `others` (the HQ's other Question Sets) that test an old option no new one has the same text
 * as (the logic-problem checks then flag them).
 */
export function adoptionLosses(doc: ChapterJson, name: string, from: ElementJson, others: ChapterJson[] = []): { disclosures: number; conditions: number } {
  const el = findElement(structuredClone(doc), name)?.el;
  if (!el || !Array.isArray(from.choices)) return { disclosures: 0, conditions: 0 };
  const disclosures = optionsOf(el).filter((o) => o.clinicalOutputs?.length).length;
  const map = optionMapFor(el, from);
  const unmapped = new Set(optionsOf(el).map((o) => String(o.value)).filter((v) => !map.has(v)));
  const tests = (expr: string) =>
    [...expr.matchAll(new RegExp(`\\{(?:panel\\.)?${name}\\}[^'\\[{]*(\\[[^\\]]*\\]|'[^']*')`, "g"))].some((m) =>
      [...m[1].matchAll(/'([^']*)'/g)].some((v) => unmapped.has(v[1])),
    );
  // This set's conditions, calculations and skip rules, and every other set's (a later set's
  // Question Set condition may test it: verification 2, finding 1).
  let conditions = 0;
  for (const d of [doc, ...others]) {
    const exprs = expressionsIn(structuredClone(d)).map((x) => x.expr);
    if (typeof d.chapterVisibleIf === "string") exprs.push(d.chapterVisibleIf);
    conditions += exprs.filter(tests).length;
  }
  return { disclosures, conditions };
}

/** Renames questions (and their options) in every condition, calculation and skip rule of a chapter, and its Question Set condition. */
export function renameReferences(doc: ChapterJson, names: Map<string, string>, options = new Map<string, Map<string, string>>()): ChapterJson {
  return edit(doc, (d) => renameIn(d, names, options));
}

function renameIn(d: ChapterJson, names: Map<string, string>, options: Map<string, Map<string, string>>) {
  const fix = (el: ElementJson) => {
    for (const key of EXPRESSION_KEYS) if (typeof el[key] === "string") el[key] = rewriteReferences(el[key] as string, names, options);
    for (const c of el.choices ?? []) if (typeof c === "object" && c.visibleIf) c.visibleIf = rewriteReferences(c.visibleIf, names, options);
    if (isContainer(el)) childrenOf(el).forEach(fix);
  };
  if (typeof d.chapterVisibleIf === "string") d.chapterVisibleIf = rewriteReferences(d.chapterVisibleIf, names, options);
  for (const page of d.pages ?? []) {
    if (page.visibleIf) page.visibleIf = rewriteReferences(page.visibleIf, names, options);
    (page.elements ?? []).forEach(fix);
  }
  for (const t of triggersOf(d)) {
    t.expression = rewriteReferences(t.expression, names, options);
    if (t.gotoName && names.has(t.gotoName)) t.gotoName = names.get(t.gotoName);
  }
  // Test cases keep their answers under the new key, with the values mapped (PRV-05).
  for (const tc of Array.isArray(d.testCases) ? (d.testCases as { answers?: Record<string, unknown> }[]) : []) {
    if (!tc.answers) continue;
    for (const [old, renamed] of names) {
      if (!(old in tc.answers)) continue;
      const map = options.get(old);
      const value = tc.answers[old];
      const mapOne = (v: unknown) => (typeof v === "string" && map?.has(v) ? map.get(v) : v);
      delete tc.answers[old];
      tc.answers[renamed] = Array.isArray(value) ? value.map(mapOne) : mapOne(value);
    }
  }
}

/** Gives every disclosure in an element (on it, its options and inside it) a new ID. */
function renewOutputIds(el: ElementJson) {
  const renew = (list: ClinicalOutput[] | undefined) => list?.map((o) => ({ ...o, id: newId("out", 6) }));
  if (el.clinicalOutputs) el.clinicalOutputs = renew(el.clinicalOutputs);
  for (const c of el.choices ?? []) if (typeof c === "object" && c.clinicalOutputs) c.clinicalOutputs = renew(c.clinicalOutputs);
  if (isContainer(el)) childrenOf(el).forEach(renewOutputIds);
}
