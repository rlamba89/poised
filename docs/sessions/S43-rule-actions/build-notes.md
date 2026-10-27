# S43 build notes: tasks, flags and the anaesthetist's ASA from rules

Builder, automatic run, 27 Oct 2026. Rahul's answer ([answers.md](answers.md)), followed where it differs from the brief:
1. **A rule's flag is cleared by a clinician with one click and a short reason;** who, when and why are recorded. It comes back if a re-send fires the rule again.

## What was built

### Go

- **`internal/triage/actions.go` (new):** `ActionsOf(evaluate.Result) Actions` turns a sent HQ's fired rules into the writes: `Tasks` (rule, title, team), `Flags` (rule, name), `RequireASA` (the first rule requiring the anaesthetist's ASA grade) and `Suggestions` (rule, status), in the rules' order. A held rule (S42) keeps them (only its auto-triage was taken out). The same action twice in one rule counts once; anything `asarules.Check` would refuse (a blank title, an unknown team, a status that can't be suggested, an unknown kind) is left out as a backstop. The General notes words: `RuleTask.Note()`, `RuleFlag.Note()`, `RequireNote`. `NeedsAnaesthetistASA` is the refusal ("Add the anaesthetist's ASA grade first."); `AnaesthetistASAGuard` is S42's guard words, now shared. Tests `TestActions` (9 cases), `TestActionNotes`.
- **Migration 00034:** `episode_flags` (episode, name, rule, created_at, cleared_by/at/reason, all three set together), a partial unique index so one standing flag per rule and name; `episodes.anaesthetist_asa_required_by` (the rule's name; met while `anaesthetist_asa` is set) and `episodes.rule_suggestions` (jsonb, `[{rule, status}]`, default `[]`).
- **Queries:** `CreateRuleTask` (no row if that rule's task with that title is still open), `AddRuleFlag` (no row if that rule's flag is still standing), `RequireAnaesthetistASA` (no row if a rule already requires it), `SetRuleSuggestions`, `ListEpisodeFlags`, `GetEpisodeFlag`, `ClearEpisodeFlag`. `LockEpisodeStatus` and `GetEpisode` return the requirement (and `GetEpisode` the suggestions); `ListEpisodes` returns each row's standing flags' names (`flags`, `[]` when none).
- **On send** (`submitForm`, both routes, `ruleActions` in `patient.go`), in the send's transaction, after the result is stored and before S42's `autoTriage`: each task, flag and new requirement is written with an **automatic** General notes line ("Task added by the rule “High risk” for the anaesthetist team: Book anaesthetic review", "Flag added by the rule “High risk”: Anaesthetic review", "The rule “High risk” requires the anaesthetist's ASA grade before Ready for admission"); the suggestions replace any earlier ones. The evaluator failing → none of them.
- **`autoTriage`** also holds an auto-triage to Ready for admission while a requirement from an **earlier** send is unmet (a backstop: `triage.Decide` only sees this send's rules).
- **Statuses:** `applyStatus` refuses a move to Ready for admission (409, "Add the anaesthetist's ASA grade first.") while the requirement is unmet: the pick (`set`) and an **undo** back to it; not **un-archive** (it restores exactly the status it was archived from). The episode's `moves` carry it: `episode.Option` gains `blocked` (set on Ready for admission by `statusView`).
- **Clearing a flag:** `POST /api/o/{oid}/h/{hid}/episodes/{eid}/flags/{fid}/clear {reason, revision}` (`flags.go`), audited **`flag.clear`**, any clinician at the hospital. 400: no reason ("Say why the flag is cleared."), over 1,000 characters, no revision, not JSON; 404: not this episode's flag, an unknown id, another hospital's or trust's episode; 409: stale revision (`code: stale`), already cleared ("This flag is already cleared."), cancelled or archived. Bumps the revision; General notes "Flag cleared: Anaesthetic review: <reason>" by the user. It shares the tasks' transaction helper, renamed `changeEpisode` (was `changeTasks`).
- **GET …/episodes/{eid}** returns `flags` (standing and cleared, with `clearedByName`, `clearedReason`), `episode.anaesthetistAsaRequiredBy` and `episode.ruleSuggestions`.
- **Integration tests** (`rule_actions_integration_test.go`, fake evaluator):
  - `TestRuleActionsOnSend`: high risk sent by the patient → the task (with its rule, no user), the flag, the requirement, Ready for review, the three automatic notes, on Anaesthetic review needed, the episode JSON and the list row's flags, the Tasks JSON naming the rule; staff on behalf the same; with Low risk to Ready for admission too → Ready for review (S42's guard); a requirement from an earlier send holds a later Low risk; a suggestion → shown, nothing moves; the fake failing → none; **sent again** (the test resets the episode to HQ not complete in SQL, since nothing re-sends yet) → no second open task or standing flag, the requirement's note once; with the task done and the flag cleared, a third send adds both again.
  - `TestReadyForAdmissionNeedsTheAnaesthetistsASA`: the dropdown's move `blocked`, 409 with the message, other moves fine, the nurse's grade doesn't meet it, the anaesthetist's does (then not blocked, the move works); the grade cleared → unmet again; an undo back to Ready for admission refused while unmet; no rule → as before; the fake failing → no requirement.
  - `TestClearFlag`: the five cases (happy path with who, when, why, the note, the revision, the audit row, off the list; no sign-in 401; a clinician at the trust's other hospital 404; another trust 404; invalid input 400 and another episode's flag 404) plus twice (409), a stale revision (409 stale), archived and cancelled (409).
  - S42's `TestAutoTriageOnSend` now expects the new notes lines in two cases (a rule's task, and the anaesthetist's requirement), since those rules' actions are now made.

### TypeScript (for Rahul to review)

- **`apps/web/src/features/episodes/status.ts`:** `StatusOption.blocked` (Go's `episode.Option.Blocked`); three small functions, tested in `status.test.ts`:
  - `pickLabel(move)`: the dropdown's label, "Ready for admission (add the anaesthetist's ASA grade first)" when blocked;
  - `asaRequirement(episode)`: null when no rule requires it, else `{met, text}`: "Ready for admission needs the anaesthetist's ASA grade first (the rule “High risk”)." or "The anaesthetist's ASA grade is added, as the rule “High risk” asks.";
  - `suggestionLines(suggestions)`: "The HQ's rules suggest: Ready for POA", with the rule.
- **`tasks.ts`:** a rule's task says "Added by rule: High risk on …" (the brief's words; was "Added by rule High risk").
- **`types.ts`:** the episode's `anaesthetistAsaRequiredBy` and `ruleSuggestions`; `EpisodeFlag`.
- **`StatusControl.tsx`:** a blocked move is greyed out in the dropdown with its reason; the red requirement line ("… Add it on the POA Summary."); the blue suggestion lines.
- **`EpisodePage.tsx`:** the red flag badges beside the patient's name; a new **Flags** card (each flag, "Added by the rule “…” on …", **Clear** opening a short form: "Why is it cleared?", Enter sends, the stale-revision Reload as elsewhere; a cleared one in grey with "Cleared by X on …: reason"; no Clear on a cancelled or archived episode).
- **`EpisodeList.tsx`:** each row's standing flags as red badges under the patient's name (`FlagBadge`).
- **`PoaSummary.tsx`:** the ASA grades section says whether the rule's requirement is met.

### e2e

- **J11b** (`e2e/journeys/test_j11b_high_risk.py`): the high-risk patient (phone) → Anaesthetic review needed with the row's flag, the badge, the Flags card, the rule's task, the requirement line and notes; Ready for admission greyed out and refused by the API until the anaesthetist's ASA is saved on the POA Summary, then it works; Clear needs a reason, then the badge goes and who/when/why stay (after a reload too, and off the list row). Second test: a suggested status shows the line, nothing moves. `e2e/README.md` updated.

## Test report 1 (27 Oct): 16 of 17 passed; F-1 and exploring note 3 fixed

- **F-1 (Minor, T-07): after Reload on a stale Clear, the form stayed open under the now-cleared flag** (its Clear flag then got "This flag is already cleared."). **Fixed test-first:** new journey `test_j11b_high_risk.py::test_a_stale_tabs_clear_is_told_and_closes_once_reloaded` (it failed); the Flags card now closes its form when that flag is no longer standing (or the episode became read-only) after a reload.
- **Exploring note 3 (cosmetic): the reason box wasn't focused when the form opened** (Mantine's `data-autofocus` works only inside a focus trap). Now `autoFocus`; the same journey checks it.
- **Exploring note 4 (cosmetic, not this session's):** the General notes say "Anaesthetist ASA grade set to 3" while the page says "ASA III" (S20's wording, checked by older tests). Not changed; added to pending.md.
- **Exploring note 5:** T-06's expected text corrected in the plan (S08's "You don't have access to this hospital" for another hospital's URL).
- Note 6 (phone size through an iframe at 390 px CSS width): J11b runs the patient at iPhone 13 size in WebKit.

## Test report 2 (27 Oct): all passed

All 14 re-run cases passed, including T-07 (F-1), T-05's focus and T-06's corrected wording. Exploring: a stale Clear whose flag is still standing keeps its form open with the reason (right: it can still be cleared); one on an episode archived meanwhile closes after Reload; staff's Fill in for the patient makes the same writes. Note 5 (an archived row shows no open-task count) is S29's rule (`openTaskCount`), as intended. The "ASA grade set to 3" wording is in pending.md (not this session's).

## Verification 1 (27 Oct): needs fixes; all done test-first

1. **A rule with the same task title for two teams made only the first team's task** (`CreateRuleTask`'s open-task check ignored the team). New integration case in `TestRuleActionsOnSend` (two tasks, two notes, on Anaesthetic review needed, not doubled on a second send) failed; the check now matches the team too.
2. **The Undo back to Ready for admission was offered while the requirement was unmet** (then refused). `undo` now carries `blocked` like the moves (`TestReadyForAdmissionNeedsTheAnaesthetistsASA` checks it); the button is greyed out: "Undo: back to Ready for admission (add the anaesthetist's ASA grade first)" (`pickLabel`, reused).
3. **The flag backstop didn't apply `asarules.Check`'s limits** (a newline, over 100 characters: the insert would fail and the send answer 500). `ActionsOf` now leaves those out too (`TestActions`).
4. **Two rules adding the same flag name showed the badge twice (and a repeated React key).** The list's `flags` gives each standing name once (integration case), and the page header uses `standingFlagNames` (`status.test.ts`); the Flags card still lists each flag with its rule.
5. **The suggested-status line shows whatever the status:** a question for Rahul, in pending.md §2 (recommended: only before admission, not once cancelled or archived).
6. **`flags` null when none:** the new assertion (`"flags":[]` in the raw JSON) passed as it was: sqlc's `emit_empty_slices` already sends `[]`. Kept the assertion; no code change.

`make lint`, `make test` and `make test-integration` pass after the fixes.

## Test report 3 (27 Oct, retest after verification 1): all passed

11 of 11, including verification 1's four fixes. Its exploring notes (narrow-window wrapping of a long flag's Clear and the blocked Undo's label, the random same-time order of one send's flags and tasks, two identical "Complete: …" button names for different teams, S20's Undo chain) went to pending.md §3; no code changes (conductor's instruction).

## Decided by the builder (automatic mode)

1. **"No duplicate" means no second *open* task from the same rule with the same title, and no second *standing* flag from the same rule with the same name.** A task that was done, or a flag that was cleared, comes back if a later send fires the rule again, as Rahul's answer says for the flag; the task follows the same rule so the two behave alike.
2. **The requirement is a column** (`anaesthetist_asa_required_by`, the first rule's name), met while an anaesthetist's grade is saved: no separate "met" record. Clearing the grade again makes it unmet again.
3. **Which moves it refuses:** a pick and an undo into Ready for admission; **not un-archive** (it restores exactly what was archived, else an archived episode could be stuck). Auto-triage is held by S42's guard, and now also by a requirement standing from an earlier send.
4. **The suggestions are stored on the episode** (`rule_suggestions`, the latest send's) rather than read back from `form_outputs` on every GET; they're shown always (a record of what the rules suggested), never applied.
5. **The flag's reason is required** (answers.md's "a short reason"), at most 1,000 characters like the status reasons.
6. **Each task, flag and new requirement gets an automatic General notes line**, so the notes say what the rules did, as S42's auto-triage line does.
7. **"Re-open and re-send"** can't be done in the app (an HQ is sent once; re-open goes to Ready for review without asking the patient again), so the integration test simulates the second send by resetting the episode in SQL. The test plan lists it under out of scope.

## Differences from the brief and plan

- Answer 1: the flag is cleared with a reason (the brief's question offered one click, or clearing with the task).
- Decision 3: an undo into Ready for admission is refused too; un-archive isn't.

## Hand checks for Rahul

- Optional: the patient's send in a real 390×844 window (the tester's browser couldn't be resized, so they used a 390 px iframe; J11b runs the patient at iPhone 13 size in WebKit).

## Test results (27 Oct)

- `make lint`: clean.
- `make test`: every Go package ok (incl. `internal/triage`'s new tests); Vitest web 284, apps/evaluate 10, packages/clinical 411; infra 124.
- `make test-integration`: every package ok (29), incl. the three new tests.
- **e2e** on a private stack (web :3100, API :8180, evaluate :8190, the shared local database migrated to 00034; the conductor's make dev untouched, the private stack stopped afterwards). Full run: **94 passed, 5 skipped, 3 failed**:
  - `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a` (the known huge local audit log) and `test_j7_patient_invites.py` (texts to a reused mobile on the long-lived database; it passed alone): not this session's.
  - `test_statuses.py::test_a_stale_tab_is_told_and_keeps_its_input` failed twice out of five runs: the second tab could load the episode *after* the first tab's save (a cold page), so it held the newer revision and wasn't stale. A timing race in the journey, not in the app (it saved what it was sent); **fixed in the journey**: both tabs wait until the episode has loaded before either saves. Passed 3 of 3 after.
  - J11b's first full run read the list's flag badges before the episode page had opened; it now waits for the episode's URL. Then J11b, J11, J5, J9, statuses and tasks journeys: 19 of 19 (J11b twice more, 2 of 2).

## Suggested commits

1. `S43: triage.ActionsOf, the rules' writes and their notes` — `internal/triage/actions.go`, `actions_test.go`, `triage.go` (the shared guard words).
2. `S43: rule tasks, flags, the anaesthetist's ASA requirement and suggestions on send; clearing a flag` — migration 00034, `db/queries/{flags,tasks,episodes}.sql`, `internal/db`, `internal/episode/episode.go` (Option.Blocked), `httpapi/{patient,statuses,flags,tasks,episodes,router}.go`, `rule_actions_integration_test.go`, `triage_integration_test.go`.
3. `S43: the episode page's flags, the status control's requirement and suggestions, the list's badges` — `apps/web/src/features/episodes/*`.
4. `S43: J11b; the stale-tab journey waits for both tabs` — `e2e/journeys/test_j11b_high_risk.py`, `e2e/journeys/test_statuses.py`, `e2e/README.md`.
5. `S43: docs` — this folder, decisions.md, pending.md, CLAUDE.md, apps/web/AGENTS.md, plans and sessions READMEs, the C6 plan's status.
