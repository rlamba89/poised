# S23: Question keys in the editor

**Plan:** C4 ([c4-question-keys-prefill.md](../../plans/c4-question-keys-prefill.md)), part 1 of 4.
**Requirements:** PRE-05 (the "Pre-fills from N other HQs" / "New question" label and "Always ask fresh"), §6 points 1, 2, 4 and 5 of [saas-requirements.md](../../saas-requirements.md#6-pre-filling-from-earlier-hqs-agreed-need-proposed-design).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S22 verified (plan C3 done).

**Why first:** pre-fill (S25) is only safe if a key means the same thing everywhere it's used. This session makes keys trustworthy before anything copies answers by them.

**Ask Rahul first:**
1. **Where Go finds a trust's keys.** The label, the option lock and "Same as…" (S24) all need "every question with this key in the trust's HQs".
   - (a) Walk the JSON of the trust's published and retired chapters on each request. No new table.
   - (b) A `question_keys` table filled at publish, plus a backfill for what's already published.

   **Recommended: (a).** A trust has tens of HQs, so the walk is cheap, and there's nothing to keep in step. Add a table only if it gets slow.
2. **What "option values are locked" means** (§6 point 4). Option values are generated IDs that authors never type, so the real risks are removing a value and re-using it.
   - (a) Once a keyed question is published, a later publish (any HQ in the trust with that key) may **add** options and **reword** them, but never **remove** a published value.
   - (b) The set of values must stay exactly the same.

   **Recommended: (a).** A removed option would be hidden with a condition instead. Old answers keep their meaning, and authors can still extend a list.
3. **Keys unique across the whole HQ** (§6 point 1). Today a name is unique only within its Question Set, and Go doesn't check.
   - **Recommended:** the editor refuses a duplicate across the HQ's sets, and Go's publish refuses one too, naming both sets.
4. **"Always ask fresh"** (`alwaysFresh: true`): a switch in the question's settings, beside Clinician only, with a badge on the card.
   - **Recommended: yes**, offered on every answerable question. Clinician-only questions don't need it (they're never pre-filled).
5. **The PRE-05 label.** On each answerable patient question: "Pre-fills from 2 other HQs" (it opens a list of their names), or "New question".
   - "Other HQs" counts the trust's other questionnaires (not versions) with a published or retired version holding the key.
   - **Recommended: yes**, fetched once when the editor opens, from one Go route.

## Build

- **`internal/keys`** in Go, test-first:
  - a walk that lists every keyed question in a chapter's JSON (name, type, title, option values, clinician-only, `alwaysFresh`), at any depth, including dynamic-panel children;
  - the lock check from answer 2: given the trust's published questions and a draft's, list each removed value;
  - the duplicate-key check from answer 3.
- **Publish (Go):** `publishQuestionnaire` runs both checks between loading the chapters and publishing, and refuses with 422 and a list of problems. The editor's Publish window shows them as it shows `publishProblems` today.
- **The editor runs the same two checks before Publish**, so the author sees them early. `publishProblems` gets the trust's published keys passed in.
- **`alwaysFresh`:** a hidden SurveyJS property (as `questionTypes.ts` does for the others), kept when a question changes type, with its switch and badge.
- **A route for the keys:** `GET /api/o/{oid}/question-keys?questionnaire=` returns, per key, the other questionnaires that use it and its published option values. Super clinician and above, the five integration cases. No patient data, so not audited.
- **Keys survive "New version"** (they already do; add an integration test that proves it).
- **Integration tests:**
  - publishing is blocked when a keyed question has lost a published value, in the same HQ's new version and in a different HQ of the trust;
  - publishing is blocked by a key used twice in one HQ;
  - another trust's HQs never count, for the label or the lock.

## Not in this session

- The question library and "Same as…" (S24).
- Pre-fill itself (S25).
- Copying a whole HQ (FRM-05, not built). Copying a question or a Question Set inside an HQ still gives new IDs, which is right: keys must stay unique in an HQ.

## Manual test focus

1. Publish an HQ, make a new version, delete an option from a published question, and Publish: refused, naming the question and the option. Add an option instead: publishes.
2. Reword an option and a question title: publishes.
3. Two Question Sets with the same key (made by hand in the JSON, or via the API, since the editor never makes one): Publish is refused, naming both sets.
4. Switch on Always ask fresh: the badge shows, it survives a save, a reload, a type change and a new version.
5. A question also used in another published HQ of the trust says "Pre-fills from 1 other HQ" and lists it. A brand-new question says "New question". Trust B's HQs never show up for Trust A.
6. The patient view and the POA Summary look exactly as before.
