# S44c: C6's last open items (Rahul's answers of 28 Oct)

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), a follow-up session added on 28 Oct, before S45.
**Requirements:** the open items S44a and S44b left for Rahul in [pending.md](../../pending.md), answered in chat on 28 Oct ([answers.md](answers.md)).
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S44b verified.

**Ask Rahul first:** nothing. Every item below was answered on 28 Oct. If an item turns out bigger or different from what's written here, ask (questions.md), in simple English with a short example (answers.md, last line); don't widen it.

## Build (each part test-first: the failing test, then the code)

### 1. The patient's form clears an answer when its question is hidden (answer 1)

- The patient's form, and staff's Fill in for the patient, clear a question's answer as soon as a condition hides it (the question, its page or its whole Question Set), so what the patient sees is what's saved and sent. Check the SurveyJS docs first (`clearInvisibleValues: "onHidden"`, or the documented equivalent); say in the build notes how it treats a pre-filled answer in a question that's hidden when the form opens.
- The save after a clear must reach the server like any other change (autosave), so a reload shows the answer gone.
- S44a's server-side "keep only what the form showed" on send stays as the backstop.
- The chain case in a test: B Yes → A Yes → C answered → B No: A and C are cleared on screen and in the saved row; B Yes again shows A and C empty.

### 2. A held auto-triage message stays held after Undo or un-archive (answer 2)

- With the HQ's "wait for Checked" on: when Undo or un-archive brings an auto-triaged episode back to its auto-triaged status, not yet Checked, the message is held again, so Checked sends it once. S44a's fix stands: a status set by hand (which sends its own message) still ends the wait, so nothing goes twice.
- Integration tests: auto-triaged, held → On hold → Undo → Checked sends it once; the same with archive → un-archive; On hold → Ready for admission by hand → Checked sends nothing (S44a's case, unchanged).

### 3. A High STOP-Bang holds back the default Low risk auto-triage (answer 3)

- In the platform defaults (`internal/asarules/defaults.json`): a High STOP-Bang stops the "Low risk" auto-triage, so the episode stays at Ready for review. Builder's choice how, in the simplest way the rule shapes allow (today a rule can test "a score is in these bands", not "is not"; and an HQ may have no STOP-Bang, where the Low risk rule must still work). If it needs a new rule shape, keep it to that one case, in `packages/clinical`'s `rules.ts` and Go's shape check, and show it on the Rules page and in the Preview's "what a send does".
- Only new HQs, and new versions of HQs without rules, get the defaults. Existing HQs keep their rules.
- Tests: the Preview of an HQ with STOP-Bang 8 (High) and no grade doesn't auto-triage; STOP-Bang Low still does; an HQ without STOP-Bang still auto-triages on Low risk.

### 4. STOP-Bang is Incomplete when the patient's sex is "Other" (answer 4)

- As for "Not known" (`ready.ts`; its published scoring in `ready.test.ts`). The score card's help text says so.

### 5. No "patient answered" line on a question the patient never saw (answer 5)

- Where the patient's answer is shown beside the clinician's (Validate, and the POA Summary), a question the patient's form didn't show when they sent has **no** patient-answer line (today: "Patient answered: no answer"). The clinician answers it, and S44's check already refuses Complete review until the set is validated.
- Unit or component tests for "shown at send" vs "not shown at send".

### 6. Small fixes from S44a's verifications (pending.md, "From S44a's verification 2/3")

- `TouchVersion` updates only a draft (`AND status = 'draft'`), so Publish's "last changed by" can't be overwritten by a save that waited for it.
- A clinician's save that waited for a Complete review is refused with the review-complete wording (re-read the status after no row, as `saveContent` does).
- Renaming or deleting a Question Set another author has just deleted says "This Question Set has been deleted."
- Complete review on an episode no longer waiting for review (On hold, Not ready, or auto-triaged past Ready for review) says "This episode isn't waiting for review any more."

### Docs

- pending.md: strike each item done here, pointing to this session. decisions.md: the builder's own choices.

## Not in this session

- Softer default wording for the auto-triage status message (pending.md, S42's build): before go-live, with the trust.
- The Lambda (S45).

## Manual test focus

1. The chain at phone size: B Yes → A Yes → C answered → B No → the answers go; reload; B Yes shows A and C empty; send.
2. Held message: auto-triaged with "wait for Checked" → On hold → Undo → Checked: the patient's message is captured once. The same with archive → un-archive.
3. A new HQ with STOP-Bang: in the Preview, High → no auto-triage; Low → auto-triage. A real send with High stays at Ready for review.
4. STOP-Bang with sex "Other" → Incomplete.
5. A clinician's correction showing a set the patient never saw: no "Patient answered" line on Validate or the POA Summary; Complete review is refused until it's validated.
