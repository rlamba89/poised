# S27 test report 3 (retest after verification 1)

**Summary: 5 passed, 0 failed, 0 blocked.** The cases were:
- the two new cases, V-1 (both the patient's tab and Cara's Fill in) and V-2;
- the requested regression cases, T-01, T-02 and T-07.

V-3 is automated only, as the plan says.

- **Environment:** `d8d183b` plus the uncommitted S27 work (git status not clean: the builder's changes, the verification fixes and migration 00027). The conductor ran the app with `make dev` (web http://localhost:3000, API http://localhost:8080, database at 00027). 24 Oct 2026, 13:20–13:35 BST.
- **Browser:** Claude in Chrome, Rahul's profile.
- **Tester:** manual-test agent, automatic run, retest mode.

## How I tested (deviations from the plan)

- **Phone size:** as in rounds 1 and 2, Chrome's window can't be resized from here (`innerWidth` stayed 1512 after a resize to 390 × 844).
  - T-01/T-02 and T-07's patient step ran in a **390 × 780 iframe** of the real app, on a `localhost:3000` page.
  - No sideways scroll: `scrollWidth == clientWidth` (386).
  - V-1 and V-2 ran at desktop size in normal tabs, because V-1 needs two tabs.
- **Earlier steps scripted through the app's own API:** where a case needed earlier steps done first, I scripted them against the same routes the UI calls. These were:
  - Bea's Trust B episodes;
  - the patient sending them;
  - V-2's second Trust B HQ;
  - Cara's Trust A follow-up episodes.

  Everything under test ran in the browser UI:
  - the patient's invite, date of birth and section list;
  - the cards and all three buttons;
  - Start fresh and its confirmation;
  - Fill in for the patient;
  - Send;
  - Validate.
- **The database:** I read it once, only to confirm what I'd seen (`episode_forms.submitted_at` for V-2, and `episode_answers.prefill_choice` for T-07 and the exploration). I changed nothing in it.

## Results

| Case | Result | Note |
|---|---|---|
| V-1 (patient, two tabs) | Pass | "S27 V1 1321": Your health's card was open in two tabs. **Tab 2:** Start fresh, then Yes, start fresh: the set opened empty. **Tab 1, Everything is still correct:** a red message, "The answers in this section were cleared with “Start fresh” somewhere else, so there's nothing here to check. Please reload the page.", and the card stayed. **Tab 1, Check each answer** (after closing the message): the same message, card stays. Both requests were **409**. **Reload of tab 1:** Your health shows **Start**, and opens empty with no card. |
| V-1 step 6 (Cara's Fill in as tab 1) | Pass | Cara opened About you's card on Fill in for the patient. The patient, in their own tab, pressed Start fresh and then Yes. **Cara, Everything is still correct:** the same red message (409), card stays. **Cara, Check each answer:** the same (409). **Cara reloads:** both sets say **Start**, so nothing is left to check. |
| V-2 (all from Hospital B1, two questionnaires) | Pass | "S27 V2 1324" had two sent Trust B HQs. In the second, only Your health was answered (snoring changed to No). The card reads "We've filled in 4 answers from your questionnaire at Hospital B1 on 24 Oct 2026. Please check them.", with **no "including"**. Check each answer shows snoring **No**, from the later HQ. This is the plan's accepted same-day form. See note 1 below. |
| T-01 regression (390 px) | Pass | "S27 T01x 1325": no question came before the sections. Both sets said Please check, and Send was disabled. **Your health card:** "…from your questionnaire at Hospital B1 on 24 Oct 2026…", with the three buttons. **Check each answer:** diabetes No and snoring Yes, each with "From your answers on 24 Oct 2026"; pregnancy empty. No sideways scroll. |
| T-02 regression (390 px) | Pass | **About you → Start fresh:** the "Clear these answers? … This can't be undone." prompt with Yes, start fresh and Keep the answers. **Keep the answers:** the three buttons are back. **Start fresh → Yes:** the set is empty, with no markers and nothing chosen. **Back to the sections and reload:** About you shows **Start** and opens with no card; Your health is Complete. **Answered and sent:** "Thank you, Sam". |
| T-07 regression | Pass | "S27 T07x 1325": the patient entered their date of birth on the Trust A link and opened no section. **Cara, Fill in for the patient:** both sets say Please check. **Your health card:** "We've filled in 4 answers from Sam's questionnaire at Hospital B1 on 24 Oct 2026. Please check them with Sam.", with Start fresh. Everything is still correct jumped to pregnancy. **About you → Start fresh:** "…again with Sam. This can't be undone." Yes opened an empty set. **Answered and sent:** "Sent for Sam S27 T07x 1325". **Validate About you:** "Cara Clinician chose “Start fresh” with the patient on 24 Oct 2026, so the pre-filled answers were cleared and every answer here was given afresh." |

## Failures

None.

## Notes on the plan

1. **V-2's recipe can't give two dates, even on different days.**
   - **What I expected:** the second Trust B HQ's About you would still count as the first HQ's answers.
   - **What happens:** when that HQ is sent, About you's pre-filled answers are frozen into it. Trust A then reads every answer from the **second** HQ only. The form's `prefill` has the same `answeredAt` (the second HQ's `submitted_at`, 12:24:31.106 UTC) on every entry.
   - **The result:** one source, so the card correctly says "your questionnaire at Hospital B1 on …".
   - **Reaching the two-date wording:** a person needs answers that exist only in an older form, from a different day. That can't be built by hand today.
   - **Not a product bug.** The "earlier questionnaires at Hospital B1, the latest on …" wording is covered only by the builder's unit test.

## Found while exploring

1. **A stale Start fresh after Start fresh elsewhere is accepted quietly (fine).**
   - **Steps:** Cara's card was open while the patient pressed Start fresh on About you. Cara then pressed Start fresh → Yes.
   - **What happened:** she got an empty set, with no message, and the server answered 204.
   - **Why it's right:** the result is exactly what she asked for, and the set stayed `{}` at the same revision.
2. **A stale Start fresh after "Everything is still correct" elsewhere is refused (right).** The patient pressed Everything is still correct on Your health, and Cara's stale card then pressed Start fresh → Yes. She got the red "The answers in this section have changed since you opened it, so they weren't cleared. Please reload the page." (409). Nothing was cleared.
3. **The 400 for an unknown choice** now reads "Choose “Check each answer”, “Everything is still correct” or “Start fresh”." (verification finding 4). I checked it with a request from the browser console.
4. **Staff Fill in: the jump target sits partly under the sticky header (cosmetic, seen again).** Everything is still correct on Your health scrolled so that pregnancy's options showed, but its title sat under the top bar. Report 2 noted the same.

## Console and network errors

- **Console:** no errors or warnings on the patient form, Fill in for the patient or Validate.
- **The network log shows 503 for successful 204 replies (a tool artifact, not an app error).**
  - **What the log showed:** Claude in Chrome's network log listed every 204 reply to `…/prefill/{set}` as **503**. These included T-07's Everything is still correct and Start fresh, which plainly worked.
  - **What the app really returned:** a `fetch` of the same request from the page returned **204**, with an empty body.
  - **The stored data agrees:** `prefill_choice` is `still_correct` / `start_fresh` as pressed.
  - The 409s were reported correctly.
- **No other unexpected 4xx or 5xx.**

## GIFs

- `~/Downloads/S27-V1-retest3.gif`: V-1 in the patient's tab. The card is open, Everything is still correct and Check each answer are refused with the red message, and after a reload the set is empty.
- `~/Downloads/S27-T01-retest3.gif`: T-01/T-02 at 390 px, from the invite to "Thank you, Sam".
- `~/Downloads/S27-T07-retest3.gif`: T-07, from Fill in for the patient to Validate's "Cara Clinician chose “Start fresh”…".

## Hand checks for Rahul

None.

## Test data left behind

All the patients use made-up `s27.*@example.test` emails:
- **"S27 V1 1321":** a sent Trust B HQ, and a Trust A follow-up with both sets started fresh (not sent).
- **"S27 V2 1324":** two sent Trust B HQs and a Trust A follow-up (not sent; Your health on Check each answer).
- **"S27 T01x 1325":** a sent Trust B HQ and a sent Trust A follow-up.
- **"S27 T07x 1325":** a sent Trust B HQ and a Trust A follow-up sent by Cara.
- **"S27 E1 1330":** a sent Trust B HQ and a Trust A follow-up (About you started fresh, Your health on Everything is still correct; not sent).

No settings were changed, and Trust B is untouched (Active).

STATUS: ALL_PASSED
