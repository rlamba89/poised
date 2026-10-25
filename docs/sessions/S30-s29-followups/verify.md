# S30 verification

## Verification 1 (25 Oct 2026, automatic run)

**Verdict: Verified.**

All three items in the brief are built, each with tests written first. Nothing outside the brief was added. There's no Go change, no new route and no migration.

### What was checked
- **`apps/web/src/lib/api.ts:34`:** any 5xx whose body has no API `error` now gives `NETWORK_ERROR`, and the API's own `error` still wins, a 500's included. A 4xx without `error` still gives "Something went wrong. Please try again.".
- **`apps/web/src/features/patient/prefill.ts:184` (`errorBannerShown`) and `PatientHQ.tsx:277`:** over the cleared card, only `NETWORK_ERROR` is hidden, and any other error shows.
- **`e2e/journeys/test_j8_reminders.py`:** it counts only texts that carry a link one of this episode's emails carries. The number range isn't widened, as the brief asks. J8 passed again on this long-lived database, which already holds many earlier J8 runs.
- **A-20 grep:** nothing new (only comments that were there before, and test files).
- **repocheck:** passes.
- **Security:** no secrets, and no patient data in messages. Patient and staff routing and the stripping of clinician-only content are unchanged.
- **Docs:**
  - pending.md: the three S29 items are struck through, and the tester's wording note is added.
  - sessions/README.md: the S30 row.
  - decisions.md: §1 has Rahul's two preferences (25 Oct), and §5 is updated.
  - AGENTS.md and the e2e README are updated.

### Findings (none blocking, worst first)
1. **The cleared card's wording under a refusal that won't go away** (`PatientHQ.tsx:281`):
   - When Reload is refused because the episode was cancelled, the banner says so, but the card below still says "Please try again" and offers **Reload**, which can't succeed.
   - The new J16b test treats this state as expected.
   - It's already in pending.md ("From S30's test report 1"), for Rahul.
2. **The Go API's own short-lived 5xx still shows a banner over the cleared card** (`prefill.ts:185`):
   - Two examples: Go's `serverError` 500 "Something went wrong. Please try again.", and the 503 "The database is not reachable.".
   - Both carry an `error`, so they count as API refusals. That is what the brief says ("Hide only `NETWORK_ERROR`").
   - The check compares the message text. A `network` flag on `ApiError` would be sturdier if the wording ever changes. It's a tidy-up, not a bug.
3. **A server-side outage can show two different messages** (`api.ts:34`):
   - Gateway JSON says "Check your connection".
   - Go's own 500 says "Something went wrong".
   - Both are as decided (S29: Rahul's wording, 25 Oct; S30's brief). Noted only.
4. **Tidy-ups in the e2e code:**
   - `reload_fails` is copied in two J16b tests (`test_j16b_prefill_cross_trust.py:213` and `:246`).
   - `NETWORK_ERROR` is defined again in J16b and J5 rather than once in `e2e/support.py`.

### Spot checks (Claude in Chrome, trk profile, as Cara on a POA Summary)
- **Save comment** with each simulated answer:
  - the gateway's JSON 502 gives "We couldn't reach the server…";
  - a 409 with `error` gives its own message;
  - a 500 with `error` gives its own message;
  - a 504 with an HTML body gives the network message.
- "Something went wrong" never showed.
- Moving to **Episodes** in the app under a JSON 503 shows the network message on the list page.
- The cleared card (cancelled meanwhile, and patient at phone size) is covered by J16b's new test, which passed this run, and by test report 1 (T-04, T-05). It wasn't repeated by hand.

### Test results (25 Oct)
- `make lint`: clean.
- `make test`:
  - Go: all ok;
  - Vitest: 244 passed (web, 23 files) and 161 passed (clinical, 19 files);
  - CDK: 124 passed.
- `make test-integration`: all packages ok.
- `make e2e` (against the conductor's `make dev`): **72 passed, 5 skipped** (the Cognito journeys run on QA only), in 5 min 14 s.

### Hand checks for Rahul
- **On AWS QA (later):** API Gateway's real 5xx, when a Lambda times out or throws, shows the plain network message.

### Commits
1. `S30: the plain network message for any 5xx without an API error`:
   - `apps/web/src/lib/api.ts`, `api.test.ts`;
   - `e2e/journeys/test_j5_poa_summary.py`;
   - `apps/web/AGENTS.md`.
2. `S30: the cleared card hides only the network message`:
   - `prefill.ts`, `prefill.test.ts`, `PatientHQ.tsx`;
   - `e2e/journeys/test_j16b_prefill_cross_trust.py`.
3. `S30: J8 reads only its own episode's texts`:
   - `e2e/journeys/test_j8_reminders.py`;
   - `e2e/README.md`.
4. `docs: S30 verified`:
   - this folder;
   - `docs/sessions/README.md`, `docs/pending.md`, `docs/decisions.md`.
