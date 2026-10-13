# S10 test report 1

**Summary:** 18 passed, 1 failed (T-15, which has 2 Minor issues), 0 blocked. Regression R-1 to R-5 all passed.

- **Commit:** `2c3054d`. The working tree isn't clean: the S10 code and the planning docs aren't committed yet, as expected in the automatic run.
- **When:** 13 Oct 2026, from 02:21 BST.
- **URLs:** http://localhost:3000 (Sam Superadmin) and http://tester.localhost:3000 (the second person). The conductor's `make dev` was running against a freshly seeded database.
- **Browser:** Chrome, through Claude in Chrome, at a desktop window size. No phone-size cases this session.
- **How I tested:** through the UI. For long lists of invalid inputs (T-03, T-05 step 5, T-08, T-16) I filled the fields with a small page script that types into the real inputs, then clicked the real Save or Create button and read the message shown. I used the API or the database only to confirm things, and say so in the notes.
- **Tooling note:** the browser tab often ran in the background, which slowed Mantine dialog animations. Sometimes text typed while a dialog was still opening was lost. I retried each of those, and none of them is reported as an app bug.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 | Pass | Sam lands on `/admin` "Trusts" with the purple PLATFORM ADMIN badge. Trust A (TA, 2, Active), Trust B (TB, 1, Active) and the E2E trusts are listed. `/` shows the platform-admin card, and its link goes back to `/admin`. |
| T-02 | Pass | QA Trust One / `qa1` → "Code QA1". The green message, contact details, logo address, Zoe Admin One "INVITED · EXPIRES 20/10/2026" and "No hospitals yet…" all show. Opt-ins: Assessments, PROMs, Files and Observations On, the rest Off. It's in the Trusts list with 0 hospitals. GIF `S10-T02.gif`. |
| T-03 | Pass | Every invalid input gave the expected red message under the form, and the form stayed open. None of these trusts was created. |
| T-04 | Pass | `TA` and `ta` → "Another trust already has the code TA. Choose another." `qa1` → the same for QA1. |
| T-05 | Pass | QA General (two-line address, phone, "Trust's email", Europe/London (default), Active) and QA North (America/Toronto, without "(default)") were added. `qa general` → "This trust already has a hospital called qa general." with the dialog still open. The empty name, empty address, `ring ring`, `poa@` and `ftp://…` inputs were each refused. Opt-in columns show "Follow trust (…)". Step 7: see the deviation note below. |
| T-06 | Pass | Zoe Admin One gets the picker (QA General, QA North, Admin), and the header shows QA Trust One / QA General / ADMIN with Staff and Settings. Her invite to Zoe Clinician One worked. Zoe Clinician One goes straight to QA General Episodes as CLINICIAN, with no Staff or Settings and only QA General in the box. Sam's page then shows Zoe Admin One as Active. |
| T-07 | Pass | Zoe Admin Two's start page shows QA Trust Two (Admin, whole trust) and the "has no hospitals in Poised yet…" note. After Sam added QA Two General, a reload took her straight in. |
| T-08 | Pass | Inviting Zoe Admin Three showed "An invitation was sent…" and listed her as Invited. Cara → "…already had an account, so they were made an admin of QA Trust Two." and Active. Her start page then lists QA Trust Two and Trust A Hospital A1. Zoe Admin Two's email → a red message in the dialog. `nope` → "Enter a valid email address…". Enter submits the form. |
| T-09 | Pass | Ada has no Platform admin link. `/admin`, `/admin/trusts/new` and `/admin/trusts/<QA1 id>` all show "Only Poised's platform admins can open this page" with "Go to your hospitals". Cara is refused the same way. |
| T-10 | Pass | As Ada: 403 `{"error":"Only Poised's platform admins can do this."}`. Signed out: 401 `{"error":"Please sign in."}`. I also checked the write routes as Ada (POST orgs, PATCH org, POST hospitals, POST admins, GET opt-ins): all return 403. |
| T-11 | Pass | Trust A's page shows only trust, contact, hospital, admin and opt-in details. The admin API response for Trust A has no patient or episode fields (checked). Sam on `/o/TA/h/A1/episodes` sees "You don't have access to this hospital". `/api/o/TA/episodes` → "Trust not found.", `/api/o/TA/h/A1/patients` → "Hospital not found.", and `/api/o/TA/settings` → "Trust not found.". |
| T-12 | Pass | The in-app confirm dialog explains the effect. After suspending: "Trust B is suspended. Its staff can't sign in.", a SUSPENDED badge, the red box, and the button changes to Re-activate. Bea's open session then gets "Hospital not found." Signing in again → "Your trust or hospital is suspended…", and she stays on the sign-in page. Max signs in straight to Hospital A2, and `/api/me` lists only Trust A. Re-activate → "Trust B is active again…", Bea signs in, and Max sees both trusts. I checked the Trusts-list "Suspended" status with QA Trust Two (Trust B was already active again), and it showed SUSPENDED. GIF `S10-T12.gif`. |
| T-13 | Pass | Hospital A1 shows SUSPENDED and Re-activate, and Trust A stays Active. Step 3: Cara is no longer A1-only, because T-08 made her an admin of QA Trust Two. She signed in to QA Two General, which is correct. I checked the refusal with **Sue Super** (A1 only) and got the "suspended" message. Ada goes straight to Hospital A2. "All hospitals" (API) has only A2 episodes. The A1 address shows "You don't have access to this hospital". After re-activating, Sue and Cara sign in and Ada sees both hospitals. I also checked the patient link: see "Found while exploring". |
| T-14 | Pass | "PROMs at Hospital A1 is on." appeared, and A2 still says "Follow trust (Off)". With EQ-5D-5L trust On, both hospitals say "Follow trust (On)", and A2 can be set to Off. Everything was kept after a reload. Ada's Settings show A1 PROMs ON and EQ ON, and A2 PROMs OFF and EQ OFF, each "Set by Poised's platform admin". Everything is reverted to follow the trust. (The green message is at the top of the page: see F2.) |
| T-15 | **Fail** | It works, but: **F1:** at a hospital, the "Keep episode data" and "Offer answers…" badges are missing. **F2:** the "saved" message appears ~2,000px above the Save button. Steps 1, 2 and 4–8 otherwise pass. |
| T-16 | Pass | 0 and 101, `0, 99`, `2, 2`, six days, `soon`, the missing {{link}}, {{name}}, `nope`, `logo.png` and `call us` each gave a red message naming the setting. The red message appears next to Save. Nothing was saved (checked with GET settings). The server refuses the same values when sent directly. |
| T-17 | Pass | "None" plus Trust A's published questionnaires. Drafts (e.g. "E2E test case 1310 021635 0cd3") aren't offered. My choice was kept after a reload. |
| T-18 | Pass | Alex and Sue have no Settings link. `…/settings` → "Only the trust's admins can change its settings" (API 403). At B1, Max has Settings with Trust B's hospitals only (B1). At A2 he has no link and the page refuses him. Ada on `/api/o/<Trust B>/settings` → 404 "Trust not found.". |
| T-19 | Pass | The trust page survives a reload and opens from a copied address. Back goes trust page → Trusts. Signed out, a trust address goes to the sign-in page, and the address opens after signing in as Sam. `/admin/nope` → the 404 page. |
| R-1 | Pass | Max's picker shows Trust A (A2, Clinician) and Trust B (B1, Admin). Switching in the header changes the trust and the role. |
| R-2 | Pass | Ada invited `zoe.r2.s10@test.example` to A1 as Clinician. The new person was Invited, signed in to A1 Episodes, and then showed as Active. |
| R-3 | Pass | Ada's "All hospitals" includes A1 and A2 episodes (API: 280 episodes from both hospitals). |
| R-4 | Pass | Cara created an episode (procedure "MT S10 R4 knee"), opened the patient link in a tab, and answered "Do you smoke?". The answer was saved (`q_smokes: o_no`, checked with GET `/api/p/<token>`). |
| R-5 | Pass | Alex changed "Do you smoke?" to "Do you smoke? (MT S10)" in a draft, and it was kept after a reload. (Committing the edit needed a blur: see "Found while exploring".) |

**Deviations from the plan:**
- **T-05 step 7:** I added "QA General" to an **E2E trust** that had no hospitals (E2E Trust 1310 021240 38a5), not to Trust B. It was accepted, which shows names are unique per trust only. I didn't use Trust B because `make seed` doesn't remove hospitals, and a second hospital there would permanently change Bea's and Max's flows (picker, T-12 and T-18 expectations).
- **T-13 step 3:** I used Sue Super, for the reason given in the table.

## Failures in detail

### F1 (Minor): at a hospital, the trust-only settings aren't shown at all
- **Case:** T-15 step 3. Signed in as Ada Admin at http://tester.localhost:3000/o/6ef150fb-…/h/00000000-0000-4000-8000-00000000000a/settings.
- **Steps:**
  1. Settings, then "Settings for" → **Hospital A2** (or A1).
- **Expected (plan, and build-notes line 49):** "Keep episode data" and "Offer answers…" are shown as badges "Set for the whole trust".
- **Actual:** the whole "Whole trust" group is missing. The page goes straight from the blue note to "Features and opt-ins", so a hospital view never shows the retention period or the cross-trust pre-fill.
- **Likely cause:** `apps/web/src/features/settings/SettingsPage.tsx` skips a group when every setting in it is trust-level: `if (hospital && defs.every((d) => d.level === "trust")) return null;`. The "Whole trust" group holds only those two settings, so the `ReadOnly` "Set for the whole trust" badge is never drawn.
- **Console and network:** none.

### F2 (Minor): the success message after Save, or after an opt-in change, appears far above where the user is looking
- **Cases:** T-15 steps 2 and 4, T-17 step 2, and T-14 step 1 on Sam's trust page.
- **Steps (settings):**
  1. As Ada, open Settings (whole trust).
  2. Untick a default and change a value.
  3. Scroll to the bottom and click **Save the trust's settings**.
- **Expected:** "Trust A's settings were saved." is visible.
- **Actual:**
  - The green message is drawn at the top of a ~3,000px page, about 2,100px above the Save button (measured: alert top −2105px, scrollY 2321). The page doesn't scroll to it, so after clicking Save nothing visible changes.
  - Error messages and "Nothing has changed." do appear next to the button, so only the success message is affected.
  - If you then click Save again with nothing changed, the old green "…were saved." stays at the top while "Nothing has changed." shows at the bottom.
  - On Sam's trust page, the same happens with "PROMs at Hospital A1 is on.": it's drawn at the top while the opt-in selects are at the bottom of the page.
- **Console and network:** none.

## Found while exploring

- **Patient link for a suspended hospital (good):**
  - With A1 suspended, an A1 episode's link returns 403 `This questionnaire isn't available at the moment. Please contact the hospital.`. I took the token from the database to check this.
  - In the browser, `/p/<token>` shows that message in a red box. An A2 link still returned 200.
- **Odd inputs on New trust (good):**
  - A 300-character name → "The trust's name must be 200 characters or fewer."
  - Name `   MT S10 🏥 Trust Ünïcode   ` and code `  qa-4  ` → saved as trimmed "MT S10 🏥 Trust Ünïcode" and code "QA-4".
  - Bogus trust addresses (`/admin/trusts/<random uuid>` and `/admin/trusts/not-a-uuid`) → "Trust not found."
  - Double-clicking "Suspend trust" in the confirm dialog suspended the trust once, with no error.
- **Settings API checks (good):** a trust-only key for a hospital → 400 "…is set for the whole trust only." An opt-in from a trust admin → 403. Another trust's hospital id → 400. A missing `hospitalId` → 400.
- **Cosmetic, admin invite dialog:** inviting someone who's already an admin says "Zoe Admin Two already has that access. **Change it in the list instead.**" But the console's admins list has no change control. The wording is carried over from the Staff screen.
- **Cosmetic, start page:** for a trust admin whose trust has no hospitals, the start page is headed "Choose a hospital" although there's nothing to choose. The note under it is right.
- **Minor, header switch from a refused Settings page:** Max on `/o/TA/h/A2/settings` ("Only the trust's admins…") switched the header box to Hospital B1, where he is admin. He landed on B1 **Questionnaires**, not Settings. When Ada switched between hospitals she stayed on Settings (T-15 step 7 passes), so this only happens when coming from the refused page.
- **Usability, long lists:** the "Default health questionnaire" select lists 316 published questionnaires (mostly E2E leftovers) and can't be searched. The HQ select on New episode is the same (older).
- **Older than S10 (S01 editor):** in the questionnaire editor, editing "Question text" updates the canvas and saves only when the field loses focus. While typing, the header already says "All changes saved". Typing a new title and reloading without leaving the field lost the edit, while it said "All changes saved".
- **Plan note for the builder:** T-08 makes Cara an admin of QA Trust Two. That changes T-13 step 3, where Cara is no longer refused, and every later "Cara is A1 only" assumption. `make seed` doesn't undo the new membership.

## Console and network errors

- **Console:** none in either tab while I was watching, from partway through the run to the end.
- **Network:** no 5xx. Every 4xx was one the cases expected:
  - 400 and 409 on invalid or duplicate creates;
  - 403 and 404 on refusals;
  - the 404 on the suspended trust's hospital.
- Each admin GET is sent twice (React development double-render). This is harmless.

## Data left behind

- **Created:** QA Trust One (QA1, hospitals QA General and QA North), QA Trust Two (QA2, QA Two General) and "MT S10 🏥 Trust Ünïcode" (QA-4), plus a "QA General" hospital in E2E Trust 1310 021240 38a5.
- **New people:** Zoe Admin One, Zoe Admin Two and Zoe Admin Three; Zoe Clinician One; Zoe R2 Clinician (Hospital A1).
- **Cara Clinician** is now also an admin of QA Trust Two.
- **Episode and questionnaire:** a new A1 episode for "Sam A1 1210 224401 0993", with one answer. The draft "E2E test case 1310 021635 0cd3" question title now ends "(MT S10)".
- **Re-activated:** Trust B, Hospital A1 and QA Trust Two are active again. Trust A's opt-ins are back to following the trust.
- **Trust A's settings are still changed:**
  - phone `0100 222 3333`;
  - retention 10 years;
  - default HQ "E2E explore 1210 145942 080c".

  I tried to reset them to Poised's defaults, but the action was refused, so they're left for Rahul or the next seed of settings. Hospital A2's own "What happens next" text ("MT S10 A2: we will call you within 3 days.") is also still set.

## GIFs

`S10-T02.gif` (main happy path: creating a trust) and `S10-T12.gif` (suspending a trust) were exported by Claude in Chrome to Chrome's download folder (normally `~/Downloads`). The failures are layout issues, described in detail above with measurements, so they have no GIF.

## Hand checks for Rahul

None. No case needed a browser alert or confirm dialog: the suspend confirmations are in-app dialogs.
