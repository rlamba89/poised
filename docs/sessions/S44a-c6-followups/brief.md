# S44a: C6 follow-ups (Rahul's answers of 27 Oct, and the defects found in S38–S44)

**Plan:** C6 ([c6-scores-asa-triage.md](../../plans/c6-scores-asa-triage.md)), a follow-up session added on 27 Oct, before S45.
**Requirements:** the open items S38–S44 left for Rahul in [pending.md](../../pending.md), answered in chat on 27 Oct ([answers.md](answers.md); decisions.md, 27 Oct, "C6's open items answered").
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S44 verified.

**Ask Rahul first:** nothing. Every item below was answered on 27 Oct. If an item turns out bigger or different from what's written here, ask (questions.md); don't widen it.

## Build (each part test-first: the failing test, then the code)

### 0. The local database (do this first)

- Reset the local `sj` database to migrated + seeded (Rahul, 27 Oct), keeping **HJE Full HQ** as decisions.md §4 says (dump it, rebuild, load it back). It clears the 435 MB audit log that makes `test_audit` fail locally.
- **Then, if any draft's saved test case still fails only because of S39's "E" rule** (expects "ASA IV", now gets "ASA IVE"), update its expected line with a one-off local script (Rahul: "update drafts automatically"). After the reset there may be none; say so in the build notes.
- **`test_audit.py` reads the audit CSV as a stream** (so a big log can't break it again). **Don't cap what a "Listed episodes" row records:** the audit log is the record of who saw which patients, so it stays complete.

### 1. Scores: a question hidden on purpose adds 0 (S38)

- A score input that a condition hides (the question, its page or its whole Question Set) adds **0 points**. A shown question left unanswered still makes the score **Incomplete**. "Add N points when…" lines that test a hidden answer don't fire.
- One place: `packages/clinical`'s score code, so the Preview, test cases, the clinician's pages, the POA Summary and the evaluate service all agree. Update the score card's help text, and pending.md.
- Unit tests: hidden question → 0 and a band; shown and unanswered → Incomplete; a hidden set; STOP-Bang unchanged (`ready.test.ts`).

### 2. On send, keep only what the patient's form showed (S38)

- When the patient sends (both routes: the patient's and staff's on-behalf), the frozen patient answers are **only those the patient's form showed at that moment**. Answers to a Question Set that its condition hid, and to questions hidden inside a shown set, are dropped. So a set a clinician's correction brings back later starts **empty** for the clinician (Rahul: "Start empty").
- Go never evaluates: the evaluate service works it out (e.g. `/evaluate` also returns the answers as sent, with hidden values cleared, using `packages/clinical`'s shown-sets and SurveyJS's own visibility), and Go stores that. A pre-filled answer in a hidden set is dropped too.
- If the service can't be reached, keep every answer, as today (S41's fail-safe: Ready for review and the note).
- Integration tests (fake evaluator): a hidden set's answers aren't in the frozen row; a shown set's are; evaluator failure keeps all. Extend the shared fixture so both sides agree.

### 3. Publish refuses a patient condition on clinician-only content (S38)

- A condition the patient's form must work out (on a patient Question Set, page or question) that tests a **clinician-only** element (e.g. a clinician-only score's band) is a Publish problem: "“Sleep follow-up” shows only when Sleep risk is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set."
- In `publish.ts` (so the browser's Publish window and the server's check through the evaluate service both refuse it). Unit tests, and one case in the publish fixture.

### 4. The thank-you page's wording (S42)

- One neutral wording for every send, promising no review: **"Thank you, {first name}. Your answers have been sent to the hospital. We'll contact you about what happens next."** (`PatientHQ.tsx`, and its translations if the page has them).

### 5. Authors are told when auto-triage can't work (S42)

- Auto-triage stays blocked when a score is Incomplete, including a clinician Question Set's score that needs clinician-entered inputs (Rahul: rightly stopped; the clinician enters them and carries on).
- **Tell the author** where auto-triage is set up (the Rules page) and in the Publish window, as a **warning that doesn't block**: "Auto-triage won't move patients on for this HQ: the score “Neck size risk” needs answers a clinician enters (“Neck circumference”), so it's always incomplete when the patient sends. These patients go to Ready for review." Name each such score and its inputs.
- In `packages/clinical` (unit-tested), shown by the web app.

### 6. The rules' suggested status hides once reached (S43)

- "The HQ's rules suggest: Ready for POA" shows **until the episode reaches or passes that status**, and not on a cancelled, archived or admitted episode. "Passed" follows the main path (HQ not complete → Ready for review → Ready for POA → POA complete → Ready for admission → Admitted); for On hold and Not ready, use the episode's history (has it ever reached the suggested status?). The suggestion stays in the history. `StatusControl.tsx`, unit-tested in `status.test.ts`.

### 7. Defects found by the verifiers (pending.md)

- **S42: a held Ready for admission message can go twice.** Clear `triage_message_pending` on every status change made by hand (`SetEpisodeStatus`), as the S42 builder's decision 6 intended. Integration test: auto-triaged with the message held → On hold → Ready for admission (sends it) → Checked sends nothing.
- **S44: a save that clears a validation can slip past Complete review.** `SaveClinicianAnswers` takes the episode row `FOR SHARE` (Complete review holds it `FOR UPDATE`). Test the order, as S44's tests do.
- **S41: a rules save racing Publish.** Re-read the version row `FOR UPDATE` (and its chapters and rules) inside Publish's transaction, so what's checked is what's published.
- **S38: the score picker offers questions inside a repeating group** (`scoredQuestions`), which `score()` can't read. Leave them out, as `pointsSubjects` does.

### 8. Small improvements Rahul chose

- **Two clearer messages:** a second Complete review → "The HQ review is already complete." (not `notInReview`'s wording about answers); a refused Checked → "Someone checked this while you had it open."
- **ASA guide nudges for authors (S39):** Publish **warns** (doesn't block) when a disclosure's code offers ASA examples but no grade was chosen; the editor **remembers a deliberate "No ASA grade"** (store it explicitly, e.g. `asaNone: true`, so an older disclosure without a grade still reads as "not chosen").
- **The suggested ASA in the right places (S39):** the episode page shows it only once the patient has sent (`patientSubmittedAt`); the **Validate page shows it too**, updating as the clinician corrects answers.
- **The age in the hospital's time zone (S38, S44):** the browser works out "the age on the day of sending" in the hospital's time zone, as Go does (the time zone comes from the API, never a literal: A-20), so an age-conditioned set is shown the same way in both.

### 9. Authors' own bands get "also when" (S38)

- Any band of an author's own score can have an **"also when"** line, as STOP-Bang's has (e.g. "High also when the score is 2 or more and sex is male"), built from the same menus as "Add N points when…", never typed. `packages/clinical` (unit-tested: the band, Incomplete inputs, the Preview and test cases), the score card in the editor, and the publish checks (a band's line testing something deleted is named).

### Docs

- pending.md: strike each item done here, pointing to this session; decisions.md: the builder's own choices.

## Not in this session

- The ASA grade history (S44b).
- The production policy for changes to clinical logic (the regression check and pinning: C10, decisions.md 27 Oct).
- The Lambda (S45).

## Manual test focus

1. A score with a conditional question: hidden → a band, with 0 for it; shown and skipped → Incomplete; the same in the Preview and the clinician's view.
2. The snoring example: Yes → answer the follow-up → No → send. The follow-up's answers aren't stored; a clinician's correction to Yes shows the follow-up empty.
3. Publish refuses a patient set conditioned on a clinician-only score, with the message; warns about auto-triage and an ungraded code, and lets the author go on.
4. The thank-you page's new words, at phone size.
5. The suggested-status line disappears once that status is reached.
6. The held message goes once only (On hold → Ready for admission → Checked).
7. An "also when" band on an author's score, in the editor, the Preview and a patient's send.
8. The suggested ASA on Validate changes as an answer is corrected.
