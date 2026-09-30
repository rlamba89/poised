// Where outputs live on a question, how they are validated, and how {answer} is rendered.
import type { ItemValue, Question } from "survey-core";
import { outputsOf } from "./properties";
import type { ClinicalOutput } from "./types";

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
