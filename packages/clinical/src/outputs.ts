// Where outputs live on a question, how they are validated, and how {answer} is rendered.
import { Serializer, type ItemValue, type Question, type SurveyModel } from "survey-core";
import { bandFor, type Band } from "./logic";
import { isClinicianOnly, outputsOf } from "./properties";
import type { AsaGrade, ClinicalOutput } from "./types";

/**
 * Select One, Select Many (and any other choice type) carry outputs on their options. Matrices
 * have a `choices` property too (their columns' default), but carry outputs on the question.
 */
export function isChoiceQuestion(q: Question): boolean {
  return Serializer.isDescendantOf(q.getType(), "selectbase") && Array.isArray((q as unknown as { choices?: unknown }).choices);
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
  // Several dates (a date list): each as a date, joined.
  if (q.getPropertyValue("dateList") && Array.isArray(value)) {
    return value.map((row: { date?: unknown }) => (row?.date ? formatDate(String(row.date)) : "")).filter(Boolean).join(", ");
  }
  if (isChoiceQuestion(q)) {
    const values = Array.isArray(value) ? value : [value];
    return values
      .map((v) => choicesOf(q).find((c) => c.value === v)?.text ?? String(v))
      .join(", ");
  }
  const inputType = (q as unknown as { inputType?: string }).inputType;
  if (inputType === "date" || inputType === "month") return formatDate(String(value));
  return String(value);
}

/** yyyy-MM-dd → dd/MM/yyyy, and yyyy-MM (a month) → MM/yyyy. Anything else is returned unchanged. */
export function formatDate(iso: string): string {
  const m = /^(\d{4})-(\d{2})(?:-(\d{2}))?/.exec(iso);
  if (!m) return iso;
  return m[3] ? `${m[3]}/${m[2]}/${m[1]}` : `${m[2]}/${m[1]}`;
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
 * Questions in each entry of a repeating group count once per entry.
 * With {viewer} = "patient", clinician-only questions never count, even if present.
 * Outputs sit on options, on grid cells (QT-10), on score bands (CAL-02), or on the question.
 */
export function computeOutputs(model: SurveyModel): ComputedOutputs {
  const patient = model.getVariable("viewer") === "patient";
  const outputs: ProducedOutput[] = [];
  for (const q of model.getAllQuestions(true, false, true)) {
    if (q.isEmpty() || (patient && isClinicianOnlyInTree(q))) continue;
    const answer = formatAnswer(q, q.value);
    const base = { questionId: q.name, questionTitle: q.title };
    if (q.getType() === "matrix") {
      const cells = (q.getPropertyValue("cellOutputs") ?? {}) as Record<string, Record<string, ClinicalOutput[]>>;
      const rows = (q as unknown as { rows: ItemValue[] }).rows;
      const columns = (q as unknown as { columns: ItemValue[] }).columns;
      for (const [row, column] of Object.entries(q.value as Record<string, string>)) {
        const label = `${rows.find((r) => r.value === row)?.text ?? row}: ${columns.find((c) => c.value === column)?.text ?? column}`;
        for (const output of cells[row]?.[column] ?? []) {
          outputs.push({ ...base, answerId: `${row}.${column}`, answerLabel: label, output, noteText: note(output, label) });
        }
      }
      continue;
    }
    const band = bands(q);
    if (band) {
      for (const output of band.clinicalOutputs ?? []) {
        outputs.push({ ...base, answerId: band.id, answerLabel: band.label, output, noteText: note(output, answer) });
      }
      continue;
    }
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

/** A Calculation with bands: the band its value is in. */
function bands(q: Question): Band | undefined {
  const list = q.getType() === "expression" ? q.getPropertyValue("bands") : undefined;
  return Array.isArray(list) && list.length ? bandFor(list, q.value) : undefined;
}

/** Whether the question, or a group it sits in, is clinician-only. */
export function isClinicianOnlyInTree(q: Question): boolean {
  // A question in a repeating group's entry has that entry as parent, and the group as parentQuestion.
  for (let el: unknown = q; el; el = (el as { parent?: unknown }).parent ?? (el as { parentQuestion?: unknown }).parentQuestion) {
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
