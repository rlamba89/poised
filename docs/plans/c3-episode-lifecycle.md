# C3: Episode lifecycle

**Goal:** the whole journey from invite to archive. This covers procedures, every status, worklists, tasks, cancelling, archiving, and auto-archive.

**Requirements:** EPI-01…09, PRC-01, TSK-01, the journey in §5. **Depends on:** C2.

## Data

- `procedures`: org_id, `hospital_id NULL`, name, code, specialty, default_version_id, info content (used from C5), PROMs plan (used from C9).
- **`episode_forms`**: episode_id, version_id, kind hq|assessment|proms, due_at, closes_at, status, submitted_at. `episode_answers` moves from per-episode to **per form**. Because it's a fresh app, this is a reshape, not a migration.
- `episodes`: + procedure_id, planned_date, poa_date, admitted_at, discharged_at, cancel_reason, `archived_from`, and the new status keys.
- `tasks`: episode_id, title, created_by (user or rule), role, done_by, done_at.

## Design

- **Transitions:** one table in Go, `allowed[from] → []to`, plus who may make each move. It's unit-tested exhaustively (written TDD).
  - Each move writes an event with who, from, to and the reason.
  - Automatic moves name their rule.
- **Worklists (EPI-02):** saved filters over one indexed query, `GET …/episodes?worklist=…`.
- **Archive:**
  - `archived_from` stores the previous status, and un-archive restores it.
  - **Auto-archive** is a daily job, with the hospital's N/M days from effective settings.
- **Re-open (EPI-07):** for super clinicians, with a reason. It clears the review stamp; the answers stay.
- **Status messages to the patient (EPI-08):** a hospital template per status, sent through the C2 messaging. It's a Should, so cut it if the plan runs long.

## API

- `GET/POST/PATCH /api/o/{oid}/procedures`
- `PATCH …/episodes/{eid}/status` (with a reason)
- `POST …/episodes/{eid}/cancel`, `…/archive`, `…/unarchive`, `…/reopen`
- `GET/POST/PATCH …/episodes/{eid}/tasks`
- `GET …/episodes?worklist=`

## Tests

- **Integration:**
  - every illegal transition → 409;
  - a role without the right → 403;
  - cancel needs a reason;
  - an archived episode is read-only (writes → 409);
  - un-archive restores the exact previous status;
  - each worklist returns exactly its episodes;
  - picking a procedure fills the default HQ.
- **Unit:** the transition table, and the auto-archive decision function.
- **End-to-end:**
  - **J8:** a full lifecycle. Create with a procedure → patient submits → review → Ready for admission → admitted → discharged → archive → un-archive.
  - **J8b:** cancel with a reason.

## Status

Not started.
