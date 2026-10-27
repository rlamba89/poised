// Warnings for authors (S44a): things worth knowing before publishing that don't block it, shown in
// the Publish window (and, for auto-triage, on the Rules page): auto-triage that can't work, and a
// code the ASA guide grades left without a grade. The author may go ahead anyway.
import type { AsaChoice } from "./asa";
import { allElements, elementLabel, optionsOf, textOf, type ChapterJson, type ElementJson } from "./doc";
import { bandsOf } from "./logic";
import { normalizeOutput } from "./properties";
import type { ClinicalOutput, CodeRef } from "./types";
import { clinicianOnlyIn } from "./questionSets";
import type { RuleSet } from "./rules";
import { ALL_ANSWERED, bandWhenInputs, isGuardedScore } from "./scores";

type Sets = { name: string; doc: ChapterJson; audience?: string }[];

/** The inputs a score names in allAnswered(…): its questions, `q.bmi` as `q`, the patient's age and sex. */
function inputsOf(expression: string): string[] {
  const m = new RegExp(`${ALL_ANSWERED}\\(([^)]*)\\)`).exec(expression);
  return m ? [...m[1].matchAll(/'([^']*)'/g)].map((x) => x[1].split(".")[0]) : [];
}

/**
 * Why auto-triage can't move a patient on for this HQ (S44a, Rahul 27 Oct): a score that needs
 * answers a clinician enters (a clinician-only question, or one in a clinician Question Set, or a
 * score that needs them) is Incomplete when the patient sends, and an Incomplete score stops
 * auto-triage (S42's guard, rightly: the clinician enters them and carries on). One line per such
 * score, naming the inputs; none when no rule auto-triages.
 */
export function autoTriageWarnings(sets: Sets, rules: RuleSet | null | undefined): string[] {
  if (!rules?.rules.some((r) => r.then.some((a) => a.kind === "autoTriage"))) return [];
  const clinicianEntered = new Map<string, ElementJson>();
  const scores: ElementJson[] = [];
  for (const s of sets) {
    for (const [name, el] of clinicianOnlyIn(s.doc, s.audience ?? "patient")) if (el.type !== "expression") clinicianEntered.set(name, el);
    for (const { el } of allElements(s.doc)) if (el.type === "expression" && isGuardedScore(el.expression)) scores.push(el);
  }
  // A score that needs a score which is always incomplete is too: work it out until nothing changes.
  const needs = new Map<string, ElementJson[]>();
  for (let changed = true; changed; ) {
    changed = false;
    for (const sc of scores) {
      if (needs.has(sc.name)) continue;
      const names = [...inputsOf(String(sc.expression)), ...bandWhenInputs(sc.name, sc.bands).map((n) => n.split(".")[0])]; // "also when" too (S44a)
      const inputs = names.flatMap((n) => (clinicianEntered.has(n) ? [clinicianEntered.get(n)!] : needs.has(n) ? needs.get(n)! : []));
      if (inputs.length) {
        needs.set(sc.name, [...new Set(inputs)]);
        changed = true;
      }
    }
  }
  return scores.filter((sc) => needs.has(sc.name)).map((sc) => {
    const inputs = needs.get(sc.name)!.map((el) => `“${elementLabel(el)}”`).join(", ");
    return `Auto-triage won't move patients on for this HQ: the score “${elementLabel(sc)}” needs answers a clinician enters (${inputs}), so it's always incomplete when the patient sends. These patients go to Ready for review.`;
  });
}

/** A code's examples in the ASA guide (GET /api/codes' asaChoices); undefined while not known. */
export type AsaGuide = (code: CodeRef) => AsaChoice[] | undefined;

/** Every disclosure on an element, with where it is: the question, an option, a band or a grid cell. */
function disclosuresOf(el: ElementJson): { where: string; output: ClinicalOutput }[] {
  const q = elementLabel(el);
  const out: { where: string; output: ClinicalOutput }[] = [];
  for (const o of el.clinicalOutputs ?? []) out.push({ where: `“${q}”`, output: o });
  for (const c of optionsOf(el)) for (const o of c.clinicalOutputs ?? []) out.push({ where: `“${q}” answered “${textOf(c.text) || c.value}”`, output: o });
  for (const b of bandsOf(el)) for (const o of b.clinicalOutputs ?? []) out.push({ where: `“${q}” band “${b.label ?? ""}”`, output: o });
  const cells = (el.cellOutputs ?? {}) as Record<string, Record<string, ClinicalOutput[]>>;
  for (const row of Object.values(cells)) for (const list of Object.values(row)) for (const o of list) out.push({ where: `“${q}”`, output: o });
  return out.map((d) => ({ ...d, output: normalizeOutput(d.output) }));
}

/**
 * A disclosure with a code the ASA guide gives examples for, but no ASA grade and no deliberate
 * "No ASA grade" (S44a, S39 verification 2): e.g. "From the code library" left a code with several
 * examples ungraded. One line each, naming the set, the question and the code.
 */
export function ungradedAsaWarnings(sets: Sets, guide: AsaGuide): string[] {
  const out: string[] = [];
  for (const s of sets) {
    for (const { el } of allElements(s.doc)) {
      for (const { where, output } of disclosuresOf(el)) {
        if (output.asa || output.asaNone) continue;
        const graded = output.codes.filter((c) => (guide(c) ?? []).length > 0);
        if (!graded.length) continue;
        const grades = [...new Set(graded.flatMap((c) => guide(c)!.map((x) => x.grade)))].join(" or ");
        const names = graded.map((c) => `“${c.display} (${c.code})”`).join(", ");
        out.push(`${s.name}: ${where} has ${names}, which the ASA guide grades ASA ${grades}, but no ASA grade was chosen. Choose one in its disclosure, or No ASA grade.`);
      }
    }
  }
  return out;
}

/** Everything the Publish window warns about without blocking (S44a). `guide` is the ASA guide, once fetched. */
export function publishWarnings(sets: Sets, rules: RuleSet | null | undefined, guide?: AsaGuide): string[] {
  return [...autoTriageWarnings(sets, rules), ...(guide ? ungradedAsaWarnings(sets, guide) : [])];
}
