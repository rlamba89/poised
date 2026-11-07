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
import {
  APFEL, APFEL_KEYS, apfelClinicianPart, apfelGroup, DASI, DASI_KEYS, dasiGroup, PRISMA_7, PRISMA7_KEYS, prisma7Group, STOP_BANG, STOP_BANG_KEYS as K, stopBangGroup,
} from "./ready";
import { hqSets, type Answers } from "./respondent";
import { modelFor, type SamplePatient } from "./testcases";
import { bandIn, isGuardedScore } from "./scores";
import { showBands, INCOMPLETE } from "./questionTypes";

export type ItemSource = "hq" | "observation" | "profile" | "clinician";

/** The BMI the assessment's STOP-Bang uses: a number the pre-fill (or the clinician) gives. */
export const STOP_BANG_BMI = "stopbang_bmi";

/** One item as the clinician sees it: its number or letter, what it asks, its value and source. */
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

type ItemDef = {
  key: string; letter: string; label: string; kind: AssessmentItem["kind"]; scores: (v: unknown) => boolean;
  /** What Go's messages call it, if not its label ("BMI: a number from 5 to 150."). */
  short?: string;
};
const isYes = (v: unknown): boolean => v === "o_yes";
const yesNo = (key: string, letter: string, label: string, scores: (v: unknown) => boolean = isYes): ItemDef => ({ key, letter, label, kind: "yesNo", scores });
const profile = (key: "patientAge" | "patientSex", letter: string, label: string, scores: (v: unknown) => boolean): ItemDef =>
  ({ key, letter, label, kind: "profile", scores, short: key === "patientAge" ? "Age" : "Sex" });

/**
 * An assessment: its id and name, its locked content (built from the ready-made score in ready.ts,
 * so the HQ and the assessment share wording, points and keys), and its items in the published order.
 */
type AssessmentDef = { id: string; name: string; elements: () => ElementJson[]; items: ItemDef[] };

const DASI_LABELS = [
  "Self-care", "Walk indoors", "Walk a block or two", "Stairs or a hill", "Run a short distance", "Light housework", "Moderate housework",
  "Heavy housework", "Yardwork", "Sexual relations", "Moderate recreation", "Strenuous sports",
];
const PRISMA7_LABELS = ["Health limits activities", "Needs regular help", "Health keeps them at home", "Someone close to count on", "Stick, walker or wheelchair"];

const DEFS: AssessmentDef[] = [
  {
    id: STOP_BANG.id,
    name: STOP_BANG.name,
    elements: () => [
      { type: "text", name: STOP_BANG_BMI, title: "BMI (kg/m²)", inputType: "number", min: 5, max: 150, isRequired: true },
      stopBangGroup(STOP_BANG_BMI),
    ],
    // In the acronym's order. The same limits as the ready-made score's lines (ready.ts); assessments.test.ts checks they agree.
    items: [
      yesNo(K.snoring, "S", "Snoring"),
      yesNo(K.tired, "T", "Tired"),
      yesNo(K.observed, "O", "Observed stopping breathing"),
      yesNo(K.pressure, "P", "High blood pressure"),
      { key: STOP_BANG_BMI, letter: "B", label: "BMI over 35", short: "BMI", kind: "number", scores: (v) => Number(v) > 35 },
      profile("patientAge", "A", "Age over 50", (v) => Number(v) > 50),
      { ...yesNo(K.neck, "N", "Neck large"), short: "Neck" },
      profile("patientSex", "G", "Gender male", (v) => v === "male"),
    ],
  },
  {
    id: DASI.id,
    name: DASI.name,
    elements: () => [dasiGroup()],
    items: DASI_KEYS.items.map((k, n) => yesNo(k, String(n + 1), DASI_LABELS[n])),
  },
  {
    id: PRISMA_7.id,
    name: PRISMA_7.name,
    elements: () => [prisma7Group()],
    items: [
      profile("patientAge", "1", "Older than 85", (v) => Number(v) > 85),
      profile("patientSex", "2", "Male", (v) => v === "male"),
      ...PRISMA7_KEYS.items.map((k, n) => yesNo(k, String(n + 3), PRISMA7_LABELS[n])),
    ],
  },
  {
    id: APFEL.id,
    name: APFEL.name,
    // S48 answers 4: the HQ's two questions, then the clinician's opioids item and the score.
    elements: () => [apfelGroup(), ...apfelClinicianPart()],
    items: [
      profile("patientSex", "1", "Female", (v) => v === "female"),
      yesNo(APFEL_KEYS.smoking, "2", "Smoking", (v) => v === "o_no"), // a non-smoker's point
      yesNo(APFEL_KEYS.ponv, "3", "PONV or travel sickness before"),
      yesNo(APFEL_KEYS.opioids, "4", "Opioids after surgery"),
    ],
  },
];

/** The assessments a clinician can add to an episode, in the order they're offered. */
export const ASSESSMENTS: { id: string; name: string }[] = DEFS.map(({ id, name }) => ({ id, name }));

/**
 * An assessment's locked SurveyJS JSON, or undefined for one Preopsis hasn't. `labels` gives what
 * Go's messages call each item, by key ("Answer every item before confirming: Neck has no answer.").
 */
export function assessmentContent(id: string): (ChapterJson & { labels: Record<string, string> }) | undefined {
  const def = DEFS.find((d) => d.id === id);
  if (!def) return undefined;
  return {
    pages: [{ name: `p_${def.id}`, title: def.name, elements: def.elements() }],
    labels: Object.fromEntries(def.items.map((i) => [i.key, i.short ?? i.label])),
  };
}

/** The items of the assessment a content is: the definition whose questions it has. */
function itemsOf(content: ChapterJson): ItemDef[] {
  const names = new Set(allElements(content).map(({ el }) => el.name));
  return DEFS.find((d) => d.items.every((i) => i.kind === "profile" || names.has(i.key)))?.items ?? [];
}

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

/**
 * What the pre-fill reads from the HQ, worked out once per HQ (`from.hq`, the same object while the
 * HQ, its answers and the patient are unchanged) and set of keys: every survey it takes is built
 * once, not again on each keystroke or for each card (S47 verifications 2–4).
 */
const hqParts = new WeakMap<PrefillFrom["hq"], Map<string, { asked: Answers; reported?: number }>>();
function hqPart(hq: PrefillFrom["hq"], keys: string[]): { asked: Answers; reported?: number } {
  let byKeys = hqParts.get(hq);
  if (!byKeys) hqParts.set(hq, (byKeys = new Map()));
  const k = keys.join(",");
  let part = byKeys.get(k);
  if (!part) {
    part = { asked: hqValues(hq, keys), reported: reportedBmi(hq.sets, (id) => hq.answers[id], hq.patient) };
    byKeys.set(k, part);
  }
  return part;
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
  const defs = itemsOf(content);
  const { asked, reported } = hqPart(from.hq, defs.filter((d) => d.kind === "yesNo").map((d) => d.key));
  // A BMI outside the question's limits isn't used (Go would refuse it): the other one, else missing.
  const bmiEl = els.get(STOP_BANG_BMI);
  const fits = (v: number | undefined) =>
    v !== undefined && (typeof bmiEl?.min !== "number" || v >= bmiEl.min) && (typeof bmiEl?.max !== "number" || v <= bmiEl.max) ? v : undefined;
  const own = fits(reported);
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
  const defs = itemsOf(content);
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
 * out by its own score expression, as in the HQ, then its other Calculations as they show (DASI's
 * "34 (Poor functional capacity), 6.9 METs"); "Incomplete" while an item is missing. An age and
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
  // Its other Calculations, as they show (DASI's "6.9 METs"), after the score.
  const extras = allElements(content)
    .filter(({ el }) => el.type === "expression" && el.name !== calc.name)
    .map(({ el }) => m.getQuestionByName(el.name) as unknown as { value: unknown; formatedValue: string })
    .filter((x) => x && !empty(x.value))
    .map((x) => x.formatedValue);
  m.dispose();
  if (empty(value)) return { text: INCOMPLETE };
  const score = Number(value);
  const text = [band ? `${score} (${band.label})` : String(score), ...extras].join(", ");
  return band ? { score, band: band.label, text } : { score, text };
}
