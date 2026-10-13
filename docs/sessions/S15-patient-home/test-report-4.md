# S15 test report 4 (retest, round 4)

**Summary: 4 passed, 0 failed, 0 blocked.** The cases run were T-24 (new), plus T-01, T-02 and one sign-in case (T-04), to confirm nothing else changed. This was the scope the conductor gave for round 4. T-03 and T-05 to T-23 and R-1 to R-5 were not re-run: the only code change since test report 3 is "← Your questionnaires" waiting for the open section's save. R-6 is Rahul's hand check and was skipped. Verification 1's finding 1 (an unjoined twin signing in to the other twin's account) is an accepted risk and is not reported.

- **Environment:**
  - HEAD `08a3be0`. The working tree is not clean: S15's uncommitted build plus the planning-doc changes.
  - 13 Oct 2026, 20:06 to 20:15 BST.
  - `make dev` was started by the conductor (API :8080, web :3000). `/api/health` was ok.
- **Browser:** Claude in Chrome.
  - Staff pages ran at desktop size: Cara on `localhost:3000`, Bea on `bea.localhost:3000`.
  - Patient pages ran in a 390×844 iframe ("phone") on `tester.localhost:3000`.
- **Test data:** all made up, initials `R4`.
  - **Sam HomeR4:** 17/05/1980, mobile 07700 900561.
    - At Hospital A1 on "Medicines R4", procedure Knee replacement. Created in the New episode UI.
    - At Hospital B1 on "E2E J7c B 1310 200348 3904", procedure Hip replacement. Created (API) with the same `POST …/episodes` call the page makes, from Bea's tab.
  - **Tia SaveR4:** 09/09/1990, save.r4@example.org, at Hospital A1 on "Medicines R4", which has a free-text question. Created (API) from Cara's tab.
- **Pressed Reset sign-in limits** on `/qa/messages` at the start.
- **How checks were made:**
  - Captured messages were read with `GET /api/qa/messages`, the data behind `/qa/messages`.
  - For T-24, a small logger was added in the tester's own phone tab. It wraps the iframe's `fetch` and records `input` and `click` events with timestamps, to prove the tap really came within half a second of typing. No app code was touched.
  - The stored answers were confirmed with read-only selects on `episode_answers`.

## Results

| Case | Result | Note |
|---|---|---|
| **T-01** One home lists both trusts | Pass | A1's invite → Continue ("Hospital A1 needs your date of birth") → `17 5 1980` → the questionnaire → **← Your questionnaires** gives `/home` with "Hello Sam". Two cards, newest first: Hospital B1 (Procedure: Hip replacement), then Hospital A1 (Procedure: Knee replacement). Each shows its questionnaire's name, "For Sam HomeR4", **To do** and **Start**. Mobile shows `•••561`, Email "Not given". There is no date of birth or full mobile in the page text, and no sideways scroll (scrollWidth equals clientWidth at 386). |
| **T-02** Open from the home, either trust, and come back | Pass | B1: **Start** opens it, the bar shows "Hospital B1". Answered "No" to "Do you smoke?", waited for **Saved**, then **← Your questionnaires**: the B1 card is **In progress** with **Continue**. Reopened, and "No" is still chosen. A1: chose "Yes" for blood thinners, the same result. Reopened, and "Yes" is still chosen. |
| **T-04** Returning patient signs in by text | Pass | Sign out → "You've been signed out" → **Get a sign-in link** → `07700 900561` → Enter gives "Check your text messages". The captured text "Poised: your sign-in link. It works once, within 15 minutes: …/s/…" → "Sign in to Poised" → Continue → "Confirm it's you" → `17 5 1980` → home with both trusts' cards, both now **In progress**. |
| **T-24** Leaving straight after typing keeps the answer | Pass | **Step 2:** typed `Warfarin 5mg` and tapped **← Your questionnaires** 325 ms after the last keystroke. The logger shows no save before the tap. The tap sent `PUT …/answers/…` (200), and only then `GET /api/p/home`. The home showed **In progress** at once, with no reload. Reopened, and the box says "Warfarin 5mg". **Step 3:** Cara pressed **Resend invite** ("Invite sent again by email."). The patient typed `Warfarin 3mg` and tapped **← Your questionnaires** 334 ms later. The PUT got 401, and the page stayed on `/home/episodes/<id>` (it did not go home). It showed "You've been signed out" with the orange **Not saved** box ("Your latest change in Your medicines wasn't saved. When you open the link again, please check your answers in that section."), the security text and **Get a sign-in link** (which opens `/signin`). The database still holds "Warfarin 5mg". After the new invite, the box shows "Warfarin 5mg", as the message warned. GIF recorded. |

## Failures

None.

## Found while exploring

1. **Minor, not part of this change: a save that fails because of the network shows the raw "TypeError: Failed to fetch" to the patient.**
   - **How it was tested:** a simulated network failure. The tester's logger made `fetch` throw for PUTs.
   - **What happened:**
     - Tapping **← Your questionnaires** correctly stayed on the section, with the red **Not saved** status.
     - The red alert above the section reads "TypeError: Failed to fetch", which is a technical message on a patient page.
     - The same raw alert appears when the ordinary autosave fails after typing, so it is S14's save-error display, not new in this round.
   - **Suggestion:** use friendlier wording, e.g. "We couldn't save your latest change. Check your connection and try again."
2. **Good: a failed save is retried on leaving.**
   - With the network back, tapping **← Your questionnaires** re-sent the unsaved change (PUT 200) before going home.
   - The database then held "Warfarin 4mg daily".
3. **Cosmetic: a double tap on "← Your questionnaires" right after typing sends the save twice.**
   - Both PUTs got 200, and the revision went from 2 to 4.
   - The answer stored was right ("Warfarin 2mg"), with no error or 409. This is harmless.
4. **Leaving with no change sends no save,** only `GET /api/p/home`. As expected.
5. **"Your medicines" shows COMPLETE before anything is answered.** Neither question in "Medicines R4" is required, so this is S14 behaviour and not new.
6. **The home makes two `GET /api/p/home` calls on each visit.** This is probably React StrictMode in dev, and harmless.

## Console and network errors

- **Console:** none in the phone or Cara's tab (`error|Error|warn`). Note that console tracking started part-way through the run.
- **Network:** the only 4xx was T-24 step 3's expected 401 on the PUT after **Resend invite**.

## GIFs

- `~/Downloads/S15-T24-leave-saves-r4.gif`: T-24. Typing then tapping at once saves and goes home; after Resend invite, the same tap shows "You've been signed out" with **Not saved**.

## Hand checks for Rahul

- **R-6:** suspending Trust B (`/admin`). Skipped, as directed.
