# S14 test report 4: patient invites and magic-link sign-in (retest after verification 2)

**Summary: 4 passed, 1 failed, 0 blocked.** As the top of test-plan.md asks, I ran the new T-21 and T-22, plus a quick T-06 step 6 and T-07. The conductor also asked me to try typing then pressing **Back**, and typing then **Send**. I've added those as X-1 and X-2.
- **X-1 failed (Minor).** On the patient's form, an answer typed or tapped less than about 0.6 s before the browser's **Back** (or a reload) is lost, and the patient isn't told.
- **The fixed paths work.** Sign out, the on-behalf **← Episode** and the staff browser Back all keep the answer just typed.

**Environment:**
- HEAD `14981f1`. S14's changes are uncommitted in the working tree, including round 4's: flushing on Sign out and **← Episode**, sending a waiting save on unmount, and `textUpdateMode: "onTyping"`. The tree also has planning-doc changes that aren't part of S14.
- 13 Oct 2026, 14:37–14:47 BST.
- The app at http://localhost:3000, served by the conductor's `make dev` (MESSAGING=capture).
- Claude in Chrome, automatic mode.

**Phone size:** as in reports 1–3, every patient step ran in a **390 × 844 iframe**, on `http://tester.localhost:3000` (a fresh browser with no staff session). "Reload" means reloading that iframe's page. "Back" means the browser tab's real Back button (the tool's `navigate back`).

**Round 4 data:** every surname ends in `R4`. The mobiles were M4 (`+447700900404`) and M5 (`+447700900405`). The emails were `s14.r4.b` and `s14.r4.h@example.org`.

| Episode | Contact | HQ | Used for |
| --- | --- | --- | --- |
| Sam Typing R4 | M5 | Medicines R4 | T-21, T-06 step 6, X-1 |
| Sam Leaving R4 | `s14.r4.h` | Medicines R4 | T-22, the staff browser Back |
| Sam SendTyped R4 | M4 | Medicines R4 | X-2 (typing, then Send) |
| Sam Phone R4 | `s14.r4.b` | Trust A demo HQ v1 | T-07 |

**Where I used tools other than the UI:**
- **Medicines R4** (T-21 step 1): I created and published it as Alex Author with `fetch`, calling the same API the editor uses. Its content is the same as `e2e/data/medicines-hq.json`: one Patient Question Set, "Your medicines", with the text question "What medicines do you take?" and the Yes/No question "Do you take blood thinners?". Neither is required. Authoring wasn't under retest.
- **Episodes:** I created *Sam Typing R4* through the New episode form. It said "Invite sent by text message." I created the other three with `fetch` to the same create-episode API, signed in as Cara.
- **Links:** I read them with `curl /api/qa/messages?to=…`.
- **Database:** I ran read-only `select`s on `episode_answers` and `episodes` to confirm what was saved.

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-21 Sign out straight after typing keeps the answer | Pass | Link → Continue → `17 5 1980` → Your medicines. I typed `Warfarin 5mg` and tapped **Sign out** in the same action batch, with no other tap. The page showed "You've been signed out — You've signed out. To come back, open the link…" with **no** Not saved box. The DB had `{"q_medicines":"Warfarin 5mg"}`, rev 1. Step 4: the link again → the section showed `Warfarin 5mg`. Step 5: I tapped **Yes** and then Sign out at once. The DB had `q_thinners: o_yes`, rev 2. The link again → the section showed **Yes** and the text. |
| T-22 Leaving the on-behalf form straight after typing keeps the answer | Pass | As Cara on Sam Leaving R4: **Fill in for the patient** → Review → I typed `Aspirin 75mg` and clicked **← Episode** at once. I landed on the episode. Under Your medicines it said "Waiting for the patient" and "**Patient's answers entered by Cara Clinician**". **Fill in for the patient** again → the section showed `Aspirin 75mg`. |
| T-06 step 6 Sign out, then reload (quick) | Pass | On Typing R4: link → right date → **Sign out** gave "You've been signed out" ("You've signed out. To come back…"). After a reload it still said "You've been signed out" ("For your security, you've been signed out…"), with no questionnaire. |
| T-07 Staff fill in on behalf (quick) | Pass | Sam Phone R4, on Trust A demo HQ v1:<br>• `/patient-form` had the blue box "You're filling in Sam Phone R4's health questionnaire for them…" and only About you and Your health.<br>• I answered both sections, then **Send the answers** → confirm. It showed "Sent for Sam Phone R4 — The answers are with the team for review."<br>• On the episode: **Ready for review**; "Sent by Cara Clinician on the patient's behalf on 13/10/2026, 14:44."; "Patient's answers entered by Cara Clinician" under both sets; the note "HQ completed on the patient's behalf"; the invite card was gone.<br>• Validate About you had "The patient's answers here were entered by Cara Clinician on the patient's behalf."<br>• The POA Summary had "Patient (entered by Cara Clinician) on 13/10/2026, 14:44". |
| X-1 (asked by the conductor) Typing then **Back** on the patient's form | **Fail (Minor)** | An answer typed or tapped less than about 0.6 s before Back is lost, and the patient isn't told. See below. |
| X-2 (asked by the conductor) Typing then **Send** | Pass | On Sam SendTyped R4, at phone size: I typed `Metformin 500mg` and tapped **Finish this section** at once. Then **Send my answers** → confirm gave "Thank you, Sam". The DB had `{"q_medicines":"Metformin 500mg"}`, and the episode was `ready_for_review`. There's no Send button on the section page, so a patient always leaves the section through Finish or All sections first, and both wait for the save. |

## Failures

### X-1 Back (or a reload) within about 0.6 s of typing drops the answer, without a word (Minor)

**Where:** the patient's questionnaire, `/home/episodes/<id>`, at 390 × 844. I was the patient of Sam Typing R4, on `tester.localhost`.

**Steps:**
1. Open the invite link → Continue → `17 5 1980` → **Review** on Your medicines. The answers already saved are `Warfarin 5mg, Aspirin 75mg` and **Yes**.
2. Tap the end of the text box and type `, Omeprazole`. Then tap **No** for blood thinners.
3. Straight away, press the browser's **Back** button.
4. Open **Review** on Your medicines again.

**Expected:** the text and the **No** are kept, as with Sign out (T-21) and the staff's **← Episode** (T-22). The code's own comment says that leaving a set "any other way (Back, a link) sends a save still waiting".

**Actual:**
- Step 3 lands on the questionnaire list, with no message.
- Step 4 shows the **old** answers: `Warfarin 5mg, Aspirin 75mg` and **Yes**.
- `episode_answers` was unchanged (rev 3, 13:41:26).
- The network log had **no PUT** after the Back, only the list's GETs.
- I reproduced it three times: text only (`, Ibuprofen`), text plus a radio change (`, Omeprazole` + No), and text then a reload of the page (`, Statin`).

**Control (no failure):**
- If I type and then wait about 2 s, the save goes (rev 3: `, Aspirin 75mg` was saved while the status still said "Saved").
- On the **staff** on-behalf form, typing `, Clopidogrel` and then pressing the browser's Back at once **did** save it (rev 2, `updated_by` set).

**Why (from reading, not editing, `apps/web/src/features/patient/PatientHQ.tsx`):**
- Opening a section isn't a new history entry: the URL and `history.state` stay the same, and the section is page state. So on the patient's pages, **Back leaves the page altogether**, as T-06 step 4 expects. The same goes for a reload or closing the tab.
- The new "send a waiting save" runs in React's unmount cleanup, and that never runs when the page unloads. The 600 ms timer (`SAVE_DELAY_MS`) dies with the page.
- No `pagehide`, `visibilitychange` or `beforeunload` handler sends the save, for example with `fetch(..., {keepalive: true})`.
- On the staff side, Back from `/patient-form` is an in-app route change, so the unmount cleanup runs. That's why it works there.

**Severity, honestly:**
- **Minor.** Only input from the last ~0.6 s before Back, reload or close is lost. Anything older than that is already saved, because text now saves as you type.
- But the loss is silent. A *changed* answer (Yes → No above) quietly goes back to the old value, which is the kind of thing verification 2 wanted to avoid. "Tap an answer, then swipe back" is a natural move on a phone.
- It isn't the T-21/T-22 bug (those are fixed). It's the same gap through the browser's own navigation.

**Recording:** `~/Downloads/S14-X1-back-drops-typed-answer.gif`.

## Found while exploring

1. **A section that has no required questions shows COMPLETE and "1 of 1 sections complete" before anything is answered** (Medicines R4). **Send my answers** is enabled with nothing typed. That follows from "nothing is required", and the plan already puts "Send with no answers" out of scope. I'm noting it only because a patient might read COMPLETE as "done".
2. **While typing, the status says "Saved", not "Saving…",** until the 600 ms timer fires. The plan says it "may briefly say Saving…", so this isn't a failure. It does mean the patient has no sign that the latest keystrokes are still waiting, which matters for X-1.
3. **Tooling notes, not app bugs** (as in reports 1–3):
   - The first click after a page load was dropped twice: **Fill in for the patient**, and the **Back to the episode** link clicked through its accessibility ref. Clicking by position worked.
   - The Health questionnaire dropdown on New episode isn't searchable by typing. I picked the option by scrolling to it. That's earlier sessions' behaviour, not S14's.

## Console and network errors

- Console: no errors or warnings on the staff tab or the phone tab while I was tracking. The phone pages ran inside an iframe, whose console the extension may not capture.
- Network: nothing unexpected. Every PUT was 200. In X-1 there was simply no PUT.

## Recordings

- `~/Downloads/S14-T21-signout-keeps-typed-answer.gif`: the main happy path (T-21 at phone size). The link, the date of birth, typing `Warfarin 5mg` → Sign out, the link again showing it, then Yes → Sign out → kept.
- `~/Downloads/S14-T22-episode-link-keeps-typed-answer.gif`: T-22 (typing → **← Episode** → "entered by Cara Clinician").
- `~/Downloads/S14-X1-back-drops-typed-answer.gif`: X-1 (typing and tapping No → Back → the section shows the old answers).

## Hand checks for Rahul

- None new. X-1 doesn't need a person: it reproduces with the tab's real Back button.

STATUS: FAILURES 1
