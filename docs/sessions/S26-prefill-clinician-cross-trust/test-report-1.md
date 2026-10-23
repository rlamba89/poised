# S26 test report 1

**20 passed, 0 failed, 0 blocked** (15 test cases and 5 regression cases).

**Environment**
- Commit `08c7645`. The working tree isn't clean: it has the uncommitted S26 build (45 changed or new files).
- 23 Oct 2026, 13:46–14:15 BST. Local: http://localhost:3000 (API on :8080), migrated to 00026 and seeded by the conductor.
- Browser: Claude in Chrome, Rahul's "rahul chrome" profile. Patient screens ran in a same-origin 390 × 844 iframe (scaled up on screen only, so the layout stayed 390 wide). Staff screens ran at desktop size.
- **Settings:** `crossTrustPrefill` was off in both trusts at the start (no override saved). Both are back to that state at the end: the override was cleared, `trust: {}`, effective `false`. I checked this with the API.

**What I did through the API or console rather than the UI** (setup only, to save time; each flow was also tested through the UI at least once):
- **Settings:** Ada switched Trust A on in the **Settings UI** once (T-05/T-15). Later switches for both trusts used `PATCH /api/o/{oid}/settings` with Ada's or Max's dev-login cookie.
- **Trust B histories** for T-05, T-08, T-09 and the extra T-11 person: Bea created the episode through the API, and the patient's link, date of birth, answers and Send also went through the API (`/api/p/...`). The same applies to:
  - T-08's Trust A pre-op HQ;
  - T-14's E1.

  The T-01 person's Trust A history went entirely through the UI.
- **Some Trust A episodes** after T-04 were created through the API as Cara. These were T-05's second episode onwards, T-09, T-10, T-11, T-14 and R-05. T-01 to T-05's first episodes were created through the UI.
- **Quick finishes:** in T-08, the rest of the follow-up HQ (pregnancy, weight, stairs) was filled and sent with `fetch` in the patient's own browser session. The consent question and the card were done in the UI. R-05's answers were saved the same way through the staff on-behalf API, then sent with the UI's **Send the answers** button.
- **T-13:** the "new incognito window" step used `curl` with no cookies, because the tool can't open an incognito window. Staff cookies were also tried.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Validate markers | Pass | **Your health:** the line reads "6 answers … the patient changed 1 … “Everything is still correct” on 23 Oct 2026 …". Diabetes "(matched by same wording)". Snoring is Yes "(matched by link)", which is Cara's correction. Sickness shows the orange "Changed by patient: pre-filled “Yes”" and the answer No. No marker on pregnancy. Problem, What happened and Your allergies (page 2) have markers. **About you:** "Check each answer". Smoking, alcohol and height have markers; weight and stairs don't. |
| T-02 POA Summary | Pass | Both tabs: the button lines, "Pre-filled on Page 1" (5 items), the **CHANGED BY PATIENT** badge with "pre-filled “Yes”, patient answered “No”", and "Pre-filled on Page 2" (Your allergies). Nothing about pregnancy or weight. |
| T-03 Only shown questions marked | Pass | Problem → No: "What happened" and its marker disappear. Back to Yes: both return, with the text kept. The line still says 6. The sickness answer was already No (pre-filled from the 2nd HQ), so there was no "changed" here. |
| T-04 Staff pressed the button | Pass | "6 answers here were pre-filled … Cara Clinician chose “Check each answer” with the patient on 23 Oct 2026." The orange "entered by Cara Clinician on the patient's behalf" note is there. No "Changed by patient". |
| T-05 Only when both are on | Pass | A on and B off: no question, both sections show **Start**. B on and A off, then a reload: the same. |
| T-06 The question (phone) | Pass | "Use your earlier answers?" / "Use the answers you gave Hospital B1 in October 2026?", the explanation, and the two buttons. No section tiles. 390 wide: `scrollWidth` = `clientWidth` = 388, and both buttons are 314 px (full width). "Your questionnaires" works, and coming back shows the question again. |
| T-07 Yes fills from Trust B | Pass | **Patient:** after Yes, both sections show Please check. A reload doesn't ask again. The card says "We've filled in 5 answers from your questionnaire on 23 Oct 2026". Diabetes No and snoring Yes, each with "From your answers on 23 Oct 2026". **Validate:** "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust), 23 Oct 2026 (matched by link)", and diabetes "(matched by same wording)". |
| T-08 Trust A first | Pass | Snoring is No (Trust A's), not Trust B's Yes. After sending, every marker on Validate says "Pre-filled from Demo pre-op HQ, 23 Oct 2026 (matched by …)" with no "another trust". Trust A's pre-op HQ was sent but not reviewed, so these are the patient's answers. |
| T-09 No, thanks remembered | Pass | Both sections show **Start**. No question after a reload, and none on a second Trust A episode. I double-clicked No, thanks: one 204, and no error on screen. |
| T-10 Setting off after a Yes | Pass | B off, then a new episode: no question, and every source is Trust A's own earlier follow-up HQ (checked in the GET's `prefill` map). T-07's episode still shows its "Hospital B1 (another trust)" markers. B was switched back on. |
| T-11 On behalf: never Trust B | Pass | **T-05's person:** no question; everything comes from Trust A's own follow-up HQ, none marked as another trust's. **Extra check:** a person with **only** a Trust B history who said Yes (on another, unsent Trust A HQ). Cara's **Fill in for the patient** on a new episode shows both sections as **Start** and fills nothing. |
| T-12 Audit in both trusts | Pass | **Trust A** (UI as Ada, filtered to S26 T09, and the API): rows "Patient answered whether to use another trust's answers" for T-05, T-08, T-09 and T-11, at Hospital A1. Of 297 Trust A rows naming S26 patients, none names a Trust B patient or episode id. **Trust B** (UI as Max at Hospital B1): "Patient's answers used to pre-fill their HQ at another trust (they agreed)" by "Patient (their link)", at Hospital B1, for T-05, T-08 and T-11. None for T-09. |
| T-13 Refused requests | Pass | All bodies are plain-English `{"error": …}`. The results: <ul><li>already answered, an unknown uuid, and the form's own trust: 409 "This question has already been answered, or no longer applies. Please reload the page.";</li><li>`not-a-uuid`, and no `agree`: 400;</li><li>a broken body: 400 "The request could not be read.";</li><li>the zero uuid, and another person's episode: 404 "Episode not found.";</li><li>signed out, and a staff cookie: 401.</li></ul> |
| T-14 Stale tab | Pass | Tab 1 shows "Your answers weren't saved … your last change was not saved." with **Reload**. After Reload, the "Your earlier answers" card (5 answers) is back. Tab 3 showed both sections as Please check. |
| T-15 Help text | Pass | "Used only when the other trust has it on too, and the patient is always asked first. Never when staff fill in an HQ for the patient." |
| R-01 Same-trust card | Pass | "We've filled in 6 answers from your questionnaire on 23 Oct 2026. Please check them." **Everything is still correct** jumps to the pregnancy question, with its text on screen. Changing the sickness answer removed its marker. |
| R-02 Started | Pass | A pre-filled HQ that was opened but not touched shows **To do** on the home. After **Everything is still correct** it shows **In progress**. |
| R-03 Clinician corrections | Pass | Episode 1's POA Summary, Validated tab: "Changed by the clinician on Health – Do you snore loudly? Patient answered “No”, now “Yes”". |
| R-04 No history | Pass | A brand-new patient's Demo pre-op HQ: no question, card or markers; sent normally. Cara's Validate shows no pre-fill line. |
| R-05 On behalf, no history | Pass | **Fill in for the patient** shows both sections as Start. After **Send the answers** the episode is **Ready for review**. |

## Failures
None.

## Found while exploring

1. **Minor, and probably older than S26:** Validate shows a false correction marker on a list's optional column.
   - **Where:** on **Your health** page 2, under "What happens when you have it?" in **Your allergies**.
   - **What it shows:** the orange "Patient answered: no answer" marker, as if the clinician had changed it.
   - **Why it's wrong:** the patient's answer and the clinician's copy are identical: `[{"allergies_what":"Penicillin","allergies_reaction":"A rash"}]` in both rows, from `GET …/episodes/{eid}/hq`.
   - **Where else:** it also appears on T-01's first episode, which has no pre-fill (`…/episodes/9692e5d8-…/sets/fca45d0c-…`, page 3). So it isn't caused by S26.
   - **Not affected:** the POA Summary doesn't list it as a correction.
   - **To reproduce:** as the patient, add one allergy with both fields filled and send. As Cara, open Validate for Your health and go to the allergies page.
2. **Cosmetic (API only):** a bad `orgId` gets the wrong message. `POST /api/p/episodes/{eid}/prefill-consent` with `{"orgId":"not-a-uuid","agree":true}` returns 400 "Choose “Yes, use them” or “No, thanks”.", which describes a missing answer. The page never sends this.
3. **Observation: where a copied answer came from is lost.** Answers copied from Trust B into a Trust A HQ become Trust A's own once sent.
   - In T-10, Trust B had switched its setting off. The next Trust A HQ was still filled with those values, marked only "Pre-filled from Demo follow-up HQ" (Trust A).
   - Nothing tells the clinician that the value first came from Hospital B1.
   - This fits "answers already copied stay", but Rahul may want the hazard log to mention it.
4. **Observation: the audit wording can overstate a read.** For T-08's person, Trust A's own answers covered every question, so nothing from Trust B was used. Trust B's log still says "Patient's answers used to pre-fill their HQ at another trust (they agreed)". The build notes describe this choice: every read is logged, even when nothing matched. "Read to pre-fill" might be more exact.
5. **Works as designed:** the question is asked even when nothing will be used. I created a new Trust B HQ for T-05's person. They were asked "Use the answers you gave Hospital A1 in October 2026?", and after Yes everything still came from Trust B's own earlier HQ. This is the builder's documented "deciding to ask reads no answers".
6. **Checked, no problem:**
   - A patient asking for another person's episode (`GET /api/p/episodes/{other}`) gets 404.
   - Double-clicking **No, thanks** sends one request.
   - Reloading after Yes or No doesn't ask again.

## Console and network errors
- After console tracking started (from T-14 on), there were no console errors or warnings on the staff Validate, POA Summary and Audit pages, or on the patient pages.
- I saw no unexpected 4xx or 5xx. The only refusals were T-13's deliberate ones, and T-14's expected 409 on the stale save.

## Hand checks for Rahul
None. T-13's incognito step was done with `curl` and no cookies (401), which tests the same thing.

## GIFs
- `/Users/rahul/Downloads/S26-T01-happy-path.gif`: R-01/T-01, the patient's "Everything is still correct", through to Validate and the POA Summary.
- `/Users/rahul/Downloads/S26-T07-cross-trust-yes.gif`: T-07, the cross-trust question, Yes, Please check, and Validate with "Hospital B1 (another trust)".

## Test data created
- **Trust A:** patients "Sam S26 T01 1350", "S26 T05 1410", "S26 T08 1425", "S26 T09 1430", "S26 T11 1440", "S26 T14 1445" and "S26 R05 1450", all born 17/05/1980, with emails `s26.*@example.test`.
- **Trust B:** "S26 T05/T08/T09/T11 …", and "S26 Probe", an unsent HQ left over from a setup script check.
