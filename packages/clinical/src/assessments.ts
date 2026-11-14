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
import { reportedBody } from "./poa";
import {
  APFEL, APFEL_KEYS, apfelClinicianPart, apfelGroup, CFS_KEYS, cfsElements, DASI, DASI_KEYS, dasiGroup, MORSE, MORSE_KEYS, morseClinicianPart, morseGroup, MUST,
  MUST_KEYS, mustClinicianPart, PRISMA_7, PRISMA7_KEYS, prisma7Group, STOP_BANG, STOP_BANG_KEYS as K, stopBangGroup, VTE, VTE_ITEMS, VTE_KEYS, vteClinicianPart,
  vteGroup,
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
  /** choice: an option answer (Yes or No, or one of several); number: a measurement; profile: from the patient's record (frozen at Confirm). */
  kind: "choice" | "number" | "profile";
  /** A number item the observations fill: the BMI or the weight (the hospital's, or the patient's own, as the trust says). */
  measure?: "bmi" | "weight";
  /**
   * A measure the pre-fill has no value for (MUST's BMI where the hospital has a weight but no
   * height): one the clinician types here fills a gap, so it keeps the other measure beside it
   * (S49 verification 5, Rahul 13 Nov). Worked out once, from the pre-fill alone, so a saved typed
   * value changed later still fills the gap (S49a).
   */
  fillsGap?: true;
  /** The question's wording, for choice and number items. */
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
  key: string; letter: string; label: string; kind: AssessmentItem["kind"];
  /** Whether a value scores the item's point; none for an item that scores nothing alone (MUST's weight). */
  scores?: (v: unknown) => boolean;
  /** What Go's messages call it, if not its label ("BMI: a number from 5 to 150."). */
  short?: string;
  measure?: AssessmentItem["measure"];
  /** Read from the HQ by other keys (S49: MUST's weight lost, from "lost weight?" and "how much"), else by its own key. */
  fromHq?: { keys: string[]; value: (asked: Answers) => unknown };
  /** Never pre-filled from an HQ: a clinician's judgement (the Clinical Frailty Scale). */
  noHq?: true;
};
const isYes = (v: unknown): boolean => v === "o_yes";
const yesNo = (key: string, letter: string, label: string, scores: (v: unknown) => boolean = isYes): ItemDef => ({ key, letter, label, kind: "choice", scores });
/** A choice whose answer scores unless it's the first option (the one worth 0). */
const choice = (key: string, letter: string, label: string, zero: string): ItemDef => ({ key, letter, label, kind: "choice", scores: (v) => v !== zero });
const measured = (key: string, letter: string, label: string, measure: "bmi" | "weight", scores?: (v: unknown) => boolean, short?: string): ItemDef =>
  ({ key, letter, label, kind: "number", measure, scores, short });
const profile = (key: "patientAge" | "patientSex", letter: string, label: string, scores: (v: unknown) => boolean): ItemDef =>
  ({ key, letter, label, kind: "profile", scores, short: key === "patientAge" ? "Age" : "Sex" });

/**
 * An assessment: its id and name, its locked content (built from the ready-made score in ready.ts,
 * so the HQ and the assessment share wording, points and keys), and its items in the published order.
 */
type AssessmentDef = {
  id: string; name: string; elements: () => ElementJson[]; items: ItemDef[];
  /** Other assessments whose confirmed results the card shows as hints (Rockwood: DASI and PRISMA-7, S49). */
  hints?: string[];
};

const DASI_LABELS = [
  "Self-care", "Walk indoors", "Walk a block or two", "Stairs or a hill", "Run a short distance", "Light housework", "Moderate housework",
  "Heavy housework", "Yardwork", "Sexual relations", "Moderate recreation", "Strenuous sports",
];
const PRISMA7_LABELS = ["Health limits activities", "Needs regular help", "Health keeps them at home", "Someone close to count on", "Stick, walker or wheelchair"];

/**
 * A ready-made HQ part as an assessment holds it: its questions without the HQ's conditions (the
 * clinician answers every item) and without "always ask fresh" (pre-fill between HQs only).
 */
const withoutConditions = (g: ElementJson): ElementJson => ({
  ...g, elements: (g.elements ?? []).map(({ visibleIf: _if, alwaysFresh: _fresh, ...e }) => e),
});

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
      measured(STOP_BANG_BMI, "B", "BMI over 35", "bmi", (v) => Number(v) > 35, "BMI"),
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
  {
    id: MUST.id,
    name: MUST.name,
    // S49 answers 3: the BMI and today's weight from the observations, the weight lost from the HQ.
    elements: () => mustClinicianPart(),
    items: [
      measured(MUST_KEYS.bmi, "1", "BMI", "bmi", (v) => Number(v) <= 20),
      measured(MUST_KEYS.weight, "2", "Weight now", "weight"),
      {
        key: MUST_KEYS.lostKg, letter: "3", label: "Unplanned weight loss in the past 3–6 months (kg)", short: "Weight lost", kind: "number",
        // A No is none lost; a Yes needs how much (shown only then).
        fromHq: { keys: [MUST_KEYS.lost, MUST_KEYS.lostKg], value: (a) => (a[MUST_KEYS.lost] === "o_no" ? 0 : a[MUST_KEYS.lost] === "o_yes" ? a[MUST_KEYS.lostKg] : undefined) },
      },
      yesNo(MUST_KEYS.acute, "4", "Acute disease effect"),
    ],
  },
  {
    id: MORSE.id,
    name: MORSE.name,
    elements: () => [withoutConditions(morseGroup()), ...morseClinicianPart()],
    items: [
      yesNo(MORSE_KEYS.falls, "1", "History of falling"),
      yesNo(MORSE_KEYS.secondary, "2", "Secondary diagnosis"),
      choice(MORSE_KEYS.aid, "3", "Ambulatory aid", "o_aid_none"),
      yesNo(MORSE_KEYS.iv, "4", "IV or heparin lock"),
      choice(MORSE_KEYS.gait, "5", "Gait", "o_gait_normal"),
      choice(MORSE_KEYS.mental, "6", "Mental status", "o_mental_oriented"),
    ],
  },
  {
    id: VTE.id,
    name: VTE.name,
    // The patient's part as the HQ has it (vteGroup), then the clinician's, worded as the tool words
    // them. Without the HQ's conditions: the clinician answers every item, a man's HRT too, and it counts.
    elements: () => [withoutConditions(vteGroup()), ...vteClinicianPart()],
    items: [
      ...VTE_ITEMS.thrombosis.map(([key, label], n): ItemDef =>
        key === "patientAge" ? profile("patientAge", `T${n + 1}`, label, (v) => Number(v) > 60)
          : key === VTE_KEYS.bmi ? measured(key, `T${n + 1}`, label, "bmi", (v) => Number(v) > 30, "BMI")
            : yesNo(key, `T${n + 1}`, label)),
      ...VTE_ITEMS.bleeding.map(([key, label], n) => yesNo(key, `B${n + 1}`, label)),
    ],
  },
  {
    id: "rockwood",
    name: "Clinical Frailty Scale",
    // A clinician's judgement (never in an HQ), with DASI's and PRISMA-7's confirmed results as hints.
    elements: () => cfsElements(),
    items: [{ key: CFS_KEYS.level, letter: "1", label: "Level", kind: "choice", noHq: true }],
    hints: [DASI.id, PRISMA_7.id],
  },
];

/** The assessments a clinician can add to an episode, in the order they're offered. */
export const ASSESSMENTS: { id: string; name: string }[] = DEFS.map(({ id, name }) => ({ id, name }));

/** The assessments whose confirmed results an assessment's card shows as hints (Rockwood's: DASI and PRISMA-7). */
export const assessmentHints = (id: string): string[] => DEFS.find((d) => d.id === id)?.hints ?? [];

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
  /** The latest weight that counts on the episode, if any (S49: MUST). */
  hospitalWeight?: number;
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
const hqParts = new WeakMap<PrefillFrom["hq"], Map<string, Answers>>();
function hqPart(hq: PrefillFrom["hq"], keys: string[]): Answers {
  let byKeys = hqParts.get(hq);
  if (!byKeys) hqParts.set(hq, (byKeys = new Map()));
  const k = keys.join(",");
  let asked = byKeys.get(k);
  if (!asked) byKeys.set(k, (asked = keys.length ? hqValues(hq, keys) : {})); // the Frailty Scale asks for none
  return asked;
}

/** The patient's own BMI and weight from the HQ, worked out once per HQ, and only for a card that measures (verification 1, finding 6). */
const reportedParts = new WeakMap<PrefillFrom["hq"], { body?: ReturnType<typeof reportedBody> }>();
function reportedPart(hq: PrefillFrom["hq"]): ReturnType<typeof reportedBody> {
  let part = reportedParts.get(hq);
  if (!part) reportedParts.set(hq, (part = { body: reportedBody(hq.sets, (id) => hq.answers[id], hq.patient) }));
  return part.body;
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
  const hqKeys = defs.flatMap((d) => (d.noHq || d.kind === "profile" || d.measure ? [] : (d.fromHq?.keys ?? [d.key])));
  const asked = hqPart(from.hq, hqKeys);
  const reported = defs.some((d) => d.measure) ? reportedPart(from.hq) : undefined;
  // A value outside the question's limits isn't used (Go would refuse it): the other one, else missing.
  const fits = (key: string, v: unknown) => {
    const el = els.get(key);
    const n = typeof v === "number" ? v : typeof v === "string" && v.trim() !== "" ? Number(v) : NaN;
    return Number.isFinite(n) && (typeof el?.min !== "number" || n >= el.min) && (typeof el?.max !== "number" || n <= el.max) ? n : undefined;
  };
  /**
   * A measurement: the hospital's or the patient's own, as the trust's precedence says. A card with a
   * BMI and a weight (MUST) takes both from one source (verification 1, Rahul's answer 1): the
   * hospital's whenever it has a weight, so a BMI the hospital can't work out (no height) is left
   * for the clinician, never the patient's own BMI beside the hospital's weight.
   */
  const keyOf = (m: "bmi" | "weight") => defs.find((d) => d.measure === m)?.key;
  const paired = !!keyOf("bmi") && !!keyOf("weight");
  const hospitalOf = (m: "bmi" | "weight") => (m === "bmi" ? from.hospitalBmi : from.hospitalWeight);
  const pairSource = (): "observation" | "hq" | undefined => {
    const hospital = fits(keyOf("weight")!, from.hospitalWeight) !== undefined;
    const own = fits(keyOf("weight")!, reported?.weight) !== undefined;
    if (from.obsPrecedence) return hospital ? "observation" : own ? "hq" : undefined;
    // Off: the patient's, unless only the hospital's gives a BMI (verification 2, Rahul 13 Nov).
    const whole = (bmi: number | undefined) => fits(keyOf("bmi")!, bmi) !== undefined;
    if (own && hospital && !whole(reported?.bmi) && whole(from.hospitalBmi)) return "observation";
    return own ? "hq" : hospital ? "observation" : undefined;
  };
  const pair = paired ? pairSource() : undefined;
  const measurement = (key: string, measure: "bmi" | "weight"): Pick<AssessmentItem, "value" | "source"> => {
    const own = fits(key, reported?.[measure]);
    const hospital = fits(key, hospitalOf(measure));
    if (paired) {
      const v = pair === "observation" ? hospital : pair === "hq" ? own : undefined;
      return v === undefined ? {} : { value: v, source: pair };
    }
    return hospital !== undefined && (from.obsPrecedence || own === undefined) ? { value: hospital, source: "observation" }
      : own !== undefined ? { value: own, source: "hq" } : {};
  };
  const items = defs.map((d) => {
    const item = itemOf(d);
    const scored = item.source && d.scores ? { ...item, yes: d.scores(item.value) } : item;
    return d.measure && measurement(d.key, d.measure).source === undefined ? { ...scored, fillsGap: true as const } : scored;
  });
  return saved?.confirmed ? items : unpaired(items);
  function itemOf({ scores: _scores, fromHq, noHq, ...d }: ItemDef): AssessmentItem {
    const el = els.get(d.key);
    const base: AssessmentItem = { ...d, title: el ? textOf(el.title) : undefined, options: d.kind === "choice" ? optionsOf(el) : undefined };
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
    if (noHq) return base;
    if (d.measure) return { ...base, ...measurement(d.key, d.measure) };
    const v = fromHq ? fromHq.value(asked) : asked[d.key];
    if (d.kind === "number") {
      const n = fits(d.key, v);
      return n === undefined ? base : { ...base, value: n, source: "hq" };
    }
    return base.options?.some((o) => o.value === v) ? { ...base, value: v, source: "hq" } : base;
  }
}

/**
 * The items with the clinician's changes on this screen (`draft`, by key) in place: each changed
 * one is the clinician's. A change back to the value shown keeps where that came from, except on a
 * card with a BMI and a weight (MUST) once the other is the clinician's: then a number typed in
 * this box is the clinician's too, even the pre-filled one typed back (S49 verification 4,
 * finding 1). A profile item can't be changed. The one-body rule (`unpaired`) runs only when a
 * BMI or weight changed here, so an unchanged confirmed card shows as confirmed (finding 2).
 */
export function withDraft(content: ChapterJson, items: AssessmentItem[], draft: Answers): AssessmentItem[] {
  const defs = itemsOf(content);
  const typed = (i: AssessmentItem) => i.kind !== "profile" && i.key in draft && !empty(draft[i.key]);
  const changed = (i: AssessmentItem) => typed(i) && draft[i.key] !== i.value;
  const measures = items.filter((i) => i.measure);
  const otherIsClinicians = (i: AssessmentItem) =>
    measures.length === 2 && measures.some((o) => o !== i && (o.source === "clinician" || changed(o)));
  const out = items.map((i) => {
    const d = defs.find((x) => x.key === i.key);
    if (!d || !(changed(i) || (typed(i) && i.measure && otherIsClinicians(i)))) return i;
    return { ...i, value: draft[i.key], source: "clinician" as const, yes: d.scores?.(draft[i.key]) };
  });
  return out.some((o, n) => o !== items[n] && o.measure) ? unpaired(out) : out;
}

/**
 * An item as it shows with every change on this screen but its own (`key`): what its box showed
 * before the clinician typed in it. A box the pairing rule emptied showed nothing (verification 5,
 * finding 2).
 */
export function shownWithout(content: ChapterJson, items: AssessmentItem[], draft: Answers, key: string): AssessmentItem | undefined {
  const { [key]: _mine, ...rest } = draft;
  return withDraft(content, items, rest).find((i) => i.key === key);
}

/**
 * The screen's changes after one more (`value` in `key`'s box; undefined for an emptied box). Any
 * box whose key changes nothing on the card (every item shows the same value, from the same place,
 * without it) drops its key, so a box typed in and put back counts as untouched (S49 verification
 * 5, finding 1), and so does one typed in earlier that putting the other box back has made
 * untouched again (verification 6): the draft keeps what was changed, not every box typed in.
 */
export function draftAfter(content: ChapterJson, items: AssessmentItem[], draft: Answers, key: string, value: unknown): Answers {
  const { [key]: _old, ...rest } = draft;
  let out: Answers = empty(value) ? rest : { ...rest, [key]: value };
  const same = (a: AssessmentItem[], b: AssessmentItem[]) => a.every((x, n) => x.value === b[n].value && x.source === b[n].source);
  for (let dropped = true; dropped; ) {
    dropped = false;
    const shown = withDraft(content, items, out);
    for (const k of Object.keys(out)) {
      const { [k]: _k, ...without } = out;
      if (same(shown, withDraft(content, items, without))) {
        out = without;
        dropped = true;
        break;
      }
    }
  }
  return out;
}

/**
 * On a card with both a BMI and a weight (MUST), one the clinician entered leaves the other, if
 * pre-filled, Missing for them to enter: a BMI and a weight on one card always describe one body
 * (S49 verification 3, Rahul 13 Nov; the pre-fill's own rule is the pair's source). Both entered:
 * both kept. One entered in a box the pre-fill left empty (`fillsGap`: the hospital's weight with
 * no height, so the BMI is the clinician's) keeps the other (verification 5, Rahul 13 Nov), saved
 * or not (S49a). A card with a BMI only is unchanged.
 */
function unpaired(items: AssessmentItem[]): AssessmentItem[] {
  const bmi = items.find((i) => i.measure === "bmi");
  const weight = items.find((i) => i.measure === "weight");
  if (!bmi || !weight) return items;
  const typed = (i: AssessmentItem) => i.source === "clinician";
  const clears = (mine: AssessmentItem, other: AssessmentItem) => typed(mine) && !mine.fillsGap && !!other.source && !typed(other);
  const other = clears(weight, bmi) ? bmi : clears(bmi, weight) ? weight : undefined;
  return other ? items.map((i) => (i === other ? { ...i, value: undefined, source: undefined, yes: undefined } : i)) : items;
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
    .map(({ el }) => m.getQuestionByName(el.name) as unknown as { value: unknown; formatedValue: string } | null)
    .filter((x) => !!x)
    .map((x) => ({ value: x.value, formatedValue: x.formatedValue })); // read before the model goes (verification 4, finding 3)
  m.dispose();
  // Incomplete while any result is: VTE's bleeding count has inputs its first result hasn't
  // (S49 verification 2, finding 1).
  if (empty(value) || extras.some((x) => empty(x.value))) return { text: INCOMPLETE };
  const score = Number(value);
  const text = [band ? `${score} (${band.label})` : String(score), ...extras.map((x) => x.formatedValue)].join(", ");
  return band ? { score, band: band.label, text } : { score, text };
}
