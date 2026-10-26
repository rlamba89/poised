# S42: Auto-triage and the Auto-triaged worklist

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 5 of 8.
**Requirements:** ASA-02 (the low-risk rule), A-8 ([saas-requirements.md](../../saas-requirements.md) §0, §5, §8); EPI-02 (the Auto-triaged worklist), EPI-03 (automatic changes name their rule), EPI-07 (re-opening an auto-triage); OPS-04 (the hazard-log lines).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S41 verified.

**What exists today:**
- The transition table (`internal/episode/episode.go`): a send moves HQ not complete → Ready for review (`Submit`). A super clinician can re-open from Ready for POA, POA complete, On hold, Not ready or Ready for admission back to Ready for review (`Reopen`, with a reason).
- The status history `episode_status_events` (migration 00021) already has a `rule` column for automatic changes ("NULL now"); its `move` CHECK lists today's moves (00023). The General notes write "Automatic (…)" lines for the jobs (e.g. auto-archive).
- The worklists are one SQL function, `in_worklist` (migration 00022), and `episode.Worklists` (`internal/episode/tasks.go`, whose comment says "Auto-triaged comes with auto-triage (C6)"); tabs on the clinician home (`features/episodes/EpisodeList.tsx`).
- Status messages to the patient (EPI-08, S21): `episode.StatusMessage` sends the hospital's "readyForAdmission" message when a clinician sets Ready for admission, if the hospital switched it on.
- After S41, every send stores the evaluator's result (`form_outputs`), including the rules that fire.

**Ask Rahul first:**
1. **How an episode leaves the Auto-triaged list.**
   - (a) **A clinician opens it and clicks "Checked"** (their name and the time are recorded); it leaves the list.
   - (b) It stays on the list until its status changes.

   **Recommended: (a).** The list is the quick human check after an automatic decision; (b) would keep checked patients on it for weeks.
2. **The patient's "You're ready for your operation" message** (if the hospital has it switched on), when the episode is auto-triaged to Ready for admission.
   - (a) **Send it only after a clinician clicks Checked.**
   - (b) Send it straight away, as when a clinician sets the status.

   **Recommended: (a).** No patient hears "you're fit for surgery" before a person has looked.

## Build

- **The move, test-first (Go unit tests):** a new move `auto_triage` in the transition table, from HQ not complete to the rule's target (Ready for admission, Ready for POA or POA complete), made only by the system (no role may make it by hand; it isn't in `Options`). `TestCheckEveryPair` grows with it. Migration: the `move` CHECK gains `auto_triage`.
- **On submit,** when the stored result has an auto-triage action (and the evaluator didn't fail): in the same transaction, record Submit, then the auto-triage move to the target, with `rule` = the rule's name, `user_id` NULL, and a General notes line "Automatic (Auto-triaged: ASA II, no red flags → Ready for admission)". Both submit routes (the patient, and staff on their behalf). **If the evaluator failed, or no auto-triage rule fired, nothing changes** (Ready for review, as today). If a rule also requires the anaesthetist's ASA (S43), auto-triage to Ready for admission doesn't happen.
- **The stamp:** every Question Set of an auto-triaged episode shows "Auto-triaged (ASA II, no red flags)" on the episode page and the POA Summary. The POA Summary of an auto-triaged episode shows the patient's answers (there's no validated copy).
- **Re-open (EPI-07):** the existing re-open from the target status back to Ready for review works for an auto-triaged episode (super clinicians, with a reason); integration test.
- **Checked (question 1):** `POST /api/o/{oid}/h/{hid}/episodes/{eid}/triage-checked` (audited, `triage.checked`), any clinician at the hospital, with the episode revision; records who and when (next migration), a General notes line, and (question 2) sends the status message. Five integration cases, plus: only an auto-triaged, unchecked episode (409 otherwise), and a stale revision (409).
- **The worklist:** **Auto-triaged** in `in_worklist` and `episode.Worklists` (help line: "Moved on automatically by the HQ's rules; not yet checked by a clinician"), counted like the others. A cancelled or archived episode is in none.
- **Integration tests** (fake evaluator): low ASA → the target status, one Submit and one auto-triage event with the rule's name, the note, on the worklist; the fake failing → Ready for review, no auto-triage; a result with no auto-triage action → Ready for review; staff on-behalf send auto-triaged the same way.
- **Hazard log (OPS-04, C10 writes it):** add the auto-triage lines to pending.md's hazard-log list (an episode goes to Ready for admission with nobody involved; the mitigations: rules signed off and tested with the HQ, the Auto-triaged list, Checked, re-open, and the fail-safe).
- **E2E:** J11's patient half (`e2e/journeys/test_j11_triage.py`): a low-risk patient (phone) sends the HQ; Cara sees the episode on **Auto-triaged** as Ready for admission, with the stamp and the history naming the rule; she clicks Checked and it leaves the list. Update `e2e/README.md`.

## Not in this session

- Tasks, flags, the anaesthetist's ASA requirement and suggested statuses from rules (S43).
- Auto-triage when a clinician completes a review, or re-running rules later.

## Manual test focus

1. A low-risk patient sends the HQ: the episode is Ready for admission straight away, on Auto-triaged, with "Automatic (Auto-triaged: …)" in the history and the stamp on each set.
2. A high-risk patient: Ready for review, not on Auto-triaged.
3. Stop the evaluator and send a low-risk HQ: Ready for review, with the "automatic checks didn't run" note.
4. Checked: the episode leaves the list; the patient's message goes then (if switched on), not before. Cara at Hospital A1 can't check Hospital A2's episode.
5. Sue Super re-opens an auto-triaged episode with a reason: back to Ready for review; Validate works.
6. Change the HQ's target to Ready for POA, publish, and send: it lands on Ready for POA.
