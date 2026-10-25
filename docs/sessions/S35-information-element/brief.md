# S35: The Information element

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 5 of 6.
**Requirements:** INF-03 ([saas-requirements.md](../../saas-requirements.md) §7).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S34 verified.

**What exists today:**
- A **Statement** element (`kind` "statement", a SurveyJS `html` element): plain text typed in a box, turned into escaped `<p>` lines (`textToHtml` and `plainText` in `doc.ts`; `SettingsPanel.tsx`, `ElementCard.tsx`). Go's incumbent import makes the same escaped HTML (`hqimport/convert.go`). So no author can put raw HTML in front of a patient today.
- `html` is one of the translatable keys (`LOCALIZABLE_KEYS` in `doc.ts`).
- The web app has few dependencies (Mantine core, SurveyJS, React Router). There's no Markdown library, rich-text editor or HTML cleaner yet.

**Ask Rahul first:**
1. **How authors format Information text.** (The plan said to decide this at the start of C5.)
   - (a) **Simple marks typed in the text** (like `**bold**`, `- list`, `[link](https://…)`), with a live preview beside it. We turn it into HTML ourselves, allowing only a few safe tags, so nothing unsafe can get through. No new libraries.
   - (b) A Word-like toolbar editor (Mantine's, built on Tiptap). Easier for authors, but about 10 new packages, and the HTML it saves must be cleaned with an extra library.

   **Recommended: (a).**
2. **Statement and Information.**
   - (a) **One block: Statement becomes Information**, plain text plus formatting. Old statements still show as before.
   - (b) Two blocks side by side.

   **Recommended: (a).** Fewer kinds for authors to choose from.
3. **Pictures in Information text.** We can't store files until plan C8.
   - (a) **A picture is a link to an image already on the web** (`https://` only). Uploading comes with C8.
   - (b) No pictures until C8.

   **Recommended: (a).**

## Build

(As answered; this assumes the recommendations.)
- **`packages/clinical`, test-first:**
  - the converter from formatted text to HTML: paragraphs, headings, bold, italic, lists, links (`https:`, `mailto:`, `tel:` only; they open in a new tab), pictures (`https:` only), everything else escaped;
  - **the sanitiser**: a final pass over the HTML shown, keeping only the allowed tags and attributes, run on every Information block before display (old statements and imported ones too). Unit tests with script tags, `javascript:` links, event attributes and broken HTML.
- **Editor:** the Statement kind becomes **Information**: a text box with the preview, and a short "how to format" hint. Old statements open as their plain text.
- **Patient and clinician views:** Information shows formatted, at phone size, with readable pictures (never wider than the screen) and links that work.
- **Translations:** Information text can be translated (TranslateView), like the statement's `html` today.
- **J10 extended:** the author adds an Information block with a link and a list; the patient sees it formatted; a `<script>` typed into the text shows as text.

## Not in this session

- Uploading pictures or leaflets (C8).
- The patient's Information on the home and the estimated time (S36).

## Manual test focus

1. Type a heading, bold text, a list, a link and a picture link: the preview and the patient view (phone size) match.
2. Type `<script>alert(1)</script>`, `[x](javascript:alert(1))` and `<img src=x onerror=alert(1)>`: each shows as harmless text, with no pop-up.
3. An old HQ's statements look exactly as before; editing one keeps its text.
4. An imported incumbent HQ's statements still show.
5. Translate an Information block: the patient sees the translation in that language.
