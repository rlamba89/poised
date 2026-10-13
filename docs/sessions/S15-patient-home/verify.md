# S15 verification

## Verification 2 (13 Oct 2026, automatic run)

**Verdict: Verified.** Verification 1's finding 2 is fixed correctly and proved by a journey, a retest and my own probe. Finding 1 is Rahul's accepted risk (answers 10), recorded in decisions.md and pending.md, so it doesn't block. Nothing else in the session is a serious defect. A fresh code review (`/code-review high`, whole working tree) found only minor items, listed below and added to pending.md.

### What changed since verification 1

Only two code files changed after verification 1 (by modification time). The rest are docs.
- **`apps/web/src/features/patient/PatientHQ.tsx` (`goHome`, about line 137):** "← Your questionnaires" now works like the on-behalf **Episode** link (`leave`) and Sign out:
  - `preventDefault`, then `await flushRef.current()`, and `navigate("/home")` only when the save succeeded;
  - with no set open (`flushRef` is null), the link navigates normally;
  - when the save is refused, the patient stays and the save's message shows; a 401 shows the signed-out card with **Not saved**.
- **`e2e/journeys/test_j7c_returning_patient.py`:** the new journey `test_j7c_leaving_for_the_home_straight_after_typing_keeps_the_answer` covers both the saved case and the 401 case. The build notes say it failed before the fix.
- **Docs:** answers 10, decisions.md (the accepted risk), pending.md (findings 3–7 and report 4's exploring items), T-24 in the test plan, test report 4, build notes, and status rows.

### Findings (none blocking)

1. **Accepted risk, not re-reported:** verification 1's finding 1 (Rahul, answers 10).
2. **Minor (from the code review), added to pending.md (S15):**
   - **A session now covers the whole account, but it still lives and dies with the one link it was opened with** (`links.sql` session checks; `accounts.go` notes). Trust A's **Resend**, or five wrong dates on Trust A's invite, signs the patient out of Trust B's form as well. The patient is told what wasn't saved (S14's lost-work card), so nothing is lost silently. The builder documented this choice ("It's still tied to the link it was opened with").
   - **Confirming a contact change ends no open sessions** (`accounts.go`, `confirmContact`). The old contact is told, but a session left open on a shared phone carries on for up to 30 idle minutes.
   - **`rate_limit_hits` keeps rows forever for keys never seen again** (`DropOldHits` in `db/queries/accounts.sql:161` deletes only the key being hit). The keys hold plain emails and mobiles, so this is a retention issue. The migration's comment says "rows older than a day are dropped". The fix is to drop the key filter, or clean up in S16's jobs.
   - **`greeting` can pick a lower-case fallback over an older properly-cased name** (`accounts.go:415`). For example, newest "mary ann" and older "Mary Ann" give "Hello Mary ann". This is cosmetic.
   - **`AccountsForSignin` / `AccountHasContact` use `mobile = $1 OR email = $1` without `<> ''`**, so the partial indexes may not be used under a generic plan. This is performance only.
   - **`signinDob` picks the account before the link is locked** (`accounts.go` about line 300). A contact confirmed in that instant could still let the old contact's link sign in. The window is tiny.
3. **Considered and dismissed:**
   - **"account.join audit rows are written even if the commit fails":** `audited` stores the response's status in each row, so a 500 row reads as a failed request, as on every audited route.
   - **"both of the record's contacts are treated as proved":** verification 1 dismissed this as the builder's recorded decision. It still needs the full name and date of birth.
   - **The QA reset route's tests:** already in pending.md for S17 (verification 1, finding 7).

### Checks

- **Scope, tests, isolation and security:** as in verification 1. Nothing has changed in Go since then.
- **A-20 grep:** comments only (`format.ts`, `patient.go`).
- **Org-name grep:** clean.

### Spot checks (Claude in Chrome, `tester.localhost:3000`, a 390×844 iframe)

I made up "Sam VerifyV2…", born 21/03/1977, with one email, at Hospital A1 (Medicines) and Hospital B1 (Workflow). I also made a different patient at A1. Then I opened B1's invite and entered the date of birth.
- **The sweep:** `GET /api/p/home` lists **both trusts**, B1 then A1. It shows the email in part (`v•••@example.org`), no date of birth, and no sideways scroll (386/386).
- **Another patient's episode** (`GET /api/p/episodes/<other A1 episode>`): **404 "Episode not found."**
- **The fix:** I opened A1's Your medicines, set the text box to "Apixaban V2", and clicked **← Your questionnaires** in the same tick. A `fetch` logger showed the `PUT …/answers/…` (200) finished *before* `GET /api/p/home` was sent. The home showed A1 **In progress** at once.
- **Not repeated (covered by J7c and test report 4's T-24):** the 401 case, where the page stays and shows **Not saved**.

### Test results (13 Oct)

- `make lint`: passed.
- `make test`: Go ok; Vitest 137 passed (web) and 116 passed (clinical).
- `make test-integration`: all packages ok (`internal/httpapi` 10.5 s).
- `make seed` then `make e2e`: **42 passed** (163 s).

### Hand checks for Rahul

- **R-6:** suspend Trust B as Sam Superadmin. The patient's home should then list only Trust A's questionnaires, and B1's episode addresses should be refused. Re-activate Trust B afterwards.

### Commits

These are verification 1's suggested commits:
1. data and API;
2. web and journeys;
3. docs. Planning files that mix S15's changes are committed whole, and the message says so.

## Verification 1 (13 Oct 2026, automatic run)

**Verdict: Needs fixes.** One Major privacy gap (finding 1) and one small save-safety gap (finding 2) should be fixed test-first. Everything else is minor and goes on the pending list. The rest of S15 is solid:
- the account-matching rule (date of birth + contact + first name + surname) is applied the same way at the invite, at sign-in and in the sweep;
- sessions are the account's, and `loadPatientForm` refuses any episode whose record isn't joined to it;
- `/signin` always answers 202;
- sign-in links work once and expire after 15 minutes, and verification links last 24 hours;
- the rate limits use advisory locks;
- each trust's audit log gets its own rows;
- the new routes are on the patient router only.

### Findings, worst first

1. **Major (privacy): a person whose record hasn't joined an account yet can sign in to the other account they share a contact and date of birth with.**
   - **Where:** `apps/api/internal/httpapi/accounts.go:306-328` (`signinDob`) and `AccountsForSignin` in `apps/api/db/queries/accounts.sql`.
   - **What happens:** the name step (409 `code: "name"`) is asked only when **more than one account** has the link's contact and the date of birth. Take twins Sam and Lee on one email. Sam opened Sam's invite, so Sam has an account. Lee hasn't opened Lee's invite yet, so Lee's record is unjoined.
     - Lee asks `/signin` for a link on the shared email. The link is sent, because Sam's account uses that email.
     - Lee enters the shared birthday. Exactly one account matches, so there is no name step, and Lee lands on **Sam's home** ("Hello Sam", Sam's questionnaires, Start or Continue).
     - Report 3's Sammy HomeR3 / Sam HomeR3 probe has the same shape. Only Sam's side was tested ("before Sammy opened her invite, Sam's sign-in had no name step"), not Sammy's.
   - **Reproduced:** I ran a throwaway integration test (deleted afterwards): `newPerson` Sam, Lee = the same person with first name "Lee", both at A1; Sam's invite opened; `signIn(sam.email, sam.dob)`. Lee's home listed only "Sam Person …" and greeted "Sam".
   - **Why it matters:**
     - Rahul's answers 6 and 8 say people sharing a phone or email stay separate. Here one person sees, and can fill in, another person's health questionnaire. That is a clinical-safety problem too.
     - The sign-in page invites it. A twin who heard "sign in with your email" would take exactly this path.
     - Opening Sam's invite in the shared inbox gives the same access, but the invite at least goes to Sam by name of hospital and episode. The sign-in flow implies "this is you".
   - **Fix (recommended, fails safe):**
     - When exactly one account matches but **an unjoined record with the link's contact and that date of birth has a different name key** from the account's, answer 409 `code: "name"` as for twins. It isn't a try.
     - The given names must then match the account. A name that matches no account is a wrong try. The builder may add wording such as "If this is your first time, use the link your hospital sent you."
   - **Tests first:**
     - `TestSignin`: an unjoined twin on the shared contact gets the name step, the account holder's name signs in, the twin's name doesn't, and the twin never sees the other's episode.
     - A J7c step, if cheap.
   - **Option for Rahul (not built without asking):** a sign-in link's right date of birth and names that match an unjoined record are the same proof as an invite, so it could make that person's own account. That changes answer 4, "sign-in for returning patients only".
2. **Minor (answers can be lost silently): "← Your questionnaires" doesn't flush and await the open set's save.**
   - **Where:** `apps/web/src/features/patient/PatientHQ.tsx:136` (`homeLink`).
   - **What happens:** the Episode link (`leave`) and Sign out await `flushRef.current()` before leaving (S14 verification 2's pattern). The new link relies on the unmount's fire-and-forget save.
     - If that save fails (401 or network), the "not saved" message never shows, because the page has gone.
     - Even when it succeeds, `/home` can load before the save does and show "To do" instead of "In progress".
   - **Fix:** the same `onClick` as `leave`: `preventDefault`, then `if (await flushRef.current()) navigate('/home')`.
3. **Minor (local data only): migrations 00014 and 00015 unjoin merged records but leave the contacts those records had filled into the account.**
   - **Where:** `apps/api/db/migrations/00014_account_first_name.sql:22-24` and `00015_account_surname.sql:16-18`.
   - **What happens:** under the old rule, a joining record filled an account's empty mobile or email (`FillAccountContacts`). After the unjoin, that contact still signs in to the other person's account. Open patient sessions were also left running on the account.
   - **Why it's minor:** only Rahul's local database and the per-test databases have run these migrations, all with made-up data. Nothing is deployed (local first). **Record in pending.md** that 00014/00015's data fix-up isn't safe for a database holding real accounts. If a QA or real database ever needs it, a later migration should rebuild each account's contacts from its remaining records and end open patient sessions.
4. **Minor (for S17): `POST /api/p/signin` gives away whether an account exists when sending fails or is slow.**
   - **Where:** `accounts.go:157-163`. A provider failure returns 502 only for contacts with an account, and the send's latency only happens for them.
   - **Why it's harmless now:** capture mode never fails.
   - **Record in pending.md for S17:** answer 202 regardless (log the failure), or send after answering or from a job (S16).
5. **Minor: a contact change revokes the earlier verification link before the new one is sent.**
   - **Where:** `accounts.go:476-485`. If the send fails (502), both links are dead, and the failed request was counted against the rate limit.
   - **Fix:** revoke only after the send succeeds, keeping the new link. Record in pending.md.
6. **Cosmetic: after signing out from a sign-in-link session, the card says "open the link in your text message or email again".** A sign-in link works once, so that link now says "expired". The card's second line, "You can also sign in…", does cover it.
   - **Where:** `apps/web/src/features/patient/PatientHome.tsx:31`.
   - **Possible wording:** "To come back, open your hospital's link again or sign in below." Record in pending.md.
7. **Already in pending.md, confirmed:**
   - sign-in date-of-birth attempts write no audit row (no trust);
   - `clientIP` is `RemoteAddr` until S06;
   - capture mode, including the new unauthenticated `DELETE /api/qa/rate-limits`, must never run in a real deployment.
     - Today `config.Load` accepts only `MESSAGING=capture`, and strict mode must opt in explicitly. When real sending lands, **S17 should add a router test that `/api/qa/messages` and `/api/qa/rate-limits` are 404 unless capture is on.** No such test exists yet; please add this to the S17 pending item.

**Considered and dismissed:**
- **`greeting`'s "newest written with a capital":** records on one account now always share a first name key, so only case differs and the result is right.
- **Both of the invite record's contacts treated as proved:** this was the builder's recorded decision. The invite itself went to both contacts, and a match still needs the full name and date of birth.
- **`ConfirmContact`'s own error mapping:** verification links have no lock, so the cases don't diverge.

### Checks

- **Scope:** everything in the brief is built: accounts, linking, sign-in links, the home in one query, logout, rate limits, PAT-11 and J7c. So are PAT-10 (Rahul's widening) and answers 6–9. Nothing big is there that wasn't asked for, apart from the QA reset route, which is capture-only, as `/api/qa/messages` is.
- **Tests:** all six new routes have integration tests in `accounts_integration_test.go`, covering the happy path, no session, a clinician's cookie, another person or trust, and invalid input. `TestPatientRouter` and `TestStaffRouter` pin the new routes to the patient router. Unit tests: `TestContact`, `TestMask`, `TestGreeting`, `links.test.ts` and `home.test.ts`. The journey is J7c (`e2e/journeys/test_j7c_returning_patient.py`).
- **Isolation:**
  - `ListHome` reads by `account_id` only and leaves out suspended trusts and hospitals.
  - `loadPatientForm` uses `ownedBy`: the record's `account_id` must equal the session's.
  - The staff routes are unchanged.
- **A-20 grep:** comments only (`format.ts`, `patient.go`). The phone country comes from `PHONE_COUNTRY`.
- **Org-name grep:** clean.
- **Secrets and logging:** no secrets. The logs carry errors only, no contacts or names. The sign-in and verification texts name no patient.

### Spot checks (Claude in Chrome, `tester.localhost:3000`, plus one throwaway integration probe)

- `/home` with no session shows the signed-out card with **Get a sign-in link**. `GET /api/p/home` is 401.
- `/signin` at 390 px (iframe): no horizontal scroll (scrollWidth = clientWidth). It shows one box and **Send me a link**.
- `POST /api/p/signin`:
  - an unknown email gets 202 `{channel: "email"}`;
  - `+1 202 555 0143` gets 400 with PAT-11's message.
- An unknown sign-in token's `/dob` and an unknown verification token's `/confirm` get 404 with the right "doesn't work" wording.
- Finding 1 reproduced, as above.

### Test results (13 Oct)

- `make lint`: passed.
- `make test`: Go ok; Vitest 137 passed (web), 116 passed (clinical).
- `make test-integration`: all packages ok (`internal/httpapi` 12.3 s).
- `make seed` then `make e2e`: **41 passed** (174 s).

### Suggested commits (once the fixes are verified)

1. **Patient accounts, sign-in links, the home and contact changes: data, Go and API tests (C2 / S15: data and API).** Contents: `apps/api/**`.
2. **The sign-in page and link, the patient home, confirming a contact, and J7c (C2 / S15: web and journeys).** Contents: `apps/web/**`, `e2e/**`.
3. **Docs: S15 brief, answers, build notes, test plan, test reports, verification and status.** Contents:
   - `docs/sessions/S15-patient-home/**`;
   - `docs/plans/c2-patients-signin.md`, `docs/plans/README.md`;
   - `docs/sessions/README.md`, `docs/decisions.md`, `docs/pending.md`;
   - plus the planning files that mix S15 changes, committed whole and said so in the message.

### Hand checks for Rahul

- **R-6:** suspend Trust B as Sam Superadmin. A patient's home should then list only Trust A's questionnaires, and B1's episode addresses should be refused. Re-activate Trust B afterwards.

### For the builder

> Verification found problems: read `docs/sessions/S15-patient-home/verify.md`, fix them test-first, then ask me for a retest.

Findings 1 and 2 must be fixed. Findings 3–6, and the S17 note in finding 7, go into pending.md (S15).
