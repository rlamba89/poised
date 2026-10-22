# S20 verification

## Verification 1 (21 Oct 2026, automatic run)

**Verdict: Verified.** One question for Rahul (below) on how far undo reaches; it doesn't block the session.

Reviewed: the whole working tree against `fb2245b` (40 changed files, 9 new), the brief, answers.md, the build notes, the test plan and test reports 1 (20/21, F-1) and 2 (22/22 passed). `/code-review high` was run and its findings weighed below.

### Question for Rahul: how far does undo reach? (answer 2)

Your answer: **any clinician can undo any manual step**, one step at a time, repeatably, with a reason; cancel stays final.

What was built (`apps/api/internal/episode/episode.go:154` `undoable`, `UndoTarget`): undo steps back a **pick** (e.g. On hold), **Admitted** and **Discharged**, and stops at the first of these in the history:
- **cancel**: matches your answer (cancel is final, answer 3);
- **an undo**: there's no redo, which reads naturally from "one step at a time… to go further back";
- **completing the HQ review**, **staff sending the HQ on the patient's behalf**, and **re-open**: these are manual steps too, so this part is **a narrowing, not just a reading**.

The builder's reason is sound: undoing *Complete HQ review* is a re-open, which answer 1 (accepted table) and answer 6 give only to a super clinician or admin, with a reason. Letting any clinician undo it would quietly override that. Undoing a re-open would need to restore the old review stamp, and undoing an on-behalf submit would unfreeze the patient's answers.

Options:
1. **Keep it as built (recommended):** undo covers the status moves made by hand (pick, Admit, Discharge). Completing the review is undone by **Re-open** (super clinician, with a reason), and a mistaken re-open is fixed by completing the review again.
2. Let undo also step back completing the review (= a re-open by any clinician) and re-open (restore the stamp, back to the status before). This changes answer 1's "re-open is a super clinician's".
3. As 2, and also un-send an on-behalf HQ submit.

Verified on the live app: Cara undid four moves in a row on "S20 Roles" (Ready for admission → POA complete → On hold → POA complete → On hold → Ready for POA), each with a reason and an "(undone)" mark, then undo was no longer offered, because the step before was Complete HQ review.

### Findings (none blocking; worst first)

1. **Cancelling one episode signs the patient out of every episode** if their session was opened from that episode's invite (`apps/api/db/queries/episodes.sql:132` `RevokeEpisodeInvites`, with `links.sql:133` `TouchPatientSession`, which checks the session's link). A patient with a second, active episode is signed out and must use that episode's link or sign in again. It errs on the safe side, and answer 5 ("their invite link and sessions for the episode end") reads that way, so it's not a defect. Worth a line in pending.md if it matters at go-live.
2. **Cancel's length check counts the recorded words** (`statuses.go:100`: "Procedure cancelled: " + details), while the UI lets Details reach 1,000 characters (`StatusControl.tsx:164`). Details of 980+ characters are refused with "too long". `len()` also counts bytes, not characters (`statuses.go:160`), so accented text reaches the limit sooner. Edge cases only.
3. **Two quick ASA saves on the same page**: the second is sent with the revision the page loaded, so if it's pressed before the first save's reload returns, it gets the stale message for the user's own change (`PoaSummary.tsx:254`). It's safe: the value is kept, with Reload. Locally the window is a few milliseconds.
4. **A patient's save racing a cancel** can still write answers to the just-cancelled episode: `loadForm` checks the status, but `SavePatientAnswers` checks only that the form is open. The window is tiny and the episode is read-only afterwards. Closing the HQ form on cancel would also remove the many `status <> 'cancelled'` checks (code-review's "altitude" note); consider it in S22 alongside archive.
5. **Small wording gaps:** Admit's and Discharge's optional reason is in the status history but not in the General notes line (`statuses.go:240`, `:251`), and a Complete HQ review that loses a race with a cancel says "already complete" (`validation.go:141`).
6. **Episodes made before migration 00021 have no status history** (documented in the build notes and decisions.md). No real data exists, so this is fine.

### Checks

- **Scope:** everything in the brief is done: the table in Go (all pairs × moves × roles unit-tested), status events (`episode_status_events`), the revision (both pending.md §1a items closed), the three routes plus `POST …/undo` (needed by answer 2), the dates, the cancel reasons, the patient's side of cancel, the status control, the history, and J9b. Nothing large is outside the brief.
- **Tests first:** every new route has the five cases, plus every illegal pair → 409, role → 403, cancel without a reason → 400 and stale → 409 (`statuses_integration_test.go`). "Wrong role" for status, cancel and undo is a clinician at the trust's other hospital, because every member is at least a clinician; re-open has a real 403.
- **Tenant isolation:** every route goes through `requireHospital` and `loadEpisode` (hospital-scoped). Checked live: Bea (Trust B) gets 404 on GET, status, cancel, undo and re-open for a Trust A episode, through both Trust A's path and her own.
- **Audit:** all four routes are wrapped in `audited` (`router.go:189–192`), with labels.
- **A-20:** the grep finds only comments, the config and tests, as before S20.
- **Security:** no secrets; no patient data in the new notes or logs; the patient router is unchanged except the cancelled refusal; `chapter.ForPatient` is untouched.
- **Docs:** plan C3 status and differences, plans README, sessions README, decisions.md (21 Oct S20 section), pending.md all updated. The incumbent-name grep prints nothing.

### Spot-checks (Claude in Chrome, "trk" profile, http://localhost:3000)

- As Cara Clinician, on a POA complete episode: `moves` offered the other four reviewed statuses and Cancel, with no re-open. `POST /reopen` → 403 "Only a super clinician or an admin can re-open the HQ review."
- **Two status changes at once from the same revision:** one 204, the other 409 `stale`. A later PATCH with the old revision → 409 `stale`. Undo without a reason → 400.
- **Repeated undo** as above, then no undo offered at the completed review. On the episode page, Cara saw the Episode status control and Cancel episode, and no Undo or Re-open.
- **Another trust:** Bea → 404 on all five calls (above).

### Test results

- `make lint`: passed.
- `make test`: Go all ok; Vitest 189 passed (19 files) + 116 passed (15 files); CDK 124 passed.
- `make test-integration`: every package ok (httpapi 14.9 s, apitest incl. `TestEpisodeStatusesMigration`).
- `make e2e` (my own API on :8092 with database `sj_v20`, Vite on :3102): **53 passed, 5 skipped** (J9b, `test_statuses.py` and J5 included).

### Hand checks for Rahul

- None.

### Suggested commits

1. `S20: every status, cancel, re-open and undo — the transition table, status history and episode revision` (code, migration 00021, unit, integration and e2e tests).
2. `docs: S20 verified — build notes, test plan and reports, answers, verification, decisions, plan and session status`.
