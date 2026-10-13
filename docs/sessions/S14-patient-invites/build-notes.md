# S14 build notes: patient invites and magic-link sign-in

Builder, automatic run (`/run-sessions`), 13 Oct 2026. Rahul's answer is in [answers.md](answers.md): before the date-of-birth check the patient API returns only the hospital's name; after it, the HQ without the date of birth or any identifier the patient didn't type.

## What was built

### Data (migration `00011_patient_invites.sql`; up, down, up checked on the local database)

- **`login_links`:** `token_hash` (SHA-256 of a 32-byte random token; **the token itself is never stored**), `kind` (`invite`; `signin` is allowed for S15), `patient_id`, `episode_id`, `created_by`, `created_at`, `expires_at` (NULL for an invite), `used_at` (the last right date of birth), `revoked_at` (replaced by a resend), `failed_dob_attempts`, `locked_at`.
- **`episodes.patient_token` is dropped** (the old `/p/:token` link, which opened the HQ by itself). Existing local episodes have no link until staff press **Send invite / Resend invite**.
- **`episodes.patient_submitted_by`:** the staff member who sent the HQ on the patient's behalf (PAT-09); NULL when the patient did.
- **`sent_messages`:** every captured text and email (MESSAGING=capture).

### Go

- **`internal/messaging`:** `SMS` and `Email` interfaces; **`Capture`** (stores in `sent_messages`, sends nothing) and **`Recorder`** (the test fake). `httpapi.Deps` gained `SMS` and `Email` (a missing one panics at start-up, like Cognito). `internal/serve` wires `Capture` for both.
- **`internal/config`:** **`MESSAGING`**, default `capture` locally and with `CONFIG_STRICT=true`; any other value stops start-up ("real sending comes with S17"). `GET /api/qa/messages` exists only in capture mode.
- **`internal/auth/patient.go`:** the patient's session cookie: an HS256 token with audience `patient` naming the **session** (a `patient_sessions` row, added after test report 1; it first named the patient record), valid 30 minutes (`PatientIdle`). **Each request issues it again** (a sliding idle timeout). Cookie `poised_patient`, `HttpOnly`, `SameSite=Strict`, `Path=/api/p` (never sent to the staff API), `Secure` when `APP_URL` is https. The staff `Verify` now refuses any token with an audience, so neither token works as the other. `NewLinkToken` and `HashLinkToken` (replacing `episode.NewToken`).
- **`internal/patient.SMSAllowed`:** PAT-11: a text goes only to a number with the deployment's country calling code (`PHONE_COUNTRY`).
- **`internal/settings.Fill`:** fills a message template's `{{firstName}}`, `{{hospital}}`, `{{link}}` in one pass (a name typed as `{{link}}` stays as typed). **The default invitation text is now PAT-03's wording:** `{{hospital}}: please complete your pre-op health questionnaire: {{link}}`.
- **Invites (`internal/httpapi/invites.go`):**
  - **Creating an episode sends the invite** after the episode is saved: a new link (earlier ones of the episode revoked), the hospital's effective `templates.inviteSms` / `templates.inviteEmail` (S10 settings), the link `APP_URL/i/<token>`, by SMS (if the mobile may be texted) and by email (if there is one). Email subject: `<hospital>: your health questionnaire`.
  - **What happened is written to the episode's General notes**, by the staff member: "Invite sent by text message and email." / "…by email. No text message: texts go only to mobile numbers in this country." / "No invite sent. The patient has no mobile or email." / "No invite sent. The text message couldn't be sent. The email couldn't be sent." **A failed send never stops the episode being created.**
  - **`POST …/episodes/{eid}/resend-invite`** (`invite.resend`, audited): a new link by both, replacing the old one (which also clears a lock). 409 once the answers are sent, or when there's no mobile that can be texted and no email; 502 if every message failed (noted in General notes).
  - **`GET /api/qa/messages?to=`:** the newest 100 captured messages whose recipient contains `to` (all when empty). No sign-in; capture mode only.
- **The patient's routes (`internal/httpapi/patient.go`, patient router):**
  - `POST /api/p/links/{token}/continue` (`link.open`): **only `{hospital}`**; changes nothing. 404 for a bad, unknown or replaced link; 403 `{code: "locked"}` for a locked one; 403 while the trust or hospital is suspended. **There is no GET on a link.**
  - `POST /api/p/links/{token}/dob` (`link.dob`) `{dateOfBirth: "YYYY-MM-DD"}`: right → the session cookie and `{episodeId}`, attempts reset to 0; wrong → 401 "…You have N tries left."; **the 5th wrong one locks the link** (403 locked), and then even the right date is refused. A date that can't be read is a 400 and not a try. The count and lock are single SQL updates (no race).
  - With the session (`requirePatient`): `GET /api/p/episodes/{eid}` (`link.view`), `PUT …/answers/{cid}` (`link.save`), `POST …/submit` (`link.submit`): only the session's patient's episodes (404 otherwise). `POST /api/p/logout`.
  - **What the HQ returns** (S14 answers 1): `patient: {firstName, name, age, sex}` (the age worked out in Go on the hospital's day, in its time zone), `hospital`, the questionnaire's name, the stripped patient Question Sets, answers, revisions, submitted. **Never the date of birth, the NHS number, the hospital number, the mobile, the email or the patient's record id.**
- **Staff completing for the patient (PAT-09):** `GET …/episodes/{eid}/patient-form` (`behalf.view`), `PUT …/patient-form/answers/{cid}` (`behalf.save`), `POST …/patient-form/submit` (`behalf.submit`), all audited. The same handlers as the patient's; the answers are stored as the patient's with **`updated_by` = the staff member** ("entered by"), and the submit sets `patient_submitted_by` and the note "HQ completed on the patient's behalf" (by that staff member). Once staff have entered a set's answers it **stays marked** even if the patient saves it again.
- `GET …/episodes/{eid}` now returns `invite: {sentAt, lockedAt} | null` and `episode.patientSubmittedByName`.
- **Audit labels:** "Patient opened their link", "Patient entered their date of birth", "Resent the patient's invite", "Opened the patient's HQ to fill in for them", "Saved answers for the patient", "Sent the HQ for the patient".

### Web app

TypeScript for Rahul to review, in plain terms:
- **`features/patient/InviteLink.tsx`** (`/i/:token`): a **Continue** card that shows nothing about the patient; Continue POSTs and gets only the hospital's name; then **Confirm it's you** with Day / Month / Year boxes (`dob.ts` builds `YYYY-MM-DD` and refuses impossible dates, with a test). A wrong date shows the server's "N tries left"; a locked or dead link shows its own card. The right date **replaces** the page with `/home/episodes/:eid` (so Back never returns to the date-of-birth form).
- **`features/patient/PatientHQ.tsx`:** now takes the API `path` (`/p/episodes/<id>` for the patient, `…/episodes/<id>/patient-form` for staff) and `onBehalf`. It uses the server's `{name, age, sex}` for the survey's variables instead of working the age out from a date of birth. A 401 shows **"You've been signed out"**. The patient's bar shows the hospital's name and **Sign out**.
- **`features/episodes/PatientFormPage.tsx`** (`…/episodes/:eid/patient-form`): the same form for staff, with a blue banner ("You're filling in …'s health questionnaire for them…"), "Send the answers", and **Back to the episode**.
- **The episode page:** the "Patient link / Copy" card is replaced by **Patient's invite**: when it was last sent, a red **The link is locked** box after five wrong dates of birth, **Resend invite** (or **Send invite** if none was sent; disabled with no mobile or email), and **Fill in for the patient**. The Question Sets table says "Patient's answers entered by <name>"; "Sent by <name> on the patient's behalf" when they sent it. The Validate page and the POA Summary ("Patient (entered by <name>) on …") show the mark too.
- **`features/qa/CapturedMessages.tsx`** (`/qa/messages`): the captured texts and emails, newest first, with a **To** filter and clickable links. No sign-in.
- `routes.tsx`: `/p/:token` is gone; `/i/:token`, `/home/episodes/:eid`, `/qa/messages` and `…/episodes/:eid/patient-form` added.

### Journeys (`e2e/`)

- **J3** now gets the invite from the captured text (`invite_path`) and goes through Continue and the date of birth.
- **New `journeys/test_j7_patient_invites.py`:** **J7** (create in the UI → the captured-messages page shows both → phone: Continue shows nothing about the patient → a wrong date ("4 tries left") → the right one → nothing they didn't type is shown → reload keeps them in → fill in and send → the link opens again later → Sign out, and a reload stays signed out), **J7b** (5 wrong → locked, even with the right date → the clinician sees the lock and resends → the old link "doesn't work", the new one opens the HQ), and **on the patient's behalf** (fill in and send → Ready for review, the note, "entered by" on the episode and the POA Summary).
- `support.py`: `Api.episode` gives each patient a made-up email (overridable), and returns `email`, `mobile` and `dob`; `Patient` (the `patient_link` fixture) opens the captured invite with Continue and the date of birth and keeps the session; `captured()` and `invite_path()` read `/api/qa/messages`.

## Round 1 (test report 1: 19 of 19 passed): Rahul's decisions on its "found while exploring" items

Rahul answered in [answers.md](answers.md) (items 2–4). Built test-first:

- **Patient sessions held in the database** (answers 4; report item 1: after Sign out a copied cookie still worked, and a Resend or a lock didn't end an open session). Migration **`00012_patient_sessions.sql`**: `patient_sessions` (`link_id`, `patient_id`, `created_at`, `last_seen_at`, `ended_at`). The right date of birth inserts a row; the cookie names only its id (**no patient record id in the cookie**, report item 2). `requirePatient` runs **one query** per request (`TouchPatientSession`): the row isn't signed out, was seen in the last 30 minutes, and its link isn't revoked, locked or expired; it also records the request. **Sign out** sets `ended_at` (a copy of the cookie then gets 401); a **Resend** revokes the link, so its sessions end; a **lock** (five wrong dates on the link) ends its sessions. Still no server memory (Lambda-safe).
- **`Cache-Control: no-store`** on every `/api/p` response (`noStore` in `router.go`), and **`<meta name="referrer" content="no-referrer">`** in `apps/web/index.html`, so the link's code never leaves in a Referer (report item 3). F4's hosting should also send a `Referrer-Policy` header.
- **Tests first:** `TestPatientSessionEndsOnTheServer` (a session row per right date and no record id in the cookie; Sign out ends it for a copied cookie; a Resend ends the old link's sessions but not another episode's; a lock ends them; the server-side idle check with a still-valid cookie; a forged cookie 401 and logout without a session harmless; `no-store` on continue, dob, the HQ and a 401). J7 checks the referrer meta on the link's page. apitest gained `Client.Cookie` and `Client.WithCookie`.
- **Recorded** (answers 2–3): name, age and sex after the check, and links lasting until C3, in decisions.md. **Logged in pending.md:** Send with no answers accepted by the API (needs C6's evaluate Lambda), sending the full name and age only when a questionnaire uses them, sticky "entered by", the bare "Episode not found." page, and capture mode's plain-text links (S17).
- **Results (13 Oct):** `make lint` passed; `make test`: Go ok, Vitest 116 (clinical) + 112 (web); `make test-integration`: all packages ok; `make seed` then `make e2e`: **35 passed** (132 s).

## Round 2 (test report 2: all passed)

No code changes. Its notes are logged in pending.md (S14): opening a second link in the same browser leaves the first session valid until idle; Sign out ends only that browser's session (a Resend or a lock ends them all); an answer changed after the session ended isn't saved and no message says so. Status: **Tested** (13 Oct), for verification.

## Verification 1 (Needs fixes): fixes, each test-first

1. **A change refused because the session ended is named, not lost** (blocking). New `features/patient/signedOut.ts` (`lostWork`, with a Vitest test): the signed-out card adds an orange **Not saved** box, "Your latest change in *About you* wasn't saved. When you open the link again, please check your answers in that section." (the set's name from the save that got the 401) or "Your answers haven't been sent yet…" (Send), above "…open the link in your text message or email again". `PatientHQ` passes what each refused request carried. New journey `test_a_change_refused_because_the_session_ended_is_named_not_lost` (save, Resend, change → the message; back in with the new link, the old answer is there). The pending.md item is struck through.
2. **A failed Resend no longer kills the working link** (`invites.go`). The order is now: build the messages, store the new link, send, then `settleInviteLink`: lock the episode row (`LockEpisode`, `FOR UPDATE`), and if a message went revoke the links **older** than the new one (`RevokeOlderInviteLinks`), else revoke the new one (`RevokeLink`). So "Last sent" stays truthful, the patient's old link and open session carry on, and of two Resends that cross the newer link wins. Tests: a Create where every send fails gives `invite: null` and no working link; a Resend where every send fails gives 502 and keeps the old link (Continue 200), the open session (200) and `invite.sentAt`; a Resend waits for the episode's lock (`meanwhile` holding `FOR NO KEY UPDATE`, which blocks only the explicit lock, not the new link's foreign-key check; the test fails without the lock), and five Resends at once leave exactly one working link.
3. **`MESSAGING` must be set with `CONFIG_STRICT=true`** (`config.go`): a missing one stops start-up and is named; locally it still defaults to `capture`. Config tests updated (strict without it fails, with it starts); `TestConfig`'s strict environment sets it. The captured-messages page's comment is corrected.
4. **The on-behalf form's reload dependency** (`PatientHQ.tsx`): `endedBy` depends on a boolean (`isPatient`), not the `onBehalf` object, so a re-render of the hospital layout can't reload the form and drop staff typing. No automated test: the web app has no DOM test tooling, and adding a library for one line isn't worth it; the journeys still pass.
5. **A-20:** the captured-messages page's placeholder no longer holds a UK number: "A mobile number (as stored, with its country code) or an email address".
- **Logged in pending.md** (verifier's list): `{patientAge}` vs the clinician views around a birthday; `%`/`_` in the `?to=` filter; a patient's probe of another patient's episode isn't audited.
- **Results (13 Oct):** `make lint` passed; `make test`: Go ok, Vitest 116 (clinical) + 116 (web); `make test-integration`: all packages ok; `make seed` then `make e2e`: 35 passed plus the new journey's file (4 of 4) after a test fix (it now waits for the first save's PUT).

## Round 3 (test report 3: all passed)

No code changes. Its exploring notes are logged in pending.md (S14): the "Not saved" box doesn't survive a reload of the signed-out page, and the section's name in it isn't bold. **For the verifier: rerun the full `make e2e`** (after `make seed`). After verification 1's fixes I ran the full suite before a test fix to the new journey, then only the J7 file (4 of 4) after it. Status: **Tested** (13 Oct), for re-verification.

## Verification 2 (Needs fixes, one finding): the fix, test-first

- **Sign out and the on-behalf form's "← Episode" link dropped the answer just typed** (the unmount cancelled the 600 ms autosave). Now (`PatientHQ.tsx`):
  - the open set registers a **"save now"** (`flushRef`): it takes the focus off the answer box, then saves at once and reports whether the server has it;
  - **Sign out** awaits it before `/p/logout`; if the save fails, the signed-out card's **Not saved** box names the section (a 401 does so through `endedBy` as before);
  - the on-behalf form's **← Episode** link (moved from `PatientFormPage.tsx` into `PatientHQ`, also shown while loading) awaits it and stays, with the save's message, if it fails;
  - leaving the set any other way sends a waiting save instead of cancelling it (after a 409 the saver refuses it without a request);
  - **text answers count as they're typed** (`textUpdateMode = "onTyping"`). The first full-suite run had one failure of the new Sign-out journey (an empty text box), which suggests the focus-change commit isn't always enough; with "onTyping" the full suite passed twice and the J7 file five times.
- **Tests first:** new data file `e2e/data/medicines-hq.json` (a text question and a Yes/No), and two journeys in `test_j7_patient_invites.py`: type "Warfarin 5mg" then Sign out at once → the text is there after reopening the link; tap Yes then Sign out at once → kept; staff type "Aspirin 75mg" then click **← Episode** at once → "entered by" on the episode and the text kept. The patient journey failed before the fix (empty box). The on-behalf journey's failure before the fix wasn't confirmed: it first failed on an ambiguous link name, which I then fixed.
- **Logged in pending.md** (the verifier's list): a Resend whose settle step fails after sending; the journeys reusing made-up mobiles; `loadForm`'s ownership check living in its callers.
- **Results (13 Oct):** `make lint` passed; `make test`: Go ok, Vitest 116 + 116; `make test-integration`: all ok; `make seed` then **full `make e2e`: 38 passed, twice**.

## Round 4 (test report 4: 4 passed; X-1 skipped by Rahul)

- **X-1 (minor):** the patient's browser Back, a reload or closing the tab within ~0.6 s of typing loses that input silently: the waiting save is sent only by React's unmount cleanup, which doesn't run on page unload. **Rahul (13 Oct, answers 5): skip for now, an edge case.** No code change; logged in pending.md with the fix (`pagehide`/`visibilitychange` + a `keepalive` fetch). The misleading code comment in `PatientHQ.tsx` (it said Back sends the waiting save) is corrected; comment only.
- **Exploring items logged in pending.md:** a section with no required questions shows COMPLETE before anything is answered; the status says "Saved" while a change waits for its autosave.
- Status: **Tested** (13 Oct), for re-verification. The full `make e2e` last ran after verification 2's fix (38 passed, twice); nothing has changed since but a comment.

## Decided by the builder (automatic mode)

- **What "any identifier the patient didn't type" leaves in the HQ:** the first name (for "Hello Sam") and the survey variables `{patientName}` (full name), `{patientAge}` and `{patientSex}`, which questionnaires can use; the age is worked out in Go so the date of birth never leaves the server. Left out: date of birth, NHS number, hospital number, mobile, email, record id. **Reason:** Rahul's answer; the full name stays because the HQ's `{patientName}` shows it and the patient has just proved who they are.
- **Continue returns only the hospital's name and changes nothing.** **Reason:** the brief and the answer: Continue exists so a link scanner can't use anything up; for an invite nothing is used up at all.
- **A resend replaces the episode's earlier links** (only the newest works), and so also unlocks. **Reason:** only the hash is stored, so the same link can't be sent again; keeping old ones alive would leave a link sent to a wrong number working; PAT-05 says a locked patient "must ask for a new one".
- **The right date of birth resets the wrong-attempt count.** **Reason:** an invite is used many times over days (PAT-05); without a reset, four slips spread over a week would lock it on the next one.
- **"Until the episode closes":** no status closes an episode yet (C3 adds discharge and cancel), so an invite link works while the episode exists. The query is ready for an expiry (`expires_at`).
- **SMS country check by calling code** (`+44` for GB, which also covers the Crown Dependencies; `+1` is shared across North America). It doesn't check that the number is a mobile: Ofcom's made-up 07700 900xxx numbers must pass.
- **The default invitation SMS changed to PAT-03's wording** (`{{hospital}}: please complete your pre-op health questionnaire: {{link}}`), as the brief says; the email keeps "Hello {{firstName}}…". A trust's or hospital's own template (S10) still wins.
- **Sending happens in the request, after the episode is saved** (S16 moves it to jobs). A failure is logged and noted on the episode, never a failed Create.
- **"Entered by" is per Question Set, sticky:** once staff save a set's answers for the patient, the set stays marked with the last staff member to do so, even if the patient later adds to it.
- **Resend refusals:** 409 after the answers are sent, or with nothing to send to; 502 when every message failed.
- **`/api/qa/messages` matches the recipient by "contains"** (so a tester can type `900123`), newest 100; no sign-in (capture mode only; S17's CDK check refuses capture outside QA). It isn't audited: it holds only made-up local data.
- **The patient's Sign out** (a small addition): `POST /api/p/logout` is in the plan's API, and on a shared phone the patient needs a way to end the session before 30 minutes.
- **The steps of `/i/:token` are page state, not URLs:** reloading starts again at Continue; Back from the HQ leaves the app rather than returning to the date-of-birth form.

## Differences from the plan

- C2 lists `GET/PUT …/forms/{fid}/answers/{cid}`; an episode has one HQ until C3, so it's `/api/p/episodes/{eid}` (+ `/answers/{cid}`, `/submit`). Staff's on-behalf form is `…/episodes/{eid}/patient-form`.
- C2's patient URLs list `/home`; S14 builds only `/home/episodes/:eid` (where the HQ opens). `/home` itself is S15.
- Linking to a patient account (`patients.account_id`) on the first invite is S15's (accounts don't exist yet).

## Tests

- **Unit:** `internal/auth` (patient session: within/after the 30-minute idle timeout, renewed by requests, not a staff token and vice versa, forged; link token and hash), `internal/patient.SMSAllowed`, `internal/settings.Fill` (PAT-03 default, hospital template, no placeholder injection, missing template), `internal/config` (`MESSAGING`); web `dob.test.ts`, `routes.test.ts`.
- **Integration (API → database):**
  - `TestPatientLinkContinue`: only the hospital's name, nothing used up, no cookie; no GET on the link (no data); bad, unknown and replaced links 404; locked 403 with `code: "locked"`.
  - `TestPatientLinkDob`: invalid dates are 400 and not tries; 4 wrong then right (tries left in the message, no cookie on a wrong one, cookie flags, attempts reset, `used_at`); the invite opens again; **the 5th wrong locks**, even the right date then; another link untouched.
  - `TestPatientSession`: no session 401 on all three; a signed-in clinician 401; another patient's episode (same trust or another) 404 and nothing written; the same patient's other episode opens; **the idle timeout** (20-minute gaps keep it open; 31 minutes idle → 401); logout.
  - `TestPatientHQ`: stripped content, answers, revisions, `{firstName, name, age, sex}`; **none of the date of birth, NHS number, mobile, email, hospital number or record id in the body**.
  - `TestAgeOnTheHospitalsDay`: the age turns on the hospital's own day (Auckland vs UTC).
  - `TestSavePatientAnswers`, `TestSubmitPatientHQ`: the S02a cases ported to the session routes.
  - `TestCreateEpisodeSendsTheInvite`: the text (PAT-03 wording, the hospital's name) and email with the same link; only the hash stored; the note; the hospital's own template; email only; **PAT-11 no text to +1**; no contacts → nothing; a provider failure → the episode still made, the note says so.
  - `TestResendInvite`: the five cases; a new link to both, the old one 404; a locked link is replaced and staff see `invite.lockedAt` until then; no contacts 409; after submit 409.
  - `TestCompleteOnBehalf`: the five cases; the answers marked with Cara; the patient adding later keeps the mark; `GET …/hq` names her; submit → Ready for review, `patient_submitted_by`, the note, `patientSubmittedByName`; frozen after.
  - `TestCapturedMessages` (with `Capture`): stored, by recipient, newest first.
  - Updated: `TestAuditRowsWritten` (link.open, a wrong and a right link.dob, link.view/save/submit, behalf.*, invite.resend; unknown link, no session and another hospital's on-behalf write nothing), `TestListAudit`, `TestAuditFailsClosed`, `TestAdminSuspendTrust` (the link and an open session are refused while suspended), `TestPatientRouter`, `TestStaffRouter`, `TestCreateEpisode`, the apitest self-test.
- **Results (13 Oct):** `make lint` passed; `make test`: Go ok, Vitest 116 (clinical) + 112 (web) passed; `make test-integration`: all packages ok; `make e2e`: **35 passed** (123 s).

## Suggested commits

1. **Patient invites and sign-in: data, Go and API tests (C2 / S14: data and API)** — `apps/api/**` (migrations 00011 and 00012, queries, sqlc output, `internal/messaging`, `internal/auth/patient.go`, `internal/patient`, `internal/settings`, `internal/config`, `internal/serve`, `internal/httpapi`, `internal/apitest`, `internal/episode`), `.env.example`.
2. **The invite link, the patient's form, on-behalf and captured messages pages, and journeys (C2 / S14: web and journeys)** — `apps/web/**`, `e2e/**`.
3. **Docs: S14 brief, answers, build notes, test plan, test reports, verification and status** — `docs/sessions/S14-patient-invites/**`, `docs/plans/c2-patients-signin.md`, `docs/plans/README.md`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plan-workflow.md`.
