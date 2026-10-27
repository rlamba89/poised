# S43 verification

## Verification 2 (27 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1's fixes are all in and tested. Finding 1 (the two teams) shows in the live data: episode `48304c87…` has one "Book anaesthetic review" task for the nurse team and one for the anaesthetist team. One test was flaky, and I fixed it (finding 0). The rest are gaps or tidying, recorded in pending.md for Rahul.

### Findings, worst first

0. **Fixed by me: the new two-teams integration case was flaky** (`apps/api/internal/httpapi/rule_actions_integration_test.go:240`).
   - **Why:** it expected the nurse task before the anaesthetist one. Both tasks of one send share `created_at`, so `ORDER BY created_at, id` falls back to random UUIDs. Test report 3's note 3 saw the same random order in the app.
   - **What happened:** my first `make test-integration` failed on it, with the tasks in the other order.
   - **The fix (test only):** the teams are now compared as a set. The case then passed 20 of 20 runs, and the full `make test-integration` passed. The app's order of one send's rows was already in pending.md §3.
1. **The requirement can be got round after Ready for admission** (`apps/api/internal/httpapi/statuses.go:279`, `updateEpisode`).
   - The check runs only on a move *into* Ready for admission. Once the episode is there, the POA Summary can still clear the anaesthetist's grade. **Admit** is then offered and accepted, with only the red line as a warning.
   - **Seen in the browser:** on `48304c87…`, PATCH `anaesthetistAsa: 0` answered 204. The moves still offered `admit:admitted`, and the page showed the red requirement line. I put the grade (III) back afterwards.
   - **Related:** if such an episode is then cancelled, its Undo is greyed out. A cancelled episode's grade can't be added, so the cancel can't be undone.
   - **Why it doesn't block:** it needs a clinician to delete the grade on purpose. The brief only asks that the move *to* Ready for admission be refused, and that is done and tested. It's now a question for Rahul in pending.md §2, with options; the recommended one is to refuse clearing the grade while a rule requires it, from Ready for admission on.
2. **Minor, in pending.md §3:**
   - The red "… Add it on the POA Summary." line also shows on cancelled and archived episodes, where the grade can't be changed.
   - The requirement is checked in four places (`applyStatus`, `statusView` twice, `autoTriage`) rather than in `internal/episode`'s table.
   - `ActionsOf` copies `asarules.Check`'s flag-name rules instead of sharing a helper.
   - The suggestion line showing whatever the status was already a question for Rahul (verification 1, finding 5).

Checked and fine:
- **Verification 1's fixes:**
  - Finding 1: the guard matches the team, there's an integration case, and it shows in the live data.
  - Finding 2: the Undo carries `blocked`, and its label gives the reason through `pickLabel`.
  - Finding 3: the backstop applies the newline and 100-character limits (`TestActions`).
  - Finding 4: the list and the header show each name once. In Chrome, the header showed one "Anaesthetic review" badge while the Flags card listed both rules' flags.
  - Finding 6: `flags` is `[]`.
- **Tenant isolation:**
  - Flag reads and the clear are scoped to an episode resolved under `requireHospital`.
  - In Chrome, Val (Hospital A2) and Bea (Trust B) clearing an A1 flag both got 404 "Hospital not found.", and the flag stayed standing.
  - The patient-API path answered 404.
- **The rest:** the route is audited (`flag.clear`); there are no UK literals in the change; `repocheck` passes (in `make test`).
- **Docs:** decisions.md, the READMEs, the C6 plan, CLAUDE.md, AGENTS.md and the e2e README are all updated.

### Test results

- `make lint`: clean.
- `make test`: every Go package ok. Vitest: web 285 passed (29 files), apps/evaluate 10, packages/clinical 411. Infra: 124 passed.
- `make test-integration`:
  - First run: `internal/httpapi` FAIL, the flaky case in finding 0.
  - After the test fix: all 29 packages ok (`internal/httpapi` 14.1 s), and the S43 tests passed 12 times in a row.
- `make e2e` (against the conductor's `make dev`): **97 passed, 5 skipped, 1 failed**. The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known huge local audit log, not S43's. J11b, J11, J7 and the statuses and tasks journeys all passed.
- **Evaluate service:** I didn't stop it. One listener is on 127.0.0.1:8090: node PID 4410, the conductor's copy.

### Hand checks for Rahul

- Optional: the patient's send in a real 390×844 window. The testers' browser couldn't be resized; J11b runs the patient at iPhone 13 size in WebKit.

### Commits

As verification 1 suggested, below, committed locally; the hashes are in the conductor's summary. `docs/market/` and `docs/sessions/run-state.md` are left out.

## Verification 1 (27 Oct 2026, automatic run)

**Verdict: Needs fixes.** One real defect (finding 1): a rule's task can be lost without a trace. The rest of the session is sound: the brief is covered, the tests are thorough, and the spot checks passed. Findings 2 to 6 are minor. Fix them if they're cheap; none blocks on its own.

### Findings, worst first

1. **A rule with the same task title for two teams makes only the first team's task** (`apps/api/db/queries/tasks.sql:40-41`, `CreateRuleTask`).
   - **What happens:** the `NOT EXISTS` guard matches on the episode, the rule and the title, open tasks only. It ignores `team`. `triage.ActionsOf` keeps tasks apart by rule, title *and* team (`actions.go`), and `asarules.Check` lets one rule add "Book review" for the nurse team and "Book review" for the anaesthetist team.
   - **The result:** the first insert succeeds. The second then sees an open task from the same rule with the same title, in the same transaction, and inserts nothing. No note says the task was skipped.
   - **Why it matters:** the anaesthetist team never gets its task, so the episode never appears on **Anaesthetic review needed** (EPI-02). That's a clinical miss.
   - **Fix, test first:** add `AND t.team = sqlc.arg(team)` to the guard and run `make sqlc`. Add an integration case to `TestRuleActionsOnSend`: one rule with the same title for two teams gives two tasks, two notes, and the episode on Anaesthetic review needed.
2. **The Undo back to Ready for admission is offered while the requirement is unmet, then refused** (`apps/api/internal/httpapi/statuses.go:547`).
   - `statusView` sets `blocked` only on `moves`, not on `undo`.
   - **Seen in the browser:** episode `53ef5f0b…` went Ready for admission → On hold, then I cleared the anaesthetist's grade. The page offers "Undo: back to Ready for admission". After a reason is typed and Undo is pressed, it answers "Add the anaesthetist's ASA grade first.".
   - The red requirement line above the button already explains why, and nothing wrong is saved. Even so, the dropdown greys the same move out while the Undo button doesn't.
   - **Suggested fix:** give `undoOption` a `blocked` and disable the button with the reason, or hide the Undo while the requirement is unmet.
3. **`ActionsOf`'s flag backstop doesn't match `asarules.Check`** (`apps/api/internal/triage/actions.go:55-58`).
   - It trims the name and drops empty ones only. Check also refuses a newline and anything over 100 characters (`MaxName`), and `episode_flags.name` has `CHECK (length BETWEEN 1 AND 100)`.
   - A longer name from the evaluator would make `AddRuleFlag` fail. The whole send would then answer 500, and the patient couldn't send their HQ.
   - Rules are checked when they're saved, so this shouldn't happen. But the comment says the backstop leaves out "anything that check would refuse", so it should apply the same limits.
4. **Duplicate React keys when two rules add a flag with the same name** (`apps/web/src/features/episodes/EpisodeList.tsx:135`, `key={f}`).
   - The unique index is per rule and name, so "High risk" and "Airway" can each add "Anaesthetic review". The list row then shows the badge twice, and React warns about the repeated key.
   - **Suggested fix:** use the index as the key, or show each name once (on the row and in the page's header).
5. **The suggested-status line is shown whatever the episode's status** (`apps/web/src/features/episodes/StatusControl.tsx:147`).
   - An admitted, cancelled or archived episode still says "The HQ's rules suggest: Ready for POA".
   - The builder chose this on purpose (build notes, decision 4: "a record of what the rules suggested"). **Question for Rahul:** should the line show only while the episode hasn't yet passed the suggested status, or before admission?
6. **`GET …/episodes/{eid}` sends `"flags": null` when there are none** (`apps/api/internal/httpapi/episodes.go:251`). Every consumer uses `?? []`, so nothing breaks. `moves` and `statusHistory` are normalised to `[]`, and `flags` should be too.

Checked and fine:
- **Tenant isolation:** `GetEpisodeFlag` and `ClearEpisodeFlag` are scoped to the episode, which is resolved under `requireHospital`.
- **Audit:** `flag.clear` is audited, with IDs only.
- **Patient router:** it has no flag route.
- **Locale:** no UK literals in the change.
- **Repo check:** `repocheck` passes.
- **Docs:** decisions.md, the plans and sessions READMEs, the C6 plan, CLAUDE.md, apps/web/AGENTS.md and e2e/README.md are all updated.
- **Brief:** every item in its Build list is done. Answer 1 (clear with a reason) is followed. The builder's 7 decisions are reasonable and recorded.

### Spot checks (Chrome, conductor's `make dev`)

1. **Clearing another hospital's or trust's flag**, on a standing flag on an A1 episode (`7c17035b…`):

   | Who and URL | Answer |
   | --- | --- |
   | Signed out | 401 |
   | Val, A1 URL | 404 "Hospital not found." |
   | Val, A2 URL | 404 "Episode not found." |
   | Bea, A1 URL | 404 "Hospital not found." |
   | Bea, B1 URL | 404 "Episode not found." |
   | Patient-API path | 404 |
   | Cara, blank reason | 400 |
   | Cara, stale revision | 409 `stale` |
   | Cara, no revision | 400 |

   All as expected.
2. **Ready for admission while the requirement is unmet** (`53ef5f0b…`):
   - With the anaesthetist's grade cleared, the move is `blocked` and the PATCH answers 409 with the message.
   - The nurse's grade doesn't meet the requirement. The anaesthetist's grade does, and then the move works. The episode was put back to Ready for admission with ASA 3.
   - This check found finding 2 (the Undo).
3. **Clearing in the page:** Clear opens with the reason box focused, and Enter sends.
   - The card turns grey: "Cleared by Cara Clinician on …: Verifier spot check".
   - There's one `flag.clear` audit row (204) with IDs only.
   - The General notes line is by Cara, after the three automatic lines (task, flag, requirement).

The patient's screens didn't change in this session. J11b runs them at iPhone 13 size in WebKit.

### Test results

- `make lint`: clean.
- `make test`: every Go package ok. Vitest: web 284 passed, apps/evaluate 10, packages/clinical 411. Infra: 124 passed.
- `make test-integration`: every package ok (`internal/httpapi` 19.1 s).
- `make e2e` (against the conductor's `make dev`): **97 passed, 5 skipped, 1 failed**. The failure is `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known huge local audit log, not from S43. J11b, J11, the statuses and tasks journeys, and J7 all passed.

### Hand checks for Rahul

- Optional: the patient's send in a real 390×844 window. The testers' browser couldn't be resized, so they used a 390 px iframe.

### Suggested commits (after the fix and a re-verify)

1. `S43: triage.ActionsOf, the rules' writes and their notes`: `internal/triage/actions.go`, `actions_test.go`, `triage.go`.
2. `S43: rule tasks, flags, the anaesthetist's ASA requirement and suggestions on send; clearing a flag`: migration 00034, `db/queries/{flags,tasks,episodes}.sql`, `internal/db`, `internal/episode/episode.go`, `httpapi/{patient,statuses,flags,tasks,episodes,router}.go`, `rule_actions_integration_test.go`, `triage_integration_test.go`.
3. `S43: the episode page's flags, the status control's requirement and suggestions, the list's badges`: `apps/web/src/features/episodes/*`.
4. `S43: J11b; the stale-tab journey waits for both tabs`: `e2e/journeys/test_j11b_high_risk.py`, `e2e/journeys/test_statuses.py`, `e2e/README.md`.
5. `S43: docs`: this folder, decisions.md, pending.md, CLAUDE.md, apps/web/AGENTS.md, the plans and sessions READMEs, the C6 plan's status. Leave out `docs/market/` and `docs/sessions/run-state.md`.

For the builder: "Verification found problems: read `docs/sessions/S43-rule-actions/verify.md`, fix them test-first, then ask me for a retest."
