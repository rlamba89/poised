# S27 test report 4 (short retest after verification 2)

**Summary: 3 passed, 0 failed, 0 blocked.** The cases were T-01 and T-02 as regression, plus a look at the card wording that the seeded trusts can reach. The plan's top note says no manual retest is needed for verification 2's two changes. The first, `test_audit.py`, is automated. The second, the wording for two or more other hospitals, can't be built with the seeded trusts. The conductor asked for this short regression round anyway.

- **Environment:** `d8d183b` plus the uncommitted S27 work (git status not clean: the builder's changes, the verification fixes and migration 00027). The app was already running under `make dev` (web http://localhost:3000, API http://localhost:8080), started by the conductor after `make migrate seed`. 24 Oct 2026, 14:14–14:30 BST.
- **Browser:** Claude in Chrome, Rahul's profile.
- **Tester:** manual-test agent, automatic run, retest mode.

## How I tested

- **Phone size:** as in earlier rounds, the patient's pages ran in a **390 × 780 iframe** of the real app, on a `localhost:3000` page. No sideways scroll: `scrollWidth == clientWidth` (386).
- **Earlier steps scripted through the app's own API:** I used the e2e helpers (`e2e/support.py`) from a scratch script and changed no repo files. The script did these steps:
  - Bea's Trust B episode;
  - the patient answering and sending it;
  - Cara's Trust A follow-up episode.

  Everything under test ran in the browser UI: the invite, the date of birth, the section list, the cards, Start fresh and its confirmation, answering, and Send.
- **The database:** read once, only to explain the wording check below. I changed nothing in it.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 regression (390 px) | Pass | "Sam S27 T01r4 1414". No question came before the sections. Both sets said **Please check**, and **Send my answers** was disabled. The Your health card read "We've filled in 4 answers from your questionnaire at Hospital B1 on 24 Oct 2026. Please check them." It had Check each answer, Everything is still correct and Start fresh, and **no "including"**. Check each answer showed diabetes No and snoring Yes, each marked "From your answers on 24 Oct 2026"; pregnancy was empty. No sideways scroll. |
| T-02 regression (390 px) | Pass | The About you card read "…3 answers from your questionnaire at Hospital B1…". **Start fresh** showed "Clear these answers? You'll answer every question in this section again. This can't be undone." **Keep the answers** brought the three buttons back. **Start fresh, then Yes, start fresh** opened an empty set: no card, nothing chosen, no markers. **Back to the sections, then reload:** About you showed **Start** and opened with no card; Your health still had its answers. **Answered both sets and sent:** "Thank you, Sam". |
| Card wording reachable with the seeded trusts | Pass | **One other hospital** (T-01): "…from your questionnaire at Hospital B1 on …", with no "including". **This trust only** ("Sam S27 MX 1414", see the note below): "…from your questionnaire on 24 Oct 2026…", with no hospital named and no Start fresh, as in T-03. |

## Failures

None.

## Notes on the plan

1. **The "including one at Hospital B1" wording (this trust's answers mixed with another trust's) can't be reached with the seeded questionnaires either.**
   - **What I tried:** "Sam S27 MX 1414" had three questionnaires:
     - a sent Trust A Demo pre-op HQ, with About you fully answered and only diabetes answered in Your health;
     - then a fully answered, sent Trust B Demo pre-op HQ;
     - then a Trust A follow-up.
   - **What happened:** the follow-up was filled only from Trust A's own HQ. Your health's card read "…1 answer from your questionnaire on 24 Oct 2026…", and the stored `prefill` holds only Trust A's entries.
   - **Why this is right:** another trust's answers fill only questions that this trust's own forms **didn't ask**. Trust A's pre-op HQ asked snoring and the rest, even though they were left unanswered. CLAUDE.md states this rule.
   - **What a mix would need:** Trust A history from a questionnaire that asks fewer questions than Trust B's. None is seeded, so the "including" and two-other-hospitals wordings are covered only by the unit tests, as the plan says.

## Found while exploring

Nothing new. This round was kept short at the conductor's request.

## Console and network errors

- **Console:** no errors or warnings on the patient's pages.
- **Network:** no unexpected 4xx or 5xx (`/api/config` and `/api/p/episodes/…` all 200).
- **A brief extension disconnect:** Claude in Chrome dropped out once while typing the alcohol units. I checked that the value had been typed. This is a tool hiccup, not an app error.

## GIFs

- `~/Downloads/S27-T01-retest4.gif`: T-01 and T-02 at 390 px, from the invite to "Thank you, Sam".

## Hand checks for Rahul

None.

## Test data left behind

All the patients use made-up `@e2e.example` emails.
- **"Sam S27 T01r4 1414":** a sent Trust B HQ and a sent Trust A follow-up (About you started fresh).
- **"Sam S27 MX 1414":** two sent HQs, Trust A then Trust B, and a Trust A follow-up (not sent; the cards were opened, no choice made).

No settings were changed.

STATUS: ALL_PASSED
