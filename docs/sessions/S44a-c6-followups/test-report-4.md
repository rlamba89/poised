# S44a test report 4 (retest after verification 2)

**11 passed, 0 failed, 0 blocked.** This round re-ran the "Retest after verification 2" cases at the top of the plan:
- the reorder with an autosave straight after it;
- two quick reorders;
- a reorder in a stale tab after Publish.

It also re-ran the short regression the plan asks for (T-04, T-08, T-17 step 2, R-02), plus R-05, T-12's thank-you page and a full Complete HQ review as happy paths.

- Commit `97e62e2`, working tree **dirty**: S44a's changes and the verification fixes are still uncommitted.
- 27 Oct 2026, 22:58–23:13 GMT.
- Local: web http://localhost:3000, API :8080, evaluate :8090 (`clinical@97e62e2e699f-dirty`).
- Tester: automatic run (Claude in Chrome). Rahul wasn't watching.

**How the test data was made.**
- **HQ "MT S44a R4 reorder 2300":** built as Alex in the editor and published.
  - It started with three Question Sets, used for the reorder cases.
  - It ended as T-04's HQ: **About you** (required "Do you snore at night?"), then **Sleep follow-up**, shown when that question is Yes. Sleep follow-up holds a required Yes/No "Do you stop breathing at night?" and a text "Tell us more".
  - The Low risk rule was deleted.
- **Draft HQ "MT S44a R4 T08 2315":** built as Alex and left as a draft. It has:
  - a patient set **Sleep**, with two Yes/No questions scored Yes 1 / No 0;
  - a Calculation **Sleep risk** adding up both, with bands Low ≤ 2 and High above, and "Also High when" the score ≥ 2 and Sex is Male;
  - a patient set **Sleep follow-up**, shown when Sleep risk band is High.
- **Episode "Rob MT S44a R4 T04 2310"** (`43e7dc62-…`): made as Cara on New episode (born 17/05/1980, male). The patient answered through the invite link from /qa/messages.
- **Signing in:** user switches used `POST /api/dev/login` from the page. This does the same as clicking a name on the stub login page.

**Calls made outside the UI:**
- **API calls from the signed-in browser:**
  - a concurrency check: reorders and content saves fired at the same moment;
  - the response of one reorder (see the note under Console and network);
  - permission checks on the reorder route.
- **One read-only database query:** the frozen `episode_answers` rows for T-04.

**Phone size.** The first tab's window still wouldn't resize from here. `resize_window` reported success, but the viewport stayed **500×757**, so the patient pages were checked at 500 px wide, close to phone width. A second tab opened at 1500×806 and was used for the staff pages. On the narrow tab, the editor was zoomed out with CSS (`zoom: 0.55`) to fit its two panels; nothing in the app was changed.

## Results

| Case | Result | Note |
|---|---|---|
| Reorder, then autosave straight after (new) | Pass | Question Set 2 was moved up with **Move up**. Straight after, its question was renamed "Do you snore loudly?": the content PUT went about 1 s after the reorder and returned 200. "All changes saved" showed and no error banner appeared. The same steps with **Move down** and "Do you snore at night?" also passed. After a reload, the order and the text were both kept. The sidebar reorders through the set's ⋯ menu (**Move up**/**Move down**); sets can't be dragged, only pages can. |
| Reorder twice quickly (new) | Pass | Question Set 3 was moved up twice in a row, opening its menu again straight after the first move. The server order is Set 3, Set 1, Set 2, which matches the sidebar. There was no error. |
| Concurrency check (API, new) | Pass | 15 rounds, each firing 3 requests together on one draft: a reorder, a content save of a set the reorder moves, and a second reorder. All 45 requests gave **204/200/204**: no 500 and no "deadlock". The order and the content were intact afterwards. A second race added Publish (whose body I left empty) and a second save with the same revision to the mix. It gave 204, 200, 409 stale (Publish), 204, and 409 stale (the duplicate save): still no 500. |
| Stale-tab reorder after Publish (new) | Pass | Tab B had the draft editor open, then tab A published V1. In tab B, **Move up** on Sleep follow-up gave a red banner, **"Only draft versions can be changed."**. The page refreshed to V1 · PUBLISHED, READ ONLY, and the order was unchanged. |
| T-04 | Pass | The patient answered Snore Yes, then the follow-up (Yes, "loud"), then went back and changed Snore to No. The follow-up set disappeared ("1 of 1 sections complete") and the send worked. Cara then checked the episode:<br>• **Episode page:** only About you is listed.<br>• **After correcting Snore to Yes and validating:** Sleep follow-up is listed again, **Not started**.<br>• **Sleep follow-up's Validate page:** both questions are empty, with no "Patient answered: Yes" or "loud".<br>• **POA Summary, Patient answers tab:** no answers from the follow-up.<br>• **Frozen patient row:** `{}` for Sleep follow-up. |
| T-08 | Pass | **Step 2:** Publish was blocked, with the Publish button disabled. It listed: "Sleep follow-up · “Sleep follow-up” shows only when Sleep risk band is High, but patients can't see Sleep risk. Make the score visible to patients, or move the condition to a clinician Question Set."<br>**Step 4:** with Sleep follow-up's audience set to Clinician, there was no problem and Publish was enabled.<br>**Step 3:** with the audience back to Patient and Clinical unticked on the score, there was no problem and Publish was enabled.<br>Each time the dialog was cancelled, and the HQ stays a draft. |
| T-17 step 2 | Pass | Under High, **Also High when…** was clicked, then "The score ≥ 2" was set, then **Add line**, then "Sex is Male". The editor showed "Also High when all of these hold:" with both lines, and "All changes saved". After a reload of the editor, both lines are still there. The operator menu shows symbols (=, ≠, >, ≥, <, ≤), not "is at least". |
| R-02 | Pass | **Send my answers** stayed disabled at "1 of 2 sections complete". With every shown set complete, the send worked and the episode went to **Ready for review**. |
| R-05 | Pass | While Sleep follow-up wasn't validated, **Complete HQ review** was disabled, with "Validate every Question Set to complete the HQ review.". |
| T-12 (happy path) | Pass | The thank-you page, at 500 px wide: "Thank you, Rob" and "Your answers have been sent to the hospital. We'll contact you about what happens next." |
| Complete HQ review (happy path) | Pass | Cara validated Sleep follow-up (No, plus a note), then clicked **Complete HQ review**. The episode moved to **Ready for POA**, showing "HQ review completed by Cara Clinician on 27/10/2026, 23:12.". |

## Failures

None.

## Found while exploring

- **Permissions on the reorder route** (API, T08 draft):
  - Cara (clinician): 403.
  - Sue (A1-only super clinician): 403.
  - Bea (Trust B): 404 "Hospital not found.".
  - Alex with a missing id, or with a duplicated id: 409 "The chapter list has changed. Reload and try again.".
  - Alex with a body that isn't JSON: 400.
- **Note, not a defect:** on Validate for a set the patient never saw (T-04's Sleep follow-up), answering a question as Cara adds a line under it: "Patient answered: no answer". That is accurate, because the patient saw nothing, but a reader could take it to mean the patient skipped the question. Mention it to Rahul if the wording matters.
- **Note:** the editor's sidebar has no drag for Question Sets, only **Move up**/**Move down** in the ⋯ menu. The plan's "drag Question Sets" was tested with the menu instead.

## Console and network errors

- No console errors or warnings on the episode, Validate or POA Summary pages; those pages were reloaded with tracking on.
- **The extension's network log is wrong for 204s.** Claude in Chrome's network log showed the editor's `PUT …/chapter-order` as **503**, every time. The same request made with `fetch` from the page returned **204**, and the order was applied on the server. The editor showed no error, and the log listed one of my own `fetch` 204s as 503 too. This is a fault in the tool, not the app.
- The only 4xx this round were the intended ones:
  - the stale reorder's 409;
  - the permission checks above;
  - the 409s in the race check.

## GIFs

Both are in `~/Downloads`:
- `S44a-r4-reorder-autosave.gif`: main happy path. Sets reordered from the menu, the question edited straight after each move, and two quick reorders.
- `S44a-r4-stale-reorder.gif`: a stale tab's reorder after Publish, refused with "Only draft versions can be changed.".

## Hand checks for Rahul

- **The patient pages at 390×844.** The Chrome window can't be resized from here, so the patient flow ran at 500 px wide. Round 1 checked the thank-you page at 390×844 with Playwright.
