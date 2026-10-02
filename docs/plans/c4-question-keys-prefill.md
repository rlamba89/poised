# C4: Question keys and pre-fill

**Goal:** a question the patient answered before, in any earlier HQ, is filled in for them, and they only confirm it.

**Requirements:** PRE-01…06, §6, A-9 (answers never expire). **Depends on:** C3.

## Data

- `library_questions`: org_id NULL (platform) or a trust, key, SurveyJS element JSON, tags. Keys are readable, e.g. `smoker`, `snoring`, `ponv_history`.
- Element properties in the chapter JSON:
  - `name` is the key (already the stable ID);
  - `alwaysFresh: true`;
  - `linkedFrom` (for information only).
- `episode_answers.prefill jsonb`: `{key: {fromForm, answeredAt}}`.
- Patient consent for cross-trust pre-fill: stored per account and trust.

## Design

- **Keys survive** new versions, copies and library inserts. Library inserts **keep** their key, which changes PNL-06; update [requirements.md](../requirements.md).
- **"Same as…":**
  - the editor searches the keys in the trust's published HQs and the library;
  - adopting a key needs a compatible type and option values (`keys.Compatible`, written TDD);
  - once a keyed question is published, **its option values are locked**, and the publish checks enforce it.
- **Pre-fill:**
  - When the patient opens a form, Go walks their earlier **submitted** forms, newest first, across the records linked to their account.
  - It prefers the clinician-validated answer over the patient's.
  - It skips clinician-only and `alwaysFresh` questions, and Profile.
  - Other trusts are used only if both trust settings allow it **and** the patient has agreed.
  - This is a plain JSON walk: Go still never evaluates SurveyJS.
  - Repeating panels are copied row by row.
- **Patient UI:**
  - A summary card per set: "We've filled in N answers from 3 Mar 2026", with **Check each answer** or **Everything is still correct**.
  - Each pre-filled answer has a marker.
- **Clinician UI:** a marker on pre-filled answers, plus "changed by patient" where they edited one.

## Tests

- **Unit:**
  - the key walk over answer JSON, including dynamic panels;
  - `keys.Compatible`;
  - the precedence (validated over patient; newest first);
  - the skip rules.
- **Integration:**
  - pre-fill from the same trust;
  - pre-fill from another trust only after consent;
  - clinician-only and fresh questions are never pre-filled;
  - publishing is blocked when a keyed question's option values have changed;
  - prefill metadata is stored.
- **End-to-end:** **J9**. A patient with an earlier episode opens a new HQ, sees the pre-filled answers, chooses "Everything is still correct", answers the rest and submits. The clinician sees the pre-filled markers.

## Status

Not started.
