# S26 test report 3 (retest after verification 1)

**14 passed, 0 failed, 0 blocked.** This retest covers the changes made after verification 1 and Rahul's cross-trust matching rule:
- **T-07:** platform library keys match "by link", everything else "by same wording", and choices are mapped by their option text. Trust A's `…/hq` has no Trust B ids.
- **The consent card's error message stays on screen.**
- **Regression checks:** T-01, T-06, T-08, T-09, T-11, T-12, T-13, R-01 and R-02.

**Environment**
- Commit `08c7645`. The working tree isn't clean: it holds the uncommitted S26 build plus the fixes.
- 23 Oct 2026, 15:01–15:14 BST, on http://localhost:3000 (API on :8080). The conductor's `make dev` was running on the latest code.
- **Browser:** Claude in Chrome, Rahul's "rahul chrome" profile.
  - Patient screens ran in a same-origin 390 × 844 iframe.
  - Staff screens ran at desktop size.
- **Settings:** `crossTrustPrefill` was off in both trusts at the start (`trust: {}`). Both were switched on through `PATCH /api/o/{oid}/settings`, with Ada's and Max's dev-login sessions.
  - **At the end, both were set back to off.** Both trusts now show `crossTrustPrefill: false` and an effective value of false (checked through the API).
  - They're now explicit `false` values, not cleared overrides, so the effective value is the same as before.

**What I did through the API rather than the UI** (setup only, with the e2e helpers and the real API):
- **T-01's first Trust A Demo pre-op HQ:** answered and sent, then reviewed by Cara, who corrected snoring to Yes.
- **Trust B Demo pre-op HQs, answered and sent,** for T-07, T-08, T-09 and a new T-11 person. Each had snoring Yes, sickness after an anaesthetic Yes, an anaesthetic problem "I was sick all night", and Penicillin / A rash. T-11's was smaller.
- **T-08:** first a Trust A pre-op HQ (snoring No, reviewed), then the Trust B one, with both settings still off.
- **The Trust A Demo follow-up HQ episodes:** the second follow-ups for T-09 and T-07, and both of T-11's.

**Through the UI:**
- T-01's second episode: Cara searched for the patient on New episode and created it.
- Every patient step: the invite, Continue, the date of birth, the consent card, the opening card, the answers, and Send with its confirmation.
- Validate, Fill in for the patient, and both trusts' Audit pages.
- Clicks inside the phone iframe were real mouse clicks. I worked out their positions with a small script.

## Results

| Case | Result | Note |
|---|---|---|
| **T-07** Yes fills from Trust B (rule change) | Pass | **Patient (phone):** after **Yes, use them**, both sections show Please check and the card is gone. A reload doesn't ask again.<br>**Your health card:** "We've filled in 6 answers from your questionnaire on 23 Oct 2026", with "From your answers on 23 Oct 2026" under each answer. Diabetes No, snoring Yes, sickness Yes, problem Yes, and "I was sick all night". Allergies (Penicillin / A rash) are filled on page 2.<br>**About you:** 3 answers. Smoking shows **"I used to smoke"**, which is mapped by its option text, plus alcohol 6 and height 170. Weight and stairs are empty.<br>**Validate (Cara):** each answer says "Pre-filled from Demo pre-op HQ at Hospital B1 (another trust), 23 Oct 2026", followed by:<ul><li>**(matched by link)** for snoring, sickness, the anaesthetic problem, what happened, allergies, smoking and alcohol (and height);</li><li>**(matched by same wording)** for diabetes.</li></ul>**The staff API:** I opened `…/episodes/<eid>/hq` as Cara in the browser. It has 9 entries with `"otherTrust": true` and `"hospital": "Hospital B1"`, with the keys value, hospital, matchedBy, answeredAt, otherTrust and questionnaire.<ul><li>There's **no `fromForm`**.</li><li>The only uuids in the response are Trust A's three chapter ids. None of Trust B's episode, patient, trust or hospital ids appear.</li><li>Diabetes is stored as Trust A's own option value `"n"`, not Trust B's `"o_no"`, which confirms the mapping by option text.</li></ul> |
| **Consent error stays visible** (fix) | Pass | I caused the error by hand. T-09's patient had two phone frames open on the consent card. In the first I pressed **No, thanks**; in the stale second one I then pressed **Yes, use them**.<br>The second frame shows a red message, "This question has already been answered, or no longer applies. Please reload the page.", with a close ×. It was **still on screen 7+ seconds later**, above the reloaded tiles. Those show **Start**: the No was kept. GIF below. |
| T-01 Validate markers (happy path) | Pass | **Your health:** "6 answers here were pre-filled … and the patient changed 1. The patient chose “Everything is still correct” on 23 Oct 2026, so these pre-filled answers weren't checked one by one."<ul><li>Diabetes says "(matched by same wording)".</li><li>Snoring is **Yes**, "(matched by link)", which is Cara's correction.</li><li>Sickness is No, with "Changed by patient: pre-filled “Yes”" in orange/red.</li><li>Pregnancy has no marker.</li><li>Problem and What happened have markers. None of these markers mention another trust.</li></ul>**About you:** "3 answers … The patient chose “Check each answer” on 23 Oct 2026." Smoking and alcohol have markers. |
| R-01 Same-trust card | Pass | "We've filled in 6 answers from your questionnaire on 23 Oct 2026. Please check them." **Everything is still correct** jumps to the pregnancy question. Changing sickness to No removed its marker at once. |
| R-02 Started | Pass | T-11's person had pressed Yes, so their HQ was pre-filled but not touched. Their home showed the Hospital A1 HQ as **TO DO**, with Start. T-01's second HQ showed Please check on both sections. |
| T-06 The question (phone) | Pass | "Use your earlier answers?" / "Use the answers you gave Hospital B1 in October 2026?", the explanation, and Yes / No buttons, each 314 px wide in the 388 px frame. There's no sideways scroll (`scrollWidth` 388 = `innerWidth`) and no section tiles. |
| T-08 Trust A first | Pass | After Yes, Your health shows **snoring No** (Trust A's) with the other Trust A answers. After Send, Validate shows "Pre-filled from Demo pre-op HQ, 23 Oct 2026 (matched by link)" **without** "at Hospital B1 (another trust)", and diabetes "(matched by same wording)". |
| T-09 No, thanks remembered | Pass | **No, thanks:** both sections show **Start**. After a reload: no question, still Start. **A new follow-up episode** for the same person (a new invite, date of birth): no question, both sections Start. |
| T-11 On behalf: never Trust B | Pass | **A person with only a Trust B history, who had said Yes** (on an unsent Trust A HQ). Cara's second episode for them, **Fill in for the patient**: no question, both sections **Start**, and opening Your health shows no answers, cards or markers.<br>**T-07's person:** Fill in for the patient showed Please check, but the `…/hq` response shows every entry comes from their **own Trust A** Demo follow-up HQ (sent at 15:04, no `otherTrust`, no hospital). That is the "also has a Trust A history" exception, so it's correct. |
| T-12 Audit, both trusts | Pass | **Trust A** (Ada, filter "R3T"): "Patient answered whether to use another trust's answers" for T-07, T-08, T-09 and T-11, all DONE. T-13's refusals for T-09 show REFUSED (400)/(409). The API response has none of the Trust B patient, episode, trust or hospital ids.<br>**Trust B** (Max, Hospital B1): "Patient's answers read to pre-fill their HQ at another trust (they agreed)" for T-07, T-08 and T-11, by "Patient (their link)", at Hospital B1. The API shows the Trust B record and episode (T-11: episode `f5998c60…`). There's **none for T-09** (No), and none from the staff fill-in. |
| T-13 Refused requests | Pass | Run from the patient's own page (T-09's `/home/episodes/cd5bc43d…`):<ul><li>Trust B, already answered: 409. An unknown uuid: 409. Both say "This question has already been answered, or no longer applies. Please reload the page."</li><li>`not-a-uuid`, and an orgId with leading spaces: 400 "We couldn't tell which answers this is about. Please reload the page."</li><li>No `agree`: 400 "Choose “Yes, use them” or “No, thanks”."</li><li>The zero uuid, and another person's episode: 404 "Episode not found."</li><li>Signed out (`curl`, no cookies): 401.</li></ul> |

## Failures
None.

## Found while exploring
1. **Still there, and older than S26** (round 1's found 1, round 2's found 1): Validate shows a false correction marker on a list's optional column.
   - **Where:** Your allergies, under "What happens when you have it?". The field holds "A rash".
   - **What it shows:** "Patient answered: no answer".
   - Seen again on T-07's episode, where the allergy came from Trust B.
2. **Observation (as the plan expects):** T-08's person was asked about Trust B's answers and said Yes. Trust A's own history covered every question, so nothing from Trust B was used.
   - Trust B still logs a "read … (they agreed)" row for them.
   - The plan lists this row as expected. I note it only because the read had no effect.
3. **Not a bug:** on New episode, choosing an existing patient pre-fills **Hospital number** from their record ("E2E"). Typing then appends to it. That's the S13 behaviour.
4. **Checked, no problem:** two tabs on the consent card. The server keeps the first answer, and the stale tab explains itself and reloads its tiles.

## Console and network errors
- No console errors or warnings on the staff pages after the console tracking started: Validate, Fill in for the patient, and Audit.
- No unexpected 4xx or 5xx. The only refusals were T-13's deliberate ones and the deliberate stale-tab 409.

## Hand checks for Rahul
None.

## GIFs
- `/Users/rahul/Downloads/S26-retest3-T07-cross-trust-yes.gif`: T-07 on the phone. The invite and date of birth, the cross-trust question, Yes, Please check, the opening card, the filled answers with markers, the remaining answers, and Send.
- `/Users/rahul/Downloads/S26-retest3-consent-error-stays.gif`: two phone frames. No, thanks in one, then Yes in the stale one: the error message stays on screen.

## Test data created
All are born 17/05/1980, with emails `s26r3.*@example.test`.
- **Trust A:**
  - "Sam S26 R3T01 1505": the pre-op HQ (reviewed) and the follow-up HQ (sent).
  - "Sam S26 R3T07 1505": the follow-up HQ (sent), and a second one, not started.
  - "Sam S26 R3T08 1505": the pre-op HQ (reviewed) and the follow-up HQ (sent).
  - "Sam S26 R3T09 1505": two follow-up HQs, not started.
  - "Sam S26 R3T11 1505": two follow-up HQs, not started. The first is pre-filled from Trust B.
- **Trust B:** a sent Demo pre-op HQ each for R3T07, R3T08, R3T09 and R3T11.
