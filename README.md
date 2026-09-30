# Lifebox Authoring on SurveyJS: demo slice

A thin end-to-end slice of the clinical questionnaire authoring tool:
- **Designer:** author a chapter in SurveyJS Creator.
- **Clinical outputs:** attach codes, notes, ASA grades and review flags to answers.
- **Preview:** try the chapter as a patient or as a clinician, and watch the outputs change live.

What and why are in [docs/requirements.md](docs/requirements.md) and [docs/plan.md](docs/plan.md).

## Setup

You need Docker, Go 1.26 or newer (older Go downloads 1.26 by itself) and Node 20.9 or newer.

```sh
docker compose up -d db     # Postgres 17 on localhost:5432
make migrate seed           # schema, then Lifebox codes, 2 hospitals and 3 users
make dev                    # Go API on :8080 and Next.js on :3000; Ctrl-C stops both
```

Open http://localhost:3000 and sign in as a seeded user:

| User | Hospital | Role |
| --- | --- | --- |
| Alex Author | Hospital A | author |
| Val Viewer | Hospital A | viewer |
| Bea Author | Hospital B | author |

- **Environment:** `make` creates `.env` from `.env.example` on first run. `NEXT_PUBLIC_SURVEYJS_KEY` is the SurveyJS licence key. It's empty by default, so the Creator shows the trial banner. Never commit a real key.
- **Other commands:**
  - `make test`: `go test` and Vitest.
  - `make lint`: `go vet` and `tsc`.
  - `make sqlc`: regenerates `apps/api/internal/db` after editing `db/queries`.
  - `make migrate-down`: rolls back the last migration.

## Demo script

1. Sign in as **Alex Author**.
2. Create **Pre-op assessment**, add the chapter **Heart and circulation**, then click **Design**.
3. Add **Yes / No** and set its title to "Do you smoke?". Click **Outputs (0)** and, on **Yes**:
   - add SNOMED **77176002 Smoker** (search "smok")
   - add the note "Current smoker" in the Lifestyle category
   - set the flag to amber
4. Add **Number** and set its title to "How many per day?":
   - Under Conditions, set "Make the question visible if" with the wand: Do you smoke? equals Yes.
   - Add an output with the note "Smokes {answer} per day" (Lifestyle).
5. Add **Statement** and tick **Clinician only**.
6. Click **Preview**. As a patient, answer Yes and 20: the panel shows both outputs and the amber flag. Switch to No and the outputs disappear. The statement isn't shown.
7. Switch to **Clinician**: the statement appears.
8. Sign out and sign in as **Bea Author** (Hospital B): the questionnaire isn't visible.

## Layout

```
apps/api            Go API: net/http, pgx, sqlc, goose (as go tools)
  cmd/api           server wiring
  cmd/seed          loads db/seed/lifebox/*.csv plus demo hospitals and users
  internal/httpapi  routes, auth and hospital-scope middleware, handlers
  internal/auth     dev login token (HS256 JWT in an httpOnly cookie)
  internal/chapter  chapter content rules
  internal/db       sqlc output (do not edit)
apps/web            Next.js + Mantine; /api/* is proxied to the Go API
  src/features      questionnaires, chapters, designer, outputs, preview
packages/clinical   shared TS: properties, stable IDs, viewer rules, computeOutputs
spikes              step 0 spike results (throwaway; see spikes/README.md)
```

## Deviations from the plan

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
- **Text questions are one line only (partly QT-04).** Long text isn't in the toolbox.
- **Sample patient variables:** the preview's sample patient sets `{patientName}`, `{patientAge}`, `{patientSex}` and `{viewer}`, but no conditions use them yet (LOG-03 is Release 1).
- **Test coverage:** tests are unit-level only, as agreed. The demo script was run end to end in a headless browser while building, but those scripts are not in the repo.
