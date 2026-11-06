// Assessments (plan C7, session S47; §9 "Assessments through the HQ"): a validated instrument a
// clinician completes on an episode, such as STOP-Bang. Its definition is the ready-made score's
// (ready.ts), so the editor, the HQ and the assessment share one wording, scoring and set of keys;
// Go stores it as a platform questionnaire (apps/api/internal/assessment/<id>.json, written from
// here by `npm run assessments -w packages/clinical`).
//
// Each item is pre-filled, with where its value came from (its source):
//   - hq: by key, from the HQ's answers that count now (the clinician's copy once started, else the
//     patient's), only where the question is shown: one a condition hides counts as unanswered;
//   - observation: the hospital BMI (observations.ts), or the patient's own, as the trust's
//     observations.precedence setting says (OBS-04, S47 answers 3);
//   - profile: the patient's age and sex, from their record (stored at Confirm, then frozen: answers 7);
//   - clinician: what the clinician entered (the neck, S47 answers 2, or a change to any other).
// A clinician confirms every assessment (S47 answers 4), even one the HQ filled completely.
import type { Question } from "survey-core";
import { allElements, textOf, type ChapterJson, type ElementJson } from "./doc";
import { registerClinicalProperties } from "./properties";
import { reportedBmi } from "./poa";
import { STOP_BANG, STOP_BANG_KEYS as K, stopBangGroup } from "./ready";
import { hqSets, type Answers } from "./respondent";
import { modelFor, type SamplePatient } from "./testcases";
import { bandIn, isGuardedScore } from "./scores";
import { showBands, INCOMPLETE } from "./questionTypes";

export type ItemSource = "hq" | "observation" | "profile" | "clinician";

/** The assessments a clinician can add to an episode. */
export const ASSESSMENTS: { id: string; name: string }[] = [{ id: STOP_BANG.id, name: STOP_BANG.name }];

/** The BMI the assessment's STOP-Bang uses: a number the pre-fill (or the clinician) gives. */
export const STOP_BANG_BMI = "stopbang_bmi";

/** An assessment's locked SurveyJS JSON, or undefined for one Preopsis hasn't. */
export function assessmentContent(id: string): ChapterJson | undefined {
  if (id !== STOP_BANG.id) return undefined;
  return {
    pages: [{
      name: "p_stopbang",
      title: STOP_BANG.name,
      elements: [
        { type: "text", name: STOP_BANG_BMI, title: "BMI (kg/m²)", inputType: "number", min: 5, max: 150, isRequired: true },
        stopBangGroup(STOP_BANG_BMI),
      ],
    }],
  };
}

/** One item as the clinician sees it: the acronym's letter, what it asks, its value and source. */
export type AssessmentItem = {
  key: string;
  letter: string;
  label: string;
  /** yesNo: an option answer; number: a measurement; profile: from the patient's record (frozen at Confirm). */
  kind: "yesNo" | "number" | "profile";
  /** The question's wording, for yes/no and number items. */
  title?: string;
  options?: { value: string; text: string }[];
  /** A number item's limits, as the question sets them (Go refuses a value outside them). */
  min?: number;
  max?: number;
  value?: unknown;
  source?: ItemSource;
  /** Whether its value scores the item's point (a Yes, a BMI over 35…); undefined while missing. */
  yes?: boolean;
};

type ItemDef = { key: string; letter: string; label: string; kind: AssessmentItem["kind"]; scores: (v: unknown) => boolean };
const isYes = (v: unknown) => v === "o_yes";
/** STOP-Bang's eight items, in the acronym's order. */
const STOP_BANG_ITEMS: ItemDef[] = [
  { key: K.snoring, letter: "S", label: "Snoring", kind: "yesNo", scores: isYes },
  { key: K.tired, letter: "T", label: "Tired", kind: "yesNo", scores: isYes },
  { key: K.observed, letter: "O", label: "Observed stopping breathing", kind: "yesNo", scores: isYes },
  { key: K.pressure, letter: "P", label: "High blood pressure", kind: "yesNo", scores: isYes },
  // The same limits as the ready-made score's lines (ready.ts); assessments.test.ts checks they agree.
  { key: STOP_BANG_BMI, letter: "B", label: "BMI over 35", kind: "number", scores: (v) => Number(v) > 35 },
  { key: "patientAge", letter: "A", label: "Age over 50", kind: "profile", scores: (v) => Number(v) > 50 },
  { key: K.neck, letter: "N", label: "Neck large", kind: "yesNo", scores: isYes },
  { key: "patientSex", letter: "G", label: "Gender male", kind: "profile", scores: (v) => v === "male" },
];

/** The answers a confirmed or saved assessment holds, with each one's source. */
export type SavedAssessment = { data: Answers; sources: Record<string, ItemSource>; confirmed: boolean };

/** Where the pre-fill reads from. */
export type PrefillFrom = {
  /** The HQ's Question Sets, the answers that count now (countingHqAnswers), and the patient as the HQ saw them. */
  hq: { sets: { id: string; name: string; content: ChapterJson }[]; answers: Record<string, Answers | undefined>; patient: SamplePatient };
  /** The patient now (or on the day it was confirmed): their age and sex. */
  patient: SamplePatient;
  /** The hospital BMI, if the episode has one (hospitalBmi in observations.ts). */
  hospitalBmi?: number;
  /** The trust's observations.precedence: the hospital BMI first (on), or the patient's own (off). */
  obsPrecedence: boolean;
};

/**
 * The HQ's answers that count now, per set: the clinician's copy once they've started it (validated
 * or not, S47 answers 5, as the rest of the episode page), else the patient's.
 */
export function countingHqAnswers(
  sets: { id: string }[], patient: Record<string, Answers | undefined>, clinician: Record<string, { data: Answers; validatedAt?: string | null } | undefined>,
): Record<string, Answers | undefined> {
  return Object.fromEntries(sets.map((s) => [s.id, clinician[s.id]?.data ?? patient[s.id]]));
}

const empty = (v: unknown) => v === undefined || v === null || v === "";

/** The HQ's answers to `keys`, by key, from the sets and questions shown only (the clinician's view). */
function hqValues(from: PrefillFrom["hq"], keys: string[]): Answers {
  registerClinicalProperties();
  const out: Answers = {};
  const { shown, earlier } = hqSets(from.sets, from.answers, "clinician", from.patient);
  for (const set of shown) {
    const wanted = keys.filter((k) => !(k in out) && allElements(set.content).some(({ el }) => el.name === k));
    if (!wanted.length) continue;
    const data = from.answers[set.id] ?? {};
    const m = modelFor(set.content, "clinician", from.patient, earlier.get(set.id));
    m.data = data;
    const visible = new Set((m.getAllQuestions(true) as Question[]).map((q) => q.name)); // skips hidden pages and groups too
    m.dispose();
    for (const k of wanted) if (visible.has(k) && !empty(data[k])) out[k] = data[k];
  }
  return out;
}

const optionsOf = (el: ElementJson | undefined) =>
  Array.isArray(el?.choices) ? (el.choices as { value: string; text: unknown }[]).map((c) => ({ value: String(c.value), text: textOf(c.text) })) : [];

/**
 * An assessment's items with their values and sources: pre-filled from the HQ, the observations and
 * the profile; a missing one has no source. `saved` is the clinician's row: while it isn't
 * confirmed, the items the clinician entered are kept and the rest are filled afresh (the HQ or a
 * reading may have changed); once confirmed, it's exactly what was confirmed.
 */
export function assessmentItems(content: ChapterJson, from: PrefillFrom, saved?: SavedAssessment): AssessmentItem[] {
  const els = new Map(allElements(content).map(({ el }) => [el.name, el]));
  const defs = els.has(K.group) ? STOP_BANG_ITEMS : [];
  const asked = hqValues(from.hq, defs.filter((d) => d.kind === "yesNo").map((d) => d.key));
  // A BMI outside the question's limits isn't used (Go would refuse it): the other one, else missing.
  const bmiEl = els.get(STOP_BANG_BMI);
  const fits = (v: number | undefined) =>
    v !== undefined && (typeof bmiEl?.min !== "number" || v >= bmiEl.min) && (typeof bmiEl?.max !== "number" || v <= bmiEl.max) ? v : undefined;
  const own = fits(reportedBmi(from.hq.sets, (id) => from.hq.answers[id], from.hq.patient));
  const hospital = fits(from.hospitalBmi);
  const bmi: Pick<AssessmentItem, "value" | "source"> =
    hospital !== undefined && (from.obsPrecedence || own === undefined) ? { value: hospital, source: "observation" }
      : own !== undefined ? { value: own, source: "hq" } : {};
  return defs.map((d) => {
    const item = itemOf(d);
    return item.source ? { ...item, yes: d.scores(item.value) } : item;
  });
  function itemOf({ scores: _scores, ...d }: ItemDef): AssessmentItem {
    const el = els.get(d.key);
    const base: AssessmentItem = { ...d, title: el ? textOf(el.title) : undefined, options: d.kind === "yesNo" ? optionsOf(el) : undefined };
    if (d.kind === "number") Object.assign(base, { min: el?.min as number | undefined, max: el?.max as number | undefined });
    if (d.kind === "profile") {
      // Frozen at Confirm (S47 answers 7); one confirmed before they were stored reads the record.
      if (saved?.confirmed && !empty(saved.data[d.key])) return { ...base, value: saved.data[d.key], source: "profile" };
      const v = d.key === "patientAge" ? from.patient.age : from.patient.sex;
      const known = d.key === "patientAge" ? typeof v === "number" : v === "male" || v === "female";
      return known ? { ...base, value: v, source: "profile" } : base;
    }
    const keep = saved && !empty(saved.data[d.key]) && (saved.confirmed || saved.sources[d.key] === "clinician");
    if (keep) return { ...base, value: saved.data[d.key], source: saved.sources[d.key] ?? "clinician" };
    if (d.kind === "number") return { ...base, ...bmi };
    const v = asked[d.key];
    return base.options?.some((o) => o.value === v) ? { ...base, value: v, source: "hq" } : base;
  }
}

/**
 * The items with the clinician's changes on this screen (`draft`, by key) in place: each changed
 * one is the clinician's. A change back to the value shown keeps where that came from. A profile
 * item can't be changed.
 */
export function withDraft(content: ChapterJson, items: AssessmentItem[], draft: Answers): AssessmentItem[] {
  const defs = allElements(content).some(({ el }) => el.name === K.group) ? STOP_BANG_ITEMS : [];
  return items.map((i) => {
    const d = defs.find((x) => x.key === i.key);
    if (!d || i.kind === "profile" || !(i.key in draft) || empty(draft[i.key]) || draft[i.key] === i.value) return i;
    return { ...i, value: draft[i.key], source: "clinician", yes: d.scores(draft[i.key]) };
  });
}

/** How many items have a value. */
export const filledCount = (items: AssessmentItem[]) => items.filter((i) => i.source).length;

/**
 * What the clinician's row stores: each item's value and source, the patient's age and sex too, so
 * a confirmed score never moves with a later correction to their record (S47 answers 7).
 */
export function assessmentAnswers(items: AssessmentItem[]): { data: Answers; sources: Record<string, ItemSource> } {
  const data: Answers = {};
  const sources: Record<string, ItemSource> = {};
  for (const i of items) {
    if (!i.source) continue;
    data[i.key] = i.value;
    sources[i.key] = i.source;
  }
  return { data, sources };
}

/**
 * The assessment's score and band for these answers and this patient, e.g. "5 (High risk)", worked
 * out by its own score expression, as in the HQ; "Incomplete" while an item is missing. An age and
 * sex stored with the answers (at Confirm) win over the patient's.
 */
export function assessmentResult(content: ChapterJson, data: Answers, patient: SamplePatient): { score?: number; band?: string; text: string } {
  registerClinicalProperties();
  const calc = allElements(content).find(({ el }) => el.type === "expression" && isGuardedScore(el.expression))?.el;
  if (!calc) return { text: INCOMPLETE };
  const { patientAge, patientSex, ...answers } = data;
  const who: SamplePatient = {
    ...patient,
    ...(empty(patientAge) ? {} : { age: Number(patientAge) }),
    ...(empty(patientSex) ? {} : { sex: String(patientSex) }),
  };
  const m = modelFor(content, "clinician", who);
  showBands(m);
  m.data = answers;
  const q = m.getQuestionByName(calc.name) as unknown as { value: unknown };
  const value = q.value;
  const band = bandIn(m, calc.bands, value);
  m.dispose();
  if (empty(value)) return { text: INCOMPLETE };
  const score = Number(value);
  return band ? { score, band: band.label, text: `${score} (${band.label})` } : { score, text: String(score) };
}
