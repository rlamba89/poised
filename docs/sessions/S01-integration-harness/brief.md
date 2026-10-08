# S01: Integration test harness, and tests for the authoring routes

**Plan:** F1 ([f1-test-foundation.md](../../plans/f1-test-foundation.md)), Step 1, plus Step 2 for the authoring route groups. This is part 1 of 3 of F1.
**Depends on:** nothing. It's the first session.

**Ask Rahul first:** nothing. Check Docker is running (`docker compose up -d db`).

## Build

- **The `apps/api/internal/apitest` harness,** exactly as F1 Step 1 describes:
  - the `integration` build tag;
  - a template database made by goose;
  - a database per test;
  - `LoginAs`, `Do`, `DB` and the fixtures;
  - `make test-integration`, and `TEST_DATABASE_URL` in `.env.example`.
- **The first test:** `GET /api/health` → 200.
- **Integration tests with the five cases** (happy path, no sign-in, wrong role, other hospital's data, invalid input) for these route groups, including F1's "must also cover" items:
  - questionnaires, versions and publish;
  - chapters (Question Sets): a stale revision gives 409 with nothing written, and the revision goes up by 1 on save;
  - option lists, codes and categories: code search by code prefix and by words in the description.
- **If a test finds a bug:** record it in the F1 Status section and in `build-notes.md`. Fix it test-first, as its own suggested commit.

## Not in this session

- the episode, patient-link, clinician and notes routes (S02);
- Playwright (S03);
- any change to app behaviour, apart from bug fixes.

## Done when

- `make test-integration` passes.
- A deliberately broken migration makes `TestMain` fail with a clear message (try it, then revert).
- `make test` still runs without a database.

## Manual test focus

There's **no behaviour change**, so the test plan is a regression of authoring. Use the seeded users from the root README:
- create a questionnaire, then add a Question Set, page and question;
- save, reload, and check it's kept;
- option lists, and code search;
- publish, then "Create new version";
- Val Viewer can't edit;
- Bea Author (Hospital B) can't open Hospital A's questionnaire by URL.

The tester also runs `make test-integration` once in a terminal and reports the summary line.
