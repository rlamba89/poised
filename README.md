<img src="docs/brand/preopsis-logo.svg" alt="Preopsis" height="48">

# Preopsis

Preopsis is a pre-operative assessment platform for hospitals and their patients. The plans and requirements are in [docs/plans/](docs/plans/README.md) and [docs/saas-requirements.md](docs/saas-requirements.md).

**New here (person or Claude)?** Read [CLAUDE.md](CLAUDE.md), then [docs/decisions.md](docs/decisions.md). The decisions doc records every decision, preference and gotcha from the project's history.

What's built so far is a clinical questionnaire authoring tool and one pass of the episode workflow. Content is stored as SurveyJS JSON, and the editor is laid out like the Lifebox Author tool:
- **Editor:** a Structure tree of Question Sets and pages on the left, and the selected page's question cards on the right. Clicking a card opens its **Settings | Disclosures | Logic** panel in place of the tree. Every change saves automatically.
- **Disclosures (clinical outputs):** codes, a note with a category, an ASA grade and a review flag. They can sit on an answer, a grid cell or a score band.
- **Logic:** a builder with no code to type.
  - Rules joined by All / Any / None / Not all, with groups nested to any depth.
  - Tests for options, numbers, dates, text, the patient's age and sex, who is viewing, and score bands.
  - Used for show, required and read-only, for each option, for pages, for whole Question Sets (testing earlier sets), and for skip rules.
  - Cards say each condition in plain words, and the header flags conditions that can't work.
- **Question types:** Lifebox's, plus Rating, Grid, Calculation (score totals with bands, years since a date, BMI), File upload and Signature. A group can be made repeatable.
- **Settings per type:**
  - **Text:** format and maximum length.
  - **Number:** range, decimal places, unit and soft warnings.
  - **Date:** format, past or future only, and several dates.
  - **Select:** dropdown or searchable list, minimum and maximum ticks, Don't know, Prefer not to say, Other and Select all, plus options from the code library or a saved option list.
  - **All questions:** placeholders and required messages.
- **Translate:** English beside each language, gaps flagged, and CSV export and import.
- **Preview:**
  - **Patient view:** clinician content removed, and each Section on its own screen.
  - **Clinician view:** patient questions on the left, clinician items beside them as boxes, and each page's Clinical summary.
  - **Screens and languages:** phone, tablet and desktop widths, and any translated language.
  - **Outputs:** the side panel updates live as you answer.
  - **Test cases:** save the answers and the outputs they produce, then run them all again after changes.
- **Lifebox import:** brings in a questionnaire exported from Lifebox.

What and why:
- [docs/requirements.md](docs/requirements.md)
- [docs/plan.md](docs/plan.md): the first slice
- [docs/plan-redesign.md](docs/plan-redesign.md): the Lifebox-style redesign
- [docs/plan-features.md](docs/plan-features.md): the SurveyJS features added on top

What we copied from Lifebox is in [docs/lifebox-ui-notes.md](docs/lifebox-ui-notes.md).

## Setup

You need Docker, Go 1.26 or newer (older Go downloads 1.26 by itself) and Node 22.22 or newer (React Router 8 needs it).

```sh
docker compose up -d db     # Postgres 17 on localhost:5432
make migrate seed           # schema, then Lifebox codes, 2 trusts, 3 hospitals and 8 users
make dev                    # Go API on :8080 and the web app (Vite) on :3000; Ctrl-C stops both
```

Open http://localhost:3000 and sign in as a seeded user:

| User | Trust and hospital | Role |
| --- | --- | --- |
| Ada Admin | Trust A, whole trust | admin |
| Alex Author | Trust A, whole trust | super clinician |
| Cara Clinician | Trust A: Hospital A1 only | clinician |
| Sue Super | Trust A: Hospital A1 only | super clinician (can't change the trust's questionnaires: that takes a whole-trust role) |
| Val Clinician | Trust A: Hospital A2 only | clinician |
| Bea Author | Trust B, whole trust | super clinician |
| Max Multi | Trust A: Hospital A2 only; Trust B, whole trust | clinician in A; admin in B |
| Sam Superadmin | none (a platform admin) | — |

Roles are one ladder (A-5): clinician < super clinician (also authors and publishes) < admin. Anyone with more than one hospital picks one after signing in, and switches from the header.

- **Environment:** `make` creates `.env` from `.env.example` on first run. No SurveyJS licence is needed: survey-core and survey-react-ui are MIT, and the Creator is no longer used.
- **Other commands:**
  - `make lint`: `go vet` and `tsc`.
  - `make sqlc`: regenerates `apps/api/internal/db` after editing `db/queries`.
  - `make migrate-down`: rolls back the last migration.

## Tests

The rules for each kind are in [docs/plans/README.md](docs/plans/README.md#testing-rules-apply-to-every-plan).

| Command | What it covers | Needs |
| --- | --- | --- |
| `make test` | **Unit tests:** Go packages (`go test ./...`) and `packages/clinical` and the web app (Vitest). | Nothing running |
| `make test-integration` | **API → database:** real HTTP requests through the real Go router into a real migrated Postgres, checking the response and the rows. Every route has its happy path, no sign-in, wrong role, another hospital's data and invalid input. | Docker (it starts the compose database) |
| `make e2e` | **End-to-end journeys** in a real browser (Playwright + pytest, in [e2e/](e2e/README.md)): J1 author, J2 publish, J3 patient (on an iPhone 13), J4 clinician validates (with a stale second tab), J5 POA Summary. | `make e2e-setup` once (Python 3.12+), then `make dev` running |

`make e2e ARGS=--headed` shows the browsers as the journeys run. A failed journey keeps its Playwright trace; [e2e/README.md](e2e/README.md) explains how to read it.

## Demo script

1. Sign in as **Alex Author** and click **New Questionnaire**. Name it **Pre-op assessment** and save. The editor opens.
2. Click **Add Question Set**, rename it **Heart and circulation** with the pencil, and pick an icon. Then click **Add Page**.
3. Click **Yes / No** under "Add content". In Settings, set the question text to "Do you smoke?".
   - Open the **Disclosures** tab and click **Add** next to **Yes**.
   - Add SNOMED **77176002 Smoker** (search "smok"), the note "Current smoker" (Lifestyle), and an amber flag.
   - The card now shows a `🔗 1` chip next to Yes.
4. Add **Number** with the text "How many per day?".
   - In **Logic**, choose Conditionally → Do you smoke? → Is → Yes. The card gets an orange border and a "Displays when" box.
   - In Disclosures, add the note "Smokes {answer} per day" (Lifestyle).
5. Add **Statement** "Review smoking cessation support." and switch **Clinical** on. The card shows a purple CLINICIAN badge. Turn on **Clinical summary** at the top right of the page.
6. Click **Preview**.
   - **As a patient:** answer Yes and 20. The panel shows both outputs and the amber flag. Switch to No and they disappear. The statement isn't shown.
7. Switch to **Clinician**.
   - The statement sits in the right-hand column beside the questions.
   - Below it, the **Page 1 summary** box lists "Current smoker" and "Smokes 20 per day" above Clinical comments.
8. Sign out and sign in as **Bea Author** (Trust B): the questionnaire isn't visible.

### More features (plan-features.md)

1. Back in the editor, open **How many per day?** and go to **Settings**. Set Minimum 1, Maximum 100, Decimal places "Whole numbers", Unit "a day", and a soft warning above 60.
2. **Scores and bands:**
   - On **Do you smoke?**, switch on **Scores** in its options and give Yes 1 point.
   - Add a **Calculation**. Choose "A score total" and add up Do you smoke?.
   - Click **Add band** twice. That gives Low ≤ 2 and High above.
3. **Logic:**
   - Add a **Text** question. In **Logic**, choose Display → Conditionally.
   - Pick Do you smoke? is Yes, then **Add condition**: Patient's age ≥ 65.
   - Switch Match to **Any**.
   - The card reads "Displays when Do you smoke? **is** Yes **or** Patient's age **≥** 65".
4. Open the page's ⋯ menu → **Logic**, then **Add skip rule**: when Do you smoke? is No, End this Question Set.
5. Click **Translate** and choose German. Type a translation for "Do you smoke?". Missing ones stay flagged.
6. **Preview:**
   - Try **Phone** and **German**.
   - Answer the form, then **Save answers** as a test case. Change a disclosure in the editor and click **Run all** in the preview: the case fails and lists the difference.

## Importing a Lifebox questionnaire (HJE Full HQ)

1. Open the questionnaire in the Lifebox Author tool in Chrome, open DevTools (⌥⌘J), paste [tools/export-lifebox-hq.js](tools/export-lifebox-hq.js) and press Enter. It only reads. It downloads `lifebox-hq-<id>.json` with every Question Set, page, question, option, condition and disclosure.
2. Run `cd apps/api && go run ./cmd/import-lifebox -file ~/Downloads/lifebox-hq-<id>.json`. It creates a new draft in Trust A (for all its hospitals), then lists anything it couldn't carry over (for example, conditions on another Question Set, or codes missing from our library).

Without an export, `python3 tools/hq-from-notes.py docs/hje-full-hq-content.md > hq.json` builds Question Sets 1–4 of HJE Full HQ from the screenshot transcription. It has no disclosures, because the screenshots don't show them. Import it the same way.

## Layout

```
apps/api            Go API: net/http, pgx, sqlc, goose (as go tools)
  cmd/api           the local all-in-one server (make dev): staff + patient routes
  cmd/staff-api     staff routes only; cmd/patient-api: the patient's link only
                    (on AWS, separate Lambdas; locally `make staff-api` / `make patient-api`)
  internal/config   env settings, incl. the per-country values (A-20); GET /api/config
  cmd/seed          loads db/seed/lifebox/*.csv plus demo hospitals and users
  internal/httpapi  routers (staff, patient, both), auth and hospital-scope middleware, handlers
  internal/auth     dev login token (HS256 JWT in an httpOnly cookie)
  internal/chapter  chapter content rules
  internal/db       sqlc output (do not edit)
  cmd/import-lifebox  loads a Lifebox export (internal/lifebox converts it)
apps/web            React + Vite + React Router + Mantine; /api/* is proxied to the Go API
  src/routes.tsx    every URL and its page (src/pages); see apps/web/AGENTS.md
  src/features      questionnaires, editor (Lifebox layout), outputs, preview
packages/clinical   shared TS: editor model (doc.ts), conditions (logic.ts), per-type settings
                    (fields.ts), question types and functions, properties, stable IDs, viewer
                    rules and patient screens, clinical summary, computeOutputs, test cases,
                    translations
e2e                 end-to-end journeys J1–J5 (Playwright + pytest); see e2e/README.md
tools               Lifebox export snippet; HQ-from-notes converter
spikes              step 0 spike results (throwaway; see spikes/README.md)
```

## Deviations from the plan

These are from the first slice. The SurveyJS Creator rows no longer apply: the Creator was replaced by our own editor on 1 Oct 2026 (docs/plan-redesign.md).

| Plan | Built | Why |
| --- | --- | --- |
| `clinicalOutputs` is an array property set with `setPropertyValue` | In memory it's held in an immutable `{ items }` object with its own type. JSON is still a plain array. All code goes through `outputsOf` / `setOutputs`. | Spike 2: survey-core overwrites an existing array in place, which breaks redo. A `string`-typed property would also merge quick edits into one undo step. Agreed 30 Sep. |
| `showTitlesInExpressions` | `useElementTitles: true` | Deprecated and hidden in v3 (spike 2). |
| "None" item carries outputs | "None of these" is a normal option with `isExclusive: true`. The built-in None, Other, Don't know and Refuse options are hidden, on the canvas too. Select all stays. | SurveyJS saves the built-in special options without custom properties, so outputs on them would be lost. Agreed 30 Sep. |
| "Yes / No" (type not specified) | A Select One preset with Yes and No options | Each answer gets a stable ID, outputs and an editable label with no special case. Agreed 30 Sep. |
| IDs like `q_7f3k2p` | Letters only, e.g. `q_kfbwpx`, `o_tqmd` | The Creator numbers new options from digits in the last option's value, so digits made its placeholder read `o_u8l4` instead of "Item N". |
| `clinicianOnly` in a "Clinical" category listed first | First setting of the **General** tab | In v3 each category is a separate tab and the first tab opens by default. A one-checkbox Clinical tab first would hide the title and description. Outputs are edited in the modal. |
| `categories.id int` | `uuid`, keeping Lifebox's own category IDs | Keeps the Lifebox mapping for later import. |
| `codes.category_id` required | Nullable | 21 Lifebox codes have no category, and "Unassigned" is note-only. |
| 278 codes | 276 | That's how many rows `codes.csv` holds: one quoted field spans lines, so the line count is higher. |
| Seed with `\copy` | A Go seed command (`make seed`) | No `psql` needed on the machine; it's safe to rerun (upserts). |
| Steps 1 and 2 committed separately | One commit | Docker Desktop was hung while step 1 was built, so both were verified together. |
| — | Default titles "Question N" / "Group N" | SurveyJS numbers default names from existing names, which are IDs here, so every new question was "question1". |
| — | sqlc `emit_interface` | Handlers take the generated `db.Querier`, so the revision and scoping rules are unit-tested with a fake. |

## Known limits and data notes

- **Data problem to fix at source:** Lifebox ICD-10 `F17.1` reads "Mental,SYSTEM and behavioural disorders…". The seed collapses the line break but doesn't edit Lifebox's text.
- **Not built yet:** disclosures on "Other (please specify)".
- **Saved option lists are copies.** Using a list copies its options into the question, with new IDs. Changing or deleting the list later doesn't change questions that already used it.
- **Question Set conditions in the preview:** a one-set preview can't see earlier sets' answers. It decides a condition only when it tests just the patient, and otherwise explains it above the form.
- **Publishing:** one-click Publish (publisher role) is blocked by logic problems, failing test cases and unsaved changes. There's no review or sign-off step yet (SGN).
- **Skip rules:** a skip rule stays on the page whose Logic panel it was added on, and may test that page and earlier ones. Rules saved before 1 Oct (without a page) are listed under the latest page they test.
- **Test coverage:** see [Tests](#tests). The journeys cover the main path through each screen, not every feature; [docs/manual-test-plan.md](docs/manual-test-plan.md) marks which manual cases they automate.
