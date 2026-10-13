# S10 test report 2 (retest)

**Summary:** 18 passed, 0 failed, 0 blocked, with 8 cases not re-run. Report 1's failures (T-15 F1 and F2) are fixed. The new cases T-20 and T-21 pass. Regression R-1 to R-5 all pass.

- **Commit:** `2c3054d`. The working tree isn't clean: the S10 code and the planning docs aren't committed yet, as expected in the automatic run.
- **When:** 13 Oct 2026, from 08:01 BST.
- **URLs:**
  - http://localhost:3000, signed in as Sam Superadmin.
  - http://tester.localhost:3000, for the second person: Zoe Admin Five, Zoe Clinician Five, Zoe Admin Six, Ada, Max, Cara and Alex.
  - The conductor's `make dev` was running against a freshly migrated and seeded database.
- **Seed reset (checked in the database before starting):** Trust A's and Trust B's settings were `{}`, and the hospitals had no settings of their own. So the new `make seed` behaviour works.
- **Browser:** Chrome, through Claude in Chrome, at desktop size. No phone-size cases.
- **How I tested:**
  - Through the UI.
  - I used page scripts only to read field values and alerts.
  - I used `fetch` only for the API checks the plan asks for (T-20 step 5), plus a few extra validation probes. These are marked below.
  - I read the database only to confirm the seed reset and the cleanup.
- **Fresh data this round:** the trusts from report 1 (QA Trust One and Two) still exist, so I used new ones:
  - QA Trust Five (QA5), with admin Zoe Admin Five and clinician Zoe Clinician Five;
  - QA Trust Six (QA6), with admin Zoe Admin Six.
- **Tooling note:** the tabs often ran in the background, which slows Mantine dialog animations. A few times, text typed or a click made while a dialog or page was still opening was lost, so I retried. None of these is reported as an app bug.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Sam lands on `/admin` "Trusts" with the PLATFORM ADMIN badge. `/` shows the platform-admin card. |
| T-02 | Pass | QA Trust Five / `qa5` → "Code QA5". The green "…was created. An invitation was sent to zoe.admin5.r2@test.example. Next, add its hospitals." appears. Contact and logo address are shown. Zoe Admin Five is "INVITED · EXPIRES 20/10/2026". "No hospitals yet…" is shown. PROMs is On with the defaults. GIF `S10-r2-T02-T20-T21.gif`. |
| T-03 | Not re-run | Passed in report 1. Not changed since. |
| T-04 | Not re-run | Passed in report 1. Duplicate codes on **edit** were checked in T-20. |
| T-05 | Pass (steps 1–4) | QA General has a two-line address, its phone, "Trust's email", "Europe/London (default)" and ACTIVE. QA North shows America/Toronto without "(default)". `qa general` → "This trust already has a hospital called qa general." and the dialog stays open. Steps 5–7 weren't re-run (they passed in report 1). |
| T-06 | Pass | Zoe Admin Five gets the picker (QA General and QA North, Admin) and a header with Staff and Settings. Her invite of Zoe Clinician Five → "Invitation sent to …". Zoe Clinician Five goes straight to QA General Episodes as CLINICIAN, with no Staff or Settings. After a reload, Sam sees Zoe Admin Five as ACTIVE. |
| T-07 | Pass | Zoe Admin Six's start page is headed **"Your trusts"** (fixed). It lists QA Trust Six (Admin, whole trust) with the note "…has no hospitals in Poised yet…". After Sam added QA Six General, a reload took her straight in. |
| T-08 | Pass (step 3) | Inviting Zoe Admin Six again → "Zoe Admin Six already has access to the whole trust. The trust's admins can change their role on the Staff screen." The dialog stays open. Report 1's wording issue is fixed. Enter submits the dialog. |
| T-09 | Not re-run | Passed in report 1. |
| T-10 | Not re-run | Passed in report 1. The new edit routes were checked in T-20 step 5. |
| T-11 | Not re-run | Passed in report 1. |
| T-12 | Pass (on QA Trust Five) | I used my own trust, not Trust B: the dialog explains the effect, then "QA Trust Five Renamed is suspended. Its staff can't sign in.", a SUSPENDED badge and the red box. Zoe Clinician Five's sign-in → "Your trust or hospital is suspended…". Re-activate → "…is active again. Its staff can sign in." |
| T-13 | Not re-run | Passed in report 1. Suspending and re-activating a hospital was rechecked on QA North (see "Found while exploring"). |
| T-14 | Pass | "PROMs at Hospital A1 is on." now appears **just above the opt-ins table**, visible without scrolling (fixed). EQ-5D-5L trust On → "Follow trust (On)" for both, then A2 set to Off. Kept after a reload. Ada sees A1 PROMs ON and EQ ON, and A2 PROMs OFF and EQ OFF, each "Set by Poised's platform admin". Everything was set back to follow the trust. |
| T-15 | **Pass** | Both of report 1's failures are fixed. **F1:** at Hospital A2 (and A1), the "Whole trust" group is shown, with "Keep episode data" **10** and "Offer answers…" **OFF** as "Set for the whole trust" badges. **F2:** "Trust A's settings were saved." and "Hospital A2's settings were saved." appear directly above the Save button, in view. Steps 1 and 3–8 also pass: <br>• step 1: defaults 8, "2, 5" and the default texts, greyed and ticked; <br>• step 3: A2 shows the trust's phone, greyed; <br>• steps 4–5: A2 keeps its own phone and text after a reload, and A1 shows the trust's phone; <br>• step 6: ticking "Use the trust's setting" puts it back; <br>• step 7: switching the header box stays on Settings; <br>• step 8: "Nothing has changed." **replaces** the saved message. |
| T-16 | Not re-run | Passed in report 1. |
| T-17 | Pass | The "Default health questionnaire" list can now be searched: typing `145942` narrows it to "E2E explore 1210 145942 080c". Picked, saved, and kept after a reload. |
| T-18 | Pass (steps 1, 2 and 4) | Alex on `…/A1/settings` → "Only the trust's admins can change its settings", with no Settings link. Max on `…/A2/settings` is refused. **Step 4 (new):** from the refused page, Max switches the header box to Hospital B1 and lands on **B1's Settings** (Trust B, ADMIN), not Questionnaires. Step 3 wasn't re-run. |
| T-19 | Not re-run | Passed in report 1. |
| T-20 | **Pass** | **Edit details** opens filled with name, code, logo and contact. Saving name `QA Trust Five Renamed`, code `qa5r`, phone `0100 999 9999` and an empty logo → "QA Trust Five Renamed's details were saved." The heading, "Code QA5R", the phone and no logo are shown, and the Trusts list shows the new name and QA5R. Zoe Clinician Five's header shows "QA Trust Five Renamed" after a reload. Each bad value is refused in the dialog, which stays open: <br>• `TA` → "Another trust already has the code TA. Choose another."; <br>• empty name → "Enter the trust's name."; <br>• email `nope` → "Contact: Enter a valid email address, like name@hospital.org."; <br>• `javascript:alert(1)` → the logo message; <br>• `qa1` → "…already has the code QA1…". <br>Its own code typed in lower case (`qa5r`) saves. Cancel changed nothing, and reopening shows the saved values. As Ada: GET and PATCH of `/api/admin/orgs/<id>` → 403 "Only Poised's platform admins can do this.", and so does PATCH of `…/hospitals/<id>`. GIF `S10-r2-T02-T20-T21.gif`. |
| T-21 | **Pass** | **Edit** on QA General opens filled with its name, address, phone, empty email, "Default (Europe/London)" and empty logo. I saved QA Central with the address "9 Central Square", the phone cleared, email `central@qa.example` and America/Toronto. → "QA Central's details were saved." appears just above the Hospitals table. The row shows the new name and address, "Trust's phone", the email, and America/Toronto without "(default)". The opt-ins column header also renamed. Zoe Clinician Five's header shows "QA Central". The bad values are refused: <br>• `QA North` → "This trust already has a hospital called QA North."; <br>• an empty address → "Enter the hospital's address.". <br>Setting the zone back to "Default (Europe/London)" → "Europe/London (default)" again. |
| R-1 | Pass | Max's picker shows Trust A (A2, Clinician) and Trust B (B1, Admin, whole trust). Switching in the header changes the trust and the role (T-18 step 4). |
| R-2 | Pass | This round I covered it with S09's Staff invite as a trust admin (Zoe Admin Five → Zoe Clinician Five, in T-06): the invitee was Invited, signed in, and then showed as Active. |
| R-3 | Pass | Ada → Episodes → "All hospitals" lists 286 episodes with a Hospital column: 271 at A1 and 15 at A2. |
| R-4 | Pass | Cara created an episode ("MT S10 r2 R4 hip") on a published HQ. The patient link opened the HQ, and answering "No" was saved (`q_smokes: o_no`, checked with GET `/api/p/<token>`). |
| R-5 | Pass | Alex opened the draft "E2E test case 1310 075911 eeb5" and set the "About you" description to "MT S10 r2 R5 description". It was kept after a reload. |

## Failures in detail

None.

## Found while exploring

- **Editing a suspended trust or hospital keeps it suspended (good).**
  - I edited the suspended QA Trust Five's phone: it saved, and the trust stayed SUSPENDED until I re-activated it.
  - I renamed the suspended QA North to "QA North Edited": it stayed SUSPENDED until I re-activated it.
- **Edit API checks (good), as Sam with `fetch`:**
  - Trust PATCH:
    - a 300-character name → "…200 characters or fewer.";
    - code `A B` → the code message;
    - phone `call me` → the phone message;
    - `status: bogus` → "The status must be active or suspended.";
    - an empty body → "Nothing to change.".
  - Hospital PATCH:
    - time zone `Mars/Olympus` → "Choose a time zone from the list…";
    - a blank name → "Enter the hospital's name.";
    - Trust A's Hospital A1 under QA Trust Five's address → 404 "Hospital not found." (no cross-trust edit).
- **Minor, logo address wording vs rule:**
  - The message says "enter a web address starting with https://", but `http://` addresses are accepted.
  - I saved `http://x.example/a.png` as QA Central's logo through the hospital PATCH and got 200. The cause is `apps/api/internal/settings/settings.go:244`, which allows `http` or `https`.
  - This was the same in round 1 and applies to create too. Either the rule or the text should change.
  - I cleared the logo afterwards.
- **Cosmetic, dev sign-in page only:**
  - When you click a person far down the long development sign-in list, the "Your trust or hospital is suspended…" refusal appears at the top of the page, about 1,700px above, and the page doesn't scroll to it. It looks as if nothing happened.
  - The development page stands in for Cognito (S12), so this may not matter.
- **Harmless data note:** turning a trust default back Off stores an explicit `false` (Trust A now has `{"instrument.eq5d5l": false}`) rather than removing the key. It behaves the same, and the next seed clears it.

## Console and network errors

- **Console:** none, in either tab, from when tracking started to the end.
- **Network:** no 5xx. The only 4xx were the ones the cases expected:
  - 400 and 409 on invalid or duplicate input;
  - 403 on Ada's admin calls.
- Admin GETs are sent twice (React development double render), as in round 1. This is harmless.

## Data left behind

- **New trusts:**
  - **QA Trust Five Renamed** (QA5R; hospitals QA Central and QA North Edited; phone `0100 888 8888`);
  - **QA Trust Six** (QA6; QA Six General).
- **New people:** Zoe Admin Five, Zoe Clinician Five and Zoe Admin Six.
- **Episode:** a new Hospital A1 episode "MT S10 r2 R4 hip" with one answer.
- **Draft questionnaire:** "E2E test case 1310 075911 eeb5"'s "About you" description is now "MT S10 r2 R5 description".
- **Demo data put back:**
  - Trust A's settings are back to Poised's defaults (retention, phone and default HQ ticked back to the default and saved).
  - Hospital A2's own phone and text are removed.
  - Trust A's opt-ins all follow the trust again.
  - All demo trusts and hospitals are active.

## GIFs

`S10-r2-T02-T20-T21.gif` (main happy path: creating a trust, adding hospitals, editing trust and hospital details) was exported by Claude in Chrome to Chrome's download folder (normally `~/Downloads`). There are no failures, so there are no failure GIFs.

## Hand checks for Rahul

None. No case needed a browser alert or confirm dialog: the suspend confirmations are in-app dialogs.
