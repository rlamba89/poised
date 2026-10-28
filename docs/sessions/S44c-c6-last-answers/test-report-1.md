# S44c test report 1

**Summary: 5 passed, 1 failed, 15 blocked (round cut short).** The browser connection was lost partway: after a Claude in Chrome disconnect, the session was reattached to another connected browser ("Browser 2"), which gets ERR_CONNECTION_REFUSED for http://localhost:3000. The tool says not to choose between connected browsers without the user, so the rest of the round couldn't be run. The curl checks still showed the app up. The cases that were run are below. Everything else is listed as Blocked and needs a re-run once the session is pointed at Rahul's local Chrome again.

- Environment: `f143b17` with the S44c changes uncommitted (working tree dirty), 28 Oct 2026, 13:34 to about 14:10 GMT, http://localhost:3000 (`make dev`: API :8080, web :3000, evaluate :8090, `clinical@f143b177d60e-dirty`).
- Phone size: Chrome wouldn't make the window narrower than **500 px** wide (`innerWidth` 500 when 390×844 was asked for), so the "phone" cases ran at 500×701. Nothing scrolled sideways (`scrollWidth` = `innerWidth` = 500).
- Test data:
  - Built in the UI as Alex:
    - **MT S44c chain 1335** v1: published, Low risk rule deleted;
    - **MT S44c held 1355** v1: published, "wait for Checked" ticked, default rules;
    - **MT S44c stopbang 1400**: draft, with "Health" holding a BMI page and a STOP-Bang page.
  - Episodes on the chain (Hospital A1):
    - "Sam MT S44c T01" was made through the New episode page.
    - T02, T03, T12, T14, T15 and R01 were made with the same `POST …/episodes` call the page makes, run from the browser as Cara, to save time.
  - Invite links came from /qa/messages.
- **API reads used (said where used):** `GET …/episodes/{id}/hq`, run as staff, to confirm the stored patient rows after the screens changed.

## Results

| Case | Result | Note |
|---|---|---|
| T-01 The chain, phone size | **Pass** | Details below the table. |
| T-02 Leaving straight after a change | **Pass** | B→No, then All sections at once (no wait). A reload showed Pillows not listed and Breathing with B only. The stored rows matched. B Yes showed A empty; after A Yes, Pillows' D was empty. |
| T-03 Fill in for the patient | **Pass** | At desktop size: the same clearing. A reload, then Yes, brought A, C and D back empty. Answering D and pressing **← Episode** at once returned to the episode page, and D's answer was saved (rows checked). Nothing was refused. |
| T-04 On hold → Undo → Checked | Blocked | Browser lost before it was run. Ada's Hospital A1 setting wasn't changed, so there's nothing to clean up. |
| T-05 Archive → Un-archive → Checked | Blocked | Browser lost. |
| T-06 On hold → RFA by hand → Checked | Blocked | Browser lost. |
| T-07 STOP-Bang rule in words | **Pass** | Details below the table. |
| T-08 Test the whole HQ High/Low | Blocked (part checked) | Details below the table. |
| T-09 Real send, High stays Ready for review | Blocked | Browser lost before publishing MT S44c stopbang 1400 (it's still a draft). |
| T-10 HQ without STOP-Bang | Blocked (steps 1–2 **pass**) | Details below the table. |
| T-11 Sex Other → Incomplete | **Fail** (step 1); step 2 **pass**; step 3 blocked | Details below the table. |
| T-12 No "Patient answered" on unseen questions | Blocked | Episode T12 created, not yet answered. |
| T-13 Rename/delete a deleted set | Blocked | Browser lost. |
| T-14 Complete review, not waiting | Blocked | Episode T14 created, not yet answered. |
| T-15 Save after review completed elsewhere | Blocked | Episode T15 created, not yet answered. |
| R-01 Send keeps only what was shown | Blocked | Episode R01 created, not yet answered. Partly covered: T-01's send was B=No only. |
| R-02 Grade history | Blocked | |
| R-03 Low-risk text at once | Blocked | |
| R-04 ASA III task and flag | Blocked | |
| R-05 Pre-fill from an earlier HQ | Blocked | |

**T-01 in detail:**
- The list showed only Breathing ("0 of 1 sections complete").
- After B Yes, A Yes and C "Coughing", Pillows appeared. With D Yes: "2 of 2".
- Review, then B→No: A and C went at once, and the status read Saved.
- All sections: Pillows was gone ("1 of 1"). The stored rows were Breathing `{B: No}` and Pillows `{}`.
- After a reload: B was No, and A and C weren't shown. B Yes showed A empty; A Yes showed C empty.
- After C was filled and the set finished, Pillows was listed with **Start**, and D was empty.
- B back to No, then Send: "Thank you, Sam".
- No console errors.

**T-07 in detail:**
- The Rules list reads "When the suggested ASA is I or II and there are no red flags and STOP-Bang score isn't High risk: auto-triage to Ready for admission".
- Edit shows "and a score isn't in a band" ticked, with STOP-Bang score and High risk.
- The Publish window reads "This version auto-triages: ASA I–II, no red flags, STOP-Bang score not High risk → Ready for admission. …" (cancelled, not published).

**T-08 in detail:**
- Rules page, then Test the whole HQ:
  - Male, 60, 170 cm and 116 kg (BMI 40.1), all five STOP-Bang Yes: "Rules that fire: None".
  - Female, 30, 170 cm and 64 kg (BMI 22.1), all No: "Low risk: auto-triage to Ready for admission".
- So the rule behaviour is right.
- **Not checked:** the score value itself ("8 (High risk)" / "0 (Low risk)"). The Test the whole HQ page doesn't show it, and the browser was lost before the Preview's clinician view could be opened.

**T-10 in detail:**
- On MT S44c held 1355, which has no STOP-Bang:
  - Low risk reads "…and STOP-Bang score isn't High risk (not in this HQ): auto-triage to Ready for admission".
  - Edit shows the score as "STOP-Bang score (not in this HQ)", visually cut off (see Exploring).
- Publish went through with no problem, and its line reads "This version auto-triages: ASA I–II, no red flags → Ready for admission. … The patient's message goes when a clinician clicks Checked."
- Steps 3–4 are blocked.

**T-11 in detail:**
- Step 1 fails: the Sample patient's Sex offers only Female and Male (more under Failures).
- Step 2 passes: the STOP-Bang score card's settings end with "A patient whose sex is Not known or Other makes STOP-Bang Incomplete."
- Step 3 is blocked.

## Failures

### F1 (Minor): Sex "Other" can't be chosen for the sample patient, so T-11 step 1 can't be done as written

- **Where:** As Alex Author, on MT S44c stopbang 1400: Rules, then Test the whole HQ (`…/questionnaires/12eea10b-…/rules/test`).
- **Steps:** Open the Sample patient's **Sex** dropdown.
- **Expected (T-11 step 1):** you can choose sex **Other**, see STOP-Bang Incomplete, then switch to Female and get a number and a band.
- **Actual:**
  - The dropdown lists only **Female** and **Male**, so the new "Other → Incomplete" rule can't be tried before publishing. (The field can be cleared, which may stand in for "Not known", but that wasn't checked.)
  - The Preview's Sample patient has the same two options. I read this in `apps/web/src/features/preview/PreviewPage.tsx` (Sex `data` = Female, Male) to confirm what I saw, and didn't open it in the browser before the connection was lost.
  - `HqTestPage.tsx` line 167 matches what the UI showed.
- **Console and network:** none.
- An author can't preview the behaviour the score card's help text describes. The real-episode path (T-11 step 3: New episode offers Other and Not known) wasn't reached.

## Found while exploring

- **Cosmetic:** On the Rules page's Low risk edit, for an HQ without STOP-Bang, the score select shows "STOP-Bang score (not in t…". The value is "STOP-Bang score (not in this HQ)", but the input is 218 px wide for 264 px of text. A reader can't see "(not in this HQ)" without opening the list.
- **Not a fault, worth knowing:** a BMI Calculator page is locked. Every "Add content" button is greyed on that page, so "STOP-Bang in the same set as the BMI" needs a second page in that set.
- **Not a fault:**
  - A question title is committed to the canvas on blur, not as you type. The canvas kept "Yes / No question" until the text box lost focus.
  - A required follow-up left empty after "Yes again" stops **Finish this section** with "Response required." That's expected, and T-01 step 7's "Finish" needs C filled first.
- No console errors on the patient pages during T-01 to T-03. No unexpected 4xx or 5xx seen.

## Console and network errors not tied to a case

None seen.

## Hand checks for Rahul

None of the cases run needed one. The blocked cases need a re-run in a browser that can reach http://localhost:3000, rather than hand checks.

## GIFs

- `~/Downloads/S44c-T01.gif`: the T-01 happy path at phone size, from Continue to "Thank you, Sam".
