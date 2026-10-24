# S27 test report 1

**Summary: 19 passed, 2 failed, 0 blocked** (16 cases T-01 to T-16, plus 5 regression cases R-01 to R-05).

- **Environment:** `d8d183b` plus the uncommitted S27 work (git status not clean: the builder's changes, migration 00027). App run by the conductor with `make dev` (web http://localhost:3000, API http://localhost:8080, database at 00027). 24 Oct 2026, 12:13–12:45 BST.
- **Browser:** Claude in Chrome, Rahul's profile.
- **Tester:** manual-test agent, automatic run.

## How I tested (deviations from the plan)

- **Phone size:** Chrome's window couldn't be resized (`resize_window` reported success, but the viewport stayed 1512 × 805). So every patient step ran in a **390 × 780 iframe** of the real app, opened from a `localhost:3000` page. That gives a real 390-px viewport: media queries, layout and scrolling all apply. "Nothing wider than the screen" was checked as `scrollWidth == clientWidth == 390`.
- **Test data set up through the app's own API:** to save time, I set up repeated preconditions with a small script calling the same API routes the UI uses (dev login, `POST …/episodes`, the patient's `continue` / `dob` / `answers` / `submit`, Cara's validate and `complete-review`). This covered:
  - the Trust B sent HQs for T-04, T-07 and T-09;
  - the Trust A episodes for T-04, T-07, T-09 and T-11 to T-13;
  - T-03's and T-15's first sent HQ;
  - T-13's reviewed episodes.

  Everything under test ran in the browser UI: T-01's two episodes and the patient's whole journey, T-03's second episode (found by search), every card and button, Start fresh, Validate, the POA Summary, archive and un-archive, cancel, Settings, Audit, the admin console, publishing and the jobs page.
- **Captured messages** (T-13 and the invite links) were read from `GET /api/qa/messages` as Sam Superadmin: the same data `/qa/messages` shows.
- **T-16 Print:** I rendered the POA Summary with print media to a PDF using headless Chromium (Playwright `page.pdf()`, signed in as Cara), then opened the PDF.
- **T-15 step 3:** the window couldn't be made 500 px high. At 805 px the summary still scrolls (1328 px tall), so the scroll-position checks are meaningful.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Trust B history fills a Trust A HQ, the card names Hospital B1 | Pass | No question first. Both sets say Please check and Send is disabled. The card reads "We've filled in 4 answers from your questionnaire at Hospital B1 on 24 Oct 2026. Please check them." with all 3 buttons. Diabetes is No and snoring Yes, each with "From your answers on 24 Oct 2026"; pregnancy is empty. No sideways scroll at 390 px. |
| T-02 Start fresh asks first, empties one set, stays empty | Pass | The confirm text is exact, and Keep the answers restores the card. After Yes, the set is empty, with no card or markers. After a reload, About you shows **Start** and stays empty, and Your health keeps its answers. Answered both sets and sent. |
| T-03 No Start fresh for the same trust's answers | Pass | "We've filled in 4 answers from your questionnaire on 24 Oct 2026. Please check them.": no hospital named, and only the two buttons. |
| T-04 Start fresh refused after a change in another tab | Pass | The red message is exact, the confirmation closes and the card stays. After a reload, snoring is still No and nothing was cleared. |
| T-05 Trust A's JSON names no Trust B records | Pass | The `prefill` entries have `otherTrust: true`, `hospital: "Hospital B1"`, `questionnaire: "Demo pre-op HQ"` and no `fromForm`. About you has `prefillChoice: "start_fresh"`. Trust B's episode id `4af84aac…` isn't in the page. |
| T-06 Validate and POA Summary markers | Pass | The Check each answer line is shown. Snoring says "(matched by link)" and diabetes "(matched by same wording)". About you has the Start fresh sentence and no markers. The POA Summary has "Pre-filled on Page 1" naming Hospital B1 (another trust), and nothing for About you. |
| T-07 Staff filling in get the same answers, card and Start fresh | **Fail** | As written (the patient never opened the Trust A invite), both sets say **Start**, nothing is copied and no card shows. The same steps on a person whose Trust A record had joined their account (they had entered their date of birth on the Trust A link) passed: see Failures. |
| T-08 Audit: Trust B sees the reads, Trust A only its own | Pass | Max's log has 3 rows, "Patient's answers read to pre-fill their HQ at another trust", with no "(they agreed)": T-01 and T-04 by Patient (their link), T-07b by Cara Clinician. The CSV has the same label, the patients and the Hospital B1 episode ids. Ada's CSV for the day contains no Trust B episode or patient-record id and no "Hospital B1". (T-07 as written has no row, since nothing was read.) |
| T-09 A suspended trust's answers aren't used | Pass | With Trust B suspended, both sets say Start and Your health opens with no card. After re-activating, Bea signs in and her episode list loads (200). |
| T-10 Setting gone from Settings and the admin console | Pass | Trust A Settings (Ada), Trust B Settings (Max) and `/admin/trusts/<Trust A>` show no "other trusts" setting. "Keep episode data for (years)" is still there. `defs` has no `crossTrust…` key. |
| T-11 Sent then archived: Sent with a read-only View | Pass | The link opens "Thank you, Sam…" with no Send. The home shows **SENT** with View, which opens the thank-you page. T-11b (unsent, then archived): "This questionnaire is closed…", and the home says **CLOSED BY THE HOSPITAL**. |
| T-12 Archived hides NOT RESPONDING | Pass | After jobs run as of 01/11/2026 (23 marked), the badge shows. Archived, it shows ARCHIVED only, and still none after a reload. Un-archived (with a reason), the badge is back. |
| T-13 Neutral Cancelled text and email | **Fail** (Minor) | Each body is exactly right in all 4 statuses (HQ not complete, Ready for review, Ready for POA, Ready for admission), and so is the SMS for the one with a mobile. But the email **subject** still says "Hospital A1: your health questionnaire was cancelled". Settings were switched back to the default afterwards. |
| T-14 Publish refusal names the format in words | Pass | "…its answer format changed from any text to phone number.", "…to NHS number.", "…to UK postcode.", and "…to the pattern “^[0-9]+$”." Also checked: "…to email address." Back to Any text, it publishes. |
| T-15 Clickable POA Summary answers | Pass | Diabetic, the snoring line and the Anaesthetics line each open Validate at the question (`?q=…`), centred where the page allows and ringed in yellow for about 2 s, with "← POA Summary". The browser's Back and "← POA Summary" both return to the same tab and scroll position (Diabetic at the same y). The Patient answers tab is `?tab=patient` and survives Back and a reload. Step 9 (a T-01 "Pre-filled on" line) opens Validate at snoring. A direct URL opens highlighted, with no "← POA Summary". |
| T-16 Print renders the POA Summary to PDF | Pass | One A4 page. Each answer line is readable as plain text (Diabetic, both "Changed by the clinician" lines), with no URLs or extra symbols. Nothing is cut off at the sides. Tabs and buttons still show (C9, out of scope). |
| R-01 Same-trust follow-up: Everything is still correct jumps to the first empty question | Pass | It jumps to pregnancy (focused) in Your health and to weight in About you. Send works. |
| R-02 "Changed by patient: pre-filled “…”" | Pass | T-04's snoring shows "Changed by patient: pre-filled “Yes”", and the header says "…and the patient changed 1." |
| R-03 Rewording a published option is refused | Pass | "the option “Option 2” was reworded to “Option two”…" |
| R-04 Cancel and archive make it read-only; the home says Cancelled | Pass | A cancelled episode has no Validate, Fill in or Resend: only Add comment. The patient's home says **CANCELLED BY THE HOSPITAL**, and the cancelled invite link says "This link doesn't work". |
| R-05 A correction shows "Patient answered: …" and saves | Pass | "Patient answered: No" on snoring and on sickness after an anaesthetic, saved and validated. |

## Failures

### F1 (Major for the feature, or a test-plan question): staff filling in get no other trust's answers unless the patient has already proved their date of birth at this trust (T-07)

**Steps:**
1. Bea (Hospital B1) creates an episode on the Demo pre-op HQ for Sam "S27 T07 1225", born 17/05/1980, email `s27.t07.1225@example.test`. The patient answers and sends.
2. Cara (Hospital A1) creates a Demo follow-up HQ episode with the same name, date of birth and email: `97dc067b-9216-4fdc-90ac-639cc1a09601`. The patient does **not** open the Trust A invite.
3. Cara presses **Fill in for the patient**.

**Expected:** both sets say Please check, and Your health's card reads "We've filled in N answers from Sam's questionnaire at Hospital B1… Please check them with Sam.", with Start fresh.

**Actual:**
- Both sets say **Start**. No answers are copied and no card shows.
- `GET …/episodes/97dc067b…/hq` has `answers: []`.
- Trust B's audit log has no `prefill.read` row for this patient.
- No console errors, and every request returned 200.

**Where:** `http://localhost:3000/o/…fa/h/…0a/episodes/97dc067b-9216-4fdc-90ac-639cc1a09601/patient-form`, signed in as Cara Clinician.

**Cause, as far as a tester can see:**
- Go finds other trusts' records only through the person's sign-in account. The `prefillForm` comment says "other trusts' records are found only through the person's account (S15)".
- A new Trust A record joins that account only when the patient enters their date of birth from a Trust A link.
- The plan's T-07 step 1 explicitly says not to open the invite, so the Trust A record has no account yet.

**Control (passed):**
- I set up person "S27 T07b 1240" the same way. The patient then entered their date of birth on the Trust A link (via the API: `continue` and `dob` only, without loading the form). Then Cara pressed Fill in for the patient. Episode `4648d92b-f482-470b-a698-e3eab5ce7705`.
- **Everything in T-07 then passed:**
  - Both sets said Please check.
  - The card read "We've filled in 4 answers from Sam's questionnaire at Hospital B1 on 24 Oct 2026. Please check them with Sam." and offered Start fresh.
  - The confirmation read "…again with Sam. This can't be undone."
  - About you opened empty.
  - Validate said "Cara Clinician chose “Start fresh” with the patient on 24 Oct 2026…".
  - The T-08 audit row was by Cara.

**For the builder to decide:**
- Either the plan's T-07 setup is wrong (the patient must have joined first, so on-behalf pre-fill from another trust is only for patients known to the account),
- or on-behalf pre-fill should also find other trusts' records for a patient who hasn't joined yet.

Matching records across trusts without the patient's date-of-birth proof is an identity and privacy decision, so it may need Rahul. If the behaviour stays as it is, the T-07 step and the brief's "staff get other trusts' answers too" should say it needs a joined account.

**GIF:** `~/Downloads/S27-T07.gif`

### F2 (Minor): the Cancelled email's subject still says "your health questionnaire was cancelled" (T-13)

**Steps:** Ada switches on "Email the patient when the episode is cancelled". Cara cancels an episode in **Ready for admission** (`e74ee0ef-6e83-427f-ac64-7321c0db72f8`, "S27 T13d 1232"); the same happens in the other three statuses.

**Expected:** the whole email reads neutrally for any status. The brief: "neutral wording, since Cancel is allowed from any status".

**Actual:**
- **Body (correct):** "Hello Sam,\n\nHospital A1 has cancelled your pre-operative assessment. Please contact the hospital if you have any questions."
- **Subject:** "Hospital A1: your health questionnaire was cancelled". That's odd for a patient whose questionnaire was sent and reviewed long ago (Ready for POA, Ready for admission), and it doesn't match the new body.
- The subject is hardcoded in `apps/api/internal/invite/invite.go:305`, not in the editable template.

**GIF:** none. It's a message-text issue; the captured message is quoted above.

## Found while exploring

- **Changing a question in a stale tab after sending is refused clearly.** In T-04's second tab, after the patient had sent from the first, changing snoring showed "Your answers weren't saved. These answers have already been sent, so they can't be changed here." with Reload. Good.
  - Trust A's audit log records that refused PUT as "Patient saved answers" with Result 409. The Result column shows it failed, but the label reads as if it worked: cosmetic, and older than S27.
- **Another patient's episode is refused.** As a signed-in patient, `/home/episodes/<someone else's episode>` shows "Episode not found." (API 404), and `POST /api/p/episodes/<someone else's>/prefill/<set>` with `start_fresh` is a 404. Good.
- **A double-click on "Yes, start fresh"** (two clicks in a row) cleared the set once, with no error shown.
- **After a trust is re-activated, its answers fill the open forms.** T-09: once Trust B was re-activated, reloading the patient's Trust A form (still with nothing typed) showed both sets as Please check with Trust B's answers. That follows the "never filled again once a set has a row" rule (no row had been written while suspended), so it looks intended. Noting it in case it isn't.
- **A "NEW QUESTION: WON'T PRE-FILL" badge shows on questions already published.** It showed on "Your contact number" and "Select One question" in the published v1 and in drafts v2 and v3 of "S27 formats 1235", although v3's questions were published in v1 and v2. If the badge means "new in this version", that's wrong. If it means "no library key, so it won't pre-fill from other questionnaires", it's fine. Low confidence, and not S27's change.
- **Every Text question's Format list now includes "Email address"**, and it's named in words in the refusal ("…to email address."). Good.

## Console and network errors

- No console errors or warnings on the staff pages (POA Summary, Validate, Fill in for the patient) or on the patient pages.
- No unexpected 4xx or 5xx. The only 409s were the ones the cases expected (T-04's refused Start fresh, and the stale save after sending).

## GIFs and files

- `~/Downloads/S27-T01-T02.gif`: the main patient happy path at 390 px: the card naming Hospital B1, Check each answer, Start fresh, then send.
- `~/Downloads/S27-T15.gif`: clickable POA Summary answers, Back and "← POA Summary".
- `~/Downloads/S27-T07.gif`: F1, Fill in for the patient showing Start with nothing copied.
- Audit CSVs: `~/Downloads/audit-2026-10-24-122428.csv` (Trust B) and `~/Downloads/audit-2026-10-24-122947.csv` (Trust A).
- The T-16 print PDF: `/private/tmp/claude-501/-Users-rahul-poised/89abe13c-35bf-4ef8-b749-f29beea6dff0/scratchpad/print/poa-t15.pdf`

## Hand checks for Rahul

None. No case needed a browser dialog, and Print was rendered to a PDF as agreed.

## Test data left behind

- Patients "Sam S27 T01 1215", "S27 T03 1240", "S27 T04 1225", "S27 T07 1225", "S27 T07b 1240", "S27 T09 1225", "S27 T11 1230", "S27 T11b 1230", "S27 T12 1230", "S27 T13a–d 1232" and "S27 T15 1240" (made-up emails `s27.*@example.test`; one mobile, +447700900131).
- The questionnaire "S27 formats 1235": v1 and v2 published, v3 a draft with a reworded option.
- Trust B was suspended, then re-activated (now **Active**).
- Trust A's cancelled-message settings are back to "Use Preopsis's default".
- The jobs were run once as of 01/11/2026 12:27. That sent 21 reminders and marked 23 episodes "not responding" across the trusts, including older test data.
