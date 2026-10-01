// The Clinical summary box (QT-09): at most one per page, for clinicians only. It lists the
// clinical notes this page's answers produce, above a comments box. Lifebox ClinicalPageSummary.
import type { QuestionHtmlModel, SurveyModel } from "survey-core";
import { computeOutputs } from "./outputs";

type Json = Record<string, unknown>;

export const SUMMARY_PREFIX = "cs_";

/**
 * A copy of the chapter JSON with a summary group at the end of every page whose
 * `clinicalSummary` is on. Only for the clinician view: patients never receive it.
 */
export function withClinicalSummaries<T extends object>(json: T): T {
  const copy = structuredClone(json) as Json;
  for (const page of (copy.pages as Json[] | undefined) ?? []) {
    if (page.clinicalSummary !== true) continue;
    const id = String(page.name);
    const title = typeof page.title === "string" && page.title ? page.title : id;
    page.elements = [
      ...((page.elements as Json[] | undefined) ?? []),
      {
        type: "panel",
        name: `${SUMMARY_PREFIX}${id}`,
        title: `${title} summary`,
        clinicianOnly: true,
        elements: [
          { type: "html", name: `${SUMMARY_PREFIX}notes_${id}`, html: "" },
          {
            type: "comment",
            name: `${SUMMARY_PREFIX}comments_${id}`,
            title: "Clinical comments",
            description: "These will be displayed on the POA summary.",
            rows: 3,
          },
        ],
      },
    ];
  }
  return copy as T;
}

/** The notes this page's answers produce, for its summary box. */
export function pageNotes(model: SurveyModel, pageName: string): string[] {
  const onPage = new Set(model.getPageByName(pageName)?.questions.map((q) => q.name) ?? []);
  return computeOutputs(model)
    .outputs.filter((o) => onPage.has(o.questionId) && o.noteText)
    .map((o) => o.noteText!);
}

/** Refreshes every summary box's list of notes. Call after each answer changes. */
export function refreshClinicalSummaries(model: SurveyModel): void {
  for (const page of model.pages) {
    const notes = model.getQuestionByName(`${SUMMARY_PREFIX}notes_${page.name}`) as QuestionHtmlModel | null;
    if (!notes) continue;
    const list = pageNotes(model, page.name);
    const html = list.length
      ? `<ul class="sj-summary-notes">${list.map((n) => `<li>${escapeHtml(n)}</li>`).join("")}</ul>`
      : `<p class="sj-summary-empty">No disclosures on this page yet.</p>`;
    if (notes.html !== html) notes.html = html;
  }
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}
