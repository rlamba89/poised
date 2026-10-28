// Filling in an episode's HQ (plan-workflow.md Steps 3–4): the patient on their link, and the
// clinician validating. The same rules decide which Question Sets are shown and when one is done.
import { settings, type Question, type SurveyModel } from "survey-core";
import { asaRank, suggestAsa, type SuggestedAsa } from "./asa";
import { namesIn, referencesIn, SUGGESTED_ASA, type ChapterJson } from "./doc";
import { computeOutputs, formatAnswer, isChoiceQuestion, type ProducedOutput, type ScoreResult } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { chapterConditionOf, combineChapters, isChapterShown } from "./questionSets";
import { earlierKeysUsed, isEmpty, type Earlier } from "./scores";
import { modelFor, type SamplePatient } from "./testcases";
import type { Viewer } from "./viewer";

/** The episode's patient as the API returns them. Made-up details only (NFR-03). */
export type EpisodePatient = { firstName: string; lastName: string; dateOfBirth: string; sex: string };

export type Answers = Record<string, unknown>;

/**
 * The calendar date at a moment in a time zone (S44a: the hospital's, as Go works it out), or in the
 * browser's own without one (or with a zone the browser doesn't know). Always the Gregorian calendar
 * and Latin digits, whatever the browser's locale (verification 1: th-TH's year is 2569); `locale` is
 * for the tests, the browser's own otherwise.
 */
export function dateIn(at: Date, timeZone?: string, locale?: string): { y: number; m: number; d: number } {
  if (timeZone) {
    try {
      const parts = new Intl.DateTimeFormat(locale, { timeZone, calendar: "gregory", year: "numeric", month: "numeric", day: "numeric", numberingSystem: "latn" }).formatToParts(at);
      const part = (type: string) => Number(parts.find((p) => p.type === type)?.value);
      const out = { y: part("year"), m: part("month"), d: part("day") };
      if (out.y && out.m && out.d) return out;
    } catch {
      // an unknown zone: the browser's own
    }
  }
  return { y: at.getFullYear(), m: at.getMonth() + 1, d: at.getDate() };
}

/** Whole years from a YYYY-MM-DD date of birth to `today` (its date in `timeZone`, if given). */
export function ageFrom(ymd: string, today = new Date(), timeZone?: string): number {
  const [y, m, d] = ymd.split("-").map(Number);
  const now = dateIn(today, timeZone);
  const age = now.y - y;
  return now.m < m || (now.m === m && now.d < d) ? age - 1 : age;
}

/** The survey variables {patientName}, {patientAge} and {patientSex} for this patient (the age at `timeZone`, if given). */
export function patientVariables(p: EpisodePatient, today = new Date(), timeZone?: string): SamplePatient {
  return { name: `${p.firstName} ${p.lastName}`, age: ageFrom(p.dateOfBirth, today, timeZone), sex: p.sex };
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
 * - A set a condition hides gives no answers; its questions are listed as hidden, so a later score
 *   counts them as 0 (S44a, Rahul 27 Oct).
 * - Only the questions shown in a set count (one a condition hides, or on a hidden page or group, is
 *   listed as hidden too), plus every Calculation's value, worked out with the sets before it.
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
  let hidden: string[] = [];
  const produced: ProducedOutput[] = [];
  if (usesAsa) seen = { ...seen, [SUGGESTED_ASA]: 0 };
  for (const set of sets) {
    if (!isChapterShown(set.content, seen)) {
      hidden = [...hidden, ...namesIn(set.content)]; // a hidden set's questions add 0 to a later score (S44a)
      continue;
    }
    shown.push(set);
    const context: Earlier = { doc: combineChapters(before), values, hidden };
    earlier.set(set.id, context);
    const m = modelFor(set.content, viewer, patient, context);
    m.data = answers[set.id] ?? {};
    const next: Answers = { ...values };
    const nextHidden = [...hidden];
    const worked: Answers = {};
    for (const q of m.getAllQuestions(false, false, false)) {
      const calc = q.getType() === "expression";
      const visible = calc || (q.isVisible && q.isParentVisible);
      if (calc) worked[q.name] = q.isEmpty() ? undefined : q.value;
      if (visible && !q.isEmpty()) next[q.name] = q.value;
      if (!visible) nextHidden.push(q.name);
    }
    hidden = nextHidden;
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

/**
 * The answers as the viewer's form shows them (S44a, Rahul 27 Oct: "on send, keep only what the
 * patient's form showed"): each set's answers, without those to questions a condition hides there
 * (the question, its group or its page; their "Other" text too), and {} for a set its condition hides.
 * A Calculation's value stays. Keys the form doesn't know are kept as they are.
 *
 * Repeated until nothing changes (verification 1, finding 2): a follow-up shown only through a hidden
 * question's old answer (B no hides A, whose old Yes kept C shown) goes too, so what's kept is exactly
 * what every later view shows from it, and running it again changes nothing.
 */
export function shownAnswers<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): Record<string, Answers> {
  let now = shownAnswersOnce(sets, answers, viewer, patient);
  // Each pass only drops answers, so this ends; the limit is a guard.
  for (let i = 0; i < 50; i++) {
    const next = shownAnswersOnce(sets, now, viewer, patient);
    if (JSON.stringify(next) === JSON.stringify(now)) break;
    now = next;
  }
  return now;
}

function shownAnswersOnce<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): Record<string, Answers> {
  registerClinicalProperties();
  const { shown, earlier } = hqSets(sets, answers, viewer, patient);
  const out: Record<string, Answers> = {};
  for (const s of sets) out[s.id] = {};
  for (const s of shown) {
    const m = modelFor(s.content, viewer, patient, earlier.get(s.id));
    m.data = answers[s.id] ?? {};
    const hidden = new Set(
      m.getAllQuestions(false, false, false).filter((q) => q.getType() !== "expression" && (!q.isVisible || !q.isParentVisible)).map((q) => q.name),
    );
    m.dispose();
    const suffix = settings.commentSuffix;
    out[s.id] = Object.fromEntries(
      Object.entries(answers[s.id] ?? {}).filter(([k]) => !hidden.has(k) && !(k.endsWith(suffix) && hidden.has(k.slice(0, -suffix.length)))),
    );
  }
  return out;
}

/**
 * The questions the patient's form showed for these answers (S44c, Rahul 28 Oct): for each patient
 * Question Set shown, the keys of its questions a condition doesn't hide (nor their page or group).
 * A set the form hid has none, and a clinician set isn't the patient's. Given the answers the patient
 * sent, it's what their form showed when they sent (a send keeps only those, S44a), so a question it
 * didn't show gets no "Patient answered" line: the clinician answers it.
 */
export function shownToPatient<T extends { id: string; name?: string; content: ChapterJson; audience?: string }>(
  sets: T[], answers: Record<string, Answers | undefined>, patient: SamplePatient,
): Map<string, Set<string>> {
  registerClinicalProperties();
  const mine = sets.filter((s) => !s.audience || s.audience === "patient");
  const { shown, earlier } = hqSets(mine, answers, "patient", patient);
  const out = new Map<string, Set<string>>();
  for (const s of shown) {
    const m = modelFor(s.content, "patient", patient, earlier.get(s.id));
    m.data = answers[s.id] ?? {};
    out.set(s.id, new Set(m.getAllQuestions(false, false, false).filter((q) => q.isVisible && q.isParentVisible).map((q) => q.name)));
    m.dispose();
  }
  return out;
}

/** The questions a form shows now: not hidden by a condition, nor on a hidden page or group. */
function shownQuestionsOf(model: SurveyModel): Set<string> {
  return new Set(model.getAllQuestions(false, false, false).filter((q) => q.getType() !== "expression" && q.isVisible && q.isParentVisible).map((q) => q.name));
}

/**
 * Choices clear hidden answers at once (verification 3, F4): a pick is final, and a chain must never
 * show a follow-up through a hidden answer. Every other change (text, numbers, a BMI's height and
 * weight, a Calculation they feed, lists) waits for the save, after the pause in typing.
 */
const CHOICES = new Set(["radiogroup", "checkbox", "dropdown", "tagbox", "imagepicker", "boolean", "rating", "ranking"]);
const isChoice = (q: { getType(): string } | null | undefined) => !!q && CHOICES.has(q.getType());

/**
 * Loads a set's answers into the patient's form, which from then on clears an answer once a
 * condition hides its question, page or group (S44c, Rahul 28 Oct), so what's on screen is what's
 * saved; each cleared answer is a change (onValueChanged), which the form saves.
 * - Only answers to questions shown since the last check go, repeated while a cleared answer hides
 *   more (a chain). So loading clears nothing (verification 1, F1): an answer hidden when the set
 *   opens (e.g. a pre-filled one) is kept until the set is finished (SurveyJS's default clears
 *   hidden answers on "Finish this section"), and a send drops it anyway (shownAnswers, S44a).
 * - A choice clears at once; any other change only at the save, after the pause in typing (`atSave`;
 *   verifications 2–3, F3/F4): a number typed 20 → 2 → 25, or a weight 120 → 12 → 125, hides a
 *   follow-up only for an instant, and mustn't wipe its answer. A question such a change shows is
 *   remembered as shown, so hiding it again before the save still clears it.
 */
export function loadForPatient(model: SurveyModel, data: Answers): { atSave(): void } {
  model.data = data;
  let shown = shownQuestionsOf(model);
  let clearing = false;
  const clear = () => {
    if (clearing) return;
    clearing = true;
    try {
      for (let i = 0; i < 50; i++) {
        const now = shownQuestionsOf(model);
        const gone = [...shown].filter((n) => !now.has(n)).map((n) => model.getQuestionByName(n)).filter((q) => q && !q.isEmpty());
        shown = now;
        if (!gone.length) break;
        for (const q of gone) q.clearValue();
      }
    } finally {
      clearing = false;
    }
  };
  model.onValueChanged.add((_, o) => {
    if (clearing) return;
    if (isChoice(o.question ?? model.getQuestionByName(o.name))) clear();
    else for (const n of shownQuestionsOf(model)) shown.add(n);
  });
  return { atSave: clear };
}

/**
 * The other Question Sets' answers a change in one set hides (S44c), to clear and save: answers the
 * form showed before the change (`before`, what the server had) that it no longer shows after it
 * (`after`), e.g. a follow-up set whose condition the change made false, and a set or question that
 * set's answers showed. By set id, each set's answers without them; a set losing nothing isn't
 * listed, nor `changed` (its own form clears its answers, loadForPatient). An answer already
 * hidden before (e.g. pre-filled into a hidden question) is left for the send to drop.
 */
export function clearedAnswers<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], before: Record<string, Answers | undefined>, after: Record<string, Answers | undefined>, patient: SamplePatient, changed: string,
): Record<string, Answers> {
  // Building every set's survey isn't free, and the form saves often (each pause in typing): only
  // when another set mentions an answer that changed (its condition, a score) can it lose anything.
  const a = before[changed] ?? {};
  const b = after[changed] ?? {};
  const keys = [...new Set([...Object.keys(a), ...Object.keys(b)])].filter((k) => !sameAnswer(a[k], b[k])).map((k) => k.split(settings.commentSuffix)[0]);
  const others = sets.filter((s) => s.id !== changed).map((s) => JSON.stringify(s.content));
  const mentioned = keys.some((k) => {
    const word = new RegExp(`\\b${k.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")}\\b`);
    return others.some((json) => word.test(json));
  });
  if (!mentioned) return {};
  const was = shownAnswers(sets, before, "patient", patient);
  const now = shownAnswers(sets, after, "patient", patient);
  const out: Record<string, Answers> = {};
  for (const s of sets) {
    if (s.id === changed) continue;
    const gone = Object.keys(was[s.id] ?? {}).filter((k) => !isEmpty(was[s.id][k]) && !(k in (now[s.id] ?? {})));
    if (!gone.length) continue;
    out[s.id] = Object.fromEntries(Object.entries(after[s.id] ?? {}).filter(([k]) => !gone.includes(k)));
  }
  return out;
}

/** Whether a Question Set's condition tests the suggested ASA (only a clinician set's may). */
const testsSuggestedAsa = (doc: ChapterJson) => referencesIn(chapterConditionOf(doc) ?? "").includes(SUGGESTED_ASA);

/**
 * What the whole HQ's answers produce (S39, S40): every shown Question Set's disclosures (each
 * naming its set), its scores, and the suggested ASA, each set built as every view builds it
 * (hqSets: a hidden set gives nothing, a score may use earlier sets' answers).
 */
export function hqResult<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): {
  shown: T[]; outputs: (ProducedOutput & { setId: string })[]; scores: ScoreResult[]; suggestedAsa?: SuggestedAsa;
  /** The same, set by set (S41's evaluate service stores each set's own). */
  sets: { id: string; outputs: ProducedOutput[]; scores: ScoreResult[] }[];
} {
  registerClinicalProperties();
  const { shown, earlier } = hqSets(sets, answers, viewer, patient);
  const outputs: (ProducedOutput & { setId: string })[] = [];
  const scores: ScoreResult[] = [];
  const bySet: { id: string; outputs: ProducedOutput[]; scores: ScoreResult[] }[] = [];
  for (const s of shown) {
    const m = modelFor(s.content, viewer, patient, earlier.get(s.id));
    m.data = answers[s.id] ?? {};
    const r = computeOutputs(m);
    for (const o of r.outputs) outputs.push({ ...o, setId: s.id });
    scores.push(...(r.scores ?? []));
    bySet.push({ id: s.id, outputs: r.outputs, scores: r.scores ?? [] });
    m.dispose();
  }
  const suggestedAsa = suggestAsa(outputs);
  return { shown, outputs, scores, ...(suggestedAsa ? { suggestedAsa } : {}), sets: bySet };
}

/**
 * The suggested ASA for the whole HQ (S39, ASA-01): the highest grade across every shown Question
 * Set's disclosures, with its reasons (hqResult). `viewer` "clinician" with the answers that
 * count gives the validated suggestion; "patient" with the patient's answers (and their sets only)
 * gives the patient's. Each reason names its set (`setId`).
 */
export function hqSuggestedAsa<T extends { id: string; name?: string; content: ChapterJson }>(
  sets: T[], answers: Record<string, Answers | undefined>, viewer: Viewer, patient: SamplePatient,
): SuggestedAsa | undefined {
  return hqResult(sets, answers, viewer, patient).suggestedAsa;
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
