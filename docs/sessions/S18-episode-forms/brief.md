# S18: Episode forms

**Plan:** C3 ([c3-episode-lifecycle.md](../../plans/c3-episode-lifecycle.md)), part 1 of 5.
**Requirements:** none new. This is the reshape the plan's Data section asks for ("`episode_answers` moves from per-episode to per form"), so C7 (assessments) and C9 (PROMs) can add more forms to one episode.
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S17 verified (plan C2 done).

**Why first (Rahul, 20 Oct):** C4–C6 add more code that reads answers, so reshaping now means less to change later.

**Ask Rahul first:**
1. **What does an `episode_forms` row hold now?**
   - The plan lists: episode_id, version_id, kind `hq|assessment|proms`, due_at, closes_at, status, submitted_at.
   - Only the HQ exists until C7/C9.
   - **Recommended:** id, episode_id, version_id, kind (the CHECK allows only `hq` for now), status, submitted_at. C7 and C9 add their kinds, and due_at and closes_at, when they use them. Columns nobody reads are just guesses about the future.
2. **Existing data: migrate it, or reset?** QA on AWS (Neon) and every developer's database hold made-up episodes.
   - (a) A goose migration creates one HQ form per episode and moves its answers onto that form, so nothing needs resetting.
   - (b) Reshape the tables and re-seed (the plan's "a fresh app, a reshape not a migration").

   **Recommended: (a).** It's a few SQL statements, and QA keeps working after the pipeline deploys it.
3. **Routes:** keep today's URLs (`…/episodes/{eid}/hq`, `…/answers/{cid}`, the patient's `/api/p/episodes/{eid}/…`), which resolve the episode's one HQ form on the server.
   - Routes with a form ID come with the second kind of form (C7/C9).
   - **Recommended: yes.** It keeps this session to a change inside the API, with no URL churn in the web app or the journeys.

## Build

- **`episode_forms`**, as answered.
  - Creating an episode creates its HQ form in the same transaction.
  - `episodes.version_id` moves to the form.
  - The patient's submit time (`patient_submitted_at`) moves to the form's `submitted_at`.
- **`episode_answers`** is keyed by `(episode_form_id, chapter_id, actor)`. The answer revisions (S02a) are kept as they are.
- **Every query that reads the HQ version or the answers** goes through the form. The code map (20 Oct) found about 12 queries: in `episodes.sql`, `accounts.sql` (the patient home), `patients.sql` and `jobs.sql` (the follow-up's "started" check). Plus `internal/apitest/fixtures.go`.
- **The API's JSON stays the same** where it can. The episode still reports its HQ version, so the web app changes as little as possible.
- **A migration test** (if answered (a)): a database holding episodes and answers in the old shape is migrated, and each episode gets one HQ form with its answers, revisions and submit time intact.
- **Every existing test passes unchanged in meaning:** `make test`, `make test-integration`, and `make e2e` (J1–J8 and the others).

## Not in this session

- Procedures (S19); statuses (S20).
- Any form kind except the HQ; due and close dates.
- Routes with a form ID.

## Manual test focus

There's nothing new to see. This is a **full regression** of the episode workflow:
1. Create an episode, invite the patient, and fill in the HQ as the patient at phone size, sign-in included. Leave halfway and come back: the answers are still there.
2. Fill in an HQ on the patient's behalf.
3. Submit. The episode is Ready for review, and the patient home shows it as sent.
4. Validate each Question Set, then complete the review. The POA Summary shows the patient's answers and the clinician's changed answers, marked.
5. A stale save in a second tab still gets "changed somewhere else" (S02a).
6. Reminders stop after submit (S16's "run jobs as of" page).
7. A patient with episodes at two trusts still sees both on their home.
