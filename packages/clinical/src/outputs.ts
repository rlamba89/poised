// Where outputs live on a question, how they are validated, and how {answer} is rendered.
import type { ItemValue, Question, SurveyModel } from "survey-core";
import { isClinicianOnly, outputsOf } from "./properties";
import type { AsaGrade, ClinicalOutput } from "./types";

/** Select One, Select Many (and any other choice type) carry outputs on their options. */
export function isChoiceQuestion(q: Question): boolean {
  return Array.isArray((q as unknown as { choices?: unknown }).choices);
}

export function choicesOf(q: Question): ItemValue[] {
  return (q as unknown as { choices?: ItemValue[] }).choices ?? [];
}

/** Statements (html) show text only, so they have no outputs. */
export function canHaveOutputs(q: Question): boolean {
  return q.getType() !== "html";
}

/** How many outputs a question has: its options' for choice questions, else its own. */
export function countOutputs(q: Question): number {
  return isChoiceQuestion(q)
    ? choicesOf(q).reduce((n, item) => n + outputsOf(item).length, 0)
    : outputsOf(q).length;
}

/** CLN-06: at least one code or a note, and a note needs a category. Returns problems in plain words. */
export function validateOutput(o: ClinicalOutput): string[] {
  const problems: string[] = [];
  const hasNote = !!o.note?.text.trim();
  if (!o.codes.length && !hasNote) problems.push("Add at least one code or a clinical note.");
  if (hasNote && !o.note?.category) problems.push("Choose a category for the note.");
  return problems;
}

/**
 * The answer as it appears in a note (CLN-07, slice subset): text as typed, a number
 * as is, a date as dd/MM/yyyy, chosen options as their labels joined with ", ".
 */
export function formatAnswer(q: Question, value: unknown): string {
  if (value === undefined || value === null || value === "") return "";
  if (isChoiceQuestion(q)) {
    const values = Array.isArray(value) ? value : [value];
    return values
      .map((v) => choicesOf(q).find((c) => c.value === v)?.text ?? String(v))
      .join(", ");
  }
  if ((q as unknown as { inputType?: string }).inputType === "date") return formatDate(String(value));
  return String(value);
}

/** yyyy-MM-dd (the date input's value) → dd/MM/yyyy. Anything else is returned unchanged. */
export function formatDate(iso: string): string {
  const m = /^(\d{4})-(\d{2})-(\d{2})/.exec(iso);
  return m ? `${m[3]}/${m[2]}/${m[1]}` : iso;
}

export function renderNote(template: string, answer: string): string {
  return template.replaceAll("{answer}", answer);
}

/** One output produced by the current answers (OUT-04). */
export type ProducedOutput = {
  questionId: string;
  questionTitle: string;
  /** The option that produced it; absent for question-level outputs. */
  answerId?: string;
  answerLabel: string;
  output: ClinicalOutput;
  /** The note with {answer} filled in. */
  noteText?: string;
};

export type ComputedOutputs = {
  outputs: ProducedOutput[];
  /** The highest ASA grade among the outputs (CLN-17). */
  suggestedAsa?: { grade: AsaGrade; emergency: boolean };
};

const ASA_ORDER: AsaGrade[] = ["I", "II", "III", "IV", "V", "VI"];

/**
 * The outputs the current answers produce. Only questions the respondent can see
 * count (LOG-09, CLN-14): `getAllQuestions(true)` skips hidden questions, and
 * questions in hidden groups and pages, whose answers stay in `survey.data`.
 * With {viewer} = "patient", clinician-only questions never count, even if present.
 */
export function computeOutputs(model: SurveyModel): ComputedOutputs {
  const patient = model.getVariable("viewer") === "patient";
  const outputs: ProducedOutput[] = [];
  for (const q of model.getAllQuestions(true)) {
    if (q.isEmpty() || (patient && isClinicianOnlyInTree(q))) continue;
    const answer = formatAnswer(q, q.value);
    const base = { questionId: q.name, questionTitle: q.title };
    if (isChoiceQuestion(q)) {
      const chosen: unknown[] = Array.isArray(q.value) ? q.value : [q.value];
      for (const item of choicesOf(q)) {
        if (!chosen.includes(item.value)) continue;
        for (const output of outputsOf(item)) {
          outputs.push({ ...base, answerId: String(item.value), answerLabel: item.text, output, noteText: note(output, answer) });
        }
      }
    } else {
      for (const output of outputsOf(q)) {
        outputs.push({ ...base, answerLabel: answer, output, noteText: note(output, answer) });
      }
    }
  }
  return { outputs, suggestedAsa: highestAsa(outputs) };
}

function note(o: ClinicalOutput, answer: string): string | undefined {
  return o.note ? renderNote(o.note.text, answer) : undefined;
}

function isClinicianOnlyInTree(q: Question): boolean {
  for (let el: unknown = q; el; el = (el as { parent?: unknown }).parent) {
    if (isClinicianOnly(el as Question)) return true;
  }
  return false;
}

function highestAsa(outputs: ProducedOutput[]): ComputedOutputs["suggestedAsa"] {
  let best: ComputedOutputs["suggestedAsa"];
  for (const { output } of outputs) {
    if (!output.asa) continue;
    const rank = ASA_ORDER.indexOf(output.asa.grade);
    const bestRank = best ? ASA_ORDER.indexOf(best.grade) : -1;
    if (rank > bestRank) best = { ...output.asa };
    else if (rank === bestRank && output.asa.emergency && best) best.emergency = true;
  }
  return best;
}
