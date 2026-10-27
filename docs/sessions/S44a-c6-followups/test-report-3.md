# S44a test report 3 (retest after verification 1)

**17 passed, 0 failed, 0 blocked.** This round re-ran the cases listed under "Retest after verification 1" in the plan: T-04, T-05, the new chain case, T-08 with the refused change after Publish, T-18 and T-02. It also re-ran T-19, T-20, T-21, all six regression cases (R-01 to R-06) and two happy paths (T-12's thank-you page and a full Complete HQ review).

- Commit `97e62e2`, working tree **dirty**: S44a's changes and the verification fixes are still uncommitted.
- 27 Oct 2026, 22:05–22:27 GMT.
- Local: web http://localhost:3000, API :8080, evaluate :8090 (`clinical@97e62e2e699f-dirty`).
- Tester: automatic run (Claude in Chrome). Rahul wasn't watching.

**How the test data was made.**
- **New HQ:** "MT S44a R3 chain 2210" was built as Alex in the editor and published. It has one patient set with B "Any chest pain?", then A "Does it spread to your arm?" (shown when B is Yes), then text C "How bad is it?" (shown when A is Yes). The Low risk rule was deleted.
- **Reused HQs from round 1:** "MT S44a snore", "MT S44a asa", "MT S44a scores", "MT S44a pubchk" and "MT S44a stopbang".
- **Episodes:** all five were made as Cara on **New episode**. Every patient was born 17/05/1980.
  - "Rob MT S44a R3 T04 2207" (`e7bed37f-…`)
  - "Asa MT S44a R3 T18 2208" (`e0f134cd-…`)
  - "Nia MT S44a R3 T02 2209" (`d810780e-…`)
  - "Cal MT S44a R3 chain 2216" (`ea325ff7-…`)
  - "Dee MT S44a R3 R03 2222" (`01665501-…`)
- **Answers:** every patient answered through their invite link from /qa/messages and the date-of-birth check.
- **Signing in:** later user switches used `POST /api/dev/login` from the page, which does the same as clicking a name on the stub login page.
- **API calls from the signed-in browser were used only:**
  - to confirm the refused writes on a published version (content save, add a set, reorder, delete);
  - for R-05's refused Complete review (the button is disabled in the UI);
  - for T-20's A2-path 404s.
- **Database reads (read-only):** the frozen `episode_answers` rows (chain, T-04), `form_outputs` (T-02's score; no `sent` copy) and the chain set's audience after the refused change.

**Phone size.** The Chrome window still wouldn't resize: `resize_window` reported success, but the viewport stayed 1512×805. The patient pages were checked at desktop width (see Hand checks).

## Results

| Case | Result | Note |
|---|---|---|
| Chain (new) | Pass | The patient answered B Yes, A Yes and C "severe", then changed B to No and sent. The frozen row is `{B: No}` only, so neither A nor C was kept. Cara's Validate shows only B. Correcting B to Yes shows A empty with no "Patient answered" line, and no C. Answering A Yes as Cara then shows C **empty**, with no "severe" and no "Patient answered: severe". The POA Summary's Patient answers tab has no "severe". |
| T-04 | Pass | The patient answered Snore Yes, then the follow-up (Yes, "loud"), then changed Snore back to No and sent. The episode lists only About you and Smoking. Cara corrected Snore to Yes and validated: **Sleep follow-up** reappears, **Not started**. Its Validate page shows both questions empty, with no "Patient answered". The POA Summary's Patient answers tab has nothing for it. The frozen row is `{}`. |
| T-05 | Pass | Same episode. The patient answered Smoke Yes and "10", then changed Smoke to No. In Cara's Validate, Smoke corrected to Yes shows **How many a day?** empty, with no "10". |
| T-08 (Publish) | Pass | On "MT S44a pubchk", Sleep follow-up's audience was switched to Patient. Publish was then blocked, listing: "Sleep follow-up · “Sleep follow-up” shows only when Sleep risk band is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set." The Publish button was disabled. With the audience back to Clinician, there was no problem and Publish was enabled; it was cancelled without publishing. |
| T-08 (stale tab after Publish) | Pass | A second tab had the chain HQ's set settings open, then the first tab published. Changing **Who is it for?** to Clinician in the stale tab gave a red banner, **"Only draft versions can be changed."**, and the page refreshed to read-only (V1 · PUBLISHED). The audience in the database is still `patient`. API check from the same tab: a content save, adding a set, reordering and deleting a set were each refused with 409 "Only draft versions can be changed.", so the content save no longer says "Someone else changed this chapter". |
| T-18 | Pass | Before the send, the episode page had no Suggested ASA line, even with both answers saved. After the send: "Suggested ASA III: Heart failure, unspecified (I50.9) (the anaesthetist confirms)". On Validate Heart: ASA III. Changing to No switched it at once to "Suggested ASA II: Smoker (77176002)", and back to Yes gave ASA III again, with no reload. |
| T-02 | Pass | Nights is hidden for the 46-year-old: the patient saw only Sleep. The patient answered Snore Yes and Loud No. Validate Sleep shows Sleep risk **1 (Low)**, not Incomplete. As in round 1, the POA Summary shows no score values (no band has a disclosure). The stored evaluate result has Sleep risk 1, band Low. |
| T-19 | Pass | On the episode page and the patient's list, Nights stays hidden for the 46-year-old. |
| T-20 | Pass | Val (A2) and Bea (Trust B) each opened the URLs of T-04's episode, its Sleep follow-up Validate page and "MT S44a warn"'s Rules page. Each showed "You don't have access to this hospital", with nothing about the HQ. Through Val's own A2 path, the episode and its `/hq` gave 404 "Episode not found.". |
| T-21 | Pass | The answer was corrected on Validate (Heart failure No → ASA II), then the page was reloaded: it still shows ASA II, matching the saved answer. Browser Back from Validate returns to the episode page. |
| T-12 (happy path) | Pass | Every send showed "Thank you, <first name>" and "Your answers have been sent to the hospital. We'll contact you about what happens next." That includes the auto-triaged Demo pre-op HQ send (R-03). This was checked at desktop width. |
| Complete HQ review (happy path) | Pass | Nia: Validate Sleep, then **Complete HQ review**. The episode moved to **Ready for POA**, with "HQ review completed by Cara Clinician on 27/10/2026, 22:21." |
| R-01 | Pass | In the Preview of "MT S44a stopbang" (Clinician view; sample patient male, 54; BMI 38), all Yes gives **STOP-Bang score: 8 (High risk)**. |
| R-02 | Pass | **Send my answers** stays disabled while a set is incomplete ("1 of 3", "2 of 3 sections complete" once the follow-up set appeared). Once every set was complete, the send worked and the episode went to Ready for review. The server's 409 for an incomplete send was checked in round 2 and wasn't repeated. |
| R-03 | Pass | Dee's Demo pre-op HQ send, low risk, was auto-triaged to **Ready for admission** by the rule "Low risk" and was on the **Auto-triaged** worklist. **Checked** showed "Checked by Cara Clinician on 27/10/2026, 22:23." and took it off the worklist. |
| R-04 | Pass | Asa's ASA III send gave the **Anaesthetic review** badge and the task "Book anaesthetic review" (ANAESTHETIST, "Added by rule: High risk"). The status card says Ready for admission needs the anaesthetist's ASA grade first. |
| R-05 | Pass | On T-04's episode, `POST …/complete-review` returned 409 "Validate every Question Set first: “Sleep follow-up” and “Smoking” aren't validated.". The **Complete HQ review** button is disabled in the UI. |
| R-06 | Pass | As Ada, the Audit page lists today's rows. **Export CSV** saved `~/Downloads/audit-2026-10-27-222533.csv`: 6,057 lines with the right header (Time, Who, Email, Action, Hospital, Patients, …), 290 of them naming this round's "MT S44a R3" patients. |

## Failures

None.

## Found while exploring

- **Note, already in pending.md (verification finding 2's product question):** on the patient's own screen, C ("How bad is it?" with "severe") stays visible after B is changed to No. A is hidden by then, but its stale value keeps C shown. On send, C is dropped, as the fix intends, so every staff view matches what was frozen. The patient saw an answer on screen that then isn't kept.
  - The patient's saved draft before sending was `{B: No, C: "severe"}`: A had already gone from the draft, and C was still there.
- **Note, not S44a:** in the STOP-Bang Preview, "Low risk: auto-triage to Ready for admission" fires with STOP-Bang **8 (High risk)**. This follows from the rule as written ("ASA I or II and no red flags"; a score's band isn't a flag, and the score gives no ASA grade). Rahul may want to know that a high STOP-Bang alone doesn't stop the default Low risk rule.
- Enter submits the patient's date-of-birth form.

## Console and network errors

- No console errors or warnings on the Validate, POA Summary or episode pages, which were reloaded with console tracking on.
- No failed API requests (all 200) on those pages.
- The only 4xx this round were the intended ones:
  - the refused changes to a published version (409 ×5);
  - R-05's 409;
  - T-20's 404s.

## GIFs

All are in `~/Downloads`:
- `S44a-r3-chain.gif`: main happy path. The patient's chain answers, B changed to No, the send, then Cara's Validate showing only B, and A and C empty after the correction.
- `S44a-r3-T08-stale-audience.gif`: the stale tab's refused audience change after Publish.
- `S44a-r3-T18.gif`: the suggested ASA changing on Validate, and after a reload.

## Hand checks for Rahul

- **The patient pages at 390×844.** The Chrome window can't be resized from here, so the chain, T-04, T-18, T-02 and R-03 patient flows ran at desktop width. Round 1 checked the thank-you page at 390×844 with Playwright.
