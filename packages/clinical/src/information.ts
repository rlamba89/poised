// The Information element (S35, INF-03). It was the Statement: it is still SurveyJS's `html` type,
// so everything that knows statements (keys, pre-fill, the incumbent import) still knows it.
//
// - The author types simple marks (`**bold**`, `- list`, `[link](https://…)`, `![picture](https://…)`),
//   stored as typed in `markdown` (translatable). `markdownToHtml` turns them into HTML; anything
//   else they type is escaped, so it shows as text.
// - Statements saved before S35 keep their HTML in `html` (one escaped <p> per line, from the editor
//   or Go's incumbent import). They show as before, and open in the editor as their plain text; saving
//   one moves it to `markdown`.
// - `sanitizeHtml` is the last step before anything is shown, whatever made the HTML: it rebuilds it
//   from scratch keeping only a few tags and attributes, with links to https, mailto and tel only and
//   pictures from https only. `informationHtml` is what every survey shows, always through it.
import type { QuestionHtmlModel, SurveyModel } from "survey-core";
import { textOf, type ChapterJson, type ElementJson, type LocString } from "./doc";

// ---------------------------------------------------------------- links and pictures

/** A link may go to a web page (https), an email address or a phone number. */
export function allowedHref(url: string): boolean {
  const u = url.trim();
  return /^(https:\/\/|mailto:|tel:)/i.test(u) && !/[\u0000-\u001F\u007F\s]/.test(u);
}

/** A picture must come from an https address (uploads come with plan C8). */
export function allowedSrc(url: string): boolean {
  const u = url.trim();
  return /^https:\/\/[^\s]+$/i.test(u) && !/[\u0000-\u001F\u007F]/.test(u);
}

const LINK_ATTRS = 'target="_blank" rel="noopener noreferrer"';
// No referrer: the picture's site isn't told which page (or patient) is looking.
const IMG_ATTRS = 'referrerpolicy="no-referrer" loading="lazy"';

// ---------------------------------------------------------------- the author's marks

// An address may hold one level of brackets, e.g. a Wikipedia page "…/Anaesthesia_(UK)".
const URL_PART = "((?:[^()\\s]|\\([^()\\s]*\\))*)";
const INLINE = new RegExp(
  [
    `!\\[([^\\]\\n]*)\\]\\(${URL_PART}\\)`, // ![alt](src)
    `\\[([^\\]\\n]+)\\]\\(${URL_PART}\\)`, // [text](href)
    "\\*\\*(\\S(?:[\\s\\S]*?\\S)?)\\*\\*", // **bold**
    "\\*(\\S(?:[\\s\\S]*?\\S)?)\\*", // *italic*
  ].join("|"),
  "g",
);

/**
 * The author's text with marks → HTML. One paragraph per line; a blank line between lines is a gap
 * (an empty paragraph), as statements had (test report 1); blank lines before the first line and
 * after the last show nothing.
 */
export function markdownToHtml(md: string): string {
  const out: string[] = [];
  let list: { tag: "ul" | "ol"; start: number; items: string[] } | null = null;
  let gaps = 0; // blank lines waiting for a line after them
  const flush = () => {
    // A numbered list starts at its first item's number, so steps split by a gap or a line keep theirs (verification 1, F2).
    const start = list?.tag === "ol" && list.start !== 1 ? ` start="${list.start}"` : "";
    if (list) out.push(`<${list.tag}${start}>${list.items.map((i) => `<li>${i}</li>`).join("")}</${list.tag}>`);
    list = null;
  };
  const block = (html: string) => {
    flush();
    out.push(html);
  };
  const item = (tag: "ul" | "ol", text: string, start = 1) => {
    if (list?.tag !== tag) flush();
    list ??= { tag, start, items: [] };
    list.items.push(inline(text));
  };
  for (const raw of md.replace(/\r\n?/g, "\n").split("\n")) {
    const line = raw.trim();
    if (!line) {
      if (out.length || list) gaps++;
      continue;
    }
    if (gaps) {
      flush();
      for (; gaps > 0; gaps--) out.push("<p>&nbsp;</p>");
    }
    let m: RegExpExecArray | null;
    if ((m = /^(#{1,3})\s+(.+)$/.exec(line))) {
      const h = m[1].length + 1;
      block(`<h${h}>${inline(m[2])}</h${h}>`);
    } else if ((m = /^[-*+]\s+(.+)$/.exec(line))) item("ul", m[1]);
    else if ((m = /^(\d{1,3})[.)]\s+(.+)$/.exec(line))) item("ol", m[2], Number(m[1]));
    else block(`<p>${inline(line)}</p>`);
  }
  flush();
  return out.join("");
}

function inline(text: string): string {
  let out = "";
  let last = 0;
  for (const m of text.matchAll(INLINE)) {
    out += escapeText(text.slice(last, m.index));
    last = m.index + m[0].length;
    const [whole, alt, src, label, href, bold, italic] = m;
    if (src !== undefined) {
      out += allowedSrc(src) ? `<img src="${escapeAttr(src.trim())}" alt="${escapeAttr(alt)}" ${IMG_ATTRS}>` : escapeText(whole);
    } else if (href !== undefined) {
      out += allowedHref(href) ? `<a href="${escapeAttr(href.trim())}" ${LINK_ATTRS}>${inline(label)}</a>` : escapeText(whole);
    } else if (bold !== undefined) out += `<strong>${inline(bold)}</strong>`;
    else out += `<em>${inline(italic)}</em>`;
  }
  return out + escapeText(text.slice(last));
}

/** The links and pictures in the author's text that will show as text (not https, mailto or tel). */
export function refusedLinks(md: string): string[] {
  const out: string[] = [];
  for (const m of md.matchAll(INLINE)) {
    const [, , src, label, href, bold, italic] = m;
    if (src !== undefined && !allowedSrc(src)) out.push(src);
    if (href !== undefined && !allowedHref(href)) out.push(href);
    // Inside bold, italic or an allowed link's words, as inline() reads them.
    else if (href !== undefined) out.push(...refusedLinks(label));
    if (bold !== undefined || italic !== undefined) out.push(...refusedLinks(bold ?? italic));
  }
  return out;
}

// ---------------------------------------------------------------- the sanitiser

const ALLOWED = new Set([
  "p", "br", "b", "strong", "i", "em", "u", "s", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6",
  "a", "img", "div", "span", "blockquote", "hr",
]);
const VOID = new Set(["br", "img", "hr"]);
// Dropped with everything inside them.
const DROP_CONTENT = new Set([
  "script", "style", "textarea", "title", "xmp", "iframe", "noembed", "noframes", "noscript", "plaintext", "template", "object",
]);
const TAG_START = /<(\/?)([a-zA-Z][a-zA-Z0-9-]*)/y;

/**
 * Rebuilds HTML from scratch, keeping only the allowed tags; a `class` of plain words; a link's
 * https, mailto or tel address (opening in a new tab); a picture's https address and its text.
 * Every other tag is dropped (keeping its text, except script, style and the like), every other
 * attribute too, comments go, and all text is escaped again. So nothing can run, whatever went in.
 */
export function sanitizeHtml(html: string): string {
  let out = "";
  const stack: string[] = [];
  const n = html.length;
  let i = 0;
  while (i < n) {
    const lt = html.indexOf("<", i);
    if (lt < 0) {
      out += cleanText(html.slice(i));
      break;
    }
    out += cleanText(html.slice(i, lt));
    i = lt;
    if (html.startsWith("<!--", i)) {
      const end = html.indexOf("-->", i + 4);
      i = end < 0 ? n : end + 3;
      continue;
    }
    if (html[i + 1] === "!" || html[i + 1] === "?") {
      const end = html.indexOf(">", i);
      i = end < 0 ? n : end + 1;
      continue;
    }
    TAG_START.lastIndex = i;
    const m = TAG_START.exec(html);
    if (!m) {
      out += "&lt;";
      i++;
      continue;
    }
    const closing = m[1] === "/";
    const name = m[2].toLowerCase();
    const tag = readAttributes(html, i + m[0].length);
    i = tag.end;
    if (closing) {
      const at = VOID.has(name) ? -1 : stack.lastIndexOf(name);
      if (at >= 0) while (stack.length > at) out += `</${stack.pop()}>`;
      continue;
    }
    if (DROP_CONTENT.has(name)) {
      const close = new RegExp(`</${name}[\\s/>]`, "ig");
      close.lastIndex = i;
      const found = close.exec(html);
      if (!found) break;
      const end = html.indexOf(">", found.index);
      i = end < 0 ? n : end + 1;
      continue;
    }
    if (!ALLOWED.has(name)) continue;
    const opened = openTag(name, tag.attrs);
    if (!opened) continue;
    out += opened;
    if (!VOID.has(name)) stack.push(name);
  }
  while (stack.length) out += `</${stack.pop()}>`;
  return out;
}

function openTag(name: string, attrs: Map<string, string>): string | null {
  const cls = attrs.get("class");
  const classAttr = cls !== undefined && /^[A-Za-z0-9_ -]+$/.test(cls) ? ` class="${cls}"` : "";
  if (name === "a") {
    const href = attrs.get("href");
    // A link to anywhere else is no link: its words stay.
    return href !== undefined && allowedHref(href) ? `<a href="${escapeAttr(href.trim())}"${classAttr} ${LINK_ATTRS}>` : null;
  }
  if (name === "img") {
    const src = attrs.get("src");
    return src !== undefined && allowedSrc(src)
      ? `<img src="${escapeAttr(src.trim())}" alt="${escapeAttr(attrs.get("alt") ?? "")}"${classAttr} ${IMG_ATTRS}>`
      : null;
  }
  // A numbered list may start at another number (digits only).
  const start = attrs.get("start");
  const startAttr = name === "ol" && start !== undefined && /^\d{1,4}$/.test(start) ? ` start="${start}"` : "";
  return `<${name}${classAttr}${startAttr}>`;
}

/** Reads a tag's attributes (the way browsers do, roughly) up to and past its closing `>`. */
function readAttributes(html: string, from: number): { attrs: Map<string, string>; end: number } {
  const attrs = new Map<string, string>();
  const n = html.length;
  let i = from;
  const isSpace = (c: string) => c === " " || c === "\t" || c === "\n" || c === "\r" || c === "\f";
  while (i < n) {
    while (i < n && (isSpace(html[i]) || html[i] === "/")) i++;
    if (i >= n) break;
    if (html[i] === ">") return { attrs, end: i + 1 };
    let name = html[i++]; // the first character may be anything, even "=" or "<"
    while (i < n && !isSpace(html[i]) && html[i] !== "/" && html[i] !== ">" && html[i] !== "=") name += html[i++];
    while (i < n && isSpace(html[i])) i++;
    let value = "";
    if (html[i] === "=") {
      i++;
      while (i < n && isSpace(html[i])) i++;
      const q = html[i];
      if (q === '"' || q === "'") {
        const end = html.indexOf(q, i + 1);
        value = html.slice(i + 1, end < 0 ? n : end);
        i = end < 0 ? n : end + 1;
      } else {
        const start = i;
        while (i < n && !isSpace(html[i]) && html[i] !== ">") i++;
        value = html.slice(start, i);
      }
    }
    name = name.toLowerCase();
    if (!attrs.has(name)) attrs.set(name, decodeEntities(value));
  }
  return { attrs, end: n };
}

// What the sanitiser decodes and escapes again. Any other named character (&eacute;, &pound;) is passed
// on as it is, for the browser to show: a named reference can never make a tag (verification 1, F5).
const BASIC: Record<string, string> = { amp: "&", lt: "<", gt: ">", quot: '"', apos: "'", nbsp: "\u00a0" };
// For an old statement's text in the editor, a few common ones too.
const NAMED: Record<string, string> = {
  ...BASIC, eacute: "é", egrave: "è", agrave: "à", ccedil: "ç", uuml: "ü", ouml: "ö", auml: "ä", szlig: "ß", pound: "£", euro: "€",
  lsquo: "‘", rsquo: "’", ldquo: "“", rdquo: "”", ndash: "–", mdash: "—", hellip: "…", deg: "°", middot: "·", copy: "©", times: "×",
};

/** Text between tags, escaped again: characters and known references are re-escaped, other named ones kept. */
function cleanText(s: string): string {
  return s.replace(/&(?:#[xX][0-9a-fA-F]{1,6};?|#[0-9]{1,7};?|([a-zA-Z][a-zA-Z0-9]{1,31});)|[&<>\u00a0]/g, (whole, named?: string) => {
    if (named !== undefined && !Object.hasOwn(BASIC, named)) return whole;
    return escapeText(whole.length === 1 ? whole : decodeEntities(whole));
  });
}

function decodeEntities(s: string): string {
  return s.replace(/&(?:#[xX]([0-9a-fA-F]{1,6});?|#([0-9]{1,7});?|([a-zA-Z]+);)/g, (whole, hex, dec, named) => {
    if (named) return Object.hasOwn(NAMED, named) ? NAMED[named] : whole;
    const code = hex ? parseInt(hex, 16) : Number(dec);
    return code > 0 && code <= 0x10ffff && (code < 0xd800 || code > 0xdfff) ? String.fromCodePoint(code) : "\ufffd";
  });
}

function escapeText(s: string): string {
  return s.replace(/&/g, "&amp;").replace(/</g, "&lt;").replace(/>/g, "&gt;").replace(/\u00a0/g, "&nbsp;");
}

function escapeAttr(s: string): string {
  return escapeText(s).replace(/"/g, "&quot;");
}

// ---------------------------------------------------------------- the element

/** The text the editor shows: what the author typed, or an old statement's plain text. */
export function informationText(el: ElementJson): string {
  return el.markdown !== undefined ? textOf(el.markdown) : htmlToLines(textOf(el.html));
}

/**
 * An old statement's HTML as the lines an author would type: a line per paragraph, list item or
 * line break, an empty paragraph as a blank line. Cleaned first, so a script's text is never part of
 * it. Hand-made HTML with words outside paragraphs keeps them on their own lines (test report 1).
 */
export function htmlToLines(html: string): string {
  const BLOCK = "(?:p|h[1-6]|li|ul|ol|div|blockquote)";
  return decodeEntities(
    sanitizeHtml(html)
      .replace(/<p>\s*<\/p>/g, "<p>&nbsp;</p>") // an empty paragraph is a blank line too
      // Block tags next to each other start one new line; each line break is one too.
      .replace(new RegExp(`(?:\\s*</?${BLOCK}\\b[^>]*>\\s*)+`, "g"), "\n")
      .replace(/<br>|<hr>/g, "\n")
      .replace(/<[^>]*>/g, ""),
  )
    .split("\n")
    .map((l) => (l.replace(/\u00a0/g, " ").trim()))
    .join("\n")
    .replace(/^\n+|\n+$/g, "");
}

/**
 * The change that saves the author's text. An old statement moves to `markdown`, and its
 * translations become their plain text; a new one's translations are kept by updateElement.
 */
export function informationPatch(el: ElementJson, text: string): Partial<ElementJson> {
  if (el.markdown === undefined && el.html && typeof el.html === "object") {
    const others = Object.entries(el.html as Record<string, string>)
      .filter(([k]) => k !== "default")
      .map(([k, v]) => [k, htmlToLines(String(v))]);
    if (others.length) return { markdown: { ...Object.fromEntries(others), default: text } as LocString, html: undefined };
  }
  return { markdown: text, html: undefined };
}

/** Its first words as plain text, without the marks (for labels and the Translate view). */
export function informationLabel(el: ElementJson): string {
  const html = sanitizeHtml(el.markdown !== undefined ? markdownToHtml(textOf(el.markdown)) : textOf(el.html));
  // A block's end is a space; bold, italic and links leave none.
  const words = html.replace(/<\/?(?:p|h\d|li|ul|ol|div|br|hr|blockquote)\b[^>]*>/g, " ").replace(/<[^>]*>/g, "");
  return decodeEntities(words).replace(/\s+/g, " ").trim();
}

/** How the editor shows a block (its card and the preview): formatted and cleaned, in English. */
export function informationPreview(el: ElementJson): string {
  return sanitizeHtml(el.markdown !== undefined ? markdownToHtml(textOf(el.markdown)) : textOf(el.html));
}

/**
 * What a survey shows for an Information block (or an old statement), always cleaned. The text is
 * turned into HTML and cleaned with its `{placeholders}` still in it; then each placeholder between
 * tags gets its value (an answer, the patient's name) as plain text: escaped, on one line, never a
 * mark, link or picture (verification 1, F1). A placeholder inside a link's address isn't filled.
 */
export function informationHtml(q: QuestionHtmlModel): string {
  const md = q.getLocalizableString("markdown");
  const html = md && !md.isEmpty ? markdownToHtml(md.pureText) : (q.locHtml.pureText ?? "");
  return fillPlaceholders(sanitizeHtml(html), q);
}

function fillPlaceholders(html: string, q: QuestionHtmlModel): string {
  if (!q.survey) return html;
  return html
    .split(/(<[^>]*>)/)
    .map((part) => (part.startsWith("<") ? part : part.replace(/\{[^{}<>]+\}/g, (token) => escapeText(valueOf(q, token)))))
    .join("");
}

function valueOf(q: QuestionHtmlModel, token: string): string {
  let value = q.getProcessedText(token);
  // A composite's inner question (the Profile block) reads the survey's variables from the survey.
  if (value === token) value = (q.survey as unknown as SurveyModel).processText(token, true);
  return value.replace(/\s*[\r\n]+\s*/g, " ");
}

/** Survey-level HTML SurveyJS would draw as it is; the editor never writes it (verification 1, F3). */
export const SURVEY_HTML_KEYS = ["completedHtml", "completedHtmlOnCondition", "completedBeforeHtml", "loadingHtml"] as const;

/** A chapter without survey-level HTML: every survey is built from this. */
export function withoutSurveyHtml<T extends ChapterJson>(json: T): T {
  if (!SURVEY_HTML_KEYS.some((k) => k in json)) return json;
  const out: ChapterJson = { ...json };
  for (const k of SURVEY_HTML_KEYS) delete out[k];
  return out as T;
}
