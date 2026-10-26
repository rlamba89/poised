# S37 verification

## Verification 1 (26 Oct 2026, automatic run)

**Verdict: Verified.**

Reviewed the whole uncommitted change against the brief, ran every automated test, ran `/code-review high` and weighed its findings, and spot-checked the riskiest paths in Chrome (the "rahul chrome" profile) against the conductor's `make dev`.

### Scope

Everything in the brief is done:
- **No Logic on a summary:** `logicOffered` (`packages/clinical/src/summary.ts:114`) hides the Logic tab and panel (`apps/web/src/features/editor/SettingsPanel.tsx:43-45, 66, 195`); a request for the Logic tab on a summary falls back to Settings, which covers both callers that open Logic (`QuestionnaireEditor.tsx:204-208` `openLogic`, and `:406` the logic-problems list).
- **Publish refuses a summary with a condition:** `summaryProblems` (`summary.ts:141`) adds one problem per summary with a non-empty `visibleIf`, `requiredIf` or `enableIf`; `publishProblems` already prefixes the Question Set's name.
- **A draft that already has one** shows the problem, an orange warning in its Settings (not to read-only viewers), and keeps the Logic tab only so the condition can be removed; `LogicEditor.tsx:94-96` also lists Required if / Enable if on such an element.
- **Published versions are never rewritten:** nothing touches stored content; `withClinicalSummaries` is unchanged.
- Tests: unit tests in `summary.test.ts` and `publish.test.ts`; J10 extended (no Logic on a summary; a draft from `e2e/data/summary-condition-hq.json` is refused, fixed in the Code tab, then published). No integration test, correctly (see finding 1).
- Nothing outside the brief. No new route, query or migration, so tenancy, audit and the patient/staff split are unchanged.

### Findings (worst first; none blocks)

1. **Go's publish doesn't check summaries (accepted, recorded).** `publishQuestionnaire` (`apps/api/internal/httpapi/questionnaires.go`) checks only question keys (S23). Summary problems (two in one Group, one in a repeating group), logic problems and test cases are all checked in the browser's Publish window (plan 2.1). So a client that calls the API directly can still publish a summary with a condition. That's the project's existing pattern, and the brief says the Go check is only needed "if Go checks summaries at publish". The builder recorded it in decisions.md. Moving the structural summary checks into Go would be a separate change that covers every summary rule, not just this one, so I've noted it in pending.md and left it out of this session.
2. **The HQ importer can still put a condition on a summary** (`apps/api/internal/hqimport/convert.go:225-236`). It copies the incumbent's `renderIf`/`visibleIf` onto every element, including a `ClinicalPageSummary`, which becomes a `clinicalsummary`. An imported draft with one now shows the publish problem until the author removes it by hand. Nothing breaks, because the check catches it. The importer could drop the condition with a `c.warn`. Added to pending.md (the brief left the importer's summary notes out of scope).
3. **The Logic Builder on a summary that already has a condition can also add one** (`SettingsPanel.tsx:44`). For example, on a summary that has only a `requiredIf`, Display: Conditionally can add a `visibleIf`. This only applies to drafts made outside the editor, and Publish still refuses them. Low; noted in pending.md.
4. **`requiredIf`/`enableIf` block Publish although they do nothing on a summary**, and the message says "a summary is always shown". The brief asks for all three to be refused, so this is as specified. The wording is a pending.md note (S37 test report 1 already notes that Display: Always can't remove them).
5. *(Code-review claim, not reproduced.)* "After a summary's condition is removed, the parent's tab state stays `logic`, so the next question opens on Logic." In Chrome, after Display: Always on the Breathing summary (Logic tab gone), selecting "Do you have asthma?" opened on **Settings**. Dismissed.

### Checks

- **A-20:** the grep prints only pre-existing comments (`apps/web/src/lib/format.ts`, `apps/api/internal/patient/patient.go`); S37 adds no country literal.
- **Repo check:** `internal/repocheck` passes (in `make test`).
- **Docs:** decisions.md (S37 entry; the 26 Oct answer marked built), pending.md (S37's test report notes), sessions README row, `apps/web/AGENTS.md`. No plan of its own, so there's no plan status to update.

### Test results (26 Oct)

- `make lint`: passes.
- `make test`: Go ok (repocheck included); Vitest web 27 files / 268 tests passed; clinical 22 files / 309 tests passed; CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` (against the conductor's `make dev`): 84 passed, 5 skipped (the Cognito-only ones) in 16 min 10 s.

### Browser spot-checks (Chrome, "rahul chrome")

1. **Publish before opening a set:** as Alex Author, on the tester's "S37 Summary HQ 1031" v3 draft, Publish listed "About you · The clinical summary “Breathing summary” has a condition: remove it (a summary is always shown)" (plus two Video problems from the tester's leftovers), and the Publish button was disabled. **Pass.**
2. **A summary with a condition:** the card shows "Displays when Patient's age ≥ 18"; Settings show the orange warning and a Logic tab. In Logic → Builder, **Display: Always** removed it: the Logic tab and the warning went, and the card lost "Displays when". **Pass.**
3. **The tab after removal:** the next question selected opened on Settings (finding 5). Undo brought the condition back, and the API confirmed `cs_meeela` has no `visibleIf` after the removal, before Undo. **Pass.**
4. **Read-only viewer (Sue Super):** the same draft is READ ONLY with no Publish button; the summary shows a disabled Title, a Logic tab, and **no** warning. **Pass.**
5. Console: no errors.

### Suggested commits

1. `S37: no conditions on summary boxes`: `packages/clinical/src/{summary,summary.test,publish.test}.ts`, `apps/web/src/features/editor/{SettingsPanel,LogicEditor}.tsx`, `e2e/journeys/test_j10_authoring.py`, `e2e/data/summary-condition-hq.json`, `apps/web/AGENTS.md`.
2. `S37: docs — brief, build notes, test plan and report, verification`: `docs/sessions/S37-summary-no-conditions/`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`.

Not committed: `docs/market/`, `docs/sessions/run-state.md`.

### Hand checks for Rahul

None.

### Left behind

The tester's "S37 Summary HQ 1031" v3 draft is as the tester left it (its Breathing summary still has its condition: removed in the spot-check, then restored with Undo).
