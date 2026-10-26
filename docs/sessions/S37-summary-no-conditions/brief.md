# S37: No conditions on summary boxes

**Plan:** none of its own. Rahul's answer of 26 Oct to S32's open question ([decisions.md](../../decisions.md) §2, "questions from the C5 run answered"; [pending.md](../../pending.md), "From S32's verification 1").
*Local phase: runs on Rahul's Mac only. No AWS, and no real email or SMS.*

**Depends on:** S36 verified (plan C5 done).

**Asked:** nothing. Rahul, 26 Oct: a Clinical Summary box can't have a condition.

## Build

- **The editor offers no Logic on a Clinical Summary** (`apps/web/src/features/editor`): no "Displays when…" condition and no Logic tab or button for a summary element (`type: "clinicalsummary"`), the same way other elements that can't take one are handled.
- **Publish refuses a summary with a condition** (`packages/clinical/src/publish.ts`, and Go's publish check if summaries are checked there): a `visibleIf` (or `enableIf`/`requiredIf`) on a summary is a publish problem naming the Question Set and the summary's title, linking to the item, as other publish problems do.
- **A draft that already has one** (made by hand or by the API) shows the problem in the Publish window, and the author can remove the condition from the Code tab or by deleting the summary.
- **Published versions are never rewritten.** A published version that already has a summary with a condition keeps working as today.
- **Unit tests** (`packages/clinical`, test-first): the publish problem; a summary without a condition passes.
- **Integration test:** Go's publish refuses a summary with a condition (only if Go checks summaries at publish; otherwise say so in the build notes).
- **End-to-end:** J10 or the summary journey checks that a summary card offers no Logic.

## Not in this session

- The other notes in pending.md about summaries (indirect repeating groups, two summaries from the importer, capitalisation).

## Manual test focus

1. Add a Clinical Summary: its settings and card offer no Logic or "Displays when…".
2. Give a summary a condition in the Code tab (or via the API) and press Publish: refused, naming the summary.
3. Remove the condition: Publish works.
4. Other elements still offer Logic as before.
