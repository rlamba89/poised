// The episode's assessments (plan C7, session S47; §9): STOP-Bang, added by a clinician, pre-filled
// from the HQ, the observations and the profile (@poised/clinical assessments.ts), completed and
// confirmed. Types, the request bodies and the words; the card is AssessmentsPanel.tsx.
import { useCallback, useEffect, useState } from "react";
import {
  assessmentAnswers, assessmentHints, assessmentItems, assessmentResult, countingHqAnswers, hospitalBmi, latest, patientVariables, withDraft,
  type Answers, type AssessmentItem, type ChapterJson, type ItemSource, type PrefillFrom, type Reading,
} from "@poised/clinical";
import { api, ApiError } from "@/lib/api";
import { deploymentTimeZone } from "@/lib/config";
import { formatDateTime } from "@/lib/format";
import { hqPatient, type EpisodeHQ } from "./hq";
import type { ObservationsView } from "./observations";
import type { Episode } from "./types";

/** One assessment on the episode, as GET …/assessments sends it. data is null until the first save. */
export type AssessmentRow = {
  id: string;
  assessment: string;
  name: string;
  createdAt: string;
  versionNo: number;
  chapterId: string;
  content: ChapterJson;
  data: Answers | null;
  sources: Record<string, ItemSource> | null;
  /** The clinician's row's revision; 0 before the first save. */
  revision: number;
  updatedAt: string | null;
  /** Confirm: when, by updatedByName. A later save clears it. */
  confirmedAt: string | null;
  updatedByName: string | null;
};

export type AssessmentsView = {
  assessments: AssessmentRow[];
  /** The assessments that can still be added. */
  available: { id: string; name: string }[];
  /** The trust's observations.precedence (OBS-04): the hospital BMI first. */
  obsPrecedence: boolean;
};

/**
 * The episode's assessments, loaded again after each change; null while loading, "off" with
 * assessments off at the hospital. `on` is the episode's assessmentsOn: off, the route isn't
 * called, as a refused call would write an audit row on each page open.
 */
export function useAssessments(path: string, on: boolean | undefined) {
  const [view, setView] = useState<AssessmentsView | null | "off">(null);
  const [error, setError] = useState("");
  const load = useCallback(
    () =>
      api<AssessmentsView>(`${path}/assessments`)
        .then((v) => {
          setView(v);
          setError("");
        })
        .catch((e: Error) => {
          if (e instanceof ApiError && e.status === 403) setView("off");
          else setError(e.message);
        }),
    [path],
  );
  useEffect(() => {
    if (on) load();
  }, [load, on]);
  return { view: on === false ? ("off" as const) : view, error, load };
}

/** The patient's age and sex on a day (today, or the day it was confirmed), at the hospital. */
export const patientOn = (e: Episode, at?: string) =>
  patientVariables(e, at ? new Date(at) : new Date(), e.hospitalTimeZone ?? deploymentTimeZone());

/**
 * Where the pre-fill reads from: the HQ's answers that count, the hospital BMI and the profile. The
 * patient's answers count once they've sent them: before that they may still change them.
 */
export function prefillFrom(hq: EpisodeHQ | null, e: Episode, readings: Reading[] | undefined, obsPrecedence: boolean, at?: string): PrefillFrom {
  return {
    hq: hqPart(hq, e),
    patient: patientOn(e, at),
    hospitalBmi: readings ? hospitalBmi(readings)?.bmi : undefined,
    hospitalWeight: readings ? (latest(readings, "weight")?.value ?? undefined) : undefined,
    obsPrecedence,
  };
}

/**
 * The pre-fill's HQ part, one object per loaded HQ and patient: @poised/clinical works out what it
 * reads from the HQ once per object, so the cards, a re-confirm's other day and each keystroke share
 * one walk of the HQ's surveys (S47 verifications 2–4). A reloaded HQ, a send or a corrected record
 * gives a new one.
 */
const hqParts = new WeakMap<EpisodeHQ, Map<string, PrefillFrom["hq"]>>();
function hqPart(hq: EpisodeHQ | null, e: Episode): PrefillFrom["hq"] {
  const patient = hqPatient(e);
  const make = (): PrefillFrom["hq"] => ({
    sets: hq?.chapters ?? [],
    answers: hq && e.patientSubmittedAt ? countingHqAnswers(hq.chapters, hq.patient, hq.clinician) : {},
    patient,
  });
  if (!hq) return make();
  let byPatient = hqParts.get(hq);
  if (!byPatient) hqParts.set(hq, (byPatient = new Map()));
  const key = JSON.stringify([e.patientSubmittedAt, patient]);
  let part = byPatient.get(key);
  if (!part) byPatient.set(key, (part = make()));
  return part;
}

/**
 * The card's notes on the age, for an assessment that uses it: which day's age it is (`day`: the
 * day it was confirmed, while shown as confirmed and unchanged; else today). A record with no male
 * or female sex is said in the headline (filledText, S49a).
 */
export function cardNotes(items: AssessmentItem[], day: string | undefined): string[] {
  const out: string[] = [];
  const age = items.find((i) => i.key === "patientAge");
  if (age) out.push(`The age is the patient's ${day ? `on the day it was confirmed (${age.source ? String(age.value) : "not known"})` : "today"}.`);
  return out;
}

/**
 * The day the patient's age is worked out on: the day it was confirmed while it's shown as
 * confirmed and unchanged, else today, the day a Confirm records (and the POA Summary then uses).
 */
export const scoreDay = (row: AssessmentRow, changed: boolean): string | undefined =>
  row.confirmedAt && !changed ? row.confirmedAt : undefined;

/**
 * Whether the card's inputs are in: the HQ, and the observations unless they're off. Until then it
 * can't be saved or confirmed, or the BMI could quietly be the patient's against the trust's
 * observations.precedence. `problem` says which didn't load.
 */
export function inputsState(
  hq: { hq: EpisodeHQ | null; error: string }, obs: { view: ObservationsView | null | "off"; error: string },
): { ready: boolean; problem?: string } {
  if (hq.error) return { ready: false, problem: "The health questionnaire didn't load, so its answers can't be checked. Reload the page to save or confirm." };
  if (obs.view !== "off" && obs.error) {
    return { ready: false, problem: "The observations didn't load, so the hospital's readings can't be checked. Reload the page to save or confirm." };
  }
  return { ready: !!hq.hq && obs.view !== null };
}

/** The items as shown: saved, pre-filled, then the changes on this screen. */
export function shownItems(row: AssessmentRow, from: PrefillFrom, draft: Answers): AssessmentItem[] {
  const saved = row.data ? { data: row.data, sources: row.sources ?? {}, confirmed: !!row.confirmedAt } : undefined;
  return withDraft(row.content, assessmentItems(row.content, from, saved), draft);
}

/**
 * The items once the card is changed: as shown, with the screen's changes, but the age and sex from
 * the patient's record (`from`, as of today), since they can't be changed on the card and a Confirm
 * stores them afresh (S47 answers 7; report 4, F-1). The rest stays as confirmed until changed.
 */
export function changedItems(row: AssessmentRow, from: PrefillFrom, draft: Answers): AssessmentItem[] {
  const record = assessmentItems(row.content, from);
  return shownItems(row, from, draft).map((i) => (i.kind === "profile" ? (record.find((r) => r.key === i.key) ?? i) : i));
}

/** The body of a save or a Confirm. */
export const saveRequest = (row: AssessmentRow, items: AssessmentItem[]) => ({ ...assessmentAnswers(items), revision: row.revision });

export const SOURCE_LABELS: Record<ItemSource, string> = { hq: "HQ", observation: "Observation", profile: "Profile", clinician: "Clinician" };

/** A measured number's name: "BMI" or "weight". */
const measureName = (i: AssessmentItem) => (i.measure === "weight" ? "weight" : "BMI");

/** Where the value came from, in words. */
export function sourceText(i: AssessmentItem): string {
  if (!i.source) return "Missing";
  if (i.kind === "number" && i.source === "clinician") return "Entered by clinician";
  if (i.measure) return i.source === "observation" ? `Hospital ${measureName(i)}` : `Patient reported ${measureName(i)}`;
  return SOURCE_LABELS[i.source];
}

/** The value in words: Yes or No, a BMI, an age, a sex. */
export function valueText(i: AssessmentItem): string {
  if (!i.source) return "—";
  if (i.kind === "choice") return i.options?.find((o) => o.value === i.value)?.text ?? String(i.value);
  if (i.key === "patientSex") return i.value === "male" ? "Male" : "Female";
  return String(i.value);
}

/**
 * A number box's placeholder: "BMI" for a BMI, else the unit its wording ends with ("Weight now
 * (kg)" → "kg"), else nothing (S49 verification 2: not "kg" for every other number).
 */
export const boxPlaceholder = (i: AssessmentItem): string =>
  i.measure === "bmi" ? "BMI" : (/\(([^()]+)\)\s*$/.exec(i.title ?? "")?.[1] ?? "");

/**
 * What's typed in a number item's box: a value (at most one decimal place, a comma allowed, within
 * the item's limits), a mistake to show under the box, or neither while it's empty.
 */
export function numberInput(text: string, limits: { min?: number; max?: number }): { value?: number; error?: string } {
  const t = text.trim();
  if (!t) return {};
  if (!/^\d+([.,]\d)?$/.test(t)) return { error: "A number, one decimal place" };
  const n = Number(t.replace(",", "."));
  // Above 0 unless the item's limits allow 0 (MUST's weight lost, S49).
  const { min, max = Infinity } = limits;
  if ((min === undefined ? n <= 0 : n < min) || n > max) return { error: `Between ${min ?? 0} and ${max}` };
  return { value: n };
}

/**
 * What a number box keeps of what the clinician typed (null: the box shows the item's value, or is
 * empty). A value they typed that the item no longer holds is one the MUST pairing rule has since
 * emptied, so the box goes empty with it (S49 test report 7, F-1); a mistake or an emptied box stays.
 */
export function boxTyped(typed: string | null, i: AssessmentItem): string | null {
  return typed !== null && !i.source && numberInput(typed, i).value !== undefined ? null : typed;
}

/**
 * The items with each box that holds something that isn't a value (a mistake, or emptied) as
 * missing: the count and the score don't go on using the value typed before it.
 */
export const withoutInvalid = (items: AssessmentItem[], invalid: Record<string, boolean>): AssessmentItem[] =>
  items.map((i) => (invalid[i.key] ? { ...i, value: undefined, source: undefined, yes: undefined } : i));

/**
 * "7 of 8 filled", asking for the missing items unless they can't be answered (read-only). The sex
 * can't be answered on the card: where the record says neither male nor female (Other or Not
 * known), the headline says so instead of asking for it (S49a).
 */
export function filledText(items: AssessmentItem[], readOnly: boolean): string {
  const missing = items.filter((i) => !i.source);
  const noSex = missing.some((i) => i.key === "patientSex");
  const ask = missing.filter((i) => i.key !== "patientSex").length;
  const head = `${items.length - missing.length} of ${items.length} filled`;
  const words = !ask || readOnly ? `${head}.` : `${head}: answer the ${ask === 1 ? "item" : `${ask} items`} highlighted.`;
  return noSex ? `${words} The patient's record has no male or female sex, so this score can't be worked out.` : words;
}

/** How a measurement's source reads in a sentence: "hospital BMI", "patient reported weight". */
const sourceName = (i: AssessmentItem) => `${i.source === "observation" ? "hospital" : "patient reported"} ${measureName(i)}`;

/**
 * What a confirmed assessment used that has changed since (S47 answers 6), one sentence each, e.g.
 * "The hospital BMI is now 24.7 (37 when confirmed).". `confirmed` are its items as confirmed;
 * `fresh` a pre-fill now, with the patient's record as of the day of Confirm, so a birthday since
 * isn't a change but a corrected date of birth is. The clinician's own answers aren't compared.
 * Nothing changes until a clinician confirms again.
 */
export function sinceConfirmed(confirmed: AssessmentItem[], fresh: AssessmentItem[]): string[] {
  const out: string[] = [];
  for (const was of confirmed) {
    const now = fresh.find((i) => i.key === was.key);
    if (!now || !was.source || was.source === "clinician" || (now.value === was.value && now.source === was.source)) continue;
    const then = valueText(was);
    if (was.measure) {
      // A BMI left for the clinician beside a hospital weight (the pair rule): the hospital has no height (verification 3).
      const hospitalWeightOnly = was.measure === "bmi" && fresh.some((i) => i.measure === "weight" && i.source === "observation");
      if (!now.source && hospitalWeightOnly) out.push(`The hospital has a weight but no height now (BMI ${then} when confirmed).`);
      else if (!now.source) out.push(`There's no ${measureName(was)} now (${then} when confirmed).`);
      else if (now.source === was.source) out.push(`The ${sourceName(now)} is now ${valueText(now)} (${then} when confirmed).`);
      else out.push(`The ${sourceName(now)} is now ${valueText(now)} (${sourceText(was)} ${then} when confirmed).`);
    } else if (was.kind === "number") {
      // MUST's weight lost: "The HQ now gives 3 kg for Unplanned weight loss… (0 kg when confirmed)." (verification 5, finding 3)
      const unit = boxPlaceholder(was);
      const label = unit ? was.label.replace(/\s*\([^()]+\)\s*$/, "") : was.label;
      const n = (i: AssessmentItem) => (unit ? `${valueText(i)} ${unit}` : valueText(i));
      out.push(now.source ? `The HQ now gives ${n(now)} for ${label} (${n(was)} when confirmed).` : `The HQ no longer answers ${label} (${n(was)} when confirmed).`);
    } else if (was.kind !== "profile") {
      out.push(now.source ? `The HQ now says ${valueText(now)} to ${was.label} (${then} when confirmed).` : `The HQ no longer answers ${was.label} (${then} when confirmed).`);
    } else if (was.key === "patientAge") {
      out.push(now.source ? `The patient's record now gives an age of ${valueText(now)} that day (${then} when confirmed).` : `The patient's record has no date of birth now (${then} when confirmed).`);
    } else {
      out.push(now.source ? `The patient's record now says ${valueText(now)} (${then} when confirmed).` : `The patient's record no longer says male or female (${then} when confirmed).`);
    }
  }
  return out;
}

/**
 * The card's hints (S49): the confirmed results of the assessments this one shows beside it (the
 * Clinical Frailty Scale: DASI's and PRISMA-7's), each as confirmed, e.g. "DASI: 34 (Poor functional
 * capacity), 6.9 METs (confirmed 5 Nov 2026, 10:00)". None until one is confirmed.
 */
export function cardHints(row: AssessmentRow, view: AssessmentsView, e: Episode): string[] {
  return assessmentHints(row.assessment).flatMap((id) => {
    const a = view.assessments.find((x) => x.assessment === id);
    if (!a?.confirmedAt || !a.data) return [];
    return [`${a.name}: ${assessmentResult(a.content, a.data, patientOn(e, a.confirmedAt)).text} (confirmed ${formatDateTime(a.confirmedAt)})`];
  });
}

/** Who confirmed it and when, or that it isn't. */
export const confirmText = (row: AssessmentRow) =>
  row.confirmedAt ? `Confirmed by ${row.updatedByName ?? "someone"} on ${formatDateTime(row.confirmedAt)}` : "Not confirmed";

/**
 * The POA Summary's lines: each assessment's score and band as confirmed (with the patient's age on
 * the day of Confirm), e.g. "5 (High risk)", or "Not confirmed". If they couldn't be loaded, one
 * line says so (as the Hospital BMI's does): a missing line would read as "no assessment done".
 */
export function poaLines(view: AssessmentsView | "off" | null, e: Episode, error: string): { name: string; text: string }[] {
  if (view === "off") return [];
  if (!view) return error ? [{ name: "Assessments", text: "Couldn't be loaded" }] : [];
  return view.assessments.map((a) => ({
    name: a.name,
    text: a.confirmedAt && a.data ? assessmentResult(a.content, a.data, patientOn(e, a.confirmedAt)).text : "Not confirmed",
  }));
}
