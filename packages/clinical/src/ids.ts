// Stable IDs (QST-04, OPT-02). A question's `name` and an option's `value` ARE the IDs:
// short, generated, never typed by authors (e.g. `q_kfbwpx`, `o_tqmd`). The author edits `title` / `text` instead.
import type { ItemValue, PanelModel, Question, SurveyModel } from "survey-core";

// Letters only: the Creator numbers new options from the digits in the last option's
// value, so digit-free IDs keep its placeholder labels as "Item N".
const ALPHABET = "abcdefghijklmnopqrstuvwxyz";

export const QUESTION_ID = /^q_[a-z]{6}$/;
export const GROUP_ID = /^g_[a-z]{6}$/;
export const OPTION_ID = /^o_[a-z]{4}$/;

/** A random id like `q_kfbwpx`, retried until `taken` says it's free. */
export function newId(prefix: string, length: number, taken: (id: string) => boolean = () => false): string {
  for (;;) {
    const bytes = crypto.getRandomValues(new Uint8Array(length));
    const id = `${prefix}_${Array.from(bytes, (b) => ALPHABET[b % ALPHABET.length]).join("")}`;
    if (!taken(id)) return id;
  }
}

const elementNameTaken = (survey: SurveyModel | undefined) => (id: string) =>
  !!survey && (!!survey.getQuestionByName(id) || !!survey.getPanelByName(id));

/**
 * Gives a new or copied question a fresh id. The displayed title falls back to
 * `name`, so an empty title is pinned to the old name first (the default label).
 */
export function assignQuestionId(q: Question): void {
  if (q.locTitle.isEmpty) q.title = q.name;
  q.name = newId("q", 6, elementNameTaken(q.survey as SurveyModel));
}

/** Same as assignQuestionId, for groups (panels). */
export function assignGroupId(p: PanelModel): void {
  if (p.locTitle.isEmpty) p.title = p.name;
  p.name = newId("g", 6, elementNameTaken(p.survey as SurveyModel));
}

/** Gives an option an id unique within its question, keeping its label. */
export function assignOptionId(item: ItemValue, siblings: ItemValue[]): void {
  if (!item.hasText) item.text = String(item.value);
  item.value = newId("o", 4, (id) => siblings.some((s) => s !== item && s.value === id));
}

/** Gives every option of a choice question an id, unless it already has a valid unique one. */
export function assignMissingOptionIds(choices: ItemValue[]): void {
  for (const item of choices) {
    const value = String(item.value);
    const duplicate = choices.some((s) => s !== item && String(s.value) === value);
    if (!OPTION_ID.test(value) || duplicate) assignOptionId(item, choices);
  }
}
