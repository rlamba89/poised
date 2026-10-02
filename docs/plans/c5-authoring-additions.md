# C5: Authoring additions

**Goal:** clinician capture blocks, several Clinical Summaries per page, and Video and Information elements, with a content library.

**Requirements:** CAP-01/02, SUM-01, INF-01…05, PX-10, PX-13 (content), A-6, A-7. **Depends on:** C3. It can run in parallel with C4, because the code areas differ.

## Design

- **Capture (CAP):**
  - A `capture: true` flag on a Group or Section.
  - When saving, `packages/clinical` sets `clinicianOnly` on every child, so Go's patient strip doesn't change.
  - The clinician view shows a tinted capture box, and the POA Summary lists its answers under its title.
- **Summaries (SUM-01):**
  - A new element kind, `clinicalSummary`, which **replaces the page toggle**.
  - Its scope is its parent Group, Section or page. It has its own title and its own clinician comment.
  - The POA Summary shows one block per summary.
  - Update SGN-04 in [requirements.md](../requirements.md), which currently blocks more than one summary per page.
- **Video (INF-01/02):**
  - A YouTube or Vimeo URL, checked against an allowlist of URL patterns.
  - Embedded through `youtube-nocookie.com`, or Vimeo with `dnt=1`.
  - An optional, possibly required, "I have watched this" tick.
  - The picker searches the content library by tag.
- **Information (INF-03):**
  - Rich text with links and images.
  - **Decide at the start of C5:** Mantine's rich-text editor (Tiptap) or Markdown with a preview. Leaning: Markdown, which needs fewer dependencies and is easy to sanitise.
  - The HTML is **sanitised** before display.
- **Content library (INF-05):**
  - `content_items`: org_id NULL (platform), kind video|leaflet, title, url, length, tags, status.
  - Screens for the platform admin and trust admins.
- **Estimated time (PX-10):** worked out from the question count. This is a unit-tested helper.

## API

- `GET/POST/PATCH /api/admin/content`
- `/api/o/{oid}/content`
- `GET /api/o/{oid}/content/search?tag=`

## Tests

- **Unit (written TDD in `packages/clinical`):**
  - capture propagation;
  - summary scoping (which disclosures each summary lists);
  - the video URL allowlist and the embed URL builder;
  - the sanitiser;
  - estimated time.
- **Integration:**
  - the patient JSON has no capture blocks;
  - content library tenancy (platform items visible to all; a trust's items visible only to it);
  - only admins can manage content.
- **End-to-end:** **J10**. An author adds a capture Section, two summaries and a library video, and publishes. The patient (at phone size) sees the video but not the capture block. The clinician fills in the capture block. The POA Summary shows both summaries.

## Status

Not started.
