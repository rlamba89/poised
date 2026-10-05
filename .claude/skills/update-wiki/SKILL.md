---
name: update-wiki
description: Record what this session decided or learned into the repo's shared docs (docs/decisions.md, the docs/plans status, CLAUDE.md), so other developers and future Claude sessions on any machine have it. Use it when the user says "update the wiki", "record this decision", "log this" or "hand over", when a decision is agreed or reversed, when a plan step is finished, when a non-obvious gotcha is found, and before the session ends if any of these happened.
---

# Update the wiki

The project's memory is **in the repo**, not in Claude's memory or chat history, because those stay on one machine. These files are the "wiki":

| File | What goes there |
| --- | --- |
| `docs/decisions.md` | §1 the product owner's working preferences; §2 the decision log (newest first); §3 SurveyJS findings; §4 environment gotchas; §5 open items |
| `docs/plans/README.md` | The status table of plans F1–F4 and C1–C10 |
| `docs/plans/<plan>.md` | The plan's own **Status** section, with the differences from the plan |
| `docs/saas-requirements.md` | §0 decision table (A-n agreed, D-n proposed). Update it when a SaaS decision changes. |
| `CLAUDE.md` | Only if commands, architecture or working rules changed. Keep it short and point to the docs above. |

## Steps

1. **Collect** from this session (the conversation, `git log`, `git diff`) only what a newcomer would need:
   - decisions agreed or reversed, each with its **reason** and the date;
   - new preferences or corrections from the product owner;
   - gotchas that cost time, with the exact command or fix;
   - plan steps finished, and any differences from the plan;
   - items left open.

   Skip anything already recorded, routine steps, and what the code or git history shows plainly.
2. **Read the target sections first**, then edit in place:
   - **decisions.md §2:** add today's entries under a heading `### <D Mon YYYY>: <topic>`, at the **top** of §2 (or extend today's heading if it exists). One bullet per decision: **the decision in bold**, then the reason, then where it's detailed.
   - **Never delete a superseded decision.** Mark it *superseded <date> by …*, and add the new one.
   - **Status:** if only *proposed*, say so. Don't record a proposal as agreed.
   - **§5 open items:** remove items that are now done, and add new ones.
   - **Plans:** update the status table row (e.g. `Done 2026-10-07`) and the plan's Status section.
   - Use plain English, short bullets, and real paths and commands. Use the same terms as the existing docs.
3. **Safety checks before saving:**
   - Never write the Lifebox company's GitHub organisation name. Describe it, don't spell it out. (Run the grep in step 4.)
   - No secrets, tokens, connection strings, or real patient data.
   - Nothing that's only true on one person's machine unless it's labelled as such (e.g. "on Rahul's Mac").
4. **Check:**
   - `git diff -- docs CLAUDE.md` reads well;
   - `grep -rIlE "lifebox-health(c)are" . --exclude-dir=node_modules --exclude-dir=.git` prints nothing. (The pattern finds the name without containing it.)
5. **Show the user** a short summary of what was recorded and where. **Commit only if they agree**, as `docs: record <topic>`, with the docs alone (never mixed with code changes). Push only if asked.
