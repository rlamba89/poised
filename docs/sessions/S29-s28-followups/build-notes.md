# S29 build notes: the follow-ups after S28

Builder, automatic run (`/run-sessions`), 25 Oct 2026.

## Answers used

[answers.md](answers.md): the one question (which sets a read after a later join may fill) was answered "neither": nothing is built for a later join, and S28's "once per form" stands. Everything else is the brief as written.

## What was built

### Go
- **An audit action for each opening-card button** (`prefill.go` `prefillChoice`, `audit.go`): once the body's `choice` is valid, the request's audit row's action becomes `<route action>_<choice>`:
  - staff: `behalf.prefill_check_each` "Checked each pre-filled answer with the patient", `behalf.prefill_still_correct` "Confirmed pre-filled answers with the patient", `behalf.prefill_start_fresh` "Started a Question Set fresh for the patient";
  - patient: `link.prefill_check_each` "Patient chose to check each pre-filled answer", `link.prefill_still_correct` "Patient confirmed pre-filled answers", `link.prefill_start_fresh` "Patient started a Question Set fresh".

  The old `behalf.prefill` and `link.prefill` labels stay for rows written before S29 (append-only), and for a request refused before the choice is read (unknown set, bad JSON, unknown choice).
- **The cancelled message on an archived-from-Cancelled form** (`patient.go` `loadForm`; `GetEpisodeForm` now selects `e.archived_from`, sqlc regenerated): the patient's form for an episode archived from Cancelled answers 403 `formCancelled` with `code: "cancelled"`, for every method, sent or not (as their home already says "Cancelled by the hospital", with no View). Archived from any other status keeps `formArchived` (and a sent one its read-only View, S27). Staff are unchanged: the on-behalf form of any archived episode is the archived 409.
- **"Same as…" says what a question is already linked to** (`keys.LinkedTo`; `question_keys.go` `searchQuestionKeys`): the search's response has `linkedTo`, the text of the question the question's own key already belongs to, from the library if the key is there, else the trust's newest published or retired version holding it (the same rule as a candidate's title); `""` when no question that takes part in pre-fill has that key (a new question).

### TypeScript (for Rahul to review)
- **`packages/clinical/src/poa.ts` `reportedBmi(sets, answersOf, patient)`:** now takes the patient (age, sex: the POA Summary's `vars`), and reads only answers that are shown. It runs `shownSets` (a Question Set's condition, from the answers in use), then for each shown set builds the SurveyJS model with those answers and keeps only the questions `getAllQuestions(true)` returns (it skips questions hidden by their own condition and those in a hidden page or group). Same answers as before: the clinician's once the set is validated, else the patient's.
- **`PoaSummary.tsx`:** passes `vars` to `reportedBmi`.
- **`apps/web/src/lib/api.ts`:** `NETWORK_ERROR` = "We couldn't reach the server. Check your connection and try again." `api` throws `ApiError(0, NETWORK_ERROR)` when `fetch` itself fails (an abort stays an abort), and `ApiError(status, NETWORK_ERROR)` for a 5xx with no JSON body (a gateway's, e.g. the dev proxy's empty 502). A JSON body's `error` is always kept; a 4xx with no JSON is still "Something went wrong. Please try again."
- **`PatientHQ.tsx`:** the red error banner is hidden while the "The answers in this section were cleared" card shows.
- **`tasks.ts` `openTaskCount(status, open)`:** null once archived. `EpisodeList.tsx` (the row's "N open tasks") and `TasksPanel.tsx` (the Tasks title's "(N open)") show nothing then; the tasks stay listed.
- **`library.tsx` `SameAsWindow`:** keeps `linkedTo` from each search and shows an orange line under the window's introduction: "This question is already linked to “<text>”. Using this replaces that link." (`data-testid="same-as-linked"`). Reset each time the window opens.

## Decided by the builder (automatic mode)
- **There is no "What filter" on the Audit page.** The brief says "the audit page's What filter offers the new actions", but the page's filters are Who, From, To and Patient; "What" is a column. Adding an action filter would be a new feature, so the new labels show in the **What** column (and the CSV's Action column) only. Say if an action filter is wanted.
- **The action names the button pressed**, even when that press records nothing (a second button in another tab gets 204 and keeps the first choice): the row says what the person did.
- **"Archive a validated one: the closed message" (manual test focus 4):** a validated episode was sent, so once archived its form opens read-only (S27's View), with no closed message; the closed message shows on any change (and on an unsent archived one). The test plan checks the View for the validated one and the closed message on an unsent one archived from HQ not complete.
- **Archived from Cancelled, sent:** cancelled message, no View, matching the home (which shows "Cancelled by the hospital" for it, not Sent with View).
- **Which questions are "shown" for the BMI:** the clinician's view of the set (as on the validated summary), with the answers in use. Clinician-only questions count there; the reported BMI comes from patient questions in practice.
- **"Same as…" warns for a key from this questionnaire's own published versions too:** a question in a new draft that was in v1 already pre-fills by its key from v1's answers, so replacing the key changes that link as well. A new question (no published or library question has its ID) shows nothing.
- **The warning is a line in the window, not a second confirmation step** (the brief: "the window says … before Use this"). The existing "what will be lost" confirmation is unchanged.
- **The Tasks title is in `TasksPanel.tsx`** (the brief named `EpisodePage.tsx`, which renders it).

## Tests

### Go
- `TestPrefillChoiceAuditActions` (new, integration): each of the three buttons, by the patient and by Cara for them, writes one row with its own action and status 204, none with the old action; Trust A's admin reads each label in words (`GET /audit`).
- `TestPrefillChoice` and `TestStartFresh` (changed): their audit checks name the new actions. Their five cases (no sign-in, wrong role, another trust, invalid input, happy path) still cover the unchanged routes.
- `TestArchivedFromCancelledFormSaysCancelled` (new, integration): archived from Cancelled, unsent and sent, GET, save and send all answer 403 `code: "cancelled"` with the cancelled message; archived from Ready for POA (sent), GET is the View and a save is `code: "archived"`, "closed"; staff still get the 409.
- `TestSameAsSearch` (changed, integration): `linkedTo` is "" for a new question, "Do you smoke?" for a question with a published key (`q_smokes`) whatever its own text, the library's text for a library key, and "" for another trust's library key.
- `TestLinkedTo` (new, unit, `internal/keys`): the library's wording wins; a published-only key; a repeating group; "" for a new key, a clinician-only question, and a group's row question.

### TypeScript (Vitest)
- `poa.test.ts`: `reportedBmi` skips a height its condition hides, before and after validation, and counts it once shown; a hidden BMI question, page and group; a hidden Question Set, going on to the next shown one.
- `lib/api.test.ts` (new): a failed fetch and 500/502/503/504 with no JSON give the plain message; a JSON `error` (a 500's included) and its body are kept; a 4xx without JSON is the general message; 204 is null; an abort stays an abort.
- `tasks.test.ts`: `openTaskCount` counts, and is null once archived.

### End-to-end (Playwright)
- **J5** (`test_j5_poa_summary.py`), new fourth test: with the HQ request aborted, the POA Summary shows "We couldn't reach the server…" and no "Failed to fetch"; back online, a comment meeting an empty 502 shows the same message.
- **Audit** (`test_audit.py`), new test: Cara starts Your health fresh for a patient with a Trust B history; Ada's log, filtered by Cara and the surname, has one "Started a Question Set fresh for the patient" row and no "Checked pre-filled answers with the patient".

### Results (25 Oct)
- `make lint`: clean.
- `make test`: Go all ok; Vitest 240 passed (web, 23 files) and 161 passed (clinical, 19 files); CDK 124 passed.
- `make test-integration`: all packages ok.
- `make e2e` (against `make dev`, then stopped): **71 passed, 5 skipped** (the Cognito journeys, QA only).

## Test rounds
- **Test report 1:** blocked by Chrome before any case ran (the conductor said to ignore it). No code change.
- **Test report 2 (25 Oct): all passed, 17 of 17** (T-01 to T-11, R-01 to R-06), the tester overriding `window.fetch` for T-05 and T-06. No code change. Test plan corrected: T-11 step 6 and R-05 named the pregnancy question, which is Always ask fresh, so its label says "Always ask fresh" rather than "Pre-fills from N other HQs" (as designed); they now use "Do you snore loudly?".
- **Exploring notes (no change):** a hidden height held in the patient's stored answers (sent through the API) gives "—" before validation too; the plain message also shows on page loads; the known double-click on Save comment and the squeezed Use this button are already in `pending.md`; the patient's cancelled, closed and not-found pages show no hospital name at the top right (older than S29); "Everything is still correct" showed the first page when the next unanswered question was on it (as designed).

## Hand checks for Rahul
- **A gateway error by hand:** stop only the Go API while Vite runs (so `/api` answers the proxy's empty 502), and use a staff page: it should say "We couldn't reach the server…". Automated in J5 (a routed 502) and the `api` unit tests; a browser tester can only fake a failed `fetch` (see the test plan's T-05 and T-06).

## Suggested commits
1. `S29: Patient reported BMI from shown answers only` — `packages/clinical/src/poa.ts`, `poa.test.ts`, `PoaSummary.tsx`.
2. `S29: an audit action for each opening-card button` — `audit.go`, `prefill.go`, `prefill_integration_test.go`, `prefill_cross_trust_integration_test.go`, `e2e/journeys/test_audit.py`.
3. `S29: plain network-error wording, and no red banner over the cleared card` — `apps/web/src/lib/api.ts`, `api.test.ts`, `PatientHQ.tsx`, `e2e/journeys/test_j5_poa_summary.py`, `e2e/README.md`.
4. `S29: no open-task count on archived episodes` — `tasks.ts`, `tasks.test.ts`, `EpisodeList.tsx`, `TasksPanel.tsx`.
5. `S29: the cancelled message on an archived-from-Cancelled form` — `db/queries/episodes.sql`, `internal/db/episodes.sql.go`, `patient.go`, `archive_integration_test.go`.
6. `S29: "Same as…" warns before replacing a question's link` — `keys.go`, `keys_test.go`, `question_keys.go`, `library_integration_test.go`, `library.tsx`.
7. `docs: S29 build notes, test plan, AGENTS.md` — this folder, `apps/web/AGENTS.md`, `docs/sessions/README.md`.
