# S44c test report 2 (retest)

**Summary: 16 passed, 0 failed, 0 blocked.** Every case listed at the top of the test plan was run: F1 (T-11 in full), the cut-off score name (T-10 step 1) and the 15 cases blocked in round 1. One part of one case (T-14 steps 2–3) can't be done in the UI as the plan words it. The server's message was confirmed through the API instead; details under T-14.

- **Environment:** `f143b17` with the S44c changes uncommitted (working tree dirty, 51 changed or new paths). 28 Oct 2026, 14:07 to about 14:35 GMT. http://localhost:3000 (`make dev`: API :8080, web :3000, evaluate :8090 reporting `clinical@f143b177d60e-dirty`).
- **Browser:** Claude in Chrome, Browser 1 ("rahul chrome", bce1f965…).
  - The extension dropped once around 14:21. After about 15 s it reconnected to Browser 1 on its own, and nothing was lost.
  - The window was 1512 px wide, and later 1549 px. No phone-size cases were in scope this round.
- **Test data:** made in the UI as Alex unless noted.
  - **MT S44c stopbang 1400** v1 was published (it was the round 1 draft). v2 is a draft, used for T-10 step 4.
  - **MT S44c nostop 1415** is new: one patient set with one Yes/No question and the default rules. v1 is published; v2 is a draft whose Low risk was edited (T-10 step 4).
  - **MT S44c sets 1430** is new. It started with three sets, then two after T-13, and v1 is published.
  - The round 1 HQs **MT S44c chain 1335** and **MT S44c held 1355** were reused, with the round 1 chain episodes T12, T14, T15 and R01.
  - **Episodes:** "Sam MT S44c T04" was made on the New episode page as Cara. T05, T06, R03, T09M, T09F, T11 and R04, and the R-05 episode for the existing T09M patient, were made with the same `POST …/episodes` call the page makes, run from the browser as Cara or Alex, to save time.
  - **Mobiles:** +44 7700 900841 to 900844. These had no earlier messages.
- **API, database and terminal use (said where used):**
  - Invite links were read from `sent_messages`, the table behind /qa/messages.
  - Text counts per mobile came from `sent_messages`.
  - R03's status and the R-05 pre-fill row were checked in the database.
  - One `POST …/complete-review` call was made for T-14.
  - The patient-path check in "Found while exploring" used the patient's own session.

## Results

| Case | Result | Note |
|---|---|---|
| **F1** / T-11 step 1 | **Pass** | Sex in Preview and Test the whole HQ: Female, Male, Other, Not known. Other → Incomplete; Female → 5 (High risk); Not known → Incomplete; Other on Test the whole HQ → "Low risk: no auto-triage: a score is Incomplete". |
| T-04 On hold → Undo → Checked | **Pass** | Details below the table. |
| T-05 Archive → Un-archive → Checked | **Pass** | Archived, the card dropped the "waits for Checked" line. Un-archived (with a reason), the line came back. No text until Cara clicked Checked, then exactly 1. |
| T-06 On hold → RFA by hand → Checked | **Pass** | The text arrived on the manual move to Ready for admission. The card then read "Not yet checked by a clinician." with no "message goes" line. Checked sent no second text (still 1). |
| T-08 Preview and Test the whole HQ: High and Low | **Pass** | Preview, Clinician view: male, 60, 170 cm, 116 kg, all Yes → "STOP-Bang score: 8 (High risk)", rules fired: None. Female, 30, 170 cm, 64 kg, all No → "0 (Low risk)", "Low risk: auto-triage to Ready for admission". Test the whole HQ rules matched in round 1 and again here. |
| T-09 Real send, High stays at Ready for review | **Pass** | T09M (male, BMI 40.1, all Yes): Ready for review, no auto-triage. T09F (female, BMI 22.1, all No): auto-triaged to Ready for admission by "Low risk". |
| T-10 HQ without STOP-Bang | **Pass** | Details below the table. |
| T-11 step 2 (help text) | Pass (round 1) | Not repeated. |
| T-11 step 3 (real episode, sex Other) | **Pass** | Ready for review. General notes: "Not auto-triaged by the rule “Low risk”: a score is Incomplete. Review this HQ by hand." Validate: "STOP-Bang score: Incomplete". The POA Summary doesn't list STOP-Bang at all, for any patient (see Exploring). |
| T-12 No "Patient answered" on unseen questions | **Pass** | Details below the table. |
| T-13 Rename or delete a set another author deleted | **Pass** | Details below the table. |
| T-14 Complete review on an episode no longer waiting | **Pass**, with a plan note | Step 4 was done in the UI and passed. Steps 2–3 can't be done in the UI as written; the message was confirmed through the API. Details below the table. |
| T-15 Clinician save after review completed elsewhere | **Pass** | Tab 1 on Validate Breathing; tab 2 completed the review. In tab 1, changing B gave "Not saved / The HQ review is already complete, so the answers can't be changed." with Reload. |
| R-01 A send keeps only what was shown | **Pass** | The patient answered B Yes, A Yes, C "Coughing", then B No, finished and sent. Validate Breathing shows only B (No). |
| R-02 Grade history | **Pass** | Details below the table. |
| R-03 Low-risk text at once | **Pass** | On MT S44c nostop 1415 (default rules, no wait, Hospital A1's text on), the Ready for admission text went at 14:16:07 on send. Checked was never clicked. |
| R-04 ASA III: task and flag | **Pass** | Details below the table. |
| R-05 Pre-fill from an earlier HQ | **Pass** | Details below the table. |

**T-04 in detail:**
- After the send: Ready for admission, auto-triaged, and the card said "Not yet checked by a clinician. The patient's message goes when you click Checked." /qa/messages had only the invite.
- **On hold:** the card dropped the "message goes" sentence.
- **Undo** (with a reason): back at Ready for admission, with no new text, and the sentence came back.
- **Checked:** exactly 1 Ready for admission text. General notes: "Ready for admission message sent by text message."

**T-10 in detail:**
- **Step 1:** Low risk reads "…and STOP-Bang score isn't High risk (not in this HQ): auto-triage to Ready for admission". In the edit form, the score box now shows "STOP-Bang score (not in this HQ)" in full, so the cosmetic fix is confirmed.
- **Step 2:** publish went through with no problem. The line reads "This version auto-triages: ASA I–II, no red flags → Ready for admission…".
- **Step 3:** R03 was auto-triaged to Ready for admission by "Low risk" (page and database).
- **Step 4, unticking:** on the v2 draft, unticking "and a score isn't in a band" and saving worked. The rule now reads "When the suggested ASA is I or II and there are no red flags: auto-triage to Ready for admission".
- **Step 4, re-ticking:** on this HQ the checkbox can't be ticked again, because it greys out with "This HQ has no score with bands." So that half of step 4 was run on the MT S44c stopbang 1400 v2 draft. Untick, tick again, score "STOP-Bang score" with no band, then Save rule gave "Rule “Low risk”: choose the score and at least one of its bands." in the form.

**T-12 in detail:**
- **Validate Breathing:**
  - B → Yes shows "Patient answered: No".
  - A appears empty; after A Yes there's no line under A.
  - C appears; after typing "Coughing" there's no line under C.
- **After Validate:** Pillows is listed as Not started, and Complete HQ review is disabled.
- **Validate Pillows:** D Yes, with no "Patient answered" line (the banner reads "The patient didn't answer this Question Set."). Complete HQ review is then enabled.
- **Review Breathing again:** exactly one "Patient answered" line, under B.
- **POA Summary:**
  - Breathing has one change line: "Do you get short of breath? Patient answered “No”, now “Yes”". The box heading reads "Changed by the clinician on Page 1", the page's name, under the Breathing heading.
  - There's no box for Pillows.
- Step 7 was skipped (optional; no optional question in the HQ).
- No console errors.

**T-13 in detail:**
- The editor was open in two tabs. Tab 1 deleted Question Set 3.
- **Rename in tab 2** (stale) → "This Question Set has been deleted." Tab 2 then reloaded its list.
- **Delete in tab 2:** a new set 3 was added and shown in tab 2, then deleted in tab 1. Deleting it from the stale tab 2 gave the same message.
- **After publishing in tab 1:** renaming Question Set 1 in tab 2 gave "Only draft versions can be changed."

**T-14 in detail:**
- **Steps 2–3 can't be done in the UI as written.**
  - At Ready for review, the status select is disabled ("The status can be changed by hand once the HQ review is complete."). The only moves offered are Cancel and Archive, and those give their own messages.
  - So a second tab can't put an unreviewed episode On hold.
  - The new message was confirmed through the API: `POST …/complete-review` on T09F (auto-triaged, never reviewed) → 409 "This episode isn't waiting for review any more."
- **Step 4 passed in the UI:**
  - On T14, Breathing was validated. Tab 2 completed the HQ review, then set On hold.
  - The stale tab 1's Complete HQ review gave "The HQ review is already complete."

**R-02 in detail:**
- On the T09M POA Summary, Add grade ASA III, then ASA IV with the reason "MT S44c new echo result".
- The history lists ASA IV (**COUNTS**, with the reason) above ASA III, each with Entered in error.
- The header shows Anaesthetist ASA grade ASA IV.

**R-04 in detail:**
- This used the published E2E J11 send HQ: default rules, with heart failure giving ASA III.
- After the send: Ready for review and "Suggested ASA III".
- The rule added the flag "Anaesthetic review", the task "Book anaesthetic review" (anaesthetist), and "Ready for admission needs the anaesthetist's ASA grade first".
- General notes have all three automatic lines.

**R-05 in detail:**
- This was a second episode on MT S44c stopbang 1400 for the T09M patient.
- The patient's list shows Health tagged "Filled in earlier" with **Check**. The card reads "We've filled in 5 answers from your questionnaire on 28 Oct 2026. Please check them." with "Check each answer" and "Everything is still correct".
- **Check each answer:** the five STOP-Bang answers came back Yes, each marked "From your answers on 28 Oct 2026".
- The BMI block wasn't copied. That's by design (`internal/prefill`: the BMI block doesn't take part).

## Failures

None.

## Found while exploring

- **Plan note (T-14, not an app fault):** steps 2–3 ask a second tab to set an unreviewed Ready for review episode On hold. The UI has never allowed that, because the status is locked until the review is complete. In the UI, "This episode isn't waiting for review any more." can only be reached through a race. The integration test covers it, and the API check above confirms the wording. The plan's step should be reworded, or marked "integration test only".
- **Cosmetic (T-13):** in the stale editor tab, the red "This Question Set has been deleted." banner stays at the top after you choose another set and work on. It's replaced only by the next error, so it looks stale.
- **Not S44c's, worth knowing:**
  - The POA Summary shows no STOP-Bang answers or score, neither the Validated summary nor Patient answers, for any of T09M, T09F or T11.
  - The score's value (including Incomplete) is visible only on Validate.
- **Not S44c's (S44b wording):** General notes say "Anaesthetist ASA grade set to 3" and "changed from 3 to 4: …". The rest of the page says "ASA III" and "ASA IV".
- **Not S44c's, test data:**
  - Round 1's chain episodes have General notes dated in the future: "Day 2 reminder… on 30/10/2026", "Day 5 reminder… on 02/11/2026" and "Not responding… on 04/11/2026".
  - This is presumably because the e2e run between rounds moved the QA jobs clock forward and ran the jobs over every open episode.
  - It's harmless, but it makes manual-test data look odd after an e2e run.
- **Patient path with someone else's ID:** signed in as patient T09M, I opened `/home/episodes/<T11's episode>`. The page showed "Episode not found." Good.
- **Keyboard:** on Validate, Tab from the last text answer (C) moves focus to the header's hospital select, not to the Validate button. It's minor, and it's an existing layout matter.
- No console errors on the staff or patient tabs during the cases. The only 4xx responses were the 409s the cases expected.

## Console and network errors not tied to a case

None.

## Hand checks for Rahul

None.

## Clean-up

As Ada: Settings → Hospital A1 → "Text the patient when the episode is Ready for admission" is back to "Use the trust's setting", and saved.

## GIFs

- `~/Downloads/S44c-T11-retest.gif`: F1. Sex Other, Female and Not known in the Preview's Clinician view; T-08's High (8) and Low (0); Test the whole HQ with Other.
- `~/Downloads/S44c-T04.gif`: the held message through On hold, Undo and Checked.
- `~/Downloads/S44c-T12.gif`: Validate with only one "Patient answered" line; Pillows appears; the POA Summary change list.
- (Round 1's `~/Downloads/S44c-T01.gif` is still there.)
