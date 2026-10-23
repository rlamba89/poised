# S25 test report 2 (retest): pre-fill and the patient's confirmation screens

**Summary: 19 passed, 0 failed, 0 blocked** (14 cases + 5 regression cases). Nothing new failed while exploring.

- **Tester:** manual-test agent, automatic run (`/manual-test S25 retest auto`).
- **Environment:**
  - `fe0d30c`, with the S25 build and the builder's fixes uncommitted in the working tree (git status isn't clean).
  - 23 Oct 2026, 12:42–13:02 BST.
  - http://localhost:3000, API on :8080.
  - Chrome, "rahul chrome" profile.
- **Phone size:**
  - I tried `resize_window` 390 × 844 first. It reported success, but the page still measured 2560 px wide, so it had no effect.
  - The patient screens therefore ran in a same-origin **390 × 844 iframe** in their own tab.
  - For an extra check of T-03 I also used a 375 × 667 iframe (a small phone).
  - Staff screens ran in a desktop tab of the same browser. The staff and patient cookies are separate.
- **Test data (all made up):**
  - **Sam S25 MT1243 Round1** (`s25.mt1243a@example.test`): R-A, then 6 follow-up episodes and 1 S25 Short episode.
  - **Sam S25 MT1243 Round2** (`…b@`) and **Round3** (`…c@`): an R-A each (sent, not reviewed), then one follow-up each, for R-03.
  - **Sam S25 Cross 1243** (`s25.cross1243@example.test`): one record in Trust B and one in Trust A.
  - Questionnaire **S25 Short 1243** (Trust A).
- **How I worked:**
  - Invite links came from `GET /api/qa/messages?to=…`, the same data /qa/messages shows.
  - For the setup recipes, and for some reads and checks, I typed, clicked and read the patient iframe through page JavaScript. The steps each case is about were done with real clicks, typing and scrolling.
  - R-A used **Demo pre-op HQ v5**, because v1 is no longer offered (earlier runs published new versions). The same was true in report 1.

## Results

| Case | Result | Note |
|---|---|---|
| R-A / R-B (setup) | Done | First episode reviewed, with the snoring correction (Yes). Each R-B episode was created through the New episode form. |
| T-01 Pre-filled sections wait to be checked | Pass | Both sections show **Please check** / **Check**. "0 of 2 sections complete". Send disabled. No sideways scroll (scrollWidth = clientWidth = 386). |
| T-02 Card counts and date | Pass | "We've filled in 6 answers from your questionnaire on 23 Oct 2026. Please check them." Both buttons are shown, with no questions under them. |
| **T-03** Everything is still correct jumps to the first unfilled question | **Pass (F1 fixed)** | See "Retest of F1" below. |
| T-04 Changing an answer removes its marker; hidden answers aren't lost | Pass | Pregnancy → No. Sickness → No removes its marker. Problem → No hides "What happened". Back to Yes brings back `I was sick all night` and both markers. Next page: the allergy row (Penicillin / A rash) has a marker; medicines are empty. Finish → **Complete**. |
| T-05 Leaving and coming back fills nothing in again | Pass | After a reload the section is still Complete. Review shows no card, sickness No and pregnancy No. |
| T-06 Check each answer walks the section | Pass | The card says 3 answers. Page 1: the box is headed **Your details** (Found 2 fixed) and shows Name / Age 46 / Sex female, with no marker; smoking and alcohol have markers. Page 2: height 170 has a marker; weight and stairs are empty. After weight 80 and stairs Yes: Complete, 2 of 2, Send enabled. |
| T-07 The card shows once; an unanswered card keeps the section waiting | Pass | Leaving with **All sections** keeps Please check. **Everything is still correct** goes to page 2 with the weight input focused and visible. The next open has no card. |
| T-08 Earlier answers come from the newest sent HQ | Pass | After T-06's HQ was sent, the next follow-up has sickness **No** (the T-04 change). The card's date is 23 Oct 2026, the same day as R-A, so the date alone can't tell them apart. |
| T-09 Everything is still correct with nothing left moves on | Pass | "We've filled in 1 answer … Please check it." It goes back to the list: Complete, 1 of 1, Send enabled. The home now says **In progress** for it (it said To do in report 1). |
| T-10 Filling in for the patient | Pass | Both sections Please check. The card: "We've filled in 4 answers from Sam's questionnaire on 23 Oct 2026. Please check them with Sam." **4** is correct: the sent T-06 HQ answered stairs, as in report 1. It goes to page 2 with weight focused, and the markers read "From Sam's answers on 23 Oct 2026". Opened as the patient afterwards: *About you* has no card (In progress / Continue), and *Your health* is still Please check. |
| T-11 Another trust's answers are never used | Pass | Bea's Trust B patient filled in and sent the HQ. The Trust A patient with the same details and email gets **Start** on both sections, no card, empty answers and no markers. |
| T-12 Clinician-only answers are never pre-filled | Pass | Sent T-10's HQ as Cara (Send the answers → in-app confirm). Validate → Nurse checks: Blood pressure is empty. |
| **T-13** Home doesn't call an untouched pre-filled HQ "started" | **Pass (Found 1 fixed)** | Step 2: untouched shows **To do / Start**. Step 3: after an answer changed, **In progress / Continue**. **Step 4 (new):** on Round3's follow-up, Check → Everything is still correct with nothing changed: the home says **In progress / Continue**. |
| T-14 Deep links and the back button | Pass | Reloading on a card goes back to the list, still Please check. Home → HQ → Back → Forward → Back: the right page each time, no errors. |
| R-01 (S14) New patient with no earlier HQ | Pass | R-A's first HQ (and Round2/Round3/Cross): Start on both sections, no card. Filled in and sent: "Thank you, Sam". |
| R-02 (S02a) Two tabs | Pass | Two iframes on one follow-up section. Pregnancy Yes saved in one; No in the other gave "Your answers weren't saved … changed somewhere else … Reload". After Reload it shows Yes. |
| **R-03** (S16) Reminders | **Pass** | Jobs run as of Now + 2 days: 26 jobs, **1 reminder sent**. **Round2** (untouched pre-filled follow-up) got "a reminder about your health questionnaire". **Round3** (T-13 step 4: card answered, nothing typed) got **no** reminder. Round1 got none either: all its open follow-ups have a card answered or an answer changed. Checked on /qa/messages data. |
| R-04 (S20/S22) Cancelled episode | Pass | Cancelled one Round1 follow-up (reason Patient withdrew; an in-app panel, no browser dialog). The patient's home shows **Cancelled by the hospital … nothing to fill in**. Opening it shows "This questionnaire was cancelled by the hospital…" with no form. Its invite link shows the general "This link doesn't work", as in report 1 and S20's plan. |
| R-05 (S23/S24) Editor pre-fill labels | Pass | "Do you have diabetes?" shows **Pre-fills from 100 other HQs**, and the list starts **Demo pre-op HQ · by same wording** (then E2E Audit HQs). "Do you smoke?" shows **Pre-fills from 6 other HQs**: **Demo pre-op HQ · by link**, S24 Lib B HQs and S25 Short 1207. The editor also shows the details box as **Your details**. |

## Retest of F1 (T-03, phone-width scroll)

- **Fixed.** At 390 × 844, after **Everything is still correct** on *Your health*:
  - the pregnancy question's card is at the very top of the screen (top = 0, `scrollY` 671);
  - its text "Is there any chance you could be pregnant?" is fully visible (title 16–64 px from the top);
  - the focus is on its first option.
- **Stable:** the position was the same at 0.3, 0.8, 1.5 and 3 s.
- **Repeated three times** (Round1, Round3, Round1 again), and at 375 × 667 on Round2: the title was at 16 px each time.
- **The hold doesn't fight the patient:** I scrolled the wheel up about 0.4 s after the button (on *About you*, whose target is weight on page 2). The page went to the top and stayed there, so the "keep it in place" logic stops when the patient scrolls.
- The GIF `S25-retest-T13-T03.gif` shows it.

## Failures

None.

## Found while exploring

1. **Another patient's episode:** I opened Round1's episode URL while signed in as Round2. The page shows "Episode not found." A `POST /api/p/episodes/<Round1's episode>/prefill/<set>` from Round2's session returns **404**.
2. **(Observation, not S25) Patient home header:** on /home the header has only **Sign out**, while the HQ screens also show "Hospital A1". This is probably intended, because the home lists several hospitals.
3. **(Observation) Statuses agree now:** a fully pre-filled, confirmed HQ (S25 Short) shows **In progress / Continue** on the home, and the list says Complete, 1 of 1. The home and the list no longer disagree.
4. **Still open from report 1 (no change expected):**
   - the details box says "Sex **female**" in lowercase (the profile template, noted in pending.md);
   - "Pre-fills from **100** other HQs" on diabetes (S23's label, inflated by E2E HQs).

## Console and network errors

- No console errors or warnings in the staff tab or the patient's top-level tab (tracked from 12:59; I reloaded a staff page to capture its load).
- The iframe's own console isn't captured by the tool.
- All staff API calls on the reloaded episode page returned 200. (They were duplicated, which is React's development double render.)
- The only non-2xx responses were the expected ones:
  - the 404 probe above;
  - R-02's conflict.

## GIFs

- `~/Downloads/S25-retest-happy-path.gif`: T-01 to T-06 at phone size: date of birth, both sections waiting, the card, Everything is still correct landing on the pregnancy question, marker changes, Check each answer with "Your details", Complete.
- `~/Downloads/S25-retest-T13-T03.gif`: T-13 step 4 on Round3: Check → Everything is still correct (the pregnancy question on screen) → Your questionnaires shows In progress.

## Hand checks for Rahul

- None needed for this round.
- Optional: T-03 in Chrome's real device toolbar at 390 × 844, because `resize_window` had no effect here and an iframe stood in for it. Report 1's same iframe method reproduced the bug, and it now passes.
