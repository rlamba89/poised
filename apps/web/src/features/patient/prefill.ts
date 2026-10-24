// Pre-filled answers in the patient's HQ (S25; PRE-02, PRE-03). Go fills a Question Set from the
// patient's earlier HQs the first time it's opened and says where each answer came from
// (`prefill`, per set). Here:
//   - the opening card's sentence (cardText), shown until the patient presses one of its
//     buttons, which Go records. Another trust's answers (S27: copied without asking) are named by
//     their hospital, and the card adds Start fresh (offersStartFresh), which Go answers by
//     clearing the set; after it, nothing in the set counts as pre-filled (prefilledAnswers);
//   - which pre-filled answers the patient can see (shownPrefilled): only those count, so an
//     answer to a question a condition hides is neither counted nor lost (SurveyJS keeps it, and
//     clears it only if the set is finished with the question still hidden);
//   - where "Everything is still correct" goes (firstUnfilled);
//   - whether the set waits for the card before it can count as complete (awaitsCheck);
//   - the marker under each question that still holds its pre-filled answer (stillPrefilled).
import type { Model, Question } from "survey-core";
import {
  modelFor, prefilledAnswers, sameAnswer, type Answers, type ChapterJson, type PrefillChoice, type PrefillEntry, type SamplePatient,
  type SetPrefill,
} from "@poised/clinical";
import { formatDay } from "@/lib/format";

// The shapes Go sends (SetPrefill, PrefillEntry) are in @poised/clinical, shared with the clinician's view (S26).
export type { PrefillChoice, PrefillEntry, SetPrefill };

/** Whether a question still holds the answer pre-filled into it. */
export function stillPrefilled(entry: PrefillEntry | undefined, value: unknown): boolean {
  return entry !== undefined && sameAnswer(entry.value, value);
}

// Elements that hold no answer, so "Everything is still correct" never stops at one.
const NO_ANSWER = new Set(["html", "expression", "image", "profile"]);

/** The questions the patient can see now, in order, that still hold their pre-filled answer. */
export function shownPrefilled(model: Model, prefill: SetPrefill | undefined): Question[] {
  const entries = prefilledAnswers(prefill);
  if (!entries) return [];
  return (model.getAllQuestions(true) as Question[]).filter((q) => stillPrefilled(entries[q.name], q.value));
}

/**
 * Whether the card offers Start fresh (S27): one of the answers the card counts (`shown`, the
 * entries of shownPrefilled, as cardText gets them) is another trust's, and no button was pressed
 * yet. Another trust's answer on a hidden question doesn't count: that card names no hospital
 * (verification 1, finding 3).
 */
export function offersStartFresh(prefill: SetPrefill | undefined, shown: PrefillEntry[]): boolean {
  return !!prefill && !prefill.choice && shown.some((e) => e.otherTrust);
}

/**
 * Where "Everything is still correct" goes: the first question the patient can see that wasn't
 * pre-filled (or no longer holds its pre-filled answer), on any page; null if there's none.
 */
export function firstUnfilled(model: Model, prefill: SetPrefill | undefined): Question | null {
  const entries = prefilledAnswers(prefill);
  for (const q of model.getAllQuestions(true) as Question[]) {
    if (NO_ANSWER.has(q.getType()) || q.isReadOnly) continue;
    if (!stillPrefilled(entries?.[q.name], q.value)) return q;
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
 * them." With answers from several dates it names the latest; another trust's answers name their
 * hospital ("…from your questionnaire at Hospital B1 on 3 Mar 2026…", S27). `whose` is the
 * patient's first name when staff fill it in for them.
 */
export function cardText(entries: PrefillEntry[], formatDay: (iso: string) => string, whose?: string): string {
  const days = new Set(entries.map((e) => formatDay(e.answeredAt)));
  const latest = entries.reduce((a, b) => (new Date(b.answeredAt) > new Date(a.answeredAt) ? b : a), entries[0]);
  const n = entries.length === 1 ? "1 answer" : `${entries.length} answers`;
  const your = whose ? `${whose}'s` : "your";
  // Another trust's answers name their hospital (S27): "your questionnaire at Hospital B1 on …".
  const hospitals = [...new Set(entries.filter((e) => e.otherTrust).map((e) => e.hospital ?? "another hospital"))];
  // Every answer another trust's: "at Hospital B1" (or "at Hospital B1 and Hospital C1"), whether
  // from one questionnaire or several; "including" only when some are this trust's own
  // (verification 1, finding 2; verification 2, other finding 1).
  const names = hospitals.length <= 1 ? hospitals.join("") : `${hospitals.slice(0, -1).join(", ")} and ${hospitals[hospitals.length - 1]}`;
  const allOthers = hospitals.length > 0 && entries.every((e) => e.otherTrust);
  const oneSource = days.size === 1 && (hospitals.length === 0 || (allOthers && hospitals.length === 1));
  const at = hospitals.length === 1 ? ` at ${hospitals[0]}` : "";
  const including = hospitals.length === 0 ? "" : allOthers ? ` at ${names}` : `, including ${hospitals.length === 1 ? "one" : "ones"} at ${names}`;
  const from = oneSource
    ? `${your} questionnaire${at} on ${formatDay(latest.answeredAt)}`
    : `${your} earlier questionnaires${including}, the latest on ${formatDay(latest.answeredAt)}`;
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
  const entries = prefilledAnswers(prefill) ?? {};
  const sync = (q: Question) => {
    const root = roots.get(q.name);
    const entry = entries[q.name];
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
