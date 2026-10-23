// Pre-filled answers in the patient's HQ (S25; PRE-02, PRE-03). Go fills a Question Set from the
// patient's earlier HQs the first time it's opened and says where each answer came from
// (`prefill`, per set). Here:
//   - the opening card's sentence (cardText), shown until the patient presses one of its two
//     buttons, which Go records;
//   - which pre-filled answers the patient can see (shownPrefilled): only those count, so an
//     answer to a question a condition hides is neither counted nor lost (SurveyJS keeps it, and
//     clears it only if the set is finished with the question still hidden);
//   - where "Everything is still correct" goes (firstUnfilled);
//   - whether the set waits for the card before it can count as complete (awaitsCheck);
//   - the marker under each question that still holds its pre-filled answer (stillPrefilled).
import type { Model, Question } from "survey-core";
import { modelFor, sameAnswer, type Answers, type ChapterJson, type SamplePatient } from "@poised/clinical";
import { formatDay } from "@/lib/format";

/** Where one pre-filled answer came from, as Go stored it (internal/prefill.Entry). */
export type PrefillEntry = {
  fromForm: string;
  /** The earlier questionnaire's send date: the date the patient sees. */
  answeredAt: string;
  /** What was copied. The patient's answer differing from it is a change. */
  value: unknown;
  comment?: unknown;
  matchedBy: "key" | "wording";
};

export type PrefillChoice = "check_each" | "still_correct";

/** One pre-filled Question Set: its answers' sources, and the card's button once pressed. */
export type SetPrefill = { answers: Record<string, PrefillEntry>; choice: PrefillChoice | null; choiceAt: string | null };

/** Whether a question still holds the answer pre-filled into it. */
export function stillPrefilled(entry: PrefillEntry | undefined, value: unknown): boolean {
  return entry !== undefined && sameAnswer(entry.value, value);
}

// Elements that hold no answer, so "Everything is still correct" never stops at one.
const NO_ANSWER = new Set(["html", "expression", "image", "profile"]);

/** The questions the patient can see now, in order, that still hold their pre-filled answer. */
export function shownPrefilled(model: Model, prefill: SetPrefill | undefined): Question[] {
  if (!prefill) return [];
  return (model.getAllQuestions(true) as Question[]).filter((q) => stillPrefilled(prefill.answers[q.name], q.value));
}

/**
 * Where "Everything is still correct" goes: the first question the patient can see that wasn't
 * pre-filled (or no longer holds its pre-filled answer), on any page; null if there's none.
 */
export function firstUnfilled(model: Model, prefill: SetPrefill | undefined): Question | null {
  for (const q of model.getAllQuestions(true) as Question[]) {
    if (NO_ANSWER.has(q.getType()) || q.isReadOnly) continue;
    if (!stillPrefilled(prefill?.answers[q.name], q.value)) return q;
  }
  return null;
}

/**
 * Whether a set waits for the patient to answer its opening card: pre-filled answers show, and
 * neither button has been pressed. Until then it doesn't count as complete.
 */
export function awaitsCheck(content: ChapterJson, data: Answers | undefined, prefill: SetPrefill | undefined, patient: SamplePatient): boolean {
  if (!prefill || prefill.choice) return false;
  const m = modelFor(content, "patient", patient);
  m.data = data ?? {};
  const shown = shownPrefilled(m, prefill).length;
  m.dispose();
  return shown > 0;
}

/**
 * The opening card: "We've filled in 23 answers from your questionnaire on 3 Mar 2026. Please check
 * them." With answers from several dates it names the latest. `whose` is the patient's first name
 * when staff fill it in for them.
 */
export function cardText(entries: PrefillEntry[], formatDay: (iso: string) => string, whose?: string): string {
  const days = new Set(entries.map((e) => formatDay(e.answeredAt)));
  const latest = entries.reduce((a, b) => (new Date(b.answeredAt) > new Date(a.answeredAt) ? b : a), entries[0]);
  const n = entries.length === 1 ? "1 answer" : `${entries.length} answers`;
  const your = whose ? `${whose}'s` : "your";
  const from =
    days.size === 1
      ? `${your} questionnaire on ${formatDay(latest.answeredAt)}`
      : `${your} earlier questionnaires, the latest on ${formatDay(latest.answeredAt)}`;
  const check = entries.length === 1 ? "Please check it" : "Please check them";
  return `We've filled in ${n} from ${from}. ${check}${whose ? ` with ${whose}` : ""}.`;
}

/** The marker under a question that still holds its pre-filled answer. */
export const markerText = (day: string, whose?: string) => `From ${whose ? `${whose}'s` : "your"} answers on ${day}`;

/**
 * Shows the marker under each question that still holds its pre-filled answer, and takes it away
 * when the answer is changed (back again if it's changed back). SurveyJS draws the questions, so the
 * marker is added to each one's element, as the clinician's corrections are (corrections.ts).
 * Returns the clean-up.
 */
export function showPrefillMarkers(model: Model, prefill: SetPrefill, whose?: string): () => void {
  const roots = new Map<string, HTMLElement>();
  const sync = (q: Question) => {
    const root = roots.get(q.name);
    const entry = prefill.answers[q.name];
    if (!root || !entry) return;
    const marker = root.querySelector(":scope > .sj-prefill-marker");
    const show = stillPrefilled(entry, q.value);
    if (show && !marker) {
      const m = document.createElement("div");
      m.className = "sj-prefill-marker";
      m.dataset.testid = "prefill-marker";
      m.textContent = markerText(formatDay(entry.answeredAt), whose);
      root.appendChild(m);
    } else if (!show && marker) {
      marker.remove();
    }
  };
  const rendered = (_: Model, o: { question: Question; htmlElement: HTMLElement }) => {
    roots.set(o.question.name, o.htmlElement);
    sync(o.question);
  };
  const changed = (_: Model, o: { name: string }) => {
    const q = model.getQuestionByName(o.name);
    if (q) sync(q);
  };
  model.onAfterRenderQuestion.add(rendered);
  model.onValueChanged.add(changed);
  return () => {
    model.onAfterRenderQuestion.remove(rendered);
    model.onValueChanged.remove(changed);
  };
}

/**
 * Brings a question to the top of the screen, its text showing, and puts the focus in it; returns
 * the clean-up. SurveyJS's own question.focus() scrolls to the input it focuses, and SurveyJS then
 * settles its layout (its phone styles) just after it first draws, which moved the question's text
 * off the top of a phone's screen (S25 test report 1, F1). So the question is scrolled to itself,
 * the focus doesn't scroll again, and it's kept in place while the page's size settles, for a
 * second at most, or until the patient scrolls or types.
 */
export function focusQuestion(model: Model, name: string): () => void {
  const q = model.getQuestionByName(name);
  const root = q && (document.getElementById(q.id) ?? document.querySelector<HTMLElement>(`[data-name="${CSS.escape(name)}"]`));
  if (!root) {
    q?.focus();
    return () => {};
  }
  const pin = () => root.scrollIntoView({ block: "start" });
  pin();
  root.querySelector<HTMLElement>("input:not([type=hidden]), textarea, select")?.focus({ preventScroll: true });
  const settle = new ResizeObserver(pin);
  settle.observe(document.body);
  const userEvents = ["wheel", "touchstart", "keydown", "mousedown"] as const;
  const stop = () => {
    settle.disconnect();
    clearTimeout(timer);
    for (const e of userEvents) window.removeEventListener(e, stop);
  };
  const timer = setTimeout(stop, 1000);
  for (const e of userEvents) window.addEventListener(e, stop, { passive: true });
  return stop;
}
