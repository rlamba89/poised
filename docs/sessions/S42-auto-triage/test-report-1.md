# S42 test report 1: auto-triage and the Auto-triaged worklist

**Summary: 18 passed, 1 failed, 0 blocked** (14 cases + 5 regression checks).

- **Environment:** HEAD `cb8857f` with the S42 work uncommitted (dirty tree). 27 Oct 2026, 01:36–02:10 GMT.
- **App:** the conductor's `make dev` (web http://localhost:3000, API :8080, evaluate service 127.0.0.1:8090), freshly migrated and seeded.
- **Browser:** Chrome (Claude in Chrome) for every staff screen.
- **Tester (automatic run):** manual-test agent.

## How I tested (and where I deviated)

- **Phone-size patient screens.** Chrome wouldn't resize below its tiled size: `resize_window` had no effect, and the viewport stayed 1400×717. So every patient send ran in **real Chromium at 390×844** (Playwright, `is_mobile`) through the patient's UI:
  - the invite link from `/api/qa/messages`, then Continue and the date of birth;
  - Start, tapping each option, Finish this section, Send my answers, and confirming;
  - each run took screenshots and checked that the page had no sideways scroll (`scrollWidth <= innerWidth`). It was **true every time**, with no console errors.
  - The script is `/private/tmp/claude-501/-Users-rahul-poised/e354aaca-05a3-4639-ae42-c43325b43018/scratchpad/mt/patient.py`.
- **Test data.**
  - The HQ was built and published in the UI (T-01, T-02, then v2 and v3), and T-03's episode was created in the UI.
  - To save time, later episodes were created through the API as the same staff user (dev login → `POST …/episodes`, the request the New episode page sends). Each one's page, worklists, Checked, re-open, cancel, archive and fill-in-for-patient were then used in the browser.
  - Made-up patients only ("Sam MT42 …", born 17/05/1980, 07700 9004xx).
- **v1 is retired once v2 is published.** T-10 and T-13 say to use "MT S42 v1", but by then only the newest version can be given to patients (`Choose a published HQ.`). This is a plan issue, not a bug.
  - T-10 used v2.
  - I ran **T-13 before T-12** (on v2) so its low-risk send still targets Ready for admission.
- **Hospital A1's Ready for admission text** was switched on in T-08 and left on through T-12 and R-02 (the plan switches it off after T-09 and on again in T-12). It was cleaned up at the end: every Hospital A1 setting is back to "Use the trust's setting".
- **Plan wording vs the app.** The disclosure categories have no "Cardiac" or "Sleep", so I used Cardiovascular (chest pain) and Lifestyle (tiredness). This made no difference to the flags.
- **v2 differs slightly from the plan.** In v2 I gave the Smoker disclosure a note ("Smoker", Lifestyle), and T-09's patient answered Yes to smoking (still ASA II, low risk). The reason: the POA Summary lists only disclosures that have a note, and the plan's smoker disclosure (code only) produces nothing there (see T-03).

## Results

| Case | Result | Note |
|---|---|---|
| T-01 Rules page message setting | Pass | New box unticked by default, with the "Off: …" line. Ticked, it says the message goes once a clinician has checked. Both states kept after a reload. Default rules unchanged. |
| T-02 Build and publish | Pass | Publish window: "This version auto-triages: ASA I–II, no red flags → Ready for admission. Patients with no ASA grade count as ASA I." Published. |
| T-03 Low-risk patient on a phone | Pass | Thank-you page, fits 390 px. On Auto-triaged (count 8 → 9), not on Ready for review. Teal card, Checked button, both sets stamped "Auto-triaged (ASA II, no red or amber flags)" with View. History rows by Patient and "Automatic (Low risk)", notes by Automatic, no Undo. POA Summary has the teal line. The by-line and listed Smoker note were checked on v2 (T-09 episode), since the plan's code-only disclosure lists nothing (expected). |
| T-04 Checked, reload, Back/Forward, two tabs | **Fail (Minor)** | Checked works: "Checked by Cara Clinician on …", button gone, note "Auto-triage checked", kept after a reload, off the tab (count −1), `?worklist=auto_triaged` kept on Back/Forward. **But the second tab's refused Checked shows no message** (F-1). Nothing was recorded twice. |
| T-05 High risk (heart failure) | Pass | Ready for review, Suggested ASA III, sets "Not started" with Validate, no auto-triage lines. On Ready for review, not on Auto-triaged. |
| T-06 Red / amber | Pass | Red: Ready for review, no auto-triage note. Amber: Ready for review with Automatic note "Not auto-triaged by the rule “Low risk”: the answers have a red or amber flag. Review this HQ by hand." Neither on Auto-triaged. |
| T-07 Fill in for the patient | Pass | Ready for admission. Card and stamps say "(no ASA grade, no red or amber flags)". "HQ sent" by Cara Clinician, then Automatic (Low risk). Notes "HQ completed on the patient's behalf" (Cara), then the auto-triage line (Automatic). On Auto-triaged. Wording nit in E-1. |
| T-08 Message at once by default | Pass | Ready for admission text captured 2 s after the invite, before any Checked. Note "Ready for admission message sent by text message." by Automatic. Card "Not yet checked by a clinician." with no message line. |
| T-09 Message held until Checked (v2) | Pass | Publish window ends "…The patient's message goes when a clinician clicks Checked." After the send, only the invite was captured, and the card adds "The patient's message goes when you click Checked." After Checked, the text arrived; notes "Auto-triage checked" and "Ready for admission message sent by text message.", both by Cara. |
| T-10 Other hospital's episode | Pass | Val (A2) sees it on Auto-triaged (count 1). Cara: the A2 address gives "You don't have access to this hospital", and the same episode under A1 gives "Episode not found." with no details or Checked. Not on Cara's tab. Checked as Cara through the API: 404 under both hospitals' paths (done to confirm). |
| T-11 Re-open (Sue Super) | Pass | Cara has no Re-open. Sue without a reason: "Say why the HQ review is being re-opened." With "MT: check by hand": Ready for review, card and stamps gone, sets "Not started" with Validate. History "HQ review re-opened: Ready for admission → Ready for review", Sue, with the reason. Off Auto-triaged. Validate ×2 and Complete HQ review gave Ready for POA. |
| T-12 Target Ready for POA (v3) | Pass | Publish window "→ Ready for POA". Episode Ready for POA, note "…→ Ready for POA", history "Auto-triaged: Ready for review → Ready for POA". On Auto-triaged, and no Ready for admission text (switch on). |
| T-13 Evaluate service stopped | Pass | Service killed (PID 77787): health failed, web and API up. Patient got the normal thank-you page. Episode Ready for review with "The automatic checks didn't run. Review this HQ by hand." (Automatic), no card. Not on Auto-triaged. Restarted (`make evaluate`): health ok. The next low-risk send was auto-triaged to Ready for admission. |
| T-14 Cancelled / archived | Pass | Count equalled rows each time (11, then 10, then 9, then 10). The cancelled one ("Patient withdrew") left the list, and its page has no Checked. Sue's archived one left the list and came back unchecked after un-archive ("MT"). |
| R-01 Review by hand | Pass | T-05 episode: Validate both sets, then Complete HQ review: Ready for POA. |
| R-02 Hand-picked Ready for admission text | Pass | The text went, with the note by Cara. Switch then cleaned up. |
| R-03 Test the whole HQ | Pass | Smoker: "Suggested ASA II", "Low risk: auto-triage to Ready for POA" (v3's target). See E-2. |
| R-04 Suggested ASA, POA ASA section | Pass | T-03: "Suggested ASA II: Smoker (77176002) – Current smoker (the anaesthetist confirms)". POA "Pre operative assessment ASA grades: Suggested ASA II". |
| R-05 No error on a normal send | Pass | "The automatic checks didn't run" appeared only on the T-13 send made while the service was down. |

## Failures

### F-1 (Minor): a refused Checked gives no plain-words message

**Steps** (as Cara Clinician, Hospital A1):
1. Open an unchecked auto-triaged episode in two tabs (e.g. `…/episodes/fad43eca-8e74-4c6c-bdbb-91cbc1a21a93`, and again with `…/94f96cde-4454-49c3-97c9-03468dbc82ec`).
2. Click **Checked** in the first tab.
3. In the second tab, without reloading, click **Checked**.

**Expected:** the second click is refused in plain words ("This episode changed since you opened it…" or "already checked"), and the page shows the episode as checked.

**Actual:**
- The POST returns **409** with `{"code":"stale","error":"This episode changed since you opened it…"}`.
- The page then reloads straight to "Checked by Cara Clinician on …" with **no message at all**. The clinician isn't told their click did nothing, or that someone else checked it.
- Nothing is recorded twice (one "Auto-triage checked" note).

**Cause (from reading the page code, not changed):** in `EpisodePage.tsx`'s `AutoTriage` card, `check()` sets `error` and calls `onChecked()` on a stale error. But the error `<Alert>` is drawn only inside the not-yet-checked branch. Once the reloaded episode is checked, that branch (and the message) is gone.

**Caveat:** if the second tab has been in the background, it reloads by itself when you switch to it, so the button is already gone. Seeing this needs a tab that hasn't reloaded yet, e.g. a clinician who left the page open on screen.

**GIF:** `~/Downloads/S42-T04.gif`.

## Found while exploring

- **E-1 (Cosmetic): a staff-sent HQ is described as the patient's.** When staff send on the patient's behalf (T-07):
  - the card says "Moved to Ready for admission automatically **when the patient sent the HQ**";
  - the confirm window says "the episode moves to Ready for review";
  - the sent page says "The answers are with the team for review."

  All three read oddly when the episode is auto-triaged.
- **E-2 (Minor): the Rules test page shows Low risk firing on an amber flag.** On Rules → Test the whole HQ (v3), Smoker Yes + Tired Yes (amber) still shows "Rules that fire: Low risk: auto-triage to Ready for POA". On a real send the S42 guard stops it ("Not auto-triaged…: the answers have a red or amber flag"). An author testing the HQ is told an amber patient will be auto-triaged when they won't be. The rule's text and the Publish window also say "no red flags", while the stamp and guard say "no red or amber flags".
- **E-3 (unexplained, not reproduced):** on one UI double-click of Checked (T-12 episode `4a74e155…`), Chrome's network log showed a single `POST …/triage-checked` with status **503**, yet the episode ended up checked once.
  - The Go handler and the API have no 503 on this route (only health and publish return 503).
  - A second double-click on another episode sent one request (204), so the button's loading state works.
  - Three requests sent at the same moment gave 204, 409 stale and 409 stale.
  - Probably a quirk of the network log. Recorded only in case it recurs.
- **E-4 (plan): T-10 and T-13 can't use v1.** Publishing v2 retires it.
- **POA Summary lists only disclosures with a note.** This is how it already worked, not S42: the plan's code-only smoker disclosure leaves the POA Summary with "No answers produce anything for the summary yet." and no per-set by-line. With a note, the by-line "Auto-triaged (ASA II, no red or amber flags) · Patient on …" and "Smoker" show as expected.
- **Admins see the full set of controls:** Ada (admin) sees Re-open and Archive on an auto-triaged episode, as a super clinician does (as designed, A-5).

## Console and network errors

- **Console:** no errors or warnings on the episode page, the worklists, the POA Summary, or the 390×844 patient runs.
- **Network:** only the expected 409s (T-04 stale clicks) and 404s (T-10), plus E-3's one unexplained 503.

## Evaluate service

- At the end exactly **one** evaluate service is listening on 127.0.0.1:8090: **node PID 93123**, health `{"ok":true,…}`.
- It's the copy I started in T-13 with `make evaluate` (log `/tmp/mt-s42-evaluate.log`). Its processes:
  - **make** 93046
  - **npm run evaluate** 93055
  - **npm run serve** 93081
  - **sh** 93116
  - **node** 93123
- The conductor's original node (PID 77787) was stopped in T-13. Its parent `sh` (77777) has exited too.
- The `make dev` API and web were never touched.

## GIFs

- `~/Downloads/S42-T03-T04-happy-path.gif`: T-03 and T-04, staff side.
- `~/Downloads/S42-T04.gif`: F-1.

Patient screenshots (390×844) are in `/private/tmp/claude-501/-Users-rahul-poised/e354aaca-05a3-4639-ae42-c43325b43018/scratchpad/mt/` (`t03-*.png`, …).

## Hand checks for Rahul

- None.
