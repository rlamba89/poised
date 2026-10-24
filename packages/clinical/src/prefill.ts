// Pre-filled answers (plan C4; S25 for the patient, S26 for the clinician, PRE-04). Go fills a new
// HQ's Question Sets from the patient's earlier HQs and stores, per set, where each answer came
// from (`prefill`). This file holds the shared shape and what the clinician is shown:
//   - prefillMarks: the pre-filled answers among the questions shown, and whether the patient
//     changed each ("Changed by patient", with the pre-filled answer, as corrections show the
//     patient's);
//   - prefillSourceText: "Pre-filled from Demo pre-op HQ, 3 Mar 2026 (matched by link)", with the
//     hospital when it's another trust's;
//   - prefillChoiceText: which button of the opening card the patient pressed;
//   - prefilledAnswers: none once the patient chose Start fresh (S27).
import type { Question, SurveyModel } from "survey-core";
import { isClinicianOnlyInTree } from "./outputs";
import { describeAnswer, sameAnswer, type Answers } from "./respondent";

/** Where one pre-filled answer came from, as Go stored it (internal/prefill.Entry). */
export type PrefillEntry = {
  fromForm: string;
  /** The earlier questionnaire's send date: the date the patient sees. */
  answeredAt: string;
  /** What was copied. The patient's answer differing from it is a change. */
  value: unknown;
  /** "Other" or comment text copied with it. */
  comment?: unknown;
  matchedBy: "key" | "wording";
  /** The earlier questionnaire's name (S26; absent on rows S25 filled). */
  questionnaire?: string;
  /** Another trust's answer (S26), and its hospital. */
  otherTrust?: boolean;
  hospital?: string;
};

/**
 * The opening card's button: Check each answer, Everything is still correct, or (S27, only on a
 * card for another trust's answers) Start fresh, which cleared the set's pre-filled answers.
 */
export type PrefillChoice = "check_each" | "still_correct" | "start_fresh";

/**
 * One pre-filled Question Set: its answers' sources, and the opening card's button once pressed,
 * with the staff member who pressed it for the patient (the clinician's view only).
 */
export type SetPrefill = {
  answers: Record<string, PrefillEntry>;
  choice: PrefillChoice | null;
  choiceAt: string | null;
  choiceByName?: string | null;
};

/** A pre-filled answer the clinician sees: the question, where it came from, and whether the patient changed it. */
export type PrefillMark = {
  questionId: string;
  title: string;
  entry: PrefillEntry;
  /** The patient's final answer differs from the pre-filled one (or its "Other" text does). */
  changed: boolean;
  /** The pre-filled answer, and the patient's final one, in words. */
  was: string;
  now: string;
};

/**
 * The mark for one question, or null if it wasn't pre-filled (or is clinician-only: the patient
 * never had it). `patient` is the patient's final answers to its Question Set.
 */
export function prefillMark(q: Question, entries: Record<string, PrefillEntry> | undefined, patient: Answers): PrefillMark | null {
  const entry = entries?.[q.name];
  if (!entry || isClinicianOnlyInTree(q)) return null;
  const changed =
    !sameAnswer(entry.value, patient[q.name]) || (entry.comment !== undefined && !sameAnswer(entry.comment, patient[`${q.name}-Comment`]));
  return { questionId: q.name, title: q.title, entry, changed, was: describeAnswer(q, entry.value), now: describeAnswer(q, patient[q.name]) };
}

/**
 * The pre-filled answers among the questions the model shows now, in order: a question a condition
 * hides isn't marked, though its answer is kept (S25).
 */
export function prefillMarks(model: SurveyModel, entries: Record<string, PrefillEntry> | undefined, patient: Answers): PrefillMark[] {
  if (!entries) return [];
  return (model.getAllQuestions(true) as Question[]).flatMap((q) => prefillMark(q, entries, patient) ?? []);
}

/** "Pre-filled from Demo pre-op HQ, 3 Mar 2026 (matched by link)"; another trust's names its hospital. */
export function prefillSourceText(e: PrefillEntry, formatDay: (iso: string) => string): string {
  const from = e.questionnaire ?? "an earlier HQ";
  const where = e.otherTrust && e.hospital ? ` at ${e.hospital} (another trust)` : e.otherTrust ? " at another trust" : "";
  const how = e.matchedBy === "key" ? "matched by link" : "matched by same wording";
  return `Pre-filled from ${from}${where}, ${formatDay(e.answeredAt)} (${how})`;
}

/**
 * The set's pre-filled answers that are still pre-filled: none once the patient chose Start fresh
 * (S27), since Go cleared them and every answer there is the patient's own, even one that happens
 * to match what was copied. Go keeps the map as the record of what was copied.
 */
export function prefilledAnswers(p: SetPrefill | undefined): Record<string, PrefillEntry> | undefined {
  return p && p.choice !== "start_fresh" ? p.answers : undefined;
}

/** Which of the opening card's buttons was pressed (S25), for the clinician; "" if none was. */
export function prefillChoiceText(p: SetPrefill, formatDay: (iso: string) => string): string {
  if (!p.choice) return "";
  const who = p.choiceByName ? `${p.choiceByName} chose` : "The patient chose";
  const withThem = p.choiceByName ? " with the patient" : "";
  const on = p.choiceAt ? ` on ${formatDay(p.choiceAt)}` : "";
  if (p.choice === "start_fresh") {
    return `${who} “Start fresh”${withThem}${on}, so the pre-filled answers were cleared and every answer here was given afresh.`;
  }
  return p.choice === "still_correct"
    ? `${who} “Everything is still correct”${withThem}${on}, so these pre-filled answers weren't checked one by one.`
    : `${who} “Check each answer”${withThem}${on}.`;
}
