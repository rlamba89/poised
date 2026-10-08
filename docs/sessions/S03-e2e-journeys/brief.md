# S03: End-to-end journeys J1–J5

**Plan:** F1 ([f1-test-foundation.md](../../plans/f1-test-foundation.md)), Steps 3 and 4. This is part 3 of 3; F1 is done after this session.
**Depends on:** S02 verified.

**Ask Rahul first:** nothing. Check Python 3.12+ is installed.

## Build

- **`e2e/`,** with Playwright + pytest, exactly as F1 Step 3 describes:
  - pinned requirements, `conftest.py`, `journeys/` and a README;
  - data set up through the API, with unique names;
  - role and label selectors.
- **Journeys J1–J5**, from F1's table. J3's patient runs on an iPhone 13.
- **`make e2e-setup` and `make e2e`.** Traces are kept on failure.
- **Clean up:**
  - delete `spikes/ui-workflow.ts`;
  - add a Tests section to the root `README.md`;
  - add "automated: J<n>" notes in `docs/manual-test-plan.md`.

## Not in this session

- new journeys for features that don't exist yet;
- app changes, except `data-testid` attributes where an element has no accessible name. List those in `e2e/README.md`.

## Done when

- `make e2e` passes headless 3 runs in a row.
- Breaking the patient submit button's label makes J3 fail, with a readable trace.

## Manual test focus

1. **Follow `e2e/README.md` literally,** as a newcomer would: setup, a headless run, then a `--headed` run, watching it. Report any unclear or wrong step.
2. **Do J1–J5 by hand once in the browser,** and report anything the automation does differently from a real user.
