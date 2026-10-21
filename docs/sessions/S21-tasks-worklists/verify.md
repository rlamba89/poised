# S21 verification

## Verification 1 (21 Oct 2026, automatic run)

**Verdict: Verified.** Nothing blocks. The findings below are small; the ones worth doing later are in pending.md §3 ("From S21's verification").

Reviewed: the whole working tree against `de4da42` (30 changed files, 19 new), the brief, answers.md, the build notes, the test plan and test report 1 (26/26 passed). `/code-review high` was run and its findings weighed below.

### Findings (none blocking; worst first)

1. **The default "Cancelled" message only fits an open HQ.** Cancel is allowed from every status (`apps/api/internal/episode/episode.go:102`) and `StatusMessage` sends on any cancel (`apps/api/internal/episode/tasks.go:93`), but the default wording says "…has cancelled your pre-operative health questionnaire, so you don't need to fill it in" (`apps/api/internal/settings/settings.go`, `templates.cancelledSms` / `cancelledEmail`). A patient who already sent the HQ, or one at Ready for admission or Admitted, would get a message that doesn't fit. It's off by default and hospitals can reword it, so it doesn't block. Options: (a) neutral default wording ("…has cancelled your pre-operative assessment…"), recommended; (b) send the Cancelled message only while the HQ is open.
2. **Ready for admission is sent again each time it's picked.** On hold → Ready for admission, or Undo then pick again, sends a second "you are ready" message (test report 1 T-16 saw this). It's a reasonable reading of answer 4, because the status was reached again. Listed so Rahul knows.
3. **The General notes line after a status message uses the request's context** (`apps/api/internal/httpapi/statuses.go` `statusMessage` → `s.event`, `episodes.go:397`), while the send itself uses `context.WithoutCancel`. If the browser disconnects during a slow send, the message goes and `episode_messages` records it, but the "… message sent" note can be lost (only logged). The fix is one line: pass the uncancelled context to the note.
4. **`CountWorklists` skips some of `ListEpisodes`' inner joins** (`apps/api/db/queries/episodes.sql`: patients, the HQ `episode_forms` row, version, questionnaire). Since S18, every episode has its HQ form, so counts and lists can't differ today. A later episode without an HQ form would be counted but not listed. `TestWorklists` would catch it only if such an episode were seeded.
5. **The status email subjects are fixed in Go** (`apps/api/internal/invite/invite.go` `SendStatus`): any status other than Ready for admission gets the "cancelled" subject. Invites' subjects are fixed in the same way, and only two statuses send, so this is fine today. A third status message would need its own subject.
6. **Smaller points:**
   - The episode list's "N open tasks" also shows on cancelled rows (already in pending.md from the test report).
   - Task times use the database's `now()`, not the app clock. That matches the other stamps (`updated_at`, `undone_at`, `review_completed_at`), and no rule depends on a task's time.
   - `EpisodeList.load` fetches the counts again when the status filter changes, and a `/worklists` error can be cleared by the rows request's success (`EpisodeList.tsx:48`).
   - `episode.TeamLabels` is unused in Go. The note says "the anaesthetist team" (the key, lower case), which reads fine.

### Checks

- **Scope:** everything in the brief is done:
  - tasks (table, the three routes, the panel with who and when);
  - the six worklists (one SQL rule, `in_worklist`, used by both the list and the counts);
  - tabs with counts beside All, kept in the URL;
  - `worklists.soonDays` per hospital (default 14);
  - the four status-message switches and templates, off by default, `{{link}}` optional only for them;
  - `episode_messages` kind `status`, with the undelivered split;
  - the journey.

  The one route the brief didn't list, `GET …/worklists` (and the trust's), is small and needed for the counts. It's recorded in the plan's differences.
- **Tests first:** every new route has the five cases:
  - tasks: `tasks_integration_test.go`;
  - worklists: `worklists_integration_test.go`;
  - status messages: `status_messages_integration_test.go`, through the status routes.

  "Wrong role" is a clinician at the trust's other hospital, as in S20, since anyone at the hospital may change tasks. The worklist tests seed episodes just inside and just outside each rule: day 0 and day 14 in, day 15 and yesterday out, at UTC+14 and UTC−11. Unit tests: `episode/tasks_test.go`, `settings_test.go`, `delivery_test.go`, `tasks.test.ts`, `undelivered.test.ts`. Journey: `e2e/journeys/test_tasks_worklists.py`.
- **Tenant isolation:**
  - Every task route goes through `requireHospital` and `loadEpisode`; `GetTask` is scoped by episode.
  - The worklist queries keep `org_id`, the hospital list, and `all_hospitals` only for trust-wide members.
  - Checked live (below).
- **Audit:** the three task routes are wrapped in `audited` (`router.go`), and the rows were checked in `audit_log`. The counts routes aren't audited: they return no names or IDs. The lists still are.
- **A-20:** the grep finds only comments in `format.ts` and `patient.go`, as before. Today's date and the time zone in `in_worklist` come from the hospital or `cfg.Country.TimeZone`.
- **Security:**
  - no secrets in the repo;
  - the logs carry episode IDs only;
  - status messages hold only the first name and the hospital, as invites do, and their `{{link}}` is the plain sign-in page, not a token;
  - the patient router is unchanged; `chapter.ForPatient` is untouched.
- **Docs:** the plan's status and differences, the plans README, the sessions README, decisions.md (21 Oct S21 section), pending.md and `e2e/README.md` are updated. The Lifebox org-name grep prints nothing.

### Spot-checks (Claude in Chrome, "trk" profile, http://localhost:3000, as Cara Clinician, Hospital A1)

- **Every tab's count equals its list's length** on the live data: Ready for review 57/57, Not responding 218/218, Anaesthetic review 2/2, Procedure within 14 days 24/24, On hold 1/1, Open tasks 4/4. Her "All hospitals" counts are the same and its lists hold only Hospital A1. `?worklist=nonsense` → 400 "Choose a worklist.".
- **Other hospital and other trust:** for Cara, Hospital A2's worklists and lists → 404. An A2 episode's tasks through the A1 path → 404, and a POST to it → 404. Another trust's episode's tasks → 404.
- **The task lifecycle by API, then on the page:**
  - Adding an anaesthetist task → 201: Open tasks went from 4 to 5 and Anaesthetic review from 2 to 3.
  - A second add with the old revision → 409 `stale`. An unknown team → 400.
  - Completing it → 204, and both counts went back. Completing it again → 409 "already done".
  - A task on a cancelled episode → 409.
  - General notes had "Task added for the anaesthetist team: …" and "Task completed: …".
  - The panel showed the task struck through, the Done badge, "Added by / Completed by Cara Clinician", and Re-open.
- **Status-message settings:** Trust A and Hospital A1 have no `statusMessages.*` keys stored (off, as the tester left them). I didn't switch them on: the tester and `TestStatusMessages` cover sending.

### Test results

- `make lint`: passed.
- `make test`: Go all ok; Vitest 197 passed (20 files) + 116 passed (15 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 12.9 s; apitest incl. `TestTasksWorklistsMigration`).
- `make e2e` (my own API on :8093 with database `sj_v21`, Vite on :3103; both stopped and the database dropped afterwards): **55 passed, 5 skipped** (QA-only Cognito journeys), `test_tasks_worklists.py` included.

### Hand checks for Rahul

- **T-20 at a real phone size:** open a status message's `{{link}}` (the sign-in page) in a real 390×844 window. The tester could only use an iframe of that size.

### Suggested commits

1. **S21: tasks, worklists and status messages**: migration 00022 and its test, the queries and generated `internal/db`, `internal/episode` tasks/worklists/status messages, settings, `invite.SendStatus`, `delivery.Note`, the routes and their integration tests, the Tasks panel, the worklist tabs, the undelivered split, the journey, `apps/web/AGENTS.md`, `e2e/README.md`.
2. **docs: S21 verified**: the S21 session docs, decisions, pending, plan and session status, plus the conductor's S20 edits (answers.md answer 7, and the matching "Settled 21 Oct" sentence in decisions.md's S20 entry).

Next: `/build-session S22`.
