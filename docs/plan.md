# Lifebox Authoring on SurveyJS: Demo Slice Plan

Sep 30, 2026 · @Rahul Lamba

This plan turns *Lifebox Authoring on SurveyJS — Requirements* ([requirements.md](requirements.md)) into a build. **Implementers: read section 12 (handover) first.** It covers a **thin demo slice** first: a small end-to-end proof of the riskiest ideas. The full Release 1 plan comes after the slice has proved them.

Guiding rule: **the simplest thing that meets the requirement.** Before anything is added, ask whether SurveyJS already does it, or whether a plainer way exists.

---

## 1. What the demo slice proves

An author can:

1. **Author a chapter in the designer.** Sign in as a seeded user, create a questionnaire, add chapters, and build a chapter in SurveyJS Creator with clinical toolbox names, autosave, undo and redo, a clinician-only flag and stable IDs.
2. **Attach clinical outputs to answers.** Put several outputs (note + category + codes + ASA + flag) on answer options and on text, number and date questions, picking codes from a small seeded SNOMED CT / ICD-10 library.
3. **Preview with live outputs.** Preview the chapter as a patient or as a clinician, with a sample patient, and see the codes, notes, ASA and flags produced as they answer.

**Not in the slice:** publish, versions beyond the first draft, review and sign-off, translations, clinical panels, reports, export and import, audit, real SSO, hosting. The data model leaves room for them (section 5).

### Requirements the slice touches

| Area | IDs (fully or partly) |
| --- | --- |
| Form library | FRM-01 (list), FRM-04, FRM-06 (drafts), FRM-07 |
| Lifecycle | LCY-03, LCY-04, LCY-05, LCY-07 (IDs generated and shown; publish lock comes later) |
| Structure | STR-01, STR-02, STR-06, STR-09 (clinician-only + output count badges) |
| Questions | QST-01, QST-02, QST-03, QST-04, QT-01, QT-02, QT-03, QT-04, QT-05 (full date), QT-06, QT-08 |
| Logic | LOG-01, LOG-04, LOG-05, LOG-09 (SurveyJS Logic tab, built in) |
| Clinical | CLN-01, CLN-02, CLN-05, CLN-06, CLN-07 (`{answer}` only), CLN-08, CLN-09 (search only), CLN-12 (question-level badge variant), CLN-14, CLN-15 (single engine) |
| Preview | PRV-01, PRV-03, PRV-04, VEW-01 (without short screens), VEW-02 |
| Designer | DSG-01 (subset of toolbox), DSG-04 |

---

## 2. Decisions taken

| Topic | Decision | Why |
| --- | --- | --- |
| Repo | Monorepo: npm workspaces (TS) + one Go module + root Makefile | One command for dev and test, no build orchestrator |
| Frontend | Next.js (App Router), TypeScript, Mantine UI | Team knows Next.js. Mantine covers tables, forms and modals without Tailwind. |
| SurveyJS | **v3.1.2, exact pin** (`survey-core`, `survey-react-ui`, `survey-creator-core`, `survey-creator-react`) | Current line, which avoids a v2→v3 migration soon. Upgrades are deliberate (NFR-14). |
| Backend | Go, standard library `net/http` | Team knows Go. Go 1.22+ routing is enough, no framework. |
| API | REST + JSON | Payloads are mostly "one chapter JSON document" |
| Database | PostgreSQL. Tables for structure, JSONB for chapter content. `pg_trgm` for code search. | One store. Search and transactions built in. |
| DB tooling | `sqlc` (typed Go from plain SQL) + `goose` (plain SQL migrations) | No ORM. Every query is readable SQL. |
| Content storage | **One SurveyJS JSON per chapter** | Chapter icon and audience stay in tables. Edit conflicts are per chapter (LCY-04). |
| Clinical evaluation | A shared TS package (`packages/clinical`) runs in the browser for preview | Guarantees one engine (CLN-15). No backend TS for now. |
| Server-side replay | **None. The frontend is trusted.** All answer replay (preview, test cases, Lifebox parity scenarios) runs in the browser using `packages/clinical`. Go stores the results and enforces rules on them. No Node service, no TS on the backend. | No Go reimplementation of SurveyJS exists, and a port would quietly disagree with preview (see 2.1). Trusting the author's browser is acceptable for an internal, signed-in tool. |
| Output editing UI | "Outputs (n)" badge on each **question**, opening a modal that lists the question and each option with their outputs | Uses the documented adorner API. A per-option badge would need an internal SurveyJS override. |
| Auth | Stub login: pick a seeded user. The API issues a signed token with the same shape real SSO will provide. | Hospital scoping and roles work from day one. Cognito or another IdP is swapped in later. |
| Codes | Lifebox's own seed data: 278 SNOMED CT / ICD-10 codes + the 13 categories, from CSVs in `apps/api/db/seed/lifebox/` (see 12.3) | Real, already-curated codes. No licence or download needed. Never invent codes. |
| Local dev | Postgres in Docker. Go API and Next.js run natively. | Fastest edit loop |
| Hosting | Local only for now | Decide later |
| Licence | Build on SurveyJS trial. The key goes in config later and is never committed. | Trial banner is acceptable for the demo |
| Tests | Unit tests only: Vitest for `packages/clinical`, `go test` for Go logic | Chosen for speed. Integration and E2E come with Release 1. |

### 2.1 Why the backend is not all Go (for the record)

- **No non-JS evaluator exists.** SurveyJS logic (conditions, calculated values, repeating groups, cascading visibility) is evaluated only by `survey-core`, which is JavaScript. No Go, .NET, Java or Python implementation exists.
- **A port is large and drifts silently.** The expression engine alone is about 200 KB of TypeScript, with subtle rules: `"1" = 1` is true, `'' empty` is true, array comparison ignores order. SurveyJS releases weekly. A Go copy would drift, and preview and server would give different clinical outputs with no error.
- **Most server checks don't need evaluation.** Missing or forward condition references, option counts, notes without a category, retired codes and changed IDs are all plain JSON reading, so they stay in Go.
- **Answer replay runs in the browser (decided).** Test cases and Lifebox parity scenarios run there too. Three things follow:
  - **Stale results:** each test result is saved with the chapter `revision` it ran against. Publish accepts only passing results for the latest revision of every chapter.
  - **Hazard log:** "test cases run client-side and are trusted by the server" is recorded in the DCB0129 hazard log for the clinical safety officer to accept.
  - **Downstream integration:** any future system computing outputs from real patient answers must either take them from the patient's browser (saved with the answers) or run `survey-core` itself. Go cannot derive them from the JSON alone.
- **Possible reuse:** Lifebox's private `surveyjs` repo (Go, 2022) reads coding metadata from SurveyJS answers. It might help with Go-side JSON reading later.

---

## 3. Repository layout

```
poised/
├── Makefile                 # dev, test, lint, db-up, migrate, seed, sqlc
├── docker-compose.yml       # postgres only
├── package.json             # npm workspaces: apps/web, packages/*
├── .env.example             # DB URL, token secret, SurveyJS key (empty)
├── docs/
│   └── plan.md
├── apps/
│   ├── web/                 # Next.js frontend (TypeScript)
│   │   ├── next.config.ts   # rewrites /api/* → Go API (same origin, no CORS)
│   │   └── src/
│   │       ├── app/                         # routes only, thin
│   │       │   ├── login/
│   │       │   ├── h/[hospitalId]/questionnaires/
│   │       │   ├── h/[hospitalId]/questionnaires/[qid]/
│   │       │   ├── h/[hospitalId]/chapters/[cid]/design/
│   │       │   └── h/[hospitalId]/chapters/[cid]/preview/
│   │       ├── features/                    # one folder per feature
│   │       │   ├── questionnaires/          # list, create, delete
│   │       │   ├── chapters/                # list, reorder, icon, audience
│   │       │   ├── designer/                # Creator setup, toolbox, adorners, autosave
│   │       │   ├── outputs/                 # outputs modal, code picker
│   │       │   └── preview/                 # patient/clinician switch, sample patient, outputs panel
│   │       └── lib/                         # api client (fetch wrapper), auth context
│   └── api/                 # Go backend (own go.mod)
│       ├── cmd/api/main.go  # wiring: config, db pool, router, listen
│       ├── internal/
│       │   ├── httpapi/     # router, middleware (auth, hospital scope), handlers, JSON errors
│       │   ├── auth/        # dev token issue and verify; later OIDC/JWT verify
│       │   ├── chapter/     # content save rules (revision check, basic JSON shape)
│       │   └── db/          # sqlc-generated code (do not edit)
│       └── db/
│           ├── migrations/  # goose .sql files
│           ├── queries/     # sqlc .sql files
│           ├── seed/        # hospitals, users, roles, categories, codes
│           └── sqlc.yaml
└── packages/
    └── clinical/            # shared TS, no React, no browser APIs
        └── src/
            ├── types.ts         # ClinicalOutput, Code, NoteTemplate…
            ├── properties.ts    # registerClinicalProperties(): SurveyJS custom properties
            ├── ids.ts           # stable ID generation
            ├── viewer.ts        # stripClinicianOnly(json), viewer variables
            ├── outputs.ts       # computeOutputs(model) → ProducedOutput[]
            └── *.test.ts
```

**Separation rules**
- `apps/web` talks to `apps/api` only over HTTP (`/api/*`). It never touches the database.
- `packages/clinical` is plain TS with only `survey-core` as a dependency (no React), so preview, test-case runs and parity runs all share one engine.
- Go handlers call sqlc queries directly. A package like `chapter/` exists only where there are real rules. Don't add service or repository layers until they're needed.
- The API contract is the JSON documented in section 6, typed by hand on both sides (Go structs, TS types). It's small enough that codegen isn't worth it yet.

---

## 4. How the pieces fit

```
Browser (Next.js)                                   Go API                 Postgres
┌────────────────────────────────────────┐   /api/*  ┌──────────────┐      ┌─────────┐
│ Designer: SurveyJS Creator             │──────────▶│ auth, scope  │─────▶│ tables  │
│   + packages/clinical (properties, IDs)│  (Next    │ handlers     │ sqlc │ + JSONB │
│   + Outputs modal, code picker         │  rewrite) │ revision chk │      │ pg_trgm │
│ Preview: survey-react-ui               │           └──────────────┘      └─────────┘
│   + packages/clinical computeOutputs() │
└────────────────────────────────────────┘
```

Test cases (later) follow the same pattern as preview: the browser runs them with `packages/clinical` and posts the results to Go.

---

## 5. Data model (slice)

Only what the slice needs. Tables are named so later features slot in without restructuring.

```sql
hospitals        (id uuid pk, name text)
users            (id uuid pk, name text, email text unique)
memberships      (user_id, hospital_id, role text  -- viewer|author|reviewer|publisher|hospital_admin
                  , pk(user_id, hospital_id, role))

questionnaires   (id uuid pk, hospital_id fk, name text, description text,
                  created_by fk, created_at)
questionnaire_versions
                 (id uuid pk, questionnaire_id fk, version_no int, status text  -- 'draft' only in slice
                  , updated_by fk, updated_at, unique(questionnaire_id, version_no))
chapters         (id uuid pk, version_id fk, position int, name text, description text,
                  icon text, audience text  -- patient|clinician|clinician_document
                  , content jsonb           -- the SurveyJS JSON
                  , revision int            -- +1 on every content save (LCY-04)
                  , updated_by fk, updated_at)

categories       (id int pk, name text unique, note_only bool)  -- 13 Lifebox categories + "Unassigned" (note_only)
codes            (id uuid pk, code_set text  -- 'SNOMED'|'ICD10'
                  , code text, description text, full_name text, category_id fk,
                  billable bool, status text  -- active|retired
                  , unique(code_set, code))
                 -- GIN pg_trgm indexes on code and description
```

- **Unique draft name (FRM-04):** enforced in the handler inside a transaction (look for another questionnaire in the hospital whose version is a draft with that name). This is fine in the slice because there is only one draft version per questionnaire.
- **Versions table from day one.** Chapters belong to a version, not a questionnaire. Adding versions later would otherwise mean restructuring.

---

## 6. Clinical metadata inside the SurveyJS JSON

The chapter JSON is self-contained (OUT-01): anything that reads it can compute the same outputs.

**Stable IDs (QST-04, OPT-02, OUT-02)**
- A question's SurveyJS `name` *is* its stable ID. It is generated on add or copy, e.g. `q_7f3k2p`. The author edits `title`, never `name`.
- A choice's `value` *is* the option's stable ID, e.g. `o_a1b2`, unique within its question. The author edits `text`.
- Answers are therefore stored by ID, not label, natively.
- The Creator is set to show question titles, not names, in the Logic tab and in expressions (`useElementTitles: true`; `showTitlesInExpressions` is deprecated in v3), so authors never see raw IDs in conditions.
- The JSON editor tab is **off** because it would bypass the ID guards.

**Custom properties**, registered by `packages/clinical/properties.ts` and all non-translatable (CLN-08):

| On | Property | Type |
| --- | --- | --- |
| question, panel | `clinicianOnly` | boolean, shown in a "Clinical" category listed first in the settings panel |
| question (no-option types) | `clinicalOutputs` | `ClinicalOutput[]`, hidden from the settings panel and edited through the Outputs modal |
| itemvalue (options, incl. exclusive "None of these") | `clinicalOutputs` | same |

```jsonc
// ClinicalOutput
{
  "id": "out_x81k",
  "codes": [ { "set": "SNOMED", "code": "77176002", "display": "Smoker" } ],
  "note":  { "text": "Smokes {answer} per day", "category": "Respiratory" },
  "asa":   { "grade": "III", "emergency": false },   // optional
  "flag":  "amber"                                    // optional: amber | red
}
```

- **Validation rule (CLN-06):** at least one code or a note, and a note needs a category. The modal enforces it now; Go enforces it at publish time later.
- **Denormalised copies:** code `display` and category name are stored in the JSON so the form stands alone. The code library remains the source when picking.

**Decisions from the step 0 spikes (30 Sep 2026)** (details in `spikes/README.md`):
- **`clinicalOutputs` is held in a wrapper.** In memory, the value is an immutable `{ items: ClinicalOutput[] }` object with the custom type `clinicaloutputs`; in JSON, it is a plain array (`onSerializeValue` / `onSetValue`). A bare array breaks redo: survey-core overwrites an existing array in place. A `string`-typed property would also merge rapid edits into one undo step. All code reads and writes outputs through `outputsOf(obj)` / `setOutputs(obj, items)`.
- **"None of these" is a normal choice with `isExclusive: true`**, not the built-in `showNoneItem`. The built-in None item saves only its label, so it can't carry outputs. The exclusive choice gets a stable ID like any other option. The built-in None setting is hidden in the settings panel.
- **"Yes / No" is a Select One preset** (a radiogroup with Yes and No options), not the `boolean` question. Each answer then has a stable ID, outputs and an editable label, and the outputs modal and `computeOutputs` need no special case.
- **"Don't know" in one click (OPT-04)** is not in the slice. Authors can already add such an option and mark it exclusive.

**`{answer}` in notes (CLN-07, slice subset)** becomes:
- text → the typed text
- number → the number
- date → `dd/MM/yyyy`
- choice → the chosen option label(s), joined with ", "

Other answers and repeating-group rows come later.

**`computeOutputs(model)`** in `packages/clinical/outputs.ts` works like this:
1. Walk `model.getAllQuestions(true)`, which respects hidden panels and pages. `isVisible` alone does not, as SurveyJS docs and testing confirmed. This gives LOG-09 and CLN-14.
2. For each answered question:
   - choice types: collect the outputs of the selected option(s), including "None of these"
   - others: collect the question's own outputs
3. Render note templates.
4. Return `{ questionId, questionTitle, answerId?, answerLabel, output }[]` plus the suggested ASA (the highest). This supports CLN-17 later at no extra cost.

**Who is viewing (VEW-01/02)**
- Patient view: `stripClinicianOnly(json)` removes clinician-only elements *before* the survey is built, so they never reach a patient's browser. The model also gets the variable `{viewer} = 'patient'`.
- Clinician view: the full JSON and `{viewer} = 'clinician'`.
- The `{viewer}` variable is what makes LOG-03's "who is viewing" condition possible later.

---

## 7. API (slice)

Every hospital route checks that the caller has a membership in `{hid}`, in middleware (NFR-04). Errors are JSON `{ "error": "<plain-language message>" }` (DSG-04).

| Method & path | Purpose |
| --- | --- |
| `GET  /api/dev/users` | Seeded users for the stub login page |
| `POST /api/dev/login` `{userId}` | Sets an httpOnly signed-token cookie |
| `POST /api/logout` | Clears the cookie |
| `GET  /api/me` | User + memberships (hospitals and roles) |
| `GET  /api/h/{hid}/questionnaires?q=&page=` | List (FRM-01) |
| `POST /api/h/{hid}/questionnaires` `{name, description}` | Create questionnaire + draft v1 (FRM-04) |
| `DELETE /api/h/{hid}/questionnaires/{qid}` | Delete draft, creator or hospital admin only (FRM-06) |
| `GET  /api/h/{hid}/questionnaires/{qid}` | Questionnaire + draft version + chapter list (no content) |
| `POST /api/h/{hid}/questionnaires/{qid}/chapters` | Add chapter |
| `PATCH /api/h/{hid}/chapters/{cid}` | Rename, describe, icon, audience |
| `PUT  /api/h/{hid}/questionnaires/{qid}/chapter-order` `[cid…]` | Reorder |
| `DELETE /api/h/{hid}/chapters/{cid}` | Delete chapter |
| `GET  /api/h/{hid}/chapters/{cid}` | Metadata + `content` + `revision` |
| `PUT  /api/h/{hid}/chapters/{cid}/content` `{content, revision}` | Save. `409` if `revision` is stale (LCY-04). Returns the new `revision`. |
| `GET  /api/codes?set=SNOMED&q=smok` | Code search, top 20, active only (CLN-09) |
| `GET  /api/categories` | Categories for note and code pickers |

**Stub token.** An HMAC-signed JWT with `sub` and `name` (golang-jwt), stored in a cookie. Later, the same middleware also accepts a real IdP bearer token. Hospitals and roles always come from our `memberships` table, never from the token.

---

## 8. Build steps

Each step ends with something runnable.

**Step 0: Spikes (½–1 day).** These prove the four risky assumptions before real building; each is a throwaway page:
1. SurveyJS Creator v3 renders in Next.js via `dynamic(..., { ssr: false })`.
2. Changing `clinicalOutputs` from our React modal (`setPropertyValue`) marks the Creator as modified, triggers autosave, and can be undone and redone.
3. `onQuestionAdded` / `onItemValueAdded` can set generated `name` / `value` without clobbering the default label, including on copy (STR-07).
4. `getAllQuestions(true)` excludes questions in hidden panels.

If any spike fails, stop and revisit the design before building on it.

**Step 1: Skeleton.**
- Makefile, docker-compose (Postgres), npm workspaces.
- Go module with a health endpoint.
- goose + sqlc wired.
- Next.js + Mantine app with the `/api` rewrite.
- `make dev` runs everything.

**Step 2: Auth stub and questionnaires.** Migrations and seed (2 hospitals, 3 users, categories, codes). Login page, hospital switcher, questionnaire list, create and delete.

**Step 3: Chapters.** Questionnaire page with chapters: add, rename, description, icon picker (Tabler icons, stored by name), audience, drag-reorder, delete with confirmation.

**Step 4: Designer.**
- Creator page with toolbox renamed and ordered: Group, Yes / No, Select One, Select Many, Text, Date, Number, Statement.
- Only the Designer and Logic tabs are shown.
- New questions required by default.
- Autosave with saving and saved state, handling `409` with a clear "someone else changed this chapter" message.
- Stable IDs, and the `clinicianOnly` property.
- A "Clinician only" badge on the canvas.

**Step 5: Clinical outputs.**
- `packages/clinical` types and properties.
- `GET /codes` with pg_trgm.
- An "Outputs (n)" adorner on each question opens a modal listing the question and each option (including "None of these").
- Per row, authors add, edit and delete outputs: codes picked by search within a code set, a note with a category, ASA, flag.
- The note editor shows a worked example of `{answer}`.

**Step 6: Preview.**
- Preview page using `survey-react-ui`.
- Patient / Clinician switch.
- Sample patient form (name, age, sex). These become survey variables.
- A side panel shows `computeOutputs` live on every value change: codes, notes grouped by category, suggested ASA, flags.

**Step 7: Tests and tidy-up.**
- Vitest for `computeOutputs`: hidden question, hidden panel, clinician-only, none item, multi-select, date formatting, several outputs.
- Vitest for `stripClinicianOnly` and ID generation.
- `go test` for token verification, revision conflict rule and scoping middleware.
- README with a 3-command setup.

**Demo script (acceptance for the slice):**
1. Log in as Author at Hospital A.
2. Create "Pre-op assessment", add chapter "Heart and circulation".
3. Add "Do you smoke?" (Yes/No). Yes → SNOMED 77176002 "Smoker" + note "Current smoker" (Lifestyle), flag amber.
4. Add "How many per day?" (Number), shown when Yes, with note "Smokes {answer} per day".
5. Add a clinician-only statement.
6. Preview as patient: answer Yes and 20. The panel shows both outputs. Switch to No: the outputs disappear. The clinician-only item is not shown.
7. Preview as clinician: the clinician-only item appears.
8. Log in as Hospital B: the questionnaire is not visible.

---

## 9. After the slice: route to Release 1 (all Musts)

This is the order to confirm once the slice is accepted. It will be detailed in a follow-up plan.

1. **Versions and publish.** Statuses, read-only non-drafts, create new version, ID lock (LCY-01/02/06/07). Go static publish checks (SGN-04 minus test cases). Publish confirmation.
2. **Sign-off workflow.** Submit, approve or return, self-approval block (SGN-01/02). Roles enforced server-side.
3. **Structure and logic completeness.** Patient short screens (STR-03, open question 1). Copy keeps outputs and remaps conditions (STR-07). Delete warnings with dependants (STR-08). Plain-word conditions (LOG-06). Forward and cycle detection (LOG-08). Clinical functions (CAL-03). Viewer, age and sex conditions (LOG-03).
4. **Remaining question types and clinical panels** (QT-01 labels, QT-05 formats, QT-09 clinical summary, PNL-01–04, OUT-05).
5. **Outputs completeness** (CLN-07 full templates, CLN-12 as agreed).
6. **Translations** (LNG-01/02/04) using the Creator's Translation tab, and **preview devices and languages** (PRV-02).
7. **Code library screens** (COD-01–06) and **reports** (RPT-01).
8. **Export/import and Lifebox migration** (EXP-01/02/04, NFR-02). This is the biggest unknown: it needs the Lifebox export format and a list of in-scope questionnaires. The same-answers-same-outputs proof runs as a browser page in the tool.
9. **Audit** (AUD-01), **real SSO** (NFR-04), hosting, performance checks (NFR-08).

---

## 10. Further decisions (confirmed 30 Sep 2026)

1. The SurveyJS `name` / `value` *are* the stable IDs, as generated short IDs (not readable names like `smoker`). Option IDs are unique only within their question.
2. Clinician-only content is **removed** from the JSON for patient view, not just hidden with a condition.
3. We use **our own preview page**, not the Creator's built-in Preview tab, because we need the switch, the sample patient and the outputs panel. The Creator's Preview, Translation, Theme and JSON tabs are off in the slice.
4. Next.js proxies `/api/*` to Go, so the browser sees one origin: no CORS, and the auth cookie just works.
5. Chapter icons come from the Tabler icon set, stored by name.
6. Tool versions: Go, Node LTS, Next.js and Mantine at their current stable versions when the repo is scaffolded, pinned in the lockfiles.
7. Delete is a hard delete for drafts in the slice. Audit comes later.

## 11. Open questions from the spec that the slice does not block

These are needed before the Release 1 plan: patient screens rule (STR-03), chapter navigation, whether ASA is wanted, terminology source, master template rules, migration scope, clinician-only options, downstream consumers, review flag levels, admin viewer. The slice is built so that none of them force a rewrite.

---

## 12. Handover for the implementing session

### 12.1 Working rules
- Build the steps in section 8 in order. Commit after each step. Stop and report if a step 0 spike fails.
- Simplicity first. Don't add layers, libraries or abstractions the plan doesn't call for. If something seems to need one, ask.
- Never modify SurveyJS. Use only its public extension APIs (12.2).
- Pin SurveyJS packages to an **exact** version (`3.1.2`, no `^`). All four packages must be the same version.
- The Go backend never evaluates SurveyJS expressions. All answer replay runs in the browser through `packages/clinical` (section 2.1).
- Never invent clinical codes. Use only the seed CSVs (12.3).
- Tests are unit-level only for now (section 8, step 7).

### 12.2 SurveyJS v3 APIs to use (checked against the docs on 30 Sep 2026)

| Need | API |
| --- | --- |
| Next.js | Designer and survey components are client-only: `'use client'` + `next/dynamic` with `ssr: false` |
| Custom properties | `Serializer.addProperty("question" \| "panel" \| "itemvalue", { name, type, category, categoryIndex, visible, isLocalizable: false })`. A low `categoryIndex` puts the "Clinical" category first. |
| Hide or lock settings | `creator.onPropertyShowing` (`options.show`), `creator.onPropertyGetReadOnly` (works for itemvalue via `parentProperty`) |
| Buttons and badges on canvas elements | `creator.onElementGetActions`: push an action with a dynamic `title` and an `action` that opens our React modal. `onElementAllowOperations` hides built-in ones. |
| Generated IDs | `creator.onQuestionAdded` (`options.reason` tells toolbox-add apart from copy), `creator.onPanelAdded`, `creator.onItemValueAdded` |
| Autosave | `creator.autoSaveEnabled`, `creator.autoSaveDelay`, `creator.saveSurveyFunc = (saveNo, callback) => …`, `creator.state` ("", "modified", "saving", "saved"), `creator.onModified` |
| Undo and redo | Built in (`creator.undo()` / `redo()`) |
| Tabs | `showJSONEditorTab: false`, `showPreviewTab: false`, `showTranslationTab: false`, `showThemeTab: false`, `showLogicTab: true` |
| Toolbox | `creator.toolbox`: `getItemByName(name).title`, `removeItem`, `defineCategories` / item order |
| Custom functions (later) | `registerFunction({ name, func })` (v3 name; older docs say `FunctionFactory.Instance.register`) |
| Visible questions | `model.getAllQuestions(true)`. **Not** `question.isVisible`, which stays true inside a hidden panel. |
| Variables | `model.setVariable("viewer", "patient")`, referenced in conditions as `{viewer}` |

These **v1 names no longer exist**, so don't use them: `onDefineElementMenuItems`, `onGetQuestionTitleActions`, `onPropertyValueChanging`, `onPropertyEditorUpdateTemplate`, `isAutoSave`, `onElementCopied`.

Known traps:
- **Per-option badges:** there is no public API for a badge on each option. Don't override internal components such as `svc-item-value`; use the question-level badge and modal (section 2).
- **Titles in the Logic tab:** `showTitlesInExpressions` is deprecated and hidden in v3. Use `useElementTitles: true` (checked in spike 2).
- **Hidden values stay in the data:** values of hidden questions remain in `survey.data`. `computeOutputs` must walk only visible questions, not `survey.data`.
- **Composite questions** (later, clinical panels) store one nested object, and authors can't edit their inner questions.
- **The UI Preset Editor needs a PRO licence.** Don't use it; use `onPropertyShowing`.

Docs:
- Designer: https://surveyjs.io/survey-creator/documentation
- Form library: https://surveyjs.io/form-library/documentation
- Breaking changes: https://surveyjs.io/stay-updated/breaking-changes

### 12.3 Seed data
`apps/api/db/seed/lifebox/` holds Lifebox's own reference data as CSV:

| File | Contents |
| --- | --- |
| `code_categories.csv` | `id,name`: the 13 categories |
| `code_sets.csv` | `id,name`: `SNOMED`, `ICD10` |
| `codes.csv` | `id,code,code_set_id,fsn,synonym,code_category_id,billable,created_by`: 278 codes |

- **Load:** load the CSVs with a seed step (`make seed`, e.g. Postgres `\copy` into staging tables, then insert).
- **Field mapping:** `synonym` → `description`, `fsn` → `full_name`, `code_set_id` → `code_set` name. All codes are `active`.
- **"Unassigned":** add it as a `note_only` category, as Lifebox does.
- **Demo data:** also seed 2 hospitals (Hospital A, Hospital B) and 3 users (Author at A, Viewer at A, Author at B).

### 12.4 Definition of done for the slice
- All of section 8's demo script works on a fresh clone with: `docker compose up -d db`, `make migrate seed`, `make dev`.
- `make test` passes (Vitest + go test).
- The README has setup steps, and any deviation from this plan is listed with a reason.
