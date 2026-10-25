# S28 test report 1

**Summary: 18 passed, 0 failed, 0 blocked** (13 cases T-01 to T-13 and 5 regression cases R-01 to R-05).

**Environment**
- Commit `41805f5` on master. The working tree is **not clean**: it holds S28's uncommitted changes (39 changed or new paths, including migration `00028_prefill_once.sql`).
- Date: Sun 25 Oct 2026, 12:22 to 12:58 GMT.
- URL: http://localhost:3000. `make dev` was already running (API :8080 healthy, web :3000 200).
- Browser: Chrome ("rahul chrome" profile) through Claude in Chrome. Staff pages used a desktop window (1456 × 829, later 1456 × 717).
- **Phone size:** the Chrome window wouldn't resize below full screen. So every patient page ran inside a 390 × 780 same-origin iframe (a page on localhost:3000 holding the app at phone width; it shares the real cookies). I checked for horizontal overflow there with `scrollWidth`; it was 390 every time.
- Users were switched with **Sign out** and the stub login page. Invite links were read from the captured messages (`/api/qa/messages`, the same data the /qa/messages page shows).
- Test data: patients "Sam S28 T01 / T04 / T04b / T04x / T06 / T09 / T10 / T11 / T12 / R01 / R02" and "sam s28 t04", emails `s28.*.r1@example.test`, DOB 17/05/1980 for the people with a Trust B history.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Fill in for a patient who never opened Trust A's link | Pass | Banner "You're filling in Sam S28 T01's health questionnaire for them…". Both sets **Please check**. Your health card: "We've filled in 4 answers from Sam's questionnaire at Hospital B1 on 25 Oct 2026. Please check them with Sam.", with all three buttons. Check each answer showed diabetes No and snoring Yes, each with a marker. Start fresh on About you gave an empty set with no card. After reload: About you **Start**, Your health kept its answers (In progress). |
| T-02 Audit logs | Pass | Trust A: exactly one "Patient's record joined their sign-in account" row, by Cara Clinician, plus her fill-in rows. Trust B: one "read to pre-fill" row, Who **Staff at Trust A** with no email; the patient's own rows are "Patient (their link)"; no "Cara" anywhere in the T01 results. Who box has **Staff at Trust A** (no brackets); that filter shows only Staff at Trust A rows. The CSV has Who "Staff at Trust A", an empty Email column and 0 matches for "cara". |
| T-03 No contact copied | Pass | Trust A patient page: Mobile —, email only. Bea's Trust B episode still shows +447700900101 and the same email. |
| T-04 One detail different | Pass | T04x (other surname) and T04b (no email, other mobile): both sets **Start**, no card. No join row in Trust A and no read row in Trust B for either. |
| T-05 Reopening reads Trust B no more | Pass | Opened Fill in again and reloaded. Trust B still has exactly one read row for T01 (12:29). |
| T-06 First open is the only pre-fill (same trust) | Pass | Follow-up opened first (Start/Start), then the pre-op was sent by the patient. Reopened by Cara: still Start/Start. Opened by the patient on their invite (phone size): Start/Start. |
| T-07 Start fresh line on the POA Summary | Pass | "Answers were pre-filled here from the patient's earlier questionnaires. Cara Clinician chose “Start fresh” with the patient on 25 Oct 2026, so the pre-filled answers were cleared and every answer here was given afresh." It's on both tabs and identical to Validate's About you banner. |
| T-08 Patient reported BMI | Pass | 28.4 on both tabs before validating. After validating About you with weight 90: 31.1 on both tabs. T-06's unanswered follow-up: "—". An extra check on the patient-sent pre-op: 28.4. |
| T-09 Archived View has no review promise | Pass | Before archiving: "…A member of the team will review them." After Sue archived it, both the invite and Your questionnaires → View read "Thank you, Sam / Your answers have been sent to the hospital." with no review sentence. Width 390. |
| T-10 Cancelled after sent | Pass | Sent and cancelled: "The hospital has cancelled this. Please contact them if you have questions." Unsent and cancelled: "The hospital has cancelled this, so there's nothing to fill in. Please contact them if you have any questions." Both are tagged **Cancelled by the hospital**. Width 390. |
| T-11 NOT RESPONDING hidden once cancelled | Pass | Jobs run as of 02/11/2026 12:50 (25 marked not responding). Red **Not responding** badge shown; gone after Cancel and after reload; still gone after Sue archived it. |
| T-12 "Everything is still correct" title below the top bar | Pass | It jumped to the pregnancy question. The title's top was 66 px against the top bar's bottom at 56 px, so it was fully visible. |
| T-13 Failed reload after Start fresh | Pass | Done with my tools: I patched `window.fetch` in the page so that only the GET after the Start fresh POST failed. The page showed "The answers in this section were cleared / The questionnaire couldn't be loaded again just now. Please try again." with **Reload**, and Reload opened the empty set. (See the cosmetic note below.) |
| R-01 Patient opens the Trust A invite themselves | Pass | Your health card: "…from your questionnaire at Hospital B1 on 25 Oct 2026…", with Start fresh. Trust B's row is by "Patient (their link)". |
| R-02 Same-trust follow-up pre-filled at first open (phone) | Pass | Both sets Please check. **Everything is still correct** jumped to the pregnancy question. |
| R-03 Trust A Who filter and CSV | Pass | Entries are "Name (email)". The Cara filter gives only Cara rows (500 shown). The CSV downloaded (8,864 rows, all Cara). |
| R-04 Clickable POA answer and Back | Pass | Clicked "Do you snore loudly?" in the "Pre-filled on Page 1" box on Patient answers. Validate opened at `?q=snoring` with the question highlighted. Browser Back returned to `?tab=patient` at the same scrollY (456). Leaving straight afterwards by the Episode link: the next page started at scrollY 0. |
| R-05 Cancelled episode is read-only | Pass | The staff page shows "This episode is cancelled", Views only, and locked tasks. The patient's home says Cancelled by the hospital, and the old link says "This link doesn't work". |

## Failures in detail

None.

## Found while exploring

- **The join ignores case and spacing (works).** I made a Trust A record "sam" / "s28  t04" (lower case, double space) with the email in capitals and no mobile. Fill in for the patient joined it to the T04 Trust B account: both sets Please check. The name is stored and shown as "sam s28 t04".
- **Cosmetic, T-13's error state:** above the friendly "The answers in this section were cleared" card, a red banner shows the raw browser error text "Failed to fetch". It reads as a technical message next to the friendly one. Seen at `/o/…/episodes/339fb1c6-…/patient-form` as Cara, with the simulated network failure.
- **Observation, audit wording:** in Trust A's log, Start fresh and Check each answer both appear as "Checked pre-filled answers with the patient". T01 has two such rows at 12:30, one for each choice. The log can't tell which choice was made. This probably predates S28; I'm noting it for Rahul.
- **Observation, out of scope per the plan:** Trust B's **Who** box still lists "Cara Clinician (cara.clinician@hospital-a.example)". Filtering by her shows only 4 rows, all from 24 Oct (S27 testing, before this session). Every row made today names only "Staff at Trust A".
- **Observation:** each staff page view writes two audit rows (for example, "Opened an episode" twice in the same second). This looks like the dev build's double fetch and isn't new.
- Cancel with reason **Other** and empty details correctly refuses with "Say why the episode is cancelled." (My first T-11 cancel attempt used Other, which is why it didn't go through the first time.)
- Cara opening a Trust B episode URL gets "You don't have access to this hospital". The same episode ID under Trust A's patient-form URL gets "Episode not found."

## Console and network errors

- No console errors or warnings on any page checked (POA Summary, fill-in form, patient pages).
- Claude in Chrome's network log listed `POST /api/logout` and `POST /api/dev/login` as **503** on every user switch, but sign-in worked every time. The same calls return **204** from curl (both through :3000 and :8080) and from an in-page `fetch`, and no service worker is registered. I'm treating it as a recording artefact of the tool, not an app error.
- No other unexpected 4xx or 5xx.

## Hand checks for Rahul

None. Every case could be run. T-13 was run by simulating the failed request in the page instead of a real network block.

## GIFs

- `~/Downloads/S28-T01.gif`: the main happy path (T-01: Fill in for the patient, the Hospital B1 card, Check each answer, Start fresh, reload).
- There were no failures, so there are no other GIFs.
