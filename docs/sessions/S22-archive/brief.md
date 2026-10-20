# S22: Archive, auto-archive and closing a patient account

**Plan:** C3 ([c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md)), part 5 of 5.
**Requirements:** EPI-04, EPI-05, EPI-09 (placed here by Rahul on 20 Oct), and the plan's J8 journey, named **J9** here because `test_j8_reminders` already took J8.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S21 verified.

**Ask Rahul first:**
1. **Who can archive and un-archive by hand?** The status table (saas-requirements.md §5.2) says "automatic, or a clinician". **Recommended:** a clinician can archive, from any status. Un-archive needs a super clinician, with a reason.
2. **What "read-only" covers:** every write to an archived episode gets 409 except un-archive: status, answers, notes, tasks, dates and Resend. **Recommended: yes.** Archived episodes leave every worklist, and stay in search, on the patient page, and in the All list behind a "Show archived" filter.
3. **"No activity" for M months** (EPI-05, pre-op episodes). **Recommended:** an `episodes.last_activity_at` that every write to the episode moves forward: answers, status, notes, tasks and messages sent.
4. **The daily job.** **Recommended:** a new job kind `auto_archive` with one row per hospital. It runs at 02:00 in the hospital's time zone and schedules its next run itself. The timer (locally) and the jobs Lambda (on AWS) already run every due job, so nothing new is scheduled. "After the last PROMs form closes" waits for C9.
5. **Closing a patient account (EPI-09):**
   - "Close my account" sits on the patient home, behind a confirmation;
   - it removes the login: the account, its sessions and its sign-in links;
   - it unlinks the trust's patient records, which stay with every episode;
   - a later invite creates a new account as usual (S15).
   - If an HQ is unsent, the confirmation warns that it won't be sent.

   **Recommended: yes.** Staff see "The patient closed their account on …" in the episode's notes.

## Build

- **Archive (EPI-04):**
  - `archived_from` holds the previous status, and un-archive restores it **exactly**.
  - `POST …/archive` and `POST …/unarchive`, each with the five integration cases plus **every write to an archived episode → 409** and **un-archive restores the exact previous status**. Audited.
  - Archiving goes through S20's transition table and writes a status event.
- **Auto-archive (EPI-05):**
  - A pure decision function, `due(episode, settings, now) → archive?`, unit-tested test-first, covering:
    - N days after discharge;
    - M months without activity for pre-op episodes;
    - the hospital's override of both;
    - a just-not-due case.
  - The job as answered. It reuses S16's runner and the dev/QA "run jobs as of" page.
  - The status event names the rule ("Auto-archive: 30 days after discharge").
- **Closing an account (EPI-09),** as answered. Integration tests:
  - the account's rows go;
  - the patient records and episodes stay;
  - the old sign-in links fail;
  - another patient's account is untouched.
- **End-to-end J9, the full lifecycle:** create with a procedure → patient submits → review → Ready for admission → admitted → discharged → archive → un-archive.

## Not in this session

- Retention and deletion after 8 years (EPI-06, a Should; not in C3's plan).
- Archiving when PROMs close (C9).

## Manual test focus

1. Archive an episode by hand. It leaves every worklist, can't be changed (each control is disabled, and the API refuses), and is still found by search and on the patient's page.
2. Un-archive as a super clinician: it's back in its exact previous status and worklist. A clinician can't un-archive.
3. Run jobs as of +31 days after a discharge: the episode is archived automatically, and its history names the rule. At +29 days it isn't.
4. A pre-op episode untouched for 6 months is archived; one with a note added last week isn't.
5. A hospital overrides the days (e.g. 7). Its episodes follow the override, and the trust's other hospital keeps 30.
6. As a patient: close the account. The home is gone, an old sign-in link fails, and staff still see the patient and episodes with the note. A new invite starts a fresh account.
