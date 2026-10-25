# S32: The Clinical Summary element

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 2 of 6.
**Requirements:** SUM-01 ([saas-requirements.md](../../saas-requirements.md) §8); SGN-04 changed in [requirements.md](../../requirements.md).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S31 verified.

**What exists today:**
- A page has a **Clinical summary** switch in the editor (`Canvas.tsx`, `PageCanvas`), stored as `page.clinicalSummary`. So a page has at most one summary.
- `packages/clinical/src/summary.ts`: `withClinicalSummaries` adds a clinician-only panel `cs_<page>` at the end of each such page, with a notes box (`cs_notes_<page>`) and a comments box (`cs_comments_<page>`); `refreshClinicalSummaries` fills the notes with the page's disclosure notes. `clinicianView.ts` styles it yellow (`sj-summary`).
- The POA Summary (`poa.ts`, `PoaSummary.tsx`) shows one "<page> Summary" block per page with notes or comments. A page's notes show even with no summary box, so none are lost.
- Go's incumbent import (`internal/hqimport/convert.go`) turns the incumbent's `ClinicalPageSummary` into the page switch, losing where it sat on the page.

**Ask Rahul first:**
1. **A summary inside a Group, and another for the whole page.** Which notes does the page's summary list?
   - (a) **Only the notes no nearer summary lists.** Each note appears once, under the nearest summary.
   - (b) Every note on the page, so a note can appear twice (in the Group's summary and the page's).

   **Recommended: (a).** The POA Summary doesn't repeat itself.

## Build

- **`packages/clinical`, test-first:**
  - a new element kind, **Clinical Summary**: clinician-only, with a title ("<Group or page name> summary" by default, editable) and its own comments box. Its scope is its parent Group or Section, or the page if it sits on the page;
  - which notes each summary lists, as answered (unit tests: page level, Group level, nested Groups, a Section, notes outside any summary);
  - notes that no summary covers still show on the POA Summary, in a block named after the page, as today;
  - **old content keeps working:** a page with the old switch is read as a page-level summary at the end of the page, named `cs_<page>`, so its comments keep the answer key `cs_comments_<page>`. Published versions are never rewritten; drafts are converted when the editor opens them;
  - a publish problem (`publish.ts`) for two summaries with the same scope, and the editor doesn't offer a summary inside a repeating group.
- **Editor:** Clinical Summary in the Add bar; the page's Clinical summary switch goes. The card shows its title and which Group it covers.
- **Clinician view** (preview and Validate): each summary box sits where the author placed it, listing its notes, with its comments box.
- **POA Summary:** one block per summary, titled with its own title, with its notes and comments (S27's clickable links kept).
- **Go:** `hqimport` makes a summary element where the incumbent's `ClinicalPageSummary` sat, not the page switch. Unit test.
- **SGN-04** in requirements.md: replace "a page has more than one clinical summary" with "two clinical summaries cover the same Group, Section or page".
- **Integration test:** the patient's JSON never contains a summary element.
- **J10 extended:** the author adds two summaries (one for a Group, one for the page); the POA Summary shows both, each with its own notes and comment.

## Not in this session

- Videos, the content library and Information blocks (S33–S35).
- Changing which notes a disclosure makes (outputs are unchanged).

## Manual test focus

1. Add a summary inside a Group and one on the page. The clinician preview shows each box where it was placed; each lists only its own notes (as answered in question 1).
2. Two summaries in the same Group: Publish is refused, naming the set.
3. Open a draft that used the old page switch: it now shows a summary element at the end of the page. An episode on an old published version still shows its summary box and its saved comments.
4. Validate: type a comment in each summary. The POA Summary shows one block per summary, with its title, notes and comment.
5. A note on a page with no summary still shows on the POA Summary.
6. The patient never sees a summary, at any screen size.
