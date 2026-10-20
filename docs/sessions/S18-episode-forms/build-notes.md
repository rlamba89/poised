# S18 build notes: episode forms

Builder, automatic run (`/run-sessions`), 20 Oct 2026.

## Answers (from [answers.md](answers.md), asked by the conductor on 20 Oct)

1. **`episode_forms` columns:** the plan's full set: id, episode_id, version_id, kind (`hq|assessment|proms`), due_at, closes_at, status, submitted_at.
2. **Existing data:** (b) reshape the tables and re-seed, no data migration.
3. **Routes:** keep today's URLs; they resolve the episode's one HQ form on the server.

## What was built

- **Migration `00019_episode_forms.sql`:**
  - removes every episode, with everything of theirs: answers, General notes, messages and follow-up jobs (by `ON DELETE CASCADE`), and the invite links with the patient sessions opened from them. Patient records, accounts, sign-in links, staff and questionnaires are kept;
  - creates **`episode_forms`**: id, episode_id, version_id, kind (`hq|assessment|proms`), status (`open|submitted`), due_at, closes_at, submitted_at, submitted_by, created_at. A CHECK keeps `status` and `submitted_at` in step; a unique partial index allows **one HQ per episode**;
  - recreates **`episode_answers`** keyed by `(episode_form_id, chapter_id, actor)`, with the S02a `revision` as before;
  - drops `episodes.version_id`, `patient_submitted_at` and `patient_submitted_by`;
  - its Down puts back the old shape (again with no episodes).
- **Queries** (`db/queries/*.sql`, then `make sqlc`): every one that read the HQ version or the answers now goes through the HQ form (`JOIN episode_forms f ON f.episode_id = e.id AND f.kind = 'hq'`):
  - `episodes.sql`: `CreateEpisode` (the episode and its HQ form in one statement, a data-modifying CTE), `ListEpisodes`, `GetEpisode` (adds `hq_form_id`), `GetEpisodeForm` (adds `form_id`), `ListEpisodeAnswers`, `SavePatientAnswers`, `SaveClinicianAnswers` and `SubmitPatientHQ` (now keyed by the form id; the submit sets the form `submitted` and the episode Ready for review in one statement);
  - `accounts.sql` `ListHome` (the patient home and its "started"), `patients.sql` `ListPatientEpisodesHere`, `jobs.sql` `GetFollowup`, `CountReminder`, `MarkNotResponding`, and `links.sql` `GetInviteDetails`.
- **Go handlers:** `patient.go` and `validation.go` pass the form id instead of the episode id to the answers queries. Nothing else changed: the moved columns keep their old names through SQL aliases (`f.submitted_at AS patient_submitted_at`), so the Go fields and **the JSON are unchanged**, apart from one added field, `hqFormId`, on the staff episode.
- **Test harness:** `apitest.Fixtures.EpisodeOf` creates the HQ form too, and `apitest.Episode` has `FormID`. New `apitest.NewMigrations` (`internal/apitest/migrations.go`): an empty database migrated only up to a given version, for testing a migration against rows in the older shape.
- **No web or TypeScript change**, and no change to the e2e journeys.

## Decided by the builder (automatic mode)

- **"Reshape and re-seed" is done by the migration itself**, not by editing migration 00004 and dropping every database by hand. Same result (no old episodes, nothing migrated), but it runs everywhere unattended, including QA through the pipeline's migrate step, so **QA needs no hand reset**. The seed creates no episodes, so there's nothing to re-seed. A migration test proves it on a database in the old shape. (Recorded in decisions.md §2 and §5, replacing the "QA needs a reset" item.) **Know that the next push deletes every episode on QA.**
- **Form status values are `open` and `submitted`.** The plan named a status column without values. C7/C9 add more if they need them.
- **`patient_submitted_by` (PAT-09) moved to the form as `submitted_by`**, beside `submitted_at`; the brief named only the submit time, but leaving "who sent it" on the episode while "when" is on the form would split one fact across two tables.
- **`created_at` on the form** (not in the plan's list), for ordering an episode's forms once there are several.
- **`hqFormId` is added to the staff episode JSON** (`GET …/episodes/{eid}`). Adding a field breaks nothing; the web app ignores it.

## Differences from the plan or brief

- The brief's **migration test** was for answer (a). With (b) there is still one: `TestEpisodeFormsMigration` (`internal/apitest/migrations_integration_test.go`) migrates a database holding an old-shape episode (with answers, notes, a message, a job, an invite link and its session, plus a sign-in link and its session) and checks what's removed and what's kept, the new constraints (one HQ per episode, the kinds, status in step with submitted_at), cascade on delete, and that Down then Up work.
- Recorded in the plan: [c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md) Status.

## Tests (TDD)

Written first and seen failing (red), then passing:
- `TestEpisodeFormsMigration` (new): failed with "relation episode_forms does not exist" before 00019.
- `TestCreateEpisode`: new checks that the created episode has exactly one form, its open HQ on the chosen version with no dates, and that the refusals made no form.
- `TestSubmitPatientHQ`: the HQ form is `submitted` with `submitted_at`, and `submitted_by` NULL for the patient. `TestCompleteOnBehalf`: `submitted_by` is the staff member.
- `TestGetEpisode` (new subtest): `versionId`, `versionNo` and `patientSubmittedAt` in the episode's JSON come from the form, before and after the submit.
- Every other test that wrote or read answers or the submit directly in SQL now does so through the form (`episode_form_id`, `episode_forms.submitted_at`), unchanged in meaning: `episodes`, `validation`, `patient`, `behalf`, `invites`, `routers` integration tests.

### Results (20 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 163 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok (httpapi 12.1 s, jobs 12.8 s, apitest with the migration test).
- `make e2e` (against my own API on :8091 and Vite on :3011, `E2E_BASE_URL=http://localhost:3011`, so the conductor's ports stayed free): **43 passed, 5 skipped, 1 failed**; the failure, J7, then **passed on a re-run** (6 passed). It's a **pre-existing flake, not S18's**: J7 picks a random mobile among 1,000 (`07700 900000`–`999`) and expects exactly one captured text to it, but the local database holds 2,271 captured messages from earlier runs, so the number has often been used before. Suggested follow-up for pending.md: give J7 a mobile unlikely to repeat, or clear its captured messages first.

## Test report 1 (20 Oct): all passed

- Every case passed; no code changed.
- Four expectations in test-plan.md were wrong, and the app followed the agreed behaviour. They're corrected (note at the top of the plan):
  - T-01: the Hospital column shows only in "All hospitals".
  - T-04: the card has no "sent by the patient" line; the time is in General notes.
  - T-13: day 7 still marks "Not responding" once the patient has started (S16's rule).
  - R-3: the old link shows S14's "This link doesn't work…" wording.
- Blocked by the tester's tools: R-1's **Export CSV** (a file download), already covered by S11's tests. It's a hand check for Rahul if wanted.

## Hand checks for Rahul

- None new. (After the push: QA's episodes are gone, by design.)

## Known limits (minor, for the verifier)

- `CountReminder` and `MarkNotResponding` now check "the HQ is still open" through a subquery on `episode_forms` rather than a column of the row they update. If a submit commits at the very instant a reminder is being counted, Postgres's re-check of the updated row may not see the new form status, so in that race one extra reminder could go. Before S18 the check was on the same row and exact. The window is milliseconds and the cost is one surplus reminder; `GetFollowup` still reads the submit before deciding.

## Suggested commit

One plan step, one commit:

```
S18: episode forms — the HQ and its answers move from the episode to a form

Migration 00019 reshapes episode_answers onto episode_forms (kind hq|assessment|proms,
status, due/close dates, submitted_at/by) with no data migration: it removes the
made-up episodes (and their links and sessions) and keeps everything else. Every
query that read the HQ version or the answers goes through the episode's HQ form; the
routes and the JSON are unchanged (plus hqFormId). Tests: the migration test on an
old-shape database, the HQ form on create and submit, the JSON's moved fields.
```

Files: `apps/api/db/migrations/00019_episode_forms.sql`, `apps/api/db/queries/{episodes,accounts,patients,jobs,links}.sql`, `apps/api/internal/db/*` (generated), `apps/api/internal/httpapi/{patient,validation}.go`, `apps/api/internal/apitest/{fixtures,migrations}.go`, `apps/api/internal/apitest/migrations_integration_test.go`, the six updated `internal/httpapi/*_integration_test.go`, `CLAUDE.md`, `docs/decisions.md`, `docs/plans/README.md`, `docs/plans/c3-episode-lifecycle.md`, `docs/sessions/README.md`, `docs/sessions/S18-episode-forms/*`.
