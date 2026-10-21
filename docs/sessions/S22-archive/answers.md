# S22 answers

Asked by the conductor (`/run-sessions S18 S22`), 20 Oct 2026.

1. **Who can archive and un-archive by hand?** **A super clinician for both** (not the recommendation of a clinician archiving). Un-archive needs a reason (as in the brief).
2. **Read-only:** **yes** (recommended). Every write to an archived episode gets a 409 except un-archive. Archived episodes leave every worklist, and stay in search, on the patient page and in the All list behind "Show archived".
3. **"No activity":** **yes** (recommended). `episodes.last_activity_at`, moved forward by every write (answers, status, notes, tasks, messages sent).
4. **The daily job:** **yes** (recommended). Job kind `auto_archive`, one row per hospital, at 02:00 in the hospital's time zone, scheduling its own next run. "After the last PROMs form closes" waits for C9.
5. **Closing a patient account (EPI-09):** **yes** (recommended), as in the brief.
