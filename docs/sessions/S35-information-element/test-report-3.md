# S35 test report 3 (retest after verification 1)

**Summary: 7 passed, 0 failed, 0 blocked.** The retest list: the new T-23 to T-26, then quick re-runs of T-13 (the patient at 390px), T-11 and T-18.

- **Environment:** HEAD `f6a336c`. The working tree isn't clean: it holds the uncommitted S35 build and the verification-1 fixes, plus `docs/market/`. Sunday 25 Oct 2026, about 23:33 to 23:41 GMT. http://localhost:3000 (web) and :8080 (API), started by the conductor (`make dev`, freshly migrated and seeded).
- **Browser:** Claude in Chrome ("trk" profile) with a desktop window. Patient cases ran in a same-origin 390px iframe, as in earlier rounds.
- **Test data:**
  - **Questionnaire: S35 test HQ MT 2335 retest2** (`3dd15577-…`). I made it with the updated setup script (with q_note and i_echo), then made these additions:
    - **By API (T-25):** an old statement `i_ent`, `<p>caf&eacute; &pound;5 it&rsquo;s</p>`.
    - **In the editor (T-24):** two Information blocks, the "1 / blank / 2 / blank / 3" steps and the "3. Three / 4. Four" list.
    - **By API (extra check for F1's second path):** an old statement `i_oldecho`, `<p>Old: {q_note}</p>`.
  - v1 is published. v2 is a draft that holds only exploration blocks.
  - **Episode:** **Mt S35Retest3 Green** (`eca93b04-81ac-42bb-934d-25b5e406e305`, Hospital A1). The answers were sent, and Validate was opened as Cara.
- **No browser dialogs.** The XSS payloads set `document.title` instead of calling `alert`.

## Results

| Case | Result | Note |
|---|---|---|
| T-23 | Pass | **The patient at 390px:** the answer was `[Sign in again](https://evil.example/login) ![](https://evil.example/p.png)` / `# Big` / `two <img src=x onerror="document.title='XSS'">` over three lines. The block shows "You told us:" with exactly that text in bold, as one paragraph: no link, no picture, no heading, and the new lines became spaces. The DOM has 0 `a[href*=evil]`, 0 `img[src*=evil]` and 0 `img[src=x]`, the only `<strong>` is the bold, and the title is unchanged. **The old-statement path** (`<p>Old: {q_note}</p>`) shows the same text, with no child elements at all. **Validate (Cara):** the same for both blocks, with 0 evil links or pictures. **The POA Summary:** 0 evil links or pictures. **The chapter Preview (draft):** the same. |
| T-24 | Pass | `1.` / blank / `2.` / blank / `3.` shows 1, 2 and 3 with gaps (`<ol>`, `<ol start="2">`, `<ol start="3">`). `3. Three` / `4. Four` shows 3 and 4 (`<ol start="3">`). It was correct in the live Settings Preview, on the card, in the chapter Preview, on the patient's form (390px) and in Validate. Never "1. 1. 1.". |
| T-25 | Pass | I added the block by API, as the setup script does. The card, the Settings Preview, the Text box, the chapter Preview, the patient's form and Validate all show "café £5 it’s". |
| T-26 | Pass | In Translate (German), I gave i_old "Bitte früh kommen." / "Bademantel mitbringen.", which saved as html with `<p>`s. I then added " Thanks" to the English and left the box. Translate still shows the German on two lines, it's stored as `markdown.de` `"Bitte früh kommen.\nBademantel mitbringen."`, and the Preview in German shows two paragraphs. |
| T-13 | Pass | Quick re-run at 390px: no sideways scroll (`scrollWidth` 388 = `clientWidth`). The mountain picture ends at x=355. "Parking" is 18px against 22px for the set title. Nothing has `script`, `onerror` or `onclick`, the title is unchanged, and i_bad shows words only. |
| T-11 | Pass | v1 published with no problems listed. |
| T-18 | Pass | Validate shows the blocks formatted and cleaned, the numbered lists keep their numbers, and "café £5 it’s" is correct. There were no console errors, and every API call returned 200. |

## Failures

None.

## Found while exploring

1. **A clinician's own edit in Validate is also plain text (works).** In Validate I appended ` {q_lift} **x** [Clin](https://evil.example/c)` to the clinician's copy of q_note. The block re-rendered with all of it as plain text in the bold: `{q_lift}` wasn't filled in a second time, `**x**` stayed literal, and nothing was linked. After a reload the edit was still there, and there were still no evil links.
2. **A placeholder inside a link's address is left as written (safe).**
   - **Information block:** with `[link {q_note}](https://example.org/{q_note})`, the link's words are filled with the answer as plain text. The `href` stays literally `https://example.org/{q_note}`, and the `<a>` has only `href`, `target` and `rel`. An answer full of quotes and `onmouseover=` added no attributes.
   - **Old statement:** with `<a href="https://example.org/{q_note}">` and `<img src="…/{q_note}.png" alt="{q_note}">`, the `href`, `src` and `alt` all stay literally `{q_note}`. The text `T:{q_note}` is filled and escaped.
   - This matches the fix advice: values in `href`/`src` are left out. The side effect is that an author who puts a placeholder in a picture's `alt` sees the literal `{q_note}`. That case is unlikely, and I'm noting it without calling it a bug.
3. **The orange note now covers links inside bold** (verify nit 1). `**[Form](http://example.org)**` shows as text, and the note lists `http://example.org`.
4. **Numbering edge cases follow the usual list rules.**
   - A numbered item, a plain line, then `2. Second step` resumes at 2.
   - `007. Leading zeros` straight after it continues the list as **3**, not 7. Standard behaviour: later numbers in a running list are ignored.
   - `99999999999. Huge` (more than 9 digits) is plain text.
5. **Typed entities in a new Information block show as typed.** `&eacute;`, `&amp;`, `&foo;` and `&#60;b&#62;` appear literally in the Preview. That's correct for text an author types; only old statements decode them.
6. **Cosmetic (probably not S35):** in Validate, the orange "Patient answered: …" note under a changed long text answer is one unbroken line. With a long answer it runs about 200px past the right edge of the question card, though not off the page (`scrollWidth` = `clientWidth`).
7. **Cosmetic, carried over from round 2:** on the editor card, i_bad shows "bad link" and "Click me" closer together than "Imported text" and "bad link".
8. **Cosmetic:** a blank line between numbered steps gives a much bigger gap in the Settings Preview (about 80px) than on the card (about 60px). Both read fine.

## Console and network errors

- No console errors or warnings on the Validate page after a reload. I saw no failed API calls. The three API writes I made (`i_ent`, `i_oldecho`, `i_oldlink`) and the setup script's four calls all returned 2xx.

## GIFs

- `~/Downloads/S35-T-23-retest3.gif`: the main happy path. Patient at 390px: start the set, answer with the link/picture/heading payload, finish and send; then Cara's Validate, with the clinician's extra edit.

## Hand checks for Rahul

- None new. The build notes' existing ones still stand: an HQ imported from an incumbent export, and a real picture on a real phone.
