# S46 build notes: observations

[Brief](brief.md) · [Answers](answers.md) · [Test plan](test-plan.md). Plan C7, part 1 of 4. Built 29 Oct 2026.

## Answers (Rahul, 29 Oct)

1. **Ranges: platform defaults each trust can change** (not the recommended platform-only list). Built as one trust-level setting, `observations.ranges`; some rules stay fixed in Go whatever a trust sets.
2. **Correct and Entered in error, each with a reason**, as S44b's ASA grades.
3. **One stored unit per kind; each country's display units from Go's country config** (`GLUCOSE_UNIT`, `TEMPERATURE_UNIT`).
4. The C7 split (S46–S49) is agreed.

## What was built

### Go

- **`internal/observation` (new):** the kinds (OBS-01: height, weight, blood pressure (systolic and diastolic), heart rate, SpO₂, temperature, respiratory rate, blood glucose, peak flow, and ECG and urinalysis as notes), each measure's stored unit and decimal places, the **platform's default ranges** (made up until a clinician checks them), `Effective` (the trust's own ranges over the platform's), `CheckRanges` (the rules no trust can change) and `Check` (a reading's hard limits). `DisplayUnitOK` says which display units a country may name: only the stored ones for now.
- **Migration 00037 `observations`:** one row per reading, as the plan says, plus `note`, `entered_at`, `corrects` with its `reason`, and the entered-in-error fields. Triggers refuse any edit but one supersede or one entered-in-error mark, and any delete but the episode's own (as S44b's grades). A reading is corrected at most once (unique index on `corrects`).
- **Routes** (`internal/httpapi/observations.go`), each `audited` (`observation.list|add|correct|error`, with labels on the Audit page):
  - `GET …/episodes/{eid}/observations`: the readings, oldest first, plus the kinds, measures and ranges that apply;
  - `POST …/observations`: add one (400 with the reason outside the hard limits);
  - `POST …/observations/{obsid}/correct`: a new reading of the same kind supersedes it (422 without a reason; 409 if it no longer counts);
  - `POST …/observations/{obsid}/entered-in-error` (422 without a reason; 409 if it no longer counts).
- **Settings:** `observations.ranges`, a trust-only setting of a new kind, `obsRanges` (group "Observations"), whose default is the platform's table; `settings.Validate` checks the shape (four numbers per measure, no unknown fields) and then `observation.CheckRanges`. The definition carries the measures for the editor.
- **Config (A-20):** `GLUCOSE_UNIT` and `TEMPERATURE_UNIT` are country values: required with `CONFIG_STRICT`, UK defaults locally (mmol/L, °C), and only those accepted until conversions are built. `GET /api/config` sends them as `observationUnits`.

### TypeScript (for Rahul to review)

- **`packages/clinical/src/observations.ts`:** the `Reading` type (what the GET sends); `counting` (leaves out corrected and entered-in-error readings); `latest` (the newest that counts, by when it was measured); **`hospitalBmi`** (from the latest height and weight that count, with when); `outsideUsual` (the measures outside their usual range, and none when any value is outside its hard limits, so the server's refusal shows instead of a warning). `bmiOf` is the one BMI formula: the HQ's `bmi()` and `reportedBmi` now call it too, so all three round the same way. In Go terms these are small pure functions with table tests.
- **`apps/web/src/features/episodes/observations.ts`:** the panel's words and request bodies (`valueText` "128/82 mmHg", `historyLine`, `errorLine`, `warningText`, `readingRequest`, `correctionRequest`, and the date-time box's conversion to and from the browser's time zone). Unit-tested.
- **`ObservationsPanel.tsx`:** the episode page's card, below Tasks: the hospital BMI, **Add a reading** (the kind, then a box per measure with its unit; Measured at, now by default; a yellow "Save it anyway?" box outside the usual range), and the history newest first, with **Correct** and **Entered in error** inline forms on each reading that still counts. `useObservations` loads it; a 403 (observations off) hides the card. The POA Summary uses the same hook for its **Hospital BMI** line, under Patient reported BMI.
- **Settings page:** an editor for the `obsRanges` kind (a row per measure: Usual from, Usual to, Refused below, Refused above). An unchanged table doesn't save again: `sameRanges` in `features/settings/obsRanges.ts` compares both sides as Save would send them (*corrected 31 Oct, verification 2:* it first compared them as text, which never matched, as the database returns the keys in its own order).
- `lib/config.ts`: `observationUnit(measure)`.

### Infra

`infra/countries.py`: `glucose_unit` and `temperature_unit` (uk-qa: mmol/L, °C), checked against the same lists; `api_stack.py` passes them to the Lambdas. **QA needs these before its next deploy**, or the Go functions refuse to start (strict). The pipeline deploys them with the push.

### E2E

**J12** (`e2e/journeys/test_j12_observations.py`). It found two bugs in my first version, both fixed test-first:
- a value outside the hard limits got the "Save anyway?" warning before the server's refusal;
- picking the same kind again deselected it (Mantine's default), so a second blood pressure couldn't be added the obvious way.

## Decided by the builder

1. **Readings don't bump the episode's revision or write General notes lines.** They have their own history (who, when, why). Each reading is saved on its own, so recording one never makes another tab's episode edit stale. The routes still take the episode's lock and refuse a cancelled or archived episode (409).
2. **Observations off at the hospital: every route is a 403** ("Observations aren't turned on at this hospital."), and the card and the POA Summary line are hidden.
3. **A correction keeps the kind and the original's time** unless a time is sent (the panel doesn't send one: a correction is of the same measurement).
4. **The diastolic must be strictly lower than the systolic.**
5. **A time up to 5 minutes ahead of the server's clock is accepted** (a browser's clock); later is "in the future".
6. **Decimal places:** height, weight, temperature and glucose take 1; the rest whole numbers. More is refused, for readings and for a trust's limits.
7. **Values are stored as `double precision`**, shown with the measure's decimal places; each row keeps its unit.
8. **The editor saves the measures that differ from the defaults** (*changed 30 Oct, verification 1:* it saved the whole table at first). The API accepts some measures only; the rest keep the platform's. Fixed rules: no limit below 0; usual range inside the hard limits, lowest first; SpO₂ at most 100.
9. **The ranges setting lives in the trust's Settings page** (whole-trust admins); hospitals show it as "Set for the whole trust".
10. **The jobs Lambda gets the units like the others** (strict config needs them everywhere).

## Differences from the brief

- Brief question 1 was answered (b), so the ranges are a trust setting rather than only a Go file; the Go file still holds the platform's defaults.
- The brief said a missing reason is 422; a malformed reading is 400, as the brief says. A broken trust range is 400, like any setting.

## Test results (29 Oct)

- `make lint`: passed.
- `make test`: Go all packages ok (new: `internal/observation`, config's units, settings' `observations.ranges`); Vitest: web 318 (18 new), `apps/evaluate` 25, `packages/clinical` 482 (10 new); CDK 132 (2 new).
- `make test-integration`: all packages ok. (The first run's only failure, `TestConfig`, needed the new units in its strict environment: fixed.)
- New integration tests: `TestAddObservation`, `TestListObservations`, `TestCorrectObservation`, `TestMarkObservationInError` (the five cases each, plus hard limits, the trust's own limits, a cancelled or archived episode, observations off, 409s), `TestObservationsAudited`, `TestObservationsNeverEdited` (the triggers), `TestObservationRangesSetting`.
- `make e2e` against `make dev`: **102 passed, 5 skipped** (the QA-only Cognito journeys) in 9 min 37 s, J12 included.

## Test report 1 (29 Oct): 14 passed, 3 failed

Each failure was reproduced, a test written that failed because of it, then fixed:

- **F-1 (Major): ECG and urinalysis couldn't be saved.** Go sent `"measures": null` for the note kinds (a nil slice), and the panel's warning check called `.some` on it. Fixed in Go (an empty list; `TestKindsJSONHasNoNullMeasures`), and the web app no longer reads a note kind's measures (`warningText`, unit test with `null`). J12 now saves an ECG note in the browser.
- **F-2 (Minor): a diastolic above the systolic got "Save it anyway?" first.** The form now refuses it at once with Go's words ("The diastolic must be lower than the systolic."), for a new reading and a correction (unit tests; J12 checks no warning appears).
- **F-3 (Minor): a trust range of "-1" was saved as 1.** Mantine's NumberInput dropped the minus as it was typed. The ranges editor's boxes are now plain text boxes that keep what's typed; Save sends plain numbers as numbers and anything else as typed, so Go refuses it with its own message (`rangesForSave`, unit tests; a J12 test types -1 as Ada and sees "Heart rate: a limit can't be below 0.").

From "Found while exploring", fixed because they were small and in this session's code:
- **A stale tab:** a 409 on Correct or Entered in error now reloads the list, so the reading shows as it is (corrected, no buttons) under the message (J12's second test).
- **"Measured at" lagged the save:** while the box isn't changed, the time sent is the time of Save (`measuredAt`, unit test); once changed, it's what was typed.
- **"1e2" was read as 100:** the boxes take only plain decimals (and a comma), so "1e2" and "0x10" say "enter a number" (unit test).

After the fixes: `make lint` passed; `make test`: Go ok, Vitest web 325, `apps/evaluate` 25, `packages/clinical` 482, CDK 132; `make test-integration` ok; `make e2e` **104 passed, 5 skipped** (J12 now 3 tests).

Not changed (added to pending.md): the print layout wrapping the Hospital BMI line; an earlier-timed reading sorting below the newer ones with no other sign it was saved; the Tasks card's Complete button cut off at 390 px (S21's); two "Opened an episode's observations" audit rows per page load in dev (React's double effects; check on QA).

## Test report 2 (29 Oct): 13 passed, 2 failed

F-1 to F-3 confirmed fixed. Both failures were one new bug:

- **F-4 (Minor): "Say why…" stayed after a correction or mark was saved.** Since test report 1 a reading's message shows outside its closed form (for a stale tab's 409), but a save that worked closed the form without clearing it, so the old refusal stayed under the reading, one per reading. Back had the same gap. Both now clear it (`close()` in `ReadingEntry`); only a 409's message stays. J12 failed on it first: it now checks no message is left after a correction, after an entered in error that was first refused, and after Back.

From "Found while exploring", fixed because they were small:
- **The ranges' "give the four limits as numbers" didn't name the row:** Go now says "Weight: give all four limits as numbers." for a missing, empty, text or `null` limit (`unnumbered` in `internal/settings`, unit tests). It also found that `null` was read as 0 before (then refused with a confusing message); it's refused as not a number now.
- **"Measured at" showed the time the form opened:** the box now says "The time you save it, unless you change it." until it's changed (J12).

After the fixes: `make lint` passed; `make test`: Go ok, Vitest web 325, `apps/evaluate` 25, `packages/clinical` 482, CDK 132; `make test-integration` ok; `make e2e` **104 passed, 5 skipped**.

Not changed (cosmetic, added to pending.md): the respiratory rate label wraps at desktop width; the kind dropdown can't be typed into. Still open from report 1: the print wrap, the Tasks card at 390 px. The `Listed episodes … REFUSED (500)` at 17:53 was during an earlier e2e run, before the retest; not from S46's routes (noted in pending.md for whoever looks at the list next).

## Test report 3 (30 Oct): 13 passed, 0 failed

F-4 and the two small changes after it confirmed. Not re-run: T-08, T-09 and T-11, which passed before and weren't touched by the fixes; T-01 only in part (all 11 kinds passed in report 2). No code changed after this report. The long unbroken note it found is in pending.md, with the phone-size reading text.

**Hand check for Rahul:** T-12 and the patient's view at 390×844 in real Chrome (the tester's window stops at 500 px; headless Chromium at that size passed).

## Verification 1 (30 Oct): Needs fixes

Rahul's answers: F-2, leave it (a correction marked entered in error doesn't bring the old value back; the form says so). Fix the long note now. Do all three of the verifier's non-blocking items now, not later.

Each was seen to fail first, except where noted:

- **F-1 (Major): a correction got no soft warning.** The Correct form now warns like Add a reading, with Save anyway and Change it (`ReadingEntry`; J12 corrects a weight to 380).
- **F-2: the form says what happens.** The Entered in error form on a correction reads "The value it corrected won't count again. To put a value back, use Correct." No table change (J12). This differs from the ASA grades on purpose: decisions.md, 30 Oct.
- **F-3: simultaneous corrections answered 500.** `correctObservation` reads the reading again under the lock before it inserts (`TestCorrectObservationAtTheSameMoment`: 5 rounds of 6 at once gave 500s before; now one 201, the rest 409).
- **F-4: each POA tab switch reloaded the readings and wrote audit rows.** `useObservations` moved up to `PoaSummary`, which stays mounted (J12 counts the requests across two switches).
- **F-5: times compared as text.** `byTime` in `@poised/clinical`, used by `latest`, `hospitalBmi` and `newestFirst` (unit tests with mixed offsets).
- **The long unbroken note (test report 3)** now wraps: `overflow-wrap: anywhere` on the reading's text (J12's fourth test).
- **Measured at has an earlier limit:** before the patient's date of birth is a 400, "The time it was measured can't be before the patient's date of birth." (`observation.Check` takes `born`; unit and integration tests). A date of birth is a date, so the limit is 14 hours before its midnight in UTC, the earliest that day starts in any zone.
- **The ranges editor sends only the measures that differ from Preopsis's defaults** (`rangesForSave(v, defaults)`, unit tests). All the same sends `{}`, which Go accepts. A trust that saved all ten before sends the shorter value on its next save.
- **Tidying:** removed `setKind(kind)` (it did nothing) and `gone` (the same as `!counts`).

J12's F-1 and F-2 steps went in with the fixes, not run first. The two that could have passed without proving anything were checked by putting each bug back briefly: the long-note check and the tab-switch check both failed, then the fixes were restored.

After the fixes: `make lint` passed; `make test`: Go ok, Vitest web 328, `apps/evaluate` 25, `packages/clinical` 485, CDK 132; `make test-integration` ok; `make e2e` **105 passed, 5 skipped** (J12 now 4 tests).

## Test report 4 (31 Oct): 23 passed, 0 failed

All 18 cases and 5 regressions, including T-13 to T-18 and simultaneous corrections against the running app. No code changed after this report. Its three small points are in pending.md: a pasted note cut at 2,000 characters with no message, a reading allowed on the day before the date of birth (by design), and Save reading moving when the hint goes.

**Hand check for Rahul:** T-12 and the patient's view at 390×844 in real Chrome (the tester's window stops at 500 px; headless Chromium at that size passed).

## Verification 2 (31 Oct): Needs fixes

All three seen to fail first, and all are web code:

- **F-1 (Major): a stale Settings page undid another admin's range.** The stored ranges come back in the database's key order and the editor's in ours, so they compared unequal as text and the ranges went in every Save. `sameRanges` compares them as Save would send them. So Save with no edits says "Nothing has changed.", and a page left open no longer sends the ranges when it changes something else (unit tests in database order; J12 saves unchanged). A page that really edits the ranges is still last save wins, like every other setting.
- **F-2 (Minor): a five-digit year made Save do nothing, with no message.** `readingRequest` now answers "When was it measured? Enter the date and time." for a time that isn't one. The two date-box helpers moved from `features/qa/jobsTime.ts` to `lib/localTime.ts`, so there's one copy, and it's the one that checks (unit tests; Playwright can't type that year into the box, so it's T-20 for the tester).
- **F-3 (Minor): a failed load hid the Hospital BMI.** `PoaSummary` keeps the hook's error, and `hospitalBmiRow` says "Couldn't be loaded", so it can't read as observations being off. Off, or still loading, has no row (unit tests; a J12 test forces a 500).

The verifier's non-blocking items are in pending.md.

After the fixes: `make lint` passed; `make test`: Go ok, Vitest web 338, `apps/evaluate` 25, `packages/clinical` 485, CDK 132; `make test-integration` ok; `make e2e` **107 passed, 5 skipped** (J12 now 6 tests).

## Test report 5 (31 Oct): 16 passed, 0 failed, one Major from exploring

Every planned case passed, but the tester found a page left open still undid a change to another row of the ranges table. Rahul chose to refuse the second save, and asked for all the open smaller items too. Each test was seen to fail first.

- **Two admins editing the ranges:** a save that changes them sends what the page loaded (`loaded`), and `staleRanges` compares it as values, under the existing lock: 409 if it's gone. Applies to going back to the default too. Other settings on the page still save (`TestObservationRangesLeftOpen`; J12 with two pages).
- **The five unexplained 500s:** not S46's. A browser leaving a page mid-request cancels it, the query fails, and the row said Refused (500). Confirmed by dropping 40 local requests: 40 new rows (they stay in the local log, which is append-only). `auditStatus` records 499 for a 5xx when the caller left, shown as **Cancelled** (`outcome` in `features/audit/filters.ts`). A timeout stays a 500. `TestAuditCallerLeft` holds a table lock so the request is really mid-way when it's dropped.
- **The silent 2,000-character note:** no `maxLength` on the box; `valuesOf` refuses it by characters, as Go counts, with what's pasted kept. Go sends `maxNote`, and the body limit for readings is 16 KB, enough for the longest note in any script.
- **Refused rows with observations off:** the episode's GET sends `observationsOn`; `useObservations(path, on)` doesn't call the route when it's false, or before the episode has loaded. J12's first version of this test proved nothing (its route caught the page's own URL), so after fixing it the bug was put back briefly: it failed, and the fix was restored.
- **Tidying (no behaviour change):** `lockedObservations` uses `s.inTx`; the `n == 0` branch in `correctObservation` is gone (the read under the lock covers it); `SaveAnyway` replaces the block written twice; `parseRanges` parses the JSON once. A range row with an extra field now names its measurement.

**How the journeys ran this round:** Rahul's `make dev` predates the Go changes, so a separate copy ran on :8180 and :3100 with the same database (`E2E_BASE_URL=http://localhost:3100`). That copy was stopped afterwards.

After the fixes: `make lint` passed; `make test`: Go ok, Vitest web 340, `apps/evaluate` 25, `packages/clinical` 485, CDK 132; `make test-integration` ok; `make e2e` **109 passed, 5 skipped** (J12 now 8 tests).

## Test report 6 (31 Oct): 18 passed, 0 failed

The new cases T-22 to T-25, the regressions and four happy paths. Report 5's Major held up in five more variants and five saves at the same moment (one 200, four 409). No code changed after this report. Its two cosmetic points are in pending.md: a corrected long note shown twice, and the note box growing with a long paste.

## Suggested commits

1. `S46: observations in Go: the kinds, ranges and limits, the table and its routes` — `apps/api/internal/observation/*`, `apps/api/db/migrations/00037_observations.sql`, `apps/api/db/queries/observations.sql`, `apps/api/internal/db/*` (generated), `apps/api/internal/httpapi/{observations,observations_integration_test,audit,router}.go`.
2. `S46: a trust's own observation ranges, and the countries' display units` — `apps/api/internal/settings/*`, `apps/api/internal/config/*`, `apps/api/internal/httpapi/{settings_integration_test,routers_integration_test}.go`, `infra/{countries,api_stack}.py`, `infra/tests/*`, `.env.example`.
3. `S46: the Observations card, the hospital BMI and the ranges editor; J12` — `packages/clinical/src/{observations,observations.test,questionTypes,poa,index}.ts`, `apps/web/src/features/episodes/{ObservationsPanel.tsx,observations.ts,observations.test.ts,EpisodePage.tsx,PoaSummary.tsx}`, `apps/web/src/features/settings/*`, `apps/web/src/features/admin/types.ts`, `apps/web/src/lib/config.ts`, `e2e/journeys/test_j12_observations.py`, `e2e/README.md`.
4. `S46: docs` — the C7 split (`docs/sessions/S46–S49`, `docs/sessions/README.md`, `docs/plans/*`), `docs/decisions.md`, `CLAUDE.md`, `apps/web/AGENTS.md`, this file and the test plan.
