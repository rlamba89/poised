# S01 verification

**Verifier session:** 8 Oct 2026. The working tree is on top of `223e69b`, with S01 uncommitted.

## Verdict: **Verified**

Everything in the brief is built. Every route in the three groups has its five cases, and the "Done when" checks pass when I rerun them myself. Both bug fixes are small and each has its own test. The docs are current. Nothing in the findings below blocks a commit.

## Findings, worst first

None of these block a commit.

1. **The advisory lock runs integration packages one at a time, and each rebuilds the template:** [apitest.go:84](../../../apps/api/internal/apitest/apitest.go#L84).
   - **Why it matters:** this is fine with 2 packages (about 5 s). Once S02+ adds packages, a package that is waiting for the lock still counts against `go test`'s 10-minute timeout.
   - **What to do:** revisit if the run gets slow. The build notes already say so. One option is to rebuild only when the goose version changes.
2. **`chapterDetails.validate()` trims through pointers that point at the loaded row:** [chapters.go:96](../../../apps/api/internal/httpapi/chapters.go#L96).
   - **Why it matters:** today it's correct and tested, but a future caller that only wants to check input would change `c` without meaning to.
   - The "Enter a name for the chapter." rule is also repeated in `addChapter`. Style only.
3. **Known and already tracked, not S01 regressions:**
   - `PATCH /episodes/{id}` with a null field still crashes. It's in the S02 brief, with a recover middleware.
   - `total` is 0 past the last page, so `?page=99999999999` returns `"page":107374182,"total":0`. It's in pending.md §3.
4. **Two doc slips, fixed by the verifier (docs only):**
   - decisions.md said leftover `t_…` databases are dropped. The real prefix is `sj_test_…`.
   - F1 Status said `c.LoginAs(user)`. The real signature is `c.LoginAs(t, user)`.

## Checks

| Check | Result |
| --- | --- |
| **Scope** | All 18 routes in the three groups are tested: questionnaires 6, chapters 6, option lists 4, codes and categories 2. F1's "must also cover" items are all asserted: retire on publish, published content refused, stable IDs on a new version, stale revision → 409 with nothing written, revision +1, and code search by prefix and by words. No app change except the two bug fixes. |
| **Tests first** | Each bug has a unit test (`TestPageNumber`, `TestChapterDetailsValidate`) plus an integration case. Every refusal also asserts that rows are unchanged. |
| **Tenant isolation** | **Mutation test:** I removed `hospital_id` from `GetChapterMeta` and `DeleteOptionList` in the generated code. The "another hospital" case of 5 tests failed (`TestGetChapter`, `TestUpdateChapter`, `TestDeleteChapter`, `TestSaveChapterContent`, `TestDeleteOptionList`). Then I restored the files from git. |
| **A-20 grep** | 3 `en-GB` hits in `apps/web`. They're older than S01 (commit `3be7c4a`), S01 changed no web code, and S05's brief tracks them. |
| **Org-name grep** | Prints nothing. |
| **Secrets** | Only the dev placeholder `TOKEN_SECRET` and the test-only `apitest-secret`. There's no real connection string; the local `sj:sj` one is the same as the existing `DATABASE_URL`. |
| **Patient data and logs** | S01 adds no logging and no patient-facing code. The `Episode` fixture uses a made-up patient. |
| **Broken migration** | A temporary `00005_verify_broken.sql` stopped every test with `db/migrations/00005_verify_broken.sql: ERROR: syntax error at or near ")"`. Removed, and the template rebuilt. |
| **`make test` without a database** | `go test -count=1 ./...` passes with both URLs pointing at a closed port. |
| **Docs** | F1 Status, plans README and sessions README are updated. decisions.md has the S01 entry. pending.md has the tester's findings. The S02 brief has the episode crash. |

## Browser spot-checks (Claude in Chrome, against `make dev`)

I created `S01 verify <time>` as Alex through in-page `fetch` (the real proxy and cookie), then deleted it.

| Check | Result |
| --- | --- |
| A stale-revision save | **409**, "Someone else changed this chapter…". |
| Bug fix: chapter PATCH with `{"name": null}` | **400**, "The request could not be read." The name is unchanged. |
| Bug fix: `?page=99999999999` | **200**, `items: []`. |
| Val Viewer | Reads a chapter (200). PATCH, content save and publish are **403** with plain messages. |
| Bea (Hospital B), API | A's URL → **404** "Hospital not found." B's URL with A's questionnaire or chapter id → **404** on GET, PATCH and DELETE. |
| Bea (Hospital B), UI | A's editor URL shows "You don't have access to this hospital." and no content. |
| After the refusals | The chapter is unchanged ("Verify set", revision 1, `q_vfy`). No console errors. |

No phone-size check: S01 changes no patient screen.

## Test results (8 Oct)

`make lint test test-integration`, exit 0:

```
go vet -tags integration ./...     (no findings)
tsc --noEmit                       (no errors)
make test:  ok auth, chapter, episode, httpapi, lifebox;  Test Files 13 passed (13), Tests 103 passed (103)
make test-integration:
ok  	github.com/rlamba89/poised/apps/api/internal/apitest	2.207s
ok  	github.com/rlamba89/poised/apps/api/internal/auth	1.491s
ok  	github.com/rlamba89/poised/apps/api/internal/chapter	0.386s
ok  	github.com/rlamba89/poised/apps/api/internal/episode	0.929s
ok  	github.com/rlamba89/poised/apps/api/internal/httpapi	4.813s
ok  	github.com/rlamba89/poised/apps/api/internal/lifebox	1.227s
```

`make e2e` doesn't exist yet (S03).

## Suggested commits

The same as the build notes, in this order:

1. **Add the integration test harness and make test-integration (F1 Step 1):** `apps/api/internal/apitest/`, `apps/api/internal/httpapi/main_integration_test.go`, `Makefile`, `.env.example`, `apps/api/go.mod`.
2. **Fix a 500 on a huge ?page= in the questionnaire list:** `apps/api/internal/httpapi/questionnaires.go`, `questionnaires_test.go`.
3. **Fix a crash when a chapter PATCH sends a null field:** `apps/api/internal/httpapi/chapters.go`, `chapters_test.go`.
4. **Integration tests for the authoring routes (F1 Step 2, part 1 of 2):** `apps/api/internal/httpapi/{cast,questionnaires,chapters,optionlists,codes}_integration_test.go`.
5. **Docs: S01 build notes, test plan, test report, verification and status:** `docs/sessions/S01-integration-harness/`, `docs/sessions/README.md`, `docs/sessions/S02-workflow-route-tests/brief.md`, `docs/plans/README.md`, `docs/plans/f1-test-foundation.md`, `docs/decisions.md`, `docs/pending.md`, `CLAUDE.md`.

`reports/` and `research_notes/` stay untracked (decisions §5).
