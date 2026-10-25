# S33: The content library

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 3 of 6.
**Requirements:** INF-05, PX-13 (content), A-6 ([saas-requirements.md](../../saas-requirements.md) §0, §7). Also the superadmin screen for the platform's starter questions that S24's answers moved to C5 ([S24 answers](../S24-question-library/answers.md), Q2).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S32 verified.

**Why before the video element:** the video picker (S34) searches this library.

**What exists today:**
- The platform admin's console (`pages/admin/AdminLayout.tsx`, `features/admin/`, routes `/api/admin/…` in `router.go`) has trusts and hospitals only.
- The question library (S24): `library_questions` with `org_id` NULL for platform items, routes in `internal/httpapi/library.go`, platform items seeded from `apps/api/cmd/seed/library.json`. Platform items can't be managed in the app.
- Trust admin pages sit in the hospital layout (Settings, Staff, Procedures, Audit; `apps/web/src/routes.tsx`).
- Videos are agreed to live on YouTube or Vimeo only, never on our servers (A-6).

**Ask Rahul first:**
1. **Starter videos.** We have no videos of our own yet.
   - (a) **Seed a few made-up example entries** that point to one public test video, marked as examples, so the picker can be tried. Real ones replace them before go-live.
   - (b) Start empty.

   **Recommended: (a).**
2. **Leaflets.** We can't store files until plan C8.
   - (a) **A leaflet is a link to a PDF on the web** (e.g. the trust's own site). Uploading comes with C8.
   - (b) Leave leaflets out until C8.

   **Recommended: (a).**
3. **Tags, and the PX-13 checklist** (fasting, what to bring, medicines, the anaesthetic explained, and so on).
   - (a) **Free tags, with the checklist topics offered as ready-made tags.** The library page shows which topics have nothing yet.
   - (b) Free tags only.

   **Recommended: (a).**
4. **The platform's starter questions** (S24 said a screen for them comes with C5).
   - (a) **A list on the superadmin's library page where a starter question can be retired** (no longer offered). New ones still come from the seed file.
   - (b) Leave it for later.

   **Recommended: (a).**

## Build

- **`content_items`** (next migration): id, `org_id` (NULL = platform), kind `video|leaflet`, title, url, length (seconds, videos only, typed by hand), tags, status `active|retired`, created_by, times.
- **The video link check, in Go, test-first:** only YouTube and Vimeo links are accepted (watch, short `youtu.be`, embed and `vimeo.com/<id>` forms, plus Vimeo's private-link form). A leaflet is an `https://` link. S34 adds the same check in `packages/clinical`; a test keeps the two lists in step (as `keys.textFormatNames` does with `fields.ts`).
- **Routes**, each with the five integration cases. No patient data, so not audited.
  - `GET/POST/PATCH /api/admin/content`: platform admins only.
  - `GET/POST/PATCH /api/o/{oid}/content`: the trust's own items. Trust admins (whole trust) change them; whole-trust super clinicians may read them.
  - `GET /api/o/{oid}/content/search?tag=&q=`: active platform items plus the trust's own, for authors (S34's picker).
  - Tenancy tests: platform items are seen by every trust; a trust's items only by that trust; Trust B can't read, change or find Trust A's.
- **Screens:**
  - superadmin: **Content library** in the console (`/admin/content`): videos and leaflets (add, edit, retire), and the starter questions (answer 4);
  - trust admin: **Content library** in the hospital layout: the trust's own items (add, edit, retire), with the platform items shown read-only, and the checklist (answer 3).
- **Seed:** the example items from answer 1 (made up, idempotent).

## Not in this session

- The Video element and its picker (S34); the patient's Information on the home (S36).
- Uploading leaflets or images (C8).
- A trust hiding platform items it doesn't want.

## Manual test focus

1. As Sam Superadmin, add a platform video with tags; a non-YouTube/Vimeo link is refused with a plain message. Retire it.
2. As Ada Admin, add a Trust A video and a leaflet. Bea (Trust B) never sees them; both trusts see the platform items.
3. Cara Clinician and Alex Author can't add or change items; the routes refuse them too.
4. Search by tag returns platform and own items, never retired ones or another trust's.
5. The checklist shows which topics have no item; adding an item with that tag ticks it.
6. Retire a starter question: the editor's Insert from library no longer offers it; questions already inserted are unchanged.
