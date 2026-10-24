# S27 build notes: the follow-ups after plan C4

Builder, automatic run (`/run-sessions`), 24 Oct 2026.

## Answers used

Rahul's answers are in [answers.md](answers.md) (24 Oct). They match the brief's recommendations:
1. **The card for another trust's answers** names its hospital ("We've filled in 9 answers from your questionnaire at Hospital B1 on 3 Mar 2026. Please check them."), keeps **Check each answer** and **Everything is still correct**, and adds **Start fresh**.
2. **Start fresh** clears the pre-filled answers in **that Question Set** (from any trust). It's recorded like the other buttons, and offered only until something in the set changes.
3. **On the patient's behalf:** staff get other trusts' answers too, with the same card and markers.
4. **S26's consent table and route** are removed with migration 00027, with the consent screen. Answers already copied stay. *(Not asked: the brief's default, since nothing is pushed.)*
5. **Clickable POA Summary answers:** each answer and disclosure link opens Validate for that Question Set, scrolled to the question and briefly highlighted. Back returns to the same place.

## What was built

### Go
- **Migration 00027** (`00027_prefill_without_asking.sql`):
  - drops `patient_prefill_consents`;
  - removes any stored `crossTrustPrefill` from `orgs.settings` and `hospitals.settings`;
  - lets `prefill_choice` be `start_fresh`.

  Down puts the table back, empty, and turns a `start_fresh` into `check_each`. The settings' old values can't come back; they were off by default.
- **`internal/settings`:**
  - `crossTrustPrefill` is gone, so the Settings page and the superadmin console no longer show it (both draw their fields from Go's definitions). A PATCH naming it is a 400, as for any unknown setting.
  - **The default Cancelled text and email are neutral:** "Hello {{firstName}}, {{hospital}} has cancelled your pre-operative assessment. Please contact the hospital if you have any questions." They're still off by default and editable. `episode.StatusMessage` needed no change: it only names the template.
- **Pre-fill from other trusts without asking** (`httpapi/prefill.go`):
  - `prefillForm` now reads the form's own trust, then **every other active trust where the person's account has a sent HQ** (`ListPrefillTrusts`, which replaces `ListPrefillOffers`). It does this for the patient **and for staff on their behalf** (answer 3).
  - Each trust is still read with its own `ListPrefillSources` call, by its `org_id`, and each read writes a `prefill.read` audit row under that trust.
  - The matching is unchanged (`internal/prefill`): the own trust first, a platform library key, otherwise exact wording, choices by option text, and no trust B id kept.
  - A suspended trust is still left out (`o.status = 'active'`).
  - `crossTrust`, `crossTrustOn`, `consentQuestion`, the `prefillConsent` handler and its route are gone, and the form's GET no longer sends `prefillConsent`.
  - The `prefill.read` label is now "Patient's answers read to pre-fill their HQ at another trust", without "(they agreed)". `link.prefill_consent`'s label stays, for the rows already written.
- **Start fresh** (`POST …/prefill/{cid}` `{choice: "start_fresh"}`, the patient's or the on-behalf route; audited as `link.prefill` / `behalf.prefill`, as the other buttons are):
  - **`StartFreshPrefill`** empties the set's patient row (`data = {}`), records the button (`by`: the staff member, or NULL for the patient) and moves the revision on. A tab still showing the answers therefore gets the stale 409 and can't save them back.
  - The `prefill` map stays, as the record of what was copied.
  - It's done only while the form is open, no button has been pressed, the row is at revision 1 (as Go filled it: nothing changed since), and the map holds another trust's answer.
  - **When it's refused**, `startFresh` says why (409):
    - "already sent";
    - "only for answers copied from another hospital's questionnaire";
    - "The answers in this section have changed since you opened it, so they weren't cleared. Please reload the page." (changed since, or another button pressed first).

    A second press of Start fresh is a 204, and changes nothing.
  - A set started fresh has a row, so it's never filled again. It counts as started (revision 2).
- **The archived View** (`patient.go` `loadForm`, `accounts.go`):
  - A patient's **GET** of a form that was **sent before the episode was archived** is allowed. It's the same read-only form as a sent one: `submitted: true`, so the page shows "Thank you…".
  - Every write (save, send, the card's buttons) is still a 403 with `code: "archived"`.
  - On the home, a sent questionnaire stays **Sent**, with **View**, whether archived or not. One archived before it was sent is still **Closed by the hospital**.
  - Staff's on-behalf form stays refused once archived (409): staff have the episode's own pages.
- **Format names in the publish refusal** (`internal/keys`):
  - `formatOf` names the editor's known patterns in words (`textFormatNames`): "phone number", "NHS number", "UK postcode". So the refusal reads "its answer format changed from any text to phone number". A custom pattern is still named by itself.
  - "NHS number" and "UK postcode" are hardcoded, the recorded A-20 exception (24 Oct).
  - **`TestKnownTextFormatsMatchTheEditor`** reads `packages/clinical/src/fields.ts` and fails if one of these patterns isn't there, so the two can't drift.
  - The same words now show in the "Same as…" refusal ("Its answer format is phone number…"). The lock and the wording match compare the same names on both sides, so nothing else changes.

### TypeScript (for Rahul to review)
- **`packages/clinical/src/prefill.ts`:**
  - `PrefillChoice` gains `"start_fresh"`.
  - **`prefilledAnswers(set)`** returns the set's pre-filled entries, or nothing once the patient chose Start fresh. Every answer there is then the patient's own, even one that happens to equal what was copied. Go keeps the map; this function decides what's shown.
  - `prefillChoiceText` for Start fresh: "The patient chose “Start fresh” on …, so the pre-filled answers were cleared and every answer here was given afresh." (or "Cara Clinician chose … with the patient …").
  - `poa.ts` marks only `prefilledAnswers`.
- **`apps/web/src/features/patient/prefill.ts`:**
  - **`cardText`** names another trust's hospital:
    - "…from your questionnaire at Hospital B1 on 3 Mar 2026…";
    - mixed with the trust's own answers, or from several hospitals: "…from your earlier questionnaires, including one at Hospital B1, the latest on …".
  - **`offersStartFresh(set)`**: some answer is another trust's, and no button has been pressed yet.
  - `shownPrefilled`, `firstUnfilled` and `showPrefillMarkers` use `prefilledAnswers`, so a set started fresh has no card and no markers.
- **`PatientHQ.tsx`:**
  - The consent screen is gone.
  - The card adds **Start fresh** (a subtle button) when `offersStartFresh` says so. It asks first, inline: "Clear these answers? You'll answer every question in this section again. This can't be undone.", with **Yes, start fresh** and **Keep the answers**.
  - Yes posts `start_fresh`, then loads the HQ again, so the set opens empty and its saver starts from the new revision.
  - If Go refuses, the reason shows at the top of the page, the confirmation closes, and the card stays.
  - On the patient's behalf the same card shows, worded for staff ("…from Sam's questionnaire at Hospital B1… Please check them with Sam.", "…again with Sam.").
  - `formatMonth` (`lib/format.ts`) was used only by the consent screen and is removed.
- **Clickable POA Summary answers** (`features/episodes/jump.ts`, `PoaSummary.tsx`, `ValidateSet.tsx`):
  - On both tabs, these are links to `…/episodes/:eid/sets/:cid?q=<question>`, with `state: {from: "poa"}`:
    - each disclosure note;
    - each Medication, Admissions or BMI table's title;
    - each pre-filled answer's question;
    - each "Changed by the clinician" line.
  - **Validate** reads `?q=` (`questionOf`). `highlightQuestion` opens the question's page of the set, waits for SurveyJS to draw it (a second at most), and scrolls it to the middle of the screen. It keeps it there for a second while SurveyJS settles its layout (as S25's `focusQuestion` does), and marks it `data-jumped` for 3 seconds: a yellow ring that fades (`globals.css`).
  - It's a data attribute, not a class, because React would overwrite the class on SurveyJS's next render. A question the answers hide isn't drawn; then the page alone shows.
  - **Back:** opened from the POA Summary, Validate shows a **← POA Summary** link beside the patient's name. It does `navigate(-1)`, the same as the browser's Back.
  - **The POA Summary's tab is in its URL** (`?tab=patient`, replaced rather than pushed), so Back and a reload keep it.
  - **The scroll position:** on a click, the link's id (`data-poa-anchor`: set, kind and question) and its distance from the top of the screen go into session storage (`saveSpot`). When the POA Summary loads again by Back (`useNavigationType() === "POP"`), `takeSpot` reads it once and `restoreSpot` scrolls until the link is at that distance again.
  - This was needed because React Router's own `ScrollRestoration` runs before the summary's data has loaded, so it lands at the top. Without storage (a private window) Back still works, from the top.
- **The NOT RESPONDING badge:** `notResponding(e)` (`status.ts`) is false once the episode is archived; `EpisodePage` uses it.

## Decided by the builder (automatic mode)
- **Start fresh is offered only on a card whose set holds another trust's answer** (answers 1–2 describe it as part of that card; decisions.md says "a pre-filled section from another trust … with a way to start fresh"). A same-trust-only card keeps its two buttons. Go enforces this too (409).
- **"Offered only until something in the set changes"** means: on the card, before any button is pressed, and only while the set is as Go filled it (revision 1). The card hides the survey, so in one tab nothing can change first. Go's check catches another tab or staff on the patient's behalf.
- **Start fresh asks for confirmation inline**, because it can't be undone (answers 2) and it clears clinical answers. "Nothing may break silently" (decisions.md §1). It's the same pattern as Close my account.
- **After Start fresh the `prefill` map is kept** (the record of what was copied), and the clinician sees one line instead of markers: "Answers were pre-filled here from the patient's earlier questionnaires. The patient chose “Start fresh” on …, so the pre-filled answers were cleared and every answer here was given afresh." The POA Summary shows no pre-fill box for such a set. It shows a set only when the set has something to summarise, as before.
- **The archived View is the same as a sent questionnaire's today:** the "Thank you, … Your answers have been sent" card, not a list of the answers. Showing the answers themselves would be new (the patient's View has never shown them). Only a GET is allowed; the patient's session must own the episode, as before.
- **On behalf, the `prefill.read` row under the other trust names the staff member** who opened the form (`actor: staff`, `user_id`: Cara). So Trust B's admin sees Trust A's staff member's name in Trust B's audit log: who read Trust B's answers. That's what an audit row should say.
- **The format names are a fixed Go map kept equal to `fields.ts` by a test**, rather than sending the editor's names to Go. Go can't import TypeScript, and the editor sends only the validators.
- **Clickable items:** "each answer … and each disclosure link" is read as everything on the POA Summary that names a question: disclosure notes, the capture tables' titles, the pre-filled lines and the clinician's changes.

## Not built (as the brief says)
The print-designed POA Summary PDF (C9); an account merge (C10); IG/DPO sign-off on copying across trusts (C10).

## Tests
- **Go unit:**
  - `internal/settings`: `crossTrustPrefill` is refused (unknown); the Cancelled defaults are neutral (`TestStatusMessagesAreOffByDefault`).
  - `internal/keys`: `TestKnownTextFormatsAreNamed` (the three names, a custom pattern, the refusal's words); `TestKnownTextFormatsMatchTheEditor`.
- **Integration:**
  - `TestPrefillWithoutAskingMigration` (`internal/apitest`):
    - the setting is dropped from both tables and the rest kept;
    - the table is gone and copied answers stay;
    - `start_fresh` is allowed, and an unknown choice isn't;
    - Down turns `start_fresh` into `check_each` and puts the table back.
  - `TestPrefillCrossTrust` (rewritten):
    - another trust's answers are copied at the first open, with no setting, no question and no `prefillConsent`;
    - by wording, with the hospital and questionnaire, and no `fromForm`;
    - no trust B id in trust A's rows or in the staff `…/hq`;
    - a `prefill.read` row under trust B with the new label, and trust A's rows naming only trust A's records;
    - a second open reads no more, and another episode copies again;
    - an old stored `crossTrustPrefill: false` changes nothing;
    - the platform library key; own trust first, and nothing typed overwritten;
    - **on behalf:** copied, and audited under trust B as Cara;
    - a suspended trust isn't used;
    - the consent route is gone (404/405).
  - `TestStartFresh`, the five cases and more:
    - no sign-in: 401;
    - wrong role: another patient, or staff at the other hospital: 404;
    - another trust's staff: 404;
    - invalid input:
      - a same-trust-only set: 409 "another hospital";
      - a set changed since: 409 "reload", nothing cleared;
      - after Check each answer: 409;
      - an unknown choice: 400;
    - happy path, patient:
      - empty, revision 2, recorded, map kept;
      - not filled again; the other set kept; In progress on the home;
      - a second press is a 204 with no change;
      - a stale tab's save of the old answers: 409;
      - audited;
    - happy path, staff: recorded as Cara;
    - once sent: 409, and nothing cleared.
  - `TestArchivedAfterSentKeepsItsView`: the home says sent (and closed for one archived unsent); the GET returns the answers, `submitted: true`; save, send, the card's button and the unsent one's GET are 403 `archived`; the staff on-behalf form is a 409.
  - `TestArchivedEpisodeIsReadOnly`: changed. Its episode was sent (Ready for review), so the patient's GET is now 200; the writes are still refused.
  - `TestSettings`: a trust PATCH of `crossTrustPrefill` is a 400.
  - `TestPrefill`: its "another trust's answers are never used" case is removed (now `TestPrefillCrossTrust`). `TestPrefillConsentRoute` is removed with the route.
- **Vitest:**
  - `packages/clinical/src/prefill.test.ts`: Start fresh's text; `prefilledAnswers`; no POA marks after Start fresh.
  - `features/patient/prefill.test.ts`:
    - the card names the hospital: alone, for staff, mixed and several hospitals;
    - `offersStartFresh`;
    - after Start fresh, no card, no markers, and firstUnfilled is the first question.
  - `features/episodes/jump.test.ts`: `validateAt` and `questionOf`; `saveSpot` and `takeSpot` per episode, taken once, with no storage, broken storage or rubbish.
  - `features/episodes/status.test.ts`: `notResponding`.
  - `lib/format.test.ts`: `formatMonth`'s case is removed with the function.
- **E2E:**
  - **J16b, rewritten** (phone size):
    - no question; both sets Please check;
    - Your health's card names Hospital B1 and offers Start fresh, with Check each answer and the markers;
    - About you: Start fresh, Keep the answers backs out, then Yes, start fresh empties it (no card, no markers), still empty after a reload;
    - the API shows the hospital, how each answer matched, no `fromForm`, and `start_fresh` with `{}`.
    - Its second test: a refused Start fresh shows Go's reason and keeps the card.
  - **J9:** after Sue archives, the patient's phone opens the link to the "Thank you" View (no Send), and the home says Sent with View, not "Closed by the hospital"; View opens it.
  - **J5's second test** (a 1280 × 420 window, so the answers are below the fold):
    - a validated disclosure, the clinician's change and the patient's answer each open Validate at "Do you smoke?", `data-jumped` and in view, with the POA Summary link;
    - the browser's Back and that link each return to the same tab with the link clicked on screen;
    - `?tab=patient` survives a reload.
    - **Mutation check:** with `restoreSpot` switched off, it fails at "in viewport" after Back (run by hand, then put back).

## Results
- `make lint`: passed.
- `make test`: Go all ok; Vitest 226 (web) + 152 (clinical); CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` on my own instance (database `sj_s27` migrated to 00027 and seeded; API :8097, Vite :3107; both stopped afterwards): **65 passed, 5 skipped, 0 failed**.
- **The local `sj` database is still at migration 00026.** The conductor's restart (`make migrate seed`) takes it to 00027, which drops `patient_prefill_consents` and any stored `crossTrustPrefill`. No reset is needed, and HJE Full HQ is untouched.

## Suggested commits
1. `S27: pre-fill from other trusts without asking (crossTrustPrefill and S26's consent question, route and table removed, migration 00027; for the patient and on their behalf; the card names the hospital and adds Start fresh), a sent-then-archived questionnaire keeps its read-only View, no Not responding badge once archived, neutral Cancelled message, format names in the publish refusal, clickable POA Summary answers`, with its tests:
   - Go unit (`internal/settings`, `internal/keys`);
   - integration (`TestPrefillWithoutAskingMigration`, `prefill_cross_trust_integration_test.go` rewritten with `TestStartFresh`, `TestArchivedAfterSentKeepsItsView`, `archive_integration_test.go`, `settings_integration_test.go`, `prefill_integration_test.go`);
   - Vitest (`prefill.test.ts` in clinical and patient, `jump.test.ts`, `status.test.ts`, `format.test.ts`);
   - e2e (J16b rewritten, J9, J5's second test; README rows).
2. `docs: S27 …` (build notes, test plan and reports, decisions, pending, requirements, CLAUDE.md, AGENTS.md, session status) once verified.

## Test report 1 (24 Oct): 19 of 21 passed

1. **F1, T-07: staff filling in found no other trust's answers** when the patient hadn't yet entered their date of birth on the Trust A link.
   - **This is the designed behaviour, not a bug.** Other trusts' records are found only through the person's sign-in account (S15). A trust's record joins the account only when the patient proves their date of birth on one of its links.
   - Finding them for a record that hasn't joined would mean matching people across trusts by typed details alone. That's a new identity and privacy decision, which the brief and answers don't make. Answer 3 says staff get other trusts' answers "too", that is, the same ones the patient gets. The patient's own open always comes after their date of birth, so it always has the account.
   - The tester's control run (the patient joined first, then Cara filled in) passed every step of T-07.
   - **Changes:**
     - the test plan's T-07 now has the patient join first;
     - a new step checks that an unjoined record gets nothing;
     - a new integration case pins it: `TestPrefillCrossTrust` "on behalf, a record that hasn't joined the person's account finds no other trust's answers";
     - recorded in pending.md for Rahul, in case he wants on-behalf pre-fill for patients who haven't signed in at this trust.
2. **F2, T-13: the Cancelled email's subject still said "your health questionnaire was cancelled".** It's hardcoded in Go (`invite.SendStatus`), not in the editable template.
   - **Fixed, test first:** "<hospital>: your pre-operative assessment was cancelled". Test: `TestStatusMessages` "the cancelled email's subject is neutral too, from any status".
3. **Exploring notes:**
   - **Not changed:** a refused save is audited as "Patient saved answers" with Result 409 (older than S27); the "NEW QUESTION: WON'T PRE-FILL" badge on published questions (S23's label, not S27's change).
   - **Expected:** answers fill an open form after its trust is re-activated (no row was written while suspended).

**Results after the fixes:** `make lint`, `make test` (Vitest 226 + 152, CDK 124) and `make test-integration` pass. No page changed, so e2e wasn't re-run.

**Retest:** T-07 (the corrected steps) and T-13 (the subject).

## Test report 2 (24 Oct): all passed
- **10 of 10:**
  - T-07 with its corrected steps, and its "not joined" step;
  - T-13 with the neutral subject;
  - T-01/T-02, T-03 and T-15 re-run;
  - R-01 to R-05.
- **Exploring notes:** recorded in pending.md ("From S27's test report 2"), with no code changed, as the session's scope stays put:
  - **T-07's read is the patient's, not Cara's.** The patient's own open after their date of birth already copies Trust B's answers, so T-08's "by Cara" needs a person joined through an earlier Trust A episode. The test plan has a note, and the tester confirmed a real on-behalf read is attributed to Cara.
  - **The sticky staff top bar can hide a jumped-to question's title** on desktop.
  - **The patient home's older cancelled-card wording** for a questionnaire already sent.
  - **The POA Summary's BMI shows "—"** before validation.
- **S27 is tested**, for verification. There's no plan of its own, so the only status change is the sessions table: **Tested**.

## Hand checks for Rahul
None. Every case was run in the browser. Print was rendered to PDF and the CSV checked from `~/Downloads`, as the tester rule says.

## Suggested commits (final)
1. `S27: pre-fill from other trusts without asking (crossTrustPrefill and S26's consent question, route and table removed, migration 00027; for the patient and on their behalf; the card names the hospital and adds Start fresh), a sent-then-archived questionnaire keeps its read-only View, no Not responding badge once archived, neutral Cancelled message and subject, format names in the publish refusal, clickable POA Summary answers`, with its tests:
   - Go unit (`internal/settings`, `internal/keys`);
   - integration:
     - `TestPrefillWithoutAskingMigration`;
     - `prefill_cross_trust_integration_test.go` rewritten, with `TestStartFresh` and the unjoined on-behalf case;
     - `TestArchivedAfterSentKeepsItsView` and the `archive_integration_test.go` change;
     - `TestStatusMessages`' subject case;
     - the `settings_integration_test.go` and `prefill_integration_test.go` changes;
   - Vitest (`prefill.test.ts` in clinical and patient, `jump.test.ts`, `status.test.ts`, `format.test.ts`);
   - e2e (J16b rewritten, J9, J5's second test; README rows).
2. `docs: S27 tested — build notes, test plan and reports, decisions, pending, requirements, CLAUDE.md, AGENTS.md, session status`.

## Verification 1 (24 Oct): fixes

All test first.
1. **Must fix: after Start fresh, a stale card's other button was accepted** (204, "the first button stays"). A set where every question shown was pre-filled could then be sent empty, though the sender saw it as checked.
   - **Fix:** once the recorded choice is `start_fresh`, Check each answer and Everything is still correct get a 409: "The answers in this section were cleared with “Start fresh” somewhere else, so there's nothing here to check. Please reload the page." The card shows Go's message and stays.
   - **Test:** `TestStartFresh` "after Start fresh, a stale card's other buttons are refused…", covering both buttons for the patient, and staff on the patient's behalf after the patient started fresh.
2. **"…including one at Hospital B1" when every answer came from Hospital B1** (several dates).
   - **Fix:** `cardText` now says "…from your earlier questionnaires at Hospital B1, the latest on …".
   - **Test:** a Vitest case.
3. **Start fresh could show on a card naming no other hospital** (another trust's answer on a hidden question).
   - **Fix:** `offersStartFresh(prefill, shown)` decides from the answers the card counts (`shownPrefilled`, as `cardText` gets them). Go's own check is unchanged.
   - **Test:** a Vitest case with a hidden `otherTrust` entry.
4. **The 400 for an unknown choice** now names all three buttons (`TestStartFresh` checks for "Start fresh").
5. **Notes:** recorded in pending.md ("From S27's verification 1"):
   - every open re-reads other trusts;
   - the badge on a cancelled episode;
   - "Custom pattern" against Go's name for it.

**Results after the fixes:**
- `make lint`: passed.
- `make test`: Go all ok; Vitest 227 (web) + 152 (clinical); CDK 124 passed.
- `make test-integration`: every package ok.
- `make e2e` on my own instance (`sj_s27`, :8097/:3107, stopped afterwards): **65 passed, 5 skipped**.

**Retest:** findings 1–3 (the test plan's note at the top), plus T-01/T-02 and T-07 as regression.

## Test report 3 (24 Oct): all passed
- **5 of 5:**
  - V-1, for the patient's tab and for Cara's Fill in: a stale card's other buttons get the 409 message, and the set is empty after a reload;
  - V-2: "…at Hospital B1…", with no "including";
  - T-01, T-02 and T-07 as regression.

  V-3 is automated only.
- **Notes, nothing changed:**
  - **V-2 can't give two dates by hand.** A sent HQ freezes the answers copied into it, so the newest form holds them all. The two-date wording is unit-tested only, and the test plan now says so.
  - **The network log's 503s for successful 204s** are Claude in Chrome's known artifact (decisions.md §4).
  - **The sticky header over a jumped-to question** is already in pending.md (test report 2).
  - **A stale Start fresh after Start fresh elsewhere** is a quiet 204: the result is what was asked for.
- **S27 is tested again**, for verification 2.

## Verification 2 (24 Oct): fixes

1. **Must fix: `test_audit.py::test_another_trusts_admin_sees_nothing_of_trust_a`** failed on the conductor's database.
   - **Why:** it expected Cara to be missing from Trust B's **Who** filter. S27's on-behalf `prefill.read` rows put her there, as agreed. The journey passed on my fresh `sj_s27` only.
   - **Fix (in the test):**
     - it keeps the surname filter's "Nothing matches these filters.";
     - Cara also opens the on-behalf form for her new patient, who has no Trust B history;
     - it reads Trust B's audit CSV, and asserts that neither this episode's id nor its record's id is in it;
     - it reads Trust A's CSV and asserts the episode is there, so the check isn't vacuous.

     It now depends only on its own data.
2. **Other finding 1 (fixed, Vitest first):** when every answer came from two or more other hospitals, the card said "including ones at …". It now says "…from your earlier questionnaires at Hospital B1 and Hospital C1, the latest on …". "Including" stays for a mix with this trust's own answers.
3. **Other findings 2–5 and the note for Rahul:** recorded in pending.md ("From S27's verification 2"), with no change:
   - the whole-HQ reload after Start fresh;
   - the POA Summary's line for a set started fresh (Rahul to judge);
   - Go accepting a hidden-only Start fresh;
   - the way back's cmd-click and closed accordions;
   - Trust B's log naming Trust A's staff, for the IG/DPO review.
4. **Gotcha, recorded in the S27 decisions entry:** run the whole e2e suite on the shared dev database, not only a fresh one.


**Results after verification 2's fixes:**
- `make lint`: passed.
- `make test`: Go all ok; Vitest 227 (web) + 152 (clinical); CDK 124 passed.
- `make test-integration`: every package ok.
- **The whole `make e2e` suite against the conductor's running `make dev`** (its `sj` database, with the testers' cross-trust data): **65 passed, 5 skipped, 0 failed**. Not stopped or restarted.
- **The whole suite on my own fresh instance** (`sj_s27`, :8097/:3107, stopped afterwards): **65 passed, 5 skipped**.

**Retest:** none needed by hand (a journey and a unit-tested wording change). T-01/T-02 is the regression if the conductor wants one.

## Test report 4 (24 Oct): all passed
- **3 of 3:** T-01 and T-02 as regression, and the card wordings the seeded trusts can reach:
  - one other hospital: "…at Hospital B1…", with no "including";
  - this trust only: no hospital named, and no Start fresh.
- **Note, nothing changed:** the mixed "including one at …" wording needs Trust A history from a questionnaire that asks fewer questions than Trust B's, and none is seeded. It and the two-other-hospitals wording are unit-tested only; the test plan says so.
- **S27 is tested again**, for verification 3.

