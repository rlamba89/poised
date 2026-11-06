# S47 test report 1: assessments, STOP-Bang

**16 cases + 5 regressions: 20 passed, 1 failed, 0 blocked.**

- **Environment:** commit `10b1087` plus the uncommitted S47 work (52 changed or new files in `git status`), local `make dev` at http://localhost:3000 (API :8080 healthy). Thursday 5 Nov 2026, 17:29–18:00 IST. Chrome (Claude in Chrome), desktop window 1456×829.
- **Phone size:** the Chrome window wouldn't resize below 1280 wide, so the phone-size case (T-11) used the project's pinned Playwright (`e2e/.venv`, Chromium, 390×844, mobile) against the same local app.
- **Test data (all made up, Hospital A1):**

| Episode | Patient | Setup | Used for |
| --- | --- | --- | --- |
| E1 `074f9411` | Tom MT S47 1730 One (M, 1960) | Demo sleep HQ, patient sent | T-01–T-03, T-11, T-12, T-14–T-16 |
| E2 `6fec85b0` | Ann MT S47 1736 Two (F, 1980) | Demo sleep HQ, filled in for the patient | T-04, T-07, R-01, R-03 |
| E3 `3145d6fe` | Raj MT S47 1745 Three (M, 1955) | Demo STOP-Bang HQ | T-05, R-02 |
| E4 `bfc52fa3` | Sid MT S47 1742 Four (M, 1970) | Demo sleep HQ | T-06 |
| E5 `c33d9520` | Eve MT S47 1744 Five (F, 1990) | Demo sleep HQ, no height or weight | T-08, then cancelled (T-13) |
| E6 `de6a5cea` | Kim MT S47 1746 Six (sex Other, 1965) | Demo sleep HQ | T-09, T-10, then archived by Sue (T-13) |
| E7 `33c86055` | Ian MT S47 1750 Seven (M, 1975) | Demo sleep HQ, HQ not sent | T-10 repro, exploring |

## Results

| Case | Result | Note |
| --- | --- | --- |
| T-01 7 of 8 filled, sources | Pass | Matches exactly: S, T, O, P from HQ; B 37 from Hospital BMI; A 66 and G Male from Profile; N Missing in orange; Incomplete; Confirm greyed; Add assessment menu gone. |
| T-02 Confirm and POA Summary | Pass | N No → Clinician, "6 (High risk)". "Confirmed by Cara Clinician on 05/11/2026, 17:33". Kept on reload. POA shows STOP-Bang 6 (High risk) under Hospital BMI on both tabs. Hospital BMI shows "37 (measured …)", not the plan's "37.0" (see exploring). |
| T-03 A change needs a new Confirm | Pass | "Changed: confirm it again", 7 (High risk). Undo restored No and "Confirmed by". Save gave "Not confirmed", and the POA said "Not confirmed". Confirm gave "7 (High risk)" on the POA. |
| T-04 Which BMI (trust setting) | Pass | On: 37 Hospital BMI, 4 (High risk). Off: 28.4 Patient reported BMI, 3 (Intermediate risk). Back on: 37. The setting is listed under Observations with its help text. Restored to "Use Preopsis's default". |
| T-05 HQ holding STOP-Bang | Pass | 8 of 8 filled. S, T, O, P and N from HQ, B 29.4 Patient reported BMI, A and G Profile. 5 (High risk), Not confirmed. POA "Not confirmed", then 5 (High risk) after Confirm. |
| T-06 Before the patient sends | Pass | "2 of 8 filled: answer the 6 items highlighted." After the send and a reload, S, T, O, P came from HQ and B from Patient reported BMI (28.4). |
| T-07 Clinician changes a pre-filled item | Pass | S No → Clinician, score down one (4 to 3). After reload S stays No/Clinician; T, O and P stay HQ. |
| T-08 Typed BMI; a bad one | Pass | B Missing. "abc" shows "A number, one decimal place". 36.5 shows "Entered by clinician" and scores the point (4 with N answered). With "2", Save is refused: "BMI: that answer isn't one this assessment takes." A reload shows the last saved state. |
| T-09 Sex Other | Pass | G Missing with "The patient's record doesn't say male or female, so the score stays Incomplete, as in the HQ." Confirm stays greyed with every other item answered. |
| **T-10 Two tabs** | **Fail** | Step 1: tab 2's add gets a 409, but "STOP-Bang is already on this episode." disappears straight away, so the user never sees it. The card then shows the one STOP-Bang. Step 2 passes: the "changed somewhere else" message with Reload, and Reload shows tab 1's answer. |
| T-11 Patient never sees it (phone) | Pass | At 390×844 (Playwright): the Thank-you screen and the home show only "Demo sleep HQ"; no Assessments or STOP-Bang; no horizontal scroll. The patient's episode API has only the HQ's own keys (no neck answer, no assessment). |
| T-12 Other hospital and other trust | Pass | Val and Bea get "You don't have access to this hospital" on both URLs. By API, every assessment route and add is a 404 for both, including through Val's own A2 path. |
| T-13 Cancelled and archived are read-only | Pass | Cancelled E5 and E6 (archived by Sue) show items and Confirm state, with no buttons, inputs or Add menu. A direct PUT on the cancelled episode is a 409: "This episode is cancelled, so it can't be changed." |
| T-14 Assessments switched off | Pass | Off for A1: no card, no STOP-Bang POA line, and no `/assessments` calls; the API answers 403 "Assessments aren't turned on at this hospital." Back to "Follow trust (On)": the card and POA show 7 (High risk) as before. |
| T-15 Audit log | Pass | For "MT S47 1730 One": "Opened an episode's assessments", "Added an assessment", "Saved an assessment's answers", "Confirmed an assessment", each by Cara at Hospital A1. While the feature was off, a "Refused (403)" row shows too. |
| T-16 Reload, deep links, Back | Pass | An unsaved change is dropped on reload. POA, then Back, shows the episode with the assessment as saved. Forward shows the POA. A pasted episode URL in a new tab opens with its Assessments card. |
| R-01 Observations | Pass | Weight 120 corrected to 90 with a reason: the old reading is struck through and marked "Corrected", the hospital BMI is 27.8, and the POA matches. The unconfirmed STOP-Bang's B followed to 27.8 at once. |
| R-02 Validate and Complete review | Pass | On E3 (Demo STOP-Bang HQ, Ready for review; one set, my own data rather than Demo pre-op HQ): Validate, then Complete HQ review. "HQ review completed by Cara Clinician…", status Ready for POA. |
| R-03 Fill in for the patient | Pass | "Sent by Cara Clinician on the patient's behalf on 05/11/2026, 17:36." The episode moved on (auto-triaged). |
| R-04 Ready-made STOP-Bang in the editor | Pass | As Alex: new HQ "MT S47 1800 R04", a BMI on page 1, Ready-made score → STOP-Bang on page 2 (a page holding a BMI takes no other content). Inserted and locked; published as v1. The platform's STOP-Bang assessment doesn't appear in the trust's questionnaire list. |
| R-05 Settings ranges | Pass | Peak flow "Usual to" 700 → 650 saved and kept on reload, then "Use Preopsis's default" restored (700). |

## Failures

### F-1 (Minor) T-10: "STOP-Bang is already on this episode." flashes and disappears

**Steps:**
1. As Cara, open an episode with no assessment (e.g. E7 `33c86055-7fec-4fca-a46c-b693eaefae09`) in two tabs.
2. In tab 1, **Add assessment → STOP-Bang**.
3. In tab 2, without reloading, **Add assessment → STOP-Bang**.

- **Expected:** "STOP-Bang is already on this episode." shows, then the card shows the one STOP-Bang.
- **Actual:**
  - The card appears with no message.
  - The network shows `POST …/episodes/{eid}/assessments` 409, then `GET …/assessments` 200 (the audit has `assessment.add` 409 at 12:16:35 UTC for E6).
  - A page watcher I added before the click caught the exact text "STOP-Bang is already on this episode." being rendered. It was gone before the first screenshot straight after the click, so the reload after the 409 seems to clear it.
- **Where:** `/o/…fa/h/…0a/episodes/33c86055-…` (and E6 `de6a5cea-…`), signed in as Cara Clinician.
- **Console:** no errors.
- **GIF:** `~/Downloads/S47-T10.gif`.

## Found while exploring

1. **Cosmetic: the source badges are cut off.** "Patient reported BMI" shows as "PATIENT REPORTE…" and "Entered by clinician" as "ENTERED BY CLINI…", with no tooltip. This is at 1456 wide; the badge's text is 130px in a 112px box. It's the key fact on row B for T-04, T-05 and T-08.
2. **Minor: an out-of-range BMI isn't flagged in the browser.**
   - 2 and 151 show no hint under the box, the badge scores them, and Confirm stays active.
   - Only the server refuses them, with "BMI: that answer isn't one this assessment takes.", which doesn't say the allowed range (5–150).
   - By contrast, "abc", "36.55" and "-5" get "A number, one decimal place" at once.
3. **Cosmetic: a read-only card still gives instructions.** A cancelled or archived episode's card still says "6 of 8 filled: answer the 2 items highlighted." when nothing can be answered.
4. **Question: a confirmed assessment beside a newer hospital BMI.**
   - On E1, the weight was corrected 120 → 80 after Confirm. The Observations card and the POA now say Hospital BMI 24.7, but the confirmed STOP-Bang still shows B "37 · Hospital BMI" and 7 (High risk).
   - Nothing tells the clinician the reading changed since Confirm.
   - This is by design (builder note 4: a confirmed assessment shows what was confirmed). Raising it in case a hint is wanted.
5. **Cosmetic: the source doesn't go back to HQ.** Clicking an item back to the value the HQ gave keeps the source "Clinician" (S on E2: No, then Yes again).
6. **Keyboard: Enter doesn't save.** Enter in the BMI box does nothing; you have to click Save or Confirm. Tab reaches the Yes/No buttons.
7. **Plan wording.** T-02 expects "Hospital BMI 37.0"; the app shows "37 (measured …)", the same as S46's Observations card.
8. **Held up:**
   - Double-clicking Save or Confirm sends one request.
   - "36,5" (comma) is stored as 36.5.
   - Another episode's assessment id in the URL is a 404 "Assessment not found." (PUT and confirm).
   - An unknown assessment ("dasi") is a 400 "There's no such assessment."

## Console and network errors not tied to a case

- None. No console errors or warnings on the episode, POA, settings, admin or editor pages.
- The only 4xx responses were the ones the cases provoked: 409 add, 409 stale save, 422/400 BMI, 403 off, 404 other hospital or trust.

## GIFs

- `~/Downloads/S47-T01-T02-happy-path.gif`: the main happy path (T-01 → Confirm → POA Summary).
- `~/Downloads/S47-T10.gif`: F-1.
- The phone-size screenshots for T-11 are in the session's scratchpad, not the repo.
