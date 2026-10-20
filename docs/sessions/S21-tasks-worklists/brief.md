# S21: Tasks, worklists and status messages

**Plan:** C3 ([c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md)), part 4 of 5.
**Requirements:** TSK-01, EPI-02, EPI-08 (a Should, included in C3 by Rahul on 20 Oct).
*Local phase: runs on Rahul's Mac only. No AWS. Messages are captured, not sent.*

**Depends on:** S20 verified.

**Ask Rahul first:**
1. **Who is a task assigned to?** The permission roles (clinician, super clinician, admin) aren't job functions, and "Anaesthetic review needed" needs one.
   - (a) A **team** from a fixed list: nurse, anaesthetist, admin.
   - (b) A named staff member.
   - (c) Both: a team, optionally a person.

   **Recommended: (a)** for now. Anyone at the hospital may complete any task.
2. **The worklists.** Recommended: fixed and built in (user-defined saved filters later):
   - Ready for review;
   - Not responding (S16's flag);
   - Anaesthetic review needed (an open anaesthetist task);
   - Procedure within N days and not Ready for admission;
   - On hold;
   - Open tasks.

   **Auto-triaged** comes with C6, when auto-triage exists.
3. **"Within 14 days":** a hospital setting (default 14), like the reminder days? **Recommended: yes.**
4. **Status messages (EPI-08):**
   - **Which statuses:** Ready for admission and Cancelled, each by text and email.
   - **Per-hospital switch:** each template has an on/off setting, **off by default**.
   - **Placeholders:** `{{firstName}}` and `{{hospital}}`, with `{{link}}` optional (today's templates require it).

   **Recommended: yes.**

## Build

- **`tasks`:** episode_id, title, team (or as answered), created_by (a user, or a rule's name for C6), done_by and done_at.
  - Routes: `GET/POST/PATCH …/episodes/{eid}/tasks`, each with the five integration cases. Audited.
  - **A Tasks panel** on the episode page: add, complete, re-open a task. Who did what is shown.
- **Worklists:**
  - `GET …/episodes?worklist=<key>`: one indexed query, scoped to the user's hospitals as today.
  - Integration tests: **each worklist returns exactly its episodes**, with seeded episodes that sit just inside and just outside each rule.
  - **The clinician home** shows the worklists as tabs with counts, beside today's "All" list and its filters.
- **Status messages:**
  - The templates and switches as answered. The settings screens (S10) show them with no new screen code.
  - Sent through the C2 messaging when the status changes; captured locally.
  - Recorded in `episode_messages` (a new kind) with failures shown as for invites (S17).
- **End-to-end:** add a task, find the episode in "Open tasks", complete the task, and it leaves the list.

## Not in this session

- Tasks created by rules (C6, e.g. "Book anaesthetic review").
- The Auto-triaged worklist (C6).
- Archive (S22): archived episodes leaving every worklist is tested there.

## Manual test focus

1. Add tasks for each team; complete one; re-open it. The panel shows who and when.
2. Each worklist tab shows exactly the right episodes. Seed or build one episode per rule, and one that just misses it (e.g. a planned date 15 days away).
3. Counts on the tabs match the lists, and change after a status move.
4. Cara (Hospital A1) never sees Hospital A2's episodes in any worklist; an admin of the whole trust sees both.
5. Switch on the "Ready for admission" message for Hospital A1, move an episode there, and the captured text and email use the hospital's wording. With the switch off, nothing is sent.
6. A template without `{{link}}` is accepted for status messages, but still refused for invites.
