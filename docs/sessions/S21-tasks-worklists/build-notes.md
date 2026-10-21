# S21 build notes: tasks, worklists and status messages

Builder, automatic run (`/run-sessions`), 21 Oct 2026.

## Answers (from [answers.md](answers.md), asked by the conductor on 20 Oct; also decisions.md §2)

1. **Who is a task assigned to?** A **team** from a fixed list: nurse, anaesthetist, admin. Anyone at the hospital may complete any task.
2. **Worklists:** the six fixed ones: Ready for review; Not responding; Anaesthetic review needed; Procedure within N days and not Ready for admission; On hold; Open tasks.
3. **"Within N days":** a hospital setting, default 14.
4. **Status messages (EPI-08):** Ready for admission and Cancelled, by text and email. Each template has a per-hospital on/off setting, **off by default**. `{{firstName}}` and `{{hospital}}`, with `{{link}}` optional.

## What was built

### Data (migration `00022_tasks_worklists.sql`)

- **`tasks`**: id, episode_id, title (1–200 characters), team (`nurse | anaesthetist | admin`), `created_by` (a user) or `created_by_rule` (a rule's name, for C6; exactly one of the two), created_at, `done_by` + `done_at` (set together on completing, cleared on re-opening), and `reopened_by` + `reopened_at` (the latest re-open, so the panel can say who re-opened it). Indexes: by episode, and the open tasks (`WHERE done_at IS NULL`, with the team).
- **`in_worklist(worklist, episode_id, status, not_responding_at, planned_date, today, soon_days)`**: a SQL function holding **each worklist's rule in one place**. The list's `?worklist=` filter and the tabs' counts both call it, so a tab's count is always its list's length.
- **`episode_messages`**: kind can now be `status`, with a new `status` column (the status it was sent for; required for that kind, refused for the others).
- An index on `episodes (hospital_id, status)` for the status-based worklists.

### The worklist rules (EPI-02, answers 2)

| Key | Tab | In it |
| --- | --- | --- |
| `ready_for_review` | Ready for review | status Ready for review |
| `not_responding` | Not responding | S16's flag (`not_responding_at`) while the status is still HQ not complete |
| `anaesthetic_review` | Anaesthetic review needed | an open task for the anaesthetist team |
| `procedure_soon` | Procedure within 14 days (the hospital's N) | a planned procedure date from today to today + N at the hospital, and not Ready for admission, Admitted, Discharged or Cancelled |
| `on_hold` | On hold | status On hold |
| `open_tasks` | Open tasks | any open task |

A cancelled episode is in none of them. "Today" is the date at the episode's hospital (its time zone, else the deployment's), and N is the hospital's `worklists.soonDays` (its own, else its trust's, else 14).

### Routes (each with the five integration cases)

- **Tasks** (`internal/httpapi/tasks.go`, each wrapped in `audited`: `task.list`, `task.create`, `task.update`):
  - `GET …/episodes/{eid}/tasks`: oldest first, with who added, completed and re-opened each;
  - `POST …/episodes/{eid}/tasks` `{title, team, revision}` → 201 `{id}`;
  - `PATCH …/episodes/{eid}/tasks/{tid}` `{done, revision}` → 204: complete (`true`) or re-open (`false`); already in that state → 409.
  - Each write: lock the episode, check the revision (stale → 409 `code: "stale"`) and that it isn't cancelled (409), change the task, **bump the episode's revision**, and write a General notes line ("Task added for the anaesthetist team: …", "Task completed: …", "Task re-opened: …").
- **Worklists** (`internal/httpapi/worklists.go`):
  - `GET …/episodes?worklist=<key>` (the hospital's list and the trust's "All hospitals" list): one query, scoped as before; an unknown key → 400 "Choose a worklist.". Each row now has `openTasks`.
  - **`GET /api/o/{oid}/h/{hid}/worklists`** and **`GET /api/o/{oid}/worklists`**: the tabs, `[{key, label, help, count}]`, in the agreed order. The hospital's label names its days ("Procedure within 14 days"); the trust's list, whose hospitals may differ, says "Procedure soon".
- **Status messages** have no route of their own: `applyStatus` (S20) sends one after the move is committed (`internal/invite` `Sender.SendStatus`).

### Settings (S10's screens show them with no new screen code)

- **Worklists:** `worklists.soonDays` ("Procedure within N days" worklist: number of days), per hospital, 1–365, default 14.
- **Messages when the status changes** (a new group), per hospital, each switch off by default:
  - `statusMessages.readyForAdmissionSms` / `templates.readyForAdmissionSms`;
  - `statusMessages.readyForAdmissionEmail` / `templates.readyForAdmissionEmail`;
  - `statusMessages.cancelledSms` / `templates.cancelledSms`;
  - `statusMessages.cancelledEmail` / `templates.cancelledEmail`.
- The status templates are the `template` kind with **`LinkOptional`**: `{{link}}` may be left out (invites and reminders still need it). Their help says so.

### Status messages (EPI-08, answers 4)

- **Which moves send one** (`episode.StatusMessage`): a clinician's pick of **Ready for admission**, and **Cancel**. Nothing else.
- **Each channel goes only if its switch is on** at the hospital (effective settings: its own, else the trust's, else off), to the patient's mobile (only this country's, PAT-11) and email as on their record now.
- **`{{link}}`** is the patient's sign-in page (`<APP_URL>/signin`, S15), not a new link: a status message gives no access by itself.
- **Recorded** in `episode_messages` (kind `status`, with the status). **General notes** say how it went ("Ready for admission message sent by text message and email.", "… not sent. The email couldn't be sent.", "… not sent. The patient has no mobile or email."). With both switches off, nothing is sent and nothing is noted.
- **Failures** are shown as for invites (S17): a send that fails, or AWS's later report (`delivery.Record`), puts it in the episode's `undelivered` list, and the episode page shows a red **Message not delivered** alert under the status, separate from the invite card's. The General notes line says "The Ready for admission text message wasn't delivered: …. Check the patient's mobile number, and tell them another way."
- **The move itself always stands**: a message that can't be sent is noted, never refused.

### Web

- **Tasks panel** on the episode page (`TasksPanel.tsx`, below the status): the tasks (open first), each with its team, "Added by … on …" and "Completed by …" or "Re-opened by …"; **Add task** opens an inline form (Task, Team; `noValidate`, Enter sends; "Enter the task." / "Choose the team."); **Complete** and **Re-open** buttons. A stale page is refused with **Reload**. A cancelled episode's tasks are shown without buttons.
- **The clinician home** (`EpisodeList.tsx`): tabs **All**, then each worklist with its count (from `GET …/worklists`, loaded with every list). The tab is in the URL as `?worklist=<key>`, so reload, deep links and Back keep it; it works with **All hospitals** (`?hospitals=all`) too. The status filter belongs to All; a worklist shows its help line instead. Each row shows "1 open task" / "2 open tasks".
- **Undelivered:** `undelivered.ts` names a status message ("The Ready for admission text message wasn't delivered: …"); the invite card and the "Invite not delivered" badge count only the invite's and reminders'.

### TypeScript Rahul should know about

- `features/episodes/tasks.ts` (unit-tested in `tasks.test.ts`): the `Task` shape, the three teams' labels (`TEAMS`, `teamLabel`), `taskHistory` (the "Added by / Completed by / Re-opened by" lines) and `openFirst` (open tasks first). Which teams exist is Go's (`episode.Teams`); the API refuses any other.
- `features/episodes/TasksPanel.tsx`: the panel; it reloads the tasks whenever the episode's revision changes, so after any change (here or elsewhere, after Reload) the list is the server's.
- `features/episodes/EpisodeList.tsx`: the tabs; `setParam` changes one URL parameter and keeps the others. **The worklists' rules, labels and counts are all Go's**; the page only shows them.
- `features/episodes/undelivered.ts`: `inviteUndelivered` / `statusUndelivered` split the list between the invite card and the status alert.

## Decided by the builder (automatic mode)

- **A task change is an episode change:** POST and PATCH send the episode's `revision` and bump it (S20's lesson: every staff write sends it). A stale page gets 409 `stale` with Reload, as for the status. The cost: a status move from a tab opened before someone added a task is refused as stale, which is the S20 rule working as intended.
- **A cancelled episode's tasks can't be changed** (409), as the rest of a cancelled episode (S20: read-only except notes), and **a cancelled episode is in no worklist**, so its open tasks don't linger in Open tasks.
- **Who re-opened a task is kept** (`reopened_by`, `reopened_at`), so the panel can say so after `done_by` is cleared; every add, complete and re-open is also a General notes line.
- **"Procedure within N days"** is from **today to today + N inclusive** at the hospital; a date already past is out (the procedure has gone by), and so are Admitted, Discharged and Cancelled as well as Ready for admission (they're past the point the list is for).
- **"Not responding"** is S16's flag while the HQ is still waiting for the patient (status HQ not complete); it leaves the list when they send it, as the episode's badge does.
- **The worklists' counts are a route of their own** (`GET …/worklists`), not audited: they hold no names or IDs. Opening a list is audited, as before (`episode.list`, `episode.list_trust`).
- **The tab label at "All hospitals" is "Procedure soon"**, since each hospital may have its own N; its help line says so.
- **Status messages go only on a move forward by hand:** undoing an admission lands on Ready for admission again but sends nothing; nor does any automatic or workflow move.
- **`{{link}}` in a status message is the patient's sign-in page** (`/signin`), not a token link, and the default templates don't use it.
- **The default wording** (made up, plain, no clinical claim): Ready for admission: "Hello {{firstName}}, {{hospital}}: your pre-operative assessment is complete and you are ready for your operation. The hospital will contact you about your admission." Cancelled: "Hello {{firstName}}, {{hospital}} has cancelled your pre-operative health questionnaire, so you don't need to fill it in. Please contact the hospital if you have any questions." The emails say the same at more length. Email subjects: "<hospital>: ready for your operation" and "<hospital>: your health questionnaire was cancelled".
- **The team names** are Nurse, Anaesthetist and Admin, as the answer gives them.

## Differences from the plan or brief

- **The tasks table** has the team (`team`) where the plan said `role` (answers 1), plus `created_by_rule` for C6's rules and `reopened_by` / `reopened_at`.
- **`PATCH …/tasks/{tid}`** names the task in the path (the brief wrote `PATCH …/episodes/{eid}/tasks`).
- **One route the brief didn't list:** `GET …/worklists` (and the trust's), for the tabs' counts.

## Tests (TDD)

- **Unit (Go):**
  - `internal/episode/tasks_test.go`: `TestTaskInput` (trim, one line, 200 characters, the team's key), `TestEveryTeamHasALabel`, `TestWorklists` (the six keys in order, the labels with and without N), `TestStatusMessage` (only a pick of Ready for admission and Cancel).
  - `internal/settings`: the new keys are per hospital; a status template may leave out `{{link}}`, an invite's and a reminder's may not; the switches are off by default and sit beside their templates; `worklists.soonDays` defaults to 14.
  - `internal/delivery`: `TestNote` for a status message.
- **Integration:**
  - `tasks_integration_test.go`: `TestListTasks`, `TestCreateTask`, `TestUpdateTask`: the five cases each; invalid input (no or bad revision, empty, too long, two lines, unknown team; another episode's task → 404); stale → 409 `stale`; a task for each team, by Cara, another clinician and a trust-wide super clinician; complete by someone else, then re-open, each recorded (rows, revision, General notes); already done or open → 409; a cancelled episode → 409.
  - `worklists_integration_test.go`: `TestWorklists`: **each worklist returns exactly its episodes**, at Hospital A (UTC+14) and A2 (UTC−11, its own 7 days), with episodes just inside and just outside each rule (day 0 and day 14 in, day 15 and yesterday out, Ready for admission / Admitted / Discharged out, a done task out, a cancelled episode with an open task out); the counts are the lists' lengths in the agreed order; a trust-wide admin's All hospitals has both; Cara's All hospitals is still only A1; a worklist and a status together; a trust's own days; the five cases (no sign-in, the other hospital's clinician, another trust, unknown worklists, bad ids). `TestWorklistsFollowChanges`: adding a task, completing it and a status move change the counts.
  - `status_messages_integration_test.go`: `TestStatusMessages`: off by default (nothing sent, no note); switched on at the hospital: its own wording without a link by text, and with `{{link}}` as the sign-in page by email; undoing an admission sends nothing; each channel's own switch, and another hospital that sets nothing sends nothing; a trust's switch; a send that fails is recorded, shown in `undelivered` and noted, and the move stands; only email on and no email → noted; AWS's later report → noted; no contact → noted.
  - `settings_integration_test.go`: a status template without `{{link}}` is accepted for one hospital, an invite's is refused with a message naming `{{link}}`.
  - `apitest` `TestTasksWorklistsMigration`: the new CHECKs, `in_worklist`, and Down.
- **Web (Vitest):** `tasks.test.ts`, `undelivered.test.ts` (a status message; the split).
- **End-to-end:** `journeys/test_tasks_worklists.py`: Cara adds a task for the anaesthetist team (Enter sends; "Choose the team." first); on the home the episode is in **Open tasks** and **Anaesthetic review needed** (not On hold), the tab's count is the list's length, the tab survives Back and reload; she completes it ("Completed by Cara Clinician") and the episode **leaves Open tasks**; re-opened, it comes back. And: with Hospital A1's Ready for admission email switched on (by the API, put back after), moving a reviewed episode there sends the captured email in the hospital's wording.

### Results (21 Oct)

- `make lint`: passed.
- `make test`: Go all ok; Vitest 197 + 116 passed; CDK tests 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against my own API on :8091 with its own database `sj_s21`, and Vite on :3101, `E2E_BASE_URL=http://localhost:3101`): **55 passed, 5 skipped** (the QA-only Cognito journeys), the new `test_tasks_worklists.py` included.
- Checked headless (Playwright screenshots, not the testers' Chrome): the tabs at 1280 px and 390 px (they wrap), and the Tasks card; no console errors.
- **Found while checking:** "Procedure within 1 days" for a hospital set to 1 day: now "1 day" (test first, `TestWorklists`).

- **Local database:** `sj` migrated to 00022 (`make migrate`); no re-seed needed. My e2e database `sj_s21` can be dropped.

## Test report 1 (21 Oct): all 26 passed

- Every plan case (T-01–T-21) and regression (R-1–R-5) passed.
- **Logged in pending.md §3, no code change** (as the conductor asked): General notes dates are ISO (older than S21); "← Episodes" returns to All, not the worklist; the tabs' help is a native tooltip and All has none; a cancelled episode's Tasks title still counts its open tasks.
- The tester's window couldn't be resized, so T-20 and R-2 were checked at 390×844 in an iframe: see the hand check below.

## Hand checks for Rahul

- **T-20 at a real phone size:** open a status message's `{{link}}` (the sign-in page) in a real 390×844 window; the tester could only use an iframe of that size.


- None yet.

## Suggested commits

1. **S21: tasks, worklists and status messages**: everything above (migration 00022 and its test, the queries, `internal/episode` tasks/worklists/status messages and their unit tests, the settings, `invite.SendStatus`, `delivery.Note`, the routes and their integration tests, the Tasks panel, the worklist tabs, the undelivered split, the journey).
2. **docs: S21 build notes, test plan, decisions, plan and session status**.
