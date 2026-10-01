// Translations (LNG-01/02/03/04). Texts stay SurveyJS localisable strings: a plain string is
// English; once translated it becomes { default: "English", de: "Deutsch" }. Clinical notes,
// codes and other clinician metadata are never listed here (CLN-08).
import {
  childrenOf, edit, isContainer, LOCALIZABLE_KEYS, LOCALIZABLE_LISTS, pagesOf, pageTitle, plainText, textOf,
  type ChapterJson, type ElementJson,
} from "./doc";

/** Languages offered first (LNG-01); any other ISO code works too. */
export const LANGUAGES: Record<string, string> = { de: "German", es: "Spanish", fr: "French", pl: "Polish", pt: "Portuguese", it: "Italian", ar: "Arabic", ur: "Urdu" };

const FIELD: Record<string, string> = {
  title: "Text", description: "Description", html: "Statement", placeholder: "Placeholder", requiredErrorText: "Required message",
  otherText: "Other option", selectAllText: "Select all", minRateDescription: "Low label", maxRateDescription: "High label",
  panelAddText: "Add button", panelRemoveText: "Remove button", addRowText: "Add button", choices: "Option", rows: "Row", columns: "Column",
  validators: "Message",
};

/** One translatable text: where it is (a path into the JSON), and its value. */
export type TextEntry = { path: (string | number)[]; where: string; field: string; value: unknown; english: string; isHtml: boolean };

/** Every patient-facing text in a chapter, in page order. Clinician-only content is left out. */
export function textEntries(doc: ChapterJson): TextEntry[] {
  const out: TextEntry[] = [];
  const add = (path: (string | number)[], where: string, field: string, value: unknown, isHtml = false) => {
    const english = textOf(value);
    if (english.trim()) out.push({ path, where, field, value, english: isHtml ? plainText(english) : english, isHtml });
  };
  pagesOf(doc).forEach((page, pi) => {
    const pageName = pageTitle(page, pi);
    add(["pages", pi, "title"], pageName, "Page name", page.title);
    const walk = (els: ElementJson[], base: (string | number)[]) => {
      els.forEach((el, i) => {
        if (el.clinicianOnly) return;
        const path = [...base, i];
        const where = `${pageName} › ${textOf(el.title) || plainText(el.html ?? "").slice(0, 40) || el.name}`;
        for (const key of LOCALIZABLE_KEYS) add([...path, key], where, FIELD[key], el[key], key === "html");
        for (const list of LOCALIZABLE_LISTS) {
          (Array.isArray(el[list]) ? (el[list] as unknown[]) : []).forEach((c, ci) => {
            if (c && typeof c === "object") add([...path, list, ci, "text"], where, FIELD[list], (c as { text?: unknown }).text);
          });
        }
        (Array.isArray(el.validators) ? (el.validators as { text?: unknown }[]) : []).forEach((v, vi) => add([...path, "validators", vi, "text"], where, FIELD.validators, v.text));
        if (isContainer(el)) walk(childrenOf(el), [...path, el.type === "paneldynamic" ? "templateElements" : "elements"]);
      });
    };
    walk(page.elements ?? [], ["pages", pi, "elements"]);
  });
  return out;
}

/** The text in one language, or "" when it isn't translated yet. */
export function translationOf(value: unknown, locale: string): string {
  return value && typeof value === "object" ? String((value as Record<string, string>)[locale] ?? "") : "";
}

/** Languages a chapter has any translation for. */
export function localesIn(doc: ChapterJson): string[] {
  const found = new Set<string>();
  for (const e of textEntries(doc)) {
    if (e.value && typeof e.value === "object") for (const k of Object.keys(e.value)) if (k !== "default") found.add(k);
  }
  return [...found].sort();
}

/** Sets one text in one language. An empty text removes that translation (English is used). */
export function setTranslation(doc: ChapterJson, path: (string | number)[], locale: string, text: string, isHtml = false): ChapterJson {
  return edit(doc, (d) => {
    let parent: Record<string | number, unknown> = d as Record<string, unknown>;
    for (const key of path.slice(0, -1)) parent = parent[key] as Record<string | number, unknown>;
    const key = path[path.length - 1];
    const current = parent[key];
    const value: Record<string, string> =
      current && typeof current === "object" ? { ...(current as Record<string, string>) } : { default: String(current ?? "") };
    const html = isHtml && text.trim() ? text.split(/\n/).map((l) => `<p>${escapeHtml(l) || "&nbsp;"}</p>`).join("") : text;
    if (text.trim()) value[locale] = html;
    else delete value[locale];
    const others = Object.keys(value).filter((k) => k !== "default");
    parent[key] = others.length ? value : value.default;
  });
}

/** Translations as CSV for a translator (LNG-03): path, where, field, English, translation. */
export function toCsv(doc: ChapterJson, locale: string): string {
  const rows = [["id", "where", "field", "english", locale]];
  for (const e of textEntries(doc)) {
    const t = translationOf(e.value, locale);
    rows.push([e.path.join("/"), e.where, e.field, e.english, e.isHtml ? plainText(t) : t]);
  }
  return rows.map((r) => r.map(csvCell).join(",")).join("\r\n");
}

/**
 * Reads a translator's CSV back. Only filled-in cells are applied: a blank cell never removes a
 * translation (someone may have added it after the export). Rows whose English text has changed
 * since the export are skipped.
 */
export function fromCsv(doc: ChapterJson, locale: string, csv: string): { doc: ChapterJson; applied: number; skipped: number } {
  const entries = new Map(textEntries(doc).map((e) => [e.path.join("/"), e]));
  let next = doc;
  let applied = 0;
  let skipped = 0;
  const [header, ...rows] = parseCsv(csv);
  const col = header?.indexOf(locale) ?? -1;
  if (col < 0) return { doc, applied, skipped: rows.length };
  for (const r of rows) {
    const text = (r[col] ?? "").trim();
    if (!text) continue;
    const e = entries.get(r[0]);
    if (!e || e.english !== r[3]) {
      skipped++;
      continue;
    }
    next = setTranslation(next, e.path, locale, text, e.isHtml);
    applied++;
  }
  return { doc: next, applied, skipped };
}

const csvCell = (s: string) => (/[",\r\n]/.test(s) ? `"${s.replace(/"/g, '""')}"` : s);

function parseCsv(text: string): string[][] {
  const rows: string[][] = [];
  let row: string[] = [];
  let cell = "";
  let quoted = false;
  for (let i = 0; i < text.length; i++) {
    const c = text[i];
    if (quoted) {
      if (c === '"' && text[i + 1] === '"') (cell += '"'), i++;
      else if (c === '"') quoted = false;
      else cell += c;
    } else if (c === '"') quoted = true;
    else if (c === ",") row.push(cell), (cell = "");
    else if (c === "\n" || c === "\r") {
      if (c === "\r" && text[i + 1] === "\n") i++;
      row.push(cell), rows.push(row), (row = []), (cell = "");
    } else cell += c;
  }
  if (cell || row.length) row.push(cell), rows.push(row);
  return rows.filter((r) => r.some((x) => x !== ""));
}

const escapeHtml = (s: string) => s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;");

/** How many texts still need translating into a language (LNG-02 flags gaps). */
export function missingCount(doc: ChapterJson, locale: string): number {
  return textEntries(doc).filter((e) => !translationOf(e.value, locale)).length;
}
