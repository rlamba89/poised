// Filling in an episode's HQ (plan-workflow.md Steps 3–4): the patient on their link, and the
// clinician validating. The same rules decide which Question Sets are shown and when one is done.
import type { Question } from "survey-core";
import { asaRank, suggestAsa, type SuggestedAsa } from "./asa";
import { referencesIn, SUGGESTED_ASA, type ChapterJson } from "./doc";
import { computeOutputs, formatAnswer, isChoiceQuestion, type ProducedOutput } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { chapterConditionOf, combineChapters, isChapterShown } from "./questionSets";
import { earlierKeysUsed, isEmpty, type Earlier } from "./scores";
import { modelFor, type SamplePatient } from "./testcases";
import type { Viewer } from "./viewer";

/** The episode's patient as the API returns them. Made-up details only (NFR-03). */
export type EpisodePatient = { firstName: string; lastName: string; dateOfBirth: string; sex: string };

export type Answers = Record<string, unknown>;

/** Whole years from a YYYY-MM-DD date of birth to `today`. */
export function ageFrom(ymd: string, today = new Date()): number {
  const [y, m, d] = ymd.split("-").map(Number);
  const age = today.getFullYear() - y;
  return today.getMonth() + 1 < m || (today.getMonth() + 1 === m && today.getDate() < d) ? age - 1 : age;
}

/** The survey variables {patientName}, {patientAge} and {patientSex} for this patient. */
export function patientVariables(p: EpisodePatient, today = new Date()): SamplePatient {
  return { name: `${p.firstName} ${p.lastName}`, age: ageFrom(p.dateOfBirth, today), sex: p.sex };
}

/**
 * Whether every required question the viewer can see is answered, without showing errors. `earlier`
 * (earlierContexts) matters when a condition here tests a score that uses earlier sets' answers.
 */
export function isSetComplete(json: ChapterJson, data: Answers | undefined, viewer: Viewer, patient: SamplePatient, earlier?: Earlier): boolean {
  registerClinicalProperties();
  const m = modelFor(json, viewer, patient, earlier);
  m.data = data ?? {};
  const ok = m.validate(false);
  m.dispose();
  return ok;
}

/**
 * Whether a patient's set counts as complete on their list, so "Send my answers" can go: the
 * answers the server has (`saved`) are complete, and what's on screen (`draft`, if any) has been
 * saved. A change whose save failed keeps the set from counting, so the HQ can't be sent without it
 * (S02a). `sameAnswer` ignores key order.
 */
export function isSetReadyToSend(json: ChapterJson, saved: Answers | undefined, draft: Answers | undefined, patient: SamplePatient, earlier?: Earlier): boolean {
  if (draft !== undefined && !sameAnswer(draft, saved ?? {})) return false;
  return isSetComplete(json, saved, "patient", patient, earlier);
}

/**
 * The Question Sets shown, in order (LOG-02). A set's condition sees the answers of the sets
 * shown before it, and the patient's age and sex. When a score uses another set's answers (S38),
 * an earlier set's Calculations count with the values worked out now, not the ones saved with its
 * answers, which go stale when the patient changes an earlier set (verification 2, F4).
 */
export function shownSets<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): T[] {
  return hqSets(sets, answers, viewer, patient).shown;
}

/**
 * What each shown Question Set's survey gets from the sets shown before it (S38): their answers by
 * key, and their JSON, so a score can use earlier sets' questions. One helper for every view: the
 * patient's form, Validate, the POA Summary (and S41's evaluate service).
 * - A set a condition hides gives nothing, so its questions count as unanswered.
 * - Only the questions shown in a set count (one a condition hides, or on a hidden page or group, is
 *   unanswered), plus every Calculation's value, worked out with the sets before it.
 * - `viewer` decides what is in each set: a patient's form has no clinician-only questions.
 * An HQ none of whose scores use another set's questions gets none (so older HQs work as before).
 */
export function earlierContexts<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): Map<string, Earlier> {
  return hqSets(sets, answers, viewer, patient).earlier;
}

/**
 * Both in one pass: the sets shown, and what each gets from the sets before it. For an HQ whose
 * scores use only their own set, and whose sets' conditions don't test the suggested ASA, the saved
 * answers decide, as before S38, and no survey is built. Otherwise each shown set's survey is built
 * with the sets before it, and the next set's condition sees the saved answers with each
 * Calculation's worked-out value in place of its saved one (an Incomplete score has none), and
 * `{suggestedAsa}` (S39 answers 2): the suggested ASA from the disclosures of the sets shown before
 * it, as a number (III is 3), 0 for none.
 */
export function hqSets<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): { shown: T[]; earlier: Map<string, Earlier> } {
  let seen: Answers = { patientAge: patient.age, patientSex: patient.sex, viewer };
  const earlier = new Map<string, Earlier>();
  const usesAsa = sets.some((s) => testsSuggestedAsa(s.content));
  if (!usesAsa && !sets.some((s) => earlierKeysUsed(s.content).length)) {
    const shown = sets.filter((s) => {
      if (!isChapterShown(s.content, seen)) return false;
      seen = { ...seen, ...answers[s.id] };
      return true;
    });
    return { shown, earlier };
  }
  registerClinicalProperties();
  const shown: T[] = [];
  const before: { name: string; doc: ChapterJson }[] = [];
  let values: Answers = {};
  const produced: ProducedOutput[] = [];
  if (usesAsa) seen = { ...seen, [SUGGESTED_ASA]: 0 };
  for (const set of sets) {
    if (!isChapterShown(set.content, seen)) continue;
    shown.push(set);
    const context: Earlier = { doc: combineChapters(before), values };
    earlier.set(set.id, context);
    const m = modelFor(set.content, viewer, patient, context);
    m.data = answers[set.id] ?? {};
    const next: Answers = { ...values };
    const worked: Answers = {};
    for (const q of m.getAllQuestions(false, false, false)) {
      const calc = q.getType() === "expression";
      if (calc) worked[q.name] = q.isEmpty() ? undefined : q.value;
      if ((calc || (q.isVisible && q.isParentVisible)) && !q.isEmpty()) next[q.name] = q.value;
    }
    if (usesAsa) produced.push(...computeOutputs(m).outputs);
    m.dispose();
    values = next;
    seen = { ...seen, ...answers[set.id], ...worked };
    if (usesAsa) {
      const asa = suggestAsa(produced);
      seen[SUGGESTED_ASA] = asa ? asaRank(asa.grade) : 0;
    }
    before.push({ name: set.name ?? set.id, doc: set.content });
  }
  return { shown, earlier };
}

/** Whether a Question Set's condition tests the suggested ASA (only a clinician set's may). */
const testsSuggestedAsa = (doc: ChapterJson) => referencesIn(chapterConditionOf(doc) ?? "").includes(SUGGESTED_ASA);

/**
 * The suggested ASA for the whole HQ (S39, ASA-01): the highest grade across every shown Question
 * Set's disclosures, with its reasons, each set built as every view builds it (hqSets: a hidden set
 * gives nothing, a score may use earlier sets' answers). `viewer` "clinician" with the answers that
 * count gives the validated suggestion; "patient" with the patient's answers (and their sets only)
 * gives the patient's. Each reason names its set (`setId`).
 */
export function hqSuggestedAsa<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): SuggestedAsa | undefined {
  registerClinicalProperties();
  const { shown, earlier } = hqSets(sets, answers, viewer, patient);
  const outputs: (ProducedOutput & { setId: string })[] = [];
  for (const s of shown) {
    const m = modelFor(s.content, viewer, patient, earlier.get(s.id));
    m.data = answers[s.id] ?? {};
    for (const o of computeOutputs(m).outputs) outputs.push({ ...o, setId: s.id });
    m.dispose();
  }
  return suggestAsa(outputs);
}

/** Key order doesn't matter; a missing answer equals an empty one. */
function canonical(v: unknown): unknown {
  if (isEmpty(v)) return null;
  if (Array.isArray(v)) return v.map(canonical);
  if (typeof v === "object") return Object.fromEntries(Object.keys(v as object).sort().map((k) => [k, canonical((v as Answers)[k])]));
  return v;
}

/** Whether two answers to a question are the same (a clinician correction is a difference). */
export function sameAnswer(a: unknown, b: unknown): boolean {
  return JSON.stringify(canonical(a)) === JSON.stringify(canonical(b));
}

/** An answer in words, for "Patient answered: …". Lists such as medications are one row each. */
export function describeAnswer(q: Question, value: unknown): string {
  if (isEmpty(value)) return "no answer";
  if (!isChoiceQuestion(q) && typeof value === "object") {
    const rows = Array.isArray(value) ? value : [value];
    return rows.map((r) => (typeof r === "object" && r ? Object.values(r).filter((x) => !isEmpty(x)).join(" ") : String(r))).join("; ");
  }
  return formatAnswer(q, value);
}
