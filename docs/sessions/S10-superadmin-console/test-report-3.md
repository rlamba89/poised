# S10 test report 3 (retest after verification 1)

**Summary:** 4 passed, 0 failed, 0 blocked (T-22, T-23, T-12, T-13). Other cases weren't re-run: the plan asks only for these after verification 1. R-1 was rechecked in passing.

- **Commit:** `2c3054d`. The working tree isn't clean: the S10 code and the planning docs aren't committed yet, as expected in the automatic run.
- **When:** 13 Oct 2026, from 08:34 BST.
- **URLs:**
  - http://localhost:3000, signed in as Sam Superadmin.
  - http://tester.localhost:3000, for the second person: Zoe Admin Seven, Eight and Nine, Bea, Max, Cara, Ada, Sue and Alex.
  - The conductor's `make dev` was running on the latest code.
- **Browser:** Chrome, through Claude in Chrome, at desktop size.
- **How I tested:**
  - Through the UI.
  - I used page scripts only to read the sign-in alert text, count episode rows by hospital, and read `/api/me` and one API status while exploring.
  - I read the database once at the end, to confirm that everything I had suspended was active again.
- **Fresh data:**
  - QA Trust Seven (QA7, with QA Seven General; admins Zoe Admin Seven and Sam);
  - QA Trust Eight (QA8, with QA Eight General; admin Zoe Admin Eight);
  - QA Trust Nine (QA9, no hospital; admin Zoe Admin Nine).

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-22 | **Pass** | Fix 1 works. <br>• **Steps 1–2:** I created QA Trust Seven and QA Seven General. Inviting `sam.superadmin@poised.example` gave "sam.superadmin@poised.example already had an account, so they were made an admin of QA Trust Seven.", and Sam is listed ACTIVE. <br>• **Step 3:** after signing in again, Sam goes straight into QA Seven General (Questionnaires, role ADMIN), and the header has a **Platform admin** link. <br>• **Step 4:** the trust was suspended through the in-app dialog. <br>• **Step 5:** signing in as Sam **works**, with no suspended message. He lands on `/admin`, the header has no "Your hospitals" link, and `/` shows only the "You're one of Poised's platform admins." card. <br>• **Step 6:** Re-activate → "QA Trust Seven is active again. Its staff can sign in." `/` then goes straight into QA Seven General again. <br>• **Step 7:** while the trust was suspended, Zoe Admin Seven was refused with "Your trust or hospital is suspended in Poised at the moment, so you can't sign in. Please contact your trust." After re-activation she signs in to QA Seven General. |
| T-23 | **Pass** | Fix 2 works. <br>• **Steps 1–2:** Zoe Admin Eight signs in straight to QA Eight General. <br>• **Step 3:** Sam suspended QA Eight General (the hospital); the dialog says "Staff of the whole trust keep its other hospitals". <br>• **Step 4:** her Episodes shows "Hospital not found.". `/` shows "You don't have access to any hospital yet", **not** "…has no hospitals in Poised yet". <br>• **Step 5:** signing in again is refused with the suspended message. <br>• **Step 6:** after re-activation she signs in straight to QA Eight General. <br>• **Step 7:** Zoe Admin Nine, of the brand-new QA Trust Nine with no hospitals, still gets "Your trusts" with "QA Trust Nine has no hospitals in Poised yet. Poised's platform admin adds them…". |
| T-12 | **Pass** | On the seeded Trust B. <br>• Bea's open Questionnaires page shows "Hospital not found." after the suspension. <br>• Signing in again gives the suspended refusal, and she stays on `/login`. <br>• Max Multi signs in straight to Hospital A2 Episodes, and his hospital box lists only Trust A / Hospital A2. <br>• After re-activation, Max's picker shows Trust A (A2, Clinician) and Trust B (B1, Admin, whole trust), and Bea signs in to Hospital B1 again. |
| T-13 | **Pass** | Hospital A1 suspended: it shows SUSPENDED with Re-activate, and Trust A stays ACTIVE. <br>• Cara and Sue Super (both A1 only) are refused with the suspended message. <br>• Ada signs in straight to Hospital A2, and her hospital box lists only A2. Episodes → All hospitals lists 17 rows, all at Hospital A2 and none at A1. <br>• A1's old Episodes address shows "You don't have access to this hospital". <br>• Alex Author (super clinician of the whole trust) also signs in straight to A2. <br>• After re-activation, Cara signs in to A1 Episodes, and Ada's picker lists A1 and A2. |
| R-1 | Pass (in passing) | Max's picker lists Trust A (Hospital A2, Clinician) and Trust B (Hospital B1, Admin, whole trust). Switching wasn't re-run this round. |

## Failures in detail

None.

## Found while exploring

- **The sign-in rules hold in the mixed cases (good):**
  - **Sam with a suspended hospital:** Sam was a platform admin, and QA Seven General (his only hospital) was suspended while the trust stayed active.
    - He still signs in, to `/admin`.
    - `/api/me` returns `"trusts":[]`, so the trust isn't shown as "no hospitals yet".
    - Opening the hospital's Episodes address shows "You don't have access to this hospital".
    - `GET /api/o/<QA7>/h/<QA Seven General>/episodes` → 404 "Hospital not found.".
  - **Max with Hospital B1 suspended:** Trust B's only hospital was suspended while Trust B stayed active.
    - Max signs in to A2.
    - `/api/me` lists only Trust A, with no empty Trust B entry. This is consistent with T-23.
  - **Ada's Settings while A1 was suspended:** the "Settings for" list shows "Hospital A1 (suspended)". That's clear.
- **Minor, wording:**
  - Zoe Admin Eight is admin of the whole trust. When her trust's only hospital is suspended, `/` says "You don't have access to any hospital yet. Ask an admin of your trust to invite you."
  - She *is* the trust's admin, and "yet" suggests nothing has been set up.
  - T-23 step 4 expects this heading, so this isn't a failure. Wording that fits a suspension, such as "Your hospitals are suspended at the moment", would be clearer.
  - In practice she only sees it in a session that was already open, because signing in again is refused.
- **Cosmetic, header link not updated:**
  - When Sam re-activates a trust or hospital where he is staff, the console header's **Your hospitals** link doesn't appear until the page is reloaded. The header's copy of `/api/me` isn't refreshed.
  - The same happens in reverse after a suspension.
  - A reload fixes it.

## Console and network errors

- **Console:** no errors or warnings in either tab after tracking started.
  - Tracking started late in the round, so I reloaded the console page and Max's start page to cover page loads.
- **Network:** no 4xx or 5xx on those loads. Admin GETs are sent twice, from React's development double render. This is harmless, as in earlier rounds.
- Every 404 and refusal seen during the cases was one the plan expects.

## Data left behind

- **New trusts:**
  - QA Trust Seven (QA7; QA Seven General). Its admins are Zoe Admin Seven and **Sam Superadmin**: Sam is now staff of this trust, so his sign-in and `/` go straight into QA Seven General.
  - QA Trust Eight (QA8; QA Eight General; Zoe Admin Eight).
  - QA Trust Nine (QA9, no hospitals; Zoe Admin Nine, invited and signed in).
- **New people:** Zoe Admin Seven, Eight and Nine.
- **Everything is active again:** checked in the database at the end. Trust A, Trust B, A1, A2, B1 and the QA trusts and hospitals are all active, and nothing is suspended.

## GIFs

`S10-r3-T22-T23.gif` covers the main path: creating QA Trust Seven, Sam as its admin, suspending it, Sam still signing in, re-activating it, then T-23's hospital suspension. Claude in Chrome exported it to Chrome's download folder (normally `~/Downloads`). There are no failures, so there are no failure GIFs.

## Hand checks for Rahul

None. The suspend confirmations are in-app dialogs, not browser dialogs.
