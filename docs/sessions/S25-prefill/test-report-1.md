# S25 test report 1: pre-fill and the patient's confirmation screens

**Summary: 18 passed, 1 failed, 0 blocked** (14 cases + 5 regression cases), plus 3 findings from exploring.

- **Tester:** manual-test agent, automatic run (`/manual-test S25 auto`).
- **Environment:** `fe0d30c` with the S25 build uncommitted in the working tree (git status not clean: the S25 changes). 23 Oct 2026, 12:07–12:30 BST. http://localhost:3000 (API :8080). Chrome ("rahul chrome" profile).
- **Phone size:** `resize_window` didn't change the viewport, so patient screens ran in a same-origin **390 × 780 iframe** in a separate tab, as in earlier sessions. Staff ran in a desktop tab in the same browser; the staff and patient cookies are separate.
- **Test data (all made up):**
  - Patient **Sam S25 MT1207 Round1**, `s25.mt1207a@example.test`, 10 episodes at Hospital A1.
  - Trust B patient **Sam S25 Cross 1207** and a Trust A patient with the same details, both `s25.cross1207@example.test`.
  - Questionnaire **S25 Short 1207** (Trust A).
- **How I worked:**
  - Invite links came from `GET /api/qa/messages?to=…` (curl), the same data as /qa/messages.
  - For the setup recipes and some reads, I clicked, typed and read the patient iframe through page JavaScript. The cases themselves were driven by real clicks.
  - The R-03 reminder targets were confirmed with one read-only query on `jobs`.
  - The R-A episode used **Demo pre-op HQ v5**: earlier runs have published new versions, and v1 is no longer offered.

## Results

| Case | Result | Note |
|---|---|---|
| R-A / R-B (setup) | Done | Reviewed first episode; the snoring correction was Yes. |
| T-01 Pre-filled sections wait to be checked | Pass | Both sections show **Please check** / **Check**; "0 of 2 sections complete"; Send disabled; no sideways scroll (scrollWidth equals clientWidth, 388). |
| T-02 Card counts and date | Pass | "We've filled in 6 answers from your questionnaire on 23 Oct 2026. Please check them." Both buttons are shown, with no questions under them. |
| T-03 Everything is still correct jumps to the first unfilled question | **Fail (Minor)** | The values and markers are right. At phone width the page scrolls past the pregnancy question: its text and the "Yes" option are off the top of the screen. |
| T-04 Changing an answer removes its marker; hidden answers aren't lost | Pass | Sickness → No removes its marker. Problem → No hides "What happened". Back to Yes brings back `I was sick all night` and both markers. Allergy row with its marker; medicines empty. Section **Complete**. |
| T-05 Leaving and coming back fills nothing in again | Pass | After reload the section is still Complete; Review shows no card; sickness No and pregnancy No are kept. |
| T-06 Check each answer walks the section | Pass | Card says 3 answers. Page 1: details box with no marker, smoking and alcohol with markers. Page 2: height 170 with a marker, weight empty, stairs empty. After filling them: Complete, 2 of 2, Send enabled. |
| T-07 Card shows once; an unanswered card keeps the section waiting | Pass | Leaving with **All sections** keeps Please check. **Everything is still correct** goes to page 2 with the weight question focused. Next open has no card. |
| T-08 Earlier answers come from the newest sent HQ | Pass | After sending T-06's HQ, the next follow-up has sickness **No** (the T-04 change), not Yes. The card's date is 23 Oct 2026; R-A was the same day, so the date alone can't tell the two apart. |
| T-09 Everything is still correct with nothing left moves on | Pass | "We've filled in 1 answer … Please check it." It goes back to the list: Complete, 1 of 1, Send enabled. After reload, Review shows **I used to smoke** with its marker. |
| T-10 Filling in for the patient | Pass | Both sections Please check. Card: "…from Sam's questionnaire on 23 Oct 2026. Please check them with Sam." It goes to page 2 at weight, and the marker reads "From Sam's answers on 23 Oct 2026". Opened as the patient afterwards: About you has no card (it shows In progress / Continue); Your health is still Please check. The card said **4** answers rather than the plan's 3, which is correct: T-08 had sent a follow-up HQ with the stairs question answered (Yes), and it was filled by key. |
| T-11 Another trust's answers are never used | Pass | Trust B patient filled in and sent the same answers as R-A. The Trust A patient with the same details and email gets **Start** on both sections, no card, nothing checked or filled, no markers. |
| T-12 Clinician-only answers are never pre-filled | Pass | Sent T-10's HQ as Cara. Validate → Nurse checks: Blood pressure is empty. |
| T-13 Home doesn't call an untouched pre-filled HQ "started" | Pass | Untouched: **To do / Start**. After changing alcohol in one section: **In progress / Continue**. |
| T-14 Deep links and the back button | Pass | Reloading on a card goes back to the list with Please check. Home → Back → Forward → Back: no errors, list as before. |
| R-01 (S14) New patient with no earlier HQ | Pass | R-A's first HQ: Start on both sections, no card; filled in and sent ("Thank you, Sam"). |
| R-02 (S02a) Two tabs | Pass | Tab 2 saved alcohol 5. Tab 1's edit gave "Your answers weren't saved … Reload"; after Reload it shows 5. |
| R-03 (S16) Reminders | Pass | Jobs run as of Now + 2 days (39 jobs, 17 reminders). The untouched pre-filled follow-up got a reminder; episodes where the patient changed an answer didn't (their next job is the day-7 one). Checked against the `jobs` table. |
| R-04 (S20/S22) Cancelled episode | Pass | Cancelled one follow-up (reason Patient withdrew). The patient's home shows **Cancelled by the hospital … nothing to fill in**. The link shows the general "This link doesn't work", as S20's plan says; no form and no pre-fill. |
| R-05 (S23/S24) Editor pre-fill labels | Pass | "Do you have diabetes?" lists **Demo pre-op HQ · by same wording** (plus E2E Audit HQs). "Do you smoke?" lists **Demo pre-op HQ · by link** (plus S24 Lib HQs and my S25 Short 1207). |

## Failures

### F1 (Minor, visible to the patient). At phone width, "Everything is still correct" scrolls past the question it focuses (T-03)

**Signed in as:** patient Sam S25 MT1207 Round1, via the invite link of a Demo follow-up HQ episode. URL: `/home/episodes/<id>`, in a 390 × 780 iframe.

**Steps:**
1. Do R-A, then R-B, and open R-B's invite at phone size.
2. On *Your health*, tap **Check**, then **Everything is still correct**.

**Expected:** the page shows "Is there any chance you could be pregnant?" with its options.

**Actual:**
- The right question has focus (the pregnancy question's "Yes" radio), but the page is scrolled 135 px too far.
- The question's text and its "Yes" option are above the top of the screen. The patient first sees the "No / I'm not sure" options with no question above them, then the sickness question.
- Measured: `scrollY` 806 in a 778 px viewport; the question's top is at −135 and the focused input at −40.
- It happened every time, on two different episodes. A GIF shows it.

**Comparison:**
- In a 1000 px wide iframe that needs scrolling, the question lands exactly at the top (0).
- In a desktop top-level tab with extra space added above the content, it lands at 157 (visible).
- So this is about phone width, not about the iframe. It's still worth a hand check on a real phone or the device toolbar.

On T-07's About you, where the target (weight) is on another page, focus and position are fine.

No console errors and no failed requests.

## Found while exploring

1. **(Minor) A section finished with "Everything is still correct" doesn't count as started.**
   - **The case:** S25 Short 1207. The only section was all pre-filled; the patient tapped **Everything is still correct**. The list then says Complete, "1 of 1 sections complete", and **Send my answers** is enabled.
   - **Home:** the patient's home still says **To do** with **Start**.
   - **Reminders:** running the jobs as of Now + 2 days sent this patient a **reminder** ("please fill in your health questionnaire"); that episode's next job is the day-5 reminder.
   - **Why:** "started" means a saved patient revision above 0, and confirming with the button saves nothing.
   - **The same gap elsewhere:** after Everything is still correct on *About you* (T-07, T-10), the HQ list shows **In progress / Continue** while the home shows **To do / Start**, and reminders still go.
   - **Effect:** a patient who has confirmed everything is told they haven't started, and could be marked "Not responding" on day 7. Possibly intended ("pre-filled, not yet saved"), but the home and the list disagree.
2. **(Cosmetic, seed) The Demo follow-up HQ's details box is headed "profile".**
   - The patient sees the heading **profile** in lowercase: the seed's `{"type":"profile","name":"profile"}` has no `title`. The editor's default is "Your Profile".
   - The box also shows "Sex **female**" in lowercase. That comes from the profile template's `{patientSex}`, so it may predate S25, but this seed is the first patient flow that shows the box.
   - The editor shows "profile" for this element too.
3. **(Observation) Card buttons and other routes behave as expected.**
   - **Double-clicking "Everything is still correct"** sends one POST and lands once on the right question.
   - **Probing `POST /api/p/episodes/{eid}/prefill/{cid}` from the patient's session:**
     - a repeat of the same choice: 204;
     - a different choice after the first: 204 (first kept);
     - `"banana"`: 400;
     - the clinician's set: 404;
     - a random episode: 404;
     - another patient's episode: 404;
     - a sent form: 409.
   - **Two tabs:** a tab opened before the card was answered elsewhere still shows the card (stale), which is harmless.
   - **Editor:** "Do you have diabetes?" in the Demo follow-up HQ says "Pre-fills from **100** other HQs", because many E2E HQs share the wording. That's S23's label, not S25. I didn't check whether 100 is a cap.

## Console and network errors

- No console errors or warnings captured in the staff tab or in the patient's top-level tab.
- The iframe's own console isn't captured by the tool.
- The only non-2xx responses seen were the probes above and the tool's known spurious 503 on the prefill POST, which succeeded.

## GIFs

- `~/Downloads/S25-happy-path.gif`: T-01 to T-06 at phone size (date of birth, both sections waiting, card, Everything is still correct, marker changes, Check each answer, Complete).
- `~/Downloads/S25-T03.gif`: F1, the list, then the card, then **Everything is still correct** landing past the pregnancy question.

## Hand checks for Rahul

- F1 on a real phone-size window (Chrome device toolbar, 390 × 844): after **Everything is still correct** on *Your health*, is the pregnancy question's text visible?
