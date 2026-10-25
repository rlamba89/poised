// The Clinical Summary element (SUM-01, S32): a box for clinicians only, placed by the author on a
// page or inside a Group or Section. It lists the clinical notes the answers produce within its scope
// (its parent Group or Section, or the page), above its own comments box. A page can have several;
// each note shows once, under the nearest summary (Rahul, 25 Oct). Incumbent ClinicalPageSummary.
//
// Stored as one small element ({ type: "clinicalsummary", name: "cs_<id>", title?, clinicianOnly }).
// The clinician view expands it into a SurveyJS panel holding the notes (cs_notes_<id>) and the
// comments box (cs_comments_<id>, the answer key). Patients never receive it: it is clinician-only.
//
// Before S32 a page had a "Clinical summary" switch (page.clinicalSummary). That is read as a
// page-level summary at the end of the page named cs_<page>, so its comments keep their key.
import type { QuestionHtmlModel, SurveyModel } from "survey-core";
import {
  childrenOf, elementLabel, isContainer, isRepeating, isSummary, pageTitle, pagesOf, textOf, SUMMARY_TYPE, type ChapterJson, type ElementJson,
  type PageJson,
} from "./doc";
import { computeOutputs } from "./outputs";

export const SUMMARY_PREFIX = "cs_";

/** A summary's id: its name without the cs_ prefix (cs_comments_<id> holds its comments). */
const idOf = (name: string) => (name.startsWith(SUMMARY_PREFIX) ? name.slice(SUMMARY_PREFIX.length) : name);

/**
 * The content with the old page switch read as summary elements (at the end of each such page).
 * Returns the same object when there is nothing to convert. Never changes the original.
 */
export function upgradeSummaries<T extends ChapterJson>(doc: T): T {
  if (!pagesOf(doc).some((p) => "clinicalSummary" in p)) return doc;
  const copy = structuredClone(doc);
  for (const page of pagesOf(copy)) {
    if (!("clinicalSummary" in page)) continue;
    const on = page.clinicalSummary === true;
    delete page.clinicalSummary;
    const name = `${SUMMARY_PREFIX}${page.name}`;
    if (on && !hasElement(page.elements ?? [], name)) {
      page.elements = [...(page.elements ?? []), { type: SUMMARY_TYPE, name, clinicianOnly: true }];
    }
  }
  return copy;
}

function hasElement(els: ElementJson[], name: string): boolean {
  return els.some((el) => el.name === name || (isContainer(el) && hasElement(kids(el), name)));
}

/** A Group's, Section's or repeating group's elements, without adding an empty list (childrenOf does). */
const kids = (el: ElementJson): ElementJson[] => (el.type === "paneldynamic" ? el.templateElements : el.elements) ?? [];

export type SummaryInfo = {
  name: string;
  /** Its own title, or "<Group, Section or page name> summary". */
  title: string;
  page: string;
  /** What it covers, in words: Group “X”, Section “X” or page “X”. */
  scope: string;
  /** The Group or Section it covers; absent for the page. */
  scopeName?: string;
  /** Inside a repeating group, which is not allowed. */
  inList: boolean;
  notesId: string;
  commentsId: string;
};

type Visit = { el: ElementJson; owner?: string };

/**
 * Walks the content (old switch read as a summary): every summary with its scope, and every other
 * element with the summary that lists its notes, the nearest one around it, if any.
 */
function walk(doc: ChapterJson): { summaries: SummaryInfo[]; visits: Visit[] } {
  const summaries: SummaryInfo[] = [];
  const visits: Visit[] = [];
  const visit = (els: ElementJson[], page: PageJson, scope: string, scopeName: string | undefined, label: string, owner: string | undefined, inList: boolean) => {
    const here = els.find(isSummary);
    const mine = here?.name ?? owner;
    for (const el of els) {
      if (isSummary(el)) {
        const id = idOf(el.name);
        summaries.push({
          name: el.name, title: textOf(el.title) || `${label} summary`, page: page.name, scope, scopeName, inList,
          notesId: `${SUMMARY_PREFIX}notes_${id}`, commentsId: `${SUMMARY_PREFIX}comments_${id}`,
        });
        continue;
      }
      visits.push({ el, owner: mine });
      if (isContainer(el)) {
        const name = elementLabel(el);
        const kind = el.patientPage ? "Section" : "Group";
        visit(kids(el), page, `${kind} “${name}”`, el.name, name, mine, inList || isRepeating(el));
      }
    }
  };
  pagesOf(upgradeSummaries(doc)).forEach((page, i) => {
    const title = pageTitle(page, i);
    visit(page.elements ?? [], page, `page “${title}”`, undefined, title, undefined, false);
  });
  return { summaries, visits };
}

/** Every summary, in document order. */
export const summariesIn = (doc: ChapterJson): SummaryInfo[] => walk(doc).summaries;

/** Which summary lists each element's notes: element name → summary name (absent: none does). */
export function summaryOwners(doc: ChapterJson): Map<string, string> {
  const out = new Map<string, string>();
  for (const v of walk(doc).visits) if (v.owner) out.set(v.el.name, v.owner);
  return out;
}

/** What blocks publishing (SGN-04): two summaries covering the same Group, Section or page; one inside a repeating group. */
export function summaryProblems(doc: ChapterJson): string[] {
  const out: string[] = [];
  const seen = new Set<string>();
  for (const s of summariesIn(doc)) {
    const key = `${s.page}/${s.scopeName ?? ""}`;
    if (seen.has(key)) out.push(`Two clinical summaries cover the same ${s.scope}: keep one`);
    seen.add(key);
  }
  for (const s of summariesIn(doc)) {
    if (s.inList) out.push(`The clinical summary “${s.title}” is inside a repeating group: move it out`);
  }
  return [...new Set(out)];
}

/**
 * A copy of the chapter JSON for the clinician view: each summary element becomes a clinician-only
 * panel, where the author placed it, with its notes and its comments box. Patients never receive it.
 */
export function withClinicalSummaries<T extends object>(json: T): T {
  const doc = structuredClone(upgradeSummaries(json as ChapterJson));
  const info = new Map(summariesIn(doc).map((s) => [s.name, s]));
  const expand = (els: ElementJson[]) => {
    els.forEach((el, i) => {
      const s = isSummary(el) ? info.get(el.name) : undefined;
      if (s) els[i] = panelFor(el, s);
      else if (isContainer(el)) expand(childrenOf(el));
    });
  };
  for (const page of pagesOf(doc)) expand(page.elements ?? []);
  return doc as T;
}

function panelFor(el: ElementJson, s: SummaryInfo): ElementJson {
  return {
    type: "panel",
    name: el.name,
    title: el.title ?? s.title,
    clinicianOnly: true,
    ...(el.visibleIf ? { visibleIf: el.visibleIf } : {}),
    elements: [
      { type: "html", name: s.notesId, html: "" },
      { type: "comment", name: s.commentsId, title: "Clinical comments", description: "These will be displayed on the POA summary.", rows: 3 },
    ],
  };
}

/** The notes each summary lists, for the answers in the model: summary name → notes, in order, once each. */
export function summaryNotes(model: SurveyModel, content: ChapterJson): Map<string, string[]> {
  const owners = summaryOwners(content);
  const out = new Map<string, string[]>(summariesIn(content).map((s) => [s.name, []]));
  for (const o of computeOutputs(model).outputs) {
    const list = o.noteText ? out.get(owners.get(o.questionId) ?? "") : undefined;
    if (list && !list.includes(o.noteText!)) list.push(o.noteText!);
  }
  return out;
}

/** Refreshes every summary box's list of notes. Call after each answer changes. `content` is the chapter's JSON. */
export function refreshClinicalSummaries(model: SurveyModel, content: ChapterJson): void {
  const notes = summaryNotes(model, content);
  for (const s of summariesIn(content)) {
    const box = model.getQuestionByName(s.notesId) as QuestionHtmlModel | null;
    if (!box) continue;
    const list = notes.get(s.name) ?? [];
    const where = !s.scopeName ? "on this page" : s.scope.startsWith("Section") ? "in this section" : "in this group";
    const html = list.length
      ? `<ul class="sj-summary-notes">${list.map((n) => `<li>${escapeHtml(n)}</li>`).join("")}</ul>`
      : `<p class="sj-summary-empty">No disclosures ${where} yet.</p>`;
    if (box.html !== html) box.html = html;
  }
}

function escapeHtml(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");
}
