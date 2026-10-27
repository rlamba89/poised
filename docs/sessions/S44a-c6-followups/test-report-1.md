# S44a test report 1

**26 passed, 1 failed, 0 blocked** (27 cases: T-01–T-21, R-01–R-06).

- Commit `97e62e2`, working tree **dirty**: S44a's changes are uncommitted (64 files).
- 27 Oct 2026, 20:26–21:05 GMT. Local: web http://localhost:3000, API :8080, evaluate :8090 (`clinical@97e62e2e699f-dirty`).
- Tester: automatic run (Claude in Chrome). Rahul wasn't watching.

**How the test data was made.**
- **T-01–T-03's HQ ("MT S44a scores")** was built entirely in the editor.
- **The other HQs** were made with the editor's own API calls from the signed-in browser, to save time: their sets, questions and default rules. The keys were given an `x44` suffix so the key lock didn't clash with e2e HQs.
- **The feature each case tests was done in the UI:**
  - T-08's band condition and the Clinician-only/audience switches;
  - T-11's disclosure and ASA guide choice;
  - T-17's "Also High when…" lines;
  - every Rules-page check and every Publish.
- **T-11 step 4 and T-17 steps 4 and 6:** the "older disclosure with no grade", the female copy of the test case and the line testing a question were written into the chapter JSON through the API.
- **Episodes:** T-02's was made through **New episode**; the rest through the same API.
- **Patient sends:**
  - **In the patient UI:** T-02, T-04/T-05, T-07, the Demo pre-op HQ (T-12/R-03).
  - **Through the patient API:** T-13, T-14, T-17, T-18.
  - **Fill in for the patient:** T-06.
- **Hospital A1's "Text the patient when Ready for admission"** was switched on as Ada for T-14, then put back to the trust's setting. `hospitals.settings` is `{}` again.

**Phone size.** The Chrome window couldn't be resized (it stays 1512 px wide), so the phone-size checks used Playwright (Chromium, 390×844, `e2e/.venv`):
- T-12's thank-you page after a normal send and after an auto-triaged send;
- reloading it;
- no sideways scroll (scrollWidth 390).

The patient sends themselves were done at desktop width.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 | Pass | Preview, Clinician: Snore Yes → Incomplete; Loud Yes → 3 (High); Snore No → 0 (Low). Help text reads "A question a condition hides (or its whole Question Set) adds 0, and a line testing it adds nothing." |
| T-02 | Pass | Nights hidden (age 46). Validate Sleep shows **1 (Low)**, not Incomplete. The POA Summary shows no score values at all (no band has a disclosure). The evaluate service's stored result (`form_outputs`, read in the DB) holds Sleep risk 1, band Low. |
| T-03 | Pass | "Add 5 points when Is it loud? is Yes": Snore Yes + Loud Yes → 8 (High); Snore No → 0 (Low). |
| T-04 | Pass | Only About you and Smoking listed. After correcting Snore to Yes, Sleep follow-up appears **Not started**; its Validate page is empty, with no "Patient answered". POA Summary Patient answers has nothing for it. API check: the frozen row for that set is `{}`. |
| T-05 | Pass | Smoke Yes → 10 → No, then sent. After correcting to Yes, How many a day? is empty, with no "Patient answered: 10" (only "Patient answered: No" under Smoke). |
| T-06 | **Fail** | The answers are dropped correctly (`{}`), but the hidden set's row and its Validate page say the patient's answers were entered by Cara. See F1. |
| T-07 | Pass | Everything shown: the follow-up keeps Yes and "very loud". |
| T-08 | Pass | Publish blocked with the exact message. Unticking Clinician only removes it; putting it back brings it back; making the follow-up a Clinician set removes it. |
| T-09 | Pass | Both messages are right ("Make the question visible…" and "Move the condition to a clinician Question Set…"). The server's publish also returns 422 with both problems. |
| T-10 | Pass | **Rules page:** yellow box with the exact line; still there after a reload. **Publish window:** the line under "Before you publish (these don't stop it)", Publish enabled, and it published. **Low risk deleted** (on v2, since v1 is read-only once published): the warning goes. |
| T-11 | Pass | "No ASA grade" is still selected on reopening, and Publish doesn't warn. A disclosure with E11.9 and no grade gets the yellow warning naming the set, question and answer. The code's own display is used ("Type 2 diabetes mellitus without complications (E11.9)"). Publish stays enabled. |
| T-12 | Pass | "Thank you, Tom/Ann/Ivy" and "Your answers have been sent to the hospital. We'll contact you about what happens next." Same after a reload, and for the auto-triaged Demo pre-op HQ send. Checked at 390×844 (Playwright). |
| T-13 | Pass | "The HQ's rules suggest: Ready for POA (the rule "Fit for POA")" shows after the send. It goes after Complete HQ review, stays gone after On hold, and doesn't show on the cancelled episode. |
| T-14 | Pass | **On send:** auto-triaged with "The patient's message goes when you click Checked", and only the invite text in /qa/messages. **On hold → Ready for admission:** one Ready for admission text (id 135). **Checked:** no second text. |
| T-15 | Pass | The stale tab shows "Someone checked this while you had it open." and "Checked by Cara Clinician on 27/10/2026, 20:58." |
| T-16 | Pass | The second Complete HQ review shows "The HQ review is already complete." (see E4 about the stale page). |
| T-17 | Pass | **Editor:** "Also High when all of these hold:" with both lines, saved and still there after a reload. **Preview:** Male → 2 (High) with the High note; Female → 2 (Low); sex cleared → Incomplete. **Test cases:** male passes, the female copy fails. **Male patient's send:** Validate shows 2 (High), and the stored result has band High. **Deleting the question a line tests:** Publish names "Sleep score" band "High": it tests something that no longer exists (q_aw1x44). |
| T-18 | Pass | No "Suggested ASA" line before the send. After the send: "Suggested ASA III: Heart failure, unspecified (I50.9)". On Validate it switches to "ASA II: Smoker (77176002)" when Heart failure is set to No, and back to III, with no reload. |
| T-19 | Pass | Nights stays hidden for the 46-year-old on the episode page and Validate. |
| T-20 | Pass | Val (A2) and Bea (Trust B) see "You don't have access to this hospital" for the episode, Validate and Rules URLs. The API gives 404 Episode/Questionnaire/Chapter not found. |
| T-21 | Pass | **Validate:** a reload after the correction matches the saved answers (ASA II). **Rules:** the warning is still there after a reload. **Back** from Validate returns to the episode page. |
| R-01 | Pass | STOP-Bang ready-made, male, 54, BMI 38.1, all Yes → 8 (High risk). |
| R-02 | Pass | The UI's Send stays disabled with a set incomplete. The patient API's submit returns 409 "Please finish "Sleep follow-up" before sending your answers."; once it's complete, the send works (204). |
| R-03 | Pass | The Demo pre-op HQ's low-risk send is auto-triaged to Ready for admission and is on the Auto-triaged worklist; Checked takes it off. |
| R-04 | Pass | The ASA III send adds the "Book anaesthetic review" task, the "Anaesthetic review" flag and the anaesthetist-ASA requirement, each with a General notes line. |
| R-05 | Pass | Complete review with sets not validated → 409 "Validate every Question Set first: "Smoking" and "Heart" aren't validated." (sent by API; the button is disabled in the UI). |
| R-06 | Pass | Ada's Audit page lists rows. Export CSV downloaded `~/Downloads/audit-2026-10-27-205657.csv` (1,985 lines, 240 mention "MT S44a"). |

## Failures

### F1 (Minor). T-06: a hidden set's dropped answers still say "Patient's answers entered by Cara Clinician"

**Signed in as:** Cara Clinician, Hospital A1.
**Episode:** "Bob MT S44a T06 8244" (`72583bd5-20dd-4e71-acc7-4c335983a935`, HQ "MT S44a snore" v1).

**Steps:**
1. On a new episode of the snore HQ, click **Fill in for the patient**.
2. About you: Snore **Yes**. Sleep follow-up: **Yes**, and "loud".
3. Back to About you: Snore **No**. Smoking: **No**. Send.
4. Validate About you, correct Snore to **Yes**, Validate, and go back to the episode.

**Expected** (as T-04 steps 3–5): Sleep follow-up is listed **Not started**, and nothing says the patient answered it.

**Actual:**
- The answers *are* dropped: the frozen patient row for Sleep follow-up is `{}`.
- But the episode page's row for **Sleep follow-up** says "Not started" plus, in orange, **"Patient's answers entered by Cara Clinician"**.
- Its Validate page (`…/episodes/72583bd5-…/sets/428a9d99-0da3-4110-abcf-cc78e8051b38`) opens with the banner **"The patient's answers here were entered by Cara Clinician on the patient's behalf."** above two empty questions.

So a clinician is told the patient gave answers in a set that holds none (they were dropped on send). In the patient's own version (T-04), no such marker shows. It looks like the marker comes from the row's `updated_by` (Cara saved into that set before it was hidden), not from whether any answers are left.

**Evidence:**
- No console errors, and no failed requests.
- GIF: `~/Downloads/S44a-T06.gif`.
- Screenshot: `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-FoNO8i/screenshot-1793134402890-0.png`.

## Found while exploring

- **E1 (Cosmetic):** the score card in the editor says "Total of the scores of 2 questions, plus **1 points line**". It should be singular, e.g. "plus 1 points line" → "plus 1 line".
- **E2 (Minor, misleading):** after deleting a question that a score adds up (T-17 step 6, v2 draft), the score card reads "Total of the scores of 2 questions **(1 in earlier Question Sets)**". The deleted question is counted as being in an earlier set; there are no earlier sets. Publish does name the problem correctly, as a logic problem for the score and for the band.
- **E3 (Cosmetic):** the new publish problems end a question with double punctuation, e.g. "but patients can't see Nurse: concerns?." and "…needs review?.".
- **E4 (Minor):** a refused second Complete HQ review (T-16) shows the right message, but the stale tab isn't refreshed. It still shows Ready for review, an enabled Complete HQ review button, and "The HQ's rules suggest: Ready for POA". A refused Checked (T-15) reloads the card.
- **E5 (to confirm, possibly from S43):** a rule's suggested status (T-13) writes no General notes line. The task, flag and anaesthetist-ASA actions each write one (R-04). CLAUDE.md says each S43 action has an automatic General notes line.
- **E6 (Note):** the POA Summary never shows a score's value or band unless the band has a disclosure. So T-02 step 5 ("the same score, 1 (Low)") can't be seen there. It was checked in the stored evaluate result instead.
- **E7 (Note, not reproduced):** in the Preview, after switching Sample patient sex Male → Female once, the score card inside the form briefly still read "2 (High)" while the Clinical outputs panel said "2 (Low)". Repeating the switch gave "2 (Low)" in both at once.
- **E8 (Note):** a hidden set's Validate page can still be opened by URL, and shows empty questions. This is probably fine, since a clinician's correction may bring the set back.
- **Design, not a bug:** a page holding the BMI Calculator disables every other Add content button, so STOP-Bang needs its own page.

## Console and network errors

- No console errors or warnings on the editor, episode or Validate pages checked.
- No unexpected 4xx/5xx. The only 4xx were the intended ones: T-09 publish 422, R-02 409, R-05 409, and T-20's 404s.

## GIFs, screenshots and downloads

All GIFs and downloads are in `~/Downloads`:

- **GIFs:**
  - `S44a-T01.gif` (main happy path: score in the Preview);
  - `S44a-T04.gif` (the snoring example, patient → clinician);
  - `S44a-T18.gif` (suggested ASA on Validate);
  - `S44a-T06.gif` (failure F1).
- **Phone screenshots:**
  - `/private/tmp/claude-501/-Users-rahul-poised/c92da616-784f-4500-966e-bd8e6e97afd6/scratchpad/S44a-T12-t04-phone.png`
  - `/private/tmp/claude-501/-Users-rahul-poised/c92da616-784f-4500-966e-bd8e6e97afd6/scratchpad/S44a-T12-r03-phone.png`
- **Audit CSV:** `~/Downloads/audit-2026-10-27-205657.csv`.

## Hand checks for Rahul

None. Every case ran. Phone size was checked with Playwright, because the Chrome window wouldn't resize.
