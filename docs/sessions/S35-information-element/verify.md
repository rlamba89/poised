# S35 verification

## Verification 2 (25–26 Oct 2026, automatic run)

**Verdict: Verified.**

Verification 1's findings are fixed, and I checked them adversarially, F1 above all. Nothing I tried put a live link, picture or script on the page, and no remaining problem is worth another round. The minor findings below are in pending.md ("From S35's verification 2").

### Verification 1's findings, checked again

- **F1 (answers in placeholders): fixed.** `informationHtml` (`packages/clinical/src/information.ts:342-366`) turns the text into HTML and cleans it with the `{placeholders}` still in place. It then fills only the text between tags, with each value escaped and kept on one line. The value is never re-scanned, and it never goes into an attribute.
  - **Unit probe** (a temporary Vitest file, deleted). I used an answer holding a Markdown link, a Markdown picture, an HTML `<a>` and an `<img>`, in these places:
    - in a Markdown paragraph, heading, bullet and numbered item, italic, and a link's words;
    - in a picture's `alt` and a link's address;
    - in an old statement's `<p>`, `<li><b>` and `<h2>`, an `<a>`'s words, `href` and `title`, an `<img>`'s `src` and `alt`, and a `<script>`;
    - with several lines (`# h`, `- l`);
    - in a checkbox's "other" text and a dropdown choice's text;
    - in a repeating group's `{panel.t}`;
    - nested, as an answer `{q2}` whose own answer is a link.

    Every value came out as escaped text. No answer-made `<a>` or `<img>` appeared, addresses and `alt` kept `{q1}` literally, and the nested `{q2}` wasn't filled.
  - **Chrome (trk, Alex Author):** I made a probe HQ through the API. It had a Markdown block and an old statement that each show `{q_v}` in a paragraph, heading, list, link words and link address. It also had hand-made HTML (an `onerror` picture, `<svg onload>`, an entity-encoded `javascript:` link, `onclick`/`style`/`class`, `<iframe srcdoc>`) and survey-level `completedHtml`/`loadingHtml`. I typed an answer with a Markdown link and picture, `# Big`, `- l`, an `<img onerror>` that sets the title and a window variable, an HTML `<a>` and `{q_v}`.
    - **Patient and Clinician Preview:** 0 `evil.example` links or pictures, no element in an Information block had `onerror`, `onclick`, an `<svg>` or an `<iframe>`, `window.__v2` stayed unset and the title didn't change.
    - **Test report 3's episode, as Alex:** Validate and the POA Summary also showed 0 `evil` links or pictures, and both blocks displayed the payload as words only.
    - I deleted the probe HQ afterwards.
- **F2 (list numbers): fixed.** `<ol start>` is written from the first item's number, and the sanitiser keeps only a digits-only `start` (`information.ts:208-210`). In Chrome, 1 / gap / 2 / gap / 3 showed as `start` none, 2 and 3, in the Preview and in Validate. `start="1 onclick=x"` is dropped.
- **F3 (survey-level HTML): fixed.** `withoutSurveyHtml` is applied at the only three places a `Model` is built: `modelFor` (`testcases.ts:40`), `clinicianModel` (`clinicianView.ts:11`) and the Preview (`PreviewPage.tsx:90`). Unit tests cover it in clinical and in `clinicianView.test.ts`. The probe's `completedHtml` didn't run.
- **F4 (translations' lines): fixed.** `informationPatch` uses `htmlToLines` (`information.ts:321-327`), and the unused `textToHtml`/`plainText` are gone from `doc.ts`.
- **F5 (entities): fixed.** Named references other than the basic six are passed through. They can only ever be text, and the split in `fillPlaceholders` still sees only our own tags.
- **Nits: fixed.** `refusedLinks` now looks inside bold and italic, an empty `<p></p>` becomes a blank line, and "information block" is lower case.

### Findings (minor, none blocking; in pending.md)

1. **Editing an old statement reads its lines as marks** (`information.ts:292`, `informationText` → `informationPatch`). Once edited, a line like `# of tablets` or `1. Stop aspirin` becomes a heading or a list.
   - Low risk: the editor and the import only ever wrote escaped plain text, so no link or bold can be lost, and the live preview shows the change before the author leaves the box.
2. **A `class` of plain words survives on any allowed tag** (`information.ts:204`). HTML written through the API could borrow `sd-btn` and look like a button. Nothing can run, and only staff who can change questionnaires can write the JSON. Allowing only `sj-…` classes would close it.
3. **The safe renderer is registered by importing `InformationQuestion.tsx`** in each survey page (all three do today). A new page that forgets the import gets SurveyJS's raw drawing back. Registering it once at app start would remove the trap.
4. **A picture inside a link** (`[a ![b](…)](…)`) is misread as a broken link.
5. **Small:**
   - `[words]()` lists an empty address in the orange note;
   - `&amp` without its `;` in hand-made HTML shows literally;
   - `elementLabel` formats a whole block just to take its first 60 characters.

`/code-review high` also raised two design points that I don't count as findings:
- the import still writes the old `html`, which was decided ("No Go change", decisions.md, 25 Oct);
- `InformationSettings` duplicates `BlurTextarea`'s blur-save, which is a quality nit.

### Checked and fine

- **Scope:** the brief and answers.md (1a, 2a, 3a) are fully built, and nothing big was added beyond them.
- **Tests:**
  - 288 clinical unit tests, including the new F1–F5 cases, plus the new `clinicianView.test.ts`;
  - J10 extended, and `test_information.py`, now with the patient's link-and-picture answer seen by the patient and by Cara;
  - no new routes, so no new integration cases are needed.
- **Tenant isolation:** no query changes.
- **A-20 grep:** only the existing comments in `format.ts`/`patient.go` and `infra/countries.py`.
- **Security:**
  - no secrets;
  - no patient data in logs or messages;
  - Go still strips clinician-only elements;
  - pictures are sent with no referrer, and links get `noopener noreferrer`.
- **Docs:**
  - decisions.md, pending.md, requirements.md, the plan, plans/README and sessions/README are updated;
  - `repocheck` passes.

### Test results (25–26 Oct, this verification)

- `make lint`: passed.
- `make test`:
  - Go: all ok;
  - web: 26 files, **261 passed**;
  - clinical: 21 files, **288 passed**;
  - CDK: **124 passed**.
- `make test-integration`: every package ok.
- `make e2e` (against the conductor's `make dev`): **81 passed, 5 skipped, 1 failed**.
  - The failure was `test_j7_patient_invites.py::test_j7_invite_by_text_and_email_then_date_of_birth_and_send`, which found 3 captured texts for its made-up mobile. That's the known clash with earlier runs' messages on the shared database, as in the build notes, and S35 touches nothing there.
  - Run again alone: **6 passed**.
- **Browser:** see F1–F3 above.

### Hand checks for Rahul (don't block)

- An HQ imported from an incumbent export: its statements show as before.
- A real picture from a real website loads on a real phone.
- A `tel:` link on a real phone offers to call.

### Commits

1. `S35: the Information element in packages/clinical`
2. `S35: Information in the editor and every survey, cleaned before display`
3. `S35: J10 extended and the Information journey`
4. `S35: docs` (never `docs/market/` or `docs/sessions/run-state.md`)

## Verification 1 (25 Oct 2026, automatic run)

**Verdict: Needs fixes.**

The sanitiser itself holds up well. I tried about 40 hand-made payloads, in unit form and in the real Preview: `javascript:` with odd case, leading spaces, tabs and entities; `data:`; quote breaking in `href`, `alt` and `class`; `srcset`, `style`, `onerror` and `onload`; SVG and MathML; `<noscript>`, `<textarea>`, `<xmp>`, `<iframe srcdoc>` and `<base>`/`<meta>`/`<form>`; comment tricks (`<!-->`, `--!>`); and `<scr<script>ipt>`. None ran a script. Every result was idempotent, and `window.__v1…__v9` and `document.title` stayed unchanged in Chrome. Because the output is rebuilt from scratch with escaped, double-quoted attributes, no script can get through.

There are still two real problems. Values filled into `{placeholders}` can still put **live links and pictures** in front of a clinician (F1). Numbered lists can show **wrong step numbers** to patients (F2). Both are below the bar for an element whose job is safe, correct patient information.

### Findings (worst first)

**F1. A patient's answer becomes a real link or picture in an Information block (or an old statement) that shows it with `{placeholder}`.** `packages/clinical/src/information.ts:322-331` (`informationHtml`).
- **Markdown path:** `getProcessedText` fills the answer in *before* the marks are read. An answer like `[Click here to sign in again](https://evil.example/login) ![](https://evil.example/pixel.png)` comes out as a working link and a remote picture.
- **Old-statement path:** `locHtml.renderedHtml` fills it in raw, and the sanitiser then *keeps* `<a href="https://…">` and `<img src="https://…">`, because they are allowed tags.
- **Confirmed in Chrome:** in a draft's Preview, an Information block `You said: {qv1}` and an old statement `<p>Old: {qv1}</p>` both rendered `evil.example` links, and the first also rendered an `evil.example` picture.
- **Impact:** the same drawing is used in Validate, so a patient can put a clickable link with any wording in front of a clinician (phishing), plus a tracking picture that reports when the clinician opened the set. No script runs, and this is much better than before S35, when it was full XSS.
- **Docs to correct:** the build notes and the code comment say answers are "escaped like everything else typed". That's true for HTML tags, not for marks, links or pictures.
- **Fix:** a placeholder's value must show as plain text, without marks, links, pictures, headings or new lines/paragraphs, in both paths. One way: turn the author's text into HTML and clean it with `{name}` left in place, then fill each placeholder in the text with its value HTML-escaped (e.g. `q.getProcessedText("{name}")` per placeholder). Values that land inside an `href`/`src` should be left out or URL-encoded.
- **Test first:** unit tests for both paths (a link, a picture, `# ` and a multi-line answer), plus a journey assertion if it's cheap.
- The Profile block's `{patientName}` and the Clinical Summary notes go through the same path, so they're covered by the same fix.

**F2. Numbered lists restart at 1.** `information.ts:56-84` (`markdownToHtml`), and the sanitiser has no `start` on `<ol>` (`information.ts:166-178`).
- These all show "1. … 1. … 1.":
  - `1. Stop eating at midnight`, a blank line, `2. Take your usual tablets`, a blank line, `3. Arrive at 7am`: the blank line closes the `<ol>` and opens a new one;
  - a numbered item, a plain line, then the next numbered item;
  - a list that starts at `3.`.
- **Impact:** wrong step numbers in pre-op instructions are a clinical risk.
- **Fix:** keep the author's number. Write `<ol start="N">` from the first item's number, and let the sanitiser keep a digits-only `start` on `ol`. Optionally, a blank line between two numbered items could also leave the list open. Unit tests first.

**F3. The claim that "nothing in an HQ's JSON can run" is broader than the code.** `apps/web/AGENTS.md` (the S35 paragraph) and `InformationQuestion.tsx`.
- Only the `html` question is cleaned. Survey-level HTML (`completedHtml`, `completedHtmlOnCondition`, `completedBeforeHtml`, `loadingHtml`) is still drawn raw by survey-react-ui if a chapter's JSON holds it. That can only happen through the content API, since the editor never writes these, and it was true before S35.
- **Fix (recommended):** drop or `sanitizeHtml` those keys wherever a survey model is built (`modelFor`, ValidateSet, PreviewPage, `clinicianView.ts`), with a unit test. **Or (smaller):** narrow the AGENTS.md wording and add it to pending.md.

**F4. An old statement's translations lose their line breaks when it's first saved.** `information.ts:301` (`informationPatch`) uses `plainText` for the translations but `htmlToLines` for the English. A translated `<h2>…</h2><ul><li>…</li></ul>` becomes one run-together line.
- **Fix:** use `htmlToLines` here, with a unit test. `textToHtml`/`plainText` in `doc.ts:707-…` then have no production caller; remove them or say so.

**F5 (minor). Named entities other than the six known ones show literally** (`&eacute;`, `&pound;`, `&rsquo;`), in old or hand-made statements and in their editor text. `information.ts:248-255` (`decodeEntities`).
- The editor and Go's import never write these, so the risk is low.
- **Fix (cheap and safe):** an unknown `&name;` can be passed through unescaped, because a named reference in text can never form markup. Or decode a few common ones.

**Nits (fix if touching the file, otherwise pending.md):**
- `refusedLinks` (`information.ts:110`) doesn't warn about a refused link inside `**bold**`, `*italic*` or a link's words, e.g. `**[Form](http://…)**`.
- `htmlToLines` (`information.ts:287`) drops a gap made by an empty `<p></p>` (as opposed to `<p>&nbsp;</p>`) in hand-made HTML.
- `keys.go:516`: "Information block" is capitalised inside sentences ("changed from Information block to single choice"), unlike the other kind names. "information block" reads better.

### Checked and fine

- **Scope:** the brief and answers.md (1a, 2a, 3a) are built. Nothing big beyond the brief: cleaning every `html` element is justified in the build notes.
- **Tests first:** 28+ unit tests in `information.test.ts`, J10 extended, and the new `test_information.py`. There are no new routes, so no new integration cases are needed. The Go change is wording only.
- **Tenant isolation:** no query changes.
- **A-20 grep:** only the existing comments in `format.ts`/`patient.go` and the infra tests that assert UK values are absent. Nothing new.
- **Security:**
  - no secrets;
  - no patient data in logs or messages;
  - Go still strips clinician-only elements;
  - pictures are sent with `referrerpolicy="no-referrer"`;
  - links get `rel="noopener noreferrer"`.
- **Docs:** decisions.md, pending.md, requirements.md, the plan, plans/README and sessions/README are updated (status "Tested"). `repocheck` passes.

### Test results (25 Oct, this verification)

- `make lint`: passed.
- `make test`:
  - Go: all ok;
  - web: 25 files, **260 passed**;
  - clinical: 21 files, **277 passed**;
  - CDK: **124 passed**.
- `make test-integration`: every package ok.
- `make e2e`: not re-run by me. The conductor's full run after the fixes gave **82 passed, 5 skipped**, which matches the build notes.
- **Browser (Claude in Chrome, trk, Alex Author):**
  - a draft HQ with hand-made payloads in an old statement and in an Information block: nothing ran, every `__v*` was unset and the title was unchanged;
  - typed `<img onerror>`, `javascript:` and `data:` showed as text;
  - F1 was reproduced as above.
  - The probe questionnaire was deleted afterwards.

### Hand checks for Rahul (from the build notes; don't block)

- An HQ imported from an incumbent export: its statements show as before.
- A real picture from a real website loads on a real phone.
- A `tel:` link on a real phone offers to call.

### Suggested commits (once fixed and verified)

1. `S35: the Information element in packages/clinical`: `packages/clinical/src/{information,doc,translations,questionTypes,index}.ts`, `information.test.ts`, `doc.test.ts`.
2. `S35: Information in the editor and every survey, cleaned before display`: `apps/web/src/features/information/`, `features/editor/{InformationSettings,SettingsPanel,ElementCard,AddContentBar,TranslateView}.tsx`, the three survey pages, `globals.css`, `apps/web/AGENTS.md`, `apps/api/internal/{keys/keys.go,httpapi/library.go}`.
3. `S35: J10 extended and the Information journey`: `e2e/journeys/{test_j10_authoring,test_information}.py`, `e2e/support.py`, `e2e/data/information-hq.json`, `e2e/README.md`.
4. `S35: docs`: this folder, `docs/{decisions,pending,requirements}.md`, the plan and the session status. Never `docs/market/` or `docs/sessions/run-state.md`.

**For the builder:** "Verification found problems: read `docs/sessions/S35-information-element/verify.md`, fix them test-first, then ask me for a retest."
