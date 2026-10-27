# S42 build notes: auto-triage and the Auto-triaged worklist

Builder, automatic run, 27 Oct 2026. Rahul's answers ([answers.md](answers.md)), followed where they differ from the brief:
1. An episode leaves the **Auto-triaged** list when a clinician clicks **Checked** (who and when recorded).
2. **A setting on each HQ; by default the auto-triaged episode moves straight to Ready and the patient's message goes at once.** Guards that can't be switched off: the server's check ran, no score Incomplete, no amber or red flags, and the rules pass their test cases. The episode is still on the Auto-triaged list for Checked. The wording avoids "fit for surgery". The trust's clinical safety officer signs this default off before go-live (C10).

## What was built

### Go

- **`internal/episode`:** a new move **`AutoTriage` ("auto_triage")**, made **only by the system**: it's not in the role table (`allowed`), so `Check` refuses it to every role and `Options` never lists it. `CheckAutomatic(from, to, move)` allows it from **Ready for review** to the three targets (asarules.Targets). `StatusMessage(AutoTriage, ReadyForAdmission)` is the Ready for admission message. Auto-triage isn't undoable (re-open instead). The **Auto-triaged** worklist ("Moved on automatically by the HQ's rules; not yet checked by a clinician."), second, after Ready for review.
  - Tests: `TestCheckEveryPair` has `AutoTriage` in its moves (always illegal by hand); new `TestCheckAutomaticEveryPair` (every pair × every move); `TestUndoTarget`, `TestStatusMessage`, `TestWorklists` cases.
- **`internal/triage` (new):** `Decide(evaluate.Result) Decision`. The **first fired rule with an auto-triage action** (with a known target) wins. **Guards** (always): a red or amber flag; a score Incomplete; a fired rule requiring the anaesthetist's ASA when the target is Ready for admission (S43's action). A stopped decision keeps the rule's name and says why. `Note()` is the General notes line; the stamp's words are "ASA II, no red or amber flags" ("no ASA grade, …" when none; an emergency grade keeps its E). Unit-tested (`TestDecide`, 15 cases; `TestDecisionWords`).
- **`internal/evaluate`:** `Result.Sets` (each shown set's scores, `Incomplete`), so Go can read the Incomplete guard; the fixture test checks it decodes.
- **`internal/asarules`:** `RuleSet.MessageAfterCheck` (`messageAfterCheck`, optional: a set stored before S42 reads as off; anything but true/false refused); `defaults.json` names it (`false`). Test `TestCheckKeepsWhenTheMessageGoes`.
- **Migration 00033:** the `move` CHECK gains `auto_triage`; `episodes.triage_rule`, `triage_summary`, `triaged_at`, `triage_message_pending`, `triage_checked_by`, `triage_checked_at` (with a CHECK keeping them consistent); `in_worklist` gains `auto_triaged` (auto-triaged, unchecked, not cancelled or archived).
- **Queries:** `GetEpisode` returns the triage columns and the checker's name (so the episode JSON has `triageRule`, `triageSummary`, `triagedAt`, `triageMessagePending`, `triageCheckedAt`, `triageCheckedByName`); `SetEpisodeStatus` **clears them on a re-open**; new `AutoTriageEpisode`, `LockEpisodeTriage`, `CheckEpisodeTriage`.
- **On send** (`submitForm`, both routes): `evaluateSent` now returns the result (nil when the call failed); `autoTriage` then, in the same transaction: no rule → nothing; a guard → an automatic note "Not auto-triaged by the rule “Low risk”: <why>. Review this HQ by hand."; else the episode moves to the target, the status history gets `auto_triage` (from Ready for review, the rule's name, no user) after the Submit event, and an **automatic** note "Auto-triaged by the rule “Low risk”: ASA II, no red or amber flags → Ready for admission". After the commit, the Ready for admission message goes at once (automatic notes, `automaticStatusMessage`) unless the HQ's `messageAfterCheck` holds it (`triage_message_pending`).
- **Checked:** `POST /api/o/{oid}/h/{hid}/episodes/{eid}/triage-checked {revision}`, audited **`triage.checked`**, any clinician at the hospital. 409: stale revision (`code: stale`), archived, cancelled, never auto-triaged, already checked. Records who and when, bumps the revision, writes "Auto-triage checked" (the user's), and sends a held message if the episode is still Ready for admission.
- **Integration tests** (`triage_integration_test.go`, fake evaluator):
  - `TestAutoTriageOnSend`: low risk sent by the patient → Ready for admission, one submit and one auto_triage event naming the rule (no user), the automatic note, on the worklist and not Ready for review, the episode JSON's triage fields; staff on behalf the same; targets Ready for POA and POA complete; the fake failing → Ready for review, no auto-triage; no auto-triage action → Ready for review, nothing more; each guard (amber, Incomplete, anaesthetist's ASA) → Ready for review with its note; the tab's count, and a cancelled or archived one in none.
  - `TestAutoTriageMessages`: the text at once by default (automatic note); held with `messageAfterCheck` until Checked, then sent; moved on by hand before Checked → none; re-opened → none, and the triage cleared.
  - `TestTriageChecked`: the five cases (happy path with the audit row; no sign-in 401; a clinician at the trust's other hospital 404; another trust 404; invalid input 400) plus twice, never auto-triaged, not sent, cancelled, archived (409) and a stale revision (409 stale).
  - `TestReopenAutoTriaged`: a clinician can't (403), no reason 400, a super clinician re-opens → Ready for review, the triage cleared, off the list, Checked 409, and Validate saves again.
  - `TestWorklists` (S21's) has auto-triaged episodes just inside and outside the rule at both hospitals.

### TypeScript (for Rahul to review)

- **`packages/clinical/src/rules.ts`:** `RuleSet` gains an optional `messageAfterCheck` (the same setting as Go's). `ruleSetOf` reads it as `true` only when the stored value is exactly `true`. `autoTriageSummary` (the Publish window's line) adds "The patient's message goes when a clinician clicks Checked." when it's on. Tests in `rules.test.ts`.
- **`apps/web/src/features/episodes/status.ts`:** the move `auto_triage` (history: "Auto-triaged: Ready for review → Ready for admission", by "Automatic (Low risk)", from S20's `historyWho`); two small functions: `triageStamp(episode)` gives "Auto-triaged (ASA II, no red or amber flags)" or null, and `awaitingCheck(episode)` is true while auto-triaged, unchecked, and not cancelled or archived (as Go's worklist rule). Tests in `status.test.ts`. `types.ts`: the episode's six triage fields.
- **`EpisodePage.tsx`:** a new `AutoTriage` card (shown while the episode is auto-triaged): the rule, where it went, when, "No clinician validated the answers…", then either "Not yet checked by a clinician." (plus "The patient's message goes when you click Checked." when held) with a **Checked** button (POST with the episode's revision; a stale one reloads the page), or "Checked by X on …". The Health questionnaire card shows the stamp in each set's state column instead of "Not started".
- **`PoaSummary.tsx`:** for an auto-triaged episode, the orange "The HQ review isn't complete yet" line becomes a teal "Auto-triaged (…) by the rule “…”. No clinician validated these answers, so this shows the patient's own." (the Validated tab already shows the patient's answers where there's no clinician copy); each set's by-line starts with the stamp.
- **`RulesPage.tsx`:** a second checkbox under the no-grade one: "When a rule auto-triages a patient, wait for a clinician to click Checked before the patient's status message goes" (off by default).

### e2e

- **J11's patient half** (`test_j11_triage.py::test_a_low_risk_patient_is_auto_triaged_and_checked`): a smoker (ASA II), phone, sends; Cara's Auto-triaged tab lists it as Ready for admission; the card, three stamps, the history row by "Automatic (Low risk)", the note; the POA Summary's line and by-line; Checked → "Checked by Cara Clinician…", kept after a reload, and off the tab. `e2e/README.md` updated.
- **`support.Api.published_hq` takes the default rules' auto-triage out unless `auto_triage=True`**, so journeys written before S42 still find Ready for review after a send. New `Api.without_auto_triage(hq)` (used by J10's two journeys that publish in the browser and then validate) and `Api.review_by_hand(episode)` (a super clinician re-opens an auto-triaged episode; used by J5's started-fresh journey, on the seeded follow-up HQ).

## Test report 1 (27 Oct): 18 of 19 passed; F-1, E-1 and E-2 fixed

- **F-1 (Minor, T-04): a refused Checked (409 stale from a second tab) reloaded to "Checked by…" with no message.** The card drew its error only while unchecked. **Fixed test-first:** new journey `test_j11_triage.py::test_a_second_tabs_checked_is_refused_in_plain_words` (it failed: no message); the error now shows on the card whichever way the episode is (`triage-check-error`).
- **E-2 (Minor): Test the whole HQ, the Preview and test cases said Low risk auto-triages a patient with an amber flag (or an Incomplete score), but a real send's guard stops it.** **Fixed test-first** so they agree, as the C6 plan intends ("preview, test cases and the server always agree"): `packages/clinical`'s `firedRules` now applies the same guards and holds the rule's auto-triage with a **`heldReason`** (`noGrade` as before, or `flag`, `incomplete`, `anaesthetistAsa`); `firedLine`/new `firedWords` say "no auto-triage: a red or amber flag" (etc.); both pages use `firedWords`. The evaluate result carries `heldReason`; Go's `triage.Decide` turns a guard reason into the "Not auto-triaged…" note (and keeps its own checks as a backstop; a later rule not held, e.g. one to Ready for POA when the anaesthetist's ASA is required, still moves it, as the Preview says). The Publish window's line adds "Never with a red or amber flag, a score Incomplete, or the anaesthetist's ASA required first."; the Rules page's intro says the same. Tests: `rules.test.ts` (the three guards, a held rule keeping its other actions, the Publish line), `evaluate.test.ts` (held with its reason), `TestDecide` (+5 cases), `TestAutoTriageOnSend` (a held result). **J11's authoring half now answers the snoring question** (unanswered, its guarded score is Incomplete, so Low risk is held) and checks the held line first.
  - *Effect on saved test cases:* a whole-HQ test case whose answers leave a score Incomplete or give an amber flag now gets the held line, so it fails until saved again. Only drafts are checked (publish), and this is the true outcome of such a send.
- **E-1 (Cosmetic): a staff-sent HQ described as the patient's.** The card now says "when <name> sent the HQ on the patient's behalf"; the on-behalf confirm says the episode "moves to Ready for review, or on from there if the HQ's rules auto-triage it"; the sent page "The answers are with the team. The episode page shows its status now."
- **E-3 (an unreproduced 503 on one double-click):** the route has no 503 and the episode was checked once; the tester's three-at-once requests gave 204/409/409. Not changed; noted.
- **E-4 (plan):** T-10 and T-13 now use the newest published version (publishing v2 retires v1). The plan's categories and the smoker note follow the tester's notes.
- After the fixes: `make lint`, `make test` (clinical 411, web 279), `make test-integration` pass. Full e2e on the private stack: **94 passed, 5 skipped, 1 failed** (`test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`, the known huge local audit log). The private stack was stopped afterwards; the conductor's make dev untouched.

## Test report 2 (27 Oct): all passed

Every case passed, including the retests (T-02, T-04, T-07, T-10, T-13, T-15). Two exploring notes went to pending.md, not changed (a retest would be needed): the thank-you page's "A member of the team will review them." against the hospital's Ready for admission text on an auto-triaged send (a product question for Rahul), and the refused Checked's standard "Reload to see the latest" wording. An archived auto-triaged episode keeps its card without Checked, as designed.

## Decided by the builder (automatic mode)

1. **Auto-triage goes from Ready for review, after the Submit event** (the brief's "record Submit, then the auto-triage move"), not from HQ not complete as the brief's first bullet says: so the history reads "HQ sent: HQ not complete → Ready for review", then "Auto-triaged: Ready for review → target". Since the table is keyed by (from, to) and Ready for review → Ready for POA is already Complete review, the system's moves are a small table of their own (`CheckAutomatic`), which also makes "no role may make it by hand" structural.
2. **The setting is two-way** (message at once / after Checked), as answers.md words it. The research's third choice, "suggest only", needs no setting: the author replaces the rule's auto-triage action with "suggest Ready for admission" (S43 shows suggestions).
3. **The guard "the rules pass their test cases"** is met at publish (S41's server check refuses a version whose whole-HQ test case fails), so it isn't re-run on send. Versions seeded or published before S41 were checked by the browser (or have no test cases: the seed's).
4. **A guard that stops a rule writes an automatic note saying why** (the brief says only "nothing changes"), so a clinician sees why a low-risk-looking patient is Ready for review.
5. **A re-open ends the auto-triage** (clears its columns): off the list, no stamp, a held message never goes, Checked refused. The history and notes keep what happened.
6. **A held message goes on Checked only if the episode is still Ready for admission** (a clinician who moved it on by hand made another decision).
7. **The anaesthetist's-ASA guard is applied now** (the brief's "If a rule also requires the anaesthetist's ASA (S43)…"), reading the fired rules' actions; S43 adds the requirement itself.
8. **The status message's wording is unchanged** (S21's hospital setting: "…your pre-operative assessment is complete and you are ready for your operation…"). It doesn't say "fit for surgery"; softer default words are a question for Rahul (pending.md).
9. **The stamp says "no red or amber flags"** (the brief's example says "no red flags"; the guard makes both true).
10. **The Checked route reuses the status routes' body reader** (`{revision}`; any other field is ignored).

## Differences from the brief and plan

- Decision 1 (the move's from-status), decision 2 (two-way setting), decision 4 (the guard's note), decision 5 (re-open clears).
- Answer 2 replaces the brief's question 2 (the message goes at once by default).

## Hazard log

Added to pending.md (S42's section): the hazard (an episode moved on with nobody involved, the message at once by default) and the mitigations (rules tested and signed off, the guards, the Auto-triaged list and Checked, the hold setting, re-open, the fail-safe), and the CSO's sign-off before go-live.

## Hand checks for Rahul

- None yet.

## Test results (27 Oct)

- `make lint`: clean.
- `make test`: every Go package ok (26 with tests, incl. `internal/triage`, `internal/episode`, `internal/asarules`, `internal/evaluate`); Vitest web 279, apps/evaluate 10, packages/clinical 409; infra 124.
- `make test-integration`: every package ok (29), incl. the four new tests and S21's `TestWorklists`.
- **e2e** on a private stack (web :3100, API :8180, evaluate :8190, the shared local database migrated to 00033; the conductor's make dev untouched, and the private stack stopped afterwards): **88 passed, 5 skipped, 6 failed** in the full run. Fixed and re-run alone: J10's two summaries/scores journeys and J5's started-fresh journey (their HQs were auto-triaged; now `without_auto_triage` / `review_by_hand`), 5 of 5 with J16. Not mine: `test_audit.py` × 2 (the known huge local audit log) and `test_j7_patient_invites.py` (3 texts to a reused mobile on the long-lived database; it passed alone). J11 (4 tests) passed.

## Suggested commits

1. `S42: the auto_triage move (system only), the Auto-triaged worklist, the message on auto-triage` — `internal/episode/*`.
2. `S42: triage.Decide and its guards; evaluate.Result's scores; asarules messageAfterCheck` — `internal/triage/*`, `internal/evaluate/*`, `internal/asarules/*`.
3. `S42: auto-triage on send, Checked, re-open clears it` — migration 00033, `db/queries/episodes.sql`, `internal/db`, `httpapi/patient.go`, `statuses.go`, `router.go`, `triage_integration_test.go`, `worklists_integration_test.go`.
4. `S42: the episode page's card and stamps, the POA Summary, the Rules page's setting` — `packages/clinical/src/rules.ts` + test, `apps/web/src/features/episodes/*`, `features/rules/RulesPage.tsx`.
5. `S42: J11's patient half; published_hq without auto-triage` — `e2e/**`.
6. `S42: docs` — CLAUDE.md, apps/web/AGENTS.md, this folder, decisions.md, pending.md, plans and sessions READMEs, the C6 plan's status.
