// The POA Summary's content (plan-workflow.md Step 5), laid out as Lifebox's: one section per
// Question Set shown, and in it one block per page with its disclosure notes and clinical
// comments ("<page> Summary"), plus the Medication, Admissions and BMI answers as tables.
// The same function builds both tabs: the validated summary (clinician answers) and the
// patient's original answers.
import type { Question } from "survey-core";
import { allElements, findElement, pagesOf, pageTitle, type ChapterJson } from "./doc";
import { computeOutputs, isClinicianOnlyInTree } from "./outputs";
import { registerClinicalProperties } from "./properties";
import { describeAnswer, sameAnswer, shownSets, type Answers } from "./respondent";
import { SUMMARY_PREFIX } from "./summary";
import { modelFor, type SamplePatient } from "./testcases";
import type { Viewer } from "./viewer";

export type PoaNote = { text: string; questionId: string; corrected: boolean };
export type CaptureKind = "medication" | "admissions" | "bmi";
export type PoaCapture = { questionId: string; kind: CaptureKind; title: string; rows: Answers[] };
/** An answer the clinician changed: what the patient said, and the final answer. */
export type PoaChange = { questionId: string; title: string; patient: string; clinician: string };
export type PoaPage = { name: string; title: string; notes: PoaNote[]; comments: string; captures: PoaCapture[]; changes: PoaChange[] };
export type PoaSet = { id: string; name: string; pages: PoaPage[] };

const CAPTURES: CaptureKind[] = ["medication", "admissions", "bmi"];

type SetInput = { id: string; name: string; content: ChapterJson };

/**
 * The POA Summary for one set of answers.
 * - `viewer` "clinician" with the clinician's answers gives the validated summary; "patient"
 *   with the patient's answers gives the patient's original answers.
 * - `original`, the patient's answers, marks the clinician's corrections.
 * Pages and Question Sets with nothing to show are left out, as in Lifebox. Unlike Lifebox,
 * a page's notes are shown even when the page has no Clinical summary box, so none are lost.
 */
export function poaSummary(
  sets: SetInput[],
  answers: Record<string, Answers | undefined>,
  viewer: Viewer,
  patient: SamplePatient,
  original?: Record<string, Answers | undefined>,
): PoaSet[] {
  registerClinicalProperties();
  const out: PoaSet[] = [];
  for (const set of shownSets(sets, answers, viewer, patient)) {
    const data = answers[set.id] ?? {};
    const model = modelFor(set.content, viewer, patient);
    model.data = data;
    const before = original?.[set.id];
    const pages = new Map<string, PoaPage>();
    pagesOf(set.content).forEach((p, i) =>
      pages.set(p.name, { name: p.name, title: pageTitle(p, i), notes: [], comments: "", captures: [], changes: [] }),
    );
    const pageOf = (questionId: string) => pages.get(findElement(set.content, questionId)?.page.name ?? "");

    // Corrections first, so notes from a corrected answer can be marked.
    const corrected = new Set<string>();
    if (before) {
      for (const { el } of allElements(set.content)) {
        const q = model.getQuestionByName(el.name);
        if (!q || isClinicianOnlyInTree(q) || sameAnswer(before[el.name], data[el.name])) continue;
        corrected.add(el.name);
        pageOf(el.name)?.changes.push({
          questionId: el.name, title: q.title, patient: describeAnswer(q, before[el.name]), clinician: describeAnswer(q, data[el.name]),
        });
      }
    }

    for (const o of computeOutputs(model).outputs) {
      if (!o.noteText) continue;
      const page = pageOf(o.questionId);
      if (page && !page.notes.some((n) => n.text === o.noteText)) {
        page.notes.push({ text: o.noteText, questionId: o.questionId, corrected: corrected.has(o.questionId) });
      }
    }

    for (const q of model.getAllQuestions(true) as Question[]) {
      const kind = q.getType() as CaptureKind;
      if (!CAPTURES.includes(kind) || q.isEmpty()) continue;
      const rows = (Array.isArray(q.value) ? q.value : [q.value]).filter((r: unknown) => r && typeof r === "object") as Answers[];
      pageOf(q.name)?.captures.push({ questionId: q.name, kind, title: q.title, rows });
    }

    for (const page of pages.values()) {
      const comments = data[`${SUMMARY_PREFIX}comments_${page.name}`];
      if (typeof comments === "string") page.comments = comments.trim();
    }
    const kept = [...pages.values()].filter((p) => p.notes.length || p.comments || p.captures.length || p.changes.length);
    if (kept.length) out.push({ id: set.id, name: set.name, pages: kept });
    model.dispose();
  }
  return out;
}

/** The patient-reported BMI, from the first BMI answer in the summary. */
export function reportedBmi(summary: PoaSet[]): number | undefined {
  for (const s of summary) {
    for (const p of s.pages) {
      const bmi = p.captures.find((c) => c.kind === "bmi")?.rows[0]?.bmi;
      if (typeof bmi === "number") return bmi;
    }
  }
  return undefined;
}
