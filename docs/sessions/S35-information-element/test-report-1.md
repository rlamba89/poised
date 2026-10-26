# S35 test report 1

**Summary: 23 passed, 1 failed, 0 blocked** (24 cases: T-01 to T-19, R-01 to R-05).

- **Environment:** HEAD `f6a336c`, working tree not clean (the uncommitted S35 build, plus `docs/market/`). Sunday 25 Oct 2026, 22:16 to about 22:45 GMT. http://localhost:3000 (web) and :8080 (API), started by the conductor (`make dev`, freshly migrated and seeded).
- **Browser:** Claude in Chrome ("trk" profile), desktop window 1512×771. Patient cases ran in a same-origin 390×844 iframe (390×760 for R-04).
- **Test data:** questionnaire **S35 test HQ MT 2216** (v1 published, v2 draft). Episode **Mt S35Info Smith** (`f5255406-87c8-4f30-839d-0ce9f43c128a`, S35 HQ v1, answers sent). Episode **Mt S35Demo Jones** (Demo pre-op HQ v5, started but not sent).
- **One change to the steps:** as the conductor asked, T-06 used payloads that would set `window.__xss` or change `document.title` instead of `alert(1)`: `<script>window.__xss=5;document.title='XSS5'</script>`, `[x](javascript:window.__xss=6)` and `<img src=x onerror="window.__xss=7;document.title='XSS7'">`. No browser dialogs were triggered.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 | Pass | The Add bar has **Information** with an "i" icon. "Statement" appears nowhere on the page. |
| T-02 | Pass | The card shows two plain lines. The header says Information. The Text box holds exactly the two lines (no `<p>`, `&` not `&amp;`). How to format, Preview and the Clinical switch are all present. |
| T-03 | Pass | The Preview showed **Thank you** in bold while typing, and the card did too after Tab. "All changes saved". After a reload the Text box holds the three lines unchanged. |
| T-04 | Pass | i_bad shows "Imported text", "bad link" (not a link) and "Click me". No `<img>`, no `javascript:` href, no `onclick`/`onerror` attributes and no script in the DOM. `window.__xss` is undefined. |
| T-05 | Pass | The live Preview and the card both show an h2 "Fasting", bold and italic, a bullet list and a numbered list, two links (`https:`, `mailto:`, `target=_blank`, `rel=noopener noreferrer`), and the picture (437px in a 461px preview box). "All changes saved". |
| T-06 | Pass | All three lines show as typed, as plain text, in the Preview and on the card. The orange note says "Shown as text, not as a link or picture: javascript:window.__xss=6. A link must start with https://, mailto: or tel:, and a picture with https://." `__xss` stayed undefined and the title stayed "Preopsis". No console errors. (`cmd+End` put the caret mid-text, so the lines landed after "2. Remove nail varnish". That changes nothing the case checks.) |
| T-07 | Pass | `[plain web](http://…)` and `![old picture](http://…)` show as text. The note lists `http://example.org` and `http://example.org/a.png`. |
| T-08 | Pass | Undo brought back the text from before T-07, on the card and in the Text box. With Clinical on, the card shows the CLINICIAN marker; turning it off removes the marker. (I typed the T-07 lines again afterwards for the patient cases.) |
| T-09 | Pass | Translate shows the old statement in English as plain lines. "Parking" already has German. The English column keeps the marks (`**car park B**`). The German Fasting text saved. In the Preview in German: "Parken" (heading), **Parkhaus B**, "Nüchtern" (heading), **essen**, and the old statement still in English. Back in English, everything is English. |
| T-10 | Pass | Desktop Preview (640px blocks): all blocks are formatted, pictures are exactly 640px (no wider), and links are teal `rgb(17,94,89)` and underlined. i_bad shows only its words. `__xss` is undefined. The Phone preview fits too. (Heading size: see F1.) |
| T-11 | Pass | "V1 · PUBLISHED / READ ONLY". No problems were listed. |
| T-12 | Pass | Sue Super on published v1 and on draft v2: the Text box is disabled, the Preview shows, the Clinical switch is greyed, there is no Add bar or Publish button, and the header says READ ONLY. |
| T-13 | **Fail (Minor)** | Everything listed is correct at 390px: the old lines plus bold Thank you, i_bad's words only, Parking, Fasting, and the T-06/T-07 lines as text. Both pictures end at x=355 in a 388px page, and there is no sideways scroll (`scrollWidth` 388). **But the headings are far too big:** "Fasting" (h2) is 48px and "Parking" (h3) is 32px, against 22px for the set title and about 21px/18px in the editor's preview. See F1. |
| T-14 | Pass | "the map" and "Our fasting page" each opened a new tab (example.org) and the form stayed open. The hrefs are `tel:+10000000000` and `mailto:poa@example.org`. |
| T-15 | Pass | Tapping "Click me" and "bad link" did nothing. `__xss` is undefined in the frame and in the parent. |
| T-16 | Pass | I answered Yes, finished the section, sent and confirmed (an in-app modal), and got "Thank you, Mt". In the DB, `episode_answers.data` = `{"q_lift":"o_yes"}` only, so the Information blocks have no answer keys (checked with psql). |
| T-17 | Pass | After a reload, and after going back to All sections and opening the set again, the blocks rendered with the same tags as before (h3/h2, lists, links, img). |
| T-18 | Pass | Validate shows the same blocks formatted, with i_bad cleaned and `__xss` undefined. The POA Summary doesn't list the Information blocks. (Heading size is the same as F1: 48px/32px.) |
| T-19 | Pass | Bea's questionnaires list (UI and API) doesn't include the S35 HQ. The direct API GET is 404. Val sees "You don't have access to this hospital" for the episode URL; the API is 404 under both A1 and A2 paths. |
| R-01 | Pass | A Clinical Summary inside a Group says 'Covers the Group "Group"'. Validating the J10 episode, the summaries list their notes as bullets ("Asthma", "Angina"). |
| R-02 | Pass | The library video card shows "YouTube · Length 10:35". The Preview shows the youtube-nocookie player. |
| R-03 | Pass | The Preview shows Name Sam Sample, Age 54 and Sex female in bold labels, with the note below. |
| R-04 | Pass | A Cara episode on the Demo pre-op HQ v5 (picked by choosing a procedure). The patient at 390px answered the smoking and alcohol questions, saved with PUT 200, with no sideways scroll. |
| R-05 | Pass | The German title "Fährt Sie jemand nach Hause?" shows in the Preview in German. |

## Failures

### F1 (Minor): Information headings are oversized for patients, in Preview and in Validate, and don't match the editor

**Steps** (Cara made the episode; the patient is at 390px):
1. As Alex, add an Information block with `# Fasting`, or use the S35 test HQ ("## Parking").
2. Look at the editor's Settings Preview and card: the heading is about 21px (h2) or 18px (h3).
3. Open the same set as the patient at 390×844, or in the chapter Preview, or in Validate.

**Expected:** the heading is about the same size as in the author's preview (the brief says "the preview and the patient view (phone size) match"), and no bigger than the set title.

**Actual:**
- `# Fasting` → h2 at **48px**, and `## Parking` → h3 at **32px**.
- The set title "Before you come in" is 22px, so a single `#` heading at phone size is more than twice the page title and dominates the screen.
- Screenshot: `/private/tmp/claude-501/-Users-rahul-poised/8735e5a4-9e31-4654-a7d9-6d69c8a4b98f/scratchpad/S35-patient-heading-size.png`.

**Likely cause** (for the builder):
- `globals.css` sets `.sj-info h2 { font-size: 1.3em }` and `.sj-info h3 { 1.15em }`.
- SurveyJS's own `.sd-html h2 { font-size: var(--sjs2-font-size-x600) }` and `.sd-html h3 { …x400 }` from `survey-core.min.css` have the same specificity and win. They also set line-height 56px and 40px.

**URL / user:**
- Patient: `/home/episodes/f5255406-87c8-4f30-839d-0ce9f43c128a` (patient Mt S35Info Smith).
- Also `/o/…/chapters/<id>/preview` as Alex, and `/o/…/episodes/f5255406…/sets/e0a3c194…` as Cara.

**Console / network:** none.

## Found while exploring

1. **Minor: editing a hand-made HTML statement can run its words together.**
   - i_bad's Text box (and the Translate English column) shows `Imported textbad linkClick me`, while the card shows three lines.
   - I added "." and left the box (v2 draft). The card now shows "Imported textbad linkClick me." on one line, so the spacing is lost for good.
   - This only affects HTML that isn't one `<p>` per line, which nobody can type and the incumbent import doesn't produce.
   - An import-style statement with `<p>&nbsp;</p>` (added by API to v2 as `i_imp`) opens correctly, with "Line A", a blank line, and "Line B <tag> & more".
2. **Cosmetic: Translate overflows.** In the English column, long unbroken text (the Wikimedia URL, `onerror="window.__xss=7;…"`) runs past the column under or into the German box.
3. **Cosmetic: blank lines.** An import-style statement with a blank line (`<p>&nbsp;</p>`) shows the gap on the card but not in the Settings Preview, which collapses it.
4. **Cosmetic, probably not S35:** the patient sections list at 390px truncates the status badge to "COMPL…" and clips the "Review" button text to "Revie".
5. **Probably not S35:** reloading inside a set returns the patient to the sections list (the URL has no set in it).
6. **Not confirmed:** I clicked Preview right after adding a Profile block, while the header may still have said "Saving…". That Preview didn't show the new Profile; reopening it did. It may be a Preview-before-save race.
7. **A question rather than a bug:** the editor's page header says "Clinical Page" above "Your visit", although the set is for Patients.
8. **Works:** the Tab key from the Information Text box moves focus into the Preview's links, and the focus ring is visible. New version (v2) copied the Information blocks, the markdown and the German translations intact.

## Console and network errors

- No console errors or warnings were captured on the editor, the patient or validate pages.
- Every API call I saw returned 2xx, except the expected 404s in T-19.

## Hand checks for Rahul

None. No case needed a browser dialog, a download or printing.

## GIFs and screenshots

- `~/Downloads/S35-T05-information-editor.gif`: the editor happy path, T-05 to T-08 (adding the block, the live preview, the unsafe lines, Undo and Clinical).
- `/private/tmp/claude-501/-Users-rahul-poised/8735e5a4-9e31-4654-a7d9-6d69c8a4b98f/scratchpad/S35-patient-heading-size.png`: F1 at 390px.
