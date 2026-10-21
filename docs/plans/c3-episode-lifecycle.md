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

**Split into sessions on 20 Oct 2026.** **S18 verified** (20 Oct; [build notes](../sessions/S18-episode-forms/build-notes.md), [verify.md](../sessions/S18-episode-forms/verify.md)). **S19 verified** (21 Oct; [build notes](../sessions/S19-procedures/build-notes.md), [verify.md](../sessions/S19-procedures/verify.md)). **S20 verified** (21 Oct; [build notes](../sessions/S20-statuses/build-notes.md), [verify.md](../sessions/S20-statuses/verify.md)). **S21 verified** (21 Oct; [build notes](../sessions/S21-tasks-worklists/build-notes.md), [verify.md](../sessions/S21-tasks-worklists/verify.md)). The briefs are in [docs/sessions/](../sessions/README.md):
- **S18:** episode forms (the reshape, first, so C4–C6 build on it);
- **S19:** procedures and episode dates;
- **S20:** every status, cancel and re-open;
- **S21:** tasks, worklists and status messages;
- **S22:** archive, auto-archive and closing a patient account.

Differences from this plan, agreed by Rahul on 20 Oct:
- **EPI-08 (status messages) is in**, in S21, rather than cut.
- **EPI-09 (a patient closes their account) is added**, in S22.
- **The end-to-end journeys are J9 and J9b**, because `test_j8_reminders` took J8.

Differences in S18 (episode forms, 20 Oct; details in its [build notes](../sessions/S18-episode-forms/build-notes.md)):
- **A form's status is `open` or `submitted`** for now; the database keeps it in step with `submitted_at`. C7 and C9 add any others they need.
- **Who sent the HQ for the patient** (`patient_submitted_by`, PAT-09) moved to the form too, as `submitted_by`.
- **The reshape is migration 00019,** which removes every made-up episode (with its answers, notes, messages, jobs and invite links) and keeps everything else. So QA needs no hand reset: the pipeline's migrate step does it.

Differences in S19 (procedures and episode dates, 20 Oct; details in its [build notes](../sessions/S19-procedures/build-notes.md)):
- **The default HQ is stored as a questionnaire (`default_questionnaire_id`), not a version** (the plan said `default_version_id`). A new episode gets that questionnaire's published version at the time, so publishing a new version doesn't strand the default; S10's `defaultHq` setting works the same way.
- **The info content (INF-04) and PROMs plan (PRM-01) columns are added now** (S19 answers 3) as empty JSON lists, returned by the API but not yet editable: C5 and C9 define their items.
- **The free-text procedure is carried across, not re-seeded:** migration 00020 makes each trust's distinct texts into its procedures and links the episodes (only those made since 00019).
- **The seed adds one published "Demo pre-op HQ" per trust**, so the demo procedures have a default HQ.
- **"Picking a procedure fills the default HQ" is tested on `GET …/procedures?hospital=`**, which gives each procedure the HQ it picks there; the New episode page applies it.

Differences in S20 (statuses, cancel and re-open, 21 Oct; details in its [build notes](../sessions/S20-statuses/build-notes.md)):
- **Undo, not a super clinician's step back** (S20 answers 2): any clinician steps back the latest move made by hand, with a reason, repeatably. It has its own route, `POST …/episodes/{eid}/undo`, beside the plan's status, cancel and re-open.
- **The status events are a new table, `episode_status_events`**, and every episode write carries a **revision** (a stale one → 409), which the plan didn't name.
- **Admitted and discharged are dates** (`admitted_at`, `discharged_at` as `date`), as entered.

Differences in S21 (tasks, worklists and status messages, 21 Oct; **tested 21 Oct, test report 1 all passed; verified 21 Oct**; details in its [build notes](../sessions/S21-tasks-worklists/build-notes.md)):
- **A task is for a team** (`team`: nurse, anaesthetist, admin; S21 answers 1), where the plan said `role`; `created_by_rule` names a rule's task (C6), and `reopened_by` / `reopened_at` keep who re-opened it. The PATCH names the task: `PATCH …/episodes/{eid}/tasks/{tid}`.
- **Each worklist's rule is one SQL function, `in_worklist`**, used by both `GET …/episodes?worklist=` and a new route for the tabs' counts, **`GET …/worklists`** (and the trust's `GET /api/o/{oid}/worklists`), which the plan didn't name.
- **A task change is an episode change:** it sends the episode's revision and bumps it (S20).
