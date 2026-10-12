# Decisions, conventions and lessons

**What this file is:** the project's memory. It replaces the Claude chat history, which doesn't travel to other machines or developers.

- **What it holds:** every decision made in the Claude sessions so far, the reason for each, Rahul's working preferences, and environment gotchas.
- **Keeping it current:** add new decisions at the top of §2 in the same format.
- **Superseded decisions:** mark them as superseded, with the date and what replaced them; never delete them.

Related:
- [CLAUDE.md](../CLAUDE.md): the short hand-over for Claude;
- [plans/README.md](plans/README.md): the roadmap;
- [saas-requirements.md](saas-requirements.md) §0: the SaaS decision table, in more detail.

## 1. Working with Rahul (the product owner)

- **Simplest thing that works.** He pushes back on extra services, steps and abstractions. Explain the cost and the reason up front, especially for managed cloud services.
- **Ask, with a recommendation.** For stack and product choices, give 2–4 options with the recommended one first, then wait. Don't decide alone. But fix routine problems yourself (e.g. "you can fix the docker yourself and then start").
- **Check the official docs before planning** (SurveyJS, AWS, Amplify…). Several plans changed because the docs said otherwise: Amplify supports Next.js ≤ 15 only; Cognito has no native magic link; Aurora Serverless takes about 15 s to wake.
- **He reads Go, not TypeScript.** Keep backend logic in Go, and explain TypeScript he must review.
- **Lifebox is not the architecture reference** (2 Oct). Lifebox (the incumbent product he knows well) is evidence of what hospitals need. Never justify a tech choice with "Lifebox does it".
- **Functionality over content fidelity.** Gaps in copied Lifebox questionnaire content aren't a priority; working features are.
- **Show the tests running.** He likes a visible-browser run (headed) when checking a flow.
- **Personal account only.** The code lives in his private GitHub (`rlamba89/poised`). Nothing goes to the Lifebox org, and the Lifebox company's GitHub organisation name must never appear anywhere in the repo, its docs or its commit messages. It was scrubbed from the git history on 2 Oct, so describe it, never spell it out.
- **Messages he'll send on** (e.g. IT tickets) shouldn't mention Claude.
- **Healthcare: nothing may break silently.** Hence the testing rules (§2, 2 Oct).
- **Ready, not built** (8 Oct). Don't build for a market or feature before a customer needs it. But don't hardcode anything that would make it a rewrite later: another country must be config, not code.
- **The local dev database is disposable** (8 Oct). Rahul is the only developer and nothing is in production, so a builder may drop and rebuild the local `sj` database to match the code, and needn't keep other branches working. Copy back anything that can't be recreated from the repo (today: HJE Full HQ, see §4).

## 2. Decision log (newest first)

Format: **decision**, then the reason, then where it's recorded in more detail.

### 12 Oct 2026: S05, per-country config and the router split (builder, automatic run)

- **`internal/config` is the only home for per-country values** (A-20). It reads `APP_URL`, `DEFAULT_TIME_ZONE`, `DEFAULT_LOCALE` and `PHONE_COUNTRY` from the environment.
  - **Locally,** a missing one falls back to the UK.
  - **With `CONFIG_STRICT=true`** (AWS, from S06), a missing one stops start-up, and one error names them all. A bad value (an unknown zone, `en_GB`, `gb`) stops start-up in both modes.
  - **A guard test** (`internal/config/literals_test.go`) fails on a locale, IANA zone or AWS region literal in `apps/api`, `apps/web/src` or `packages/clinical/src` (tests and comments aside).
  - **Reason:** A-20 rule 2, enforced by a test rather than by review.
- **The time-zone database is built into the Go binary** (`time/tzdata`). **Reason:** a zone mustn't depend on what the host (Lambda's base image) has installed.
- **The dev login, `/api/dev/users` included, exists only when `DEV_LOGIN=true`.** `make dev` and `make staff-api` set it; nothing else does. **Reason:** F3 Step 1. The user list is part of the stub and lists everyone.
- **`GET /api/config` returns only `{locale, timeZone}`.** It needs no sign-in, and both routers serve it. **Reason:** the sign-in page and the patient's link need it first, and nothing secret or internal (web address, phone country) needs to reach the browser yet.
- **The web app loads `/api/config` once, before the first render** (`apps/web/src/main.tsx`, `src/lib/config.ts`), and every date goes through `src/lib/format.ts`.
  - If the fetch fails, it renders anyway with the browser's locale and logs a console error. The pages show their own "can't reach the server" errors.
  - **Times are still shown in the browser's time zone;** only the locale comes from config. S08's hospital time zones can decide which zone staff screens use.
- **`httpapi.Deps` is `{Pool, Config}`.** The token secret comes from the config, so there's one source for every setting.
- **Routers:** `NewStaffRouter` and `NewPatientRouter` run one each (`cmd/staff-api`, `cmd/patient-api`, through `internal/serve`). `NewRouter` mounts both for `make dev` (`cmd/api`) and the tests.
  - A route a router doesn't serve gets Go's plain-text 404. **Reason:** a JSON catch-all would also turn every wrong-method 405 into a 404.
- **`make patient-api` / `make staff-api` (`ADDR`, default `:8081`)** run one part alone beside `make dev`. Point a second Vite at it with `API_URL=http://localhost:8081 npm run dev -w apps/web -- --port 3001`.
- **Tests:** `apitest.NewWith(t, Options{Router, Env})` starts a test server with another router or environment, through the real `config.Load`.
- **`CONFIG_STRICT=true` doesn't refuse `DEV_LOGIN=true`** (S05 verification). **Reason:** the QA environment on AWS runs strict *and* needs the dev login (F4); the guard against the dev login in Training or Production is F4's CDK synth check.
- **Accepted after testing (report 1):** `DEFAULT_LOCALE` is checked only for shape, so a well-formed but unknown tag (`zz-ZZ`) starts, and the browser quietly uses its own locale. **Reason:** each country's value is set once, in `infra/countries.py` (S06), and Go has no list of locales the browser knows without adding a library. `/api/config` has no `Cache-Control` header; it's fetched once per full page load, which is cheap.

### 12 Oct 2026: answers for the local phase, asked up front by the conductor

Rahul answered every "Ask Rahul first" question for S05–S16 at the start of the first `/run-sessions` run. Each session's full answers are in its `answers.md` (`docs/sessions/<id>-*/answers.md`).
- **Git:** the 12 Oct planning-doc changes (skills, plans, briefs, S17) were **left uncommitted** at the start of the run. Each verifier commits its own session's files. Where a file holds both (e.g. `docs/plans/README.md`), the planning changes go into that session's docs commit.
- **S05: dates use the locale from `GET /api/config`,** not a hardcoded `en-GB`. **Reason:** it removes a UK assumption (A-20).
- **S08:**
  - **a time zone per hospital** (A-20). **Reason:** Canada, the US and Australia span several zones.
  - **Roles map** as author/reviewer/publisher → super clinician, hospital_admin → admin, viewer and clinician → clinician, with a reseed.
  - **The `dev` branch (`557be2c`) is ignored entirely.** Builders don't read or reuse it.
  - **A hospital-scoped user sees the trust's patient record** (name, date of birth, contacts), but only their own hospital's episodes.
- **S10:**
  - **a logo URL field now,** replaced by an upload in C8;
  - **all of ORG-04's settings are built now** (trust-only and hospital-overridable), even ones nothing reads until later sessions. This widens the brief at Rahul's choice; the recommendation was only what's used now.
- **S11: one audit row per request** that reads or changes patient data, list screens included. **Nothing is deleted for now.**
- **S13:**
  - **`national_id` + `national_id_type`** (A-20);
  - **patient search** as in C2's "New episode page" design (confirmed);
  - **duplicate matching uses email and mobile too.** On Create:
    - the same first name, surname, date of birth **and** the same email or mobile → link silently to the trust's record, as with a matching NHS number;
    - the same name and date of birth with a different or missing contact → a warning plus "Create anyway". Nothing about the other record is shown.
  - **Reason (Rahul):** in Lifebox, email is what has helped here, and the NHS number is often empty. Name + date of birth alone was rejected, because two people in a large trust can share both.
- **S14: before the date-of-birth check, the patient API returns only the hospital's name.** After the check, it returns the HQ without the date of birth or any identifier the patient didn't type. This settles the 8 Oct open item ([pending.md](pending.md) §3).
- **S15:**
  - **patients can update their own email or mobile now (PAT-10),** verified before use. This widens the brief at Rahul's choice.
  - **Sign-in links are rate-limited** to 3 per contact per 15 minutes and 10 per IP address per hour.
- **S16:**
  - **reminders are sent at any time,** with no 09:00–19:00 window;
  - **starting the HQ stops reminders,** not only submitting it.
  - Both are Rahul's choice over the recommendation.

### 12 Oct 2026: local first, and automatic session runs (session ae16fe2f)

- **Local only until the first demo or customer.** No AWS for now. **Reason:** nothing needs AWS until someone outside Rahul's laptop must use the app, and building locally lets automatic runs go without AWS stops.
  - **This reverses the 2 Oct order** (F3–F4 before C1). The cost is that the AWS hosting setup and its near-£0 idle cost are proven later.
  - **The local phase:** S04, S05, S08–S11, S13–S16.
  - **The AWS phase**, later and with Rahul there: S06 (it now also takes the Lambda half of F3), S07, S12, and a new **S17** (jobs Lambda, EventBridge schedule, real SES/SMS).
  - **Details:** [sessions/README.md](sessions/README.md).
- **Stand-ins for the AWS pieces:**
  - **email and SMS:** capture mode, with a captured-messages page;
  - **staff sign-in:** the dev login instead of Cognito;
  - **files:** MinIO instead of S3 (C8);
  - **scheduled jobs:** a timer in the local server instead of EventBridge.
- **Write the code as if it already runs on Lambda,** so the AWS phase is wiring, not a rewrite:
  - jobs are one function, `jobs.RunDue(ctx, now)`;
  - no state in the server's memory;
  - no reliance on local disk;
  - files go through presigned URLs.
- **Docker is unchanged:** only Postgres runs in it (later MinIO too). The API, web and jobs run on the Mac with `make dev`. Rahul asked about dockerising everything. It was kept as is (the 1 Oct decision) for fast feedback, and because Docker is unreliable on his Mac (§4).
- **Automatic runs: a conductor session (`/run-sessions`).**
  - **What it does:** for each session in the run order, it runs:
    1. a builder agent;
    2. a new tester agent for every test round, with the reports going back to the same builder;
    3. a verifier agent;
    4. a local commit.
  - **What it handles:**
    - it asks every brief's questions up front;
    - it restarts `make dev` before each test round, because `make dev` doesn't reload Go;
    - it sends phone notifications;
    - it resumes from `docs/sessions/run-state.md`.
  - **Rahul's standing permission (12 Oct):** the verifier commits locally to **`master`** after a Verified verdict, without asking. **Nothing is ever pushed.**
  - **Limits:** at most **4 test reports** and **2 verifications** per session, then it stops and asks.
  - **Testers mark cases their tools can't do** (blocking a network request, the real print dialog) as **hand checks for Rahul**, not failures.
  - **Checked 12 Oct:** an agent started by a session can use Claude in Chrome. It needs `createIfEmpty: true` to get its own tab group.
  - **Rejected:** a scripted workflow, and a shell loop over `claude -p`. Neither can pause to ask Rahul a question and carry on.

### 12 Oct 2026: S04, the frontend on React + Vite

- **The UI is renamed from "Lifebox" to "Poised" in S04** (Rahul, 12 Oct). This covers the browser tab title, the hospital header, the login page title ("Lifebox Authoring" becomes "Poised") and the patient page header. Code comments that cite Lifebox as layout evidence stay.
  - **Reason:** every layout is rewritten in S04 anyway, and it's the only visible change S04 allows.
  - **Details:** [S04 brief](sessions/S04-react-vite/brief.md).
- **Node 22.22 or newer is required** (was 20.9; S04 builder). React Router 8 needs it.
  - **Reason:** React Router 8 is the current major version, and Node 20 has been end-of-life since April 2026. F4's build images and the `evaluate` Lambda should use Node 22 or newer.
- **Each page's code loads when the page is first opened** (React Router's route `lazy`, in `apps/web/src/routes.tsx`; S04 builder).
  - **Reason:** with plain imports, any first page load downloaded every screen (2.3 MB of JavaScript), the editor included on the patient's phone. Next.js had split the code per page by itself. Now the first load is 0.53 MB of JavaScript.
- **Pages are keyed by their URL IDs, and the hospital layout by the hospital** (S04 builder).
  - **Reason:** Next.js started a page afresh when the ID in its URL changed; React Router reuses the component. Keying keeps the old behaviour, so one episode's or hospital's state never lingers on the next.
- **A page whose code can't be downloaded reloads the app once, then shows "Something went wrong" with Reload** (S04, after verification 1). The root route's error screen is `apps/web/src/pages/RouteError.tsx`; a `sessionStorage` timestamp allows at most one automatic reload in 10 seconds.
  - **Reason:** with per-page loading, anyone with the app open across a deploy gets a missing file on the next page they open (and a patient on a patchy signal can too). Next.js recovered with a full page load; React Router's default is a raw error screen with no way out.
- **Details:** [S04 build notes](sessions/S04-react-vite/build-notes.md).

### 12 Oct 2026: S03, the end-to-end journeys (builder's choices, for verification)

- **The patient's journey (J3) runs in WebKit with Playwright's iPhone 13 profile,** not in Chromium at phone size.
  - **Reason:** it's the closer match to a real iPhone (Safari's engine), and F1 already installs WebKit.
  - The `phone` fixture keeps its trace through pytest-playwright's private `_artifacts_recorder`; check it when upgrading pytest-playwright.
- **Each journey sets up its own data through the API** (`e2e/support.py`), as the seeded users, with unique names. J4 sends the patient's answers through the patient-link API; J5 also validates and completes the review through the API. Nothing is deleted.
  - **Reason:** journeys must not depend on each other or their order (testing rule 5), and the UI is used only for the journey being tested.
- **No `data-testid` was added.** Everything is found by role or label, apart from three ids that already existed, listed in `e2e/README.md`.
- **`make e2e ARGS=…` passes options to pytest** (e.g. `ARGS=--headed`), and **`make e2e-setup PYTHON=…`** names the Python 3.12+ to use (default `python3.12`).
- **Details:** [S03 build notes](sessions/S03-e2e-journeys/build-notes.md), [e2e/README.md](../e2e/README.md).

### 11 Oct 2026: finding the patient when creating an episode

- **Matching a patient to an existing record is silent, and shows nothing from other hospitals** (Rahul, 11 Oct).
  - **What staff see:** they type the patient's details from the referral, and the server links the episode to the trust's existing record when the identifiers agree.
  - **What staff never see:**
    - which other hospital has seen the patient;
    - the patient's episodes, answers or notes there;
    - any stored value they didn't type.
  - **A mismatch** (same NHS number, different date of birth or name) is reported without showing the stored values.
  - **Other trusts' records are never searched.**
  - **Reason:** privacy, and no sharing of information Hospital A doesn't need. Claude's first proposal showed "last seen at Hospital B" and earlier episodes, and Rahul rejected it.
- **The New episode page is rebuilt in C2,** as a dedicated page. It replaces the drawer and its Existing/New patient tabs (Rahul, 11 Oct).
  - **Reason:** silent matching needs C2's trust-wide records keyed by NHS number.
  - **Staff can search patients, but only their own hospital's** (Rahul, 11 Oct). C2's plan had a trust-wide search, and it's now hospital-scoped.
  - **The search must work with millions of patients:** today's dropdown doesn't.
  - **The design** is in [plans/c2-patients-signin.md](plans/c2-patients-signin.md): server-side search as you type, at most 20 results, and a precise term required. Its details (what counts as "this hospital's patient", the search terms) are Claude's proposal, for C2's build session to confirm with Rahul.


- **The patient's autosave gets the same revision check as the clinician's** (`PUT /api/p/{token}/answers/{cid}`).
  - **Reason:** a patient with the link open on a phone and a laptop loses answers in the same way when the older one saves last, and just as silently. It's the same pattern on the same table.
  - **Details:** [S02a brief](sessions/S02a-answer-revisions/brief.md).
- **S02a also fixes the episode PATCH status write-back.**
  - **The fix:** the UPDATE succeeds only while the status is still the one loaded; otherwise the PATCH gets a 409.
  - **Its own commit,** test-first.
  - **Reason:** it's small and the same idea. It was found by S02's verification (pending.md §1a, and S02's verify.md item 2).
- **S03's J4 gains a two-tab step:** a stale tab's save gets the "changed somewhere else" message.
  - **Reason:** the integration tests prove the 409, but only an end-to-end test can guard the browser's message and Reload.
- **Any 409 on an answers save stops autosaving and offers Reload,** not only a stale revision. That includes "the review is already complete" and the patient's "already sent" (S02a builder; it passed manual testing on 9 Oct).
  - **Reason:** in every case the page is out of date, and Reload shows the truth. The refused change stays on screen, locked, until Reload, so nothing typed vanishes before the user reads why.
- **The patient page doesn't save a section the server already has** (S02a builder, 9 Oct; corrected after S02a's verification the same day). Leaving a section used to save it every time.
  - **"Already has" means what the server accepted, not the page's own copy.** The first version compared with the page's copy, so a failed save was never sent again (verify.md finding 1).
  - **The section list, "Complete" and "Send my answers" go by what the server has,** so a section whose save failed can't look complete or be sent.
  - **And a section with a change that hasn't saved isn't complete either** (corrected again after verification 2, 11 Oct). A section that was already complete, then changed while offline, used to stay Complete, and the HQ could be sent without the change. `isSetReadyToSend` in `packages/clinical` decides it. After a failed save, the section's label says "Not saved".
  - **Reason:** with the revision check, a tab left open on another device would otherwise get the conflict message for doing nothing. Before S02a, it would have overwritten newer answers with stale ones. The clinician page already worked this way.
  - **Details:** [S02a build notes](sessions/S02a-answer-revisions/build-notes.md).
- **Race tests make the race happen in a fixed order with `meanwhile`** (`apps/api/internal/httpapi/cast_integration_test.go`).
  - **How:** the conflicting change holds a row lock until `pg_stat_activity` shows the request waiting, and only then commits.
  - **Reason:** two concurrent HTTP requests would pass or fail by timing. This way it goes through the real API and database, with no fakes.
  - **Use it for any future "between the read and the write" bug.**

### 8 Oct 2026: S02, what the workflow integration tests prove

- **Lost updates on answers are prevented by a revision check, not a lock** (Rahul, 8 Oct). Each answers row gets a revision. A save must send the revision it started from; if someone saved in between, it's a 409 and nothing is written. This is the same pattern as Question Set saves.
  - **Reason:** S02's manual test found that a stale tab's Validate silently overwrote a newer answer (X-1). A per-person lock wouldn't stop one person's two tabs. A lock needs expiry, a "still here" signal and a take-over button, and still needs this check after a lock expires.
  - **Built in session S02a** ([brief](sessions/S02a-answer-revisions/brief.md)), after S02 and before S03.

- **The integration tests prove what Go enforces, not the two workflow rules the browser applies** (Rahul, 8 Oct). Two of F1's "must also cover" items are browser rules:
  - **"Complete review is refused until every set is validated."** The server refuses only when the episode isn't Ready for review. The browser enables the button once every *visible* set is validated, and visibility depends on Question Set conditions, which Go can't evaluate (the key rule in CLAUDE.md).
  - **"The first write copies the patient's answers."** The browser pre-fills the clinician view from the patient's row and sends the full data. The server stores a separate clinician row and leaves the patient's row as it was.
  - **Reason:** S02 makes no behaviour changes beyond the two its brief asked for (the episode PATCH `null` fix and the recover middleware), and the server can't know which sets are visible without evaluating expressions. S03's J4 covers both rules end-to-end. The gap is in [pending.md](pending.md) §1a, for C6 (the `evaluate` Lambda can work out the visible sets).

### 8 Oct 2026: S01, the integration test harness (session a8c9de3a)

All of these are detailed in [sessions/S01-integration-harness/build-notes.md](sessions/S01-integration-harness/build-notes.md) and the F1 Status section.

- **Integration tests: one Postgres database per test, copied from a template.** `apitest.Main` migrates `sj_test_template` with goose (as a library); `apitest.New(t)` copies it (about 0.3 s) and serves the real router over `httptest`.
  - **Reason:** every test starts clean and tests run in parallel, with no clean-up code in the tests.
- **A Postgres advisory lock is held while a test package runs, and leftover `sj_test_…` test databases are dropped at the start.**
  - **Reason:** `go test ./...` runs packages in parallel, and two packages would otherwise drop each other's template. Killed runs would otherwise leave databases filling the disk.
- **`make test-integration` uses `-count=1`, and `make lint` vets with `-tags integration`.**
  - **Reason:** Go's test cache doesn't know the database changed, and untagged vet would skip the integration test files.
- **How the five cases apply where one can't exist** (agreed by Rahul, 8 Oct):
  - for a read route, the "wrong role" case asserts that a viewer **can** read;
  - for codes and categories (reference data shared by every hospital), the "another hospital" case asserts a user in another hospital gets the same results;
  - every refusal also asserts that nothing was written.
- **A bug found by the integration tests is fixed with a unit test of a small pure function, and committed before the route tests.**
  - **Reason:** each commit must pass on its own, and the route-test commit already contains the case that exposed the bug.
  - S01 found two bugs: a 500 on `?page=99999999999` in the questionnaire list, and a crash on a `null` field in a chapter PATCH.

### 7–8 Oct 2026: countries, superadmin and the platform features (session ae16fe2f)

- **Rahul's six platform features are already planned in C1 and C2.** No new plan was needed. The six:
  - superadmin;
  - hospital onboarding;
  - staff and patient invites;
  - permissions;
  - staff in several hospitals;
  - patients in several hospitals.
- **Superadmin: only Rahul, created by hand, with no screen** (A-21).
  - Reason: there will be one or very few, and no screen means nothing to build or attack.
  - A small command creates the Cognito user and the `platform_admins` row. It's run once per environment and country ([plans/c1-trusts-staff.md](plans/c1-trusts-staff.md)).
- **Residency rules usually cover where data is processed and accessed, not only where it's stored.** Rahul expected storage only. Research on 8 Oct covered Canada, the US, the EU, Ireland, Australia and New Zealand ([report](../reports/Health%20data%20residency%20storage%20vs%20processing.md); notes in `research_notes/`). It found:
  - **Storage-only rules exist only in Texas, Florida and France (HDS).**
  - **The EU's GDPR counts remote access from the UK as a transfer.** It's allowed only because of the UK adequacy decision, renewed until Dec 2031.
  - **Some rules cover access, use or processing:** Nova Scotia, Quebec and Alberta; Germany, Italy and Spain; Australian state rules.
  - **Hospital tenders and contracts often add support location,** e.g. an HSE tender's "supported from within the EEA" and Texas Medicaid contracts.
  - **London is lawful with the right contract** under US federal law (HIPAA), and in Ontario, New Zealand and Irish law.
  - **It's research, not legal advice.** A local lawyer confirms the rules before the first contract in a new country.
- **One deployment per country, each at its own web address. Only the UK runs now, and nothing is hardcoded for the UK** (A-20, which supersedes A-19).
  - **How we got there:**
    - 7 Oct: Claude proposed a copy per country. Rahul rejected it as too much to maintain alone.
    - Rahul then proposed code in London with each country's database in that country (the same idea as A-19). The research showed that only helps in Texas, Florida and France.
    - 8 Oct: Rahul agreed to build only the UK, but to make a new country quick to spin up, at its own subdomain.
  - **Why it's little extra maintenance:** one codebase, one build and one pipeline. A fix is one push that deploys every country, and every country's alarms go to one email.
  - **Details:** [saas-requirements.md §13 "Countries"](saas-requirements.md). The groundwork is in F3 (`internal/config`) and in F4:
    - `infra/countries.py`;
    - a CDK test that synthesises a made-up second country;
    - the "add a country" runbook.
  - **Dropped from A-19:**
    - `data_region`;
    - `store.ForOrg`;
    - the control-plane / data-cell split;
    - `account_records`;
    - the jobs loop over regions.
- **The plans are built in work sessions** (8 Oct; [sessions/README.md](sessions/README.md)). F1–F4 and C1–C2 are split into S01–S16, each with a `brief.md`.
  - **Three roles, each in its own Claude session,** run by project skills:
    - **builder** (`/build-session`): builds test-first and writes `test-plan.md`;
    - **tester** (`/manual-test`): a new session for every round, which drives Chrome like a manual tester and never edits code;
    - **verifier** (`/verify-session`): reviews, runs everything, spot-checks, and offers the commits.
  - **The loop:** the builder fixes what each test report finds, until a report says everything passed.
  - **Nothing is committed until verified,** and a session starts only when the one before it is verified and committed.
  - **Reason:** Rahul wants to start each session himself, with independent manual testing in a real browser, and a check before anything is committed.
- ***Proposed, not agreed*** (to confirm when C1 and C2 start):
  - `national_id` + type instead of an `nhs_number` column; *agreed 12 Oct (S13 answers)*;
  - a time zone per hospital; *agreed 12 Oct (S08 answers)*;
  - dates formatted in the deployment's locale; *agreed 12 Oct (S05 answers)*;
  - the UK on `uk.<domain>` from day one;
  - one AWS account per environment, with a region per country;
  - support tooling that never shows patient answers, plus customer-approved break-glass access (OPS-06).

### 2 Oct 2026: SaaS productionisation (session bf5d9b3a)

All of these are in [saas-requirements.md](saas-requirements.md) §0 and [plans/](plans/README.md).

- **Name: Poised** (was sj-demo).
  - Go module `github.com/rlamba89/poised/apps/api`; npm packages `poised`, `@poised/web`, `@poised/clinical`.
  - **Kept on purpose:** the CSS prefix `sj-`, the local Postgres user and database `sj`, and the docker volume `sj-demo_dbdata`, pinned in `docker-compose.yml` so local data survives the folder rename.
- **Repo:** private, at `github.com/rlamba89/poised`, branch `master`. On 2 Oct the history was rewritten with `git filter-repo` to remove the Lifebox organisation name, and the GitHub repo was deleted and recreated so no old commits remain.
- **Fresh app, no historic data:** schemas can be reshaped freely before go-live.
- **Tenancy:** a **trust** (the tenant and data controller) has many **hospitals**. Settings resolve platform default → trust → hospital, and each setting says whether a hospital can override it.
- **Roles:** `clinician < super_clinician < admin`, plus our platform admin. **Everyone in a hospital sees all its data**; roles only limit what people can do.
- **Patients:** one login account, plus a record per trust identified by **NHS number** (optional, check-digit validated). The hospital number is only a label on the episode.
- **Patient sign-in:** a **magic link (SMS or email) + date of birth**, built in Go.
  - Reason: Cognito has no native magic link; codes are friction for older patients.
  - A **Continue** button before using the link stops email scanners from consuming it.
- **Staff sign-in:** **Cognito** (managed login + TOTP MFA). After the OAuth callback, Go sets our own session cookie.
- **Pre-fill:** any question answered in an earlier HQ is pre-filled in a new one, **with no expiry**.
  - Matching is by **question key** (the SurveyJS `name` / stable ID).
  - There's a "Same as…" link, "Always ask fresh", and locked option values.
- **ASA:** platform defaults per code, which can be changed per HQ.
  - **Rules per HQ.** Defaults: ASA 1–2 with no red flags → **auto-triage straight to Ready for admission**; ASA ≥ 3 → a "Book anaesthetic review" task + flag + anaesthetist ASA required.
  - The clinician can change the target per HQ.
- **D-1: server-side evaluation.** A Node **`evaluate` Lambda** runs `packages/clinical` on patient submit.
  - Reason: a patient's browser isn't trusted, and Go can't evaluate SurveyJS.
  - This **supersedes, for this one case, the 30 Sep rule "no Node service"**.
- **"Capture" Group/Section:** filled in only by clinicians.
- **More than one Clinical Summary per page:** the summary becomes an element. This needs SGN-04 in requirements.md updated.
- **Videos:** hosted on **YouTube or Vimeo** (no-cookie / `dnt=1` embeds), never by us. There's a built-in suggested library plus author links.
- **Assessments:** **STOP-Bang first**, then DASI, PRISMA-7, Apfel, MUST, Falls, VTE, Rockwood. **MMSE is deferred** (copyrighted).
- **PROMs:** a baseline plus follow-ups after discharge.
  - **EQ-5D-5L and the Oxford Hip/Knee Scores are built now.** Each is **opt-in per trust, with a per-hospital override**, and the licence cost may become a customer add-on. Licensors aren't contacted for now.
  - Free recovery check-ins are always available.
- **Files:** **no virus scanning**, a 25 MB limit, and no patient–clinician messaging yet.
- **Out of scope:** EPR integration.
- **Cloud: AWS eu-west-2**, serverless, **near-£0 when idle before the first customer**.
  - Other **data** regions per trust must be possible later (code stays in London). The groundwork is a `data_region` per trust, `store.ForOrg`, no cross-trust SQL joins of patient data, and no patient data in control-plane tables.
  - *Superseded 8 Oct by one deployment per country (A-20); see the 7–8 Oct entry above.*
- **Compute: 4 Lambdas from one Go codebase** (`staff-api`, `patient-api`, `jobs`) + the Node `evaluate`.
  - Reason: separate permissions and blast radius for the public patient API, without one function per route.
  - Lambda Managed Instances, durable functions and SnapStart were checked; none is needed.
- **Database: plain Postgres.** **Neon** (AWS London, free, scales to zero) until the first customer, then **RDS**.
  - RDS rejected for now: it can't scale to zero, and it needs a VPC + NAT.
  - Aurora DSQL rejected: no extensions (we use `pg_trgm`).
  - Aurora Serverless v2 rejected: about 15 s to wake.
  - Supabase rejected: the free tier pauses after 7 days.
- **Frontend: a React + Vite + React Router SPA on Amplify Hosting** (plan F2).
  - Reason: Amplify supports Next.js ≤ 15 (we're on 16.3), and the app uses no server features.
  - Vercel rejected: $20/seat, no commercial use on Hobby, a second platform, and its proxy would carry patient data.
  - A static S3 export was rejected by Rahul.
- **Infrastructure: AWS CDK in Python + CodePipeline** (CDK Pipelines). No SAM.
- **Messaging:** SES (email) and AWS End User Messaging (SMS).
- **Testing:**
  - TDD unit tests;
  - **integration tests (preferred) = API → real database only**;
  - **Playwright + Python e2e in `e2e/`**, in this repo, not a separate one.
  - Full rules in [plans/README.md](plans/README.md).
- **Plan structure:** foundation first (F1 tests → F2 React/Vite → F3 Lambda entry points → F4 AWS + pipeline), then core C1–C10, each testable before the next starts.

### 1–2 Oct 2026: the episode workflow (sessions dc7d83c0, 74e1fcfc, 57814611)

Mostly recorded in [plan-workflow.md](plan-workflow.md).

- **Build the whole workflow thinly end to end first**; platform work (auth, hospitals, accounts) later. Reason: see the core flow working first.
- **Answers per actor:** the patient's copy is frozen at submit, and the clinician's working copy is final. The POA Summary marks changed answers, and the patient's originals are kept for audit.
- **Triage is only a status** (Ready for review), set on submit.
- **The patient link `/p/<token>` needs no login, and Publish is one click.** Validation is per Question Set; statuses follow Lifebox's list. (The link is superseded by magic-link sign-in in plan C2.)
- **Go strips clinician-only content** for patients, and disclosures are computed in the browser at view time.
- **Clinician answers are locked** after the review is complete. There's no re-open yet (plan C3 adds it).
- **Clinician boxes sit in the row of the patient question above them**, with no "linked to" setting. Same as Lifebox's layout.
- **Only Postgres runs in Docker.** The API and web run with `make dev`, so there's no image rebuilding.
- **The manual test plan** ([manual-test-plan.md](manual-test-plan.md), MT-01…MT-68) was fully automated headless (63/63 passing on 1 Oct) using gitignored scripts in `spikes/mt/`. Plan F1 replaces these with committed Playwright tests.

### 1 Oct 2026: Lifebox-style redesign and SurveyJS features (session 5ac0c8c2)

Mostly recorded in [plan-redesign.md](plan-redesign.md), [plan-features.md](plan-features.md) and [lifebox-ui-notes.md](lifebox-ui-notes.md).

- **Our own editor; the SurveyJS Creator is dropped.**
  - Reasons: no Creator licence is needed, and content is edited as plain JSON data. survey-core still runs the preview, runtime and outputs.
  - **Copy Lifebox's layout and behaviour, not its styling**: the Structure tree, card canvas, Settings / Disclosures / Logic panel, and Lifebox terms (Question Set, Page, Disclosures).
- **Our output model stays richer than Lifebox's:** review flags + ASA on disclosures. Lifebox has no flags, and its ASA is disabled.
- **"Create the HQ" means rebuilding the HJE Full HQ content** in our tool, from screenshots ([hje-full-hq-content.md](hje-full-hq-content.md)).
  - Sets 1–4 were imported.
  - All 11 sets + disclosures need `tools/export-lifebox-hq.js` run in the Lifebox Author tool, which hasn't been done yet.
- **The logic Builder uses SurveyJS's own `ConditionsParser`**, so the Builder and the Code tab always agree. It grew from a single Is / Is not condition to All / Any / None / Not all at any depth.
- **Other choices:**
  - **Test cases** are stored in the chapter JSON (`testCases`) and stripped for patients.
  - **Question Set conditions** use `chapterVisibleIf`.
  - **Skip rules** record their `page`.
  - `calcKind` and `textFormat` are stored explicitly, not inferred.
- **Saved option lists are copied into a question, not linked.** This protects stable IDs. *Still awaiting Rahul's confirmation.*
- **Editor implementation choices** (not in other docs):
  - **Selection is kept in the URL** (`?set=&page=`).
  - **Autosave** is debounced at 600 ms, with the 409 revision guard. Undo is a list of snapshots.
  - **The Structure tree stays mounted** (hidden) while Settings is open; unmounting it collapsed the expanded sets.
  - **Mantine multi-selects are `searchable`**, otherwise the input can't be clicked.
  - **Decide behaviour by question type, not `inputType`.**
- **Clinician two-column layout:**
  - The grid sits on `.sd-page__content`, and `.sd-row:has(.sj-clinician)` goes in column 2.
  - The clinical summary panel `cs_<page>` (`cs_notes_` / `cs_comments_`) is injected in the clinician view only.

### 30 Sep 2026: first plan and demo slice (sessions 684ab54e, 28534d3d)

Recorded in [plan.md](plan.md) (§2, §6, §10, §12) and the README's "Deviations" table.

- **Stack:**
  - a monorepo (npm workspaces + a Go module + a Makefile);
  - Next.js + Mantine (*superseded 2 Oct by React + Vite*);
  - Go stdlib `net/http`, REST;
  - Postgres with sqlc + goose;
  - **SurveyJS 3.1.2, pinned exactly**;
  - one SurveyJS JSON per chapter;
  - stub login, local only.
- **No Go port of SurveyJS, and no Go evaluation of expressions.**
  - Reason: no non-JS evaluator exists, and a port would drift silently, which is a clinical safety risk.
  - Also rejected: a restricted condition subset, and **goja** (JS inside Go, untried).
- **Trust the frontend** (Rahul's choice over a recommended Node evaluate service).
  - All answer replay (preview, test cases, outputs) runs in the browser through `packages/clinical`, and Go stores and enforces.
  - Test results are tied to the chapter revision.
  - *Partly superseded 2 Oct (D-1): patient submits are evaluated server-side.*
- **Use real Lifebox codes; never invent codes.** Seed CSVs are in `apps/api/db/seed/lifebox/` (276 codes; 21 have no category; ICD-10 `F17.1` text is broken at source).
- **Stable IDs are the SurveyJS `name` / option `value`.** They're letters only (`q_kfbwpx`, `o_tqmd`) and never change once published.
- **Content rules:** clinician-only content is stripped for patients; our own preview page; `/api/*` is proxied; Tabler icons; drafts are hard-deleted.
- **sqlc settings** (`apps/api/db/sqlc.yaml`):
  - `uuid.UUID` and `time.Time` overrides, camelCase JSON tags, pointers for nulls;
  - `emit_interface`, so handlers take `db.Querier` (the fake-querier unit tests use it; new tests use the real database);
  - search arguments joined with `||` need a `::text` cast.
- **Two test HQs** were built for manual testing: "General pre-operative assessment" and "Comprehensive pre-operative assessment (test)" (7 chapters, 76 questions, every option type).
  - They exist **only in Rahul's local database**. The generator script wasn't saved.


## 3. SurveyJS findings (v3.1.2, exact pin)

- **`clinicalOutputs`:** held in memory in an immutable `{ items }` box behind `outputsOf` / `setOutputs`; the JSON stays a plain array. Reason: `setPropertyValue` overwrites existing arrays in place, which breaks redo. (README "Deviations")
- **`useElementTitles: true`** replaces the deprecated `showTitlesInExpressions`.
- **"None of these"** is a normal option with `isExclusive: true`. The built-in None, Other, Don't know and Refuse options don't save custom properties, so outputs on them would be lost.
- **"Yes / No"** is a Select One preset, not a boolean.
- **Question CSS:** `cssClasses.mainRoot` is the question's box; `root` is only the inner input.
- **`model.validate(false)`** checks required answers without showing errors.
- **`showCompleteButton`** (v3) lets the clinician's Validate button replace Complete.
- **Warning-type validators** (`notificationType: "warning"`) don't block Next.
- **No non-JS evaluator exists**, so Go must never evaluate expressions (exception: the `evaluate` Lambda, D-1).
- **`setPropertyValue`** copies into an existing array, which breaks redo. Untyped custom properties default to `string`, so edits within 1 s merge into one undo step.
- **`question.isVisible` stays true inside a hidden panel.** Use `getAllQuestions(true)`.
- **There's no "render if"**; use visible-if, enable-if and required-if.
- **"Is not" / `notcontains` is true when the question is unanswered.** Lifebox behaves the same way.
- **A matrix has `choices`.** Date lists, Medication and Admissions were mistaken for choice questions, and their disclosures never fired (fixed). Branch on the question type.
- **Dispose the old survey model** when switching view or language. Otherwise ResizeObserver throws "scrollWidth" errors (seen with Rating and File upload).
- **Unit tests must call the custom-property registration first**, or every custom flag is reported as unknown.
- **A skip rule fires only once the page's required questions are answered.**

## 4. Environment gotchas

- **Docker keeps dying / the disk fills up (on Rahul's Mac).**
  - **Cause:** Microsoft Defender's network extension (`com.microsoft.wdav.netext.systemextension`) isn't pre-approved by the Mac's management (Jamf). It loops, copying 121 MB into `/Library/SystemExtensions/.staging` every 12–20 s (~29 GB/hour).
  - **Check:** `df -h /System/Volumes/Data`, `ls /Library/SystemExtensions/.staging | wc -l`, `mdatp health`, `systemextensionsctl list`.
  - **Real fix:** allow the extension and give Defender Full Disk Access in System Settings, or IT approves it.
  - **Temporary:** restart, or clear `.staging` with `sudo` (keep the newest copy).
  - Quit Docker before the disk fills. **Never factory-reset Docker**, because it wipes other projects' data.
  - If Docker hangs after a disk-full dialog, force-kill its backend process.
- **Postgres won't start after a full disk** ("bogus data in lock file"):
  ```sh
  docker run --rm -v sj-demo_dbdata:/data alpine rm /data/postmaster.pid
  docker compose up -d --force-recreate db
  ```
- **Long-running dev servers started by Claude** are stopped after their time limit (max 2 h). Run `make dev` in your own terminal for long sessions.
- **Headless browser checks** (`spikes/ui-*.ts`, gitignored, so they won't exist on a new machine; *superseded 12 Oct 2026 by the committed journeys in `e2e/`*):
  - They use Playwright's `chrome-headless-shell` from `~/Library/Caches/ms-playwright/`, via `CHROME_PATH`.
  - A visible run needs `HEADED=1 SLOWMO=100 HEADED_CHROME_PATH=<full Chromium>`.
  - Use an explicit browser context to open several tabs.
  - Plan F1 replaced all this with `e2e/` (S03, 12 Oct).
- **Driving the Lifebox Training site from a script:**
  - It redirects `/author` URLs to `/hospitals` until a hospital is picked.
  - Clicking from VS Code needs "Allow JavaScript from Apple Events" in Chrome.
  - When capturing, screenshot only the Chrome window (`screencapture -l <CGWindowID>`): other apps are open.
- **zsh treats `===` specially**, so avoid it in `echo` separators.
- **npm:** a workspace package needs a `version` field before npm will link it.
- **Go 1.26** is required because sqlc (run as a Go tool) needs it.
- **Docker Desktop hangs:** `docker info` and CLI calls can hang, including on the `docker-ai` plugin. Force-quitting the backend fixes it, but it also stops other projects' containers, so ask first.
- **Invalid HTML in React** (e.g. a Mantine `<Badge>` (a div) inside `<Text>` (a p)) shows as a console error. Check the console after UI changes. (Under Next.js it also showed in the dev overlay's "Issues"; *Next.js removed 12 Oct, S04*.)
- **Playwright selectors:** Mantine `Select` isn't exposed as a textbox, so find it by label or placeholder. The code picker needs a short pause after selecting.
- **Expected console noise:** two 401s from `/api/me` on the login page, and 409s in the duplicate-name and two-editor tests.
- **The exact headless Chrome path used:** `CHROME_PATH=$HOME/Library/Caches/ms-playwright/chromium_headless_shell-1223/chrome-headless-shell-mac-arm64/chrome-headless-shell npx tsx spikes/ui-<name>.ts`.
- **Leftover test data:** published "Workflow test …" questionnaires stay in Hospital A, because published versions can't be deleted. Reset the local database if it gets cluttered.
- **After using another branch, master's tools break** (found 8 Oct, after the `dev` branch ran on 7 Oct).
  - **The local `sj` database keeps that branch's schema** (its migrations 5–8), so `make seed` and the app fail. Rebuild it:
    ```sh
    docker compose exec -T db psql -U sj -d postgres -c "DROP DATABASE sj WITH (FORCE)" -c "CREATE DATABASE sj OWNER sj"
    make migrate seed
    ```
  - **`apps/web/.next/dev/types/` keeps that branch's routes,** so `tsc` in `make lint` fails with "Cannot find module '…/page.js'". Delete the folder; `next dev` regenerates it. (*Only on branches older than S04; master has no Next.js since 12 Oct.*)
- **HJE Full HQ can't be recreated from the repo:** it was transcribed from screenshots. Before rebuilding the database, save it, then load it back after `make migrate seed`:
  ```sh
  docker compose exec -T db pg_dump -U sj --data-only -t questionnaires -t questionnaire_versions -t chapters sj > hje.sql
  ```
  (This works while the database holds only HJE Full HQ; otherwise filter the rows.)
- **On Rahul's Mac, the `docker` CLI isn't on the PATH of Claude's shells** (builder and tester sessions). Docker itself runs; Claude prefixes `PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"` for `make test-integration`.
- **Claude in Chrome's tab can stay "hidden" even with Chrome in front** (S02a testing, 9 and 11 Oct).
  - **Symptoms:**
    - `document.visibilityState` is `hidden`, and no animation frames run;
    - Mantine drawers, modals and dropdowns are in the page but never appear, and screen clicks on them are lost;
    - timers can be slowed to about once a minute.
  - **Check:** run `requestAnimationFrame` in the page; if it doesn't fire within a second, the tab is hidden.
  - **Workaround the testers used:**
    - click the app's own controls from script, and confirm results in the page text and the request log;
    - send a dialog's request directly when its content can't show, and say so in the report;
    - run long scripts in the background and poll their results, because the tool times out after 45 s.
- **End-to-end journeys (`e2e/`, found in S03, 12 Oct):**
  - **SurveyJS hides the real radio input** behind a drawn one, so Playwright's `check()` waits forever ("intercepts pointer events"). Tap the option's text instead: `support.choose(page, question, option)`.
  - **`make dev` adds a Next.js tools button** whose name contains "Next", so `get_by_role("button", name="Next")` matches two buttons. Use `exact=True` (until F2 removes Next.js). *Superseded 12 Oct (S04): Next.js is gone; the journeys keep `exact=True`, which is harmless.*
  - **A failed journey keeps one trace per browser plus one for the API set-up** (`trace-1.zip`…, in no fixed order), in `e2e/test-results/<journey>/`.
  - **To stop a pytest run with one clean message, raise `pytest.UsageError` in `pytest_sessionstart`.** `pytest.exit` from a fixture prints it twice, with pytest noise.
  - **On Rahul's Mac, `python3` is 3.10;** `python3.12` and `python3.13` come from Homebrew. `make e2e-setup` uses `python3.12`.
  - **For F4's pipeline:** CodeBuild needs Python 3.12+ (`PYTHON=` if it isn't `python3.12`), and the journeys sign in with the dev stub, so QA needs it until S12.
- **`docker` may not be on a Claude Code shell's PATH on Rahul's Mac** (found verifying S03, 12 Oct), so `make test-integration` stops at `db-up` with "docker: No such file or directory". Prefix the command with `PATH="/Applications/Docker.app/Contents/Resources/bin:$PATH"`.
- **The Claude in Chrome network log shows every `204 No Content` as 503** (found in S01 testing, 8 Oct). `fetch` and `curl` both return the real 204. Testers should ignore a 503 on a route that answers 204, such as a DELETE, publish or logout. Other error codes are reported correctly.

## 5. Open items (as of 8 Oct 2026)

- **Next work:** plan **F3** ([plans/f3-lambda-entrypoints.md](plans/f3-lambda-entrypoints.md)), session S05 (the local half) verified 12 Oct; its minor follow-ups are in [pending.md](pending.md) §3. Next: S08; F3's Lambda half is S06 (AWS phase). Plan F2 is done (S04 verified 12 Oct; the verifier checked T-19 in the production build, so no hand check is needed). **Rahul to click Print once** on a POA Summary. Plan F1 is done (S03 verified 12 Oct); its six minor e2e follow-ups are in [S03's verify.md](sessions/S03-e2e-journeys/verify.md).
- **The `dev` branch** (on `origin` too; commit `557be2c`, 7 Oct): *settled 12 Oct: ignored entirely* (§2). Delete it later if wanted.
- **Build order: decided 12 Oct.** Local first (S04, S05, S08–S11, S13–S16), then the AWS phase (S06, S07, S12, S17), as in §2.
- **No domain chosen yet.** `<domain>` in the docs is a placeholder.
- **Research files:** `research_notes/` and `reports/` at the repo root are untracked. Either keep them (e.g. move them to `docs/research/` and fix the links in saas-requirements §0 and §13 and in this file) or delete them.
- **What the patient link returns:** *settled 12 Oct* (§2, S14). S14 builds it.
- **The proposed parts of A-20** (above, 7–8 Oct): the national ID, the per-hospital time zone and the locale were agreed 12 Oct. `uk.<domain>`, one AWS account per environment and the support tooling still need Rahul's yes (the AWS phase).
- **App branding** still says "Lifebox" in the UI (`apps/web/src/app/layout.tsx`, `h/[hospitalId]/layout.tsx`, `PatientHQ.tsx`). Rename it to Poised. *Done in S04 (12 Oct), pending verification.*
- **S04's manual tests and verification found small items, none blocking:** they're in [pending.md](pending.md) §3 ("Found by the S04 manual tests" and "From S04's verification"). The worst: the POA Summary lists a clinician's change to a question that a later change hid. **Rahul to click Print once** on a POA Summary (no tester could open the real dialog).
- **Unanswered:** should clinician-only Question Sets render full width in `ValidateSet.tsx`, rather than in an empty two-column grid?
- **Unconfirmed:** whether saved option lists should be copied (as built) or linked.
- **Unsaved test data:**
  - **The comprehensive test HQ is gone.** On 8 Oct the local database held only HJE Full HQ. Rebuild it from `apps/api/internal/apitest/testdata` fixtures if it's needed again.
  - **HJE Full HQ exists only in local databases.** Consider saving it as a file in the repo (§4 has the commands).
- **Content:** HJE Full HQ sets 5–11 and their disclosures still wait on a real Lifebox export (`tools/export-lifebox-hq.js`). Low priority (functionality over content).
- **Small issues:**
  - "Years since a date" opens with an empty list and no message.
  - The Medication, Admissions and date-list disclosure fixes have no test-plan cases.
- **Stale docs:**
  - README says there's no publish workflow, and its seed-user table lacks Cara Clinician and Alex's publisher role.
  - `pending.md` §4 lists Versions and publish as not started.
