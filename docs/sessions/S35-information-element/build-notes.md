# S35 build notes: the Information element

Builder, automatic run, 25 Oct 2026.

## Answers used

[answers.md](answers.md) (Rahul, 25 Oct), all three as recommended: (1) **simple marks typed in the text** (Markdown-style) with a live preview, no new libraries; (2) **Statement becomes Information** (one block); (3) **pictures by https link only** until C8.

## What was built

### `packages/clinical` (TypeScript, for Rahul to review)

- **`information.ts`** (new):
  - `markdownToHtml(text)`: the author's marks → HTML. `#`, `##`, `###` headings (shown as h2, h3, h4); `**bold**`; `*italic*` (a star next to a space stays a star, so "2 * 3" is left alone); `- ` / `* ` / `+ ` bulleted lists and `1. ` / `1) ` numbered lists from consecutive lines; `[words](link)`; `![what it shows](picture)`. **One paragraph per line**, as statements were; a blank line between lines is a gap (an empty paragraph), as in statements (changed after test report 1). **Everything else is escaped**, so `<script>` typed in the text is shown as text.
  - Links: **https, mailto and tel only** (`allowedHref`), opening in a new tab with `rel="noopener noreferrer"`. Pictures: **https only** (`allowedSrc`), with `referrerpolicy="no-referrer"` and `loading="lazy"`. Any other link or picture shows as the text typed; `refusedLinks(text)` lists them for the editor's warning.
  - **`sanitizeHtml(html)`**: the last step before anything is shown. A small tokenizer reads the HTML and **rebuilds it from scratch**: only `p br b strong i em u s ul ol li h1–h6 a img div span blockquote hr`; a `class` only if it's plain words; a link only with an allowed address (else its words stay, without the link); a picture only from https (else dropped). Every other tag is dropped (its text kept), and `script`, `style`, `iframe`, `textarea`, `template`, `object` and the like are dropped with their content. Comments go; entities are decoded and all text is escaped again. **Why it's safe without a library:** the output is written by us, so it can only hold the allowed tags and attributes with escaped values, however oddly the input was written (the worst a misread can do is show some markup as text).
  - The element: `informationText(el)` (what the editor shows: the marks as typed, or an old statement's plain text), `informationPatch(el, text)` (saves the text in `markdown` and removes `html`; an old statement's translations become their plain text), `informationLabel(el)` (its first words, for labels and the Translate view), `informationPreview(el)` (the editor's card and preview), and **`informationHtml(question)`** (what every survey shows: the marks for the survey's language, or an old statement's HTML; always through `sanitizeHtml`, and only then each `{placeholder}` between tags filled with its value as plain text: escaped, on one line, never a mark, link or picture; one inside a link's address isn't filled). Since verification 1: `withoutSurveyHtml` drops survey-level HTML (`completedHtml`, `completedHtmlOnCondition`, `completedBeforeHtml`, `loadingHtml`) wherever a survey is built (`modelFor`, the clinician view, the Preview).
- **`doc.ts`:** the kind `statement` is now **`information`** ("Information"; still SurveyJS's `html` type). A new one is `{ type: "html", name, markdown: "Information for the patient" }`. `markdown` is a translatable key. `elementLabel` uses `informationLabel`. `ElementJson.html` and `.markdown` are typed as translatable strings.
- **`questionTypes.ts`:** registers `markdown` on SurveyJS's `html` type as a translatable property, so a survey shows the right language.
- **`translations.ts`:** both `html` (old statements) and `markdown` are listed as "Information". The `markdown` text is translated as typed, marks and all, so the translator keeps them.

### `apps/web` (TypeScript)

- **`features/information/InformationQuestion.tsx`:** replaces SurveyJS's own drawing of **every** `html` element (Information, old statements, imported ones, the Profile block and the Clinical Summary lists) with one that always goes through `informationHtml`. Imported once by `PatientHQ`, `ValidateSet` and `PreviewPage` (the only pages with a survey), as the Video element is.
- **`features/editor/InformationSettings.tsx`:** the Text box (saves when it loses focus, as other boxes do), a short **How to format** list, an orange warning when a link or picture will show as text, and a **Preview** that follows every key press. The Clinical switch stays below.
- **Editor card:** shows the block formatted (`informationPreview`). Add bar: **Information** (an "i" icon) instead of Statement.
- `globals.css`: `.sj-info`: headings, lists, links (brand teal 8, darker than SurveyJS's own link colour for contrast), **pictures never wider than their column**, long links wrap.

### Go

- Wording only: the library's refusal says "Information blocks, calculations and the profile hold no answer…", and the keys' kind name for `html` is "Information block". **Nothing else changes in Go**: the type is still `html`, so keys, pre-fill, the patient strip and the incumbent import treat it as before, and Go never renders it.

### End-to-end

- **J10 extended** (`test_j10_authoring.py`): Alex adds an Information block with a heading, a list, an https link and a typed `<script>`; the preview shows them formatted and the script as text; the patient on a phone (WebKit) sees it formatted, the link opening in a new tab, the `<script>` as text, and nothing runs (`window.__xss` stays unset).
- **`test_information.py`** (new, fixture `data/information-hq.json`): an old statement opens as its plain text and keeps it when edited; a block with hand-made HTML (an `onerror` picture, a `javascript:` link, a script, an `onclick`) shows only its words, in the editor and the surveys; the live preview (heading level 2, bold, italic, both lists, an https link; `javascript:` link, `<script>`, `<img onerror>` as text; the warning); the Preview with the picture and the German translation; the patient on a phone: formatted, links with `target="_blank"` and `rel`, the picture loaded, no wider than the screen, no sideways scroll; Cara's Validate cleaned the same way.
- `support.serve_test_pictures(context)`: answers `https://pictures.example.org/…` with a made-up picture 1200 px wide, so it runs offline and the picture is wider than a phone.

## Decided by the builder (automatic mode)

- **The text is stored as typed (`markdown`), not as HTML.** One source of truth, translated as typed; the HTML is made only when shown. Old statements keep `html` until edited.
- **One paragraph per line**, as statements were, rather than strict Markdown (where lines join into one paragraph). Authors press Enter for a new line and get one. **A blank line is a gap**, as statements had (`<p>&nbsp;</p>`), so an old statement looks the same after editing, and the preview matches its card (test report 1, exploring 3). Blank lines before the first line or after the last show nothing.
- **Every `html` element is cleaned, not only Information**, including the Profile block and the Clinical Summary lists, because SurveyJS fills `{patientName}` and answers (`{q1}`) into `html` without escaping: a patient's name or answer holding HTML could otherwise run in a clinician's browser. This was possible before S35.
- **`{placeholders}` still work in Information** (as they did in statements). Since verification 1 (F1) they're filled in *after* the marks are read and the HTML cleaned, with the value as plain text, so an answer can't become a link, picture, heading or new paragraph.
- **Headings `#`–`###` show as h2–h4**, so a block's heading never outranks the page's own title.
- **Pictures are sent with no referrer** (the picture's site isn't told which page is open) and load lazily.
- **Translations are tested in the editor's Preview**, because patients can't choose a language yet (true of every text before S35; noted in pending.md for Rahul).
- **The editor warns** (orange) when a link or picture will show as text, naming it; Publish isn't blocked (it's harmless, and the preview shows it).

## Tests (each new test failed first, then passed)

- **TypeScript:** `packages/clinical/src/information.test.ts` (28): the marks (paragraphs, headings, bold and italic, both lists, links and pictures allowed and refused, everything else escaped, attribute quotes); `refusedLinks`; **the sanitiser** (allowed tags kept; old and imported statements, the Profile and summary lists unchanged; script, style, iframe dropped with their content, even unclosed; event attributes, `style`, `id`, odd classes, svg and forms dropped; `javascript:` in every disguise (case, spaces, entities, a tab, `data:`, `http:`); broken HTML: unclosed tags closed, stray closers dropped, a lone `<`, comments, `<` inside attributes, `<<script>`; entities; cleaning twice changes nothing); the kind (label, new element, old statements open as their text, saving moves them to `markdown` with translations, `updateElement` keeps translations, labels without marks or script text); translations; `informationHtml` with a SurveyJS model (marks, an old statement cleaned, German and the English fallback, an answer with HTML never put on the page, as marks or in an old statement). `doc.test.ts`: the kind's name.
  - Honest note: `information.test.ts` was written first and failed (no module); `information.ts` then made it pass. The later additions (`informationPreview`, the import's escapes, labels without script text and without gaps) were each a failing test first.
- **Go:** wording only; the existing tests pass.
- **End-to-end:** J10 extended; `test_information.py` (2 tests).

### Results (25 Oct)

- `make lint`: passed.
- `make test`: Go all ok; web 25 files, 260 passed; clinical 21 files, 276 passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (all journeys, on a dev stack on spare ports 3093/8093 on the shared `sj` database, stopped afterwards): **81 passed, 5 skipped, 1 failed**: `test_j7_patient_invites.py::test_j7_invite_by_text_and_email_then_date_of_birth_and_send` found two captured texts for its made-up mobile (a collision with an earlier run's messages on the shared database; S35 touches nothing there). Run again alone: **6 passed**. After the last changes, `test_information.py`, J10 and J1 passed again (4 passed).
- **Checked at phone size by me:** the patient's form at 390 px (WebKit, iPhone 13): heading, bold, list, links in brand teal, the 1200 px picture shrunk to the column, no sideways scroll; and the editor's settings, preview, card and the Translate view on a desktop.

## Hand checks for Rahul

- **An HQ imported from an incumbent export** (`go run ./cmd/import-hq -file …`): its statements show as before in the editor, the Preview and the patient's form. The tester can't run the import; covered by `hqimport`'s Go test (the HTML it writes) and `information.test.ts` (that exact HTML is shown unchanged).
- **A real picture from a real website** loads on a real phone (the journeys answer a made-up address offline).
- **A `tel:` link on a real phone** offers to call.

## Suggested commits

1. `S35: the Information element in packages/clinical` — `packages/clinical/src/{information,doc,translations,questionTypes,index}.ts`, `information.test.ts`, `doc.test.ts`.
2. `S35: Information in the editor and every survey, cleaned before display` — `apps/web/src/features/information/InformationQuestion.tsx`, `features/editor/{InformationSettings,SettingsPanel,ElementCard,AddContentBar}.tsx`, the three survey pages' import, `globals.css`, `apps/web/AGENTS.md`; Go wording (`internal/keys/keys.go`, `internal/httpapi/library.go`).
3. `S35: J10 extended and the Information journey` — `e2e/journeys/{test_j10_authoring,test_information}.py`, `e2e/support.py`, `e2e/data/information-hq.json`, `e2e/README.md`.
4. `S35: docs` — this folder, `docs/{decisions,pending,requirements}.md`, plan and session status.

## Test rounds

- Test plan written 25 Oct ([test-plan.md](test-plan.md)).
- **Test report 1** (25 Oct): 23 passed, **1 failed (F1)**.
  - **F1, fixed:** Information headings were 48px / 32px in every survey (SurveyJS's own `.sd-html h2/h3` rules, with the same specificity as ours, won) against about 21px / 18px in the editor. Now `div.sj-info h2/h3/h4` have fixed sizes and line heights (20px/28px, 18px/26px, 16px/24px), the same in the editor and every survey. **Test first:** `test_information.py` now checks the Parking and Fasting headings' size and line height are the same in the settings' preview and the Preview page, and that the patient's Parking heading is 18px and no bigger than the set's title (failed at 32px before the fix).
  - **Exploring 1, fixed:** hand-made HTML with words outside paragraphs ran together in the Text box and the Translate view ("Imported textbad linkClick me"), and editing it kept them joined. `htmlToLines` (new, `information.ts`) now gives a line per paragraph, list item, line break or loose run of words, cleaned first; the Text box, Translate's English column and its CSV use it. Unit tests first.
  - **Exploring 2, fixed:** Translate's English column now wraps long unbroken text (`overflow-wrap: anywhere`).
  - **Exploring 3, fixed:** a blank line is now a gap, as in statements (above). Unit tests first (`markdownToHtml` gaps; an old statement's text turned back into HTML is unchanged).
  - **Exploring 4–7: not S35, no code change** (pending.md, "From S35's test report 1"): the patient's set-list badge and Review button cut short at 390px (already noted after S34); a reload inside a set goes back to the sections list; Preview opened while "Saving…" may miss the last edit (not confirmed); the editor's "Clinical Page" header on a patient set's page.
  - Results after the fixes: `make lint test test-integration` pass (web 260, clinical 277, CDK 124, Go all ok); journeys `test_information.py`, J10, `test_video.py`, J1, J3, J4, J5: 12 passed (spare ports 3093/8093, stopped afterwards). Checked the patient's form at 390px again: Parking is a modest heading.
- **Test report 2** (25 Oct, retest): **all passed, 10 of 10.** **S35 tested.** One cosmetic note carried over (loose words in hand-made HTML spaced unevenly on the editor card; pending.md, "From S35's test report 2"); no code change.
  - Full `make e2e` after the fixes (spare ports 3093/8093 on the shared `sj` database, stopped afterwards): **82 passed, 5 skipped**, none failed.
- **Verification 1** (25 Oct): Needs fixes. All fixed test-first:
  - **F1 (security), fixed:** a patient's answer filled into a `{placeholder}` became a working link or picture (Information: filled before the marks were read; old statements: filled raw, and `<a>`/`<img>` with https are allowed). Now `informationHtml` turns the text into HTML and cleans it with the placeholders still in it, then fills each placeholder between tags with its value escaped and on one line; one inside a link's address isn't filled. Both paths read the raw text (`pureText`), not SurveyJS's processed one. Unit tests first (a link, a picture, a heading, several lines, an HTML link, for both paths; an address; the Profile block's `{patientName}` still filled, escaped). Journey: `test_information.py` has the patient type `[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)` into a question shown back by a block (`{q_note}`); the patient and Cara see it as words only. Checked that the journey fails with the old code and passes with the fix.
  - **F2, fixed:** a numbered list starts at its first item's number (`<ol start="N">`; the sanitiser keeps a digits-only `start` on `ol`), so steps split by a gap or a line, or starting at 3, keep their numbers.
  - **F3, fixed:** survey-level HTML is dropped from every survey (`withoutSurveyHtml` in `modelFor`, `clinicianModel` and the Preview); unit tests in clinical and a new `apps/web/src/features/preview/clinicianView.test.ts`.
  - **F4, fixed:** an old statement's translations keep their lines when it's first saved (`htmlToLines`). The old `textToHtml`/`plainText` (no callers left) are removed from `doc.ts`.
  - **F5, fixed:** named characters other than the six basic ones (`&eacute;`, `&pound;`, `&rsquo;`) are passed on for the browser to show (they can't form a tag), and the editor's text decodes the common ones.
  - **Nits, fixed:** `refusedLinks` also warns about a refused link inside bold or italic; `htmlToLines` keeps an empty `<p></p>` as a blank line; the keys' kind name is "information block" (lower case, like the others).
  - Results after the fixes: `make lint test test-integration` pass (web 261, clinical 288, CDK 124, Go all ok); full `make e2e` on spare ports 3093/8093 (stopped afterwards): **82 passed, 5 skipped**, none failed. The test plan's setup script was re-checked with the new blocks.
- **Test report 3** (25 Oct, retest after verification 1): **all passed, 7 of 7.** **S35 tested again**, for re-verification. Exploring notes (no code change, pending.md "From S35's test report 3"): a placeholder in a picture's description or a link's address stays as written (by design, F1); Validate's "Patient answered" note doesn't wrap a long answer (not S35); a gap between numbered steps is a little bigger in the Settings Preview than on the card.
