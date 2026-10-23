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
- **End-to-end:** **J16** (J9 is C3's lifecycle, and C5–C9's plans claim J10–J15). A patient with an earlier episode opens a new HQ, sees the pre-filled answers, chooses "Everything is still correct", answers the rest and submits. The clinician sees the pre-filled markers.

## Status

**Done** (23 Oct 2026). **S26 verified** (23 Oct 2026, [verification 2](../sessions/S26-prefill-clinician-cross-trust/verify.md); [test report 3](../sessions/S26-prefill-clinician-cross-trust/test-report-3.md): 14 of 14 passed after verification 1's fixes; another trust's answers match by key only for a platform library key, else by exact wording, choices always by option text (Rahul, after verification 1); earlier test reports 1 (20 of 20) and 2 (8 of 8); minor leftovers in pending.md; [build notes](../sessions/S26-prefill-clinician-cross-trust/build-notes.md): the clinician's markers in Validate and the POA Summary, other trusts' answers with both settings and the patient's Yes, once per trust, never on their behalf, audited under the source trust; S25's three "for S26" items done; differences from the plan there). **S25 verified** (23 Oct 2026, [verification 1](../sessions/S25-prefill/verify.md); [test report 2](../sessions/S25-prefill/test-report-2.md): 19 of 19 passed after test report 1's fixes; minor leftovers in pending.md, some for S26; [build notes](../sessions/S25-prefill/build-notes.md): pre-fill by key then wording, the opening card and markers, the seeded Demo follow-up HQ; differences from the plan there). **S24 verified** (23 Oct 2026, [verification 3](../sessions/S24-question-library/verify.md); test report 4 all passed after verification 2's fix; minor leftovers in pending.md; [build notes](../sessions/S24-question-library/build-notes.md): the library, "Same as…", the medication type frozen; differences from the plan there). **S23 verified** (22 Oct 2026, [verification 3](../sessions/S23-question-keys/verify.md); [test report 5](../sessions/S23-question-keys/test-report-5.md): 23 of 23 passed, after verification 1's and 2's fixes and test report 3's R-05 fix; [build notes](../sessions/S23-question-keys/build-notes.md)). Split into sessions on 21 Oct 2026. The briefs are in [docs/sessions/](../sessions/README.md):
- **S23:** question keys in the editor: unique across an HQ, option values locked at publish (enforced in Go), "Always ask fresh", the "Pre-fills from N other HQs" label;
- **S24:** the question library (platform and trust) and "Same as…", with `keys.Compatible`; PNL-06 changed;
- **S25:** pre-fill from the same trust, and the patient's opening card and markers (PRE-01–03, PRE-06); J16's patient half;
- **S26:** the clinician's markers (PRE-04) and pre-fill from another trust with the patient's agreement; J16 completed, J16b.

**Answered 22 Oct** (each session's `answers.md`; [decisions.md](../decisions.md) §2). Two differences from this plan: **pre-fill matches by key, then by exact wording** (with safeguards), and a published keyed question's **options are frozen completely** (values and wording).

**Differences from the plan in S23** (build notes): the editor's route is `GET …/h/{hid}/questionnaires/{qid}/question-keys` (not `/api/o/{oid}/question-keys?questionnaire=`) and is re-fetched after every save; the editor's placeholder texts ("Text question", "Group"…) never match by wording; after verification 1 the lock freezes the whole answer (kind, unit, options, rows, columns, scale; Rahul), signatures, uploads and the BMI block never pre-fill, and the key checks are Go's only (the Publish window shows them from the route); after verification 2 a date's format and a text question's input format are frozen too.

Proposed in the briefs, confirmed 22 Oct:
- **The prefill map also stores the copied value** (`{key: {fromForm, answeredAt, value}}`), so "changed by patient" needs no second read. The plan says `fromForm`; [saas-requirements.md](../saas-requirements.md) §6 said `fromEpisode`, which predates `episode_forms` (S18).
- **The keys are found by walking the trust's published chapters' JSON**, with no `question_keys` table, until that's slow.
- **`keys.Compatible` is Go only**; the editor asks a Go route instead of keeping a TypeScript copy.
