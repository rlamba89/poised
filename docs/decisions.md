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

## 2. Decision log (newest first)

Format: **decision**, then the reason, then where it's recorded in more detail.

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
  - `national_id` + type instead of an `nhs_number` column;
  - a time zone per hospital;
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
- **Headless browser checks** (`spikes/ui-*.ts`, gitignored, so they won't exist on a new machine):
  - They use Playwright's `chrome-headless-shell` from `~/Library/Caches/ms-playwright/`, via `CHROME_PATH`.
  - A visible run needs `HEADED=1 SLOWMO=100 HEADED_CHROME_PATH=<full Chromium>`.
  - Use an explicit browser context to open several tabs.
  - Plan F1 replaces all this with `e2e/`.
- **Driving the Lifebox Training site from a script:**
  - It redirects `/author` URLs to `/hospitals` until a hospital is picked.
  - Clicking from VS Code needs "Allow JavaScript from Apple Events" in Chrome.
  - When capturing, screenshot only the Chrome window (`screencapture -l <CGWindowID>`): other apps are open.
- **zsh treats `===` specially**, so avoid it in `echo` separators.
- **npm:** a workspace package needs a `version` field before npm will link it.
- **Go 1.26** is required because sqlc (run as a Go tool) needs it.
- **Docker Desktop hangs:** `docker info` and CLI calls can hang, including on the `docker-ai` plugin. Force-quitting the backend fixes it, but it also stops other projects' containers, so ask first.
- **Invalid HTML in React** (e.g. a Mantine `<Badge>` (a div) inside `<Text>` (a p)) shows in the dev overlay's "Issues". Check it after UI changes.
- **Playwright selectors:** Mantine `Select` isn't exposed as a textbox, so find it by label or placeholder. The code picker needs a short pause after selecting.
- **Expected console noise:** two 401s from `/api/me` on the login page, and 409s in the duplicate-name and two-editor tests.
- **The exact headless Chrome path used:** `CHROME_PATH=$HOME/Library/Caches/ms-playwright/chromium_headless_shell-1223/chrome-headless-shell-mac-arm64/chrome-headless-shell npx tsx spikes/ui-<name>.ts`.
- **Leftover test data:** published "Workflow test …" questionnaires stay in Hospital A, because published versions can't be deleted. Reset the local database if it gets cluttered.

## 5. Open items (as of 8 Oct 2026)

- **Next work:** plan **F1** ([plans/f1-test-foundation.md](plans/f1-test-foundation.md)).
- **Build order, undecided:** the roadmap order (F1 → F2 → F3 → F4 → C1 → C2), or F1 and F2, then C1 and C2 before the AWS work (F3, F4). The roadmap order stands until Rahul decides.
- **No domain chosen yet.** `<domain>` in the docs is a placeholder.
- **Research files:** `research_notes/` and `reports/` at the repo root are untracked. Either keep them (e.g. move them to `docs/research/` and fix the links in saas-requirements §0 and §13 and in this file) or delete them.
- **The proposed parts of A-20** (above, 7–8 Oct) need Rahul's yes when C1 and C2 start.
- **App branding** still says "Lifebox" in the UI (`apps/web/src/app/layout.tsx`, `h/[hospitalId]/layout.tsx`, `PatientHQ.tsx`). Rename it to Poised.
- **Unanswered:** should clinician-only Question Sets render full width in `ValidateSet.tsx`, rather than in an empty two-column grid?
- **Unconfirmed:** whether saved option lists should be copied (as built) or linked.
- **Unsaved test data:** the comprehensive test HQ exists only in Rahul's local database. Export it with the API, or rebuild it once plan F1's fixtures exist, before moving machines.
- **Content:** HJE Full HQ sets 5–11 and their disclosures still wait on a real Lifebox export (`tools/export-lifebox-hq.js`). Low priority (functionality over content).
- **Small issues:**
  - "Years since a date" opens with an empty list and no message.
  - The Medication, Admissions and date-list disclosure fixes have no test-plan cases.
- **Stale docs:**
  - README says there's no publish workflow, and its seed-user table lacks Cara Clinician and Alex's publisher role.
  - `pending.md` §4 lists Versions and publish as not started.
