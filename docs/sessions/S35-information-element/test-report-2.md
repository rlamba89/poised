# S35 test report 2 (retest)

**Summary: 10 passed, 0 failed, 0 blocked** (the retest list: T-13, T-20, T-21, T-22, plus quick re-runs of T-02/T-03, T-09, T-15, T-16, T-18).

- **Environment:** HEAD `f6a336c`, working tree not clean (the uncommitted S35 build and its fixes, plus `docs/market/`). Sunday 25 Oct 2026, about 22:40 to 22:50 GMT. http://localhost:3000 (web) and :8080 (API), started by the conductor (`make dev`, freshly migrated and seeded).
- **Browser:** Claude in Chrome ("trk" profile), desktop window 1512×771. Patient cases ran in a same-origin 390×844 iframe.
- **Test data:** a new questionnaire **S35 test HQ MT 2245 retest**. v1 was made with the plan's setup script, then I added the "Fasting" block (T-05 to T-07 text) and a "Line one / blank / Line two" block in the editor. I also added an import-style statement `i_imp` (`<p>Line A</p><p>&nbsp;</p><p>Line B &lt;tag&gt; &amp; more</p>`) by API, to compare blank lines with an old statement. v1 is published; v2 is a draft. The episode is **Mt S35Retest Brown** (`b0fedeac-5bb9-4d2e-9dec-f0159eda7c40`, Hospital A1, S35 retest HQ v1, answers sent).
- **Same change to the steps as round 1:** the T-06 lines use `window.__xss` / `document.title` payloads instead of `alert(1)`. No browser dialogs were triggered.

## Results

| Case | Result | Note |
|---|---|---|
| T-13 | Pass | At 390px: `# Fasting` is an h2 at **20px** and `## Parking` an h3 at **18px**, against **22px** for the set title "Before you come in". The rest is as in round 1: the old lines, i_bad's words only, Parking (bold, bullets, links), and Fasting (bold, italics, both lists, two links, the T-06/T-07 lines as plain text). Both pictures end at x=355 in a 388px page; `scrollWidth` 388 (no sideways scroll). `__xss` undefined, title "Preopsis". No `script`, `onerror` or `onclick` in the blocks. |
| T-20 | Pass | "Parking" is 18px and "Fasting" 20px everywhere: the Settings Preview, the editor card, the chapter Preview (desktop), the patient form at 390px, and Validate. The set title is 22px in both the patient form and Validate. `### Small heading` (explored) is 16px in the Settings Preview and on the card. |
| T-21 | Pass | `Line one`, a blank line, then `Line two`: the live Preview and the card both show a gap, like the old statement `i_imp`, whose blank `<p>&nbsp;</p>` now shows the gap in the Settings Preview too (fixed since round 1). The chapter Preview, the patient form (published v1) and Validate all show the gap. |
| T-22 | Pass | i_bad's Text box shows "Imported text", "bad link" and "Click me" on three lines. Translate (German) shows the same three lines in the English column. Long text wraps inside the English column (the Wikimedia URL, and `<script>…</script>` / `onerror="…"`): nothing runs into the German box. On the v2 draft, adding "." to i_bad and leaving the box keeps three lines on the card ("Click me."), so round 1's run-together problem is gone. |
| T-02 / T-03 | Pass | On v2: i_old's Text box holds the two plain lines. Adding `**Thank you**` shows it bold in the Preview and on the card. |
| T-09 | Pass | I typed `# Nüchtern` / `Nichts **essen** nach Mitternacht.` in Translate and it saved. In the Preview in German: "Parken" h3 at 18px, **Parkhaus B**, "Nüchtern" h2 at 20px, **essen**. The old statement and i_imp stay in English. |
| T-15 | Pass | Clicking "Click me" did nothing. "bad link" is plain text (not an `<a>`). `__xss` is undefined in both the frame and the parent. |
| T-16 | Pass | I answered the question, finished the section, sent and confirmed (in-app modal): "Thank you, Mt". |
| T-18 | Pass | Validate shows the blocks formatted and cleaned (no script, event handlers or `javascript:` links). `__xss` is undefined. The POA Summary lists none of the Information texts. |
| T-11 | Pass | v1 published with no problems listed. |

## Failures

None.

## Found while exploring

1. **Works:** three blank lines in a row give a bigger gap (three empty paragraphs) in the Preview and on the card. Leading spaces are dropped, an emoji shows fine, and `#NoSpace` stays plain text, not a heading.
2. **Cosmetic, carried over:** on the editor card, i_bad (hand-made HTML with a bare "bad link" between two `<p>`s) shows "bad link" and "Click me" closer together than "Imported text" and "bad link". The Settings Preview spaces all three evenly. This is only for HTML nobody can type, and it goes away once the block is edited (it is saved as three paragraphs).
3. **Tooling note, not an app bug:** clicking into the New questionnaire Name field or the New episode name fields opens the 1Password inline menu in this Chrome profile, which blocks the extension's clicks and typing on that tab. I worked around it by filling those fields with `form_input` in a fresh tab.
4. Round 1's notes 4 to 7 (the "COMPL…" badge and "Revie" button clipped at 390px, a reload inside a set returning to the sections list, "Clinical Page" above a patient set's page) still look the same. They aren't S35.

## Console and network errors

- No console errors or warnings on the editor, the patient form or the validate page.
- No failed API calls seen. The setup script's four calls and my one `i_imp` PUT all returned 2xx.

## Hand checks for Rahul

None.

## GIFs and screenshots

- `~/Downloads/S35-retest-T13-patient.gif`: the patient happy path at 390px (invite, date of birth, the formatted set, finish and send).
- `/private/tmp/claude-501/-Users-rahul-poised/8735e5a4-9e31-4654-a7d9-6d69c8a4b98f/scratchpad/S35-retest-patient-390.jpg`: the bottom of the patient set at 390px (the T-06/T-07 lines as text, and the Line one / Line two gap).
