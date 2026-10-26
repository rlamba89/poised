# S40: ASA rules in the HQ

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), part 3 of 8.
**Requirements:** ASA-02, ASA-03, ASA-04 ([saas-requirements.md](../../saas-requirements.md) §8); A-8 (§0); PRV-05 ([requirements.md](../../requirements.md)).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S39 verified.

**What this session does:** rules are written, stored, previewed and tested. **Nothing acts on them yet**: S42 and S43 apply them when a patient sends the HQ.

**What exists today:**
- A questionnaire version (`questionnaire_versions`, migration 00002) holds only its chapters; it has no settings of its own. "New version" (`createVersion` in `internal/httpapi/questionnaires.go`) copies the chapters. Published versions are read-only.
- The editor (`features/editor/QuestionnaireEditor.tsx`) has no HQ-level settings page; its header has Translate and Publish.
- Test cases are saved per Question Set (`packages/clinical/src/testcases.ts`), with expected lines such as "Suggested ASA III"; `publishProblems` (`publish.ts`) blocks publishing while any fails, in the browser. Go's publish checks only question keys.
- Review flags (amber, red) exist on disclosures (`ClinicalOutput.flag`).
- Agreed defaults (A-8, 2 Oct): suggested ASA 1–2 and no red flags → auto-triage straight to Ready for admission; ASA 3 or more → task "Book anaesthetic review", a flag, and the anaesthetist's ASA required before Ready for admission.

**Ask Rahul first:**
1. **Questionnaires made before rules exist.** When someone makes a new version of one, does it get the default rules?
   - (a) **Yes.** The new draft gets the default rules, which the author sees on the Rules page and can change before publishing.
   - (b) No. It has no rules until an author adds them.

   **Recommended: (a).** Every HQ then behaves the same; nothing changes until a person publishes.
2. **A patient whose answers give no ASA grade at all** (for example, a fit and well patient who ticks "No" to everything).
   - (a) **Count it as ASA I,** so the "ASA 1–2, no red flags" rule can auto-triage them.
   - (b) Never auto-triage them: a nurse reviews.

   **Recommended: (a).** A fit patient's answers usually give no grade; the Auto-triaged list is the quick human check.
3. **Rules and the nurse's ASA.** A rule could also look at the grade a nurse types in later, e.g. "when the nurse gives ASA III, add Book anaesthetic review".
   - (a) **Not now.** Rules run once, when the HQ is sent, on the suggested ASA.
   - (b) Yes. Rules also run again each time a nurse saves an ASA grade.

   **Recommended: (a).** Simpler; a nurse can add the task by hand.
4. **Test cases and rules.** Test cases are saved per Question Set, but rules look at the whole questionnaire.
   - (a) **A set's test case also lists the rules its answers alone would fire,** e.g. "Rule fires: High risk".
   - (b) Add separate test cases for the whole questionnaire.

   **Recommended: (a).** It reuses what authors already know.

## Build

- **The rule model** (`packages/clinical`, test-first; explain the TypeScript for Rahul): a rule has a name, a **when** and a **then**.
  - **when:** the suggested ASA compared with a number (`>=`, `<=`, `=`), plus optional extras: no red flags, a score band (`band('<calc>')` is one of …), a disclosure code produced, the episode's procedure;
  - **then:** a fixed list of actions: **auto-triage** (to a target status: Ready for admission by default, or Ready for POA or POA complete), **add a task** (title and team, as S21's tasks), **add a flag** (a name), **require the anaesthetist's ASA**, **suggest a status**;
  - **the matcher**: `firedRules(result, context)` returns the rules that fire and their actions, from the same result `computeOutputs` and S39's whole-HQ suggested ASA give. Unit tests for each kind of "when" and each action, and the defaults on both sides of ASA 2/3.
  - Question 2's answer for "no grade".
- **Storage (Go stores and checks the shape only; it never evaluates):** `questionnaire_versions.rules jsonb` (next migration). Go's `internal/asarules` (or similar) holds **the platform default rules** as a JSON file and checks a rule list's shape (known actions, a target from the allowed list, a team from `episode.Teams`, a task title as `episode.TaskInput`). A new HQ gets the defaults; "New version" copies the rules (question 1 for old ones). The seed's Demo HQs get the defaults.
- **Routes**, each with the five integration cases (not audited: no patient data):
  - the rules come with the questionnaire (`GET …/questionnaires/{qid}`);
  - `PUT /api/o/{oid}/h/{hid}/questionnaires/{qid}/rules`: the draft's rules, for whoever may edit the HQ's chapters (`canEdit` in `internal/httpapi/chapters.go`); refused on a published version (409) and for a bad shape (400, plain words). Stale writes get 409 (a revision, as chapters have).
- **The Rules page:** a **Rules** button in the editor header opens a page listing the rules in plain words ("When the suggested ASA is 1 or 2 and there are no red flags: auto-triage to Ready for admission"), with add, edit, delete and reorder. Read-only for those who can't edit, and on a published version.
- **Preview:** under Suggested ASA, "Rules that fire" for the current answers (the set alone).
- **Test cases (question 4):** expected lines can include the rules that fire; a changed rule makes a saved case fail, which blocks Publish (the browser's `publishProblems`; S41 makes the server check it too).
- **Publish window:** says in one line what this version will do automatically ("This version auto-triages: ASA 1–2, no red flags → Ready for admission").
- **E2E:** J11 starts (`e2e/journeys/test_j11_triage.py`, its authoring half): Alex opens Rules on a new HQ, sees the two defaults, changes the target to Ready for POA and back; a test case expecting "Rule fires: Low risk" passes, and fails (Publish blocked) after the rule's ASA limit is changed.

## Not in this session

- Running rules on the server and acting on them (S41–S43).
- Rules on the nurse's ASA (question 3); trust-level rules shared by several HQs.
- A superadmin screen for the platform defaults (they're a file).

## Manual test focus

1. A new HQ shows the two default rules in plain words. Sue Super (hospital-only) sees them read-only; Alex edits them.
2. Add a rule with each kind of "when" and each action; a bad one (a task with no title) is refused with a plain message.
3. Preview: answers giving ASA II with no red flags show "Low risk" firing; add a red flag and it stops.
4. A test case expecting a rule fails after the rule changes, and Publish is blocked.
5. Publish: the window says what the version will do automatically. Afterwards the rules are read-only; New version copies them.
6. Trust B can't read or change Trust A's rules.
