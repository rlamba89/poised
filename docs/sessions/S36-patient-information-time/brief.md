# S36: The patient's Information, and the estimated time

**Plan:** C5 ([c5-authoring-additions.md](../../plans/c5-authoring-additions.md)), part 6 of 6.
**Requirements:** INF-04, PX-10 ([saas-requirements.md](../../saas-requirements.md) §7); PRC-01's information content.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S35 verified.

**Why last:** the estimated time counts video lengths (S34), and the Information list uses the content library (S33).

**What exists today:**
- `procedures.info_content` (S19, migration 00020) is an empty JSON list on every procedure, returned by the API but not yet editable (`ProceduresPage.tsx`).
- The hospital's patient phone and email are settings (`contact.phone`, `contact.email`, `internal/settings/settings.go`).
- The patient home (`features/patient/PatientHome.tsx`, `GET /api/p/home`) shows a card per episode and the patient's own contact details. The patient's HQ (`PatientHQ.tsx`) lists Question Sets as tiles.

**Ask Rahul first:**
1. **Where the patient sees an episode's Information** (videos, leaflets, the hospital's phone and email).
   - (a) **An Information section on the episode's card in the patient home**, opening a simple list.
   - (b) A separate Information page.

   **Recommended: (a).**
2. **When a trust changes a procedure's videos or leaflets,** do episodes already made see the change?
   - (a) **Yes.** An episode shows its procedure's current list, plus anything a clinician added to that episode.
   - (b) No. The list is copied onto the episode when it's made.

   **Recommended: (a).** Patients get the latest leaflets, and there's nothing to copy.
3. **How the estimated time is worked out** ("about 4 minutes").
   - (a) **About 15 seconds for each question the patient sees, plus each video's length,** rounded up to whole minutes (at least 1). Shown on each Question Set tile, and in the editor.
   - (b) The same, with longer times for free-text and repeating questions.

   **Recommended: (a).** Simple to explain; it can be tuned later.

## Build

- **PX-10, test-first:** an `estimatedMinutes` helper in `packages/clinical`, counting only what the patient gets (after `forPatient`, so clinician-only and capture blocks don't count). The patient's tiles and the editor's Question Set header show it.
- **The procedure's Information** (INF-04): admins (and whole-trust super clinicians, who manage procedures) pick videos and leaflets from the content library for each procedure; stored in `info_content` as library item IDs. Retired items stop showing.
- **A clinician adds items to one episode:** from the library, on the episode page. Stored on the episode (new column or table). Audited, since it's on a patient's episode.
- **The patient's view:** the Information section as answered, with videos embedded (S34's builder) and leaflets as links, and the hospital's phone and email.
- **Routes**, each with the five integration cases: setting a procedure's items (the existing PATCH), adding and removing an episode's items, and the patient's read (only their own episodes; another trust's items never show).
- **J10 or J7c extended:** a procedure with a video and a leaflet; the patient (phone size) sees them under Information, with the hospital's phone number, and the set tile says "about N minutes".

## Not in this session

- Leaflet uploads (C8). The patient's to-do list of required videos (PX-08's later part).
- Measuring real completion times (PX-12).

## Manual test focus

1. Give a procedure a video and a leaflet. A new episode for it: the patient sees both under Information, with the hospital's phone and email.
2. Add a leaflet to one episode only: that patient sees it, another patient with the same procedure doesn't.
3. Change the procedure's list: existing episodes follow it (as answered).
4. Retire a library item: it disappears from every patient's Information.
5. A set's tile says "about N minutes"; adding five questions raises it; clinician-only questions and capture blocks don't count; a video adds its length.
6. Trust B's patients never see Trust A's items.
