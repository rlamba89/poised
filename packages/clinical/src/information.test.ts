import { describe, expect, it } from "vitest";
import { Model, type QuestionHtmlModel } from "survey-core";
import { addElement, elementLabel, kindOf, KIND_LABEL, newElement, updateElement, type ChapterJson, type ElementJson } from "./doc";
import { modelFor } from "./testcases";
import {
  SURVEY_HTML_KEYS, withoutSurveyHtml,
  informationHtml, informationLabel, informationPatch, informationPreview, informationText, markdownToHtml, refusedLinks, sanitizeHtml,
} from "./information";
import { registerClinicalProperties } from "./properties";
import { setTranslation, textEntries } from "./translations";

registerClinicalProperties();

const A = 'target="_blank" rel="noopener noreferrer"';
const IMG = 'referrerpolicy="no-referrer" loading="lazy"';

describe("markdownToHtml: the author's marks", () => {
  it("makes one paragraph per line, and a blank line a gap, as statements did (test report 1)", () => {
    expect(markdownToHtml("Line one\nLine two\n\nLine three")).toBe("<p>Line one</p><p>Line two</p><p>&nbsp;</p><p>Line three</p>");
    expect(markdownToHtml("a\n\n\nb")).toBe("<p>a</p><p>&nbsp;</p><p>&nbsp;</p><p>b</p>");
    // Blank lines before the first line and after the last show nothing.
    expect(markdownToHtml("\n \nLine\n\n")).toBe("<p>Line</p>");
    expect(markdownToHtml("")).toBe("");
    expect(markdownToHtml("  \n \n")).toBe("");
  });

  it("shows an edited old statement as it was, gaps included", () => {
    const old = "<p>Line A</p><p>&nbsp;</p><p>Line B &lt;tag&gt; &amp; more</p>";
    const text = informationText({ type: "html", name: "o", html: old });
    expect(text).toBe("Line A\n\nLine B <tag> & more");
    expect(markdownToHtml(text)).toBe(old);
  });

  it("makes headings from #, ## and ###", () => {
    expect(markdownToHtml("# Before you come in\n## Fasting\n### Water")).toBe("<h2>Before you come in</h2><h3>Fasting</h3><h4>Water</h4>");
    // No space after #, or more than three: an ordinary line.
    expect(markdownToHtml("#hashtag\n#### four")).toBe("<p>#hashtag</p><p>#### four</p>");
  });

  it("makes bold and italic", () => {
    expect(markdownToHtml("Do **not** eat *after midnight*")).toBe("<p>Do <strong>not</strong> eat <em>after midnight</em></p>");
    // A lone star, or stars around spaces, stay as they are.
    expect(markdownToHtml("2 * 3 * 4 and a * star")).toBe("<p>2 * 3 * 4 and a * star</p>");
    expect(markdownToHtml("**bold with *italic* inside**")).toBe("<p><strong>bold with <em>italic</em> inside</strong></p>");
  });

  it("makes bulleted and numbered lists from consecutive lines", () => {
    expect(markdownToHtml("Bring:\n- your medicines\n* your glasses\nThanks")).toBe(
      "<p>Bring:</p><ul><li>your medicines</li><li>your glasses</li></ul><p>Thanks</p>",
    );
    expect(markdownToHtml("1. Shower\n2) Dress\n- one")).toBe("<ol><li>Shower</li><li>Dress</li></ol><ul><li>one</li></ul>");
    expect(markdownToHtml("- a\n\n- b")).toBe("<ul><li>a</li></ul><p>&nbsp;</p><ul><li>b</li></ul>");
    expect(markdownToHtml("- **Bold** item")).toBe("<ul><li><strong>Bold</strong> item</li></ul>");
  });

  it("keeps the author's step numbers (verification 1, F2)", () => {
    expect(markdownToHtml("1. Stop eating\n\n2. Take your tablets\n\n3. Arrive at 7am")).toBe(
      '<ol><li>Stop eating</li></ol><p>&nbsp;</p><ol start="2"><li>Take your tablets</li></ol><p>&nbsp;</p><ol start="3"><li>Arrive at 7am</li></ol>',
    );
    expect(markdownToHtml("1. One\nA note\n2. Two")).toBe('<ol><li>One</li></ol><p>A note</p><ol start="2"><li>Two</li></ol>');
    expect(markdownToHtml("3. Three\n4. Four")).toBe('<ol start="3"><li>Three</li><li>Four</li></ol>');
    expect(markdownToHtml("1. a\n1. b")).toBe("<ol><li>a</li><li>b</li></ol>");
    expect(sanitizeHtml('<ol start="2"><li>x</li></ol>')).toBe('<ol start="2"><li>x</li></ol>');
    expect(sanitizeHtml('<ol start="2x"><li>x</li></ol><ul start="3"></ul>')).toBe("<ol><li>x</li></ol><ul></ul>");
  });

  it("makes links to https, mailto and tel only, opening in a new tab", () => {
    expect(markdownToHtml("See [the map](https://example.org/map?a=1&b=2)")).toBe(
      `<p>See <a href="https://example.org/map?a=1&amp;b=2" ${A}>the map</a></p>`,
    );
    expect(markdownToHtml("[Email us](mailto:poa@example.org) or [call](tel:+441234567890)")).toBe(
      `<p><a href="mailto:poa@example.org" ${A}>Email us</a> or <a href="tel:+441234567890" ${A}>call</a></p>`,
    );
    expect(markdownToHtml("[**Bold** link](https://example.org)")).toBe(`<p><a href="https://example.org" ${A}><strong>Bold</strong> link</a></p>`);
  });

  it("shows any other link as the text typed", () => {
    expect(markdownToHtml("[x](javascript:alert(1))")).toBe("<p>[x](javascript:alert(1))</p>");
    expect(markdownToHtml("[x](http://example.org)")).toBe("<p>[x](http://example.org)</p>");
    expect(markdownToHtml("[x](JavaScript:alert(1))")).toBe("<p>[x](JavaScript:alert(1))</p>");
    expect(markdownToHtml("[x](data:text/html,hi)")).toBe("<p>[x](data:text/html,hi)</p>");
    expect(markdownToHtml("[x](/relative)")).toBe("<p>[x](/relative)</p>");
  });

  it("makes pictures from https links only", () => {
    expect(markdownToHtml("![The car park](https://example.org/park.png)")).toBe(
      `<p><img src="https://example.org/park.png" alt="The car park" ${IMG}></p>`,
    );
    expect(markdownToHtml("![x](http://example.org/a.png)")).toBe("<p>![x](http://example.org/a.png)</p>");
    expect(markdownToHtml("![x](javascript:alert(1))")).toBe("<p>![x](javascript:alert(1))</p>");
    expect(markdownToHtml('![a" onerror="alert(1)](https://example.org/a.png)')).toBe(
      `<p><img src="https://example.org/a.png" alt="a&quot; onerror=&quot;alert(1)" ${IMG}></p>`,
    );
  });

  it("escapes everything else, so typed HTML shows as text", () => {
    expect(markdownToHtml("<script>alert(1)</script>")).toBe("<p>&lt;script&gt;alert(1)&lt;/script&gt;</p>");
    expect(markdownToHtml("<img src=x onerror=alert(1)>")).toBe("<p>&lt;img src=x onerror=alert(1)&gt;</p>");
    expect(markdownToHtml("A & B < C")).toBe("<p>A &amp; B &lt; C</p>");
    expect(markdownToHtml('[a](https://x.org/"onmouseover="alert(1))')).toBe(
      `<p><a href="https://x.org/&quot;onmouseover=&quot;alert(1)" ${A}>a</a></p>`,
    );
  });
});

describe("refusedLinks: what the editor warns about", () => {
  it("lists the links and pictures shown as text", () => {
    expect(refusedLinks("[a](https://ok.org) [b](javascript:alert(1)) ![c](http://x.org/c.png) ![d](https://ok.org/d.png)")).toEqual([
      "javascript:alert(1)",
      "http://x.org/c.png",
    ]);
    expect(refusedLinks("nothing here")).toEqual([]);
    // Inside bold, italic or a link's words too.
    expect(refusedLinks("**[Form](http://a.org)** *![p](http://b.org/p.png)* [x](https://ok.org) **[y](javascript:z)**")).toEqual([
      "http://a.org", "http://b.org/p.png", "javascript:z",
    ]);
  });
});

describe("sanitizeHtml: the last step before anything is shown", () => {
  it("keeps the allowed tags", () => {
    const ok = "<h2>Hi</h2><p>A <strong>b</strong> <em>c</em> <b>d</b> <i>e</i><br></p><ul><li>x</li></ul><ol><li>y</li></ol>";
    expect(sanitizeHtml(ok)).toBe(ok);
  });

  it("keeps old statements and the built-in blocks exactly", () => {
    expect(sanitizeHtml("<p>Review the patient's allergies.</p><p>&nbsp;</p>")).toBe("<p>Review the patient's allergies.</p><p>&nbsp;</p>");
    // As Go's incumbent import writes them (hqimport textToHTML): escaped, &nbsp; for a blank line.
    const imported = "<p>Review the patient's allergies &amp; drugs &lt;3 &gt; 1</p><p>&nbsp;</p><p>Two</p>";
    expect(sanitizeHtml(imported)).toBe(imported);
    const profile = "<div class=\"sj-profile\"><p><b>Name</b> Sam</p></div><p class=\"sj-profile-note\">If any</p>";
    expect(sanitizeHtml(profile)).toBe(profile);
    expect(sanitizeHtml("<ul class='sj-summary-notes'><li>Asthma</li></ul>")).toBe('<ul class="sj-summary-notes"><li>Asthma</li></ul>');
  });

  it("drops script and style with their content", () => {
    expect(sanitizeHtml("<p>a</p><script>alert(1)</script><p>b</p>")).toBe("<p>a</p><p>b</p>");
    expect(sanitizeHtml("<SCRIPT src=//evil.org/x.js></SCRIPT>ok")).toBe("ok");
    expect(sanitizeHtml("<style>body{display:none}</style>ok")).toBe("ok");
    expect(sanitizeHtml("<iframe src=https://evil.org></iframe>ok")).toBe("ok");
    expect(sanitizeHtml("<script>never closed")).toBe("");
  });

  it("drops event attributes, styles and anything not allowed", () => {
    expect(sanitizeHtml('<img src="x" onerror="alert(1)">')).toBe("");
    expect(sanitizeHtml('<img src=https://ok.org/a.png onerror=alert(1)>')).toBe(`<img src="https://ok.org/a.png" alt="" ${IMG}>`);
    expect(sanitizeHtml('<p onclick="alert(1)" style="color:red" id="x">a</p>')).toBe("<p>a</p>");
    expect(sanitizeHtml('<p class="x" onmouseover=alert(1)>a</p>')).toBe('<p class="x">a</p>');
    expect(sanitizeHtml('<p class="a&quot;onclick=alert(1)">a</p>')).toBe("<p>a</p>");
    expect(sanitizeHtml("<svg onload=alert(1)><circle></circle></svg>ok")).toBe("ok");
    expect(sanitizeHtml("<form action=https://evil.org><input name=a><button>Go</button></form>")).toBe("Go");
  });

  it("allows only https, mailto and tel links, and https pictures", () => {
    expect(sanitizeHtml('<a href="javascript:alert(1)">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="  JaVaScRiPt:alert(1)">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="&#106;avascript:alert(1)">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="java\tscript:alert(1)">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="data:text/html;base64,PHNjcmlwdD4=">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="http://example.org">x</a>')).toBe("x");
    expect(sanitizeHtml('<a href="https://example.org/?a=1&amp;b=2" target="_self">x</a>')).toBe(`<a href="https://example.org/?a=1&amp;b=2" ${A}>x</a>`);
    expect(sanitizeHtml('<a href="mailto:a@b.org">x</a><a href="tel:123">y</a>')).toBe(`<a href="mailto:a@b.org" ${A}>x</a><a href="tel:123" ${A}>y</a>`);
    expect(sanitizeHtml('<img src="http://example.org/a.png">')).toBe("");
    expect(sanitizeHtml('<img src="data:image/png;base64,AAAA">')).toBe("");
  });

  it("copes with broken HTML", () => {
    expect(sanitizeHtml("<p><strong>unclosed")).toBe("<p><strong>unclosed</strong></p>");
    expect(sanitizeHtml("</p>stray</li>")).toBe("stray");
    expect(sanitizeHtml("<p>a<ul><li>b</p></ul>")).toBe("<p>a<ul><li>b</li></ul></p>");
    expect(sanitizeHtml("a < b > c")).toBe("a &lt; b &gt; c");
    expect(sanitizeHtml("<!-- <script>alert(1)</script> -->ok")).toBe("ok");
    expect(sanitizeHtml("<p title='<script>'>x</p>")).toBe("<p>x</p>");
    expect(sanitizeHtml("<img src=https://ok.org/a.png alt='x\" onerror=alert(1)'>")).toBe(
      `<img src="https://ok.org/a.png" alt="x&quot; onerror=alert(1)" ${IMG}>`,
    );
    expect(sanitizeHtml("<a href='https://ok.org'<script>x</a>")).toBe(`<a href="https://ok.org" ${A}>x</a>`);
    expect(sanitizeHtml("<<script>script>alert(1)<</script>/script>")).toBe("&lt;/script&gt;");
    expect(sanitizeHtml("&lt;script&gt;")).toBe("&lt;script&gt;");
    expect(sanitizeHtml("&#x3C;b&#62;")).toBe("&lt;b&gt;");
    // Other named characters are passed on for the browser to show (verification 1, F5); they can't form markup.
    expect(sanitizeHtml("<p>caf&eacute; &pound;5 it&rsquo;s &unknown;</p>")).toBe("<p>caf&eacute; &pound;5 it&rsquo;s &unknown;</p>");
    expect(sanitizeHtml("&amp;eacute; & ;")).toBe("&amp;eacute; &amp; ;");
  });

  it("is stable: cleaning twice changes nothing", () => {
    const html = sanitizeHtml(markdownToHtml("# T\n- **a** [l](https://x.org)\n![p](https://x.org/p.png)\n<b>"));
    expect(sanitizeHtml(html)).toBe(html);
  });
});

describe("the Information kind (Statement becomes Information)", () => {
  it("is called Information, and a new one holds Markdown", () => {
    expect(KIND_LABEL.information).toBe("Information");
    const el = newElement("information", new Set());
    expect(el).toMatchObject({ type: "html" });
    expect(typeof el.markdown).toBe("string");
    expect(el.html).toBeUndefined();
    expect(kindOf(el)).toBe("information");
    expect(kindOf({ type: "html", name: "old", html: "<p>Old</p>" })).toBe("information");
  });

  it("opens an old statement as its plain text, and a new one as typed", () => {
    expect(informationText({ type: "html", name: "a", html: "<p>Line one</p><p>A &lt; B</p>" })).toBe("Line one\nA < B");
    expect(informationText({ type: "html", name: "a", html: { default: "<p>English</p>", de: "<p>Deutsch</p>" } })).toBe("English");
    expect(informationText({ type: "html", name: "a", markdown: "**Hi**", html: "<p>stale</p>" })).toBe("**Hi**");
    expect(informationText({ type: "html", name: "a", markdown: { default: "**Hi**", de: "**Hallo**" } })).toBe("**Hi**");
  });

  it("saving an old statement moves it to Markdown, keeping its translations as their text", () => {
    const old: ElementJson = { type: "html", name: "a", html: { default: "<p>English</p>", de: "<p>Deutsch</p><p>Zwei</p>" } };
    expect(informationPatch(old, "English **now**")).toEqual({ markdown: { default: "English **now**", de: "Deutsch\nZwei" }, html: undefined });
    expect(informationPatch({ type: "html", name: "a", html: "<p>Old</p>" }, "New")).toEqual({ markdown: "New", html: undefined });
    // Each translation keeps its lines (verification 1, F4).
    const listed: ElementJson = { type: "html", name: "a", html: { default: "<p>E</p>", de: "<h2>Titel</h2><ul><li>eins</li><li>zwei</li></ul>" } };
    expect(informationPatch(listed, "E")).toEqual({ markdown: { default: "E", de: "Titel\neins\nzwei" }, html: undefined });
  });

  it("saving a new one keeps its translations (through updateElement)", () => {
    let doc: ChapterJson = { pages: [{ name: "p1", elements: [{ type: "html", name: "i1", markdown: { default: "Hi", fr: "Salut" } }] }] };
    const el = doc.pages![0].elements![0];
    doc = updateElement(doc, "i1", informationPatch(el, "Hello"));
    expect(doc.pages![0].elements![0]).toEqual({ type: "html", name: "i1", markdown: { default: "Hello", fr: "Salut" } });
  });

  it("is named by its first words, without the marks", () => {
    const el: ElementJson = { type: "html", name: "i1", markdown: "# Fasting\n- **No food** after [midnight](https://x.org)" };
    expect(informationLabel(el)).toBe("Fasting No food after midnight");
    expect(elementLabel(el)).toBe("Fasting No food after midnight");
    expect(elementLabel({ type: "html", name: "o", html: "<p>Old one</p>" })).toBe("Old one");
    // Inline marks leave no gap; a script's text is never part of it.
    expect(informationLabel({ type: "html", name: "i", markdown: "Park in **car park B**." })).toBe("Park in car park B.");
    expect(informationLabel({ type: "html", name: "o", html: "<p>Hi</p><script>window.x=1</script>there" })).toBe("Hi there");
    expect(informationText({ type: "html", name: "o", html: "<p>Hi</p><script>window.x=1</script><p>there</p>" })).toBe("Hi\nthere");
    // HTML that isn't one <p> per line keeps its lines apart (test report 1, exploring 1).
    expect(informationText({ type: "html", name: "o", html: "<p>Imported text</p>bad link<p onclick=x>Click me</p>" })).toBe("Imported text\nbad link\nClick me");
    expect(informationText({ type: "html", name: "o", html: "<div>a<br>b</div><ul><li>c</li><li>d</li></ul>" })).toBe("a\nb\nc\nd");
    expect(informationText({ type: "html", name: "o", html: "<p>a</p><p></p><p>b</p>" })).toBe("a\n\nb");
    expect(informationText({ type: "html", name: "o", html: "<p>caf&eacute; &pound;5</p>" })).toBe("café £5");
  });

  it("can be translated, as plain text with its marks", () => {
    let doc: ChapterJson = { pages: [{ name: "p1", elements: [{ type: "html", name: "i1", markdown: "**Bring** your medicines" }] }] };
    const entry = textEntries(doc).find((e) => e.path.at(-1) === "markdown")!;
    expect(entry).toMatchObject({ field: "Information", english: "**Bring** your medicines", isHtml: false, where: "Page 1 › Bring your medicines" });
    doc = setTranslation(doc, entry.path, "de", "**Bringen** Sie Ihre Medikamente", entry.isHtml);
    expect(doc.pages![0].elements![0].markdown).toEqual({ default: "**Bring** your medicines", de: "**Bringen** Sie Ihre Medikamente" });
    // An old statement is still listed as before.
    const old: ChapterJson = { pages: [{ name: "p1", elements: [{ type: "html", name: "o", html: "<p>Old</p>" }] }] };
    expect(textEntries(old)[0]).toMatchObject({ field: "Information", english: "Old", isHtml: true });
  });

  it("shows in the editor formatted and cleaned, an old statement as its HTML", () => {
    expect(informationPreview({ type: "html", name: "i", markdown: "**Hi** <b>x</b>" })).toBe("<p><strong>Hi</strong> &lt;b&gt;x&lt;/b&gt;</p>");
    expect(informationPreview({ type: "html", name: "o", html: { default: "<p>Old</p><script>x</script>" } })).toBe("<p>Old</p>");
  });

  it("can be added to a page", () => {
    const { doc } = addElement({ pages: [{ name: "p1", elements: [] }] }, { page: "p1" }, "information");
    expect(kindOf(doc.pages![0].elements![0])).toBe("information");
  });
});

describe("informationHtml: what a survey shows", () => {
  const model = (el: ElementJson, data?: Record<string, unknown>) => {
    const m = new Model({ pages: [{ name: "p1", elements: [el, { type: "text", name: "q1" }] }] });
    if (data) m.data = data;
    return { m, q: m.getQuestionByName(el.name) as QuestionHtmlModel };
  };

  it("shows Markdown formatted", () => {
    const { q } = model({ type: "html", name: "i1", markdown: "# Fasting\n- No **food**" });
    expect(informationHtml(q)).toBe("<h2>Fasting</h2><ul><li>No <strong>food</strong></li></ul>");
  });

  it("shows an old statement as before, cleaned", () => {
    expect(informationHtml(model({ type: "html", name: "o", html: "<p>Old</p><p>Two</p>" }).q)).toBe("<p>Old</p><p>Two</p>");
    expect(informationHtml(model({ type: "html", name: "o", html: "<p>Hi</p><script>alert(1)</script><img src=x onerror=alert(1)>" }).q)).toBe("<p>Hi</p>");
  });

  it("shows the survey's language, or English when not translated", () => {
    const { m, q } = model({ type: "html", name: "i1", markdown: { default: "**Hello**", de: "**Hallo**" } });
    m.locale = "de";
    expect(informationHtml(q)).toBe("<p><strong>Hallo</strong></p>");
    m.locale = "fr";
    expect(informationHtml(q)).toBe("<p><strong>Hello</strong></p>");
    m.locale = "";
  });

  it("never lets an answer put HTML on the page", () => {
    const bad = "<img src=x onerror=alert(1)>";
    expect(informationHtml(model({ type: "html", name: "i1", markdown: "You said {q1}" }, { q1: bad }).q)).toBe(
      "<p>You said &lt;img src=x onerror=alert(1)&gt;</p>",
    );
    expect(informationHtml(model({ type: "html", name: "o", html: "<p>You said {q1}</p>" }, { q1: bad }).q)).toBe(
      "<p>You said &lt;img src=x onerror=alert(1)&gt;</p>",
    );
  });

  // Verification 1, F1: an answer shows as plain text: no link, picture, heading, mark or new line.
  const answers = [
    ["a link and a picture", "[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)",
      "[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)"],
    ["a heading and bold", "# Big **bold**", "# Big **bold**"],
    ["several lines", "one\n- two\n\n<b>three</b>", "one - two &lt;b&gt;three&lt;/b&gt;"],
    ["an HTML link", '<a href="https://evil.example">x</a>', '&lt;a href="https://evil.example"&gt;x&lt;/a&gt;'],
  ];
  it.each(answers)("shows %s in an answer as plain text, in Information", (_, value, shown) => {
    const html = informationHtml(model({ type: "html", name: "i1", markdown: "You said: **{q1}**" }, { q1: value }).q);
    expect(html).toBe(`<p>You said: <strong>${shown}</strong></p>`);
    expect(html).not.toMatch(/<a |<img|<h\d|<li/);
  });
  it.each(answers)("shows %s in an answer as plain text, in an old statement", (_, value) => {
    const html = informationHtml(model({ type: "html", name: "o", html: "<p>Old: {q1}</p>" }, { q1: value }).q);
    expect(html).not.toMatch(/<a |<img|<h\d|<li|<b>/);
    expect(html.startsWith("<p>Old: ")).toBe(true);
  });

  it("never fills an answer into a link's address", () => {
    const html = informationHtml(model({ type: "html", name: "i1", markdown: "[Map](https://example.org/{q1})" }, { q1: "x\" onclick=\"y" }).q);
    expect(html).toBe('<p><a href="https://example.org/{q1}" target="_blank" rel="noopener noreferrer">Map</a></p>');
  });

  it("still fills in the patient's details, e.g. the Profile block's name", () => {
    const m = modelFor({ pages: [{ name: "p1", elements: [{ type: "profile", name: "pr" }, { type: "html", name: "i1", markdown: "Hello {patientName}" }] }] }, "patient", {
      name: "Sam <b>Sample</b>", age: 54, sex: "female",
    });
    expect(informationHtml(m.getQuestionByName("i1") as QuestionHtmlModel)).toBe("<p>Hello Sam &lt;b&gt;Sample&lt;/b&gt;</p>");
    const inner = (m.getQuestionByName("pr") as unknown as { contentQuestion: QuestionHtmlModel }).contentQuestion;
    const profile = informationHtml(inner);
    expect(profile).toContain("<b>Name</b> Sam &lt;b&gt;Sample&lt;/b&gt;");
    expect(profile).toContain("<b>Age</b> 54");
  });
});

describe("survey-level HTML (verification 1, F3)", () => {
  it("is dropped from every survey, since SurveyJS would draw it as it is", () => {
    const json: ChapterJson = {
      completedHtml: "<img src=x onerror=alert(1)>", completedHtmlOnCondition: [{ expression: "true", html: "<script>x</script>" }],
      completedBeforeHtml: "<b>x</b>", loadingHtml: "<b>y</b>", pages: [{ name: "p1", elements: [] }],
    };
    const clean = withoutSurveyHtml(json);
    for (const k of SURVEY_HTML_KEYS) expect(clean[k]).toBeUndefined();
    expect(clean.pages).toEqual(json.pages);
    expect(json.completedHtml).toBeDefined(); // the original isn't changed
    for (const viewer of ["patient", "clinician"] as const) {
      const m = modelFor(json, viewer, { name: "Sam", age: 54, sex: "female" });
      // SurveyJS's own words, not the chapter's HTML.
      expect(m.completedHtml).not.toContain("<img");
      expect(m.loadingHtml).not.toContain("<b>");
      expect(m.completedBeforeHtml).not.toContain("<b>");
      expect(m.completedHtmlOnCondition).toHaveLength(0);
    }
  });
});
