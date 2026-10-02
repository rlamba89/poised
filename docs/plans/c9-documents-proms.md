# C9: Documents and PROMs

**Goal:** custom POA document layouts that can be downloaded as PDF, and PROMs (baseline and follow-ups) with licensed instruments as opt-ins.

**Requirements:** DOC-01…03, D-3, PRM-01…08, A-18, ORG-05. **Depends on:** C3 (episode_forms, procedures) and C2 (jobs, messaging).

## Documents

- `document_templates`: org_id NULL (platform), hospital_id NULL, name, `blocks jsonb`. Blocks come from a fixed list, each with its own options (§8).
- **Editor:** a simple list (add, remove, reorder, options per block), **not** a free-form designer.
- **Renderer:** reuses the POA Summary components.
- **PDF:** the browser prints to PDF (D-3), and the page shows the template name, the date and who generated it. A server-side PDF is later.

## PROMs

- **Instruments:** platform library questionnaires of kind `proms`, with locked wording and scoring:
  - **EQ-5D-5L** (5 items + the visual analogue scale, scored with the **UK value set**);
  - **Oxford Hip Score** and **Oxford Knee Score** (12 items, scored 0–48);
  - **recovery check-ins**, which are free and authored by the trust.
- **Opt-ins (PRM-08):**
  - Each licensed instrument is opt-in **per trust, with a per-hospital override** (ORG-05), set when the trust or hospital is added, or later.
  - A hospital that hasn't opted in can't add or send it.
  - The licence holder and expiry can be noted. Licensors aren't contacted for now.
- **Plans:** on a procedure: a list of (instrument, **baseline** | discharge + N days, window).
  - **Baseline** forms join the patient's to-do list at invite, pre-filled by key where possible.
  - **Follow-ups** are jobs at discharge + N days: a message, one reminder after 7 days, and **Missed** when the window closes.
  - The patient can opt out.
- **Results:** a score per time point and the change from baseline. A drop beyond a threshold creates a task. There's also a per-procedure report with CSV export (Should).
- The episode stays in `recovery` while any form is open.

## Tests

- **Unit (written TDD):**
  - EQ-5D-5L scoring against the published UK value-set examples;
  - OHS and OKS scoring;
  - the schedule decision function (given a discharge date and a plan, which forms are due and when);
  - the change-from-baseline task rule;
  - the opt-in check.
- **Integration:**
  - a hospital that hasn't opted in can't add or send a licensed instrument (403);
  - a hospital override wins over the trust default;
  - discharge creates the scheduled `episode_forms`;
  - opt-out stops the sends;
  - document templates: platform default vs trust template; tenancy.
- **End-to-end:**
  - **J14:** an admin builds a template and downloads a POA document; it contains the chosen blocks.
  - **J15:** for an opted-in hospital, a hip-replacement episode shows the EQ-5D-5L and Oxford Hip baseline in the patient's to-do list; the patient completes them; the scores appear on the episode.
  - Follow-up timing is covered by the unit and integration tests, because end-to-end tests don't time-travel.

## Status

Not started.
