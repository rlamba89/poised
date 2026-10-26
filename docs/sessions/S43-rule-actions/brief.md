# S43: Tasks, flags and the anaesthetist's ASA from rules

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 6 of 8.
**Requirements:** ASA-02 (the high-risk rule), ASA-03 (the actions), TSK-01 (rule-created tasks), EPI-02 (Anaesthetic review needed) ([saas-requirements.md](../../saas-requirements.md) §5, §8); A-8.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S42 verified.

**What exists today:**
- **Tasks** (S21, migration 00022): `tasks.created_by_rule` already exists for "a task a rule made (C6)" (exactly one of `created_by` and `created_by_rule`). The Tasks panel (`features/episodes/TasksPanel.tsx`) and its routes (`internal/httpapi/tasks.go`). The **Anaesthetic review needed** worklist is any open task for the anaesthetist team.
- **Statuses:** a clinician moves between the reviewed statuses by hand (`Set`), including to Ready for admission (`internal/episode/episode.go`; `changeStatus` in `internal/httpapi/statuses.go`). Nothing blocks it.
- **The ASA grades:** `episodes.nurse_asa` and `anaesthetist_asa` (1–6), saved from the POA Summary (`PoaSummary.tsx`, `AsaGrades`) through the episode PATCH with its revision.
- After S42, the submit applies the auto-triage action from the stored result (`form_outputs`); the other actions are stored but ignored.

**Ask Rahul first:**
1. **Clearing a flag.** The high-risk rule puts a flag such as "Anaesthetic review" on the episode, shown as a badge on the episode and in the lists.
   - (a) **Any clinician clears it with one click;** who and when are recorded.
   - (b) It clears itself when the rule's task is done.

   **Recommended: (a).** Clear and simple; the task and the flag can mean different things.

## Build

- **On submit** (in the same transaction as S42's moves), Go applies each fired action from the stored result, test-first (a Go unit test turns a result into the writes; integration tests check the rows):
  - **add a task:** a row in `tasks` with `created_by_rule` = the rule's name and the rule's title and team. The Tasks panel says "Added by rule: High risk". No duplicate if the same rule fires again later (re-open and re-send);
  - **add a flag:** **`episode_flags`** (next migration: episode, name, the rule, created at, cleared by, cleared at). Shown as a badge on the episode page and the episode list rows;
  - **require the anaesthetist's ASA:** recorded on the episode (a column or `episode_requirements`). While it's unmet, a move to **Ready for admission** is refused (409, "Add the anaesthetist's ASA grade first.") and the status control says why. Saving an anaesthetist grade meets it. If an auto-triage rule targets Ready for admission at the same time, the episode goes to Ready for review instead (S42);
  - **suggest a status:** shown as a line on the episode ("The HQ's rules suggest: Ready for POA"); nothing changes by itself.
- **Clearing a flag (question 1):** `POST /api/o/{oid}/h/{hid}/episodes/{eid}/flags/{fid}/clear` (audited, `flag.clear`), any clinician at the hospital, with the episode revision; a General notes line. Five integration cases, plus clearing twice (409) and a stale revision (409).
- **The episode's flags** come with the episode (`GET …/episodes/{eid}`) and the list (a column on each row); no new read route.
- **Integration tests** (fake evaluator): a high-ASA result → the task (with its rule), the flag, the requirement, and Ready for review; Ready for admission refused until the anaesthetist's grade is saved, then allowed; the fake failing → none of them. Each of the five cases for every changed route.
- **E2E:** J11b (`e2e/journeys/test_j11b_high_risk.py`): a high-risk patient (phone) sends the HQ; Cara sees a "Book anaesthetic review" task added by the rule, the flag, and the episode on Anaesthetic review needed; Ready for admission is refused until the anaesthetist's ASA is saved. Update `e2e/README.md`.

## Not in this session

- Running rules again later (on the nurse's grade, on complete review).
- A worklist per flag name (the anaesthetist's tasks already have one).
- Server checks on send and complete review (S44).

## Manual test focus

1. A high-risk patient sends the HQ: the task (added by rule), the flag badge, the requirement, and Ready for review.
2. Try Ready for admission before the anaesthetist's grade: refused with a plain reason; save the grade, then it works.
3. Clear the flag: the badge goes, with who and when in the notes. Val (Hospital A2) can't clear Hospital A1's.
4. A rule that suggests a status shows the line and changes nothing.
5. Re-open and re-send: the rule's task isn't added twice.
