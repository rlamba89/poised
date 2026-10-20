# S18 verification

## Verification 1 (20 Oct 2026, verifier, automatic run)

**Verdict: Verified.**

What I checked:
- the whole working tree against the brief, the answers and plan C3's Data section;
- `/code-review high`, with each finding weighed below;
- every suite, including `make e2e` against the conductor's `make dev`;
- the dev database's invariants after the tester's and the journeys' runs, and the riskiest routes over real HTTP (see "Spot checks").

No finding blocks the session. They're all minor. The ones worth keeping are in pending.md ("From the S18 build" and "From the S18 verification").

### Scope

Everything in the brief, as answered, is built:
- **`episode_forms`** with the plan's full column set (answers 1), plus `submitted_by` and `created_at`; status `open|submitted`, kept in step with `submitted_at` by a CHECK; one HQ per episode by a unique partial index.
- **`episode_answers`** keyed by `(episode_form_id, chapter_id, actor)`, with the S02a revision.
- **Reshape, no data migration (answers 2):** migration 00019 deletes the made-up episodes and their invite links and sessions, and keeps patients, accounts, sign-in links and questionnaires.
- **Today's URLs kept (answers 3):** the routes resolve the HQ form on the server. The JSON is the same, plus `hqFormId`.
- **Every query that read the version, the submit or the answers** goes through the HQ form: `episodes.sql`, `accounts.sql` (`ListHome`), `patients.sql`, `jobs.sql`, `links.sql`. I grepped the Go, SQL, Python and TypeScript for `patient_submitted_*`, `e.version_id` and `episode_answers … episode_id`: nothing outside migrations ≤ 00018 and the migration test's old-shape fixture still uses them.
- **Nothing extra:** no web or TypeScript change, no new route, no form kind other than the HQ is created.

### Findings (worst first, none blocking)

1. **Pushing S18 breaks QA briefly, then empties its episodes.** `infra/pipeline_stack.py:143` deploys the Lambdas and only then migrates (`post=[db_web, e2e]`). 00019 isn't backward-compatible: it drops `episodes.version_id`, `patient_submitted_at` and `patient_submitted_by`. Between the deploy and the migrate step, every episode route and the jobs follow-up return 500 on QA. If the migrate step fails, that lasts until it's fixed. The migration then deletes every QA episode, by design (answers 2). This is QA only, with made-up data, so it's acceptable. **Know it before the push.** Expand-then-contract migrations, or migrating before the deploy, matter once there's a customer. Added to pending.md.
2. **The reminder race** (`apps/api/db/queries/jobs.sql:56-64`). It's already in pending.md (S18 build, item 2). I checked it, and it's real. Under READ COMMITTED, Postgres re-checks only the updated `episodes` row, not the `EXISTS` on `episode_forms`. But the cost is tiny, because `jobs.followup` (`internal/jobs/jobs.go:136-151`) already sent the reminder *after* `CountReminder` committed, outside any transaction. So a submit landing between the count and the send could already let one reminder go, before S18. S18 widens that window by a few milliseconds. `MarkNotResponding` racing a submit only stores a date. The web app hides the badge once the HQ is sent (`EpisodePage.tsx:58`), but a "Not responding" General note could be added. If wanted later, `… FOR SHARE` on the form row in the subquery makes both exact.
3. **A patient save racing a submit can change submitted answers** (`apps/api/db/queries/episodes.sql:130-138`). `f.status = 'open'` is checked only in the `INSERT … SELECT`'s snapshot, and the `ON CONFLICT DO UPDATE` path re-checks only the revision. This was the same before S18 (the check was on `episodes.patient_submitted_at`, also in the `SELECT` only), so it's not a regression. Noted in pending.md. `FOR SHARE` on the form row fixes it.
4. **Nothing in the database guarantees that every episode has its HQ form,** yet every read inner-joins it. An episode created without one (a future create path, an import) would vanish from lists and return 404, and `GetFollowup` would return no rows, so its reminders would stop silently. Today the only creates are `CreateEpisode` (one statement, a data-modifying CTE) and the test fixture, and the dev database holds 0 episodes without exactly one HQ form. Watch it in S19 (a procedure-based create) and C7/C9. Noted in pending.md.
5. **Comment nits:**
   - `apps/api/db/migrations/00019_episode_forms.sql:10` says "`make seed` puts back the demo data", and `migrations_integration_test.go:54` says "the seed puts back the demo data". The seed creates no episodes, as decisions.md says ("nothing to re-seed").
   - `apps/api/db/queries/episodes.sql:50-51` says "the episode's JSON is unchanged", but it gains `hqFormId`. The build notes and decisions.md say so correctly.
6. **Repetition (code review, not acted on):**
   - The HQ join is copied into about 9 queries. An `episode_hq` view could hold it once C7/C9 add form kinds.
   - `apitest/migrations.go` repeats `recreateTemplate`'s goose-provider setup.
   - Three tests hand-write the "submit" SQL.

   It's fine for one form kind. Reconsider in C7.

### Tests first

- `TestEpisodeFormsMigration` (new) runs on a real old-shape database at migration 18. It checks:
  - what's removed and what's kept;
  - the new constraints;
  - the cascade;
  - Down, then Up.
- `TestCreateEpisode`, `TestSubmitPatientHQ`, `TestCompleteOnBehalf` and `TestGetEpisode` gain form assertions. The other integration tests now write and read through the form, unchanged in meaning.
- No new route, so the five-case rule adds nothing. The existing routes keep their five cases, and they pass.

### Tenant isolation and security

- The queries keep their `org_id` / `hospital_id` scoping. The form is reached only through an episode that the middleware has already scoped (`GetEpisode … WHERE e.id = $1 AND e.hospital_id = $2`; the patient handler checks the account owns the episode). Then `ListEpisodeAnswers`, `SaveClinicianAnswers` and `SavePatientAnswers` take that episode's own `hq_form_id` / `form_id`, never one from the request.
- No secrets, no patient data in logs or messages, no change to the patient/staff router split or to `chapter.ForPatient`.
- The A-20 grep finds only comments that were already there (`apps/web/src/lib/format.ts`, `internal/patient/patient.go`). The org-name grep prints nothing.

### Spot checks

**Claude in Chrome couldn't reach the local servers this time.** Every `localhost`, `[::1]` and `tester.localhost` page, :8080 included, gave `ERR_CONNECTION_REFUSED`, while the shell reached both. So I spot-checked over real HTTP through the Vite proxy (`[::1]:3000`) with the dev login, and in the database:
1. **Data invariants** (the dev database after the tester's and the journeys' runs, at goose version 19):
   - 0 episodes without exactly one HQ form;
   - episode and form status always agree (`hq_not_complete`↔`open`, and every later status↔`submitted`);
   - 0 forms on an unpublished version;
   - 0 answers on a chapter outside the form's version;
   - 0 forms on another trust's questionnaire.
2. **Staff episode JSON** (Cara, a Ready-for-POA A1 episode): `versionId`, `versionNo` and `patientSubmittedAt` are present and come from the form, and `hqFormId` equals the form row's id. `…/hq` returns both actors' answers.
3. **Other hospital, other trust:**
   - Val (A2) and Bea (Trust B) get 404 "Hospital not found." on that episode and its `/hq`;
   - signed out, the request gets 401.

The tester's report already covers the patient screens at phone width (390 px), the stale-save 409s and the concurrent submits.

### Test results (20 Oct)

- `make lint`: passed.
- `make test`: Go ok; Vitest 163 + 116 passed; CDK 124 passed.
- `make test-integration`: every package ok (httpapi 11.3 s, jobs 11.6 s, apitest with the migration test).
- `make e2e` (against the conductor's `make dev`): **43 passed, 5 skipped, 1 failed**. The failure was J8 (`test_j8_reminders.py:53`), and it **passed on a re-run**. It's the same pre-existing flake as the builder's J7: the journey picks one of 1,000 mobiles (`07700 900000`–`999`) and counts every text ever captured to it. Here 8 texts were found, including ids 633 and 2355 from earlier runs. Not S18's; pending.md's S18 item 1 now names J8 too.

### Hand checks for Rahul

- R-1: **Export CSV** on the Audit log as Ada (a file download, blocked for the tester).
- Optional: one patient screen at a real 390×844 window. The tester's window couldn't be resized, and my browser couldn't reach the app.
- **At push time:** QA's episodes are deleted by 00019, and there's a short window of 500s between the Lambda deploy and the migrate step (finding 1).

### Commits

1. `S18: episode forms — the HQ and its answers move from the episode to a form` (the migration, the queries, the generated `internal/db`, the handlers, the test harness and every test).
2. `docs: S18 verified — build notes, test plan and report, verification, decisions, plan and session status` (CLAUDE.md, decisions.md, pending.md, the plan and session READMEs, c3, `docs/sessions/S18-episode-forms/*` including answers.md).
