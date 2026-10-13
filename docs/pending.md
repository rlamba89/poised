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
- **Statuses:** admitted, discharged, recovery and archive are not built yet.
- **Patients:**
  - accounts, and invitations by email or SMS
  - patient search
  - Short HQ
- **Re-opening:** a review can't be re-opened after it is complete.
- **Two simultaneous episode PATCHes can overwrite each other's ASA grade** (found by S02a's verification, 9 Oct; older than S02a).
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
| Real HJE Full HQ export | Today's copy has only Question Sets 1–4, transcribed from screenshots, with no disclosures. Run `tools/export-lifebox-hq.js` in the Lifebox Author tool and import the file with `go run ./cmd/import-lifebox` to get all 11 sets and their disclosure codes (EXP-04). |
| Saved option lists: copied or linked? | Built as copies: a question keeps its options when the list changes, which protects stable IDs (OPT-02). Confirm that's what's wanted. |

## 3. Small gaps in what's built

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
    - `reloadOnce` (`apps/web/src/lib/chunkError.ts`) allows one reload per 10 seconds, so on a connection where each failed load takes longer than that, it reloads forever. Clear the flag only after a page renders, or widen the window.
    - Keep `/assets/*` out of Amplify's SPA rewrite, so a missing file is a real 404. Otherwise Safari's "'text/html' is not a valid JavaScript MIME type" isn't recognised. Alternatively, add Vite's `vite:preloadError` listener as a backstop.
  - ~~**The error screen's "Go to the start page" sends a patient to the staff sign-in** (`RouteError.tsx`; `/` → `/login` when not signed in). Hide it on `/p/…` with S14/S15.~~ *Done in S15: on a patient's page it's "Go to your questionnaires" (`/home`).*
  - **Small:**
    - the staff `HospitalLayout` is in the first-load bundle, so the patient's link downloads it (make it lazy);
    - the Vite proxy key `/api` also matches `/api-…` (use `^/api/`);
    - in dev, StrictMode flashes "Something went wrong" just before the automatic reload.
- **From S05's verification, 12 Oct** (minor; details in [S05 verify.md](sessions/S05-lambda-entrypoints/verify.md)):
  - **Answer dates in notes, outputs and the POA Summary are still day/month/year** (`packages/clinical/src/outputs.ts` `formatDate`, `yyyy-MM-dd` → `dd/MM/yyyy`). The guard test can't see it (no locale literal). Fine while only the UK runs; **Rahul to decide** before a second country whether `packages/clinical` takes the locale (recommended: pass it in, as `format.ts` does).
  - **`DEFAULT_LOCALE` is checked only for shape,** and some well-shaped tags (`en-12`, `en-gb-gb`) make the browser's `Intl.DateTimeFormat` throw, so every date on a page would crash it. Tighten the region part to `[A-Z]{2}` or three digits, or check the tag in `infra/countries.py` (S06).
  - **The first render waits for `GET /api/config` with no timeout** (`apps/web/src/main.tsx`). A hung request leaves a blank page. Add a timeout (e.g. `AbortSignal.timeout(5000)`) before S06 (Lambda cold starts).
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
  - **S06 must make the rate limit's client IP the real patient's** (`clientIP` in `accounts.go` reads `RemoteAddr`). Behind API Gateway and Amplify's rewrite that is a proxy's address unless the Lambda adapter sets it from the request context (or a trusted `X-Forwarded-For`): otherwise every patient shares one address, and 10 sign-ins an hour would be the whole country's.
  - **S14's X-1 is partly better:** leaving the form by **← Your questionnaires** or the browser's Back to the home is an in-app route change, so a waiting autosave is sent; a reload or closing the tab within ~0.6 s still loses it.
  - **An account keeps the date of birth proved when it was made.** If a hospital later corrects a record's date of birth, sign-in with the corrected date fails for that account (invites still work). A staff correction could update the account later.
  - **A patient can change or add a mobile or email, not remove one.**
  - **A first-time patient (never opened an invite) can't use `/signin`:** by design (PAT-06 is the returning patient's), and the screen says to use the hospital's link.
  - **Sign-in links' wrong dates of birth, and their joins when no record changed, are in no trust's log** (no trust); a security log may want them (as S14's probe item).
  - **A save that fails on the network shows the patient the raw "TypeError: Failed to fetch"** (S15 test report 4, exploring 1; S14's save-error display, also on the ordinary autosave). The page does stay with **Not saved**, and leaving later re-sends the change. Friendlier wording: "We couldn't save your latest change. Check your connection and try again." (`api.ts` could turn a network failure into that message.)
  - **A double tap on "← Your questionnaires" right after typing sends the save twice** (S15 test report 4, exploring 3, cosmetic): both 200, the revision moves by 2, the stored answer is right. Disabling the link while it saves would stop it.
  - **Accepted risk (Rahul, 13 Oct, S15 answers 10; verification 1, finding 1):** a person sharing a contact and date of birth with an account holder, whose own record isn't joined yet (an unjoined twin), signs in to the **other** person's account with a sign-in link, because the name step runs only when two or more accounts match. Rahul: too rare to change sign-in for. If it's ever revisited, the verifier's fix: ask for the name also when an unjoined record with the link's contact and that date of birth has another name (`signinDob`, `AccountsForSignin`).
  - **Migrations 00014 and 00015 aren't safe for a database with real accounts** (S15 verification 1, finding 3): they unjoin records merged under the old rules but leave the contacts those records filled into the account (which then still signs in to it), and leave open patient sessions running. Only local and test databases (made-up data) have run them. If a QA or real database ever needs them, a later migration should rebuild each account's contacts from its remaining records and end its open sessions.
  - **For S17: `POST /api/p/signin` gives away whether an account exists when sending fails or is slow** (S15 verification 1, finding 4): a provider failure is a 502 only for contacts with an account, and only they wait for the send. Answer 202 regardless (log the failure), or send after answering or from a job (S16). Harmless while capture mode never fails.
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
  - **"Not responding" is a badge on the episode only**; the worklist (EPI-02) is C3's.
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
  - **The patient cookie has no `Secure` flag locally** (http); it's set when `APP_URL` is https. S06 should check it on QA.
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
- **Lifebox import:**
  - Conditions that cross Question Sets on pages and questions are still dropped with a warning.
  - Lifebox has no Question Set-level conditions to import.

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
