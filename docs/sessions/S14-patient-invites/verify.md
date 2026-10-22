# S14 verification

## Verification 3 (13 Oct 2026, automatic run; an extra one Rahul allowed)

**Verdict: Verified.**

Verification 2's one finding is fixed and I confirmed it myself, at phone size and on the on-behalf form. Since verification 2 the only code changes are that fix and one corrected comment. The rest of the session still holds: I re-ran every suite, repeated the security checks with curl, and weighed a fresh `/code-review high`. Nothing new is serious enough to block. Test report 4's X-1 (Back or reload within ~0.6 s of typing) was skipped by Rahul (answers 5) and is in pending.md.

### Verification 2's fix, checked

- **The code** (`apps/web/src/features/patient/PatientHQ.tsx`):
  - The open set registers a "save now" (`flushRef`, `:85`, set at `:402–411`). It blurs the box, takes `model.data` and calls `persist()`.
  - **Sign out** (`:282`) awaits it before `/p/logout`. If the save fails, the card's Not saved box names the set.
  - **← Episode** (`leave`, `:129`) awaits it and stays if the save fails.
  - The unmount cleanup (`:440–446`) now sends a waiting save instead of cancelling it.
  - `textUpdateMode = "onTyping"` (`:368`).
- **Saves can't cross.** `answerSaver` (`packages/clinical/src/answerSaver.ts`) puts saves in a queue. So a flush that overlaps a save already in flight goes after it, with the new revision. After a 409 it refuses without sending a request, so the cleanup's save can't overwrite newer answers.
- **Phone size** (390 × 844 iframe on tester.localhost, made-up episode *Sam SignoutV3*):
  1. Link → Continue showed only "Hospital A1".
  2. Date of birth → Review "Your medicines".
  3. I typed `Digoxin 125mcg`, tapped **Yes** and tapped **Sign out**, all in one batch.
  4. The page showed "You've been signed out" with no Not saved box.
  5. `episode_answers`: one row, rev 1, `{"q_thinners":"o_yes","q_medicines":"Digoxin 125mcg"}`.
- **On-behalf** (Cara, *Sam BehalfV3*):
  1. I typed `Apixaban 5mg`, tapped **No** and clicked **← Episode** at once.
  2. The episode showed "Patient's answers entered by Cara Clinician".
  3. The database had both answers, with `updated_by` set.
- **The journeys** `test_sign_out_straight_after_typing_keeps_the_answer` and `test_leaving_the_on_behalf_form_straight_after_typing_keeps_the_answer` passed in the full run.

### Findings from `/code-review high`, weighed (none blocks; all logged in pending.md)

The review raised these:
- items verifications 1 and 2 had already logged: `/api/qa/messages` open in capture mode, `{patientAge}` versus the clinician views, the ILIKE wildcards, sending inside the Create request;
- `time.LoadLocation` on each load, which isn't a problem: `time/tzdata` is built in (`internal/config/config.go:21`).

**New, logged in pending.md:**
1. **The date-of-birth boxes are fixed in day, month, year order, and the example is "27 3 1985"** (`apps/web/src/features/patient/InviteLink.tsx:107–115`, plus Go's error text). This touches A-20. The boxes are labelled Day, Month and Year, so nothing is misread, and only the UK runs today. Before a second country, take the order and the example from the deployment's locale.
2. **The right date of birth resets the wrong-try count** (a builder decision in build-notes). Someone who holds a link gets 4 more guesses each time the real patient signs in. That's slow to exploit (about 36,500 possible dates). Rate-limiting per link would close it.
3. **When two Resends cross,** the slower one's messages can arrive after the newer one's, carrying a link that's already been revoked. The patient then taps the newest text and gets "This link doesn't work". It's rare (it needs a double Resend and a slow provider), and pressing Resend again fixes it.
4. **The set's unmount save ignores its result** (`PatientHQ.tsx:443`). Staff who leave the on-behalf form through the app's top bar within 0.6 s of typing, and whose save then fails (409 or 500), aren't told. The ← Episode link and Sign out do report a failed save.
5. **A patient session opens every episode of that patient** (as designed and tested: `TestPatientSession`, "the same patient's other episode opens"). A lock or Resend on episode B's link doesn't end a session opened through episode A's link. Revisit this with S15's patient home.

### Spot checks (13 Oct, against the conductor's `make dev`)

- **The link alone:**
  - `GET …/continue` → 405.
  - `POST …/continue` → 200 `{"hospital":"Hospital A1"}` with `Cache-Control: no-store`.
  - The page shows only Continue, then the date-of-birth form.
- **The patient's session:**
  - Its own episode → 200, and the body has no `dateOfBirth`, `mobile`, `email`, `hospitalNumber` or `nationalId`.
  - Another patient's episode → 404.
  - The staff API with the patient's cookie → 401.
- **The fix,** at phone size and on-behalf (above).
- **The A-20 grep** finds only comments (`format.ts`, `patient.go`). **The incumbent-name grep** prints nothing.
- **Test data left in the local database (made up):** the HQ "V3 Medicines 1310 145537 aa48" (published, Hospital A1), and the episodes *Sam SignoutV3* and *Sam BehalfV3*.

### Test results

- `make lint`: passed.
- `make test`: all Go packages ok. Vitest: 116 passed (clinical) and 116 passed (web).
- `make test-integration`: all packages ok (`internal/httpapi` 11.9 s).
- `make seed`, then the full `make e2e`: **38 passed** in 157.10 s.

### Hand checks for Rahul

- None new. S11's check of the patient page in a real 390 × 844 window still applies to the `/i/` flow.

### Commits (made by the verifier, automatic mode)

1. **Patient invites and sign-in: data, Go and API tests (C2 / S14: data and API):** `apps/api/**`, `.env.example`.
2. **The invite link, the patient's form, on-behalf and captured-messages pages, and journeys (C2 / S14: web and journeys):** `apps/web/**`, `e2e/**`.
3. **Docs: S14 brief, answers, build notes, test plan, test reports, verification and status:** `docs/sessions/S14-patient-invites/**`, `docs/plans/c2-patients-signin.md`, `docs/plans/README.md`, `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`, `docs/plan-workflow.md`.
   - S14's `brief.md` also holds its 12 Oct planning edits (the local-phase note and the capture-mode wording). It's committed whole.

**Next:** `/build-session S15`.

---

## Verification 2 (13 Oct 2026, automatic run; the last one the conductor allows)

**Verdict: Needs fixes.** There is one finding, and it's small to fix.

All five fixes from verification 1 are in and correct. I read the code, ran the tests and checked by hand (details below). The security core still holds. But the patient's **Sign out** button, which is new in S14, throws away the answer the patient was just typing, and says nothing. I reproduced it in the browser at phone size. That breaks the same rule as verification 1's finding 1 ("nothing may break silently"), with a more ordinary trigger.

**For Rahul (the conductor's fix rounds are used up):**
- **Recommended:** allow one more short fix round for finding 1. It's a few lines in `PatientHQ.tsx`, plus a journey step.
- **Alternative:** commit S14 as it is and move finding 1 to pending.md, to fix with S15, which reworks the patient's pages anyway. Until then, a patient who types an answer and taps Sign out at once loses that answer without being told.

### Findings, worst first

1. **Sign out drops the answer being typed, without a word** (`apps/web/src/features/patient/PatientHQ.tsx:256` `signOut`; `:352` the 600 ms save timer; `:368` the unmount cleanup that cancels it).
   - **Why it happens:**
     - SurveyJS commits a text answer only when the box loses focus (the default `textUpdateMode: "onBlur"`).
     - A change is then saved 600 ms later.
     - Tapping **Sign out** takes the focus off the box, so the answer commits and the 600 ms timer starts. `signOut` then calls `/p/logout` and shows the signed-out card at once.
     - Showing that card unmounts `SetForm`, and its cleanup (`:368`) cancels the waiting save.
     - So the save never goes, `lost` is `{}`, and no "Not saved" box appears.
     - While the patient types, the status still says **Saved**.
   - **Reproduced (13 Oct, 390 × 844 iframe on tester.localhost, a real click):**
     1. I set up a made-up HQ with one patient text question ("What medicines do you take?"), episode *Tess Text V2*.
     2. Link → Continue → date of birth → Start → typed "Warfarin 5mg".
     3. Clicked **Sign out**. The page showed "You've been signed out — You've signed out. To come back…", with no Not saved box.
     4. `episode_answers` for the episode: **0 rows**.
   - **The same happens with a radio answer** tapped less than 600 ms before Sign out (episode *Vic Verify V2*: answered Yes, signed out 200 ms later, 0 rows).
   - **It isn't a rare race for text answers:** "type my medicines, tap Sign out, come back later" loses the text every time. When the patient comes back, a new answer is blank; that might be noticed. A changed one (an added medicine, or No changed to Yes) shows the old value, and nothing prompts them to check it.
   - **The on-behalf form has the same cause.** Its **← Episode** link (`apps/web/src/features/episodes/PatientFormPage.tsx:13`) unmounts the form the same way. A staff member who types an answer and clicks it loses that answer. I didn't click this one through; it's the same code path.
   - **"All sections" and "Finish this section" are fine:** they await `save()` first.
   - **Fix (recommended):**
     - Sign out first sends any waiting save, and waits for it. If that save is refused, the card says what wasn't kept, through the existing `lostWork` and `lost.unsavedSet`.
     - Leaving the form any other way also sends a waiting save. Today the unmount cleanup cancels it; instead, send it at once unless the page is showing the 409 conflict (which is why the cleanup cancels it today).
     - One way to do it: `SetForm` registers a `flush()` with the parent through a ref, and `signOut` does `await flush()` before `/p/logout`.
   - **Test first:** a J7 step: answer, then Sign out at once, then open the link again; the answer is there. For a text answer, add a patient text question to the journey's HQ, or put one in a new data file. Ideally the on-behalf "← Episode" too.

**Log in pending.md (no fix needed now; from `/code-review high`, weighed by me):**
- **A Resend whose messages went but whose settle step failed** (a database error in `settleInviteLink`, `invites.go:91`) answers 500. The General notes then say "No invite sent: something went wrong", although the patient got a working link and the older links weren't revoked. It's rare, and pressing Resend again puts it right.
- **The journeys reuse made-up mobiles.** J3 uses a fixed `+447700900123`, and `made_up_mobile` picks from 1000 numbers, while `sent_messages` keeps growing in the dev database. A count such as `to_have_count(1)` could one day see an old message to the same number. Not seen yet: 36 of 36 passed.
- **`loadForm` (`patient.go:265`) returns ok for an episode that isn't the caller's,** and each caller then checks ownership itself. That's correct today, as tested (another patient: 404; another hospital or trust: 404), but it's fragile for the next caller. Passing the owner check into `loadForm` would be safer.
- *(Already logged or planned: sending happens inside the Create request until S16's jobs; `{patientAge}` versus the clinician views; `%`/`_` in `?to=`; capture mode's public `/api/qa/messages`. S17's brief already has CDK refuse capture outside QA. S06's QA stack must now set `MESSAGING=capture` explicitly, and start-up names it if it's missing.)*

### Verification 1's fixes, checked

1. **The "Not saved" box:** `signedOut.ts` (`lostWork`) has a Vitest test. `PatientHQ` passes the set's name on a refused save and `unsent` on a refused Send. The new journey covers it, and test report 3's T-19 passed at phone size.
2. **A failed Resend keeps the old link** (`invites.go`). The order is now:
   1. build the messages;
   2. store the new link;
   3. send;
   4. `settleInviteLink` under `SELECT … FOR UPDATE` on the episode: revoke links older than the new one by `(created_at, id)` if a message went, or else revoke the new link.

   I reasoned through crossing Resends: the newest link that was sent wins, whichever order they settle in. A new link's foreign-key check waits on the row lock, so a later revoke statement sees it. The integration tests cover:
   - all sends failing on Create (`invite: null`);
   - all sends failing on Resend (502; the old link, its session and `sentAt` are kept);
   - the lock;
   - five Resends at once (one working link).
3. **`MESSAGING` under `CONFIG_STRICT`:** strict with no `MESSAGING` fails start-up and names it, and any value other than `capture` fails. It still defaults to `capture` locally. Unit tests updated.
4. **The on-behalf reload dependency:** `endedBy` now depends on `isPatient`, a boolean. No DOM test; test report 3 checked re-renders by hand.
5. **A-20 placeholder:** the text is neutral now. The A-20 grep finds only comments and config, seed and test files. The incumbent-name grep prints nothing.

### Spot checks (13 Oct, against the conductor's `make dev`)

- **The link alone:**
  - `GET …/continue` → 405.
  - `POST …/continue` → 200 `{"hospital":"Hospital A1"}`, with `Cache-Control: no-store` and no cookie.
  - A wrong date → 401 "You have 4 tries left".
  - The page has `<meta name="referrer" content="no-referrer">`.
- **After the date of birth:**
  - The HQ's keys are `answers, chapters, hospital, patient, questionnaireName, revisions, submitted`, and `patient` is `{firstName, name, age, sex}`.
  - The body holds no date of birth, mobile, email or hospital number.
- **Isolation:**
  - the patient cookie on the staff API → 401, and the staff cookie on the patient API → 401;
  - the patient's session on another patient's episode → GET 404, PUT 404;
  - Bea (Trust B) on Resend and on-behalf, through her own hospital's path and through Trust A's → 404.
- **Resend ends the open session:** 200 before and 401 after. The episode's `invite.sentAt` moved to the Resend.
- **Phone size (390 px):** link → Continue → date of birth → the HQ, "Hello Vic". Finding 1 was reproduced there.
- **Screenshot of finding 1** (the signed-out card after Sign out; the typed answer isn't in the database): `/var/folders/fy/sqvv2t291gdbrdvk20959mm40000gn/T/claude-chrome-screenshots-IrEXIV/screenshot-1791897273189-0.jpg`.
- **Test data left in the local database (made up):**
  - HQ "V2 text check" (published, Hospital A1);
  - episodes *Vic Verify V2* and *Tess Text V2*.

### Test results

- `make lint`: passed.
- `make test`: all Go packages ok. Vitest: 116 passed (clinical) and 116 passed (web).
- `make test-integration`: all packages ok (`internal/httpapi` 17.1 s).
- `make seed` then the full `make e2e`: **36 passed** in 160.65 s. This includes J7's file, 4 of 4.

### Suggested commits (after the fix, or if Rahul accepts as is)

The three commits suggested under verification 1 still apply. The web commit also takes the finding 1 fix and its journey step.

**For the builder:** "Verification found problems: read `docs/sessions/S14-patient-invites/verify.md`, fix them test-first, then ask me for a retest."

---

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Needs fixes.**

The security core is sound. I checked it myself with curl and in the browser, as well as through the tests:
- the link alone returns only the hospital's name, and a GET on it is refused;
- only the token's hash is stored;
- the lock works, and staff see it;
- sessions are held in the database and end on Sign out, Resend or a lock;
- the patient and staff cookies can't be swapped;
- a patient can't open another patient's episode (404, nothing written);
- another trust's staff get 404 on on-behalf and Resend;
- `no-store` is set, and every patient-data route writes an audit row.

One finding breaks "Healthcare: nothing may break silently", and I confirmed it in the browser. Two more are cheap to fix and matter once real sending arrives. None needs a decision from Rahul: each has a recommended fix.

### Findings, worst first

1. **A patient's changed answer is lost without a word when their session has ended** (`apps/web/src/features/patient/PatientHQ.tsx:79`, `:107`, `:118`, `:137`; `SignedOut` at `:266`).
   - **Reproduced at phone size (390 px iframe):**
     1. The patient answered "Do you smoke?" = **No**, and it saved.
     2. Cara pressed Resend invite.
     3. The patient changed the answer to **Yes**. The page went straight to "You've been signed out: For your security, you've been signed out. Please open the link…".
     4. The database still says `{"q_smokes": "o_no"}`. Nothing tells the patient that their last change wasn't saved, or (for Send) that the answers weren't sent.
   - **This isn't rare.** The usual cause will be the **30-minute idle timeout**: a patient leaves the form open to fetch their medicines or look something up, comes back, types the answer and loses it. When they open the link again, the form shows the old answer, and they have no reason to check it.
   - This is the clinical-safety case decisions.md §"nothing may break silently" is about. pending.md (S14) calls it "acceptable, since the patient must reopen the link anyway", but reopening doesn't tell them anything was lost.
   - **Fix:**
     - When a **save** gets the 401, the signed-out card also says the change wasn't saved and names the Question Set. For example: "Your last change in *About you* wasn't saved. When you open the link again, please check your answers in that section."
     - When **Send** gets the 401, the card says the answers haven't been sent yet.
     - Test it first: a J7 step (resend, change an answer, then assert the message), or a Vitest test of the message logic.
     - Then remove or replace the pending.md item.

2. **A failed Resend kills the patient's working link and their open session, and gives them nothing in its place** (`apps/api/internal/httpapi/invites.go:57`).
   - `invite()` revokes the episode's links and stores the new one (`storeInviteLink`, `:98`) *before* it builds the messages (`inviteMessages`, `:60`) and before any message goes. So if the provider is down, or the hospital's settings fail to decode, the patient is signed out mid-form, their text's link now says "doesn't work", and no new link reached them. Staff get a 502 or 500.
   - This can't happen in capture mode, but it will with S17's real sending.
   - **The same cause makes the Episode page say "Last sent <time>" when nothing was sent** (`apps/web/src/features/episodes/EpisodePage.tsx:118`, `GetEpisodeInvite`'s `created_at`). On Create with every send failing, the card claims the invite went out, while the note says it didn't. Staff may not press Resend.
   - **Fix:**
     1. Build the messages first.
     2. Insert the new link.
     3. Send.
     4. Revoke the older links only when at least one message went. If none went, revoke the new link instead, so the old link and its sessions carry on.
   - **Test first,** with `Recorder.FailWith`: a resend where both sends fail leaves the old link working (Continue 200), leaves the open session at 200, and `invite` stays as it was. A Create where both fail shows `invite: null` (so the page offers **Send invite**).

3. **The unauthenticated `GET /api/qa/messages` is on by default even with `CONFIG_STRICT=true`** (`apps/api/internal/config/config.go:140–146`; `apps/api/internal/httpapi/router.go:73`).
   - An empty `MESSAGING` silently becomes `capture` in strict mode too. That contradicts the file's own rule (`config.go:7`: under strict "there is no fallback").
   - It means any AWS stack started without the variable serves every trust's captured texts and emails to anyone: names, contacts, hospital names and working invite links.
   - `CapturedMessages.tsx:4` says "the API answers 404 everywhere else", which is false today.
   - DEV_LOGIN, the other dev-only door, is an explicit opt-in. This should be one too.
   - **Fix:** under `CONFIG_STRICT`, a missing `MESSAGING` stops start-up (keep the `capture` default only locally). Add a config unit test for it, and correct the comment.

4. **Two Resends at once can leave two working links** (`apps/api/internal/httpapi/invites.go:98`). Under READ COMMITTED, each transaction's revoke misses the other's insert. That breaks "only the newest link works", and the page shows only one of them. Low risk: both links go to the same patient.
   - **Fix:** lock the episode row (`SELECT … FOR UPDATE`) at the start of the transaction.
   - Fold this into finding 2's rework.

5. **The on-behalf form can reload and drop staff typing on a re-render** (`apps/web/src/features/episodes/PatientFormPage.tsx:16`). `onBehalf={{ back }}` is a new object on every render, and it's a dependency of `endedBy` → `load` → the load effect. Any re-render of the hospital layout refetches the HQ and resets drafts and savers.
   - It doesn't happen in normal use today, because nothing re-renders the layout mid-form. It's fragile, though.
   - **Fix:** depend on a boolean (`!!onBehalf`) in `endedBy`.

6. **A-20: a UK number is hardcoded** as the captured-messages page's placeholder (`apps/web/src/features/qa/CapturedMessages.tsx:57`, `+447700900123`). The check in the skill finds it.
   - **Fix:** use country-neutral text ("A mobile number or an email address").

**Log in pending.md (no fix needed now):**
- **The patient's `{patientAge}` and the clinician views can disagree around a birthday.** The patient's age is worked out in Go on the hospital's date, while the clinician views (`ageFrom`) use the browser's local date, so a condition such as `{patientAge} >= 70` can differ for a few hours.
- **The `?to=` filter treats `%` and `_` in ILIKE as wildcards** (`queries/links.sql:66`; QA tool only).
- **A signed-in patient trying another patient's episode id isn't written to any audit log** (by design, so it doesn't land in the other trust's log; it may be worth a security log later).

### Checks

- **Scope:** everything in the brief is built:
  - `Deps.SMS`/`Email` with fakes; `MESSAGING=capture`; `/api/qa/messages` and its page;
  - an invite on Create, and Resend; `login_links` storing only the hash;
  - `/i/:token` with Continue → date of birth → a 30-minute session; the 5-try lock;
  - the patient API replacing `/p/:token`; on-behalf with "entered by";
  - J7, J7b and J3.

  The extras (DB-held sessions, `no-store`, `no-referrer`, patient Sign out) are Rahul's answers 4 or the plan's API.
- **Tests first:** new routes have the five cases (Resend and on-behalf: happy path, no sign-in, another hospital in the trust (wrong role), another trust, invalid input). The patient routes cover no session, a staff cookie, another patient in the same trust and in another trust, bad, unknown and replaced links, and invalid dates. Unit tests cover auth, config, settings.Fill and SMSAllowed. `/api/qa/messages` has only its happy path (it's dev-only; finding 3 adds the strict-mode test).
- **Tenant isolation:** staff routes go through `requireHospital`, and `loadStaffForm`/`loadEpisode` check the hospital. Patient routes check `PatientID == patientOf(r)`. Confirmed with curl (below).
- **A-20 grep:** only finding 6 (the other hits are comments).
- **Security:**
  - no secrets in the repo;
  - texts and emails carry the first name, hospital and link only (PAT-03);
  - the patient cookie is `Path=/api/p`, and the staff `Verify` refuses tokens with an audience;
  - `chapter.ForPatient` still strips clinician-only content (`TestPatientHQ`).
- **Docs:** C2's Status, plans/README, sessions/README and decisions.md (answers 1–4) are updated. The incumbent-name grep prints nothing.

### Spot checks (13 Oct, against the conductor's `make dev`)

- **The link before the date of birth:** `GET /api/p/links/<t>/continue` gives 405 and the old `GET /api/p/<t>` gives 404. `POST …/continue` returns 200 `{"hospital":"Hospital A1"}` with `Cache-Control: no-store` and no cookie.
- **After the date of birth:** the HQ's keys are `answers, chapters, hospital, patient, questionnaireName, revisions, submitted`, and `patient` is `{firstName, name, age, sex}`. The body has no date of birth, email or hospital number.
- **Isolation:**
  - patient 1's session on patient 2's episode: GET 404, PUT 404;
  - the patient cookie on the staff API: 401; the staff cookie on the patient API: 401;
  - Bea (Trust B) on Trust A's episode, through her own hospital's path: on-behalf GET, Resend and submit are all 404.
- **The lock:** 4 × 401, then 403 on the 5th, then 403 `code: locked` on the right date. Staff's episode shows `invite.lockedAt`.
- **Audit:** `link.open` 200, `link.dob` 200/401/403 and `link.view` 200 were written, with actor `patient`, in the right trust.
- **Phone size (390 px iframe, tester.localhost):** the link → Continue → the right date → `/home/episodes/<id>` with "Hello Vera" and scrollWidth 388. Finding 1 was reproduced here.

### Test results

- `make lint`: passed.
- `make test`: all Go packages ok. Vitest: 116 passed (clinical) and 112 passed (web).
- `make test-integration`: all packages ok (`internal/httpapi` 12.4 s).
- `make e2e`: not re-run by the verifier (the builder's run, 13 Oct: 35 passed). It isn't needed for a Needs-fixes verdict; run it after the fixes.

### Suggested commits (after the fixes)

1. **Patient invites and sign-in: data, Go and API tests (C2 / S14: data and API):** `apps/api/**`, `.env.example`.
2. **The invite link, the patient's form, on-behalf and captured-messages pages, and journeys (C2 / S14: web and journeys):** `apps/web/**`, `e2e/**`.
3. **Docs: S14 brief, answers, build notes, test plan, test reports, verification and status:**
   - `docs/sessions/S14-patient-invites/**`;
   - `docs/plans/c2-patients-signin.md`, `docs/plans/README.md`, `docs/sessions/README.md`;
   - `docs/decisions.md`, `docs/pending.md`, `docs/plan-workflow.md`.

   The message should say which of these files also hold 12 Oct planning changes.

### Hand checks for Rahul

- None new. S11's real-window 390×844 patient check still applies to the new `/i/` flow.

**For the builder:** "Verification found problems: read `docs/sessions/S14-patient-invites/verify.md`, fix them test-first, then ask me for a retest."
