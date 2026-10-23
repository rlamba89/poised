# Pending items

State on 1 Oct 2026. Plans: [plan-redesign.md](plan-redesign.md), [plan-features.md](plan-features.md) and, next, [plan-workflow.md](plan-workflow.md) (the episode workflow, which comes before the items below). Requirement IDs refer to [requirements.md](requirements.md).

## 1. Blocking now

Nothing. The episode workflow ([plan-workflow.md](plan-workflow.md)) is built and committed.

**After a full disk:** if Postgres won't start and logs `bogus data in lock file`, remove the stale lock and recreate the container. The data is safe in its volume:

```sh
docker run --rm -v sj-demo_dbdata:/data alpine rm /data/postmaster.pid
docker compose up -d --force-recreate db
```

## 1a. Episode workflow: next items

- **POA Summary sections not built yet:**
  - Audit log and Work plan
  - comments per section
  - Observations, Investigation results, Assessments
  - Files, and Send to EPR
  - hospital-reported BMI
  - a suggested ASA from disclosures
- **Statuses:** recovery (C9) is not built yet. *Admitted, discharged and cancelled: S20. Archived: S22.*
- **Patients:**
  - accounts, and invitations by email or SMS
  - patient search
  - Short HQ
- ~~**Re-opening:** a review can't be re-opened after it is complete.~~ *Built in S20 (super clinicians, with a reason).*
- ~~**Two simultaneous episode PATCHes can overwrite each other's ASA grade**~~ *Fixed in S20 by the episode revision, with the related wrong-status 409 below.* (found by S02a's verification, 9 Oct; older than S02a).
  - S02a's guard covers only the status. The handler still fills the fields a PATCH didn't send from the row it loaded, so of two PATCHes in the same milliseconds (Nurse ASA and Anaesthetist ASA), the last empties the other's grade.
  - **Fix:** an UPDATE that sets only the fields sent (`sqlc.narg` with `coalesce`), or a revision on the episode row. Details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md) item 2.
  - **Related (S02a verification 2, finding 5):** the PATCH 409 says "This episode changed since you opened it", but it compares with the status the request loaded a moment earlier, not the one the user's page loaded. A stale POA Summary page isn't detected. A revision on the episode row would fix both, and make the message true.
- **A save whose response is lost can make the tab conflict with itself** (found by S02a's verification, 9 Oct).
  - **How it happens:** the server commits an answers save, but the browser gets a network error. The saver keeps the old revision, so the tab's next save gets a 409, and the user sees "changed somewhere else" though no other tab exists.
  - **Not silent:** the message and Reload show, and Reload shows what was saved.
  - **Fix when it matters:** after a failed save, read the set's current revision and data, and carry on if the server already has this tab's answers. Details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md) item 3.
- **An answers save's 409 can give the wrong reason** (found by S02a's verification, 9 Oct; low).
  - **How it happens:** if the patient submits on another device, or another clinician completes the review, while a save is in flight, that save says "changed somewhere else" instead of "already sent" or "review complete".
  - **Why it's low:** it's still a 409, and Reload shows the truth.
  - **Where:** `patient.go` and `validation.go` pick the message from the episode as loaded at the start of the request.
- **Before C1: after a panic, the recover middleware sends headers the handler had already set** (found by S02's verification, 8 Oct).
  - `recoverPanics` writes the 500 without clearing `w.Header()`, so a cookie set before a panic would go out with the error.
  - No handler can do that today: both cookie handlers write straight after `SetCookie`. Fix it test-first before the Cognito session code (C1). Details in [verify.md](sessions/S02-workflow-route-tests/verify.md) item 3.
- **Complete review isn't checked on the server** (found by S02, 8 Oct). `POST …/complete-review` succeeds while the episode is Ready for review, even if no Question Set is validated. Only the browser disables the button until every visible set is validated. Enforce it on the server in C6, where the `evaluate` Lambda can work out which sets are visible.

## 2. Waiting on input

| Item | Detail |
| --- | --- |
| Real HJE Full HQ export | Today's copy has only Question Sets 1–4, transcribed from screenshots, with no disclosures. Run `tools/export-hq.js` in the incumbent's authoring tool and import the file with `go run ./cmd/import-hq` to get all 11 sets and their disclosure codes (EXP-04). |
| Saved option lists: copied or linked? | Built as copies: a question keeps its options when the list changes, which protects stable IDs (OPT-02). Confirm that's what's wanted. |

## 3. Small gaps in what's built

### From S25's verification (23 Oct; [verify.md](sessions/S25-prefill/verify.md), minor)

- **For S26: a pre-filled row has revision 0, the same as "no row yet"**, so a tab loaded before a set was pre-filled can overwrite it on its first save without a 409 (`SavePatientAnswers`). The patient's typed answers win, but the `prefill` map then describes answers no longer in the row, so "changed by patient" would mislead. Give pre-filled rows their own starting revision, or refuse revision 0 against an existing row.
- **For S26: a repeating-list row whose only non-empty cells are in columns the new list lacks is dropped** (`prefill.fitRows`). It can't happen within one trust (S23's lock, `SameWording`). For other trusts, refuse the whole answer instead ("if in doubt, don't fill").
- **For S26: a Question Set hidden by a set condition is pre-filled too** and sent as it is; the clinician's view and the POA Summary skip hidden sets. The clinician's markers should mark only shown sets and questions.
- **A second, different card button (another tab) returns 204** and that tab carries on as if its choice were recorded; Go keeps the first. Return the stored choice so the page follows it.
- **Small:** `awaitsCheck` builds a second SurveyJS model per pre-filled set on each list render; the "started" rule is copied in `accounts.sql` and `jobs.sql` (one SQL function would do); `showPrefillMarkers`' clean-up is never called (harmless: the model is built once per set).

### From S25's build (23 Oct)

- **Before demoing pre-fill on QA:** QA's Neon database has neither the library-based Demo pre-op HQ (above) nor the seeded **Demo follow-up HQ**. The seed adds the follow-up HQ on the next `make seed` there, but it shares keys with the new Demo HQ only.
- **Pre-fill reads every earlier submitted form's Question Sets in full on each first open** (`ListPrefillSources`), and repeats until each set has a row. Fine at today's sizes. If a patient ever has many long HQs, cache or limit to the newest few forms.
- **`ListPrefillSources` adds one more copy of the HQ-form join** (`f.kind = 'hq'`). The `episode_hq` view idea (S18 verification, item 5) still stands for C7/C9.
- **A hidden pre-filled answer is sent if the patient sends without finishing that section** (as a hidden typed answer is). Outputs and the POA Summary skip hidden questions, but S26's clinician markers should only mark questions shown.
- **The patient's details box shows "Sex female" in lower case** (S25 test report 1, found 2): the profile block's template prints `{patientSex}` as stored. Older than S25; capitalise it, or map it to a label, in the template.
- **The audit journey's tooltip check flakes** (`test_audit.py::test_a_list_row_names_ten_patients_and_counts_them_all`, seen in S25's e2e runs): the hover's tooltip sometimes doesn't appear; it passes on its own.
- **The hazard log (OPS-04, C10)** should cover: the newest-form rule, wording matches, "Everything is still correct" skipping the patient's re-reading, and the recorded button.

### From S24's build (22 Oct)

- **QA's Neon database keeps its old Demo pre-op HQ** (q_smokes Yes/No…): the seed never changes a published demo HQ. Before demoing S25's pre-fill on QA, reset QA's demo HQs (or publish a library-based new version by hand).
- **From S24's test report 1:** a "Same as…" candidate refused for two reasons (its ID already in this HQ, and another kind of answer) shows only the first; and the seeded Demo pre-op HQ asks "Is there any chance you could be pregnant?" of every patient, men included (no condition on sex; authors add one with Logic). Neither changed.
- **From S24's test report 2 (Rahul to judge; no change):**
  - **"Same as…" is offered on a question that already has a key from the library or a published HQ,** and Use this replaces that key (the question stops being linked to its old one). Allowed in a draft, as any key change is; a warning ("This question is already linked to …") could be added if authors trip over it.
  - **A library question may have no text** (the API accepts an empty title; the row then shows its ID). Harmless; the editor's own questions always have text unless the author clears it.
  - **The local database's HJE Full HQ description names the incumbent** (local data only, not in the repo); Rahul may want to edit it.
  - The read-only editor still shows "Add content or drag and drop…" in an empty Group or Section (not S24).
- ~~"Same as…" drops the question's own disclosures and doesn't map conditions on its old option values (the logic-problem button flags them).~~ *Corrected 23 Oct (S24 verification 1, finding 1): that check didn't exist. Now conditions are mapped to the new option with the same text, the logic-problem check (and Publish) flags a condition testing an answer a choice question doesn't have, and the Same as… window says what will be lost before Use this. The question's own disclosures are still dropped, by design (said in the window).*
- **From S24's test report 3 (21 of 21 passed; cosmetic, not changed, recorded at the end of the run's test rounds):**
  - The Same as… row's **Use this** button is squeezed to a sliver when the "In …" list of HQ names is long (e.g. a key published in ~40 HQs). Fix: `wrap="nowrap"` on the button's side / `flexShrink: 0`, and cap the list ("In Demo pre-op HQ and 39 others").
  - The **Save to library** window reopens on "Saved." for the same question; it should start afresh each time, like the other windows (reset on opening, as `SameAsWindow` does).
  - A condition left testing a removed answer shows the raw value on the card ("Displays when … is o_glyj"); "(a removed answer)" would read better. The logic problem already flags it.
  - Not S24: the editor's page header says "Clinical Page" on a patient Question Set's page (`Canvas.tsx`, since 1 Oct); an expired staff session (12 hours) shows only as "Please sign in." inside the window that made the request.
- **From S24's test report 4 (21 of 21 passed; the last round, no code changed):**
  - **⌘Z after "Same as…" restores only the current Question Set:** another set's Question Set condition that the adoption renamed keeps the new key (e.g. `{ponv_history} = 'o_qunt'`), which is no longer in the HQ. Not silent: Publish refuses it ("…it tests something that isn't in an earlier Question Set (ponv_history)"), and the author fixes it by hand. A fix would make the editor's undo span the sets one adoption changed (one undo step across chapters), or rename them back on undo.
  - **Save to library reopens on "Saved." for the same question** even after selecting another question and coming back (report 3 saw it reset); a reload resets it. Already noted above (report 3); the fix is to reset on opening.
  - The "fresh if any copy is fresh" rule shows "Always ask fresh, like the original" on keys the automated tests' HQs mark fresh (e.g. `q_smokes`): as designed.
- **From S24's verification 1 (minor, not changed):** a library repeating group keeps its row questions' own conditions (they name siblings in the group, so they still work); `keySources` repeats `publishedQuestions`' newest-version walk; Same as… walks every published version of the trust on each search (fine at today's sizes, as S23's route).
- **From S24's verification 3 (23 Oct; minor, none silent, not changed):**
  - **The unknown-answer checks read expressions with regexes:** `unknownAnswers` (`logic.ts`) compares case-sensitively (a hand-written `{q} = 'Yes'` against `yes` works in SurveyJS but is refused at Publish) and doesn't judge a reversed `'v' = {q}` or a double-quoted value (only the Code tab writes those); `adoptionLosses` and `rewriteReferences` have their own regexes. Move all three onto SurveyJS's `ConditionsParser`, as the rest of `logic.ts` does.
  - **Saving a repeating group to the library checks only its own key** (`library.go` `saveToLibrary`): its row questions' keys aren't checked against the library or the trust's published questions, so the library can hold a row key twice. An HQ using it is still refused at Publish by S23's lock.
  - **A saved question keeps its option conditions, and a repeating group its rows' conditions** (only the top-level ones are dropped). Publish refuses one that names the source HQ's questions.
  - **`keys.Compatible` passes any choice question, but `adoptKey` copies options only from a `choices` array:** a published question with `choicesFromQuestion` would give its key without its options (refused at Publish; no such question today).
  - Small: a Yes/No question adopting two other options stays labelled "Yes / No"; the seed doesn't check a new platform key against the trusts' own library items (only matters if `library.json` gains a key); a punctuation-only Same as… search lists every candidate; `renameIn` repeats `doc.ts`'s rewrite loop.

### From S23's test report 1 (22 Oct; all passed)

- **A medication question's type isn't frozen yet** (S23 verification 3): a published medication key can switch Prescribed → Recreational and publish. **Carried into S24's brief as its first build item**, so it's fixed before S25 copies answers by key.
- **The editor's badges can go stale until the next content save** (S23 verification 3, from `/code-review`): changing a Question Set's audience, or adding or deleting a set, doesn't re-fetch `question-keys`; the editor also fetches it twice as it opens. The Publish window always re-fetches, so publishing isn't affected.
- **If the re-fetch after a 422 fails, the Publish window shows "ready" again** (S23 verification 3): the refusal is cleared and Publish is enabled; clicking gives the same 422. Go still refuses, so nothing wrong is published. Rare (the GET has to fail).
- **`keyPlace` walks the whole document for each card on every render** (S23 verification 3): fine at today's sizes; pass the place down through `ElementList` if a big set feels slow.
- ~~**The editor's pre-fill badge isn't reachable from the keyboard**~~ *Fixed after test report 3: it's a button; Escape closes its list.*
- ~~**`GET …/question-keys` can be large**~~ *Fixed after verification 1: the route no longer sends the published options; Go returns the key problems instead.* It still walks every published chapter of the trust on each call (once per save); add a key table if that gets slow (S23 answers, Q1).
- **A key-linked question whose options were changed in the draft still shows "Pre-fills from 1 other HQ · by link"**; only the Publish window warns. Publish blocks it, so nothing reaches a patient (5).
- **Not S23:** a question in a clinician Question Set still shows the grey PATIENT chip on its editor card (7).
- **The phone format's refusal shows its raw pattern** (test report 5, found 3): "…changed from any text to the pattern “^\+?[0-9 ()-]{7,20}$”". Go names only "email address"; naming the editor's other formats ("phone number"…) in Go would copy `fields.ts`'s TEXT_FORMATS, some of them country-specific (A-20). Cosmetic; Rahul to judge.
- **Not S23 (test report 5, found 4):** the editor's Text Format list offers "NHS number" and "UK postcode" (`fields.ts` TEXT_FORMATS), country-specific options outside the config (A-20). They predate S23.
- **Not S23 (test report 3, found 6):** the delete dialog for any question says "… will be deleted, with its options and disclosures", even for Text or Number questions, which have no options.
- **A race older than S23 (verification 1, item 8):** a chapter save that has passed its draft check can commit between Publish's key checks and the publish itself, so a just-saved change could be published unchecked. Closing it needs the save to take the trust's lock too (or re-check the status under it).
- ~~**Hand checks for Rahul:** R-05 and R-02 at 390×844~~ *Both passed in test reports 3 and 4 (R-05 after the old "S02 test HQ" data was removed).*

### From S22's verification (21 Oct; none blocking, details in [verify.md](sessions/S22-archive/verify.md))

- **For Rahul: a patient's sent questionnaire loses its View once archived** (normally 30 days after discharge): the home shows "Closed by the hospital" and the form answers 403. Recommended: show an archived-after-sent questionnaire as Sent with a read-only View; or keep it closed if the patient's copy should end with the episode.
- **Closing an account can deadlock with an archive or cancel of one of its episodes in the same instant** (opposite lock order: `closeAccount` locks `jobs` then the episode; `applyStatus` and the auto-archive the reverse). One side gets a clean rollback and a 500. Fix: lock the records' episodes first in `closeAccount`.
- **A patient autosave racing an archive can still be saved** (`SavePatientAnswers` checks only that the form is open). Add `status <> 'archived'` to its SQL.
- **`RunDue` returns before the follow-ups if adding an auto-archive job fails;** log and carry on instead.
- **An episode archived from Cancelled:** the patient's home says "Cancelled by the hospital", an old link's form says "This questionnaire is closed". Use the cancelled message for both.

### From S22's test report 1 (21 Oct)

- **Archived rows still say "1 open task"** in the episode list (with Show archived) and on the episode's Tasks title. The tasks can't be changed while it's archived, so it may read as work outstanding. Cosmetic; Rahul to judge (hide the count on archived rows, or keep it as a record).
- **General notes look out of order after a future-dated run, then a real-time change** (test report 3): e.g. an auto-archive dated 31 days on by a /qa/jobs run, then an un-archive dated now; notes sort by their time. Testers' tool only.
- **An archived episode still shows the red NOT RESPONDING badge** beside ARCHIVED (test report 3). Harmless (the flag is history); Rahul to judge whether to hide it once archived.
- **/qa/messages shows a reminder's real send time, while its General note is dated with the "as of" run's time** (test report 2). Testers' tool only (the message log is capture mode's); on a real clock they're the same moment.
- **A run of /qa/jobs "as of" a later time dates what it writes as of then** (fixed in S22, F-2): its notes and the activity they mean are in that future, so the episode's General notes can list them after later real comments, and a simulated episode won't auto-archive until 6 months after the simulated time. Testers' tool only; on AWS and locally the run's time is the real time.

### From S21's verification (21 Oct; none blocking, details in [verify.md](sessions/S21-tasks-worklists/verify.md))

- **For Rahul: the default "Cancelled" message assumes an open HQ** ("…so you don't need to fill it in"), but Cancel is allowed from any status and sends it. Recommended: a neutral default ("…has cancelled your pre-operative assessment…"); or send it only while the HQ is open. Off by default and editable, so nothing is wrong today (`settings.go`, `episode.StatusMessage`).
- **Ready for admission is sent again** each time the status is picked again (On hold → Ready for admission, or Undo then pick). A reasonable reading of answer 4; say if once per episode is wanted.
- **The "… message sent" General notes line uses the request's context** while the send doesn't (`statuses.go` `statusMessage` → `s.event`): a browser disconnect mid-send could lose the note (the `episode_messages` row stays). One-line fix: write the note with the uncancelled context.
- **`CountWorklists` skips `ListEpisodes`' inner joins** (patients, HQ form, version, questionnaire). Equal today, because every episode has its HQ form (S18); keep the two in step if that ever changes (`episodes.sql`).
- **Status email subjects are fixed in Go** (`invite.SendStatus`); a third status message would need its own subject.

### From S21's test report 1 (21 Oct; all passed, no code change, as the conductor asked)

- **General notes dates are ISO, not the locale** (older than S21: S19's and S20's lines): "Planned procedure date set to 2026-10-28", "Status changed to Admitted (admission date 2026-10-21)". The notes are stored text, so they'd need the date formatted when shown (or stored as a value beside the text).
- **"← Episodes" on the episode page goes to All, not the worklist the user came from** (browser Back does return to it). Could carry `?worklist=` (and `?hospitals=all`) through the episode link, or go back in history (`EpisodePage.tsx`).
- **The worklist tabs' help is a native `title` tooltip** (slow to show, not reliably read by screen readers), and **All** has none. The selected tab's help also shows as a line under the tabs. A Mantine `Tooltip` or a visible line for every tab would be better (`EpisodeList.tsx`).
- **A cancelled episode's Tasks title still counts its open tasks** ("(1 open)"), which can never be done now (accurate, read-only). Could say "(1 left open)" or hide the count once cancelled (`TasksPanel.tsx`).

### From S20's test report 1 (21 Oct)

- **An Undo form open in a stale tab retargets after Reload** (test report 2): its title changes to the new step back ("back to POA complete") and the reason stays, instead of closing as a no-longer-offered move does. Safe: nothing is sent without another click, and the title says what will happen. Could close too if the target changed (`StatusControl.tsx`, the `offered` check).
- **POA Summary at 500 px** (test report 2, cosmetic, older than S20): when the two ASA grade fields wrap, "Anaesthetist ASA grade" sits about 20 px further right than "Nurse ASA grade" (its `ml="lg"` in `PoaSummary.tsx`).
- **Staff's on-behalf form for a cancelled episode shows the patient's wording** ("…Please contact the hospital if you have any questions."), from the shared refusal in `patient.go` (`formCancelled`). Cosmetic: staff see the episode is cancelled.

- **Disclosures on "Other (please specify)":** not possible yet.
- **Question Set conditions:**
  - The one-set preview can't see answers from earlier sets. It explains the condition but decides it only when it tests just the patient.
  - There is no whole-questionnaire preview yet.
- **Questionnaire list:** no status filter or sorting (FRM-02). Search and paging work.
- **Found by the S01 manual test, 8 Oct** (all older than S01; details in [test-report-1.md](sessions/S01-integration-harness/test-report-1.md)):
  - **The delete dialog shows the previous error.** After a refused delete ("Only drafts can be deleted…"), opening Delete on another questionnaire still shows that message. The delete itself works.
  - **The list API's `total` is 0 on a page past the last one,** and every item carries a stray `total`. Both come from `count(*) OVER ()` in `ListQuestionnaires`.
  - **The access-error pages are dead ends.** "You don't have access to this hospital." and "Questionnaire not found." have no header, no Sign out and no way back.
  - **"chapter" shows in user-facing text:** the two-editor conflict message says "Someone else changed this chapter…" under a "Question Set" title.
  - **Small:**
    - Enter doesn't submit "Save these options as a list".
    - A saved list is deleted with one click, with no confirmation.
    - The viewer's ⋯ → View uses the pencil icon.
    - The State badge truncates to "D…" below about 760 px.
- **Found by the S02 manual test, 8 Oct** (all older than S02; details in [test-report-1.md](sessions/S02-workflow-route-tests/test-report-1.md)):
  - **A double-click on "Save comment" saves the note twice.** Notes are append-only, so the duplicate can't be removed, and it shows on the POA Summary. The button should be disabled while it saves.
  - ~~**The patient link's response (`GET /api/p/{token}`) includes the patient's last name, date of birth and sex,** though the page shows only the first name. **Decide before C2 (S14):** if patients sign in with magic link + date of birth, this response must not include the date of birth, or anyone holding the link can read the answer to the check.~~ *Done in S14 (13 Oct): the link alone returns only the hospital's name; after the date of birth, the HQ has the first name, full name, age and sex, never the date of birth or an identifier.*
  - **Sign out sometimes seemed not to sign out** (2 of about 10, on refusal pages reached by pasting an address). Unconfirmed; it may have been the testing tool's timing.
  - **Episode refusal pages:** the hospital picker is blank on "You don't have access to this hospital." (The questionnaire refusal pages have no header at all; see S01's item above.)
  - **"No episodes yet." shows when a status filter matches nothing,** though episodes exist.
  - **A PATCH with an unknown episode status gets 409** ("can't be changed to that now"), where 400 would be more accurate.
  - **The ASA grade controls work before the HQ review is complete,** while Episode status is locked. Check whether that's wanted.
  - **Small:** the patient's set page says "Saved" before any answer; the delete dialog's text flickers to `""` as it closes.
  - **From the retest (report 2):**
    - **A patient's section counts as complete once every required question is answered,** even without "Finish this section". Probably by design; confirm.
    - **"Send your answers?" can reappear over a section** if a click lands behind it as it animates in. Confirming it still submits correctly.
    - **The staff header wraps at narrow widths** (about 500 px), crowding the back link. Staff pages aren't phone-targeted.
- **Found by S02a's second verification, 11 Oct** (details in [S02a verify.md](sessions/S02a-answer-revisions/verify.md), "Verification 2"):
  - **Network errors show technical text** (finding 2, older than S02a): a failed fetch shows "TypeError: Failed to fetch" (patient banner) or "Failed to fetch" (staff). `apps/web/src/lib/api.ts` could turn a network failure into a plain message, such as "We couldn't reach the server. Check your connection." The patient page already says **Not saved** next to the section.
  - **If Reload fails on the clinician's set page, the page is stuck on the error** (finding 3, new in S02a): `ValidateSet.tsx` doesn't wait for or catch the reload, and the error from `useEpisode` / `useEpisodeHQ` is never cleared. Refreshing the browser recovers.
- **Found by the S02a manual test, 9 Oct** (older than S02a; [test-report-1.md](sessions/S02a-answer-revisions/test-report-1.md) item 3):
  - **The editor says "All changes saved" while a title edit is still in the focused field.** It saves only when the field loses focus. *Seen again in S10 test report 1: a typed question text was lost on reload without leaving the field.* So a stale editor tab finds out about a conflict only then, and an edit can look saved when it isn't.
- **Found by the S03 manual test, 12 Oct** (all older than S03; details in [test-report-1.md](sessions/S03-e2e-journeys/test-report-1.md), "Found while exploring"):
  - **A "comes later" logic problem opens an empty-looking condition.** Clicking the problem opens the Logic tab with a blank "Select a question…" picker, because the tested question isn't in the earlier-questions list, while the card still says "Displays when Do you smoke? is Yes". The author isn't shown the cause.
  - **A question in a Clinician Question Set is badged PATIENT** on its card ("Blood pressure" in Nurse checks).
  - **Undo can't be reached while a question is selected:** the Settings panel replaces the Structure panel, where Undo lives.
  - **The New episode "Health questionnaire" dropdown can't be searched.** Each `make e2e` run publishes four more `E2E …` HQs in Hospital A, so the list grows; rebuild the local database when it's cluttered. C2 rebuilds this page.
  - **Clicks straight after a page loads were sometimes ignored** (3 times, in dev mode). Possibly the testing tool's timing; unconfirmed.
- **Found by the S04 manual tests, 12 Oct** (all older than S04; details in [S04's build notes](sessions/S04-react-vite/build-notes.md). The "chapter" text, the dead-end refusal pages, the blank hospital picker and the delete dialog's flicker are listed above already):
  - **The POA Summary lists a clinician's change to a question that a later change hid.** E.g. the clinician changes "Do you vape?", then sets smoking to No, which hides it; the summary still says "Do you vape? Patient answered “No”, now “Yes”". Answers to hidden questions are kept (`packages/clinical`).
  - **An empty Question Set (no pages) can be published.** The patient sees it as Complete, and the validate screen shows SurveyJS's "The survey doesn't contain any visible elements."
  - **A non-clinician sees "New episode"** above "Only clinicians can see patients and episodes."
  - **The printed POA Summary would include screen-only controls:** Add comment, the two tabs and the section chevrons (the print CSS hides only `.sj-no-print` and the header). Rahul to click Print once to check.
  - **Enter doesn't save a test case** in the preview's Test cases box; you must click **Save answers** (S04 report 3).
- **From S04's verification, 12 Oct** (minor; details in [S04 verify.md](sessions/S04-react-vite/verify.md)):
  - **The editor writes `?set=`/`&page=` with `window.history.replaceState`,** behind React Router's back (F-2). It works (it keeps the router's history state), but the router's own `location.search` goes stale. Use `useSearchParams` + `setSearchParams(…, { replace: true, preventScrollReset: true })` when `QuestionnaireEditor.tsx` is next touched.
  - **Leftovers from Next.js** (F-4, cosmetic): `PatientHQ`, `PoaSummary`, `ValidateSet` and `PreviewPage` keep default exports that only `next/dynamic` needed; the pages read params with `useParams() as {…}`, so a renamed `:param` would compile and pass `undefined`.
  - **No "loading" cue when a page is opened for the first time** (F-7). The old page stays until the new page's code arrives (0.34 MB gzipped for the SurveyJS pages). Add a thin progress bar (`useNavigation().state`) before patients use it on phones.
  - **Before the first deploy (S06), harden the reload-after-deploy screen** (S04 verification 2):
    - ~~`reloadOnce` (`apps/web/src/lib/chunkError.ts`) allows one reload per 10 seconds, so on a connection where each failed load takes longer than that, it reloads forever. Clear the flag only after a page renders, or widen the window.~~ *Done in S06: no time window; `Root` clears the flag once a page renders.*
    - ~~Keep `/assets/*` out of Amplify's SPA rewrite, so a missing file is a real 404. Otherwise Safari's "'text/html' is not a valid JavaScript MIME type" isn't recognised. Alternatively, add Vite's `vite:preloadError` listener as a backstop.~~ *Done in S06: the SPA rule (`infra/web_stack.py`) leaves files and `/assets/…` alone; `infra/tests` checks it.*
  - ~~**The error screen's "Go to the start page" sends a patient to the staff sign-in** (`RouteError.tsx`; `/` → `/login` when not signed in). Hide it on `/p/…` with S14/S15.~~ *Done in S15: on a patient's page it's "Go to your questionnaires" (`/home`).*
  - **Small:**
    - the staff `HospitalLayout` is in the first-load bundle, so the patient's link downloads it (make it lazy);
    - the Vite proxy key `/api` also matches `/api-…` (use `^/api/`);
    - in dev, StrictMode flashes "Something went wrong" just before the automatic reload.
- **From S05's verification, 12 Oct** (minor; details in [S05 verify.md](sessions/S05-lambda-entrypoints/verify.md)):
  - **Answer dates in notes, outputs and the POA Summary are still day/month/year** (`packages/clinical/src/outputs.ts` `formatDate`, `yyyy-MM-dd` → `dd/MM/yyyy`). The guard test can't see it (no locale literal). Fine while only the UK runs; **Rahul to decide** before a second country whether `packages/clinical` takes the locale (recommended: pass it in, as `format.ts` does).
  - ~~**`DEFAULT_LOCALE` is checked only for shape,** and some well-shaped tags (`en-12`, `en-gb-gb`) make the browser's `Intl.DateTimeFormat` throw, so every date on a page would crash it. Tighten the region part to `[A-Z]{2}` or three digits, or check the tag in `infra/countries.py` (S06).~~ *Done in S06: `countries.py` accepts only `language-REGION` (e.g. `en-CA`); the Go check is unchanged.*
  - ~~**The first render waits for `GET /api/config` with no timeout** (`apps/web/src/main.tsx`). A hung request leaves a blank page. Add a timeout (e.g. `AbortSignal.timeout(5000)`) before S06 (Lambda cold starts).~~ *Done in S06: `loadConfig` gives up after 5 seconds.*
  - **Small:** `APP_URL=/` trims to empty and passes strict mode (validate after trimming); the literal guard skips lines starting with `*` and reads only `.go`/`.ts`/`.tsx` files.
- **From S08's manual tests, 12 Oct** (minor; details in [S08 test-report-2.md](sessions/S08-trusts-permissions/test-report-2.md)):
  - **Cosmetic, older than S08: the editor's Disclosures dialog shows "No disclosures yet." while it fades out** after **Done** (report 4). The same fix as the other dialogs (keep the last content while closing) applies in `apps/web/src/features/editor/DisclosureModal.tsx`.
  - **Cosmetic: a long test-case name squeezes its PASS/FAIL and viewer badges** to "P.." in the Preview's Test cases list (report 4). The badges should keep their width (`flexShrink: 0`) and only the name be truncated (`features/preview/TestCases.tsx`).
  - **Keyboard: Enter in the Preview's test-case name box doesn't save**; only clicking **Save answers** does (report 4). Wrap the name box and button in a form.
  - **Staff pages aren't laid out for phones.** The header now scrolls sideways inside itself below about 768px (fixed in S08), but the Episodes table and other staff screens are still wider than a phone. Nothing requires staff phone layouts yet.
- **From S08's verification, 13 Oct** (minor; details in [S08 verify.md](sessions/S08-trusts-permissions/verify.md)):
  - ~~**For S10: `GET /api/me` leaves out a trust with no hospitals yet**~~ *Fixed in S10 (13 Oct): it's listed with `hospitals: []`, and the start page explains.*
  - **Once hospital-owned content exists:** the option-list buttons in the editor don't use `canChange` (a hospital-only super clinician on their hospital's own HQ would be offered "Save as a list" and get 403); `DraftNameExists` checks the whole trust, including hospital-owned drafts the caller can't see; and the refusal message always says "of the whole trust".
  - **`createVersion` checks the draft name without `LockOrgQuestionnaires`** (older than S08; the check is now trust-wide).
  - **`memberships` has no primary key** after migration 00006 (only `UNIQUE NULLS NOT DISTINCT`). Add a surrogate `id` before any logical replication (Neon → RDS, C10).
  - **Migration 00006's Down** fails on a questionnaire owned by a trust with no hospitals (none exist today).
- **From S09's build, 13 Oct** (minor; details in [S09 build-notes.md](sessions/S09-staff-invites/build-notes.md)):
  - **Every `make e2e` run adds two or three "Zoe E2E …" accounts to the dev sign-in list.** Their access is removed at the end of each journey, but accounts are never deleted (past actions keep their author). Reset the local database if the list gets long.
  - **A user's screen doesn't change until they reload** after an admin changes their role: what the server allows changes on their next request, but buttons (e.g. New Questionnaire) come from `/api/me`, read once per hospital page load. (Seen again in S09 report 2: after losing whole-trust access, the "Hospital | All hospitals" switch stayed until a reload.)
- **From S09's manual tests, 13 Oct** (minor or cosmetic; details in [S09 test-report-2.md](sessions/S09-staff-invites/test-report-2.md), "Found while exploring"):
  - **The Remove dialog can say someone loses a hospital they keep.** Removing "Hospital A1 · Clinician" from someone who also has the whole trust says "They lose access to Hospital A1 at once. Their other access stays.", but the whole-trust access still reaches A1. The dialog should say they keep A1 through their whole-trust access (`RemoveForm` in `apps/web/src/features/staff/StaffPage.tsx`). Also "access at the whole trust" should read "access to the whole trust" in its title.
  - **A change to your own access always sends you to the hospital picker with no message,** even **Add access**, which can't reduce what you can do (`done` in `StaffPage.tsx` navigates to `/` for any change to yourself). Only changes that may drop your whole-trust admin need to.
  - **Inviting someone already on this trust's staff to another access** says "…already had an account, so they were given access to… No new account was made." That's the wording for someone from another trust; for an existing member, "now also has access to…" (as Add access says) would be clearer.
- **From S09's verification, 13 Oct** (minor; details in [S09 verify.md](sessions/S09-staff-invites/verify.md)):
  - **The last-admin guard counts whole-trust admins who have never signed in,** even with an expired invite (`CountOtherTrustAdmins` in `apps/api/db/queries/staff.sql`). So the only working admin can step down in favour of someone who never signs in. S10's superadmin can recover a trust; decide whether "another admin" should mean one who has signed in.
  - **Invite state is per account, across trusts.** A new invite from trust A to an account that never signed in reopens sign-in for it, and the first sign-in accepts trust B's expired invite too, so the B access B granted becomes usable. This follows the agreed rule (access starts at the invite; 7 days is the first sign-in's deadline). The staff list's "Invited · expires …" can also show another trust's deadline (the invite join in `ListStaff` isn't limited to the trust). Revisit with S12 (Cognito).
  - **Inviting an email shows whether it has an account anywhere,** and the name another trust gave it (`existingAccount` and the stored name in the 201, and the 409 text). It's inherent in STF-02's "never a second account", but it does tell an admin about other tenants' staff.
  - **For S12:** the Cognito call runs inside the invite transaction, holding the trust's staff lock and a database connection for the round trip; and `serve.Main` always wires `cognito.Local{}`, so a deployed stack without the real Cognito would accept invites that send nothing.
  - **The admin check happens before the staff lock** (in `requireOrg`), so a request already past it can still commit just after the caller is demoted. A very narrow race.
- **From S10's build, 13 Oct** (minor; details in [S10 build-notes.md](sessions/S10-superadmin-console/build-notes.md)):
  - **Trusts and hospitals can't be deleted** (suspend them instead). *Editing their details was built in S10 round 1 (Rahul, 13 Oct).*
  - **No licence-holder note per instrument** (PRM-08: "the admin can note who holds the licence and until when").
  - **The console can't promote someone who already has another whole-trust role** to admin (409 "already has that access"); the trust's own admin must change it on Staff. Matters only when recovering a trust with no working admin.
  - **Every `make e2e` run adds three or four "E2E Trust …" trusts** (and their "Zoe E2E …" accounts) to the console and the sign-in list. Reset the local database when they get long.
  - **From S10's test report 2** (minor; [test-report-2.md](sessions/S10-superadmin-console/test-report-2.md), "Found while exploring"):
    - **The logo address message says "starting with https://", but `http://` is accepted** (`internal/settings/settings.go`, the URL kind allows http and https), on create and edit alike. Either require https (recommended: a page served over https would block an http image anyway) or change the message.
    - **On the dev sign-in page, a refusal (e.g. "Your trust or hospital is suspended…") appears at the top**, about 1,700px above a name far down the list, with no scroll, so nothing seems to happen. The page stands in for Cognito until S12.
    - **Turning a trust's opt-in default back to its platform default stores an explicit value** (e.g. `{"instrument.eq5d5l": false}`) rather than removing the key. It behaves the same; the console has no "use Poised's default" choice for the trust's opt-ins. `make seed` clears it for the demo trusts.
  - **From S10's verification 1** (minor; [verify.md](sessions/S10-superadmin-console/verify.md), findings 4–9):
    - **`make seed` can fail after hospitals or trusts are renamed in the console:** its upserts use `ON CONFLICT (id)` only, so re-seeding "Hospital A1" when another hospital of the trust is now called that (or a code another trust took) hits `hospitals_org_name_key` / `orgs_code_upper_key` and the reset aborts (`apps/api/cmd/seed/main.go`). Test data only; reset the database if it happens.
    - **The default HQ can only be a trust-level questionnaire** (`PublishedQuestionnairesInOrg` has `hospital_id IS NULL`), though the setting is hospital-overridable. Nothing reads it until S13: decide there whether a hospital's own published HQ may be its default.
    - **New trusts store every opt-in explicitly:** the New trust form sends all seven at their defaults (`NewTrustForm.tsx`), so console-made trusts never follow a later change to a Poised default, and the opt-in table has no "use Poised's default". Send only the changed keys when convenient (related to the explicit-`false` item above).
    - **Email checks exist three times with slightly different rules:** `normalizeEmail` (staff.go), the settings Email kind (`settings.go`, which accepts `help@trust.`) and `cmd/platform-admin`. Make one shared checker.
    - **Suspension is a `status = 'active'` filter repeated in six queries.** S13's patient records and S16's reminder job must apply it too (or patients of a suspended trust get reminders whose links answer 403); note it in those briefs, or move it into one view or helper.
    - **Small duplication:** the seed and apitest's `PlatformAdmin` fixture insert into `platform_admins` with raw SQL, though the `AddPlatformAdmin` query exists.
  - **From S10's test report 3** (minor; [test-report-3.md](sessions/S10-superadmin-console/test-report-3.md), "Found while exploring"):
    - **Wording when a trust's hospitals are all suspended:** in a session that was already open, the start page (`/`) tells the trust's own whole-trust admin "You don't have access to any hospital yet. Ask an admin of your trust to invite you." Something like "Your hospitals are suspended at the moment" would fit (`apps/web/src/pages/Home.tsx`; `/api/me` would need to say why a trust is missing). A new sign-in is refused with the suspended message, so it's rarely seen.
    - **The console header's "Your hospitals" link isn't refreshed** after the platform admin suspends or re-activates a trust or hospital where they're also staff; a reload shows the change (`AdminLayout.tsx` reads `/api/me` once).
  - **From S10's verification 2** (minor; [verify.md](sessions/S10-superadmin-console/verify.md), "Verification 2"):
    - **A trust whose hospitals are all suspended still answers its trust-level routes** (`/api/o/{oid}/staff`, `/settings`) to a whole-trust member who is already signed in or signs in through another trust: `ListOrgMemberships` checks only the trust's status, while `ListMyHospitals` and `SignInState` now treat such a trust as unusable. Not reachable from the UI (those pages sit under a hospital) and no patient data; fold it into the single "usable membership" helper suggested above.
    - **An empty hospital text setting means two things:** trust Settings stores `""` as the hospital's own value (e.g. no patient phone at A1, hiding the trust's), while the console's hospital edit treats `""` as "follow the trust", and its table then shows A1 as using the trust's phone. Decide one meaning before S15 shows contact details to patients (`trust_settings.go` `check`, `admin.go` `textSettings`).
    - **Editing a trust made before S10 runs the full new-trust checks** on all its stored details, so a migrated trust (code `migrated-<uuid>`, empty contact) can't have its name changed until its code and contact are fixed too (`admin.go` `adminUpdateOrg`). Local data only; status changes aren't affected.
    - **A PATCH with `"optIns": {}` is refused "Nothing to change."** even when it also changes the status or details (`admin.go` `statusPatch.validate`). The UI never sends it.
    - **The journeys assume the seeded Sam Superadmin is staff of no trust.** Test report 3 left him admin of QA Trust Seven, so his sign-in went to that hospital and two J6 journeys failed until `make seed` (which resets the demo users' memberships) was run. Run `make seed` before `make e2e` after a manual test round.
  - **Email subjects and per-status messages (EPI-08)** aren't settings yet; add them with the sessions that send them (S14–S16, C3).
- **The audit log (S11)** ([build notes](sessions/S11-audit-log/build-notes.md)):
  - **The CSV export has no size limit and is buffered twice** (`writeAuditCSV`, then the "no row, no data" hold in `heldResponse`). The full local export is already **5.6 MB**; on AWS it will pass **Lambda's 6 MB response limit**. Before the AWS phase (S06/S17): export to S3 and return a pre-signed link, or stream it (writing the export's row first). (S11 verification 1, finding 5.)
  - **Export CSV is a plain download link** (`AuditPage.tsx`): a 400, 403 or 500 is saved as a file holding `{"error": …}`, and the page says nothing. Fetch it and show the error instead, when convenient (finding 6).
  - **The audit insert is a separate statement after the handler's own write** (`audited`): if it fails after a POST saved, the user gets a 500 for a change that happened and a retry duplicates the patient, episode or note; if a handler panics after saving, `recoverPanics` answers 500 and no row is written. Rare; the root fix is writing the row in the handler's transaction when handlers move to transactions, and a deferred status-500 row would close the panic case (finding 7).
  - **`heldResponse` hides the real writer** from `http.MaxBytesReader`, so an oversized body doesn't mark the connection to close. Harmless today; add an `Unwrap` or pass the real writer to the reader (finding 8).
  - **The screen shows the newest 500 rows** with no paging; the CSV has them all.
  - **Rows don't record the IP address or browser.** Add them if a trust or the DSPT asks.
  - **The "Who" filter's list is a `DISTINCT` over the trust's whole log** (`ListAuditUsers`); add an index or read it from memberships plus removed staff when the log is large.
  - **Retention isn't built:** nothing is deleted (Rahul, 12 Oct). Deleting rows past the trust's retention period needs a deliberate change to the append-only trigger.
  - **Found by S11's test report 1** (cosmetic; [test-report-1.md](sessions/S11-audit-log/test-report-1.md)):
    - **The CSV's Patients column has names only,** while the screen shows "Name (hospital number)". The record IDs have their own column; add the hospital number to the CSV if admins want it (`writeAuditCSV` in `audit.go`).
    - **The Audit screen's Who dropdown is as narrow as its field,** so "Name (email)" options wrap onto two lines (`AuditPage.tsx`; e.g. Mantine `comboboxProps={{ width: "target" }}` → a wider dropdown).
  - **Found by S11's test report 3** (cosmetic; [test-report-3.md](sessions/S11-audit-log/test-report-3.md)):
    - **While a changed filter loads, the previous rows stay with no spinner** (the spinner shows only before the first table). On a slow connection the address already shows the new filter over the old rows for a moment. Show the spinner (or dim the table) whenever a load is in flight (`AuditPage.tsx`, `loading`).
    - **Opened straight onto a refused search, Who lists no people**, because the people come only from a successful load; it recovers after Clear. Send the people list with the refusal, or load it separately.
  - **Found by S11's verification 2** ([verify.md](sessions/S11-audit-log/verify.md); none blocks):
    - **Filtered by a patient, a list row's ten names usually don't include that patient** (`auditListRows` in `audit.go`): searching "Bloggs" gave 333 list rows, none naming Jo Bloggs among its ten. Put the matched patients first in each row when the `patient` filter is set. Small and worth doing soon: it's the main way the screen is used.
    - **Every list load writes every listed patient's ID into its row** (`listPatients`, `writeEpisodes`), so row size and GIN index writes grow with the trust (about 800 KB per row at 50k patients). This follows Rahul's "the record IDs it touched"; revisit (e.g. store the filters a list used instead) before a large trust goes live.
    - **A short Patient search ("a") records every patient it matches** as looked up by the admin, and sends them all to `ListAudit`'s `&&`. Consider a minimum length, or a cap on matches.
    - **`audited` sends the response without a row when the trust is unknown** (`audit.go`, the `e.org == uuid.Nil` branch). Today only an unknown patient link reaches it; a future route wrapped outside `requireHospital`/`requireOrg`, or a `/api/p/…` handler that skips `loadByToken`, would skip the log silently. Answer 500 there unless the response is a 404.
    - **The spinner can stay on** if a bad-day link loads while an earlier load is still running (`AuditPage.tsx`: the bad-day branch doesn't clear `loading`). Back or a link only; cosmetic.
    - **The export sends the browser's time zone,** which Go's zone database might not know; that export answers 400 and the plain link saves the error as the file (see "plain download link" above). `parseAuditFilter` also repeats `validTimeZone` (`admin.go`).
  - **AUD-01's questionnaire change history** (who changed what in authoring, before and after) isn't part of this log, which covers patient data only.
- **Patient accounts, sign-in links and the home (S15, [build notes](sessions/S15-patient-home/build-notes.md)):**
  - ~~**S06 must make the rate limit's client IP the real patient's** (`clientIP` in `accounts.go` reads `RemoteAddr`). Behind API Gateway and Amplify's rewrite that is a proxy's address unless the Lambda adapter sets it from the request context (or a trusted `X-Forwarded-For`): otherwise every patient shares one address, and 10 sign-ins an hour would be the whole country's.~~ *Built in S06: `internal/serve` sets `RemoteAddr` from `X-Forwarded-For` (`FORWARDED_HOPS`, set by `infra/api_stack.py`).*
    - **Still open (S06):** someone calling the HTTP API's own `execute-api` address directly, not through Amplify, has one hop fewer, so the address they put in `X-Forwarded-For` is the one counted, and they can dodge the limit. Closing it needs the API reachable only through Amplify (e.g. a custom domain with the default endpoint switched off, or a header only the rewrite adds).
  - **S14's X-1 is partly better:** leaving the form by **← Your questionnaires** or the browser's Back to the home is an in-app route change, so a waiting autosave is sent; a reload or closing the tab within ~0.6 s still loses it.
  - **An account keeps the date of birth proved when it was made.** If a hospital later corrects a record's date of birth, sign-in with the corrected date fails for that account (invites still work). A staff correction could update the account later.
  - **A patient can change or add a mobile or email, not remove one.**
  - **A first-time patient (never opened an invite) can't use `/signin`:** by design (PAT-06 is the returning patient's), and the screen says to use the hospital's link.
  - **Sign-in links' wrong dates of birth, and their joins when no record changed, are in no trust's log** (no trust); a security log may want them (as S14's probe item).
  - **A save that fails on the network shows the patient the raw "TypeError: Failed to fetch"** (S15 test report 4, exploring 1; S14's save-error display, also on the ordinary autosave). The page does stay with **Not saved**, and leaving later re-sends the change. Friendlier wording: "We couldn't save your latest change. Check your connection and try again." (`api.ts` could turn a network failure into that message.)
  - **A double tap on "← Your questionnaires" right after typing sends the save twice** (S15 test report 4, exploring 3, cosmetic): both 200, the revision moves by 2, the stored answer is right. Disabling the link while it saves would stop it.
  - **Accepted risk (Rahul, 13 Oct, S15 answers 10; verification 1, finding 1):** a person sharing a contact and date of birth with an account holder, whose own record isn't joined yet (an unjoined twin), signs in to the **other** person's account with a sign-in link, because the name step runs only when two or more accounts match. Rahul: too rare to change sign-in for. If it's ever revisited, the verifier's fix: ask for the name also when an unjoined record with the link's contact and that date of birth has another name (`signinDob`, `AccountsForSignin`).
  - **Migrations 00014 and 00015 aren't safe for a database with real accounts** (S15 verification 1, finding 3): they unjoin records merged under the old rules but leave the contacts those records filled into the account (which then still signs in to it), and leave open patient sessions running. Only local and test databases (made-up data) have run them. If a QA or real database ever needs them, a later migration should rebuild each account's contacts from its remaining records and end its open sessions.
  - **`POST /api/p/signin` still differs in timing for a contact with an account** (S15 verification 1, finding 4): only they wait for the send. *The failure half was fixed in S17 (14 Oct): a failed send answers 202 like "no account".* To close the timing half, send after answering, or from a job.
  - **A contact change revokes the earlier verification link before sending the new one** (S15 verification 1, finding 5; `requestContactChange`): if the send fails (502) both links are dead, and the failed request counted against the rate limit. Revoke only after the send succeeds.
  - **After signing out from a sign-in-link session, the card says "open the link in your text message or email again"** (S15 verification 1, finding 6, cosmetic; `PatientHome.tsx`): a sign-in link works once, so that link now says "expired" (the card's "You can also sign in…" line does cover it). Possible wording: "To come back, open your hospital's link again or sign in below."
  - **For Rahul: a person with two accounts can't join them** (S15 test report 3, exploring 1). Different contacts at two trusts (or a later surname change) give the same person two accounts and two homes; changing a contact to the other account's is refused ("already used by another account with your details… contact your hospital"), but **no staff tool joins accounts yet**. Decide whether staff (or PAT-08) get an account merge.
  - **That collision is refused only at Confirm** (S15 test report 3, exploring 1): `POST /api/p/contact` answers 202 and sends the confirmation, and only opening it says "already used". Refusing at the request (the same `ContactTaken` check) would save the wasted step and message. Rare: the same full name and date of birth on two accounts.
  - **The mobile box's refusals mention email** (S15 test report 3, exploring 2, cosmetic): `hello` gets "Enter a real mobile number or email address."; a mobile in another country gets "…Please use your email address instead." Box-specific wording would read better.
  - **Change / Add on the home doesn't put the cursor in the new box** (S15 test report 3, exploring 3, cosmetic; keyboard users must Tab). The sign-in name step does focus.
  - **The per-contact limit counts verification emails too** (S15 test report 3, exploring 4): two confirmations to a shared inbox plus one sign-in link use up its 3 in 15 minutes, and the 429 says "You've asked for 3 links". A limit on messages to that inbox, as intended; the wording could say "messages".
  - **From S15 verification 2's code review (all minor):**
    - **A session covers the whole account but lives and dies with the link it was opened with:** Trust A's Resend, or five wrong dates on Trust A's invite, also signs the patient out of Trust B's form (they're told what wasn't saved). Consider ending sessions per account, not per link.
    - **Confirming a contact change ends no open sessions** (`confirmContact`): a session left open on a shared phone carries on (up to 30 idle minutes) after the owner is told.
    - **`rate_limit_hits` keeps rows forever for keys never seen again** (`DropOldHits` deletes only the key being hit), and the keys hold plain emails and mobiles: drop every row older than a day (remove the key filter), or clean up from S16's jobs.
    - **`greeting` can prefer a capitalised lower-case name over an older properly-cased one** ("mary ann" newest, "Mary Ann" older → "Hello Mary ann"). Cosmetic.
- **Scheduled jobs and reminders (S16, [build notes](sessions/S16-jobs-reminders/build-notes.md)):**
  - **The invite is still sent in the Create request**, not from a job (C2's S14 status line had said S16 would move it; the brief didn't list it). Staff keep the immediate "Invite sent by …" note. **For S17 / Rahul:** move it to a job if retries or request time matter once real SMS and email are slow or fail.
  - **The jobs write no audit rows** (S11's actors are staff and patient). What they did is in the episode's General notes ("Automatic"). If a "system" actor is wanted in the audit log, it's a small change (`audit_log.actor`).
  - **A reminder whose sending fails isn't retried** (as an invite): the General notes say so and staff can resend. The reminder is counted before it's sent, so a crash between counting and sending loses that reminder rather than sending it twice.
  - ~~**"Not responding" is a badge on the episode only**; the worklist (EPI-02) is C3's.~~ *Built in S21: the Not responding worklist.*
  - **Old episodes have no follow-up:** only episodes whose first invite went after migration 00016 get one (no backfill: local data only).
  - **`rate_limit_hits` clean-up** (S15's item above) could now be a job.
  - **For Rahul: on catch-up the reminder and "Not responding" can come together** (S16 test report 1, exploring 1): after a suspended hospital is re-activated past day 7 (or a run missed days 5–7), one run sends the day 5 reminder and flags the episode "Not responding" at the same moment. As designed (only the latest reminder, once); the alternative is to skip a reminder whose day is at or after the "not responding" day.
  - **Reword the staff lock banner (Rahul, 13 Oct, S16 answers 5):** it says "…every link sent to the patient for this episode is locked…", but a returning patient can still reach the episode through an S15 sign-in link (its own 5 tries, sent only to their contact), which Rahul keeps as a fair way back in. Say "the invite links are locked" (and perhaps "they can still sign in with their mobile or email") in `EpisodePage.tsx`.
  - **The signed-out card points to a now-locked link** (S16 test report 2, exploring 2, cosmetic, older than S16): after a lock ends an open session, "Please open the link in your text message or email again" leads to "This link is locked"; "Get a sign-in link" is just below. Could say "ask the hospital for a new link, or sign in below" when the session ended by a lock.
  - **`episodes.dob_failed_attempts` can go past 5** (S16 test report 2, exploring 3, no visible effect): wrong dates sent in parallel that were already waiting on the episode's row still add to the count after the lock (9 after 12 parallel tries); the lock happens at the fifth and a Resend resets it. Cap it with `least(…, max)` if anyone reads the column.
  - **From S16 verification 1 (minor; [verify.md](sessions/S16-jobs-reminders/verify.md)):**
    - **A Resend can hand out an already-locked link (race, fails closed):** a fifth wrong date that lands while the Resend's messages go also locks the new link (`FailInviteDob` locks every live link), and `settle` resets only the count. Fix: in `invite.settle` (First/Again) also clear the new link's `locked_at` and `failed_dob_attempts`.
    - **A reminder link made just after a lock comes out unlocked (race, negligible):** it goes only to the patient's contacts, and one wrong try re-locks it. A root fix is one `episodes.dob_locked_at` read by the link checks.
    - **QA "run as of":** `qaRunJobs` should use `context.WithoutCancel` (a closed tab can leave a reminder counted, not sent); a job's lease and backoff are taken from the "as of" time, so a job failing in a "+7 days" run waits for that real day.
    - **`GET /api/qa/jobs` says the timer is on for staff-api and patient-api** (`cfg.JobsTimer`, not whether this process runs it). For S17: pass the effective value.
    - **The staff "Last sent …" shows the newest reminder's time** (`GetEpisodeInvite` counts reminder links). Cosmetic.
    - **If `settle` fails after the messages went**, the note says "not sent" and a first invite doesn't start the follow-up (a database failure; S14's pattern).
    - **J7 and J8 flake:** their made-up mobiles are one of 1,000 (`07700 900nnn`) and the local database has texts to 177 of them, so "exactly one text" fails about one run in six. Filter by time or link, or widen the range.
    - **Cleanups:** a `settings.ForHospital(org, hospital)` helper for the decode-and-combine done in three places; one `FailLinkDob` call in `signinDob`; drop `invite.Sender.now()`.
  - **Cosmetic (S16 test report 1):** double-clicking **Run jobs** sends two requests (harmless: the second finds nothing due); the captured messages and automatic General notes show the real time, not the "as of" time, while `not_responding_at` holds the "as of" time (a test helper's quirk); each staff episode view writes two audit rows (React StrictMode in `make dev`, older than S16: see S11).
    - **`AccountsForSignin` / `AccountHasContact` match `mobile = $1 OR email = $1` without `<> ''`**, so the partial indexes may not be used under a generic plan. Add the guards as `FindAccountForRecord` does.
    - **`signinDob` picks the account before locking the link**: a contact change confirmed in that instant could still let the old contact's link sign in. Tiny window; re-check inside the transaction.
  - **On a shared phone or inbox, "your contact details were changed/added" notices reach the other person too** (S15 test report 3, exploring 5): expected with shared contacts; noted only.
  - **Invite texts don't name the patient** (S15 test report 2, exploring 10): on a shared phone a parent can't tell which twin's invite is which until after the date of birth (the home then says whose). A privacy trade-off; noted only.
  - **Emails must be plain ASCII** (S15 answers 7): `ü` addresses are refused, which may matter for other countries later (A-20).
  - **A surname change (marriage) keeps a separate account** (S15 answers 8, fails safe): the patient then signs in to each account separately until staff or PAT-08 offer a merge.
  - **View on a sent questionnaire shows only "Thank you, Sam"**, not the answers (S15 test report 1, exploring 7). A read-only view of what they sent would help.
  - **The patient cookie has no `Secure` flag locally** (http); it's set when `APP_URL` is https. S06 should check it on QA. *S06: the staff cookie and both sign-outs are now Secure on https too (integration test); on QA the staff cookie arrives `HttpOnly; Secure; SameSite=Lax` (checked 14 Oct).*
  - **Rahul to hand-check R-6:** suspend Trust B as Sam Superadmin; a patient's home then lists only Trust A's questionnaires; re-activate it (the tester's permissions refused Suspend).
  - **`test_audit.py`'s tooltip journey failed once** in a full `make e2e` run (S15 round 1) and passed alone and in the next full run: a hover flake to watch.
  - **No General notes entry when a patient changes their contact:** staff see the new contact on the record, and the trust's audit log has "Patient changed their contact details".
- **Patient invites and sign-in (S14, [build notes](sessions/S14-patient-invites/build-notes.md), [test report 1](sessions/S14-patient-invites/test-report-1.md)):**
  - **The API accepts Send with no answers.** `POST /api/p/episodes/{eid}/submit` (and staff's on-behalf submit) doesn't check required answers: the web app keeps "Send my answers" disabled until every section is complete, but Go can't evaluate SurveyJS. **Needs C6's `evaluate` Lambda** to check on the server.
  - **Send the full name and age only when a questionnaire uses them.** After the date of birth the HQ includes `{patientName}` (with the surname) and `{patientAge}` for every questionnaire (confirmed by Rahul, answers 2); a later refinement could send them only when the questionnaire's text or conditions use `{patientName}`/`{patientAge}`.
  - **"Entered by" stays on a set the patient changes afterwards** (by design: sticky per Question Set). Part of the set may then be the patient's own.
  - ~~**Cosmetic:** a patient opening someone else's episode address sees only the red "Episode not found." with no Poised bar or Sign out.~~ *Done in S15: inside the Poised bar, with ← Your questionnaires.*
  - **Opening a second link in the same browser doesn't end the first session on the server** (test report 2, item 2; low risk): the browser's cookie is replaced, but the first session's row stays valid until 30 minutes idle, usable only by someone who already copied that cookie. A fix: end the browser's previous session when a new one starts (the date-of-birth step can read the old cookie).
  - **Sign out ends only that browser's session** (test report 2, item 1; as intended): other sessions on the same link carry on until idle; a Resend or a lock ends them all. Rows ended by a Resend or a lock keep `ended_at` empty (refused through the link's state instead).
  - ~~**An answer changed after the session ended isn't saved, and no message says so** (test report 2, item 3).~~ *Fixed after S14 verification 1: the signed-out card names the section whose change wasn't saved, or says the answers weren't sent.*
  - **The patient's browser Back, a reload or closing the tab within about 0.6 s of typing loses that last input, silently** (S14 test report 4, X-1; minor; **Rahul, 13 Oct: skip for now, an edge case**, answers.md item 5). Opening a section isn't a history entry, so Back leaves the page; the waiting save is sent only by React's unmount cleanup, which doesn't run when the page unloads. **Suggested fix:** on `pagehide` (and `visibilitychange` to hidden), send the waiting save with `fetch(…, { keepalive: true })`. Staff's on-behalf form is unaffected (Back there is an in-app route change).
  - **A section with no required questions shows COMPLETE ("1 of 1 sections complete") before anything is answered**, and **Send my answers** is enabled (S14 test report 4, exploring 1): it follows from "nothing is required", but a patient may read COMPLETE as "done".
  - **While typing, the status says "Saved" until the 600 ms autosave fires** (S14 test report 4, exploring 2): no sign the latest keystrokes are still waiting. Showing "Saving…" (or "Not saved yet") from the first keystroke would help, and matters for X-1.
  - **A Resend whose messages went but whose settle step failed** (a database error in `settleInviteLink`; S14 verification 2) answers 500 and the notes say "No invite sent: something went wrong", though the patient got a working link and the older links weren't revoked. Rare; pressing Resend again puts it right.
  - **The journeys reuse made-up mobiles** (J3's fixed `+447700900123`; `made_up_mobile` picks from 1000) while `sent_messages` keeps growing in the dev database, so a count such as `to_have_count(1)` could one day see an old message (S14 verification 2; not seen yet).
  - **`loadForm` returns ok for an episode that isn't the caller's**, and each caller checks ownership itself (`patient.go`). Correct and tested today, but fragile for the next caller: pass the owner check into `loadForm` (S14 verification 2).
  - **The "Not saved" box doesn't survive a reload** (S14 test report 3, item 1): reloading the signed-out page after a refused change shows the plain "You've been signed out", and the hospital's name leaves the top bar. Acceptable: the patient has seen the box, and the reload starts a page that knows nothing about the old session.
  - **Cosmetic:** the section's name in the "Not saved" box ("About you") is plain text; the test plan showed it in bold (S14 test report 3, item 2).
  - **The patient's `{patientAge}` and the clinician views can disagree around a birthday** (S14 verification 1): the patient's age is worked out in Go on the hospital's date, the clinician views (`ageFrom`) on the browser's local date, so a condition such as `{patientAge} >= 70` can differ for a few hours.
  - **`/api/qa/messages?to=` treats `%` and `_` as wildcards** (ILIKE; the QA tool only).
  - **A signed-in patient trying another patient's episode id is in no audit log** (by design, so it doesn't land in the other trust's log); a security log may want it later.
  - **Capture mode holds full links in plain text** (`sent_messages`, `/qa/messages`): S17 must make sure capture can never run in a real deployment. *S15 adds the unauthenticated `DELETE /api/qa/rate-limits` (capture only); when real sending lands, S17 should add a router test that `/api/qa/messages` and `/api/qa/rate-limits` are 404 unless capture is on (S15 verification 1).*
  - **The date-of-birth boxes are fixed in day, month, year order, with the example "27 3 1985"** (`InviteLink.tsx`, and Go's error text; S14 verification 3; A-20). The boxes are labelled, so nothing is misread, and only the UK runs today. Before a second country, take the order and the example from the deployment's locale.
  - **The right date of birth resets the wrong-try count** (a builder decision), so someone holding a link gets 4 more guesses each time the real patient signs in (S14 verification 3). That's slow to exploit; a per-link rate limit would close it.
  - **Two Resends that cross:** the slower one's messages can arrive after the newer one's, with a link already revoked, so the newest-looking text says "This link doesn't work" (S14 verification 3; rare; Resend again fixes it). S16's jobs could send in order.
  - **The on-behalf form's unmount save ignores its result** (`PatientHQ.tsx`, the SetForm cleanup): staff who leave through the top bar within 0.6 s of typing, and whose save then fails (409 or 500), aren't told. ← Episode and Sign out do report it (S14 verification 3).
  - **A patient session opens every episode of that patient** (as designed and tested): a lock or Resend on episode B's link doesn't end a session opened through episode A's link (S14 verification 3). Revisit with S15's patient home. *S15: a session is now the account's, so it opens every joined record's episodes in every trust; it still ends only with the link it was opened with.*
- **Patient records (S13, [build notes](sessions/S13-patient-records/build-notes.md)):**
  - **Nothing edits a patient record or an episode's hospital number after Create.** Patients update their own contacts in S15 (PAT-10); staff editing (and who may) isn't planned yet.
  - **No merging of duplicate records:** "Create anyway" can make two records for one person, and a record created without an NHS number never gains one by linking.
  - **NHS numbers are shown as 10 digits,** not the usual 3-3-4 spacing (a display format per scheme in `internal/nationalid` would do it).
  - **The CSV's Patients column (S11) still has names only;** the screen now shows "Name (NHS number)".
  - **For Rahul to decide: guessing a date of birth reveals a record (by design).** An NHS-number **mismatch** (409) tells a clinician the trust holds that NHS number; trying dates of birth until one matches then **links silently**, and the episode and patient pages show the trust's record (first name, mobile, email). Likewise name + date of birth + a guessed mobile links and shows the stored email. Every try is in the audit log ("Created an episode", refused 409), and a success creates a real episode. **Suggested follow-up:** a per-user limit on mismatches (e.g. 5 an hour, then refuse and alert the trust's admins), or an audit alert on repeated mismatches. (S13 test report 1, item 3.)
  - **Rapid clicks on Create episode can send several requests** (three synthetic clicks in one tick sent three POSTs; a real double-click sent one). With an NHS number the unique index refuses the extras; without one, two records could be made. Disable the button synchronously on the first click (`NewEpisodePage.tsx`). Low risk. (Test report 1, item 4.)
  - **Cosmetic (test report 1):** the Audit screen's Patient box placeholder says "Name or hospital number" though the NHS number works too (`AuditPage.tsx`). *(The patient page's Episodes highlight and the "not found" page's ← Episodes link were fixed after test report 2.)*
  - **Change patient also empties a hospital number staff typed themselves** (test report 2, item 2), not only one filled in from the chosen patient. The safe side of the round 1 fix; keep a typed one if it matters (`NewEpisodePage.tsx`, `changePatient`).
  - **The New episode search table has no visible "Patients found" heading** (it's the table's accessible name only). Cosmetic (test report 2).
  - **Found by S13's test report 3** (cosmetic; [test-report-3.md](sessions/S13-patient-records/test-report-3.md)):
    - **A patient chosen from the search isn't kept in the address,** so a reload drops the choice; one chosen from the patient page (`?patient=`) survives a reload. Consistent with "Change patient forgets the patient"; put `?patient=` on Choose too if wanted.
    - **`…/episodes/new?patient=<a patient not at this hospital>`** shows "Patient not found." at the bottom, above Create episode, easy to miss on a short screen; show it at the top (and drop the parameter).
    - **Cancel on New episode always goes to Episodes,** even when it was opened from a patient's page; returning there might be expected.
  - **Found by S13's verification** (minor; [verify.md](sessions/S13-patient-records/verify.md)):
    - **The `?patient=` load has no stale guard** (`NewEpisodePage.tsx`): Change patient clicked before it answers is undone by the late answer (the card shows again, so it's visible).
    - **Two staff creating the same patient without an NHS number at the same moment make two records** (`resolvePatient`): only the NHS number has a unique index. An advisory lock on (trust, date of birth) would serialise them.
    - **Name length counts bytes, not characters** (`patients.go`, `details`): non-Latin names are refused early (A-20).
    - **`TooBroad` says "an " + the identifier's label** (`patient.go`): wrong for another country's label (A-20); reword.
    - **The patient search ORs three conditions** (`patients.sql`, `SearchPatients`), so it can't use its indexes; a UNION of three indexed branches before a large trust.
    - **The Audit filter compares `national_id` without its type** (`audit.sql`, `MatchAuditPatients`); harmless with one scheme per deployment.
- **QA on AWS (S06, [test report 1](sessions/S06-aws-qa-stack/test-report-1.md)):**
  - ~~**QA is open to anyone who has its address until S12 (Rahul, 14 Oct: leave it; keep the URL private meanwhile).**~~ *Closed by S12 (14 Oct; deployed to QA by hand, tested and verified 14 Oct): Cognito sign-in, no dev login on AWS, the tools for platform admins only, Amplify's security headers.* Before S12, with no sign-in:
    - `GET /api/dev/users` lists every account, and `POST /api/dev/login` signs in as any of them, the superadmin included;
    - the captured messages (`/qa/messages`, `GET /api/qa/messages`) list every text and email, with live `/i/`, `/s/` and `/v/` links;
    - **Reset sign-in limits** undoes the 5 tries;
    - `/qa/jobs` runs the jobs or moves the clock;
    - the web app has no security headers, so it can be framed by another site.

    **S12 closes all of it:**
    - Cognito replaces the dev login;
    - the QA tool pages (captured messages, `/qa/jobs`, the rate-limit reset, `/api/qa/client`) are only for a signed-in platform admin;
    - Amplify custom headers add CSP `frame-ancestors`/`X-Frame-Options`, HSTS and `X-Content-Type-Options` ([S12 answers](sessions/S12-cognito-signin/answers.md) item 3).

    The data is made up.
  - ~~**An invited staff member who hasn't signed in yet is in the dev login's list**, and clicking them counts as their first sign-in (exploring 3; S09's design). It goes with the dev login in S12.~~ *Gone on QA with the dev login (S12); local only now.*
  - **Signing in from a pasted deep link lands on Episodes**, not the page asked for (exploring 6; older than S06). The login page could take a `?next=` path, kept to the app's own addresses.
  - **Reloading the patient's questionnaire part-way through a set goes back to the set list**, not the page they were on (exploring 7; older). The answers and the session are kept, and Continue reopens the set.
  - **The API Gateway address bypasses the rate limit's hop count:** a direct call to `https://<api id>.execute-api.<region>.amazonaws.com` with a made-up `X-Forwarded-For` of two or more addresses picks the one the caller chose (see S15's "Still open (S06)" above). Fix with the API reachable only through Amplify, or a secret header only the rewrite adds (Amplify can't add one today).
  - **The captured messages' "To" box has no label of its own** (its accessible name is the placeholder; exploring 9, cosmetic, older).
  - **From S06's verification, 14 Oct** (minor; details in [S06 verify.md](sessions/S06-aws-qa-stack/verify.md)):
    - **The `execute-api` bypass above must be closed before real patients** (C10 at the latest); add it to a brief (S07 or S12) when it fits.
    - **`FORWARDED_HOPS` isn't required under `CONFIG_STRICT` on Lambda** (`config.go`): the CDK always sets it (and a CDK test checks it), but strict mode would catch a stack that didn't.
    - **Docs:** CLAUDE.md's requirements line doesn't say `make test` now needs Python 3.12 (for the CDK tests); `infra/requirements.txt` mentions a `make infra-setup` target that doesn't exist.
    - **`make aws-setup` passes the Neon URL on the command line** (`aws ssm put-parameter --value`), briefly visible in the process list; `--cli-input-json` from a 0600 file would avoid it.
    - **Nits:** the token secret's name is CDK-generated (`TokenSecret…`, tagged `project=poised`); `serve.onLambda` and `config.Load` both test `AWS_LAMBDA_FUNCTION_NAME`.
- **The pipeline (S07, 14 Oct):**
  - **The idle-cost check is due 21 Oct 2026** (plan F4 Step 4): Rahul reads Cost Explorer for the week (tag `poised-env` = `uk-qa`, plus CodePipeline, CodeBuild and S3 for the pipeline) and records the monthly figure in F4's Status. Target: under £1/month for QA, excluding build minutes.
  - **The E2E JUnit report labels every journey `[chromium]`** (S07 test report 1, exploring 3): the patient journeys run on WebKit through the `phone` fixture (`e2e/conftest.py`), but `junit.xml` names only the page's browser, so it can't show which ran on WebKit. Tag the phone journeys (e.g. a `[webkit]` marker or a name suffix) if the report is ever read by machine.
  - **The pipeline's CodeBuild projects aren't named `poised…`** (S07 test report 1, exploring 4): CDK names them `PipelineBuildSynthCdkBuildP-…`, `PipelineukqaE2E…` and so on (only `poised-selfupdate` is). The runbook says how to find them. Giving each `CodeBuildStep` a `project_name` (`poised-synth`, `poised-<env>-e2e`, …) would make them easy to spot in a shared account.
  - **The journeys' test data is the UK's:** J3, J7, J8 and the patient-records journey type a UK mobile (`07700 900…`) or expect a UK date ("17 May 1980"), so in another country's QA they fail at those lines (seen in S07's us-east-1 proof, where J3 passed once given a US mobile and "May 17, 1980"). Before a second country's pipeline stage, make them read the phone country and locale from `GET /api/config` (e.g. a made-up number per country in `e2e/support.py`).
  - **From S07's verification (14 Oct):**
    - **Only one account:** the pipeline can deploy only to environments in its own account (`cross_account_keys=False`, and the Database-and-web and E2E roles live in the pipeline's account). An environment in another account needs `cross_account_keys=True` (a KMS key, about $1 a month) and cross-account roles for those two steps. Decide this before the first environment outside account 026090536258.
    - **Migrations run after the new Lambdas are live** (Database-and-web is a `post` step), so a push with a migration plus code that needs it can give QA a minute or two of 500s. For C10, run goose as a `pre` step and keep migrations expand-then-contract.
    - **Every environment gets the demo seed and the journeys** (nothing checks `stage`), so C10 must limit them to QA, or fail synth for a Training or Production entry, before adding those entries.
    - **Administrator rights in a shared account:** Poised's CDK bootstrap gives CloudFormation `AdministratorAccess` (`aws-setup.sh`), so any commit on `master` can create anything in the account. The self-mutation role may also assume any CDK bootstrap deploy or publishing role in the account, other projects' included: CDK's default condition matches by tag, not by qualifier. Before go-live (C10), give Poised its own account(s), or a scoped `--cloudformation-execution-policies` plus a permissions boundary.
    - **The Database-and-web role may deploy to any Amplify app** in its account and region (`apps/*`). Scope it to the WebStack's app.
- **Staff sign-in with Cognito (S12, 14 Oct):**
  - **Cognito's own email sends about 50 a day per pool** (invitations, password resets), from `no-reply@verificationemail.com`. Enough for QA and a demo; **a trust's onboarding needs SES** (a verified domain, out of the sandbox) as the pool's email (`UserPoolEmail.with_ses`), with Preopsis's own From address. Do it with S17's SES or before the first customer (C10).
  - **The managed login looks like Cognito's**, not Preopsis's (`CfnManagedLoginBranding` with Cognito's values). Add the logo and colours before a demo, in CDK (the branding's settings JSON). The logo and colours exist since 20 Oct: [brand/](brand/README.md).
  - **No adaptive authentication or compromised-password checks:** those are the Plus plan's (about $0.02 per active user, no free tier). Essentials still locks an account for a while after repeated wrong passwords. Decide with C10's security review.
  - **The staff session cookie lasts 12 hours and Sign out only clears it in that browser** (older than S12: `auth.TTL`, a signed token with no server-side session). A copied cookie still works until it expires. Patients' sessions are held in the database (S14); staff could be too, before go-live.
  - **Training and Production stages have no test users**, so the pipeline's journeys couldn't sign in there (the E2E step fetches test users on QA only). Settle with S07 verification's "every environment gets the demo seed and the journeys" item, before adding those entries (C10).
  - **Two journeys run only locally** (`test_staff.py`, `test_j6_onboarding.py`, marked `dev_login`): they sign in as people they've just invited, and on QA that person's temporary password is in Cognito's email. Their API paths are covered by the integration tests on every push; the invite email itself is Rahul's hand check.
  - **Each authenticator code works once**, so a tester and the pipeline signing in as the same test user in the same 30 seconds can see "Invalid or expired code"; the journeys retry once with the next code, a tester just waits for it.
  - **Another open tab keeps showing patient names after a sign-out in a different tab** (S12 test report 2, exploring 1; older than S12, Minor). Tab 2's Episodes list stays readable until someone clicks; then a red "Please sign in." box appears under the still-shown header ("Bea Author · Super clinician", Sign out) instead of going to `/login`. The session really has ended (every API call answers 401), so nothing new loads, but on a shared workstation a list left open in another tab stays readable. **Fix:** any 401 from the staff API sends the page to `/login`, and Sign out tells the other tabs (a `storage` event or a `BroadcastChannel`) so they empty their pages and go there too.
  - **Hand re-check (S12 test report 1, finding 3; S11/S15 tools, not S12):** the tester's automation couldn't make the captured-messages **To** filter or the Audit **Who** filter narrow the list (setting the value by script, then Show); typed and submitted by hand, the messages filter did narrow. Probably the automation and React's controlled inputs, but Rahul should try both filters by hand once on QA (as Sam: `/qa/messages`, To = a patient's email; as Ada: Audit, Who = Cara Clinician). The journeys (J7, J8, the audit journey) use both filters and pass, locally and on QA.
  - **Accounts in QA's database with no Cognito user** (e.g. the "Zoe E2E" invitees left by S06–S07's journeys) can't sign in until they're invited again: **Resend invite** creates the Cognito user when it's missing.
  - **From S12's verification, 14 Oct** (all Minor; details and line numbers in [S12 verify.md](sessions/S12-cognito-signin/verify.md)):
    - **A failure after a successful callback leaves the Poised session alive** (`AuthCallback.tsx`): if `GET /api/me` fails after `POST /api/auth/callback` set the cookie, the page says "sign in again" but the cookie stays valid. **Fix:** `POST /api/logout` in the `catch` before `refuseSignIn`.
    - **Sign-out skips Cognito's `/logout` when the config failed to load** (`lib/config.ts` `staffSignIn()` falls back to the dev login, fail-open on AWS). **Fix:** retry `loadConfig()` in `signOut`, or never assume the dev login without the config.
    - **Every OAuth error from the token endpoint is "sign in again", unlogged** (`cognito/aws.go` `Exchange`): `invalid_client` or a `redirect_uri` mismatch (e.g. after a domain change) would hide an outage of all staff sign-in. **Fix:** only `invalid_grant` is `ErrCode`; log the rest, and the `ErrCode` reason.
    - **The email fallback re-links an account that already has another `cognito_sub`, silently** (`cognitoCallback`). Fine for QA's re-created pool; on a real stage, audit a re-link or allow it only when `cognito_sub IS NULL`. Decide with C10.
    - **JWKS holds its lock across the fetch, and a failed fetch isn't rate-limited** (`cognito/idtoken.go`): reachable only after a successful code exchange; set `fetched` on failure too.
    - **`cmd/qa-users setup` saves to SSM only at the end**, so a failure part-way loses the new secrets of users already re-made, and the next run keeps their stale ones. Save after each user.
    - **`make aws-platform-admin` for an email that already has a Poised account makes no pool user**, so that person can't sign in on AWS; it reports success.
    - **The pipeline's Cognito admin actions rely on `aws:ResourceTag/poised-env`** (Cognito's ABAC is "Partial" in its IAM docs): proven only by the first pipeline run (H-3).
    - **Nits:** `serve.go`'s comment still says `COGNITO_*`; `config/config_test.go` isn't gofmt-clean; the same `signOut(…)` call is in four places; `infra/scripts/lib.sh` became executable.
- **From S17 (jobs and real messaging on AWS, 14 Oct):**
  - **Before a real trust, in the demo account:**
    - SES production access, and a domain to send from. Until then QA's emails come "from" Rahul's Gmail address through SES and may land in spam.
    - SMS production access.
    - **Registration of the UK sender ID PREOPSIS** (POISED until 20 Oct): the UK form in the End User Messaging console (company details), or a sender ID of the trust's.
    - Raising the **$1 monthly SMS spend limit** (a support request; about 20 UK texts a month).
  - **A real undelivered text can't be tested in the UK sandbox.** AWS's SMS simulator numbers are American, which PAT-11 refuses, and QA sends to no unverified number. The path is covered by Go tests (`delivery`, the jobs Lambda's handler) and needs a check in production.
  - **The protect configuration's country list is AWS's on 14 Oct** (`infra/sms_countries.py`). A country AWS adds later is allowed until it's added there (CloudFormation drift shows it). Go's PAT-11 check still refuses it.
  - **The invite is still sent in the Create request,** not from a job. With real sending, a slow SES or SMS call slows Create; a refused send is noted and shown as undelivered, and isn't retried.
  - **Undelivered sign-in links and contact-change messages** count for the alarm but aren't shown anywhere: they belong to no episode.
  - **Cosmetic, from S17's test reports (S16's wording and behaviour):**
    - **The Run jobs summary can count more results than jobs**, e.g. "Ran 17 jobs …: 14 reminders sent, 17 marked 'not responding'". One follow-up can do both when "as of" jumps 7 days. Say "17 episodes checked" instead.
    - **The invite card's "Last sent" moves on when a reminder goes** (it shows the newest link's time), though it sits under "Patient's invite" and reads as the invite's own time. Say "Last message sent …", or show the invite's time.
  - **The jobs Lambda (`cmd/jobs`) logs nothing on a run with no due jobs** (S17 test report 1). Only Lambda's own START/END lines show it ran, and `/qa/jobs`'s last run shows it on QA. Log one line per run if a run's absence ever needs proving from the logs.
  - **From S17 verification 1 (15 Oct, minor; [verify.md](sessions/S17-aws-messaging-jobs/verify.md)):**
    1. **The "no retries" comments are wrong about Lambda:** the schedule and SNS invoke the jobs Lambda asynchronously, and with no EventInvokeConfig Lambda retries a failed run twice (`infra/api_stack.py` "A failed run isn't retried"; `delivery.go`'s "SNS tries again" is really Lambda's retry). Harmless (S16's lease makes runs overlap-safe), but correct the comments or set `retry_attempts`/`max_event_age` explicitly, keeping the SNS path's retries.
    2. **`make test-recipients` stores the list unchecked** (`infra/scripts/test-recipients.sh`): one entry Go refuses (a mobile with spaces, `07700…`) stops all three QA Lambdas at start-up. Validate it in the script first.
    3. **The `episode_messages` row is written after the send and a failed insert is only logged** (`invite.Sender.record`): a later bounce for that message is then dropped as "not an episode's". Record first, or return the error.
    4. **A repeated delivery event logs `DELIVERY_FAILED` again** (`delivery.Record`, "already noted"), so the alarm can re-fire for one message. Log repeats without the marker.
    5. **An unreadable failure event isn't counted by the alarm** (`delivery.HandleSNS` logs `delivery: …` without the marker).
    6. **The badge says "Invite not delivered" when only a reminder failed**, and shows in statuses without the invite card (no box or Resend there). Cosmetic: "Message not delivered".
    7. **The sending grants are account-wide** (`sms-voice:SendTextMessage` on the account's sms-voice resources; `ses:SendEmail` on `identity/*`). Narrow them to the configuration set, sender ID and SES identity before the account holds anything else (re-test the builder's simulator note now that POISED exists).
    8. **`POST /api/p/signin` answers 202 for any send failure**, including a database failure making the link, which gets no alarm. Consider the marker for non-AWS failures too.
    9. **Nits:** `serve/lambda.go`'s imports aren't gofmt-sorted; `undelivered.ts` repeats `delivery.Note`'s wording (the API could return the sentence); `serve.senders` loads the AWS config twice per cold start; `TestRecipients.allowed` re-normalises what config already did.
    10. **J3 flaked once** in a full `make e2e` (`test_j3_patient.py:74`, "In progress" after "All sections"); it passed 3 of 3 alone. Watch it.
  - **From the S18 build (20 Oct, minor; [build notes](sessions/S18-episode-forms/build-notes.md)):**
    1. **J7 (and J8, in the S18 verification) flake on a well-used local database:** it picks a random mobile among 1,000 (`07700 900000`–`999`) and expects exactly one captured text to it, but earlier runs have used most numbers (2,271 captured messages locally). It passed on a re-run. Give it a mobile unlikely to repeat, or count only messages sent after it starts.
    2. **A reminder racing a submit:** `CountReminder` and `MarkNotResponding` now check "the HQ form is still open" in a subquery, not on the row they update, so a submit committing at the same instant could let one extra reminder go. Before S18 the check was exact. *(Verifier: the reminder is sent after the count commits anyway, so the window was already there; `FOR SHARE` on the form row in the subquery would make both exact.)*
  - **From the S18 verification (20 Oct, minor; [verify.md](sessions/S18-episode-forms/verify.md)):**
    1. **The pipeline deploys the Lambdas before it migrates** (`infra/pipeline_stack.py`, `post=[db_web, e2e]`). A migration that isn't backward-compatible, like 00019, gives 500s between the two steps, and for longer if the migrate step fails. That's fine on QA; before a customer, use expand-then-contract migrations, or migrate before the deploy.
    2. **A patient save racing a submit can change the submitted answers:** `SavePatientAnswers` checks the form is open only in its `INSERT … SELECT`, and the `ON CONFLICT DO UPDATE` path doesn't re-check it. This was the same before S18. `FOR SHARE` on the form row fixes it.
    3. **Nothing in the database enforces "every episode has its HQ form",** yet every read inner-joins it, so an episode without one would vanish quietly and get no reminders. Keep every create path in one statement (S19's procedure-based create, C7/C9), or add a deferred constraint trigger.
    4. **Comment nits:**
       - 00019's header and the migration test say "the seed puts back the demo data", but the seed has no episodes;
       - `GetEpisode`'s comment says the JSON is unchanged, but it gains `hqFormId`.
    5. **The HQ-form join is copied into about 9 queries.** Consider an `episode_hq` view when C7/C9 add form kinds.
  - **From the S19 manual test (20 Oct, minor; [test-report-1.md](sessions/S19-procedures/test-report-1.md)):**
    1. **The header overflows for a whole-trust admin below about 1280 px:** at 1126 px Sign out is off-screen (Ada: Questionnaires, Episodes, Procedures, Staff, Settings, Audit). S19's Procedures link adds about 114 px, but without it Sign out would still end at about 1140 px. A menu for the trust's admin pages (Procedures, Staff, Settings, Audit) would fix it.
    2. **The New episode page's Health questionnaire select isn't searchable** (older than S19). With many published HQs (1,243 locally, from the journeys) it's hard to use; the Settings page's Default HQ select is searchable.
    3. **(Cosmetic) After Edit on a lower row of Procedures, the form's heading sits under the fixed header** (test report 4): the form scrolls into view with `block: "nearest"`, which puts its top about 17 px from the window's top, under the 56 px header. Name and the fields below are visible. A `scroll-margin-top` of the header's height on the form would fix it.
  - **From the S19 verification (21 Oct, minor; [verify.md](sessions/S19-procedures/verify.md)):**
    1. **The Procedures page's Default HQ choices come from `…/h/{hid}/published-hqs`** (which includes the hospital's own questionnaires), but the API accepts only the trust's (`PublishedQuestionnairesInOrg`). Not reachable while questionnaires can only be the trust's; fix it before hospital-only HQs can be made (list the trust-wide set, or let a hospital-only procedure take its hospital's HQ).
    2. **New episode: a procedure picked before the HQ list has loaded loses its default HQ** (`hqForProcedure` sees an empty list; the hospital's default fills in). A window of milliseconds; re-apply the pick when the HQs arrive, or enable the Procedure select only once both have loaded.
    3. **`episodes.procedure_id` and `procedures.default_questionnaire_id` are single-column FKs**, unlike the repo's composite `(org_id, …)` tenancy FKs. Go checks the trust on every write; a follow-up migration (unique `(org_id, id)` on procedures and questionnaires, then composite FKs) would make the database enforce it too.
    4. **QA push:** 00020 drops `episodes.procedure`, so as with S18 the episode routes return 500 on QA between the Lambda deploy and the migrate step.
- **Incumbent import:**
  - Conditions that cross Question Sets on pages and questions are still dropped with a warning.
  - The incumbent has no Question Set-level conditions to import.

## 4. Not started: Must

| Area | Requirements | Notes |
| --- | --- | --- |
| Versions | LCY-01, 02, 06, 07 | Draft → In review → Approved → Published → Retired. Published versions are read-only, with "Create new version" that keeps stable IDs. |
| Review, sign-off, publish | SGN-01, 02, 04, 05 | The approver must not be an editor of that version. Publishing must be blocked by logic problems and failing test cases. Both checks already exist in the editor and preview. |
| Duplicate questionnaire | FRM-05 | |
| Code library screen | COD-01, 02, 04, 05 | Today codes can only be picked inside disclosures. |
| Export and import between installations | EXP-01, 02, 03 | One file with all codes, and promotion from QA to Training to Production. |
| Audit trail | AUD-01 | |
| Accessibility check | PX-02 (WCAG 2.2 AA) | SurveyJS mostly covers it, but nobody has verified it. |
| Answers recorded against a version | OUT-03 | Needed once real patients fill in forms. |

## 5. Not started: Should

- **Outputs:**
  - an all-outputs screen (CLN-13)
  - outputs from combinations of answers (CLN-04)
  - outputs limited by a condition on the answer (CLN-03)
- **Panels:** Allergies (PNL-05), and library items: saved groups (PNL-06).
- **Versions:** compare (LCY-08) and retire (LCY-09).
- **Reviews and reports:**
  - reviewer comments on items (SGN-03)
  - reports: outputs across questionnaires, where a code is used, and options with no output (RPT-01/02/03)
- **Hospitals:** master templates and adopting their updates (TEN-03/04), and per-hospital settings (TEN-05).
- **Respondent experience:**
  - print or PDF with outputs (EXP-06)
  - the review view (VEW-03)
  - resuming later (PX-05)
  - pre-filling answers (PX-06)
  - hospital branding (PX-07)
- **Validated scoring tools** (PNL-07, Could): STOP-BANG, PHQ-9 and others, with locked wording.
